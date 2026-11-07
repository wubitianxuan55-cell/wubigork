# Agorai × aidebate 双源蒸馏调研：议事厅该抄什么、不该抄什么（2026-11-07）

> **钉版**
> - Agorai：`StevenJohnson998/Agorai@1e469be645e975e8f38313ad431a364a494cfd28`（v0.8.0，2026-04-12，AGPLv3，TypeScript，src+packages ≈ 27.6k 行，含测试与模板）
> - aidebate：`orlenko/aidebate@80f3b77fa1b64a287928cab68b60c406d2ae0ba6`（v0.4.7，2026-04-16，Apache-2.0，Python，src+tests ≈ 3.2k 行）
> - 本地克隆：`clones/agorai`、`clones/aidebate`（均已 `--depth 1`，只读，未安装依赖、未跑测试）
>
> **纪律**
> - 取道不取器：只吸收机制、状态机、字段、判据、提示词形状，不移植代码。Agorai 是 **AGPLv3**，代码移植有许可污染风险，与本仓对 so-novel 的红线同款；aidebate 是 Apache-2.0，许可不挡，但纪律不变。
> - 上游均已静默约 7 个月（最后一次提交 2026-04），**钉版即止，不做滚动跟踪**；重评只 diff 相关文件。
> - 本文只调研、不立项、不写生产代码。所有事实带 `path:line`；只从文档推断的会标注「文档推断」。
> - 落点假设：**众神殿**（多席议事）。若目标另有其人（如办公 agent 协作面），§5 的判断需重做，见 §11 拍板项。

---

## 0. 结论先行

**一句话**：Agorai 是一个「多 agent 协作平台」，aidebate 是一个「把真实交互式 CLI 塞进 tmux 的辩论编排器」；两者都不是更好的议事协议，但都各自握着一件众神殿确实缺的东西——**Agorai 的「沉默是一等输出」，aidebate 的「开场前先证明这席能用」**。其余大部分能力，要么众神殿已有更硬的形态，要么与既有红线冲突。

三句话展开：

1. **协议层不必重开。** 两源的共识检测（关键词匹配）和超时（无上下限的自适应）都是我们已经明确拒绝过的做法；把它们当参考，只能当反例。
2. **真正的增量只有三小刀**：席位「无新增」语义（pass）、开场功能冒烟（canary）、议事健康度量的确定性指标。三刀都不改轮次协议、不加权限、不加持久化表。
3. **两源各自的大件（Agorai 的记忆/技能/可见性、aidebate 的 tmux 交互协议）都挂池**，复活条件写清楚——不是「没读到」，是「现在做没有对应的产品面」。

---

## 1. 两源事实层

| 维度 | Agorai | aidebate |
|---|---|---|
| 定位 | 多 agent 协作平台：Bridge（共享工作区）+ Keryx（讨论编排器）+ GUI | 单用户辩论编排器：把 Claude Code / Gemini CLI / Codex CLI 塞进 tmux pane 互辩 |
| 运行形态 | 常驻 HTTP 服务（MCP Streamable HTTP，默认 `127.0.0.1:3100`）+ Web GUI | 一次性进程：起 tmux 会话 → 五阶段 → 归档到 `~/.aidebate/sessions/<id>/` |
| 席位是什么 | 订阅某会话的 agent（MCP 客户端 / 内部 LLM 适配器 / Ollama） | 一个 tmux pane 里跑着的交互式 REPL |
| 编排器 | Keryx：纯确定性 TS，零 LLM 依赖，Ecclesia（并行轮次）/ Socratic（严格轮流） | 宿主进程：串行阶段 + 线程池，靠**文件旗标**协调 |
| 状态 | SQLite 7 表（agents/projects/project_memory/conversations/conversation_agents/messages/message_reads） | `session.json` 清单 + `chat.jsonl` + 每阶段 `*.prompt.md` / `*.answer.md` / `*.done` |
| 规模 | 109 个 TS/TSX（src+packages）、43 个测试文件、README 称 547 tests | 16 个 Python 源文件、10 个测试文件 |
| 许可 | AGPLv3（双许可可谈） | Apache-2.0 |
| 活跃度 | 最后提交 2026-04-12 | 最后提交 2026-04-16 |

两源的气质完全不同：Agorai 想当**基础设施**（任何模型、任何框架都能接进来，记忆与技能跨会话累积）；aidebate 明说是**一个人的玩具**（`CLAUDE.md` 原话：one-user toy turned actually-useful），功能裁剪极狠，只做「把订阅的 CLI 拉出来互殴」这一件事。

这个差异决定了它们对我们的价值位置：Agorai 值得看**产品面的取舍**，aidebate 值得看**不可靠执行体的协议设计**。

**证据强度警告（重要）**：Agorai 这个 checkout **自身不自洽**，不能当「可运行系统」读——

- `src/bridge/server.ts:1503/1543/1567` 调用了 `store.getPublicMessages` / `getActiveConversations` / `getConversationStats`，但 `src/store/` 里**没有任何定义**，`IStore` 接口也没有声明；
- `src/keryx/types.ts:46` 与 `mode-interface.ts` 声明了 `wild-agora` 模式，但 `src/keryx/modes/` 只有 `ecclesia.ts` 与 `socratic.ts`（CHANGELOG 声称的 `modes/wild-agora.ts` 全仓不存在）。

也就是说：这份代码大概率连类型检查都过不了，v0.8 是以「半成品」状态归档的。**这不改变机制调研的价值，但改变阅读方式**——Agorai 按「设计意图」读，不按「已验证行为」读；凡涉及上述缺口，本文不当证据。顺带也是一个结论：**停更 + 不自洽的上游，只能钉版取道，不能追。**

aidebate 没有这个问题（测试纯函数化、能跑），但它的 README 与源码同样不一致（见 §5.4 末）。两源共同教训：**文档是线索，源码才是事实。**

---

## 2. 为什么此刻值得一次蒸馏

众神殿的既有蒸馏线（Orca 两轮、OpenMausBot 两轮、MALLM/Captain Claw/ARTEMIS/AgentTeams 四框架）已经把「证据真相层、动作可核查、分歧台账、签注」这条主线铺完。留下的问题不再是「协议够不够硬」，而是三个更土的问题：

1. **一场会太贵。** 4 席 × 4 轮 × 单席 6 分钟，实弹跑过 83 分钟、140 分钟被用户取消（`docs/gaea-pantheon-seat-hang-cache-2026-11.md:9-46`）。
2. **话太多、信息太少。** 每席每轮都必须交一段完整陈词，模型没有「我这轮没新东西」的合法出口，于是复述、铺垫、表态。
3. **席位坏得晚。** 凭据缺失、权限弹窗、卡死，往往要跑完一轮才发现；发现的代价是一整轮的时间和额度。

Agorai 与 aidebate 恰好各自在这三点上给出了答案（见 §5.1、§5.2），而且**两个互不相干、气质完全相反的项目，独立收敛到了同一条设计原则：允许沉默**——Agorai 用 `[NO_RESPONSE]`，aidebate 用 `(pass)`，都没把「这轮没话说」当成错误。aidebate 还多走了一步：开赛前先证明每个席位真的能用。这种跨源收敛，比单一来源的功能列表更值得信。

---

## 3. gaea 现状基线（本轮已核）

| 能力 | 现状 | 证据 |
|---|---|---|
| 轮次模型 | 串行轮转；仅「独立开场」协议的第 1 轮并行 | `internal/pantheon/manager.go:1137-1158` |
| 席位历史投递 | 四形态：梯度分发 / 会话 Seed / 全量窗口 / 增量（引擎会话续接） | `prompts.go:377-409` |
| 主席机制 | 轮间小结、私有信封派工、终局信号、回执 | `manager.go:1379-1407`、`chair_envelope.go`、`chair_receipt.go` |
| 分歧台账 | 按分歧点记账、跨轮注入、逐条闭合、结案转 dissent | `types.go:435-483`、`manager.go:2059-2312` |
| 停机判据 | 全员一致（自报 + 剩余分歧 + 翻转举证）、DID（无新可执行核验则停） | `manager.go:3067-3090`、`did.go:28-77` |
| 证据 | Usage 账目、工具证据摘要、Observed 出处、Gaps、签注哈希 | `types.go:107-126/205-247/551-601` |
| 决策门 | 每轮门、席位提问、用户回答、用户插话 | `gates.go:103-206/406-520` |
| 席位护栏 | 工具预算 120 次、无进展 120s、开局静默 5 分钟 | `seatguard.go:40-48` |
| 席位探测 | 只探「可执行文件在不在 + 版本」 | `gates.go:635-657` |
| 协议标签 | `Protocol`（信息可见性轴）+ `Preset`（议题形状）+ 评测 | `types.go:183-192`、`eval.go` |

**结论**：协议层不欠账。缺的是「席位侧的诚实与效率」，以及「开场前的功能证明」。

---

## 4. 交叉对照矩阵

| 维度 | Agorai | aidebate | gaea | 判决 |
|---|---|---|---|---|
| 轮次推进 | Ecclesia 并行轮次 + 自动进下一轮；Socratic 严格排队 | 阶段串行，阶段内并行 | 串行轮转，独立开场并行 | 形态不同，无增量 |
| 共识判定 | 关键词匹配（`isConsensusResponse`） | 无（主持人裁决） | 自报 + 剩余分歧 + DID | **gaea 更硬，不抄** |
| 沉默语义 | `[NO_RESPONSE]` 是一等输出，且**被算作共识** | `(pass)` 合法，明说「silence is better than filler」 | 无 pass；每席每轮必发言 | **有增量（但要去掉「沉默=同意」）** |
| 超时 | 四因子自适应，**无上下限** | 固定 turn_timeout 300s + wallclock 300s + 静默 180s | 固定三闸（120s/5min/8min） | **不抄自适应；gaea 更可测** |
| 席位自检 | 无 | **canary handshake**（开场前每人写文件 + touch done） | 只探可执行性 | **有增量** |
| 交互式 CLI | 无（MCP/HTTP 客户端） | 核心能力：tmux + send-keys + 权限弹窗 + 启动键 | 无（headless CLI） | 形态不匹配，挂池 |
| 席位间定向消息 | @mention + 广播 | `chat-say --to role` + 每角色收件队列 + 批量 | 定向质询（宿主中介，指向贡献 ID） | gaea 近似已覆盖 |
| 记忆 | 项目共享记忆 + 每席私有记忆（3 作用域） | 无 | 场次存档（episodic，红线） | **红线冲突，不立项** |
| 可见性 | 四级 clearances，存储层强制过滤 | 无 | 单用户本地，无分级 | 挂池（等本机席位） |
| 技能 | 三段式渐进披露（元数据/正文/附件文件） | 无（adapter 即全部） | 有 skill 域，但席位不挂技能 | 挂池 |
| 健康检测 | 循环（Levenshtein>0.7）/漂移（余弦<0.3）/支配（>40%） | 静默监视 + 主持人催场 | seat_ring 环检测 | **有增量（作为展示指标）** |
| 人机同场 | `/pause /skip /mode /summary` 命令 | 无中途干预（只能 attach 看） | 门/暂停/插话/提问/回答 | **gaea 更强，不抄** |

---

## 5. 逐项机制

### 5.1 席位「无新增」语义（Agorai `[NO_RESPONSE]` + aidebate `(pass)`）

**上游怎么做**

- Agorai 把 `[NO_RESPONSE]` 写进系统提示词，并给出判定条件：「讨论正在收敛、要点已覆盖、你只会重复别人、已经有人总结过结论」（`src/agent/context.ts:468-470`）。
- 编排器把 `[NO_RESPONSE]` **当成共识票**：`isConsensusResponse()` 命中 `nothing to add / fully agree / consensus reached` 等 10 条短语即返回 true（`src/keryx/patterns.ts:15-35`），`closeRound` 用 `consensusCount === respondedAgents.size` 判定全场一致（`src/keryx/modes/ecclesia.ts:203-226`）。
- aidebate 更克制：`(pass)` 只是「本轮不发言」，明写「Silence is better than filler」，且**不进入任何一致性统计**（`src/aidebate/core/crossexam.py:194-210`）。

**gaea 现状**

每席每轮必须交一段陈词；`roundReachedConsensus` 要求本轮每个席位都有贡献且自报一致（`manager.go:3067-3090`）。模型没有「本轮无新增」的表达方式，只能在正文里注水。已有 `DiscussionAct` 六种动作（claim/challenge/respond/revise/reinforce/broaden），**没有 pass**（`discussion_act.go:67-86`）。

**差距**：不是「缺一个动作」，是**缺一个合法的低成本出口**。没有它，"全员发言"这个判据会持续奖励凑字数。

**吸收方式（P0）**：新增 `动作：无新增`（`DiscussionActPass`），语义三条：

1. **pass ≠ 同意**：不计入一致票，不参与 `roundReachedConsensus` 的分子；有 pass 的轮次按「未全员一致」处理，除非其余席位一致且 pass 方在上一轮已表态。
2. **pass 必须带一句理由**（≤1 行），且只能用于「无新证据、无待回应质疑」；宿主不校验真伪，但把理由落档，主席下一轮可点名要求其发言。
3. **收口判据扩展**：连续 N 轮（建议 N=2）全员 pass 且无 open issue/DID → 提前进入汇总，事件写明「无新增收口」，与 DID 停机同型但不同因。

**风险**：模型把 pass 当逃避手段，导致关键席位长期不表态。缓解：主席小结可点名；`Gaps` 不把 pass 记成缺口（它是主动行为，不是失败）；统计面板单列 pass 次数，让用户看得见谁在摸鱼。

**为什么这条是 P0**：它同时降本（少一次长文生成）、降噪（少一段复述）、提升判据质量（"没人有新东西"终于可表达），且不改轮次协议。

---

### 5.2 开场功能冒烟（aidebate canary handshake）

**上游怎么做**

- 起完 pane 后、第一阶段前，对**每个**席位并行跑一次 canary：让它把单词 `ok` 写进一个文件并 `touch` 完成标记，超时 120s（`src/aidebate/core/turn.py:118-162`）。
- 失败策略分两级：主持人 canary 失败 = **整场中止**；普通辩手失败 = 记 `dropouts[]`（role/agent/phase/error/at）并从本场移除，只要还剩至少一个辩手就继续（`src/aidebate/core/debate.py:344-397`）。
- 清单在开场前就落盘（`status: running` + `dropouts: []`），进程崩了也能看出「它死在哪一步」（`debate.py:282-296`）。

**gaea 现状**

`probeSeats` 只调用 `Agents()`：可执行文件存在、版本可读即算可用（`gates.go:635-657`）。真实的失败（DSH 凭据缺失 1.6s 结束、CLI 弹权限框等人、冷启动卡住）要等第一轮跑完才暴露；`Gaps` 是**结案时**才算的（`gates.go:65-87`）。

**差距**：可用性探针证明的是「装了」，不是「能用」。中间隔着一个凭据、一个弹窗、一个工作目录。

**吸收方式（P0）**：`Start` 时对每个参席跑一次最小调用（独立超时，建议 90s），判据三条：进程零退出 + 正文非空 + 未命中已知错误特征（`MISSING_CREDENTIAL`/`permission`/`sandbox denied`）。结果写进 `Participant`（`canary: ok|failed|skipped` + 一句原因），**失败即前置为 gap**，不进入第一轮。主席席失败按 aidebate 的口径处理（拒绝开场或自动换席，二选一，需拍板）。

**风险与边界**

- 成本：每席一次短调用（几十 token、数秒）。可在设置里给「严格开局」开关，默认开。
- 会额外触发一次引擎会话；若席位支持 `--session-id` 续接，应让 canary 会话继续用于第一轮，避免白开一个会话（DSH 场景）。
- canary 不能替代 `seatguard`：它只证明「起得来」，不证明「跑得完」。

---

### 5.3 议事健康指标（Agorai 三个纯函数探测器）

**上游怎么做**（全部确定性、零 LLM）

- **循环**：同一 agent 连续两条消息 Levenshtein 相似度 > 0.7 即判循环（`src/keryx/patterns.ts:80-110`）。
- **漂移**：当前话题与最近消息的余弦相似度 < 0.3 判漂移（`patterns.ts:112-190`，分词是英文停用词表）。
- **支配**：3 席以上时，某席消息数占比 > 40% 判支配（`patterns.ts:192-230`）。

**gaea 现状**：`seat_ring.go` 已有环检测；没有漂移与支配的度量；记录页只有「谁说了什么」，没有「这场会健康不健康」。

**差距**：用户看不出「这轮在原地打转」还是「真的在收敛」。这是一个**展示层缺口**，不是协议缺口。

**吸收方式（P1）**：纯函数三件，输出到记录页与导出，**只展示、不驱动**：

1. 发言分布（每席字数/轮次占比）——支配的可解释版本，比 40% 阈值更适合中文长文。
2. 轮次重复度（相邻轮同席文本的 2-gram Jaccard）——环检测的轮级视角。
3. 议题漂移（当前轮 vs 议题的 BM25/余弦）——复用仓内 `bm25` 与 CJK 分词，不要抄英文停用词表。

**明确不做**：不根据指标自动收口、不降权某席、不弹「检测到漂移」的告警。指标一旦驱动决策，就会变成第二个裁决者，与「主席只维护议程」的红线冲突。

---

### 5.4 定向消息与收件箱（aidebate crossexam）

**上游怎么做**

- 每个角色一个 worker 线程 + 队列，**同一角色同时只跑一个 turn**，正在跑时到达的消息批量合并进下一轮（`src/aidebate/core/crossexam.py:293-371`）。
- 消息带 `to: [role...]` 或 `*`；watcher 每 0.75s 读 `chat.jsonl`，只把**发给你的**消息投进你的队列（`crossexam.py:378-401`）。
- 发言用注入到 agent cwd 的 `chat-say` 脚本，一次 shell 调用写一条合法 JSONL，避免模型手搓格式（`crossexam.py:44-74`）。
- 终止：wallclock 300s **或** 连续 180s 无新消息（`crossexam.py:259-273/441-452`）；静默 60s 时单独催主持人（`crossexam.py:405-420`）。

**Agorai 有同型机制**：`whisper` 定向消息带 `recipients[]`，且「@ 了非收件人直接拒发」（`src/bridge/server.ts:620-635`）；比 aidebate 多一道「收件人强制」，思路相同——**消息有地址，地址可校验**。

**gaea 现状**：席位之间不直接通信，全部经宿主转述；定向质询是「指向贡献 ID」的宿主中介（`prompts.go:411-436` 的定向质询段、`Issue.TargetContributionID`）。终止靠轮次/一致/DID，不靠静默。

**判断**：**不立项**。理由三条：

1. gaea 的席位是一次性 CLI 进程，没有「常驻收件箱」；要造就得造一个常驻消息总线，等于把众神殿改成聊天服务器。
2. 「静默终止」在 gaea 已有更强形态：DID 是**内容判据**（没有新的可执行核验就停），比「一段时间没人说话」更难被拖时间骗过。
3. 「发给你的消息」在 gaea 等价于「本次提示词里与你相关的台账条目 + 被挑战的命题」，已经是定向的。

**但有一条值得留**：`chat-say` 这种「给模型一个专用小工具，而不是让它手搓协议格式」的思路，与本仓 `DiscussionAct` 的容错解析是两种路线。我们选了容错解析（协议行可选、解析失败不吞正文），在异构 CLI 上更稳，维持。

**为什么这套协议不能搬：三个已发生的失效模式**（详见 `docs/research/aidebate-distill/01-source-analysis.md` §9）

| 编号 | 失效模式 | 证据 | 教训 |
|---|---|---|---|
| F1 | 交叉质询收尾只等 30s，下一阶段立刻对**同一 pane** 发新 prompt → 两个回合同时打字、两个 `.done` 互相冒充 | `crossexam.py:456-461` → `debate.py:480-486` | 回合所有权必须显式化（锁/租约/独占句柄），不能靠「应该跑完了」 |
| F2 | `run_turn` 从不清理旧旗标（canary 却会清）→ 目录被重入时返回旧 answer，全链路无人察觉 | `turn.py:72-75` vs `124-127` | 同一协议两套卫生标准 = 给假成功留正门 |
| F3 | claude/codex 没有 `ready_patterns`，`wait_until_ready` 第一行就 `return True`，就绪判据退化为 `sleep 3.0` | `pane.py:53-54`、`turn.py:91` | 固定 sleep 是调参型可靠性，换机器就复现不了 |

这三条都不是「实现 bug」级别，而是**传输层选型**的必然代价：pane 是给人看的，没有帧、没有边界、没有 ack。gaea 用 `--output-format stream-json` 绕开了整个问题（`agents.go`），**不要为了「能看它思考」把 PTY 请回来**；要看过程，用结构化事件流比抓屏可靠一个数量级。

顺带记一笔**文档失实**：aidebate README 写的是三阶段（opening/rebuttal/verdict，目录 `phase-2-rebuttal`），源码是五阶段（`phase-2-crossexam`/`phase-3-rebuttal`/`phase-4-verdict`/`phase-5-roast`）；README 说「not terminal scraping」，实际全程读屏。**这也是本文所有判断以源码为准、不以 README 为准的原因。**

---

### 5.5 材料可见性与席位信任（Agorai 四级 clearance）

**上游怎么做**：每条实体带 `visibility`（public < team < confidential < restricted），**所有读操作在存储层按 agent 的 clearance 过滤**，越权行对该 agent 不存在（`src/store/sqlite.ts:4-5/716-736/909-926`）；提示词还要求「回复的密级取输入的最高级，永不降级」（`src/agent/context.ts:123-129`）。

**gaea 现状**：单用户本地应用，材料是用户显式给的 `EvidenceItem`（name + content，`types.go:144-147`），所有席位看到同一份；没有分级的字段，也没有「哪些席位是本机」的概念。

**判断：挂池，不做。** 四级可见性解决的是多租户协作平台的越权问题，gaea 没有租户。硬要做只能做成「本机席位 vs 云席位」两档，而当前众神殿的四个适配器全是云端引擎（Grok/xAI、Codex、ZCode、DSH，见 `agents.go:113-182/215/311/513`）——**没有本机席位，分级就没有受众**，这与「不为不存在的 inbox 造 UI」是同一条纪律。

**复活条件**（写死，免下一轮重复讨论）：

1. 众神殿接入本机模型席位（Ollama / Herdsman / 本地 llama.cpp），且
2. 出现真实需求：「这段材料只能给本机模型看」，且
3. 用户接受「席位在缺料情况下作答」的代价。

届时最小设计：`EvidenceItem.Visibility: all|local`，席位声明 `trust: local|cloud`，`buildBrief` 按席过滤，`InputManifest` 记「谁收到了哪条」，**签注哈希把过滤事实算进去**（复用现有 Gaps 机制，不另造第二真相）。

---

### 5.6 技能三段式披露（Agorai skills）

**上游怎么做**：订阅时只给技能元数据（title/summary/instructions/tags），需要时 `get_skill` 取正文，再需要时 `get_skill_file` 取附件；官方称省 80-90% 上下文（`src/agent/context.ts:180-194`）。

**gaea 现状**：有 `internal/skill` 与工作区 `.gaea/skills/`，但**众神殿席位不挂技能**；席位的「协议知识」写在提示词与 `Preset` 里。

**判断：挂池。** 众神殿席位是只读一次性调用，没有「按需取技能」的工具面；要给就得给席位一套工具（读技能文件），这会把只读议事扩成有工具面的议事，安全面变大。当前 `Preset` + 提示词已覆盖「红队/定向质询」这类协议差异。

**复活条件**：席位工具面（只读）落地之后，且出现第三种以上议事协议需求。

---

### 5.7 反例三则（两源教我们「别做什么」）

**(a) 共识 = 关键词匹配。** Agorai 的 `isConsensusResponse` 把 `[NO_RESPONSE]` 与 "fully agree" 一并算作共识票，且只看单条消息文本（`patterns.ts:15-35`）。三个致命细节：① 短语表全是英文（`nothing to add` / `consensus reached` …），中文议事的输出一条也命中不了；② 比较的是**已答席位**而非应到席位——4 席里 2 席答了且都说"没啥补充"，就算全场一致（`ecclesia.ts:218`）；③ 把"没话说"翻译成"同意结论"。众神殿的三件套（自报 + 剩余分歧非空即阻断 + 翻转须举证）正是为了防这个（`manager.go:3055-3090`），**维持**。

**(b) 自适应超时无上下限。** Agorai 的 `calculateAdaptiveTimeout` 把复杂度、历史响应时间（60/40 混合）、轮次（首轮 ×1.5、后续 ×0.8）、席位数（>3 每席 +10%）连乘，注释明写 "Fully dynamic — no floor/ceiling"（`src/keryx/timing.ts:57-101`）。没有上下限的超时无法测试、无法解释、也无法向用户承诺「最坏等多久」。众神殿的 120s/5min/8min 三闸是可测的，**维持**。

**(c) 多数关轮。** Agorai 在过半席位已答时再等 10s 就关轮（`ecclesia.ts:395-403`）。沉默不等于同意，缺席不等于弃权——这正是本仓 `Gaps` 与「不能悄悄算成弃权」要防的事（`types.go:580-583`）。**不抄**。

---

## 6. 候选总表

| # | 候选 | 源 | 价值 | 档 | 判决 |
|---|---|---|---|---|---|
| A1 | 席位「无新增」语义（pass） | 双源 | 高 | P0 | 做 |
| A2 | 开场功能冒烟（canary） | aidebate | 高 | P0 | 做 |
| A3 | 议事健康指标（分布/重复/漂移） | Agorai | 中 | P1 | 做（只展示） |
| A4 | 主席小结按需触发 | aidebate | 中 | P2 | 观察后定 |
| A5 | 路径卫生（给模型的路径一律引号化/句柄化） | aidebate | 低 | P2 | 纸面纪律 |
| B1 | 材料分级投递 | Agorai | 中（条件未到） | 挂池 | 等本机席位 |
| B2 | 技能三段式披露 | Agorai | 中（条件未到） | 挂池 | 等席位工具面 |
| B3 | 交互式 CLI 席位协议 | aidebate | 低（形态不匹配） | 挂池 | 未来选项 |
| C1 | 共识关键词匹配 | Agorai | 负 | 不立项 | 反例 |
| C2 | 无界自适应超时 | Agorai | 负 | 不立项 | 反例 |
| C3 | 多数关轮 | Agorai | 负 | 不立项 | 反例 |
| C4 | 四级可见性 | Agorai | 低 | 不立项 | 无租户 |
| C5 | 每席持久记忆 | Agorai | 负 | 不立项 | 红线（episodic） |
| C6 | 任务认领/公告/能力发现 | Agorai | 低 | 不立项 | 无工作台场景 |
| C7 | 运行时切模式（`/mode`） | Agorai | 低 | 不立项 | 毁掉场次可比性 |
| C8 | 赛后 roast 环节 | aidebate | 零 | 不立项 | 娱乐，不是议事 |
| C9 | tmux 会话目录契约 | aidebate | 低 | 不立项 | 已有人类可读存档 |

---

## 7. 刀池

### P0-a：无新增语义（pass）

| 项 | 内容 |
|---|---|
| 改动面 | `discussion_act.go`（+1 动作 + 解析）、`manager.go`（一致性/收口判据）、`prompts.go`（合法出口与条件）、前端（「未新增」徽标）、`export.go`（统计） |
| Go 形状 | `DiscussionActPass = "pass"`；`Contribution.NoNewContent` 不新增字段，复用 `DiscussionAct`；`roundReachedConsensus` 增加 `passCount` 排除；新增 `passStreak` 判定（连续 2 轮全员 pass → `did_stop` 同型事件） |
| 不变量 | pass 不计一致票；pass 不记 gap；pass 必须有一行理由；主席可点名要求发言；旧存档零迁移 |
| 验收 | ① 一席 pass + 三席一致 → 不收口，事件说明「X 未表态」；② 两轮全员 pass → 提前汇总，事件可读；③ pass 不改变既有 10 场回放的一致结论；④ 前端徽标与导出计数对账 |
| 风险 | 模型滥用 pass 逃避 → 统计可见 + 主席点名；pass 理由为空 → 按普通发言处理，不吞正文 |

### P0-b：开场功能冒烟（canary）

| 项 | 内容 |
|---|---|
| 改动面 | `agents.go`（canary 调用形状）、`manager.go`（Start 前置）、`types.go`（`Participant.Canary`）、前端（席位卡「开局自检」）、`export.go` |
| Go 形状 | `Probe(ctx, status) CanaryResult{OK bool, Reason string, DurationMs int64}`；判据=零退出 + 正文非空 + 无已知错误特征；独立超时 90s；失败前置 gap |
| 不变量 | canary 不改 `StartRequest` 语义；不改轮次协议；失败席位仍可在用户手动重试后加入（与 `RetryAgent` 同型）；不新增持久化表 |
| 验收 | ① 假 CLI 三种失败（缺凭据/空输出/卡住）在开场前被识别并进 gaps；② 全席失败 → 拒绝开场（文案给修复路径）；③ canary 通过后第一轮不再重复探测；④ 关掉开关时行为与今日完全一致 |
| 风险 | 额外一次调用成本 → 默认开、可关；慢启动 CLI 误杀 → 独立超时不低于 90s；引擎会话浪费 → 支持续接的席位复用 canary 会话 |
| 附带条款 | 开场前就减员的场次，主席提示词必须写明「本场缺 X 席」，并加一句「只评判实际发言的席位，不得替缺席者虚构立场」——aidebate 在裁决提示词里就是这么做的（`debate.py:183-195`），本仓 `consensusShortfallWarning` 已有「缺席是运行状态」的宿主侧口径（`manager.go:3126-3128`），缺的是模型侧的同款纪律 |

### P1：议事健康指标

| 项 | 内容 |
|---|---|
| 改动面 | `internal/pantheon`（新纯函数文件）、前端记录页、`export.go` |
| 内容 | ① 每席字数/轮次占比；② 相邻轮同席 2-gram Jaccard（重复度）；③ 议题漂移（复用 `bm25` + CJK 分词） |
| 不变量 | 只展示、不驱动；无 LLM；空场/单席返回空态不造数；不确定就写「样本不足」 |
| 验收 | 纯函数单测（构造三场：正常收敛/原地打转/跑题）；记录页与导出一致；不改任何协议行为 |

### P2（纸面）：路径卫生

aidebate 把「产物路径不能有空格」提升为系统级不变量并用测试钉住，理由是 agent 会大量 emit 未加引号的 `touch` / `cd` / `printf >`（`src/aidebate/core/session.py:20-36`、`tests/test_session_paths.py:42-48`）。

gaea 的席位提示词也会把路径交给模型（会议记录文件、证据文件）。**低成本纪律**：宿主渲染任何路径时统一加反引号或引号，并在工具段加一句「路径含空格或中文时用引号」。不需要系统级改造；若将来出现「席位因路径失败」的实证，再升级为句柄化（给文件编号，不让模型碰路径）。

### P2（观察）：主席小结按需

aidebate 的主持人只在「回避问题 / 绕圈 / 关键假设无人质疑」三种情形介入（`crossexam.py:157-169`），其余时间不发言。众神殿每轮必小结，是一次额外调用。

**先观察再动**：统计 10 场里「小结是否产生了新分歧/新派工/新必答」。若多数小结只是复述，再考虑条件化触发（有新增 open issue 或有人被点名回应时才小结）。条件化会削弱收敛牵引，不能凭感觉改。

---

## 8. 不立项与红线（不重复立项）

| 项 | 为什么不做 |
|---|---|
| 每席持久记忆 / 共享记忆库 | 众神殿是 episodic 的：一场议事的记忆就是 `session.json`；给席位加持久记忆会造第二真相（与 OpenMausBot 第二轮 §9 同款红线，`MEMORY.md` 已判过） |
| 四级可见性 | 单用户本地，无租户；分级没有受众 |
| 任务认领 / 公告 / 能力发现 | 众神殿没有「席位自行领活」的路径；发言由宿主调度 |
| 运行时切模式 | 一场一协议是可比性与评测的前提；中途换模式等于换了一场会，`Protocol` 标签会失真 |
| 关键词共识 | 反例；会把沉默翻译成同意 |
| 无界自适应超时 | 不可测、不可解释、不可承诺 |
| 多数关轮 | 缺席 ≠ 弃权 |
| roast / 赛后吐槽 | 娱乐内容，不产生可签注资产 |
| tmux / pane 驱动的席位 | gaea 的席位是 headless CLI，读屏与按键协议没有对应物；若将来接交互式订阅 CLI，再按 aidebate 的「canary → 权限弹窗并行处置 → done 文件 + answer 存在性校验」三件套重做 |
| 技能市场 / 工具档案 | 无插件体系；`Preset` 已覆盖协议差异 |

---

## 9. 与既有契约的冲突检查

| 既有红线 | 本刀池是否触碰 | 说明 |
|---|---|---|
| 议事阶段不自动写盘、不运行副作用命令 | 否 | 三刀全在只读面；canary 只做最小调用 |
| 不造第二套完成状态 | 否 | canary 结果进 `Participant`，不新建状态机；pass 复用 `DiscussionAct` |
| 收口不等于消灭分歧 | 否 | pass 收口事件明写「无新增」，open issue 仍转 dissent |
| 不按 Agent 排名/自评决定话语权 | 否 | 健康指标只展示，不参与调度 |
| 执行须另签 | 否 | 无新增执行面 |
| 旧存档零迁移 | 是（需保证） | 新增字段全部 `omitempty`；旧 `session.json` 读回不崩 |

---

## 10. 源码索引

### Agorai（`clones/agorai@1e469be`）

| 主题 | 位置 |
|---|---|
| 架构总览（Bridge/Keryx/Debate 三层） | `clones/agorai` 内 `ARCHITECTURE.md:1-80` |
| Ecclesia 轮次生命周期 | `src/keryx/modes/ecclesia.ts:102-246` |
| 升级链（超时→催场→强关） | `src/keryx/modes/ecclesia.ts:374-420` |
| 一致性关键词判定 | `src/keryx/patterns.ts:15-35` |
| 循环/漂移/支配三探测器 | `src/keryx/patterns.ts:80-230` |
| 自适应超时（无上下限） | `src/keryx/timing.ts:20-101` |
| 席位上下文构建（含轮次截止） | `src/agent/context.ts:288-441` |
| 系统提示词（NO_RESPONSE / @mention / 禁止扮演） | `src/agent/context.ts:450-560` |
| 可见性存储层过滤 | `src/store/sqlite.ts:4-5/512-517/716-736/909-926` |
| 技能三段式 | `src/agent/context.ts:180-194` |
| 记忆数据模型 | `src/store/types.ts`（MemoryEntry / 三作用域） |
| 首轮盲写（轮次截止时间戳） | `src/agent/internal-agent.ts:320-325`、`context.ts:407-411` |
| whisper 收件人强制 | `src/bridge/server.ts:620-635` |
| 原子认领 / 失联回收 | `src/store/sqlite.ts:1547-1561/1448-1468` |

证据层：`docs/research/agorai-distill/01-source-analysis.md`（15 条候选、9 条不要抄、失实清单、调研边界）。

### aidebate（`clones/aidebate@80f3b77`）

| 主题 | 位置 |
|---|---|
| 五阶段编排与清单 | `src/aidebate/core/debate.py:1-25/282-397` |
| canary 握手 | `src/aidebate/core/turn.py:118-162` |
| turn：写 prompt → 发键 → 等 done | `src/aidebate/core/turn.py:65-115` |
| 权限弹窗并行处置 | `src/aidebate/core/turn.py:21-62` |
| 交叉质询驱动（队列/批量/静默） | `src/aidebate/core/crossexam.py:259-463` |
| chat-say 专用小工具 | `src/aidebate/core/crossexam.py:44-74` |
| 主持人介入三条件 | `src/aidebate/core/crossexam.py:145-170` |
| adapter 契约 | `src/aidebate/core/adapter.py:24-60` |
| tmux 会话与 pane 布局 | `src/aidebate/core/session.py:18-187` |

证据层：`docs/research/aidebate-distill/01-source-analysis.md`（15 条候选、8 条不要抄、F1–F12 失效模式、失实清单）。

### gaea 落点

`internal/pantheon/{discussion_act.go, manager.go, prompts.go, types.go, gates.go, agents.go, export.go}` + 前端 `PantheonRecord`。

---

## 11. 结论与下一步

**判定**：两源都不足以重开议事协议。Agorai 的产品面（记忆/技能/可见性/多租户）与本仓的本地单用户形态错位；aidebate 的执行面（tmux/交互式 CLI）与 headless 席位错位。真正能落地的，是两源各自那一条「席位侧诚实」的小机制，加上一条纯展示的体检指标。

**给用户的三个拍板项**：

1. **目标板块确认**：本轮按「众神殿」判断。若实际想蒸馏到别处（如办公 agent 协作、或模型中心的多模型互评），§5 需重做。
2. **P0 两刀是否开**：pass 与 canary 都会改变现有议事行为（前者影响收口判据，后者多一次调用）。建议先做 P0-a（pass），观察两场再决定 canary 的默认开关。
3. **canary 的主席失败策略**：拒绝开场 vs 自动换席——二选一，前者诚实，后者顺手。

**不做的承诺**：本文不宣称任何机制能提升「会议质量」。两源都没有可迁移的效果证据（Agorai 无评测；aidebate 是单人玩具），我们能保证的只是「更少的空转」与「更早的坏消息」。
