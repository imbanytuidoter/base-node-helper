# Design: Base Azul Upgrade Support

**Date:** 2026-05-12  
**Status:** Approved  
**Activation:** Mainnet ~2026-05-21 18:00 UTC (`1779386400`), Sepolia 2026-04-20 18:00 UTC (`1776708000`)

---

## Background

Base Azul is a network upgrade introducing Osaka features + TEE/ZK proof systems. After activation:

- `op-reth`, `op-geth`, `op-node`, `nethermind`, `kona` → **not supported**
- Only `base-reth-node` (EL) + `base-consensus` (CL) are supported
- All `OP_NODE_*` env vars → `BASE_NODE_*` equivalents
- 10 `OP_NODE_*` vars dropped entirely

`base-node-helper` currently hardcodes `OP_NODE_L1_ETH_RPC` / `OP_NODE_L1_BEACON` in preflight and has no `base-reth` client type — both break silently on Azul nodes.

---

## Goals

1. Warn operators on unsupported clients before activation (staged urgency)
2. Block `start` after activation unless explicit override flag
3. Fix env var reading to support both pre- and post-Azul config
4. Add `base-reth` as valid client in profile

---

## Architecture

### New package: `internal/azul/`

Single responsibility: determine Azul readiness status given network, client, and current time.

```go
package azul

import (
    "fmt"
    "time"
    "github.com/imbanytuidoter/base-node-helper/internal/config"
)

// Activation timestamps (Unix seconds, UTC)
const (
    SepoliaActivationUnix int64 = 1776708000 // 2026-04-20 18:00 UTC
    MainnetActivationUnix int64 = 1779386400 // 2026-05-21 18:00 UTC (tentative)
    UrgentWindowDays            = 7          // days before activation to show urgent warning
)

// Status describes Azul readiness for the current client/network/time.
type Status int

const (
    StatusSafe         Status = iota // client is base-reth — no action needed
    StatusPreWarning                 // legacy client, >UrgentWindowDays before activation
    StatusUrgent                     // legacy client, ≤UrgentWindowDays before activation
    StatusBlocked                    // legacy client, at or past activation time
)

// Result holds the status and a human-readable message.
type Result struct {
    Status          Status
    Message         string
    ActivationTime  time.Time
    DaysUntil       int
}

// Check returns the Azul readiness status for the given network, client, and time.
// Returns StatusSafe if the network is not mainnet/sepolia or if the client is base-reth.
func Check(network config.Network, client config.Client, now time.Time) Result
```

**Activation map** is keyed by `config.Network`. Devnet has no Azul activation → always `StatusSafe`.

**Legacy clients**: `reth`, `geth` → trigger warnings. `base-reth` → `StatusSafe`.

### Config changes (`internal/config/config.go`)

```go
const (
    ClientReth     Client = "reth"      // legacy, deprecated after Azul
    ClientGeth     Client = "geth"      // legacy, deprecated after Azul
    ClientBaseReth Client = "base-reth" // Azul-native EL client
)
```

`Validate()` updated to accept `base-reth` alongside existing clients.

`init` wizard updated to offer `base-reth` as default with a note about Azul.

### Env var dual-read (`internal/cli/start.go`)

```go
// firstNonEmpty returns the first non-empty string from candidates.
func firstNonEmpty(candidates ...string) string

// buildPreflight reads BASE_NODE_* first, falls back to OP_NODE_* for backward compat.
l1RPC := firstNonEmpty(env["BASE_NODE_L1_ETH_RPC"], env["OP_NODE_L1_ETH_RPC"])
beacon := firstNonEmpty(env["BASE_NODE_L1_BEACON"], env["OP_NODE_L1_BEACON"])
```

This handles both pre-Azul (OP_NODE_*) and post-Azul (BASE_NODE_*) `.env` files transparently.

### Command integration

Each of `start`, `doctor`, `status`, `monitor` calls `azul.Check(cfg.Network, cfg.Client, time.Now())` immediately after loading the profile.

| Status | `start` | `doctor` | `status` | `monitor` |
|--------|---------|---------|---------|---------|
| `StatusSafe` | — | — | — | — |
| `StatusPreWarning` | `WARNING` line to stderr | WARN result in report | `WARNING` line | log WARN once per interval |
| `StatusUrgent` | `WARNING` + countdown to stderr | FAIL result in report | `WARNING` + days | log WARN (every interval) |
| `StatusBlocked` | `return error` (unless `--i-understand-azul-risk`) | FAIL result | `WARNING` (non-blocking) | log ERROR (non-blocking) |

`doctor` treats `StatusUrgent` and `StatusBlocked` as `preflight.Fail` — matching severity semantics.  
`status` and `monitor` never block (display-only commands).

### Bypass flag

`start` gets `--i-understand-azul-risk bool` flag (same pattern as `down --i-understand`):

```go
cmd.Flags().BoolVar(&azulOverride, "i-understand-azul-risk",
    false, "allow start on legacy client after Azul activation (DANGEROUS)")
```

---

## Files

### New
| File | Purpose |
|------|---------|
| `internal/azul/azul.go` | `Check()`, `Status`, `Result`, activation constants |
| `internal/azul/azul_test.go` | unit tests for all Status transitions |

### Modified
| File | Change |
|------|--------|
| `internal/config/config.go` | add `ClientBaseReth`, update `validProfileName` (no change needed — already regex) |
| `internal/config/validate.go` | accept `base-reth` in client switch |
| `internal/cli/start.go` | azul check + `--i-understand-azul-risk`, dual env var read |
| `internal/cli/doctor.go` | azul check → preflight.Fail if Urgent/Blocked |
| `internal/cli/status.go` | azul check → warning line, non-blocking |
| `internal/cli/monitor.go` | azul check → log warning, non-blocking |
| `internal/cli/init.go` | add `base-reth` option in client wizard step |
| `internal/config/validate_security_test.go` | add test for `base-reth` acceptance |

---

## Test Coverage

`internal/azul/azul_test.go` covers:
- `base-reth` → always `StatusSafe` regardless of date
- Legacy client + devnet → `StatusSafe`
- Legacy client + mainnet, 30 days before → `StatusPreWarning`
- Legacy client + mainnet, 5 days before → `StatusUrgent`
- Legacy client + mainnet, day-of → `StatusBlocked`
- Legacy client + mainnet, 1 day after → `StatusBlocked`
- Message contains countdown days when `StatusUrgent`

---

## Security Invariants (unchanged)

All existing security invariants from the developer skill remain. The bypass flag follows the exact `--i-understand` pattern already established in `down.go`.

---

## Migration Note for Operators

After activating Azul (May 21):
1. `cd <base_node_repo> && git pull origin main`
2. `docker compose up -d` (no re-sync needed for existing reth users)
3. Update `~/.base-node-helper/profiles/<name>/config.yaml`: set `client: base-reth`
4. Run `bnh doctor` to verify

`bnh doctor` will show FAIL until step 3 is done.
