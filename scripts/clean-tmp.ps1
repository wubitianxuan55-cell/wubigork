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

# 2026-10-01 随版收口：目录体积一律按 -File 枚举 + 空集合兜底。此前直接对
# Get-ChildItem -Recurse 的结果 Measure-Object -Property Length——枚举集里只有
# 目录（无一有 Length 属性）时 Measure-Object 抛 PropertyNotFound，在
# $ErrorActionPreference='Stop' 下整个守卫退出 1，CI 首闸就挂（v4.439.0 发版
# 实测：edge-prof5 被僵尸进程锁住删不掉，文件删净后只剩目录树，复测必炸）。
function Get-SizeMB([string]$path) {
    $sum = (Get-ChildItem $path -Recurse -File -Force -ErrorAction SilentlyContinue |
        Measure-Object -Property Length -Sum).Sum
    if ($null -eq $sum) { return 0 }
    return [math]::Round($sum / 1MB)
}

$sizeMB = Get-SizeMB $tmp
if ($sizeMB -le $ThresholdMB) {
    Write-Host "[clean-tmp] .tmp ${sizeMB}MB <= 阈值 ${ThresholdMB}MB，无需清理"
    exit 0
}

Write-Host "[clean-tmp] .tmp ${sizeMB}MB > 阈值 ${ThresholdMB}MB，清理瞬态模式..."
$patterns = @('Test*', '*.log', 'edge-*', 'walk-*', 'ui-sweep*', 'dsh-context*', 'smoke-*.exe', 'go-build*', 'uiwalk*', 'home-shots*', '*-shots*')
$freed = 0
foreach ($p in $patterns) {
    Get-ChildItem $tmp -Filter $p -Force -ErrorAction SilentlyContinue | ForEach-Object {
        $sz = 0
        if ($_.PSIsContainer) { $sz = Get-SizeMB $_.FullName } else { $sz = [math]::Round($_.Length / 1MB) }
        Remove-Item $_.FullName -Recurse -Force -ErrorAction SilentlyContinue
        if (-not (Test-Path $_.FullName)) { $freed += $sz }
    }
}
# 2026-09-26 补洞：无头 Edge 走查 profile 常驻在**子目录**里（.tmp/uiwalk/edgeprofile*），
# 上面的根级模式扫不到——实测 577MB 全在那里。新增按名递归删 profile 目录，
# 同级的探针脚本（*.js/*.mjs）与目检截图（*.png）一律保留。
Get-ChildItem $tmp -Recurse -Directory -Filter 'edgeprofile*' -Force -ErrorAction SilentlyContinue | ForEach-Object {
    $sz = Get-SizeMB $_.FullName
    Remove-Item $_.FullName -Recurse -Force -ErrorAction SilentlyContinue
    if (-not (Test-Path $_.FullName)) { $freed += $sz }
}
$after = Get-SizeMB $tmp
Write-Host "[clean-tmp] 清理约 ${freed}MB，.tmp 现为 ${after}MB"
exit 0
