package chrome

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// localStateBackupSuffix names the one-time pristine snapshot this tool
// keeps of Local State before its first modification, as a manual rollback
// point (see backupFileOnce in atomicfile.go).
const localStateBackupSuffix = ".go-chrome-ai.bak"

// PatchResult describes what changed in Local State.
type PatchResult struct {
	Modified                                        bool
	GLICEligiblePatched                             bool
	VariationsCountryPatched                        bool
	VariationsPermanentConsistencyCountryWasPatched bool
	DisabledFlags                                   []string
	RevertedFlags                                   []string
}

// PatchOptions controls which transforms PatchLocalState applies.
type PatchOptions struct {
	// AIDownloadFlags is the set of chrome://flags entry names (from
	// AvailableAIDownloadFlags) to force to Disabled. Any managed flag not
	// in this set has its override removed (reverted to Chrome's default)
	// if present.
	AIDownloadFlags []string
}

func ReadLastVersion(userDataPath string) (string, error) {
	lastVersionFile := filepath.Join(userDataPath, "Last Version")
	content, err := os.ReadFile(lastVersionFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// PatchLocalState updates Local State for one Chrome profile directory.
func PatchLocalState(userDataPath, lastVersion string, dryRun bool, opts PatchOptions) (PatchResult, error) {
	localStateFile := filepath.Join(userDataPath, "Local State")
	raw, err := os.ReadFile(localStateFile)
	if err != nil {
		return PatchResult{}, err
	}

	var localState map[string]any
	if err := json.Unmarshal(raw, &localState); err != nil {
		return PatchResult{}, fmt.Errorf("parse Local State failed: %w", err)
	}

	result := PatchResult{}

	if setAllIsGLICEligible(localState) {
		result.GLICEligiblePatched = true
		result.Modified = true
	}

	if v, ok := localState["variations_country"]; !ok || v != "us" {
		localState["variations_country"] = "us"
		result.VariationsCountryPatched = true
		result.Modified = true
	}

	if value, exists := localState["variations_permanent_consistency_country"]; exists {
		if entries, ok := value.([]any); ok && len(entries) >= 2 {
			needsPatch := entries[0] != lastVersion || entries[1] != "us"
			if needsPatch {
				entries[0] = lastVersion
				entries[1] = "us"
				localState["variations_permanent_consistency_country"] = entries
				result.VariationsPermanentConsistencyCountryWasPatched = true
				result.Modified = true
			}
		}
	}

	if disabled, reverted := syncManagedFlags(localState, AllAIDownloadFlagNames(), opts.AIDownloadFlags); len(disabled) > 0 || len(reverted) > 0 {
		result.DisabledFlags = disabled
		result.RevertedFlags = reverted
		result.Modified = true
	}

	if !result.Modified || dryRun {
		return result, nil
	}

	encoded, err := json.Marshal(localState)
	if err != nil {
		return result, fmt.Errorf("encode Local State failed: %w", err)
	}

	// Reuse the file's existing permissions rather than guessing. A stat
	// failure here is surfaced rather than silently falling back to 0644,
	// which could widen a profile that was deliberately locked down (e.g.
	// 0600) and would otherwise happen invisibly.
	fileInfo, err := os.Stat(localStateFile)
	if err != nil {
		return result, fmt.Errorf("stat Local State failed: %w", err)
	}
	fileMode := fileInfo.Mode().Perm()

	// Keep a one-time pristine snapshot before the first modification, so
	// there is a manual rollback point if the patched profile misbehaves.
	if err := backupFileOnce(localStateFile, localStateBackupSuffix); err != nil {
		return result, fmt.Errorf("backup Local State failed: %w", err)
	}

	if err := writeFileAtomic(localStateFile, encoded, fileMode); err != nil {
		return result, fmt.Errorf("write Local State failed: %w", err)
	}

	return result, nil
}

func setAllIsGLICEligible(v any) bool {
	switch typed := v.(type) {
	case map[string]any:
		modified := false
		for key, value := range typed {
			if key == "is_glic_eligible" {
				boolValue, ok := value.(bool)
				if !ok || !boolValue {
					typed[key] = true
					modified = true
				}
				continue
			}
			if setAllIsGLICEligible(value) {
				modified = true
			}
		}
		return modified
	case []any:
		modified := false
		for _, item := range typed {
			if setAllIsGLICEligible(item) {
				modified = true
			}
		}
		return modified
	default:
		return false
	}
}
