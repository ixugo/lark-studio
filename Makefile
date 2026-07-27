# vdub Makefile
# ────────────────────────────────────────────
# 开发工作流 + 打包构建

BINARY     := vdub
GO_SRC     := .
UI_DIR     := ui
GOARCH     ?= $(shell go env GOARCH)
GOOS       ?= $(shell go env GOOS)
BUILD_DIR  := build/$(GOOS)_$(GOARCH)
APP_BUNDLE := $(UI_DIR)/build/macos/Build/Products/Release/vdub_ui.app

.PHONY: build run dev test e2e e2e-dub clip clean bundle help

help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

build: ## 编译 Go 引擎
	go build -o $(BUILD_DIR)/$(BINARY) $(GO_SRC)

test: ## 运行全量 Go 测试
	go test ./...

# E2E 测试需要环境变量：
#   VDUB_LLM_BASE_URL   LLM API 地址
#   VDUB_LLM_API_KEY     LLM API 密钥
#   VDUB_LLM_MODEL       LLM 模型名（默认 opus）
#   VDUB_WHISPER_MODEL   whisper ggml 模型路径
e2e: ## E2E 翻译测试（裁剪 10~30s + whisper + translate + burn）
	go test -tags=integration -run TestIntegration_E2E_Whisper -v -timeout 15m ./internal/core/pipeline/

e2e-dub: ## E2E 配音测试（翻译 + TTS + 混音 + 烧录）
	go test -tags=integration -run TestIntegration_E2E_Dub -v -timeout 15m ./internal/core/pipeline/

clip: build ## CLI 模式裁剪+翻译测试（需设置 VDUB_LLM_* 环境变量）
	$(BUILD_DIR)/$(BINARY) -run "$(VIDEO)" -ss 10 -to 30 -mode 2

dev: build ## 开发模式：编译 Go + Flutter debug 运行
	@echo "Go binary: $(BUILD_DIR)/$(BINARY)"
	cd $(UI_DIR) && flutter run -d macos

run: build ## 仅启动 Go 引擎（HTTP 模式）
	$(BUILD_DIR)/$(BINARY)

bundle: build ## 打包 macOS .app（Go 二进制内嵌）
	cd $(UI_DIR) && flutter build macos --release
	@mkdir -p "$(APP_BUNDLE)/Contents/Resources"
	cp $(BUILD_DIR)/$(BINARY) "$(APP_BUNDLE)/Contents/Resources/$(BINARY)"
	@echo "✓ $(APP_BUNDLE)"

clean: ## 清理产物
	rm -rf $(BUILD_DIR)
	cd $(UI_DIR) && flutter clean
