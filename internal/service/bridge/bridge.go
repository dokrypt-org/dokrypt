package bridge

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/dokrypt/dokrypt/internal/service"
)

type BridgeService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func New(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*BridgeService, error) {
	deps := cfg.DependsOn
	deps = append(deps, cfg.Chains...)

	return &BridgeService{
		BaseService: service.BaseService{
			ServiceName: name, ServiceType: "mock-bridge", Runtime: runtime, ProjectName: projectName,
			HostPorts: make(map[string]int), ServiceURLs: make(map[string]string), Dependencies: deps,
		},
		cfg: cfg,
	}, nil
}

func (s *BridgeService) Start(ctx context.Context) error {
	slog.Info("starting bridge relay", "service", s.ServiceName, "chains", s.cfg.Chains)
	port := 7000

	script := `const http = require('http');
let msgId = 0;
const messages = [];
const server = http.createServer((req, res) => {
  res.setHeader('Content-Type', 'application/json');
  if (req.method === 'GET' && req.url === '/health') {
    return res.end(JSON.stringify({ status: 'ok', queued: messages.length }));
  }
  if (req.method === 'GET' && req.url === '/messages') {
    return res.end(JSON.stringify({ messages }));
  }
  if (req.method === 'POST' && req.url === '/relay') {
    let body = '';
    req.on('data', c => body += c);
    req.on('end', () => {
      try {
        const data = JSON.parse(body);
        const msg = {
          id: ++msgId,
          from_chain: data.from_chain || '',
          to_chain: data.to_chain || '',
          sender: data.sender || '',
          recipient: data.recipient || '',
          amount: data.amount || '0',
          data: data.data || '',
          status: 'relayed',
          timestamp: new Date().toISOString()
        };
        messages.push(msg);
        res.end(JSON.stringify({ status: 'relayed', message: msg }));
      } catch (e) {
        res.statusCode = 400;
        res.end(JSON.stringify({ error: 'invalid JSON' }));
      }
    });
    return;
  }
  res.statusCode = 404;
  res.end(JSON.stringify({ error: 'not found' }));
});
server.listen(7000, '0.0.0.0', () => console.log('Bridge relay running on :7000'));`

	err := s.StartContainer(ctx, &container.ContainerConfig{
		Image:   "node:18-alpine",
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

func (s *BridgeService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *BridgeService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *BridgeService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }
func (s *BridgeService) Health(ctx context.Context) error {
	url, ok := s.ServiceURLs["http"]
	if !ok {
		return fmt.Errorf("bridge URL not available")
	}
	return s.HTTPHealthCheck(ctx, url+"/health")
}

func (s *BridgeService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func Factory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return New(name, cfg, runtime, projectName)
}
