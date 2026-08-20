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

func policyStorageDescription(applying bool) string {
	verb := "write"
	if !applying {
		verb = "remove"
	}
	return "Linux: " + verb + " " + linuxManagedPolicyDir + "/" + linuxPolicyFileName + " (requires sudo)"
}

func disableAIDownloadPolicyBackend() policyBackend {
	target := filepath.Join(linuxManagedPolicyDir, linuxPolicyFileName)

	return policyBackend{
		location: target,
		isSet: func() (bool, error) {
			existing, ok, err := readLinuxPolicy(target)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
			v, isNum := existing[GenAIPolicyName].(float64)
			return isNum && v == 1, nil
		},
		exists: func() (bool, error) {
			_, ok, err := readLinuxPolicy(target)
			return ok, err
		},
		write: func() (string, error) {
			if err := os.MkdirAll(linuxManagedPolicyDir, 0o755); err != nil {
				return "", fmt.Errorf("create %s failed (sudo required?): %w", linuxManagedPolicyDir, err)
			}
			payload := map[string]any{GenAIPolicyName: 1}
			encoded, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(target, encoded, 0o644); err != nil {
				return "", fmt.Errorf("write %s failed (sudo required?): %w", target, err)
			}
			return target, nil
		},
		clear: func() (string, error) {
			if err := os.Remove(target); err != nil {
				return "", fmt.Errorf("remove %s failed (sudo required?): %w", target, err)
			}
			return target, nil
		},
	}
}

// readLinuxPolicy reads and parses the managed-policy JSON file. ok is
// false (with a nil error) when the file simply doesn't exist; a non-nil
// error means the file exists but could not be read/parsed (permission
// denied, corrupt JSON, ...) and must not be treated as "not set".
func readLinuxPolicy(path string) (policy map[string]any, ok bool, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, true, fmt.Errorf("parse %s: %w", path, err)
	}
	return out, true, nil
}
