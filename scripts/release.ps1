# release.ps1 — 发版台账自动化（2026-09-27，审计优化档 P0-3）
#
# 把一次发版的机械动作收进一条命令，人手只写「发布说明正文」：
#   1. 预检：git 工作树干净 + 三处版本漂移闸通过
#   2. 三处版本源同步（复用 scripts\sync-version.ps1）
#   3. 构建：cmd /c build.bat（wails build + 产物新鲜度 + 冒烟 + 桌面副本；
#      -SkipSmoke 跳冒烟仅限快速迭代，发版禁用）
#   4. 产物搬运：build\bin\gaea.exe → releases\gaea-v<X>.exe，
#      写 SHA256SUMS-v<X>.txt（`<sha256>  <文件名>` 两空格，可 sha256sum -c 直验）
#   5. 保留策略：只留最近 5 版 exe，删第 6 新
#   6. releases\README.md：实存二进制行 / 校验和计数 / 发布说明计数 / V4_RECENT 表
#      （插新行裁到 34 席）
#   7. 根 README.md：当前版本行
#   8. CHANGELOG.md：追加骨架条目（已有同版条目则跳过，不重复插）
#
# 人手收尾（≤3 件事）：写 releases\v<X>.md 正文、填 CHANGELOG 骨架文本、
# commit（"release: vX.Y.Z …"）+ tag + push。
#
# 用法：
#   powershell -NoProfile -File scripts\release.ps1 -Version 4.421.0
#   powershell -NoProfile -File scripts\release.ps1 -Version 4.421.0 -DryRun   # 只预检+打印计划
#   powershell -NoProfile -File scripts\release.ps1 -Version 4.421.0 -SkipBuild # 用 build\bin 现有产物
#
# 幂等性注记（2026-09-27 实证）：对已发版本重跑，内容零变化（SUMS 逐字节一致、
# 表/计数/骨架均跳过）；git status 可能报 releases/* 幻影 M——autocrlf EOL 伪影，
# 内容 diff 为空，git add 后即净，勿当真回归。
param(
    [Parameter(Mandatory = $true)][string]$Version,
    [switch]$SkipBuild,
    [switch]$SkipSmoke,
    [switch]$DryRun
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$RecentExeKeep = 5      # releases 根保留的 exe 数（用户 2026-09-10 拍板）
$RecentSeats   = 34     # V4_RECENT 表席位
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)

if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    Write-Host "ERROR: 版本号格式应为 X.Y.Z，得到：$Version" -ForegroundColor Red
    exit 1
}
$tag = "v$Version"

function Read-Text($path) { [System.IO.File]::ReadAllText($path) }
function Write-Text($path, $content) { [System.IO.File]::WriteAllText($path, $content, $utf8NoBom) }

# ── 1. 预检 ────────────────────────────────────────────────────────────────
$dirty = @(& git status --porcelain) | Where-Object { $_ -ne '' }
if ($dirty.Count -gt 0) {
    Write-Host "ERROR: 工作树不干净（发版提交应是纯台账+已提交代码），先 commit/stash：" -ForegroundColor Red
    $dirty | ForEach-Object { Write-Host "  $_" }
    exit 1
}
Write-Host "[1/8] 预检：工作树干净"
& powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'check-version-drift.ps1')
if ($LASTEXITCODE -ne 0) { throw "版本漂移闸未过（exit $LASTEXITCODE）" }

# ── 2. 版本三处同步 ────────────────────────────────────────────────────────
Write-Host "[2/8] 三处版本源同步 → $Version"
if (-not $DryRun) {
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'sync-version.ps1') -Version $Version
    if ($LASTEXITCODE -ne 0) { throw "sync-version 失败（exit $LASTEXITCODE）" }
}

if ($DryRun) {
    Write-Host "`n[DryRun] 计划（未执行任何写操作）："
    Write-Host "  - cmd /c build.bat $(if ($SkipSmoke) { 'skip-smoke' })（-SkipBuild 可跳过）"
    Write-Host "  - build\bin\gaea.exe → releases\gaea-$tag.exe + SHA256SUMS-$tag.txt"
    Write-Host "  - 保留策略：只留最近 $RecentExeKeep 版 exe"
    Write-Host "  - releases\README.md：实存二进制行/校验和计数/发布说明计数/V4_RECENT 插 $tag 裁到 $RecentSeats 席"
    Write-Host "  - README.md：当前版本行 → $Version"
    Write-Host "  - CHANGELOG.md：追加 $tag 骨架条目（已有则跳过）"
    Write-Host "  - 人手收尾：releases\$tag.md 正文 + CHANGELOG 填文本 + commit+tag"
    exit 0
}

# ── 3. 构建 ────────────────────────────────────────────────────────────────
$binExe = Join-Path $root 'build\bin\gaea.exe'
if ($SkipBuild) {
    if (-not (Test-Path $binExe)) { throw "-SkipBuild 但 build\bin\gaea.exe 不存在" }
    Write-Host "[3/8] 跳过构建（用现有 build\bin\gaea.exe）"
} else {
    Write-Host "[3/8] 构建（wails build + 冒烟 + 桌面副本，约 1~3 分钟）"
    $buildArg = if ($SkipSmoke) { 'skip-smoke' } else { '' }
    & cmd /c build.bat $buildArg
    if ($LASTEXITCODE -ne 0) { throw "build.bat 失败（exit $LASTEXITCODE）" }
}

# ── 4. 产物搬运 + SUMS ─────────────────────────────────────────────────────
Write-Host "[4/8] 产物搬运 + SHA256SUMS-$tag.txt"
$relExe = Join-Path $root "releases\gaea-$tag.exe"
if ((Test-Path $relExe) -and ($SkipBuild)) {
    # 防覆盖：已入档的 exe 不得被不同产物悄悄替换（身份以 SUMS 为准）
    $old = (Get-FileHash $relExe -Algorithm SHA256).Hash.ToLower()
    $new = (Get-FileHash $binExe -Algorithm SHA256).Hash.ToLower()
    if ($old -ne $new) { throw "releases\gaea-$tag.exe 已存在且与 build\bin 哈希不同——拒绝覆盖（真实发版请去掉 -SkipBuild 或先清档）" }
}
Copy-Item $binExe $relExe -Force
$hash = (Get-FileHash $relExe -Algorithm SHA256).Hash.ToLower()
# 尾换行与历史 SUMS 格式逐字节一致（HEAD 版本带 \n，勿删）
Write-Text (Join-Path $root "releases\SHA256SUMS-$tag.txt") "$hash  gaea-$tag.exe`n"
Write-Host "  SHA256 = $hash"

# ── 5. 保留策略：删第 6 新 ─────────────────────────────────────────────────
Write-Host "[5/8] 保留策略（只留最近 $RecentExeKeep 版）"
$exes = Get-ChildItem (Join-Path $root 'releases') -Filter 'gaea-v*.exe' |
    ForEach-Object { if ($_.BaseName -match '^gaea-v(\d+\.\d+\.\d+)$') {
        [pscustomobject]@{ V = [version]$Matches[1]; Path = $_.FullName; Name = $_.Name }
    } } | Sort-Object V -Descending
if ($exes.Count -gt $RecentExeKeep) {
    $exes | Select-Object -Skip $RecentExeKeep | ForEach-Object {
        Remove-Item $_.Path -Force
        Write-Host "  已删 $($_.Name)（第 $($RecentExeKeep+1) 新）"
    }
} else {
    Write-Host "  实存 $($exes.Count) 版 ≤ $RecentExeKeep，无删除"
}

# ── 6. releases\README.md ──────────────────────────────────────────────────
Write-Host "[6/8] releases\README.md"
$relReadmePath = Join-Path $root 'releases\README.md'
$relReadme = Read-Text $relReadmePath

# 6a. 实存二进制行（从磁盘实况重生成，历史对账注记不再手维护）
$keepNames = (($exes | Select-Object -First $RecentExeKeep).Name -replace '^gaea-', '' -replace '\.exe$', '') -join ' / '
$relReadme = [regex]::Replace($relReadme,
    '\*\*只保留最近 5 个版本的 exe\*\*（当前实存二进制 = [^）]*）',
    "**只保留最近 $RecentExeKeep 个版本的 exe**（当前实存二进制 = $keepNames）")

# 6b. 校验和计数 + 发布说明计数（实数，勿在文中硬写）
#     校验和按**全目录递归**（releases/ 根 + archive/）——check-docs 就是这样对账的；
#     早期写成非递归只数根目录，重生成后文中数字（429）与实测（449）不符、卫生守卫红。
$sumsCount  = (Get-ChildItem (Join-Path $root 'releases') -Recurse -Filter 'SHA256SUMS-*.txt').Count
$notesCount = (Get-ChildItem (Join-Path $root 'releases') -Filter 'v*.md' |
    Where-Object { $_.Name -match '^v\d' }).Count
$relReadme = [regex]::Replace($relReadme, '现有 \*\*\d+ 份校验和\*\*', "现有 **$sumsCount 份校验和**")
$relReadme = [regex]::Replace($relReadme, '\d+ 个发布说明', "$notesCount 个发布说明")

# 6b-2. 「共 N 份发布说明」自述行与「最近 N 版」——卫生守卫对账口径（v4.471 发版实测
#       两处不随版同步会挂守卫：共数=全目录递归含 archive/；最近 N 版=V4_RECENT 席位）。
#       v4.472 再修：脚本跑时 v<tag>.md 尚未写（人手收尾第 1 件），实数天然差 1——
#       按「收尾完成后」口径计入待写文档（已存在则按实数，重跑/补档幂等）。
$notesRoot = @((Get-ChildItem (Join-Path $root 'releases') -Filter 'v*.md') | Where-Object { $_.Name -match '^v\d+\.\d+\.\d+\.md$' }).Count
$notesArch = @((Get-ChildItem (Join-Path $root 'releases\archive') -Filter 'v*.md' -ErrorAction SilentlyContinue) | Where-Object { $_.Name -match '^v\d+\.\d+\.\d+\.md$' }).Count
$pendingDoc = if (Test-Path (Join-Path $root "releases\$tag.md")) { 0 } else { 1 }
$notesRoot = $notesRoot + $pendingDoc
$notesTotal = $notesRoot + $notesArch
$today = (Get-Date).ToString('yyyy-MM-dd')
$relReadme = [regex]::Replace($relReadme,
    '本目录共 \d+ 份发布说明（截至 \d{4}-\d{2}-\d{2} 实测：根目录 \d+ \+ `archive/` \d+',
    "本目录共 $notesTotal 份发布说明（截至 $today 实测：根目录 $notesRoot + ``archive/`` $notesArch")
$relReadme = [regex]::Replace($relReadme, '对账），最近 \d+ 版', "对账），最近 $RecentSeats 版")

# 6c. V4_RECENT 表：插新行（已存在则跳过）并裁到 $RecentSeats 席
$startM = '<!-- V4_RECENT:start -->'; $endM = '<!-- V4_RECENT:end -->'
$sx = $relReadme.IndexOf($startM); $ex = $relReadme.IndexOf($endM)
if ($sx -lt 0 -or $ex -lt 0 -or $ex -lt $sx) { throw 'releases/README.md 缺 V4_RECENT 标记' }
$block = $relReadme.Substring($sx + $startM.Length, $ex - $sx - $startM.Length)
$rows = @($block -split "`n" | Where-Object { $_ -match '^\s*-\s\[v\d' } | ForEach-Object { $_.TrimEnd() })
if (-not ($rows | Where-Object { $_ -match [regex]::Escape("./$tag.md") })) {
    $rows = @("- [$tag.md](./$tag.md) — $tag「<待填标题>」") + $rows
} else {
    Write-Host "  V4_RECENT 已含 $tag 行，跳过插入"
}
$rows = $rows | Select-Object -First $RecentSeats
$newBlock = "`n`n" + ($rows -join "`n") + "`n`n"
$relReadme = $relReadme.Remove($sx + $startM.Length, $ex - $sx - $startM.Length).Insert($sx + $startM.Length, $newBlock)
Write-Text $relReadmePath $relReadme

# ── 7. 根 README.md 当前版本行 ─────────────────────────────────────────────
Write-Host "[7/8] README.md 当前版本行"
$readmePath = Join-Path $root 'README.md'
$readme = Read-Text $readmePath
$readme = [regex]::Replace($readme, '当前 \*\*v\d+\.\d+\.\d+\*\*。', "当前 **$tag**。")
Write-Text $readmePath $readme

# ── 8. CHANGELOG 骨架 ──────────────────────────────────────────────────────
Write-Host "[8/8] CHANGELOG.md 骨架条目"
$clPath = Join-Path $root 'CHANGELOG.md'
$cl = Read-Text $clPath
if ($cl -match "(?m)^## $([regex]::Escape($tag)) ") {
    Write-Host "  CHANGELOG 已含 $tag 条目，跳过"
} else {
    $today = Get-Date -Format 'yyyy-MM-dd'
    $skeleton = "## $tag · <待填：一句话标题>（$today）`n> <待填：摘要段——背景/落地/测试/门禁/产物/文档>`n`n"
    Write-Text $clPath ($skeleton + $cl)
}

Write-Host ""
Write-Host "=== 机械步骤完成 ===" -ForegroundColor Green
Write-Host "人手收尾（3 件事）："
Write-Host "  1. 写 releases\$tag.md 发布说明正文（根因/修复/验证/产物）"
Write-Host "  2. 填 CHANGELOG.md 的 $tag 骨架（标题+摘要段）"
Write-Host "  3. commit（release: $tag …）+ tag $tag + push"
