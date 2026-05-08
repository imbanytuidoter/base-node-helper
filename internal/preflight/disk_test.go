package preflight

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDiskSpaceInsufficient(t *testing.T) {
	c := &DiskSpaceCheck{Path: t.TempDir(), RequiredBytes: 1 << 60}
	r, _ := c.Run(context.Background())
	if r.Status != Fail {
		t.Errorf("status=%v msg=%q", r.Status, r.Message)
	}
}

func TestDiskSpaceSufficient(t *testing.T) {
	c := &DiskSpaceCheck{Path: t.TempDir(), RequiredBytes: 1 << 20}
	r, _ := c.Run(context.Background())
	if r.Status != Pass {
		t.Errorf("status=%v msg=%q", r.Status, r.Message)
	}
}

func TestDiskSpeedSlowReportsWarn(t *testing.T) {
	c := &DiskSpeedCheck{Path: filepath.Join(t.TempDir()), SampleBytes: 1 << 20, P99FailNs: 1, P99WarnNs: 1}
	r, _ := c.Run(context.Background())
	if r.Status == Pass {
		t.Errorf("expected Warn or Fail, got Pass")
	}
}
