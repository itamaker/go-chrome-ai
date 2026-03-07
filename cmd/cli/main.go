package main

import (
	"flag"
	"fmt"
	"os"

	"go-chrome-ai/internal/chrome"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Show what would change without modifying files")
	noRestart := flag.Bool("no-restart", false, "Do not restart Chrome after patching")
	flag.Parse()

	summary, err := chrome.Run(chrome.Options{
		DryRun:    *dryRun,
		NoRestart: *noRestart,
	}, chrome.Callbacks{
		Log: func(message string) {
			fmt.Println(message)
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	fmt.Printf(
		"Done. detected=%d patched=%d skipped=%d restarted=%d\n",
		summary.DetectedInstallations,
		summary.PatchedInstallations,
		summary.SkippedInstallations,
		summary.RestartedExecutables,
	)
}
