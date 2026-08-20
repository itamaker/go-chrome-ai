package app

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/itamaker/go-chrome-ai/internal/chrome"
	"github.com/itamaker/go-chrome-ai/internal/meta"
)

// cliFlags holds every RunCLI flag. Defining them in one place (newFlagSet)
// that both RunCLI and Usage draw from means the flag descriptions shown by
// -h/--help/`go-chrome-ai help` can never drift from what RunCLI actually
// implements — they previously were a separate, hand-maintained copy in
// cmd/go-chrome-ai/main.go that had already drifted from the real flag
// semantics.
type cliFlags struct {
	version          *bool
	dryRun           *bool
	noRestart        *bool
	selectAllAIFlags *bool
	selectedFlagName *[]string
	applyPolicy      *bool
}

func newFlagSet(name string, output io.Writer, availableFlagNames []string) (*flag.FlagSet, *cliFlags) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(output)
	// Route every help path (explicit "help"/-h/--help dispatch in main,
	// and flag.ErrHelp triggered mid-parse by an unrecognized/misplaced
	// -h) through the same rendering, instead of flag's bare
	// PrintDefaults-only default.
	fs.Usage = func() { Usage(output) }

	var f cliFlags
	f.version = fs.Bool("version", false, "Print the version and exit")
	f.dryRun = fs.Bool("dry-run", false, "Show what would change without modifying files")
	f.noRestart = fs.Bool("no-restart", false, "Do not restart Chrome after patching")
	flagsClause := "disabling every known chrome://flags entry (currently none — see -disable-flag)"
	if len(availableFlagNames) > 0 {
		flagsClause = "disabling every known chrome://flags entry (" + strings.Join(availableFlagNames, ", ") + ")"
	}
	f.selectAllAIFlags = fs.Bool("disable-ai-download", true,
		"Block on-device Gemini Nano download by "+flagsClause+" plus writing the OS Enterprise policy. "+
			"Use -disable-ai-download=false to pick individual flags with -disable-flag and/or "+
			"the policy with -disable-ai-policy instead.")
	var selected []string
	fs.Func("disable-flag",
		"Individual chrome://flags entry to disable (repeatable); only takes effect when -disable-ai-download=false",
		func(v string) error {
			selected = append(selected, v)
			return nil
		})
	f.selectedFlagName = &selected
	f.applyPolicy = fs.Bool("disable-ai-policy", true,
		"Write the "+chrome.GenAIPolicyName+" Enterprise policy (chrome://policy) that also blocks "+
			"on-device AI downloads; causes Chrome to show the \"managed by your organization\" banner. "+
			"Independent of which chrome://flags are disabled. Only takes effect when -disable-ai-download=false.")

	return fs, &f
}

// Usage prints full CLI help — the fixed banner/subcommand summary plus
// every flag's description (the exact same descriptions RunCLI parses
// against, from newFlagSet) — to w. Used by `go-chrome-ai help`/-h/--help
// and by flag.ErrHelp triggered mid-parse.
func Usage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  go-chrome-ai [flags]")
	fmt.Fprintln(w, "  go-chrome-ai gui")
	fmt.Fprintln(w, "  go-chrome-ai cli [flags]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Every run syncs to the current selection: a deselected flag or policy is")
	fmt.Fprintln(w, "actively reverted (chrome://flags reset to default, chrome://policy entry")
	fmt.Fprintln(w, "removed) if this tool previously set it.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")

	// Build a throwaway FlagSet purely to print its registered -name/usage
	// pairs. fs.Usage is cleared first so PrintDefaults can never recurse
	// back into Usage (newFlagSet sets fs.Usage = func(){ Usage(w) } for
	// the RunCLI/flag.ErrHelp path).
	fs, _ := newFlagSet("go-chrome-ai", w, chrome.AllAIDownloadFlagNames())
	fs.Usage = nil
	fs.PrintDefaults()
}

// RunCLI parses args and runs one patch pass, writing normal progress to
// stdout and warnings/errors to stderr. It returns the process exit code:
// 0 on full success, 1 if the run itself failed or any installation/policy
// action failed, 2 on a flag-parsing/usage error.
func RunCLI(args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	fmt.Fprintln(stdout, "go-chrome-ai -", meta.RepoURL)

	availableFlagNames := chrome.AllAIDownloadFlagNames()
	// Usage text is captured into a buffer rather than written straight to
	// stderr, because where it belongs depends on *why* it fired: an
	// explicit -h/--help is requested output and belongs on stdout (the
	// conventional Unix behavior — compare `ls --help` vs `ls --bogus`),
	// but flag.Parse calls the exact same fs.Usage() for a genuinely
	// malformed flag too, and that case is an error and belongs on
	// stderr. The destination isn't knowable until after Parse returns.
	var usageBuf bytes.Buffer
	fs, f := newFlagSet("go-chrome-ai", &usageBuf, availableFlagNames)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			io.Copy(stdout, &usageBuf)
			return 0
		}
		io.Copy(stderr, &usageBuf)
		return 2
	}

	if *f.version {
		fmt.Fprintln(stdout, meta.Version)
		return 0
	}

	var aiDownloadFlags []string
	var aiDownloadPolicy bool
	if *f.selectAllAIFlags {
		aiDownloadFlags = availableFlagNames
		aiDownloadPolicy = true
	} else {
		known := make(map[string]bool, len(availableFlagNames))
		for _, name := range availableFlagNames {
			known[name] = true
		}
		for _, name := range *f.selectedFlagName {
			if !known[name] {
				fmt.Fprintf(stderr, "Error: unknown -disable-flag %q (known: %s)\n",
					name, strings.Join(availableFlagNames, ", "))
				return 2
			}
		}
		aiDownloadFlags = *f.selectedFlagName
		aiDownloadPolicy = *f.applyPolicy
	}

	actions := chrome.DisableAIDownloadActions(aiDownloadFlags, aiDownloadPolicy)
	fmt.Fprintln(stdout, "Chrome AI-download configuration:")
	for _, line := range chrome.FormatActions(actions) {
		fmt.Fprintln(stdout, line)
	}

	summary, err := chrome.Run(chrome.Options{
		DryRun:           *f.dryRun,
		NoRestart:        *f.noRestart,
		AIDownloadFlags:  aiDownloadFlags,
		AIDownloadPolicy: aiDownloadPolicy,
	}, chrome.Callbacks{
		Log: func(message string) {
			// The runner prefixes every warning/error log line with
			// "Warning:" or "Error:" (see internal/chrome/runner.go);
			// route those to stderr so scripts piping stdout don't miss
			// them, and so a plain `2>/dev/null` run stays quiet on
			// success but not on trouble.
			if strings.Contains(message, "Warning:") || strings.Contains(message, "Error:") {
				fmt.Fprintln(stderr, message)
				return
			}
			fmt.Fprintln(stdout, message)
		},
	})
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}

	fmt.Fprintln(stdout, chrome.FormatSummary(summary))

	// Run only returns a hard error when every detected installation
	// failed outright; a partial failure (some installs patched, others
	// didn't) or a policy failure is reported via Summary instead, so it
	// has to be checked separately here to still produce a non-zero exit
	// code — otherwise a run where real work failed would look identical,
	// on the exit code, to one where everything succeeded.
	if summary.FailedInstallations > 0 || summary.PolicyError != nil {
		return 1
	}
	return 0
}
