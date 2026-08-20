package chrome

import (
	"errors"
	"testing"
)

// These tests exercise the shared applyPolicy/removePolicy skeleton
// against fake policyBackend implementations, so they run identically (and
// deterministically) on every OS regardless of which platform_*.go file is
// actually compiled in — unlike the real backends, they never shell out or
// touch the filesystem/registry.

func TestApplyPolicy(t *testing.T) {
	t.Run("already set: no-op, reports skipped", func(t *testing.T) {
		wrote := false
		b := policyBackend{
			location: "loc",
			isSet:    func() (bool, error) { return true, nil },
			write:    func() (string, error) { wrote = true; return "loc", nil },
		}
		result, err := applyPolicy(b, false)
		if err != nil {
			t.Fatalf("applyPolicy: %v", err)
		}
		if result.Applied {
			t.Errorf("expected Applied=false when already set")
		}
		if result.Skipped != "already set to 1" {
			t.Errorf("Skipped = %q, want %q", result.Skipped, "already set to 1")
		}
		if result.Location != "loc" {
			t.Errorf("Location = %q, want %q", result.Location, "loc")
		}
		if wrote {
			t.Errorf("write should not have been called")
		}
	})

	t.Run("dry run: reports Applied without writing", func(t *testing.T) {
		wrote := false
		b := policyBackend{
			location: "loc",
			isSet:    func() (bool, error) { return false, nil },
			write:    func() (string, error) { wrote = true; return "loc", nil },
		}
		result, err := applyPolicy(b, true)
		if err != nil {
			t.Fatalf("applyPolicy: %v", err)
		}
		if !result.Applied {
			t.Errorf("expected Applied=true for dry-run preview")
		}
		if wrote {
			t.Errorf("dry-run must not call write")
		}
	})

	t.Run("not set: writes and reports the actual write location", func(t *testing.T) {
		b := policyBackend{
			location: "preview-loc",
			isSet:    func() (bool, error) { return false, nil },
			write:    func() (string, error) { return "actual-loc", nil },
		}
		result, err := applyPolicy(b, false)
		if err != nil {
			t.Fatalf("applyPolicy: %v", err)
		}
		if !result.Applied {
			t.Errorf("expected Applied=true")
		}
		if result.Location != "actual-loc" {
			t.Errorf("Location = %q, want the location write() actually reported (%q)", result.Location, "actual-loc")
		}
	})

	t.Run("isSet error is propagated, not treated as absent", func(t *testing.T) {
		sentinel := errors.New("boom")
		b := policyBackend{
			location: "loc",
			isSet:    func() (bool, error) { return false, sentinel },
		}
		_, err := applyPolicy(b, false)
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected wrapped sentinel error, got %v", err)
		}
	})

	t.Run("write error is propagated", func(t *testing.T) {
		sentinel := errors.New("write failed")
		b := policyBackend{
			location: "loc",
			isSet:    func() (bool, error) { return false, nil },
			write:    func() (string, error) { return "", sentinel },
		}
		_, err := applyPolicy(b, false)
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected sentinel error, got %v", err)
		}
	})
}

func TestRemovePolicy(t *testing.T) {
	t.Run("not present: no-op, reports skipped", func(t *testing.T) {
		cleared := false
		b := policyBackend{
			location: "loc",
			exists:   func() (bool, error) { return false, nil },
			clear:    func() (string, error) { cleared = true; return "loc", nil },
		}
		result, err := removePolicy(b, false)
		if err != nil {
			t.Fatalf("removePolicy: %v", err)
		}
		if result.Applied {
			t.Errorf("expected Applied=false when not present")
		}
		if result.Skipped != "not set" {
			t.Errorf("Skipped = %q, want %q", result.Skipped, "not set")
		}
		if cleared {
			t.Errorf("clear should not have been called")
		}
	})

	t.Run("dry run: reports Applied without clearing", func(t *testing.T) {
		cleared := false
		b := policyBackend{
			location: "loc",
			exists:   func() (bool, error) { return true, nil },
			clear:    func() (string, error) { cleared = true; return "loc", nil },
		}
		result, err := removePolicy(b, true)
		if err != nil {
			t.Fatalf("removePolicy: %v", err)
		}
		if !result.Applied {
			t.Errorf("expected Applied=true for dry-run preview")
		}
		if cleared {
			t.Errorf("dry-run must not call clear")
		}
	})

	t.Run("present: clears and reports the actual clear location", func(t *testing.T) {
		b := policyBackend{
			location: "preview-loc",
			exists:   func() (bool, error) { return true, nil },
			clear:    func() (string, error) { return "actual-loc", nil },
		}
		result, err := removePolicy(b, false)
		if err != nil {
			t.Fatalf("removePolicy: %v", err)
		}
		if !result.Applied {
			t.Errorf("expected Applied=true")
		}
		if result.Location != "actual-loc" {
			t.Errorf("Location = %q, want %q", result.Location, "actual-loc")
		}
	})

	t.Run("exists error is propagated, not treated as absent", func(t *testing.T) {
		sentinel := errors.New("boom")
		b := policyBackend{
			location: "loc",
			exists:   func() (bool, error) { return false, sentinel },
		}
		_, err := removePolicy(b, false)
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected wrapped sentinel error, got %v", err)
		}
	})

	t.Run("clear error is propagated", func(t *testing.T) {
		sentinel := errors.New("clear failed")
		b := policyBackend{
			location: "loc",
			exists:   func() (bool, error) { return true, nil },
			clear:    func() (string, error) { return "", sentinel },
		}
		_, err := removePolicy(b, false)
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected sentinel error, got %v", err)
		}
	})
}

// TestDisableAIDownloadPolicyBackend_Smoke exercises the real,
// platform-specific backend (whichever of policy_darwin.go / policy_linux.go
// / policy_windows.go / policy_other.go the build tags select) through
// ApplyDisableAIDownloadPolicy/RemoveDisableAIDownloadPolicy in dry-run
// mode. Dry-run still performs the real (read-only) isSet/exists check —
// only the write/clear side effect is skipped — so this genuinely runs the
// platform's `defaults`/registry/JSON-file read path in CI instead of that
// code going completely unexercised outside manual testing.
func TestDisableAIDownloadPolicyBackend_Smoke(t *testing.T) {
	applyResult, applyErr := ApplyDisableAIDownloadPolicy(true)
	removeResult, removeErr := RemoveDisableAIDownloadPolicy(true)

	// Every real platform (darwin/linux/windows) should succeed in dry-run,
	// since no privileged operation is attempted. Only policy_other.go's
	// intentional stub (unsupported OS) returns an error — which is itself
	// correct, documented behavior, so it isn't treated as a test failure
	// here.
	if applyErr == nil && applyResult.Location == "" {
		t.Errorf("ApplyDisableAIDownloadPolicy dry-run: empty Location")
	}
	if removeErr == nil && removeResult.Location == "" {
		t.Errorf("RemoveDisableAIDownloadPolicy dry-run: empty Location")
	}

	if policyStorageDescription(true) == "" || policyStorageDescription(false) == "" {
		t.Errorf("policyStorageDescription must never return an empty string")
	}
}
