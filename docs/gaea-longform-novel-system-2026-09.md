# 长篇小说创作系统（gaea 小说域重构方向）——规格

> 2026-09-28 立项。来源=用户指令「你要解决的是如何实现写一个优秀的长篇小说，不是修修补补」。
> 本文档回答三件事：**为什么现在的链路写不出优秀长篇**（诊断，全部带代码证据）→
> **优秀长篇需要哪些可机械化的机制**（创作学转译）→ **怎么分阶段落地**（刀序与验收）。
> 关联：`docs/gaea-novel-revolution-2026.md`（场景制/生成门，已收官）、`docs/distill/`
> （MuMu 蒸馏六域）、`docs/gaea-optimization-direction-2026-09.md`（审计候选池）。

## 0. 一句话结论

现在的 gaea 是**「逐章文本生成器 + 事后体检报告」**：写一章时模型的全部"意图输入"
≈ 一句剧情要求；章节计划契约（`ChapterPlan`）落库后**零消费**；分析产出的结构化情报
（推进点/节奏/建议/钩子）**不回流**成下一章的规划；质量门只提示不收敛。
长篇写不好不是句子问题，而是**结构、意图、状态三者的闭环缺失**。

## 1. 诊断（每条都能指到代码）

### 1.1 写一章的"意图输入"只有一句剧情要求
`prompts/create-chapter.json:5-26` 的槽位共五个：`plot_req`（本章剧情要求）· `setting`
（整篇世界观 markdown，**未按相关性裁剪**）· `characters` · `prev_summary`（最近 10 章 ×
180 rune 摘要）· `character_states`（P1）。`internal/app/create_chapter_handler.go:85-100`
按此组装，再追加伏笔/文风/世界观要点（`:98`、`ctxBudgetTotal=4000`，`:740`）与场景圣经
（`:126-135`、`ctxSceneBibleBudget=2200`，`:747`）。

后果：模型知道"不能写错什么"（设定/状态/伏笔），但**不知道这一章必须完成什么**——
没有叙事目标、没有关键事件、没有冲突类型、没有结尾类型、没有情绪走向。5000 字的输出，
"为什么写这一章"的信息量不足 100 字。→ 逐章局部最优，全书无形状。

### 1.2 章节计划契约存在，但零消费方
`internal/types/plot_v2.go:11-22` 定义了 `ChapterPlan`：`PlotSummary(200-300字)` ·
`KeyEvents(2-4条，跨章不得重复)` · `CharacterFocus` · `EmotionalTone` · `NarrativeGoal` ·
`ConflictType` · `EndingType(5值)` · `EstimatedWords`；落盘口径写的是
`chapters/plans.json`（`:9`）。而 `releases/v4.278.0.md:114` 自己记着：
「t1 契约中除伏笔体检外的类型**尚无消费方**（…、`ChapterPlan`、…）」——全仓 grep 证实：
除类型定义与 `compat_test.go` 往返测试外，**无任何读写路径**；生成链、写前契约、
分析回写都不认识它。

后果：长篇最需要的"章节级意图"在数据模型里躺着，在创作链路上不存在。

### 1.3 生成单位错了：整章 blob，场景是事后物化
`internal/app/create_chapter_handler.go:482-489`：生成完整章 → `WriteChapter` 落 blob →
`rebuildScenesFromBlob(pm, targetNum)` **物化成单个场景**（注释自陈"整章重写=场景重置…
删除既有场景并从新 blob 物化单场景"）。也就是说 v4「场景制」在**生成侧不成立**：
场景是排版容器，不是叙事单位。不足字数时靠"续写"补（`maxContinues=20`，`:115`、
`chapterCurrentBody` `:254`）。

后果：一章内部的场景数量、场景边界、场景目标/冲突/转折全由模型即兴决定 →
节奏塌陷（该收的戏拖长、该展开的戏一句带过）无法治理；场景级重写至今被拒绝
（`novel_rewrite_handler.go:155`「场景工程章暂不支持局部重写」）。

### 1.4 场景卡缺创作学核心字段
`internal/types/types.go:246-259` 的 `SceneMeta` 有：`POVCharID / Location / TimeOfDay /
Emotion / Tags / Status / Order / WordCount`——**坐标齐、工艺缺**：没有
goal（这场戏要什么）、conflict（谁/什么阻挡）、turn（价值翻转/转折）、outcome（结果，且
必须改变状态）、sequel（reaction·dilemma·decision）、exit hook（退出钩子）。
`internal/types/analysis_v2.go:134` 的 `AnalysisScene` 同样只有 `Location/Atmosphere/Duration`。

后果：系统能描述"戏在哪、谁在看"，不能描述"戏要干什么、怎么算写成了"。

### 1.5 分析情报只做体检，不回写规划
分析 V2（`internal/types/analysis_v2.go`，9 维）落 `analysis-v2.json` 后，被消费的只有三类：
角色状态差分（`characterstate.ApplyChapterDiff`）、伏笔回收（`analysis.SyncForeshadows`）、
记忆抽取（`analysis.ExtractStoryMemories` → 语义召回）。而 `plot_points`（含 importance/impact）、
`pacing`、`dialogue_ratio`、`scores`、`suggestions` **只用于展示**：`suggestions` 唯一的
程序化消费是重写弹窗的勾选清单（`RewriteModal` → `NovelChapterSuggestions`）。
→ 「写完这一章得到的情报」不进入「下一章怎么写」的决策。

### 1.6 写前契约是软提示，且只查大纲节点的空字段
`internal/novelgate/gate.go:41-62` `OutlineContractIssues(node)`：只检查节点
Title/Summary/KeyPoints/Emotion 是否为空，注释明写「缺项**不阻断**创作」。
它不看 `ChapterPlan`，也不看场景卡。→ 模型可以在"没有任何抓手"的状态下开写，
系统只在事后告诉你写得不够好。

### 1.7 质量门不收敛
写后有：确定性硬信号（`ChapterQualityIssues`：段落堆叠/电报体/标点堆砌…）、平台评审
（`novelreview`，15 维 PASS/WARN/FAIL + S1~S4）、AI 味打分与去味、伏笔一致性 Lint、
情感曲线、章际对比。全是**报告**；从报告到修复是人工动作（整章重写/局部重写/去味），
没有"体检不过 → 定向修补 → 复检 → 判收敛"的自动闭环，也没有收敛判据（避免无限重写）。

### 1.8 上下文是"堆料"，不是按优先级编译
同一次生成里，"世界观"至少出现三处：模板 `setting` 槽位（整篇 markdown，未裁剪）、
`buildWorldviewSection`（`create_chapter_handler.go:891`，并进 4000 rune 池）、
`SceneBible`（`:126`，2200 rune）。文风同样两处（`buildStyleSection` `:839`、
SceneBible 内文风区段）。各池独立、互不知情，没有"总预算 + 优先级 + 去重 + 相关性"的
编译层。→ 真正的结构信息（若有）会与整篇设定争预算。

### 1.9 风格一致性靠事后打分与手写偏好
`internal/novelstyle` 提供指纹（ComputeFingerprint/Delta/ScoreText）、参考档
（`fingerprint.json`）与去味；`style.md` 是作者手写偏好（`:839`）。但**生成时不学习作者
成稿的表达习惯**去约束输出——参考档只用于"体检打分"。

### 1.10 整书层没有任何持续状态
没有主题论证（这本书在讲什么道理）、人物弧线水位、结构节拍位（第几幕/哪个转折点）、
支线开关表、未解问题池、读者已知/未知信息边界。唯一接近的是大纲树（卷/章）+ 伏笔登记
+ 角色状态。→ 纵向结构（长篇的命门）在系统里没有承载物。

### 1.11 故事主线字段只有读取路径，没有写入路径（最致命的一条）
`internal/types/types.go:189` 定义 `StoryThread`，`internal/novelcontext/*:230-235` 会把它
注入生成上下文；但全仓 grep 显示**无任何写入点**：`internal/outline/outline.go` 的
`Continue`/`ExpandNode` 把模型回复解析进 `newOF` 后**只回收 `.Nodes`**（`:186-207`），
`story_thread` 被丢弃；所有 `WriteOutlines` 调用点（`outline.go:329/351/356/371/375/394/410`、
`outline_handler.go:168`、`create_chapter_handler.go:588/683/720/729`）只写 Nodes。
→ `prompts/outline-continue.json:1` 要求模型基于【故事主线 P0】规划五卷，而该槽位**恒为空**：
整本书的主题/核心冲突是模型每次临时编的，且永远不会被保存与复用。这是"没有纵向结构"的
真正原因，比"没有骨架字段"更精确：字段有、注入有、**写入没有**。

### 1.12 正文的时间跨度只有 10 章（约 1800 字摘要）
`internal/app/create_chapter_handler.go:761-797`：前文窗口 = 本章之前**最近 10 章** ×
单章摘要截断 **180 rune**；更早章节仅通过"语义记忆召回"（`recallStoryMemories`）兜底，
而该链路依赖 `hubSemanticStore` + 本地 embedder 可用，未启用时**整条静默失效**
（`story_memory_recall.go:107-157`）。→ 300 章的作品里，第 11 章之后模型实际上在
"无前文"状态下写作，长篇的一致性只是概率事件。

### 1.13 「情感曲线」读的字段生产端从不写入
`internal/app/visual.go:143-174` 的张力/情感曲线读 `ChapterSummary.EmotionTone`，
而 `create_chapter_handler.go:568` 落盘章节摘要时**只写 Title + Summary**；该字段恒空 →
曲线恒为默认值 5（`visual.go:115`），且该函数没有前端调用点。→ 面板上的"情感曲线"
是一条假直线：既没有数据，也没有人看。

### 1.14 章内多次调用之间没有衔接
字数续写最多 20 次（`create_chapter_handler.go:115`），每次只带**断点前 1500 rune**
（`:427-436`），不带本章已确立的伏笔、状态与场景目标。→ 8000 字以上章节的后段与前段
各自为政：重复叙述、语气断裂、埋了又忘——而章内本应是最容易保持一致的范围。

## 2. 机制设计（创作学 → 可机械化）

> 来源=方法论调研线（14 机制 / 25 条判据 / 三层架构），全文见该线交付物。
> **两条总原则**：①**确定性优先**——能用字段、计数、图算法、统计量判定的绝不交给 LLM；
> LLM 只做语义/意图判断且必须输出结构化字段 + 原文证据（便于抽查误报）。②**可写、可注入、
> 可回写**——一个机制若不能"立项写入 / 生成前注入 / 成文后回写"，就只是口号，降级为人工清单。
> 单一事实源：**先更新状态库，再生成下一单元**；正文与状态库冲突时以状态库为准并触发回填。

### 2.1 十四个机制（P0 十条，是"写得成"的底线）
| 机制 | 解决什么 | 可机械化的硬判据（示例） | 优先级 |
|---|---|---|---|
| M1 场景—续场双拍（Swain） | 流水账、中段塌陷 | 单元必须声明 `type∈{scene,sequel,montage}`；scene 必须有 goal/conflict/turnout（turnout 必须负向），sequel 必须 reaction/dilemma/decision 且 **decision = 下一场 goal** | P0 |
| M2 价值翻转与五律（McKee/Coyne） | 场景没有转弯 | 每场记 `value_at_stake` + 开闭电荷，要求**电荷发生变化**；各级（书/幕/章/场）可定位 crisis（≥2 选项）与 climax | P0 |
| M3 错误信念弧线（Weiland/Cron/Truby） | 动机漂移、成长不可见 | 主角 `want/need/misbelief` + 分段配额：前 1/3 ≥2 次错误行动（首次 ≤10%）、中段 ≥3 次反证且代价递增、终局由**行动**推翻（禁台词宣告）；水位单调 | P0 |
| M4 类型义务场景（Coyne/Snyder） | 结尾无力、类型期待落空 | 立项声明 genre → 载入义务场景清单，每项绑定 unit 或显式 `waived + 理由`；终局前扫未兑现 | P0 |
| M5 节拍网格与张力曲线（Snyder/Sanderson） | 张力平坦、高潮泄压 | 每单元 `tension(0-10)`；硬约束：每 N 单元至少一个局部峰、中段不得连续 K 单元低于均值、climax 为全局最大 | P0 |
| M6 线程台账与 MICE 配额（Kowal） | 支线遗忘 | 每线程 `mice_type/open/close/parent/status`；结局前全部 closed 或**显式** abandoned；嵌套默认 LIFO；`parked` 超时告警 | P0 |
| M7 伏笔池与回收契约 | 悬空伏笔、机械降神 | 每个 setup 记 `surface_form/promise_type/回收窗口/payoff`；终局段（最后 15%）仍有 promise 无 payoff 即阻断；**反向扫描**：climax 每个关键手段必须有前置 setup | P0 |
| M8 信息状态表与认知差 | 信息早泄、纸片反转 | 三张表（world_fact / 角色认知 / 读者认知）+ 每场 `information_delta`；问题计数终局前不归零；reveal 依赖的 fact 此前不得进过读者表 | P0 |
| M9 因果链与时间线台账 | episodic、时间线矛盾 | 每单元 ≥1 入边/出边 + 全图连通；时间线/位置冲突检测 | P0 |
| M14 冷读闸门与修订闭环 | 局部完美全局失效、累积漂移 | 结构闸→连续性闸→冷读闸（固定四问）；**同一状态库重跑同一规则差异必须为 0** | P0 |
| M10 场景去重与节拍指纹 | 重复场景 | 单元指纹相似度超阈值告警（阈值**按本书**标定）+ n-gram 重复率 | P1 |
| M11 设定与规则台账 | 规则漂移、魔幻解决 | canon 条目含成本/限制；任何解决手段可追溯到已确立规则；能力变化带"获得事件" | P1 |
| M12 主题论证与反论压力（Truby） | 主题漂浮、说教 | 控制理念 + 反理念；每幕反论有其最强陈述；climax 选择即论证的实验证明 | P2（**误报最高，只出报告不做闸门**） |
| M13 叙事声音与风格指纹 | 角色同声、AI 味 | 每角色 voice profile；统计量（句长分布/副词上限/感官动词下限/POV 越界） | P1 |

### 2.2 三层架构（级间只通过结构化契约通信，方向固定）
| 层 | 持有什么 | 生成时注入 | 写后回写与校验 |
|---|---|---|---|
| **故事层**（整书） | 控制理念与反论 · 弧线水位 · 线程表（MICE）· promise 池与回收窗口 · fact/认知表 · 因果 DAG · 时间线 · canon · 类型义务清单 · 张力曲线 | **只注入本场相关切片**：要触碰的 arc beat、要推进/收束的线程、可释放的 fact 子集、必须兑现的义务场景、禁止越界的 canon | 回写状态变更（水位/线程热度/promise/认知转移/时间线/新 canon），重跑 M3–M9、M11 与全局曲线 |
| **场景层**（场景卡） | 每单元 type/goal/conflict/turnout、reaction/dilemma/decision、价值电荷、五律定位、进出状态、指纹、tension、causes/effects | 编译上层义务为硬字段 → **缺字段/电荷未翻转/义务未绑定即拒绝生成** | 校验字段齐全、决策链闭合、因果边、指纹去重；把**实际发生**的事回写成下一单元输入 |
| **文本层**（句子） | voice profile · style sheet · POV 模式 · 禁用清单 · 本书统计基线 | 注入 voice profile + 本章节拍清单；约束句长/感官/副词/禁说教 | 回写统计量；**只能提案**新伏笔/新事实/新 canon（须场景层确认），不得直接改故事状态 |

**判据清单**：25 条"优秀长篇"机械判据中 **22 条可确定性自动化**（逐场状态变化、五律可定位、
电荷翻转、misbelief 配额、义务场景覆盖、张力曲线形状、指纹去重、伏笔回收、无降神、线程配对、
问题计数、因果连通、时间线自洽、规则不漂移、POV 越界、telling 密度、角色声音区分…）；
仅 3 条（主题可陈述、结尾不无力、冷读四问）需 LLM/人评，且**只出报告不入闸**。

## 3. 外部对标与失败模式

> 来源=工具调研线（一手文档抓取）。共性短板四条，正是本系统要补的洞：
> ①**记忆 ≠ 状态**（检索到的设定不等于"世界已变成什么样"）；②**没有承诺—偿还契约**
> （伏笔/支线/义务场景只能靠人记）；③**校验在事后或人工**，无法阻断生成；
> ④**一致性靠上下文长度而非不可变账本**，篇幅越长漂移越大。

- **Novelcrafter**：强在 Codex（场景级按"正文提及"自动挑选相关条目注入，四档 AI Context 开关
  + 排除词表防假阳性）与"当前 beat 之前所有场景摘要进 system prompt"；弱在无结构机器判据，
  且 Act/Book 摘要刻意不给 AI（防剧透）。
- **Sudowrite**：强在 Story Bible 工作流 + Chapter Continuity（≤25 章 / 20,000 词）；弱在
  20k 词硬顶 ≈ 3–4 万汉字，Brainstorm 不读 Story Bible。
- **Plottr**：强在多情节线时间线 + 30 个结构模板（每个节拍预置"这一幕该完成什么"）；**完全不能写**。
- **Scrivener**：强在 Snapshot（单条目保存点/回滚/**两快照 diff**）与编译重组；无 AI。
- **Fictionary / Story Grid**：把编辑方法论做成场景表（进出钩子/POV/价值转变/字数）与
  15 条 insight + 弧线对比图 —— **只诊断不治疗**，且公认短板是**无支线追踪**、无角色弧可视化、
  手工录入 6–10 小时、推荐弧线是模板（结构创新作品会被判"错"）。
- **NovelAI / AI Dungeon（反例最有用）**：Lorebook 的激活键/插入位置/token 预算/级联
  （手工版"上下文分页"）值得抄；但 **AI Dungeon 官方自陈**："AI 在几千 token 后忘记你的选择，
  那些选择就毫无意义"。其 Required/Dynamic 70% 规则会在超预算时**整体丢弃**低优先级条目——
  长篇失忆的根因不是检索质量，而是**固定预算 + 优先级截断 + 条目没有"全局状态"概念**。
- **学术侧（已被验证有效）**：Re3（结构化 plan + 注入 story state + 多候选 rerank + 事实一致性
  编辑）、DOC（**更细的层级大纲 + 生成时对齐**，人类评估显著优于 Re3）、CONCOCT（vaguest-first
  扩展求均匀节奏）、LongStory（长短期上下文权重 + **结构位置标记 LSP**）、IS-CoT（>2000 词出现
  "长度坍缩"，应做 **Plan–Write–Reflect 内嵌循环**且粒度到 ~500 词级）、Lost in the Middle
  （信息位置呈 U 形：**加长窗口不能替代结构化记忆**）。
- **明确排除**：把全稿塞进单次调用（位置偏差 + 成本）、纯 RAG 片段检索（拿不回"已建立的事实"）、
  一次生成整章再重写（治不了规划缺失）、只靠提示词堆设定（预算一到就被静默切掉）、
  云端向量库（与本地优先冲突）。
- **本地优先的取舍**：放弃"更长上下文"路线，把价值押在**外部化状态 + 可审计的上下文预算**
  （预留 + 插入位置 + 优先级截断）；并且**所有 AI 行为必须可见**（哪条进了上下文、为什么、
  占多少 token），否则用户无法调试长篇一致性。

## 4. 评测闭环

> 来源=评测线（设计全文见其交付物）。原则：**单点取值**（每指标只有一个实现入口）·
> **四级单元**（段→章→幕/卷→全书，上层只聚合下层、不重读正文）· **增量优先**（迭代只看 Δ）·
> **双通道不混算**（确定性可阻断／LLM 只报警）· **每指标必须绑一个补丁动作**，否则不进 MVP。
> 指标点必带元数据：`metricId · unit · confidence · source · algoVersion · evidenceSpan`。

- **三层指标**：文本层（AI 味逐段子分 + 全书 P90 · 重复度 SimHash/5-gram >0.12 · 语域漂移
  Delta · 对话比与章间波动 ±0.15 · 段落匀称度 >0.6 可疑 · 口头禅跨章）；场景/章层（场景目标
  达成 · 转折发生度（resolution_progress 增量 + 极性翻转）· 退出钩子强度 · 状态变化量
  ΔC/ΔRel/ΔOrg（连续 3 章为 0 即停滞）· 信息释放 · 张力 T = 0.3·conflict + 0.3·emotion
  + 0.2·Σimportance + 0.2·钩子）；**整书层**（结构完整性/幕长带 25-50-25±10% · 弧线推进曲线 ·
  张力曲线形状：峰应落 75%~95%、谷深 < 均值×0.5 且长 >8 章即**塌陷** · 伏笔回收率与 resolveLag
  分布 · 支线开关完整性（G=12 章）· 重复场景聚类余弦>0.85 · 结尾收束度 · 主题论证需 3 支持
  事件 + 3 反例以压谄媚）。
- **回归集**：自建夹具 3 本 × 30 章 × 2500 字（≈22.5 万字），**人工植入"标准答案病历"**
  （3 塌陷区 + 2 遗忘支线 + 4 不回收伏笔）；公开文本只用公版（当代网文授权不明即不可用）。
  基线必带 `promptSetHash`（prompts/*.json 的 sha256）+ 模型/温度/大纲/角色库哈希，
  **任一哈希变更即 stale，禁止直接对比**。
- **A/B**：同章号配对 + bootstrap 置信区间（小样本禁正态近似）；区间跨 0 判"证据不足"而非
  "无改进"。防作弊：200 段人工校准集（ρ≥0.5 才允许改指标）+ 盲评成对比较 + 反指标
  （TTR/段长匀称度/对话比漂移/情绪词密度暴涨）联动标 suspect。
- **收敛判据（防无限重写）**：章节级 = S1=0 且 S2=0 且 S3<3，或仅剩 S3/S4；章长变化 ≤±15%
  且 Delta 不劣化；**最多 2 轮自动重写，第 3 轮必须换机制或交人**；任一轮问题数不降立即停。
- **作者是上帝边界**：必须人批=情节/角色命运/主题取向/结局/支线取舍/连续 3 轮不收敛/
  AI 味与审美冲突（作者胜）；可自动=确定性指标计算与定位、S1/S2 机械修补（段落/句子/用词/
  记账）、伏笔超期提醒与延后记账、快照与跑批。
- **MVP 三件事（约 6 人日，实施顺序 3→1→2）**：③ 回归跑批 + `baseline.json` + 成对盲评表
  （**先建可比性**）→ ① 全书张力曲线 + 塌陷区定位（纯聚合，零新增 LLM 成本）→
  ② 伏笔回收率/位置分布 + 支线开关完整性（复用 ForeshadowLintReport）。验收线：张力曲线
  须把夹具里植入的 3 处塌陷定位到 ±1 章；伏笔回收率与手工核对一致；同一夹具两次
  `baseline.json` 除时间戳外逐字节相同。

## 5. 刀序（草案，待拍板）

| 刀 | 范围 | 为什么这个顺序 | 验收要点 |
|----|------|----------------|----------|
| 刀1（P0） | **章节计划闭环**：`plans.json` 落盘 + AI 提计划草案（作者审批）+ 写前硬闸 + 生成注入 + 写完后的偏差回写 | 契约已有零消费，是"从逐章生成到结构化创作"的最小杠杆；不动生成算法就能立刻提升意图信息量 | 计划七字段齐备可审批落盘；生成 prompt 含计划区段；缺计划时硬闸拒绝（可显式覆盖）；写后产出"计划达成/偏差 + 下一章草案" |
| 刀2（P0） | **场景卡与场景级生成**：`SceneMeta` 扩 goal/conflict/turn/outcome/sequel/exit_hook；章节按场景卡逐场景生成与落盘；解禁场景级重写 | 1.3/1.4 的正面解 | 一章多场景而非单场景物化；场景卡字段参与生成；单场景可重写且不影响他场 |
| 刀3（P0） | **故事层骨架**：主题论证 + 人物弧线节点 + 节拍位 + 支线开关 + 未解问题池；每 10 章结构体检 | 1.10 的正面解；刀1/2 的约束来源 | 骨架可编辑并注入生成；结构体检能定位"中段塌陷/支线遗忘"并给可执行动作 |
| 刀4（P1） | **质量收敛闭环**：体检 → 定向修补 → 复检 → 收敛判据 | 1.7 的正面解 | 给定失败维度自动修补并复检；达判据即停，报告修补轨迹 |
| 刀5（P1） | **风格学习回灌**：从作者手改成稿学指纹/句法偏好，生成时约束 | 1.9 的正面解 | 生成时按学习档约束；对照实验可量化风格距离下降 |
| 刀6（P2） | **上下文编译器**：总预算 + 优先级 + 去重 + 相关性 | 1.8 的正面解 | 同样信息量下 prompt 更短；结构信息不再被整篇设定挤掉 |
| 刀7（P2） | **评测基线**：指标快照 + 回归集 + A/B | 让前六刀"变好"可证 | 每次机制变更产出可比快照 |

## 6. 明确不做

- 不做"一键写完一本"：作者是上帝，系统负责结构、情报与校验，不替作者决定故事。
- 不用"续写凑字数"（`maxContinues`）当作篇幅手段：篇幅应由场景齐备度决定。
- 不引入云端向量库 / 多智能体平台化 / 外部写作 SaaS 依赖（本地优先不变）。
- 不推翻既有资产：ChapterPlan / SceneMeta / analysis-v2 / novelstyle / novelgate 全部复用扩展。

## 7. 第一刀（故事脊椎 + 章节计划闭环）——契约与验收

> 形态=**先修"意图为零"，再建"意图载体"**。本刀不改生成算法、不做场景级拆分，
> 只把"这一章为什么存在"从「一句自由文本」变成「主线 + 计划 + 大纲要点」，并接上
> 写前硬闸与写后偏差回写。选它先行是因为它是刀2/刀3 的承载面（场景卡挂在计划下，
> 结构骨架通过计划落地）。

### 7.1 数据契约

1. **修复 `StoryThread` 写入路径**（§1.11）：`internal/outline/outline.go` 的
   `Continue`/`ExpandNode` 解析模型回复时**同时回收 `story_thread`**
   （当前只回收 `.Nodes`，`:186-207`），随 `OutlineFile` 持久化；`novelcontext` 既有
   注入点立即生效（零新绑定、零前端改动）。验收：规划一次五卷后 `outline.json` 内
   `story_thread` 非空，且下一次章节生成的 SceneBible 里出现主线区段。
2. **`chapters/plans.json`**（复用 `types.ChapterPlan` 七字段，不新建数据模型）：
   `{"version":1,"plans":{"12":{…}}}`；缺失=空正常态；**损坏文件不覆盖**并如实报错；
   写入走 `fileutil.AtomicWrite`（与残稿侧车同款）。
3. **计划齐备性判据**（硬闸输入，确定性）：`NarrativeGoal` · `KeyEvents(≥2)` ·
   `ConflictType` · `EndingType` · `EmotionalTone` · `CharacterFocus(≥1)` 六项齐备即可生成；
   缺任一即硬闸。`KeyEvents` **跨章去重**：与全库 `plans.json` 及已写章节的 `key_events`
   比对，重复即点名失败（对齐 `types.ChapterPlan` 注释"跨章不得重复"）。

### 7.2 绑定（NovelB 门面；统一由主代理跑 `gen_bindings` + drift 锁 + spaceBindings）

| 绑定 | 语义 |
|---|---|
| `NovelChapterPlanGet(chapterNum)` | 读该章计划（不存在返回空，不报错） |
| `NovelChapterPlanSave(chapterNum, planJSON)` | **审批落盘**：齐备性 + 跨章去重校验，非法拒绝并点名 |
| `NovelChapterPlanPropose(chapterNum)` | AI 草案：大纲节点 + 故事主线 + 前文窗口 + 已写 `key_events` + 该章 analysis-v2 载荷；产出七字段（不落盘） |
| `NovelChapterPlanDeviation(chapterNum)` | 计划 vs 实际：未达成关键事件 / 结尾类型不符 / 情绪漂移 / 跨章重复 |
| `NovelChapterGatePrecheck(chapterNum)` | 生成前预检：计划齐备性 + 大纲契约 + 是否允许生成（硬闸唯一判据来源） |

### 7.3 生成链改造

- `prompts/create-chapter.json` 新增两槽位：`chapter_plan`（P0，计划区段）与
  `outline_points`（大纲节点 `KeyPoints`/`Emotion`——**此前零进入 prompt**，§1.2 证据）。
- `internal/app/create_chapter_handler.go`：生成前调预检；不通过即拒绝并返回可执行入口
  （`AllowOverride` 显式覆盖，默认关）；通过则注入两个新区段，纳入既有预算体系
  （`ctxBudgetTotal` 口径，不与 SceneBible 重复堆叠）。
- 硬闸只拦「没有抓手」，**不评判计划质量**（质量仍由写后门 + 作者判断）。
- **护栏（E2E 取证发现）**：生成收尾把解析到的章节摘要写回大纲节点时，**摘要为空/纯空白不得覆盖**既有 Summary。否则模型漏输出 `---CHAPTER_SUMMARY---` 会清空节点摘要，而同一章的下一次预检会被 `outline_summary_empty`(S2) 拦下——硬闸把原本的显示问题放大成"自锁"，且报错指向错误原因（同款纪律：线A 的 `mergeStoryThread` 非空才覆盖）。

### 7.4 写后回写

章节分析（analysis-v2）完成后计算计划偏差，落 `analysis/plan-deviation/<MMM>.json`
（缺失空态、损坏不覆盖），并在面板给「下一章计划草案」建议——**不自动落盘**（审批制）。

### 7.5 分线（足迹互斥，主代理收口）

| 线 | 负责人 | 足迹 |
|---|---|---|
| 线A 契约与存储 | 子代理 | `internal/types/*`（PlanFile/PlanDeviation/PlanGateReport 契约）· `internal/project/*`（plans.json / deviation 读写）· `internal/outline/*`（StoryThread 回收） |
| 线B 生成链与硬闸 | 子代理 | `internal/app/create_chapter_handler.go` · `internal/novelgate/*`（PlanContractIssues/跨章去重）· `prompts/create-chapter.json` |
| 线C 计划生产与偏差 | 子代理 | 新 `internal/app/novel_plan_handler.go`（Get/Save/Propose/Deviation/Precheck）· `internal/app/bindings_novel.go` 委托 |
| 线D 前端计划卡 | 子代理 | 新 `frontend/src/components/novel/ChapterPlanCard.tsx` · `pages/CreatePage.tsx` 挂载与硬闸提示 |
| 契约面（生成物） | 主代理 | `frontend/src/wailsjs/go/app/NovelB.*`（gen_bindings）· `bindingNames.ts` · `spaceBindings` 锁 · `frontend/src/gaea/lib/bridge/novel.ts` · `mock/novel.ts` |

### 7.6 验收

1. **Go**：`plans.json` 往返 / 缺失 / 损坏三态；齐备性矩阵（六项缺一即拒）；`key_events`
   跨章重复点名；`StoryThread` 落盘往返；硬闸在**生成前**生效（缺计划时 CreateChapter 拒绝、
   `AllowOverride` 放行）；偏差报告四类判据各一例。
2. **前端**：计划卡显示/编辑/生成草案/审批；缺计划时生成入口给硬闸提示 + 「一键补计划」；
   偏差报告可见；vitest 小说域既有测试零回归。
3. **契约面**：绑定 +5 走 `gen_bindings`，drift 锁与 `spaceBindings` 同步，dev mock 诚实降级
   （草案如实抛错而非假装成功）。
4. **门禁与文档**：`scripts/ci.ps1` 全量绿；漂移闸 OK@新版本号；规格/CHANGELOG/README/
   releases/AGENTS/progress 收口。
5. **可复现证据**：用一本 3 章夹具演示「计划缺失 → 硬闸拒绝 → 生成草案 → 审批 → 生成 →
   偏差报告」全链，并留下命令与输出。

### 7.7 本刀明确不做

场景卡与逐场景生成（刀2）· 结构节拍/弧线/支线表（刀3）· 质量自动收敛（刀4）·
风格学习回灌（刀5）· 上下文编译器重构（刀6）· 评测基线（刀7）。

**已知缺口（E2E 取证，记观察池）**：跨章去重的第二来源未接——`PlanContractIssues` 只看
`[]types.ChapterPlan`，而「已写但没有计划条目」的旧章其 `KeyEvents`（在
`ChapterSummary` 里）不参与去重；根因是 `ChapterSummary` 无章号字段、
`ReadAllChapterSummaries()` 只按文件名排序、无法可靠映射章号。本刀不猜章号糊过去：
新建章一律经硬闸先建计划，缺口只影响"存量旧章 + 手改 plans.json"两种边角。
正确修法是给 `internal/project` 加带章号的批量摘要读接口（或扩展判据入参），随刀2 一起做。

**E2E 取证**：`internal/app/plan_loop_e2e_test.go::TestPlanLoopE2E_PlanMissingToDeviation`
（3 章夹具 + 单 SSE 假 LLM 双链）跑通完整证据链——缺计划被拦且**模型调用 0 次** →
草案不落盘 → 审批落盘 → 预检放行 → 生成落章（prompt 含「本章计划」区段）→ 偏差四判据
（未达成事件 / 结尾不符 / 情绪漂移 / 跨章重复）+ summary + nextSuggestion + 偏差落盘回读。
