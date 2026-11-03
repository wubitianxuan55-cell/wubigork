# 众神殿多智能体议事框架调研：MALLM / Agent Council / ARTEMIS / AgentTeams

> 日期：2026-11-03
> 状态：调研完成；本文件不改运行时代码。
> 范围：围绕“讨论、质疑、收敛、评审，以及讨论后推进任务”的机制蒸馏。
> 取证纪律：优先读上游源码和官方论文/文档；star、功能清单和自述不能替代机制证据。

## 0. 结论

这四个项目不在同一层，不能直接合并成一个“万能多 Agent 框架”：

| 项目 | 真正解决的问题 | 对众神殿的价值 | 对众神殿的定位 |
|---|---|---|---|
| **MALLM** | 把 MAD 拆成可组合、可比较的研究变量 | 协议轴、决策协议、离线评测 | **研究台** |
| **Captain Claw / Agent Council** | 让人能看见、插入并推进一场结构化会议 | 明确议事动作、主持/轮值、投票、纪要 | **交互范式** |
| **ARTEMIS** | 把辩论生成与陪审团评审分开 | 独立评审、异议保留、评审视角 | **读出层警示与素材** |
| **AgentTeams** | 把多 Agent 组织成可执行、可审计的任务团队 | 人类在房间中干预、Manager-Worker、任务账本 | **裁决后的执行层** |

一句话判断：

> **MALLM 教我们“把协议当变量”，Captain Claw 教我们“把动作说清楚”，ARTEMIS 教我们“别让辩手兼任裁判”，AgentTeams 教我们“结论如何进入真实工作”。众神殿不需要接入它们的框架，需要蒸馏这四层职责。**

建议保留众神殿当前定位：**证据议事、保留分歧、人工签注**。不把它改成模型竞技场、法庭或自动施工队。

## 1. 取证快照

| 项目 | 本次核验版本 | 证据级别 | 判断 |
|---|---|---|---|
| MALLM | GitHub `eeec9f5c28961a7349cbb4306a5aa860bdaef6dd`；ACL EMNLP 2025 Demo 论文 | 高：论文 + 源码 + 测试 + 评测数据 | 最适合作为协议设计与评测参考 |
| Agent Council | PyPI `captain-claw 0.4.17`（用户引用版本）+ GitHub 当前 `0.8.5`、`f5f2805c47d39b2cb4ff70f445f61f12cd549948` | 高：前后端源码、持久化模型、运行提示词 | 交互机制成熟；没有公开的受控效果评估 |
| ARTEMIS | GitHub `599e66ff9ca12475f86a8bb41672a5b6279b570f`；PyPI `artemis-agents 2.0.0` | 中：源码可读；论文是 Technical Disclosure，不是同行评审会议论文；自测基准存在口径问题 | 可借鉴职责分离，不宜把其评分当真相 |
| AgentTeams | GitHub `d9366ac43a153d31c73ef4def125e378869b8d85`；官方最新发布说明 `v1.2.5` | 高：架构文档、CRD、实现、测试 | 执行闭环强；不是辩论框架，部署面明显重于众神殿 |

说明：Captain Claw 用户引用的是 `0.4.17`，上游当前源码已到 `0.8.5`。核心议事模型没有变化，但新版本增加了自动组队、行动项、恢复、持久化等机制，因此下文以当前源码为主，并保留对 `0.4.17` 的核验。

## 2. MALLM：最值得吸收的是“可比较性”

### 2.1 它拆对了维度

MALLM 把 MAD 明确拆成四块：

1. **Agent persona**：谁在参与，例如无角色、领域专家、IPIP 大五人格。
2. **Response generator**：怎么表达，例如 Simple、Critical、Reasoning。
3. **Discussion paradigm**：信息如何流动、谁在什么时候看到什么。
4. **Decision protocol**：何时结束，以及如何把多份答案变成最终答案。

这比“给每个 Agent 一个性格提示词，然后让它们自由聊天”更接近可研究系统。论文中的 144 种配置，是这些轴组合后的实验空间，不应被理解成 144 种都值得产品化的会议模板。

### 2.2 源码中的讨论范式

当前源码实现了：

- **Memory**：所有 Agent 能看到所有历史，信息最透明。
- **Relay**：信息沿发言链传递，后位只看到接力内容。
- **Report**：各 Agent 独立处理，由中央 Agent 汇总。
- **Debate**：先由中立提案者提出草案，再让参与者在若干轮内质疑、修订，由决策协议收口。
- **Collective Refinement**：各自先答，之后同时看其他答案并改进自己的答案。

众神殿当前其实已经覆盖了两种最有用的近似：

- **独立开场**约等于“先 Report，再进入共享历史”。
- **圆桌接力**约等于“同轮可见的 Relay/Memory 混合”。

因此不值得把 MALLM 的五个类原样搬进来；值得吸收的是**把“开场可见性、后续记忆、轮次推进、最终读出”拆开表达**，而不是继续用一个 `protocol` 字符串暗示所有语义。

### 2.3 决策协议不是一回事

MALLM 的决策协议包括：

- 简单投票、批准投票、排序投票、累积投票；
- 多数、绝对多数、全体一致、混合阈值共识；
- 单裁判综合。

其论文结果清楚地说明“没有万能决策协议”：

- 在 MMLU / MMLU-Pro / GPQA 等知识题上，Consensus 平均约高 2.8 个百分点；
- 在 SQuAD 2.0 / StrategyQA / MuSR 等推理题上，Voting 平均约高 13.2 个百分点；
- StrategyQA 示例中，Critical 响应风格比 Simple 高 2.8 个百分点，而强结构的 Reasoning 风格反而低 6.4 个百分点。

这些数字不是众神殿效果的保证，因为任务、模型、数据与实现都不同；它们提供的是反例：**不能用“辩论越长越接近真相”作为默认产品假设，也不能用一个固定裁决协议覆盖所有议题**。

### 2.4 MALLM 的评测设计值得借，评测宣判不能照搬

值得借：

- 每个实验固定任务、模型、Agent 数、轮次、可见历史范围；
- 一次只变一个协议变量；
- 重复运行并报告方差；
- 同时记录消息、投票、配置、轮次、耗时和最终答案；
- 支持答案型任务和自由文本任务的不同指标。

必须加上的边界：

- 众神殿的 Agent 是不同 Harness（Codex / ZCode / DSH / Grok），模型和工具能力天然混杂；对照组必须固定 Harness 版本与权限，不应把“异构 Agent 的平均分”误写成“某协议更好”。
- 自建评测不能只让一个 LLM 裁判打分，否则裁判偏好会变成“正确性”。

**蒸馏结论：吸收四轴建模与离线评测；不吸收“把所有组合铺给用户”。**

## 3. Captain Claw / Agent Council：最值得吸收的是显式动作与可干预流程

### 3.1 实际机制

当前源码与用户给出的 `0.4.17` 快照都支持：

- 2 到 N 个 Agent 围绕议题分轮发言；
- 会话类型区分 Debate、Brainstorm、Review、Planning、Interview、Troubleshoot、Critique、Freeform；
- 每位 Agent 在陈词开头声明：
  - `SUITABILITY: 0..1`：自评本席对该议题的适配度；
  - `ACTION: answer | respond | challenge | refine | broaden | pass`；
  - `TARGET`：可选的被回应/被挑战 Agent；
- Round-robin 模式按 suitability 排序后全员发言；
- Moderator 模式由一个主持人根据适配度和讨论流选择下一位；
- 支持用户随时插入、静音 Agent、固定关键消息；
- 主持人或首位 Agent 综合结论；综合后所有 Agent 以 agree / disagree / abstain 加理由投票；
- 导出 Markdown 纪要，并保留每席 TL;DR、投票、行动项和产物。

这与众神殿已有的“用户插话、问某一席、每轮决策门、主席小结”高度相邻。

### 3.2 值得吸收的三点

1. **动作意图显式化**

   当前众神殿的贡献主要是自由文本。读者知道某席“又说了很多”，却不容易一眼看出它是在提出新论证、回应旧论证、承认修正，还是只是拓宽背景。

   可在“定向质询”模式中要求每席明确声明：

   ```text
   动作：主张 / 质疑 / 回应 / 修正 / 补强
   目标贡献：贡献 ID（质疑/回应时必填）
   本轮只挑战一个最弱论证
   ```

   这不是增加表演性，而是让“讨论有没有推进”可核查。

2. **主持人负责议程，不负责赢家**

   Captain Claw 的 moderator 会选下一位、综合、发起投票。众神殿当前分支已落地的主席轮间小结已经吸收了更重要的部分：提炼真分歧、裁决建议、下轮必答一问。

   与众神殿目标一致的做法是：

   - 主持仍由汇总席承担；
   - 轮间只生成**议程与分歧账本**，不给 Agent 排名；
   - 固定轮换顺序保留，避免模型用自评分数操控话语权；
   - 主席小结全文进入下一轮，使下一轮先回应未决问题。

3. **结论后的“确认”与“行动项”**

   Captain Claw 在综合后让所有 Agent 投票，并抽取各自行动项。这个结构值得拆成两个可选阶段：

   - **结论确认**：各席对匿名化结论投“支持 / 有条件支持 / 反对”，附一条关键理由；不评选赢家。
   - **行动清单**：从结论中提取负责人、依赖、验收证据和下一步，不在议事阶段自动执行。

### 3.3 不照搬的地方

- **自评 suitability 不宜作为默认发言权。** 这是模型自报，不是独立可信的专长测量；在异构 Agent 场景下容易把“提示词更自信”误当成“更合适”。
- **不把 agree/disagree 当正确答案。** 投票只表示席位态度，不表示事实为真。
- **不把 8 种会话类型全部塞进首屏。** 先提供少量可解释 preset；更多类型应围绕真实任务逐步增加。
- **不把行动项直接写入 Agent 的待办。** 众神殿当前是只读议事；执行必须另立授权和运行面。

**蒸馏结论：吸收动作类型、目标贡献、主持人议程、结论确认和纪要资产；拒绝自评分数决定话语权与自动执行。**

## 4. ARTEMIS：最值得吸收的是“辩论者 ≠ 评审者”

### 4.1 机制

ARTEMIS 的主流程是：

```text
开场陈述
  → 若干轮主辩论
  → 结案陈述
  → 陪审团独立评审
  → 汇总赢家、置信度、异议与评分明细
```

它有五个评审视角：分析、伦理、实践、对抗、综合。每位陪审员独立评分，再按视角权重和置信度聚合。分歧意见被保留在 verdict 中。

这对众神殿最有价值的启发不是“给 Agent 打分”，而是：

> **生成段与读出段必须职责分离。**

当前众神殿的主席既参与讨论，又负责最终综合。这样虽节省调用，但存在“自己总结自己论证”的偏差。可选的高保障模式应当增加独立读出，而不是让参与席互评后直接选赢家。

### 4.2 其评分实现不能直接当“客观裁判”

对当前源码的核验发现：

- “逻辑连贯”主要依赖连接词、编号、关键词计数；
- “证据质量”主要依赖数量、已标注 evidence 数和正则抽取；
- “因果推理”主要依赖正则和默认强度；
- “说服力”会用长度、强调词、反问号加分；
- 置信度是 jurors 的投票一致性与个体 confidence 的启发式合成，并非校准概率；
- 陪审员最终选的是“表现最好的 Agent”，不是逐条验证哪个命题为真。

因此它能提供结构启发，不能提供事实权威。

### 4.3 可信度边界

- 官方论文来源是 **Technical Disclosure Commons / Defensive Publications**，不是同行评审会议论文；README 将“production-ready”和“peer-reviewed research”写得比证据更强。
- `artemis-agents 2.0.0` 的 README 快速开始写 `from artemis import Debate, Agent, JuryPanel`，但当前 wheel 和源码的 `artemis/__init__.py` 并未导出这几个名字；需从 `artemis.core.*` 导入，公开 API 与文档不一致。
- 自测宣称 86% decision accuracy，但对应实验是 3 个议题 × 3 次 × 3 框架；ARTEMIS 实际 benchmark 配置只有 1 名陪审员；评分由 GPT-4o 执行，且评分规则给“明确 verdict”显著加分。这是**项目自测的宣传材料**，不是独立基准。

### 4.4 可蒸馏部分

- 开场、讨论、读出三个相位分离；
- 评审者独立工作，再聚合；
- 明确记录异议，而不是把异议折叠进主席结论；
- 对高影响决定提供可选的多视角检查；
- 任何分值都必须注明是“模型评分”，不能写成客观正确率。

**蒸馏结论：吸收职责分离、独立评审和异议保留；不吸收按 Agent 排名、关键词评分和未校准 confidence。**

## 5. AgentTeams：最值得吸收的是“裁决之后如何成为可干预的工作”

### 5.1 它解决的不是辩论问题

AgentTeams 是 Manager-Workers 协作运行时，主要对象是：

- **Manager**：组织 Worker、判断分工、跟踪进度、向人类汇报；
- **Team Leader**：在团队内拆解任务、协调成员；
- **Worker**：在独立运行时和容器中执行工作；
- **Human**：按权限进入房间，观察并随时改方向；
- **Project / Task / Result**：把工作拆成依赖关系、状态和交付物。

它采用 Matrix 房间作为统一协作层，并用共享文件系统交换任务说明、结果和大体积产物。Agent 之间的协作在房间中可见；人类可 @mention 指定的 Agent 插入新要求。

当前源码还实现了较低层的可靠性细节：

- 任务分配前先把 `spec`/元数据写入共享存储，再发 Matrix 通知，最后记录 `event_id` 和 `assigned`；
- 用稳定事务 ID 重试通知，避免重复投递；
- 任务状态有 `pending / delegated / completed / blocked / revision / cancelled`；
- 结果有 `SUCCESS / SUCCESS_WITH_NOTES / REVISION_NEEDED / BLOCKED / INTERRUPTED`；
- 心跳检查 Worker 进度，必要时唤醒 Worker、追问阻塞并报告管理员；
- 权限分级、房间拓扑和通信白名单约束谁能和谁说话；
- Worker 只持有网关 Consumer token，真实模型与 MCP 凭证留在 AI Gateway。

### 5.2 房间拓扑值得借

其团队拓扑是：

```text
Leader Room：Manager + Admin + Team Leader
Team Room：  Team Leader + Admin + Workers（Manager 不在）
Worker Room：Team Leader + Admin + Worker
```

Manager 不穿透团队，只和 Team Leader 通信；Team Leader 负责内部拆解。这降低了中心 Manager 的通信负担，也让权限边界和审计对象清楚。

但这条规则**不能照搬到辩论阶段**。辩论的公正性要求主席能看到所有席位原始论证，Team Room 隐藏 Manager 会导致汇总席缺证据。它只适合未来的执行阶段。

### 5.3 对众神殿最合适的两层映射

第一层，现在就能做，仍是只读：

- 每次议事从裁决中生成结构化 `ActionPlan`，字段包括行动、理由、依赖、验收证据、建议执行席；
- 行动只停留在“提议”，必须由用户签注；
- 不开放写盘、安装软件或外部副作用。

第二层，未来另立模式，不能藏在当前只读开关下：

- 用户签注后，显式进入“事工房”；
- Manager 负责拆任务，Worker 负责执行，人类可随时定向插话；
- 每个任务有独立工作区、状态、证据和验收；
- 所有 Agent 消息、用户指令和工具结果进入同一议事/工作时序；
- 任务完成不等于事实正确，仍需人类验收。

### 5.4 不照搬的地方

- 不把 Matrix / Kubernetes / MinIO / Higress 整套基础设施塞进 Wails 桌面应用；
- 不为一次讨论启动一队常驻容器；
- 不把“Manager 不进入 Team Room”的边界用于辩论；
- 不让 Worker 在议事阶段获得写权限；
- 不把“任务完成状态”当作“结论正确”。

**蒸馏结论：吸收房间级透明、人类定向干预、任务账本、幂等派工和执行验收；把它放在签注之后，并作为独立产品模式。**

## 6. 众神殿现状对照

当前实现已经具备不少基础能力，不应重复造：

| 能力 | 当前状态 | 结论 |
|---|---|---|
| 独立开场 / 圆桌接力 | 已有 | 保留为“信息可见性”维度 |
| 手动轮次 / 至共识 | 已有 | 保留为“结束条件”维度 |
| 单席失败、重试、Attempt 证据 | 已有 | 继续作为执行真相层 |
| 用户插话 | 已有，广播给后续席位 | 可考虑增加“定向给某席” |
| 每轮决策门、问题/回答 | 已有 | 是 AgentTeams “human in the loop”的本地等价物 |
| 主席轮间小结 | 当前分支已落地 | 先稳定和验证，不另造第二套机制 |
| 协作综合与人工签注 | 已有 | 保留；不把模型结论自动当正式结论 |
| 结构化动作/目标贡献 | 缺失 | 下一阶段最值得补的议事语义 |
| 辩论后的独立确认 | 缺失 | 作为高保障可选阶段，不默认全开 |
| 结构化行动项 | 缺失 | 可先做只读行动清单 |
| 协议对照评测 | 缺失 | 应先补离线实验，再扩 UI |
| 签注后执行闭环 | 不在当前范围 | 另立模式，避免扩大只读议事的权限面 |

需要同时处理的现有语义问题：

- 当前 `StartRequest.Protocol` 实际只区分 `independent` 与 `council`，语义是“首轮是否可见同轮前序”，并不是完整的议事协议。
- 工作树中新增的主持纪律要求“每轮至少点名挑战一个最弱论证”，而 `discussionPrompt` 末尾仍写“不要攻击弱点或持续反驳”。这里应统一为：**质疑主张和证据，不攻击席位与人格；必须明确挑战一条论证，但不进行胜负叙事。**
- `PhaseIndependent / PhaseReview / PhaseChallenge / PhaseVerdict` 目前主要是兼容标签，不应在未定义新状态机前直接复活成一套四阶段协议。

## 7. 建议的蒸馏方案

### 7.1 内部分成四个正交维度，用户只选 preset

```text
信息可见性
  independent | sequential

轮次推进
  fixed_round_robin | moderator_agenda

发言职责
  balanced | critical | verifier | red_team

读出方式
  synthesis | consensus | support_check
```

这不是要把界面做成 144 项配置，而是让实现和实验有稳定坐标。用户侧保留少量命名清楚的预设：

| Preset | 适用场景 | 核心流程 | 额外成本 |
|---|---|---|---|
| **圆桌会诊** | 默认、开放性判断 | 独立开场 → 共享讨论 → 主席综合 | 当前基线 |
| **定向质询** | 已有明显分歧、需要推进收口 | 独立开场 → 主席提炼分歧 → 定向质疑/回应 → 综合 | 每轮多一次主持调用 |
| **红队评审** | 方案、文档、架构、风险审查 | 独立初判 → 指定红队挑错 → 被挑战席回应 → 综合 | 可由现有席位兼任，不默认新增模型 |
| **至共识** | 需要明确收口条件 | 当前至共识流程 + 轮间议程 | 现有能力 |

“正反辩”适合二选一决策，但不应成为默认会议模板；多数 Gaea 议题不是法庭式二元命题。

### 7.2 定向质询的最小契约

只在 `targeted_challenge` preset 中增加：

```text
动作：主张 | 质疑 | 回应 | 修正 | 补强
目标贡献：<贡献 ID，质疑/回应时必填>
被挑战命题：<一句话，必须能在目标贡献中找到>
证据或验证路径：<必须给出>
```

规则：

- 每席每轮最多发起一个主要质疑；
- 质疑只能针对论证、证据、适用条件和验证方法；
- 被挑战席下一轮必须回应“接受、部分接受或拒绝”，并说明修正；
- 主持小结只维护“未决分歧账本”，不排名；
- 解析失败时降级为普通陈词并记 warning，不能让格式错误吞掉一次有效发言。

这比“让 Agent 自选何时发言”更可控，也比无边界的自由辩论更容易审计。

### 7.3 可选的事后确认

对高影响结论，可在主席综合后增加一轮轻量确认：

```text
支持
有条件支持：条件是什么
反对：反对的是哪条结论、证据、适用条件
```

要求：

- 输入优先使用匿名化结论与原贡献 ID，不展示 Agent 品牌；
- 不合成“团队分数”，不选“最佳 Agent”；
- 存在阻塞性反对时，交主席修订一次，再交用户签注；
- 反对意见永久保留；
- 默认关闭，只在“高保障/红队评审”或用户明确选择时启用。

这吸收了 Captain Claw 的投票和 ARTEMIS 的独立评审，但把“投票给谁”改成“对结论的哪一部分有异议”。

### 7.4 行动清单先只读，执行另立模式

建议在 `Verdict` 中增加可选的行动清单：

```json
{
  "action": "需要完成的具体动作",
  "rationale": "为什么现在做",
  "dependsOn": ["前置动作"],
  "acceptance": "怎样算完成，附证据要求",
  "suggestedOwner": "用户 / 某类能力席位",
  "risk": "失败或越界风险"
}
```

它只负责把讨论结果变成可执行的提案，不触发执行。

若未来进入执行：

- 必须另行签注，明确工作空间、写入范围、外部副作用和预算；
- 任务、负责人、状态、结果、证据与用户插话分别入账；
- 任务分配要先落任务说明，再通知执行者；
- 重试要幂等，不能重复派工或重复副作用；
- 人类可直接向某个 Worker 补充要求，但这条指令必须进入可见时序；
- Manager 只负责任务组织，不冒充裁决正确性。

## 8. 明确不抄的部分

1. **不暴露 144 种配置。** 可组合性留给测试与高级配置，产品入口保持在 3 至 4 个 preset。
2. **不让模型自评 suitability 决定默认话语权。** 自评分不能替代真实专长、工具证据或人工指定顺序。
3. **不把陪审团变成 Agent 排行榜。** 评审应针对主张、证据和适用条件，不针对席位品牌。
4. **不使用未校准数值冒充可信度。** 任何评分必须标明来源和用途。
5. **不把 winner-of-agents 当 truth-of-claims。** 赢了一场辩论不等于命题为真。
6. **不在议事阶段自动写盘或派工。** 执行需要独立的权限、预算、工作区和验收。
7. **不为桌面单机引入 Matrix/Kubernetes 全套基础设施。** 可借房间语义，不借部署负担。
8. **不把多轮讨论默认设为“更强”。** 必须先证明相对单席、独立多答和投票基线的增益。

## 9. 建议刀序

### D0：调研定稿（本文件）

不改运行时代码；后续设计以本文件为协议词表与取舍依据。

### D1：稳定当前主持小结与 prompt 语义

- 复核当前已落地的主持小结、重试保留、用户插话改动；
- 统一“挑战论证”与“不攻击席位”的措辞；
- 确保主席小结不会被当成额外参与者或最终裁决；
- 记录它的调用成本、失败降级和 UI 呈现。

### D2：加入“定向质询”preset

- 先只做后端 preset 与 prompt，不重构全部协议；
- 增加动作/目标贡献的结构化收声；
- 复用现有主席小结作为议程；
- 在 UI 显示“谁在质疑哪条论证”，但不显示胜率。

### D3：高保障确认

- 增加可选的事后支持/条件/反对检查；
- 匿名化结论，保留原始贡献引用；
- 输出阻塞异议与修订记录；
- 默认只在红队评审或用户显式开启时运行。

### D4：离线协议评测

先不增加更多会议模式，先回答：

| 对照 | 目的 |
|---|---|
| 单席直答 | 最低成本基线 |
| 独立多答 + 固定投票 | 检验“多数”本身的价值 |
| 当前圆桌会诊 | 检验共享上下文的价值 |
| 定向质询 | 检验“挑战已知分歧”的额外价值 |
| 至共识 | 检验提前停止与成本取舍 |

建议覆盖 20 至 30 个真实议题，并刻意加入：

- 只有一个席位持有正确反例；
- 多数席位被同一错误前提带偏；
- 分歧来自证据而非价值取向；
- 结论正确但关键证据缺失；
- 议题会自然漂移到实现细节。

每场记录：协议 preset、Harness/模型版本、工具权限、轮数、耗时、调用成本、截断次数、最终结论、证据引用、保留的少数意见、用户是否接受。

### D5：可选执行衔接

只有 D4 证明讨论值得进入工作时再做，且作为独立“事工”模式：

- 从签注结论生成任务 DAG；
- 先由用户确认行动清单；
- 再分配 Worker、工作区、验收标准与预算；
- 所有派工和人工插话进入统一审计记录；
- 不把 AgentTeams 作为当前桌面应用的内嵌依赖。

## 10. 验收标准

本调研转化的第一版机制至少应满足：

1. 用户能看懂本轮是在“主张、质疑、回应、修正”中的哪一种；
2. 每次质疑都能定位到具体贡献 ID，而不是模糊地说“有人认为”；
3. 主席小结只收敛议程，不宣称事实已经裁定；
4. 任何“共识”都区分：
   - 支持同一结论；
   - 接受适用条件；
   - 仍有非阻塞争议；
   - 存在阻塞异议；
5. 少数意见和未验证项不会被综合席吞掉；
6. 高保障模式能指出结论的反对意见，但不伪造数值裁判；
7. 行动清单可导出、可签注、可拒绝，但不会在只读会话中自动执行；
8. 协议对照能看到成本、耗时、证据覆盖和人工接受率，而不只是“讨论看起来很热闹”；
9. 议题若没有从中获益，产品文案应降级为“多 Agent 会诊”，不宣称辩论提高正确率；
10. 没有新的隐藏委派、隐藏写操作或第二套状态真相。

## 11. 源码与资料索引

### MALLM

- 论文：[MALLM: Multi-Agent Large Language Models Framework](https://aclanthology.org/2025.emnlp-demos.29/)
- 仓库：[Multi-Agent-LLMs/mallm](https://github.com/Multi-Agent-LLMs/mallm/tree/eeec9f5c28961a7349cbb4306a5aa860bdaef6dd)
- 讨论范式：[`mallm/discussion_paradigms/`](https://github.com/Multi-Agent-LLMs/mallm/tree/eeec9f5c28961a7349cbb4306a5aa860bdaef6dd/mallm/discussion_paradigms)
- 决策协议：[`mallm/decision_protocols/`](https://github.com/Multi-Agent-LLMs/mallm/tree/eeec9f5c28961a7349cbb4306a5aa860bdaef6dd/mallm/decision_protocols)
- 响应生成：[`mallm/models/discussion/`](https://github.com/Multi-Agent-LLMs/mallm/tree/eeec9f5c28961a7349cbb4306a5aa860bdaef6dd/mallm/models/discussion)

### Captain Claw / Agent Council

- 用户引用版本：[captain-claw 0.4.17](https://pypi.org/project/captain-claw/0.4.17/)
- 当前仓库：[kstevica/captain-claw](https://github.com/kstevica/captain-claw/tree/f5f2805c47d39b2cb4ff70f445f61f12cd549948)
- 议事状态机：[`flight-deck/src/stores/councilStore.ts`](https://github.com/kstevica/captain-claw/blob/f5f2805c47d39b2cb4ff70f445f61f12cd549948/flight-deck/src/stores/councilStore.ts)
- 议事提示词：[`flight-deck/src/instructions/council/`](https://github.com/kstevica/captain-claw/tree/f5f2805c47d39b2cb4ff70f445f61f12cd549948/flight-deck/src/instructions/council)
- 持久化模型：[`captain_claw/flight_deck/db.py`](https://github.com/kstevica/captain-claw/blob/f5f2805c47d39b2cb4ff70f445f61f12cd549948/captain_claw/flight_deck/db.py)

### ARTEMIS

- 仓库：[bassrehab/artemis-agents](https://github.com/bassrehab/artemis-agents/tree/599e66ff9ca12475f86a8bb41672a5b6279b570f)
- 技术披露：[Adaptive Reasoning and Evaluation Framework for Multi-agent Intelligent Systems in Debate-driven Decision-making](https://www.tdcommons.org/dpubs_series/7729/)
- 辩论流程：[`artemis/core/debate.py`](https://github.com/bassrehab/artemis-agents/blob/599e66ff9ca12475f86a8bb41672a5b6279b570f/artemis/core/debate.py)
- 陪审团：[`artemis/core/jury.py`](https://github.com/bassrehab/artemis-agents/blob/599e66ff9ca12475f86a8bb41672a5b6279b570f/artemis/core/jury.py)
- 启发式评估：[`artemis/core/evaluation.py`](https://github.com/bassrehab/artemis-agents/blob/599e66ff9ca12475f86a8bb41672a5b6279b570f/artemis/core/evaluation.py)
- 自测口径：[`benchmarks/ANALYSIS.md`](https://github.com/bassrehab/artemis-agents/blob/599e66ff9ca12475f86a8bb41672a5b6279b570f/benchmarks/ANALYSIS.md)、[`benchmarks/results/EXPERIMENT_SETUP.md`](https://github.com/bassrehab/artemis-agents/blob/599e66ff9ca12475f86a8bb41672a5b6279b570f/benchmarks/results/EXPERIMENT_SETUP.md)

### AgentTeams

- 仓库：[agentscope-ai/AgentTeams](https://github.com/agentscope-ai/AgentTeams/tree/d9366ac43a153d31c73ef4def125e378869b8d85)
- 架构：[`docs/design/k8s-native-orchestration.md`](https://github.com/agentscope-ai/AgentTeams/blob/d9366ac43a153d31c73ef4def125e378869b8d85/docs/design/k8s-native-orchestration.md)
- 快速开始与人工干预：[`docs/quickstart.md`](https://github.com/agentscope-ai/AgentTeams/blob/d9366ac43a153d31c73ef4def125e378869b8d85/docs/quickstart.md)
- 任务状态机：[`manager/src/agentteams_manager/agentteams_manager/task.py`](https://github.com/agentscope-ai/AgentTeams/blob/d9366ac43a153d31c73ef4def125e378869b8d85/manager/src/agentteams_manager/agentteams_manager/task.py)
- Matrix 任务工具：[`manager/src/agentteams_manager/agentteams_manager/tools/taskflow.py`](https://github.com/agentscope-ai/AgentTeams/blob/d9366ac43a153d31c73ef4def125e378869b8d85/manager/src/agentteams_manager/agentteams_manager/tools/taskflow.py)
- 人工插话测试：[`tests/test-04-human-intervene.sh`](https://github.com/agentscope-ai/AgentTeams/blob/d9366ac43a153d31c73ef4def125e378869b8d85/tests/test-04-human-intervene.sh)

### 众神殿当前实现

- `internal/pantheon/manager.go`
- `internal/pantheon/prompts.go`
- `internal/pantheon/types.go`
- `frontend/src/pages/pantheon/PantheonComposer.tsx`
- `frontend/src/pages/pantheon/PantheonRecord.tsx`
- `docs/gaea-pantheon-market-research-2026-10-29.md`
- `docs/gaea-orca-pantheon-distill-2026-11.md`
- `docs/gaea-orca-pantheon-distill-round2-2026-11.md`

## 12. 最终判断

**四个上游都不接为众神殿内核。**

第一优先不是“再加几个辩论协议”，而是把议事语义补到可比较、可审计：

1. 保留独立开场和固定轮换；
2. 先落“定向质询”；
3. 把主持小结变成分歧议程；
4. 高保障议题再开独立确认；
5. 用离线评测决定哪些模式值得保留；
6. 执行作为签注后的独立模式，另算权限与成本。

这与众神殿既有方向一致：**让判断被校正，而不是让 Agent 赢；让结论可执行，而不是让讨论偷偷变成施工。**
