package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Beacon struct {
	url  string
	http *http.Client
}

func NewBeacon(rawURL string) (*Beacon, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("empty Beacon URL")
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("rpc URL must use http:// or https://, got %q", rawURL)
	}
	return &Beacon{url: strings.TrimRight(rawURL, "/"), http: &http.Client{Timeout: 10 * time.Second}}, nil
}

type genesisResp struct {
	Data struct {
		GenesisTime string `json:"genesis_time"`
	} `json:"data"`
}

func (b *Beacon) Genesis(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", b.url+"/eth/v1/beacon/genesis", nil)
	if err != nil {
		return "", fmt.Errorf("build beacon request: %w", err)
	}
	resp, err := b.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("beacon genesis http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read beacon response: %w", err)
	}
	var g genesisResp
	if err := json.Unmarshal(body, &g); err != nil {
		return "", err
	}
	return g.Data.GenesisTime, nil
}
