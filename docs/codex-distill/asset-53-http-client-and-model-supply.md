# asset-53 · HTTP 客户端、网络策略与模型供给（B9 · 施工级）

> 证据：`codex-rs/http-client/src/{route_aware_client_pool.rs, route_aware_client_pool/execution.rs,
> route_aware_redirect.rs, network_policy.rs, client_builder.rs, outbound_proxy.rs, windows_proxy_auth.rs,
> custom_ca.rs}`、`models-manager/src/{manager.rs, cache.rs}`、`model-provider/src/{provider.rs, auth.rs,
> models_endpoint.rs}`、`codex-api/src/{sse/responses.rs, rate_limits.rs, common.rs, error.rs}`（`rust-v0.165.0`）。
> gaea 侧只读对照：`internal/gaea/provider/retry.go`（其余未读，见 §11 未读清单）。
> 仅出机制与施工卡，不含代码改动。

---

## 1. 一句话结论

codex 的出网不是"一个 reqwest 客户端"，而是**三层可判定结构**：
① **路由感知连接池**——按"解析出的出网路由"复用连接（上限 16 条路由），**每一跳重定向都重新做一次路由决策**；
② **网络策略许可**——发送前 `acquire(url)` 拿 permit，拒因是**确定性的四态**（不可用 / 目标被拒 / 被撤销 / 传输不支持），
与可重试的网络故障**分家**；
③ **模型供给与目录**——目录有磁盘缓存（`models_cache.json`，TTL 300s），鉴权恢复由 provider 自己拥有
（`NotConfigured / Recovered`），SSE 侧把服务端权威信息（模型名、限流快照、turn state）从**响应头**吸进事件流。
gaea 已有成熟的**重试/退避层**（`retry.go`：500ms→8s 默认、429 用 5s→60s、解析 `Retry-After` 并封顶），
缺的是**路由池 + per-hop 决策 + 策略许可**这三件。

---

## 2. 机制地图

| 关注点 | 文件 | 关键行 |
|---|---|---|
| 路由池 | `http-client/src/route_aware_client_pool.rs` | 31 `MAX_CACHED_ROUTES=16`；53–62 结构（`clients: Mutex<HashMap<Route, TransportClient>>` + `client_build: tokio::Mutex` + rustls 缓存）；47–51 文档说明"按解析路由复用、每跳重定向重新决策" |
| 请求执行 | `http-client/src/route_aware_client_pool/execution.rs` | 38–43 `execute`；56–70 `send_with_resolver`（**先 acquire 策略许可**再发）；24–30 重定向与 TLS 回退导入 |
| 重定向 | `http-client/src/route_aware_redirect.rs` | 31 `MAX_REDIRECTS = 10`；跨源去敏感头 + 补 `Referer` |
| 网络策略 | `http-client/src/network_policy.rs` | 19–39 `DestinationPolicy::{Unrestricted, Restricted{allowed_hosts}}`；41–52 `NetworkPolicyDenied` 四态；257 `acquire`；270 `acquire_for_unsupported_sdk`；151/169/223 revision 与 `changes()` watch |
| 客户端构造 | `http-client/src/client_builder.rs` | 41 `connect_timeout`；90–99 严格池化客户端 + 路由池构造 |
| 代理 / TLS | `http-client/src/{outbound_proxy.rs(795), outbound_proxy/{windows,macos}.rs, windows_proxy_auth.rs(265), custom_ca.rs(755)}` | 平台分流 + Windows 代理认证管道 + 三态 CA 回退 |
| 模型目录 | `models-manager/src/manager.rs` | 38 `MODEL_CACHE_FILE="models_cache.json"`；39 `DEFAULT_MODEL_CACHE_TTL=300s`；305 `OpenAiModelsManager`；317 `StaticModelsManager`；384 `with_provider_catalog` |
| 供给与鉴权 | `model-provider/src/provider.rs` | 42–45 `ProviderAccountState`；49–54 `ProviderUnauthorizedRecovery::{NotConfigured, Recovered}`；58–61 恢复期文案结构 |
| SSE / 限流 | `codex-api/src/sse/responses.rs` | 32–36 五个响应头常量；38–60 `spawn_response_stream`（先解析限流与权威模型名）；`rate_limits.rs` 23/28/57/135/181 |

---

## 3. 路由感知连接池

```rust
struct RouteAwareClientPool {
    http_client_factory: HttpClientFactory,
    route_class: ClientRouteClass,
    client_builder: HttpClientBuilder,
    default_headers: HeaderMap,
    custom_ca_fallback: CustomCaFallback,   // LegacyDirect | Disabled(默认) | LegacyTransportDefault
    clients: Arc<Mutex<HashMap<OutboundProxyRoute, TransportClient>>>,
    client_build: Arc<tokio::sync::Mutex<()>>,   // 构造串行化
    rustls_clients: Option<RustlsClientCache>,
}
```

四条关键语义（都能在注释里找到原意）：

1. **"按路由"而不是"按主机"复用**：`OutboundProxyRoute` 是**解析后的出网路径**（直连 / 某个代理）。
   同一主机的请求可能因为策略变化走不同代理，因此缓存键是路由，不是 URL。
2. **缓存有上限**：`MAX_CACHED_ROUTES = 16`——避免"每次策略抖动都新建一个连接池"。
3. **解析先于凭据处理**：注释写明"在 reqwest 处理 URL 凭据之前，用完整输入 URL 解析初始路由"，
   否则 `user:pass@host` 会让路由判定错位。
4. **重定向按新请求处理**：每一跳都回到池里做一次路由决策，**但连接仍按路由复用**——
   这样"跳到另一个域"既换策略又不丢连接复用。

`client_build` 是**构造互斥**（`tokio::sync::Mutex<()>`）：同一时刻只允许一个协程在造客户端，
其余等它造完直接命中缓存——防止惊群式重复建池。

### 执行顺序（别写反）

```
① network_policy().acquire(request.url())  → 拿 permit（拒绝则直接返回确定性错误）
② 按 permit 的路由取/造 TransportClient
③ 发送；若响应是可重定向状态 → 走 route_aware_redirect（≤10 跳，跨源去敏感头、补 Referer）
④ 若 TLS 握手类失败且 should_retry_with_rustls → 用 rustls 后端重试一次
```

---

## 4. 网络策略：确定性拒绝 vs 可重试故障

```rust
enum DestinationPolicy { Unrestricted, Restricted { allowed_hosts: BTreeSet<String> } }
enum NetworkPolicyDenied { Unavailable, Destination, Revoked, UnsupportedTransport }
```

- `Restricted` 只放行 **https / wss** 且**主机名精确匹配**（先剥掉末尾 `.`），不做通配、不做后缀匹配。
- 四个拒因各自独立：`Unavailable`（策略还没拿到）/ `Destination`（目标不在白名单）/ `Revoked`（许可被撤销）/
  `UnsupportedTransport`（该 SDK 传输被禁用）——注释强调这是**确定性拒绝，与可重试的网络失败不同**。
- 支持 `revision` + `changes()` watch：策略变更后**已发出的 permit 可按 revision 失效**（`Revoked`），
  这是"运行中收紧出网"的实现方式。
- `acquire_for_unsupported_sdk()`：对不支持的传输直接给出 `UnsupportedTransport`，而不是放行后再失败。

**为什么值得抄**：把"不允许"和"连不上"分成两类错误，前端才能给出正确处置（改配置 vs 重试）。

---

## 5. 代理与 TLS

- 出网代理按平台分文件实现（`outbound_proxy/{windows,macos}.rs`），Windows 另有 `windows_proxy_auth.rs`
  用**命名管道**把代理凭据交给传输层（`ProxyPipe`）——避免把口令塞进 URL。
- 自定义 CA 有**三态回退**：`LegacyDirect` / `Disabled`（默认）/ `LegacyTransportDefault`——
  默认不启用旧行为，只有显式要求才走兼容路径。
- TLS 后端可回退：`SelectedTlsBackend::{TransportDefault, RustlsFallback}` + `should_retry_with_rustls`，
  失败只重试**一次**（换后端，不换参数）。

---

## 6. 模型目录与供给

- `OpenAiModelsManager` 拉远程目录，落盘 `models_cache.json`，`DEFAULT_MODEL_CACHE_TTL = 300s`；
  `StaticModelsManager` 用于无网络/固定目录场景；`with_provider_catalog` 允许把"provider 自带目录"合进来。
- 目录响应带 `ModelsEndpointResponse` 与 **etag**（SSE 侧 `X-Models-Etag` 头，见 §7）——可做"没变就不重下"。
- 供给层把"账号状态"结构化：`ProviderAccountState { account, requires_openai_auth }`；
  鉴权失败恢复**由 provider 自己拥有**：`ProviderUnauthorizedRecovery::{NotConfigured, Recovered}`，
  并带面向用户的"开始恢复/恢复成功"两句话（`ProviderAuthRecoveryMessages`）。
- 多 provider 并存：OpenAI / Amazon Bedrock（含 catalog 与 credential export）/ GCP / 网关 OAuth；
  另有一层 `workspace_routing`（工作区路由）。

---

## 7. SSE：把服务端权威信息吸进事件流

`spawn_response_stream(stream_response, idle_timeout, telemetry, turn_state)` 在**建流之前**读响应头：

| 头 | 用途 |
|---|---|
| `openai-model` | **服务端权威模型名**（覆盖客户端以为的模型） |
| `x-request-id` | 上游请求 id（排障主键） |
| `x-codex-turn-state` | 回合状态令牌（写入 `OnceLock<String>`，供后续请求带回） |
| `x-reasoning-included` | 是否含推理内容（存在即为真） |
| `X-Models-Etag` | 目录版本（判断是否重下） |

限流从**同一响应头**里解析出**多个**快照（`parse_all_rate_limits` 返回 `Vec<RateLimitSnapshot>`），
另有 `parse_rate_limit_event(payload)` 从流内事件补充；`parse_promo_message` 读促销/公告文案。
`SafetyBuffering` 的处理方式也是**由响应头驱动**（`treatment_from_headers`）。
流内错误分两层：`parse_failed_response`（流内失败）与 `parse_terminal_provider_error`（终止性 provider 错误）。

---

## 8. 常量与上限

| 名称 | 值 | 位置 | 生产/测试 | 语义 |
|---|---|---|---|---|
| `MAX_CACHED_ROUTES` | 16 | `route_aware_client_pool.rs:31` | 生产 | 路由→连接池缓存上限 |
| `MAX_REDIRECTS` | 10 | `route_aware_redirect.rs:31` | 生产 | 单请求重定向上限 |
| `MODEL_CACHE_FILE` | `models_cache.json` | `models-manager/src/manager.rs:38` | 生产 | 目录磁盘缓存文件名 |
| `DEFAULT_MODEL_CACHE_TTL` | 300s | `models-manager/src/manager.rs:39` | 生产 | 目录缓存有效期 |
| TLS 回退次数 | 1（换后端） | `tls_backend_fallback::should_retry_with_rustls` | 生产 | 只重试一次，不放大 |
| 允许协议 | 仅 `https`/`wss`（Restricted 时） | `network_policy.rs:32` | 生产 | 白名单模式的硬约束 |
| 默认 CA 回退 | `Disabled` | `route_aware_client_pool.rs:33–39` | 生产 | 默认不启用旧行为 |

---

## 9. 失败路径

1. **策略拒绝**：`acquire` 返回 `NetworkPolicyDenied::{Unavailable|Destination|Revoked|UnsupportedTransport}`，
   在**发送前**结束，不产生网络流量。
2. **构造失败**：`RouteAwareClientPoolError::{Resolve, Build, BuildTask}`——解析路由失败 / 建 HTTP 客户端失败 /
   构造任务 panic（`JoinError`）分别可辨。
3. **重定向过多**：超过 10 跳即终止（错误来自 `route_aware_redirect`），不再继续。
4. **TLS 后端不兼容**：只回退一次到 rustls；再失败即上抛，不再抖动。
5. **模型目录拉取失败**：走缓存（TTL 内）；无缓存则回退静态目录（`StaticModelsManager` 路径）。
6. **鉴权失效**：provider 声称 `Recovered` 才允许重试一次；`NotConfigured` 时把错误交给上层（不静默重试）。

---

## 10. gaea 对照（只读实测）

| 维度 | codex | gaea | 差在哪 |
|---|---|---|---|
| 重试/退避 | 由 API 层与传输层分担，SSE 有 idle timeout | `provider/retry.go`：`DefaultBackoff 500ms→8s`、`RateLimitBackoff 5s→60s`、全抖动、`Retry-After` 解析并 `cap`、`IsRetryableStatus` | **gaea 这块更完整**（有 jitter 与 Retry-After 封顶）；不用搬 |
| 连接池 | 按**出网路由**分池（≤16），构造互斥 | 未读到（`internal/gaea/provider/` 未逐文件读） | 待验证；若 gaea 是单 `http.Client`，则"策略切换后连接复用错位"是潜在问题 |
| 重定向 | 每跳重新决策路由 + 跨源去敏感头 | 未读到显式重定向处理 | 缺"跨源去敏感头"这类默认安全行为 |
| 出网策略 | `DestinationPolicy` + permit + revision 失效，**发送前**拒绝 | 未读到统一出网许可层（沙箱在 `internal/gaea/sandbox`，未读） | 缺"确定性与可重试分离"的错误分类 |
| 模型目录 | 磁盘缓存 + 300s TTL + etag | 未读到（`internal/gaea/provider` 目录清单里无 models 缓存模块） | 若每次都重拉目录，冷启动与限额都吃亏 |
| SSE 元数据 | 权威模型名 / turn state / 限流快照 / 安全缓冲处理均由头驱动 | 未读到 | gaea 若用"客户端指定模型"，会与服务端实际模型不一致 |

> 结论：B9 里对 gaea **真正值得搬的只有三件**——① 重定向时的跨源去敏感头 + 每跳策略复核；
> ② 统一的"出网许可"错误分类（确定性 vs 可重试）；③ 模型目录的 TTL 缓存 + etag 复用。
> 重试层不要动，gaea 现有实现更成熟。

---

## 11. 施工卡（2 刀；默认关闭）

### 刀 1 · 出网许可与重定向安全（约半天）

- 新增 `internal/gaea/provider/outbound.go`：`type Policy struct{ Mode; AllowedHosts map[string]struct{} }`，
  `Acquire(url) (Permit, error)`，错误四态 `Unavailable / Destination / Revoked / UnsupportedTransport`。
- 在 `provider` 的 HTTP 发送入口统一 `Acquire`（**发送前**），并在重定向钩子里：跨源则去除
  `Authorization` / `Cookie` / `Proxy-Authorization`，补 `Referer`，`≤10` 跳。
- 禁区：不改 `retry.go` 的退避参数与 `Retry-After` 口径；策略默认 `Unrestricted`（关时行为等价）。

### 刀 2 · 模型目录缓存（约半天）

- `models_cache.json` 等价物：`{fetched_at, etag, models[]}`，TTL 300s；命中即用，过期先条件请求（`If-None-Match`）。
- 拉取失败 → 用过期缓存（若存在）→ 再退静态内置目录；三层兜底顺序固定。

---

## 12. 验收断言

| # | 断言 |
|---|---|
| N1 | `Restricted` 策略下 `http://` 与 `ws://` 一律 `Destination` 拒绝（协议不在白名单） |
| N2 | 主机名末尾带 `.` 时仍能匹配（先剥离再比） |
| N3 | 策略变更后旧 permit 失效为 `Revoked`，新请求按新策略判定 |
| N4 | 跨源 302：请求头中的 `Authorization`/`Cookie` 被移除，`Referer` 被补上；同源 302 不删头 |
| N5 | 重定向 11 跳终止并给出明确错误，不发第 11 次请求 |
| N6 | TLS 后端回退只发生一次（第二次同类失败直接上抛） |
| N7 | 目录缓存：TTL 内不发网络请求；过期且 `If-None-Match` 命中 304 时只更新 `fetched_at` |

---

## 13. 不搬 / 风险 / 未读

- **不搬**：路由池的 `HashMap<Route, TransportClient>` 具体形态（gaea 若单进程单账号，按"策略版本"分池即可）；
  Windows 代理认证管道（命名管道属平台细节）；Bedrock/GCP 供给（gaea 无对应 provider）。
- **不搬**：`ProviderUnauthorizedRecovery` 两态（gaea 的 provider 错误分类已够用，见 `provider.go:328`）。
- **风险**：① 出网许可若默认 `Restricted` 会直接打断现有 provider（必须默认关闭）；
  ② 目录缓存与"用户手动切换模型"要区分"目录里的模型"与"本地覆盖"，否则会出现"缓存里没有但用户想用"。
- **未读（诚实边界）**：`internal/gaea/provider/` 除 `retry.go` 外的全部文件、`internal/gaea/sandbox`、
  gaea 的模型目录/模型列表实现；`http-client/src/outbound_proxy.rs` 全文（只读结构与平台分层）、
  `model-provider/src/{amazon_bedrock,gcp}/**`。
