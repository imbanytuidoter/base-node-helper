package azul

import (
	"fmt"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/config"
)

// Azul activation timestamps (Unix seconds, UTC).
// MainnetActivationUnix is tentative — subject to change by Base team.
const (
	SepoliaActivationUnix int64 = 1776708000 // 2026-04-20 18:00 UTC
	MainnetActivationUnix int64 = 1779386400 // 2026-05-21 18:00 UTC (tentative)
	UrgentWindowDays      int   = 7          // days before activation → StatusUrgent
)

// Status describes Azul readiness for a given client/network/time.
type Status int

const (
	StatusSafe        Status = iota // client is base-reth — no action needed
	StatusPreWarning                // legacy client, >UrgentWindowDays before activation
	StatusUrgent                    // legacy client, ≤UrgentWindowDays before activation
	StatusBlocked                   // legacy client, at or past activation time
)

func (s Status) String() string {
	switch s {
	case StatusSafe:
		return "safe"
	case StatusPreWarning:
		return "pre-warning"
	case StatusUrgent:
		return "urgent"
	case StatusBlocked:
		return "blocked"
	default:
		return "unknown"
	}
}

// Result holds Azul readiness status and a human-readable message.
type Result struct {
	Status         Status
	Message        string
	ActivationTime time.Time
	DaysUntil      int
}

func activationTime(network config.Network) time.Time {
	switch network {
	case config.NetworkMainnet:
		return time.Unix(MainnetActivationUnix, 0).UTC()
	case config.NetworkSepolia:
		return time.Unix(SepoliaActivationUnix, 0).UTC()
	default:
		return time.Time{} // devnet / unknown — no activation
	}
}

func isLegacyClient(client config.Client) bool {
	return client == config.ClientReth || client == config.ClientGeth
}

// Check returns Azul readiness for the given network, client, and current time.
// Returns StatusSafe if network has no activation or client is base-reth.
func Check(network config.Network, client config.Client, now time.Time) Result {
	if !isLegacyClient(client) {
		return Result{Status: StatusSafe}
	}
	act := activationTime(network)
	if act.IsZero() {
		return Result{Status: StatusSafe}
	}
	if !now.Before(act) {
		return Result{
			Status: StatusBlocked,
			Message: fmt.Sprintf(
				"Azul activated — client %q is not supported. "+
					"Migrate: cd <base_node_repo> && git pull && docker compose up -d, "+
					"then set 'client: base-reth' in your profile.",
				client,
			),
			ActivationTime: act,
			DaysUntil:      0,
		}
	}
	daysUntil := int(act.Sub(now).Hours() / 24)
	if daysUntil <= UrgentWindowDays {
		return Result{
			Status: StatusUrgent,
			Message: fmt.Sprintf(
				"WARNING: Azul activates in %d day(s) (%s). "+
					"Client %q will stop working. Migrate to base-reth before activation.",
				daysUntil, act.Format("2006-01-02 15:04 UTC"), client,
			),
			ActivationTime: act,
			DaysUntil:      daysUntil,
		}
	}
	return Result{
		Status: StatusPreWarning,
		Message: fmt.Sprintf(
			"WARNING: Azul upgrade required. Client %q is deprecated after %s (%d days). "+
				"Plan migration to base-reth now. See: https://docs.base.org/base-chain/node-operators/base-v1-upgrade",
			client, act.Format("2006-01-02"), daysUntil,
		),
		ActivationTime: act,
		DaysUntil:      daysUntil,
	}
}
