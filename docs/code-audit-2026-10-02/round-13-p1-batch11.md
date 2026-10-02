# 第十三轮修复记录（批次十一：P0 收官 + 静默失败诚实化 + 原语单源）· 2026-10-02

> **承接**：批次一~十（`b6f092f7` → `6fe5befa`）。
> **范围**：审计 25 条 P0 中最后 3 条「一刀可落」项（AP9-01 / AP9-02 / FE6-02）+ 一批静默失败诚实化（P1/P2）+ 截断族与 GenUI 上限单源收敛。
> **范式**：4 条互斥并行子代理线（线1 Go/internal-app · 线2 Go/非 app 包 · 线3 前端 · 线4 原语单源），主代理定契约、裁决降级、收线缝合、全量验证。
> **验证**：（收线补：`go build`/`go vet` 0；触达包 `go test -count=1` 全绿；`tsc -b` exit 0；eslint 0；vitest 触达面全绿；三份守卫脚本）

---

## 〇、动手前的底账更正（三处，先纠正再开刀）

### 更正 1：P0 状态不能用「是否在 round 记录里被提及」判定

最初按「fid 是否出现在 `round-*.md`」判定，得出「只剩 3 条 P0」——**方法错误**：`round-1-p0-status.md` 会**罗列**仍开放的条目，被提及 ≠ 已修。改用「状态表 + 逐行读码实测」复核后的真实底账（按入口 `README.md` §3 的 25 行口径）：

| 状态 | 条数 | 明细 |
|---|---|---|
| 已闭合 | **19** | #1 GA5-01、#2 GA5-02、#3 AP4-02、#4 AP1-01、#5 AP1-02、#6 GA3-01、#7 GA3-02、#8 GA5-03、#10 AP4-04、#11 AP4-03、#12 AP6-01、#13 AP6-02、#14 AP7-01、#18 FE6-01、#20 FE7-01、#21 FE7-02、#22 IN2-01、#23 X1-01、#24 X1-02 |
| 本轮处理 | **3** | #15 AP9-01、#16 AP9-02、#19 FE6-02 |
| 待拍板（结构项） | **2** | #9 AP4-01 三套编排合并、#25 X1-03 五重 `*core` 嵌入裸构造 |
| 已证伪撤销 | **1** | #17 FE2-01（`setAllSelected` 定义在场，本轮二次确认） |

**其中 #6/#7/#8/#12 由「另一执行体」批次落地，未在本目录任何 round 记录里留痕**——`round-2-fixes.md` §三只以一句「另一执行体 13:05 批次（#6/#7/#8/#12）不在本记录重复展开」带过，而 `round-1-p0-status.md` 仍把它们列为「仍开放」，两处口径互斥。本轮**逐行读码实测**确认这 4 条确已在树：

| P0 | 实测证据（当前工作树） |
|---|---|
| #6 GA3-01 | `internal/gaea/knowledge/sqlite.go` `func (b *sqliteBackend) List() ([]EntrySummary, error)`，`db.Query` 失败分支带注释「审计 P0#6 GA3-01：读失败不得吞成空切片」 |
| #7 GA3-02 | `internal/gaea/tool/builtin/memory_search.go` 已从包级单例改为**按空间分槽**：`memoryIndexBySpace` + `memoryIndexMu`、`SetMemorySearchIndexForSpace`、`MemorySearchIndexForSpace` 可选接口 + ctx 盖章解析 |
| #8 GA5-03 | `internal/gaea/trajectory/fold.go:342` `r := f.assistantRecord(e)` / `if r == nil { // 兜底绝不解引用 nil` / `return }` |
| #12 AP6-01 | `internal/voice/voice_manager.go` `SetWhisperChatFn` 已改 `m.whisperChatPtr.Store(&fn)`（`atomic.Pointer`） |

### 更正 2：审计入口表与状态复核表不是同一张表

`README.md` §3 的 25 行与 `round-1-p0-status.md` 的 25 行**编号与内容都不一致**：后者含 3 条重复行（其 #19/#20/#21 实为 GA5-01/02/03 的重复登记）、且**不含** AP9-01/AP9-02/FE6-02 三条。README 脚注「上表 25 行中 #19/#20/#21 与 #1/#2/#8 是同一条……独立 P0 实为 22 条」**是按后者编号写的**，挂在 README 自己的表上时指向的是 FE6 麦克风 / FE7 baselines / FE7 assignment 三行——**该脚注在 README 口径下不成立**。按 README/`machine/findings_all.json` 口径：**25 行互不重复，与 25 条 P0 findings 一一对应**；「独立 22 条」只在 `round-1-p0-status.md` 那张表下成立。本更正不改变任何实质结论，只纠正引用口径。

### 更正 3：一条任务在动手前被降级（切口纪律）

`GA6-09`（`cost.Store.Search/List` 加 `error`）原列入本批，实测其调用点横跨 4 个包 **20+ 处**：`internal/app`（`gaea_cost_compose.go:83`、`gaea_cost_graph.go:36`、`gaea_cost_import_vision.go:650`、`gaea_cost_inquiry.go:60`、`gaea_cost_rerank.go:112`、`gaea_semantic_index.go:20`、`gaea_semantic_search.go:98`、`memory_hub.go:237/408/566/583`）、`internal/gaea/tool/builtin`（`cost_compose.go:69`、`cost_tools.go:105/146/364`）、`internal/gaea/costimport/costimport.go:294`、`internal/gaea/pricefeed/pricefeed.go:312`。改签名会**同时横跨线 1 的 `internal/app` 与线 4 的 `internal/gaea/tool/builtin`**（同 Go 包不可并行），违反切口纪律 → **当场降级为「只出迁移清单、不动刀」**，登记为独立一刀（见 §四）。

---

## 一、已落地

### 线 2 · 非 app 包静默失败诚实化（4 条已交付）

**GA3-13 · 知识库迁移失败不再静默**（`knowledge/sqlite.go`、`service.go`、`store.go`）：迁移标记三态化——「已迁移 / 标记不存在（正常，继续迁移）/ 标记**读取失败**（异常，不再当作未迁移反复重跑）」；`filepath.Glob` 错误上抛；跳过文件计数 + `slog.Warn`。`Service` 新增导出 `MigrationState{Ran, Failed, Reason}`，迁移失败记状态但**不阻断** Store 可用性（启动路径与面板可查）。`fileBackend.Index()` 读 INDEX.md 失败改为 `slog.Warn` 留痕（此前只在文案里降级）。
证据：`TestMigrateMarkerReadFailureIsNotUnmigrated`、`TestMigrateGlobFailureReported`、`TestServiceMigrationFailureVisible`（对照 `TestServiceMigrationSuccessState` / `TestMigrateMarkerAbsentProceeds`）；**反向证据**＝临时还原旧逻辑后前两例 FAIL（`n=1` / `err=nil`）。

**GA4-11 · 备份与回滚不再「部分成功报成成功」**（`backup/backup.go`）：`Create` 的 `os.Stat` 失败按 `os.IsNotExist` → 追加 `Warnings`「源缺失: \<路径\>」并跳过；其它错误（权限/路径错/磁盘掉线）→ **中止并返回 error**（此前一律 `continue`，zip 会静默少数据而 `EntryCount` 看着正常）。`RollbackBefore` 收集失败项、`failures` 非空即返回 `(false, 聚合 error)`（逐条点名 + 已回滚计数），失败时保留 `.restore-before` 供重试；**不再用 `moved > 0` 当成功标志**。
证据：`TestCreateStatErrorAborts`、`TestCreateMissingSourceRecordedAsWarning`、`TestRollbackBeforePartialFailureIsNotSuccess`、`TestRollbackBeforeAllSucceed`、`TestRollbackBeforeNoBeforeDir`；反向证据＝还原旧逻辑后前两例 FAIL（`ok=true` / 不报错）。

**GA4-13 + GA4-14 · 乱码与拆文件遗留注释**：见 §〇 更正 4（乱码实为 **18 行**、含用户可见错误文案字面量）与下方「审计原文纠错」。纯注释/文档改动，零行为变化，`gofmt -l` 空。

**IN3-15 · 截屏不再把失败帧当成功**（`screen/screen_windows.go`）：`SelectObject`（含 `HGDI_ERROR`）与 `BitBlt` 返回码均检查，失败走 `syscall.GetLastError()` 报错（资源释放仍由既有 `defer` 负责）。**超出审计的发现**：源矩形与虚拟桌面**零交集**时 BitBlt 返回非 0 且产出全黑帧——光查返回码查不出「成功但空白」，故另加交集前置守卫（探针实测越界区域 `err=nil`、非黑像素 `0/256`）。
证据：`TestCaptureAreaOffVirtualScreenErrors`、`TestCaptureAreaPartialOverlapAllowed`（**对照：部分相交仍可捕，确认没有顺手关掉可行路径**）、`TestGdiFailureYieldsError`。

**线 2 验证**：`go test ./internal/gaea/cost/... ./internal/gaea/knowledge/... ./internal/gaea/backup/... ./internal/gaea/control/... ./internal/screen/... -count=1` 全绿（cost 1.690s / knowledge 1.771s / backup 1.333s / control 2.387s / screen 0.649s）；6 包 `go vet` OK；`gofmt -l` 空；14 个改动文件 UTF-8 无 BOM。

### 线 1 · app 内 P0 收官与静默失败（4 条 + 顺带；13 文件，无越界）

**AP9-01（P0）数字生命读取全吞错 → 逐表降级 + `Warnings` 契约**（`herdsman_digitallife.go`）：`digitalCount` → `(int, error)`（此前连 `Scan` 错误都不看）；`digitalJSON` → `(T, error)`（降级说明要能说清「坏在哪」）；8 张计数表**逐表降级**（失败计 Warnings 并保持 0，让面板能同时显示「有数据」与「读不到」的部分）；4 处明细 `Query` 失败 + 5 处 `rows.Err()` 一并降级；单行 `Scan`/JSON 失败按表**汇总**（`digitalRowFailures`：条数 + 首条示例错误，既不静默也不刷屏）；**8 表全失败（库损坏）才致命**，且此时 `out.Warnings` 已填好。契约字段 `Warnings []string \`json:"warnings"\`` 初始化成空切片（JSON 出 `[]` 而非 null）；绑定方法签名零变更。
证据：`TestLoadHerdsmanDigitalLife`（补 `warnings==0` 断言）、`_TableRenamedWarns`、`_BadRowJSONWarns`、`_CorruptDB`、`TestDigitalCountReturnsError` 全 PASS；**改坏能红**：临时把 `digitalCount` 恢复成吞错 + 坏行静默 `continue` → 三条降级用例全 FAIL。

**AP9-02（P0）Herdsman 超时只杀父进程 → 杀整棵进程树**（`herdsman_lifecycle.go`）：新增 `killProcessTree(pid)`（Windows 走 `taskkill /T /F`，**趁父进程仍存活**枚举子孙；非 Windows 退化单杀）；`runHerdsmanCLI` 由 `exec.CommandContext` 改为 `Start` + 等 ctx，超时杀树 + 有界回收 + 日志（PID 与清理结果），错误文案明确「超时 + PID + 清理结果」。**缓冲只在 `<-waited` 胜出后读**（主代理复审指出的 data race，按方案 a 修，纪律写进注释）。
证据：`TestKillProcessTree_KillsGrandchild`（2.60s）、**`TestParentOnlyKillLeavesGrandchild`（3.57s，反向对照——直接证明旧写法会留下孙进程）**、`TestRunHerdsmanCLI_TimeoutKillsProcessTree`（5.55s）PASS；**改坏能红**：去掉 `taskkill` 的 `/T` → 12.11s / 20.54s FAIL「进程 [18824] / [41160] 在 10s 内未消失」。

**AP8-16（P2）`SetFeatureModel` 落盘失败仍报成功**（`feature_model_handler.go`）：三处 `config.Save` 经 `featureModelSave` 缝 + `errors.Join` 收集；**任一失败即回滚内存 engine/model/enabled 并返回 error**；同 bug 类的 `SetFeatureModelEnabled` 一并补回滚。
证据：`TestSetFeatureModel_SaveFailureRollsBack`、`TestSetFeatureModelEnabled_SaveFailureRollsBack`、`TestSetFeatureModel_SaveSuccessKeepsBinding` PASS；**改坏能红** → FAIL「落盘失败必须返回 error（此前 return nil 伪成功）」。

**AP5-16（P2）后台 panic 只记日志、状态位不复位**：新增 `gaeaBackgroundPanicNotice` 在 `*core` 统一收口（`slog.Error` + **既有** `gaea-event` Notice/LevelWarn，**未新造通道**）；`gaea_dream.go` panic 后复位 running/`last*` 并通知；**`gaea_whisper_proactive.go` 把 recover **移入每轮 `runTick` 体内**——单次 panic 不再让主动关心循环永久停摆（本批最大行为变化）；`gaea_subagent_followup.go` 令牌复位先于通知。
证据：`TestMaybeDreamAfterTurn_PanicResetsStateAndNotifies`、`TestGaeaBackgroundPanicNotice_DefaultsToGaeaEvent`、`TestProactiveLoop_ContinuesAfterPanic`、`_StopsOnSignal`、`TestGaeaSubagentFollowUp_PanicReleasesClaimAndNotifies` PASS；**改坏能红**：三处 recover 改回「只 `slog.Error`」+ whisper recover 移回循环外层 → 3 条 FAIL（含「首轮 panic 后循环未继续」）；`TEMP-BREAK` 残留计数 = 0。

**顺带 · `gaea_dag.go` 丢弃落盘错误留痕**：`dagMarkNodeLogged` 仅加 `slog.Warn`，**`dagMarkNode` 签名/返回类型零变更**；替换 6 处 `dagExecute` + 1 处 `dagFailAll`（**审计清单漏了 `dagFailAll`**）。

**线 1 验证（真实输出）**：`go build ./...` exit 0；`go vet ./...` exit 0；`go test ./internal/app/... -count=1` → `ok internal/app 109.357s` + `ok internal/app/board 0.514s`。**主代理独立复跑**：`ok internal/app 117.684s` + `ok internal/app/board 0.521s`，exit 0。

**主代理收线补做（app 侧原样留给主代理的三处）**：
| 项 | 处置 |
|---|---|
| `gaea_subagents.go` `truncateRunes` | 退化为一行委托 `strutil.TruncateRunesEllipsis`，注释写明与单源的唯一差别（n==1 / n≤0）**在全部调用点的可达域外**（实参均为 ≥20 字面量或具名 rune 预算常量） |
| `gaea_pptx.go` `truncateRunesEllipsis` | 退化为一行委托 `strutil.TruncateRunesSuffix(s, n, "…")`（同语义：前 n rune + 后缀） |
| `gaea_ui_extra.go` `searchSkipDirs` | 改由 `wspath.Merge(wspath.CoreSkipDirs, wspath.SearchExtra)` 构造（`SearchExtra` 与 wssearch 逐项相同 ⇒ 合并结果与收敛前 8 项**逐项一致**，扫描范围零变化）；`:484/:539` 的 map 查询与 `.tmp` 前缀判断原样保留 |
| `gaea_route_suggestions.go:157-159` | **被本批新语义推翻的旧注释**（原写「落盘失败只告警不回滚」）→ 更正为「落盘失败即回滚并返回 error」，并标明 :171 建议记录落盘仍只告警的理由（改绑已生效、重算幂等，不谎报失败） |

另：`gaea_diagram.go` 的 `truncateStr` 是**按字节**截断（非 rune），与 X1-06 列出的 `truncateRunes`/`truncateRunesEllipsis` 不同语义，本批**按原样保留**并登记（与更正 8 的 office 两处同类）。

### 线 3 · 前端 P0 与静默失败（5 条已交付；`ChatComposer` 收口中）

**FE6-02（P0）麦克风静默降级 → 显式降级位**（`hooks/useVoiceChat.ts`）：新增 `degraded: 'mic-unavailable' | null`（接口/初值/`start`/`stop` 全贯通）；**彻底删除 `simTimerRef` 假音量**（含 4 处清理路径）——`getUserMedia` 失败只置 `degraded` + `volume=0` + `speaking=false`，绝不伪造「正在聆听」。`startCapture` 改为返回 `Promise<boolean>`，`start()` 按**返回值**分支决定是否启动本地识别（不再读 `stateRef.current`——该写法依赖 React 对单个待处理更新的 eager state 优化，队列里有别的更新时会失效）。
警示条落 **3 处生产消费面**：主壳命令条 `ModuleLauncher.tsx`（降级位优先于一切状态文案，并**摘掉 `is-listening` 呼吸动效**——该动效本身就是「正在聆听」的视觉语义）、`WelcomeScreen.tsx`、`VoiceChatOrb.tsx`（新增可选 prop `micUnavailable`，唯一真实调用方= WelcomeScreen，既有契约零变化）。
证据：`useVoiceChat.degraded.test.tsx` 4/4、`ModuleLauncher.test.tsx` 语音降级 2 例、`VoiceChatOrb.mic.test.tsx` 2/2；**反向 mutate（`micDegraded=false`）→ 用例变红**。
**收口（复审追补）**：全仓扫描（正则覆盖 `正在聆听|准备聆听|聆听中|请说话|voiceListening|listening ?`）发现**生产界面只剩 `components/chat/ChatComposer.tsx:113`** 仍在撒谎（输入框占位符「正在聆听…请说话」）——其余命中均为已修落点/测试/注释/三语字典。已交回线 3 补第 4 处，落地时发现这是**二阶谎言**：

| 面 | 修前 | 修后 |
|---|---|---|
| 占位符 `:117` | `voiceOn ? (transcript \|\| '正在聆听…请说话') : '输入消息…'` | 降级位优先 → `'麦克风不可用，请输入文字'` |
| 输入框 `:122` | `disabled={sending \|\| voiceOn}` | `disabled={sending \|\| (voiceOn && !voiceDegraded)}`——**降级时放行文字输入**（否则「请输入文字」写在一个打不了字的框上，是另一种撒谎） |
| 发送钮 `:136` | `disabled={!input.trim() \|\| voiceOn}` | 同上放行 |
| 语音钮 `:106-108` | `title/ariaLabel = '结束聆听'` | 降级时改「结束语音（麦克风不可用，未在采集音频）」——不再自称在聆听 |

证据：`ChatComposer.voiceDegraded.test.tsx` 4/4（降级 → 占位符「麦克风不可用，请输入文字」且 `not.toContain('正在聆听')`、`aria-label` 变「结束语音」；**对照 2 例**：未降级 + `voiceOn` 仍「正在聆听…请说话」且 `disabled=true`、未开语音仍「输入消息…」；**反向 mutate（占位符改回原式）→ 变红**）；`ChatPage.tsx:466` 传 `voiceDegraded={voice.degraded === 'mic-unavailable'}`。

**各处还顺带补了同一降级位的同源面**：`WelcomeScreen.tsx` 遥测行的 `VOICE <b>LISTEN</b>` 在降级时显示 `MIC OFF`（非降级逐字零变化）——这是该文件此前唯一被扫到的 `LISTEN` 面。

**第五次全仓扫描（复用同一正则，`frontend/src/**/*.{ts,tsx}`）结论：无第五处真实消费面。** 余下命中全部是「已受降级位前置保护」或注释/字典：

| 命中 | 性质 |
|---|---|
| `ChatComposer.tsx:122` / `VoiceChatOrb.tsx:232` / `ModuleLauncher.tsx:570,572` | **降级位前置保护**（三目链里 `micDegraded`/`micUnavailable` 短路在前，降级不可达） |
| `ChatComposer.tsx:30`、`WelcomeScreen.tsx:47`、`ModuleLauncher.tsx:562,581`、`VoiceChatOrb.tsx:39`、`useVoiceChat.ts:20,354,386,388,486` | 注释 |
| `locales/zh.ts:642` `shell.launcher.voiceListening` | 三语字典（合法） |

**线 3 欠账（真机）**：jsdom 已覆盖降级位与四处文案，但**真机拔麦走查未做**（需在有麦环境点一次）——按本仓惯例登记为真机欠账，不假装已验。

**FE4-12 附件落盘失败**（`gaea/hooks/useComposerAttachments.ts`）：三处空 `catch {}` 改 `toast` 可见上报（图片/文件/选择），`handlePickFiles` 仅在 `typeof app.PickFiles !== 'function'`（绑定不存在）时静默。证据 5/5。

**FE7-14 DCMA / 摘要 / diff 静默吞**：`DcmaView.tsx` 加 `thresholdFail` 态 + 提示条；`gschedSummary.ts` 加 `parseSchedSummaryDetail`（**保留原 `null` 契约**，既有单测不动）；`FilePreview.tsx` 解析失败把原因上屏并回落原文；`ScheduleDiffCard.tsx` 加 `beforeReadError` 区分「读失败」与「确无内容」。证据 DcmaView 5/5、DiffCard 9/9、gschedSummary 14/14、FilePreview 27/27。**遵守「不打断编辑」纪律**：只出页面内提示条，不弹 modal、不刷 toast。

**FE3-10 mock 冷路径订阅者泄漏**（`gaea/lib/bridge/events.ts`）：加 `cancelled` 标志，`then` 回调内先检查，cleanup 置位。证据 `events.coldpath.test.ts` 4/4（**mutate `if (cancelled)` → 红**）+ `events.test.ts` 4/4。

**任务 5 · 对接线 1 的 `Warnings` 契约**（`memoryhub/DigitalLifeLibrary.tsx`）：`warnings?: string[]`，非空时渲染「数字生命数据降级 N 项」并逐条列出；**未动既有 snake_case 契约层**。证据 4/4。

**线 3 验证**：`npx tsc -b` → **exit 0**；`npx eslint`（20 个改动文件）→ **0 error**；`npx vitest run`（19 文件）→ **165 例全绿**（含 ChatPage 15、FilePreview 27、ScheduleFileCard 12、Composer.queue 7、bridge 8、perf-guards 17、contrast-hex 4）。

### 线 4 · 截断族与 GenUI 上限单源收敛（任务 1、2 已交付；任务 3 进行中）

**契约文件** `internal/gaea/strutil/truncate.go`：三个名字即说明行为的原语（**不做万能函数**）——
| 原语 | 语义 | 边界 |
|---|---|---|
| `TruncateRunes(s, n)` | 前 n 个 rune，无后缀，总长 ≤ n | `n ≤ 0` → `""`（旧实现在 `n<0` 时切片越界 panic） |
| `TruncateRunesEllipsis(s, n)` | 前 n-1 rune + `…`，总长恰为 n | `n==1` → 首个 rune；`n ≤ 0` → `""` |
| `TruncateRunesSuffix(s, n, suffix)` | 前 n rune + suffix（表达 `r[:n]+"…"`） | 未超长优先返回原串，再判 `n ≤ 0` |

**对拍方法（比要求更好）**：临时工具 `snapharness` 里保留**旧语义参照实现**，`Inputs()`（12 个：空串/ASCII/恰好长度/中文/emoji/零宽字符/首尾空白/超长）× `Ns()`（54 个：-1/0/1/n±1/各站点字面量附近），`safeCall` 把旧实现的 **PANIC 也记进快照**（旧代码 `n<0` 越界，无输出可对拍）。仓外 `%TEMP%\gaea-audit-b11\{before,after}` 逐字节 diff。

**对拍结果**：
| 类别 | 数量 | 说明 |
|---|---|---|
| 逐字节全等 | **13 个文件** | bookimport、pins、hook.clipRunes、skill.clipRunes、memory×3、novelcontext×3、archive、crosslink、xlsxedit |
| 仅 `n=-1` 行差异（各 12 行） | **8 个文件** | characterstate、wssearch、costimport、analysis、`util.Truncate`、weixin、`whisper.truncateStr`、`whisper.truncateString`；**差异全部是旧实现 PANIC → 新实现给确定值**，属有意改进并单列 |
| GA2-10 有意修正 | **143 行** | `tool/builtin` 的 `truncate`（websearch 摘要）：旧输出含非法 UTF-8 达 **82 行**，新输出 `\xNN` = 0、U+FFFD = 0；空串/`n=0`/纯 ASCII 全部不变 |

**边界裁决（逐站点，不接受「边界值不会出现」式跳过）**：全域逐字节等价 3 处（`pins`（唯一动态实参 `maxTotalRunes-total` 可为负，旧实现 `n≤0→""` 与原语一致）、`hook.clipRunes`、`skill.clipRunes`——后两者保留本地守卫的薄包装）；其余站点以「字面量正实参 + 可达域 `reachable(n)`」证明，域外行为在测试里单列断言（例：`r[:n-1]+"…"` 语义只在 `n≥2` 可达，`n==1` 时旧给 `"…"`、原语给首 rune）。

**对拍结论已固化**进 `internal/gaea/strutil/truncate_test.go`：含 **9 个 `old*` 参照实现**（逐字复制收敛前代码）+ 全矩阵等价断言 + 各站点 `reachable(n)` 可达域声明。**临时脚手架已删净**：`Test-Path ...\snapharness` = False、`zz_audit_snapshot_test.go` 计数 = 0（17 个全删，删前逐个校验绝对路径均在仓内）；`git status --short` 无 `zz_`/snapharness 残留。

**主代理裁决**：①`tool/builtin` 的 `truncate` **保留名字与一行薄包装**（本地政策 `TrimSpace` 真实存在；为省一个定义去动归属外的 `knowledge_search.go`/`websearch_live_test.go` 不划算）——该薄包装的存在与原因在此登记；②`knowledge_search.go:254` 的 200 上限随同一 helper 由「字节」变「rune」，算**同一处 GA2-10 修正的连带受益方**（纯 ASCII 无变化，中文只能变好），不算意外漂移。

**线 4 验证（任务 1）**：`go build ./...` = exit 0（7 个同包断链修完，另含 `media_upload`/`media_download`/`dispatch_router`/`memory_contradiction`/`triple_extractor`/`reconstruct`/`foreshadow_prompt`）；`go vet ./internal/gaea/strutil/...` = 0；触及包测试全绿（strutil/util/whisper×3/bookimport/weixin/analysis/characterstate/pins/costimport/hook/memory/skill/wssearch/tool+builtin/novelcontext/archive/office×5）。

**任务 3 · GenUI 上限单源（GA2-05 / X1-07 / FE6-03 / FE6-04 / FE6-19）——选 (a)**
- **选 (a) 的理由**：GA2-05 的后果是**模型拿假绿灯**（Go 校验放行、前端截断）；删常量只会把权威推给前端、问题仍在。补齐后 Go 与渲染器**同强度**，6 个常量第一次真正被执行。
- **补齐的检查**：`text`/`callout.content` → `maxString`；`code.code` → `maxCode`；`table.rows/columns`；`select/radio.options`；`chart.data` 与 `series[].data` → `maxChartPoints`。**字符串口径用 UTF-16 码元**（`utf16.Encode`），与 JS `String.length`（即 `guard.ts` 的 `str(v,cap)` 截断点）逐值一致 ⇒ **emoji 上不会出现「后端放行、前端截断」**。
- **机器守卫（X1-07 的核心要求）**：Go `internal/gaea/genui/limits.go` 为**唯一真相源** → 生成 `frontend/src/genui/limits.ts`（`GENUI_GO_LIMITS` + `GENUI_GO_NODE_TYPES`）；`spec.ts` 由它构 `GENUI_LIMITS`/`GENUI_NODE_TYPES`（渲染器专属 9 项仍留在 spec.ts 一处）。`TestGenuiLimitsSync` **逐字节比对**（反向验证：手改 `limits.ts` 的 `maxChartPoints: 60→61` → FAIL 并打印 got/want，重新生成后 OK）；`TestEveryLimitIsEnforced`（10 个常量**各造一个刚超限规格**，任一未接校验即红）+ 边界值必须合法。命名统一为 `maxChartPoints`（旧 `maxChartPoint` 消失）。
- **FE6-04 裁决：删除**。`validateGenuiSpec` + `countGenuiNodes` + `requiredFields`（`guard.ts:690-826`，约 **137 行**）全仓零消费（含 `scripts/`、`internal/`）→ 删函数 + `index.ts` 再导出 + 专项测试。前端围栏链走 `parse.ts → repairGenuiSpec`，模型侧走 Go `genui_validate`，**不留第二张规则表**。
- **FE6-19**：`genuiPanel.ts` 的 `200` → `GENUI_LIMITS.maxNodes`、`400` → `maxNodes*2`（注推导）；`interaction.ts` 的 `MAX_BLOCKS=200` 一并同源。

**保留未合并（线 4 如实上报，非失败）**：`archive/archive.go:236`（字节、无后缀，调用点 :58 是 2000 字节预算）、`office/crosslink.go:188` 与 `office/xlsxedit.go:772`（字节 + `…`，截子进程 stderr，见更正 8）、`memory/doc.go:129`（32KB 字节预算 + HTML 注释）、`memory/activation.go:119`（截断后对前缀 `TrimSpace` 再拼标记、总长 ≤ cap，与 `TruncateRunesSuffix` 不等价）。**未触碰**：`agent/{agent_helpers,compact_summary,session/log}.go`、`agent/textutils`、`toolguard`、`bash.go`、`control/refs.go`（按行/字节预算或非归属）。

**线 4 最终验证（真实输出）**：`go build ./...` = 0；`go vet ./...` = 0；**35 个包** `go test -count=1` 全 ok；`gofmt -l`（strutil/wspath/genui）空；新增文件无 BOM；`snapharness` = False、`zz_audit_snapshot_test.go` = 0；`tsc -b` = 0；`vitest src/genui` 4 文件 26 例 + `genuiPanel.test.ts` 5 文件 31 例全绿。

---

## 二、跨线裁决与缝合（主代理收线清单）

**已裁决**：
1. **GA6-09 当场降级**为「只登记迁移清单、不动签名」（切口纪律，见 §〇 更正 3）；绑定面裁决见 §五。
2. **GA6-09 绑定面定调**：保持绑定方法签名不变，`CostSearchPage` 加 `Error` 字段（见 §五）。
3. **线 4 的边界语义分叉**：`TruncateRunesEllipsis` 的 `n==1`/`n<=0` 在 `gaea_subagents.go` / `skill/index.go` / `hook/runner.go` / `weixin/clawbot.go` 四个站点上语义各不相同 → 要求逐站点「给不可达证明」或「留一行薄包装」，不接受「边界值不会出现」式跳过。

**收线执行结果**：

| # | 项 | 处置 | 状态 |
|---|---|---|---|
| 1 | app 侧截断收敛 | `gaea_subagents.go` 的 `truncateRunes`、`gaea_pptx.go` 的 `truncateRunesEllipsis` → 一行委托 `strutil`（n==1 / n≤0 的差异已在注释里声明**在全部调用点可达域之外**）；`gaea_diagram.go:244` 的 `truncateStr` 是**按字节**语义，**保留**并登记（与更正 8 同类） | ✅ 已完成 |
| 2 | app 侧跳过目录 | `gaea_ui_extra.go` 的 `searchSkipDirs` 改由 `wspath.Merge(wspath.CoreSkipDirs, wspath.SearchExtra)` 构造 ⇒ 与收敛前 8 项**逐项一致**；`:484/:539` 的 map 查询与 `.tmp` 前缀判断原样保留 | ✅ 已完成 |
| 3 | 知识库迁移状态接线 | `knowledge.Service.MigrationState()` 已导出，app 启动路径 / 面板提示尚未接 | ⏳ 留池（§三 余量 4） |
| 4 | 备份回滚部分失败的前端文案 | 签名不变、现会在部分失败时返回非 nil error | ⏳ 留池（§三 余量 5） |
| 5 | 删临时脚手架 + 固化用例 | `snapharness/` + 17 个 `zz_audit_snapshot_test.go` 已删净（主代理 `Test-Path` 复核 = False / 0）；对拍结论固化进 `truncate_test.go`（9 个 `old*` 参照实现 + 可达域 + 边界单列） | ✅ 已完成 |
| 6 | 守卫基线刷新 | `check-primitives`（截断 **41→36**、原子写 **13→8**）与 `check-test-ctors` 已 `--write-baseline`，两份 `--strict` 均无新增 | ✅ 已完成 |
| 7 | **收线新发现**：HEAD 本身不是 lint 干净 | `schedule/index.go:93`（errcheck）、`docmd/pdf.go:603`（SA4006 死赋值）——所在文件本批一字未改、与 HEAD 一致 | ✅ 已修（更正 9） |
| 8 | **收线新发现**：批次一 AP4-08 只修一半 | `internal/schedule/index.go` 新增 `scheduleIndexSaveMu` 串行化进程内写入；flaky 用例独立复跑 **6/6 全过** | ✅ 已修（更正 10） |

---

## 三、本批结论与余量

### 3.1 交付与验收

| 项 | 结果 |
|---|---|
| **P0 收官** | AP9-01（数字生命全吞错）、AP9-02（Herdsman 孤儿进程）、FE6-02（麦克风静默降级）**三条全部闭合**，且各带「改坏能红」反向证据 |
| P0 全局对账 | 25 行 P0 中 **19 已闭合 / 3 本批闭合 / 2 待拍板（AP4-01、X1-03）/ 1 已证伪撤销**——即**「一刀可落」的 P0 已清零**，余下两条是结构拍板项（见 §四） |
| 静态门禁 | `go build ./...` = 0；`go vet ./...` = 0；`gofmt -l` 空 |
| Go 测试 | 触达 **35+ 包**全绿；`go test ./internal/app/... -count=1` 主代理独立复跑 = `ok internal/app 105.613s` + `ok board 0.490s` |
| 前端 | `tsc -b` = 0（主代理独立复跑）；`eslint` 改动文件 0 error；`vitest` 改动面 20 文件 / 169 例全绿（主代理独立复跑其中 7 文件 39 例） |
| 守卫 | `check-docs` OK；`check-primitives --strict` OK；`check-test-ctors` 无新增；`check-contract-drift` 无新增漂移；`check-bindings-drift` OK@744 |
| **全量 CI** | **`scripts/ci.ps1` → `CI OK` / exit 0**：`go build` + `go vet` + `golangci-lint v2.14.0`（0 issues）+ Go 全量测试 + 前端 build + **vitest 422 文件 / 3641 例全绿** + E 系列回归守卫 + 仓库卫生守卫 + 版本漂移检查，全过 |

### 3.2 收敛账（可复跑，守卫脚本量化）

| 项 | 收敛前 | 收敛后 | 说明 |
|---|---|---|---|
| 截断原语定义 | 41 | **36** | 删 8 处旧副本、新增 3 处 `strutil` 单源；6 处保留为「薄包装 + 站点政策」 |
| 原子写旁路（internal/app） | 13 | **8** | 历史上批次已收敛 5 处，本轮刷新基线把假账抹平 |
| 前端旁路 / heredoc | 5 / 0 | 5 / 0 | 无变化 |
| 临时脚手架 | — | **0** | `snapharness/` + 17 个 `zz_audit_snapshot_test.go` 已删净并 `Test-Path` 复核 |

### 3.3 余量（如实登记，未硬凑）

1. **字节口径截断站点**：`office/crosslink.go:188`、`office/xlsxedit.go:772`（截子进程 stderr）、`app/gaea_diagram.go:244`、`archive/archive.go:236`、`memory/doc.go:129`、`memory/activation.go:119`——**语义不同或未授权改行为**，见 §五 更正 6/8。
2. **`estimateTokens` 四份**（`util` 中文感知 1.5/1.3 vs `contextview`/`trajectory`/`agent` 的字节 × 0.25）——**口径真不同**，合一会改变预算行为，属拍板项（消费者清单已核：`context/engine.go` ×6、`memory/memory.go`）。
3. **GA6-09 成本检索错误通道**——迁移清单已备（§五），绑定面裁决已定，待独立一刀。
4. **GA3-13 的 UI 接线**：`knowledge.Service.MigrationState()` 已导出，但 app 启动路径/面板提示尚未接——「失败可见」目前停在 API 层。
5. **GA4-11 的前端文案**：`GaeaDataBackupRollback` 现会在部分失败时返回非 nil error（签名不变），前端需确认按错误展示。
6. **真机欠账**：FE6-02 的真机拔麦走查（jsdom 已覆盖）；`-race` 本机不可用（无 gcc，`cgo: C compiler "gcc" not found`）→ **竞争检测门只在 Actions**，本批 AP9-02 的竞争靠「Wait 后读缓冲」设计 + E2E 覆盖正常清理路径兜底。
7. **`maxFenceBody` 口径**：Go（字节）与 TS（UTF-16）仍不同，本批只统一了数值（见更正 7）。
8. **AP8-16 残留**：三处 Save 都尝试后可能留下**部分落盘的磁盘混合态**（内存已回滚），单键事务需 config 层支持。
9. **AP9-01 致命/降级边界**：现取「8 张计数表全失败才致命」；若要求「`characters` 一失败即致命」改一处即可（线 1 已如实标注该取舍）。

### 3.4 方法论留存（可复用）

- **编译器当检查单**：跨包签名收敛时，`go build ./...` 的断链清单就是最完整的调用点清单——主代理把它抓给执行线，一轮修完（7 文件）。
- **反向对照测试**：`TestParentOnlyKillLeavesGrandchild`（证明旧写法留孤儿）、`TestCaptureAreaPartialOverlapAllowed`（证明修复没顺手关掉可行路径）、「改坏能红」四组实验——**证明「旧行为是错的」与「新行为没多收」同等重要**。
- **基线守卫必须同批刷新**：`check-test-ctors` 已有一处**入库即漏登记**的偏差（见 §四 守卫卫生），warn 档不刷基线 = 假新增攒到没人看。
- **对拍要能把差异归因到边界**：线 4 的对拍输出不是「有差异」，而是「8 个文件各 12 行差异**全部落在 `n=-1` 且旧实现是 PANIC**」——这种形态才可裁决。
- **「单一真相源」不是把并集当默认**：GA5-17 若按审计原文做会打挂 `.gaea/exports` 可检索的验收线（见更正 6）。

---

## 四、留池与待拍板

### 待拍板 A · AP4-01 三套并行编排合并不（P0·结构项）

现状（本轮读码实测，三套各自独立）：

| 编排 | 位置 | 身份 | 状态机 | 取消 | 进度 | 落盘 |
|---|---|---|---|---|---|---|
| DAG | `internal/app/gaea_dag.go` | `dag.RunView` 自有 run/node id | 节点级（pending/running/…） | `ga.dagCancels`（`sync.Map` runID→cancel，`:540/:542/:468`） | 节点态 | `dag` 包自有档 |
| 收件箱 | `internal/app/gaea_task_inbox.go` | `ti-` + 12hex | 自有 status 字符串（`taskInboxSave`/`SetStatus`） | 无独立取消 | 无 | 自有 JSON 文件（`taskInboxPath`，全量覆盖写） |
| 任务调度器 | `internal/gaea/tasks/tasks.go` | SQLite 自增 | `queued/running/stopping/succeeded/failed/cancelled` 六态 | `StatusStopping` + `OnForceKill` | `Progress.Report/Result/Output/ExitCode` | SQLite |

**选项**：
- **(a) 短期抽共用原语**（审计建议）：把「在途登记 + 事件通道 + 取消」抽成一份，三套入口保留但共用登记/取消/进度契约。范围 M，无用户可见行为变化，可作为 (b) 的第一步。
- **(b) 统一到一个编排内核**：DAG/收件箱退化为 `kind` + handler 与清单视图。范围 L/XL，跨 `internal/app` + `internal/gaea/dag` + `internal/gaea/tasks`，必须先锁定黄金测试。
- **(c) 维持现状**：三套语义继续各自演化，每个新横切能力写三遍。

### 待拍板 B · X1-03 五重 `*core` 嵌入裸构造 panic（P0·结构项）

现状：`internal/app/app.go:272-277` 为**五重**嵌入（`*core` / `*writingState` / `*mediaState` / `*whisperState` / `*officeState`），四个 state 各自再嵌 `*core`；唯一装配点是 `app.go:308`；全仓 `&App{}` 裸构造 **227 处**（`check-test-ctors.mjs` 基线 356 处，为过渡防线）。

**关键发现（决定选项可行性）**：`internal/app/bindings_*.go` 全部是**生成物**（头部 `// Code generated by scripts/gen_bindings; DO NOT EDIT.`）。因此「在绑定入口铺开 `assertAssembled()`」**不需要手改几百个方法**，改生成器 + 再生即可。

**选项**：
- **(a) 生成器注入 + 测试桩迁移**：`scripts/gen_bindings` 每个门面方法首行注入 `b.a.assertAssembled()`；代价是要把 227 处 `&App{}` 迁到统一测试构造器（否则测试全红）。范围 L，收益=启动期 fail-fast。
- **(b) 只做 nil-safe 接入层**：给四个 state 的嵌入改为显式访问器，nil 时返回「App 未装配」错误而非 nil 解引用 panic。范围 M，不破测试桩。
- **(c) 维持 `check-test-ctors.mjs` 守卫**（现状），重构另立版本。

### 守卫卫生 · 过渡防线自身已漂移（本轮实测）

收线前跑三份守卫脚本做基线，发现 `node scripts/check-test-ctors.mjs` 报 **1 处「新增裸构造」（未在基线内）**——而本批开工前工作树是**干净**的，即这是**已入库的偏差**：

| 项 | 实测 |
|---|---|
| 报出项 | `internal/app/board_manifest_test.go:242` `a := &App{}`（`TestMainBrainDefaultBranchDispatchable`，「缺 core,writingState,mediaState,whisperState,officeState」） |
| 是否真危险 | **否**——该测试是批次八（`afc39177`，AP9-03 复核回归锁）有意为之，测试内 `:254-255` 明写「裸 App 执行闭包会因 core 未装配 panic，这里只断言『有 handler』，闭包连通性由 ChatGeneral 自身用例覆盖」 |
| 真问题 | 加入该测试的批次**没有跑 `--write-baseline`**，于是守卫从此持续报 1 处假新增 |

`round-2-fixes.md` §三自述「基线按脚本自述工作流以 `--write-baseline` 刷新（+2 处）」——实测该刷新**未覆盖这条后加项**。这属审计自身反复强调的同族问题：**基线不刷的 warn 档守卫 = 静默失效的守卫**（warn 档永远 exit 0，假新增攒到没人看）。处置：收线时按「人工评审 → `--write-baseline`」刷新，并把「改守卫口径或新增裸构造必须同批刷基线」写进本记录。

## 五、审计原文纠错与遗留登记

### 更正 4：GA4-13 不是 `doc-drift`——同一处 GBK 误码命中了**生产代码字面量**（用户可见）

审计把 GA4-13 定为 `doc-drift`（注释可读性，P2）。本批修它时发现：**同一个文件里被误码的不只是注释，还有函数体里的返回值**：

```go
// internal/gaea/provider/provider.go —— 修前
func (p *Pricing) Symbol() string {
	if p == nil || p.Currency == "" {
		return "楼"          // ← 生产代码字面量，不是注释
	}
```
且 `Pricing` 的注释「is just a display symbol (default "楼")」与两处单元测试断言（`provider_test.go` `TestPricingSymbolDefault` / `TestPricingSymbolNil`）都把错误值**固化**了——测试锁住的是坏值，所以这个缺陷永远不会自己暴露。

**误码可证（不是猜测）**：本机实跑 .NET GBK(936) 编解码——

| 步骤 | 结果 |
|---|---|
| `UTF-8("¥")` 字节 | `C2 A5` |
| 以 GBK(936) 解码 `C2 A5` | `楼`（U+697C） |
| `GBK("楼")` 字节 | `C2 A5`（与 `UTF-8("¥")` 逐字节相同） |

即：原文 `¥` 的 UTF-8 字节被当作 GBK 解码一次 → `楼`。**单轮误码，原值必然为 `¥`（U+00A5）**。

**用户可见性**：`Pricing.Symbol()` 有两个生产消费方——`internal/gaea/agent/render/sink.go:279`（`cost = fmt.Sprintf(" · %s%.4f", p.Symbol(), p.Cost(u))`，回合成本渲染）与 `internal/gaea/config/render.go:91`（配置回写 TOML）。因此当 provider 的 `currency` 为空时，成本摘要显示的是「楼」而非「¥」。

**全仓同类残留扫描（本轮实跑，正则可复跑）**：只统计「正常简体中文里几乎不出现的连续误码字」，全仓结果——

| 位置 | 结论 |
|---|---|
| `internal/gaea/provider/provider.go` | **修前是唯一生产源码命中，修后已清零** |
| `docs/code-audit-2026-10-02/machine/findings_all.json`、`units/GA4.json` | 审计自己的证据摘录，**保留原样**（它就该引用坏值） |
| `frontend/src/data/templates/*.ts`、`vitest-report.json` | 误报（首次正则误收合法字「广」，如「广告/广角」；已收窄） |

**附带确认**：全仓 `楼` 的 117 处命中逐条看均为合法用法（办公楼 / 住宅楼 / 高楼 / 跳楼 / 阁楼 …），**没有任何夹具或前端把「楼」当货币符号写死**，故 `"¥"` 的还原是完整且安全的。这条定性应从 `doc-drift` 上调为「静默显示错误值」的实缺陷。

### 更正 5：本条线落地时实测出的 4 处审计原文与现场不符

| 审计原文 | 现场 | 处置 |
|---|---|---|
| GA4-13「8 行中文注释已成 GBK 乱码」 | 实为 **18 行**（9 行中文注释 + 8 行英文注释里的 `鈥?`→`—` + 1 处**用户可见错误消息字面量** `AuthError.Error()` 里的 `鈥?`→`—`），另含 3 处生产字面量「楼」= `¥` | 全数修完；定性上调（见更正 4） |
| GA3-13「`store.go:169` 索引重建失败被吞」 | `169` 是 `fileBackend.Index()` **读 INDEX.md**；真正的 `rebuildIndex`（`store.go:203-228`）错误本来就上抛。现场是「索引**读取**失败被降级成文案」 | 按现场修（读失败 → `slog.Warn`） |
| GA4-14「`TCCAStats` 注释错位」 | `controller.go:869-870` 就在 `TCCAStats` 上方，**位置正确**（误报）；`Dream` 那份是**孤儿副本**（`dream.go:308-309` 已有同份） | 误报项不动；孤儿副本按「删」而非「搬」处置 |
| IN3-15「虚拟桌面坐标越界致 `BitBlt` 失败」 | **不成立**：实测 BitBlt **返回非 0** 且产出全黑帧（探针 `1<<20` / `-(1<<20)` 均 `err=nil`、非黑像素 `0/256`） | 仅查 `r==0` 修不掉该症状 → 另加「与虚拟桌面零交集即报错」守卫；`r==0` 检查仍保留（覆盖 DC 丢失等真实失败） |

### 更正 6：GA5-17 的正确修法**必须参数化**——审计原文的「取唯一清单」会当场打挂既有验收线

审计原文（GA5-17 修法）：「在叶子包（如 `internal/gaea/wspath`）定义唯一 `SkipDirs` 与 `IsSkipped(name)`，六处引用；差异需求用参数化追加而非新列表」。落地时实测发现：**六份清单的并集不能当作站点默认**。

| 事实 | 证据 |
|---|---|
| 六份清单只有 4 项交集（`.git` / `node_modules` / `dist` / `build`），并集 23 项 | `wspath.SkipDirs` / `CoreSkipDirs` 注释与测试 |
| 并集含 `.gaea`，而 `.gaea/exports` **必须可被工位检索** | `internal/gaea/wssearch/wssearch_test.go:54` `TestSearchSkipsNoiseDirs`：建 `.gaea/sessions`、`.gaea/archive`、**`.gaea/exports/周报.md`**，断言前两者不得泄漏进结果、`foundExport` 必须为真，末尾断言「`.gaea/exports` 交付产物应可被索引」 |
| play/work 分区语义同样锁在测试里 | 同文件 `TestSearchSkipsPlayExports`：`.gaea/play/exports` 并入噪音、`.gaea/exports`（work 交付产物）仍可索引 |

**若按审计原文把并集当默认，`TestSearchSkipsNoiseDirs` 与 `TestSearchSkipsPlayExports` 会当场变红**——即「一条 P1 的修法本身会打挂一条既有验收线」，属审计修法不完整。

**实际落地的正解（本轮）**：
- `IsSkippedIn(name, extra)` = `CoreSkipDirs ∪ extra`，各站点传自己的历史增量（`IndexExtra` / `WatchExtra` / `SearchExtra` / `GrepExtra` …）⇒ **逐站点与收敛前清单逐项相同，不改变任何面板的扫描范围**；
- `IsSkipped(name)` = 并集口径，供**新**调用方使用（不含历史包袱）；
- `.gaea` 下只有 `sessions/archive/cache/play` 是噪音，由 `wssearch.isNoiseRel` 单独判定，`.gaea/exports` 保持可检索。

**主代理独立核实**：`go test ./internal/gaea/filewatch/... ./internal/gaea/fileindex/... ./internal/gaea/wssearch/... -count=1` → filewatch 2.856s ok / fileindex 1.115s ok / wssearch 1.201s ok，**生产侧收敛零回归**（红过的只有新测试文件自身的编译错，已在收口中）。

### 更正 7：本条线实测出的其余 4 处审计原文与现场不符

| 审计原文 | 现场 | 影响 |
|---|---|---|
| GA5-17 记 `filewatch.DefaultSkipDirs` **11 项** | 实测 **10 项** | 计数偏差 |
| GA5-17 把 `wssearch.skipDirs` 与 `app/gaea_ui_extra.go` 的 `searchSkipDirs` 列为**两处独立定义** | 两者**逐项相同**（8 项 = 交集 + `.cache/.codegraph/.tianxuan/.reasonix`） | 少算一处重复；也是收线时 app 侧改写的等价性依据 |
| X1-06 记截断族 **13 处** | 本批归属包内实测**截断族定义 45 处**（含字节/行预算类），其中**行为同构、可收敛 15 处** | 审计低估；也说明「45 处全收」是错的目标（大量是字节/行预算语义） |
| GA2-05 记「6 个常量零引用」 | **正确**；但 `maxFenceBody` 的 Go（字节）与 TS（UTF-16）**口径仍不同** | 本批只统一了数值，**未动口径**——如实登记为残留，不假装已统一 |

### 更正 8：GA2-10 同族但**未授权改行为**的两处（登记给下一批）

`internal/office/crosslink/crosslink.go:188` 与 `internal/office/xlsxedit/xlsxedit.go:772` 同样是**按字节**截断并追加 `…` 的实现，截的是**子进程 stderr**——与 GA2-10（websearch 中文摘要按字节切）同族，会在中文报错信息里切出半个 rune。本批**未动**（不在授权范围，且属跨包行为变更），登记为下一批候选。

### 更正 9：**HEAD（批次十）本身不是 lint 干净的**——上一批没跑全量门禁就提交

本批跑 `scripts/ci.ps1` 时，`golangci-lint`（pinned v2.14.0）报 7 条，其中 **2 条的所在文件本批一字未改、内容与 HEAD 完全一致**：

| 位置 | 问题 | 归属判定 |
|---|---|---|
| `internal/schedule/index.go:93` | `errcheck`：`tmp.Close()` 返回值未检查 | `git status` 空 + `git show HEAD:` 内容一致 ⇒ **HEAD 既有** |
| `internal/docmd/pdf.go:603` | `staticcheck SA4006`：`pos` 的这次赋值永不被读 | 同上 ⇒ **HEAD 既有**（批次十 IN3-10 重构 `stripNonTextStreams` 时留下的死赋值：`pos` 只在线 605 读，而那时必已被 `:593` 的 `pos = sc.pos` 重设） |

即：**审计报告自述的「`golangci-lint` 0 issues」在 HEAD 上已不成立**——批次九/十落地时只跑了定向测试，没跑全量 `ci.ps1`。本批**一并修掉**（2 处均为等价改写：`_ = tmp.Close()`；删除死赋值），否则本批无法做到门禁全绿。

### 更正 10：批次一的 AP4-08 只修了一半——Windows 同目标并发 rename 的竞态仍在

本批 lint 修复触及 `internal/schedule/index.go`，顺带跑该包测试，发现 **`TestSaveScheduleIndexConcurrentNoTmpClash` 稳定 flaky**（无其它负载下独立复跑 3 次：1 过 2 败；`-count=5` 也多次失败）：

```
并发保存失败: 替换索引文件失败：rename ...\schedule-index-*.tmp ...\index.json: Access is denied.
```

**根因**：批次一（`12aeaf6d`，AP4-08）引入该测试时只解决了「**临时文件名唯一**」（`os.CreateTemp` 替代固定 `index.json.tmp`），但 16 个 goroutine 仍在**并发覆盖式 rename 同一个目标**；`fileutil.RenameWithRetry` 只在 `os.ErrPermission` 上退避重试 4 次（5/10/20 ms），在 16 路争用下会**穷尽**。该包此前**没有任何互斥**（`grep sync.Mutex` 零命中）。

**根修（本批）**：`internal/schedule/index.go` 新增 `scheduleIndexSaveMu` 串行化进程内索引写入（索引是单文件资源，进程内串行是正确的语义），跨进程仍由 `RenameWithRetry` 兜底。验证：该用例**独立复跑 6/6 全过**，`go test ./internal/schedule/... -count=1` 全绿。

**这条的意义**：它是审计 P0#24 / X1-02「Windows rename/TempDir 竞态三度复发」的**第四次复发**，且复发点正是「上一批刚为它加的防线不完全」——印证 06 分册的主张：**没有机器守卫的收敛，下一版必回涨**。建议下一批把「Go 侧 flaky 分类器 / 新并发用例必须 `-count≥5` 通过」纳入门禁（本机 `-race` 不可用，见 §三 余量 6）。

### 独立一刀 · GA6-09 成本检索错误通道（本批只登记、未动签名）

**为何本批不动**：`cost.Store.Search/List` 无 error 通道，而调用点横跨 4 个包 20+ 处——改签名会同时横跨线 1 的 `internal/app` 与线 4 的 `internal/gaea/tool/builtin`（同 Go 包不可并行），违反切口纪律。故当场降级为「只出迁移清单」。

**迁移清单（一次性完成，勿分批改签名）**：

| 归属 | 调用点 |
|---|---|
| `internal/app`（9） | `gaea_cost_compose.go:83`、`gaea_cost_graph.go:36`、`gaea_cost_import_vision.go:650`、`gaea_cost_inquiry.go:60`、`gaea_cost_rerank.go:112`、`gaea_semantic_index.go:20`、`gaea_semantic_search.go:98`、`memory_hub.go:237`（`len()`）、`:408`（`range`） |
| `internal/gaea/tool/builtin`（5） | `cost_compose.go:69`、`cost_tools.go:105/146/364`、`cost_compose_test.go:66` |
| `internal/gaea/costimport`（1） | `costimport.go:294` |
| `internal/gaea/pricefeed`（1） | `pricefeed.go:312` |
| `internal/gaea/cost` 包内测试（25） | `cost_test.go` 16 处、`ranker_cache_test.go` 7 处、`bench_test.go` 2 处、`code_test.go:56` |
| `internal/app` 测试（4） | `gaea_cost_import_visibility_test.go:178/219/238/250` |
| 定义处 | `cost.go:386-388 / 392-508`（含 `417-419`、`435-437` 的 `rows.Err()` 处理） |

**主代理裁决 · 绑定面（拍板点）**：**保持绑定方法签名不变**，理由是「绑定面零变更」是本仓硬纪律，而改签名要连带 `gen_bindings` 再生噪声 + `bindingNames` + bridge + mock + `types` 四处手工清单。具体：
1. `cost.Store.Search/List` → `([]Summary, error)`（**内部 API**，非绑定面）；
2. 绑定返回结构体 `CostSearchPage` **加字段** `Error string`（`json:"error,omitempty"`，**加字段不改签名 ⇒ 零漂移**），前端分页面据此显示「读取失败，可重试」——这正好覆盖成本库主 UI 路径；
3. `GaeaCostList` / `GaeaCostSearch`（返回裸 `[]CostSummary`，无法携带 error）保持签名，失败时 `slog.Warn` + 走既有 notice 事件通道让用户可见；
4. 若后续仍想把绑定改成 error 返回（Wails 支持），那是一次独立的 API 契约刀，另立版本。
