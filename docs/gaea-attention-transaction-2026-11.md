# 注意力事务：把「打断」做成有起点、有结果、可重放的持久事务（2026-11-02）

> 来源：众神殿会话 `pan_1793618668807_6f24ec495bc1`（2026-11-02，四席：Codex / ZCode / DSH / grok 缺席）就 DeepSeek 开源 super agent 原型
> （`deepseek-ai/dsh-superagent-prototype`，MIT 研究原型）对 gaea 的启示所做讨论。结论是：可蒸馏的不是 Router/Worker，
> 也不是 watchdog，而是**把请求人的注意力做成事务**——有起点、有结果、可重放、可计数，且触发来源与阻塞类别必须贯穿
> 实时事件、持久日志与派生轨迹三层。
>
> 本档记录第一刀（判别字段贯通 + 结果侧成对事件 + 阶段 0 基线统计器）的契约与口径；阶段 2（`case_id`/`step_id` 事项维）
> 尚未落地，见文末开放项。

## 一、问题：实时层有、真相层没有

现状（本刀之前）的事实链：

| 层 | 审批请求 | 审批结果 | 提问 |
|---|---|---|---|
| 实时事件 `event.Approval` / `event.Ask` | `ID` `Tool` `Subject` `Reason` `Request` | 无 | `ID` + `Questions` |
| 会话日志 `<id>.gaea-log.jsonl` | 仅 `id` `tool` `subject`（**丢 `reason`/`request`**） | 无 | `id` + `questions` |
| 派生轨迹 `ApprovalRec` / `AskRec` | 仅 `tool` `subject` | 无 | `question` |

结论是「类型实时层存在、真相层丢失」的典型形态：

- 前端弹卡时用户能看到理由（权限升级 vs 模型主动申请），但日志里没有，事后无法区分来源；
- `Approve()` / `AnswerQuestion()` 只往内存 channel 写一个值，**不产生任何完成事件**——批了/拒了/超时/取消四种结局在日志里
  完全同形，`approval_request` 之后永远是空白；
- 于是「每轮打断几次、批了几次、等多久」这类问题**无法用日志回答**。这也正是 V5 提案里
  `docs/gaea-v5-case-os-proposal-2026-10.md:710`「≤6 次且 −50%」指标没有基线定义的原因——分母本身不存在。

## 二、计数口径（先定口径，再谈阈值）

- **一次打断 = 一张卡**。`approval_request` 或 `ask_request` 各计一张；`ask` 卡内多问题仍算**一张卡**（按卡不按问题数，问题数进
  `questions` 诊断列）。
- **审批 / 提问分项**，永不合并成一个「打断数」；`notice`（提示、超时提醒）只作诊断，不计入。
- **短路不算事务**：会话内已有 granted 规则时不弹卡、不发请求事件，因此也不产生事务——「没打扰人」的事实就是没有记录，
  不伪造一条 `allowed`。
- **分母**：目前只有轮次（`turn_started`）。「每事项」所需的 `case_id` 尚不存在（`internal/caseos` 未建），所以现阶段只能报
  「每轮打断数」，**不报「每事项打断数」**——避免造一个没有分母的达成率。
- **结果归一**：`allowed` / `denied` / `aborted` / `timeout` / `cancelled` / `answered`；明细决策串（`allow_once`、
  `allow_session`、`persist_allow`、`deny`、`abort`、`timeout`）原样保留在 `decision` 里，不因归一面丢语义。

## 三、本次落地的事件契约

新增两个结果侧 kind（**不重命名**既有 `approval_request` / `ask_request`，旧日志与前端零破坏）：

| kind | payload | 语义 |
|---|---|---|
| `approval_request` | `id,tool,subject,source,reason,request` | 审批卡出现（补回此前被日志层丢掉的判别字段） |
| `approval_resolved` | `id,tool,subject,source,decision,outcome,latency_ms` | 审批卡闭合（允许/拒绝/中止/超时/取消） |
| `ask_request` | `id,questions[…]` | 提问卡出现（不变） |
| `ask_resolved` | `id,outcome,answered,latency_ms` | 提问卡闭合（answered/cancelled） |

`source` 四值（`event.ApprovalSource*`）：

| 值 | 触发点 |
|---|---|
| `gate` | 常规权限闸（工具调用需放行） |
| `hard_ask` | 硬询问工具（如 `cost_save`，即使在 yolo 下也必须问） |
| `project_plan` | 项目计划变更确认 |
| `permission_request` | 模型主动 `request_permission` 权限升级 |

旧日志没有 `source` 字段——读端**不把「未记录」当 `gate`**，一律记 `legacy`；若同 `id` 有结果侧事件则按结果里的 `source`
精确归位（结果事件在请求发生时生成，来源同一事实）。

`latency_ms` 从「请求事件发出」到「解决事件发出」，覆盖真人思考时间、超时窗口与取消，不扣除。

第六种结局 `abandoned`（进程先死在卡上）：恢复会话时由 `session.CloseOrphanAttention` 补写，`decision` 为空、
`latency_ms=0`（不知道挂了多久）——统计器把 0 计进结果分布但不进均等待。

## 四、三层贯通与审计路径

- **实时事件**：`internal/gaea/event/event.go` 增 `ApprovalResolved` / `AskResolved` 与 `ApprovalResolution` /
  `AskResolution`；`Approval` 增 `Source`。
- **发射点**：`internal/gaea/control/controller_approval.go` 的 `promptApproval` 在 Emit 请求后 `defer` 发结果（覆盖
  reply / ctx 取消 / 超时三条出口）；`internal/gaea/control/controller.go` 的 `Ask` 同法覆盖 answered 与 cancelled。
- **持久日志**：`internal/gaea/agent/session/log.go` 的 `KindString` 与 payload 构造同步补齐；结果事件与请求事件同一 `id`，
  日志里天然成对，`seq` 单调，可重放。
- **派生轨迹**：`internal/gaea/trajectory/fold.go` 的 `applyApprovalResolved` / `applyAskResolved` 按 `id` 回填请求记录
  （`outcome` / `decision` / `latency_ms` / `answered`），不新增孤立记录；找不到请求的孤立结果事件被忽略——**日志原文仍是
  唯一真相**，折叠只是派生视图。
- **前端**：结果事件走 `controller_state` 的 `default` 分支（无状态变化），本轮**零前端改动**。它是账本，不是交互。

## 五、阶段 0 基线统计器

```powershell
go run ./scripts/attention_report                                  # 默认扫 .gaea/sessions 及 work/play 分区
go run ./scripts/attention_report <目录或日志> ...                  # 指定面
go run ./scripts/attention_report -json                            # 机器可读
```

输出逐会话一行（轮次 / 打断卡 / 审批 / 提问 / 已解决 / 未解决 / 均等待）+ 结果分布 + 触发来源 + 未解决卡清单。只读，不写任何文件，
不改运行时行为；单文件读失败跳过并在 stderr 提示，退出码恒 0（统计不是断言）。

用法纪律（承众神殿结论）：挑**三件真实任务**跑，分别统计，**分项报告审批/提问**，不要合成一个「打断 KPI」。旧日志会显示
「未解决 = 全部」——那不是运行事故，而是阶段 1 之前的历史缺口，不能拿来当退化证据。

## 六、明确不做什么

- **不重建 30 秒看门狗**：v3.0.2 用户拍板删除（`c426d3ff` 引入、`a5e33377` 删除），根因是把模型首 token 等待与信号量排队误判为
  「无推进」而误杀。本刀不碰墙钟。
- **不照搬常驻 Router**、**不整包移植**上游原型（MIT 研究原型，README 自述部分设计未实现）。
- **不追「每轮 ≤6」**：基线尚未统计，阈值没有定义；先把数出来，再谈目标。
- **不改 headless 语义**：无 Approver 的 headless 放行是有意设计（`permission.go` 注释 non-interactive: preserve autonomy），
  它不弹卡也因此不产生事务，属正确行为。

## 七、开放项（下一步）

1. **事项维**：`case_id` / `step_id` 需要 `internal/caseos`（不存在），暂缺；在它落地前，「每事项打断数」不可报。
2. **V5 事件账本对齐**：`docs/gaea-v5-case-os-proposal-2026-10.md:381` 的事件集只有 `approval.granted` 单事件，与本刀的成对
   事务（requested/resolved）口径不同；后续要么把 V5 账本改成成对，要么在提案里显式声明两者的映射，避免两套账。
3. **旧日志不可补**：历史 `approval_request` 没有结果侧，永远显示未解决；不要在报表里把它们算成「未响应」。新日志里
   进程被杀留下的悬空卡会在下次恢复时补 `abandoned`（见第九节），两者口径不同，报表按是否有结果事件区分。
4. **initiator / risk_tier**：`initiator`（谁触发的）已按回合建模为 `turn_started.trigger`（见第十节，user/wake/subagent）；
   `risk_tier`（风险档）仍缺，等有真实消费方再加。
5. **真人打断与自唤醒分离**：`wakeMaxConsecutive=3` 计的是 agent 自唤醒（`controller.go:313`），不是人打断；两套计数不要混。
   回合发起者类型化后，报表的打断率分母只取人轮，两者口径终于能在同一份日志里分开。

## 八、验证

- `internal/gaea/control/attention_test.go`：审批 8 个子测试（allow_once / deny / timeout / cancel / hard_ask 来源 /
  会话已有授权短路不产生事务）+ 提问 answered / cancelled，全部断言「请求与结果同 id 成对」。
- `internal/gaea/agent/session/log_test.go`：`TestEntryFromEventAttentionPayloads` 断言判别字段进日志、结果事件成对落盘。
- `internal/gaea/trajectory/fold_test.go`：`TestFoldAttentionResolvedPairs` 断言按 id 回填、孤立结果事件不建记录。
- `scripts/attention_report/main_test.go`：配对计数、重复结果事件去重、legacy 来源归位、目录展开去重。
- `internal/gaea/agent/session/attention_test.go`：悬空卡识别（重复请求去重 / 孤立结果无害）、补写 `abandoned` 且 `source`
  原样带回、二次收口为 0（幂等）。
- `internal/gaea/control/controller_attention_resume_test.go`：造一份「请求行写完、解决事件没来得及写」的崩溃日志，
  `ResumeFromDisk` 后断言无悬空卡，且轨迹折叠出 `outcome=abandoned / latency=0 / source=hard_ask` 的成对记录。
- `internal/gaea/control/controller_wake_test.go`：唤醒回合的 ctx 里 `trigger=wake`（类型化判别，不再看输入前缀）。
- `internal/gaea/agent/session/log_test.go`：`TestEntryFromEventTurnTrigger` —— `turn_started` 落 trigger；旧事件仍写 `{}`。
- `internal/gaea/trajectory/fold_test.go`：`TestFoldTurnTrigger` —— `Turn.Trigger` 落轨迹，旧日志回合保持空。
- `scripts/attention_report/main_test.go`：轮次按发起者拆分（人轮/唤醒/子代理）。

## 九、第二刀：悬空卡闭合（进程先死也有结局）

**为什么紧跟着做**：阶段 0 的基线要可信，就必须先让「卡永远悬空」这个洞闭上。否则进程一崩，那张卡在账本里既不是
「用户拒绝」也不是「用户没看见」，统计器只能读成「未解决」——打断数偏高、解决率偏低，而这两种事实本来完全不同：
一种是人做了决定，一种是没人做过决定。

**做法**（不碰墙钟、不推断用户意图，只承认事实）：

- `session.UnresolvedAttention(entries)`：流式扫日志，收拢「有请求、无结果」的卡；同 `id` 的重复请求只算一张，孤立结果
  事件不会凭空造卡，审批的结果不会关闭一张提问卡。
- `session.CloseOrphanAttention(logPath, space)`：为每张悬空卡追加 `approval_resolved` / `ask_resolved`，
  `outcome=abandoned`、`decision=""`、`latency_ms=0`；幂等（补写后重跑为 0 张）。
- 挂点只有一处：`Controller.ResumeFromDisk` 事件模式分支，在 `LoadWithFormat` 之后、`Resume` 之前调用。这是宿主真正
  「重新接管一份日志」的唯一收敛点（切换会话与启动恢复都走它），而 rewind 的截断重放不走这里，不会被误收口。
- 失败不阻断恢复：账本收口是尽力而为，日志原文仍是唯一真相。

**它不是什么**：不是看门狗，不设阈值、不计时、不杀回合；它只在恢复这一瞬间承认「这些卡没人回答过」。

## 十、第三刀：回合发起者类型化（合成轮不再靠文本前缀反推）

**为什么**：众神殿风险项之一——「合成唤醒目前依靠 `[system]` 文本前缀和白名单识别，准入规则部分依赖提示词而非
类型化权限约束」（上游讨论结论档 :46）。文本前缀一改就误分类，而且统计上没法把「真人的回合」与「后台唤醒的回合」
分开：打断率的分母里混进了唤醒轮和子代理轮，数字就不诚实了。

**做法**（与卡片判别字段同一条线：类型化 → 贯穿实时事件/日志/轨迹/报表）：

- `event`：`TurnStarted` 增 `Trigger` 字段，取值 `TriggerUser="user"` / `TriggerWake="wake"` /
  `TriggerSubagent="subagent"`；空值=旧事件，读端按 `user` 兜底。
- `agent`：`WithTurnTrigger(ctx, trigger)` / `TurnTriggerFromContext(ctx)`——判别字段的源头在入口（控制器知道是真人
  发送还是后台唤醒），而 `TurnStarted` 在 `agent_run.go` 的 `runDirect` 里发射，ctx 是唯一不改 `Run` 签名的贯通路径。
- `control`：`sendWithRaw(input, raw, resetWake, trigger)` 把发起者一路带进 `runSendTurn`（排队消息也记住自己的
  trigger），`maybeWakeByJobs` 传 `TriggerWake`，`SendWithRaw`/`Submit` 传 `TriggerUser`；`runSubAgentInternal` 在
  子代理调用点标 `TriggerSubagent`。
- `session`：`turn_started` payload 落 `{"trigger": ...}`；空 trigger 时仍写 `{}`，与旧日志逐字节一致。
- `trajectory`：`Turn.Trigger`（omitempty）——旧轨迹输出不变。
- `scripts/attention_report`：轮次按发起者拆分（人轮/唤醒/子代理），表头新增三列；打断率的分母只取人轮。

**仍存的文本前缀依赖（诚实边界）**：`interrupted_recovery.go:56` 的 `isSyntheticUserTurn` 仍在扫消息历史时按
`"[system] "` 等前缀判定合成轮——它作用于消息回扫而非事件，要类型化需要消息级 provenance，本轮不做，留在开放项。

**验证**：`control` 的唤醒测试断言唤醒回合 ctx 里 `trigger=wake`；`session` 断言 payload 落 trigger 且旧事件仍是 `{}`；
`trajectory` 断言 `Turn.Trigger=wake`、旧日志回合保持空；`report` 断言人轮/唤醒拆分。

