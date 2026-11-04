# gaea_install.ps1 - install/refresh the gaea daily shell + desktop shortcut.
#
# Why a script: the voice engine DLLs (onnxruntime.dll, sherpa-onnx-c-api.dll)
# are load-time linked by the Windows loader, so they must sit in the SAME
# directory as gaea.exe (go:embed cannot cover load-time imports; the upstream
# sherpa-onnx module ships DLLs only, no static libs). Single-file exe is not
# possible; the fix for desktop clutter is: exe+DLLs live together in one
# install dir, and the desktop gets ONE shortcut.
#
# Usage (from repo root):
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\gaea_install.ps1 [-Launch] [-CleanDesktop]
#
# ASCII-only discipline: PS 5.1 parses non-BOM UTF-8 comments as ANSI mojibake
# and silently swallows following lines (see voice memory 2026-11-05).
param(
    [string]$SourceDir = 'C:\AI\wubigrok\build\bin',
    [string]$InstallDir = 'C:\AI\gaea-app',
    [switch]$Launch,
    [switch]$CleanDesktop
)
$ErrorActionPreference = 'Stop'

$files = @('gaea.exe', 'onnxruntime.dll', 'sherpa-onnx-c-api.dll', 'sherpa-onnx-cxx-api.dll')

# 1. stop any running gaea shell (old desktop copy, install dir, repo shell)
$procs = Get-Process -Name 'gaea' -ErrorAction SilentlyContinue
if ($procs) {
    $procs | ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
    Start-Sleep -Seconds 2
}
$left = Get-CimInstance Win32_Process -Filter "Name='gaea.exe'"
if ($left) { Write-Host 'KILL FAIL: gaea.exe still alive' -ForegroundColor Red; exit 1 }

# 2. copy exe + DLLs into the install dir (one self-contained folder)
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
foreach ($f in $files) {
    $src = Join-Path $SourceDir $f
    if (-not (Test-Path $src)) { Write-Host ('MISSING SOURCE: ' + $src) -ForegroundColor Red; exit 1 }
    Copy-Item $src (Join-Path $InstallDir $f) -Force
    $h1 = (Get-FileHash $src -Algorithm SHA256).Hash
    $h2 = (Get-FileHash (Join-Path $InstallDir $f) -Algorithm SHA256).Hash
    if ($h1 -ne $h2) { Write-Host ('HASH MISMATCH: ' + $f) -ForegroundColor Red; exit 1 }
    Write-Host ('installed: ' + $f + '  sha256 ' + $h1.Substring(0, 12) + '...')
}
$buildTime = (Get-Item (Join-Path $SourceDir 'gaea.exe')).LastWriteTime
Write-Host ('build time: ' + $buildTime)

# 3. desktop shortcut (one icon; DLLs stay in the install dir)
$desktop = [Environment]::GetFolderPath('Desktop')
$lnkPath = Join-Path $desktop 'gaea.lnk'
$sh = New-Object -ComObject WScript.Shell
$lnk = $sh.CreateShortcut($lnkPath)
$lnk.TargetPath = Join-Path $InstallDir 'gaea.exe'
$lnk.WorkingDirectory = $InstallDir
$lnk.IconLocation = (Join-Path $InstallDir 'gaea.exe') + ',0'
$lnk.Description = 'gaea AI assistant'
$lnk.Save()
Write-Host ('shortcut: ' + $lnkPath)

# 4. remove the old loose desktop copies (exe + 3 DLLs + obsolete launcher bat)
if ($CleanDesktop) {
    $stale = @('gaea.exe', 'onnxruntime.dll', 'sherpa-onnx-c-api.dll', 'sherpa-onnx-cxx-api.dll', 'gaea-go.bat')
    foreach ($f in $stale) {
        $p = Join-Path $desktop $f
        if (Test-Path $p) { Remove-Item $p -Force; Write-Host ('desktop cleaned: ' + $f) }
    }
}

# 5. optional launch + PID-after-build verification (deliver discipline)
if ($Launch) {
    Start-Process -FilePath (Join-Path $InstallDir 'gaea.exe') -WorkingDirectory $InstallDir
    Start-Sleep -Seconds 5
    $p = Get-CimInstance Win32_Process -Filter "Name='gaea.exe'" | Select-Object -First 1
    if ($null -eq $p) { Write-Host 'LAUNCH FAIL: no gaea.exe process' -ForegroundColor Red; exit 1 }
    Write-Host ('new PID: ' + $p.ProcessId + '  started: ' + $p.CreationDate)
    if ($p.CreationDate -gt $buildTime) { Write-Host 'PID AFTER BUILD: OK' } else { Write-Host 'PID BEFORE BUILD: STALE SHELL' -ForegroundColor Red; exit 1 }
}
Write-Host 'INSTALL OK' -ForegroundColor Green
