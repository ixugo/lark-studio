# vdub Makefile
# ─────────────────────────────────────────────────────────

BINARY      := vdub
UI_DIR      := ui
GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)
BUILD_DIR   := build/$(GOOS)_$(GOARCH)
FFMPEG_DIR  := vendor/ffmpeg

VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_BRANCH  := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
GIT_HASH    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -X main.buildVersion=$(VERSION) \
               -X main.gitBranch=$(GIT_BRANCH) \
               -X main.gitHash=$(GIT_HASH) \
               -X main.buildTime=$(BUILD_TIME)
LDFLAGS_REL := $(LDFLAGS) -X main.release=true -s -w

# macOS: evermeet.cx universal2（同时支持 arm64 和 x64）
FFMPEG_MACOS_URL  := https://evermeet.cx/ffmpeg/getrelease/ffmpeg/zip
FFPROBE_MACOS_URL := https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip
# Windows: gyan.dev essentials
FFMPEG_WIN_URL    := https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip

.PHONY: build build-release test e2e e2e-dub clip dev run \
        ffmpeg-macos ffmpeg-windows bundle-macos bundle-windows \
        version clean help

# ─── 帮助 ───────────────────────────────────────────────

help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

version: ## 显示版本信息
	@echo "version:  $(VERSION)"
	@echo "branch:   $(GIT_BRANCH)"
	@echo "hash:     $(GIT_HASH)"
	@echo "time:     $(BUILD_TIME)"

# ─── 编译 ───────────────────────────────────────────────

build: ## 编译 Go 引擎（debug 模式）
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) .

build-release: ## 编译 Go 引擎（release 模式，注入版本号+裁符号表）
	go build -ldflags "$(LDFLAGS_REL)" -o $(BUILD_DIR)/$(BINARY) .

build-windows: ## 交叉编译 Windows amd64
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS_REL)" -o build/windows_amd64/$(BINARY).exe .

test: ## 运行全量 Go 测试
	go test ./...

# ─── E2E / CLI 测试 ─────────────────────────────────────

e2e: ## E2E 翻译测试（裁剪 10~30s + whisper + translate + burn）
	go test -tags=integration -run TestIntegration_E2E_Whisper -v -timeout 15m ./internal/core/pipeline/

e2e-dub: ## E2E 配音测试（翻译 + TTS + 混音 + 烧录）
	go test -tags=integration -run TestIntegration_E2E_Dub -v -timeout 15m ./internal/core/pipeline/

clip: build ## CLI 裁剪+翻译（需设 VDUB_LLM_* 环境变量，VIDEO=路径）
	$(BUILD_DIR)/$(BINARY) -run "$(VIDEO)" -ss 10 -to 30 -mode 2

# ─── 开发 ───────────────────────────────────────────────

dev: build ## 开发模式：编译 Go + Flutter debug
	@echo "Go binary: $(BUILD_DIR)/$(BINARY)"
	cd $(UI_DIR) && flutter run -d macos

run: build ## 仅启动 Go 引擎（HTTP 模式）
	$(BUILD_DIR)/$(BINARY)

# ─── FFmpeg 静态包下载 ──────────────────────────────────

ffmpeg-macos: ## 下载 macOS 静态 ffmpeg/ffprobe（不存在时自动下载）
	@mkdir -p $(FFMPEG_DIR)/darwin
	@if [ ! -f $(FFMPEG_DIR)/darwin/ffmpeg ]; then \
		echo "⬇ 下载 ffmpeg (macOS)..."; \
		curl -fSL $(FFMPEG_MACOS_URL) -o /tmp/vdub_ffmpeg.zip && \
		unzip -o /tmp/vdub_ffmpeg.zip -d $(FFMPEG_DIR)/darwin/ && \
		chmod +x $(FFMPEG_DIR)/darwin/ffmpeg && \
		rm -f /tmp/vdub_ffmpeg.zip; \
	else \
		echo "✓ ffmpeg 已存在"; \
	fi
	@if [ ! -f $(FFMPEG_DIR)/darwin/ffprobe ]; then \
		echo "⬇ 下载 ffprobe (macOS)..."; \
		curl -fSL $(FFPROBE_MACOS_URL) -o /tmp/vdub_ffprobe.zip && \
		unzip -o /tmp/vdub_ffprobe.zip -d $(FFMPEG_DIR)/darwin/ && \
		chmod +x $(FFMPEG_DIR)/darwin/ffprobe && \
		rm -f /tmp/vdub_ffprobe.zip; \
	else \
		echo "✓ ffprobe 已存在"; \
	fi

ffmpeg-windows: ## 下载 Windows 静态 ffmpeg/ffprobe（不存在时自动下载）
	@mkdir -p $(FFMPEG_DIR)/windows
	@if [ ! -f $(FFMPEG_DIR)/windows/ffmpeg.exe ]; then \
		echo "⬇ 下载 ffmpeg (Windows)..."; \
		curl -fSL $(FFMPEG_WIN_URL) -o /tmp/vdub_ffmpeg_win.zip && \
		cd /tmp && unzip -o vdub_ffmpeg_win.zip '*/bin/ffmpeg.exe' '*/bin/ffprobe.exe' && \
		find /tmp -name 'ffmpeg.exe' -path '*/bin/*' -exec cp {} $(CURDIR)/$(FFMPEG_DIR)/windows/ffmpeg.exe \; && \
		find /tmp -name 'ffprobe.exe' -path '*/bin/*' -exec cp {} $(CURDIR)/$(FFMPEG_DIR)/windows/ffprobe.exe \; && \
		rm -rf /tmp/vdub_ffmpeg_win.zip /tmp/ffmpeg-*-essentials_build; \
	else \
		echo "✓ ffmpeg.exe 已存在"; \
	fi

# ─── 打包 ───────────────────────────────────────────────

bundle-macos: ffmpeg-macos build-release ## 打包 macOS .app（Go + ffmpeg 内嵌）
	cd $(UI_DIR) && flutter build macos --release
	$(eval APP := $(UI_DIR)/build/macos/Build/Products/Release/vdub_ui.app)
	@mkdir -p "$(APP)/Contents/Resources"
	cp $(BUILD_DIR)/$(BINARY)       "$(APP)/Contents/Resources/$(BINARY)"
	cp $(FFMPEG_DIR)/darwin/ffmpeg   "$(APP)/Contents/Resources/ffmpeg"
	cp $(FFMPEG_DIR)/darwin/ffprobe  "$(APP)/Contents/Resources/ffprobe"
	@echo "✓ $(APP)"

bundle-windows: ffmpeg-windows build-windows ## 打包 Windows（Go + ffmpeg 并入 Flutter Release 目录）
	cd $(UI_DIR) && flutter build windows --release
	$(eval WIN := $(UI_DIR)/build/windows/x64/runner/Release)
	cp build/windows_amd64/$(BINARY).exe  "$(WIN)/$(BINARY).exe"
	cp $(FFMPEG_DIR)/windows/ffmpeg.exe   "$(WIN)/ffmpeg.exe"
	cp $(FFMPEG_DIR)/windows/ffprobe.exe  "$(WIN)/ffprobe.exe"
	@echo "✓ $(WIN)"

# ─── 清理 ───────────────────────────────────────────────

clean: ## 清理构建产物（不删 vendor/ffmpeg）
	rm -rf build/
	cd $(UI_DIR) && flutter clean

clean-all: clean ## 清理全部（含下载的 ffmpeg）
	rm -rf $(FFMPEG_DIR)
