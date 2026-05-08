package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/config"
	"github.com/imbanytuidoter/base-node-helper/internal/lockfile"
	"github.com/imbanytuidoter/base-node-helper/internal/preflight"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run all preflight checks and explain failures",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gf, err := resolveGlobals(cmd)
			if err != nil {
				return err
			}
			cfg, err := config.LoadProfile(afero.NewOsFs(), gf.BaseDir, gf.Profile)
			if err != nil {
				return err
			}
			lk, err := lockfile.AcquireShared(filepath.Join(gf.BaseDir, ".lock"), 2*time.Second)
			if err != nil {
				return err
			}
			defer lk.Release()
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()
			report := preflight.Run(ctx, buildPreflight(cfg))
			printReport(cmd, report)
			fmt.Fprintf(cmd.OutOrStdout(), "\nWorst status: %s\n", report.Worst())
			if report.Worst() == preflight.Fail {
				return fmt.Errorf("at least one preflight check failed")
			}
			return nil
		},
	}
}
