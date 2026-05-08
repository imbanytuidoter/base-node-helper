package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/compose"
	"github.com/imbanytuidoter/base-node-helper/internal/config"
	"github.com/imbanytuidoter/base-node-helper/internal/lockfile"
	"github.com/imbanytuidoter/base-node-helper/internal/notify"
	"github.com/imbanytuidoter/base-node-helper/internal/rpc"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type monitorState struct {
	containersDown bool
	syncing        bool
	lowPeers       bool
}

func newMonitorCmd() *cobra.Command {
	var interval int
	var once bool
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Poll node health and send notifications on threshold violations",
		Long:  "Continuously polls docker compose ps, L1 sync status, and peer count. Sends notifications (configured in profile) on state transitions. Ctrl-C to stop.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMonitor(cmd, time.Duration(interval)*time.Second, once)
		},
	}
	cmd.Flags().IntVar(&interval, "interval", 60, "polling interval in seconds")
	cmd.Flags().BoolVar(&once, "once", false, "run a single check then exit (useful for scripting)")
	return cmd
}

func runMonitor(cmd *cobra.Command, interval time.Duration, once bool) error {
	gf, err := resolveGlobals(cmd)
	if err != nil {
		return err
	}
	cfg, err := config.LoadProfile(afero.NewOsFs(), gf.BaseDir, gf.Profile)
	if err != nil {
		return err
	}

	inv, err := compose.Detect()
	if err != nil {
		return err
	}
	c := compose.New(inv)

	var l1 *rpc.L1
	if env, err := readRepoEnv(cfg.BaseNodeRepo); err == nil {
		if v := env["OP_NODE_L1_ETH_RPC"]; v != "" {
			l1, _ = rpc.NewL1(v)
		}
	}

	var prev monitorState
	first := true

	tick := func() error {
		lk, err := lockfile.AcquireShared(filepath.Join(gf.BaseDir, ".lock"), 3*time.Second)
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "warning: could not acquire lock:", err)
			return nil
		}
		defer lk.Release()

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		cur := monitorState{}

		if containers, err := c.PS(ctx, cfg.BaseNodeRepo); err == nil {
			for _, x := range containers {
				if x.State != "running" {
					cur.containersDown = true
				}
			}
		}

		if l1 != nil {
			if syncing, err := l1.Syncing(ctx); err == nil {
				cur.syncing = syncing
			}
			if cfg.Monitor.PeerCountMin > 0 {
				if peers, err := l1.PeerCount(ctx); err == nil {
					cur.lowPeers = int(peers) < cfg.Monitor.PeerCountMin
				}
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "[%s] containers_down=%v syncing=%v low_peers=%v\n",
			time.Now().Format(time.RFC3339), cur.containersDown, cur.syncing, cur.lowPeers)

		// Use a fresh context for notifications so the 30s check timeout
		// doesn't cause them to silently fail after slow RPC calls.
		notifyCtx, notifyCancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer notifyCancel()
		if (first || !prev.containersDown) && cur.containersDown {
			_ = notify.Send(notifyCtx, cfg.Notifications, "crit", "Node containers down",
				"One or more containers are not in running state")
		}
		if (first || !prev.syncing) && cur.syncing {
			_ = notify.Send(notifyCtx, cfg.Notifications, "warn", "Node syncing", "L1 node reports eth_syncing=true")
		}
		if (first || !prev.lowPeers) && cur.lowPeers {
			_ = notify.Send(notifyCtx, cfg.Notifications, "warn", "Low peer count",
				fmt.Sprintf("Peer count below minimum %d", cfg.Monitor.PeerCountMin))
		}

		prev = cur
		first = false
		return nil
	}

	if err := tick(); err != nil {
		return err
	}
	if once {
		return nil
	}
	for {
		t := time.NewTimer(interval)
		select {
		case <-cmd.Context().Done():
			t.Stop()
			return nil
		case <-t.C:
			if err := tick(); err != nil {
				return err
			}
		}
	}
}
