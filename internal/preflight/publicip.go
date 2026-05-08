package preflight

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type PublicIPCheck struct {
	Providers []string
	Timeout   time.Duration
	// client overrides the HTTP client used for requests (used in tests to
	// accept self-signed TLS certificates from httptest.NewTLSServer).
	client *http.Client
}

func NewPublicIPCheck() *PublicIPCheck {
	return &PublicIPCheck{
		Providers: []string{
			"https://api.ipify.org",
			"https://ifconfig.me/ip",
			"https://icanhazip.com",
		},
		Timeout: 3 * time.Second,
	}
}

func (p *PublicIPCheck) Name() string { return "public IP discovery" }

func (p *PublicIPCheck) Run(ctx context.Context) (Result, error) {
	timeout := p.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	for _, url := range p.Providers {
		cctx, cancel := context.WithTimeout(ctx, timeout)
		ip, err := fetchIP(cctx, url, timeout, p.client)
		cancel()
		if err != nil {
			continue
		}
		if net.ParseIP(ip) == nil {
			continue
		}
		return Result{
			Status:  Pass,
			Message: fmt.Sprintf("public IP: %s — verify reachability with: nc -vz %s 30303 (from another machine)", ip, ip),
		}, nil
	}
	return Result{
		Status:  Warn,
		Message: "could not determine public IP from any provider",
		Fix:     "check internet connectivity, or pass --no-public-ip and verify externally",
	}, nil
}

func fetchIP(ctx context.Context, providerURL string, timeout time.Duration, cl *http.Client) (string, error) {
	u, err := url.Parse(providerURL)
	if err != nil || u.Scheme != "https" {
		return "", fmt.Errorf("provider URL must use https://, got %q", providerURL)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", providerURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "base-node-helper/preflight")
	if cl == nil {
		cl = &http.Client{Timeout: timeout}
	}
	resp, err := cl.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}
