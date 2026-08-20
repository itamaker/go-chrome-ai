package meta

// RepoURL is the canonical source repository for this build.
const RepoURL = "https://github.com/itamaker/go-chrome-ai"

// Version is this build's version string. It defaults to "dev" for local
// `go build`/`go run` builds; release builds inject the real value via
// `-ldflags "-X github.com/itamaker/go-chrome-ai/internal/meta.Version=vX.Y.Z"`
// (see Makefile and .goreleaser.yaml).
var Version = "dev"
