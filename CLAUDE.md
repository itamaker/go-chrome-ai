# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`go-chrome-ai` is a cross-platform Go tool that patches a local Chrome profile to (1) enable Chrome AI features like Ask Gemini and (2) optionally block Chrome from silently downloading the on-device Gemini Nano model. It works by editing Chrome's `Local State` JSON file and writing an OS-level managed-policy entry, while Chrome is stopped. Ships as both a CLI and a Fyne GUI.

## Commands

```bash
make build            # build to output/go-chrome-ai (trimpath, stripped, version-stamped via ldflags)
make test             # go test ./...
make vet              # go vet ./...
make lint             # golangci-lint run ./... (config: .golangci.yml)
make cover            # go test ./... -race -coverprofile=coverage.out; go tool cover -func
go test ./internal/chrome -run TestSyncManagedFlags   # single test
make release-check    # goreleaser check (validate .goreleaser.yaml)
make snapshot         # build release assets into dist/ via goreleaser
make clean            # rm -rf output dist coverage.out

go run ./cmd/go-chrome-ai            # run CLI from source
go run ./cmd/go-chrome-ai gui        # run GUI from source
go run ./cmd/go-chrome-ai -dry-run   # preview without writing or killing Chrome
go run ./cmd/go-chrome-ai -version   # print the ldflags-injected version and exit
```

Requires Go 1.26+. CI (`.github/workflows/ci.yml`) runs on macOS, Linux, and Windows: `go vet`, `golangci-lint`, and `go test -race` with coverage, each scoped per OS — Linux/Windows test only the Fyne-free surface (`internal/chrome`, `internal/app`, `internal/meta`, `cmd/cli`), since `cmd/go-chrome-ai`/`cmd/gui`/`internal/guiapp` need CGO and platform GL libraries this project only ships pre-built for macOS (see Architecture). Only the macOS leg also builds `./cmd/go-chrome-ai`, cross-compiles `./cmd/cli` for linux/windows (amd64+arm64), and runs `goreleaser check`.

## Architecture

The codebase is a thin presentation layer (CLI/GUI) over a single OS-agnostic engine in `internal/chrome`.

**Three entrypoints in `cmd/` map to different release builds — this split is the key thing to understand:**
- `cmd/go-chrome-ai` — the canonical shipped binary. Dispatches subcommands (`gui`, `cli`, `help`) and defaults to CLI. Built **with CGO** for macOS releases so it can include the Fyne GUI.
- `cmd/cli` — CLI-only entry. Built **with `CGO_ENABLED=0`** for the portable Linux/Windows releases (no Fyne, no GUI). This is why prebuilt Linux/Windows binaries are CLI-only.
- `cmd/gui` — GUI-only entry, a dev convenience.

So macOS gets the full `cmd/go-chrome-ai` binary; Linux/Windows get `cmd/cli`. See `.goreleaser.yaml` builds `go-chrome-ai-macos` vs `go-chrome-ai-cli`.

**`chrome.Run(Options, Callbacks) (Summary, error)`** in `internal/chrome/runner.go` is the one orchestrator both frontends call. The engine never imports the frontends. `Callbacks{Log, Progress}` is how the CLI prints lines and the GUI drives its log view + progress bar from the same code path. `Options` carries `DryRun`, `NoRestart`, `AIDownloadFlags` (`[]string` of flag names to force Disabled; any managed flag not listed is reverted), `AIDownloadPolicy` (bool; independent of `AIDownloadFlags`). `Summary` reports `FailedInstallations`/`Errors`/`PolicyError` alongside the counts — `Run` only returns a non-nil `error` itself when *every* detected installation failed outright; a partial failure or a policy failure is surfaced through `Summary` instead, so callers (`RunCLI`) that care about the exit code check both.

The Run sequence: detect installs → shut down Chrome (gracefully: terminate, wait, escalate to kill only if needed — see `process.go`) → patch each profile's `Local State` → apply/remove the Enterprise policy → restart Chrome. If Chrome can't be confirmed stopped, `Run` aborts before touching any files rather than racing a Chrome process that might still overwrite the patch on its own exit.

`internal/chrome` files:
- `detect.go` — per-OS, per-channel (Stable/Canary/Dev/Beta) user-data paths in a `chromePaths` map. Display/detection order is *derived* from that map (`orderedChannelsFor`), not a separately hand-maintained list — add a new channel only to `chromePaths` and it's picked up automatically.
- `process.go` — uses `gopsutil` to find running Chrome processes and stop them so files aren't locked. Each match gets a graceful terminate, a poll-wait (`terminateTimeout`/`terminatePoll`), and only escalates to a forceful kill (with its own wait) if it doesn't exit in time; a process that can't be confirmed stopped fails the whole `ShutdownChrome` call. `isChromeMainProcess` matches only the top-level browser process — exact channel names on macOS (excludes `Google Chrome Helper*` subprocesses), same-name-as-parent filtering elsewhere — and `RestartChrome`/`launch` relaunch via `open -a <bundle>` on macOS so it's the app that comes back, not a bare helper binary.
- `atomicfile.go` — `writeFileAtomic` (temp file + fsync + rename, so a crash mid-write can never truncate the destination) and `backupFileOnce` (a one-time pristine snapshot, e.g. `Local State.go-chrome-ai.bak`, captured before the *first* modification and never overwritten again). Any code that rewrites a file in place in this profile should go through these rather than a bare `os.WriteFile`.
- `patch.go` — reads/parses `Local State` JSON and applies transforms: `is_glic_eligible` set to true **recursively** at every depth, `variations_country` -> `"us"`, `variations_permanent_consistency_country` -> `[lastVersion, "us"]` (only if the field already exists), and optionally disabling AI-download flags. Preserves the file's existing permission mode (a stat failure is an error, not a silent fallback to a wider mode); writes via `writeFileAtomic` after `backupFileOnce`.
- `flags.go` — encodes `chrome://flags` selections into `browser.enabled_labs_experiments`, where each entry is `<flag-name>@<choice>` and `@2` means Disabled. `AvailableAIDownloadFlags` lists the flags this tool manages; `AllAIDownloadFlagNames()` returns their names (the "select all" set). `syncManagedFlags` dedupes/validates the selection itself — it never trusts a caller-supplied name it doesn't manage — and only touches `localState["browser"]` when there's an actual change to write. `DisableAIDownloadActions()` + `GroupDisableAIDownloadActions()` produce the preview data; `format.go`'s `FormatActions`/`FormatSummary` render it (and the post-run summary line) as plain text, shared by CLI and GUI so the two can't drift out of sync the way they previously did by hand-duplicating the rendering.
- `policy.go` — a shared `applyPolicy`/`removePolicy` skeleton (read current state → skip if already satisfied → dry-run early-return → mutate) driven by a per-platform `policyBackend` (`isSet`/`exists`/`write`/`clear`), so the "read failure must not be treated as absent" contract and the dry-run/skip semantics live in exactly one place instead of being copy-pasted per OS.
- `policy_darwin.go` / `policy_linux.go` / `policy_windows.go` / `policy_other.go` — **build-tagged per platform**; each implements `disableAIDownloadPolicyBackend() policyBackend` and `policyStorageDescription(applying bool) string` for its OS (macOS `defaults`, Linux JSON file under `/etc/opt/chrome/policies/managed/` needing sudo, Windows registry — tries `HKLM` first and falls back to `HKCU` if not elevated, since a *readable* HKLM key doesn't imply a *writable* one). `policy_other.go` is the fallback for any other OS (e.g. freebsd) so the rest of the package still builds there; it reports the policy feature as unsupported rather than failing to compile. Changing policy behavior generally means touching the shared skeleton in `policy.go` plus each backend.

`internal/app/cli.go` — `RunCLI(args []string, stdout, stderr io.Writer) int` does flag parsing (`-dry-run`, `-no-restart`, `-disable-ai-download` default true, `-version`), prints the action preview, calls `chrome.Run`, prints the summary — normal progress to `stdout`, warnings/errors (and `-h`/`--help`/parse-error usage text) routed to the correct stream. `Usage()` and flag definitions (`newFlagSet`) are shared with `cmd/go-chrome-ai`'s `help` dispatch so the two can't drift. `internal/guiapp/gui.go` — Fyne UI calling the same engine (including `-dry-run`/`-no-restart` as checkboxes). `internal/meta` — `RepoURL` plus `Version` (default `"dev"`, ldflags-injected on release builds; see Makefile/`.goreleaser.yaml`).

## Conventions

- Because writing the Enterprise policy makes Chrome show the "managed by your organization" banner, `disable-ai-download` defaults to **on** but is always presented to the user as a preview first. Keep that preview-then-apply pattern when adding destructive actions.
- Adding a new platform behavior (detection path, process matching, or policy store) means touching the platform map / build-tagged file set, not a single function.
- In-place edits to a user's Chrome profile go through `writeFileAtomic` + `backupFileOnce` (`atomicfile.go`), not a bare `os.WriteFile` — this is what makes a crash mid-write non-destructive and gives the user a one-time rollback snapshot. Keep that pattern for any other profile file this tool learns to edit.
- Killing a real Chrome process goes through `ShutdownChrome`'s graceful-terminate-then-wait-then-kill sequence, not a bare `Kill()` — patching `Local State` before Chrome has actually exited is a race Chrome always wins (it overwrites the file with its own in-memory state on exit).
- `RunCLI` writes normal output to its `stdout` param and warnings/errors to `stderr`; don't reach for `fmt.Println`/`os.Stdout` directly inside it — that's what made it untestable before the split.
- Local builds go to `output/`; goreleaser output goes to `dist/`. Both are gitignored.
