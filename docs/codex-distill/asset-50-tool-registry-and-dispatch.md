# asset-50 · 工具注册与派发（B5 · 施工级）

> 证据：`core/src/tools/registry.rs`、`core/src/tools/router.rs`、`core/src/tools/spec_plan.rs`、
> `core/src/tools/parallel.rs`、`core/src/tools/orchestrator.rs`、`core/src/tools/sandboxing.rs`、
> `core/src/tools/approvals.rs`、`core/src/tools/context.rs`、`core/src/tools/mod.rs`、
> `core/src/tools/executed_tool_calls.rs`、`core/src/tools/executed_tool_calls/{request_metadata,mcp_attribution}.rs`、
> `core/src/tools/tool_namespaces_info.rs`、`core/src/tools/multi_agent_tool.rs`、
> `core/src/tools/handlers/mcp.rs`、`core/src/tools/handlers/unified_exec/exec_command.rs`、
> `core/src/tools/handlers/apply_patch.rs`、`core/src/tools/handlers/multi_agents_common.rs`、
> `core/src/tools/runtimes/mod.rs`、`core/src/unified_exec/process_manager.rs`、
> `core/src/session/{mod.rs,turn.rs,step_context.rs,step_settings.rs}`、`core/src/stream_events_utils.rs`、
> `tools/src/{lib.rs,tool_executor.rs,tool_spec.rs}`、`protocol/src/models/executed_tool_calls.rs`；
> gaea 侧：`internal/gaea/agent/{execute_one.go,tool_dispatch.go,batch_executor.go,discover_tools.go,agent.go,agent_stream.go,request_permission.go}`、
> `internal/gaea/tool/tool.go`、`docs/gaea-tool-registry-map-2026-11.md`。
> 仅出机制与施工卡，不含代码改动。

---

## 1. 一句话结论

codex 的工具链路是**八段流水线**：静态注册（`ToolRegistry` 里 runtime + 暴露面）→ 每 step 定稿（`spec_plan` 产出一份不可变 `ToolRouter`：策略过滤后的 runtime 表 + 模型可见 `ToolSpec` 数组）→ 披露（`build_prompt` 只把 `model_visible_specs` 交给模型）→ 解析（`build_tool_call` 把 ResponseItem 变成 `ToolCall`）→ 派发（每个调用单独 spawn，读写闸门控并发）→ 钩子（PreToolUse 可改写/阻断）→ 执行（handler 内部再走 `ToolOrchestrator`：**审批 → 沙箱首试 → 失败重试再审批**）→ 落账（`ExecutedToolCall` 归因 + PostToolUse + 生命周期终态）。

**审批不在注册表里，也不在派发前置闸里**——它插在 handler 内部、真正执行前的第一行（`orchestrator.rs:140`），这是与 gaea 最大的口径差。

---

## 2. 机制地图

| 关注点 | 文件 | 关键行 |
|---|---|---|
| runtime 契约（名称/spec/暴露面/并发/句柄） | `tools/src/tool_executor.rs` | 106-139 |
| 暴露面六态与位标志 | `tools/src/tool_executor.rs` | 14-29、49-99 |
| 请求侧 spec 联合体（function/namespace/tool_search/web_search/custom） | `tools/src/tool_spec.rs` | 20-56 |
| core 侧 runtime 附加契约（hooks/telemetry/延迟就绪/第三方标记） | `core/src/tools/registry.rs` | 63-194 |
| 注册：内建 / 外部 / 前置 / 策略过滤 / 重名策略 | `core/src/tools/registry.rs` | 479-578 |
| 注册：保留名 `exec_command`/`shell_command` | `core/src/tools/registry.rs` | 548-556 |
| 派发入口（计数、未知工具、payload 校验、环境解析、hooks、执行、落账） | `core/src/tools/registry.rs` | 691-984 |
| 计划定稿（暴露面策略、tool_search、code mode、模型可见 spec） | `core/src/tools/spec_plan.rs` | 369-554、628-660 |
| 注册来源（shell / MCP resource / 工具 / 协作） | `core/src/tools/spec_plan.rs` | 1185-1205 |
| 模型可见性判定 | `core/src/tools/spec_plan.rs` | 867-879 |
| …（tool_search / code mode）| `core/src/tools/spec_plan.rs` | 392-461、1017-1183、1621-1638 |
| 每 step 的不可变计划载体 | `core/src/tools/router.rs` | 85-92 |
| 响应项 → 工具调用解析 | `core/src/tools/router.rs` | 257-309 |
| 并发闸门、取消、计时 | `core/src/tools/parallel.rs` | 126-330 |
| 审批 / 沙箱 / 重试三段 | `core/src/tools/orchestrator.rs` | 125-626 |
| 审批需求三态 + 默认策略推导 | `core/src/tools/sandboxing.rs` | 152-232 |
| 审批动作与上下文 | `core/src/tools/approvals.rs` | 52-147、408-424 |
| 调用归因记录 | `core/src/tools/executed_tool_calls.rs` | 345-367、720-739 |
| 归因结构（name/arguments/来源/结果元数据） | `protocol/src/models/executed_tool_calls.rs` | 406-428 |
| 归因随请求下发 | `core/src/tools/executed_tool_calls/request_metadata.rs` | 12-61 |
| 每请求工具面清单 | `core/src/tools/tool_namespaces_info.rs` | 18-110 |
| 模型请求组装（tools 字段来源） | `core/src/session/turn.rs` | 1688-1717 |
| 在飞调用顺序收集 | `core/src/session/turn.rs` | 2601-2629、2775、3013 |
| step 快照与工具计划绑定 | `core/src/session/{mod.rs,step_context.rs,step_settings.rs}` | mod.rs:3997-4011、4147-4185；step_context.rs:23-48；step_settings.rs:36-53 |

---

## 3. 机制（按调用顺序）

### 3.1 段一：注册 —— 谁把自己的 runtime 塞进注册表

`ToolRegistry` 是 `IndexMap<ToolName, RegisteredTool>` + 一份函数名前缀策略 + 一个"首个冲突"槽位（`registry.rs:406-411`）。`RegisteredTool` 三件事：`runtime`、`exposure`（本 step 生效的暴露面）、`model_spec`（本 step 的描述覆盖，永不回写共享 runtime）（`registry.rs:381-386`）。

四条注册通道，语义各不相同：

1. `add` / `add_with_exposure` / `register_trusted[_with_exposure]`：内建工具。进去第一件事是过 `tool_policy.allows(&tool_name)`，不允许就**静默丢弃**（`registry.rs:498-507`）；名字重复走 `error_or_panic`（测试里 panic，生产里打日志）（`registry.rs:508-516`）。
2. `prepend_trusted`：插到表头，用于必须在工具列表最前的内建（`registry.rs:519-532`）。
3. `register_external[_with_exposure]`：MCP / 客户端动态工具。除了策略过滤，还硬拒两个保留名 `exec_command`、`shell_command`（`registry.rs:544-556`）；重复则 warn + 记 `first_collision`，返回 `false`（`registry.rs:559-573`）。
4. 计划期的 `apply_functions_namespace_function_prefixes`：把本 step 的函数名前缀策略刷进每个 `model_spec`（`registry.rs:430-443`）——注释明确"只影响下发，不改执行与暴露面"（`registry.rs:428-429`）。

注册来源本身在 `spec_plan::add_core_tool_sources`：shell 族、MCP 资源族、核心工具族、协作族（`spec_plan.rs:1185-1205`）。shell 族还有一层**互斥选择**：`Feature::UnifiedExec` 开着就注册 `ExecCommandHandler` + `WriteStdinHandler`（可续写进程）；关着就只注册 one-shot 版 `ExecCommandHandler`（`spec_plan.rs:1264-1272`）。协作族按 `MultiAgentVersion` 分 V1/V2 两套注册和两套暴露面（V2 的 `exposure` 还受 `non_code_mode_only` 影响）（`spec_plan.rs:1455-1585`）。动态工具走 `register_external`（`spec_plan.rs:1588-1617`），扩展工具走 `ExtensionToolAdapter` 包装（`spec_plan.rs:1640-1668`）。

**职责边界**：注册只管"这个名字背后有没有一个能跑的 runtime"，不管它是否对模型可见。

### 3.2 段二：计划定稿 —— spec_plan 什么时机生成

生成时机是**每个采样请求（step）前**，不是每次启动、也不是每个回合：

- `session/mod.rs:4147-4157`：`prepare_tools` 闭包里调用 `turn::built_tools(...)`，产物 `tool_router` 直接进 `StepContext`（`mod.rs:4167-4185`）。
- `StepContext` 的注释就是契约："本采样请求对外披露并执行的那份定稿工具计划"（`step_context.rs:44-45`），`settings` 是"请求前捕获的不可变快照"（`step_context.rs:29-30`）。

**和 step 快照（`next_step_settings`）的关系**：`TurnContext.next_step_settings` 是 `ArcSwap<ResolvedStepSettings>`（`turn_context.rs:344`）。建 step 时先 `load_full()` 拿到一份 `Arc` 快照（`mod.rs:4005`），再把 `settings.model_info` 传给 `built_tools`（`mod.rs:4150-4151`）。于是：

- 快照是**输入**，`ToolRouter` 是**输出**；同一个快照可以产出多个 `StepContext`（`step_settings.rs:37-39` 注释：替换快照不影响已经捕获它的 step）。
- 之后 `ArcSwap` 再被换掉（模型/审批策略/reasoning 变更），**已建好的 step 的工具面不跟着变**——它的 `tool_router` 与 `settings` 是两个独立字段，成对冻结。

`finalize_tool_router` 做的事按顺序（`spec_plan.rs:369-554`）：清掉不合法 hosted spec（377）→ 命名空间直连覆盖（378）→ 第三方严格延迟化（379、603-625）→ 算 `tool_mode`（380-391）→ 决定 `tool_search` 是否出现，冲突则移除并记 collision（392-445）→ 函数名前缀预算（447-452）→ 挂 tool_search 执行器（454-461）→ 生成间接命名空间前缀，失败即 `InvalidRequest`（463-467）→ 注册 code mode 执行器（468-469）→ 可选冲突硬失败（476-530，`error_on_tool_collisions` 打开时，命名空间内工具不同属主也算冲突）→ 建模型可见 spec（532-538）→ 收集工具面清单（539-543）→ 产出 `ToolRouter`（546-553）。

模型可见 spec 的判定只有一行：`exposure.is_direct() && !is_hidden_by_code_mode_only(...)`（`spec_plan.rs:867-879`）。也就是说 **"注册了"和"模型看得见"之间只隔着 `exposure` 一个字段**。

### 3.3 段三：披露 —— 模型看见什么

`build_prompt` 把 `step_context.tool_router.model_visible_specs()` 塞进 `Prompt.tools`（`turn.rs:1697-1701`），并硬写 `parallel_tool_calls: true`（`turn.rs:1702`）——**模型侧永远被告知"可以并行发多个工具调用"**，真正的并发限制在宿主侧（§3.5）。增量工具模式下 `tools` 反而是空的（`turn.rs:1697-1699`），改由历史携带声明。

序列化面在 `tools/src`：`ToolSpec` 五种形态（function/namespace/tool_search/web_search/custom）（`tool_spec.rs:20-56`），`create_tools_json_for_responses_api` 直出，`create_tools_json_for_responses_lite` 把 function/freeform 收进一个 `functions` 命名空间（`tool_spec.rs:82-142`）。

### 3.4 段四：解析 —— 响应项变工具调用

`ToolRouter::build_tool_call` 只认三种来源（`router.rs:257-309`）：

| 响应项 | 产出 | 备注 |
|---|---|---|
| `FunctionCall{name, namespace, arguments, call_id}` | `ToolPayload::Function{arguments}` | 名字统一 `with_default_namespace` |
| `ToolSearchCall{execution:"client"}` | `ToolPayload::ToolSearch{arguments}`，工具名硬编码 `tool_search` | 参数解析失败 → `RespondToModel` |
| `ToolSearchCall{execution!="client"}` | `Ok(None)` | 服务端执行，宿主不派发 |
| `CustomToolCall` | `ToolPayload::Custom{input}` | freeform 工具（如 code mode 入口） |

`dispatch` 时还会再校验一次 payload 与 runtime 是否同类（`registry.rs:759-775`，默认只接受 `Function|ToolSearch`，见 `registry.rs:119-124`），不匹配直接 `Fatal`。

### 3.5 段五：派发与并发 —— 谁先谁后，失败怎么隔离

派发点只有一个：`stream_events_utils::handle_output_item_done` 里发现是工具调用就 `call_trace::received` 记账、立刻把响应项落历史、然后 `runtime.handle_tool_call(call, token)` 得到一个 future（`stream_events_utils.rs:325-371`）。注释写明"派发在这个 future 被 poll 之前就发生"（`stream_events_utils.rs:353`）。

这些 future 进 `FuturesOrdered<InFlightFuture>`（`turn.rs:2775`、`3013`），**push 顺序 = 流里出现的顺序，收集顺序也按这个顺序**（`turn.rs:2601-2629`）。所以：

- **派发并发**：多个调用同时跑（各自 `tokio::spawn`，`parallel.rs:202-256`）。
- **结果保序**：宿主按调用顺序把结果写回历史，不是按完成顺序。
- **并发粒度由工具自己声明**：`supports_parallel = router.tool_supports_parallel(&call)`（`parallel.rs:141`），真并行取 `lock.read()`，否则取 `lock.write()`（独占）（`parallel.rs:213-217`）。声明来自 `ToolExecutor::supports_parallel_tool_calls`（默认 false，`tool_executor.rs:122-124`），并且**`Hidden` 暴露面一律视为不可并行**（`registry.rs:682-685`）。
- 实测样例：`ExecCommandHandler` 声明 true（`exec_command.rs:138-140`）；MCP 工具在"服务端自声明"或"annotations.read_only_hint==true"时声明 true（`mcp.rs:173-185`）。
- **失败隔离**：单个调用的 `FunctionCallError` 非 `Fatal` 会被就地包成失败响应（`parallel.rs:100-113`、`338-363`），其它调用不受影响；`Fatal` 才升级成 `CodexErr::Fatal` 终止（`parallel.rs:109`）。任务本身 panic/join 失败 → `Fatal`（`parallel.rs:334-336`）。

### 3.6 段六：钩子（PreToolUse / PostToolUse）与执行

`dispatch_any_with_state` 的顺序（`registry.rs:691-984`）：

1. 活动回合 `tool_calls` 计数 +1（708-723）；
2. 起 `ToolDispatchTrace`，查 runtime；查不到就 `RespondToModel("unsupported call: ...")` 并记 `success=false`、`record_failed`（725-747，文案生成见 1049-1054）；
3. 收 telemetry 标签，`mcp_server*` 单独进 trace 字段（748-758）；
4. payload 类型校验（759-775）；
5. 解析执行环境：优先 `live_environment_resolver`，否则 `environments_for_input`，失败走 `fail_before_execution`（793-800）；
6. **PreToolUse**：`Blocked` → 控制类工具记 `Rejected` 分析事件，返回 `RespondToModel`，生命周期终态 `Blocked`（801-827）；`Continue{updated_input}` → `with_updated_hook_input`，若改写动了环境选择器字段则直接拒绝（828-839，实现见 196-268）；
7. 非 MCP 工具发 `notify_tool_start`（842-846）；
8. shell 类调用顺手按 parse 结果打 `command_category` 标签（851-865）；
9. 执行：`otel.log_tool_result_with_tags(...)` 包住 `handle_any_tool`（869-884）；
10. `handle_any_tool` 里 `tool.handle(invocation)`，成功后再算 `post_tool_use_payload`，并把"外部上下文污染记忆"落库（999-1028）；
11. 成功才跑 PostToolUse；`should_block` → 结果是 `RespondToModel`（注意：**执行已经发生**，`registry.rs:933-934` 注释明说这是"拒绝结果而不是拒绝执行"）；有 feedback 则用 `PostToolUseFeedbackOutput` 把模型可见面换成反馈文本（897-968）；
12. 生命周期终态：`finishes_on_cancellation && 已取消` → `Aborted`；`Ok` → `Completed{success}`；`Err` → `Failed{handler_executed:true}`（933-946），并通过 `terminal_outcome_reached` 做一次性闩锁（986-997）。

### 3.7 段七：执行内部 —— 审批到底插在哪

handler 里做真正危险动作前，构 `ToolOrchestrator` 并 `run`：

- shell：`process_manager.rs:1511-1581`（`ToolOrchestrator::new()` → `orchestrator.run(&mut runtime, &req, &tool_ctx)`）；
- 改文件：`apply_patch.rs:582-587`。

`ToolOrchestrator::run` 三段（`orchestrator.rs:125-626`）：

- **第 1 段 审批**（`orchestrator.rs:140-226`）：先 `tool.exec_approval_requirement(req)`，缺省回落到 `default_exec_approval_requirement(approval_policy, fs_policy)`（164-166）。三态处理：`Skip` → 若 `strict_auto_review` 仍要送审（168-190）；`Forbidden{reason}` → 直接 `ToolError::Rejected`（200-202）；`NeedsApproval{reason}` → 组 `ApprovalContext` 送 `session.request_approval`（203-225）。
- **第 2 段 沙箱首试**（`orchestrator.rs:228-392`）：从 `sandbox_override_for_first_attempt` 决定是否"带限制升级"，跨远程执行器还要探能力位，否则 `Rejected`（236-282、264-282）；`ensure_native_sandbox` 在要求限制却没有本地沙箱时拒绝（`sandboxing.rs:234-247`）。
- **第 3 段 沙箱拒绝后重试**（`orchestrator.rs:394-611`）：只有 `SandboxErr::Denied` 才进；`!tool.escalate_on_failure()` 直接返回（429-438）；策略为 `Never`/`OnRequest` 且不需要再问时返回带原始输出的拒绝（439-463）；否则**重新走一次审批**（`retry_reason` 明确写给用户看的原因，486-534），再以放宽的沙箱二次执行（558-611）。

**与 asset-13 七态的对齐点**：七态是"审批结果"层（`ReviewDecision`），本链路的插入点是这套决策的**唯一生产处**。`ApprovalContext` 携带 `call_id`/`tool_name`/`approval_reason`/`retry_reason`/`network_approval_context`/`strict_auto_review`（`approvals.rs:52-62`），`ApprovalAction` 区分 `ExecCommand`/`WriteStdin`/`ApplyPatch`/`McpToolCall`/`NetworkAccess`/`RequestPermissions` 六种实体（`approvals.rs:64-147`），`ApprovalResolutionSource`（User/Hook/…）决定"这次批准能不能覆盖重试"（`orchestrator.rs:486-492`，定义在 `approvals.rs:408-424`）。所以：**七态里"记住这条"与"这次放行"的区别，正是靠 `initial_approval_source` 传进第 3 段实现的**——`User|Hook` 才算可跨重试的预授权。

### 3.8 段八：落账 —— request_metadata 记了什么

两个不同层次，别混：

**(a) 单调用归因（随请求下发给模型侧的历史）**：`ExecutedToolCall{name, arguments, tool_result_sources, tool_result_metadata}`（`protocol/src/models/executed_tool_calls.rs:406-428`）。其中 `tool_result_sources` 与 `tool_result_metadata` 都标了 `skip_deserializing`，注释写明"宿主生成的分析数据，不信任输入 JSON"（412-427）——即**这两个字段不能由模型/外部注入**。

记录时机：`record_tool_call(call, source, step_context)`（`executed_tool_calls.rs:345-352`，在 `parallel.rs:136-139` 被调用）；Direct 调用还会额外 `prepare_direct_call` 预留槽位（`executed_tool_calls.rs:355-380`），执行完后 `attach_direct_call_to_output` 挂到该次输出项上并分配历史 ID（384-445）。名字统一用 code mode 命名（`recorded_call`，720-739）。

挂载时机：`attach_to_prompt` / `attach_to_compaction_prompt`（`request_metadata.rs:12-22`），核心是"把 Direct 记录挂回它的输出项、把 Code Mode cell 里待定的嵌套调用补挂到 cell 输出项、必要时清掉完整性位"（24-330）。完整性用 `mark_tool_calls_complete` / `clear_tool_calls_complete` 表达（313-317），冲突或重复输出会把整个 cell 标记为不可信（105-180、245-249）。

**归因维度**（按可观测性列清楚）：

| 维度 | 载体 | 位置 |
|---|---|---|
| 谁发起：直连 / 直连明文消息 / code mode 嵌套 | `ToolCallSource`（含 `cell_id`、`runtime_tool_call_id`） | `context.rs:56-68`；`router.rs:56-71` |
| 哪个工具、哪个调用 | `ToolName` + `call_id`（`ToolInvocation` 字段） | `context.rs:111-125` |
| 原始模型调用项 / 窗口 | `ToolCallOrigin{item_id, window_id}` | `context.rs:127-166` |
| 属于哪个 code mode cell | `set_tool_call_cell_id` / cell 绑定 | `request_metadata.rs:310-317` |
| 本次请求的工具面清单（直连/延迟/code mode 名/来源是 MCP 还是 harness） | `TurnToolNamespacesInfo` | `tool_namespaces_info.rs:18-110` |
| 工具模式（direct / code_mode / code_mode_only） | `record_registration_metrics` 的标签 | `router.rs:399-446` |
| 线程/回合/模型/sandbox | telemetry 标签 + `codex.tool_call` 日志字段 | `parallel.rs:451-467`；`registry.rs:699-706` |
| 审批上下文 | `ApprovalContext` | `approvals.rs:52-62` |

**注意**：`call_id` 是模型给的 id，code mode 的 `runtime_tool_call_id` 只保证 cell 内唯一，**不是** Codex 工具调用 id（`context.rs:61-67` 注释明说）。做归因必须用前者 + cell_id + 历史项 id 三者组合。

---

## 4. 常量与上限

| 名称 | 值 | 位置 | 生产/测试 | 语义 |
|---|---|---|---|---|
| `MAX_PENDING_EXECUTED_TOOL_CALLS` | `256` | `core/src/tools/executed_tool_calls.rs:49` | 生产 | 在飞归因记录/cell/嵌套调用三处共用的并发上限，超限即拒绝新记录（同文件 199-235、493、536-545） |
| `MAX_RETAINED_DIRECT_METADATA_BYTES` | `1024*1024`（1 MiB） | `core/src/tools/executed_tool_calls.rs:53` | 生产 | Direct 归因元数据驻留总字节；超了按"保留资源访问 → 省略结果元数据 → 清结果元数据 → 清全部调用记录"四级降级（384-445） |
| `MAX_ID_BYTES` | `1024` | `core/src/tools/executed_tool_calls/seen_ids.rs:17` | 生产 | 单个 call id 接受的最长字节 |
| `ID_FILTER_WORDS` | `8*1024` | 同文件 `:13` | 生产 | 历史 id 过滤位图字数 |
| `MAX_HISTORY_ENTRIES` | `100_000` | 同文件 `:14` | 生产 | 历史 id 索引条目上限 |
| `MAX_HISTORY_ID_BYTES` | `4*1024*1024` | 同文件 `:15` | 生产 | 历史 id 索引总字节 |
| `REJECTION_MESSAGE_MAX_TOKENS` | `900` | `core/src/tools/events.rs:37` | 生产 | 拒绝消息回喂模型的 token 帽 |
| `EXEC_COMMAND_REJECTION_MAX_BYTES` | `900` | `core/src/tools/handlers/unified_exec/exec_command.rs:59` | 生产 | shell 拒绝消息字节帽 |
| `MAX_MCP_NAMESPACE_DESCRIPTION_BYTES` | `512*1024` | `core/src/tools/handlers/mcp.rs:50` | 生产 | MCP 命名空间描述字节帽 |
| `MAX_AGENT_PLUGIN_MCP_NAMESPACE_DESCRIPTION_BYTES` | `1_000` | 同文件 `:49` | 生产 | 插件类 MCP 命名空间描述帽 |
| `MAX_SLEEP_DURATION_MS` | `12*60*60*1000` | `core/src/tools/handlers/sleep.rs:28` | 生产 | `sleep` 工具上限（12 小时） |
| `DEFAULT_WAIT_TIMEOUT_MS` | `30_000` | `core/src/tools/handlers/multi_agents_common.rs:23` | 生产 | 等子代理默认超时 |
| `MIN_WAIT_TIMEOUT_MS` / `MAX_WAIT_TIMEOUT_MS` | **未读**（仅见别名指向 `codex_protocol` 常量） | 同文件 `:22`、`:24` | 生产 | 等子代理超时上下界；具体数值在 `codex_protocol`，本轮未打开该文件 |
| 保留工具名 | `"exec_command"` / `"shell_command"` | `core/src/tools/registry.rs:548-556` | 生产 | 外部/MCP 工具不得占用，占用即跳过并记 warning |
| `SNAPSHOT_ORIGINAL_*_ENV_KEY` 等 | 3 个字符串常量 | `core/src/tools/runtimes/mod.rs:42-44` | 生产 | shell 快照回放用的环境键名（与派发无关，仅登记以免误判为工具常量） |
| `TEST_TRUNCATION_POLICY` | `TruncationPolicy::Tokens(10_000)` | `core/src/tools/handlers/unified_exec_tests.rs:26` | **测试** | 仅测试里用的截断策略，不得当生产默认值引用 |
| `maxParallel`（gaea） | `8` | `internal/gaea/agent/batch_executor.go:367` | 生产 | gaea 单批并行 goroutine 上限 |
| `stormBreakThreshold`（gaea） | `3` | `internal/gaea/agent/batch_executor.go:394` | 生产 | 连续同因失败几次后改写为改法指令 |
| `discoverMaxActivate` / `discoverCatalogCap`（gaea） | `5` / `20` | `internal/gaea/agent/discover_tools.go:37-39` | 生产 | 动态目录单次激活上限/清单条数上限 |

---

## 5. 状态机 / 时序

### 5.1 暴露面六态（本 step 内有效）

来源：`tool_executor.rs:49-99`（定义）、`spec_plan.rs:276-288`（由位标志映射而来）、`router.rs:185-216`（判定"模型能否直调/能否搜到/能否 code mode 调"）。

| 态 | 在直连工具列表 | 可被 tool_search 搜到 | 可被 code mode 嵌套调用 | 可派发执行 |
|---|---|---|---|---|
| `Direct` | 是 | —（已直连） | 是 | 是 |
| `DirectModelOnly` | 是 | 否 | 否 | 是 |
| `Deferred` | 否 | 是 | 是 | 是 |
| `DeferredModelOnly` | 否 | 是 | 否 | 是 |
| `CodeModeOnly` | 否 | 否 | 是 | 是 |
| `Hidden` | 否 | 否 | 否 | 是（仍可派发，但不可并行） |

不可达态：`Direct + Deferred` 同时成立 —— `spec_plan.rs:287` 是 `unreachable!`，因为 `finalize` 阶段用 `difference` 强制互斥（`spec_plan.rs:266-274`）。

### 5.2 单次调用的生命周期终态

| 终态 | 触发 | 证据 | 模型看到的 |
|---|---|---|---|
| `Completed{success:true}` | handler 成功且 PostToolUse 未 block | `registry.rs:940`、948-977 | 工具输出（或 PostToolUse feedback 替换体） |
| `Completed{success:false}` | handler 返回 `Ok` 但自报失败 | `registry.rs:940` | 同上，`success_for_logging` 为假 |
| `Failed{handler_executed:true}` | handler 返回 `Err` | `registry.rs:941-943` | 错误文本（`RespondToModel`）或整轮终止（`Fatal`） |
| `Failed{handler_executed:false}` | 执行前失败（环境解析、hook 改写失败等） | `registry.rs:777-792` | 同上 |
| `Blocked` | PreToolUse 阻断 | `registry.rs:813-827` | 阻断理由 |
| `Aborted` | 取消且 handler 声明 `finishes_on_cancellation` | `registry.rs:935-939`；`parallel.rs:260-299` | `aborted by user after N s` 或 exec_command 专属格式（`parallel.rs:376-385`） |

一次性闩锁：`ToolCallState.terminal_outcome_reached` 用 `AtomicBool::swap`，保证"取消路径"和"注册表路径"不会重复发终态（`registry.rs:986-997`、`parallel.rs:300-318`）。

### 5.3 时序（一次多调用回合）

```
step 建好（settings 快照 + ToolRouter 冻结）
  └─ build_prompt → Prompt.tools = model_visible_specs
       └─ 模型流式返回 N 个工具调用
            ├─ 每个 item：call_trace::received → 落历史 → handle_tool_call()
            │     ├─ 记账 record_tool_call / prepare_direct_call
            │     ├─ spawn：wait_until_ready → 读/写闸门 → registry.dispatch_any_with_state
            │     │     ├─ 查表 / payload 校验 / 环境解析
            │     │     ├─ PreToolUse（可 block / 可改写）
            │     │     ├─ handler.handle → ToolOrchestrator（审批 → 沙箱首试 → 可选重试再审批）
            │     │     └─ PostToolUse → 终态 → AnyToolResult
            │     └─ future 入 FuturesOrdered
            └─ 流结束：drain_in_flight 按 push 顺序把结果写回历史 → 下一 step 决定是否 follow-up
```

---

## 6. 失败路径

**F1 · 未知工具（可自纠）**：`registry.rs:726-747`。查不到 runtime → `unsupported_tool_call_message`（`Custom` 与其它两套文案，`registry.rs:1049-1054`）→ otel 记 `success=false`、时长 0 → `dispatch_trace.record_failed` → 返回 `FunctionCallError::RespondToModel`。收敛方式：**变成一条工具结果回到模型**，回合继续。活动回合的 `tool_calls` 计数**已经加过 1**（`registry.rs:708-723` 早于查表），所以"尝试计数"包含无效调用。

**F2 · payload 与 runtime 不匹配（不可自纠）**：`registry.rs:759-775`。`tool.matches_kind(payload)` 为假 → `FunctionCallError::Fatal` → `parallel.rs:109` 升成 `CodexErr::Fatal` → 整轮终止，**没有工具结果项**。为什么必须 Fatal：模型改参数也修不好类型错配，重试只会烧预算。

**F3 · PreToolUse 阻断（可自纠但不算失败）**：`registry.rs:813-827`。`Blocked` → 控制类工具记 `Rejected` → `RespondToModel` → 终态 `Blocked`。与 F1 的差别在语义：F1 是"没这个工具"，F3 是"有但被策略拦"。

**F4 · 审批拒绝 / 策略禁止（收敛在 handler 内）**：`orchestrator.rs:200-202`（`Forbidden` 直接 `ToolError::Rejected`）、`orchestrator.rs:493-534`（重试前二次审批被拒）。`ToolError::Rejected` 不在 `sandbox_outcome_from_tool_error` 的映射表里（`orchestrator.rs:629-639`），因此不会污染 sandbox 指标，只是把拒绝理由回给模型。

**F5 · 沙箱拒绝且不允许升级（带回原始输出）**：`orchestrator.rs:429-473`。三条短路：`!escalate_on_failure`、策略 `Never/OnRequest` 且不满足豁免条件、`!unsandboxed_allowed` 且无网络审批上下文。收敛方式：**保留首试的原始 `ExecToolCallOutput` 一起返回**（439-442 注释），让模型看到"是被沙箱拦的"而不是"命令本身错"。

**F6 · 取消**：`parallel.rs:260-299`。`finishes_on_cancellation` 为真或终态已达成 → 等它自然结束；否则 abort 任务并合成 `aborted_response`，同时补发 `result_ready` 与 `notify_tool_aborted`。注意 `abort_message` 对 `exec_command` 有专属格式（`parallel.rs:376-385`）——下游解析别只认一种。

**F7 · round 内 join 失败**：`parallel.rs:334-336`、`turn.rs:2623-2625`。任务 join 错误 → `Fatal`；drain 阶段遇到错误走 `error_or_panic`（测试 panic / 生产日志），**不阻塞后续结果收集**。

**F8 · 归因记录自身失败（静默降级）**：`executed_tool_calls.rs:384-445`。预算不够时逐级削减归因内容直到清空，并打 `record_shedding` 指标；`reserve_direct_call` 拿不到槽位就返回 `None`（369-380）——**归因是 best-effort，绝不影响工具执行结果**。

---

## 7. gaea 对照

### 7.1 逐项对照

| 关注点 | codex | gaea | 口径差 |
|---|---|---|---|
| 注册表结构 | `IndexMap<ToolName, RegisteredTool>` + 策略 + 冲突槽（`registry.rs:406-411`）；内建重名 `error_or_panic`（512-516），外部重名 warn+记 collision（559-573） | `map[string]Tool` + `order []string` + `hidden map[string]bool` + `canon`（`internal/gaea/tool/tool.go:192-203`）；`Add` 静默覆盖并保留首次顺序（`tool.go:220-231`） | gaea 无重名可见性，也没有"注册阶段就按策略过滤"这一层（codex 在 `registry.rs:504-506` 直接 return） |
| 暴露面 | 六态位标志（`tool_executor.rs:14-29`、`49-99`），计划期由 MCP `omit_tools_from` 等策略算出（`spec_plan.rs:198-290`） | 单布尔 `hidden`（`tool.go:235-239`）+ `HideUnlessOnly` 兜底（244-260）+ 会话级 `resident/activated`（`discover_tools.go:44-74`） | gaea 能表达"藏起来但可调"（≈`Hidden`）和"动态目录激活"（≈`Deferred`），**不能**表达 `DirectModelOnly` / `CodeModeOnly` |
| 模型可见面生成 | 每 step 重建 `model_visible_specs: Arc<[ToolSpec]>`（`router.rs:85-92`、`spec_plan.rs:628-660`） | 每轮 `currentSchemas()`：有 resident 就 `FilteredSchemas(resident ∪ activated)`，否则 `Schemas()`（`agent_stream.go:22-42`、`tool.go:367-412`） | codex 的计划是**不可变快照**并与 step 同生共死；gaea 是每轮现算，且激活集"只增不减"（`discover_tools.go:57-74`） |
| 并发判据 | 工具自声明 `supports_parallel_tool_calls`（`tool_executor.rs:122-124`）+ `Hidden` 强制 false（`registry.rs:682-685`）+ 读/写闸门（`parallel.rs:213-217`） | 名字→冲突键的中心 switch（`batch_executor.go:292-362`）+ 全局键前缀 `!` 强制串行 + `maxParallel=8`（366-368） | gaea 新增工具必须改中心 switch；codex 由工具自己声明，注册表只做仲裁 |
| 批内顺序 | `FuturesOrdered`：派发按流序、结果按流序回收（`turn.rs:2601-2629`、`2775`、`3013`） | `partitionToolCalls` 连续扫描分批、**从不出队重排**（`batch_executor.go:259-282`），批内 `runParallel` 完成后按索引写 `results[i]`（100-142） | 两者都保序；差别是 codex 的"批"不是显式结构，并发度由每个调用自己的锁决定 |
| 审批插入点 | handler 内、执行前，由 harness 强制（`orchestrator.rs:140-226`）；沙箱失败后**二次审批**（493-534） | `ToolDispatcher.Check`：PermissionRequest hook → gate → PreToolUse（`tool_dispatch.go:53-104`），gate 实现是 `Gate interface`（`agent.go:208-210`）；危险操作可让模型主动调 `request_permission`（`request_permission.go:22-37`） | gaea 的审批是"静态规则闸 + 模型主动申请"；codex 是"执行前必经的三态决策 + 沙箱失败后追认" |
| 派发归因 | `ExecutedToolCall{name,arguments,result_sources,result_metadata}`（`protocol/src/models/executed_tool_calls.rs:406-428`）+ `ToolCallSource{cell_id,...}`（`context.rs:56-68`）+ 历史项 id/window（`context.rs:127-166`） | `auditFunc(tool, taskKind, readOnly, outcome, errMsg, outputLen, durationMs)`（`agent.go:522`），调用处第二参**恒为空串**（`execute_one.go:261`） | gaea 审计没有 call_id / 回合号 / 模型名；8 路并行同名调用时无法定位是哪一次失败 |
| 失败语义 | `RespondToModel`（可自纠）vs `Fatal`（终止整轮）两分类（`registry.rs:743-745`、`772-774`；`parallel.rs:109`） | `toolOutcome{blocked, recoverable, errMsg}`（`execute_one.go:285-318`），执行错误一律 `recoverable=true` | gaea 没有"不可自纠 → 终止整轮"这一档；同类效果靠风暴熔断间接实现（`batch_executor.go:394`、407-439） |
| 工具面清单的可审计性 | 运行期 `TurnToolNamespacesInfo`（`tool_namespaces_info.rs:18-110`），无静态文档 | `tool.RegistryMap` 单源生成 `docs/gaea-tool-registry-map-2026-11.md`（该文件 `:3`、表 `:7-123`），带防漂移守卫 | **gaea 强于 codex**：有编译期可校验的全量清单 |

### 7.2 结论：真正缺的是 2–3 件

1. **暴露面成文法（缺）**。gaea 只有 `hidden` 一个布尔（`tool.go:196`、`235-239`）。当我们要做"注册了但只给 code-mode 用""可被目录发现但不下发""只在直连面出现"这类策略时，只能临时加 `Hide` 调用点或改 `FilteredSchemas`，**没有中心字段可表达**。这不是重构洁癖：codex 的 `apply_mcp_tool_exposure_policy`（`spec_plan.rs:198-290`）之所以能一处收口 MCP 的 `omit_tools_from`，就是因为有那一个字段。
2. **派发归因三元组（缺）**。`call_id` / 回合号 / 模型名在 gaea 的审计里全没有（`execute_one.go:261` 第二参空串）。并行 8 路（`batch_executor.go:367`）时，日志只能告诉你"`bash` 失败了三次"，不能告诉你"哪一次、在哪一轮、哪个模型下"。
3. **并发能力声明面（缺一半）**。`getConflictKey` 是中心 switch（`batch_executor.go:292-362`），任何新工具默认落到 `default: return ""`——空键语义是"独立串行批"（278 行 `parallel: i-start > 1 && !hasGlobal` 的前提），所以**新工具默认拿不到并行**，且这个决定不在工具自己手里。

**不需要补的**：注册表全量清单（gaea 已有单源 + 防漂移守卫，比 codex 强）、批量保序（两者都做到了，`batch_executor.go:259-282` 与 `turn.rs:2601-2629` 等价）。

---

## 8. 施工卡

### 刀 1 · 暴露面字段化（`hidden` → `exposure`）

- **改动点**：`internal/gaea/tool/tool.go:192-203`（`Registry` 加 `exposure map[string]Exposure`）、`:235-239`（`Hide` 改为写 `ExposureHidden`，保留旧函数做别名）、`:367-412`（`FilteredSchemas` 的过滤条件从 `r.hidden[name]` 换成"是否含 direct 面"）。生成源同步：`internal/gaea/tool/registrymap.go`（清单单一来源）+ `docs/gaea-tool-registry-map-2026-11.md` 由脚本重生成，勿手改（该文件 `:3`）。
- **语义**：`Exposure` 取 `Direct | Deferred | Hidden` 三值（先做三值，code mode 面 gaea 没有对应物，不进第一版）。`Deferred` 的含义对齐 codex：**可被 `discover_tools` 发现、但不在常驻 schema 里**——正好与现有 `resident/activated` 机制（`discover_tools.go:44-74`）同构，实现上把"未 resident 即 deferred"显式化。
- **禁区**：不得改 `getConflictKey`（`batch_executor.go:292`）与 `Schemas()` 的返回排序（`tool.go:412` 按名排序，改了会动前缀缓存）；不得让 `Exposure` 影响 `Get()` 的可调用性（`tool.go:288-293`）——隐藏 ≠ 不可调，这是 codex 保住 `Hidden` 仍可派发的同款语义（`registry.rs:682-685`）。
- **默认关闭**：新增 `cfg.Tools.ExposureMode`（默认空=走旧 `hidden` 语义）；打开后 `Hide` 与 `Exposure` 双写一致才生效，避免装配面漂移。

### 刀 2 · 派发归因（补 `call_id` / 回合 / 模型）

- **改动点**：`internal/gaea/agent/agent.go:522`（`auditFunc` 签名**不动**，另加 `ToolCallMeta` 结构 + `opts.AuditFuncV2`）、`internal/gaea/agent/execute_one.go:254-262`（现有 `auditFunc` 调用点旁并行发 V2 事件；注意第二参 `taskKind` 现传 `""`）、`internal/gaea/agent/batch_executor.go:17`（`executeBatch` 已有 `calls []provider.ToolCall`，`call.ID` 就是 `call_id`；回合号从 `a` 当前 turn 取）。
- **语义**：每条记录写 `{call_id, tool, turn, model, exposure, read_only, source_kind(direct|batch_parallel), outcome, duration_ms, output_bytes}`。对齐 codex 的"归因是宿主观察、不参与执行决策"（`executed_tool_calls.rs:345-352` + `request_metadata.rs:1` 注释）。
- **禁区**：不得把 `call_id`/回合号写进任何回喂模型的内容（codex 的同类字段标了 `skip_serializing`/`schemars(skip)`，见 `protocol/src/models/executed_tool_calls.rs:412-427`）；不得改 `auditFunc` 现有 7 参顺序（`agent.go:872` 注入点 + 多处测试依赖）；归因写失败必须吞掉，不得让工具调用变失败（对齐 `executed_tool_calls.rs:369-380` 的 best-effort）。
- **默认关闭**：`cfg.Tools.AuditV2`，默认 false；开启后先只落盘不进事件流，观察一轮再打开事件。

### 刀 3 · 并发判据下沉到工具自声明

- **改动点**：`internal/gaea/tool/tool.go:18-36`（`Tool` 接口旁加可选接口 `ParallelSafe() bool`，**不塞进 `Tool` 主体**以免全量实现方破编译）、`internal/gaea/agent/batch_executor.go:259-282`（`partitionToolCalls` 增加一路判据：调用方工具实现 `ParallelSafe()` 且返回 true 时，即使命中冲突键也允许同批）、`:292-362`（`getConflictKey` 的 `default: ""` 保持不动）。
- **语义**：从"名字→键的中心映射"变成"工具自声明能力 + 中心键兜底"。对齐 `tools/src/tool_executor.rs:122-124`（默认 false）与 `registry.rs:682-685`（隐藏面强制 false）。
- **禁区**：不得放宽 `!` 前缀全局键（`!bash`/`!write`/`!ledger`/`!interrupt`/`!spawn`）——`bash` 永远串行这条不能碰（`batch_executor.go:351-352`）；`task` 的 `wpaths:` 声明键语义（313-325）不得被覆盖，否则 A10 的写路径串行保证失效；`maxParallel=8`（367）不得因新判据上调。
- **默认关闭**：`cfg.Tools.ParallelByCapability`，默认 false；先做影子模式——只记录"若按能力判据会有几个批变化"，不改变实际批次。

---

## 9. 验收断言

1. **注册面计数恒定**：`ToolRegistry` 在给定配置下注册出的条目数，与 `record_registration_metrics` 分类统计之和相等（六类之和 = 注册总数），且 `Hidden` 条目不进 `model_visible_specs`（对照 `router.rs:404-419`、`spec_plan.rs:867-879`）。可写成对 `tool_names` 与 `exposure` 的断言。
2. **计划不可变**：同一 `StepContext` 上连续两次采样请求，`model_visible_specs` 指针相同；替换 `next_step_settings`（`turn_context.rs:344` 的 `ArcSwap`）后**旧 step 的 router 不变**，新 step 才反映新快照（对齐 `step_settings.rs:37-39` 注释）。
3. **暴露面互斥**：不存在同时属于 `Direct` 与 `Deferred` 的条目；构造该组合必须走到 `unreachable!`（`spec_plan.rs:287`）。
4. **隐藏面不可并行**：`exposure == Hidden` 的工具 `supports_parallel_tool_calls()` 必为 `Some(false)`（`registry.rs:684`）。
5. **保留名不可注册**：以外部通道注册名为 `exec_command` 的 runtime，`register_external*` 返回 `false`、注册表内该名字仍指向内建、且 `first_collision()` 被填充（`registry.rs:548-556`、`576-582`）。
6. **未知工具不终止回合**：派发一个未注册名字，得到 `RespondToModel` 而非 `Fatal`，且活动回合 `tool_calls` 计数 +1（`registry.rs:708-747`）。
7. **类型错配终止回合**：给 `exec_command` 喂 `ToolPayload::Custom`，必须产出 `Fatal` 且**没有**工具结果项落历史（`registry.rs:759-775`、`parallel.rs:109`）。
8. **结果保序**：并发派发 3 个不同耗时调用（20ms/1ms/10ms），历史中工具结果项的顺序仍等于调用顺序（`turn.rs:2601-2629`）。
9. **失败隔离**：同批 3 个调用中有 1 个返回 `RespondToModel`，另外 2 个必须仍产出正常结果项，且该失败项 `success=Some(false)`（`parallel.rs:100-122`、`338-363`）。
10. **审批在沙箱之前**：对 `NeedsApproval` 的调用构造"用户拒绝"，断言**没有发生任何执行**（沙箱尝试数 0）且理由回给模型（`orchestrator.rs:203-225`）。
11. **重试需二次审批**：首试被沙箱拒绝且 `escalate_on_failure` 为真时，断言 `request_approval` 被调用第二次、且 `retry_reason` 非空（`orchestrator.rs:474-534`）。
12. **归因可降级不可失败**：把 `MAX_RETAINED_DIRECT_METADATA_BYTES` 调到极小，断言工具调用仍成功、归因被削减并打 `direct_retained` 削减指标（`executed_tool_calls.rs:410-444`）。
13. **gaea 刀 1 断言**：`ExposureMode` 打开后，`FilteredSchemas(nil)` 与旧实现逐字节一致（同配置下）；关闭时 `Hide` 行为与今天完全一致（回归钉，`tool.go:367-412`）。
14. **gaea 刀 2 断言**：并行批内两条同名 `bash` 调用，V2 审计记录能按 `call_id` 区分且 `turn` 相同；`auditFunc` 旧签名的调用次数与今天一致。
15. **gaea 刀 3 断言**：影子模式下实际批划分与今天逐批相同，仅影子日志条数增加；`bash` 仍全程串行（`batch_executor.go:351-352` 不变）。

---

## 10. 不搬 / 只借语义 / 风险

**不搬**：

- `ToolExposures` 三面位标志（`tool_executor.rs:14-29`）与 `CodeMode` 面：gaea 没有 code mode 运行时，多出的一面只会变成永远为假的字段。第一版只取三值枚举。
- `SpecPlan` 里与模型目录耦合的部分（`instrumented_*`、`experimental_supported_tools` 驱动的注册，如 `spec_plan.rs:1324-1350`、`1430-1437`）：那是 codex 与其模型目录的私有协议，gaea 无对应目录概念。
- `hosted_spec`（web_search 由服务端执行，`spec_plan.rs:690-720`）：gaea 的 `web_search` 是本地工具，搬过来会多一条永不触发的分支。

**只借语义**：

1. "**审批是 harness 强制插点，不是模型主动申请**"。gaea 现在是后者（`request_permission.go:22-37`）。借语义的做法：在 `ToolDispatcher.Check` 之后、`t.Execute` 之前加一个可选 `ApprovalGate`，对声明 `RequiresApproval()` 的工具强制走一次决策，不改现有 gate 规则语法（否则触发 asset-13 指出的 glob 粒度问题）。
2. "**暴露面是注册表字段，不是调用点行为**"。借语义即可，不必借六态。
3. "**派发归因是宿主观察、best-effort**"。借 `record_tool_call` 的"记不进也不影响执行"原则（`executed_tool_calls.rs:369-380`），不借其四层削减实现。

**风险**：

- **R1 前缀缓存抖动**：gaea 的 `Schemas()` 是缓存指纹的一部分（`cache/identity.go:64` 引用 `registry.Schemas()`）。刀 1 若让 `FilteredSchemas` 的输出顺序或描述字节有任何变化，会直接打穿前缀缓存命中率。缓解：刀 1 默认关闭 + 逐字节回归钉（验收 13）。
- **R2 并行放大副作用**：刀 3 把并行判据交给工具后，一个新工具误报 `ParallelSafe()` 就能让两个写操作重叠。缓解：`!` 全局键不可放宽（禁区），且影子模式先跑一轮。
- **R3 归因膨胀**：刀 2 若把完整参数写进审计，等于给每个回合加一份工具参数副本。缓解：只记 `output_bytes` 与 `call_id`，参数不入审计（对齐 codex 的字节预算纪律，`executed_tool_calls.rs:53`）。
- **R4 与 asset-13 打架**：若刀 1 顺手动了审批规则，会与 asset-13 §三 的"glob 无法区分命令边界"结论冲突。本文件不碰规则层，只碰暴露面与归因。

---

## 11. 证据校验

### 11.1 实际跑过的检索命令（PowerShell，均带 `-Pattern`）

```
# 目录盘点
Get-ChildItem "$r\core\src\tools" | Select Name,Length,Mode
Get-ChildItem "$r\core\src\tools\runtimes","$r\core\src\tools\handlers","$r\core\src\tools\executed_tool_calls","$r\core\src\tools\code_mode" -Recurse -File
Get-ChildItem "$r\tools\src" -Recurse -File

# 结构锚点复核（三批）
Select-String -Path "$r\core\src\tools\registry.rs" -Pattern 'pub\(crate\) trait CoreToolRuntime|pub\(crate\) struct RegisteredTool|pub struct ToolRegistry|fn register_external_with_exposure|fn supports_parallel_tool_calls|async fn dispatch_any_with_state|fn unsupported_tool_call_message|fn notify_tool_finish_if_unclaimed|async fn handle_any_tool'
Select-String -Path "$r\core\src\tools\router.rs"     -Pattern 'pub struct ToolRouter|pub fn build_tool_call|pub\(crate\) fn exposes_tool|pub fn tool_supports_parallel|pub\(crate\) fn record_registration_metrics'
Select-String -Path "$r\core\src\tools\spec_plan.rs"  -Pattern 'pub\(crate\) fn build_tool_router|pub\(crate\) fn finalize_tool_router|fn is_directly_visible|fn add_core_tool_sources|fn add_shell_tools|fn add_collaboration_tools|fn append_dynamic_tool_runtimes|fn apply_mcp_tool_exposure_policy|fn build_model_visible_specs'
Select-String -Path "$r\core\src\tools\orchestrator.rs" -Pattern 'pub async fn run<Rq|// 1\) Approval|// 2\) First attempt|// Second attempt|fn run_attempt'
Select-String -Path "$r\core\src\tools\parallel.rs"   -Pattern 'pub\(crate\) fn handle_tool_call\(|let supports_parallel|Either::Left|Either::Right|fn failure_response|CodexErr::Fatal\(message\)|abort_dispatch_span.record'
Select-String -Path "$r\core\src\tools\executed_tool_calls.rs" -Pattern 'MAX_PENDING_EXECUTED_TOOL_CALLS|MAX_RETAINED_DIRECT_METADATA_BYTES|pub\(crate\) fn record_tool_call|pub\(crate\) fn prepare_direct_call|fn recorded_call'
Select-String -Path "$r\core\src\tools\sandboxing.rs" -Pattern 'pub\(crate\) enum ExecApprovalRequirement|pub\(crate\) fn default_exec_approval_requirement|pub\(crate\) trait ToolRuntime'
Select-String -Path "$r\core\src\tools\approvals.rs"  -Pattern 'pub\(crate\) struct ApprovalContext|pub\(crate\) enum ApprovalAction|pub\(crate\) enum ApprovalResolutionSource'
Select-String -Path "$r\core\src\session\turn.rs"     -Pattern 'pub\(crate\) fn build_prompt|async fn drain_in_flight|let mut in_flight|in_flight.push_back|let tool_runtime = ToolCallRuntime::new'
Select-String -Path "$r\core\src\stream_events_utils.rs" -Pattern 'pub\(crate\) async fn handle_output_item_done|ToolRouter::build_tool_call'
Select-String -Path "$r\core\src\tools" -Recurse -Pattern '^const |^pub\(crate\) const |^pub const '   # 常量表来源

# gaea 侧
Select-String -Path "internal\gaea\agent\execute_one.go"  -Pattern 'func \(a \*AgentRunner\) executeOne|CodeUnknownTool|if !a.toolActive|a.dispatcher.Check|a.precheckTool|a.checkToolAccess|a.auditFunc\(call.Name|ct.ExecuteWithContext|t.Execute\(cctx'
Select-String -Path "internal\gaea\agent\tool_dispatch.go" -Pattern 'type ToolDispatcher struct|func \(d \*ToolDispatcher\) Check|Permission gate|PreToolUse hooks'
Select-String -Path "internal\gaea\agent\batch_executor.go" -Pattern 'func \(a \*AgentRunner\) executeBatch|func partitionToolCalls|func getConflictKey|func runParallel|maxParallel = 8|stormBreakThreshold = 3|for _, batch := range partitionToolCalls'
Select-String -Path "internal\gaea\tool\tool.go"          -Pattern 'type Registry struct|func \(r \*Registry\) Add|func \(r \*Registry\) Hide|func \(r \*Registry\) Schemas|func \(r \*Registry\) FilteredSchemas|func \(r \*Registry\) Get\(|MCPNamePrefix ='
Select-String -Path "internal\gaea\agent\agent.go"        -Pattern 'type Gate interface|auditFunc func|type ToolHooks interface'
Select-String -Path "internal\gaea\agent\discover_tools.go" -Pattern 'const DiscoverToolName|discoverMaxActivate|func \(a \*AgentRunner\) toolActive|func \(a \*AgentRunner\) activateTools'
Select-String -Path "internal\gaea\agent\agent_stream.go"  -Pattern 'return a.tools.Schemas\(\)|tools := a.currentSchemas\(\)'
```

### 11.2 行号复核结果

- 本文引用的 **codex 侧 `file:line` 全部由 `Select-String` 的输出直接比对过**（registry/router/spec_plan/orchestrator/parallel/executed_tool_calls/sandboxing/approvals/turn/stream_events_utils 十一组，逐条命中）。
- 常量表除 `MIN_WAIT_TIMEOUT_MS`/`MAX_WAIT_TIMEOUT_MS` 外，其余数值均来自 `Select-String '^const '` 的原文输出。
- **gaea 侧**：`execute_one.go`、`tool_dispatch.go`、`batch_executor.go`、`tool/tool.go`、`agent.go`、`discover_tools.go`、`agent_stream.go` 七组的行号同样逐条比对过。
- **`docs/gaea-tool-registry-map-2026-11.md`** 的行号来自完整读文件（136 行），`:3` 为单源生成声明、`:7-123` 为工具表、`:136` 为"第五装配面则清单失效"的适用条件。

### 11.3 未读 / 推断声明

**未读（本节结论不依赖它们，但读者若要施工级复现需自行打开）**：

- `core/src/tools/handlers/multi_agents/*`（`spawn.rs`/`wait.rs`/`send_input.rs`/`close_agent.rs`/`resume_agent.rs`）与 `handlers/multi_agents_v2/*` 各 handler 的**实现体**——只读了 `multi_agents_common.rs` 的常量、`multi_agents_spec.rs` 的命名空间常量、以及 `spec_plan.rs` 的注册点。
- `core/src/tools/spec_plan.rs:1070-1183`（code mode 执行器构造细节）、`core/src/tools/code_mode/*`（`delegate.rs`/`execute_handler.rs`/`wait_handler.rs`）——只用了 `PUBLIC_TOOL_NAME`/`WAIT_TOOL_NAME` 这两个符号。
- `core/src/tools/network_approval.rs`（51 KB）、`core/src/tools/events.rs`、`core/src/tools/sandboxing.rs:270-578` 的完整实现——只用其常量与 `ensure_native_sandbox`/`sandbox_override_for_first_attempt` 的签名段。
- `core/src/tools/runtimes/apply_patch.rs`、`runtimes/unified_exec.rs`（含 `launch.rs`/`snapshot_metrics.rs`）——只读了 `runtimes/mod.rs` 的头部与常量。
- `tools/src/json_schema/*`、`tools/src/tool_search.rs`、`tools/src/tool_discovery.rs` 的实现体——只用 `lib.rs:42-113` 的导出面。
- gaea `internal/gaea/agent/batch_executor.go:155-240`、`task.go`、`agent.go` 主循环（只读了接口定义段 195-242 与 `auditFunc` 字段）、`internal/gaea/permission/*`（该部分在 asset-13 已覆盖）。

**推断（标注"（推断）"的结论不构成证据）**：

- （推断）§5.2 的 `Completed{success:false}` 档：`ToolCallOutcome::Completed{success}` 的 `success` 取自 `result.success_for_logging()`（`registry.rs:940`），但**各 handler 的 `success_for_logging` 默认实现**未读，此处只按类型签名推断"存在 Ok 但自报失败"的情况。
- （推断）§7.1 "gaea 新工具默认拿不到并行"：依据 `getConflictKey` 的 `default: return ""`（`batch_executor.go:359-361`）与 `partitionToolCalls` 的 `parallel: i-start > 1 && !hasGlobal`（`:278`）——空键自身会被当作 `hasGlobal`（`:263`）而单独成批，**这是代码读出来的**；但"既有工具里有多少落在 default 分支"未逐个统计，该数量结论为推断。
- （推断）§3.6 第 7 步"非 MCP 工具才发 `notify_tool_start`"的动机：`registry.rs:842-846` 只显示条件判断，未读 `lifecycle.rs`，措辞为推断。
- （推断）`MIN_WAIT_TIMEOUT_MS`/`MAX_WAIT_TIMEOUT_MS` 的数值：`multi_agents_common.rs:22-24` 只是别名，实际值在 `codex_protocol` 常量里，**本轮未打开该文件**，故表中标"未读"。

### 11.4 引用基线说明

- codex 侧行号基于 `C:\AI\re\codex-0.165.0\oss\codex-0.165.0\codex-rs` 的**只读快照**，不随仓库变动。
- gaea 侧行号基于 `C:\AI\wubigrok` 的**当前工作树**（`internal/gaea/...` 相对 HEAD 存在未提交改动），若他人切到别的分支复核，个别行号可能偏移；本文件未对 gaea 做任何修改，唯一新增文件即本文件。
