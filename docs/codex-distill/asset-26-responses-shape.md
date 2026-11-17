# 资产 26 · Responses 请求体与流式事件模型

> 源码：`codex-api/src/{common.rs 15 KB, api_bridge.rs}`。
> 对照：gaea `internal/ai/{types.go 13 KB, client_stream.go 21 KB}`——gaea 走的是
> **OpenAI Chat Completions** 形态（`SSEChunk` + `ChatDelta{Content, ReasoningContent, ToolCalls}`），
> codex 走的是 **Responses API** 形态（`ResponseEvent` 15 个变体）。

---

## 一、请求体：字段顺序是接口的一部分

```rust
pub struct ResponsesApiRequest {
    // 注释原文：路由字段放最前——serde 按声明顺序序列化，
    // 网关可能在「多兆字节 input」之前就增量预检请求体。
    pub model: String,
    pub stream: bool,
    pub service_tier: Option<String>,
    pub input: Vec<ResponseItem>,          // ← 大块放后面
    pub tools: Option<ResponsesApiTools>,
    pub tool_choice: String,
    pub parallel_tool_calls: bool,
    pub reasoning: Option<Reasoning>,
    pub store: bool,
    pub stream_options: Option<StreamOptions>,
    pub include: Vec<String>,
    pub prompt_cache_key: Option<String>,
    pub text: Option<TextControls>,
    pub client_metadata: Option<HashMap<String, String>>,
    pub access_programs: Option<AccessPrograms>,
}
```

**这条注释值得记住**：`model` / `stream` / `service_tier` 放最前，`input` 放后面——
因为网关做限流、路由、鉴权时只需要前几个字段，把大字段放后面能让它**早点决策、早点拒绝**，
不必等整个 body 收完。gaea 如果自建代理或做本地网关，字段顺序应当同理。

## 二、流式事件：15 个变体，覆盖五类

```rust
pub enum ResponseEvent {
    Created { response_id: Option<String> },
    SafetyBuffering(SafetyBuffering),
    OutputItemAdded(ResponseItem),
    OutputItemDone(ResponseItem),
    ServerModel(String),                 // ★ 服务端可能换模型
    ModelVerifications(Vec<ModelVerification>),
    TurnModerationMetadata(..),
    ServerReasoningIncluded(bool),       // ★ 服务端已计入推理 token
    Completed { response_id, token_usage: Option<TokenUsage>,
                usage_metadata: Option<ResponseUsageMetadata>,
                end_turn: Option<bool> },  // ★ 可能为 None
    OutputTextDelta(String),
    ToolCallInputDelta { item_id, call_id: Option<String>, delta: String },
    ReasoningSummaryDelta { delta, summary_index: i64 },
    ReasoningSummaryDone { item_id, text, summary_index: i64 },
    ReasoningContentDelta { delta, content_index: i64 },
    ReasoningSummaryPartAdded { summary_index: i64 },
    RateLimits(RateLimitSnapshot),       // ★ 限流在流里推
    ModelsEtag(String),
}
```

按用途分成五类：**生命周期**（Created/Completed）、**内容增量**（OutputTextDelta/ToolCallInputDelta/
Reasoning*）、**服务端事实**（ServerModel/ServerReasoningIncluded/RateLimits/ModelsEtag）、
**策略与验证**（SafetyBuffering/ModelVerifications/TurnModerationMetadata）、**条目边界**（OutputItemAdded/Done）。

## 三、五条对 gaea 有价值的细节

1. **`end_turn: Option<bool>`**：注释明说「某些 provider 不设这个字段，所以为 None 时要靠兜底逻辑」。
   gaea 现在靠 `finish_reason` 判断结束——两者都要防「字段缺失」。
2. **`ServerModel(String)`**：服务端可能把你请求的模型路由到另一个（安全路由）。
   **客户端不能假设「实际模型 = 请求模型」**，否则 token 计价与 `comp_hash` 都会错。
3. **`ServerReasoningIncluded(true)`**：来自响应头 `X-Reasoning-Included: true`，
   含义是「服务端已经计过历史推理 token」，客户端**不要重复估算**。
   这条直接关系到上下文占用算得准不准。
4. **`RateLimits(RateLimitSnapshot)` 在流里**：限流信息不必等响应头，流中途就能拿到，
   可以立刻做「降级/等待」决策，而不是等这次跑完再发现。
5. **推理增量带 `summary_index` / `content_index`**：说明推理输出可能是**多段**的（summary 分片 + content），
   客户端要按 index 归并，不能简单拼串。

## 四、gaea 的差异与判断

| 维度 | codex（Responses） | gaea（Chat Completions） | 判断 |
|---|---|---|---|
| 协议 | Responses API | Chat Completions（多数 provider 兼容） | **不切换**：gaea 要接多种本地/第三方 provider，chat 形态兼容性更好 |
| 文本增量 | `OutputTextDelta` | `ChatDelta.Content` | 等价 |
| 工具参数增量 | `ToolCallInputDelta{item_id, call_id}` | `ChatToolCallDelta`（按 Index 拼装） | 等价，codex 多了 item_id 关联 |
| 推理内容 | `ReasoningContentDelta{content_index}` + summary 分片 | `ChatDelta.ReasoningContent` 单流 | **gaea 更简单**：多数本地模型只有单流推理 |
| 结束信号 | `Completed{end_turn: Option<bool>}` | `finish_reason` | 两者都要兜底 |
| 服务端事实 | ServerModel / ServerReasoningIncluded / ModelsEtag | 无 | **codex 更完备**（多 provider 聚合场景有用） |
| 限流 | 流内 `RateLimits` 事件 | 靠响应头 | codex 更及时 |

**结论：不切协议，借四个字段语义。**

## 五、gaea 落地建议（小改）

1. `SSEChunk` 增加可选字段 `ServerModel string`（若 provider 响应头/字段带实际模型名就填），
   并在 usage 计费时优先用它；
2. 增加 `ReasoningIncluded bool`（对应 `X-Reasoning-Included` 一类头），置位时**跳过推理 token 估算**；
3. `finish_reason` 为空时，用「有内容 + 无工具调用 + 流正常关闭」兜底判结束，并记一条事件（便于排查）；
4. 若将来接限流头，把 `RateLimit` 快照放进流事件而不是只在响应头读一次。

## 六、验收

1. 流结束但 `finish_reason` 缺失 → 走兜底判定，回合正常收尾且事件里有 `end_turn_fallback`；
2. provider 返回的实际模型名与请求不同 → usage 记录里体现实际模型（不是请求模型）；
3. `X-Reasoning-Included: true` 时，上下文占用估算不再叠加推理 token（用两次相同请求的估算值对比）；
4. 工具参数增量按 Index 拼装后与完整 `tool_calls` 一致（乱序分片也能拼对）。

## 七、一句话

Responses 的字段形状不必照搬（gaea 走 chat 形态更现实），但 **`end_turn` 要防 None、
实际模型要认服务端的、推理 token 不要重复计** 这三条语义，是任何协议下都成立的坑。
