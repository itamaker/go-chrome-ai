package chrome

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

func ShutdownChrome(dryRun bool) ([]string, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	terminated := make(map[string]struct{})

	for _, proc := range procs {
		name, err := proc.Name()
		if err != nil || !isChromeProcess(name) {
			continue
		}

		running, err := proc.IsRunning()
		if err != nil || !running {
			continue
		}

		parent, err := proc.Parent()
		if err == nil && parent != nil {
			parentName, err := parent.Name()
			if err == nil && parentName == name {
				continue
			}
		}

		exePath, _ := proc.Exe()
		if !dryRun {
			if err := proc.Kill(); err != nil {
				continue
			}
		}

		if exePath != "" {
			terminated[exePath] = struct{}{}
		}
	}

	paths := make([]string, 0, len(terminated))
	for p := range terminated {
		paths = append(paths, p)
	}
	return paths, nil
}

func RestartChrome(executablePaths []string) {
	seen := make(map[string]struct{})
	for _, exePath := range executablePaths {
		if exePath == "" {
			continue
		}
		if _, ok := seen[exePath]; ok {
			continue
		}
		seen[exePath] = struct{}{}
		cmd := exec.Command(exePath)
		_ = cmd.Start()
	}
}

func isChromeProcess(name string) bool {
	if runtime.GOOS == "darwin" {
		return strings.HasPrefix(name, "Google Chrome")
	}
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return base == "chrome"
}
