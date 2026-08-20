package chrome

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeJSONFile(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return v
}

// newTestProfile writes a minimal Local State + Last Version pair into a
// fresh temp dir, mimicking the two files PatchLocalState/ReadLastVersion
// expect inside a real Chrome user-data directory.
func newTestProfile(t *testing.T, localState map[string]any) (dir, localStateFile string) {
	t.Helper()
	dir = t.TempDir()
	localStateFile = filepath.Join(dir, "Local State")
	writeJSONFile(t, localStateFile, localState)
	if err := os.WriteFile(filepath.Join(dir, "Last Version"), []byte("120.0.6099.109\n"), 0o644); err != nil {
		t.Fatalf("write Last Version: %v", err)
	}
	return dir, localStateFile
}

func TestReadLastVersion(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadLastVersion(dir); err == nil {
		t.Fatalf("expected error for missing Last Version file")
	}

	if err := os.WriteFile(filepath.Join(dir, "Last Version"), []byte(" 120.0.6099.109 \n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadLastVersion(dir)
	if err != nil {
		t.Fatalf("ReadLastVersion: %v", err)
	}
	if got != "120.0.6099.109" {
		t.Fatalf("got %q, want trimmed version", got)
	}
}

// setAllIsGLICEligible is the single most dangerous function in this
// package (it recursively rewrites the entire Local State document), and
// previously had zero test coverage.
func TestSetAllIsGLICEligible(t *testing.T) {
	cases := []struct {
		name    string
		input   any
		wantMod bool
	}{
		{"top-level false flipped", map[string]any{"is_glic_eligible": false}, true},
		{"top-level already true is untouched", map[string]any{"is_glic_eligible": true}, false},
		{"non-bool value is coerced to true", map[string]any{"is_glic_eligible": "nope"}, true},
		{
			"nested inside maps and arrays, mixed states",
			map[string]any{
				"profile": map[string]any{
					"is_glic_eligible": false,
					"other":            "unchanged",
				},
				"list": []any{
					map[string]any{"is_glic_eligible": true},
					map[string]any{"is_glic_eligible": false},
				},
			},
			true,
		},
		{"no matching key anywhere", map[string]any{"unrelated": map[string]any{"x": 1}}, false},
		{"scalar string is left alone", "just a string", false},
		{"nil is left alone", nil, false},
		{"bare number is left alone", float64(42), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotMod := setAllIsGLICEligible(tc.input)
			if gotMod != tc.wantMod {
				t.Fatalf("modified = %v, want %v (state after: %#v)", gotMod, tc.wantMod, tc.input)
			}
		})
	}

	// Every is_glic_eligible in a nested structure must end up true, no
	// matter how deep or what its prior value/type was.
	state := map[string]any{
		"a": map[string]any{"is_glic_eligible": false},
		"b": []any{
			map[string]any{"is_glic_eligible": false},
			map[string]any{"nested": map[string]any{"is_glic_eligible": "x"}},
		},
	}
	if !setAllIsGLICEligible(state) {
		t.Fatalf("expected modification")
	}
	if got := state["a"].(map[string]any)["is_glic_eligible"]; got != true {
		t.Errorf("a.is_glic_eligible = %v, want true", got)
	}
	b := state["b"].([]any)
	if got := b[0].(map[string]any)["is_glic_eligible"]; got != true {
		t.Errorf("b[0].is_glic_eligible = %v, want true", got)
	}
	nested := b[1].(map[string]any)["nested"].(map[string]any)
	if got := nested["is_glic_eligible"]; got != true {
		t.Errorf("nested.is_glic_eligible = %v, want true", got)
	}

	// Re-running on an already-fully-patched structure is a no-op.
	if setAllIsGLICEligible(state) {
		t.Fatalf("expected no modification on second call")
	}
}

func TestPatchLocalState_MissingFile(t *testing.T) {
	dir := t.TempDir() // no "Local State" written
	if _, err := PatchLocalState(dir, "120.0.6099.109", false, PatchOptions{}); err == nil {
		t.Fatalf("expected error when Local State is missing")
	}
}

func TestPatchLocalState_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Local State"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := PatchLocalState(dir, "120.0.6099.109", false, PatchOptions{}); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

func TestPatchLocalState_ModifiesAndIsIdempotent(t *testing.T) {
	dir, localStateFile := newTestProfile(t, map[string]any{
		"is_glic_eligible": false,
	})

	result, err := PatchLocalState(dir, "120.0.6099.109", false, PatchOptions{})
	if err != nil {
		t.Fatalf("PatchLocalState: %v", err)
	}
	if !result.Modified || !result.GLICEligiblePatched || !result.VariationsCountryPatched {
		t.Fatalf("unexpected result: %+v", result)
	}

	onDisk := readJSONFile(t, localStateFile)
	if onDisk["is_glic_eligible"] != true {
		t.Errorf("is_glic_eligible not persisted as true: %v", onDisk["is_glic_eligible"])
	}
	if onDisk["variations_country"] != "us" {
		t.Errorf("variations_country not persisted as us: %v", onDisk["variations_country"])
	}

	// A second run against the now-patched file should be a no-op.
	result2, err := PatchLocalState(dir, "120.0.6099.109", false, PatchOptions{})
	if err != nil {
		t.Fatalf("second PatchLocalState: %v", err)
	}
	if result2.Modified {
		t.Fatalf("expected second run to be a no-op, got %+v", result2)
	}
}

func TestPatchLocalState_DryRunDoesNotWrite(t *testing.T) {
	dir, localStateFile := newTestProfile(t, map[string]any{
		"is_glic_eligible": false,
	})
	original, err := os.ReadFile(localStateFile)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}

	result, err := PatchLocalState(dir, "120.0.6099.109", true, PatchOptions{})
	if err != nil {
		t.Fatalf("PatchLocalState: %v", err)
	}
	if !result.Modified {
		t.Fatalf("expected dry-run to still report the change it detected")
	}

	after, err := os.ReadFile(localStateFile)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(after) != string(original) {
		t.Fatalf("dry-run must not modify Local State on disk")
	}
	if _, err := os.Stat(localStateFile + localStateBackupSuffix); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not create a backup file, stat err = %v", err)
	}
}

func TestPatchLocalState_PreservesFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file mode bits don't apply on Windows")
	}
	dir, localStateFile := newTestProfile(t, map[string]any{
		"is_glic_eligible": false,
	})
	if err := os.Chmod(localStateFile, 0o600); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	if _, err := PatchLocalState(dir, "120.0.6099.109", false, PatchOptions{}); err != nil {
		t.Fatalf("PatchLocalState: %v", err)
	}

	info, err := os.Stat(localStateFile)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file mode = %o, want 0600", perm)
	}
}

func TestPatchLocalState_BackupCreatedOnceAndNeverOverwritten(t *testing.T) {
	dir, localStateFile := newTestProfile(t, map[string]any{
		"is_glic_eligible": false,
	})
	backupFile := localStateFile + localStateBackupSuffix

	if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
		t.Fatalf("backup should not exist before any modifying run")
	}

	if _, err := PatchLocalState(dir, "120.0.6099.109", false, PatchOptions{}); err != nil {
		t.Fatalf("first PatchLocalState: %v", err)
	}

	backupAfterFirst, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("expected backup to exist after first modifying run: %v", err)
	}
	var backupState map[string]any
	if err := json.Unmarshal(backupAfterFirst, &backupState); err != nil {
		t.Fatalf("unmarshal backup: %v", err)
	}
	if backupState["is_glic_eligible"] != false {
		t.Fatalf("backup should capture the PRE-patch state (is_glic_eligible=false), got %v", backupState["is_glic_eligible"])
	}

	// A second modifying run (this time toggling an AI-download flag —
	// a fake name, not AllAIDownloadFlagNames(), so this test's "second run
	// must genuinely modify something" premise doesn't silently become a
	// no-op if the real list is ever empty, as it legitimately can be; see
	// flags.go) must not touch the backup — it's a one-time snapshot of
	// the pristine file.
	withFakeAIDownloadFlags(t, "fake-flag-for-backup-test")
	if _, err := PatchLocalState(dir, "120.0.6099.109", false, PatchOptions{
		AIDownloadFlags: AllAIDownloadFlagNames(),
	}); err != nil {
		t.Fatalf("second PatchLocalState: %v", err)
	}

	backupAfterSecond, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("read backup after second run: %v", err)
	}
	if string(backupAfterSecond) != string(backupAfterFirst) {
		t.Fatalf("backup was modified by a later run; backups must be captured once and left alone")
	}
}

func TestPatchLocalState_BackupFailureLeavesOriginalFileIntact(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission bits don't block file creation the same way on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission checks")
	}

	dir, localStateFile := newTestProfile(t, map[string]any{
		"is_glic_eligible": false,
	})
	original, err := os.ReadFile(localStateFile)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) // safety net so t.TempDir() cleanup can still remove it

	result, err := PatchLocalState(dir, "120.0.6099.109", false, PatchOptions{})
	if err == nil {
		t.Fatalf("expected an error when the profile directory is not writable (backup cannot be created)")
	}
	if !result.Modified {
		t.Fatalf("expected the returned result to still report the computed change on error, got %+v", result)
	}

	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("restore dir perms: %v", err)
	}
	after, err := os.ReadFile(localStateFile)
	if err != nil {
		t.Fatalf("read after failed write: %v", err)
	}
	if string(after) != string(original) {
		t.Fatalf("Local State was modified despite the write failing; it must be left untouched")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "Local State" && e.Name() != "Last Version" {
			t.Errorf("unexpected leftover file after failed write: %s", e.Name())
		}
	}
}

func TestPatchLocalState_VariationsPermanentConsistencyCountry(t *testing.T) {
	dir, localStateFile := newTestProfile(t, map[string]any{
		"is_glic_eligible":                         true,
		"variations_country":                       "us",
		"variations_permanent_consistency_country": []any{"119.0.0.0", "jp"},
	})

	result, err := PatchLocalState(dir, "120.0.6099.109", false, PatchOptions{})
	if err != nil {
		t.Fatalf("PatchLocalState: %v", err)
	}
	if !result.VariationsPermanentConsistencyCountryWasPatched {
		t.Fatalf("expected VariationsPermanentConsistencyCountryWasPatched, got %+v", result)
	}

	onDisk := readJSONFile(t, localStateFile)
	entries, ok := onDisk["variations_permanent_consistency_country"].([]any)
	if !ok || len(entries) < 2 {
		t.Fatalf("unexpected entries: %v", onDisk["variations_permanent_consistency_country"])
	}
	if entries[0] != "120.0.6099.109" || entries[1] != "us" {
		t.Fatalf("got %v, want [120.0.6099.109 us]", entries)
	}
}

func TestPatchLocalState_VariationsPermanentConsistencyCountryAbsentFieldIsNotCreated(t *testing.T) {
	dir, localStateFile := newTestProfile(t, map[string]any{
		"is_glic_eligible":   true,
		"variations_country": "us",
	})

	result, err := PatchLocalState(dir, "120.0.6099.109", false, PatchOptions{})
	if err != nil {
		t.Fatalf("PatchLocalState: %v", err)
	}
	if result.VariationsPermanentConsistencyCountryWasPatched {
		t.Fatalf("should not report patched when the field never existed")
	}

	onDisk := readJSONFile(t, localStateFile)
	if _, exists := onDisk["variations_permanent_consistency_country"]; exists {
		t.Fatalf("field should not have been created out of thin air")
	}
}
