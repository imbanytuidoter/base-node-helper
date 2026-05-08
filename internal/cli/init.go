package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/config"
	"github.com/imbanytuidoter/base-node-helper/internal/rpc"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var offline bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive setup: writes a profile under ~/.base-node-helper/profiles/<name>/",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gf, err := resolveGlobals(cmd)
			if err != nil {
				return err
			}
			return runInit(cmd.InOrStdin(), cmd.OutOrStdout(), gf.BaseDir, offline)
		},
	}
	cmd.Flags().BoolVar(&offline, "offline", false, "skip live RPC/Beacon validation")
	return cmd
}

func runInit(in io.Reader, out io.Writer, baseDir string, offline bool) error {
	r := bufio.NewReader(in)
	ask := func(prompt, def string) string {
		if def != "" {
			fmt.Fprintf(out, "%s [%s]: ", prompt, def)
		} else {
			fmt.Fprintf(out, "%s: ", prompt)
		}
		s, _ := r.ReadString('\n')
		s = strings.TrimSpace(s)
		if s == "" {
			return def
		}
		return s
	}

	network := ""
	for {
		network = ask("Network (mainnet|sepolia|devnet)", "sepolia")
		if network == "mainnet" || network == "sepolia" || network == "devnet" {
			break
		}
		fmt.Fprintln(out, "  invalid; choose one of mainnet, sepolia, devnet")
	}

	client := ""
	for {
		client = ask("Client (reth|geth)", "reth")
		if client == "reth" || client == "geth" {
			break
		}
		fmt.Fprintln(out, "  invalid; choose reth or geth")
	}

	repo := ask("Path to cloned base/node repo", "")
	for repo == "" {
		repo = ask("Required: Path to cloned base/node repo", "")
	}
	repo = filepath.Clean(repo)

	dataDir := ask("Data directory", filepath.Join(repo, "data"))
	dataDir = filepath.Clean(dataDir)

	l1RPC := ask("L1 RPC URL (or blank to skip)", "")
	if l1RPC != "" && !offline {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cl, err := rpc.NewL1(l1RPC)
		if err == nil {
			id, err := cl.ChainID(ctx)
			if err != nil {
				fmt.Fprintf(out, "  WARN: chainId failed: %v (continuing)\n", err)
			} else {
				fmt.Fprintf(out, "  ✓ chainId=%d\n", id)
			}
		}
		cancel()
	}

	beacon := ask("L1 Beacon URL (or blank to skip)", "")
	if beacon != "" && !offline {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		bc, err := rpc.NewBeacon(beacon)
		if err == nil {
			gt, err := bc.Genesis(ctx)
			if err != nil {
				fmt.Fprintf(out, "  WARN: genesis failed: %v (continuing)\n", err)
			} else {
				fmt.Fprintf(out, "  ✓ genesis_time=%s\n", gt)
			}
		}
		cancel()
	}

	profileName := ask("Profile name", "default")
	if profileName == "" {
		profileName = "default"
	}

	stopT := 300
	if v := ask("Stop timeout seconds", strconv.Itoa(stopT)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			stopT = n
		}
	}

	prof := &config.Profile{
		Network:            config.Network(network),
		Client:             config.Client(client),
		BaseNodeRepo:       repo,
		DataDir:            dataDir,
		StopTimeoutSeconds: stopT,
		Preflight:          config.PreflightOpts{PublicIPCheck: true, DiskSpeedCheck: true},
	}
	if err := config.SaveProfile(afero.NewOsFs(), baseDir, profileName, prof); err != nil {
		return err
	}
	fmt.Fprintf(out, "→ profile saved to %s/profiles/%s/config.yaml\n", baseDir, profileName)
	return nil
}
