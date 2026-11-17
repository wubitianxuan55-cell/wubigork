# asset-58 · MCP 双端：客户端传输矩阵与服务端工具目录（B8 · 施工级）

> 证据：`codex-rs/rmcp-client/src/{rmcp_client.rs, stdio_server_launcher.rs, http_client_adapter.rs,
> in_process_transport.rs, ordered_byte_transport.rs, executor_process_transport.rs, event_notification_transport.rs,
> oauth.rs, perform_oauth_login.rs, ema_resource.rs, ema_auth.rs, elicitation_client_service.rs, auth_status.rs}`、
> `codex-mcp/src/{rmcp_client.rs, connection_manager.rs, connection_manager/tool_catalog.rs, tool_catalog_cache.rs,
> catalog.rs, runtime.rs, server.rs, binding.rs, elicitation.rs, auth_elicitation.rs}`（`rust-v0.165.0`）。
> `asset-42-integrations-elicitation-and-hooks.md` 只覆盖"集成面 UI 与注入字段"，
> `asset-23-hook-output-contract.md` 覆盖 hooks 契约；本篇写 **MCP 的传输、鉴权、工具目录与双向请求**。
> gaea 侧只读对照：`internal/gaea/control/controller_mcp.go`、`internal/gaea/agent/*`（未逐文件读，见 §11）。

---

## 1. 一句话结论

MCP 在 codex 里是**两条独立管线**：
① **客户端**（连别人的 MCP server）：四种传输（stdio 子进程 / streamable HTTP / 进程内 / 执行器进程），
每种都配一条**有界**通道（事件通知队列 2 MiB、stdio 单行 8 MiB），OAuth 与 EMA 各有**明确的服务器数与跳数上限**；
② **服务端**（把自身工具暴露成 MCP）：连接管理器 + **工具目录缓存（32 条 / 30 分钟）** + 每次取目录都要
"身份指纹"作键——避免"同名 server 不同配置共用缓存"的串味。
另外 elicitation（反向追问）被设计成**协议方法族**（`elicitation/create`、`openai/form`、`openai/elicitation/create`），
并给"提前取消"留了 1024 条上限。

---

## 2. 机制地图

| 关注点 | 文件 | 关键行 |
|---|---|---|
| 客户端入口 | `rmcp-client/src/rmcp_client.rs`(1698) | 23–102 传输与适配器导入：`StreamableHttpClientTransport` / `StreamableHttpClientAdapter` + `StreamableHttpRedirectMode` / `InProcessTransportFactory` / `OrderedByteTransport` / `StdioServerLauncher` |
| stdio 启动器 | `stdio_server_launcher.rs`(788) | 子进程拉起 + 单行上限见 §6 |
| 事件通知通道 | `event_notification_transport.rs` | 23 `MAX_QUEUED_EVENT_NOTIFICATION_BYTES = 2 MiB` |
| 执行器进程传输 | `executor_process_transport.rs` | 51 `MAX_MCP_STDOUT_LINE_BYTES = 8 MiB` |
| 发现超时 | `auth_status.rs` | 25 `DISCOVERY_TIMEOUT = 5s` |
| OAuth | `oauth.rs`(1697)、`perform_oauth_login.rs`(1494)、`oauth_http_client.rs`(476) | 登录流程 + 专用 HTTP 客户端 |
| EMA（企业托管鉴权） | `ema_resource.rs` | 53 `MAX_EMA_AUTHORIZATION_SERVERS = 16`；54 `MAX_EMA_DISCOVERY_REDIRECTS = 10`；55 `EMA_DISCOVERY_TIMEOUT = 30s` |
| EMA 授权类型 | `ema_auth.rs` | 35 `ID_JAG_GRANT_PROFILE = "urn:ietf:params:oauth:grant-profile:id-jag"` |
| Elicitation 客户端 | `elicitation_client_service.rs` | 38 `progressToken`；39–41 三个方法名；72 `MAX_EARLY_CANCELLATIONS = 1024` |
| 服务端连接管理 | `codex-mcp/src/connection_manager.rs`(1254) + `tool_catalog.rs`(771) | 连接/工具目录装配 |
| 目录缓存 | `tool_catalog_cache.rs` | 34 `TOOL_CATALOG_CACHE_CAPACITY = 32`；35 `TOOL_CATALOG_CACHE_TTL = 30 min`；272 `ToolCatalogIdentity` |
| 服务端 | `codex-mcp/src/{server.rs, binding.rs, runtime.rs}` | 把 codex 工具暴露成 MCP |
| Elicitation 服务端 | `codex-mcp/src/{elicitation.rs, auth_elicitation.rs}` | 反向追问的落地 |

---

## 3. 客户端：传输矩阵与选择

| 传输 | 实现 | 何时用 | 关键约束 |
|---|---|---|---|
| stdio 子进程 | `StdioServerLauncher` | 本地 MCP server（最常见） | 单行 **8 MiB**；子进程随会话生命周期 |
| streamable HTTP | `StreamableHttpClientTransport` + `StreamableHttpClientAdapter`（含 `StreamableHttpRedirectMode`） | 远程 MCP | 重定向模式显式可选；OAuth 走专用 adapter |
| 进程内 | `InProcessTransportFactory` | 同进程内嵌 server | 不跨进程，省去序列化成本 |
| 执行器进程 | `executor_process_transport.rs` | 经执行服务转发 | stdout 单行 **8 MiB** |
| 有序字节流 | `OrderedByteTransport` | 需要"字节有序"的底层传输 | 保证顺序，不做额外缓冲 |

要点：**传输是可组合的**（适配器 + 有序字节层 + OAuth 层），而不是"一个大 enum 里 if/else"。
这让"加一种传输"不改既有分支。

---

## 4. 鉴权：OAuth 与企业 EMA 的上限

- 普通 OAuth：`oauth.rs` + `perform_oauth_login.rs`，带独立 HTTP 客户端（`OAuthHttpClientAdapter`），
  发现阶段 **5s** 超时（`auth_status.rs:25`）。
- 企业 EMA：授权服务器**最多 16 个**、发现重定向**最多 10 跳**、发现超时 **30s**；
  授权类型用 `id-jag`（RFC 草稿形态的 token exchange profile）。
- 三个数字合起来的意思是：**"发现"这一步必须有界**——否则企业环境里一个错误配置的授权服务器就能把登录拖死。

---

## 5. Elicitation（反向追问）：当成协议族来做

```rust
const MCP_ELICITATION_CREATE_METHOD: &str = "elicitation/create";
const OPENAI_FORM_METHOD: &str = "openai/form";
const OPENAI_ELICITATION_METHOD: &str = "openai/elicitation/create";
const MAX_EARLY_CANCELLATIONS: usize = 1024;
```
- 三个方法名 = "标准方法 + 两个厂商扩展"，客户端**同时支持**，按 server 能力选择。
- `progressToken` 作为 meta key：追问与进度条共用同一标识。
- `MAX_EARLY_CANCELLATIONS = 1024`：用户可能在请求真正建立前就取消——**提前取消要留痕但不能无界堆积**。

---

## 6. 服务端：工具目录与缓存身份

- 目录缓存：**32 条 / 30 分钟**（`TOOL_CATALOG_CACHE_CAPACITY / TTL`）。
- 缓存键是 `ToolCatalogIdentity`（`tool_catalog_cache.rs:272`）——把 server 身份（名称 + 配置 + 传输）
  打进键里，避免"同名不同配置"的 server 共用同一份工具目录。
- 另有 `OptionalStartupDeadline`（`:66`）：启动期取目录**可以带 deadline**，超时就让会话先跑起来、
  目录稍后补齐——这是"MCP 慢不能拖死启动"的实现。

---

## 7. 常量与上限

| 名称 | 值 | 位置 | 生产/测试 | 语义 |
|---|---|---|---|---|
| 事件通知队列 | 2 MiB | `rmcp-client/event_notification_transport.rs:23` | 生产 | 通知积压上限 |
| stdio/执行器单行 | 8 MiB | `executor_process_transport.rs:51` | 生产 | 单行消息上限 |
| 发现超时 | 5s | `auth_status.rs:25` | 生产 | 普通鉴权发现 |
| EMA 授权服务器数 | 16 | `ema_resource.rs:53` | 生产 | 企业多 IdP 上限 |
| EMA 发现跳数 | 10 | `ema_resource.rs:54` | 生产 | 重定向上限 |
| EMA 发现超时 | 30s | `ema_resource.rs:55` | 生产 | 企业发现总时限 |
| 提前取消 | 1024 | `elicitation_client_service.rs:72` | 生产 | 未建立请求的取消留痕上限 |
| 目录缓存容量 / TTL | 32 / 30min | `tool_catalog_cache.rs:34–35` | 生产 | 服务端工具目录 |

---

## 8. 失败路径

1. **stdio server 起不来**：启动器返回错误 → 该 server 标记不可用，**不阻塞其它 server**（多 server 并行可用性）。
2. **stdout 单行超限（8 MiB）**：视为协议错误并断开该 server，避免 OOM。
3. **通知积压超 2 MiB**：丢弃最旧/断开（按实现），保证内存上界。
4. **OAuth 发现超时（5s / EMA 30s）**：登录失败并给可读原因，不无限重试。
5. **目录拉取超 deadline**：会话先启动，目录后台补齐（`OptionalStartupDeadline`）。
6. **elicitation 提前取消**：记入取消表（≤1024），超限丢最旧。

---

## 9. gaea 对照

| 维度 | codex | gaea | 结论 |
|---|---|---|---|
| 传输 | 4 种 + 适配器组合 | `controller_mcp.go`(4275) 一处实现（未逐行读） | 可借"适配器组合"而非"大 enum" |
| 鉴权上限 | 发现 5s / EMA 16 服务器 / 10 跳 / 30s | 未读到 | 可借"发现必须三有界"（时长/跳数/数量） |
| 工具目录 | 32 / 30min + 身份指纹键 | 未读到目录缓存；工具列表可能每回合重取 | 可借：目录缓存必须带**身份**键，否则串味 |
| 启动期目录 | 可带 deadline，慢则后补 | 未读到 | 可借：MCP 慢不得阻塞会话启动 |
| 反向追问 | 三方法族 + progressToken + 提前取消上限 | `asset-42` 记了 gaea 的 `__approval` 注入形态（UI 侧） | 协议层未对等；若要多 server 支持，需补方法族 |

---

## 10. 施工卡（2 刀；默认关闭）

### 刀 1 · 传输适配器化 + 通道上界（约 1 天）

- 把 gaea 的 MCP 连接抽成 `Transport` 接口 + 若干适配器（stdio / http / 进程内），每个适配器自带
  单行/队列上限常量（8 MiB / 2 MiB 参照值，按 gaea 实际调）。
- 断言：某 server 起不来不影响其它 server；单行超限断该 server 不 OOM。

### 刀 2 · 目录缓存与启动解耦（约半天）

- `toolCatalogCache`：容量 32、TTL 30min、键含 server 身份（名称+传输+关键配置哈希）。
- 启动期取目录带 deadline（默认 2–5s），超时先跑、后台补齐并发 `ToolsChanged` 一类通知。

---

## 11. 不搬 / 风险 / 未读

- **不搬**：EMA（企业托管鉴权）全套、`perform_oauth_login.rs` 的浏览器回环流程细节、
  `in_process_transport`（gaea 无进程内 MCP server 需求）、`codex-mcp/src/{server,binding}.rs` 的暴露面。
- **不搬**：`ordered_byte_transport`（Rust 所有权模型下的顺序保证，Go 用 channel 即可）。
- **风险**：① 目录缓存的"身份键"若漏掉配置字段，会出现"改了 server 配置但工具列表还是旧的"；
  ② 启动 deadline 太短会让首回合缺工具（需"补齐后发通知"配合）；③ elicitation 方法族是厂商扩展，非标准。
- **未读（诚实边界）**：`rmcp-client/src/{rmcp_client.rs, oauth.rs, perform_oauth_login.rs, http_client_adapter.rs,
  stdio_server_launcher.rs}` 全文；`codex-mcp/src/{rmcp_client.rs, connection_manager.rs, catalog.rs, runtime.rs,
  server.rs}` 全文；gaea 侧 `internal/gaea/control/controller_mcp.go` 仅看文件大小与名称，未逐行读。
