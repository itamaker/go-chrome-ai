package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

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

	dryRun := fs.Bool("dry-run", false, "Show what would change without modifying files")
	noRestart := fs.Bool("no-restart", false, "Do not restart Chrome after patching")
	disableAI := fs.Bool("disable-ai-download", true,
		"Block on-device Gemini Nano download (use -disable-ai-download=false to skip)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if *disableAI {
		fmt.Println("Disable AI model download - will apply:")
		fmt.Println("  Local flag overrides (chrome://flags):")
		for _, action := range chrome.DisableAIDownloadActions() {
			if action.EnterprisePolicy {
				continue
			}
			fmt.Println("    - " + action.Label)
		}
		fmt.Println("  Enterprise policy (chrome://policy):")
		for _, action := range chrome.DisableAIDownloadActions() {
			if !action.EnterprisePolicy {
				continue
			}
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
		DryRun:                 *dryRun,
		NoRestart:              *noRestart,
		DisableAIModelDownload: *disableAI,
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
