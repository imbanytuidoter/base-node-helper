package preflight

import (
	"context"
	"fmt"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
)

type DockerProbe interface {
	Ping(ctx context.Context) error
	UserInDockerGroup() (bool, error)
}

type defaultDockerProbe struct{}

func (defaultDockerProbe) Ping(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker info failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (defaultDockerProbe) UserInDockerGroup() (bool, error) {
	if runtime.GOOS != "linux" {
		return true, nil
	}
	u, err := user.Current()
	if err != nil {
		return false, err
	}
	gids, err := u.GroupIds()
	if err != nil {
		return false, err
	}
	for _, gid := range gids {
		g, err := user.LookupGroupId(gid)
		if err != nil {
			continue
		}
		if g.Name == "docker" {
			return true, nil
		}
	}
	return false, nil
}

type DockerCheck struct {
	probe DockerProbe
}

func NewDockerCheck() *DockerCheck { return &DockerCheck{probe: defaultDockerProbe{}} }

func (d *DockerCheck) Name() string { return "docker daemon and group" }

func (d *DockerCheck) Run(ctx context.Context) (Result, error) {
	if err := d.probe.Ping(ctx); err != nil {
		return Result{
			Status:  Fail,
			Message: "Docker daemon not reachable",
			Fix:     "Start Docker (systemctl start docker on Linux; open Docker Desktop on macOS/Windows)",
		}, nil
	}
	in, err := d.probe.UserInDockerGroup()
	if err != nil {
		return Result{Status: Warn, Message: "could not determine docker group membership"}, nil
	}
	if !in {
		return Result{
			Status:  Warn,
			Message: "current user is not in the 'docker' group; you will need sudo for docker commands",
			Fix:     "sudo usermod -aG docker $USER  &&  log out and back in",
		}, nil
	}
	return Result{Status: Pass, Message: "Docker daemon reachable, user in docker group"}, nil
}
