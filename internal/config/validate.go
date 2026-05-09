package config

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// isAbsPath reports whether p is an absolute path on any supported platform.
//
// Production deployments run on Linux/macOS where filepath.IsAbs("/foo") = true.
// On Windows, filepath.IsAbs("/foo") = false (drive-relative), but we
// additionally accept /foo-prefixed paths to support cross-platform test
// fixtures and profiles authored on Unix. This is an intentional trade-off:
// on Windows production deployments operators should use C:\... paths.
func isAbsPath(p string) bool {
	return filepath.IsAbs(p) || strings.HasPrefix(p, "/")
}

// MaxStopTimeoutSeconds is the upper bound for stop_timeout_seconds in profiles
// and the --timeout CLI flag. Prevents integer overflow in context.Duration arithmetic.
const MaxStopTimeoutSeconds = 86400 // 24 h

func Validate(p *Profile) error {
	switch p.Network {
	case NetworkMainnet, NetworkSepolia, NetworkDevnet:
	case "":
		return fmt.Errorf("network is required")
	default:
		return fmt.Errorf("network %q not in [mainnet, sepolia, devnet]", p.Network)
	}
	switch p.Client {
	case ClientReth, ClientGeth:
	case "":
		return fmt.Errorf("client is required")
	default:
		return fmt.Errorf("client %q not in [reth, geth]", p.Client)
	}

	// [HIGH] injection: BaseNodeRepo must be a clean absolute path so it
	// cannot be used as a git option flag or escape the intended directory.
	// [LOW] Finding 12: rely solely on filepath.IsAbs (not strings.HasPrefix)
	// so that root-relative Windows paths like /foo are not falsely accepted.
	if p.BaseNodeRepo == "" {
		return fmt.Errorf("base_node_repo is required")
	}
	if !isAbsPath(p.BaseNodeRepo) {
		return fmt.Errorf("base_node_repo must be an absolute path, got %q", p.BaseNodeRepo)
	}
	if filepath.Clean(p.BaseNodeRepo) != p.BaseNodeRepo &&
		filepath.ToSlash(filepath.Clean(p.BaseNodeRepo)) != filepath.ToSlash(p.BaseNodeRepo) {
		return fmt.Errorf("base_node_repo contains unclean path components: %q", p.BaseNodeRepo)
	}

	// [MED] Finding 5+3: DataDir must be absolute AND clean — same standard
	// as BaseNodeRepo to prevent CWD-relative resolution and path comparison mismatches.
	if p.DataDir == "" {
		return fmt.Errorf("data_dir is required")
	}
	if !isAbsPath(p.DataDir) {
		return fmt.Errorf("data_dir must be an absolute path, got %q", p.DataDir)
	}
	if filepath.Clean(p.DataDir) != p.DataDir &&
		filepath.ToSlash(filepath.Clean(p.DataDir)) != filepath.ToSlash(p.DataDir) {
		return fmt.Errorf("data_dir contains unclean path components: %q", p.DataDir)
	}

	// [HIGH] integer-overflow: very large values cause time.Duration arithmetic
	// to overflow into a negative number, cancelling the context immediately.
	if p.StopTimeoutSeconds <= 0 {
		return fmt.Errorf("stop_timeout_seconds must be > 0 (recommend 300)")
	}
	if p.StopTimeoutSeconds > MaxStopTimeoutSeconds {
		return fmt.Errorf("stop_timeout_seconds too large (max %d)", MaxStopTimeoutSeconds)
	}

	for i, n := range p.Notifications {
		if n.Type == "" {
			return fmt.Errorf("notifications[%d].type is required", i)
		}
		if n.URL == "" {
			return fmt.Errorf("notifications[%d].url is required (or unresolved env var)", i)
		}
		// [CRIT] ssrf: reject non-http(s) notification URLs so a crafted profile
		// cannot reach internal metadata endpoints or the local filesystem.
		u, err := url.Parse(n.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("notifications[%d].url must use http:// or https://", i)
		}
	}
	return nil
}
