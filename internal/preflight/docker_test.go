package preflight

import (
	"context"
	"errors"
	"testing"
)

type fakeDockerProbe struct {
	pingErr  error
	inGroup  bool
	groupErr error
}

func (f fakeDockerProbe) Ping(ctx context.Context) error            { return f.pingErr }
func (f fakeDockerProbe) UserInDockerGroup() (bool, error) {
	return f.inGroup, f.groupErr
}

func TestDockerCheckPing(t *testing.T) {
	c := &DockerCheck{probe: fakeDockerProbe{pingErr: errors.New("connection refused")}}
	r, _ := c.Run(context.Background())
	if r.Status != Fail {
		t.Errorf("status=%v", r.Status)
	}
}

func TestDockerCheckUserNotInGroup(t *testing.T) {
	c := &DockerCheck{probe: fakeDockerProbe{inGroup: false}}
	r, _ := c.Run(context.Background())
	if r.Status != Warn {
		t.Errorf("status=%v", r.Status)
	}
}

func TestDockerCheckHappy(t *testing.T) {
	c := &DockerCheck{probe: fakeDockerProbe{inGroup: true}}
	r, _ := c.Run(context.Background())
	if r.Status != Pass {
		t.Errorf("status=%v", r.Status)
	}
}
