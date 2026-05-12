package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/azul"
	"github.com/imbanytuidoter/base-node-helper/internal/compose"
	"github.com/imbanytuidoter/base-node-helper/internal/config"
	"github.com/imbanytuidoter/base-node-helper/internal/lockfile"
	"github.com/imbanytuidoter/base-node-helper/internal/status"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show node container status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gf, err := resolveGlobals(cmd)
			if err != nil {
				return err
			}
			cfg, err := config.LoadProfile(afero.NewOsFs(), gf.BaseDir, gf.Profile)
			if err != nil {
				return err
			}
			// Azul warning — non-blocking, status is display-only.
			if ar := azul.Check(cfg.Network, cfg.Client, time.Now()); ar.Status != azul.StatusSafe {
				fmt.Fprintf(cmd.ErrOrStderr(), "AZUL: %s\n", ar.Message)
			}
			lk, err := lockfile.AcquireShared(filepath.Join(gf.BaseDir, ".lock"), 2*time.Second)
			if err != nil {
				return fmt.Errorf("could not acquire shared lock: %w", err)
			}
			defer lk.Release()
			inv, err := compose.Detect(cmd.Context())
			if err != nil {
				return err
			}
			c := compose.New(inv)
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			snap, err := status.Collect(ctx, status.Options{Compose: c, Timeout: 5 * time.Second, ProjectDir: cfg.BaseNodeRepo})
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), snap.Format())
			return nil
		},
	}
}
