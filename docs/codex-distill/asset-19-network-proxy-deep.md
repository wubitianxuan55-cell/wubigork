# 资产 19 · 出网代理深入：体积预算、凭据代理、连接归因

> 源码：`network-proxy/src/`（51 个文件，`proxy.rs` 124 KB / `runtime.rs` 91 KB / `http_proxy.rs` 82 KB /
> `credential_broker.rs` 76 KB / `socks5.rs` 46 KB / `certs.rs` 48 KB / `mitm.rs` 24 KB /
> `body_budget.rs` 6.5 KB / `attribution.rs` 4.8 KB / `connect_policy.rs` 9.5 KB）。
> 第一次挖（`hotspot-11`）只记了决策模型与域名匹配，本篇补三个工程机制。

---

## 一、体积预算：租约制，不是「限流」

```rust
//! Shared leases bound buffers across quota checks, secret substitution, and signing.
//! Eight requests each reserve space for a 16 MiB input and 16 MiB transformed output.
const BUFFER_SLOT_COUNT: usize = 8;
```

读法：**8 个槽位**，每个槽位预留「16 MiB 入 + 16 MiB 出」。请求要过代理必须**先租一个槽**，
租不到就排队或拒绝——于是内存上界是固定的 `8 × 32 MiB = 256 MiB`，与并发量无关。

**为什么这比「限流」聪明**：代理里要同时做配额检查、密钥替换、请求签名，
每一步都可能复制 body；用租约把「并发 × 单请求体积」一次性锁死，比在每个环节各写一个上限更难漏。

## 二、凭据代理：明文不进子进程

`credential_broker.rs` 的模块划分就是它的语义：

```
configured / destination / environment / matching / provider_config / providers / registry / replacement
```

- **`matching`**：哪些请求（目标 host + 路径）该被注入哪份凭据；
- **`replacement`**：把请求里的凭据**替换**成真实值（在代理侧完成）；
- **`registry`**：`active_credential_sources` / `prioritized_credentials` / **`is_builtin_shaped_credential`**
  —— 最后这个函数很关键：它用来判断「这个值看起来像内建凭据」，避免把用户自己的 API key 当成待替换项；
- 结果：**子进程只拿到占位符或环境标记**（`update_brokered_credentials_marker`），明文不落子进程内存。

配合 `mitm.rs`（按域名解 TLS 再注入）与 `certs.rs`（48 KB 证书处理），构成完整的中间人注入链路。

## 三、连接归因：magic frame + 短令牌

```rust
pub const PROXY_ATTRIBUTION_TOKEN_ENV_KEY: &str = "CODEX_NETWORK_PROXY_ATTRIBUTION";
const ATTRIBUTION_FRAME_MAGIC: &[u8; 8] = b"\0CDXPXY1";
const MAX_ATTRIBUTION_TOKEN_LEN: usize = 128;
const ATTRIBUTION_FRAME_TIMEOUT: Duration = Duration::from_secs(3);
```

Linux 侧的桥接进程在建立连接前，先发一个 8 字节魔数帧 + ≤128 字节令牌（3 秒超时），
代理据此知道「这条连接来自哪个子进程」——于是日志里能回答「是哪次工具调用访问了这个域名」。

## 四、对 gaea 的价值判断

| 机制 | 是否可移植 | 说明 |
|---|---|---|
| 域名 allowlist + 非公网默认拒绝 | **可以** | 已在 `hotspot-11` 记录，`netclient` + `sandbox` 落地 |
| 体积预算（槽位租约） | **可以**（改用途） | 不一定要做代理，但「并发 × 单请求体积」的租约思路可用于本地 HTTP 客户端与下载器 |
| 连接归因（魔数帧 + 短令牌） | **部分** | gaea 不需要代理归因，但「给每次出网请求打一个来源标记」用于审计有价值 |
| 凭据代理 + MITM + 证书管理 | **不做** | 需要自签 CA、按域名解密、替换凭据——单机应用没有「不信任的子进程」这个前提 |

**结论**：`credential_broker.rs` + `mitm.rs` + `certs.rs`（合计 ~150 KB）整块跳过；
只取「体积预算的租约思路」与「出网请求带来源标记」两条。

## 五、gaea 落地建议（小改）

1. `internal/netclient` 增加**并发租约**：`maxInflight × maxBodyBytes` 固定上界，
   超限排队而不是无界增长（当前若无上限，一次批量下载就能吃满内存）；
2. 每个出网请求带 `Origin` 标记（哪个工具/哪个会话发起），进日志便于事后审计；
3. 域名 allowlist 与「非公网默认拒绝」按 `hotspot-11` 的方案实施。

## 六、验收

1. 并发 32 个下载、单体内存涨幅不超过 `槽位 × 单请求上限`（用 `runtime.ReadMemStats` 断言）；
2. 超过并发上限时请求排队而不是失败（或按策略快速拒绝并有明确错误）；
3. 日志里每条出网请求能追到「哪个工具 / 哪个会话」，无归属的请求占比 0。

## 七、一句话

出网代理真正值得学的是**「用租约把内存上界钉死」**这条工程习惯；
凭据代理与 MITM 那套是给「有不可信子进程」的场景准备的，gaea 现在没有这个前提。
