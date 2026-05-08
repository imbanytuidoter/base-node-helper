package compose

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type recordRunner struct {
	calls [][]string
	out   []byte
	err   error
}

func (r *recordRunner) runCtx(ctx context.Context, prog string, args []string, stdout, stderr io.Writer) error {
	r.calls = append(r.calls, append([]string{prog}, args...))
	if stdout != nil {
		stdout.Write(r.out)
	}
	return r.err
}

func TestStopUsesTimeoutFlag(t *testing.T) {
	rr := &recordRunner{}
	c := &execCompose{inv: Invocation{Version: V2, Bin: "docker", SubArgs: []string{"compose"}}, runner: rr}
	err := c.Stop(context.Background(), StopOpts{ProjectDir: "/repo", TimeoutSeconds: 0})
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if len(rr.calls) != 1 {
		t.Fatalf("calls=%d", len(rr.calls))
	}
	got := strings.Join(rr.calls[0], " ")
	if !strings.Contains(got, "stop --timeout 300") {
		t.Errorf("expected --timeout 300, got: %s", got)
	}
}

func TestUpDetached(t *testing.T) {
	rr := &recordRunner{}
	c := &execCompose{inv: Invocation{Version: V2, Bin: "docker", SubArgs: []string{"compose"}}, runner: rr}
	err := c.Up(context.Background(), UpOpts{ProjectDir: "/repo", Detach: true})
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	got := strings.Join(rr.calls[0], " ")
	if !strings.Contains(got, "up -d") {
		t.Errorf("expected up -d, got: %s", got)
	}
}

func TestDownErrorPropagates(t *testing.T) {
	rr := &recordRunner{err: errors.New("boom")}
	c := &execCompose{inv: Invocation{Version: V2, Bin: "docker", SubArgs: []string{"compose"}}, runner: rr}
	err := c.Down(context.Background(), DownOpts{ProjectDir: "/repo"})
	if err == nil {
		t.Fatalf("expected error")
	}
}
