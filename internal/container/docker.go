package container

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	containerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	imageTypes "github.com/docker/docker/api/types/image"
	networkTypes "github.com/docker/docker/api/types/network"
	volumeTypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

type DockerRuntime struct {
	client *client.Client
}

func NewDockerRuntime() (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &DockerRuntime{client: cli}, nil
}

func (d *DockerRuntime) Ping(ctx context.Context) error {
	_, err := d.client.Ping(ctx)
	return err
}

func (d *DockerRuntime) Info(ctx context.Context) (*RuntimeInfo, error) {
	info, err := d.client.Info(ctx)
	if err != nil {
		return nil, err
	}
	ver, err := d.client.ServerVersion(ctx)
	if err != nil {
		return nil, err
	}
	return &RuntimeInfo{
		Name:       "docker",
		Version:    ver.Version,
		APIVersion: ver.APIVersion,
		OS:         info.OSType,
		Arch:       info.Architecture,
	}, nil
}

func (d *DockerRuntime) CreateContainer(ctx context.Context, cfg *ContainerConfig) (string, error) {
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for containerPort, hostPort := range cfg.Ports {
		cp := nat.Port(fmt.Sprintf("%d/tcp", containerPort))
		exposedPorts[cp] = struct{}{}
		hp := ""
		if hostPort > 0 {
			hp = strconv.Itoa(hostPort)
		}
		portBindings[cp] = []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: hp}}
	}

	var env []string
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}

	var binds []string
	for _, vm := range cfg.Volumes {
		bind := vm.Source + ":" + vm.Target
		if vm.ReadOnly {
			bind += ":ro"
		}
		binds = append(binds, bind)
	}

	containerCfg := &containerTypes.Config{
		Image:        cfg.Image,
		Cmd:          cfg.Command,
		Entrypoint:   cfg.Entrypoint,
		Env:          env,
		ExposedPorts: exposedPorts,
		Labels:       cfg.Labels,
		WorkingDir:   cfg.WorkingDir,
		User:         cfg.User,
		Hostname:     cfg.Hostname,
	}

	hostCfg := &containerTypes.HostConfig{
		PortBindings: portBindings,
		Binds:        binds,
	}

	if cfg.MemoryLimit > 0 {
		hostCfg.Resources.Memory = cfg.MemoryLimit
	}
	if cfg.CPULimit > 0 {
		hostCfg.Resources.NanoCPUs = int64(cfg.CPULimit * 1e9)
	}
	if cfg.ReadOnly {
		hostCfg.ReadonlyRootfs = true
	}
	if len(cfg.CapDrop) > 0 {
		hostCfg.CapDrop = cfg.CapDrop
	}

	switch cfg.RestartPolicy {
	case "always":
		hostCfg.RestartPolicy = containerTypes.RestartPolicy{Name: containerTypes.RestartPolicyAlways}
	case "unless-stopped":
		hostCfg.RestartPolicy = containerTypes.RestartPolicy{Name: containerTypes.RestartPolicyUnlessStopped}
	case "on-failure":
		hostCfg.RestartPolicy = containerTypes.RestartPolicy{Name: containerTypes.RestartPolicyOnFailure}
	}

	networkCfg := &networkTypes.NetworkingConfig{}
	if len(cfg.Networks) > 0 {
		networkCfg.EndpointsConfig = make(map[string]*networkTypes.EndpointSettings)
		for _, net := range cfg.Networks {
			epCfg := &networkTypes.EndpointSettings{}
			if aliases, ok := cfg.NetworkAliases[net]; ok {
				epCfg.Aliases = aliases
			}
			networkCfg.EndpointsConfig[net] = epCfg
		}
	}

	resp, err := d.client.ContainerCreate(ctx, containerCfg, hostCfg, networkCfg, nil, cfg.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create container %s: %w", cfg.Name, err)
	}

	return resp.ID, nil
}

func (d *DockerRuntime) StartContainer(ctx context.Context, id string) error {
	return d.client.ContainerStart(ctx, id, containerTypes.StartOptions{})
}

func (d *DockerRuntime) StopContainer(ctx context.Context, id string, timeout time.Duration) error {
	timeoutSec := int(timeout.Seconds())
	return d.client.ContainerStop(ctx, id, containerTypes.StopOptions{Timeout: &timeoutSec})
}

func (d *DockerRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	return d.client.ContainerRemove(ctx, id, containerTypes.RemoveOptions{
		Force:         force,
		RemoveVolumes: true,
	})
}

func (d *DockerRuntime) ListContainers(ctx context.Context, opts ListOptions) ([]ContainerInfo, error) {
	filterArgs := filters.NewArgs()
	for k, v := range opts.Labels {
		filterArgs.Add("label", k+"="+v)
	}

	containers, err := d.client.ContainerList(ctx, containerTypes.ListOptions{
		All:     opts.All,
		Limit:   opts.Limit,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		ports := make(map[int]int)
		for _, p := range c.Ports {
			if p.PublicPort > 0 {
				ports[int(p.PrivatePort)] = int(p.PublicPort)
			}
		}

		result = append(result, ContainerInfo{
			ID:     c.ID,
			Name:   name,
			Image:  c.Image,
			Status: c.Status,
			State:  c.State,
			Ports:  ports,
			Labels: c.Labels,
		})
	}
	return result, nil
}

func (d *DockerRuntime) InspectContainer(ctx context.Context, id string) (*ContainerInfo, error) {
	info, err := d.client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	ports := make(map[int]int)
	for cp, bindings := range info.NetworkSettings.Ports {
		containerPort := cp.Int()
		if len(bindings) > 0 {
			hp, err := strconv.Atoi(bindings[0].HostPort)
			if err == nil {
				ports[containerPort] = hp
			}
		}
	}

	var networks []string
	ipAddress := ""
	for name, net := range info.NetworkSettings.Networks {
		networks = append(networks, name)
		if ipAddress == "" {
			ipAddress = net.IPAddress
		}
	}

	return &ContainerInfo{
		ID:        info.ID,
		Name:      strings.TrimPrefix(info.Name, "/"),
		Image:     info.Config.Image,
		Status:    info.State.Status,
		State:     info.State.Status,
		Ports:     ports,
		Labels:    info.Config.Labels,
		Networks:  networks,
		IPAddress: ipAddress,
	}, nil
}

func (d *DockerRuntime) WaitContainer(ctx context.Context, id string) (int64, error) {
	waitCh, errCh := d.client.ContainerWait(ctx, id, containerTypes.WaitConditionNotRunning)
	select {
	case result := <-waitCh:
		return result.StatusCode, nil
	case err := <-errCh:
		return -1, err
	}
}

func (d *DockerRuntime) PullImage(ctx context.Context, image string) error {
	reader, err := d.client.ImagePull(ctx, image, imageTypes.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}
	defer reader.Close()
	_, err = io.Copy(io.Discard, reader)
	return err
}

func (d *DockerRuntime) BuildImage(ctx context.Context, contextPath string, opts BuildOptions) (string, error) {
	args := []string{"build"}

	for _, tag := range opts.Tags {
		args = append(args, "-t", tag)
	}

	if opts.Dockerfile != "" {
		args = append(args, "-f", opts.Dockerfile)
	}

	for k, v := range opts.BuildArgs {
		args = append(args, "--build-arg", k+"="+v)
	}

	if opts.NoCache {
		args = append(args, "--no-cache")
	}

	args = append(args, contextPath)

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker build failed: %w\nstderr: %s", err, stderr.String())
	}

	if len(opts.Tags) > 0 {
		return opts.Tags[0], nil
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (d *DockerRuntime) ListImages(ctx context.Context) ([]ImageInfo, error) {
	images, err := d.client.ImageList(ctx, imageTypes.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]ImageInfo, 0, len(images))
	for _, img := range images {
		result = append(result, ImageInfo{
			ID:      img.ID,
			Tags:    img.RepoTags,
			Size:    img.Size,
			Created: time.Unix(img.Created, 0),
		})
	}
	return result, nil
}

func (d *DockerRuntime) RemoveImage(ctx context.Context, image string, force bool) error {
	_, err := d.client.ImageRemove(ctx, image, imageTypes.RemoveOptions{Force: force})
	return err
}

func (d *DockerRuntime) ContainerLogs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error) {
	return d.client.ContainerLogs(ctx, id, containerTypes.LogsOptions{
		ShowStdout: opts.Stdout || (!opts.Stdout && !opts.Stderr),
		ShowStderr: opts.Stderr || (!opts.Stdout && !opts.Stderr),
		Follow:     opts.Follow,
		Tail:       opts.Tail,
		Since:      opts.Since,
		Timestamps: opts.Timestamps,
	})
}

func (d *DockerRuntime) ExecInContainer(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
	execCfg := containerTypes.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  opts.Interactive,
		Tty:          opts.TTY,
		Env:          opts.Env,
		WorkingDir:   opts.WorkingDir,
	}

	execResp, err := d.client.ContainerExecCreate(ctx, id, execCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := d.client.ContainerExecAttach(ctx, execResp.ID, containerTypes.ExecAttachOptions{
		Tty: opts.TTY,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer attachResp.Close()

	var stdout, stderr bytes.Buffer
	if opts.TTY {
		if opts.Stdout != nil {
			if _, err := io.Copy(opts.Stdout, attachResp.Reader); err != nil {
				return nil, fmt.Errorf("failed to read exec output: %w", err)
			}
		} else {
			if _, err := io.Copy(&stdout, attachResp.Reader); err != nil {
				return nil, fmt.Errorf("failed to read exec output: %w", err)
			}
		}
	} else {
		outWriter := io.Writer(&stdout)
		if opts.Stdout != nil {
			outWriter = opts.Stdout
		}
		if _, err := stdcopy.StdCopy(outWriter, &stderr, attachResp.Reader); err != nil {
			return nil, fmt.Errorf("failed to read exec output: %w", err)
		}
	}

	inspectResp, err := d.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec: %w", err)
	}

	return &ExecResult{
		ExitCode: inspectResp.ExitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

func (d *DockerRuntime) CreateNetwork(ctx context.Context, name string, opts NetworkOptions) (string, error) {
	driver := opts.Driver
	if driver == "" {
		driver = "bridge"
	}

	ncOpts := networkTypes.CreateOptions{
		Driver:   driver,
		Internal: opts.Internal,
		Labels:   opts.Labels,
	}

	if opts.Subnet != "" || opts.Gateway != "" {
		ipam := &networkTypes.IPAM{
			Config: []networkTypes.IPAMConfig{
				{
					Subnet:  opts.Subnet,
					Gateway: opts.Gateway,
				},
			},
		}
		ncOpts.IPAM = ipam
	}

	resp, err := d.client.NetworkCreate(ctx, name, ncOpts)
	if err != nil {
		return "", fmt.Errorf("failed to create network %s: %w", name, err)
	}
	return resp.ID, nil
}

func (d *DockerRuntime) RemoveNetwork(ctx context.Context, id string) error {
	return d.client.NetworkRemove(ctx, id)
}

func (d *DockerRuntime) ConnectNetwork(ctx context.Context, networkID, containerID string) error {
	return d.client.NetworkConnect(ctx, networkID, containerID, nil)
}

func (d *DockerRuntime) DisconnectNetwork(ctx context.Context, networkID, containerID string) error {
	return d.client.NetworkDisconnect(ctx, networkID, containerID, false)
}

func (d *DockerRuntime) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	networks, err := d.client.NetworkList(ctx, networkTypes.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]NetworkInfo, 0, len(networks))
	for _, n := range networks {
		var subnet string
		if n.IPAM.Config != nil && len(n.IPAM.Config) > 0 {
			subnet = n.IPAM.Config[0].Subnet
		}
		result = append(result, NetworkInfo{
			ID:     n.ID,
			Name:   n.Name,
			Driver: n.Driver,
			Subnet: subnet,
			Labels: n.Labels,
		})
	}
	return result, nil
}

func (d *DockerRuntime) CreateVolume(ctx context.Context, name string, opts VolumeOptions) (string, error) {
	vol, err := d.client.VolumeCreate(ctx, volumeTypes.CreateOptions{
		Name:   name,
		Driver: opts.Driver,
		Labels: opts.Labels,
	})
	if err != nil {
		return "", err
	}
	return vol.Name, nil
}

func (d *DockerRuntime) RemoveVolume(ctx context.Context, name string, force bool) error {
	return d.client.VolumeRemove(ctx, name, force)
}

func (d *DockerRuntime) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	resp, err := d.client.VolumeList(ctx, volumeTypes.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]VolumeInfo, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		result = append(result, VolumeInfo{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Labels:     v.Labels,
		})
	}
	return result, nil
}

func (d *DockerRuntime) InspectVolume(ctx context.Context, name string) (*VolumeInfo, error) {
	vol, err := d.client.VolumeInspect(ctx, name)
	if err != nil {
		return nil, err
	}
	return &VolumeInfo{
		Name:       vol.Name,
		Driver:     vol.Driver,
		Mountpoint: vol.Mountpoint,
		Labels:     vol.Labels,
	}, nil
}
