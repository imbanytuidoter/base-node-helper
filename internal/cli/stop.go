package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/compose"
	"github.com/imbanytuidoter/base-node-helper/internal/config"
	"github.com/imbanytuidoter/base-node-helper/internal/lockfile"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	var timeoutOverride int
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Gracefully stop the Base node (docker compose stop --timeout N)",
		Long:  "Sends SIGTERM with a long timeout (default 300s, configurable per-profile) to allow Reth/op-node to flush state cleanly. After timeout SIGKILL is sent — which can prolong next-start MDBX recovery.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gf, err := resolveGlobals(cmd)
			if err != nil {
				return err
			}
			cfg, err := config.LoadProfile(afero.NewOsFs(), gf.BaseDir, gf.Profile)
			if err != nil {
				return err
			}
			t := cfg.StopTimeoutSeconds
			if timeoutOverride > 0 {
				// [MED] Finding 4: --timeout flag must be bounded like the profile field,
				// otherwise large values cause time.Duration overflow → immediate ctx cancel.
				if timeoutOverride > config.MaxStopTimeoutSeconds {
					return fmt.Errorf("--timeout %d exceeds maximum allowed %d seconds",
						timeoutOverride, config.MaxStopTimeoutSeconds)
				}
				t = timeoutOverride
			}
			lk, err := lockfile.AcquireExclusive(filepath.Join(gf.BaseDir, ".lock"), 5*time.Second)
			if err != nil {
				return err
			}
			defer lk.Release()

			inv, err := compose.Detect(cmd.Context())
			if err != nil {
				return err
			}
			c := compose.New(inv)
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(t+30)*time.Second)
			defer cancel()
			fmt.Fprintf(cmd.OutOrStdout(), "→ docker compose stop --timeout %d\n", t)
			if err := c.Stop(ctx, compose.StopOpts{
				ProjectDir:     cfg.BaseNodeRepo,
				TimeoutSeconds: t,
				Stdout:         cmd.OutOrStdout(),
				Stderr:         cmd.ErrOrStderr(),
			}); err != nil {
				return err
			}
			// [MED] context: use cmd.Context() so Ctrl-C cancels this too.
			psCtx, psCancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer psCancel()
			containers, err := c.PS(psCtx, cfg.BaseNodeRepo)
			if err == nil {
				for _, x := range containers {
					if x.ExitCode != nil && *x.ExitCode == 137 {
						fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: %s exited with 137 (SIGKILL) — increase stop_timeout_seconds in profile to prevent dirty shutdown next time\n", x.Service)
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&timeoutOverride, "timeout", 0, "override stop timeout in seconds (default: profile.stop_timeout_seconds)")
	return cmd
}
