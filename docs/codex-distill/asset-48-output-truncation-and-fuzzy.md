# asset-48 · 输出截断口径与模糊打分（B12 · 施工级）

> 证据：`codex-rs/utils/output-truncation/src/lib.rs`、`utils/string/src/truncate.rs`、
> `utils/fuzzy-match/src/lib.rs`、`protocol/src/protocol.rs:3535–3583`（`TruncationPolicy`）。
> gaea 侧只读对照：`internal/gaea/agent/agent_helpers.go:33`、`agent/compact_summary.go:191`、
> `control/controller.go:717`、`agent/archive.go:236`。

---

## 1. 一句话结论

codex 只有**一套**截断模型：`TruncationPolicy = Bytes(n) | Tokens(n)`，预算**头尾各一半**，
marker 是否计入预算有**两种明确语义**（`truncate_middle_chars` 不计 / `truncate_middle_with_marker` 计），
token 与 byte 用 **4 bytes ≈ 1 token** 互折；另有 **1.2 序列化余量**（且**不许重复乘**）。
gaea 现在有 **4 套互不相干的截断**（`truncateStr`/`truncateText`/`truncate`/`truncateSummary`），
口径不一（字节 vs rune）、只留头不留尾、无预算互折、无余量概念——且 `truncateStr` 按**字节**切会切碎 UTF-8。

---

## 2. 预算模型（先记这四个事实）

| 事实 | 值 / 位置 | 影响 |
|---|---|---|
| 近似换算 | `APPROX_BYTES_PER_TOKEN = 4`；`tokens = ceil(bytes/4)`；`bytes = tokens*4`（saturating） | 前后两向都可算，且都是**上取整** |
| 头尾分配 | `left = budget/2`，`right = budget - left` | 奇数时**尾部多 1 字节** |
| 两种 marker 语义 | `truncate_middle_chars`：预算只算保留段，**marker 额外**；`truncate_middle_with_marker`：**含 marker**，marker 都放不下就退化成"最长前缀" | 选错会导致"结果仍超限" |
| 序列化余量 | `with_serialization_allowance(p) = p * 1.2`（`ceil`） | 注释明确：**已含余量的历史预算不许再乘** |

`TruncationPolicy::token_budget()` / `byte_budget()` 互为换算入口——**不要在各调用点手写 `/4`**。

---

## 3. 对外形态

### 3.1 文本

```rust
formatted_truncate_text(content, policy)   // 超限时前置两行：
// Warning: truncated output (original token count: N)
// Total output lines: M
// <截断后的正文>
```
- 未超 `policy.byte_budget()` 时**原样返回**（零开销、零改写）。
- 两行 header 里的 token 数是**原始**值（用 `approx_token_count`），不是截断后。

### 3.2 MCP 工具结果（最容易踩坑的一处）

`truncate_mcp_tool_result(result, max_bytes)`：
1. 先整体 `serde_json::to_string`，未超限直接 `Cow::Borrowed`（不复制）。
2. 超限就把 `content` 换成一个 `text` 预览，保留 `is_error`（**错误状态不许丢**），`structured_content` 清空。
3. **迭代缩放**：因为预览会被 JSON 转义（引号/换行膨胀），按"整包实际长度"反推下一次预算：
   `next = (budget * max_bytes / truncated_len).min(budget - 1)`。
   `.min(budget-1)` 保证**严格下降**，否则会出现不收敛的死循环。
4. 终止条件：`truncated_len <= max_bytes` 或 `budget == 0`。

### 3.3 内容项数组（多模态）

`truncate_function_output_items_with_policy(items, policy, estimate_audio_token_count)`：
- 预算从 `byte_budget()`/`token_budget()` 起，**按顺序**消耗。
- 空 `InputText` **直接跳过**：不占预算，但仍会占 API 数组槽位（注释写明）。
- 一条超预算：截断到剩余预算，然后把剩余置 0；截断后为空则计入 omitted。
- **音频**先由回调估 token，再按 policy 折算成字节；超预算就整条丢并计入 omitted。
- **图片 / 加密内容不截断，直通**（顺序保留在文本之后）。
- 收尾追加 `[omitted N text items ...]` / `[omitted N audio items ...]` —— **省略要可见**，不许静默丢。
- 另有合并版：多段文本先拼成一段整体截断（返回 `Some(原始 token 数)`），图片等再按序追加。

---

## 4. 模糊打分（`fuzzy-match`）

```
fuzzy_match(haystack, needle) -> Option<(Vec<usize> 原字符下标, i32 分数)>
```

- 匹配是**子序列**（不要求连续），大小写不敏感。
- **索引映射回原串**：先把 haystack 逐字符 `to_lowercase()` 展开，并维护 `lowered → 原字符下标`；
  这样 `İ`（小写展开成 `i` + 组合点）也能给出正确的**高亮位置**（多对一 → `sort + dedup`）。
- 打分：`window = (末位 - 首位 + 1) - needle_len`，`score = max(window, 0)`；
  **首位在 0 时再减 100**（强前缀奖励）。越小越好。
- 空 needle：`Some(([], i32::MAX))`——"匹配但排最后"，不是 `None`。
- 已知边界（测试里明确写了）：这是**小写化**而非完整 casefold，
  `straße` 不匹配 `strasse`（`ß` 不会展开成 `ss`）；要做完整 casefold 得换实现。
- 复杂度 O(n)：`cur` 只前进不回退，适合每次按键都跑的弹层过滤。

---

## 5. gaea 对照（只读实测）

| 位置 | 现状 | 问题 |
|---|---|---|
| `agent/agent_helpers.go:33 truncateStr` | `s[:maxLen]`（**字节**） | 会切碎多字节字符产生非法 UTF-8；用于 dedup key，风险低但口径错 |
| `agent/compact_summary.go:191 truncateText` | 按 rune 头截 + `"…"` | **只留头**，尾部信息全丢；marker 不占预算；无 token 概念 |
| `control/controller.go:717 truncateSummary` | 240 rune 硬编码 | 常量散落，无策略对象 |
| `agent/archive.go:236 truncate` | 另一套 max | 第四套口径 |
| 多处 omitted 提示 | `compress.go:392`、`interrupted_recovery.go:199/211`、`recall_archived.go:149` | 各写各的文案与计数口径，**没有统一"省略必须可见"的契约** |

**差距三条**：① 没有统一的 `Policy` 抽象（Bytes/Tokens + 1.2 余量）；② 没有头尾保留（长输出尾部常有错误行/结论行，
只留头恰好吃掉最关键的部分）；③ 没有"截断结果是否仍超限"的硬保证（`truncateText` 的结果是 `maxChars + 1` rune）。

---

## 6. 施工卡（两刀，Go 侧；默认关闭）

### 刀 1 · 统一截断策略（约半天）

- 新增 `internal/gaea/strutil/truncate.go`：`type Policy struct{ Mode Mode; Limit int }`（`ModeBytes|ModeTokens`），
  方法 `ByteBudget()`、`TokenBudget()`、`WithSerializationAllowance()`（×1.2，`math.Ceil`）。
- `TruncateMiddle(s, policy)`：**按 rune 边界**做头尾 50/50；两种入口 `TruncateMiddleChars`（marker 额外）
  与 `TruncateMiddleWithMarker`（marker 计入，放不下退化前缀）。
- 把四处既有实现改为**薄封装**（保留函数名，行为迁移），先加断言再改调用点。
- 禁区：不改 `dedup key` 的**语义稳定性**（键里若原本是字节切，迁移要一次性重建缓存或保持旧路径可回退）。

### 刀 2 · 省略可见 + 模态豁免（约半天）

- 统一 omitted 文案契约：`[omitted N <kind> items ...]`、`…N tokens truncated…`、`…N chars truncated…`，
  由 `strutil` 提供唯一实现。
- 图片/附件类内容**不截断**，只做"数量上限 + 省略提示"。
- 禁区：不动现有 SSE 事件类型；`TurnDone`/`Notice` 等文案自写（不抄上游原文）。

---

## 7. 验收断言

| # | 断言 |
|---|---|
| T1 | `Policy{Bytes,1000}` 与 `Policy{Tokens,250}` 的 `ByteBudget()` 相等（4:1 互折） |
| T2 | `TruncateMiddle` 对 1001 字节 ASCII 输入：结果头 500 + marker + 尾 500（尾部多 1 的奇数分配要单独测） |
| T3 | 含中文/emoji 的输入：输出必须是合法 UTF-8，且头尾都在 rune 边界上 |
| T4 | `TruncateMiddleWithMarker` 结果长度 **≤ 预算**；marker 比预算还长时返回最长前缀 |
| T5 | `WithSerializationAllowance` 只乘一次：对已含余量的预算再调，断言不重复乘（用注释/常量守护） |
| T6 | 模糊打分：`abc` vs `a-b-c` 分数严格更差；首位命中 -100；空 needle 返回 `i32::MAX` 而非不匹配；`İstanbul` 的 `<is>` 命中下标为 `[0,1]` |
| T7 | 省略提示与计数一致：丢 3 条文本则提示 `3`，且总数可被上层复算 |

---

## 8. 不搬

- `Cow`/借用优化（Rust 特有）——Go 侧按值传递即可。
- 音频 token 估算回调（gaea 语音链路自带估算，不引入第二套）。
- 完整 casefold（需要 ICU；先用小写化并在文档里写明 `ß` 边界）。
