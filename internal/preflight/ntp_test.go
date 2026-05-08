package preflight

import (
	"context"
	"testing"
	"time"
)

func TestNTPClockSane(t *testing.T) {
	if testing.Short() {
		t.Skip("network test")
	}
	c := &NTPCheck{MaxDrift: 10 * time.Second}
	r, _ := c.Run(context.Background())
	if r.Message == "" {
		t.Errorf("empty message")
	}
}
