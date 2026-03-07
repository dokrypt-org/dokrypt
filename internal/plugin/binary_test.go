package plugin

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newBinaryPlugin(name, version, desc, author, basePath string) *binaryPlugin {
	return &binaryPlugin{
		info: &Info{
			Manifest: Manifest{
				Name:        name,
				Version:     version,
				Description: desc,
				Author:      author,
				Type:        "binary",
			},
			Path: basePath,
		},
	}
}

func TestBinaryPlugin_Name(t *testing.T) {
	p := newBinaryPlugin("my-bin", "1.0.0", "desc", "alice", "/plugins/my-bin")
	assert.Equal(t, "my-bin", p.Name())
}

func TestBinaryPlugin_Version(t *testing.T) {
	p := newBinaryPlugin("my-bin", "2.3.4", "desc", "alice", "/plugins/my-bin")
	assert.Equal(t, "2.3.4", p.Version())
}

func TestBinaryPlugin_Description(t *testing.T) {
	p := newBinaryPlugin("my-bin", "1.0.0", "A binary plugin", "alice", "/plugins/my-bin")
	assert.Equal(t, "A binary plugin", p.Description())
}

func TestBinaryPlugin_Author(t *testing.T) {
	p := newBinaryPlugin("my-bin", "1.0.0", "desc", "bob", "/plugins/my-bin")
	assert.Equal(t, "bob", p.Author())
}

func TestBinaryPlugin_BinaryPath(t *testing.T) {
	p := newBinaryPlugin("my-bin", "1.0.0", "desc", "alice", "/plugins/my-bin")
	expected := filepath.Join("/plugins/my-bin", "my-bin")
	assert.Equal(t, expected, p.binaryPath())
}

func TestBinaryPlugin_BinaryPath_NestedDir(t *testing.T) {
	p := newBinaryPlugin("tool", "0.1.0", "", "", "/home/user/.dokrypt/plugins/tool")
	expected := filepath.Join("/home/user/.dokrypt/plugins/tool", "tool")
	assert.Equal(t, expected, p.binaryPath())
}

func TestBinaryPlugin_Commands_Empty(t *testing.T) {
	p := newBinaryPlugin("my-bin", "1.0.0", "desc", "alice", "/plugins/my-bin")
	cmds := p.Commands()
	assert.Empty(t, cmds)
}

func TestBinaryPlugin_Commands_Single(t *testing.T) {
	p := &binaryPlugin{
		info: &Info{
			Manifest: Manifest{
				Name: "my-bin",
				Type: "binary",
				Commands: []CommandDef{
					{Name: "deploy", Description: "Deploy contracts"},
				},
			},
			Path: "/plugins/my-bin",
		},
	}
	cmds := p.Commands()
	assert.Len(t, cmds, 1)
	assert.Equal(t, "deploy", cmds[0].Use)
	assert.Equal(t, "Deploy contracts", cmds[0].Short)
}

func TestBinaryPlugin_Commands_Multiple(t *testing.T) {
	p := &binaryPlugin{
		info: &Info{
			Manifest: Manifest{
				Name: "multi",
				Type: "binary",
				Commands: []CommandDef{
					{Name: "cmd1", Description: "First"},
					{Name: "cmd2", Description: "Second"},
					{Name: "cmd3", Description: "Third"},
				},
			},
			Path: "/plugins/multi",
		},
	}
	cmds := p.Commands()
	assert.Len(t, cmds, 3)
	assert.Equal(t, "cmd1", cmds[0].Use)
	assert.Equal(t, "cmd2", cmds[1].Use)
	assert.Equal(t, "cmd3", cmds[2].Use)
}

func TestBinaryPlugin_ImplementsPlugin(t *testing.T) {
	var _ Plugin = (*binaryPlugin)(nil)
}
