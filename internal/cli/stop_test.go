package cli

import (
	"bytes"
	"strings"
	"testing"
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
