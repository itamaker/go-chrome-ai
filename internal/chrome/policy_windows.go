//go:build windows

package chrome

import (
	"fmt"
	"os/exec"
	"strings"
)

// On Windows, Chrome reads managed policies from
// HKLM\Software\Policies\Google\Chrome (machine-wide) or the same path
// under HKCU (per-user). Writing HKLM requires an elevated process; this
// tool tries HKLM first and falls back to HKCU on failure rather than
// guessing elevation from a prior read, since a successful *read* of HKLM
// does not imply a *write* will succeed — policy keys are commonly
// readable by any user but writable only by admins, which was the source
// of a real bug here (see writeWindowsPolicy/disableAIDownloadPolicyBackend).
const winRegPath = `Software\Policies\Google\Chrome`

var windowsHives = []string{"HKLM", "HKCU"}

func policyStorageDescription(applying bool) string {
	if applying {
		return `Windows: HKLM\` + winRegPath + `\` + GenAIPolicyName + ` = 1 (REG_DWORD; falls back to HKCU if not running elevated)`
	}
	return `Windows: delete ` + GenAIPolicyName + ` from HKLM\` + winRegPath + ` or HKCU\` + winRegPath
}

func windowsPolicyLocation(hive string) string {
	return fmt.Sprintf(`%s\%s\%s`, hive, winRegPath, GenAIPolicyName)
}

func disableAIDownloadPolicyBackend() policyBackend {
	previewLocation := fmt.Sprintf(`HKLM or HKCU \%s\%s`, winRegPath, GenAIPolicyName)

	return policyBackend{
		location: previewLocation,
		isSet: func() (bool, error) {
			_, value, found, err := findWindowsPolicy()
			if err != nil {
				return false, err
			}
			return found && value == "0x1", nil
		},
		exists: func() (bool, error) {
			_, _, found, err := findWindowsPolicy()
			return found, err
		},
		write: func() (string, error) {
			if err := writeWindowsPolicy("HKLM"); err == nil {
				return windowsPolicyLocation("HKLM"), nil
			} else if hkcuErr := writeWindowsPolicy("HKCU"); hkcuErr == nil {
				return windowsPolicyLocation("HKCU"), nil
			} else {
				return "", fmt.Errorf("reg add failed on both HKLM (%v) and HKCU (%v)", err, hkcuErr)
			}
		},
		clear: func() (string, error) {
			hive, _, found, err := findWindowsPolicy()
			if err != nil {
				return "", err
			}
			if !found {
				return previewLocation, nil // nothing to clear; exists() already gates this in practice
			}
			if err := deleteWindowsPolicy(hive); err != nil {
				return "", err
			}
			return windowsPolicyLocation(hive), nil
		},
	}
}

// findWindowsPolicy checks HKLM then HKCU (HKLM's precedence order for
// Chrome policy resolution) and returns the first hive where the value is
// present at all, alongside its raw value. found is false (with a nil
// error) when neither hive has it; a non-nil error means a query itself
// failed for some other reason (e.g. `reg` not on PATH) and must not be
// treated as "not set".
func findWindowsPolicy() (hive, value string, found bool, err error) {
	for _, h := range windowsHives {
		v, ok, err := readWindowsPolicy(h)
		if err != nil {
			return "", "", false, err
		}
		if ok {
			return h, v, true, nil
		}
	}
	return "", "", false, nil
}

func writeWindowsPolicy(hive string) error {
	cmd := exec.Command(
		"reg", "add", fmt.Sprintf(`%s\%s`, hive, winRegPath),
		"/v", GenAIPolicyName,
		"/t", "REG_DWORD",
		"/d", "1",
		"/f",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("reg add %s failed: %w: %s", hive, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func deleteWindowsPolicy(hive string) error {
	cmd := exec.Command(
		"reg", "delete", fmt.Sprintf(`%s\%s`, hive, winRegPath),
		"/v", GenAIPolicyName,
		"/f",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("reg delete %s failed: %w: %s", hive, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// readWindowsPolicy reads the current value of GenAIPolicyName from the
// given hive. ok is false (with a nil error) when the key/value simply
// doesn't exist; a non-nil error means the query itself failed for some
// other reason (e.g. `reg` not on PATH) and must not be treated as "not
// set" — conflating the two previously reported real failures to the user
// as a misleading "already absent".
func readWindowsPolicy(hive string) (value string, ok bool, err error) {
	out, err := exec.Command(
		"reg", "query", fmt.Sprintf(`%s\%s`, hive, winRegPath),
		"/v", GenAIPolicyName,
	).Output()
	if err != nil {
		if _, isExit := err.(*exec.ExitError); isExit {
			// `reg query` exits non-zero when the key or value is absent —
			// that's "not set", not a failure to determine state.
			return "", false, nil
		}
		return "", false, err
	}
	v, found := parseRegDWORDValue(string(out))
	return v, found, nil
}

// parseRegDWORDValue extracts a REG_DWORD value (e.g. "0x1") from `reg
// query /v` output by scanning line by line and taking the last field of
// the first line that mentions REG_DWORD. The previous implementation
// searched the whole multi-line blob for the substring "REG_DWORD" and
// took everything after it, which would swallow unrelated text from
// subsequent lines into the "value" and break idempotence (it would keep
// deciding the policy wasn't set to 1 and rewrite it every run).
func parseRegDWORDValue(output string) (string, bool) {
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "REG_DWORD") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		return fields[len(fields)-1], true
	}
	return "", false
}
