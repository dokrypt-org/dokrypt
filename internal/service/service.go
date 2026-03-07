package service

import (
	"context"
	"io"
)

type Service interface {
	Name() string
	Type() string // "ipfs", "indexer", "explorer", "oracle", "bridge", "monitoring", "faucet", "custom"

	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
	IsRunning(ctx context.Context) bool
	Health(ctx context.Context) error

	Ports() map[string]int    // name -> host port
	URLs() map[string]string  // name -> URL

	DependsOn() []string // Names of services/chains this depends on

	Logs(ctx context.Context, opts LogOptions) (io.ReadCloser, error)
}

type LogOptions struct {
	Follow     bool
	Tail       string
	Since      string
	Timestamps bool
}

type ServiceStatus struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Status  string            `json:"status"` // "running", "stopped", "unhealthy", "starting"
	Ports   map[string]int    `json:"ports"`
	URLs    map[string]string `json:"urls"`
	Healthy bool              `json:"healthy"`
}
