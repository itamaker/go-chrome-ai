package chrome

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRun_DryRunSmoke runs the full Run() pipeline in dry-run mode against
// whatever real Chrome installations happen to exist on the machine
// running the test (there may be none, e.g. on a bare CI runner). Dry-run
// makes every step read-only — nothing is written and no process is
// killed — so this is safe to run anywhere, and it's the one test that
// exercises Run's actual orchestration rather than its pieces in
// isolation.
func TestRun_DryRunSmoke(t *testing.T) {
	installs, err := DetectInstallations()
	if err != nil {
		t.Fatalf("DetectInstallations: %v", err)
	}

	// Snapshot every Local State file's mtime, plus whether a backup file
	// already exists (a prior *real* run of this tool on the machine
	// running the test would have created one — that's legitimate
	// pre-existing state, not something this test should assume away), so
	// we can assert dry-run truly touched neither.
	type snapshot struct {
		path          string
		modTime       int64
		backupExisted bool
		backupModTime int64
	}
	var before []snapshot
	for _, in := range installs {
		lsPath := filepath.Join(in.UserDataPath, "Local State")
		info, err := os.Stat(lsPath)
		if err != nil {
			continue // this channel has no Local State yet; nothing to snapshot
		}
		snap := snapshot{path: lsPath, modTime: info.ModTime().UnixNano()}
		if bakInfo, err := os.Stat(lsPath + localStateBackupSuffix); err == nil {
			snap.backupExisted = true
			snap.backupModTime = bakInfo.ModTime().UnixNano()
		}
		before = append(before, snap)
	}

	var logs []string
	summary, runErr := Run(Options{DryRun: true, AIDownloadPolicy: false}, Callbacks{
		Log: func(msg string) { logs = append(logs, msg) },
	})

	if len(installs) == 0 {
		// No Chrome on this machine (common on a bare CI runner): Run must
		// report the documented error, not silently succeed.
		if runErr == nil {
			t.Fatalf("expected an error when no Chrome installation is detected")
		}
		return
	}

	if runErr != nil {
		t.Fatalf("Run (dry-run): %v", runErr)
	}
	if summary.DetectedInstallations != len(installs) {
		t.Errorf("DetectedInstallations = %d, want %d", summary.DetectedInstallations, len(installs))
	}
	if summary.RestartedExecutables != 0 {
		t.Errorf("dry-run must never report a restart, got %d", summary.RestartedExecutables)
	}
	if len(logs) == 0 {
		t.Errorf("expected at least one log line")
	}

	for _, snap := range before {
		info, err := os.Stat(snap.path)
		if err != nil {
			t.Errorf("Local State disappeared during dry-run: %s", snap.path)
			continue
		}
		if info.ModTime().UnixNano() != snap.modTime {
			t.Errorf("dry-run modified %s (mtime changed)", snap.path)
		}

		bakInfo, bakErr := os.Stat(snap.path + localStateBackupSuffix)
		switch {
		case !snap.backupExisted && bakErr == nil:
			t.Errorf("dry-run must not create a backup file at %s", snap.path+localStateBackupSuffix)
		case snap.backupExisted && bakErr != nil:
			t.Errorf("a pre-existing backup at %s disappeared during dry-run", snap.path+localStateBackupSuffix)
		case snap.backupExisted && bakErr == nil && bakInfo.ModTime().UnixNano() != snap.backupModTime:
			t.Errorf("dry-run modified the pre-existing backup at %s (mtime changed)", snap.path+localStateBackupSuffix)
		}
	}
}

// TestRun_NoCallbacks confirms Run tolerates a zero-value Callbacks (no Log
// or Progress function set) without panicking, exercising the nil-callback
// defaulting at the top of Run.
func TestRun_NoCallbacks(t *testing.T) {
	installs, err := DetectInstallations()
	if err != nil || len(installs) == 0 {
		t.Skip("no Chrome installation on this machine to exercise Run against")
	}
	if _, err := Run(Options{DryRun: true}, Callbacks{}); err != nil {
		t.Fatalf("Run with zero-value Callbacks: %v", err)
	}
}
