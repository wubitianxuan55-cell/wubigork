# gaea 2.0 P0 基线闸门：构建 + 静态检查 + 全量测试 + 前端 lint/build/test + E 系列守卫
# 2026-09-10：补齐前端 lint/vitest（此前本脚本不跑——本地「CI OK」与 GitHub
# Actions 门禁不等价），并把原生命令的 stderr 处理修正（见 Invoke-Native）。
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# 避免 Windows 应用控制策略拦截 Temp 目录（与 build.bat 一致）
$tmp = Join-Path $root '.tmp'
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
$env:TMP = $tmp
$env:TEMP = $tmp

# 原生命令（go/npm）把进度、警告、Vite 构建汇总写到 stderr——那属于正常输出。
# $ErrorActionPreference='Stop' 会把任何 stderr 当终止错误：实测 vite build 的
# 产物汇总走 stderr，脚本在构建**已成功**后中断（NativeCommandError）。
# 故原生命令段统一放行 stderr，成败只认 $LASTEXITCODE。
function Invoke-Native {
    param([string]$Name, [string]$Exe, [string[]]$Arguments)
    Write-Host "=== $Name ==="
    $prev = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try { & $Exe @Arguments } finally { $ErrorActionPreference = $prev }
    if ($LASTEXITCODE -ne 0) { throw "$Name failed (exit $LASTEXITCODE)" }
}

Invoke-Native 'go build' 'go' @('build', './...')
Invoke-Native 'go vet' 'go' @('vet', './...')
Invoke-Native 'go test' 'go' @('test', './...', '-count=1')

Push-Location frontend
if (-not (Test-Path node_modules)) { Invoke-Native 'frontend install' 'npm.cmd' @('install') }
Invoke-Native 'frontend lint' 'npm.cmd' @('run', 'lint')
Invoke-Native 'frontend build' 'npm.cmd' @('run', 'build')
Invoke-Native 'frontend tests (vitest)' 'npm.cmd' @('run', 'test')
Pop-Location

Invoke-Native 'frontend E-series regression guard' 'node' @('scripts\frontend-e-check.mjs')

if (-not (Test-Path (Join-Path $root 'dist\index.html'))) { throw 'dist/index.html missing' }
Write-Host 'CI OK'
