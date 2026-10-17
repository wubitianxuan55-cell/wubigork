# tianxuan-serve.ps1 — 天璇引擎（gufo，Windows gfx1151）启动器：gaea 8094 接入用
#
# 为什么有这个脚本：gufo 自带的 start.cmd 是交互式菜单（每次要按键、上下文/会话数
# 靠现场选），gaea 侧只认一个固定的 OpenAI 兼容端点（默认 127.0.0.1:8094/v1）。
# 本脚本把「模型 → 预设参数 → 8094」固化成一条命令，并落日志供对擂脚本取旁证。
#
# 用法：
#   powershell -File scripts\tianxuan-serve.ps1 -Model 27b          # Qwen3.8-27B + DFlash2
#   powershell -File scripts\tianxuan-serve.ps1 -Model 27b -DryRun  # 只打印将要执行的命令
#
# 约定（与 gaea 侧接线一致）：
#   - 端口 8094：8080 herdsman / 8091 strata / 8888 modelhub / 8092 MoE4All；
#   - 免鉴权：不发 --api-key（Key 留空即不带 Authorization 头）；
#   - 会话数默认 2：gaea 并发档（office 子代理 / 双路对比）实测用得到，显存够；
#   - 磁盘续跑缓存默认开：%LOCALAPPDATA%\gufo\cache（重启后长会话免重算 prefill）。
#
# 设备互斥：同一时刻只跑一个引擎（strata/herdsman/MoE4All 会抢同一份 UMA），
# 中途 kill 可能留 D 状态进程——先确认端口空闲再启动。

[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)][ValidateSet('27b')][string]$Model,
  [int]$Port = 8094,
  [int]$Sessions = 0,   # 0 = 模型预设（27b=2）
  [int]$Context = 32768,
  [ValidateSet('auto', 'dflash2', 'mtp', 'off')][string]$Speculative = 'auto',
  [string]$LogDir = 'C:\AI\wubigrok\.tmp\tianxuan',
  [int]$DraftTokens = 0,
  [ValidateSet('', 'adaptive', 'fixed')][string]$DraftPolicy = '',
  [string[]]$Extra = @(),
  [string]$Binary = '',
  [switch]$NoCacheDisk,
  [switch]$DryRun
)

$ErrorActionPreference = 'Stop'

# 二进制选择：默认用我们的自构建（TheRock 工具链，
# 全部自有改动，与 clones/gufo 工作区同步）；-Binary 显式指定，最后回退 2026-10-03 预构建包。
$selfBuild = 'C:\AI\wubigrok\clones\gufo\build\release\gufo-server.exe'
$prebuilt = Join-Path 'C:\AI\gufo-win\gufo-windows-2026-10-03' 'bin\gufo.exe'
if ($Binary -ne '' -and (Test-Path $Binary)) { $gufo = $Binary }
elseif (Test-Path $selfBuild) { $gufo = $selfBuild }
elseif (Test-Path $prebuilt) { $gufo = $prebuilt }
else { throw "找不到 gufo 二进制（自构建 $selfBuild / 预构建 $prebuilt 都不在）" }
$env:PATH = (Split-Path -Parent $gufo) + ';' + $env:PATH

# 模型与 drafter 路径：27B 用的是 herdsman 目录里已装的那份 UD-Q4_K_M + gufo-win 里的 DFlash2 draft。
$models = @{
  '27b' = @{
    Gguf      = 'C:\Users\wubi\.herdsman\models\Qwen3.8-27B\windows\amd64\amd\Qwen3.8-27B-UD-Q4_K_M.gguf'
    Mmproj    = 'C:\Users\wubi\.herdsman\models\Qwen3.8-27B\windows\amd64\amd\mmproj-BF16.gguf'
    Draft     = 'C:\AI\gufo-win\Qwen3.8-27B-DFlash2-Q4_K_M.gguf'
    Spec      = 'dflash2'
    ModelName = 'Qwen3.8-27B-DFlash2'
    Sessions  = 2
    # 实测（2026-10-17 gufo bench，Q4 + DFlash2）：AR 12.48 → adaptive d7 20.41 →
    # adaptive d4 22.44；fixed 策略 17.02（接受率 25.5%，别用）。d4 是最优档。
    Extra     = @('--draft-tokens', '4')
  }
}
$m = $models[$Model]
if (-not (Test-Path $m.Gguf)) { throw "模型不存在: $($m.Gguf)" }
if ($Speculative -eq 'auto') { $Speculative = $m.Spec }

# 预设自带的引擎环境变量（Env 表）在 Start-Process 前套用，
# 子进程按继承环境取用。
if ($m.Env) {
  foreach ($k in $m.Env.Keys) { Set-Item -Path ("env:" + $k) -Value ([string]$m.Env[$k]) }
}

if ($Sessions -le 0) { $Sessions = [int]$m.Sessions }
$argv = @('serve', 'llm', '-m', $m.Gguf, '-p', "$Port", '-j', "$Sessions", '-c', "$Context",
          '--served-model-name', $m.ModelName, '--speculative', $Speculative)
if ($m.Mmproj -and (Test-Path $m.Mmproj)) { $argv += @('--mmproj', $m.Mmproj) }
if ($Speculative -eq 'dflash2' -and $m.Draft -and (Test-Path $m.Draft)) {
  $argv += @('--dflash-model', $m.Draft)
}
# 调优旋钮（批次 3 扫参用）：-DraftTokens / -DraftPolicy / -Extra 原样透传给 gufo；
# 模型预设自带的 Extra 先上，命令行同名项追加在后（gufo 取最后一个同名选项）。
if ($m.Extra -and $m.Extra.Count -gt 0) { $argv += $m.Extra }
if ($DraftTokens -gt 0) { $argv += @('-d', "$DraftTokens") }
if ($DraftPolicy -ne '') { $argv += @('--draft-policy', $DraftPolicy) }
if ($Extra.Count -gt 0) { $argv += $Extra }
if (-not $NoCacheDisk) {
  $cache = Join-Path $env:LOCALAPPDATA 'gufo\cache'
  New-Item -ItemType Directory -Force $cache | Out-Null
  $argv += @('--cache-disk', $cache)
}

if ($DryRun) {
  Write-Output ("& '{0}' {1}" -f $gufo, ($argv -join ' '))
  return
}

# 端口占用先查：与其它引擎互斥，撞端口时 gufo 的报错不如这里直白。
$busy = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
if ($busy) { throw "端口 $Port 已被 pid=$($busy.OwningProcess) 占用——同一时刻只跑一个引擎（8080 herdsman / 8091 strata / 8092 MoE4All / 8888 modelhub）" }

New-Item -ItemType Directory -Force $LogDir | Out-Null
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$log = Join-Path $LogDir ("serve-{0}-{1}.log" -f $Model, $stamp)
$errLog = Join-Path $LogDir ("serve-{0}-{1}.err.log" -f $Model, $stamp)
"tianxuan serve: model=$Model spec=$Speculative port=$Port sessions=$Sessions ctx=$Context" | Tee-Object -FilePath $log
Write-Output ("& '{0}' {1}" -f $gufo, ($argv -join ' ')) | Tee-Object -FilePath $log -Append
Write-Output "stdout: $log"
Write-Output "stderr: $errLog"

# 只用 Start-Process 落盘：PowerShell 5.1 下 `& gufo 2>&1 | Tee-Object` 会把引擎每
# 一条 stderr 日志当成 NativeCommandError 终止脚本（实测踩过，引擎随包装脚本一起死）。
# 前台等价：本脚本等到引擎退出才返回；关窗口/任务管理器结束进程，不硬杀（D 状态风险）。
$argLine = ($argv | ForEach-Object { if ($_ -match '\s') { '"{0}"' -f $_ } else { $_ } }) -join ' '
$proc = Start-Process -FilePath $gufo -ArgumentList $argLine -NoNewWindow -PassThru `
  -RedirectStandardOutput $log -RedirectStandardError $errLog
Write-Output "pid: $($proc.Id)"
$proc.WaitForExit()
Write-Output "exit: $($proc.ExitCode)"
