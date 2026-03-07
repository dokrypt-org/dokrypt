package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type fakePlugin struct {
	name    string
	version string
	desc    string
	author  string

	initCalled bool
	upCalled   bool
	downCalled bool
	initErr    error
	upErr      error
	downErr    error
	panicOnUp  bool
}

func (f *fakePlugin) Name() string        { return f.name }
func (f *fakePlugin) Version() string     { return f.version }
func (f *fakePlugin) Description() string { return f.desc }
func (f *fakePlugin) Author() string      { return f.author }

func (f *fakePlugin) OnInit(ctx context.Context, env Environment) error {
	f.initCalled = true
	return f.initErr
}

func (f *fakePlugin) OnUp(ctx context.Context, env Environment) error {
	if f.panicOnUp {
		panic("intentional panic in OnUp")
	}
	f.upCalled = true
	return f.upErr
}

func (f *fakePlugin) OnDown(ctx context.Context, env Environment) error {
	f.downCalled = true
	return f.downErr
}

func (f *fakePlugin) Commands() []*cobra.Command { return nil }
func (f *fakePlugin) Health(ctx context.Context) error { return nil }

func TestHookType_Values(t *testing.T) {
	assert.Equal(t, HookType("on_init"), HookOnInit)
	assert.Equal(t, HookType("on_up"), HookOnUp)
	assert.Equal(t, HookType("on_down"), HookOnDown)
	assert.Equal(t, HookType("on_transaction"), HookOnTransaction)
	assert.Equal(t, HookType("on_block_mined"), HookOnBlockMined)
	assert.Equal(t, HookType("on_contract_deployed"), HookOnContractDeployed)
	assert.Equal(t, HookType("on_test_end"), HookOnTestEnd)
}

func TestHookEvent_Fields(t *testing.T) {
	env := newTestEnv()
	e := HookEvent{
		Hook: HookOnInit,
		Env:  env,
		Data: map[string]any{"tx": "0xabc"},
	}
	assert.Equal(t, HookOnInit, e.Hook)
	assert.Equal(t, env, e.Env)
	assert.Equal(t, "0xabc", e.Data["tx"])
}

func TestNewHookDispatcher(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	d := NewHookDispatcher(m)
	assert.NotNil(t, d)
	assert.Equal(t, m, d.manager)
}

func TestPluginSubscribes_NoHooks_AcceptsAll(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	m.plugins["p1"] = &Info{Manifest: Manifest{Name: "p1", Hooks: nil}}

	d := NewHookDispatcher(m)
	assert.True(t, d.pluginSubscribes("p1", HookOnInit))
	assert.True(t, d.pluginSubscribes("p1", HookOnUp))
	assert.True(t, d.pluginSubscribes("p1", HookOnDown))
	assert.True(t, d.pluginSubscribes("p1", HookOnTransaction))
}

func TestPluginSubscribes_SpecificHooks(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	m.plugins["p1"] = &Info{Manifest: Manifest{Name: "p1", Hooks: []string{"on_init", "on_up"}}}

	d := NewHookDispatcher(m)
	assert.True(t, d.pluginSubscribes("p1", HookOnInit))
	assert.True(t, d.pluginSubscribes("p1", HookOnUp))
	assert.False(t, d.pluginSubscribes("p1", HookOnDown))
	assert.False(t, d.pluginSubscribes("p1", HookOnTransaction))
}

func TestPluginSubscribes_UnknownPlugin(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	d := NewHookDispatcher(m)
	assert.False(t, d.pluginSubscribes("nonexistent", HookOnInit))
}

func TestDispatch_OnInit(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	fp := &fakePlugin{name: "fp1"}
	m.plugins["fp1"] = &Info{Manifest: Manifest{Name: "fp1"}}
	m.loaded["fp1"] = fp

	d := NewHookDispatcher(m)
	d.Dispatch(context.Background(), HookEvent{Hook: HookOnInit, Env: newTestEnv()})

	assert.True(t, fp.initCalled)
	assert.False(t, fp.upCalled)
	assert.False(t, fp.downCalled)
}

func TestDispatch_OnUp(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	fp := &fakePlugin{name: "fp1"}
	m.plugins["fp1"] = &Info{Manifest: Manifest{Name: "fp1"}}
	m.loaded["fp1"] = fp

	d := NewHookDispatcher(m)
	d.Dispatch(context.Background(), HookEvent{Hook: HookOnUp, Env: newTestEnv()})

	assert.True(t, fp.upCalled)
}

func TestDispatch_OnDown(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	fp := &fakePlugin{name: "fp1"}
	m.plugins["fp1"] = &Info{Manifest: Manifest{Name: "fp1"}}
	m.loaded["fp1"] = fp

	d := NewHookDispatcher(m)
	d.Dispatch(context.Background(), HookEvent{Hook: HookOnDown, Env: newTestEnv()})

	assert.True(t, fp.downCalled)
}

func TestDispatch_ErrorDoesNotPropagate(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	fp := &fakePlugin{name: "fp1", initErr: errors.New("init failed")}
	m.plugins["fp1"] = &Info{Manifest: Manifest{Name: "fp1"}}
	m.loaded["fp1"] = fp

	d := NewHookDispatcher(m)
	d.Dispatch(context.Background(), HookEvent{Hook: HookOnInit, Env: newTestEnv()})

	assert.True(t, fp.initCalled)
}

func TestDispatch_PanicRecovery(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	fp := &fakePlugin{name: "fp1", panicOnUp: true}
	m.plugins["fp1"] = &Info{Manifest: Manifest{Name: "fp1"}}
	m.loaded["fp1"] = fp

	d := NewHookDispatcher(m)
	assert.NotPanics(t, func() {
		d.Dispatch(context.Background(), HookEvent{Hook: HookOnUp, Env: newTestEnv()})
	})
}

func TestDispatch_SkipsNonSubscribers(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)

	fpSub := &fakePlugin{name: "sub"}
	fpNoSub := &fakePlugin{name: "nosub"}

	m.plugins["sub"] = &Info{Manifest: Manifest{Name: "sub", Hooks: []string{"on_init"}}}
	m.plugins["nosub"] = &Info{Manifest: Manifest{Name: "nosub", Hooks: []string{"on_up"}}}
	m.loaded["sub"] = fpSub
	m.loaded["nosub"] = fpNoSub

	d := NewHookDispatcher(m)
	d.Dispatch(context.Background(), HookEvent{Hook: HookOnInit, Env: newTestEnv()})

	assert.True(t, fpSub.initCalled)
	assert.False(t, fpNoSub.initCalled)
}

func TestDispatch_MultiplePlugins(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)

	fp1 := &fakePlugin{name: "p1"}
	fp2 := &fakePlugin{name: "p2"}

	m.plugins["p1"] = &Info{Manifest: Manifest{Name: "p1"}}
	m.plugins["p2"] = &Info{Manifest: Manifest{Name: "p2"}}
	m.loaded["p1"] = fp1
	m.loaded["p2"] = fp2

	d := NewHookDispatcher(m)
	d.Dispatch(context.Background(), HookEvent{Hook: HookOnInit, Env: newTestEnv()})

	assert.True(t, fp1.initCalled)
	assert.True(t, fp2.initCalled)
}

func TestDispatch_NoLoadedPlugins(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	d := NewHookDispatcher(m)
	d.Dispatch(context.Background(), HookEvent{Hook: HookOnInit, Env: newTestEnv()})
}

func TestDispatch_UnknownHookType(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	fp := &fakePlugin{name: "p1"}
	m.plugins["p1"] = &Info{Manifest: Manifest{Name: "p1"}}
	m.loaded["p1"] = fp

	d := NewHookDispatcher(m)
	d.Dispatch(context.Background(), HookEvent{Hook: HookType("on_weird"), Env: newTestEnv()})

	assert.False(t, fp.initCalled)
	assert.False(t, fp.upCalled)
	assert.False(t, fp.downCalled)
}

func TestDispatchHook(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	fp := &fakePlugin{name: "fp1"}
	m.plugins["fp1"] = &Info{Manifest: Manifest{Name: "fp1"}}
	m.loaded["fp1"] = fp

	d := NewHookDispatcher(m)
	d.DispatchHook(context.Background(), HookOnDown, newTestEnv())

	assert.True(t, fp.downCalled)
}

func TestDispatch_ExtendedHook_NonContainerPlugin(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	fp := &fakePlugin{name: "fp1"}
	m.plugins["fp1"] = &Info{Manifest: Manifest{Name: "fp1"}} // no hooks filter = all
	m.loaded["fp1"] = fp

	d := NewHookDispatcher(m)
	assert.NotPanics(t, func() {
		d.Dispatch(context.Background(), HookEvent{Hook: HookOnTransaction, Env: newTestEnv()})
	})
}

func TestDispatch_ExtendedHook_ContainerNoImage(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	cp := &containerPlugin{
		info: &Info{
			Manifest: Manifest{Name: "ctr", Type: "container"},
		},
	}
	m.plugins["ctr"] = cp.info
	m.loaded["ctr"] = cp

	d := NewHookDispatcher(m)
	assert.NotPanics(t, func() {
		d.Dispatch(context.Background(), HookEvent{Hook: HookOnBlockMined, Env: newTestEnv()})
	})
}
