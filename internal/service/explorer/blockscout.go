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

const blockscoutImage = "blockscout/blockscout:latest"

type BlockscoutService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func NewBlockscout(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*BlockscoutService, error) {
	deps := cfg.DependsOn
	if cfg.Chain != "" {
		deps = append(deps, cfg.Chain)
	}

	return &BlockscoutService{
		BaseService: service.BaseService{
			ServiceName:  name,
			ServiceType:  "blockscout",
			Runtime:      runtime,
			ProjectName:  projectName,
			HostPorts:    make(map[string]int),
			ServiceURLs:  make(map[string]string),
			Dependencies: deps,
		},
		cfg: cfg,
	}, nil
}

func (s *BlockscoutService) Start(ctx context.Context) error {
	slog.Info("starting blockscout explorer", "service", s.ServiceName)
	port := s.cfg.Port
	if port == 0 {
		port = 4000
	}

	chainHost := "ethereum"
	if s.cfg.Chain != "" {
		chainHost = s.cfg.Chain
	} else if len(s.cfg.DependsOn) > 0 {
		chainHost = s.cfg.DependsOn[0]
	}
	rpcURL := fmt.Sprintf("http://dokrypt-%s-%s:8545", s.ProjectName, chainHost)

	databaseURL := fmt.Sprintf(
		"postgresql://postgres:postgres@dokrypt-%s-postgres:5432/blockscout",
		s.ProjectName,
	)

	env := map[string]string{
		"ETHEREUM_JSONRPC_HTTP_URL": rpcURL,
		"DATABASE_URL":             databaseURL,
	}
	for k, v := range s.cfg.Environment {
		env[k] = v
	}

	err := s.StartContainer(ctx, &container.ContainerConfig{
		Image: blockscoutImage,
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

func (s *BlockscoutService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *BlockscoutService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *BlockscoutService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }

func (s *BlockscoutService) Health(ctx context.Context) error {
	url, ok := s.ServiceURLs["http"]
	if !ok {
		return fmt.Errorf("blockscout URL not available")
	}
	return s.HTTPHealthCheck(ctx, url)
}

func (s *BlockscoutService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func BlockscoutFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return NewBlockscout(name, cfg, runtime, projectName)
}
