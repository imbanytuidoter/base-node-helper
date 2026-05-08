# Changelog

## [v0.2.0] — 2026-05-05

### Added
- `monitor` command: continuously polls docker compose ps, L1 sync status, and peer count; sends Discord/webhook notifications on state transitions (containers-down, syncing, low peers)
- `upgrade` command: runs `git pull --ff-only` in `base_node_repo`; `--restart` flag stops and starts containers around the pull
- `internal/notify` package: generic webhook and Discord notification sender with severity filtering
- `L1.PeerCount()`: `net_peerCount` RPC method on the L1 client

### Fixed
- `install.sh`: version detection now uses `python3` JSON parser with grep/cut fallback; validates version format matches semver
- `globalFlags` unexported (was `GlobalFlags`) — no public API change
- Improved `redactedWriter.Write` godoc comment

### Changed
- `--verbose` flag now suppresses compose stdout during `start` (previously always streamed); pass `-v` to see full output

---

## v0.1.0-alpha.1 — 2026-05-04

### Added
- `init` interactive setup writing per-profile config under `~/.base-node-helper/`
- `start` runs preflight then `docker compose up -d`
- `stop` does `docker compose stop --timeout 300` (configurable per-profile)
- `down --force --i-understand` for repair scenarios
- `status` shows container state with shared lock
- `doctor` runs full preflight diagnostic
- Preflight checks: docker daemon, ports listening, public IP discovery, firewall heuristics, L1 RPC, L1 Beacon, disk space, 4K random-read p99, NTP drift, data-dir permissions
- Multi-profile via `--profile`
- Secret-redacting structured logger
- Single static binary; goreleaser-driven multi-OS release; install.sh with sha256 verify
