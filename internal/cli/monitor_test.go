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

// TestMonitorIntervalTooSmall verifies the spin-loop guard (Finding 6/F12).
func TestMonitorIntervalTooSmall(t *testing.T) {
	for _, bad := range []string{"0", "1", "9", "-5"} {
		root := NewRoot()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs([]string{"monitor", "--interval", bad, "--once"})
		err := root.Execute()
		if err == nil {
			t.Errorf("--interval %s: expected error, got nil", bad)
			continue
		}
		if !strings.Contains(err.Error(), "at least 10") {
			t.Errorf("--interval %s: error=%v, want 'at least 10'", bad, err)
		}
	}
}

// TestMonitorIntervalMinimumAccepted verifies 10 is the valid lower bound.
func TestMonitorIntervalMinimumAccepted(t *testing.T) {
	// We can't run a full monitor without docker, but we CAN verify the flag
	// passes validation by checking the error is NOT about the interval.
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"monitor", "--interval", "10", "--once"})
	err := root.Execute()
	// Error expected (no profile/docker), but must NOT be about interval.
	if err != nil && strings.Contains(err.Error(), "at least 10") {
		t.Errorf("interval=10 should not trigger the minimum-interval error: %v", err)
	}
}
