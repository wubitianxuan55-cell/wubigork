# 资产 09 · 压缩施工级：三种形态、四种触发、新窗口到底保留什么

> 源码：`core/src/compact_token_budget.rs`（69 行，全文已读）、`core/src/session/mod.rs::start_new_context_window`、
> `core/src/compact_remote_v2.rs`（保留预算常量）、`core/src/session/token_budget.rs`、`rollout_budget.rs`。
> 对照：gaea `internal/gaea/agent/compact.go`（40 KB，已有三入口 + 规划器 + `CompactionStarted/Done` + `PreCompact`）。

---

## 一、四种触发原因（比 gaea 现在多两种）

```rust
pub enum CompactionTrigger { Manual, Auto }
pub enum CompactionReason {
    UserRequested,     // 用户手按
    ContextLimit,      // 额度逼近窗口
    ModelDownshift,    // 切到更小的模型 → 先把历史压小再交过去
    CompHashChanged,   // 模型指纹变了（comp_hash）→ 旧口径的历史不再适用
}
```

**gaea 现在只有「用户手按」和「额度逼近」两种语义**。缺的两条都跟「换模型」有关：
- `ModelDownshift`：从 500k 窗口的模型切到 128k，历史必须**先压再交**，否则一进去就超限；
- `CompHashChanged`：模型一变，之前按旧模型口径做的压缩摘要就不可信，要重新压一次。

这两条正好对应 gaea「切模型是直接把消息数组带过去」的现状——`ModelDownshift` 是修补它的正确姿势。

## 二、三种形态

| 形态 | 做什么 | 产物 |
|---|---|---|
| **local** | 本进程把历史摘要成新窗口 | 摘要文本 |
| **remote** | 交服务端摘要（`compact_remote_v2*`） | 服务端返回的摘要 + 保留项 |
| **token-budget** | **不做任何摘要**，直接装一个新窗口 | 新窗口 = 初始上下文 + 保留项 + 回合上下文 |

`compact_token_budget.rs` 只有 69 行，核心就一句：**跳过模型/服务端摘要，装一个新上下文窗口**，
但仍然走完整的 compaction 生命周期（下面 §四）。

## 三、新窗口到底保留什么（最容易写错的地方）

`Session::start_new_context_window` 的真实步骤：

```
1) clone_history()                                    // 拿当前历史
2) input_goal_ids = UserGoalUpdate::message_ids(history)   // ★ 继承「用户目标」的消息 id
3) retained_client_items = 过滤历史，只留两类：
      · 保留型工具输出（Feature::RetainClientToolOutputs）
      · 客户端作者写的 developer 消息（Feature::RetainClientDeveloperMessages）
   并按 RETAINED_MESSAGE_TOKEN_BUDGET 截断
4) state.start_new_context_window() → (window_number, window_ids)
5) build_initial_context_with_world_state(...)         // ★ 重建初始上下文（含世界状态）
6) merge_world_state_updates + chain(retained_client_items)
7) replace_compacted_history(context_items, turn_context_item,
      world_state_snapshot,
      CompactedHistoryMetadata { input_goal_ids, message: "", window_number, window_ids, .. })
```

关键常量与字段：

```rust
pub(crate) const RETAINED_MESSAGE_TOKEN_BUDGET: usize = 64_000;

struct CompactedHistoryMetadata {
    input_goal_ids: HashSet<ResponseItemId>,   // 用户目标不因换窗口而丢失
    message: String,                            // token-budget 形态下为空串（没有摘要）
    window_number: u64,                         // 窗口序号递增
    window_ids: AutoCompactWindowIds,
    compaction_response_id: Option<String>,
    compaction_model_hash: Option<String>,
    reviewer_compaction_hash: Option<String>,
}
```

**四条施工要点**

1. **不是「清空历史」**，是「重建初始上下文 + 有界保留项 + 回合上下文」——
   初始上下文包含 system 与世界状态，所以模型仍有完整人格与项目约定。
2. **保留项是白名单而且有预算**：只留「保留型工具输出」与「客户端 developer 消息」两类，
   总预算 64k token。不要把整段历史原样塞进去（那等于没压缩）。
3. **`input_goal_ids` 必须继承**：用户定的目标如果只存在于被压掉的消息里，
   换窗口后模型会「忘了要干什么」——这是 token-budget 形态最容易踩的坑。
4. **`message` 留空**：token-budget 形态没有摘要文本，UI 不能显示「空摘要卡片」，
   要靠 `CompactionTrigger` 区分「没摘要的压缩」和「摘要失败」。

## 四、生命周期与回退（三种形态共用）

```
run_pre_compact_hooks(trigger)
  ├─ Stopped → Err(TurnAborted)          // ★ 钩子可以否决压缩
  └─ Continue
emit_turn_item_started(ContextCompaction)
[start_new_context_window / 摘要压缩]
emit_turn_item_completed(ContextCompaction)
run_post_compact_hooks(trigger)
  └─ Stopped → Err(TurnAborted)
```

**回退纪律**：pre-compact 被否 → 回合以 `TurnAborted` 结束，**不写压缩结果**；
post-compact 被否 → 压缩已发生但回合中止（记录保留，让用户知道发生了什么）。

## 五、gaea 落点与改造量

`agent/compact.go` 已有：`maybeCompact` / `midTurnMaybeCompact` / `compactIfOver(trigger)` / `planCompaction`
（保留 ≥25% 给近期轮次、钉前 15% 用户窗口）/ `tryOverflowRecovery` / `CompactionStarted|Done` 事件 / `PreCompact` hook。

**要补的只有四件**（都不动现有摘要管线）：

1. 新增 `newWindowCompact(reason)`：重建初始上下文（复用现有 boot 装配）+ 白名单保留项 + 回合上下文；
2. 保留项白名单与预算：先只保留「用户目标消息」与「本轮引用过的文件摘要」，给一个保守预算（不必照抄 64k，
   本地小模型窗口小，按窗口的 30% 取更合适）；
3. 触发原因补 `ModelDownshift` 与 `CompHashChanged`：在切模型处判断——新模型窗口 < 当前用量 → 先压；
   模型 hash 变化且上次压缩摘要来自旧模型 → 重新压；
4. 元数据落库：`window_number` / `input_goal_ids` / 空摘要标记，进 `event` 流供 UI 与回放。

## 六、验收（可写成单测）

1. 触发 token-budget 压缩后，历史里**没有**摘要占位消息，但有 `ContextCompaction` 起止两条事件；
2. `input_goal_ids` 在压缩前后一致（用户目标不丢）；
3. `window_number` 单调递增，回放时能按窗口边界重放；
4. 保留项总 token ≤ 预算，超出时按时间倒序截断（保留最近的）；
5. pre-compact hook 返回停止 → 回合以 `TurnAborted` 结束且**历史未被修改**；
6. 切到更小窗口的模型时自动先压缩一次，且压缩发生在**发起请求之前**（用事件顺序断言）。

## 七、不建议照搬

- **`compact_remote_v2*` 那一家子**（远程摘要 + 图片压缩 + 安装校验）：依赖后端，gaea 本地生成没有对应物；
- **64k 保留预算**：那是给 272k–500k 窗口的模型定的；gaea 跑本地小模型时按窗口比例取，硬抄会撑爆；
- **`reviewer_compaction_hash`**：给 Guardian 评审用的压缩口径指纹，等 Guardian 上线后再考虑。
