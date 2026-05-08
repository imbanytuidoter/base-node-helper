package lockfile

import (
	"path/filepath"
	"testing"
	"time"
)

func TestExclusiveLockBlocksSecond(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	l1, err := AcquireExclusive(path, 0)
	if err != nil {
		t.Fatalf("first AcquireExclusive: %v", err)
	}
	defer l1.Release()

	l2, err := AcquireExclusive(path, 100*time.Millisecond)
	if err == nil {
		l2.Release()
		t.Fatalf("expected ErrLocked, got nil")
	}
	if !IsLocked(err) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
}

func TestSharedLocksAllowMultiple(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	l1, err := AcquireShared(path, 0)
	if err != nil {
		t.Fatalf("first AcquireShared: %v", err)
	}
	defer l1.Release()
	l2, err := AcquireShared(path, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("second AcquireShared: %v", err)
	}
	defer l2.Release()
}

func TestExclusiveBlockedByShared(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	ls, err := AcquireShared(path, 0)
	if err != nil {
		t.Fatalf("AcquireShared: %v", err)
	}
	defer ls.Release()

	_, err = AcquireExclusive(path, 100*time.Millisecond)
	if err == nil {
		t.Fatalf("expected ErrLocked")
	}
	if !IsLocked(err) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
}

func TestNonBlockingFailsImmediately(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	l1, err := AcquireExclusive(path, 0)
	if err != nil {
		t.Fatalf("first AcquireExclusive: %v", err)
	}
	defer l1.Release()

	start := time.Now()
	_, err = AcquireExclusive(path, 0)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("expected ErrLocked, got nil")
	}
	if !IsLocked(err) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("non-blocking call took %v, expected < 50ms", elapsed)
	}
}
