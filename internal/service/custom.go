package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
)

type CustomService struct {
	BaseService
	cfg config.ServiceConfig
}

func NewCustomService(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*CustomService, error) {
	return &CustomService{
		BaseService: BaseService{
			ServiceName:  name,
			ServiceType:  "custom",
			Runtime:      runtime,
			ProjectName:  projectName,
			HostPorts:    make(map[string]int),
			ServiceURLs:  make(map[string]string),
			Dependencies: cfg.DependsOn,
		},
		cfg: cfg,
	}, nil
}

func (s *CustomService) Start(ctx context.Context) error {
	slog.Info("starting custom service", "service", s.ServiceName, "image", s.cfg.Image)

	ports := make(map[int]int)
	for _, p := range s.cfg.Ports {
		ports[p] = 0
	}
	if s.cfg.Port > 0 {
		ports[s.cfg.Port] = 0
	}

	var volumes []container.VolumeMount
	for _, v := range s.cfg.Volumes {
		vm := container.VolumeMount{}
		parts := strings.SplitN(v, ":", 3)
		switch len(parts) {
		case 1:
			vm.Source = parts[0]
			vm.Target = parts[0]
		case 2:
			vm.Source = parts[0]
			vm.Target = parts[1]
		case 3:
			vm.Source = parts[0]
			vm.Target = parts[1]
			vm.ReadOnly = parts[2] == "ro"
		}
		volumes = append(volumes, vm)
	}

	err := s.StartContainer(ctx, &container.ContainerConfig{
		Image:   s.cfg.Image,
		Command: s.cfg.Command,
		Ports:   ports,
		Env:     s.cfg.Environment,
		Volumes: volumes,
	})
	if err != nil {
		return err
	}

	for name, containerPort := range s.cfg.Ports {
		if hp, ok := s.HostPorts[fmt.Sprintf("%d", containerPort)]; ok {
			s.ServiceURLs[name] = fmt.Sprintf("http://localhost:%d", hp)
		}
	}

	if s.cfg.Healthcheck != nil && s.cfg.Healthcheck.HTTP != "" {
		return s.WaitForHealthy(ctx, s.Health)
	}

	return nil
}

func (s *CustomService) Stop(ctx context.Context) error { return s.StopContainer(ctx) }

func (s *CustomService) Restart(ctx context.Context) error {
	if err := s.Stop(ctx); err != nil {
		return err
	}
	return s.Start(ctx)
}

func (s *CustomService) IsRunning(ctx context.Context) bool {
	return s.IsContainerRunning(ctx)
}

func (s *CustomService) Health(ctx context.Context) error {
	if s.cfg.Healthcheck == nil || s.cfg.Healthcheck.HTTP == "" {
		return nil // No health check configured
	}
	for _, url := range s.ServiceURLs {
		return s.HTTPHealthCheck(ctx, url+s.cfg.Healthcheck.HTTP)
	}
	return nil
}

func (s *CustomService) Logs(ctx context.Context, opts LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func CustomFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (Service, error) {
	return NewCustomService(name, cfg, runtime, projectName)
}
