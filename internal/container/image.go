package container

import (
	"context"
	"fmt"
	"log/slog"
)

type PullOptions struct {
	Force      bool                            // Pull even if image exists locally
	OnProgress func(status string, pct int)    // Progress callback
}

type ImageManager struct {
	runtime Runtime
}

func NewImageManager(rt Runtime) *ImageManager {
	return &ImageManager{runtime: rt}
}

func (m *ImageManager) Pull(ctx context.Context, ref string, opts PullOptions) error {
	if !opts.Force {
		exists, err := m.Exists(ctx, ref)
		if err != nil {
			slog.Warn("failed to check image existence, pulling anyway", "image", ref, "error", err)
		} else if exists {
			slog.Debug("image already exists locally", "image", ref)
			return nil
		}
	}
	slog.Info("pulling image", "image", ref)
	if opts.OnProgress != nil {
		opts.OnProgress("pulling "+ref, 0)
	}
	if err := m.runtime.PullImage(ctx, ref); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", ref, err)
	}
	if opts.OnProgress != nil {
		opts.OnProgress("pulled "+ref, 100)
	}
	return nil
}

func (m *ImageManager) Build(ctx context.Context, contextPath string, opts BuildOptions) (string, error) {
	slog.Info("building image", "context", contextPath, "tags", opts.Tags)
	return m.runtime.BuildImage(ctx, contextPath, opts)
}

func (m *ImageManager) List(ctx context.Context, filter string) ([]ImageInfo, error) {
	images, err := m.runtime.ListImages(ctx)
	if err != nil {
		return nil, err
	}
	if filter == "" {
		return images, nil
	}
	var filtered []ImageInfo
	for _, img := range images {
		for _, tag := range img.Tags {
			if len(tag) >= len(filter) && tag[:len(filter)] == filter {
				filtered = append(filtered, img)
				break
			}
		}
	}
	return filtered, nil
}

func (m *ImageManager) Remove(ctx context.Context, ref string, force bool) error {
	return m.runtime.RemoveImage(ctx, ref, force)
}

func (m *ImageManager) Exists(ctx context.Context, ref string) (bool, error) {
	images, err := m.runtime.ListImages(ctx)
	if err != nil {
		return false, err
	}
	for _, img := range images {
		if img.ID == ref {
			return true, nil
		}
		for _, tag := range img.Tags {
			if tag == ref {
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *ImageManager) PullParallel(ctx context.Context, refs []string, opts PullOptions) error {
	errs := make(chan error, len(refs))
	for _, ref := range refs {
		go func(r string) {
			errs <- m.Pull(ctx, r, opts)
		}(ref)
	}
	var firstErr error
	for range refs {
		if err := <-errs; err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
