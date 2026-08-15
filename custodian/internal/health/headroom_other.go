//go:build !linux

package health

// memHeadroom is a no-op off the deployment target: only Linux exposes the
// system-memory stat custodian trusts, so elsewhere the disk check stands alone.
func memHeadroom() bool { return true }
