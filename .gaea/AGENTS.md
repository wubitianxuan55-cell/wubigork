# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 更早版本见 `docs/archive/agents-version-history-2026-09.md`（覆盖 v4.173–v4.146 与 v4.49 及更早）；**v4.50–v4.145 区间本仓只在 `CHANGELOG.md` / `releases/` 有记录**（速览逐版滚出后未回填）。本段只留最近 14 版：本文件曾达 104 KB 超工作区指令预算（65536 B），尾部纪律/规划段一度对后续会话不可见，2026-09-10 二次分流。
- **最新发布：v4.289.0（2026-09-14）「oh-story T6：书级文风档案（style.md 协议对齐）」**——oh-story 线最后一项，**T1~T6 全部收官**。书级文风档案 `<项目目录>/style.md` 注入章节生成上下文（buildChapterContextSections 文风区段，伏笔→文风→世界观共享预算）：协议逐条对齐上游 style-resolution——无文件不建占位、一句偏好也有效、空白/纯标题/待补充不算、区段头声明「只约束表达维度，事实与信息边界以细纲为准」（事实与表达分开裁决）、超长截断 1200 rune；白名单 `.deslop-whitelist` 补上游注记格式兼容锁（`# 来源：…；用途：…` 整行忽略）；与文风指纹互补（统计基线 vs 显式偏好）。**测试**=app +3 + novelstyle 注记断言。**门禁**=ci.ps1 全绿、drift OK@658、版本三处 4.289.0；产物 exe 50,027,520B SHA256=4F6E8CB200FA92AF5403C905497904F02C40CE56BB63E84D25BFCE907F840442（冒烟 200）。**未做**=style.md 编辑入口按反馈；拆书线长书后台任务态/角色名→ID 匹配/tail×反推串联。
- **最新发布：v4.288.0（2026-09-14）「oh-story T4：篇幅路由与骨架聚合——长书反推从 200 章细纲变卷级粗纲」**——设计拍板三档骨架：短篇 <3 万字或 <5 章=章级细纲现状；中篇=每 10 章聚合；长篇 ≥80 万字或 ≥300 章=每 30 章聚合（阈值沿上游 length-routing，常量在 internal/bookimport/skeleton.go 改之即改路由）。**① 纯函数层**=RouteTier（长篇信号优先/零章短篇）+SegmentSize+AggregateSkeleton（「第X-Y章」+章标题拼接摘要 200 rune 截断）。**② 反推**=NovelOutlineReconstruct 算 tier/segmentSize 进预览；中/长篇预览 Items 替换为骨架条目（chapterFrom/To）；短篇逐字节不变。**③ apply**=骨架条目新建卷级参考节点（seg-NNN/planned/根级），**重复应用先移除旧 seg-* 再落新**（幂等不堆积）；短篇零变化；混合载荷兼容。**④ 前端**=CreatePage 确认弹窗按 tier 分叉文案。**测试**=bookimport +3 + app +2（中篇 60 章聚合/短篇回归）+既有 2 例原样过。**门禁**=ci.ps1 全绿、drift OK@658（**修正 v4.287 bindingNames 漏登记 ImportNovelBookEx**——彼时检查读到旧数误报 OK，本刀按生成器流程重产）、版本三处 4.288.0；产物 exe 50,025,472B SHA256=9C78EDB2C5B04FE164A96FD8AB008266B32474A596A44F6CC6370E0A9C580C34（冒烟 200）。**未做**=骨架 AI 丰富化按反馈；oh-story T5/T6；拆书线长书后台任务态/角色名→ID 匹配/tail×反推串联。
- **最新发布：v4.287.0（2026-09-14）「拆书导入 tail 模式出口：长书只取末 N 章入书架」**——拆书导入线在案欠账（引擎 applyExtractMode 自 v4.279 就绪但绑定只走 full）；tail 产品意图=用末尾几章反推「这本书接下来该怎么写」。**① 后端**=新绑定 `ImportNovelBookEx(filePath,title,genre,style,extractMode,tailChapters)`（NovelB +1→**658**）：full|tail 非法值起跑即报错；ImportNovelBook 原签名保留=full 委托；opts 全链透传 parseNovelFile/parseTextChapters；引擎语义照旧（5 倍数取整/>50 或 ≥总章数降级全本/裁剪告知进 warnings/落库从 1 重编号）；EPUB 无末尾语义 tail 时如实告知按全本。**② 前端**=导入向导「提取范围」Select（全本/末 5~50 章）+提示语+HomePage 传参复位。**测试**=Go +2（末 10 章重编号+裁剪告警/降级+非法模式+full 回归）+vitest +3（选项表/当前值/EPUB 提示）。**门禁**=ci.ps1 全绿、drift OK@658、版本三处 4.287.0；产物 exe 50,015,744B SHA256=5C212AD239AF68E5E842887C8605961209CC48B7684F44B5D5CF9107520AA7F8（冒烟 200）。**未做（下刀）**=oh-story T4 需先定聚合粒度设计（章级 vs 卷级，建议交拍板）；长书后台任务态/逐章预览/角色名→ID 匹配；tail×AI 反推串联按反馈。
- **最新发布：v4.286.0（2026-09-14）「oh-story T2 内核消费：模式级判定进 novelstyle + 书级白名单豁免」**——oh-story 线 T2 的内核消费刀（此前只落 gates.json/ai-patterns 资产层模型执行）；与 T1 平台评审/T3 生成门三线汇合，零新绑定 drift OK@657、前端零改动。**① 模式资产**=新 `internal/novelstyle/patterns.json`（go:embed）：negationFlipHigh（门禁 B 同句高置信四式 RE2，命中 high）+ explanatoryMarkers（门禁 G advisory：之所以/这意味着/殊不知/句首原来等，命中 low+**封顶 5 条**）；语义裁决留技能；覆盖同 words.json 纪律（惯例目录整体替换、坏正则整表拒绝不换出）。**② 打分**=规则 10/11 进 ScoreText（rune span、权重口径不变）。**③ 书级白名单**=`<项目目录>/.deslop-whitelist`（无文件不建空表）：`ApplyWhitelist` 片段互含摘 issue+重算分数；`DeSlopRewriteEx` 出现级豁免（上下文命中原样保留）+after 同口径复测（「分数不降才落盘」不变）；`DeSlopRewrite` 原签名保留。**④ 接线**=一键去味/生成自动去味+done 体检分/对照体检三处三件套齐挂（词表+模式+白名单），自动手动同口径。**测试**=novelstyle +6 + app +2（v4 场景路由夹具：无白名单替换、授权豁免原样保留）。**门禁**=ci.ps1 全绿、版本三处 4.286.0；产物 exe 50,010,624B SHA256=2330CBC9F7C224BF055813C7DA132E64A864B000CF807E1C54FE591379FA1BD4（冒烟 200）。**未做**=门禁 C/E/F 留技能（防过度设计）；白名单 UI 按反馈；oh-story 余项 T4/T5/T6/写前契约硬闸。
- **最新发布：v4.285.0（2026-09-14）「书源取书 t3 收官：搜索历史 + 泛搜索引擎规则可编辑」**——t3 余项两件，**书源线功能项收官**（t1 后端→t2 前端入口→v4.283.1 走查补刀→t3 重试+历史+规则编辑）。**① 搜索历史**=新 `utils/bookSearchHistory.ts`（localStorage trim/去重最近在前/封顶 8/损坏当空/存储禁用兜底）+ Modal 历史 chips 点击即搜+清空+成功即记录。**② 引擎规则可编辑**=新 `BookSearchEnginesModal`（启用开关/名称/SERP 地址/查询塑形/两个选择器/删除/新增模板行；高级字段 linkParam 等按行原样往返不丢）+ 绑定 `NovelBookSourceEnginesGet/Save`（NovelB +2→**657**，锁 477→479）：Get 缺失=空清单、损坏显式报错；Save **整体替换+fail-closed**（逐条 Validate 同装载纪律、引擎名去重、空清单拒绝、临时文件+改名原子落盘、校验失败原文件不动；规则每次搜索现读保存即生效）。**测试**=Go +1（回环+五类 fail-closed+原文件不破坏）+vitest +10（历史 util 6/编辑器 4/集成 1）。**门禁**=ci.ps1 全绿、drift OK@657、版本三处 4.285.0；产物 exe 49,975,296B SHA256=2100FA0C0E5E46F33EBBEF30892FCCB9A45F61017161AEB514102B54C0A0ED3C（冒烟 200）。**观察池**=SERP 首搜抖动、线上源站漂移、真机补下/编辑器走查；sin 书源线 t2~t5 归并行线。
- **最新发布：v4.284.0（2026-09-14）「书源取书 t3：失败章重试（一键补下，项目端续编落库）」**——t3 首项（规格 §5 观察池驱动；整本下载零星失败章此前只能整本重来）。**① 引擎**=新 `booksource.Engine.DownloadChapters`（显式 URL 清单抓章；编排水抽 `fetchChapters` 共用，纪律零变化）。**② app**=新绑定 `NovelBookSourceImportChapters(source, projectPath, chaptersJSON)`（NovelB +1→**655**，play，锁 476→477）：清单=导入 done 事件 Failed 原样回传（会话内存活，关闭即弃，§6 不做持久化照旧）；预检 fail-closed 三查；追加落库=现有最大 OrderIndex 续编章号+大纲 imp-NNN 追加既有节点原样保留（**并发写核实**：outline.Agent 写路径每次重读 outline.json 无缓存副本，外部追加安全）；进度/取消全复用（同通道 +append-done 终态，同登记簿）。**③ 前端**=BookSearchModal 失败面板（失败章直显+「重试补下 N 章」+进度可取消+完成态）；**关闭权收归组件**（全部成功自动关/带失败留面板，父层不再代关）；HomePage.onAppended 刷新书架。**测试**=Go +2（清单保序/空正文进 Failed/进度成功数语义；续编与大纲追加/既有保留/全败不动大纲）+vitest +3（面板渲染/重试参数逐参对齐/append-done 回调/无失败自动关）。**门禁**=ci.ps1 全绿、drift OK@655、版本三处 4.284.0；产物 exe 49,975,296B SHA256=2100FA0C0E5E46F33EBBEF30892FCCB9A45F61017161AEB514102B54C0A0ED3C（冒烟 200）。**未做**=t3 余项搜索历史/规则可编辑（按反馈）；失败清单持久化（有意不做 §6）；真机补下走查（复用 v4.283.1 假站配方改 fail 计数）。
- **最新发布：v4.283.0（2026-09-14）「书源取书 t2：书架「在线搜书」入口（搜索→候选→目录预览→范围下载→进度→入书架）」**——续书源 t1（四绑定非版本刀），把 `NovelBookSource*` 接成书架可用取书链（规格 `docs/gaea-novel-booksource-import-2026-09.md` §5 t2 版本刀；实施记录 §9）。**① 入口**=书架工具条「在线搜书」（与导入小说并列；零新绑定 drift OK@654、不动原罪）。**② BookSearchModal 自包含流程**=候选表书源聚合+泛搜索同表（**hasRule=唯一可导入通道**，免规则标「正文不可解析·仅参考」禁用）；各源失败 warnings 直显；目录预览（总数+首8末4截断样例）；范围下载（起止章默认全本钳 [1,total]）；进度订阅 `novel-import-progress:<jobId>`（events.ts +`BOOK_IMPORT_PROGRESS`/`bookImportProgressChannel`，后端事件常量 24→25）+取消（canceled 转写「已取消导入」）；error 面板直显可重试；done 失败章清单提示「N 章未入库」；导入期关闭受锁、关闭必退订。**③ 报告面收口**=新 `utils/novelImportReport.ts`（strategyLabel 补 **booksource→在线书源**；formatImportSuccess/Warnings），文件导入内联逻辑改共用。**④ mock 诚实空态**=mock/novel.ts 四方法（搜索空候选+说明/导入如实抛错）。**测试**=vitest +15（BookSearchModal 7：bridge mock+window.runtime 桩投事件；报告 util 6；HomePage 入口 1；events 频道 1）。**门禁**=ci.ps1 全绿、版本三处 4.283.0、wailsjs 再生（gitignore 构建产物）；产物 exe 49,951,744B SHA256=2D42376FF1B23C3EEB165473DE5773343B74AC7DAA751BCE0DA64CBA0266ED47（冒烟 200）。**未做（下刀）**=t3 失败章重试/搜索历史/规则可编辑（按反馈）；sin 书源线 t2~t5 归并行线。**走查补刀 v4.283.1（2026-09-14）**=t2 真机走查（CDP+本地假站+临时规则，走查完即删）抓到 **P0 零注入 nil-Fetcher panic**（`newCrawler` 只兜底 Sleeper/Rand，四绑定全传 `Options{}`；单测全注入假站故零暴露）→ 引擎一处兜底四绑定修净 + Go 2 例（零注入三构造路径 fetch 必非 nil 回归锁 / 零值 HTTPFetcher 本地 httptest 取页）；复走查全链过（搜索同表 HasRule 打标→选书→目录 25 章样例→范围导入落库→磁盘核验→零前端错误）；观察注记=首搜零候选复搜稳定的 SERP 首请求抖动归漂移池；exe 49,951,744B SHA256=66A9765143487D3D8324AE08AAED5F9F6846B2F9151D51CF46D4DCB4682A174C（版本三处 4.283.1）。
- **最新发布：v4.282.0（2026-09-13）「oh-story 蒸馏首刀接线：平台质量评审（rubric 引擎 + 创作间面板）+ 生成门确定性两路直显」**——把已在库的三份资产（T1 评审 rubric / T2 去 AI 味门禁 / T3 生成门）接成**用户可用的一条链**（规格 `docs/gaea-novel-ohstory-distill-2026-09.md`；上游 oh-story-claudecode=MIT，机制重推导 + 知识资产随附许可）。**① 引擎**（新 `internal/novelreview`，零 LLM 零网络纯函数）=15 个可机械判定维度（字数区间/开篇钩子/章尾钩子/预告式收尾/情绪节点密度/爽点密度/段落节奏/对话占比/标点节奏/破折号/格式合规/人称视角/主角存在感/金手指提及/字数表述核对），每维 `PASS|WARN|FAIL|SKIP` + S1~S4 + **原文证据（rune 区间 + 段落号）** + 改法，结论 `APPROVE|CONCERNS|REJECT` 同 rubric 门槛；**语义维度不下结论**，缺外部数据显式 SKIP 说明原因（**不静默给 PASS**）。**② 数据资产**=`internal/novelreview/rubric.json`（四档位 通用/番茄/起点/知乎盐言 + 冲突/情绪/悬念/爽点/金手指/预告收尾词表），可被 <工作区>/.gaea/skills/novel-review/rubric.json **整体替换**（同 novel-deslop 词表纪律），fail-closed 校验拒坏资产。**③ 接线**=NovelB +2（`NovelChapterReview`/`NovelReviewPlatforms`，648→650，drift OK）+ 创作间 rail「平台评审」面板（档位选择/结论/S1~S4 计数/逐维按严重度排序/证据摘录/黄金三问）+ `NovelInspector`「章节体检」直显 T3 两路（写前契约 N 项待补 / 写后硬信号 N 项）。**④ 口径对齐**=`.gaea/skills/novel-review/rubrics/generic.json` 18→**26 维**（+8 引擎扩展标 `measurable/engine`）+ `deterministicEngine` 段 + SKILL.md「确定性引擎」节（**资产=评审协议、引擎=可机械判定子集，共用维度 id 与 S1~S4**）。**测试**=Go +16（novelreview 12 / app 4，含「合格章节整体 APPROVE」集成夹具与覆盖文件 fail-closed）+vitest +10 + spaceBindings 锁 470→472。**门禁**=ci.ps1 全绿、drift OK@650、版本三处 4.282.0；产物 exe 49,597,440B SHA256=83F5BFD603DB5EC1B091E006F8CACB85B37C3283FF653CCED669F44A9628EAD9（冒烟 200）。**未做（下刀）**=T2 内核消费（`gates.json` 模式级门禁进 novelstyle）/ T4 篇幅路由 / T5 角色卡资产 / T6 风格档案协议 / 写前契约硬闸。
- **非版本刀（2026-09-13）oh-story 蒸馏 T3：生成门补确定性两路（写前大纲契约 + 写后质量体检）**——续 T1（评审 rubric 资产）与 T2（去 AI 味门禁资产），本刀把上游 hooks 的「写前守契约、写后查质量」补进内核（规格 §2 真增量第 1 条：gaea 此前只有事后去味与体检，**无生成前结构契约闸、无生成后确定性质量闸**）。**落地**=①新包 `internal/novelgate`（零 LLM 零网络纯函数）：`OutlineContractIssues`（标题/本章计划/关键要点/情感基调四项齐备性，缺席 S2~S4）+ `ChapterQualityIssues`（空正文 S1 / 超长段落 S3 带行号 / **电报体** S2〔短句占比 >40% 且平均句长 <10 字——把逗号长句拆碎同属 AI 味〕/ 平均句长偏短 S3 / 连续堆叠问号感叹号 S3〔RE2 无反向引用，用「同类标点 ≥3 连」表达〕/ 省略号滥用 S3 / 通篇句号化 S3）；`Issue{Code,Severity,Message,Evidence}` 与评审 rubric 的 S1-S4 同轴、必带证据；阈值常量集中、口径一律 rune。②接线=`RunChapterGate`（章节生成门）新增确定性两路 `outlineContract` + `deterministic`，与 AI 四路（analysis/review/consistency/aiTaste）并列：必出、零成本、失败不阻断（找不到大纲节点返回空表）。**测试**=Go +7（契约齐备与空项分级/空正文 S1/正常文本零误报/电报体/超长段落/标点堆砌与省略号/句号化/证据随行）。**门禁**=本刀自身证据（go build/vet exit 0 + internal/novelgate 7 例 + 生成门定向用例绿）；全量 ci.ps1 未取全绿——**并行线在制品** internal/app/novel_review_handler_test.go:117 为红（与本刀无关）；drift OK@648；**未抬版本**（随下次发版入 exe）。**未做（下刀）**=T4 导入结构映射与篇幅路由；写前契约硬闸（需先有「一键补大纲」）；书级白名单 .deslop-whitelist 与 T2 gates.json 的内核消费。
- **最新发布：v4.281.0（2026-09-13）「拆书导入 P1 接线：AI 反推大纲（预览载荷 + 幂等落库 + 创作间入口）」**——续 v4.280.0（反推引擎），把 t2 P1 接成**用户可用的一条链**（规格 §8.2/§8.3 首刀）。**后端**（新 `internal/app/novel_import_ai.go`，绑定 646→648）=`NovelOutlineReconstruct`：读当前工程章节（标题取大纲节点、正文走 `ReadChapterAsStitch`，上限 200 章）→ Stage1 立项反推（模板 `book-import-project`，采样前 3 章 ×2000 rune）→ 分批章节大纲（`book-import-outline`，batchSize=5、**逐批独立降级**）→ `OutlineReconstructPreview`（aiUsed + 立项五字段 + 逐章 summary/scenes/characters/keyPoints/emotion/goal + warnings），**零落库**；`NovelOutlineReconstructApply(itemsJSON)` 按**章号**命中既有节点写 summary/scene_ideas/key_points/emotion，**幂等**（同载荷连跑两次一字不差）、不新建不删除、**不碰章节正文**、角色名只进预览（角色 ID 匹配另刀）；空载荷/全不匹配**显式报错不静默成功**。**降级诚实**=无模型或解析失败逐级回落规则兜底并写 warnings（aiUsed=false），单批失败只影响该批。**前端**=CreatePage rail「AI 反推大纲」→ 反推（长书约 1-3 分钟）→ 确认弹窗（推断题材/视角/目标字数 + 前两条告警 + 明确「不覆盖正文、可重复执行」）→ 应用 → 刷新大纲 + rail 消息。**接线**=NovelB +2、bindingNames 648（drift OK）、spaceBindings 归 play（锁 468→470）、wailsjs 由 `wails generate module` 再生。**测试**=Go +2（无模型全规则兜底且逐章字段完整 / 幂等与作用域：只写命中章、未命中不动、连跑两次字节一致、空载荷与全不匹配显式报错）+vitest spaceBindings 4 例 + CreatePage 9 例。**门禁**=ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@648、版本三处 4.281.0；产物=exe 49,455,616B SHA256=7FB09E36…2A693（releases/gaea-v4.281.0.exe + SHA256SUMS-v4.281.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做（下刀）**=导入向导逐章预览 UI（核对/单章编辑后应用）/长书后台任务态（`tasks` 表 `KindBookImport`）/角色名→角色库 ID 匹配/tail 模式出口。
- **最新发布：v4.280.0（2026-09-13）「拆书导入 P1 引擎：大纲反推编排 + Prompt 模板移植 + 输入节稳定序」**——续 v4.279.0（t2 P0 解析引擎），本刀落 t2 P1「AI 反推」**引擎层**（规格 `docs/distill/02-book-import.md` §8.2/§3.7/§3.8；零 LLM 依赖、全部可单测，真模型编排与预览 UI 下刀接线）。**落地**=①反推编排引擎（新 `internal/bookimport/reconstruct.go`）：具名契约 `ProjectSuggestion`/`OutlineStructure` + **逐字段归一化**（rune 截断；characters 逐项须对象且带 name、type 仅 character|organization；**title/chapter_number 强制用输入值**）+ **位置对齐**批量归一化（按输入顺序逐位取 ai[i]，非对象槽位 fallback——AI 少返/乱序不错章）+ 数量不符**整批回退规则结构**断言式防线 + 规则兜底逐字对齐规格（summary 取正文首句）+ 视角归一（中文三值 + 11 英文别名）+ target_words <1000 回落/>3e6 夹取；②`CallJSON` 负反馈重试（**期望类型显式 object|array**、失败原文**截 200 字**注入、传输错误不重试、ctx 取消零调用）+ `ExtractJSON` 容忍 markdown 包裹；③新增 RTCO 模板 `prompts/book-import-project.json` / `prompts/book-import-outline.json`；④**补掉规格点名的引擎缺陷**：RTCO `BuildUserPrompt` 原按 Go map 遍历输入节（`internal/prompt/prompt.go:194`）→ 同模板两次渲染字节不同、长文本无法稳定置尾、供应商前缀缓存无法命中 → `InputDef` 增 `order`，按 **Order 升序 → key 字典序**稳定排序，新模板长文本固定置尾（order=9）。**测试**=`internal/bookimport` +9 例（视角别名矩阵/立项字段回落与夹取/位置对齐与强制标题章号/逐字段截断与 type 归一/兜底取首句/重试提示要素/类型不符报错/传输错误不重试与 ctx 取消/JSON 容错与采样）+`internal/prompt` +2 例（**同模板连续 20 次渲染字节一致**、order 覆盖 key 序）。**门禁**=ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646（零新绑定）、版本三处 4.280.0；产物=exe 49,405,440B SHA256=0F9BF89A…8EF4B（releases/gaea-v4.280.0.exe + SHA256SUMS-v4.280.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做（下刀）**=app 绑定 `NovelImportReconstruct*`（真模型编排 + 预览载荷）/导入向导 UI（预览→应用，含 tail 出口）/幂等落库与三件套（§8.3，`tasks` 表 `KindBookImport`）。
- **最新发布：v4.279.0（2026-09-13）「拆书导入 P0：三级分章 + 5 级编码链（MuMu 蒸馏 t2 首刀）」**——续 v4.278.0（t1 共享契约），本刀落 t2「拆书/导入反推」**P0 阶段**（规格 `docs/distill/02-book-import.md` §8.1，**纯规则零 AI**）。**改前短板**=导入只有强标题正则（识别不到即退化成单章「全文」）；编码链 `utf8.Valid→GB18030→GBK` 缺 utf-8-sig 与 Big5，且 GB18030 几乎不报错 + 二次校验**静默放过误判**。**落地**=①新包 `internal/bookimport`（纯规则零 IO）：`Decode` 五级链 + UTF-16 BOM + **候选含替换符即判误判续探**（实测 GB18030/GBK 解 Big5 字节不 error 只出「材�彻」乱码）/`Clean` 六步顺序敏感清洗（全角空格→两半角空格，不是删除）/`Split` 三级切分（**强标题存在时不叠加弱标题**、弱标题需 **≥2 候选**——否则正文短行被切碎、整篇短文本被当成一章标题；无标题 >5000 字走兜底窗口：3000~5000 内取最靠后句读边界，找不到才硬切，标题伪造第N章）/`tail` 裁剪（5 倍数向上取整、>50 降 full）/四类告警（过短 <300·过长 >12000·标题重复·裁剪告知）；②app 接线（`parseTextChapters` 委托新引擎并带出 `bookimport.Report`；`NovelImportResult` 增 `encoding`/`split_strategy`/`warnings`，与既有 snake_case 形状同风格）；③前端导入成功文案直显「编码 · 切分策略」+ 告警弹窗（≤2 条 + 计数）。**有意偏离 MuMu（两处，已注释在码）**=首标题前正文 <200 字**并入首章**不丢（≥200 才成独立「前言」章）；阈值一律 `utf8.RuneCountInString`（MuMu 用 `len()`=字节，中文失真）。**测试**=`internal/bookimport` 17 例（含「Big5 不被 GBK 遮蔽」回归 / BOM / UTF-16 / 兜底不 panic / 清洗六步 / 三级切分 / 窗口上界 / 三类告警 / tail 取整与降级）+`internal/app` 导入 5 例（BOM / GB18030 / 无标题 8000 字窗口 / 弱标题 3 章 / e2e 报告字段）。**门禁**=ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646、版本三处 4.279.0；产物=exe 49,397,760B SHA256=D8947306…7659E（releases/gaea-v4.279.0.exe + SHA256SUMS-v4.279.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**未做（下刀）**=AI 反推编排（字段级归一化 + JSON 负反馈重试）/幂等落库与任务态（`tasks` 表 `KindBookImport`，禁用内存 dict）/导入向导 UI（tail 引擎已就绪，当前仅 full 可达）。
- **最新发布：v4.278.0（2026-09-13）「小说域 v2 共享契约落库 + 伏笔一致性体检（含 wire 口径纠偏）」**——两条线汇合收口：①并行池小说线续刀=**伏笔一致性 Linter**（plan §4 并行池小说行后半：文风指纹 v4.277.0 已落）；②另起的 MuMuAINovel 蒸馏实施线（规格 `docs/distill/`）先落 t1 共享契约后**停摆**，用户确认后由 Codex 侧接管，把在制品 + t1 契约一并落库。**落地**=①伏笔一致性体检（`internal/app/novel_foreshadow_lint_handler.go`：ordering/status-mismatch（partial 豁免）/dangling/stale≥10 章（长线豁免）/duplicate 五类确定性检查 + 报告 totalChapters/items/planted/hinted/revealed/longTerm/findings，空登记=空报告正常态；`ForeshadowPanel`「一致性体检」概要行+severity Tag 轻/中/重直显；绑定 645→646）②t1 契约（`internal/types/`：伏笔并集 6 态+计划回收章+来源/评分/注入控制+urgency **显式 must_resolve**（MuMu 静默丢弃字段的教训）；角色三态章节水位守卫+方案 C 职业引用+派生 member_count+亲密/delta_hint 钳制；9 维分析+三维分档评分+建议数硬联动；ChapterPlan/RewriteVersion/Annotation；模板解析-覆盖-渲染-校验与上下文组装接口+Lint 接口+ChapterNumOf；旧 JSON 零迁移）③**P0 口径纠偏**=在制品曾把「已回收」**写入口径**定为 `resolved`（`revealed` 降读取别名，与 handoff §2-3/§4-7 相反，会让统计回收率/lint 计数/章节注入/SaveForeshadows 白名单等 7 类消费方静默读空）→ 纠回 `revealed` 写入口径 + `IsResolvedStatus` 覆盖两套 wire 值并接进 stats/lint/章节注入/analysis + SaveForeshadows 白名单 3 态→并集 6 态（别名先归一化）④`docs/distill/` 六域规格入库并登记 docs/README。**测试**=Go 全量绿（types 兼容矩阵+两套 wire 值新用例/app 伏笔 Lint/stats/analysis）+vitest 体检 3 例+spaceBindings 锁 467→468。**门禁**=ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646、版本三处 4.278.0；产物=exe 49,379,328B SHA256=E6E17B1B…C9FE1（releases/gaea-v4.278.0.exe + SHA256SUMS-v4.278.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。**欠账**=**六域实施 t2~t7 未开工**（拆书导入/长程一致性/情节分析重写/角色职业接线/提示词工坊/前端统一接线；规格在库、契约就位）；t1 契约除伏笔体检外暂无消费方（契约先行）；观察项=ChapterNumOf 双实现/stats·narrative 无 owner/`docs/mumu-distill/` 重复副本待删。
- **最新发布：v4.274.0（2026-09-13）「UX 线第六刀：自定义强调色亮态自动深化（对比度保障）」**——对比度普查观察池第一项挖到底：**非主题令牌缺陷**（6 主题 lightFn glow 全是深化值），是**用户自定义强调色（gaea-accent）不分明暗覆盖 glow/colorPrimary**——暗色调亮的 #1dd7bf 切亮态压浅底对比仅 ~1.5，全站 accent 文字（空间 chip/active tab/徽标/链接）看不清。**修复**=①`ensureLightContrast` 纯函数（lib/accent.ts：对白底 WCAG<4.5 时保色相饱和度迭代压 HSL 亮度，下限 0.12、灰兜底；达标/非法原样返回；vitest 7 例含真实案例）②App.tsx effTokens 亮态深化暗态原样（用户存储色不变）。**真机复验**=复跑 13 页×两态：light 告警 63→53 accent 类清零（亮态 glow 实测 #128475=深化输出），dark 15 不变；剩余为禁用态/图片背景误报/占位符/设计弱化。**观察池**=weixin「运行中」antd success 绿字 2.21（语义色惯例不动）/schedule 行号与 placeholder（设计弱化）。**门禁**=ci.ps1 全绿/版本三处 4.274.0/drift OK@642；产物见 SHA256SUMS-v4.274.0.txt。
- **最新发布：v4.275.0（2026-09-13）「价格带数据源调研结案：信息价『除税价』列适配（拍板池候选4 收官）」**——调研（docs/gaea-priceband-datasource-research-2026-09.md）：价格带数据面**基础设施已完整在产**（CostEntry Source/Region/PriceType/PriceDate 四要素+ComputePriceBand+文件/AI/视觉导入链+OCR 询价飞轮），「手动导入信息价」无需开发；外部自动接入四路评估=B 抓取不建议（脆弱+合规灰）/C 商业 API 不做（询比价已拍板不做其上游）/D LLM 查价不作基线——**建议候选4 结案**。**真机验证抓实锤当场修**：信息价样本 CSV 走 GaeaCostImportPreview（预览零落库）——「除税价（元）」不在 fieldPrice 字典→价格列 unmapped、12 行全 skip；当场修字典增「除税价/含税价」+Go 回归测试（TestParseCSV_InfoPriceChushuiColumn）；复测 **12/12 全识别**。**门禁**=ci.ps1 全绿/版本三处 4.275.0/drift OK@642；产物见 SHA256SUMS-v4.275.0.txt。
- **AGENTS.md 八迁分流（2026-09-13，非版本刀，纯文档）**——CI 卫生守卫 WARN（59926B 逼近预算）触发：v4.261.0~v4.255.0 六条入 archive（八迁记录更新），主文件恢复 14 版，水位 59926→47518B；迁移完整性三项断言全过。坑=CRLF 行尾下 node indexOf 锚点不带行尾+写入前验 includes。
- **对比度普查收账（2026-09-13，非版本刀，零代码）**：13 页×明暗两态 WCAG 程序化扫描（.tmp/walk-v4274-contrast.mjs）——dark 15/light 63 告警逐类甄别后**大头为脚本误报**（渐变/图片背景无法合成，目检实际清晰）；真实低对比仅亮态 accent 弱化文本（~10 处 1.5~1.8）+schedule 行号 1.3+weixin purple tag 3.39——全为装饰性文本非正文，**无 AA 硬伤不动**。观察池新增=亮态 accent 弱化文本打磨候选（修则需动 lightFn 令牌，视觉拍板项）。**坑=对比度自动扫描对 background-image/渐变必然误报，告警须逐类目检甄别后才能定刀**。
- **v4.270 留池补验收官（2026-09-13，非版本刀，零代码）**：sin_illustrate live 端到端**全通**——模型真调工具/图片真实落盘（sin/art「雨夜回眸」.png 1.8MB）/轨迹 artifacts 在位/`extra.illustrations['tool0']` 回写画廊可见/过程卡「思考过程·319 字·生成插图」元数据在位。**历史两次失败归因翻案**=上游 grok-4.6 一次性空返回（仅 reasoning 无正文无工具），后端兜底如实报错（sin_handler.go 空正文不落库），重试即成——**非 gaea 缺陷**。清场=故事删+产物图删（壳句柄锁删挂起，杀壳后 PowerShell 删成）。**坑**=①node rmSync 对被占用文件不抛错但删不动（Windows delete-pending），删后必须 existsSync 复核 ②走查脚本清场要放 finally（v4.270 脚本超时 throw 路径跳过清场）。**观察池新增**=sin/art 存 3 张疑似历史走查孤儿图（00:36/05:22/09:34「雨夜站台」走查 prompt 产物），待人工确认删除。
- **最新发布：v4.273.0（2026-09-13）「UX 线第五刀：键盘焦点环全站恢复（可访问性修复）」**——「继续」续 UX 线。普查 reduced-motion=基建已齐（index.css:457 系统级 + :772 应用内 ui-reduced-motion 双全局兜底）不动。CDP `Input.dispatchKeyEvent` Tab 遍历六页逐站采样抓到大问题：**v4.255 全局 :focus-visible 环全站性失效**（home 7 站/cost 5 站/sin 6 站无环）——元素 `matches(':focus-visible')===true` 但 computed `outline:3px none`，枚举 styleSheets 实锤打包产物 tailwind-*.css 有条**源码不存在的** `:focus-visible{outline:none}`（Tailwind v4.3.3 编译链路注入，源码 grep 不到），同特异性按文档序吃掉普通规则。**修复（index.css 单文件）**=①全局环 `!important` 必胜（forced-colors 同款标准做法）②antd 输入类豁免段同步 !important（自管 border/shadow 焦点态防双环）③**ant-btn 移出豁免**（Button text/default 聚焦零视觉变化，豁免=键盘用户找不到焦点）。**真机复验**=六页 Tab 遍历四页零无环，余 2 站为有意豁免 ant-input（probe 60ms 采样早于 antd shadow 0.2s transition，误报）；antd 按钮聚焦实拍 2px glow 环贴合圆角。**观察池**=动效时长令牌化不做（裸 ms/s ~95 处 vs --dur 消费 65 处，节奏差异属设计自由度）。**门禁**=ci.ps1 全绿/版本三处 4.273.0/drift OK@642；产物见 SHA256SUMS-v4.273.0.txt。

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
