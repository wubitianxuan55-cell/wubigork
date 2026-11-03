# Orca × 众神殿 第二轮蒸馏：投递、观测空间与结案纪律

> 对象：Orca 1.4.214 @ `8cd6a7e5ee77c4ed04e78a3e924f5fc83a366bbf`（2026-11-02，MIT）
> 落点：Gaea 众神殿 `internal/pantheon/`
> 第一轮：`docs/gaea-orca-pantheon-distill-2026-11.md`（M1–M8、刀池 O-A…O-G、P0 已落地）
> 读法：本轮不改 `clones/orca` 任何文件；换一种读法——不按产品面看"Orca 有什么"，而按**一条状态从产生到被相信的路径**逐段读代码，看它用什么纪律挡住"看起来完成了"。

---

## 1. 本轮产出什么

第一轮给的是定位与八条机制，这一轮给的是**可实现的判定式**：新鲜度到底几秒、跨进程能不能比序号、什么情况下允许把"失败"改成"完成"、一个 run 里有兄弟在跑时能不能结案、门在重启后怎么不溜过去。

三件事：

1. 收紧第一轮的三处口径（§2），其中两处会把实现带偏；
2. 新增九条机制 M9–M17（§3），每条都有源码位置；
3. 重排刀池为 O-D / O-E / O-H / O-I 四刀，带 Go 形状与验收（§4、§5）。

结论没有变，只是更硬：**Orca 的可搬部分不是界面和调度器，而是"证据→裁决→结案"这条链上的六个硬规矩**。

---

## 2. 第一轮的三处口径修正

M1–M8 全部成立，以下三处需要收紧，否则落地会走偏。

### 2.1 freshness 是两层，不是一个数

| 层 | 常量 | 位置 | 用途 |
|---|---|---|---|
| attempt 观测层 | `DEFAULT_FRESH_AFTER_MS = 60_000` | `src/main/runtime/orchestration/db/attempt-outcome-projection.ts:11` | 这条 live 证据还能不能用来判 attempt 结果 |
| UI 状态点层 | `AGENT_STATUS_STALE_AFTER_MS = 30 * 60 * 1000` | `src/shared/agent-status-freshness.ts:4` | 侧栏那个转圈图标还能亮多久 |
| 水合保留 | `AGENT_STATUS_HYDRATE_MAX_AGE_MS = 7 天` | 同上 `:7` | 重启后非终态行最多留多久 |

观测层超时后不是"变旧"，是**降级为不可判定**：

```ts
// attempt-outcome-projection.ts:74-80
if (freshness.status !== 'fresh') {
  return { status: 'unverifiable', reason: `live observation is ${freshness.status}`, freshness }
}
```

第一轮把两层写成一个"stale 窗口"。抄成一个数就会出两种错：判定窗口太宽 → 拿十分钟前的证据宣布完成；UI 窗口太窄 → 议员刚说一半就被显示成掉线。

### 2.2 排序只在同一个 authority 内成立

`AgentStatusObservation` 的三件套是 `authorityId / incarnation / revision`，但边界写在注释里（`src/shared/agent-status-observation.ts:53-55`）：

> Order incarnation/revision only within one authorityId; restarts/machines/byte parsers are incomparable and use the timestamp fallback.

配套三条：

- `revision` 在 authority 内单调，但**只比较、不计数**（`:33-34`，别的 pane 共用计数器，会留空洞）；
- pane 重新绑定（重启、重连、复用）时 `incarnation` 前进（`:95-99`）；
- `authorityId` 每次进程启动重新生成（`:127-130`）——重启后的计数器不能和重启前比。

第一轮只写了"状态带来源"，漏了"跨来源不可比"。这正是众神殿最容易踩的：应用重启后拿新 `revision` 和旧值比大小，会得出"状态倒退了"的假结论，进而触发一次本不该发生的重试。

### 2.3 租约防进程复用，dispatchId 防迟到报告

单写者租约的裁决头注释（`src/shared/agent-session/lifecycle/agent-session-lease-adjudication.ts:3-7`）：

> Every decision here fails closed: expiry alone never grants a second owner.

PID 命中不够，还要 `process-start-time` 或 `spawn-token`（`:20-24`）：

> A PID match needs reuse-safe start-time/token evidence; otherwise liveness is indeterminate.

但租约**不负责**拒绝迟到的完成报告——那是 `taskId + dispatchId` 双命中的事（`task_dispatch_mismatch` / `stale_dispatch`，CLI 文档也写了"Include both task and dispatch IDs so stale retries cannot complete the wrong dispatch"，`docs/site/content/docs/cli/orchestration.mdx:138`）。第一轮把两条写在一段里，落地时容易以为有租约就不用校验 dispatchId。

---

## 3. 九条新机制

### M9 投递是一条状态机，消息不是

消息落库即事实；**投递是可回滚的动作**。Orca 把它拆成 staged（已暂存）→ flight（在途）→ ack（已确认），失败时回滚：

```
mailbox-pointer-delivery.ts:169   db?.markAsUndelivered(flight.stagedMessageIds)
```

由此得到三条纪律：

1. 未确认的投递会在下次 `check` 重投；`--peek` 不消费、不返回 `deliveryId`（`cli/orchestration.mdx:118`）。
2. 一条回答只被一个消费者取走——取走者立刻标记已读已投，避免再作为信箱邮件投第二遍：
   ```ts
   // question-answer-delivery.ts:88-92
   if (current.answer_message_id) {
     args.runtime.getOrchestrationDb().markAsReadAndDelivered([current.answer_message_id])
   }
   ```
3. **等待者自己退出时不许吞掉答案**（`:79-83`）：调用方已中止，就把答案留作未读并通知 dispatch，让 worker 之后还能收到。

### M10 快照不续命

状态观测分三态（`agent-status-observation.ts:25-26`）：`transition` 断言一次变化、`snapshot` 是重绘/回放、`identity-only` 只刷新身份占位。只有前一类算新证据。

实现上靠"只认第一次本地接收时间戳"（`agent-status-freshness.ts:28-40`）：authority 戳没变时，重复快照沿用旧 receipt，于是**安静 pane 不会被重复快照续命**；水合出来的非终态行 `restoredUnconfirmed` 永远不算新鲜（`:57-58`）。

对应到 Orca 自己的心跳：等待期间 CLI 每 15 秒往 stderr 打一行心跳 JSON（`cli/orchestration.mdx:130`）——那是进度显示，不是证据。

### M11 未知态必须显式，且不许被时钟骗

`attempt-outcome-projection.ts` 里有五个词，各有自己的判定条件：

| 值 | 条件 | 位置 |
|---|---|---|
| `never` | 该 facet 从未有观测 | `:37-39` |
| `unverifiable` | 有观测但时钟缺失/不可比 | `:43-45` |
| `future` | `observedAt - now > 5_000`（`FUTURE_TOLERANCE_MS`） | `:46-48` |
| `fresh` / `stale` | `ageMs <= 60_000` | `:49-55` |

`future` 单独一档，是为了防止跨机时钟漂移被当成"很新鲜"。

### M12 终态图 + 唯一例外

`lifecycle-transition.ts:73-90` 把转移写成显式图。三个 entity 的终态都是自环：

```
dispatch: completed → ['completed'], failed → ['failed'], circuit_broken → ['circuit_broken']
worker:   succeeded → ['succeeded'], failed → ['failed'], stopped → ['stopped'], abandoned → ['abandoned']
```

唯一例外是一个**具名**的纠正通道（`:147-161`），只允许 failed / circuit_broken → completed / succeeded，且注释写明它存在的理由：

> Narrow exception for a worker report correcting an unobserved prompt start.

转移靠**单条带条件的 UPDATE** 落地，`changes !== 1` 就抛 `lifecycle_conflict`（`:176-187`）——并发下不做"先读后写"。

### M13 结案看兄弟，不看单个 dispatch

```ts
// attempt-outcome-projection.ts:144-149
const activeSibling = args.activeSibling ?? false
return { …, taskOutcome: activeSibling && outcome !== 'in_progress' ? 'outcome_unknown' : outcome }
```

一个 run 里还有活着的兄弟，就不能拿其中一个的完成给整个 task 结案。这条直接回答众神殿的老问题：「五个席位四个说完，第五个还在跑，这一轮算不算完？」

### M14 门在重启后要重新压住

`coordinator-decision-gates.ts:51-59`：

> Only explicit human gate resolution may unblock these approval checkpoints.

实现是把所有 pending 的 gate 对应的任务**打回 blocked**，而不是信任它当前的 ready。门的建立也有准入：`consumer_fenced` / `task_not_startable` 的申请被拒（`:38-45`），payload 缺 `taskId` / `dispatchId` / `question` 直接丢弃并留日志（`:21-24`）——**不完整的门不允许存在**，否则它挡不住任何东西。

### M15 陈词是三句式 + 显式 outcome

`preamble.ts:90-95` 的原文，比第一轮的转述更严：

```
# Finish: send exactly once. --body is three sentences: what you did, what
# you found, what is left. Use --outcome failed if the work is not done;
# never hide failure in prose. Add --files-modified or --report-path only
# with real values. You are done when it confirms; if it fails, follow the
# recovery it prints.
```

三条约束值得逐字搬：**三段式**（做了什么 / 发现什么 / 还剩什么）、**失败必须在 outcome 里声明**（不许只写在正文里等读者自己看出来）、**证据字段只填真实值**（宁可空着，也不填占位符）。收声后回 idle，禁止 sleep/poll 循环（`:159-163`），避免那种"看起来还在忙"的假活性。

### M16 释放与归档分离

CLI 文档（`cli/orchestration.mdx:89`）：

> Do not leave completed worker terminals open just to re-read output — use `worker-read` after `worker-release`. Do not substitute a broad `terminal close` when release returns `release_pending` or `release_unknown`; follow the receipt's recovery action.

两条可搬：**先归档可读输出，再关资源**；**回收未完成时不许用大范围操作顶替**，要按回执恢复。对众神殿就是：取消一个席位，取消的是"继续"，不是"已经发生"——已产出的发言和工具轨迹必须留下。

### M17 重试要重声明运行面

`cli/orchestration.mdx:85-86` 的注释："retry placement is explicit — `--retry-of` does not inherit `--on`/worktree"。失败常常正因运行面不对（工具面、目录、后端），静默继承会把同一个错误再跑一遍，还会让"重试"看起来像"重放"。

---

## 4. 映射到众神殿

### 4.1 类型：状态要带出处

现在 `Participant.Status` 是自由字符串（代码里实际有 `waiting / running / finished / failed / skipped / retrying / unavailable` 七个取值），没有出处、没有新鲜度。补一个观测结构：

```go
// Observed 记录一条席位状态是被谁、在什么序号上、什么时候看到的。
type Observed struct {
	Origin      string // adapter|launcher|process|event|user|recovery
	AuthorityID string // 每次应用启动重生成；禁止跨进程比较
	Incarnation int64  // 同一席位重新绑定（重试、换后端、换图片通道）+1
	Revision    int64  // 同一 authority 内单调，只比较不计数
	At          int64  // 证据时间（毫秒）
}
```

`Participant` 加 `Observed Observed` 与 `Evidence string`（一句人能读的"这条状态凭什么"，直接进 UI 悬停）。

状态词表改成受转移表约束的枚举，不再是自由字符串：

| 现态 | 允许到 | 备注 |
|---|---|---|
| `waiting` | `running` / `skipped` / `unavailable` | 开场前可用性检查落这里 |
| `running` | `finished` / `finished_unverified` / `failed` / `needs_input` / `cancelled` / `skipped` | `needs_input` 必须带 QuestionID；`skipped` 是用户中途跳过这一席 |
| `needs_input` | `running` / `skipped` / `cancelled` | 回答到达才回 `running`，否则不许自动续 |
| `retrying` | `running` / `failed` / `skipped` / `cancelled` | 重试期间用户终止，席位要跟着收口 |
| `finished_unverified` | `finished` | 仅同一 attempt 且陈词被接受（M12 的唯一例外） |
| `finished` / `failed` / `skipped` / `unavailable` / `cancelled` | 自身 | 终态自环，幂等 |

表里还有一条不写进转移图的通路：**重试与整场重开是新的 attempt，不是状态转移**。它们由 `rebindSeatLocked` 抬 `Incarnation` 后直接置目标态，绕过转移表——否则"从 `failed` 重试"这条每天都在用的路径会先被自己的表拒绝。这正是 M12 那条具名例外的形状：例外必须显式、具名、单点，而不是把终态改成可回退。

（2026-11-02 落地时对表做了两处增补：`running → skipped` 与 `retrying → cancelled`。两处都是把已有代码路径接进表时暴露出来的，不是从 Orca 抄来的边。）

`Session.Status` 加两个值：`waiting_input`（有未答问题，议事挂起而不是失败）、`interrupted`（重启时正在跑，但没观测到收声）。

### 4.2 结案判定

```
completed ⇔ 每个席位都有终态
           ∧ 没有任何席位处于 running / needs_input
Gaps     = {seats: finished_unverified | unavailable | skipped}
```

`Gaps` 进 `Session`、进结案事件、进 `SignOff` 的哈希范围。这样"四比一决议"不会把那个断线的第五席悄悄算成弃权——它在缺口里，且签注哈希覆盖它。

### 4.3 活性与新鲜度

- `running` 席位超过 **90 秒**没有新证据 → UI 显示"活性未知"，撤销转圈，**但不自动改状态**（自动改会造出假终态）。90s 是我方取值（Orca 判定层是 60s，按席位发言的停顿节奏放宽），它是可调参数，不是抄来的常量。
- 心跳、空闲事件、重绘事件**不刷新** `Observed.At`；只有 delta / tool / output / error 这类新证据才刷新（M10）。
- 跨重启不做序号比较，只退回时间戳（M10 的 authority 规则）。

### 4.4 重启恢复

`RecoverInterrupted()`（`internal/pantheon/manager.go:175-218`）现在把 running 一律置 `StatusFailed`，参与者置 `failed` + 「应用重启，本次调用被中断」。这条对"议事失败"是过度陈述：没人说它失败，只是没人看见。改成：

| 恢复时状态 | 恢复后 |
|---|---|
| Session `running` | `interrupted`（不是 `failed`） |
| Participant `running` | `finished_unverified` + 既有 delta 保留 + `Evidence: "应用重启，未观测到收声"` |
| Session `waiting_input` | 保持 `waiting_input`，问题与选项原样保留（M14：门不许被重启绕过） |

### 4.5 问题与回答

新增两条事件与一条投递通道：

- `EventQuestion{QuestionID, Options, AskedBy, At}`、`EventAnswer{QuestionID, Body, By, At}`、`EventGate{Action: open|resolve|dismiss, Reason}`。
- 回答投递只允许一个消费者取走并标 `delivered`（M9）；如果该 attempt 已终止，回答保留为孤儿并在 UI 标「这一次没送出去」，不许静默丢弃。
- 应用在 `waiting_input` 时重启，恢复后仍是 `waiting_input`；用户 dismiss 必须写理由。

---

## 5. 刀池（第二轮重排）

四把刀，按依赖排序。O-F / O-G 维持后置。

### O-D P1 席位状态证据与未知态 —— **已落地（2026-11-02）**

**改**：`types.go` 加 `Observed`、状态枚举与转移表；`manager.go` 的 `recordAgentStarted` / `recordAgentOutput` / `recordAgentError` / `recordAgentDelta` 统一走一个 `observeSeat(seat, observation)` 单写入口，转移非法即拒绝并落 `EventError`；UI 读 `Observed`。

**验收**（每条一个测试）：

1. 席位已 `finished` 后再 `recordAgentStarted`，必须被拒且不改变状态（终态自环）。
2. 连续 91 秒无 delta，`Observed.At` 不变；再发一条心跳事件，`Observed.At` 仍不变。
3. 重启恢复：`Session=running` → `interrupted`；该席位 → `finished_unverified` 且 `Evidence` 非空、既有发言仍在。
4. 重试后旧 attempt 的 `finished_unverified` 可因新陈词转 `finished`，但只在 `CallRef.AttemptID` 命中时；不命中必须拒。

### O-E P1 决策门与回答投递

**改**：`Session` 加 `waiting_input`；`Event` 加 question/answer/gate；`Manager` 加 `AskSeat` / `AnswerSeat` / `DismissQuestion`；`RecoverInterrupted` 按 §4.4 分流。

**验收**：

1. 席位提问后会话立刻 `waiting_input`，其余席位不再开始发言。
2. 回答后只有该席位被推进，且回答被标记 `delivered` 恰一次。
3. 在 `waiting_input` 时重启，恢复后仍是 `waiting_input`、问题还在——不许变 `failed`，也不许自己继续。
4. 同一条回答投递两次，第二次必须被拒（幂等）。
5. 缺 `QuestionID` 或 `Options` 的提问被判无效并留痕（M14 的准入）。

### O-H P1 陈词契约与结案缺口

**改**：`prompts.go` 加三段式与 outcome 必填；收陈词时校验非空 + attempt 命中；结案写 `Gaps`；`SignOff` 校验 `Gaps` 已列且进哈希。

**验收**：

1. 空陈词不产生 `finished`。
2. attempt 不匹配的陈词被拒（沿用已有 `CallRef` 命中）。
3. 有 `finished_unverified` 席位时，`completed` 的 `Gaps` 非空，且 `VerdictSHA256` 因 `Gaps` 变化而变化。
4. outcome 与正文矛盾时以 outcome 为准，UI 明示失败。

### O-I P2 取消与重试的证据保留

**改**：取消/跳过/重试不清事件；重试重建 `Incarnation` 与 `CallRef`；重试必须重声明 `RunMode`。

**验收**：

1. 取消后已产生的 delta 与工具事件仍在，且可按 `AttemptID` 检索。
2. 重试后旧 attempt 的事件不被覆盖、不被重排。
3. 重试缺 `RunMode` 时默认不得沿用上一次，必须失败或显式要求（M17）。

---

## 6. 补充「明确不抄」

第一轮已列七条；本轮再加四条，都来自这次读到的具体做法：

1. **不抄 30 分钟自动降级为 idle**。众神殿里"安静"不等于"空闲"：议员沉默是合法状态，把它显示成空闲会诱导用户去催。
2. **不抄完整信箱协议**（`--peek` / `--all` / `--ack deliveryId` 那一套）。我们没有多生产者信箱，只有一条"问题—回答"通道，够用的最小状态是 `asked / answered / delivered`。
3. **不抄估算成本**。Orca 的 Stats 会显示"• inferred pricing"并注明"prefer the provider console for authoritative spend"；众神殿宁可没有数字，也不给推断数字。
4. **不抄以快照当证据**的重绘路径。我们没有 PTY 重绘，`snapshot` 这一态在众神殿没有对应物。

---

## 7. 成功标准（在第一轮七条之上加五条）

1. 任何"完成"都能三问三答：谁报的、绑定哪个 attempt、有没有被接受。
2. 重启后不存在被伪装成"完成"或"失败"的席位；只有 `interrupted` 与 `finished_unverified`。
3. 问题不因重启消失，回答不因重投重复。
4. 结案页能列出缺口席位，且缺口进签注哈希。
5. 心跳不能续命：没有新证据的席位，不会被任何周期性事件显示成"正在发言"。

---

## 8. 源码索引（本轮新增）

| 机制 | Orca 侧 | 众神殿侧落点 |
|---|---|---|
| M9 投递状态机 | `src/main/runtime/orchestration/question-answer-delivery.ts:79-92`；`mailbox-pointer-delivery.ts:169`；`cli/orchestration.mdx:117-118` | `manager.go` 新 `AnswerSeat`；`types.go` 问题/回答事件 |
| M10 快照不续命 | `src/shared/agent-status-freshness.ts:28-40,57-58`；`agent-status-observation.ts:25-26` | `observeSeat` 的证据分类；UI 活性判定 |
| M11 新鲜度与未知态 | `src/main/runtime/orchestration/db/attempt-outcome-projection.ts:11-12,33-56,74-107` | `Observed.At` 判定；`finished_unverified` / `unavailable` |
| M12 终态图 | `src/main/runtime/orchestration/db/lifecycle-transition.ts:73-90,147-161,176-187` | 席位状态转移表；唯一例外通道 |
| M13 兄弟在跑不结案 | `attempt-outcome-projection.ts:144-149` | 结案判定（§4.2） |
| M14 门重阻塞 | `src/main/runtime/orchestration/coordinator-decision-gates.ts:21-24,38-45,51-59` | `RecoverInterrupted` 分流；`DismissQuestion` 需理由 |
| M15 陈词契约 | `src/main/runtime/orchestration/preamble.ts:90-95,159-163`；`cli/orchestration.mdx:136-139` | `prompts.go`；收陈词校验 |
| M16 释放与归档 | `cli/orchestration.mdx:89` | 取消/跳过保留事件 |
| M17 重试重声明 | `cli/orchestration.mdx:85-86`（`--retry-of` 不继承 `--on`/worktree） | 重试重建 `CallRef` 与 `RunMode` |

---

## 9. 一句话

Orca 真正值钱的不是它替人开了多少 Agent，而是它**从不相信"看起来完成了"**：完成要有报告、报告要绑 dispatch、报告过期要降级、降级要显示、显示要能追到证据、证据跨进程不许比较。众神殿要搬的是这六条，不是它的工作台。

---

## 10. 落地记录（2026-11-02，O-D 已接上，未提交）

改动只在 `internal/pantheon` 与 `frontend`，未 commit、未抬版本；`clones/orca` 保持只读。

**Go**

- `internal/pantheon/observe.go`（新）：`seatObservation`、`isFreshnessEvidence`、`observeSeatLocked`（席位状态唯一观测入口）、`rebindSeatLocked`（新 attempt 的控制动作入口）、`shortEvidence`。两个入口的分工是这刀的核心：**观测只能举证，控制只能重建**。
- `internal/pantheon/seatstate.go`：转移表加 `running → skipped`、`retrying → cancelled`（都来自接已有代码路径时暴露的真实边，见 §4.1 末注）。
- `internal/pantheon/types.go`：`Observed` / `Evidence` / 十态枚举 / `waiting_input` / `interrupted` / question·answer·gate 三条事件（O-E 预留，尚未接线）/ `Session.Gaps`。
- `internal/pantheon/manager.go`：`authorityID` 由 `NewManager` 生成，`observeSeatLocked` 发现 authority 变了就把 `Revision` 归 1；`recordAgentStarted` / `recordAgentOutput` / `recordAgentError` / `recordAgentDelta` / 跳过 / 重试 / 取消全部改走单写入口；`RecoverInterrupted` 把"置 failed"改成"置 interrupted + 席位 finished_unverified"；继续议事改走 `rebindSeatLocked`（终态席位抬 `Incarnation` 回 `waiting`）。
- `internal/pantheon/prompts.go`：`contributionContract()` 三段式陈词 + "没做成的事进置信度行" + "证据只填真实值"，接进开场与补充论证两处 prompt。
- 测试：`observe_test.go` 五条（非法转移被拒且不留痕、快照/心跳不续命、同态观测推进 revision、跨 authority 归 1、重试抬 incarnation）；`manager_test.go` 的重启恢复测试改断言 `interrupted` + `finished_unverified` + `Observed.Origin=recovery`，并断言已收声的席位不受波及。

**前端**

- `pantheon.ts` 加 `PantheonObserved` 与 `evidence`；`PantheonRecord.tsx` 的状态词表加 `finished_unverified`（未验证返回）/`needs_input`/`cancelled`，`statusMeta` 加 `interrupted`（已中断）与 `waiting_input`（等待回答），秒表与"可恢复"判定纳入 `interrupted`/`waiting_input`；`PantheonPage.tsx` 的列表状态、筛选、删除守卫把钱等回答的议题当"活着"；`PantheonAgentAvatar.tsx` 与 `pantheon.css` 加 `is-unverified`（虚线点，不是成功也不是失败）。

**门禁**

- `go build ./...` 通过；`go vet ./internal/pantheon/` 通过；`go test ./internal/pantheon/ -count=1` 通过。
- `npx tsc -b` 通过；`npx vitest run src/pages/pantheon` 46 条通过。

**没做的**（留给后续刀，不是遗漏）

- `Participant.Observed` 是值类型，`omitempty` 对 struct 无效，`session.json` 里每个席位都会多一个 `"observed":{}`。要区分"从没有过观测"得改 `*Observed`，那是 O-D 之外的字段变更。

---

## 11. O-E / O-H 落地（2026-11-02 续，仍未提交）

§10 里那三条"没做的"已经全部消掉，另外接上了决策门与回答投递。

**门与提问（O-E）**

- `internal/pantheon/gates.go`（新）：`AskSeat` / `AnswerSeat` / `OpenGate` / `ResolveGate` 四个动作，加三个只读推导函数 `hasOpenGateLocked`、`pendingQuestionsLocked`、`seatGaps`。
- 门与问题**都不存独立字段**，一律从 `session.Events` 重放推导。理由：会话文件是唯一真相，重启后重拉一次快照就能原样重建"还有没有人在等"，不会出现"字段说没门、事件里还挂着一条"。
- `run` 循环有两个停车点：轮次之间、席位之间。已经在跑的那一席不打断（外部进程没法拦腰插问），下一个席位绝不启动；停下时留一条 `phase` 事件说明停在哪一轮。
- `StartRequest.GateEachRound` 支持"每轮之间停下确认"，此时 `run` 在本轮收声后开门返回。
- `Resume` 现在会拒：还有未答问题时报"还有 N 个问题没有回答"，门没解时报"有一道决策门没被处理"。以前它会硬着头皮续跑、然后立刻又被停回去，看起来像"点继续没反应"。
- `RecoverInterrupted` 明确跳过 `waiting_input`：等人不是被中断，重启不替人回答，也不替人解门。

**回答投递的边界**

- 回答只绑 `QuestionID`，答第二次直接被拒（`该提问不存在或已经回答`），对应 Orca 的 exactly-once 投递。
- 回答**不触发续跑**。续跑是新的 attempt：席位走 `rebindSeatLocked` 回到 `waiting`，由人再点一次"继续讨论"（M12：控制动作与状态观测分开）。
- 前端在 `waiting_input` 时渲染等待面板：谁在问、问什么、选项按钮、自由输入；门打开时给"放行 / 放弃后续轮次（必填理由）"。

**结案缺口（O-H）**

- `run` 收尾时 `session.Gaps = seatGaps(session)`，`finished_unverified / unavailable / skipped / cancelled` 四种席位具名进缺口，完成事件文本带上"有 N 席没交出可用结论：…"。
- 签注指纹改走 `signoffFingerprint(verdict, gaps)`（`internal/pantheon/trace.go`）：无缺口时与旧的 `verdictFingerprint` 完全一致，有缺口时把缺口拼进去——同样的小结、不同的缺口不能签出同一个指纹。`SignOff` 同时落 `Gaps`。
- 前端会话类型补 `gaps`、`signoff.gaps`、`question/answer/gate` 三个事件载荷。

**接线**

- `internal/app/pantheon_handler.go`：SSE 载荷补 `question/answer/gate`；新增 `PantheonAskSeat/PantheonAnswerSeat/PantheonOpenGate/PantheonResolveGate` 四个包装，并重跑 `scripts/gen_bindings`、`wails generate module` 与 `bindingNames.ts` 生成（漂移锁要求三处同步）。
- 席位级活性：`PantheonRecord.tsx` 里 `seatStatusText` 在看得到席位观测后，超过 90 秒没有更新就把"正在发言"换成"活性未知（最近观测 N 秒前）"——不判失败，只是不再替它保证在干活。

**门禁**

- `go build ./...` 通过；`go test ./internal/pantheon/` 通过（新增 8 条门/回答/缺口用例）；`go test ./internal/app/` 通过。
- `npx tsc -b` 通过；`npx vitest run src/pages/pantheon` 46 条通过。

**仍然没做**

- `Observed` 值类型那条照旧（需要 `*Observed`，属于字段口径变更）。
- `AskSeat` 目前只有 App 绑定入口，还没有"席位在发言里自己提出一个问题"的自动识别：那需要在适配器输出里做结构化解析，属于 O-E 的第二期。

---

## 12. O-I 落地：重试重声明、取消不收回证据（2026-11-03 续，仍未提交）

上一节写"没做"的时候漏复核了两条，先更正：SSE 早已由 `emitSafe` → `app.emit("pantheon-event")` 直推（`internal/app/pantheon_handler.go:65-67`），`session.warnings` 前端在 `frontend/src/pages/pantheon/PantheonRecord.tsx:1605` 一直有渲染块。它们不是待办，是我写 §10 时的旧结论没跟着改。

**重试要自己声明运行方式（M17）**

- `internal/pantheon/manager.go` 的 `RetryAgent` 现在从 `m.adapters[agentID].runMode` 取本次声明，取不到就拒绝：`%s 没有声明运行方式，重试不能沿用上一次的口径`。拒绝时不写 `agent_retry` 事件、不改席位状态，用户能看出"重试没发生"，而不是等一圈发现没动静。
- 允许重试时，事件文本带上实际 placement：`正在重新调用，保留此前发言（运行方式：…；工作空间：…）`。重试不再是"大概还是老样子"，记录本身能回答"第二遍拿什么跑的"。
- 这条只约束重试。首次调用沿用适配器声明是正常的，因为声明本来就在同一次调用里。

**取消与重试不收回证据（O-I 主体）**

- `internal/pantheon/retry_evidence_test.go`（新，4 例）：
  - `TestRetryRefusesUndeclaredRunMode`：声明为空时拒绝且不留 retry 事件；补上声明后同一次重试能走通。
  - `TestRetryDeclaresPlacementInEvent`：重试事件里同时有运行方式与工作空间。
  - `TestCancelKeepsAttemptEvidence`：取消后事件 `seq` 仍是从 1 起的连续序列（不删、不重排），且该 attempt 的 `agent_start` 仍在。
  - `TestRetryKeepsEarlierAttemptEvents`：重试产生新 attempt（`RetryOfAttempt` 指向旧 attempt），旧 attempt 的 `agent_start` 的 `seq` 一位不动。

**活性汇总**

- `PantheonRecord.tsx` 标题区加"`N` 席活性未知"：任一在跑席位超过 90 秒没有新观测就显示，tooltip 点名是哪几席。单席位在拓扑上已经看得到，这条只是让整场不必逐个点开才知道有几席在硬撑。
- 样式 `.pantheon-record-stale` 用虚线框，与"失败"和"正常"都区分开。

**门禁**

- `go test ./internal/pantheon/ -count=1` 通过（新增 4 条）。
- `npx tsc -b` 通过；`npx vitest run src/pages/pantheon` 46 条通过。

**仍然没做**

- `AskSeat` 的自动识别（同 §11）。
- 跳过（`SkipAgent`）没有同样的"重声明"约束：跳过不启动新进程，不涉及 placement，留作已知差异。


