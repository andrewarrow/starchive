//go:build windows

package util

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Usage returns total, used, and free bytes for the drive containing path.
// Example path: "C:\\"
func Usage(path string) (total, used, free uint64, err error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetDiskFreeSpaceExW")

	lpDirectoryName, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return
	}

	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	r1, _, e1 := proc.Call(
		uintptr(unsafe.Pointer(lpDirectoryName)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)
	if r1 == 0 {
		if e1 != syscall.Errno(0) {
			err = e1
		} else {
			err = syscall.EINVAL
		}
		return
	}

	total = totalNumberOfBytes
	free = freeBytesAvailable
	used = total - free
	return
}

func Pretty(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPEZY"[exp])
}
