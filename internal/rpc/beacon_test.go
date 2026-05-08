package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewBeaconRejectsInvalidScheme(t *testing.T) {
	_, err := NewBeacon("file:///etc/passwd")
	if err == nil {
		t.Fatal("expected error for file:// scheme")
	}
}

func TestNewBeaconRejectsEmpty(t *testing.T) {
	_, err := NewBeacon("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestBeaconGenesis(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"genesis_time":"1606824023"}}`))
	}))
	defer srv.Close()
	b, _ := NewBeacon(srv.URL)
	gt, err := b.Genesis(context.Background())
	if err != nil {
		t.Fatalf("Genesis: %v", err)
	}
	if gt != "1606824023" {
		t.Errorf("got %q", gt)
	}
}
