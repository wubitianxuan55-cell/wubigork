# asset-49 · 回合状态机：step 快照、受限更新、输入队列与重放（B4 · 施工级）

> 证据：`codex-rs/core/src/session/{turn.rs, step_activation.rs, step_settings.rs, input_queue.rs,
> turn_input.rs, token_budget.rs, rollout_reconstruction.rs, async_compaction.rs}`（`rust-v0.165.0`）。
> gaea 侧只读对照：`internal/gaea/control/controller.go`（`Steer`/`SetPermission`/`turnLoop`）、`internal/gaea/agent/*`。
> 只出机制与施工卡，不含代码改动。

---

## 1. 一句话结论

codex 的"回合"不是一个函数，而是**三层结构**：
① `run_turn` 拥有 **step 0**（种子上下文 + 第一次采样），失败时**先把输入落库再返回**；
② 每个 step 读一份**不可变设置快照**，运行中改设置走"受限更新"——预检在锁外、复检与落盘在锁内，
且**审批政策/Guardian 授权面/模型分类不允许中途变**；
③ 输入分**回合内待处理**与**会话级邮箱**两条通道，`steer` 用**取消当前采样**实现插话，唤醒做到"先订阅再检查"。
gaea 已有 `Steer`（`controller.go:790`）与回合队列，缺的是 **step 快照 + 受限更新的安全边界**与**重放选择规则**。

---

## 2. 机制地图

| 关注点 | 文件 | 关键行 |
|---|---|---|
| 回合入口 / step 0 | `turn.rs` | 173–298（`run_turn`）；950 `run_hooks_and_record_inputs`；2654 `try_run_sampling_request` |
| 压缩触发点 | `turn.rs` | 1394 `run_pre_sampling_compact`；1428 `comp_hash_changed`；1438 `maybe_run_previous_model_inline_compact`；1529 `run_auto_compact` |
| 受限更新 | `step_activation.rs` | 27 `check_legacy_turn_safety`；87 `check_legacy_model_safety`；300–385 应用路径；387–423 `prepare_step_settings_activation`；428–448 `validate_active_step_settings` |
| 设置模型 | `step_settings.rs` | `ResolvedStepSettings` / `StepSettingsConstraints` / `StepSettingsUpdate` / `apply_update` / `revalidate` |
| 输入队列 | `input_queue.rs` | 36 `TurnInput`；83 `InputQueueActivity`；207–243 mailbox；245–277 drain + start_options 继承；294–315 `watch_user_input`；317–325 `clear_pending`；327–345 `defer_mailbox_delivery_to_next_turn` |
| 预算提醒 | `token_budget.rs` | `resolve_token_budget` / `apply_model_defaults` / `maybe_record`（claim 一次性） |
| 重放 | `rollout_reconstruction.rs` | 15–30 `RolloutReconstruction`；39–50 `TurnReferenceContextItem`；66 `select_input_compaction` |

---

## 3. 回合入口：step 0 的归属

`run_turn(sess, turn_context, input, …)` 的前 120 行是**失败也要保住输入的流水线**：

1. `auto_compaction_enabled = 扩展未禁用`；`auto_summarization_enabled = 前者 && 未开 TokenBudget`（两者互斥）。
2. Guardian 基本会话先 `check_pending_guardian_input`（有未决输入直接拒）。
3. `drain_async_hook_results(before_user_prompt=true)`：**上一回合迟到的 hook 结果在本回合用户提示之前落**。
4. `prewarmed_client_session` 优先复用（预热连接），否则新建。
5. `run_pre_sampling_compact` 失败时：
   - 先 `run_hooks_and_record_inputs(...)` **把本轮输入记下来**（否则这次输入就丢了）；
   - `TurnAborted` / `ToolCollision` 直接上抛；
   - 其余：`emit_turn_error_lifecycle` → `EventMsg::Error`（远程压缩时前缀文案不同）→ 返回 `Ok(None)`（**回合结束但不是错误**）。
6. 收集 `required_mcp_servers_for_input` + 显式提及的插件 → `capture_step_context_with_required_mcp_servers` 拿 **first step context**。
   —— **step 0 由 `run_turn` 自己拥有**，之后每步由工具循环再取 step context。

**为什么值得抄**：它把"输入不可丢"放在所有失败分支之前，且把"压缩失败"和"回合失败"分成两种结局。

---

## 4. step 快照与受限更新（本篇最值钱的一段）

### 4.1 三种结构

- `ResolvedStepSettings`：解析后的**不可变**设置（模型、审批政策、协作模式、环境等）。
- `TurnContext.next_step_settings: Arc<ArcSwap<…>>`：**下一步**要用的设置——更新只写这里，
  当前步继续用自己那份，所以"改设置"永远不会撕裂正在跑的一步。
- `StepSettingsConstraints`：更新时必须满足的约束（`requirements`、`guardian_approval_enabled`、
  `trusted_guardian_reviewer`、`has_full_disk_write_access`）。

### 4.2 更新路径（顺序很关键）

```
① 预检（可能在锁外，含模型目录查询）  prepare_step_settings_activation → apply_update(...)
② 目标确认（锁 active_turn）           done/turn_context 必须 ptr_eq、token 未 cancel → 否则 TargetUnavailable
③ 复检（锁 state）                     环境校验 → validate_active_step_settings → （换模型时）check_legacy_turn_safety
④ 落盘                                 next_step_settings.store(Arc::new(settings))；环境列表随后安装
⑤ 结果                                 Applied / Rejected{reason} / TargetUnavailable
```

四个"为什么不那样做"：

- **目标确认用 `Arc::ptr_eq` 而不是 ID 比较**：注释写明"a later task may reuse the same context and turn ID"，
  同 ID 的新任务不能被旧更新命中（`done` 是每任务新建的完成信号，`turn_context` 是每任务新建的 Arc）。
- **复检放在锁内、模型查询放在锁外**：避免持锁做 IO；但复检必须看到"最新的 managed requirements"，
  因为 `requirements` 可能在模型查询期间变化。
- **先存 `next_step_settings` 再装环境**：注释写明"下一步在我们放锁前读不到两者"，保证原子观感。
- **更新失败不重试**（`TurnSettingsUpdateOutcome::Rejected` 带原因直接回给调用方），
  不做"降级应用"，避免出现"部分生效"的设置。

### 4.3 哪些东西**不许**中途变（legacy 安全项）

`check_legacy_turn_safety` + `check_legacy_model_safety` 的拒绝清单（每条都有独立错误串）：

| 禁止变化的维度 | 理由（注释原意） |
|---|---|
| 审批政策（含 constrained 形态） | 审批与 Guardian 仍读"准入时"的 `TurnContext` |
| 模型是否"要求自动评审"（required review） | 变更等于改审批权威 |
| prefix 规则是否对 Cyber 模型忽略 | 命令审批仍用 `TurnContext::allow_prefix_rules` |
| fallback 模型元数据（准入或目的任一） | 只有兜底元数据时不可判定安全 |
| Cyber 分类（Guardian 熔断策略来源） | 会改 Guardian 拒绝策略 |
| `computer_use_review_required`（node REPL 评审） | `TurnMetadataState` 钉住两个 REPL 标志 |
| `guardian` 覆盖范围、`node_repl_disabled` | Guardian 覆盖面/可用性 |
| `auto_review_model_override`（显式评审模型） | 换评审模型等于换授权判断者 |
| Guardian V2 分类设置（含 transcript 默认归一） | V2 分类器设置 |
| 父级回退的 node REPL 策略 | 评审器不可用时的回退路径 |

其余（如普通模型元数据、诊断字段）**允许不同**——注释说明是为了"让未迁移的消费者暴露出来"。

---

## 5. 输入队列：两条通道

### 5.1 结构

```rust
enum TurnInput { UserInput{content, client_id, metadata}, FunctionCallOutput(..),
                 ResponseItem(..), InterAgentCommunication(PendingAgentMessage) }
struct UserInputMetadata { acceptance_order: Option<u64>, origin: UserInputOrigin }
struct TurnInputQueue { items: Vec<TurnInput>, has_pending_injection: bool }   // 回合内
struct InputQueue { activity_tx: watch, controller: Option<(ThreadId, Arc<dyn AgentControl>, watch)>, mailbox_pending_mails: Mutex<VecDeque<..>> } // 会话级
```

- `activity_tx` 是 **`watch<InputQueueActivity>`**（`Mailbox | Steer`）——只广播"有新东西"，不携带内容。
- 序列化边界有硬约束：**带 annotation 的 `ResponseItemEnvelope` 不允许跨回合输入边界**
  （序列化直接 `Err("annotated response items cannot cross …")`），保证线上格式不变。

### 5.2 三条时序纪律

1. **邮箱优先**：`enqueue_mailbox_communication` 先 `read_mailbox`（把"未加载期间的邮件"读进来）再 push，
   注释：*"Mail retained while unloaded precedes new submissions to the loaded session."*
2. **先订阅再检查**：`watch_user_input` 先 `activity_tx.subscribe()`，再查 `pending_input.has_user_input()`——
   否则"检查完、订阅前"到达的消息永远不会唤醒。
3. **插话 = 取消当前采样**：命中待处理用户输入就 `interrupt.cancel()`；采样侧收到取消后进入下一步，
   于是"插话"表现为**当前采样被中断 + 下一步带上新指引**，而不是并发写上下文。
   `AbortOnDropHandle` 保证等待任务随 step 结束自动撤销。

### 5.3 阈值：什么时候延到下一回合

`defer_mailbox_delivery_to_next_turn`：只要队列里存在**非"仅入队型"子代理邮件**（`trigger_turn == true`），
就**不延迟**（本回合必须再采一次）；只有"仅入队型"子代理邮件时才保留到下一回合，
由任务完成记录带走，避免白白多跑一次采样。

`drain_mailbox_input_items` 还规定了继承：`start_options` 取**最后一条** `trigger_turn` 邮件的选项，
`parent_turn_id` 要求所有触发邮件**一致**才保留（否则清空），`root_turn_id` 只在父子都非空时保留。

---

## 6. 预算提醒与兜底压缩（claim 一次性）

`maybe_record(sess, turn_context, base_window_tokens_remaining, allow_auto_compact_fallback)`：

- 特性未开 / 没有剩余量 → 直接返回。
- `剩余 ≤ reminder_threshold_tokens` 且 `state.claim_token_budget_reminder()` 返回 true →
  注入 `TokenBudgetReminder`（模板来自配置）。**claim 保证每窗口只提醒一次**。
- `剩余 == 0` 且允许兜底 且 `claim_auto_compact_fallback()` 为 true →
  注入 `AutoCompactFallbackPrompt`（同样是每窗口一次）。
- 阈值来源优先级：**显式配置 > 模型自带默认**；模型默认 `validate()` 失败只 `warn` 并回退到配置值。
- `apply_model_defaults` 只在**没有任何显式 token_budget 配置**时才播种，且注释要求"保持提示词未解析"，
  以便后续换模型时用新模型自己的默认。

---

## 7. 重放选择：从 rollout 重建历史

`reconstruct_history_from_rollout` 返回 `RolloutReconstruction`：`history` + `retained_context` +
`guardian_history` + `last_started_turn_id` + `previous_turn_settings` + `reference_context_item` +
`turn_attribution` + `world_state_baseline` + 窗口三元组（`number/first_window_id/previous_window_id/window_id`）。

两条关键规则：

1. **只有"最新且可用"的压缩点能限定重放**（`select_input_compaction`）：
   最新压缩若**不完整**，**不能**用更旧的压缩替代——因为更新的那次可能已经改写了历史或窗口状态。
   没有可用压缩点就**全量重放**。
2. **`TurnReferenceContextItem` 三态**：`NeverSet`（从未建立基线）/`Cleared`（建立过、但被后续压缩作废）/
   `Latest(..)`。只有 `Cleared` 必须在 resume/fork 时**显式发出清除段**；`NeverSet` 不许凭空造一段清除。

---

## 8. gaea 对照（只读实测）

| 维度 | codex | gaea | 差在哪 |
|---|---|---|---|
| 回合结构 | `run_turn` 拥有 step 0，工具循环再取 step | `runGuarded`/`turnLoop`（`controller.go:404–466`）+ agent 内部循环 | 结构近似；差别在"step 边界"没有显式类型 |
| 中途改设置 | 快照 + 受限更新 + 十项禁止变化 + 三态结果 | `SetPermission`（`controller.go:935`）直接写 `c.permLevel`；模型切换无守卫 | **没有"准入快照"概念**：跑到一半改权限/模型，行为跟着变，且没有拒绝理由可回 |
| 插话 | `watch_user_input` → 取消采样 → 下一步带指引 | `Steer`（`controller.go:790`）：`SteerAccept` 失败**降级成 Send**，绝不丢 | 语义接近；gaea 缺"命中即取消当前采样"的显式信号与"先订阅再检查"的防漏 |
| 队列分层 | 回合内 `pending_input` + 会话级 mailbox + `start_options` 继承规则 | `pendingSends` 单层 FIFO（上限 8） | 无"未加载期间的邮件优先"、无父子回合 id 继承与一致性校验 |
| 预算提醒 | claim 一次性（提醒/兜底各一次），阈值显式配置优先于模型默认 | `maybeRecallReminder` 等自研提醒，无 claim 去重口径 | 会重复提醒；无"模型默认 vs 显式配置"的优先级规则 |
| 重放 | 最新可用压缩点 + 不完整则全量 + 三态清除段 | `Restore`（checkpoint+tail） | 无"压缩点完整性"判据，restore 后可能丢窗口状态 |

---

## 9. 施工卡（三刀；默认关闭）

### 刀 1 · step 快照与受限更新（约 1 天）

- 新增 `internal/gaea/control/stepsnapshot.go`：`type StepSettings struct { Model, ApprovalPolicy, PrefixRules, GuardianCoverage … }`（不可变值）；
  会话持 `nextStep atomic.Pointer[StepSettings]`，每步开始 `Load()`。
- `UpdateStepSettings(update) (Applied|Rejected{reason}|TargetUnavailable)`：
  预检（可做模型查询）→ 目标确认（回合句柄 + 回合 token 同一性 + 未取消）→ 复检（锁内）→ 落盘 → 返回。
- **禁止变化清单**照抄十项（审批政策、required review、prefix 规则、fallback 元数据、Cyber 分类、
  node REPL 评审、guardian 覆盖、评审模型 override、V2 分类、父级回退策略）；错误串按自己文案写。
- 禁区：不改现有 `SetPermission` 的**即时生效**语义（那是产品选择），只新增"step 内不允许变"的维度。

### 刀 2 · 输入队列分层（约半天）

- 回合内 `pendingInput`（四类输入 + metadata）+ 会话级 `mailbox`（未加载期间邮件优先）。
- 唤醒：`watch` 通道 + **先订阅再检查**；`Steer` 命中即 `cancel()` 当前采样，失败降级 `Send`（保留现有语义）。
- 继承：`startOptions` 取最后一条触发邮件，`parentTurnID` 不一致则清空——**一致性校验**是重点。
- 禁区：不改 `pendingSends` 的 8 条上限与既有事件序列。

### 刀 3 · 重放与一次性提醒（约半天）

- 重放：只在"最新压缩点完整"时用它限定重放，否则全量；三态清除段（NeverSet / Cleared / Latest）。
- 提醒：`claimTokenBudgetReminder` / `claimAutoCompactFallback` 每窗口各一次；阈值优先级"显式配置 > 模型默认"，
  模型默认非法时只 warn 回退。

---

## 10. 验收断言

| # | 断言 |
|---|---|
| S1 | 压缩失败但非 TurnAborted：本轮输入**已落库**，事件序列为 `Error`，回合以 `Ok(None)` 结束 |
| S2 | 中途改审批政策 → `Rejected`（原因含"approval policy"），当前 step 继续用旧快照，下一步也不变 |
| S3 | 同 ID 新回合：旧更新作用在旧句柄上 → `TargetUnavailable`，新回合不受影响（`ptr_eq` 语义） |
| S4 | 更新应用后：下一步读到新设置，**当前步**仍读旧设置（快照不撕裂） |
| S5 | 先订阅再检查：在"检查后、等待前"投递消息，等待者必须被唤醒（不丢唤醒） |
| S6 | 插话命中：当前采样被取消，下一步输入含该指引；`SteerAccept` 失败时降级 `Send` 且消息不丢 |
| S7 | 只含"仅入队型"子代理邮件的回合：不额外采样，邮件留到下一回合 |
| S8 | 提醒/兜底压缩：同窗口各只注入一次（第二次 claim 返回 false） |
| S9 | 最新压缩点不完整：不采用更旧压缩点，走全量重放；`NeverSet` 不产生清除段 |

---

## 11. 不搬 / 风险

- **不搬**：`ArcSwap`/`AbortOnDropHandle` 的具体 API（Go 用 `atomic.Pointer` + `context` 表达）。
- **不搬**：`TurnReferenceContextItem`/窗口 UUID 三元组（gaea 用 SQLite 会话 + checkpoint，窗口概念由会话行承担）。
- **只借语义**：`InterAgentCommunication.trigger_turn`（对应 gaea 后台任务"完成注记是否触发新回合"）。
- **风险**：① 十项禁止清单容易照抄成"什么都不能改"，需按 gaea 实际消费者裁剪；
  ② 快照化后 `SetPermission` 的即时生效与"step 内不可变"冲突——必须先拍板"权限是否属于快照维度"（建议：**属于**，
  即时生效只影响下一次审批，不影响已准入的审批权威）；③ claim 状态要跟窗口/会话复活一起重置，否则重启后不再提醒。
