package service

import (
	"fmt"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
)

type Factory func(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (Service, error)

type Registry struct {
	factories map[string]Factory
}

func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
	}
}

func (r *Registry) Register(serviceType string, factory Factory) {
	r.factories[serviceType] = factory
}

func (r *Registry) Create(name string, cfg config.ServiceConfig, runtime container.Runtime, projectName string) (Service, error) {
	factory, ok := r.factories[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unknown service type %q for service %q", cfg.Type, name)
	}
	return factory(name, cfg, runtime, projectName)
}

func (r *Registry) HasType(serviceType string) bool {
	_, ok := r.factories[serviceType]
	return ok
}

func (r *Registry) Types() []string {
	types := make([]string, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	return types
}
