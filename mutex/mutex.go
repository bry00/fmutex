package mutex

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// A Mutex is a mutual exclusion lock based on filesystem primitives.
type Mutex struct {
	id              string
	directory       string
	deadAgeRecovery time.Duration
	pulse           time.Duration
	refresh         time.Duration
}

// DefaultPulse determines default frequency of locking attempts, i.e. defines delay between subsequent locking attempts.
const DefaultPulse = 500 * time.Millisecond

// DefaultRefresh determines default frequency of saving current timestamp in a locking file.
const DefaultRefresh = 10 * time.Second

// DefaultDeadTimeout determines how long takes to consider given mutex as "dead".
// "Dead" mutexes are removed during locking attempts.
const DefaultDeadTimeout = 60 * time.Minute

// A lockCandidateTemplate defines locking candidate file name template.
const lockCandidateTemplate = "%s-candidate-*.tmp"

// A lockTemplate defines locking file name template.
const lockTemplate = "%s-mutex.lck"

// Id return given Mutex id.
func (m *Mutex) Id() string {
	return m.id
}

// Lock locks given Mutex. Panics in case of any error. Conforms to the sync.Locker interface.
func (m *Mutex) Lock() {
	if err := m.TryLock(0); err != nil {
		panic(err)
	}
}

// Unlock unlocks given Mutex. Panics in case of any error. Conforms to the sync.Locker interface.
func (m *Mutex) Unlock() {
	if err := m.TryUnlock(); err != nil {
		panic(err)
	}
}

// TryLock tries to lock given Mutex and returns error in case of failure.
// If timeout is greater than 0, the unsuccessful lock attempt is failed after timeout.
func (m *Mutex) TryLock(timeout time.Duration) error {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return m.LockWithContext(ctx)
}

// TryUnlock unlocks given Mutex or returns error in case of failure.
func (m *Mutex) TryUnlock() error {
	return os.Remove(m.LockPath())
}

// LockWithContext waits indefinitely to acquire given Mutex with timeout governed by passed context
// or returns error in case of failure.
func (m *Mutex) LockWithContext(ctx context.Context) error {
	candidateLock, err := ioutil.TempFile(m.directory, fmt.Sprintf(lockCandidateTemplate, m.id))
	if err != nil {
		return fmt.Errorf("cannot create candidate lock %s: %w", m.id, err)
	}
	candidateLock.Close()
	candidate := candidateLock.Name()
	defer os.Remove(candidate) // clean up

	target := m.LockPath()

	var lastTimestamp int64 = 0
	for {
		if lastTimestamp == 0 || now()-lastTimestamp > millis(m.refresh) {
			if f, err := os.Create(candidateLock.Name()); err == nil {
				if lastTimestamp, err = writeCurrentTimestamp(f); err != nil {
					return fmt.Errorf("cannot write current timestamp for candidate lock %s: %w", m.id, err)
				}
			}
			if m.deadAgeRecovery >= 0 {
				if otherTimestamp := readTimestamp(target); otherTimestamp > 0 {
					if now()-otherTimestamp > millis(m.deadAgeRecovery) {
						os.Remove(target)
						time.Sleep(m.pulse * 2)
					}
				}
			}
		}
		if err := os.Link(candidate, target); err == nil {
			if now()-lastTimestamp > millis(m.refresh) {
				if f, err := os.Create(target); err == nil {
					_, err = writeCurrentTimestamp(f)
				}
				if err != nil {
					return fmt.Errorf("cannot write current timestamp for target lock %s: %w", m.id, err)
				}
			}
			return nil
		}
		if sleepOrDone(ctx, m.pulse) {
			return errors.New("expired")
		}
	}
}

func NewMutex(root string, lockId string) (*Mutex, error) {
	return NewMutexExt(root, lockId, DefaultPulse, DefaultRefresh, DefaultDeadTimeout)
}

func NewMutexExt(root string, lockId string, pulse time.Duration, refresh time.Duration, deadTimeout time.Duration) (*Mutex, error) {
	if !filepath.IsAbs(root) {
		var err error
		if root, err = filepath.Abs(root); err != nil {
			return nil, err
		}
	}
	dir := path.Join(root, lockId)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("cannot create directory (%s): %w", root, err)
	}
	if pulse <= 0 {
		pulse = DefaultPulse
	}
	if refresh <= 0 {
		refresh = DefaultRefresh
	}
	return &Mutex{
		id:              strings.ToLower(lockId),
		directory:       dir,
		deadAgeRecovery: deadTimeout,
		pulse:           pulse,
		refresh:         refresh,
	}, nil
}

// LockPath returns the path of the lock file
func (m *Mutex) LockPath() string {
	return path.Join(m.directory, fmt.Sprintf(lockTemplate, m.id))
}

// When returns time of when a given mutex has been created or "zero time" if mutext is in unlocked state
func (m *Mutex) When() time.Time {
	if tm := readTimestamp(m.LockPath()); tm != 0 {
		return time.Unix(0, tm*int64(time.Millisecond))
	}
	return time.Time{}
}

func sleepOrDone(ctx context.Context, delay time.Duration) bool {
	select {
	case <-ctx.Done():
		return true
	case <-time.After(delay):
	}
	return false
}

func nano2Millis(v int64) int64 {
	return v / 1000000
}

func millis(d time.Duration) int64 {
	return nano2Millis(int64(d))
}

func now() int64 {
	return nano2Millis(time.Now().UnixNano())
}

func readTimestamp(fileName string) int64 {
	if b, err := ioutil.ReadFile(fileName); err == nil {
		if value, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64); err == nil {
			return value
		}
	}
	return 0
}

func writeCurrentTimestamp(f *os.File) (int64, error) {
	defer f.Close()
	timestamp := now()
	if _, err := f.Write([]byte(fmt.Sprintf("%d\n", timestamp))); err != nil {
		return timestamp, err
	}
	return timestamp, nil

}
