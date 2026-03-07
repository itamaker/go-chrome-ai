OUT_DIR := output
APP_BIN := $(OUT_DIR)/go-chrome-ai
RELEASE_DIR := $(OUT_DIR)/release
APP := go-chrome-ai

.PHONY: build release release-macos release-macos-arm64 release-macos-amd64 checksums clean

build: $(OUT_DIR)
	go build -trimpath -ldflags="-s -w" -o $(APP_BIN) ./cmd/go-chrome-ai

$(OUT_DIR):
	mkdir -p $(OUT_DIR)

$(RELEASE_DIR):
	mkdir -p $(RELEASE_DIR)

release: release-macos checksums

release-macos: release-macos-arm64 release-macos-amd64

release-macos-arm64: $(RELEASE_DIR)
	rm -rf $(RELEASE_DIR)/$(APP)-darwin-arm64 $(RELEASE_DIR)/$(APP)-darwin-arm64.tar.gz
	mkdir -p $(RELEASE_DIR)/$(APP)-darwin-arm64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o $(RELEASE_DIR)/$(APP)-darwin-arm64/$(APP) ./cmd/go-chrome-ai
	tar -C $(RELEASE_DIR)/$(APP)-darwin-arm64 -czf $(RELEASE_DIR)/$(APP)-darwin-arm64.tar.gz $(APP)

release-macos-amd64: $(RELEASE_DIR)
	rm -rf $(RELEASE_DIR)/$(APP)-darwin-amd64 $(RELEASE_DIR)/$(APP)-darwin-amd64.tar.gz
	mkdir -p $(RELEASE_DIR)/$(APP)-darwin-amd64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o $(RELEASE_DIR)/$(APP)-darwin-amd64/$(APP) ./cmd/go-chrome-ai
	tar -C $(RELEASE_DIR)/$(APP)-darwin-amd64 -czf $(RELEASE_DIR)/$(APP)-darwin-amd64.tar.gz $(APP)

checksums: $(RELEASE_DIR)
	cd $(RELEASE_DIR) && shasum -a 256 $(APP)-darwin-arm64.tar.gz $(APP)-darwin-amd64.tar.gz | tee SHA256SUMS

clean:
	rm -rf $(APP_BIN) $(RELEASE_DIR)
