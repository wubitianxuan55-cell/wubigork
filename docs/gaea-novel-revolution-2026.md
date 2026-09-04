# gaea 小说板块 · 下一步架构（终版设计 · 代码实测 + 市场实测）

> 定位：roadmap §5（小说）× §13（创作间）×「上下文工程取代 prompt 工程 / 记忆即操作系统」
> 三条主线的落地。本文是**代码实测 + 市场实测 + 可动刀架构 + 分阶段可回退执行计划**的
> 终版；取代此前基于「愿景当现状」的错误稿（那版把 POV/场景当空白——实测已存在）。
>
> 生成：2026。完成度：前端+后端逐文件审计完毕；市场/社区/开源竞品逐家调研完毕。
> 硬数据来自马良写作（20 万字后设定矛盾率 83%→23%）、anti-ai-polish、anti-ai-checklist。

## 落地状态（每完成一刀勾一刀，随迭代更新）

- [x] **刀 3 · 确定性 AI 味引擎**（`internal/novelstyle`）：`StyleFingerprint` + `AiTasteScorer`(0-100 + rune span) + 定点改写规则。5 测试全绿；`Delta(same)=0.29 vs diff=0.87`。
- [x] **刀 2 · 场景圣经编译器**（`internal/novelcontext`）：`CompileSceneBible`/`BuildSceneBibleFromChapter`/`Render`，按 `SceneMeta.POVCharID` 做视角掩码（公开可见/秘密进 HiddenFacts）+ 子图检索 + 未回收伏笔 + 时间锚点 + 文风。4 测试全绿，**POV 掩码核心测试通过**。
- [x] **刀 6 · 叙事状态机 + 审批账本**（`internal/narrative`）：`ApplyPatch`(纯函数) + `ValidateStatePatch` + `AuthorizeAndSettle`（`approved=false` 不入账本）+ append-only 可回放。5 测试全绿。
- [x] **接入生成管线**（`internal/app/create_chapter_handler.go`）：`CreateChapter` 注入 POV 场景圣经；`done` 事件携带 `novelstyle` AI 味分。`go build ./...` + 既有章节测试通过。
- [x] **前端 AI 味反馈**（`CreatePage.tsx` + `chapterStreamTypes.ts`）：生成完成展示 `aiTaste` 分数 + 命中问题。vitest 14/14 + `tsc -b` 通过。
- [x] **刀 5 · 确定性定点重写引擎**（`internal/novelstyle/rewrite.go`）：`DeSlopRewrite`（AI 高频词→平实替代表 + 标点归一，只改命中词、不碰剧情/篇幅）+ 接入生成管线——`story-deslop` 启用时生成即去味，分数与`RewriteReport`入 `done` 事件。8/8 测试绿（分数下降/干净文本保留/标点归一）。
- [x] **刀 1 · 接通生成→场景**（`internal/app/scene_gen_handler.go`）：新增 `CreateScene` + `GenerateScene` 绑定（POV 感知逐场景生成，`novelcontext.CompileSceneBible` 场景圣经注入，落 `scene.Manager`）；绑定面 561→566，`bindingNames.ts` 重生成 + bridge `LegacySurfaceNames` 同步，drift PASS。
- [x] **刀 6 · 叙事状态审批制结算**（`internal/app/novel_state_handler.go`）：新增 `GetNovelState`/`BuildNovelStatePatch`/`SettleNovelState` 绑定，走 `narrative.AuthorizeAndSettle`（`approved=false` 不入账本）；绑定面 +bridge 同步，drift PASS。
- [x] **刀 5 续 · 手动「一键去味」**：新增 `DeSlopChapterAiTaste` 绑定（对**任意已有章节**做确定性 `DeSlopRewrite`，v4 逐场景 / v3 整章，零 LLM）；绑定面 566→567，`bindingNames`/bridge/drift 同步；前端 CreatePage「一键去味」按钮 + 去味结果反馈。
- [x] **刀 5 再续 · LLM 受限重写**：新增 `RewriteChapterAiTaste` 绑定（打分→定位命中句→LLM 批量受限重写→安全替换→**复测分数下降才落盘**的安全闸→v4 逐场景/v3 整章）；绑定面 567→568，`bindingNames`/bridge/drift 同步；前端 CreatePage「高级去味」按钮。纯函数测试 `splitSentences`/`pickFlaggedSentences` 通过（模型路径由编译+安全闸逻辑保障，离线无法端到端）。
- [ ] `GenerationGate` 完整闭环（生成后自动 Analyze/Review/Consistency + AI 味 + 修复 + 复检的编排）。
- [x] **刀 7 · 前端（执行部分）**：AI 味分 + 命中问题展示（`CreatePage`）；**叙事状态账本面板 + 「AI 生成状态建议」+「批准结算」审批 Modal**（调用 `GetNovelState/BuildNovelStatePatch/SettleNovelState`）。vitest 14/14 + `tsc -b` 0 + eslint 0。
- [ ] 刀 7 续 · 前端：场景生成按钮（`ChapterEditor` 逐场景 `GenerateScene`，需加场景 ID 追踪）、全文脑图、文风指纹面板、POV 视图。
- [ ] 刀 8 · 跨模块验收 + 回退演练 + `go test -race` 门禁（本环境 `CGO_ENABLED=0` 无法跑 `-race`，CI 侧另配）。

---

## 0. 一句话结论（读前记得这一句）

> **下一代不是「再造小说工具」，而是把 gaea 已经建好、但没接上的「v4 场景制 + 实体库 +
> 状态卡 + 审阅」，用一根主轴串成一台连续运转的创作机：**
> **场景驱动生成 → 确定性 AI 味/一致性闸门 → 定点修复 → 作者审批 → 状态结算回写。**
> 差异牌 = ① 本地优先/数据不出机 ② 作者是上帝（AI 有建议权、审批制）③ POV 视角掩码。

---

## 1. 现状：gaea 小说板块完整架构图（逐文件实测）

### 1.1 数据层（`internal/types` + `internal/project`）——已经很完整

- **一部小说 = 一个文件夹**。`project.json` + `worldview.json`（固定 6 维 era/geography/factions/rules/culture/history）+ `characters.json` + `outline.json`（卷→章→场景树）+ `foreshadows.json` + `lorebook.json` + `branches.json`（剧情分支 sidecar）+ `.gaea/v4` 标记。
- **v4 场景制（关键，一直被忽视）**：每章 = `chapters/NNN/scenes/MMM-slug.md + .meta.json`。`SceneMeta` **已含 `POVCharID / Location / TimeOfDay / Emotion / Tags(climax·action·dialogue) / Order / Status / WordCount`**。`scene.Manager.Stitch()` 拼装，`snapshot.Store` 行级 diff 版本 + 回滚（`trigger=manual/ai-rewrite/ai-generate`）。`OutlineNode` 已含 `SceneIdeas/SceneRefs/KeyPoints/Emotion/Branch`。
- 原子写（`writeFileAtomic`）贯穿，用户数据不会写坏。

### 1.2 生成层（`internal/app/create_chapter_handler.go`）——落差所在

- `CreateChapter(setting, prevSummary, plotReq, chapterNum, branchFromNodeID, skillName, minWords, temperature)` 注入 `setting + characters + prev_summary(每章截断200字) + 未回收伏笔 + 世界观要点`（`buildChapterContextSections`）**扁平拼装**，受 `ctxBudgetTotal` 预算约束。
- `streamCreateChapter` 流式 + 字数续写（≤20 次），**最后 `pm.WriteChapter` 写 `chapters/NNN.md`（v3 blob）**——不拆场景、不读 `SceneMeta`。

### 1.3 编排层（5 个 agent，全是 prompt 包装器，零互编）

| Agent | 文件 | 实测 |
|---|---|---|
| 世界观 | `worldview.go` | 分维 CRUD + 对话 + `ChatWithAutoSave`（`---WORLDVIEW_SECTION_UPDATE---`/` ```markdown ` 提取） |
| 角色 | `character.go` | 批量/单个生成 + 去重合并 + 剧照补全/落盘(`portraits/`) + 组织/关系 CRUD + `---CHARACTER_UPDATE---` 提取 |
| 大纲 | `outline.go` | `Chat`/`ChatNode`/`Continue`（count==5 全量替换+旧ID过滤 / 否则追加去重）/`ExpandNode`/CRUD + `GenerateOutlineWithDialogue`（学生-专家，蒸馏 MM-StoryAgent） |
| 章节 | `chapter.go` | `GenerateSummary` + `GenerateSceneIllustration`(Aurora 图) + `ReviewChapter`(**蒸馏 MM-StoryAgent**，`{score,strengths,weaknesses,revise_plan}`，`reviewChapterFallback` prompt 已写「是否含 AI 套话」，**恒走 fallback**——因为 `prompts/` 无 `chapter-review.json`） |
| 分析 | `analysis.go` | 9 维 + `syncForeshadows`(stable_id 合并，manual_ 不被冲掉) + `syncCharacterStates` + `ReviewBook` + `AggregateBookData` |

### 1.4 审阅/校验层（最强，但全是旁路）

- **9 维分析**：hook/foreshadows/conflict/emotion_curve/character_states/key_events/scene_rhythm/quality_score/improvement_tips。
- **规则一致性**（`graph.CheckConsistency`）：**正则，弱**（角色眼睛/头发/状态词）。
- **AI 状态卡深检**（`consistency_deep_handler.go`，强）：`deepStateCard{time_mark/time_relation/scene_notes/travel_notes/characters[status·location·items]/items_lost/items_regained}`，`deepCompareStateLine` 检出时间倒流/死亡再出场/位置瞬移/物品凭空消失/无中生有，纯函数可测。
- **实体库**（`graph/entity.go`）：角色/地点/物品/组织/概念/事件 + 关系边 + `[[wiki-link]]` 反链 + `SyncFromProject`。

### 1.5 前端（`NovelPage` 五 Tab）

| Tab | 组件 | 实测 |
|---|---|---|
| 创作 | `CreatePage` | `CreateChapter`+`useChapterStream`；右 `CreateInspector`：设定预览/**写作技能下拉（默认 `story-deslop`）**/目标字数 1000-20000/温度/剧情方向 wizard+直接生成/统计 |
| 阅读 | `ChapterPage` | **场景多文本框**：`ChapterEditor`（增删场景、右键 AI 操作「丰富描写/场景扩展」、Cmd+K「展示代替讲述」、GhostText） |
| 设定/角色/书架 | `NovelSettingPage`/`CharacterPage`/`HomePage` | 分维编辑 / 角色·组织·关系·剧照 / 书架 |

### 1.6 新确认的死角（比上一轮更狠）

- **绑定面只有 `SaveScene/ReorderScenes/GetChapterScenes/CreateSnapshot/ListSnapshots/RestoreSnapshot`，没有 `CreateScene` / 场景级生成**。v4 场景只在 `MigrateV3ToV4` 时被创建；`CreateChapter` 生成的新章写的是 blob，**没有场景**。`OutlineNode.SceneRefs` 基本是死字段。
- **`prompts/` 无 `chapter-review.json`** → `ReviewChapter` 恒走内置 fallback。

---

## 2. 真正的结构问题（6 个，全有代码证据）

1. **生成写 blob、编辑读场景**：`streamCreateChapter→WriteChapter→chapters/NNN.md`；`GetChapterScenes` 才读场景。生成不拆、不用 `SceneMeta`、不产场景。
2. **5 个 agent 零编排**：没有「导演」按 设定→人设→大纲→场景→草稿→校验→结算 串起来；`GenerateOutlineWithDialogue`/`ReviewChapter` 等「蒸馏自 MM-StoryAgent」的高阶能力散落、互不调用。
3. **审阅是旁路、从不回灌**：`Analyze/Review/CheckConsistency` 的结论（`revise_plan`/`quality_score`/`improvement_tips`/issue）从不自动重写、从不自动结算。
4. **上下文不按场景/POV 编译**：`ProjectContext`+`BuildRichContext/InjectMemories/BuildContextBudget` 都在，但 `CreateChapter` 用的是扁平 `setting+chars+prev_summary`。没按「本场景实体+各自状态+POV 知识视图+时间锚点+未回收伏笔+文风指纹」检索。
5. **AI 味 = LLM 启发式 + 静态黑名单**：`story-deslop` 是写进 prompt 的规则文本；`reviewChapterFallback` 让 LLM 主观判断。无确定性分数/span 定位/定点重写。
6. **一致性/状态只有「读」没有「写账本」**：`EntityDB`+`deepStateCard` 只喂审计；生成前不注入、生成后不结算、作者不审批。

---

## 3. 市场/社区/开源竞品（这次实测 + 硬数据）

### 3.1 国际商业长文工具 = 全都在做「持久世界记忆」
- **Sudowrite** Story Bible、**NovelCrafter** Codex（实体卡+片段）+ 上下文预算、**NovelAI** Lorebook。[Jenova 评测](https://www.jenova.ai/en/resources/which-ai-novel-writing-assistant-remembers-character-arcs-timelines-and-worldbuilding-best)标题即「谁最记得角色弧线/时间线/世界观」。**NovelCrafter 的「AI-assisted authoring + Codex 实体注入 + 上下文预算」与 gaea 的 v4 场景制几乎同构**——方向已被验证，gaea 有本地+审批差异化。

### 3.2 中文平台（真正的对手）——硬数据
- **马良写作**（[百万字不烂尾](https://maliangwriter.com/blog/ai-long-novel-consistency-guide/)）：**不建知识图谱，20 万字后设定矛盾率 83%；建了降到 23%**（实测平台数据）。三武器 = **知识图谱**（自动提实体/状态/物品/地点/已揭示信息）+ **活大纲**（总纲稳、卷纲每卷重写、章蓝图每章微调）+ **角色状态时间线**；另单卖 **AI 味浓度检测 / 去 AI 味改写 / 50 万字后设定审计**（[对位 Sudowrite/NovelCrafter/笔灵](https://maliangwriter.com/blog/)）。
- **阅文作家助手·妙笔**（千万字理解、平台绑定）、**番茄作家助手**（AI 检测/风险）、**笔灵 AI 写作**、Coze [长篇小说AI工作室](https://xiaping.coze.com/skill/58e8e145-dc1d-4508-a0ff-1a636c2dda36)/[降AI检测写作助手](https://xiaping.coze.com/skill/601aa22d-2499-49a4-862f-3cc19d0e7b7b)。

### 3.3 社区反 AI 味（可编码成确定性规则）
- [anti-ai-checklist](https://github.com/tance-mang/chinese-webnovel-skills/blob/main/references/anti-ai-checklist.md) 四条：① 一段至多一比喻、删"为好看"排比叠喻、用动作→细节；② **语域一致**（先定语域：通俗文别引典故、古风别用现代梗、叙述者认知边界）；③ **show don't tell**（情绪词→动作+对话，虐点克制）；④ 大白话短句。
- [anti-ai-polish 阈值](https://raw.githubusercontent.com/DankerMu/novel-writer-cli/main/docs/anti-ai-polish.md)：四字成语≤3/500字且连用≥2违禁、强调词≤2/300字、形容词≤6/300字、段落20-100字、单句段落占比、省略号≤1/段≤5/章、感叹号≤1/段≤8/章、句长 std 偏小预警、AI 高频词命中。
- 统计爆发性：句长 CV / TTR / 逗号密度（[yotta-humanize](https://github.com/YottaMeta/yotta-humanize) 已验证可零依赖做 0-100 分）。

### 3.4 开源侧（GitHub，已核验 star 数/时间）
- 一致性：graphify-novel（图检索注入）、hzhang092/novel-agent（canon facts+状态卡+审批后入库）、xiehuanyi/NovelForge（6 agent+三层记忆）、museflow（10 agent+逐章定点修复）、loom-novel（外置大脑+写作指纹，**只学用户手改不学 AI 输出**）、blackzhanzhan/novel_agent（LoreGit）。
- 长文：LongWriter-AgentWrite（先大纲再扩写，ICLR 2025）、RecurrentGPT（摘要循环）、chinese-webnovel-skills。
- AI 味：`detect-gpt`/fast-detect-gpt/HC3 数据集（中文）；yotta-humanize（确定性检测器，1 star）。
- **共同缺口 = 我们的机会**：**POV 视角掩码**（「人物此刻知道什么」的实体级知识隔离）开源几乎无人做扎实。**这个 gaea 的 `SceneMeta.POVCharID` 已有字段，接通即拉开差距。**

---

## 4. 下一代架构（可动刀图 + 每个断点的接法）

```
作者意图/大纲节点
   │
   ▼ ①CompileSceneBible(pm, scene)   ← 新包 internal/novelcontext
   │   按 POVCharID 裁剪：只注入本场景实体+各自状态+pov知识视图+未回收伏笔+时间锚点+文风指纹
   ▼ ②(逐场景) DraftScene(scene)      ← 用 SceneMeta(POV/Location/Time/Emotion/Tags) 驱动
   ▼ ③Gate {                          ← 生成后自动跑，非旁路
   │    ConsistencyGate (graph规则 + deepStateCard + POV泄漏检测)
   │    AiTasteGate  (internal/novelstyle 确定性 0-100 + span定位)
   │    QualityGate  (analysis.Analyze / chapterAgent.ReviewChapter)
   │  } ─未过→ ④DeSlopLoop: 只重写被判罚 span/节拍 → 复检(预算上限, 断点可恢复)
   ▼ ⑤AuthorApprove                    ← 作者是上帝：AI 只给建议，不写状态
   ▼ ⑥SettleState                      ← append-only journal：结算回写 characters状态/EntityDB/伏笔/文风指纹(只学作者手改)
   └→ 环形：下一场景/下一章
```

### 4.1 每个断点对应到真实代码

| 断点 | 现状 | 改动 |
|---|---|---|
| ① 场景圣经 | `CreateChapter` 扁平拼装 | 新增 `CompileSceneBible(pm, scene)`；读取 `SceneMeta.POVCharID`，注入该场景相关实体+状态+**POV 知识视图** |
| ② 场景级生成 | `streamCreateChapter` 写 blob | 改成逐场景生成，`scene.Manager.Write` 落场景，`Stitch` 成章；补 `CreateScene` 绑定 |
| ③ 生成后自动闸门 | `Analyze/Review/CheckConsistency` 手动 | 新增 `GenerationGate`，生成完成回调自动跑；`reviewChapterFallback` 的「AI 套话」判断抽成确定性 `AiTasteScorer` |
| ④ 定点修复 | 无 | `DeSlopLoop`：只重写命中 span；纯算法节奏重排(长切短/长短交错/打散连续段)+词级替换+句级受限重写 |
| ⑤ 作者审批 | 无审批 | 复用既有 `hardAskTools`/`dream-audit` 纪律，状态 patch 过 schema 校验 + 作者审批 |
| ⑥ 状态结算 | `EntityDB/状态卡` 只读 | `SettleState`：逐章 diff 状态，回写 characters/EntityDB/伏笔/文风指纹 |

### 4.2 确定性 AI 味引擎（`internal/novelstyle`，新增，全部可测）

- **`StyleFingerprint`**（确定性，只学作者手改）：`function_word_vec(z 化 30-60 词) + Burrows Delta + 句长{mean,sd,p10,p90,long_tail} + 段长{mean,sd,short_para_ratio} + ttr_1000 + hapax + 词频熵 + dialog_ratio + four_char_ratio + connective_density + adj_adv_density + punctuation{ellipsis,excl,dash,comma_per_sentence} + top_bigrams/trigrams + author_signature_words`。**只从 `ChapterPage` 场景编辑/CmdK 的用户改动蒸馏，绝不用 AI 输出。**
- **`AiTasteScorer`**（0-100 + 逐 span 定位）：anti-ai-checklist 四条 + anti-ai-polish 字数量化阈值 + 统计爆发性（句长 CV/TTR/逗号密度）。每个命中返回 `{span, reason, severity, fix_suggestion}`。
- **`DeSlopLoop`**：整章打分→定位→仅重写 flagged span（词级替换/句级重写/节奏重排/加身体反应）→diff 回填→回归复测（分数降、字数/实体/剧情不变）。

### 4.3 POV 视角掩码（差异化，一期做布尔简化版）

- `SceneMeta.POVCharID` 已存在。给实体状态（或 `graph.EntityDB`）加**布尔知识视图 `known_by`**；`CompileSceneBible` 按 `POVCharID` 只注入该角色知情信息；`ConsistencyGate` 加 **POV 泄漏检测**（A 角色说出其不知情的秘密/上帝视角/剧透）。
- 开源几乎没人做，一期用「布尔知识视图 + 泄漏检测」，验证价值后再加深（不做过度复杂图谱嵌入）。

---

## 5. 分阶段执行计划（每刀独立提交、可回退、可验收）

| 刀 | 内容 | 新文件/复用 | 验收 |
|---|---|---|---|
| **1** | 接通生成→场景：`streamCreateChapter` 改 `scene.Manager`，逐场景生成；补 `CreateScene` 绑定 + `SceneRefs` | 改 `create_chapter_handler.go`、`bindings_novel.go`、`scene.go` | 生成后 `GetChapterScenes` 能看到场景；`Stitch` 与 blob 一致；`go test -race` |
| **2** | `CompileSceneBible`（`internal/novelcontext`）按 POV 裁剪注入 | 新增包；改 `CreateChapter` | 生成 prompt 含本场景实体+状态+POV 视图；POV 泄漏单测 |
| **3** | `internal/novelstyle`：`StyleFingerprint` + `AiTasteScorer` | 新增包 | 单测覆盖各指标；在「人写 vs AI」样本方向正确；可解释 span |
| **4** | `DeSlopLoop` 定点重写 + 复检 | 新增 | 重写后分降、字数/实体/剧情不变；大/小模型两档；失败诚实降级 |
| **5** | `GenerationGate` 串联：生成后自动 Analyze/Review/Consistency + AI 味 + 修复 | 新增 + 改 `streamCreateChapter` | 端到端生成一章后自动过闸；修复可复现 |
| **6** | 审批制结算：AI 状态 patch 过 schema+作者审批 → `SettleState` append-only journal | 新增 `internal/narrative` | AI 建议无一自动入库；journal 可回放；审计 |
| **7** | 前端：AI 味检查面板（分数+span 高亮+一键去味）、文风指纹面板、场景/POV 视图、全文脑图（复 GraphView/CostGraphView） | 前端新增 | `tsc/eslint 0`、vitest 增长、bridge 同步、绑定面 drift |
| **8** | 验收 + 回退演练 | — | 全量 `go test -race`/vitest/冒烟；每刀可 `git revert` |

**验收口径（量化）**：
- 长文：构造 50 章，验证「死亡角色不复活/物品不凭空消失/时间不倒流/位置不瞬移/不 OOC/POV 零泄漏」。
- AI 味：人写 vs AI 样本方向正确；一键去味分降且字数/实体不变；断点可恢复。
- 作者是上帝：AI 建议补丁无一自动入库；全部走审批→结算；审计可回放。
- 性能：AI 味检测<1s（确定性无模型）；书级重检有窗口夹取（复用 `deepClampMaxChapters`）。

---

## 6. 明确不做（防过度设计）

- 不引入 Neo4j/图数据库；沿 SQLite + 事件日志投影（gaea 独有优势，可随时重建）。
- 不做 AI 全自动写完入库；坚持作者审批制（平台反 AI + 创作者要掌控感）。
- POV 一期只做「布尔知识视图 + 泄漏检测」，不做复杂图谱嵌入。
- 不堆多 agent 训练/自反思；先做确定性状态机 + 确定性 AI 味引擎（可测的主线）。
- 不一次铺满前端；刀 7 按 AI 味+指纹最小面板先行，其余增量。
- 不重写现有 analysis/consistency 成熟逻辑，只升级 + 接线。

---

## 7. 风险

- **POV 是硬骨头**：先做简化版验证价值；阈值须在自采语料重标（社区值仅作基线）。
- **幻觉状态 patch**：schema 校验 + 作者审批 + append-only 回放兜底。
- **成本/延迟**：定点重写/节拍化增加调用 → 按阶段模型路由（规划/草稿/编辑分模型，复用 Seam + 分层路由）。
- **绑定面 drift**：加绑定方法须同步 `bindingNames`/前端 bridge 类型（既有 CI 门禁）。
- **场景制迁移**：新章默认 v4 场景；存量 blob 经 `MigrateV3ToV4`（非破坏，备份 `_v3_backup/`）。

---

## 8. 调研来源

**GitHub**：graphify-novel · hzhang092/novel-agent · xiehuanyi/NovelForge · letta · blackzhanzhan/novel_agent · museflow · novel-bot · LongWriter · RecurrentGPT · loom-novel · fast-detect-gpt · DetectGPT · chatgpt-comparison-detection(HC3) · GPTDetector · Gx664-AIGC · yotta-humanize · de-aigc-ch · novel-creator-skill · chinese-webnovel-skills · Ai-Novel · tavern-cards · TinyStyler · stylometric-transfer · koboldcpp · SillyTavern
**中文产品/平台**：马良写作（KG/活大纲/状态时间线/去AI味改写，[百万字不烂尾](https://maliangwriter.com/blog/ai-long-novel-consistency-guide/)）；阅文作家助手·妙笔；番茄作家助手；笔灵 AI 写作；Coze 长篇小说AI工作室/降AI检测助手
**社区反 AI 味**：[anti-ai-checklist](https://github.com/tance-mang/chinese-webnovel-skills/blob/main/references/anti-ai-checklist.md) · [anti-ai-polish](https://raw.githubusercontent.com/DankerMu/novel-writer-cli/main/docs/anti-ai-polish.md)
**商业**：Sudowrite / NovelCrafter / NovelAI（[Jenova](https://www.jenova.ai/en/resources/which-ai-novel-writing-assistant-remembers-character-arcs-timelines-and-worldbuilding-best)、[NovelAI vs Sudowrite](https://thetoolsverse.com/blog/novelai-vs-sudowrite)）
**论文**：[LongWriter 2408.07055](https://arxiv.org/abs/2408.07055) / 2506.18841 · [Fast-DetectGPT 2310.05130](https://ar5iv.labs.arxiv.org/html/2310.05130v3) · [DetectGPT 2301.11305](https://arxiv.org/pdf/2301.11305v1.pdf)

> 注：Sudowrite/NovelAI/GPTZero 为闭源，无开源仓库，仅作参照；「CHF」检测器名待核。
