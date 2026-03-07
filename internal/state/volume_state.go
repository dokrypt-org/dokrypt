package state

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dokrypt/dokrypt/internal/container"
)

type VolumeStateManager struct {
	store   *Store
	runtime container.Runtime
}

func NewVolumeStateManager(store *Store, runtime container.Runtime) *VolumeStateManager {
	return &VolumeStateManager{store: store, runtime: runtime}
}

func (m *VolumeStateManager) BackupVolume(ctx context.Context, snapshotName string, serviceName string, volumeName string) (*VolumeSnapshot, error) {
	volumesDir := m.store.VolumesDir(snapshotName)
	archiveName := fmt.Sprintf("%s.tar.gz", serviceName)
	archivePath := filepath.Join(volumesDir, archiveName)

	info, err := m.runtime.InspectVolume(ctx, volumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume %s: %w", volumeName, err)
	}

	if err := createTarGz(archivePath, info.Mountpoint); err != nil {
		return nil, fmt.Errorf("failed to backup volume %s: %w", volumeName, err)
	}

	fi, err := os.Stat(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup archive %s: %w", archivePath, err)
	}
	size := fi.Size()

	return &VolumeSnapshot{
		Name:        volumeName,
		Service:     serviceName,
		ArchiveFile: filepath.Join("volumes", archiveName),
		Size:        size,
	}, nil
}

func (m *VolumeStateManager) RestoreVolume(ctx context.Context, snapshotName string, vs VolumeSnapshot) error {
	archivePath := filepath.Join(m.store.SnapshotDir(snapshotName), vs.ArchiveFile)

	info, err := m.runtime.InspectVolume(ctx, vs.Name)
	if err != nil {
		return fmt.Errorf("volume %s not found: %w", vs.Name, err)
	}

	return extractTarGz(archivePath, info.Mountpoint)
}

func createTarGz(archivePath string, sourceDir string) error {
	outFile, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})
}

func extractTarGz(archivePath string, destDir string) error {
	inFile, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	gzReader, err := gzip.NewReader(inFile)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destDir, header.Name)

		if !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(destDir, filepath.Clean(header.Name))
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}
