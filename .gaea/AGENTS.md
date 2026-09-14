# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 更早版本见 `docs/archive/agents-version-history-2026-09.md`（覆盖 v4.173–v4.146 与 v4.49 及更早）；**v4.50–v4.145 区间本仓只在 `CHANGELOG.md` / `releases/` 有记录**（速览逐版滚出后未回填）。本段只留最近 14 版：本文件曾达 104 KB 超工作区指令预算（65536 B），尾部纪律/规划段一度对后续会话不可见，2026-09-10 二次分流。
- **最新发布：v4.304.0（2026-09-15）「t4-C3 首刀：驱动式整章重写 + 版本库（应用/丢弃/恢复）」**——MuMu 蒸馏 t4 情节分析域第二刀（§5.2/§8.3 C3，契约 plot_v2.go v4.278 已落）；C1 结构化建议产出后补重写消费链。**落地**=①重写引擎（新 internal/rewrite 纯规则零 IO）：NormalizeRequest（custom 需指令/suggestions 需索引/mixed 至少其一；字数缺省 3000 钳 [500,10000]；未知 focus 丢弃）+BuildModificationInstruction（四段固定顺序：改进问题〔仅选中建议越界静默跳过 MuMu L129〕/自定义要求/重点方向五选映射/保留元素逐条展开）+CleanRewriteOutput（8 前缀命中即 break+成对引号 rune 级剥离）+ComputeDiff（difflib.SequenceMatcher 以本地 textsim CJK 二元组 Dice 替代，码内注明）；②版本库（契约路径 rewrites/<chapterNum>/）：index.json 只存索引时间倒序**无全文下发**（防 MuMu 1.74MB 全量推送）、全文 <id>.json 按需读、Save 自动生成 ID rw-<chapter>-<unixnano>、Update 同步双文件；③六绑定（NovelB，672→678）：NovelChapterRewrite（归一→analysis-v2 取该章建议〔缺分析显式报错〕→指令→RTCO rewrite-chapter 模板〔原全文+指令+前文衔接；{word_count} 复用既有替换器〕→LLM 温度 0.7〔MuMu L92〕10 分钟超时→清理/空结果放弃→diff→版本 completed **不自动落章**）+List（索引）/Get（全文对比）/Apply（写回+applied 幂等）/Discard（快照保留可再应用）/Restore（OriginalContent 写回+RestoredAt/RestoredFrom 审计——**gaea 强制增量：MuMu 放弃只清前端内存且无 restore 路由**）；④场景级工程章（正文按场景存储 SceneManager.List 非空）显式拒绝下刀放开；前端对比确认 UI 随消费刀补（契约面已登记：bindingNames/spaceBindings 锁 494→500/bridge 类型）。**测试**=rewrite +4（归一矩阵/指令四段含越界跳过/清理含非成对不剥/diff 相同 100 与篇幅-47%）+project +1（回环/倒序/状态同步/非法 ID 路径护栏）+app +4（桩 LLM httptest SSE e2e 全链：重写剥前缀→不自动落章→应用→恢复审计/守卫链/丢弃语义/camelCase 形状锁）；既有零改动。**门禁**=ci.ps1 全绿、绑定面 +6 drift OK@678、版本三处 4.304.0；产物 exe 50312704B SHA256=c27ff6d3fb3bf0909ed2d46f7e76a3a7c6afec1d88121200f058a2baa1b625b3（冒烟 200）。**未做（下刀）**=partial 局部重写（字符区间+长度模式+前后文衔接+应用拼接）/场景工程整章重写/前端对比确认 UI；t4-C4 锚点标注高亮；t5 角色职业/t6 提示词工坊/t7 前端接线未开工；真机走查等闲置窗口。
- **最新发布：v4.303.0（2026-09-15）「t3-P2 记忆召回消费：语义检索进 SceneBible 记忆区段（长程一致性全闭环）」**——MuMu 蒸馏 t3 长程一致性域第三刀（§3.3/§3.4/§12.1/§12.4）；分析自动回填（P1）+语义召回注入（P2）全通；检索复用 semantic.Store（Ensure 快照比对）+本地 embedding（Herdsman bge-m3），不引 ChromaDB（§12.3 不适配）。**落地**=①结构化 query（§12.4-1 本域最可借鉴段）：人物[:8]/关键事件[:6]/叙事目标/情绪/本章要求[:150] 分段限额拼接整体[:800] 换行压空格，不嵌大纲原文；②四段流水线（§3.4）：阈值筛选 storyMemoryThreshold=0.35（gaea 归一化余弦语义，不照抄 MuMu 未归一化 L2 的 0.6）→全低分兜底 storyMemoryFallback=3（「无高分不如少带」）→注入 storyMemoryTopK=8，三常量唯一声明；③SceneBible 增 Memories 字段+Render「相关记忆（按相关度）」区段（行 `- (相关度:0.82) 内容[:120]` MuMu :789 同型；区段预算 memoryBudget=400；空不出标题；参考信息非硬约束与伏笔硬约束层区分）；④索引维护规避 MuMu D1 向量残留：Ensure 增量+Stale(kind,keep) 清死向量+kind=story_memory|<项目目录> 隔离（§12.3 单机以项目目录隔离 semantic_vectors 全局表）+当前章排除（正在写的内容不召回）；⑤CreateChapter 接线：BuildSceneBible 后按节点 ID 定位→recallStoryMemories→Render 前；全链容错无库/无 embedding/无记忆/无节点→nil 不出区段绝不中断生成；20s 超时；零绑定面。**测试**=app +4（query 分段限额与总长/流水线三态/行格式/无 embedder 降级）+novelcontext +2（区段渲染截断预算/空态）；既有零改动。**门禁**=ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.303.0；产物 exe 50259456B SHA256=de5cd5ef359df1e3d04bc80b666d23dc405f7d1fdca1e977ce3949c9f78d3485（冒烟 200）。**未做（观察池）**=t3 三刀收官（窗口预算化→记忆生产者→召回消费）；真机检索质量（相关度分布）待真实长书评估 threshold=0.35；记忆质量可评测（调研档候选 2）开放；t4-C3 驱动式重写/t4-C4 锚点标注/t5 角色职业/t6 提示词工坊/t7 前端接线未开工；真机走查等闲置窗口。
- **最新发布：v4.302.0（2026-09-15）「t3-P1 记忆生产者：规则表抽取 StoryMemory 按章落盘（自动回填闭环）」**——MuMu 蒸馏 t3 长程一致性域第二刀（§2.3/§12.1/§12.2/§12.4-4）；t4-C1 解锁 V2 载荷后补「分析→结构化记忆」抽取——**分析一章，设定库自动回填闭环咬合**（市场差异化点：Novelcrafter 手填摩擦/Sudowrite 书长极限，调研档 §9）。**落地**=①types.StoryMemory 字段子集+确定性 ID `<MMM>-<type>-<ordinal>`（§12.2 幂等前提）+五类类型常量+is_foreshadow 三值（0 普通/1 埋下/2 回收 MuMu :47 口径）；无 Position 字段（依赖未落地的 t4-C4 偏移标注，不留恒 0 死字段 D11）；②提取规则表纯函数 ExtractStoryMemories（六类对齐 MuMu :305-486）：chapter_summary 必有一条（summary→前 3 推进点拼接→正文前 300 字纯截断三级回退链，全空跳过不造数，固定 0.6）/hook strength≥6 min(s/10,1)/foreshadow 全部入库〔planted=1 resolved=2，strength 缺省 5 与 t1-P2 同口径〕/plot_point importance≥0.6 原样/character_event 每差分一条固定 0.7 related=[角色名]/conflict level≥7 记 plot_point 类；门槛常量唯一声明改之即改；③按章落盘 memories/MMM-<n>-memory.json（project.ReadChapterMemories/WriteChapterMemories 整章替换写=重分析幂等且天然规避 MuMu D1「重分析不清旧数据」；空 items=删除该章记忆文件不留陈旧档；缺文件=空正常态）；④Analyze 主链接线 persistStoryMemories（容错注入失败只记日志）。**测试**=analysis +3（六类规则表逐条含门槛边界 5/6·0.4/0.6·6/8/确定性 ID 重提取同 ID+序号编排/summary 三级回退链含纯截断与三级全空跳过）+project +1（回环/空写删文件/缺文件空态）；既有零改动。**门禁**=ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.302.0；产物 exe 50247168B SHA256=02218224c7e46554928d4e641819d25960ab0e82b4cbda9e87215c11c290db4c（冒烟 200）。**未做（下刀）**=t3-P2 召回消费（novelcontext.RecallStoryMemories 走 semantic kind=story_memory+SceneBible 记忆区段 memoryBudget+三常量 topK=8/threshold=0.35/fallback=3+结构化 query 构造 §12.4-1+Ensure/Remove 索引维护）——生产者已就位召回接上即全闭环；t4-C3 驱动式重写/t4-C4 锚点标注；真机走查等闲置窗口。
- **最新发布：v4.301.0（2026-09-14）「t4-C1 分析代理 V2 化：9 维结构化+三维评分联动+analysis-v2.json 落盘」**——MuMu 蒸馏 t4 情节分析域首刀（§8.3 C1）；市场调研增量轮（调研档 §9：Novelcrafter Codex 手填摩擦/Sudowrite 书长极限/蛙蛙设定库）抬升优先级——自动设定库回填是行业痛点，V2 载荷是回填数据底座。**落地**=①模板 analysis-chapter 原地 V2 九维化（hooks/conflict/emotional_arc/character_states 差分/plot_points/scenes/pacing/文白比+scores 三维+suggestions 联动+summary；t1-P2 伏笔追踪约束与 P1 候选槽位原样保留，ForeshadowHit 契约零改动）；②服务端权威归一 normalizeAnalysisV2：三维钳位 [1,10]+overall 一律 ComputeOverallScore 重算（模型自报丢弃=禁安全分机制保障）+建议条数 SuggestionCountForOverall 只裁上限不代拟；③analysis-v2.json 落盘（project.UpsertAnalysisV2 按章号覆盖/升序/原子写；ChapterAnalysisResult 带引擎/模型/analyzer_source；容错注入失败不影响分析返回）；④旧 wire 零破坏 deriveLegacyAnalysis（AnalyzeChapter 返回键零变化：hook=首钩/conflict=描述/情感曲线=主导情绪：轨迹/关键事件=推进点前 5/节奏中文映射/质量分=overall 取整/角色状态差分映射）；syncCharacterStates 升 V2 差分（空 NewState 不写不猜值）。**测试**=analysis +4（钳位重算裁剪含 6.0 落 [2,3] 档边界/分档矩阵/legacy 映射逐键/空载荷安全）+project 落盘回环；既有零改动。**门禁**=ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.301.0；产物 exe 50229760B SHA256=739399a4a2098e63937246dc2a1444634b9083d7e01f17404523fd901b582f83（冒烟 200）。**未做（下刀）**=t3-P1 记忆生产者（已解锁：StoryMemory 确定性 ID+入库门槛规则表 hook.strength≥6/plot_point.importance≥0.6/conflict.level≥7+从 analysis-v2 抽取落盘=自动回填闭环）；t4-C3 驱动式重写（internal/rewrite+版本快照）；t4-C4 锚点标注高亮；真机走查等闲置窗口。
- **最新发布：v4.300.0（2026-09-14）「t3 首刀：章节前文摘要窗口预算化 + 回退链 + 反重复约束」**——MuMu 蒸馏线 t3 长程一致性域首刀（零绑定面）。§11.3 对照显形的 gaea 自身缺口：章节生成 prevSummary 遍历**全部**前章、每章 200 rune **无累计上限**（200 章≈40k rune prompt 前缀），且不参与 ctxBudgetTotal 预算体系。**落地**=①最近 10 章窗口（buildPrevSummaryWindow 纯函数）：只注入本章之前最近 10 章（ctxPrevWindowChapters=10 对齐 MuMu :1375）、章号升序确定性拼接、单章 180 rune 截断（ctxPrevChapterLen 对齐 :1408）、无可注入返回空串跳槽位；②窗口声明头「（仅含第 X～Y 章概要；更早章节的剧情同样有效，只是不在此重复）」——部分视图语义显式化，防模型把窗口外既定剧情当不存在；③摘要回退链（§12.4-5）：大纲节点 Summary → chapters/NNN-summary.json 章节摘要（原实现从未进前文窗口）→ 该章跳过不占位；④反重复约束（§12.4-6 对齐 MuMu prompt_service.py:792-805 写法）：create-chapter 模板 must 增「前文摘要仅作衔接参考：直接推进本章新进展，不得复述其中已完成的剧情；前文摘要只含最近章节，更早的既定剧情同样不得矛盾」。**测试**=app +4（200 章窗口过滤+分隔计数+声明头/单章截断/回退链含两路皆空整章跳过/乱序升序+三类空态+小 limit 钳位）；既有测试零改动（grep 确认无 prev_summary 断言牵连）。**门禁**=ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.300.0；产物 exe 50211840B SHA256=344b72d9228fe53ea774253b7c7f0fbf0096036382bc8411c30623085b0e6bd6（冒烟 200）。**未做（t3 下刀）**=P1 记忆生产者（types.StoryMemory 确定性 ID+入库门槛规则表 hook.strength≥6/plot_point.importance≥0.6/conflict.level≥7+落盘 memories/MMM-n-memory.json——依赖 t4 产出 AnalysisResultV2，建议与 t4 排序联动）；P2 语义召回消费（semantic kind=story_memory+SceneBible 记忆区段 memoryBudget 300-500+三常量 topK=8/threshold=0.35/fallback=3）；真机走查等闲置窗口。
- **最新发布：v4.299.0（2026-09-14）「伏笔面板接调度可视面：清理入口/统计/紧急度 Badge/同步结果（t1-P4）」**——MuMu 蒸馏 t1 伏笔域第四刀（收官刀；P1~P3 全在后端，前端面板仍是旧三态视图）。**落地**=①SyncResult 上绑定面（v4.297 欠账）：analysis.Agent 捕获最近一轮同步结果（读写锁）+新绑定 GetLastForeshadowSync（NovelB 委托，面 671→672），回收/新埋/内容匹配/跳过计数+skippedReasons 全量（D3 跳过可见不静默），尚未分析显式报错；②GetForeshadows 附带 urgency 运行时投影（types.ForeshadowUrgency：level/remaining/overdue/resolveStatus/mustResolve；UrgencyLevel/ClassifyResolve 共享入口=A5 不落库+D7 后端算+D9 唯一；回收/废弃不带键；currentChapter 自动按已写章数）；③ForeshadowPanel 四件消费：后端统计行（分状态+超期红 Tag，失败降级前端计数）/紧急度 Badge（3 红 2 橙 1 金+tooltip 时机中文）/生命周期清理折叠区（重分析前清理/删除本章〔只删分析默认开〕/项目重置，全 Popconfirm+toast+自动重载，文案明示手动条目永不批量删除）/上次分析同步常显卡（计数+前 2 条跳过原因+溢出注记，未分析不显示）；④A5 护栏=写回经 stripForeshadowUrgency 剥离投影防意外持久化（形状锁测试）；⑤契约面=bridge/novel.ts 四载荷类型+GetLastForeshadowSync 签名、types.ForeshadowItemData +urgency?、dev mock +5 诚实空态、load() 增量绑定缺失降级（旧桥接/局部 mock 不崩）。**测试**=Go +2（投影矩阵：超期 level3/剩余 1 章=急需 level2 时机 not_yet/无计划 level0 仍投影/回收不带键/currentChapter 自动；lastSync 未分析报错+D3 可见；SyncForeshadows 按规格签名转正导出）+vitest 面板 14/14（+6：统计行/Badge/写回剥离/同步直显/删除·清理逐参）。**门禁**=ci.ps1 全绿（run1 抓 SaveForeshadows 三例 items 类型断言未随 map 投影更新——GetForeshadows 形状有意变化，当场改助手 JSON 回环）、绑定面 +1 drift OK@672（spaceBindings 锁 493→494 play）、版本三处 4.299.0；产物 exe 50,209,280B SHA256=1209d0f139ac7d47d2f92738f1e7a126585947539b828bb3063dc3b7a28dc623（冒烟 200）。**未做（观察池）**=真机走查（分析→回收→清理→统计闭环，等闲置窗口；%APPDATA%/gaea/prompts 旧模板覆盖先核）；SyncResult 持久化不做（会话内诊断面）。
- **最新发布：v4.298.0（2026-09-14）「伏笔生命周期清理与统计 + Lint 扩两码（t1-P3）」**——MuMu 蒸馏 t1 伏笔域第三刀（spec §8.1/§5.1/P3.4；P2 打通自动回收后重分析/重新生成缺「干净重来」入口，手工条目与分析产物混居一张表）。**落地**=①三个清理入口（新 `internal/app/foreshadow_cleanup_handler.go`，语义严格区分）：DeleteChapterForeshadows(chapterFile,onlyAnalysisSource)=删埋入∨回收本章条目（默认只删分析来源）；CleanChapterAnalysisForeshadows(chapterFile)=重分析前只删「分析∧埋入本章」+回退本章回收条目→planted 清回收痕迹（§8.2 系统内部迁移）；ClearProjectForeshadowsForReset()=删全部来源，手动条目**重置不删除**→pending 清章节关联与时间戳（标题/描述/长线保留）；关键不变量=source_type 唯一判据、手动条目永不批量删除（存量无 source_type 按手动保护）；统一返回 ForeshadowCleanupResult{deleted,rolledBack?,resetManual?}；②统计 GetForeshadowStats(currentChapter)（§5.1）：分状态+longTermCount+overdueCount（存活×ClassifyResolve 唯一入口=D9 延续；currentChapter≤0 自动按已写章数），resolved 别名归一并入，Total=分桶和；③Lint 扩两码 5→7：overdue（medium，计划回收章<已写章数——超期条目正以硬约束进生成上下文应尽早处置）+unplanned（low，无 target_resolve_in 且埋入≥10 章长线豁免；与 stale 差异=「该收了」vs「缺计划字段无法按章调度」，两码并存），既有五类零改动；④有意偏离=不引 SourceAnalysis 引用分支（gaea 不写该字段，引入即 D5 死代码同款）、§8.3 分析记录联动无落盘目标暂不做。**测试**=app +6（三入口 e2e 来源护栏/回退含 partial/重置字段清空；统计别名归一+超期口径 2→3；Lint 新码矩阵+既有回归；camelCase 形状锁）。**门禁**=ci.ps1 全绿（run1 netclient 测试 exe AV 瞬时锁 Access-denied 已知 flaky 隔离绿，run2 全绿）、绑定面 +4 drift OK@671（NovelB 门面 4 委托，spaceBindings 全归 play 锁 489→493）、版本三处 4.298.0；产物 exe 50,193,920B SHA256=489ef004ba0bc2cbd9f54eebd798640413e7283da51e6711d42dfad770a437c5（冒烟 200）。**未做（下刀）**=t1-P4 前端（面板接清理/统计/新体检码+Badge 用后端 urgency，四绑定 mock 随消费面补）；真机走查（分析→回收→清理闭环，等闲置窗口）。
- **最新发布：v4.297.0（2026-09-14）「伏笔分析驱动自动回收：三级匹配闭环+候选清单三层渲染（t1-P2）」**——MuMu 蒸馏 t1 伏笔域第二刀（spec P2 闭环核心；此前回收只做「描述精确相等」，候选清单是整包 JSON 直塞 prompt）。**落地**=①纯函数层（新 `internal/types/foreshadow_match.go`）：WordOverlap（rune 级 2/3-gram Jaccard o2*0.4+o3*0.6）+MatchForeshadowByContent 六策略加权（标题族取最大 1.0/0.95 后缀剥离/0.8/0.75、关键词 0.75、内容×0.6；引用章+0.15/分类+0.1/角色 Jaccard+0.1 累加；采纳=严格大于且≥0.5 同分取先=最早埋入）；②同步层重写（新 `internal/analysis/foreshadow_sync.go`，消费 v2 契约 ForeshadowHit 替代旧 ForeshadowAction）：回收三级匹配=精确 ID（查不到**禁止回落内容匹配**〔原则 3 关键回归〕）→内容兜底→跳过不新建（只有埋入创建记录）；已回收不重置（区分本章/历史章）、pending/abandoned 拒收、hinted/partial 可回收（D15）；埋入两道防重（稳定 ID+同章同题）+每章新建≤5（更新不占额）+Importance=min(Strength/10,1) 唯一派生+计划章缺失不猜值；SyncResult 完整含 SkippedReasons 六类 Kind（修 MuMu D3 静默跳过）进 slog；③候选清单三层渲染（新 `internal/analysis/foreshadow_prompt.go`）：L1 本章必须回收逐条带「⚠️ 回收时 reference_stable_id 填写: {id}」紧邻指令/L2 超期≤5/L3 其他≤10+溢出注记，分层走 ClassifyResolve 唯一入口（D9），include_in_context=false 不进 prompt 面（同步池不受影响），auto_remind=false 分析侧不消隐（D8 只管生成侧 L3）；④`prompts/analysis-chapter.json` 升级：foreshadows 换 v2 字段+候选槽位升 P1+🔴伏笔追踪任务指令+三条强约束（标题禁「回收」后缀/ID 回收必填/预估章埋入必填）+ID 追踪自检（spec §6.1）；写入口径仍 revealed，前端对分析 foreshadows 形状零消费零破坏。**测试**=types +4/analysis 同步 15 例（4 迁移+11 新增：无效引用不回落/无匹配不新建/同批不双吃/上限与防重/评分钳制）+渲染 4 例（三层/排除口径/折叠上限/空态 D14）。**门禁**=ci.ps1 全绿、drift OK@667（零绑定面）、版本三处 4.297.0；产物 exe 50,173,440B SHA256=d107e3ae8369c9b250a9976061edcf39eb613788eaae41d66ec8fc69f0bf5306（冒烟 200）。**未做（下刀）**=t1-P3 清理入口/Lint 扩展 overdue·unplanned/统计口径；t1-P4 前端（表格/Badge 用后端 urgency）；真机走查（分析闭环+书源线一条龙，等闲置窗口）；SyncResult 上绑定面随 P4。
- **最新发布：v4.296.0（2026-09-14）「原罪·书源线收官 t5：成书送小说导入 + EPUB 导出」**——sin 书源线最后一刀（t1 引擎→t2 六绑定→t3 书源卡→t4 故事工具→t5 出口，t2~t5 自 DSH 移交本线执行）。**落地**=①成书行「送入小说」：sin 前端直调既有 ImportNovelBookEx（full 全本走同一落库链）+成功后 NAVIGATE {page:'novel'} 跳书架（零新后端耦合）；ImportNovelBookEx 按 legacy 直调转正升进 bridge NovelBindings（含 NovelImportResult 载荷，facets 归 play，drift legacy 清单移除）；②SinBookSourceBookExportEpub（SinB +1→21，面 667）：TXT→同名 .epub 同目录覆盖语义，确定性逆解析自己组装的格式（书名/作者/简介头+「第N章」章头+非空行=段）→go-epub，护栏抽 sinGuardBookPath 共用，删除成书连带清同名 .epub；③mock 诚实拒绝两件。**测试**=Go +4/vitest +2（NAVIGATE detail 断言），sin 域 86/86。**门禁**=ci.ps1 全绿、drift OK@667、版本三处 4.296.0；产物 exe 50,127,872B SHA256=DD72F0781F3E3C9E810EC7F696185730E49DD740532F2E08EB90FE7514A3736E（冒烟 200）。**未做（观察池）**=真机一条龙走查（书源卡+工具链，等闲置窗口）；断点续传/cf-bypass/真实规则出厂维持不做。
- **最新发布：v4.295.0（2026-09-14）「原罪右栏卡片化」**——用户反馈「五个标签并排太多，参考办公右侧面板做卡片化」；页签条改手风琴卡片堆：五功能卡（角色/大纲/设定/插图/书源）卡头常显、一次展开一张（办公右栏「一次聚焦一个视图」同型），展开卡高亮+箭头旋转+计数徽标（书源卡无计数=清单自管），新增面板头（「创作面板」+问号气泡收于此），展开卡即滚动容器（删旧页签条与 .sin-side-body 样式），宽度拖拽/底稿编辑/画廊预览零变化，展开卡沿用 gaea.sin.panelTab 旧值直接迁移，卡头补 aria-expanded+aria-orientation=vertical。**测试**=sin 域 84/84（卡头文本与 role 语义保持，既有断言零改动通过）。**门禁**=ci.ps1 全绿、绑定面零变化 drift OK@666、版本三处 4.295.0；产物 exe 50101248B SHA256=（冒烟 200）。**未做**=sin 书源线 t4 工具接线/t5 送小说导入+EPUB；真机走查等闲置窗口。
- **最新发布：v4.294.0（2026-09-14）「原罪·书源接线 t3：右栏「书源」页签」**——sin 书源线 t3（t2~t5 自 DSH 移交本线执行，t2 六绑定 79271433 非版本刀：SinB 14→20、规则目录按「以先落地为准」收敛共享层 %配置目录%/gaea/booksource/rules、成书落 sinRoot()/books sin 数据面）。**落地**=①原罪右栏第五页签「书源」（sinPanelState 兼容旧值，无计数徽标）；②SinBookSourcePanel：搜书→候选表（HasRule=可下载/免规则「仅参考」禁用=后端 fail-closed UI 同款；warnings 直显）→目录预览（总数+首末样例+truncated）→范围选择（默认全本钳 [1,total]）→下载成书（进度每 20 章上报+取消；关面板不打断；事件 sin-booksource:<jobId> 三路 progress/done/error 必退订）→成书清单（新→旧+Popconfirm 删除）；③dev mock 诚实空态（mock/sin.ts +6）；④events.ts SIN_BOOKSOURCE_PROGRESS（后端事件 24→25）。**测试**=vitest +6+SinSidePanel 页签断言更新（sin 域 84/84）；坑复训=antd 两字按钮插空格（「取 消」「删 除」）、listitem 可访问名来自 title 属性非内容。**门禁**=ci.ps1 全绿、drift OK@666、版本三处 4.294.0；产物 exe 50,098,176B SHA256=4C0D8C0B38AEC991EAC30D8DEE648DC06A389640521BBACB305F385A22294614（冒烟 200）。**未做（下刀）**=t4 book_search/book_download 进 sinToolOrder；t5 送小说导入（直调 ImportNovelBookEx 零后端耦合）+EPUB 导出（go-epub 已在依赖）；真机取书走查（状态化假站配方等闲置窗口）。
- **最新发布：v4.293.0（2026-09-14）「伏笔分层注入：按计划回收章调度生成上下文（t1-P1）」**——MuMu 蒸馏 t1 伏笔域消费方第一刀（契约 v4.278 落库后零消费方；计划回收章=本域最大增量）。**落地**=①纯函数层（新 `internal/types/foreshadow_urgency.go`）：UrgencyLevel（0-3 运行时不落库；D15 partial/hinted 有回收压力；缺计划章不猜值）+ClassifyResolve 四值+ForeshadowLayerOf 分层唯一入口（L1 必须回收/L2 超期/L3 近期参考/L4 本章计划埋入/L5 无计划兜底〔gaea 扩展：存量无 target_resolve_in 条目保底注入防回归〕）+调度常量唯一声明（lookahead=5 上限 20/remind 缺省 5/急需≤2/每章新建上限 5）；②include_in_context=false 全层排除、auto_remind=false 只抑制 L3（D8 硬约束不消隐）、远期不注入；③四层渲染（新 `internal/app/foreshadow_context.go`）替代旧单层注入：模板对齐 spec §4.2（D14 省略号按长度判断）、L2≤3/L3≤5/总量≤15/整区含标题≤1600 rune、消耗顺序 L1→L2→L3→L4→L5、空层不输出标题、区段头带调度规则约束行；④`resolveTargetChapterNum`（显式/分支父节点/顺延，与 ensureChapterNode 同源复用）供分层按正确本章章号判定；⑤novelcontext 场景圣经同步分层优先采样（硬约束层带 `[本章必须回收]`/`[已超期N章]` 标注先于参考层；D9 消除：分层逻辑全仓唯 types 一处）；⑥删 `docs/mumu-distill/` 重复副本（v4.278 观察项销账）。**测试**=types +4/app +7/novelcontext +1（抓出超期章数负号 bug）/既有 context 测试迁移分层口径。**门禁**=ci.ps1 全绿、drift OK@660（零绑定面）、版本三处 4.293.0；产物 exe 50,061,312B SHA256=DDEFF9A62EC67F470262728874D9E73ED3AA93C44DD8310B42A5BB41F1541E51（冒烟 200）。**未做（下刀）**=t1-P2 分析驱动自动回收（MatchByContent 六策略+SyncForeshadows 三级匹配+分析 Prompt 候选清单）；t1-P3/P4。

- **最新发布：v4.292.0（2026-09-14）「tail×反推串联：末 N 章导入完成后一键反推续写大纲」**——拆书线按反馈项：tail 产品意图（用末尾几章反推「接下来该怎么写」）此前只完成一半，反推还得自己进创作间点。HomePage tail 导入成功后弹「立即反推这部分大纲？」→ 确认派发 novel:goto-tab + novel:auto-reconstruct 两事件；CreatePage 监听后者触发既有任务化反推流（Start→轮询→确认→应用，v4.291）；tab 组件常驻监听器必然在位。**测试**=HomePage +1/CreatePage +1（修正 Apply mock 调用计数跨用例累积假阳性）。**门禁**=ci.ps1 全绿、drift OK@660（零绑定面）、版本三处 4.292.0；产物 exe 50,041,344B SHA256=7AE305A5C672BB6852F680CDD7CDEE69909470887428E5E892122AD639A244D2（冒烟 200）。**未做**=反推任务取消绑定/反向角色补建/骨架 AI 丰富化/style.md 编辑入口按反馈。

- **最新发布：v4.291.0（2026-09-14）「拆书导入线：AI 反推大纲任务化（长书后台态）」**——反推是 1-3 分钟长操作（上限 10 分钟），同步绑定页面一关结果就丢——接进 tasks 任务队列：新 tasks.KindOutlineReconstruct（outline_reconstruct）+ handler 注册进启动装配块，跑与同步绑定同一套 outlineReconstructCore（语义零变化），预览 JSON 存任务 Result、Progress 上报阶段；NovelB +2→660（play，锁 479→481）：NovelOutlineReconstructStart（play 空间提交）+NovelOutlineReconstructTaskGet（succeeded 携预览）；explicitOverrides 点名两绑定归 novel（*App 接收者回退会误入 core）；CreatePage 改 Start→3 秒轮询 TaskGet（12 分钟上限）→确认弹窗→应用，队列不可用回落同步绑定（老后端兼容）。**测试**=Go +2（真实 tasks 队列 e2e queued→succeeded→mid 档 6 骨架/诚实报错）+vitest +2。**门禁**=ci.ps1 全绿、drift OK@660、wailsjs 再生、版本三处 4.291.0；产物 exe 50,040,832B SHA256=FC0EF9E4C935601C5543B23BB007EC34375AF58EF0BD0BF09D0C45A5D4F245DE（冒烟 200）。**未做**=反推任务取消绑定；tail×反推串联（v4.292 已销）。

- **非版本刀（2026-09-13）oh-story 蒸馏 T3：生成门补确定性两路（写前大纲契约 + 写后质量体检）**——续 T1（评审 rubric 资产）与 T2（去 AI 味门禁资产），本刀把上游 hooks 的「写前守契约、写后查质量」补进内核（规格 §2 真增量第 1 条：gaea 此前只有事后去味与体检，**无生成前结构契约闸、无生成后确定性质量闸**）。**落地**=①新包 `internal/novelgate`（零 LLM 零网络纯函数）：`OutlineContractIssues`（标题/本章计划/关键要点/情感基调四项齐备性，缺席 S2~S4）+ `ChapterQualityIssues`（空正文 S1 / 超长段落 S3 带行号 / **电报体** S2〔短句占比 >40% 且平均句长 <10 字——把逗号长句拆碎同属 AI 味〕/ 平均句长偏短 S3 / 连续堆叠问号感叹号 S3〔RE2 无反向引用，用「同类标点 ≥3 连」表达〕/ 省略号滥用 S3 / 通篇句号化 S3）；`Issue{Code,Severity,Message,Evidence}` 与评审 rubric 的 S1-S4 同轴、必带证据；阈值常量集中、口径一律 rune。②接线=`RunChapterGate`（章节生成门）新增确定性两路 `outlineContract` + `deterministic`，与 AI 四路（analysis/review/consistency/aiTaste）并列：必出、零成本、失败不阻断（找不到大纲节点返回空表）。**测试**=Go +7（契约齐备与空项分级/空正文 S1/正常文本零误报/电报体/超长段落/标点堆砌与省略号/句号化/证据随行）。**门禁**=本刀自身证据（go build/vet exit 0 + internal/novelgate 7 例 + 生成门定向用例绿）；全量 ci.ps1 未取全绿——**并行线在制品** internal/app/novel_review_handler_test.go:117 为红（与本刀无关）；drift OK@648；**未抬版本**（随下次发版入 exe）。**未做（下刀）**=T4 导入结构映射与篇幅路由；写前契约硬闸（需先有「一键补大纲」）；书级白名单 .deslop-whitelist 与 T2 gates.json 的内核消费。
- **AGENTS.md 八迁分流（2026-09-13，非版本刀，纯文档）**——CI 卫生守卫 WARN（59926B 逼近预算）触发：v4.261.0~v4.255.0 六条入 archive（八迁记录更新），主文件恢复 14 版，水位 59926→47518B；迁移完整性三项断言全过。坑=CRLF 行尾下 node indexOf 锚点不带行尾+写入前验 includes。
- **对比度普查收账（2026-09-13，非版本刀，零代码）**：13 页×明暗两态 WCAG 程序化扫描（.tmp/walk-v4274-contrast.mjs）——dark 15/light 63 告警逐类甄别后**大头为脚本误报**（渐变/图片背景无法合成，目检实际清晰）；真实低对比仅亮态 accent 弱化文本（~10 处 1.5~1.8）+schedule 行号 1.3+weixin purple tag 3.39——全为装饰性文本非正文，**无 AA 硬伤不动**。观察池新增=亮态 accent 弱化文本打磨候选（修则需动 lightFn 令牌，视觉拍板项）。**坑=对比度自动扫描对 background-image/渐变必然误报，告警须逐类目检甄别后才能定刀**。
- **v4.270 留池补验收官（2026-09-13，非版本刀，零代码）**：sin_illustrate live 端到端**全通**——模型真调工具/图片真实落盘（sin/art「雨夜回眸」.png 1.8MB）/轨迹 artifacts 在位/`extra.illustrations['tool0']` 回写画廊可见/过程卡「思考过程·319 字·生成插图」元数据在位。**历史两次失败归因翻案**=上游 grok-4.6 一次性空返回（仅 reasoning 无正文无工具），后端兜底如实报错（sin_handler.go 空正文不落库），重试即成——**非 gaea 缺陷**。清场=故事删+产物图删（壳句柄锁删挂起，杀壳后 PowerShell 删成）。**坑**=①node rmSync 对被占用文件不抛错但删不动（Windows delete-pending），删后必须 existsSync 复核 ②走查脚本清场要放 finally（v4.270 脚本超时 throw 路径跳过清场）。**观察池新增**=sin/art 存 3 张疑似历史走查孤儿图（00:36/05:22/09:34「雨夜站台」走查 prompt 产物），待人工确认删除。

- **品牌资产补齐（2026-09-10，非版本刀）**——2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核），**两个二进制件没跟上**：`build/appicon.png`（1024²）与 `build/windows/icon.ico` 停在 **2026-08-05** 旧视觉（翡翠球体+破土嫩芽+星芒，深蓝底 `#0F172A`）→ **exe 内嵌图标/任务栏/资源管理器/桌面快捷方式全显示旧 logo，只有窗口内 UI 是新 logo（「换了一半」）**。**落地**=① 无头 Edge 渲染 `appicon.svg` →1024×1024 RGBA PNG（**必须传 `--default-background-color=00000000`**，否则圆角外合成不透明白、alpha 静默丢失）；② Pillow 由同一母图出 **7 档 ICO**（16/24/32/48/64/128/256，旧资产缺 24，本次补 Windows 标准全集）；③ `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**）。**验证三条独立证据**=① 几何精度（核心圆 r=50@512 实测直径 **200px**、圆心 (539.5,511.5) 对理论 (540,512)；环 stroke 48 实测线宽 **96px**；圆角 rx=104 对角线首不透明像素 d=**61** 对理论 60.9——零缩放零偏移）；② exe 内嵌图标提取（`ExtractAssociatedIcon`：底板 `#0B1210`/米色核/翡翠环=新「地核 G」）；③ dist bundle 新 logo 独有标记 `gaea-logo-ring`×3、`gaea-logo-light-ring`×3、`0B1210`×2 在册且旧独占色 `#6ee7b7`/`#047857` **零命中**+按旧 SHA256 全仓比对零旧 logo 残留。**门禁**=build exit 0 / 冒烟 200 / **纯资产刀（零前端零 Go 源码改动、绑定面不变）**。**产物**=`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希；中间态 903F9759…C36EC068=只换 PNG+主图 ICO 的一版，小尺寸优化后重建覆盖；原始 59C2B518… 系 v4.208.0 归档值，releases 档案自洽未动）。**坑**=① 无头 Edge 截图默认白底，CSS 里写 `background:transparent` 无效（截图不继承页面背景），透明基线拿旧资产角像素 alpha=0 对照；② **判新旧前须验该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变与**旧** logo 主色，拿它判会把新资产误报成旧的，判据只能用**独有**标记（SVG id 名 / 独占色）；③ `pwsh` 不在 PATH，手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（**非 app 故障，易误判为冒烟失败**），用 `powershell`（build.bat 内已有 fallback）；④ 运行中的 exe 可被 `Copy-Item` 直接覆盖（进程持旧映射），桌面副本无需先关 app，但**已加载实例不会换图标，需重启**。**小尺寸可辨识（同日追加）**=原方案对大图统一降采样，致 16×16 环宽仅 1.5px / 核 3.1px、G 字形糊成深色块；新增派生资产 `build/appicon-small.svg`（环 48→64、核 r50→62、去 halo），ICO **按尺寸分流源图：16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**（若在 24→32 切会跳）；**坑=Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 逐帧 16B + PNG 帧，**256 档宽/高字段写 0**，写 256 溢出单字节）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。**未抬版本**——按 README 发版约定「文档整理、令牌对照、单点样式等小改记入 CHANGELOG，不单独抬版本」，本刀属视觉资产补齐，与 `progress.md` 的「工作空间与文档整理（非版本刀）」同类处置。

- **壳内全板块渲染健康巡检（2026-09-12，非版本刀，随 v4.237 线收尾）**——CDP 起壳逐 rail 页点击+错误边界断言：闲庭七页（闲庭首页/首页/聊天/小说/绘梦/模型中心/角色库）+书斋六页（首页/办公/造价数据库/记忆中枢/模型中心/青鸟）**13 页全绿零接管**；配方=walk-pages.mjs（rail 坐标 DOM 定位+逐页停留+错误文本断言+失败自动截图）；巡检结论=无其他「静默坏死」页面。零代码改动不抬版本。

- **DAG 6.3 壳内真机走查（2026-09-12，非版本刀，60c68a34）**——§6 最后一条余项清账（**§6 余项全清，DAG 6.3 线收官**）：CDP 9333 DOM 断言走查（不点危险操作不打模型）——注入体检 Drawer 真跑=GaeaMemoryEvalRun 真实 wire+真实库**端到端打通**（通过：五条结构不变量全部成立；晨报预载 2 条·104/600、本体 2 引用·181/600、门控四开关、toast；截图 .tmp/eval-drawer.png）；办公流水线区 dag-section 在位（空态引导正确）；全程 exceptionThrown=0/页面渲染出错=0。**观察池新增**=办公记忆库面板「不可用—未配置」与同库体检读出 2 条活跃并存——面板 available 旗与体检取数路径（hubOfficeStore 直连）口径不同，既有语义非本线回归，按需另刀。配方=.tmp/walk-v4243.mjs/.tmp/walk-dag.mjs（rail 悬停展开→按 title 前缀点库入口→按钮文本带计数须前缀匹配；**节点列表/状态徽标在展开层，先点 goal 展开**）。零代码改动不抬版本。














- **v4.174–v4.178（2026-09-09）瘦身 P3 版3/P4 三刀与造价条目匹配键刀**——全文迁入 docs/archive/agents-version-history-2026-09.md（2026-09-11 三迁腾预算，check-docs 指令预算守卫）。
## 执行纪律：默认并发子代理（2026-09-03 用户强化习惯）

用户已把「并发子代理」从点名指令固化为**默认习惯**：后续任务默认按此执行、
无需再次点名「并行使用子代理」。

1. **开工先拆线**：任务含 ≥2 条互不相交的线时，先列出「线 × 文件足迹」再动工；
   2-4 条独立线用并发子代理分头执行（运行环境 4 并发槽位），主代理负责定契约、
   跑全量门禁、集成收口——v4.54-v4.59 的「三线并行 + 主代理收口」即标准形。
2. **单线也倾向派活**：一条独立成刀的任务（调研/实现/测试/文档）若体量超过一轮，
   优先派子代理并发执行，而不是排队串行。
3. **足迹互斥铁律**：线间文件足迹不相交；契约/生成类文件（types.ts / bridge.ts /
   mock.ts / 三语字典 / gen_bindings 产物）必须指定单一负责人，生成动作由主代理在
   所有后端子代理完成后统一执行，防止并发写覆盖。
4. **每刀回写**：刀末把本次分线/收口经验（含教训）写回本文件与 `.gaea/progress.md`，
   让习惯持续强化；若某刀必须串行，收口时说明原因。

## 交互纪律：不许用「等待」换时间（2026-09-10 用户拍板）

用户原话（对上一轮执行的批评）：「后台运行的东西你为什么要等待」「在浪费我的时间」。
耗时本身不可怕，**干等与重复**才是浪费。后续每一轮工作按下办：

1. **后台任务不阻塞**：起后台任务后立刻去做别的事（改码/读文档/跑定向用例/写文档），
   收到完成通知再取结果。**禁止起完就守着等**——那几分钟是白扔的。
2. **重活先报 ETA 再跑**：实测耗时——全量 vitest ≈ **140s**（324 文件 2799 例）、
   `scripts/ci.ps1` 全门禁 ≈ **5–7 分钟**（Go 全量 + 前端 lint/build/vitest）、
   `go test ./...` ≈ **2–4 分钟**。开跑前一句话说清「跑什么、约多久」，用户可否决。
3. **定向优先，全量收尾**：改动后先跑目标文件（3–10s）确认；全量只在交付前跑一次。
   一轮改动 = 中间定向 + 收尾全量，**同一验证不重复跑**（上一轮全量跑了 4 遍=反例）。
4. **超时上限按实际需要写**：不要写 900s 这种夸张数字——那是上限不是耗时，
   只会造成「要卡你十几分钟」的观感。
5. **不把「等待」当进度表达**：进度由已完成的具体产出说话，不由轮询次数说话。
   用户催问时先答「在跑什么、还要多久」，再继续。
6. **能并行就并行**：长任务是可并行的（截图取证 / 读码 / 单测互不依赖），
   串行排队本身就是一种浪费。

> 与上一节的关系：并发是**手段**（把时间省下来），本节是**约束**（别把省下的时间又等回去）。

## 工装：CDP 走查与前端调试（2026-09-10 增补）

- `scripts/cdp-walk.mjs` 支持 **`--target <url 片段>`**（多标签时选定目标页，缺省取第一个
  page）与 **`@文件路径` 传入 JS 表达式**（PowerShell 5.1 传原生命令参数会吃掉内层引号，
  长表达式一律写文件再 `@` 传入）。真机走查配方：无头 Edge
  `--headless=new --remote-debugging-port=9333 --window-size=1440,920 --user-data-dir=<tmp>`
  + Vite dev（`?mock=1`）→ 本脚本 eval/截图。
- **Vite dev 缓存坑**：同一 CSS/源文件在**同一秒内的两次编辑**，mtime 粒度相同会让
  HMR/转换缓存认不出改动（表现为浏览器仍跑旧样式，reload 也无用）。处置=改完
  `(Get-Item file).LastWriteTime = Get-Date` 再 reload。
- **`vitest` worker 上限**：`maxWorkers` 必须按**物理核**（≈逻辑核/2，夹 [4,8]）而不是
  逻辑核——超配会把单用例墙钟拉到 CPU 时间的数倍，重组件首个 `await import` 直接
  撞 `testTimeout` 假红（详见 `frontend/vite.config.ts` 注释）。

## 长期规划（权威，2026 定稿）

- **下一阶段规划 = `docs/gaea-next-stage-plan-2026-09.md`（2026-09-10 活跃指导）**：接棒长期规划（阶段一~四已收官）——阶段五 记忆 OS+上下文编译（主轴）/阶段六 书斋纵深——办公主角（可审计默认化·记忆驱动项目本体·多文件 DAG），进度（DCMA 体检·蒙特卡洛工期带）与跨域 EVM 为两翼/板块并行池/拍板池（信息价接入·MCP 翻案·壳外 computer use）/维持轨。提方向前先对其 §0 在册边界（GoalCard v3.6.0 撤下、MCP/平台化不进规划、不做算量、DSH 独立窗口等）。
- **用户拍板（2026-09-08，收敛计划 §0）：性质路线=A「终极个人工具」**——产品化
  降为期权不作承诺，不为想象中的用户写代码；**LICENSE=私有 All Rights Reserved**
  （根目录 LICENSE），未来开源须另行发布开源许可证覆盖对应模块并与私有部分区隔。
- **瘦身长期总规划（2026-09-08 立项）= `docs/gaea-slim-masterplan-2026-09.md`**：七面
  （认知/资产/结构/轮子/运行/产物/数据知识）×六阶段（P0 基线→P5 维持）；治理规则仍在
  收敛计划。防复发规约自 P1 起生效（新下载走 saveExportBlob/新选取走 pickFile/新 diff
  复用 lib/diff/新 slug 用 strutil.TitleSlug/新解码用 b64ToBytes/NAVIGATE 用 manifest id/
  新域先问「能否是文档」）。
- **用户再拍板（2026-09-08）：工位与乐园并列平级**——办公不是乐园的上位核心，瘦身
  不是「收乐园保办公」；认知面问题=13 板块平铺单一导航面、并列结构未表达。P2 措辞
  已由「乐园折叠」改为「双空间并列落地」，乐园板块在乐园空间内一级可见。
- **唯一权威路线图 = `docs/gaea-nextgen-roadmap-2026.md`**（11 个子代理调研合成）：
  8 板块竞品调研（办公/造价/AI 底座/编程/小说/绘梦/轻语/微信+语音）+ WorkBuddy×灵犀
  深度拆解（§12）+ **版本重定义"双空间"（§10）** + 四层落地（§13 后端/前端/UX/UI）+
  执行计划（§14 阶段 0 地基 → 阶段 1 双空间内核 → 阶段 2 双空间壳 → 阶段 3+ 领域包）。
- **用户拍板：工作与娱乐分开、互不干扰**——工位（办公/造价/编程/资料+工作记忆）与
  乐园（轻语/小说/绘梦/阅读）双空间硬隔离；记忆分区互不检索、模型策略各配各的、
  上下文永不跨界；跨空间仅用户显式发起（如"把乐园封面放进报告"）。
  ~~"陪伴×办公融合"（旧 v4.3）已删除~~。
- **本轮关键纠正**：灵犀 = 金山 WPS 独立 AI 办公 Agent（非阿里/通义系）；
  WorkBuddy = 腾讯云 CodeBuddy 全场景 AI 办公工作台（非 Kimi 系；Kimi Work 是月之暗面的）。
- **i18n 决策（2026 追加）**：采用审计 §405「诚实 zh-only」选项——**壳层 chrome +
  设置外壳三语**（已完成，消灭壳层混合语言根因）；**页面内容层保持 zh 单语**，
  不再逐页铺 en/zh-TW 字典（个人中文工具无国际化受众，~5000 字符边际价值≈0；
  未来需国际化时按 S2.3b WireShape 模式整页迁移）。
- **文档纪律**：docs/ 旧调研/已落地计划已归档至 `docs/archive/`（见其 README）；
  后续会话以本文件 + 长期规划 + `.gaea/progress.md` 为权威，勿引用 docs/archive 结论。
  **新文档必须登记 `docs/README.md`**——守卫 `node scripts/check-docs.mjs` 四查（孤儿登记 /
  `docs/` 悬空引用 / **本文件字节预算 ≤ 65536 B**，超了尾部纪律段就对会话不可见 /
  **含非 ASCII 的 .ps1 必须带 UTF-8 BOM**——2026-09-10 实撞：编辑 ci.ps1 丢 BOM，整条门禁语法错静默不跑），已接入 `scripts/ci.ps1`。
- **执行审计（2026-08-30）**：`docs/archive/audit-2026-08-30-v4-execution-review.md` 记录
  v4.x 全量「承诺 vs 代码」对照——裁决=最小版执行（骨架真、纵深欠账）。红线缺口
  三条（记忆注入跨空间未接线 / 任务分账未启用 / 事件过滤仅 1 处）与补课刀序见该文
  §B/§E；后续每刀验收新增「纵深检查」，发布说明必须列欠账清单。
- **执行状态（v4.8.0 后）**：审计欠账大面收账——读屏纵深（多显示器/OCR 本地
  摘要/截图留档）、intent LLM 兜底分类器（默认关，白名单+置信门+硬超时）、
  生图产物 CardPath 接通、iLink 微信通道离线收敛（限频/下载防线/识别管线/
  防御解析/SendFileCard seam）、全局离线模式总开关（EngineType.IsLocal +
  routeModel 云过滤）、成本知识图谱可视化（BuildGraph + CostGraphView 第 8
  模块，绑定面 533）、实时语音 Realtime S0 铺底（internal/realtime seam）。
  剩余欠账（Realtime S1/S2、iLink 真机窗口、离线模式设置 UI、权限升级请求+
  stubGate 竞态、XlsxPreview 虚拟滚动/生命库可写化=观察项）见
  `releases/v4.8.0.md` 欠账清单。
- **欠账收尾小步（2026-08-30，v4.8.3 后）**：VoiceStart realtime 门小修
  （端到端回复走服务端 response 事件，whisperChatFn=nil 也可启动，拼接
  管线双门逐字节保留）；持久化套件统一（desktop_session 原子写 + archive
  JSONL 单次 Write 落整行）；XlsxPreview 大表格行虚拟滚动（观察项收账，
  300 行以上只渲染可见窗口 ±overscan）。Go 全量绿、vitest 809/809、
  tsc/eslint 0、绑定面 535 不变。
- **下一执行**：v4.8.3 已发布（微信图片双向真协议）；剩余——Realtime 真机
  验证轮（真 key 下端到端对话/打断体感/AEC 实效，S2 骨架已就绪待真机数据）；
  手写体识别质量复测（多模态 Qwen 升级后）；iLink 语音/视频等未探明 item
  维持宁漏勿误静默跳过；生命库可写化=观察项。

## 版本状态（历史存档）

> 2026-09-09 整理：本段逐版历史（v2.x~v4.135，约 2100 行）整体迁往 `docs/archive/agents-version-history-2026-09.md`，按需检索；当前动态见顶部速览，发布物全文见 releases/。

## 项目定位

gaea 是 Windows 桌面端「通用办公」AI 助手（Wails v2：Go 1.26 后端 + React/TypeScript/Vite 前端）。
核心能力：文档撰写、表格处理、格式转换（docx/xlsx/pdf → Markdown）、图表生成、报告拼装、
知识库与记忆中枢、方案编写。品牌定位已从「土壤修复工程办公」全面转为「通用办公」。

## 技术栈与关键约定

- 桌面框架 Wails v2.13（Go + WebView2）；后端事件总线 + 前端 zustand 桥接（bridge.ts → window.go.app.App）
- **绑定面（v2.17.0 起）**：App 不再直接绑定 Wails；429 个导出方法拆 10 个板块门面
  （internal/app/bindings_*.go：CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB，
  纯委托零逻辑改动）。改绑定面方法后用 `go run ./scripts/gen_bindings` 重新生成 +
  `TestBindingsCompleteness` 兜底；前端调用经 gaea/lib/bridge.ts（按方法名路由门面）或
  api/bridge.ts 的 window.go.app.App 兼容代理；旧 wailsjs 导入走 src/wailsjsCompat 重导出；
  wails build 会重生成 wailsjs/go/app/<门面>.js
- 单模型架构：一个 executor 完成规划与执行，无独立规划器；任务/技能子代理走 `task` / `run_skill`
- 内置工具精简为 17 个核心工具（v2.4.3 起）：文件/命令、网络、任务、记忆/知识、技能、format_convert、chart_gen
- 文档能力交给 ModelScope 技能：docx / pdf / xlsx（安装在 `~/.codex/skills` 与 `.gaea/skills`），
  转换引擎共用 `internal/office/docmd`（format_convert 工具与预览面板同一实现）
- 内置子代理技能：format-convert / chart-builder / doc-assemble
- 记忆系统：SQLite（`%APPDATA%\gaea\Hephaestus.db` facts 表，按项目 slug 隔离）+ 文档记忆（AGENTS.md 层级）
- 环境依赖：LibreOffice（soffice）、node 全局 docx、Python 3.13（lxml/openpyxl/pypdf/pdfplumber/reportlab/pandas/matplotlib 等）
- 本地 AI 底座：**Herdsman**（localhost:8080/v1，~110GB 模型：35B 对话 ×2、zimage-turbo、voxcpm2、
  mineru、embedding/reranker、paddleocr、sherpa-onnx 等）；gaea 的聊天/视觉/检索/OCR/解析/ASR/TTS/生图/翻译
  全部依赖它，herdsman 升级可能破坏契约——用 App.HerdsmanProbe 启动探测

## 发布流程（2026-08-14 修订：补版本资源步骤）

1. 更新 CHANGELOG.md / README.md（版本表）/ wails.json（productVersion）/ releases/README.md（版本表）
2. **同步版本资源**：`build/windows/info.json` 是 Wails 生成版本信息的模板（fixed 段必须含
   `product_version`，否则 exe 的 ProductVersion 为 0.0.0.0）；根目录 `versioninfo.rc` 是遗留物，一并更新以免误导
3. 构建（本沙箱：`cd frontend; npm run build` → `wails build -s`；本机：`cmd /c build.bat`），
   产物 build/bin/gaea.exe（同时复制到桌面）；本机 build.bat 已内置真实退出码检查 +
   默认自动冒烟（.tmp 临时副本 → scripts/smoke.ps1，18999 /api/health 200，失败即停；
   `build.bat skip-smoke` 可跳过，发布前不得跳过）
4. 复制 exe 到 `releases/gaea-v<版本>.exe`，生成 `releases/SHA256SUMS-v<版本>.txt`；
   **本地产物只保留最近 5 版**（2026-09-10 用户拍板）——发版后删掉第 6 新的那一版，
   更早版本的身份以 `SHA256SUMS-vX.Y.Z.txt` 为准（旧 exe 不入库）
5. 写 `releases/v<版本>.md` 发布说明（含 SHA256 与冒烟结果），更新 releases/README.md 版本表
6. 冒烟：`scripts/smoke.ps1 -ExePath releases\gaea-v<版本>.exe`（/api/health 200 即通过）
7. 更新 `.gaea/progress.md` 进度记忆与本文件（版本状态）

## 沙箱环境备忘（2026-08-14 整理，详细版见 docs/2026-08-14-sandbox-environment-notes.md）

**防止重蹈覆辙的四条铁律**：
1. `go telemetry off` 已持久生效；构建缓存写入问题随 danger-full-access 策略解除，无需再覆盖 GOCACHE
2. **wails build 前端编译会挂起**（wails 捕获前端输出走管道）——必须 `cd frontend && npm run build`
   再 `wails build -s`（-s = 跳过前端编译，9s 完成）
3. `go test ./...` 单进程树会被 harness 终止、个别测试二进制偶发 `fork/exec Access is denied`——
   逐包验证 + `scripts/test-all.ps1`（逐包/重试/状态续跑）；exec 拒绝用 `go test -c` 手动运行证明代码无恙
4. .ps1 脚本必须 UTF-8 带 BOM（否则 powershell.exe 按 GBK 解析报错）；npx 用 `& 'C:\Program Files\nodejs\npx.cmd'`

## 本地 TTS 引擎（重要记忆，勿遗忘；2026-08-09 整理）

> ⚠️ **VoxCPM2 已于 v2.6.9 移除**：实测不达标（耗时长、音色男女混乱、克隆不稳定）。
> 下方 VoxCPM/Vulkan 相关方法保留为「已废弃教训」，勿重新安装；当前本地 TTS 为 CosyVoice2。
> 注：herdsman 侧实测 voxcpm2 可用（冷启动约 50s，不支持预设音色），qwen3-tts-* 未安装。

本机（Radeon 8060S 核显 / 64GB 统一内存 / Windows）本地 TTS 有两条引擎线，gaea 只连 OpenAI 兼容 8020/8010。

### ~~VoxCPM2~~（已移除 v2.6.9，以下为废弃记录）

- `8030`：主后端 `C:\AI\llama-omni\build\bin\llama-tts-server.exe`（llama.cpp-omni，C++/ggml + Vulkan）
  - 模型：`C:\AI\llama-omni\models\VoxCPM2-BaseLM-Q8_0.gguf`（1.65GB）+ `VoxCPM2-Acoustic-F16.gguf`（1.74GB）
  - 8060S 识别 `KHR_coopmat + bf16`，全量 29 层 offload Vulkan0，加载约 2s
- `8021`：备胎 ROCm PyTorch（`C:\AI\voxcpm\server.py` + `VOXCPM_PORT=8021`）
- `8020`：适配层 `C:\AI\voxcpm\adapter.py`（FastAPI，gaea 入口，契约不变）
- 一键启动：`C:\AI\voxcpm\start_voxcpm_stack.ps1`（8030/8021/8020）

### CosyVoice2（端口 8010）

- `C:\AI\cosyvoice\server.py`：LLM 段 GGUF + Vulkan（`gguf\cosyvoice_f16.gguf`），flow 段 ONNX + DirectML（5 步）
- 启动：`C:\AI\cosyvoice\start_cosyvoice.bat`；约 14s 加载+预热，短句 ~1.5s

### 音色（两引擎统一 4 个，火山引擎 Speech-AI-Forge-spks 录音室样本）

- 中文女 `zh_female.wav`（f0≈221Hz）、中文男 `zh_male.wav`（f0≈133Hz）
- 英文女 `en_female.wav`（f0≈191Hz）、英文男 `en_male.wav`（f0≈109Hz）
- 参考音频 ~7s / 16kHz；转写在 `C:\AI\voxcpm\voices\_meta.json`

### 本次优化方法（AMD 核显提速教训，勿重蹈覆辙）

1. 不要再用纯 ROCm PyTorch 追赶速度：iGPU 共享内存架构下 ROCm 与 CPU 基本相同（RTF ≈1.06~1.12）；
   Vulkan + ggml 的 GEMM/coopmat 才是突破口（克隆 RTF 0.65~0.84）
2. 构建：MSYS2 UCRT64，`cmake -B build -DGGML_VULKAN=ON -DGGML_NATIVE=ON`
3. 坑 1（端口绑不上）：server-voxcpm2 会构造 SSLServer，空证书导致 is_valid_=false 任何端口 bind 失败；
   本地回环不需 TLS，改普通 httplib::Server
4. 坑 2（克隆近静音）：AudioVAE 参数特征必须 frame-major（`ggml_cont(latent)`），
   不能 `cont(transpose(latent))`（dim-major）
5. 坑 3：llama.cpp-omni 的 CLI `-r` 克隆偶发偏静音，HTTP server 路径正常；生产走 server
6. 坑 4：VoxCPM Python 长文本 CFG 2.0 会「静音+整段重试」（RTF 4.8~7.8），CFG 1.5 稳定；
   C++ server 端用 max_steps 限制解码上限
7. 网络：HuggingFace LFS 直连/hf-mirror 都不通，ModelScope 直链快（8.6MB/s）
8. 实测：短句克隆 RTF 0.65~0.84（6 步）、语音设计 0.57~0.60；同 seed 输出确定

### 详细记录

- `docs/archive/2026-08-09-voxcpm2-integration.md`（VoxCPM2 全部历程）
- `docs/archive/2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md`（CosyVoice GGUF 提速）

### 自动启动（当前仅 CosyVoice2）

- gaea 启动时后端 ensure cosyvoice；模型中心 TTS 模型卡片「启动」按钮 → `App.StartLocalTTSService(engineId)`；
  引擎连接测试兜底 ensure（等约 8s）；TTS 合成前兜底 ensure
- 实现：`internal/app/tts_service.go`（core.ensureLocalTTSService 幂等 + 异步轮询，emit `tts-service-status`；
  CosyVoice 直接 python server.py，隐藏窗口 CREATE_NO_WINDOW）
- 端口探测：CosyVoice2 `8010/v1/models`

## 已知注意

- 角色库剧照默认跟随绘梦（ImageBackend/ImageModel），可在模型中心单独绑定
- 文生视频依赖本地 ComfyUI 安装 LTX-Video 模型
- 里程碑：2026-08-12 完成通用办公全面打磨（显示/布局/安全三线：成本库/记忆/知识库/技能写入全部硬性确认
  含 yolo、子代理路径；子代理不再继承持久化写入工具）
