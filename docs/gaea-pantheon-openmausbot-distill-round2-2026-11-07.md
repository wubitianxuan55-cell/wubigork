# OpenMausBot × 众神殿 第二次蒸馏计划：收口、证据与提问（2026-11-07）

> 上游钉版：`milind-soni/OpenMausBot` commit `1bc4749aa8307d7d6998bf86de4ef6478460dd93`（2026-11-07T04:24:54Z），package.json `0.1.146`，Apache-2.0；本地克隆 `C:\AI\tmp\OpenMausBot`。
>
> 上游增量核对：`origin/main` 已到 `0b54240`（2026-11-07T17:55:55+05:30，*Meeting notes: bots join Meet, Teams and Jitsi calls*），diff 47 个文件，主体是 locales 与 `third_party/vexa`，属会议笔记能力，不在议事范围。**钉版维持 `1bc4749`**，不因该提交重开仓库级蒸馏；下次重评仍只 diff 相关纯函数与契约文件。
>
> 前情：第一轮 `docs/gaea-pantheon-openmausbot-distill-2026-11-07.md`（v2，G0 证据冻结）。A1 主席信封、A2a 环形检测纯函数、A5 共享上下文已落地（提交 `61694638`；Grok 侧并行会话已把 Seed 路径实弹证伪后回退为全量窗口内联）；A2b 关闭；B1/A3/A4/B2/B3/B4/B5 按门控挂池。
>
> 读法：本轮不按“上游还有什么功能”列清单，而按**一条结论从产生到被相信的路径**读代码：席位说了什么 → 宿主实际看见什么 → 主席据此怎么收口 → 用户签注时能核对什么。OpenMausBot 真正可搬的不是它的房间 UI，而是这条路径上的**证据与收口纪律**。
>
> **落地状态（2026-11-07 晚）**：E1 / E2 / E3 / E4 已落地并通过定向验证；E5 仍是观察项（只落了 `RepeatFailures` 数据）；E6/E7/E8 挂池。落地明细见 §15。

---

## 0. 一句话与默认路线

第一轮解决的是“**怎么把会开起来、把上下文接住**”；第二轮只解决“**怎么让会开完之后，记录说的和实际发生的是同一件事**”。

因此第二轮只加**可核对性**，不加**能力**：不引入写权限、不引入 Jev 路由、不建第二套席位/分歧/签注状态。候选收敛为四刀 + 一个观察项：

| 刀 | 名称 | 一句话 |
|---|---|---|
| **E1** | 席位工具证据摘要 | 每席陈词旁记下宿主实际看见的工具调用（次数/成败/重复失败），不是模型自述 |
| **E2** | 派工回执 + A1 透明补完 | 主席派了什么、下一轮谁真的发言了、谁失败/缺席，一屏可查；先补上 A1 承诺过但没渲染的派工内容 |
| **E3** | 主席终局信号 | 主席可显式说“可收口 / 被阻塞 / 需人回答”，宿主据此停车；不能绕过既有共识与分歧门 |
| **E4** | 席位自述提问 | 席位在陈词末尾写 `NEEDS INPUT:`，宿主在轮末停车、把问题交给用户；这是 Orca 第二轮明确留给第二期的缺口 |
| **E5** | 同位失败连击止损 | **观察项，不先做**：先用 E1 记账，再决定要不要在连续同一失败时终止 |

**默认路线（已执行）**：

1. **P0 = E1 + E2**。两者都不改停车点、不改权限：E1 只加一个可选字段与展示，E2 只用现有存档做只读派生并把派工/回执显示出来；唯一的行为变化是主席下次小结能看到回执（仍走现有 moderation 路径）。
2. **P1 = E3 + E4**。两者会改变“什么时候停、停下来等谁”，已按 §10 门控一次落地。
3. **P2 = E5 观察项 + E6/E7/E8 挂池**。没有数据不立项；E1 的 `RepeatFailures` 已开始记账。

如果只能做一刀：做 **E2**。它同时补上第一轮 A1 的透明度缺口（派工已入库但 UI/导出看不见）和第二轮最核心的“派工有没有回执”。

---

## 1. 第一轮落地复核：先销三笔

### 1.1 A2c 状态表口径更正

第一轮 §0 表把 A2c（重复派单检测）写成“**阻塞于 A1**”，但 §3.1 与代码已经是“**A1 随行落地，同一（席位, 归一化指令）≥3 次只告警、不阻断**”（`internal/pantheon/chair_envelope.go:129-164`，`internal/pantheon/manager.go:1407-1410`）。以代码为准：A2c 不是待解锁项，是已落地的**弱语义告警**。

这不是要把它升级成阻断。上游 `group-goal-run.ts:292-298` 在“同一个 coordinator 主持的有界 goal run”里三次重复就直接 `blocked`，因为那里有明确的 reassign 队列与终止语义；众神殿是圆桌，主席重复派工可能是在催同一件事，升级为阻断会制造新的“主席一意孤行就停会”的失败模式。A1 契约选择“只告警”仍成立，**不重开**。

### 1.2 A1 的透明度承诺没有完全兑现

A1 契约写的是“用户经 Warnings 与导出可见派工内容（透明性优先于隐蔽性）”。实际核对：

- `internal/pantheon/export.go:256-281` 只导出 moderation 贡献的 `Content`（已剥信封的可见文本），**没有渲染 `ChairAssignments`**；
- 前端 `PantheonRecord.tsx` 也没有派工块（grep `chairAssignments` 零命中）；
- Warnings 只在“信封解析失败/派工指向非参会席位/重复派工”时出现，正常派工本身不进 Warnings。

所以 A1 目前的事实是：**派工写进了 `session.json`，但人看不见**。第二轮 E2 的第一步就是把这个缺口补上——先把已存的派工在记录与导出里显示出来，再加回执。这是“补第一轮的账”，不是新机制。**E2 落地后，记录页/导出已补上派工与回执（见 §15）**。

### 1.3 第一轮主线成立，但“派工—回执—收口”这条链缺两环

第一轮把 A1 做到了“派工能注入下一轮提示词”，A5 做到了“四席都有全文记忆”（Grok 当前走全量窗口内联，Seed 路径已由并行实弹证伪回退），O-E（Orca 第二轮）做到了“人能提问、门能停车”。三条链各自成立，但合起来还缺：

1. **派工发出后，没有回执**：主席不知道被派席下一轮到底说没说话，更不知道它失败/被跳过/缺席；用户只能逐条翻贡献。
2. **席位自己不能提问**：`AskSeat` 只有 App 绑定入口，席位在陈词里遇到“必须问人才能继续”的情况，只能猜或写进置信度行。Orca 第二轮落地记录已经明确把它列为“第二期”（`docs/gaea-orca-pantheon-distill-round2-2026-11.md:410`）。

第二轮的四刀就是补这两环，并把“证据”做成可核对的底账。

---

## 2. 候选总表

| 编号 | 候选 | 证据类别 | 状态 | 影响/成本 | 依赖 |
|---|---|---|---|---|---|
| **E1** | 席位工具证据摘要 | **源码直证** `turn-outcome.ts:115-125`、`runtime/types.ts:160-172` | **已落地（P0）** | 中/低 | 无 |
| **E2** | 派工回执 + A1 透明补完 | **源码直证原则 + 本地再设计派生** `group-goal-run.ts:282-339`、`runtime-contract` §2/§7 | **已落地（P0）** | 中/低 | A1（已落） |
| **E3** | 主席终局信号 | **源码直证** `group-goal-run.ts:13-15,58-85,276-279` | **已落地（P1）** | 中/中 | A1、O-E 门 |
| **E4** | 席位自述提问 | **源码直证 + 本地再设计时序** `runtime-coordinator.ts:72-83`、`ask-cards.ts:253-260` | **已落地（P1）** | 中/中 | O-E 问题/门 |
| **E5** | 同位失败连击止损 | **源码直证但上游默认关闭** `host/loop-guard.ts:140-200` | **观察 P2** | 中/中 | E1 数据 |
| **E6** | 席位上下文重开 | 源码直证 `room-context.ts:10-15,101-112` | **挂池** | 中/中 | 引擎窗口上报 |
| **E7** | 等待回答的存续与提醒 | 源码直证 `host/person-waits.ts:31-35,100-130` | **挂池** | 低/中 | 通知/收件箱 |
| **E8** | 材料来源分级 | 源码直证 `host/outside-fence.ts`、`host/outside-writes.ts` | **挂池** | 高/高 | 产品拍板 |

**分类口径**（沿用第一轮）：

- **源码直证**：上游有可平移的机制契约，锚点可逐条核对；
- **本地再设计**：上游有相邻物，但众神殿的形态不同（例如“派工回执”在上游是 assignment 运行时的一等状态，我们只能从存档派生），必须自证；
- **观察项**：上游有，但默认关闭或需要本地数据才敢开；
- **挂池 / 不立项**：见 §8、§9，写明理由，不重复立项。

---

## 3. E1 席位工具证据摘要

### 3.1 上游机制

`server/turn-outcome.ts:115-125` 从一次回合的活动行里算出一份**宿主观察**，不是模型自述：

```ts
export function turnEvidence(rows: readonly Message[]): TurnEvidence {
  const tools = toolRows(rows);
  const errors = tools.filter((row) => row.tool!.ok === false).map(...).slice(-ERRORS_KEPT);
  const okToolCalls = tools.filter((row) => row.tool!.ok !== false && row.tool!.itemId !== undefined).length;
  const outward = outwardActions(rows);
  return { toolCalls: ..., okToolCalls, errors, ...(outward.length ? { outward } : {}) };
}
```

配套三条：

1. `TurnEvidence` 的结构（`server/runtime/types.ts:160-172`）：`commands / filesChanged / toolCalls / okToolCalls / errors / outward` —— 全是宿主观察，不含模型文本；
2. 报告里单独一行 “Observed by the harness: N tool calls (M ok)” （`server/runtime/envelope.ts:148-160`），让读报告的人能区分“席位说它读了”与“宿主看见它读了”；
3. 这份证据又被两个地方消费：`server/runtime/tree.ts:45-63` 的循环守卫（连续无成功工具调用、同一错误三次）与 `runtime-contract` §6 的停滞判据。**先有证据，后有止损**，顺序不能倒。

### 3.2 我方缺口

`internal/pantheon` 已经有工具事件（`EventToolStart` / `EventToolResult`，带 `CallRef`、`ToolID`、`ToolName`、`ToolStatus`、`ToolInput`、`ToolOutput`），也已经有引擎侧账目 `Contribution.Usage`（token/缓存/步数，`types.go:195-207`），但**没有把“这一席实际调了什么工具”固化成陈词的一部分**：

- 实时 UI 有 `ToolTraceList`，但那是直播态；
- 导出只写陈词正文，读者无法核对“它说读了三个文件”；
- `Contribution` 没有 `ToolEvidence` 字段（grep 零命中）。

所以 E1 不是新采集，而是**把已有事件在陈词落档时压成一份可核对摘要**。

### 3.3 设计（Go 形状）

```go
// types.go
type ToolEvidence struct {
    Calls          int         `json:"calls,omitempty"`          // 去重后的工具调用数（按 ToolID）
    OK             int         `json:"ok,omitempty"`             // status=completed
    Failed         int         `json:"failed,omitempty"`         // status=failed
    RepeatFailures int         `json:"repeatFailures,omitempty"` // 同一（工具,归一化输入）连续失败的最大次数
    RepeatTool     string      `json:"repeatTool,omitempty"`     // 该连续失败的工具名（不含参数）
    Tools          []ToolCount `json:"tools,omitempty"`          // 最多 6 项，按次数降序；只存工具名
}

type ToolCount struct {
    Name  string `json:"name"`
    Count int    `json:"count"`
}

// Contribution 增：
ToolEvidence *ToolEvidence `json:"toolEvidence,omitempty"`
```

**计算点**：`runOne` 结束、`seatResult` 返回前，按 `CallRef.AttemptID`（为空则 `CallID`）筛 `session.Events`，只读计数；不要重新解析 stdout。

**归零规则**：

- 适配器没有工具流能力（Grok 模型席）→ `nil`，UI 显示“该席不上报工具轨迹”，**不是 0 次**；
- 有工具流能力但确实 0 次 → 非 nil 零值，UI 显示“0 次工具”；
- 工具 `started` 有、`completed/failed` 无 → 计入 `Calls`，但不计入 `OK/Failed`；UI 以 `Calls - OK - Failed` 显示“未回报”，不猜成功。

**重复失败**：只在内存里算。按 `ToolName + 归一化 ToolInput` 建 key，连续 `failed` 才累计；遇到成功或换 key 归零。只把**最大连续次数 + 工具名**落盘，**不落错误原文、不落命令、不落输出**——避免把凭证/路径带进纪要。需要看原文的人点开直播轨迹。

**隐私边界**：`Tools` 只存工具名，不存 `ToolInput`/`ToolOutput`；工具名可能暴露“用了哪个 MCP server”，但本地纪要本来就展示实时轨迹，且不新增外发面。

### 3.4 展示与导出

- 席位卡摘要行（`PantheonRecord.tsx:1295-1302`）在 `seatUsageLabel` 后追加：
  `6 次工具（4 成功 / 1 失败 / 1 未回报）`；
- title 悬浮补充：重复失败“同一工具连续失败 3 次：命令”；不列参数；
- 导出 `export.go` 的“各席观点”每条陈词后加一行：
  `> 宿主观察：6 次工具调用（4 成功 / 1 失败 / 1 未回报）；重复失败：命令 ×3`；
  只在 `ToolEvidence != nil` 时写；nil 时写“该席不上报工具轨迹”，不写“0 次”。

### 3.5 不变量与测试

1. 证据只来自事件，不接受模型在正文里自报的数字；
2. 证据不进入任何席位提示词（避免模型为了“好看”刷工具）；
3. 同一 `AttemptID` 的事件重放两次，计数一致；
4. `Calls/OK/Failed` 与事件流逐条对账；`Calls < OK+Failed` 视为不变量破坏，测试直接失败；
5. 模型席 `nil`，CLI 席零工具非 nil；
6. 旧 `session.json` 读回零值，导出/前端不崩。

**验收**：`go test ./internal/pantheon` 新增 ≥4 例（计数、失败、重复、模型席 nil）；`tsc -b`；Pantheon 前端定向用例新增 1 例（摘要行）。

---

## 4. E2 派工回执与 A1 透明补完

### 4.1 上游机制

`group-goal-run.ts` 的有界 goal run 里，“派工”不是一句话，而是一步状态：

- coordinator 每轮给 `next/instruction`，worker 步进后必须返回**非空**结果，否则整场 `failed`（`:305-339`）；
- 同一个 `(memberId, 归一化 instruction)` 三次重复 → `blocked`（`:292-298`）；
- worker `busy/parked` → 让 coordinator 改派，累计三次等待耗尽 → `blocked`（`:321-334`）；
- 到上限 → `limit-reached`，不是“悄悄结束”（`:342-344`）。

上游 runtime 把这条做成了一等记录：`Assignment.status/outcome`、`executions`、`result`、`evidence`、`reportedAt`，并有“report 绑 dispatch、一次性、不重复投递”的不变量（`runtime-contract` §2、§7 的 I2/I3/I7）。

### 4.2 我方缺口

A1 把主席派工存进了 `Contribution.ChairAssignments`，并在下一轮用 `chairInstructionFor` 注入提示词（`chair_envelope.go:111-127`），但：

- 没有“这一条派工下一轮有没有被接住”的记录；
- 派工本身在 UI/导出不可见（§1.2）；
- 主席下一次小结看不到**结构化回执**；它最多从 A5 的会话历史里自己回想，不能保证不漏，也不能保证把“失败/跳过”区分出来。

### 4.3 设计：只读派生，不建第二真相

不新增 `Assignment` 表，不新增 `Session.Assignments` 字段。做一个纯函数：

```go
type ChairAssignmentReceipt struct {
    Round         int    `json:"round"`         // 派工所在的主席小结轮
    AgentID       string `json:"agentId"`
    AgentName     string `json:"agentName"`
    Instruction   string `json:"instruction"`
    Status        string `json:"status"`        // spoke | silent | failed | skipped | unavailable | pending
    ContributionID string `json:"contributionId,omitempty"`
    Detail        string `json:"detail,omitempty"`
}

func chairAssignmentReceipts(session *Session) []ChairAssignmentReceipt
```

判定顺序（对每条 `ChairAssignment`）：

1. 在 `session.Contributions` 里先定位该 moderation 贡献的下标，再找该席**在其后**的第一条贡献（按切片顺序，不按 `Round` 比较——迟到输出可能同轮落档）→ `spoke`，带贡献 ID；**这只证明它发言了，不证明它完成了指令**；
2. 没有贡献，但派工之后有该席的 `agent_error` → `failed`，`Detail` 取错误首行（截断，不含完整命令）；
3. 有 `agent_skipped` → `skipped`；
4. 该席在 `session.Participants` 里已是 `unavailable` → `unavailable`；
5. 派工那一轮之后整场已结束、没有下一轮 → `pending`（`Detail` 写“议事在下一轮前结束”）；
6. 其余（下一轮还没跑，或跑了但既无贡献也无事件）→ `pending` / `silent`，按“有没有下一轮”区分。

**关键措辞**：`spoke` 的中文标签是“**已发言（未证明完成）**”，不是“已完成”。回执是**投递回执**，不是完成证书；是否落实由主席读贡献判断。这条必须同时写进 UI tooltip、导出图例、主席提示词，否则读者会把“发言”当成“交差”。

### 4.4 主席提示词注入

`moderationPrompt` 增加 `receipts []ChairAssignmentReceipt` 参数，在第 R 轮小结时传“第 R-1 轮派工的回执”：

```text
上一轮派工回执（只证明该席是否在下一轮发言，不证明完成）：
- alpha：已发言（贡献 ID …）——请读其发言判断是否落实
- beta：失败（原因…）
- gamma：缺席
未落实的派工：重新指派、改成收口动议，或写进风险；不要把“已发言”写成“已完成”。
```

没有派工时整段省略，不占 300 字预算（回执段不计入 300 字限制，与信封同规则）。

### 4.5 A1 透明补完（同一刀的前置）

在做回执之前，先把已有 `ChairAssignments` 显示出来：

- **记录页**：moderation 贡献卡下加“主席派工”块，每条 `AgentID → instruction`；有回执时并排显示状态；
- **导出**：`export.go` 在 moderation 贡献的可见文本后加：
  `**主席派工**：alpha —— …；beta —— …`；有回执时加“回执：已发言/失败/…”。
- **不新增绑定**：这些都在现有 `Session` 载荷里，前端只读。

### 4.6 不变量与测试

1. 回执是纯派生，不落盘；重启/重试/续议后按同一存档重算，结果一致；
2. `spoke` 只认“派工之后该席的第一条贡献”，按存档切片顺序定位，不按时间戳猜；
3. 失败/跳过/缺席必须有事件或参与者状态佐证，不能把“没找到”写成“失败”；
4. 回执不改变席位状态、不改变议事推进、不触发重试；
5. 主席提示词里的回执只用于“改派或收口”，不用于给席位打分；
6. 导出里的派工/回执不包含信封原文（A1 的剥除规则继续成立）。

**验收**：纯函数单测 ≥6 例（spoke/failed/skipped/unavailable/pending/多条派工）；moderation prompt 断言含回执且措辞是“未证明完成”；导出 golden 含派工与回执；前端 1 例渲染。

---

## 5. E3 主席终局信号

### 5.1 上游机制

`group-goal-run.ts:13-15` 把 coordinator 的终局做成四值：

```ts
| { status: "continue"; next: string; instruction: string; detail?: string }
| { status: "completed" | "needs-input" | "blocked"; detail: string }
```

解析规则（`:58-85`）：只认最后一个完整信封、剥除可见文本、JSON 坏/字段不全 → 降级为普通小结而不是吞发言。终局一到，run 就 `finish(status, detail)`（`:276-279`），不会继续烧轮次。

### 5.2 我方缺口

A1 信封只有 `assignments`，没有“整场该怎么走”的显式信号。主席想说“可以收口了”只能在 300 字小结里写一句自然语言，宿主读不到；想说“被阻塞了”也只能写正文，议事继续按轮次跑。

O-E 已经有决策门（`OpenGate/ResolveGate`）与 `openGateInSession`（`gates.go:103-127`），**载体现成**，缺的是“主席把终局信号放进信封、宿主把它翻译成停车”。

### 5.3 设计：信封 v2，终局只停车不越权

信封载荷扩展为：

```json
{"decision":"continue|completed|needs-input|blocked","detail":"一句话","assignments":[{"agentId":"…","instruction":"…"}]}
```

兼容规则：

- `decision` 缺省/空 = `continue`，旧信封逐字兼容；
- `completed/needs-input/blocked` 必须带 `detail`（非空），否则降级为 `continue` + warning；
- 终局状态下仍带 `assignments` → 忽略派工 + warning（终局不该再派新活）；
- 仍只认最后一个完整信封；可见文本剥除规则不变；JSON 坏 → 降级为普通小结 + warning。

宿主语义：

| decision | 宿主动作 | 边界 |
|---|---|---|
| `continue` | 照常进下一轮；派工照 A1 注入 | 无 |
| `blocked` | 本轮结束后 `openGateInSession(reason="主席判定阻塞："+detail)`，run 返回 | 门必须由人 resolve/dismiss；不自动结束、不自动续跑 |
| `needs-input` | 本轮结束后写一条**宿主提问**（`QuestionPayload.AskedBy=""`、`Origin=chair`、`Text=detail`），整场 `waiting_input`，run 返回；回答走 `AnswerSeat`，与 E4 共用 `user_answer` 贡献 | 不伪造回答，不替人填默认；**不走 `AddUserNote`**，否则会把一条回答记成分歧条目 |
| `completed` | **只记录，不越权**：若 `UntilConsensus` 且既有停会条件已满足（全员一致、无 open 分歧、无缺口），循环本来就会停；否则记 warning“主席建议收口，但仍有未闭合分歧/缺口，未提前停止” | 不覆盖 `roundReachedConsensus`、不覆盖 open issue、不覆盖 gaps |

**为什么 `needs-input` 不复用 `AskSeat`**：主席小结结束时，该席已经 `finished`；`finished → needs_input` 不是合法转移（`seatstate.go:26` 的 `SeatFinished` 只自环）。所以 `needs-input` 走**宿主提问**：`AskedBy` 留空、`Origin=chair`，只把整场置 `waiting_input`，不改任何席位状态；回答时 `AnswerSeat` 也不会 rebind 席位。这样既保留“主席在问人”的语义，又不伪造一次席位复活。

**为什么 `completed` 不直接停**：众神殿的红线是“剩余分歧≠无”“不把多数投票当事实”。主席说完成只是一条建议；让它直接跳过既有停会条件，等于把“主席自评”变成“事实”。上游 goal run 可以这样做，因为那里的 coordinator 是唯一裁决者；众神殿的裁决者是证据 + 用户签注。

**固定轮次模式**：用户明确要 N 轮时，`completed` 不改变轮次；只在记录里留一条“主席建议收口”。这是尊重用户设定，不是忽略主席。

### 5.4 不变量与测试

1. 终局信号只能让议事**停着等人**，不能让它“悄悄结束”或“自动续跑”；
2. `blocked` 必须产生一条可读 gate 事件；`needs-input` 必须产生一条 question 事件；没有事件就不算停；
3. `completed` 不得绕过 `roundReachedConsensus` / open issues / gaps；
4. 同一轮最多一道门；已有门时不再开第二道（`openGateInSession` 已守）；
5. 信封坏/字段不全 → 普通小结 + warning，不吞发言、不停会；
6. 终局 + assignments 的组合必须留痕，不静默丢派工；
7. `needs-input` 不改变任何席位状态；回答不 rebind 已结束的主席。

**验收**：解析向量 ≥6（continue/completed/blocked/needs-input/坏 JSON/终局带派工）；run 级测试 3 例（blocked 停车且 gate 可解、needs-input 产生宿主提问且回答进下一轮、completed 在 until-consensus 下不越权）；导出/前端显示 decision。

---

## 6. E4 席位自述提问

### 6.1 上游机制

`runtime-coordinator.ts:72-83` 的解析非常克制：

```ts
/** The question a reply ends on: its last line starts with NEEDS INPUT:, or the line before a closing `Default:` line. */
export function needsInputQuestion(text: string): string | undefined {
  const lines = text.split(/\r?\n/);
  const filled = (from: number) => { let at = from; while (at >= 0 && lines[at]!.trim() === "") at -= 1; return at; };
  const last = filled(lines.length - 1);
  if (last < 0) return undefined;
  const at = /^default:/i.test(lines[last]!.trim()) ? filled(last - 1) : last;
  const line = at < 0 ? "" : lines[at]!.trim();
  if (!NEEDS_INPUT.test(line)) return undefined;
  const question = [line.slice("NEEDS INPUT:".length), ...lines.slice(at + 1)].join("\n").trim();
  return question || undefined;
}
```

三条纪律：

1. **只认最后一行**（或 `Default:` 的上一行），正文中间提到 `NEEDS INPUT:` 不触发；
2. `Default:` 被拆成“问题 + 默认选项”（`runtime/questions.ts:28-33`）；
3. `personOnly` 单独分类“只有人能做”的问题（登录/支付/凭证/本人账户，`runtime/questions.ts:13-26`），用于决定谁能答；在众神殿里不用来决定“谁能答”（都归用户），但可以用来在 UI 上标明“这不是一个普通选择题”。

`ask-cards.ts:253-260` 把它变成问题卡：席位回复末尾的 `NEEDS INPUT:` 被替换成卡片，答案可以来自卡片、聊天里的一句话、另一个席位代答，且“答案在哪里给都算”（`question-routing.ts:2-6`）。

### 6.2 我方缺口

O-E 的 `AskSeat` 只能由 App 绑定调用；席位在外部 CLI 进程里跑完就结束了，宿主没有解析它的最后一行。Orca 第二轮的落地记录已经把“AskSeat 的自动识别”明确列为第二期（`docs/gaea-orca-pantheon-distill-round2-2026-11.md:410`）。这就是 E4 的出处。

### 6.3 设计：轮末停车，不在席位中间打断

外部进程不能在输出到一半时插问（M7 约束），所以 E4 不试图“边跑边问”。时序：

1. 一席返回后，宿主用移植的 `seatNeedsInputQuestion` 解析**最后一行**；
2. 命中：记录一条 `EventQuestion`（`QuestionPayload` 增 `Default`、`Origin` 字段；`Origin=seat` 区分人工 `AskSeat`），席位观测为 `needs_input`；**但不立刻把整场置 waiting_input**，本轮剩余席位继续发言；
3. 本轮结束（所有席收声）后，若存在未回答的席位提问 → `session.Status = waiting_input`，留一条 `phase` 事件“本轮结束，N 席提问等待回答”，run 返回；**主席小结与下一轮都不启动**。检查点放在每轮收声之后、主席小结之前；最后一轮同样先停，再决定是否进汇总；
4. 用户回答（`AnswerSeat`）→ 新增 `PhaseUserAnswer = "user_answer"` 贡献（`AgentID=user`、`Contribution` 增可选 `QuestionID` 字段，旧档零值兼容；不进分歧台账），席位回 `waiting`，整场仍是 waiting_input；
5. 用户点“继续讨论” → `Resume` 从下一轮开始，提示词的 `【用户回答】` 段带入答案；
6. 最后一轮命中提问 → 在汇总前停；Resume 时 `startRound = Rounds+1`，直接进汇总，答案已在历史里。

**为什么在轮末停而不是席间停**：`Resume` 目前从 `CurrentRound+1` 开始；若在席间停，剩余席位会被跳过，轮次完整性被破坏。轮末停保证“每一轮每个可用席位都发过言”，代价是问题要等到本轮结束才阻塞——但事件是即时上屏的，用户不会“看不见”。

**`Default:` 的处理**：不作为自动回答（众神殿不替人做决定），只作为问题卡上的一个一键选项；用户点它等于回答该默认。`personOnly` 命中时 UI 加“只能由你回答”徽标，不改变停车逻辑。

**上限**：同一场最多 3 个未答席位提问；超过的只记录为普通事件并 warning，不再增加停车负担。同一席位同一轮只认第一条，避免一段里复读。

**失败降级**：

- 解析不到/问题为空 → 不产生问题、不改变状态，正文原样入档；
- 问题超长（>2000 runes）→ 截断并在 `Detail` 指向贡献 ID；
- 正文里出现 `NEEDS INPUT:` 但不在最后一行 → 不触发（严格最后一行）；
- 代码块/引用里的 `NEEDS INPUT:` → 不触发（先做 fence/quote 掩码，再解析）；
- 恢复时存在未答提问 → `Resume` 拒绝并提示“还有 N 个问题没有回答”（现有逻辑已守），不替人回答。

### 6.4 与现有门/回答的关系

| 现有 | E4 复用/差异 |
|---|---|
| `AskSeat`（人工） | 复用 `QuestionPayload`/`AnswerSeat`/前端等待面板；`Origin=user` |
| `OpenGate` | 不复用：门是“停整场”，席位提问是“停在该席的问题上”；但都用 `waiting_input` |
| `AddUserNote` | 不复用：插话不绑问题、会开分歧条目；答案必须绑 `QuestionID`、不开分歧条目 |
| `Resume` | 复用；但必须带 `user_answer` 贡献，且拒绝未答问题（现有） |

### 6.5 不变量与测试

1. 只在**最后一行**（或 `Default:` 上一行）认 `NEEDS INPUT:`；中间/引用/代码块不认；
2. 命中不吞正文：`Contribution.Content` 保留原文，问题另存事件；UI 可折叠，但不删；
3. 一轮最多一次停车；多个提问合并为一次停车；
4. 答案 exactly-once：同一 `QuestionID` 第二次回答被拒（`AnswerSeat` 已守）；
5. 答案不触发自动续跑；必须用户显式 Resume；
6. 答案进下一轮提示词，且不被当成“用户插话”开分歧条目；
7. 最后一轮提问在汇总前停，不跳过汇总；
8. 恢复/重启后未答问题仍挡 Resume。

**验收**：解析向量直接移植上游 4 条（末行命中、`Default:`、中间不命中、空问题不命中）+ 3 条本地（代码块、引用、超长）；fake runner 端到端 1 例（两席，一席提问 → 本轮另一席仍发言 → 停车 → 回答 → Resume → 下一轮提示词含答案）；导出含问题与答案；前端等待面板复用。

---

## 7. E5 同位失败连击止损（观察项，不先做）

### 7.1 上游机制

`server/host/loop-guard.ts:140-200`：同一条 shell 命令**以同样方式连续失败**到 `steerAt`（默认 5）时，向运行中的回合发一条纠正提示；只有显式设置 `stopAt` 才终止；而且“一条等待人/等待电脑的回合从不下手”（注释 `:2-4`）。整个 guard **默认关闭**（`:42`：`Off by default: it speaks into a running turn`）。

### 7.2 我方现状

`seatguard.go` 已有三道护栏：120s 无输出、120 次工具预算、5 分钟未开正文。它们拦的是“进程还活着但什么都没发生”，**拦不住“一直在动、但同一条命令反复失败”**——每次失败都有工具事件，看门狗不会响。

### 7.3 为什么先不做

1. 上游自己默认关闭，说明误报/打扰成本高；
2. 合法轮询（等 health check、等构建产物）就是“同一失败重复”，杀掉会误伤；
3. 我们的进程不能在回合中 steer，只能“杀掉 + 重试一次”，语义比上游的“提醒一句”重得多；
4. 没有本地数据证明“重复失败”是当前的主要浪费。第一轮 seatguard 的 138 次实弹调用复核显示，真正的慢是“工具次数多/单次工具 30s”，不是“同一失败循环”。

### 7.4 复活条件

E1 落地后，回放现有会话（`session.json` 的 `tool_result` 事件）统计：

- 有 ≥3 场出现“同一 `(工具, 输入)` 连续失败 ≥5 次”；
- 且这些场次的 `DurationMs` 或工具次数显著高于同轮中位数；
- 且用户/主席没有从这些失败里得到有效信息（抽样人工看）。

三条同时成立，才以**默认关闭 + 需显式开启**的方式试点；否则永久留在观察池。E1 的 `RepeatFailures/RepeatTool` 就是为这一天准备的免费数据。

---

## 8. E6 / E7 / E8 挂池（写明为什么现在不做）

### 8.1 E6 席位上下文重开

**上游**：`room-context.ts:10-15` 按窗口比例算压缩点（默认 0.9），`:101-112` 的 `takeReset` 只在 token 读数达到阈值或用户请求时让该席重开会话；重开会留一条可见的“已压缩”提示。

**为什么挂池**：

1. 我们只有 DSH/ZCode 上报 token；Codex/Grok 的窗口大小与当前上下文读数没有同形载体，阈值只能猜；
2. A5 刚把“引擎会话续接 + 增量投递”接通，缓存命中与记忆连续性是这个方向的主要收益；在没有“上下文爆掉”的实证前重开，会主动丢掉这些收益；
3. 重开是一次“合法失忆”，必须让用户看见；这需要先定义“失忆点”的产品语言，不是纯后端字段；
4. 现有失败样本里没有“上下文窗口耗尽”这一因，先不造需求。

**复活条件**：某引擎上报 `contextWindow` 与 `usedTokens`（或现有 usage 能稳定折算），且出现 ≥2 场“接近窗口后质量明显下降/调用失败”的实证。

### 8.2 E7 等待回答的存续与提醒

**上游**：`host/person-waits.ts:31-35` 把“回合结束但卡还开着”做成“resting on the person”，`:100-130` 提供收件箱式的 `waiting()` 与跨会话通知。

**为什么挂池**：众神殿目前是单场会话（一个议题一个等待面板）；`waiting_input` 已经在列表/记录页可见（O-E 落地），没有“同时有十个会在等你”的跨会话收件箱问题。等真的出现“多场并行 + 用户漏看”的反馈再做，避免为一个不存在的 inbox 造 UI。

### 8.3 E8 材料来源分级

**上游**：`outside-fence.ts` 给“外部来源的回合”限定可触达的内部路由；`outside-writes.ts` 让“外部来源提议写记忆/公告”必须等人卡片批准；`peer-provenance.ts` 用统一的 `[From … · … · authority]` 标注来源。

**为什么挂池**：众神殿的材料是用户显式给的 `EvidenceItem` 与工作区文件；席位只读，不能写记忆/公告，也没有“外部 webhook 驱动”的入口。把 provenance 做成类型化字段是产品级安全设计，不是议事机制；当前没有“外部来源要在议会里行动”的场景。**不因为上游有就搬**——这正是第一轮“取道不取器”的纪律。

### 8.4 已调研但不立项的轮次机制

这些机制都读过源码，结论是“众神殿已有更简形态，或形态不匹配”，列出以免下一轮重复调研：

| 上游文件 | 机制 | 为什么不搬 |
|---|---|---|
| `server/room-holds.ts:1-168` | 用户 `@某人 stop/pause` 的指令解析：引用/代码掩码、mention 邻接、多语言停词、被 hold 消息最多 20 条 | 众神殿的停止是显式按钮（取消/跳过/门），不需要从自然语言里猜“这句 stop 是不是命令”；猜错会把一次正常讨论停掉。M7 已定“外部进程不能拦腰插话”，hold 的“先记消息、恢复再投递”在我们的轮次模型里没有对应物 |
| `server/room-queue.ts:1-51` | 同一房间串行队列、位置/前驱、并行幻觉的提示词 | 众神殿的轮次循环本身就是串行（独立开场并行是显式协议），队列位置对用户没有新增信息 |
| `server/room-member-lock.ts:1-58` | 席位在“开跑前”先占锁，防止唤醒的等待者都读到空闲 | 我们的轮次循环按 `roundOrder` 逐个调用席位，同一席在同一轮只出现一次；没有上游那种“多个等待者被同时唤醒”的并发入口。不搬锁，也不声称已有等价物——形态不同 |
| `server/room-turn-timeout.ts:51-165` | 回合无进展看门狗：等人/等电脑时时钟暂停，工具在飞时放宽静默上限 | 我们的席位调用是“一次 CLI 进程”，提问发生在调用结束之后；没有“回合中等人”的窗口。seatguard 的 120s/5min 已覆盖进程层卡死；“总上限不挂钟”是零成本窄增量，留观察 |
| `server/room-post-budget.ts:50-104` | 发言预算：重复、环形、发送者速率、无人回应升级、熔断冷却；**拒绝不记账** | 众神殿没有“席位自行往房间发帖”的路径；发言由宿主按轮次调度，A2a 环检测已作为纯函数在册。`拒绝不记账` 这条纪律值得记住，但没有可接的入口 |
| `server/host/turn-order.ts:1-23` | 人的工作优先于 bot 的续跑 | 众神殿没有“人/机器人共用同一线程队列”的模型；用户插话已经以贡献身份入档，不参与调度排序 |
| `server/host/room-transcript.ts:25-78` | 房间窗口的 provenance（digest/webhook/peer/reaction 标注） | 这是房间消息来源标注，不是议事贡献；E8 已把 provenance 整类挂池，不在这里重复 |

**结论**：这一组不是“没读到”，是“读到了但形态不匹配”。第二轮不为了覆盖面把房间调度器搬进圆桌。

---

## 9. 不立项与红线（不重复立项）

| 项 | 为什么不做 | 证据/边界 |
|---|---|---|
| **B1 Jev 式发言路由** | 红线 #2：不让模型自评 suitability 决定默认发言权；快分类器仍是“路由决定”，0.6 阈值只在上游的 53 条标注上验证过，未在我们的异构席位验证 | `docs/gaea-pantheon-v2-roadmap-2026-11-03.md` 明确不抄 #2；第一轮 B1 挂池 |
| **A2b 停滞判据上线** | 统计门槛未闭合（预冻结标签、非本场双标 κ≥0.6、n≥9、N_fire≥3）；先做只会得到不可解释的告警 | 第一轮 §4/§5；不重开 |
| **A4 神格卡可移植** | 上游 preset/team 文件是**安全 allowlist**：不带模型/文件夹/审批级别/MCP/浏览器，文件导入的技能默认关闭（组织预置才默认开启），地址中的密钥被抹掉；这是高安全面产品，不是议事机制 | `upstream docs/{presets,team-sharing}.md`；第一轮挂池 |
| **B4 MCP 控制面** | 上游 MCP v1 明确不能批准/记住授权/删除数据/导团队/改凭证；众神殿当前 Wails 绑定已够用，没有外部客户端场景 | `upstream docs/{mcp-server}.md:8-15`；第一轮挂池 |
| **B3 定时圆桌 / B5 语音旁白** | 需求拉动；没有用户场景前不做 | 第一轮挂池 |
| **A2c 升级为阻断** | 圆桌没有 reassign 队列；主席重复派工可能是催办，阻断会制造“主席一意孤行就停会” | §1.1；A1 契约“只告警” |
| **工具选择 / 审批级别** | 众神殿是只读议事；引入每席写权限/审批级别会直接违反红线 #6（议事阶段不自动写盘、不运行副作用命令），并把安全面从“四席只读”扩大到“四席可写” | 第一轮 §0/§9；`upstream docs/{tool-selection,approval-levels}.md` |
| **MEMORY.md / 每席持久记忆** | 众神殿是 **episodic** 的：一场议事的记忆就是 `session.json` 存档；给每席加持久记忆会造第二真相，也让“这场说了什么”与“它记得什么”分叉 | B2 已用“跨场立场 brief”纸面契约覆盖跨场需求；第一轮挂池 |
| **Sentinel / proactive routines / voice** | 常驻健康扫描、主动例行、语音是“always-on bot”的能力，不是议事厅 | `upstream docs/{sentinel,proactive-routines,voice-mode}.md`；不立项 |

**一句话**：第二轮不把众神殿变成“更能干的 Agent 工作台”，只把它变成“**更可信的议事记录**”。

---

## 10. 施工顺序与门控

### 10.1 顺序

| 阶段 | 交付 | 改动面 | 退出条件 |
|---|---|---|---|
| **P0-a** | A1 透明补完 + E2 回执 | `export.go`、`PantheonRecord.tsx`、`moderationPrompt`、新增 `chair_receipt.go` | 派工与回执在记录/导出可见；纯函数单测全绿；不改停车点/权限 |
| **P0-b** | E1 证据摘要 | `types.go`、`manager.go`（落档时算）、`export.go`、`PantheonRecord.tsx` | 计数与事件对账；模型席 nil；不改提示词 |
| **P1-a** | E3 信封 v2 | `chair_envelope.go`、`manager.go`（run 停点）、`prompts.go`、前端 | `blocked/needs-input` 停车可解；`completed` 不越权；旧信封兼容 |
| **P1-b** | E4 席位提问 | `prompts.go`（解析）、`gates.go`（记录/答案）、`manager.go`（轮末停）、前端 | 解析向量全过；轮末停车；答案进下一轮；不吞正文 |
| **P2** | E5 观察 | 只回放 E1 数据，不改代码 | 三条复活条件同时成立才试点 |

### 10.2 门控（沿用第一轮的证据门，不用日历）

| 门 | 最小交付 | 通过/停止 |
|---|---|---|
| **G0 证据冻结** | 本文件：钉版、增量核对、四类证据、源码锚点 | 每个事实有 `file:line` 或标“本地再设计” |
| **G1 契约** | E1/E2 的类型与派生规则；E3/E4 的解析、状态转移、失败降级、隐私、回滚 | 无失败出口不上线；不新增第二真相 |
| **G2 机械回放** | 现有 10 场会话回放 E2 回执/E1 摘要，检查不变量与异常 | 计数对不上、回执归因错、旧存档崩溃 → 停 |
| **G3 假 runner 端到端** | E3/E4 用假 CLI 跑完整两轮（不调真模型） | 停车/回答/续跑/导出全链路通过 |
| **G4 真机冒烟（可选）** | 一场真实议题，只读；验证 UI/导出与真引擎工具事件 | 只验证“记录是否真实”，不宣称“会议效果提升” |

**显式出口**：E3/E4 若在 G3 出现“误停车/吞发言/答案丢失”任一，退回 P0，只交付 E1/E2；不影响第一轮已落地能力。

---

## 11. 验收标准

1. **证据可核对**：任一有工具事件的席位陈词，`ToolEvidence.Calls/OK/Failed` 与事件流逐条对账一致；模型席显示“不上报”，不显示 0。
2. **派工可见**：主席每次派工在记录页与导出都能看到“谁、什么指令”；不再只存在于 `session.json`。
3. **回执诚实**：`spoke` 只表示“下一轮发过言”，UI/导出/提示词三处措辞一致；失败/跳过/缺席各有事件佐证。
4. **终局不越权**：`completed` 不绕过全员一致/open issues/gaps；`blocked/needs-input` 只停车，必须人来解。
5. **提问不吞正文**：`NEEDS INPUT:` 只在最后一行触发；正文保留；代码块/引用不触发；问题超长截断留指针。
6. **答案 exactly-once**：同一问题第二次回答被拒；答案不自动续跑；答案进下一轮提示词且不开分歧条目。
7. **旧档兼容**：第一轮 `session.json` 读回不崩；旧信封（只有 assignments）按 `continue` 处理。
8. **无新权限**：不新增写盘/执行/网络权限；不新增席位状态；不新增持久化表。
9. **测试**：`go test ./internal/pantheon` 全绿（新增 ≥12 例）；`go vet`；`tsc -b`；Pantheon 前端定向用例新增 ≥3 例；本文件已在 `docs/README.md` 登记且无新增孤儿/悬空引用（仓库既有 guard 欠账是旧账，不是本轮新增，不在本刀范围）。
10. **不抬版本、不 commit**：按项目惯例，落地批由用户拍板后再提交。

---

## 12. 风险与反例

| 风险 | 反例 | 缓解 |
|---|---|---|
| **解析吞发言** | 席位正文引用提示词里的 `NEEDS INPUT:` 示例，被误判成提问 | 只认最后一行/`Default:` 上一行；先掩码代码块与引用；命中也不删正文 |
| **误停车** | 席位把 `NEEDS INPUT:` 当修辞，每轮都写 | 同场最多 3 个未答问题；同一轮同席只认第一条；UI 可一键用默认答案 |
| **证据变成评分** | 主席/用户看到“工具多=更认真”，开始按次数投票 | 证据不进入任何席位提示词；UI 文案写“宿主观察，不是质量分”；导出图例同 |
| **回执被读成完成** | 主席把 `spoke` 当 `completed`，不再追问 | 三处措辞统一“已发言（未证明完成）”；提示词明确要求读贡献 |
| **主席滥用 blocked** | 每轮都说“被阻塞”，把会拖死 | 门必须人解；`blocked` 不结束、不自动续跑；连续 blocked 进 warning（可选） |
| **completed 越权** | 主席说完成，但还有 open 分歧，宿主直接汇总 | 只在既有停会条件满足时生效；否则 warning，不改变流程 |
| **跨引擎事件形状不一** | Codex 的 `item.completed` 可能不带 `exit_code`，ZCode 用 `result` | 只认 `ToolStatus` 三值；`started` 无结果计“未回报”；每个适配器一条测试 |
| **旧档/重试漂移** | 重试后旧 attempt 的事件被新 attempt 复用，证据串写 | 按 `AttemptID` 过滤；为空退回 `CallID`；回执按贡献顺序不按时间戳 |
| **提示词膨胀** | 回执段 + 派工段挤占 300 字小结 | 回执不计入 300 字；无派工则整段省略；最多 8 条派工（A1 已限） |
| **隐私** | 证据把命令/输出里的 key 带进纪要 | 只存计数与工具名；不存输入/输出/错误原文；重复失败只存工具名 |

---

## 13. 源码索引

### 13.1 上游（OpenMausBot `1bc4749`）

- 有界 goal run / 终局信封：`server/group-goal-run.ts:7,13-15,58-85,112-143,162-164,217-344`
- 指派回执/状态：`server/runtime/types.ts:50-75,80-172,327-355`；`server/runtime/tree.ts:45-63`；`upstream docs/{runtime-contract}.md` §2/§4/§6/§7
- 工具证据：`server/turn-outcome.ts:100-135`；`server/runtime/envelope.ts:148-160`
- 席位提问：`server/runtime-coordinator.ts:33,72-83`；`server/host/ask-cards.ts:16-23,64-68,105-126,253-260`；`server/runtime/questions.ts:13-33`
- 回答路由：`server/host/question-routing.ts:2-6,53-131`
- 循环守卫：`server/host/loop-guard.ts:1-4,42-52,132-200`
- 上下文重开：`server/room-context.ts:10-15,101-112`
- 等待人：`server/host/person-waits.ts:31-35,100-130`
- 来源边界：`server/host/outside-fence.ts:7-73`；`server/host/outside-writes.ts:1-85`
- 边界文档（只作边界证据，不搬）：`upstream docs/{presets,team-sharing,mcp-server,tool-selection,approval-levels,memory,sentinel,proactive-routines,voice-mode}.md`

### 13.2 我方（Gaea 众神殿）

- 类型：`internal/pantheon/types.go:195-207`（SeatUsage）、`:254-259`（QuestionPayload）、`:336-381`（Contribution）、`:511-539`（Session）
- 运行循环：`internal/pantheon/manager.go:1108-1422`（轮次/停车点/主席小结）
- 重试/工具事件：`internal/pantheon/manager.go:1599-1625`、`:1719-1725`、`:2507-2554`
- 门/提问：`internal/pantheon/gates.go:103-127`、`:145-268`、`:334-485`
- 提示词：`internal/pantheon/prompts.go:97-112`（陈词契约）、`:573-632`（主席小结）
- 信封/重复派工：`internal/pantheon/chair_envelope.go:36-90`、`:111-164`
- 导出：`internal/pantheon/export.go:256-281`
- 前端：`frontend/src/pages/pantheon/PantheonRecord.tsx:273-293`（席位账目）、`:1295-1302`（摘要行）
- 前情与门控：`docs/gaea-pantheon-openmausbot-distill-2026-11-07.md`；`docs/gaea-orca-pantheon-distill-round2-2026-11.md:410`（E4 出处）；`docs/gaea-pantheon-v2-roadmap-2026-11-03.md`（红线）；`docs/gaea-pantheon-seat-hang-cache-2026-11.md`（seatguard 实弹基线）

---

## 14. 已拍板（2026-11-07 晚，随实施确定）

1. **路线**：E1+E2+E3+E4 一次落地；E5 只做观察数据；E6/E7/E8 挂池。
2. **E2 回执语义**：采用 `spoke`（已发言）而不是“必须写回应派工 ID 才算回执”。理由：后者要给派工编号、加一条提示词协议，且模型仍可敷衍；先把“发没发言”做成机械可核对的事实，是否落实交给主席读贡献判断。
3. **E3 `completed`**：只在 `UntilConsensus` 下作为建议；固定轮次模式不改变用户设定的轮数；任何情况下都不绕过全员一致/open issues/gaps。
4. **E4 停车点**：轮末停（保轮次完整），不在席间打断/跳过剩余席位。
5. **E4 未答上限**：同场最多 3 个未答席位提问；同席只认第一条。
6. **E5**：留在观察池，不因连续失败终止席位；`RepeatFailures/RepeatTool` 已随 E1 记账，三条复活条件见 §7.4。

---

## 15. 落地记录（2026-11-07 晚）

范围：E1 / E2 / E3 / E4 一次落地；E5 只落观察数据（`RepeatFailures`）；E6/E7/E8 未动。未 commit、未抬版本。

### 15.1 后端

- `types.go`：新增 `ToolEvidence` / `ToolCount`、`Contribution.ToolEvidence/ChairDecision/QuestionID`、`PhaseUserAnswer`、`QuestionPayload.Default/Origin`、`ChairDecision`。
- `evidence.go`（新）：按 `AttemptID`（为空退回 `CallID`）从工具事件压出 `Calls/OK/Failed/RepeatFailures/RepeatTool/Tools`；`toolEvidenceLine`；`hasCapability`。不落命令/输出/错误原文。
- `chair_receipt.go`（新）：`ChairAssignmentReceipt` + 纯派生 `chairAssignmentReceipts`；状态 `spoke/silent/failed/skipped/unavailable/pending`；`receiptStatusLabel` / `formatChairReceipts`。
- `seat_question.go`（新）：`NEEDS INPUT:` 解析（只认最后一行/`Default:` 上一行、拆 Default、掩码围栏与引用）。
- `chair_envelope.go`：抽出 `lastChairEnvelopePayload`；新增 `parseChairDecision`（decision 缺省=continue，终局必须带 detail，旧信封逐字兼容）。
- `observe.go`：新增 `rebindSeatLockedWithOrigin`，让“席位自述提问”这种控制动作能声明 `OriginEvent`，不冒充用户动作。
- `gates.go`：`AnswerSeat` 追加 `PhaseUserAnswer` 贡献（绑 `QuestionID`，不进分歧台账）；新增 `recordSeatQuestion`（同场最多 3 个未答、同席只认第一条）、`askHost`（`AskedBy=""`、`Origin=chair`）、`pendingSeatQuestionsLocked`、`pauseForSeatQuestions`。
- `manager.go`：讨论/主席贡献落档时附工具证据；席位陈词末尾解析提问并在轮末停车；主席小结收上一轮派工回执；`blocked` 开 gate、`needs-input` 开宿主提问、`completed` 只记录不越权。
- `prompts.go`：主席小结补“派工回执”段与终局信封说明；讨论 prompt 补【用户回答】指引；陈词契约补 `NEEDS INPUT` 必须是最后一行；历史分发把 `user_answer` 与插话分开。
- `seat_history.go`：`【用户回答】` 与 `【用户插话】` 分标签。
- `export.go`：每条陈词附“宿主观察：… 次工具…”；主席小结附“主席派工与终局”及回执。
- `agents.go`：Grok 办公席位在具备工具流时声明 `tool-trajectory`，E1 才有证据来源。

### 15.2 前端

- `gaea/lib/types/pantheon.ts`：补 `Question.default/origin`、`Contribution.chairAssignments/chairDecision/toolEvidence/questionId`、`PantheonChairAssignment/ChairDecision/ToolEvidence/ToolCount`。
- `pages/pantheon/chairReceipts.ts`（新）+ `chairReceipts.test.ts`：从 `contributions/events` 派生回执的 TS 镜像；不落盘、不新增绑定；与 Go 纯函数同口径，测试覆盖 spoke/skipped/failed/unavailable/pending。
- `PantheonRecord.tsx`：席位摘要行与 title 加工具证据；主席小结下渲染“主席终局 / 主席派工 + 回执”；新增 `chairDecisionLabel` 与证据标签。
- `pages/pantheon.css`：`.pantheon-chair-block` / `.pantheon-chair-decision` 样式。

### 15.3 验证

- `go test ./internal/pantheon -count=1` → ok（约 16–20s）。
- `go vet ./internal/pantheon` → 清。
- `CGO_ENABLED=0 go build ./...` → 清（本机无 gcc，默认 cgo 构建会报 `gcc not found`，是环境既有约束，不是本次改动）。
- `go test ./internal/app -run Pantheon` → ok。
- `npx tsc -b` → 零错。
- `npx vitest run src/pages/pantheon` → 65 passed（含新增 `chairReceipts.test.ts` 2 例）。
- `node scripts/check-docs.mjs` → 仓库既有欠账仍 FAIL（12 份旧孤儿 + 35 处旧悬空）；本轮两份文档已登记，且不再新增悬空引用。

新增测试：`tool_evidence_test.go`（3）、`chair_receipt_test.go`（4）、`seat_question_test.go`（8 向量）、`chair_envelope_test.go`（+8 终局向量 + 回执段）、`round2_manager_test.go`（blocked 开 gate / needs-input 宿主提问 / 席位提问轮末停车+回答进下一轮）、前端 `chairReceipts.test.ts`（2）。

### 15.4 有意差异与已知边界

- **E3 `needs-input` 走宿主提问，不走 `AskSeat`**：主席小结结束时该席已 `finished`，`finished → needs_input` 不是合法转移；宿主提问 `AskedBy=""`，回答不 rebind 已结束的主席。终局信号只在**非最后一轮**的主席小结里生效（最后一轮本来就不做轮间小结）。
- **E4 轮末停车**：不打断正在跑的席位，也不跳过本轮剩余席位；同场最多 3 个未答席位提问、同席只认第一条；`Default:` 只是问题卡上的一键选项，不自动回答；解析掩码围栏与引用，但**不处理四空格缩进代码块**（已知窄边界）；靠陈词契约要求 `NEEDS INPUT` 必须是最后一行。
- **E1 只记计数与工具名**：不落命令、输出、错误原文，因此“同一失败”只能显示工具名与次数，看原文要点开实时轨迹；`nil`（不上报）与非 nil 零值（上报了、0 次）严格区分。
- **E2 `spoke` ≠ 完成**：回执只证明下一轮发过言；UI/导出/提示词三处统一写“已发言（未证明完成）”。前端与后端各有一份派生实现（同一存档、不持久化），两边都有测试。
- **E5 未启用**：只落 `RepeatFailures/RepeatTool` 数据，不因连续失败终止席位；三条复活条件见 §7.4。
- **无新增权限/绑定/表**：不新增 Wails 绑定、不新增持久化表、不新增席位状态；旧 `session.json` 零值兼容，旧主席信封（只有 `assignments`）按 `continue` 处理。
- **并行工作树说明**：本工作区另有并行会话在改 Grok 共享上下文（移除 `GrokHistoryBackend`/Seed，改走 `historyFullWindow`）与 Codex `-C` 规格；这些不是 E1–E4 的改动，本记录未把它们算作本轮交付。E1 对 Grok 仍成立：`StreamGrokWithTools` 通道在，Grok 适配器据此声明 `tool-trajectory`。

---

## 16. 结论

OpenMausBot 的第一轮可搬物是“**把会开起来、把上下文接住**”；这一轮真正值得搬的是它**不相信“看起来完成了”**的那条链：报告要带宿主观察、派工要有回执、终局要是类型化的、席位提问要能被接住。

但众神殿不是 goal run，不能把“主席说完成”当事实，也不能把“工具多”当质量。所以这四刀都只做**可核对性**：让记录能回答“谁说了什么、宿主看见什么、谁没有接住、停在哪里”，而不是让议会更会自动开工。**证据先于控制，收口先于扩张**——这是第二轮的唯一排序。

---

*本文为调研与计划，不含代码改动；上游钉版 `1bc4749`，本地克隆 `C:\AI\tmp\OpenMausBot` 保持只读；未 commit、未抬版本。*
