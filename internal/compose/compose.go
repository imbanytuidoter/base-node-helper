package compose

import (
	"context"
	"io"
)

// Compose is the abstract interface for Compose ops.
type Compose interface {
	Up(ctx context.Context, opts UpOpts) error
	Stop(ctx context.Context, opts StopOpts) error
	Down(ctx context.Context, opts DownOpts) error
	PS(ctx context.Context, projectDir string) ([]Container, error)
}

type UpOpts struct {
	ProjectDir string
	Detach     bool
	Stdout     io.Writer
	Stderr     io.Writer
}

type StopOpts struct {
	ProjectDir     string
	TimeoutSeconds int
	Stdout         io.Writer
	Stderr         io.Writer
}

type DownOpts struct {
	ProjectDir    string
	RemoveVolumes bool
	Stdout        io.Writer
	Stderr        io.Writer
}

type Container struct {
	Service  string
	State    string
	ExitCode *int
	Status   string
}
