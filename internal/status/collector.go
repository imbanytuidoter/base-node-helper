package status

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/imbanytuidoter/base-node-helper/internal/compose"
)

type Options struct {
	Compose    compose.Compose
	Timeout    time.Duration
	ProjectDir string
}

type Snapshot struct {
	Containers  []compose.Container
	GeneratedAt time.Time
}

func Collect(ctx context.Context, opts Options) (Snapshot, error) {
	if opts.Compose == nil {
		return Snapshot{}, fmt.Errorf("nil compose")
	}
	cctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	cs, err := opts.Compose.PS(cctx, opts.ProjectDir)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Containers: cs, GeneratedAt: time.Now()}, nil
}

func (s Snapshot) Format() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Status as of %s\n", s.GeneratedAt.Format(time.RFC3339)))
	if len(s.Containers) == 0 {
		b.WriteString("  no containers\n")
		return b.String()
	}
	for _, c := range s.Containers {
		b.WriteString(fmt.Sprintf("  %-20s %-10s %s\n", c.Service, c.State, c.Status))
	}
	return b.String()
}
