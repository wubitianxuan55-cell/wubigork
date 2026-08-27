# check-bindings-drift.ps1 — 绑定面漂移检查（T6-10.3）
#
# 校验 frontend/src/gaea/lib/bindingNames.ts 与 Go 侧实际绑定方法集一致：
#   1. 运行 `go run ./scripts/gen_bindings -names` 取全部导出绑定方法名（稳定字典序）；
#   2. 解析 bindingNames.ts 的 const bindingNames = [...] as const 导出数组；
#   3. 逐项 diff（两端各自排序后 Compare-Object），不一致 exit 1。
#
# bindingNames.ts 是入库清单（勿手改），由 `go run ./scripts/gen_bindings -names`
# 重新生成。Go 侧新增/改名/删除绑定方法后必须同步重新生成，否则本脚本在 CI 失败。
# 注意：只校验 bindingNames.ts ↔ Go；AppBindings 与 bindingNames 的类型级双向断言
# 在 frontend/src/gaea/lib/bridge.ts（tsc 编译期），两边共同构成完整漂移防线。
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$tsPath = Join-Path $root 'frontend/src/gaea/lib/bindingNames.ts'

# ── 1. Go 侧方法名 ────────────────────────────────────────────────────────
Push-Location $root
try {
  $goNames = @(& go run ./scripts/gen_bindings -names) |
    ForEach-Object { $_.Trim() } |
    Where-Object { $_ -ne '' }
  if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: go run ./scripts/gen_bindings -names 失败 (exit $LASTEXITCODE)" -ForegroundColor Red
    exit 1
  }
}
finally {
  Pop-Location
}

# ── 2. bindingNames.ts 中的导出数组 ───────────────────────────────────────
if (-not (Test-Path $tsPath)) {
  Write-Host "ERROR: 缺少 $tsPath —— 请先运行 gen_bindings 生成。" -ForegroundColor Red
  exit 1
}
$raw = Get-Content -Raw -Encoding UTF8 $tsPath
$raw = $raw -replace "`r`n", "`n"          # 行尾归一（兼容 CRLF/LF）
$raw = $raw.TrimStart([char]0xFEFF)        # 去 BOM
$tsNames = @(
  [regex]::Matches($raw, '^\s*"([^"]+)",?\s*$', [System.Text.RegularExpressions.RegexOptions]::Multiline) |
    ForEach-Object { $_.Groups[1].Value }
)

# ── 3. 对比 ───────────────────────────────────────────────────────────────
$goSorted = @($goNames | Sort-Object)
$tsSorted = @($tsNames | Sort-Object)
# @() 包裹：单条差异时 $diff 是单个 PSCustomObject（PS 5.1 无 .Count 属性，
# $null -gt 0 为 False 会静默放行）——强制数组化保证单条差异也能被检出。
$diff = @(Compare-Object -ReferenceObject $goSorted -DifferenceObject $tsSorted)
if ($diff.Count -gt 0) {
  Write-Host "绑定面漂移：Go 侧 ($($goSorted.Count) 个) 与 bindingNames.ts ($($tsSorted.Count) 个) 不一致：" -ForegroundColor Red
  $diff | ForEach-Object {
    $side = if ($_.SideIndicator -eq '<=') { 'Go 侧有，TS 缺' } else { 'TS 有，Go 缺' }
    Write-Host "  $side : $($_.InputObject)"
  }
  Write-Host "修复：运行 'go run ./scripts/gen_bindings -names' 核对方法名后重新生成 bindingNames.ts（勿手改）。" -ForegroundColor Yellow
  exit 1
}

Write-Host "OK：bindingNames.ts 与 Go 绑定面一致（$($goSorted.Count) 个方法）。" -ForegroundColor Green
exit 0
