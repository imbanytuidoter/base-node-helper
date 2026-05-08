package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/imbanytuidoter/base-node-helper/internal/config"
	"github.com/spf13/cobra"
)

type Globals struct {
	BaseDir string
	Profile string
}

func resolveGlobals(cmd *cobra.Command) (*Globals, error) {
	profile, _ := cmd.Root().PersistentFlags().GetString("profile")
	override, _ := cmd.Root().PersistentFlags().GetString("config")
	if override != "" {
		return &Globals{BaseDir: override, Profile: profile}, nil
	}
	d, err := config.DefaultBaseDir()
	if err != nil {
		return nil, err
	}
	return &Globals{BaseDir: d, Profile: profile}, nil
}

func readRepoEnv(repoDir string) (map[string]string, error) {
	path := filepath.Join(repoDir, ".env")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, "=")
		if i < 0 {
			continue
		}
		k := strings.TrimSpace(line[:i])
		v := strings.Trim(strings.TrimSpace(line[i+1:]), `"'`)
		out[k] = v
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read .env: %w", err)
	}
	return out, nil
}
