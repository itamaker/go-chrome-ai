package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/itamaker/go-chrome-ai/internal/chrome"
	"github.com/itamaker/go-chrome-ai/internal/meta"
)

func RunCLI(args []string, stderr io.Writer) int {
	if stderr == nil {
		stderr = os.Stderr
	}

	fmt.Println("go-chrome-ai -", meta.RepoURL)

	fs := flag.NewFlagSet("go-chrome-ai", flag.ContinueOnError)
	fs.SetOutput(stderr)

	availableFlagNames := chrome.AllAIDownloadFlagNames()

	dryRun := fs.Bool("dry-run", false, "Show what would change without modifying files")
	noRestart := fs.Bool("no-restart", false, "Do not restart Chrome after patching")
	selectAllAIFlags := fs.Bool("disable-ai-download", true,
		"Block on-device Gemini Nano download by disabling all known chrome://flags entries "+
			"("+strings.Join(availableFlagNames, ", ")+") plus writing the OS Enterprise policy. "+
			"Use -disable-ai-download=false to pick individual flags with -disable-flag and/or "+
			"the policy with -disable-ai-policy instead.")
	var selectedFlagNames []string
	fs.Func("disable-flag",
		"Individual chrome://flags entry to disable (repeatable); only takes effect when -disable-ai-download=false",
		func(v string) error {
			selectedFlagNames = append(selectedFlagNames, v)
			return nil
		})
	applyPolicy := fs.Bool("disable-ai-policy", true,
		"Write the "+chrome.GenAIPolicyName+" Enterprise policy (chrome://policy) that also blocks "+
			"on-device AI downloads; causes Chrome to show the \"managed by your organization\" banner. "+
			"Independent of which chrome://flags are disabled. Only takes effect when -disable-ai-download=false.")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	var aiDownloadFlags []string
	var aiDownloadPolicy bool
	if *selectAllAIFlags {
		aiDownloadFlags = availableFlagNames
		aiDownloadPolicy = true
	} else {
		known := make(map[string]bool, len(availableFlagNames))
		for _, name := range availableFlagNames {
			known[name] = true
		}
		for _, name := range selectedFlagNames {
			if !known[name] {
				fmt.Fprintf(stderr, "Error: unknown -disable-flag %q (known: %s)\n",
					name, strings.Join(availableFlagNames, ", "))
				return 2
			}
		}
		aiDownloadFlags = selectedFlagNames
		aiDownloadPolicy = *applyPolicy
	}

	actions := chrome.DisableAIDownloadActions(aiDownloadFlags, aiDownloadPolicy)
	applyFlags, revertFlags, policyActions := chrome.GroupDisableAIDownloadActions(actions)
	fmt.Println("Chrome AI-download configuration:")
	if len(applyFlags) > 0 {
		fmt.Println("  Local flag overrides (chrome://flags):")
		for _, action := range applyFlags {
			fmt.Println("    - " + action.Label)
		}
	}
	if len(revertFlags) > 0 {
		fmt.Println("  Local flag resets (chrome://flags):")
		for _, action := range revertFlags {
			fmt.Println("    - " + action.Label)
		}
	}
	if len(policyActions) > 0 {
		fmt.Println("  Enterprise policy (chrome://policy):")
		for _, action := range policyActions {
			fmt.Println("    - " + action.Label)
			if action.Detail != "" {
				fmt.Println("      " + action.Detail)
			}
			if action.PolicyNote != "" {
				fmt.Println("      ! " + action.PolicyNote)
			}
		}
	}

	summary, err := chrome.Run(chrome.Options{
		DryRun:           *dryRun,
		NoRestart:        *noRestart,
		AIDownloadFlags:  aiDownloadFlags,
		AIDownloadPolicy: aiDownloadPolicy,
	}, chrome.Callbacks{
		Log: func(message string) {
			fmt.Println(message)
		},
	})
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}

	fmt.Printf(
		"Done. detected=%d patched=%d skipped=%d restarted=%d\n",
		summary.DetectedInstallations,
		summary.PatchedInstallations,
		summary.SkippedInstallations,
		summary.RestartedExecutables,
	)
	return 0
}
