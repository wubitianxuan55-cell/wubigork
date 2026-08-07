# gaea 2.0.1 数据备份：whisper_data / novels / office / 用户配置 → 时间戳目录
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$out = Join-Path $root ("backups\" + $stamp)
New-Item -ItemType Directory -Force -Path $out | Out-Null

$items = @(
  (Join-Path $root 'whisper_data'),
  (Join-Path $root 'novels'),
  (Join-Path $env:USERPROFILE '.gaea_config.json')
)
foreach ($src in $items) {
  if (Test-Path -LiteralPath $src) {
    Copy-Item -LiteralPath $src -Destination $out -Recurse -Force
    Write-Output "backed up $src"
  }
}
Write-Output "备份完成：$out"
