package config

import "fmt"

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
	if p.BaseNodeRepo == "" {
		return fmt.Errorf("base_node_repo is required")
	}
	if p.DataDir == "" {
		return fmt.Errorf("data_dir is required")
	}
	if p.StopTimeoutSeconds <= 0 {
		return fmt.Errorf("stop_timeout_seconds must be > 0 (recommend 300)")
	}
	for i, n := range p.Notifications {
		if n.Type == "" {
			return fmt.Errorf("notifications[%d].type is required", i)
		}
		if n.URL == "" {
			return fmt.Errorf("notifications[%d].url is required (or unresolved env var)", i)
		}
	}
	return nil
}
