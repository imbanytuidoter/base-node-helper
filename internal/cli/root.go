package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

type GlobalFlags struct {
	Profile string
	Config  string
	Verbose bool
}

func NewRoot() *cobra.Command {
	gf := &GlobalFlags{}
	cmd := &cobra.Command{
		Use:           "base-node-helper",
		Short:         "Safe operational wrapper around base/node",
		Long:          "base-node-helper provides preflight diagnostics, safe start/stop, and status for a Base node running via the official base/node Docker Compose setup.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringVar(&gf.Profile, "profile", "default", "profile name under ~/.base-node-helper/profiles/")
	cmd.PersistentFlags().StringVar(&gf.Config, "config", "", "override path to ~/.base-node-helper")
	cmd.PersistentFlags().BoolVarP(&gf.Verbose, "verbose", "v", false, "verbose output")

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newDownCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newMonitorCmd())
	cmd.AddCommand(newUpgradeCmd())
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "base-node-helper %s (commit %s, built %s)\n", Version, Commit, Date)
			return err
		},
	}
}
