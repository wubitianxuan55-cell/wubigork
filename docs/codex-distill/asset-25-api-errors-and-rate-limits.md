# 资产 25 · API 层错误分类与限流解析（codex-api 侧）

> 源码：`codex-api/src/{error.rs 3.2 KB, rate_limits.rs 13 KB, api_bridge.rs 16.8 KB, common.rs 15 KB}`。
> 对照：gaea `internal/ai/{client.go 17.8 KB, client_retry.go, client_failover.go, client_stream.go 21 KB, types.go 13 KB}`。
> **范围说明**：本篇只覆盖「错误分类 + 限流解析 + 错误映射」这三块；
> 请求体/流式事件的**字段形状**在 `codex-api/src/common.rs` 与 `codex-client`，本篇未展开（下轮补）。

---

## 一、错误分类：18 个变体，按「能否重试」自然分堆

```rust
pub enum ApiError {
    Transport(TransportError),
    Api { status: StatusCode, message: String },
    Stream(String),
    ContentFilter,
    ResponseIncomplete(Box<ResponseIncompleteError>),
    ContextWindowExceeded,
    QuotaExceeded,
    UsageNotIncluded,
    Retryable { message: String, retry_after: Option<RetryAfter> },
    RateLimitExceeded { message: String, retry_after: Option<RetryAfter> },
    RateLimit(String),
    InvalidRequest { message: String },
    ModelNotFound { message: String, status: Option<StatusCode> },
    InvalidPrompt { message: String },
    CyberPolicy { message: String },          // 策略类
    BioPolicy { message: String },
    MisalignmentPolicyViolation { message: String, misalignment: Option<..> },
    FlexUnavailable,
    ServerOverloaded { retry_after: Option<RetryAfter> },
}
```

**三个设计要点**

1. **`retry_after` 是结构化字段**（`Option<RetryAfter>`），不是从错误文本里正则抠出来的字符串——
   重试逻辑直接用它算等待时间，不依赖文案。
2. **策略类错误单列**（`CyberPolicy` / `BioPolicy` / `Misalignment*`）：它们**不可重试、也不该自动改写提示词重试**，
   混进 `InvalidPrompt` 会导致「被拒了还在原地重试」。
3. **`ContextWindowExceeded` 与 `QuotaExceeded` 分开**：前者要压缩历史，后者只能等额度或有条件降级——
   两种处置完全不同，合成一个 `429` 就没法自动处理。

## 二、`code × status` 双重校验（防供应商乱报）

```rust
fn parse_terminal_provider_error(error: Option<&Value>, status: Option<StatusCode>) -> Option<ApiError> {
    let code = error.get("code")?.as_str()?;
    match (code, status) {
        ("flex_unavailable", None | Some(TOO_MANY_REQUESTS)) => Some(FlexUnavailable),
        ("model_not_found", None | Some(BAD_REQUEST | FORBIDDEN | NOT_FOUND)) => Some(ModelNotFound{..}),
        _ => None,
    }
}
```

**只有 `code` 与 HTTP 状态同时落在允许组合里才认定**，否则宁可不认（返回 `None`，交给通用分支）。
理由：供应商的 `code` 字段经常被复用或拼错，「只看 code」会做出错误处置
（例如把一个普通的 500 当 `model_not_found`，然后切模型——问题并没解决）。

## 三、限流：从响应头解析成结构化快照

```rust
parse_default_rate_limit(headers) -> Option<RateLimitSnapshot>
parse_all_rate_limits(headers)    -> Vec<RateLimitSnapshot>   // 多 family，按 limit id 分组
```

`RateLimitSnapshot` 里带 `PlanType` / `CreditsSnapshot` / `RateLimitWindow`（窗口内已用/剩余/重置时间）。
`api_bridge.rs` 把这些映射成内部错误 `CodexErr::{ConnectionFailed, RetryLimitReached, UsageLimitReached, UnexpectedResponse}`。

**为什么值得抄**：限流信息不是「一个 429」，而是**带窗口与重置时间的结构化状态**。
有了它才能做「等到重置」「换 key」「降级模型」这类决策；只有一个 429 就只能干等或放弃。

## 四、错误只在一个地方翻译

`api_bridge::map_api_error(ApiError) -> CodexErr` 是**唯一**的映射点：外部 API 错误 → 内部错误类型。
好处是所有调用方看到的是同一套内部错误，重试/降级/压缩策略集中在一处判断，
不会出现「A 模块把 429 当可重试、B 模块当致命」。

---

## 五、gaea 对照与建议

gaea 已有 `client_retry.go`（重试）与 `client_failover.go`（故障转移），但建议核对三件事：

1. **错误分类表是否齐**：至少有 `可重试（含 RetryAfter）` / `限流` / `窗口超限` / `配额超限` /
   `内容过滤` / `响应不完整` / `无效请求` / `模型不存在` 这几类，且**分别处置**；
2. **`retry_after` 是否独立字段**：如果现在是从错误字符串里正则提取，建议改成结构化；
3. **是否有单一映射点**：若 `ai` 包与上层各自判 429，建议收拢成一个 `Classify(err) kind`。

**gaea 不需要抄的**：`CyberPolicy` / `BioPolicy` / `Misalignment` 这些是 OpenAI 侧的合规分类，
gaea 走本地模型或第三方 provider 不会有；`FlexUnavailable`（弹性容量）同理。

## 六、验收（可断言）

1. 构造 `{code:"model_not_found"}` + HTTP 500 → **不**判定为模型不存在（走通用错误）；
2. 构造 `{code:"model_not_found"}` + HTTP 404 → 判定为模型不存在，且**不重试**；
3. 限流响应带 `retry-after: 30` → 重试逻辑等待约 30 秒，而不是退避到下一次固定间隔；
4. `ContextWindowExceeded` 触发压缩路径，`QuotaExceeded` 触发等待/降级路径——两条分支互不串；
5. 全仓只有一处把 provider 错误翻译成内部错误（静态检查：`Classify` 调用点唯一）。

## 七、一句话

这层真正值得抄的是**两条纪律**：`retry_after` 结构化、`code` 必须与状态码互相印证；
以及**一个结构事实**：错误分类里「窗口超限」与「配额超限」必须是两条不同的路，否则自动处置无从谈起。
