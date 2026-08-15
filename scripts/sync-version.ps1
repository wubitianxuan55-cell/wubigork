# sync-version.ps1 — 三处版本源同步脚本（gaea3 Step 0 / 阶段 7 发布用）
#
# 统一维护三处版本源，保持一致：
#   1. internal/app/app_info.go   —— const AppVersion = "X.Y.Z"
#   2. wails.json                 —— info.productVersion = "X.Y.Z"
#   3. versioninfo.rc             —— FILEVERSION/PRODUCTVERSION X,Y,Z,0 + VALUE 字符串
#
# 用法：
#   scripts\sync-version.ps1                  # 从 wails.json 读取基线版本，同步其余两处
#   scripts\sync-version.ps1 -Version 2.35.0  # 指定新版本，三处一起写
#
# 退出码：0 成功；1 参数/格式/写盘失败。
param([string]$Version = '')

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$appInfo = Join-Path $root 'internal/app/app_info.go'
$wails   = Join-Path $root 'wails.json'
$rcFile  = Join-Path $root 'versioninfo.rc'

$utf8NoBom = New-Object System.Text.UTF8Encoding($false)

# ── 读取三处当前版本 ────────────────────────────────────────────────────────
function Get-VersionFromAppInfo($path) {
  $raw = [System.IO.File]::ReadAllText($path)
  $m = [regex]::Match($raw, 'const\s+AppVersion\s*=\s*"([^"]+)"')
  if (-not $m.Success) { throw 'app_info.go 中未找到 AppVersion' }
  return $m.Groups[1].Value
}
function Get-VersionFromWails($path) {
  $raw = [System.IO.File]::ReadAllText($path)
  $m = [regex]::Match($raw, '"productVersion"\s*:\s*"([^"]+)"')
  if (-not $m.Success) { throw 'wails.json 中未找到 productVersion' }
  return $m.Groups[1].Value
}
function Get-VersionFromRc($path) {
  $raw = [System.IO.File]::ReadAllText($path)
  $m = [regex]::Match($raw, 'FILEVERSION\s+([0-9]+,[0-9]+,[0-9]+)(?:,[0-9]+)?')
  if (-not $m.Success) { throw 'versioninfo.rc 中未找到 FILEVERSION' }
  return ($m.Groups[1].Value -replace ',', '.')
}

# ── 解析目标版本 ────────────────────────────────────────────────────────────
if ($Version -eq '') {
  $Version = Get-VersionFromWails $wails
  Write-Host "未指定 -Version，从 wails.json 读取基线版本：$Version"
}
if ($Version -notmatch '^\d+\.\d+\.\d+$') {
  Write-Host "ERROR: 版本号格式应为 X.Y.Z，得到：$Version" -ForegroundColor Red
  exit 1
}

$curApp = Get-VersionFromAppInfo $appInfo
$curRc  = Get-VersionFromRc $rcFile

# ── 打印同步前状态（差异） ─────────────────────────────────────────────────
Write-Host '同步前：'
Write-Host "  app_info.go    AppVersion      = $curApp"
Write-Host "  wails.json     productVersion  = $Version (基线)"
Write-Host "  versioninfo.rc FILEVERSION     = $curRc"

# ── 写 app_info.go（UTF-8 无 BOM） ─────────────────────────────────────────
$appNew = [regex]::Replace(
  [System.IO.File]::ReadAllText($appInfo),
  'const\s+AppVersion\s*=\s*"[^"]+"',
  'const AppVersion = "' + $Version + '"')
[System.IO.File]::WriteAllText($appInfo, $appNew, $utf8NoBom)

# ── 写 wails.json（UTF-8 无 BOM） ──────────────────────────────────────────
$wailsNew = [regex]::Replace(
  [System.IO.File]::ReadAllText($wails),
  '"productVersion"\s*:\s*"[^"]+"',
  '"productVersion": "' + $Version + '"')
[System.IO.File]::WriteAllText($wails, $wailsNew, $utf8NoBom)

# ── 写 versioninfo.rc（数字 + 字符串两处，UTF-8 无 BOM） ───────────────────
$rcNew = [System.IO.File]::ReadAllText($rcFile)
$rcVerComma = ($Version -replace '\.', ',')
$rcNew = [regex]::Replace($rcNew, 'FILEVERSION\s+[0-9]+,[0-9]+,[0-9]+,[0-9]+', 'FILEVERSION ' + $rcVerComma + ',0')
$rcNew = [regex]::Replace($rcNew, 'PRODUCTVERSION\s+[0-9]+,[0-9]+,[0-9]+,[0-9]+', 'PRODUCTVERSION ' + $rcVerComma + ',0')
$rcNew = [regex]::Replace($rcNew, 'VALUE\s+"FileVersion",\s*"[^"]+"', 'VALUE "FileVersion", "' + $Version + '"')
$rcNew = [regex]::Replace($rcNew, 'VALUE\s+"ProductVersion",\s*"[^"]+"', 'VALUE "ProductVersion", "' + $Version + '"')
[System.IO.File]::WriteAllText($rcFile, $rcNew, $utf8NoBom)

# ── 回读校验 + 打印结果 ────────────────────────────────────────────────────
$newApp   = Get-VersionFromAppInfo $appInfo
$newWails = Get-VersionFromWails $wails
$newRc    = Get-VersionFromRc $rcFile
if ($newApp -ne $Version -or $newWails -ne $Version -or $newRc -ne $Version) {
  Write-Host "ERROR: 同步后仍不一致：$newApp / $newWails / $newRc" -ForegroundColor Red
  exit 1
}
Write-Host '同步完成：'
Write-Host "  app_info.go    AppVersion      = $newApp"
Write-Host "  wails.json     productVersion  = $newWails"
Write-Host "  versioninfo.rc FILEVERSION     = $newRc"
Write-Host "OK：三处版本已一致（$Version）。" -ForegroundColor Green
exit 0
