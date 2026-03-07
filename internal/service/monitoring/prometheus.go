package monitoring

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/dokrypt/dokrypt/internal/service"
)

type PrometheusService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func NewPrometheus(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*PrometheusService, error) {
	return &PrometheusService{
		BaseService: service.BaseService{
			ServiceName: name, ServiceType: "prometheus", Runtime: runtime, ProjectName: projectName,
			HostPorts: make(map[string]int), ServiceURLs: make(map[string]string), Dependencies: cfg.DependsOn,
		},
		cfg: cfg,
	}, nil
}

func (s *PrometheusService) Start(ctx context.Context) error {
	slog.Info("starting prometheus", "service", s.ServiceName)
	port := 9090
	err := s.StartContainer(ctx, &container.ContainerConfig{
		Image: "prom/prometheus:latest",
		Ports: map[int]int{port: 0},
	})
	if err != nil {
		return err
	}
	if hp, ok := s.HostPorts[fmt.Sprintf("%d", port)]; ok {
		s.ServiceURLs["http"] = fmt.Sprintf("http://localhost:%d", hp)
	}
	return s.WaitForHealthy(ctx, s.Health)
}

func (s *PrometheusService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *PrometheusService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *PrometheusService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }

func (s *PrometheusService) Health(ctx context.Context) error {
	url, ok := s.ServiceURLs["http"]
	if !ok {
		return fmt.Errorf("prometheus URL not available")
	}
	return s.HTTPHealthCheck(ctx, url+"/-/healthy")
}

func (s *PrometheusService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func PrometheusFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return NewPrometheus(name, cfg, runtime, projectName)
}
