BINARY := go-chrome-ai

.PHONY: build test release-check snapshot clean

build:
	mkdir -p output
	go build -trimpath -ldflags="-s -w" -o output/$(BINARY) ./cmd/go-chrome-ai

test:
	go test ./...

release-check:
	goreleaser check

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf output dist
