package plugin

import (
	"context"
	"fmt"
	"log/slog"
)

type HookType string

const (
	HookOnInit             HookType = "on_init"
	HookOnUp               HookType = "on_up"
	HookOnDown             HookType = "on_down"
	HookOnTransaction      HookType = "on_transaction"
	HookOnBlockMined       HookType = "on_block_mined"
	HookOnContractDeployed HookType = "on_contract_deployed"
	HookOnTestEnd          HookType = "on_test_end"
)

type HookEvent struct {
	Hook HookType
	Env  Environment
	Data map[string]any // arbitrary key-value data for the hook
}

type HookDispatcher struct {
	manager *Manager
}

func NewHookDispatcher(manager *Manager) *HookDispatcher {
	return &HookDispatcher{manager: manager}
}

func (d *HookDispatcher) Dispatch(ctx context.Context, event HookEvent) {
	for name, p := range d.manager.loaded {
		if !d.pluginSubscribes(name, event.Hook) {
			continue
		}
		d.invokeHook(ctx, name, p, event)
	}
}

func (d *HookDispatcher) DispatchHook(ctx context.Context, hook HookType, env Environment) {
	d.Dispatch(ctx, HookEvent{
		Hook: hook,
		Env:  env,
	})
}

func (d *HookDispatcher) pluginSubscribes(name string, hook HookType) bool {
	info, ok := d.manager.plugins[name]
	if !ok {
		return false
	}
	if len(info.Manifest.Hooks) == 0 {
		return true
	}
	for _, h := range info.Manifest.Hooks {
		if h == string(hook) {
			return true
		}
	}
	return false
}

func (d *HookDispatcher) invokeHook(ctx context.Context, name string, p Plugin, event HookEvent) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("plugin panicked during hook", "plugin", name, "hook", event.Hook, "panic", fmt.Sprint(r))
		}
	}()

	var err error
	switch event.Hook {
	case HookOnInit:
		err = p.OnInit(ctx, event.Env)
	case HookOnUp:
		err = p.OnUp(ctx, event.Env)
	case HookOnDown:
		err = p.OnDown(ctx, event.Env)
	case HookOnTransaction, HookOnBlockMined, HookOnContractDeployed, HookOnTestEnd:
		if cp, ok := p.(*containerPlugin); ok {
			err = cp.runHook(ctx, string(event.Hook), event.Env)
		} else {
			slog.Debug("extended hook not supported for non-container plugin", "hook", event.Hook, "plugin", name)
		}
	default:
		slog.Warn("unknown hook type", "hook", event.Hook, "plugin", name)
		return
	}

	if err != nil {
		slog.Warn("plugin hook failed", "plugin", name, "hook", event.Hook, "error", err)
	}
}
