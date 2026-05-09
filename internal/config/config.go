package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type Network string

const (
	NetworkMainnet Network = "mainnet"
	NetworkSepolia Network = "sepolia"
	NetworkDevnet  Network = "devnet"
)

type Client string

const (
	ClientReth Client = "reth"
	ClientGeth Client = "geth"
)

type Profile struct {
	Network            Network        `yaml:"network"`
	Client             Client         `yaml:"client"`
	BaseNodeRepo       string         `yaml:"base_node_repo"`
	DataDir            string         `yaml:"data_dir"`
	StopTimeoutSeconds int            `yaml:"stop_timeout_seconds"`
	Preflight          PreflightOpts  `yaml:"preflight"`
	Monitor            MonitorOpts    `yaml:"monitor"`
	Notifications      []Notification `yaml:"notifications"`
}

type PreflightOpts struct {
	PublicIPCheck  bool `yaml:"public_ip_check"`
	DiskSpeedCheck bool `yaml:"disk_speed_check"`
}

type MonitorOpts struct {
	Enabled                 bool `yaml:"enabled"`
	RethMemThresholdPct     int  `yaml:"reth_mem_threshold_pct"`
	SyncLagThresholdSeconds int  `yaml:"sync_lag_threshold_seconds"`
	PeerCountMin            int  `yaml:"peer_count_min"`
}

type Notification struct {
	Type     string `yaml:"type"`
	URL      string `yaml:"url"`
	Severity string `yaml:"severity"`
}

var envRefRE = regexp.MustCompile(`\$\{([A-Z][A-Z0-9_]*)\}`)

func safeProfilePath(baseDir, name string) (string, error) {
	profilesDir := filepath.Clean(filepath.Join(baseDir, "profiles"))
	path := filepath.Clean(filepath.Join(profilesDir, name, "config.yaml"))
	rel, err := filepath.Rel(profilesDir, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid profile name %q: must not contain path separators or '..'", name)
	}
	return path, nil
}

func interpolate(s string) (string, error) {
	var missing []string
	result := envRefRE.ReplaceAllStringFunc(s, func(m string) string {
		key := m[2 : len(m)-1]
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("env var(s) not set: %s", strings.Join(missing, ", "))
	}
	return result, nil
}

func interpolateAll(p *Profile) error {
	for i := range p.Notifications {
		url, err := interpolate(p.Notifications[i].URL)
		if err != nil {
			return fmt.Errorf("notifications[%d].url: %w", i, err)
		}
		p.Notifications[i].URL = url
	}
	return nil
}

// validProfileName enforces a strict allow-list to prevent path traversal,
// null-byte injection, and Windows reserved-name attacks.
var validProfileName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func LoadProfile(fs afero.Fs, baseDir, name string) (*Profile, error) {
	// [CRIT] path-traversal: use allow-list instead of deny-list so that
	// null bytes, Unicode overrides, and OS-reserved names are all rejected.
	if !validProfileName.MatchString(name) {
		return nil, fmt.Errorf("invalid profile name %q: only [a-zA-Z0-9_-] (1-64 chars) allowed", name)
	}
	path, err := safeProfilePath(baseDir, name)
	if err != nil {
		return nil, err
	}
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, fmt.Errorf("read profile %q: %w", name, err)
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile %q: %w", name, err)
	}
	if err := interpolateAll(&p); err != nil {
		return nil, fmt.Errorf("interpolate profile %q: %w", name, err)
	}
	if err := Validate(&p); err != nil {
		return nil, fmt.Errorf("invalid profile %q: %w", name, err)
	}
	return &p, nil
}

func SaveProfile(fs afero.Fs, baseDir, name string, p *Profile) error {
	if !validProfileName.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: only [a-zA-Z0-9_-] (1-64 chars) allowed", name)
	}
	// [LOW-F11] validate content before writing so errors surface immediately
	// at save time rather than confusingly at start/stop/doctor runtime.
	if err := Validate(p); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}
	path, err := safeProfilePath(baseDir, name)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := fs.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create profile dir %q: %w", name, err)
	}
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal profile %q: %w", name, err)
	}
	if err := afero.WriteFile(fs, path, data, 0o600); err != nil {
		return fmt.Errorf("write profile %q: %w", name, err)
	}
	return nil
}
