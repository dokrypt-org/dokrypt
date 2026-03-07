package plugin

import (
	"fmt"
	"log/slog"
)

type Loader struct {
	manager *Manager
}

func NewLoader(manager *Manager) *Loader {
	return &Loader{manager: manager}
}

func (l *Loader) LoadAll() ([]Plugin, error) {
	var loaded []Plugin
	for name, info := range l.manager.plugins {
		p, err := l.Load(info)
		if err != nil {
			slog.Warn("failed to load plugin", "plugin", name, "error", err)
			continue
		}
		l.manager.loaded[name] = p
		loaded = append(loaded, p)
		slog.Info("loaded plugin", "plugin", name, "type", info.Manifest.Type)
	}
	return loaded, nil
}

func (l *Loader) Load(info *Info) (Plugin, error) {
	switch info.Manifest.Type {
	case "container":
		return l.loadContainer(info)
	case "binary":
		return l.loadBinary(info)
	case "library":
		return l.loadLibrary(info)
	default:
		return nil, fmt.Errorf("unknown plugin type %q for plugin %q", info.Manifest.Type, info.Manifest.Name)
	}
}

func (l *Loader) loadContainer(info *Info) (Plugin, error) {
	slog.Debug("loading container plugin", "plugin", info.Manifest.Name, "image", info.Manifest.Container.Image)
	return &containerPlugin{info: info}, nil
}

func (l *Loader) loadBinary(info *Info) (Plugin, error) {
	slog.Debug("loading binary plugin", "plugin", info.Manifest.Name, "path", info.Path)
	return &binaryPlugin{info: info}, nil
}

func (l *Loader) loadLibrary(info *Info) (Plugin, error) {
	slog.Debug("loading library plugin", "plugin", info.Manifest.Name, "path", info.Path)
	return &binaryPlugin{info: info}, nil
}
