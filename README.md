<p align="center">
  <img src=".github/logo.svg" alt="Dokrypt" width="60" />
</p>

<h1 align="center">Dokrypt</h1>

<p align="center">
  <strong>Accelerated container orchestration for Web3 development</strong>
</p>

<p align="center">
  <a href="https://github.com/dokrypt-org/dokrypt/releases"><img src="https://img.shields.io/github/v/release/dokrypt-org/dokrypt?style=flat-square&color=7c3aed" alt="Release" /></a>
  <a href="https://www.npmjs.com/package/dokrypt"><img src="https://img.shields.io/npm/v/dokrypt?style=flat-square&color=7c3aed" alt="npm" /></a>
  <a href="https://github.com/dokrypt-org/dokrypt/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/dokrypt-org/dokrypt/ci.yml?style=flat-square" alt="CI" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/dokrypt-org/dokrypt?style=flat-square" alt="License" /></a>
</p>

<p align="center">
  <a href="https://docs.dokrypt.com">Documentation</a> &middot;
  <a href="https://github.com/dokrypt-org/dokrypt/issues">Issues</a> &middot;
  <a href="https://docs.dokrypt.com/quickstart">Quickstart</a> &middot;
  <a href="https://www.npmjs.com/package/dokrypt">npm</a> &middot;
  <a href="https://discord.gg/HG94Jg6PS2">Discord</a>
</p>

---

```bash
npm install -g dokrypt
dokrypt init my-app --template evm-defi
cd my-app
dokrypt up
```

## Platform Support

Dokrypt runs on **all major platforms**:

| Platform | Architecture | Install Method |
|----------|-------------|----------------|
| Linux | x64, ARM64 | npm, Binary, Docker |
| macOS | x64, Apple Silicon (M1/M2/M3) | npm, Binary |
| Windows | x64 | npm, Binary |

## Installation

### npm (recommended)

Works on all platforms. Automatically downloads the correct binary for your OS.

```bash
npm install -g dokrypt
```

Verify:

```bash
dokrypt version
```

### Binary Download

Download pre-built binaries from [GitHub Releases](https://github.com/dokrypt-org/dokrypt/releases/latest):

**Linux / macOS:**

```bash
# Linux x64
curl -L https://github.com/dokrypt-org/dokrypt/releases/latest/download/dokrypt_0.1.0_linux_amd64.tar.gz | tar xz
sudo mv dokrypt /usr/local/bin/

# macOS Apple Silicon
curl -L https://github.com/dokrypt-org/dokrypt/releases/latest/download/dokrypt_0.1.0_darwin_arm64.tar.gz | tar xz
sudo mv dokrypt /usr/local/bin/

# macOS Intel
curl -L https://github.com/dokrypt-org/dokrypt/releases/latest/download/dokrypt_0.1.0_darwin_amd64.tar.gz | tar xz
sudo mv dokrypt /usr/local/bin/
```

**Windows:**

Download `dokrypt_0.1.0_windows_amd64.zip` from [Releases](https://github.com/dokrypt-org/dokrypt/releases/latest), extract, and add to your PATH.

### Docker

Run Dokrypt without installing anything:

```bash
docker pull ghcr.io/dokrypt-org/dokrypt:latest
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/dokrypt-org/dokrypt:latest --help
```

Or pin to a specific version:

```bash
docker pull ghcr.io/dokrypt-org/dokrypt:0.1.0
```

### Go Install

```bash
go install github.com/dokrypt-org/dokrypt/cmd/dokrypt@latest
```

### Build from Source

```bash
git clone https://github.com/dokrypt-org/dokrypt.git
cd dokrypt
make build
# Binary at ./bin/dokrypt
```

### Requirements

- **Docker** (or Podman) running locally — required for all chain and service operations
- **Go 1.24+** — only if building from source

## Why Dokrypt

**The problem**: Setting up a Web3 dev environment means juggling Anvil, Hardhat, Docker Compose files, block explorers, oracles, IPFS nodes, indexers, and more. Each project starts with hours of configuration.

**Dokrypt fixes this**: Define your entire stack in a single `dokrypt.yaml`, run `dokrypt up`, and everything starts in the right order with the right configuration. When you're done, `dokrypt down` tears it all down cleanly.

| Without Dokrypt | With Dokrypt |
|---|---|
| Write Docker Compose for each service | `dokrypt up` |
| Configure chain RPC URLs manually | Auto-wired service discovery |
| Set up block explorer from scratch | Built-in Blockscout/Otterscan |
| Run separate test commands | `dokrypt test --gas-report --coverage` |
| Manual CI pipeline setup | `dokrypt ci generate --provider github` |

## Features

### Chain Management
- **Multiple engines**: Anvil, Hardhat, Geth — switch with one flag
- **Chain manipulation**: Mine blocks, time-travel, set balances, impersonate accounts
- **Mainnet forking**: Fork any EVM chain at any block height
- **Multi-chain**: Run multiple chains simultaneously with automatic networking

### Built-in Services
- **Block Explorers**: Blockscout, Otterscan
- **Oracles**: Chainlink mock, Pyth mock, custom oracles
- **Indexers**: Ponder, Subgraph, custom indexers
- **IPFS**: Local IPFS node with pinning
- **Monitoring**: Prometheus + Grafana dashboards
- **Bridge**: Cross-chain asset transfers between local chains
- **Wallet**: Faucet service for test tokens

### Developer Tools
- **Test runner**: Built-in test execution with gas reports, coverage, parallel mode, JSON output
- **Snapshots**: Save and restore environment state instantly
- **CI/CD**: Generate GitHub Actions and GitLab CI workflows with `dokrypt ci generate`
- **Plugins**: Extend with gas profilers, MEV simulators, security scanners
- **Marketplace**: Share and install community templates and plugins

### Project Templates

```bash
dokrypt init my-app --template evm-basic      # Counter + SimpleToken contracts
dokrypt init my-app --template evm-token      # ERC-20 with vesting, staking, multisig
dokrypt init my-app --template evm-nft        # ERC-721 with marketplace + royalties
dokrypt init my-app --template evm-dao        # Governor + Treasury + Timelock
dokrypt init my-app --template evm-defi       # AMM + Lending + Staking + Oracle
```

## Quick Start

### 1. Create a project

```bash
dokrypt init my-dapp
cd my-dapp
```

### 2. Start the environment

```bash
dokrypt up
```

This reads your `dokrypt.yaml` and starts all configured services:

```
ethereum (anvil)    http://localhost:8545
blockscout          http://localhost:4000
ipfs                http://localhost:5001
```

### 3. Interact with the chain

```bash
dokrypt chain info                              # Chain ID, block number, accounts
dokrypt chain mine 10                           # Mine 10 blocks
dokrypt chain set-balance 0x... 1000            # Set balance to 1000 ETH
dokrypt chain time-travel 7d                    # Advance 7 days
dokrypt exec ethereum "cast call 0x..."         # Run commands inside containers
```

### 4. Run tests

```bash
dokrypt test                                    # Run all tests
dokrypt test --gas-report                       # With gas profiling
dokrypt test --coverage --parallel 4            # Coverage + parallel execution
dokrypt test --json > results.json              # Machine-readable output
```

### 5. Fork mainnet

```bash
dokrypt fork ethereum --block 19000000          # Fork at specific block
dokrypt fork accounts                           # List funded accounts
dokrypt fork fund 0x... 100                     # Fund an address
```

### 6. Snapshots

```bash
dokrypt snapshot save before-exploit            # Save state
dokrypt snapshot restore before-exploit         # Restore state
dokrypt snapshot list                           # List all snapshots
```

### 7. Tear down

```bash
dokrypt down                                    # Stop everything
```

## Configuration

All configuration lives in `dokrypt.yaml` at your project root:

```yaml
name: my-dapp
version: "1"

chains:
  ethereum:
    engine: anvil
    chain_id: 31337
    block_time: 1
    accounts: 10
    balance: "10000"

services:
  blockscout:
    type: explorer:blockscout
    port: 4000
  ipfs:
    type: ipfs
    port: 5001
  prometheus:
    type: monitoring:prometheus
    port: 9090

settings:
  gas_report: true
  coverage: true
```

## CLI Reference

| Command | Description |
|---|---|
| `dokrypt init` | Scaffold a new project from a template |
| `dokrypt up` | Start all services |
| `dokrypt down` | Stop all services |
| `dokrypt restart` | Restart services |
| `dokrypt status` | Show running services |
| `dokrypt logs` | Stream service logs |
| `dokrypt test` | Run tests with gas/coverage |
| `dokrypt exec` | Execute commands in containers |
| `dokrypt chain` | Chain manipulation (mine, time-travel, balances) |
| `dokrypt fork` | Fork mainnet chains |
| `dokrypt snapshot` | Save/restore environment state |
| `dokrypt bridge` | Cross-chain transfers |
| `dokrypt ci` | Generate CI/CD workflows |
| `dokrypt config` | Validate and inspect config |
| `dokrypt template` | Manage project templates |
| `dokrypt plugin` | Install and manage plugins |
| `dokrypt marketplace` | Browse community templates/plugins |
| `dokrypt doctor` | Diagnose environment issues |

## Architecture

```
cmd/dokrypt/          CLI entrypoint
internal/
  cli/                Command implementations (Cobra)
  engine/             Core orchestration engine
  chain/              Chain management (Anvil, Hardhat, Geth)
  container/          Docker/Podman runtime abstraction
  service/            Service implementations
    bridge/           Cross-chain bridge
    explorer/         Block explorers (Blockscout, Otterscan)
    indexer/          Indexers (Ponder, Subgraph)
    ipfs/             IPFS node
    monitoring/       Prometheus + Grafana
    oracle/           Oracles (Chainlink, Pyth)
    wallet/           Faucet service
  config/             YAML config parsing and validation
  state/              Snapshot and state management
  network/            DNS, proxy, multi-chain networking
  plugin/             Plugin system (hooks, sandboxing, binary)
  template/           Project scaffolding and templates
  testrunner/         Test execution, gas, coverage
  marketplace/        Template/plugin registry
  rpc/                JSON-RPC and WebSocket clients
  abi/                ABI decoder
  common/             Shared utilities
pkg/
  types/              Public types (Config, Chain, Service)
  testing/dtest/      Test helpers for Dokrypt environments
plugins/              Built-in plugins
  gas-profiler/       Gas usage tracking and suggestions
  mev-simulator/      MEV opportunity analysis
  security-scanner/   Contract security scanning
```

## Development

```bash
make build           # Build binary
make test            # Run all tests
make test-short      # Quick tests
make lint            # Run linter
make fmt             # Format code
make docker-build    # Build Docker image
make release         # GoReleaser snapshot
```

## Documentation

Full documentation is available at **[docs.dokrypt.com](https://docs.dokrypt.com)**.

## License

Copyright (c) 2026 Dokrypt. All rights reserved. See [LICENSE](LICENSE) for details.
