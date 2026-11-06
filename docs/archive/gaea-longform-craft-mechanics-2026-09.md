# 长篇小说创作的机制清单（方法论 → 可校验机制）

**两条总原则。**
① **确定性优先**：凡能用结构字段、计数、图算法、统计量判定的，绝不交给 LLM。LLM 只承担语义/情感/意图类判断，且必须输出结构化字段 + 原文证据片段，便于抽查误报率。
② **可写、可注入、可回写**：一个机制若不能"立项时写入数据 / 生成前注入约束 / 成文后回写状态"，就只是口号，应降级为人工评审清单。所有机制共享一个**单一事实源（single source of truth）**：先更新状态库，再生成下一单元；正文与状态库冲突时以状态库为准并触发回填。

---

## M1 场景—续场双拍（Scene–Sequel / Action–Reaction Cycle）

- **失败模式**：中段塌陷；场景变流水账（"事情发生了"但不推进任何东西）；节奏平坦。
- **创作学来源**：Dwight Swain《Techniques of the Selling Writer》提出 scene = goal / conflict / disaster，sequel = reaction / dilemma / decision；Randy Ingermanson 的 [Writing the Perfect Scene](https://www.advancedfictionwriting.com/articles/writing-the-perfect-scene/) 做了可操作化总结。
- **可机械化**：每个叙事单元落库时必须声明 `type ∈ {scene, sequel, montage}`；scene 必须有 `goal / conflict / turnout`，sequel 必须有 `reaction / dilemma / decision`；缺字段即硬闸拒绝生成。`turnout` 必须是"比进入时更糟"的负向变化（yes-but / no-and 变体），且 `sequel.decision` 必须等于下一 scene 的 `goal`（链式匹配）。
- **必须追踪的状态**：`unit_id, type, goal, conflict, turnout(±), reaction, dilemma, decision, pov, time_place_anchor`，以及本场改变的下游字段清单。
- **可自动校验**：确定性——字段齐全度、decision→goal 链式一致性、turnout 极性；LLM——"该冲突是否构成真正对抗"。误报风险：抒情章/蒙太奇天然无 goal，须用 `type=montage` 白名单豁免。
- **优先级**：**P0**。最便宜的硬闸，直接消灭"没有冲突的场景"。

## M2 场景价值翻转与故事网格五律（Value Turn & Five Commandments）

- **失败模式**：场景"没有转弯"，读者感觉原地踏步；章节无状态变化；幕结构缺失。
- **创作学来源**：Robert McKee《Story》——场景必须转动一个价值电荷，[Do Your Scenes Turn?](https://mckeestory.com/do-your-scenes-turn/)；Shawn Coyne 的[故事五律](https://storygrid.com/five-commandments-of-storytelling/)（inciting incident / progressive complications / crisis / climax / resolution），可递归应用于全书、幕、章、场景。
- **可机械化**：每个 scene 记录 `value_at_stake`（如信任/自由/生存）、`value_open_charge`、`value_close_charge`，要求开闭电荷不同（+ → − 或 − → +）；五律作为递归模板，全书/每幕/每章/每场都必须能定位到一个 `crisis`（≥2 选项）与 `climax`（选择后的行动）。
- **必须追踪的状态**：价值维度表；各层级五律定位；crisis 的选项与各自代价；climax 的选择结果与不可逆后果。
- **可自动校验**：确定性——电荷是否变化、crisis 是否给出 ≥2 选项、五律各层覆盖；LLM——"这是真两难还是伪选择"。误报风险：氛围章允许标记 `non-turning`，但需限额（如全书 ≤10%）。
- **优先级**：**P0**。与 M1 共同构成"这一场有没有用"的最小判据，且完全可结构化。

## M3 错误信念弧线与 Want/Need 水位（The Lie the Character Believes）

- **失败模式**：人物动机漂移；角色成长不可见（结局的转变没有铺垫）；中段主角沦为工具人。
- **创作学来源**：K.M. Weiland [The Lie the Character Believes](https://www.helpingwritersbecomeauthors.com/new-way-to-think-about-the-lie-the-character-believes/)；Lisa Cron《Story Genius》的 misbelief 与场景工作表（[Plottr 版](https://plottr.com/story-genius-scene-worksheet/)）；John Truby 的 want vs need（[The Anatomy of Story](https://archive.org/details/anatomyofstory220000trub)）。
- **可机械化**：主角必须有 `want`（外部、可判成败）、`need`（内部）、`misbelief`（一句陈述）与"该信念驱动的行为模式"；三段配额——**建立**（前 1/3 内 ≥2 次错误行动，首次不晚于全文 10%）、**挑战**（中段 ≥3 次现实反证，代价递增）、**推翻/确认**（终局由行动证明，禁止台词宣告）。`arc_waterline` 除刻意 regression（需白名单）外单调推进。
- **必须追踪的状态**：每角色 `misbelief_text, want, need, waterline, 触发事件 id + 方向, 抗性强度`；反派/配角的独立信念系统。
- **可自动校验**：确定性——分段配额计数、水线单调性、终局 arc beat 存在性；LLM——"该行动是否真由 misbelief 驱动"。误报风险：多主角并行时归属错判，必须绑定 `pov` 与 `actor_id`。
- **优先级**：**P0**。弧线是长篇最常崩塌的部分，且配额计数完全确定。

## M4 类型义务场景与惯例（Genre Obligatory Scenes & Conventions）

- **失败模式**：结尾无力；读者类型期待落空（悬疑无审讯、推理无线索公平性、爱情无"虚假胜利/表错情"）；genre promise 未兑现。
- **创作学来源**：Shawn Coyne 的 [Editor's Six Core Questions](https://www.storygrid.com/wp-content/uploads/2017/12/THE-EDITOR%E2%80%99S-SIX-CORE-QUESTIONS-Genre.pdf) 与 [Performance Story Cheat Sheet](https://storygrid.com/wp-content/uploads/2017/09/Performance-Story-Cheat-Sheet.pdf)；Blake Snyder《Save the Cat》类型分类（[官方 beat sheet](https://savethecat.com/get-started/page/118)、Jessica Brody [小说版 starter kit](http://www.jessicabrody.com/wp-content/uploads/2020/01/Save_the_Cat_Writes_a_Novel_Starter_Kit_v6.pdf)）。
- **可机械化**：立项即声明 `primary_genre` + 至多 1 个 `secondary_genre`，载入该类型的义务场景清单（每项 = 必须出现的关键事件 + 必须被回答的核心问题 + 惯例检查项）；每项绑定到具体 unit，或标记 `waived` + 理由。终局前做一次未兑现扫描。
- **必须追踪的状态**：`genre_id, obligatory_scene[]{scene_id, status: planned|written|waived, 核心问题, 是否被回答}, convention_checks[], 失约风险等级`。
- **可自动校验**：确定性——清单覆盖与 waiver 完整性；LLM——"该场景是否**实质**兑现义务"。误报风险：场景存在但敷衍（关键词匹配漏检）；跨类型融合会被误判为缺失。
- **优先级**：**P0**。解决"结尾无力"最直接，且清单是静态数据。

## M5 节拍网格与张力曲线（Beat Grid & Tension Curve；Promise–Progress–Payoff）

- **失败模式**：张力平坦（一条直线）；中段塌陷；高潮提前泄压；结局的 payoff 少于 promise。
- **创作学来源**：Blake Snyder 的 15 节拍；Coyne 的 beginning hook / middle build / ending payoff 与"每 1/5 处转折"；Brandon Sanderson 的 [Promise–Progress–Payoff](https://podcasts.apple.com/za/podcast/promise-progress-payoff-plot-theory-brandon-sandersons/id1798308121?i=1000695633007)。
- **可机械化**：给每个 unit 打 `tension`（0–10，由结构化因子合成：风险等级、代价不可逆性、倒计时紧迫度、信息增量），生成整书曲线；硬约束——每 N 个 unit 至少一个局部峰值；中段不得连续 K 个 unit 张力低于前段均值；climax 必须是全局最大值，且其后无可逆让步。
- **必须追踪的状态**：`tension per unit`、局部峰值表、倒计时/时钟存续、stakes 最大值、payoff 队列。
- **可自动校验**：确定性——方差、峰值间隔、单调窗等统计量；LLM——张力打标本身。误报风险：慢热文学类型与"峰值间隔"规则冲突，须按 genre 切换曲线模板，不用绝对阈值。
- **优先级**：**P0**。曲线是全局诊断，也是阅读体验最好的可计算代理指标。

## M6 线程台账与 MICE 配额（Thread Ledger & MICE Quotient）

- **失败模式**：支线遗忘（open loop 从此消失）；线索堆叠导致结构失衡；结尾关闭顺序混乱。
- **创作学来源**：Mary Robinette Kowal 的 MICE Quotient（Milieu / Inquiry / Character / Event，[Writing Excuses 16.35](https://writingexcuses.com/16-35-what-is-the-m-i-c-e-quotient/)），含"嵌套线程后开先关"规则。
- **可机械化**：每个 thread 声明 `mice_type, open_unit, close_unit, parent_thread, status ∈ {open, parked, closed, abandoned}`；硬约束——结局前所有线程必须 `closed` 或有**显式** `abandoned`（且正文中必须有收束动作，不能静默丢弃）；嵌套线程关闭顺序默认 LIFO，破例需显式声明。
- **必须追踪的状态**：线程表、父/子嵌套关系、热度（距最近一次被触碰的 unit 数）、读者承诺强度。
- **可自动校验**：确定性——开/关配对、LIFO 检查、`parked` 超时告警（"这条线 8 章没碰过"）；LLM——"该收束是否令读者满意"。误报风险：复杂多线作品的合理 LIFO 破例需白名单。
- **优先级**：**P0**。"支线遗忘"是长文本最普遍的崩塌点，而规则完全确定。

## M7 伏笔池与回收契约（Setup/Payoff Ledger）

- **失败模式**：伏笔悬空；机械降神（未埋的解决手段）；契诃夫之枪未响；promise 多、payoff 少。
- **创作学来源**：Sanderson 的 promise/progress/payoff（同上）；Coyne 关于线索与信息释放的规定；契诃夫原则（[Story Grid 一页纸诊断](https://storygrid.com/foolscap/) 亦要求识别关键线索）。
- **可机械化**：每个 setup 记录 `planted_unit, surface_form（读者可见的具体物件/台词）, promise_type（物件/能力/关系/信息/规则）, 期望回收窗口[unit 区间], payoff_unit, 回收方式（直接/反转/被打破）`。硬闸两条——进入终局段（最后 15%）时任何 promise 无 payoff 计划即阻断生成；**反向扫描**：climax 用到的每个关键手段必须有前置 setup id。
- **必须追踪的状态**：promise 池、payoff 队列、未回收清单、回收窗口、读者可见性标记（在**正文**出现过，而不只是在设定里）。
- **可自动校验**：确定性——配对、窗口越界、"climax 手段无前置"（能抓住绝大多数结尾无力与降神）；LLM——`surface_form` 是否真被读者看见。误报风险：设定里写了但正文没写＝伪伏笔，必须做正文字符串/语义双检。
- **优先级**：**P0**。

## M8 信息状态表与认知差（Knowledge Ledger & Dramatic Irony）

- **失败模式**：信息早泄（悬念被提前答完）；读者困惑（关键信息从未给出）；纸片反转（结尾揭示读者无法回溯的信息）。
- **创作学来源**：Coyne 的信息释放节奏与 [Editor's Six Core Questions](https://www.storygrid.com/wp-content/uploads/2017/12/THE-EDITOR%E2%80%99S-SIX-CORE-QUESTIONS-Genre.pdf)；Truby 的 reveal 与认知层级；Lisa Cron《Wired for Story》关于大脑对"未闭合问题"的反应（[作者资源页](https://www.wiredforstory.com/free)）。
- **可机械化**：三张表——`world_fact`、角色认知（character × fact × {不知|怀疑|已知|误解}）、读者认知（reader × fact）。每场戏声明 `information_delta`，并强制"问题—延迟—部分回答—新问题"链条：任何 fact 进入 reader 表后，必须在同场或邻近 unit 至少引入 1 个新问题，`open_question` 计数在终局前不得归零。
- **必须追踪的状态**：fact 表、per-character belief 状态、reader 已知集、open question 队列及年龄、刻意植入的错误认知及其纠正点。
- **可自动校验**：确定性——问题计数、状态矛盾（角色用到了某 fact 而状态仍是"不知"）、结尾 reveal 的 fact 是否曾提前进入 reader 表；LLM——"延迟是否有效而非拖延"。误报风险：隐瞒型叙事、不可靠叙述者会与状态矛盾检查冲突，需按作品声明关闭部分检查。
- **优先级**：**P0**（信息早泄是弃书头号原因；表结构简单但必须与生成流程强绑定）。

## M9 因果链与时间线台账（Causal Ledger & Timeline；"therefore" vs "and then"）

- **失败模式**：episodic（事件像清单，可任意换序）；时间线矛盾；行为无前因（动机断裂）。
- **创作学来源**：Cron 的因果链（Story Genius 强调 cause-and-effect 而非 and-then）；Trey Parker / Matt Stone 的 "therefore / but" 测试；Coyne 的"每个场景必须改变下一个场景"。
- **可机械化**：每个 unit 必须记录 `causes[]`（来自哪些前序 unit 的具体状态）与 `effects[]`（改变了哪些状态字段）；确定性闸门——除开场外每个 unit ≥1 条入边、除终场外 ≥1 条出边，并做整图连通性检查；时间线表记录 `story_time、逝去时长、角色位置`，同一角色不得同时身处两地。
- **必须追踪的状态**：因果 DAG、状态快照版本、时间线、角色位置表、道具归属表。
- **可自动校验**：确定性——DAG 连通性/孤立节点、时间与位置冲突、行程时长是否可行（距离 ÷ 速度粗算）；LLM——"这个 because 是否成立"。误报风险：巧合与超自然因果会被标红，需要设定白名单。
- **优先级**：**P0**。这是把"事件清单"和"故事"区分开的核心判据，图算法完全确定。

## M10 场景去重与节拍指纹（Beat Fingerprint & Scene Dedup）

- **失败模式**：重复场景（同一冲突换个地点再打一次）；同一情绪节拍反复；文本层自相似（AI 味的环形重复）。
- **创作学来源**：Coyne"没有两个 progressive complication 可以相同"；修订工艺中的 scene inventory（场景清单）做法；编辑实务的去味准则（crutch words / 结构性重复）。
- **可机械化**：每个 unit 生成指纹 `{pov, location, goal_type, conflict_type, outcome_polarity, emotional_valence, 关键道具, 对抗双方}`；任意两 unit 指纹相似度超阈值即告警"可能重复"，并要求声明"本场与那场的关键差异"。文本层：n-gram 重复率、句长分布、段落开头模式、高频修饰词黑名单。
- **必须追踪的状态**：指纹向量库、已用冲突类型计数、本书文本统计基线（不引用外部绝对阈值）。
- **可自动校验**：确定性——指纹冲突与 n-gram 统计（高精度）；LLM——"两场戏是否真的同质"。误报风险：刻意回环/递进重复是常见修辞，阈值必须在**本书内部**标定。
- **优先级**：**P1**。对 AI 生成的质量提升极大，但阈值需按书校准，先告警不硬闸。

## M11 设定与规则台账（Canon & Rule Ledger）

- **失败模式**：规则漂移（能力忽强忽弱）；设定自相矛盾；魔幻解决（能力无代价）；角色知识越界。
- **创作学来源**：Brandon Sanderson 三定律（"限制 > 力量"、"先解决问题再深化"）；Story Grid 的 world/canon 层；系列写作 bible 传统。
- **可机械化**：canon 条目 = `{rule_id, 表述, 成本, 限制, 可观测性（读者是否已知）, 被引用处}`；生成前注入"本场相关 canon 子集 + 已确立的限制"，生成后回写新确立规则。硬闸：任何解决手段必须可追溯到已确立规则且不违反其成本；能力变化必须带"获得事件"id。
- **必须追踪的状态**：规则表、能力/道具的持有者与状态、规则可见性、违反日志。
- **可自动校验**：确定性——成本与限制字段是否声明、能力—持有者一致性、未确立规则被使用（关键词 + rule_id 双检）；LLM——"该使用是否违背规则精神"。误报风险：软魔法体系与诗意留白天然模糊，需按作品声明 hard/soft magic。
- **优先级**：**P1**。长篇与系列必备；单本可轻量化。

## M12 主题论证与反论压力（Moral Argument & Counter-Argument）

- **失败模式**：主题漂浮（"好看但没说什么"）；结尾道德结论与前文矛盾；说教（台词代替证明）。
- **创作学来源**：John Truby 的 moral argument 与 22 steps（[The Anatomy of Story](https://archive.org/details/anatomyofstory220000trub)）；Coyne / McKee 的 controlling idea（价值 + 原因）。
- **可机械化**：立项必须写一句**控制理念**（`X 因为 Y 而胜/败`）与**反控制理念**；每幕至少一个 unit 让反论获得其最强陈述；终局 climax 的选择必须是该论证的"实验证明"，胜负由行动与代价体现（禁止旁白结论）；建立说教句式黑名单（抽象名词 + 判断句）。
- **必须追踪的状态**：`controlling_idea, counter_idea`、主题压力表（正/反论在各段的强度）、主角最终选择及其代价、说教句命中记录。
- **可自动校验**：确定性——控制理念是否存在、每幕反论陈述是否存在、climax 是否有选择记录；LLM——"结尾价值取向是否与 climax 一致""是否说教"。**这是全清单里误报率最高的一项，只做评审报告，不做闸门。**
- **优先级**：**P2**。对"优秀"极关键，但自动判定不可靠。

## M13 叙事声音与风格指纹（Voice Fingerprint & De-AI Scrub）

- **失败模式**：所有角色同一声音；AI 味（句长均匀、总结式收尾、抽象名词堆砌、解释性副词）；telling 而非 showing。
- **创作学来源**：Swain 的 MRU（motivation–reaction unit，见 [Writing the Perfect Scene](https://www.advancedfictionwriting.com/articles/writing-the-perfect-scene/)）；受限视角/自由间接引语传统；编辑实务的 style sheet。
- **可机械化**：每角色 `voice_profile`（句长分布、词汇层级、比喻来源域、口头禅、禁用词），生成时作约束注入、生成后算分布距离；确定性规则——每千字感官动词下限、抽象名词上限、`-ly` 副词上限、以总结句结尾的段落比例上限、对白/叙述比、同段 POV 越界检测（只能写出 POV 角色可感知的信息）。
- **必须追踪的状态**：voice profile、style sheet、本书统计基线、POV 模式（第一人称/第三限知/全知）、禁用清单。
- **可自动校验**：确定性——统计量、POV 越界（实体—感知映射：人物/地点是否在 POV 视野内）、对白归属清晰；LLM——"是否真的像这个角色在说"。误报风险：刻意模仿、方言、实验文体。
- **优先级**：**P1**。决定"读起来像人写的"，但必须在 P0 结构之后，否则只是给塌陷的故事化妆。

## M14 冷读闸门与修订闭环（Cold-Read Gate & Revision Ledger）

- **失败模式**：局部完美、全局失效（每章都不错，合起来不成立）；连续生成累积漂移；修改引入新矛盾。
- **创作学来源**：编辑实务的 cold read 与 continuity pass；Coyne 的逐层诊断（[Foolscap 一页纸](https://storygrid.com/foolscap/)）；Fictionary StoryCoach 的场景审计思路（[sample edit](https://fictionary.co/wp-content/uploads/2020/08/Fictionary-StoryCoach-Sample-Edit.pdf)）。
- **可机械化**：三级闸门——**(1) 结构闸**：M1–M9 的确定性规则，每次生成后运行，失败即阻断；**(2) 连续性闸**：合并/回填后对全量 unit 重跑时间线、状态、伏笔一致性；**(3) 冷读闸**：用"无上下文读者"模型逐章回答固定四问（本章谁想要什么 / 阻碍是什么 / 章末状态变了什么 / 产生了什么新问题），任一问无法回答即"读者困惑"信号。
- **必须追踪的状态**：每 unit 的校验结果与版本、违规历史、修订请求队列（问题 → 受影响 unit 集合 → 需回填的状态）、生成参数与 prompt 版本（保证可复现回归）。
- **可自动校验**：混合。关键指标——**同一状态库重跑同一条规则，差异必须为 0**（非确定性规则禁止入闸）；冷读闸误报集中在慢热/文学化章节，须允许人工 `waiver` 并留痕。
- **优先级**：**P0**。没有闭环，前面所有机制都会随篇幅腐化。

---

## A. 一部"优秀长篇"的机械判据清单

| # | 判据 | 具体形式 / 阈值 | 判定方式 |
|---|---|---|---|
| 1 | 每个单元都有状态变化 | `close_state ≠ open_state`（至少一个被追踪字段改变） | 确定性 |
| 2 | 每幕/每章有五律结构 | 各层级能定位 crisis（≥2 选项）与 climax | 确定性 |
| 3 | 场景价值电荷翻转 | `value_open_charge ≠ value_close_charge` | 确定性 |
| 4 | 每场有可验证目标 | `goal` 有可判定成败，冲突真实对抗 | 确定性 + LLM |
| 5 | misbelief 前 1/3 建立 | 前 1/3 内 ≥2 次错误行动，首次 ≤ 全文 10% | 确定性 |
| 6 | 中段被挑战且代价递增 | 中段 ≥3 次反证，代价序列严格递增 | 确定性 + LLM（强度） |
| 7 | 结尾推翻或确认 | 终局 arc beat 由行动完成，与 climax 选择绑定 | 确定性 + LLM（性质） |
| 8 | want 与 need 冲突 | 两者存在，且 ≥3 处冲突记录 | 确定性 + LLM |
| 9 | obligatory scenes 到位 | 类型清单全覆盖或显式 waived | 确定性 |
| 10 | 张力曲线非平台 | 峰值间隔上限、中段窗口方差下限、climax 为全局最大 | 确定性 |
| 11 | 无重复场景 | 指纹冲突为 0（本书内标定阈值） | 确定性 + LLM 复核 |
| 12 | 伏笔全部回收或显式放弃 | promise 池清空或 `abandoned` 留痕 | 确定性 |
| 13 | 无机械降神 | climax 每个关键手段有前置 setup id | 确定性 |
| 14 | 每条支线有开有关 | thread 配对完成，LIFO 或有破例声明 | 确定性 |
| 15 | 信息按"问题—延迟—部分回答—新问题"推进 | `open_question` 计数终局前恒 ≥1 | 确定性 |
| 16 | 无信息早泄 | reveal 依赖的 fact 此前未进入 reader 表 | 确定性 |
| 17 | 因果连通 | DAG 无孤立节点，每 unit 有入边/出边 | 确定性 |
| 18 | 时间线自洽 | 位置、时刻、时长无冲突，行程可行 | 确定性 |
| 19 | 规则不漂移 | 无未确立规则被使用，成本/限制被遵守 | 确定性 + LLM |
| 20 | 无 POV 越界 | 场内信息均为 POV 角色可感知 | 确定性 + LLM |
| 21 | 无 telling 泛滥 | 抽象名词/副词密度低于本书基线 | 确定性 |
| 22 | 角色声音可区分 | 角色间句长/词汇分布距离超阈值 | 确定性 |
| 23 | 主题可陈述 | 控制理念与反理念存在，climax 选择体现之 | LLM / 人评 |
| 24 | 结尾不无力 | 终局有不可逆代价、无余留主线问题、无"然后呢" | LLM / 人评 |
| 25 | 冷读通过 | 逐章四问均可回答 | LLM |

**读法**：1–22 应全部自动化并进闸门；23–25 只作为评审报告。任何一项被 `waiver` 都必须写入修订队列并留痕，否则"优秀"无法回归复现。

---

## B. 三层架构建议

**一段话**：把系统切成**故事层 / 场景层 / 文本层**三级，级间只通过**结构化契约**通信，且方向必须是"上层下发约束 → 下层生成 → 下层回写状态 → 上层重校验"。故事层是唯一事实源，持有弧线、线程、伏笔、时间线、canon、控制理念；它不生成文本，只回答"下一场必须履行什么义务"。场景层把义务编译成一张场景卡（五律定位 + 目标/冲突/转折 + 要推进的 thread + 要触碰的 arc beat + 允许释放的 fact 子集），是**唯一可以阻塞生成的闸门所在层**。文本层只负责把场景卡写成句子，受 voice profile 与禁用项约束，回写统计量与新增 canon/伏笔/事实的候选，绝不自行决定情节。三层之间必须共享同一个 `unit_id` 主键与状态版本号，否则回写无法定位、修订无法回归。

| 层 | 该持有什么数据 | 生成时怎么注入 | 写作后怎么回写与校验 |
|---|---|---|---|
| **故事层**（整书结构/弧线/主题） | 控制理念与反理念；arc 水位；thread 表（MICE、开/关、嵌套）；promise 池与回收窗口；fact 表与 reader/character 认知；因果 DAG；时间线；canon 规则；genre 义务场景清单；张力曲线 | 只注入**本场相关的切片**：要触碰的 arc beat、要推进/收束的 thread、可释放的 fact 子集、必须兑现的 obligatory scene、禁止越界的 canon 限制 | 回写状态变更（水位 ±、thread 热度、promise 新增/回收、fact 认知转移、时间线推进、新 canon），重跑 M3–M9、M11 的确定性规则与全局曲线 |
| **场景层**（场景卡/续场/节拍） | 每 unit：type、goal/conflict/turnout、reaction/dilemma/decision、value charge、五律定位、enter/exit state、指纹、tension、causes/effects | 把上层义务编译为硬字段与硬闸（缺字段/电荷未翻转/义务未绑定 → 拒绝生成）；给文本层下发"beat 序列 + POV + 声音 profile + 禁用项" | 校验字段齐全、决策链闭合、与前序 unit 的因果边、指纹去重；把本场实际发生的事（而非计划）回写为下一场的输入 |
| **文本层**（句子/风格/去味） | voice profile、style sheet、POV 模式、禁用清单、本书统计基线、段落/对白统计 | 注入 voice profile 与本章节拍清单；约束句长分布、感官细节下限、副词上限、禁说教句式 | 回写统计量（句长、n-gram 重复、POV 越界、telling 指标），并**提案**新伏笔/新事实/新 canon（须经场景层确认后入账）——文本层不得直接改故事状态 |

**主流工具在这一层的做法与公认短板**

- **Novelcrafter**：强在 codex（场景级自动挑选相关设定条目注入，[codex 文档](https://www.novelcrafter.com/help/faq/codex/detail-codex-entries)）——本质是**检索式记忆**；弱在没有弧线/张力/伏笔的机器判据。
- **Sudowrite**：强在 prose 与 Muse 的灵感扩展、[Story Bible](https://sudowrite.com/blog/story-bible-template-how-to-build-one-and-how-sudowrite-does-it-for-you/) 便利；弱在 bible 是"参考文本"而非可校验状态机，长篇越写越漂。
- **Plottr**：强在节拍/时间线可视化与 [Story Genius 场景工作表](https://plottr.com/story-genius-scene-worksheet/)这类结构模板；弱在纯人工操作，没有生成与回写闭环。
- **Fictionary**：强在**成稿后**的结构审计（场景清单 + 故事要素检查，[StoryCoach 示例](https://fictionary.co/wp-content/uploads/2020/08/Fictionary-StoryCoach-Sample-Edit.pdf)）；弱在检查发生在事后，无法作为闸门阻断生成。
- **NovelAI**：强在 lorebook 与上下文预算控制、可复现采样（[官方文档](https://docs.novelai.net/)）；弱在零结构概念——记忆停留在 token 层，故事层完全由用户的头脑承担。

**共性短板（也是这套机制要补的四个洞）**：① **记忆 ≠ 状态**：检索到的设定不等于"世界已经改变成什么样"；② **没有承诺—偿还契约**：伏笔、支线、义务场景只能靠人记；③ **校验在事后或人工**，无法阻断生成；④ **一致性靠上下文长度而非不可变账本**，篇幅越长漂移越大。因此落地顺序建议：先做 M1/M2/M7/M9/M14（结构硬闸 + 闭环），再做 M3/M4/M5/M6/M8（曲线与账本），最后做 M10–M13（去重、canon、主题、声音）。
