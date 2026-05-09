package config

import "testing"

// baseValidProfile returns a minimal valid profile for security sub-tests.
func baseValidProfile() *Profile {
	return &Profile{
		Network:            NetworkMainnet,
		Client:             ClientReth,
		BaseNodeRepo:       "/home/user/base-node",
		DataDir:            "/var/data/base",
		StopTimeoutSeconds: 300,
	}
}

// --- Notification URL scheme (SSRF) ---

func TestValidateNotificationHTTPSAccepted(t *testing.T) {
	p := baseValidProfile()
	p.Notifications = []Notification{{Type: "webhook", URL: "https://hooks.example.com/abc"}}
	if err := Validate(p); err != nil {
		t.Errorf("https URL should be accepted: %v", err)
	}
}

func TestValidateNotificationHTTPAccepted(t *testing.T) {
	p := baseValidProfile()
	p.Notifications = []Notification{{Type: "webhook", URL: "http://internal.example.com/hook"}}
	if err := Validate(p); err != nil {
		t.Errorf("http URL should be accepted: %v", err)
	}
}

func TestValidateNotificationFileSchemeRejected(t *testing.T) {
	p := baseValidProfile()
	p.Notifications = []Notification{{Type: "webhook", URL: "file:///etc/passwd"}}
	if err := Validate(p); err == nil {
		t.Error("expected error for file:// notification URL (SSRF)")
	}
}

// TestValidateNotificationIMDSURLAccepted documents that http:// private-IP
// URLs (including AWS IMDS) are ACCEPTED by scheme validation. Blocking
// RFC-1918 addresses would prevent legitimate private RPC endpoints, so SSRF
// prevention for private IPs is the operator's responsibility.
func TestValidateNotificationIMDSURLAccepted(t *testing.T) {
	p := baseValidProfile()
	p.Notifications = []Notification{{Type: "webhook", URL: "http://169.254.169.254/latest/meta-data/"}}
	if err := Validate(p); err != nil {
		t.Errorf("http:// private-IP URL should be accepted (SSRF prevention is operator responsibility): %v", err)
	}
}

func TestValidateNotificationFTPSchemeRejected(t *testing.T) {
	p := baseValidProfile()
	p.Notifications = []Notification{{Type: "webhook", URL: "ftp://example.com/payload"}}
	if err := Validate(p); err == nil {
		t.Error("expected error for ftp:// notification URL")
	}
}

// --- BaseNodeRepo path validation (injection) ---

func TestValidateBaseNodeRepoRelativeRejected(t *testing.T) {
	p := baseValidProfile()
	p.BaseNodeRepo = "relative/path/to/repo"
	if err := Validate(p); err == nil {
		t.Error("expected error for relative base_node_repo path")
	}
}

func TestValidateBaseNodeRepoEmptyRejected(t *testing.T) {
	p := baseValidProfile()
	p.BaseNodeRepo = ""
	if err := Validate(p); err == nil {
		t.Error("expected error for empty base_node_repo")
	}
}

// --- DataDir path validation (Finding 3 + 5 from audit) ---

func TestValidateDataDirRelativeRejected(t *testing.T) {
	p := baseValidProfile()
	p.DataDir = "relative/data/path"
	if err := Validate(p); err == nil {
		t.Error("expected error for relative data_dir path")
	}
}

func TestValidateDataDirEmptyRejected(t *testing.T) {
	p := baseValidProfile()
	p.DataDir = ""
	if err := Validate(p); err == nil {
		t.Error("expected error for empty data_dir")
	}
}

func TestValidateDataDirAbsoluteAccepted(t *testing.T) {
	p := baseValidProfile()
	p.DataDir = "/var/data/base"
	if err := Validate(p); err != nil {
		t.Errorf("absolute data_dir should be accepted: %v", err)
	}
}

// --- StopTimeoutSeconds upper bound (integer overflow) ---

func TestValidateStopTimeoutTooLargeRejected(t *testing.T) {
	p := baseValidProfile()
	p.StopTimeoutSeconds = MaxStopTimeoutSeconds + 1
	if err := Validate(p); err == nil {
		t.Errorf("expected error for stop_timeout_seconds > %d", MaxStopTimeoutSeconds)
	}
}

func TestValidateStopTimeoutMaxAccepted(t *testing.T) {
	p := baseValidProfile()
	p.StopTimeoutSeconds = MaxStopTimeoutSeconds
	if err := Validate(p); err != nil {
		t.Errorf("stop_timeout_seconds=%d should be accepted: %v", MaxStopTimeoutSeconds, err)
	}
}

// --- Profile name allow-list (path traversal) ---

func TestValidProfileNames(t *testing.T) {
	for _, name := range []string{"default", "mainnet-prod", "node_1", "test123", "A", "z-9_X"} {
		if !validProfileName.MatchString(name) {
			t.Errorf("valid profile name rejected: %q", name)
		}
	}
}

func TestInvalidProfileNames(t *testing.T) {
	for _, name := range []string{
		"../../etc/passwd",
		"../evil",
		"foo/bar",
		"foo\\bar",
		"",
		"name with spaces",
		"name\x00null",
		"toolongname_toolongname_toolongname_toolongname_toolongname_toolooong",
	} {
		if validProfileName.MatchString(name) {
			t.Errorf("invalid profile name accepted: %q", name)
		}
	}
}
