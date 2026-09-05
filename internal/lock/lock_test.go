package lock

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireUnlock(t *testing.T) {
	p := filepath.Join(t.TempDir(), "turn.lock")
	l, err := Acquire(p, time.Second, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Unlock(); err != nil {
		t.Fatal(err)
	}
}

func TestNowaitBusy(t *testing.T) {
	p := filepath.Join(t.TempDir(), "turn.lock")
	l1, err := Acquire(p, time.Second, false)
	if err != nil {
		t.Fatal(err)
	}
	defer l1.Unlock()
	_, err = Acquire(p, time.Second, true)
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("want busy, got %v", err)
	}
}
