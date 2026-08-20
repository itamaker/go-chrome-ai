package chrome

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandUserPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir available: %v", err)
	}

	got, err := expandUserPath("~/Library/Application Support/Google/Chrome")
	if err != nil {
		t.Fatalf("expandUserPath: %v", err)
	}
	want := filepath.Join(home, "Library/Application Support/Google/Chrome")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	// A path with no ~/ prefix is left as an absolute path, not touched.
	got2, err := expandUserPath("/etc/opt/chrome")
	if err != nil {
		t.Fatalf("expandUserPath (absolute): %v", err)
	}
	if got2 != "/etc/opt/chrome" {
		t.Fatalf("got %q, want unchanged absolute path", got2)
	}
}

func TestPathExists(t *testing.T) {
	dir := t.TempDir()
	if !pathExists(dir) {
		t.Errorf("pathExists(%q) = false, want true", dir)
	}
	missing := filepath.Join(dir, "does-not-exist")
	if pathExists(missing) {
		t.Errorf("pathExists(%q) = true, want false", missing)
	}
}

func TestDetectInstallations_UnsupportedPlatform(t *testing.T) {
	// DetectInstallations keys off runtime.GOOS via the package-level
	// chromePaths map; every platform this repo builds for (darwin, linux,
	// windows) has an entry, so on those OSes this just confirms it doesn't
	// error and every returned Install has a channel name and a path that
	// really exists.
	installs, err := DetectInstallations()
	if err != nil {
		// Only expected on an OS this tool doesn't target (e.g. running go
		// test on freebsd), matching the "unsupported platform" error path.
		t.Skipf("DetectInstallations returned an error on this platform: %v", err)
	}
	for _, in := range installs {
		if in.Channel == "" {
			t.Errorf("install with empty channel: %+v", in)
		}
		if !pathExists(in.UserDataPath) {
			t.Errorf("install %+v: UserDataPath does not exist", in)
		}
	}
}

func TestOrderedChannelsForCoversChromePathsKeys(t *testing.T) {
	// Guards against the bug class this replaced: a channel added to
	// chromePaths for some platform silently missing from the display
	// order (and therefore never detected) because a separate hard-coded
	// list wasn't updated to match.
	for platform, channels := range chromePaths {
		got := orderedChannelsFor(platform)
		if len(got) != len(channels) {
			t.Errorf("orderedChannelsFor(%q) returned %d channels, chromePaths has %d", platform, len(got), len(channels))
		}
		gotSet := make(map[string]bool, len(got))
		for _, c := range got {
			gotSet[c] = true
		}
		for channel := range channels {
			if !gotSet[channel] {
				t.Errorf("orderedChannelsFor(%q) is missing channel %q present in chromePaths", platform, channel)
			}
		}
	}
}
