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

type PythMockService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func NewPythMock(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*PythMockService, error) {
	deps := cfg.DependsOn
	if cfg.Chain != "" {
		deps = append(deps, cfg.Chain)
	}
	return &PythMockService{
		BaseService: service.BaseService{
			ServiceName: name, ServiceType: "pyth-mock", Runtime: runtime, ProjectName: projectName,
			HostPorts: make(map[string]int), ServiceURLs: make(map[string]string), Dependencies: deps,
		},
		cfg: cfg,
	}, nil
}

func (s *PythMockService) Start(ctx context.Context) error {
	slog.Info("starting pyth mock oracle", "service", s.ServiceName)
	port := 6689

	script := `const http = require('http');
const url = require('url');
const feeds = {
  'e62df6c8b4a85fe1a67db44dc12de5db330f7ac66b72dc658afedf0f4a415b43': { id: 'e62df6c8b4a85fe1a67db44dc12de5db330f7ac66b72dc658afedf0f4a415b43', pair: 'BTC/USD', price: 6800000000000, expo: -8, conf: 1500000000 },
  'ff61491a931112ddf1bd8147cd1b641375f79f5825126d665480874634fd0ace': { id: 'ff61491a931112ddf1bd8147cd1b641375f79f5825126d665480874634fd0ace', pair: 'ETH/USD', price: 350000000000, expo: -8, conf: 500000000 },
  'ef0d8b6fda2ceba41da15d4095d1da392a0d2f8ed0c6c7bc0f4cfac8c280b56d': { id: 'ef0d8b6fda2ceba41da15d4095d1da392a0d2f8ed0c6c7bc0f4cfac8c280b56d', pair: 'SOL/USD', price: 15000000000, expo: -8, conf: 100000000 }
};
function makeVAA(f) {
  const ts = Math.floor(Date.now() / 1000);
  return Buffer.from(JSON.stringify({ price: f.price, conf: f.conf, expo: f.expo, publish_time: ts })).toString('base64');
}
function makePriceFeed(f) {
  const ts = Math.floor(Date.now() / 1000);
  return { id: f.id, price: { price: String(f.price), conf: String(f.conf), expo: f.expo, publish_time: ts }, ema_price: { price: String(f.price), conf: String(f.conf), expo: f.expo, publish_time: ts } };
}
http.createServer((req, res) => {
  res.setHeader('Content-Type', 'application/json');
  const parsed = url.parse(req.url, true);
  if (parsed.pathname === '/health') return res.end(JSON.stringify({ status: 'ok' }));
  if (parsed.pathname === '/api/latest_vaas') {
    const ids = parsed.query.ids ? [].concat(parsed.query.ids) : Object.keys(feeds);
    const vaas = ids.map(id => feeds[id] ? makeVAA(feeds[id]) : null).filter(Boolean);
    return res.end(JSON.stringify(vaas));
  }
  if (parsed.pathname === '/api/latest_price_feeds') {
    const ids = parsed.query.ids ? [].concat(parsed.query.ids) : Object.keys(feeds);
    const result = ids.map(id => feeds[id] ? makePriceFeed(feeds[id]) : null).filter(Boolean);
    return res.end(JSON.stringify(result));
  }
  res.statusCode = 404;
  res.end(JSON.stringify({ error: 'not found' }));
}).listen(6689, '0.0.0.0', () => console.log('Pyth mock oracle running on :6689'));`

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

func (s *PythMockService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *PythMockService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *PythMockService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }
func (s *PythMockService) Health(ctx context.Context) error {
	url, ok := s.ServiceURLs["http"]
	if !ok {
		return fmt.Errorf("pyth oracle URL not available")
	}
	return s.HTTPHealthCheck(ctx, url+"/health")
}

func (s *PythMockService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func PythFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return NewPythMock(name, cfg, runtime, projectName)
}
