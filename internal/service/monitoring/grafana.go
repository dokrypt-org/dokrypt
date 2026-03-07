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

const (
	defaultGrafanaAdminUser     = "admin"
	defaultGrafanaAdminPassword = "dokrypt"
)

type GrafanaService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func NewGrafana(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*GrafanaService, error) {
	return &GrafanaService{
		BaseService: service.BaseService{
			ServiceName: name, ServiceType: "grafana", Runtime: runtime, ProjectName: projectName,
			HostPorts: make(map[string]int), ServiceURLs: make(map[string]string), Dependencies: cfg.DependsOn,
		},
		cfg: cfg,
	}, nil
}

func (s *GrafanaService) Start(ctx context.Context) error {
	slog.Info("starting grafana", "service", s.ServiceName)
	port := s.cfg.Port
	if port == 0 {
		port = 3000
	}

	adminUser := defaultGrafanaAdminUser
	adminPassword := defaultGrafanaAdminPassword

	env := map[string]string{
		"GF_SECURITY_ADMIN_USER":     adminUser,
		"GF_SECURITY_ADMIN_PASSWORD": adminPassword,
		"GF_AUTH_ANONYMOUS_ENABLED":  "true",
		"GF_AUTH_ANONYMOUS_ORG_ROLE": "Viewer",
	}
	for k, v := range s.cfg.Environment {
		env[k] = v
	}

	err := s.StartContainer(ctx, &container.ContainerConfig{
		Image: "grafana/grafana:latest",
		Ports: map[int]int{port: 0},
		Env:   env,
	})
	if err != nil {
		return err
	}
	if hp, ok := s.HostPorts[fmt.Sprintf("%d", port)]; ok {
		s.ServiceURLs["http"] = fmt.Sprintf("http://localhost:%d", hp)
	}
	return s.WaitForHealthy(ctx, s.Health)
}

func (s *GrafanaService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *GrafanaService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *GrafanaService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }

func (s *GrafanaService) Health(ctx context.Context) error {
	url, ok := s.ServiceURLs["http"]
	if !ok {
		return fmt.Errorf("grafana URL not available")
	}
	return s.HTTPHealthCheck(ctx, url+"/api/health")
}

func (s *GrafanaService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func GrafanaFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return NewGrafana(name, cfg, runtime, projectName)
}
