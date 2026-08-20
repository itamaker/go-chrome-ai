package chrome

import "fmt"

// GenAIPolicyName is the Chrome Enterprise policy that controls whether the
// Gemini Nano / on-device foundational model is downloaded and used locally.
//
// Value 0 = Allowed (default), 1 = Disabled (do not download, delete cached
// copy). See:
// https://chromeenterprise.google/policies/gen-ai-local-foundational-model-settings/
const GenAIPolicyName = "GenAILocalFoundationalModelSettings"

// PolicyResult is the outcome of applying a host-level Chrome policy.
type PolicyResult struct {
	Applied  bool   // true if the platform store was updated (or would be, in dry-run)
	Location string // human-readable destination (file path, registry key, defaults domain)
	Skipped  string // populated when nothing was written and Applied == false
}

// policyBackend is the minimal, platform-specific surface the shared
// apply/remove skeleton (below) needs. Each policy_<os>.go file implements
// one of these and hands it to applyPolicy/removePolicy — the "read
// current, compare, dry-run early-return, mutate, wrap error" control flow
// that used to be copy-pasted three times (once per OS, twice within each
// file for apply vs. remove) now lives in exactly one place.
type policyBackend struct {
	// location is the human-readable destination shown to the user for
	// results that don't actually perform a write (dry-run preview,
	// already-satisfied, already-absent). It's static because those paths
	// never learn anything more precise than "where a write would go".
	location string
	// isSet reports whether the policy is currently set to the value this
	// tool would write (1 / Disabled). An error here means "could not
	// determine the current state" and must NOT be treated as "not set" —
	// that conflation previously reported real failures (missing binary,
	// permission denied) to the user as a misleading "already absent".
	isSet func() (bool, error)
	// exists reports whether the policy is present at all, in any value —
	// used by removePolicy to decide whether there's anything to remove.
	// An error here has the same "do not treat as absent" contract as
	// isSet.
	exists func() (bool, error)
	// write sets the policy to 1 (Disabled) and returns the location it
	// actually ended up at. This is a function rather than always being
	// `location` because Windows doesn't know in advance whether it can
	// write HKLM (needs elevation) or has to fall back to HKCU.
	write func() (location string, err error)
	// clear removes the policy entirely, reverting Chrome to its unmanaged
	// default, and returns the location it actually removed it from.
	clear func() (location string, err error)
}

// applyPolicy writes a policyBackend's target to 1 (Disabled), unless it is
// already set that way. dryRun previews the change without writing.
func applyPolicy(b policyBackend, dryRun bool) (PolicyResult, error) {
	set, err := b.isSet()
	if err != nil {
		return PolicyResult{}, fmt.Errorf("read current policy state: %w", err)
	}
	if set {
		return PolicyResult{Applied: false, Location: b.location, Skipped: "already set to 1"}, nil
	}

	if dryRun {
		return PolicyResult{Applied: true, Location: b.location}, nil
	}

	location, err := b.write()
	if err != nil {
		return PolicyResult{}, err
	}
	return PolicyResult{Applied: true, Location: location}, nil
}

// removePolicy clears a policyBackend's target, unless it is already
// absent. dryRun previews the change without writing.
func removePolicy(b policyBackend, dryRun bool) (PolicyResult, error) {
	present, err := b.exists()
	if err != nil {
		return PolicyResult{}, fmt.Errorf("read current policy state: %w", err)
	}
	if !present {
		return PolicyResult{Applied: false, Location: b.location, Skipped: "not set"}, nil
	}

	if dryRun {
		return PolicyResult{Applied: true, Location: b.location}, nil
	}

	location, err := b.clear()
	if err != nil {
		return PolicyResult{}, err
	}
	return PolicyResult{Applied: true, Location: location}, nil
}

// ApplyDisableAIDownloadPolicy writes GenAILocalFoundationalModelSettings=1
// to the platform's Chrome managed-policy store.
func ApplyDisableAIDownloadPolicy(dryRun bool) (PolicyResult, error) {
	return applyPolicy(disableAIDownloadPolicyBackend(), dryRun)
}

// RemoveDisableAIDownloadPolicy removes GenAILocalFoundationalModelSettings
// from the platform's Chrome managed-policy store, reverting Chrome to its
// unmanaged default (no "managed by your organization" banner from this
// policy).
func RemoveDisableAIDownloadPolicy(dryRun bool) (PolicyResult, error) {
	return removePolicy(disableAIDownloadPolicyBackend(), dryRun)
}
