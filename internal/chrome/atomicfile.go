package chrome

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomic writes data to path by writing to a temporary file in the
// same directory, syncing it to disk, then renaming it into place. Rename
// within a directory is atomic on every OS this tool targets, so a crash or
// full disk mid-write can never leave path itself truncated or corrupt — the
// temp file is left behind (or cleaned up) instead. The temporary file must
// be created alongside path rather than in a generic temp directory, since
// os.Rename fails across filesystem/device boundaries.
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	// Best-effort cleanup: once the rename below succeeds this is a no-op
	// (the path no longer exists under tmpPath), so the error is ignored.
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close() // the write error above is the one that matters
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close() // the sync error above is the one that matters
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file into place: %w", err)
	}
	return nil
}

// backupFileOnce copies src to src+suffix, but only if that backup does not
// already exist. It gives the user exactly one rollback point captured
// before this tool's first modification, and never clobbers it on later
// runs (so it stays a snapshot of the pristine, untouched file).
func backupFileOnce(src, suffix string) error {
	backupPath := src + suffix
	if _, err := os.Stat(backupPath); err == nil {
		return nil // backup already exists from a previous run; leave it alone
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat backup: %w", err)
	}

	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source for backup: %w", err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source for backup: %w", err)
	}
	if err := writeFileAtomic(backupPath, data, info.Mode().Perm()); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}
	return nil
}
