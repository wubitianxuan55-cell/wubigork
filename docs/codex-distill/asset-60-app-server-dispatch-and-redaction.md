# asset-60 · app-server 请求派发、线程订阅与响应脱敏（B1 补充 · 施工级）

> 证据：`codex-rs/app-server/src/{message_processor.rs, request_processors.rs, request_serialization.rs(见 asset-47),
> outgoing_message.rs, bespoke_event_handling.rs, error_code.rs}`、
> `app-server/src/request_processors/{thread_processor.rs, thread_resume_redaction.rs, turn_processor.rs,
> thread_lifecycle.rs, config_processor.rs, mcp_processor.rs, plugins.rs, catalog_processor.rs,
> process_exec_processor.rs, command_exec_processor.rs, thread_goal_processor.rs, thread_queue_processor.rs,
> apps_processor.rs, account_processor.rs, feedback_processor.rs, windows_sandbox_processor.rs, projects.rs,
> initialize_processor.rs}`、`app-server/src/{thread_sample.rs, thread_status.rs(见 asset-47), config_manager_service.rs,
> skills_watcher.rs, mcp_refresh.rs, turn_cost_worker.rs}`（`rust-v0.165.0`）。
> `asset-47` 已覆盖"准入 / 关闸 / 连接闸门 / 状态广播"；本篇补**派发结构、订阅语义与脱敏**。

---

## 1. 一句话结论

app-server 的请求路径是**三段式**：`process_request`（连接级前置）→ `dispatch_initialized_client_request`
（分类 + 准入 + 串行化入队）→ **按域拆分**的 processor（thread / turn / config / mcp / plugins / catalog /
account / feedback / apps / command-exec / process-exec …）。
本篇记录三条最容易被忽略、但 gaea 直接可用的机制：
① **线程订阅/解订阅是三态**（`NotLoaded` / `Unsubscribed` / 已订阅），并绑定连接；
② **恢复时按客户端名做响应级脱敏**——只影响响应体，不碰落盘历史与模型历史；
③ **限流与上限是"接口级常量"**（`thread/sample` 提示词 4 KB、输出 64 KiB；扩展告警截断 256 B）。

---

## 2. 机制地图

| 关注点 | 文件 | 关键行 |
|---|---|---|
| 连接级入口 | `message_processor.rs` | 644 `process_request`；983 `dispatch_initialized_client_request`；1088 `handle_initialized_client_request` |
| 分类与准入 | `message_processor.rs` | 1006–1026 两类准入（runtime change / turn work，见 asset-47）；1042–1051 串行化入队 + 二次准入 |
| 域处理器 | `request_processors/*.rs` | thread(281 KB 源文件) / turn(70 KB) / account(65 KB) / config(55 KB) / thread_lifecycle(44 KB) / feedback(38 KB) / mcp(33 KB) / catalog(29 KB) / process_exec(25 KB) / goal(22 KB) / windows_sandbox(19 KB) / apps(17 KB) / projects(16 KB) / queue(14 KB) / command_exec(13 KB) / attachments(12 KB) / initialize(11 KB) |
| 线程订阅 | `request_processors/thread_processor.rs` | 564 `thread_unsubscribe`；1079–1100 `thread_unsubscribe_response_inner`（三态判定） |
| 恢复脱敏 | `request_processors/thread_resume_redaction.rs` | 11 `REDACTED_PAYLOAD="[redacted]"`；12–13 远程客户端白名单（android/ios remote）；16–18 `should_redact_…`；20–40 `redact_…`（MCP 参数/结果/错误替换，`ImageGeneration` 项**直接删除**） |
| 样本接口上限 | `thread_sample.rs` | 57 `MAX_PROMPT_BYTES=4_000`；58 `MAX_OUTPUT_BYTES=64*1024`；83–86 超限报错文案 |
| 扩展告警截断 | `extensions.rs` | 176 `MAX_EXTENSION_WARNING_BYTES=256`；268–274 截断 |
| 错误码 | `error_code.rs` | 8 `INPUT_TOO_LARGE_ERROR_CODE="input_too_large"`；另有 `-32001` 过载（`OVERLOADED_ERROR_CODE`） |
| 输入过大遥测 | `request_processors/turn_processor.rs` | 510 `"input_error_code": INPUT_TOO_LARGE_ERROR_CODE` |

---

## 3. 三段式派发

```
① process_request(connection_id, request, transport, session)
      ├─ 连接是否已 initialize / experimental API 是否允许（999 行起的判定）
      ├─ 消息大小与形状校验（超限 → -32602 invalid_params 或 input_too_large）
      └─ 交给 dispatch
② dispatch_initialized_client_request(request)
      ├─ 准入分类（runtime change 一次准入 / turn work 准入 + 队列内二次准入）
      ├─ 事件流（McpServerEventStreamStart 单独起流）
      └─ serialization_scope → 入队（per-resource FIFO，见 asset-47 §3.4）
③ 具体 processor
      thread_processor / turn_processor / config_processor / mcp_processor / …
```

**为什么分三段**：① 前置校验与"是否已 initialize"属于连接，不该散在每个处理器里；
② 准入与串行化属于"调度"，不该散在每个 domain；③ domain 处理器只处理自己的状态机。
三段一分，`thread_processor.rs` 才有资格长到 281 KB 而不腐化。

---

## 4. 线程订阅与解订阅（三态）

```rust
thread_unsubscribe(thread_id, connection_id) -> ThreadUnsubscribeStatus
```
- 线程**未加载** → `NotLoaded`（不报错，报状态）。
- 已加载但该连接没订阅 → **幂等**（不报错）。
- 订阅了 → 真正解订阅 → `Unsubscribed`。

配套的是 `ThreadWatchManager`（asset-47 §3.5）：订阅者拿到 `watch::Receiver<ThreadStatus>`，
无人订阅时 watcher 被回收（`receiver_count()==0`）。

**语义要点**：解订阅返回的是"发生了什么"，而不是"成功了"——客户端据此更新本地 UI，
而不是把 `NotLoaded` 当错误重试。

---

## 5. 恢复脱敏：只脱响应，不脱历史

```rust
const REDACTED_PAYLOAD: &str = "[redacted]";
const CHATGPT_REMOTE_CLIENT_NAMES: &[&str] = &["codex_chatgpt_android_remote", "codex_chatgpt_ios_remote"];
```
- **唯一触发源是客户端名白名单**（不是"远程传输"、也不是"是否 TLS"）——因为问题不是传输不安全，
  而是"移动端客户端承载不了大 payload"。
- 只改 `thread/resume` 的**响应**：落盘 rollout、模型侧恢复历史、其它 API 一律不变（注释明确标注
  "Temporary bandaid for remote clients … Keep this response-only"）。
- 处理粒度：MCP 调用的 `arguments` → `"[redacted]"`；`result` → 一个只含占位文本的结果；
  `error.message` → `"[redacted]"`；**`ImageGeneration` 项整个丢弃**（连元数据都不留）。

**为什么值得学**：脱敏最怕"顺手把存储也改了"。这里用"响应级 + 客户端名白名单 + 明确注释"三层把它钉死。

---

## 6. 上限与错误码（接口级）

| 接口/位置 | 上限 | 超限行为 |
|---|---|---|
| `thread/sample` 提示词 | 4,000 B | 参数错误，文案说明"prompt of at most 4000 bytes" |
| `thread/sample` 输出 | 64 KiB | 截断累积（398 行起按 `saturating_add` 判定） |
| 扩展告警文本 | 256 B | 截断并加省略 |
| 入站请求 | 见 `asset-56`（分档 8 KiB / 流式 2×） | 拒收 |
| 过载 | `-32001` | 明确"过载"而非"内部错误" |

`input_too_large` 作为**字符串错误码**（不是数字）单独存在，并被打进回合遥测——便于统计"用户输入超限"这一具体失败。

---

## 7. 失败路径

1. **未 initialize 就发请求**：`invalid_request("Not initialized")`（990–992）。
2. **实验性 API 未开**：`experimental_required_message(reason)`，明确告知需要哪个开关。
3. **输入超限**：按接口返回 `invalid_params` 或 `input_too_large`；遥测记录错误码。
4. **解订阅未加载线程**：返回 `NotLoaded` 而非错误。
5. **恢复响应脱敏失败**：不会失败——脱敏是纯内存替换；但若客户端名不在白名单，**不脱敏**（fail-open 是刻意的：
   脱敏是"为弱客户端减负"，不是安全边界）。

---

## 8. gaea 对照（只读实测）

| 维度 | codex | gaea | 结论 |
|---|---|---|---|
| 派发结构 | 三段式 + 按域拆 processor | `internal/app/*.go` 的 bindings + `internal/gaea/control` 单控制器 | gaea 的"域"更少但耦合更紧；可借"前置校验/调度/域状态机"三分 |
| 订阅语义 | 三态返回 + 连接绑定 + watcher 回收 | 无订阅概念（前端单窗口直接读 store） | 若做多窗口/多前端，需要这套；否则不搬 |
| 响应级脱敏 | 按客户端名白名单，仅响应 | 未读到（绑定层直接回全量） | **可借**：给"移动/远程前端"做响应裁剪，别动存储 |
| 接口级上限 | sample 4KB/64KiB、告警 256B、入站分档 | 未读到统一上限 | 可借"每个对外接口都有上限常量" |
| 错误码分类 | `input_too_large` 字符串码 + `-32001` 过载 | 未读到 | 可借：把"用户输入超限"与"服务过载"分开 |

---

## 9. 施工卡（2 刀；默认关闭）

### 刀 1 · 响应级脱敏（约 2 小时，低成本高收益）

- 在 binding 出口加 `redactForClient(clientKind, payload)`：白名单客户端（如移动/远程）对
  大 payload（工具参数/结果/图片）替换为占位，**不改变落库与模型历史**。
- 禁区：不动 `chat` 包与 SQLite 落盘；占位文本用自己文案；白名单放配置里，默认空（即全量返回）。

### 刀 2 · 接口级上限常量表（约 2 小时）

- 给每个对外绑定/HTTP 接口定义 `maxRequestBytes` / `maxResponseBytes`，集中在
  `internal/app/limits.go`；超限返回明确错误码（`input_too_large` 语义）。
- 断言：超限请求被拒且不进入业务逻辑；错误码可被前端区分。

---

## 10. 验收断言

| # | 断言 |
|---|---|
| A1 | 未 initialize 的连接发请求 → `Not initialized`，不产生副作用 |
| A2 | `thread/sample` 提示词 4001 B → 明确参数错误，文案含上限值 |
| A3 | 脱敏：白名单客户端拿到的 MCP 参数/结果为 `[redacted]`，图片项消失；**落库与模型历史不受影响** |
| A4 | 非白名单客户端：响应逐字节不变（脱敏不影响主路径） |
| A5 | 解订阅未加载线程 → 返回 `NotLoaded`，不报错、不重试 |
| A6 | 过载返回 `-32001` 而非 `-32603`（可与"内部错误"区分） |

---

## 11. 不搬 / 未读

- **不搬**：`account_processor`(65 KB)、`feedback_processor`(38 KB)、`plugins`(103 KB)、`catalog_processor`、
  `apps_processor`、`windows_sandbox_processor` —— 属云/插件/平台域。
- **不搬**：`bespoke_event_handling.rs`(3.9k) 的"特例事件修补"（上游历史包袱，不值得学）。
- **未读（诚实边界）**：`thread_processor.rs`(281 KB) 只读了 API 列表与订阅/解订阅段；
  `turn_processor.rs`(70 KB)、`config_processor.rs`(55 KB)、`thread_lifecycle.rs`(44 KB)、
  `mcp_processor.rs`(33 KB) 只读入口与常量；`config_manager_service.rs`、`skills_watcher.rs`、
  `mcp_refresh.rs`、`turn_cost_worker.rs` 已在 asset-47 §4 速记过但未逐行。
