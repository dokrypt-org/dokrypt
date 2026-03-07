package container

import (
	"fmt"
	"os"
	"runtime"

	"github.com/docker/docker/client"
)

type PodmanRuntime struct {
	*DockerRuntime
}

func NewPodmanRuntime() (*PodmanRuntime, error) {
	socketPath := podmanSocketPath()

	cli, err := client.NewClientWithOpts(
		client.WithHost(socketPath),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Podman client: %w", err)
	}

	return &PodmanRuntime{
		DockerRuntime: &DockerRuntime{client: cli},
	}, nil
}

func podmanSocketPath() string {
	if runtime.GOOS == "linux" {
		uid := os.Getuid()
		if uid != 0 {
			return fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", uid)
		}
		return "unix:///run/podman/podman.sock"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "unix:///var/run/podman/podman.sock"
	}
	return fmt.Sprintf("unix://%s/.local/share/containers/podman/machine/podman.sock", home)
}
