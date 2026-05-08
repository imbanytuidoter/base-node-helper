package preflight

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
)

type FirewallCheck struct{}

func NewFirewallCheck() *FirewallCheck { return &FirewallCheck{} }

func (f *FirewallCheck) Name() string { return "firewall heuristics (advisory)" }

func (f *FirewallCheck) Run(ctx context.Context) (Result, error) {
	switch runtime.GOOS {
	case "linux":
		// Try iptables first
		out1, err1 := exec.CommandContext(ctx, "iptables", "-L", "INPUT", "-n").Output()
		if err1 == nil && len(out1) > 0 {
			s := string(out1)
			if strings.Contains(s, "DROP") || strings.Contains(s, "REJECT") || strings.Contains(s, "drop") {
				return Result{Status: Warn, Message: "iptables/nft rules contain DROP/REJECT. Confirm tcp/30303 + udp/30303 are allowed inbound."}, nil
			}
			return Result{Status: Pass, Message: "no obvious DROP rules in iptables/nftables"}, nil
		}
		// Fall back to nft
		out2, err2 := exec.CommandContext(ctx, "nft", "list", "ruleset").Output()
		if err2 == nil && len(out2) > 0 {
			s := string(out2)
			if strings.Contains(s, "DROP") || strings.Contains(s, "REJECT") || strings.Contains(s, "drop") {
				return Result{Status: Warn, Message: "iptables/nft rules contain DROP/REJECT. Confirm tcp/30303 + udp/30303 are allowed inbound."}, nil
			}
			return Result{Status: Pass, Message: "no obvious DROP rules in iptables/nftables"}, nil
		}
		// Neither available — advisory warning, not a hard failure
		return Result{Status: Warn, Message: "could not inspect firewall (need root or iptables/nft not available). Verify port 30303 inbound manually."}, nil
	case "darwin":
		out, _ := exec.CommandContext(ctx, "/usr/libexec/ApplicationFirewall/socketfilterfw", "--getglobalstate").CombinedOutput()
		if strings.Contains(string(out), "enabled") {
			return Result{Status: Warn, Message: "macOS firewall is enabled. Allow inbound for the docker process or expose ports via NAT/UPnP."}, nil
		}
		return Result{Status: Pass, Message: "macOS firewall not enforcing inbound"}, nil
	case "windows":
		return Result{Status: Warn, Message: "Windows firewall not inspected; verify tcp/30303 + udp/30303 inbound manually."}, nil
	default:
		return Result{Status: Warn, Message: "firewall check not supported on " + runtime.GOOS}, nil
	}
}
