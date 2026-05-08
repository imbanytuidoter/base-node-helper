package compose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/imbanytuidoter/base-node-helper/internal/log"
)

type ctxRunner interface {
	runCtx(ctx context.Context, prog string, args []string, stdout, stderr io.Writer) error
}

type osCtxRunner struct{}

func (osCtxRunner) runCtx(ctx context.Context, prog string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, prog, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

type execCompose struct {
	inv    Invocation
	runner ctxRunner
}

// New returns a Compose that shells out using the detected invocation.
func New(inv Invocation) Compose {
	return &execCompose{inv: inv, runner: osCtxRunner{}}
}

func (e *execCompose) base(projectDir string) []string {
	args := append([]string{}, e.inv.SubArgs...)
	args = append(args, "--project-directory", projectDir)
	return args
}

func (e *execCompose) run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	var so, se bytes.Buffer
	var soW, seW io.Writer = &so, &se
	if stdout != nil {
		soW = io.MultiWriter(&so, stdout)
	}
	if stderr != nil {
		seW = io.MultiWriter(&se, stderr)
	}
	err := e.runner.runCtx(ctx, e.inv.Bin, args, soW, seW)
	if err != nil {
		return fmt.Errorf("%s %s: %w (%s)", e.inv.Bin, strings.Join(args, " "), err, log.Redact(strings.TrimSpace(se.String())))
	}
	return nil
}

func (e *execCompose) Up(ctx context.Context, o UpOpts) error {
	args := e.base(o.ProjectDir)
	args = append(args, "up")
	if o.Detach {
		args = append(args, "-d")
	}
	return e.run(ctx, args, o.Stdout, o.Stderr)
}

func (e *execCompose) Stop(ctx context.Context, o StopOpts) error {
	if o.TimeoutSeconds <= 0 {
		o.TimeoutSeconds = 300
	}
	args := e.base(o.ProjectDir)
	args = append(args, "stop", "--timeout", strconv.Itoa(o.TimeoutSeconds))
	return e.run(ctx, args, o.Stdout, o.Stderr)
}

func (e *execCompose) Down(ctx context.Context, o DownOpts) error {
	args := e.base(o.ProjectDir)
	args = append(args, "down")
	if o.RemoveVolumes {
		args = append(args, "-v")
	}
	return e.run(ctx, args, o.Stdout, o.Stderr)
}

// PS queries container status for the compose project at projectDir.
func (e *execCompose) PS(ctx context.Context, projectDir string) ([]Container, error) {
	args := e.base(projectDir)
	args = append(args, "ps", "--format", "json")
	var so, se bytes.Buffer
	if err := e.runner.runCtx(ctx, e.inv.Bin, args, &so, &se); err != nil {
		return nil, fmt.Errorf("ps: %w (%s)", err, log.Redact(strings.TrimSpace(se.String())))
	}
	return parsePS(so.Bytes())
}

func parsePS(output []byte) ([]Container, error) {
	output = bytes.TrimSpace(output)
	if len(output) == 0 {
		return []Container{}, nil
	}
	type rawContainer struct {
		Service  string `json:"Service"`
		State    string `json:"State"`
		Status   string `json:"Status"`
		ExitCode int    `json:"ExitCode"`
	}
	toContainer := func(raw rawContainer) Container {
		c := Container{Service: raw.Service, State: raw.State, Status: raw.Status}
		if raw.State == "exited" {
			ec := raw.ExitCode
			c.ExitCode = &ec
		}
		return c
	}
	// Try JSON array format (Compose v2.17+)
	if output[0] == '[' {
		var items []rawContainer
		if err := json.Unmarshal(output, &items); err == nil {
			out := make([]Container, 0, len(items))
			for _, item := range items {
				out = append(out, toContainer(item))
			}
			return out, nil
		}
	}
	// Fall back to line-by-line JSON objects (Compose v2 < 2.17)
	out := []Container{}
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}
		var raw rawContainer
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, err
		}
		out = append(out, toContainer(raw))
	}
	return out, nil
}
