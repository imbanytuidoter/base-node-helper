package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/imbanytuidoter/base-node-helper/internal/config"
)

func TestDownRequiresConfirm(t *testing.T) {
	var stderr bytes.Buffer
	cmd := NewRoot()
	cmd.SetErr(&stderr)
	cmd.SetOut(&stderr)
	cmd.SetArgs([]string{"down"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error without --i-understand")
	}
	if !strings.Contains(err.Error(), "--i-understand") {
		t.Errorf("error msg: %v", err)
	}
}

// TestStopTimeoutFlagUpperBound verifies the integer-overflow guard (Finding 7/F4).
func TestStopTimeoutFlagUpperBound(t *testing.T) {
	tooBig := fmt.Sprintf("%d", config.MaxStopTimeoutSeconds+1)
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"stop", "--timeout", tooBig})
	err := root.Execute()
	if err == nil {
		t.Fatalf("expected error for --timeout %s, got nil", tooBig)
	}
	if !strings.Contains(err.Error(), "maximum") {
		t.Errorf("error=%v, want 'maximum'", err)
	}
}

// TestStopTimeoutFlagMaxAccepted verifies MaxStopTimeoutSeconds itself is accepted.
func TestStopTimeoutFlagMaxAccepted(t *testing.T) {
	max := fmt.Sprintf("%d", config.MaxStopTimeoutSeconds)
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"stop", "--timeout", max})
	err := root.Execute()
	// Error expected (no profile/docker), but must NOT be about max timeout.
	if err != nil && strings.Contains(err.Error(), "maximum") {
		t.Errorf("--timeout %s should not trigger the maximum error: %v", max, err)
	}
}
