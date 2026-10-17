param([int]$Parallel = 8, [string]$Url = 'https://hf-mirror.com/unsloth/Qwen3.6-35B-A3B-MTP-GGUF/resolve/main/')
$ErrorActionPreference = 'Stop'
$dir  = 'D:\gufo-models\35b'
$name = 'Qwen3.6-35B-A3B-UD-Q8_K_XL.gguf'
$target = Join-Path $dir $name
$url = $Url + $name
$L = 39099447584

# 清掉上一轮（含被误留的旧 writer）留下的碎片，避免口径混用。
Get-ChildItem $dir -Filter 'part-*.bin' | Remove-Item -Force -ErrorAction SilentlyContinue
Get-Process curl -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

$S = 0
if (Test-Path $target) { $S = (Get-Item $target).Length }
if ($S -eq $L) { "already complete: $S"; return }
if ($S -gt $L) { throw "target larger than expected: $S > $L" }
$remain = $L - $S
$chunk = [math]::Ceiling($remain / $Parallel)
$procs = @()
$parts = @()
for ($i = 0; $i -lt $Parallel; $i++) {
  $start = $S + $i * $chunk
  if ($start -ge $L) { break }
  $end = [math]::Min($L - 1, $start + $chunk - 1)
  $part = Join-Path $dir ('part-{0}.bin' -f $i)
  $parts += [pscustomobject]@{ Path = $part; Start = $start; Expect = ($end - $start + 1) }
  $cargs = @('-sSL', '--retry', '20', '--retry-all-errors', '--retry-delay', '3',
             '--range', "$start-$end", '-o', $part, $url)
  $procs += Start-Process -FilePath 'curl.exe' -ArgumentList $cargs -PassThru -WindowStyle Hidden
}
"launched $($parts.Count) range downloads from offset $S"
$procs | Wait-Process
foreach ($p in $parts) {
  $len = (Get-Item $p.Path).Length
  if ($len -ne $p.Expect) { throw "part size mismatch $($p.Path): $len != $($p.Expect)" }
}
# 按起始偏移定位写入（不依赖「文件恰好在 S 截止」这一前提）。
$fs = [System.IO.File]::Open($target, [System.IO.FileMode]::Open, [System.IO.FileAccess]::Write, [System.IO.FileShare]::None)
try {
  foreach ($p in $parts) {
    $fs.Seek($p.Start, [System.IO.SeekOrigin]::Begin) | Out-Null
    $in = [System.IO.File]::OpenRead($p.Path)
    try { $in.CopyTo($fs, 8MB) } finally { $in.Dispose() }
  }
} finally { $fs.Dispose() }
foreach ($p in $parts) { Remove-Item $p.Path -Force }
$final = (Get-Item $target).Length
"final: $final / $L"
if ($final -ne $L) { throw 'merged size mismatch' }
'DONE'
