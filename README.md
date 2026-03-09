# go-chrome-ai

[![All Contributors](https://img.shields.io/badge/all_contributors-1-orange.svg?style=flat-square)](#contributors-)

English | [中文](docs/README.zh.md)

`go-chrome-ai` is a cross-platform Chrome profile patcher written in Go, with both **CLI** and **GUI** modes.
It helps enable Chrome AI-related features, including **Ask Gemini**, without reinstalling Chrome or recreating your profile.

## Support

[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-FFDD00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/amaker)

## Screenshot

![go-chrome-ai GUI](docs/images/go-chrome-ai-gui.png)

## Quickstart

### Install

Install with your preferred method:

```bash
# Install the latest macOS release
curl -fsSL https://raw.githubusercontent.com/itamaker/go-chrome-ai/main/scripts/install.sh | sh
```

```bash
# Or install via Homebrew (custom tap)
brew tap itamaker/tap
brew install --cask go-chrome-ai
```

<details>
<summary>You can also download binaries from <a href="https://github.com/itamaker/go-chrome-ai/releases">GitHub Releases</a>.</summary>

Current release archives:

- macOS (Apple Silicon/arm64): `go-chrome-ai-darwin-arm64.tar.gz`
- macOS (Intel/x86_64): `go-chrome-ai-darwin-amd64.tar.gz`

Each archive contains a single executable: `go-chrome-ai`.

</details>

The install script also accepts:

```bash
# Install a specific release into a custom directory
curl -fsSL https://raw.githubusercontent.com/itamaker/go-chrome-ai/main/scripts/install.sh | VERSION=v1.0.1 INSTALL_DIR=$HOME/bin sh
```

Set `SKIP_PATH_SETUP=1` if you do not want the installer to edit your shell profile.

### First Run

Run:

```bash
go-chrome-ai        # CLI mode
go-chrome-ai gui    # GUI mode
```

On some macOS systems, Gatekeeper may block first launch for downloaded binaries. If that happens, run:

```bash
xattr -d com.apple.quarantine $(which go-chrome-ai)
```

Typical warning:

> Apple could not verify “go-chrome-ai” is free of malware that may harm your Mac or compromise your privacy.

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

## Build From Source

```bash
make build
```

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

## Acknowledgements

[![Built with OpenAI Codex](https://img.shields.io/badge/Built%20with-OpenAI%20Codex-10A37F?style=for-the-badge&logo=openai&logoColor=white)](https://chatgpt.com/codex)

Special thanks to **OpenAI Codex** for assisting with parts of the implementation of this project.

## Contributors ✨

| [![Zhaoyang Jia][avatar-zhaoyang]][author-zhaoyang] |
| --- |
| [Zhaoyang Jia][author-zhaoyang] |



[author-zhaoyang]: https://github.com/itamaker
[avatar-zhaoyang]: https://images.weserv.nl/?url=https://github.com/itamaker.png&h=120&w=120&fit=cover&mask=circle&maxage=7d
