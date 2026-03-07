# go-chrome-ai

<!-- ALL-CONTRIBUTORS-BADGE:START - Do not remove or modify this section -->
[![All Contributors](https://img.shields.io/badge/all_contributors-1-orange.svg?style=flat-square)](#contributors-)
<!-- ALL-CONTRIBUTORS-BADGE:END -->

English | [中文](README.zh.md)

`go-chrome-ai` is a cross-platform Chrome profile patcher written in Go, with both **CLI** and **GUI** modes.
It helps enable Chrome AI-related features, including **Ask Gemini**, without reinstalling Chrome or recreating your profile.

## Screenshot

![go-chrome-ai GUI](docs/images/go-chrome-ai-gui.png)

## Quickstart

### Installing and running go-chrome-ai

Install with your preferred method:

```bash
# Build from source
make build
```

```bash
# Or install via Homebrew (custom tap)
brew tap itamaker/tap
brew install --cask go-chrome-ai
```

Then run:

```bash
go-chrome-ai        # CLI mode
go-chrome-ai gui    # GUI mode
```

<details>
<summary>You can also download binaries from GitHub Releases.</summary>

Current release archives:

- macOS (Apple Silicon/arm64): `go-chrome-ai-darwin-arm64.tar.gz`
- macOS (Intel/x86_64): `go-chrome-ai-darwin-amd64.tar.gz`

Each archive contains a single executable: `go-chrome-ai`.

</details>

It enables Chrome AI-related features (such as **Ask Gemini**) by patching local profile state:

- `is_glic_eligible` (recursive) -> `true`
- `variations_country` -> `"us"`
- `variations_permanent_consistency_country` -> `["<last_version>", "us"]` (if field exists and is patchable)

## Requirements

- Go `1.26+`
- Google Chrome installed (Stable / Canary / Dev / Beta)

## Run CLI

```bash
go run ./cmd/go-chrome-ai
```

Flags:

- `-dry-run`: show changes without writing files or killing Chrome
- `-no-restart`: patch but do not restart Chrome

## Run GUI

```bash
go run ./cmd/go-chrome-ai gui
```

The GUI includes:

- auto-detection of installed Chrome channels
- one-click patch flow
- progress bar
- real-time logs

## Build

```bash
go build -o output/go-chrome-ai ./cmd/go-chrome-ai
```

Makefile:

- `make build`
- `make release` to generate Homebrew cask release assets in `output/release/`

All build artifacts are written to `output/`.

Installed binary usage:

```bash
go-chrome-ai        # CLI mode
go-chrome-ai gui    # GUI mode
```

## What It Does

1. Detects Chrome user-data directories per OS/channel.
2. Stops running Chrome processes to avoid file locks.
3. Patches `Local State`.
4. Restarts previously running Chrome executables (unless disabled).

## Notes

- Back up Chrome `User Data` if you want a safety net.
- Run with the same OS user that owns the Chrome profile.
- Not affiliated with Google. Use at your own risk.

## Contributors ✨

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tr>
    <td align="center"><a href="https://github.com/itamaker"><img src="https://github.com/itamaker.png?size=100" width="100px;" alt="itamaker"/><br /><sub><b>Zhaoyang Jia</b></sub></a><br /><a href="https://github.com/itamaker/go-chrome-ai/commits?author=itamaker" title="Code">💻</a> <a href="https://github.com/itamaker/go-chrome-ai/commits?author=itamaker" title="Documentation">📖</a></td>
  </tr>
</table>
<!-- markdownlint-enable -->
<!-- prettier-ignore-end -->
<!-- ALL-CONTRIBUTORS-LIST:END -->
