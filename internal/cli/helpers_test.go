package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imbanytuidoter/base-node-helper/internal/config"
	"github.com/imbanytuidoter/base-node-helper/internal/preflight"
)

func TestExpectedL1ChainID(t *testing.T) {
	cases := []struct {
		network  config.Network
		expected uint64
	}{
		{config.NetworkMainnet, 1},
		{config.NetworkSepolia, 11155111},
		{config.NetworkDevnet, 0},
	}
	for _, c := range cases {
		got := expectedL1ChainID(c.network)
		if got != c.expected {
			t.Errorf("expectedL1ChainID(%q) = %d; want %d", c.network, got, c.expected)
		}
	}
}

func TestEstimateRequiredBytes(t *testing.T) {
	giB := int64(1) << 30
	cases := []struct {
		network  config.Network
		expected int64
	}{
		{config.NetworkMainnet, 3000 * giB},
		{config.NetworkSepolia, 500 * giB},
		{config.NetworkDevnet, 200 * giB},
	}
	for _, c := range cases {
		prof := &config.Profile{Network: c.network}
		got := estimateRequiredBytes(prof)
		if got != c.expected {
			t.Errorf("estimateRequiredBytes(%q) = %d; want %d", c.network, got, c.expected)
		}
	}
}

func TestPrintReport(t *testing.T) {
	var buf bytes.Buffer
	cmd := NewRoot()
	cmd.SetOut(&buf)
	report := preflight.Report{
		Results: []preflight.Result{
			{Name: "docker", Status: preflight.Pass, Message: "running"},
			{Name: "ports", Status: preflight.Warn, Message: "conflict", Fix: "free port 8545"},
			{Name: "disk", Status: preflight.Fail, Message: "insufficient"},
		},
	}
	printReport(cmd, report)
	out := buf.String()
	if !strings.Contains(out, "PASS") {
		t.Error("missing PASS in output")
	}
	if !strings.Contains(out, "WARN") {
		t.Error("missing WARN in output")
	}
	if !strings.Contains(out, "FAIL") {
		t.Error("missing FAIL in output")
	}
	if !strings.Contains(out, "free port 8545") {
		t.Error("missing fix in output")
	}
}

func TestReadRepoEnv(t *testing.T) {
	dir := t.TempDir()
	envContent := `# comment
KEY1=value1
KEY2="quoted"
KEY3='single'
BAD_LINE
`
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0o600); err != nil {
		t.Fatal(err)
	}
	env, err := readRepoEnv(dir)
	if err != nil {
		t.Fatalf("readRepoEnv: %v", err)
	}
	if env["KEY1"] != "value1" {
		t.Errorf("KEY1=%q", env["KEY1"])
	}
	if env["KEY2"] != "quoted" {
		t.Errorf("KEY2=%q", env["KEY2"])
	}
	if env["KEY3"] != "single" {
		t.Errorf("KEY3=%q", env["KEY3"])
	}
}

func TestReadRepoEnvMissing(t *testing.T) {
	_, err := readRepoEnv("/nonexistent/dir")
	if err == nil {
		t.Error("expected error for missing .env file")
	}
}

func TestResolveGlobalsDefault(t *testing.T) {
	cmd := NewRoot()
	cmd.SetArgs([]string{})
	// Parse flags by running through the root command's persistent pre-run
	// We just call the flag accessor directly
	gf, err := resolveGlobals(cmd)
	if err != nil {
		t.Fatalf("resolveGlobals: %v", err)
	}
	if gf.Profile != "default" {
		t.Errorf("profile=%q, want default", gf.Profile)
	}
	if gf.BaseDir == "" {
		t.Error("BaseDir should not be empty")
	}
}

func TestResolveGlobalsOverride(t *testing.T) {
	dir := t.TempDir()
	cmd := NewRoot()
	cmd.SetArgs([]string{"--config", dir})
	// Must parse flags first
	_ = cmd.ParseFlags([]string{"--config", dir})
	gf, err := resolveGlobals(cmd)
	if err != nil {
		t.Fatalf("resolveGlobals: %v", err)
	}
	if gf.BaseDir != dir {
		t.Errorf("BaseDir=%q, want %q", gf.BaseDir, dir)
	}
}

func TestBuildPreflightIncludesOptionals(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Profile{
		Network:      config.NetworkSepolia,
		Client:       config.ClientReth,
		BaseNodeRepo: dir,
		DataDir:      dir,
		StopTimeoutSeconds: 300,
		Preflight: config.PreflightOpts{
			PublicIPCheck:  true,
			DiskSpeedCheck: true,
		},
	}
	// Write a .env with RPC and Beacon entries
	envContent := fmt.Sprintf("OP_NODE_L1_ETH_RPC=http://localhost:8545\nOP_NODE_L1_BEACON=http://localhost:5052\n")
	os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0o600)

	checks := buildPreflight(cfg)
	// Should have at least: docker, ports, firewall, perms, publicip, diskspeed, diskspace, ntp, rpc, beacon
	if len(checks) < 8 {
		t.Errorf("expected >= 8 checks, got %d", len(checks))
	}
}
