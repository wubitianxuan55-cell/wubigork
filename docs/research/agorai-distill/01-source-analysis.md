# Agorai 源码蒸馏分析（只读调研，v0.8.0 / HEAD 1e469be）

调研对象：`C:\AI\wubigrok\clones\agorai`（GitHub StevenJohnson998/Agorai），`package.json` 版本 0.8.0，HEAD=1e469be（2026-04-12，commit message 为 CHANGELOG 条目）。
调研方式：只读源码与文档，未执行 `npm install` / 测试 / 构建，未修改仓库。所有结论以源码为主，文档仅作对照；凡只有文档依据的，标注「文档推断，未见实现」。

---

## 一句话判定

Agorai 是一个「多 AI 席位 + 人类同场」的**会话协作桥**：核心价值不在「多模型辩论」（那部分实现幼稚），而在**它把「编排」做成了不依赖 LLM 的、基于存储事件的进程管理器**——Keryx 只发 `type:"status"` 的状态消息、从不生产内容，席位靠轮询/SSE 拉取并用文本协议自组织成轮次；真正值得蒸馏的是它围绕这个骨架长出来的一整套**治理基础设施**：4 级可见性 + clearance 上限 + 高水位标记、定向消息（whisper）与 @mention 一致性校验、原子任务认领 + 5 分钟失联回收、三级渐进披露技能、DB 托管 pass-key + 工具白名单、按预算分档降级、以及「本轮盲写、次轮全量」的上下文截断规则。不值得蒸馏的是它的**共识判定**（10 条英文短语白名单 + 与「已答席位」而非「应到席位」比较）、**状态机的生命周期管理**（轮次计数只增不清、达上限后永久静默）、**半成品交互**（`summary`/`extend` 有入口无实现）、以及**内存态编排 + 文本正则耦合**这一整套「看起来在工作」的实现方式。

> 仓库自洽性警告（影响可信度判断）：本 checkout 里 `src/bridge/server.ts:1503,1543,1567` 调用了 `getPublicMessages/getActiveConversations/getConversationStats`，但 `src/store/sqlite.ts` 与 `src/store/interfaces.ts` 中**不存在**这三个方法；`Conversation.publicRead`、`messages.level` 列也不存在（`src/store/types.ts:109-119`、`src/store/sqlite.ts:156-169`）。`CHANGELOG.md:51` 声称已实现并注册 `src/keryx/modes/wild-agora.ts`，但全仓只有类型声明与文案（`src/keryx/mode-interface.ts:15`、`src/keryx/module.ts:412`），**文件不存在、未注册**（`src/keryx/module.ts:77-78` 只注册 Socratic 与 Ecclesia）。`CHANGELOG.md:62-92` 自己写着这些特性「未在线上实例手工验证」。因此：Keryx / store / bridge 主体代码可当规格读，public-read / level / wild-agora 一律按「文档推断，未见实现」处理。

---

## 1. Keryx 编排器（Ecclesia / Socratic）

### 1.1 结论先行

- Keryx 是**事件驱动的进程管理器**，不是调度器：它不做「调用某一个席位」，只往会话里写一条状态消息，然后**等**席位的普通消息回来（`src/keryx/module.ts:148-181`）。
- 一轮的推进完全由「消息事件 + 定时器」驱动，状态机 6 态：`idle | open | collecting | synthesizing | closed | interrupted`（`src/keryx/types.ts:10-16`）。
- **「谁决定下一个发言者」在 Ecclesia 里不存在**——所有 `expectedAgents` 同时被期待；Socratic 里是固定队列（按名字字母序）。
- **共识不是算法，是 10 条英文短语的白名单匹配**（`src/keryx/patterns.ts:13-36`），阈值是同义反复的「所有已答者都说了短语」。
- **综合（synthesis）委派给「具备 `synthesis` 能力的订阅者」，找不到就随机挑一个本轮已答者**（`src/keryx/modes/ecclesia.ts:248-298`）；喂给它的不是结构化数据，只是一条 `@名字 请综合 rounds 1-N` 的状态消息。
- 超时「自适应」是真公式（4 因子加权 + 与历史均值的 0.6/0.4 混合），但它混合的是**毫秒级绝对值**，一旦历史值被污染就会直接把定时器打到溢出（见 §9.1）。
- 席位沉默 → 升级链 nudge → 强制关轮；席位报错 → 立即从本轮 `expectedAgents` 删除；席位超时 → 关轮时不计入。**三者都不是重试**，只有「从轮次里摘掉」。

### 1.2 一轮的完整推进（Ecclesia）

| 步骤 | 触发 | 代码 | 关键字段/动作 |
|---|---|---|---|
| 0 | 人类发消息 | `src/keryx/module.ts:148-181` | 非自己的消息 → 入 50 条滚动窗（`module.ts:39,647-658`）→ 命中 `@keryx` 命令则拦截（`module.ts:171-174`）→ 否则派发给 mode |
| 1 | `currentRound==null/closed/idle` 且发送者是 human（`agent.type==="human"`） | `src/keryx/modes/ecclesia.ts:32-39` | 只有人类消息能开轮；agent 的消息不会开轮 |
| 2 | 开轮 | `ecclesia.ts:102-155` | `roundNumber = roundHistory.length + 1`（`:105`）；`> maxRoundsPerTopic(3)` 直接 return（`:107-110`，**不发任何提示**）；`expectedAgents = 除触发者外的全部非 keryx/非 orchestrator/非 human 订阅者`（`:112-113` + `module.ts:614-631`）；`topic = 触发消息前 100 字`（`:121-122`）；`timeout = calculateTimeout`（`:124`） |
| 3 | 广播开轮 | `ecclesia.ts:141-149` | 仅一条 `type:"status"`、`tags:["keryx"]` 的模板文本（`module.ts:633-645`、`templates.ts:15-23`） |
| 4 | 收言 | `ecclesia.ts:157-175` | 同轮同席位第二条被丢弃（`:161-164`）；`responseContents: Map<agentId, content>`、`responseMessageIds[]`、`updateAgentProfile`（`:166-170`） |
| 5 | 全员已答判定 | `ecclesia.ts:177-188` | `[...expectedAgents].every(id => respondedAgents.has(id))`（`:181-183`）——**不是「多数」**，是「应到全到」 |
| 6 | 关轮 | `ecclesia.ts:190-246` | 统计 `consensusCount`（`:203-206`）→ 发关轮模板 → 判三选一：全共识 / 到轮上限 / 继续 |
| 7a | 全共识 | `ecclesia.ts:218-226` | `consensusCount === respondedAgents.size && respondedAgents.size > 0`（`:218`）→ `status="synthesizing"` → 委派综合 |
| 7b | 到上限 | `ecclesia.ts:227-237` | `round.id >= maxRoundsPerTopic` → 同上 |
| 7c | 继续 | `ecclesia.ts:238-245` | 压入 `roundHistory`、清空 `currentRound`、跑模式检测、`autoOpenNextRound` |
| 8 | 自动开下一轮 | `ecclesia.ts:300-348` | 主题**永远是 `roundHistory[0].topic`**（`:306`），触发消息 ID 也复用（`:324`）——**话题永久绑定第一轮** |
| 9 | 综合回传 | `ecclesia.ts:350-368` | **任何人在 `synthesizing` 期发的第一条消息都被当成综合**（`:352-354`，无发送者校验）→ 关轮入历史 |

### 1.3 共识检测：算法与阈值（这是蒸馏时要替换的部分）

```ts
// src/keryx/patterns.ts:30-36
isConsensusResponse(content):
  trimmed === "[NO_RESPONSE]"  → true
  lower.includes(phrase) for phrase in 10 条英文短语 → true
```
10 条短语（`patterns.ts:13-24`）：`nothing to add` / `no further points` / `fully agree` / `i agree with everything` / `consensus reached` / `all points covered` / `no new insights` / `nothing new to add` / `i have nothing to add` / `no additional input`。

判定式（`ecclesia.ts:203-220`）：

```
consensusCount = |{ r ∈ responseContents.values() : isConsensusResponse(r) }|
allConsensus   = consensusCount == respondedAgents.size && respondedAgents.size > 0
```

三个必须记录的缺陷：
1. **分母错**：比的是「已答席位」而不是「应到席位」。5 席里 1 席答「nothing to add」而其余 4 席未答时，`consensusCount=1 == responded=1` → **直接终局**（`ecclesia.ts:218`）。
2. **子串匹配**：`lower.includes("nothing to add")` 会把「The upgrade notes say there's nothing to add to the schema」判成弃权（`patterns.ts:35`）。
3. **语义混淆**：关轮模板把 `consensusCount` 当 `noResponseCount` 展示（`ecclesia.ts:208-215` + `templates.ts:25-38`），「共识」与「弃权」在 UI 上不可区分。

真正的「分歧」信息在 Keryx 里**完全不存在**：没有立场聚类、没有分歧清单、没有置信度。对比另一条死掉的路径：`src/consensus/vote.ts:17-68` 用 `weight = selfConfidence × personaBonus`、`dissent = weight ≥ winner×0.5`；`src/consensus/debate.ts:28-58` 用 30% 阈值，两者都**只被 `src/orchestrator.ts` 的 CLI 辩论路径使用**，与 Keryx 无关（Keryx 不 import consensus/*）。

### 1.4 综合（synthesis）委派给谁、喂什么

- 选人：`store.findAgentsByCapability(config.synthesisCapability)`（默认 `"synthesis"`，`src/config.ts:201`）→ 与「本会话订阅者」取交集且排除 Keryx 自己（`ecclesia.ts:251-260`）→ 找不到就 `random(responders)`（`:262-268`）→ 再找不到就关轮（`:270-276`）。
- 喂什么：**只有一条模板文本** `🔄 @name — Please synthesize the full discussion (rounds 1–N). Topic: …  Summarize key points, areas of agreement/disagreement, and any open questions.`（`templates.ts:46-55`、`ecclesia.ts:278-285`）。
- 它的实际输入 = 它自己去读会话时拿到的上下文 = **最近 20 条消息**（`src/agent/internal-agent.ts:329-338`、`src/agent/context.ts:294`），不是「全量讨论」。`rounds 1–N` 这句话在 >20 条消息后就是**假的**。
- 超时：`baseTimeoutMs × 2`（`ecclesia.ts:287`）——**不参与自适应**，且超时后只是 `status="closed"` 入历史，不发任何「综合失败」提示（`:288-295`）。

### 1.5 Socratic 模式与 Ecclesia 的差别

| 维度 | Ecclesia | Socratic |
|---|---|---|
| 并发 | 全员同时被期待 | 严格单席，`awaitingResponse` 门闩（`socratic.ts:131,175`） |
| 发言顺序 | 无 | `participants.sort(by name)` **字母序**（`socratic.ts:121`）；但技能文案写「subscription order」（`socratic.ts:91`）——文档与代码不一致 |
| 轮次定义 | round = 全员一轮 | cycle = 队列走一圈（`socratic.ts:247-254`） |
| 结束条件 | 全共识或 maxRounds | 所有未 pass 的席位都 pass，或 max cycles（`socratic.ts:160-218`）——**没有 max cycles 计数器**，靠 `passedAgents` 收敛 |
| 弃权语义 | `[NO_RESPONSE]` 计入共识 | `[NO_RESPONSE]` 加入 `passedAgents`，后续 cycle 跳过（`socratic.ts:236-240`），但仍留在队列 |
| 超时处理 | nudge → 强制关轮 | 跳过该席、进下一席，**该席下一 cycle 仍会被叫**（`socratic.ts:188-205`，注释 `:16-17`） |
| 人类插话 | 只作为开轮触发 | **中断并重开讨论**（`socratic.ts:54-59`）——话题被替换，旧进度全丢 |
| 综合 | 能力匹配 + 随机兜底 | 仅在 `activeIds` 里随机挑一个（`socratic.ts:274-283`） |

Socratic 的一个数值 bug：`ctx.updateAgentProfile(message.fromAgent, Date.now())`（`socratic.ts:233`）把**绝对时间戳**当「响应耗时」写入共享 profile（`module.ts:660-673`），见 §9.1。

### 1.6 两种模式的自适应超时（公式全文）

`src/keryx/timing.ts:20-46` 复杂度 `c ∈ [0,1]`：

```
wordFactor  = min(words/500, 1)
codeFactor  = min(```对数/3, 1)
questFactor = min('?'数量/5, 1)
techFactor  = min((URL | 3位以上数字 | {}[]<>|&) 命中数/10, 1)
c = 0.4*wordFactor + 0.25*codeFactor + 0.2*questFactor + 0.15*techFactor
```

`timing.ts:57-101` 超时（`base = 45000ms`，`src/config.ts:198`）：

```
t = base
t *= (0.5 + 1.5*c)                      // 0.5x ~ 2.0x
若本轮 expectedAgents 有历史 profile：
    avg = mean(avgResponseTimeMs of those)
    t = t*0.6 + avg*1.5*0.4             // 60/40 混合，历史项放大 1.5 倍
t *= (roundNumber == 1 ? 1.5 : 0.8)
若 subscriberCount > 3: t *= 1 + (subscriberCount-3)*0.1   // 每多一席 +10%
return round(t)
```

要点：**无上下限**（注释 `timing.ts:6` 明说 no floor/ceiling），**没有 token 预算参与**，`roundNumber` 取 `roundHistory.length+1`（`timing.ts:88`）因此「轮次越晚越快」——但结合 §9.2，轮次概念本身会被配额卡死。
`nudgeAfterMs`（默认 60000，`src/config.ts:199`）**声明了但从未被读取**（全仓只有 config/types/test 命中）——死配置。

### 1.7 席位沉默 / 报错 / 超时，编排器分别怎么处理

| 情形 | 检测方式 | 处理 | 是否重试 |
|---|---|---|---|
| 沉默（无消息） | 定时器 `startEscalationChain`（`ecclesia.ts:374-420`） | 超时 → 若有「多数」→ 10s 宽限后关轮（`:385-402`）；否则发 nudge（`:406-408`）→ 再等 `base*0.5` → 强制关轮（`:410-418`） | 否（只催一次） |
| 报错 | 席位自己发 `type:"status", tags:["agent-error"]`（`src/agent/internal-agent.ts:438-449`） | `module.ts:159-162` 拦截 → `ecclesia.handleAgentError` 从 `expectedAgents` 删除并检查完成（`ecclesia.ts:449-458`）；Socratic 直接跳下一席（`module.ts:439-466`） | **席位侧会无限重试**（见 §9.3） |
| 超时（Socratic） | 每席独立 timer（`socratic.ts:187-205`） | 跳过、进下一席，下 cycle 仍叫 | 否 |
| 全部失败 | — | `allFailed` 仅在 CLI 辩论路径存在（`src/orchestrator.ts:400-407`）；Keryx 无此判定，靠 `expectedAgents` 被删空 → `checkRoundCompletion` 立即关轮 | — |

`escalationLevel` 字段存在（`types.ts:36-37`），但**只被赋 0/1/2**（`ecclesia.ts:406,415`），`ARCHITECTURE.md:143-152` 宣称的 4 级链（1.0x 静默 / 1.5x nudge / 2.5x CC 备份 / 4.0x 上报人类）**在代码里不存在**：没有 `escalateToHuman` 调用、没有 backup agent、`templates.escalateToHuman`（`templates.ts:68-81`）从未被调用。→ 该 4 级表属于「文档推断，未见实现」。

---

## 2. 轮次与并行

**结论**：Agorai 的「并行」不是并发调度，而是**一次广播 + 各席位自行轮询**；没有任何并发上限、排队、取消或背压。唯一「防拖死」机制是「多数已答 → 10 秒宽限 → 关轮」。

1. **并行粒度 = 整轮**：开轮只写 1 条状态消息（`ecclesia.ts:141-149`），随后 `startEscalationChain` 起定时器（`:152`）。真正的调用发生在每个席位自己的进程里：内部席位 3s 轮询（`internal-agent.ts:47`）+ 事件总线唤醒（`:88-104`）；外部席位靠 MCP SSE 通知或自行轮询。因此**「并行」= 尽可能同时开始，结束时间不可控**，且 Keryx 不知道谁在跑。
2. **每轮每席最多一条**：`respondedAgents` 去重（`ecclesia.ts:161-164`）；席位侧还有第二道去重——「触发消息之后我是否已发过非 status 消息」（`internal-agent.ts:308-318`）。
3. **「多数已答即关闭」存在，但是二级机制**：`hasMajority = responded > 0 && responded > expected/2`（`ecclesia.ts:385-388`），命中后**不是立刻关**，而是再给 10 秒宽限（`:396-402`，硬编码 `10_000`）。「全员已答」走 `checkRoundCompletion` 立即关（`:177-188`）。
4. **防拖死的三道闸**：① nudge（`:406-408`）；② `base*0.5` 后强制关轮（`:410-418`）；③ 席位报错即摘除（`:449-458`）。**没有 per-seat 硬截止**：席位侧自己的 adapter 超时是 `30s + 20ms/token`（CLI，上限 5min）或 `60s + 15ms/token`（HTTP，上限 10min）（`src/adapters/base.ts:57-65`），远大于轮超时，所以**迟到回复会落进下一轮**（`recordResponse` 只检查「当前轮 expectedAgents 是否包含它」，`ecclesia.ts:42,157-175`）——跨轮归属污染。
5. **不取消**：轮次关闭时没有向任何席位发取消信号；`AbortSignal` 只用于进程关停（`module.ts:103,111`）。已发出的 LLM 调用继续烧钱。
6. **轮次配额是全局的**：`roundNumber` 由历史长度推导（`ecclesia.ts:105`），`roundHistory` **全程只增不清**（全仓无清空点，见 grep：仅 `module.ts:584` 初始化与各处 push）→ 同一会话累计 3 轮后，人类再发消息 `openRound` 直接 return（`:107-110`），**无提示、无 UI 反馈**，会话永久静默。重启进程才会清空（内存态）。

---

## 3. 上下文经济

**结论**：每个席位每轮看到的是**固定配额的「最近 20 条消息 + 10 条项目记忆 + 技能指令 + 3 层私有记忆」**，没有 token 计数、没有摘要压缩、没有检索；唯一的「裁剪」是首轮的时间戳截断（同轮盲写）。最长上下文风险是 20 × 100KB。

| 组成 | 配额/规则 | 证据 |
|---|---|---|
| 最近消息 | `messageLimit = 20`（默认），`ORDER BY created_at ASC` 后取前 20（**注意：取的是最早 20 条**，因为 SQL 是 ASC + JS `slice(0, limit)`） | `src/agent/context.ts:294`、`src/store/sqlite.ts:1236,1256-1258` |
| 首轮盲写 | 若本轮是 Round 1（模板含 `Topic:` 而非 `Continuing:`），把上下文截断在**开轮消息的 `createdAt`**：`messages.filter(m => m.createdAt <= cutoff)` | `src/agent/internal-agent.ts:320-325`、`src/agent/context.ts:87-91,407-411` |
| 次轮起 | 不截断 → 能看到上一轮所有人的发言（也无「只看主席简报」） | `internal-agent.ts:324-325` |
| 项目记忆 | `getMemory(limit: 10)`，每条 `content` 超 200 字截断 | `context.ts:354,520-529` |
| 技能 | 只注入 `instructions` 字段（tier-1），不注入 `content` | `context.ts:509-517` |
| 私有记忆 | global / project / conversation 三作用域全文拼接 | `context.ts:531-540` |
| 会话元信息 | 项目名、保密模式、默认可见性、订阅者名单（含 online，阈值 5 分钟） | `context.ts:496-507,385-399` |
| 桥规则 | 5 条固定规则（mark_read 纪律、可见性取输入最大值、双 metadata 模型、标准工作流、@mention 规则）+ 条件规则 | `context.ts:115-154` |
| Keryx 规则 | 仅当会话被判定为 Keryx 管理时注入「每轮一次、[NO_RESPONSE]、被要求综合时怎么写」 | `context.ts:217-225` |
| token 预算 | **无**。Keryx 路径不统计 token；`bridge` 侧只有 `messages.maxContentLength=10000`（`config.ts:209`）与 Zod 100KB 上限（`src/bridge/tools.ts:29`） | `internal-agent.ts:329-341` 无任何计数 |
| 摘要压缩 | Keryx 路径**没有**。仅 CLI 辩论路径有「预算超 80% 时把上轮回复截到 200 字」 | `src/orchestrator.ts:179-190,308` |

可见性规则（席位之间能否互看）：
- **按 clearance 过滤**：`getMessages` 用 `visibilityToInt(row.visibility) <= maxVis`（`sqlite.ts:1241-1242`）。
- **whisper 定向**：带 `recipients` 的消息只对发送者与收件人可见（`sqlite.ts:1244-1250`；SSE 侧同样门控 `bridge/server.ts:1905`）。
- **同轮盲写**：仅 Round 1 生效（见上）。这不是「互相不可见」的强隔离，而是**首轮独立陈词**的近似实现——Round 2 起全量互看。
- **人类消息**：与普通消息同构（`type:"message"`，`fromAgent=human 的 agent id`），只是 `isHumanMessage` 用 `agent.type === "human"` 判定（`module.ts:599-611`）来决定「是否开轮」。人类**不是** `expectedAgents` 成员（`module.ts:625` 排除 human），所以人类不占席位、不参与「全员已答」。

---

## 4. 共享记忆与 4 级可见性

### 4.1 数据模型（三套并存，别混）

| 层 | 表 / 类型 | 字段 | 作用域 | 证据 |
|---|---|---|---|---|
| 项目共享记忆 | `project_memory` / `MemoryEntry` | `id, project_id, type('note'/'context'/'decision'/'skill'/'digest'), title, tags[], priority('high'/'normal'/'low'), visibility, content, created_by, created_at, updated_at` | 每项目 | `sqlite.ts:117-131`、`src/store/types.ts:95-107`、`tools.ts:75-83` |
| Agent 私有记忆 | `agent_memory` / `AgentMemory` | `(agent_id, scope, scope_id) PK, content, updated_at`；scope ∈ `global|project|conversation` | 每 agent | `sqlite.ts:226-234`、`types.ts:344-352` |
| 技能 | `skills` + `skill_files` | 见 §5 | bridge/project/conversation | `sqlite.ts:236-263` |
| （死层）Blackboard | `src/memory/base.ts` + `src/memory/sqlite.ts` | 项目/辩论/键值 + private/public | — | 实现全是 `throw "not implemented (v0.2)"`（`memory/sqlite.ts:35-97`），只有 `scanForSensitiveData` 是真的（`:99-109`） |

`priority` 只被存储，**没有任何读取路径用它排序或加权**（`getMemory` 的 ORDER BY 只有 `created_at DESC`，`sqlite.ts:920`）。`type` 只是字符串过滤（`:915-918`），没有枚举约束（`tools.ts:77` 只是 describe 里列了几个值）。

### 4.2 4 级可见性在存储层怎么强制

顺序映射：`public=0 < team=1 < confidential=2 < restricted=3`（`src/store/types.ts:17-22`）。

**读**（4 处，全部在 JS 层过滤，不在 SQL 层）：
1. `getMessages`：`SELECT * FROM messages WHERE conversation_id=?`（**无 SQL LIMIT**）→ JS `.filter(vis <= clearance)` → whisper 过滤 → tag 过滤 → 再 `slice(0,limit)`（`sqlite.ts:1214-1258`）。
2. `getMemory`：同样先全量取回再过滤（`sqlite.ts:908-937`，注释 `:922` 明说「No SQL LIMIT」）。
3. `getUnreadCount`：JOIN 后 JS 过滤（`sqlite.ts:1337-1350`）。
4. SSE 推送：按 `agentVisInt < messageVisInt` 跳过（`bridge/server.ts:1891-1902`）+ whisper 跳过（`:1905`）。

**写**：
- 消息**有**上限裁剪：`requestedVis > senderClearance` 时降到 `senderClearance`，并写入 `bridgeMetadata.visibilityCapped / originalVisibility`（`sqlite.ts:1139-1156`）。
- 记忆**没有**上限裁剪：`setMemory` 直接用 `entry.visibility ?? "team"`（`sqlite.ts:873-891`）——低 clearance 席位可写入自己以后读不到的 `restricted` 记忆（写权限与读权限不对称）。
- 防伪造：`send_message` 的 `metadata` 里凡以 `_bridge` 开头或名为 `bridgeMetadata/bridge_metadata` 的键被剥离（`sqlite.ts:1158-1168`）；`bridgeMetadata` 由 bridge 独占生成。

**高水位标记**（被动审计）：每次 `getMessages` 成功后，把本次返回消息的最大可见性以「只增不减」的 UPSERT 记到 `agent_high_water_marks(agent_id, project_id, max_visibility)`（`sqlite.ts:1260-1302`，SQL CASE 保证单调）。仓库里**没有任何消费方**（只有 `getHighWaterMark` 读接口，`sqlite.ts:1305-1322`）——是给未来「你读过机密，输出也必须机密」用的，但当前只是记账。

### 4.3 「agent 不知道被隐藏数据存在」的实现与漏洞

实现方式非常朴素：**过滤掉，不返回任何计数**。`getUnreadCount` 也只数可见的（`sqlite.ts:1349`）；SSE 不发、JSON 里没有 `hidden_count` 之类字段。

但有 5 条侧信道（蒸馏时必须补上）：
1. **会话活跃时间恒被刷新**：任何消息（含 restricted）都会 `UPDATE conversations SET updated_at`（`sqlite.ts:1191`）→ 低 clearance 席位能看到「会话在动但没新消息」。
2. **发送者 clearance 泄露**：每条消息的 `bridgeMetadata` 里含 `senderClearance`（`sqlite.ts:1148-1152`），收件人可推断对方等级。
3. **`getConversation(id)` 不做任何权限校验**（`sqlite.ts:995-1000`，签名里没有 agentId）→ 拿到 UUID 就能读标题/默认可见性/创建者。`getProject` 只校验 visibility，**不校验 `access_mode`/成员资格**（`sqlite.ts:715-724`；仓库自己的测试把这点写死了：`src/__tests__/task-access.test.ts:106-125`「store 层不强制 hidden-project membership，handler 必须 gate」）。
4. **任务可绕过成员检查**：`list_tasks/get_task` 只要 `getProject` 通过就返回（`sqlite.ts:1516-1526`）。
5. **驳回语义泄露**：`ACCESS_DENIED` 与 404 用了同一个文案（`bridge/server.ts:145`），设计上是防探测的（好），但 `subscribe` 对 hidden 项目直接 `ACCESS_DENIED`、对可见项目创建 access request（`:536-555`）——**响应差异本身**就能区分「项目存在但隐藏」与「项目可见但非成员」。

### 4.4 记忆如何注入 prompt

- 共享记忆：**全量取最近 10 条**（不是检索、不是相关性排序），每条 200 字截断（`context.ts:354,520-529`）。
- 私有记忆：三作用域**全文拼接**，无截断（`context.ts:531-540`）。
- 无 embedding、无 FTS、无 tag 加权；`tags` 只在 `getMemory` 的过滤器里用（`sqlite.ts:928-931`）。

---

## 5. 技能系统（progressive disclosure）

**结论**：两段式元数据是真的（metadata 在订阅时给，content 要 `get_skill` 再取），但**「节省 80-90%」没有任何计算逻辑**，是营销数字；而且内部席位的 prompt 里其实注入了 `instructions`（第三段），所以真实是 3 段。

字段（`src/store/types.ts:283-336`）：

```
Skill        { id, title, summary, instructions, scope('bridge'|'project'|'conversation'),
               scopeId, selector{type?, capability?}, agents[], tags[], content, files[], createdBy, ... }
SkillMetadata = Skill 去掉 content
SkillFile    { id, skillId, filename, content }
CreateSkill  { scope, scopeId?, title, summary?, instructions?, selector?, agents?, tags?, content, createdBy }
```

三段的暴露时机：

| 段 | 内容 | 何时给 | 证据 |
|---|---|---|---|
| Tier 1 | `id,title,summary,instructions,tags,scope,agents,files` | `subscribe` 返回（含 already_subscribed 分支） | `bridge/server.ts:469-478`、`:520-533` |
| Tier 1.5 | 仅 `instructions` | 内部席位每次 prompt 的 `## Skills` 段 | `context.ts:509-517` |
| Tier 2 | `content` | 显式 `get_skill(skill_id)` | `bridge/server.ts:841-846`（`store.getSkill`）|
| Tier 3 | 文件内容 | 显式 `get_skill_file(skill_id, filename)` | `bridge/server.ts:1936-1951`（`store.getSkillFile`） |

匹配规则（`sqlite.ts:1867-1899`）：
1. 取 `bridge(scope_id='')` ∪ `project(scope_id=projectId)` ∪ `conversation(scope_id=convId)`，排序 `bridge < project < conversation`，同 scope 按 `created_at ASC`（`:1873-1881`）。
2. `agents[]` 非空 → 必须包含该 agent 名（大小写不敏感，**AND** 语义，`:1886-1891`）。
3. `selector.type` / `selector.capability` → 必须匹配（`:1893-1896`）。
4. 无 tag / 无相似度 / 无「按需命中」判断——**订阅即全量匹配**，没有「上下文相关性触发」这回事。

「节省比例」：`FEATURES.md:167` 与 `README.md:99` 写 `~80-90% context savings`，全仓 grep `savings|80-90|percentSav|contextSaving` **零命中**（无埋点、无计算、无测试断言）。→ 「文档推断，未见实现」。真实可蒸馏的只是「tier-1 只给 summary+instructions」这个**形状**，比例要自己按字符数算。

技能的实际用途在 Agorai 里是「把协议写进 agent 的 prompt」：Keryx 启动时把当前模式的协议写成 bridge 级技能（`module.ts:689-710`），Delegation Protocol 也这样注入（`bridge/server.ts:1460-1480`）。

---

## 6. 任务 / 协作原语

### 6.1 原子任务认领（防抢）

```sql
-- src/store/sqlite.ts:1552-1554
UPDATE tasks SET status='claimed', claimed_by=?, claimed_at=?, updated_at=?
WHERE id=? AND status='open'
```
`changes === 0` → 返回 null（别人抢到了）（`sqlite.ts:1556`）。**单条 UPDATE 的条件更新就是全部并发控制**（SQLite 单写者 + `status='open'` 谓词），没有锁表、没有重试。

- 完成：`WHERE id=? AND claimed_by=? AND status='claimed'`（`:1566`）。
- 释放：允许 **认领者或创建者** 释放（`:1580`）。
- 修改：**只有创建者**能改（`:1590-1595`）。
- 失联回收：每次 `claimTask/listTasks` 前先跑 `releaseStaleTaskClaims()`，把 `status='claimed' AND agents.last_seen_at < now-5min` 的任务打回 open，并逐条发 `task:updated(released)`（`:1448-1468`）。**回收触发点是「有人调用」，不是定时器**。
- 字段：`Task{ id, projectId, conversationId?, title, description, requiredCapabilities[], status('open'|'claimed'|'completed'|'cancelled'), createdBy, claimedBy, claimedAt, completedAt, result, ... }`（`types.ts:242-257`）。
- 语义：`requiredCapabilities` **只是标签，认领时不校验**（`claimTask` 只校验 status，`:1547-1561`）——「能力匹配」靠调用方自觉。

### 6.2 定向消息 / whisper

- 数据形状：`Message.recipients: string[] | null`（`types.ts:136-137`）；`CreateMessage.recipients?: string[]`（`:192-194`）；最多 20 个收件人（`tools.ts:131`）。
- 存储：`messages.recipients` JSON 列（`sqlite.ts:1183`）；`bridgeMetadata.whisper=true, recipients=[...]`（`:1155`）。
- 读门控：`if (msg.recipients) { if (sender!==me && !recipients.includes(me)) return false }`（`sqlite.ts:1244-1250`）。
- 一致性校验：发送时若正文 @ 了不在 `recipients` 里的 agent → **直接报错拒绝**（`bridge/server.ts:620-635`），防止「以为是密语其实对方看不到」。
- 离线提醒：人类/外部 agent @ 到离线席位时，系统以 `system` 身份 whisper 一条「⚠️ X is currently offline — do not wait for a reply」，5 分钟去重（`bridge/server.ts:1819-1868`）。

### 6.3 能力发现（capability discovery）

`discover_capabilities{capability?}` → `findAgentsByCapability` = `listAgents()` 全表 + `capabilities.some(c => c.toLowerCase() === lower)` 精确匹配（`sqlite.ts:662-666`）。返回全部 agent 字段但剥离 `apiKeyHash`（`bridge/server.ts:284-299`）。**没有打分、没有在线度排序、没有延迟/成本画像**；在线状态靠 `lastSeenAt`（5 分钟阈值）。

### 6.4 Agent 错误上报

约定：失败时发 `type:"status"` + `tags:["agent-error"]` + 正文 `[agent-error] Failed to generate response: <120字截断>`（`internal-agent.ts:438-449`）。Keryx 在 `handleMessage` 里**先于一切**拦截这类消息（`module.ts:157-162`）。语义 = 「把该席位从本轮摘除」，不是「重试」。

### 6.5 Pass-key（DB 托管密钥）

- 生成：`randomBytes(24).toString("base64url")`（`cli.ts:1223`）→ `HMAC-SHA-256(salt, key)`（`bridge/auth.ts:37-42`）→ 只存 hash 进 `agents.api_key_hash` → **明文只打印一次**（`cli.ts:1237-1243`）。
- 轮换：已存在且非 `internal:` 前缀时拒绝，除非 `--force`（`cli.ts:1210-1220`）。
- 吊销：`key revoke` 删除 hash（`cli.ts:1146-1147` 分派）。
- 校验链：`ChainAuthProvider` = `DatabaseAuthProvider`（DB 查 hash，`:115-147`）→ `ApiKeyAuthProvider`（config 明文/`keyEnv`，`:52-106`）。
- 保护：`registerAgent` 不允许 `internal:` 前缀的内部 hash 覆盖真实 DB hash（`sqlite.ts:567-571`）。
- 内部席位用 `internal:<name>` 作为 hash（`internal-agent.ts:77`、`module.ts:33`），即**无凭证的进程内身份**。

### 6.6 Tool profiles / tool groups

- 7 组 42 工具：`core(14) memory(6) tasks(6) skills(6) access(3) members(3) attachments(4)`（`bridge/server.ts:148-178`）。
- 3 个 profile 白名单：`agent(11)` / `orchestrator(20)` / `admin(42)`（`:180-198`）；**profile 优先于 group**（`:202-209`）。
- 效果不只是「工具不可见」，还会**裁剪 MCP instructions 章节**（`:212-222` + `context.ts:105-113,156-225`）——按 profile 决定要不要讲 access request、membership、skills、attachments、delegation。
- 注意：`core` 组永远打开（`:218`），所以 `agent` profile 实际是「core ∩ AGENT_TOOLS」，不是「core 全部」。

### 6.7 其它协作原语（一句话）

- 结构化消息类型：`message|spec|result|review|status|question|proposal|decision`（`tools.ts:128`）。
- 委派协议（**纯约定，无强制**）：`proposal + tag action-request` → `status + tag action-accepted` → `result + tag action-result`（`context.ts:206-215`、`llms.txt:100-108`）。
- 访问请求：非成员订阅 → 自动创建 `access_requests(status='pending')`；审批有第三种 `silent_deny`（请求方永远看到 pending）（`types.ts:203`、`tools.ts:162`）。
- 消息标签与按 agent 过滤：`GetMessagesOptions{tags, fromAgent}`（`types.ts:230-236`）。

---

## 7. 持久化与可回放

**结论**：**消息与订阅是持久的，编排状态完全不是**。Keryx 的全部轮次/暂停/模式都在 `Map<string, ConversationState>` 里（`module.ts:50`，注释 `:49` 明说 ephemeral, not persisted），进程重启即归零；消息 ID 是 UUID v4（无序），排序只靠 `created_at` 毫秒 ISO 字符串（无序号、无 tiebreaker）；SSE 断线补拉靠**客户端时间戳**，没有 `Last-Event-ID`、没有去重。

表结构（`sqlite.ts:84-320`，15 张表）：`agents, projects, project_members, project_memory, conversations, conversation_agents, messages, message_reads, agent_high_water_marks, access_requests, tasks, agent_memory, skills, skill_files, users, sessions, message_attachments`。

| 关注点 | 实现 | 证据 |
|---|---|---|
| conversation/session 表 | `conversations(id, project_id, title, status, default_visibility, created_by, created_at, updated_at)`；**没有 session 表**，Keryx 的「会话状态」不落库 | `sqlite.ts:133-143` |
| 消息表 | `messages(id, conversation_id, from_agent, type, visibility, content, tags, recipients, metadata, agent_metadata, bridge_metadata, created_at)` | `sqlite.ts:156-169` |
| 事件模型 | 进程内 `EventEmitter`：`message:created` / `access-request:created` / `task:created` / `task:updated` | `src/store/events.ts:30-95` |
| 事件是否持久 | **否**。无 outbox、无 event log 表；订阅者掉线期间的事件直接丢 | `events.ts:37-49` |
| 消息 ID | `randomUUID()`（v4） | `sqlite.ts:1136` |
| 顺序 | `ORDER BY created_at ASC`，`created_at = new Date().toISOString()`（毫秒） | `sqlite.ts:67-69,1236` |
| 同毫秒并发 | **没有 tiebreaker**（无自增 seq），同 ms 内顺序不确定 | 同上，索引也只是 `(conversation_id, created_at)`（`sqlite.ts:169`） |
| 已读 | `message_reads(message_id, agent_id, read_at)` 逐条；`mark_read` 事务批量 | `sqlite.ts:171-178,1324-1335` |
| 断线重连 | 前端 `EventSource` 断开 → 指数退避 1s→16s → 用 `?since=<最后一条的 createdAt>` 拉 `messages-since` → 再重连 | `src/gui/public/sse-client.js:103-142` |
| 补拉去重 | `getMessages` 用 `created_at > since`（严格大于），**同毫秒消息会丢**；`messages-since` 还会把「自己的消息」过滤掉（避免重复渲染） | `sqlite.ts:1221-1224`、`src/gui/routes/conversations.ts:317-354` |
| SSE 载荷 | GUI：**整段 HTML**（EJS 渲染）直接塞 `data:`；MCP：`notifications/message` 只给 200 字 preview + `messageId` | `src/gui/routes/sse.ts:38-74`、`bridge/server.ts:1872-1889` |
| 通知可靠性 | fire-and-forget：`transport.send()` 失败只 debug 日志，注释明说「agent will fall back to polling」 | `bridge/server.ts:1915-1920` |
| 重放 | **无**「从头重放某轮」能力；`Round.responseMessageIds` 只在内存里 | `types.ts:32-33` |

因此「不重复不丢」实际上靠三条弱保证：①`mark_read` 在成功发送之后才执行（`internal-agent.ts:421-423`），保证「没答就会再看到」；②`since` 严格大于；③各席位自己的 `already responded after trigger` 去重（`internal-agent.ts:308-318`）。没有幂等键、没有服务端游标。

---

## 8. 人机同场

**结论**：人类是「特权外部 agent」——拥有 `type:"human"` 的 agent 记录，绕过成员检查（`sqlite.ts:729,739,1015`、`bridge/server.ts:490-509`），但**不占席位**、**不能隐身**、**所有编排操作都要靠打字命令**。

人类能做什么（GUI 侧逐项）：

| 动作 | 入口 | 语义/作用域 | 证据 |
|---|---|---|---|
| 发消息 | `POST /c/:id/send` | 普通消息，可带 `attachment_ids`；`visibility` 表单里**没有选择器**，只能取默认 | `src/gui/routes/conversations.ts:655-690`、`views/conversation.ejs:54-77` |
| 上传文件 | `POST /c/:id/upload`（14MB 限制） | 两步：先 upload 拿 id，再随消息 link | `conversations.ts:537-597`、`bridge/server.ts:649-665` |
| 加/移除参与者 | `POST /c/:id/add-participant` / `remove-participant` | 直接改订阅；移除即退订并删该会话的私有记忆 | `conversations.ts:455-536`、`sqlite.ts:1072-1080` |
| 建项目/会话、隐藏/显示、改名、删除 | `/c/create-project` 等 | 隐藏项对非成员不可见 | `conversations.ts:126-315` |
| Keryx 命令 | 输入框 slash 菜单插入文本 | `pause/resume/skip/extend/status/interrupt/enable/disable/summary`（GUI 没有 `mode` 项） | `views/conversation.ejs:349-357`、`keryx/commands.ts:25` |
| 审批用户 | admin 面板 | 注册需审批，`pending` 不能登录 | `gui/routes/admin.ts`、`types.ts:389-409` |
| 查看即已读 | 打开会话页 | 页面加载时把**全部可见消息**标已读 | `conversations.ts:381-385` |

命令语义与作用域（`module.ts:232-422`）：
- 作用域 = **单个 conversation 的内存状态**（`state.paused/disabled/mode/currentRound`），不跨会话、不落库。
- `pause` 只挡住「开轮/记账」：`handleMessage` 里 pause 检查在 mode 内部（`ecclesia.ts:28`），但**已经开着的轮的定时器不会被清**——暂停后仍会 nudge 和强制关轮。
- `skip` 立刻关轮并清 timer（`module.ts:245-271`）；`interrupt` 把轮置 `interrupted` 并等**发起者本人**下一条消息才恢复（`ecclesia.ts:464-496`、`:53-58`）。
- `extend`：**只清 timer、重置 escalationLevel，从不重新 arm**（`module.ts:294-304`，代码里留着 `TODO: expose a resetTimer method`）→ 执行后本轮**永远不再超时关闭**，只能靠人类再说 `skip`。
- `summary`：若当前轮开着 → 置 `synthesizing` 后**空实现**（`module.ts:469-487`，注释「For now, reuse the pattern from the old monolithic code」）→ 本轮卡住直到 2×base 的综合超时把它关掉；若不在轮中 → 拿最后一条历史轮做一次 `synthesizing`（`:488-502`），同样没有真正的委派调用。
- 授权：任何**非 keryx、非 `internal:` 前缀**的 agent 都能发命令（`module.ts:224-225`）——**席位自己就能 pause/skip/改模式**；且命令匹配是**正文任意位置的正则**（`commands.ts:25`，测试 `keryx.test.ts:736` 明确「command embedded in longer text」），所以一条讨论正文里引用 `@keryx skip` 就会真的跳过本轮。

GUI 呈现面的缺失（影响交互形状）：消息气泡不渲染 `visibility`、`whisper`、`tags`、`level`（`views/partials/message.ejs` 只有 `createdAt` 与作者），参与者面板只有 online/error 三色点（`participants.ejs:21-30`）——**人类看不到「这条是密语」「这条是机密」**，与 §4 的「隐藏」哲学正好相反：agent 被隐藏，人类也没被告知。

---

## 9. 明确的反面教材（源码里能看出的坑）

### 9.1 数值污染：把绝对时间当耗时，跨模式污染并打爆定时器

`socratic.ts:233` 传 `Date.now()` 作为 `responseTimeMs` → `module.ts:660-673` 写入 `agentProfiles` → `timing.ts:80-85` 与毫秒级历史均值做 60/40 混合 → 结果约 `1e12 ms` 量级 → `setTimeout` 超过 `2^31-1`（约 24.8 天）会被 Node **截成 1ms**。后果：该席位的后续超时几乎立即触发；又因为 `agentProfiles` 是 **module 级全局 Map**（`module.ts:53`），同一席位在**其它会话的 Ecclesia 轮**也会被立刻判超时 → 轮次刚开就关。这是「一个模式跑一次，另一个模式全废」的经典共享可变状态事故。

### 9.2 轮次配额按「会话生命期」计，且用尽后静默

`roundNumber = roundHistory.length + 1`（`ecclesia.ts:105`）；`roundHistory` 全仓**无清空**（仅 `module.ts:584` 初始化与各 `push`）；`> maxRoundsPerTopic` 时 `return` 且**不发消息**（`ecclesia.ts:107-110`）。叠加 §8 的「人类不占席位、只触发开轮」，会话在累计 3 轮后**对人类任何发言都毫无反应**；而重启进程又会让 `roundHistory` 归零、`paused/disabled/mode` 全部重置成默认 `ecclesia`（`module.ts:42,578-589`）。「看起来在工作」的最坏形态：UI 正常、消息正常入库、没有任何报错。

### 9.3 失败重试无上限：不 mark_read 的 retry 循环

`invokeAndSend` 只在成功/限流时 `markRead`（`internal-agent.ts:403-431`）；普通异常路径**不标记已读**（`:425-452`），于是下一轮 poll（3s）会再次拿到同一条 Round-open，再调一次 LLM、再失败、再发一条 `[agent-error]` status。而 Keryx 的去重条件排除 `type==="status"`（`:313`），所以这条错误消息**不会**阻止重试。结果：一个坏 endpoint 的席位会以每 3 秒一次的频率持续烧 API 调用，直到进程退出；同时每 3 秒往会话里灌一条错误状态消息。仓库没有 retry budget / backoff / 熔断（只有 429 有指数退避 `30s*2^n` 上限 300s，`:143-147`）。

### 9.4 文本模板即协议：改一句文案，编排静默失效

席位靠**正则匹配 Keryx 自己的文案**来判断「是否该我说话」：
- 开轮：`/Round \d+/i && /Participants:/`（`internal-agent.ts:277-279`）；
- 综合/催促：`/🔄|Please synthesize|⏰|Please respond/`（`:284`）；
- 续轮 vs 首轮：`/Continuing:/`（`:324`）；
- 是否包含我：`@<agentName>` 子串（`:271-274,302`）。

`templates.ts` 里任何一个 emoji、大小写、字段标签的改动都会让**所有内部席位同时失聪**，而类型系统、测试都不会发现（测试只断言模板字符串本身）。这是「LLM 编排用自然语言当 RPC」的典型代价。

### 9.5 「半成品交互」比没有更危险

`@keryx extend` 不重启计时器（§8）、`@keryx summary` 空实现（§8）、`wild-agora` 只有类型没有文件（`mode-interface.ts:15` vs `module.ts:77-78`）、`memory/sqlite.ts` 整层 `throw not implemented`（`:35-97`）、`src/bridge/server.ts` 调用不存在的 store 方法（`:1503,1543,1567`）。这些都能通过「看起来有入口」骗过评审；运行时表现为静默挂起或崩溃。

### 9.6 共识用 LLM 自报置信度 + 关键词选协议（CLI 路径）

`orchestrator.ts:132-154` 用关键词计数选协议（security 词 ≥2 → quorum；comparison 词 ≥2 → vote；否则 debate），而 `getConsensusImpl` 把 `quorum` **直接映射到 `VoteConsensus`**（`:156-164`）——「quorum」是空壳；`vote.ts` 的权重是 `selfConfidence × personaBonus`，`selfConfidence` 由 LLM 在回复末尾自报 `[confidence: 0.XX]`，缺失时默认 `0.5`（`adapters/base.ts:39-48`）。`debate.ts:11-13` 自认「no convergence tracking or synthesis」。把「LLM 自评分」当投票权重，是最容易被「所有模型都学会说 0.9」击穿的机制。

### 9.7 可见性在 JS 层过滤 + 无 SQL LIMIT

`getMessages/getMemory` 都是 `SELECT *` 全量取回后在 JS 过滤再 `slice`（`sqlite.ts:1218-1258,912-935`），注释还专门解释了「因为过滤在 JS，所以不能加 SQL LIMIT」。后果：①长会话每次读都全表扫描该会话消息（含不可见的）；②`mark_read` 走 `getMessages` 全量再逐条插入（`bridge/server.ts:1146-1149`）；③内存与延迟随历史线性增长，且**不可见数据也参与扫描**——「隐藏」只在返回层成立，资源侧仍然可被探测（时序）。

### 9.8 编排状态不持久 + 事件不落库 + 无幂等序号

`module.ts:50` 的 `Map` 就是全部编排状态；`events.ts` 的 EventEmitter 无 outbox；消息 ID 是 UUID、排序靠毫秒时间戳（§7）。一次崩溃/重启 = 轮次全丢、模式回默认、暂停失效；一次断线 = 依赖客户端时间戳补拉，同毫秒丢消息。任何「可回放 / 可审计 / 断点续议」的需求在这个模型上都做不出来。

### 9.9 人类权限与席位权限没有分层

席位可以 `pause/skip/mode`（§8），可以写超 clearance 的记忆（§4.2），可以 `silent_deny` 别人的访问请求（`tools.ts:162`）；而人类看不到 whisper/visibility（§8）。治理方向是反的：**执行者拥有编排权，观察者失去观察权**。

### 9.10 每 10 秒一次全量发现（N+1）

`module.ts:106-112` 每 10s 跑 `discoverConversations()`：`listProjects` → 每个 project `listConversations` → 每个 conversation `isSubscribed`（`:558-593`）。Keryx 以 `restricted` clearance 注册（`:94`），能看到所有非隐藏项目 → 项目/会话数增长时这是 O(projects × conversations) 的 SQL 轮询，且它自己**订阅全部会话**（`:574`），意味着所有会话都被判为「Keryx-managed」（见 §9.11）。

### 9.11 订阅即接管：Keryx 在场 = 全员被门控

席位判断「是否需要回应」的依据是「会话里有没有 keryx/orchestrator 订阅者」（`internal-agent.ts:261-268`）。而 Keryx 会自动订阅**每一个**它能发现的会话（`module.ts:572-576`）。于是：
- 一旦 Keryx 启动，所有会话的席位都进入「只回应 Round-open / 直接请求」模式，**普通对话不再触发回复**；
- `@keryx disable` 之后 `state.disabled=true`（`module.ts:360`）→ 不再开轮 → 席位仍在门控模式（`isKeryxManaged` 依然为真）→ **整个会话彻底失声**，且没有任何提示；
- 反过来，`unsubscribe` Keryx 或让它崩溃，会话又会回到「自由讨论」模式。同一个开关同时控制「编排」和「是否允许说话」，语义耦合错误。

---

## 候选蒸馏点（8–15 条）

[1] 六态轮次状态机 — `idle→open→collecting→synthesizing→closed`（+`interrupted`），状态迁移只由「人类消息 / 席位消息 / 定时器」三类事件触发，编排器只写 status 消息、从不产内容 — `src/keryx/types.ts:10-42`、`src/keryx/modes/ecclesia.ts:27-59,190-246` — 高
[2] 首轮盲写 / 次轮全量 — 用「开轮消息时间戳」截断上下文，Round 1 只看开轮前，Round 2+ 才看全场；一条 `createdAt` 比较就实现「独立陈词→互相启发」两阶段 — `src/agent/internal-agent.ts:320-325`、`src/agent/context.ts:407-411` — 高
[3] 多数已答 + 宽限关轮 — `responded > expected/2` 时不再等全员，改等固定 10s 收割掉队者再关轮，兼顾时延与完整度 — `src/keryx/modes/ecclesia.ts:385-402` — 高
[4] 四因子自适应超时 + 历史混合 — `c=0.4*词/500+0.25*代码块/3+0.2*问号/5+0.15*技术密度/10`；`t*=0.5+1.5c`；`t=t*0.6+avg*1.5*0.4`；首轮×1.5、后续×0.8、>3 席每席 +10% — `src/keryx/timing.ts:20-46,57-101` — 中（公式可取，`avg` 的接线方式必须重写）
[5] 逐级升级（nudge→强关） — 超时先 @ 未答者催一次，再等 `base*0.5` 强制关轮；两级即可，避免「无限等」 — `src/keryx/modes/ecclesia.ts:374-442`、`src/keryx/templates.ts:57-66` — 中
[6] 席位错误即摘除 — 失败席位主动发 `status+tags:["agent-error"]`，编排器把它从 `expectedAgents` 删除并立即重算完成度，而不是等超时 — `src/agent/internal-agent.ts:438-449`、`src/keryx/module.ts:157-162`、`ecclesia.ts:449-458` — 高
[7] 综合委派的三级兜底 — 能力匹配（`synthesis`）→ 与订阅者取交集 → 随机挑一个本轮已答者 → 都没有则关轮；综合超时独立为 `2×base` — `src/keryx/modes/ecclesia.ts:248-298` — 高（选人策略），输入侧不可取
[8] 中断/续会语义 — `interrupt` 把轮置 `interrupted` 并只认「打断者本人的下一条消息」来恢复，把话语权交回打断者 — `src/keryx/modes/ecclesia.ts:464-496`、`module.ts:342-351` — 中
[9] 4 级可见性 + 写上限 + 高水位 — `public<team<confidential<restricted` 顺序映射；写入按发送者 clearance 裁剪并留 `visibilityCapped/originalVisibility` 审计；读侧记账 `max_visibility` 只增不减 — `src/store/types.ts:17-22`、`src/store/sqlite.ts:1139-1156,1260-1302` — 高
[10] 可见性「沉默」契约 — 越权数据在返回层直接消失：不返回条数、不返回 `hidden_count`、unread 只数可见、SSE 不发；配套「拒绝与不存在同文案」防探测 — `src/store/sqlite.ts:1241-1254,1337-1350`、`bridge/server.ts:145,1891-1905` — 高
[11] whisper + @mention 一致性校验 — 定向消息 `recipients[]`，读侧双向门控；发送时若 @ 了非收件人直接拒发，消除「以为私聊其实没发到」 — `src/store/sqlite.ts:1244-1250`、`bridge/server.ts:620-635` — 高
[12] 原子认领 + 失联回收 — 单条 `UPDATE ... WHERE status='open'` 做 CAS；回收靠 `last_seen_at < now-5min` 的惰性清扫（读时触发） — `src/store/sqlite.ts:1547-1561,1448-1468` — 高（gaea 侧无任务原语）
[13] 三段式渐进披露技能 — `summary/instructions`（tier1，随订阅下发）+ `content`（tier2，`get_skill`）+ `files`（tier3）；匹配维度 `scope ∪ agents[] ∪ selector{type,capability}` — `src/store/types.ts:283-314`、`src/store/sqlite.ts:1867-1899`、`bridge/server.ts:511-533` — 中
[14] DB 托管 pass-key 生命周期 — 24 字节随机 → HMAC-SHA-256(salt) → 只存 hash → 明文仅打印一次 → `--force` 轮换 / `revoke` 吊销；内部身份用 `internal:` 前缀且禁止覆盖真 hash — `src/cli.ts:1166-1245`、`src/bridge/auth.ts:37-42,115-147`、`sqlite.ts:567-571` — 中
[15] 预算分档降级 — 80% 摘要压缩（截 200 字）/ 90% 减到单席 / 95% 本轮收尾 / 100% 硬停，每档都写 `budgetActions` 供事后解释 — `src/orchestrator.ts:272-306`、`memory/base.ts:42-49` — 中

---

## 不要抄的清单

1. **短语白名单当共识**（`patterns.ts:13-36` + `ecclesia.ts:218`）：英文子串匹配 + 用「已答数」当分母，一人弃权即终局、正文引用即误判。要抄就抄「显式弃权通道」，判定必须换成结构化字段（立场/分歧/置信度）。
2. **轮次配额按会话生命期计且用尽后静默**（`ecclesia.ts:105-110`）：第 4 轮开始人类说话没有任何反馈。配额必须按「议题」而非「会话」计，并且必须显式回执。
3. **把编排写进文本模板再用正则读回来**（`internal-agent.ts:277-289`）：改文案即断链、无法类型检查、无法本地化。编排信令必须是结构化字段/事件。
4. **内存态编排 + 无事件日志 + 无单调序号**（`module.ts:50`、`events.ts`、`sqlite.ts:1136,1236`）：重启丢轮、断线丢消息、同毫秒乱序，做不出审计与回放。
5. **「入口存在、实现留空」的半成品**（`module.ts:299-304,482-487`、`memory/sqlite.ts:35-97`）：`extend` 不重启计时器、`summary` 空转、Blackboard 整层 throw，比没有更危险。要么删入口，要么 fail loud。
6. **共享可变 profile 里混绝对时间**（`socratic.ts:233` + `timing.ts:80-85`）：一次调用污染全局、跨会话生效、并把 `setTimeout` 打到溢出。耗时统计必须在调用边界用 `start/end` 差值。
7. **失败重试无预算、靠「不 mark_read」自然重试**（`internal-agent.ts:425-452`）：3 秒一次无限重试 + 状态消息灌屏。重试必须有上限、退避与熔断。
8. **可见性在 JS 层全量过滤 + 无 SQL LIMIT**（`sqlite.ts:1218-1258,912-935`）：长会话线性退化，且不可见数据仍参与扫描。要么在 SQL 里过滤/分页，要么接受「返回层过滤」但必须分页 + 索引。
9. **「Keryx 在场即全员门控」的隐式耦合**（`internal-agent.ts:261-268` + `module.ts:572-576`）：一个订阅者决定了整场会话的发言规则，`disable` 等于静音。编排开关与发言许可必须解耦。

（9 条，超出要求的 5–8 条，因为其中 3 条是本仓库最贵的教训，建议全部纳入 gaea 的评审 checklist。）

---

## 附：调研边界与未读清单

- 已精读：`src/keryx/*`（全部）、`src/orchestrator.ts`、`src/consensus/*`、`src/agent/context.ts`、`src/agent/internal-agent.ts`、`src/store/{types,events,interfaces}.ts`、`src/store/sqlite.ts`（表结构 + 记忆/消息/任务/技能/项目/成员相关方法，约 900 行）、`src/bridge/{server,tools,auth,validation}.ts`（工具注册与 SSE 分发段）、`src/memory/*`、`src/adapters/{base,openai-compat}.ts`、`src/cli.ts`（key 管理段）、`src/gui/routes/{sse,conversations}.ts`、`src/gui/public/sse-client.js`、`src/gui/views/{conversation.ejs,partials/*}`（交互面）、`src/__tests__/keryx.test.ts`（Phase 1–8）、`README.md`、`FEATURES.md`（grep）、`llms.txt`、`docs/ARCHITECTURE.md`、`CHANGELOG.md`（v0.8 以后各段）。
- 未逐行读：`src/store/sqlite.ts` 的 user/session/attachment 段（约 1300 行外）、`src/gui/routes/{api,auth,admin}.ts`、`packages/agorai-connect/*`（轮询型外部 agent，机制与 `internal-agent.ts` 同构，未验证差异）、`src/project-manager.ts`、`src/personas.ts`、`src/tools.ts`、多数 GUI 视图模板。
- gaea 侧对照仅基于 `internal/pantheon` 的文件清单与关键词检索（`visibilit|clearance|whisper|recipients|claim|skill|timeout` 等），**未逐行比对**；因此「对 gaea 的潜在价值」是机制层面的判断，不构成「gaea 没有 X」的断言。可确认的是：gaea 有 `seatstate/seatguard/seat_history/chair_envelope/chair_receipt/gates/discussion_act/attestation/DID`，且 `defaultAgentLimit` 是固定超时（`internal/pantheon/manager.go:1750`），未见 4 级可见性、whisper、任务认领、渐进披露技能这几类命名。
