package cli

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/compose"
	"github.com/imbanytuidoter/base-node-helper/internal/config"
	"github.com/imbanytuidoter/base-node-helper/internal/lockfile"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func newUpgradeCmd() *cobra.Command {
	var restart bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Pull latest base/node repo updates",
		Long:  "Runs 'git pull --ff-only' in base_node_repo. Pass --restart to stop containers, pull, then start them again.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpgrade(cmd, restart)
		},
	}
	cmd.Flags().BoolVar(&restart, "restart", false, "stop and start containers around the pull")
	return cmd
}

func runUpgrade(cmd *cobra.Command, restart bool) error {
	gf, err := resolveGlobals(cmd)
	if err != nil {
		return err
	}
	cfg, err := config.LoadProfile(afero.NewOsFs(), gf.BaseDir, gf.Profile)
	if err != nil {
		return err
	}

	lk, err := lockfile.AcquireExclusive(filepath.Join(gf.BaseDir, ".lock"), 5*time.Second)
	if err != nil {
		return fmt.Errorf("another helper command is running: %w", err)
	}
	defer lk.Release()

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	inv, err := compose.Detect(cmd.Context())
	if err != nil {
		return err
	}
	c := compose.New(inv)

	// t is guaranteed > 0 by Validate() called inside LoadProfile.
	t := cfg.StopTimeoutSeconds

	if restart {
		fmt.Fprintf(cmd.OutOrStdout(), "→ docker compose stop --timeout %d\n", t)
		if err := c.Stop(ctx, compose.StopOpts{
			ProjectDir:     cfg.BaseNodeRepo,
			TimeoutSeconds: t,
			Stdout:         cmd.OutOrStdout(),
			Stderr:         cmd.ErrOrStderr(),
		}); err != nil {
			return err
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "→ git pull --ff-only")
	gitCmd := exec.CommandContext(ctx, "git", "-C", cfg.BaseNodeRepo, "pull", "--ff-only")
	gitCmd.Stdout = cmd.OutOrStdout()
	gitCmd.Stderr = cmd.ErrOrStderr()
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git pull: %w", err)
	}

	if !restart {
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "→ docker compose up -d")
	return c.Up(ctx, compose.UpOpts{
		ProjectDir: cfg.BaseNodeRepo,
		Detach:     true,
		Stdout:     cmd.OutOrStdout(),
		Stderr:     cmd.ErrOrStderr(),
	})
}
