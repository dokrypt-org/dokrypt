# Contributing to Dokrypt

Thanks for your interest in contributing to Dokrypt.

## Getting Started

1. Fork the repository
2. Clone your fork
3. Create a feature branch: `git checkout -b feat/my-feature`
4. Make your changes
5. Run tests: `make test`
6. Run linter: `make lint`
7. Commit and push
8. Open a pull request

## Development Setup

```bash
git clone https://github.com/dokrypt-org/dokrypt.git
cd dokrypt
make build
make test
```

### Requirements

- Go 1.24+
- Docker (for integration tests)
- golangci-lint (for linting)

## Pull Requests

- Keep PRs focused — one feature or fix per PR
- Include tests for new functionality
- Make sure all tests pass before submitting
- Follow existing code style and patterns

## Reporting Bugs

Open an issue at [github.com/dokrypt-org/dokrypt/issues](https://github.com/dokrypt-org/dokrypt/issues) with:

- Dokrypt version (`dokrypt version`)
- OS and architecture
- Steps to reproduce
- Expected vs actual behavior

## Feature Requests

Open an issue with the `enhancement` label describing:

- The problem you're trying to solve
- Your proposed solution
- Any alternatives you've considered
