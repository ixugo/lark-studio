#!/usr/bin/env bash
set -euo pipefail

# 构建与发布包同架构的官方 whisper.cpp 命令行运行时。
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WHISPER_VERSION="v1.8.6"
TARGET_OS="${1:-darwin}"
TARGET_ARCH="${2:-$(uname -m)}"

case "$TARGET_ARCH" in
  aarch64) TARGET_ARCH="arm64" ;;
  x86_64) TARGET_ARCH="amd64" ;;
esac

if [ "$TARGET_OS" != "darwin" ]; then
  echo "此脚本仅构建 macOS 运行时"
  exit 1
fi

WORK_DIR="$(mktemp -d)"
SOURCE_DIR="$WORK_DIR/whisper.cpp"
BUILD_DIR="$WORK_DIR/build"
OUTPUT_DIR="$ROOT_DIR/vendor/whisper/$TARGET_OS"

git clone --depth 1 --branch "$WHISPER_VERSION" https://github.com/ggml-org/whisper.cpp.git "$SOURCE_DIR"
cmake -S "$SOURCE_DIR" -B "$BUILD_DIR" \
  -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_OSX_ARCHITECTURES="$TARGET_ARCH" \
  -DCMAKE_INSTALL_PREFIX="$OUTPUT_DIR" \
  -DCMAKE_INSTALL_RPATH='@loader_path/../lib' \
  -DWHISPER_BUILD_TESTS=OFF \
  -DWHISPER_BUILD_EXAMPLES=ON \
  -DWHISPER_BUILD_SERVER=OFF
cmake --build "$BUILD_DIR" --config Release --target whisper-cli
cmake --install "$BUILD_DIR" --config Release

test -x "$OUTPUT_DIR/bin/whisper-cli"
