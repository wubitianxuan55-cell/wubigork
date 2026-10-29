# gaea V4 结构性审计（V5 架构立项输入）

- 审计日期：2026-10-29
- 审计对象：C:\AI\wubigrok（Wails v2.13 + Go 1.26 + React 18/TS/Vite，module github.com/gaea/gaea）
- 审计性质：**只读**。未修改任何代码文件；本文件是唯一产出。
- 口径：行号取自 2026-10-29 工作树；`clones/`、`node_modules/`（deepseek-reasonix / deepseek-harness / zcode / reasonix 等外部克隆与依赖）不计入 V4 规模统计；规模用 `Get-Content | Measure-Object -Line` 与 `Select-String` 实测，不引用发布说明。

---

## 一、十條结论

### 结论 1｜绑定面 686 个导出方法，不是文档的 635／654；11 门面是**命名分层**，不是依赖分层
实测 11 个门面文件 `^func \([a-z]+ \*[A-Za-z]+B\) [A-Z]` = **686**，`frontend/wailsjs/go/app/*.d.ts` 的 `export function` 合计同为 **686**（OfficeB 201 / NovelB 129 / CostB 70 / CoreB 53 / MemoryB 51 / ModelB 51 / VoiceB 33 / ImageB 30 / SinB 28 / CharlibB 22 / ChatB 18）。`.gaea/AGENTS.md:159` 仍写「A1① 后 635 个导出方法」，`releases` 的 v4.457.0 说明写「635→654」——两者都落后实际值至少 32／与 686 差 32。更关键的是门面方法体清一色 `return b.a.Xxx(...)` 纯委托，11 个门面共享同一个 `*App`（`bindings_manifest.go:11-23`），所以这层"拆分"只解决命名空间，不解决状态所有权。
→ **对 V5 的含义**：绑定层可原样搬运（零重构风险），但它承载不了 case 维度——每加一个"对案子操作"就是往门面再加方法，686 只会涨到 800。绑定面不是天花板，是**天花板的下游**。

### 结论 2｜「板块＝页面＝功能」在编译期被写死，业务实体无处安放
`frontend/src/gaea/boards/manifests.ts` 的 `canonicalBoards` 13 条，每条 `id` 与 `page` 一一对应（`:80-153`）；`BoardId` 由该数组字面量派生（`:159`），加/删板块是编译期变更；`boards/pageRegistry.ts:12-22` 是单 key→单组件的 Map，一个板块只能挂一个页面组件。`App.tsx` 只有 241 行（主题令牌 + `<MainLayout/>`，`:224-238`），导航/路由事实全在 manifests + MainLayout（723 行）。前端 pages 目录 **161 文件 / 33,478 行**，gaea 域 612 文件 / 104,225 行。
→ **含义**：V4 里"一个业务实体"必须以"一个新板块 + 一个新页面 + 一批新绑定方法"三位一体存在。V5 引入 case（案卷）维度后，这 13 块必须降级为 **lens（同一案卷的不同视图）**，否则每接一个业务域就再切一刀导航。

### 结论 3｜运行时是单 loop：无规划阶段、无 step 持久化、无 resume
`internal/gaea/agent/agent_run.go:18 Run` → `:19 return a.runDirect(...)` 是唯一执行路径（`:22` 注释「original single-model execution path」），主循环 `:126 for step := 0; a.maxSteps <= 0 || step < a.maxSteps || graceRound; step++`——规划与执行在同一轮 LLM 调用里，没有 plan→execute 两段式。中断恢复只有**文本级交接**：`interrupted_recovery.go` 把 `<interrupted-turn-recovery>` 块塞回上下文，进程重启即丢。唯一落盘 seam 是 `agent.go:566 SetCheckpointFlusher` / `:571 FlushCheckpointFailClosed`，语义是"flush 会话快照"，不是可续跑的步骤机。
→ **含义**：V5 的 step／transaction／resume 是**新建**，不是给 V4 打补丁。V4 现有 turn 边界（`defer a.flushJournal()`，`agent_run.go:53`）是唯一天然的事务切分点。

### 结论 4｜session event log 是全仓唯一真正可重放的事实源，V5 最该继承的就是它
`internal/gaea/agent/session/log.go:3-9` 三条硬不变量：seq = 已写行数、payload 必须无损 JSON、"模型可见必入日志"；`LogEntry{Seq,Ts,Kind,Space,Payload}`（`:92-101`）；`request_header` 携带 `compileHash/prevCompileHash/prefixStable`（`:111-123`），即**每次上下文编译都可指纹追溯**。落盘 sink 为 `session/sink.go:17-157`（`EventLogSink`，`boot/boot.go:171-174` 装配为 `event.Sync`）。前端 item 列表**完全由日志折叠重建**：`gaea/lib/store/controller_state.ts:57-109`（id = u/a/sa + logSeq，唯一跨重启稳定身份是日志 seq），`controller.ts:40-90 ensureEventsBound` 做 seq 缺口检测 + `GaeaResyncEvents` 补拉。
→ **含义**：案卷的事实层应当**长在 session log 语义上**（append-only + seq 身份 + 折叠重建），而不是再造一套事件表。这是 V4 给 V5 最大的现成资产。

### 结论 5｜60 张表里没有 case 维度；唯一"决策台账"是死表
`internal/gaea/db/schema.go` 实测 CREATE TABLE 60 张。全仓 `case_id|caseId|CaseID` 命中 **0**。`facts`（`:16`）有 `project` 列 + `UNIQUE(project,name)`；`memory_events`（`:418`）带 `project/space/source_session`；但 `knowledge`（`:44`）与 `cost_entries`（`:67`）是**全局唯一命名空间**（`name UNIQUE`），无 project/case 维。`tasks`（`:197`）的 `session_id` 是 SchemaV20（`:478-479`）后补的。`whisper/db/schema_v6.go:38` 的 `decision_log`（signal_json/decision/reason/tool_decision/user_feedback）**全仓零 Go 标识符引用**——只在 schema 与 `whisper/db/database.go:210` 的 wipe 名单里出现。
→ **含义**：decision ledger 是新建；但 `SchemaV18`（`:411-416`）已经给出 V5 需要的分层范式——`memory_events` 追加式唯一事实源、`mem_graph_*` 是可 DROP 重建的物化投影（"日志即真相"）。case 层照抄这对关系即可。

### 结论 6｜证据卡已存在，但比对粒度是"8KB 截断摘要"而非内容 hash，回滚无版本边界
`evidence.ChangeRecord`（`evidence/journal.go:36-55`）有 `BaselinePath/OpsJSON/BeforeSummary/AfterSummary`，`SummaryLimit = 8KB`（`:22`，`ClampSummary` `:28-33`），落到 `<cwd>/.gaea/work/journal/<sessionKey>.jsonl` 追加 + 回合投影 `exports/journal/<key>/turn-<n>.md`（`:7-8`）。回滚 `internal/app/gaea_verify.go:256-316`：BaselinePath 缺失即拒、用 `ClampSummary` 字符串比对做"已被手工修改"守卫（`:270-288`）、恢复前再快照（`:295-298`）、`fileutil.AtomicWrite` 落盘。`evidence.Receipt/Ledger`（`evidence.go:34-121`）则是**纯内存、不序列化**（`:32-33` 注释），只服务 `complete_step` 校验。
→ **含义**："变更收据"雏形可用，但"案卷可信"要求 hash + 版本 + 事务边界；8KB 截断摘要对大于 8KB 的文档变更会**静默丢失差异**，这是 V5 必须替换的第 1 号存储语义。

### 结论 7｜审批是双轨，且在子代理上系统性退化为"全放行"
纯函数在 `internal/gaea/permission/permission.go`：`Policy.Decide`（`:114-128`，优先级 deny>ask>allow，readOnly 恒 allow）、`Gate.Check`（`:257-299`）、`Approver` 接口（`:228-233`）、`AlwaysAsk` 硬门（`:248`、`:258-270`）。**缺陷**：Approver 为 nil 时 `ask` 直接放行（`:259-261`、`:281-283` "non-interactive: preserve autonomy"）。启动路径 `boot/boot.go:273 headlessGate := permission.NewGate(policy, nil)`，`:262-267` 注释自陈「The headless gate (no Approver) resolves "ask" to allow ... Sub-agents always run headless: they have no UI to answer a prompt, so they inherit this same gate」，`:301-302` 把这个 gate 交给 `NewTaskTool`——**所有子代理（task / DAG 节点 / run_skill 派生）的写盘与命令都不弹卡**。主回合才有交互：`control/controller.go:772-782 EnableInteractiveApproval`，`:838-868 SetPermLevel` 三档 ask/auto/yolo（yolo 仍保留 hardAsk 硬门 `:851-858`）。`agent/request_permission.go:95-104`：headless 无 PermissionRequester 时明确拒绝"升级权限"。risk tier 只在 DAG 层（`dag/dag.go:33-36 RiskNormal/RiskHigh` + `StatusHold`），工具层与权限层无 risk 概念。
→ **含义**：V4 的审批模型支撑不了"高风险操作进案卷"。V5 需要 per-case 授权域 + 动作级 risk tier，并且必须改掉 boot 这一行装配（否则任何新运行时都会继承同一漏洞）。

### 结论 8｜并发批处理不是"无锁 map"，而是"N 个锁 N 个不变量"
`agent/batch_executor.go:291 runParallel`、`:292 const maxParallel = 8`、`:293` 信号量、`:299 go func()`；每个 goroutine 只写各自 `results[i]`（安全）。共享态逐个补锁：`execute_one.go:123/303/325/404/445` 五处 `a.turnMu.Lock()`（守卫 staleWrittenFiles / staleReadFiles / repeatSuccessCounts，`:325` 附近注释自述 "audit P0 — concurrent map writes crash"）、`file_observation.go:47-52` fileObs、`preMu` 守 preOutcomes、`evidence.Ledger` 内建 `sync.Mutex`（`evidence.go:49`）、`tool.Registry` 已改 `sync.RWMutex`（`tool/tool.go:192-226`，摘除 `:264-280`）。
→ **含义**：历史 P0（并发 map 写崩）已逐点修好，但**每加一个跨工具共享态就要人肉记得加锁**，这是结构性风险。V5 若引入 case 级并发，应收敛为"单一状态机所有权 + 通道传递"，而不是继续加第 6 把锁。

### 结论 9｜工具层已经在承载业务语义（48 个内置工具 vs 文档 17 个）
`internal/gaea/tool/builtin` 实测 **46** 个非测试文件、48 个工具名，其中 `cost_compose / cost_indicators / cost_tools / cad_ask / schedule_tools / docrender / chart_gen / diagram_gen / genui_validate / knowledge_add/search / memory_search / spacetags / workspace / bgjobs / browser_tools / hide_window_*` 都是**领域语义**。而 `.gaea/AGENTS.md:166` 仍写「内置工具精简为 17 个核心工具（v2.4.3 起）」。
→ **含义**：工具面是"业务长进内核"的第二处耦合点。V5 要把域工具降级为 case 可挂载的能力包（capability pack），内核只留文件/命令/网络/记忆/任务这 5 类原语。

### 结论 10｜可复用资产密度高且边界干净，但编排层全在 App 层
内部包干净：office 真编辑 `internal/office/{docxedit 3 文件/909 行, xlsxedit 4/1248, pptxedit 2/612, ooxml 4/496, xlsxpreview 4/823, qa 2/490, anchor 2/372, crosslink 2/363, standard 5/388, typography 3/213, cjk 2/179}`；转换 `internal/docmd`（ocr.go 772 + ocr_uocr2.go 603 / office.go 433 / pdf.go 761 / soffice.go 131）；进度 `internal/schedule`（cpm.go 416 / mpp.go 560 MS Project 导入 / xlsx.go 617 / ops.go 642 / baseline.go 159 / analysis.go 148）；CAD `internal/gaea/cad`（cad.go 440 / count.go 284 / pdf.go 104）；ComfyUI `internal/ai/image_comfyui.go 1003`；GenUI 后端极薄 `internal/gaea/genui`（validate.go 356），真身在前端 `frontend/src/genui`（19 文件/3,095 行，spec.ts 289 / guard.ts 658 / renderNode.tsx 1030）。**编排却长在 App**：`internal/app/gaea_dag.go 806`、`gaea_verify.go`（回滚 256-316）、`gaea_deliverable_registry.go:8-40`（产物登记，从事件日志折叠）。
→ **含义**：V5 迁移工作量约 30% 在搬包（低风险），**70% 在把 App 里的编排抽成 case runtime**（高风险，无测试护栏的那部分也在这里）。

### 附：案卷雏形在三处已经长得像了
- **DAG**：`internal/gaea/dag/dag.go:1-6` 包注释自陈"真相是 `<cwd>/.gaea/work/dag/<id>.json`，本包只做纯存储+纯函数，**执行在 App 层**——节点=TaskTool.RunNew 的全新子代理会话，产物=证据卡窗口增量，验收=人拍板回流记忆"；节点状态机 `:20-28`（pending→running→done|failed→skipped→accepted，+hold 待审批）；`Node` 结构 `:48-64`；ID 格式 `dag_<ts>_<seq>`（`:89-91`，测试钉住字典序=时间序）。
- **产物面板**：`gaea_deliverable_registry.go:8-40` 的 `DeliverableRegistryView` **从 session log 折叠**（`trajectory.FoldDeliverables` + `session.ReadLogRepaired`），上限 200 条——已取代早期"正文扩展名白名单"启发式；前端 `components/deliverables/{DeliverableRow 290, EvidenceSection 328, RegistrySection 76}`。
- **领域门禁**：小说 `internal/novelreview/review.go:98` 纯函数 `Review(text, platformID, opts)` + 8 维 `dims.go` + `rubric.go`；造价 `internal/gaea/workcost` 14 文件（store.go 803 / projectimport_sheets.go 687 / export.go 509 / bill.go 398）。

---

## 二、证据表

| # | 结论 | 文件:行 | 说明 |
|---|---|---|---|
| 1 | 绑定面数量 | `internal/app/bindings_*.go`（11 文件） | 实测 686 个导出方法；d.ts 同步 686 |
| 2 | 文档漂移 | `.gaea/AGENTS.md:159` | 写「635 个导出方法」，落后实际 51 |
| 3 | 发布说明漂移 | `releases/archive/v4.457.0.md`（见 `.gaea/AGENTS.md:10`） | 写「635→654」，仍低于实测 686 |
| 4 | 门面=纯委托 | `internal/app/bindings_manifest.go:11-23` | 返回 11 个 `&XxxB{a: a}`，共享同一 `*App` |
| 5 | 唯一绑定入口 | `main.go:79` | `Bind: app.NewBindings(application)`（main.go 共 94 行） |
| 6 | App 是 God 类型 | `internal/app/app.go:283-301` | 内嵌 `*core`(:52-88)/`*writingState`(:91-142)/`*mediaState`(:145-184)/`*whisperState`(:187-236)/`*officeState`(:239-279) |
| 7 | 冻结策略 | `.gaea/AGENTS.md:155` | God 类型不拆，新字段必须挂子结构体 |
| 8 | App 层规模 | `internal/app` 591 文件 / 117,871 行；`internal/gaea` 788 文件 / 136,672 行 | 两层各约 12–13 万行 |
| 9 | God file | `gaea_ui_extra.go 1112 / image_handler.go 1099 / create_chapter_handler.go 1006 / whisper_handler.go 954 / sin_handler.go 908 / app.go 828 / gaea_dag.go 806` | 单文件过千行 3 个 |
| 10 | 板块=页面硬耦合 | `frontend/src/gaea/boards/manifests.ts:80-153` | 13 条 board，id→page 一一对应 |
| 11 | 编译期锁死 | `boards/manifests.ts:159` | `BoardId` 由数组字面量派生 |
| 12 | 单 key→单组件 | `boards/pageRegistry.ts:12-22` | 一板块只能一个页面组件 |
| 13 | 导航收口 | `boards/manifests.ts:163-200` | deriveMenuBoards / NavigateWhitelist / ShortcutMap |
| 14 | 后端清单 seam | `boards/manifests.ts:253-285, 334-347` | `CoreB.GetBoardManifests`，失败回退静态清单 |
| 15 | App.tsx 只做壳 | `frontend/src/App.tsx:224-238`（241 行） | 主题令牌 + `<MainLayout/>`；MainLayout 723 行 |
| 16 | 前端规模 | `frontend/src/{gaea 612/104,225, pages 161/33,478, components 252/38,711}` | 行数含 tsx/css |
| 17 | 单 executor | `internal/gaea/agent/agent_run.go:18,19,22` | `Run` 直接 `runDirect`，唯一路径 |
| 18 | 主循环 | `agent_run.go:126` | `for step := 0; a.maxSteps <= 0 \|\| step < a.maxSteps \|\| graceRound; step++` |
| 19 | 无 resume | `agent_run.go:53`；`agent.go:566,571` | turn 边界只 `flushJournal`；checkpoint 是会话 flush 不是 step 机 |
| 20 | 中断=文本交接 | `internal/gaea/agent/interrupted_recovery.go` | 写 `<interrupted-turn-recovery>` 回灌上下文 |
| 21 | 日志不变量 | `internal/gaea/agent/session/log.go:3-9` | seq=已写行数；payload 无损 JSON；模型可见必入日志 |
| 22 | 日志结构 | `session/log.go:92-101, 28-87` | LogEntry{Seq,Ts,Kind,Space,Payload} + kind 全集 |
| 23 | 编译指纹 | `session/log.go:111-123` | request_header 带 compileHash/prevCompileHash/prefixStable |
| 24 | 落盘 sink | `session/sink.go:17-157`；`boot/boot.go:171-174` | EventLogSink 经 `event.Sync` 装配 |
| 25 | 前端折叠重建 | `gaea/lib/store/controller_state.ts:57-109` | item id = u/a/sa + logSeq；跨重启稳定身份=seq |
| 26 | 断点补拉 | `gaea/lib/store/controller.ts:40-90` | seq 缺口检测 + `GaeaResyncEvents` |
| 27 | 无 case 维度 | 全仓 grep `case_id|caseId|CaseID` = 0 | — |
| 28 | 表数量 | `internal/gaea/db/schema.go` | 60 张 CREATE TABLE |
| 29 | 事实/投影分层 | `schema.go:411-416`（SchemaV18 注释） | memory_events 追加式唯一事实源；mem_graph_* 可 DROP 重建 |
| 30 | 全局命名空间缺口 | `schema.go:44`（knowledge）、`:67`（cost_entries） | `name UNIQUE`，无 project/case 维 |
| 31 | tasks 加 session | `schema.go:197`、`:478-479` | 状态机 queued→running→succeeded\|failed\|cancelled + retry_count |
| 32 | 决策台账死表 | `internal/whisper/db/schema_v6.go:38` | decision_log 零 Go 引用，只出现在 schema 与 `whisper/db/database.go:210` wipe 名单 |
| 33 | 审批纯函数 | `internal/gaea/permission/permission.go:114-128` | deny>ask>allow；readOnly 恒 allow |
| 34 | nil Approver 放行 | `permission.go:259-261, 281-283` | "non-interactive: preserve autonomy" |
| 35 | 子代理全放行 | `boot/boot.go:273` + `:262-267` 注释 + `:301-302` | headlessGate → NewTaskTool，注释自陈子代理无 UI |
| 36 | 主回合交互审批 | `control/controller.go:772-782` | EnableInteractiveApproval + SetAsker + SetPermissionRequester |
| 37 | 三档权限 | `controller.go:838-868` | ask/auto/yolo，yolo 仍保留 hardAsk（:851-858） |
| 38 | retry_until 过闸 | `agent/task.go:1021-1037`（拒绝 `:1007-1011`，测试 `task_retry_gate_test.go:68`） | 走 `NewToolDispatcher(t.gate).Check` 后直接 Execute——**第二条执行路径**，重接闸靠人 |
| 39 | registry 锁 | `tool/tool.go:192-226, 264-280` | RWMutex；Add 在锁外做 schema 规范化 |
| 40 | risk 只在 DAG | `dag/dag.go:33-36` | RiskNormal/RiskHigh + StatusHold；工具层无 risk |
| 41 | 证据卡结构 | `evidence/journal.go:36-55`、`:22`、`:28-33` | Before/AfterSummary 截断 8KB；落 `.gaea/work/journal/*.jsonl` |
| 42 | 回滚无 hash | `internal/app/gaea_verify.go:256-316`（守卫 `:270-288`） | ClampSummary 字符串比对 + AtomicWrite；无 sha256 |
| 43 | 证据台账内存态 | `evidence/evidence.go:32-33, 34-121` | Receipt/Ledger 纯内存不序列化 |
| 44 | 并发 8 路 | `agent/batch_executor.go:291-299` | runParallel + maxParallel=8 + sem |
| 45 | 逐 map 加锁 | `execute_one.go:123,303,325,404,445`；`file_observation.go:47-52` | turnMu 五处；历史 P0 "concurrent map writes crash" |
| 46 | 证据写入点 | `execute_one.go:271, 274` | 成功/失败两路 `a.evidence.Record` |
| 47 | 工具读缓存 | `agent.go:466, 776`；`execute_one.go:140,206,214` | `cache.New(-1)` session 级 + mtime 校验 |
| 48 | 双空间 | `agent.go:106-129`；`agent_run.go:547-583` | WithSpace；play 空间不落 journal |
| 49 | DAG 自陈边界 | `dag/dag.go:1-6` | 真相在 workspace JSON；执行在 App 层；本包纯函数 |
| 50 | DAG 节点模型 | `dag/dag.go:20-28, 48-64, 67-73, 89-91` | 状态机 / Node / Run / `dag_<ts>_<seq>` 字典序=时间序 |
| 51 | 产物登记 | `internal/app/gaea_deliverable_registry.go:8-40` | 从 session log 折叠（FoldDeliverables），上限 200 |
| 52 | 产物 UI | `gaea/components/deliverables/{DeliverableRow 290, EvidenceSection 328, RegistrySection 76}` | 已有证据链/产物面板 |
| 53 | 小说门禁 | `internal/novelreview/review.go:98`；`dims.go`、`rubric.go` | 纯函数 Review + 8 维 + rubric |
| 54 | 造价模型 | `internal/gaea/workcost`（14 文件，store.go 803 最大） | 工料机/清单/组价，见 `.gaea/AGENTS.md:10` |
| 55 | office 引擎 | `internal/office/{docxedit,xlsxedit,pptxedit,ooxml}` | 909/1248/612/496 行 |
| 56 | 转换/OCR | `internal/docmd/{ocr.go 772, ocr_uocr2.go 603, pdf.go 761, soffice.go 131}` | 预览与 format_convert 同源 |
| 57 | 进度 CPM | `internal/schedule/{cpm.go 416, mpp.go 560, xlsx.go 617, ops.go 642, baseline.go 159}` | 含 MS Project 导入 |
| 58 | CAD/视觉 | `internal/gaea/cad/{cad.go 440, count.go 284}`；`vision/vision.go 250` | 读图/计数 |
| 59 | ComfyUI | `internal/ai/image_comfyui.go 1003`；`internal/app/image_comfyui_proc.go 558` | 本地生图进程 |
| 60 | 模型路由 | `App.routeModel("<feature>")`，调用点 `app/chat_handler.go:11`、`chat_service.go:73/130`、`create_chapter_handler.go:546/749` | 按**功能名字符串**路由，无集中注册表 |
| 61 | 路由配置 API | `bindings_core.go:29-30`、`bindings_model.go:49-50` | GetFeatureModel/SetFeatureModel |
| 62 | GenUI | `internal/gaea/genui/validate.go 356`；`frontend/src/genui/{spec.ts 289, guard.ts 658, renderNode.tsx 1030}` | 后端薄、前端厚 |
| 63 | eval | `internal/app/{eval_baseline_handler.go 389, gaea_retrieval_eval.go 226, gaea_memory_eval.go 89}` | 已有基线/检索/记忆三套 |
| 64 | CI/发布 | `scripts/{ci.ps1 78, release.ps1 180, smoke.ps1 64, test-all.ps1 60, check-docs.mjs 174}` | 门禁脚本薄，重活在 go/npm |
| 65 | 工具面超编 | `internal/gaea/tool/builtin` 46 文件/48 工具；`.gaea/AGENTS.md:166` 写 17 个 | 域工具（cost/cad/schedule/docrender）已进内核 |

---

## 三、可复用资产

| 资产 | 入口 | 规模 | 成熟度 | V5 处置 |
|---|---|---|---|---|
| session event log（append-only + 折叠重建 + 编译指纹） | `internal/gaea/agent/session/{log.go,sink.go}` | 日志层小、语义硬 | ★★★★★ 有测试钉不变量、前端已依赖 | **直接继承**为 case 事实层 |
| 绑定门面机制（11 门面 + 生成器 + 漂移守卫） | `internal/app/bindings_*.go`、`scripts/gen_bindings`、`check-bindings-drift.ps1` | 686 方法 | ★★★★☆ 自动化完整，数量已漂移 | 原样搬运；V5 收敛入口 |
| office 真编辑引擎 | `internal/office/{docxedit,xlsxedit,pptxedit,ooxml}` | 3,265 行 | ★★★★★ 有 qa/standard/typography | 直接继承为"改写器"能力 |
| 转换/OCR/PDF | `internal/docmd`（10 文件） | 2,400+ 行 | ★★★★☆ 与预览同源 | 直接继承 |
| 进度 CPM + MS Project 导入 | `internal/schedule`（31 文件） | 8,868 行 | ★★★★☆ 含 baseline/analysis | 继承为"工期 lens" |
| CAD 读图/计数 | `internal/gaea/cad` | 828 行 | ★★★☆☆ 依赖视觉模型 | 继承为能力包 |
| ComfyUI 生图 | `internal/ai/image_comfyui*.go` | 1,345 行 | ★★★★☆ 进程管理已收 | 继承为能力包 |
| 证据卡 + journal + 回滚 | `evidence/journal.go`、`app/gaea_verify.go` | 变更收据 300+ 行 | ★★★☆☆ 无 hash | **改造后继承**（换 hash/版本） |
| DAG 存储 + 纯函数校验 | `internal/gaea/dag`（dag.go 347 + filestore.go 109 + template.go 131 = 587 行） | 状态机/无环校验/ID 格式 | ★★★★☆ fail-closed + 格式钉 | 继承为 case 的 plan 层 |
| 产物登记（日志折叠） | `gaea_deliverable_registry.go` | 40 行 | ★★★★☆ 已替代启发式 | 继承，键改 case_id |
| 小说门禁纯函数 | `internal/novelreview` | 456 行 + 8 维 | ★★★★☆ 纯函数好测 | 继承为 case 校验器 |
| 造价三维模型 | `internal/gaea/workcost` | 14 文件 | ★★★★☆ 五表/活公式已验 | 继承为业务 lens |
| GenUI | `frontend/src/genui` + `internal/gaea/genui` | 3,095 + 507 行 | ★★★★☆ 词表/守卫/渲染齐 | 直接继承 |
| 模型路由 | `App.routeModel` + `internal/modelengine` | 34 文件/8,117 行 | ★★★☆☆ 字符串路由散落 | **替换签名**，保留引擎实现 |
| eval 框架 | `eval_baseline_handler.go` 等 5 文件 | ~800 行 | ★★★☆☆ 三套各自为政 | 合并为 case 级 eval |
| release/ci | `scripts/*.ps1`、`check-docs.mjs` | 薄脚本 | ★★★★☆ 已有全量/快闸两档 | 继承 |

---

## 四、必须替换／冻结清单

### 4.1 冻结（不改，原样搬进 V5）
1. **绑定门面协议**：`XxxB{a *App}` + 生成器 + `TestBindingsCompleteness` + excludedBindings 清单（`bindings_completeness_test.go:13-57`）。理由：686 方法的自动化链路完整，重写收益为负。
2. **session log 三不变量**：`session/log.go:3-9`。理由：这是全仓唯一"可重放"的硬保证，任何改动都会打断前端折叠重建（`controller_state.ts:57-109`）。
3. **office/docmd/schedule/cad/comfy 包边界**：这些包不依赖 `internal/app`，是 V4 里少见的干净分层，冻结其导出面即可整体搬运。
4. **App/mediaState/writingState 结构**：不拆（`.gaea/AGENTS.md:155` 已拍板），V5 新状态一律进 case runtime，不再往 App 加字段。

### 4.2 必须替换
| 项 | 位置 | 替换理由 | V5 目标形态 |
|---|---|---|---|
| headlessGate 装配 | `boot/boot.go:273, 301-302` | 子代理永久全放行（结论 7） | per-case 授权域 + 动作级 risk；子代理继承**受限**域而非放行 |
| 无 step 持久化 | `agent_run.go:126`、`interrupted_recovery.go` | 崩溃即丢（结论 3） | step 记录进 case log，可 resume |
| 8KB 摘要作为变更真值 | `evidence/journal.go:22, 36-55`、`gaea_verify.go:270-288` | 截断丢差异（结论 6） | content hash + 版本号 + 事务边界 |
| "板块=页面=功能" | `boards/manifests.ts:80-153, 159` | 业务实体无处安放（结论 2） | board 降级为 lens 注册表；case 是一等公民 |
| 功能名字符串路由 | `routeModel("<feature>")` 各 handler | 无注册表、无 case 维度（结论 5） | `Resolve(caseID, feature, risk)` 集中注册 |
| decision_log 死表 | `whisper/db/schema_v6.go:38` | 零引用（结论 5） | 重建为 case decision ledger（append-only） |
| 工具层域语义 | `tool/builtin` 48 工具中的 cost/cad/schedule/docrender | 业务进内核（结论 9） | 域工具降级为 case 能力包（capability pack） |
| 逐 map 加锁 | `execute_one.go` 五处 turnMu | N 锁 N 不变量（结论 8） | 单一状态机所有权 + 通道 |

### 4.3 禁止新增（V5 立项红线）
- 禁止往 `App` 顶层加字段（走 case runtime）。
- 禁止新增板块来表达新业务（只允许新增 lens 或 capability pack）。
- 禁止不经 case 上下文直接新增 Wails 绑定方法（686 已经够大）。
- 禁止在新的内部执行路径里绕过 `Gate.Check`（`task.go:1021-1037` 是唯一正例，也是唯一反例）。

---

## 五、迁移成本分级

### 低（机械改动，可脚本化 / 1 周内）
- 加 `case_id` 列 + 索引（可空，向后兼容）：`facts` `knowledge` `cost_entries` `semantic_vectors` `tasks` `cost_projects` `memory_events` `mem_graph_nodes` `mem_graph_edges` `session_index_meta` `gf_bill_items` `episodes` `procedural_habits` `turn_traces` `decision_log`；`knowledge`/`cost_entries` 的 `name UNIQUE` 需改为 `UNIQUE(case_id,name)`（这是唯一非纯加列的改动）。
- 文档与计数对齐：`.gaea/AGENTS.md:159`（635→686）、`:166`（17→48）、`:170`（facts 表→60 表 + memory_events/投影）。
- 前端保留为 lens：13 个页面组件（manifests.ts:80-153 的 page 字段）**零改动**，只改注册方式（board→lens registry）。
- 判定：**低**。

### 中（需要改签名但不改语义，2–4 周）
- `routeModel(feature)` → `Resolve(ctx, caseID, feature)`：调用点集中在 handler 层（`chat_handler.go:11`、`chat_service.go:73/130`、`create_chapter_handler.go:546/749` 等 10 余处）+ 配置 API `GetFeatureModel/SetFeatureModel`（`bindings_core.go:29-30`、`bindings_model.go:49-50`）。
- 产物登记键：`DeliverableRegistryView` 加 `CaseID`，前端 `RegistrySection/EvidenceSection` 加筛选（组件保留）。
- 绑定面：给门面方法加 case 上下文参数——**不要 686 个全加**，只给"写"类方法加（读类靠 session→case 映射解析）。
- `tasks` 表补齐 `case_id` + 状态机迁移（`schema.go:197`）。
- 判定：**中**。

### 高（新建 / 重构，1–2 个季度）
- **step 持久化 + resume**：目前完全不存在（结论 3），要从 `agent_run.go:126` 抽出 step 边界，写入 case log，做 replay/diff。
- **审批授权域重构**：从"全局 policy + headlessGate"改成"case 域 + 动作 risk + 子代理受限继承"（结论 7），并重接所有内部执行路径（`task.go:1021-1037` 模式）。
- **把 App 编排抽成 case runtime**：`gaea_dag.go 806` + `gaea_verify.go`（回滚）+ `gaea_deliverable_registry.go` + `gaea_ui_extra.go 1112`。这是最大的一块，且它是"无测试护栏"的那部分。
- **证据真值替换**：hash/版本/事务边界（结论 6），涉及 journal 格式迁移与全部历史 journal 的兼容读。
- 判定：**高**。

---

## 六、P0 风险

| # | 风险 | 证据 | 触发条件 | 处置 |
|---|---|---|---|---|
| P0-1 | 子代理审批全放行：任何 task/DAG 节点写盘与命令都不弹卡，且注释把这种行为写成设计 | `boot/boot.go:273, 301-302, 262-267`；`permission.go:259-261, 281-283` | 模型在子代理里执行破坏性命令 | V5 首批必修；V4 期间加"子代理写盘必须进证据卡"的旁路告警 |
| P0-2 | 无 step 持久化：崩溃/重启后长 DAG 节点从零重跑，且"已完成"无法证明 | `agent_run.go:126`、`interrupted_recovery.go` | 长任务中断 | 立 step 记录（先只写不读，风险最低） |
| P0-3 | 8KB 截断摘要当变更真值：大文档差异静默丢失，回滚"已被手工修改"守卫可被绕过 | `evidence/journal.go:22`、`gaea_verify.go:270-288` | 变更 >8KB 或首尾摘要相同 | 换 content hash |
| P0-4 | 文档漂移已成体系：绑定 635/654 vs 686、工具 17 vs 48、记忆"facts 一张表" vs 60 张表 | `.gaea/AGENTS.md:159,166,170` | 立项决策引用旧数字 | 本审计后立刻订正；V5 设计文档不得引用未复核数字 |
| P0-5 | 知识/造价表是全局命名空间：`name UNIQUE` 无 project/case 维，跨案卷同名即冲突 | `schema.go:44, 67` | 第二个案子复用同名知识条目 | 迁移时改复合唯一键 |
| P0-6 | 双空间不对称：play 不落 journal，审计链在乐园侧断点 | `agent_run.go:547-583` | 跨空间产物引用 | 明确"案卷只覆盖 work"或补 play 的只读收据 |

---

*本文件由只读审计生成，未修改仓库任何代码。所有 file:line 均可直接定位复核。*
