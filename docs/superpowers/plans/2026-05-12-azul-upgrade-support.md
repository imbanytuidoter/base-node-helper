# Azul Upgrade Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Base Azul upgrade awareness to base-node-helper — warn operators on legacy clients before activation, block start after activation, fix env var reading for post-Azul nodes.

**Architecture:** New `internal/azul` package with a pure `Check(network, client, now)` function returns a `Result{Status, Message, DaysUntil}`. Four commands (`start`, `doctor`, `status`, `monitor`) call it after `LoadProfile` and react according to their severity level. `firstNonEmpty` helper in `helpers.go` enables transparent dual `BASE_NODE_*`/`OP_NODE_*` env var reading.

**Tech Stack:** Go 1.23, Cobra, standard library only (no new deps).

---

## File Map

| Action | File | What changes |
|--------|------|-------------|
| Create | `internal/azul/azul.go` | `Status`, `Result`, `Check()`, activation constants |
| Create | `internal/azul/azul_test.go` | 8 unit tests covering all Status transitions |
| Modify | `internal/config/config.go` | Add `ClientBaseReth = "base-reth"` constant |
| Modify | `internal/config/validate.go` | Accept `base-reth` in client switch |
| Modify | `internal/config/validate_security_test.go` | Test `base-reth` accepted, error message updated |
| Modify | `internal/cli/helpers.go` | Add `firstNonEmpty()` helper |
| Modify | `internal/cli/helpers_test.go` | Test `firstNonEmpty` |
| Modify | `internal/cli/start.go` | Azul check, `--i-understand-azul-risk`, dual env var |
| Modify | `internal/cli/doctor.go` | Azul check → fail if Urgent/Blocked |
| Modify | `internal/cli/status.go` | Azul check → warning, non-blocking |
| Modify | `internal/cli/monitor.go` | Azul check + dual env var in `runMonitor` |
| Modify | `internal/cli/init.go` | Add `base-reth` option in client wizard |
| Modify | `.claude/skills/bnh-developer.md` | Document Azul support, new constants, new command flags |

---

## Task 1: Create `internal/azul` package (TDD)

**Files:**
- Create: `internal/azul/azul_test.go`
- Create: `internal/azul/azul.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/azul/azul_test.go
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
```

- [ ] **Step 2: Run tests — verify they fail (package missing)**

```
go test ./internal/azul/...
```

Expected: `cannot find package "github.com/imbanytuidoter/base-node-helper/internal/azul"`

- [ ] **Step 3: Implement `internal/azul/azul.go`**

```go
// internal/azul/azul.go
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
```

- [ ] **Step 4: Run tests — verify they pass**

```
go test ./internal/azul/... -v
```

Expected: 8 tests PASS.

- [ ] **Step 5: Commit**

```
git add internal/azul/
git commit -m "feat(azul): add Azul readiness check package with 8 unit tests"
```

---

## Task 2: Add `ClientBaseReth` to config + validate

**Files:**
- Modify: `internal/config/config.go` (line 25–28)
- Modify: `internal/config/validate.go` (line 33–38)
- Modify: `internal/config/validate_security_test.go`

- [ ] **Step 1: Write failing test for base-reth acceptance**

Add to `internal/config/validate_security_test.go`:

```go
func TestValidateClientBaseRethAccepted(t *testing.T) {
	p := baseValidProfile()
	p.Client = ClientBaseReth
	if err := Validate(p); err != nil {
		t.Errorf("base-reth client should be accepted: %v", err)
	}
}

func TestValidateClientUnknownRejected(t *testing.T) {
	p := baseValidProfile()
	p.Client = "op-reth"
	if err := Validate(p); err == nil {
		t.Error("expected error for unknown client 'op-reth'")
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

```
go test ./internal/config/... -run TestValidateClient -v
```

Expected: `FAIL` — `ClientBaseReth` undefined.

- [ ] **Step 3: Add `ClientBaseReth` to `internal/config/config.go`**

Find the Client constants block (around line 25) and replace:

```go
const (
	ClientReth     Client = "reth"      // legacy EL client — deprecated after Azul
	ClientGeth     Client = "geth"      // legacy EL client — deprecated after Azul
	ClientBaseReth Client = "base-reth" // Azul-native EL client (required post-activation)
)
```

- [ ] **Step 4: Update client switch in `internal/config/validate.go`**

Replace the existing `switch p.Client` block:

```go
switch p.Client {
case ClientReth, ClientGeth, ClientBaseReth:
case "":
	return fmt.Errorf("client is required")
default:
	return fmt.Errorf("client %q not in [reth, geth, base-reth]", p.Client)
}
```

- [ ] **Step 5: Run all config tests**

```
go test ./internal/config/... -v
```

Expected: all tests PASS including new `TestValidateClientBaseRethAccepted`.

- [ ] **Step 6: Commit**

```
git add internal/config/config.go internal/config/validate.go internal/config/validate_security_test.go
git commit -m "feat(config): add ClientBaseReth for Azul-native EL client"
```

---

## Task 3: Add `firstNonEmpty` helper

**Files:**
- Modify: `internal/cli/helpers.go`
- Modify: `internal/cli/helpers_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/cli/helpers_test.go`:

```go
func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("a", "b"); got != "a" {
		t.Errorf("got %q, want 'a'", got)
	}
	if got := firstNonEmpty("", "b"); got != "b" {
		t.Errorf("got %q, want 'b'", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("got %q, want ''", got)
	}
	if got := firstNonEmpty(); got != "" {
		t.Errorf("got %q, want ''", got)
	}
	// BASE_NODE_* takes priority over OP_NODE_*
	env := map[string]string{
		"BASE_NODE_L1_ETH_RPC": "https://base-rpc.example.com",
		"OP_NODE_L1_ETH_RPC":   "https://op-rpc.example.com",
	}
	got := firstNonEmpty(env["BASE_NODE_L1_ETH_RPC"], env["OP_NODE_L1_ETH_RPC"])
	if got != "https://base-rpc.example.com" {
		t.Errorf("got %q, want base-rpc URL", got)
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

```
go test ./internal/cli/... -run TestFirstNonEmpty -v
```

Expected: `FAIL` — `firstNonEmpty` undefined.

- [ ] **Step 3: Add `firstNonEmpty` to `internal/cli/helpers.go`**

Append to the end of `helpers.go`:

```go
// firstNonEmpty returns the first non-empty string from candidates.
// Used to prefer BASE_NODE_* env vars over legacy OP_NODE_* with fallback.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
```

- [ ] **Step 4: Run tests**

```
go test ./internal/cli/... -run TestFirstNonEmpty -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```
git add internal/cli/helpers.go internal/cli/helpers_test.go
git commit -m "feat(cli): add firstNonEmpty helper for dual BASE_NODE_*/OP_NODE_* env var reading"
```

---

## Task 4: Update `start.go` — Azul check + dual env vars

**Files:**
- Modify: `internal/cli/start.go`

- [ ] **Step 1: Add `--i-understand-azul-risk` flag to `newStartCmd`**

Replace the `newStartCmd` function:

```go
func newStartCmd() *cobra.Command {
	var skipPreflight bool
	var azulOverride bool
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run preflight checks then start the Base node via docker compose",
		Long:  "Runs all preflight checks. If any FAIL, refuses to start (override with --skip-preflight). On PASS/WARN, runs `docker compose up -d` against base_node_repo.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd, skipPreflight, azulOverride)
		},
	}
	cmd.Flags().BoolVar(&skipPreflight, "skip-preflight", false, "skip preflight (DANGEROUS)")
	cmd.Flags().BoolVar(&azulOverride, "i-understand-azul-risk",
		false, "allow start on legacy client after Azul activation (DANGEROUS)")
	return cmd
}
```

- [ ] **Step 2: Update `runStart` signature and add Azul check**

Replace `func runStart(cmd *cobra.Command, skipPreflight bool) error` with:

```go
func runStart(cmd *cobra.Command, skipPreflight bool, azulOverride bool) error {
	gf, err := resolveGlobals(cmd)
	if err != nil {
		return err
	}
	cfg, err := config.LoadProfile(afero.NewOsFs(), gf.BaseDir, gf.Profile)
	if err != nil {
		return err
	}

	// Azul upgrade readiness check — runs before preflight to give clear actionable error.
	ar := azul.Check(cfg.Network, cfg.Client, time.Now())
	switch ar.Status {
	case azul.StatusPreWarning, azul.StatusUrgent:
		fmt.Fprintf(cmd.ErrOrStderr(), "AZUL: %s\n", ar.Message)
	case azul.StatusBlocked:
		if !azulOverride {
			return fmt.Errorf("AZUL: %s\nTo override (DANGEROUS): add --i-understand-azul-risk", ar.Message)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "AZUL OVERRIDE: proceeding with legacy client after Azul activation\n")
	}

	lockPath := filepath.Join(gf.BaseDir, ".lock")
	if err := os.MkdirAll(gf.BaseDir, 0o700); err != nil {
		return fmt.Errorf("create base dir %s: %w", gf.BaseDir, err)
	}
	lk, err := lockfile.AcquireExclusive(lockPath, 5*time.Second)
	if err != nil {
		return fmt.Errorf("another helper command is running: %w", err)
	}
	defer lk.Release()

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
	defer cancel()

	inv, err := compose.Detect(cmd.Context())
	if err != nil {
		return err
	}

	if !skipPreflight {
		report := preflight.Run(ctx, buildPreflight(cfg))
		printReport(cmd, report)
		if report.Worst() == preflight.Fail {
			return fmt.Errorf("preflight FAILED — refusing to start. Fix issues above or pass --skip-preflight")
		}
	}

	c := compose.New(inv)
	fmt.Fprintln(cmd.OutOrStdout(), "→ docker compose up -d")
	return c.Up(ctx, compose.UpOpts{
		ProjectDir: cfg.BaseNodeRepo,
		Detach:     true,
		Stdout:     cmd.OutOrStdout(),
		Stderr:     cmd.ErrOrStderr(),
	})
}
```

- [ ] **Step 3: Update `buildPreflight` to use dual env var reading**

Replace the env block inside `buildPreflight` (the `if env, err := readRepoEnv` block):

```go
	if env, err := readRepoEnv(cfg.BaseNodeRepo); err == nil {
		if v := firstNonEmpty(env["BASE_NODE_L1_ETH_RPC"], env["OP_NODE_L1_ETH_RPC"]); v != "" {
			checks = append(checks, &preflight.RPCCheck{URL: v, ExpectedChainID: expectedL1ChainID(cfg.Network)})
		}
		if v := firstNonEmpty(env["BASE_NODE_L1_BEACON"], env["OP_NODE_L1_BEACON"]); v != "" {
			checks = append(checks, &preflight.BeaconCheck{URL: v})
		}
	}
```

- [ ] **Step 4: Add import for `azul` package**

Add to the import block in `start.go`:

```go
"github.com/imbanytuidoter/base-node-helper/internal/azul"
```

- [ ] **Step 5: Build and run tests**

```
go build ./...
go test ./internal/cli/... -v
```

Expected: build succeeds, all tests pass.

- [ ] **Step 6: Commit**

```
git add internal/cli/start.go
git commit -m "feat(start): add Azul check, --i-understand-azul-risk flag, dual BASE_NODE_*/OP_NODE_* env reading"
```

---

## Task 5: Update `doctor.go` — Azul check (blocking)

**Files:**
- Modify: `internal/cli/doctor.go`

- [ ] **Step 1: Update `newDoctorCmd` RunE**

Replace the full `RunE` body:

```go
RunE: func(cmd *cobra.Command, _ []string) error {
    gf, err := resolveGlobals(cmd)
    if err != nil {
        return err
    }
    cfg, err := config.LoadProfile(afero.NewOsFs(), gf.BaseDir, gf.Profile)
    if err != nil {
        return err
    }

    // Azul check — printed before preflight so it's the first thing seen.
    ar := azul.Check(cfg.Network, cfg.Client, time.Now())
    if ar.Status != azul.StatusSafe {
        fmt.Fprintf(cmd.ErrOrStderr(), "AZUL: %s\n\n", ar.Message)
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
    // Azul Urgent/Blocked → fail doctor even if all other preflight passed.
    if ar.Status == azul.StatusUrgent || ar.Status == azul.StatusBlocked {
        return fmt.Errorf("Azul upgrade required: migrate client to base-reth before %s",
            ar.ActivationTime.Format("2006-01-02"))
    }
    return nil
},
```

- [ ] **Step 2: Add import for `azul` package**

Add to the import block in `doctor.go`:

```go
"github.com/imbanytuidoter/base-node-helper/internal/azul"
```

- [ ] **Step 3: Build and test**

```
go build ./...
go test ./internal/cli/... -v
```

Expected: all pass.

- [ ] **Step 4: Commit**

```
git add internal/cli/doctor.go
git commit -m "feat(doctor): add Azul check — urgent/blocked clients fail doctor"
```

---

## Task 6: Update `status.go` — Azul warning (non-blocking)

**Files:**
- Modify: `internal/cli/status.go`

- [ ] **Step 1: Add Azul check to `RunE` in `newStatusCmd`**

After `config.LoadProfile` and before `lockfile.AcquireShared`, insert:

```go
// Azul warning — non-blocking, status is display-only.
if ar := azul.Check(cfg.Network, cfg.Client, time.Now()); ar.Status != azul.StatusSafe {
    fmt.Fprintf(cmd.ErrOrStderr(), "AZUL: %s\n", ar.Message)
}
```

- [ ] **Step 2: Add import for `azul` package**

Add to import block:

```go
"github.com/imbanytuidoter/base-node-helper/internal/azul"
```

- [ ] **Step 3: Build and test**

```
go build ./...
go test ./internal/cli/... -v
```

Expected: all pass.

- [ ] **Step 4: Commit**

```
git add internal/cli/status.go
git commit -m "feat(status): add Azul warning output (non-blocking)"
```

---

## Task 7: Update `monitor.go` — Azul warning + dual env vars

**Files:**
- Modify: `internal/cli/monitor.go`

- [ ] **Step 1: Add Azul check at top of `runMonitor`**

After `config.LoadProfile` (line 50 area), insert:

```go
// Azul warning — printed once at monitor start, non-blocking.
if ar := azul.Check(cfg.Network, cfg.Client, time.Now()); ar.Status != azul.StatusSafe {
    fmt.Fprintf(cmd.ErrOrStderr(), "AZUL: %s\n", ar.Message)
}
```

- [ ] **Step 2: Update env var reading in `runMonitor`**

Replace the existing env var block (around line 61–70):

```go
var l1 *rpc.L1
if env, err := readRepoEnv(cfg.BaseNodeRepo); err == nil {
    if v := firstNonEmpty(env["BASE_NODE_L1_ETH_RPC"], env["OP_NODE_L1_ETH_RPC"]); v != "" {
        var l1Err error
        l1, l1Err = rpc.NewL1(v)
        if l1Err != nil {
            fmt.Fprintf(cmd.ErrOrStderr(), "warning: invalid L1 ETH RPC URL: %v (sync/peer checks disabled)\n", l1Err)
        }
    }
}
```

- [ ] **Step 3: Add import for `azul` package**

Add to import block:

```go
"github.com/imbanytuidoter/base-node-helper/internal/azul"
```

- [ ] **Step 4: Build and run all tests**

```
go build ./...
go test ./... -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```
git add internal/cli/monitor.go
git commit -m "feat(monitor): add Azul warning and dual BASE_NODE_*/OP_NODE_* env var reading"
```

---

## Task 8: Update `init.go` — add `base-reth` to wizard

**Files:**
- Modify: `internal/cli/init.go`

- [ ] **Step 1: Update client prompt loop**

Find the client prompt block (around line 69–76) and replace:

```go
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
```

- [ ] **Step 2: Verify init test still passes**

```
go test ./internal/cli/... -run TestInit -v
```

Expected: PASS (existing tests use piped input that provides valid values).

- [ ] **Step 3: Build**

```
go build ./...
```

- [ ] **Step 4: Commit**

```
git add internal/cli/init.go
git commit -m "feat(init): add base-reth client option with Azul deprecation notice for legacy clients"
```

---

## Task 9: Coverage check + final tests

- [ ] **Step 1: Run full test suite with coverage**

```
go test -coverprofile=coverage.txt -covermode=atomic ./...
go tool cover -func=coverage.txt | tail -1
```

Expected: total coverage ≥ 60%.

- [ ] **Step 2: If coverage < 60%, add targeted tests**

Add to `internal/azul/azul_test.go` (boundary at exactly UrgentWindowDays):

```go
func TestCheckUrgentBoundary(t *testing.T) {
    // Exactly UrgentWindowDays before activation → still Urgent (not PreWarning)
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
```

- [ ] **Step 3: Final build verification**

```
go build ./...
go vet ./...
```

Expected: no errors.

- [ ] **Step 4: Commit coverage fix if needed**

```
git add internal/azul/azul_test.go
git commit -m "test(azul): add boundary and geth coverage tests"
```

---

## Task 10: Update developer skill + create PR

**Files:**
- Modify: `.claude/skills/bnh-developer.md`

- [ ] **Step 1: Add Azul section to skill**

Add to `## Key Constants & Conventions` table:

```markdown
| `SepoliaActivationUnix` | `1776708000` | `azul/azul.go` | Sepolia Azul activation (2026-04-20 18:00 UTC) |
| `MainnetActivationUnix` | `1779386400` | `azul/azul.go` | Mainnet Azul activation (~2026-05-21 18:00 UTC) |
| `UrgentWindowDays` | `7` | `azul/azul.go` | Days before activation to show urgent warning |
| `ClientBaseReth` | `"base-reth"` | `config/config.go` | Azul-native EL client |
```

Add to `## Package Architecture`:

```
  azul/          — Azul upgrade readiness check (Status, Result, Check())
```

Add to `## Security Invariants`:

```markdown
### 11. Azul upgrade gate (`internal/azul/azul.go`, `internal/cli/start.go`)
- `start` **must** call `azul.Check()` and return error if `StatusBlocked` without `--i-understand-azul-risk`
- `doctor` **must** fail if `StatusUrgent` or `StatusBlocked`
- `status` and `monitor` show warning but **never block**
- Activation timestamps are constants — never derive from external source
```

- [ ] **Step 2: Commit skill update**

```
git add .claude/skills/bnh-developer.md
git commit -m "docs(skill): add Azul upgrade support to bnh-developer skill"
```

- [ ] **Step 3: Push branch and open PR**

```
git checkout -b audit-fix-20260512
git push -u origin audit-fix-20260512
gh pr create \
  --title ":zap: feat: Base Azul upgrade support + full code audit fixes" \
  --body "..."
```

---

## Self-Review

**Spec coverage check:**
- ✅ `internal/azul/` package with `Check()` → Task 1
- ✅ `StatusSafe/PreWarning/Urgent/Blocked` → Task 1
- ✅ `ClientBaseReth = "base-reth"` → Task 2
- ✅ `validate.go` accepts base-reth → Task 2
- ✅ `firstNonEmpty` helper → Task 3
- ✅ `start` Azul check + `--i-understand-azul-risk` → Task 4
- ✅ Dual `BASE_NODE_*`/`OP_NODE_*` in buildPreflight → Task 4
- ✅ `doctor` fails on Urgent/Blocked → Task 5
- ✅ `status` warning non-blocking → Task 6
- ✅ `monitor` warning + dual env vars → Task 7
- ✅ `init` base-reth option → Task 8
- ✅ Coverage gate ≥ 60% → Task 9
- ✅ Skill updated → Task 10

**Placeholder scan:** No TBD, no "implement later", all code complete.

**Type consistency:**
- `azul.Status` / `azul.Result` / `azul.Check()` — consistent across Tasks 1, 4, 5, 6, 7
- `azul.StatusSafe/PreWarning/Urgent/Blocked` — defined Task 1, used Tasks 4–7
- `firstNonEmpty()` — defined Task 3, used Tasks 4, 7
- `ClientBaseReth` — defined Task 2, referenced in azul.go `isLegacyClient`
