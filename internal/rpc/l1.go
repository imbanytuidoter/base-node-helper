package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type L1 struct {
	url  string
	http *http.Client
}

func NewL1(rawURL string) (*L1, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("empty L1 URL")
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("rpc URL must use http:// or https://, got %q", rawURL)
	}
	return &L1{url: rawURL, http: &http.Client{Timeout: 10 * time.Second}}, nil
}

type rpcReq struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResp struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *L1) call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	if params == nil {
		params = []interface{}{}
	}
	body, err := json.Marshal(rpcReq{Jsonrpc: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return nil, fmt.Errorf("marshal rpc request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var r rpcResp
	if err := json.Unmarshal(raw, &r); err != nil {
		// [CRIT] info-leak: do NOT include raw body in error — it may contain
		// API keys echoed from auth error responses or URLs with secrets.
		return nil, fmt.Errorf("decode rpc response: %w", err)
	}
	if r.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", r.Error.Code, r.Error.Message)
	}
	return r.Result, nil
}

func (c *L1) ChainID(ctx context.Context) (uint64, error) {
	res, err := c.call(ctx, "eth_chainId")
	if err != nil {
		return 0, err
	}
	var hex string
	if err := json.Unmarshal(res, &hex); err != nil {
		return 0, err
	}
	var id uint64
	if _, err := fmt.Sscanf(hex, "0x%x", &id); err != nil {
		return 0, err
	}
	return id, nil
}

func (c *L1) Syncing(ctx context.Context) (bool, error) {
	res, err := c.call(ctx, "eth_syncing")
	if err != nil {
		return false, err
	}
	var b bool
	if err := json.Unmarshal(res, &b); err == nil {
		return b, nil
	}
	// when syncing, returns object → true
	return true, nil
}

// PeerCount returns the number of peers connected to the node via net_peerCount.
func (c *L1) PeerCount(ctx context.Context) (uint64, error) {
	res, err := c.call(ctx, "net_peerCount")
	if err != nil {
		return 0, err
	}
	var hex string
	if err := json.Unmarshal(res, &hex); err != nil {
		return 0, err
	}
	var count uint64
	if _, err := fmt.Sscanf(hex, "0x%x", &count); err != nil {
		return 0, err
	}
	return count, nil
}
