package preflight

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBeaconCheckPass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"genesis_time":"1606824023"}}`))
	}))
	defer srv.Close()

	chk := &BeaconCheck{URL: srv.URL}
	res, err := chk.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != Pass {
		t.Errorf("status=%v, want Pass; msg=%s", res.Status, res.Message)
	}
}

func TestBeaconCheckFailInvalidURL(t *testing.T) {
	chk := &BeaconCheck{URL: "file:///etc/passwd"}
	res, err := chk.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != Fail {
		t.Errorf("status=%v, want Fail", res.Status)
	}
}

func TestBeaconCheckFailBadResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	chk := &BeaconCheck{URL: srv.URL}
	res, err := chk.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != Fail {
		t.Errorf("status=%v, want Fail; msg=%s", res.Status, res.Message)
	}
}

func TestBeaconCheckName(t *testing.T) {
	chk := &BeaconCheck{URL: "http://localhost"}
	if chk.Name() == "" {
		t.Error("Name() should not be empty")
	}
}
