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

type ChainlinkMockService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func NewChainlinkMock(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*ChainlinkMockService, error) {
	deps := cfg.DependsOn
	if cfg.Chain != "" {
		deps = append(deps, cfg.Chain)
	}
	return &ChainlinkMockService{
		BaseService: service.BaseService{
			ServiceName: name, ServiceType: "chainlink-mock", Runtime: runtime, ProjectName: projectName,
			HostPorts: make(map[string]int), ServiceURLs: make(map[string]string), Dependencies: deps,
		},
		cfg: cfg,
	}, nil
}

func (s *ChainlinkMockService) Start(ctx context.Context) error {
	slog.Info("starting chainlink mock oracle", "service", s.ServiceName)
	port := 6688

	script := `const http = require('http');
const feeds = {'ETH/USD':3500.00,'BTC/USD':68000.00,'LINK/USD':15.50};
http.createServer((req, res) => {
  res.setHeader('Content-Type','application/json');
  if (req.url==='/health') return res.end(JSON.stringify({status:'ok'}));
  if (req.url==='/feeds') return res.end(JSON.stringify(feeds));
  const m = req.url.match(/^\/feeds\/(.+)/);
  if (m) { const p=decodeURIComponent(m[1]); if (feeds[p]!==undefined) return res.end(JSON.stringify({pair:p,price:feeds[p]})); }
  res.statusCode=404; res.end(JSON.stringify({error:'not found'}));
}).listen(6688,'0.0.0.0',()=>console.log('Oracle mock running on :6688'));`

	err := s.StartContainer(ctx, &container.ContainerConfig{
		Image:   "node:20-alpine",
		Command: []string{"node", "-e", script},
		Ports:   map[int]int{port: 0},
	})
	if err != nil {
		return err
	}
	if hp, ok := s.HostPorts[fmt.Sprintf("%d", port)]; ok {
		s.ServiceURLs["http"] = fmt.Sprintf("http://localhost:%d", hp)
	}
	return s.WaitForHealthy(ctx, s.Health)
}

func (s *ChainlinkMockService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *ChainlinkMockService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *ChainlinkMockService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }
func (s *ChainlinkMockService) Health(ctx context.Context) error {
	url, ok := s.ServiceURLs["http"]
	if !ok {
		return fmt.Errorf("oracle URL not available")
	}
	return s.HTTPHealthCheck(ctx, url+"/health")
}

func (s *ChainlinkMockService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func ChainlinkFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return NewChainlinkMock(name, cfg, runtime, projectName)
}
