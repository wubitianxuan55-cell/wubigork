# clean-tmp.ps1 — .tmp 卫生守卫（v4.360，2026-09-20 发现）
#
# 背景：build.bat 与 ci.ps1 把 TMP/TEMP 重定向到 <root>\.tmp（规避 Windows
# SAC/AV 对系统 Temp 的策略）。长期不清时 .tmp 会堆积数 GB（go test 残留目录、
# edge-* 浏览器 profile、build 日志）——v4.359 实证：堆到 2.4GB 后 vitest
# worker 间歇失败（连续两轮 CI 挂在 vitest 步），清理到 157MB 后稳过。
#
# 策略：只删「已知安全」的瞬态模式（Test* 测试残留 / *.log / edge-*·walk-*
# ui-sweep·dsh-context 等旧诊断 profile / smoke-*.exe / go-build*），不动
# .tmp 里其他内容（可能有正在使用的工具产物）。超阈值才清，平时零成本。
#
# 用法：scripts\clean-tmp.ps1 [-ThresholdMB 512]
# 退出码：0 成功（含无需清理）；1 参数/路径错误。
param([int]$ThresholdMB = 512)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$tmp = Join-Path $root '.tmp'
if (-not (Test-Path $tmp)) { Write-Host '[clean-tmp] .tmp 不存在，跳过'; exit 0 }

$sizeMB = [math]::Round((Get-ChildItem $tmp -Recurse -Force -ErrorAction SilentlyContinue |
    Measure-Object -Property Length -Sum).Sum / 1MB)
if ($sizeMB -le $ThresholdMB) {
    Write-Host "[clean-tmp] .tmp ${sizeMB}MB <= 阈值 ${ThresholdMB}MB，无需清理"
    exit 0
}

Write-Host "[clean-tmp] .tmp ${sizeMB}MB > 阈值 ${ThresholdMB}MB，清理瞬态模式..."
$patterns = @('Test*', '*.log', 'edge-*', 'walk-*', 'ui-sweep*', 'dsh-context*', 'smoke-*.exe', 'go-build*')
$freed = 0
foreach ($p in $patterns) {
    Get-ChildItem $tmp -Filter $p -Force -ErrorAction SilentlyContinue | ForEach-Object {
        $sz = 0
        if ($_.PSIsContainer) {
            $sz = [math]::Round((Get-ChildItem $_.FullName -Recurse -Force -ErrorAction SilentlyContinue |
                Measure-Object -Property Length -Sum).Sum / 1MB)
        } else { $sz = [math]::Round($_.Length / 1MB) }
        Remove-Item $_.FullName -Recurse -Force -ErrorAction SilentlyContinue
        if (-not (Test-Path $_.FullName)) { $freed += $sz }
    }
}
$after = [math]::Round((Get-ChildItem $tmp -Recurse -Force -ErrorAction SilentlyContinue |
    Measure-Object -Property Length -Sum).Sum / 1MB)
Write-Host "[clean-tmp] 清理约 ${freed}MB，.tmp 现为 ${after}MB"
exit 0
