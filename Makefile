.PHONY: build test test-short lint fmt vet generate clean docker-build docker release help

BINARY_NAME=dokrypt
BUILD_DIR=./bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-s -w -X github.com/dokrypt/dokrypt/internal/cli.Version=$(VERSION) -X github.com/dokrypt/dokrypt/internal/cli.Commit=$(COMMIT) -X github.com/dokrypt/dokrypt/internal/cli.Date=$(DATE)"

## build: Build the CLI binary
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dokrypt

## test: Run all tests with race detection
test:
	go test ./... -v -race

## test-short: Run tests in short mode
test-short:
	go test ./... -short

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format all Go source files
fmt:
	go fmt ./...
	gofumpt -l -w .

## vet: Run go vet
vet:
	go vet ./...

## generate: Run go generate
generate:
	go generate ./...

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## docker-build: Build Docker image for CLI
docker-build:
	docker build -f deploy/docker/Dockerfile.cli -t dokrypt:$(VERSION) .

## docker: Alias for docker-build
docker: docker-build

## release: Run GoReleaser in snapshot mode
release:
	goreleaser release --snapshot --clean

## help: Show this help message
help:
	@echo "Dokrypt — Web3-native containerization platform"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | column -t -s ':'
