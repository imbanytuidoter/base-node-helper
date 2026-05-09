package config

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// maxStopTimeoutSeconds prevents integer overflow in context.Duration arithmetic.
const maxStopTimeoutSeconds = 86400 // 24 h

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
	// Accept Unix-style /path (primary deployment target) and Windows C:\path.
	if p.BaseNodeRepo == "" {
		return fmt.Errorf("base_node_repo is required")
	}
	if !strings.HasPrefix(p.BaseNodeRepo, "/") && !filepath.IsAbs(p.BaseNodeRepo) {
		return fmt.Errorf("base_node_repo must be an absolute path, got %q", p.BaseNodeRepo)
	}
	if filepath.Clean(p.BaseNodeRepo) != p.BaseNodeRepo &&
		filepath.ToSlash(filepath.Clean(p.BaseNodeRepo)) != filepath.ToSlash(p.BaseNodeRepo) {
		return fmt.Errorf("base_node_repo contains unclean path components: %q", p.BaseNodeRepo)
	}

	if p.DataDir == "" {
		return fmt.Errorf("data_dir is required")
	}

	// [HIGH] integer-overflow: very large values cause time.Duration arithmetic
	// to overflow into a negative number, cancelling the context immediately.
	if p.StopTimeoutSeconds <= 0 {
		return fmt.Errorf("stop_timeout_seconds must be > 0 (recommend 300)")
	}
	if p.StopTimeoutSeconds > maxStopTimeoutSeconds {
		return fmt.Errorf("stop_timeout_seconds too large (max %d)", maxStopTimeoutSeconds)
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
