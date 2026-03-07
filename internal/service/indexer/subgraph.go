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
	subgraphImage      = "graphprotocol/graph-node:latest"
	defaultGraphQLPort = 8000
	defaultAdminPort   = 8020
)

type SubgraphService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func NewSubgraph(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*SubgraphService, error) {
	deps := cfg.DependsOn
	if cfg.Chain != "" {
		deps = append(deps, cfg.Chain)
	}

	return &SubgraphService{
		BaseService: service.BaseService{
			ServiceName:  name,
			ServiceType:  "subgraph",
			Runtime:      runtime,
			ProjectName:  projectName,
			HostPorts:    make(map[string]int),
			ServiceURLs:  make(map[string]string),
			Dependencies: deps,
		},
		cfg: cfg,
	}, nil
}

func (s *SubgraphService) Start(ctx context.Context) error {
	slog.Info("starting subgraph indexer", "service", s.ServiceName)

	graphqlPort := s.cfg.GraphQLPort
	if graphqlPort == 0 {
		graphqlPort = defaultGraphQLPort
	}
	adminPort := s.cfg.AdminPort
	if adminPort == 0 {
		adminPort = defaultAdminPort
	}

	chainName := "mainnet"
	if s.cfg.Chain != "" {
		chainName = s.cfg.Chain
	} else if len(s.cfg.DependsOn) > 0 {
		chainName = s.cfg.DependsOn[0]
	}

	rpcURL := fmt.Sprintf("http://dokrypt-%s-%s:8545", s.ProjectName, chainName)

	ipfsURL := s.cfg.IPFS
	if ipfsURL == "" {
		ipfsURL = fmt.Sprintf("dokrypt-%s-ipfs:5001", s.ProjectName)
	}

	postgresHost := fmt.Sprintf("dokrypt-%s-postgres", s.ProjectName)
	postgresDB := "graph-node"
	postgresUser := "postgres"
	postgresPass := "postgres"

	env := map[string]string{
		"postgres_host": postgresHost,
		"postgres_db":   postgresDB,
		"postgres_user": postgresUser,
		"postgres_pass": postgresPass,
		"ipfs":          ipfsURL,
		"ethereum":      fmt.Sprintf("%s:%s", chainName, rpcURL),
	}
	for k, v := range s.cfg.Environment {
		env[k] = v
	}

	err := s.StartContainer(ctx, &container.ContainerConfig{
		Image: subgraphImage,
		Ports: map[int]int{graphqlPort: 0, adminPort: 0},
		Env:   env,
	})
	if err != nil {
		return err
	}

	if hp, ok := s.HostPorts[fmt.Sprintf("%d", graphqlPort)]; ok {
		s.ServiceURLs["graphql"] = fmt.Sprintf("http://localhost:%d", hp)
	}
	if hp, ok := s.HostPorts[fmt.Sprintf("%d", adminPort)]; ok {
		s.ServiceURLs["admin"] = fmt.Sprintf("http://localhost:%d", hp)
	}

	return s.WaitForHealthy(ctx, s.Health)
}

func (s *SubgraphService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *SubgraphService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *SubgraphService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }

func (s *SubgraphService) Health(ctx context.Context) error {
	url, ok := s.ServiceURLs["graphql"]
	if !ok {
		return fmt.Errorf("subgraph GraphQL URL not available")
	}
	return s.HTTPHealthCheck(ctx, url)
}

func (s *SubgraphService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func SubgraphFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return NewSubgraph(name, cfg, runtime, projectName)
}
