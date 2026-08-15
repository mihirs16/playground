package health

import "golang.org/x/sys/unix"

// minDiskFree is the fraction of the working filesystem that must stay available
// for custodian to call itself healthy. Below it, writes are at risk of failing.
const minDiskFree = 0.05

// localHeadroom reports whether the working filesystem and system memory both
// have room to spare. Memory is checked only where the platform exposes it
// (see headroom_linux.go); everywhere else the disk check stands alone.
func localHeadroom() bool {
	return diskFree(".") >= minDiskFree && memHeadroom()
}

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
