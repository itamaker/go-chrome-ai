package chrome

import (
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// startReapedProcess starts cmd and reaps it in the background as soon as
// it exits, avoiding a lingering zombie that could make a concurrent
// IsRunning() check on its PID flaky/slow. The returned func blocks until
// the process has been fully reaped; callers should run it via t.Cleanup.
func startReapedProcess(t *testing.T, cmd *exec.Cmd) (*process.Process, func()) {
	t.Helper()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %v: %v", cmd.Args, err)
	}
	proc, err := process.NewProcess(int32(cmd.Process.Pid))
	if err != nil {
		t.Fatalf("NewProcess: %v", err)
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait() // reaping the process is the point; exit status is irrelevant here
		close(done)
	}()
	return proc, func() { <-done }
}

// withShortProcessTimeouts shrinks the package-level graceful-shutdown
// timing so tests that exercise stopProcess's wait/escalate logic don't
// have to burn the real (multi-second) production timeouts. Restores the
// originals on test cleanup.
func withShortProcessTimeouts(t *testing.T) {
	t.Helper()
	oldTerminate, oldPoll, oldKill := terminateTimeout, terminatePoll, killTimeout
	terminateTimeout = 300 * time.Millisecond
	terminatePoll = 20 * time.Millisecond
	killTimeout = 2 * time.Second
	t.Cleanup(func() {
		terminateTimeout, terminatePoll, killTimeout = oldTerminate, oldPoll, oldKill
	})
}

func TestStopProcess_GracefulTerminateIsEnough(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a unix sleep helper process")
	}
	withShortProcessTimeouts(t)

	proc, wait := startReapedProcess(t, exec.Command("sleep", "30"))
	t.Cleanup(wait)

	if err := stopProcess(proc); err != nil {
		t.Fatalf("stopProcess: %v", err)
	}
	if running, _ := proc.IsRunning(); running {
		t.Fatalf("process still running after stopProcess")
	}
}

func TestStopProcess_EscalatesToKillWhenTerminateIsIgnored(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a unix shell helper that traps SIGTERM")
	}
	withShortProcessTimeouts(t)

	// The shell traps SIGTERM (ignoring it) then execs into `sleep`,
	// keeping the same PID so there's no separate orphaned child: SIGKILL
	// on this PID must be what actually stops it, exercising stopProcess's
	// escalation path rather than the plain graceful-terminate path.
	proc, wait := startReapedProcess(t, exec.Command("sh", "-c", `trap "" TERM; exec sleep 30`))
	t.Cleanup(wait)

	if err := stopProcess(proc); err != nil {
		t.Fatalf("stopProcess: %v", err)
	}
	if running, _ := proc.IsRunning(); running {
		t.Fatalf("process still running after stopProcess escalation to kill")
	}
}

func TestIsChromeMainProcess(t *testing.T) {
	if runtime.GOOS == "darwin" {
		mainNames := []string{"Google Chrome", "Google Chrome Canary", "Google Chrome Dev", "Google Chrome Beta"}
		for _, name := range mainNames {
			if !isChromeMainProcess(name) {
				t.Errorf("isChromeMainProcess(%q) = false, want true", name)
			}
		}
		// Helper/renderer/GPU subprocesses must NOT match: killing them
		// directly (instead of the main process) races Chrome's own child
		// process management, and "restarting" a helper binary later just
		// launches a process that exits immediately.
		helperNames := []string{
			"Google Chrome Helper",
			"Google Chrome Helper (Renderer)",
			"Google Chrome Helper (GPU)",
			"Google Chrome Helper (Plugin)",
			"Google Chrome Framework",
		}
		for _, name := range helperNames {
			if isChromeMainProcess(name) {
				t.Errorf("isChromeMainProcess(%q) = true, want false (helper process)", name)
			}
		}
		return
	}

	cases := []struct {
		name string
		want bool
	}{
		{"chrome", true},
		{"chrome.exe", true},
		{"Chrome", false}, // case-sensitive: real process name is lowercase
		{"chromium", false},
		{"firefox", false},
	}
	for _, tc := range cases {
		if got := isChromeMainProcess(tc.name); got != tc.want {
			t.Errorf("isChromeMainProcess(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMacAppBundlePath(t *testing.T) {
	cases := []struct {
		exePath string
		want    string
	}{
		{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Google Chrome.app",
		},
		{
			"/Users/me/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/Users/me/Applications/Google Chrome Canary.app",
		},
		{"/usr/local/bin/some-binary", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := macAppBundlePath(tc.exePath); got != tc.want {
			t.Errorf("macAppBundlePath(%q) = %q, want %q", tc.exePath, got, tc.want)
		}
	}
}

func TestRestartChrome_EmptyAndDuplicatePaths(t *testing.T) {
	// executablePaths containing only blanks/duplicates should launch
	// nothing and report zero started, without error.
	started, err := RestartChrome([]string{"", ""})
	if err != nil {
		t.Fatalf("RestartChrome: %v", err)
	}
	if started != 0 {
		t.Fatalf("started = %d, want 0", started)
	}
}
