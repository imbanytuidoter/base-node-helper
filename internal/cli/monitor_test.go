package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestMonitorOnceFlag(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"monitor", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "--once") {
		t.Error("expected --once flag in help output")
	}
	if !strings.Contains(out.String(), "--interval") {
		t.Error("expected --interval flag in help output")
	}
}
