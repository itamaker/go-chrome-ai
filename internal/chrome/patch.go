package chrome

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PatchResult describes what changed in Local State.
type PatchResult struct {
	Modified                                        bool
	GLICEligiblePatched                             bool
	VariationsCountryPatched                        bool
	VariationsPermanentConsistencyCountryWasPatched bool
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
func PatchLocalState(userDataPath, lastVersion string, dryRun bool) (PatchResult, error) {
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

	if !result.Modified || dryRun {
		return result, nil
	}

	encoded, err := json.Marshal(localState)
	if err != nil {
		return PatchResult{}, fmt.Errorf("encode Local State failed: %w", err)
	}

	fileInfo, err := os.Stat(localStateFile)
	fileMode := os.FileMode(0o644)
	if err == nil {
		fileMode = fileInfo.Mode().Perm()
	}

	if err := os.WriteFile(localStateFile, encoded, fileMode); err != nil {
		return PatchResult{}, err
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
