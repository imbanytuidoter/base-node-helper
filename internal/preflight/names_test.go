package preflight

import (
	"testing"
)

// TestCheckNames ensures Name() methods are exercised and return non-empty strings.
func TestCheckNames(t *testing.T) {
	dir := t.TempDir()

	checks := []Check{
		NewDockerCheck(),
		NewPortsCheck(),
		NewFirewallCheck(),
		&PermsCheck{Path: dir},
		NewPublicIPCheck(),
		&DiskSpeedCheck{Path: dir},
		&DiskSpaceCheck{Path: dir, RequiredBytes: 1},
		NewNTPCheck(),
		&RPCCheck{URL: "http://localhost:8545", ExpectedChainID: 1},
		&BeaconCheck{URL: "http://localhost:5052"},
	}

	for _, c := range checks {
		name := c.Name()
		if name == "" {
			t.Errorf("check %T returned empty Name()", c)
		}
	}
}

func TestStatusString(t *testing.T) {
	cases := []struct {
		s    Status
		want string
	}{
		{Pass, "PASS"},
		{Warn, "WARN"},
		{Fail, "FAIL"},
		{Status(99), "?"},
	}
	for _, c := range cases {
		got := c.s.String()
		if got != c.want {
			t.Errorf("Status(%d).String() = %q, want %q", c.s, got, c.want)
		}
	}
}

func TestReportWorstOnEmpty(t *testing.T) {
	r := Report{}
	if r.Worst() != Pass {
		t.Errorf("empty report worst = %v, want Pass", r.Worst())
	}
}
