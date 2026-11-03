# Orca × Gaea 众神殿机制蒸馏（2026-11-02）

> **源**：`stablyai/orca`，浅克隆提交 `8cd6a7e5ee77c4ed04e78a3e924f5fc83a366bbf`
> （2026-11-02），`package.json` 版本 `1.4.214`，MIT License。
> **核验范围**：README、`AGENTS.md`、站点文档、状态/编排/会话源码与关键测试；
> 未逐文件审计约 42,843 个工作树文件。
> **纪律**：取道不取器。Orca 是“把活可靠地做完”的 Agent 工作台，众神殿是
> “让不同判断被相互校正后再收口”的议事厅；两者相邻，但不是同一产品。
> **实施状态（2026-11-02）**：P0 的 O-A / O-B / O-C 已落地；O-D 起的决策门、
> 写执行与 daemon 仍按门控后置。

## 0. 结论

Orca 值得蒸馏，但不值得整体移植。

- 它对本地异构 Agent 的产品抽象比当前众神殿更完整：一个席位不是“模型名”，
  而是 **Agent CLI + 终端/Pane + 工作树 + 可恢复会话 + 当前调用身份**。
- 它最硬的价值不在多开终端，而在几条反自欺纪律：状态必须带证据来源；重试必须
  是新的 Attempt；完成必须命中当前 Dispatch；过期租约不等于进程已死；进程退出
  但未交报告只能算“未验证完成”。
- 众神殿已有独立开场、多轮讨论、上下文分栏、共识检测、跳过/重试、小结算和人工
  签注，这些决策协议部分比 Orca 更贴近当前目标，不应被 Orca 的 Task/DAG 模型替换。
- Orca 不解决“哪种判断更可信”。它的文档与源码核心都是任务分派、终端监督、消息、
  工作树和恢复，没有众神殿式的立场演化、证据核验、分歧保全与协作综合。
- Orca 默认给 Agent 预置 `--dangerously-skip-permissions` /
  `--dangerously-bypass-approvals-and-sandbox` 一类完整自治参数，并把“工作树本身”
  当沙箱。这个模型能把并行开发做顺，却不能证明文件系统、网络、凭证与 OS 进程已隔离。
  众神殿必须继续 fail-closed、只读优先，不能照搬。

一句话定位：

> **Orca 负责让一支 Agent 队伍可靠地开工；众神殿负责让一支 Agent 队伍可靠地形成
> 一个可追溯、可保留异议的判断。Orca 补的是执行真相层，不是议事协议。**

## 1. 产品与仓库形态

Orca 自述为 “The AI Orchestrator for 100x builders / Next-gen IDE for parallel
agentic development”。实际形态是一个完整 Electron IDE：

- 每个任务一份真实 `git worktree`，不同 Agent 不共享工作文件；
- 任意 CLI Agent 都可在终端中启动；
- Agent Dashboard、Agents feed、通知、移动 companion、SSH/remote runtime；
- SQLite 编排状态、PTY daemon、会话恢复、休眠、原生会话历史；
- CLI 可反向驱动 Orca，并支持带监督的多 Agent Run。

源码体量说明它是产品级 monorepo，而不是一个可轻量嵌入的“多 Agent 库”。因此适合
提炼协议和不变量，不适合把 Orca 当众神殿的后端依赖。

### 1.1 两者不是正面同构竞争

| 维度 | Orca | Gaea 众神殿 |
|---|---|---|
| 核心对象 | worktree / terminal / task | 议题 / 轮次 / 立场 / 纪要 |
| 目标 | 并行完成并交付工作 | 独立研判并形成建议 |
| 主要并发 | Agent fleet 并行 | 圆桌顺序发言，后位可见前文 |
| 上下文 | 每个 worktree 独立 | 每席独立上下文 + 公共观点窗 |
| 写权限 | 默认完整自治，worktree 隔离工作文件 | 强制只读，禁止副作用 |
| 完成判据 | 当前 Dispatch 的 `worker_done` | 有效陈词 + 汇总 + 用户签注 |
| 少数意见 | 不是核心 | 必须保留 |
| 失败处理 | 状态机、租约、恢复、资源释放 | 单席失败隔离、跳过、重试、残影丢弃 |

产品判断：Orca 抢占的是“本地 Agent 编排器”位置；众神殿的机会仍是
“有证据、有分歧、可签注的审议记录”。最多做可选互操作，不应把众神殿降级为
Orca 的一个任务面板。

## 2. Orca 的核心模型

Orca 的编排文档把系统拆成：

```text
Run（持久命名空间 + coordinator inbox，本身不调度）
  └─ Task（逻辑工作项）
       └─ Dispatch（Task 在某终端上的一次 Attempt，也是完成权威）
            ├─ worker（执行过程）
            ├─ message（status / question / escalation / worker_done / heartbeat）
            └─ observations（process / turn / liveness / artifact / report）
```

关键不是这五个名词，而是它们各自只拥有一种真相：

- **Task** 表达“要做什么”，不冒充某次执行；
- **Dispatch** 表达“这一次是谁在做”，重试必须指向旧 Dispatch；
- **Message** 表达“谁向谁报告”，不冒充运行时事实；
- **Observation** 表达“从什么来源看到了什么”，不冒充完成；
- **Decision gate** 表达“这里必须由人决定”，显式阻塞下游。

众神殿不需要完整照搬 DAG，但可以把当前较扁的
`Session → Participant/Event/Contribution` 补成：

```text
CouncilSession
  └─ Round（议题的逻辑阶段）
       └─ SeatCall（某席位在这一轮的一次调用）
            ├─ AttemptID（每次实际启动；重试/new process 必须变化）
            ├─ Observation（start / delta / tool / exit / error）
            └─ Contribution（仅被接受的有效陈词）
  └─ Verdict → SignOff
```

这样“Agent 名”与“具体哪一次调用”不会混为一谈。

## 3. 八条值得吸收的机制

### M1. Agent 身份必须展开为会话身份

Orca 文档定义得很清楚：一个 agent session 是
“one CLI agent running in one terminal in one worktree”；恢复时保存的是 provider
session id、cwd、launch arguments、私有环境和 pane 身份，而不是只记一个 Agent 名。

这比当前 `Participant.AgentID` 多了四层信息：

1. 哪个 Harness / CLI；
2. 哪个版本与启动参数；
3. 在哪个工作空间；
4. 哪一次 provider session / process attempt。

**众神殿判定**：吸收。至少把“席位”与“调用”解耦。未来做恢复、跨工作树执行或
移动端接管时，没有这层身份一定会把旧进程输出记到新调用上。

### M2. 状态不是布尔值，而是带来源和新鲜度的观察

Orca 的组合状态只有四类：`working / blocked / waiting / done`。但它另外保存：

- 主 Agent 自己的状态与子代理名册分开；
- `hook / osc / title / process / launch / orchestration / structured` 七类证据源；
- `authorityId / incarnation / revision / observedAt` 排序信息；
- `restoredUnconfirmed`，恢复出来的旧状态不得冒充“仍然活着”；
- freshness 与 stale 窗口，静默 hook 最终降为不可确认；
- 状态历史与回合起止时间。

这解决的是一个很常见的问题：**“绿点”可能来自 Agent 本身，也可能只是最后一条
旧终端的 snapshot。**

**众神殿判定**：吸收原则，不搬七种来源。当前 Adapter 直接持有 stdout / tool
callbacks，主要来源应是 `runner / stream / tool / app_recovery / user`；记录每个
事件属于哪次 Attempt，切进程时 bump incarnation。远端或终端抓标题以后再考虑。

### M3. 完成、退出和“看起来完成”必须分开

Orca 的 Attempt 投影不会把进程退出直接当成功：

- 有被接受的 `worker_done`，才得到 `succeeded / failed`；
- 进程结束但没有有效报告，得到 `finished_unverified`；
- 活性未知，得到 `outcome_unknown`；
- 同一 Task 还有活动 sibling 时，Task 结果不提前结案；
- 同一 facet 只取最新 sequence；
- 完成状态是合法状态机里的终态，不能从 `completed` 再转回旧状态。

众神殿已有类似防护：ZCode 增量只有见到回合完成事件才重组为陈词；跳过/重试后的
残影会被丢弃；汇总失败有 fallback warning。但当前持久事件没有 Attempt 身份，
“完成/失败”的证明主要靠实时判定，重连或未来异步执行时证据不够硬。

**众神殿判定**：吸收。为每次实际调用生成 `SeatCall.Ref` 与 `AttemptID`；旧
Attempt 的 late delta/tool/error 只能进入审计边带，不能更新当前席位状态、贡献或
最终小结。无明确 CLI 完成门时，结果标为 `finished_unverified`，而不是静默当完整。

### M4. Dispatch 是完成权威，单写者租约必须防进程复用

Orca 的 completion settlement 要求 `taskId` 与 `dispatchId` 同时命中，迟到重试报
告会因 `task_dispatch_mismatch / stale_dispatch` 被拒。Dispatch 行同时保存
`retry_of_dispatch_id`、`process_incarnation`、`consumer_generation` 与终止原因。

对多个消费者/客户端，Agent session lease 采用单写者：

- 租约过期只说明“没续租”，不证明进程停止编辑或消费 token；
- PID 存在时还要验证 start time / spawn token；
- 不可验证的 owner 按“可能还活着”处理；
- crash 后若只有同 PID 存活而原生 stdio 已丢失，不能直接重接，只能进入 recovery；
- 恢复完成前不授予第二个 writer。

这是众神殿最该学的并发纪律。当前 `activeAgentCall` 以调用指针识别残影，对单进程
顺序圆桌足够；一旦有远端 relay、应用重启续跑或多个窗口共享同一会话，仅靠内存指针
与 `agentId + round` 不再足够。

**众神殿判定**：P0/P1 吸收。无需做完整 lease，只需接受四条最小不变量：

1. 每轮每席有稳定 `CallID`；
2. 每次重试生成新 `AttemptID`；
3. 所有输出/工具/错误事件携带 AttemptID；
4. 只有当前 AttemptID 可提交贡献，旧 ID 一律拒绝或标审计。

### M5. 工作树隔离适合“执行”，不是“审议”的默认答案

Orca 的强设计是每个 Task 一份独立 worktree，从共同 base ref 启动，完成后按 diff
审查、选择结果并删除失败分支。它还把 gitignored 的共享目录与 `.worktreeinclude`
分开：大依赖/缓存可链接共享，`.env` 等本地配置可复制，各自有明确语义。

这对未来“允许多 Agent 实际修改文件”的议会很关键。但当前众神殿的设计目标是
共同证据 + 独立观点 + 只读核验，不是并行施工；此时另建 worktree 只会：

- 增加同步、清理和磁盘成本；
- 让“同样的证据基线”反而更难证明；
- 诱导产品从议事偷偷变成自动施工。

**众神殿判定**：后置，仅用于独立的“议会执行”模式。届时一席一 worktree、同一
base ref、人工验收，不自动 merge。此模式不能藏在现有只读开关后面，必须有独立
授权与风险文案。

### M6. Worktree 不是安全沙箱

Orca 文档明说它默认给每个支持的 Agent 预置完整自治参数，理由是
“the worktree itself is the sandbox”。源码和常规 Git 语义都可以确认：

`git worktree` 隔离的是 Git 工作文件与分支，不隔离：

- 用户主目录、凭证文件、系统网络；
- 其他绝对路径；
- OS 进程、剪贴板、窗口、IPC；
- 外部命令和安装脚本的副作用。

它降低“两个 Agent 同时改同一文件”的冲突，不降低“一个 Agent 越界读凭证或执行
危险命令”的风险。

**众神殿判定**：拒绝照搬完整自治默认值。只读边界需要 Adapter 的原生 sandbox /
permission / 工具白名单与 prompt 同时成立；worktree 只能作为冲突隔离，不能替代
权限隔离。`RunMode` 继续逐席记录，不能把四种不同机制写成一个“已只读”绿勾。

### M7. 外置委派必须显式，避免“一个席位偷偷变成一群人”

Orca 给每个 worker 注入固定 preamble：

- “Only this task, yourself: no subagents or helper workers unless the task asks”；
- 如果 worker 需要提问，必须走 orchestration `ask`，不能使用它自己的本地 TUI
  问题卡，因为 coordinator 看不到；
- `worker_done` 只能发一次，内容必须写清做了什么、发现什么、还剩什么；
- 新 Dispatch 必须按新任务契约执行，不能沿用旧完成语义。

众神殿当前 Prompt 已严格限制写盘、安装、凭证和副作用，但没有明确禁止席位自行
拉起子代理。Codex / Claude 等 Harness 若内部再派工，会出现：

- 实际参与者数量与 UI 席位数不一致；
- 子代理的工具轨迹混进父席陈词；
- 一个席位用更多算力换取“更强论证”的不公平；
- 取消父调用后，子任务是否终止取决于 Harness。

**众神殿判定**：P0 吸收一句明确纪律：

> 本席位必须亲自完成本轮发言；不得派生任何子代理、并行助手或二次委派。若需要
> 额外调查，只能使用当前只读工具并在陈词中说明缺口。

只有以后显式支持“席位委派”时，才允许在记录中展开子席及独立预算。

### M8. 恢复必须证明“还是同一个会话”，不是恢复一张截图

Orca 的会话恢复包含：

- 后台 PTY daemon 持有运行中的 Agent；
- provider session id、cwd、resume argv、私有环境与 effort；
- warm reattach 到同一进程，或 cold resume 到同一 provider 会话；
- 休眠前检查是否仍有未结算 Dispatch、活子代理或移动端 writer；
- provider 会话不可恢复时开新终端，并明确保留旧记录。

众神殿当前在应用重启时把 `running` 记为失败，已比“假装还在跑”诚实；但没有
“恢复同一 Agent 会话继续该席”的契约。短期不必做 daemon，先做可验证的恢复清单：

```json
{
  "seat": "codex",
  "workspace": "...",
  "runMode": "read-only-sandbox",
  "launch": {"version": "...", "argsHash": "..."},
  "providerSession": {"kind": "session_id", "id": "..."},
  "call": {"round": 2, "attempt": 3}
}
```

没有 provider session 身份时，只能重放简报，明确标注“新调用”，不能宣称接续了
旧内部上下文。

## 4. 可直接映射到众神殿的接口纪律

### 4.1 建议的最小身份结构

```text
CouncilSession.ID
Round
SeatID
CallID          # 某轮某席的逻辑调用，可跨重试保持
AttemptID       # 每次实际进程/模型调用都不同
RetryOfAttempt  # 可空
RuntimeID       # 本次应用/执行器实例
Source          # runner / stream / tool / app_recovery / user
ObservedAt      # 生产者时钟
```

落盘兼容方式：给 `Event`、`Contribution`、`Participant` 增可选字段，不重命名旧字段；
旧记录缺失时显示 `legacy / unverified`，不能假装它们有 Attempt 证据。

### 4.2 拒绝矩阵

| 到达事件 | 条件 | 处理 |
|---|---|---|
| delta / tool | AttemptID 不是当前 Attempt | 丢弃直播；如保留则只进审计边带 |
| contribution | CallID 过期或 Attempt 已取消 | 不进入讨论历史 |
| error | 旧 Attempt | 记审计，不改当前席位状态 |
| retry | 原 Attempt 尚未证明退出 | 标 `stop_unknown`，不并发开新 Attempt |
| process exit | 没有 CLIs 完成门 | `finished_unverified`，等待核对或明确失败 |
| app restart | 原状态为 running | `interrupted` / `unknown`，绝不 `completed` |

### 4.3 能力协商，不接受后再忽略

Orca 的 Session 请求把“请求字段在位性”与宿主能力逐项核对，缺失即拒，而不是
接收后静默忽略。众神殿当前 `AgentStatus.Capabilities` 主要用于展示，没有形成
启动前的协议能力门。一个具体例子：图片附件会进入 Codex/ZCode 的启动参数，但 DSH
适配器的 `build` 当前忽略 `images`；若用户只选 DSH，没有明确降级提示时，
“附件已加入议事”就是失真的。

建议为 StartRequest 推导要求集：

```text
images present       → image-input
tools required       → read-only-tool-trajectory
streaming required   → streaming
resume requested     → provider-session-resume
```

逐席核对；不满足时二选一：

1. 从可选席位中排除并解释原因；
2. 用户明确接受“该席仅看文本摘要/图片未送达”的降级，并写入 `RunMode` 与纪要。

## 5. UI 层蒸馏

Orca 的 dashboard 分为 `Needs You / Working / Done / Idle`，并把跨 worktree 的
完成、阻塞提问和工树创建聚合为一条 Agents feed。它对众神殿的直接启发不是照做
Kanban，而是补齐状态语义：

| 当前席位状态 | 可补语义 | 用户动作 |
|---|---|---|
| 候场 | 等待前位 / 等预算 | 无 |
| 正在发言 | 生产中 | 跳过、重试 |
| 需要你 | Agent 提出问题或证据不足 | 回答、补充材料、继续 |
| 已完成 | 有完整陈词 | 查看贡献、引用 |
| 未验证完成 | 进程结束但无完成门 | 人工核对、重试 |
| 失败 | 可证明调用失败 | 重试、跳过 |
| 中断 | 用户取消或应用重启 | 续议、重跑 |
| 未知 | 活性无法证明 | 不显示绿色；给恢复动作 |

保留当前“先看问题，再看席位”和“金只做判断落点”的视觉原则。Orca 的高密度
terminal/status chrome 不适合直接套入众神殿。

## 6. 明确不抄的部分

1. **不做 Agent IDE。** 众神殿不承担终端模拟、编辑器、浏览器、Git review 和
   worktree 管理。
2. **不复制完整自治默认值。** `dangerously-*`、`--yolo` 与
   “worktree 即沙箱”不能进入众神殿默认路径。
3. **不把 Run/Task/Dispatch/Message/Gate 五层一次全建。** 当前圆桌是顺序同步
   执行，先补 Attempt 与状态证据即可；过早建 mailbox 和 DAG 会形成第二套业务状态。
4. **不把 terminal title/OSC 当主状态源。** 能用结构化 Adapter 回调就不用屏幕
   猜测；弱证据必须显示为弱。
5. **不在当前阶段做常驻 daemon。** Orca 的 daemon/lease/recovery 复杂度由移动端、
   远程主机、多客户端和长时间终端会话共同支撑；众神殿还没有这些产品面。
6. **不把并行工作树当成讨论多样性。** 同模型同 prompt 的三个分支，不等于三个
   独立判断主体。
7. **不以 star、测试数量或“每天发版”证明辩论质量。** Orca 的编排能力与稳定性
   值得学；它的多 Agent 协作仍明确标为 Experimental。

## 7. 众神殿刀池

### O-A（P0，小刀，已落地）：Attempt 身份闭环

- `CallRef` 已贯通 `Participant / Event / Contribution`，JSON 字段为
  `callId / attemptId / retryOfAttempt / attempt`；
- `CallID` 按 `session + agent + phase + round` 稳定，重试保持同值；
  `AttemptID` 每次实际进程/模型调用新生成，重试携带 `retryOfAttempt`；
- start / delta / tool / output / error / skip / retry 全部带 Attempt；
- 前端直播键按 Attempt 区分；旧 Attempt 被封口后，即使新尝试已开始，其迟到
  delta 仍不能复活；
- `agentCallAborted` 同时检查 skip、retry 与“已不是当前调用”，进程清理后的
  异步残影也不能更新席位。

验收：同一席位连续两次重试，第一次迟到 delta/tool/result 不进入第二次直播、不写
贡献、不改变席位状态；JSON 导出能看见两次 attempt 身份且没有串写。

### O-B（P0，超小刀，已落地）：禁止隐性派生

- 在 `collaborationRules()` 增“亲自完成本轮，不派生 subagent/helper”；
- 在工具轨迹中已有 `子代理` 类卡片继续保留，但若父席调用未声明派生，只作诊断，
  不把子代理产物计入父席的有效陈词；
- 与现有只读、反重试、来源核验纪律并列。

验收：Prompt 契约测试钉住该句；任何子代理工具轨迹不改变参与席数或 `successful`
计数。

### O-C（P0/P1，中小刀，已落地）：能力门

- `AgentStatus.Capabilities` 已有三类图片通道：`image-input`（Codex / ZCode
  原图）、`image-path-tools`（Grok 读路径后调用本地识图/OCR）、
  `shared-image-summary`（四席共享 app 层生成的识图摘要；DSH 走此通道）；
- `pantheonAttachmentEvidence` 先一次生成共享识图摘要，后端 Start 再逐席核对
  至少存在一条图片通道；三者全无才拒绝，不做“接收后忽略”；
- Composer 对三者全无的席位点名并禁用开议；已选的不兼容席位仍可点击退出；
- Prompt 保留真实的共享识图摘要契约，同时要求具备原图能力的席位按原图核验。

验收：测试用零图片能力席位时，前端显示“不支持图片附件”并禁用开议，直接调用
后端同样拒绝；DSH 因 `shared-image-summary` 仍可正常参会。

### O-D（P1，中刀）：状态证据与未知态

- 参与者状态从自由字符串收敛为类型；
- 状态带 source、observedAt、attempt；
- 增 `needs_input / finished_unverified / interrupted / unknown`；
- 恢复会话只显示“上次已知旧状态”，不得亮绿；
- 超过静默阈值显示“活性未知”，而不是继续“正在发言”。

验收：进程正常退出但没有可识别完成门，最终状态为未验证；应用重启后的 running
会话只进入 interrupted/unknown；UI 用文字、点位与动作共同表达，不只靠颜色。

### O-E（P1，中刀）：决策门

- 每轮允许某席显式提出 `question`，议题进入“需要你”；
- 用户回答只投递给当前 Run，并作为下一轮公共证据；
- 未回答时下游轮次不启动；
- 已有汇总后的用户签注继续作为最终门，不把模型共识冒充人工确认。

这一步才真正靠近众神殿的“议事厅”定位；Orca 提供了 gate 的可靠阻塞语义，但具体
交互应围绕证据与分歧设计，而不是终端邮件。

### O-F（P2，大刀）：写执行模式

仅当出现真实需求时：

- 一席一 worktree，同 base ref；
- 每席只输出 diff + 验证证据；
- 人工选择/合并，不自动裁决；
- 不复制 `.env`/凭证，除非用户逐项批准；
- 删除 worktree 前做 dirty/unmerged 检查；
- 写模式与会话记录独立建模，禁止从只读会话自然漂移。

### O-G（P3，超大刀）：daemon / remote / mobile

需要独立执行器进程、PTY 所有权、租约、跨机时钟、会话恢复和多客户端 fence。
以当前个人工具定位和真实使用量，不先做。

## 8. 成功标准

Orca 的机制是否值得长期保留，不看“功能像不像 Orca”，而看以下指标：

1. 迟到/重试残影造成的错误贡献数 = 0；
2. 任何“完成”都能回答：哪个 Attempt、什么完成证据、哪个执行器；
3. 任何失败/未知都能给出恢复动作，不用用户猜；
4. 图片、工具、流式、恢复等能力不满足时，启动前拒绝或明确降级；
5. 任何席位派生行为都不改变议会人数与有效陈词归属；
6. 只读模式没有写副作用；worktree 若未来出现，只用于冲突隔离且另走授权；
7. 议事纪要保留共识、分歧、证据缺口与少数意见，不因执行层更可靠就宣称判断更正确。

## 9. 源码索引

### Orca

- 产品能力：`clones/orca/README.md`
- Agent 会话与危险默认值：
  `clones/orca/docs\site\content\docs\model\agents-sessions.mdx`
- Worktree 模型：
  `clones/orca/docs\site\content\docs\model\worktrees.mdx`
- 会话恢复：
  `clones/orca/docs\site\content\docs\model\session-restore.mdx`
- 编排模型：
  `clones/orca/docs\site\content\docs\cli\orchestration.mdx`
- Dispatch / Worker 状态：
  `clones/orca/src/main/runtime/orchestration/types.ts`
- Attempt 结果投影：
  `clones/orca/src/main/runtime/orchestration/db/attempt-outcome-projection.ts`
- 生命周期状态机：
  `clones/orca/src/main/runtime/orchestration/db/lifecycle-transition.ts`
- 单写者租约：
  `clones/orca/src/shared/agent-session/lifecycle/agent-session-lease-adjudication.ts`
- 状态来源与排序：
  `clones/orca/src/shared/agent-status-observation.ts`
  `clones/orca/src/shared/agent-status-types.ts`
- Worker Prompt 契约：
  `clones/orca/src/main/runtime/orchestration/preamble.ts`
- 休眠安全条件：
  `clones/orca/docs\site\content\docs\agents\hibernation.mdx`

### 当前众神殿对证

- 数据形状：`internal/pantheon/types.go`
- 圆桌执行与重试：`internal/pantheon/manager.go`
- Prompt 与只读边界：`internal/pantheon/prompts.go`
- Adapter 与逐席 RunMode：`internal/pantheon/agents.go`
- 页面契约：`design-system/gaea/pages/pantheon.md`
- 既有市场判断：`docs/gaea-pantheon-market-research-2026-10-29.md`

## 10. 最终判断

**不接 Orca 为内核，不复制它的工作台；接它的执行真相层纪律。**

第一刀不应是“多 Agent 并行工作树”，而应是 Attempt 身份和“未知也是状态”的诚实
语义。第二刀是显式禁止席位偷偷派生，并补能力门。完成这两步后，再根据真实使用
决定是否进入决策门、写执行、daemon 与远程接管。

这与众神殿已经坚持的方向一致：结果优先、异议保留、失败如实、证据可回看、人工
签注才算正式结论。Orca 让这套方向在执行层更硬，而不是改变它的议事目标。
