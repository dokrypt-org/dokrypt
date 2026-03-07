package plugin

import (
	"context"

	"github.com/spf13/cobra"
)

type Plugin interface {
	Name() string
	Version() string
	Description() string
	Author() string

	OnInit(ctx context.Context, env Environment) error
	OnUp(ctx context.Context, env Environment) error
	OnDown(ctx context.Context, env Environment) error

	Commands() []*cobra.Command

	Health(ctx context.Context) error
}

type Environment interface {
	ProjectName() string
	ChainRPCURL(name string) string
	ServiceURL(name string) string
}

type Manifest struct {
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description"`
	Author      string            `yaml:"author"`
	License     string            `yaml:"license"`
	Type        string            `yaml:"type"` // "container" or "binary"
	Container   ContainerConfig   `yaml:"container,omitempty"`
	Config      map[string]any    `yaml:"config,omitempty"`
	Hooks       []string          `yaml:"hooks,omitempty"`
	Commands    []CommandDef      `yaml:"commands,omitempty"`
}

type ContainerConfig struct {
	Image       string            `yaml:"image"`
	Ports       map[string]int    `yaml:"ports,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
}

type CommandDef struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Info struct {
	Manifest  Manifest `json:"manifest"`
	Installed bool     `json:"installed"`
	Path      string   `json:"path"`
	Global    bool     `json:"global"`
}
