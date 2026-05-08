package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/config"
)

// Send posts a notification to all entries in nn whose Severity matches
// (or is empty, meaning "receive all"). Errors from individual endpoints
// are collected and returned as a single joined error.
func Send(ctx context.Context, nn []config.Notification, severity, title, body string) error {
	cl := &http.Client{Timeout: 10 * time.Second}
	var errs []string
	for _, n := range nn {
		if n.Severity != "" && n.Severity != severity {
			continue
		}
		if err := post(ctx, cl, n, title, body); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", n.Type, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("notify: %s", strings.Join(errs, "; "))
	}
	return nil
}

func post(ctx context.Context, cl *http.Client, n config.Notification, title, body string) error {
	var payload interface{}
	switch n.Type {
	case "discord":
		payload = map[string]string{"content": title + "\n" + body}
	default:
		payload = map[string]string{"title": title, "body": body}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", n.URL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return nil
}
