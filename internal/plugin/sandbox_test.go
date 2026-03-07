package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newContainerPlugin(name, version, desc, author, image string) *containerPlugin {
	return &containerPlugin{
		info: &Info{
			Manifest: Manifest{
				Name:        name,
				Version:     version,
				Description: desc,
				Author:      author,
				Type:        "container",
				Container: ContainerConfig{
					Image: image,
				},
			},
			Path: "/plugins/" + name,
		},
	}
}

func TestContainerPlugin_Name(t *testing.T) {
	p := newContainerPlugin("my-ctr", "1.0.0", "desc", "alice", "img:latest")
	assert.Equal(t, "my-ctr", p.Name())
}

func TestContainerPlugin_Version(t *testing.T) {
	p := newContainerPlugin("my-ctr", "2.0.0", "desc", "alice", "img:latest")
	assert.Equal(t, "2.0.0", p.Version())
}

func TestContainerPlugin_Description(t *testing.T) {
	p := newContainerPlugin("my-ctr", "1.0.0", "A container plugin", "alice", "img:latest")
	assert.Equal(t, "A container plugin", p.Description())
}

func TestContainerPlugin_Author(t *testing.T) {
	p := newContainerPlugin("my-ctr", "1.0.0", "desc", "bob", "img:latest")
	assert.Equal(t, "bob", p.Author())
}

func TestContainerPlugin_Image(t *testing.T) {
	p := newContainerPlugin("ctr", "1.0.0", "", "", "my-image:v1")
	assert.Equal(t, "my-image:v1", p.image())
}

func TestContainerPlugin_Image_Empty(t *testing.T) {
	p := newContainerPlugin("ctr", "1.0.0", "", "", "")
	assert.Empty(t, p.image())
}

func TestContainerPlugin_RunHook_NoImage(t *testing.T) {
	p := newContainerPlugin("ctr", "1.0.0", "", "", "")
	env := newTestEnv()
	err := p.runHook(context.Background(), "init", env)
	assert.NoError(t, err)
}

func TestContainerPlugin_Health_NoImage(t *testing.T) {
	p := newContainerPlugin("ctr", "1.0.0", "", "", "")
	err := p.Health(context.Background())
	assert.NoError(t, err)
}

func TestContainerPlugin_Commands_Empty(t *testing.T) {
	p := newContainerPlugin("ctr", "1.0.0", "", "", "img:latest")
	cmds := p.Commands()
	assert.Empty(t, cmds)
}

func TestContainerPlugin_Commands_Multiple(t *testing.T) {
	p := &containerPlugin{
		info: &Info{
			Manifest: Manifest{
				Name: "ctr",
				Type: "container",
				Container: ContainerConfig{
					Image: "img:latest",
				},
				Commands: []CommandDef{
					{Name: "analyze", Description: "Analyze contracts"},
					{Name: "report", Description: "Generate report"},
				},
			},
			Path: "/plugins/ctr",
		},
	}
	cmds := p.Commands()
	assert.Len(t, cmds, 2)
	assert.Equal(t, "analyze", cmds[0].Use)
	assert.Equal(t, "Analyze contracts", cmds[0].Short)
	assert.Equal(t, "report", cmds[1].Use)
	assert.Equal(t, "Generate report", cmds[1].Short)
}

func TestContainerPlugin_OnInit_NoImage(t *testing.T) {
	p := newContainerPlugin("ctr", "1.0.0", "", "", "")
	err := p.OnInit(context.Background(), newTestEnv())
	assert.NoError(t, err)
}

func TestContainerPlugin_OnUp_NoImage(t *testing.T) {
	p := newContainerPlugin("ctr", "1.0.0", "", "", "")
	err := p.OnUp(context.Background(), newTestEnv())
	assert.NoError(t, err)
}

func TestContainerPlugin_OnDown_NoImage(t *testing.T) {
	p := newContainerPlugin("ctr", "1.0.0", "", "", "")
	err := p.OnDown(context.Background(), newTestEnv())
	assert.NoError(t, err)
}

func TestContainerPlugin_ImplementsPlugin(t *testing.T) {
	var _ Plugin = (*containerPlugin)(nil)
}
