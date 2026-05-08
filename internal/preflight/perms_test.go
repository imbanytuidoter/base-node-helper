package preflight

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPermsCheckMissingDir(t *testing.T) {
	c := &PermsCheck{Path: filepath.Join(t.TempDir(), "does-not-exist")}
	r, _ := c.Run(context.Background())
	if r.Status != Warn {
		t.Errorf("status=%v", r.Status)
	}
}

func TestPermsCheckExisting(t *testing.T) {
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := &PermsCheck{Path: d}
	r, _ := c.Run(context.Background())
	if r.Status != Pass {
		t.Errorf("status=%v msg=%q", r.Status, r.Message)
	}
}
