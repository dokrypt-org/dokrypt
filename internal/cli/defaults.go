package cli

import (
	"github.com/dokrypt/dokrypt/internal/service"
	"github.com/dokrypt/dokrypt/internal/service/bridge"
	"github.com/dokrypt/dokrypt/internal/service/explorer"
	"github.com/dokrypt/dokrypt/internal/service/indexer"
	"github.com/dokrypt/dokrypt/internal/service/ipfs"
	"github.com/dokrypt/dokrypt/internal/service/monitoring"
	"github.com/dokrypt/dokrypt/internal/service/oracle"
	"github.com/dokrypt/dokrypt/internal/service/wallet"
)

func newDefaultRegistry() *service.Registry {
	r := service.NewRegistry()
	r.Register("ipfs", ipfs.Factory)
	r.Register("subgraph", indexer.SubgraphFactory)
	r.Register("ponder", indexer.PonderFactory)
	r.Register("blockscout", explorer.BlockscoutFactory)
	r.Register("otterscan", explorer.OtterscanFactory)
	r.Register("chainlink-mock", oracle.ChainlinkFactory)
	r.Register("pyth-mock", oracle.PythFactory)
	r.Register("grafana", monitoring.GrafanaFactory)
	r.Register("prometheus", monitoring.PrometheusFactory)
	r.Register("faucet", wallet.FaucetFactory)
	r.Register("mock-bridge", bridge.Factory)
	r.Register("custom", service.CustomFactory)
	return r
}
