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
#
# §4 死绑定检测（审计 FE3-03）：「名字被认领」≠「有人调用」——legacy 面
# （legacyBindings.ts，gen_bindings -legacy-ts 产出）逐名在 frontend/src 源码
# 做全名文本命中检查，零命中=死绑定。排除生成物/守卫/类型声明/mock（它们
# 引用名字不代表有真实调用路径）；命中口径=文本命中（注释/测试也算，审计
# 原口径，偏保守：只抓真正全仓不可达的）。新增死绑定 exit 1；在册死名单
# （knownDead，附理由）仅提示；在册条目复活提示清册。
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

# ── 4. 死绑定检测（FE3-03）：legacy 面逐名文本命中 ────────────────────────
$legacyPath = Join-Path $root 'frontend/src/gaea/lib/legacyBindings.ts'
if (Test-Path $legacyPath) {
  # 在册死名单 = 2026-10-03 首轮普查（FE3-03 守卫首次全量实测）：下述 109 名
  # 后端仍在导出、frontend/src（排除生成物/守卫/类型声明/mock 后）零文本命中。
  # 处置留拍板（删除=绑定面 744→641 候选，v4.429 时代在册口径「零调用者绑定
  # 删除需拍板」）；本守卫只拦**普查之外的新增死绑定**（新导出即失联=接线漏），
  # 并在在册条目复活时提示清册。逐名排查/删除走拍板后的独立一刀。
  $knownDead = @(
    'AddOutlineNode', 'AnalyzeStyle', 'BrainCrossRefs', 'BrainWrite', 'BrainstormBranches',
    'BuildBacklinkIndex', 'BuildContextBudget', 'BuildRichContext', 'ChatCharacter', 'ChatCharacterDetail',
    'ChatOutline', 'ChatOutlineNode', 'CheckModuleIntegrity', 'CloseProject', 'ContinueOutline',
    'CreateSnapshot', 'DeleteCharacter', 'DeleteLorebookEntry', 'ExpandOutlineNode', 'ExportHTML',
    'ExtractCharacterHeatmap', 'ExtractEmotionCurve', 'ExtractTimeline', 'FindLorebookTriggers', 'FindUnlinkedMentions',
    'GaeaDataBackupPending', 'GaeaModel', 'GaeaSetEngine', 'GenerateCharacters', 'GenerateDefaultCanvas',
    'GenerateOutlineWithDialogue', 'GenerateSingleCharacter', 'GetActiveASRModel', 'GetActiveTTSModel', 'GetAllEntityNames',
    'GetBacklinks', 'GetBookData', 'GetChatVoiceModel', 'GetCompileTemplates', 'GetDashboard',
    'GetEngineList', 'GetImageBackend', 'GetImageBackendConfig', 'GetLorebookEntries', 'GetStyleProfile',
    'GetTTSConfig', 'GetTTSStatus', 'GetWorldMapImage', 'HerdsmanHealth', 'HerdsmanProbe',
    'ImportStyleProfile', 'InjectMemories', 'ListSnapshots', 'LocalTranslate', 'MainBrainChat',
    'MigrateProjectToV4', 'OfficeCancelJob', 'OfficeExecute', 'OfficeGetJobState', 'OfficeGetMode',
    'OfficeIsTask', 'OfficeListFolder', 'OfficeReadFile', 'OfficeSetMode', 'ParseLinks',
    'QueryEntities', 'RestoreSnapshot', 'ReviewBook', 'RunModule', 'SaveCharacter',
    'SaveCharacters', 'SaveLorebookEntry', 'SaveOutlineNode', 'SaveTTSConfig', 'SaveToken',
    'SaveWorldMapImage', 'SaveWorldviewSection', 'SearchMemories', 'SetDistFS', 'SetPromptFS',
    'Shutdown', 'StartTTSServer', 'Startup', 'StopTTSServer', 'SyncEntityDB',
    'TTSSpeak', 'VoiceGetState', 'VoiceRestartService', 'VoiceSetInputChannel', 'VoiceSetMode',
    'WhisperChat', 'WhisperChatWithSearch', 'WhisperGetConfig', 'WhisperGetEngine', 'WhisperGetEngines',
    'WhisperGetImageModel', 'WhisperGetModel', 'WhisperSetEngine', 'WhisperSetImageModel', 'WhisperSetModel',
    'WhisperTaskPlanResume', 'WhisperTaskPlanStatus', 'WhisperWebSearch',
    'GaeaCallTool', 'GaeaEngines', 'GaeaInit', 'GaeaPermLevel', 'GaeaSkills', 'GaeaTools'
  )

  # 4a. legacy 名单（生成物，勿手改）。行尾可能挂版本注释（生成器
  # legacyTSAnnotations 随生随挂），正则必须容许 // 尾巴，否则漏名。
  $legacyRaw = Get-Content -Raw -Encoding UTF8 $legacyPath
  $legacyRaw = $legacyRaw -replace "`r`n", "`n"
  $legacyNames = @(
    [regex]::Matches($legacyRaw, '^\s*"([^"]+)",?\s*(//.*)?$', [System.Text.RegularExpressions.RegexOptions]::Multiline) |
      ForEach-Object { $_.Groups[1].Value }
  )

  # 4b. 收集 frontend/src 源码文本（排除生成物/守卫/类型声明/mock——引用不算调用路径）
  $excludePatterns = @(
    'gaea/lib/bindingNames.ts',
    'gaea/lib/legacyBindings.ts',
    'gaea/lib/bindingSignatures.ts',
    'gaea/lib/bridge/drift.ts',
    'gaea/lib/spaceBindings.ts',
    'types/wails.d.ts',
    '/mock/'          # 模拟实现不是真实调用路径（bindingSignatures 全名清单也在此目录）
  )
  $sb = New-Object System.Text.StringBuilder
  Get-ChildItem -Path (Join-Path $root 'frontend/src') -Recurse -Include *.ts,*.tsx -File |
    Where-Object {
      $p = $_.FullName -replace '\\', '/'   # 归一分隔符再匹配（FullName 是反斜杠）
      -not ($excludePatterns | Where-Object { $p -like ('*' + $_ + '*') })
    } | ForEach-Object {
      [void]$sb.AppendLine((Get-Content -Raw -Encoding UTF8 $_.FullName))
    }
  $allSource = $sb.ToString()

  # 4c. 逐名全词文本命中（零命中=死绑定候选）
  $dead = @()
  foreach ($n in $legacyNames) {
    if ([regex]::IsMatch($allSource, ('\b' + [regex]::Escape($n) + '\b'))) { continue }
    $dead += $n
  }

  $newDead = @($dead | Where-Object { $knownDead -notcontains $_ })
  $revived = @($knownDead | Where-Object { $dead -notcontains $_ })
  if ($newDead.Count -gt 0) {
    Write-Host "死绑定检测（FE3-03）：发现清单外死绑定 $($newDead.Count) 个（legacy 面共 $($legacyNames.Count) 名，零源码命中）：" -ForegroundColor Red
    $newDead | ForEach-Object { Write-Host "  - $_" }
    Write-Host "处置：后端仍在导出但前端无任何调用路径。确认可删则从 Go 侧下导出并再生 legacyBindings.ts；确属预留则登记进本脚本 knownDead 并附理由。" -ForegroundColor Yellow
    exit 1
  }
  if ($revived.Count -gt 0) {
    Write-Host "死绑定检测：在册死名单中 $($revived.Count) 个已出现源码命中（请清册）：" -ForegroundColor Yellow
    $revived | ForEach-Object { Write-Host "  - $_" }
  }
  Write-Host "OK：死绑定检测通过（legacy $($legacyNames.Count) 名 / 在册死名单 $($knownDead.Count) 条$(if ($dead.Count -gt 0) { '，其中在册死 ' + $dead.Count + ' 条' })）。" -ForegroundColor Green
}
exit 0
