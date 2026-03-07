package container

import (
	"context"
	"io"
	"time"
)

type Runtime interface {
	CreateContainer(ctx context.Context, cfg *ContainerConfig) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, timeout time.Duration) error
	RemoveContainer(ctx context.Context, id string, force bool) error
	ListContainers(ctx context.Context, opts ListOptions) ([]ContainerInfo, error)
	InspectContainer(ctx context.Context, id string) (*ContainerInfo, error)
	WaitContainer(ctx context.Context, id string) (int64, error)

	PullImage(ctx context.Context, image string) error
	BuildImage(ctx context.Context, contextPath string, opts BuildOptions) (string, error)
	ListImages(ctx context.Context) ([]ImageInfo, error)
	RemoveImage(ctx context.Context, image string, force bool) error

	ContainerLogs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error)
	ExecInContainer(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error)

	CreateNetwork(ctx context.Context, name string, opts NetworkOptions) (string, error)
	RemoveNetwork(ctx context.Context, id string) error
	ConnectNetwork(ctx context.Context, networkID, containerID string) error
	DisconnectNetwork(ctx context.Context, networkID, containerID string) error
	ListNetworks(ctx context.Context) ([]NetworkInfo, error)

	CreateVolume(ctx context.Context, name string, opts VolumeOptions) (string, error)
	RemoveVolume(ctx context.Context, name string, force bool) error
	ListVolumes(ctx context.Context) ([]VolumeInfo, error)
	InspectVolume(ctx context.Context, name string) (*VolumeInfo, error)

	Ping(ctx context.Context) error
	Info(ctx context.Context) (*RuntimeInfo, error)
}

type ContainerConfig struct {
	Name         string
	Image        string
	Command      []string
	Entrypoint   []string
	Env          map[string]string
	Ports        map[int]int // container port -> host port (0 = auto-assign)
	Volumes      []VolumeMount
	Networks     []string
	Labels       map[string]string
	WorkingDir   string
	User         string
	RestartPolicy string // "", "always", "unless-stopped", "on-failure"
	MemoryLimit  int64  // bytes, 0 = no limit
	CPULimit     float64 // cores, 0 = no limit
	ReadOnly     bool
	CapDrop      []string
	Hostname     string
	NetworkAliases map[string][]string // network -> aliases
}

type ContainerInfo struct {
	ID        string
	Name      string
	Image     string
	Status    string // "running", "exited", "created", etc.
	State     string
	Ports     map[int]int // container port -> host port
	Labels    map[string]string
	Networks  []string
	CreatedAt time.Time
	StartedAt time.Time
	IPAddress string
}

type ListOptions struct {
	All     bool              // Include stopped containers
	Labels  map[string]string // Filter by labels
	Limit   int
}

type BuildOptions struct {
	Tags       []string
	Dockerfile string
	BuildArgs  map[string]string
	NoCache    bool
}

type ImageInfo struct {
	ID      string
	Tags    []string
	Size    int64
	Created time.Time
}

type LogOptions struct {
	Follow     bool
	Tail       string // "all", or a number
	Since      string
	Timestamps bool
	Stdout     bool
	Stderr     bool
}

type ExecOptions struct {
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Interactive bool
	TTY         bool
	Env         []string
	WorkingDir  string
}

type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type VolumeMount struct {
	Source   string // Host path or volume name
	Target  string // Container path
	ReadOnly bool
}

type NetworkOptions struct {
	Driver     string // "bridge", "host", "none"
	Internal   bool   // No external access
	Labels     map[string]string
	Subnet     string
	Gateway    string
}

type NetworkInfo struct {
	ID     string
	Name   string
	Driver string
	Subnet string
	Labels map[string]string
}

type VolumeOptions struct {
	Driver string
	Labels map[string]string
}

type VolumeInfo struct {
	Name       string
	Driver     string
	Mountpoint string
	Labels     map[string]string
	CreatedAt  time.Time
}

type RuntimeInfo struct {
	Name        string // "docker" or "podman"
	Version     string
	APIVersion  string
	OS          string
	Arch        string
}

func NewRuntime(name string) (Runtime, error) {
	switch name {
	case "docker", "":
		return NewDockerRuntime()
	case "podman":
		return NewPodmanRuntime()
	default:
		return nil, &UnsupportedRuntimeError{Name: name}
	}
}

type UnsupportedRuntimeError struct {
	Name string
}

func (e *UnsupportedRuntimeError) Error() string {
	return "unsupported container runtime: " + e.Name
}
