package oracle

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/dokrypt/dokrypt/internal/service"
)

type CustomOracleService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func NewCustomOracle(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*CustomOracleService, error) {
	return &CustomOracleService{
		BaseService: service.BaseService{
			ServiceName: name, ServiceType: "custom-oracle", Runtime: runtime, ProjectName: projectName,
			HostPorts: make(map[string]int), ServiceURLs: make(map[string]string), Dependencies: cfg.DependsOn,
		},
		cfg: cfg,
	}, nil
}

func (s *CustomOracleService) Start(ctx context.Context) error {
	slog.Info("starting custom oracle", "service", s.ServiceName)
	image := s.cfg.Image
	if image == "" {
		return fmt.Errorf("custom oracle requires an image")
	}
	port := s.cfg.Port
	if port == 0 {
		port = 8080
	}
	err := s.StartContainer(ctx, &container.ContainerConfig{
		Image: image,
		Ports: map[int]int{port: 0},
		Env:   s.cfg.Environment,
	})
	if err != nil {
		return err
	}
	if hp, ok := s.HostPorts[fmt.Sprintf("%d", port)]; ok {
		s.ServiceURLs["http"] = fmt.Sprintf("http://localhost:%d", hp)
	}
	return nil
}

func (s *CustomOracleService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *CustomOracleService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *CustomOracleService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }
func (s *CustomOracleService) Health(ctx context.Context) error  { return nil }

func (s *CustomOracleService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func CustomOracleFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return NewCustomOracle(name, cfg, runtime, projectName)
}
