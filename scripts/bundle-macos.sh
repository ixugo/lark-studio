#!/usr/bin/env bash
# 打包 macOS arm64 一体化 dmg：Lark Studio.app 内嵌 ffmpeg，ad-hoc 签名。
# 用法：scripts/bundle-macos.sh <version> <binary> <ffmpeg> <out-dmg>
set -euo pipefail

VERSION="$1"
BINARY="$2"
FFMPEG="$3"
OUT_DMG="$4"

APP_NAME="Lark Studio.app"
WORK_DIR="$(mktemp -d)"
trap 'trash "$WORK_DIR"' EXIT

DMG_STAGE="$WORK_DIR/stage"
APP_DIR="$DMG_STAGE/$APP_NAME"
MACOS_DIR="$APP_DIR/Contents/MacOS"
RES_DIR="$APP_DIR/Contents/Resources"
mkdir -p "$MACOS_DIR" "$RES_DIR"

# 1. 拷贝主程序与 ffmpeg 与 图标
cp "$BINARY" "$MACOS_DIR/lark-studio"
chmod +x "$MACOS_DIR/lark-studio"
cp "$FFMPEG" "$MACOS_DIR/ffmpeg"
chmod +x "$MACOS_DIR/ffmpeg"
ICONSET_DIR="$WORK_DIR/LarkStudio.iconset"
mkdir -p "$ICONSET_DIR"
for size in 16 32 128 256 512; do
  sips -z "$size" "$size" internal/wails/lark-logo.png \
    --out "$ICONSET_DIR/icon_${size}x${size}.png" >/dev/null
  doubled=$((size * 2))
  sips -z "$doubled" "$doubled" internal/wails/lark-logo.png \
    --out "$ICONSET_DIR/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$ICONSET_DIR" -o "$RES_DIR/AppIcon.icns"

# 2. Info.plist
cat > "$APP_DIR/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>Lark Studio</string>
  <key>CFBundleDisplayName</key><string>Lark Studio</string>
  <key>CFBundleIdentifier</key><string>com.ixugo.larkstudio</string>
  <key>CFBundleVersion</key><string>${VERSION}</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleExecutable</key><string>lark-studio</string>
  <key>CFBundleIconFile</key><string>AppIcon</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>LSMinimumSystemVersion</key><string>13.0</string>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
EOF

# 3. 清理隔离属性与执行 ad-hoc 签名，防止 macOS AMFI 拦截杀进程
xattr -cr "$APP_DIR" || true
codesign -s - --force "$MACOS_DIR/ffmpeg"
codesign -s - --force --deep "$APP_DIR"

# 4. 创建 Applications 软链接以支持拖拽安装
ln -s /Applications "$DMG_STAGE/Applications"

# 5. 打 dmg
mkdir -p "$(dirname "$OUT_DMG")"
hdiutil create -volname "Lark Studio" -srcfolder "$DMG_STAGE" -ov -format ULMO "$OUT_DMG" >/dev/null
echo "✓ dmg: $OUT_DMG"
