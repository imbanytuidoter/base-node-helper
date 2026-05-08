package preflight

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicIPFirstProviderSucceeds(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("203.0.113.5"))
	}))
	defer srv.Close()
	c := &PublicIPCheck{Providers: []string{srv.URL}, client: srv.Client()}
	r, _ := c.Run(context.Background())
	if r.Status != Pass {
		t.Fatalf("status=%v msg=%q", r.Status, r.Message)
	}
	if !strings.Contains(r.Message, "203.0.113.5") {
		t.Errorf("message missing IP: %q", r.Message)
	}
}

func TestPublicIPFallsBackOnFailure(t *testing.T) {
	failSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", 500)
	}))
	defer failSrv.Close()
	okSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("198.51.100.7"))
	}))
	defer okSrv.Close()
	c := &PublicIPCheck{Providers: []string{failSrv.URL, okSrv.URL}, client: okSrv.Client()}
	r, _ := c.Run(context.Background())
	if r.Status != Pass {
		t.Fatalf("status=%v msg=%q", r.Status, r.Message)
	}
	if !strings.Contains(r.Message, "198.51.100.7") {
		t.Errorf("did not fall back: %q", r.Message)
	}
}

func TestPublicIPAllFailReturnsWarn(t *testing.T) {
	c := &PublicIPCheck{Providers: []string{"https://127.0.0.1:1"}}
	r, _ := c.Run(context.Background())
	if r.Status != Warn {
		t.Errorf("status=%v", r.Status)
	}
}
