package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootShowsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewRoot()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if stderr.Len() > 0 {
		t.Errorf("unexpected stderr: %q", stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "base-node-helper") {
		t.Errorf("help missing app name: %q", out)
	}
	if !strings.Contains(out, "--profile") {
		t.Errorf("help missing --profile flag: %q", out)
	}
	if !strings.Contains(out, "--verbose") {
		t.Errorf("help missing --verbose flag: %q", out)
	}
}

func TestVersionSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewRoot()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if stderr.Len() > 0 {
		t.Errorf("unexpected stderr: %q", stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"base-node-helper", "(commit", "built"} {
		if !strings.Contains(out, want) {
			t.Errorf("version output missing %q: got %q", want, out)
		}
	}
}
