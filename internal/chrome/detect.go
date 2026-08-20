package chrome

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

// channelPriority orders known channels from most to least stable, purely
// for consistent, predictable display order. It is deliberately not the
// source of truth for which channels exist on a platform — chromePaths is.
// See orderedChannelsFor.
var channelPriority = []string{"Stable", "Canary", "Dev", "Beta"}

// orderedChannelsFor returns every channel key present in
// chromePaths[platform], in channelPriority order, followed by any channels
// not covered by channelPriority (sorted alphabetically). Deriving the list
// from chromePaths itself — rather than maintaining a separate hard-coded
// list — means a channel added to chromePaths for one platform can never be
// silently skipped just because a parallel list wasn't updated to match.
func orderedChannelsFor(platform string) []string {
	channels := chromePaths[platform]
	seen := make(map[string]bool, len(channels))
	ordered := make([]string, 0, len(channels))

	for _, channel := range channelPriority {
		if _, ok := channels[channel]; ok {
			ordered = append(ordered, channel)
			seen[channel] = true
		}
	}

	var rest []string
	for channel := range channels {
		if !seen[channel] {
			rest = append(rest, channel)
		}
	}
	sort.Strings(rest)
	return append(ordered, rest...)
}

// DetectInstallations returns the Chrome channels present on this machine.
func DetectInstallations() ([]Install, error) {
	platform := runtime.GOOS
	channelPaths, ok := chromePaths[platform]
	if !ok {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	installs := make([]Install, 0, len(channelPaths))
	for _, channel := range orderedChannelsFor(platform) {
		resolved, err := expandUserPath(channelPaths[channel])
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
