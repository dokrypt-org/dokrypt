package container

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

type VolumeCreateOptions struct {
	VolumeOptions
	Project string
	Service string
}

type VolumeManager struct {
	runtime Runtime
}

func NewVolumeManager(rt Runtime) *VolumeManager {
	return &VolumeManager{runtime: rt}
}

func (m *VolumeManager) Create(ctx context.Context, name string, opts VolumeCreateOptions) (string, error) {
	if opts.Labels == nil {
		opts.Labels = make(map[string]string)
	}
	opts.Labels["dokrypt.volume"] = "true"
	if opts.Project != "" {
		opts.Labels["dokrypt.project"] = opts.Project
	}
	if opts.Service != "" {
		opts.Labels["dokrypt.service"] = opts.Service
	}

	slog.Info("creating volume", "name", name)
	return m.runtime.CreateVolume(ctx, name, opts.VolumeOptions)
}

func (m *VolumeManager) Remove(ctx context.Context, name string, force bool) error {
	return m.runtime.RemoveVolume(ctx, name, force)
}

func (m *VolumeManager) List(ctx context.Context, projectFilter string) ([]VolumeInfo, error) {
	volumes, err := m.runtime.ListVolumes(ctx)
	if err != nil {
		return nil, err
	}
	var result []VolumeInfo
	for _, v := range volumes {
		if v.Labels["dokrypt.volume"] != "true" {
			continue
		}
		if projectFilter != "" && v.Labels["dokrypt.project"] != projectFilter {
			continue
		}
		result = append(result, v)
	}
	return result, nil
}

func (m *VolumeManager) Inspect(ctx context.Context, name string) (*VolumeInfo, error) {
	return m.runtime.InspectVolume(ctx, name)
}

func (m *VolumeManager) Export(ctx context.Context, volumeName, targetPath string) error {
	slog.Info("exporting volume", "volume", volumeName, "target", targetPath)

	containerID, err := m.runtime.CreateContainer(ctx, &ContainerConfig{
		Name:    fmt.Sprintf("dokrypt-vol-export-%s", volumeName),
		Image:   "alpine:latest",
		Command: []string{"sleep", "3600"},
		Volumes: []VolumeMount{
			{Source: volumeName, Target: "/data", ReadOnly: true},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create export container: %w", err)
	}
	defer m.runtime.RemoveContainer(ctx, containerID, true)

	if err := m.runtime.StartContainer(ctx, containerID); err != nil {
		return fmt.Errorf("failed to start export container: %w", err)
	}

	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer outFile.Close()

	result, err := m.runtime.ExecInContainer(ctx, containerID, []string{"tar", "cf", "-", "-C", "/data", "."}, ExecOptions{
		Stdout: outFile,
	})
	if err != nil {
		return fmt.Errorf("failed to create export archive: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("tar export failed (exit code %d): %s", result.ExitCode, result.Stderr)
	}
	return nil
}

func (m *VolumeManager) Import(ctx context.Context, volumeName, sourcePath string) error {
	slog.Info("importing volume", "volume", volumeName, "source", sourcePath)

	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source archive %s: %w", sourcePath, err)
	}
	defer srcFile.Close()

	containerID, err := m.runtime.CreateContainer(ctx, &ContainerConfig{
		Name:    fmt.Sprintf("dokrypt-vol-import-%s", volumeName),
		Image:   "alpine:latest",
		Command: []string{"sleep", "3600"},
		Volumes: []VolumeMount{
			{Source: volumeName, Target: "/data"},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create import container: %w", err)
	}
	defer m.runtime.RemoveContainer(ctx, containerID, true)

	if err := m.runtime.StartContainer(ctx, containerID); err != nil {
		return fmt.Errorf("failed to start import container: %w", err)
	}

	result, err := m.runtime.ExecInContainer(ctx, containerID, []string{"tar", "xf", "-", "-C", "/data"}, ExecOptions{
		Stdin: srcFile,
	})
	if err != nil {
		return fmt.Errorf("failed to extract archive into volume: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("tar extraction failed (exit code %d): %s", result.ExitCode, result.Stderr)
	}

	return nil
}

func (m *VolumeManager) Size(ctx context.Context, volumeName string) (int64, error) {
	containerID, err := m.runtime.CreateContainer(ctx, &ContainerConfig{
		Name:    fmt.Sprintf("dokrypt-vol-size-%s", volumeName),
		Image:   "alpine:latest",
		Command: []string{"sleep", "3600"},
		Volumes: []VolumeMount{
			{Source: volumeName, Target: "/data", ReadOnly: true},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create size check container: %w", err)
	}
	defer m.runtime.RemoveContainer(ctx, containerID, true)

	if err := m.runtime.StartContainer(ctx, containerID); err != nil {
		return 0, fmt.Errorf("failed to start size check container: %w", err)
	}

	result, err := m.runtime.ExecInContainer(ctx, containerID, []string{"du", "-sb", "/data"}, ExecOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get volume size: %w", err)
	}
	if result.ExitCode != 0 {
		return 0, fmt.Errorf("du failed (exit code %d): %s", result.ExitCode, result.Stderr)
	}
	var size int64
	if _, err := fmt.Sscanf(result.Stdout, "%d", &size); err != nil {
		return 0, fmt.Errorf("failed to parse size from output %q: %w", result.Stdout, err)
	}
	return size, nil
}
