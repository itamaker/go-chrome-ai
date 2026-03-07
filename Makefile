OUT_DIR := output
CLI_BIN := $(OUT_DIR)/go-chrome-ai-cli
GUI_BIN := $(OUT_DIR)/go-chrome-ai-gui

.PHONY: build cli gui clean

build: cli gui

$(OUT_DIR):
	mkdir -p $(OUT_DIR)

cli: $(OUT_DIR)
	go build -trimpath -ldflags="-s -w" -o $(CLI_BIN) ./cmd/cli

gui: $(OUT_DIR)
	go build -trimpath -ldflags="-s -w" -o $(GUI_BIN) ./cmd/gui

clean:
	rm -f $(CLI_BIN) $(GUI_BIN)
