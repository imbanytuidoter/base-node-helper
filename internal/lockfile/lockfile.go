package lockfile

import (
	"errors"
	"time"

	"github.com/gofrs/flock"
)

var ErrLocked = errors.New("lockfile: already locked")

type Lock struct {
	fl *flock.Flock
}

func (l *Lock) Release() error {
	if l == nil || l.fl == nil {
		return nil
	}
	return l.fl.Unlock()
}

func IsLocked(err error) bool { return errors.Is(err, ErrLocked) }

// AcquireExclusive acquires an exclusive (write) lock on path, blocking for up to timeout.
// If timeout is zero the call is non-blocking. Returns ErrLocked if the lock cannot be
// acquired within the deadline.
func AcquireExclusive(path string, timeout time.Duration) (*Lock, error) {
	fl := flock.New(path)
	return acquire(fl, timeout, true)
}

// AcquireShared acquires a shared (read) lock on path. Multiple shared locks may coexist,
// but a shared lock blocks an exclusive lock. Same timeout semantics as AcquireExclusive.
func AcquireShared(path string, timeout time.Duration) (*Lock, error) {
	fl := flock.New(path)
	return acquire(fl, timeout, false)
}

const pollInterval = 20 * time.Millisecond

func acquire(fl *flock.Flock, timeout time.Duration, exclusive bool) (*Lock, error) {
	tryOnce := func() (bool, error) {
		if exclusive {
			return fl.TryLock()
		}
		return fl.TryRLock()
	}

	ok, err := tryOnce()
	if err != nil {
		return nil, err
	}
	if ok {
		return &Lock{fl: fl}, nil
	}
	if timeout == 0 {
		return nil, ErrLocked
	}

	deadline := time.Now().Add(timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, ErrLocked
		}
		sleep := pollInterval
		if sleep > remaining {
			sleep = remaining
		}
		time.Sleep(sleep)
		ok, err = tryOnce()
		if err != nil {
			return nil, err
		}
		if ok {
			return &Lock{fl: fl}, nil
		}
	}
}
