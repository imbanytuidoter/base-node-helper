package preflight

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type NTPCheck struct {
	Servers  []string
	MaxDrift time.Duration
}

func NewNTPCheck() *NTPCheck {
	return &NTPCheck{
		Servers:  []string{"time.cloudflare.com:123", "pool.ntp.org:123", "time.google.com:123"},
		MaxDrift: 5 * time.Second,
	}
}

func (n *NTPCheck) Name() string { return "system clock vs NTP" }

func (n *NTPCheck) Run(ctx context.Context) (Result, error) {
	servers := n.Servers
	if len(servers) == 0 {
		servers = NewNTPCheck().Servers
	}
	var badDrifts []string
	for _, s := range servers {
		drift, err := queryNTP(ctx, s, 2*time.Second)
		if err != nil {
			continue
		}
		abs := drift
		if abs < 0 {
			abs = -abs
		}
		if abs <= n.MaxDrift {
			return Result{Status: Pass, Message: fmt.Sprintf("clock vs %s drift=%v", s, drift)}, nil
		}
		badDrifts = append(badDrifts, fmt.Sprintf("%s drift=%v", s, drift))
	}
	if len(badDrifts) > 0 {
		return Result{
			Status:  Fail,
			Message: fmt.Sprintf("clock drift exceeds %v: %v", n.MaxDrift, badDrifts),
			Fix:     "enable NTP (timedatectl set-ntp true on Linux; macOS/Windows: enable auto time)",
		}, nil
	}
	return Result{Status: Warn, Message: "no NTP servers reachable"}, nil
}

func queryNTP(ctx context.Context, addr string, timeout time.Duration) (time.Duration, error) {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "udp", addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	req := make([]byte, 48)
	req[0] = 0x1B // LI=0, VN=3, Mode=3 (client)

	t1 := time.Now()
	if _, err := conn.Write(req); err != nil {
		return 0, err
	}
	resp := make([]byte, 48)
	if _, err := conn.Read(resp); err != nil {
		return 0, err
	}
	t4 := time.Now()

	const ntpEpochOffset = 2208988800
	secs := binary.BigEndian.Uint32(resp[40:44])
	frac := binary.BigEndian.Uint32(resp[44:48])
	if secs == 0 {
		return 0, fmt.Errorf("invalid NTP response")
	}
	serverTime := time.Unix(int64(secs)-ntpEpochOffset, int64(float64(frac)*1e9/(1<<32)))
	rtt := t4.Sub(t1)
	estServer := serverTime.Add(rtt / 2)
	drift := t4.Sub(estServer)
	return drift, nil
}
