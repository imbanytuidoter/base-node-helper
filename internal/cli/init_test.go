package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imbanytuidoter/base-node-helper/internal/config"
	"github.com/spf13/afero"
)

func TestInitWritesValidProfile(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "base-node")
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(repoDir, 0o755)
	os.MkdirAll(dataDir, 0o755)

	in := strings.NewReader(strings.Join([]string{
		"sepolia",
		"reth",
		repoDir,
		dataDir,
		"",
		"",
		"default",
	}, "\n") + "\n")

	var out bytes.Buffer
	if err := runInit(in, &out, dir, true); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	cfg, err := config.LoadProfile(afero.NewOsFs(), dir, "default")
	if err != nil {
		t.Fatalf("LoadProfile after init: %v", err)
	}
	if cfg.Network != config.NetworkSepolia {
		t.Errorf("network=%v", cfg.Network)
	}
}
