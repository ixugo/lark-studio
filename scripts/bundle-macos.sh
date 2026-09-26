#!/usr/bin/env bash
# 打包 macOS arm64 一体化 dmg：vdub.app 内嵌 ffmpeg，ad-hoc 签名。
# 用法：scripts/bundle-macos.sh <version> <binary> <ffmpeg> <out-dmg>
set -euo pipefail

VERSION="$1"
BINARY="$2"
FFMPEG="$3"
OUT_DMG="$4"

APP_NAME="vdub.app"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

APP_DIR="$WORK_DIR/$APP_NAME"
MACOS_DIR="$APP_DIR/Contents/MacOS"
RES_DIR="$APP_DIR/Contents/Resources"
mkdir -p "$MACOS_DIR" "$RES_DIR"

# 1. 拷贝主程序与 ffmpeg
cp "$BINARY" "$MACOS_DIR/vdub"
chmod +x "$MACOS_DIR/vdub"
cp "$FFMPEG" "$MACOS_DIR/ffmpeg"
chmod +x "$MACOS_DIR/ffmpeg"

# 2. Info.plist
cat > "$APP_DIR/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>vdub</string>
  <key>CFBundleDisplayName</key><string>vdub</string>
  <key>CFBundleIdentifier</key><string>com.ixugo.vdub</string>
  <key>CFBundleVersion</key><string>${VERSION}</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleExecutable</key><string>vdub</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>LSMinimumSystemVersion</key><string>12.0</string>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
EOF

# 3. 签名说明：ffmpeg-static 的 Mach-O LINKEDIT 段无足够空间插入签名，
# codesign 对其报 "invalid or unsupported format for signature"。
# ad-hoc 签名对 Gatekeeper 本无豁免作用（用户仍需右键打开），故跳过签名直接打 dmg。
# 若未来引入 Developer ID 公证，需改用可签名的 ffmpeg 构建（如 evermeet 或自编译）。

# 4. 打 dmg
mkdir -p "$(dirname "$OUT_DMG")"
hdiutil create -volname "vdub" -srcfolder "$APP_DIR" -ov -format ULMO "$OUT_DMG" >/dev/null
echo "✓ dmg: $OUT_DMG"
