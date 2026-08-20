package chrome

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// These are vars rather than consts so tests can shrink them to exercise
// stopProcess's wait/escalate logic without burning the real multi-second
// timeouts (see withShortProcessTimeouts in process_test.go). Production
// behavior is unaffected — nothing else in this package reassigns them.
var (
	// terminateTimeout is how long ShutdownChrome waits for a process to
	// exit on its own after a graceful terminate (SIGTERM on POSIX) before
	// escalating to a forceful kill.
	terminateTimeout = 10 * time.Second
	// terminatePoll is how often ShutdownChrome checks whether a process
	// has exited yet while waiting.
	terminatePoll = 200 * time.Millisecond
	// killTimeout is how long ShutdownChrome waits for a process to
	// disappear after a forceful kill before giving up on it.
	killTimeout = 2 * time.Second
)

// ShutdownChrome stops every running main Chrome process (across all
// channels) so their user-data files are not locked while this tool patches
// them. Each process is asked to exit gracefully first and only killed
// forcefully if it does not, and ShutdownChrome waits for the process to
// actually disappear before returning — patching Local State while Chrome
// is still alive is unsafe, since Chrome overwrites its own copy of that
// file on exit and would silently discard the patch.
//
// Only top-level browser processes are targeted (see isChromeMainProcess);
// helper/renderer/GPU subprocesses are left for Chrome's own process
// management to tear down, since killing them directly races the browser
// and (on macOS in particular) they are not meaningfully "restartable" on
// their own.
//
// It returns the executable paths of every browser process it stopped (for
// RestartChrome) and a non-nil error if any matched process could not be
// confirmed stopped. Callers should treat a non-nil error as unsafe to
// proceed past — some Chrome process may still be running.
func ShutdownChrome(dryRun bool) ([]string, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	terminated := make(map[string]struct{})
	var failures []string

	for _, proc := range procs {
		name, err := proc.Name()
		if err != nil || !isChromeMainProcess(name) {
			continue
		}

		running, err := proc.IsRunning()
		if err != nil || !running {
			continue
		}

		// On Linux/Windows, Chrome's subprocesses share the same process
		// name as the main browser; skip anything whose parent is also a
		// same-named Chrome process so only the top-level process (which
		// owns and tears down its own children) is targeted.
		if parent, err := proc.Parent(); err == nil && parent != nil {
			if parentName, err := parent.Name(); err == nil && parentName == name {
				continue
			}
		}

		exePath, _ := proc.Exe()

		if dryRun {
			if exePath != "" {
				terminated[exePath] = struct{}{}
			}
			continue
		}

		if err := stopProcess(proc); err != nil {
			label := exePath
			if label == "" {
				label = name
			}
			failures = append(failures, fmt.Sprintf("%s: %v", label, err))
			continue
		}

		if exePath != "" {
			terminated[exePath] = struct{}{}
		}
	}

	paths := make([]string, 0, len(terminated))
	for p := range terminated {
		paths = append(paths, p)
	}

	if len(failures) > 0 {
		return paths, fmt.Errorf("failed to stop %d Chrome process(es): %s", len(failures), strings.Join(failures, "; "))
	}
	return paths, nil
}

// stopProcess asks proc to exit gracefully and waits for it to actually
// disappear, escalating to a forceful kill if it does not exit in time.
//
// Chrome shows a "didn't shut down correctly" restore-pages prompt on the
// next launch after being stopped this way (profile.exit_type = "Crashed"
// in its Preferences, despite no crash dump ever being written and the
// process having exited cleanly in response to the signal). This was
// investigated — tried routing through an AppleEvent quit on macOS
// (`tell application "Google Chrome" to quit`, the same path a real Cmd+Q
// takes) on the theory that SIGTERM bypasses Chrome's
// applicationShouldTerminate: handling — but confirmed live, isolated from
// this tool entirely, that a plain AppleEvent quit produces the exact same
// exit_type = "Crashed" result. So whatever actually controls that marking
// isn't the signal-vs-AppleEvent distinction; still unclear what is. Left
// as a known, cosmetic, non-data-affecting limitation (tabs/session are
// still restorable, nothing is lost) rather than adding an unproven
// workaround.
func stopProcess(proc *process.Process) error {
	if err := proc.Terminate(); err != nil {
		// Some platforms/process states don't support a graceful signal;
		// fall back to a hard kill instead of failing outright.
		return killAndWait(proc)
	}
	if waitGone(proc, terminateTimeout) {
		return nil
	}
	return killAndWait(proc)
}

func killAndWait(proc *process.Process) error {
	if err := proc.Kill(); err != nil {
		// The process may have exited between our last check and this
		// call; only treat Kill's error as fatal if it's really still running.
		if running, rerr := proc.IsRunning(); rerr == nil && !running {
			return nil
		}
		return err
	}
	if waitGone(proc, killTimeout) {
		return nil
	}
	return fmt.Errorf("still running after kill")
}

// waitGone polls proc until it is no longer running or timeout elapses,
// returning whether it confirmed the process gone.
func waitGone(proc *process.Process, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if running, err := proc.IsRunning(); err != nil || !running {
			return true
		}
		time.Sleep(terminatePoll)
	}
	running, err := proc.IsRunning()
	return err == nil && !running
}

// RestartChrome relaunches every executable path previously returned by
// ShutdownChrome, deduplicating repeats. It returns how many executables it
// successfully started and a non-nil error describing any that failed to
// launch.
func RestartChrome(executablePaths []string) (started int, err error) {
	seen := make(map[string]struct{}, len(executablePaths))
	var failures []string

	for _, exePath := range executablePaths {
		if exePath == "" {
			continue
		}
		if _, ok := seen[exePath]; ok {
			continue
		}
		seen[exePath] = struct{}{}

		if err := launch(exePath); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", exePath, err))
			continue
		}
		started++
	}

	if len(failures) > 0 {
		return started, fmt.Errorf("failed to restart %d executable(s): %s", len(failures), strings.Join(failures, "; "))
	}
	return started, nil
}

// launch starts exePath as a new detached process. On macOS it goes through
// `open -a <bundle>` when exePath resolves inside a .app bundle, which gives
// correct app-activation semantics (a plain exec of the binary inside
// Contents/MacOS launches the process but not "the app" as Finder/Dock see
// it, and is wrong for anything but the single-binary case).
func launch(exePath string) error {
	if runtime.GOOS == "darwin" {
		if bundle := macAppBundlePath(exePath); bundle != "" {
			return startDetached(exec.Command("open", "-a", bundle))
		}
	}
	return startDetached(exec.Command(exePath))
}

func startDetached(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	// Release rather than Wait: this tool does not manage Chrome's
	// lifecycle after restarting it, and holding the *os.Process around
	// would leak resources for however long go-chrome-ai keeps running
	// (notably the GUI, which stays open after a run completes).
	return cmd.Process.Release()
}

// macAppBundlePath walks up from a binary inside a macOS .app bundle (e.g.
// ".../Google Chrome.app/Contents/MacOS/Google Chrome") to the bundle root.
// Returns "" if exePath is not inside a .app bundle.
func macAppBundlePath(exePath string) string {
	dir := filepath.Dir(exePath)
	for dir != "/" && dir != "." {
		if strings.HasSuffix(dir, ".app") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// isChromeMainProcess reports whether name is one of Chrome's top-level
// browser process names, as opposed to a helper/renderer/GPU/utility
// subprocess. On macOS those subprocesses have distinct names (e.g. "Google
// Chrome Helper (Renderer)"), so an exact match against the known channel
// names is both correct and sufficient. On Linux/Windows every Chrome
// process — main and subprocess alike — shares the same name, so
// ShutdownChrome separately filters subprocesses out by parentage.
func isChromeMainProcess(name string) bool {
	if runtime.GOOS == "darwin" {
		switch name {
		case "Google Chrome", "Google Chrome Canary", "Google Chrome Dev", "Google Chrome Beta":
			return true
		default:
			return false
		}
	}
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return base == "chrome"
}
