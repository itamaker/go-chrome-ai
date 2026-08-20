package chrome

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")

	if err := writeFileAtomic(path, []byte("hello"), 0o640); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o640 {
			t.Fatalf("mode = %o, want 0640", perm)
		}
	}

	// Overwrite: content must be fully replaced, and no temp files left behind.
	if err := writeFileAtomic(path, []byte("world!!"), 0o640); err != nil {
		t.Fatalf("writeFileAtomic overwrite: %v", err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after overwrite: %v", err)
	}
	if string(got) != "world!!" {
		t.Fatalf("got %q after overwrite, want %q", got, "world!!")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("expected exactly one file in dir, got %v", names)
	}
}

func TestWriteFileAtomic_NoDestinationLeftOnFailure(t *testing.T) {
	// Writing into a directory that doesn't exist must fail cleanly (no
	// partial file anywhere) rather than panicking or silently no-op'ing.
	dir := t.TempDir()
	path := filepath.Join(dir, "missing-subdir", "target.txt")

	if err := writeFileAtomic(path, []byte("hello"), 0o640); err == nil {
		t.Fatalf("expected an error writing into a nonexistent directory")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("target should not exist, stat err = %v", err)
	}
}

func TestBackupFileOnce(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(src, []byte("v1"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if err := backupFileOnce(src, ".bak"); err != nil {
		t.Fatalf("backupFileOnce: %v", err)
	}
	backup := src + ".bak"
	got, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(got) != "v1" {
		t.Fatalf("backup content = %q, want %q", got, "v1")
	}

	// Change the source and back up again: the existing backup must survive
	// untouched — it's a one-time snapshot, not a rolling copy.
	if err := os.WriteFile(src, []byte("v2"), 0o644); err != nil {
		t.Fatalf("rewrite source: %v", err)
	}
	if err := backupFileOnce(src, ".bak"); err != nil {
		t.Fatalf("second backupFileOnce: %v", err)
	}
	got, err = os.ReadFile(backup)
	if err != nil {
		t.Fatalf("read backup after second call: %v", err)
	}
	if string(got) != "v1" {
		t.Fatalf("backup content changed to %q, want it to remain %q", got, "v1")
	}
}

func TestBackupFileOnce_MissingSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "does-not-exist.txt")
	if err := backupFileOnce(src, ".bak"); err == nil {
		t.Fatalf("expected error backing up a nonexistent source")
	}
}
