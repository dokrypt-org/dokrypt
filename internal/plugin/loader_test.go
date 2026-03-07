package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	l := NewLoader(m)
	assert.NotNil(t, l)
	assert.Equal(t, m, l.manager)
}

func TestLoader_Load_Container(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	l := NewLoader(m)

	info := &Info{
		Manifest: Manifest{
			Name:    "ctr-plugin",
			Version: "1.0.0",
			Type:    "container",
			Container: ContainerConfig{
				Image: "my-image:v1",
			},
		},
		Path: "/plugins/ctr-plugin",
	}

	p, err := l.Load(info)
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "ctr-plugin", p.Name())
	assert.Equal(t, "1.0.0", p.Version())

	_, ok := p.(*containerPlugin)
	assert.True(t, ok)
}

func TestLoader_Load_Binary(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	l := NewLoader(m)

	info := &Info{
		Manifest: Manifest{
			Name:    "bin-plugin",
			Version: "0.5.0",
			Type:    "binary",
		},
		Path: "/plugins/bin-plugin",
	}

	p, err := l.Load(info)
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "bin-plugin", p.Name())

	_, ok := p.(*binaryPlugin)
	assert.True(t, ok)
}

func TestLoader_Load_Library(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	l := NewLoader(m)

	info := &Info{
		Manifest: Manifest{
			Name:    "lib-plugin",
			Version: "1.2.3",
			Type:    "library",
		},
		Path: "/plugins/lib-plugin",
	}

	p, err := l.Load(info)
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "lib-plugin", p.Name())

	_, ok := p.(*binaryPlugin)
	assert.True(t, ok)
}

func TestLoader_Load_UnknownType(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	l := NewLoader(m)

	info := &Info{
		Manifest: Manifest{
			Name: "mystery",
			Type: "wasm",
		},
	}

	p, err := l.Load(info)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "unknown plugin type")
	assert.Contains(t, err.Error(), "wasm")
}

func TestLoader_LoadAll_Empty(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	l := NewLoader(m)

	loaded, err := l.LoadAll()
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestLoader_LoadAll_MixedTypes(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)

	m.plugins["ctr"] = &Info{
		Manifest: Manifest{Name: "ctr", Type: "container", Container: ContainerConfig{Image: "img:v1"}},
		Path:     "/plugins/ctr",
	}
	m.plugins["bin"] = &Info{
		Manifest: Manifest{Name: "bin", Type: "binary"},
		Path:     "/plugins/bin",
	}

	l := NewLoader(m)
	loaded, err := l.LoadAll()
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
	assert.Len(t, m.loaded, 2)
}

func TestLoader_LoadAll_SkipsUnknown(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)

	m.plugins["good"] = &Info{
		Manifest: Manifest{Name: "good", Type: "binary"},
		Path:     "/plugins/good",
	}
	m.plugins["bad"] = &Info{
		Manifest: Manifest{Name: "bad", Type: "unknown-type"},
		Path:     "/plugins/bad",
	}

	l := NewLoader(m)
	loaded, err := l.LoadAll()
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
	assert.Equal(t, "good", loaded[0].Name())
	assert.Len(t, m.loaded, 1)
}
