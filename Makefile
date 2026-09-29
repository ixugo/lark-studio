# lark-studio Makefile
# ─────────────────────────────────────────────────────────

BINARY      := lark-studio
FRONTEND_DIR := frontend
GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)
BUILD_DIR   := build/$(GOOS)_$(GOARCH)
FFMPEG_DIR  := vendor/ffmpeg
WHISPER_PREFIX ?= $(shell brew --prefix whisper-cpp 2>/dev/null)
GGML_PREFIX    ?= $(shell brew --prefix ggml 2>/dev/null)
LIBOMP_PREFIX  ?= $(shell brew --prefix libomp 2>/dev/null)

RECENT_TAG  := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
ifeq ($(RECENT_TAG),v0.0.0)
COMMITS     := $(shell git rev-list --count HEAD 2>/dev/null || echo 0)
else
COMMITS     := $(shell git rev-list --count $(RECENT_TAG)..HEAD 2>/dev/null || echo 0)
endif
VERSION_MAJOR := $(shell echo $(RECENT_TAG) | cut -d. -f1 | sed 's/^v//')
VERSION_MINOR := $(shell echo $(RECENT_TAG) | cut -d. -f2)
VERSION_PATCH := $(shell echo $(RECENT_TAG) | cut -d. -f3)
FINAL_PATCH := $(shell echo '$(VERSION_PATCH) $(COMMITS)' | awk '{print $$1 + $$2}')
VERSION     := v$(VERSION_MAJOR).$(VERSION_MINOR).$(FINAL_PATCH)
GIT_BRANCH  := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
GIT_HASH    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -X main.buildVersion=$(VERSION) \
               -X main.gitBranch=$(GIT_BRANCH) \
               -X main.gitHash=$(GIT_HASH) \
               -X main.buildTime=$(BUILD_TIME)
LDFLAGS_REL := $(LDFLAGS) -X main.release=true -s -w

# macOS 构建以 macOS 13 为最低部署目标，且令外部链接器使用同一目标。
MACOSX_DEPLOYMENT_TARGET := 13.0
MACOS_LDFLAGS := -extldflags=-mmacosx-version-min=$(MACOSX_DEPLOYMENT_TARGET)

# macOS arm64: 使用 ffmpeg-static 发布的静态构建
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
	MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) go build -trimpath -ldflags "$(LDFLAGS_REL) $(MACOS_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) .

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
	@MACOSX_DEPLOYMENT_TARGET=$(MACOSX_DEPLOYMENT_TARGET) go build -ldflags "$(LDFLAGS) $(MACOS_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) .
	@if [ "$(GOOS)" = "darwin" ]; then \
		bash scripts/make-dev-app.sh "$(BUILD_DIR)/$(BINARY)" "$(BUILD_DIR)/Lark Studio.app" "$(VERSION)"; \
	fi
	@bash -c '\
		APP_PID=; OPEN_PID=; VITE_PID=; BASE_APP_PIDS=; \
		process_tree() { \
			local parent="$$1" child; \
			for child in $$(pgrep -P "$$parent" 2>/dev/null || true); do process_tree "$$child"; done; \
			printf "%s\\n" "$$parent"; \
		}; \
		cleanup() { \
			trap - EXIT INT TERM; \
			pids=""; \
			for root in "$$APP_PID" "$$OPEN_PID" "$$VITE_PID"; do \
				if [ -n "$$root" ] && kill -0 "$$root" 2>/dev/null; then pids="$$pids $$(process_tree "$$root")"; fi; \
			done; \
			if [ -n "$$pids" ]; then kill -TERM $$pids 2>/dev/null || true; sleep 0.5; kill -KILL $$pids 2>/dev/null || true; fi; \
			wait 2>/dev/null || true; \
		}; \
		trap cleanup EXIT; trap "exit 130" INT; trap "exit 143" TERM; \
		(cd $(FRONTEND_DIR) && npm run dev) & \
		VITE_PID=$$!; \
		for i in $$(seq 1 30); do \
			curl -s http://127.0.0.1:5173 >/dev/null 2>&1 && break; \
			sleep 0.2; \
		done; \
		echo "✓ 前端热更服务已就绪，正在拉起桌面端..."; \
		if [ "$(GOOS)" = "darwin" ]; then \
			BASE_APP_PIDS=$$(pgrep -f "^$(CURDIR)/$(BUILD_DIR)/Lark Studio.app/Contents/MacOS/lark-studio$$" 2>/dev/null || true); \
			open -n -W --env FRONTEND_DEVSERVER_URL=http://127.0.0.1:5173 "$(CURDIR)/$(BUILD_DIR)/Lark Studio.app" & \
			OPEN_PID=$$!; \
			for i in $$(seq 1 30); do \
				for candidate in $$(pgrep -f "^$(CURDIR)/$(BUILD_DIR)/Lark Studio.app/Contents/MacOS/lark-studio$$" 2>/dev/null || true); do \
					case " $$BASE_APP_PIDS " in *" $$candidate "*) ;; *) APP_PID="$$candidate"; break ;; esac; \
				done; \
				[ -n "$$APP_PID" ] && break; sleep 0.2; \
			done; \
			wait "$$OPEN_PID"; \
		else \
			FRONTEND_DEVSERVER_URL=http://127.0.0.1:5173 $(BUILD_DIR)/$(BINARY); \
		fi \
	'

run: build ## 编译并启动桌面应用（单二进制内嵌模式）
	$(BUILD_DIR)/$(BINARY)

# ─── FFmpeg 静态包下载 ──────────────────────────────────

ffmpeg-macos: ## 下载 macOS 静态 ffmpeg（不存在时自动下载）
	@mkdir -p $(FFMPEG_DIR)/darwin
	@if [ ! -s $(FFMPEG_DIR)/darwin/ffmpeg ]; then \
		echo "⬇ 下载 ffmpeg (macOS arm64)..."; \
		tmpfile=$$(mktemp); \
		curl -fSL $(FFMPEG_MACOS_URL) | gunzip > "$$tmpfile" && \
		mv "$$tmpfile" $(FFMPEG_DIR)/darwin/ffmpeg && \
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

bundle/macos/arm64: ffmpeg-macos build-release ## 打包 macOS arm64 dmg（Lark Studio.app 内嵌 ffmpeg，ad-hoc 签名）
	@echo "⚙ 打包 macOS dmg..."
	@bash scripts/bundle-macos.sh "$(VERSION)" "$(BUILD_DIR)/$(BINARY)" "$(FFMPEG_DIR)/darwin/ffmpeg" "$(DIST_DIR)/$(BINARY)_$(VERSION)_macos_arm64.dmg"

bundle/windows: ffmpeg-windows build-windows ## 打包 Windows amd64 zip（lark-studio.exe + ffmpeg.exe 同目录，解压即用）
	@command -v upx >/dev/null && { echo "⚙ UPX 压缩 Go 二进制..."; upx --best --lzma build/windows_amd64/$(BINARY).exe; } || echo "⚠ 跳过 UPX（未安装）"
	@echo "⚙ 打包 Windows zip..."
	@bash scripts/bundle-windows.sh "$(VERSION)" "build/windows_amd64/$(BINARY).exe" "$(FFMPEG_DIR)/windows/ffmpeg.exe" "$(DIST_DIR)/$(BINARY)_$(VERSION)_windows_amd64.zip"

# ─── 清理 ───────────────────────────────────────────────

clean: ## 清理构建产物（不删 vendor/ffmpeg）
	rm -rf build/
	@if [ -d "$(FRONTEND_DIR)/dist" ]; then rm -rf $(FRONTEND_DIR)/dist; fi

clean-all: clean ## 清理全部（含下载的 ffmpeg）
	rm -rf $(FFMPEG_DIR)
