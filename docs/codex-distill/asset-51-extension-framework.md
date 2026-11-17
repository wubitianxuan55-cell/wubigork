# asset-51 · 扩展框架（B6 · 施工级）

> 证据（codex `rust-v0.165.0`，路径相对 `codex-rs/`）：
> `ext/extension-api/src/{lib.rs, registry.rs, contributors.rs, state.rs, turn_admission.rs, tool_policy.rs,
> session_isolation.rs, message_lifecycle.rs, model_request.rs, catalog_refresh.rs, user_instructions.rs}`、
> `ext/extension-api/src/capabilities/{mod,events,metrics,conversation_history,response_items}.rs`、
> `ext/extension-api/src/contributors/{thread_lifecycle,turn_lifecycle,turn_input,context,approval_review,
> skill_invocation,tool_lifecycle,prompt,world_state,mcp}.rs`、`ext/extension-api/Cargo.toml`；
> 实现样板 `ext/skills/src/{extension.rs,host_service.rs}`、`ext/web-search/src/extension.rs`、
> `ext/image-generation/src/extension.rs`、`ext/history-notes/src/extension.rs`、
> `ext/multi-env-copy-file/src/extension.rs`、`ext/goal/src/{extension.rs,spec.rs}`、`ext/mcp/src/lib.rs`、`ext/items/src/lib.rs`；
> 装配与广播 `app-server/src/{extensions.rs,turn_admission.rs}`、`core/src/tasks/lifecycle.rs`、
> `core/src/tools/lifecycle.rs`、`core/src/tools/spec_plan.rs`、`core/src/{session/session.rs,session/mod.rs,
> session/turn.rs,session/turn_input.rs,session/mcp.rs,session/world_state.rs,stream_events_utils.rs,skills.rs,
> mcp.rs,thread_manager.rs,codex_delegate.rs,guardian/decision.rs}`。
> gaea 只读对照：`internal/gaea/agent/{agent.go,tool_dispatch.go}`、`internal/gaea/control/controller.go`、
> `internal/gaea/boot/{boot.go,sysprompt.go,plugins.go,dynamic_tools.go}`、`internal/gaea/tool/registrymap.go`、
> `docs/gaea-tool-registry-map-2026-11.md`。
> 只出机制与施工卡，不含任何代码改动；未读部分见 §12。

---

## 1. 一句话结论

codex 的"扩展"不是一个 `trait Extension`，而是**一张 16 槽的 typed contributor registry**：
宿主在 `app-server/src/extensions.rs:52`（`thread_extensions`）一处把所有能力按 trait 分别塞进固定槽位，
`Arc::new(builder.build())`（`extensions.rs:136`）之后不可变；Core 在**每个生命周期点按注册顺序广播**
（`for contributor in registry.xxx_contributors() { contributor.on_xxx(..).await }`，
如 `core/src/tasks/lifecycle.rs:28`、`core/src/tools/lifecycle.rs:41`）。
扩展**拿不到全局单例**：只有（a）按 session/thread/turn/step 四层切分的类型化 KV `ExtensionData`
（`state.rs:49`），（b）宿主在构造期显式递给它的句柄（event sink、metrics、mcp client、环境 fs、
`AuthManager`、`HttpClientFactory`），（c）**能力的缺席即边界**——`ext/extension-api/Cargo.toml:32` 里
tokio 只开 `time` 一个 feature，整个 API crate 没有 reqwest、没有 `std::process`。
所以隔离靠"不给你"，不靠沙箱。

---

## 2. 机制地图

| 关注点 | 文件:行 | 作用 |
|---|---|---|
| 注册表定义（不可变） | `ext/extension-api/src/registry.rs:161`–`:178` | 16 个字段：14 条 contributor 向量 + `event_sink` + `turn_start_admission` |
| 注册表构造器 | `ext/extension-api/src/registry.rs:22`–`:158` | 每个槽一个 `*_contributor()` 方法，`build()` 冻结（`:155`） |
| 槽位读取 | `ext/extension-api/src/registry.rs:207`–`:306` | `admit_turn_start()`、各 `*_contributors()` getter |
| 审批聚合语义 | `ext/extension-api/src/registry.rs:255`–`:266` | 注册顺序扫描，**第一个 `Some` 胜出** |
| 14 个 contributor trait | `ext/extension-api/src/contributors.rs:89`–`:490` | 每个 trait 一小节，见 §3.1 |
| 扩展私有状态 KV | `ext/extension-api/src/state.rs:49`–`:137` | `TypeId` 为键、`Arc<dyn Any+Send+Sync>` 为值的四层 store |
| 启动准入闸 | `ext/extension-api/src/turn_admission.rs:10`–`:12` | `admit_turn_start() -> Option<Box<dyn Send>>` |
| 工具白名单 | `ext/extension-api/src/tool_policy.rs:10`–`:43` | `allowed_tools` 等四项，启动期一次冻结 |
| 会话隔离 | `ext/extension-api/src/session_isolation.rs:11`–`:24` | `Inherit` / `Isolated` + `IsolatedSessionExtensions` |
| 宿主能力 trait | `capabilities/events.rs:19`、`capabilities/metrics.rs:5`、`capabilities/conversation_history.rs:10`、`capabilities/response_items.rs:15`、`user_instructions.rs:44`/`:67`、`contributors/approval_review.rs:25` | event sink / metrics / 历史快照 / 同回合注入 / 指令提供者 / 同步审批复核 |
| **唯一装配点** | `app-server/src/extensions.rs:52`–`:137` | 14 个模块依次 `install(&mut builder, …)` |
| 事件外发实现 | `app-server/src/extensions.rs:179`–`:308` | `AppServerExtensionEventSink`：goal/queue 事件与 warning |
| 回合广播 | `core/src/tasks/lifecycle.rs:16`–`:124` | turn start/stop/abort/error、thread idle |
| 线程广播 | `core/src/session/session.rs:1735`、`codex_thread.rs:342`/`:360`、`session/handlers.rs:345` | thread start/ready/resume/stop |
| 工具广播 | `core/src/tools/lifecycle.rs:21`–`:162`、`core/src/tools/parallel.rs:164` | dispatch/start/command/mcp-result/finish/timing |
| 工具注入 | `core/src/tools/spec_plan.rs:350`–`:366` | `tools_for_step()` 收集成 step 工具集 |
| 上下文注入 | `core/src/session/mod.rs:4595`–`:4622`（turn）、`:4681`–`:4699`（thread） | 三槽位分流 |
| world state 注入 | `core/src/session/world_state.rs:350`–`:368` | `add_extension_section()` |
| 回合输入注入 | `core/src/session/turn.rs:1265`–`:1319` | user-role 片段，带取消 |
| 审批入口 | `core/src/guardian/decision.rs:44`–`:121` | `ReviewRequest::decide(&extensions)` |
| 最完整实现 | `ext/skills/src/extension.rs:610`–`:623` | 一个对象注册到 7 个槽位 |

---

## 3. 扩展契约

### 3.1 一个扩展**声明什么**

14 个 facet trait，各自独立可选实现（无统一 trait，`lib.rs:54`–`:112` 逐条 re-export）：

| trait | 声明的方法 | 证据 |
|---|---|---|
| `McpServerContributor<C>` | `id()`（**唯一带稳定身份的方法**）、`contribute()`、`hosted_plugin_ids()`、`selected_plugins()` | `contributors.rs:89`–`:115` |
| `ContextContributor` | `contribute_thread_context()`、`contribute_turn_context()`、`contribute_world_state()` | `contributors.rs:122`–`:158` |
| `ThreadLifecycleContributor<C>` | `on_thread_start/ready/resume/idle/stop` | `contributors.rs:165`–`:210` |
| `TurnLifecycleContributor` | `turn_start_phase()`、`requires_mcp_runtime()`、`on_turn_start/stop/abort/error`、`refresh_catalog_for_step()`、`on_item_completed()` | `contributors.rs:217`–`:283` |
| `TurnInputContributor` | `contribute() -> Vec<Box<dyn ContextualUserFragment>>`（**唯一必填方法**） | `contributors.rs:291`–`:302` |
| `ConfigContributor<C>` | `on_config_changed(prev, new)`（同步） | `contributors.rs:308`–`:318` |
| `TokenUsageContributor` | `on_token_usage()` | `contributors.rs:325`–`:339` |
| `SkillInvocationContributor` | `requires_host_skill_discovery()`、`on_skill_invocation()` | `contributors.rs:345`–`:363` |
| `ToolContributor` | `prepare_for_step()`、`tools()`、`tools_for_step()` | `contributors.rs:382`–`:408` |
| `ToolLifecycleContributor` | `on_tool_dispatch/start/command_start/mcp_tool_result/finish/timing`、`on_code_mode_cell_timing` | `contributors.rs:415`–`:464` |
| `ApprovalReviewContributor` | `decide() -> Option<ApprovalDecision>` | `contributors.rs:468`–`:476` |
| `TurnItemContributor` | `contribute(.., item: &mut TurnItem) -> Result<(), String>` | `contributors.rs:483`–`:490` |
| `ModelRequestContributor` | `request() -> Option<Box<dyn ModelResponseInterceptor>>` | `model_request.rs:33`–`:44` |
| `MessageLifecycleContributor` | `on_publish()`、`on_consumed()`（同步） | `message_lifecycle.rs:18`–`:23` |

声明里没有"能力位/权限清单"，只有四个**自述开关**会改变宿主的调度而非授权：
`turn_start_phase`（默认 `BeforeTaskRegistration`，`contributors.rs:220`–`:222`）、
`requires_mcp_runtime`（默认 `false`，`:226`–`:228`）、
`requires_host_skill_discovery`（默认 `true`，`:349`–`:351`，在 `core/src/session/session.rs:655` 处
决定要不要扫描宿主技能根）、`McpServerContributor::id()`（仅用于 provenance，`contributors.rs:91`）。

**一个 struct 被注册多次是常态**：`ext/skills/src/extension.rs:616`–`:623` 把同一个 `Arc<SkillsExtension>`
先后推进 thread_lifecycle、turn_lifecycle、config、context（prompt）、turn_input、skill_invocation、tool
七个槽位。这解释了为什么没有 `trait Extension`——"扩展"是**注册动作的集合**，不是类型。

### 3.2 一个扩展**拿到什么**

(a) **四层类型化 KV**（`state.rs:49`–`:137`）：`session_store` / `thread_store` / `turn_store` / `step_store`
每个都是 `ExtensionData`，`get<T>()`、`get_or_init<T>()`、`insert<T>()`、`insert_if<T>()`、`remove<T>()`
（`:74`–`:132`）；类型不匹配直接 `unreachable!`（`:144`），**类型即权限**。
四层作用域的身份来自 `level_id()`（`:69`）——skills 直接把它当 thread_id 用（`extension.rs:205`、`:220`）。

(b) **宿主注入的句柄**（最完整的一份是 `ThreadStartInput`，`contributors/thread_lifecycle.rs:18`–`:35`）：
`config: &C`、`session_source`、
`persistent_thread_state_available`、`environments`、`mcp_resource_client: Option<Arc<McpResourceClient>>`、
`extension_metrics`、两个 store。

(c) **上下文快照与惰性证据**：`ToolStartInput.permissions` 是一个 **future**
（`contributors/tool_lifecycle.rs:138`–`:139`），注释明确"只有 await 才算取到证据，`None` 阻止缓存审批"；
`conversation_history: Arc<dyn ConversationHistorySnapshot>`（`:145`，定义 `capabilities/conversation_history.rs:10`）。

(d) **取消**：只有两处被显式套上取消——`prepare_for_step` 走
`core/src/session/mod.rs:4137` 的 `.or_cancel(cancellation_token)`（取消 → 整个回合 `Err`），
turn-input `contribute` 走 `core/src/session/turn.rs:1308`（取消 → 返回 `None`，该段输入丢弃）。

(e) **出站通道**：`ExtensionEventSink::emit/emit_warning`（`capabilities/events.rs:19`–`:28`），
宿主实现见 `app-server/src/extensions.rs:179`。扩展只能"发事件"，持久化/排序/扇出全归宿主。

### 3.3 **被谁在什么时候调用**

| 槽位 | 调用者（file:line） | 触发时机 |
|---|---|---|
| `turn_start_admission` | `core/src/session/turn_input.rs:409` | 任何**非** parent-delegated 的 turn-input 提交之前（`:396`–`:408` 先判委派） |
| `thread_lifecycle.on_thread_start` | `core/src/session/session.rs:1735` | session 构造、history 回放完、`ExtensionData` 就绪之后 |
| `..on_thread_ready` / `on_thread_resume` | `core/src/codex_thread.rs:342` / `:360` | 线程注册进宿主 / 从持久历史重建 runtime |
| `..on_thread_idle` | `core/src/tasks/lifecycle.rs:78` | active_turn 为空且无 mailbox 触发项（`:64`–`:76` 双检） |
| `..on_thread_stop` | `core/src/session/handlers.rs:345` | runtime 拆除、持久历史关闭之前 |
| `turn_lifecycle.on_turn_start` | `core/src/tasks/lifecycle.rs:37` | 按 `turn_start_phase` 过滤后（`:29`），`RegularTaskStart` 阶段还会先按需刷 MCP（`:32`–`:36`） |
| `..refresh_catalog_for_step` | `core/src/session/mcp.rs:441` | 每个模型 step 前；返回 `true` → `mark_mcp_runtime_dirty()`（`:452`） |
| `..on_item_completed` | `core/src/session/mod.rs:2834` | 每个 turn item 落定后，流式 delta 不受影响 |
| `..on_turn_stop/abort/error` | `core/src/tasks/lifecycle.rs:52`/`:95`/`:113` | 回合结束 / 中止（先回调再发事件）/ 观测到错误 |
| `tool_contributor.prepare_for_step` | `core/src/session/mod.rs:4123` | 每个 step 组装工具前**一次**（带取消） |
| `..tools_for_step` | `core/src/tools/spec_plan.rs:360` | 同上，紧接其后收集可执行工具 |
| `tool_lifecycle.on_tool_dispatch` | `core/src/tools/parallel.rs:164` | 直接调用进入 dispatch、readiness/hook/policy 之前（**同步**） |
| `..on_tool_start` | `core/src/tools/lifecycle.rs:41` | pre-tool hooks 定稿、执行之前 |
| `..on_mcp_tool_result` | `core/src/tools/lifecycle.rs:94` | MCP 返回后、送客户端与模型之前（可改 `result`） |
| `..on_tool_finish` | `core/src/tools/lifecycle.rs:148` | 返回/被阻断/失败/取消统一出口 |
| `context_contributor.contribute_thread_context` | `core/src/session/mod.rs:4681` | 上下文装配（thread 级） |
| `..contribute_turn_context` | `core/src/session/mod.rs:4601` | 上下文装配（turn 级） |
| `..contribute_world_state` | `core/src/session/world_state.rs:350` | 每个采样步的 world state |
| `turn_input_contributor.contribute` | `core/src/session/turn.rs:1300` | 回合输入落库之前 |
| `approval_review_contributor.decide` | `core/src/guardian/decision.rs:114` → `registry.rs:260` | 审批请求需要 automatic review 时 |
| `skill_invocation_contributor` | `core/src/skills.rs:136`（显式）/`:234`（隐式） | 技能加载或命令分发之内联 |
| `mcp_server_contributor` | `core/src/mcp.rs:223`、`:297` | MCP 解析（按 step 或全局） |
| `config_contributor.on_config_changed` | `core/src/session/mod.rs:2365` | 线程配置提交之后（**同步**） |
| `token_usage_contributor` | `core/src/session/mod.rs:5282` | 记录 token usage 后、发客户端通知前 |
| `message_lifecycle_contributor` | `core/src/agent/control.rs:267` | 本地 agent 控制通道发布消息时（**同步**） |
| `model_request_contributor` | `core/src/session/session.rs:1838` | 每次模型请求（`Generation`/`Warmup`，`model_request.rs:12`–`:17`） |

---

## 4. 生命周期与失败语义

**总规律先说**：除三个返回类型带"否"语义的槽位外，**所有生命周期回调签名都是 `ExtensionFuture<'a, ()>`
或 `fn(..)`**（`contributors.rs:167`、`:233`、`:260`、`:269`、`:277`、`:327`、`:354`、`:424`、`:432`、`:437`、`:445`）。
也就是说：**扩展在结构上没有"让回合失败"的能力**，失败只能表达为"不生产力 / 发 warning / 不 return"。
真正能改变流程的只有三处：`TurnItemContributor -> Result<(),String>`、`ApprovalReviewContributor -> Option`、
`TurnStartAdmission -> Option`。

| 钩子 | 触发点 | 同步性 | 失败后果 |
|---|---|---|---|
| `admit_turn_start` | `core/src/session/turn_input.rs:409` | 同步 | 返回 `None` → `TurnInputSubmission::NotSubmitted{ServerDraining}`（`:410`–`:414`），**不消耗 pending 输入**（`turn_admission.rs:6`–`:7`）；delegated 子代理输入不过闸（`:406`–`:408`） |
| `on_thread_start` | `core/src/session/session.rs:1735` | 异步、串行 `await` | 返回 `()`；**不会失败**，但会阻塞 session 构造 |
| `on_thread_ready` / `on_thread_resume` | `core/src/codex_thread.rs:342` / `:360` | 异步 | 同上 |
| `on_thread_idle` | `core/src/tasks/lifecycle.rs:78` | 异步 | 同上；文档允许回调里"提交后续输入"，但是否成回合由宿主决定（`contributors.rs:190`–`:194`） |
| `on_thread_stop` | `core/src/session/handlers.rs:345` | 异步 | 同上；合约要求"返回前自己 cancel 并 join 后台任务"（`contributors.rs:202`–`:203`） |
| `turn_start_phase` | `core/src/tasks/lifecycle.rs:29` | **同步纯函数** | 无副作用；决定本回调**是否被调用**（不匹配直接 `continue`） |
| `requires_mcp_runtime` | `core/src/tasks/lifecycle.rs:33` | **同步纯函数** | 仅 `RegularTaskStart` 生效；`true` → 先 `refresh_mcp_if_dirty()` |
| `on_turn_start` | `core/src/tasks/lifecycle.rs:37` | 异步 | 返回 `()`；`RegularTaskStart` 阶段可能被取消（`contributors.rs:230`–`:232`），stop/abort 必须容忍半初始化 |
| `refresh_catalog_for_step` | `core/src/session/mcp.rs:441` | 异步 → `bool` | `true` 不阻断，只 `mark_mcp_runtime_dirty()`（`:450`–`:453`） |
| `on_item_completed` | `core/src/session/mod.rs:2834` | 异步 | `()`；不得修改 item、不得阻塞 delta（`contributors.rs:248`） |
| `on_turn_stop` / `on_turn_abort` / `on_turn_error` | `core/src/tasks/lifecycle.rs:52`/`:95`/`:113` | 异步 | `()`；abort 回调**先于** aborted 事件发出（`contributors.rs:267`–`:268`） |
| `prepare_for_step` | `core/src/session/mod.rs:4123` | 异步 + `.or_cancel()` | 取消 → `Err(CodexErr::TurnAborted)`（`:4137`），**这是唯一会中止回合的扩展路径** |
| `tools_for_step` | `core/src/tools/spec_plan.rs:360` | **同步**（返回 `Vec`） | 无法失败；返回空 = 本步不提供工具（`contributors.rs:389`–`:393`） |
| `on_tool_dispatch` | `core/src/tools/parallel.rs:164` | **同步** | 不得阻断、不得影响执行（`contributors.rs:417`、`tool_lifecycle.rs:259`） |
| `on_tool_start` | `core/src/tools/lifecycle.rs:41` | 异步 | `()`；被 hooks 阻断的调用**不会**走到这里（`contributors.rs:421`–`:423`） |
| `on_mcp_tool_result` | `core/src/tools/lifecycle.rs:94` | 异步 | `()`，但 `result: &mut CallToolResult`（`tool_lifecycle.rs:194`）→ **可改内容不可改流程** |
| `on_tool_finish` | `core/src/tools/lifecycle.rs:148` | 异步 | `()`；不保证有对应 start（阻断/取消，`contributors.rs:441`–`:444`） |
| `turn_input.contribute` | `core/src/session/turn.rs:1300` | 异步 + `.or_cancel()` | 取消 → `.ok()?` 返回 `None`（`:1308`–`:1310`），整段扩展输入丢弃 |
| `turn_item.contribute` | `core/src/stream_events_utils.rs:243` | 异步 → `Result` | `Err` → `warn!("turn item contributor failed")` 后继续（`:245`–`:251`）**只警告** |
| `decide`（审批） | `core/src/guardian/decision.rs:114` | 异步 | `None` → 下一个 contributor → 全 `None` → 走宿主用户流；`None` **绝不是隐式放行**（`decision.rs:44`） |
| `on_config_changed` | `core/src/session/mod.rs:2365` | **同步** | 返回 `()`；实现里常直接 `insert/remove` thread store（`history-notes/extension.rs:98`、`:66`） |
| `on_token_usage` | `core/src/session/mod.rs:5282` | 异步 | `()`；合约要求"保持廉价"（`contributors.rs:322`–`:324`） |
| `on_publish` / `on_consumed` | `core/src/agent/control.rs:267` | **同步** | 合约："不得阻塞"（`message_lifecycle.rs:16`–`:17`） |
| `on_skill_invocation` | `core/src/skills.rs:136` | 异步 | `()`；随技能加载/命令分发内联（`contributors.rs:343`–`:344`） |

**warning 是唯一的"软失败"通道**：`emit_warning` → `AppServerExtensionEventSink` 截断到
`MAX_EXTENSION_WARNING_BYTES = 256`（`app-server/src/extensions.rs:176`、`:268`–`:274`，按 UTF-8 char boundary
回退），等订阅者最多 `EXTENSION_WARNING_SUBSCRIBER_TIMEOUT = 10s`（`:177`、`:292`–`:305`），
超时**丢警告**而不是丢回合。skills 自己就这么用：渲染超预算、读书失败、目录告警全部走这条
（`ext/skills/src/extension.rs:220`、`:237`、`:416`、`:484`）。

**没有 panic 兜底**：`core/src` 与 `ext/` 全树 `catch_unwind` 只有一处，且在 `ext/extension-api/tests/state.rs:134`
（测试）。所以一个 panic 的 contributor 会顺着 `await` 打穿宿主任务——这是 codex 自己的债，见 §11 风险 1。

---

## 5. 扩展与工具面 / 上下文 / 审批的交互

### 5.1 工具面：只"新增"，不改"别人的 schema"，不拦"调用"

**新增工具**——`ToolContributor::tools()`（thread 级）/ `tools_for_step()`（step 级，
默认回落到 `tools()`，`contributors.rs:400`–`:407`），在
`core/src/tools/spec_plan.rs:350`–`:366` 按 contributor 顺序 `flat_map` 成 step 工具集：

```rust
// core/src/tools/spec_plan.rs:354
session.services.extensions.tool_contributors().iter()
    .flat_map(move |c| c.tools_for_step(&session_store, &thread_store, step_store))
```

实例：`web.run`（`ext/web-search/src/extension.rs:122`–`:150`，用 `thread_store` 里的
`WebSearchExtensionConfig.available` 决定注册与否，`:130`–`:135`）、
`image_generation`（`ext/image-generation/src/extension.rs:86`–`:112`）、
`copy_file`（`ext/multi-env-copy-file/src/extension.rs:15`–`:23`，一行 `vec![Arc::new(CopyFileTool)]`）、
goals 三件套（`ext/goal/src/spec.rs:9`–`:11` 定义 `get_goal`/`create_goal`/`update_goal`，
`ext/goal/src/extension.rs:555`–`:566` 产出）、history-notes 全动作表
（`ext/history-notes/src/extension.rs:206`–`:218`，`tools_for_step` 还会读 `step_store` 里的
`HistoryToolMessages`/`NotesToolMessages`，`:180`–`:181`）。

**改写 schema**——扩展**不能**改别家工具的 schema。能做的是三件：
① 用 `ToolPolicy` 整条过滤（`spec_plan.rs:377` `hosted_specs.retain(|s| registry.tool_policy.allows(..))`，
策略本体 `ext/extension-api/src/tool_policy.rs:35`–`:43`）；
② 同名注册挤掉内建（冲突被记账：`spec_plan.rs:387`–`:389` `registry.remove(..).is_some() → record_collision`）；
③ 通过 `expose_additional_permissions` 决定要不要广告"额外权限"参数（`tool_policy.rs:19`–`:20`）。

**拦截调用**——结构上做不到。`ToolStartInput.payload: &'a ToolPayload` 是不可变引用
（`contributors/tool_lifecycle.rs:143`），并且 trait 文档把边界写死了：

> "Use `ToolContributor` for owning a tool implementation and hooks for policy that changes tool payloads."
> —— `ext/extension-api/src/contributors.rs:411`–`:414`

即：**改工具参数归 hooks（见 asset-05）**，扩展层只观测。唯一能改的是**结果**：
`on_mcp_tool_result(.., result: &'a mut CallToolResult)`（`tool_lifecycle.rs:194`，
注释"Changes feed the normal client and model output paths"）与 `TurnItemContributor` 的
`item: &'a mut TurnItem`（`contributors.rs:488`）。

### 5.2 上下文装配：三个槽位 + 一个 world state 章节

`PromptSlot` 只有三值（`contributors/prompt.rs:11`–`:16`）：

```rust
pub enum PromptSlot { DeveloperPolicy, DeveloperCapabilities, ContextWindow }
```

- thread 级 `contribute_thread_context(session_store, thread_store)`（`contributors.rs:124`–`:135`）
  → `core/src/session/mod.rs:4681`–`:4699`：`ContextWindow` 收进 `context_window_hints`，另两个进
  `developer_sections`（`:4690`–`:4697`）。
- turn 级 `contribute_turn_context(TurnContextContributionInput)`（`contributors.rs:137`–`:146`；
  入参带 `model_context_window`，`contributors/context.rs:18`–`:19`）
  → `core/src/session/mod.rs:4601`–`:4621` 全部拼成 developer message。
- **turn 输入级** `TurnInputContributor::contribute` 产出的是 **user 角色片段**
  （`Vec<Box<dyn ContextualUserFragment>>`，`contributors.rs:301`），
  在 `core/src/session/turn.rs:1311`–`:1315` `into_boxed_response_item()` 后被当作本回合输入的一部分。
  这是"扩扩展能注入 user 消息"的唯一口子，也是 skills 把"用户显式点名的技能正文"塞进去的路径
  （`ext/skills/src/extension.rs:467`、`:515`）。
- **world state 章节**：`contribute_world_state(WorldStateContributionInput)`
  （`contributors.rs:148`–`:157`）产出 `WorldStateSectionContribution`
  ——扩展自己拥有 `id`、比较快照与 diff 渲染函数，**harness 只负责持久化与信封**
  （`contributors/world_state.rs:94`–`:104`，`render_diff` 签名 `:110`–`:125`）；
  宿主侧 `core/src/session/world_state.rs:350`–`:368` `add_extension_section()`，
  上一份快照从 `previous_world_state` 传入（`contributors/world_state.rs:27`–`:30`，注释明确"压缩后可能只剩保留元数据"）。

顺序 = **注册顺序**（都是 `Vec` push，`registry.rs:76`/`:84`/`:91`/`:96`；读取按 slice 迭代）；
体积由**生产者**负责，渲染不截断（`world_state.rs:41`–`:43`）。

### 5.3 审批：own 整张决策，`None` 不是放行

`ApprovalReviewContributor::decide(&ApprovalDecisionInput) -> Option<ApprovalDecision>`（`contributors.rs:468`–`:476`），
`ApprovalDecision` 三态 `Allow / Reviewed(..) / AskUser`（`contributors/approval_review.rs:14`–`:19`），
聚合在 `registry.rs:255`–`:266`（**注册顺序第一个 `Some` 胜出**），调用点
`core/src/guardian/decision.rs:114` `.decide(&session.services.extensions)`。
边界写在模块头一行：

> "A review only satisfies the review gate; the host enforces permissions."
> —— `ext/extension-api/src/contributors/approval_review.rs:1`

即扩展的审批回答**只满足"审"这一关**，权限仍由宿主策略强制；且 `None` 表示"我不管"，
一路 `None` 到底就走用户流（`decision.rs:44`："No contributor is never an implicit allow"）。
取消优先级高于扩展结论：即便拿到缓存决策，`history_reset`/`cancellation` 已取消也强制 `Abort`
（`decision.rs:116`–`:121`）。实现范例 `ext/guardian-v2/src/async_scorer/extension.rs:208`
（`registry.approval_review_contributor(Arc::new(GuardianApprovalReviewer{..}))`）。

### 5.4 与插件（asset-17）的分工：交汇在 **MCP 服务器** 与 **技能根**

插件 = 资源包（manifest 里全是路径：skills / mcp_servers / apps / hooks，无一行可执行代码，见 asset-17 §一）；
扩展 = 代码（Rust trait 实现）。两者只在**三个数据面**交汇，不会互相加载：

1. **MCP 服务器解析**：`McpServerContributor::selected_plugins(context, plugins_config)`
   让扩展声明"哪些插件被选中"（`contributors.rs:103`–`:114`），返回
   `SelectedPlugin { selected_root_id, plugin_id, identity, enabled, mcp: ExtensionFuture<..> }`
   （`contributors/mcp.rs:188`–`:194`），**MCP 数据是延迟的 future**（`:185`–`:187` 注释：
   只要身份的可丢掉 `mcp`）。宿主侧消费在 `core/src/mcp.rs:211`–`:240`，
   并把 `available_roots`/feature flag/`disabled_plugin_ids` 逐层过滤（`:232`–`:240`）。
2. **MCP 覆盖**：`McpServerContribution` 五种（`Set` / `SetWithProtocolMode` / `HostedApps` /
   `HostedPluginConnectors` / `Remove`，`contributors/mcp.rs:208`–`:236`）；
   `HostedPluginConnectors` 的注释就是分工红线：**执行器目录里的插件必须走 `selected_plugins`，
   即使它只提供 connectors**（`:228`–`:229`）。
3. **来源标注**：工具生命周期里拿到 `McpToolSource::Plugin { id, root }` 与 `SelectedPlugin`
   （`contributors/tool_lifecycle.rs:45`–`:61`），宿主用 `from_prepared_call` 判定 provenance
   （`:75`–`:104`）；技能侧则是 `PluginSkillRoot` 进 `HostSkillsLoadInput.effective_skill_roots`
   （`ext/skills/src/host_service.rs:53`、`:70`–`:77`）。

**关键点**：插件**永远不被扩展加载器加载**，扩展也**不读插件目录**——插件由宿主
`plugins_manager` 解析（`core/src/mcp.rs:211` 外），扩展只通过上面三个接口"看见"插件身份。

---

## 6. 隔离与授权边界

**能不能读文件 / 发网络 / 改配置？**（结论：API 层三个都没有，全靠宿主给句柄）

| 面 | 事实 | 证据 |
|---|---|---|
| 网络/进程 | `codex-extension-api` 的 tokio 只开 `features = ["time"]`；依赖表里无 reqwest / 无 process | `ext/extension-api/Cargo.toml:32`、`:16`–`:32` |
| 文件系统 | 只以**入参引用**出现：`TurnInputEnvironment.fs: &'a dyn EnvironmentAccess`、`CommandStartInput.file_system: &'a dyn ExecutorFileSystem` | `contributors/turn_input.rs:16`、`contributors/tool_lifecycle.rs:172` |
| HTTP / Auth | 由宿主构造扩展时传入 `Arc<AuthManager>`、`HttpClientFactory`、`ModelProviderInfo` | `app-server/src/extensions.rs:34`–`:50`（`ThreadExtensionDependencies`）、`:112`–`:114` |
| 改配置 | 只能**观察**：`on_config_changed(prev, new)`，且注释要求把两个值当不可变快照 | `contributors.rs:304`–`:317` |
| 共享状态 | 只有四层 `ExtensionData`；键是 `TypeId`，跨扩展共享靠"约定同一个类型" | `state.rs:8`–`:19`、`:74`–`:96` |
| 隔离会话 | `SessionIsolation::Isolated` → 用 `empty_extension_registry()` 或调用方显式 `IsolatedSessionExtensions` | `session_isolation.rs:11`–`:24`、`core/src/thread_manager.rs:2320`–`:2325`、`core/src/codex_delegate.rs:98`–`:105` |
| 工具白名单 | `ToolPolicy.allowed_tools`（`None`=不限，空表=一个都不给，`tool_policy.rs:12`） | `tool_policy.rs:10`–`:21`、`:35`–`:43` |

**谁授权**：宿主在**构造期**授权，不是运行期申请。`app-server/src/extensions.rs:72` 起逐个 feature flag 门控
（`Feature::MultiEnvCopyFile` `:73`–`:78`、`Goals` 走 config 闭包 `:95`–`:98`、
skills 的 `include_instructions`/`max_context_tokens`/`shadow_selection_enabled` `:125`–`:133`）。
扩展拿到什么，取决于 `ThreadExtensionDependencies` 里有什么——**没有能力白名单这层东西，因为不需要**
（给不了的就不在结构体里）。

**`ToolPolicy` 是唯一的"上限"**，并且**启动期一次冻结**：

> "Supply through `ExtensionDataInit` before starting a thread. The host captures the policy once;
> later extension-state changes cannot relax it." —— `tool_policy.rs:6`–`:8`

宿主侧取用：`core/src/session/session.rs:1053`–`:1062`，reviewer 会话若未显式给，则回落到
`codex_guardian_reviewer::reviewer_tool_policy()`（`:1057`–`:1059`）。

**边界外的洞（要抄时注意）**：`SessionIsolation::Isolated` 只"移除继承"，不"授予"——
注释明写 "Isolation only removes inherited capabilities; it never grants review authority."
（`session_isolation.rs:1`–`:2`）；`Isolated` 仍保留 managed execution rules（`:16`–`:18`）。
换句话说 **codex 的隔离是"少给"，不是"沙箱"**；真正的执行隔离在 `sandboxing/`、`bwrap`、
`windows-sandbox-rs`（本文件未读）。

---

## 7. 常量与上限

| 常量 | 值 | 位置 | 生产/测试 |
|---|---|---|---|
| `MAX_EXTENSION_WARNING_BYTES` | 256 | `app-server/src/extensions.rs:176` | 生产 |
| `EXTENSION_WARNING_SUBSCRIBER_TIMEOUT` | 10 s | `app-server/src/extensions.rs:177` | 生产 |
| `CatalogRefreshBackoff` 首延迟 / 倍率 / 上限 | 30 s / ×2 / 300 s | `ext/extension-api/src/catalog_refresh.rs:45`–`:46` | 生产 |
| `MAX_THREAD_HINT_BYTES` | 4 000 | `ext/history-notes/src/extension.rs:33` | 生产 |
| `DEFAULT_SKILL_METADATA_CHAR_BUDGET` | 8 000 | `ext/skills/src/render.rs:20` | 生产 |
| `MAX_CONFIGURED_SKILL_METADATA_TOKEN_BUDGET` | 10 000 | `ext/skills/src/render.rs:21` | 生产 |
| `MAX_SKILL_PROMPT_BYTES` | 8 000 | `ext/skills/src/render.rs:22` | 生产 |
| `SKILL_METADATA_CONTEXT_WINDOW_PERCENT` | 2 % | `ext/skills/src/render.rs:23` | 生产 |
| `MAX_CATALOG_SKILL_DESCRIPTION_CHARS` | 1 024 | `ext/skills/src/render.rs:24` | 生产 |
| `APPROX_BYTES_PER_TOKEN` | 4 | `ext/skills/src/render.rs:26` | 生产 |
| `MAX_SKILL_NAME_BYTES` | 256 | `ext/skills/src/render.rs:29` | 生产 |
| `MAX_SKILL_PATH_BYTES` | 1 024 | `ext/skills/src/render.rs:30` | 生产 |
| `MAX_CONCURRENT_ROOT_SCANS` | 8 | `ext/skills/src/loader/mod.rs:18` | 生产 |
| `CONFIG_SKILLS_CACHE_CAPACITY` | 32 | `ext/skills/src/host_service.rs:41` | 生产 |
| `max_goal_token_budget` | 由 config 传入（`None`=不限） | `ext/goal/src/extension.rs:56`、`app-server/src/extensions.rs:97` | 生产 |
| 警告截断断言（64 个 emoji → 256 B） | 测试值 | `app-server/src/extensions.rs:396`–`:406` | **测试** |

---

## 8. gaea 对照

| 维度 | gaea 现状（file:line） | codex 口径（file:line） | 判定 |
|---|---|---|---|
| 扩展点 | **无**：`internal/gaea` 全树 `Contributor`/`ExtensionRegistry`/`LifecycleContributor` **0 命中**；工具面靠 `boot.Build` 里逐行 `reg.Add`/`addGated` 硬编码（`internal/gaea/boot/boot.go:598`、`:603`、`:611`、`:649`、`:758`、`:993`、`:1008`） | 16 槽 typed registry，`app-server/src/extensions.rs:52`–`:137` 一处装配 | **缺**（结构性缺口，不是少写一个函数） |
| 生命周期钩子 | 只有工具级 `ToolHooks` 7 方法：`PermissionRequest`/`PreToolUse`/`PostToolUse`/`PostLLMCall`/`HasPostLLMCall`/`SubagentStop`/`PreCompact`（`internal/gaea/agent/agent.go:226`–`:242`）；回合只有 `TurnStarted` 事件（`internal/gaea/agent/agent_run.go:59`） | thread 5 + turn 6 + tool 7 + 审批/配置/用量/技能/消息/模型请求 共 14 个 facet（§3.1） | **口径不同**：gaea 是"外部脚本钩子"，codex 是"进程内 trait 广播" |
| 改工具参数 | 允许：`PermissionRequest` 可返回 `modifiedArgs`（`internal/gaea/agent/tool_dispatch.go:55`–`:67`），`PreToolUse` 可 block（`:91`–`:101`）；顺序 PermissionRequest → Gate → PreToolUse（`:54`/`:69`/`:90`） | 扩展层**禁止**改 payload（`contributors/tool_lifecycle.rs:143`），改参数归 hooks（`contributors.rs:411`–`:414`"Called after pre-tool hooks finalize an invocation"） | **口径不同**，且 gaea 是"钩子先于闸门"，codex 是"hooks 先定稿、扩展后观测"（`contributors.rs:420`–`:423`） |
| 上下文装配 | 单函数 `buildSystemPrompt` 顺序拼 L1（身份 + 输出风格 + 语言策略 + 持久记忆 + 技能索引，`internal/gaea/boot/sysprompt.go:36`–`:51`） | 三槽位 `PromptSlot` + world state 章节 + turn-input user 片段（`contributors/prompt.rs:11`–`:16`、`core/src/session/mod.rs:4681`） | **缺**：gaea 无法让外部单元"往上下文里插一段"而不改 boot |
| 工具契约登记 | **有且比 codex 强**：`RegistryMap` 五字段契约（Source/Spaces/Effect/Evidence/Predicate，`internal/gaea/tool/registrymap.go:34`–`:42`），文档由单源生成 + 三重防漂移测试（`docs/gaea-tool-registry-map-2026-11.md:3`），未知工具直接红（`tool/registrymap_test.go:17`–`:27`） | 无对应物；扩展新增工具无需登记，只有 `record_collision`（`core/src/tools/spec_plan.rs:388`） | **gaea 领先，落地扩展框架时必须保住这条** |
| 隔离 | 无会话级隔离；靠 `permission.Gate` + 空间标签（`internal/gaea/boot/boot.go:346`–`:352`、`builtin.AllowsSpace` 于 `boot/plugins.go:49`） | `SessionIsolation` + 空 registry（`session_isolation.rs:11`、`core/src/thread_manager.rs:2321`） | **缺** |
| 装配顺序 | `boot.Build` 一个 57 KB 函数里，顺序即语义（`internal/gaea/boot/boot.go:622`–`:636` 示例） | 装配顺序 = `extensions.rs:77`–`:135` 的调用行序 | **口径相同**（都靠行序），但 gaea 无隔离层 |

**结论（正面回答"gaea 现在有没有扩展点"）**：**没有**。gaea 现在有两件半成品——
① `tool.Registry`（按名字注册，声明式，但注册点在 boot 里硬编码）；
② `agent.ToolHooks` 接口（可插拔，但语义是"外部 shell 钩子"，且挂在**工具**这一层）。
缺的是三样：**typed 槽位**（一个扩展能被多种角色调用）、**四层私有状态**、**thread/turn 级广播**。
最小可行切入点是 `ToolContributor` 等价物（把 `boot.go` 的 `reg.Add` 收敛成 contributor 注册），
最大阻力是 `boot.Build` 单函数内的装配顺序与 `RegistryMap` 守卫的交叉约束。

---

## 9. 施工卡（3 刀）

**刀 1 · `ExtRegistry`：把 boot 的 reg.Add 收敛成声明式注册**
- 内容：Go 侧建 `type Registry struct { tools []ToolContributor; threads []ThreadContributor; turns []TurnContributor; ... }`，
  每个槽用 **slice（不是 map）**保序；`Build()` 后不可变。`boot.Build` 里逐行 `addGated` 改为
  `reg.ToolContributor(builtinWorkspace{})`。
- 不动：`RegistryMap` 五字段契约与三重守卫（`tool/registrymap_test.go:17`、`boot/registrymap_census_test.go`、
  `app/registrymap_extra_test.go`）。新增 contributor 必须仍能反查出工具名来源。
- 验收：`TestRegistryMapCoversAllBuiltins` 仍绿；`reg.Names()` 与改造前逐字节一致（工具面哈希不变，
  前缀缓存不破，对齐 `boot/dynamic_tools.go:12`–`:13` 的"零 schema 增量"纪律）。

**刀 2 · 四层 Store（session / thread / turn / step）**
- 内容：`type Store struct { mu sync.RWMutex; m map[string]any }`，键用**显式字符串 + 版本**
  （Go 无 `TypeId`，见 §11 风险 4）；`get/getOrInit/insert/insertIf/remove` 五个方法对齐
  `ext/extension-api/src/state.rs:74`–`:132`。
- 落点：`session` = 现有 boot 产出的会话级依赖；`thread` = `controller` 每会话一份；
  `turn` = 现有 `TurnStarted` 生命周期内（`agent_run.go:59`）；`step` = 每个模型步一次
  （对齐 `core/src/session/mod.rs:4123` 的 `prepare_for_step` 语义）。
- 验收：技能索引/记忆块的注入改成"从 thread store 读"，删除 boot 里的直传参数至少 3 个。

**刀 3 · 工具面三分：新增 / 改参数 / 改结果 归属明确化**
- 内容：把现在的 `ToolHooks` 拆成三张显式契约：`ToolProvider`（新增工具，等价 `ToolContributor`）、
  `ArgRewriter`（改参数，等价 hooks `PermissionRequest`）、`ResultObserver`（改结果，等价
  `on_mcp_tool_result`，`contributors/tool_lifecycle.rs:194`）。
- 关键点：**保持 gaea 的顺序**（PermissionRequest → Gate → PreToolUse，`tool_dispatch.go:54`/`:69`/`:90`），
  不要抄 codex 的"hooks 先定稿再通知扩展"（`core/src/tools/lifecycle.rs:41`）——gaea 的顺序更安全。
- 验收：`ToolDispatcher.Check` 的单测仍绿；新增 `ResultObserver` 后 MCP 工具结果可被改写且有测试。

---

## 10. 验收断言

1. **装配点唯一**：生产代码里 `ExtensionRegistryBuilder` 的 `build()` 只有一处
   ——`app-server/src/extensions.rs:136`；`core/src` 生产文件只把它当**参数类型**用
   （`core/src/agent_message_board.rs:42`），从不自己 `new()`/`build()`（其余命中全在 `*_tests.rs`）。
2. **槽位数量守恒**：`ExtensionRegistry` 字段 16 个 = 14 条 contributor 向量 + `event_sink` + `turn_start_admission`
   （`ext/extension-api/src/registry.rs:161`–`:178`）；新增槽位必须同时改三处
   ——`Default`（`:26`–`:48`）、`to_builder`（`:182`–`:203`）、以及新 getter（`:220`–`:306` 任选位置）。
3. **无统一 Extension trait**：`pub trait *Contributor` 共 14 个
   （`contributors.rs:89/122/165/217/291/308/325/345/382/415/468/483` + `model_request.rs:33` +
   `message_lifecycle.rs:18`）；任何"给扩展一个总接口"的改造都会破坏"一个对象注册 7 次"的既有形态
   （`ext/skills/src/extension.rs:616`–`:623`）。
4. **失败不阻断**：所有生命周期回调返回 `()`，只有 3 个槽位返回可否定类型
   （`TurnItemContributor`→`Result`、`ApprovalReviewContributor`→`Option`、`TurnStartAdmission`→`Option`）。
   断言：若新增回调需要阻断，必须**新开返回类型**，不得复用 `()` 后再偷偷 panic。
5. **取消必须显式**：只有 `prepare_for_step`（`core/src/session/mod.rs:4137`）与 turn-input `contribute`
   （`core/src/session/turn.rs:1308`）被 `or_cancel` 包裹；其余 await **不可取消**。
6. **warning 上限恒定**：>256 B 的 message 必须按 char boundary 截断（`app-server/src/extensions.rs:268`–`:274`），
   且测试用 64 个 `🙂`（256 B）钉死该行为（`:393`–`:406`）。
7. **审批 `None` ≠ 放行**：`decide` 全 `None` → 用户流（`core/src/guardian/decision.rs:44`）；
   且取消优先于缓存决策（`:116`–`:121`）。
8. **`ToolPolicy` 启动冻结**：只能从 `ExtensionDataInit` 取（`core/src/session/session.rs:1053`），
   运行期扩展状态改不动它（`tool_policy.rs:6`–`:8`）；`allowed_tools: Some(vec![])` = 一个工具都不给
   （`tool_policy.rs:12`）。
9. **隔离是"少给"不是"沙箱"**：`Isolated` 会话用空 registry 或显式 registry
   （`core/src/thread_manager.rs:2321`–`:2325`），且不获得任何评审权限（`session_isolation.rs:1`–`:2`）。
10. **扩展层不碰 payload**：`ToolStartInput.payload` 为 `&`（`contributors/tool_lifecycle.rs:143`）；
    断言方式：给 `ToolLifecycleContributor` 加写参数能力会直接编译不过。

---

## 11. 不搬 / 只借语义 / 风险

**不搬**
- `<C: Sync>` 泛型贯穿全部 14 个 trait（`registry.rs:22`、`contributors.rs:89`、`:165`、`:308`）。
  Go 没有等价物，硬抄只会得到一堆 `any` + 类型断言。**借语义**：把"config 形状"做成注入的闭包/函数值
  ——codex 自己就这么干（`ext/skills/src/extension.rs:79` `config_from_host: Arc<dyn Fn(&C) -> SkillsExtensionConfig>`，
  `app-server/src/extensions.rs:125`–`:133` 传的是 `|config: &Config| SkillsExtensionConfig{..}`）。
- `guardian-v2` 的测试面（`ext/guardian-v2/src/async_scorer/extension_tests.rs` 142 680 B）——
  参考价值低、维护成本高。

**只借语义**
- **World State section 的"谁拥有什么"**：扩展拥有 `id` + 比较快照 + diff 渲染，harness 拥有持久化与信封
  （`contributors/world_state.rs:94`–`:104`）。gaea 侧对应的是折叠/章节机制（`internal/gaea/contextview/fold.go`，
  本文件未细读），可以照这个所有权划分加"可插拔章节"。
- **`prepare_for_step` 的幂等约定**："Never publish partial state"（`contributors.rs:383`）+
  "已存在就 return"（`ext/skills/src/extension.rs:266`–`:268`）——这是一条可以直接抄进 Go 注释的纪律。
- **审批的"审查只满足审查关"**（`approval_review.rs:1`）：gaea 的 Guardian 接线（`internal/gaea/control/controller_guardian.go`）
  应保持同样的分层，不要让它顺手放行权限。

**风险**
1. **panic 无兜底**：codex 全树无 `catch_unwind`（仅 `ext/extension-api/tests/state.rs:134`）。
   Go 侧必须给每个 contributor 调用点包 `defer recover()`，否则一个扩展崩掉整个回合。
2. **顺序即语义**：审批是"第一个 `Some` 胜出"（`registry.rs:260`–`:263`），上下文是 `Vec` 顺序拼接
   （`core/src/session/mod.rs:4601`）。Go 若用 `map` 存 contributor，行为将不可复现——**必须 slice**。
3. **全 inline await，一个慢扩展拖死回合**：只有 `on_tool_dispatch` 与 timing 观察者被明确要求
   "must return promptly"（`contributors.rs:417`、`tool_lifecycle.rs:244`），其余没有超时。
   gaea 落地时应默认加 per-callback 超时（对齐 `app-server/src/extensions.rs:292` 那种 10 s 兜底思路）。
4. **`TypeId` 键不可移植**：`state.rs:8`–`:19` 用 `TypeId::of::<T>()` 做键，类型不匹配直接 `unreachable!`（`:144`）。
   Go 侧用字符串键必须自己承担版本与冲突责任（参考 gaea 已有的 `RegistryMap` 显式契约思路）。
5. **`ToolPolicy` 只在启动期读**：`session.rs:1053` 之后不可变；若 gaea 想"运行中收紧工具面"，
   需要另开机制，不要假装改 `ToolPolicy` 就能生效。

---

## 12. 证据校验

实际跑过的命令（PowerShell，工作目录 `C:\AI\wubigrok`）：

```powershell
# 1. ext 目录全量清单（19 个 crate，含每个 .rs 的字节数）—— 确认扩展边界
$root='C:\AI\re\codex-0.165.0\oss\codex-0.165.0\codex-rs'
Get-ChildItem "$root\ext" -Recurse -File -Filter *.rs | ForEach-Object { "{0}`t{1}" -f $_.FullName.Replace($root,''), $_.Length }

# 2. 装配点定位（结论：生产 build() 仅 app-server/src/extensions.rs:136）
Get-ChildItem "$root\core\src","$root\app-server\src" -Recurse -Filter *.rs |
  Select-String -Pattern 'ExtensionRegistryBuilder'
# → core 侧生产命中仅 agent_message_board.rs:17(use) / :42(参数类型)，其余全 *_tests.rs

# 3. 每个槽位的消费点（本轮 15 条 patten 一次跑完）
foreach($pat in @('tool_contributors\(\)','context_contributors\(\)','decide_approval','admit_turn_start',
  'turn_input_contributors\(\)','turn_item_contributors\(\)','message_lifecycle_contributors\(\)',
  'model_request_contributors\(\)','token_usage_contributors\(\)','skill_invocation_contributors\(\)',
  'mcp_server_contributors\(\)','config_contributors\(\)','requires_host_skill_discovery')) { ... }

# 4. 生命周期回调的宿主调用点（校验 §3.3 / §4 每一行）
Get-ChildItem "$root\core\src" -Recurse -Filter *.rs | Where-Object { $_.Name -notmatch '_tests\.rs$' } |
  Select-String -Pattern 'on_turn_start|on_turn_stop|on_turn_abort|on_turn_error|on_thread_start|on_thread_ready|on_thread_resume|on_thread_idle|on_thread_stop|refresh_catalog_for_step|on_item_completed|on_tool_start|on_tool_finish|on_tool_dispatch|on_mcp_tool_result|on_config_changed|on_token_usage'

# 5. 隔离 / 白名单落点
Get-ChildItem "$root\core\src" -Recurse -Filter *.rs | Select-String -Pattern 'codex_extension_api::ToolPolicy|tool_policy'
Get-ChildItem "$root\core\src" -Recurse -Filter *.rs | Select-String -Pattern 'SessionIsolation::Isolated|IsolatedSessionExtensions'

# 6. panic 兜底核查（结论：core/ext 仅 state 测试命中）
Get-ChildItem "$root\core\src","$root\ext" -Recurse -Filter *.rs | Select-String -Pattern 'catch_unwind'

# 7. gaea 侧"有没有扩展点"（结论：0 命中）
Get-ChildItem 'C:\AI\wubigrok\internal\gaea' -Recurse -Filter *.go | Where-Object { $_.Name -notmatch '_test\.go$' } |
  Select-String -Pattern 'Contributor|ExtensionRegistry|type Extension interface|LifecycleContributor'
```

**已读**（约 70 个文件，逐行读或按行段读）：
`ext/extension-api` 全部 26 个 `.rs`（除 `examples/`）、`ext/skills/src/{extension.rs, host_service.rs(前 120 行)}`、
`ext/{web-search,image-generation,history-notes,multi-env-copy-file}/src/extension.rs`、
`ext/goal/src/{spec.rs 全文, extension.rs 结构索引}`、`ext/{mcp,queue,items,agent-message-board}/src/lib.rs`（及三处 install 行）；
`app-server/src/{extensions.rs, turn_admission.rs}` 全文；
`core/src/{tasks/lifecycle.rs, tools/lifecycle.rs}` 全文、
`core/src/{tools/spec_plan.rs, session/mod.rs, session/turn.rs, session/turn_input.rs, session/mcp.rs,
session/session.rs, session/world_state.rs, stream_events_utils.rs, skills.rs, mcp.rs, thread_manager.rs,
codex_delegate.rs, guardian/decision.rs, tools/parallel.rs}` 关键行段；
gaea：`internal/gaea/agent/{agent.go(行段), tool_dispatch.go, tool_sources.go}`、
`internal/gaea/boot/{boot.go(行段), sysprompt.go(前 60 行), plugins.go, dynamic_tools.go, skilluse.go}`、
`internal/gaea/control/controller.go(行段)`、`internal/gaea/tool/registrymap.go(行段)`、
`docs/gaea-tool-registry-map-2026-11.md(前 40 行)`。

**未读（明确标注，禁止当结论引用）**：
- `ext/goal/src/{accounting.rs(25 KB), runtime.rs(25 KB), tool.rs(19 KB), api.rs(14 KB)}` —— 只读了
  `spec.rs` 全文与 `extension.rs` 的 impl 索引，未读内部账本/运行逻辑；
- `ext/queue/src/service.rs`（21 KB）只读了 `lib.rs` 的 install 行；
- `ext/memories/*`、`ext/guardian-reviewer/*`、`ext/guardian-v2/*`（除 `async_scorer/extension.rs` 的
  install 段与 `sync_reviewer/mod.rs` 的 install 行）、`ext/guardian-v3`、`ext/connectors/*`、
  `ext/agent/*`、`ext/agent-message-board/src/extension.rs`（只读了 install 行）、
  `ext/mcp/src/{plugin.rs, plugin_contributor.rs, stream_manager.rs, cloud_plugin.rs}`；
- `ext/skills/src/{host_service.rs 120 行之后, render.rs, world_state.rs, state.rs, catalog.rs, tools/*}` ——
  只按常量与关键行取用；
- `ext/extension-api/examples/*`、`ext/extension-api/tests/{registry.rs,state.rs,capabilities.rs}`（仅引用一处
  `catch_unwind` 位置）；
- `core/src/guardian/*`（除 `decision.rs`）、`core/src/hooks` 侧（见 asset-05）、`core/src/tools/{router.rs, registry.rs}`、
  `app-server/src` 其余文件、`frontend/`、`sandboxing/`/`bwrap/`/`windows-sandbox-rs/`；
- gaea `internal/gaea/control/controller_approval.go`（33 KB，仅按函数名 grep）、
  `internal/gaea/boot/boot.go` 除引用行段之外的 57 KB 其余部分。

> ⚠ 行号时效：codex 侧引用基于 `C:\AI\re\codex-0.165.0\oss\codex-0.165.0\codex-rs`（只读，稳定）。
> gaea 侧引用基于**读取时刻**的行号；本次会话期间该仓库存在并发改动（例：`internal/gaea/control/controller_session.go`
> 的 mtime 与本会话重叠），若后续行号漂移，请按 `file:函数名` 重新定位，不要按行号硬对。
