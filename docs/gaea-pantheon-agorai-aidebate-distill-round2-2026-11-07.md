# Agorai × aidebate 蒸馏调研·第二轮增量（2026-11-07）

> **与前序档的关系**：`docs/gaea-pantheon-agorai-aidebate-distill-2026-11-07.md`（并行执行体产出，未提交）已完成双源主判（A1 pass / A2 canary / A3 健康指标 / B 挂池 / C 不立项 + 拍板 3 项）。**本文不重复其结论，只录独立深读的增量事实、三处修正/分歧、新增候选 M/B 系**。两档并存，拍板时合并读。
> **本轮克隆**：`C:\AI\_distill\agorai`、`C:\AI\_distill\aidebate`（depth 1，只读）。钉版与前序档一致（Agorai@1e469be / aidebate@80f3b77）。
> **纪律**：AGPLv3 只取道不取器；全部结论 `path:line` 实锚；gaea 侧断言已逐一 grep 验证（见各条「已核」）。

---

## 1. 增量事实层（前序档未覆盖）

### Agorai（按「已验证行为」读，本轮补的运行路径实锤）

| # | 事实 | 锚 |
|---|---|---|
| AG-1 | `roundHistory` 只初始化一次从不重置：「每议题最多 3 轮」实为**每会话终身 3 轮**，用满后新人类消息被静默吞掉；续轮议题钉死在第一轮议题 | `src/keryx/modes/ecclesia.ts:105-110,306`、`module.ts:584` |
| AG-2 | 内置 agent 把含 `[NO_RESPONSE]` 的输出**直接吞掉不发**→ pass 的席=没作答的席，「全员 pass」永远凑不出，讨论只能跑满轮数；单测靠直接注入 store 绕过 | `src/agent/internal-agent.ts:403-408`、`__tests__/keryx.test.ts:955-987` |
| AG-3 | 子串判共识对否定无感：「I don't fully agree」也算共识票 | `src/keryx/patterns.ts:13-36` |
| AG-4 | 委托综合「收件不认人」：进入 synthesizing 后**任意下一条消息**（含人类的）都被当成综合吞掉 | `src/keryx/modes/ecclesia.ts:48-51,350-368` |
| AG-5 | `/pause` 只翻标志位，**升级链定时器与自动开轮不检查它**——暂停期间轮次照样超时关闭、续开；`/extend` 清了定时器没重设（TODO），该轮从此无超时 | `ecclesia.ts:300-304,390-419`、`module.ts:273-311` |
| AG-6 | 带内 @命令权限检查形同虚设：判断发送者 ID 是否 `internal:` 前缀，实际 ID 是 UUID→**AI 输出文本里出现命令也会生效** | `module.ts:224-225`、`store/sqlite.ts:602` |
| AG-7 | Socratic 模式下内置席完全不可用：点名话术与内置席的 Keryx 闸不匹配，永不即时应答，迟到回答又被丢弃；终止条件只有「全员 pass」无上界 | `agent/internal-agent.ts:276-289`、`modes/socratic.ts:60,120-217` |
| AG-8 | 自适应超时两个关键因子（历史画像/轮次系数）在生产路径读空值从不生效，测试靠手工塞值才测到 | `keryx/timing.ts:20-101`、`ecclesia.ts:124/139,317/332` |

### aidebate（钉版 v0.4.7，本轮补的裁决面与提示词逐字）

| # | 事实 | 锚 |
|---|---|---|
| AD-1 | **裁决输入缺口**：辩手块只遍历 `active_sides`→驳论阶段才掉线的席，其已提交的开场**不进裁决**（却进 roast）。「只评已提交者」被实现成「只评撑到最后者」。gaea 汇总吃全量 history（`manager.go:1506`）已规避 | `core/debate.py:176-182,231-233,503` |
| AD-2 | **主持人不参与辩论**：`moderator` 是保留名，单独窗格，只裁决。gaea 主席从参会席选出且照常发言=运动员兼裁判，只有红队预设才全员独立确认（`manager.go:1561-1588`） | `cli.py:147-149`、`web/server.py:469-470`、`debate.py:326` |
| AD-3 | 防和稀泥六手段中 **gaea 缺 ①③⑤**：①逐个对手都要覆盖（gaea 只要求每轮点一个前文论证，`prompts.go:86`）；③缺陷分四类（事实/逻辑/证据/未回应反例，gaea 台账 Origin 无分型）；⑤开场须答「预期最强反驳」（gaea 五问只问不确定处，`prompts.go:354-359`） | `core/debate.py:123-125,152-162` |
| AD-4 | verdict prompt 逐字：Winner 段+每席 Scorecard（最强贡献/最大弱点各一句）+Open questions 2-3+「**Avoid mushy "both sides have merit"** unless genuine tie」 | `core/debate.py:205-215` |
| AD-5 | hostile-eyes 对抗自审：六步清单+diff>10 行须 HEAD 标记新鲜才能建 PR；两缺陷=双标记文件不同步、标记证明「调用过」不证明「已处置」 | `.claude/commands/hostile-eyes.md:8-35`、`scripts/hostile-eyes:34-61` |
| AD-6 | 时间口径混用：清单 `created_at`/`dropouts[].at` 是本地时间，事件流与群聊是 UTC Z——同一存档两套时钟 | `core/debate.py:289,314` vs `core/events.py:11-12` |

---

## 2. 与前序档的三处修正/分歧

1. **轮次形态「无增量」→ 有条件改判（M2，需拍板）**。前序 §4 判「串行 vs 并行=形态不同，无增量」；本轮从**从众**与**耗时**两维修正：gaea 第 2 轮起串行接力是从众翻转的根源（`manager.go:1231-1249` 在治标），而 aidebate 驳论并行+互盲（只看第 1 轮全文、看不到同轮他席，`debate.py:480-486`）让同轮从众的来源直接消失，总耗时≈2×最慢席+1 次汇总（对照实弹 4 席 4 轮 83 分钟）。落点已备好：预留常量 `PhaseChallenge`（`types.go:26`）、同轮可见性边界 `beforeRound`（`seat_history.go:23-26`）。代价=失去轮内接力吸收。
2. **pass 信号「必须入档不吞」有了上游反例背书**。前序 A1（pass 动作）的三条不变量之外，AG-2/AG-4 补充两条设计纪律：**宿主绝不吞弃权信号**（agorai 吞了导致全员 pass 永远凑不出）、**综合/收口必须绑定具体那次调用**（gaea CallRef 已对，agorai「收任意下一条」是反例）。
3. **过半+宽限关轮：维持前序 C3 不抄**。本轮调研员曾提「并行首轮过半+10s 宽限提速」作可选项；与前序 C3「缺席≠弃权」红线冲突，且 gaea 缺席已有独立归因（`manager.go:3126-3128`）。**判决：不立项**，记录冲突防止下一轮重提。

---

## 3. 新增候选总表（编号续接前序 A/B/C 池，M 系=本 rounds 新立）

| # | 候选 | 源 | 档 | 判决 | 已核 |
|---|---|---|---|---|---|
| M1 | **席位调用 prompt 快照**：CallRef 加 `promptSha256`/`promptRunes`，可选落 `sessions/<id>/prompts/<callId>.md`。前序档没覆盖的「导出缺什么」答案=每席实际看到了什么 | aidebate 每回合 prompt 入档 `turn.py:73,80` | P0 级 | **做** | ✅ CallRef 无此字段（`types.go:68-76`），仓内无 promptSha/prompt 落盘 |
| M2 | **立场→同轮互盲驳论→汇总**三段制式，第三种 Protocol，进评测对照组 | `debate.py:480-486` | P1 | **拍板**（见 §2-①） | ✅ PhaseChallenge 预留/beforeRound 边界在 |
| M3 | **对手覆盖矩阵**：纯函数复用环检测边定义（challenge/respond+TargetContributionID），输出「谁回应了谁/哪席主张没人接」；只展示+注入主席小结，不驱动停机 | `debate.py:152-162` | P1 | 做 | ✅ 边定义 `seat_ring.go:3-16` 可复用 |
| M4 | **质疑缺陷分型** `Issue.FlawKind`（事实/逻辑/证据/反例四类→映射 IssueReview 默认去向：needs_verification/辩论闭合/user_decision），解析失败不吞正文 | `debate.py:154-157` | P2 | 做 | ✅ 现有 Origin 仅 challenge/residual/user |
| M5 | 开场五问加一问「预期最强反驳+你的回应」，可提前在台账开条目 | `debate.py:123-125` | P2 低成本 | 做 | ✅ 现五问 `prompts.go:354-359` 无此问 |
| M6 | 汇总提示词注入 Gaps 清单+「只评实际发言席、不得替缺席者编立场」——前序 P0-b 附带条款，**经核对 `prompts.go:674-771` 至今未落地**，立为独立小刀 | `debate.py:183-195` | P0-b 随刀 | 做 | ✅ 本轮复核未落地 |
| M7 | **席外主席**选项（主席不参与轮次发言只做小结汇总），消除运动员兼裁判；代价=多占一引擎席位 | AD-2 | 观察 | 拍板 | ✅ `manager.go:3259-3273` 现状=席内主席 |
| B2 | **定向插话**（非密语）：复用主席按席派工通道全文注入目标席提示词，仍入档导出可见；借「@ 目标必须在集合中」校验 | `bridge/server.ts:620-635` | P2 | 观察 | ✅ `AddUserNote` 现只能广播（`gates.go:406-465`） |
| B3 | **导出前敏感扫描闸**：导出/定稿时对可配正则扫描，命中让用户确认；不做存储级可见性 | agorai Blackboard 提升公开前扫描 `memory/base.ts:131-145` | P2 | 做 | ✅ Export/ResultPath 无任何扫描（`manager.go:885-921`、`export.go:9,220`） |
| B4 | **整场预算阶梯**：按消耗分档——≥80% 降历史档→≥90% 减席→≥95% 本轮即末轮→100% 停止写「预算收口」；不抄「砍到一席」 | `orchestrator.ts:272-308` | P1 | **做（直接治「一场会太贵」）** | ✅ 每调用 Usage 已记账（`manager.go:1273,1433,1528`），无整场预算线；现有仅单席工具预算+prompt 截断 |
| B6 | 决策门加 amend（修订议题续跑，台账保留旧议题） | `ecclesia.ts:464-496` | P2 低优先 | 观察 | ✅ 现只 resolve/dismiss（`gates.go:516-526`） |
| T1 | 自测门 hostile-eyes 思想：清单+HEAD 新鲜度，但标记绑「发现清单哈希」不绑调用动作；众神殿专属检查项（新字段 omitempty/旧 session.json 读回/新 Phase 进 phaseLabel+导出/信封解析降级留痕） | AD-5 | 工具线 | 观察 | 现有 `scripts/ci.ps1` 无此门 |

**C 档维持不立项**（本轮再次确认）：Winner/Scorecard 打分（与「不打分排名」红线冲突，`prompts.go:77,576,678,741`）、给席指派立场（挂池，复活条件=用户点名要唱反调席）、独立 events.jsonl（第二真相）、YAML 适配器外置、roast、tmux、词法类检测器（agorai 三检测器全部因误报降级为日志=佐证 gaea 结构化路线）、复杂度自适应超时（AG-8 两因子生产路径失效）。

---

## 4. 拍板项合并清单（前序 3 项 + 本轮新增）

前序档已列：①目标板块确认（本轮按众神殿执行，同答）②P0 两刀（pass/canary）是否开 ③canary 主席失败策略（拒绝开场 vs 自动换席）。

本轮新增：

4. **M2 互盲三段制式**是否立项为第三种 Protocol 并进评测对照组？（前序档判「无增量」、本轮判「从众+耗时两维有增量」，两档结论冲突需裁）
5. **B4 整场预算阶梯**是否开？（直接对实弹 83 分钟痛点；顺序=先降历史档后减席最后收口）
6. **M1 prompt 快照**形态：只记哈希+字数（零迁移 omitempty），还是连原文落 `prompts/<callId>.md`（单 prompt 上限 6 万 rune，体积需权衡）？
7. **M7 席外主席**是否值得为主席中立多占一个引擎席位？（或非红队也加轻量独立确认）

---

## 5. 本轮调研边界

- 两仓均 `--depth 1` 静态深读，未安装依赖未运行；「推断」处已在对话稿标注。
- Agorai checkout 自身不自洽（缺 `getPublicMessages` 等三方法、wild-agora 模式缺失），沿前序「按设计意图读」口径；本轮新增运行路径结论均以**实际存在的代码路径**为准。
- 本档不写生产代码；落地动刀须用户点名（[[zcode-continue-means-research-not-land]]）。

---

## 6. 两本账主轴与拍板落地（2026-11-07 深夜续，用户点题+拍板）

**用户点题**：众神殿核心问题 = 各 agent 多轮保留自己上下文（含工具结果）+ 共享每轮结论 → 两本账模型（席位私有账/场次共享账/对账线）。代码实态：共享账闭环（A5 四形态）；CLI 三席私有账=引擎侧续接健康；**grok 私有账裸奔**（`GrokHistoryBackend` 被 pan_1794058438005 实弹证伪撤回——Seed 历史后 12 步全工具循环零陈词×3，退回宿主全量窗口内联 `pantheon_grok.go:28-31`，seam 留口）；对账线半开（E1 只答计数）。

**用户拍板**：轻刀两件开、重刀挂池、grok 维持撤回现状。

### 6.1 轻刀落地记录（未提交，等用户验收后随批次提交）

**轻刀② E1 升格（非空锚+对账线）**：
- `types.go`：`ToolEvidence.NonEmpty`（跨全部工具非空成功返回总数，不受 Tools 6 项截断）；`ToolCount` 加 `OK/NonEmpty/Digest`（该工具首个非空成功输出的 sha256 前 16 hex，锚定宿主看到的有界预览，不落原文）。
- `evidence.go`：`evidenceFromEvents` 按 `toolStat` 累计；`toolOutputDigest`；**对账纯函数 `evidenceMismatchNote`**（证据非 nil 且 NonEmpty=0 且陈词含【工具核验】→ 一行事实陈述警示；nil/有非空/无标注三态均返空）；`toolEvidenceLine` 追加「N 次有非空返回」（零值保持沉默）。
- `export.go`：贡献导出在宿主观察行后挂 `> ⚠ 宿主对账…` 行。边界：汇总席（PhaseSynthesis）证据同样落账，但定稿贡献循环跳过该相位——对账行只覆盖讨论/主持陈词，汇总自有 Verdict 证据链。
- 前端同批：`pantheon.ts` 类型 + `PantheonRecord.tsx` 标签加「N 次有非空返回」；`wailsjs/go/models.ts` 按生成器格式手同步（下次 wails build 再生应零差异）。

**轻刀① 工具发现要点契约**：`prompts.go` 转述纪律加一行——【工具核验】条目必须就地带「出处 + 要点」（出处=URL/文件路径，要点=一句实际读到的内容），只给结论不给出处的按【未核验】处理。对账警示的措辞**不进提示词**（守住 E1「证据不进提示词」边界，避免刷工具博弈）。

**测试**（tool_evidence_test.go +6 用例全绿）：非空计数语义（空白/失败/未回报三态都不算）+ digest 确定性与 16-hex 不含原文 + NonEmpty 不受 6 项截断 + 对账纯函数五态 + 导出端到端（空返回+【工具核验】→ 定稿带警示）+ 端到端非空锚落贡献。验证：pantheon 16.6s 全绿/全树编译/vet/gofmt 清/tsc 0/vitest 众神殿 65/65/app 众神殿 7/7 + 全量 app 套件后台过。

### 6.2 挂池与维持

- **重刀挂池**：宿主侧 per-seat 工具账本（`sessions/<id>/seats/<agent>/toollog`，截断+脱敏原文）——复活条件：resume 丢失自救或跨席对证出现真实需求；届时先解决凭证脱敏与第二真相边界。
- **grok 私有账维持撤回现状**：seam（`GrokHistoryBackend`）留口，复活条件=实测健康的引擎 opt-in（全量窗口内联为现行行为基线）。
- **正交刀待拍板池**（与两本账相关）：B4 整场预算阶梯、M2 互盲三段制式（M1 已落，见 §6.3）。

### 6.3 M1 prompt 快照落地（2026-11-07 深夜第三刀，用户点名，未提交）

**对账线输入侧闭合**：`CallRef` 加 `PromptSha256`（实际发出任务书 sha256 前 16 hex）+`PromptRunes`（截断后字数）——锚 **clampPrompt 之后**真正交给 adapter 的那段（截断前哈希会在超长任务书上撒谎，`seatPromptSnapshot` 单一盖章入口）；重试共用同一份任务书，故锚 CallID 级而非 AttemptID 级；只落哈希不落原文（原文落盘挂池）。盖章点=`runOne` 双分支（模型席+CLI 席），讨论/主持/汇总全调用类型自动覆盖。

- `export.go`：贡献导出加「任务书快照：sha256:… · N 字」行（与对账警示行同区）。
- 前端同批：`pantheon.ts` CallRef 接口 + `models.ts` 三个扁平化类（Contribution/Event/Participant）replace_all 同步（`retryOfAttempt` 作锚避开无关类，3/3 零误伤）。
- 测试 +3：指纹确定性（16-hex/不嵌原文/可复算）、截断后锚（超长任务书 runes≤上限且哈希=clamp 后段）、端到端哈希=实发段（捕获型 runner 验证贡献指纹===runner 实收 Stdin 指纹，恰一匹配）。
- 验证：pantheon 17.7s 全绿/全树编译/vet/gofmt 清/tsc 0/vitest 65/65/app 众神殿族绿（M1 不触 app 层 Go 代码，未重跑 186s 全量）。

**两本账对账线至此完整**：输入侧（M1 任务书指纹）+ 输出侧（E1 非空锚+digest）+ 契约侧（①出处+要点）+ 宿主裁决（对账警示行）。剩余可动=B4 预算阶梯、M2 互盲制式，待点名。

### 6.4 B4 整场预算阶梯落地（2026-11-08 凌晨，未提交）

**点名过程**：用户「继续」→ 点名问句未答 → 按既给推荐次序先落 B4（协议不动、代价最小），M2 留池。

**方案调整（落地前实证改判）**：预算单位由 token 改为**议事墙钟（分钟）**。实证：token 账目只覆盖自报 Usage 的席位——DSH（status/step_end）与 ZCode（result）有，codex 只报 thread_id（`seatguard.go` 注释「token 账目暂无同形载体」），模型席 `runOne` 分支根本不回 Usage（`manager.go` seatResult 无 Usage）。四席会里半数席零值，token 阶梯会系统性少算、晚触发；墙钟对所有席位同样可测，且正是实弹痛点本身（83 分钟/140 分钟取消）。token 预算待 codex/模型席账目补齐后再议。

**阶梯两档**（只在宿主确定性检查点生效）：
- **≥80% 历史降档**：`runSeat` 历史形态选择后一行 `budgetDowngradeMode`——seeded/全量窗口退回梯度；delta 已是最低形态不动；梯度是基线不动。
- **≥90% 轮间收口**：`budgetStopsAfterRound` 在 DID 检查之后、轮间小结之前——落 `budget_stop` 事件（与 did_stop 同形，事件即单一事实源，不另设会话字段）+ 警示，跳过小结直接汇总。余 10% 留给汇总（交付物，允许越线完成）。自然末轮不评估（不对本就要结束的会报警）。
- **不抄上游「减席」档**：缺席≠弃权（前序档 C3）。上游四档（80 压缩/90 减席/95 末轮/100 停）在本仓只在轮界执行，90/95/100 合并为一档收口。

**落点**：`budget.go`（新，纯函数+两个检查点+导出块）；`types.go`（`StartRequest.TimeBudgetMinutes` omitempty 0=不限 + `EventBudgetStop`）；`manager.go`（`budgetElapsed` 时钟缝——nil=开议起墙钟，恢复/重启不归零；测试注入固定用时不靠 sleep）；`export.go`（`## 预算` 独立节——任何协议都写；UntilConsensus 停止信号补「预算收口」分支，否则会误报「达到安全上限」）。前端：Composer「每轮确认」旁加分钟输入（0–480，0=不限，Tooltip 写明两档）+ 载荷 `timeBudgetMinutes`（0→不发）+ seed 透传（续议继承）+ `pantheon.ts`/`models.ts` StartRequest 同步（Go 字段序 parentSessionId→timeBudgetMinutes→gateEachRound）。

**测试**：Go +5（百分比/档位边界含超支不截断、降档四态、导出节与收口回读、**收口端到端**——注入 95/100 分钟→第 1 轮后收口，无第 2 轮贡献、事件+警示+导出三处在档、汇总照常完成、**零预算不变量**——注入 10 小时用时但预算 0→无事件无警示无预算节）；前端 +1 用例（填 45 分钟→请求带 45）+1 断言（未填→字段缺省）。**坑**：独立开场第 1 轮贡献 Phase=opening 不是 discussion——按轮次判不按相位名判（首版断言误用相位名假红，dump 会话实态后定位）。

**验证**：pantheon 15.7s 全绿/全树编译/vet/gofmt 清/tsc 0/vitest 66/66/app 全量 179.9s 绿（StartRequest 是绑定类型，全量含绑定面守卫）。

**未覆盖（如实）**：~~降档档的端到端未单测~~（**已补**：`TestBudgetDowngradeSwitchesModelSeatToGradient`——模型席默认全量窗口，对照腿无预算第 2 轮带「（你此前的发言」全文标注，预算腿 85% 高水位标注消失=真退回梯度；双 manager ~0.5s）；真机实弹未跑。余池：M2 互盲制式（待点名）。

### 6.7 收尾补强（2026-11-08 第二轮：挂账清零+构建冒烟）

1. **B4 降档端到端补缺**：见 §6.4 划线更新——用 `fakeGrokBackend` 模型席走通「全量窗口→梯度」的真实切换路径，对照腿证明断言判别力（无预算时标注在场）。pantheon 全量 23.0s 绿。
2. **装机构建冒烟过**：按构建指南（ucrt64 gcc 前置+CGO_ENABLED=1）`wails build` 3m05s 产物 `build/bin/gaea.exe`（sha256 前 16 `b8beb754974798eb`，01:42）；构建前孤儿进程查清（无）。**手同步的 models.ts 与生成器重写输出逐字一致零漂移**（ToolCount ok/nonEmpty/digest、CallRef promptSha256/promptRunes、StartRequest timeBudgetMinutes 全在且风格同款）——四批全部前端字段经真实生成器验证。未安装（不动用户在跑的实例），实弹由用户按需装机。
3. **app 首跑 FAIL 复现尝试**：按原失败条件（app 全量 -v 与前端 tsc+vitest 并发重载）复刻一枪，**全量落盘不截断**——0 FAIL / 161s，vitest 同时 67/67。累计 1 败（用例名不可追溯）4 胜；pantheon 族 app 用例每跑都过、失败与本线改动无可追溯关联。结论维持「疑似 flaky 待观察」；再现时的采集法已标准化（全量落盘+grep `--- FAIL` 上下文）。
4. **提交**：`d74d3bdf`（docs 7 文件）+ `05a1fcd6`（合账 38 文件，双执行体署名）；留树 MemoryHub 五件（第三线）+ `ws_probe.exe`；ahead 4 未 push。

### 6.5 M2 互盲驳论协议落地（2026-11-08，未提交）

**冲突裁决方式**：§2-① 与前序档「轮次形态无增量」的分歧不靠拍板裁——按 `types.go` 协议注释原话「对照实验必须可表达、可比对必须可打标签」落成**第三种对照协议**（默认仍是独立开场），效果交给 Protocol 标签的评测对照。

**核心发现（省掉一半预设工程）**：同轮互盲在 gaea 是**结构性的**——并行轮期间 `history` 切片由父协程独占（WG.Wait 后才追加），且四形态历史投递（梯度/seeded/全量窗口/delta）全部以 `history` 参数为原料；并行轮里各席天然只拿轮基历史（严格往轮）。**零快照代码**，只需把并行条件从「独立开场第 1 轮」扩成「M2 全部轮次」。aidebate 的互盲是「刻意只投对手开场全文」，gaea 的互盲是「并行调度的自然性质」——后者更强（无选择性投递偏差）。

**落点**：
- `manager.go`：`ProtocolBlindRebuttal="blind_rebuttal"` 常量（注释写明收益/代价/默认不变）+ NormalizeProtocol/ProtocolLabel（互盲驳论）+ `parallelRound` 条件扩展（M2 全轮并行；独立开场仍只第 1 轮）+ round==1 盲分支纳入 M2。
- `prompts.go`：`blindRebuttalInstruction`（仅 M2 协议 round≥2）——互盲声明（引用只指向往轮贡献 ID）+ 逐席覆盖义务（对每位参与者上一轮立场各给一段回应，缺陷四类：事实/逻辑/证据/未回应反例）+ 打最强点 + 完全接受须明说。模板加一个 `%s` 槽位（共享观点段后）。
- `eval.go`：`EvaluationVariantBlindRebuttal="blind_rebuttal"` + 标签「互盲驳论」+ 排序接在圆桌接力（历史）后；`EvaluationVariantOf` 在 council 分支后接 M2 分支。
- 前端：协议下拉加「互盲驳论」选项 + 类型联合 + protocolLabel + seed 续议继承 + pantheon.ts 注释。

**设计取舍**：①不定死两轮——M2 泛化为「所有轮次并行互盲」，2 轮即 aidebate 原形（立场→驳论），3+ 轮=多轮盲议；②轮间主席小结保留（收敛牵引不丢）；③与定向质询 preset 的组合未禁（前端协议选择在 preset 下已禁用，后端宽容）；④DID/共识判据/预算阶梯全部自然兼容（它们只看贡献流，不看协议）。

**测试**：Go——归一含 blind、`TestBlindRebuttalRoundsAreParallelAndBlind`（端到端：第 1 轮互盲无驳论指令、第 2 轮两席互见对方 R1、互不见对方 R2、指令与逐席覆盖在位、协议入档、评测变体+标签正确）；前端 +1 用例（选互盲驳论→请求 protocol=blind_rebuttal）。

**验证**：pantheon 19.99s 全绿/全树编译/vet/gofmt 清/tsc 0/vitest 67/67/app 全量三连绿（verbose 1833 用例零 FAIL 零 panic+复跑两绿）。**首跑 FAIL 定性**：与前端 tsc+vitest 并发重载同跑的那一枪失败，用例名因采集用 `tail -3` 截断未捕获（采集失误，已记坑）；随后三连绿+零交集疑点=按「负载期疑似 flaky 待观察」挂账（同 MemoryPanel 负载期 flaky 先例），不定案闭合。

**待实弹**：M2 的真实收益（从众消失+耗时≈2×最慢席 vs 失去接力吸收）由 Protocol 标签评测对照回答——跑同题 M2 与独立开场各一场，对比分歧台账闭合率与墙钟。

### 6.6 收尾审计（2026-11-08，四批合账后自查，hostile-eyes 思想）

逐面核过，两处结论+一处修复：

1. **`budget_stop` 前端时间线可见性：无需改**。`budget_stop` 不在共享时间线事件白名单（`PantheonRecord.tsx:1937`），但收口时宿主同文本落 `warning` 孪生事件（`addWarning` 发 warning 事件），warning 在白名单内→时间线可见；若把 budget_stop 加进白名单，同文本双键反而造重复条目。**双写设计（事件=审计事实源，警示=用户可见面）在此处正好互补**。
2. **单席 M2 无需守卫**：既有校验「非独立开场至少两席」（`manager.go:444/472`，`req.Protocol != ProtocolIndependent`）天然覆盖 blind_rebuttal。
3. **修复：续议 seed 漏带预算**——seed 构造（`PantheonPage.tsx:978-1000`）继承了 protocol/preset/rounds 却漏 `timeBudgetMinutes`，Composer 的 `seed?.timeBudgetMinutes` 恒空。已补：`PantheonComposerSeed` 类型 + seed 构造两处（续议继承预算=与 protocol/rounds 同一继承哲学，Composer 可改）。回归 tsc 0 / vitest 67/67。

### 6.8 更正「池空」误报 + M5/M6 落地（2026-11-08）

**更正**：M2 落地后我两度报「余池空」——**错**。§3 候选表里判「做」却从未落地的还有 M3/M4/M5/M6/B3，前序档的 P0 两刀（A1 pass / A2 canary）及其两个拍板项（是否开、canary 主席失败策略）也一直悬着。误报源于只盯了「点名池」（M1/B4/M2）没回对 §3 全表。

**本轮落地（无拍板依赖的两把提示词小刀）**：
- **M5 开场预期最强反驳**：开场第 4 问扩为「指出最需补充或验证的内容；并用一句话给出你预期的最强反驳与你的回应」——保持五问形状不加编号，台账可更早开条目。
- **M6 汇总缺席纪律**（即前序档 P0-b 附带条款的模型侧，经核对此前未落地）：`synthesisPromptMode` 加显式参 `absentees`（仓内既定模式：`synthesisPrompt` wrapper 传 nil 保兼容）；有缺席才注入【缺席纪律】——点名缺席席位、「只评实际发言者，不得替缺席者虚构立场，缺席≠弃权≠同意」、影响完整性写 risks 注明「缺席未发言」。生产侧 `absentParticipantNames` 按**贡献存档**派生（「已提交者」口径——中途掉线但交过开场的不算缺席，正是 aidebate AD-1 输入缺口的反面），名册在锁内快照（席位观测并发改 Participants）。
- 测试 +4：开场问句在位、缺席段有无两态、「已提交者」口径（掉线交过开场不算/汇总不算发言/从未发言补齐）、**端到端**（一席全程失败→生产路径汇总提示词点名该席）。验证 pantheon 15.4s/全树/vet/gofmt/app 众神殿族绿（日志全量落盘）。

**真正剩余的池**（全部需用户点名或拍板）：
| 项 | 状态 |
|---|---|
| 前序 A1 pass（「无新增」动作）| P0，待拍板「是否开」 |
| 前序 A2 canary（开场冒烟）| P0，待拍板「是否开」+「主席失败=拒绝开场 vs 自动换席」 |
| 前序 A3 议事健康指标 | P1，只展示 |
| M3 对手覆盖矩阵 / M4 缺陷分型 / B3 导出敏感扫描 | P1/P2，判「做」待点名 |
| M7 席外主席 / B2 定向插话 / B6 amend / T1 自测门 | 拍板/观察 |

### 6.9 实弹翻车诊断+修复：waiting_input 答完后的 UI 死路（2026-11-08 上午）

**实弹**：pan_1794098356822（独立开场 4 席 4 轮，议题=工具权限）。**现象**：未形成自动小结、无法继续讨论（无按钮）。**取证**（session.json 全量）：round 4（末轮）DSH 发言后提 NEEDS INPUT（seq 281）→ 轮末停车（seq 283「本轮结束，1 席提问等待回答」→ status=waiting_input）→ 用户 09:28 回答（seq 284）→ 之后零事件。

**根因=纯前端三无死路**（E4 答后不自动续跑的设计 + UI 没给出口）：①WaitingPanel 在「问题全答+无开门」时 return null——等待面板整个消失；②「恢复讨论」按钮只住在失败告示板（failed/cancelled/interrupted 态）；③「基于小结续议」依赖不存在的 verdict。**后端 Resume 本就就绪**：waiting_input 可恢复、无 pending 守卫通过、resumeRound=5>4 跳过轮循环**直接进汇总**（用户的回答已在存档 user_answer 贡献里，汇总可见）。

**修复**（纯前端）：WaitingPanel 待办清空且 status=waiting_input 时改出「待办已清，讨论可以继续」面板——【继续讨论】主按钮（onResume）+ 提示「轮次已用完时续跑直接进入协作汇总」；非 waiting_input 维持原 null。+3 vitest（答完出钮触发 onResume/仍有未答保持原面板/running 不渲染）。

**验证**：vitest 70/70、tsc 0、重建壳 `798681431c53508e`。**用户解法**：装机新壳后打开该会话→点「继续讨论」→直接进汇总（回答已入档，汇总吃全量）。
