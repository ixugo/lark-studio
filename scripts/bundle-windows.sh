#!/usr/bin/env bash
# 打包 Windows amd64 zip：lark-studio.exe + ffmpeg.exe 同目录，解压即用。
# 用法：scripts/bundle-windows.sh <version> <exe> <ffmpeg-exe> <out-zip>
set -euo pipefail

VERSION="$1"
EXE="$2"
FFMPEG_EXE="$3"
OUT_ZIP="$4"

# 转为绝对路径，避免 cd 后相对路径失效
case "$OUT_ZIP" in
  /*) ;;
  *) OUT_ZIP="$(pwd)/$OUT_ZIP" ;;
esac

WORK_DIR="$(mktemp -d)"
trap 'trash "$WORK_DIR"' EXIT

STAGE="$WORK_DIR/lark-studio"
mkdir -p "$STAGE"
cp "$EXE" "$STAGE/lark-studio.exe"
cp "$FFMPEG_EXE" "$STAGE/ffmpeg.exe"

mkdir -p "$(dirname "$OUT_ZIP")"
(cd "$WORK_DIR" && zip -q -r "$OUT_ZIP" lark-studio)
echo "✓ zip: $OUT_ZIP"
