package preflight

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func rpcDispatch(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")
	s := string(body)
	switch {
	case strings.Contains(s, "eth_chainId"):
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x2105"}`))
	case strings.Contains(s, "eth_syncing"):
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":false}`))
	default:
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":null}`))
	}
}

func TestRPCCheckPassWhenInSync(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(rpcDispatch))
	defer srv.Close()

	chk := &RPCCheck{URL: srv.URL, ExpectedChainID: 0x2105}
	res, err := chk.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != Pass {
		t.Errorf("status=%v, want Pass; msg=%s", res.Status, res.Message)
	}
}

func TestRPCCheckWarnWhenSyncing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		s := string(body)
		if strings.Contains(s, "eth_chainId") {
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
		} else {
			// eth_syncing → object means syncing=true
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"startingBlock":"0x0","currentBlock":"0x100","highestBlock":"0x200"}}`))
		}
	}))
	defer srv.Close()

	chk := &RPCCheck{URL: srv.URL, ExpectedChainID: 0}
	res, err := chk.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != Warn {
		t.Errorf("status=%v, want Warn; msg=%s", res.Status, res.Message)
	}
}

func TestRPCCheckFailOnBadURL(t *testing.T) {
	chk := &RPCCheck{URL: "file:///etc/passwd", ExpectedChainID: 0}
	res, err := chk.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != Fail {
		t.Errorf("status=%v, want Fail", res.Status)
	}
}

func TestRPCCheckFailOnWrongChainID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer srv.Close()

	chk := &RPCCheck{URL: srv.URL, ExpectedChainID: 8453}
	res, err := chk.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != Fail {
		t.Errorf("status=%v, want Fail; msg=%s", res.Status, res.Message)
	}
}
