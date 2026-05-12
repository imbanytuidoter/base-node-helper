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
	"github.com/imbanytuidoter/base-node-helper/internal/log"
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
			// [LOW] Finding 10: pass cmd.Context() so Ctrl-C during RPC validation
			// cancels the outbound request rather than waiting for the 5s timeout.
			return runInit(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), gf.BaseDir, offline)
		},
	}
	cmd.Flags().BoolVar(&offline, "offline", false, "skip live RPC/Beacon validation")
	return cmd
}

func runInit(ctx context.Context, in io.Reader, out io.Writer, baseDir string, offline bool) error {
	r := bufio.NewReader(in)
	ask := func(prompt, def string) string {
		if def != "" {
			fmt.Fprintf(out, "%s [%s]: ", prompt, def)
		} else {
			fmt.Fprintf(out, "%s: ", prompt)
		}
		// [LOW] error-ignored: io.EOF with partial content is acceptable
		// (piped input ending without newline); other errors abort cleanly.
		s, err := r.ReadString('\n')
		if err != nil && len(s) == 0 {
			return def
		}
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
		client = ask("Client (reth|geth|base-reth) [base-reth recommended for Azul]", "base-reth")
		if client == "reth" || client == "geth" || client == "base-reth" {
			break
		}
		fmt.Fprintln(out, "  invalid; choose reth, geth, or base-reth")
	}
	if client != "base-reth" {
		fmt.Fprintf(out, "  NOTE: %q is deprecated after Azul activation (~2026-05-21). "+
			"Consider migrating to base-reth. See: https://docs.base.org/base-chain/node-operators/base-v1-upgrade\n", client)
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
		// [LOW-F5] defer cancel() so a panic cannot leak the timer goroutine.
		rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		cl, err := rpc.NewL1(l1RPC)
		if err != nil {
			// [MED-F1] Redact the error: NewL1 embeds rawURL which may contain
			// an API key (e.g. ws://host/v2/SECRETKEY).
			fmt.Fprintf(out, "  WARN: RPC URL rejected: %s (continuing)\n", log.Redact(err.Error()))
		} else {
			id, err := cl.ChainID(rpcCtx)
			if err != nil {
				fmt.Fprintf(out, "  WARN: chainId failed: %s (continuing)\n", log.Redact(err.Error()))
			} else {
				fmt.Fprintf(out, "  ✓ chainId=%d\n", id)
			}
		}
	}

	beacon := ask("L1 Beacon URL (or blank to skip)", "")
	if beacon != "" && !offline {
		bcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		bc, err := rpc.NewBeacon(beacon)
		if err != nil {
			// [MED-F1] Redact Beacon URL rejection for same reason.
			fmt.Fprintf(out, "  WARN: Beacon URL rejected: %s (continuing)\n", log.Redact(err.Error()))
		} else {
			gt, err := bc.Genesis(bcCtx)
			if err != nil {
				fmt.Fprintf(out, "  WARN: genesis failed: %s (continuing)\n", log.Redact(err.Error()))
			} else {
				fmt.Fprintf(out, "  ✓ genesis_time=%s\n", gt)
			}
		}
	}

	profileName := ask("Profile name", "default")
	if profileName == "" {
		profileName = "default"
	}

	stopT := 300
	for {
		v := ask("Stop timeout seconds", strconv.Itoa(stopT))
		if v == "" {
			break
		}
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			fmt.Fprintf(out, "  invalid; enter a positive integer\n")
			continue
		}
		// [MED-F2] enforce MaxStopTimeoutSeconds here so the profile is valid
		// immediately, rather than failing at runtime with a confusing error.
		if n > config.MaxStopTimeoutSeconds {
			fmt.Fprintf(out, "  invalid; maximum allowed is %d seconds\n", config.MaxStopTimeoutSeconds)
			continue
		}
		stopT = n
		break
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
