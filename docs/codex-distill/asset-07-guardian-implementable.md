# 资产 07 · Guardian 施工级：输出契约 / 容错 / 熔断 / 超时

> 源码：`ext/guardian-reviewer/src/`（19 个实现文件）、`ext/guardian-v2/src/`（25 个）、
> `core/src/guardian/`（21 个）。本篇只挑**施工必需**的部分：评审模型输出什么、出错怎么退、什么时候熔断。
> 二进制侧证据：`policy_template`（9,734 字符）与 `policy`（8,281 字符）已提取在
> `C:\AI\re\codex-0.165.0\embedded\policies\`。

---

## 一、输出契约（可以照抄的 JSON Schema）

评审要求模型输出**严格 JSON**，schema 原文：

```json
{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "risk_level":         { "enum": ["low", "medium", "high", "critical"] },
    "user_authorization": { "enum": ["unknown", "low", "medium", "high"] },
    "outcome":            { "enum": ["allow", "deny"] },
    "rationale":          { "type": "string" }
  },
  "required": ["outcome"]
}
```

**只有 `outcome` 是必填**——这个设计很关键：低风险动作鼓励模型直接回 `{"outcome":"allow"}`
（省 token、省延迟），其余三项缺失时由解析器给默认值。

## 二、容错解析（三层兜底，代码不到 20 行）

```rust
// 1) 先按严格 JSON 解析
// 2) 失败则取文本里第一个 '{' 到最后一个 '}' 的切片重试（容忍模型包了一层散文）
// 3) 仍失败 → 判定为评审失败（不是「高风险」）
let risk_level = payload.risk_level.unwrap_or(match outcome {
    Allow => Low,      // 允许但没给风险 → 低
    Deny  => High,     // 拒绝但没给风险 → 高
});
let user_authorization = payload.user_authorization.unwrap_or(Unknown);
let rationale = payload.rationale
    .filter(|r| !r.trim().is_empty())
    .unwrap_or_else(|| match outcome {
        Allow => "Auto-review returned a low-risk allow decision.",
        Deny  => "Auto-review returned a deny decision without a rationale.",
    });
```

三条纪律值得直接抄：
1. **拒绝但未给风险 → 按高风险处理**（保守方向）；
2. **拒绝但未给理由 → 填一句可读兜底**，不让 UI 出现空白理由；
3. **非 JSON 不等于高风险**——它是**评审失败**，走另一条处理路径（见 §四）。

## 三、超时：一个共享截止时间，但取消始终有效

```rust
pub enum ReviewDeadline { At(Instant), Unlimited }
```

- `from_timeout(Option<Duration>)`：给了超时就是绝对截止时间，没给就是 `Unlimited`；
- **`Unlimited` 不等于「不可取消」**：注释明确「Unlimited reviews still honor cancellation at every stage」，
  每个阶段都检查取消令牌；
- 截止时间在**会话启动与上下文准备阶段**就生效，不是只包住最后一次模型调用。

## 四、失败分类（错误绝不冒充风险）

```rust
pub enum GuardianReviewOutcome { CachedApproval, Completed(GuardianAssessment), Error(GuardianReviewError) }
pub enum GuardianReviewError {
    InputBudgetExceeded,          // 输入预算炸了
    PromptBuild { message },      // 提示词拼装失败
    Session { message, error_info, retry_at },  // 会话层失败（可带重试时间）
    Parse { message },            // 输出不是合法 JSON
    Timeout,                      // 超时
}
```

模块注释一句话点题：**"Distinguishes completed assessments from failures without assigning risk to errors."**
——失败就是失败，不给它安一个风险等级，也不因此拒绝动作，而是**回落到用户流程**（`None`）。

## 五、熔断：拒绝太多就中断回合，而不是无限重试

```rust
const MAX_CONSECUTIVE_CYBER_GUARDIAN_DENIALS_PER_TURN: u32 = 1;
const MAX_CONSECUTIVE_GUARDIAN_DENIALS_PER_TURN: u32 = 3;
const MAX_RECENT_CYBER_AUTO_REVIEW_DENIALS_PER_TURN: u32 = 1;
const MAX_RECENT_AUTO_REVIEW_DENIALS_PER_TURN: u32 = 10;
const AUTO_REVIEW_DENIAL_WINDOW_SIZE: usize = 50;
```

- 按**回合**计数：连续拒绝 3 次（高危/网安模型 1 次）→ 触发中断；
- 另有滑动窗口（最近 50 次里拒绝 ≥10 次 / 高危模型 ≥1 次）→ 提示降级；
- **中断由宿主执行**（注释："the host carries out any requested interruption"），
  评审器只负责计数与请求，不自己动手——职责清晰。

## 六、预算三层（前一篇已提，这里给施工口径）

1. **输入预算**（`core/src/guardian/input_budget.rs`）：按字节/token 记账，超限先压缩再判定，
   记账跨所有裁剪阶段（`original_bytes` / `retained_bytes` 分别累计）；
2. **请求预算**（`request_budget.rs`）：`INPUT_TOKEN_MARGIN = 256`；
3. **会话预算**（`review_session.rs`）：评审会话自己的 token 用量增量（`token_usage_delta`）单独统计。

---

## 七、gaea 施工方案（三刀，可分别验收）

### 刀一：骨架 + 契约（1–2 天，不花钱）

1. `internal/gaea/guardian` 新包：`Assessment{RiskLevel, UserAuthorization, Outcome, Rationale}` +
   `Parse(text) (Assessment, error)`（照 §二的三层兜底与默认值）；
2. JSON Schema 常量与提示词模板**分开存放**（注释要求两者对齐），模板文案自己写；
3. 事件侧：把 `event.ApprovalAuto` 扩成带 `risk/authorization/rationale` 三字段的账本。

**验收**：喂 6 条样例文本（严格 JSON / 散文包裹 / 缺 risk / 缺 rationale / 空字符串 / 非 JSON），
前 4 条得到预期默认值，后 2 条返回错误而不是 `allow`。

### 刀二：评审调用 + 预算（2–3 天）

4. 用 `internal/ai` 起一次**独立会话**（不要复用主会话上下文），system = 策略模板，user = 证据包；
5. 输出用 JSON Schema 约束（若后端支持 `response_format`，否则靠提示词 + 刀一解析兜底）；
6. 加 `ReviewDeadline`（默认给一个上限，例如 20 s；`Unlimited` 也要能取消）；
7. 预算：输入按字节估算 → 超限先裁证据（记录裁了多少）→ 仍超则 `InputBudgetExceeded`。

**验收**：模拟超时 → 返回 `Timeout` 且**不产生拒绝账本**；模拟预算超限 → 同上；
正常路径下台账里能看到 `risk/authorization/rationale` 三字段。

### 刀三：接线 + 熔断 + 追认（2 天）

8. 决策接线：`Allow → 放行`、`Deny → Denied{rejection: rationale}`、`Error|None → 回落用户流程`；
9. 熔断：连续拒绝 3 次（高危场景 1 次）→ 请求中断当前回合（由现有 controller 执行，不由 guardian 动手）；
10. 追认：`ApproveDenied(action_id)` → 按原参数重放一次，走正常闸门。

**验收**：连续 3 次 deny 后回合被中断一次且事件可查；追认后动作确实被执行一次；
`None` 路径下用户看到审批卡而不是自动放行（这条是红线）。

---

## 八、不建议照搬

- **`guardian-v2` 的 25 个文件**（分类器 25.9 KB / 采样器 17.6 KB / 观测 17.9 KB）：那是为大规模线上评审做的
  采样与统计体系，单机应用用不上，先做「一次同步评审」即可；
- **`CachedApproval`**（异步证据竞速）：需要长期在线的评审缓存，收益依赖高频重复动作，先不做；
- **`routing.rs` 的模型路由**：gaea 是本地单模型为主，路由价值低。

## 九、一句话

Guardian 的施工难点不在提示词，而在**四条边界**：输出容错（拒绝但缺字段怎么补）、
失败与风险分家（错误不许冒充高风险）、超时可取消、拒绝计数熔断。
把这四条写进代码，剩下的提示词是产品文案问题。
