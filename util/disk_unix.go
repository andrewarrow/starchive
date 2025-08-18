//go:build unix

package util

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// Usage returns total, used, and free bytes for the filesystem at path.
func Usage(path string) (total, used, free uint64, err error) {
	var st unix.Statfs_t
	if err = unix.Statfs(path, &st); err != nil {
		return
	}

	// Note: On some Linux systems Frsize is the "fundamental" block size.
	// unix.Statfs_t on Linux includes Frsize; on other UNIXes it doesn't.
	// To keep this file portable across UNIXes, we stick to Bsize here.
	bsize := uint64(st.Bsize)

	total = bsize * uint64(st.Blocks)
	free = bsize * uint64(st.Bavail) // space available to unprivileged user
	used = total - free
	return
}

// Pretty is optional: formats bytes as a human-friendly string.
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
