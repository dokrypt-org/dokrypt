package types

import "time"

type EnvironmentConfig struct {
	Name     string                 `json:"name" yaml:"name"`
	Settings SettingsConfig         `json:"settings" yaml:"settings"`
	Chains   map[string]ChainConfig `json:"chains" yaml:"chains"`
	Services map[string]ServiceConfig `json:"services,omitempty" yaml:"services,omitempty"`
}

type SettingsConfig struct {
	Runtime        string `json:"runtime" yaml:"runtime"`
	LogLevel       string `json:"log_level" yaml:"log_level"`
	AccountBalance string `json:"account_balance,omitempty" yaml:"balance,omitempty"`
}

type ChainConfig struct {
	Engine    string `json:"engine" yaml:"engine"`
	ChainID   uint64 `json:"chain_id" yaml:"chain_id"`
	BlockTime string `json:"block_time,omitempty" yaml:"block_time,omitempty"`
	Accounts  int    `json:"accounts,omitempty" yaml:"accounts,omitempty"`
}

type ServiceConfig struct {
	Type      string `json:"type" yaml:"type"`
	Enabled   bool   `json:"enabled" yaml:"enabled"`
	Image     string `json:"image,omitempty" yaml:"image,omitempty"`
}

type Environment struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	Region       string            `json:"region"`
	RPCEndpoints map[string]string `json:"rpc_endpoints,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type Snapshot struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}
