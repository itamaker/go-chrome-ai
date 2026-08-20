//go:build darwin

package chrome

import (
	"fmt"
	"os/exec"
	"strings"
)

// On macOS, Chrome reads user-level managed preferences from the
// `com.google.Chrome` defaults domain. Writing there with `defaults` is the
// supported user-mode way to set policies without an MDM profile.
const macChromeDefaultsDomain = "com.google.Chrome"

func policyStorageDescription(applying bool) string {
	if applying {
		return "macOS: defaults write " + macChromeDefaultsDomain + " " + GenAIPolicyName + " -int 1"
	}
	return "macOS: defaults delete " + macChromeDefaultsDomain + " " + GenAIPolicyName
}

func disableAIDownloadPolicyBackend() policyBackend {
	location := fmt.Sprintf("defaults domain %s (%s)", macChromeDefaultsDomain, GenAIPolicyName)

	return policyBackend{
		location: location,
		isSet: func() (bool, error) {
			value, ok, err := readMacPolicy()
			if err != nil {
				return false, err
			}
			return ok && value == "1", nil
		},
		exists: func() (bool, error) {
			_, ok, err := readMacPolicy()
			return ok, err
		},
		write: func() (string, error) {
			cmd := exec.Command("defaults", "write", macChromeDefaultsDomain, GenAIPolicyName, "-int", "1")
			if out, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("defaults write failed: %w: %s", err, strings.TrimSpace(string(out)))
			}
			return location, nil
		},
		clear: func() (string, error) {
			cmd := exec.Command("defaults", "delete", macChromeDefaultsDomain, GenAIPolicyName)
			if out, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("defaults delete failed: %w: %s", err, strings.TrimSpace(string(out)))
			}
			return location, nil
		},
	}
}

// readMacPolicy reads the current value of GenAIPolicyName from the
// defaults domain. ok is false (with a nil error) when the key is simply
// not set; a non-nil error means the read itself failed for some other
// reason (e.g. the `defaults` binary missing) and must not be treated as
// "not set" — conflating the two previously reported real failures to the
// user as a misleading "already absent".
func readMacPolicy() (value string, ok bool, err error) {
	out, err := exec.Command("defaults", "read", macChromeDefaultsDomain, GenAIPolicyName).Output()
	if err != nil {
		if exitErr, isExit := err.(*exec.ExitError); isExit {
			// `defaults read` exits non-zero and writes "...does not
			// exist" to stderr when the domain/key is absent — that's
			// "not set", not a failure to determine state.
			if strings.Contains(string(exitErr.Stderr), "does not exist") {
				return "", false, nil
			}
		}
		return "", false, err
	}
	return strings.TrimSpace(string(out)), true, nil
}
