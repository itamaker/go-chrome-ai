package chrome

import "strings"

// Chrome stores flag selections in Local State under
// `browser.enabled_labs_experiments` as a list of strings of the form
// `<flag-name>@<choice>` where 0=Default, 1=Enabled, 2=Disabled.
const flagDisabledSuffix = "@2"

// AIDownloadFlagNames are the chrome://flags entries this tool forces to
// "Disabled" so Chrome does not download Gemini Nano / on-device models.
var AIDownloadFlagNames = []string{
	"optimization-guide-on-device-model",
	"prompt-api-for-gemini-nano",
}

// DisableAIDownloadAction describes one transform applied by the
// "disable AI model download" feature.
type DisableAIDownloadAction struct {
	Label            string // human-readable label, e.g. "chrome://flags/#foo -> Disabled"
	Detail           string // optional second line (e.g. policy storage location)
	EnterprisePolicy bool   // true if the action writes a managed Chrome policy
	PolicyNote       string // extra warning shown when EnterprisePolicy is true
}

// DisableAIDownloadActions returns the ordered list of changes applied when
// the user enables "disable AI model download". The last entry is an
// Enterprise policy write that causes Chrome to display the
// "managed by your organization" banner.
func DisableAIDownloadActions() []DisableAIDownloadAction {
	actions := make([]DisableAIDownloadAction, 0, len(AIDownloadFlagNames)+1)
	for _, name := range AIDownloadFlagNames {
		actions = append(actions, DisableAIDownloadAction{
			Label: "chrome://flags/#" + name + " -> Disabled",
		})
	}
	actions = append(actions, DisableAIDownloadAction{
		Label:            GenAIPolicyName + " = 1 (Disabled)",
		Detail:           policyStorageDescription(),
		EnterprisePolicy: true,
		PolicyNote:       `Chrome will show the "managed by your organization" banner`,
	})
	return actions
}


// setFlagsDisabled rewrites browser.enabled_labs_experiments so each requested
// flag appears exactly once with the Disabled choice (@2). Returns the list
// of flag names whose state actually changed.
func setFlagsDisabled(localState map[string]any, flags []string) []string {
	browser, _ := localState["browser"].(map[string]any)
	if browser == nil {
		browser = map[string]any{}
		localState["browser"] = browser
	}

	rawList, _ := browser["enabled_labs_experiments"].([]any)
	existing := make([]string, 0, len(rawList))
	for _, item := range rawList {
		if s, ok := item.(string); ok {
			existing = append(existing, s)
		}
	}

	targets := make(map[string]bool, len(flags))
	for _, name := range flags {
		targets[name] = true
	}

	kept := make([]string, 0, len(existing))
	alreadyDisabled := make(map[string]bool, len(flags))
	for _, entry := range existing {
		name := entry
		if idx := strings.IndexByte(entry, '@'); idx >= 0 {
			name = entry[:idx]
		}
		if targets[name] {
			if entry == name+flagDisabledSuffix && !alreadyDisabled[name] {
				alreadyDisabled[name] = true
				kept = append(kept, entry)
			}
			continue
		}
		kept = append(kept, entry)
	}

	changed := make([]string, 0, len(flags))
	for _, name := range flags {
		if !alreadyDisabled[name] {
			kept = append(kept, name+flagDisabledSuffix)
			changed = append(changed, name)
		}
	}

	if len(changed) == 0 && len(kept) == len(existing) {
		return nil
	}

	next := make([]any, len(kept))
	for i, s := range kept {
		next[i] = s
	}
	browser["enabled_labs_experiments"] = next
	return changed
}
