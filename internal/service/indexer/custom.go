package indexer

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/dokrypt/dokrypt/internal/service"
)

type CustomIndexerService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func NewCustomIndexer(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*CustomIndexerService, error) {
	return &CustomIndexerService{
		BaseService: service.BaseService{
			ServiceName: name, ServiceType: "custom-indexer", Runtime: runtime, ProjectName: projectName,
			HostPorts: make(map[string]int), ServiceURLs: make(map[string]string), Dependencies: cfg.DependsOn,
		},
		cfg: cfg,
	}, nil
}

func (s *CustomIndexerService) Start(ctx context.Context) error {
	slog.Info("starting custom indexer", "service", s.ServiceName)
	image := s.cfg.Image
	if image == "" {
		return fmt.Errorf("custom indexer requires an image")
	}
	port := s.cfg.Port
	if port == 0 {
		port = 3000
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
	return s.WaitForHealthy(ctx, s.Health)
}

func (s *CustomIndexerService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *CustomIndexerService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *CustomIndexerService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }

func (s *CustomIndexerService) Health(ctx context.Context) error {
	url, ok := s.ServiceURLs["http"]
	if !ok {
		return fmt.Errorf("indexer URL not available")
	}
	return s.HTTPHealthCheck(ctx, url+"/health")
}

func (s *CustomIndexerService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func CustomIndexerFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return NewCustomIndexer(name, cfg, runtime, projectName)
}
