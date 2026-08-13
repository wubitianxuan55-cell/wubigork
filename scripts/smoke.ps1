# gaea 发布冒烟测试：产物构建后启动 exe → 探活 HTTP 桥接 → 确认进程存活 → 回收。
# 用途：wails build 之后、发布之前执行，把「启动即白屏/崩溃」挡在发布前。
# 用法：pwsh -File scripts\smoke.ps1 [-ExePath build\bin\gaea.exe] [-Port 18999] [-TimeoutSec 90]
param(
    [string]$ExePath = '',
    [int]$Port = 18999,
    [int]$TimeoutSec = 90
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if (-not $ExePath) { $ExePath = Join-Path $root 'build\bin\gaea.exe' }
if (-not (Test-Path $ExePath)) { Write-Host "冒烟失败：产物不存在 $ExePath"; exit 1 }

# 隔离运行时临时目录（与 build.bat 一致，规避 SAC/Temp 策略）
$tmp = Join-Path $root '.tmp'
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
$env:TMP = $tmp
$env:TEMP = $tmp
# 避免与用户正在运行的 gaea 实例（默认 8080 桥接）抢端口
$env:GAEA_HTTP_PORT = "$Port"

Write-Host "=== 冒烟：启动 $ExePath（HTTP 桥接 127.0.0.1:$Port）==="
$proc = Start-Process -FilePath $ExePath -PassThru
$health = "http://127.0.0.1:$Port/api/health"
$deadline = (Get-Date).AddSeconds($TimeoutSec)
$ok = $false
$respText = ''
try {
    while ((Get-Date) -lt $deadline) {
        Start-Sleep -Milliseconds 800
        if ($proc.HasExited) {
            Write-Host "冒烟失败：进程启动后即退出，ExitCode=$($proc.ExitCode)"
            exit 1
        }
        try {
            $resp = Invoke-WebRequest -Uri $health -UseBasicParsing -TimeoutSec 2
            if ($resp.StatusCode -eq 200) {
                $ok = $true
                $respText = $resp.Content
                break
            }
        } catch {
            # 未就绪，继续轮询
        }
    }
    if ($ok) {
        Write-Host "冒烟通过：/api/health 200，响应: $respText"
        exit 0
    }
    Write-Host "冒烟失败：${TimeoutSec}s 内 /api/health 未就绪（进程存活但桥接未起来）"
    exit 1
} finally {
    if (-not $proc.HasExited) {
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
        Write-Host "已回收冒烟进程 PID=$($proc.Id)"
    }
}
