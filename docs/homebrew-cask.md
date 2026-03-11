# Homebrew Cask (Tap)

To support `brew install --cask`, publish the cask in your tap repo [`itamaker/homebrew-tap`](https://github.com/itamaker/homebrew-tap):

- [`Casks/go-chrome-ai.rb`](https://github.com/itamaker/homebrew-tap/blob/main/Casks/go-chrome-ai.rb)

## Release asset naming

Upload these files to each GitHub Release:

- `go-chrome-ai_1.0.5_darwin_arm64.tar.gz`
- `go-chrome-ai_1.0.5_darwin_amd64.tar.gz`
- `go-chrome-ai_1.0.5_linux_arm64.tar.gz`
- `go-chrome-ai_1.0.5_linux_amd64.tar.gz`
- `go-chrome-ai_1.0.5_windows_arm64.zip`
- `go-chrome-ai_1.0.5_windows_amd64.zip`
- `checksums.txt`

The cask only consumes the two macOS archives, but the release can also ship Linux and Windows CLI packages under the same naming pattern.

You can generate them via:

```bash
make snapshot
```

Each archive should contain:

- `go-chrome-ai`

Then update the cask:

```bash
./scripts/render-homebrew-cask.sh --owner itamaker --version v1.0.5 > /path/to/homebrew-tap/Casks/go-chrome-ai.rb
```

## User install

```bash
brew tap itamaker/tap
brew install --cask go-chrome-ai
```
