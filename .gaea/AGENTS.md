# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 速览只留最近 3 版（2026-09-15 整段分流：水位根治，此前 14 版口径废止——10 条迁 archive 段首）；更早版本全部见 `docs/archive/agents-version-history-2026-09.md`（v4.49 及更早、v4.50–v4.145 仅 CHANGELOG/releases 有记录）。
- **最新发布：v4.426.0（2026-09-29）「原罪板块优化批3：切故事边界收口 + 插图落库闸门 + 工具产物链补全」**——用户指令「继续优化原罪板块」（拍板审计优化批，对齐 v4.421/v4.423/v4.425 方法论）。定刀依据=四线并行只读审计（Go 流/存储 13 项 · Go 工具/导出/书源 11 项 · 前端状态层 13 项 · 前端渲染交互 15 项，约 50 项）+ 主代理读码逐项复核。前端 14 文件（含 1 新增模块 sinBookDownload.ts）+ Go 13 文件 + 6 个新测试锚，绑定面 726 零变更，零功能删除。**论点（三个系统性缺口）**=①「切故事」边界只有 selectStory 有收尾：createStory 直接改 activeId（发送中新建=旧流后台烧完+新故事 Composer 被 sending 锁死+cancel 按 activeIdRef 取消错故事+绕过 confirmSwitchThen 脏草稿确认）；saveCast 保存闭包绑定发起故事，在途切换把 A 的生效清单写进 B 的面板状态（点「带入故事」即覆盖 B，后端无 baseline=不可恢复）；useSinNotes.save 回填绕过读路径 live 守卫。②插图「不落库出图」族：流式中间态手动按钮可点、停止/超时/报错残行（messageId 恒 0）照样自动出图（只生成不回写，重开退占位+重烧 GPU）；超时路径整段覆盖已流出正文且不调 SinCancel；取消迟到 rejection 把主动取消呈现成红色失败。③工具产物链半截：sin_illustrate 实例整回合复用而 arts 只增不减（「再画一张」轨迹重复携带第一张、toolN 编 cue 同图占两键）；sinAttachIllustration 不校验 messageID 归属话题（本板块唯一 fail-open 写口）；MD/EPUB 导出只认正文标记数字 cue=工具图整批消失且 caption 落库即丢；首轮任意错误被误判「模型不支持工具」（假 notice+静默关工具重试）。**落地**=前端 teardownStream 抽取+createStory 前置收尾、cancel 按 streamTopicRef 优先、超时先 SinCancel 保留正文提示走 notice、loadMessages 读失败不清空、reloadMessages 流中跳过、saveCast/saveNotes 归属校验+失败如实返回、SinIllustration 出图 messageId 闸+cancelledRef 收敛取消竞态、新建按钮接 confirmSwitchThen、角色保存成功才关选择器、重命名空输入保留原标题、导出按钮 disabled 透传、冲突覆盖切故事后给回声、书源下载会话模块级单例 sinBookDownload.ts（卡切换不丢进度/toast/jobId+起跑防双击+假取消刷清单=v4.423 观察池立案清账）、过程卡产物加载占位+流式 result 帧 artifacts 全链贯通（流中缩略图此前被丢弃）、Composer 含空格路径附件改 @"..." 引号注入（不含空格路径既有口径零变化）、光标无效 aria-label 改 aria-hidden；Go=Execute 入口重置 arts、sinAttachIllustration 加 topicID 归属校验 fail-closed、首轮降级加 sinErrSuggestsNoTools 判据、失败零正文落库用户指令（镜像零正文取消口径）、收尾轮工具调用可见化+接回上下文（正文重复根因消除）、MD/EPUB 导出补工具产物附录（caption 替代文本）、识图前「正在解析 N 个图片附件」notice、轨迹 Args 2000 rune 闸、EPUB 跳零正文行+连续指令逐条入引块+退化标题回退默认名、原子写统一（四处固定 .tmp 换 fileutil.AtomicWrite=双开壳互踩根除+书源 EPUB 唯一临时文件+rename）、sin_file_refs 引号 token+io.ReadFull 读满+空文件不报错+sinTrimPartialUTF8 共享、成书节选同口径、Windows 保留设备名避让（sinExportFileName+sanitizeDirName 小说导入共用）、参考图 data URL 8MB 上限、下载进度语义改已处理/总数（有失败章不再卡 N-1 瞬跳）。**测试**=Go 新增 10 锚+改写 5 锚；前端新增 5 锚+重写 7 锚；sin 域 111/111 绿、全量 vitest 409 文件 3541 例绿、tsc 0、eslint 0 错 0 警、internal/app 整包+booksource -count=1 绿。**门禁**=全量 ci.ps1 → CI OK；漂移闸 OK@4.426.0（726 一致）。**产物**=`releases/gaea-v4.426.0.exe` 51,567,616 B SHA256=`24175989253fb2b04ab51cf2b97c1ed41208cd4b5c19d4908722f5d72b8e800f`（SUMS 在案，冒烟 200 过）；保留策略删 v4.421.0.exe，实存 5 版。**文档**=releases/v4.426.0.md+CHANGELOG/README+releases/README+AGENTS 速览（迁 1 插 1：一百三十九迁 v4.416.0 入 archive）+progress。**观察池**=跨实例族（sinRuns 进程内互斥/话题 id 撞名/messageID 启发定位）/Delete·Clear TOCTOU 残余窗口/画廊 base64 常驻/sin_insight 三口全量折叠/识图有界并发/illustration merge 锁+stat-then-write（既有）/ChatTabs tab 语义（办公共用另批）/sin 手风琴 aria-controls/设定卡与插图队列统一/illustrations 值结构升级/novelstyle rewrite 链（需拍板）/真机观察解释腔与工具误用率。
- **最新发布：v4.425.0（2026-09-28）「小说板块优化批2：未保存保护收口 + 跨书失效治理 + 竞态守卫」**——用户指令「继续优化小说板块」（拍板审计优化批，对齐 v4.421/v4.423 方法论）。定刀依据=四线并行只读审计（前端创作/阅读 10 项 · 前端设定/书架/导入导出 13 项 · Go 后端 11 项 · 计划台账 18 条抽验）+ 主代理读码复核（规格 `进度计划/gaea-novel-audit-20260929.md`）。前端 45 文件 + Go 9 文件 + 5 新测试文件，绑定面 726 零变更，零功能删除。**论点（三个系统性缺口）**=①v4.421 未保存保护只覆盖页内动作，不覆盖「服务端写盘成功→前端重载」反向路径：局部重写与重写历史应用/恢复原文无条件 `loadChapter` 把磁盘正文灌回并抹掉脏标志（同文件 `refreshEditorAfterServerRewrite` 反而有护栏＝漏网点）；②窗口/面板跨书状态不失效（v4.421 只修两面板）：`WorldviewSectionsEditor` 会把 A 书六维整表写进 B 书（后端无备份）、设定页 markdown 静默丢、章节分析主 load/伏笔体检/全书体检/评审/指纹同类；③异步响应缺上下文守卫：生成中切项目旧书 `done` 把正文挂新书同号章并标已保存、载入在途点保存跨章写、摘要/问书迟到错章。Go 同型两处：生成链整份过期 outline 快照回写（流式期间大纲编辑被静默回滚）、书源补下章号基准只取顶层且不查磁盘（续写大纲压成 1..5 卷后从 006 起覆盖既有正稿）。**落地**=线1 新 `lib/boardActive.ts` 板块级可见信号（MainLayout 声明 + NovelPage pane active 取 AND，修隐藏阅读页吞 Ctrl+S/F11）+ 新 `novelSwitchGuard.ts` 跨页切书闸门（登记+单一拦截点，HomePage 四处 openProject 全过闸）+ 删当前书即 closeProject + 搜书兜底对账刷书架 + TTSPlayer/CharacterLibraryPage 整通道裸 EventsOff 改 subscribeWailsEvent（+GhostText 收敛）与源码守卫 `events-discipline.test.ts`；线2 创作页 A1/A2a 脏则不重载+入口三选闸、A3 requestedPath 守卫+切书取消在途生成、A4 saveActive 在途早退+按钮 disabled、A6 rail disabled、A7 生成中向导可见提示、A8 新角色弹窗按 active 挂起补弹、B5a 切书清评审/指纹态；线3 阅读页 A2b 按面板章号判脏、A9 摘要/问书 seq+章号双守卫、A10 载入在途只读 loading、主代理补「恢复上次阅读章」迟到回填不覆盖已开标签；线4 B1 维度化编辑器依赖 projectPath（P0）、B2 设定页脏登记+切书提示、B3 分析主 load seq+projectPath、B4/B5b 切书清态、B6 导出全失败不再弹成功、B7/B8 关闭前问脏、B9 onShelfReconciled、B10 体检失败≠空态、B11 详情可重试、B12 回填校验章号、M4 伏笔在途禁用；线5 Go G1 回写改读-改-写按 nodeID 定点合并、N8 落盘 fail-closed、N7 协程入 WG、M5 残稿名纳秒+序号、G2 补下基准 max(递归大纲,磁盘)+挑空号、G6 AtomicWrite、G4 场景圣经预算配账（脊椎段前置——原末尾截断恰砍掉 v4.422 故事主线）、G5 上一章摘要按章号、G7 整篇 span 不豁免、G8 去味 before 口径统一、G9 证据区间合法、G3 伏笔写盘失败如实上报；台账更正=补登 v4.421 线5（原漏登）+两条更正（裸 EventsOff 当时未清零 / 在途禁用当时只有队列）+docs 生成物路径 `frontend/src/wailsjs/`→`frontend/wailsjs/`。**测试**=小说域定向 53 文件 465 例全绿（基线 49/401，净增 4 文件 64 例）；tsc 0；45 前端文件 eslint 0 错 0 警；Go `internal/app` 整包 142.5s 绿+六包绿+gofmt/vet/build 0；反向守卫逐条变异验证（板块门控/跨页闸门六态/重写不覆盖/生成跨书/载入在途跨章/切书清态/恢复章不覆盖；Go 补下基准、白名单整篇、证据区间）。**门禁**=全量本地快闸 `ci.ps1` → **CI OK**（含全量 vitest 409 文件 3536 例+E 系+卫生守卫）；漂移闸 OK@4.425.0。**产物**=`releases/gaea-v4.425.0.exe` SHA256=`1e58ea6a6a615b8eb61b2bf61be4f37a17a5535b3c2d54e8e8081bbad0750e35`（SUMS 在案，冒烟 200 过+桌面副本）；保留策略删 v4.420.0.exe，实存 5 版。**文档**=规格 `进度计划/gaea-novel-audit-20260929.md`+`releases/v4.425.0.md`+CHANGELOG/README+releases/README+AGENTS 速览（迁 1 插 1：一百三十八迁 v4.415.0 入 archive）+progress。**观察池**=前端 9 项（局部重写坐标错位需改后端契约/设定 Agent 回填覆盖在途键入/情感曲线无取消/伏笔写回交错窗口/重写历史跨书面/Cmd+K 接受需真机/停止生成连接建立期/切项目后端是否续写/跨客户端并发）+Go 10 项（N5 分支摘要混入/N9 缺口即停同族 3 处/N12 迁移标记/N13 嵌套节点/G10 SceneRefs/G11 无消费点模板/plans.json 无锁读改写/scene_gen 同款吞错/chapter-review fallback/零调用者绑定口径）。
- **最新发布：v4.424.0（2026-09-28）「原罪提示词批：工具教学面与注册表同步 + 反 AI 腔对齐蒸馏判据」**——用户指令「继续优化原罪，你应该看看原罪的提示词」。Go 2 文件（sin_prompt.go+测试），绑定面 726 零变更。**三修正**=①【工具】节补齐 5 个未教工具（注册表 9 个只教 4 个；sin_illustrate 与标记协议二义：点名要图=工具、正文配图=标记、同图不双路径；book_search/download 素材化不照搬；sin_export 明确才调+本回合正文未落库拿不到）②【写作规范】补解释腔五标记（她不知道的是/殊不知/多年以后/之所以…是因为/这意味着）与否定翻转（不是…而是…偶尔一用不连用）两条——对齐 novelstyle patterns.json 蒸馏判据（T2 门禁 B/G 同源），提示词劝阻+内核门禁同判据③【插图协议】重写（一轮至多 2 张+去重复表述）。**防回归闸 ×2**=TestSinSystemPromptCoversToolRegistry（注册表工具名必须全在提示词）+AntiAIFlavorRules。【测试】sin 全族+2 闸绿；全量 ci 绿；漂移闸 OK@4.424.0。【观察池】sin 正文接 novelstyle rewrite 链（需拍板）/真机观察解释腔出现率。
- **最新发布：v4.421.0（2026-09-28）「小说板块优化批：未保存保护 + 常驻页副作用治理 + 取消防覆盖」**——用户指令「优化小说板块」（拍板 A+B 合并批＝正确性修复 + 体验收口）。定刀依据=三线并行只读审计（小说前端 15 项 / Go 后端 14 项 / 计划台账清账与 STALE 校正）+主代理读码复核，落地 20 项。前端 15 文件（3 新增）+Go 4 文件（2 新增测试），绑定面 720 零变更，零功能删除。**论点**=小说五子页常驻挂载（NovelPage 全 pane 同挂 + CSS 隐藏，隐藏≠卸载）衍生三类真实缺陷：①窗口级副作用不随页隐藏（ChapterPage 与 NovelSettingPage 各挂无条件 Ctrl+S——全仓仅此两处：一次按键同存「设定」与阅读页那章，创作页手写正文反而不保存；F11 在任意子页翻转不可见阅读页专注模式）②面板不随书刷新（ForeshadowPanel/ConsistencyPanel 的 load 只依赖恒 undefined 的 disabled，切书仍显上一本；引导态死代码）③未保存缓冲无脏保护（创作页切章/切项目/生成/删除直接 setContent 覆盖，阅读页早有 needsCloseConfirm 先例）。**P0 根修（Go）**=取消生成覆盖已完成章节：残稿此前无条件覆盖 chapters/NNN.md，改为「开局快照 existedBefore + 取消时复查 chapterBodyExists」→ 已存在则另存 NNN[分支].partial-<ts>.md（fileutil.AtomicWrite，绝不回落覆盖正稿），事件增 partialSaved/partialPath/notice，前端如实提示落点且不标已保存。**落地**=线1 创作页（共享原语 components/novel/unsavedGuard.tsx 两选/三选/并列多分支+切章·生成·删除·重写全覆盖+载入失败可见化+并列分支不再绑 Esc+反推卸载守卫与等待时长+设定读取失败归因+去味后刷新编辑区+创作参数持久化+工具轨 12 钮分三组）；线2 常驻页快捷键门控（NovelPage 五 pane 下发 active，非当前页不响应也不 preventDefault，Ctrl+S 单一归属；目录点章改 commit 后派发）；线3 设定页与两面板（load 依赖 projectPath+换书清 AI 深检+导入确认+下行可折叠）；线4 组件批（在线导入进度失落兜底+章际对比 seq 守卫+三处裸 EventsOff 改 subscribeWailsEvent+提示词工坊脏保护+新建小说空标题禁用与防双击+伏笔全表写回串行化）；Go 同刀（写回路径收敛 writeBackRewritten：写失败报 error+v4 场景写回后 syncBlobFromScenes；取消互斥 nil 占位由 unregister 独占清理）。**测试**=Go 新 create_chapter_partial_test/novel_writeback_test，internal/app 定向 PASS 82/FAIL 0+场景族 17 绿，gofmt/vet/build 0；前端新增原语 7 例+NovelPage.test 3 例+各线守卫 30 余例，小说域定向全绿+tsc 0+eslint 0。**门禁**=全量 ci 绿+漂移闸 OK@4.421.0。**产物**=见 SHA256SUMS-v4.421.0.txt（冒烟 200 过）。**文档**=规格 进度计划/gaea-novel-polish-20260928.md+releases/v4.421.0.md+CHANGELOG/README+releases/README+AGENTS 速览（本条；v4.420.0 计划模式 chip 此前未入速览，见 CHANGELOG）+progress。**观察池**=跨页切书确认（需跨页共享脏态）/EditorPanel 内 rune↔code-unit 换算收敛/板块级 keepAlive（切到别的板块时小说页仍常驻→Ctrl+S·F11 仍吞键，全仓各页共有，需「板块级可见」信号）/Go 余 9 项（搜索标题命中抑制正文扫描·生成后协程未托管·建章忽略 WriteOutlines 错误·ForEachChapter 缺口即停·分支摘要混入·书架全库全章 IO·迁移跳章仍落 v4 标记·嵌套大纲节点不可见·11 个零调用者导出绑定）/残稿文件名秒级时间戳。
- **非版本刀（2026-09-19）todos 对账清账 + booksource 进度回调乱序根修**——起因=「TaskCenter 会话维度」行标 ⬜ 实际 v4.229.0 已落（SchemaV20+SubmitSpaceSession+过滤面 chip 熄灯）险些重做；子代理全表对照 git log/CHANGELOG 核查：**关 3 行**（TaskCenter/审计刀A v4.165.0/knip v4.268.0 清账）+**改写 2 子句**（t4-C4 已随 t7 收官〔面板只读高亮+锚点 v4.325.0〕编辑器 overlay=有意裁剪入观察池；造价 §6=v4.209.0 已收官〔基线源=自有库〕）+删 7.3-1 旧欠账注记（v4.333.0 已收口）+补记 7.3-2 缺省翻转候拍板；**过时源曾传染 v4.338 未做段——排刀前先对账，勿按旧未做段排刀**。**根修**=booksource fetchChapters OnProgress 锁外发射乱序（全量 ci 实测 [0/3 2/3 1/3]，负载 flaky 家族+1）且对消费方并发调用——发射互斥下现读计数：序列非降/末次=成功总数/消费方免同步，契约注释补「轻量回调」纪律；-count=10 绿，本机无 gcc -race 不可跑（race 门=Actions）。**教训**=进度类回调的发射序是契约一部分——计数加锁≠发射有序；todos「已落地未关」会传导进 release notes。
- **非版本刀（2026-09-19）真机走查班：v4.337~v4.343 新面清池（用户在场背书，只读纪律）**——CDP 9333 附着打包壳（v4.343.0 exe）：**双空间 13 页巡检全绿**（书斋 6+闲庭 7，错误边界 0/空白 0/console error 0/exception 0）。**夹具实数据深验**（.tmp/makefixture-v343 造 12 章+大纲+前 3 章 analysis-v2+第 1 章两标注，走查后删净双复核）：①章际对比差值表逐字精确（7.8 vs 6.0 Δ-1.8，四维+情感强度）②**定位镜像高亮真机完美**——「夜风从窗缝里钻进来」青色 mark 精确罩住正文对应字串，真实字体/滚动条下零偏移（v4.343 最险面过关）③情感曲线 3 点+tooltip 逐字精确④体检/分析空态诚实。**坑**=①书架工程清单在 C:\AI\xiaoshuo（novelsDir 设置），夹具造在仓库 novels/ 不上墙②工程列表启动时读，壳须重启重扫③CreatePage activeChapterNum 有 lastMainChapter 兜底——分析面板章号≠树选中章，夹具驱动须先点树行（内层 div [title*=摘要] 才是点击目标，外层容器无 handler）④嵌套模板字面量里 \d 会被外层吞掉弄坏 eval 正则——CDP 探针用 [0-9] 或 DOM 直读。**清场**=杀壳→删 C:\AI\xiaoshuo\对话流走查工程→ls 双复核（用户真实工程零触碰）。**剩余挂池**=pptx 刀2/mspdi 样本类深走查（需数据）、持久全量 overlay（真机镜像对齐已过，可议）。
- **最新发布：v4.419.0（2026-09-26）「动词回执可见化：notice 摘出过程卡（真机走查实锤缺陷修复）」**——v4.418 真机走查池实锤的「pre-turn 动词 Notice 不可见」修复。前端 1 文件+1 测试，绑定面 720 零变更，Go 零改动。**缺陷**=/plan on 等动词回执藏在「过程」折叠卡内默认不可见；**根因**=Transcript alternatingSegments 分组谓词把 notice 与 tool/phase/compaction 一并收编（v4.26 phase 收编时顺带）；**根修**=谓词摘出 notice，一律独立成行渲染（warn 仍走 ErrorCard；info 走既有 .notice 弱化行零新样式），失败告警同样受益。**测试**=ProcessCard.test 增 1 例（buildSegments 断言 notice 恒落 outsideItems）13/13 绿；真机目检沙箱 /plan on 回执独立行可见（截图在案）；全量 ci 绿。**门禁**=漂移闸 OK@4.419.0。**产物**=见 SHA256SUMS-v4.419.0.txt（冒烟 200 过）；保留策略删 v4.414.1.exe。**文档**=releases/v4.419.0.md+CHANGELOG/README+releases/README（443→444+裁 v4.388）+AGENTS 迁 1 插 1（一百三十七迁：v4.414.1 入 archive）+progress。
- **最新发布：v4.418.0（2026-09-26）「走查沙箱开关：GAEA_WALKTHROUGH=1 强制隔离工作区（真机走查防复发刀）」**——2026-09-26 真机走查险情（首条消息自动装配把回合接进用户真实会话）根因封堵。Go 1 文件改+1 新测试（内部 2 文件+助手新件），绑定面 720 零变更，前端零改动。**背景**：真机走查「先建专用会话再发消息」无效——装配前 NewSession 必然诚实报错，首条消息走自动装配+自动恢复，workspace 解析到哪回合落到哪（当日实证接进用户真实会话，模型调用即败未持久化，已按 RewindLog 口径清场归零）。**落地**：GAEA_WALKTHROUGH=1 → ApplyWalkthroughOverride 强制 workspace=%TEMP%\\gaea-walkthrough\\workspace（自动创建、删目录即清场），覆盖用户/项目配置（走查隔离优先于一切）；未设零变化；激活 slog.Warn 可审计。**双入口必须同调**：壳内办公引擎不走 config.Load()，走 app 层 gaeaLoadConfig()（自带 Default+toml 合并）——当日首版漏壳侧即失效，真机验证抓出后补齐。**真机验证实录**：沙箱启动→GaeaInit→ListDir 探针=沙箱（14 真实会话零出现）；沙箱内 /plan on 实弹确认「pre-turn Notice 不可见」=真实缺陷（藏在过程折叠卡，候可见 chip 小刀）；/plan off 正常。**诊断坑三连**：①双入口漏一即失效②装配前 ga.cfg=nil 探针全假（gaeaCwd 走 Getwd 兜底），必须 GaeaInit 后再探③CDP 探针可打在僵尸旧实例（gaea-v4.418.0.exe≠gaea.exe 按名漏杀），跨实例诊断先对 PID。**测试**=config+app 包回归绿；全量 ci 绿（OriginalSinPage.tabs 负载 flaky 一轮隔离复跑绿，v4.414.0 同款）。**门禁**=漂移闸 OK@4.418.0。**产物**=见 SHA256SUMS-v4.418.0.txt（冒烟 200 过）；保留策略删 v4.414.0.exe。**文档**=releases/v4.418.0.md+CHANGELOG/README+releases/README（442→443+裁 v4.387）+AGENTS 走查配方增补沙箱用法+迁 1 插 1（一百三十六迁：v4.414.0 入 archive）+progress。
- **最新发布：v4.417.0（2026-09-26）「前端 UI 完善批三：硬编码色值收口 + 可访问性补尾」**——接 v4.415/v4.416 观察池。前端 6 文件，绑定面 720 零变更，Go 零改动。**色值收口**：ContextView 趋势图例裸 hex（亮态绿白底 2:1）→ tailwind v4 变量语法 text-(color:--color-success/--color-destructive)；WhisperTracePanel MiniBar aff/sec/aro/dom 四色→primary/success/warning/--whisper-accent；WhisperEmotionPanel 六维条 #60a5fa→--whisper-accent。**证伪保留**：RelationGraph #6b7280×5/GraphView #64748b/TisorRadar 渐变 stops=SVG/3D canvas 字面量域（startsWith 校验+混色逻辑会破 var()），加注入池。**可访问性补尾**：ChapterEditor 右键菜单三个可点 div→role=menu+menuitem button+开启即聚焦+Esc 关闭+↑↓ 漫游（视觉零变化）；MemoryHub 总览计数加载中「…」占位（数字/—/… 三态齐整）；CharacterLibEditor 大 Modal 补 aria-label。**圆角令牌化专项证伪降级**：审计前提不成立——--md-sys-radius-* 在 appStore 主题表 dS/lS 明暗恒 8/12/16/28px，500 处迁移今日零用户可见收益，仅当圆角需随密度/主题缩放才立项（间距栅格收敛同理维持观察）。**测试**=定向 4 套件 80 例绿；全量 ci 绿。**门禁**=漂移闸 OK@4.417.0。**产物**=见 SHA256SUMS-v4.417.0.txt（仅本地；冒烟 200 过）；保留策略删 v4.413.0.exe。**文档**=releases/v4.417.0.md+CHANGELOG/README+releases/README（441→442+裁 v4.386）+AGENTS 迁 1 插 1（一百三十五迁：v4.413.0 入 archive）+progress。

- **AGENTS.md 八迁分流（2026-09-13，非版本刀，纯文档）**——CI 卫生守卫 WARN（59926B 逼近预算）触发：v4.261.0~v4.255.0 六条入 archive（八迁记录更新），主文件恢复 14 版，水位 59926→47518B；迁移完整性三项断言全过。坑=CRLF 行尾下 node indexOf 锚点不带行尾+写入前验 includes。
- **对比度普查收账（2026-09-13，非版本刀，零代码）**：13 页×明暗两态 WCAG 程序化扫描（.tmp/walk-v4274-contrast.mjs）——dark 15/light 63 告警逐类甄别后**大头为脚本误报**（渐变/图片背景无法合成，目检实际清晰）；真实低对比仅亮态 accent 弱化文本（~10 处 1.5~1.8）+schedule 行号 1.3+weixin purple tag 3.39——全为装饰性文本非正文，**无 AA 硬伤不动**。观察池新增=亮态 accent 弱化文本打磨候选（修则需动 lightFn 令牌，视觉拍板项）。**坑=对比度自动扫描对 background-image/渐变必然误报，告警须逐类目检甄别后才能定刀**。
- **v4.270 留池补验收官（2026-09-13，非版本刀，零代码）**：sin_illustrate live 端到端**全通**——模型真调工具/图片真实落盘（sin/art「雨夜回眸」.png 1.8MB）/轨迹 artifacts 在位/`extra.illustrations['tool0']` 回写画廊可见/过程卡「思考过程·319 字·生成插图」元数据在位。**历史两次失败归因翻案**=上游 grok-4.6 一次性空返回（仅 reasoning 无正文无工具），后端兜底如实报错（sin_handler.go 空正文不落库），重试即成——**非 gaea 缺陷**。清场=故事删+产物图删（壳句柄锁删挂起，杀壳后 PowerShell 删成）。**坑**=①node rmSync 对被占用文件不抛错但删不动（Windows delete-pending），删后必须 existsSync 复核 ②走查脚本清场要放 finally（v4.270 脚本超时 throw 路径跳过清场）。**观察池新增**=sin/art 存 3 张疑似历史走查孤儿图（00:36/05:22/09:34「雨夜站台」走查 prompt 产物），待人工确认删除。

- **品牌资产补齐（2026-09-10，非版本刀）**——2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核），**两个二进制件没跟上**：`build/appicon.png`（1024²）与 `build/windows/icon.ico` 停在 **2026-08-05** 旧视觉（翡翠球体+破土嫩芽+星芒，深蓝底 `#0F172A`）→ **exe 内嵌图标/任务栏/资源管理器/桌面快捷方式全显示旧 logo，只有窗口内 UI 是新 logo（「换了一半」）**。**落地**=① 无头 Edge 渲染 `appicon.svg` →1024×1024 RGBA PNG（**必须传 `--default-background-color=00000000`**，否则圆角外合成不透明白、alpha 静默丢失）；② Pillow 由同一母图出 **7 档 ICO**（16/24/32/48/64/128/256，旧资产缺 24，本次补 Windows 标准全集）；③ `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**）。**验证三条独立证据**=① 几何精度（核心圆 r=50@512 实测直径 **200px**、圆心 (539.5,511.5) 对理论 (540,512)；环 stroke 48 实测线宽 **96px**；圆角 rx=104 对角线首不透明像素 d=**61** 对理论 60.9——零缩放零偏移）；② exe 内嵌图标提取（`ExtractAssociatedIcon`：底板 `#0B1210`/米色核/翡翠环=新「地核 G」）；③ dist bundle 新 logo 独有标记 `gaea-logo-ring`×3、`gaea-logo-light-ring`×3、`0B1210`×2 在册且旧独占色 `#6ee7b7`/`#047857` **零命中**+按旧 SHA256 全仓比对零旧 logo 残留。**门禁**=build exit 0 / 冒烟 200 / **纯资产刀（零前端零 Go 源码改动、绑定面不变）**。**产物**=`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希；中间态 903F9759…C36EC068=只换 PNG+主图 ICO 的一版，小尺寸优化后重建覆盖；原始 59C2B518… 系 v4.208.0 归档值，releases 档案自洽未动）。**坑**=① 无头 Edge 截图默认白底，CSS 里写 `background:transparent` 无效（截图不继承页面背景），透明基线拿旧资产角像素 alpha=0 对照；② **判新旧前须验该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变与**旧** logo 主色，拿它判会把新资产误报成旧的，判据只能用**独有**标记（SVG id 名 / 独占色）；③ `pwsh` 不在 PATH，手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（**非 app 故障，易误判为冒烟失败**），用 `powershell`（build.bat 内已有 fallback）；④ 运行中的 exe 可被 `Copy-Item` 直接覆盖（进程持旧映射），桌面副本无需先关 app，但**已加载实例不会换图标，需重启**。**小尺寸可辨识（同日追加）**=原方案对大图统一降采样，致 16×16 环宽仅 1.5px / 核 3.1px、G 字形糊成深色块；新增派生资产 `build/appicon-small.svg`（环 48→64、核 r50→62、去 halo），ICO **按尺寸分流源图：16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**（若在 24→32 切会跳）；**坑=Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 逐帧 16B + PNG 帧，**256 档宽/高字段写 0**，写 256 溢出单字节）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。**未抬版本**——按 README 发版约定「文档整理、令牌对照、单点样式等小改记入 CHANGELOG，不单独抬版本」，本刀属视觉资产补齐，与 `progress.md` 的「工作空间与文档整理（非版本刀）」同类处置。

- **壳内全板块渲染健康巡检（2026-09-12，非版本刀，随 v4.237 线收尾）**——CDP 起壳逐 rail 页点击+错误边界断言：闲庭七页（闲庭首页/首页/聊天/小说/绘梦/模型中心/角色库）+书斋六页（首页/办公/造价数据库/记忆中枢/模型中心/青鸟）**13 页全绿零接管**；配方=walk-pages.mjs（rail 坐标 DOM 定位+逐页停留+错误文本断言+失败自动截图）；巡检结论=无其他「静默坏死」页面。零代码改动不抬版本。

- **DAG 6.3 壳内真机走查（2026-09-12，非版本刀，60c68a34）**——§6 最后一条余项清账（**§6 余项全清，DAG 6.3 线收官**）：CDP 9333 DOM 断言走查（不点危险操作不打模型）——注入体检 Drawer 真跑=GaeaMemoryEvalRun 真实 wire+真实库**端到端打通**（通过：五条结构不变量全部成立；晨报预载 2 条·104/600、本体 2 引用·181/600、门控四开关、toast；截图 .tmp/eval-drawer.png）；办公流水线区 dag-section 在位（空态引导正确）；全程 exceptionThrown=0/页面渲染出错=0。**观察池新增**=办公记忆库面板「不可用—未配置」与同库体检读出 2 条活跃并存——面板 available 旗与体检取数路径（hubOfficeStore 直连）口径不同，既有语义非本线回归，按需另刀。配方=.tmp/walk-v4243.mjs/.tmp/walk-dag.mjs（rail 悬停展开→按 title 前缀点库入口→按钮文本带计数须前缀匹配；**节点列表/状态徽标在展开层，先点 goal 展开**）。零代码改动不抬版本。

- **v4.174–v4.178（2026-09-09）瘦身 P3 版3/P4 三刀与造价条目匹配键刀**——全文迁入 docs/archive/agents-version-history-2026-09.md（2026-09-11 三迁腾预算，check-docs 指令预算守卫）。
- **非版本刀（2026-09-23）源码包归档推送**——wubigork-v4.399.0-source.tar.gz（13028977B/4744 files，SUMS 双行）+ 29 个未推 commit 与 v4.373~v4.399 tag 全量上 remote；画室编辑力八连发里程碑打包。坑：git archive 的 -o 是全局选项须放 pathspec 前（--output= 形式最稳）。

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
   **「CI 绿」口径（2026-09-27 定）**：写门禁结论必须带定语——「本地快闸」=
   `scripts/ci.ps1`（构建/vet/golangci v2.14.0/Go 全量/漂移/前端/E系/卫生）；
   「Actions」=.github/workflows/ci.yml 三 job（backend+race+frontend，race 检测
   只在 Actions：本机无 gcc）。禁无定语「CI 绿」。
3. **定向优先，全量收尾**：改动后先跑目标文件（3–10s）确认；全量只在交付前跑一次。
   一轮改动 = 中间定向 + 收尾全量，**同一验证不重复跑**（上一轮全量跑了 4 遍=反例）。
   **工具（2026-09-27 立）**：`scripts/test-related.ps1` = git 改动 → Go 包
   （吃结果缓存，不带 -count=1）+ `vitest related`，几秒~分钟级；`ci.ps1 -Quick`
   = 静态闸（build/vet/lint/卫生守卫）+ 受影响面测试，实测 ~1 分钟（vs 全量
   5-10 分钟）。**全量档（ci.ps1 不带参）保留在 push 前/发版前跑**；
   用户对「几分钟」的抱怨=2026-09-27 拍板用两档制化解，勿再无脑全量。
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
- **真机走查沙箱开关（v4.418，必用）**：`GAEA_WALKTHROUGH=1` 启动壳=workspace 强制指向
  `%TEMP%gaea-walkthroughworkspace`（覆盖用户/项目配置），会话/记忆/交付物全落沙箱，
  走查后删沙箱目录即清场——首条消息自动装配+自动恢复会把回合接进真实会话，此开关是
  唯一根治（2026-09-26 险情防复发）。未设时零行为变化。
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
- **审计衍生优化方向（2026-09-26）= `docs/gaea-optimization-direction-2026-09.md`（候选池；**P0-1 门禁落地 + P0-2 comfyUI 竞态已于 2026-09-27 落地**，见该档 §3 状态注记）**：
  本次全仓审计（三路并行模块审计 + 本机实测复现）——综合 7.0/10 + 分维与 14 模块评分 + P0~P2
  优化项（门禁落地 / internal-app 竞态 / 台账自动化 / 上帝包拆分 / 281 遮蔽绑定 / 平行实现收敛 /
  效果验证闭环 / 甘特库评估 / 仓库体积）+ 本次整理清单 + 未验证项观察池。**排刀前先读它的 §3 与 §5**；
  与既有权威路线冲突时，仍以本段规划为准。
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
