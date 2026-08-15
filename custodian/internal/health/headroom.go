package health

// minDiskFree is the fraction of the working filesystem that must stay available
// for custodian to call itself healthy. Below it, writes are at risk of failing.
const minDiskFree = 0.05

// localHeadroom reports whether the working filesystem and system memory both
// have room to spare. Both checks are platform-specific: disk in
// headroom_unix.go / headroom_windows.go, memory in headroom_linux.go /
// headroom_other.go. Off the Linux deployment target they degrade to no-ops.
func localHeadroom() bool {
	return diskFree(".") >= minDiskFree && memHeadroom()
}
