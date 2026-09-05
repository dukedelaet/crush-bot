package lock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

var ErrBusy = errors.New("turn lock busy")

// Lock is an exclusive flock on a bot's turn.lock.
type Lock struct {
	f *os.File
}

func Acquire(path string, timeout time.Duration, nowait bool) (*Lock, error) {
	if err := os.MkdirAll(dirOf(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open turn.lock: %w", err)
	}
	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return &Lock{f: f}, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			f.Close()
			return nil, fmt.Errorf("flock: %w", err)
		}
		if nowait {
			f.Close()
			return nil, ErrBusy
		}
		if timeout > 0 && time.Now().After(deadline) {
			f.Close()
			return nil, fmt.Errorf("%w: timed out after %s", ErrBusy, timeout)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (l *Lock) Unlock() error {
	if l == nil || l.f == nil {
		return nil
	}
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	err := l.f.Close()
	l.f = nil
	return err
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
