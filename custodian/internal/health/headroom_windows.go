//go:build windows

package health

// diskFree is a no-op off the deployment target: custodian ships as a Linux
// binary, so on a Windows dev box the disk check reports full headroom rather
// than reaching for a platform stat the service never uses in production.
func diskFree(string) float64 { return 1 }
