# gaea 全量测试 runner（沙箱分块版，支持续跑）
# 背景：DSH 沙箱限制「单进程树内大量测试二进制」与前台命令 600s 上限。
# 本脚本逐包执行 + 失败重试，进度写入状态文件；多次调用自动续跑，直到全部完成。
# 用法（分多次调用，每次续跑剩余包）：
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\test-all.ps1
param([string]$StateFile = '')
$ErrorActionPreference = 'Continue'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root
if (-not $StateFile) { $StateFile = Join-Path $root '.tmp\testall-state.json' }

# 隔离运行时临时目录（与 build.bat 一致，规避 SAC/AV 扫描）
$tmp = Join-Path $root '.tmp'
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
$env:TMP = $tmp
$env:TEMP = $tmp

# 加载续跑状态
$done = @()
$failed = @()
if (Test-Path $StateFile) {
    try {
        $state = Get-Content $StateFile -Raw | ConvertFrom-Json
        $done = @($state.done)
        $failed = @($state.failed)
    } catch { Write-Host "状态文件损坏，从头开始: $($_.Exception.Message)" }
}

$packages = @(go list ./... 2>$null)
$total = $packages.Count
$skipped = $done.Count
$start = Get-Date
$pending = @($packages | Where-Object { $_ -notin $done })
$i = 0

foreach ($p in $pending) {
    $i++
    $ok = $false
    for ($try = 1; $try -le 3 -and -not $ok; $try++) {
        $null = go test $p -count=1 2>&1
        if ($LASTEXITCODE -eq 0) { $ok = $true }
        elseif ($try -lt 3) { Start-Sleep -Milliseconds 800 } # 等 AV 扫描锁释放后重试
    }
    $done += $p
    if (-not $ok) { $failed += $p }
    # 每 10 个包落盘一次进度（续跑能力）
    if ($i % 10 -eq 0 -or $i -eq $pending.Count) {
        @{ done = @($done | Sort-Object); failed = @($failed) } | ConvertTo-Json | Set-Content $StateFile -Encoding UTF8
    }
}

# 汇总
$remaining = $total - $done.Count
if ($remaining -gt 0) {
    Write-Host "PARTIAL: $($done.Count - $failed.Count)/$total passed, $remaining remaining (call again to resume)"
    exit 0
}
Write-Host "RESULT: $($total - $failed.Count)/$total packages passed"
if ($failed.Count -gt 0) {
    Write-Host "FAILED:"
    $failed | ForEach-Object { Write-Host "  $_" }
    exit 1
}
Remove-Item $StateFile -ErrorAction SilentlyContinue
exit 0
