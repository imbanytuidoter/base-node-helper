package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestStartShowsUsageWithoutProfile(t *testing.T) {
	var stderr bytes.Buffer
	cmd := NewRoot()
	cmd.SetErr(&stderr)
	cmd.SetOut(&stderr)
	cmd.SetArgs([]string{"start", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(stderr.String(), "preflight") {
		t.Errorf("help missing preflight mention: %q", stderr.String())
	}
}
