//go:build !darwin && !linux && !windows

package chrome

import (
	"fmt"
	"runtime"
)

// This platform has no managed-policy storage mechanism implemented.
// Keeping this fallback — rather than leaving internal/chrome uncompilable
// here — means the rest of the package (profile detection, Local State
// patching, process management) still builds and works on any Go-supported
// OS; only the Enterprise-policy feature reports itself unavailable.
func policyStorageDescription(applying bool) string {
	verb := "write"
	if !applying {
		verb = "remove"
	}
	return fmt.Sprintf("(no support to %s the managed policy on %s)", verb, runtime.GOOS)
}

func disableAIDownloadPolicyBackend() policyBackend {
	err := fmt.Errorf("disabling the %s Enterprise policy is not supported on %s", GenAIPolicyName, runtime.GOOS)
	location := policyStorageDescription(true)
	return policyBackend{
		location: location,
		isSet:    func() (bool, error) { return false, err },
		exists:   func() (bool, error) { return false, err },
		write:    func() (string, error) { return "", err },
		clear:    func() (string, error) { return "", err },
	}
}
