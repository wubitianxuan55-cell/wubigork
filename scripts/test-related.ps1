# test-related.ps1 — 受影响面定向测试（迭代档，2026-09-27）
#
# 动机（用户 2026-09-27：「软件大了，全量动不动几分钟」）：一次改动通常只碰
# 1-2 个包/套件，全量 5-10 分钟大部分在烧无关代码。本脚本把「定向优先」从
# 纪律变成一条命令：
#   - Go 改动   → 映射到所在包，go test（不带 -count=1，吃结果缓存，没动的包秒回）
#   - 前端改动  → npx vitest related <改动源文件>（按 import 图只跑受影响套件）
#
# 用法：
#   powershell -NoProfile -File scripts\test-related.ps1                 # 只看工作树未提交改动
#   powershell -NoProfile -File scripts\test-related.ps1 -Base origin/main
#                 # 并入 origin/main...HEAD 的已提交未推送改动（push 前快验）
#
# 口径：本脚本是「迭代快验」，不是门禁——push 前跑一次 ci.ps1 全量、发版跑
# ci.ps1 -count=1 全量不变（本地快闸两档：-Quick≈本脚本+静态闸；全量=原样）。
param([string]$Base = "")

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# ── 收集改动文件 ───────────────────────────────────────────────────────────
$changed = @()
$changed += (& git diff --name-only HEAD) | Where-Object { $_ -ne '' }
$changed += (& git ls-files --others --exclude-standard) | Where-Object { $_ -ne '' }
if ($Base -ne '') {
    $changed += (& git diff --name-only "$Base...HEAD") | Where-Object { $_ -ne '' }
}
$changed = $changed | Sort-Object -Unique
if ($changed.Count -eq 0) {
    Write-Host '[test-related] 无改动，无事可测。'
    exit 0
}

$failed = 0

# ── Go：文件 → 所在包 ──────────────────────────────────────────────────────
$goFiles = $changed | Where-Object { $_ -match '\.go$' -and (Test-Path (Join-Path $root $_)) }
if ($goFiles.Count -gt 0) {
    $pkgs = @($goFiles | ForEach-Object {
        $dir = $_ -replace '/[^/]+$', ''
        if ($dir -eq $_ -or $dir -eq '') { '.' } else { "./$dir" }
    } | Sort-Object -Unique)
    Write-Host ("[test-related] Go 相关包 {0} 个：{1}" -f $pkgs.Count, ($pkgs -join ' '))
    Write-Host '[test-related] go test（吃结果缓存，无 -count=1）'
    & go test @pkgs
    if ($LASTEXITCODE -ne 0) { $failed = 1 }
} else {
    Write-Host '[test-related] 无 Go 改动，跳过 go test'
}

# ── 前端：vitest related（按 import 图取受影响套件）────────────────────────
$feExt = '\.(ts|tsx|js|jsx|css)$'
$feFiles = @($changed |
    Where-Object { $_ -match '^frontend/' -and $_ -match $feExt } |
    Where-Object { -not ($_ -match '^frontend/(dist|node_modules)/') } |
    Where-Object { Test-Path (Join-Path $root $_) } |
    ForEach-Object { $_ -replace '^frontend/', '' })
if ($feFiles.Count -gt 0) {
    Write-Host ("[test-related] 前端改动 {0} 个文件 → vitest related" -f $feFiles.Count)
    Push-Location (Join-Path $root 'frontend')
    try {
        & npx vitest related @feFiles
        if ($LASTEXITCODE -ne 0) { $failed = 1 }
    } finally {
        Pop-Location
    }
} else {
    Write-Host '[test-related] 无前端源码改动，跳过 vitest'
}

if ($failed -ne 0) {
    Write-Host '[test-related] 相关测试有失败。' -ForegroundColor Red
    exit 1
}
Write-Host '[test-related] 相关测试全绿。'
