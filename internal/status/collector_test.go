package status

import (
	"context"
	"testing"
	"time"
)

func TestSnapshotZeroValueOK(t *testing.T) {
	s := Snapshot{}
	if s.Format() == "" {
		t.Errorf("Format on zero value should not be empty")
	}
}

func TestCollectAcceptsNilCompose(t *testing.T) {
	_, err := Collect(context.Background(), Options{Timeout: 100 * time.Millisecond})
	if err == nil {
		t.Errorf("expected error from Collect with no compose")
	}
}
