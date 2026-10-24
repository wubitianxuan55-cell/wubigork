# ZCode Agent 运行时内核 蒸馏报告（S1）

> 调研对象：`C:\AI\wubigrok\clones\zcode`（TypeScript monorepo，只读）
> 落地方：gaea 办公板块本地 agent（Go + Wails，`C:\AI\wubigrok`）
> 调研范围：**只覆盖 agent 运行时内核**——agent loop、tool 调度、消息/事件流、取消与中断、重试与 429、权限模型、上下文管理、会话持久化、goal/plan 状态、子代理运行时。
> 明确不覆盖：`packages/desktop`（Electron）、`packages/web`、`packages/server`、`packages/ui` 的 UI 实现；`office-runtime`、`documents-plugin`、`spreadsheets-plugin`、`presentations-plugin`、`pdf-plugin` 的文档能力本身（只在「不要抄」中作为边界证据出现）。
> 调研时间：2026-10-24。代码为当时的检出状态，未修改任何文件。

---

## 目录

- [0. 阅读约定与证据分级](#0-阅读约定与证据分级)
- [1. 总览：运行时分层与状态所有权](#1-总览运行时分层与状态所有权)
- [2. Agent Loop 与 Turn 状态机](#2-agent-loop-与-turn-状态机)
  - [M1 Turn 状态机（唯一合法转移表）](#m1-turn-状态机唯一合法转移表)
  - [M2 每步重建请求的 model step 循环](#m2-每步重建请求的-model-step-循环)
  - [M3 事件流的单点事实源](#m3-事件流的单点事实源)
- [3. Tool 调度与执行](#3-tool-调度与执行)
  - [M4 拓扑调度 → 并行组 → 有界并发](#m4-拓扑调度--并行组--有界并发)
  - [M5 可暂停的工具 deadline 与 abort 联动](#m5-可暂停的工具-deadline-与-abort-联动)
  - [M6 大结果落盘 + 预览信封](#m6-大结果落盘--预览信封)
  - [M7 工具结果决定「停轮 / 继续」](#m7-工具结果决定停轮--继续)
- [4. 取消与中断](#4-取消与中断)
  - [M8 每轮一个 abort scope，没有进程级总闸](#m8-每轮一个-abort-scope没有进程级总闸)
  - [M9 取消后的持久化收口](#m9-取消后的持久化收口)
- [5. 重试与 429](#5-重试与-429)
  - [M10 结构化失败分类器](#m10-结构化失败分类器)
  - [M11 指数退避 + Retry-After 优先](#m11-指数退避--retry-after-优先)
  - [M12 重试预算档位](#m12-重试预算档位)
  - [M13 两层恢复边界：adapter retry vs core stream recovery](#m13-两层恢复边界adapter-retry-vs-core-stream-recovery)
- [6. 权限模型](#6-权限模型)
  - [M14 分层判定顺序（一个函数看全）](#m14-分层判定顺序一个函数看全)
  - [M15 规则匹配：工具名 + subject 通配](#m15-规则匹配工具名--subject-通配)
  - [M16 Broker 协议：请求/应答的最小协议面](#m16-broker-协议请求应答的最小协议面)
  - [M17 审批竞争、取消与 requestId 生命周期](#m17-审批竞争取消与-requestid-生命周期)
  - [M18 Plan mode 只读约束与计划文件连续性](#m18-plan-mode-只读约束与计划文件连续性)
  - [M19 会话级 allow 规则](#m19-会话级-allow-规则)
- [7. 上下文管理](#7-上下文管理)
  - [M20 auto compact 阈值：窗口先扣输出预留](#m20-auto-compact-阈值窗口先扣输出预留)
  - [M21 microcompact：先清旧 tool result，不动消息链](#m21-microcompact先清旧-tool-result不动消息链)
  - [M22 compact 产物：边界 + 摘要 + 保留段 + 恢复提醒](#m22-compact-产物边界--摘要--保留段--恢复提醒)
  - [M23 continuity：跨压缩续接的四件套](#m23-continuity跨压缩续接的四件套)
  - [M24 上下文占用统计（分类 + 置信度）](#m24-上下文占用统计分类--置信度)
  - [M25 输出 token 预留与截断续写](#m25-输出-token-预留与截断续写)
- [8. 会话持久化、恢复、fork、排队](#8-会话持久化恢复fork排队)
  - [M26 session store 契约](#m26-session-store-契约)
  - [M27 冷恢复：单飞 activation + 历史 hydration](#m27-冷恢复单飞-activation--历史-hydration)
  - [M28 fork：先固定 anchor，再开子会话](#m28-fork先固定-anchor再开子会话)
  - [M29 busy 输入：steer 队列 vs CommandInbox](#m29-busy-输入steer-队列-vs-commandinbox)
- [9. Goal 与 Plan 状态](#9-goal-与-plan-状态)
  - [M30 Goal 状态机 + 完成校验器 + 自动续跑](#m30-goal-状态机--完成校验器--自动续跑)
  - [M31 状态提醒的 deferral（语法安全）](#m31-状态提醒的-deferral语法安全)
- [10. 子代理运行时](#10-子代理运行时)
  - [M32 派生：独立子会话 + 工具面裁剪 + 权限模式继承](#m32-派生独立子会话--工具面裁剪--权限模式继承)
  - [M33 结果回传：通知报文 + 用量聚合](#m33-结果回传通知报文--用量聚合)
- [11. 不要抄](#11-不要抄)
- [12. 迁移优先级与成本矩阵](#12-迁移优先级与成本矩阵)
- [13. 未找到 / 无法确证](#13-未找到--无法确证)
- [14. 附录：证据文件索引](#14-附录证据文件索引)

---

## 0. 阅读约定与证据分级

**路径前缀**（下文一律省略，便于阅读）：

| 简写 | 实际路径 |
| --- | --- |
| `P/` | `clones/zcode/apps/zcode-cli/packages/` |
| `D/` | `clones/zcode/apps/zcode-cli/docs/design/v2/` |
| `Z/` | `clones/zcode/`（仓库根） |

例：`P/core/src/compact/policy.ts:9` = `clones/zcode/apps/zcode-cli/packages/core/src/compact/policy.ts` 第 9 行。

**证据分级**：

- **源码确证**：直接读到实现代码，行为可从代码推出。
- **文档推断**：来自 `D/` 下的设计 spec（`docs/design/v2` 是该仓的合同层，AGENTS.md 要求「新增或修改行为前先更新对应 spec」），说明设计意图；**个别 spec 已落后于实现**（例如 `D/loop/tool-calling-loop.md:65` 还写着「Permission Check 暂跳过」，而实现早已接入权限服务），凡冲突一律以源码为准，本报告对这类条目额外标注。
- 未找到证据的结论一律写「未找到」，不做补全。

**本报告的口径**：所有「解决了什么问题」都落在一个**办公场景**（Word/PPT/Excel/PDF 的真实编辑、格式转换、知识库检索、多文件交付），不写「帮程序员改代码」。

---

## 1. 总览：运行时分层与状态所有权

ZCode 的 agent 运行时不是「一个大 loop 文件」，而是四层，状态所有权很清楚：

```text
        ┌───────────────────────────────────────────────────────────┐
 协议层 │ bootstrap/zcode-protocol-v4  (CommandInbox / gateway)      │  命令准入、幂等、投影
        └──────────────────────────┬────────────────────────────────┘
                                   │ RuntimeCommand（prompt / steer / notification）
        ┌──────────────────────────▼────────────────────────────────┐
 运行时 │ core/runtime  (AgentRuntime + methods/*)                    │ turn 生命周期、compact、
        │  turn.ts → turn-loop.ts → turn-model-step.ts → turn-tools.ts│ 权限闸门调度、事件写入
        └───────┬───────────────────────────────┬───────────────────┘
                │                               │
        ┌───────▼────────────┐        ┌─────────▼──────────────┐
 能力层 │ core/tool           │        │ core/permission         │  工具执行/调度/超时
        │  scheduler,executor │        │  service, broker, rules │  判定/审批/规则
        └───────┬────────────┘        └─────────┬──────────────┘
                │                               │
        ┌───────▼───────────────────────────────▼──────────────┐
 事实层 │ contracts（端口 + 事件 + schema） ← adapters（sqlite、模型、FS）│
        └───────────────────────────────────────────────────────┘
```

关键设计取向（后面每条机制都服务于它）：

1. **单一事实源**：turn 的每个状态变化都落成 `SessionEvent`（`P/contracts/src/events/session.events.ts:84`），UI/恢复/诊断都读事件，不读内存快照。
2. **声明式能力**：工具自己声明 `readOnly / destructive / sideEffectScope / riskLevel / timeout / alwaysAsk`，权限、调度、并发、超时全部读声明，不在调用点临时猜（`P/core/src/tool/types.ts`、`P/core/src/permission/service.ts:49`）。
3. **长程优先**：不设 tool call 次数硬停（`D/loop/tool-calling-loop.md:3`），边界交给 token/context compact、用户取消、权限拒绝、工具超时、输出截断、provider 重试上限。

> **对办公的意义**：办公任务的形状是「一次请求 → 十几个工具调用 → 生成 3 个文件 → 用户中途改需求」。上面三条恰好对应：可恢复（事件流）、可审批（声明式权利）、可长跑（压缩 + 取消）。这三点比「loop 怎么写」重要得多。

---

## 2. Agent Loop 与 Turn 状态机

### M1 Turn 状态机（唯一合法转移表）

- **证据**
  - `P/core/src/agent/turn-state.ts:27-38`：10 个 phase 的枚举（`idle / processing_input / awaiting_model_response / streaming / scheduling_tools / executing_tools / aggregating_results / awaiting_permission / completing / error`）。
  - `P/core/src/agent/turn-state.ts:233-265`：`canTransitionTo` 里**显式列出**每条合法边；例如 `awaiting_permission → [executing_tools, error]`，`aggregating_results → [awaiting_model_response, scheduling_tools, completing, error]`。
  - `P/core/src/agent/turn-machine.ts:92-104`：`transition()` 非法转移直接抛 `InvalidTurnPhase`（recoverable），**不静默降级**。
  - `P/core/src/agent/turn-state.ts:151-158`：`TurnResultType` 里 `cancelled` 是**正常结束**（注释：用户中断走 TurnComplete 上报，不走 TurnError）。
- **解决的问题（办公）**：一份 30 页 PPT 的生成过程会长时间停在「等审批（要不要覆盖原文件）」和「执行工具（7 个 slide 分别写入）」之间来回；状态机把「现在到底卡在哪」变成可查的枚举，前端才能给出「正在等待你确认…」而不是转圈。
- **迁到 gaea 落点**：`internal/gaea/agent/turn_state.go`（新增 phase 常量 + 转移表），`agent_run.go` 的 run 主链按 phase 打点；`internal/gaea/event` 每个 phase 变化发一条事件。gaea 现有 `agent/agent_run.go`（570 行）与 `agent/turnstate_race_test.go` 已证明有并发 turn 状态，可直接在其上收敛为显式转移表。
- **成本 / 风险**：**S**。风险是历史代码里存在「隐式阶段」（例如压缩中、等权限中混在一个 run 里），补表时会暴露若干既有非法转移，需要允许临时 `error` 边。
- **证据强度**：源码确证。

### M2 每步重建请求的 model step 循环

- **证据**
  - `P/core/src/runtime/methods/turn.ts:93-131`：`executeTurn` 不直接跑循环，而是把输入封装成 `PromptRuntimeCommand` 入队（`priority: "next"`），由 active loop 消费——**外部输入与内部续跑走同一条路**。
  - `P/core/src/runtime/methods/turn-loop.ts:53-276`：真正的 `while (true)`；每轮依次做：`throwIfTurnAborted` → drain 运行中插入的命令 → `microcompactIfNeeded` → `autoCompactIfNeeded` → `initializeMcp` + 刷新工具 → 逐个 `commitTurnRequestEntries` 追加 reminder（plan 退出、runtime mode、todo、output style、tool search 公告）→ `prepareRuntimeProviderRequest` → `runModelBackedTurnStep`。
  - `P/core/src/runtime/methods/turn-loop.ts:117-121`：compact 后立刻 `recordCompactSuccess` 并再次 `throwIfTurnAborted`，**压缩本身也是可取消点**。
  - `P/core/src/runtime/methods/turn-loop.ts:244-249`：每次 model step 都重新 `startModelRequest(...)`，把**本次**实际 messages 写入 turn state（不是复用上一轮）。
- **解决的问题（办公）**：同一轮对话里第 3 次调用模型时，上下文已经和第一次完全不同（多了 5 个工具结果、2 条提醒）。「每步重建 + 每步可压缩 + 每步可取消」让「读 80 页标书再写摘要」这种 20+ 步的任务不会因为中途一次上下文爆掉就整轮失败。
- **迁到 gaea 落点**：`internal/gaea/agent/agent_run.go` 的循环体拆成 `prepareStep()`（压缩 + 提醒 + 工具面刷新）与 `runStep()`；`internal/gaea/context/manager.go` 承担 prefix 重建（gaea 已有 `agent/compile.go` 做 compile）。
- **成本 / 风险**：**M**。风险是把 reminder 注入点从「集中拼接」改成「每步提交」，容易与 gaea 现有的 prefix 缓存（`agent/compile_cache_test.go`、`agent/prefix_stability_test.go`）打架，必须同步做缓存失效标记。
- **证据强度**：源码确证。

### M3 事件流的单点事实源

- **证据**
  - `P/contracts/src/events/session.events.ts:84-184`：`SessionEventType` 全集，含 `turn_steer_*`、`model_*`、`tool_call_*`、`permission_requested/resolved/denied`、`compact_*`、`microcompact_boundary`、`target_*`、`subagent_*`、`checkpoint_created`。
  - `P/contracts/src/events/session.events.ts:124-129`：流恢复专用事件 `stream_recovery_anchor_created / started / anchor_selected / tail_discarded / retry_started / blocked`——**恢复过程本身是一等事件**，不是日志。
  - `P/core/src/runtime/methods/context-usage.ts:248-251`：快照里显式带 `warnings`，说明「当前 token 是本地估算、不是 provider count」。
- **解决的问题（办公）**：办公交付物要「可解释」——用户问「为什么这份合同摘要少了第三章」，能沿 `tool_call_scheduled → tool_call_result → model_request → compact_boundary` 复现。事件分类越细，售后成本越低。
- **迁到 gaea 落点**：`internal/gaea/event/event.go`（266 行，已有事件骨架）补齐 `tool_batch_complete / permission_resolved / compact_boundary / stream_recovery_*`；Wails 前端订阅同一事件流，禁止前端自行拼状态。
- **成本 / 风险**：**M**。风险是事件膨胀导致持久化写放大；ZCode 用「事件 + 投影」两层（`P/bootstrap/src/zcode-protocol-v4/product-projection.ts`）分离，gaea 需要同样一条投影层。
- **证据强度**：源码确证。

---

## 3. Tool 调度与执行

### M4 拓扑调度 → 并行组 → 有界并发

- **证据**
  - `P/core/src/tool/scheduler.ts:60-83`：`schedule()` = 拓扑排序 → 分并行组 → 环检测，输出 `{items, parallelGroups, executionOrder}`。
  - `P/core/src/tool/scheduler.ts:85-103`：能否并行的判定顺序是 `destructive → false`、`concurrentSafe` 显式优先、`readOnly → true`、`sideEffectScope === "none" → true`。
  - `P/core/src/tool/scheduler.ts:48`：`DEFAULT_MAX_CONCURRENCY = 10`；`:187-189` 组满即切组。
  - `D/loop/tool-calling-loop.md:365-367`：调度结果是**执行契约**，runtime 必须把 `parallelGroups` 交给 executor，不得退回「一次性全并行」。
- **解决的问题（办公）**：一次「把这三份 Excel 的汇总页合成一个 PPT」会同时产生 3 个只读读取 + 1 个写文件；只读并行、写入串行、写入排在有依赖的读取之后——既快又不会出现「两个工具同时改同一个 pptx」。
- **迁到 gaea 落点**：`internal/gaea/agent/batch_executor.go`（388 行，已有批次执行）+ `task_writepaths.go`（写路径归类）；建议新增 `agent/schedule.go` 输出与 ZCode 同形的 `parallelGroups`，供 `batch_executor` 消费。
- **成本 / 风险**：**S~M**。风险：gaea 的办公工具里有「读出内容 → 内存里改 → 写回」的复合工具，声明 `readOnly=false` 但内部含读，调度粒度需要按工具整体算。
- **证据强度**：源码确证。

### M5 可暂停的工具 deadline 与 abort 联动

- **证据**
  - `P/core/src/tool/executor/timeout.ts:10-18`：`ToolDeadline` 的设计说明——**模型请求排队时暂停计时、拿到票再续，剩余时长守恒**；理由是超时守的是「provider 挂了」而不是「我们自己的队列长」。
  - `P/core/src/tool/executor/timeout.ts:181-204`：`resolveTimeoutMs` 支持 `kind:"none"`（无超时）、`defaultMs`、工具自报 `resolveTimeoutBudgetMs`、调用方 `timeout_ms` 覆盖、`maxMs` 封顶、`cleanupGraceMs` 宽限。
  - `P/core/src/tool/executor/timeout.ts:206-229`：`linkAbortSignal` 把父 signal 连到子 controller（父子取消链）。
  - `P/core/src/tool/executor/timeout.ts:133-150`：超时错误带 `queuedMs`（区分「慢」和「排队」）并 `abortController.abort(error)`，**先让工具自己清理再 reject**。
  - `P/core/src/tool/executor/call-runner.ts:460-467`、`:556-561`：每次调用建独立 controller + deadline，真正执行走 `executeWithTimeout`。
- **解决的问题（办公）**：LibreOffice 转 PDF 可能 3 分钟，知识库检索可能 20 秒，模型请求可能被限流排队 40 秒。三种「慢」必须区别对待：转换给长超时 + 可取消，检索给短超时，排队时间不算工具超时——否则一次限流会把所有工具的预算烧光。
- **迁到 gaea 落点**：`internal/gaea/agent/execute_one.go:21`（`executeOne`）外层加 `toolDeadline`；`internal/gaea/proc`（子进程）与 `internal/gaea/spill` 侧接收 `ctx` 取消；`internal/gaea/tool/tool.go` 的 Tool 接口补 `TimeoutSpec`。
- **成本 / 风险**：**M**。风险：Go 侧 `exec.CommandContext` 的取消只 kill 进程本身，LibreOffice 常有子进程树，需要进程组 kill；ZCode 靠 `entry.cancellation?.cleanup` 声明清理策略（`timeout.ts:139-141`），gaea 需要等价物。
- **证据强度**：源码确证。

### M6 大结果落盘 + 预览信封

- **证据**
  - `P/core/src/tool/result-persistence-format.ts:1-5`：`<persisted-output>` 标签 + **独立预览预算 2000 字符**，注释明确「artifact 策略用 resultBudget 当落盘触发，落盘后用独立预览预算控制 provider 可见片段」。
  - `P/core/src/tool/result-persistence-format.ts:29-42`：信封内容 = 原始字节数 + 完整路径 + 前 N 字符预览。
  - `P/core/src/tool/result-persistence-format.ts:44-48`：`isPersistedOutputContent` 可反查，避免二次包装。
  - `P/core/src/tool/executor/result-serialization.ts`（369 行）与 `P/core/src/tool/executor/result-content-projection.ts`（309 行）是这套信封的消费侧。
- **解决的问题（办公）**：读一份 200 页 PDF 的抽取文本可能是 300KB；直接回灌上下文会当场爆窗。落盘 + 2000 字预览 = 模型知道「有这么个东西、在哪、大概什么内容」，需要细节时再 `Read` 指定页。
- **迁到 gaea 落点**：`internal/gaea/spill/spill.go`（38 行，已有落盘雏形）+ `internal/gaea/agent/spill.go`；把 `<persisted-output>` 语义固化为 `tool.Result` 的一个变体（`internal/gaea/tool/envelope.go`，90 行）。
- **成本 / 风险**：**S**。风险：办公场景常见「大二进制 + 结构化摘要」，落盘必须带**结构化摘要**（页数、表数、sheet 名），否则模型只知道有个大文件却不知道查哪里。
- **证据强度**：源码确证。

### M7 工具结果决定「停轮 / 继续」

- **证据**
  - `P/core/src/tool/executor/turn-control.ts:44-82`：`ExitPlanMode` 被用户拒绝时，**不是**把错误喂回模型让它自我重写，而是 `turnControl.stopTurnAfterResult = true` 直接停轮（注释：拒绝退出计划代表用户要继续讨论）。
  - `P/core/src/tool/executor/turn-control.ts:60-72`：若拒绝理由带专用 `reasonSource: "plan_approval_feedback"`，则升级为 `followUpUserInput`（变成真实 user message），且 tool_result 文案**不承诺**反馈一定跟随（避免 provider 看到不存在的后续消息）。
  - `P/core/src/tool/executor/turn-control.ts:152-166`：`stopTurnOnSuccess` 走**工具 metadata 声明**而非按工具名硬编码——任何「终态工具」（如 `submit_result`）都能复用。
  - `P/core/src/tool/executor/turn-control.ts:20-42`：创建上限（20 个自动化任务）这类**产品边界**也走同一条停轮通道，并给模型一句「不要重试」的隐藏文案。
- **解决的问题（办公）**：用户拒绝「覆盖原 PPT」之后，agent 不该继续尝试绕路（改文件名、换工具）——那在办公场景是灾难。把「某些结果 = 本轮终止」做成声明式，比在 prompt 里写「用户拒绝就停下」可靠得多。
- **迁到 gaea 落点**：`internal/gaea/agent/tool_dispatch.go`（92 行）后在结果里加入 `TurnControl{Stop, Reason, FollowUpUserInput}`；`internal/gaea/agent/stop_gate.go`（139 行）已有停轮闸门，可直接接。
- **成本 / 风险**：**S**。风险：停轮后「已排队的后续工具」必须补 synthetic 结果（见 `D/loop/tool-calling-loop.md:72`），否则下一次请求会出现 dangling tool call。
- **证据强度**：源码确证。

---

## 4. 取消与中断

### M8 每轮一个 abort scope，没有进程级总闸

- **证据**
  - `D/turn-cancellation.md:13-24`：**明确写「没有 process global abort controller」**，因为并发 session、subagent、后台任务不能互相取消；每个 active turn 拥有一个 root cancellation scope；所有 adapter 只收 `AbortSignal`，永不收可写 `AbortController`。
  - `D/turn-cancellation.md:66-72`：runtime 在这些边界检查取消：上下文/模型工作前、每次模型请求前、模型返回后调度工具前、每个工具批次前、工具结果后的后续请求前、**重试 sleep 与权限等待期间**。
  - `D/turn-cancellation.md:78-86`：网络层的取消必须是**传输层取消**（销毁 in-flight HTTP/SSE），不能只是本地标志位。
  - `P/core/src/runtime/methods/turn.ts:182-183`：`createTurnAbortScope(options?.abortSignal)` 每轮造 scope。
- **解决的问题（办公）**：用户一边跑「整理 12 个 Excel」，一边打开另一个会话问「这个公式怎么改」。全局取消闸会让两个任务互相误杀；每轮 scope 才能做到「只停这一个」。
- **迁到 gaea 落点**：`internal/gaea/agent/task.go`（1227 行，多任务）已有每任务 ctx；`internal/gaea/agent/interrupt_tool.go`（74 行）是「中断工具」入口。需要补的是「取消检查点清单」——把 `D/turn-cancellation.md:66-72` 的 6 个边界逐个落到 Go 代码里。
- **成本 / 风险**：**S**。风险：Go 的 `context.Context` 容易在工具实现里被丢弃（`ctx` 未透传），需要 lint 级约束。
- **证据强度**：文档推断（边界清单来自 spec）+ 源码确证（scope 创建）。

### M9 取消后的持久化收口

- **证据**
  - `D/turn-cancellation.md:90-97`：取消后 **不得**留下 provider 可见的 dangling tool call；已打开的 assistant 消息要补 interrupted 标记；pending/running 的 tool part 要 hydrate 成 interrupted 结果。
  - `P/core/src/agent/turn-state.ts:151-158`：`cancelled` 是正常结果类型，不是 error。
  - `P/core/src/runtime/methods/cancelled-stream-persistence.ts`（53 行）专门处理「流被取消时如何落库」。
  - `P/core/src/tool/executor/turn-control.ts:152-166` 的 `stopTurnAfterResult` 与 `P/core/src/runtime/methods/turn-tools.ts` 的合成结果机制共同保证「assistant 的每个 tool call 都有对应 result」。
- **解决的问题（办公）**：用户按了「停止」然后关掉应用，明天打开会话继续问「刚才那个 PPT 改完了吗」——历史里不能出现「模型说要改但没有任何工具结果」的黑洞。
- **迁到 gaea 落点**：`internal/gaea/agent/interrupted_recovery.go`（220 行，已有 interrupted 恢复）+ `persisted_run.go`（93 行）；需要补「取消时给未执行的 tool call 补 synthetic result」这一步。
- **成本 / 风险**：**M**。风险：synthetic result 的文案若写成「被取消」，模型下一轮可能反复重试；ZCode 的做法是区分 skill/工具级 `cancellation.userVisibleMessage`（`timeout.ts:106`）。
- **证据强度**：文档推断 + 部分源码确证。

---

## 5. 重试与 429

### M10 结构化失败分类器

- **证据**
  - `P/adapters/src/model/failure-classifier.ts:81-90`：入口先 `unwrapRetryError`，取出 `statusCode / code / headers / retryAfterMs`，再按**结构化字段**分类，不靠错误文本。
  - `P/adapters/src/model/failure-classifier.ts:183-193`：`429 → ModelRateLimited + retryable`；`:195-205`：`529 → ProviderOverloaded`；`:207-216`：`401/403 → ProviderNotConfigured, retryReason: AuthRefresh, retryable:false`；`:218-245`：`400/422` 里再区分 context_exceeded。
  - `P/adapters/src/model/failure-classifier.ts:386-437`：provider 业务错误码分支（覆盖「HTTP 200 但 body 里带业务错误」的情况），`:427-437` 再次识别 429。
  - `P/adapters/src/model/failure-classifier.ts:308-310`：`isRetryableFailure` 额外把 `Cancelled` 判为不可重试（**取消绝不重试**）。
- **解决的问题（办公）**：办公用户大多用「按量付费 / 高并发受限」的国内模型；限流、超时、内容审核、上下文超限四种失败的处理完全不同（重试 / 续写 / 换模型 / 压缩）。把它们折叠成「请求失败」会让 agent 在审核拒绝时死重试。
- **迁到 gaea 落点**：`internal/gaea/provider`（provider 抽象）+ `internal/ai`；建议新增 `internal/gaea/agent/failure.go`：`ClassifiedFailure{Code, Reason, RetryReason, Retryable, RetryAfterMs, StatusCode}`，与 ZCode 同形。
- **成本 / 风险**：**M**。风险：gaea 走多层 provider（OpenAI 兼容 / Anthropic / 本地），业务错误码表（`failure-provider-business-codes.ts`）需要按实际供应商维护，否则「审核拒绝」会被当成 5xx 重试。
- **证据强度**：源码确证。

### M11 指数退避 + Retry-After 优先

- **证据**
  - `P/adapters/src/model/runner-retry.ts:38-56`：`calculateRetryDelay` = `base * factor^(attempt-1)`，封顶 `maxDelayMs`；**若 Retry-After 合理则直接用它**；有 jitter 时按 `0.5 + rand*0.5` 抖动。
  - `P/adapters/src/model/runner-retry.ts:15`：`MAX_REASONABLE_RETRY_AFTER_MS = 5 * 60_000`，注释说明「旧 60s 上限会把合法限流等待退化为本地短退避」。
  - `P/adapters/src/model/failure-inspection.ts:286-300`：`parseRetryAfterMs` 先读 `retry-after-ms`，再读 `retry-after`；`:290-292` 注释：**部分 provider 同时返回 `retry-after` 与 `x-should-retry:false`，此时 retry-after 只能作诊断、不能驱动等待**。
  - `P/adapters/src/model/retry-policy.ts:13-30`：默认 10 次重试、2s 起、×2、封顶 60s、jitter 开。
  - `P/adapters/src/model/runner-retry.ts:81-108`：重试决策单独打一条 `model.retry.delay.resolved` 日志，带 `retryAfterHeader / retryAfterSource / xShouldRetryHeader`——**限流问题可复盘**。
- **解决的问题（办公）**：办公使用高峰（上午 9 点整层楼都在生成周报）会常态撞 429。尊重 Retry-After 能把「每 2 秒重试一次直到 10 次全废」变成「等到 provider 说的 40 秒后一次成功」。
- **迁到 gaea 落点**：`internal/gaea/netclient` 或 `internal/ai` 的 retry 中间件；把「Retry-After 与 x-should-retry 冲突时的优先级」写成注释与测试（这是最容易踩的坑）。
- **成本 / 风险**：**S**。风险：无限等待 Retry-After 会让 UI 假死；ZCode 用 5 分钟上限兜住，gaea 需同时给出「正在等待限流恢复（预计 40s）」的 UI 事件。
- **证据强度**：源码确证。

### M12 重试预算档位

- **证据**
  - `P/adapters/src/model/retry-budget.ts:4-13`：三档说明——`unbounded`（只放宽「放弃条件」，**不动退避曲线/分类/Retry-After 优先级**）、`single-attempt`（把本次 `maxAttempts` 收敛为 1）、`default`。
  - `P/adapters/src/model/retry-budget.ts:32-57`：三处闸门统一从这里读，避免「attempt 循环 / 失败闸门 / 空补全重试 / 状态事件」四处各写一遍。
  - `P/adapters/src/model/retry-budget.ts:15-18`：`UNBOUNDED_RETRY_MAX_ATTEMPTS = 0` 作为「无上限」哨兵（Infinity 不可序列化）。
- **解决的问题（办公）**：用户点了「生成 40 页方案」这类长任务时，网络抖动不该让整轮失败（unbounded）；而「快速预览」这类前台小任务，用户等不起 10 次退避，应该 1 次失败就换成降级模型（single-attempt）。
- **迁到 gaea 落点**：`internal/gaea/agent/task_retry_gate.go`（144 行，已有重试闸门）+ 任务级的 `RetryBudget` 字段；Wails 侧把档位暴露成「长任务/快速」两档即可，不必三档全开。
- **成本 / 风险**：**S**。风险：`unbounded` 在办公场景可能撞「用户以为卡死」；ZCode 靠持续的重试状态事件喂 UI，gaea 必须同步。
- **证据强度**：源码确证。

### M13 两层恢复边界：adapter retry vs core stream recovery

- **证据**
  - `D/model/stream-error-retry.md:53-60`：**adapter 只重试「未越过可见边界」的失败**；重试期间暂存并丢弃无副作用的流式前奏（`start / text_start / reasoning_start / tool_input_*` 等）；空的 `reasoning_delta`（含 Anthropic 签名）也属于可丢弃前奏。
  - `D/model/stream-error-retry.md:98-105`：越过边界后的恢复表格——非空 `text_delta` 属「不完整 tail，恢复时必须丢弃」；**工具已执行且结果已提交** 是 provider-safe anchor，恢复请求必须保留该组；**工具执行状态未知或有副作用时不得重复执行**，必须写 synthetic interrupted 结果。
  - `D/model/stream-error-retry.md:83-90`：恢复对象是**模型语义状态**，不是 SSE 连接（明确不做 `Last-Event-ID` 续接，理由：provider 侧可能已继续生成或丢弃内部状态）。
  - `P/contracts/src/events/session.events.ts:124-129`：`stream_recovery_anchor_created / tail_discarded / retry_started` 等事件把恢复过程显式化。
  - `D/model/stream-error-retry.md:55`：状态事件用 `attempt/maxAttempts`，UI 层换算成 `1/N`——**协议字段与展示文案分离**。
- **解决的问题（办公）**：生成 40 页方案到第 25 页断流。直接整轮重来会造成「已经写入 25 页」和「重写 40 页」的重复劳动甚至冲突；从最近的 provider-safe anchor（上一个成功写入的工具结果）重新发起，既省 token 又不会写坏文件。
- **迁到 gaea 落点**：`internal/gaea/agent/streaming`（流式处理）+ `internal/gaea/agent/tool_coherence.go`（87 行，工具结果与 assistant tool_call 配对）；需要新增「anchor 记录 + tail 丢弃」；`internal/gaea/agent/output_continue.go`（36 行）可承接「截断续写」。
- **成本 / 风险**：**L**。风险最高的一条：办公工具多为**有副作用**（写文件），「状态未知不得重复执行」需要工具级幂等键或「写前快照」，否则恢复即污染。建议先只对只读工具做恢复，写工具一律保守终止。
- **证据强度**：文档推断（spec 详尽）+ 源码确证（事件类型、配对逻辑存在）。

---

## 6. 权限模型

### M14 分层判定顺序（一个函数看全）

- **证据** — `P/core/src/permission/service.ts:101-266`，顺序即优先级：

  | # | 分支 | 行号 | 结果 |
  | --- | --- | --- | --- |
  | 1 | `EnterPlanMode/ExitPlanMode` 转移特判 | `:108-114` | allow/deny（不进审批） |
  | 2 | `requiresUserInteraction` | `:116-132` | deny（若在 disallowed）否则 ask |
  | 3 | Guarded + 工具 `userApprovalRule` | `:136-146` | ask，`approvalMode:"user-once"`（**审批不能豁免**） |
  | 4 | 工具自报 `alwaysAsk` | `:149-151` → `:368-427` | 先过强阻断（auto/ disallowed / project deny），再 session allow，再 ask |
  | 5 | `yolo` / `guarded` 且非 plan | `:153-155` | allow（`mode.yolo`） |
  | 6 | `auto` 模式 | `:157-164` | **deny**（「保留但未实现」） |
  | 7 | `disallowedTools` | `:166-173` | deny |
  | 8 | 项目规则 deny → ask | `:175-191` | deny / ask |
  | 9 | 特定网络工具预批 | `:195-209` | allow |
  | 10 | plan 模式 | `:211-213` → `:447-489` | 只读 allow，其余 deny |
  | 11 | 项目规则 allow | `:215-222` | allow |
  | 12 | WebFetch 预批 URL / workflow 草稿 | `:224-250` | allow |
  | 13 | `config.allowedTools` | `:252-259` | allow |
  | 14 | `edit` 模式特判 | `:261-263` → `:561-575` | 文件编辑放行 |
  | 15 | `build` 模式兜底 | `:265` → `:495-559` | 只读/低风险放行，`critical` 必 ask |

  - `P/core/src/permission/service.ts:134-135`：**独立 Plan 优先于 Guarded 的 YOLO**——「审批不能解除 Plan 的执行限制」。
  - `P/core/src/permission/service.ts:682-700`：每次判定都返回结构化 `ruleId + reason + riskLevel + sideEffectScope + escalated`，供事件和 UI 直接使用。
- **解决的问题（办公）**：「删除第 7 页」「覆盖原文件」「调用 LibreOffice」「上传到知识库」四件事的风险完全不同。用一条有序判定链 + 稳定 `ruleId`，才能在办公场景做出「删页要问、读页不问、覆盖要问、另存不问」的细腻体验，而不是「全问」或「全不问」。
- **迁到 gaea 落点**：`internal/gaea/permission/permission.go` 已有 `Policy.Decide`（`:114`）、`Gate.Check`（`:257`）、`Approver`（`:228`）；落点是把「单层 Decide」升级为「有序链 + 规则号」，并补 plan 分支（gaea 的 `agent/plan_mode.go` 166 行已有 plan 概念）。
- **成本 / 风险**：**M**。风险：判定链的顺序本身就是安全语义，迁移时若顺序错位（例如把 `allowedTools` 提到 project deny 之前）会造成提权；必须为每条边写测试。
- **证据强度**：源码确证。

### M15 规则匹配：工具名 + subject 通配

- **证据**
  - `P/core/src/permission/service.ts:318-333`：`ruleSubjects` 从 input 里**按固定 key 顺序**取 subject：`command / url / file_path / path / pattern / patch_text`；WebFetch 特殊走 domain。
  - `P/core/src/permission/service.ts:342-355`：`ruleContent` 支持三种匹配——`prefix:*`（按空格/tab 边界，防 `rm -rf/` 越界）、含 `*` 走通配转正则、否则全等。
  - `P/core/src/permission/rule-matching.ts:1-22`：`wildcardToRegExp` 锚定 `^...$`；`webFetchRuleSubjects` 把 URL 归一成 `domain:host`。
  - `P/core/src/permission/service.ts:299-302`：`Write` 工具可被 `Edit` 规则命中（等价类）。
- **解决的问题（办公）**：办公场景的「总是允许」不是「总是允许写文件」，而是「总是允许写 `D:\项目\XX\*`」或「总是允许导出到 PDF」。subject 归一 + 前缀边界，才能既好用又不放大权限。
- **迁到 gaea 落点**：`internal/gaea/permission/permission.go:171`（`subjectKeys`）与 `:199`（`matchGlob`）已存在；需要补「前缀边界语义」与「等价工具类」两处，并把 subjectKeys 扩到办公字段（`sheet`、`slide_index`、`target_path`）。
- **成本 / 风险**：**S**。风险：办公工具参数形态多（路径可能是相对/绝对/UNC），subject 归一不彻底会让「总是允许」失效或过宽。
- **证据强度**：源码确证。

### M16 Broker 协议：请求/应答的最小协议面

- **证据**
  - `P/contracts/src/interfaces/permission.port.ts:54-72`：`PermissionBrokerRequest` 带 `requestId / sessionId / turnId / traceId / toolCallId / toolName / input / mode / ruleId / reason / riskLevel / sideEffectScope / suggestedPermissionUpdates / optionsPolicy / requestedAt`。
  - `P/contracts/src/interfaces/permission.port.ts:74-97`：`PermissionBrokerResult` 支持 `decision`（allow/deny/modify）、`reason`、`modifiedInput`、`permissionUpdates`（落盘规则）、`sessionPermissionUpdates`（仅内存）、`inputAdjustments`（用户在确认窗里改的参数）。
  - `P/contracts/src/interfaces/permission.port.ts:45-52`：`PermissionOptionsPolicy` 收窄选项集——`no-always-allow`（每次代码都不同的工具，如脚本生成）、`session-always-allow`。
  - `P/contracts/src/interfaces/permission.port.ts:35-39`：`PermissionUpdate = { type:"addRules", behavior, rules[] }`——**「以后都允许」就是一条可复用的规则数据**，不是布尔开关。
  - `P/contracts/src/interfaces/permission.port.ts:82-87`：session 授权「永不被持久化」的注释。
- **解决的问题（办公）**：确认窗不只是「允许/拒绝」。用户看到「将把 `汇总.xlsx` 的 Sheet2 覆盖」时需要能：改目标 sheet、只允许这一次、以后都允许这个目录。这三件事在协议上分别是 `inputAdjustments`、`approvalMode:"user-once"`、`permissionUpdates`。
- **迁到 gaea 落点**：`internal/gaea/agent/request_permission.go`（132 行）已有请求路径；Wails 侧 binding 需要升级为结构化请求体（现在若只传 `tool + args` 字符串则不足），并新增 `inputAdjustments`（办公场景强烈需要：改页码/改 sheet/改输出路径）。
- **成本 / 风险**：**M**。风险：`inputAdjustments` 需要工具显式支持「应用调整」，否则协议字段存在但无效（ZCode 靠 `applyInputAdjustments` 声明，`permission-flow.ts:317-325`）。
- **证据强度**：源码确证。

### M17 审批竞争、取消与 requestId 生命周期

- **证据**
  - `P/core/src/tool/executor/permission-request.ts:34-121`：一次 ask 内部并发三方——broker 应答、`PermissionRequest` hook 应答、取消信号，用 `racePermissionResponders` 决出唯一胜者；`:48` 注释解释「先发布会让立即响应早于 deferred 登记」的根因。
  - `P/core/src/tool/executor/permission-request.ts:122-137`：`Promise.race([Promise.all([published, outcome]), cancellation])`——**发布失败/取消都不等慢 IO**；`finally` 里 abort 并解绑。
  - `P/core/src/permission/broker.ts:64-133`：pending map + `settled` 幂等 + `abort` 监听 + `timeoutMs`；`:108-126` 超时错误带 `requestId/toolCallId/toolName`。
  - `P/core/src/permission/broker.ts:24-39`：`DenyPermissionBroker` 作为安全默认——没有客户端时**默认拒绝**（guarded 模式下抛配置错误而不是静默放行）。
  - `P/core/src/tool/executor/permission-responder-race.ts`（87 行）与 `P/core/src/tool/executor/permission-input-recheck.ts`（79 行）：hook 改了输入后必须**重新判权限**，防止「先批准 A 后执行 B」。
- **解决的问题（办公）**：用户可能在审批弹窗上停留 3 分钟（去开会了），也可能在 hook/插件里自动回答。审批链路必须：超时可控、重复点击幂等、取消能打断、被改写后的参数重新判权。
- **迁到 gaea 落点**：`internal/gaea/agent/request_permission.go` + `internal/gaea/permission/permission.go:257` 的 `Gate.Check`；需要补三样：`settled` 幂等、超时 + 取消、`modifiedInput` 后重判。
- **成本 / 风险**：**M**。风险：Wails 的一次性 binding 调用容易变成「等待前端返回」的裸阻塞，必须有超时与取消；否则关窗会挂住整个 agent goroutine。
- **证据强度**：源码确证。

### M18 Plan mode 只读约束与计划文件连续性

- **证据**
  - `P/core/src/permission/service.ts:447-489`：plan 模式放行集非常窄——`readOnly && !destructive`、非破坏性 MCP、`allowedInPlanMode && sideEffectScope==="session" && !destructive && !needsApproval`；其余一律 `mode.plan.nonReadOnly` deny。
  - `P/core/src/permission/plan-mode-policy.ts:20-42`：`EnterPlanMode` 无弹窗放行；`ExitPlanMode` **只在 plan 激活时可用**，否则 deny（`mode.plan.exitOnly`）。
  - `P/contracts/src/tools/plan-mode.ts:13`：计划文本上限 `PLAN_MODE_MAX_PLAN_CHARS = 20_000`；`:55-67`：`ExitPlanMode` 输入是 `plan` + 可选 `allowedPrompts`（**执行计划所需的权限类别**）。
  - `P/core/src/tool/handlers/plan-mode.ts:214-222`：退出计划**先把计划写成文件**再退出（`persistApprovedPlanFileBeforeExitPlanMode` → `writeApprovedPlanFile`）。
  - `P/core/src/runtime/helpers/plan-file-continuity.ts:18-28`：计划文件固定落在 `<workspace>/.zcode/plans/plan-<sessionId>.md`；`:86-104`：重读后包成 `plan_file_reference` 提醒（「如与该计划相关且未完成，请继续」）。
- **解决的问题（办公）**：办公里「plan」不是写代码计划，而是**方案大纲**（章节结构、数据来源、图表清单）。让 agent 先出大纲、用户批准、再逐章生成，是办公长文档的标准工艺；把批准过的大纲落成文件，跨压缩/跨会话都还在。
- **迁到 gaea 落点**：`internal/gaea/agent/plan_mode.go`（166 行，已有 plan）+ `internal/gaea/docmd`（文档模型）；建议计划文件落 `workspace/.gaea/plans/plan-<session>.md` 并把「重新注入」接到 gaea 的 `agent/recall_reminder.go`（42 行）附近。
- **成本 / 风险**：**S**。风险：20K 字符上限对办公大纲偏紧（一份 40 页方案大纲可能超）；建议 gaea 提到 40K 或改为「分节计划」。
- **证据强度**：源码确证。

### M19 会话级 allow 规则

- **证据**
  - `P/core/src/permission/service.ts:88-99`：`sessionRules` 是**实例级**内存规则，注释说明「一个实例 = 一个会话，重启/冷恢复/新建会话天然得到一个空实例」——所以不需要任何过期机制。
  - `P/core/src/permission/service.ts:398-407`：session allow 的位置在「阻断分支之后、ask 之前」，且**不看模式**（yolo/plan/build 一致）。
  - `P/contracts/src/interfaces/permission.port.ts:82-87`：session 授权永不被写进持久化。
- **解决的问题（办公）**：用户在「把 12 个 Excel 都导出成 PDF」时会连续面对 12 次同类审批。会话级「以后都允许」+ 不落盘，是既省点击又不留长期后门的最优解。
- **迁到 gaea 落点**：`internal/gaea/permission/permission.go` 的 `Policy` 增加 `sessionAllow []Rule`（内存）；注意 gaea 的 Gate 有 `rememberRule`（`:304`），需确认其是否落盘，若落盘要拆成两档。
- **成本 / 风险**：**S**。风险：「不落盘」与用户预期可能冲突（用户以为永远记住）；UI 需明确文案。
- **证据强度**：源码确证。

---

## 7. 上下文管理

### M20 auto compact 阈值：窗口先扣输出预留

- **证据**
  - `P/core/src/compact/policy.ts:6-14`：常量——默认窗口 200K、**auto compact 输出预留 32K**、preflight 预留 21K、摘要最大输出 20K、buffer 13K、连续失败熔断 3 次。
  - `P/core/src/compact/policy.ts:67-73`：`effectiveContextWindow = contextWindow - min(outputReserve, contextWindow)`，注释：provider 的 context window 是 **input+output 共享**窗口，压缩只能让出输入侧。
  - `P/core/src/compact/policy.ts:84-88`：`threshold = effectiveContextWindow - buffer`。
  - `P/core/src/compact/policy.ts:124-154`：判定的 4 个短路——disabled、消息太少、**熔断**、below/above。
  - `P/core/src/compact/policy.ts:136-144`：连续失败达到 `MAX_CONSECUTIVE_AUTOCOMPACT_FAILURES` 直接 `circuit_breaker`，不再尝试（避免压缩风暴）。
- **解决的问题（办公）**：办公会话经常是「先聊 20 轮需求，再一次性生成长文档」。如果不预留输出窗口，压缩会恰好在「需要写出 8000 字」时触发，导致「刚压完就又被截断」。
- **迁到 gaea 落点**：`internal/gaea/agent/compact.go:106`（`maybeCompact`）/`:155`（`compactIfOver`）/`:249`（`tryOverflowRecovery`）已有骨架；需要补的正是「先扣输出预留再算阈值」和「连续失败熔断」。
- **成本 / 风险**：**S**。风险：gaea 的 `contextview/estimate.go` 是字符估算，若与 provider usage 口径不一致，会出现「本地以为没满、provider 报超窗」——ZCode 用 `tokenSource: estimate|provider_usage` 显式标注来源（`policy.ts:28`、`:103-104`）。
- **证据强度**：源码确证。

### M21 microcompact：先清旧 tool result，不动消息链

- **证据**
  - `P/core/src/compact/microcompact.ts:12-29`：占位符 `[Old tool result content cleared]`、**保留最近 5 个** tool result、最小节省 256 token、阈值比例 0.9、buffer 2000、可清理工具白名单（含 Read/Bash/Grep/…）。
  - `P/core/src/compact/microcompact.ts:102-158`：触发条件（时间型 idle 60 分钟 / 压力型 token 阈值）→ 收集候选组 → **保留最近 N 组，清前面的** → 若节省不足阈值则整体放弃。
  - `P/core/src/compact/microcompact.ts:221-247`：按 **assistant tool-call 分组**收集候选，跳过错误结果（默认）、已清过的、**含 image/video/file 媒体块的结果**（`:258-265`）。
  - `P/core/src/runtime/helpers/compact.ts:105-173`：microcompact 只做 **copy-on-write** 替换被清理的那几条 tool result，**不删消息、不改 session store**（`:157-158` 注释）。
  - `P/contracts/src/events/session.events.ts:173`：独立的 `microcompact_boundary` 事件，不与 full compact 混淆。
- **解决的问题（办公）**：agent 为了「看一下当前 PPT 结构」读了 12 个 slide 的 XML，共占 40K token；这些内容 3 步之后就没用了。microcompact 把它们换成占位符，既省上下文又保留「我读过」的痕迹（模型知道要重读）。
- **迁到 gaea 落点**：`internal/gaea/agent/compress.go`（307 行）+ `prune.go`（74 行）+ `fold_ladder.go`（190 行）已有多级压缩；建议把「按 tool-call 分组保留最近 N 组」这一条作为独立策略层接到 `fold_ladder` 之上，并单独发事件。
- **成本 / 风险**：**S**。风险：办公工具的结果常是**结构化 JSON**（slide 列表、sheet 行列），清成占位符后模型容易「凭空重写」；必须保留**摘要**而不是纯占位符（ZCode 因 coding 场景取舍为纯占位符，办公建议改为 `[已清理: <一句话摘要> <原大小>]`）。
- **证据强度**：源码确证。

### M22 compact 产物：边界 + 摘要 + 保留段 + 恢复提醒

- **证据**
  - `P/core/src/runtime/helpers/compact.ts:72-88`：post-compact 顺序固定为 `context prefix → summary → preserved entries → post-compact reminders`。
  - `P/core/src/runtime/methods/compact-active.ts:554-562`：恢复提醒由四部分组成——agent listing、**plan 文件引用**、**已读文件状态重放**、工具搜索公告。
  - `P/core/src/runtime/methods/compact-active.ts:596-628`：`CompactBoundary` payload 带 `preCompactTokenCount / postCompactTokenCount / truePostCompactTokenCount / summarizedMessageCount / keptMessageCount / preservedSegment / willRetriggerNextTurn`。
  - `P/core/src/runtime/helpers/compact-preservation.ts:13-45`：保留段按 **assistant 起始轮次分组**取最近 N 组，并记录 `anchorMessageId / headMessageId / tailMessageId` 供 replay relink。
  - `D/compact/README.md:29-39`：标准顺序的合同（「先声明边界，再给 summary，再接原始保留消息和恢复上下文」）；`:127-139`：compact 必须进 append-only session log，不能只改内存数组。
- **解决的问题（办公）**：压缩后如果只剩「摘要 + 最后一句话」，agent 会忘记「用户已经否掉了第一版版式」这类关键约束。保留最近一组原始往返 + 结构化的用户约束，是办公文档「越改越对」的前提。
- **迁到 gaea 落点**：`internal/gaea/agent/compact.go`（817 行）+ `compact_summary.go`（219 行）+ `compact_util.go`（99 行）；需要补 `preservedSegment` 与 `willRetriggerNextTurn` 两个字段，否则压缩后无法判断「是否马上又要压」。
- **成本 / 风险**：**M**。风险：保留段与 rewind/fork 的交互复杂（gaea 有 `task_fork_test.go`），索引错位会导致历史错乱。
- **证据强度**：源码确证。

### M23 continuity：跨压缩续接的四件套

这是 ZCode 处理得最细的一块，办公场景价值极高。

- **证据**
  - ① **续接话术** — `P/core/src/compact/prompt.ts:142-164`：压缩后注入的消息模板「This session is being continued from a previous conversation that ran out of context…」，并在 `suppressFollowup` 时追加「**直接继续，不要复述摘要、不要说“我继续”**」。
  - ② **可回溯原始 transcript** — `P/core/src/compact/prompt.ts:146-148`：模板里给出完整 transcript 路径，「需要细节时去读」——摘要是有损的，但**原始记录永远可达**。
  - ③ **已读文件状态重放** — `P/core/src/runtime/helpers/compact-post-reminders.ts:8-51`：把最近读过的文件（默认最多 5 个、单文件 5K token、总 50K token）重新注入；超预算的只给一行「`<path>` 在压缩前读过，内容太大，需要时用 Read」。
  - ④ **计划文件引用** — `P/core/src/runtime/helpers/plan-file-continuity.ts:86-104`：注入「存在计划文件 + 全文 + 若相关且未完成请继续」。
  - 另有 `P/core/src/compact/prompt.ts:119-131`：`formatCompactSummary` 会**剥掉 `<analysis>` 草稿块**，只保留 `<summary>`——草稿不回灌主上下文。
- **解决的问题（办公）**：写 40 页方案时被压缩，agent 必须知道：正在写第几章、已批准的大纲在哪、引用过哪几个数据文件、上一轮用户否掉了什么。四件套分别覆盖「任务连续性」「事实可达性」「素材连续性」「计划连续性」。
- **迁到 gaea 落点**：`internal/gaea/agent/compact.go` 的 summary 注入处 + `agent/prefix_compaction_test.go` 既有测试；新增 `internal/gaea/agent/compact_continuity.go`：`{续接话术, transcript 引用, 已读文件重放, 计划文件引用}` 四段生成器 + 预算裁剪。
- **成本 / 风险**：**M**。风险：重放内容会占掉压缩省下的额度（ZCode 用 maxFiles/maxTotalApproxTokens 三重预算硬约束），gaea 必须同样有预算上限，否则「压缩 → 立刻再压缩」。
- **证据强度**：源码确证。

### M24 上下文占用统计（分类 + 置信度）

- **证据**
  - `P/core/src/runtime/methods/context-usage.ts:158-192`：占用按 7 类统计——System prompt / Meta user context / Skills / Tool prompt / System tool schemas / MCP tool schemas / Messages，并算 `percentTokens`。
  - `P/core/src/runtime/methods/context-usage.ts:234-251`：快照带 `tokenMethod: "estimated"`、`confidence: "low"`、`tokenizer: "zcode.estimateTokens.v1"` 和两条 warnings——**估算就明确标成估算**。
  - `P/core/src/runtime/methods/context-usage.ts:100-117`：尾部 8 条消息的诊断（bytes、block 数、是否含 tool call），用于排查「为什么这次请求这么大」。
  - `P/core/src/runtime/methods/model-token-limits.ts:60-74`：`inputBudgetTokens = contextWindow - maxOutputTokens` 直接进日志。
- **解决的问题（办公）**：办公场景最容易「工具 schema 占了 30% 上下文」——十几个文档工具 + 知识库工具 + MCP。分类统计能把「上下文不够」从玄学变成「Tool prompt 占 28%，砍掉两个不用的工具」。
- **迁到 gaea 落点**：`internal/gaea/context/metrics.go`（184 行）与 `internal/gaea/contextview/estimate.go`（53 行）已有估算；建议对齐分类维度并暴露成前端「上下文构成」面板（这是办公用户最需要的可解释性之一）。
- **成本 / 风险**：**S**。风险：估算与真实 tokenizer 差异；必须以 provider usage 为权威（ZCode 用 `AutoCompactTokenOverride.source` 表达）。
- **证据强度**：源码确证。

### M25 输出 token 预留与截断续写

- **证据**
  - `P/core/src/runtime/methods/model-token-limits.ts:5-15`：普通请求默认 `maxOutputTokens = 32_000`，但**模型声明了值就原样用**（注释：不能把 64K 模型裁成 32K）。
  - `P/core/src/runtime/methods/model-token-limits.ts:17-41`：`resolveModelStepMaxOutputTokens` = `min(baseline, contextWindow - estimatedUsage - 1000)`，即**每步按剩余窗口动态收窄**。
  - `P/core/src/runtime/methods/turn-output-token-continuation.ts:12-18`：截断续写提示「Output token limit hit. Resume directly — no apology, no recap…」 + 最多 **3 次**续写。
  - `P/core/src/runtime/methods/turn-output-token-continuation.ts:27-57`：区分 `length / max_tokens / max_output_tokens / model_context_window_exceeded`；`:44-55` 有长注释解释「成功的 `model_context_window_exceeded` 属于 Continue 而非超窗失败」这一取舍。
  - `P/core/src/runtime/methods/turn-output-token-continuation.ts:70-76`：续写提示在**下一轮请求时被过滤掉**（`queryScope: "output_token_continuation"`），不污染历史。
- **解决的问题（办公）**：生成 8000 字方案必然会撞输出上限。撞了之后如果重写整篇，用户看到两份不一致的稿子；「原地续写 + 不复述」才能得到一篇连续文档。
- **迁到 gaea 落点**：`internal/gaea/agent/output_continue.go`（36 行，已有）+ `agent/finalize_text.go`；需要补「续写次数上限」与「续写提示不落历史」两条。
- **成本 / 风险**：**S**。风险：续写拼接处容易出现重复段落；ZCode 用「不复述、可接半句」的提示压住，但仍建议 gaea 在拼接时做后缀重叠去重。
- **证据强度**：源码确证。

---

## 8. 会话持久化、恢复、fork、排队

### M26 session store 契约

- **证据**
  - `P/contracts/src/interfaces/session-store.port.ts:1099-1112`：`SessionStorePort` 主接口（createSession / createForkedSessionWithMetadata / commitForkBundle / updateSession / softDeleteSession …）。
  - `P/contracts/src/interfaces/session-store.port.ts:808-827`：`session_entries` 的类型化 KV——`target_completion_verification / bash_shell_selection / model_selection / execution_state / user_input_auto_resolution / workspace_checkpoint / workspace_file_rewind / dynamic_workflow_activation`。
  - `P/contracts/src/interfaces/session-store.port.ts:525-568`：`CompactionPart`、`:596-611`：`ContextCompactionTimelinePart`、`:612-623`：`GoalVerificationTimelinePart`、`:624-631`：`SessionForkTimelinePart`——**时间线也是类型化的 part**。
  - `P/adapters/src/storage/session-store/sqlite-session-store.ts`（928 行）与其 `repositories/`（sessions 307 / session-inputs 379 / session-entries 75）是 SQLite 实现。
- **解决的问题（办公）**：办公交付物流转周期长（今天起草、明天改、下周导出），需要「消息 + 工具结果 + 检查点 + 验证记录 + fork 关系」一起持久化且**追加写**，不能靠内存重建。
- **迁到 gaea 落点**：`internal/gaea/db`（SQLite）+ `internal/gaea/agent/session_alias.go`；建议把 `session_entries` 的「类型化 KV」直接借用（办公需要的 `evidence`、`plan`、`checkpoint` 都能挂上去）。
- **成本 / 风险**：**M**。风险：gaea 已有自己的会话表结构，加 typed entries 需要 migration；ZCode 有专门的 migration 目录（`adapters/src/storage/session-store/migrations/`），gaea 应同样先设计 migration。
- **证据强度**：源码确证。

### M27 冷恢复：单飞 activation + 历史 hydration

- **证据**
  - `P/bootstrap/src/zcode-protocol-v4/cold-session-resume.ts:49-82`：`ColdSessionResumeCoordinator` 用 `flights: Map<sessionId, Promise>` 保证**同一会话只恢复一次**（并发订阅 join 同一个 flight），settle 即释放。
  - `P/bootstrap/src/zcode-protocol-v4/cold-session-resume.ts:5-10`：错误分型为结构化 `reasonCode`——`fault.subscribe.sessionNotFound`（真没有）vs `fault.subscribe.resumeFailed`（中途失败），**不靠错误文本分流**。
  - `P/core/src/agent/session-history-hydrator.ts:58`：`hydrateMessageHistoryFromSession` 把持久化消息还原成运行时 history；`:251`：`activeSessionMessages` 按 branch cut / rewind 目标裁剪出**活跃链**。
  - `P/core/src/runtime/methods/resume.ts:61`：`resumeFromStore` 是恢复主链；`:395`/`:419`：恢复后重读 todo 与 target（goal）。
- **解决的问题（办公）**：用户关掉 gaea 再打开，点开上周的方案会话，必须回到「活跃链」而不是「全部历史」；同时两个窗口同时点开同一会话不能重复加载（重复加载会导致两次压缩、两份状态）。
- **迁到 gaea 落点**：`internal/gaea/agent/task.go` 的 resume 路径 + `internal/gaea/context/store_file.go`（104 行）；单飞机制建议放 `internal/gaea/agent/` 的 session loader（gaea 有 `sessionquery` 包可复用）。
- **成本 / 风险**：**S**。风险：单飞失败后必须清 flight（ZCode 在 `finally` 里删），否则一次失败会永久卡住该会话。
- **证据强度**：源码确证。

### M28 fork：先固定 anchor，再开子会话

- **证据**
  - `P/core/src/runtime/methods/stable-fork-boundary.ts:16-92`：在 `TurnComplete` 对订阅者可见**之前**，把 fork 所需的段固定到最终 assistant 的 `anchor`（`orderedMessageIds / boundaryMessageId / historyRoundCount / goalBoundary`）。
  - `P/core/src/runtime/methods/stable-fork-boundary.ts:94-104`：goal 快照**清空 activeRun 字段**（注释：active run 是 parent 的瞬时执行权，不是 child 可继承的状态）。
  - `P/core/src/runtime/methods/session-fork.ts:1061`：`forkStableConversationAtMessage`；`:1189`：`copyGoalStateForFork`；`:1366`：`deriveForkedGoalStatusFromCopiedVerifications`（**fork 后按已复制的验证记录重算 goal 状态**）。
  - `P/core/src/runtime/methods/session-fork.ts:137`：`ForkIdentityMap`（消息/part id 的重映射表），`:219`：`createForkIdentityMap`。
  - `P/core/src/runtime/methods/stable-fork-boundary.ts:53-57`：找不到精确段时**不写伪 anchor**，让 resolver 走 legacy 规则（保守）。
- **解决的问题（办公）**：「这个方案第 3 版不错，从那里再出个精简版」——fork 必须精确到某一轮末尾，且新会话的 goal/证据链要自洽（不能继承「上一轮正在跑」的运行态）。
- **迁到 gaea 落点**：`internal/gaea/agent/task_fork_test.go` 说明已有 fork；需要补 anchor 固定与「goal/证据重算」两步。
- **成本 / 风险**：**L**。风险：id 重映射是长尾 bug 源（消息、part、tool_call、附件都要重映射）；ZCode 用 `ForkIdentityMap` 统一，gaea 若各处手写映射必然漏。
- **证据强度**：源码确证。

### M29 busy 输入：steer 队列 vs CommandInbox

ZCode 这里是**两层**，职责不同，必须分开看：

- **证据（第一层：协议准入 CommandInbox）**
  - `P/bootstrap/src/zcode-protocol-v4/command-inbox.ts:110-218`：`handle()` 用 **key gate → session gate** 双锁（`:142-144` 注释固定锁序），保证同会话不同命令按实际 admission 顺序串行。
  - `P/bootstrap/src/zcode-protocol-v4/command-inbox.ts:288-310`：查询顺序 = in-flight → live input → settled LRU → 四个持久化回源（transcript / timeline / child / discarded）——**幂等查询跨重启有效**。
  - `P/bootstrap/src/zcode-protocol-v4/command-inbox.ts:176-178`：注释「旧单表 LRU 会在 >512 条 churn 时淘汰仍在执行的命令，随后 query 返回 unknown、重试再次执行」——这就是为什么要 pin。
  - `P/bootstrap/src/zcode-protocol-v4/command-inbox.ts:312-448`：`decide()` 做 revision / logEpoch 的 CAS 校验，落后就返回 `stale`，不执行。
  - `P/bootstrap/src/zcode-protocol-v4/v4-gateway.ts:1088`、`:2604`：`pinLiveInput / releaseLiveInput` 的实际调用点。
- **证据（第二层：core 运行中插入 steer）**
  - `D/loop/turn-steering.md:151-161`：**drain 边界**——只能在「assistant tool calls 已入历史 + 所有 tool results 已注入 + 下一次 model request 之前」插入；其余位置一律等。理由：provider 语法不允许 user message 夹在 tool_use 与 tool_result 之间。
  - `D/loop/turn-steering.md:163-172`：pending queue 是**内存态、非恢复态**——resume 不恢复 queued-but-not-drained 输入；崩溃后丢弃。
  - `D/loop/turn-steering.md:92-96`：四个事件 `turn_steer_queued / drained / rejected / discarded`，UI 只做乐观展示，不作为队列事实源。
  - `P/core/src/runtime/command-queue.ts:10`：优先级 `now / next / later`；`:110-116`：六种命令 mode（prompt / target-continuation / task-notification / subagent-message / control-only-turn）。
  - `P/core/src/runtime/command-queue.ts:91-97`：`ControlOnlyTurnRuntimeCommand` 的注释解释「user 消息插不进正在跑的 turn（provider 语法），所以排到 turn 结束后」。
- **解决的问题（办公）**：用户看 agent 正在写第 5 页时插一句「第 3 页的表格换成柱状图」。这句话必须：被接受（不能丢）、不打断当前工具、在下一个安全边界生效、且模型看到的顺序正确。
- **迁到 gaea 落点**：
  - 协议层：`internal/taskinbox/taskinbox.go`（已有输入收件箱）+ `internal/gaea/channels`；补「双锁 + 幂等查询回源 + revision CAS」。
  - 运行时：`internal/gaea/agent/steer_test.go` / `steer_receipt_test.go` 说明已有 steer 回执；把 `D/loop/turn-steering.md:151-161` 的 drain 边界写成不变式测试。
- **成本 / 风险**：**M**。风险：两层容易合并成一层（历史上常见错误），合并后要么丢幂等要么破坏 provider 语法。必须保留「准入 ≠ 消费时机」的分离。
- **证据强度**：源码确证 + 文档推断（drain 边界语义）。

---

## 9. Goal 与 Plan 状态

### M30 Goal 状态机 + 完成校验器 + 自动续跑

ZCode 把「目标」叫 target/goal，是一个**会话级长程目标对象**。

- **证据**
  - `P/contracts/src/tools/target.ts:9-16`：状态只有四态 `active / paused / budget_limited / complete`。
  - `P/contracts/src/tools/target.ts:46-63`：`SessionGoal{objective, summaryTitle, status, tokenBudget, tokensUsed, timeUsedSeconds, activeRunStartedAtMs…}`。
  - `P/contracts/src/tools/target.ts:119-135`：给模型看的 goal 状态，objective 用 `<untrusted_objective>` 包裹并转义——**用户目标文本按不可信数据处理**。
  - `P/contracts/src/tools/target.ts:137-194`：续跑提示包含「完成审计清单」（逐条需求 → 证据；不得用「通过测试/写了计划/花了很多时间」当完成证据；`不确定性 = 未完成`）。
  - `P/contracts/src/tools/target.ts:192`：**「不要自己宣布完成，runtime 会在本轮之后跑校验器」**——模型无权写 complete。
  - `P/contracts/src/tools/target.ts:196-245`：校验器 prompt：只允许输出 `{passed, reason, nextAction}` JSON，不许调工具；`:213-218` 专门处理「问候/致谢类非任务目标」，避免把闲聊判成未完成而无限续跑。
  - `P/contracts/src/tools/target.ts:289-333`：坏 JSON 走 **fail-open**（判 passed=true），理由是「裁判链路故障不该把已交付的 goal 卡死」。
  - `P/core/src/runtime/methods/target-continuation-loop.ts:45-86`：续跑循环——`while(!aborted)`，**若 runtimeCommandQueue 有 pending 就 return**（让位给用户），每轮后 `verifyBeforeContinue = true`。
- **解决的问题（办公）**：办公里「把这 6 个部门的预算表汇总成一个分析报告」不是一轮能完成的。goal 状态 + 独立校验器 + 让位于用户输入，解决了「agent 过早说完成」和「agent 无限空转」两个失败模式。
- **迁到 gaea 落点**：`internal/gaea/agent/complete_subtask.go`（280 行）+ `agent/judge.go`（116 行）+ `agent/task.go` 的目标字段；建议新增 `agent/goal.go`（状态机 + verifier 调用 + 让位规则），并把 verifier 做成**可选模型**（成本敏感场景关掉，退化为「用户确认完成」）。
- **成本 / 风险**：**L**。风险：校验器本身是额外模型调用（成本 + 延迟）；`fail-open` 策略意味着「校验器挂了会误判完成」，办公场景建议改为「fail-open 但标记待人工确认」。
- **证据强度**：源码确证。

### M31 状态提醒的 deferral（语法安全）

- **证据**
  - `P/core/src/runtime/methods/goal-state-reminder.ts:15-25`：若 active turn 处于「goal 提醒 deferral 开启」状态，**先暂存**；注释解释根因：Stop 会先暂停 goal 再 abort 正在执行的工具，若立刻写 history，reminder 会落在 `tool_use` 与 `cancelled tool_result` 之间，**下一次请求违反 provider 语法**。
  - `P/core/src/runtime/methods/goal-state-reminder.ts:35-62`：关闭 deferral 时物化 pending；`:49-61` 物化失败只 warn，不阻断 stop turn 的终态（权威 target 已先落库）。
  - `P/core/src/runtime/methods/goal-state-reminder.ts:64-78`：物化为 `visibility: "model-only"` 的 synthetic user notice（用户看不见，模型看得见）。
- **解决的问题（办公）**：办公里状态提醒特别多（「目标已暂停」「已用 62% 预算」「计划已批准」）。这些提醒插入位置错了会直接导致下一次请求 400，表现为「agent 突然不工作了」。
- **迁到 gaea 落点**：`internal/gaea/agent/tool_coherence.go`（87 行，保证 tool_call/result 配对）附近加一个 `pendingReminders` 队列，只在安全边界 flush；`model-only` 可见性可复用 gaea 现有的消息可见性字段。
- **成本 / 风险**：**S**。风险：延迟物化可能丢提醒；ZCode 用一个显式 deferral 窗口 `open/close` 控制，gaea 照抄成对函数即可。
- **证据强度**：源码确证。

---

## 10. 子代理运行时

### M32 派生：独立子会话 + 工具面裁剪 + 权限模式继承

- **证据**
  - `P/core/src/subagent/runner.ts:811-816`：子会话 id = `subagent_<agentId>`，agentId 独立生成。
  - `P/core/src/subagent/runner.ts:1142-1169`：真正的派生请求字段——`allowedTools`（由 profile + options 解析）、`disallowedTools`、`sessionId`、`maxTurns`、**`permissionMode`**、`systemPrompt`、`workingDirectory / workspaceRoot`、`traceContext`。
  - `P/core/src/subagent/tool-policy.ts:7-31`：所有子 agent **强制剔除** `EnterPlanMode / ExitPlanMode`，理由（`:25-26`）：「子 agent 没有独立的 plan approval 恢复面，暴露 plan tools 会让 ExitPlanMode 等用户确认并卡住父 turn」。
  - `P/core/src/subagent/profile.ts:23-37`：内置 Explore agent 是**只读工具面**（Bash/Glob/Grep/Read/WebFetch/WebSearch/TodoWrite）；`:72-87`：general-purpose 是 `tools: ["*"]`。
  - `P/core/src/subagent/runner.ts:1202-1239`：`createSubagentActivityWatchdog`——子 agent 空闲超过阈值即判 `ToolTimeout` 并 abort（防止子 agent 静默挂死拖住父轮）。
  - `P/core/src/subagent/runner.ts:1151-1153`：`get background()` 用 getter 实时读 registry，而不是启动时快照（前台转后台后事件镜像语义要跟着变）。
- **解决的问题（办公）**：「把这份 100 页 PDF 里的表格全部提取出来」适合派给一个只读子 agent 并行跑，主 agent 继续和用户对话。子 agent 必须：只能读（不能偷偷改文件）、不能弹审批（会卡住父轮）、超时要能被收回。
- **迁到 gaea 落点**：`internal/gaea/agent/subagent_store.go`（906 行）+ `subagent_limit.go`（100 行）+ `subagent_message_test.go`；需要补：子 agent 工具面裁剪（尤其「禁止一切审批型工具」）与 permissionMode 继承的显式字段。
- **成本 / 风险**：**M**。风险：权限委派是最容易越权的地方——子 agent 拿到的 `permissionMode` 若比父宽就是提权；ZCode 用 profile 声明 + 同一 `PermissionService` 实例语义约束，gaea 必须保证「子 ≤ 父」。
- **证据强度**：源码确证。

### M33 结果回传：通知报文 + 用量聚合

- **证据**
  - `P/core/src/subagent/completion-notification.ts:24-46`：完成通知统一走 `formatTaskNotification`，带 `agentId / description / status / error / outputFile / taskId / taskType:"local_agent" / toolUseId / usage`。
  - `P/core/src/subagent/completion-notification.ts:48-60`：失败时 summary 必须**带上与 `<error>` 相同的失败原因**（注释说明这是前端「子智能体输出」卡片的数据来源）。
  - `P/core/src/subagent/runner.ts:1176-1199`：结果 = `childResult.response` 文本 + `totalToolUseCount / totalDurationMs / totalTokens / usage`（从子会话事件聚合，`:2034` `aggregateModelUsage`）。
  - `P/core/src/runtime-task/notification.ts`（397 行）是通知的公共格式层；`P/core/src/runtime/command-queue.ts:66-74` 的 `TaskNotificationRuntimeCommand` 说明通知**以队列命令的形式**回到父 turn（有 `branchGeneration`，rewind 后丢弃）。
- **解决的问题（办公）**：子 agent 跑完不是「把一大段文本塞回主上下文」，而是「一条结构化通知（做了什么、用了多少、结果文件在哪）+ 结果正文」。这样才能既省上下文又可展示进度与成本。
- **迁到 gaea 落点**：`internal/gaea/agent/subagent_message_test.go` 与 `subagent_journal`；建议把通知结构体化为 `SubagentCompletion{Status, Summary, OutputPath, Usage, DurationMs}`，并挂到 `internal/gaea/agent/task.go` 的父任务事件上。
- **成本 / 风险**：**S**。风险：`branchGeneration` 这类「rewind 后丢通知」的机制若缺失，会出现回滚后幽灵任务；gaea 需要等价版本号。
- **证据强度**：源码确证。

---

## 11. 不要抄

这一节列出 **ZCode 为 coding 场景存在、对办公模块是负债** 的东西，逐条给证据与原因。

### N1 终端/bash 为中心的假设

- **证据**
  - `P/core/src/tool/handlers/` 下 **37 个 `bash*` 文件**（`bash.ts` 单文件 564 行），其中 24 个是 `bash-readonly-policy-*.ts`：把 shell 命令解析成 argv / flags / git 子命令 / io 重定向，只为判断「这条命令是不是只读」。
  - `P/core/src/tool/bash-timeout-policy.ts:5-6`：默认 120s / 上限 600s，专为 shell 设计。
  - `P/core/src/permission/service.ts:622-624`：`isDestructiveTool` 只认 `Bash`——整套「破坏性」概念是围绕 shell 建的。
  - `P/core/src/permission/service.ts:318-333`：规则 subject 首位是 `command`。
- **为什么是负债**：办公工具的权利维度是「改哪个文档的哪一部分」（覆盖原文件？改第 7 页？删 sheet？），不是「这条命令是否只读」。把 shell 的只读判定表搬过来，会得到一批**用不上但必须维护**的解析器；而真正需要的「pptx 局部写回是否安全」它一点也覆盖不到。
- **建议**：gaea 保留 `bash` 作为**兜底工具**（且默认高风险、需审批），但权限与超时模型必须按**办公动作**建模（`doc.write_range`、`slide.replace`、`sheet.set_cells`、`file.overwrite`）。

### N2 git 工作流假设

- **证据**
  - `P/adapters/src/context/git-snapshot.ts:28-32`：启动时先判 `insideWorkTree`，非 git 仓库直接跳过。
  - `P/adapters/src/context/git-snapshot.ts:165-168`：把 `git status` 注入 **provider-visible** 上下文，并按 2K 字符截断「与上游 CLI 保持一致」。
  - `Z/AGENTS.md:68-69`：`workspaceIdentity` / `workspacePath` / worktree 状态是仓库级一等概念。
- **为什么是负债**：办公用户的「工作区」是一个文件夹里的 pptx/xlsx/docx，没有 diff、没有分支、没有 commit。注入 git status 只会浪费上下文；而办公真正需要的「版本」语义是**文件级检查点与回滚**（ZCode 里也有 `workspace_checkpoint`，见 `P/contracts/src/interfaces/session-store.port.ts:815`，那部分才该抄）。
- **建议**：抄 checkpoint/rewind（M28 的 anchor 思路），不抄 git 上下文注入。

### N3 Electron 多宿主架构

- **证据**
  - `Z/AGENTS.md:31`：`packages/desktop`：Electron main / host / renderer。
  - `Z/AGENTS.md:62`：Desktop 通过 **stdio** 与 Agent 通信，协议改动要同步 `packages/shared/src/zcode-protocol/index.ts`。
  - `Z/AGENTS.md:68-69`：Host owner/lease、跨 Host 路由、stale run 防护。
  - `P/bootstrap/src/zcode-protocol-v4/v4-gateway.ts:1088`、`:2604`：owner/lease 与 inbox pin 的耦合。
- **为什么是负债**：gaea 是单进程 Wails 应用（Go 后端 + WebView2 前端），没有「多 Host、多窗口、跨进程 owner/lease」问题。把 owner/lease、session residency、stale run 这套搬进来，只会给一个进程内的 HTTP/binding 调用加上一整层不必要的分布式协调。
- **建议**：只抄 `CommandInbox` 的**幂等 + 串行准入**语义（M29 第一层），丢掉 owner/lease 与跨 Host 路由。

### N4 远程 SSH / WSL / 远端 workspace 链路

- **证据**
  - `Z/packages` 根目录存在 `provider-node`、`server`、`zcode-server-cli`、`web`、`web-share`、`android`（多端/远端形态）。
  - `Z/AGENTS.md:64`：「远程 workspace 由窗口内的连接注册表管理，不另建 Desktop Remote Host」；`Z/AGENTS.md:65`：手机远控连接桌面已有 Host attachment。
  - `Z/AGENTS.md:66`：`desktop-continuous` 与 `web-remote-replayable` 两条链路必须同时验证。
  - `P/adapters/src/storage/session-target.ts`（434 行）承担 workspace 目标解析（含远程路径修复），配套 `P/core/src/runtime/helpers/persisted-remote-session-path-repair.ts`（184 行）专门修复远程会话路径。
- **为什么是负债**：办公文件在本地磁盘（或 OneDrive/共享盘），路径语义就是 Windows 路径。远端路径修复、远程会话恢复、传输层差异会带来大量边界代码，收益为零。
- **建议**：只保留「本地绝对路径 + UNC」两种，不做远端 workspace。

### N5 终端 UI / 键位语义

- **证据**
  - `D/turn-cancellation.md:27-46`：大段 `Esc` / 双 `Ctrl-C` 的键位语义合同（含「第一次 Ctrl-C 只提示，第二次才退出」）。
  - `Z/apps/zcode-cli/AGENTS.md`（CLI 包级）：键盘操作优先、鼠标是增益能力。
- **为什么是负债**：Wails 是图形界面，取消/审批靠按钮。把 TUI 的键位状态机搬进 core，会让 core 充满与 UI 形态绑定的分支。
- **建议**：抄「取消是一等结果」的语义（M8 的 6 个检查点），不抄键位。

### N6 多模型/多供应商的加速卡与 off-peak 调度

- **证据**
  - `P/core/src/runtime/off-peak/`（目录）+ `P/core/src/runtime/methods/turn-execution-fallback.ts`（282 行）+ `turn.ts:237-383`（创建模型失败时按候选链回退）。
  - `P/core/src/runtime/methods/turn-model.ts` 的 `resolveExecutionModelRetryBudget` 与 highspeed 卡语义（`P/contracts` 中的 `ModelRetryBudget`）。
- **为什么是负债**：这是为「coding 订阅的高峰限流 + 加速卡营销」服务的复杂度（fallback 声明、单次重试、卡片作废投影）。办公场景只需要「主模型 + 一个降级模型」。
- **建议**：抄 M12 的**预算档位概念**（长任务 vs 快速任务），不抄加速卡/回退候选链/卡片生命周期。

### N7 动态 workflow / 多 agent 编排（graph、amend、fill-hole）

- **证据**
  - `P/core/src/tool/handlers/` 下 `create-workflow*`、`amend-workflow*`、`fill-workflow-hole*`、`get-workflow-run*`、`save-workflow*` 等 20+ 文件。
  - `P/core/src/permission/service.ts:408-445`：为 workflow 修订专门开了「本会话的 run + 非用户停止 + 无新 provider」的免确认规则。
  - `P/dynamic-workflow` 包（含 `engine/replay-order.ts`、`journal-store`）。
- **为什么是负债**：这是「用 JS 脚本编排一批 subagent 跑仓库级任务」的机制。办公场景的编排需求是**文档级流水线**（解析 → 结构化 → 生成 → 校验 → 导出），gaea 已有 `internal/gaea/dag` 与 `docs/gaea-office-dag-63-design-2026-09.md` 的设计，不需要 ZCode 的脚本化 workflow + journal replay。
- **建议**：抄 M32/M33 的**子代理派生与结果回传**，不抄 workflow 脚本引擎。

### N8 自动续跑（goal continuation）+ 完成校验器 的全量启用

- **证据**：M30 全部证据（`P/contracts/src/tools/target.ts:137-194`、`P/core/src/runtime/methods/target-continuation-loop.ts:45-86`）。
- **为什么是（部分）负债**：coding agent 可以「自己判断测试过了没有」；办公交付物的「完成」是**用户说了算**。若默认开启自动续跑 + 模型自评校验器，办公场景会出现「用户已经满意，agent 还在自动改第 8 版」。
- **建议**：抄状态机与「模型无权宣布完成」，**默认关闭自动续跑**，改为「预算驱动 + 用户确认」；仅在用户显式说「做到我说停为止」时开启。

---

## 12. 迁移优先级与成本矩阵

| 优先级 | 机制 | 落点 | 成本 | 主要风险 |
| --- | --- | --- | --- | --- |
| **P0** | M14/M15 权限分层链 + 规则 subject | `internal/gaea/permission` | M | 顺序错位 = 提权；需逐边测试 |
| **P0** | M16/M17 Broker 协议（含超时/幂等/改参重判） | `internal/gaea/agent/request_permission.go` | M | Wails 裸阻塞挂住 goroutine |
| **P0** | M20/M21 压缩阈值（先扣输出预留）+ microcompact | `internal/gaea/agent/compact.go` | S | 估算口径与 provider usage 不一致 |
| **P0** | M23 continuity 四件套 | `agent/compact_continuity.go`（新） | M | 重放预算超支导致「压完立刻再压」 |
| **P1** | M1 Turn 状态机 + M3 事件流补全 | `agent/turn_state.go`、`event` | S~M | 暴露既有隐式阶段 |
| **P1** | M4/M5 调度 + 可暂停 deadline | `agent/schedule.go`、`execute_one.go` | M | 进程组 kill、复合工具语义 |
| **P1** | M6 大结果落盘信封 | `spill` + `tool/envelope.go` | S | 需带结构化摘要 |
| **P1** | M10/M11 失败分类 + Retry-After | `agent/failure.go`（新）、`netclient` | M | 供应商业务码表维护 |
| **P1** | M29 两层排队（准入 ≠ 消费时机） | `taskinbox` + `agent/steer` | M | 合并成一层会退化为丢幂等/坏语法 |
| **P2** | M22/M24 压缩产物结构 + 占用统计面板 | `agent/compact.go`、`context/metrics.go` | M | 与 rewind/fork 交互 |
| **P2** | M25 输出截断续写 | `agent/output_continue.go` | S | 拼接重复 |
| **P2** | M27/M28 冷恢复单飞 + fork anchor | `agent/`（session loader / fork） | S（恢复）/ L（fork） | id 重映射长尾 bug |
| **P2** | M30/M31 goal 状态机 + 提醒 deferral | `agent/goal.go`（新） | L | 校验器成本；fail-open 语义要改 |
| **P2** | M32/M33 子代理派生 + 结构化回传 | `agent/subagent_*` | M | 权限委派必须「子 ≤ 父」 |
| **P3** | M7 turnControl 停轮 | `agent/stop_gate.go` | S | 需配 synthetic result |
| **P3** | M12 重试档位 | `agent/task_retry_gate.go` | S | 长任务「假死」感知 |
| **P3** | M13 两层恢复边界（只对只读工具） | `agent/streaming` | L | 写工具重复执行 = 污染 |
| **P3** | M18 Plan mode + 计划文件 | `agent/plan_mode.go` | S | 20K 上限偏紧 |
| **P3** | M19 会话级 allow | `permission` | S | 与持久化规则混淆 |
| **P3** | M26 session store typed entries | `db` + `agent` | M | migration |

**综合建议的实施顺序**：先做 P0 四条（它们直接决定「办公 agent 是否可被信任」），再做 M29 + M4/M5（决定「是否可用」），最后做 M30/M13（决定「是否高级」）。

---

## 13. 未找到 / 无法确证

诚实记录调研边界：

1. **未找到**「429 的 Retry-After 直接驱动任务级调度（例如把整批任务推迟到 reset 时刻）」的代码。只找到 adapter 级等待（`runner-retry.ts:38-56`）与 retryAfterMs 作为诊断字段向下传递（`runner-retry.ts:132-134`）。
2. **未找到** `CommandInbox` 与 core 的 `RuntimeCommandQueue` 之间的直接调用/映射代码。二者通过 `v4-gateway.ts:1088/2604` 的 pin/release 与队列命令（`command-queue.ts:66-74`）间接衔接，未见单一函数完成转换——这可能是我未定位到 adapter，而非不存在。
3. **未找到**办公类工具（docx/pptx/xlsx）在**权限 subject 归一**上的专门处理。`ruleSubjects`（`service.ts:318-333`）只有 `command/url/file_path/path/pattern/patch_text` 六个 key，文档类工具（如 slide index、sheet name）没有专属 subject。
4. **未找到** microcompact 对「结构化工具结果」的保摘要策略。实现是**纯占位符**替换（`microcompact.ts:250-252`），注释也明确只保留占位符与 id；这对办公结构化结果偏激进（见 M21 风险）。
5. **未找到** 关于 compact 摘要质量的自动评估/回归门禁（找到的是 E2E 与单测：`P/core/tests/runtime-compact.test.ts` 5488 行、`P/bootstrap/tests/compact-e2e-case-utils.test.ts`）。摘要质量问题在 ZCode 里靠 prompt 约束（`prompt.ts:14-109`）而非自动评分。
6. **文档与实现不一致处（以源码为准）**：`D/loop/tool-calling-loop.md:65`、`:226` 仍写「权限检查暂跳过」，实际 `permission-flow.ts:44` 已完整接入。引用该 spec 时需注意。
7. **未验证**：`P/core/src/subagent` 的权限委派是否**运行期**强制「子权限 ≤ 父权限」。代码里子 agent 拿到的是 `profile.permissionMode`（`runner.ts:1159`），未见显式的「取交集」校验函数——若 profile 可被用户/项目配置写成更宽的模式，存在提权面。**这一点建议 gaea 不要照抄，而是显式做交集。**

---

## 14. 附录：证据文件索引

按包分组，便于复核。行号为本报告引用的关键行。

**core/src/agent**
- `turn-state.ts:27-38`（phase 枚举）、`:151-158`（结果类型）、`:233-265`（转移表）
- `turn-machine.ts:92-104`（非法转移抛错）
- `session-history-hydrator.ts:58`（恢复 history）、`:251`（活跃链裁剪）

**core/src/runtime**
- `methods/turn.ts:93-131`（executeTurn 入队）、`:182-183`（abort scope）
- `methods/turn-loop.ts:53-276`（主循环）、`:83-121`（每步压缩）、`:244-271`（每步重建请求）
- `methods/turn-stop.ts:157-236`（无工具调用的收口）
- `methods/turn-output-token-continuation.ts:12-18`、`:27-57`、`:70-76`
- `methods/compact.ts:184`（autoCompactIfNeeded）、`:356`（reactiveCompact）
- `methods/compact-active.ts:554-562`、`:596-628`、`:579-581`
- `methods/context-usage.ts:158-192`、`:234-251`
- `methods/model-token-limits.ts:5-15`、`:17-41`
- `methods/goal-state-reminder.ts:15-25`、`:64-78`
- `methods/target-continuation-loop.ts:45-86`
- `methods/stable-fork-boundary.ts:16-92`、`:94-104`
- `methods/session-fork.ts:1061`、`:1189`、`:1366`
- `methods/resume.ts:61`
- `helpers/compact.ts:72-88`、`:105-173`
- `helpers/compact-preservation.ts:13-45`
- `helpers/compact-post-reminders.ts:8-51`
- `helpers/plan-file-continuity.ts:18-28`、`:86-104`
- `command-queue.ts:10`、`:66-74`、`:91-97`、`:110-116`

**core/src/compact**
- `policy.ts:6-14`、`:67-88`、`:90-155`
- `microcompact.ts:12-29`、`:83-174`、`:221-247`
- `prompt.ts:14-109`、`:119-131`、`:133-165`

**core/src/permission**
- `service.ts:101-266`（判定链）、`:88-99`（session rules）、`:134-135`（plan 优先）、`:299-355`（规则匹配）、`:368-427`（alwaysAsk）、`:447-489`（plan）、`:495-575`（build/edit）、`:682-700`（结构化结果）
- `rule-matching.ts:1-22`
- `plan-mode-policy.ts:20-42`
- `broker.ts:24-39`、`:64-133`

**core/src/tool**
- `scheduler.ts:48`、`:60-103`、`:154-202`、`:233-243`
- `result-persistence-format.ts:1-5`、`:29-48`
- `executor/timeout.ts:10-18`、`:93-179`、`:181-204`、`:206-229`
- `executor/permission-flow.ts:44-333`
- `executor/permission-request.ts:34-137`
- `executor/approval-gate.ts:36-74`
- `executor/turn-control.ts:20-42`、`:44-82`、`:152-166`
- `executor/call-runner.ts:460-467`、`:556-561`
- `handlers/plan-mode.ts:214-222`
- `handlers/bash.ts`（564 行）与 24 个 `bash-readonly-policy-*.ts`

**core/src/subagent**
- `runner.ts:811-816`、`:1142-1169`、`:1202-1239`、`:1176-1199`
- `tool-policy.ts:7-31`
- `profile.ts:23-37`、`:72-87`
- `completion-notification.ts:24-60`

**contracts**
- `events/session.events.ts:84-184`
- `interfaces/permission.port.ts:35-39`、`:45-52`、`:54-97`
- `interfaces/session-store.port.ts:808-827`、`:1099-1112`
- `tools/plan-mode.ts:13`、`:55-67`
- `tools/target.ts:9-16`、`:46-63`、`:119-135`、`:137-194`、`:196-245`、`:289-333`

**adapters**
- `model/failure-classifier.ts:81-90`、`:183-245`、`:386-437`
- `model/runner-retry.ts:15`、`:38-56`、`:81-108`
- `model/failure-inspection.ts:286-300`
- `model/retry-policy.ts:13-30`
- `model/retry-budget.ts:4-18`、`:32-57`
- `context/git-snapshot.ts:28-32`、`:165-168`
- `storage/session-store/sqlite-session-store.ts`（928 行）

**bootstrap**
- `zcode-protocol-v4/command-inbox.ts:110-218`、`:288-310`、`:312-448`
- `zcode-protocol-v4/cold-session-resume.ts:49-137`
- `zcode-protocol-v4/queue-auto-drain.ts:7-19`
- `zcode-protocol-v4/v4-gateway.ts:1088`、`:2604`

**docs（design v2）**
- `loop/tool-calling-loop.md:3`、`:7`、`:72`、`:365-367`
- `loop/turn-steering.md:92-96`、`:151-161`、`:163-172`
- `turn-cancellation.md:13-24`、`:27-46`、`:66-72`、`:78-86`、`:90-97`
- `compact/README.md:29-39`、`:94-111`、`:127-139`
- `model/stream-error-retry.md:53-60`、`:83-105`
- `Z/AGENTS.md:31`、`:62`、`:64-66`、`:68-69`

**gaea 落点（本次核实过的现状）**
- `internal/gaea/agent/execute_one.go:21`、`compact.go:106/136/155/249/287`、`batch_executor.go`、`stop_gate.go`、`output_continue.go`、`interrupt_tool.go`、`interrupted_recovery.go`、`plan_mode.go`、`request_permission.go`、`subagent_store.go`、`fold_ladder.go`、`tool_coherence.go`
- `internal/gaea/permission/permission.go:114`（Decide）、`:171`（subjectKeys）、`:199`（matchGlob）、`:228`（Approver）、`:257`（Gate.Check）、`:304`（rememberRule）
- `internal/gaea/context/metrics.go`、`contextview/estimate.go`、`internal/gaea/event/event.go`、`internal/gaea/tasks/tasks.go`、`internal/taskinbox/taskinbox.go`、`internal/gaea/spill/spill.go`、`internal/gaea/tool/tool.go`、`internal/gaea/db`

---

### 一句话总结

ZCode 的运行时内核真正值得抄的不是「loop 写法」，而是四件**事实与边界的纪律**：

1. **状态有唯一转移表**（M1）——办公长任务的「卡在哪」必须可回答；
2. **权限是有序链 + 可复用规则数据**（M14/M16）——办公的动作粒度远比「读/写文件」细；
3. **压缩必须可续接**（M23）——办公交付物是跨压缩连续写完的同一份文档；
4. **取消与恢复都必须落成事件事实**（M3/M9/M13）——办公用户会中途打断、会关应用、会第二天接着问。

而这四条的**载体**（终端/shell、git、Electron 多宿主、远程 workspace、workflow 脚本引擎）恰恰是办公模块最该丢掉的重量。
