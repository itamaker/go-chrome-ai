//go:build linux

package chrome

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// On Linux, Chrome reads managed policies from JSON files dropped into
// /etc/opt/chrome/policies/managed/ (system-wide). There is no user-level
// equivalent — the file must be readable by the Chrome process.
const linuxManagedPolicyDir = "/etc/opt/chrome/policies/managed"

const linuxPolicyFileName = "go-chrome-ai.json"

func policyStorageDescription() string {
	return "Linux: " + linuxManagedPolicyDir + "/" + linuxPolicyFileName + " (requires sudo)"
}

func applyDisableAIDownloadPolicy(dryRun bool) (PolicyResult, error) {
	target := filepath.Join(linuxManagedPolicyDir, linuxPolicyFileName)

	if existing, err := readLinuxPolicy(target); err == nil {
		if v, ok := existing[GenAIPolicyName].(float64); ok && v == 1 {
			return PolicyResult{Applied: false, Location: target, Skipped: "already set to 1"}, nil
		}
	}

	if dryRun {
		return PolicyResult{Applied: true, Location: target}, nil
	}

	if err := os.MkdirAll(linuxManagedPolicyDir, 0o755); err != nil {
		return PolicyResult{}, fmt.Errorf("create %s failed (sudo required?): %w", linuxManagedPolicyDir, err)
	}

	payload := map[string]any{GenAIPolicyName: 1}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return PolicyResult{}, err
	}

	if err := os.WriteFile(target, encoded, 0o644); err != nil {
		return PolicyResult{}, fmt.Errorf("write %s failed (sudo required?): %w", target, err)
	}
	return PolicyResult{Applied: true, Location: target}, nil
}

func readLinuxPolicy(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
