# go-chrome-ai

English | [中文](README.zh.md)

`go-chrome-ai` is a cross-platform Chrome profile patcher written in Go, with both **CLI** and **GUI** modes.
It helps enable Chrome AI-related features, including **Ask Gemini**, without reinstalling Chrome or recreating your profile.

It enables Chrome AI-related features (such as **Ask Gemini**) by patching local profile state:

- `is_glic_eligible` (recursive) -> `true`
- `variations_country` -> `"us"`
- `variations_permanent_consistency_country` -> `["<last_version>", "us"]` (if field exists and is patchable)

## Requirements

- Go `1.26+`
- Google Chrome installed (Stable / Canary / Dev / Beta)

## Run CLI

```bash
go run ./cmd/cli
```

Flags:

- `-dry-run`: show changes without writing files or killing Chrome
- `-no-restart`: patch but do not restart Chrome

## Run GUI

```bash
go run ./cmd/gui
```

The GUI includes:

- auto-detection of installed Chrome channels
- one-click patch flow
- progress bar
- real-time logs

## Build

```bash
go build -o output/go-chrome-ai-cli ./cmd/cli
go build -o output/go-chrome-ai-gui ./cmd/gui
```

Makefile:

- `make build` (or `make cli` / `make gui`)

All build artifacts are written to `output/`.

## What It Does

1. Detects Chrome user-data directories per OS/channel.
2. Stops running Chrome processes to avoid file locks.
3. Patches `Local State`.
4. Restarts previously running Chrome executables (unless disabled).

## Notes

- Back up Chrome `User Data` if you want a safety net.
- Run with the same OS user that owns the Chrome profile.
- Not affiliated with Google. Use at your own risk.
