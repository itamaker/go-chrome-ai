# Publishing

## CI

The repository includes `.github/workflows/ci.yml` to run:

- `go test ./...`
- `go build ./cmd/go-chrome-ai`
- `goreleaser check`

## Release

Tagging a semantic version triggers `.github/workflows/release.yml`.

```bash
git tag v1.0.4
git push origin v1.0.4
```

That workflow publishes release archives and `checksums.txt` through GoReleaser.

Published macOS assets follow the same versioned artifact pattern as the other managed repos:

- `go-chrome-ai_1.0.4_darwin_arm64.tar.gz`
- `go-chrome-ai_1.0.4_darwin_amd64.tar.gz`
- `checksums.txt`

## Homebrew Cask

After the release is live:

```bash
./scripts/render-homebrew-cask.sh --owner itamaker --version v1.0.4 > /path/to/homebrew-tap/Casks/go-chrome-ai.rb
```

Commit the rendered file to `https://github.com/itamaker/homebrew-tap` as `Casks/go-chrome-ai.rb`.

Users can then install with:

```bash
brew tap itamaker/tap https://github.com/itamaker/homebrew-tap
brew install --cask go-chrome-ai
```
