# check-version-drift.ps1 — 版本漂移检查（CI / 发布闸门用）
#
# 校验三处版本源是否一致：
#   1. internal/app/app_info.go   —— const AppVersion = "X.Y.Z"
#   2. wails.json                 —— info.productVersion = "X.Y.Z"
#   3. versioninfo.rc             —— FILEVERSION X,Y,Z,0
#
# 不一致则 exit 1（供 CI/发布闸门拦截），一致 exit 0。
# 修复：运行 scripts\sync-version.ps1 [-Version X.Y.Z]
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$appInfo = Join-Path $root 'internal/app/app_info.go'
$wails   = Join-Path $root 'wails.json'
$rcFile  = Join-Path $root 'versioninfo.rc'

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

try {
  $app = Get-VersionFromAppInfo $appInfo
  $wls = Get-VersionFromWails $wails
  $rc  = Get-VersionFromRc $rcFile
} catch {
  Write-Host "ERROR: $($_.Exception.Message)" -ForegroundColor Red
  exit 1
}

Write-Host '版本源现状：'
Write-Host "  app_info.go    AppVersion      = $app"
Write-Host "  wails.json     productVersion  = $wls"
Write-Host "  versioninfo.rc FILEVERSION     = $rc"

if ($app -ne $wls -or $app -ne $rc) {
  Write-Host "ERROR: 版本漂移！三处不一致（app_info.go=$app / wails.json=$wls / versioninfo.rc=$rc）" -ForegroundColor Red
  Write-Host '修复：运行 scripts\sync-version.ps1 [-Version X.Y.Z]' -ForegroundColor Yellow
  exit 1
}
Write-Host "OK：三处版本一致（$app）。" -ForegroundColor Green
exit 0
