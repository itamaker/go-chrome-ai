package chrome

import "strings"

// Chrome stores flag selections in Local State under
// `browser.enabled_labs_experiments` as a list of strings of the form
// `<flag-name>@<choice>`. For a plain boolean flag (about_flags.cc
// ENABLE_DISABLE_VALUE / FEATURE_VALUE with no named variations), choice
// indices are fixed: 0=Default, 1=Enabled, 2=Disabled. That fixed mapping
// is NOT guaranteed once a flag has named variations
// (FEATURE_WITH_PARAMS_VALUE) — Chrome can insert those between the
// Default/Enabled/Disabled slots, shifting Disabled to a higher index (see
// AvailableAIDownloadFlags below for a confirmed real example). Only add a
// flag name here after checking its actual about_flags.cc entry type, or
// by writing "<name>@2" and independently confirming — live, not by
// assumption — that Chrome shows it as "Disabled" after a restart.
const flagDisabledSuffix = "@2"

// AIDownloadFlag describes one chrome://flags entry this tool can force to
// "Disabled" so Chrome does not download Gemini Nano / on-device models.
type AIDownloadFlag struct {
	Name string // chrome://flags entry name
}

// AvailableAIDownloadFlags lists every chrome://flags entry this tool knows
// how to disable and currently knows is both (a) a real, present flag ID
// and (b) confirmed to actually mean "Disabled" at flagDisabledSuffix.
// Callers (CLI/GUI) present each one as an individually selectable option,
// plus a "select all" convenience over this same list.
//
// This is empty because, as of Chrome 151 (confirmed live and against
// Chromium's current about_flags.cc source), neither flag this tool
// previously listed still qualifies:
//   - "optimization-guide-on-device-model" no longer exists as a flag ID at
//     all (removed from about_flags.cc, no successor — the feature moved
//     to a Settings toggle).
//   - "prompt-api-for-gemini-nano" was renamed to "prompt-api", but its
//     current option index 2 is NOT "Disabled" — it's "Enabled
//     Multilingual" (confirmed live: writing "prompt-api@2" and
//     restarting leaves the flags UI showing "Enabled Multilingual"
//     selected). It uses FEATURE_WITH_PARAMS_VALUE in about_flags.cc,
//     whose named variation ("Multilingual") lands at index 2, pushing
//     "Disabled" to a different index this tool hasn't verified.
//
// A stale-but-nonexistent flag ID (the first case) is a harmless no-op —
// Chrome just drops the unrecognized entry on its next launch — but
// "prompt-api" is not: writing it with the wrong index actively sets a
// different, wrong state while this tool reports it as "Disabled". Rather
// than list something that either does nothing or does the wrong thing,
// this stays empty until a flag is verified end-to-end (real flag ID, real
// confirmed Disabled index). The GenAILocalFoundationalModelSettings
// Enterprise policy (policy.go) is independent of this list and is the
// mechanism actually confirmed reliable — see README.
var AvailableAIDownloadFlags = []AIDownloadFlag{}

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
// not. Flags outside managed are left untouched, including any non-string
// entries (which are preserved verbatim rather than dropped). Names in
// selected that are not in managed are ignored — callers (CLI/GUI) should
// still validate user input for a good error message, but the engine does
// not trust them to, since a bad name here would otherwise be written
// straight into the user's Chrome profile. Duplicate names in selected are
// coalesced. Returns the flags that were newly disabled and the flags whose
// override was removed.
//
// localState["browser"] is only created if a change actually needs to be
// written; a call that ends up being a no-op never touches localState at
// all, even to insert an empty browser map.
func syncManagedFlags(localState map[string]any, managed, selected []string) (disabled, reverted []string) {
	existingBrowser, _ := localState["browser"].(map[string]any)

	var rawList []any
	if existingBrowser != nil {
		rawList, _ = existingBrowser["enabled_labs_experiments"].([]any)
	}

	isManaged := make(map[string]bool, len(managed))
	for _, name := range managed {
		isManaged[name] = true
	}

	// Filter selected down to known, deduplicated names, preserving order.
	isSelected := make(map[string]bool, len(selected))
	orderedSelected := make([]string, 0, len(selected))
	for _, name := range selected {
		if !isManaged[name] || isSelected[name] {
			continue
		}
		isSelected[name] = true
		orderedSelected = append(orderedSelected, name)
	}

	kept := make([]any, 0, len(rawList)+len(orderedSelected))
	alreadyDisabled := make(map[string]bool, len(orderedSelected))
	wasPresent := make(map[string]bool, len(managed))
	for _, item := range rawList {
		entry, ok := item.(string)
		if !ok {
			kept = append(kept, item) // not a flag override we understand; preserve as-is
			continue
		}
		name, _, _ := strings.Cut(entry, "@")
		if !isManaged[name] {
			kept = append(kept, item)
			continue
		}
		wasPresent[name] = true
		if !isSelected[name] {
			continue // revert: drop the existing override entirely
		}
		if entry == name+flagDisabledSuffix && !alreadyDisabled[name] {
			alreadyDisabled[name] = true
			kept = append(kept, item)
		}
	}

	for _, name := range orderedSelected {
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

	if len(disabled) == 0 && len(reverted) == 0 && len(kept) == len(rawList) {
		return nil, nil
	}

	browser := existingBrowser
	if browser == nil {
		browser = map[string]any{}
		localState["browser"] = browser
	}
	browser["enabled_labs_experiments"] = kept
	return disabled, reverted
}
