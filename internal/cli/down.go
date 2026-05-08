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

func newDownCmd() *cobra.Command {
	var force, iUnderstand, removeVolumes bool
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Tear down containers (DANGEROUS — prefer 'stop')",
		Long: `Runs 'docker compose down', which stops AND removes containers.
Volumes are preserved unless --remove-volumes is also given (which DESTROYS the chain DB).

This command exists for repair scenarios. For routine shutdowns, use 'stop'.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !force || !iUnderstand {
				return fmt.Errorf("'down' requires --force AND --i-understand. If you just want to stop, use 'stop' instead")
			}
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
				return err
			}
			defer lk.Release()

			inv, err := compose.Detect()
			if err != nil {
				return err
			}
			c := compose.New(inv)
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()
			if removeVolumes {
				fmt.Fprintln(cmd.ErrOrStderr(), "DESTRUCTIVE: --remove-volumes will WIPE chain data. You have 5 seconds to ctrl-C.")
				select {
				case <-time.After(5 * time.Second):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return c.Down(ctx, compose.DownOpts{
				ProjectDir:    cfg.BaseNodeRepo,
				RemoveVolumes: removeVolumes,
				Stdout:        cmd.OutOrStdout(),
				Stderr:        cmd.ErrOrStderr(),
			})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "acknowledge this is not the routine shutdown command")
	cmd.Flags().BoolVar(&iUnderstand, "i-understand", false, "acknowledge that 'down' may interact poorly with state if used carelessly")
	cmd.Flags().BoolVar(&removeVolumes, "remove-volumes", false, "DESTROY volumes (chain DB) — irreversible")
	return cmd
}
