//go:build windows

package chrome

import (
	"fmt"
	"os/exec"
	"strings"
)

// On Windows, Chrome reads managed policies from
// HKLM\Software\Policies\Google\Chrome (machine-wide) or the same path under
// HKCU (per-user). We try HKCU first because it does not require an elevated
// process; HKLM is preferred when running elevated.
const winRegPath = `Software\Policies\Google\Chrome`

func policyStorageDescription() string {
	return `Windows: HKLM\` + winRegPath + `\` + GenAIPolicyName + " (REG_DWORD = 1)"
}

func applyDisableAIDownloadPolicy(dryRun bool) (PolicyResult, error) {
	hive := pickWindowsHive()
	target := fmt.Sprintf(`%s\%s\%s`, hive, winRegPath, GenAIPolicyName)

	if current, err := readWindowsPolicy(hive); err == nil && current == "0x1" {
		return PolicyResult{Applied: false, Location: target, Skipped: "already set to 1"}, nil
	}

	if dryRun {
		return PolicyResult{Applied: true, Location: target}, nil
	}

	cmd := exec.Command(
		"reg", "add", fmt.Sprintf(`%s\%s`, hive, winRegPath),
		"/v", GenAIPolicyName,
		"/t", "REG_DWORD",
		"/d", "1",
		"/f",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return PolicyResult{}, fmt.Errorf("reg add failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return PolicyResult{Applied: true, Location: target}, nil
}

func pickWindowsHive() string {
	// HKLM if we can write to it (admin), otherwise HKCU.
	probe := exec.Command("reg", "query", `HKLM\Software\Policies\Google\Chrome`)
	if err := probe.Run(); err == nil {
		return "HKLM"
	}
	return "HKCU"
}

func readWindowsPolicy(hive string) (string, error) {
	out, err := exec.Command(
		"reg", "query", fmt.Sprintf(`%s\%s`, hive, winRegPath),
		"/v", GenAIPolicyName,
	).Output()
	if err != nil {
		return "", err
	}
	text := string(out)
	idx := strings.Index(text, "REG_DWORD")
	if idx < 0 {
		return "", fmt.Errorf("value not found")
	}
	return strings.TrimSpace(text[idx+len("REG_DWORD"):]), nil
}
