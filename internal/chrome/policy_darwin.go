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

func policyStorageDescription() string {
	return "macOS: defaults write " + macChromeDefaultsDomain + " " + GenAIPolicyName + " -int 1"
}

func applyDisableAIDownloadPolicy(dryRun bool) (PolicyResult, error) {
	location := fmt.Sprintf("defaults domain %s (%s)", macChromeDefaultsDomain, GenAIPolicyName)

	current, err := readMacPolicy()
	if err == nil && current == "1" {
		return PolicyResult{Applied: false, Location: location, Skipped: "already set to 1"}, nil
	}

	if dryRun {
		return PolicyResult{Applied: true, Location: location}, nil
	}

	cmd := exec.Command("defaults", "write", macChromeDefaultsDomain, GenAIPolicyName, "-int", "1")
	if out, err := cmd.CombinedOutput(); err != nil {
		return PolicyResult{}, fmt.Errorf("defaults write failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return PolicyResult{Applied: true, Location: location}, nil
}

func readMacPolicy() (string, error) {
	out, err := exec.Command("defaults", "read", macChromeDefaultsDomain, GenAIPolicyName).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
