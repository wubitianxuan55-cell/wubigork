# asset-54 · 观测三件：analytics / otel / rollout-trace（B10 · 施工级）

> 证据：`codex-rs/analytics/src/{lib.rs, events.rs, client.rs, reducer.rs, facts.rs, accepted_lines.rs}`、
> `otel/src/{metrics/client.rs, metrics/buffered.rs, metrics/config.rs, events/session_telemetry.rs, trace_context.rs}`、
> `rollout-trace/src/{protocol_event.rs, bundle.rs, raw_event.rs, inference.rs, thread.rs, tool_dispatch.rs}`（`rust-v0.165.0`）。
> gaea 侧只读对照：`internal/gaea/event/event.go`、`internal/gaea/contextview`（缓存 metrics）；其余未读（见 §12）。
> 仅出机制与施工卡，不含代码改动。

---

## 1. 一句话结论

codex 的观测是**三条互不耦合的管道**，各有明确预算：

1. **analytics（产品事实流）**：按"事实（fact）"建模，进 256 容量的队列，**10s 单次超时 / 25s flush 超时**，
   `reducer` 里对每一类事实都有**硬上限**（工具响应 256、语音窗口 16、插件维度 8、标识符 64 字节、待定回合 256）；
   另有 **4096 键去重表**防重复计。
2. **otel（运行指标）**：meter 名 `codex`，时长统一 `ms` 且**自定义直方图边界**；
   缓冲层 `MAX_PENDING_OPERATIONS=256 / MAX_METADATA_BYTES=1024 / MAX_TAGS=16`，
   并支持 `enable / disable / suspend` 三态（**同意（consent）撤回时挂起而非丢弃配置**）。
3. **rollout-trace（可回放轨迹）**：**不新增 hook**，把既有 protocol 事件**翻译**成更小的 trace 词汇；
   产物是 bundle 目录（`manifest.json` + `trace.jsonl` + `payloads/` + `state.json`，三者 schema 均为 v2），
   reducer 把原始事件归约成 `state.json`，可离线回放。

gaea 现在有**事件流**（`event/event.go`）与**局部 metrics**（contextview 缓存命中率一类），
但**没有遥测导出层、没有批处理/去重预算、没有可回放轨迹 bundle**——这三件是 B10 的真缺口。

---

## 2. 机制地图

| 关注点 | 文件 | 关键行 |
|---|---|---|
| analytics 队列与超时 | `analytics/src/client.rs` | 87 `QUEUE_SIZE=256`；88 `EVENTS_TIMEOUT=10s`；90 `FLUSH_TIMEOUT=25s`；91 `DEDUPE_MAX_KEYS=4096` |
| analytics 事实与容量 | `analytics/src/reducer.rs` | 182 `MAX_TOOL_RESPONSE_ENTRIES=256`；183 `MAX_RECENT_VOICE_SESSION_WINDOWS=16`；186 `MAX_PLUGIN_MEASUREMENT_DIMENSIONS=8`；187 `IDENTIFIER_BYTES=64`；209 `MAX_PENDING_TURNS=256` |
| 事件词汇 | `analytics/src/events.rs` / `facts.rs` | `TrackEventsRequest{events}`（批量）；事实类型：Compaction / Guardian / Hook / Plugin / Skill / Artifact / Turn / Steer / Goal / Image / Elicitation … |
| 序列化纪律 | `analytics/src/lib.rs` | `serialize_enum_as_string`、`usize_to_u64`、`option_i64_to_u64`（防跨语言数值不一致） |
| otel 指标 | `otel/src/metrics/client.rs` | 51 `METER_NAME="codex"`；52–54 时长单位 `ms` + 自定义边界；330 `new(config)`；420–472 `counter/histogram/gauge` |
| otel 缓冲 | `otel/src/metrics/buffered.rs` | 15 `MAX_PENDING_OPERATIONS=256`；16 `MAX_METADATA_BYTES=1024`；17 `MAX_TAGS=16`；112/142/149 `enable/disable/suspend` |
| 会话遥测 | `otel/src/events/session_telemetry.rs` | 会话级事件聚合（1.3k 行） |
| trace bundle | `rollout-trace/src/bundle.rs` | 8 `manifest.json`；9 `trace.jsonl`；10 `payloads/`；12 `state.json`；13/14 `SCHEMA_VERSION=2` |
| 事件翻译 | `rollout-trace/src/protocol_event.rs` | 1–12 设计说明（**复用 protocol 事件、穷举 match 作为编译期提示**）；40 `codex_turn_trace_event` |
| 推理调用追踪 | `rollout-trace/src/inference.rs` | 28 `x-codex-inference-call-id`；注释：禁用时"记录空操作"、不改变 HTTP/WS/重试同一条代码路径 |
| 预览上限 | `rollout-trace/src/tool_dispatch.rs` | 337 `MAX_PREVIEW_CHARS=160` |
| trace 根目录 | `rollout-trace/src/thread.rs` | 44 `CODEX_ROLLOUT_TRACE_ROOT` |

---

## 3. analytics：事实流 + reducer 归约

### 3.1 结构

```
事实(facts.rs 里的强类型)  ──► events.rs 组装成 TrackEventRequest
        │                            │
        └──► reducer.rs（会话内归约，带硬上限）──► client.rs（队列 256 / 10s / flush 25s / 去重 4096）
```

- **事实而不是日志**：每类事实都是具名结构（如 `TurnTokenUsageFact`、`CompactionStrategy`、`GuardianReviewDecision`），
  枚举序列化成字符串，数值统一成 `u64`——**跨语言仪表盘不依赖 Rust 的表示**。
- **批量协议**：`TrackEventsRequest { events: Vec<TrackEventRequest> }`，`TrackEventRequest` 是
  `#[serde(untagged)]` 的联合——服务端按形状判别，客户端不必带 type 标签。
- **reducer 的意义**：把"每回合都会刷的事件"归约成"每会话一份状态"，避免无限增长的数组；
  每类都有一个上限（§2 表），**超限即丢最旧**，这是"观测不能把内存吃穿"的硬保证。
- **去重**：`DEDUPE_MAX_KEYS = 4096`——键表有上限，说明去重是"尽力而为"，不是精确集合。

### 3.2 三个"为什么不那样做"

- 不用通用 `log!` 打点：因为仪表盘需要**稳定字段名**，日志字段随时会被人改。
- 不在 reducer 里存原始事件：只存归约后的计数与最后状态（否则和 trace 重复）。
- 去重表不设无限：观测系统自身的**内存上界必须可证**。

---

## 4. otel：指标与同意门

- 单一 meter `codex`；时长统一 `ms`，并为不同量级准备**两组直方图边界**（毫秒级与秒级），
  避免"一个边界套所有指标"导致 p99 失真。
- 缓冲层三态：`enable`（接上 client）/`disable`（清空并停）/`suspend`（保留已收数据但不导出）——
  **用户撤回遥测同意时用 suspend，而不是 disable**，这样重新同意后不用重建。
- `counter_with_analytics_consent(...)`：计数本身要**带同意语义**，说明"是否上报"是逐指标的决策，不是全局开关。
- 导出间隔可配（`MetricsConfig.export_interval`），shutdown 带超时（`shutdown_with_timeout`）。

---

## 5. rollout-trace：可回放轨迹

### 5.1 bundle 布局与版本

```
<root>/manifest.json      # 清单（schema v2）
<root>/trace.jsonl        # 原始事件流（schema v2，一行一事件）
<root>/payloads/          # 大负载单独落盘（按内容寻址）
<root>/state.json         # reducer 归约后的状态（schema v2）
```

- **三个 schema 版本号都独立**（manifest / raw event / reduced state 各 v2）——允许"原始格式不变、归约格式升级"。
- 根目录由 `CODEX_ROLLOUT_TRACE_ROOT` 指定，未设则功能关闭。

### 5.2 设计原则（注释明说）

1. **不新增 hook**：session 层已经在发 protocol 事件（回合生命周期、终端会话、patch、MCP、协作工具），
   trace 只是**翻译**它们——避免"两套观测点漂移"。
2. **穷举 match 当编译期提示**：新增 `EventMsg` 变体时，编译器逼你决定"这个事件要不要进 trace"。
3. **禁用时零成本**：`inference.rs` 注释写明"禁用的上下文什么都不记"，使一次性 HTTP、WS 复用、重试/回退
   **共用同一条代码路径**——观测不能改变被测行为。
4. **预览有上限**：工具派发预览 `MAX_PREVIEW_CHARS = 160`，大内容进 `payloads/`。

---

## 6. 常量与上限

| 名称 | 值 | 位置 | 生产/测试 | 语义 |
|---|---|---|---|---|
| `ANALYTICS_EVENTS_QUEUE_SIZE` | 256 | `analytics/client.rs:87` | 生产 | 待发送事件队列 |
| `ANALYTICS_EVENTS_TIMEOUT` | 10s | `:88` | 生产 | 单次上报超时 |
| `ANALYTICS_EVENTS_FLUSH_TIMEOUT` | 25s | `:90` | 生产 | 关闭时 flush 上限 |
| `ANALYTICS_EVENT_DEDUPE_MAX_KEYS` | 4096 | `:91` | 生产 | 去重键表上限 |
| 工具响应归约上限 | 256 | `reducer.rs:182` | 生产 | 防无限增长 |
| 语音窗口保留 | 16 | `reducer.rs:183` | 生产 | 只留最近 16 个 |
| 插件维度 / 标识符 | 8 / 64B | `reducer.rs:186–187` | 生产 | 维度与字符串上界 |
| 待定回合 | 256 | `reducer.rs:209` | 生产 | 未完成回合暂存 |
| `MAX_PENDING_OPERATIONS` | 256 | `otel/metrics/buffered.rs:15` | 生产 | 指标缓冲 |
| `MAX_METADATA_BYTES` | 1024 | `:16` | 生产 | 单条元数据 |
| `MAX_TAGS` | 16 | `:17` | 生产 | 单指标标签数 |
| `METER_NAME` | `codex` | `otel/metrics/client.rs:51` | 生产 | 指标命名空间 |
| trace schema | 2 / 2 / 2 | `rollout-trace/bundle.rs:13–14`、`raw_event.rs:26` | 生产 | manifest / state / raw |
| 工具预览 | 160 字符 | `rollout-trace/tool_dispatch.rs:337` | 生产 | 内联预览上限 |

---

## 7. 状态机 / 时序

**analytics 一条事件**：`fact → event 组装 → 入队（满则丢最旧/丢新，见 client 策略） → 批量发送（10s 超时）
→ 成功则清键；失败按 25s 内重试；关闭时 flush ≤25s`。

**otel 指标**：`record → 缓冲（≤256 条、标签≤16、元数据≤1KiB） → enable 时接 client → 按 export_interval 导出；
consent 撤回 → suspend（保留不导出）；进程退出 → shutdown_with_timeout`。

**rollout-trace**：`protocol 事件 → protocol_event 翻译（穷举 match） → raw_event 落 trace.jsonl
→ reducer 归约 → state.json`；payload 大对象进 `payloads/`；`x-codex-inference-call-id` 把一次推理调用与
其重试/回退串成同一条记录。

---

## 8. 失败路径

1. **analytics 上报失败**：不阻断业务（事件是旁路）；队列满时按容量策略丢弃，**丢弃本身不计入事实流**（避免自激）。
2. **otel 未 consent**：`suspend` 保留已收数据但不导出；`counter_with_analytics_consent` 在无同意时只做本地汇总。
3. **trace 目录不可写**：初始化失败即视为"trace 关闭"（`CODEX_ROLLOUT_TRACE_ROOT` 未设也关闭），不影响回合。
4. **reducer 超限**：按类上限丢最旧/丢新增（各文件内自带策略），保证状态文件大小有界。
5. **schema 版本不匹配**：读 `state.json` 时按版本判定，不支持则不还原（不猜）。

---

## 9. gaea 对照（只读实测）

| 维度 | codex | gaea | 差在哪 |
|---|---|---|---|
| 事件模型 | 强类型 fact + 归约 + 稳定字段名 | `internal/gaea/event/event.go`（23KB，Kind/Level/Text 等）+ `slog` 日志 | gaea 的事件是**面向 UI 的**，不是可上报的事实流；无"事实"与"展示"分层 |
| 上报管线 | 队列 256 / 10s / flush 25s / 去重 4096 | 未读到（无 analytics 模块） | 缺批处理与内存上界纪律 |
| 指标 | otel meter `codex`，ms 边界，consent 三态 | `contextview` 里有局部 metrics（`Metrics()/RecordCompact...`，`manager.go:40/82/103`） | gaea 的指标是**模块内自用**，无统一导出与同意语义 |
| 轨迹 | bundle（manifest/trace.jsonl/payloads/state.json，schema v2）可离线回放 | 有 `internal/gaea/trajectory` 包（未读） | 需核对 gaea 的 trajectory 是否可回放、是否有版本号与 payload 分离 |
| 观测不改变行为 | trace 禁用时零成本、同一条代码路径 | 未读到显式纪律 | 建议补一条"观测旁路不得改变主路径"的约束 |

> 结论：**真正值得搬三件**——① 事实流（强类型 + 归约 + 上界）；② 指标导出的同意三态与"逐指标同意"；
> ③ 轨迹 bundle 的"四件套 + 三版本号"结构。其余（具体事件名、字段）不要照搬。

---

## 10. 施工卡（3 刀；默认关闭）

### 刀 1 · 事实流与归约（约 1 天）

- 新增 `internal/gaea/telemetry/facts.go`（强类型 fact，字段名即契约）+ `reducer.go`（每类上限，参照 §6 表）。
- 队列 256 / 单次 10s / flush 25s / 去重 4096（键表有上限，允许误重）。
- 禁区：不改 `event.Event`（UI 事件流另有契约）；上报失败不得影响回合。

### 刀 2 · 指标与同意（约半天）

- `telemetry/metrics.go`：`enable/disable/suspend` 三态；标签上限 16；单条元数据 1KiB；导出间隔可配；
  退出带超时。同意撤回 → `suspend`（不丢配置、不导出）。

### 刀 3 · 轨迹 bundle（约 1 天，可选）

- `trajectory/` 输出四件套：`manifest.json` + `trace.jsonl` + `payloads/` + `state.json`，
  三者各带 schema 版本；环境变量控制根目录；未设即关闭；禁用时零成本。
- 大负载（工具输出、图片）进 `payloads/`，内联预览 ≤160 字符。

---

## 11. 验收断言

| # | 断言 |
|---|---|
| O1 | 事实流队列满 1000 条时，内存不增长（上限 256 生效），且丢弃不产生新事实 |
| O2 | 上报挂起 30s：flush 在 25s 内返回（不无限阻塞关闭） |
| O3 | 去重：同一键重复 1 万次只计一次；键表第 4097 个键替换最旧者 |
| O4 | reducer 超限：工具响应 > 256 时只保留最后 256 条，`state.json` 大小有界 |
| O5 | consent 撤回 → `suspend`：不再导出，但 `enable` 后无需重建配置即可恢复 |
| O6 | trace 根目录不可写 → 功能降级为关闭，回合正常完成（零影响） |
| O7 | trace 关闭时，推理调用路径与开启时**逐字节等价**（同一条代码路径，零分支代价需在文档标注为"逻辑等价"） |
| O8 | `state.json` schema 版本不支持 → 不还原并给出明确日志（不猜、不半还原） |

---

## 12. 不搬 / 风险 / 未读

- **不搬**：具体 fact 名称与字段（属于上游产品口径）；`session_telemetry.rs` 的 1.3k 行聚合细节；
  `reducer/tool/**` 的工具专属归约。
- **风险**：① 事实流与 UI 事件流容易混成一套（必须两套契约分开，否则 UI 改字段会打断报表）；
  ② 同意三态与"本地调试"冲突（调试时要能强制开启但不上报）；③ trace 可能写进用户工作区（必须默认落在
  `CODEX_HOME` 一类专属目录，且默认关闭）。
- **未读（诚实边界）**：`analytics/src/{facts.rs, accepted_lines.rs, product_attribution.rs, guardian_v2.rs}` 全文、
  `otel/src/{events/session_telemetry.rs, provider.rs, trace_context.rs, otlp.rs}` 全文、
  `rollout-trace/src/reducer/**` 与 `thread.rs`/`writer` 细节；gaea 侧 `internal/gaea/trajectory`、
  `internal/gaea/event`、`internal/gaea/contextview` 仅粗读。
