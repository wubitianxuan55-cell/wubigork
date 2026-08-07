# gaea 2.0 P0 基线闸门：构建 + 静态检查 + 全量测试 + 前端构建
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# 避免 Windows 应用控制策略拦截 Temp 目录（与 build.bat 一致）
$tmp = Join-Path $root '.tmp'
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
$env:TMP = $tmp
$env:TEMP = $tmp

Write-Host '=== go build ==='
go build ./...
if ($LASTEXITCODE -ne 0) { throw 'go build failed' }

Write-Host '=== go vet ==='
go vet ./...
if ($LASTEXITCODE -ne 0) { throw 'go vet failed' }

Write-Host '=== go test ==='
go test ./... -count=1
if ($LASTEXITCODE -ne 0) { throw 'go test failed' }

Write-Host '=== frontend build ==='
Push-Location frontend
if (-not (Test-Path node_modules)) { npm.cmd install }
npm.cmd run build
if ($LASTEXITCODE -ne 0) { Pop-Location; throw 'frontend build failed' }
Pop-Location

if (-not (Test-Path (Join-Path $root 'dist\index.html'))) { throw 'dist/index.html missing' }
Write-Host 'CI OK'
