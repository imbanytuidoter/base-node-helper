package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestUpgradeHelp(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"upgrade", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "--restart") {
		t.Error("expected --restart flag in help")
	}
}

func TestUpgradeMissingProfile(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"upgrade", "--profile", "no-such-profile-xyzzy"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error for missing profile")
	}
}
