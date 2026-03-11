# Homebrew Cask (Tap)

To support `brew install --cask`, publish the cask in your tap repo [`itamaker/homebrew-tap`](https://github.com/itamaker/homebrew-tap):

- [`Casks/go-chrome-ai.rb`](https://github.com/itamaker/homebrew-tap/blob/main/Casks/go-chrome-ai.rb)

## Release asset naming

Upload these files to each GitHub Release:

- `go-chrome-ai-darwin-arm64.tar.gz`
- `go-chrome-ai-darwin-amd64.tar.gz`

You can generate them via:

```bash
make snapshot
```

Each archive should contain:

- `go-chrome-ai`

Then update the cask:

```bash
./scripts/render-homebrew-cask.sh --owner itamaker --version v1.0.3 > /path/to/homebrew-tap/Casks/go-chrome-ai.rb
```

## User install

```bash
brew tap itamaker/tap
brew install --cask go-chrome-ai
```
