package chrome

import "testing"

// withFakeAIDownloadFlags temporarily swaps the package-level
// AvailableAIDownloadFlags for a fixed, fake set, restoring the original on
// cleanup. Tests use this instead of the real list so they exercise
// DisableAIDownloadActions/GroupDisableAIDownloadActions's own logic
// regardless of how many (if any) flags are currently verified-safe to
// ship — that list is expected to change independently of whether this
// logic is correct.
func withFakeAIDownloadFlags(t *testing.T, flags ...string) []string {
	t.Helper()
	original := AvailableAIDownloadFlags
	fake := make([]AIDownloadFlag, len(flags))
	for i, name := range flags {
		fake[i] = AIDownloadFlag{Name: name}
	}
	AvailableAIDownloadFlags = fake
	t.Cleanup(func() { AvailableAIDownloadFlags = original })
	return flags
}

func TestDisableAIDownloadActions(t *testing.T) {
	all := withFakeAIDownloadFlags(t, "fake-flag-one", "fake-flag-two")

	t.Run("select all + policy: every flag applied, policy applied, nothing reverted", func(t *testing.T) {
		actions := DisableAIDownloadActions(all, true)
		if len(actions) != len(all)+1 {
			t.Fatalf("got %d actions, want %d (flags + policy)", len(actions), len(all)+1)
		}
		applyFlags, revertFlags, policy := GroupDisableAIDownloadActions(actions)
		if len(applyFlags) != len(all) {
			t.Errorf("applyFlags = %d, want %d", len(applyFlags), len(all))
		}
		if len(revertFlags) != 0 {
			t.Errorf("revertFlags = %d, want 0", len(revertFlags))
		}
		if len(policy) != 1 {
			t.Fatalf("policy = %d, want 1", len(policy))
		}
		if policy[0].Revert {
			t.Errorf("policy action should not be a revert when includePolicy=true")
		}
		if !policy[0].EnterprisePolicy {
			t.Errorf("policy action must have EnterprisePolicy=true")
		}
		if policy[0].Detail == "" {
			t.Errorf("policy action should carry a storage-location Detail")
		}
	})

	t.Run("select none, no policy: everything reverted", func(t *testing.T) {
		actions := DisableAIDownloadActions(nil, false)
		applyFlags, revertFlags, policy := GroupDisableAIDownloadActions(actions)
		if len(applyFlags) != 0 {
			t.Errorf("applyFlags = %d, want 0", len(applyFlags))
		}
		if len(revertFlags) != len(all) {
			t.Errorf("revertFlags = %d, want %d", len(revertFlags), len(all))
		}
		if len(policy) != 1 || !policy[0].Revert {
			t.Fatalf("expected exactly one revert policy action, got %+v", policy)
		}
	})

	t.Run("mixed selection: only selected flags are applied, rest reverted", func(t *testing.T) {
		one := all[0]
		actions := DisableAIDownloadActions([]string{one}, true)
		applyFlags, revertFlags, _ := GroupDisableAIDownloadActions(actions)
		if len(applyFlags) != 1 {
			t.Fatalf("applyFlags = %d, want 1", len(applyFlags))
		}
		if applyFlags[0].Label != "chrome://flags/#"+one+" -> Disabled" {
			t.Errorf("unexpected apply label: %q", applyFlags[0].Label)
		}
		if len(revertFlags) != len(all)-1 {
			t.Errorf("revertFlags = %d, want %d", len(revertFlags), len(all)-1)
		}
	})

	t.Run("unknown selected name does not fabricate an extra action", func(t *testing.T) {
		// DisableAIDownloadActions previews AvailableAIDownloadFlags only;
		// an unrecognized name in selectedFlags must not create a bogus
		// preview row (validating the name is the caller's job — see
		// internal/app.RunCLI — but the preview itself must stay truthful).
		actions := DisableAIDownloadActions([]string{"not-a-real-flag"}, false)
		if len(actions) != len(all)+1 {
			t.Fatalf("got %d actions, want %d (unknown name must not add one)", len(actions), len(all)+1)
		}
	})
}

func TestDisableAIDownloadActions_NoAvailableFlags(t *testing.T) {
	// The real AvailableAIDownloadFlags can legitimately be empty (e.g.
	// pending a verified replacement after Chrome renames/retires a flag —
	// see flags.go) — DisableAIDownloadActions must still work, producing
	// just the policy action.
	withFakeAIDownloadFlags(t)

	actions := DisableAIDownloadActions(nil, true)
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1 (policy only)", len(actions))
	}
	applyFlags, revertFlags, policy := GroupDisableAIDownloadActions(actions)
	if len(applyFlags) != 0 || len(revertFlags) != 0 {
		t.Fatalf("expected no flag actions, got apply=%v revert=%v", applyFlags, revertFlags)
	}
	if len(policy) != 1 {
		t.Fatalf("policy = %d, want 1", len(policy))
	}
}

func TestGroupDisableAIDownloadActions_Empty(t *testing.T) {
	applyFlags, revertFlags, policy := GroupDisableAIDownloadActions(nil)
	if len(applyFlags) != 0 || len(revertFlags) != 0 || len(policy) != 0 {
		t.Fatalf("expected all-empty groups for nil input, got %v %v %v", applyFlags, revertFlags, policy)
	}
}
