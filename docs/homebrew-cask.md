# Homebrew Cask (Tap)

To support `brew install --cask`, publish a cask in your tap repo (for example `itamaker/homebrew-tap`):

- `Casks/go-chrome-ai.rb` (use [packaging/homebrew/Casks/go-chrome-ai.rb](../packaging/homebrew/Casks/go-chrome-ai.rb) as template)

## Release asset naming

Upload these files to each GitHub Release:

- `go-chrome-ai-darwin-arm64.tar.gz`
- `go-chrome-ai-darwin-amd64.tar.gz`

You can generate them via:

```bash
make release
```

Each archive should contain:

- `go-chrome-ai`

Then update the cask:

1. `version`
2. arm64/amd64 `sha256`

## User install

```bash
brew tap itamaker/tap
brew install --cask go-chrome-ai
```
