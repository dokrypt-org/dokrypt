package ipfs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/dokrypt/dokrypt/internal/service"
)

const (
	defaultImage       = "ipfs/kubo:latest"
	defaultAPIPort     = 5001
	defaultGatewayPort = 8080
)

type Service struct {
	service.BaseService
	cfg config.ServiceConfig
}

func New(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*Service, error) {
	apiPort := cfg.APIPort
	if apiPort == 0 {
		apiPort = defaultAPIPort
	}
	gatewayPort := cfg.GatewayPort
	if gatewayPort == 0 {
		gatewayPort = defaultGatewayPort
	}

	return &Service{
		BaseService: service.BaseService{
			ServiceName:  name,
			ServiceType:  "ipfs",
			Runtime:      runtime,
			ProjectName:  projectName,
			HostPorts:    make(map[string]int),
			ServiceURLs:  make(map[string]string),
			Dependencies: cfg.DependsOn,
		},
		cfg: cfg,
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	slog.Info("starting IPFS service", "service", s.ServiceName)

	apiPort := s.cfg.APIPort
	if apiPort == 0 {
		apiPort = defaultAPIPort
	}
	gatewayPort := s.cfg.GatewayPort
	if gatewayPort == 0 {
		gatewayPort = defaultGatewayPort
	}

	err := s.StartContainer(ctx, &container.ContainerConfig{
		Image: defaultImage,
		Ports: map[int]int{apiPort: 0, gatewayPort: 0},
		Env: map[string]string{
			"IPFS_PROFILE": "server",
			"IPFS_PATH":    "/data/ipfs",
		},
	})
	if err != nil {
		return err
	}

	if hp, ok := s.HostPorts[fmt.Sprintf("%d", apiPort)]; ok {
		s.ServiceURLs["api"] = fmt.Sprintf("http://localhost:%d", hp)
	}
	if hp, ok := s.HostPorts[fmt.Sprintf("%d", gatewayPort)]; ok {
		s.ServiceURLs["gateway"] = fmt.Sprintf("http://localhost:%d", hp)
	}

	if err := s.WaitForHealthy(ctx, s.Health); err != nil {
		return err
	}

	ipfsCfg := DefaultIPFSConfig()
	for _, cmd := range ipfsCfg.InitCommands() {
		result, err := s.Runtime.ExecInContainer(ctx, s.ContainerID, cmd, container.ExecOptions{})
		if err != nil {
			slog.Warn("failed to run IPFS init command", "cmd", cmd, "error", err)
		} else if result.ExitCode != 0 {
			slog.Warn("IPFS init command failed", "cmd", cmd, "exit_code", result.ExitCode, "stderr", result.Stderr)
		}
	}

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	return s.StopContainer(ctx)
}

func (s *Service) Restart(ctx context.Context) error {
	if err := s.Stop(ctx); err != nil {
		return err
	}
	return s.Start(ctx)
}

func (s *Service) IsRunning(ctx context.Context) bool {
	return s.IsContainerRunning(ctx)
}

func (s *Service) Health(ctx context.Context) error {
	apiURL, ok := s.ServiceURLs["api"]
	if !ok {
		return fmt.Errorf("IPFS API URL not available")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL+"/api/v0/id", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("IPFS health check returned status %d", resp.StatusCode)
	}
	return nil
}

func (s *Service) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func Factory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return New(name, cfg, runtime, projectName)
}
