package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/compose"
	"github.com/imbanytuidoter/base-node-helper/internal/config"
	"github.com/imbanytuidoter/base-node-helper/internal/lockfile"
	"github.com/imbanytuidoter/base-node-helper/internal/preflight"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var skipPreflight bool
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run preflight checks then start the Base node via docker compose",
		Long:  "Runs all preflight checks. If any FAIL, refuses to start (override with --skip-preflight). On PASS/WARN, runs `docker compose up -d` against base_node_repo.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd, skipPreflight)
		},
	}
	cmd.Flags().BoolVar(&skipPreflight, "skip-preflight", false, "skip preflight (DANGEROUS)")
	return cmd
}

func runStart(cmd *cobra.Command, skipPreflight bool) error {
	gf, err := resolveGlobals(cmd)
	if err != nil {
		return err
	}
	cfg, err := config.LoadProfile(afero.NewOsFs(), gf.BaseDir, gf.Profile)
	if err != nil {
		return err
	}

	lockPath := filepath.Join(gf.BaseDir, ".lock")
	if err := os.MkdirAll(gf.BaseDir, 0o700); err != nil {
		return fmt.Errorf("create base dir %s: %w", gf.BaseDir, err)
	}
	lk, err := lockfile.AcquireExclusive(lockPath, 5*time.Second)
	if err != nil {
		return fmt.Errorf("another helper command is running: %w", err)
	}
	defer lk.Release()

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
	defer cancel()

	inv, err := compose.Detect(cmd.Context())
	if err != nil {
		return err
	}

	if !skipPreflight {
		report := preflight.Run(ctx, buildPreflight(cfg))
		printReport(cmd, report)
		if report.Worst() == preflight.Fail {
			return fmt.Errorf("preflight FAILED — refusing to start. Fix issues above or pass --skip-preflight")
		}
	}

	c := compose.New(inv)
	fmt.Fprintln(cmd.OutOrStdout(), "→ docker compose up -d")
	return c.Up(ctx, compose.UpOpts{
		ProjectDir: cfg.BaseNodeRepo,
		Detach:     true,
		Stdout:     cmd.OutOrStdout(),
		Stderr:     cmd.ErrOrStderr(),
	})
}

func buildPreflight(cfg *config.Profile) []preflight.Check {
	checks := []preflight.Check{
		preflight.NewDockerCheck(),
		preflight.NewPortsCheck(),
		preflight.NewFirewallCheck(),
		&preflight.PermsCheck{Path: cfg.DataDir},
	}
	if cfg.Preflight.PublicIPCheck {
		checks = append(checks, preflight.NewPublicIPCheck())
	}
	if cfg.Preflight.DiskSpeedCheck {
		checks = append(checks, &preflight.DiskSpeedCheck{Path: cfg.DataDir})
	}
	checks = append(checks, &preflight.DiskSpaceCheck{
		Path:          cfg.DataDir,
		RequiredBytes: estimateRequiredBytes(cfg),
	})
	checks = append(checks, preflight.NewNTPCheck())
	if env, err := readRepoEnv(cfg.BaseNodeRepo); err == nil {
		if v := env["OP_NODE_L1_ETH_RPC"]; v != "" {
			checks = append(checks, &preflight.RPCCheck{URL: v, ExpectedChainID: expectedL1ChainID(cfg.Network)})
		}
		if v := env["OP_NODE_L1_BEACON"]; v != "" {
			checks = append(checks, &preflight.BeaconCheck{URL: v})
		}
	}
	return checks
}

func expectedL1ChainID(n config.Network) uint64 {
	switch n {
	case config.NetworkMainnet:
		return 1
	case config.NetworkSepolia:
		return 11155111
	}
	return 0
}

func estimateRequiredBytes(cfg *config.Profile) int64 {
	const giB = int64(1) << 30
	switch cfg.Network {
	case config.NetworkMainnet:
		return 3000 * giB
	case config.NetworkSepolia:
		return 500 * giB
	}
	return 200 * giB
}

func printReport(cmd *cobra.Command, r preflight.Report) {
	out := cmd.OutOrStdout()
	for _, x := range r.Results {
		fmt.Fprintf(out, "[%s] %s — %s\n", x.Status, x.Name, x.Message)
		if x.Fix != "" {
			fmt.Fprintf(out, "       fix: %s\n", x.Fix)
		}
	}
}
