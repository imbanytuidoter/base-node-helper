package preflight

import (
	"context"
	"runtime"
	"testing"
)

func TestFirewallReturnsAdvisoryOnAnyOS(t *testing.T) {
	c := NewFirewallCheck()
	r, _ := c.Run(context.Background())
	if r.Status == Fail {
		t.Errorf("firewall check should never Fail (advisory): got Fail / %q", r.Message)
	}
	if r.Message == "" {
		t.Errorf("empty message on %s", runtime.GOOS)
	}
}
