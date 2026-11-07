# OpenMausBot × 众神殿 蒸馏规划 v2（G0 证据冻结修订版，2026-11-07）

上游钉版：`milind-soni/OpenMausBot` commit `1bc4749aa8307d7d6998bf86de4ef6478460dd93`（2026-11-07T04:24:54Z），package.json `0.1.146`，Apache-2.0；本地克隆 `C:\AI\tmp\OpenMausBot`。长期维护只做版本钉住，不追上游；重评时只 diff 相关纯函数与契约文件+重跑 V1–V5 向量与历史回放，不重开仓库级蒸馏。

版本史：v1（本日午前）=纯调研报告；随后两场众神殿论证——主战 pan_1794048286641（4 轮，结构化判决：A2 拆 A2a/A2b/A2c、硬熔断否决、3+3 只能筛查）+ 续议 pan_1794051214760（3 轮，主席未结构化综合、签注 accepted，三席收敛「只交付+G0」）→ 本 v2 = G0 修订交付（论证 action-2）。

## 0. 规划总表（论证后）

| 编号 | 候选 | 证据类别 | 状态 | 影响/成本 |
|---|---|---|---|---|
| A1 | 主席带外控制信封 | **源码直证** group-goal-run.ts:39-85,112-143 | **已落地+G1 契约入档**（2026-11-07 晚，见 §3.1）；A2c 解锁 | 中/中 |
| A2a | 环形互动检测 | **源码直译** room-post-budget.ts:43-48 | **纯函数已落** DetectInteractionRing（seat_ring.go，稳健性池**不接线**——V1~V5 向量测试在册：三席环检出/星形免疫/二元与开链不判/未定位自指无边）；复活触发器不变（L35 原规则复现 ≥3 正例跨 ≥2 簇，或 B1 试点随行） | 低-中/低 |
| A2b | 停滞判据上线 | **本地再设计**（我方台账签名判据） | **关闭**。门槛未闭合：预冻结标签+非本场人工双标（κ≥0.6）+n≥9 窗跨会话分层+N_fire≥3 阳性密度门+G4a 行动数据 | 高/高 |
| A2c | 重复派单检测 | **源码直证** group-goal-run.ts:162-164,292-297（memberId:instruction 归一 key 3 次阻断） | **阻塞于 A1**（须信封 schema 稳定可解析） | 低/低 |
| ~~A2 原第三件~~ | 无进展超时 | — | **已淘汰=重复立项**。seatguard.go:14-48（2026-11-06）已有：120s 无输出看门狗（progress 续命与 RoomTurnDeadline 同构）+5min 开局静默闸+8min 总上限；138 次实弹无超 8min 案例。窄增量=「单一工具合法耗时数分钟时总上限不挂钟」→影响低零成本，挂池不立项 | 低/零 |
| B1 | Jev 式发言路由 | 源码直证 decider/room-routing.ts:83-104 等 | **红线 #2（不让模型自评 suitability 定默认发言权）约束，挂池**。若试点：自由圆桌 preset/user_note 定向投递形态；**ring 随行采集 would-fire（A2a 唯一复活触发器）** | 中/中 |
| B2 | 跨场立场 brief | **本地再设计**（思想源自 memory.md recent-work brief） | 默认=G1 纸面契约，不烧新会话；同题续议对仅 2 组→案例级，估不了复燃率；收益可离线盲评（同题新场有/无 brief 两臂，数旧分歧复燃，公分母=每场节约轮次）；载体契约独立立，不搭 A1 便车 | 中/中 |
| A3 | 决策透明行+stateHash 日志 | 源码直证 | 低影响低成本，不阻塞主实验时可试点 | 低/低 |
| A4 | 神格卡可移植 | 源码直证 shared/package-format.ts | 高安全面，等共享需求 | 高/高 |
| B4 | MCP 控制面三件套 | 源码直证 `upstream docs/{mcp-server}.md` | 等明确场景（衔接 V5） | 高/高 |
| B3/B5 | 定时圆桌/语音旁白 | 源码直证 | 需求拉动 | 中/中 |
| — | 命令面/事件面分离 | 源码直证 | **销账**：DSH 撤回「最缺」定级，保留低影响余量，不阻塞 | 低/— |
| A5 | **官方共享上下文**：续接席位增量投递 + codex resume 补齐 | **源码直证** host/room-sessions.ts（详见 §3.5） | 论证未覆盖（源起用户实弹观察）；**①②③ 全部落地**（2026-11-07 晚）：③ grok Seed 历史+seeded 提示词；① zcode/dsh/codex 讨论轮增量投递（delta 提示词）；② codex buildResume（摘 ephemeral+`-c sandbox_mode`+thread.started 采集，探针实锤续接） | 中/中 |

## 1. 默认路线（三席共识，签注 accepted）

**只交付 + G0（本文件即 G0 交付）**，不被任何 open issue 阻塞。可选加项仅二：
1. Codex 按 v1 报告 L35 原规则正式复现环形正例（双口径）——出 ≥3 正例且跨 ≥2 簇，A2a 回到中高影响、实验线可议；
2. B1 未来试点时 ring 随行采集 would-fire——数据免费，是 A2a 的具体复活触发器。

若用户批准进实验线，门槛流水线见 §4；A2b 生产开关全程关闭，A2c 不启动；独立标注者须先落实（非本场人工，双标+κ≥0.6）。

## 2. 已淘汰清单（论证结论，不再立项）

1. **A2 硬熔断**（「目标贡献集与证据句柄集均无新增」判据）：DiscussionAct/TargetContributionID/EvidenceHandle 均可选字段，覆盖率 61.8%/57.8%/10.8%，只够诊断不能断停滞（types.go:358-365、discussion_act.go:31-64；DSH r3 撤回，全场接受）。
2. **「同论点指纹 3 次熔断」作源码直蒸机制**：本地再设计误归属——group-goal-run.ts:162-164 实为 memberId:instruction 重复派单防护（v1 报告 L36，本版已改标）。
3. **3+3 零误报直接上线**：单侧 95% 上界≈63.2%，只能做淘汰筛查（全场对 Clopper-Pearson 复算一致）。
4. **no-progress 守卫重复立项**：seatguard.go 三道护栏已在（2026-11-06，早于 v1 成稿——L37「现无 per-call 守卫」在成稿时点即错）。
5. **逐轮 Issue 快照支持停滞上线判据**：快照不解决「停滞 vs 深化」二义性（types.go:506-534）。
6. **G2a「覆盖≥2 类守卫」通过条件**：Contribution schema 无 instruction 类字段（ZCode r3 亲读 types.go:336-376 后正式撤回），结构不可满足——改为「环形直证设门（≥3 命中且跨≥2 簇）+重复派单/无进展弱语义单列不设门」。

## 3. 证据分类（G0 三列制的候选级落点）

- **源码直证**（原仓可平移的机制契约）：A1 信封、A2a 环检测、A2c 重复派单、A3、A4、B1 路由、B3、B4、B5——锚点见 §6 与 v1 各节。
- **本地再设计**（源码无此物，须自证）：A2b 台账签名停滞判据；B2 brief 载体（注入格式/行数预算/失效条件契约尚不存在）；v1 报告 L35-36 两处（已改标）。
- **已淘汰**：§2 六条。

### 3.1 A1 主席控制信封 G1 契约（已落地，2026-11-07 晚）

**输入**：主席轮间小结的模型输出（moderationPrompt 已补收 open 台账——论证实证的注入不对称点销项——与席位 ID 名册；信封协议段随提示词下发）。**输出**：`Contribution.ChairAssignments []ChairAssignment{agentId,instruction}` 随 moderation 贡献持久化；`Content` 为剥除信封后的可见文本。**不变量**：①只认最后一个完整信封（防引用示例劫持）；②残片/损坏=malformed 降级为普通小结+warning 留痕，不吞发言（不抄 #9）；③派工只认参会席位（名册外丢弃+留痕）；④≤8 条/≤200 字（收口手段不是新议程）；⑤可见文本永不残留信封原文。**失败降级**：解析失败/无信封=普通小结照常入档，议事不受阻。**隐私边界**：信封是主席→宿主→席位的定向指令，用户经 Warnings 与导出可见派工内容（透明性优先于隐蔽性——与上游 goal envelope 的「人类不可见」取舍不同，我方记录即证据纪律）。**回滚**：ChairAssignments 为可选字段，旧存档零值兼容；提示词协议段去掉即回退单发小结。**消费**：runSeat 按 `chairInstructionFor`（查上一轮 moderation 存档，无运行态台账）注入【主席定向指令】段，优先级声明=高于一般议程、冲突先说明。**A2c 随行**：`duplicatedAssignmentWarning` 同（席位,归一化指令）≥3 次告警（拒绝不记账原则：只告警不阻断）。



**落地记录（A5③ grok 席持久记忆，用户拍板「根据规划开始吧」）**：pantheon seam 新增 `SeatMessage`+`GrokHistoryBackend`（agents.go），`buildSeatHistory` 从贡献存档**确定性重建**该席视角的往轮会议记录（seat_history.go：按存档序严格回放「何时看到什么」，本席缺席轮次内容顺延累积、不丢记忆；`beforeRound` 轮界=同轮可见性归提示词共享段，独立开场「互不可见」纪律不被历史泄漏；synthesis 的 round=0 恰为无界）；`runModelSeat` 历史分支优先走会话后端，历史为空自动退回单发路径；prompts.go 新增 `discussionPromptMode`/`synthesisPromptMode` 的 seeded 形态（往轮梯度整段省掉，只发当轮新增+指针行——双份记忆根除）；app 侧 `pantheonGrokSeat.StreamGrokSession` 把历史经 `session.Seed` 注入办公 AgentRunner 消息日志（system 恒首位）。**不维护可变台账**：跳过/重试/恢复/重启后按同一函数重建，永远与存档一致。

**落地记录（A5①② 增量投递+codex 续接，同日续刀）**：②codex 探针三实锤——`codex exec resume <SESSION_ID>` 续接真通（新会话原样复述上一会话回复）、`--ephemeral` 会话不落 rollout（resume 报 no rollout found，**必须摘掉**）、resume 子命令拒收 `-s`（沙箱走 `-c sandbox_mode=…`）；`codexExecSpec` 统一 build/buildResume（resumeID 空=回落新建，守接口契约），thread id 经 `{"type":"thread.started","thread_id":…}` 采集进 `parseSeatRunStats`（新 case）；代价=codex 议事会话落 ~/.codex/sessions（与 zcode/dsh 同款取舍）。①三态 `seatHistoryMode`（gradient/seeded/delta）：delta 形态=提示词只带「自你上次发言以来的新增」**全量块**（`deltaContributionItems`+`seatRecordPart` 与 ③ 共用同一呈现形状，两路记忆不出现两套真相）；接线 `seatDeltaCapable`（buildResume+活跃 EngineSessionID 才判 capable，会话丢失自动退回梯度）；【用户插话】指引段改按模式可见集触发（防往轮插话复读病复发）。汇总对 CLI 席保持梯度形态（其引擎会话只见梯度视图，全量记录须在场）。**效果**：四席 round≥2 讨论轮全部告别梯度重发——grok 全文在场（宿主 Seed）、zcode/dsh/codex 全文在场（引擎侧续接+增量块），每轮往轮信息只发一次。验证：pantheon 13.8s+app 168.3s 全绿+vet 清；codex spec 断言改用假可执行文件不再静默 SKIP；新增用例含 DSH 两轮 delta 端到端（讨论=增量/汇总=梯度双断言）。

**落地记录（A1 信封+A2a 环纯函数，同日三批）**：A1 全链=types.go `ChairAssignments` 字段+chair_envelope.go（parseChairEnvelope 取最后完整信封/剥残片/malformed 降级留痕、filterChairAssignments 参会席位过滤、chairInstructionFor 存档查表、duplicatedAssignmentWarning A2c 三次告警）+moderationPrompt 补收 open 台账与席位 ID 名册+runSeat 注入【主席定向指令】段；manager 级集成测试（round1 信封→剥除入档→round2 提示词收到指令→ghost 过滤留痕）。A2a=seat_ring.go DetectInteractionRing 纯函数（不接线；V1~V5 向量：三席环/星形免疫/二元开链不判/未定位自指无边）。验证：pantheon 16.1s+app 170.4s 全绿+vet/fmt 清；新增 10 用例。至此规划池可动项全清：A1/A2a/A2c（随 A1）/A5①②③ 落地，B2 契约纸面（§0 表），B1/B3/B4/B5/A4 需求拉动挂池。

**对方机制（源码直证）**：harness 对每线程每引擎保持一个热会话，每轮 resume 同一会话；提示词只发增量块 `[自你上次发言以来新增的消息…]`（renderRoomDelta，room-sessions.ts:103-118）。核心不变式（该文件第 1 行注释）：**「每条 window line 要么在本次发送里，要么曾以完全相同的形式给过这个 session」**——从不悄悄压缩已发过的内容；逐行 hash 判定（unseenLines:225-258、lineHash:120）；**任何怀疑回退全量**（会话换/配置 hash 变/近 compact 0.8/超时/异常，decide():260-319，fail-closed）。白名单 `ROOM_SESSION_ENGINES`（:12）当前仅 Claude Code——其余引擎每轮全量 30 条窗口，无压缩梯度。配套记账：RoomContextTracker（room-context.ts:32-151）token 读数 ≥0.9 才 takeReset 让该席重开会话（全系统唯一合法失忆点，只影响该席）。

**我方现状实证（pan_1794051214760 participants）**：
- zcode `sess_8b841f66…`：`--resume` 已接线（agents.go:280-304）→ 引擎侧有完整历史，**但每轮仍重发梯度历史（4800/800/索引）=双份记忆**，token 白花且引入失真；
- dsh `session-116aba8b…`：`--session-id` 续接（agents.go:498-505）→ 同病；
- codex `engineSessionId=""`：**无 buildResume**（agents.go 全包仅 zcode/dsh 两处）→ 每轮全新引擎会话，内部零记忆，失忆最重席；
- grok（模型席）：**用户指正实锤——办公栈会话能力现成，只差接线**。pantheon_grok_agent.go:154 每次调用 `gaeaAgent.NewSession(...)` 全新会话是接线选择，非能力缺失：`session.Session` 本就支持 `Seed(msgs)`（v4.380 fork 型子代理）/`Snapshot()`/`Save/Load`（事件日志格式+检查点游标，办公对话跨重启恢复同款，session/session.go:35-56,111）——且 ai.Client 另有多轮 messages API（ChatStreamMessages client_stream.go:624）。

**我方落点（三刀）**：
1. **zcode/dsh 改增量投递**：每轮 prompt 改「本轮新增贡献+必答指令」，梯度历史整个省掉；hash 校验沿用对方不变式，任何怀疑回退全量（fail-closed）。「全文见文件」索引行的价值随之重估——引擎侧已有全文时不再需要；
2. **codex buildResume 调研**（小刀前置）：`codex exec` 是否支持会话续接；能则补 buildResume 进增量档，不能则 codex 至少吃「自己往轮全文」的保真修正；
3. **grok 模型席（成本最低，率先可做）**：GrokBackend seam 扩消息形态入口（pantheon 包自定义最小 Message 结构，维持不 import app/ai 的边界）；app 侧 pantheonGrokSeat 按 (sessionID, seat) 持消息台账——每轮 `Seed(历史)` + Run 后 `Snapshot()` 存回。grok 即得全保真跨轮记忆（不再吃 800 字卡/索引梯度），且天然满足 A5 增量不变式（append-only、前缀稳定=缓存友好）；长场超窗交给办公栈自己的 compact 机制（ContextWindow 200_000）。

**契约要点（过堂时须答）**：增量判定与回退条件（同席重试/引擎会话丢失/resume 静默失败→全量）；与 maxPromptRunes=60000、clampPrompt、转述纪律（贡献 ID+文件路径）的兼容重述；Session 维度的新增记账字段（每席上次已发行游标）。

### 3.2 B2 跨场立场 brief 纸面契约（G1，2026-11-07 晚补）

**定位**：泛化既有 parentStanceEvidence（续议链：ParentSessionID 存在时父会话最近 ≤8 条转立场卡注入 Evidence，2026-10-31 已落地）——B2 覆盖**非续议**新场的「同席跨场」记忆。两者边界：有父子链走 parentStanceEvidence（现状不动）；无父子链时可选注入跨场 brief。**载体**：StartRequest 简报组装链新增【跨场立场】段，只读派生自 sessions 存档（该席各场的 consensus 行+Residual 摘要+议题+日期），**不建第二真相**（存档是唯一事实源，brief 是每次开席现算的投影，不落盘）。**预算**：≤10 行×每行 ≤120 字（总 ≤1.5KB），超窗取最近 N 场；行文只列「最终立场+残留分歧」，不复述论证。**失效条件**：存档读取失败/无历史场→整段静默省略（记忆是增益不是依赖）；本席在某场缺席（failed/skipped）→该场不计入其 brief。**隐私**：全部本机存档派生，不出境。**收益评测**（论证冻结）：同题新场有/无 brief 盲评两臂，数「旧分歧复燃/重复论证」次数（TargetContributionID 链半自动计数+人工确认），公分母=每场节约轮次；池内同题续议对仅 2 组→先案例级，不成统计结论。**实现成本估**：中低（纯函数扫描存档+prompt 组装一处；无新存储、无新绑定）。**不立项边界**：不做夜间整理/划线/归档树（memory.md 全家族超需求），只做这一段。

### 3.3 未闭合项销项记录（2026-11-07 晚）

- **git log 形式确认（ZCode r3 留）**：`2530ca23`（2026-11-07 10:10 +0800，commit message=席位卡死根修四刀）是 seatguard.go 唯一提交；文件头注释自证护栏写于 11-06（引 11-06 实弹会话）；v1 报告成稿于同日晚——「L37 在成稿时点即错」判定成立。
- **上游钉版复核**：`1bc4749aa8307d7d6998bf86de4ef6478460dd93`（2026-11-07T04:24:54Z）与 §0 记载一致。
- **Codex L35 原规则双口径复现**：仍待出数（A2a 实验线唯一复活门）。
- **A2/B2 产品序**：归用户（默认路径下暂不需答）。

## 4. 门槛流水线（仅当用户批准进实验线；证据门控非日历驱动）

| 阶段 | 最小交付 | 通过/停止条件 |
|---|---|---|
| P0 用户门 | 只交付报告 / 进 shadow 线二选一；指定标注责任人 | 未明确选择则停在 G0，不进生产代码 |
| G0 证据冻结 | 本文件（三列制+固定 SHA+许可证+状态分列） | 每事实有 file:line 或标「本地再设计」 |
| G1 适配契约 | A1/A2a/B2 各自独立契约（输入/输出/不变量/失败降级/隐私/回滚）；冻结事件名与 V1–V5 测试向量 | 无失败出口不上线；静默期指标只要求 would_fire 覆盖/密度/簇分布，行动指标预注册在 G4a |
| G2a 机械回放 | 37 个轮末边界窗跑发送集，25 个多席讨论窗作标注池；按环形直证/目标链再设计/重复派单弱语义分列输出命中数与簇分布；不可标注命中单列 | 环形直证 N_fire<3、或仅落一个会话簇、或正式复现为 0 → 挂起经验门；不得无预算扩窗 |
| G2b 独立双标 | 命中窗+等量分层对照窗；非本场人工双标；κ 与簇校正区间预注册 | 无标注者/样本不足/一致性不达标 → 挂起，不得记检测器失败 |
| G3 静默 shadow | 只记 would_fire；A2b 开关保持关闭 | 静默数据不足或事件覆盖失败 → 停 |
| G4a 可见 shadow | 非阻断、限频、可关闭、可回滚小流量告警；采集 warning_shown/user_action | 行动链不通/误报超预注册上界/用户嫌扰 → 退 |
| G4b 正式开启 | 仅保留有行动收益且可回滚者 | 未达 G4a 预注册指标 → 永久停在 G4a 或撤回 |

**显式出口（合法终止，非失败）**：行动密度为零且 G4a 采不到行动 → A2 价值上限=纯诊断信号，G4b 永不开启。**坍塌规则**：正式源码规则回放无足够命中 → 挂起经验实验、保留纯函数回归，不因历史零命中把未来多客户端/延迟投递/自由圆桌形态下的风险永久判零。

## 5. 已冻结统计纪律

零误报 n 窗单侧 95% 上界 = 1−0.05^(1/n)：n=3→63.2%、n=6→39.3%、n=9→28.3%、n=14→19.3%、n=29→9.8%。同会话聚簇按簇校正：28 窗名义 10.1%，10 簇校正后 25.9%（诚实区间 10%–26%）。字段缺失一律记 insufficient，不并入「无进展/停滞」。预登记：≤30% 需 n≥9、≤20% 需 n≥14，窗口跨会话分层。

**实证底数（DSH r1 预检，续议承用）**：13 会话解析 → 3 场 OpenMausBot 同题污染出池 → 剩 10 会话/162 贡献/47 轮边界/28 讨论窗（25 窗 ≥3 席）；行动代理结构性缺失（user_note=0、gate 不在事件流）；同题续议对仅 2 组。

## 6. 修正记录（v1 → v2）

1. L35 环检测我方落点：改标「本地再设计，代理回放 0 正例，正式复现待 Codex」。
2. L36「同神同论点指纹 3 次熔断」：改标本地再设计（源码为 memberId:instruction 重复派单防护）。
3. L37「无 per-call 守卫」：删除——seatguard.go:14-48（2026-11-06）已有三道护栏；窄增量（总上限不挂钟）挂池。
4. 拍板池重排：A2 从首项降稳健性池；具体产品序（A2 vs B2）归用户，且按默认路径只在「复现出正例」后才需要回答。
5. 补执行点实证：manager.go:1353-1365（整轮收齐后、小结前）；moderationPrompt 未收 Issues=注入不对称点（A1 契约的现成落点）。

## 7. 台账处置（两场 9+7 条，按 ZCode r3 建议）

销账：主战 issue-1~9 全 resolved（含硬熔断否决/3+3 降级/A1 只阻塞 A2c/事件面撤回）；续议 issue-2/3（实质合流、schema 亲证撤回）。休眠：续议 issue-4（分母口径之争，有正例后再议）。保留：续议 issue-5/6（翻转条件已具体化=L35 复现 ≥3 正例跨 ≥2 簇，或 B1 试点）。归用户：续议 issue-1（A2/B2 产品序+标注者，默认路径下暂不需答）。

## 8. 源码索引

- 房间编排：server/index.ts:10820-11020、server/room-{member-lock,queue,turn-end,turn-timeout,post-budget,holds,same-name-mentions}.ts、store.ts:209-271
- 决策路由：server/decider/{room-routing,jev,types,index}.ts、`upstream docs/{decision-model}.md`
- goal run：server/group-goal-run.ts、shared/group-goal-run.ts
- 上下文/会话：server/host/{room-transcript,room-sessions,room-routing-input}.ts、server/room-context.ts
- harness：server/harness/{registry,bus}.ts
- 文档族：docs/{decision-model,team-sharing,presets,memory,mcp-server,voice-mode,approval-levels,proactive-routines,routine-schedules,sentinel,bot-box,tool-selection}.md
- 我方对照：internal/pantheon/{types,manager,runner,prompts,preset,eval,seatguard}.go、docs/gaea-pantheon-v2-roadmap-2026-11-03.md（「明确不抄」红线）

## 9. 同构等价（不立项，v1 结论维持）

串行轮次、主席小结、分歧台账、EngineSessionID 续接、事件时间轴、user_note 注入、InputDigest、无进展看门狗（seatguard）——均有等价物或更简形态。
