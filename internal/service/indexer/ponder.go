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

const (
	defaultPonderImage = "node:18-alpine"
	defaultPonderPort  = 42069
)

type PonderService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func NewPonder(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*PonderService, error) {
	deps := cfg.DependsOn
	if cfg.Chain != "" {
		deps = append(deps, cfg.Chain)
	}

	return &PonderService{
		BaseService: service.BaseService{
			ServiceName:  name,
			ServiceType:  "ponder",
			Runtime:      runtime,
			ProjectName:  projectName,
			HostPorts:    make(map[string]int),
			ServiceURLs:  make(map[string]string),
			Dependencies: deps,
		},
		cfg: cfg,
	}, nil
}

func (s *PonderService) Start(ctx context.Context) error {
	slog.Info("starting ponder indexer", "service", s.ServiceName)

	image := s.cfg.Image
	if image == "" {
		image = defaultPonderImage
	}

	port := s.cfg.Port
	if port == 0 {
		port = defaultPonderPort
	}

	chainHost := "ethereum"
	if s.cfg.Chain != "" {
		chainHost = s.cfg.Chain
	} else if len(s.cfg.DependsOn) > 0 {
		chainHost = s.cfg.DependsOn[0]
	}
	rpcURL := fmt.Sprintf("http://dokrypt-%s-%s:8545", s.ProjectName, chainHost)

	command := s.cfg.Command
	if len(command) == 0 {
		command = []string{"sh", "-c", "npm install -g ponder && ponder dev --port " + fmt.Sprintf("%d", port)}
	}

	env := map[string]string{
		"PONDER_RPC_URL_1": rpcURL,
	}
	for k, v := range s.cfg.Environment {
		env[k] = v
	}

	var volumes []container.VolumeMount
	for _, v := range s.cfg.Volumes {
		volumes = append(volumes, container.VolumeMount{
			Source: v,
			Target: "/app",
		})
	}

	containerCfg := &container.ContainerConfig{
		Image:   image,
		Command: command,
		Ports:   map[int]int{port: 0},
		Env:     env,
		Volumes: volumes,
	}

	if len(volumes) > 0 {
		containerCfg.WorkingDir = "/app"
	}

	err := s.StartContainer(ctx, containerCfg)
	if err != nil {
		return err
	}

	if hp, ok := s.HostPorts[fmt.Sprintf("%d", port)]; ok {
		s.ServiceURLs["http"] = fmt.Sprintf("http://localhost:%d", hp)
	}
	return s.WaitForHealthy(ctx, s.Health)
}

func (s *PonderService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *PonderService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *PonderService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }

func (s *PonderService) Health(ctx context.Context) error {
	url, ok := s.ServiceURLs["http"]
	if !ok {
		return fmt.Errorf("ponder URL not available")
	}
	return s.HTTPHealthCheck(ctx, url+"/health")
}

func (s *PonderService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func PonderFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return NewPonder(name, cfg, runtime, projectName)
}
