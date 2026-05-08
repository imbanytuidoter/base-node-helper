package config

import (
	"testing"

	"github.com/spf13/afero"
)

func TestLoadProfileMinimalValid(t *testing.T) {
	fs := afero.NewMemMapFs()
	yaml := `
network: mainnet
client: reth
base_node_repo: /home/user/base-node
data_dir: /var/data/base
stop_timeout_seconds: 300
preflight:
  public_ip_check: true
  disk_speed_check: true
`
	afero.WriteFile(fs, "/home/user/.base-node-helper/profiles/default/config.yaml", []byte(yaml), 0644)

	c, err := LoadProfile(fs, "/home/user/.base-node-helper", "default")
	if err != nil {
		t.Fatalf("LoadProfile error: %v", err)
	}
	if c.Network != "mainnet" {
		t.Errorf("Network = %q", c.Network)
	}
	if c.Client != "reth" {
		t.Errorf("Client = %q", c.Client)
	}
	if c.StopTimeoutSeconds != 300 {
		t.Errorf("StopTimeoutSeconds = %d", c.StopTimeoutSeconds)
	}
}

func TestLoadProfileMissingRequired(t *testing.T) {
	fs := afero.NewMemMapFs()
	yaml := `network: mainnet`
	afero.WriteFile(fs, "/home/user/.base-node-helper/profiles/default/config.yaml", []byte(yaml), 0644)

	_, err := LoadProfile(fs, "/home/user/.base-node-helper", "default")
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
}

func TestEnvInterpolation(t *testing.T) {
	t.Setenv("DISCORD_HOOK", "https://discord.com/api/webhooks/123/abc")
	fs := afero.NewMemMapFs()
	yaml := `
network: mainnet
client: reth
base_node_repo: /repo
data_dir: /data
stop_timeout_seconds: 300
notifications:
  - type: webhook.discord
    url: ${DISCORD_HOOK}
    severity: ">=warning"
`
	afero.WriteFile(fs, "/h/.base-node-helper/profiles/default/config.yaml", []byte(yaml), 0644)
	c, err := LoadProfile(fs, "/h/.base-node-helper", "default")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if len(c.Notifications) != 1 {
		t.Fatalf("notifications len=%d", len(c.Notifications))
	}
	if c.Notifications[0].URL != "https://discord.com/api/webhooks/123/abc" {
		t.Errorf("env not interpolated: %q", c.Notifications[0].URL)
	}
}

func TestLoadProfilePathTraversalRejected(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, err := LoadProfile(fs, "/home/user/.base-node-helper", "../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal profile name")
	}
}

func TestSaveProfilePathTraversalRejected(t *testing.T) {
	fs := afero.NewMemMapFs()
	p := &Profile{
		Network:            "mainnet",
		Client:             "reth",
		BaseNodeRepo:       "/repo",
		DataDir:            "/data",
		StopTimeoutSeconds: 300,
	}
	err := SaveProfile(fs, "/home/user/.base-node-helper", "../../etc/cron.d/evil", p)
	if err == nil {
		t.Fatal("expected error for path traversal profile name")
	}
}

func TestLoadProfileMissingEnvVar(t *testing.T) {
	fs := afero.NewMemMapFs()
	yaml := `
network: mainnet
client: reth
base_node_repo: /repo
data_dir: /data
stop_timeout_seconds: 300
notifications:
  - type: webhook.discord
    url: ${DOES_NOT_EXIST_XYZ_123}
    severity: ">=warning"
`
	afero.WriteFile(fs, "/h/.bnh/profiles/default/config.yaml", []byte(yaml), 0644)
	_, err := LoadProfile(fs, "/h/.bnh", "default")
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
}

func TestValidateInvalidClient(t *testing.T) {
	p := &Profile{Network: "mainnet", Client: "parity", BaseNodeRepo: "/r", DataDir: "/d", StopTimeoutSeconds: 300}
	if err := Validate(p); err == nil {
		t.Fatal("expected error for unknown client")
	}
}
