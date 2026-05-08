package config

import (
	"testing"

	"github.com/spf13/afero"
)

func TestSaveAndLoadProfile(t *testing.T) {
	fs := afero.NewMemMapFs()
	prof := &Profile{
		Network:            NetworkSepolia,
		Client:             ClientReth,
		BaseNodeRepo:       "/home/user/base-node",
		DataDir:            "/home/user/data",
		StopTimeoutSeconds: 300,
		Preflight:          PreflightOpts{PublicIPCheck: true, DiskSpeedCheck: false},
	}
	if err := SaveProfile(fs, "/tmp/bnh", "myprofile", prof); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	loaded, err := LoadProfile(fs, "/tmp/bnh", "myprofile")
	if err != nil {
		t.Fatalf("LoadProfile after save: %v", err)
	}
	if loaded.Network != NetworkSepolia {
		t.Errorf("network=%q", loaded.Network)
	}
	if loaded.Client != ClientReth {
		t.Errorf("client=%q", loaded.Client)
	}
	if loaded.StopTimeoutSeconds != 300 {
		t.Errorf("stop_timeout_seconds=%d", loaded.StopTimeoutSeconds)
	}
	if !loaded.Preflight.PublicIPCheck {
		t.Error("public_ip_check should be true")
	}
}

func TestDefaultBaseDir(t *testing.T) {
	d, err := DefaultBaseDir()
	if err != nil {
		t.Fatalf("DefaultBaseDir: %v", err)
	}
	if d == "" {
		t.Error("DefaultBaseDir returned empty string")
	}
}

func TestValidateAllNetworks(t *testing.T) {
	base := &Profile{
		Client:             ClientReth,
		BaseNodeRepo:       "/repo",
		DataDir:            "/data",
		StopTimeoutSeconds: 300,
	}
	for _, net := range []Network{NetworkMainnet, NetworkSepolia, NetworkDevnet} {
		p := *base
		p.Network = net
		if err := Validate(&p); err != nil {
			t.Errorf("Validate(%q): %v", net, err)
		}
	}
}

func TestValidateInvalidNetwork(t *testing.T) {
	p := &Profile{
		Network:            "badnet",
		Client:             ClientReth,
		BaseNodeRepo:       "/repo",
		DataDir:            "/data",
		StopTimeoutSeconds: 300,
	}
	if err := Validate(p); err == nil {
		t.Error("expected error for invalid network")
	}
}

func TestValidateZeroTimeout(t *testing.T) {
	p := &Profile{
		Network:            NetworkSepolia,
		Client:             ClientGeth,
		BaseNodeRepo:       "/repo",
		DataDir:            "/data",
		StopTimeoutSeconds: 0,
	}
	if err := Validate(p); err == nil {
		t.Error("expected error for zero stop_timeout_seconds")
	}
}
