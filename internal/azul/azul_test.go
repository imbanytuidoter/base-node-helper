package azul

import (
	"strings"
	"testing"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/config"
)

func mainnetAct() time.Time { return time.Unix(MainnetActivationUnix, 0).UTC() }

func TestCheckBaseRethAlwaysSafe(t *testing.T) {
	for _, net := range []config.Network{config.NetworkMainnet, config.NetworkSepolia, config.NetworkDevnet} {
		r := Check(net, config.ClientBaseReth, mainnetAct().Add(-24*time.Hour))
		if r.Status != StatusSafe {
			t.Errorf("base-reth on %s: got %v, want StatusSafe", net, r.Status)
		}
	}
}

func TestCheckDevnetLegacyAlwaysSafe(t *testing.T) {
	r := Check(config.NetworkDevnet, config.ClientReth, time.Now())
	if r.Status != StatusSafe {
		t.Errorf("devnet reth: got %v, want StatusSafe", r.Status)
	}
}

func TestCheckPreWarning(t *testing.T) {
	now := mainnetAct().Add(-30 * 24 * time.Hour)
	r := Check(config.NetworkMainnet, config.ClientReth, now)
	if r.Status != StatusPreWarning {
		t.Errorf("30d before: got %v, want StatusPreWarning", r.Status)
	}
	if r.DaysUntil <= UrgentWindowDays {
		t.Errorf("DaysUntil=%d should be >%d", r.DaysUntil, UrgentWindowDays)
	}
}

func TestCheckUrgent(t *testing.T) {
	now := mainnetAct().Add(-5 * 24 * time.Hour)
	r := Check(config.NetworkMainnet, config.ClientReth, now)
	if r.Status != StatusUrgent {
		t.Errorf("5d before: got %v, want StatusUrgent", r.Status)
	}
	if !strings.Contains(r.Message, "5 day(s)") {
		t.Errorf("message should contain '5 day(s)', got: %s", r.Message)
	}
}

func TestCheckUrgentGeth(t *testing.T) {
	now := mainnetAct().Add(-3 * 24 * time.Hour)
	r := Check(config.NetworkMainnet, config.ClientGeth, now)
	if r.Status != StatusUrgent {
		t.Errorf("geth 3d before: got %v, want StatusUrgent", r.Status)
	}
}

func TestCheckBlockedAtActivation(t *testing.T) {
	r := Check(config.NetworkMainnet, config.ClientReth, mainnetAct())
	if r.Status != StatusBlocked {
		t.Errorf("at activation: got %v, want StatusBlocked", r.Status)
	}
}

func TestCheckBlockedAfterActivation(t *testing.T) {
	r := Check(config.NetworkMainnet, config.ClientReth, mainnetAct().Add(24*time.Hour))
	if r.Status != StatusBlocked {
		t.Errorf("1d after: got %v, want StatusBlocked", r.Status)
	}
	if !strings.Contains(r.Message, "base-reth") {
		t.Errorf("blocked message should mention base-reth, got: %s", r.Message)
	}
}

func TestCheckSepoliaBlocked(t *testing.T) {
	sepoliaAct := time.Unix(SepoliaActivationUnix, 0).UTC()
	r := Check(config.NetworkSepolia, config.ClientReth, sepoliaAct.Add(24*time.Hour))
	if r.Status != StatusBlocked {
		t.Errorf("sepolia after act: got %v, want StatusBlocked", r.Status)
	}
}

func TestCheckUrgentBoundary(t *testing.T) {
	now := mainnetAct().Add(-time.Duration(UrgentWindowDays) * 24 * time.Hour)
	r := Check(config.NetworkMainnet, config.ClientReth, now)
	if r.Status != StatusUrgent {
		t.Errorf("at UrgentWindowDays before act: got %v, want StatusUrgent", r.Status)
	}
}

func TestCheckPreWarningGeth(t *testing.T) {
	now := mainnetAct().Add(-30 * 24 * time.Hour)
	r := Check(config.NetworkMainnet, config.ClientGeth, now)
	if r.Status != StatusPreWarning {
		t.Errorf("geth 30d before: got %v, want StatusPreWarning", r.Status)
	}
	if !strings.Contains(r.Message, "docs.base.org") {
		t.Errorf("pre-warning should include docs URL, got: %s", r.Message)
	}
}
