package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockEnvironment struct {
	project  string
	chainURL string
	svcURL   string
}

func (m *mockEnvironment) ProjectName() string          { return m.project }
func (m *mockEnvironment) ChainRPCURL(name string) string { return m.chainURL }
func (m *mockEnvironment) ServiceURL(name string) string  { return m.svcURL }

func newTestEnv() *mockEnvironment {
	return &mockEnvironment{
		project:  "test-project",
		chainURL: "http://localhost:8545",
		svcURL:   "http://localhost:3000",
	}
}

func TestManifest_ZeroValue(t *testing.T) {
	var m Manifest
	assert.Empty(t, m.Name)
	assert.Empty(t, m.Version)
	assert.Empty(t, m.Description)
	assert.Empty(t, m.Author)
	assert.Empty(t, m.License)
	assert.Empty(t, m.Type)
	assert.Empty(t, m.Hooks)
	assert.Empty(t, m.Commands)
	assert.Nil(t, m.Config)
}

func TestManifest_Fields(t *testing.T) {
	m := Manifest{
		Name:        "my-plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Author:      "tester",
		License:     "MIT",
		Type:        "binary",
		Container: ContainerConfig{
			Image: "my-image:latest",
			Ports: map[string]int{"http": 8080},
			Environment: map[string]string{
				"KEY": "value",
			},
			DependsOn: []string{"db"},
		},
		Hooks:    []string{"on_init", "on_up"},
		Commands: []CommandDef{{Name: "run", Description: "Run the plugin"}},
		Config:   map[string]any{"debug": true},
	}

	assert.Equal(t, "my-plugin", m.Name)
	assert.Equal(t, "1.0.0", m.Version)
	assert.Equal(t, "A test plugin", m.Description)
	assert.Equal(t, "tester", m.Author)
	assert.Equal(t, "MIT", m.License)
	assert.Equal(t, "binary", m.Type)
	assert.Equal(t, "my-image:latest", m.Container.Image)
	assert.Equal(t, 8080, m.Container.Ports["http"])
	assert.Equal(t, "value", m.Container.Environment["KEY"])
	assert.Equal(t, []string{"db"}, m.Container.DependsOn)
	assert.Len(t, m.Hooks, 2)
	assert.Len(t, m.Commands, 1)
	assert.Equal(t, "run", m.Commands[0].Name)
	assert.True(t, m.Config["debug"].(bool))
}

func TestContainerConfig_Empty(t *testing.T) {
	var cc ContainerConfig
	assert.Empty(t, cc.Image)
	assert.Nil(t, cc.Ports)
	assert.Nil(t, cc.Environment)
	assert.Nil(t, cc.DependsOn)
}

func TestCommandDef_Fields(t *testing.T) {
	cd := CommandDef{
		Name:        "deploy",
		Description: "Deploy contracts",
	}
	assert.Equal(t, "deploy", cd.Name)
	assert.Equal(t, "Deploy contracts", cd.Description)
}

func TestInfo_Fields(t *testing.T) {
	info := &Info{
		Manifest: Manifest{
			Name:    "test",
			Version: "0.1.0",
		},
		Installed: true,
		Path:      "/tmp/plugins/test",
		Global:    false,
	}
	assert.Equal(t, "test", info.Manifest.Name)
	assert.True(t, info.Installed)
	assert.Equal(t, "/tmp/plugins/test", info.Path)
	assert.False(t, info.Global)
}

func TestMockEnvironment(t *testing.T) {
	env := newTestEnv()
	assert.Equal(t, "test-project", env.ProjectName())
	assert.Equal(t, "http://localhost:8545", env.ChainRPCURL("mainnet"))
	assert.Equal(t, "http://localhost:3000", env.ServiceURL("api"))
}
