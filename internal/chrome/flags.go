package chrome

import "strings"

// Chrome stores flag selections in Local State under
// `browser.enabled_labs_experiments` as a list of strings of the form
// `<flag-name>@<choice>` where 0=Default, 1=Enabled, 2=Disabled.
const flagDisabledSuffix = "@2"

// AIDownloadFlag describes one chrome://flags entry this tool can force to
// "Disabled" so Chrome does not download Gemini Nano / on-device models.
type AIDownloadFlag struct {
	Name string // chrome://flags entry name
}

// AvailableAIDownloadFlags lists every chrome://flags entry this tool knows
// how to disable. Callers (CLI/GUI) present each one as an individually
// selectable option, plus a "select all" convenience over this same list.
var AvailableAIDownloadFlags = []AIDownloadFlag{
	{Name: "optimization-guide-on-device-model"},
	{Name: "prompt-api-for-gemini-nano"},
}

// AllAIDownloadFlagNames returns the names of every available AI-download
// flag, i.e. the set selected by a "select all" option.
func AllAIDownloadFlagNames() []string {
	names := make([]string, len(AvailableAIDownloadFlags))
	for i, f := range AvailableAIDownloadFlags {
		names[i] = f.Name
	}
	return names
}

// DisableAIDownloadAction describes one transform previewed/applied by the
// "disable AI model download" feature.
type DisableAIDownloadAction struct {
	Label            string // human-readable label, e.g. "chrome://flags/#foo -> Disabled"
	Detail           string // optional second line (e.g. policy storage location)
	EnterprisePolicy bool   // true if the action reads/writes the managed Chrome policy
	PolicyNote       string // extra warning shown when EnterprisePolicy is true
	Revert           bool   // true if this action undoes a previous change instead of applying one
}

// DisableAIDownloadActions returns one action per managed item — every entry
// in AvailableAIDownloadFlags plus the Enterprise policy — describing
// whether it will be applied (selected / includePolicy) or reverted to
// Chrome's default (not selected / !includePolicy). The policy is
// independent of which flags are selected.
func DisableAIDownloadActions(selectedFlags []string, includePolicy bool) []DisableAIDownloadAction {
	selected := make(map[string]bool, len(selectedFlags))
	for _, name := range selectedFlags {
		selected[name] = true
	}

	actions := make([]DisableAIDownloadAction, 0, len(AvailableAIDownloadFlags)+1)
	for _, f := range AvailableAIDownloadFlags {
		if selected[f.Name] {
			actions = append(actions, DisableAIDownloadAction{
				Label: "chrome://flags/#" + f.Name + " -> Disabled",
			})
			continue
		}
		actions = append(actions, DisableAIDownloadAction{
			Label:  "chrome://flags/#" + f.Name + " -> Default (reset)",
			Revert: true,
		})
	}

	if includePolicy {
		actions = append(actions, DisableAIDownloadAction{
			Label:            GenAIPolicyName + " = 1 (Disabled)",
			Detail:           policyStorageDescription(true),
			EnterprisePolicy: true,
			PolicyNote:       `Chrome will show the "managed by your organization" banner`,
		})
	} else {
		actions = append(actions, DisableAIDownloadAction{
			Label:            GenAIPolicyName + " removed (reset to default)",
			Detail:           policyStorageDescription(false),
			EnterprisePolicy: true,
			PolicyNote:       `Removes the "managed by your organization" banner, if shown`,
			Revert:           true,
		})
	}
	return actions
}

// GroupDisableAIDownloadActions splits actions (as returned by
// DisableAIDownloadActions) into three buckets for display: chrome://flags
// entries to apply, chrome://flags entries to revert, and the Enterprise
// policy action (apply or revert).
func GroupDisableAIDownloadActions(actions []DisableAIDownloadAction) (applyFlags, revertFlags, policy []DisableAIDownloadAction) {
	for _, a := range actions {
		switch {
		case a.EnterprisePolicy:
			policy = append(policy, a)
		case a.Revert:
			revertFlags = append(revertFlags, a)
		default:
			applyFlags = append(applyFlags, a)
		}
	}
	return applyFlags, revertFlags, policy
}

// syncManagedFlags rewrites browser.enabled_labs_experiments so every flag
// in managed is Disabled (@2) when it also appears in selected, or has any
// existing override removed (reverted to Chrome's default) when it does
// not. Flags outside managed are left untouched. Returns the flags that
// were newly disabled and the flags whose override was removed.
func syncManagedFlags(localState map[string]any, managed, selected []string) (disabled, reverted []string) {
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

	isManaged := make(map[string]bool, len(managed))
	for _, name := range managed {
		isManaged[name] = true
	}
	isSelected := make(map[string]bool, len(selected))
	for _, name := range selected {
		isSelected[name] = true
	}

	kept := make([]string, 0, len(existing))
	alreadyDisabled := make(map[string]bool, len(selected))
	wasPresent := make(map[string]bool, len(managed))
	for _, entry := range existing {
		name := entry
		if idx := strings.IndexByte(entry, '@'); idx >= 0 {
			name = entry[:idx]
		}
		if !isManaged[name] {
			kept = append(kept, entry)
			continue
		}
		wasPresent[name] = true
		if !isSelected[name] {
			continue // revert: drop the existing override entirely
		}
		if entry == name+flagDisabledSuffix && !alreadyDisabled[name] {
			alreadyDisabled[name] = true
			kept = append(kept, entry)
		}
	}

	for _, name := range selected {
		if !alreadyDisabled[name] {
			kept = append(kept, name+flagDisabledSuffix)
			disabled = append(disabled, name)
		}
	}
	for _, name := range managed {
		if !isSelected[name] && wasPresent[name] {
			reverted = append(reverted, name)
		}
	}

	if len(disabled) == 0 && len(reverted) == 0 && len(kept) == len(existing) {
		return nil, nil
	}

	next := make([]any, len(kept))
	for i, s := range kept {
		next[i] = s
	}
	browser["enabled_labs_experiments"] = next
	return disabled, reverted
}
