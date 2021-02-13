package mutex

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func temporaryCatalog(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "temp-*.dir")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("error removing temporary directory: %v", err)
		}
	})
	return tempDir
}

func newTestMutex(root string, id string) *Mutex {
	result, err := NewMutex(root, id)
	if err != nil {
		log.Fatalf("Cannot create mutex \"%s\": %v", id, err)
	}
	return result
}

func TestSimpleMutex(t *testing.T) {
	const mutexId = "simple-test-mutex"
	mutexRoot := temporaryCatalog(t)
	mx := newTestMutex(mutexRoot, mutexId)
	value := 0
	mx.Lock()
	go func(v *int) {
		mx.Lock()
		defer mx.Unlock()
		want := 33
		if *v != want {
			t.Fatalf("wrong value %d instead of %d", *v, want)
		}
	}(&value)
	value = 33
	mx.Unlock()
}

func TestSimpleMutexN(t *testing.T) {
	const mutexId = "simple-test-mutex"
	var wg sync.WaitGroup

	mutexRoot := temporaryCatalog(t)
	value := 100

	mx, err := NewMutex(mutexRoot, mutexId)
	if err != nil {
		t.Fatalf("cannot create the mutex: %v", err)
	}
	mx.Lock()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, v *int) {
			defer wg.Done()
			lmx, err := NewMutex(mutexRoot, mutexId)
			if err != nil {
				t.Fatalf("cannot create the mutex: %v", err)
			}
			lmx.Lock()
			defer lmx.Unlock()
			*v += 1
		}(&wg, &value)
	}
	time.Sleep(10 * time.Millisecond)
	want := 100
	if value != want {
		t.Fatalf("wrong value %d instead of %d", value, want)
	}
	mx.Unlock()
	wg.Wait()
	want = 200
	if value != want {
		t.Fatalf("wrong value %d instead of %d", value, want)
	}
}

func TestLockPath(t *testing.T) {
	const mutexId = "simple-test-mutex"
	mutexRoot := temporaryCatalog(t)
	mx := newTestMutex(mutexRoot, mutexId)
	want := filepath.Join(mutexRoot, mutexId, fmt.Sprintf("%s-mutex.lck", mutexId))
	got := mx.LockPath()

	if want != got {
		t.Fatalf("wrong value \"%s\" instead of \"%s\"", got, want)
	}
}

func TestId(t *testing.T) {
	const mutexId = "simple-test-mutex"
	mutexRoot := temporaryCatalog(t)
	mx := newTestMutex(mutexRoot, mutexId)
	want := mutexId
	got := mx.Id()

	if want != got {
		t.Fatalf("wrong value \"%s\" instead of \"%s\"", got, want)
	}
}

func TestWhen(t *testing.T) {
	const mutexId = "simple-test-mutex"
	mutexRoot := temporaryCatalog(t)
	mx := newTestMutex(mutexRoot, mutexId)
	mx.Lock()
	defer mx.Unlock()
	if file, err := os.Create(mx.LockPath()); err != nil {
		t.Fatalf("cannot create the mutex file: %v", err)
	} else {
		if want, err := writeCurrentTimestamp(file); err != nil {
			t.Fatalf("cannot write the timestamp: %v", err)
		} else {
			got := mx.When().UnixNano() / int64(time.Millisecond)
			if want != got {
				t.Fatalf("wrong value %d instead of %d", got, want)
			}
		}
	}
}

func TestTry1(t *testing.T) {
	const mutexId = "try1-test-mutex"
	mutexRoot := temporaryCatalog(t)
	mx1 := newTestMutex(mutexRoot, mutexId)
	mx1.Lock()
	go func() {
		defer mx1.Unlock()
		time.Sleep(3 * time.Second)
	}()
	mx2 := newTestMutex(mutexRoot, mutexId)
	defer mx2.Unlock()
	if err := mx2.TryLock(5 * time.Second); err != nil {
		t.Fatalf("TryLock failed (%v), but should succeed.", err)
	}
}

func TestTry2(t *testing.T) {
	const mutexId = "try2-test-mutex"
	mutexRoot := temporaryCatalog(t)
	mx1 := newTestMutex(mutexRoot, mutexId)
	mx1.Lock()
	go func() {
		defer mx1.Unlock()
		time.Sleep(3 * time.Second)
	}()
	mx2 := newTestMutex(mutexRoot, mutexId)
	defer mx2.Unlock()
	if err := mx2.TryLock(1 * time.Second); err == nil {
		t.Fatal("TryLock succeed but should failed.")
	}
}

func TestMutexDefaults(t *testing.T) {
	const mutexId = "mutex-defaults"
	mutexRoot := temporaryCatalog(t)
	zero := time.Duration(0)
	if mx, err := NewMutexExt(mutexRoot, mutexId, zero, zero, zero); err != nil {
		t.Fatal(err)
	} else {
		expected := DefaultPulse
		if got := mx.pulse; got != expected {
			t.Fatalf("Wrong pulse default: got: %v, expected: %v", got, expected)
		}
		expected = DefaultRefresh
		if got := mx.refresh; got != expected {
			t.Fatalf("Wrong pulse refresh: got: %v, expected: %v", got, expected)
		}
	}
}

func TestMutexRoot(t *testing.T) {
	const mutexId = "mutex-root"
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	tmpdir := temporaryCatalog(t)
	if err := os.Chdir(tmpdir); err != nil {
		t.Fatal(err)
	}
	mutexRoot := "./here"
	if mx, err := NewMutex(mutexRoot, mutexId); err != nil {
		t.Fatal(err)
	} else {
		if !path.IsAbs(mx.directory) {
			t.Fatalf("Wrong lock directory - should be absolute (%s)", mx.directory)
		}
	}
}
