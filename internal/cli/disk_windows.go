//go:build windows

package cli

import (
	"syscall"
	"unsafe"
)

func freeDiskSpaceGB() (int, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpace := kernel32.NewProc("GetDiskFreeSpaceExW")

	var freeBytesAvailable uint64
	path, _ := syscall.UTF16PtrFromString(".")
	ret, _, err := getDiskFreeSpace.Call(
		uintptr(unsafe.Pointer(path)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		0,
		0,
	)
	if ret == 0 {
		return 0, err
	}
	return int(freeBytesAvailable / (1024 * 1024 * 1024)), nil
}
