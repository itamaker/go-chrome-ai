# go-chrome-ai

[English](README.md) | 中文

`go-chrome-ai` 是一个用 Go 编写的跨平台 Chrome 配置修补工具，同时支持 **CLI** 和 **GUI**。
它可以在不重装 Chrome、不重建用户配置的情况下启用相关 AI 功能（包括 **Ask Gemini**）。

## 截图

![go-chrome-ai GUI](docs/images/go-chrome-ai-gui.png)

它通过修改 Chrome 本地配置来启用相关 AI 功能（如 **Ask Gemini**）：

- 递归将 `is_glic_eligible` 设为 `true`
- 将 `variations_country` 设为 `"us"`
- 将 `variations_permanent_consistency_country` 设为 `["<last_version>", "us"]`（仅当该字段存在且可修改）

## 环境要求

- Go `1.26+`
- 已安装 Google Chrome（Stable / Canary / Dev / Beta）

## 运行 CLI

```bash
go run ./cmd/cli
```

参数：

- `-dry-run`：只显示将修改的内容，不写入文件，也不关闭 Chrome
- `-no-restart`：修补后不重启 Chrome

## 运行 GUI

```bash
go run ./cmd/gui
```

GUI 功能：

- 自动检测已安装的 Chrome 渠道
- 一键修补
- 进度条
- 实时日志

## 构建

```bash
go build -o output/go-chrome-ai-cli ./cmd/cli
go build -o output/go-chrome-ai-gui ./cmd/gui
```

Makefile：

- `make build`（或 `make cli` / `make gui`）

所有构建产物统一输出到 `output/`。

## 执行流程

1. 按系统和渠道检测 Chrome 用户目录。
2. 关闭运行中的 Chrome，避免文件锁。
3. 修改 `Local State`。
4. 重启修补前已运行的 Chrome（可通过参数禁用）。

## 注意事项

- 建议先备份 Chrome `User Data`。
- 请使用拥有该 Chrome 配置的同一系统用户运行。
- 本项目与 Google 无关，风险自担。
