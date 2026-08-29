# gaea3 架构改造评审 · 05 核心机制（engine-core）

> 调研范围：`internal/gaea` 下的 agent / control / boot / event / provider / context / cache / compact（实为 agent/compact*）/ jobs / hook / permission / session 子域。
> 调研方式：只读精读（agent_run.go、controller.go、boot.go、session/save.go、provider/provider.go、event/event.go 等 40+ 文件），grep 全库交叉验证调用链与装配点。
> 结论一律附 `文件路径:行号`。行号基于当前 HEAD 快照。

---

## 1. 概览（子域文件清单 + 职责）

### 1.1 agent/（92 个 .go 文件，核心循环）

| 文件 | 职责 |
|---|---|
| `internal/gaea/agent/agent.go` | AgentRunner 结构体（137-336）、Asker/Gate/ToolHooks/Runner 接口（28-111）、New() 构造（491-562）、全部 Setter |
| `internal/gaea/agent/agent_run.go` | **主循环 runDirect**（18-378）：plan→step→stream→工具调用→gates→turn 结束 |
| `internal/gaea/agent/agent_stream.go` | stream()（13-152）：provider 流式消费、batcher、只读工具预执行、PostLLMCall 变换 |
| `internal/gaea/agent/agent_plan.go` | Plan()（34-64）：开工规划一次性 JSON 生成 |
| `internal/gaea/agent/agent_config.go` | **Options 全集**（10-67）——seam 注入面 |
| `internal/gaea/agent/execute_one.go` | executeOne（18-278）：单工具执行 + 全部执行期守卫 |
| `internal/gaea/agent/batch_executor.go` | executeBatch（15-119）：参数修复、param storm、并行分区、结果事件 |
| `internal/gaea/agent/tool_dispatch.go` | ToolDispatcher（19-23）：集中式执行前检查管线 |
| `internal/gaea/agent/compact.go` 等 | 压缩机制（见 §6.3） |
| `internal/gaea/agent/task.go` | task 子代理工具（97-556） |
| `internal/gaea/agent/subagent_store.go` | 子代理 transcript 持久化（continue_from）——**当前未接线** |
| `internal/gaea/agent/session/` | Session/Save/Load/state/branch（见 §6） |
| `internal/gaea/agent/{stop_gate,repeat_detect,param_storm,budget,cache,judge,detector,canonical_todo}.go` 等 | 各类 gate/守卫/工具 |

### 1.2 control/（25 个 .go 文件，会话驱动层）

`controller.go`（1003 行，全部方法清单见 §3）、`controller_approval.go`（审批桥）、`controller_memory.go`（记忆命令）、`controller_plan.go`（开工计划门）、`controller_submit.go`（Submit 斜杠路由）、`watchdog.go`（运行态看门狗）、`audit.go`（审计日志——**死代码**）、`dream.go`（Dream/Distill）、`input.go`（Compose/CustomCommand/RunSkill/MCPPrompt）、`refs.go`（@引用）、`controller_mcp.go`（MCP 热增删）、`slash.go`、`decisions.go`、`attachments.go`。

### 1.3 boot/（5 个 .go 文件，合成根）

`boot.go`（652 行，Build 装配见 §4）、`plugins.go`（MCP 插件加载）、`sysprompt.go`（L1 系统提示词组装）。

### 1.4 支撑包

- `event/`：event.go（Kind/Sink/Event 定义）+ sync.go（并发安全包装）——见 §5
- `provider/`：provider.go（接口+注册表）+ bridge/（kind=wubigrok → 模型中心）+ retry.go——见 §7
- `context/`：manager.go（TCCA 四层上下文内核，11-17）
- `cache/`：compiler.go/identity.go（L1）/runtime.go（L2）/skill.go（L3）/flow 在 context/ 下
- `jobs/`：jobs.go（会话级后台任务表）
- `hook/`：runner.go（15-20）——PreToolUse/PostToolUse/PostLLMCall/PromptSubmit/Stop 等
- `permission/`：permission.go——Policy（纯规则）+ Gate（90-97, 217-229）

> 注意：**没有独立的 compact/ 包**——压缩在 `agent/compact.go`、`compact_summary.go`、`compact_util.go`。

---

## 2. agent 循环与注入点（seam 雏形清单）

### 2.1 Run 完整流程（`internal/gaea/agent/agent_run.go`）

`Run → runDirect`（13-15）：

1. **turn 初始化**：traceID（20-21）、drain steer 队列（22）、evidence.Reset（24-26）、`Emit(TurnStarted)`（27）、语言偏好包裹（30）、用户消息入 session（31）、从历史重建 todo 状态（35，canonical_todo.go:18）、工具过滤重置（38-40）、预执行缓存/dedup/steer 计数/bg 标志/stale 文件追踪重置（43-53）、参数风暴断路器重置（61-63）、recall 提醒（67，recall_reminder.go:20）。
2. **主循环**（77）：`maxSteps <= 0 || step < maxSteps || graceRound`。
3. **mid-turn steer 消费**（80-83）：队列中用户引导 → session + `Emit(Steer)`。
4. **stream()**（84，agent_stream.go:13-152）：组装 tools（activeSchemas 优先，18-23）、消息直接用 `a.session.Messages`（24，**不走 ctxMgr.AssemblePrompt**）、prefix shape 校验（28，cache_guard.go:16）、`prov.Stream`（30-34）。流式分块处理：Reasoning/Text 经 batcher（render/batcher.go:77,85）、ToolCallStart → 早发 `ToolDispatch(Partial)`（70-73）、只读工具**预执行** goroutine（74-95）、Usage → 累计 session 缓存计数 + CompareShape 诊断（96-113）、ChunkError → 中断/终错（114-122）。PostLLMCall hook 存在时缓冲 reasoning 流末尾统一变换（43, 128-135），签名 reasoning 原文存储（142-144），收尾 `Emit(Message)`（149）。
5. **流恢复**（87-110）：`StreamInterruptedError` + `Emit(Retrying)`，step-- 不消耗轮次预算；恢复耗尽 → 保留部分输出返回错误（111-125）。
6. **输出修正**：finish_reason=length → 续写 nudge（130，output_continue.go）；空输出 → 重试注入（134）。
7. **Usage 事件 + 预算门**（138-156）：`Emit(Usage)`（含 Pricing/SessionHit/Miss）；budgetGate Check——Warn 发 Notice、Block 直接终止回合（143-155，budget/gate.go）。
8. **cache-shape 指纹**（157-161，cache_shape.go:50）。
9. **自动压缩**（173-175，grace 轮跳过）——见 §6.3。
10. **助手消息入 session**（181-187，保留 reasoning 但不上传）；`turnLastSummary` 记录；archive.RecordMessage（192-200，archive/archive.go:49）。
11. **无工具调用 → 终局门链**（202-253）：
    - graceRound 收尾（205-211）；
    - 空终答检测（215-225，3 次封顶）；
    - **Gate 1 taskGate**（229，stop_gate.go:24）；
    - **Gate 2 goalGate**（233，stop_gate.go:69）；
    - **finalReadinessCheck**（238-248，evidence 校验 complete_step 缺失，agent.go:726-741）；
    - steer 队列非空继续（249-251）；
    - 全过 → 返回 TurnResult（252）。
12. **工具批执行**（269，batch_executor.go:15-119）：repairArguments 先行（17-20）、paramStorm 抑制（23-36）、ToolDispatch 事件（45-51）、并行分区执行（87-95，runParallel 194-214）、ToolResult 事件（100-109）、截断 Notice（110-112）、批次一致性检查（116）、storm breaker（117，235-273）。
13. **结果后处理**（273-330）：写工具路径记入 TurnResult（275-284）、错误收集（291-297）、suppressed 占位（299-307）、**只读工具结果确定性去重**（309-323）、工具消息入 session（324-329）。
14. **complete_step 推进 canonical todo**（333-340）。
15. **循环打断器**：mid-turn steer 检测（343-345，agent.go:574-622）、bg start-kill 循环（349-351，agent.go:630-659）、重复工具检测（354-356，repeat_detect.go）。
16. **grace round nudge**（362-370）：maxSteps 到后注入「不要再调工具」的收尾提示。
17. 循环外返回 paused 错误（377）。

### 2.2 Options 注入点全集（`agent_config.go:10-67`）——seam 雏形清单

| 字段 | 行号 | 作用 | 当前 boot 是否接线 |
|---|---|---|---|
| MaxSteps / Temperature / Pricing | 11-13 | 轮次预算/采样/计价 | ✅ boot.go:304-306 |
| Gate | 16 | 权限门（permission.Gate 实现） | ✅ headlessGate（boot.go:307） |
| Hooks | 20 | ToolHooks（hook.Runner 实现） | ✅ boot.go:308 |
| Jobs | 23 | 后台任务管理器 | ✅ boot.go:309 |
| ContextWindow | 26 | 压缩窗口 | ✅ boot.go:310 |
| Compaction | 28 | 压缩配置 | ✅ boot.go:311 |
| Dispatcher | 31 | 执行前检查管线 | ✅ boot.go:312 |
| **CtxMgr** | 34 | TCCA 上下文内核 | ⚠️ 不在 Options 里传，走 SetCtxMgr（boot.go:383） |
| **AuditFunc** | 37 | 工具审计回调 | ❌ **从未接线**（全库无调用者） |
| ParamStorm | 41 | 参数风暴断路器 | ❌ 未接线 |
| BudgetLimit | 45 | 会话预算（元） | ❌ 未接线 |
| ModelProfile | 49 | 模型压缩阈值覆盖 | ❌ 未接线 |
| TemplatePrefix | 54 | 子代理模板前缀（缓存对齐） | ✅ 子代理/技能路径 |
| ActiveSchemas | 58 | 工具集覆盖（缓存对齐） | ✅ 子代理路径（task.go:341） |
| RuntimePrompt / Goal / DisableVerify | 59-66 | L2 注入 / 停止条件 / 校验抑制 | 部分 |

**运行时 Setter（agent.go）**：SetActiveSchemas（341）、SetGate（352）、SetAsker（361）、MergeRuntimePrompt（366，运行时上下文并到 L1 末尾）、SetGoal（373）、SetMemoryQueue（377）、SetSessionSaver（380）、SetPromoter（383）、SetSink（393）、SetArchive（395）、Session/SetSession（408-433，含前缀指纹与缓存计数重置）、SetCtxMgr（665-670）。

### 2.3 关键接口（已是可替换实现）

- `Asker`（agent.go:28-30）：ask 工具 → 用户；controller 实现（controller.go:612-634）
- `Gate`（agent.go:86-88）：每次工具调用的放行决策
- `ToolHooks`（agent.go:95-111）：PreToolUse/PostToolUse/PermissionRequest/PostLLMCall/SubagentStop/PreCompact
- `Runner`（agent.go:75-77）：AgentRunner 本身实现
- `Sink`（event.go:221-223）：单方法 Emit
- `Provider`（provider.go:282-289）：Name + Stream 双方法

### 2.4 单工具执行链（executeOne，execute_one.go:18-278）——执行期守卫全序

1. 工具查找（19-25）：未知工具 → 包装错误。
2. **重复成功守卫**（29-35）：同轮写工具签名成功 ≥2 次即 block（repeatedSuccessBlock 303-315，repeatSuccessSignature 331-353 覆盖 write/edit/multi_edit/delete/bash 写命令）。
3. **预执行检查**（41-93）：dispatcher 路径（Check：PermissionRequest hooks → gate → PreToolUse hooks，tool_dispatch.go:53-104）；无 dispatcher 时内联同样三段。
4. **确定性预检查**（98-104）：编辑锚点存在性预检（tool_precheck.go）。
5. **stale anchor 守卫**（106-115）：同轮编辑过的文件必须先 read_file 再编辑。
6. **工具结果缓存**（117-127, 164-184）：read_file 按 path+offset 缓存，写工具失效（agent/cache/toolcache.go）。
7. **调用上下文装配**（129-146）：withCallContext（sink/asker）、evidence ledger、jobs manager、memory queue/session saver/promoter。
8. **执行**（150-160）：ContextualTool（带 session 消息）或普通 Execute；duration 计时。
9. **auditFunc**（187-195）：V3.2 审计钩子——**全库未接线**。
10. **workspace observer**（198-202）：NotifyEdit → ContextManager.OnFileEdited（manager.go:115-117）→ RuntimeLayer.TrackEdit（runtime.go:150-176，版本计数）。
11. **evidence 记账**（204-212）：complete_step/普通调用都记 Receipt（evidence/evidence.go）。
12. **PostToolUse hooks**（215-217，不可阻塞）。
13. **错误包装**（218-232）：可恢复错误标记 + 非法 JSON 附带 schema（226-228）。
14. **成功记账**（233-252）：repeat success、bg 启停模式追踪（bash run_in_background / bash_output / kill_shell 分支）。
15. **stale 文件追踪**（254-269）、**SubagentStop hook**（271-273）、**SmartCompress 压缩输出**（274-277）。

---

## 3. Controller 命令面（`internal/gaea/control/`）

包注释明确定位：**transport-agnostic 会话驱动层，所有前端共用同一命令面 + 事件流**（controller.go:1-11）。

### 3.1 回合驱动（controller.go）

| 方法 | 行号 | 职责 |
|---|---|---|
| Send / SendWithRaw | 366-368 / 379-397 | 启动回合；运行中消息**排队**（sendQueueLimit=8，138-140），队满拒绝 |
| turnLoop | 287-314 | 回合循环：执行 + FIFO 排空 pendingSends，每次 TurnDone |
| runTurnWithRaw | 415-487 | 单回合主体：session 运行标记（422-437）、ctxMgr.ProcessFirstTurn（445-453）、Compose（455）、UserPromptSubmit/Stop hooks（461-470）、autoPlan 计划门（472-478）、runner.Run（479）、回合后 Snapshot（483-485） |
| Run | 537-562 | headless 同步执行（无 TurnDone/排队） |
| Cancel | 566-573 | 取消当前回合 ctx |
| Running | 576-580 | 回合在途查询 |

### 3.2 审批/提问

| 方法 | 行号 | 职责 |
|---|---|---|
| Approve | 585-593 | 按 ID 回答 ApprovalRequest（allow/session 记忆放行） |
| EnableInteractiveApproval | 600-607 | 换交互 Gate（permission.NewGate + gateApprover）+ 挂 Asker |
| Ask | 612-634 | 实现 agent.Asker：发 AskRequest 并阻塞 |
| AnswerQuestion | 638-646 | 按 ID 回答 AskRequest |
| SetPermLevel / PermLevel | 653-683 / 686-690 | ask/auto/yolo 三档；yolo 也保留 hardAskTools 硬门（controller_approval.go:18-25） |
| requestApproval | 68-122 | 发 ApprovalRequest + 阻塞；会话 grant 短路；promptMu 串行化 |

### 3.3 会话生命周期

| 方法 | 行号 | 职责 |
|---|---|---|
| NewSession | 793-816 | Snapshot → SessionEnd → 轮换文件 → SetSession → SessionStart |
| Resume | 821-828 | 载入 transcript 换 Session + 固定路径 |
| Snapshot | 834-849 | **唯一调用 session.Save 的路径**（回合后 + 手动） |
| SetSessionPath / SessionDir / SessionPath | 853-869 | 持久化路径管理 |
| History | 873-878 | 当前消息快照（Snapshot 拷贝） |
| maybeSessionStart | 729-743 | 首个回合懒触发 SessionStart + 自动建文件 |
| Close | 981-994 | SessionEnd + jobs.Close + cleanup |

### 3.4 查询/观测

ContextSnapshot（882-891）、SeedContextUsage（895-900）、CompactRatio（904-909）、LastUsage（913-918）、SessionCache（923-928）、Balance（933-938，billing.Fetch）、TCCAStats/TCCAReport（747-780）、SystemPrompt（783-788）、Host/Commands/Skills/Tools/HookRunner（942-967）、Jobs（998-1003）、Label（977）。

### 3.5 命令路由与杂项

- **Submit**（controller_submit.go:13-249）：`/compact`（**实为 no-op 桩**，见 §9）、`/dream`、`/memories`、`/goal`、`/perm`、`/distill`、`/new`、`#note` 记忆速记、`/mcp__*`、自定义命令/技能斜杠、默认走回合。
- **Compose**（input.go:14-68）：turn-tail 注入 <memory-update>、<session-facts>、<background-jobs>、记忆 RecallBlock。
- **CustomCommand/RunSkill/MCPPrompt**（input.go:73-144）。
- **Dream/Distill**（dream.go:310-347，SaveDreamFacts 写入记忆 + 审计 controller_memory.go:280-316）。
- **AddMCPServer/RemoveMCPServer** 等（controller_mcp.go:15-146）。
- **watchdog**（watchdog.go）：墙钟 10 分钟/停滞 30 秒默认（43-46），sink 包装器观测推进（199-219），与用户 Cancel 同一取消链路（165-174）。

---

## 4. boot 合成根与 sysprompt

### 4.1 Build 装配顺序（`internal/gaea/boot/boot.go:82-418`）

1. **config.Load**（87）→ 模型解析：opts.Model 回退 cfg.DefaultModel（91-94）→ ResolveModel（95）→ ContextWindow 0 兜底 1M（101-103）→ RequireKey 校验（104-108）。
2. **sink 串行化 + jobs**（114-115）：`event.Sync(opts.Sink)` + `jobs.NewManager`。
3. **provider 构建**（117）：`NewProvider(entry)`（479-494）→ `provider.New(e.Kind, Config{...})`——bridge 的 kind 注册名为 **"wubigrok"**（bridge.go:159）。
4. **系统提示词**（132-141）：`buildSystemPrompt`（见 4.3），产出 prompt/mem/skills/compiler/runtimeCtx/skillStore。
5. **工具注册表**（143）：addBuiltins（155，Workspace 绑定 / ConfineWriters+ConfineBash，500-533）。
6. **插件**（159-161）：`startPlugins`（plugins.go:23-55）——plugin.NewHost + PluginSpecs(cfg.AutoStartPlugins()) + CONTEXT7 + plugin.StartAvailable + MCPStartupNotice（失败警告）。
7. **maxSteps**（162-165）。
8. **权限**（173-174）：`permission.New(Mode, Allow, Ask, Deny)` + headlessGate（Approver=nil → 自治放行）。
9. **hooks**（181-190）：项目级 hook 需信任（hook.IsTrusted），非阻塞输出经 sink 发 Notice。
10. **task 工具**（197-214）：agent.NewTaskTool（复用 execProv/reg/maxSteps/headlessGate）；SubagentModel 覆盖子代理 provider（204-213）。
11. **记忆工具**（219-222）：remember/forget/promote_session_facts/memory_get；ask 工具（228）。
12. **技能 + 子代理包装**（236-283）：skillRunner 按技能可选独立模型；RunSubAgent 以 childCompiler.Fork() 的 L1 + RuntimePrompt 为 L2（256-270）；run_skill/install_skill + 4 个内置子代理工具；模板注册（275-277）；largefile 摘要工具（283）。
13. **编译器接线**（285-292）：compiler.SetRegistry(reg)、taskTool.SetCompiler(适配器 587-592)、SetRuntimePrompt。
14. **ToolDispatcher**（295）：NewToolDispatcher(headlessGate, hookRunner)。
15. **工具集压缩**（298-300）：applyCompactToolset 隐藏冗余工具（633-651）。
16. **执行器构造**（303-313）：`agent.New(execProv, reg, execSess, Options{MaxSteps,Temperature,Pricing,Gate,Hooks,Jobs,ContextWindow,Compaction,Dispatcher}, sink)`——**注意：未传 AuditFunc/ParamStorm/BudgetLimit/CtxMgr/ModelProfile**。
17. **archive**（316-323）：.gaea/archive + executor.SetArchive。
18. **自定义命令 + slash 工具**（329-353）。
19. **ExtraTools**（356-360）：桌面端注入 image_gen/diagram/routine_llm 等。
20. **label/runner**（366-367）：单模型架构，runner=executor。
21. **TCCA ContextManager**（369-383）：四层（compiler.IdentityLayer()/runtimeCtx/skillLayer/FlowLayer{Window,TailTokens:16384}），executor.SetCtxMgr。
22. **缓存预热**（386-391）：L1 hash 落盘 .gaea/cache/identity.hash（identity.go:101-129）。
23. **control.Options → control.New**（393-417）。

### 4.2 插件加载（plugins.go）

startPlugins（23-55）：只做三件事——构造 Host、扩展 specs（CONTEXT7 追加 28-35）、plugin.StartAvailable + 注册工具 + 失败 Notice。host 永远构造（boot.go:158 注释），保证 /mcp add 可热增。

### 4.3 sysprompt 组装（sysprompt.go:34-108）

1. ResolveSystemPrompt（35，读 system_prompt_file 或 DefaultSystemPrompt，config.go:380）。
2. outputstyle.Apply（42-44）。
3. 追加 LanguagePolicy（45）。
4. 记忆迁移 + memory.Load（47-54）→ memory.Compose（57-62）。
5. pins.Block 常用资料（65-67）。
6. 技能索引 skill.ApplyIndex（72-74）+ read_skill resolver（76-82）。
7. 项目画像 Profile.Scan（84-85，cache/profile.go:28-35）。
8. L1 compiler = cache.New(sysPrompt, nil)（86）。
9. L2 runtimeCtx = NewRuntimeLayer + SetProject（88-98）+ SetCompactL2(true)。
10. 产出（100-107）。

> 注意：buildSystemPrompt **不**构造 execSess 之外的 sysPrompt；execSess 的系统消息 = `compiler.SystemPrompt() + "\n\n" + agent.SingleModelPrompt`（boot.go:302）——SingleModelPrompt 是执行纪律（single_prompt.go:9）。

---

## 5. 事件系统与持久化缺口

### 5.1 Kind 全集（`internal/gaea/event/event.go:19-69`）

TurnStarted / Reasoning / Text / Message / ToolDispatch / ToolResult / Usage / Notice / Phase / ApprovalRequest / AskRequest / TurnDone / CompactionStarted / CompactionDone / Steer / Retrying（16 种）。

Event 载荷字段（188-214）：Text/Reasoning/Tool/Usage/Pricing/UsageSource/SessionHit/SessionMiss/Turn/Level/Approval/Ask/Err/Compaction/RetryAttempt/RetryMax。UsageSource 区分 main/subagent/executor（81-85）。Tool 支持 Partial 早期卡 + ParentID 子代理嵌套（90-111）。Ask 可携带结构化 Plan（142-145）。

### 5.2 Sink 接口与实现

- `Sink{ Emit(Event) }`（221-223）；FuncSink 适配（226-233）；Discard（237）。
- `event.Sync`（sync.go:16-32）：并发 Emit 串行化包装——boot 在入口统一包装一次（boot.go:114）。
- 终端渲染 Sink（render/sink.go:57-180）；流批处理 batcher（render/batcher.go:27-87，32B/4ms 合并）。
- **watchdogSink**（watchdog.go:228-236）：转发前观测推进状态——**sink 包装链已是既有模式**。

### 5.3 事件发射点清单（非测试代码）

| 位置 | 行号 | 事件 |
|---|---|---|
| agent_run.go | 27 / 82 / 101 / 139 / 146,150 / 164 / 220 / 243 | TurnStarted / Steer / Retrying / Usage / Notice(budget×2) / Notice(finish_reason) / Notice(empty-final) / Notice(readiness) |
| agent_stream.go | 70 / 109 / 133 / 149 | ToolDispatch(partial) / Notice(prefix-changed) / Reasoning(transformed) / Message |
| batch_executor.go | 32 / 45 / 100 / 111 / 270 | Notice(param storm) / ToolDispatch / ToolResult / Notice(truncated) / Notice(loop guard) |
| compact.go | 110 / 131 / 145 / 153 / 183 / 198 / 217 / 576 | Notice(soft) / Notice(pruned) / Notice(skip) / Notice(stuck) / CompactionStarted / Notice(fallback) / CompactionDone / Usage(summarizer) |
| cache_guard.go | 27 / 32 / 38 | Notice(prefix drift) |
| canonical_todo.go | 115 / 117 | ToolDispatch/ToolResult（todo 状态同步） |
| controller.go | 251 / 301,311 / 338 / 384 / 531 / 623 | Notice(已运行) / TurnDone / Notice(看门狗) / Notice(队满) / Notice / AskRequest |
| controller_approval.go | 98 | ApprovalRequest |
| controller_plan.go | 62 | AskRequest(PlanCard) |
| boot.go | 185 / 188 / 388 | Notice(hook 输出) / Notice(hooks 未信任) / Notice(cache warm) |
| plugins.go | 48 | Notice(MCP 失败) |
| jobs.go | 126 / 184 | Notice(任务开始/结束) |

### 5.4 持久化现状（对比：session.Save 只存 Messages）

**事件本身零持久化**——全部事件只经 Sink 实时转发给前端（桌面端 gaea_handler.go:79-85 转 `gaea-event` 回调，387-488 gaeaEventMap 转 WireEvent）。磁盘上存在：

1. **session.Save**（save.go:24-38）：JSONL 消息，**仅 provider.Message 四角色**；由 controller.Snapshot（controller.go:834-849）在**每回合结束后**调用。
2. **archive.RecordMessage**（archive/archive.go:49-76）：只记 assistant 消息（截断 2000 字符），供 Dream/Distill。
3. **audit.jsonl**（control/audit.go）：**AuditLogger 全库无调用者**（NewAuditLogger 只定义不构造），AuditFunc 从未在 boot 装配——死 seam。
4. **state.json**（session/state.go:15-19）：running 标记 + 摘要，崩溃恢复用。
5. **子代理 transcript**（subagent_store.go）：NewSubagentStore/WithTranscripts **无生产调用者**——continue_from 功能未接线。
6. **记忆/成本库/知识库**：各自落盘（memory/knowledge/cost），非会话事件。

### 5.5 事件持久化缺口清单（Step 1 要补的）

- ❌ ToolDispatch/ToolResult 的**参数与输出**（工具调用事实源）
- ❌ Usage（tokens/成本/缓存命中）——只有前端临时统计
- ❌ ApprovalRequest/AskRequest 及**用户答复**
- ❌ CompactionStarted/Done（trigger/summary/archive 路径）
- ❌ Steer/Retrying/Notice/Phase
- ❌ Turn 边界元数据（turn id、时间戳、TurnResult 摘要）

现有 JSONL 消息日志无法重建上述事实：压缩会 `Replace` 掉中间消息（compact.go:214），工具结果只作为 tool 消息内容存在且可被 prune/压缩折叠。

### 5.6 事件流到前端的完整链路（桌面端）

1. agent 各发射点 → `a.sink`（AgentRunner 持有）。
2. boot 统一 `event.Sync` 串行化（boot.go:114）。
3. controller.New 时若 watchdog 启用，再包 `watchdogSink`（controller.go:224-235）。
4. 桌面端最终 sink = `event.FuncSink`（app/gaea_handler.go:79-85）：`a.emit("gaea-event", gaeaEventMap(e))`；`TurnDone && Err==nil` 触发后台"自动做梦"（82-84）。
5. `gaeaEventMap`（387-488）转 `gaeaW WireEvent` 兼容格式：kind 名映射表见 491-504（Steer/Retrying 等新 Kind 未映射 → "unknown"）。

**结论**：事件从发射点到前端之间已形成"Sync → watchdog → FuncSink"的包装链，Step 1 的持久化 sink 插在 2-3 之间即可，不改动任何发射点。

---

## 6. session 全链路（Step 1 改造面评估）

### 6.1 Session 核心（`agent/session/session.go`）

- 结构（16-20）：`Messages []provider.Message` + `rewriteVersion`（压缩重写计数）。
- 方法：New（23）/ Add（32）/ PrependSystem（41）/ Truncate（49）/ Replace（59，压缩用）/ Snapshot（68，跨 goroutine 拷贝）/ HasContent（77）/ RewriteVersion（89）/ IncrementRewrite（91）。
- `agent/session_alias.go`：**已做包拆分兼容层**——Session 等类型与函数 re-export（8-34），新代码应直用 session 子包。

### 6.2 save.go 整文件机制（Step 1 重写对象）

- **Save**（24-38）：全量 JSONL 重写 + fileutil.AtomicWrite 原子落盘（临时文件+rename）。注释明确：不 append 是因为压缩会改中间消息，append 无法对齐（21-23）。
- **Load**（43-67）：json.Decoder 流式解码（不用 Scanner——单条多 MiB 输出会超行缓冲）。
- Info/List/ListArchived（72-130）：预览（首个 user 消息截 80 字符）+ 回合数；无 user 消息的会话跳过。
- Archive/Unarchive（167-199）：移动 .jsonl + .meta 到 archive/ 子目录。
- NewPath（204-210）：`<UTC时间戳>-<model>.jsonl`。

### 6.3 压缩与保存的交互（`agent/compact.go`）

- 触发：maybeCompact（89-157）——soft 50% 提示 → prune 过期工具结果（prune.go:25）→ 90s 超时 LLM 摘要（Tier 3）→ 机械折叠兜底。stuck 检测：连续 2 次压缩不达标即暂停（150-156）。
- compact（161-220）：planCompaction 定界（287-328，pinned 前缀 + tail 预算 16384/25% 窗口）、partitionFold 拆分（397-407）、折叠消息先 archive（188-194）、摘要以 `<compaction-summary>` user 消息注入（206-213）、`session.Replace + IncrementRewrite`（214-215）。
- 前缀不变性：L1+L2+首条 user（可 pin）+旧 digest 永远保留（pinnedPrefixLen 359-375）；KeepPolicy 保留错误/标记/受保护工具结果（keepIndexes 427-456）。
- **与保存的交互**：压缩在回合**中途**改内存 Messages；session.Save 只在**回合结束**时整文件重写 → 磁盘上只有压缩后的状态，被折叠的原文仅存在于 .gaea/archive。**这就是"消息日志不可重放"的根因**——事件日志改造的直接动机。

### 6.4 branch/state/checkpoint

- branch.go：BranchMeta sidecar（18-26）+ Load/Save/Ensure/Touch（59-127）+ ListBranches（130-180）；**分支=独立 .jsonl 文件 + .meta 指针**，无真正 fork-copy。
- state.go：中断状态 sidecar（<base>.state.json），Running 残留即"上次未完成"信号（1-19）。
- checkpoint：**无独立实现**——controller 注释提到"per-session checkpoint store"（controller.go:236,456）但只有 agent.go:401 处的残留注释，无代码。

### 6.5 Step 1 最小改动面评估

要动的一切：
1. **save.go 重写**：Save 从"全量消息 JSONL"改为/并存"事件日志"；Load 与 Resume 兼容（Resume 路径：app/gaea_ui.go:516-517,584）。
2. **Snapshot 触发点**（controller.go:483-485, 834-849；controller_submit.go:33-35,64-66）：回合后/手动。
3. **event 增加持久化 sink**：在 boot.go:114 的 `event.Sync` 之后串一个持久化包装（同 watchdogSink 模式）。
4. **子代理会话**（task.go:311-374 + subagent_store.go）与主会话共用同一 Save/Load 格式。
5. **压缩钩子**：CompactionDone 事件已带 Summary/Archive，事件日志可作为压缩的事实保留（无需再解析消息）。
6. 测试面：save_test.go、session_concurrency_test.go、controller_send_test.go、session_test.go 等。

---

## 7. provider 层与模型桥接

### 7.1 Provider 接口与注册表（`provider/provider.go`）

- 消息模型（28-63）：Message（含 ReasoningContent/ReasoningSignature/ToolCalls/ToolCallID）、ToolCall、ToolSchema、Request。
- **Provider 接口只有两个方法**：Name() + Stream(ctx, Request)（282-289）——最小 seam，易于替换实现。
- Factory 注册表：Register（330-335，重复 kind panic）→ New（338-351）→ Kinds（354-361）。Config（292-299）含 Engine 字段（bridge 专用）。
- SanitizeToolPairing（83-135）：发送前修复 tool_calls 配对（断点续传占位）。
- 错误面：AuthError（306-321）、StreamInterruptedError（367-389，可恢复流中断）。
- retry.go：BackoffStrategy/RetryPolicy 通用重试（31-111）+ Retry-After 解析（118-143）+ 可重试判定（146-163）。

### 7.2 bridge 桥接（`provider/bridge/bridge.go`）——kind = **"wubigrok"**

> 任务描述写 kind=gaea，但代码注册名是 `"wubigrok"`（159-173），"gaea" 只是配置里的 ProviderEntry.Name（app/gaea_handler.go:55-63）。

- Provider{name, model, client ai.LLMClient, engine}（18-23）。
- **Stream 转发**（29-122）：gaea Request → ai.ChatRequest（含 EngineID；herdsman/ollama 开 EnableThinking 且 max_tokens 抬到 4096，41-51）→ ai.Client.ChatStream → 逐块转 provider.Chunk；**ChunkUsage 透传**（83-87，统计面板依赖）；ctx 取消防阻塞发送（67-74）。
- **模型解析链**：配置 ProviderEntry（Model 可为空）→ boot.NewProvider（boot.go:479-494）→ bridge factory：cfg.Model 空则回退 featureModel（SetFeature 注入的功能级模型）；Engine 空则回退 featureEngine（165-171）。
- **注入点**：app.GaeaInit（app/gaea_handler.go:135-194）——bridge.SetClient(a.client)（143）、GetFeatureModel("gaea") → SetFeature（147-150）；运行时改绑经 applyOfficeFeatureModel 重建 controller（228-237）。
- **定价与统计**：计价在 config（ProviderEntry.Price/Prices，config.go:267-272；默认表 480-484）；费用计算 Pricing.Cost（provider.go:254-261）；Usage 事件带 Pricing 供前端统计（agent_run.go:139）；钱包余额 billing.Fetch（controller.go:933-938）；会话累计缓存命中 sessCacheHit/Miss（agent.go:167-168）。bridge 本身不含价格逻辑。
- **关键事实**：全库 provider.Register 只有 bridge 一处（boot_test 除外）——**"openai"/"xai" kind 无工厂实现**，config.go:480-484 的默认条目是死配置；桌面端始终覆盖为 kind=wubigrok。

### 7.3 消息/工具转换细节（bridge.go:176-217）

- `toChatMessages`（176-198）：gaea Message → ai.ChatMessage；**ReasoningContent 不转换**——reasoning 只在服务端"思考"阶段消费，不回传（与 agent_run.go:177-180 注释一致：重发 reasoning 会按 prompt 计费且无缓存收益）。
- `toChatTools`（201-217）：ToolSchema → ai.ChatToolFunctionSpec（OpenAI 兼容 function 格式）。
- 语义缺口：ai.ChatMessage 无 ReasoningSignature 字段——Anthropic 式签名思考块无法经 bridge 走通（bridge 只面向 OpenAI 兼容的模型中心）。
- engine 特判（41-51）：herdsman/ollama 强制 EnableThinking + chat_template_kwargs，且 max_tokens < 4096 时抬到 4096（防"只有推理无正文"）。

### 7.4 子代理的独立 provider 路径（task.go:311-374）

runSubSession 支持 `subagentProv`（SetSubagentProvider，295-299）：配置 `subagent_model`/技能级模型时子代理用独立 provider（boot.go:204-213, 236-244），父代理前缀缓存不受子代理 API 调用影响；ActiveSchemas 强制与父代理工具集对齐（341,351）以复用 DeepSeek 前缀缓存。

---

## 8. 与 3.0 目标相关的关键发现

### 8.1 既有 seam 雏形（接口已可换实现）

1. **事件流已是 typed stream**：16 种 Kind + 结构化载荷 + Sink 单方法 + Sync/watchdog 包装链——"会话事件日志作事实源"的发送端已就绪，只缺持久化端。
2. **Provider 注册表**：Factory 模式 + init() 自注册，加 kind 即插即用（Provider seam 化基本完成）。
3. **Gate/Asker/ToolHooks/Runner**：全部 interface 化 + nil 安全 + happens-before 契约（agent.go:216-219）。
4. **permission 纯规则 Policy + Gate 分离**（permission.go:90-128, 217-229）。
5. **session 已拆子包**（session_alias.go），兼容层保留。
6. **Options 结构体 + New() 统一构造**——但部分注入面未接线（见 8.3）。
7. **TaskCompiler 接口**（task.go:84-87）——cache.Compiler 与 agent 解耦的既有适配层（taskCompilerAdapter，boot.go:587-592）。
8. **headless vs 交互双 Gate 切换**：EnableInteractiveApproval 运行时换装（controller.go:600-607）。

### 8.2 事件持久化缺口（见 §5.5）——Step 1 的核心增量

### 8.3 Step 1 最小改动面评估（汇总）

- **事件日志 sink**：新增一个包装 Sink 落盘（JSONL/轮转），挂到 boot.go:114。
- **save.go 重写**：session 文件从"消息全量重写"演进为"事件追加 + 派生 Messages"；保留 Load/Resume 兼容（或一次性迁移）。
- **压缩对接**：CompactionDone 事件写入日志后，消息日志与事件日志不再互相矛盾。
- **watchdog 观测**：watchdogSink 已是"观察者包装"范式，持久化 sink 复用同一模式。
- 桌面端 gaeaEventMap（app/gaea_handler.go:387-488）需要加新 Kind 映射（目前 unknown）。

---

## 9. 缺陷与风险

1. **审计功能是死的**：AuditLogger（control/audit.go:19-23）与 AuditFunc（agent.go:290）定义齐全但**全库无调用者**——声称的"V3.2 基础审计轨"未落地。风险：审计期望落空。
2. **/compact 是 no-op 桩**：controller.Compact（716-722）只发一条"无需手动压缩"的 Notice 就返回；依赖自动压缩。文档/UI 若有手动压缩入口会误导。
3. **子代理 continue_from 未接线**：subagent_store.go 完整实现但 NewSubagentStore/WithTranscripts 无生产调用者——功能只存在于测试。
4. **TCCA 与 Session 双轨消息源**：stream() 明确用 session.Messages 而非 ctxMgr.AssemblePrompt()（agent_stream.go:14-24 注释）——FlowLayer 只在压缩时更新，存在"信息污染"风险注释；两套真相需维护一致。
5. **消息日志不可重放**：压缩 Replace（compact.go:214）+ prune 使 JSONL 丢失中间事实；回合中途崩溃丢失整个回合（Snapshot 只在回合后）。
6. **provider kind 死配置**：默认表（config.go:480-484）kind=openai/xai 无工厂；文档注释（provider.go:2-4）提到 provider/openai 子包已不存在。桌面端靠 gaeaLoadConfig 强制覆盖规避，CLI 路径（若有）会直接报 unknown kind。
7. **AuditFunc/ParamStorm/BudgetLimit/CtxMgr 未装配**：Options 有字段但 boot 不传——seam 存在而无效，测试覆盖率与实际运行不一致。
8. **event.Tool 的 Args 全量 JSON 透传**：大输出只在 ToolResult 截断展示（truncateToolOutput），事件本身仍带全量——持久化时需定义截断策略。
9. **Session.Save 全量重写 O(n)**：会话增长后每回合全量序列化 + 原子写；事件日志改造若改为 append 需处理压缩对齐。
10. **watchdog 默认开启**（10 分钟墙钟）对超长办公任务可能误杀（豁免仅覆盖工具执行/等用户期，watchdog.go:199-219）。

---

## 10. 改造建议

1. **Step 1 落地事件日志**：在 boot.go:114 的 Sync 之后追加持久化 Sink（JSONL，按 session 分文件，含 Kind/载荷/时间戳/turn id），事件为唯一事实源；session.Save 降级为"派生视图"。
2. **save.go 演进而非推倒**：保留 JSONL 加载兼容，新增 event-log 读取重建 Messages 路径；Load 优先事件日志，缺失时回退旧格式。
3. **接线既有死 seam**：boot.go:303-313 补传 AuditFunc（接 AuditLogger）、ParamStorm、BudgetLimit（预算门在 runDirect 已就绪，只差装配）。
4. **Provider 注册表补全**：注册 openai/xai 工厂或删除默认表死条目，避免 kind 解析假死。
5. **/compact 或真正实现、或移除入口**：把手动压缩改为调用 CompactNow（compact.go:223-225）。
6. **子代理 transcript 接线**：boot 创建 SubagentStore 并 WithTranscripts，让 continue_from 生效；其 Save/Load 与事件日志共用格式。
7. **事件日志与压缩协同**：CompactionDone 记录折叠区间 + archive 路径；恢复时以事件日志重放，压缩摘要作为 checkpoint 快照，彻底解决"消息被折叠不可恢复"。
8. **双轨消息源收敛**：评估让 stream() 切到 ctxMgr.AssemblePrompt() 或明确 FlowLayer 仅为压缩服务的定位，消除两套真相。
9. **TCCA 层板块 Manifest 化**：L1/L2/L3/L4 已是四层结构（context/manager.go），可作为板块清单（Manifest）化的起点——每层一个接口 + 构建器。
10. **测试对齐**：为审计、预算、事件持久化补装配级测试（boot_test.go 模式），避免 seam 空转。

---

*报告完 · gaea3-review/05-engine-core.md · 只读调研，无代码改动*