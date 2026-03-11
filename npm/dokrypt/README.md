<p align="center">
  <img src="https://raw.githubusercontent.com/dokrypt-org/dokrypt/main/.github/logo.svg" alt="Dokrypt" width="60" />
</p>

<h1 align="center">Dokrypt</h1>

<p align="center">
  <strong>Accelerated container orchestration for Web3 development</strong>
</p>

<p align="center">
  <a href="https://github.com/dokrypt-org/dokrypt/releases"><img src="https://img.shields.io/github/v/release/dokrypt-org/dokrypt?style=flat-square&color=7c3aed" alt="Release" /></a>
  <a href="https://www.npmjs.com/package/dokrypt"><img src="https://img.shields.io/npm/v/dokrypt?style=flat-square&color=7c3aed" alt="npm" /></a>
  <a href="https://github.com/dokrypt-org/dokrypt/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/dokrypt-org/dokrypt/ci.yml?style=flat-square" alt="CI" /></a>
</p>

<p align="center">
  <a href="https://dokrypt.com">Website</a> &middot;
  <a href="https://docs.dokrypt.com">Docs</a> &middot;
  <a href="https://github.com/dokrypt-org/dokrypt">GitHub</a> &middot;
  <a href="https://discord.gg/HG94Jg6PS2">Discord</a>
</p>

---

Dokrypt spins up fully configured blockchain development environments in seconds. One command gives you a local chain, block explorer, IPFS, oracles, indexers, monitoring, and everything else your dApp needs.

## Install

```bash
npm install -g dokrypt
```

## Quick Start

```bash
dokrypt init my-app --template evm-defi
cd my-app
dokrypt up
```

That's it. Your local chain, block explorer, and services are running.

## What You Get

- **Chain management** — Anvil, Hardhat, Geth with forking, time-travel, balance manipulation
- **Built-in services** — Blockscout, IPFS, Chainlink/Pyth oracles, Ponder/Subgraph indexers, Prometheus + Grafana
- **Testing** — Gas reports, coverage, parallel execution, JSON output
- **Snapshots** — Save and restore entire environment state
- **CI/CD** — Generate GitHub Actions and GitLab CI workflows
- **Cross-chain** — Bridge simulator, multi-chain networking
- **Contract tools** — Verify on Etherscan/Arbiscan/Sourcify, replay transactions, track deployments
- **Plugins** — Gas profiler, MEV simulator, security scanner

## Templates

```bash
dokrypt init my-app --template evm-basic      # Counter + SimpleToken
dokrypt init my-app --template evm-token      # ERC-20 with vesting, staking, multisig
dokrypt init my-app --template evm-nft        # ERC-721 with marketplace + royalties
dokrypt init my-app --template evm-dao        # Governor + Treasury + Timelock
dokrypt init my-app --template evm-defi       # AMM + Lending + Staking + Oracle
dokrypt init my-app --template evm-arbitrum   # L2 bridge + Token gateway + Arbitrum fork
```

## Commands

| Command | Description |
|---|---|
| `dokrypt init` | Scaffold a new project from a template |
| `dokrypt up` | Start all services |
| `dokrypt down` | Stop all services |
| `dokrypt status` | Show running services |
| `dokrypt logs` | Stream service logs |
| `dokrypt test` | Run tests with gas/coverage |
| `dokrypt chain` | Chain manipulation (mine, time-travel, balances) |
| `dokrypt fork` | Fork mainnet chains |
| `dokrypt snapshot` | Save/restore environment state |
| `dokrypt bridge` | Cross-chain transfers |
| `dokrypt verify` | Verify contracts on explorers |
| `dokrypt replay` | Replay and debug live transactions |
| `dokrypt deploy` | Track deployments across chains |
| `dokrypt ci` | Generate CI/CD workflows |
| `dokrypt plugin` | Install and manage plugins |
| `dokrypt doctor` | Diagnose environment issues |

## Configuration

All configuration lives in `dokrypt.yaml`:

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
```

## Platform Support

| Platform | Architecture |
|----------|-------------|
| Linux | x64, ARM64 |
| macOS | x64, Apple Silicon (M1/M2/M3) |
| Windows | x64 |

## Requirements

- **Docker** (or Podman) running locally
- **Node.js 16+** for npm install

## Links

- [Documentation](https://docs.dokrypt.com)
- [GitHub](https://github.com/dokrypt-org/dokrypt)
- [Website](https://dokrypt.com)
- [Discord](https://discord.gg/HG94Jg6PS2)

## License

Copyright (c) 2026 Dokrypt. All rights reserved. See [LICENSE](https://github.com/dokrypt-org/dokrypt/blob/main/LICENSE) for details.
