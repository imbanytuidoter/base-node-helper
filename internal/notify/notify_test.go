package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imbanytuidoter/base-node-helper/internal/config"
)

func TestSend_Discord(t *testing.T) {
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	nn := []config.Notification{{Type: "discord", URL: srv.URL, Severity: "warn"}}
	if err := Send(context.Background(), nn, "warn", "Title", "body text"); err != nil {
		t.Fatal(err)
	}
	if got["content"] == "" {
		t.Error("expected discord content field to be populated")
	}
}

func TestSend_Webhook(t *testing.T) {
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	nn := []config.Notification{{Type: "webhook", URL: srv.URL}}
	if err := Send(context.Background(), nn, "crit", "Title", "body text"); err != nil {
		t.Fatal(err)
	}
	if got["title"] != "Title" {
		t.Errorf("expected title field, got %v", got)
	}
	if got["body"] != "body text" {
		t.Errorf("expected body field, got %v", got)
	}
}

func TestSend_SeverityFilter(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	nn := []config.Notification{{Type: "webhook", URL: srv.URL, Severity: "crit"}}
	if err := Send(context.Background(), nn, "warn", "T", "b"); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Error("webhook should not have been called for mismatched severity")
	}
}

func TestSend_EmptySeverityMatchesAll(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	nn := []config.Notification{{Type: "webhook", URL: srv.URL, Severity: ""}}
	if err := Send(context.Background(), nn, "warn", "T", "b"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("webhook should have been called when Severity is empty")
	}
}

func TestSend_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	nn := []config.Notification{{Type: "webhook", URL: srv.URL}}
	err := Send(context.Background(), nn, "warn", "T", "b")
	if err == nil {
		t.Error("expected error on HTTP 500")
	}
}
