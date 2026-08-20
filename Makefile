BINARY := go-chrome-ai
MODULE := github.com/itamaker/go-chrome-ai
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(MODULE)/internal/meta.Version=$(VERSION)

.PHONY: build test vet lint cover release-check snapshot clean

build:
	mkdir -p output
	go build -trimpath -ldflags="$(LDFLAGS)" -o output/$(BINARY) ./cmd/go-chrome-ai

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

cover:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -func=coverage.out

release-check:
	goreleaser check

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf output dist coverage.out
