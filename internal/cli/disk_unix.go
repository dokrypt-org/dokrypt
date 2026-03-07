//go:build !windows

package cli

import "syscall"

func freeDiskSpaceGB() (int, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(".", &stat); err != nil {
		return 0, err
	}
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	return int(freeBytes / (1024 * 1024 * 1024)), nil
}
