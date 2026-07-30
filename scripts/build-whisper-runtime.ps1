param(
  [ValidateSet('windows')]
  [string]$TargetOS = 'windows',
  [ValidateSet('amd64')]
  [string]$TargetArch = 'amd64'
)

$ErrorActionPreference = 'Stop'
$RootDir = Split-Path -Parent $PSScriptRoot
$WhisperVersion = 'v1.8.6'
$WorkDir = Join-Path ([System.IO.Path]::GetTempPath()) "vdub-whisper-$WhisperVersion-$TargetArch"
$SourceDir = Join-Path $WorkDir 'whisper.cpp'
$BuildDir = Join-Path $WorkDir 'build'
$OutputDir = Join-Path $RootDir "vendor\whisper\$TargetOS"

git clone --depth 1 --branch $WhisperVersion https://github.com/ggml-org/whisper.cpp.git $SourceDir
cmake -S $SourceDir -B $BuildDir `
  -DCMAKE_BUILD_TYPE=Release `
  -DCMAKE_INSTALL_PREFIX=$OutputDir `
  -DWHISPER_BUILD_TESTS=OFF `
  -DWHISPER_BUILD_EXAMPLES=ON `
  -DWHISPER_BUILD_SERVER=OFF
cmake --build $BuildDir --config Release --target whisper-cli
cmake --install $BuildDir --config Release

$Binary = Join-Path $OutputDir 'bin\whisper-cli.exe'
if (-not (Test-Path $Binary)) {
  throw "whisper-cli build failed: $Binary"
}
