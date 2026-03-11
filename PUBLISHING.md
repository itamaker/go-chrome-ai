# Publishing

## CI

The repository includes `.github/workflows/ci.yml` to run:

- `go test ./...`
- `go build ./cmd/go-chrome-ai`
- `goreleaser check`

## Release

Tagging a semantic version triggers `.github/workflows/release.yml`.

```bash
git tag v1.0.3
git push origin v1.0.3
```

That workflow publishes release archives and `SHA256SUMS` through GoReleaser.

Published macOS assets keep the current install-compatible names:

- `go-chrome-ai-darwin-arm64.tar.gz`
- `go-chrome-ai-darwin-amd64.tar.gz`
- `SHA256SUMS`

## Homebrew Cask

After the release is live:

```bash
./scripts/render-homebrew-cask.sh --owner itamaker --version v1.0.3 > /path/to/homebrew-tap/Casks/go-chrome-ai.rb
```

Commit the rendered file to `https://github.com/itamaker/homebrew-tap` as `Casks/go-chrome-ai.rb`.

Users can then install with:

```bash
brew tap itamaker/tap https://github.com/itamaker/homebrew-tap
brew install --cask go-chrome-ai
```
