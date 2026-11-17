# asset-56 · 执行服务实现层（B11 · 施工级 · 判定：实现不搬，借五条协议纪律）

> 证据：`codex-rs/exec-server/src/{client_inbound_request_limit.rs, websocket_pong_watchdog.rs,
> noise_relay/{mod.rs, message_framing.rs, ordered_ciphertext.rs}, remote/reconnect_backoff.rs,
> no_follow/{mod.rs, unix.rs, windows.rs, windows_volume_fallback.rs},
> sandbox_integrity/{mod.rs, bubblewrap.rs, seatbelt.rs, mxc.rs, runner.rs},
> discoverV2/{mod.rs, capability_manager.rs, capability_locations/*}, client/{connection_failure.rs, recovery.rs}}`
> （`rust-v0.165.0`）。协议层已由 `asset-21-exec-server.md` 覆盖，本篇只写**实现纪律**。
> gaea 侧无对应模块（单机执行）——本篇价值在"边界与心跳纪律"，不在功能移植。

---

## 1. 一句话结论

exec-server 有 6.6 万行，但**对 gaea 可搬的只有五条协议纪律**，且每条都是"数字 + 理由"型：
① 入站大小**按消息种类分档**（请求 8 KiB / 流式增量 2×body 上限 / 响应不限）；
② 保活用 **pong 看门狗**（ping 记 deadline、pong 清 deadline、超时理由写进错误）；
③ 加密中继的**序号永不回绕**（`u32` 用尽即协议错误，因为它是隐式 nonce 的排序键）；
④ WebSocket **帧/消息上限 256 KiB**，在 protobuf 与 Noise 校验**之前**就把分配钉死；
⑤ 重连退避 `1s → 30s`，且**连接稳定 120s 才重置退避**。

其余（能力发现 v2、远端注册、沙箱完整性探测、跨平台 safe-open）都是"多机执行"的配套，**判定不搬**。

---

## 2. 机制地图

| 关注点 | 文件 | 关键行 |
|---|---|---|
| 入站大小分档 | `client_inbound_request_limit.rs` | 11 `MAX_CLIENT_INBOUND_REQUEST_LEN = 8 * 1024`；13 `MAX_CLIENT_INBOUND_NOTIFICATION_LEN = 2 * MAX_HTTP_BODY_DELTA_BYTES`；15–38 按 `method` 与消息种类选上限（响应/错误不限） |
| 保活 | `websocket_pong_watchdog.rs` | 5/7 `WEBSOCKET_PONG_TIMEOUT`（测试 100ms / 生产 60s）；8 理由串 `"pong_timeout"`；20/24/28/32 `ping_sent/deadline/write_deadline/received_pong` |
| 中继上限 | `noise_relay/mod.rs` | 15 `MAX_NOISE_RELAY_WEBSOCKET_MESSAGE_SIZE = 256 * 1024`（注释：在 protobuf 与 Noise 记录校验前就限制分配）；16–23 同时设 `max_frame_size` 与 `max_message_size` |
| 序号纪律 | `noise_relay/mod.rs` | 25–35 `take_next_sequence`：**不 wrap**，`u32` 用尽返回 `Protocol("Noise relay sequence number exhausted")` |
| 分帧 | `noise_relay/message_framing.rs` | 10 `LENGTH_PREFIX_BYTES = size_of::<u32>()` |
| 重连退避 | `remote/reconnect_backoff.rs` | 5 `INITIAL_BACKOFF=1s`；6 `MAX_BACKOFF=30s`；8 `STABLE_CONNECTION_DURATION=120s` |
| 安全打开 | `no_follow/{unix.rs, windows.rs, windows_volume_fallback.rs}` | 按平台"不跟随符号链接"地打开文件（Windows 还有卷回退路径） |
| 沙箱完整性 | `sandbox_integrity/{bubblewrap, seatbelt, mxc, runner, dependencies}.rs` | 不假设沙箱生效，而是**探测验证**（按平台 runner）+ 依赖检查 + 遥测 |
| 能力发现 v2 | `discoverV2/{capability_manager.rs, capability_locations/*}` + 两个 cache | 位置候选 → 管理器 → 缓存（含 v2 专用缓存与 watcher） |

---

## 3. 五条协议纪律（逐条给出"为什么"）

### 3.1 入站大小按消息种类分档

```rust
pub(crate) const MAX_CLIENT_INBOUND_REQUEST_LEN: usize = 8 * 1024;
pub(crate) const MAX_CLIENT_INBOUND_NOTIFICATION_LEN: usize = 2 * MAX_HTTP_BODY_DELTA_BYTES;
```
- 只有四个"流式增量"方法（`exec/outputDelta`、`exec/exited`、`exec/closed`、`http/requestBodyDelta`）
  用大档；**响应与错误完全不限**（`Ok(Response(_)) | Ok(Error(_)) => return None`）。
- 注释说明了两件事：① 传输层可能在**知道 JSON-RPC 类型之前**就先物化一帧，所以请求档要能容纳它；
  ② 流式 body 在 base64 与 JSON-RPC 包装之前最多 1 MiB，因此通知档给 `2×`。
- **为什么值得抄**：一刀切上限要么卡死流式传输，要么让小请求也能吃满内存。分档才是正确形状。

### 3.2 pong 看门狗

```rust
ping_sent(now)      → deadline = Some(now + timeout)
received_pong()     → deadline = None
deadline()          → Option<Instant>（供写超时使用）
```
- 生产 60s、测试 100ms；超时理由固定为 `"pong_timeout"`（便于日志聚合与告警分类）。
- `write_deadline(now)` 在没有未决 ping 时返回 `now + timeout`——即"下一次写操作也要有上限"，
  避免"对端不读"把写阻塞无限拖长。

### 3.3 加密中继序号永不回绕

```
seq = next_seq; next_seq = next_seq.checked_add(1) 否则 Protocol("… sequence number exhausted")
```
注释写明：**relay 序号是隐式 Noise nonce 的显式排序键**，回绕到 0 会造成歧义（同 nonce 复用），
所以宁可**直接报协议错误并断链**，也不复用。

### 3.4 帧上限先于解析

`max_frame_size = max_message_size = 256 KiB`，注释：*"This bounds allocation in tungstenite before
protobuf and Noise record validation run."* —— 在**解析之前**限制，才叫防护；解析后再检查只是"事后报警"。

### 3.5 重连退避：稳定期才重置

`1s → 30s` 指数退避；**只有连接持续 `120s` 才把退避重置**——否则"连上就断"的抖动会把退避重置成 1s，
形成忙循环（这是最容易被忽略的一条）。

---

## 4. 沙箱完整性：不假设，而是探测

`sandbox_integrity/` 按平台提供 runner（`bubblewrap` / `seatbelt` / `mxc`），并检查依赖是否可用，
把结果打进遥测。**核心立场**：沙箱"配置成功"不等于"真的生效"，必须**主动探测**（跑一个受限于沙箱的最小动作，
看是否被正确拒绝/允许），再把结论作为事实记录。

对 gaea 的直接启发：`internal/gaea/sandbox` 若只"按配置声明沙箱"，同样需要一次"探测验证"。

---

## 5. 能力发现 v2（判定不搬，仅记录形状）

`capability_locations`（候选位置）→ `capability_manager`（含 `capability_watchers`）→ 两个缓存
（`capability_discovery_cache` / `capability_discovery_v2_cache`）。
v1 是"扫描"，v2 是"上报位置 + 监听变化"。gaea 单机不涉及多执行器，**不搬**；
唯一可借的是"发现结果要有缓存 + 失效 watcher"，与 B8 的工具缓存同构。

---

## 6. 常量与上限

| 名称 | 值 | 位置 | 生产/测试 | 语义 |
|---|---|---|---|---|
| 入站请求上限 | 8 KiB | `client_inbound_request_limit.rs:11` | 生产 | 普通请求/未知消息 |
| 入站通知上限 | `2 × MAX_HTTP_BODY_DELTA_BYTES` | `:13` | 生产 | 仅四个流式增量方法 |
| pong 超时 | 60s / 100ms | `websocket_pong_watchdog.rs:5,7` | 生产 / 测试 | 保活判死 |
| 中继帧/消息上限 | 256 KiB | `noise_relay/mod.rs:15` | 生产 | 解析前的分配上界 |
| 序号空间 | `u32` 不回绕 | `noise_relay/mod.rs:25–35` | 生产 | nonce 排序键 |
| 重连退避 | 1s → 30s | `remote/reconnect_backoff.rs:5–6` | 生产 | 指数退避 |
| 稳定期 | 120s | `:8` | 生产 | 退避重置门槛 |
| 长度前缀 | 4 B（`u32`） | `noise_relay/message_framing.rs:10` | 生产 | 分帧 |

---

## 7. 失败路径

1. **超限消息**：`client_inbound_message_exceeded_limit` 返回 `Some(max_len)`，上层据此拒绝并断开（不是截断）。
2. **pong 超时**：watchdog deadline 到期 → 断链，理由 `pong_timeout`（可观测、可聚合）。
3. **中继协议错误**：`NOISE_RELAY_RESET_REASON = "noise_relay_protocol_error"`，序号耗尽也走这条。
4. **重连失败**：退避递增到 30s 封顶；期间注册重试另有 `registration_retry`。
5. **沙箱探测失败**：记遥测并**降级为"未验证"**（不静默当成"已沙箱"）。

---

## 8. gaea 对照

| 维度 | codex | gaea | 结论 |
|---|---|---|---|
| 跨机执行器 | 完整（发现/注册/中继/远端 FS/进程） | 无（单机） | 不搬 |
| 入站大小分档 | 请求 8 KiB / 流式 2× / 响应不限 | Wails binding 无显式上限 | **可借**：给 HTTP/SSE 与 WS 入口分档 |
| 保活 | pong 看门狗 + 固定理由串 | 未读到显式保活 | **可借**（若做 WS/长连接） |
| 帧上限先于解析 | 256 KiB | 未读到 | **可借**（任何外部帧入口都适用） |
| 退避重置门槛 | 稳定 120s 才重置 | `provider/retry.go` 有退避但**无稳定性门槛**（未读到） | **可借**：连接型退避必须有"稳定期" |
| 沙箱探测 | 平台 runner 主动验证 | `internal/gaea/sandbox`（未读细节） | **可借**：配置成功 ≠ 生效 |

---

## 9. 施工卡（2 刀，均为"借纪律"，不引入远端执行）

### 刀 1 · 入站分档与帧上限（约半天）

- 在 HTTP/SSE 与（若存在的）WS 入口加 `maxRequestBytes` / `maxStreamDeltaBytes`（分档）与
  `maxFrameBytes`（解析前生效）；超限返回**明确的 4xx/关闭码**，不做静默截断。
- 断言：普通请求 8 KiB 上限生效；流式增量走大档；响应/错误不设限。

### 刀 2 · 保活与退避稳定性（约半天）

- 若 gaea 有长连接：加 pong 看门狗（ping 记 deadline / pong 清 deadline / 超时理由串固定）。
- 连接型退避加"稳定期"门槛（默认 120s）：稳定后重置退避，避免抖动忙循环。

---

## 10. 验收断言

| # | 断言 |
|---|---|
| E1 | 9 KiB 普通请求被拒绝且连接不进入半开状态（明确错误码/关闭码） |
| E2 | 流式增量 1.5 MiB 通过（大档），2.1 MiB 被拒 |
| E3 | 响应/错误消息不受限（大响应能回传） |
| E4 | ping 后 61s 无 pong → 断链，日志含固定理由 `pong_timeout` |
| E5 | 连接抖动：连 3s 即断，退避持续增长（不重置）；连满 120s 再断，退避重置为 1s |
| E6 | 任何超限都在**解析前**拒绝（用畸形帧验证不进入解码器） |

---

## 11. 不搬 / 风险 / 未读

- **不搬**：`discoverV2/*`、`remote/*`（注册/auth/诊断）、`server/*`（handler/registry/session）、
  `noise_relay/*` 的 Noise 实现、`no_follow/*` 平台实现、`shell_snapshot*`、`process*`。
- **风险**：① 入站分档要按**真实调用方**定档，照抄 8 KiB 可能卡死现有前端（gaea 的 bindings 有整页状态）；
  ② pong 看门狗在**单进程内**没有对端，只在真做长连接时才有意义。
- **未读（诚实边界）**：`discoverV2/*`、`server/*`、`remote/*`、`noise_relay/{executor_stream,stream_handler,
  ordered_ciphertext}.rs` 全文、`sandbox_integrity/{bubblewrap,seatbelt,mxc}.rs` 全文、`client/*` 大部分文件。
