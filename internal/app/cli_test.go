package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/itamaker/go-chrome-ai/internal/meta"
)

// runCLI is a small test harness around RunCLI: it captures stdout/stderr
// into buffers (rather than the real os.Stdout/os.Stderr) so tests can
// assert on exact output and exit code without subprocessing.
func runCLI(args ...string) (stdout, stderr string, code int) {
	var outBuf, errBuf bytes.Buffer
	code = RunCLI(args, &outBuf, &errBuf)
	return outBuf.String(), errBuf.String(), code
}

func TestRunCLI_UnknownDisableFlag(t *testing.T) {
	stdout, stderr, code := runCLI("-disable-ai-download=false", "-disable-flag", "not-a-real-flag")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, `unknown -disable-flag "not-a-real-flag"`) {
		t.Errorf("stderr missing expected error, got: %q", stderr)
	}
	if strings.Contains(stdout, "unknown -disable-flag") {
		t.Errorf("the unknown-flag error must go to stderr, not stdout; stdout was: %q", stdout)
	}
}

func TestRunCLI_Version(t *testing.T) {
	stdout, stderr, code := runCLI("-version")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, meta.Version) {
		t.Errorf("stdout should contain the version (%q), got: %q", meta.Version, stdout)
	}
	if stderr != "" {
		t.Errorf("stderr should be empty on -version, got: %q", stderr)
	}
}

func TestRunCLI_ExplicitHelpGoesToStdout(t *testing.T) {
	// -h/--help is requested output, not an error: it must land on stdout
	// (matching the Unix convention `ls --help` vs `ls --bogus`), exit 0,
	// and leave stderr untouched.
	for _, helpArg := range []string{"-h", "-help", "--help"} {
		stdout, stderr, code := runCLI(helpArg)
		if code != 0 {
			t.Errorf("%s: exit code = %d, want 0", helpArg, code)
		}
		if stderr != "" {
			t.Errorf("%s: stderr should be empty, got: %q", helpArg, stderr)
		}
		if !strings.Contains(stdout, "Usage:") || !strings.Contains(stdout, "-dry-run") {
			t.Errorf("%s: stdout missing expected usage content, got: %q", helpArg, stdout)
		}
	}
}

func TestRunCLI_MalformedFlagGoesToStderr(t *testing.T) {
	// An actual parse error (as opposed to a deliberate -h) must land on
	// stderr and exit 2, even though flag.Parse renders the same usage
	// text for both cases internally.
	stdout, stderr, code := runCLI("-this-flag-does-not-exist")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("stderr should contain usage text, got: %q", stderr)
	}
	if strings.Contains(stdout, "Usage:") {
		t.Errorf("usage text must not also be duplicated onto stdout, got: %q", stdout)
	}
}

func TestRunCLI_DryRunDoesNotErrorAndExitsZero(t *testing.T) {
	// This exercises RunCLI end-to-end (including a real chrome.Run call)
	// in dry-run mode, which is safe regardless of what's actually
	// installed on the test machine: chrome.Run itself returns a
	// documented error when no Chrome installation is found, which is a
	// legitimate, expected outcome here (not a test bug), so both branches
	// are accepted.
	stdout, stderr, code := runCLI("-dry-run")
	if strings.Contains(stdout, "no available Chrome user-data path found") ||
		strings.Contains(stderr, "no available Chrome user-data path found") {
		if code != 1 {
			t.Fatalf("no-Chrome-detected case should exit 1, got %d", code)
		}
		return
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stdout=%q stderr=%q)", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Done. detected=") {
		t.Errorf("stdout should contain the summary line, got: %q", stdout)
	}
}

func TestRunCLI_NilWritersDefaultToOSStreams(t *testing.T) {
	// RunCLI must tolerate nil stdout/stderr (falling back to os.Stdout /
	// os.Stderr) rather than panicking — exercised here via -version,
	// which is side-effect-free.
	code := RunCLI([]string{"-version"}, nil, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}
