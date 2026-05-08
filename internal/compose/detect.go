package compose

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

type Version int

const (
	VUnknown Version = iota
	V1
	V2
)

type Invocation struct {
	Version Version
	Bin     string
	SubArgs []string
}

type runner interface {
	run(ctx context.Context, prog string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) run(ctx context.Context, prog string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, prog, args...).CombinedOutput()
}

// Detect returns the preferred Compose invocation (v2 over v1).
func Detect() (Invocation, error) { return detectWith(context.Background(), execRunner{}) }

func detectWith(ctx context.Context, r runner) (Invocation, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := r.run(ctx, "docker", "compose", "version"); err == nil {
		return Invocation{Version: V2, Bin: "docker", SubArgs: []string{"compose"}}, nil
	}
	if _, err := r.run(ctx, "docker-compose", "version"); err == nil {
		return Invocation{Version: V1, Bin: "docker-compose"}, nil
	}
	return Invocation{}, fmt.Errorf("neither 'docker compose' (v2) nor 'docker-compose' (v1) found in PATH")
}
