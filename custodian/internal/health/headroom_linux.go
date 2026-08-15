//go:build linux

package health

import "golang.org/x/sys/unix"

// minMemFree is the fraction of system RAM that must stay free for custodian to
// call itself healthy on the deployment target.
const minMemFree = 0.05

// memHeadroom reports whether free system memory is above the threshold. A
// kernel that will not answer Sysinfo is treated as headroom present rather than
// flipping the gauge on an unreadable stat.
func memHeadroom() bool {
	var info unix.Sysinfo_t
	if err := unix.Sysinfo(&info); err != nil {
		return true
	}
	if info.Totalram == 0 {
		return true
	}
	return float64(info.Freeram)/float64(info.Totalram) >= minMemFree
}
