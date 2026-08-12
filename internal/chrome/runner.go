package chrome

import (
	"errors"
	"fmt"
	"path/filepath"
)

// Options controls runtime behavior.
type Options struct {
	DryRun    bool
	NoRestart bool
	// AIDownloadFlags is the set of chrome://flags entries (from
	// AvailableAIDownloadFlags) to force Disabled. Any managed flag not in
	// this set has its override removed (reverted to Chrome's default) if
	// present.
	AIDownloadFlags []string
	// AIDownloadPolicy, when true, writes the
	// GenAILocalFoundationalModelSettings Enterprise policy; when false, it
	// removes the policy if present (reverting to Chrome's unmanaged
	// default). Independent of AIDownloadFlags.
	AIDownloadPolicy bool
}

// Callbacks allows CLI/GUI to receive status updates.
type Callbacks struct {
	Log      func(string)
	Progress func(int)
}

// Summary is the result of one run.
type Summary struct {
	DetectedInstallations int
	PatchedInstallations  int
	SkippedInstallations  int
	RestartedExecutables  int
	PolicyApplied         bool
	PolicyReverted        bool
	PolicyPath            string
}

func Run(opts Options, cb Callbacks) (Summary, error) {
	logf := cb.Log
	progress := cb.Progress
	if logf == nil {
		logf = func(string) {}
	}
	if progress == nil {
		progress = func(int) {}
	}

	installs, err := DetectInstallations()
	if err != nil {
		return Summary{}, err
	}
	if len(installs) == 0 {
		return Summary{}, errors.New("no available Chrome user-data path found")
	}

	summary := Summary{DetectedInstallations: len(installs)}
	logf(fmt.Sprintf("Detected %d Chrome installation(s)", len(installs)))

	progress(10)
	terminatedChrome, err := ShutdownChrome(opts.DryRun)
	if err != nil {
		logf(fmt.Sprintf("Warning: failed to enumerate Chrome processes: %v", err))
	}
	if len(terminatedChrome) > 0 {
		if opts.DryRun {
			logf("Dry-run: Chrome processes matched but not killed")
		} else {
			logf("Shutdown Chrome")
		}
	}

	total := len(installs)
	for i, install := range installs {
		progress(20 + int(60*float64(i+1)/float64(total)))
		logf(fmt.Sprintf("Patching Chrome %s (%s)", install.Channel, install.UserDataPath))

		lastVersion, err := ReadLastVersion(install.UserDataPath)
		if err != nil {
			logf(fmt.Sprintf("  Warning: missing Last Version file at %s",
				filepath.Join(install.UserDataPath, "Last Version")))
			summary.SkippedInstallations++
			continue
		}

		result, err := PatchLocalState(install.UserDataPath, lastVersion, opts.DryRun, PatchOptions{
			AIDownloadFlags: opts.AIDownloadFlags,
		})
		if err != nil {
			logf(fmt.Sprintf("  Error: failed to patch Local State: %v", err))
			summary.SkippedInstallations++
			continue
		}

		if result.GLICEligiblePatched {
			logf("  Patched is_glic_eligible")
		}
		if result.VariationsCountryPatched {
			logf("  Patched variations_country")
		}
		if result.VariationsPermanentConsistencyCountryWasPatched {
			logf("  Patched variations_permanent_consistency_country")
		}
		for _, name := range result.DisabledFlags {
			logf(fmt.Sprintf("  Disabled chrome://flags/#%s", name))
		}
		for _, name := range result.RevertedFlags {
			logf(fmt.Sprintf("  Reverted chrome://flags/#%s to default", name))
		}

		if result.Modified {
			if opts.DryRun {
				logf("  Dry-run: patch changes detected (not written)")
			} else {
				logf("  Succeeded in patching Local State")
			}
			summary.PatchedInstallations++
		} else {
			logf("  No need to patch Local State")
		}
	}

	if opts.AIDownloadPolicy {
		policy, err := ApplyDisableAIDownloadPolicy(opts.DryRun)
		switch {
		case err != nil:
			logf(fmt.Sprintf("Warning: failed to apply %s policy: %v", GenAIPolicyName, err))
		case policy.Applied:
			summary.PolicyApplied = true
			summary.PolicyPath = policy.Location
			if opts.DryRun {
				logf(fmt.Sprintf("Dry-run: would set %s=1 at %s", GenAIPolicyName, policy.Location))
			} else {
				logf(fmt.Sprintf("Applied %s=1 at %s", GenAIPolicyName, policy.Location))
			}
		default:
			summary.PolicyPath = policy.Location
			logf(fmt.Sprintf("%s already configured (%s)", GenAIPolicyName, policy.Skipped))
		}
	} else {
		policy, err := RemoveDisableAIDownloadPolicy(opts.DryRun)
		switch {
		case err != nil:
			logf(fmt.Sprintf("Warning: failed to remove %s policy: %v", GenAIPolicyName, err))
		case policy.Applied:
			summary.PolicyReverted = true
			summary.PolicyPath = policy.Location
			if opts.DryRun {
				logf(fmt.Sprintf("Dry-run: would remove %s from %s", GenAIPolicyName, policy.Location))
			} else {
				logf(fmt.Sprintf("Removed %s from %s", GenAIPolicyName, policy.Location))
			}
		default:
			summary.PolicyPath = policy.Location
			logf(fmt.Sprintf("%s already absent (%s)", GenAIPolicyName, policy.Skipped))
		}
	}

	progress(90)
	if !opts.NoRestart && !opts.DryRun && len(terminatedChrome) > 0 {
		RestartChrome(terminatedChrome)
		summary.RestartedExecutables = len(terminatedChrome)
		logf("Restart Chrome")
	}

	progress(100)
	return summary, nil
}
