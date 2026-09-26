# vdub Makefile
# ─────────────────────────────────────────────────────────

BINARY      := vdub
FRONTEND_DIR := frontend
GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)
BUILD_DIR   := build/$(GOOS)_$(GOARCH)
FFMPEG_DIR  := vendor/ffmpeg
WHISPER_PREFIX ?= $(shell brew --prefix whisper-cpp 2>/dev/null)
GGML_PREFIX    ?= $(shell brew --prefix ggml 2>/dev/null)
LIBOMP_PREFIX  ?= $(shell brew --prefix libomp 2>/dev/null)

VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_BRANCH  := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
GIT_HASH    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -X main.buildVersion=$(VERSION) \
               -X main.gitBranch=$(GIT_BRANCH) \
               -X main.gitHash=$(GIT_HASH) \
               -X main.buildTime=$(BUILD_TIME)
LDFLAGS_REL := $(LDFLAGS) -X main.release=true -s -w

# macOS arm64: ffmpeg-static 6.1.1（与 SmartSub 同源，43MB vs evermeet 77MB）
FFMPEG_MACOS_URL  := https://github.com/eugeneware/ffmpeg-static/releases/download/b6.1.1/ffmpeg-darwin-arm64.gz
# Windows: gyan.dev essentials
FFMPEG_WIN_URL    := https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip

.PHONY: build build-release test e2e e2e-dub clip dev run \
        ffmpeg-macos ffmpeg-windows whisper-macos bundle-macos bundle-windows \
        bundle/macos/arm64 bundle/windows \
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

build-frontend: ## 编译前端静态资产
	@echo "⚙ 构建前端资产..."
	@cd $(FRONTEND_DIR) && npm run build

build: build-frontend ## 编译 Go + Wails3 一体化桌面应用（debug 模式）
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) .

build-release: build-frontend ## 编译 Go + Wails3 一体化桌面应用（release 模式，注入版本号+裁符号表+trimpath）
	@mkdir -p $(BUILD_DIR)
	go build -trimpath -ldflags "$(LDFLAGS_REL)" -o $(BUILD_DIR)/$(BINARY) .

build-windows: build-frontend ## 交叉编译 Windows amd64
	@mkdir -p build/windows_amd64
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS_REL)" -o build/windows_amd64/$(BINARY).exe .

test: ## 运行全量测试（含 Go 与前端）
	go test ./...
	cd $(FRONTEND_DIR) && npm test

# ─── E2E / CLI 测试 ─────────────────────────────────────

e2e: ## E2E 翻译测试（裁剪 10~30s + whisper + translate + burn）
	go test -tags=integration -run TestIntegration_E2E_Whisper -v -timeout 15m ./internal/core/pipeline/

e2e-dub: ## E2E 配音测试（翻译 + TTS + 混音 + 烧录）
	go test -tags=integration -run TestIntegration_E2E_Dub -v -timeout 15m ./internal/core/pipeline/

clip: build ## CLI 裁剪+翻译（需设 VDUB_LLM_* 环境变量，VIDEO=路径）
	$(BUILD_DIR)/$(BINARY) -run "$(VIDEO)" -ss 10 -to 30 -mode 2

# ─── 开发 ───────────────────────────────────────────────

dev: ## 开发模式：一键启动 Vite 开发服务与 Go 桌面端联动运行
	@if [ ! -f "$(FRONTEND_DIR)/dist/index.html" ]; then $(MAKE) build-frontend; fi
	@echo "🚀 启动开发环境 (Vite 热重载 + Go 桌面客户端)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) .
	@bash -c '\
		cleanup() { kill 0 2>/dev/null || true; exit 0; }; \
		trap cleanup EXIT INT TERM; \
		(cd $(FRONTEND_DIR) && npm run dev) & \
		for i in $$(seq 1 30); do \
			curl -s http://127.0.0.1:5173 >/dev/null 2>&1 && break; \
			sleep 0.2; \
		done; \
		echo "✓ 前端热更服务已就绪，正在拉起桌面端..."; \
		FRONTEND_DEVSERVER_URL=http://127.0.0.1:5173 $(BUILD_DIR)/$(BINARY) \
	'

run: build ## 编译并启动桌面应用（单二进制内嵌模式）
	$(BUILD_DIR)/$(BINARY)

# ─── FFmpeg 静态包下载 ──────────────────────────────────

ffmpeg-macos: ## 下载 macOS 静态 ffmpeg（不存在时自动下载）
	@mkdir -p $(FFMPEG_DIR)/darwin
	@if [ ! -f $(FFMPEG_DIR)/darwin/ffmpeg ]; then \
		echo "⬇ 下载 ffmpeg (macOS arm64)..."; \
		curl -fSL $(FFMPEG_MACOS_URL) | gunzip > $(FFMPEG_DIR)/darwin/ffmpeg && \
		chmod +x $(FFMPEG_DIR)/darwin/ffmpeg; \
	else \
		echo "✓ ffmpeg 已存在"; \
	fi

ffmpeg-windows: ## 下载 Windows 静态 ffmpeg（不存在时自动下载）
	@mkdir -p $(FFMPEG_DIR)/windows
	@if [ ! -f $(FFMPEG_DIR)/windows/ffmpeg.exe ]; then \
		echo "⬇ 下载 ffmpeg (Windows)..."; \
		tmpdir=$$(mktemp -d); \
		curl -fSL $(FFMPEG_WIN_URL) -o $$tmpdir/ffmpeg.zip && \
		unzip -o -q $$tmpdir/ffmpeg.zip -d $$tmpdir && \
		find $$tmpdir -name 'ffmpeg.exe' -path '*/bin/*' -exec cp {} $(CURDIR)/$(FFMPEG_DIR)/windows/ffmpeg.exe \; && \
		rm -rf $$tmpdir; \
	else \
		echo "✓ ffmpeg.exe 已存在"; \
	fi

whisper-macos: ## 安装 macOS whisper.cpp 打包依赖
	@command -v brew >/dev/null || { echo "未找到 Homebrew"; exit 1; }
	@brew list whisper-cpp >/dev/null 2>&1 || brew install whisper-cpp

# ─── 打包 ───────────────────────────────────────────────

DIST_DIR := build/dist

bundle-macos: ffmpeg-macos whisper-macos build-release ## 打包 macOS 一体化应用包（旧别名，仅编译）
	@echo "✓ 构建一体化 macOS 桌面端: $(BUILD_DIR)/$(BINARY)"

bundle-windows: ffmpeg-windows build-windows ## 打包 Windows 一体化应用包（旧别名，仅编译+UPX）
	@command -v upx >/dev/null && { echo "⚙ UPX 压缩 Go 二进制..."; upx --best --lzma build/windows_amd64/$(BINARY).exe; } || echo "⚠ 跳过 UPX（未安装）"
	@echo "✓ 构建一体化 Windows 桌面端: build/windows_amd64/$(BINARY).exe"

bundle/macos/arm64: ffmpeg-macos build-release ## 打包 macOS arm64 dmg（vdub.app 内嵌 ffmpeg，ad-hoc 签名）
	@echo "⚙ 打包 macOS dmg..."
	@bash scripts/bundle-macos.sh "$(VERSION)" "$(BUILD_DIR)/$(BINARY)" "$(FFMPEG_DIR)/darwin/ffmpeg" "$(DIST_DIR)/$(BINARY)_$(VERSION)_macos_arm64.dmg"

bundle/windows: ffmpeg-windows build-windows ## 打包 Windows amd64 zip（vdub.exe + ffmpeg.exe 同目录，解压即用）
	@command -v upx >/dev/null && { echo "⚙ UPX 压缩 Go 二进制..."; upx --best --lzma build/windows_amd64/$(BINARY).exe; } || echo "⚠ 跳过 UPX（未安装）"
	@echo "⚙ 打包 Windows zip..."
	@bash scripts/bundle-windows.sh "$(VERSION)" "build/windows_amd64/$(BINARY).exe" "$(FFMPEG_DIR)/windows/ffmpeg.exe" "$(DIST_DIR)/$(BINARY)_$(VERSION)_windows_amd64.zip"

# ─── 清理 ───────────────────────────────────────────────

clean: ## 清理构建产物（不删 vendor/ffmpeg）
	rm -rf build/
	@if [ -d "$(FRONTEND_DIR)/dist" ]; then rm -rf $(FRONTEND_DIR)/dist; fi

clean-all: clean ## 清理全部（含下载的 ffmpeg）
	rm -rf $(FFMPEG_DIR)
