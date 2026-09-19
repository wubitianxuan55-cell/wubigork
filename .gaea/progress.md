## 最新发布：v4.350.0（2026-09-19）「前端优化第二轮：事件总线连环炸根修 / P0 幽灵令牌 / 破坏操作确认 / 静默失败 / 流式渲染热点 / 主题破绽」

- **动机**：用户口径「优化 gaea 前端」（v4.349 五线后第二轮）。同范式：四路并行**只读审计**（事件与定时器泄漏 / 交互反馈三态 / 渲染热点 / 视觉一致性，各带 file:line 与量级推理）再按证据定刀；纯前端 32 源码 + 6 测试（3 新 3 适配），零 Go 逻辑改动、零绑定变更（706 不动）。
- **线1 事件总线连环炸根修**：❗`onTaskEvent` 清理走 `EventsOff(channel)` 全清（wails 语义=注销通道全部监听者；v4.61 事故红线 v4.62.2 只修了 onEvent/onSubagentText，本条漏网）——gaea-task 有 5 个并发订阅点（运行角标/任务自动激活/任务面板/价格源/索引任务），任一卸载（关一次任务面板即触发）=角标冻结+自动打开失灵+下载卡「进行中」，keepAlive 下直至应用重启→一律 `subscribeWailsEvent` 按监听者退订；onUpdaterProgress/onReady 同纪律收口。polyfill（网页模式）双修：EventsOn 返回 void→消费方 deps 每变往 eventBus **叠加** handler（ModelCenter 四通道单事件 N 次重复后端拉取）→改返回「只摘自己」退订对齐 wails v2.13 桌面语义；EventsOff(带 callback) 无条件 sse.close()→共享通道他人推送被掐断→只摘自己、通道空才关 SSE。
- **线2 P0 幽灵令牌**（12 主题功能性坏点）：`--md-sys-color-error` 全仓零定义而 gaea 四组件 26 处无 fallback 消费（CSS invalid：失败态红点/红字/红框全消失）→注入 colorDestructive 语义别名；`--md-sys-color-surface-container-low` 零定义→追问发送按钮箭头 12 主题恒隐形+输入区透明+VersionTimeline 对比底丢失→color-mix 插值；`--color-info`/`--md-sys-color-info` 补注入（#38bdf8/#0369a1 明暗分档）；幽灵四连 `--v3-fg-soft`(17 处)/`--v3-line-soft`/`--color-text-tertiary`/`--md-sys-color-text-tertiary` 明暗分档定义（暗主题 12px 次要文字 3.9:1<AA）；modelcenter.css 13 处 gaea 作用域 token 换主令牌直连（--fg 系只在 gaea/styles.css 定义且该页不加载→亮主题 #ddd 白底 1.3~2.5:1）。
- **线3 破坏操作二次确认 9 处**：高危批量/清空 4=知识库批量删除（双分支）/成本库批量删除（**连带假成功根修**：吞错后无条件「已删除 N 条」→诚实成功 N 失败 M）/绘梦清空历史（prompt/seed 即时持久化）/原罪清空故事消息（ToolbarButton 无 forwardRef→Modal.confirm 命令式）；单条 5=任务收件箱删除（**连带吞错可见化**+i18n 三语新键）/青鸟提醒/组织/角色关系/划线想法列表（span 挡冒泡防触发跳转）。
- **线4 静默失败反馈**：三脑检索失败与 0 命中三态化（命中/换关键词/失败重试，role=status）；排程导出 XML 补成败提示（SaveFileAs 抛错原落 unhandled rejection）；章节载入失败白板→message.error 带原因（原只 console.error 且 saved:true 误导正文丢失）。
- **线5 渲染性能**：Transcript 提及扫描按文本内容缓存（原每 chunk 对全会话 assistant 正文跑双全局正则+O(m²) 重叠；完成后文本不可变→缓存命中只付流式段成本，上限 512）；ChatPage topicList memo+ChatTopicSidebar React.memo+过滤 useMemo+行 hover 删 hoveredId 改纯 CSS；ToolCard summary/prettyArgs/outputLines 三处 memo（write_file args 内联整文件数百 KB）；`useNow(active)` 条件订阅（已完成消息/过程卡/RunStatus 无条件订阅 1s 时钟=O(N)/秒常驻重渲染清零；**伴生修复**「思考 Xs」完成后持续增长→完成边沿 Date.now() 定格）。
- **线6 视觉**：RelationGraph 画布 #ddd/#555/#333/#444 写死（6 亮主题节点名不可见）→resolveCSSColor 主题令牌解析+darkMode 入 deps；mermaid 跟应用 data-hl 而非 OS prefers-color-scheme。
- **辨伪**：ModelCenter ctx（100+ 字段）不 memo——五 hook 返回对象字面量每渲染新引用 deps 无法建立，强行 memo=陈旧 ctx 真 bug（挂观察池）；TTSPlayer/AIConsole 单实例全清无害；z-index 1080 阶梯脆弱但无叠加路径；boot 亮色反闪挂池。
- **测试**：events.test 新建 4 例（双订阅退订隔离+EventsOff 永不调用/space 过滤不变/updater+ready 同纪律）+runtimePolyfill +3 例+useNow 3 例；TaskInboxPanel/WeixinPage/KnowledgePanel 删除用例改两步确认。
- **坑**：①EventsOff 红线是「禁全清」非「禁用」——polyfill 带 callback 只摘自己+通道空才关 SSE 才是完整语义②Popconfirm clone 注入 onClick 覆盖子元素（v4.348 Dropdown 坑家族），子按钮原 onClick 必须移除；无 forwardRef 按钮包不上改命令式③确认钮定位一律 testid（okButtonProps 透传）防「删 除」空格名④go test 全量负载 flaky（足迹零交叠复跑绿）；ci 日志勿 tail 截断否则 FAIL 无包名。
- **门禁**：tsc 0+eslint 0/0（新色值走 // hex-exempt）+全量 ci.ps1 绿（vitest 384 文件 3280 例）+drift OK@706+版本三处 4.350.0；产物见 releases/v4.350.0.md。**文档**=releases/v4.350.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十五迁：v4.347 入 archive）+progress/todos。**未做（下刀池）**=boot 亮色主题启动反闪（index.html 内联预读主题，动 splash 链路）；MarkdownContent pre 底 rgba(0,0,0,0.3) 亮主题脏块；白色半透明 hover 底亮主题失效群（PersonaPicker/imagegen）；--color-scheme 声明；ModelCenter ctx context 拆分（state/actions 两通道）；绘梦历史无上限渲染+dataURL 全量回填（挂载内存膨胀）；ForeshadowPanel 行 memo。

## 最新发布：v4.349.0（2026-09-19）「前端优化五线：可访问性动线 / 亮态可读性 / 运行开销 / 布局 / 稳健性」

- **动机**：用户口径「优化迭代 gaea，进行前端优化」。先四路并行**只读审计**（主题可读性/交互可访问性/渲染性能/布局响应式，各带 file:line 与实测数字），再按证据定刀；审计判错的项当轮辨伪不改。纯前端 18 源码+9 测试，**零 Go 逻辑改动、零绑定变更**。
- **线1 可访问性动线**：记忆中枢全局检索命中行（主入口动线）裸 div → `role=button`+`tabIndex=0`+Enter/Space；轻语记忆面板标题/摘要键盘化 + 确认/取消/编辑/删除四图标钮补 `aria-label`（此前只挂 Tooltip，**不参与可访问名**）；阅读「删除高亮」补 `Popconfirm`（误点即删不可恢复）；Lightbox 关闭钮补名；批准弹窗 `aria-modal` false→true。
- **线2 亮态可读性**：①**真缺陷**——`changesdiff-tok.css` 写死 One Dark 七色而 `ChangesDiff` 容器不设背景（继承面板=亮态浅底），实测 `.tok-typeName #e5c07b` **1.38:1**、`.tok-string` 1.61:1（文件注释自称「浅色下同样可辨识」被证伪）→ 接共享调色板 `--hl-*`（真源 `hljs-theme.css` 已备浅色深阶组并随 data-hl 切；ChangesDiff 显式 import 真源防懒加载链 var 未定义），文件零字面色值；②`lightFn` 下沉：warning `#d97706`(2.90~3.13:1)→`#92400e`、success `#059669`→`#047857`（Moss lime 阶保留）、primary Jade/Rose/Amber→`#0f766e`/`#be123c`/`#b45309`（accentRgb/glow 跟随）；**Violet/Moss/Slate 原达标一律不动**；③新增 `appStore.contrast.test.ts` 12 主题×5 令牌×2 背景矩阵机检 + 深阶锁值 + hljs 浅色组逐令牌 ≥4.5 + tok 文件零字面色值。
- **线3 运行开销**：①`main.tsx` 诊断层 500ms 轮询只写一个时间戳（≈17.3 万次/天纯空转）→ 心跳改由既有 rAF 探测循环每帧维护（跨 IIFE 共享 `mainThreadBeatAt`），诊断层只做 8s 低频判读；**降级路径补 500ms 兜底续跳**（否则误报卡死）；②`getThemeTokens()` 每次返回新对象且留在渲染体 + `ConfigProvider theme={{…}}` 内联字面量 → 每次渲染重跑 ≈45 个 setProperty 与 antd 主题算法 → 双 memo 化；③`SelectionToComposer` 的 selectionchange 先判 `isCollapsed` 短路（原上来就 `toString()` 物化整页选区文本）；④`WhisperMemoryList` 过滤/分组 useMemo + 查询词预小写（原每次渲染 ≈7N 谓词）。
- **线4 布局**：摘要 chip 补 `whitespace-nowrap + shrink-0`（窄列曾标签内折行、「基线 N」整排参差）；基线槽位行 `flexWrap:'wrap'` + 工期/时间戳固定段 `nowrap/flexShrink:0`（原固定部分 326px>minWidth 300 致行内换行）；海报描述补 line-clamp（`overflow:hidden` 定高卡原先静默裁切无省略号）。
- **线5 稳健性**：①书架 `loadProjects` 只 `console.error` → 首页把「读失败」渲染成「书架空空如也」**假空态** → store 增 `projectsError`（只表态不抛出，调用方 await 序列零变化）+ 首页错误态含原因与重试；②「最近文档」仅挂载时读一次而 v4.346 起页面常驻保活 → `lib/recentFiles` 增订阅通道（写入广播 + 解析缓存保快照引用稳定，满足 `useSyncExternalStore`），首页账页与办公「最近文件」条同改订阅（空态挂载后首次写入也出现）。
- **审计辨伪（记录在案，当轮不改）**：①印泥章 `.w-seal #d06055` 不改——20px/**700 粗体**属 WCAG large text（阈值 **3:1**），实测白底 3.83/近白 3.49~3.67 **达标**，按 4.5 判死是误报，改深会漂移品牌记号色（阈值已写入测试注释）；②reduced-motion **无缺口**（`index.css:461` 全局 `*` 兜底经 main.tsx 确认已加载 + GSAP 三 hook 均真实调用 `prefersReducedMotion()`）；③暗态 6 主题×27 配对 **0 失败**（最差 4.69）；亮暗两态 `colorText/colorTextSecondary` 本就 ≥4.63——亮态问题**只**集中在 primary/warning/success 被当文字用；④硬编码 `rgb(255 255 255)` 多落在固定深色 scrim（character-detail.css:141/488）上 = 正确豁免。
- **测试**：新增 4 文件 44 例（对比度矩阵 18 / 记忆面板 5 / 阅读浮层 3 / 列表 memo 5 / 结构守卫 13）+ 扩展 5 文件 8 例（命中行键盘 2 / 首页失败重试 1 / 最近文件广播 2 / 挂载后写入 2 / 选区短路 1）。
- **门禁**：tsc 0 + eslint **0 error 0 warn**（全仓）+ **全量 ci.ps1 绿**（vitest **382 文件 3270 例**）+ drift OK@706（零绑定）+ 版本三处 4.349.0；**首轮 ci 被 frontend build 拦下**（新增测试文件后未重跑 tsc：`[...NodeList]` 需 DOM.Iterable、ReadingAnnotation 缺 nodeId/createdAt）→ 修复后复跑全绿。
- **目检**（vite dev ?mock=1 + 无头 Edge CDP 9347，只读，收工已回收进程与临时 profile）：三页错误边界 0/空白 0；检索「振动锤」5 条命中**全部 `role=button`+`tabindex=0`**；亮态实测 `--md-sys-color-primary #0f766e` / warning `#92400e` / success `#047857` 与锁值一致；`.p-poster-desc` 实测 `webkitLineClamp:4`+`overflow:hidden`；`.p-poster.is-hero` 中段空白实测 **273px**（卡高 420，与审计 276px 吻合=本轮有意不动的拍板项）；console 仅 antd Spin tip 警告 + 自诊断 longtask 66ms，**无 exception**；截图 `.tmp/uiwalk-v4349/*.png`+report.json。
- **产物**：exe **50,865,152 B（48.5 MB）** SHA256=**615319a09773dc693c2be6d943904bae226dd686d797291c905d605b5174ecea**（releases+SUMS，格式 `hash *file` 与历版一致；桌面副本同哈希）；冒烟 health 200 过；保留策略执行留 v4.345~v4.349 删 v4.344；顺带纠回 `releases/README.md` 的「最近 5 版」行与「34 席」索引（此前落后三版）与发布说明计数（254→371）。
- **坑**：①审计阈值必须复核——large text 3:1 被按 4.5 判死会白改品牌色②`--hl-*` 是懒加载 CSS 变量，接色板必须同时 import 真源③source-guard 里 `indexOf('.hl-scope-dark')` 会命中文档头注释，取色块须 `lastIndexOf`④antd 两字按钮可访问名带空格（「删 除」「重 试」），测试按 `/^删除$/` 找不到⑤心跳改 rAF 驱动必须处理降级路径，否则每 8s 误报卡死⑥**新加测试文件后必须重跑 tsc**（tsconfig.app 含 src）。
- **未做（下刀）**：海报墙 hero 中段留白 276px（须加内容槽=内容决策，改动需同时验 1440/1180/1100/760 四断点）；其余可点 div 键盘化（章节树/大纲/统计分类 ~8 处）与 `ResizableDrawer` 手写弹层焦点管理按动线权重分刀；进度计划页 `.gsched` 工具栏 spacer 换行与基线 Popover **本轮未实测**（mock 书斋 rail 无「进度计划」入口，代码级证据在案待真机）；跨窗口最近文件同步（storage 事件）本仓单实例无场景。

## 最新发布：v4.348.0（2026-09-19）「原罪板块优化双刀：故事底稿直注前情 + EPUB 电子书导出」

- **动机**：原罪板块优化完善班。①故事越长前情窗口（12 条×1500 rune）越丢早期设定，模型自记的便签/大纲底稿不在提示词里——要花工具轮 read 自己记的设定，忘 read 就自相矛盾；②故事成品只有 Markdown（插图本地路径引用），阅读器/传设备缺内嵌插图的可分发格式（go-epub 已在依赖，书源线有先例）。
- **刀1 底稿直注**：buildSinUserPrompt 增底稿块（角色块后、前情前）——大纲防御性再截 4000 rune + 便签合计预算 sinPromptNotesBudgetRunes=4000 从头带，超限「其余 N 条未展示」如实报数；空稿=提示词逐字零漂移；回合中途 write 下一轮生效；sinDraftForPrompt fail-open+锁内读。工具描述/系统提示词同步改口「底稿已随前情附上」，通常无需 read（省工具轮），核对单条全文才 read。
- **刀2 EPUB 导出**：SinExportEpub（705→706，play）——每条助手回合一节（第N回中文序数）+用户指令引块置节首+插图内嵌（go-epub AddImage；首图兼封面；webp 不在支持集→「插图未能内嵌」占位）+未生成占位（cue 次序键优先/描述键兜底同 MD）；落 sin/exports 同名不覆盖-2/-3、半截产物即删；前端顶栏导出改下拉（Markdown 另存为/EPUB 后端落盘告知路径，mock 如实抛）。
- **测试**：sin_draft_test 5 用例（空稿逐字一致/注入排序/预算截断+未展示计数/大纲便签独立判空/fail-open）+ sin_export_epub_test 2 用例（zip 全链：正文/引块/img/占位/illu-1 恰 1 张/cover.xhtml/同名-2；守卫：空故事/聊天话题拒绝）；既有 buildSinUserPrompt 三处调用点补 sinNotesDoc{} 断言零改动。
- **门禁**：go build/vet 0+internal/app 全量绿（102s）+tsc 0+eslint 0+vitest 定向 94/94+drift OK@706+spaceBindings 锁 528→529+bindingNames 全量再生+版本三处 4.348.0；全量 ci.ps1 绿；产物=exe 50863104B SHA256=f84e9bd9164f2bd55b1b12220f6963c6251f39247abad8dba1bb8a6a0249ed00（releases+SUMS；桌面副本同哈希；冒烟 health 200 过）；本地 exe 保留策略执行留 v4.344~v4.348 删 v4.343；
- **坑**：①Dropdown clone 子元素注入 onClick 会覆盖 ToolbarButton 的 prop（无 forwardRef 且 onClick 必填）——占位空函数即净，不为原罪改办公组件②单条恰=预算的便签 runes>room 为假应整条收下——截断语义「超了才截」，测试前提咬一次③go-epub AddImage 收本地路径返回值直接作 img src，webp 宁占位不硬塞。
- **未做（下刀）**：sin 真机一条龙走查（书源卡+导出下拉+底稿直注实写）挂闲置窗口池；插图变体历史候立项；章节化结构候拍板。

# 任务进度

> 本文件为**最近发布速览**。完整历史磁带见 `docs/archive/progress-history-2026-09.md`
> 与 `releases/`。

## 最新发布：v4.347.0（2026-09-19）「GLM-5.3-FlashX 新模型接入目录 + Flash 补官方绝对价」

- **来源**：智谱 9 月中旬新发 GLM-5.3-FlashX（Flash 加速版，320B 总参/18B 激活，200 tokens/s）；官方四页交叉核实（模型概览/详情/定价/coding 活动）后接入模型中心 GLM 静态目录。纯数据刀，零新绑定 705。
- **落地**：`glm_catalog.json` +1 条目（1M/128K、caps 同 Flash、国内绝对价 2/7 元/M、缓存命中 0.57 记 price_note）；coding 套餐官方明示暂未开放 FlashX→不配积分系数不进别名；Flash 补绝对价 0.8/2.8 元/M（「仅相对价回退 USD」旧口径作废，估算 0.65 USD→3.6 CNY /M+M）。
- **测试**：锚定清单+计数 45→46 全对齐 ×8；FlashX 元数据/价格锁值；积分守卫+估算锁值翻新；modelengine 包全绿。
- **验收**：drift OK@705；全量 ci 绿；产物见 releases/v4.347.0.md。
- **未做（下刀）**：coding 套餐若开放 FlashX 补 points/别名（覆盖文件可先行）。
## 最新发布：v4.346.0（2026-09-19）「UI 健壮性双修：错误边界页级隔离（keepAlive 连坐根修）+ TisorRadar 缺档崩页」

- **来源**：UI 优化班——隔离目检（vite dev ?mock=1 + 无头 Edge CDP）双空间全页×明暗两态；纯前端四文件，零新绑定 705。
- **根修**：①MainLayout 错误边界下沉 keepAlive 每页内部（原包整组外面=任何一页崩卸载全部页且后续导航全停错误态，keepAlive 状态俱失）②TisorRadar dims 守卫+CharacterCard 调用方守卫（缺档不渲染不出假图）③mock 三角色补 dims 对齐真契约。
- **测试**：TisorRadar.test +3 +CharacterCard.test +1（按 circle 计数断言）定向 19/19；tsc 0。
- **验收**：修复前角色库明暗皆崩；修复后 3 卡×30 circle 雷达在位、双空间明暗全绿；全量 ci 绿；drift OK@705；产物见 releases/v4.346.0.md。
- **未做（下刀）**：亮态 accent 对比度打磨（视觉拍板维持观察池）；动效手感待上手定论；真机走查清池班；技能核数 ≈09-30。
## 最新发布：v4.345.0（2026-09-19）「filewatch 关闭竞态根修：fs 通道关闭路径漏 close(out) 致消费方挂起」

- **来源**：全量 ci 偶发时间型 flaky 深挖——预算拉满 10s 仍挂=真缺陷非调度慢。
- **根因**：loop 三退出路径中 fs 通道关闭两条裸 return 漏 close(out)，与 done 分支 select 竞速，输则消费方 range Events() 永久挂起（~1/20）。
- **修复**：outOnce+closeOut() 三路径幂等关闭；测试 2s 硬超时改终态轮询 10s 预算。
- **验收**：filewatch 20 连跑全绿（修前 1/20 挂）；全量 ci 绿；产物见 releases/v4.345.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.343.0（2026-09-19）「t7 收尾「编辑器标注高亮」轻量版：定位单条镜像 mark + 编辑即失效」

- **来源**：观察池最后一项（v4.325 有意裁剪）的轻量落地；纯前端 EditorPanel+一条 CSS，零新绑定 705。
- **落地**：locate 时镜像背景层（同排版/文本透明仅 mark 显色/textarea 底透出）；滚动条占宽补偿+滚动同步；content/activeNode 变化即失效（偏移漂移结构性规避）。
- **测试**：EditorPanel 域 11/11（+1）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.343.0.md。
- **未做（下刀）**：真机走查清池班（镜像对齐需真机目检后再议持久全量）；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.341.0（2026-09-19）「t7 观察池「情感曲线图形化」：全书体检面板情感弧线折线」

- **来源**：观察池条目（v4.325 记录）；纯前端单文件 BookHealthPanel，零新绑定 705、零新依赖。
- **落地**：体检面板情感曲线区——按需逐章拉分析 V2 情感弧线强度（0-10，缺档跳过计数），纯 SVG 折线（参考线+tooltip+间隙如实呈现），<2 章诚实不足态。
- **测试**：BookHealthPanel 域 7/7（+4）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.341.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t7 余项候拍板。
## 最新发布：v4.344.0（2026-09-19）「t7 最终形态：编辑器持久标注高亮（overlay 常驻 + 编辑退场）」

- **来源**：v4.343 轻量版完全体（真机镜像对齐已过）；纯前端三文件，零新绑定 705、零新依赖。
- **落地**：annotationMarks.ts 分段工具收敛（面板/编辑器同一实现）；EditorPanel annotations 驱动全量 mark 常驻 + dirty 编辑退场 + gutter 实测；CreatePage 随激活章加载标注（alive 守卫）；locate 契约不变。
- **测试**：EditorPanel 11/11（重写+2）+ Panel 11/11（重构等价）+ CreatePage 15/15（补漏桩）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.344.0.md。
- **未做（下刀）**：真机走查清池班（持久 overlay 目检随下一班）；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.342.0（2026-09-19）「t7 观察池「多章对比」最小形态：分析面板章际对比」

- **来源**：观察池条目最小可用形态；纯前端两文件，零新绑定 705。
- **落地**：分析面板「章际对比」——选对比章拉其 V2，四维+情感强度差值表（Δ 上绿下红、情感中性）；未分析行内提示；chapterOptions 缺省整体隐藏。
- **测试**：Panel 域 11/11（+3）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.342.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t7 仅剩 overlay 高亮候拍板。
## 最新发布：v4.340.0（2026-09-19）「t2 观察池「反推任务取消绑定」：AI 反推大纲轮询期可取消」

- **来源**：观察池条目（t2 拆书线 v4.291 体验余项）；纯前端单文件 CreatePage，零新绑定 705。
- **落地**：轮询期「取消反推」钮（taskId 到手门控）→ 停止等待 + GaeaTaskCancel 尽力协作取消（完成竞态被吞，任务中心权威）；同步回落路径不出入口。
- **测试**：CreatePage 域 15/15（+2）；补 NovelOutlineReconstruct/TaskCancel 两处漏桩。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.340.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t2 余项按反馈。
## 最新发布：v4.339.0（2026-09-19）「t7 观察池「标注定位编辑器光标」：分析面板→正文编辑器定位闭环」

- **来源**：观察池条目（v4.325 记录）按既定习惯开刀；零新绑定 705，纯前端三组件。
- **落地**：EditorPanel forwardRef+locate 句柄（rune→code-unit 换算+选区+聚焦滚动）；分析面板标注行「编辑器定位」钮（stopPropagation）；CreatePage 接线（gate 跳他章时入口按章一致条件隐藏）。
- **测试**：EditorPanel +3 + Panel +3（18/18）；tsc/eslint 0。
- **验收**：全量 ci 绿；drift OK@705；产物见 releases/v4.339.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；t7 观察池余项候拍板。
## 最新发布：v4.338.0（2026-09-19）「无参绑定会话语义审计收官 + 死链清理（绑定面 707→705）」

- **来源**：todos 挂账（v4.181.0 起池）收官；审计档 docs/gaea-session-binding-audit-2026-09.md。
- **审计结论**：需修 0 项——会话模型「看历史必经 ResumeSession 切内核」，主流水线无参读内核构造性自洽；v4.181 旁路看板族无同构残余；复审口径入档。
- **顺手清账**：删 GaeaCheckpoints 死链（UI 回退实走 GaeaRewind）+GaeaTCCAReport 孤儿链（state.tcca 零消费）；绑定面 707→705。
- **拍板池**：GaeaSkillDraftFromSession 参数化（从历史会话蒸馏技能）候拍板。
- **验收**：go build/vet 0+internal/app ok；tsc/eslint 0；store 域 38/38；drift OK@705；全量 ci 绿；产物见 releases/v4.338.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；TaskCenter 会话关联结构刀。
## 最新发布：v4.337.0（2026-09-18）「办公对话流美化三轮收官：对齐 Codex web（降噪+动线+语义归位）」

- **来源**：办公对话流美化线程收口发版（在途三轮合一版）；纯前端零新绑定 707，零 Go 改动。
- **落地**：用户消息气泡化；裸思考行 bare 模式（行数 meta 三语+英文单复数）；过程卡完成即收起；WorkHeader 三改（doneSuppressed/纯问答 0 步不渲染/运行态对齐过程条）；尾随工具段合并（过程卡归位正文前）；轮尾登记卡补挂+omitPaths 跨段去重；交付卡统一边框容器；消息操作 hover 淡入；顶条只留 waiting 档；ToolGroup「bash × 3」三语。
- **测试**：ProcessCard 展开态测试重写钉新行为（旧测试钉已废行为，全量首跑 1 红）；vitest 3199/3199；tsc/eslint 零；三语 +4 键同轴。
- **目检**：vite dev 9344 mock=demo+独立 Edge 9345 CDP 两轮五态截图全过（bare「思考 1 行」实测）。
- **验收**：全量 ci.ps1 绿；drift OK@707；产物见 releases/v4.337.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；绘梦阶段二候拍板。
## 最新发布：v4.336.0（2026-09-18）「chapter-gate 通知跳转 + oh-story T5 落库接线」

- **来源**：用户指令「继续」；规格书 进度计划/gaea-gate-jump-t5-wiring-20260918.md。
- **甲件**：自动门通知可点击——onOpen 回调+CreatePage 覆盖态+跨章拉正文锚定；无回调零变化（v4.331 观察池头名收口）。
- **乙件**：T5 七角色卡升格可派发——SKILL.md runAs=subagent+派发契约；spawn 写作域模板 subagent_writing+boot 映射；验收钉 Go +3。oh-story 只剩写前契约硬闸（等上游）。
- **验收**：全量 ci 绿；drift OK@707；产物见 releases/v4.336.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；绘梦阶段二候拍板。

## 最新发布：v4.335.0（2026-09-17）「办公输入动线对齐 DSH/Codex：运行中 Enter=排队，插话改显式」

- **来源**：用户报告「办公板块消息输入的排序/插话/撤回不见了，直接插入正在跑的对话中」；拍板「按 DSH/Codex 方式处理」。
- **归因（非回归）**：队列本体（v4.200/201）一直在，v3.6.0 起 Enter=插话、排队退到 Clock 小按钮——队列卡从未被 Enter 动线触达=体感丢失。
- **落地**：Enter=排队（队列卡回主动线，回合结束 FIFO 派发）；Alt+Enter=直插（显式插话）；Shift+Enter 纠正/Esc 停止不变；三语文案+title 三态更新；steerTitle 键删（0 死键）；组装逻辑抽 helper。零绑定 707。
- **验收**：Composer.queue +3/I18n 对齐；composer 域 22/22；tsc/eslint/ci.ps1 全绿；产物见 releases/v4.335.0.md。
- **未做（下刀）**：真机走查清池班；技能核数 ≈09-30；绘梦阶段二候拍板。

## 最新发布：v4.334.0（2026-09-17）「revolution 刀8 收官：跨模块验收+回退演练+版本三处漂移根治——**revolution 刀 1–8 全线收官**」

- **来源**：用户指令「继续优化迭代 gaea」；三线并发子代理+主代理收口；规格书 进度计划/gaea-rev-knife8-acceptance-20260917.md。
- **跨模块验收 Go +7（零生产代码改动）**：线A=AI 味性能钉（3.3~5ms<1s）+去味保真（分降/rune 漂移 0/实体句数不变）；线B=生成主轴端到端史上首测（双路桩贯通自动门成功路径+chapter-gate 事件 httpbridge 捕获）+50 章书级体检；线C=审批链 app 级（approved=false 零入库/journal 回放一致/账本只增）+快照往返+真 v3 迁移非破坏。
- **版本漂移根治**：app_info.go/versioninfo.rc 停 4.323.0（十版未跑 sync-version）——三处同步 4.334.0+ci.ps1 新增 version drift check 段；顺手 eslint 5→0（EXTRACT_OPTIONS 归位 novelOptions.ts）。
- **验收**：go vet/test 全量绿；vitest 3185/3185；tsc 零错；drift OK@707（零新绑定）；产物=exe 50832384B SHA256=4d2a01c5…81bc（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：RetryJSON 桩回包须包 ```json 围栏；rebuildScenesFromBlob 无场景章 no-op；project.Create 直写 v4 标记（旧 v3 夹具假设过时）。
- **未做（下刀）**：真机走查清池班（v4.319~v4.334 挂池，须闲置窗口）；技能核数 ≈09-30；绘梦阶段二与缺省形态翻转候拍板。

## 最新发布：v4.333.0（2026-09-17）「阶段七出口对账 + 7.3-1 会话级回源收口」

- **来源**：用户指令「继续」；规格书 进度计划/gaea-stage7-audit-resume-20260917.md。
- **会话级回源**：pendingSessionResume 双态通道（keepAlive 事件/冷启 pending）+gaea App 消费（ref 防过期闭包）+收件箱精确回源（V1 板块粒度回退零变化）。零新绑定 707。
- **对账**：7.1/7.2/7.3 已落、7.4 已清；挂账=本地建议样本（真机）/复跑抽样+≥5 次（≈09-30）/总判据⑤走查班。
- **验收**：前端 +5；go 全量复跑绿+vitest 3185/3185；ci 第三跑 OK（首两撞环境竞争先例+2）；产物=exe 50832384B SHA256=89d1de3d…0ba2（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **未做（下刀）**：真机走查清池班（须闲置窗口）；技能核数 ≈09-30；刀8 余项/绘梦阶段二候拍板。

## 最新发布：v4.332.0（2026-09-17）「7.3-2 板块降级为任务视图（阶段七『层跃升』刀）」

- **来源**：用户「开始吧」提前拍板解锁（原窗口 ≈09-23）；规格书 进度计划/gaea-tasks-first-home-20260917.md。**本版=阶段七层跃升版本**。
- **论点**：首页从「用哪个工具」翻转为「什么在等我」。裁决=homeLayout 开关缺省 classic（渐进零变化）；TaskInboxBoard 内联抽取（面板零破坏）；回退开关双位；零新绑定 707；manifest 零改动。
- **落地**：appStore homeLayout+守卫；TasksFirstHome（收件箱大区/最近会话/能力 chips 全量直达/切回经典）；ModuleLauncher 分支+SpaceSwitch 快捷钮；设置页 HomeLayoutPanel；三语 12 键。
- **验收**：前端 +10；零回归实证（TaskInboxPanel 9/9+ModuleLauncher 13/13 零改动）；vitest 3180/3180 一次全绿；ci.ps1 全绿；产物=exe 50831872B SHA256=80dacda4…d7eb（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：测试尾追加吞 describe 闭合；LocaleProvider 包裹；ESM 无 require。
- **未做（下刀）**：阶段七出口判据复核对账（三门收口）；刀8 余项；绘梦阶段二提案候拍板。

## 最新发布：v4.331.0（2026-09-17）「自动门可见化 + CI race 扩面（刀8 race 项收口）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-gate-notice-race-20260917.md。
- **论点**：自动门零感知（事件无消费）+race 门禁只盖办公包。落地=useChapterGateNotice 轻通知（四维拼装/防叠/分支标记/无通道静默）挂 CreatePage；CI race job 扩面六包（app 注释不进候评估）。零新绑定 707。
- **验收**：前端 +3；vitest 3172/3172 一次全绿；六包本地全绿；ci.ps1 全绿；产物=exe 50824704B SHA256=a5e239f2…3198（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：vi.stubGlobal('window') 整窗替换炸 react-dom——只换 window.runtime 属性。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；刀8 余项（验收演练）候排。

## 最新发布：v4.330.0（2026-09-17）「刀7续收官：POV 视图（场景圣经消费面）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-pov-view-20260917.md。前置核对：场景生成/脑图/指纹已落，刀7续仅剩 POV 视图；revolution 勾选对账后只剩刀8。
- **论点**：POV 掩码只在生成链内部消费，视角纪律不可审视。裁决=+1 绑定（707）双路编译（场景/整章合成）；视图 camelCase 恒非 nil；空区段如实空。
- **落地**：novel_scene_bible.go wire 投影；SceneBibleDrawer（已知绿/不知情红对照+角色卡状态回灌）；ChapterEditor「视角」按钮。锁 529→530；drift OK@707。
- **验收**：Go +3+前端 +3；ChapterEditor 7/7 零破坏；vitest 3169/3169 一次全绿；ci.ps1 全绿；产物=exe 50823680B SHA256=4ea48652…4eeb（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；刀8 race 门禁（CI 侧）。

## 最新发布：v4.329.0（2026-09-17）「绘梦阶段一刀 E：画室消耗（月度聚合+折叠面板）——绘梦阶段一全清」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-studio-usage-20260917.md。**绘梦阶段一（A 资产面板/B 参考槽/C 指令编辑/D 配图 v2/E 画室消耗）全清**。
- **论点**：台账有 Cost+CreatedAt 无聚合视图。裁决=+1 绑定（706）；当月过滤×模型分组；记录单价为事实源（缺价回落目录）；可解析才估算、未定价诚实计数；全免费显示 0；不用积分话术。
- **落地**：imagehub_usage.go 聚合（cwd 参数化供测试）；StudioUsagePanel 折叠面板挂 ImageGenPage；bridge/mock/锁 529/bindingNames 706。
- **验收**：Go +4+前端 +4；vitest 3166/3166 一次全绿；ci.ps1 全绿；产物=exe 50809856B SHA256=90e9ec56…ad78（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：裸 "0" 单价解析、gaeaCwd 全局态、mock 尾锚点。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；绘梦阶段二规划拍板。

## 最新发布：v4.328.0（2026-09-17）「绘梦阶段一刀 D 核心片：章节配图管线 v2（角色参考图+风格槽）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-scene-illustration-v2-20260917.md。
- **论点**：配图纯文生图（一致性靠文字、风格写死）。裁决=签名扩展空串零变化（绑定面 705 不变）；参考路由按后端能力附 img2img（SceneRefDenoise=0.6）或 refNote 诚实降级；风格槽替换风格描述。
- **落地**：chapter.Agent +V2 方法；handler opts+参考路由+readImageFileAsDataURL；前端风格输入+角色勾选+refNote 行；修复挂载效应自动重跑 bug（didMountRun 守卫）。
- **验收**：Go +7+前端 3 改 5；vitest 3162/3162 一次全绿；ci.ps1 全绿；产物=exe 50796032B SHA256=070f39f8…6c39（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **观察池**：书封参考；素材库变体/设书封；灯箱共用；denoise 可调。
- **未做（下刀）**：刀 E 画室消耗轻量面板或 7.3-2（≈09-23）。

## 最新发布：v4.327.0（2026-09-17）「绘梦阶段一刀 C：指令编辑『改图』MVP（云端先行）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-instruct-edit-20260917.md。前置核对：刀 A/B/E 已落，刀 C 是真缺口（改图=整幅重绘无语义编辑）。
- **论点**：gaea OpenAI 兼容图片面缺 /images/edits（Qwen-Image-Edit 系云端网关标准面）。裁决=复用请求字段（Mode=edit）；三后端诚实口径（openai 实现/glm·comfyui 拒绝）；零新绑定 705 不动。
- **落地**：image_openai.go editImage multipart 分支+parseImageResponse 共用提取；GenerateMedia edit 原图必填；InstructionEditModal（对照+用到画布）+ResultStage「指令编辑」Action+ImageGenPage 接线。
- **验收**：Go +5+前端 +4；ResultStage 5/5 零破坏；ci.ps1 全绿（FilePreviewModal 1 例 flaky 隔离绿，两日两现候选常驻）；产物=exe 50785280B SHA256=855b56f1…9ae0（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **观察池**：mask/保留区域；ComfyUI 本地档；多图输入；edit-of-edit 谱系；刀 D/刀 E 轻量收尾。
- **未做（下刀）**：绘梦阶段一余项（刀 D 章节配图 v2/刀 E 画室消耗）或 7.3-2（≈09-23）。

## 最新发布：v4.326.0（2026-09-16）「GenerationGate 闭环收口：生成后自动分析门 + 全书体检」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-gen-gate-closure-20260916.md；阶段七 §4 在册项（revolution「GenerationGate 完整闭环」未勾项收口）。
- **论点**：生成 done 只挂去味+AI 味分+角色提取，分析路（V2 落盘/伏笔同步/记忆回填）从不自动发生——三条已建成管道水源靠手动。裁决=自动门异步默认武装；只跑确定性三路+分析路一次 LLM（review/consistency 留手动）；分支章跳分析路；chapter-gate 事件零消费（t7 面板天然消费面）；全书体检纯确定性零 LLM。
- **落地**：novel_book_health.go（buildAutoGateReport 同步可测+runAutoGateAfterGeneration go 位+RunBookHealthCheck 逐章三路+伏笔 Lint 复用+V2 覆盖）；create_chapter_handler done 后挂接；NovelB +1（704→705）；BookHealthPanel（聚合卡+最差 AI 味告警+逐章表红标+伏笔 findings）；锁 527→528；drift OK@705。
- **验收**：Go +5（自动门三例+体检两例）+前端 +5；CreatePage 既有测试零破坏；tsc/eslint 零错；产物=exe 50774528B SHA256=0581c0be3ea4a61f33989eecf4ebd666a287cd648a10ec2a21e5da304aa5f83b（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **坑**：analysis.New(nil client) 在 Analyze 内 panic——测试用无引擎真 client；JSX 泛型不认索引访问抽别名；无大纲节点=无契约可违不计。
- **出口**：生成后 V2/伏笔同步/记忆回填自动发生；全书体检聚合报告零 LLM；revolution 未勾项勾销。
- **未做（下刀）**：7.3-2 板块降视图（≈09-23）；闲庭在册清欠沿列车。

## 最新发布：v4.325.0（2026-09-16）「t7 前端统一接线收官：分析 V2 消费面板 + 标注高亮视图」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-analysis-v2-panel-t7-20260916.md。**本刀合入=小说·MuMu 蒸馏六域前端接线全清（t7 收官）**。
- **论点**：分析 V2（九维落盘）与标注层（keyword→rune 偏移）后端在位 UI 零消费；AnalyzeChapter 困在 Legacy 面零入口。裁决=types 直连（不复制九维子类型树）；缺章诚实报错接分析按钮闭环；高亮 V1=面板只读视图不动编辑器；rune→code-unit 换算+交叠裁剪。
- **落地**：analysis_handler.go +NovelChapterAnalysisV2（703→704）；AnalyzeChapter 出 Legacy 转正（AppBindings+mock）；ChapterAnalysisPanel（头部评分+chips+meta、九维分节空节隐藏、标注区 列表|高亮视图 双模式+锚点定位）；CreatePage rail「章节分析」。锁 525→527；drift OK@704。
- **验收**：Go +3+前端 +8（含交叠裁剪断言）；CreatePage 13/13 零破坏；vitest 全量 3151 例（1 例环境 flaky 隔离绿）；tsc/eslint 零错；产物=exe 50756096B SHA256=6604f8e06b3f27cd586053794bfe22d8b881851e72210a7e7b0e2aeb22d89528（releases/gaea-v4.325.0.exe+SHA256SUMS-v4.325.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）
- **出口**：分析按钮→V2 落盘→九维可见；标注列表+只读高亮+锚点；AnalyzeChapter 出 Legacy；既有面零回归。
- **观察池**：编辑器 overlay 持久高亮；标注定位编辑器光标；多章对比；情感曲线图形化。
- **未做（下刀）**：7.3-2 板块降视图（等 v4.318 零功能周≈09-23）；闲庭在册清欠沿既有列车。

## 最新发布：v4.324.0（2026-09-16）「t6-C2 提示词工坊第二刀：模板包导入导出（content_hash 三态）」

- **来源**：用户指令「继续优化迭代 gaea」。小刀单线主代理直做；规格书 进度计划/gaea-prompt-bundle-t6c2-20260916.md；蒸馏依据 docs/distill/06-prompt-workshop.md §6.4+§11.4。
- **论点**：覆盖表只有一份本地状态——换机/重装无搬运手段、内置升级无对账工具。补全量快照 JSON 包+三态导入（kept/converted/created_or_updated + gaea 三闸 invalid/unknown/duplicate）；对 MuMu 升级=导入真校验（error 跳行不阻断整包）+去重；裁剪=未知键不建行。
- **落地**：internal/promptstore/bundle.go（ContentHash/SameTemplate canonical/BuildBundle/ImportBundle 零 IO 纯函数）；gaea_prompt_store.go +2 绑定（Export 回 JSON 字符串/Import 三态落盘）；面板顶栏导出（saveExportBlob 双门）/导入（pickFileAsFile 双门+结果 Modal）；mock/契约同步。绑定面 701→703（NovelB +2 显式覆盖表归域——v4.323 同款坑复现即修；play 锁 523→525）。
- **验收**：Go +4 测试函数（三态矩阵 10 分支+round-trip 内核+回魂 e2e）+前端 +4；tsc -b 零错；vitest 全量 3143 例（1 例环境 flaky 复跑绿）；drift OK@703；产物=exe 50734592B SHA256=5c1b3916518cd268f84ec9dfc6b61a3b3746c7d12f778151c1076736380d0abc（releases+SUMS+桌面副本同哈希；冒烟 200 过）。
- **出口**：干净环境导入覆盖逐字节回魂；内置升级三态分支钉死；坏模板行不落盘整包其余正常；壳内导出/导入双门。
- **观察池**：项目级 scope；自建模板键；升级合并交互；按书切换配方；triggers；模板包签名/加密。
- **未做（下刀）**：t7 前端统一接线（小说线最后一域）；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.323.0（2026-09-16）「t6 提示词工坊首刀：模板可编辑覆盖层（{{name}} 渲染+三级解析+工坊面板）」

- **来源**：用户指令「继续优化迭代 gaea，记得使用子代理」。三线并行子代理（A 引擎纯函数/B handler/C 前端）→ 主代理收口；规格书 进度计划/gaea-prompt-workshop-t6-20260916.md；蒸馏依据 docs/distill/06-prompt-workshop.md。
- **论点**：20 个 prompts/*.json 只读两层用户改不了；占位符单点硬编码；保存零校验（MuMu §9.3 同型）。裁决={{name}} 统一（旧语法兼容）；V1 只全局覆盖；裁剪云端工坊三表；列表零正文。
- **落地**：internal/prompt +四元数据字段+RenderPlaceholders+SetOverride 三级解析+Names；新包 promptstore 六规则校验+Upsert 自增；prompts ×20 元数据+2 文件占位符迁移；gaea_prompt_store.go 状态文件+五 App 方法+两装配点挂钩+substituteWordCount 双语法；PromptWorkshopPanel 三区+api/prompt.ts+CreatePage 入口+mock 五档。绑定面 696→701（NovelB +5 显式覆盖表归域；play 锁 518→523）。
- **验收**：Go +22 测试函数+前端 +12（CreatePage 13/13 零破坏）；tsc -b 零错；ci.ps1 全绿 exit 0（go 130 包+vitest 367 文件 3138 例）；drift OK@701；产物=exe 50754048B SHA256=1a712949e53414c308ae8a14f85de914d95e021b47fc9711f3ae3e880fdcf4d6（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
- **出口**：20 模板可看可改可恢复；覆盖即时生效（eng.Get 三级解析全链共享）；保存真校验；{{name}} 统一。
- **观察池**：模板包导入导出（content_hash 三态，t6-C2）；项目级 scope；自建模板键；triggers；升级合并交互；风格注入与 novelstyle 打通。
- **未做（下刀）**：t6-C2 模板包导入导出 / t7 前端统一接线（小说线最后两域）；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.322.0（2026-09-16）「t4-C3 收官：场景工程整章重写（单场景替换）」

- **来源**：用户指令「继续」。小刀单线主代理直做；规格书 进度计划/gaea-scene-rewrite-20260916.md。
- **产品裁决**：单场景替换（应用/Restore 对称）；LLM 分场景回写与场景粒度重写入观察池。
- **落地**：移除 whole v4 拒绝+stitch 读；Apply/Restore +rebuildScenesFromBlob（复用既有原语零新机制）；partial v4 守卫；EditChrome 双按钮+ChapterPage 挂两弹窗。零新绑定 696。
- **验收**：Go +2+前端 +2（11/11）；ci.ps1 全绿 exit 0（go 129 包+vitest 365 文件）；drift OK@696；产物=exe 50638848B SHA256=5b1b105ad81bff7b5dd391cc445aeda8cb77aed575a516c996e10441c38ea29e（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
- **出口**：t4-C3 余项全清（三通道×两存储矩阵仅剩 partial×场景=守卫+观察池）。
- **未做（下刀）**：t6 提示词工坊/t7 前端统一接线；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.321.0（2026-09-16）「t4-C3 余项：partial 选段局部重写（版本库升级版）」

- **来源**：用户指令「可继续」。三线并行子代理（A/B/C 足迹互斥全部一次绿）→ 主代理收口；规格书 进度计划/gaea-partial-rewrite-20260916.md。
- **论点**：整章重写对「改一段」太重；partial=选区+指令+长度模式只重写选中片段。对 MuMu 升级=走版本库（全文快照可回滚）；全链 rune 口径；零新绑定（696 不动）。
- **落地**：rewrite/partial.go 四纯函数+NormalizeRequest 五规则（whole 零字节变化）+novelChapterRewritePartial（拼接钉死前后文零变化+版本字段）+前端 EditorPanel 选区 rune 换算/PartialRewriteModal 三态/CreatePage 挂载。
- **验收**：rewrite 9 函数+app +3+前端新增 6（合并定向 34 全绿）；ci.ps1 全绿 exit 0（go 129 包+vitest 365 文件）；drift OK@696；产物=exe 50638336B SHA256=c0f1abd770b4b4a2253c17e9c92d912f06f7992fbcbc62fe155e1e456a97fa7f（releases+SUMS；桌面副本同哈希；冒烟 200 过）。
- **观察池**：选段高亮回写编辑器；流式输出；deslop 入口（类型已留）。
- **未做（下刀）**：t4-C3 余项剩场景工程整章重写；t6 提示词工坊/t7 前端统一接线；7.3-2 板块降视图（等零功能周）。

## 最新发布：v4.320.0（2026-09-16）「t4-C3 余项：重写版本历史面板（版本库前端消费）」

- **来源**：用户指令「继续」。小刀单线主代理直做（纯前端后端零改动）；契约先行（规格书 进度计划/gaea-rewrite-history-panel-20260916.md）。
- **论点**：v4.304 版本库只有「刚重写完那一刻」的 RewriteModal 能操作，List/Get 两绑定零 UI 消费——关弹窗/换会话后历史版本不可看不可用（v4.305「用户到不了的功能等于没有」同型）。
- **落地**：RewriteHistoryPanel（多行展开+详情懒拉缓存+动作镜像后端状态机门控：completed→应用+放弃/discarded→应用/applied→恢复原文/其余只读；Popconfirm 显式 okText）+CreatePage rail「重写历史」入口（打开才拉）+mock 两样本。零后端：绑定面 696/锁数不动、drift OK@696。
- **验收**：面板 5 用例+CreatePage 既有 13/13 零破坏；tsc/eslint 零错；ci.ps1 全绿 exit 0（go 129 包+vitest 364 文件）；产物=exe 50613248B SHA256=4bc1d56d44a5f97a92fde68d1fb8b18026dec5d0c4a8c6fec19731fca8dca837（releases/gaea-v4.320.0.exe+SUMS；桌面副本同哈希；冒烟 /api/health 200 过）。
- **观察池**：双栏 diff 视图；partial 入口随 partial 刀；按模式/状态过滤。
- **未做（下刀）**：t4-C3 余项剩 partial 局部重写+场景工程；t6 提示词工坊/t7 前端接线；7.3-2 板块降视图（等 v4.318 稳定一个零功能周）。

## 最新发布：v4.319.0（2026-09-16）「7.2-2 判据②收口：结晶技能调用计数（skill_stats）」

- **来源**：用户指令「继续」。小刀单线主代理直做（体量一轮未拆子代理）；契约先行（规格书 进度计划/gaea-skill-usage-stats-7-2-2-20260916.md）。
- **论点**：v4.317 出口判据②欠账「结晶技能调用 ≥5 次且成功率可见」：现状核实=技能计数是前端会话级（useToolStats 只算当前会话 run_skill，不持久、无成功率、漏 read_skill 主消费通道——boot 按需加载器才是结晶技能被「翻开来用」的时刻）；本刀补**工具级调用计数**：read_skill+run_skill 双钩子 → <DataRoot>/skill_stats.json 持久化 → GaeaSkillStats 绑定 → 能力面板技能行「N 次 · 成功率 X%」。诚实口径（V1）=成功率工具级（read_skill 交付正文/run_skill 管线无错=ok；读不到名也计 fail 不静默）；回合级留观察池；「≥5 次」使用事实门槛不做硬闸。
- **落地**：①新纯包 internal/skillstats（零 IO 表驱动，先例 routesuggest/taskinbox）：Record（空名忽略/MaxSkills=500 新名拒收防漂移堆积/零值 File 自动建表）+View（calls 降序→name 升序稳定）+JSON camelCase 钉死。②boot 双钩子：Options.OnSkillUse（缺省 nil 零行为变化，EmitSubagentText 先例）+sysprompt resolver 包装（read_skill 失败也计）+run_skill 装饰器 skillUseCountTool（args.name 解析失败不计宁少勿扰；CompactDescriptor 回退语义同注册表缺省）。③App：skill_stats.json route_suggestions 同款容错读（缺失/损坏/version≠1 回空表）+原子写+skillStatsMu 串行；recordSkillUse 包级回调；绑定 GaeaSkillStats（OfficeB +1，**695→696**）只读现算。④前端：CapabilitiesPanel open 态拉 app.SkillStats 一次零轮询（reload 随引擎热加载刷新）→SkillsSection 可选 usage prop→SkillRow 徽标行灰字「N 次 · 成功率 X%」（skill-usage-stat；缺省/零调用不渲染，会话 chip 语义不变）+SkillStatView（types/memory）+三语 +2 键+mock 桩+spaceBindings work+1（锁 517→518）。
- **验收**：skillstats 5 函数+app 3 例+前端 2 例；tsc -b 零错、eslint 零告警；ci.ps1 全绿 exit 0 一次过（go 129 包+vitest 363 文件全过）、drift OK@696、版本三处 4.319.0；产物=exe 50606592B SHA256=43739c69ad424a8356a0994525d3d4d93c4dc367e6037e00802bf249b0641de1（releases/gaea-v4.319.0.exe+SUMS；桌面副本同哈希；冒烟 /api/health 200 过）。
- **观察池**：回合级成功率（点赞点踩/重生成信号另刀）；蒸馏建议 tab 内联已结晶技能计数；两周后真机清池核「≥5 次」。
- **未做（下刀）**：7.3-2 板块降视图（等 v4.318 稳定一个零功能周）；闲庭在册清欠沿既有列车。

## 最新发布：v4.318.0（2026-09-16）「阶段七第五刀：多入口统一任务收件箱（任务持久实体+四入口存为任务+状态机追踪，7.3-1）」

- **来源**：用户指令「继续推进」（沿「继续 并行使用子代理，优化迭代 gaea」习惯）。契约先行（规格书 进度计划/gaea-task-inbox-7-3-1-20260915.md：A/B/C 三线足迹互斥+出口对照）→ A/B/C 三线并行 → 主代理收口。三线全部一次绿，**无卡死接管**（v4.316/v4.317 连续两刀接管后首回全并行一次过）。
- **论点**：四入口（意图中枢/微信/语音/Ctrl+K）v4.5 已汇同一内核（intent.Parse→routeIntent），本刀补「任务」为**持久实体**+统一收件箱：指令除即时执行外可「存为任务」→ 状态机（待处理/进行中/已完成/已放弃）→ 跳回来源板块。**收件箱是清单不是调度器**（零轮询零定时器零后台事件，读写均用户动作触发=「关闭即停」昼夜运转拍板）；space 必带 work/play 隔离（跨空间仅显式，不做移动/复制）；不加新板块（挂既有双空间首页）。
- **线A**：新纯包 internal/taskinbox（零 IO 表驱动，先例 skilldistill）：CanTransition 四条合法迁移（pending→doing→done、pending|doing→abandoned；终态拒迁同值恒 false）+NormalizeTitle（哨兵 ErrEmptyTitle、120 rune 截断）+ValidSource 五来源（ctrlk|palette|voice|weixin|inbox）+ParseTaskID（ti-+12hex 残渣防御）+FilterBySpace（''=全部，GaeaTaskList 变参先例）+Sort（状态组序→组内 UpdatedAt 降序→ID 稳定=行动优先）。9 测试函数 ~104 断言（JSON camelCase 形状逐字钉死）。
- **线B**：intent 新动作 save_task（存为任务X/记个任务X/帮我添加一个任务X/任务：X/待办：X 首尾锚定两式，排提醒后让位——「提醒我存个任务明天开会」归提醒；空标题不命中坠回聊天宁漏勿误；+9 表用例）+状态文件 <DataRoot>/task_inbox.json（{version:1,tasks:[]} camelCase，route_suggestions 同款容错读+temp+rename 原子写）+四绑定（OfficeB +4）：List（FilterBySpace+Sort 损坏回空表）/Save（id 空=新建 pending+ti-+12hex crypto/rand+超 500 拒；id 非空只改 title/note——**Source/Action/Target/Session/CreatedAt 不可变=来源审计链**；source 缺省兜底 inbox）/SetStatus（同值短路零落盘）/Delete；执行层 execSaveTask：语音/微信走 routeIntent→space=gaeaEffectiveSpace() 归一回退 work、assistantID 判 weixin/voice、reply「已存入任务收件箱：<title>」；**Ctrl+K/面板不走后端执行**——dry-run 照常命中，前端直调 Save（shell 空间更准，来源可区分）。app 9 测试函数（camelCase 钉死禁 created_at 泄漏/"WORK" 大写不泄漏/dry-run 零落盘）。
- **线C**：TaskInboxPanel（antd Modal 四档 tab 计数、CanTransition 前端镜像非法组合按钮不渲染、action=navigate→去板块/session→回会话 V1 板块粒度、V3Empty 两分；9 用例）+双空间首页挂点（书斋 w-vitals 第五节 desk-task-inbox；闲庭 p-foot 第五节 garden-task-inbox 内联 gridColumn '1 / -1' 跨全列——固定四列网格零 CSS 改动）+四入口（Ctrl+K 指令卡〔存为任务〕次按钮+save_task「执行」短路直调 Save；面板 query 非空动态项「存为任务『query』」**捕获期 input 监听**镜像输入——CommandPalette 契约不动，观察池列清退方案；语音/微信零前端改动）+lib/types 本地重述 TaskInboxView（herdsman 防环先例）+三语 +28 键（zh=en=zh-TW=1518 逐键相等）。
- **主代理收口**：gen_bindings+bindingNames 再生 695（drift OK@695；bindings_novel 多行委托规整单行 +101/−101 零语义——v4.313/4.317 重排噪音同款甄别）+bridge/core +4（同名前缀无需映射，GetProgrammingWebStatus 先例）+mock/chat 四桩（两空间样本内存态）+spaceBindings 四名 shared（隔离由 space 参数承担，UnifiedSearch 口径）+收口修 5 处 tsc（bindingNames 再生丢 as const→drift 串味；mock vi.fn 空数组推断 never[]→补 Promise<TaskInboxView[]> 完整样本；antd Text 无 rows 配置→Paragraph）。
- **绑定面**：691→695（OfficeB +4）；锁数 513→517（shared +4）；drift OK@695。
- **出口对账**：①四入口来源可区分 ✅（五 source 落库，测试钉死）③空间隔离零泄漏 ✅（space 必带+严格过滤+"WORK" 大写不泄漏）④零常驻 ✅（无轮询/定时器/事件订阅）；②板块级 ✅（Target 精确导航）、**会话级欠账**——按路径恢复 seam 存在（palette sessionItems→onResumeSession(path)）但活在 gaea App 树内首页拿不到，Session 字段已落库数据就绪后续刀接。
- **验收**：taskinbox 9 函数+app 9 函数+intent +9；前端 33/33（Panel 9+ModuleLauncher 12+SearchModal 8+锁数 4）+App palette/export 源级锁 6/6；tsc -b 零错、eslint 零告警。
- **门禁**：ci.ps1 全绿 exit 0（一次过）、drift OK@695、版本四处 4.318.0；产物=exe 50592256B SHA256=63be1263d9077006a5b312f552abb2984787e9f4d3553ecaeb31bfdf3aaeb311（releases/gaea-v4.318.0.exe+SUMS；桌面副本同哈希；冒烟 /api/health 200 过）。
- **观察池**：命令面板 query DOM 捕获→CommandPalette 可选 prop 清退（约 6 行）；精确回源会话接 seam（面板挂工作台或 resumeSession 提全局事件）；任务模板联动（GaeaTaskTemplates 从模板建任务）；LLM 兜底分类器对 save_task 覆盖；releases exe 留版数待用户拍板（沿 v4.315 观察项）。
- **未做（下刀）**：7.3-2 板块降视图（依赖本刀稳定一个零功能周）或 7.2-2 判据②调用计数小刀（statsFile 先例）；闲庭在册清欠沿既有列车。

## 最新发布：v4.317.0（2026-09-15）「阶段七第四刀：journal 历史蒸馏（重复模式挖掘+结晶审阅+审计链，7.2-2）」

- **来源**：用户指令「继续并行使用子代理，优化迭代 gaea」。契约先行（规格书 进度计划/gaea-journal-distill-7-2-2-20260915.md：契约+足迹互斥表+出口对照）→ A/B/C 三线并行 → 主代理收口。**坑**=A 线纯函数包子代理跑满上下文**零落盘**——主代理接管代写（卡死判据先例再+1：v4.316 B 线同款「长时间零落盘」）；B/C 两线正常交付（B 10/10、C 12/12）。
- **论点**：journal 证据链（.gaea/work/journal/*.jsonl，每卡=一次已应用变更）是书斋审批链既有沉淀，把它变成程序性记忆矿床——确定性挖掘跨会话重复 ≥3 次的工具应用模式 → 建议卡只提示不打扰 → 点「结晶为技能」→ LLM 从多次执行回放蒸馏草稿 → 复用 7.2-1 审阅弹窗编辑落盘 → 决定与结晶全程审计。**7.2 技能结晶双入口齐装**（演示式录制 v4.313+历史蒸馏本刀）。
- **线A（主代理代写）**：新纯包 internal/skilldistill（零 IO 表驱动，先例 routesuggest）：MineFlows 会话聚合确定性排序+Distill（连续同 (Tool,Shape) 折叠→会话级序列精确匹配分组→Repeat=不同会话数≥3→步数∈[2,12]→ignored 静默→截 3 宁少勿扰）+BuildPatternID（jd-+sha256 前 8hex 幂等）+ParsePatternID 残渣防御。V1 裁决=滑窗/子序列不进（观察池）。
- **线B**：internal/app/gaea_skill_distill.go 三绑定（OfficeB +3）——Candidates（journal 现算只读零 LLM，ignored∪crystallized 双静默）/Draft（buildJournalReplay ≤3 最近会话分段+AfterSummary 200 rune+整段 600 rune 防护→prompts/skill-from-journal.json（RTCO：多次执行共性进步骤、差异进 cautions、文件名通用化占位）→RetryJSON 2→复用 SkillRecordResult 只回不落盘）/Decide（ignore|crystallized；crystallized 必带技能名，审计条目=sessions+skill+decidedAt）；状态 <DataRoot>/skill_distill_state.json route_suggestions 同款容错读+原子写。
- **线C**：SkillDistillSection（PatternLine 步骤序+重复徽章+证据折叠+结晶/不再提示同卡互斥 loading+空态两分）挂 MemoryPanel 建议 tab「流程蒸馏」分区（props 全可选缺省隐藏，既有调用方零改动）；SkillRecordModal 加 preload/onSaved 可选（preload 直接回填不调内部蒸馏，缺省与 7.2-1 逐字节一致；7.2-1 Composer 入口不动，MemoryPanel 本地托管第二实例零复制）；三语 memory.distill.* 11 键三语键集相等。
- **主代理收口**：gen_bindings+bindingNames 再生 691（bindings_novel 368 行重排噪音甄别还原，v4.313 坑再证）+bridge/core +3（SkillDistillView 局部重述 herdsman 防环先例）+mappings+mock/chat 三桩（走查样本）+spaceBindings work 510→513+App.tsx 接线（useSessionHandlers 六值解构+MemoryPanel 六 props）+建议 tab 首开自动拉候选（C 线 distillView 初值 null 使 undefined 判据失效——主代理改 ref 守卫；fetch 期显示 loading 而非「不可用」误判）+补 SkillRecordModal preload 2 例。
- **绑定面**：688→691（OfficeB +3）；work 锁 510→513；drift OK@691。
- **出口对账**：①只提示不打扰、拒绝即静默 ✅（Decide ignore→状态→复算跳过，测试钉死）；③审计链完整（哪次历史/谁确认/哪版）✅；②结晶技能调用 ≥5 次且成功率可见=**欠账如实入档**（需技能调用计数机制 statsFile 先例另刀）。
- **验收**：skilldistill 15 例+app 10 例+前端 18 例（含收口补 2 例）；tsc -b 零错（双向漂移防线过）、eslint 零告警。
- **门禁**：ci.ps1 全绿 exit 0（复跑：首跑 exit 1 无 FAIL——flaky 先例再+1）；版本四处 4.317.0；产物=exe 50576384B SHA256=e5577402da81afbf0b0135631db066494aa7e415455da67eee2dbe9b0d5e3d2d（releases/gaea-v4.317.0.exe+SUMS；桌面副本同哈希；冒烟 200 过）。
- **观察池**：滑窗/子序列挖掘 V2 等真实 journal 数据分布；journal 回放纳入 verdicts（复核结论）过滤；真机走查（同型任务×3→建议卡→结晶→新会话命中技能一条龙）挂闲置窗口；releases/README V4_RECENT 实存条数与「最近 34 版」头注不符（v4.315 前已存在，非本刀引入）。
- **未做（下刀）**：7.3-1 任务收件箱（收窄版任务空间制首刀）或 7.2-2 判据②调用计数小刀；闲庭在册清欠沿既有列车。

## 最新发布：v4.316.0（2026-09-15）「阶段七第三刀：路由学习（任务级账目+成本归因+建议制改绑，7.1-2）」

- **来源**：用户指令「继续并行使用子代理，优化迭代 gaea」。契约先行（规格书 进度计划/gaea-route-learning-7-1-2-20260915.md：三线调研结论+契约+足迹互斥表）→ A/B/C 三线并行 → 主代理收口。**坑**=B 线原实现子代理 40 分钟零落盘、两次检查点未应——中断接管主代理代写（子代理卡死判据：长时间零落盘+检查点静默；先例入 AGENTS 执行纪律候选）。
- **判据①任务级账目全键覆盖**：ChatRequest/ChatSimpleOptions 加 Feature 带外字段（json:"-"）→ ai 包 recordUsage/parseStreamEvents/prepareStreamRequest 透传 → statsKey 三维 feature|engine|model（whisper→chat 归一）→ statsFile v2→v3 兼容读 → 24 handler 文件 ~40 调用点一行式透传七键全覆盖（grep 复核零遗漏）；copilot 三入口获批扩迹；bridge 工厂 Provider 打标签（boot→gaea/显式引擎→office）。残留如实报：platform_handler 风格分析（internal/style 足迹外，落未标记桶）、gaea_diagram 两处全局引擎。
- **判据②成本归因视图**：模型中心「成本归因」tab（AttributionSection：KPI×4+功能分组表三维桶行+未标记桶最后；useAttributionState；api/engines.ts +4 函数）。
- **判据③建议制改绑**：新纯包 internal/routesuggest（score=成功率×成本因子域内归一；ScoreGapThreshold 0.25+MinSamples 20 双门槛宁少勿扰；单域至多 1 条；ID 确定性重算幂等+解析残渣防御）；App 三绑定 Suggestions/Apply/Ignore（Apply 走 SetFeatureModel 既有链路绝不自动改绑；状态原子写 <DataRoot>/route_suggestions.json）。
- **判据④本地引擎同池**：候选=全部启用引擎 llm 模型（Kind 回退 ClassifyModelKind）本地云端同池；mock 契约内置 herdsman 建议卡样本（走查锚点；真机采纳样本挂闲置窗口）。
- **绑定面**：684→688（ModelB +4，gen_bindings 覆盖表→model 门面）；work 锁 506→510（shared）；bridge/model.ts 局部重述类型+mock 四桩+新契约测试 mock-contract-route。
- **验收**：routesuggest 11 例+stats 3 例+ai 3 例+app 5 例+前端 9 例+锁数；tsc -b 零错、eslint 零告警。
- **门禁**：ci.ps1 全绿 exit 0（一次过）、drift OK@688、版本四处 4.316.0；产物=exe 50515456B SHA256=82c0e4537d28dfb53314770054672ea094e636aac2645509227d76de88113213（桌面副本同哈希；冒烟 200 过）。
- **观察池**：按任务实例切片留 7.3 联动；bridge 启发式边界（显式引擎 provider=office 域现状 6 处）；route_suggestions.json 无 UI 清单；真机走查（聊天→归因 tab 见桶行+建议卡）。
- **未做（下刀）**：7.2-2 journal 历史蒸馏或 7.3-1 任务收件箱；闲庭在册清欠沿既有列车。

## 最新发布：v4.315.0（2026-09-15）「书斋首页 v9『案头』重设计+外观模块下拉化（零功能删除）」

- **来源**：用户指令「继续并行使用子代理，优化迭代 gaea」。工作树发现上会话中断的在飞刀（两份规格书随刀入库：进度计划/gaea-desk-home-redesign-v9-202609.md+gaea-appearance-module-redesign-202609.md；实现与工具链验证上会话已完成）——本轮接管收口：独立审查子代理七点对账+CDP 走查取证+全量门禁+发版。
- **v9 诊断**（v8 遗留）：砍刊头后无排印锚点（首屏最大字号 19px）/清单海 ~80% hairline 行+等宽点 10 处/命令条 54px 主角不立/右舷五件纵排无图形锚点。
- **落地**：①masthead 身份区（w-seal 印章 44px 印泥朱 #d06055 唯一 hex 豁免+台名 clamp(30,3vw,42px)+lede+chip/pill 迁入；与闲庭月洞门身份对仗）②kicker 行退役+命令条 54→60px③w-index→w-plaques 双列案牌④w-rail-panel→w-vitals（写作升首节、ml-ring 72→80px 作用域隔离）⑤外观面板 ChoiceCards/ThemeCard 退役→Select 泛型下拉（hover 预览「预览中」徽章）+SegmentedRow 三面板⑥WhisperTracePanel hex→--md-sys-* 令牌（亮色可读）。
- **契约**：五钩子 testid 全保；三语键集 1479 零增删（1 键文案随交互更新）；闲庭逐行不动；零绑定面 drift OK@684。
- **验收**：CDP 9333 DOM 断言书斋 17 项+闲庭 5 项全绿+亮色档令牌实测（surface=#f0fdf9）+三截图留档（.tmp/v9-desk-1440.png 等）；ModuleLauncher 8/8+AppearancePanel 6/6+审查对账（className 双向孤儿=0）。
- **门禁**：ci.ps1 全绿 exit 0（第三跑——首跑 netclient flaky 定向复跑绿、二跑 internal/app TempDir 清理竞争，环境 flaky 先例再+2；三跑清走 Vite/无头 Edge 负载后全绿）、版本四处 4.315.0；产物=exe 50464768B SHA256=356b7c6e8ac9ed4ecfa4f0fb5b1f9fa6c52f61b1f1dcdcc1bd13b620e3bc2a1d（桌面副本同哈希；冒烟 200 过）。
- **文档**：home.md 补 v9 节+releases README V4_RECENT 补齐 v4.308~v4.315 九版滞后+规格书入库+AGENTS 三十迁。
- **观察池**：releases 实存 33 版 exe 与「留 5 版」拍板不符（不动文件待用户定夺）；v3-rise 入场序号两对同拍；pnpm 污染两枚已清+frontend/NUL 坏文件已删。
- **未做（下刀）**：7.1-2 路由学习（三线调研已成：R1 遥测=RecordCall 全量收口+ChatRequest 缺 feature 字段；R2 绑定=featureModelKeys 8 键+SetFeatureModel 链路+routesuggest 纯包落点+route_suggestions.json 存储；R3 前端=模型中心「成本归因」tab 落点推荐+命名避让 GaeaCostAttribution。契约待主代理定稿→后端账目/接线+建议生成+前端三线并行）。

## 最新发布：v4.314.0（2026-09-15）「书斋首页 v8 重设计『驾驶舱台面』（技能库驱动，零功能删除）」

- **来源**：用户指令「使用技能重新设计书斋首页」——ui-ux-pro-max 技能库三路检索定方向（Swiss 数据密集+星枢令牌 MASTER 权威）+ Edge 无头 CDP 截图 + analyze_image 诊断。
- **诊断**：v7 刊头大标题占 ~10% 视高/左栏长卷目录挤出首屏/账页×目录同构列表纵向堆叠/右栏五节等权无主次。
- **落地**：①刊头压缩为 kicker 行（w-mast→w-deck）②命令台升格第一主角件（玻璃抬升面收命令条/气泡流/语音状态）③账页×目录双列 7fr/5fr+旗舰横带压缩+目录单列④右栏主从：晨报置顶⑤间距校准。
- **契约不变**：testid 全保留/i18n 零键增删/功能零删除；既有测试 9/9 零改动。
- **门禁**：ci.ps1 全绿 exit 0、drift OK@684、版本四处 4.314.0；产物见 `releases/SHA256SUMS-v4.314.0.txt`。
- **观察池**：晨报有数据时右栏高度回看；写作环空态密度优化候选。
- **未做（下刀）**：7.1-2 路由学习或 7.2-2 journal 历史蒸馏。

## 最新发布：v4.313.0（2026-09-15）「阶段七第二刀：会话录制技能（演示式录制，7.2-1）」

- **来源**：阶段七 7.2-1（平台调研验证形态：Anthropic Record a Skill / OpenAI Record & Replay 均已发布；SKILL.md 成事实标准）。已有单轮沉淀+记忆聚类候选两通道，缺「会话级多轮回放→结构化草稿→审阅入库」。
- **落地**：①skill-from-session.json 模板（蒸馏纪律+JSON 草稿契约）②office_skill_record.go（buildSkillReplay 末段 60 条回放/parseSkillDraft 归一断拒/renderSkillDraftBody）③GaeaSkillDraftFromSession（本地优先路由+RetryJSON，只回不落盘）+GaeaSkillDraftSave（审阅后落盘）④saveSkillFile 共享通道重构（单轮/会话录制同一落点）⑤SkillRecordModal+Composer 工具栏入口；绑定面 682→684（work 锁 506）。
- **纪律**：LLM 只产草稿，落盘必经用户审阅。
- **测试**：Go 6 例+vitest 4 例；既有零改动。坑=heredoc 转义破坏 TS 字符串/锁数量忘同步/gen_bindings 噪音再证。
- **门禁**：ci.ps1 全绿 exit 0、drift OK@684、版本四处 4.313.0；产物见 `releases/SHA256SUMS-v4.313.0.txt`。
- **未做（下刀）**：7.1-2 路由学习或 7.2-2 journal 历史蒸馏；闲庭在册清欠沿既有列车。

## 最新发布：v4.312.0（2026-09-15）「阶段七首刀：记忆质量受控评测（三脑四域零依赖基线+经验习得题集，7.1-1）」

- **来源**：阶段七规划（docs/gaea-stage7-plan-2026-09.md，2026-09-15 拍板启动）首刀；闭合在册开放增量「记忆质量可评测」（对齐物升级为 BEAM/LongMemEval-V2）。三脑记忆域检索质量此前零量化——记忆不可信则 7.2 技能结晶复用成功率不可测。
- **落地**：①新包 internal/memoryeval（纯函数+种子语料零外部依赖）：Evaluate（recall@10 门槛 0.8 沿 T5-6 口径+precision 均值）+ParseSet/ResolveSetPath；②三域 runner 全走生产路径：story=纯 Go BM25/work=gaea bm25 Ranker/persona=whisper SQLite FTS 含中文 2-gram LIKE 降级；③经验习得题集 22 题（≥20 达标，LongMemEval-V2「习得经验」方向，查询=未来任务口吻）——7.2 评测基建预置；④种子语料 needle-in-haystack（12+40/12+28/10+20/22），题集与种子 ID 漂移由测试硬断言兜住。
- **测试**：memoryeval 4 例（解析/数学/结构硬断言/基线软告警——低于门槛只 WARN 不阻断）；既有零改动。
- **基线**：四域 recall@10 全 1.000（precision 均值 0.433/0.478/0.661/0.727）落档 docs/memory-eval-set.md；评测产出真实发现=persona 泛主语查询淹没（「用户」二字组 LIKE 命中全部事实）——生产弱点记录在档不动引擎。
- **门禁**：ci.ps1 全绿（vitest 首跑 ContextView 类 flaky 28 例、全量复跑 356 文件/3064 全过）、零绑定面 drift OK@682、版本四处 4.312.0；产物见 `releases/SHA256SUMS-v4.312.0.txt`。
- **未做（下刀）**：7.1-2 路由学习（任务级成本归因+建议制改绑）或 7.2-1 演示式录制（无依赖可先行）；闲庭在册清欠（t4-C3 余项/t6/t7/GenerationGate 收口）沿既有列车。

## 最新发布：v4.311.0（2026-09-15）「t5 收官刀：关系图谱升级（三类节点·四层边·分类筛选·详情浮层，§7.5）」

- **来源**：MuMu 蒸馏线 t5 收官刀（§6.1/§6.2/§7.5）。改造前组织节点是孤岛、职业不可见——状态机数据图谱可视化是收官题眼。
- **落地**：①纯函数模块 graphData.ts（buildGraphData）：节点三类+分组虚拟节点、边四层（组织成员 0.5/分组 0.35/主职 0.3/副职 0.2/人际 0.02——MuMu layoutWeight 映射 d3 strength，不引 dagre 不引新依赖）、人际 pair 去重、无结构边时人际 0.3 兜底、active 过滤+悬空防御；②组件：五类筛选 checkbox/点击详情浮层/职业方形节点/虚线边/选中描边；③零新依赖，调用点零改动。
- **测试**：vitest graphData +4；tsc 全绿；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面、版本四处 4.311.0；产物见 `releases/SHA256SUMS-v4.311.0.txt`。
- **t5 域全清**：差分器→回灌→组织差分→职业 UI→图谱，五刀收官。
- **未做（下刀）**：t4-C3 余项；t6/t7。

## 最新发布：v4.310.0（2026-09-15）「t5 第四刀：职业管理页（角色-职业绑定 UI，§7.6 断线补齐）」

- **来源**：MuMu 蒸馏线 t5 第四刀（§6.4/§7.6/§7.7-#9）。MuMu 职业接口后端完备前端断线；gaea 四刀全通后同样「用户到不了」——本刀补 UI 通路。
- **落地**：①NovelB +SetCharacterCareer/RemoveCharacterCareer（主职业允许替换〔手工意图 vs 差分器防 LLM 误改，两处语义有意不同〕/副职上限 2/同名幂等/阶段钳 ≥1/手工不写水位；存储=项目 characters.json 与差分器同库即设即生成可见）②CharacterPage Drawer 增「职业体系」区块（主/副管理+即时保存+上限禁用+操作后同步快照）+types v2 字段（GetCharacters 早已返回前端未消费）③契约面：bridge/mock/bindingNames 682/spaceBindings 锁 504。
- **测试**：character +1（绑定矩阵）+vitest 3 例（渲染/逐参/上限+Popconfirm 确认）。
- **维护**：AGENTS 速览整段分流——留 3 版迁 10 条入 archive，水位 61671→38151B 根治。
- **门禁**：ci.ps1 全绿、drift OK@682、版本四处 4.310.0；产物见 `releases/SHA256SUMS-v4.310.0.txt`。
- **未做（下刀）**：t5 余项唯一=关系图谱升级 §7.5；t4-C3 余项；t6/t7。

## 最新发布：v4.309.0（2026-09-15）「t5 第三刀：组织状态顶层差分接线（分析链三路输入全通）」

- **来源**：MuMu 蒸馏线 t5 第三刀（§3.7/§3.8/§7.7-#5）。差分器有组织管道但调用侧传 nil、模板无契约、载荷无字段——有管道无水源。
- **落地**：①V2 wire 契约 OrganizationStateChangeV2/OrgMemberChangeV2 名称式引用（与差分器 ID 式输入分层，对齐 character_states 先例）+AnalysisResultV2 +organization_states；②模板 output 加 organization_states JSON 形状+must 稀疏差分纪律（名称精确一致否则省略，不猜测）；③syncCharacterStatesV2 名称→ID 匹配（失败 WARN 跳过）传第 4 参；④**修首刀缺陷：覆灭短路**（destroyed=true 清零势力值+跳过同条 power 与成员变更；destroyed=false 不复活）。
- **范围裁决**：PowerValue 用 0-100 绝对值而非 MuMu 相对增量——幂等可重放，免疫 MuMu §8 坑 2（org 更新无水位守卫）。
- **测试**：analysis +2（端到端：晋升/加入/缺省忠诚/ID 匹配/不存在跳过/覆灭忽略 power/空载荷零写盘）+characterstate +1（覆灭短路纯函数钉死+destroyed=false 不复活）；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@680、版本四处 4.309.0；产物见 `releases/SHA256SUMS-v4.309.0.txt`。
- **未做（下刀）**：t5 余项（关系图谱升级 §7.5/职业管理页 §7.6）；t4-C3 余项；t6/t7。

## 最新发布：v4.308.0（2026-09-15）「t5 第二刀：状态回灌（SceneBible 实体属性 + 章节生成 character_states 槽位，降噪）」

- **来源**：MuMu 蒸馏线 t5 角色状态机域第二刀（§3.9/§7.4）。v4.307 差分器写盘后生成侧看不见——状态机一半价值在写回生成侧。MuMu 两通道格式不一致（历史债），gaea 统一多行式。
- **落地**：①SceneBible 实体属性回灌（零新增管道）：entityFromCharacter +current_state/career_main/career_sub，sceneCharFromEntity 读回+角色文件兜底，formatSceneChar 渲染（存量老角色零噪声）；②create-chapter 模板 +character_states P1 槽位，注入器 buildCharacterStatesSection 多行式（存活 emoji/心理截150/职业「剑修·3阶」/组织+忠诚度/关系+亲密度），接线 CreateChapter 主链；③降噪（照搬 MuMu :794/821）：intimacy==50 与 loyalty==50（0 视为未记录）不注入、past 关系跳过、非 active 成员不显示、关系前5/组织前3、区段预算 1500 rune、存量项目全员无 v2 字段返回空串零噪声；④types.CareerMainLabel/CareerSubLabels（无最大阶——方案 C 死解析不做已裁决；CareerName 快照优先）。
- **范围裁决**：chapter-generate.json 是 embedded 死资产（无消费方），槽位打真实链路 create-chapter，死资产不养死槽位。
- **测试**：types +1 + novelcontext +3（roundtrip/兜底/零噪声）+ app +4（降噪矩阵/模板联通）；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@680、版本四处 4.308.0（含 versioninfo.rc，sync-version.ps1 同步）；产物见 `releases/SHA256SUMS-v4.308.0.txt`。
- **未做（下刀）**：t5 余项（组织顶层差分 UI/关系图谱升级 §7.5/职业管理页 §7.6）；t4-C3 余项；t6/t7。

## 最新发布：v4.307.0（2026-09-15）「t5 首刀：角色状态机差分更新器（水位守卫+存活级联+关系差分）」

- **来源**：MuMu 蒸馏线 t5 角色状态机域首刀（§3.3~§3.6/§7.2/§7.3）。契约 v4.278 已落，差分器实现是缺口。
- **落地**：①internal/characterstate 差分更新器（三阶段严格顺序+七条不变量全实现：双水位单调/存活短路/**存活级联三件套**/职业钳制/MaxSubCareers=2/副职业水位守卫）；②亲密度算法最长匹配优先禁子串累加（修「不信任」-5 失真）+LLM DeltaHint 优先；③关系差分双向无向/新建基线 50/时间线追加；④模板扩 survival_status+relationship_changes+稀疏差分纪律；⑤syncCharacterStatesV2 委托差分器。
- **测试**：characterstate +5（级联全链/守卫/关系双向/词典/职业组织矩阵）。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@680、版本三处 4.307.0；产物见 `releases/SHA256SUMS-v4.307.0.txt`。
- **未做（下刀）**：t5 余项（回灌 SceneBible/chapter-generate 槽位/组织差分 UI/图谱升级）；t4-C3 余项；t6/t7。

## 最新发布：v4.306.0（2026-09-15）「t4-C4 分析标注层：keyword→正文偏移（坐标反投影）+ 读取绑定」

- **来源**：MuMu 蒸馏线 t4 情节分析域第三刀（§3.10/§8.4）。C1 keyword 锚点坐标从未回投正文，本刀补标注层数据源。
- **落地**：①定位四段式（精确/去标点反投影 indexMap 修 MuMu 坐标缺陷/前缀 15/未命中）；②构建规则（锚定失败丢弃不留 -1；hook Content 兜底锚点；suggestion 留 -1 仅列条目；伏笔带 tags）；③落盘 analysis/annotations/<MMM>.json+Analyze 接线；④NovelChapterAnnotations 绑定（缺档有分析按需重建）。
- **测试**：analysis +3（定位四段式/构建矩阵/空载荷）。
- **门禁**：ci.ps1 全绿、绑定面 +1 drift OK@680（锁 501→502）、版本三处 4.306.0；产物见 `releases/SHA256SUMS-v4.306.0.txt`。
- **未做（下刀）**：前端内联高亮随 t7；t4-C3 余项；t5~t7 未开工。

## 最新发布：v4.305.0（2026-09-15）「t4-C3 收尾：整章重写前端消费（对比确认 UI）+ 建议读取绑定」

- **来源**：t4-C3 收尾刀——v4.304 后端六绑定零前端消费。
- **落地**：①NovelChapterSuggestions（678→679，读 analysis-v2 建议，无分析空数组正常态）；②RewriteModal 三态（表单：建议勾选/自定义要求/重点方向/保留元素/字数，source 自动判；结果：统计+新全文+应用/放弃；应用后恢复入口）；③CreatePage rail 按钮+onApplied 刷新编辑器；④mock +7 诚实空态。
- **测试**：vitest RewriteModal 5 例；既有零改动。
- **门禁**：ci.ps1 全绿、绑定面 +1 drift OK@679（锁 500→501）、版本三处 4.305.0；产物见 `releases/SHA256SUMS-v4.305.0.txt`。
- **未做（下刀）**：partial 局部重写/场景工程/版本历史面板；t4-C4 锚点标注。

## 最新发布：v4.304.0（2026-09-15）「t4-C3 首刀：驱动式整章重写 + 版本库（应用/丢弃/恢复）」

- **来源**：MuMu 蒸馏线 t4 情节分析域第二刀（§5.2/§8.3 C3）。C1 建议产出后补重写消费链；gaea 强制增量=版本库自带恢复（MuMu 无 restore 路由）。
- **落地**：①internal/rewrite 引擎（指令四段/输出清理/diff 统计/请求归一，纯规则）；②版本库（索引无全文+按需读，契约路径）；③六绑定：重写（建议来自 analysis-v2，温度 0.7，不自动落章）/列表/读全文/应用/丢弃/恢复（原文快照写回+审计）。
- **测试**：rewrite +4 / project +1 / app +4（桩 LLM e2e 全链含恢复审计）。
- **门禁**：ci.ps1 全绿、绑定面 +6 drift OK@678（spaceBindings 锁 494→500）、版本三处 4.304.0；产物见 `releases/SHA256SUMS-v4.304.0.txt`。
- **未做（下刀）**：partial 局部重写/场景工程/前端对比 UI；t4-C4 锚点标注。

## 最新发布：v4.303.0（2026-09-15）「t3-P2 记忆召回消费：语义检索进 SceneBible 记忆区段（长程一致性全闭环）」

- **来源**：MuMu 蒸馏线 t3 长程一致性域第三刀（§3.3/§3.4/§12.1/§12.4）。分析自动回填（P1）+语义召回注入（P2）全通；检索复用 semantic.Store+本地 embedding，不引 ChromaDB。
- **落地**：①结构化 query（人物[:8]/关键事件[:6]/叙事目标/情绪/本章要求[:150]，整体[:800]）；②四段流水线（阈值 0.35 归一化余弦/兜底 3/topK 8，三常量唯一声明）；③SceneBible.Memories+「相关记忆（按相关度）」区段（预算 400）；④Ensure 增量+Stale 清死向量+kind=项目目录隔离+当前章排除；⑤CreateChapter 全链容错接线。
- **测试**：app +4 / novelcontext +2；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.303.0；产物见 `releases/SHA256SUMS-v4.303.0.txt`。
- **未做（观察池）**：t3 三刀收官；真机检索质量评估；候选 2 记忆评测开放；t4-C3/C4、t5~t7 未开工。

## 最新发布：v4.302.0（2026-09-15）「t3-P1 记忆生产者：规则表抽取 StoryMemory 按章落盘（自动回填闭环）」

- **来源**：MuMu 蒸馏线 t3 长程一致性域第二刀（§2.3/§12.1/§12.2/§12.4-4）。t4-C1 解锁 V2 载荷，本刀补「分析→结构化记忆」抽取——自动回填闭环咬合（市场差异化点，调研档 §9）。
- **落地**：①types.StoryMemory+确定性 ID `<MMM>-<type>-<ordinal>`+五类常量+is_foreshadow 三值（无 Position 死字段 D11）；②提取规则表纯函数（chapter_summary 必有三级回退固定 0.6/hook≥6/foreshadow 全部/plot_point≥0.6/character_event 0.7/conflict≥7→plot_point，门槛常量唯一声明）；③memories/MMM-<n>-memory.json 按章整文件替换写（幂等规避 MuMu D1，空=删文件）；④Analyze 接线 persistStoryMemories（容错）。
- **测试**：analysis +3（规则表/确定性 ID/回退链）+project +1（回环/空写删档）。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.302.0；产物见 `releases/SHA256SUMS-v4.302.0.txt`。
- **未做（下刀）**：t3-P2 召回消费（semantic kind=story_memory+SceneBible 记忆区段+三常量+结构化 query）——接上即全闭环；t4-C3/C4。

## 最新发布：v4.301.0（2026-09-14）「t4-C1 分析代理 V2 化：9 维结构化 + 三维评分联动 + analysis-v2.json 落盘」

- **来源**：MuMu 蒸馏线 t4 情节分析域首刀（§8.3 C1）。市场调研增量轮（调研档 §9）抬升优先级：自动设定库回填是行业痛点（Novelcrafter 手填摩擦/Sudowrite 书长极限），V2 载荷是回填数据底座。
- **落地**：①模板 V2 九维化原地升级（伏笔追踪约束原样保留，ForeshadowHit 契约零改动）；②服务端权威归一（三维钳位/overall 重算/建议联动只裁上限）；③analysis-v2.json 按章号 upsert 落盘（容错注入）；④旧 wire 零破坏派生（AnalyzeChapter 返回键零变化）；syncCharacterStates 升 V2 差分。
- **测试**：analysis +4 + project 落盘回环；既有零改动。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.301.0；产物见 `releases/SHA256SUMS-v4.301.0.txt`。
- **未做（下刀）**：t3-P1 记忆生产者（已解锁，下一刀闭环自动回填）；t4-C3 驱动式重写；t4-C4 锚点标注。

## 最新发布：v4.300.0（2026-09-14）「t3 首刀：章节前文摘要窗口预算化 + 回退链 + 反重复约束」

- **来源**：MuMu 蒸馏线 t3 长程一致性域首刀。§11.3 缺口：prevSummary 全前章 200 rune 无界拼接（200 章≈40k rune 前缀），不参与预算体系。
- **落地**：①最近 10 章窗口（buildPrevSummaryWindow 纯函数，单章 180 rune，章号升序确定性，空返回跳槽位）；②窗口声明头（部分视图语义显式化）；③摘要回退链：大纲 Summary→章节摘要文件→跳过；④create-chapter 模板反重复 must（衔接不复述+窗口外既定剧情不得矛盾）。
- **测试**：app +4（窗口/截断/回退链/乱序空态钳位）。
- **门禁**：ci.ps1 全绿、零绑定面 drift OK@672、版本三处 4.300.0；产物见 `releases/SHA256SUMS-v4.300.0.txt`。
- **未做（t3 下刀）**：P1 记忆生产者（StoryMemory+规则表+落盘，依赖 t4 产出 AnalysisResultV2）；P2 语义召回消费（semantic kind=story_memory+记忆区段+三常量）。

## 最新发布：v4.299.0（2026-09-14）「伏笔面板接调度可视面：清理入口/统计/紧急度 Badge/同步结果（t1-P4）」

- **来源**：MuMu 蒸馏线 t1 伏笔域第四刀（收官刀）。P1~P3 全在后端，前端面板仍是旧三态视图。
- **落地**：①SyncResult 上绑定面（v4.297 欠账）：GetLastForeshadowSync（面 671→672），同步计数+skippedReasons 全量（D3 可见），未分析显式报错；②GetForeshadows 附带 urgency 运行时投影（共享入口算，A5 不落库，回收/废弃不带键）；③面板四件：后端统计行（分状态+超期红 Tag，降级前端计数）/紧急度 Badge（红橙金+tooltip）/清理折叠区（三入口全确认+自动重载）/上次同步常显卡（计数+前 2 条跳过原因）；④A5 护栏=写回 stripForeshadowUrgency 剥离；⑤契约面+mock+5+load() 增量绑定降级。
- **测试**：Go +2（投影矩阵/lastSync；SyncForeshadows 按规格签名转正导出）+ vitest 面板 14/14（+6）。
- **门禁**：ci.ps1 全绿、绑定面 +1 drift OK@672（spaceBindings 锁 493→494）、版本三处 4.299.0；产物见 `releases/SHA256SUMS-v4.299.0.txt`。
- **未做（观察池）**：真机走查闭环一条龙（等闲置窗口）；SyncResult 持久化不做（会话内诊断面）。

## 最新发布：v4.298.0（2026-09-14）「伏笔生命周期清理与统计 + Lint 扩两码（t1-P3）」

- **来源**：MuMu 蒸馏线 t1 伏笔域第三刀（spec §8.1/§5.1/P3.4）。P2 打通分析自动回收后，重分析/重新生成场景缺「干净重来」入口。
- **落地**：①三个清理入口（新 `internal/app/foreshadow_cleanup_handler.go`，语义严格区分）：DeleteChapterForeshadows（删埋入∨回收本章，默认只删分析来源）/CleanChapterAnalysisForeshadows（重分析前：删「分析∧埋入本章」+回退本章回收→planted 清回收痕迹）/ClearProjectForeshadowsForReset（删全部来源；手动**重置不删除**→pending 清章节关联时间戳）；关键不变量=source_type 唯一判据、手动条目永不批量删除；②统计 GetForeshadowStats：分状态+longTermCount+overdueCount（ClassifyResolve 唯一入口，currentChapter≤0 自动），resolved 别名归一并入；③Lint 扩两码 5→7：overdue（medium）+unplanned（low，长线豁免，与 stale「该收了」vs「缺计划字段」并存）。
- **测试**：app +6（三入口 e2e/统计口径/Lint 新码矩阵/camelCase 形状锁）。
- **门禁**：ci.ps1 全绿、绑定面 +4 drift OK@671（spaceBindings 锁 489→493）、版本三处 4.298.0；产物见 `releases/SHA256SUMS-v4.298.0.txt`。
- **未做（下刀）**：t1-P4 前端（面板接清理/统计/新体检码+Badge，mock 随消费面补）；真机走查（分析→回收→清理闭环，等闲置窗口）。

## 最新发布：v4.297.0（2026-09-14）「伏笔分析驱动自动回收：三级匹配闭环 + 候选清单三层渲染（t1-P2）」

- **来源**：MuMu 蒸馏线 t1 伏笔域第二刀（spec P2 闭环核心）。此前分析侧回收只做「描述精确相等」，候选清单是整包 JSON 直塞 prompt——模型记不住该回填哪个 ID。本刀按 `docs/distill/01-foreshadow-spec.md` §3.3~§3.5/§6.1 把分析侧接上 v2 契约，埋入→回收闭环咬合，零绑定面。
- **落地**：①纯函数层（新 `internal/types/foreshadow_match.go`）：WordOverlap（rune 级 2/3-gram Jaccard 加权）+ MatchForeshadowByContent 六策略加权（标题族取最大/关键词/内容相似/引用章/分类/角色 Jaccard 累加，采纳≥0.5 同分取先=最早埋入）；②同步层重写（新 `internal/analysis/foreshadow_sync.go`，消费 `types.ForeshadowHit` 替代旧 ForeshadowAction）：回收三级匹配=精确 ID（查不到禁止回落内容匹配）→内容兜底→跳过不新建；已回收不重置、pending 拒收、hinted/partial 可回收（D15）；埋入两道防重+每章新建≤5+Importance=min(Strength/10,1) 唯一派生+计划章缺失不猜值；SyncResult 完整含 SkippedReasons 六类 Kind（修 MuMu D3 静默跳过）；③候选清单三层渲染（新 `internal/analysis/foreshadow_prompt.go`）：L1 必须回收逐条带「⚠️ 回收时 reference_stable_id 填写」紧邻指令/L2 超期≤5/L3 其他≤10+溢出注记，分层走 ClassifyResolve 唯一入口（D9）；④`prompts/analysis-chapter.json`：foreshadows 换 v2 字段+候选槽位升 P1+伏笔追踪任务指令+三条强约束+ID 追踪自检（spec §6.1）；写入口径仍 revealed，前端零消费零破坏。
- **测试**：types +4 / analysis 同步 15 例（4 迁移+11 新增，含无效引用不回落关键回归）/ 渲染 4 例（三层/排除口径/折叠上限/空态 D14 截断）。
- **门禁**：ci.ps1 全绿、drift OK@667（零绑定面）、版本三处 4.297.0；产物见 `releases/SHA256SUMS-v4.297.0.txt`。
- **未做（下刀）**：t1-P3 清理入口/Lint 扩展 overdue·unplanned/统计口径；t1-P4 前端（表格/Badge 用后端 urgency，SyncResult 上绑定面随 P4）；真机走查（分析闭环+书源线一条龙，等闲置窗口）。

## 最新发布：v4.293.0（2026-09-14）「伏笔分层注入：按计划回收章调度生成上下文（t1-P1）」

- **来源**：MuMuAINovel 蒸馏线 t1 伏笔域消费方第一刀。契约（6 态并集+计划回收章+注入控制）v4.278.0 已落库但零生成侧消费方；本刀按 `docs/distill/01-foreshadow-spec.md` §4 落分层注入（spec P1），零绑定面。
- **落地**：①纯函数层（新 `internal/types/foreshadow_urgency.go`）：UrgencyLevel（0-3 运行时不落库；D15 修正 partial/hinted 有回收压力；缺计划章不猜值不假超期）+ClassifyResolve 四值+ForeshadowLayerOf 分层唯一入口（L1 必须回收/L2 超期/L3 近期参考/L4 本章计划埋入/L5 无计划兜底〔gaea 扩展：存量无 target_resolve_in 条目保底注入防回归〕）+调度常量唯一声明；②include_in_context=false 全层排除、auto_remind=false 只抑制 L3（D8：L1/L2 硬约束不消隐）、远期不注入；③四层渲染（新 `internal/app/foreshadow_context.go`）替代旧「未回收伏笔（创作约束）」单层：模板对齐 spec §4.2（D14 省略号按长度判断）、L2≤3/L3≤5/总量≤15/整区含标题≤1600 rune、消耗顺序 L1→L2→L3→L4→L5、空层不输出标题、区段头带调度规则约束行；④`resolveTargetChapterNum`（显式/分支父节点/顺延，与 ensureChapterNode 同源复用）；⑤novelcontext 场景圣经分层优先采样（硬约束层带层标注先于参考层；分层逻辑全仓唯 types 一处=D9 消除）；⑥删 docs/mumu-distill 重复副本。
- **测试**：types +4 / app +7 / novelcontext +1（抓出超期章数负号 bug）/ 既有 context 测试迁移分层口径。
- **门禁**：ci.ps1 全绿、drift OK@660（零绑定面）、版本三处 4.293.0；产物见 `releases/SHA256SUMS-v4.293.0.txt`。
- **未做（下刀）**：t1-P2 分析驱动自动回收（MatchByContent 六策略+SyncForeshadows 三级匹配+分析 Prompt 候选清单）；t1-P3 清理/Lint 扩展；t1-P4 前端。

## v4.283.0 ~ v4.292.0 段落补登（2026-09-14；逐版全文在 CHANGELOG/releases，速览补账）

- **v4.292.0** tail×反推串联：tail 导入成功→确认→novel:goto-tab+novel:auto-reconstruct 两事件→CreatePage 任务化反推链自动开跑。
- **v4.291.0** 反推任务化：tasks.KindOutlineReconstruct+Start/TaskGet（NovelB 660），长书后台态页面关闭不丢。
- **v4.290.0** 角色名→角色库 ID 匹配：matchCharacterIDs 合并进反推 Apply（不冲手工）。
- **v4.289.0** oh-story T6 书级文风档案 style.md 注入（T1~T6 收官）；**v4.288.0** T4 篇幅路由三档骨架（bookimport/skeleton.go）。
- **v4.287.0** tail 出口 ImportNovelBookEx；**v4.286.0** oh-story T2 内核消费（patterns.json+书级白名单）。
- **v4.285.0** 搜索历史+引擎规则编辑器（NovelB 657）；**v4.284.0** 失败章重试补下（NovelB 655）；**v4.283.0** 在线搜书入口；v4.283.1 nil-Fetcher panic 根修。
- 非版本刀：真机走查（假站 /unfail 配方）、flaky 治理（ProgrammingPage 15s/ContextView 40s）、rename 根修（fileutil.RenameWithRetry）。

## 最新发布：v4.282.0（2026-09-13）「oh-story 蒸馏首刀接线：平台质量评审（rubric 引擎 + 创作间面板）+ 生成门确定性两路直显」

- **来源**：用户「把 oh-story-claudecode（MIT 网文写作插件）蒸馏给小说板块」——把已在库的三份资产（T1 评审 rubric / T2 去 AI 味门禁资产 / T3 生成门资产）**接成用户可用的一条链**（规格 `docs/gaea-novel-ohstory-distill-2026-09.md`）。
- **① 平台质量评审引擎**（新 `internal/novelreview`，零 LLM/零网络/纯函数）：15 个可机械判定维度（字数区间 / 开篇钩子 / 章尾钩子 / 预告式收尾 / 情绪节点密度 / 爽点密度 / 段落节奏 / 对话占比 / 标点节奏 / 破折号 / 格式合规 / 人称视角 / 主角存在感 / 金手指提及 / 字数表述核对），每维 `PASS|WARN|FAIL|SKIP` + S1~S4 + **原文证据（rune 区间 + 段落号）** + 改法；结论 `APPROVE|CONCERNS|REJECT` 与 rubric 同门槛；**语义维度不下结论**，缺外部数据显式 SKIP 并说明原因，**不静默给 PASS**。阈值/词表/档位是数据资产 `internal/novelreview/rubric.json`（四档：通用/番茄/起点/知乎盐言；可被 `<工作区>/.gaea/skills/novel-review/rubric.json` 整体替换；fail-closed 校验）。
- **② 接线**：NovelB +2（`NovelChapterReview`/`NovelReviewPlatforms`，648→650，drift OK）；创作间 rail「平台评审」→ 面板（档位选择 + 结论 + 逐维按 S1→S4 排序 + 证据摘录 + 黄金三问）。主角名取自角色库 `role_type=protagonist`。
- **③ T3 两路直显**：`NovelInspector`「章节体检」区新增「写前契约：N 项待补 / 齐备」与「写后硬信号：N 项 / 未命中」+ 严重度排序条目（与 AI 四路并列；缺省诚实降级）。
- **④ 口径对齐**：`.gaea/skills/novel-review/rubrics/generic.json` 18 维 → **26 维**（+8 引擎扩展，标 `measurable/engine`）+ `deterministicEngine` 段；SKILL.md 补「确定性引擎」一节（15 维映射 / 覆盖路径 / `plot_loop` 代理判定口径）——**资产=评审协议、引擎=可机械判定子集，共用同一套维度 id 与 S1~S4**。
- **测试**：Go +16（`internal/novelreview` 12 / `internal/app` 4）+vitest +10（面板 6 / 检查器 3 / CreatePage 1）+spaceBindings 锁 470→472。
- **门禁**：ci.ps1 全绿、drift OK@650、版本三处 4.282.0；产物=exe 49,597,440B SHA256=83F5BFD603DB5EC1B091E006F8CACB85B37C3283FF653CCED669F44A9628EAD9（releases 归档 + 桌面副本同哈希 + 冒烟 200）。
- **未做（下刀）**：T2 内核消费（`gates.json` 模式级门禁进 novelstyle 打分/去味）/ T4 导入结构映射与篇幅路由 / T5 子代理角色卡资产 / T6 风格档案协议；写前契约「拒绝生成」硬闸与书级白名单。

## 非版本刀（2026-09-13）：小说·生成门补确定性两路——写前大纲契约 + 写后质量体检（oh-story 蒸馏 T3）

- **来源**：oh-story-claudecode（MIT）蒸馏第三刀（规格 docs/gaea-novel-ohstory-distill-2026-09.md §2 真增量第 1 条：gaea 缺「写前结构契约闸 + 写后确定性质量闸」）。
- **落地**：①新包 `internal/novelgate`（零 LLM 纯函数）——`OutlineContractIssues`（标题/计划/要点/情感四项目齐备性，S2~S4）+ `ChapterQualityIssues`（空正文 S1 / 超长段落 S3 带行号 / 电报体 S2〔短句占比 >40% 且均句长 <10 字〕/ 平均句长偏短 S3 / 标点堆砌 S3 / 省略号滥用 S3 / 通篇句号化 S3）；Issue 与评审 rubric S1-S4 同轴、必带证据；阈值常量集中、rune 口径。②接线 `RunChapterGate` 新增确定性两路 `outlineContract` + `deterministic`，与 AI 四路并列（必出、零成本、不阻断）。
- **测试**：Go +7（契约齐备与分级/空正文 S1/正常文本零误报/电报体/超长段落/标点与省略号/句号化/证据随行）。
- **门禁**：本刀自身证据（go build/vet exit 0 + novelgate 7 例 + 生成门定向用例绿）；全量 ci.ps1 未取全绿——并行线在制品 internal/app/novel_review_handler_test.go:117 当前为红（与本刀无关）；drift OK@648；**未抬版本**（下次发版随 exe 出）。
- **未做（下刀）**：T4 导入结构映射与篇幅路由；写前契约硬闸（需先有「一键补大纲」）；书级白名单与 T2 `gates.json` 的内核消费。

## 最新发布：v4.281.0（2026-09-13）「拆书导入 P1 接线：AI 反推大纲（预览载荷 + 幂等落库 + 创作间入口）」

- **来源**：续 v4.280.0（反推引擎），把 t2 P1 接成用户可用的一条链（规格 §8.2/§8.3 首刀）。
- **后端**（新 `internal/app/novel_import_ai.go`，绑定 646→648）：`NovelOutlineReconstruct` 读当前工程章节（标题取大纲节点、正文 `ReadChapterAsStitch`，上限 200 章）→ Stage1 立项反推（模板 book-import-project，采样 3 章 ×2000 rune）→ 分批章节大纲（batchSize=5，逐批独立降级）→ 预览载荷零落库；`NovelOutlineReconstructApply` 按**章号**命中节点写 summary/scene_ideas/key_points/emotion，**幂等**、不新建不删除、不碰正文、角色名只进预览；空载荷/全不匹配**显式报错**。
- **降级诚实**：无模型或解析失败逐级回落规则兜底并把原因写进 warnings（aiUsed=false），单批失败只影响该批。
- **前端**：CreatePage rail「AI 反推大纲」→ 反推 → 确认弹窗（题材/视角/目标字数 + 前两条告警 + 「不覆盖正文、可重复执行」）→ 应用 → 刷新大纲 + rail 消息。
- **接线**：NovelB +2、bindingNames 648、spaceBindings 归 play（锁 468→470）、wailsjs 由 `wails generate module` 再生。
- **测试**：Go +2（无模型全规则兜底且逐章完整 / 幂等与作用域 + 空载荷与全不匹配报错）+vitest spaceBindings 4 例 + CreatePage 9 例。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@648、版本三处 4.281.0；产物=exe 49,455,616B SHA256=7FB09E36…2A693（releases/gaea-v4.281.0.exe + SHA256SUMS-v4.281.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **未做（下刀）**：导入向导的逐章预览 UI；长书后台化任务态（`tasks` 表 `KindBookImport`）；角色名→角色库 ID 匹配；tail 模式出口。

## 最新发布：v4.280.0（2026-09-13）「拆书导入 P1 引擎：大纲反推编排 + Prompt 模板移植 + 输入节稳定序」

- **来源**：续 v4.279.0（t2 P0），本刀落 t2 P1「AI 反推」**引擎层**（规格 `docs/distill/02-book-import.md` §8.2/§3.7/§3.8）；真模型调用编排与预览 UI 下刀接线（本刀零 LLM 依赖、全部可单测）。
- **落地**：①反推编排引擎（`internal/bookimport/reconstruct.go`）——具名契约 + 逐字段归一化（rune 截断 / characters type 二元 / **title 与 chapter_number 强制用输入值**）+ **位置对齐**批量归一化（AI 少返/乱序不错章）+ 数量不符**整批回退规则结构**的断言式防线 + 规则兜底逐字对齐规格 + 视角 11 别名归一 + target_words <1000 回落/>3e6 夹取；②`CallJSON` 负反馈重试（期望类型显式 object|array、失败原文截 200 字注入、传输错误不重试、ctx 取消零调用）+ `ExtractJSON` 容忍 markdown 包裹；③新增两个 RTCO 模板 `prompts/book-import-project.json` / `book-import-outline.json`；④**补掉规格点名的引擎缺陷**：`BuildUserPrompt` 原按 Go map 遍历输入节（同模板两次渲染字节不同、长文本无法稳定置尾、前缀缓存无法命中）→ `InputDef` 增 `order`，按 Order 升序 → key 字典序稳定排序，长文本固定置尾。
- **测试**：`internal/bookimport` +9 例（视角矩阵/字段回落与夹取/位置对齐/截断与 type 归一/兜底取首句/重试提示要素/类型不符报错/传输错误与 ctx 取消/JSON 容错与采样）+`internal/prompt` +2 例（**同模板 20 次渲染字节一致**/order 覆盖 key 序）。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646（零新绑定）、版本三处 4.280.0；产物=exe 49,405,440B SHA256=0F9BF89A…8EF4B（releases/gaea-v4.280.0.exe + SHA256SUMS-v4.280.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **未做（下刀）**：app 绑定 `NovelImportReconstruct*`（真模型编排 + 预览载荷）/ 导入向导 UI（预览→应用，含 tail 模式出口）/ 幂等落库与三件套（§8.3，`tasks` 表 `KindBookImport`）。

## 最新发布：v4.279.0（2026-09-13）「拆书导入 P0：三级分章 + 5 级编码链（MuMu 蒸馏 t2 首刀）」

- **来源**：续 v4.278.0（t1 共享契约），本刀落 t2「拆书/导入反推」**P0 阶段**（规格 `docs/distill/02-book-import.md` §8.1，纯规则零 AI、可独立验收）。
- **改前短板**：导入只有强标题正则，识别不到即退化成单章「全文」；编码链缺 utf-8-sig 与 Big5，且 GB18030 解码几乎不报错 + `utf8.Valid` 二次校验**静默放过误判**。
- **落地**：①新包 `internal/bookimport`（纯规则零 IO）——`Decode` 五级链 + UTF-16 BOM + **候选结果含替换符即判误判续探**（实测 GB18030/GBK 解 Big5 不报错只出乱码）；`Clean` 六步顺序敏感清洗；`Split` 三级切分（强标题存在不叠加弱标题、弱标题需 ≥2 候选、无标题 >5000 字走 3000~5000 窗口取最靠后句读边界）；`tail` 裁剪（5 倍数取整、>50 降 full）；四类告警。②app 接线（委托 + `NovelImportResult` 增 `encoding`/`split_strategy`/`warnings`）。③前端成功文案直显「编码 · 切分策略」+ 告警弹窗。
- **有意偏离 MuMu**：首标题前 <200 字并入首章（不丢书名/作者）；阈值一律 rune 计数（MuMu 用字节）。
- **测试**：`internal/bookimport` 17 例（Big5 不被 GBK 遮蔽回归/BOM/UTF-16/兜底不 panic/清洗顺序/三级切分/窗口上界/三类告警/tail 取整与降级）+`internal/app` 导入 5 例（BOM/GB18030/无标题窗口/弱标题/e2e 报告字段）。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646、版本三处 4.279.0；产物=exe 49,397,760B SHA256=D8947306…7659E（releases/gaea-v4.279.0.exe + SHA256SUMS-v4.279.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **未做（下刀）**：AI 反推编排（字段级归一化 + JSON 负反馈重试）/幂等落库与任务态（`tasks` 表 `KindBookImport`，禁用内存 dict）/导入向导 UI（tail 引擎已就绪，当前仅 full 可达）。

## 最新发布：v4.278.0（2026-09-13）「小说域 v2 共享契约落库 + 伏笔一致性体检（含 wire 口径纠偏）」

- **来源**：两条线汇合——①「优化完善gaea」并行池小说线续刀=伏笔一致性 Linter（文风指纹 v4.277.0 已落）；②MuMuAINovel 蒸馏实施线（规格 docs/distill/）由外部团队先落 t1 共享契约后停摆，用户确认后由 Codex 侧接管收口。
- **①伏笔一致性体检**：新 `internal/app/novel_foreshadow_lint_handler.go`（纯函数 5 类确定性检查：ordering / status-mismatch〔partial 豁免〕/ dangling / stale≥10 章〔长线豁免〕/ duplicate）+ 报告 totalChapters/items/planted/hinted/revealed/longTerm/findings；前端 ForeshadowPanel「一致性体检」（概要行 + severity Tag 轻/中/重 + 说明 + 原文条目 + 章节引用，Go 侧 omitempty/null 全防御）；绑定 645→646。
- **②t1 共享契约落库**（internal/types/）：伏笔并集 6 态 + 计划回收章 + 来源/评分/关联/注入控制 + urgency 显式 must_resolve（MuMu 静默丢弃字段的教训）+ 引用失效禁止回落内容匹配；角色三态章节水位守卫 + 方案 C 职业引用 + 派生 member_count + 亲密/delta_hint 钳制；9 维分析 + 三维分档评分 + 建议数硬联动；ChapterPlan / RewriteVersion+Index / Annotation；模板解析-覆盖-渲染-校验与上下文组装接口 + Lint 接口 + ChapterNumOf；Character/Organization/Relationship v2 可选字段（全 omitempty 零迁移）+ compat_test 兼容矩阵。
- **③P0 wire 口径纠偏**：在制品曾把「已回收」**写入口径**定为 `resolved`（`revealed` 降读取别名）——与 handoff §2-3/§4-7 相反，会让统计回收率/lint 计数/章节注入/SaveForeshadows 白名单等 7 类消费方**静默读空**；纠回 `revealed` 写入口径 + 新增 `IsResolvedStatus` 覆盖两套 wire 值并接进 stats / 伏笔 lint（检查·标签·计数）/ create_chapter 注入闸 / analysis 聚合；SaveForeshadows 白名单 3 态→并集 6 态（别名先归一化）。
- **④规格入库**：`docs/distill/`（01~07 六域 + 09-impl-handoff）入库并在 docs/README 登记；docs/mumu-distill/ 为重复副本未入库。
- **测试**：Go 全量绿（types 兼容矩阵 + 两套 wire 值新用例 / app 伏笔 Lint 5 类 / stats / analysis）+ vitest 体检 3 例 + spaceBindings 锁 467→468。
- **门禁**：ci.ps1 全绿（Go 全量 exit 0 / 前端 lint 0 error〔5 既有 warning〕/ vite build 成功 / vitest 345 文件 2994 例全绿 / E 系列守卫 OK / 仓库卫生守卫 OK）、drift OK@646、版本三处 4.278.0；产物=exe 49,379,328B SHA256=E6E17B1B…C9FE1（releases/gaea-v4.278.0.exe + SHA256SUMS-v4.278.0.txt；桌面副本同哈希；冒烟 /api/health 200 过）。
- **欠账**：六域实施 t2~t7 未开工（规格在库、契约就位）；t1 契约除伏笔体检外暂无消费方；**走查未覆盖创作面壳内真机**（mock 无工程夹具）；观察项=ChapterNumOf 双实现 / stats·narrative 无 owner / docs/mumu-distill 重复副本待删 / **novel api 直取 `window.go.app.App`（`components/novel/api/outlines.ts` 等，mock 模式 console 报 Wails runtime unavailable；生产壳不受影响）**。

## 最新发布：v4.277.0（2026-09-13）「小说·文风指纹落地面：参考档构建 + 对照体检」

- **来源**：「优化完善gaea」→ 拍板池三项均用户决策项，按既定习惯从板块并行池挑小说线既定余项（novel-revolution 刀3 内核指纹算子齐备但 app 层零消费）。
- **落地**：参考档 fingerprint.json（Manager 三方法原子写）+ NovelFingerprintBuild（全部已写章节 ComputeFingerprint 落盘，门槛 ≥3 章 ≥3000 字诚实拒绝）+ NovelFingerprintScore（有参考档 ScoreText+Delta 对照 / 无参考档通用阈值，两路可用）+ NovelFingerprintStatus + CreatePage「文风指纹」面板（空态引导/摘要格子/体检大数字+Δ 基线距离+issues 摘录）。绑定 642→645。v1 口径=作者成稿即样本，「只学作者手改」留后续刀。
- **测试**：Go+6+形状锁（camelCase 断言+delta=0 指针零值不吞）+vitest+5+spaceBindings 锁 464→467；tsc/eslint 0；drift OK@645。
- **真机走查**：夹具工程（project.Create 造 3 章约 4000 字）壳内 CDP 全流程全通（空态→构建→体检→重建覆盖）+console 零错误+清场。坑复训=海报墙须 scrollIntoView 再点/无 outline 夹具致体检钮 disabled/delete-pending 须 PowerShell 复删。
- **门禁**：ci.ps1 run1 抓 v4.276 存量缺陷当场修（ChatPage.test 语音断言未随 camelCase 发送对齐，改齐后单文件 15/15）run2 全绿/版本三处 4.277.0/桌面副本同哈希冒烟 200。产物 exe 49,352,704B SHA256=2CB99312…802CD。

## 最新发布：v4.276.0（2026-09-13）「绑定面 JSON 形状普查：造价参考/复盘笔记/会话组价价格带断链修复」

- **普查方法**：把 v4.269 走查抓到的「Go struct 缺 json 标签→Wails 线上 PascalCase→前端 camelCase 读空」bug 类升级为全仓 AST 闭包普查（绑定面签名类型递归展开×缺标签求交）：327 缺标签 struct 中绑定面可达仅 7。
- **三真断链当场修**：cost.PriceBand/BandSource（v4.191 起会话组价价格带卡全空+证据表离群判定失效）/costref.Note（复盘笔记 validUntil/refCount 读空）/costref.Indicator（造价参考视图读空）；四输入方向防御补齐。连带抓出 useChatVoice.ts 三处 PascalCase 发送点改 camelCase。
- **结构性守卫**：TestWireShapeGuard 把普查逻辑测试化进 ci（缺标签即 FAIL，负向验证过）——v4.269 修 7 结构体没防住本轮同款，守卫是根治。
- **真机走查**：CDP 桥验 GaeaCostNoteList 返回 camelCase+UI 断言笔记卡片全字段渲染+console 零错误+清场干净；坑=整段式走查脚本 UI 段卡死，拆探针分段+看门狗全链过。
- **门禁**：ci.ps1 全绿（run1 负载 flaky 两例隔离复跑绿）/ drift OK@642 / 版本三处 4.276.0；产物=exe 49,294,336B SHA256=1A12B1CA…DA869（SHA256SUMS-v4.276.0.txt，冒烟 200 过）。

## 最新发布：v4.275.0（2026-09-13）「价格带数据源调研结案：信息价『除税价』列适配」

- **调研**（docs/gaea-priceband-datasource-research-2026-09.md）：价格带数据面基础设施已完整在产（四要素字段+统计+导入链+询价飞轮），外部自动接入四路评估后建议候选4 结案（抓取脆弱+合规灰、商业 API 同「询比价不做」口径、LLM 查价不作基线）。
- **真机验证抓到实锤当场修**：信息价样本 CSV 走 ImportPreview（零落库）——「除税价（元）」不在 fieldPrice 字典→12 行全 skip；当场修字典增「除税价/含税价」+Go 回归测试，复测 12/12 全识别。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.275.0；产物见 SHA256SUMS-v4.275.0.txt。**候选4 结案待用户确认标记**。

## AGENTS.md 八迁分流（2026-09-13，非版本刀，纯文档）：恢复 14 版水位

- CI 仓库卫生守卫 WARN（.gaea/AGENTS.md 59926 B 逼近 65536 B 预算）触发既定分流：迁 v4.261.0/v4.260.0/v4.259.0/v4.257.1/v4.256.0/v4.255.0 六条入 `docs/archive/agents-version-history-2026-09.md`（八迁记录已更新），主文件恢复「最近 14 版」。水位 59926→47518 B。
- **坑**：迁移动作用 node 脚本而非 sed/正则手改——archive 是 CRLF 行尾，`indexOf('## 版本状态\n')` 匹配不上；锚点一律不带行尾、写入前先验 includes。迁移后三项完整性断言（主文件条目数 14/六条入档/迁入记录更新）全过。

## 最新发布：v4.274.0（2026-09-13）「UX 线第六刀：自定义强调色亮态自动深化」

- **根因**（对比度普查观察池第一项挖到底）：非主题令牌缺陷（lightFn glow 全是深化值），是**自定义强调色不分明暗覆盖 glow/primary**——暗色调亮的 #1dd7bf 切亮态压浅底对比仅 ~1.5，全站 accent 文字看不清。
- **修复**：`ensureLightContrast` 纯函数（lib/accent.ts，保色相压亮度至 WCAG 4.5，单测 7 例含真实案例）+ App.tsx effTokens 亮态深化暗态原样（用户存储色不变）。
- **复验**：重建 exe 复跑对比度扫描——light 告警 63→53 accent 类清零（亮态 glow 实测 #128475），dark 15 不变；剩余为禁用态/误报/设计弱化。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.274.0；产物见 SHA256SUMS-v4.274.0.txt。

## 每版度量看板补账（2026-09-13，维持轨）：v4.273.0 六行落账，滑步注记

- **发现**：slim-baseline §7「自 v4.267 起恢复逐版落账」实际只落了 v4.267 一行，v4.268~272 五版滑步（CHANGELOG v4.252/253 漏记同型教训再现）。
- **补采 v4.273.0 六行**：板块 5/5（manifests 未动）；entry chunk gz **243.6KB**（v4.267=243.0，+0.6KB≈UX 三刀纯 CSS）；冷启动 **1.39~1.70s 六轮**（bind→UI 405~424ms，同 v4.267 区间）；内存 139.1MB 工作集/152.9 私有（采样时开着用户真实工程，环境态口径注）；exe 47.0MB（49,292,288B）；**>50KB 源文件 6 个**（同 v4.267，零变化）。**全部指标零回归**。
- 表格补 v4.273.0 行+滑步注记行，自本行起恢复逐版落账。

## 对比度普查收账（2026-09-13，非版本刀，零代码）：无正文级硬伤，不改

- **方法**：CDP 13 页×明暗两态程序化扫描——枚举可见文本宿主，沿祖先合成有效背景色，算 WCAG 对比度（正文 4.5 / 大字 3.0），脚本 .tmp/walk-v4274-contrast.mjs，数据 .tmp/ui-contrast/report.json。
- **结论**：dark 15 项 / light 63 项告警，逐类甄别后**大头为脚本误报**（渐变/图片背景无法从 backgroundColor 合成——角色卡白字压图片、首页「闲庭」chip 压渐变底、accent 按钮白字，目检实际清晰可读）；真实低对比仅剩**亮态 accent 弱化文本**（徽标/链接 chip ~10 处 1.5~1.8）与 schedule 甘特行号（1.3，设计弱化）+ weixin antd purple tag（3.39）——**全部为装饰性/弱化文本，非正文，无 AA 硬伤，按「收益趋零不做」纪律不动**。
- **观察池新增（打磨候选待拍板）**：亮态 accent 弱化文本（徽标/链接类 chip）对比 1.5~1.8——若要修需动主题 lightFn 令牌（glow/accent 亮态深化），影响所有 accent 消费面，属视觉拍板项。
- **坑**：对比度自动扫描的背景合成对 background-image/渐变必然误报——告警必须逐类目检甄别后才能定刀。

## 留池清账（2026-09-13，非版本刀，零代码）：v4.270 sin_illustrate live 端到端补验通过

- **结论**：live 模型调 `sin_illustrate` 端到端全通——模型真调工具、图片真实落盘（sin/art/…「雨夜回眸」.png 1.8MB）、轨迹 artifacts 在位、`extra.illustrations['tool0']` 回写（画廊可见）、过程卡「✓思考过程·319 字·生成插图」元数据在位；截图 .tmp/walk-v4270-illustrate.png。脚本 .tmp/walk-v4270-illustrate.mjs（含空间切换修正）+ .tmp/retry-illustrate.cjs。
- **历史失败归因翻案**：前两次「模型回合未落消息」实为**上游 grok-4.6 一次性空返回**（只有 reasoning 354 字、无正文无工具）——后端兜底如实报「模型没有返回内容，请重试」（sin_handler.go:339 空正文不落库），重试一轮即成功。发送/流式/错误呈现链路全部正常，**非 gaea 缺陷**。
- **清场**：走查故事已删；产物图删除时被壳句柄锁住（Windows 删除挂起，rmSync 静默），杀壳后 PowerShell 删除成功。
- **观察池新增**：sin/art/ 存 3 张疑似历史走查孤儿图（00:36/05:22/09:34 三张「雨夜站台」走查 prompt 产物，v4.270 失败轮次超时未清场遗留），待人工确认后删除。
- **坑**：①node rmSync 对被占用文件不抛错但删除不生效（Windows delete-pending），删完必须 existsSync 复核；②v4.270 脚本超时 throw 路径没走清场（清场代码在正常尾部）——走查脚本清场应放 finally。

## 最新发布：v4.273.0（2026-09-13）「UX 线第五刀：键盘焦点环全站恢复」

- **来源**：「继续」续 UX 线。reduced-motion 普查=基建已齐（双全局兜底）不动；CDP Tab 实测抓到 v4.255 全局焦点环**全站性失效**：Tailwind v4.3.3 产物注入源码不存在的 `:focus-visible{outline:none}`，同特异性按文档序吃掉普通规则。
- **修复**（index.css 单文件）：全局环 `!important` 必胜 + antd 输入类豁免同步 !important + **ant-btn 移出豁免**（Button 聚焦零视觉，豁免=键盘用户找不到焦点）。
- **复验**：重建 exe 六页 Tab 遍历，四页零无环，余 2 站为有意豁免的 ant-input（probe 采样时序误报）；antd 按钮聚焦实拍 2px glow 环。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.273.0；产物见 SHA256SUMS-v4.273.0.txt。

## 最新发布：v4.272.0（2026-09-13）「UX 线第四刀：窄窗响应式收口」

- **来源**：「继续」续 UX 线，换镜头 CDP Emulation 900×600 复跑 13 页——布局自适应面健康，抓到两处硬伤。
- **修复**：①底部遥测条右缘截断（全站六页）：引擎 pod max-width 240 可截断（.v3-pod-text 出省略号）+tele-key/value shrink-0，挤压顺序 pod→工程名→遥测数值完整保；②sin composer 工具行按钮竖排折字：ComposerToolbar 外层 flex-wrap+按钮组 shrink-0+按钮 nowrap（办公共享组件两处受益）。
- **测试**：tsc 0/eslint 0/composer 26+layouts 5 用例绿；重建 exe 复跑 13 页窄窗全 ok（修复前六页越界），目检两处修复实证。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.272.0；产物见 SHA256SUMS-v4.272.0.txt。

## 最新发布：v4.271.0（2026-09-13）「UX 线第三刀：滚动条全站单一真源」

- **来源**：用户「继续优化迭代前端 UI」接 v4.255/v4.260 UX 线。普查=CDP 起壳 26 页次（13 页×两态）零硬伤；视觉层抓到四套滚动条并存，v1.3.0 霓虹遗产（glow 渐变 thumb）喧宾夺主。
- **落地**（纯 CSS 零 TS 零 Go）：index.css 8px 中性胶囊单一真源（module-launcher 形态升全局）；删霓虹化覆写+清 ::selection 死规则（v4.255 拍板版一直被旧规则盖着从未生效）；删 styles.css/redesign.css/module-launcher 三处重复定义。
- **走查**：重建 exe 复跑 26 页次两态——亮青粗条全变中性胶囊，novel/sin 零变化，零回归。观察池新增=weixin 通道详情留白（待拍板）/办公右栏窄窗卡片横滚兜底/novel 技能下拉裸 id（全站惯例不动）。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.271.0；产物见 SHA256SUMS-v4.271.0.txt（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.270.0（2026-09-13）「sin_illustrate 工具入册（拍板放行）」

- **来源**：拍板池清单后用户连说「继续」=按序放行；推翻 §13.2「刻意不做」，前提（产物链）本刀补齐。
- **落地**：工具集 6→7；单链路委托 SinIllustrate；Artifacts 轨迹链（trace 字段+循环收集+result 帧）+落库回写 tool0..toolN；过程卡缩略图+元数据「生成插图」。
- **测试**：Go+4+计数锁 6→7+vitest +3；零新绑定。
- **门禁**：ci.ps1 全绿 / drift OK@642 / 版本三处 4.270.0。走查=部分完成如实记录（live 模型调工具留池补验，详见 releases/v4.270.0.md）；产物=exe 49,293,312B SHA256=523EAB82…ECF5（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.269.0（2026-09-13）「造价域线上断链修复：json 标签+五算带入」

- **来源**：维持轨「五算带入与含量对照端到端」走查清账时抓到两个线上真 bug。
- **Bug1**：costproject/coststage 结构体无 json 标签→线上 PascalCase，前端读 camelCase→列表卡空名/¥NaN、五算面板同断（v4.204 起）；修复=7 结构体补 camelCase 标签（存量数据兼容）。
- **Bug2**：五算带入/手动保存/询价录入 payload 带空串时间字段→Wails time.Time 解析必炸（v4.194/4.2 起）；修复=载荷省略+类型改可选。
- **测试**：Go+2（TestWireShapeCamelCase×2 形状回归锁）+vitest 2977 全绿；零新绑定。
- **门禁**：ci.ps1 全绿（run3）/ drift OK@642 / 版本三处 4.269.0。真机走查=一次性项目端到端全通（列表卡正确渲染+带入桥验落库+溯源备注+compose 诚实回退），零前端 JS 错误；产物=exe 49,281,024B SHA256=88864D3C…C2A3（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.268.0（2026-09-13）「knip unused exports 甄别清账」

- **动机**：瘦身 P5 既定池项（§100「169 甄别」）；实跑仅 11+6+8（存量数字陈旧）。
- **甄别**：全死删（含 bridge.ts 五个死 re-export）/双出口裁未用侧/契约哨兵 @public 保留/downloadBlob 误报注明（test spy 委托）；knip.json ignore 补 bindingNames.ts+drift.ts。
- **测试**：tsc 0/eslint 0/knip unused 清零/vitest 2977 例全绿；零行为变化零 Go。
- **门禁**：ci.ps1 全绿（run2）/ drift OK@642 / 版本三处 4.268.0。真机走查=CDP 四页导航扫描渲染全过零前端错误；产物=exe 49,281,024B SHA256=10A2566A…F15D5（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.267.0（2026-09-13）「原罪工具集 v1.1：sin_export 图文导出入册」

- **动机**：v4.262 留存合约清账（13.2「下刀接完产物链路再入册」）；sin_illustrate 维持刻意不做待拍板。
- **落地**：sin_export 入册（工具集 5→6，order 末位）——委托既有 SinExportMarkdown、落 sin/exports 同名不覆盖、文件名净化 fail-closed、原子写；过程卡「图文导出」+Download 图标。零新 App 绑定。
- **测试**：Go+3+计数锁 5→6+vitest+1。
- **门禁**：ci.ps1 全绿（run1+run2）/ drift OK@642 / 版本三处 4.267.0。真机走查=临时故事两回合端到端：模型真调工具、导出件存在且含正文、过程卡渲染 ✓；走查抓到「同回合写完就导=本回合正文未落库只导头部」缺口，边界写进工具 Description 并复验过，零 JS 错误；产物=exe 49,281,024B SHA256=BD2AEE86…6D827（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.266.0（2026-09-13）「原罪底稿面板内编辑：大纲/设定集」

- **动机**：v4.263 下刀候选清账（「需冲突合并设计」）；此前设定集/大纲只读。
- **设计（§14.7）**：冲突合并=回合互斥+锁内基线比对确认，不做字段级 merge——sending 锁编辑；SinNotesSave 锁内比对编辑起始快照，不符拒（「底稿冲突：」前缀）→前端覆盖确认→force；限长与工具侧同口径。
- **落地**：新绑定 SinNotesSave（641→642，play）；大纲/设定页签编辑器（编辑/写大纲/记便签入口、逐条增删改、rune 计数）；空底稿也给入口。
- **测试**：Go+1（SaveBinding 矩阵）+vitest+8。
- **门禁**：ci.ps1 全绿（run4）/drift OK@642/版本三处 4.266.0。真机走查=临时故事端到端不打模型：编辑保存桥验落盘+冲突弹窗 force 覆盖全通，零 JS 错误；产物=exe 49,269,248B SHA256=6D40A0D9…CC3C（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.265.0（2026-09-13）「原罪画廊重新生成入口」

- **动机**：v4.263 下刀候选清账——流内插图有「重新生成」，画廊只能看。
- **落地**：零 Go 零绑定（SinIllustrate 全链路既有，`arts[cue]=path` 覆盖回写）。画廊缩略图悬浮钮+大图 Modal footer 钮；与流内同一 illustrationQueue 串行；成功后 reloadMessages 全链刷新；SinIllustration 外部换图采纳（extPathRef，不在生成中才采纳）防画廊/流内不同步。失败如实 notice 原图保留；persisted:false 也提示。
- **测试**：vitest +8（定位回调/busy/sending 禁用/空 prompt/Modal+预览跟新/collect 字段/外部换图×2/reloadMessages）。
- **门禁**：ci.ps1 全绿（vitest 342 文件 2969 例）/ 版本三处 4.265.0 / drift OK@641。真机走查=临时故事端到端（结束即删，用户故事未动）：重生成 busy 态→回写换新路径→画廊/流内同步换图，零前端错误；产物=exe 49,257,984B SHA256=868668BD…65AB（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.264.0（2026-09-12）「原罪右栏面板宽度可拖拽」

- **动机**：用户口径「右侧面板宽度应该是可以自由拉伸的」（v4.263.0 下刀候选清账）。
- **交互**：左缘拖拽手柄（骑边框 8px 命中带 + accent 显色），指针拖拽与办公 useWorkspaceLayout 同源——实时跟手/向左拖=变宽/松手持久化/pointercancel 兜底；双击复位默认 268。
- **钳制**：240~640 硬档 + 视口收敛（innerWidth-520 保故事架与故事流可读）；记忆键 gaea.sin.panelWidth 自有键续用。
- **收益**：画廊列数 auto-fill minmax(104px,1fr) 随宽自适应（默认两列不变，拖宽三/四列）。
- **测试**：vitest +5（跟手+钳制+持久化 / 收窄下限 / 双击复位 / 记忆恢复 / clamp 矩阵）；零 Go 零绑定。
- **门禁**：ci.ps1 全绿（代码树 run3+终态树 run4）/ 版本三处 4.264.0 / drift OK@641。真机走查=CDP 受信任鼠标拖拽 268→448 实时跟手（中途采样 358）+落盘+画廊列数 2→3+双击复位 268，两轮零前端错误；产物=exe 49,255,424B SHA256=4A1E96E0…0102F（桌面副本同哈希，冒烟 200 过）。

## 最新发布：v4.263.0（2026-09-12）「原罪右栏创作面板：角色 / 大纲 / 设定 / 插图（对标办公标签页面板）」

- **动机**：用户口径「继续优化原罪板块，增加类似办公的右侧面板，可以看角色、大纲、插图、设定」。改前右栏是静态说明栏，且 v4.262 工具写进 notes/<故事id>.json 的设定集/大纲**前端没有读取通道**。
- **绑定**：SinNotesGet（640→641，play 空间）只读便签文件返回 {notes, outline}；topicGuard 守卫 + 与工具侧同一把 sinNotesMu；缺失/损坏=空清单空大纲（同口径容错）。
- **面板**：SinSidePanel 四页签（角色=SinCastPanel 原样 / 大纲=只读 pre-wrap / 设定=逐条 #序号 / 插图=全故事画廊 + Modal 大图），页签计数徽标（**纯文字页签：真机走查实证 268px 下图标+两字折行**）；玩法/插图协议/边界收进头部问号气泡（内容逐字保留）。
- **开合**：顶栏 PanelRight 钮，记忆键 gaea.sin.panelOpen/panelTab（自有命名空间）；宽窗默认开窄窗默认收，替代 1180px 强制隐藏媒体查询；每回合结束自动重读底稿。
- **测试**：Go +1（绑定三态）+ vitest +14（面板 8 / collectIllustrations 2 / useSinNotes 4）。
- **真机走查**：用户真实故事四页签全通——大纲/设定空态轨道环（无底稿不编造）、插图 3 张缩略图全转 data URL、Modal 大图预览、开合与页签记忆；抓到并修掉页签折行缺陷；零前端错误。
- **门禁**：ci.ps1 全绿 ×2 / drift OK@641 / spaceBindings 分类锁 462→463 / 版本三处 4.263.0 / exe 49,253,888B SHA256=6E281D6A…0A880（桌面副本同源，冒烟 200）。
- **下刀候选**：大纲/便签面板内编辑（需设计与 AI 写入的冲突合并）、面板宽度拖拽、画廊「重新生成」入口。

## 最新发布：v4.262.0（2026-09-12）「原罪工具集 v1 + 过程卡：联网搜索 / 网页抓取 / 角色卡 / 故事便签 / 故事大纲」

- **动机**：用户问「原罪现在能用哪些工具？看不见调用工具的过程卡」。改前原罪**模型零工具**（单轮 chat 流，无 tools 字段），过程卡无从显示。
- **工具集**：`web_search` / `web_fetch`（委托办公 builtin，schema 转发零漂移）+ `sin_cast`（只读角色卡）+ `sin_notes`（便签 list/read/write/set/delete）+ `sin_outline`（大纲 read/write）；注册表 `sin_tools.go`（init 自注册/重复 panic/顺序固定）。
- **循环**：有界多轮（上限 4，末轮不带 tools 强制收尾）；新 ai 入口 `ChatStreamMessages`（与 `ChatStreamChunks` 共用装配）；首轮不支持 tools ⇒ 去工具重试 + notice；SinCancel 起手即登记、覆盖流式与工具。
- **过程卡**：新事件 `tool_dispatch`/`tool_result`/`notice`；`done.tools` 与 `extra.tools` 同形态（重开还原）；前端 `SinProcessCard`（思考折叠 + 工具行：标签/摘要/状态/用时/展开明细）。
- **数据面**：便签/大纲落 `%APPDATA%\gaea\sin\notes\<故事id>.json`（原子写 / id 白名单 / 损坏当空），不碰办公工作区与记忆。
- **未做**：`sin_illustrate` / `sin_export`（越契约实现、产物回写链路未接完，已挪 `.tmp/sin-next-knife/`，下刀接）。
- **教训**：并发子代理要显式禁改契约文件；产物类工具的跨线依赖必须在契约里点名责任人。
- **测试**：Go +14 / vitest +13（含预算拦截、收尾兜底、静默计时重置三例回归锁）；**门禁**：ci.ps1 全绿 / 版本三处 4.262.0 / 零新绑定；exe 49,242,624B SHA256=470F86EF…C6E68（桌面副本同哈希，冒烟 200）。
- **真机走查**（抓到并修 3 个单测覆盖不到的问题）：①模型一轮连搜 12 次 → 加调用预算（同名 ≤3/整轮 ≤8）；②收尾轮仍要工具 → 整单无正文且不落库 → 加收束令 + 兜底收尾轮；③前端静默超时是一次性 90s → 改每帧重置。最终=走查故事一轮出正文 1185 字（落库 1340 字）+ 9 条工具轨迹（含预算回绝），零渲染错误（走查故事已删）。

## 最新发布：v4.261.1（2026-09-12）「修复：原罪插图报『marshal image request: 图生图需要提供参考图』」

- **根因**：v4.259.0 参考槽门控只看「后端×模型能不能用」，没看「这一轮有没有取到图」——没选角色 / 只有远端剧照 / 本地文件缺失时 `refImages=0` 仍以 `mode=img2img` 发请求，Herdsman 图生图端点整单拒绝；回退条件 `refImages>0` 不成立，报错直出。
- **真机取证**：`%APPDATA%\gaea\logs\gaea-20260912.log` 20:26 两连发（图片后端 Herdsman）+ 用户 `sin/cast.json` 不存在（没选角色）。
- **修复**：新纯函数 `sinResolveRefImages`（图/名/台账归属 id）+ 解析为空即清 `mode/refMethod` 走纯文本；顺带修台账归属错位（多角色只有第二人有图时记错人）。
- **测试**：Go +3（红→绿：改前 3 子例全红；纯函数矩阵 4 面 + 端到端 3 子例）。
- **门禁**：ci.ps1 全绿 / 版本三处 4.261.1 / 零新绑定；exe 49,164,288B SHA256=CCE99E91…D5CAB（桌面副本同哈希，冒烟 200）。
- **真机走查**：v4.261.1 exe 打开原罪故事 → 两个插图位 20s 内自动出图（产物落 `%APPDATA%\gaea\sin\art\`、回写 extra.illustrations），日志零失败/零异常；该故事没选角色=与用户同一分支。
- **文档**：docs/gaea-sin-board-design-2026-09.md §12.5、releases/v4.261.1.md。

## 最新发布：v4.259.0（2026-09-12）「原罪插图：角色一致性锚定（文本锚点 + 图像参考槽）」

- **链路**：角色库 → 故事角色（v4.257）→ 插图人物（本刀）：文本锚点对所有后端生效；图像参考槽走绘梦 T2（ComfyUI krea2*/z-image-turbo · Herdsman 图生图，denoise 0.65）。
- **防互串**：提示词点名优先；单角色兜底；多角色未点名不锚定。
- **诚实**：能力门控（不支持就跳过并给原因，不硬塞）+ 参考槽失败自动退回纯文本重试（标 ref_fallback）+ caption 徽标可见。
- **未做**：ipadapter/pulid（后端未实现）。
- **门禁**：Go 全绿 / drift OK@640 / tsc 0 / eslint 0 error / vite build / vitest 全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.259.0；走查徽标实测（零异常）。
- **文档**：docs/gaea-sin-board-design-2026-09.md §12、releases/v4.259.0.md。

## 最新发布：v4.258.0（2026-09-12）「原罪插图：生成进度动画 + 串行队列 + 真停止」

- **进度**：抽共享进度组件 GenerationProgress（自绘梦 GenerationBar 抽出，类名/阶段名表单点维护）+ 轮询 hook useComfyTaskProgress；ComfyUI 走确定态（百分比+当前节点+已用时），其余后端走不定态光带 + 本地秒表（不编造百分比）。
- **串行队列**：同板块一次一张（图像后端单任务），排队显示「前面还有 N 张」，失败不阻断。
- **真停止**：新绑定 SinCancel（绑定 639→640）中止底层流并保留已生成部分落库（extra.cancelled）；插图「取消」接 CancelImageGeneration。
- **顺带修真缺陷**：sending 因 deadRef+StrictMode 卡死（输入框锁死）、Composer「停止」是空实现。
- **门禁**：Go 全绿 / drift OK@640 / tsc 0 / eslint 0 error / vite build / vitest 全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.258.0；走查：两处进度条 + 取消一张另一张继续 + 完成后停止按钮消失（零错误零异常）。
- **文档**：docs/gaea-sin-board-design-2026-09.md §11、releases/v4.258.0.md。

## 最新发布：v4.257.1（2026-09-12）「修复：真机插图报 AttachmentDataURL is not a function」

- **根因**：S2-3 兼容代理（window.go.app.App）只认字面名、不做 gaeaToGaea 短名映射 → 短名 `AttachmentDataURL` 在真机（Go 方法名 GaeaAttachmentDataURL）查找落空。
- **修复**：兼容代理补映射回退（字面名优先→映射名→仍未命中才 undefined）+ `api/image.readFileAsDataURL` 改走规范 bridge 路径 + HTTP 门面表加 SinB。
- **波及面**：同根因的 4 个方法（AttachmentDataURL/SavePastedImage/RecognizeImage/OCRText）一并修好。
- **测试/门禁**：vitest +3（真机形态假门面回归）；tsc 0 / eslint 0 error / vitest 全绿 / drift OK@639 / 版本三处 4.257.1。
## 最新发布：v4.257.0（2026-09-12）「原罪三刀：与办公硬隔离 + 模型中心独立卡片 + 角色库接入」

- **①硬隔离**：插图产物落 `<用户配置目录>/gaea/sin/art`（绘梦生成链 provenance 参数化，绘梦行为不变）→ 不写办公工作区、不依赖办公 ImageSaveDir；角色配置落 sin/cast.json；无记忆面接入；模型路由 routeModel("sin") 独立；工位关键词检索改为**整个 `.gaea/play` 判噪音**（改前只跳 play/exports，画室台账等会漏进办公检索）；顺带修「原罪插图被登记两次」缺陷。
- **②模型中心卡片**：FEATURES 增「原罪」+ 图标/说明；走查确认在「功能绑定」区在位（未绑定→跟随全局）。
- **③角色库接入**：右栏角色卡 + 多选弹层 → SinCastGet/SinCastSet（悬空过滤/去重保序/封顶 8）→ SinStream 注入角色块（外观/性格/背景/动机/口吻/说话样例 + 不得改设定）；只引用不改库。
- **未做**：角色参考图当图像参考槽（人物一致性锚定）留拍板。
- **门禁**：Go 全绿 / drift OK@639 / tsc 0 / eslint 0 error / vite build 成功 / vitest 全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.257.0；走查零错误零异常。
- **文档**：docs/gaea-sin-board-design-2026-09.md §8/§9、releases/v4.257.0.md。

## 最新发布：v4.256.0（2026-09-12）「闲庭新板块『原罪』：对话式图文混杂故事创作（复用办公组件）」

- **定位**：用户点单的闲庭新板块（原话见 CHANGELOG）；闲庭第七个一级板块，形态=「说一句→写一段→你接着指方向」的对话式故事创作 + 就地出图。
- **落地**：前端复用办公组件（Composer/Markdown/ToolbarButton/EmptyState/三件套样式，零新建聊天组件、零改办公件）；后端新门面 SinB（9 方法）=统一聊天存储 mode=sin（与聊天板块双向隔离）+功能级路由 sin（模型中心可单挂模型）+绘梦图像链内联插图+图文 Markdown 导出。
- **协议**：模型另起一行输出 `@@插图|画面描述@@`，前端按出现次序（0 起）就地生成并嵌进正文流；正文原样落库，重开按 `extra.illustrations` 渲染不重生成。
- **边界**：成人内容口径沿用 docs/ADULT_MODE.md（零新立），四条硬边界写进提示词并有测试锁定。
- **门禁**：Go 全绿 / drift OK@637 / tsc 0 / eslint 0 error / vite build 成功 / vitest 334 文件全绿 / E 系列+仓库卫生守卫绿 / 版本三处 4.256.0。
- **文档**：docs/gaea-sin-board-design-2026-09.md（设计基线+复用清单+已知边界）、releases/v4.256.0.md。

## 最新发布：v4.218.0（2026-09-11）「阶段六 6.6 首刀：跨域 EVM——进度×造价挣值」

- **定位**：阶段六 6.6 首刀（施工债级可穿插；gaea 独有跨域）。**6.6 判据「三数+两指数出且口径注全」满足**。
- **落地**：computeEvm TS 纯函数（PV 基线分摊/EV 完成率×预算/AC 手录+SPI/CPI/SV/CV/BAC/evTop，口径注直出）+ 进度页「挣值分析」视图七档（AC 录入面）。
- **门禁**：vitest +6 全绿/tsc -b 0/绑定 613 零变更/版本三处 4.218.0。
- **阶段六状态**：6.1/6.2/6.4/6.5/6.6 判据全满足；余 6.3 多文件 DAG（委托式深水区，动手前先立设计文档）。

## 最新发布：v4.217.0（2026-09-11）「阶段六 6.5 首刀：进度·蒙特卡洛工期带」

- **定位**：阶段六 6.5 首刀（进度两翼之二，nPlan 对位）。**6.5 判据「三档+稳定度本地可跑」满足**。
- **落地**：monteCarlo TS 纯函数（种子化 RNG+三角分布采样+逐次重算 CPM；P25/中位/P75 三档+直方图+关键路径稳定度/换线风险）+ SchedulePage「工期模拟」视图（不确定度滑杆）。
- **门禁**：vitest +8 全绿/tsc -b 0/绑定 613 零变更/版本三处 4.217.0。
- **下一步**：6.3 多文件 DAG（委托式深水区）；6.6 跨域 EVM（施工债级可穿插）。

## 最新发布：v4.216.0（2026-09-11）「阶段六 6.4 首刀：进度·DCMA 14 点计划质量体检」

- **定位**：阶段六 6.4 首刀（进度两翼之一，对标 Acumen Fuse）。**6.4 判据「十四点纯函数全测」满足**。
- **落地**：dcma.ts TS 纯函数（14 点逐项+阈值+超标点名+n/a 诚实口径+CPM 断裂 score=null）+ SchedulePage「质量体检」视图（DcmaView 纯展示）。
- **门禁**：vitest +11 全绿/tsc -b 0/绑定 613 零变更/版本三处 4.216.0。
- **下一步**：6.5 蒙特卡洛工期带（同工作日口径哲学）；6.3 多文件 DAG；6.6 EVM 可穿插。

## 最新发布：v4.215.0（2026-09-11）「阶段六 6.2 首刀：记忆驱动项目本体——项目本体注入」

- **定位**：阶段六 6.2 首刀（依赖 5.1 图谱底座）。**6.2 判据满足**（决策引用+可关闭）。
- **落地**：BuildProjectBrief 纯函数（固化优先+[MEM:] 引用键+决策来源归因+600 rune 预算）入 work 空间装配前缀；config project_brief 键+GaeaMemoryBrief/GaeaSetMemoryBrief 绑定（611→613）+MemoryPanel「项目本体」开关。
- **门禁**：Go 全量 0 FAIL（+2）/drift PASS@613/vitest MemoryPanel 2/2/版本三处 4.215.0。
- **下一步**：6.3 办公·多文件 DAG（委托式深水区）；6.4/6.5 进度两翼；6.6 EVM 施工债级可穿插。

## 最新发布：v4.214.0（2026-09-11）「阶段六 6.1 首刀：可审计默认化——文件预览默认版本条」

- **定位**：阶段旗交阶段六（书斋纵深·办公主角），6.1 首刀；纯前端、零新绑定、零 Go 改动。**6.1 判据满足**。
- **落地**：FileVersionStrip 默认可见版本条入文件预览（版本数/最近改动/状态/可恢复；展开=该文件 VersionTimeline 逐版本预览/恢复；pptx 页码归因来自证据卡 p3 摘要）。
- **门禁**：vitest +4（FileVersionStrip）/FilePreview 27 例全绿/tsc -b 0/绑定 611 零变更/版本三处 4.214.0。
- **下一步**：6.2 办公·记忆驱动项目本体（依赖 5.1 图谱已就位）或 6.3 多文件 DAG；6.6 EVM 施工债级可穿插。

## 最新发布：v4.213.0（2026-09-11）「5.3 首刀：记忆生命周期三态」

- **定位**：阶段五 5.3 首刀；「90 天一刀切」替代。**5.3 出口判据全满足**（蒸馏合并/预取开关此前已落地）。
- **落地**：固化（SchemaV19 pinned：豁免清理+排序加权+Pin/Unpin 事件留痕）+ 衰减（DecayScore 纯函数+LifecycleOf 分类）+ 三态可查（GaeaMemoryLifecycle/GaeaMemoryPin 绑定 609→611；OfficeMemoryLibrary 统计行+FactCard 固化锁）。
- **门禁**：Go 全量 0 FAIL（+5 用例）/ drift PASS@611 / vitest 6/6（OfficeMemoryLibrary）/ 版本三处 4.213.0。
- **阶段五状态**：5.1/5.2 证据链/5.3 判据全满足；余=5.2 主动调度与 5.1/5.2 UI 深化按需另刀，阶段五可评估出口。

## 最新发布：v4.212.0（2026-09-11）「5.2 首刀：上下文编译 dump/diff + 前缀稳定证明」

- **定位**：阶段五 5.2 首刀；纯 Go、零绑定、前端零改动。证据链出口判据落地。
- **落地**：agent 消息级编译摘要（逐消息 SHA256 链+长度前缀防拼接歧义）+ DiffMessages 相邻请求前缀稳定判定；判定随 RequestHeader 落会话日志（dump/diff/回放一体）；contextview 趋势柱携带 Prefix 与 CacheHitTokens 配对。
- **门禁**：Go 全量 0 FAIL（+4 用例）/ 版本三处 4.212.0 / vitest 以 v4.210 基线为准。
- **下一步**：5.2 深化（意图分类×预算调度的主动调度）或 5.3 记忆生命周期三态（touch 事件已留痕），按规划推进。

## 最新发布：v4.211.0（2026-09-11）「5.1 收口：回复发出前剥离悬空引用」

- **定位**：v4.210.0 的 5.1 余项收口刀；纯 Go、零绑定、前端零改动。**5.1 出口判据全满足**。
- **落地**：agent stream() 收尾定稿闸（FinalizeText 钩子：Message 事件前改写+同进 session/摘要）+ boot 记忆开关注入剥离闭包（StripDanglingCitations 纯函数，不 Touch；剥离键落 dangling cite 事件）。
- **门禁**：Go 全量 0 FAIL（+5 用例）/ 版本三处 4.211.0 / vitest 以 v4.210 基线为准（纯 Go 刀先例）。
- **下一步**：阶段五 5.2 上下文编译（意图分类→预算调度→前缀稳定排序；embedding 向量列启用）。

## 最新发布：v4.210.0（2026-09-11）「记忆语义图谱：事件日志投影出图」

- **定位**：阶段五「记忆OS+上下文编译」首刀（规划大纲 §2 主轴旗，底座#1）。设计基线=docs/gaea-memory-graph-51-design-2026-09.md。
- **落地**：memory_events 事件日志（SchemaV18 追加式，五条写路径挂钩，Touch 逐条留痕）→ ProjectEvents 纯函数投影（实体/事件/来源三向边，[MEM:name] 图寻址，悬空拒写留 Dangling）→ mem_graph_* 物化（embedding 占位列；读前水位对账懒重建=删库可重建）。
- **接线**：绑定 608→609（GaeaMemorySemanticGraph）；MemoryHub 图谱页「关联图/事件图谱」切换；回合收尾引用解析落 cite 事件（命中+悬空）。
- **门禁**：Go +9 用例 / vitest +1 / tsc 0 / drift@609 / 版本三处 4.210.0。
- **欠账**：5.1 余项=真·发送前悬空引用剥离（流式层另刀）；UI 遍历/手动重建按钮按需另刀（Go API GraphNeighbors/RebuildGraph 已备）。

## 最新发布：v4.209.0（2026-09-10）「组价含量对照：拆解含量 vs 同类条目分位带」

- **造价刀路池 §6 收官，survey 六项缺口全清**——docs/gaea-cost-domain-survey-2026-09.md 已逐项回写收口状态（§3→v4.194 / §4→v4.195 / §5→v4.196 / §6→本刀），该文档不再充当欠账池。
- **缺口**：AI 拆解人材机含量只有金额自洽校验（v4.158），含量本身离谱（如水泥 420kg 对同类 300kg）无人提示。
- **落地**：`cost.CheckContentBaseline` 纯函数——拆解组件 vs 相似条目池同键含量 P25/P75 分位带（R-7 同口径），带外 warn 带内静默；匹配键=标题归一（全角→半角/去空白/小写）+单位归一精确相等（含双方都空），**一方缺失一律不比（宁缺勿误）**；同值退化带 ±5% 容差；同键样本 <3 不比对。接进 `GaeaCostCompose` `composeChecks`（金额自洽+含量对照两层；对照池=相似条目组件明细，与价格带同池；池空该层静默），Checks 随视图展示+Apply 留痕自动承载，**零新结构零新绑定前端零改动**。边界：外部「行业含量区间」数据源明确不引入——基线源=用户自己的库（私人记忆哲学）。
- **门禁**：Go 全量 exit 0（116 包 0 FAIL，+5 用例：contentband 四例+composeChecks 接线一例）/ drift PASS@608（零绑定）/ 纯 Go 刀 vitest 以 v4.208 全绿基线为准 / 版本三处 sync 4.209.0。
- **产物**：build.bat（1m02.7s）→ `build\bin\gaea.exe` **48,580,096 B** + 冒烟 `/api/health` 200 + Desktop\gaea.exe 同哈希；SHA256=**637c6d835d6355b62edfffbed68602bda35fec1e78a7a8d1f7b948607e7f9537**（releases/SHA256SUMS-v4.209.0.txt）；未打 tag（随 v4.200+ 现状）。
- **欠账**：组价含量对照端到端待真机（观察池）；造价域新缺口另立调研。

## 品牌资产补齐（2026-09-10，非版本刀）

- **缺口**：2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核）；**两个二进制件没跟上**——`build/appicon.png`（1024²）与 `build/windows/icon.ico` 仍停在 **2026-08-05** 的旧视觉（翡翠球体 + 破土嫩芽 + 星芒，深蓝底板 `#0F172A`）。后果：exe 内嵌图标、任务栏、资源管理器、桌面快捷方式**全显示旧 logo**，只有窗口内 UI 是新 logo——「换了一半」。
- **落地**：① 无头 Edge 渲染 `build/appicon.svg` → 1024×1024 RGBA PNG（关键参数 `--default-background-color=00000000` 保透明，圆角外 alpha=0 与旧资产同规范）；② Pillow 由同一母图生成 **7 档 ICO**（16/24/32/48/64/128/256——旧资产只有 6 档、缺 24，本次补齐 Windows 标准全集）；③ 重建产物 `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**，`-s` 跳过前端编译用现成 dist）。
- **验证（三条独立证据）**：① **几何精度**——核心圆 r=50@viewBox512 实测 1024 图中直径 **200px**、圆心 (539.5, 511.5) 对理论 (540, 512)；环 `stroke-width=48` 实测线宽 **96px**；圆角 `rx=104` 对角线首个不透明像素 d=**61** 对理论 60.9——渲染零缩放零偏移。② **exe 内嵌图标提取**（`ExtractAssociatedIcon`）——底板 `#0B1210`、米色核 (243,230,198)、翡翠环 (103,235,195)=新「地核 G」；旧版是 `#0F172A` + 绿球。③ **dist bundle**——新 logo 独有标记 `gaea-logo-ring`×3 / `gaea-logo-light-ring`×3 / `gaea-logo-core`×2 / `0B1210`×2 在册，旧独占色 `#6ee7b7`/`#047857` **零命中**；另按旧 SHA256 全仓比对，**零旧 logo 二进制残留**。
- **门禁**：wails build exit 0 / 冒烟 `/api/health` **200** `{"status":"ok"}` / 纯资产刀——**零前端源码改动、零 Go 源码改动、绑定面不变**。
- **产物**：`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希）。中间态 903F9759…C36EC068 系「只换 appicon.png + 主图 ICO」那一版，小尺寸优化后重建覆盖；原始 59C2B518BDB6165CCBAC07BBC565759A2D7571F254E287E27AA00B2BCCD97A16 系 v4.208.0 归档值（releases 档案自洽，未动）。
- **坑（四条，按价值排序）**：
  1. **无头 Edge 截图默认白底**——不传 `--default-background-color=00000000` 时 SVG 圆角外会合成为**不透明白**（CSS 里写 `background:transparent` 无效，截图不继承页面背景），透明通道静默丢失；旧资产角像素 alpha=0 可作规范基线对照。
  2. **颜色特征校验必须先验证该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变（`stop-opacity=0.12`）与**旧** logo 主色，拿它判新旧会把新资产误报成旧的；判据只能用**独有**标记（SVG id 名 / 独占色 `#6ee7b7`、`#047857`）。
  3. **`pwsh` 不在 PATH**——手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（非 app 故障，易误判为冒烟失败）；`build.bat` 内已有 `where pwsh` fallback 到 Windows PowerShell，手工执行要自己判。
  4. **运行中的 exe 可被 `Copy-Item` 直接覆盖**（Windows 允许替换映像路径，进程持旧映射），桌面副本无需先关 app；但**已加载实例不会换图标**，需重启才是新 logo。
- **小尺寸可辨识优化（同日追加）**：原方案对大图统一降采样，**16×16 环宽仅 1.5px、核 3.1px**，G 字形糊成一坨深色块。新增派生资产 **`build/appicon-small.svg`**（环 48→64、核 r50→62、去 halo），ICO 按尺寸分流源图：**16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**，字重过渡平滑（不像 24→32 那样跳）。**坑：Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 每帧 ICONDIRENTRY 16B + PNG 帧；**256 档的宽/高字段写 0**，写 256 会溢出单字节）。对比目检 `.tmp/logo-render/small-size-compare.png`（16/24/32 × 浅底/深底 × 原版/变体）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。

## 工作空间与文档整理（2026-09-10，非版本刀）

- **目录**：根目录 48 份散落的 `SHA256SUMS-v4.*.txt` 归位 `releases/`（该目录现 218 份，归档位置统一）；`nul`（Windows 重定向残留）删除；`.tmp` 清掉可再生的构建/测试缓存与目检 profile（`gocache*`/`gotmp*`/`node-compile-cache`/`edge-home`/`edge-v7`/`smoke-gaea.exe` 等），**释放 ~2.98 GB**（3354 MB → 395 MB）。
- **文档索引**：`docs/README.md` 复核重建——45 份顶层文档 + 2 子目录**全覆盖**（补回 7 份漏登记件：角色域盘点/30 分钟上手/office 枢纽审计/瘦身基线+刀2+P2 三份证据/壳内残留审计），状态行按 git log 对齐，并立「未登记=孤儿」维护规则。
- **归档索引**：`docs/archive/README.md` 顶层 53 项**逐条登记**（原表格只用通配描述，实际漏登 14 项），磁带类大件标明覆盖范围与缺口。
- **AGENTS.md 瘦身（本刀最大发现）**：该文件曾 **104.9 KB**，超工作区指令预算 **65536 B**——尾部「执行纪律 / 交互纪律（不许用等待换时间）/ 工装 / 长期规划 / 项目定位 / 技术栈 / 发布流程 / 沙箱备忘 / 本地 TTS」**整段对后续会话不可见**。二次分流（v4.173–v4.146 逐版迁入 `docs/archive/agents-version-history-2026-09.md`）后 **45.7 KB**，全文可见；归档件同步标明覆盖范围（v4.173–v4.146 + v4.49 及更早）与缺口（v4.50–v4.145 只在 CHANGELOG/releases）。
- **引用完整性**：修掉 12 处「文档已移入 `docs/archive/` 但引用未跟」的悬空路径（含 AGENTS 三处：执行审计 / VoxCPM2 / CosyVoice 记录）。
- **防复发门禁**：新增 `scripts/check-docs.mjs`（① docs 顶层孤儿登记 ② `docs/` 悬空引用（带外部路径白名单）③ AGENTS.md 字节预算水位）接入 `scripts/ci.ps1`——同类漂移下次直接红。
- **产物保留策略落地（2026-09-10 用户拍板：保留最近 5 个版本）**：`releases/` 下 51 个 exe / 1996.6 MB → 保留 **v4.186.0 / v4.98.0 / v4.97.0 / v4.96.0 / v4.95.0**（226.6 MB），删掉其余 46 个 / **释放 1770 MB**。每个被删版本的**身份仍在** `releases/SHA256SUMS-vX.Y.Z.txt`（218 份校验和覆盖到 v4.99.0 及更早）；策略已写入 `releases/README.md`「本地产物保留策略」与 `.gaea/AGENTS.md` 发布流程第 4 步（下次发版删第 6 新的一版即可）。
- **未动（体量大且属用户数据，等拍板）**：`clones/` 参考仓（mpp2013/projectlibre/unsloth，**338.9 MB**——含隐藏 .git；首轮统计未计隐藏文件故报 212 MB）、`backups/` 120.4 MB、`whisper_data/` 运行时数据 49.5 MB、`.tmp/codex` 67 MB。

## 最新发布：v4.208.0（2026-09-10）「首页 v7 重设计：书斋「文书台」/ 闲庭「游园画廊」」

- **缺口**：v6 首页仍被判廉价——根因是「等大圆角卡片 + 描边」一种形态重复到底，层级只靠 13/14px 微差。
- **落地**：结构收敛为三形态（仪表条 / 账页 / 海报墙）+ 四档排印 + 三级色调面 + 版式记号（序号水印 / 月洞门细环），删极光斑；信息零删除，契约 testid 与 `.garden-banner` 保留。修复重写时遗失的 `.ml-avatar-ai`/`.ml-avatar-user`（气泡左右曾退化为同底色）、清掉空挂 `.p-foot-wide`、删除误入的 pnpm 锁/工作区桩文件。
- **附带门禁可信化**：CI 由「无条件重试一次」改为「只在册 flaky 隔离复跑」（`known-flaky.txt` 当前为空 + 分类器自测）、`maxWorkers` 跟物理核（原按逻辑核开 ~31 worker 导致重组件超时假红）、`testTimeout` 30s、RTL `asyncUtilTimeout` 5s。
- **门禁**：tsc/eslint 0、e-check OK、drift PASS@608（零绑定）、Go 全量、vitest 全量（324 文件 2799 例）、版本三处 sync 4.208.0。
- **产物**：主树 `wails build -ldflags "-s -w" -trimpath`（1m08.5s）→ `build\bin\gaea.exe` 48,570,880 B / 46.32 MB；冒烟 `/api/health` 200 通过；Desktop/build\bin 双副本落位；SHA256=59c2b518bdb6165ccbac07bbc565759a2d7571f254e287e27aa00b2bccd97a16（`releases/SHA256SUMS-v4.208.0.txt`）。构建后工作树仍干净（`npm install` 未改写 lockfile）。
- **目检**：Vite dev（`?mock=1`）+ 无头 Edge（CDP 9333），两空间 × 1440/1440 高/1100/880 共 10 张；无横向溢出，880 档书斋单列、闲庭海报墙 2 列塌缩正常。
- **欠账**：海报墙首张（小说）大样中段留白偏多（待用户观感定夺）；浅色主题两首页未目检；壳内真机走查池不变。

## v4.207.0 / v4.206.0 / v4.205.0（2026-09-10）主题令牌收口 + 小说书房工坊接线

- **v4.207**：深色主题硬编码深灰——RelationGraph 缩放/提示/图例写死 `#555/#ddd`、进度里程碑菱形/标签写死深灰，改走 `--color-text` / on-surface。
- **v4.206**：33 处 `bg-accent text-white` → `text-accent-fg`（= on-primary：暗色深字、亮色白字），App 补注入 `--color-on-primary` / `--color-surface`。
- **v4.205**：小说书房工坊接线——书架画廊头 + 正在编辑横条 + 书脊/角标；阅读页属性检查器由 `display:none` 改为默认可折叠（章节体检入口回来）。
- 三刀均纯前端、绑定 608 零变更、Go 零改动。

## 最新发布：v4.204.0（2026-09-10）「组价多方案对照：P25/中位/P75 当场切」

- **缺口**：survey §2 多方案对照——三档数字在，推荐价钉死中位数。
- **落地**：ComposeModal 点选 P25/中位/P75，应用价随档；cost_compose `mode` + 三档对照行。不拆逐步盖章。608 零变更。
- **门禁**：ComposeModal 11/11（+1）、Go TestCostCompose 绿、tsc/eslint 0、版本三处 4.204.0。
- **欠账**：流式打字机/费率政策对照不做；含量基线等样本。

## 最新发布：v4.203.0（2026-09-10）「空间策略其余功能域键：总闸当场切」

- **缺口**：v4.190 七键写路径已通，UI 只放 gaea 主控；其余功能域覆写仅非空 chip，空键看不见写不了。
- **落地**：纯前端 608 零变更。每张空间卡「其余功能域」六键常显可编（对话/轻语/小说/办公文档/角色库/例行），闭环与 gaea 同款；空串清除。
- **门禁**：vitest StrategySection 8/8（+2）、tsc/eslint 0、版本三处 sync 4.203.0。
- **欠账**：功能域×空间交叉矩阵按需另刀；壳内真机走查池不变。

## 最新发布：v4.182.0（2026-09-09）「双空间首页分版：书斋/闲庭定名 + 切换器迁首页顶栏」

- **定名**：书斋（work，Study）/闲庭（play，Lounge）三语（shell.space.*/shell.search.scope.*+space.ts 兜底）。
- **切换器迁移**：首页顶栏 SpaceSwitch（胶囊组 aria-pressed+模型徽标，直连 switchSpace）；rail 顶部改竖排空间指示徽标（非交互），分域导航不变。
- **两首页分版**：书斋「文书台」三栏效率台（Hero 命令条+最近文档流水主角面板+能力矩阵+右舷状态栏）；闲庭「游园画廊」全幅画廊（旗舰横幅+大卡两列+园底信息带：进度/继续话题 chips/记忆/遥测细条）——信息零删除形态分化（ui-ux-pro-max 定 dial）。
- **门禁**：vitest 2745（+3 ModuleLauncher.test；flaky 族单独复跑绿）、tsc/eslint 0、e-check OK、locale 0 死键、零绑定 drift PASS@602、零 Go 改动、build strip+冒烟 200、SHA256=F8B238863236F53803AB07822A1136366EEC92E97403BC5B592C1C04214E1337。
- **欠账**：壳内真机走查两首页观感/切换手感（真机池）；改名再调=纯 locale 三文件。

## 最新发布：v4.181.0（2026-09-09）「上下文/轨迹按会话读取 + GLM Coding Plan 429 分诊」

- **根因**：Agent 网络不按当前会话=GaeaTrajectory/GaeaAgentNetwork 无参绑定恒读内核 ga.ctrl 单例会话，UI 切历史会话后不跟（「UI 会话≠内核会话」根因家族，其余无参绑定待审计）。
- **修复**：两绑定变参 sessionPath（显式优先/缺省回落内核兼容）+resolveGaeaSessionPath；bridge 契约数组形态+mock+agentNetworkStore 路径声明（跨会话清快照宁空勿错）+六消费方接线；BrowserPanel 保持内核兜底。
- **附带 GLM 429 分诊**：Coding Plan 资源包挂 coding 端点计费域，标准端点 429=端点未切（非 Key 坏）；glmPing 429 时提示切换「编码套餐」端点。
- **门禁**：Go 全量 0 FAIL（+1 显式路径用例）、vitest 2742（+3；flaky 族单独复跑绿）、tsc/eslint 0、drift PASS@602（零绑定名）、build strip+冒烟 200、SHA256=D3BD8386862D5F74EC2BC63F8CA5B35394FA812588376DA306ECAC078AA62D55。
- **欠账**：TaskCenter session 维度结构刀；无参绑定「内核会话」语义逐个审计。

## 最新发布：v4.180.0（2026-09-09）「办公任务管理会话关联刀：会话待办区入任务管理页」

- **定位**：用户反馈任务管理「没与当前会话关联、全是固定内容」——页面主体 TaskCenter 数据源 `GaeaTaskList`=全局后台任务表（cron 周期任务持续在列，天然会话无关），而会话真任务（todo_write 待办）无入口；子代理/本地模型工具两区本就按会话建册。
- **修复**：TasksWorkbench 首区新增「会话待办」——直订全局 controller store items（零 props 穿层，会话切换自然更替，ContextModal 弹层同源正确）+ 复用 useTodoExtractor 提取最新 todo_write；三态条目+进度 n/m，无待办不占位。页面四区=会话待办/子代理/本地模型工具（会话）+任务中心（全局）。
- **门禁**：vitest **2739/2739 首跑全绿**（+1）、tsc 0、eslint 0、e-check OK、drift **PASS@602**（零绑定）、零 Go 改动、build strip+冒烟 200、SHA256=69C89001048CB8BFD99BCE7122C69AB92A8C01A85CB4E34D23BAA45A1A71C0DD。
- **欠账**：TaskCenter 会话关联需 Go 任务表加 session 维度（结构刀另立版本）；「会话后台任务」过滤视图等该维度就位再评估。

## 最新发布：v4.179.0（2026-09-09）「瘦身清点刀：knip 死代码清除 + 依赖显式化 + 冷启动基线打点」

- **knip 首扫三发现三处置**：①死文件 13 全删（App/AppBar/MobileSheet/SettingsMobile/SettingsUpdates/自制 Tooltip/PromptShelf/office 死链对/memoryhub 两件/typesGenerationCheck/m3-palette，全仓零 import 甄别含 scripts 层；连带清 zIndex.ts 过时注释）；②幽灵依赖 13 包显式化（jszip×8 处/katex/dayjs/unified/hast-util-sanitize/@lezer×8 共 20 处 import 靠传递依赖侥幸工作，按 lock 现版本钉死）；③顺带清死传递依赖 @codemirror/search@6.7.2（零引用）。三存疑依赖验活结案：gsap/docx-preview/unist-util-visit 全部在用保留。
- **冷启动基线打点（轨道五先测量后优化）**：Go app.New 一条+Startup 七段（migrate/logging/engine/voice+weixin/charlib+stores/tts-kickoff/schedulers）+total 落长期日志——日常启动自动积累真机基线；前端 gaea:boot mark→两 rAF gaea:interactive measure（CDP 可读；有意不落 gaea.log 避免污染 Error 级日志通道）。
- **初始化链审计结案**：TTS/价格/索引/模型刷新/预载/巡检全部已异步化，同步链只剩轻量磁盘活——懒初始化改造无需立项，轨道五收敛为真机采集（与壳内真机池同窗口）。
- **门禁**：vitest 2738/2738（5 例并发 flaky 单独复跑全绿）、tsc 0、eslint 0、drift **PASS@602**（零绑定）、e-check OK、Go 116 包 0 FAIL、build strip+冒烟 200、SHA256=3FCB187F496AF2B1A89884D860F675D1E56A92B6B50AA932E0655A65521D320F。
- **度量**：exe 46.16MB 持平（死文件本就不进 bundle，收益在仓库卫生与依赖正确性）；knip 复扫死文件/幽灵依赖双清零。
- **欠账**：knip Unused exports 169 项挂下版（测试专用/预留面/真死三态甄别）；壳内真机池/审计刀D 真机取证不变。

## 最新发布：v4.178.0（2026-09-09）「造价·条目匹配键刀：定额编码贯通存储→导入→匹配→工具面→UI」

- **造价域缺口 §1 首刀**（docs/gaea-cost-domain-survey-2026-09.md）：条目匹配全靠标题字符串 → 编码体系贯通。
- **数据层**：SchemaV17 `cost_entries`/`cost_estimate_items` 增 `code`（归一化存储）+ `idx_cost_code`；迁移测试「新库全链 V1→V17 + V16 升级」双覆盖；`cost.NormalizeCode`（全角→半角/去空白/大写，空串=未录入）。
- **导入链**：表头 `fieldCode` 映射（定额编号/清单编码/项目编码/编码等 8 关键词最长胜出，「编号」仍噪声）；**编码优先匹配**——带码命中→覆盖提示、带码未命中→直接新增（同标题不同编码不标题兜底，杜绝跨子目误覆盖）、无码原语义；AI 解析 prompt 增 code 提取；vision 有意不动。
- **匹配消费方**：costref `entryIndexes` 三索引 + `matchEntry` 编码精确优先（matchedBy=code|entry_name|title）；带码未命中宁漏勿误配（测试钉死）。沉淀携带 item.Code + 引用条目继承编码。
- **工具面/UI**：cost_save 增 code 参数（覆盖未传保留原编码）+ cost_search 表格编码列；EntryModal 编码表单项 / LibraryView 编码 chip+行前缀 / ImportModal 可编辑编码列 / ProjectsView 明细编码输入+EntryPicker 带入；mock 同批贯通。
- **附带收口：e-check 三处陈旧守卫修复**（干净 HEAD 实测即红，非本刀引入）——E25 ChatSend/ChatImportTopic 接受 bridge 形态（v4.174 双轨退役甩下）；E24 MemoryHubPage→GraphView 接受 React.lazy 动态 import（v4.176 甩下）。
- **门禁**：Go 全量 0 FAIL（五包 +9 测试）、vitest **2738/2738**（2737+1）、tsc 0、eslint 0、drift **PASS@602**（零绑定）、locale 0 死、e-check OK、build strip + 冒烟 200、SHA256=F83CDB9BED35C7B4F045CC835B59981B95B5B4B67E39EC1D5282EF8F2E199561。
- **欠账**：造价缺口 2-6 不变；编码存量回填等带码样本导入自然补全（不做标题猜测回填）；壳内真机池/审计刀D 真机取证不变。
## 最新发布：v4.177.0（2026-09-09）「瘦身 P4 懒加载三刀收官（mermaid 动态化 H1 + locales 按需 H7 + SearchModal lazy H8）」

- **H1 mermaid 动态 import**：Markdown.tsx/mermaidPng.ts 删静态 import → ensureMermaid 异步 await（模块缓存，strict/loose 配置原样）；MermaidBlock 迁 useEffect async IIFE（取消/失败语义不变）。**全仓 mermaid 静态 import 清零、entry 对 mermaid.core 零引用、566.9KB 独立 async chunk 按需**；测试零改动。
- **H7 locales 按需**：zh 静态保留（首帧零异步）+ en/zh-TW 动态 import 显式分支（Vite 静态解析独立 chunk）；useT/t/setPref 签名零变动；Provider 切换先同步渲染回退→chunk 就绪 forceRender+alive。
- **主代理关键修复**：translate 回退链 `DICTS[locale]→DICTS.zh→DICTS.en→key`——**zh 静态恒可用兜底**，非 React 调用方（tools.ts 摘要，currentLocale 默认 en）在 en chunk 未加载时裸键永不外泄（首跑 13 例失败根因）。5 测试补 beforeAll(loadLocale('en')) + ModelSwitcher en 用例 await。
- **H8 SearchModal lazy**：React.lazy + Suspense fallback null；测试不经 MainLayout 零改动。
- **门禁**：vitest 2737/2737（首跑 13 例 i18n 时序失败修复后全绿；1 例 FilePreviewModal 并发负载 flaky 单独复跑 15/15 绿在册先例）、tsc 0、eslint 0、drift PASS@602、locale 0 死、Go 回归绿、冒烟 200、SHA256=E98E49CD8134C2F0964A1E041F49714465A385AB678E6295A946F94971E7B5D0。
- **度量**：**entry 1193→755.48KB（−37%）**；en=76.95/zh-TW=74.91/SearchModal 独立 chunk；**exe strip 实战验证**（build.bat 固化 `-ldflags "-s -w" -trimpath`，46.2→46.15MB 减量甚微——Go 符号表在 Wails 应用占比小，-trimpath 安全收益为主）。
- **P4 收官**：Go >50KB 0；entry −37%；MemoryHubPage 页壳 15.95KB；mock/office 1.1KB 入口；wailsjsCompat 退役（v4.174）。
- **欠账**：壳内真机池（rail 切换器/home 最近文档/.gsched）、审计刀D 真机取证（printSvg print/拖拽/粘贴）、观察池刀3（工作台内嵌办公）待评估——全部真机绑定的观察项不变。

## 最新发布：v4.176.0（2026-09-09）「瘦身 P4 结构刀2 + entry 懒加载（mock/office 拆分 + MemoryHubPage 全 tab lazy + mock 异步 chunk）」

- **结构刀2**：mock/office.ts 51.2→1.1KB 入口（6 文件：types/state/schedule 10.5/methods_xlsx 4.9/methods_office 32.7/build）。**TS2632 教训**：`let mockScheduleCurrent` 被 ScheduleProjectCreate/Delete 直接重赋值——跨模块重赋值导入绑定被 TS 禁止，schedule 状态与方法必须同文件（与 weixin.ts 域内私有状态先例一致）。
- **运行刀 entry 懒加载**（调研修正 P0 两处：cytoscape/cynefin 是 mermaid 传递依赖非 hub 相关、pageLazy 是 PDF 逐页懒挂载非页面 lazy——页面级 lazy 早全就位）：
  - **H6 mock 异步 chunk**：proxy.ts 删静态 import → startMockChunk 单例 + 真机门控零加载 + 冷路径异步 thunk；events 走 mockEventSharedSync/waitMockReady。**index 1149→967.23KB（−182KB/−15.8%）**。
  - **H2-H5 MemoryHubPage 页内 lazy**：8 组件全 React.lazy（**1402.5→15.95KB 页壳**；GraphView/three.js 1,354.97KB、KnowledgePanel/Markdown/katex/mermaid.core 全独立 chunk 按需）。
  - **H1 mermaid 动态 import 递下一轮**（mermaidPng.ts 在 footprint 外静态可达 + memoryhub lazy 已拆共享 chunk + Markdown 测试面大）。
- **连带测试时序修复×2**：mock 异步化使 `window.__mockScheduleFile`（schedule hook）与 SearchModal RouteIntent/UnifiedSearch 演示规则在同步断言时未就绪——两测试补 `beforeAll(await waitMockReady())`，bridge.ts 入口加 waitMockReady re-export。首跑 7 例失败修复后全绿。
- **门禁**：vitest 2737/2737 全绿、tsc 0、eslint 0、drift PASS@602、locale 0 死、Go 回归绿、冒烟 200、SHA256=402A1B1A937CF6690E435C1F4E61AA6BB6D72487543A6B078E0F9F5F132DBD2F。
- **欠账=P4 续**：H1 mermaid 动态 import（+mermaidPng.ts）、H7 locales 按需（−160~200KB）、H8 SearchModal lazy（−20~40KB）、**exe strip**（wails build -ldflags "-s -w" -trimpath 发布版，dev 保留符号）；壳内真机池/审计刀D 不变。

## 历史发布（v4.176.0 及之前，已归档）

- **v4.175** 瘦身 P4 结构刀1：三巨文件拆分（config.go/engine.go/store.ts，Go >50KB 2→0）
- **v4.174** 瘦身 P3 版3终局：bridge 双轨退役收官（wailsjsCompat shim 删除，LegacySurfaceNames 292→184）
- **v4.173** 瘦身 P3 版3 批次二：chat/novel/settings 三族 22 文件迁 bridge（契约扩展 29 方法+元组契约错位修正）
- **v4.172** 瘦身 P3 版3 批次一：wailsjsCompat 首批 12 文件迁 bridge（契约扩展 16 方法+2A/2B 迁移；LegacySurfaceNames 292→276）

- **v4.170** 瘦身 P3 结构版1：巨文件首批 4 拆（App/bridge/types/GanttView，>50KB 9→5；逐字节/绑定零变更纪律）
- **v4.169** 瘦身 P2 主干：双空间并列落地 + home 空间感知化 + 走查待证项收口
- **v4.168 / v4.167** 瘦身 P2 刀1（schedule 并入办公文档面）+ 刀0（白名单解耦、P0 基线落档、审计刀D Filters、nanoid 升级）
- **v4.166 / v4.165 / v4.164** 瘦身 P1 快赢（轮子四刀、审计刀B/C、locale 死键、依赖验活）+ 拍板落档（路线A+私有许可）+ 收敛计划 W1 卫生刀
- **v4.163 / v4.162** gaea 长期日志机制（按日分文件+365 天保留）/ 进度计划导入导出壳内修复（原生对话框链路）
- **v4.161 ~ v4.159** 进度计划双代号线：对齐标杆二期（工程标尺+图面语言）、一级/二级分级横幅、画布手感（滚轮缩放+行距自适应）
- **v4.158 ~ v4.156** AI 组价复核闭环 / docx 证据链补齐 / pptx 真编辑刀2+刀3（编辑面板+版本对比，Office 三件套编辑闭环）
- **v4.155 ~ v4.150** 双工期口径四刀：SS/FF/SF×cd 全搭接重推、MPP Project 2013+ 变体导入、mspdi 互通、板块 UI、agent 通道、任务级工期单位（wd/cd）
- **v4.149 ~ v4.144** diff 确认卡四刀（回滚/结构化 diff 预览/引擎逐条确认/ops 投影对拍收官）+ 工程复制 + 导入为新工程安全化
- **v4.111 / v4.110** 进度计划板块起始：CPM 引擎+三视图自由切换、Project/斑马对齐（v4.112~v4.143 期记录以「最后更新」大段落内联保留：多工程/资源成本/AOA 手动布局/基线对比等）
- **v4.109 ~ v4.100** pptx 真编辑刀1 数据层、导图画布/多维表、Model Hub 蒸馏 MH1-4 + ComfyUI 预热、Hub 落库+绑定转正、办公搜索对齐、GenUI 围栏审计
- **v4.99 ~ v4.90** 三线并行收口：图像域契约、GenUI 蒸馏（回答即 UI）、Verifier 逐页缩略、子代理网络会话/恢复入口、CodeMirror 高亮、Mermaid strict、diff 渲染升级
- **v4.89 ~ v4.82** 上下文页与工具链：成本费率 hover、/context 弹层、Git 面板最小集、文件三态折叠、HTML 沙箱预览、工具结果缩略卡、上下文趋势跳转
- **v4.81 ~ v4.60** 早期面板期：上下文/文件/任务线（活动行级增量、任务页、记忆界面、子代理会话/并行/流式/追问闭环、办公板块交付验收 A2、better-sidebar pane 化）
