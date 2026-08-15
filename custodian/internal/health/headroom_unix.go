//go:build unix

package health

import "golang.org/x/sys/unix"

// diskFree returns the fraction of path's filesystem available to an
// unprivileged process, or 0 when the filesystem cannot be read.
func diskFree(path string) float64 {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0
	}
	if stat.Blocks == 0 {
		return 0
	}
	return float64(stat.Bavail) / float64(stat.Blocks)
}
