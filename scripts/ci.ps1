# gaea 本地快闸：构建 + 静态检查 + lint + 全量测试 + 前端 lint/build/test + E 系列守卫 + 绑定漂移闸
# 2026-09-10：补齐前端 lint/vitest（此前本脚本不跑——本地「CI OK」与 GitHub
# Actions 门禁不等价），并把原生命令的 stderr 处理修正（见 Invoke-Native）。
# 2026-09-27（审计 P0-1）：口径声明——本脚本=「本地快闸」，GitHub Actions
# (.github/workflows/ci.yml) 三 job（backend/race/frontend）为远端门禁：race
# 检测在 ubuntu job（本机无 gcc 跑不了 -race）+ internal/app race 子集在
# windows runner；两侧不等价，对外表述须写明是哪一侧（禁无定语「CI 绿」）。
# 2026-09-27 追加：-Quick 迭代档——构建/vet/门禁 lint/静态守卫全保留，测试改
# 「受影响面定向」（scripts/test-related.ps1：Go 吃结果缓存 + vitest related），
# 跳前端 build 与全量 vitest。全量档行为不变（-count=1 + build + 全量 vitest），
# push 前/发版前必跑全量档；两档口径见 AGENTS「CI 绿」定语。
param([switch]$Quick)
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

# 版本三处漂移闸门（v4.334 入：v4.324~v4.333 十版未跑 sync-version，exe 内嵌
# 版本停在 4.323.0 而闸门不在 CI 内从未拦截——收进 CI 防再犯）
Invoke-Native 'version drift check' 'powershell' @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', (Join-Path $PSScriptRoot 'check-version-drift.ps1'))

# .tmp 卫生守卫（v4.360，2026-09-20 发现）：堆积超阈值时清理瞬态模式——
# v4.359 实证 .tmp 堆到 2.4GB 后 vitest worker 间歇失败（连续两轮 CI 挂）。
Invoke-Native '.tmp hygiene guard' 'powershell' @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', (Join-Path $PSScriptRoot 'clean-tmp.ps1'))

Invoke-Native 'go build' 'go' @('build', './...')
Invoke-Native 'go vet' 'go' @('vet', './...')

# golangci-lint 门禁（2026-09-27 入：审计 P0-1 证据③——v1 配置 + action「version:
# latest」此前从未跑通，CHANGELOG 零 golangci 生效记录）。钉 v2.14.0 与 ci.yml
# 同版；本机缺二进制时一次性 go install 补齐（首次需网络）。
$gopath = (go env GOPATH | Select-Object -First 1).Trim()
$golangci = Join-Path $gopath 'bin\golangci-lint.exe'
if (-not (Test-Path $golangci)) {
    Invoke-Native 'install golangci-lint v2.14.0' 'go' @('install', 'github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0')
}
Invoke-Native 'golangci-lint (pinned v2.14.0)' $golangci @('run', '--timeout=8m', './...')

if ($Quick) {
    # 迭代档：受影响面定向测试（Go 包映射吃结果缓存 + 前端 vitest related）
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'test-related.ps1')
    if ($LASTEXITCODE -ne 0) { throw "test-related failed (exit $LASTEXITCODE)" }
} else {
    # 口径对齐（2026-10-02，审计 X1-12）：本地快闸是**单次** go test，本就没有
    # 「整批重试」这一步；远端 .github/workflows/ci.yml 的 backend job 已改为
    # 「失败包集合 ⊆ scripts/go-known-flaky.txt 才隔离复跑」
    # （scripts/classify-go-failures.mjs）。此处刻意不引入重试——本地红即红。
    Invoke-Native 'go test' 'go' @('test', './...', '-count=1')
}

Push-Location frontend
# 2026-09-27（审计 P0-1 证据①）：v4.371 起唯一 lockfile 是 pnpm-lock.yaml，
# npm install 无锁可依＝依赖树漂移风险；改 pnpm --frozen-lockfile 与 ci.yml 同口径。
if (-not (Test-Path node_modules)) { Invoke-Native 'frontend install' 'pnpm.cmd' @('install', '--frozen-lockfile') }
Invoke-Native 'frontend lint' 'npm.cmd' @('run', 'lint')
if (-not $Quick) {
    Invoke-Native 'frontend build' 'npm.cmd' @('run', 'build')
    Invoke-Native 'frontend tests (vitest)' 'npm.cmd' @('run', 'test')
} else {
    Write-Host '=== frontend build + full vitest (skipped in -Quick; related already run) ==='
}
Pop-Location

Invoke-Native 'frontend E-series regression guard' 'node' @('scripts\frontend-e-check.mjs')
Invoke-Native 'repo hygiene guard (docs + script encoding)' 'node' @('scripts\check-docs.mjs')

if (-not $Quick -and -not (Test-Path (Join-Path $root 'dist\index.html'))) { throw 'dist/index.html missing' }
Write-Host 'CI OK'
