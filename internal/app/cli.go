package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"go-chrome-ai/internal/chrome"
)

func RunCLI(args []string, stderr io.Writer) int {
	if stderr == nil {
		stderr = os.Stderr
	}

	fs := flag.NewFlagSet("go-chrome-ai", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dryRun := fs.Bool("dry-run", false, "Show what would change without modifying files")
	noRestart := fs.Bool("no-restart", false, "Do not restart Chrome after patching")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	summary, err := chrome.Run(chrome.Options{
		DryRun:    *dryRun,
		NoRestart: *noRestart,
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
