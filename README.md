# base-node-helper

A command-line tool that wraps `docker compose` to run a [Base](https://base.org) blockchain full node safely and reliably.

Instead of manually managing `docker compose`, you run `bnh start` — it checks that your machine is ready, launches the node, and alerts you if anything goes wrong.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/imbanytuidoter/base-node-helper/main/install.sh | sh
```

Or build from source (requires Go 1.23+):

```sh
git clone https://github.com/imbanytuidoter/base-node-helper
cd base-node-helper
go build -o bnh ./cmd/bnh
```

## Usage

```sh
bnh init       # interactive setup (run once)
bnh doctor     # check if your machine is ready
bnh start      # preflight checks + docker compose up
bnh stop       # graceful shutdown
bnh status     # container state
bnh monitor    # health polling + Discord notifications
bnh upgrade    # git pull + optional restart
bnh down       # force remove (--force --i-understand)
```

## How it works

`bnh start` runs 8 preflight checks before launching anything:

- Docker daemon running
- Required ports free (30303, 9222)
- Enough disk space (≥ 500 GB)
- Disk write speed within threshold
- System clock synced (NTP offset < 1s)
- L1 RPC endpoint reachable
- Peer count above minimum

If any check returns **FAIL**, the node does not start. **PASS** and **WARN** allow it to proceed.

`bnh monitor` polls health every 60 seconds and sends a notification (Discord or webhook) only when state changes — not on every poll.

## Configuration

```sh
bnh init
```

Stores a profile at `~/.config/base-node-helper/<name>.yaml` with your node repo path, notification settings, and thresholds.

## Platforms

Linux · macOS · Windows (WSL)
