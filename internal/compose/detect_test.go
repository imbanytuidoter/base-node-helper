package compose

import (
	"context"
	"errors"
	"testing"
)

type fakeRunner struct {
	v2OK bool
	v1OK bool
}

func (f fakeRunner) run(_ context.Context, prog string, args ...string) ([]byte, error) {
	switch prog {
	case "docker":
		if f.v2OK && len(args) > 0 && args[0] == "compose" {
			return []byte("Docker Compose version v2.29.7\n"), nil
		}
	case "docker-compose":
		if f.v1OK {
			return []byte("docker-compose version 1.29.2, build 5becea4c\n"), nil
		}
	}
	return nil, errors.New("not found")
}

func TestDetectV2Preferred(t *testing.T) {
	got, err := detectWith(context.Background(), fakeRunner{v2OK: true, v1OK: true})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if got.Version != V2 {
		t.Errorf("got %v, want V2", got.Version)
	}
	if got.Bin != "docker" || len(got.SubArgs) != 1 || got.SubArgs[0] != "compose" {
		t.Errorf("invocation = %+v", got)
	}
}

func TestDetectV1Fallback(t *testing.T) {
	got, err := detectWith(context.Background(), fakeRunner{v1OK: true})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if got.Version != V1 {
		t.Errorf("got %v", got.Version)
	}
}

func TestDetectNoneInstalled(t *testing.T) {
	_, err := detectWith(context.Background(), fakeRunner{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
