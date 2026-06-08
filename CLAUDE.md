# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`go-chrome-ai` is a cross-platform Go tool that patches a local Chrome profile to (1) enable Chrome AI features like Ask Gemini and (2) optionally block Chrome from silently downloading the on-device Gemini Nano model. It works by editing Chrome's `Local State` JSON file and writing an OS-level managed-policy entry, while Chrome is stopped. Ships as both a CLI and a Fyne GUI.

## Commands

```bash
make build            # build to output/go-chrome-ai (trimpath, stripped)
make test             # go test ./...
go test ./internal/chrome -run TestSetFlagsDisabled   # single test
make release-check    # goreleaser check (validate .goreleaser.yaml)
make snapshot         # build release assets into dist/ via goreleaser
make clean            # rm -rf output dist

go run ./cmd/go-chrome-ai            # run CLI from source
go run ./cmd/go-chrome-ai gui        # run GUI from source
go run ./cmd/go-chrome-ai -dry-run   # preview without writing or killing Chrome
```

Requires Go 1.26+. CI (`.github/workflows/ci.yml`) runs only on macOS, runs `go test ./...`, builds `./cmd/go-chrome-ai`, cross-compiles `./cmd/cli` for linux/windows (amd64+arm64), and runs `goreleaser check`.

## Architecture

The codebase is a thin presentation layer (CLI/GUI) over a single OS-agnostic engine in `internal/chrome`.

**Three entrypoints in `cmd/` map to different release builds — this split is the key thing to understand:**
- `cmd/go-chrome-ai` — the canonical shipped binary. Dispatches subcommands (`gui`, `cli`, `help`) and defaults to CLI. Built **with CGO** for macOS releases so it can include the Fyne GUI.
- `cmd/cli` — CLI-only entry. Built **with `CGO_ENABLED=0`** for the portable Linux/Windows releases (no Fyne, no GUI). This is why prebuilt Linux/Windows binaries are CLI-only.
- `cmd/gui` — GUI-only entry, a dev convenience.

So macOS gets the full `cmd/go-chrome-ai` binary; Linux/Windows get `cmd/cli`. See `.goreleaser.yaml` builds `go-chrome-ai-macos` vs `go-chrome-ai-cli`.

**`chrome.Run(Options, Callbacks) (Summary, error)`** in `internal/chrome/runner.go` is the one orchestrator both frontends call. The engine never imports the frontends. `Callbacks{Log, Progress}` is how the CLI prints lines and the GUI drives its log view + progress bar from the same code path. `Options` carries `DryRun`, `NoRestart`, `DisableAIModelDownload`.

The Run sequence: detect installs → shut down Chrome → patch each profile's `Local State` → apply Enterprise policy → restart Chrome.

`internal/chrome` files:
- `detect.go` — per-OS, per-channel (Stable/Canary/Dev/Beta) user-data paths in a `chromePaths` map.
- `process.go` — uses `gopsutil` to find/kill Chrome processes (so files aren't locked) and restart them afterward. macOS matches `Google Chrome*` by name; others match `chrome`.
- `patch.go` — reads/parses `Local State` JSON and applies transforms: `is_glic_eligible` set to true **recursively** at every depth, `variations_country` -> `"us"`, `variations_permanent_consistency_country` -> `[lastVersion, "us"]` (only if the field already exists), and optionally disabling AI-download flags. Preserves the file's existing permission mode on write.
- `flags.go` — encodes `chrome://flags` selections into `browser.enabled_labs_experiments`, where each entry is `<flag-name>@<choice>` and `@2` means Disabled. `AIDownloadFlagNames` lists the flags forced off. `DisableAIDownloadActions()` produces the human-readable preview shown by both CLI and GUI before running.
- `policy.go` + `policy_darwin.go` / `policy_linux.go` / `policy_windows.go` — writes `GenAILocalFoundationalModelSettings=1` to the OS managed-policy store. These are **build-tagged per platform** (macOS `defaults`, Linux JSON file under `/etc/opt/chrome/policies/managed/` needing sudo, Windows registry `HKLM`). Each file implements `applyDisableAIDownloadPolicy` and `policyStorageDescription`; changing policy behavior means editing all three.

`internal/app/cli.go` — `RunCLI` does flag parsing (`-dry-run`, `-no-restart`, `-disable-ai-download` default true), prints the action preview, calls `chrome.Run`, prints the summary. `internal/guiapp/gui.go` — Fyne UI calling the same engine. `internal/meta` — just the `RepoURL` constant used in headers.

## Conventions

- Because writing the Enterprise policy makes Chrome show the "managed by your organization" banner, `disable-ai-download` defaults to **on** but is always presented to the user as a preview first. Keep that preview-then-apply pattern when adding destructive actions.
- Adding a new platform behavior (detection path, process matching, or policy store) means touching the platform map / build-tagged file set, not a single function.
- Local builds go to `output/`; goreleaser output goes to `dist/`. Both are gitignored.
