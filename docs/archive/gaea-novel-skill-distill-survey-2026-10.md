# 网文写作 Skill 生态调研与吸收对照（2026-10-24）

> 起因：一张宣传图「写小说的一整套 Skill，十个 Skill 从选题写到完稿」（程序员晚刘，2026.09）。
> 本文蒸馏其真实来源，逐项对照 gaea 小说板块现状，给出吸收建议分档。**只调研不立项**，
> 立项项走拍板池。
>
> **执行回填（2026-10-24 当日）**：用户拍板「不要技能，直接蒸馏进代码」，A 档①③与 B 档
> 7/8/9 已落地（N3 选题扫榜全栈 / N2 契约四问+新增申报+主线配额+升华禁令 / N1 内核去味
> 信号扩充，见 .gaea/progress.md 与 internal/novelstyle、prompts/novel-scan.json、
> internal/app/novel_scan_handler.go、CreateNovelModal）。同日续刀四批清完余池：
> A②→N6 对标拆解、A⑤→N7 钩子补全、B10→N4 书封三层模板、B11→N8 设定事实词典、
> B12→N5 审核 fix 方向限制；A6→N11 技法检索库（novel_technique.go 九类自写账本+
> 确定性检索+生成链 advisory 注入，webnovel-writer GPL 只重推导）。**调研清单全项收口，
> 余池只剩 C 档观察池与真机实弹。**

## 一、来源考证：十个名字 → 四个真实仓库

宣传图 star 数不可信（营销数字）。实际考证：

| 宣传图名字 | 功能 | 真实来源 | 协议 | 状态 |
|---|---|---|---|---|
| story-long-scan | 选题 | zenstory-ai/oh-story-claudecode 合集 | MIT | 活跃（2026-10-23 仍在推） |
| story-long-analyze | 拆书 | 同上 | MIT | 活跃 |
| story-long-write | 正文 | 同上 | MIT | 活跃 |
| story-deslop | 去味 | 同上 | MIT | 活跃 |
| story-review | 审核 | 同上 | MIT | 活跃 |
| story-cover | 封面 | 同上 | MIT | 活跃 |
| character-designer | 人设 | 同上（story-setup 内置子代理卡） | MIT | 活跃 |
| chinese-novelist-skill | 开书 | PenglongHuang/chinese-novelist-skill | MIT | 活跃（v2.0 重构） |
| novel-creator-skill | 大纲 | leenbj/novel-creator-skill | README 自称 MIT | **停更约 7 个月** |
| webnovel-writer | 记忆 | lingfengQAQ/webnovel-writer（v6.2.1 插件） | **GPL-3.0** | 活跃（另有 v8 独立产品分支） |

- oh-story-claudecode：13 skill 合集（7.9k star），质量最高的一档；我们的 **novel-deslop /
  novel-agents / novel-review 三个技能正是从它蒸馏的**（v4.282 T2，见各技能目录
  `LICENSE-oh-story.txt`）。
- **协议红线：webnovel-writer 是 GPL-3.0，只能机制重推导 + 自写文本，不许逐字搬运**
  （与 MIT 四仓的「结构化收录」待遇不同）。

## 二、我们已有的覆盖（对照后真正的空白只有两个）

| 功能 | gaea 现状 | 覆盖度 |
|---|---|---|
| 选题 | **无任何面**（全仓 grep 无命中） | **空白** |
| 拆书 | novel_import（书稿导入+AI 角色匹配+骨架）+ novel-agents chapter-extractor（单章情点/写法提取）+ style_digest（文风提炼）+ 书源阅读 | 半覆盖，无「拆书知识库」编排 |
| 开书 | 建档向导 + 档位（maturecraft）+ worldview/prompts 24 模板 | 覆盖 |
| 大纲 | chapter-plan.json + 大纲节点 + stage-recap（10 章一幕）+ chapter_plan_position（起承转合位置纪律）+ plot_branch 四位置 | 覆盖 |
| 人设 | **novel-agents character-designer 卡已含全套**：反派四要素+四步法、动机链四环（起因→意图→约束→风险）、三层反差标签、「遮住名字认不出=差异化失败」检验 | 覆盖 |
| 正文 | create_chapter 管线（plan gate/cast/context/stage/first）+ GhostText + 场景局部重写 + 长跑纪律 | 覆盖 |
| 记忆 | 阶段八一致性（novelcontext/characterstate/foreshadow_v2 六态）+ 阶段九上下文预算 | 覆盖 |
| 去味 | **novel-deslop 技能（gates A-J 三层）+ 内核 novelstyle 确定性引擎**（27 替换+19 黑名单，宁漏勿误） | 覆盖 |
| 审核 | **novel-review 技能**（26 维 PASS/WARN/FAIL + 黄金三问 + S1-S4 + 番茄/起点/知乎平台档）+ novelgate 确定性触发器 + BookHealthPanel 快照/基线/张力 | 覆盖 |
| 封面 | GaeaGenerateBookCover（3:4，世界观/角色组 prompt + promptHint，落 play exports） | 覆盖（简版） |

## 三、吸收建议（A/B/C/D 分档）

### A 档：技能资产直接落（不加内核、符合「领域知识技能化」红线、成本低）

1. **novel-scan 选题技能（新技能，唯一真空白）**。吸收 story-long-scan 的：
   - 数据来源三级降级 + 每级强制诚实声明（脚本采集 → 作者粘贴/截图 → 内置知识标注
     「未实时校验，待验证假设」）；**起步只做粘贴/截图通路，不做爬虫**（见 D 档）；
   - 8 维扫榜分析（题材分布/新题材信号/头部老书/字数分布/书名模式/简介卖点…）；
   - 可行性量化门槛（同方向样本 ≥15 本 + 环比涨平 + 有差异化 = 高；首次扫榜无上期可比
     一律「封顶中」）；
   - 选题决策 12 固定字段落盘 + 「能爆原因→待拆文验证→拆文回填」与拆书技能的字段级契约。
   - 产物建议制呈现（用户显式发起才落盘，符合记忆写入红线精神）。
2. **novel-analyze 拆书技能（新技能，编排既有件）**。吸收 story-long-analyze 的：
   - 机械章节索引（行号定位，原文每块只读一次，主会话不读原文）；
   - 五层产物分层（章节摘要 / 剧情故事线 / 节奏 / 情绪模块 / 文风档案），各管一层可定点引用；
   - 断点续跑五路径分流（new / continue / direct_use / repair / enhance）。
   - 复用：novel_import 解析 + chapter-extractor 卡 + style_digest。
3. **去味信号大表扩充（纯资产增补）**。上游 banned-words 信号表比我们 gates.json 的
   markers（仅 6 个）大一个量级：解释结论链（这意味着/真正重要的是/他终于明白）、意义膨胀、
   隐性盖章点破潜台词（关切得恰到好处/谁都看得出）、告知与气场（她感到/显得有些）、
   电报体动作清单、论述公文腔、通用反应（深吸一口气/瞳孔微缩/指节泛白）、通用心理、
   空泛定性、模板比喻、自动过渡（映入眼帘/话锋一转）等大类 + 每类正反例。
   落 `novel-deslop/references/ai-patterns.md`（advisory 层，模型执行）；
   **内核 words.json 保持 27+19 保守基线不动**（宁漏勿误口径不变）。
4. **去味「防矫枉过正」反例表**。上游最独特的价值观：每类配「看似去味却改坏了」正反例
   （删否定翻转把承诺态度改没了；合并碎句凭空新增服药后果）；明确禁止同义词轮换等规避手段；
   「先治电报体再查套词」（我们 gates D 已有同款判断，反例表还没有）。同落 ai-patterns.md。
5. **钩子技法资产**。chinese-novelist-skill 的钩子十三式（突然揭示/两难/意象钩/回声钩/留白钩…）
   + 五禁忌（虚假悬念/机械降神/过度留白/低风险/同质）+ 播种—生长—收获三段铺垫 +
   「长短中三条悬念弧并行、一条结束另一条须在高潮」。落 story-architect 卡增补或
   novel-analyze 同目录技法参考。
6. **网文技法检索库思想**。webnovel-writer 九表 CSV（人设关系/写作技法/命名规则/场景写法/
   桥段套路/爽点节奏/裁决规则/金手指/题材调性，统一「关键词/意图同义词/适用题材/指令/
   核心摘要/详细展开 + **毒点/正例/反例**」列结构，按需检索注入不常驻）。**GPL 只重推导**，
   自写内容后落技能 references 或知识中枢内容。

### B 档：现有面小刀增强（需拍板）

7. **契约四问 → create-chapter 写后自检 + novel-review 核对项**（我们两处均无）：
   主角是否靠自己选择挣到本章结果？收益是否被配角/巧合无交换拿走？是否提前动用后面
   的底牌？旧期待付利息、留新期待了吗？
8. **新增物三级申报制 → create-chapter 输出契约**（治「AI 私加设定」）：
   直接写（路人/器物不申报）/ 写了要报（具名配角/新伏笔/新承诺 → 回执「本章新增」段列给作者）/
   先问作者（新主线反转/金手指规则/改已定结果 → 不写，挂起裁决）。
9. **事件配额 + 冷却 + 反收束 → 位置语境增补**（chapter_plan_position 一行纪律级）：
   每章至多一项主线级实质推进（A 主线矛盾 / B 关系升级 / C 核心秘密揭露），触发后逐章冷却；
   非终局章禁解主线冲突、章末悬念等级不得为「无」。与既有起承转合「不透支后段」同思想，
   是其细化。
10. **书封提示词增强**：三层模板（文字层书名+笔名+装饰 / 风格层平台关键词 / 画面层
    前景人物-中景场景-远景氛围）+ 安全区指令（inner ~85%, no watermark）+ 10 题材风格表
    （字体/色彩/人物/背景/光效）+ 构图先选后生成 + 封面版本自增 + prompt 留档。
    **真机验证点：书名文字渲染依赖 ComfyUI/Qwen 后端能力，上游是 GPT-Image 模板，不保证平移。**
11. **事实词典「读者已知 vs 完整真相」双列**：世界观/设定类事实加双时间线（foreshadow_v2
    六态只覆盖伏笔进度，设定事实无此分列）。先盘底 novel_state/worldview 现有字段再定刀深。
12. **审核 fix 方向限制一行纪律**：consistency/factual/causal 类 finding 的 fix 只准写
    「事实统一方向」，不准写文学创作建议（防审稿越权改剧情）。novel-review SKILL 补一句。

### C 档：观察池（先盘底 / 挂实弹，暂不立项）

13. 记忆四状态机（active/outdated/contradicted/tentative）+ 同 key 旧值降级不删除——
    novelcontext/characterstate 现状无状态机命中；若长跑出现「状态覆盖丢历史」实弹痛点再立刀。
14. 大纲锚点数值配额（「本章进度 0.14-0.15，严禁越过」）——stage-recap+位置语境已同思想，
    看长跑实弹是否剧情狂飙再定。
15. 退化检测豁免工程（对话行豁免拒绝语「我无法答应你」、自指前瞻防误伤复合词）——
    挂实证误报案例再做。
16. 字数带治理（作者区间优先 / 欠字不自动补、三选一）——先核对 create-chapter 字数行为现状。
17. 开书三层递进问答（可随机/跳过/默认降摩擦）——建档向导已有档位，UX 磨滑低优。
18. RAG 剧情检索 / 知识图谱 / 跨 Agent 双审（leenbj 仓，停更 7 个月）——阶段九检索覆盖
    同类需求，只当思想参考。

### D 档：不吸收

- **browser-cdp 爬榜**：ToS 灰色 + 站点脆弱 + 维护面大；选题技能走「作者粘贴/截图」诚实通路。
- **跨宿主运行时降级链**（Claude/Codex/OpenCode/Antigravity agent 文件预检）：gaea 单宿主
  用不上，我们的技能在内核按 run_skill 派发。
- **webnovel-writer v8 / DeepSeek Harness 产品化方向**：与我们桌面应用路线无关。

## 四、红线对照

- 领域知识技能化：A 档全部落技能资产 / 提示词模板，内核零新增领域规则 ✓
- GPL 隔离：webnovel-writer（GPL-3.0）只机制重推导 + 自写文本；MIT 四仓延续
  「结构化收录 + LICENSE 附注」既有实践 ✓
- 记忆写入确认红线：选题/拆书产物落盘须用户显式发起，报告建议制 ✓
- 提方向核对删除史：本批不涉及 GoalCard 系 / MCP / 平台化 / 跨设备记忆包 ✓

## 五、若只做三件（拍板池建议序）

1. **novel-scan 选题技能**——十个功能里唯一的真空白，完整吸收其「诚实降级 + 量化门槛 +
   落盘契约」骨架；
2. **契约四问 + 新增物申报**（B7+B8）——两处小刀，治「写跑题」与「私加设定」两大生成痛点；
3. **去味信号大表 + 反矫枉过正反例**（A3+A4）——纯资产增补，零代码风险。
