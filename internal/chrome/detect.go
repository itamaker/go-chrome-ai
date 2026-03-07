package chrome

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Install represents one Chrome channel and its user-data path.
type Install struct {
	Channel      string
	UserDataPath string
}

var chromePaths = map[string]map[string]string{
	"windows": {
		"Stable": "~/AppData/Local/Google/Chrome/User Data",
		"Canary": "~/AppData/Local/Google/Chrome SxS/User Data",
		"Dev":    "~/AppData/Local/Google/Chrome Dev/User Data",
		"Beta":   "~/AppData/Local/Google/Chrome Beta/User Data",
	},
	"linux": {
		"Stable": "~/.config/google-chrome",
		"Canary": "~/.config/google-chrome-canary",
		"Dev":    "~/.config/google-chrome-unstable",
		"Beta":   "~/.config/google-chrome-beta",
	},
	"darwin": {
		"Stable": "~/Library/Application Support/Google/Chrome",
		"Canary": "~/Library/Application Support/Google/Chrome Canary",
		"Dev":    "~/Library/Application Support/Google/Chrome Dev",
		"Beta":   "~/Library/Application Support/Google/Chrome Beta",
	},
}

// DetectInstallations returns the Chrome channels present on this machine.
func DetectInstallations() ([]Install, error) {
	platform := runtime.GOOS
	channelPaths, ok := chromePaths[platform]
	if !ok {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	orderedChannels := []string{"Stable", "Canary", "Dev", "Beta"}
	installs := make([]Install, 0, len(orderedChannels))

	for _, channel := range orderedChannels {
		rawPath, exists := channelPaths[channel]
		if !exists {
			continue
		}
		resolved, err := expandUserPath(rawPath)
		if err != nil {
			continue
		}
		if pathExists(resolved) {
			installs = append(installs, Install{
				Channel:      channel,
				UserDataPath: resolved,
			})
		}
	}

	return installs, nil
}

func expandUserPath(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[2:])
	}
	return filepath.Abs(p)
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
