package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type Orchestrator struct {
	services map[string]Service
	mu       sync.RWMutex
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		services: make(map[string]Service),
	}
}

func (o *Orchestrator) Register(svc Service) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.services[svc.Name()] = svc
}

func (o *Orchestrator) Get(name string) (Service, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	svc, ok := o.services[name]
	if !ok {
		return nil, fmt.Errorf("service %q not found", name)
	}
	return svc, nil
}

func (o *Orchestrator) All() []Service {
	o.mu.RLock()
	defer o.mu.RUnlock()
	svcs := make([]Service, 0, len(o.services))
	for _, svc := range o.services {
		svcs = append(svcs, svc)
	}
	return svcs
}

func (o *Orchestrator) StartAll(ctx context.Context) error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	graph := NewDependencyGraph()
	for name, svc := range o.services {
		graph.AddNode(name, svc.DependsOn())
	}

	groups, err := graph.IndependentGroups()
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	for i, group := range groups {
		slog.Info("starting service group", "group", i+1, "services", group)

		var wg sync.WaitGroup
		errCh := make(chan error, len(group))

		for _, name := range group {
			svc := o.services[name]
			wg.Add(1)
			go func() {
				defer wg.Done()
				slog.Info("starting service", "service", svc.Name(), "type", svc.Type())
				if err := svc.Start(ctx); err != nil {
					errCh <- fmt.Errorf("failed to start service %s: %w", svc.Name(), err)
					return
				}
				slog.Info("service started", "service", svc.Name())
			}()
		}

		wg.Wait()
		close(errCh)

		var errs []error
		for err := range errCh {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			for _, err := range errs {
				slog.Error("service start failed", "error", err)
			}
			return errs[0]
		}
	}

	return nil
}

func (o *Orchestrator) StopAll(ctx context.Context) error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	graph := NewDependencyGraph()
	for name, svc := range o.services {
		graph.AddNode(name, svc.DependsOn())
	}

	order, err := graph.ReverseOrder()
	if err != nil {
		slog.Warn("failed to resolve shutdown order, stopping all services", "error", err)
		for _, svc := range o.services {
			if stopErr := svc.Stop(ctx); stopErr != nil {
				slog.Error("failed to stop service", "service", svc.Name(), "error", stopErr)
			}
		}
		return nil
	}

	for _, name := range order {
		svc := o.services[name]
		slog.Info("stopping service", "service", name)
		if err := svc.Stop(ctx); err != nil {
			slog.Error("failed to stop service", "service", name, "error", err)
		}
	}

	return nil
}

func (o *Orchestrator) StartService(ctx context.Context, name string) error {
	o.mu.RLock()
	defer o.mu.RUnlock()
	svc, ok := o.services[name]
	if !ok {
		return fmt.Errorf("service %q not found", name)
	}
	return svc.Start(ctx)
}

func (o *Orchestrator) StopService(ctx context.Context, name string) error {
	o.mu.RLock()
	defer o.mu.RUnlock()
	svc, ok := o.services[name]
	if !ok {
		return fmt.Errorf("service %q not found", name)
	}
	return svc.Stop(ctx)
}

func (o *Orchestrator) Status(ctx context.Context) []ServiceStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var statuses []ServiceStatus
	for _, svc := range o.services {
		status := "stopped"
		healthy := false
		if svc.IsRunning(ctx) {
			status = "running"
			if svc.Health(ctx) == nil {
				healthy = true
			} else {
				status = "unhealthy"
			}
		}
		statuses = append(statuses, ServiceStatus{
			Name:    svc.Name(),
			Type:    svc.Type(),
			Status:  status,
			Ports:   svc.Ports(),
			URLs:    svc.URLs(),
			Healthy: healthy,
		})
	}
	return statuses
}
