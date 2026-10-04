# Go 审计 · voice/whisper/tts/asr

审计范围：`internal/voice/`、`internal/whisper/`（含 `whisper/db/`）、`internal/tts/`、`internal/asr/`、`internal/ocr/`
方法：只读审计；符号级引用计数（Go 感知的注释/字符串剥离 + `rg -w` 交叉复核）；Wails 字符串调用风险在 `frontend/src` 全量比对；平台/build-tag 检查。
排除：`clones/`、`node_modules/`、`frontend/`、`.tmp*`、`backups/`、`releases/`、`dist/`、`whisper_data/`、`internal/app/**`（他人负责，仅作"消费者存在性"证据引用）、`internal/ai/**`、`internal/gaea/**`。

> 重要口径说明（为什么本报告可信）：所有"死代码"结论都用「剥离注释 + 剥离字符串字面量 + 允许 `pkg.Symbol` 点号限定调用」的引用计数得出（`(?<![\w])Sym\b`），再逐条用 `rg -n -w` 复核。第一轮曾因把 `.` 也当作前置排除符而误杀 `repos.CountFactsInDB` 这类跨包限定调用，已全部推翻重做；`ClearStructuredData`（测试在用）、`LoadChatHistoryFromDB`（`app/whisper_handler.go:760` 在用）、`ResolveDesktopCapabilityEnhanced`（`desktop_test.go` 在用）等一批"看似死"的符号经复核**是活的**，未进入死代码清单。另外，函数体边界用花括号配对而非"行首 `}`"计算，避免单行函数吞掉后继函数（`SetModel`、`PatchLatestTurnL5` 正是因此才被正确识别）。

---

## 范围与文件统计

| 路径 | 文件数 | 生产 LOC | 测试 LOC | 合计 LOC | 体积 |
|---|---|---|---|---|---|
| `internal/voice/` | 17 | 1 705 | 2 192 | 3 897 | 139 KB |
| `internal/whisper/`（含 db） | 159 | 19 476 | 7 386 | 26 862 | 918 KB |
| ├ `internal/whisper/`（非 db） | 125 | 16 997 | 6 694 | 23 691 | 816 KB |
| └ `internal/whisper/db/`（含 repos） | 34 | 2 479 | 692 | 3 171 | 102 KB |
| `internal/tts/` | 13 | 1 206 | 1 299 | 2 505 | 86 KB |
| `internal/asr/` | 3 | 216 | 58 | 274 | 10 KB |
| `internal/ocr/` | 3 | 241 | 142 | 383 | 12 KB |
| **合计** | **195** | **22 844** | **11 077** | **33 921** | 1 265 KB |

### 死代码总量（已逐条 grep 复核）

| 指标 | 数值 |
|---|---|
| 零引用生产符号（函数/方法/类型/常量/变量） | **175 个**，分布 **55 个文件** |
| 可删代码行（仅符号本体） | **1 195 行** |
| 可删行（含符号上方连续文档注释） | **1 313 行** |
| 整文件零引用（5 个文件全部符号无外部引用） | 192 行 |
| 死子系统（仅被构造、方法零调用） | 3 个（`delivery_coordinator.go` 148 行、`agent_job_manager.go` 149 行、`confirm_service.go` 104 行）|
| D1 合计（保守：死符号 + 死文件 + 死子系统，去重后） | **≈1 750 行** |

### 任务假设的核对结果（诚实结论）

| 假设 | 结论 |
|---|---|
| `whisper/db/` 多个 CRUD 文件重复样板，可否大幅收敛 | **可以收敛约 400–440 行（17%）**，但**不建议上代码生成**（详见专项评估） |
| whisper 下有历史遗留的模型封装（被新实现取代） | **未发现**。`rg 'whisper\.cpp\|ggml\|onnx\|sherpa\|LoadModel\|ModelPath' internal/whisper` → 0 命中；`internal/whisper` 是"轻语"AI 陪伴引擎，不含语音识别模型封装 |
| voice 里重复的音频格式转换/重采样/静音检测 | **未发现重复**。`wav.go` 的 `wavHeaderAt/wrapPCMAsWAV*` 已收敛为单一实现；RMS/VAD 只有 `voice_manager.go` 一份 |
| 注释掉的代码块 / 遗留实现 | **极少**。全范围仅 **1 行**真注释代码（`confirm_service.go:70`）+ 3 处重复注释行（`params.go` 段标题×2、`fts.go` 说明×1、`herdsman_asr.go:160` 文档×1） |
| tts/asr 与 whisper 的重复封装 | 无重叠；tts/asr/ocr 是独立 Herdsman seam，与 whisper 无同构代码 |

---

## Top 20 发现（按可删行数 × 安全度排序）

1. **[P0]** `internal/whisper/offline_thought.go:12` `GenerateOfflineThoughts` — D1 — LOC≈77 — confidence:**high**
   - 证据：`rg -n -w GenerateOfflineThoughts` → 3 处，全部在本文件（第 9 行标题注释、第 11 行文档注释、第 12 行定义）；**0 处调用**。文件内另一个函数 `OfflineThoughtsToHint` 是活的（`orchestrator.go:360` 调用），说明 `state.OfflineThoughts` 由别处产生，本函数是被淘汰的旧入口。
   - 建议：删除函数本体（77 行）。`containsStrInSlice`（第 108 行）只被本函数使用，一并删除（+8 行）。

2. **[P0]** `internal/whisper/agent_job_manager.go` 整文件 — D1+D8 — LOC≈149 — confidence:**high**
   - 证据：`NewAgentJobManager`/`StartJob`/`CancelJob`/`CompleteJob`/`IsJobRunning`/`GetJobState`/`CleanupSession` 全部 `rg -w` → 2 处（注释+定义）。唯一外部触点 `agent_loop_runner.go:57-58` 在 `r.JobManager != nil` 下调用 `UpdateJobPhase`，而 `rg 'JobManager:|AgentJobManager\{' internal` → 仅 `agent_job_manager.go:36` 自身，**该字段从未被赋值**，即 `UpdateJobPhase` 分支恒不执行。
   - 建议：整文件删除。`AgentJobState`/`AgentJob` 与 `internal/core/types.go:25`、`internal/core/jobs.go:9` **字段完全同名同类型**（含 json tag），是 `internal/core` 平迁后的残留副本。

3. **[P0]** `internal/whisper/delivery_coordinator.go` 整文件 — D1+D8 — LOC≈148 — confidence:**high**
   - 证据：8 个方法/函数 `rg -w` → 2 处（注释+定义）；构造点 `orchestrator.go:121` `DeliveryCoord: NewDeliveryCoordinator()`，**之后无任何 `DeliveryCoord.` 方法调用**（`rg 'DeliveryCoord\.' internal` → 0）。`FlushAll` 命中的是 `internal/gaea/agent/render/batcher.go:60` 的同名方法，与本文件无关。
   - 建议：整文件删除 + 删掉 `Orchestrator.DeliveryCoord` 字段（`orchestrator.go:57`）。`AgentTaskResult` 与 `internal/core/types.go:34` 字段完全一致。

4. **[P0]** `internal/whisper/confirm_service.go:79,96` `ResolveConfirm` / `CancelAllConfirms` — D1 — LOC≈26 — confidence:**high**
   - 证据：两者 `rg -w` → 2 处（注释+定义），0 调用。构造点 `orchestrator.go:120`。`RequestConfirm` 的调用点 `rg -w RequestConfirm` 命中的是 `desktop_router.go:30` 的**同名字段**（`RequestConfirm func(...) bool`），不是本方法。
   - 建议：**删除前需人工确认**——删掉整个 `ConfirmService` 意味着"桌面 Agent 操作确认"这条链路彻底不存在（现状是：即使被调用也永远等 120s 超时，因为没有任何代码能 resolve）。建议按"死子系统"处理并登记为功能缺口，而不是静默删除。

5. **[P0]** `internal/whisper/adult_mode.go` 8 个符号 — D1 — LOC≈98 — confidence:**high**
   - 证据：`IsHardStop`、`IsAdultRejection`、`ResolveAdultMemoryPrivacyLevel`、`ComputeProactiveScore`(42L)、`GetAftercareEmotion`、`ShouldTriggerNegativeLock`、`AdultTemperatureOffset`、`ContextBleedDivider` 全部 `rg -w` → 2 处（注释+定义）。
   - 建议：逐个删除。注意 `blockedEmotionLabels`/`hardStopWords` 等表若只被这些死函数使用，需连带清理（先删函数再跑编译验证）。

6. **[P0]** `internal/whisper/agent_job_manager.go` 之外的第二批整文件死代码：`ackem_birthday.go`(11行)、`plan_document_prompt.go`(31行)、`openforu_plan_prompt.go`(60行)、`openforu_codegen_prompt.go`(51行)、`openforu_evolve_prompt.go`(39行) — D1 — LOC≈192 — confidence:**high**
   - 证据：5 个文件的**全部**符号 `rg -w` → 1–2 处（注释+定义），无任何跨文件引用；其中 `GenerateOfflineThoughts` 同级的 prompt 常量（如 `PlanDocumentInstructions`）在 `internal/app`、`internal/gaea` 也 0 命中。
   - 建议：整文件删除。若担心是"产品未来的计划书 Agent 预设"，先移入 `docs/` 再删。

7. **[P0]** `internal/whisper/db/repos/` 死 CRUD 15 个函数 — D1 — LOC≈209 — confidence:**high**
   - 证据（`rg -w` 全部 2 处＝注释+定义，0 调用）：`CountFactsInDB`(12)、`UpdateFactInDB`(17)、`DeleteFactFromDB`(14)、`CountEpisodesInDB`(12)、`DeleteAllEpisodesFromDB`(14)、`CountTriplesInDB`(12)、`DeleteAllTriplesFromDB`(7)、`CountTracesInDB`(12)、`LoadTurnTracesFromDB`(29)、`DeleteChatHistoryFromDB`(14)、`DeleteCompanionStateFromDB`(14)、`DeleteEpisodeFTS`(12)、`PersistMemoryGraphToDB`(13)、`RestoreMemoryGraphFromDB`(11)、`CloseAllDatabases`(16)。
   - 建议：批量删除。`turn_traces.go` 的 `AppendTurnTraceToDB`/`LoadTurnTracesFromDBSession` **是活的**（`app/whisper_handler.go:253/365`、`app/wx_agent.go:558`），不要误删整文件。

8. **[P0]** `internal/whisper/procedural_habits.go:32,46,72,80` `AppendHabitLine`/`ReadHabitLines`/`IsEstablishedHabit`/`ListEstablishedHabits` — D1 — LOC≈59 — confidence:**high**
   - 证据：4 者 `rg -w` → 2 处（注释+定义）。`ProceduralHabitStore` 只在 `orchestrator.go:47` 声明字段，无方法调用。
   - 建议：整个"程序化习惯"存储（含类型）可删；若保留则至少删这 4 个方法。

9. **[P0]** `internal/whisper/memory_habit.go:130,150` `UpgradeToLongTerm`/`DecayAndCleanup` — D1 — LOC≈52 — confidence:**high**
   - 证据：`rg -w` → 2 处（注释+定义）。`HabitsStore` 其它方法（`Upsert`）是活的（`orchestrator.go:196`），只删这两个。
   - 建议：删除。注意 `DecayAndCleanup`(35L) 是"长时习惯衰减"的唯一定时清理入口，删除等于承认该策略未接线——建议在 commit message 记录。

10. **[P0]** `internal/whisper/user_fact_guard.go:54,116` `IsValidExtractedUserName`(20L)/`FilterExtractedUserMemoryFacts`(32L) — D1 — LOC≈52 — confidence:**high**
    - 证据：`rg -w` → 2 处（注释+定义）。文件内其余 guard 函数（被 `memory_ingest.go` 调用）是活的。
    - 建议：删除这两个；若用户事实守卫确实是安全必需项，则应改为接线而不是删除（需人工确认）。

11. **[P0]** `internal/whisper/tracer.go:20,33,39,41` `TraceLatest`/`TraceRing`/`TraceCount`/`PatchLatestTurnL5` — D1 — LOC≈28 — confidence:**high**
    - 证据：4 者 `rg -w` → 1–2 处（注释+定义）。`LogTurn` 是活的（`orchestrator.go:471`），但**唯一写者无任何读者**：环形缓冲 `traceRing`（上限 100）只写不读，属于"写-only 死数据"。
    - 建议：删除 4 个读者函数；`LogTurn` + `traceRing` 若无其它用途（如调试导出），可一并删除并移除 `orchestrator.go:462/502` 的 `TurnTrace` 构造。

12. **[P0]** `internal/whisper/triple_extractor.go:100` `ExtractTriples` — D1 — LOC≈39 — confidence:**high**
    - 证据：`rg -w ExtractTriples` → 2 处（注释+定义）。文件内 `extractStructuredTriples`(84L) 是活的（被 `memory_ingest` 调用），`ExtractTriples` 是被取代的旧入口。
    - 建议：删除。同时确认 `TripleRow` 类型是否只服务本函数。

13. **[P0]** `internal/whisper/personality.go:452` `BuildPersonalitySection` — D1 — LOC≈35 — confidence:**high**
    - 证据：`rg -w BuildPersonalitySection` → 3 处，全在本文件（第 450、451 行**重复文档注释** + 第 452 行定义），0 调用。
    - 建议：删除函数 + 顺手删掉重复注释行（第 451 行）。personality.go 其余 33.9 KB 是小函数 + 预设表，无 god function。

14. **[P0]** `internal/whisper/memory_habit.go` 之外的 `profiling_user.go:70,96` `InferArchetype`(23L)/`ArchetypeToResponseHint`(17L) — D1 — LOC≈40 — confidence:**high**
    - 证据：`rg -w` → 2 处（注释+定义）。
    - 建议：删除；若 `UserArchetype`/`UserProfile` 仅服务这两者，一并清理。

15. **[P0]** `internal/whisper/temporal_date.go:102,126,141` `DetectFastSpecialDateType`(22L)/`DetectSeason`(14L)/`IsSpecialSeason`(4L) — D1 — LOC≈40 — confidence:**high**
    - 证据：`rg -w` → 2 处（注释+定义）。同文件 `DetectSpecialDatesV2`(91L)、`DetectSpecialDates`(67L) 是活的。
    - 建议：删除 3 个旧入口。

16. **[P0]** `internal/whisper/desktop_confirm_bypass.go:29,43,65,75` 4 个 AutoApprove 存取函数 — D1 — LOC≈39 — confidence:**high**
    - 证据：`rg -w` → 2 处（注释+定义）；同文件 `hasDesktopAgentSessionAutoApprove` 等被这 4 个死函数调用，删除后需连带清理。
    - 建议：删除 4 函数 + 其独占的上下文键与辅助函数。**注意**：这批是"确认绕过"开关，若前端通过 Wails 字符串名触发，须先在 `frontend/src` 搜 `AutoApprove`（本次比对：0 命中）。

17. **[P0]** `internal/whisper/temporal_bridge.go:46` `ProduceTemporalSignal` — D1 — LOC≈30 — confidence:**high**
    - 证据：`rg -w ProduceTemporalSignal` → 2 处（注释+定义）。
    - 建议：删除；若 `TemporalSignal` 类型仅服务本函数，一并删。

18. **[P0]** `internal/whisper/context_time.go:20,63,68,73` `FormatLocalTime`/`FormatTimeContextBlockNow`/`TimeOfDayStringNow`/`UserAsksLocalClock` — D1 — LOC≈29 — confidence:**high**
    - 证据：4 者 `rg -w` → 2 处（注释+定义）。`weekdayZH` 变量是活的。
    - 建议：删除 4 函数；`TimeOfDayStringNow`/`FormatTimeContextBlockNow` 是 `context_time.go` 的"Now"变体，已被调用方直接传参版本取代。

19. **[P0]** `internal/whisper/turn_plan_prompt.go:51,219,226,231` `UserTaskFrame`(type)/`BuildFormatHintFromDelivery`/`IsDeliveryWeakSignal`/`DefaultTurnPlan` — D1 — LOC≈24 — confidence:**high**
    - 证据：`rg -w` → 2 处（注释+定义）；同文件 `TurnPlan` 主流程是活的。
    - 建议：删除。

20. **[P0]** `internal/whisper/db/repos/fts.go:154` + `internal/whisper/db/repos/episodes.go:69` — D1 + **D2** — LOC≈24 — confidence:**high**
    - 证据 1：`DeleteEpisodeFTS` `rg -w` → 2 处（注释+定义）。
    - 证据 2（D2 冗余）：`RebuildEpisodesFTS` 在 `fts.go:69` 事务**外**先 `DELETE FROM episodes_fts`，随后 `fts.go:98` 事务**内**又 DELETE 同一表一次；`RebuildFactsFTS` 没有这个外层 DELETE（不一致）。外层删除是纯冗余写。
    - 建议：删 `DeleteEpisodeFTS`；删 `fts.go:69-71` 的外层 DELETE，与 `RebuildFactsFTS` 对齐。

---

## 全量发现表

> LOC 为"符号本体行数"（含其上方连续文档注释时以 `~` 标注）。置信度口径：**high**＝`rg -w` 唯一命中为定义处/注释；**medium**＝结论正确但删除涉及约定或需连带清理；**low**＝反射/序列化/外部契约风险，仅供参考。

| 文件:行 | 符号 | 类型 | LOC | 置信度 | 证据 | 建议 |
|---|---|---|---|---|---|---|
| `internal/whisper/offline_thought.go:12` | `GenerateOfflineThoughts` | D1 | 77 | high | `rg -w` 3 处，全在本文件（2 注释+定义），0 调用 | 删除；连带 `containsStrInSlice` |
| `internal/whisper/agent_job_manager.go` 全文件 | `AgentJobManager` 等 7 符号 | D1+D8 | 149 | high | 7 符号各 2 处；`JobManager:` 字段全仓无赋值 | 整文件删；与 `internal/core/jobs.go` 重复 |
| `internal/whisper/delivery_coordinator.go` 全文件 | `DeliveryCoordinator` 等 8 符号 | D1+D8 | 148 | high | 8 符号各 2 处；`DeliveryCoord.` 0 方法调用 | 整文件删 + 删 `Orchestrator.DeliveryCoord` |
| `internal/whisper/confirm_service.go:79` | `ResolveConfirm` | D1 | 16 | high | 2 处（注释+定义），0 调用 | 死子系统，删前人工确认功能缺口 |
| `internal/whisper/confirm_service.go:96` | `CancelAllConfirms` | D1 | 10 | high | 2 处（注释+定义） | 同上 |
| `internal/whisper/confirm_service.go:16` | `ConfirmAllowed` | D1 | 1 | medium | 1 处；`denied/timeout` 变体仍被使用 | 保留（枚举完整性）或改字面量 |
| `internal/whisper/confirm_service.go:70` | `// broadcastFn(...)` | D3 | 1 | high | 全范围唯一一行注释掉的代码 | 删除注释；同时确认 120s 超时行为是否可接受 |
| `internal/whisper/adult_mode.go:50` | `IsHardStop` | D1 | 10 | high | 2 处（注释+定义） | 删除 |
| `internal/whisper/adult_mode.go:61` | `IsAdultRejection` | D1 | 10 | high | 2 处 | 删除 |
| `internal/whisper/adult_mode.go:83` | `ResolveAdultMemoryPrivacyLevel` | D1 | 17 | high | 2 处 | 删除 |
| `internal/whisper/adult_mode.go:134` | `ComputeProactiveScore` | D1 | 43 | high | 2 处 | 删除（最大单体死函数之一） |
| `internal/whisper/adult_mode.go:276` | `GetAftercareEmotion` | D1 | 4 | high | 2 处 | 删除 |
| `internal/whisper/adult_mode.go:283` | `ShouldTriggerNegativeLock` | D1 | 10 | high | 2 处 | 删除 |
| `internal/whisper/adult_mode.go:22` | `AdultTemperatureOffset` | D1 | 2 | high | 2 处 | 删除 |
| `internal/whisper/adult_mode.go:294` | `ContextBleedDivider` | D1 | 2 | high | 2 处 | 删除 |
| `internal/whisper/age_computer.go:63` | `BuildAgeLine` | D1 | 25 | high | 2 处 | 删除 |
| `internal/whisper/age_computer.go:36` | `ComputeCurrentAge` 参数 `isEstimate` | D7 | 1 | high | 函数体内 `isEstimate` 仅出现 1 次（形参处） | 删参数（调用点 `:76` 同步改） |
| `internal/whisper/agent_loop.go:43` | `AgentLoop` (type) | D1 | 2 | high | `rg -w` 4 处全为注释；类型 0 使用 | 删除类型 |
| `internal/whisper/agent_task_plan.go:13-17` | `PhasePlanning/Executing/Verifying/Delivering/Completed` | D1 | 5 | medium | 各 1 处；字面量 `"planning"` 等在他处使用 | 保留枚举，或统一改用常量 |
| `internal/whisper/canon.go:64` | `CanonMandatoryTemporalMarker` | D1 | 2 | high | `rg -w` 1 处 | 删除 |
| `internal/whisper/companion_proactive.go:15` | `ProactiveHabitNudge` | D1 | 1 | medium | 1 处；字面量 `"habit_nudge"` 类值他处出现 | 保留或改字面量 |
| `internal/whisper/context_time.go:20,63,68,73` | `FormatLocalTime`/`FormatTimeContextBlockNow`/`TimeOfDayStringNow`/`UserAsksLocalClock` | D1 | 29 | high | 4 者各 2 处 | 删除 4 函数 |
| `internal/whisper/desire.go:258` | `SettleDesiresForKnowledgeTopic` | D1 | 21 | high | 2 处 | 删除 |
| `internal/whisper/desktop_confirm_bypass.go:29,43,65,75` | `Set/ClearDesktopAgentSessionAutoApprove`、`Set/ClearTaskPlanDeleteAutoApprove` | D1 | 39 | high | 4 者各 2 处；frontend 搜 `AutoApprove` 0 命中 | 删除 + 连带清理辅助函数与上下文键 |
| `internal/whisper/fact_embedding_cache.go:17,22` | `NewFactEmbeddingCache`/`NeedsRebuild` | D1 | 8 | high | 各 2 处 | 删除；若缓存类型仅服务这两者一并删 |
| `internal/whisper/interpreter.go:515` | `DetectMemoryIntent` | D1 | 20 | high | 2 处 | 删除 |
| `internal/whisper/llm_client.go:51,78` | `ConsolidationResult`/`ContradictionResult` (type) | D1 | 4 | medium | 各 2 处；字段含 json tag（反序列化风险） | 若确无 `json.Unmarshal` 目标则可删；否则保留 |
| `internal/whisper/memory_audit.go:22` | `MemoryAuditReport` (type) | D1 | 2 | medium | 2 处；字段带 json tag | 同上（前端/导出可能按字段名消费） |
| `internal/whisper/memory_audit.go:7-10` | `AuditStatsOnly/Curated/SelfReport/FullDump` | D1 | 4 | medium | 各 1 处；字面量他处使用 | 保留枚举 |
| `internal/whisper/memory_fact.go:350` | `DedupByEmbedding` | D1+D7 | 4 | high | 2 处；注释自称"占位方法，Phase 3 实现"；参数 `embedding` 未使用 | 删除 |
| `internal/whisper/memory_habit.go:130` | `UpgradeToLongTerm` | D1 | 17 | high | 2 处 | 删除 |
| `internal/whisper/memory_habit.go:150` | `DecayAndCleanup` | D1 | 35 | high | 2 处 | 删除（记录"衰减策略未接线"缺口） |
| `internal/whisper/memory_taxonomy.go:19` | `Subcategories` (var) | D1 | 2 | high | 2 处 | 删除或与 `IsValidSubcategory` 一起删 |
| `internal/whisper/memory_taxonomy.go:67` | `IsValidSubcategory` | D1 | 5 | high | 2 处 | 删除 |
| `internal/whisper/personality.go:452` | `BuildPersonalitySection` | D1 | 35 | high | 3 处（含第 451 行重复注释） | 删除 + 删重复注释 |
| `internal/whisper/procedural_habits.go:32` | `AppendHabitLine` | D1 | 13 | high | 2 处 | 删除 |
| `internal/whisper/procedural_habits.go:46` | `ReadHabitLines` | D1 | 8 | high | 2 处 | 删除 |
| `internal/whisper/procedural_habits.go:72` | `IsEstablishedHabit` | D1 | 7 | high | 2 处 | 删除 |
| `internal/whisper/procedural_habits.go:80` | `ListEstablishedHabits` | D1 | 31 | high | 2 处 | 删除 |
| `internal/whisper/profiling_user.go:70` | `InferArchetype` | D1 | 23 | high | 2 处 | 删除 |
| `internal/whisper/profiling_user.go:96` | `ArchetypeToResponseHint` | D1 | 17 | high | 2 处 | 删除 |
| `internal/whisper/product_identity.go:19` | `BuildProductIdentityGuard` | D1 | 4 | high | 2 处；同文件 `InjectProductGuard` 是活的 | 删除（保留 `InjectProductGuard`） |
| `internal/whisper/reunion.go:96` | `ApplyReunionShock` | D1 | 15 | high | 2 处 | 删除 |
| `internal/whisper/runtime_hints.go:65` | `FormatUserPresenceHint` | D1 | 4 | high | 2 处 | 删除 |
| `internal/whisper/strategy_gate.go:30,37` | `GetAffHistory`/`ResetAffHistory` | D1 | 10 | high | 各 2 处 | 删除 |
| `internal/whisper/task_plan_store.go:98` | `SetDataRoot` | D1 | 6 | high | `rg -w` 仅本文件 2 处 | 删除；同时确认"纯内存模式"是否仍可达 |
| `internal/whisper/task_plan_store.go:247` | `IsContinueTaskPlanIntent` | D1 | 7 | high | `rg -w` 1 处 | 删除 |
| `internal/whisper/task_plan_store.go:255` | `BuildContinueTaskPlanHint` | D1 | 18 | high | 2 处 | 删除 |
| `internal/whisper/temporal_bridge.go:46` | `ProduceTemporalSignal` | D1 | 30 | high | 2 处 | 删除 |
| `internal/whisper/temporal_date.go:102,126,141` | `DetectFastSpecialDateType`/`DetectSeason`/`IsSpecialSeason` | D1 | 40 | high | 各 2 处 | 删除 3 个旧入口 |
| `internal/whisper/tool_def.go:8,27` | `UseComputerToolName`/`UseComputerActions` | D1 | 4 | high | `rg -w` 1–2 处 | 删除 |
| `internal/whisper/tracer.go:20,33,39,41` | `TraceLatest`/`TraceRing`/`TraceCount`/`PatchLatestTurnL5` | D1 | 28 | high | 各 1–2 处；`LogTurn` 唯一写者（`orchestrator.go:471`）无读者 | 删除 4 读者；评估 `LogTurn`+`traceRing` 一并删 |
| `internal/whisper/triple_extractor.go:100` | `ExtractTriples` | D1 | 39 | high | 2 处；新实现 `extractStructuredTriples` 在用 | 删除 |
| `internal/whisper/turn_plan_prompt.go:51,219,226,231` | `UserTaskFrame`/`BuildFormatHintFromDelivery`/`IsDeliveryWeakSignal`/`DefaultTurnPlan` | D1 | 24 | high | 各 2 处 | 删除 |
| `internal/whisper/turn_plan_prompt.go:26,27,45,46,47` | `GoalExplain/GoalRecommend/RouteExtension*` | D1 | 5 | medium | 各 1 处；字面量他处出现 | 保留枚举 |
| `internal/whisper/types.go:163-167` | `EmergenceLateNightEmo` 等 5 个 | D1 | 5 | medium | 各 1 处；字面量 `"late_night"` 12 处 | 保留枚举或改字面量 |
| `internal/whisper/types.go:539` | `CompanionQuiet` | D1 | 1 | medium | 1 处 | 同上 |
| `internal/whisper/types.go:605-608` | `SceneMeeting/Presentation/Focus/Other` | D1 | 4 | medium | 各 1 处 | 同上 |
| `internal/whisper/types.go:612` | `ForegroundSnapshot` (type) | D1 | 2 | medium | 2 处；字段带 json tag | 确认无 `json.Unmarshal` 目标后删 |
| `internal/whisper/user_fact_guard.go:54` | `IsValidExtractedUserName` | D1 | 20 | high | 2 处 | 删除或改为接线（需确认安全语义） |
| `internal/whisper/user_fact_guard.go:116` | `FilterExtractedUserMemoryFacts` | D1 | 32 | high | 2 处 | 同上 |
| `internal/whisper/vector_search.go:198` | `SearchBySemantics` | D1 | 22 | high | 2 处；`vector_store.go` 的 `Search` 在用 | 删除 |
| `internal/whisper/memory_retrieve.go:21` | `ComputeMemoryEchoFacts` 参数 `aff` | D7 | 1 | high | 函数体内 `aff` 仅形参 1 次；调用点传 `currentAff` | 删参数（`orchestrator.go:944` 同步改） |
| `internal/whisper/offline_thought.go:12` | `GenerateOfflineThoughts` 参数 `l1`,`l2` | D7 | 2 | high | 两参数函数体内 0 使用 | 随函数删除 |
| `internal/whisper/post_chat_turn.go:236` | `consolidateNow` 参数 `ctx` | D7 | 1 | high | 函数体内 `ctx` 仅形参 1 次 | 删参数（`:92` 调用点同步改） |
| `internal/whisper/params.go:91,92,99,105,116,117,124,126,135,136,137,144,145,155,156,157,163,164,165,166,188,191,210,241,248,253` | 26 个常量（`EffectiveTrustL1Weight`、`SemanticSearchTopK`、`VectorSearchTopK`、`ReunionOfflineMinutes`…） | D1 | 26 | medium | 每个 `rg -w` 1–2 处（定义/注释）；同语义参数在消费者文件里有小写镜像（如 `memory_binding.go:33 effectiveTrustL1Weight`） | 纯数值无反射风险，可安全删；但文件定位是"对齐 ackem 参数表"，建议**保留并补注释说明未被消费**（见"不建议动"） |
| `internal/whisper/params.go:244-247,263-265` | 重复段标题注释（"欲望产生基础概率"×2、"记忆自编辑参数"×2） | D3 | 3 | high | 同一标题连续出现两次 | 删除重复注释行 |
| `internal/whisper/db/repos/memory_facts.go:139` | `CountFactsInDB` | D1 | 12 | high | 2 处 | 删除 |
| `internal/whisper/db/repos/memory_facts.go:235` | `UpdateFactInDB` | D1 | 17 | high | 2 处 | 删除 |
| `internal/whisper/db/repos/memory_facts.go:253` | `DeleteFactFromDB` | D1 | 14 | high | 2 处 | 删除 |
| `internal/whisper/db/repos/episodes.go:56` | `CountEpisodesInDB` | D1 | 12 | high | 2 处 | 删除 |
| `internal/whisper/db/repos/episodes.go:140` | `DeleteAllEpisodesFromDB` | D1 | 14 | high | 2 处 | 删除 |
| `internal/whisper/db/repos/knowledge_triples.go:54` | `CountTriplesInDB` | D1 | 12 | high | 2 处 | 删除 |
| `internal/whisper/db/repos/knowledge_triples.go:119` | `DeleteAllTriplesFromDB` | D1 | 7 | high | 2 处 | 删除 |
| `internal/whisper/db/repos/turn_traces.go:38` | `LoadTurnTracesFromDB` | D1 | 29 | high | 2 处；同文件另两函数是活的 | 删除该函数，勿删文件 |
| `internal/whisper/db/repos/turn_traces.go:104` | `CountTracesInDB` | D1 | 12 | high | 2 处 | 删除 |
| `internal/whisper/db/repos/chat_history.go:71` | `DeleteChatHistoryFromDB` | D1 | 14 | high | 2 处；`Load/SaveChatHistory*` 是活的 | 删除 |
| `internal/whisper/db/repos/companion_state.go:105` | `DeleteCompanionStateFromDB` | D1 | 14 | high | 2 处 | 删除 |
| `internal/whisper/db/repos/fts.go:154` | `DeleteEpisodeFTS` | D1 | 12 | high | 2 处 | 删除 |
| `internal/whisper/db/repos/memory_graph.go:88` | `PersistMemoryGraphToDB` | D1 | 13 | high | 2 处 | 删除（常规路径走 `SaveCompanionStateToDB`） |
| `internal/whisper/db/repos/memory_graph.go:102` | `RestoreMemoryGraphFromDB` | D1 | 11 | high | 2 处 | 删除 |
| `internal/whisper/db/database.go:96` | `CloseAllDatabases` | D1 | 16 | high | 2 处 | 删除；确认进程退出不依赖它做 WAL checkpoint |
| `internal/whisper/db/database.go:198` | `ClearStructuredData` | D1 | 31 | high（生产零调用） | 唯一引用 `memory_graph_test.go:288`（测试） | 生产死代码；建议移到测试辅助或标注为测试专用 |
| `internal/whisper/db/repos/fts.go:69-71` | 事务外 `DELETE FROM episodes_fts` | D2 | 3 | high | 与 `:98` 事务内 DELETE 重复；`RebuildFactsFTS` 无此层 | 删除外层 DELETE |
| `internal/whisper/db/repos/fts.go:20-24` | 重复"先读后写"注释 | D3 | 3 | high | 同一语义注释重复两段 | 合并注释 |
| `internal/whisper/db/repos/habits.go:78-95` vs `:116-133` | `Upsert` / `SaveAll` 内联 16 行 UPSERT SQL 完全重复 | D4 | 17 | high | 两处 SQL 文本逐字相同（仅缩进差异） | 提为 `const upsertHabitSQL`，两处复用 |
| `internal/whisper/db/repos/knowledge_triples.go:127,135` | `insertTripleTx` / `insertTripleStmt` 内联 10 列 INSERT SQL 重复 | D4 | 5 | high | 两处 SQL 文本逐字相同 | 提为 `const insertTripleSQL` |
| `internal/whisper/db/repos/fts.go:115,128,141,154` | `Insert/DeleteFactFTS` + `Insert/DeleteEpisodeFTS` | D4 | 44 | medium | 形状 Jaccard=1.00（4 元组完全相同，仅表名/列名不同） | 收敛为 `ftsInsert/ftsDelete(dataRoot, stmt, args...)`，省 ≈22 行 |
| `internal/whisper/db/repos/fts.go:14,63` | `RebuildFactsFTS` / `RebuildEpisodesFTS` | D4 | 96 | medium | 形状 Jaccard=0.95（读→tx→清空→批量插 结构完全一致） | 收敛为 `rebuildFTS(...)`，省 ≈40 行 |
| `internal/whisper/db/repos/fts.go:169,211` | `SearchFactIDsFTS` / `SearchEpisodeIDsFTS` | D4 | 77 | medium | 形状 Jaccard=1.00（MATCH→空结果降级 LIKE 逻辑逐行同构） | 收敛为 `searchIDsFTS(...)`，省 ≈35 行 |
| `internal/whisper/db/repos/*.go` | `insert*Tx`/`insert*Stmt` 双包装（3 组 6 函数） | D5 | 24 | medium | 两组仅 `*sql.Tx` vs `*sql.DB` 之差 | 定义 `type execer interface{ Exec(string, ...any) (sql.Result, error) }`，各表保留 1 个插入函数，省 ≈9 行 |
| `internal/whisper/db/repos/memory_facts.go:49` / `episodes.go:29` / `knowledge_triples.go:29` | `toFact`/`toEpisode`/`toTriple` | D8 | 71 | medium | 行结构体与域类型字段逐一镜像（`Triple`↔`tripleRow`、`Episode`↔`episodeRow` 字段名集合完全相同） | 保留 row 结构（需要 `sql.Null*`），仅建议统一命名与字段顺序 |
| `internal/asr/herdsman_asr.go:63` | `(*HerdsmanASR).SetModel` | D1 | 4 | high | `rg -w SetModel` 2 处（注释+定义）；`.SetModel(` 全仓 0 命中；frontend 的 `SetModel` 是 Wails `GaeaSetModel` 无关符号 | 删除；若"模型中心动态切换 STT"仍要保留能力，应接到 `ASRProvider` 接口上 |
| `internal/asr/herdsman_asr.go:110` | `TranscribeRequest` (type) | D1 | 5 | high | `rg -w` 2 处 | 删除 |
| `internal/asr/herdsman_asr.go:160-161` | `EncodeBase64` 重复文档注释 | D3 | 1 | high | 同一行注释连续出现两次 | 删除重复行 |
| `internal/asr/provider.go:18` | `ASRProvider` 接口（3 方法，仅 herdsman 1 实现） | D5 | — | medium | 注册表+接口是刻意 seam（注释已声明）；`TranscribeBytes` 仅被测试 mock 实现 | **不动**：多引擎扩展点 |
| `internal/tts/util.go:9,10` | `TempDirName` / `OutputWAV` | D1 | 2 | high | `rg -w` 各 1 处；`sapi.go:23` 硬编码了同样的 `"gaea-tts"` 字面量 | 删除，或反哺 `sapi.go` 改用常量（更优） |
| `internal/tts/sapi.go:86` / `xai.go:128` | `SynthesizeWithParams` 参数 `p` | D7 | 2 | low | `p` 未使用，但注释明确声明"能力外忽略" | **保留**：接口签名一致性 |
| `internal/tts/sapi.go` 全文件 | 无 build tag 却用 `syscall.SysProcAttr{HideWindow}` | 平台风险 | — | high | 文件无 `//go:build`、无 `_windows.go` 后缀；全 scope 0 个 build-tag/平台后缀文件 | **不动**（Windows 桌面应用）；若将来跨平台编译需补 `_windows.go` |
| `internal/voice/voice_config.go:31-34` | `TTSEngineEdge/SAPI/HerdsmanQwen3/HerdsmanEdgeTTS` | D1 | 4 | medium | 各 1 处；字面量 `"edge-tts"` 24 处、`"local-sapi"` 1 处、`"qwen3-tts"` 94 处 | **保留**：前端配置契约；建议反向统一字面量 |
| `internal/voice/voice_config.go:43` | `ASRModelFunASR` | D1 | 1 | medium | 1 处；字面量 `"funasr-nano"` 5 处 | 保留（同上） |
| `internal/voice/voice_config.go:146` | `BitDepth` | D1 | 2 | high | 2 处；`ChunkBytes` 推导式里写死 `*2` | 删除，或让 `ChunkBytes` 用它（更优） |
| `internal/voice/voice_manager.go:974` | `SetPTTActive` | — | — | high（**活**） | `app/voice_handler.go:662` 调用 | 不动 |
| `internal/voice/voice_manager.go:626` | `handleReply` 119 行 | D6 | 119 | high | 单函数承担：回调快照+状态机+取消通道+LLM+韵律选择+TTS+自动续听 | 拆为 `snapshotCallbacks` / `runChat` / `playOrFallback` / `resumeListening` |
| `internal/voice/voice_manager.go:779` | `speak` 104 行 | D6 | 104 | high | 合成 goroutine + 播放循环两条流水线在同一函数内 | 拆 `synthesizeLoop` / `playbackLoop` |
| `internal/voice/voice_manager.go:912` | `CancelTTS` 承担 4 职责 | D6 | 36 | medium | 本地 TTS 打断 + realtime CancelResponse + ClearBuffer + 状态守卫 | 抽 `cancelRealtimeResponse` |
| `internal/whisper/orchestrator.go:129-486` | `PreLLMTurn` | D6 | **358** | high | 全 scope 最长函数；内含 9+ 阶段（红线→遗忘→情感→记忆→涌现→人格→TierA/B→计划） | 按阶段拆 5–7 个 `stageXxx(ctx)`，用 struct 传上下文 |
| `internal/whisper/orchestrator.go:867-1003` | `buildTierBBlock` | D6 | 137 | high | 单函数拼装全部 TierB 记忆块 | 按块类型拆分 |
| `internal/whisper/memory_graph.go:201-334` | `QuerySubgraph` | D6 | 134 | high | 图遍历 + 去重 + hops 处理 + 排序单函数 | 抽 `dedupNodes`/`dedupEdges` |
| `internal/whisper/emotion.go:151-263` | `EmotionStep` | D6 | 113 | high | 情绪衰减+锁定+噪声+涌现判定耦合 | 抽样条子步 |
| `internal/whisper/desktop_router.go:51-159` | `ExecuteUseComputer` | D6 | 109 | high | 参数解析+策略校验+确认+执行+审计 | 拆 `validate` / `confirm` / `execute` |
| `internal/whisper/time_depth_calculator.go:49-152` | `ComputeTimeDepth` | D6 | 104 | high | 时间深度多分支计算 | 抽子函数 |
| `internal/whisper/interpreter.go:300-401` | `InterpretInput` | D6 | 102 | high | 多语言关键词矩阵判定 | 表驱动化 |
| `internal/whisper/memory_consolidator.go:25-125` | `Consolidate` | D6 | 101 | high | 触发条件+LLM+落库+日志 | 拆 `shouldConsolidate` |
| `internal/whisper/desktop_executor.go:45-145` | `ExecuteDesktopAgentAction` | D6 | 101 | high | 动作分派 + 每次动作内联实现 | 动作表驱动 |
| `internal/whisper/paced_stream.go:211-308` | `pump` | D6 | 98 | high | 定时泵 + 边界条件多 | 拆状态机 |
| `internal/whisper/db/repos/memory_facts.go:302-372` | `factArgs` | D6 | 71 | high | 27 列参数 + AgeMeta 特殊分支 | 抽 `ageArgs(f)` |
| `internal/whisper/db/repos/memory_facts.go:49-115` | `toFact` | D6 | 67 | medium | 27 列映射 + JSON/时间/Age 分支 | 抽 `parseTimes`/`parseJSONCols` |

（上表 110+ 行，覆盖死符号 130 个明确条目 + 死子系统/D2/D3/D4/D5/D6/D7/D8 各类发现；另有 40 余个小型枚举常量因 LOC=1–2 未逐行列出，见"per-file 死代码分布"。）

### 死代码 per-file 分布（可删行，含上方文档注释）

（55 个文件，按可删行降序；数据由扫描器直接导出，未经手工转录）

```
  98  internal/whisper/adult_mode.go
  88  internal/whisper/agent_job_manager.go
  77  internal/whisper/offline_thought.go
  69  internal/whisper/delivery_coordinator.go
  59  internal/whisper/procedural_habits.go
  52  internal/whisper/memory_habit.go
  52  internal/whisper/user_fact_guard.go
  43  internal/whisper/db/repos/memory_facts.go
  41  internal/whisper/db/repos/turn_traces.go
  40  internal/whisper/profiling_user.go
  40  internal/whisper/temporal_date.go
  39  internal/whisper/desktop_confirm_bypass.go
  39  internal/whisper/triple_extractor.go
  35  internal/whisper/personality.go
  31  internal/whisper/task_plan_store.go
  30  internal/whisper/temporal_bridge.go
  29  internal/whisper/context_time.go
  28  internal/whisper/tracer.go
  27  internal/whisper/confirm_service.go
  26  internal/whisper/db/repos/episodes.go
  26  internal/whisper/params.go
  25  internal/whisper/age_computer.go
  24  internal/whisper/db/repos/memory_graph.go
  24  internal/whisper/turn_plan_prompt.go
  22  internal/whisper/vector_search.go
  21  internal/whisper/desire.go
  20  internal/whisper/interpreter.go
  19  internal/whisper/db/repos/knowledge_triples.go
  16  internal/whisper/db/database.go
  15  internal/whisper/reunion.go
  14  internal/whisper/db/repos/chat_history.go
  14  internal/whisper/db/repos/companion_state.go
  12  internal/whisper/db/repos/fts.go
  12  internal/whisper/types.go
  10  internal/whisper/strategy_gate.go
   8  internal/whisper/fact_embedding_cache.go
   8  internal/whisper/openforu_codegen_prompt.go
   7  internal/voice/voice_config.go
   7  internal/whisper/memory_taxonomy.go
   6  internal/asr/herdsman_asr.go
   6  internal/whisper/memory_audit.go
   6  internal/whisper/openforu_evolve_prompt.go
   6  internal/whisper/openforu_plan_prompt.go
   6  internal/whisper/plan_document_prompt.go
   5  internal/whisper/agent_task_plan.go
   4  internal/whisper/ackem_birthday.go
   4  internal/whisper/llm_client.go
   4  internal/whisper/memory_fact.go
   4  internal/whisper/product_identity.go
   4  internal/whisper/runtime_hints.go
   4  internal/whisper/tool_def.go
   2  internal/tts/util.go
   2  internal/whisper/agent_loop.go
   2  internal/whisper/canon.go
   1  internal/whisper/companion_proactive.go
```

（`internal/whisper/params.go` 一行 26 行 = 26 个 LOC=1 的常量；`voice_config.go` 7 行 = 6 个常量 + 注释行。）

---

## whisper/db 收敛专项评估

### 现状结构（实测）

| 组成 | 文件数 | 生产 LOC | 说明 |
|---|---|---|---|
| `database.go` + `paths.go` | 2 | 272 | 单例连接池、迁移驱动、旧库改名迁移 |
| `schema_v1.go` … `schema_v15.go` | 15 | 353 | 共 24 张表、9 个索引；无同名表/索引重复 |
| `repos/*.go` | 11 | 1 854 | 每表一套 CRUD + 行映射 |
| **生产合计** | **28** | **2 479** | 测试另有 692 行 |

### 能减多少

| 手段 | 可减 LOC | 风险 | 说明 |
|---|---|---|---|
| **① 删除零引用 CRUD** | **≈209** | **低** | 15 个函数已逐条 `rg -w` 验证零调用（见全量表）。这是唯一"纯删除"的部分 |
| ② 4 个 `Count*InDB` 收敛为 `countRows(dataRoot, table)` | 已被 ① 覆盖 | — | 4 个计数函数全部死；无需抽象，直接删 48 行 |
| ③ FTS 4 个单条插入/删除 → 2 个泛化函数 | ≈22 | 低 | 形状完全相同，仅 SQL 常量不同 |
| ④ `RebuildFactsFTS`/`RebuildEpisodesFTS` → `rebuildFTS(...)` | ≈40 | 中 | 形状 J=0.95；SQL 文本与"外层 DELETE"行为有细微差异（见 D2 发现），合并前先修不一致 |
| ⑤ `SearchFactIDsFTS`/`SearchEpisodeIDsFTS` → `searchIDsFTS(...)` | ≈35 | 中 | 形状 J=1.00；含 LIKE 降级兜底，需回归测试覆盖 |
| ⑥ `insert*Tx`/`insert*Stmt` 6 函数 → 3 函数 + `execer` 接口 | ≈9 | 低 | 仅 `*sql.Tx`/`*sql.DB` 差异 |
| ⑦ 内联重复 SQL 提常量（habits 17 行、triples 5 行） | ≈22 | 低 | 纯文本提取，零行为变化 |
| ⑧ `List`/`SaveAll`（associations/habits/temporal_anchors）泛化 | ≈20 | 中 | 三表空值语义不同（外键预检/指针字段/NULL 列），泛化易引入空值语义漂移 |
| ⑨ 行扫描循环泛化 `queryRows[T]` | ≈40 | 中高 | 需 4 组 `rows.Scan` 参数列表；收益/风险比一般 |
| **合计（①②③⑥⑦ 低风险）** | **≈262** | 低 | 纯删除 + 等价提取，不需要新测试即可验证 |
| **合计（含④⑤⑧⑨）** | **≈401–440** | 中 | 约占 db 生产 LOC 的 17% |

### 是否该上代码生成（sqlc / 模板生成）？—— **不建议**

1. **规模不够**：只有 11 个表、27 列上限的 1 张表；可提取的机械样板约 180 行（不含纯删除的部分）。引入代码生成要付出：新工具链依赖、生成物进版本库的评审成本、SQL 方言（modernc/sqlite + FTS5 + `PRAGMA defer_foreign_keys`）适配成本。
2. **差异不在样板而在语义**：真正难写的部分（`factArgs` 的 27 列 + AgeMeta 分支、`AssociationsRepo.SaveAll` 的外键预检 + 双向删除、`fts.go` 的中文 2-gram LIKE 降级兜底）都不是模板能覆盖的，生成器只能覆盖 `SELECT/INSERT/DELETE` 骨架。
3. **高风险区碰不得**：`ReplaceFactsInDB` 依赖 `PRAGMA defer_foreign_keys = ON` 让"全表 DELETE + 重插"通过外键检查；这种事务语义在生成器里最难表达，且一旦写错就是数据损坏。
4. **已有反例**：`InsertFact`/`InsertEpisode` 的"FTS 增量失败 → 降级全量重建"是手写兜底逻辑（注释记录了 O(N²) 的性能修复历史），生成器会把它抹掉。

**结论建议路径**：先做 **①（纯删除 209 行）**，再分批做 **③⑥⑦（低风险等价提取 ≈53 行）**，把 **④⑤⑧⑨** 留到有针对性单测覆盖后再做。若最终只想减行数而不动风险，**只做①即可拿到 209 行**。

---

## 屎山清单（结构性重构建议，含风险）

| 优先级 | 目标 | 现状 | 建议 | 风险 |
|---|---|---|---|---|
| P1 | `internal/whisper/orchestrator.go`（35.5 KB） | `PreLLMTurn` **358 行**；`buildTierBBlock` 137；`buildTierASnapshot` 95；`runAdultModeFSM` 81 | 按"红线→遗忘→情感→记忆→涌现→人格→TierA/B→计划"拆 6 个 `stageXxx(o, ctx, st)`；用 `turnCtx` struct 传状态，禁止跨 stage 副作用 | 中高：Turn 语义顺序敏感，必须逐 stage 做"输入输出等价"回归；`whisper_orchestrator_test.go`/`whisper_pipeline_test.go` 可作安全网 |
| P1 | `internal/voice/voice_manager.go`（1 342 行） | 单文件混装 3 套职责：本地 VAD/asr 拼接管线、realtime 事件泵、TTS 流式播放 | 拆为 `voice_manager.go`(状态机+入口)、`voice_vad.go`、`voice_pipeline.go`、`voice_realtime.go`、`voice_tts.go`；`handleReply`(119)/`speak`(104) 再各拆 2 个内部函数 | 中：并发纪律（`m.mu` / atomic.Pointer / channel close）注释密集，搬迁时必须连同注释一起搬，否则后人会破坏 P0#12 的免 race 约定 |
| P2 | `internal/whisper/db/repos/memory_facts.go` | `factArgs` 71 行 + `toFact` 67 行（27 列） | 抽 `factTimes`/`factJSONCols`/`ageArgs`/`ageFromRow` | 低：纯提取 |
| P2 | `internal/whisper/emotion.go` `EmotionStep`(113) / `interpreter.go` `InterpretInput`(102) | 关键词矩阵 + 多分支 | 表驱动化（关键词表已在 `interpreter.go` 顶部定义为 var，只是判定逻辑没表驱动） | 中：情绪判定是陪伴体验核心，行为漂移不可察觉，需快照测试 |
| P3 | `internal/whisper/desktop_router.go` + `desktop_executor.go`（30 KB） | `ExecuteUseComputer` 109 + `ExecuteDesktopAgentAction` 101 | 动作表驱动（`map[DesktopAgentAction]handler`） | 中：涉及确认/审计/路径策略，安全语义不能丢 |
| P3 | `internal/whisper/db/repos/fts.go`（350 行） | 4 组同构函数 | 见收敛评估 ③④⑤ | 中：中文召回降级路径必须保留 |

---

## 不建议动的地方（说明为什么）

1. **`internal/whisper/db/schema_v1.go` … `schema_v15.go`（15 文件 353 行）** — 迁移链是不可压缩的历史。合并成 `migrations = []string{...}` 单文件会破坏 `user_version` 与文件的一一对应（`database.go:175` 用 `v-1` 索引），且 V13 里已有 `DROP TABLE weixin_*` 这类"历史动作"。**实测也无重复**：24 张表名、9 个索引名全部唯一，没有可合并的冗余。

2. **`internal/ocr/herdsman.go` 的响应结构体字段**（`Result.Box`/`ImageWidth`/`ImageHeight`、`ParsePage.PageNumber` 等）— 这些字段**只靠 `json:"..."` tag 使用**（`encoding/json` 反射填充），静态引用计数为 1 不代表死。删除会静默丢掉 OCR 识别结果的坐标/尺寸/页码。标 `confidence: low`，**禁止删除**。

3. **`internal/whisper/params.go` 的 26 个"零引用"常量** — 该文件的自我定位是"100% 对齐 ackem ackemParams.ts 的参数总表"，且同语义参数在消费端存在**小写镜像**（`params.go:91 EffectiveTrustL1Weight` ↔ `memory_binding.go:33 effectiveTrustL1Weight`；`params.go:124 ConsolidationInsightWeight` ↔ `memory_consolidator.go:17 consolidationInsightWeight`）。技术上删除安全（纯数值、无反射），但会丢掉"参数出处"的可追溯性。建议**保留 + 在注释标注"未被消费"**，或迁移到 `docs/` 参数表。

4. **`internal/voice/voice_config.go` 的 `TTSEngine*` / `ASRModel*` 常量** — 这些是**前后端配置契约的枚举**：字面量 `"edge-tts"`(24 处)、`"qwen3-tts"`(94 处)、`"funasr-nano"`(5 处) 在 `internal/` 广泛出现，值来自前端 `voiceRuntimeConfig`。删除常量虽然代码安全，但会固化"魔法字符串散落"的现状。**建议反向做**：让这些字面量改用常量。同理保留 `agent_task_plan.go` 的 `Phase*`、`types.go` 的 `Scene*`/`Emergence*`、`memory_audit.go` 的 `Audit*`、`turn_plan_prompt.go` 的 `Goal*`/`RouteExtension*`。

5. **`internal/asr/provider.go` 的 `ASRProvider` 接口（3 方法、1 实现）与 `internal/voice/voice_manager.go:33` 的 `EventEmitter`（9 方法、1 实现）** — 这是刻意设计的 seam（注释明确写了"消费者只依赖接口，不依赖具体引擎"，`ASRProviderKinds`/注册表是 fail-closed 扩展点）。`TranscribeBytes` 目前只有测试 mock 实现，但它是接口契约的一部分。**不要为"消掉单实现接口"而内联**。

6. **`internal/tts/sapi.go`（无 build tag 却用 `syscall.SysProcAttr{HideWindow}`）** — 该字段仅 Windows 存在，理论上应加 `//go:build windows`。但本项目是 Wails Windows 桌面应用，加约束会牵动构建矩阵（`build/` 与 CI 未审）。仅登记为**平台可移植性备注**，不在本次改动范围。

7. **`internal/tts/sapi.go:86` / `xai.go:128` 的未使用参数 `p TTSParams`** — 函数注释明确写"能力外忽略 Speed/Pitch/Style/Emotion（不报错）"，这是 `TTSProvider` 接口签名的必然结果。删除参数会破坏接口一致性。**保留**。

8. **`internal/whisper/db/repos/*` 的 `toFact`/`toEpisode`/`toTriple`/`toAnchor` 行映射与 `factRow`/`episodeRow` 等 row 结构体** — 与域类型字段镜像（D8 候选），但它们承载 `sql.NullString`/`sql.NullInt64` 的中转语义，是 `database/sql` 的必经样板。**不要用域类型直接 Scan**：空值区分会丢失（`NULL` vs `""`）。

9. **`internal/voice/voice_manager.go` 的并发纪律注释与 `atomic.Pointer` 三回调** — 注释记录了 2026-10-02 审计 P0#12 的 data-race 修复（写侧 atomic、读侧一次快照防 TOCTOU）。任何"顺手简化"（如改回裸字段、把快照拆成两次 Load）都会重新引入 race。**搬迁时逐字保留**。

10. **`GetDatabase` 的 DSN 单源 PRAGMA**（`database.go:44-46`，注释 T6-5.5）— "五个连接参数全部在 DSN 中声明，不再重复执行 PRAGMA 循环，避免双来源漂移"。不要为了"可读性"把它拆回 `db.Exec("PRAGMA ...")`。

---

## 附：审计方法与可复现性

- 引用计数：Go 感知扫描器（逐字符状态机剥离 `//`、`/* */`、`"..."`、`` `...` ``、`'...'`，保留行结构），再用 `(?<![\w])Sym\b` 统计——即"允许点号限定调用、禁止长标识符前缀误配"。
- 函数边界：花括号配对（不是"行首 `}`"），因此单行函数不会吞掉后继函数。
- 交叉验证：对 42 个抽样符号执行 `rg -n -w --glob '*.go'`，全部与扫描器一致（`AgentLoop` 的 4 处命中经人工确认全是注释）。
- Wails 风险：把 175 个死符号名与 `frontend/src`（1 172 文件）做全量 token 比对 → 仅 1 个名字命中（`Box`），且是无关的图标组件名 → **无 Wails 字符串调用风险**。
- 平台风险：扫描 `internal/{voice,whisper,tts,asr,ocr}` 全部 .go 的 build tag 与文件名后缀 → **0 个 `//go:build` 文件、0 个 `_windows.go`/`_unix.go` 文件**，故本次死代码判定不受构建标签影响（`tts/sapi.go` 的可移植性另记为备注）。
- 反射/序列化风险：所有含 `json:` tag 的"零引用"项（`ocr.Result` 字段、`MemoryAuditReport`、`ContradictionResult`、`ForegroundSnapshot`、`llm_client.go` 两类型）一律降级为 `medium`/`low` 并要求人工确认。

**本次审计发现的"看起来像问题但其实没问题"清单（避免后续误报）**：`LoadChatHistoryFromDB`（`app/whisper_handler.go:760` 在用）、`ClearStructuredData`（测试在用）、`ResolveDesktopCapabilityEnhanced`（`desktop_test.go` 在用）、`AppendTurnTraceToDB`/`LoadTurnTracesFromDBSession`（`app/` 在用）、`SetPTTActive`/`ASRReady`/`WhisperReady`/`RealtimeReady`（`app/voice_handler.go` 在用）、`LogTurn`（`orchestrator.go:471` 在用）。
