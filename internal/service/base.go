package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/dokrypt/dokrypt/internal/common"
	"github.com/dokrypt/dokrypt/internal/container"
)

type BaseService struct {
	ServiceName string
	ServiceType string
	Runtime     container.Runtime
	ProjectName string
	ContainerID string
	HostPorts   map[string]int
	ServiceURLs map[string]string
	Dependencies []string
	HealthURL   string // HTTP health check URL path, e.g. "/health"
	HealthPort  int    // Port to health check on
}

func (b *BaseService) Name() string { return b.ServiceName }

func (b *BaseService) Type() string { return b.ServiceType }

func (b *BaseService) Ports() map[string]int { return b.HostPorts }

func (b *BaseService) URLs() map[string]string { return b.ServiceURLs }

func (b *BaseService) DependsOn() []string { return b.Dependencies }

func (b *BaseService) GetContainerID() string { return b.ContainerID }

func (b *BaseService) StartContainer(ctx context.Context, cfg *container.ContainerConfig) error {
	containerName := fmt.Sprintf("dokrypt-%s-%s", b.ProjectName, b.ServiceName)
	cfg.Name = containerName
	if cfg.Labels == nil {
		cfg.Labels = make(map[string]string)
	}
	cfg.Labels["dokrypt.project"] = b.ProjectName
	cfg.Labels["dokrypt.service"] = b.ServiceName
	cfg.Labels["dokrypt.type"] = b.ServiceType

	networkName := fmt.Sprintf("dokrypt-%s", b.ProjectName)
	if len(cfg.Networks) == 0 {
		cfg.Networks = []string{networkName}
	}
	if cfg.NetworkAliases == nil {
		cfg.NetworkAliases = make(map[string][]string)
	}
	cfg.NetworkAliases[networkName] = append(cfg.NetworkAliases[networkName], b.ServiceName)

	slog.Info("creating service container", "service", b.ServiceName, "image", cfg.Image)

	_ = b.Runtime.StopContainer(ctx, containerName, 5*time.Second)
	_ = b.Runtime.RemoveContainer(ctx, containerName, true)

	if err := b.Runtime.PullImage(ctx, cfg.Image); err != nil {
		slog.Warn("failed to pull image, trying with cache", "image", cfg.Image, "error", err)
	}

	id, err := b.Runtime.CreateContainer(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create container for %s: %w", b.ServiceName, err)
	}
	b.ContainerID = id

	if err := b.Runtime.StartContainer(ctx, id); err != nil {
		return fmt.Errorf("failed to start container for %s: %w", b.ServiceName, err)
	}

	info, err := b.Runtime.InspectContainer(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to inspect container for %s: %w", b.ServiceName, err)
	}

	if b.HostPorts == nil {
		b.HostPorts = make(map[string]int)
	}
	for containerPort, hostPort := range info.Ports {
		b.HostPorts[fmt.Sprintf("%d", containerPort)] = hostPort
	}

	return nil
}

func (b *BaseService) StopContainer(ctx context.Context) error {
	if b.ContainerID == "" {
		return nil
	}
	slog.Info("stopping service container", "service", b.ServiceName)

	if err := b.Runtime.StopContainer(ctx, b.ContainerID, 10*time.Second); err != nil {
		slog.Warn("failed to stop container", "service", b.ServiceName, "error", err)
	}
	if err := b.Runtime.RemoveContainer(ctx, b.ContainerID, true); err != nil {
		return fmt.Errorf("failed to remove container for %s: %w", b.ServiceName, err)
	}
	b.ContainerID = ""
	return nil
}

func (b *BaseService) IsContainerRunning(ctx context.Context) bool {
	if b.ContainerID == "" {
		return false
	}
	info, err := b.Runtime.InspectContainer(ctx, b.ContainerID)
	if err != nil {
		return false
	}
	return info.State == "running"
}

func (b *BaseService) HTTPHealthCheck(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	return nil
}

func (b *BaseService) WaitForHealthy(ctx context.Context, checkFn func(context.Context) error) error {
	return common.Retry(ctx, common.RetryConfig{
		MaxAttempts:  30,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     3 * time.Second,
		Multiplier:   1.5,
		Jitter:       0.1,
	}, checkFn)
}

func (b *BaseService) ContainerLogs(ctx context.Context, opts LogOptions) (io.ReadCloser, error) {
	if b.ContainerID == "" {
		return nil, fmt.Errorf("service %s not started", b.ServiceName)
	}
	return b.Runtime.ContainerLogs(ctx, b.ContainerID, container.LogOptions{
		Follow:     opts.Follow,
		Tail:       opts.Tail,
		Since:      opts.Since,
		Timestamps: opts.Timestamps,
		Stdout:     true,
		Stderr:     true,
	})
}
