package mutex

import (
	"os"
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

func TestSimpleMutex(t *testing.T) {
	const mutexId = "simple-test-mutex"
	mutexRoot := temporaryCatalog(t)
	mx, err := NewMutex(mutexRoot, mutexId)
	if err != nil {
		t.Fatalf("cannot create the mutex: %v", err)
	}
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
