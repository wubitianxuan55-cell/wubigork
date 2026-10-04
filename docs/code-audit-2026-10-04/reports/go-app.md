# Go 审计 · internal/app（Wails 绑定层）

> 只读审计。审计基准：仓库工作副本（未做任何修改），审计脚本与中间数据全部落在 `%TEMP%\gaea-audit\`，仓库内只新增本报告文件。
> 「死代码」判据严格落实三层检查：① `frontend/wailsjs/go/**` 生成绑定是否含该方法名；② `frontend/src/**`（含 `bindingNames.ts` / `legacyBindings.ts` / `mock/bindingSignatures.ts` / `drift.ts` / 任意字符串字面量）是否含该名字；③ 全仓 Go（`internal/app` 含 `_test.go`、`internal/**` 其它包、`main.go`、`scripts/**`）是否有调用点。三者皆无才判 D1。

## 范围与文件统计

| 指标 | 数值 |
| --- | --- |
| `internal/app/**/*.go` 文件总数 | **532** |
| 其中非测试文件 | **243** |
| 其中 `_test.go` | 289 |
| 其中 `internal/app/board/`（非测试） | 3 |
| 非测试代码行数（含注释/空行） | **61556** |
| 测试代码行数 | 54738 |
| 非测试函数/方法声明数 | **2758** |
| 全目录字节数 | 4218 KB |
| 绑定门面文件 `bindings_*.go`（生成物） | 11 个 / 1594 行 |
| 注册为 Wails 绑定方法的导出方法（`bindingNames.ts`） | 635 |
| 已在册「零调用摘除」方法（`scripts/gen_bindings` excludedBindings） | 109 |

### 最大的 20 个文件（按字节，非测试）

| # | KB | 行数 | 文件 |
| --- | --- | --- | --- |
| 1 | 45.2 | 1019 | `internal/app/create_chapter_handler.go` |
| 2 | 44.3 | 1161 | `internal/app/image_handler.go` |
| 3 | 39.7 | 1191 | `internal/app/gaea_ui_extra.go` |
| 4 | 39.0 | 938 | `internal/app/sin_handler.go` |
| 5 | 36.8 | 1030 | `internal/app/whisper_handler.go` |
| 6 | 35.0 | 904 | `internal/app/app.go` |
| 7 | 33.6 | 849 | `internal/app/gaea_handler.go` |
| 8 | 33.1 | 865 | `internal/app/consistency_deep_handler.go` |
| 9 | 32.5 | 857 | `internal/app/gaea_dag.go` |
| 10 | 31.9 | 831 | `internal/app/novel_plan_handler.go` |
| 11 | 30.8 | 896 | `internal/app/characterlib_gen_handler.go` |
| 12 | 30.3 | 869 | `internal/app/memory_hub.go` |
| 13 | 29.7 | 775 | `internal/app/create_chapter_context.go` |
| 14 | 29.1 | 800 | `internal/app/gaea_ui.go` |
| 15 | 28.8 | 737 | `internal/app/novel_booksource_handler.go` |
| 16 | 27.0 | 656 | `internal/app/intent_router.go` |
| 17 | 25.9 | 734 | `internal/app/voice_handler.go` |
| 18 | 24.1 | 777 | `internal/app/gaea_cost_import_vision.go` |
| 19 | 23.6 | 582 | `internal/app/wx_agent.go` |
| 20 | 22.8 | 567 | `internal/app/novel_rewrite_handler.go` |

补充：`internal/app/board/` 是唯一的子包（board manifest / builtins / registry），体量很小，未见死代码；本域的结构性负担 100% 在 `internal/app` 顶层。

## Top 20 发现（按 可删行数 × 安全度 排序）

安全度定义：**A** = 删掉不影响编译与运行时（无任何调用点/无前端契约）；**B** = 只影响本域内部结构，需同步生成器或前端清单；**C** = 触及绑定签名/前端契约，需人工确认。

1. **[P0] `internal/app`（64 个方法，见 D1 全量表）** — D1 — LOC≈963 — confidence:high
   - 证据：109 个 excludedBindings 中 64 个方法名在全仓 Go（含测试）、frontend/wailsjs/go/**、frontend/src/** 三层均零命中，仅存在于 scripts/gen_bindings/main.go 的 excludedBindings 登记表与 internal/app/bindings_completeness_test.go 的同源物化清单里。单个方法最大 85 行（image_handler.go:975 GetImageBackendConfig），见下方 D1 全量表。
   - 建议：① 逐个删除方法体；② 从 scripts/gen_bindings/main.go excludedBindings 与 bindings_completeness_test.go 同名清单中同步除名（该测试只校验「未除名的方法必须挂门面」，除名不删方法体也能过，所以清理必须两处一起动）；③ 跑 go run ./scripts/gen_bindings -shadow-check 确认无新遮蔽。风险最低的是 loc≤3 的 30 个空壳方法（见全量表尾部）。
2. **[P0] `internal/app`（30 组重复簇，见 D4 全量表）** — D4 — LOC≈615 — confidence:high
   - 证据：去注释/去字符串/去标识符/去数字后函数体文本完全一致（md5 相同）。最大簇是 10 份 `Schema()`（115 行，可收敛 105 行）与 5 份 `Set*Key()`（85 行，可收敛 68 行）。全部 30 组明细见全量表。
   - 建议：按簇提取统一 helper（详见「统一 helper 收敛清单」）。收敛后总可删约 615 行。
3. **[P0] `internal/app`（13 个方法，见 D1 全量表 B 段）** — D1 — LOC≈223 — confidence:high
   - 证据：引用 100% 来自 internal/app/*_test.go（novel_state_acceptance_test.go / whisper_taskplan_test.go / main_brain_chat_test.go / engine_list_test.go 等），生产代码与前端零调用。合计 223 行。
   - 建议：先判断这些测试是否本身已废弃（例如 WhisperTaskPlanStatus/Resume 只被 whisper_taskplan_test.go 调）；确认后「删方法体 + 删对应测试」或「保留但标注 deprecated」。切勿只删方法体。
4. **[P0] `internal/app/image_handler.go:975` `GetImageBackendConfig`** — D1 — LOC≈85 — confidence:high
   - 证据：全仓 Go 调用点 0（`grep -rn GetImageBackendConfig` 仅命中自身声明 + `scripts/gen_bindings/main.go` 登记表 + `bindings_completeness_test.go` 清单）；`frontend/wailsjs/go/**` 无此名；`frontend/src` 字符串无此名。
   - 建议：删除方法体并同步两处清单；先确认 `GetImageBackend`（同名族、仍被 internal/ai 与 image_backend_type_app_test.go 引用）未在其间复用逻辑。附注：该名字在 `CHANGELOG.md`、`docs/code-audit-2026-10-02/machine/bindings.json`、`releases/*.md` 里有历史记载——文档引用不算调用方，但删它会让这些文档成为「提到已不存在的方法」的陈旧陈述，建议随删除在 CHANGELOG 留一行。
5. **[P0] 3 个「整文件可删」文件：`context_handler.go`(229 行)、`lorebook_handler.go`(78 行)、`gaea_herdsman_probe.go`(20 行)** — D1 — LOC≈327 — confidence:high
   - 证据：逐一核对该文件内**全部**顶层函数声明都落在零调用/仅测试调用名单里，且文件内无其它 `type`/`var`/`const` 顶层符号（`context_handler.go` 6/6 方法、`lorebook_handler.go` 3/3、`gaea_herdsman_probe.go` 1/1）。三个文件的三层检查（wailsjs / frontend/src / 全仓 Go）全部零命中。
   - 建议：`context_handler.go`、`lorebook_handler.go` 直接 `git rm`（注意 `lorebook_handler.go` 里的 `GetLorebookEntries` 等 3 个方法名要同步从两处清单除名）；`gaea_herdsman_probe.go` 的 `HerdsmanProbe` 仅被 `gaea_herdsman_probe_test.go` 调用，需连同该测试一起删。
6. **[P0] `internal/app/platform_handler.go:87` `GetDashboard`** — D1 — LOC≈141 — confidence:high
   - 证据：Go 调用点 0（仅登记表）；wailsjs 无；frontend/src 无。同文件 `GetCompileTemplates`:17(13 行)、`GetStyleProfile`:186(17 行)、`ImportStyleProfile`:205(18 行)、`AnalyzeStyle`:149(35 行) 同因同为死代码，5 个合计 141 行。
   - 建议：删这 5 个方法；**保留** `ExportHTML`:32（有 `internal/export/html.go` + `platform_handler.go` 自身 2 处引用）与它唯一的私有 helper `sanitizeFilename`:70 —— 该文件不能整删。
7. **[P0] `internal/app/tts_handler.go:16-52`（旧版子进程 TTS 6 个空壳方法）** — D1 — LOC≈27 — confidence:high
   - 证据：`GetTTSConfig`/`SaveTTSConfig`/`GetTTSStatus`/`StartTTSServer`/`StopTTSServer` 注释自称「已废弃…无操作」，函数体仅 `return nil`/返回空 map；`TTSSpeak` 直接返回错误。Go 调用点 0；wailsjs/frontend/src 零命中。`SaveTTSConfig` 的 5 个参数、`StartTTSServer` 的 3 个参数全部未使用（见 D7）。
   - 建议：整段删除；前端若仍传这些绑定名，会退化为「方法不存在」错误——虽然文本普查为 0 命中，建议真机点一次 TTS 面板再删。
8. **[P0] `internal/app/office_handler.go:15-28`（6 个 1 行包装方法）** — D1 — LOC≈13 — confidence:high
   - 证据：`OfficeIsTask`/`OfficeListFolder`/`OfficeReadFile`/`OfficeGetJobState`/`OfficeCancelJob`/`OfficeGetMode`/`OfficeSetMode` 均为 1~6 行；Go 调用点 0（`OfficeExecute` 例外，它有 2 处生产调用，**不要删**）；wailsjs/frontend/src 零命中。
   - 建议：删 6 个死方法，保留 `OfficeExecute`。
9. **[P1] `internal/app` 31 个「形参零使用」方法** — D7 — LOC≈31 — confidence:medium
   - 证据：全目录 243 个非测试文件、2758 个函数，逐函数解析形参并在函数体内做词边界计数，得 31 个形参零使用（已排除函数类型参数与 `_`）。典型：`tts_handler.go:27 SaveTTSConfig` 的 5 个形参全部未用（body 只有 `return nil`，是废弃空壳）、`tts_handler.go:40 StartTTSServer` 3 个未用、`create_chapter_handler.go:478 streamCreateChapter` 的 `plotReq`/`of` 未用（286 行函数内的死参数）、`gaea_ui.go:725/726 GaeaSummarizeFrom/UpTo` 的 `turn` 未用、`gaea_ui_extra.go:1191 GaeaSaveWindowState` 的 `state` 未用。
   - 建议：**绑定方法的形参不能删**（Wails 反射按签名转发、前端 `bindingSignatures.ts` 按形参个数做契约测试），只能把形参改名为 `_` 或补注释说明为何忽略；真正的内部函数（如 `sinRefPlan`、`pickFlaggedSentences`、`wxAgentIntentReminder`）可直接删参数。
10. **[P0] `internal/app/voice_handler.go:461-706`（`VoiceSetMode`/`VoiceSetInputChannel`/`VoiceGetState`/`VoiceRestartService`）** — D1 — LOC≈61 — confidence:high
   - 证据：4 个方法 Go 调用点 0；且 `VoiceSetMode` 与 `VoiceSetInputChannel` 是逐字节相同的重复实现（D4，见全量表）。
   - 建议：删两个重复实现里的一个（先删 `VoiceSetInputChannel`）+ 其余死方法；注意 `voice_handler.go` 里另有 `EmitVoice*` 6 个方法仅本文件调用，属可内联项。
11. **[P0] `internal/app/whisper_handler.go`（`WhisperGetConfig`/`WhisperSetEngine`/`WhisperGetEngines`/`WhisperSetModel`/`WhisperSetImageModel`/`WhisperChatWithSearch`/`WhisperWebSearch` 7 个）** — D1 — LOC≈55 — confidence:high
   - 证据：7 个方法 Go 调用点 0（`WhisperChat`/`WhisperGetEngine`/`WhisperGetModel`/`WhisperGetImageModel` 有调用点，**不要删**）；`WhisperSetEngine` 与 `WhisperSetImageModel` 另属逐字节重复簇。
   - 建议：删 7 个死方法 + 合并重复的 Set 对。
12. **[P1] `internal/app/gaea_cost_*.go` + `memory_hub.go`（`hubCost*Store()` 4 份 + 22 个 1 行转发绑定）** — D5 — LOC≈130 — confidence:high
   - 证据：`hubCostInquiryStore`/`hubCostProjectStore`/`hubCostRefStore`/`hubCostStageStore` 4 个函数体归一化后完全相同（各 10 行，可收敛 30 行）；`GaeaCostInquiry{Save,List,Delete,Expiring,Scan}`、`GaeaCostProject{Save,List,Delete}`、`GaeaCostEstimate{ItemSave,ItemDelete,Items,Versions}`、`GaeaCostNote{Save,List,Delete,BumpRef}`、`GaeaCostStage{Save,Stages}`、`GaeaPriceSource{Save,Delete}`、`GaeaPriceFetches/Ignore/History` 等 28 个方法是 `return a.hubXxxStore().Y(...)` 的单行转发。
   - 建议：把 4 份 store 覆写逻辑收敛为 `hubStore(kind)` 表驱动；单行转发保留（绑定面必须逐名暴露），但可把它们集中到一个 `*_bindings_thin.go`，与业务实现分离。
13. **[P1] `internal/app/model_engine_handler.go:222/252/292/320/352`（5 份 `Set*Key`）** — D4 — LOC≈68 — confidence:high
   - 证据：`SetDeepseekKey`/`SetGlmKey`/`SetOpencodeGoKey`/`SetOpencodeZenKey`/`SetModelHubKey` 5 个函数体各 17 行，归一化后逐字节相同。
   - 建议：抽 `setProviderKey(provider string, key string) error` + 5 个 3 行包装；或改为表驱动 `map[string]func`。
14. **[P1] 10 份 `Schema()` 方法（`gaea_diagram.go:59`、`gaea_factbase.go:84/140/178`、`gaea_routine_llm.go:29`、`gaea_specialist_tools.go:37/94`、`gaea_tools.go:30`、`gaea_translate.go:303`、`sin_tool_illustrate.go:57`、`sin_tool_notes.go:161/308`）** — D4 — LOC≈105 — confidence:high
   - 证据：10 个 `Schema()` 归一化后完全一致（各 10~15 行），合计 115 行。`CompactSchema`/`CompactDescription`/`ReadOnly`/`Description` 也各有多份结构相同的实现（同属工具描述样板）。
   - 建议：抽 `toolSchema(fields ...schemaField) json.RawMessage` + 每工具一行；`ReadOnly() bool` 统一由 embedding 的基类提供（`toolBase`），可再收敛 20+ 行。
15. **[P1] `internal/app/gaea_ui_extra.go:399` `GaeaPermLevel` / `gaea_handler.go:533/543/551/556/569/586`** — D1 — LOC≈42 — confidence:high
   - 证据：`GaeaPermLevel`(6)、`GaeaModel`(8)、`GaeaEngines`(6)、`GaeaSetEngine`(3)、`GaeaTools`(11)、`GaeaSkills`(15)、`GaeaCallTool`(7) 全为零调用（仅登记表命中）。
   - 建议：删除；这几处是 v3 时代的旧绑定面残骸。
16. **[P1] `internal/app/intent_router.go:262/308/338` 等 35 处空操作错误处理** — D2 — LOC≈105 — confidence:medium
   - 证据：`if err != nil { return nil }`（丢弃 err 且不记日志）共 35 处、分布 28 个文件；`intent_router.go:262`（`ListSessions` 失败静默返回 nil 会话集）、`gaea_dag.go:817/821`、`whisper_handler.go:104/366`、`create_chapter_handler.go:374`（返回 false 无日志）等。
   - 建议：区分「真正的 best-effort」与「漏了 slog.Warn」：前者补 `// best-effort: ...` 注释（利于后人不再当 bug 修），后者补日志。不要机械替换成 `_ =`（会触发 errcheck 豁免争议）。
17. **[P1] `internal/app/herdsman_operations.go:16` `HerdsmanOperation` 与 `:36` `herdsmanRawOperation`** — D8 — LOC≈11 — confidence:high
   - 证据：两者字段名集合完全相同（ID/Kind/Model/Status/Stage/Progress/Artifacts/CreatedAt/CompletedAt），仅 `Artifacts` 类型不同（`int` vs `[]any`），是「原始 JSON → 视图」的一次性映射。
   - 建议：保留 raw 结构（JSON 契约），把 `HerdsmanOperation` 的构造抽成 `toOperation(raw)`，并在 raw 上加注释说明为什么必须两份；或直接用 `Artifacts []any` + 自定义 MarshalJSON 收敛成一份。
18. **[P1] `internal/app/gaea_file_index.go:14` `FileIndexStatus`（未使用导出类型）** — D1 — LOC≈5 — confidence:high
   - 证据：全仓 `grep -rn FileIndexStatus` 仅命中 `gaea_file_index.go:13`（注释）与 `:14`（声明）两行；`frontend/wailsjs/**`（含 `models.ts`）与 `frontend/src/**` 零命中。注意区分同族的 `SemanticIndexStatus`（`gaea_semantic_search.go:37` 的真实返回类型，**在用**，不要动）——`FileIndexStatus` 是它的未采用孪生（字段为 total/skipped/error）。
   - 建议：直接删除这 5 行；`GaeaFileIndexRebuild` 的结果实际走 `tasks.Task`，不需要该类型。
19. **[P1] `internal/app/create_chapter_context.go:29` `ctxForeshadowLineLen`、`internal/app/modelcatalog.go:17/18` `imageTierFreeFallback`/`imageTierRemoteHeavy`** — D1 — LOC≈3 — confidence:high
   - 证据：三个常量声明后零引用（`grep -rn` 仅命中声明行；`internal/app` 内、全仓 Go、wailsjs、frontend/src 均无）。
   - 建议：直接删 3 行；`imageTierFreeFallback`/`imageTierRemoteHeavy` 疑似 `imageTierFree`/`imageTierRemote` 的历史别名，删前确认 `modelcatalog.go` 内没有靠字符串 `free_fallback` 间接使用（已确认无）。
20. **[P0] `internal/app/gaea_cost_stage.go:14/20` `SetCostStageStoreForTest` / `ResetCostStageStoreForTest`（未使用的测试注入 seam）** — D1 — LOC≈6 — confidence:high
   - 证据：全仓 Go（含 `_test.go`）零调用；对照组 `SetCostStoreForTest`/`SetCostProjectStoreForTest`/`SetCostRefStoreForTest`/`SetCostInquiryStoreForTest` 在测试里被大量调用（19/12/4/8 次），唯独 stage 这对从未被使用——已确认不是「有同名 wrapper」。
   - 建议：删除这两个函数及 `costStageStoreOverrideSet` 若无其它引用（删除后 `costStageStoreOverride` 相关分支可能连带简化）。

## 全量发现表

字段说明：`LOC` = 可删/可收敛行数估算；`置信度` = 对「这条发现本身成立」的置信度；`证据` 中「Go 调用点 0」= 全仓 Go（含 `_test.go`）除自身声明外零出现，且 wailsjs/前端字符串零命中（三层检查）。

| # | 文件:行 | 符号 | 类型 | LOC | 置信度 | 证据 | 建议 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `internal/app/image_handler.go:975` | `GetImageBackendConfig` | D1 | 85 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 2 | `internal/app/platform_handler.go:87` | `GetDashboard` | D1 | 58 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 3 | `internal/app/context_handler.go:112` | `BuildContextBudget` | D1 | 43 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 4 | `internal/app/context_handler.go:157` | `GetAllEntityNames` | D1 | 41 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 5 | `internal/app/context_handler.go:49` | `InjectMemories` | D1 | 36 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 6 | `internal/app/platform_handler.go:149` | `AnalyzeStyle` | D1 | 35 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 7 | `internal/app/graph_handler.go:110` | `QueryEntities` | D1 | 31 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 8 | `internal/app/lorebook_handler.go:28` | `SaveLorebookEntry` | D1 | 31 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 9 | `internal/app/context_handler.go:200` | `BuildRichContext` | D1 | 30 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 10 | `internal/app/context_handler.go:18` | `SearchMemories` | D1 | 29 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 11 | `internal/app/graph_handler.go:80` | `SyncEntityDB` | D1 | 28 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 12 | `internal/app/context_handler.go:87` | `FindLorebookTriggers` | D1 | 23 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 13 | `internal/app/gaea_herdsman_health.go:17` | `HerdsmanHealth` | D1 | 23 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 14 | `internal/app/auth_handler.go:107` | `SaveToken` | D1 | 23 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 15 | `internal/app/lorebook_handler.go:61` | `DeleteLorebookEntry` | D1 | 18 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 16 | `internal/app/platform_handler.go:205` | `ImportStyleProfile` | D1 | 18 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 17 | `internal/app/voice_handler.go:699` | `VoiceGetState` | D1 | 18 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 18 | `internal/app/voice_handler.go:482` | `VoiceSetInputChannel` | D1 | 18 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 19 | `internal/app/voice_handler.go:461` | `VoiceSetMode` | D1 | 18 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 20 | `internal/app/platform_handler.go:186` | `GetStyleProfile` | D1 | 17 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 21 | `internal/app/module_bindings.go:141` | `RunModule` | D1 | 17 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 22 | `internal/app/analysis_handler.go:173` | `GetBookData` | D1 | 16 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 23 | `internal/app/gaea_handler.go:569` | `GaeaSkills` | D1 | 15 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 24 | `internal/app/worldview_handler.go:83` | `SaveWorldMapImage` | D1 | 15 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 25 | `internal/app/whisper_handler.go:516` | `WhisperWebSearch` | D1 | 15 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 26 | `internal/app/outline_handler.go:34` | `ChatOutline` | D1 | 13 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 27 | `internal/app/outline_handler.go:49` | `ChatOutlineNode` | D1 | 13 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 28 | `internal/app/platform_handler.go:17` | `GetCompileTemplates` | D1 | 13 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 29 | `internal/app/lorebook_handler.go:13` | `GetLorebookEntries` | D1 | 13 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 30 | `internal/app/worldview_handler.go:100` | `GetWorldMapImage` | D1 | 13 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 31 | `internal/app/brain_bindings.go:56` | `BrainCrossRefs` | D1 | 12 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 32 | `internal/app/gaea_handler.go:556` | `GaeaTools` | D1 | 11 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 33 | `internal/app/outline_handler.go:76` | `AddOutlineNode` | D1 | 10 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 34 | `internal/app/character_handler.go:182` | `SaveCharacters` | D1 | 10 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 35 | `internal/app/outline_handler.go:64` | `SaveOutlineNode` | D1 | 10 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 36 | `internal/app/whisper_handler.go:490` | `WhisperGetConfig` | D1 | 10 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 37 | `internal/app/whisper_handler.go:427` | `WhisperSetEngine` | D1 | 10 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 38 | `internal/app/tts_handler.go:16` | `GetTTSConfig` | D1 | 9 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 39 | `internal/app/whisper_handler.go:449` | `WhisperSetModel` | D1 | 9 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 40 | `internal/app/gaea_handler.go:533` | `GaeaModel` | D1 | 8 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 41 | `internal/app/whisper_handler.go:470` | `WhisperSetImageModel` | D1 | 8 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 42 | `internal/app/gaea_handler.go:586` | `GaeaCallTool` | D1 | 7 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 43 | `internal/app/voice_handler.go:690` | `VoiceRestartService` | D1 | 7 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 44 | `internal/app/brain_bindings.go:26` | `BrainWrite` | D1 | 6 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 45 | `internal/app/gaea_handler.go:543` | `GaeaEngines` | D1 | 6 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 46 | `internal/app/gaea_ui_extra.go:399` | `GaeaPermLevel` | D1 | 6 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 47 | `internal/app/voice_model_handler.go:182` | `GetChatVoiceModel` | D1 | 6 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 48 | `internal/app/tts_handler.go:32` | `GetTTSStatus` | D1 | 6 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 49 | `internal/app/office_handler.go:16` | `OfficeListFolder` | D1 | 6 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 50 | `internal/app/worldview_handler.go:33` | `SaveWorldviewSection` | D1 | 6 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 51 | `internal/app/project_handler.go:66` | `CloseProject` | D1 | 4 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 52 | `internal/app/whisper_handler.go:533` | `WhisperChatWithSearch` | D1 | 4 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 53 | `internal/app/gaea_handler.go:551` | `GaeaSetEngine` | D1 | 3 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 54 | `internal/app/office_handler.go:22` | `OfficeReadFile` | D1 | 3 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 55 | `internal/app/tts_handler.go:27` | `SaveTTSConfig` | D1 | 3 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 56 | `internal/app/tts_handler.go:40` | `StartTTSServer` | D1 | 3 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 57 | `internal/app/tts_handler.go:45` | `StopTTSServer` | D1 | 3 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 58 | `internal/app/tts_handler.go:50` | `TTSSpeak` | D1 | 3 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 59 | `internal/app/whisper_handler.go:501` | `WhisperGetEngines` | D1 | 3 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 60 | `internal/app/office_handler.go:26` | `OfficeCancelJob` | D1 | 1 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 61 | `internal/app/office_handler.go:25` | `OfficeGetJobState` | D1 | 1 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 62 | `internal/app/office_handler.go:27` | `OfficeGetMode` | D1 | 1 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 63 | `internal/app/office_handler.go:15` | `OfficeIsTask` | D1 | 1 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 64 | `internal/app/office_handler.go:28` | `OfficeSetMode` | D1 | 1 | high | Go 调用点 0（三层检查通过）；仅登记表 + `bindings_completeness_test.go` 在册；已摘出绑定面（不在 635 个绑定名内） | 删方法体 + 两处清单除名 |
| 65 | `internal/app/chapter_handler.go:517` | `CreateSnapshot` | D1 | 34 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/novel_state_acceptance_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 66 | `internal/app/main_brain.go:48` | `MainBrainChat` | D1 | 31 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/board_manifest_test.go, internal/app/main_brain_chat_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 67 | `internal/app/chapter_handler.go:579` | `RestoreSnapshot` | D1 | 27 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/novel_state_acceptance_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 68 | `internal/app/whisper_taskplan.go:36` | `WhisperTaskPlanStatus` | D1 | 27 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/whisper_taskplan_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 69 | `internal/app/chapter_handler.go:553` | `ListSnapshots` | D1 | 24 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/novel_state_acceptance_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 70 | `internal/app/character_handler.go:18` | `ChatCharacter` | D1 | 19 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/embed_check_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 71 | `internal/app/gaea_data_backup.go:181` | `GaeaDataBackupPending` | D1 | 15 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/gaea_data_backup_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 72 | `internal/app/whisper_handler.go:29` | `GetEngineList` | D1 | 14 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/engine_list_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 73 | `internal/app/whisper_taskplan.go:68` | `WhisperTaskPlanResume` | D1 | 12 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/whisper_taskplan_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 74 | `internal/app/module_bindings.go:122` | `CheckModuleIntegrity` | D1 | 7 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/board_manifest_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 75 | `internal/app/chapter_handler.go:608` | `MigrateProjectToV4` | D1 | 7 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/novel_state_acceptance_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 76 | `internal/app/gaea_herdsman_probe.go:18` | `HerdsmanProbe` | D1 | 3 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/gaea_herdsman_probe_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 77 | `internal/app/gaea_translate.go:139` | `LocalTranslate` | D1 | 3 | high | 生产 Go 调用点 0；引用仅来自 `internal/app/gaea_translate_test.go`；wailsjs/前端零命中 | 确认测试是否废弃后「方法体+测试」一起删 |
| 78 | `internal/app/context_handler.go:1` | `（整文件）context_handler.go` | D1 | 229 | high | 文件内 6/6 顶层函数（SearchMemories/InjectMemories/FindLorebookTriggers/BuildContextBudget/GetAllEntityNames/BuildRichContext）全部零调用；文件无其它 type/var/const 顶层符号；wailsjs/frontend/src 零命中 | git rm 整个文件；同步两处 excludedBindings 清单除名 |
| 79 | `internal/app/lorebook_handler.go:1` | `（整文件）lorebook_handler.go` | D1 | 78 | high | 文件内 3/3 顶层函数（GetLorebookEntries/SaveLorebookEntry/DeleteLorebookEntry）全部零调用；无其它顶层符号 | git rm 整个文件；同步清单除名 |
| 80 | `internal/app/gaea_herdsman_probe.go:1` | `（整文件）gaea_herdsman_probe.go` | D1 | 20 | high | 文件内 1/1 顶层函数 HerdsmanProbe 仅被 gaea_herdsman_probe_test.go 调用（生产零调用） | 删文件 + 删 gaea_herdsman_probe_test.go 中对应测试 |
| 81 | `internal/app/gaea_cost_stage.go:14` | `SetCostStageStoreForTest` | D1 | 4 | high | 全仓 Go 含测试零调用（对照：其它 SetXxxForTest 被调用 4~19 次） | 删函数 + 同步 costStageStoreOverrideSet 分支 |
| 82 | `internal/app/gaea_cost_stage.go:20` | `ResetCostStageStoreForTest` | D1 | 1 | high | 同上 | 删函数 |
| 83 | `internal/app/gaea_file_index.go:14` | `FileIndexStatus` | D1 | 5 | high | 全仓 grep 仅命中自身声明+注释两行；wailsjs `models.ts` / frontend/src 零引用（与在用类型 SemanticIndexStatus 不同名，勿混淆） | 删类型 |
| 84 | `internal/app/create_chapter_context.go:29` | `ctxForeshadowLineLen` | D1 | 1 | high | 声明后零引用（grep 仅命中声明行） | 删常量 |
| 85 | `internal/app/modelcatalog.go:17` | `imageTierFreeFallback` | D1 | 1 | high | 声明后零引用 | 删常量 |
| 86 | `internal/app/modelcatalog.go:18` | `imageTierRemoteHeavy` | D1 | 1 | high | 声明后零引用 | 删常量 |
| 87 | `internal/app/gaea_diagram.go:59` | Schema | D4 | 105 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），10 份：`internal/app/gaea_diagram.go:59`, `internal/app/gaea_factbase.go:84`, `internal/app/gaea_routine_llm.go:29`, `internal/app/gaea_specialist_tools.go:37`, `internal/app/gaea_specialist_tools.go:94`, `internal/app/gaea_tools.go:30` …共 10 处 | 抽统一 helper 后删除 9 份实现 |
| 88 | `internal/app/model_engine_handler.go:222` | SetDeepseekKey / SetGlmKey / SetModelHubKey / SetOpencodeGoKey / SetOpencodeZenKey | D4 | 68 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），5 份：`internal/app/model_engine_handler.go:222`, `internal/app/model_engine_handler.go:252`, `internal/app/model_engine_handler.go:292`, `internal/app/model_engine_handler.go:320`, `internal/app/model_engine_handler.go:352` | 抽统一 helper 后删除 4 份实现 |
| 89 | `internal/app/gaea_file_index.go:48` | startFileIndexCron / startPriceCron | D4 | 24 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/gaea_file_index.go:48`, `internal/app/gaea_price_sources.go:26` | 抽统一 helper 后删除 1 份实现 |
| 90 | `internal/app/herdsman_lifecycle.go:307` | HerdsmanModelDownload / HerdsmanModelStart / HerdsmanModelStop / HerdsmanModelUninstall | D4 | 36 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），4 份：`internal/app/herdsman_lifecycle.go:307`, `internal/app/herdsman_lifecycle.go:321`, `internal/app/herdsman_lifecycle.go:335`, `internal/app/herdsman_lifecycle.go:349` | 抽统一 helper 后删除 3 份实现 |
| 91 | `internal/app/gaea_cost_compose.go:25` | ensureCostCheckParams / ensureNovelReviewRubric / ensureNovelStylePatterns / ensureNovelStyleWords | D4 | 33 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），4 份：`internal/app/gaea_cost_compose.go:25`, `internal/app/novel_deslop_handler.go:30`, `internal/app/novel_deslop_handler.go:46`, `internal/app/novel_review_handler.go:29` | 抽统一 helper 后删除 3 份实现 |
| 92 | `internal/app/character_handler.go:134` | AddOutlineNode / SaveOrganization / SaveOutlineNode / SaveRelationship | D4 | 30 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），4 份：`internal/app/character_handler.go:134`, `internal/app/character_handler.go:162`, `internal/app/outline_handler.go:64`, `internal/app/outline_handler.go:76` | 抽统一 helper 后删除 3 份实现 |
| 93 | `internal/app/gaea_cost_inquiry.go:23` | hubCostInquiryStore / hubCostProjectStore / hubCostRefStore / hubCostStageStore | D4 | 30 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），4 份：`internal/app/gaea_cost_inquiry.go:23`, `internal/app/gaea_cost_projects.go:33`, `internal/app/gaea_cost_ref.go:30`, `internal/app/gaea_cost_stage.go:23` | 抽统一 helper 后删除 3 份实现 |
| 94 | `internal/app/character_handler.go:331` | libCharToProject / projectCharToLib | D4 | 18 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/character_handler.go:331`, `internal/app/character_handler.go:351` | 抽统一 helper 后删除 1 份实现 |
| 95 | `internal/app/gaea_schedule_mpp.go:19` | GaeaScheduleImportMpp / GaeaScheduleImportXlsx | D4 | 18 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/gaea_schedule_mpp.go:19`, `internal/app/gaea_schedule_xlsx.go:32` | 抽统一 helper 后删除 1 份实现 |
| 96 | `internal/app/model_router.go:145` | SetEngineFailover / SetOfficeLocal / SetOfflineMode / SetSensitiveLocal | D4 | 27 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），4 份：`internal/app/model_router.go:145`, `internal/app/model_router.go:168`, `internal/app/model_router.go:186`, `internal/app/model_router.go:198` | 抽统一 helper 后删除 3 份实现 |
| 97 | `internal/app/voice_handler.go:461` | VoiceSetInputChannel / VoiceSetMode | D4 | 18 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/voice_handler.go:461`, `internal/app/voice_handler.go:482` | 抽统一 helper 后删除 1 份实现 |
| 98 | `internal/app/gaea_handler.go:222` | GaeaSetMemoryBrief / GaeaSetMorningPreload | D4 | 18 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/gaea_handler.go:222`, `internal/app/gaea_handler.go:252` | 抽统一 helper 后删除 1 份实现 |
| 99 | `internal/app/image_handler.go:852` | SetPortraitConfig / SetSinImageConfig | D4 | 14 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/image_handler.go:852`, `internal/app/image_handler.go:877` | 抽统一 helper 后删除 1 份实现 |
| 100 | `internal/app/novel_booksource_handler.go:410` | NovelBookSourceImportCancel / SinBookSourceDownloadCancel | D4 | 13 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/novel_booksource_handler.go:410`, `internal/app/sin_booksource_handler.go:325` | 抽统一 helper 后删除 1 份实现 |
| 101 | `internal/app/story_spine_handler.go:229` | arcStageLabel / threadStatusLabel | D4 | 13 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/story_spine_handler.go:229`, `internal/app/story_spine_handler.go:305` | 抽统一 helper 后删除 1 份实现 |
| 102 | `internal/app/gaea_benchmark.go:488` | uniqueCtxSizes / uniqueModels | D4 | 12 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/gaea_benchmark.go:488`, `internal/app/gaea_benchmark.go:501` | 抽统一 helper 后删除 1 份实现 |
| 103 | `internal/app/gaea_schedule.go:578` | SetKeepWarm / SetPreloadPlan | D4 | 12 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/gaea_schedule.go:578`, `internal/app/gaea_schedule.go:600` | 抽统一 helper 后删除 1 份实现 |
| 104 | `internal/app/gaea_ui.go:245` | loadPinned / loadSessionTitles | D4 | 12 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/gaea_ui.go:245`, `internal/app/gaea_ui.go:736` | 抽统一 helper 后删除 1 份实现 |
| 105 | `internal/app/character_states_section.go:152` | imageEngineDisplayName / survivalEmoji | D4 | 11 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/character_states_section.go:152`, `internal/app/image_backend_resolve.go:104` | 抽统一 helper 后删除 1 份实现 |
| 106 | `internal/app/gaea_prompt_store.go:49` | loadPromptOverrides / loadTaskInbox | D4 | 11 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/gaea_prompt_store.go:49`, `internal/app/gaea_task_inbox.go:106` | 抽统一 helper 后删除 1 份实现 |
| 107 | `internal/app/gaea_knowledge_meta.go:263` | sortedMemoryTags / sortedTagSet | D4 | 10 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/gaea_knowledge_meta.go:263`, `internal/app/gaea_memory_meta.go:119` | 抽统一 helper 后删除 1 份实现 |
| 108 | `internal/app/gaea_prompt_store.go:63` | savePromptOverrides / saveTaskInbox | D4 | 10 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/gaea_prompt_store.go:63`, `internal/app/gaea_task_inbox.go:123` | 抽统一 helper 后删除 1 份实现 |
| 109 | `internal/app/shelf.go:197` | setBool / setInt | D4 | 10 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/shelf.go:197`, `internal/app/shelf.go:219` | 抽统一 helper 后删除 1 份实现 |
| 110 | `internal/app/board_manifests.go:23` | boardIDsFromManifests / boardLabelsFromManifests | D4 | 9 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/board_manifests.go:23`, `internal/app/board_manifests.go:35` | 抽统一 helper 后删除 1 份实现 |
| 111 | `internal/app/gaea_ui.go:223` | loadPinnedAll / loadSessionTitlesAll | D4 | 9 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/gaea_ui.go:223`, `internal/app/gaea_ui.go:234` | 抽统一 helper 后删除 1 份实现 |
| 112 | `internal/app/novel_booksource_handler.go:370` | NovelBookSourceSearch / SinBookSourceSearch | D4 | 9 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/novel_booksource_handler.go:370`, `internal/app/sin_booksource_handler.go:286` | 抽统一 helper 后删除 1 份实现 |
| 113 | `internal/app/sin_handler.go:160` | SinTopicClear / SinTopicDelete | D4 | 9 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/sin_handler.go:160`, `internal/app/sin_handler.go:172` | 抽统一 helper 后删除 1 份实现 |
| 114 | `internal/app/whisper_handler.go:427` | WhisperSetEngine / WhisperSetImageModel | D4 | 10 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/whisper_handler.go:427`, `internal/app/whisper_handler.go:470` | 抽统一 helper 后删除 1 份实现 |
| 115 | `internal/app/brain_store.go:164` | containsAny / matchAny | D4 | 8 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/brain_store.go:164`, `internal/app/main_brain.go:38` | 抽统一 helper 后删除 1 份实现 |
| 116 | `internal/app/characterlib_gen_handler.go:406` | containsKey / visionHasField | D4 | 8 | high | 归一化函数体逐字节相同（去注释/字符串/标识符/数字后 md5 一致），2 份：`internal/app/characterlib_gen_handler.go:406`, `internal/app/gaea_cost_import_vision.go:263` | 抽统一 helper 后删除 1 份实现 |
| 117 | `internal/app/board_manifests.go:16` | `GetBoardManifests` | D5 | 3 | medium | 函数体等价于单一 `return board.BuiltinManifests()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 118 | `internal/app/board_manifests.go:49` | `boardIDList` | D5 | 3 | medium | 函数体等价于单一 `return boardIDsFromManifests(a.GetBoardManifests())`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 119 | `internal/app/board_manifests.go:54` | `boardLabelList` | D5 | 3 | medium | 函数体等价于单一 `return boardLabelsFromManifests(a.GetBoardManifests())`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 120 | `internal/app/characterlib_gen_handler.go:161` | `CharacterGenerateFill` | D5 | 3 | medium | 函数体等价于单一 `return a.characterGenerate(chJSON, "fill", nil)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 121 | `internal/app/characterlib_gen_handler.go:497` | `CharacterGeneratePortrait` | D5 | 3 | medium | 函数体等价于单一 `return a.characterGeneratePortrait(chJSON, model, "")`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 122 | `internal/app/characterlib_gen_handler.go:505` | `CharacterGeneratePortraitWithRef` | D5 | 3 | medium | 函数体等价于单一 `return a.characterGeneratePortrait(chJSON, model, strings.TrimSpace(re`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 123 | `internal/app/create_chapter_handler.go:31` | `CreateChapter` | D5 | 3 | medium | 函数体等价于单一 `return a.createChapter(setting, prevSummary, plotReq, chapterNum, bran`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 124 | `internal/app/create_chapter_handler.go:38` | `CreateChapterWithOverride` | D5 | 3 | medium | 函数体等价于单一 `return a.createChapter(setting, prevSummary, plotReq, chapterNum, bran`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 125 | `internal/app/feature_model_handler.go:68` | `featureModel` | D5 | 3 | medium | 函数体等价于单一 `return c.cfg.GetFeatureModel(feature)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 126 | `internal/app/feature_model_handler.go:169` | `GetFeatureModelEnabled` | D5 | 3 | medium | 函数体等价于单一 `return c.cfg.GetFeatureModelEnabled(feature)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 127 | `internal/app/gaea_browser.go:26` | `GaeaBrowserObserve` | D5 | 3 | medium | 函数体等价于单一 `return browserObserve(context.Background())`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 128 | `internal/app/gaea_cost_inquiry.go:35` | `GaeaCostInquirySave` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostInquiryStore().Save(r)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 129 | `internal/app/gaea_cost_inquiry.go:40` | `GaeaCostInquiryList` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostInquiryStore().List(query, limit)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 130 | `internal/app/gaea_cost_inquiry.go:45` | `GaeaCostInquiryDelete` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostInquiryStore().Delete(id)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 131 | `internal/app/gaea_cost_inquiry.go:50` | `GaeaCostInquiryExpiring` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostInquiryStore().ListExpiring(days)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 132 | `internal/app/gaea_cost_inquiry.go:70` | `GaeaCostInquiryScan` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostInquiryStore().ScanAnomalies()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 133 | `internal/app/gaea_cost_projects.go:45` | `GaeaCostProjectSave` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostProjectStore().SaveProject(p)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 134 | `internal/app/gaea_cost_projects.go:50` | `GaeaCostProjectList` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostProjectStore().ListProjects()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 135 | `internal/app/gaea_cost_projects.go:64` | `GaeaCostProjectDelete` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostProjectStore().DeleteProject(id)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 136 | `internal/app/gaea_cost_projects.go:69` | `GaeaCostEstimateItemSave` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostProjectStore().SaveItem(i)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 137 | `internal/app/gaea_cost_projects.go:74` | `GaeaCostEstimateItemDelete` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostProjectStore().DeleteItem(id)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 138 | `internal/app/gaea_cost_projects.go:79` | `GaeaCostEstimateItems` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostProjectStore().ListItems(projectID)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 139 | `internal/app/gaea_cost_projects.go:101` | `GaeaCostEstimateVersions` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostProjectStore().ListVersions(projectID)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 140 | `internal/app/gaea_cost_ref.go:82` | `GaeaCostNoteSave` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostRefStore().Save(n)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 141 | `internal/app/gaea_cost_ref.go:87` | `GaeaCostNoteList` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostRefStore().List(query, status)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 142 | `internal/app/gaea_cost_ref.go:92` | `GaeaCostNoteDelete` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostRefStore().Delete(id)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 143 | `internal/app/gaea_cost_ref.go:97` | `GaeaCostNoteBumpRef` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostRefStore().BumpRef(id)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 144 | `internal/app/gaea_cost_stage.go:35` | `GaeaCostStageSave` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostStageStore().SaveStage(v)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 145 | `internal/app/gaea_cost_stage.go:40` | `GaeaCostStages` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostStageStore().ListStages(projectID)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 146 | `internal/app/gaea_cost_stage.go:45` | `GaeaCostStageCompare` | D5 | 3 | medium | 函数体等价于单一 `return coststage.ComputeComparison(a.hubCostStageStore().ListStages(pr`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 147 | `internal/app/gaea_cost_stage.go:50` | `GaeaCostStageDeviations` | D5 | 3 | medium | 函数体等价于单一 `return coststage.ExtractDeviations(coststage.ComputeComparison(a.hubCo`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 148 | `internal/app/gaea_dag.go:145` | `dagStore` | D5 | 3 | medium | 函数体等价于单一 `return dag.NewStore(filepath.Join(gaeaCwd(), ".gaea", "work", "dag"))`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 149 | `internal/app/gaea_dag.go:480` | `dagTplStore` | D5 | 3 | medium | 函数体等价于单一 `return dag.NewTemplateStore(filepath.Join(gaeaCwd(), ".gaea", "work", `（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 150 | `internal/app/gaea_dag.go:503` | `GaeaDagTemplateList` | D5 | 3 | medium | 函数体等价于单一 `return a.dagTplStore().List()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 151 | `internal/app/gaea_data_backup.go:198` | `GaeaDataBackupCancel` | D5 | 3 | medium | 函数体等价于单一 `return gaeaBackup.ClearPending(config.DataRoot())`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 152 | `internal/app/gaea_data_backup.go:204` | `GaeaDataBackupRollback` | D5 | 3 | medium | 函数体等价于单一 `return gaeaBackup.RollbackBefore(config.DataRoot())`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 153 | `internal/app/gaea_factbase.go:204` | `GaeaFactBase` | D5 | 3 | medium | 函数体等价于单一 `return factBaseSnapshot()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 154 | `internal/app/gaea_file_index.go:89` | `GaeaFileSemanticSearch` | D5 | 3 | medium | 函数体等价于单一 `return a.fileSemanticHits(query, topN)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 155 | `internal/app/gaea_handler.go:551` | `GaeaSetEngine` | D5 | 3 | medium | 函数体等价于单一 `return a.SetActiveEngine(engineID)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 156 | `internal/app/gaea_herdsman_probe.go:18` | `HerdsmanProbe` | D5 | 3 | medium | 函数体等价于单一 `return herdsman.NewProbe("", "").Run()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 157 | `internal/app/gaea_herdsman_sec.go:20` | `HerdsmanSecurityCheck` | D5 | 3 | medium | 函数体等价于单一 `return herdsman.CheckLanExposure(herdsmanConfigPath())`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 158 | `internal/app/gaea_knowledge_import.go:50` | `hubKnowledgeStore` | D5 | 3 | medium | 函数体等价于单一 `return knowledge.Global().Store()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 159 | `internal/app/gaea_pinned.go:36` | `GaeaPinnedMaterials` | D5 | 3 | medium | 函数体等价于单一 `return pinnedView(gaeaCwd())`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 160 | `internal/app/gaea_price_sources.go:19` | `hubPriceStore` | D5 | 3 | medium | 函数体等价于单一 `return pricefeed.Open(db.GetDatabase(config.MemoryUserDir()))`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 161 | `internal/app/gaea_price_sources.go:69` | `GaeaPriceSources` | D5 | 3 | medium | 函数体等价于单一 `return a.hubPriceStore().ListSources()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 162 | `internal/app/gaea_price_sources.go:74` | `GaeaPriceSourceSave` | D5 | 3 | medium | 函数体等价于单一 `return a.hubPriceStore().SaveSource(src)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 163 | `internal/app/gaea_price_sources.go:79` | `GaeaPriceSourceDelete` | D5 | 3 | medium | 函数体等价于单一 `return a.hubPriceStore().DeleteSource(id)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 164 | `internal/app/gaea_price_sources.go:120` | `GaeaPriceFetches` | D5 | 3 | medium | 函数体等价于单一 `return a.hubPriceStore().ListFetches(30)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 165 | `internal/app/gaea_price_sources.go:195` | `GaeaPriceFetchIgnore` | D5 | 3 | medium | 函数体等价于单一 `return a.hubPriceStore().SetFetchStatus(fetchID, "ignored")`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 166 | `internal/app/gaea_price_sources.go:200` | `GaeaPriceHistory` | D5 | 3 | medium | 函数体等价于单一 `return a.hubPriceStore().ListHistory(name, 30)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 167 | `internal/app/gaea_semantic_search.go:63` | `GaeaSemanticSearch` | D5 | 3 | medium | 函数体等价于单一 `return a.semanticSearchHitsOnDemand(query, 20)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 168 | `internal/app/gaea_spaces.go:121` | `GaeaSpaceActive` | D5 | 3 | medium | 函数体等价于单一 `return gaeaSpaceActiveView()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 169 | `internal/app/gaea_templates.go:88` | `GaeaTaskTemplates` | D5 | 3 | medium | 函数体等价于单一 `return renderTaskTemplates(gaeaEffectiveSpace())`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 170 | `internal/app/gaea_translate.go:139` | `LocalTranslate` | D5 | 3 | medium | 函数体等价于单一 `return a.localTranslate(a.backgroundCtx(), req)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 171 | `internal/app/gaea_ui_extra.go:233` | `GaeaDeleteProvider` | D5 | 3 | medium | 函数体等价于单一 `return fmt.Errorf("模型中心引擎 %s 为内置固定引擎，不支持删除", name)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 172 | `internal/app/gaea_ui_extra.go:436` | `GaeaListDir` | D5 | 3 | medium | 函数体等价于单一 `return listDirEntries(rel)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 173 | `internal/app/gaea_ui_extra.go:1177` | `GaeaApplyUpdate` | D5 | 3 | medium | 函数体等价于单一 `return errors.New("桌面版无自动更新机制，请从发布渠道获取新版本")`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 174 | `internal/app/gaea_ui_extra.go:1180` | `GaeaOpenDownloadPage` | D5 | 3 | medium | 函数体等价于单一 `return errors.New("桌面版无自动更新机制，请从发布渠道获取新版本")`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 175 | `internal/app/gaea_unified_search.go:53` | `semanticSearchHits` | D5 | 3 | medium | 函数体等价于单一 `return a.semanticSearchHitsOnDemand(query, topN)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 176 | `internal/app/gaea_workspace_search.go:21` | `GaeaWorkspaceSearch` | D5 | 3 | medium | 函数体等价于单一 `return a.workspaceSearchHits(query, limit)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 177 | `internal/app/graph_handler.go:70` | `ParseLinks` | D5 | 3 | medium | 函数体等价于单一 `return graph.ParseLinks(content)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 178 | `internal/app/graph_handler.go:75` | `FindUnlinkedMentions` | D5 | 3 | medium | 函数体等价于单一 `return graph.FindUnlinkedMentions(content, entityNames)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 179 | `internal/app/imagehub_usage.go:64` | `ImageHubMonthlyUsage` | D5 | 3 | medium | 函数体等价于单一 `return imageHubMonthlyUsageAt(gaeaCwd(), space)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 180 | `internal/app/image_domain.go:329` | `ImageHubAssets` | D5 | 3 | medium | 函数体等价于单一 `return imageHubAssetSummaries(gaeaCwd(), space, sourceBoard, "", limit`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 181 | `internal/app/image_handler.go:220` | `GenerateFreeImage` | D5 | 3 | medium | 函数体等价于单一 `return a.generateFreeImageProvenanced(prompt, negative, size, style, m`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 182 | `internal/app/image_handler.go:783` | `saveImageToDisk` | D5 | 3 | medium | 函数体等价于单一 `return a.saveMediaToDisk(imageData, prompt, a.cfg.ImageSaveDir)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 183 | `internal/app/intent_router.go:67` | `routeIntentWithResult` | D5 | 3 | medium | 函数体等价于单一 `return a.routeIntentWithResultForAssistant(text, "")`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 184 | `internal/app/intent_router.go:74` | `routeIntentWithResultForAssistant` | D5 | 3 | medium | 函数体等价于单一 `return a.routeIntentModeForAssistant(text, false, assistantID)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 185 | `internal/app/intent_router.go:82` | `GaeaRouteIntent` | D5 | 3 | medium | 函数体等价于单一 `return a.routeIntentMode(text, dryRun)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 186 | `internal/app/intent_router.go:90` | `routeIntentMode` | D5 | 3 | medium | 函数体等价于单一 `return a.routeIntentModeForAssistant(text, dryRun, "")`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 187 | `internal/app/memory_hub.go:159` | `GaeaProfileDelete` | D5 | 3 | medium | 函数体等价于单一 `return a.hubProfileStore().Delete(name)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 188 | `internal/app/memory_hub.go:771` | `GaeaCostDelete` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostStore().Delete(name)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 189 | `internal/app/memory_hub.go:786` | `GaeaCostCategorySave` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostStore().SaveCategory(parentID, name, sort, id)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 190 | `internal/app/memory_hub.go:791` | `GaeaCostCategoryDelete` | D5 | 3 | medium | 函数体等价于单一 `return a.hubCostStore().DeleteCategory(id)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 191 | `internal/app/model_router.go:139` | `GetOfficeLocal` | D5 | 3 | medium | 函数体等价于单一 `return a.cfg.GetOfficeLocal()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 192 | `internal/app/model_router.go:156` | `GetSensitiveLocal` | D5 | 3 | medium | 函数体等价于单一 `return a.cfg.GetSensitiveLocal()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 193 | `internal/app/model_router.go:162` | `GetOfflineMode` | D5 | 3 | medium | 函数体等价于单一 `return a.cfg.GetOfflineMode()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 194 | `internal/app/model_router.go:180` | `GetEngineFailover` | D5 | 3 | medium | 函数体等价于单一 `return a.cfg.GetEngineFailover()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 195 | `internal/app/novel_import_handler.go:42` | `ImportNovelBook` | D5 | 3 | medium | 函数体等价于单一 `return a.ImportNovelBookEx(filePath, title, genre, style, string(booki`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 196 | `internal/app/office_handler.go:12` | `OfficeExecute` | D5 | 3 | medium | 函数体等价于单一 `return officecore.Execute(officecore.DesktopAgentAction(act), path, tg`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 197 | `internal/app/office_handler.go:15` | `OfficeIsTask` | D5 | 3 | medium | 函数体等价于单一 `return officecore.IsTask(text)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 198 | `internal/app/office_handler.go:22` | `OfficeReadFile` | D5 | 3 | medium | 函数体等价于单一 `return a.OfficeExecute("read_text", p, "", "", "", "")`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 199 | `internal/app/office_handler.go:25` | `OfficeGetJobState` | D5 | 3 | medium | 函数体等价于单一 `return jm.GetState(s)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 200 | `internal/app/office_handler.go:27` | `OfficeGetMode` | D5 | 3 | medium | 函数体等价于单一 `return sm.GetMode(s)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 201 | `internal/app/office_skill_capture.go:87` | `saveSkillFile` | D5 | 3 | medium | 函数体等价于单一 `return a.saveSkillFileContent(name, desc, skill.RenderSkillFile(name, `（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 202 | `internal/app/project_handler.go:66` | `CloseProject` | D5 | 3 | medium | 函数体等价于单一 `return a.closePM()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 203 | `internal/app/sin_booksource_handler.go:340` | `SinBookSourceBooksList` | D5 | 3 | medium | 函数体等价于单一 `return sinBooksList(sinBooksDir())`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 204 | `internal/app/sin_booksource_handler.go:345` | `SinBookSourceBookDelete` | D5 | 3 | medium | 函数体等价于单一 `return sinBookDeleteAt(sinBooksDir(), path)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 205 | `internal/app/sin_booksource_handler.go:350` | `SinBookSourceBookExportEpub` | D5 | 3 | medium | 函数体等价于单一 `return sinBookExportEpubAt(sinBooksDir(), path)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 206 | `internal/app/tts_handler.go:247` | `GaeaTTSVoiceParams` | D5 | 3 | medium | 函数体等价于单一 `return voice.GetEmotionVoiceParams(emotion)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 207 | `internal/app/tts_service.go:159` | `StartLocalTTSService` | D5 | 3 | medium | 函数体等价于单一 `return a.ensureLocalTTSService(engineID)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 208 | `internal/app/weixin_reminder.go:54` | `weixinTaskCfgPath` | D5 | 3 | medium | 函数体等价于单一 `return filepath.Join(a.whisperDataRoot, "weixin_task.json")`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 209 | `internal/app/weixin_reminder.go:303` | `remindersPath` | D5 | 3 | medium | 函数体等价于单一 `return filepath.Join(a.whisperDataRoot, "weixin_reminders.json")`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 210 | `internal/app/weixin_reminder.go:564` | `WeixinReminderSetConfig` | D5 | 3 | medium | 函数体等价于单一 `return a.setWeixinTaskCfg(cfgJSON)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 211 | `internal/app/whisper_handler.go:150` | `WhisperChat` | D5 | 3 | medium | 函数体等价于单一 `return a.whisperChat(userMsg, personalityID, "", thinking)`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 212 | `internal/app/whisper_handler.go:501` | `WhisperGetEngines` | D5 | 3 | medium | 函数体等价于单一 `return a.GetEngines()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 213 | `internal/app/whisper_handler.go:533` | `WhisperChatWithSearch` | D5 | 3 | medium | 函数体等价于单一 `return a.whisperChatWithSearch(userMsg, personalityID, "", thinking, f`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 214 | `internal/app/whisper_handler.go:560` | `whisperChatAsAssistant` | D5 | 3 | medium | 函数体等价于单一 `return a.whisperChatWithSearch(userMsg, personalityID, assistantName, `（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 215 | `internal/app/whisper_write_observability.go:53` | `whisperWriteErrorStats` | D5 | 3 | medium | 函数体等价于单一 `return a.writeErrors.stats()`（等价单行转发）；绑定面需要该名字，故调用点存在 | 保留名字，实现可下沉/合并 |
| 216 | `internal/app/intent_router.go:262` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 217 | `internal/app/intent_router.go:308` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return "" 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 218 | `internal/app/intent_router.go:338` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return "" 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 219 | `internal/app/intent_router.go:358` | (匿名错误分支) | D2 | 3 | medium | `if ierr != nil { 				return nil 			}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 220 | `internal/app/characterlib_handler.go:212` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 221 | `internal/app/characterlib_handler.go:461` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 222 | `internal/app/gaea_dag.go:817` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 223 | `internal/app/gaea_dag.go:821` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 224 | `internal/app/novel_booksource_handler.go:208` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return "" 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 225 | `internal/app/novel_booksource_handler.go:281` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return "" 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 226 | `internal/app/whisper_handler.go:104` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 			return nil 		}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 227 | `internal/app/whisper_handler.go:366` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 228 | `internal/app/consistency_deep_handler.go:380` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 229 | `internal/app/create_chapter_context.go:432` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return ""  	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 230 | `internal/app/create_chapter_handler.go:374` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return false 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 231 | `internal/app/gaea_cost_projects.go:57` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 232 | `internal/app/gaea_data_backup.go:293` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 			return nil 		}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 233 | `internal/app/gaea_dream_pending.go:62` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 234 | `internal/app/gaea_git.go:98` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		 		 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 235 | `internal/app/gaea_knowledge_import.go:264` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 236 | `internal/app/gaea_knowledge_meta.go:110` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 237 | `internal/app/gaea_spaces.go:56` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 238 | `internal/app/gaea_ui_extra.go:766` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return "" 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 239 | `internal/app/gaea_ui_meta.go:556` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 240 | `internal/app/gaea_verify.go:25` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 241 | `internal/app/gaea_xlsx_edit.go:188` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return "" 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 242 | `internal/app/imagehub_ledger.go:128` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 243 | `internal/app/image_comfyui_proc.go:321` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return 0 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 244 | `internal/app/image_handler.go:1101` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return false 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 245 | `internal/app/lorebook_handler.go:19` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 246 | `internal/app/memory_hub.go:270` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return 0 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 247 | `internal/app/scene_cards_handler.go:83` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 248 | `internal/app/story_memory_recall.go:151` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return nil 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 249 | `internal/app/tts_service.go:61` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return false 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 250 | `internal/app/worldview_handler.go:107` | (匿名错误分支) | D2 | 3 | medium | `if err != nil { 		return "" 	}` —— err 被丢弃且无日志 | 补 `slog.Warn` 或加 `// best-effort` 注释说明为何可忽略 |
| 251 | `internal/app/app.go:662` | `Shutdown` | D7 | 1 | medium | 形参 `ctx` 在函数体内零使用（函数 91 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 252 | `internal/app/brain_left.go:36` | `Write` | D7 | 1 | medium | 形参 `value` 在函数体内零使用（函数 3 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 253 | `internal/app/chapter_handler.go:263` | `sceneIllustrationRefs` | D7 | 1 | medium | 形参 `pm` 在函数体内零使用（函数 52 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 254 | `internal/app/create_chapter_handler.go:478` | `streamCreateChapter` | D7 | 1 | medium | 形参 `of` 在函数体内零使用（函数 286 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 255 | `internal/app/create_chapter_handler.go:478` | `streamCreateChapter` | D7 | 1 | medium | 形参 `plotReq` 在函数体内零使用（函数 286 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 256 | `internal/app/gaea_dag.go:62` | `Execute` | D7 | 1 | medium | 形参 `ctx` 在函数体内零使用（函数 65 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 257 | `internal/app/gaea_specialist_tools.go:58` | `Execute` | D7 | 1 | medium | 形参 `ctx` 在函数体内零使用（函数 20 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 258 | `internal/app/gaea_specialist_tools.go:115` | `Execute` | D7 | 1 | medium | 形参 `ctx` 在函数体内零使用（函数 33 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 259 | `internal/app/gaea_tasks.go:316` | `fileIndexTaskHandler` | D7 | 1 | medium | 形参 `t` 在函数体内零使用（函数 53 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 260 | `internal/app/gaea_ui.go:725` | `GaeaSummarizeFrom` | D7 | 1 | medium | 形参 `turn` 在函数体内零使用（函数 1 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 261 | `internal/app/gaea_ui.go:726` | `GaeaSummarizeUpTo` | D7 | 1 | medium | 形参 `turn` 在函数体内零使用（函数 1 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 262 | `internal/app/gaea_ui_extra.go:1191` | `GaeaSaveWindowState` | D7 | 1 | medium | 形参 `state` 在函数体内零使用（函数 1 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 263 | `internal/app/image_comfyui_proc.go:35` | `comfyProcRefClearIfCurrent` | D7 | 1 | medium | 形参 `cancel` 在函数体内零使用（函数 7 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 264 | `internal/app/intent_router.go:149` | `intentPreviewForAssistant` | D7 | 1 | medium | 形参 `assistantID` 在函数体内零使用（函数 56 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 265 | `internal/app/intent_router.go:427` | `execStatus` | D7 | 1 | medium | 形参 `it` 在函数体内零使用（函数 26 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 266 | `internal/app/novel_import_task.go:28` | `outlineReconstructTaskHandler` | D7 | 1 | medium | 形参 `tk` 在函数体内零使用（函数 16 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 267 | `internal/app/novel_llm_rewrite_handler.go:199` | `llmRewriteSentences` | D7 | 1 | medium | 形参 `chapterNum` 在函数体内零使用（函数 38 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 268 | `internal/app/novel_llm_rewrite_handler.go:238` | `pickFlaggedSentences` | D7 | 1 | medium | 形参 `content` 在函数体内零使用（函数 23 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 269 | `internal/app/sin_refs.go:47` | `sinRefPlan` | D7 | 1 | medium | 形参 `model` 在函数体内零使用（函数 12 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 270 | `internal/app/tts_handler.go:27` | `SaveTTSConfig` | D7 | 1 | medium | 形参 `modelPath` 在函数体内零使用（函数 3 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 271 | `internal/app/tts_handler.go:27` | `SaveTTSConfig` | D7 | 1 | medium | 形参 `serverPath` 在函数体内零使用（函数 3 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 272 | `internal/app/tts_handler.go:27` | `SaveTTSConfig` | D7 | 1 | medium | 形参 `port` 在函数体内零使用（函数 3 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 273 | `internal/app/tts_handler.go:27` | `SaveTTSConfig` | D7 | 1 | medium | 形参 `backend` 在函数体内零使用（函数 3 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 274 | `internal/app/tts_handler.go:27` | `SaveTTSConfig` | D7 | 1 | medium | 形参 `speed` 在函数体内零使用（函数 3 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 275 | `internal/app/tts_handler.go:40` | `StartTTSServer` | D7 | 1 | medium | 形参 `modelPath` 在函数体内零使用（函数 3 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 276 | `internal/app/tts_handler.go:40` | `StartTTSServer` | D7 | 1 | medium | 形参 `port` 在函数体内零使用（函数 3 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 277 | `internal/app/tts_handler.go:40` | `StartTTSServer` | D7 | 1 | medium | 形参 `backend` 在函数体内零使用（函数 3 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 278 | `internal/app/tts_handler.go:50` | `TTSSpeak` | D7 | 1 | medium | 形参 `text` 在函数体内零使用（函数 3 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 279 | `internal/app/tts_service.go:58` | `ttsReady` | D7 | 1 | medium | 形参 `engineID` 在函数体内零使用（函数 9 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 280 | `internal/app/wx_agent.go:323` | `wxAgentExecTool` | D7 | 1 | medium | 形参 `assistantID` 在函数体内零使用（函数 78 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |
| 281 | `internal/app/wx_agent.go:417` | `wxAgentIntentReminder` | D7 | 1 | medium | 形参 `text` 在函数体内零使用（函数 6 行）；已排除接口实现必须的 `ctx` 之外的误判 | 改名 `_` 或删除（**绑定方法签名不能改**，只能改名/加注释） |

（全量表共 **281** 条。）

## 统一 helper 收敛清单（D4 主战场，按收益排序）

| # | 提案 helper | 收敛目标 | 可收敛行数 | 风险 |
| --- | --- | --- | --- | --- |
| 1 | `toolSchema(fields []schemaField) json.RawMessage` + `toolBase`（提供 `ReadOnly()`/`CompactSchema()`/`CompactDescription()`） | 10 份 `Schema()` + `CompactSchema`/`CompactDescription`/`ReadOnly` 各 8~17 份样板（`gaea_dag.go`/`gaea_diagram.go`/`gaea_factbase.go`/`gaea_routine_llm.go`/`gaea_specialist_tools.go`/`gaea_tools.go`/`gaea_translate.go`/`sin_tool_*.go`） | ≈150~180 | 低（纯内部结构，工具接口不变） |
| 2 | `setProviderKey(provider, key string) error` | `model_engine_handler.go:222/252/292/320/352` 5 份 17 行实现 | ≈68 | 低（导出名保留，逐个转 3 行包装） |
| 3 | `hubStore[T](override *T, open func() T) T` 泛型访问器 + `hubCostStageStore` 等 4 份同构实现 | `gaea_cost_inquiry.go:23`、`gaea_cost_projects.go:33`、`gaea_cost_ref.go:30`、`gaea_cost_stage.go:23` | ≈30 | 低 |
| 4 | `ensureJSONConfigFile(path string, def any, migrate func([]byte) (any, error)) error` | `ensureCostCheckParams`、`ensureNovelStyleWords`、`ensureNovelStylePatterns`、`ensureNovelReviewRubric` 4 份 11 行同构实现 | ≈33 | 低 |
| 5 | `startCron(ctx, name, interval, fn)` | `startFileIndexCron`(gaea_file_index.go:48) 与 `startPriceCron`(gaea_price_sources.go:26) 逐字节相同 | ≈24 | 低 |
| 6 | `herdsmanModelAction(action string)` 表驱动 | `herdsman_lifecycle.go:307/321/335/349` 4 份 12 行同构实现 | ≈36 | 低 |
| 7 | `saveJSONEntity(path string, v any) error` | `SaveOrganization`/`SaveRelationship`/`SaveOutlineNode`/`AddOutlineNode`（各 10 行同构）；以及 `savePromptOverrides`/`saveTaskInbox`、`loadPromptOverrides`/`loadTaskInbox` 两对 | ≈70 | 低 |
| 8 | `keyedConfigSetter` 表驱动（`model_router.go:145/168/186/198` 4 份 9 行；`gaea_schedule.go:578/600` 2 份 12 行；`gaea_handler.go:222/252` 2 份 16~18 行） | 8 份同构 setter | ≈90 | 低 |
| 9 | `containsStr(xs []string, v string) bool` / `keyExists(m map[string]any, k string) bool` | `matchAny`(brain_store.go:164)↔`containsAny`(main_brain.go:38)；`containsKey`(characterlib_gen_handler.go:406)↔`visionHasField`(gaea_cost_import_vision.go:263) | ≈16 | 低（注意语义是否真等价，需读实现） |
| 10 | 统一 JSON 入参解码 `decodeArgs[T any](payload string) (T, error)` | 全目录 199 处 `json.Unmarshal`、139 处 `json.Marshal`、25 处 `json.NewDecoder`，其中大量是「字符串参数 → 结构体 → 校验 → 错误包装」的同构三段式 | 需先抽样评估（估 ≈200~400 可收敛，但改动面大） | 中（错误文案会影响前端提示与测试断言） |
| 11 | 统一 SSE/流式转发 `streamEmit(ch <-chan provider.Chunk, eventPrefix string, sink func(...) )` | `chat_service.go:164 runChatStreamPlain`、`sin_handler.go:382 runSinRounds`、`create_chapter_handler.go:478 streamCreateChapter`、`gaea_dag.go:567 dagExecute` 四处各自实现分块→事件转发 | ≈120 | 中（事件名必须逐字保持：`chat-stream:`/`sin-stream:`/`create-chapter-stream:`/`gaea-event`） |
| 12 | 统一「编辑前备份 + 写回滚文件」`withRollback(path string, write func([]byte) error) error` | `gaea_docx_edit.go:127`、`gaea_pptx_edit.go:42`、`gaea_xlsx_edit.go`（三处 6 行窗口逐行相同） | ≈30 | 低 |

事件名硬编码重复（D4 面状）：`a.emit("...")` 共 28 个不同事件名、91 个发射点，其中重复 >2 次的有 `sin-stream:`×16、`create-chapter-stream:`×15、`gaea-event:`×10、`chat-stream:`×7、`tts-stream:`×5、`tts-service-status:`×4、`voice-model-changed:`×4、`xai-output:`×3、`xai-login-failed:`×3。建议把事件名收进 `internal/app/events.go` 常量表（`const EventChatStream = "chat-stream:"`），前端 `wailsEvents.ts` 也按同名清单对齐——事件名是前后端字符串契约，**改名等于破坏契约**，只能抽常量不能改字面量。

## 屎山清单（结构性重构建议，含风险）

先纠正一个常见预设：本域**没有** >2000 行的文件（最大 1191 行），也**没有** >300 行的函数（最大 286 行）。真实的结构性负担是「少数巨文件 + 同构样板大量重复 + 99 个绑定面单行转发」，即**广度重复**而非**深度嵌套**。

### 1. 巨文件（单文件多主题混杂，优先级最高）

| 文件 | 行数 | 承载主题（证据见文件内注释小节与函数名） | 拆分建议 | 风险 |
| --- | --- | --- | --- | --- |
| `internal/app/gaea_ui_extra.go` | 1191 | ① 窗口/更新/下载页（`GaeaApplyUpdate`:1177、`GaeaOpenDownloadPage`:1180、`GaeaSaveWindowState`:1191）② 目录/文件浏览（`GaeaListDir`:436、`GaeaFileSearch`:465）③ 附件/附件数据 URL（`GaeaAttachmentDataURL`:980）④ 模型中心引擎增删（`GaeaDeleteProvider`:233）⑤ 权限等级（`GaeaPermLevel`:399） | 拆 `ui_window.go` / `ui_files.go` / `ui_attachment.go` / `model_provider_crud.go`；其中 ④⑤ 本次已判死代码，直接删更划算 | 低（导出名不变，纯搬文件） |
| `internal/app/image_handler.go` | 1161 | ① 生图主流程（`GenerateMedia`:537、`runImageGenLoop`:321）② ComfyUI 进程（`StartComfyUI` 在 image_comfyui_proc.go）③ 目录/扩展名工具（`mediaExt`:731、`saveImageToDisk`:783）④ 配置读写（`SetPortraitConfig`:852、`SetSinImageConfig`:877、`GetImageBackendConfig`:975→死） | 拆 `image_gen.go` / `image_config.go` / `image_paths.go` | 低 |
| `internal/app/whisper_handler.go` | 1030 | ① whisper 引擎/模型选择（`WhisperSetEngine`:427 等 7 个→死）② 对话主流程（`whisperChat`:159、`WhisperChat`:150、`WhisperChatWithSearch`:533）③ 状态持久化（`persistWhisperState`:868） | 拆 `whisper_chat.go` / `whisper_model.go`；②③ 里 144 行的 `whisperChat` 需要再拆子函数 | 中（`whisperChat` 被 3 条绑定路径共用） |
| `internal/app/create_chapter_handler.go` | 1019 | ① 建章入口（`CreateChapter`:31 / `CreateChapterWithOverride`:38 → `createChapter`:44，198 行）② 流式建章（`streamCreateChapter`:478，**286 行、嵌套 9 层，本域最长**）③ 事件/去重键（`emitXaiOutput`、`chapterGenKey`:244） | 优先拆 `streamCreateChapter`：把「取上下文 → 调模型流 → 逐块 emit → 落库 → 收尾」五段提成独立函数（每段 40~60 行，嵌套可降到 3~4）；再按职责拆文件 | 中高（流式协议 + 事件名 `create-chapter-stream:` 是前端契约，只能搬不能改） |
| `internal/app/sin_handler.go` | 938 | ① 原罪对话轮次（`runSinRounds`:382、`persistSinTurn`:491）② 主题管理（`SinTopicDelete`:160 / `SinTopicClear`:172 为逐字节重复）③ 插图（`SinIllustrate`:608）④ 流事件（`sin-stream:` 16 处） | 拆 `sin_chat.go` / `sin_topic.go` / `sin_illustrate.go`；抽出统一 `sinEmit(kind, payload)` | 中 |
| `internal/app/gaea_handler.go` | 849 | ① Gaea agent 装配（`gaeaBuildController`:100）② 事件映射（`gaeaEventMap`:124）③ 旧绑定面残骸（`GaeaModel`/`GaeaEngines`/`GaeaSetEngine`/`GaeaTools`/`GaeaSkills`/`GaeaCallTool` ≈42 行，已判死） | 先删死绑定；`gaeaEventMap` 124 行建议改表驱动 | 低 |
| `internal/app/app.go` | 904 | ① App 装配/启动（`New`:329、`Startup`:354 **244 行**、`Shutdown`:662 91 行→死）② core 状态与 emit（`emit`:314）③ FS 注入（`SetDistFS`/`SetPromptFS`） | 把 `Startup` 按「FS/配置 → 存储 → 服务 → cron/watch → 自检」五段提取子函数；`Shutdown`（死代码）删除后文件立减 91 行 | 中（`Startup` 顺序有隐式依赖，拆错会引入竞态） |
| `internal/app/consistency_deep_handler.go` | 865 | ① 一致性深检（`CheckConsistencyDeep`:67）② 状态逐行比对（`deepCompareStateLine`:665，**191 行、嵌套 8**）③ 别名解析（`deepAliasResolver.Resolve`:304） | `deepCompareStateLine` 改表驱动（字段→比较器 map）+ 早返回，可降嵌套到 3；文件按「检查/比较/别名」三分 | 中 |
| `internal/app/gaea_dag.go` | 857 | ① DAG 工具定义（`dagPlanTool.Schema`:44 **嵌套 8 但仅 16 行**，是巨型字面量）② 执行器（`dagExecute`:567 **129 行、嵌套 7**）③ 模板存储（`dagTplStore`:480） | 巨型 schema 字面量移到 `gaea_dag_schema.go`（纯数据）；`dagExecute` 拆「校验 → 拓扑执行 → 汇总」 | 中高（DAG 并发语义，需要测试兜底） |
| `internal/app/memory_hub.go` | 869 | ① profile store（`GaeaProfileDelete`:159）② cost store（`GaeaCostDelete`:771、`GaeaCostCategory*`:786/791）③ 图谱（`GaeaMemoryGraph`:309，170 行）④ 测试 seam（`SetOfficeStoreForTest`:118 等 4 个） | 拆 `memory_hub_profile.go` / `memory_hub_cost.go` / `memory_hub_graph.go`；`GaeaMemoryGraph` 拆「取数 → 构图 → 布局」 | 中 |
| `internal/app/characterlib_gen_handler.go` | 896 | ① 角色生成（`characterGenerate`:291、`CharacterGenerateFill`:161）② 立绘（`characterGeneratePortrait`）③ JSON 规范化样板（`containsKey`:406、`normalizeGenValue`:452） | 与 `characterlib_score.go` 的同构 JSON→Character 解析（`characterlib_gen_handler.go:587/805`）合并到 `characterlib_json.go` | 中 |
| `internal/app/create_chapter_context.go` | 775 | 上下文预算/裁剪/伏笔注入；`buildChapterContextSections` 与 `...Within` 是两层包装 | 与 `create_chapter_handler.go` 同域，建议合并为 `create_chapter/` 子包（见下） | 中 |
| `internal/app/intent_router.go` | 656 | ① 意图路由（`routeIntentMode`:90 及 3 层同构包装 `routeIntentWithResult`→`...ForAssistant`→`routeIntentModeForAssistant`）② 预览（`intentPreviewForAssistant`:149，56 行、11 case）③ 执行状态（`execStatus`:427） | 3 层包装压成 1 层 + 可选参数；`intentPreviewForAssistant` 改表驱动 | 中（路由被 weixin 与桌面端共用） |

### 2. 深嵌套热点（降嵌套的具体手法）

| 文件:行 | 函数 | 嵌套 | 行数 | 降嵌套手法 | 可减行数 |
| --- | --- | --- | --- | --- | --- |
| `create_chapter_handler.go:478` | `streamCreateChapter` | 9 | 286 | 五段提取子函数（`prepareChapterGen` / `runChapterStream` / `persistChapterResult` / `emitChapterDone`），块内提前 `continue`→提取为守卫函数 | 60~90 |
| `consistency_deep_handler.go:665` | `deepCompareStateLine` | 8 | 191 | 字段→比较器 map 表驱动 + 每类比较独立函数；最内层 `if/else if` 链改 `switch` + 早返回 | 50~70 |
| `whisper_state.go:65` | `startAssistantWx` | 8 | 156 | 微信回调三段（解析/投递/回执）提成独立函数；嵌套来自 try-parse-then-act 模式 | 30~40 |
| `gaea_dag.go:567` | `dagExecute` | 7 | 129 | 「校验 DAG → 并行执行节点 → 汇总」三段提取；节点执行体本身已在闭包内，可提出 | 25~35 |
| `gaea_dag.go:44` | `dagPlanTool.Schema` | 8 | 16 | 巨型 JSON 字面量移到独立文件的 `var`/`const`，文件可读性立改 | 0（纯搬迁） |
| `app.go:354` | `Startup` | 6 | 244 | 按「配置/FS → 存储 → 服务 → cron/watch → 自检」提取 5 个子函数；`Shutdown`(91 行) 是死代码可整删 | 91（删 Shutdown）+ 40 |
| `gaea_ui_extra.go:465` | `GaeaFileSearch` | 6 | 52 | 参数校验/过滤/排序三段提取 | 10~15 |

### 3. 绑定面单行转发（D5）

11 个生成门面 `bindings_*.go` 共 1594 行，其中 531 个方法是「一行转发」。**这是生成物（`// Code generated by scripts/gen_bindings; DO NOT EDIT.`），不构成人工债，不要手改**；但生成物之外的 99 个 App/嵌入状态方法同样是单行转发（D5，全量见全量表），其中 `hubCost*Store()` 链路占 28 个，是真正可收敛的对象。

### 4. 被遮蔽的重复实现（App vs 嵌入子状态同名方法）

`scripts/gen_bindings` 的 `shadowBaseline` 只登记了 **2 处**遮蔽（`feature_model_handler.go:79/147` 的 `SetFeatureModel`/`SetFeatureModelEnabled`，core 版本被 App 版本遮蔽）。同类「App 声明同名方法 → 嵌入实现被丢弃但仍编译」的模式在本域另有若干实例（如 `GetFeatureModelEnabled`（`feature_model_handler.go:169` 转发 `cfg`）、`GetImageBackend`（`image_handler.go:798` mediaState 版 vs App 版）、`WhisperChat`（`whisper_handler.go:150` 转发到 `whisperChat`）、`SaveCharacter`/`DeleteCharacter`/`GenerateCharacters`（`character_handler.go` 转发 `internal/character`））。判定「哪份该删」必须逐对读实现，**不建议在本轮删除**，只建议把实测全集同步进 `shadowBaseline` 让漂移可见（当前 `-shadow-check` 只登记 2 处，意味着其余遮蔽处于「无人登记」状态——这是守卫的盲区，而非死代码）。

## 判据的残余风险与「动态调用名」专项排查

本域的前端调用链是**静态**的，这决定了字符串扫描在本域是可信判据，但仍做了专项排查：

| 检查项 | 方法 | 结果 |
| --- | --- | --- |
| 前端是否存在按名字动态派发（`CallByName` / `window[...]` / `WailsInvoke` / `[methodName](...)`） | `grep -rn` over `frontend/src/**/*.ts(x)` | **0 处**。前端只有两条路径：① Wails 生成的静态绑定 `frontend/wailsjs/go/app/*.js`（每个方法一个具名导出函数）；② 手写门面接口 `frontend/src/gaea/lib/bridge/{core,office,memory,cost,model,voice,chat,novel,image,charlib,sin}.ts` + `appBindings.ts`（编译期漂移断言）。两者都要求方法名以**字符串字面量**出现 → 名字出现在 `frontend/src` 即等价于「被调用或被契约锁定」。 |
| 是否存在「Go 侧按名字动态分派」 | 检查 `board/manifest.go`（`Handler` = App 方法名）与 `module_registry.go` | 有，但走**字面量字符串表**（`internal/app/board/builtins.go`、`manifest.go`）。本次 token 统计**包含字符串字面量内部**的词，因此凡被 manifest/registry 登记过的方法都不会被判成 D1 —— 该层已覆盖。 |
| 名字拼接式调用（`` `Gaea${x}` ``） | 人工核查 `bridge/mappings.ts`、`legacyBindings.ts`、`mock/bindingSignatures.ts`、`mock/contract.test.ts` | 生成物是**静态名清单**；`mappings.ts` 的 `gaeaToGaea` 是「前端短名 → Go 全名」的显式映射表（如 `SemanticIndexStatus: "GaeaSemanticIndexStatus"`），键值皆字面量，无运行时拼接。 |
| 残余风险 | — | ① 前端 `MOCK_ONLY_NAMES` / `ARITY_ALLOWANCES` 白名单若含某名字会被算作命中（保守方向：只会**漏报**死代码，不会误报）；② 审计期间工作树里 `frontend/src` 有**他人正在进行的未提交修改**（`git status` 显示约 36 个文件 modified，非本次审计所为），若有人在审计后新增了对某个「判定为死」的名字的调用，结论即失效——落地删除前请重跑一次名字普查。 |

## 不建议动的地方（说明为什么）

1. **`frontend/wailsjs/go/**` 与 `frontend/wailsjs/go/models.ts`** — Wails 自动生成，任务明确排除；任何「方法名只在这里出现」的结论都必须反向视为「该方法仍然可达」。
2. **`internal/app/bindings_*.go` + `bindings_manifest.go` + `bindings_completeness_test.go`** — `// Code generated by scripts/gen_bindings; DO NOT EDIT.`。手改会在下一次生成时被覆盖；要动必须改生成器或 `excludedBindings`。
3. **635 个已在册绑定方法** — 即使某名字在 `frontend/src` 里只出现一次（例如只在 `bindingNames.ts` 清单里），按规则 2 也不算死代码。这批名字是**前后端字符串契约**：前端 `spaceBindings.ts` / `legacyBindings.ts` / `mock/bindingSignatures.ts` / `drift.ts` 与 Go 侧清单三向互锁（有专门的漂移检查 `scripts/check-bindings-drift.ps1`），改名或删除必须同步 4 个前端文件 + 重跑生成器。
4. **`internal/app/board/`** — 只有 manifest/builtins/registry 三个主题，无死代码；`Manifest.Validate`/`ValidateAll`/`BuiltinManifests` 有测试与启动自检双引用。
5. **`SetXxxStoreForTest` / seam 变量（`wxDeliverableSessionPaths` 等）** — 除已点名的 `SetCostStageStoreForTest`/`ResetCostStageStoreForTest` 外，其余 15 个 seam 都有真实测试调用（4~19 次），是测试基础设施，不是死代码。
6. **`Execute(ctx, ...)` 系列工具方法里未使用的 `ctx`** — `dagPlanTool.Execute`、`ocrTool.Execute`、`semanticSearchTool.Execute` 的 `ctx` 参数在内体未使用，但它是 `Tool` 接口实现的一部分，删参数会破坏接口/调用点；同理 `Shutdown(ctx)`。这类 D7 只能加注释，不能删。
7. **`internal/app/tts_handler.go` 的废弃空壳方法** — 虽然按三层检查是死代码，但它们对应前端可能仍挂着的「旧版 TTS 设置」入口；删除前建议真机点一次 TTS 相关面板确认无 404（本次审计**没有运行前端**，无法证伪动态拼接的调用名）。
8. **`if err != nil { return nil }` 全部 35 处不要机械批量改** — 其中相当一部分是刻意的 best-effort（例如 `intent_router.go:260` 是测试可注入 seam，失败即「无候选会话」）。逐条确认语义后再补日志或补 `// best-effort` 注释。
9. **巨型 switch（`parseReminderWhen` 14 case、`herdsmanModelHint` 14 case、`featureModelKeys` 8 case）** — case 数虽多但每支 1~3 行且语义各异，表驱动收益有限而可读性可能下降，不建议为「消除 switch」而改。
10. **`internal/app` 没有 TODO/FIXME（0 处）也几乎没有注释掉的代码（见下）** — 这不是遗漏，是本仓库的既有纪律（`golangci-lint` 启用了 `unused`/`staticcheck`，`unused` 会拦住未使用的**非导出**标识符）。因此 D1 的「私有 helper 未使用」类发现天然接近于零，本报告的 D1 全部落在**导出方法**上（linter 管不到）。这解释了为什么本域 D1 集中在 `excludedBindings` 名单：那批方法体是在「零调用摘除」时按策略保留下来的。

### 附：D3（注释掉的代码块）与 D7（TODO/FIXME）的实测结论

| 检查 | 方法 | 结果 |
| --- | --- | --- |
| 注释掉的代码块 | 去注释后逐行扫描 243 个非测试文件，按「以 Go 关键字开头 / 含 `:=` / 以 `{` `}` `;` 结尾 / 含 `x.y(` 」打分，减去中文散文特征；另查 ≥3 连续行 | **0 处**（仅 3 处提及语法符号的正常文字注释：`app.go:567` 讲 `startFileWatch` 的 go 化、`wx_file_handler.go:6` 讲函数类型签名、`plan_loop_e2e_test.go:8` 的测试命令） |
| TODO/FIXME/XXX/HACK | `grep -rn` 全目录 | **0 处** |
| `_ =` 显式丢弃 | `grep -c` | 406 处 / 160 文件（其中绝大多数是 `_ = os.MkdirAll` 这类有意忽略；未逐条列为发现） |
| `else` 块总数 | `grep -c '} else {'` | 129 处（未见「return 后接 else」的明显反模式，未列为发现） |
