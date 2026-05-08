package status

import (
	"context"
	"testing"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/compose"
)

type mockCompose struct {
	containers []compose.Container
	err        error
}

func (m *mockCompose) Up(_ context.Context, _ compose.UpOpts) error   { return nil }
func (m *mockCompose) Stop(_ context.Context, _ compose.StopOpts) error { return nil }
func (m *mockCompose) Down(_ context.Context, _ compose.DownOpts) error { return nil }
func (m *mockCompose) PS(_ context.Context, _ string) ([]compose.Container, error) {
	return m.containers, m.err
}

func TestCollectWithMock(t *testing.T) {
	containers := []compose.Container{
		{Service: "op-node", State: "running", Status: "Up 2 hours"},
		{Service: "op-geth", State: "running", Status: "Up 2 hours"},
	}
	snap, err := Collect(context.Background(), Options{
		Compose: &mockCompose{containers: containers},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(snap.Containers) != 2 {
		t.Errorf("got %d containers, want 2", len(snap.Containers))
	}
	formatted := snap.Format()
	if formatted == "" {
		t.Error("Format returned empty string")
	}
	// Should contain service names
	for _, name := range []string{"op-node", "op-geth"} {
		if !contains(formatted, name) {
			t.Errorf("Format missing %q", name)
		}
	}
}

func TestFormatWithContainers(t *testing.T) {
	exitCode := 137
	snap := Snapshot{
		Containers: []compose.Container{
			{Service: "reth", State: "exited", Status: "Exited (137) 1 min ago", ExitCode: &exitCode},
		},
		GeneratedAt: time.Now(),
	}
	out := snap.Format()
	if !contains(out, "reth") {
		t.Error("missing reth in format output")
	}
	if !contains(out, "exited") {
		t.Error("missing exited state in format output")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
