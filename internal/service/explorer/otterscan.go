package explorer

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/dokrypt/dokrypt/internal/service"
)

const otterscanImage = "otterscan/otterscan:latest"

type OtterscanService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func NewOtterscan(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*OtterscanService, error) {
	deps := cfg.DependsOn
	if cfg.Chain != "" {
		deps = append(deps, cfg.Chain)
	}
	return &OtterscanService{
		BaseService: service.BaseService{
			ServiceName: name, ServiceType: "otterscan", Runtime: runtime, ProjectName: projectName,
			HostPorts: make(map[string]int), ServiceURLs: make(map[string]string), Dependencies: deps,
		},
		cfg: cfg,
	}, nil
}

func (s *OtterscanService) Start(ctx context.Context) error {
	slog.Info("starting otterscan explorer", "service", s.ServiceName)

	chainHost := "ethereum" // default
	if s.cfg.Chain != "" {
		chainHost = s.cfg.Chain
	} else if len(s.cfg.DependsOn) > 0 {
		chainHost = s.cfg.DependsOn[0]
	}
	rpcURL := fmt.Sprintf("http://dokrypt-%s-%s:8545", s.ProjectName, chainHost)

	env := map[string]string{
		"VITE_CONFIG_JSON": fmt.Sprintf(`{"erigonURL":"%s","assetsURLPrefix":"https://assets.otterscan.io"}`, rpcURL),
	}
	for k, v := range s.cfg.Environment {
		env[k] = v
	}

	err := s.StartContainer(ctx, &container.ContainerConfig{
		Image: otterscanImage,
		Ports: map[int]int{80: 0},
		Env:   env,
	})
	if err != nil {
		return err
	}
	if hp, ok := s.HostPorts["80"]; ok {
		s.ServiceURLs["http"] = fmt.Sprintf("http://localhost:%d", hp)
	}
	return s.WaitForHealthy(ctx, s.Health)
}

func (s *OtterscanService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *OtterscanService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *OtterscanService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }

func (s *OtterscanService) Health(ctx context.Context) error {
	url, ok := s.ServiceURLs["http"]
	if !ok {
		return fmt.Errorf("otterscan URL not available")
	}
	return s.HTTPHealthCheck(ctx, url)
}

func (s *OtterscanService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func OtterscanFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return NewOtterscan(name, cfg, runtime, projectName)
}
