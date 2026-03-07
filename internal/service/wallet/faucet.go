package wallet

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/dokrypt/dokrypt/internal/service"
)

type FaucetService struct {
	service.BaseService
	cfg config.ServiceConfig
}

func NewFaucet(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (*FaucetService, error) {
	deps := cfg.DependsOn
	if cfg.Chain != "" {
		deps = append(deps, cfg.Chain)
	}
	return &FaucetService{
		BaseService: service.BaseService{
			ServiceName: name, ServiceType: "faucet", Runtime: runtime, ProjectName: projectName,
			HostPorts: make(map[string]int), ServiceURLs: make(map[string]string), Dependencies: deps,
		},
		cfg: cfg,
	}, nil
}

func (s *FaucetService) Start(ctx context.Context) error {
	slog.Info("starting faucet", "service", s.ServiceName)
	port := s.cfg.Port
	if port == 0 {
		port = 3001
	}

	chainHost := s.cfg.Chain
	if chainHost == "" && len(s.cfg.DependsOn) > 0 {
		chainHost = s.cfg.DependsOn[0]
	}
	if chainHost == "" {
		chainHost = "chain"
	}
	chainRPCURL := fmt.Sprintf("http://dokrypt-%s-%s:8545", s.ProjectName, chainHost)

	script := fmt.Sprintf(`const http = require('http');
const CHAIN_RPC = process.env.CHAIN_RPC_URL || '%s';
function rpcCall(method, params) {
  return new Promise((resolve, reject) => {
    const body = JSON.stringify({ jsonrpc: '2.0', id: 1, method, params });
    const u = new URL(CHAIN_RPC);
    const opts = { hostname: u.hostname, port: u.port || 80, path: u.pathname, method: 'POST', headers: { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(body) } };
    const req = http.request(opts, res => {
      let data = '';
      res.on('data', c => data += c);
      res.on('end', () => { try { resolve(JSON.parse(data)); } catch(e) { reject(e); } });
    });
    req.on('error', reject);
    req.write(body);
    req.end();
  });
}
const server = http.createServer(async (req, res) => {
  res.setHeader('Content-Type', 'application/json');
  try {
    if (req.method === 'GET' && req.url === '/health') {
      return res.end(JSON.stringify({ status: 'ok', chain_rpc: CHAIN_RPC }));
    }
    const m = req.url.match(/^\/balance\/(.+)/);
    if (req.method === 'GET' && m) {
      const addr = m[1];
      const result = await rpcCall('eth_getBalance', [addr, 'latest']);
      if (result.error) { res.statusCode = 500; return res.end(JSON.stringify({ error: result.error.message })); }
      return res.end(JSON.stringify({ address: addr, balance: result.result }));
    }
    if (req.method === 'POST' && req.url === '/fund') {
      let body = '';
      req.on('data', c => body += c);
      await new Promise(r => req.on('end', r));
      const data = JSON.parse(body);
      if (!data.address) { res.statusCode = 400; return res.end(JSON.stringify({ error: 'address required' })); }
      const accounts = await rpcCall('eth_accounts', []);
      if (!accounts.result || accounts.result.length === 0) { res.statusCode = 500; return res.end(JSON.stringify({ error: 'no accounts available' })); }
      const from = accounts.result[0];
      const amt = data.amount || '1';
      const weiHex = '0x' + (BigInt(Math.floor(parseFloat(amt) * 1e18))).toString(16);
      const tx = await rpcCall('eth_sendTransaction', [{ from, to: data.address, value: weiHex }]);
      if (tx.error) { res.statusCode = 500; return res.end(JSON.stringify({ error: tx.error.message })); }
      return res.end(JSON.stringify({ status: 'funded', address: data.address, amount: amt + ' ETH', tx_hash: tx.result }));
    }
    res.statusCode = 404;
    res.end(JSON.stringify({ error: 'not found' }));
  } catch (e) {
    res.statusCode = 500;
    res.end(JSON.stringify({ error: e.message }));
  }
});
server.listen(%d, '0.0.0.0', () => console.log('Faucet running on :%d, chain RPC: ' + CHAIN_RPC));`, chainRPCURL, port, port)

	err := s.StartContainer(ctx, &container.ContainerConfig{
		Image:   "node:18-alpine",
		Command: []string{"node", "-e", script},
		Ports:   map[int]int{port: 0},
		Env: map[string]string{
			"CHAIN_RPC_URL": chainRPCURL,
		},
	})
	if err != nil {
		return err
	}
	if hp, ok := s.HostPorts[fmt.Sprintf("%d", port)]; ok {
		s.ServiceURLs["http"] = fmt.Sprintf("http://localhost:%d", hp)
	}
	return s.WaitForHealthy(ctx, s.Health)
}

func (s *FaucetService) Stop(ctx context.Context) error    { return s.StopContainer(ctx) }
func (s *FaucetService) Restart(ctx context.Context) error { _ = s.Stop(ctx); return s.Start(ctx) }
func (s *FaucetService) IsRunning(ctx context.Context) bool { return s.IsContainerRunning(ctx) }
func (s *FaucetService) Health(ctx context.Context) error {
	url, ok := s.ServiceURLs["http"]
	if !ok {
		return fmt.Errorf("faucet URL not available")
	}
	return s.HTTPHealthCheck(ctx, url+"/health")
}

func (s *FaucetService) Logs(ctx context.Context, opts service.LogOptions) (io.ReadCloser, error) {
	return s.ContainerLogs(ctx, opts)
}

func FaucetFactory(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (service.Service, error) {
	return NewFaucet(name, cfg, runtime, projectName)
}
