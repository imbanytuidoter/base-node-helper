package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestL1ChainID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer srv.Close()
	cl, err := NewL1(srv.URL)
	if err != nil {
		t.Fatalf("NewL1: %v", err)
	}
	id, err := cl.ChainID(context.Background())
	if err != nil {
		t.Fatalf("ChainID: %v", err)
	}
	if id != 1 {
		t.Errorf("id=%d", id)
	}
}

func TestL1Syncing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":false}`))
	}))
	defer srv.Close()
	cl, _ := NewL1(srv.URL)
	syncing, err := cl.Syncing(context.Background())
	if err != nil {
		t.Fatalf("Syncing: %v", err)
	}
	if syncing {
		t.Errorf("expected not syncing")
	}
}

func TestL1PeerCount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x5"}`))
	}))
	defer srv.Close()
	cl, _ := NewL1(srv.URL)
	count, err := cl.PeerCount(context.Background())
	if err != nil {
		t.Fatalf("PeerCount: %v", err)
	}
	if count != 5 {
		t.Errorf("count=%d, want 5", count)
	}
}

func TestNewL1RejectsInvalidScheme(t *testing.T) {
	_, err := NewL1("file:///etc/passwd")
	if err == nil {
		t.Fatal("expected error for file:// scheme")
	}
}

func TestNewL1RejectsEmpty(t *testing.T) {
	_, err := NewL1("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}
