## v4.37.1 · D 刀收口：模型库卸载确认带释放大小（2026-09-02）
> 模型中心调研 D 刀核伪后剩余增量（绑定面 557 零变更）：卸载确认 file_size 存在时
> 显示「释放 X GB」；磁盘占用展示与 /health 透出两项经核实已存在（E1-4 已落地、
> TestConnection 已探 herdsman），剔除不入刀。
- 验证：go build/vet、tsc -b 0、vitest 模型中心目录绿、drift PASS（557）。详见 releases/v4.37.1.md。

## v4.37.0 · 健康巡检 + 故障转移 v0（2026-09-02）
> 模型中心调研 C 刀：巡检/转移在桌面与 Web 端竞品中全部空白，gaea 以既有 Status
> 持久化与 fallback 路由语义低成本补位。**绑定面 555→557（+2）**。
- **① 健康巡检**：goroutine 10 分钟周期探已启用非本地引擎（GET /models 8s 超时），
  Status 持久化+「连续 N 次探测失败」前缀+状态变化事件 engine-health-changed；
  Error 永不含 Key（对抗回显测试锚定）。
- **② 故障转移 v0**：开关 engine_failover_enabled（默认关=现状逐字节）；网络类/408/429/5xx
  才转移（401 等配置错误不转移），候选=已连接 llm 引擎按 order 取首、用其默认模型重试一次，
  流式仅首字节前转移；emit model-failover + 逐笔记账。
- **③ 前端**：「本地调度」第三开关卡「故障转移」（降级「未知」禁用）+ 双事件订阅 +
  总览引擎卡连续失败原文/巡检时间 title。收口注记：gen_bindings explicitOverrides
  登记教训（新方法先登记再生成，否则被前缀规则误归）。
- 验证：Go 全量 test exit 0、tsc -b/eslint 0、vitest **1243/1243**（+6）、drift PASS
  （**557**）。详见 releases/v4.37.0.md。

## v4.36.0 · GLM 目录 v2：能力/价格元数据 + 远程热更新（2026-09-02）
> 模型中心调研 B 刀（=「计费三件套」本体升级版）。官方价格页动态渲染抓不到正文，
> 查不到的绝对价一律不编数，估算沿用内置 z.ai USD 价口径。**绑定面 555 零变更**。
- **① 目录 schema v2**：glm_catalog.json 44 条目（legacy 22+官方新增 22），条目带
  context_length/max_output/price/currency/unit/free/caps/price_note/coding 积分系数
  （仅 glm-5.3 与 5.3-flash）；免费档 8 个；官方核实国内价 6 条；解析兼容旧裸数组。
- **② 远程热更新 v0**：config 键 glm_catalog_url（默认空禁用）+ 24h 周期拉取 + version
  比对 + 本地缓存兜底；优先级 覆盖文件>远程>内嵌；仅影响展示与估算，不碰路由/alias/鉴权。
- **③ 估算单源化**：estimatePrice GLM 分支目录优先、内置表兜底——GLM 价格更新只动目录
  不发版；估算值零回归锁；ModelStatsSummary 透传 catalog_version/source 三态。
- **④ 前端**：模型卡上下文/能力/价格徽标（免费绿标），StatsSection coding 行官方公式
  估算积分（含缓存命中通道）+ 价格目录来源小注。
- 验证：Go 全量 test exit 0、tsc -b/eslint 0、vitest **1237/1237**（+18）、drift PASS
  （555 零变更）。详见 releases/v4.36.0.md。

## v4.35.0 · 自定义引擎：OpenAI 兼容服务商任意添加（2026-09-02）
> 模型中心专项调研（docs/market-research-2026-09-02.md）A 刀：自定义服务商是桌面客户端
> 4/5 家标配，gaea 8 引擎硬编码为最大硬缺口。**绑定面 552→555（+3）**。
- **① 引擎类型与生命周期（Go）**：新类型 EngineCustom="custom"（OpenAI 兼容）+ Manager
  六方法（Add/Update/Remove CustomEngine + customKeys 注入/取用）；engineID=custom-前缀
  +slug 冲突追加序号；Key 存 config 新键 custom_engine_keys（JSON map 加密值，
  saveSetters 登记+显式往返测试），不落 engines.json 不下发前端；baseURL 校验 http(s)
  +host（v4.9.1 Key 粘错框防线延伸，Key 当地址粘入被拒）；LoadState 恢复 custom 条目
  （type+地址合法性双校验防伪造）。
- **② 聊天路径**：BuildChatURL/resolveChatEndpoint（流式+非流式）custom 分支——
  自定义引擎真正可聊天/设活跃/绑功能；空 Key 不发 Authorization 头（无鉴权本地服务可用）。
- **③ 前端**：引擎管理分区「添加自定义引擎」表单（前后端同口径校验双保险）+ custom 卡
  地址框/编辑（Key 留空=不改）/删除确认/「自定义」徽标；内置云端引擎地址框防线一字未动
  （新增回归锁）。
- 验证：Go 全量 test exit 0、tsc -b 0（bridge.ts LegacySurfaceNames 登记 3 名）、
  eslint 0、vitest **1219/1219**（+9）、drift PASS（**555**）。详见 releases/v4.35.0.md。

## v4.34.0 · 子代理气泡恢复：恢复会话不再丢失子代理答复（2026-09-02）
> 收 v4.26 沿旧欠账「子代理气泡恢复暂缺」。根因=ProjectMessages 投影无 subagent_message
> case 整条忽略；模型面投影不可动（恢复后模型上下文须与实时语义一致）→ UI 侧并行投影。
> **绑定面 552 零变更**。
- **① UI 侧锚点投影（Go）**：session 新导出 KindSubagentMessage 常量 + ProjectSubagentAnchors
  （与 ProjectMessages 逐 case 同拍的游标镜像，subagent_message 记「插在第 K 条消息后」
  锚点，projection.go 一字未动）；GaeaHistory 读磁盘事件日志按锚点合并子代理气泡
  （mergeSubagentAnchors 纯函数 + logOffset 校正检查点 system 提示导致的系统性偏移，
  负位/越界锚点宁漏勿误丢弃）；HistoryMessage 加 subagentRef 字段（golden 不变）；
  GaeaResumeSession 零改动自动生效。
- **② 恢复徽标消费（前端）**：rebuildHistoryItems assistant 分支透传 subagentRef（空串
  归一），复用实时「子代理」徽标渲染；HistoryMessage 类型交叉扩展注明可回收。
- 验证：Go 全量 test exit 0（线A -count=2，投影/Restore/golden 回归全绿）、tsc -b/eslint 0、
  vitest **1210/1210**（+3）、drift PASS（552）、wails generate module 已刷新、版本四处
  4.34.0。详见 releases/v4.34.0.md。

## v4.33.0 · 细节收口第三刀：回滚守卫统一 / pdf 占位比精确化 / 主区预览懒加载对齐（2026-09-02）
> 欠账池三线并行子代理 + 主代理集成。**绑定面 552 零变更**。
- **① 回滚守卫统一 + write_file >8KB 恒误报修复（真 bug）**：rollback 卡接入「恢复后已被
  手工修改」守卫（撤销恢复前校验，防覆盖编辑）；write_file 守卫从精确比较改
  `evidence.ClampSummary` 同口径截断比较——原精确比较对 >8KB 未手改文件必然误报拒绝；
  截断单点化（RecordChange/app 复用）。已知边界：8KB 摘要窗口外手改不可检（宁漏勿误）。
- **② pdf 占位比按实测精确化**：pageLazy 新 nextPageAspect/placeholderAspect——页图
  onLoad 读 naturalWidth/Height，首个有效测量为整档比例（不被后续页推翻），无测量回落
  A4 估计；占位→真身交换不再跳高、滚动条比例修正。弹窗 FilePreviewModal 已接线。
- **③ 主区预览 pdf 懒加载对齐弹窗**：FilePreview pdf 分支接入同款 IO 单向懒加载（初始
  4 页/800px 预挂/不卸载/大纲跳转强制渲染/ref 回调登记即补 observe），无 IO 全量降级；
  主代理集成追加测量比例接线，占位表现与弹窗一致。
- 核实：v4.26「TrajectoryView 未消费 subagent 记录」欠账已过期剔除；「子代理气泡恢复」
  留作独立刀（GaeaHistory×事件日志跨源对齐）。
- 验证：Go 全量 test exit 0（线A -count=2）、tsc -b/eslint 0、vitest **1207/1207**（+11）、
  drift PASS（552）、版本四处 4.33.0。详见 releases/v4.33.0.md。

## v4.32.0 · 细节收口第二刀：回滚可撤销 / 产物自动弹出 / 弹窗 pdf 懒加载 / 预览最大化持久化（2026-09-02）
> 用户点名「继续优化完善 gaea」——欠账池挑四条互不相交线，三并行子代理 + 主代理 App.tsx 接线集成。**绑定面 552 零变更**。
- **① 回滚先快照当前态**（收 v4.28 B1 欠账）：GaeaRollbackRecord 恢复前把目标当前内容
  快照到原基线同目录（evidence 新导出 StageBaselineTo，命名逻辑单点化），rollback 记录
  升级为完整证据卡（Before/After 原文+BaselinePath）——**恢复动作本身成为时间线里可再
  恢复的版本（撤销恢复=对 rollback 卡再点恢复）**；目标缺失/快照失败降级不阻断恢复。
- **② 产物自动弹出 + 偏好**（收 v4.30 欠账）：新 deliverablePrefs（gaea.deliverableAutoOpen，
  **默认关** opt-in）+ DeliverablesPanel 头部胶囊；App 新产物 diff 时偏好开且 tab 未停用
  →亮右栏切「产物」tab（激活即清零角标，不动 FilePreview）；单版本「版本」徽标 title
  细化为「有 N 个历史快照，可预览/恢复」（收 v4.31 欠账）。
- **③ 弹窗 pdf 逐页懒加载**（收 v4.31 欠账）：新 lib/pageLazy 纯函数 + IntersectionObserver
  单向懒加载（初始 4 页、进视口 800px 预挂、已挂载不卸载杜绝滚动跳动），大纲跳转目标页
  强制即时渲染，无 IO 环境全量降级；顺带修 preview/loading 两次提交致 IO 观察集为空、
  懒加载永不触发的真 bug（ref 回调登记即补 observe）。
- **④ 预览最大化持久化**（收 v4.30 欠账）：gaea.previewMaximized 独立键（writePrefs 只落
  sizes 数字 map），懒初始化 + toggle/拖拽退出三处落盘；半幅宽度本就落盘，还原仍回上次
  半幅。
- 验证：Go 全量 test exit 0（线A -count=2）、tsc -b/eslint 0、vitest **1196/1196**（+30）、
  drift PASS（552）、版本四处 4.32.0。详见 releases/v4.32.0.md。

## v4.31.1 · -count>1 全量绿化：测试全局态 -count 不兼容根治 + whisper 末气泡真 bug 修复（2026-09-02）
> v4.31.0 线 D 收尾延伸：全量 go test -count=2 ./... 从 FAIL → 全绿。**绑定面 552 零变更**。
- **根因（统一）**：测试写进程级全局状态（provider/billing/boot/app 注册表 kind、
  whisperSessions 会话缓存），-count 多次运行不兼容；whisper 10m 超时为**真 bug**。
- **修法**：注册 kind 改 testKind(prefix)（进程级 atomic 单调计数，任意 -count 唯一，19 注册
  点）；app whisper 会话隔离改唯一会话 ID + t.Cleanup 清理缓存（12 调用点）；whisper
  PacedStreamEmitter.pump streamDone 分支收尾末气泡（+3 生产行，修 MarkDone 挂起/末气泡
  OnBubbleEnd 永不触发）。
- 验证：五包 -count=2/-count=5 全绿、发射器 -count=300 全绿、tasks -count=20 仍全绿、**全量
  go test -count=2 ./... exit 0**；前端零改动；drift PASS（552）；版本四处 4.31.1。详见
  releases/v4.31.1.md。
## v4.31.0 · 细节收口四线并行：单版本入口 / 弹窗 pdf 预览 / 历史轮耗时 / tasks 竞态根治（2026-09-02）
> 用户点名「并行使用子代理」——四线足迹互斥并行落地 + 主代理集成。**绑定面 552 零变更**。
- **① 产物版本时间线单版本入口**（收 v4.28 B1 欠账）：徽标条件从 {rev && …} 放宽为
  {(rev || journalEntry) && …}——versions>1 按现状 vN 徽标（旧锁不破），versions≤1 但有
  journal 快照（baselinePath）的产物渲染「版本」入口徽标（title 区分「更新 N 次」与「有版本
  历史」），无快照保持空态；VersionTimeline 本体零改动。
- **② FilePreviewModal pdf/pptx 逐页预览**（收 v4.28 欠账）：弹窗 kind="pdf" 分支补齐逐页
  缩略（data-pptx-page 锚点）+ dataUrl 整本回退 + 诚实空态 + PptxOutline 大纲卡（页锚点滚动/
  「针对第 N 页修改」composer 插入）；FilePreview.tsx 本体零改动，非 pdf 分支逐字节未动。
- **③ 轨迹历史轮耗时**（收 v4.26 欠账）：后端 Turn.DurationMs（fold turn_done 分支
  Ts>StartedAt 时 =差值×1000，omitempty 向后兼容）+ 前端 TrajectoryTurn.durationMs +
  TrajectoryView 轮次头「用时 Ns」（复用 formatElapsed）；零新增绑定（结构字段级）。
- **④ TestCancelConcurrentStress flaky 根治（实现层真竞态）**：根因=pickNext 不做任务级预留
  →多 worker 同时 execute 同一 queued 任务，claim 落选者无条件 unregisterCancel 删掉
  cancelReq（用户取消意图）→ Cancel 已成功返回的任务终态被 succeeded 吞掉。修复=tasks.go
  新 clearStaleCancel（只清残留预注册、绝不删 cancelReq）+ claim 成功后胜者重登记 cancel；
  测试改事件驱动等待（50 终态事件到齐），断言不削弱且双向加固（Cancel==nil ⇒ cancelled /
  未取消 ⇒ succeeded / 终态事件不重不漏）+ 确定性契约单测 TestClearStaleCancelOwnership。
- 验证：Go 全量 0 FAIL（tasks -count=20/100 两轮全绿、-count=3 全绿）；tsc/tsc -b/eslint 0；
  vitest **1166/1166**（+9：A3+B3+C3）；drift PASS（552）；版本四处 4.31.0。详见
  releases/v4.31.0.md。
- 发布后补充（-count>1 全量绿化）：billing/boot/provider duplicate kind → testKind 唯一化；
  app whisper 会话缓存 t.Cleanup 清理 + 隔离测试唯一会话；whisper PacedStreamEmitter
  MarkDone 末气泡收尾（+3 生产行，修末气泡 OnBubbleEnd 永不触发的真 bug）；**全量
  `go test -count=2 ./...` 与 `-count=5 ./...` 均 FAIL → 全绿（exit 0）**；tasks
  `-shuffle=on -count=10` 无顺序依赖。
## v4.30.0 · 办公 UI 化繁为简第二刀：产物置前 / 行级降噪 / 命令面板视图重排 / 预览两档（2026-09-02）
> 用户点名「继续优化完善 gaea」，收 v4.29.0 欠账四项，红线不变：简化界面不是删除功能。
> **绑定面 552 零变更**（纯前端呈现重组）。
- **产物生成自动置前/角标**（Devin Auto-open 式）：App diff 会话内新产物路径 → 产物 tab
  角标（未查看数，激活即清零）+ 产物面板行「新」徽标与高亮（data-fresh 锚点可测）；
  会话切换重置基线，恢复会话不误标「新」。
- **面板行级降噪**（Cowork 一行式）：产物/变更/任务三列表次级信息（路径/相对路径/时间/
  重试计数）改悬停次行显现（group-hover opacity 过渡），title 全保留；主行断言零改动。
- **命令面板按当前视图重排**（Linear 式）：新 lib/paletteRank 纯函数——当前激活右栏面板
  cmd 置顶、chatTab=overview 时概览置顶、其余稳定保序；CommandPalette 零改动。
- **预览半幅↔最大化两档**（VS Code Toggle Maximized Panel）：FilePreview 头部新增最大化/
  还原按钮（icons 补 Maximize2/Minimize2），最大化占满可用宽度、还原回半幅（ref 记忆）、
  拖拽分割条自动退出最大化；不传回调不渲染按钮向后兼容。
- 验证：Go build/vet 0 FAIL（零 Go 变更）；tsc/tsc -b/eslint 0；vitest **1157/1157**（+10）；
  drift PASS（552）；版本四处 4.30.0。详见 releases/v4.30.0.md。
# gaea · 多功能 AI 助手

## v4.29.0 · 办公 UI 化繁为简：顶栏收拢 / 自适应标签 / 预览降噪（2026-09-02）
> 用户点名主轴「UI 界面化繁为简，参考市场同类产品」，红线 **简化≠删除功能**。
> 弹药：模块制调研两线（AI 代理工作台 / 办公文档工具）→
> docs/market-research-2026-09-01b.md（原始稿 docs/research-2026-09-01b/）。
> **绑定面 552 零变更**（纯前端呈现重组）。
- **顶栏导出收拢**：新 ExportMenu——「导出 / Word / PDF」三个常驻文字钮收进
  单钮「导出 ⌄」下拉（对标 Devin/Linear「新动作只进菜单不加按钮」+ VS Code
  顶栏单点溢出）；md/Word/PDF 三出口与统一交付管线原样保留，仅呈现收敛。
  顶栏常驻操作钮 7→5。
- **右栏 tab 窄栏自适应图标化**：容器 <420px 时 6 tab 文字 CSS 隐藏、只显
  图标（aria-label/title/角标全保留，对标 Notion 视图 tab Icon only/Text
  only）；宽栏恢复文字。6 tab 集合与数量锁不动（WorkspaceTabs compact 受控
  覆盖可测）；340px 基线宽下 6 带字 tab 拥挤问题根治。
- **预览头部降噪**：FilePreview 头部「打开/定位」图标化（title/aria-label
  保留，测试按 title 锁）+ 全部头部按钮去边框（无边框+悬停浅底）；「编辑/
  保存/取消」等状态语义动作文字保留（编辑能力保留红线，测试钉住）。
- 验证：Go 110 包 0 FAIL；tsc / tsc -b / eslint 0；vitest **1147/1147**
  （+9：ExportMenu 5、WorkspaceTabs compact 3、FilePreview 图标化 1）；drift
  PASS（552 零变更）；版本四处 4.29.0。详见 releases/v4.29.0.md。

## v4.28.0 · 浏览器与版本：观察窗 / 版本时间线 / pptx 交互（2026-09-01）
> 规划「浏览器与版本」刀（A2+B1+B2/C3）。**绑定面 550 → 552（+2：
> GaeaPptxOutline / GaeaBrowserObserve）**。三并行子代理分线+主代理集成。
- **A2 浏览器观察窗**：右栏新「浏览器」tab——CDP 截图步进流（captureScreenshot
  jpeg ≤1280 缩放；未运行 Available=false 绝不拉起）+ URL/标题 + 操作时间线
  （browser_* 倒序上限 20，对标 Trace Viewer Actions）+ 权限静态行 + 自动弹出
  胶囊（gaea.browserAutoOpen，App 接线新 browser_* 工具自动切 tab，2.5s 可见
  门控轮询）。实时帧流/人工接管远期。
- **B1 文件版本时间线**：产物 vN 徽标可点 → 内联时间线（时间/工具/轮次/状态）
  + 基线预览（GaeaPreview abs）+ 恢复（RollbackRecord，恢复=新增证据卡不丢
  历史）——**零 Go 改动**，完全长在证据链上（对标 Notion 版本史/Artifacts
  rewind，预览即护栏）。
- **B2/C3 pptx 交互**：新绑定 GaeaPptxOutline（python-pptx 结构化大纲）+
  GaeaPreview .pptx 分支（soffice→PDF 缓存 7 天 TTL + poppler 逐页缩略 ≤60
  页）→ 前端逐页预览+大纲侧栏+页锚点滚动+「针对第 N 页修改」指令插入；
  python 缺失降级诚实。
- 验证：Go 110 包 0 FAIL（stress flaky 沿旧）；tsc -b/eslint 0；vitest
  **1138/1138**（+43）；drift PASS（552）；版本四处 4.28.0。详见
  releases/v4.28.0.md。

## v4.27.4 · todo 持久化改名：.gaea/progress.md 撞名根治（2026-09-01）
> **勘误**：此前三次把 `.gaea/progress.md` 被覆写归因于「并行会话」——错误。
> 真凶是 gaea 自己的 `todo_write` 工具：计划进度持久化写 `<工作区根>/.gaea/
> progress.md`，办公代理在以 wubigrok 仓库为工作区跑任务时，每次 todo_write
> 都覆盖同名发布进度文件（一天四次，内容即任务 todo 表）。
- **修复**：todo_write 持久化改名 **`.gaea/todos.md`**（todo.go）；compaction
  读取端 `readProgressFile` 优先 todos.md、**回退旧名 progress.md**（存量工作
  区兼容）。项目记忆文件 `.gaea/progress.md` 从此不再被运行时覆盖。
- 测试 +2：saveProgressMarkdown 写 todos.md 且不碰 progress.md；读取端优先/
  回退语义（walk-up 设计使「均缺失」断言在真实机器不成立，已注明）。
- Go 110 包 0 FAIL（TestCancelConcurrentStress 负载型 flaky 沿旧）；前端零
  改动；drift PASS（550）；版本四处 4.27.4。详见 releases/v4.27.4.md。

## v4.27.3 · markdown 包裹符：交付卡片路径修复（2026-09-01）
> 用户报告「交付卡片点击无法打开、定位打开的不是文件位置」→ 真实会话实锤：
> 模型用反引号包裹路径，匹配把开头反引号吞进路径 → 预览「文件不存在」、
> 定位错位。**绑定面 550 零变更**。
- **根因**：fileLinks 路径字符集不排除 markdown 包裹符 `` ` `` 与 `*`——两者
  恰是 Windows 文件名非法字符，应作路径边界（v4.26.1 全角括号盲区第二弹）。
- **修复**：PATH_BODY/FIRST_SEG 排除 \`\` \` \`\` 与 *，PATH_BOUNDARY 纳入为边界，
  BARE_FILE_RE 分隔符后允许包裹符前缀；下划线等合法字符不受影响；存量消息
  渲染时实时重提取，重启即恢复可点。
- 测试 +5（真实会话文件名四形态+下划线守卫）；tsc -b/eslint 0；
  vitest 1095/1095；版本四处 4.27.3。详见 releases/v4.27.3.md。

## v4.27.2 · 细节收口：subagent_message 端到端 / 轨迹子代理记录 / 目录定位（2026-09-01）
> 细节打磨刀。**绑定面 550 零变更**。
- **subagent_message 端到端收口**（v4.26 回投特性此前实际未通——后端发
  kind=subagent_message、前端无消费整条被丢）：wire 层转译 kind="message"+
  subagentRef（磁盘日志仍按原始 kind 落），前端既有 message subagentRef 语义
  接管，「子代理」徽标气泡真实生效；补拉折叠同步（GaeaResyncItem.subagentRef
  恒全键、fold subagent_message→独立条目+closePending 防误续写）。
- **轨迹面板子代理记录**：TrajectoryRecordKind 加 "subagent"，徽标/Bot 图标/
  折叠行（答复摘要+ref）/详情全文/搜索命中，turns 与 betweenTurns 双落点。
- **sidebar_open 目录定位**（收 v4.25 欠账）：directory → FileTree 树中定位；
  顺带修 FileTree 目录行无 data-path 锚点导致 reveal 静默失效的暗坑。
- 验证：Go 110 包 0 FAIL（TestCancelConcurrentStress 负载型 flaky 单跑稳定）；
  tsc -b/eslint 0；vitest 1090/1090（+5）；drift PASS（550）；版本四处 4.27.2。
  详见 releases/v4.27.2.md。

## v4.27.1 · seq 防线 omitempty 失配修复：对话窗运行中只显示读秒的根因收口（2026-09-01）
> 用户报告「运行中只有一个思考读秒，没有交替出现过程卡/文本卡（只有轨迹面板
> 有显示）」。**绑定面 550 零变更**。
- **根因**：v4.26 seq 补拉防线前后端形状契约失配——Go GaeaResyncItem 全字段
  omitempty（流式 assistant 空 reasoning、写类工具 readOnly:false 的键被序列化
  省略），前端 parseResyncItems 严格校验缺键即整快照判坏 → 补拉快照 100% 被拒、
  防线静默失效；Wails 吞件期间对话窗无物可渲染（WorkHeader 是 store tick 驱动
  所以活着，轨迹面板读盘不受害）。
- **修复**：①Go 全字段去 omitempty + TestGaeaResyncItemWireAllKeys 锁「序列化
  恒全键」契约；②前端缺省键宽容（缺键→零值，类型错/kind/id/status 校验不变）。
- **真机验证**：真实应用发只读任务——对话窗 WorkHeader「已完成 · 用时 15s ·
  7 步」+ 阶段行 + 思考块 + ls 工具卡 + 正文交替（elapsed 3s→8s→14s 运行中
  逐个渲染）；v4.26.1 交付卡片同屏确认。
- 验证：Go 110 包 0 FAIL；tsc -b/eslint 0；vitest 1085/1085（+5）；版本四处
  4.27.1。详见 releases/v4.27.1.md。

## v4.27.0 · 右侧面板对齐 Codex：子代理对话实时下钻 / 对话与上下文完善（2026-09-01）
> 延续 v4.26「对齐 Codex」：右栏文件工作台、标签扁平化、对话输出、子代理实时
> 对话、上下文标签五面打磨。**绑定面 550 零变更（纯前端）**。
- **右栏文件工作台**：点文件后预览占满右栏（原顶部 3/5 小窗 + 底部文件树）；
  文件树收敛为「文件」按钮切换的 260px 侧栏；宽度上限 720→1600（视口 − 侧栏 −
  400 对话区动态钳制），首次打开文件自动抬升 560；编辑器 tab 加文件类型图标
  （lib/fileIcon 单源）；树内高亮当前编辑文件。
- **标签扁平化**：删「资料/成本库」，取消二级标签 → 文件/产物/变更/任务/分工
  一级平铺；运行角标按任务/分工下发；旧存储值自动收敛。
- **对话输出**：用户消息去气泡；第 2 轮起「第 N 轮」分隔线；助手消息复制按钮；
  编辑类工具 +N−N diffstat 芯片。
- **子代理实时下钻**：点击子代理 → 全面板对话（SubagentThread），运行中 3s
  轮询 + 事件驱动实时刷新、自动跟随底部；消息流 Codex 式（思考折叠/tool 卡）。
- **上下文标签**：总览头部水位分色（≥70% 琥珀/≥90% 红）+ 缓存/费用/刷新；空态
  引导；文件活动行点击打开预览；步骤详情「占窗口 %」；趋势图悬停构成详情。
- 验证：vitest 1082/1082（169 文件）；tsc -b/eslint 0；Go 零改动；drift PASS
  （550）；版本四处 4.27.0。详见 releases/v4.27.0.md。

## v4.26.1 · 全角括号文件名：交付卡片失配修复（2026-09-01）
> 用户报告「看不见完工交付卡片、无法点击查看文件」→ 真实会话实证：正文有
> 「交付文件：C:\…\开工筹备计划（修订）.docx」但卡片未渲染。**绑定面 550 零变更**。
- **根因**：fileLinks 的 PATH_BODY 把全角括号（）当路径终止符——文件名含（）
  （中文办公常态：（修订）（终稿））时正则截断、扩展名拼不上，交付卡片与
  内联文件链接整体失配。
- **修复**：路径体允许全角括号；扩展名仍锚定匹配末尾（「报告.docx（三份）」
  不吞补语）。+5 匹配用例 + 组件级回归守卫（DeliverableCards.regress）。
- 验证：tsc -b/eslint 0；vitest 1080/1080；Go 无改动；版本四处 4.26.1。
- 详见 releases/v4.26.1.md。

## v4.26.0 · 对话流式重造：对齐 Codex（2026-09-01）
> 用户报告「发送后对话窗静默而轨迹在动」→ 根因六连（子代理文字有意不进主聊天/
> 预处理窗零事件/Wails 吞件/Retrying 隐身/phase 空 seam/TTFT 静默）逐一对账。
> **绑定面 549 → 550（+1：GaeaResyncEvents）**。
- **工作态头部行（WorkHeader）**：turn 激活期常驻（spinner+阶段文本+已用时+
  步数，items 为空也渲染——发送那一帧起窗口不空）；完成转 Codex 式「已完成 ·
  用时 · N 步」耗时行；StreamingIndicator 收敛为兜底。
- **后端 phase 事件接线**：预处理各阶段（启动引擎/解析 @引用/装配上下文/检索
  记忆/思考中）+ Retrying/compaction 转译 phase（磁盘日志格式不变，200ms 节流）；
  phase 收编过程卡+头部。
- **子代理活动回投主回合**（Codex 2026-08 同款）：新事件 subagent_message 回投
  子代理最终答复（完成态，中途不回投防刷屏），主区消息「子代理」徽标；task 卡
  running 实时 lastText/lastTool 预览 + 完成结果摘要。
- **事件序号防线**：gaea-event 全量带 seq（会话切换归零），跳号→GaeaResyncEvents
  从磁盘日志折叠全量快照整体替换（5s 冷却/在途去重/坏快照保底/streaming 续接；
  golden 逐字节不变）。
- **重复工具折叠**「已调用 X · N 次」（Claude Code 式）；顺带修复
  weixin_reminder_test 时间炸弹。
- 验证：Go 全量 0 FAIL（golden/fold 原样通过）；tsc -b/eslint 0；vitest
  1072/1072（净增 71）；drift PASS（550）；版本四处 4.26.0。三并行子代理分线
  +主代理集成；调研 docs/research-2026-09-01/codex-streaming-ux.md。欠账与
  v4.27 顺延详见 releases/v4.26.0.md。

## v4.25.0 · 文件工作台：编辑器 tab 化 / 变更 diff / 选区联动 / 模型主动打开（2026-09-01）
> 规划 docs/gaea-office-upgrade-plan-2026-09.md 第三刀：A3 文件工作台 +
> B3 选区联动。**绑定面 549 → 549（零新增：sidebar_open 走内置工具事件管线）**。
- **编辑器 tab 化（EditorTabs）**：文件树点开 → 右栏内多文件编辑器 tab
  （lib/editorTabs 外部 store：上限 12 LRU/关闭激活相邻/localStorage 持久化
  坏值兜底）；FilePreview 新增 embedded 模式，docx/xlsx/md/图片/PDF 能力原样
  随迁（换壳不换芯红线）；双入口保留（树行点击=右栏内开 tab，右键=主区预览
  pane）；产物行「树中定位」→ FileTree 展开父链+滚动+闪烁（reveal）。
- **变更 tab diff 化（Git 面板式）**：文件行可展开 → 行级红绿 diff（lib/planDiff
  三态：edit_file/multi_edit 真 before/after；write_file/edit_lines 写入内容
  预览+原因；其余诚实不伪造）+ 回滚接证据链 Journal 最近基线（无基线诚实标注）。
- **B3 选区联动**：xlsx 选中单元格→浮动「引用到对话」；docx 框选工具栏补
  「引用到对话」；docx 渲染失败降级纯文本视图（docxText 提取正文段落+提示条）。
- **模型主动打开（sidebar_open）**：新内置 Go 工具（work 空间/ReadOnly 直允许/
  防穿越/envelope data path_rel）+ 前端解析器 + App 按事件 id 去重接线——模型
  把关键产物推到右栏编辑器 tab，file 开 tab/directory 亮文件 tab。
- 验证：Go build/vet/test 0 FAIL（+20 用例）；tsc -b/eslint 0；vitest 1001/1001
  （净增 74）；drift PASS（549）；版本四处 4.25.0。三并行子代理分线+主代理
  集成。欠账与下一刀 v4.26 详见 releases/v4.25.0.md。

## v4.24.0 · 子代理工作台：树拓扑 / 实时动态 / 产物登记表（2026-09-01）
> 规划 docs/gaea-office-upgrade-plan-2026-09.md 第二刀：A1 分工/子代理拓扑
> tab（AgentNetworkCard+SubagentsPanel 合体进化）+ C1 后端权威产物登记表。
> **绑定面 548 → 549（+1：GaeaDeliverableRegistry）**。
- **树形实时拓扑（AgentTree）**：嵌套 Children 全量渲染（此前只画两层）；
  root 折叠为「主 agent」行、更深层默认收起可展开；**新节点自动展开父链**；
  节点量化（状态色点/任务摘要/工具数/模型徽标/耗时——running 实时已用 1s
  tick/错误数）；下钻链：节点 → 详情卡 → 完整 transcript → **工具调用行点击
  定位结果消息**（收 v4.21 欠账）。
- **合并活动流（Devin 式单列 feed）**：running 子代理 lastText/lastTool 按
  updatedAt 倒序合并、上限 20、空态收起；树内行预览并存。
- **新子代理自动展开（可关，默认开）**：新 ref 出现 → App 亮出右栏切「分工」
  tab（停用时尊重停用态）；偏好键 gaea.subagentAutoOpen，损坏值回落默认。
- **C1 权威产物登记表**：trajectory.FoldDeliverables 从事件日志折叠写类 8 +
  生成导出类 3 工具的落盘登记（路径/工具/轮次/时间/次数，上限 200，Total 去重
  全量）+ 新绑定 GaeaDeliverableRegistry；DeliverablesPanel「权威产物登记」
  只读区（tool 徽标+路径+轮次+次数+时间，点击预览），补启发式漏登。
- 验证：tsc/eslint 0；vitest 927/927（净增 16）；Go build/vet/test 0 FAIL；
  drift PASS（549）；版本六处 4.24.0。详见 releases/v4.24.0.md。

## v4.23.0 · 工作台框架：右栏对标 DSH-better-sidebar 工作台化第一刀（2026-08-31）
> 用户拍板：右面板重造为 DSH-better-sidebar/Codex 式「运行工作台」（子代理/
> 浏览器/文件编辑器等实时操作面），状态显示类迁主区轨迹/上下文旁边。规划稿
> docs/gaea-office-upgrade-plan-2026-09.md（v2）。**绑定面 548 → 548（零新增）**。
- **Tab 注册表（lib/sidebarRegistry.ts）**：元数据复用清单 + render 接线单一
  数据源，右栏渲染与命令面板全派生；新增面板 = 清单 + RENDERERS 各一条，
  面板组件本体零改动（框架/内容解耦，为浏览器观察窗/编辑器 tab 留挂载点）。
- **工作台外壳三件套（学 better-sidebar）**：全局宽度键（左缘拖拽 280–720、
  最后一次拖拽胜出跨会话跟随）；声明式设置（齿轮→侧边卡片，每 tab 独立开关，
  停用即隐藏、至少保留一个、停用不进命令面板，启用集全局键）；会话记录 v2
  （JSON {v,tab,enabled,width}，v1 裸 id 兼容可读，坏值逐项兜底、失效指针修正）。
- **主区「概览」tab（统计迁移）**：ChatTabs 第 4 tab + OverviewPanel 承载原
  StatsPanel；右栏统计下线（union 全量移除，v4.22 旧 tab:"stats" 宽容收敛回
  「文件」并钉回归用例），右栏收敛 3 主 Tab×7 面板；命令面板新增概览入口。
- 验证：tsc/eslint 0；vitest 911/911（净增 38）；Go build/vet/test 0 FAIL；
  drift PASS（548）；版本四处 4.23.0。两并行子代理分线实现 + 主代理集成。
- 欠账：Tab 拆分/底部面板/自由窗口、设置二级弹窗、注册表懒加载 chunk 化；
  下一刀 v4.24「子代理工作台」。详见 releases/v4.23.0.md。

## v4.22.0 · 一次性收官：真虚拟化 / transcript 定位 / 晨报预载 UI（2026-08-31）
> 用户要求「一次性做完」：办公板块剩余本地可做欠账一次清完并整理提交收尾。
> **绑定面 546 → 548（+2：GaeaMorningPreload / GaeaSetMorningPreload）**。
- **轨迹真虚拟化（react-window v2 动态行高）**：扁平行流按视口窗口渲染
  （±overscan 12），超长会话 DOM 恒定，v4.21「首批+加载更多」分批机制退役；
  useDynamicRowHeight + ResizeObserver 实测展开行高自动重排；概览跳转走
  listRef.scrollToRow、搜索回顶、收起/展开照常；test/setup 补 ResizeObserver
  stub。
- **transcript 消息定位**：消息序号 #N + 搜索命中自动滚动到第一条命中。
- **晨报预载 UI 开关（+2 绑定）**：GaeaMorningPreload/GaeaSetMorningPreload
  （internal/config.Save 持久化 + 内存更新 + 重建引擎即时生效）；记忆面板
  「晨报预载 开/关」胶囊按钮（同款记忆开关交互）。
- 验证：Go 全量 0 FAIL（绑定面 548 PASS）；tsc/eslint 0；vitest 873/873
  （+3）；drift PASS（548）；版本四处统一 4.22.0。收尾：六轮改动作为一次
  合并发布提交 + tag v4.22.0；剩余欠账仅外部资源/官方数据项（Realtime 真机、
  自动路由、浏览器下载上传/headless UI/Windows UIA、iLink 真机窗口）。详见
  releases/v4.22.0.md。

## v4.21.0 · 长会话与 transcript：增量渲染 / 消息搜索（2026-08-31）
> 续 v4.20.0 剩余两条欠账：轨迹超长会话渲染量 + transcript 只读无搜索。
> **零新增绑定（546 不变，纯前端）**。
- **轨迹增量渲染（DOM 有界）**：轨迹改扁平行流（轮次头+展开记录+Between
  turns）按批渲染——首批 250 行，滚动到底自动续载或「加载更多（剩余 N 条）」，
  搜索词变化回首批；概览跳转同步扩可见区（不会再「跳过去了但没渲染」）；
  收起全部/展开全部在平行流上照常生效。
- **子代理 transcript 消息搜索**：查看器头部搜索框按正文/推理/工具名/参数/
  结果过滤，显示「命中/总数」，无匹配空态。
- **注释清理**：ChatTabs「轨迹暂占位」更新为 v4.17-v4.21 实际能力。
- 验证：tsc/eslint 0；vitest 872/872（+2）；Go 全量 cached 绿、drift PASS
  （546）；版本四处统一 4.21.0。欠账：增量渲染为分批 DOM 而非 react-window
  真虚拟化（渲染量有界但已渲染部分仍为真实 DOM）；transcript 只读无跳转/
  引用定位。详见 releases/v4.21.0.md。

## v4.20.0 · 剩余收官：子代理 transcript / 轨迹概览 / 旧会话趋势补齐（2026-08-31）
> 清掉 v4.17-v4.19 三刀之后的剩余欠账。**绑定面 545 → 546（+1：
> GaeaSubagentTranscript）**。
- **子代理完整 transcript 查看器（+1 绑定）**：新绑定 GaeaSubagentTranscript
  （sessionPath, ref）读取 `<sessionDir>/subagents/<ref>.jsonl` 全量消息
  （role/content/reasoning/toolCalls），ref 安全字符校验防穿越；前端 Agent
  网络详情面板增「查看完整 transcript」→ 消息流（角色徽标+推理+工具调用+
  结果，可收起）。
- **轨迹 Overview 投影 + 轮次跳转 + 折叠控制**：轨迹标签顶部概览条（每轮一
  根柱，柱高∝记录密度，工具调用高亮/报错标红，hover 明细，点击平滑跳转并
  展开）+「收起全部/展开全部」（长会话折叠成轮次索引；新回合默认展开）。
- **迁移/兜底会话趋势补齐（诚实估算）**：ToLogEntries 每回合合成
  request_header（system=真实 system 消息拼接，tools=该轮实际工具名集合，
  顺序与运行期一致）——旧会话从此有系统/工具分类与 header 轨迹记录；
  contextview 回合末估算关闭（turn_done 未见 usage 时用当前估算构成落
  estimated 记录），前端步骤详情显示「估算构成（无用量记录）」，不伪造用量。
- 验证：Go 全量 0 FAIL（+2 用例）；tsc/eslint 0；vitest 870/870（+3）；drift
  PASS（546）；版本四处统一 4.20.0。欠账：轨迹虚拟滚动未做（以收起全部+概览
  跳转缓解）；子代理 transcript 只读无搜索。详见 releases/v4.20.0.md。

## v4.19.0 · 看板收官：上下文浏览器 / /context 命令 / 子代理节点详情（2026-08-31）
> 续 v4.17.0+v4.18.0 的第三刀「继续完善」：上下文标签最后一个页脚占位收掉。
> **零新增绑定（545 不变）**。
- **上下文浏览器（surface 节点 + 归档）**：后端 contextview 折叠补全系统/
  工具节点（request_header 的 system prompt 与工具集合只在构成变化时入
  nodes，初版+变化版，每步重复不刷屏；文本=预览，全文在日志）；前端
  ContextBrowserCard——活跃/归档双页签（归档=被压缩移出节点，带「已压缩」
  标记）+ 六分类过滤 + 节点行（分类色点+≈tokens+文本预览，超长可展开）；
  页脚占位整行移除。
- **`/context` 命令**：GaeaCommands 内置 + i18n（zh/en），斜杠菜单可发现；
  classifyComposerCommand 增 context 分类，App.handleSend 拦截 → 切上下文
  标签（不发给模型）；CLI 未拦截路径走未知斜杠 Notice。
- **Agent 网络节点点击 → 子代理详情**：AgentNetworkCard 增 sessionPath
  （App 注入 currentSessionPath），点击子代理节点 → SubagentRuns 按任务前缀
  匹配（与后端 enrichAgentNetwork 同口径）→ 固定详情面板（状态/模型/工具
  调用数/更新时间 + lastText/lastTool + 最后回答摘要）；无匹配回退节点统计。
- 验证：Go 全量 0 FAIL（+1 用例）；tsc -b 0；eslint 0；vitest 867/867（+4）；
  drift PASS（545）；版本四处统一 4.19.0。欠账：子代理完整 transcript 查看
  器；轨迹 Overview 投影与虚拟滚动；迁移会话系统/工具分类与趋势柱（诚实不
  造数）。详见 releases/v4.19.0.md。

## v4.18.0 · 看板补全：文件活动 / 增量模式 / 实时刷新（2026-08-31）
> 续 v4.17.0（事件日志默认开启，数据源接通）之后的「继续完善」：收掉两个
> 看板剩余的占位尾巴。**零新增绑定（545 不变）**。
- **文件活动时间线（上下文标签新卡）**：后端 contextview 折叠新增 FileActivity
  ——工具参数确定性提取路径（path/rel/source/destination/image_path/output 键）
  + 工具→动作白名单（read_file/grep/vision/format_convert=读；write_file/
  edit_file/multi_edit/edit_lines/chart_gen/diagram_gen/screen_capture=写；
  move_file=移；ls=目录），screen_capture 从结果输出补记，bash 等无法确定性
  取路径者诚实不造数；同轮同步骤同路径合并、上限 200、空切片非 nil。前端
  文件活动卡（动作徽标+工具+路径+时间，倒序最近 40 条），页脚改为「上下文
  浏览器将在后续阶段接入」。
- **增量（Delta）模式启用**：趋势图「增量」按钮去掉灰置与 Phase B 标题，
  切换后展示每步相对上一步的净变化（绿=净增·红=净减，随模式显示图例），
  柱色改全站一致的可视化语义色。
- **运行中实时刷新**：新 hook useLiveReload 订阅 gaea 事件流——运行中节流
  刷新（1200ms）+ turn_done 立即刷新 + 整轮完成刷新；轨迹/上下文/Agent
  网络三处统一接入（替换「仅回合结束刷新」effect）。
- 验证：Go 全量 0 FAIL（+2 用例）；tsc -b 0；eslint 0；看板+mock-contract
  vitest 22/22（+2）；drift PASS（545）；版本四处统一 4.18.0。欠账：上下文
  浏览器（surface 节点浏览/归档）仍占位；Agent 节点点击跳子代理会话；轨迹
  Overview 投影与虚拟滚动；/context 命令；迁移会话系统/工具分类与趋势柱
  （旧消息无 request_header/usage，诚实不造数）。详见 releases/v4.18.0.md。

## v4.17.0 · 轨迹上下文接通：事件日志默认开启（2026-08-31）
> 用户反馈办公板块「轨迹」「上下文」标签是空壳——根因不是 UI，而是数据源
> （事件日志）缺省关闭：`session.log_format` 缺省 legacy → sink 不接线 →
> 看板恒读空日志。本刀把事件日志改为**缺省开启**并补旧会话读端兜底。
> **零新增绑定（545 不变）**。
- **事件日志默认开启（数据源接通）**：`config.EffectiveLogFormat()` 缺省
  "event"，仅显式 `log_format = "legacy"` 退回旧行为；`gaea_handler.go`
  注入生效值、boot 同源创建 EventLogSink——轨迹/上下文/Agent 网络三看板
  从下一轮对话起即有真实数据。
- **旧会话读端兜底**：`session.ReadEntriesFor` 优先事件日志、缺失时从旧
  `<id>.jsonl` 投影折叠条目（纯读不落盘）；`GaeaTrajectory`/`GaeaContextView`/
  `GaeaAgentNetwork` 统一改用，存量 legacy 会话也能看板。
- **迁移产物带回合边界**：`ToLogEntries` 每条 user 消息前写 turn_started、
  流尾写 turn_done（ProjectMessages 忽略边界，恢复投影逐字节不变）；轨迹
  折叠兼容 `assistant_message`（内嵌工具调用展开为 tool 记录并与结果合并）。
- **资源释放**：boot 把 EventLogSink.Close 挂进 Controller.Cleanup——缺省
  event 后 Windows 上会话目录可删除/迁移（文件句柄泄漏面一并修掉）。
- 验证：Go 全量 0 FAIL（+6 用例）；tsc -b 0；看板组件 vitest 12/12；drift
  PASS（545）；版本四处统一 4.17.0。欠账：迁移/兜底会话无 request_header/
  usage（系统/工具分类 0、趋势无柱，新会话完整）；Agent 网络对 legacy 会话
  仅 root；上下文增量模式/浏览器/File activity/SSE 增量刷新仍为既定欠账。
  详见 releases/v4.17.0.md。

## v4.16.0 · 四刀并行：离线收口 / 浏览器键盘与 iframe / 复核可视化 / 晨报预装配（2026-08-31）
> 用户拍板「全部并行处理」：v4.15.0 欠账清单四个可离线方向由四个并行子代理
> 同步落地（足迹隔离，主控全绿门禁）。**零新增绑定（545 不变）**。Realtime
> 真机验证排除（需用户真 key+麦克风）。
- **①persona 侧离线裂缝收口（真 bug）**：gaea_whisper_causal/gaea_whisper_retell/
  whisper_handler 三处 `featureModel("chat")` → `routeModel("chat")`——全局离线
  过滤对 persona（轻语）链路生效（此前因果解释/记忆重述/WhisperChat 绑云端照样
  发云端），用户功能绑定语义不变（同源）；+2 离线回归测试。
- **②浏览器键盘级 Input + iframe（v4.13/14 欠账，零绑定零前端）**：新工具
  `browser_press`（第 11 工具）——Input.dispatchKeyEvent 键盘级输入（key 别名表
  + ctrl/alt/shift/meta 组合 + 可选 text 真实输入，Enter 补 `\r` 触发 keypress 真机
  踩坑修复）；browser_read/click/type 加可选 `frame` 参数——getFrameTree→
  createIsolatedWorld→contextId 执行（**iframe 内交互完整实现**，真 headless Edge
  真机验证 Read/Click/Type 全通）；snapshot 不下钻 iframe 诚实拒。
- **③Verifier 通道 B 结果进前端（v4.14 欠账）**：Verdict 增 channelBRatio/
  channelBPages/channelBArtifacts（omitempty 旧卡兼容）；证据卡 verdict 内联区
  追加「视觉复核：像素差异率 x.x% · N 页」+「查看复核产物」按钮（打开产物目录
  before/after PDF + 逐页 PNG）；无通道 B 旧 verdict 不渲染。
- **④晨报深度预装配（v4.14 欠账，零绑定零前端）**：memory.BuildMorningPreloadBlock
  纯函数（复用 BuildMorningBrief 排序口径，≤600 rune 确定性零 LLM）→ sysprompt
  装配点注入「【工作记忆晨报】」块（门控 Memory.Enabled && morning_preload &&
  space==work，play/mode=off 不注入=双空间红线）；config 键 morning_preload
  （默认 true，仅配置文件可控）。
- 验证：Go 全量 0 FAIL（+20）；**vitest 861/861**（+2）；tsc/eslint 0/0；drift
  PASS（545）；build.bat 冒烟 200。欠账：Realtime 真机（需用户资源）；自动路由
  本体（待官方逐模型数字）；浏览器 snapshot 不下钻 iframe/下载上传/headless UI/
  Windows UIA；通道 B 逐页缩略图；晨报预载无 UI 开关。详见 releases/v4.16.0.md。

## v4.15.0 · 聊天路由归位：plain 聊天离线过滤修复 + 「由谁回答」回显（2026-08-31）
> v4.14.0 欠账「自动路由 v1」经用户拍板收缩为最小价值刀——砍成本档位机制/
> 开关/UI（缓存价/峰谷价无官方逐模型数字，诚实不入表），只留两块真实价值：
> ①plain 聊天离线过滤裂缝修复（真 bug）②消息级「由谁回答/为何/花了多少」回显。
> **零新增绑定（545 不变）**。
- **聊天路由归位（bug 修复）**：chat_service.go:68/:105 + chat_handler.go:9 三处
  `featureModel("chat")` → `routeModel("chat")`——用户功能绑定语义逐字节不变
  （routeModel 步骤 1 与 featureModel 同源）；新增收益=全局离线模式对 plain 聊天
  生效（修复「总闸不总」裂缝，此前绑云端照样发云端、persona 却被滤）+ 无绑定时
  全局活跃/兜底与 persona 一致 + `model.route` 事件补齐（模型中心「当前生效」可
  展示 chat）。featureModel 保留（展示用），routeModel 零改动。
- **「由谁回答/为何/花了多少」回显**：modelengine 导出 `EstimateCostCNY`（本地/
  未知恒 0、USD 按汇率折算 CNY、非法汇率回退 7.2）；chat done 帧/ChatSend 返回
  加 `answered_by{engine,model,source,cost_cny}`（流式按 chunk.Usage 实算，usage
  不可达诚实记 0）；前端 `AnsweredByLine` 消息底部小字「由 X 回答 · 标签[ · 约
  ¥x.xx]」（费用 ≤0 隐藏费用段，不虚报）+ `useChatStream` 解析（旧事件静默跳过，
  向后兼容）。
- 验证：Go 全量 0 FAIL（+7：EstimateCostCNY 表驱动 5 + plain 聊天离线回归 +
  done 帧 SSE 断言）；**vitest 859/859**（+7）；tsc/eslint 0/0；drift PASS（545）；
  build.bat 冒烟 200。欠账：自动路由本体未做（待官方逐模型缓存/峰谷数字）；
  persona 侧 gaea_whisper_causal/retell 同类离线裂缝=观察项；plain 费用口径
  usage 不可达恒 0（诚实降级）。详见 releases/v4.15.0.md。

## v4.14.0 · 三箭并行：晨报预取 + 浏览器续刀 + 复核产品化（2026-08-31）
> 用户拍板「多刀并行」：三个互不相交的小刀由三个并行子代理同步落地（文件
> 足迹隔离），主控全绿门禁收口——①浏览器欠账（空闲 TTL 自动关停 + 多标签页）
> ②路线图 T0 欠账「做梦 2.0 主动预取 MVP」（纯本地晨报）③市场调研 ★★☆
> 「Verifier 产品化」（证据链翻成 UI，可审计护城河先发占位）。绑定面
> 544→545（+1：GaeaMemoryMorningBrief）。
- **浏览器续刀（internal/gaea/browser，零新增绑定、前端零改动）**：空闲 TTL
  自动关停（Options.IdleTTL 默认 10min、GAEA_BROWSER_IDLE_TTL env 覆盖；Ensure
  成功路径刷新 lastActive，once 守护 watcher 到期调 teardownLocked 自动回收，
  Shutdown 幂等停 watcher，到期后 browser_* 自动重拉闭环）；多标签页（Manager
  重构 conn+pageID → tabs map + activePageID，/json/list 全量 target 为真源；
  ListTabs/NewTab/SwitchTab/CloseTab，切换/新建置 epoch=0 旧 refs 诚实失效，
  关 active 自动切剩余、最后一个整体回收）；新工具 ×3（browser_tabs 只读 /
  browser_new_tab / browser_switch_tab）+ browser_close 可选 tab_id（缺省保持
  现语义逐字节不变）；compact 两表同步。+8 测试（TTL 回收/零禁用/env/列表/
  新标签/ref 失效/关标签/整体回收）。
- **做梦 2.0 主动预取 MVP（纯本地晨报）**：memory.BuildMorningBrief 纯函数
  （零 LLM/零 IO/确定性：max(UpdatedAt,LastUsedAt) 降序 top5 user/project 优先 +
  procedural/rule ≤3 条 + rune 边界截断 120 + 空输入非 nil 空数组）；新绑定
  GaeaMemoryMorningBrief() (string, error)（JSON 串对齐 GaeaCostGraph 先例：
  ListInSpace("work") 只读 + 近 24h dream 审计计数，零写库零落审计，play 红线
  安全）；前端 MorningBriefCard（首页 ml-info 记忆脉搏旁，仅 work 空间渲染，
  失败/空静默隐藏，全 token 样式）+ i18n home.morningBrief.* 三语；gen_bindings
  重生成（bindingNames 545）、spaceBindings 分类 work。Go +12、vitest +4。
- **Verifier 产品化（纯前端、零新增绑定、后端零改动）**：证据卡「三步展开」——
  卡面（无 baselinePath → 回滚禁用 + 「可复核明细」徽标 + 整卡点击展开）→
  第 1 层声明↔实况 diff（opsJson 单格 op × GaeaPreview 现取实况，口径同后端
  数值容差 1e-9/去空白/公式归一，✓/✗/跳过标注 + 近似比对脚注，预览不可用降级
  仅声明回放）→ 第 2 层操作回放时间线（序号 + type 徽标 + applyOne 风格中文
  描述 + 批量 op 折叠计数，旧卡无 opsJson 回退 beforeSummary）；lib/verifyDiff.ts
  纯函数层 + types.ts 补 baselinePath/opsJson + XlsxOpView/VerifyDiffRow；
  mock/office.ts 补证据域三绑定（此前零 mock）。vitest +23（verifyDiff 16 +
  证据区 7）。
- 验证：Go 全量 0 FAIL（+25 测试）；**vitest 852/852**（150 文件）；tsc/eslint 0；
  drift PASS（545）；版本四处统一 4.14.0；build.bat 冒烟 /api/health 200。
  欠账：晨报深度预装配（进 agent 上下文）列第二刀；浏览器 iframe/键盘级 Input/
  下载上传/headless UI/Windows UIA；Verifier 通道 B 结果未进前端、复核明细绑定
  留待真实需求；本地-云端自动路由 v1 顺延下一刀。详见 releases/v4.14.0.md。

## v4.13.0 · 自动操作·浏览器：CDP 控制 Edge + 7 工具面（2026-08-31）
> 「自动操作」四柱唯一空柱的第一块砖（调研后刀序④）：gaea 获得结构化浏览器
> 自动化——CDP（Chrome DevTools Protocol）控制 Edge，权限门 + 事件留痕第一
> 天挂上。零新增绑定（544 不变），前端零改动（工具经 Registry 自动进能力
> 面板与过程卡轨迹）。
- **internal/gaea/browser 包（新）**：msedge 三段式定位（GAEA_BROWSER_EXE
  env → Program Files 候选 → LookPath）；独立临时 profile 启动（绝不碰用户
  主 profile，Job Object 绑定 gaea 进程，父死子收）；页面级 CDP WebSocket
  会话（复用 gorilla/websocket：写串行 + 超时 + 幂等关，仿 realtime 范式）；
  Ensure 幂等 + 失联自愈重拉；URL 白名单只放行 http/https。默认有头（看得
  见=可信任），测试可 headless。
- **7 个 browser_* 内置工具（work 空间）**：browser_navigate / browser_read
  / browser_snapshot / browser_click / browser_type / browser_scroll /
  browser_close。snapshot 用 ref 机制（data-gaea-ref + 代数守门，页面跳转即
  失效诚实报 stale_refs），用法=「先 snapshot 拿 ref 再 click/type」；type
  用 React 兼容原生 setter + input/change 事件派发；结构化 envelope 返回
  （ok/timeout/not_found/stale_refs/validation_error 程序化码）。
- **权限门**：browser_read/snapshot 只读档（ReadOnly 恒放行）；其余五工具写
  档（交互 ask 档弹卡一次、可记忆规则）；permission subjectKeys 追加
  "url"——授权可固化为 browser_navigate(<url-glob>) 窄规则。不进 hardAsk
  （MVP 弹卡成本控制）；play 空间物理过滤天然不含。
- **留痕零改动全通用**：工具调用经既有 ToolDispatch/ToolResult 事件进会话
  JSONL（逐行带 space）→ trajectory 折叠 → 前端过程卡/Agent 网络自动展示
  ——全链对工具名零特判。
- 验证：**真机实测 PASS**（GAEA_LIVE_BROWSER_TEST=1：真 headless Edge 导航
  httptest 页 → 读文本 → snapshot 拿 ref → ref+selector 双路点击 → type
  联动回显 → file: 拒绝 → 跳转后旧 ref 失效，1.16s）；Go 全量 0 FAIL（+25
  测试：browser 包 19 + builtin meta 3 + permission url 3）；前端零改动、
  tsc/eslint 0、vitest 825/825（全量首跑 2 例负载型 flaky、复跑 0 failed，
  既有先例）；drift PASS（544）；build.bat 冒烟 200。

## v4.12.0 · 成本透亮：GLM 计价真实性 + 编码套餐积分口径 + 目录数据驱动（2026-08-31）
> 模块制市场调研（docs/market-research-2026-08-31.md）指出的「计费快变」风险
> 落地第一刀，兼收审计 T0 缺口②③：GLM Coding Plan 已改积分制（旧模型名
> 自动切换），静态目录与 token 计价都会失真——本刀把「花多少钱」做成真的。
> **零新增绑定（544 不变）**。
- **GLM 价格表补全（modelengine/stats.go）**：原表仅 glm-4.7 一条，GLM 用量
  实际未被计价；现按官方定价页（docs.z.ai，2026-08-31 核实，USD/百万 token，
  折 CNY 走既有 usd_cny_rate）补 glm-5.3/5.2/5.1/5/4.7/4.6/4.5-air/flash 系
  与 glm-4.6v，免费档（4.7-flash/4.5-flash/4.6v-flash）计 0；无法核实的诚实
  不入表（glm-5-turbo 显式置空挡板防 glm-5 前缀误匹配；cogview 系按张计费
  非 token 口径）。
- **编码套餐积分口径（billing_mode="coding_points"）**：glm 引擎走
  /api/coding/ 端点的调用不再按 token 估算费用（EstimatedCost=0、不进
  TotalCost），聚合以 glm@coding 单列（Tokens 计入、费用 0）——套餐内计
  「积分」不按 token 扣费；同桶混合窗口以最近一次调用口径为准（注释说明）。
- **模型别名注记（仅 coding 家族）**：官方 coding-plan 概览核实 4 条自动
  切换（glm-5.2/5.1→glm-5.3、glm-5-turbo/4.7→glm-5.3-flash）；ModelInfo 加
  alias_of 下发，前端模型卡「自动切换」标记 + title 说明；std 端点不注记
  （旧名独立计价）；记账归一让 glm-5.2 的用量落 glm-5.3 价格桶。
- **GLM 目录数据驱动（内嵌 JSON + 覆盖文件热更新）**：glmStaticModels 22 模
  型迁入 glm_catalog.json（//go:embed，新测试逐字锁定一个不增不减）；覆盖
  文件经 config 新键 `glm_catalog_path`（照 usd_cny_rate 先例启动注入，非密
  钥项）注入，mtime 变更自动重读（同 ID 替换 + 新 ID 追加），坏 JSON 静默
  回退内嵌——智谱目录快变时（调研：Kimi V1 已全平台下线）无需重编译。
- **前端**：engines.ts 类型同步三字段（alias_of/billing_mode/engines）；
  EngineSection GLM 卡 coding 家族追加「积分制计费，费用估算不含该端点用
  量」说明；StatsSection 明细行积分口径标签（费用列显示「积分内」非 ¥0）+
  按引擎小计区（glm@coding 显示「glm（编码套餐）」，旧数据无 engines 字段
  整块不渲染，向后兼容）。
- 验证：go build/vet/test 全量绿（新增 5 组 Go 测试：目录逐字锁定 / 覆盖
  mtime 热重载 + 坏 JSON 回退 / 别名注记 coding 有 std 无 / coding_points
  计费门控 + glm-5.2 归一落 glm-5.3 桶 / 价格表断言）、tsc 0 / eslint 0、
  vitest 825/825（+4）、drift PASS（544）、build.bat 冒烟 200。

## v4.11.0 · GLM 全模态纵深：生图后端 + 官方双端点（2026-08-30）
> 续 GLM 引擎主线：聊天已真机打通，本刀把 GLM 从「只能对话」补成全模态
> （生图）+ 支持编码套餐（官方双端点），并修复一处模型分类误判。
- **生图后端 `ai.GLMImageBackend`（kind=glm）**：按官方「图像生成」API 实现
  ——`POST /api/paas/v4/images/generations`、Bearer 认证。关键差异：官方
  schema 只收 model/prompt/size（**无 response_format**，仅回 URL；negative/
  seed 等 OpenAI 扩展字段也不收），故不复用通用 OpenAI 后端、只发官方字段；
  响应 URL 统一下载转 data URL（复用前端显示/落盘链路）；官方错误体
  `{"error":{code,message}}` 原样透出；200 无图提示「可能触发内容审核」；
  img2img 诚实拒绝（官方端点无图生图参数）。缺省模型 cogview-4-250304。
- **App 三处接线**：initImageBackend / SetImageBackend / 角色剧照
  buildPortraitClient 均支持 glm；GLM Key 经 `Manager.GLMKey()` 与 chat 同源
  取用（不读 EngineConfig.APIKey）；size 参数保留（官方接受）。
- **官方双端点切换 `SetGlmEndpoint`（绑定面 543→544）**：std=`/api/paas/v4`
  （按量付费）/ coding=`/api/coding/paas/v4`（编码套餐额度，官方
  coding-plan/quick-start 核实——填错端点会 404 或误扣费）。后端只收两个
  官方常量（GLMBaseURLStd/GLMBaseURLCoding），不透传自由地址；GLM 引擎卡
  Segmented 切换并落盘持久化；LoadState 脏地址防线兼容。
- **生图模型目录补全 + 误分类修复**：静态目录补 glm-image/cogview-4-250304/
  cogview-4/cogview-3-flash（锚定官方图像生成 API 枚举，18→22）；修
  glm-5-turbo 被通用 turbo 关键词误判为生图（GLM 引擎先按官方目录判型再落
  通用关键词表，回归测试锁死）。
- **前端**：模型中心「图片生成」加 GLM 云端选项；设置页绘梦引擎标签诚实化
  （此前云端引擎也标「本地引擎」）；classifyModel 补 cogview；新增
  glmEndpointFamily 端点家族判定。
- 验证：Go 全量绿（+15 测试）、vitest **821/821**、tsc/eslint 0、
  drift PASS（544）。
## v4.10.0 · 修复 GLM Key 保存被拒（2026-08-30）
> 真机实测：GLM 卡片保存 Key 报「不支持的配置项: glm_api_key」——config.go
> 加了 Key 常量与字段，但漏登记 Save 白名单 saveSetters，保存被拒。
- **修复**：saveSetters 登记 glm_api_key → configFile.GLMAPIKey。
- **防回归**：TestSaveSetters_CoverAllAPIKeyFields 用反射断言 configFile 所有
  `*_api_key` 字段都有 Save setter——以后新增任何密钥类配置漏登记即测试失败。
- 验证：Go 全量绿（+1）、vitest 818/818（ProgrammingPage 1 例既有负载 flaky
  单跑复绿）、绑定面 543 不变。

## v4.10.0 · GLM 按官方文档重写：无 /models 端点（2026-08-30）
> 用户实测两轮仍不可用后核对 docs.bigmodel.cn 官方文档，发现根本性误判：
> **智谱官方没有模型列表端点**（文档仅有 chat/completions 等），此前用
> GET /models 做测试连接/刷新模型永远失败（真机 401「内部错误」）。
- **静态模型目录**：glmStaticModels 锚定官方「模型概览」（2026-08-30）——
  glm-5.3/5.2/5.1/5/5-turbo、glm-4.7 系（4.7-flash 免费）、glm-4.6/
  4.5-air/4-long、多模态 5.3-flash/4.6v、glm-tts/glm-asr-2512/embedding-3/
  rerank；Kind 经 ClassifyModelKind 统一分类。
- **Key 校验改走 chat ping**：TestConnection 对 GLM 发最小 chat 请求
  （max_tokens=1）真实验证 Bearer Key；错误体按官方形态
  {"error":{code,message}} 原样透出（如「令牌无效」），不再出现凭空 401。
- **默认模型 glm-4.6 → glm-5.3**（官方文档全部示例所用旗舰）。
- 验证：Go 全量绿（+2 测试：静态目录分类断言 / httptest ping 401「令牌
  无效」与 200 两态 + Bearer 头断言）；vitest 818/818；tsc/eslint 0；
  绑定面 543 不变。

## v4.10.0 · GLM 引擎地址防呆修复 (2026-08-30)
> 真机实测：GLM 卡片对云端引擎露出了地址编辑框，用户把 API Key 粘进地址框
> 保存——base_url 变成 Key 本体，此后所有请求报 `unsupported protocol
> scheme ""`，且 Go 原生错误把 Key 原文回显到界面（二次泄漏）。
- **UI**：地址编辑框改为仅本地引擎（ollama/herdsman/cosyvoice）显示——云端
  地址是预置常量，本就不该可编辑（此前靠黑名单排除，新增引擎易漏）。
- **后端三道防线**：SaveEngine 拒绝无 http(s) 前缀的地址（错误信息不回显
  原值，防 Key 二次泄漏）；LoadState 忽略存量脏地址（保留预置——已中招的
  engines.json 重启应用即自愈）；fetchModels 对无效地址给不回显原值的友好
  错误。
- **用户侧善后**：重启应用即恢复 GLM 预置地址；因 Key 已出现在报错浮层与
  engines.json 明文里，建议到 open.bigmodel.cn 重新生成密钥后再填入「保存
  Key」框。
- 验证：Go 全量绿（+1 回归测试：保存拒绝/载入自愈/友好错误三断言，且断言
  错误信息不回显原值）；vitest 818/818；tsc/eslint 0；绑定面 543 不变。

## v4.10.0 · 模型中心新增 GLM 引擎 (2026-08-30)
> 智谱 GLM 云端引擎（OpenAI 兼容 `https://open.bigmodel.cn/api/paas/v4`），
> 照 DeepSeek/OpenCode 模式全链路接入。端点真实性已验证（/models 与
> /chat/completions 无 key 均 401=存在且要求鉴权）。
- **modelengine**：`EngineGLM` 类型 + 预置引擎卡（Label「GLM 云端」，默认
  模型 glm-4.6，展示顺序在 DeepSeek 之后）+ `UpdateGLMKey` + fetchModels
  认证/401 文案 + BuildChatURL key 注入。云端属性（IsLocal=false）——全局
  离线模式自动跳过、路由/用量统计按既有数据面自动归类（cloudEngineSet +glm）。
- **key 全链路**：config `glm_api_key`（DPAPI 加密落盘 + 旧明文一次性迁移）
  → 启动注入 → SetGlmKey/GetGlmKeyStatus 绑定（CoreB，绑定面 541→543）→
  GaeaSetProviderKey 支持 glm/zhipu/bigmodel 环境变量映射。
- **前端**：模型中心引擎卡自动渲染（engineIcons/Colors/Labels +glm），Key
  输入卡（脱敏回显/保存/状态刷新），api/engines.ts 类型与包装函数。
- 验证：Go 全量绿（modelengine 预置 7→8 引擎断言同步 + GLM key/URL/云端
  属性测试）；vitest 818/818（148 文件）；tsc/eslint 0；drift PASS（543）。
- 使用：模型中心 → 引擎管理 → GLM (智谱) 卡片填入 open.bigmodel.cn 的
  API Key → 测试连接 → 刷新模型（glm-4.6 / glm-4.5-air / glm-4.5-flash 等）
  → 绑定到各功能域或设为活跃引擎。

## v4.10.0 · Herdsman CLI 错误透明化 (2026-08-30)
> 真机诊断：模型中心「模型库」报「模型目录不可用，herdsman CLI 调用失败:
> exit status 3」——CLI 其实把结构化错误写在 stdout，旧代码失败路径丢弃
> stdout 只回显裸退出码，真实原因全被吞掉。
- **根因（本机三路实证）**：Herdsman 桌面端本次以**管理员身份**运行——非
  提权调用方查其 MainModule 被拒、`\\.\pipe\Herdsman-skill-v1` 连
  READ_CONTROL（Get-Acl）都拒绝；提权进程创建的命名管道 DACL 只允许提权
  令牌，普通权限的 gaea 打开即 Access is denied → CLI exit 3。旧提示
  「请确认桌面端已启动」完全误导（桌面端明明在跑）。
- **修复**：runHerdsmanCLI 失败路径捕获 stdout/stderr，优先解析 CLI 的
  JSON 结构化错误（error 字符串/对象两态兼容），裸退出码只作最后兜底，
  stderr 摘录一并透出；parseHerdsmanOpResult 同步容忍对象态 error。
- **定向提示**：Access is denied → 追加「疑似 Herdsman 以管理员权限运行，
  普通权限的 gaea 无权连接其控制管道，请用普通方式重启 Herdsman 桌面端」。
- **用户侧解法**：普通方式重启 Herdsman（若快捷方式/兼容性设置勾选了
  「以管理员身份运行」请取消）；或以管理员运行 gaea（不推荐，常驻提权）。
- 验证：Go 全量绿（+3 测试：假 CLI 端到端——非零退出透出结构化错误+定向
  提示且不再只见 exit status / 两态 error 解析+BOM 兼容 / 提示文案）；
  前端零改动；绑定面 541 不变（drift PASS）。

## v4.10.0 · 工作人设收口：办公秘书 + 节奏豁免 + 出口净化 (2026-08-30)
> 用户拍板：gaea 是工作助理，应专业、严谨的办公秘书，不是文艺女青年。
> 微信/语音实测「[SPLIT] 裸漏 + 答非所问」三根因一并收口。
- **节奏引擎专业人格豁免**：preset 带 professional tag 的人格（gaea、新增
  secretary）在 DecideRhythm 永久豁免碎碎念/独白拆分——PAD 标尺
  （-100..100）下 chatter 阈值（aro>0 && aff>3）形同虚设，办公通道每轮
  回复都被压成 ≤30 字碎片，这是「答非所问感」的第一根因；乐园陪伴人格
  （genki/tsundere 等 29 个）的碎碎念节奏保持原设计不动。
- **新增「办公秘书」人格**（PersonalityPresets 30→31 + 详细模板）：
  结论先行、要点分明、完整句；禁碎碎念/撒娇/文艺腔/情绪化/客服腔/
  网感用语；角色中心可选。语音与微信默认人格仍为 gaea（VoiceGuide 本就
  专业向，从此不再被节奏引擎带偏）。
- **[SPLIT] 出口净化**：该标记是 chatter 模式的内部格式协议，全仓库此前
  **零消费方**（节奏发射器只在 GUI 气泡路径生效）——WhisperChat 是
  GUI/微信/语音三出口的共同上游，在此把标记归一为换行（SplitOnMarker
  分条、去空条），任何出口不再见到裸标记。
- **搜索触发收窄（宁漏勿误）**：删除「什么/怎么/为什么/在哪/什么时候/
  最新/最近/实时/介绍一下/告诉我/帮我找/多少钱/是谁」等对话高频子串——
  朴素匹配把无关网页摘要灌进角色回复，是「答非所问」的第二根因；保留
  显式动词（帮我搜/查一下/搜索/上网查…）+ 硬时效词（天气/股价/汇率/
  金价/油价/比分/新闻）。身份守卫保留为纵深防御。
- 验证：Go 全量绿（+8 测试：professional 豁免（含计数器强制切换前级）/
  陪伴人格原设计保持 / secretary 注册 / 预设计数 30→31 / SplitOnMarker /
  触发词收窄矩阵（误触发 5 例 + 应触发 4 例））；前端零改动（vitest 818/818
  沿用）；绑定面 541 不变（drift PASS）。
- 说明：terse 镜像（≤15 字）维持 v4.8.3 口径（短疑问豁免已在）；情绪系统
  在工作通道只影响 TTS 韵律（Mood→TTS 既有设计），不进措辞。

## v4.10.0 · 做梦 2.0 蒸馏真实合并 (2026-08-30)
> 路线图 T0「做梦 2.0」第一刀：自动做梦按 name upsert 只增不减，确定性重复
> 记忆从此有了**非破坏**合并通道。
- **检测（memory.DistillMergeCandidates 纯函数）**：同空间内 ①归一化同名异写
  （存储后端自身已归一 kebab-case，本规则为导入数据兜底）②异名+同
  type+kind+描述逐字相同（重复沉淀主场景）；跨空间一律不成候选（双空间
  红线）；组内 UpdatedAt 降序取保留条，同输入同输出，候选封顶 8 条。
- **执行（control.DistillMerge）**：锁内重算候选集校验配对——绑定面不可被
  用于归档任意记忆，越权配对拒绝；Store.Archive 归档较旧条（**可逆**，不删
  数据）+ Touch 较新条 + refreshMemoryLocked；dream-audit.jsonl 落
  source=distill_merge 审计行。
- **前端**：记忆面板「建议」新增「重复记忆合并」卡区（保留/归档对 + 理由 +
  一键合并），i18n 三语 2 键；GaeaAcceptMergeSuggestion 新绑定（540→541，
  OfficeB）。既有办公事实库的 GaeaMemoryDuplicates/Merge（模糊相似度+硬
  删除）不受影响——两个库两条通道，互补不重叠。
- 验证：Go 全量绿（+7 测试：slug 变体 / 同类型同描述 / 跨空间排除 / 封顶与
  稳定 / 执行器与检测器一致性 / 越权拒绝 / 视图映射）；vitest 818/818
  （148 文件）；tsc/eslint 0；drift PASS（541）。
- 说明：file 后端 UpdatedAt 不落盘（语义为空），检测与执行均以存储实况为准；
  大组（同描述多成员）按设计一次只出 1 个候选，合并后再次扫描渐进收账；
  「做梦 2.0 主动预取」留后续刀。

## v4.10.0 · Verifier 通道 A 引用级深化 (2026-08-30)
> 审计欠账收口：「公式重算+引用/摘要比对」中的引用级比对此前未实装——通道 A
> 只做重算零错误。
- **声明↔实况比对**：ChangeRecord 新增可选 opsJson（xlsx_apply 落卡时随卡
  携带精确 op 载荷；截断保护=超限不落，宁勿落坏 JSON）；复核时逐条回读目标
  工作簿——set_value 比值（数值浮点等值容差）、set_formula 比公式（剥 =/
  空白归一）、replace 比替换如实落盘；批量/样式类诚实跳过并计数；不符即
  通道 A fail 并给出预期/实际示例（≤3 条）。旧证据卡无 opsJson →「声明比对
  不适用」，宁漏勿误。
- **接线**：GaeaVerifyRecord xlsx_apply 重算零错误后追加引用级复核，fail
  前缀升级整条通道 A；全簿公式评估错误（含断链 #REF! 求值）仍由 recalc
  承担，本层不做字符串级全簿扫描（防误报）。
- 验证：Go 全量绿（+9 测试：声明通过 / 值不符 / 公式不符 / 数值归一 / 替换
  通过与不符 / 旧卡不适用 / 批量跳过 / 等值与归一单测）；前端零改动
  （vitest 818/818 沿用——本轮出现 1 例 ProgrammingPage 负载型超时，系既有
  flaky 先例，单跑 12/12 绿，与改动无关）；tsc/eslint 0；绑定面 540 不变
  （drift PASS）。
- 勘误（欠账清单对账）：v4.9.0 清单中「锚点策略刻度对齐」已在 90ab160 交付
  （阈值 0-1 标尺化：weight≥0.9 / selfRelevance≥0.8 等），一并移除。

## v4.10.0 · 多跳因果链 (2026-08-30)
> v4.9.0「跨事实因果推断」纵深：证据从单跳升级为 ≤2 跳因果链（一因之因）。
- **因果链收集**：buildCausalChains 以实体为起点在 KG「导致」边上有界 DFS
  （双向：溯源找根源 + 顺藤找后果），收集 ≥2 边的链并按因果序渲染
  「A → 导致 → B → 导致 → X」；防环双保险（路径内边不重走 + 链内节点不
  重访）+ 链数封顶 4 条；被链覆盖的单跳三元组去重不重复出现。
- **证据面次序**：链优先 → 未覆盖单跳 → event_chain 关联（总上限 8 行不变）；
  系统提示词补「记忆链 = 跨多步因果链，解释时可串成完整故事」。
- 验证：Go 全量绿（新增 4 测试：两跳链 / 环路终止 / 深度上限 / 链数封顶；
  另修两跳测试的种子三元组重复）；vitest 818/818 沿用（前端零改动）；
  tsc/eslint 0；绑定面 540 不变（drift PASS）。
- 说明：跳数上限 2 是诚实边界——子串式实体匹配下更深链路噪声放大（远因
  关联度骤降），语义锚定 + 置信衰减的多跳图推理留后续设计。

## v4.9.0 · 跨事实因果推断「为什么」 (2026-08-30)
> 审计 §C「推理仅邻接遍历」深度补口：不只展示因果边，还能解释因果链。
- **GaeaWhisperCausalExplain(entity, personalityID)**（绑定面 539→540，
  MemoryB/play）：用户问「为什么<entity>」时，确定性收集证据——KG「导致」
  三元组（实体出现在因/果侧）+ event_chain 关联（涉及实体的事实对，各截断、
  上限 8 条），用当前人格口吻（Label + VoiceGuide）让 LLM 解释因果链（只用
  证据、不编造、证据不足诚实说明、≤200 字）；无证据时不调 LLM，直接返回诚实
  回退文案。5 个 Go 测试（有证据 / 无证据回退 / 空实体 / 证据构建 / 无关实体
  无证据）。
- **前端**：图谱面板新增「解释因果」按钮（查询后可用，琥珀色），解释结果展示
  在图形下方；vitest +1。
- 验证：Go 全量绿；vitest **818/818**（+1）；tsc/eslint 0；绑定面漂移
  PASS（540）。
- 说明：证据收集为确定性规则，LLM 只负责把证据讲成人话——不编造由 prompt
  约束 + 无证据零调用双保险；深度「因果图推理」（多跳推导）仍列后续。

## v4.9.0 · 首页重构（AI 多功能平台 · 星枢指挥所）+ 启动动画 (2026-08-30)
> 启动默认首页 + 两段式启动动画 + 未来感 AI 多功能平台首页（ui-ux-pro-max /
> design-taste-frontend 技能驱动，沿用 3.0「星枢 Constellation OS」令牌体系，
> 零硬编码色值）。
- **启动默认首页**：MainLayout 每次启动从首页（manifest.isHome）落地，不再恢复
  上次页面；会话内跨空间切换仍按空间恢复最后页面（原持久化机制保留）。
- **启动动画（两段式）**：index.html 静态启动屏（JS 加载期遮罩，纯 CSS）→
  BootSplash React 组件接管（旋转光环 + gaea 徽记 + 分步状态文案：核心→引擎→
  记忆→就绪 + 实时进度条）；进度/卸载全部由 timer 驱动，WebView2 rAF 节流
  （gaea-raf-degraded）与 prefers-reduced-motion 全降级。
- **首页重构**：Hero 中轴（公告 pill + 巨幅标题 + 副标题 + 中央命令条
  [AI 内核 orb / 打字 / 语音 / 发送 / ⌘K] + 语音状态行 + 快捷 chips）+ AI 状态
  细条（真实遥测 4 列）+ 能力矩阵 Bento（办公 4×2 旗舰大卡 + 板块瓦片，
  manifest 驱动，断点 6/4/2/1 列；旗舰卡 `.ml-bento.ml-bento--featured` 双类
  提权防同特异性覆盖）+ 门廊（编程独立窗口 / 设置）+ 底部信息条（会话 / 记忆 /
  系统）；全部交互元素补齐焦点环（2px 主色 outline，与 v3 壳层一致）。
- **i18n**：en/zh/zh-TW 三语同步新增 home.* 与 boot.* 共 17 键（DictKey
  编译期锁死，三语缺一即构建失败）。
- 验证：tsc / vite build 0 错误；eslint 0；vitest **809/809**（148 文件全绿；
  曾现 2 例 jsdom 并行环境抖动，单独/复跑均绿，与改动无关）；12 主题浅色抽查
  对比度正常；桌面 6 列 Bento 跨度断言正确；wails build 产物 gaea.exe 冒烟
  启动正常。
- 说明：首页视觉为「AI 多功能平台」范式（对标 Poe / Perplexity / DeepSeek
  官网），语音晶核收进命令条 orb（呼吸 / 波纹 / 辉光三态）；设计规格已同步
  design-system/gaea/pages/home.md（v3）。

## v4.9.0 · 关联入图（event_chain 因果链可见） (2026-08-30)
> 审计 §C「推理仅邻接遍历」补口：记忆关联（含 event_chain）此前只活在索引里，
> 图谱面板不可见——数据已有，只差展示。
- **子图并入关联边**：GaeaWhisperGraphSubgraph 在 KG 子图基础上并入 AssocIndex
  关联——事实 Subject 映射为实体节点，关联类型映射中文标签（event_chain→因果、
  temporal→时间、entity→同实体、emotion_peak→情绪相似、self_reference→自我、
  thematic→主题），边权重 = strength；只并入至少一端已在子图内的关联（保持以
  查询实体为中心），与 KG 边去重。只读、无副作用、绑定面不变。
- **前端**：因果关联边用琥珀色虚线描边（与普通/情绪边区分），图例补「因果」。
- 验证：Go 全量绿（新增 2 测试：关联并入 / 不连通跳过）；vitest **817/817**（+1）；
  tsc/eslint 0；绑定面 539 不变（GaeaWhisperGraphSubgraph 签名未变）。
- 说明：关联数据源 = 记忆整合（LLM consolidation 的 event_chain）+ 冷启动启发式
  （同子类标 event_chain，语义近似因果前后）；LLM 深度跨事实因果推断仍留后续。

## v4.9.0 · 图谱因果维度 (2026-08-30)
> 审计 §C 欠账收口：「图谱上无因果维度」——确定性因果模式入图，无需 LLM。
- **因果三元组提取**：extractCausalTriples 从事实摘要提取 {因, 导致, 果}——
  因为/由于…所以…、X导致/引发/造成Y、X让我/使我Y 四类模式（正则锁定 + 测试），
  直接式匹配时剥离「因为/由于」前缀；情绪经 attachEmotion 随事实落图。
- **接线**：ingest 逐事实提取（extractTriples）与文档导入（fact_landing）两条
  路径都产因果边；图谱面板无需改动（「导致」谓词天然渲染 + 情绪着色已生效）。
- 验证：Go 全量绿（新增 6 测试：因为所以 / 直接式剥离 / 让我式 / 无模式 / 情绪
  携带 / 摄入链路）；vitest 816/816 沿用（前端零改动）；tsc/eslint 0；绑定面
  539 不变。tasks 压力测试在全量并发下再现已知 flake（CI 重试语义，单跑绿）。
- 说明：本版为确定性启发式因果；LLM 深度因果抽取（跨事实推断「为什么」）留
  后续设计；关联表 event_chain 已存在但图谱面板未展示，列观察项。

## v4.9.0 · 图谱情绪维度（轻语关系图谱做深） (2026-08-30)
> 审计 §C 欠账收口：「三元组主语几乎全为「用户」+ 情绪活在图外 EmotionState」。
- **三元组情绪维度**：Triple 增 EmotionLabel / EmotionalIntensity / Valence；事实
  提取时经 attachEmotion 把情感快照落进三元组（效价 >0.15 正面 / <-0.15 负面 /
  其余中性，确定性派生）；新增 AddTriple 保留情绪（Add 兼容旧调用）；子图边带
  EmotionLabel 供前端着色。
- **主语实体化**：BASIC_PROFILE 不再硬编码「用户」→ 用档案键（如「生日」）作
  主语、谓词「属性」；关系类（赞赏/表达脆弱/关系）保持「用户」主语语义。
- **持久化**：whisper hermes.db 迁移链 V13→V14（knowledge_triples 增 3 列，
  ALTER ADD COLUMN 带默认值，存量安全）；repo 读写带情绪；schema_v13 测试版本
  断言同步 V13→V14。
- **前端**：图谱边按情绪着色（正面绿 / 负面红 / 中性灰）+ 情绪图例；
  WhisperGraphEdge 增 emotionLabel。
- 验证：Go 全量绿（新增 5 测试：实体化+情绪 / 负面标签 / AddTriple 保留 / 子图
  边情绪 / 落库读回）；vitest **816/816**（+1 图谱情绪）；tsc/eslint 0；绑定面
  539 不变。全量并发下 tasks 压力测试偶发 flake（CI 同款重试语义，单跑绿）。
- 欠账延续：因果维度（「因为…所以…」边）需 LLM/事件链，未做；时空维度暂以
  事实 CreatedAt 承担，未建专门时间索引。

## v4.9.0 · 记忆重述 + 锚点刻度对齐 (2026-08-30)
> 记忆回放收尾两刀：LLM 叙事重述 + 锚点策略阈值刻度修正。
- **GaeaWhisperMemoryRetell（绑定面 538→539，MemoryB/play）**：让 gaea 以当前
  人格口吻把情节/锚点记忆「重述成故事」——输入复用确定性回放（摘要 + 情绪 +
  原文对话），系统提示要求第一人称、称对方「你」、≤300 字、不复述字段名、结尾
  带当下感受；模型绑定沿用轻语 chat 功能绑定（引擎经 opts 显式传入）。4 个 Go
  测试（episode / anchor / 未知类型 / 上下文组装）。
- **锚点策略刻度对齐（偏离 ackem 原值，已注释说明）**：factExtractionPrompt 的
  weight/selfRelevance 标尺是 0-1，原阈值（weight≥2、selfRelevance≥4.0/4.5）
  在标尺上不可达，导致里程碑/关系分支永不触发。对齐为 0.9/0.8/0.9（0-1 标尺
  语义等价），补策略测试 2 组；旧夹具按新语义修正。
- **前端**：情节/纪念日弹窗新增「让 gaea 重述这段记忆」按钮（加载/失败/叙事块
  三态），MemoryRetell 组件两处共用。vitest +2（episode / anchor 重述）。
- 验证：Go 全量绿；vitest **815/815**（+2）；tsc/eslint 0；绑定面漂移
  PASS（539）。

## v4.9.0 · 时间锚点「重访那一天」 (2026-08-30)
> 记忆回放续刀（审计 §C 欠账收口）：锚点策略接线 + 纪念日入口 + 锚点→情节回放。
- **写路径接线（审计同类骨架欠账：ShouldWriteTemporalAnchor/BuildTemporalAnchor
  有定义无生产调用）**：MemoryWritePayload/IngestTurnArgs 增 TemporalAnchorSink →
  extractFactsViaLLM 逐事实评估（IsNew = FactStore 前后计数差，去重不重复锚点）→
  Orchestrator.AddTemporalAnchor（持回合锁，防与状态快照竞态）落
  State.TemporalAnchors 随 companion_state 持久化。命中条件沿用 ackem 策略
  （周期纪念日/里程碑/关系/高情绪）。
- **读路径**：GaeaWhisperAnchors（锚点列表，日期降序）+ GaeaWhisperAnchorReplay
  （锚点 → linked 事实的 (session, turn) → 覆盖情节 → buildEpisodeReplay 重建
  原始对话；未命中情节 Replayable=false 回退锚点摘要 + 事实摘要）。绑定面
  536→538，MemoryB/play。
- **前端**：记忆库新增「纪念日」tab（日期 + 类型徽标 + 情绪）；点击打开「纪念日
  回放」弹窗（锚点摘要 + 关联事实 chips + 回放气泡，与情节回放共用
  ReplayDialogue 组件）。
- 验证：Go 全量绿（新增 8 测试：写路径 4 + 读路径 4）；vitest **813/813**（+2）；
  tsc/eslint 0；绑定面漂移 PASS（538）。
- 观察项/欠账：锚点策略阈值（weight≥2、selfRelevance≥4）与 LLM 抽取标尺（0-1）
  刻度不一致，实际常命中「周期纪念日」分支，刻度对齐留后续决策；「重访」目前为
  确定性原文回放，LLM 重述（把摘要讲成故事）未做。

## v4.9.0 · 轻语记忆回放 (2026-08-30)
> 审计 §C 乐园做深欠账收口（「记忆回放」零代码 → 确定性重建原始对话）。
- **后端 GaeaWhisperEpisodeReplay**（绑定面 535→536，MemoryB/play）：按情节 ID
  从 hermes.db 读情节，再按 SourceSessionID + [StartTurn, EndTurn] 从
  chat_history 重建原始对话——纯确定性、不调 LLM；过旧情节因 chat_history
  裁剪（最近 2000 行）无原始对话时 Replayable=false 回退为仅摘要。3 个 Go 测试
  （轮次范围重建 / 无历史回退 / 未找到与空 ID）。
- **前端记忆库情节弹窗「回放原始对话」**：用户/gaea 对话气泡 + 轮次标注，
  加载/失败/不可回放三态；WhisperMemoryLibrary +2 vitest（回放渲染/摘要回退）。
- 验证：Go 全量绿（whisper 包在全量并发下偶发环境挂起，单跑 1.5s 绿——CI 同款
  flaky 重试语义）；vitest **811/811**（148 文件）；tsc/eslint 0；绑定面漂移
  PASS（536）。
- 欠账延续：「重访雨夜」的时间锚点入口（anchor→episode 索引）与图谱情感/因果
  维度仍列后续；本版先把「原始对话可回放」落地。

## v4.9.0 · 构建冒烟自动化 (2026-08-30)
> v4.8.3 教训收口（构建链路，不增绑定）：build.bat 不再无条件打印成功块。
- **build.bat 真实退出码 + 产物新鲜度守卫**：构建前删除旧产物（被常驻实例
  锁定时显式报错），`call wails build` 失败即停（errorlevel 检查），构建后
  必须重新生成 build\bin\gaea.exe 否则报错——「判断构建成败唯一可信标准 =
  真实退出码 + 新产物」。
- **自动冒烟**：构建成功后默认复制到 .tmp\smoke-gaea.exe（临时副本，规避
  常驻实例/AV 锁）跑 scripts/smoke.ps1（127.0.0.1:18999 /api/health 200 +
  status=ok 响应体校验），失败即停并提示勿发布；`build.bat skip-smoke` 可
  跳过（快速迭代，发布前不得跳过）。
- **smoke.ps1 增强**：Start-Process 失败显式报错、响应体 JSON 校验
  status=ok、finally 判空回收。
- 验证：默认路径实跑（wails build 44.5s + 冒烟通过 + 冒烟进程回收）；失败
  路径用假 wails 桩实测 [FAIL] wails build failed + exit 1；桌面复制失败仅
  告警不阻断（SAC 可能拦截，产物仍在 build\bin）。

## v4.9.0 · 欠账收尾小步 (2026-08-30)
> v4.8.3 发布后的代码级欠账收口（不增绑定，535 不变）：VoiceStart 门小修
> + 持久化原子写统一 + XlsxPreview 大表格虚拟滚动。Go 全量绿、vitest
> 809/809、tsc/eslint 0。
- **VoiceStart realtime 门小修**：端到端实时模式的回复走服务端 response
  事件（事件泵不经 whisperChatFn），VoiceStart 在 realtime 在位时不再要求
  whisper 对话回调——whisperChatFn=nil 也能启动；拼接管线双门（ASRReady +
  WhisperReady）逐字节保留；新增两回归测试。
- **持久化套件统一收尾**：desktop_session SaveModes 走 fileutil.AtomicWrite
  （临时文件 + rename，崩溃不留半截 JSON）；gaea/archive JSONL 单次 Write
  落整行（数据 + 换行同缓冲，消除双写撕裂窗）；新增 7 回归测试。
- **XlsxPreview 大表格行虚拟滚动**（观察项收账）：300 行以上只渲染可见窗口
  ±10 overscan + spacer 行保滚动条总高，冻结行常驻；后端预览上限
  2000×100 不再整表 20 万 td；小表全量渲染逐字节不变；2 新 vitest。

## v4.8.3「微信图片双向」(2026-08-30)
> v4.8.2 发布当日真机实测复盘 + 微信图片双向真协议实装。零新增绑定（535
> 不变），协议经三方印证（本机抓包实测解密 + hermes-agent weixin.py 生产
> 实现 + openilink SDK 导出符号）。Go 全量绿、前端零改动（vitest 807/807
> 沿用）。详见 releases/v4.8.3.md。
- **识图排障（慢+乱字母复盘）**：身份类问题（你是谁/你会什么…）跳过联网
  搜索——「你是谁」误触发搜索把无关英文网页摘要注入提示词，角色扮演模型
  把片段混进回复（乱字母根因）；WhisperChat 日志预览按字节切汉字出
  \xe6\x81 伪影修复（rune 化）。
- **微信出图回推（真机 delivered）**：getuploadurl（filekey/aeskey/md5/
  PKCS7 filesize）→ CDN 密文直传（x-encrypted-param 票据）→ sendmessage
  image_item 图片卡片 + caption 独立补发；任何失败降级文本卡片（逐字节
  不变）。真机验证 stage=delivered。
- **微信发图识别（真机两连发通过）**：入站图片 type=2 + aeskey/
  media{full_url, encrypt_query_param, aes_key} 防御解析；DownloadImage
  Encrypted（dial-time SSRF/20MiB → AES-128-ECB 解密 → 魔数终审才落盘）；
  file:// 分支限 TempDir。
- **识图模型升级**：多模态 Qwen3.6-35B 主模型优先（真机探针实测视觉链路
  完好；手写体显著强于 OCR 专线；与聊天同体常驻显存零额外开销），
  PaddleOCR → MinerU → OvisOCR2 三级链降为兜底。
- **关键坑锁死**：入站图片 type=2（非 3）；aes_key 必须 base64(hex字符串)
  （base64 原始字节 = 接收端灰框）；上传域与扫码 baseurl/redirect_host
  无关（无需重扫）；media_crypt.go 手写 AES-128-ECB + PKCS7（Go 无 ECB）。
- **抓包基建留存**：wx_capture.jsonl（qr_status/inbound_media/upload_probe
  三类）——探针时代产物，排障可用。

## v4.8.2「欠账收尾」(2026-08-30)
> v4.8.1 欠账收口：权限升级请求（v3.7.0 挂账独立一刀）+ 竞态/flake 全治理
> （含 Cancel 被 succeeded 吞掉的真生产竞态）+ Realtime S2 事件环骨架
> （16k→24k 重采样/TurnControl/事件泵/五重降级护栏，真机欠账如实记账）。
> Go 全量绿、vitest 807/807、tsc/eslint 0、drift PASS。详见 releases/v4.8.2.md。
- **权限升级请求**：request_permission 工具（reason 必填/headless 降级）+
  硬纪律三闸（deny 先行/hardAsk 拒升级/批准只写 glob 规则表不绕闸门）+
  五决策接线 + 审批卡 request 形态（reason 原文块）+ 会话 glob 规则表补全。
- **竞态/flake 治理**：Cancel 收尾窗竞态修复（×10 压力绿）+ stubGate 测试桩
  加锁 + filewatch 风暴测试时序根治（全量实战零 FAIL）+ ProgrammingPage
  显式 5s 超时。
- **Realtime S2**：Resample16kTo24k 纯函数（0.0077% 误差）+ 事件常量 +7 +
  TurnControl 可选接口 + voice_manager 事件泵（barge-in 三联/24k WAV 冲洗/
  降级拼接）+ 前端 PCM 推送死门打通；未配置=逐字节零变化（守护测试）。

## v4.8.1「欠账清尾」(2026-08-30)
> v4.8.0 欠账两刀收口：全局离线模式设置 UI（绑定面 533→535）+ Realtime S1
> （配置落盘 + DPAPI key + VoiceSettingsPanel 入口）。Go 全量绿、vitest
> 803/803、tsc/eslint 0、drift PASS。详见 releases/v4.8.1.md。
- **离线模式设置 UI**：SecurityPanel「全局离线模式」总闸段（回填/即存/失败
  回滚）+ GaeaGet/SetOfflineMode 绑定 + shelf 内存同步。
- **Realtime S1**：realtime 三键落盘（provider 仅 openai/Key 走 DPAPI 密文）+
  initVoice 注入（接线位置测试守护）+ realtimeReady = 配置且构造成功 +
  VoiceApplySettings/GetSettings 扩三键（明文 Key 永不出后端）+ 面板入口段。

## v4.8.0「全面铺开 · 触点纵深」(2026-08-30)
> 六线并行调研（多子代理分工、文件足迹不相交）→ 七刀实现。意图内核纵深
> （读屏多显示器/LLM 兜底/生图产物回推）+ 微信通道离线收敛 + 全局离线模式
> 总开关 + 成本知识图谱可视化（绑定面 532→533）+ 实时语音 Realtime S0 铺底。
> Go 全量绿、vitest 800/800（146 文件）、tsc/eslint 0、drift PASS。详见
> releases/v4.8.0.md。
- **读屏纵深**：`screen.Monitors()`/`CaptureArea` 多显示器枚举；intent 序数
  解析「第N(块)屏/主屏/副屏」（动词锚定窄规则，越界诚实报错）；OCR 文本
  >300 字本地摘要朗读（只走 Herdsman，失败退截断）；截图留档默认关。
- **intent LLM 兜底**（默认关）：规则未命中的受控冷路径——白名单
  navigate/status/read_screen + 0.75 置信门 + 2s 硬超时 + manifest 校验；
  dryRun 恒不调用；命中复用既有执行层。
- **生图 CardPath 接通**：勘误「生图异步」——实为同步阻塞，首图 FilePath
  即 CardPath；微信入口「（产物：路径）」从此有真实数据。
- **iLink 离线收敛**：per-peer 限频 20 条/分 + 4KB 截断 + 多媒体上限 5；
  图片→vision 识别管线（SSRF/20MiB/魔数三重防线，OCR 注入式接线）；防御
  解析矩阵（多态 JSON 降级不炸整批）；SendFileCard seam + 真机抓包清单。
- **全局离线模式**（跨版欠账清账，默认关）：`EngineType.IsLocal()` +
  routeModel 三步云过滤；无本地可用走既有「模型不可用」降级。
- **成本知识图谱**：costref.BuildGraph 纯函数组图器（7 节点/6 边、树聚合/
  条目展开双视角、EntryName 精确优先、截断与悬挂边防护）；CostGraphView
  零依赖 SVG 双视角 + Modal 明细；成本库第 8 模块。
- **Realtime S0**：internal/realtime seam（RealtimeSession/注册表/openai 实现）
  + VoiceHealth realtimeReady + 优雅降级；14 离线测试；S1/S2 留欠账。

## v4.7.0「命令面板接内核 · 读屏」(2026-08-30)
> 路线图 §10.4a S4.6 完整收口：桌面命令面板接统一意图路由内核（语音/微信之后
> 的第三个入口）+ 屏幕感知能力（读一下屏幕）纳入能力面。GaeaRouteIntent(text,
> dryRun) 绑定（531→532，dry-run 预览-确认制防搜索词误触发）+ SearchModal 指令
> 预览卡 + 真·Ctrl+K（办公板块让位工作台面板）+ read_screen 三入口免费受益
> （截屏→OCR→TTS 朗读/内联回执，临时文件即用即删）。Go 全量绿、vitest 796/796
> （145 文件）、tsc/eslint 0。详见 releases/v4.7.0.md。
- **GaeaRouteIntent**：dryRun=true 零副作用预览（校验口径与执行层一致——未知
  板块/媒体域缺失按未命中）；false 真执行。S4.6 显式豁免旧「零新增绑定」纪律
  （面板是前端入口，绑定即其回传通道，intent_router.go 头注已记录）。
- **预览-确认制**：SearchModal 命中出「指令」卡（动作标签 + 预览语 + 执行按钮），
  点执行才真跑；执行回执内联，导航类 emit gaea-intent-navigate 复用 S4.4 切板块
  后收面板；未命中检索行为零变化。
- **真·Ctrl+K**：MainLayout 全局快捷键落地（tooltip 此前名不副实）；gaea 工作台
  内让位自有 CommandPalette 防双面板。
- **读一下屏幕**：intent.ActionReadScreen 窄规则（读/念/看/识别+屏幕、屏幕上有
  什么、读屏——不含裸读/看）+ execReadScreen（screen.Capture → 临时 PNG →
  GaeaOCRText 既有 OCR 链 → 300 字截断回传）；语音经 TTS 朗读、面板内联展示、
  微信走文本回推；失败诚实回执不坠聊天。

## v4.6.1「微信统一路由 · 规范包机制 · 归因对标」(2026-08-30)
> 审计补课续刀：S4.5 微信消息接统一路由（routeIntentWithResult 产物感知版本，
> 提醒特例之外 navigate/生图/状态/提醒全命中即执行，未命中才走聊天）+ iLink
> 图片消息协议第一刀（image_item/file_item 防御性解析 + 非文本转模型提示行，
> 未知空项宁漏勿误）；规范包机制化（Checker 注册表 + 红头/造价工程表式双检查
> 器，GaeaDocumentLint 聚合，OfficePanel 按规范包分组）；成本归因对标（明细
> vs 参考指标 P25/P75 带宽，差幅等级/贡献金额/主因 TopDrivers，参考池排除
> 本项目，绑定面 530→531，FiveCalcPanel 归因区）。Go 全量绿、vitest 791/791、
> tsc/eslint 0。详见 releases/v4.6.1.md。

## v4.6.0「双空间收尾 · 纵深补课」(2026-08-30)
> 执行审计（docs/audit-2026-08-30-v4-execution-review.md）后的第一轮补课：
> 红线缺口三条（记忆注入跨空间 / 任务分账未启用 / 事件过滤仅 1 处）全部接线，
> 前端治理收尾（keepAlive 裸轮询 8 处门控 + CSS 真硬编码 token 化），C 类纵深
> 按价值排序落地三件——Mood→TTS 闭环 / Verifier 通道 B 真视觉 diff + 失败回
> Plan / 询价异常检测 + 价格预测 + OCR 报价单自动入询价库。每刀带「纵深检查」。
> 验证：Go 全量绿（无 FAIL）；vitest **791/791**（144 文件）；tsc/eslint 0；
> 绑定面不变（变参向后兼容）。详见 releases/v4.6.0.md。
- **红线 ①记忆注入按空间收窄**：`boot/sysprompt.go` + `controller_memory.go`
  `refreshMemoryLocked` 传 `Options.Space` → `InSpace` 读端视图——work 会话只
  注入 work 记忆、play 只注入 play；mode=off 旧行为零变化。
- **红线 ②任务分账生产启用**：`[tasks]` 配置段（max_concurrent/per_space/
  priority）→ `startTaskScheduler` 落默认 {work=1, play=1} + 价格抓取优先；
  显式空表可关分账回退全局 sem。
- **红线 ③事件过滤推广**：`onTaskEvent(cb, space?)` 订阅层过滤，任务中心/
  运行角标/价格源/索引面板全按 work；MainLayout 主事件流走 subscribeForSpace。
- **keepAlive 轮询门控**：TaskCenter/SubagentsPanel/FeatureModelBar/
  ProgrammingPage/BenchmarkSection/useStatsState/useImageGenQueue/useBridgeWatch
  八处裸轮询接入 usePollingGate（后台空转归零）。
- **Mood→TTS 闭环**：长期心境 4D EWMA → 连续韵律中文指令（低沉/温暖/不安/
  平缓/轻快…），中性轮次由心境主导「听得出她今天低落」，强情绪标签仍主导。
- **Verifier 通道 B 真 diff**：soffice 转 PDF + pdftoppm 逐页渲染 + 纯 Go 像素
  差异率（页数联合判定 pass/warn/fail），审计产物落 journal/verify/<id>/；
  失败回 Plan：证据卡内联结论 + xlsx_apply 一键「重新规划」。
- **询价飞轮反向 + 异常检测 + 预测**：OCR/图片报价单确认导入自动幂等写入询价库
  （source=OCR报价）；调差建议带 正常/关注/异常 分级；同标题询价序列线性回归
  预测下期价。
- **欠账清单**（如实）：规范包机制化 / 成本知识图谱+归因 排下轮；生命库可写化
  评估结论 = 不做盲写 Herdsman 库（锁/Schema/竞争三类风险），gaea 侧角色资产
  表另案评审。

## v4.5.0「指令中枢」· 统一意图路由内核 + 语音指令 (2026-08-30)
> 路线图 §10.4a（2026-08-30 规划修订插入）第一刀：落地触点层「同内核多入口」
> 的架构承诺——一层「意图 → 能力 → 结果回传」的统一路由内核，语音 / 微信 /
> 命令面板共用；语音指令（JARVIS 一档）首发接线。「打开绘梦」「画一张赛博朋克
> 城市」「现在用什么模型」「提醒我 30分钟后 喝水」——任何模态，唤起同一个 gaea。
> 验证：Go **108/108 包**；vitest **789/789**；tsc/eslint 0；绑定面 **530 方法**
> 零新增（执行结果走事件 + TTS 回传，漂移防线不动）。详见 releases/v4.5.0.md。
- **intent 解析包（S4.1）**：纯函数规则引擎（导航/生图/状态/提醒四类意图 +
  板块别名表贪婪匹配）；「宁漏勿误」纪律——「画得不错」「画了半天」绝不触发生图；
  LLM 兜底分类器留 Capability 接口位。
- **能力执行层（S4.2）**：`routeIntent` 统一入口——navigate（manifest 校验 +
  `gaea-intent-navigate` 事件）/ generate_image（mediaState 自由生图）/ status
  （引擎摘要）/ reminder（复用离线代办解析与持久化）。
- **语音通路（S4.3）**：voice 对话回调内先过路由——命中即能力执行、回复经同一
  TTS 流程播报，未命中透传原轻语对话；voice 包零改动。
- **前端导航（S4.4）**：MainLayout 订阅 `INTENT_NAVIGATE` → `navigateBoard`
  （语音「打开绘梦」自动切空间，复用 v4.3.2c 机制）。

## v4.4.0「触点」· 微信遥控器一期：离线代办 (2026-08-30)
> 路线图 §10.4 v4.4「触点」第一刀：微信从「能聊天」升级为「能接活的遥控器」。
> 在微信里对助手说「提醒我 …」→ 桌面端中文时间解析建提醒 → 到点经微信回推——
> 官方元宝做不了的桌面端「离线代办」差异化主打。WeixinPage 书房板块页落地。
> 验证：Go **107/107 包**；vitest **789/789**；tsc/eslint 0；绑定面 **530 方法**
> 漂移 PASS（+5）；spaceBindings **247 方法**全覆盖；版本五处统一 4.4.0。
> 详见 releases/v4.4.0.md。
- **微信主动推送通路**：`weixin.Server` 记录最近活跃会话（fromUser/contextToken），
  新增 `Push` 向其回推文本（httptest 校验目标与 item）。
- **离线代办提醒域**：中文时间解析（相对时长 / 日期前缀+段词 / 裸时刻；中文数字
  含「十」进位；无段词按字面，确认文案带完整时间供纠正）→ `wxReminder` JSON
  持久化（重启恢复）→ 微信消息任务路由（提醒类接管，失败回格式提示）→
  20s ticker 到点回推（失败重试 ≤5 次标 failed）。
- **WeixinPage 落地（书房板块）**：扫码绑定流（QR 轮询 / 配对码 / confirmed 落
  WhisperAssistantSave 自动重拉通道）+ 通道状态徽标（运行/过期/未绑定）+ 提醒
  列表（手动新建 / 删除 / 回推开关）+ 指令说明。weixin 板块 inMenu=true 进
  rail 与首页左翼书房格；`WhisperWeixin*`/`WhisperAssistant*`/`WeixinReminder*`
  共 12 个绑定自 LegacySurface 转正或新增，spaceBindings 全归 work（235→247）。

## v4.3.2「双翼·中庭」· 首页重构 + 空间导航收敛 (2026-08-30)
> 首页体验重构：双翼·中庭三区布局（左书房 / 中语音对话 / 右庭院）+ 空间导航
> 收敛（移除一级导航空间切换，首页双翼即空间入口，按板块自动切空间）。
> 设计遵循 design-taste-frontend（不对称 / 磁吸核心 / 避免等宽栏）；沿用星枢
> 深空玻璃拟态 + gaea-glow。验证：Go 全量绿；vitest **789/789**；tsc/eslint 0；
> 版本五处统一 4.3.2。详见 releases/v4.3.2.md。
- **首页「双翼·中庭」**：中庭 = 语音 + 打字一体对话条（VoiceChatText 共用管道 +
  放大 orb 磁吸锚点，hero 让位细眉）；左翼「书房」2×2 紧凑格（办公/造价/记忆/模型）；
  右翼「庭院」纵向列表（聊天/小说/绘梦/角色）；门廊 = 编程（独立窗口徽标）+ 设置。
- **命名**：工位→**书房**、乐园→**庭院**（三语字典 zh/zh-TW/en 同步）。
- **空间导航收敛**：移除 rail 顶部空间切换按钮；`navigateBoard` 按板块
  manifest.space 自动切空间（书房板块→work / 庭院板块→play / 编程→independent）；
  rail 展示全部板块；搜索 scope 文案同步 书房/庭院。
- **桌面端**：gaea-v4.3.2.exe（33MB，SHA256 6a0486db）+ 冒烟 /api/health 200。

## v4.3.1「乐园」后续小步 (2026-08-30)
> v4.3 后续小步收口：主动关心定时推送频控 + 创作间世界模型面板 + 角色参考图 +
> 朗读情绪 UI。设计沿用 `docs/gaea-v43-play-deepen-design.md`（v4.3c/e/f/g 补完）。
> 验证：Go 全量 **118/118 包**；vitest **789/789**（144 文件，+20）；tsc -b / eslint 0；
> 绑定面 **525 方法**漂移 PASS（+3）；spaceBindings **235 方法**全覆盖断言；
> 版本五处统一 4.3.1。详见 releases/v4.3.1.md。
- **主动关心定时推送频控（v4.3c 补完）**：app 层 ticker 四信号评估（AttentionManager
  频控 ≤3 条/小时 → MatchHabits 作息尊重 → DetectSpecialDatesV2 生日祝福（每天首条、
  人格感知提示词）→ 门控+合成器）→ `gaea-whisper-proactive` 事件推前端；
  新绑定 `GaeaWhisperProactiveConfig/SetProactiveConfig`（开关/上限/间隔/免打扰时窗）；
  前端 WhisperGraphPanel 订阅显示推送气泡（含 birthday 徽标）。play 红线零落盘。
- **创作间世界模型面板（v4.3e/f 落地）**：设定页「维度化」模式（6 维度卡片就地编辑
  整存）+ 伏笔登记表面板（状态流转/回收率）+ 一致性检查面板（三类规则告警/重新检查）。
- **角色参考图 + 生图参考槽（v4.3g 补完）**：characterlib SchemaV2 迁移（reference/
  gallery_images 两列幂等）+ `CharacterGeneratePortraitWithRef`（img2img 参考槽
  denoise 0.55 + 模型门禁）+ 前端角色库参考图管理（以参考图生成剧照/删除/添加）。
- **文本朗读情绪 UI（v4.3d 收尾）**：朗读情绪选择器（9 标签对齐 EmotionVoiceMap）+
  会话情绪自动跟随 + 朗读携带 `TTSSpeakBase64WithParams` 情绪参数（无情绪回退原路径）。

## v4.3.0「乐园」娱乐做深 (2026-08-29)
> 阶段 3+ 领域包第二发：会客厅关系记忆 + 主动关心 + 情感语音；创作间图文联动。
> 设计 `docs/gaea-v43-play-deepen-design.md`（4 份只读调研：后端骨架约 70% 已存在，
> 本版以「接线 + 参数扩展」为主，与工位零交叉）。
- **会客厅·关系记忆**：`memory_associations`/`user_habits`/`temporal_anchors` 三表
  补 repo 闭环（此前有 schema 无 repos，重启关联全空）+ `ReseedAssociationGraph` 打通
  + hermes.db 外键延迟检查实证修复；`QuerySubgraph` 多跳邻接子图召回；前端
  WhisperGraphPanel（零依赖 SVG 环形邻接图、节点点击重查）。
- **会客厅·主动关心**：`GaeaWhisperProactiveNow` 评估绑定（门控+合成器现成复用，
  时段感知）+ 前端「轻语先开口」按钮（类型徽标/提示词）；定时推送留后续小步。
- **会客厅·情感语音**：`TTSProvider.SynthesizeWithParams` 参数扩展（speed/style/
  emotion；cosyvoice 工厂不再丢弃 voiceDescription；edge SSML 参数化）；
  情绪→参数映射 `GetEmotionVoiceParams`；长期心境维 `EmotionState.Mood`（EWMA α=0.01
  持久化）——「听得出她今天低落」原料就绪；`TTSSpeakBase64WithParams` 绑定。
- **创作间·图文联动**：章节配图复活死绑定（ChapterPage「配图」按钮 +
  ChapterIllustration 弹窗）；书封生成 `GaeaGenerateBookCover`（3:4 落 play exports
  + NovelPage「生成封面」按钮；修 Windows 盘符卷文件名清洗 bug）。
- **验证**：Go 全量绿；vitest **769/769**（144 文件，+10）；tsc -b / eslint 0
  （顺带修 v4.2 遗留 TS2488/spaceBindings 键）；绑定面 **522 方法**漂移 PASS（+5）；
  spaceBindings **233 方法**全覆盖断言；版本五处统一 4.3.0。
  详见 releases/v4.3.0.md。

## v4.2.0「智慧」工位造价包 (2026-08-29)
> 阶段 3+ 领域包第一发：AI 组价 + 询价飞轮 + 五算对比（垂直蓝海率先变现）。
> 设计 `docs/gaea-v42-cost-ai-design.md`；「无确认不落库」纪律贯穿三支柱。
- **AI 组价**：`cost.PriceBand` 价格带推荐纯函数（P25/P50/P75 + 离散度 + 离群 +
  置信度 + 证据链 BandSource，分位数 R-7 与 costref 同口径）；`GaeaCostCompose`
  绑定（相似清单检索：关键词 + 本地 bge-m3 语义召回 + rerank 精排 → 价格带 →
  **LLM 人材机拆解**（`routeSensitiveLocal("office")` 敏感域本地化，失败规则降级）
  → 建议视图 + `GaeaCostComposeApply` 一键回写）；前端 ComposeModal（测算明细行
  「AI 组价」按钮：价格带卡 + 证据链 8 列表 + 人材机行编辑 → 应用为明细行）。
- **询价飞轮**：`costinquiry` 包四源归一数据点（信息价/OCR报价/供应商比价/手动
  询价，`cost_inquiry_records`）+ 到期预警（valid_until）+ 调差建议（标题归一化
  匹配，|差幅|>2%）；前端询价视图（数据点 CRUD + 预警横幅 + 一键更新成本库）。
- **五算对比**：`coststage` 包估/概/预/结/决阶段值（`cost_stage_values` UPSERT）+
  对比计算（环比/累计差）+ 偏差特征（正常/关注/异常三档 + 规则诊断文案）；前端
  FiveCalcPanel（项目详情五算区：输入保存 + 对比表着色 + 偏差卡片）。
- **验证**：Go 全量绿；vitest **759/759**（142 文件，+21）；tsc/eslint 0；
  绑定面 **517 方法**漂移 PASS（+11）；spaceBindings **229 方法**全覆盖断言；
  版本五处统一 4.2.0（app_info / wails.json / versioninfo.rc / package.json /
  package-lock）；CHANGELOG / README / releases / AGENTS / progress 同步。
  详见 releases/v4.2.0.md。

## v3.9.0「双空间壳 + 办公信任链」(2026-08-29)
> 阶段 2 双空间壳（S2.1–S2.3/S2.3b + 页面迁入 P1）+ v4.1 办公信任链（证据链→复核→
> 回滚→规范体检）一次收官；「审阅后」护城河从设计到端到端闭环。
- **双空间壳（阶段 2）**：S2.1 壳层两视图 + 空间切换持久化（`gaea.shell.space` /
  `gaea.shell.page.<space>`）+ 删旧 10 板块导航 + 双首页（工位任务工作台 / 乐园会客厅
  创作间）+ 事件订阅空间过滤 + Ctrl+K scope；S2.2 工作台 localStorage 空间分键
  （`gaea.work.*` 旧 key 只读迁移）+ keepAlive 跨空间卸载（性能门控）+ **i18n 决策**
  （壳层 chrome+设置三语，页面 zh-only，审计「诚实 zh-only」选项）+ 页面迁入 P1
  （chat→play 对话流）；S2.3 bridge 三门面（spaceBindings 214 方法显式分类，
  workApp/playApp/sharedApp 类型级红线）+ types 全量迁移（WireShape 55 别名 +
  typesGenerationCheck 漂移校验）。
- **151 hex token 化（S0.7 遗留）**：VoiceSettingsPanel 浅色主题不可读真 bug 修复；
  语义色全 token；图表/品牌/覆盖层显式 `hex-exempt`；eslint `local/no-raw-hex` 防回归。
- **v4.1 办公信任链**：证据链（ChangeRecord 原文摘要 8KB / ChangeLedger / JournalStore
  JSONL+markdown，play 不落盘；六类写盘工具 + xlsx_apply 接入）；Verifier 双通道
  （结构/引用完整性 + 基线 PDF 渲染对比）+ 基线快照回滚（手工编辑冲突保护零覆盖）；
  GB/T 9704 红头 7 要素规范体检（GaeaDocumentLint + OfficePanel 入口）；
  前端「证据」入口（DeliverablesPanel 复核/回滚）。
- **验证**：Go 全量绿；vitest **738/738**（139 文件）；tsc/eslint 0；vite build；
  绑定面 **506 方法**漂移 PASS（新增 4 绑定）；spaceBindings 218 全覆盖；
  版本五处统一 3.9.0（app_info / wails.json / versioninfo.rc / package.json /
  package-lock）；CHANGELOG / README / releases / AGENTS / progress 同步。
  详见 releases/v3.9.0.md。

## v3.8.0「双空间内核 · 工位/乐园 + 质量地基」(2026-08-29)
> 双空间（工作/娱乐隔离）从规划到内核落地：会话/记忆/任务/产物/模型/工具/权限/护栏全按
> 空间装配、互不干扰；同步完成审计 P0/P1 质量地基（并发/安全/编辑脊柱）与长期规划定稿。
- **双空间内核（阶段 1）**：SchemaV14（facts/tasks space_id，旧数据回填 work）+
  会话目录分区 `sessions/work|play` + 事件日志/checkpoint 空间自描述 + `space.mode` 开关；
  记忆写侧盖章（remember/dream 指纹含 space/审计加列/play notes 不写 work AGENTS.md）+
  读端隔离（GetInSpace/citations 限定/UnifiedSearch scope 四组）+ 前端 hub/面板 scope 切换；
  `[space_profiles]` 模型按空间 + 工具装配期过滤（work 33 / play 1 / shared 13 + MCP spec 层滤）；
  任务 per-space 并发/优先级防饥饿 + cron 显式 work；权限策略按空间（play 默认不弹审批卡）+
  hardAsk 参数化 + persist_allow 分段回写 + play 内容护栏（5 处生成点钳制）。
- **质量地基（阶段 0）**：S0.1 回合级并发加固（turnMu，临时 worktree 实证修复前必崩）；
  S0.2 Registry 锁 + 幽灵名修复；S0.3 gate 原子化（撕裂换闸）；S0.4 retry_until 门控
  （堵审批绕过 shell）；S0.6 **edit_file 工具层**（grep/edit_file/multi_edit/edit_lines/
  move_file 五工具全落地）；knowledge 索引缓存 / office 原子写 / secure 非 Win AES-GCM /
  tasks 输出 LRU；前端聊天 memo+尾部窗口 / keepAlive 轮询门控；CI 新增 `-race` 并发门禁。
- **长期规划**：`docs/gaea-nextgen-roadmap-2026.md` 定稿（双空间版本重定义 + 四层落地 +
  执行计划 + 二次审核缺口清单）；99 个过时文档归档 `docs/archive/`；**用户拍板：编程板块
  保持独立 DSH 窗口、不并入工位/不共享工具面（防工具膨胀）**。
- **验证**：Go 全量 **115 包** + vet；vitest 全绿；eslint 0/0；tsc 0；绑定面 **502 方法**
  漂移 PASS；版本五处统一 3.8.0（app_info / wails.json / versioninfo.rc / package.json /
  package-lock）；wails build + 冒烟。详见 releases/v3.8.0.md。

## v3.7.0「办公蒸馏 codex 收官 · 引用可追溯 + 审批决策族 + 输出事件化」(2026-08-29)
> 办公蒸馏 codex 清单第二/三刀收官（C1 方案模式已随 v3.6.0 回退）。
- **C2 记忆引用可追溯**：RecallBlock 注入行带引用键 `[MEM:name]` + 句末标注纪律 +
  陈旧记忆（90 天）时效提示；回合结束解析回传并 Touch 命中记忆（未知键静默）；
  前端 remarkMemCitations + MemCitationChip 点击弹层展示记忆详情/沉淀来源
  （零新增绑定面）。
- **C4 审批决策语义族**：拒绝三分（deny 继续 / abort 终止本轮）；审批等待超时
  （`[agent] approval_timeout_secs`，默认 0=等待）；「始终允许」（persist_allow）
  策略文件回写——`ToolName(subject)` 规则写入 `[permissions].allow`，重启不失，
  hardAsk 完全降级不回写；`GaeaApprove` 重构为决策串五值，审批卡快捷键 1-5。
- **C9 任务输出事件化**：输出变更/终态经 gaea-task 事件推送整尾回放（独立节流
  与进度互不挤占），任务中心输出 dock 事件即推，2s 轮询降级为兜底。
- **C5 上下文占用状态行**：RunStatus 窗口占用迷你进度条 + 百分比；≥75%「接近
  自动压缩」/≥90%「即将强制压缩」两档预警（对齐 80%/90% 引擎线）。
- **C6 项目说明文件**：单文件 32KB 注入预算（超限 UTF-8 边界截断留标记）+
  `.gaea/AGENTS.md` 子目录约定发现（更具体者后注入，自动进记忆面板可编辑）。
- **C3 自动做梦 no-op**：dream 输入 sha256 指纹，与上次成功处理的轮次一致直接
  跳过 LLM 提炼（指纹仅在完整处理后记录）；排查确认三层渐进披露已存在，LLM
  滚动摘要层列观察项。
- **验证**：Go 全量 **114/114 包** + vet；vitest **681/681（130 文件）**；eslint 0/0；
  tsc 0；绑定面 **499 方法**漂移 PASS；版本四处统一 3.7.0；wails build + 冒烟 200。
  详见 releases/v3.7.0.md。

## v3.6.0「办公文件编辑审阅制 · 本地优先 · 对话面减负」(2026-08-29)
> xlsx AI 编辑改两段式审阅（规划不落盘 → 批准才应用）+ 原生图表嵌入工作簿 +
> PDF 统一出口 + 办公功能级 AI 本地优先 + 运行中插话；回退方案模式 v1、
> 撤下任务目标/验收清单，用户消息 Codex 式收敛。
- **xlsx AI 编辑审阅制（Plan → Apply 两段式）**：`GaeaXlsxPlanEdit`（AI 操作集在
  原文件临时副本试运行 + 逐单元格 diff 变更清单，不落盘）→ 用户批准 →
  `GaeaXlsxApplyEdit`（excelize 执行 + LibreOffice 重算）；新增 `set_style`（叠加
  样式不丢现有填充色）/`merge_cells`/`unmerge_cells`/`set_col_width`；XlsxPreview
  规划审阅卡（对标 Copilot Plan/Show Changes 范式）。
- **xlsx 原生图表**：excelize 原生图表对象嵌入工作簿（Excel/WPS 打开即可见、可继续
  编辑，非图片截图），返回锚点 + 数据供前端迷你预览。
- **PDF 统一出口**：`GaeaConvertToPdf`（LibreOffice 无头转换 + 独立 UserInstallation
  profile 防锁冲突；md/markdown 经 docx 中转），顶栏「导出 PDF」同管线；缩略图/预览
  支持 pdf/docx 文本提取。
- **办公本地优先路由**：`routeOfficeLocal`——办公功能级 AI 调用（Word/Excel 编辑、
  资料摘要、知识导入、记忆整理）默认走本地 Herdsman（数据不出本机、省 token），
  不可用回退常规路由；聊天主 agent 不受影响；`GetOfficeLocal/SetOfficeLocal` +
  安全设置面板开关（默认开）。
- **运行中插话（GaeaSteer）**：运行中消息作为当前回合 guidance 注入（不开新回合、
  不打断工具执行），未运行走 GaeaSend 排队兜底；`event.Steer` → notice 回显。
- **对话面减负**：回退 C1 方案模式 v1（mode/shouldPlan/planGate、Ask.Plan 计划卡、
  composer 模式切换器、GaeaAgentMode/GaeaSetAgentMode、AutoPlan 配置与评分字段）；
  撤下任务目标/验收清单（GoalCard + GaeaRequirement 系 8 绑定）；用户消息 Codex 式
  无气泡 + Kimi Work 式超长消息折叠（240 字符 3 行截断）。
- **修复**：whisper 关机排水丢任务（pending WaitGroup 等到排队任务执行完）；
  trajectory/contextview 空切片 null 崩溃（绑定层 Empty* 非 nil 保证 + 前端归一化）；
  TrajectoryView 测试负载 flaky 加固（显式 5s 超时）。
- **验证**：Go 全量 **114/114 包** + vet；vitest **669/669（127 文件）**；eslint 0/0；
  tsc 0；绑定面 **499 方法**漂移 PASS；版本四处统一 3.6.0；wails build + 冒烟 200。
  详见 releases/v3.6.0.md。

## v3.5.0「办公对话区标签页 · dsh-context Go 移植」(2026-08-28)
> 对话窗口上方新增 [对话 | 轨迹 | 上下文] 三标签：上下文 = 逐请求上下文构成看板，
> 轨迹 = 对齐 DSH ui-trajectory 的扁平事件账本，Agent 网络 = 主 agent + 子代理树。
- **request_header 事件（日志不变量补位）**：`event.RequestHeader` + `request_header` 日志行，
  每次模型请求前记录实际 system prompt 与工具 schema——上下文/轨迹 system/tools 分类的
  精确数据源；旧日志无此事件按估算降级。
- **上下文标签（新包 internal/gaea/contextview + GaeaContextView）**：`FoldTimeline` 纯函数
  折叠——六分类当前组成（system/tools/user/inject/assistant/tool，usage 实际 promptTokens
  等比锚定，与顶栏 ContextBar 同源；`Referenced context:` 前缀拆 inject）+ 原生 SVG 趋势图
  （步数|轮次|全局|增量四钮，增量 Phase D；压缩 ✂；点击联动步骤详情卡：输入→回复/实际
  prompt/输出/缓存）+ 事件流（注入|压缩|剪枝筛选，压缩节点 gone + 负 delta + 归档）；
  7 个折叠黄金测试。
- **轨迹标签（新包 internal/gaea/trajectory + GaeaTrajectory）**：对齐 DSH ui-trajectory
  事件账本——扁平记录表（user/request-header/assistant/tool/compact/ask/approval，带
  ts/durationMs/step；header change 检测 initial|system|tools|system-and-tools；tool
  dispatch+result 按 ID 合并、parentId 嵌套根、running/error、截断、耗时；轮间压缩
  Between-turns 区段；turn-end 错误）；前端 TrajectoryView = Duration/Turns/Calls chips +
  搜索 + 类型徽标（ASSISTANT 紫/TOOL 橙/提问 深蓝/REQUEST HEADER/COMPACTION）+ 点击展开
  检查器；9 个折叠黄金测试 + 5 个 vitest。
- **Agent 网络（FoldAgentNetwork + GaeaAgentNetwork + AgentNetworkCard）**：主 agent 根 +
  子代理树（拥有子记录的元工具调用，不写死工具名），节点聚合子树工具数/错误/估算 token/
  状态 running|error|completed；subagents/ meta 任务摘要匹配富化状态与模型；前端 SVG 树 +
  节点 token 占比环 + running 绿脉冲 + 悬停详情条；3+3 测试。
- **随版并入（并行工作流）**：记忆统一层第二刀前端收尾（GaeaMemoryUnarchiveBatch /
  GaeaMemorySetRetentionDays 的 bridge 类型、批量恢复/保留期 UI 与 mock、生命周期测试补强）。
- **验证**：Go 全量 **114/114 包** + vet；vitest **668/668（127 文件）**；eslint 0/0；
  tsc 0；绑定面 **503 方法**漂移 PASS；版本四处统一 3.5.0；wails build + 冒烟 /api/health 200。
  详见 releases/v3.5.0.md。

## v3.4.0「记忆统一层第一刀 · 统一检索收口 + 生命周期产品化」(2026-08-27)
> 路线图 V4（记忆统一层）首发：hub 搜索 4 绑定前端拼装 → 1 绑定后端聚合（GaeaUnifiedSearch
> 增三脑/文件语义两组）；归档 tab 从「永远空白」到「分页可浏览 + 一键恢复」（新增
> GaeaMemoryUnarchive）；保留期下发展示；修复漂移脚本单条差异静默放行 bug。
- **统一检索后端收口**：`GaeaUnifiedSearch` 视图扩展四组——keyword（工作区全文）+
  semantic（跨库语义）+ **brain（三脑命中，新增，a.brain==nil 时空数组不报错）** +
  **files（文件语义，新增，复用 GaeaFileSemanticSearch 抽出的私有实现）**；hub 搜索
  （MemoryHubPage.runSearch）由「4 绑定 Promise.all 前端拼装」收敛为「单次 app.UnifiedSearch」，
  四组映射回原 HubSearchHit 渲染（徽标/预览/@ 引用零变化）；WorkspaceSearchPanel 跨库模式
  零改动，语义徽标补 file kind（后端本就返回，前端类型漏声明）。测试：Combined 扩展 +
  新增 BrainNil；空 query 四组空数组。
- **归档 tab 永远空白（缺陷修复）**：前端归档 tab 读 `view.archives`，但后端 `GaeaMemory()`
  结构体无 archives 字段 → 列表永远空白；改 `GaeaMemoryArchivedList` 分页加载（每页 50 +
  加载更多 + total）。
- **恢复能力补齐（Unarchive）**：memory 包此前只有 Archive（软删）无恢复路径（注释声称
  「90 天可恢复」但实际不存在）；补双后端 `Unarchive`（sqlite 置 archived=0；file 从
  `.archive/<ts>-<name>.md` 移回主目录 + 重建索引；未归档/已清理报错）+ 新绑定
  `GaeaMemoryUnarchive`（绑定面 497→**498**）+ 归档 tab「恢复」按钮（Rollback 图标/恢复中态/
  成功后刷新提示）。
- **保留期下发展示**：`MemoryArchivedPage` 增 `retentionDays`（= ArchivedRetention 90 天），
  归档 tab 顶部「归档保留 N 天，超期可清理」，清理确认弹窗文案跟随真实保留期（此前硬编码）。
- **修复漂移脚本单条差异静默放行**：`check-bindings-drift.ps1` 判 `$diff.Count -gt 0` 但
  PS 5.1 下单条差异 `$diff` 是单个 PSCustomObject（无 .Count）→ `$null -gt 0` 为 False 静默
  放行（实测复现：新增绑定后脚本仍报 OK）；`@()` 强制数组化修复 + 脚本恢复 UTF-8 带 BOM
  （AGENTS.md 编码规范）。
- **验证**：Go 全量 **112/112 包**（+6 测试：Unarchive 双后端/app 绑定/RetentionDays/BrainNil）、
  eslint **0/0**、tsc 0 errors、vitest **654/654（124 文件，+2）**、绑定面 **498 方法**漂移
  PASS（含负向验证：单条漂移现在能红）、版本四处统一 3.4.0、wails build + 冒烟 /api/health 200

## v3.3.0「质量收敛 · eslint 存量 warnings 清零 + flaky 治理」(2026-08-27)
> v3.2.1 后的工程质量刀：366 条存量 eslint warnings 归零（配置显式化 + 死代码清理 +
> exhaustive-deps 补全 + 混合导出显式声明 + 冗余 @ts-ignore 移除）、CI/测试 flaky 治理、
> releases/README.md 历史乱码恢复、前端性能体检。纯工程质量，零功能变更。
- **eslint 366 → 0（errors 0 / warnings 0）**：
  - 配置显式化：`no-unused-vars` 加 `^_` 前缀 ignore patterns（下划线 = 显式「故意不用」，
    社区标准）、`no-empty` 开 `allowEmptyCatch`（空 catch 为降级吞错的有意设计）、
    react-refresh 开 `allowConstantExport`（纯常量导出不破坏 Fast Refresh）
  - 死代码清理 56 处（未用 import/const/函数/catch 参数/解构成员，跨 40 文件）
  - exhaustive-deps 40 处：稳定依赖补全（useCallback/store 方法/setter）+ 不稳定依赖
    显式 disable 注释（含 GhostText/useVoiceChat 两处 TDZ 陷阱的 useCallback 定义上移
    重排、GraphView/Composer 两处 disable 注释位置修正）+ 复杂表达式提取变量 +
    每渲染重建数组 wrap useMemo + ref cleanup 竞态局部变量化
  - react-refresh/only-export-components 25 处：14 个混合导出文件加文件级显式声明
    （Provider+hook 同文件、工具函数供测试/复用等设计使然）
  - 移除 10 处冗余 `@ts-ignore`/`@ts-expect-error`（wails.d.ts 已生成类型，注释多余）
- **flaky 治理**：filewatch 测试事件等待超时 3s→5s（沙箱/CI 高负载下 fsnotify 投递 +
  debounce 延迟曾致首跑假红复跑绿）；CI 后端测试失败后整体重试一次（重试后仍失败
  正常红，不掩盖真实缺陷）；确认 CI 已排除 internal/tts、test-all.ps1 已有 AV 锁重试
- **releases/README.md 乱码恢复**：v2.40.0 及更早 98 行 GBK 损坏（U+FFFD 不可逆）从
  git 历史（v3.0.1 提交 7c53db8 干净版本）逐行重建，0 残留；版本索引历史行完整可读
- **前端性能体检**：大组件 memo 复查（Transcript/CostLibraryView/Message 已 memo，
  页面级组件 memo 收益有限不额外加）；唯一热点 = XlsxPreview Excel 网格全量渲染
  （maxRow×maxCol `<td>`），修复需虚拟滚动重构、收益/风险比低——按「先体检再决定」
  纪律记录待真实卡顿反馈
- **验证**：eslint **0 errors / 0 warnings**（366→0）、tsc 0 errors、vitest
  **652/652（124 文件）**零回归、Go 全量 **112/112 包**、filewatch 5 测试绿、
  绑定面 **497 方法**漂移 PASS、版本四处统一 3.3.0、wails build + 冒烟 /api/health 200

## v3.2.1「工作区内联编辑 · C5 文本文件直接编辑保存」(2026-08-26)
> v3.2.0 第二刀（蒸馏候选清单第 9 项收尾）：工作区文本文件在预览中直接编辑保存。
- **C5 工作区内联编辑（GaeaWriteFile）**：新绑定 `GaeaWriteFile(rel, content) error`
  （绑定面 497，+1）——安全四重校验：相对路径（拒绝绝对/`..` 穿越）+ 必须落在可写根
  （WriteRoots：工作区 + allow_write）内 + 文本扩展名白名单（md/txt/csv/json/toml/
  脚本/源码等 30 种）+ 内容 ≤2MB + 仅允许编辑已存在文件；**原子写**（同目录临时文件 +
  fsync + rename，失败保留原文件）。用户显式保存视为用户意图（非 agent 自动写，不走
  审批；agent 写仍受工具权限面约束）
- **FilePreview 编辑模式**（markdown/text 且未截断才可编辑——截断内容不完整，写回会
  丢数据）：标题栏「编辑」切换 → 等宽 textarea + 脏标记（琥珀色脉冲点）+ 保存状态机
  （保存中/失败可重试）+ **Ctrl/Cmd+S 保存** + 保存成功后自动重读预览；脏状态下退出
  编辑弹内联「放弃修改/继续编辑」确认条（不用 antd 静态弹窗，测试确定性强）
- **验证**：Go 全量测试绿（新增 TestGaeaWriteFile：正常写回 + 穿越/绝对路径/非文本/
  不存在/超大 五类拒绝 + 拒绝不改动原文件）；前端 tsc/eslint 0 errors、vitest
  **652/652（124 文件，+5：FilePreview 编辑模式 5 用例）**；绑定面 **497 方法**漂移
  PASS（+1：GaeaWriteFile）；wails build 发布版 + 冒烟 /api/health 200

## v3.2.0「任务可见性 · C1 任务实时输出 + C2 子代理活动行」(2026-08-26)
> v3.2.0 第一刀（蒸馏收尾第 3 轮，按候选清单推进顺序）：办公长任务与子代理的
> 「看得见在干什么」。后端零侵入扩展（tasks 输出环形缓冲 + stopping 结束态细分 +
> SubagentRunView 活动行字段），前端任务中心输出 dock + 分工面板活动行。
- **C1 任务实时输出（GaeaTaskOutput）**：tasks 包新增 `Progress.Output(line)` 输出
  环形缓冲（200 行 / 64KB 上限，超限截断标注，仅回放不消费游标）；三个消费者
  （价格抓取/批量抓取/语义索引）逐源逐批输出时间戳行；新绑定 `GaeaTaskOutput(taskID)
  → { tail, truncated }`（绑定面 496 方法）。任务中心：任务行可点击选中 → 底部共享
  输出 dock（pre 等宽回放、运行中 2s 轮询 + 自动尾随滚动、截断标注、可关闭）
- **C1 结束态细分（stopping）**：取消运行中任务先条件置 `stopping`（正在停止…，
  WHERE status='running' 防覆盖终态竞态）再传播取消，handler 退出后终态 cancelled；
  前端「停止中」琥珀色旋转徽标 + 取消按钮文案变化；重启续跑把 stopping 一并恢复为
  queued（取消途中崩溃不丢任务）
- **C2 子代理活动行（SubagentRunView + lastText/lastTool）**：`summarizeSubagentTranscript`
  从 transcript 尾部派生 最后 assistant 文本（截断 160 字）与 最后一次工具调用摘要
  （name + 结果首行，截断 80 字）；分工面板运行中卡片显示「正在：…」+「⚙ 工具」两行
  活动线（脉冲指示点，随 5s 轮询刷新）；父子拓扑按候选清单建议暂不做（meta 无父子
  记录，退化为活动行 + 扁平列表）
- **验证**：Go 全量测试绿（tasks 新增输出缓冲/stopping 竞态 2 用例 + app 活动行派生
  1 用例）；前端 tsc/eslint 0 errors（360 存量 warnings）、vitest **647/647（123 文件，
  +4：TaskCenter 3 + SubagentsPanel 活动行 1）**；绑定面 **496 方法**漂移 PASS（+1：
  GaeaTaskOutput）；wails build 发布版 + 冒烟 /api/health 200

## v3.1.1「造价数据库闭环补齐 · 测算项目 UI + 造价参考 + 复盘笔记 + 选区转对话」(2026-08-26)
> 承接 v3.1.0 发布后盘点：costproject/costref 后端与 15 个绑定已就绪但前端无入口，
> 本版补齐三个前端 UI + C4 选区转对话 + 仓储卫生。纪律延续：不做新板块、不堆功能。
- **测算项目 UI（新组件 CostProjectsView）**：造价数据库板块新增「测算项目」导航——
  左列项目列表（名称/类型/状态徽标/条目数/合计/版本数 + 新建/删除，级联）；右区详情：
  ① 项目信息编辑（名称/类型/规模/工艺/备注）；② 工程量清单表格（标题/单位/数量/单价
  行内编辑、失焦自动保存、金额=数量×单价实时计算、「引用成本库单价」搜索下拉回填
  title/unit/price/categoryPath/entryName、新增/删除行）；③ 保存版本（不可变 JSON 快照 +
  备注，列表按版本号倒序、点击查看快照明细表、可「恢复此版本」前端编排重建明细）；
  ④ **沉淀选中行回成本库**（勾选 → UPSERT cost_entries，成功后刷新库概览统计）。
  复用 v3.1.0 已就绪的 GaeaCostProject* / GaeaCostEstimate* 绑定，零后端改动
- **造价参考 UI（新组件 CostIndicatorsView）**：「造价参考」导航——按科目/按一级分类
  分组切换，表格展示 样本数/最小值/P25/中位数/均值/P75/最大值/单位（实时聚合不落表）；
  空态引导「保存版本或沉淀后自动成为对标样本」
- **复盘笔记 UI（新组件 CostNotesView）**：「复盘笔记」导航——搜索 + 状态过滤 + 新建/
  编辑弹窗（标题/结论/适用边界/风险提示/证据来源/可信度 高中低/复核状态 草稿·已确认/
  成本分类/项目类型/有效期至），卡片展示结论摘要 + 引用次数，删除经 Modal.confirm 确认；
  复用 GaeaCostNote* 绑定
- **板块导航同步**：board cost manifest Nav children 增补 测算项目/造价参考/复盘笔记
  （概览/成本条目/测算项目/造价参考/复盘笔记/价格源/价格仓库），与页面 MODULES 对齐
- **C4 选区转对话（新组件 SelectionToComposer，纯前端）**：办公板内选中任意正文
  （对话文本/文件预览/过程输出）→ 选区上方浮出「转为提问」浮动条 → 点击把选中文本
  以 `> 引用` 块插入输入框（可编辑后发送）；忽略输入框/文本域/下拉/弹窗内选区，
  不干扰既有交互；portal 渲染 + 视口边缘收敛
- **仓储卫生**：删除根目录临时脚本 `.go` / `.split.go`（历史 stage 复制脚本）与旧版
  `gaea.exe`（2026-08-14 残留产物）；releases/README.md 补 v3.0.8 索引行 + 修复
  v3.0.1/v3.0.0 两行历史乱码（其余更深历史乱码保留，源自历史编码事故）
- **验证**：前端 tsc 0 errors、eslint 0 errors（359 存量 warnings）、vitest **643/643
  （122 文件，+13：测算项目 4 / 复盘笔记 2 / 造价参考 3 / 选区转对话 4）**；Go
  build/vet 干净 + board/bindings 测试绿；绑定面 **495 方法**漂移 PASS（零新增绑定）；
  wails build 发布版 + 冒烟 /api/health 200

## v3.1.0「造价数据库 · 一级板块 + 办公蒸馏 + 死锁修复」(2026-08-26)
> 主线 = 参照 zaojia-database 蒸馏造价数据库并提升为一级板块：综合单价=一级、
> 人材机=二级组成；按用户定调「数据库就是数据库」收口——测算引用/对标由办公
> agent 工具承担，造价数据库只做数据沉淀与管理。并行落地办公蒸馏
> （dsh-better-sidebar 右侧面板 C3/C6/C7 + 文件工作台资源管理器）与 UI 紧凑化；
> 修复办公板块初始化死锁。
- **一级板块「造价数据库」**：成本库从记忆中枢二级分类提升为独立板块（board
  cost，`CostLibraryPage`，MenuOrder 5，AccountBookOutlined，导航：概览/成本条目/
  价格源/价格仓库），复用 CostB 门面；记忆中枢成本二级入口移除（避免双入口，
  记忆图谱节点仍保留琥珀色）
- **架构定稿（市场调研驱动）**：专项调研陕西蜘蛛网工程成本平台，确认其公开内容
  全部为「综合单价分析」（人工费+材料费+机械费+管理/利润/税金→综合单价），与
  市政手册同构；三个决策点定稿：记录=综合单价子目、资源库层保留不强制关联、
  费率入库仅展示追溯。调研文档
  `docs/market-research-2026-08-cost-architecture-zonghe-danjia.md`
- **数据模型（SchemaV9/V10/V12/V13）**：`cost_entries` 增加 地区（region）/价格
  时间期数（price_date）/价格口径（price_type）/有效期（valid_until）/导入原始
  行号（source_row），`cost_price_history` 同步记录地区与口径（价格快照可追溯
  「哪个地区、什么口径、哪一期」）；再增加 人工费/材料费/机械费 三个合计与
  管理费/利润/垫资/税率（仅展示不参与计算）；新建 `cost_entry_components` 组成
  行表（kind/title/unit/quantity/price/amount/note/sort）。`cost.Store` 的
  Save/Get/Delete/Search 与导入事务写库全链路接通，Save 组成行整组替换；
  `Summary` 增 `componentCount` 支撑面板统计
- **默认分类树重构**：综合单价 → 专业（道路/交通/绿化/电力/给水/暖气/雨污/照明/
  其他）→ 分部；人材机不再平级成类，资源库层由价格源模块承载
- **《市政成本测算手册》整本导入**：任一 sheet 含「综合单价分析+人工费/材料费/
  机械费」表头即触发专有解析（多专业表 → 分部行 → 子目行）；「综合单价分析」
  文本解析为人材机组成行（段头切 kind、行尾金额、原始表达式入 note 保留损耗系数），
  同名子目按项目特征片段去重；实测 8 张专业表 234 条全部命中，通用报价单导入不受影响
- **测算项目与沉淀闭环（新包 costproject，SchemaV10）**：`cost_projects` /
  `cost_estimate_items` / `cost_estimate_versions` 三表——测算项目容器（类型/规模/
  工艺/状态）+ 明细行（引用成本库单价或手动填价，数量×单价自动算金额）+ 不可变
  版本快照（回看/对比/恢复）；「沉淀」把明细行 UPSERT 回 cost_entries，
  「沉淀即调用」闭环；App 绑定 GaeaCostProjectSave/List/Get、GaeaCostEstimate* 系列
- **造价参考与复盘笔记（新包 costref）**：对「已保存版本/已沉淀」测算项目明细行
  实时聚合 样本数/极值/P25/P75/中位数/均值（指标不落表避免双写），供下次报价对标；
  复盘笔记沉淀「判断」（结论/适用边界/风险/证据/可信度/有效期/复核状态 + 引用计数）；
  新增 `cost_indicators` 办公 agent 工具（只读聚合，测算前对标与引用依据）
- **成本库数据自愈（cost/repair.go）**：修复历史遗留平铺分类路径——1420 条里 201 条
  category_path 在分类树上无法解析（树上看不到、统计对不上），规则引擎幂等映射回
  合法路径 + 从来源字符串保守回填地区/期数（不臆造）；`Store.Open` 自动执行防再次漂移
- **造价数据库面板重设计**：概览改为 库规模 hero（条目/专业/分部/组成行 + 累计
  人材机构成占比条）+ 数据健康（缺单价/草稿/引用完备度）+ 最近更新（行内人材机
  mini 条）+ 空库三步引导 + 骨架屏；模块导航由圆角胶囊改分段下划线；条目列表加
  人材机占比条、表格加「组成」列；沿用 v3 玻璃面板/辉光卡设计语言
- **办公联动收口**：`cost_search`/`cost_save`/`cost_indicators` 三个 agent 工具
  承载测算引用、单价沉淀与分位数对标；交付物面板「沉淀到成本库」一键生成
  `cost_save` 指令；`cost_save` 与测算沉淀更新既有子目时保留人材机组成/费率，
  避免改价抹掉二级明细
- **办公蒸馏（dsh-better-sidebar，2026-08-20/26 两轮）**：C3 会话级右侧面板布局
  持久化（`gaea.rightPanel.v1:<sessionKey>`）+ C6 运行域活动角标（useRunningBadge
  计数徽标，99+ 封顶）+ C7 预览队列 chip 化（点击切换/×关闭/中键关闭 VS Code 语义）；
  FileTree → 资源管理器（行悬浮 @引用 / 右键菜单：预览·外部打开·在文件夹中显示·
  复制相对路径 / 展开态按 cwd 持久化 / 树内搜索接 GaeaFileSearch），纯前端零后端；
  删除完成轮「大过程卡」（Transcript 统一交替语义，正文独立显示）；GoalCard/TodoCard
  默认折叠紧凑化（用户：太占地方）；修复 Tailwind v4 `max-w-[--maxw]` 方括号语法
  不生成 CSS → 统一 `max-w-(--maxw)` 括号语法（四处卡片宽度对齐输入框）
- **成本库入口接线（用户决策，2026-08-26）**：办公右侧面板「文件」组新增「成本库」
  子 Tab（CostLibraryPanel：条目列表/价格源/价格仓库三态 + 一键插入输入框），
  4 主 Tab 收敛不变；workspaceTabs 单源声明 + App 渲染分支 + 测试断言同步
- **修复办公板块初始化死锁（工作空间/新建会话不可用的根因）**：`GaeaInit` 持
  `ga.mu` 初始化时会走 `resumeLastSession → syncGoalForSession`，而后者内部经
  `gaeaCtrl()` 对同一把非重入互斥锁二次加锁，导致工作区存在会话时办公板块首次
  打开即永久卡死（界面停在「连接中…」、侧栏看不到任何项目/会话、无法新建会话、
  无法切换工作空间）。修复：`syncGoalForSession` 改为显式接收控制器，不再内部
  取锁；新增两个回归测试（持锁上下文直接调用 + 工作区含会话的完整 `GaeaInit`
  场景），并让 `persistWorkspaceLocked` 同步更新磁盘上的
  `sandbox.workspace_root`，避免切换工作空间后配置文件残留旧工作区路径。
- **决策：V4.0 dsh化 验证失败，正式废弃**（2026-08-26）：曾把 gaea 改造成 DSH 插件
  体系（独立工作空间 `C:\AI\gaea-v4`），用户验证后判定失败；删除工作空间内 V4.0
  文档（v4-blueprint 蓝图 / phase4 两份实施计划 / CHANGELOG v4.0.0 章节），
  `C:\AI\gaea-v4` 与 `~/.dsh*` 工作空间外文件一律不动；继续 V3 迭代
- **验证**：`go build ./...` + vet 干净 + 全量 `go test` 通过（filewatch 首跑为
  环境抖动，单独重跑全绿）；前端 `tsc -b` 0 errors、eslint 0 errors、vitest
  **630/630（118 文件）**；绑定面 **495 方法**漂移检查 PASS；`wails build` 发布版
  成功 + 冒烟测试（/api/health 200），产物 `build/bin/gaea.exe` 复制到 releases

## v3.0.8「办公板块 · 表格可交付 + 会话产物打包 + 多智能体分工」(2026-08-17)
> 按调研 docs/market-research-2026-08-office-table-agent-and-package.md 落地 P0 三项
> （表格选中区域→一键图表、会话产物一键打包 Zip、产物缩略图增强）+ P2 首项
> （多智能体分工可见），并按用户决策做界面收敛（不做 PPT、聚焦 Word/Excel、
> 不堆功能）：右侧面板 Tab 收敛为 4 个主标签、Excel 工具栏按上下文收敛。
- **P0-1 会话产物一键打包 Zip**：会话产物面板新增「打包下载」→
  `.gaea/exports/gaea-会话产物-<stamp>.zip`；只接受工作区相对路径（拒绝绝对路径/
  `..` 穿越）、缺失/目录静默跳过、zip 内保留相对路径防同名覆盖；后端
  `GaeaZipDeliverables` + 前端 Archive 按钮（对标 Kimi 工作空间 / WorkBuddy）
- **P0-2 表格「选中区域 → 一键图表」**：XlsxPreview「图表 ▾」菜单（柱状/折线/饼图
  PNG + 图表→Word/→PPT）——选中区域/单单元格/自动前两列数据提取 → matplotlib PNG
  → 预览队列；后端 `GaeaXlsxChart`（对标千问表格 Agent / ChatExcel）
- **P1 产物缩略图增强**：FileThumb 升级为内容缩略图（xlsx 迷你表格 / md 文本摘要 /
  图片 dataURL，失败回退类型图标），接入交付卡与会话产物面板，零新后端绑定
- **P2-1 多智能体分工可见**：右侧「运行」组新增「分工」子面板（SubagentsPanel）——
  子代理状态徽标/任务摘要/模型/工具范围/耗时/回答摘要，运行中 5 秒轮询；后端
  `GaeaSubagentRuns`（对标 WorkSwarm 蜂群 / QClaw V2 多 Agent）
- **右侧面板 Tab 收敛为 4 个主标签**：文件（文件/资料）/ 成果（产物/变更）/ 运行
  （任务/分工）/ 分析（统计），第一级组 + 第二级子 Tab；workspaceTabs 重构为
  WORKSPACE_GROUPS 分组清单，App.tsx 渲染分支与命令面板零改动
- **Excel 编辑器工具栏收敛**：10 个常驻按钮 → 行操作（选中才显示）+ 重算公式 +
  图表 ▾ 下拉；选中单元格布局重排为「公式栏在上、AI 编辑（单行紧凑）在下」，
  两个输入框逻辑分层；预设点击回填指令并有激活态
- **验证**：Go `internal/app` 全量 ok（12 个新测试）；前端 tsc 0 errors、vitest
  **605 通过**（新增 18 用例）、vite build 通过；绑定面 **480 方法**漂移检查 PASS；
  版本四处统一 3.0.8；wails build 发布版 + 烟雾测试（/api/health 200）

## v3.0.7「办公板块文件交互体验 · 内置 prompt 模板兜底」(2026-08-17)
> 调研 docs/2026-08-16-office-file-interaction-research.md 的 P0+P1+P2（纯前端部分）
> 全部落地：文件从「附件/路径文本」升级为一等公民交互对象——非图片附件 chip、
> 行内文件 chip 视觉统一、最近文件快捷区、多文件预览队列、产物版本时间线、
> 大工具输出有界预览、附件上下文占用透明化。
- **P0-1 非图片附件 chip 化**：拖入/选择的 docx/xlsx/pdf 等非图片文件不再注入裸
  `@路径` 文本，而是进 Composer 附件栏渲染为「图标+文件名+扩展名 badge」chip
  （点击预览/移除）；提交仍按附件数组统一注入 `@路径`，行为零变化
- **P0-2 行内文件 chip 视觉统一**：新 FileChip 组件 + lib/fileBadge 扩展名单源
  （BADGE_EXTS 从 FileMenu 收敛）；FileLinkText / Markdown 文件链接 / 流式尾部
  htmlFileLinks 全部升级为「图标+文件名+badge」，与 @ 菜单同视觉
- **P0-3 最近文件快捷区**：lib/recentFiles localStorage 单源（@ 引用与预览共用，
  去重置顶 20 条）；文件面板顶部「最近」快捷条一键回看，预览过的文件自动记录
- **P1-1 多文件预览队列**：preview store 扩展 previewList/index/navPreview 单源
  （兼容 previewFile；已在队列去重跳转、上限 50、close 清空）；App 预览状态全部
  改由 store 驱动（消除局部副本双写不一致）；预览底部 ← index/total → 导航条
- **P1-2 产物版本时间线**：sessionDeliverables 记录同一文件会话内出现次数
  （versions），产物面板对多次更新的文件显示「vN」徽标（对标 Hermes 版本步进器）
- **P2-2 大工具输出有界预览**：boundedOutput 纯函数（>60 行折叠为头部+「已折叠
  N 行」提示）；ToolCard「展开全部 N 行/收起输出」开关，超长输出不再撑爆卡片
  （对标 QwenPaw 超长输出折叠）
- **P2-4 附件上下文占用透明化**：附件 chip 显示「4.0 KB」占用（图片 base64 估算 /
  文件 File.size / PickFiles 后端 size 补前端类型），title 注明「附件占用（进入
  上下文的体量）」（对标 QwenPaw context-usage）
- **内置 prompt 模板兜底（SetPromptFS）**：main.go go:embed 内置 prompts/ 模板，
  exe 单文件分发（旁边无 prompts/ 目录）时由内置模板兜底，磁盘 prompts/ 仍优先
  （开发期直接改模板生效）；prompt 引擎 6 个新测试
- **验证**：vitest 587 通过（新增办公文件交互 40+ 用例）、绑定面 477 方法漂移
  PASS、wails build 发布版 35.2MB、冒烟 /api/health 200

## v3.0.6「编程板块工作台 · 办公会话回退分叉 · 顶栏工具栏迁移」(2026-08-16)
> 办公板块会话能力闭环（回退/分叉/回退点 + 右侧 Tab 清单化 + 会话统计回填 +
> mock 场景补全）与编程板块桌面内嵌工作台（iframe 内嵌 DeepSeek Harness Web +
> 启动引导 + 运行中工具栏移入顶栏）。
- **办公板块 UI+功能优化**：ProcessCard 状态四态 / 思考深度三档接线 / useDrawers
  收敛 / 恢复会话状态还原 / 右侧 tabs 激活态修复
- **会话回退/分叉/回退点**：后端实现会话 rewind 链路（此前为前端空实现）——
  会话级派生统计绑定 GaeaSessionStats + 前端恢复回填，根治恢复会话成本展示不完整
- **右侧面板 Tab 体系清单化**：workspaceTabs 单源声明 + WorkspaceTabs 组件 +
  装配层测试；同步清理死绑定（移除 GaeaWorkspaceChanges/GaeaSelectTab/GaeaTabMeta）
- **mock 场景补全**：审批/提问/压缩卡三类事件流 + initBridge `?mock=` 优先——
  浏览器离线开发可用
- **命令面板任务模板 / 领域色单源 / 模板缓存去重**：办公板块体验三件套
- **编程板块桌面内嵌工作台**：DeepSeek Harness Web（http://127.0.0.1:3080）以
  iframe 内嵌桌面窗口，运行中显示工作台工具栏（运行徽标/时长/URL/刷新/浏览器
  打开/停止），未运行时提供一键启动引导视图
- **前置条件真实检查**：启动引导从静态使用说明升级为逐项清单——新增
  GetProgrammingWebPreflight（harness 目录有效 / pnpm 可用 / node_modules 已装 /
  apps/web/dist 构建就绪 / 端口 3080 空闲 + all_ready 汇总），每项绿/红呈现 + 修复
  提示 + 「重新检查」按钮；未全部就绪时启动按钮禁用，杜绝「点完没反应」
- **启动日志可查看**：新增 ProgrammingWebLogTail（自启日志尾部，n 钳制 [1,200]），
  启动视图新增可展开日志面板（读取/刷新/空态提示），排障不再需要去临时目录翻文件
- **运行时长**：GetProgrammingWebStatus 新增 uptime_s；运行中工具栏显示
  「已运行 X 小时 Y 分」芯片，外部实例显示警示芯片（不可停止，防误杀）
- **顶栏工具栏迁移**：运行中工具栏（「Harness Web 运行中」徽标 / URL / 刷新 /
  浏览器打开 / 停止）整体移入顶栏 v3-strip（portal 进 v3-prog-host 宿主，与聊天
  模式条同款模式），仅编程板块激活时自动显示、其他板块自动隐藏；iframe 独占
  工作区全高展示；宿主缺失兜底保持原布局；新增 portal 渲染用例
- **启动动画视图**：点击「启动」到端口就绪之间显示启动动画——纯 CSS 双虚线环反向
  旋转 + 发光 orb 呼吸脉冲（WebView2 下不依赖 rAF，与粒子星云回退同理）+ 「已等待
  X 秒」实时计时 + 内嵌日志折叠面板；启动失败自动展开日志并回到引导视图
- **数据源 seam 化**：ProgrammingPage 从 wailsjsCompat 直调改为 bridge app seam
  （§5.3 前端侧模式）——Wails 原生走门面代理、浏览器 mock 走 makeMockApp，两种
  环境同一套代码；mock 的 StartProgrammingWeb 延迟 3s 报错，保留启动动画演示窗口
- **修复 useVoiceChat cleanup 崩溃**：wailsjsCompat 直调在 mock/未就绪时同步 throw，
  effect cleanup 的 App.VoiceStop().catch() 拦不住同步异常导致首页渲染崩（错误边界
  兜底）——两处调用加 try/catch，浏览器 mock 模式恢复可用
- **后端可测性**：端口探测/tasklist/taskkill/LookPath/cmd.Start/日志路径/等待超时
  全部经 probe* 探针注入；新增 programming_web_test.go 16 用例（状态归属/幂等启动/
  外部占用守卫/目录与 pnpm 缺失/超时/停止三态/日志尾读/前置检查全绿与全红），
  零外部依赖（不碰真实 3080 与进程）
- **绑定面**：CoreB 468→470 方法（GetProgrammingWebPreflight / ProgrammingWebLogTail），
  绑定完备性与漂移检查 PASS（476 方法一致）；wailsjs 绑定已重新生成
- 验证：Go 全量 internal/app ok（首轮 TestGaeaPriceFetchAllTask UNIQUE 冲突为偶发
  环境抖动——与 vitest 全量并行时的时序/资源竞争，单独重跑与全量重跑均通过）；
  前端 tsc 0 errors、eslint 0 errors、vitest 544 通过（编程板块 12 用例，新增
  顶栏 portal 渲染 1 用例）、vite build 通过；浏览器 mock 实机走查
  （引导视图 → 启动动画 → 失败展开日志 → 前置条件全绿）；wails build 发布版
  通过 + 冒烟测试（/api/health 200）

## v3.0.5「首页任务指挥中心改版」(2026-08-16)
> 参照 DeepSeek 首页风格重做桌面端首页：Hero 左文右卡 + 透明 AI 状态细条 +
> 「全部模块」办公大卡 + 8 卡 4×2 网格；修复设置卡被底部信息条遮挡、编程板块
> 在首页不可见；语音晶核动效经 WebView2 实机验证后取消粒子星云，恢复发光球。
- **Hero 左文右卡**：公告 pill（在线呼吸点 + 悬停箭头）+ 大标题（clamp 30-46px
  balance）+ 副标题 + 双行动卡「开始创作 / 和 gaea 对话」——整卡可点、黑色描边、
  悬停发光 + 图标放大 + 箭头滑入（位移 ≤2px）；内容收进 1240px 居中容器
- **右侧 AI 视觉卡**：深色渐变底 + 细网格纹理（径向渐隐）+ 右上星云流光；左侧
  语音晶核（呼吸环 + 静态发光球）+ 「AI 内核」状态标题 + 活跃模型 pill
  （本地/云端模型名 + 成功绿点）
- **AI 状态细条**：4 列透明无边框信息行（活跃模型/已启用引擎/资源占用/项目写作
  进度），不再呈现卡片感，保留图标发光与等宽数字
- **全部模块 Bento 改版**：新增「全部模块」分区标题 + 副文案；左侧办公大卡
  （280px 固定列、2×2 视觉高度、与右网格等高），右侧其余 8 卡 4×2 网格——
  角色库/设置与其他卡同排，不再单独成行；模块区 `flex:0 0 auto` 不被压缩，
  根治「设置卡被下方横框挡住」
- **编程板块回归**：PageRegistry 补注册 ProgrammingPage（此前遗漏导致首页编程
  卡不可见），编程卡恢复显示
- **语音晶核动效回退**：canvas 粒子星云在 WebView2 下 requestAnimationFrame 被
  节流只剩首帧（呈现为静止图片），setTimeout 驱动实测仍不稳定，最终取消粒子
  星云组件，恢复呼吸环 + 静态发光球（纯 CSS 动画，不再依赖 JS 循环）
- **响应式**：1120px 以下 Hero 转单列；920px 以下办公大卡上移整行、其余卡 2×2；
  720px 行动卡单列、视觉卡纵向；520px 网格单列、状态单列
- 验证：前端 tsc 0 errors、vite build 通过；wails build 桌面版 35.2MB；冒烟测试
  （HTTP 桥接 /api/health 200）通过；发布 gaea-v3.0.5.exe + SHA256SUMS +
  v3.0.5.md + CHANGELOG-v3.0.5.txt + README 索引更新（v3.0.0 归档）

## v3.0.4「办公能力加强 · 小说阅读体验重构 · 角色库闭环」(2026-08-16)
> 本会话两大主线：① 办公板块——任务目标（需求→验收）升级为目标工作流，目标卡/
> 待办卡拆分重设计，文件交互市场调研；② 小说板块——阅读体验 P0+P1 全量落地
> （排版/主题/书签/划线/AI 伴读/朗读同步/全文搜索）、书架网格统一与成品小说导入、
> 角色库补齐/加入项目闭环修复、导出合并进阅读面板、项目上下文不跨板块残留。
- **验收清单**：任务目标支持逐条验收标准（添加/勾选/双击编辑/删除），全部勾选自动
  推导为「已验收」，一键验收 = 全选/全不选
- **目标卡与待办卡拆分**：底部提示区从单一抽屉拆为两张独立卡片——任务目标卡
  （会话锚点，始终展开：目标文本 + 验收清单 + 自动追踪开关）与待办卡（todo_write
  提取，**默认折叠**，折叠态显示当前任务摘要）
- **待办卡重设计**：展开后按「未完成在前 / 已完成收尾」分组（已完成项加分组头、
  置灰删除线）、阶段行小标题化、当前任务高亮 + 左缘光条、进度条全圆角令牌渐变、
  全部完成态全绿
- **自动追踪（治理下自主）**：会话级「自动追踪」开关开启后，恢复会话时把任务目标
  写入 agent goal gate——回合结束未达标会自动继续工作（由模型判定验收），关闭或
  新建会话时清空，避免跨会话残留；默认关闭，不改变既有行为
- **兼容性**：RequirementView 增量字段（items/autoPursue），旧数据零迁移；文本变更
  视为新目标重置清单、自动追踪保留
- **后端**：新增 GaeaAddRequirementItem / GaeaSetRequirementItem / GaeaRemoveRequirementItem
  / GaeaSetRequirementItemDone / GaeaSetRequirementAutoPursue 5 个绑定（office 门面
  135→140 方法），绑定漂移检查同步通过
- **测试**：Go +2 用例（验收清单闭环、goal gate 接线、新会话清目标），前端 GoalCard
  +7 / TodoCard +4 用例（勾选/增删/双击编辑/自动追踪/验收态/默认折叠/分组/摘要）
- 验证：internal/app 全量 Go 测试通过；前端 tsc 0 errors、eslint 0 errors、vitest
  519 通过（2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：角色库文件交互修复（2026-08-16）
- **补齐时随机五维人格**：一键补齐/随机补齐现在会按补全后的性格生成五维人格
  （T/I/S/O/R，0-100），不再停留在默认 50/50/50/50/50；编辑器五维人格骰子改为
  AI 按性格随机（后端不可用时本地兜底），「全部随机」同样包含五维人格
- **加入项目可选小说**：「加入项目」从「写入当前打开的小说」改为弹窗选择任意小说
  （书架项目列表，含当前项目快捷项），不再需要先打开目标小说；新增
  CharacterAssociateTo 绑定（charlib 门面 15→16 方法），后端校验目标必须在书架
  目录内且为有效项目
- **加入项目后小说面板可见**：修复「已加入项目的角色在小说角色面板看不见」——
  加入时同步物化进目标小说 characters.json（按 ID 幂等合入、保留项目内既有角色/
  组织/关系）；小说面板读取角色列表时自愈，旧版本只写关联表未物化的角色自动补齐
- **测试**：Go +2 用例（指定项目关联 + 目录校验、小说面板读取自愈物化），前端
  CharacterLibEditor +2 用例（dims 骰子调用 generateRandom(dims)、补齐默认五维计入填充数）
- 验证：internal/app 全量 Go 测试通过；前端 tsc 0 errors、eslint 0 errors、vitest
  521 通过（2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：小说阅读体验优化（2026-08-16）
- **排版可调**：阅读模式新增排版面板（Aa）——字号 A−/A+（14–24）、行距
  紧凑/标准/宽松、版宽 窄/标准/铺满，偏好全局持久化，默认铺满（延续上一轮修复）
- **段落排版**：正文按空行分段、每段首行缩进 2em、两端对齐；章节标题居中大字
- **进度与位置记忆**：阅读区顶部细进度条显示本章阅读百分比；每章滚动位置按章记忆，
  切章/退出/重开自动恢复上次位置
- **键盘导航**：阅读模式 ←/→ 上一章/下一章（Esc 退出阅读，F11 专注不变）
- **测试**：新增 readingSettings 3 用例（默认值/往返/非法回退）；前端 tsc 0 errors、
  eslint 0 errors、vitest 524 通过（2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：小说阅读 P0（主题 / 书签 / 自动滚屏，2026-08-16）
- **阅读主题**：Aa 面板新增 主题（跟随/米黄/护眼绿/夜间）+ 亮度滑杆（70-120%），
  阅读区背景与文字色按主题切换，全局持久化
- **书签**：阅读栏图钉按钮 = 书签面板；滚动到目标位置「＋ 此处」添加（带章节内
  百分比 + 段落摘录），列表点击跳回、可删除；按项目持久化
- **自动滚屏**：阅读栏播放按钮开启/暂停自动滚动（40ms 步进），速度在 Aa 面板
  慢 1-5 快 五档可调；滚轮手动干预自动暂停；章节到底自动停止
- **测试**：readingSettings 扩展到 主题/亮度/滚屏（夹紧 + 回退），新增
  readingBookmarks 3 用例（往返/损坏数据/空项目）；前端 tsc 0 errors、eslint 0
  errors、vitest 527 通过（2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：小说阅读 P1-划线/高亮/想法（2026-08-16）
- **划词工具条**：阅读模式拖动选中正文 → 选区上方浮动工具条（黄/绿/蓝/粉四色
  高亮 + 「想法」），仅限单段落内选择，滚动自动收起
- **高亮回渲染**：按摘录文本在段落中重新定位（支持多色、重叠冲突跳过），点击
  高亮可编辑/删除；章节内容变更后失效的摘录自然不显示
- **本章批注面板**：阅读栏高亮按钮列出本章全部划线/想法（颜色点 + 摘录 + 想法
  标记），点击滚动定位，可删除
- **想法弹窗**：摘录引用 + 多行想法编辑，保存后随高亮展示
- **测试**：新增 readingAnnotations 3 用例（往返/颜色映射/损坏数据过滤）；前端
  tsc 0 errors、eslint 0 errors、vitest 530 通过（2 个既有 launcher.test.ts 基线
  失败，零回归）

### 追加：小说阅读 P1-AI 伴读（章节摘要 + 划线问书，2026-08-16）
- **AI 摘要**：章节正文顶部新增「AI 摘要」折叠卡——展开即生成 3-5 条要点，按章
  缓存（同章重复展开不重调），失败可重试；只使用本章本地文本
- **划线问书**：划词工具条新增「问书」——针对摘选原文提问，弹窗内展示摘录引用 +
  问题输入 + AI 回答（可反复提问），答案基于原文、信息不足时明确说明
- **后端**：新增 NovelReadingAsk 绑定（novel 门面 67→68 方法）：summary/ask 两类，
  正文按 rune 截断 9000 字，模型走 novel 功能路由（GetFeatureModel），内联提示词
  不新增模板文件；绑定漂移检查 469 个方法一致
- **测试**：Go +1 用例（守卫分支：未初始化/空正文/空问题/空摘选/未知类型）；
  前端 tsc 0 errors、eslint 0 errors、vitest 530 通过（2 个既有 launcher.test.ts
  基线失败，零回归）

### 追加：小说阅读 P1-朗读同步 + 全文搜索（2026-08-16）
- **朗读同步**：阅读页脚新增听书入口（复用 TTSPlayer），朗读时逐句高亮当前句
  （蓝色）并平滑滚动居中跟随；停止/重播自动清除高亮；后端 tts-stream 已携带
  sentence 文本，前端按句在段落 DOM 中回定位（支持跨高亮标记）
- **全文搜索**：阅读栏新增放大镜——输入即搜全书（标题 + 正文，大小写不敏感），
  一章最多一个命中、正文取首次出现的上下文片段（±40 字）；点击结果打开目标
  章节并切到阅读模式，正文渲染后自动定位并黄色高亮首个命中
- **后端**：新增 NovelSearch 绑定（novel 门面 68→69 方法）：按大纲顺序扫描章节
  （分支章节走 ReadChapterBranch），上限 100 条、损坏章节跳过；绑定漂移检查
  470 个方法一致
- **测试**：Go +1 用例（无项目/大纲未初始化/空查询守卫 + 片段截取）；前端
  tsc 0 errors、eslint 0 errors（TTSPlayer 既有 ts-ignore 警告保留）、vitest 530
  通过（2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：项目上下文不再跨板块残留（2026-08-16）
- **修复**：打开小说项目后切到绘梦/办公/聊天等其他板块，顶栏面包屑与底栏遥测
  仍显示小说标题/进度/章数/字数的残留问题
- **规则**：项目上下文（标题、写作进度、章数、字数）只在「项目锚点板块」
  （manifest.breadcrumb.anchorTo = 小说）显示；其他板块顶栏只显示当前板块名，
  底栏保留引擎/CPU/内存/GPU 遥测但不显示小说信息；小说板块内行为不变
- 验证：前端 tsc 0 errors、eslint 0 errors（既有警告保留）、vitest 530 通过
  （2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：导出标签页合并进阅读面板（2026-08-16）
- **删除**：小说板块「导出」标签页（子导航/类型/懒加载全部移除），原
  ExportPage.tsx 删除，导出逻辑抽为可复用组件 ExportPanel
- **合并**：导出入口移到阅读面板页脚（导出图标按钮）——点击弹出「导出小说」
  弹窗，一键导出 TXT + Markdown + EPUB 到小说目录 export/ 文件夹，结果列表
  与空态保留；旧 localStorage 里残留的 export 激活值自动回退首页
- **同步清理**：小说板块清单 NOVEL_NAV 移除 export 子项；NovelInspector 的
  「导出」说明区移除
- 验证：前端 tsc 0 errors、eslint 0 errors（既有警告保留）、vitest 530 通过
  （2 个既有 launcher.test.ts 基线失败，零回归）

### 追加：书架网格统一 + 导入成品小说（2026-08-16）
- **书架网格统一**：移除「最近打开」首卡双列放大（hero）效果，所有卡片固定
  等高（288px），封面条/信息区/操作区对齐，不再大大小小；封面与信息布局不变
- **导入成品小说**：书架工具条与空态新增「导入小说」——原生文件选择
  （TXT / Markdown / EPUB），弹窗填写书名/题材/文风，确认后后端解析章节、
  新建项目（outline + chapters/）并自动打开；按「第X章/Chapter N/序章/楔子/
  Markdown 标题」切分章节，TXT 支持 GBK/GB18030 编码兜底，EPUB 按 spine 顺序
  读取正文（剥离 HTML），无章节标记时归为单章「全文」
- **后端**：新增 ImportNovelBook 绑定（novel 门面 69→70 方法），返回
  NovelImportResult（路径/书名/章节数/字数）；同名项目目录自动加序号避免覆盖；
  绑定漂移检查 471 个方法一致
- **测试**：Go +4 用例（章节切分/无标记单章/GBK 解码/目录名清洗/导入端到端/
  格式守卫）；前端 tsc 0 errors、eslint 0 errors（既有警告保留）、vitest 530
  通过（2 个既有 launcher.test.ts 基线失败，零回归）

## v3.0.3「小说/绘梦/模型中心/角色剧照体验修复」(2026-08-16)
> 迭代发布：设定页手动应用 AI 输出、创作字号调节与默认 story-deslop、剧情后台构思；
> 模型中心右侧统计与资源修复（内存采集/CPU/AMD GPU/趋势时间轴）；绘梦模板库重设计
> （合并 herdsman 12 类共 231 个模板）+ ComfyUI 实时进度 + 单图铺满；角色剧照远程 URL
> 本地化防裂图。详见 releases/v3.0.3.md。
- **小说**：设定 Agent 回复新增「应用到设定」手动按钮（代码块提取兜底）；创作面板
  字号 A−/A+ 调节（12–24px 记忆）；写作技能默认 story-deslop；剧情构思改后台执行、
  完成后弹窗确认
- **模型中心**：修复右侧「统计与资源」——GlobalMemoryStatusEx 结构尺寸错误导致内存恒 0、
  CPU 短间隔轮询清零、AMD/Intel 显存识别（跳过虚拟显示器）、趋势时间轴（今日取当天、
  7/30 天按天聚合、标签修正）、加载失败可见 + 重试 + 30s 常驻刷新
- **绘梦**：模板库重设计（内置 7 类 + herdsman 12 类 = 19 类 231 个，scripts/
  gen-herdsman-templates.mjs 生成，参考 herdsman「灵感示例」卡片与分类页签）；
  ComfyUI WebSocket 实时生成进度（百分比 + 当前节点中文名）；单张结果铺满画布
- **角色剧照**：远程剧照（xAI 临时图）保存时下载本地 portraits 目录 + 启动迁移历史
  远程 URL；助手记录与小说项目角色同步；前端裂图占位兜底
- 验证：Go 全量测试通过；前端 vitest 507/509（2 个既有 launcher.test.ts 基线失败）；
  tsc 0 errors、eslint 干净；wails build 发布构建通过，版本三处统一 3.0.3。

## v3.0.1「小说板块 UX/UI 重构 · 版本漂移修复」(2026-08-15)
> 用户指令「对 gaea 的小说板块 UX/UI 进行重构」——ui-ux-pro-max skill 驱动，保持 6-tab 信息架构，
> 重构书架/阅读/创作/控制台的视觉与交互。详见 releases/v3.0.1.md。
- **书架重构**：卡片重设计（题材→令牌 tone 封面渐变条 + 阅读进度条 + 标签/统计/相对时间）；
  搜索 + 排序工具条（最近打开/总字数/章节数/书名）；「继续阅读」主操作（新增 `novel:goto-tab`
  事件联动阅读 tab）；空态/无结果态；首本宽屏 span 2 锚点；hover 位移收敛 ≤2px
- **阅读页重构**：三层堆叠 chrome（tab/信息/工具栏）收敛为单条 `.novel-chrome`；新增阅读模式
  （居中限宽 46rem 衬线排版 17px×2 + 场景分隔符 + 页脚前后章导航）；编辑/阅读分离，
  F11 专注模式 + Esc 逐级退出
- **创作页令牌化**：EditorPanel 状态 Tag 与生成进度条统一 `--gaea-glow`（novel-gen-progress-*）；
  分支色板令牌化（usePlotBranch toneColors → 语义令牌；BranchSelector/NextChapter/PlotBranch/
  NewCharacters 选中态改 `--color-primary/-warning` 派生）
- **AI 控制台面板化**：内联样式 → v3 玻璃面板（`.ai-console-*`），字号 9-10px → 12px、
  antd Tag 预设色 → 令牌 Tag、FAB 30px 不与轨道条重叠、条目键盘可达
- **令牌化清理**：NovelSettingPage 导入/导出、ChapterEditor 重试条与右键菜单、NovelInspector 保存态、
  OrganizationEditModal、RelationGraph 画布色板（挂载时 getComputedStyle 解析令牌）；
  novel-workspace.css +566 行重构层（零硬编码 hex），新元素进 reduced-motion 降级
- **版本漂移修复**：app_info.go `AppVersion` 停在 2.40.0（v3.0.0 发布漏改）——
  sync-version.ps1 统一三处（app_info.go / wails.json / versioninfo.rc）→ 3.0.1，frontend package.json 同步
- **基建**：vite.config.ts `/api` 代理支持 `GAEA_PROXY_PORT` 环境变量（本地桥接端口冲突可指路）
- 验证：tsc 0 errors + vitest 455 通过（2 个既有失败在 launcher.test.ts，改动前基线相同）+
  `wails build` 重建通过 + `scripts/smoke.ps1` 冒烟通过 + 浏览器端到端实测（书架→大纲→阅读模式→
  创作→AI 控制台）。发布 gaea-v3.0.1.exe。

## v3.0.0「星枢 Constellation OS · UI 革命性重设计」(2026-08-15)
> 用户指令「革命性重设计整个 UI，适配 V3.0，面板布局可调整」——ui-ux-pro-max skill 驱动 +
> 12 个子代理并行实施。设计规格见 `docs/2026-08-15-gaea3-ui-constellation-os.md`，
> 令牌见 `design-system/gaea/MASTER.md`（v2）。
- **壳层革命**：顶栏横向菜单 → 左侧**指挥轨道**（icon dock，hover 展开标签、键盘方向键导航、
  激活项 = 主色容器 + 左缘光条 + 呼吸 orb）+ 顶部**轨道条**（面包屑 / 模型 pill / 主题点 / ⌘K 搜索 / 设置）
  + 底部**遥测轨道**（CPU/内存/GPU 实时面积 sparkline + 引擎 pods + 写作进度，可折叠展开）；
  快捷键升级 Ctrl+1~9（MainLayout.tsx + v3/foundation.css）
- **板块工作台革命**：10 板块统一 3 分区工作台（侧栏 zone | 主区 zone | inspector zone），
  12 个子代理并行改造：home Mission Control（Bento 网格 + AI 状态卡）/ chat 对话驾驶舱（新增
  上下文·人格 inspector）/ novel 世界构建工作台（轨道子导航 + 大纲树侧栏 + 属性检查器）/
  imagegen 画廊工作台（新增历史·队列 inspector）/ gaea 办公 3 分区（会话栏|过程卡流|工具交付物）/
  memoryhub 记忆图谱舰桥（左分类轨道 + 右详情检查器）/ modelcenter 引擎控制台（左导航 + 右统计检查器）/
  characterlib 角色档案库（左检索栏 + 右详情检查器）/ settings 控制室（左分类导航 + 帮助面板）/
  knowledge 知识舰桥（列表|详情|引用 3 分区）
- **视觉升级 Luminous Glass 2.0**：`--v3-*` 令牌派生（高光线/柔光/分区线/遥测色），卡片顶部 1px
  高光线、hover 柔光位移 ≤2px、状态三重传达；App.tsx 补齐容器色令牌 shim；零硬编码 hex、antd icons only、
  焦点环/aria 齐全、reduced-motion 双路径降级
- **修复**：modelcenter InspectorPanel 注释 `*/` 语法错误、ChapterPage activeTab TDZ、CharacterLibraryPage
  回调参数类型（build 模式 tsc -b）、test/setup.ts 补 localStorage/sessionStorage polyfill（Node 25 基线）
- **发布前打磨（T6-10.2 收官）**：
  - 聊天：模式切换条（普通对话/角色对话）从聊天窗口上方移入全局顶栏轨道条（v3-strip）——
    MainLayout 提供 `#v3-chatmode-host` 宿主容器（仅聊天板块显示），ChatPage 经 createPortal 渲染，
    ChatModeBar 新增 `variant='strip'`（去横条外壳，由轨道条统一玻璃/分割线）
  - 聊天：输入框两级重设计——工具行（搜索/深度思考/语音，激活态胶囊 + 键盘提示）独立于输入卡，
    输入卡仅 textarea + 发送（ChatComposer 新增 ComposerTool 子组件）
  - **根因修复（重要）**：7 个 CSS 文件头注释内 `--md-sys-*/--gaea-*/--color-*/--v3-*` 的 `*/` 提前闭合
    CSS 注释，解析器吞掉各文件首个规则（`.novel-hub`/`.ml`/`.v3-*` 等）——表现为小说板块面板
    「只有上半截」（高度塌缩为内容高）、首页布局约束失效；全部改 `* /` 修复（novel-workspace /
    module-launcher / foundation / imagegen / character-library / settings-page / hub）
  - 首页全屏适配：移除 `.ml-shell` 1320px 宽度上限（全屏 Bento 网格随宽扩展），卡片最小宽 176→216px，
    宽屏（≥1600px）语音卡内容居中 + 侧栏限宽
  - 验证：tsc/vite build 通过；浏览器 2560×1440 全屏 + 小说三栏撑满截图走查；`wails build` 重建通过
- 验证：tsc 0 errors + vitest 全量 **91 文件 457 用例通过** + `npm run build` 成功；
  浏览器全板块截图视觉走查（home/chat/novel/imagegen/办公/memoryhub/modelcenter/characterlib/settings）
- 版本：wails.json / versioninfo.rc / frontend package.json → 3.0.0

## v2.40.0「3.0 架构主线 · Wave 4：Step 3 收官」（2026-08-15）
> 3.0 架构改造 Wave 4：Step 3（Provider Seam）遗留收官——semantic_search 工具注册、
> BalanceKind 从 ProviderEntry 贯通、ModuleLauncher 清单化，双轨并行实施 + 父代理集成。
> 详见 releases/v2.40.0.md。
- **semantic_search 工具注册**：决策纳入——实现完整且有 E2E 测试，从死代码恢复为办公 agent
  可用工具；gaeaSpecialistTools 集中注册 ocr + semantic_search，测试断言防回归
- **BalanceKind 贯通**：ProviderEntry 新增 `balance_kind`（空 = 历史默认 deepseek 形状）；
  boot→control.Options→controller.Balance 全链路透传，改走 billing.FetchByKind——切换余额
  后端只改配置、未知 kind fail-closed，补齐 Step 3d #8 消费端贯通；config/control/boot 三层测试
- **ModuleLauncher 清单化**：新增 boards/launcher.ts 纯函数派生（deriveLauncherModules +
  LAUNCHER_DESC）；ModuleLauncher 改 useSyncExternalStore 订阅活动清单，删除静态
  canonicalBoards 引用——后端合并清单（含 knowledge）变化后首页启动器自动跟随；
  launcher.test.ts 7 用例 + manifests.test.ts 36（43/43 过）
- 验证：go build/vet 干净 + test-all.ps1 全量（见发布说明）+ 前端 tsc/eslint 0 errors +
  vitest 全量（见发布说明）+ TestBindingsCompleteness PASS（464）。发布 gaea-v2.40.0.exe。

## v2.39.0「3.0 架构主线 · Wave 3」（2026-08-15）
> 3.0 架构改造 Wave 3：Step 3b LLM Seam + Step 3c OCR/ASR/TTS Seam + Step 3d 分类统一与 8 处硬编码注册表化 +
> 前端 GetBoardManifests 接线，四路并行实施 + 父代理集成。详见 releases/v2.39.0.md。
- **Step 3b LLM Seam**：LLMProvider{Provider;Chat} + ChatFromStream 聚合；bridge 互斥自注册（DefaultLLMKind=wubigrok）；
  boot.NewProvider 经 NewLLM（providers[].kind 驱动 + fail-closed）；agent 聊天/子代理只依赖 seam 接口（19 测试）
- **Step 3c OCR/ASR/TTS Seam**：OCRProvider（ovis/tesseract，GAEA_OCR_ENGINE 驱动）+ TTSProvider（edge/sapi/herdsman/xai
  自注册，四级回退与合成器链注册表化）+ ASRProvider（herdsman，voice.Manager 接口注入）；isSTTModel 委托分类单源
- **Step 3d 分类统一 + 8 处注册表化**：modelengine 导出 ClassifyModelKind/ClassifyModelByName（六桶单源）；websearch 6 引擎
  注册表（engine_order 可配序）/embed/rerank/vision/markitdown/billing 注册表化（弃 HERDSMAN_BASE_URL/GAEA_VISION_*
  环境变量绑死）；OCR 工具补注册；image_gen 裸字符串改常量
- **前端接线**：loadBoardManifests 改调 CoreB.GetBoardManifests + normalize 差集（knowledge/home/weixin）+
  KnowledgePage 注册；45/45 测试
- **父集成**：gaea.toml 新增 [retrieval]/[vision]/[markdown_converter] 段 + [search] engine_order；boot 装配四组
  Set*Runtime；app 层 5 处 provider.New→NewLLM
- 验证：go build/vet 干净 + test-all.ps1 110/110 + 前端 tsc/eslint 0 errors + vite build 42.9s + vitest 420 过
  （27 基线失败零回归）+ TestBindingsCompleteness PASS（464）。发布 gaea-v2.39.0.exe。

## v2.38.0「3.0 架构主线 · Wave 2」（2026-08-15）
> 3.0 架构改造 Wave 2：Step 1 app 层接线 + Step 2 板块 Manifest（后端 board 包 + 前端 PageRegistry）+
> Step 3a Image Seam，四路并行实施，每 Step 独立提交。详见 releases/v2.38.0.md。
- **Step 1 app 层接线**（会话事件日志「日志即真相」运行时闭环）：Resume→Restore（DetectLegacy 迁移 →
  checkpoint+log tail 重放）、Save→日志（事件模式双写）、模型调用前 flush 检查点（fail-closed，失败中止回合）、
  压缩→checkpoint（回合后 Snapshot 刷新压缩投影 + 已消费 seq）；session.log_format 回退开关，legacy 零行为变化
- **Step 2 板块 Manifest**：board 包（Board 接口 + Manifest 16 字段 + Validate 缺陷 2 防复发）+ 10 板块 canonical
  清单（9 业务 + knowledge D7）；module_registry manifest 驱动装配（intent 无 handler 启动显式报错）；
  GetBoardManifests 挂 CoreB（绑定面 464 方法）；前端 PageRegistry + MainLayout 附 B 12 硬编码点清单化 +
  events.ts 常量表（21 后端 + 4 前端）+ ModuleLauncher 清单驱动；label 单一来源统一菜单文案（用户决策）
- **Step 3a Image Seam**：图片后端注册表化（openai 兼容/comfyui 各自 init 自注册，互斥注册 panic、未知 kind
  fail-closed）；generateImageXAI 走注册表 + 401 刷新 token 单次重试守卫（retried 防无限递归）+
  imagine:content-moderated 友好提示；config 驱动选择零代码切换
- 验证：go build/vet 干净 + test-all.ps1 110/110 包 + internal/app 26.5s + 前端 tsc/eslint 0 errors +
  vite build 通过 + vitest 404 过（27 个 jsdom localStorage 基线失败，与 v2.37.0 一致零回归）+
  TestBindingsCompleteness PASS（464）+ check-bindings-drift OK。发布 gaea-v2.38.0.exe。

## v2.37.0「正确性纵深 · 收官」（2026-08-15）
> 阶段 7 第二~四刀（T7-2 可见性收口 / T7-3 名实相符 / T7-4 前端性能收尾）并行实施 +
> 3.0 架构主线 Step 0 修债 + Step 1 会话事件日志。详见 releases/v2.37.0.md。
- **T7-2 可见性收口**：qrlogin/chatWebSearch/SaveConfig/LocalTranslate 吞错清零；成本进料截断 6000 字 +
  整批事务；测评参数钳制 + 基地址 engineMgr；token 明文清理 + 剧照 ID 哈希防穿越（41 测试）
- **T7-3 名实相符**：PDF FlateDecode 压缩流还原 + OCR 单页容错 + OvisOCR2 4096/截断检测；语义检索按需
  + search 上限 + BM25 缓存 + dashboard mtime 真实聚合 + WatchErr 回退轮询（约 33 测试）
- **T7-4 前端性能收尾**：写路径静默清零 + 三态错误重试 + Transcript/MarkdownContent memo + Toast role +
  reconcileFinalAnswer 完整文本比较（41 用例）
- **Step 0 修债**：office 模块注册 GaeaSend + MainBrainChat 全链路测试 + 版本源同步脚本（搭车）
- **Step 1 会话事件日志**（3.0 地基）：append-only 日志 + 投影 + checkpoint + 迁移 + 派生 API +
  GaeaHistory 黄金测试逐字节一致（机制层；app 接线留待 gen_bindings）
- 验证：go build/vet 干净 + 逐包测试全绿（含 session 67 / T7-2 41 / T7-3 约 33 / T7-4 41）+ 前端
  tsc/eslint 0 errors + vite build 通过 + 冒烟 /api/health 200。发布 gaea-v2.37.0.exe（34.5MB，
  SHA256=37A56F54DF653E3D9E8A5751EA282CEB34BF5BBCA2672D26439BF7BAEBA7A62B）。

## v2.34.0「正确性纵深 · 并发正确性」（2026-08-15）
> 阶段 7 第一刀（T7-1）：轻语会话并发安全、任务调度器竞态、TCCA 指标聚合收敛、AI 客户端状态与重试。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段7-正确性纵深.md；详见 releases/v2.34.0.md。
- T7-1.1 **轻语会话并发安全**（internal/whisper + whisper_handler + app.go Shutdown）：Orchestrator
  per-instance Mutex 串行化三并发入口（GUI/微信/语音）；修复 CloneFullState 浅拷贝竞态（-race 实证）→
  逐指针/切片/map 深拷贝；WorkingMemory/AssociationIndex/HabitsStore/ActiveRecall 加 RWMutex；修复
  forSession 读路径惰性写 map（-race 实证）→ 只读；跨会话持久化走 persistStateSync（回合锁内快照 +
  persistMu 落库）+ drainAndPersistAll 挂 Shutdown（末轮先 drain 再 persist）；rhythm 包级计数器移入
  Orchestrator 实例（Reset 只清自己的）；新增 12 测试（并发访问/回合串行/节奏隔离/末轮落库）；
- T7-1.2 **任务调度器竞态修复**（internal/gaea/tasks）：markTerminal 进度语义（succeeded 才置 100，
  SQL CASE WHEN）；取消优先于 succeeded（handler 返回 nil 也不吞取消）；Cancel 与出队原子化
  （WHERE status='queued' 条件 UPDATE + 出队前注册取消）；runNext 包 defer recover（handler panic →
  failed 不重试，worker 存活）；新增 10 测试（进度/取消优先/竞态 50 并发/panic 恢复），22/22 -race 全绿；
- T7-1.3 **TCCA 指标聚合收敛**（internal/gaea/context/metrics.go）：MergeChild 与 Report 同字段集
  （补齐 CacheHitTokens/CacheMissTokens/BreakCount/CompactionCount 四条漏项）；merged 标记移入 child.mu
  临界区 + 数据快照走 child.Report()（子锁，锁序 父→子→孙 无死锁）；ForkCount +1 每 child 恰好一次
  （children 移除 + merged 标记防重）；新增 6 测试（Merge 前后 Report 一致/全字段/防重/并发 -race）；
- T7-1.4 **AI 客户端状态与重试**（internal/ai/client.go）：非流式 Chat 复用流式退避（连接错误/5xx
  重试 1s/2s；401 仅 xAI 同函数内刷新重发一次，不递归不占双槽）；activeEngineID/imageBackend/token
  加 RWMutex + GetToken single-flight 刷新；修复 vet 错误（Sprintf %w→%v）；新增 7 测试
  （连接/5xx/401 刷新/不占双槽/状态并发/20 并发单飞刷新）；
- 验证：go build/vet 干净 + scripts/test-all.ps1 **109/109 包 ok** + 并发门禁 C（whisper/tasks/context/ai
  go test -race 全绿，-race 需 cgo：CC=C:/msys64/ucrt64/bin/gcc.exe）；前端零改动（tsc 0 errors、
  eslint 0 errors、359 存量 warnings 与基线一致）；TestBindingsCompleteness 兜底（无新绑定）；
  冒烟通过（/api/health 200）。

## v2.33.0「质量收敛 · 前端收敛」（2026-08-14）
> 阶段 6 第十刀（T6-10）贯穿收官：巨型文件拆分、any 清零、漂移检查恢复、mock 契约化、桥接归一、性能与补测。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.33.0.md。
- T6-10.1 **巨型文件拆分**（8 个巨型文件全部收敛，拆分全程行为测试基线先行）：
  - ChatPage.tsx 1022→370 行（pages/chat/{constants,types,utils}.ts + components/chat/{ChatComposer,
    ChatModeBar,ChatPersonaBar,MessageList,SuggestionCard,WelcomeScreen}.tsx + hooks/useChatStream/
    useChatTopics/useChatVoice/useCustomTemplates.ts）；
  - ImageGenPage.tsx 911→310 行（hooks/useImageGenConfig/useImageGenHistory/useImageGenQueue +
    components/imagegen/meta.ts）；
  - CapabilitiesPanel.tsx 803→178 行（capabilities/{ServersSection,SkillsSection,ToolsSection}.tsx +
    useCapabilitiesData.ts）；
  - Composer.tsx 786→406 行（composer/ 7 组件 + useComposer{Attachments,Menus,Workspace}.ts）；
  - mock.ts 1563→50 行（按域拆 mock/{chat,core,cost,memory,model,office,retrieval,settings,shared,state}.ts
    10 文件，11 个 no-op 逐条落实并注释）；
- T6-10.2 **any 清零**：eslint.config.js no-explicit-any warn→error 进 CI；315 处显式 any 消灭（→0），
  新增 any 即 lint 失败；历史 as any 逃生口由类型化兼容层替代（wails.d.ts 注释成文）；
- T6-10.3 **绑定漂移检查恢复（双向）**：gen_bindings 新增 -names 模式（只输出方法名稳定排序）；
  bindingNames.ts（462 方法清单）；bridge.ts 类型级双向守卫 _CheckAppBindingsHasNoStray +
  _CheckAppBindingsCoversAll（任一方向漂移 tsc 即红）；scripts/check-bindings-drift.ps1 + CI 步骤；
- T6-10.4 **mock 契约对齐**：mock-contract-e5.test.ts RetrievalEvalRun 改契约校验（total=12 真实查询集、
  threshold=0.8、passed 与 recallAt10 自洽、kind:name 形式、首条锚点「打桩设备 台班价」），不再锁定
  虚构 0.85；CostImportVisionPreview/CostCompare/UnifiedSearch 补结构断言；
- T6-10.5 **虚拟化与性能**：Sidebar 会话列表 react-window List 虚拟滚动 + 过滤防抖；CostLibraryView
  memo/useCallback/useMemo 化；新增 useDebouncedValue（空串即时同步）+ 6 测试，Composer 外 2 处消费；
- T6-10.6 **桥接归一**：删除 api/bridge.ts（123 行旧代理）；initBridge 并入 gaea/lib/bridge.ts（+478）；
  wails.d.ts 同步收敛；新增绑定单处注册；
- T6-10.7 **测试补强**：Sidebar.test +40 / CostLibraryView.test +56 / GraphView.test +47 / ChatPage.test +26 /
  CharacterLibEditor.test +8 / BindSection.test +8 / useDebouncedValue.test 新 6 用例；vitest 354→361；
- 验证：go build/vet 干净 + go test ./... **109/109 包 ok** + TestBindingsCompleteness PASS（462 方法，
  无绑定变更）+ 漂移检查 OK；tsc 0 errors、eslint 0 errors（359 存量 warnings）、vitest **361/361**
  （80 文件）、vite build 14.46s；冒烟通过（/api/health 200）。发布 gaea-v2.33.0.exe（32.8MB，
  SHA256=8FADBB7385D794DB69842171D4F95E678849FA8481686F646A4BBA6E94F4E92F）。

## v2.32.0「质量收敛 · 辅助合集·名实相符」（2026-08-14）

> 阶段 6 第九刀（T6-9）：微信生命周期与凭据、OCR 兜底名实相符、配置原子写、路径端口可配置、token 改 header。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.32.0.md。
- T6-9.1 **微信生命周期与失效自愈**（internal/channels/weixin/clawbot.go + whisper_state.go）：
  - Stop 幂等（stopMu + stopCh 关闭即置 nil，二次 Stop 不 panic）；Start 支持重启（running.Swap 幂等
    防重复拉起 + stopCh 重建 + sessionExpired 重置，Stop→Start 轮询真正恢复）；
  - 会话过期（errcode=-14）触发 OnSessionExpired 回调后退出轮询（删除 5 分钟空转）；app 层注入回调
    emit notice「微信助手 X 会话过期，请重新扫码绑定」；getUpdatesFn/notifyStartFn/notifyStopFn 测试注入点；
  3 测试；
- T6-9.2 **凭据与表治理**：
  - wxToken DPAPI 加密（assistant/manager.go：save() 落盘加密 dpapi: 前缀、Load 解密/旧明文一次性
    迁移、解密失败返回含助手 ID 明确错误；内存保持明文 List 回显不变）；
  - weixin_* 4 张死表（grep 核实零读写）SchemaV13 DROP 追加迁移链 + ClearStructuredData 移除；3 测试；
- T6-9.3 **OCR 兜底名实相符**（office/docmd/ocr.go + single_prompt.go:43）：
  - 超时杀进程树：proc.StartTracked（Job Object）+ 超时 KillTracked + 同步 Wait 回收，失败零孤儿；
  - 单图降级：OCRImageText OvisOCR2 不可用 → tesseract 降级（同 PDF 参数），双不可用才报安装提示；
  - 文案删除名不副实的「Windows 原生 OCR」（RapidOCR 核实存在保留）；4 测试；
- T6-9.4 **配置原子写**（internal/config）：saveConfigFile 临时文件+fsync+rename 原子覆盖（失败保留
  原文件）；Load 损坏备份 .gaea_config.json.corrupt-<ts> 后默认值继续；4 测试；
- T6-9.5 **CosyVoice 路径端口可配置 + 退避重试**：新配置键 cosyvoice_dir/cosyvoice_port（默认与历史
  一致，端口校验）；tts_service.go 写死常量全删由配置推导；启动失败 1s/2s/4s 退避重试 3 次；4 测试；
- T6-9.6 **token 改 header**：服务端 tokenOK 删除 ?token= 查询兜底（仅 Bearer/X-Gaea-Token，常量时间
  比较不变）；前端弃 EventSource 改 fetch 流式 SSE 带 Authorization 头（parseSSEFrame/parseSSEStream
  纯函数：跨 chunk/CRLF/keep-alive/流尾 flush）；httpToken.ts 删 URL query 读取；Go 3 + 前端 6 测试；
- 验证：改动包全绿 + vet 干净 + TestBindingsCompleteness PASS（462 方法）；tsc 0 errors、
  vitest **354/354**（79 文件）、eslint 0 errors（72 存量 warnings）、vite build 15.17s；
  全量 go test 中 hook/skill/tts 3 包 AV 锁 test.exe（单独重跑全绿，环境抖动先例）与 docmd GBK 编码
  类失败（c426d3f 基线 worktree 复现同款，非回归）。

## v2.31.0「质量收敛 · 记忆·生命周期与审计」（2026-08-14）
> 阶段 6 第八刀（T6-8）：dream 写入审计、facts 归档生命周期清理、索引截断按边界、记忆组件补测。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.31.0.md。
- T6-8.1 **dream 路径决策与审计**（internal/gaea/control/controller_memory.go + internal/app/gaea_dream.go）：
  - 审批决策成文（docs/DREAM_WRITE_POLICY.md + 代码注释）：dream 写入不纳入 hardAskTools 逐条审批
    （后台异步 90s 超时无法等人工确认；显式路径本身即用户触发），补偿 = 每次实际写入落审计日志；
  - SaveDreamFacts 签名改 (source string, facts)（source=auto_dream|explicit），审计行
    {ts, source, saved, names} 追加到 <userDir>/dream-audit.jsonl（JSONL，尽力而为不阻断写入）；
  - DreamAuditEntries 读取入口（倒序最近 N 条）；新增 3 测试（自动/显式各断言 1 条审计行 + 记忆未配置跳过）；
- T6-8.2 **facts 生命周期清理**（internal/gaea/memory + 新绑定 + 前端按钮）：
  - 保留策略：归档超 90 天（ArchivedRetention）硬删；sqliteBackend/fileBackend 双后端实现
    CleanupArchived（返回被删行含溯源字段：名称/描述/正文/归档时间/来源会话）；
  - ListArchivedPaged(limit, offset) 总量+分页（limit 钳制 [1,200]/默认 50），防全量返回；
  - 新绑定 GaeaMemoryCleanupArchived / GaeaMemoryArchivedList（gen_bindings 462 方法 + 完备性 PASS）；
    清理逐条 slog 日志 + 溯源审计 purge-audit.jsonl（GAEA_DATA_ROOT 可隔离，测试不触真实用户库）；
  - 前端「归档」tab「清理超期归档」按钮（Modal.confirm + message.success + 刷新）；
  - 新增测试 7：memory 包 5（SQLite 清理/无超期不误删/分页/文件后端清理/文件后端分页）+ app 2
    （清理幂等+审计落盘、分页绑定）；
- T6-8.3 **索引截断按边界**（internal/gaea/memory/memory.go）：
  - 预算口径统一：memoryIndexBudget（3000 runes）→ memoryIndexBudgetBytes = 4096，与 Block() 的
    4096 字节全块阈值同口径（注释成文）；
  - capMemoryIndex 改 truncateIndexByLines 纯函数：预算内最后一个 '\n' 处按行边界截断（不切半行），
    markdown 链接保护（未闭合 "[" 或 "](url" 的 ")" 被截时回退到链接起始行之前整体舍弃），
    只在 ASCII '\n' 处切 → UTF-8 安全不产生半个 rune；单行超预算宁丢整行不切半字；
  - 截断提示文案与旧实现逐字一致；预算内原样返回不追加提示；
  - 新增 6 个测试函数（行边界/字节预算/rune≠byte 中文 emoji/链接完整性 5 子测/单行超预算），
    既有 recall 预算用例全绿；
- T6-8.4 **前端组件补测**（memoryhub，新增 13 vitest 用例）：
  - GraphView 5 用例：链式 stub 替身 3d-force-graph（vi.hoisted），断言工具条/节点边计数/类型过滤
    重构图/节点点击详情 Modal/空数据/variant=home 隐藏工具条；
  - WhisperMemoryLibrary 8 用例：domain 分组（含未知归「其他」）、三关键词搜索过滤、情节 tab
    （emoji/强度条/关键词/轮次）、事实/情节详情 Modal、导出归档链路（PickDirectory +
    WhisperExportArchive 调用）、事实/情节双空态；
  - 组件实现 0 改动（纯测试）；桥接经 vi.mock("../../lib/bridge") 注入确定性数据；
- 验证：改动面 Go 包（control/memory/app）全绿 + vet 干净、TestBindingsCompleteness PASS（462 方法）；
  tsc 0 errors、vitest **348/348**（78 文件）、eslint 0 errors（存量 warnings）；
  internal/app 全量存在 2 个与 v2.30.0 基线一致的既有环境失败（TestOfficeFullPipeline GBK 编码提取、
  TestSemanticSearchTool_EndToEnd 需本地嵌入模型），已用基线 worktree 复现同失败，非本刀回归。

## v2.30.0「质量收敛 · 小说·导出与原子性」（2026-08-14）
> 阶段 6 第七刀（T6-7）：export 整改、生成中断与互斥、落盘原子化、模板占位符、CreatePage 拆分。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.30.0.md。
- T6-7.1 **export 整改与测试**（internal/export，新增 13 测试）：
  - 章节读取失败静默 continue → slog.Warn + FailedChapters 计数（四格式观测一致）；
  - 作者取自项目元数据（ProjectMeta.Author，无配置回退 "gaea"）；EPUB 与 HTML 统一 markdownToHTML
    单一分段器（删除 chapterToHTML）；TXT/MD 世界观对齐；TXT/MD/EPUB 写前 MkdirAll；
  - sanitizeFilename 加固（Windows 保留名 CON/PRN/AUX/NUL/COM1-9/LPT1-9 含 CON.txt 形式 + 尾部点/空格）；
  - EPUB AddSection 错误不再丢弃；失败分支测试因沙箱无 SeCreateSymbolicLinkPrivilege 以 Skip 降级（逻辑保留）；
- T6-7.2 **生成中断与互斥**（internal/app/create_chapter_handler.go）：
  - 请求级 context.WithCancel（取消传播到 ChatStream，双路径落盘已生成部分）；
  - 新绑定 CancelCreateChapter(chapterNum, branch) bool（幂等；gen_bindings 460 方法 + 完备性 PASS）；
  - 按章节互斥（同章节并发生成拒绝明确错误，不同章节并行）；取消未生成内容不写空文件；
- T6-7.3 **落盘原子化**（internal/project）：writeFileAtomic（CreateTemp→fsync→Rename）覆盖
  writeJSON 全部 JSON + WriteChapter/WriteChapterBranch/WriteWorldview；失败清理临时文件保留旧文件；5 测试；
- T6-7.4 **模板替换精确化**：prompts/create-chapter.json "5000" 字面量 → {word_count} 占位符，
  substituteWordCount 只替换占位符杜绝误伤；2 测试；
- T6-7.5 **前端拆分与停止按钮**：CreatePage 791→288 行（拆 8 文件：chapterStreamTypes 判别联合 +
  useChapterStream hook + ChapterTreePanel/EditorPanel/CreateInspector/NewCharactersModal/BranchWizardModal +
  characterStatus 枚举单源）；「停止生成」按钮（CancelCreateChapter 接线，false 本地兜底不悬挂）；
  cancelled 事件三路收尾保留部分正文；新增 18 vitest 用例；
- 验证：export/project/types/app 包 ok（app 24.3s）、vet 干净、TestBindingsCompleteness PASS（460 方法）；
  tsc 0 errors、vitest **334/334**（76 文件）、eslint 0 errors（存量 warnings）。

## v2.29.0「质量收敛 · 模型中心·密钥与 UI」（2026-08-14）
> 阶段 6 第六刀（T6-6）：refresh_token DPAPI 加密、汇率配置化、probe 告警修复、UI 拆分与竞态守卫。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.29.0.md。
- T6-6.1 **refresh_token 密钥一致性**（internal/auth/token.go）：Save 敏感字段（access_token/refresh_token）
  经 secure.EncryptString DPAPI 加密落盘（dpapi: 前缀，JSON 结构不变）；Load 有前缀解密、解密失败返回含字段名
  的明确错误（绝不静默 nil）；无前缀旧明文自动一次性重写为加密（迁移幂等，同一把锁内完成）；非 Windows 降级
  分支 round-trip 完整；新增 3 测试（落盘无明文/旧明文迁移/解密失败报错）；
- T6-6.2 **汇率配置化**（internal/modelengine/stats.go + internal/config + 绑定）：
  - usdToCny 写死 7.2 → 配置键 usd_cny_rate（默认 7.2，saveSetters 拒绝 <=0/NaN/Inf）；
  - 新绑定 GaeaGetUsdCnyRate/GaeaSetUsdCnyRate（gen_bindings 459 方法，TestBindingsCompleteness PASS）；
  - 注入式缓存（启动注入 statsRecorder 内存副本，修改双写即时生效，record/summary 零 IO）；
  - 前端 ModelPanel 汇率输入（回填/保存/正数校验）+ engines.ts 包装；新增 4 测试；
- T6-6.3 **probe 告警文案**（internal/herdsman/probe.go:221）：目录缺失告警改打印真实目录路径
  （filepath.Join(rootDir, name) 与 checkDir 口径一致），不再误打印 config.yaml 路径；新增 1 测试；
- T6-6.4 **UI 拆分与单源**：ModelCenterPage 顶层 useState 42→3（5 分类状态下沉到 5 个 hooks：
  useEngineState/useStatsState/useImageState/useVoiceState/useBindState）；XAI_VOICES 全前端仅 utils.tsx
  一处定义（VoiceSettingsPanel/ChatPanel 改 import 单源）；
- T6-6.5 **竞态修复**：refreshLocalModels 请求序号守卫（refreshSeq，过期结果丢弃）+ 5s 定时器随 category
  重置（effect 开头作废上一分类在途刷新）；新增 2 竞态测试；
- 验证：auth/modelengine/config/herdsman/app 5 包 ok（24.4s app）、vet 干净；tsc 0 errors、
  vitest **316/316**（72 文件）、eslint 0 errors（762 存量 warnings）。

## v2.28.0「质量收敛 · 轻语·测试与可观测」（2026-08-14）
> 阶段 6 第五刀（T6-5）：补测试 146 用例、错误可见化、异步写可观测、成人模式决策成文、db 收敛、占位清理。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.28.0.md。
- T6-5.1 **补测试**（仅新增 9 个测试文件，零实现改动）：146 个用例（要求 45+）；emotion_fusion 18/18 函数
  100% 语句覆盖；memory_consolidator 98.5% / memory_contradiction 99.3% / memory_self_editor 98.5% /
  vector_store 96.5% / dispatch_router 96.3% / agent_loop_runner 92.3% / canon 全 100% / desktop 96.8%+；
  测试暴露 3 个真实缺陷（记录未改）：normalizePath %ENV% 展开不生效（Go os.ExpandEnv 不支持 %VAR%）、
  mergeProhibitions 道歉过滤漏带引号"对不起"、self_editor log 裁剪后可重新增长至 200；
- T6-5.2 **错误可见化**（T6-1.3 已改 4 处 + 本刀补齐）：whisper_handler 记忆写协程错误全部经
  recordMemoryWriteError 汇聚（slog.Error + 计数）；
- T6-5.3 **异步写可观测**：whisperWriteErrors 计数器（count/最近错误摘要/时间）+ MemoryWriteErrorSink
  透传（LLM 失败 llm_extract / JSON 解析 json_parse / panic / persist 落库失败四类 phase 全覆盖）；
  persist 协程提取 persistStateAsync，persist 系列函数改返回 error（errors.Join 聚合，不再 _ = 吞错）；
  新增 5 测试；
- T6-5.4 **成人模式决策成文**：新增 docs/ADULT_MODE.md（六节：决策/理由/接口现状/前端实证/商用化恢复
  5 步方案/代码位置）；orchestrator.go:91 注释引用文档；**删除 WhisperSetAdultMode 死接口**
  （前端零引用、实现静默忽略参数；gen_bindings 457 方法重新生成 + TestBindingsCompleteness PASS）；
- T6-5.5 **db 细节收敛**：GetDatabase 签名改 (*sql.DB, error)（12 个 repos 文件 58 处调用点适配）；
  PRAGMA 单一来源（删除重复循环，DSN 唯一，grep 断言各恰 1 次）；V11 FTS 全量重建失败补 slog.Error；
  新增 4 测试；
- T6-5.6 **陈旧占位清理**：删除 4 个占位文件（plan_document_intent/paper_card_companion/
  desktop_mode_policy/desktop_opening——目标文件已核实或不存在），grep 零引用；
- 验证：internal/whisper 3 包 + internal/app 23.9s ok、vet 干净；tsc 0 errors、vitest 312/312、
  eslint 0 errors（存量 warnings）。

## v2.27.0「质量收敛 · 绘梦·链路真实生效」（2026-08-14）
> 阶段 6 第四刀（T6-4）：取消真实生效、flux 名实相符、历史图片可恢复、格式/注入面修复、核心链路测试补全。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.27.0.md。
- T6-4.1 **取消真实生效**（image_handler.go/image_comfyui.go）：
  - CancelImageGeneration 在 cancel context 之外调用 POST /interrupt 中断 ComfyUI 当前任务；本地取消标记拒绝后续排队提交
    （ComfyUI 无删除排队任务 API，/queue 仅查询——注释说明）；checkHistory 携带 ctx、取消后轮询即刻退出；取消幂等；
- T6-4.2 **flux 名实相符**：文生图改显式映射表 txt2imgWorkflows（krea2/z-image-turbo/flux），未知模型返回中文错误
  （静默降级已消除）；flux 实现真实工作流（UNETLoader flux1-schnell + DualCLIPLoader type=flux + ae VAE +
  4 步 KSampler）；img2img 白名单化；
- T6-4.3 **历史图片可恢复**：imageItem 增 file_path 字段，生成流程把落盘路径写入历史；前端历史分级存储
  （>200k base64 只存 path）、挂载时经 GaeaAttachmentDataURL 回填恢复、下载/剧照优先 file_path；
- T6-4.4 **尺寸解析校验**：parseSize 弃 Sscanf 改 strconv.Atoi 严格解析，非法输入中文报错，钳制 64–2048；
- T6-4.5 **端口命令注入修复**：findProcessByPort 弃 cmd/findstr 拼接改 exec.Command("netstat","-ano") 参数数组
  + 输出解析；端口白名单 1–65535；
- T6-4.6 **核心链路测试补全**：ComfyUI 客户端 httpClient/pollInterval 可注入；httptest 五链路（提交 3/轮询 4/
  取消 4/上传 3/下载 2）+ 全链路端到端；Go 新增 31 用例；前端 media/historyMeta/queue 纯逻辑模块 + 19 用例；
- 验证：internal/ai 4.2s + internal/app 23s ok、vet 干净；tsc 0 errors、vitest **312/312**（70 文件）、
  eslint 0 errors（761 存量 warnings）。

## v2.26.0「质量收敛 · 对话·流可靠」（2026-08-14）
> 阶段 6 第三刀（T6-3）：流订阅竞态与超时、落库错误透传、语音持久化、迁移一次性、导出转义。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.26.0.md。
- T6-3.1 **流订阅竞态与超时**（ChatPage.tsx）：
  - runID 一到即在同一微任务注册 EventsOn（零异步间隙，首帧不丢）；30s 无帧超时（STREAM_SILENCE_TIMEOUT_MS）→ sending 复位 + 错误展示 + finally 必执行；
  - finish 幂等收尾覆盖 done/error/超时/启动拒绝/卸载五路；新增 12 个组件测试（fake timers）。
- T6-3.2 **落库错误透传**（internal/app/chat_service.go）：
  - appendChatExchange 返回 error；流式路径落库失败 emit error 终态而非 done（前端可见失败）；
  - ChatTopicsList/ChatMessagesList 签名改返回 error（绑定签名变更，前端 try/catch + LogFrontendError 同步）。
- T6-3.3 **语音持久化**：新增绑定 ChatAppendMessages（单事务批量落库）+ 前端语音识别/回复落库（不走 ChatSend 无重复）；朗读 URL revoke（onended/onerror/play 失败/卸载）；打字循环取消标志（切话题/卸载中止）。
- T6-3.4 **迁移一次性**：migrateLegacyTopics 加持久化标记（gaea_chat_migration_v1），成功才写；失败记日志不静默、保留旧键、会话内不无限重试；ChatImportTopic 改单事务（ImportTopicTx 全成或全回滚）。
- T6-3.5 **导出转义**：ChatTopicExportMarkdown 消息原文转义 Markdown 敏感字符（行首井号/反引号/尖括号/竖线）；sanitizeChatFilename 加固（Windows 保留名 CON/PRN/AUX/NUL/COM1-9/LPT1-9、尾部点号、截断 40、空→chat）；ChatGeneral 补主脑派发依赖注释（不删除）。
- 验证：test-all.ps1 全量包 ok、vet 干净；tsc 0 errors、vitest **290/290**（67 文件）、eslint 0 errors（762 存量 warnings）。

## v2.25.0「质量收敛 · 办公引擎·正确性」（2026-08-14）
> 阶段 6 第二刀（T6-2）：docmd 分页修复、TurnResult 语义、后端看门狗、Send 排队、禁写注册表化、TCCA/evidence 补测。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.25.0.md。
- T6-2.1 **PDF 页数统计与分页过滤修复**（internal/office/docmd/docmd.go）：
  - 页数统计改 countPDFPages 精确匹配（排除 /Type /Pages 页树干扰，修复总页数恒多 ≥1）；
  - BT..ET 文本按页对象归类（页码由页对象决定而非 BT 块自增），页范围过滤不再错位；
  - OCR 循环改绝对页码（修复 pdftoppm 从 first>1 渲染时范围错位一页），OCR 范围与"已截断"提示同源；
  - 新增 pdf_pages_test.go 7 测试（构造最小合法 PDF fixture，含 /Pages 干扰/无空格/页树边界/页内多 BT 块）。
- T6-2.2 **运行链路结果语义修复**（internal/gaea/agent/agent_run.go、agent_stream.go）：
  - TurnResult 的 blocked/precheck blocked/suppressed/tool panic 计入 Errors，Success 仅整轮无错误为 true；
  - 终止流错误路径先写已收部分文本入会话再返回 err（不丢已生成内容）；
  - step-- 加下限 0，杜绝负 step 与 grace 边界组合出额外模型轮；新增 10 测试（Success 语义收紧不影响上层：controller 丢弃 TurnResult、前端无引用）。
- T6-2.3 **TCCA 与 evidence 补测**（internal/gaea/context、internal/gaea/evidence，仅新增测试零实现改动）：
  - 新增 58 个测试：context 覆盖率 39.2%→**97.0%**、evidence 91.1%（要求 ≥60%）；
  - 记录两处观察（不改实现）：MergeChild 的 ForkCount +1 语义存疑；CacheReport 不聚合子项 CacheHitTokens/CacheMissTokens/BreakCount（子代理全会话命中统计在父报告丢失）。
- T6-2.4 **落地后端看门狗**（internal/gaea/control/watchdog.go 新增）：
  - v2.13.0 声称的看门狗此前未落地（仅前端 30s 定时器）——本实现为进程内运行态看门狗；
  - 墙钟 10min / 停滞 30s 默认阈值（Options.Watchdog 可配置，==0 默认、<0 禁用维度）；
  - 触发走该回合 cancel（与用户 Cancel 同一中断链路）→ Emit TurnDone(Err) + 用户可见 Notice；
  - watchdogSink 观察推进：工具执行在途（ToolDispatch→ToolResult）与审批/提问等待豁免停滞，不误杀长任务；
  - 新增 8 测试 + 3 子测试；与 Send 队列共存回归通过（修复跨回合 channel 复用竞态）。
- T6-2.5 **Send 排队 + 禁写清单注册表化**（internal/gaea/control/controller.go、internal/gaea/tool、internal/gaea/agent/task.go）：
  - 运行中 Send 改限长队列（8 条），回合结束按 FIFO 排空（running 保持 true）；队满拒绝并发明确错误 notice；
  - 子代理禁写清单由工具注册表 PersistWrite 标记自动推导（删除手写 6 项 map），新持久化写工具加标记即自动纳入；
  - 禁写集合与 hardAskTools 完全一致（测试断言）；新增 8 测试 + 更新 2 测试。
- T6-2.6 **docmd.go 拆分**（1521→56 行，拆为 office.go/pdf.go/ocr.go/pagespec.go）：
  - 按职责纯搬迁、行为零变化（声明段重拼与原文逐字节一致验证）；23 测试全绿，覆盖率 39.2% 与拆分前一致（OCR/外部工具路径无环境跳过为既有状态）。
- 验证：test-all.ps1 **109/109 包 ok**、go vet 干净；tsc 0 errors；eslint 0 errors；vitest 274/274。

## v2.24.0「质量收敛 · 基础层·可靠性」（2026-08-14）
> 阶段 6 第一刀（T6-1）：SSE 流式加固、前端错误可见性、后端吞错清理。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段6-质量收敛.md；详见 releases/v2.24.0.md。
- T6-1.1 **SSE 流式加固**（internal/ai/client.go）：
  - 行上限 64KB → bufio.Reader 任意长行（1.2MB 单行实测不断流、逐字一致）；
  - 连接错误/5xx 指数退避重试（默认 2 次 1s/2s；200 流开始后不重试防重复生成；401 走刷新、include_usage 400 降级整体重试）；
  - 空闲超时 60s（每次读重置计时；取消作用于 streamCtx 解除阻塞读）；
  - 代理接入：与 web_fetch/web_search 同源读取代理配置（netclient.NewHTTPClient），localhost/回环强制直连（herdsman/ComfyUI 不走代理）；
  - 修复解析协程 send 守卫用调用方 ctx（防超时后丢失错误块）；新增 client_reliability_test.go 8 测试。
- T6-1.2 **前端错误可见性**（bridge.ts/store.ts）：
  - BridgeError/normalizeError/invoke 统一入口/logFrontendError；app proxy 全部方法包 invoke 层（LogFrontendError 防递归）；
  - store.ts 8 集群 14 处静默 .catch(()=>{}) 改 logBridgeError（状态逻辑零改动）；
  - 新增 bridge.test.ts 6 用例 + store.test.ts +3。
- T6-1.3 **后端吞错清理 + 日志脱敏**：
  - ChatTopicsList/ChatMessagesList 读错记日志（签名未改）；whisper_handler 两处 _= 记日志；
  - memory_ingest/memory_consolidator LLM/解析失败 slog.Error；config.Load 坏 JSON slog.Error（签名未改）；
  - main.go 桥接 token 日志脱敏 maskToken（尾 4 位）；
  - 全库 _= 扫描补 task_plan_store/characterlib_handler/gaea_ui 三组；新增 13 测试。
- 验证：test-all.ps1 109/109 包 ok、go vet 干净；tsc 0 errors、vitest 274/274、eslint 0 errors（749 存量 warnings）。
- 明确不做（留后续刀）：绑定签名变更（T6-3）、emit done 语义（T6-3）、config 原子写（T6-9.4）、非流式重试/useController 降级 catch（T6-10）。

## v2.23.0「运行纵深 · 进料与质量」（2026-08-14）
> 阶段 5 第三刀（T5-5 + T5-6）：成本库进料闭环、检索统一与质量回归。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段5-运行纵深.md；详见 releases/v2.23.0.md。
- T5-5 成本库进料闭环（唯一能力补全项，属既有成本库模块内）：
  - **PDF/图片报价单本地识别入表**（GaeaCostImportVisionPreview）：PDF 文本提取（docmd）→
    扫描件本地 OCR（OvisOCR2→Windows OCR 兜底）→ 表格线启发式解析（名称/规格/单位/价格表头，
    回退整行解析）；AI 字段归一化走本地通道（sensitive_local 强制本地，不可用降级规则解析并注明）；
    复用候选预览确认流程（无确认不落库不变），Preview.source 标记识别来源（pdf_text/pdf_scan/image）；
  - **供应商比价**（GaeaCostCompare）：库内现价/价格源抓取候选/历史快照三源聚合 + 相对现价
    跳幅 diffPct（复用 DetectAnomalies 算法）；前端比价弹层（CostCompareModal：来源/期数/价格/
    跳幅着色 ≥20% 红 / >5% 琥珀，空态提示）；
- T5-6 检索统一与质量回归：
  - **统一检索入口**（GaeaUnifiedSearch）：一次调用同时出关键词全文 + 跨库语义（已跨
    cost/knowledge/office/file）两组结果；办公搜索面板「跨库」模式单框两段展示；
    原绑定收敛为共享实现委托（单一来源）；
  - **检索质量受控测评**（GaeaRetrievalEvalRun）：真实业务查询集（docs/retrieval-eval-set.md，
    12 条造价/工程域查询 + 19 个预期命中标注）→ 逐条 Recall@10 → 汇总平均，门槛 0.8；
    模型中心「检索质量」区一键运行 + 逐查询明细表；补上「受控测评只测速度不测召回」缺口；
- 测试：检索测评 4 + 统一检索 3 + 识别/比价多组 + 既有检索回归 8/8；Go 全量 90/90 包 ok、
  vet 干净、gen_bindings 457 方法 → 10 门面；前端 tsc/eslint 0 errors、vitest 265/265；冒烟通过

## v2.22.0「运行纵深 · 速度与韧性」（2026-08-14）
> 阶段 5 第二刀（T5-3 + T5-4）：本地模型调度纵深、中断续跑。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段5-运行纵深.md；详见 releases/v2.22.0.md。
- T5-3 本地模型调度纵深：
  - **保活 keep-warm**：每 5 分钟对 catalog Running 的本地模型发轻量 SSE 探针（max_tokens=8，
    防 herdsman 卸载空闲模型）；探针失败自动降级跳过直至重新运行；开关持久化
    （keep_warm_enabled，模型中心「本地调度」设置区）；
  - **启动自动预载**：启动后后台预载功能绑定（gaea→office→chat 优先级）第一个 herdsman 模型
    （installed 且未 running 时 start --wait），首次对话免冷启动；开关 auto_preload；
  - **换模预计等待**：GaeaModelSwitchEstimate（hot/cold/download/unknown 四态，cold 提示实测
    约 15-20 秒），前端模型切换器选本地模型时弹确认；
  - **KV 缓存命中率 KPI**：gaea 侧调用记录上报 cache hit/miss token（DeepSeek/OpenAI 两种
    usage 风格归一，未上报时不污染命中率），「本地 vs 云端」统计卡新增缓存命中率
    （全局 + 云端/本地拆分）；
- T5-4 中断续跑：
  - 任务级（v2.21.0 已交付）；**agent 会话级**：每轮 turn 在 session sidecar
    （<session>.state.json）标记 running/完成 + 最后进度摘要；进程被杀残留 running=true →
    会话列表「未完成」徽标；恢复会话自动注入「上次中断于 <摘要>，请先总结进度再继续」
    并清除标记（含启动自动恢复路径）；
  - **轻语长流程**：任务计划持久化到 whisper_data/task_plan.json（重启不丢），
    WhisperTaskPlanStatus/Resume 恢复入口；
- 测试：session state 5 + controller 中断 4 + app 注入 5 + schedule 14 + stats 缓存 3 +
  overview 命中率 2 + whisper taskplan 10 + config 开关 2；Go 全量 90/90 包 ok、vet 干净、
  gen_bindings 453 方法 → 10 门面；前端 tsc/eslint 0 errors、vitest 255/255；冒烟通过

## v2.21.0「运行纵深 · 调度与异步化」（2026-08-14）
> 阶段 5 第一刀（T5-1 + T5-2）：通用任务调度器 + 批处理队列、实时文件监听。
> 规划：docs/superpowers/plans/2026-08-14-gaea长期规划-阶段5-运行纵深.md；详见 releases/v2.21.0.md。
- T5-1 通用任务调度器 + 批处理队列：新包 internal/gaea/tasks（Hephaestus.db SchemaV8 tasks 表）——
  状态机 queued→running→succeeded|failed|cancelled、进度 0-100 + 消息、取消（context 传播）、
  自动重试（指数退避）、手动重试、**重启续跑**（Startup 恢复 running→queued 重新排队）；
  进度事件经 gaea-task 通道实时推送（节流 400ms，终态必达）；
  App 绑定 GaeaTaskList/Cancel/Retry（446 方法 → 10 门面，gen_bindings 重新生成 + 完备性测试）；
  **价格抓取全异步化**（单源/一键全部/30 分钟定时 cron 全部走任务队列，同源去重、逐源进度、
  失败明细在任务结果；顺带修复 SaveFetch 按值拷贝致返回记录 ID 恒为空的历史缺陷）；
  **文件索引重建异步化**（分批 Ensure 进度、末批 Stale 清理、手动/轮询/监听共用队列去重）；
  办公右栏新增「任务」Tab（任务中心：活动/历史分组、进度条、取消/重试、失败原因）；
- T5-2 实时文件监听：新包 internal/gaea/filewatch（fsnotify 监听工作区目录树，2s 去抖合并输出
  变更/删除批次；目录级变更与事件风暴>50 标记全量重建；监听异常记录 WatchErr 回退轮询）；
  增量索引（删除直接清向量 semantic.Remove、变更内容感知重嵌、失败自愈全量重建）；
  10 分钟轮询降级兜底（监听健康时跳过）——**新文件秒级可搜**；
- 测试：tasks 包 13 组 + filewatch 包 5 组 + App 层 6 组（单源任务流/同源去重/一键抓取/
  List-Cancel-Retry 链路/定时到期跳过/cron 去重）；Go 全量 90/90 包 ok、vet 干净；
  前端 TaskCenter/价格面板/搜索面板用例 + tsc/eslint/vitest 全绿

## v2.20.1「数据可迁移·独立审查修复」（2026-08-14）
> 对 v2.20.0 变更面做独立子代理代码审查，修复 3 高危 + 4 中危 + 多项低危缺陷。
> 详见 releases/v2.20.1.md。
- 高危 #1：ApplyPending 部分失败后重试必失败 → 重构为两阶段幂等（先移走当前数据、再应用 staging，src 缺失视为已应用跳过），重试可成功且不破坏数据
- 高危 #2：home-config 从不恢复（被主循环 rename 走）→ 主循环排除 home-config 单独处理；修复 HomeConfigRel 缺 . 前缀导致的恢复到错误文件名
- 高危 #3：SQLite 快照连接无 busy_timeout + 静默回退复制缺 WAL → 加 _busy_timeout=5000 + 重试；回退改为 checkpoint 后复制；manifest 增 Warnings 告警字段
- 中危 #4：恢复失败不可见 → 失败路径也写 .restore-result.json（前端失败告警可达）
- 中危 #5：已有 pending 时再次 Restore 可堆叠/覆盖 → 拒绝并提示先取消；staging 加随机后缀防同秒撞名；Cancel 清理孤儿 staging
- 中危 #6：dirSize 全量递归阻塞 UI → 目录大小缓存（mtime + TTL 失效）
- 中危 #7：宣称可回滚但无回滚代码 → 新增 GaeaDataBackupRollback（.restore-before 移回）+ 前端失败告警「回滚到恢复前」按钮
- 低危：#8 盘符路径拒绝、#9 恢复二次确认 + zip 校验、#10 备份文件名防同秒覆盖、#11 before 保留 2 份、
  #12 WritePending 原子写、#13 Extract 两阶段、#15 shouldSkip 精确化、#16 pending 错误透出、#17 entries 数组校验
- 测试：backup 包 9 组（新增重试幂等/home-config 恢复/盘符拒绝）、App 层 5 组（新增已有 pending 拒绝/Rollback）；
  go 全量 ok、tsc/eslint 0 errors、vitest 251/251

## v2.20.0「个人使用收口·数据可迁移」（2026-08-14）
> 长期规划阶段 4 按「个人使用、不商用」重新定标（用户 2026-08-14 决策）：
> 删除商用分发项（安装器/自动更新/代码签名），聚焦个人使用最需要的
> 数据可迁移（一键备份/恢复）、模块收口（微信 beta/移动端冻结）、磁盘治理（保留 5 版约定）。
> 详见 releases/v2.20.0.md。
- P4-3 数据可迁移：设置页新增「数据」分类——一键备份（Hephaestus.db 记忆/知识/成本/语义向量 +
  whisper_data 轻语/办公/角色库/聊天 + config.toml + sessions + home 配置 → zip + manifest；
  SQLite 用 VACUUM INTO 一致性快照，运行中备份安全）；从备份恢复（两阶段：校验解压 staging +
  写 pending 标记 → 重启后自动应用，应用前先自动备份当前数据到 .restore-before-<时间>，失败可找回）；
  恢复结果提示 + 待应用告警与取消
- P4-1 模块收口：微信通道标注「个人使用实验功能（beta）」；移动端访问标注「已冻结」
- P4-2 发布形态简化：删除安装器/自动更新/代码签名（SAC/SmartScreen）等商用项；升级 = 替换 exe +
  数据备份先行（数据在用户目录不受影响）
- P4-4 磁盘治理：releases 保留最近 5 版约定明确化（README 版本表）
- 测试：Go backup 包 6 组（打包/解压往返、VACUUM INTO 快照数据完整性、manifest 校验、
  zip-slip 防穿越、pending 应用往返、SHA256）+ App 层 3 组（Info/Create/Restore/Pending/Cancel 链路、
  拒绝非法 zip、Startup 钩子）；前端 DataPanel 4 组 + SettingsPage 更新（新增「数据」分组断言）
- 个人使用声明：数据全部在本机；API 凭证（DPAPI 加密）跨机器不可解密，换机需重填

## v2.19.0「数据与成本纵深·补测评缺口」（2026-08-14）
> 长期规划阶段 3 第二刀（D3-4）：受控测评补上「看得见的缺口」——
> 报告增加每模型/长上下文/缓存复用/显存参数专项分析；新增压力专项任务
> 预设与「快速流式探针」（断流/卡顿观察）。详见 releases/v2.19.0.md。
- D3-4 报告模板化：Markdown 报告新增 5 个专项段落——每模型对比（同任务横向可比）、
  长上下文专项（TTFT vs context_size）、缓存复用专项（first vs second TTFT +
  prefill 加速比）、显存相关启动参数（effective_launch_params 关键字段）、并发专项说明
- D3-4 压力预设：受控测评任务集新增「压力·长上下文 / 压力·长输出 / 压力·显存
  （长上下文+长输出）」3 项（配合上下文 4K~32K / 并发 1/2/4 / max_tokens 使用）
- D3-4 流式探针：GaeaBenchmarkStreamProbe 对模型发起真实 SSE 请求，观察 TTFT、
  分块数、最大/平均分块间隔（卡顿指示）、是否正常 [DONE] 收尾（断流检测）；
  模型中心「受控测评」新增「快速流式探针」区（每已安装模型一键探测）
- 测试：Go 新增流式探针 3 组（SSE mock/参数校验/HTTP 错误）+ 报告专项分析 1 组；
  前端 BenchmarkSection 新增探针用例（4/4）；全量 vitest 247/247
- 真实端到端：对运行中 herdsman 的模型探测成功（冷启动 TTFT 15.2s、60 块、
  max_gap 83ms、正常收尾）

## v2.18.0「数据与成本纵深·首轮」（2026-08-14）
> 长期规划阶段 3（D3-1 ~ D3-3）：跨库统一语义检索补齐「资料」+ 索引状态、
> 本地 vs 云端分流统计与节省对比、Herdsman 受控测评产品化（一键发起/明细/报告导出）。
> 详见 releases/v2.18.0.md。
- D3-1 持久化向量索引：跨库统一语义检索（GaeaSemanticSearch）并入工作区资料（kind=file，
  复用文件索引定时维护的持久化向量，检索不扫描）；新增 GaeaSemanticIndexStatus（各库向量
  条数，记忆中枢可见索引健康度）；semantic.Store 新增 Counts
- D3-2 分流统计面板：GaeaUsageOverview 打通 gaea 侧调用记录（含费用估算）与 herdsman
  events.jsonl 本地遥测——本地 token 口径 = events 全量 + 其他本地引擎（herdsman 不重复计）；
  模型中心「调用统计」新增「本地 vs 云端·节省对比」卡（云端实际混合单价折算，无云端用量
  回退 deepseek-v4-flash 官价）
- D3-3 测评产品化：复用 herdsman /api/benchmarks（多模型 × 变体 × 上下文长度 × 并发，
  逐 case TTFT/TPS/token）——GaeaBenchmarkList/Start/Detail/Export + 模型中心「受控测评」
  分类（任务预设蒸馏自 120 组对照测评方法学、上下文 4K~32K 覆盖长上下文、并发 1/2/4、
  Markdown 报告导出含逐用例明细）；真实 herdsman 端到端验证（发起 202 + 完成 succeeded）
- 测试：Go 新增分流口径 3 组 + 测评 runs 解析/HTTP 列表发起/导出 4 组 + semantic Counts；
  前端 BenchmarkSection vitest 3/3；internal/app 全量 ok、tsc/eslint 0 errors、vitest 246/246

## v2.17.0「安全与架构收敛」（2026-08-14）
> 长期规划阶段 2（S2-1 ~ S2-4）：安全收敛与绑定面架构拆分——
> LAN 暴露告警上墙、WebView2 远程调试默认关闭、HTTP 桥接一次性 token、
> 敏感数据本地通道、429 个导出方法按板块拆 10 个绑定门面。
> 详见 releases/v2.17.0.md。
- S2-1 LAN 风险处置：全局安全横幅（启动即检测 herdsman api.lan_accessible，暴露时醒目告警 +
  中文处置指引 + 重新检测/本次忽略）；设置页新增「安全」分类（同面板可复核）
- S2-2 gaea 自身安全开关：WebView2 远程调试（9333）改 `GAEA_WEBVIEW_DEBUG=1` 才开启，默认关闭；
  HTTP 调试桥接加一次性 token（`GAEA_HTTP_TOKEN` 或每进程自动生成并打日志，/api/rpc 与
  /api/stream 须携带 Bearer/X-Gaea-Token/?token=，/api/health 保持开放），前端桥接自动透传
- S2-3 App 绑定面拆分：429 个导出方法按板块拆 10 个绑定门面（CoreB/OfficeB/MemoryB/CostB/
  ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB），方法体零改动纯委托，脚本生成
  （scripts/gen_bindings）+ 反射完备性测试兜底（App 方法集与门面并集全等）；
  前端 gaea/lib/bridge.ts 与 api/bridge.ts 单点路由，旧调用路径经 wailsjsCompat 兼容层零改动
- S2-4 敏感数据本地通道：新增「敏感域本地化」开关（默认开启，~/.gaea_config.json
  sensitive_local 持久化）——成本/报价类 AI 操作（GaeaCostImportAIParse）默认强制路由本地
  Herdsman（数据不出本机），引擎不可用自动回退常规路由；设置页「安全」分类可切换回云端
- 测试：internal/app 全量 19.8s ok（含绑定完备性）；config/httpbridge 新增 token 鉴权、
  sensitive_local 往返用例；tsc -b 通过、eslint 0 errors、vitest 243/243、vite build 通过

## v2.16.1「模型中心资源协同 + 磁盘治理」（2026-08-14）
> 长期规划 E1-4：对齐 herdsman `model_scheduling.local_concurrency=1` 的调度现实，
> 生命周期操作串行化（批量启停天然变有序队列）；模型库新增磁盘 KPI
> （已装占用 + 数据目录所在卷余量）。详见 releases/v2.16.1.md。
- 后端：`herdsmanOpMu` 串行化 Start/Stop/Download/Uninstall（下载最长 60 分钟、冷启动 20 分钟，
  并发发起会互相冲突）；`herdsmanDiskInfo`（x/sys/windows GetDiskFreeSpaceEx，可注入替身测试）+
  HerdsmanCatalog 新增 installed_bytes/disk_total/disk_free/disk_error，探测失败不阻塞目录
- 前端：模型库 KPI 新增「已装空间」「磁盘余量」（余量/总量，含探测失败提示）；fmtSize 补 TB 档
  （此前 1TB 显示为 1024.0 GB）
- 测试：磁盘解析/汇总/降级 3 组 + 操作串行化并发验证（8 goroutine × 4 操作，最大在飞必须为 1）；
  模型库 vitest 5/5；Go 变更面全绿

## v2.16.0「Herdsman 底座加固 + 工程门禁」（2026-08-14）
> 长期规划首轮（docs/superpowers/plans/2026-08-14-gaea长期规划-herdsman底座加固与工程门禁.md）：
> gaea 的本地能力链（聊天/视觉/embedding/rerank/OCR/文档解析/ASR/TTS/生图/翻译）
> 全部挂在 Herdsman 服务上，本轮把它变成「可探测、可探活、可告警」的受管底座，
> 同时修复前端 CI 门禁（此前 lint 因插件缺失直接崩溃、continue-on-error 形同虚设）。
> 详见 releases/v2.16.0.md。
- H0-1 环境探测与兼容契约：`internal/herdsman/probe.go` + `App.HerdsmanProbe`——一次探测
  config.yaml（api 段）、herdsman.exe CLI 可找到性、/v1/models 可达性（HERDSMAN_PROBE_LIVE=1 时真实探测）、
  四个数据契约（launch_records/model_stats events.jsonl/skill-operations.json/models），
  输出结构化 Probe + 中文告警清单（含 LAN 暴露、端口漂移、契约缺失）
- H0-2 服务健康检查：`internal/herdsman/health.go` + `App.HerdsmanHealth`——端口拨测（1s）+ API 存活（3s）+
  按能力归类已装模型（chat/vision/embedding/rerank/ocr/parse/asr/tts/imagegen/translation），
  Healthy=端口+API+聊天模型齐备，Summary 中文问题清单
- H0-3 TTS 默认模型动态解析：`voice.ResolveHerdsmanTTSModel`——配置值已装则用配置值，
  否则按优先级（voxcpm2 第一，本机实测唯一可用本地 TTS）从已装列表解析，
  VoiceGetSettings 返回回退标记供前端提示；修复「默认 qwen3-tts-customvoice 未安装必然先失败」
- H0-4 LAN 暴露检测与告警：`internal/herdsman/lancheck.go` + `App.HerdsmanSecurityCheck`——解析
  herdsman config.yaml 的 api.lan_accessible/port（逐行 YAML 解析，零依赖），暴露时返回中文处置指引；只提示不改配置
- H0-5 模型用途建议 + 思考模式守护：模型库卡片新增「用途建议」（`herdsmanModelHint`，依据 120 组受控测评：
  HauhauCS 日常/识图首选、LynnStyle 可审计推理、Hermes 勿开思考、voxcpm2 冷启动 50s 等）；
  本地引擎思考模式 max_tokens <4096 自动抬到 4096（bridge 与聊天流式两路径），杜绝「只有推理、无正文」
- E1-1 前端 CI 门禁修复：eslint 配置修复（react-hooks v5 flat 配置失效 + 缺 eslint-plugin-react-hooks/refresh、
  新增 lint script、globalIgnores 生成目录）；28 个硬错误清零——含 Lightbox 12 处条件调用 Hook 的真实隐患
  （顺带修复该场景下 React 运行时崩溃风险）、CreatePage case 声明、Markdown 控制字符正则等；
  存量高频风格规则（no-explicit-any 等 6 项）降为 warn 随迭代清理；CI 移除 continue-on-error、
  npm ci→npm install（仓库不提交 lockfile）、新增 vitest 步骤；修复 ComfyUI 路径文案反斜杠丢失（用户可见）
- E1-2 发布冒烟脚本：`scripts/smoke.ps1`——产物启动 + HTTP 桥接 /api/health 探活 + 进程存活 + 自动回收
- E1-3 版本节奏：本轮起转周版本，v2.15.7 后直接 v2.16.0
- 测试：internal/herdsman 全包单测（probe/lancheck/health/归类函数）；Go 相关包全绿（internal/app 22s 全量、
  bridge/ai/voice/tts）；tsc -b 通过、eslint 0 errors（725 warnings 为存量风格项）、模型库卡片 vitest 5/5

## v2.15.7「通用办公 P0 · 开工前计划卡片结构化」（2026-08-13）
> 接回通用办公优化路线图，把 v2.10 的「开工前计划确认」升级为结构化计划卡片：
> 计划生成改走严格 JSON，后端解析为「任务理解 / 步骤（资料·工具·产出物）/ 待确认」，
> 前端渲染专属计划卡片，解析失败自动回退纯文本。详见 releases/v2.15.7.md。
- 后端：planSystemPrompt 改为严格 JSON；新增 agent.ParsePlan / RenderPlanMarkdown（容错代码围栏、清洗空字段）；
  Ask 事件新增可选 Plan 结构化载荷（controller_plan 下发、gaeaEventMap 序列化），答案协议不变
- 前端：WireAsk 新增 plan 载荷；AskCard 渲染 PlanBody（目标高亮卡 + 步骤编号卡 + 资料/工具/产出物芯片 + 待确认琥珀提示），无 plan 回退 Markdown
- 测试：Go 新增 ParsePlan 4 例 + Ask.Plan 随事件下发 1 例；前端 AskCard 2 例（结构化/回退）；go vet + go test 全绿，Vitest 241→243，tsc/vite build 通过

## v2.15.6「Herdsman 深挖 P5 · 数字生命记忆联动 + 最近操作」（2026-08-13）
> 完成 Herdsman 深挖路线图收尾：把 digital-life 虚拟人格记忆（角色/关系/记忆摘要/
> 时间线/世界事件）只读接进记忆中枢，并展示 Herdsman 最近异步操作。
> 详见 releases/v2.15.6.md。
- 后端：App.HerdsmanDigitalLife（只读 life.sqlite3：角色×关系×记忆摘要合并、计数、最近时间线/世界事件）+ App.HerdsmanOperations（skill-operations.json 最近 20 条）
- 前端：记忆中枢新增「数字生命」库（角色卡片：亲密度/信任/安全条 + 摘要/高亮/强化值；最近时间线/世界事件；最近 Herdsman 操作列表）
- 测试：Go 新增数字生命/操作解析用例；前端 DigitalLifeLibrary 2 例，Vitest 239→241 全绿

## v2.15.5「Herdsman 深挖 P4 · 检索升级 + 调用统计」（2026-08-13）
> 承接 v2.15.4，落地 P4：语义检索动态升级到 qwen3-embedding-4b / qwen3-reranker-4b
> （装了自动用，没装回退 bge），并把 Herdsman 逐请求遥测接进模型中心。
> 详见 releases/v2.15.5.md。
- 检索升级：resolveHerdsmanSearchModel 动态选模型（env > qwen3 系已装 > bge 回退），覆盖成本库/知识库/办公记忆/工作区文件语义索引
- 调用统计：App.HerdsmanModelStats 聚合 model_stats/events.jsonl（调用/成功失败/token/耗时/TTFT/TPS），模型中心「模型库」新增本地统计面板（KPI + 明细表）
- 本机：qwen3-embedding-4b（2.5GB）与 qwen3-reranker-4b（2.7GB）已下载并启动；发现 Hy-MT1.5:1.8B 翻译模型已装，translate_text 自动切专用模型
- 测试：Go 新增 stats 解析/动态选型用例；前端统计面板断言；go vet + go test + tsc + Vitest 239 全绿

## v2.15.4「Herdsman 深挖 P3 · 本地翻译」（2026-08-13）
> 承接 v2.15.3，落地 P3：本地翻译能力——优先 Hunyuan-MT / Hy-MT 翻译模型
> （capability=translation），未安装时回退「常规办公」模型，本地/免费优先。
> 详见 releases/v2.15.4.md。
- 后端 `App.LocalTranslate`：翻译模型发现（hunyuan-mt/hy-mt）+ 显式 model + 回退常规办公模型（used_fallback 标注）+ 可读错误引导；文本翻译走 /v1/chat/completions（/v1/translations 是语音翻译）
- 办公专业工具 `translate_text` 注入 ExtraTools，能力面板「本地专业模型」新增入口
- 测试：Go 新增 6 例（模型命中/显式/回退/空文本/发现/工具执行），go vet + go test 全绿；tsc + Vitest 239 全绿

## v2.15.3「Herdsman 深挖 P2 · 模型生命周期管理」（2026-08-13）
> 承接 v2.15.2 模型库，本轮把「看」升级为「管」：模型卡片直接启动/停止/下载/卸载
> Herdsman 模型，并读取 launch_records 生成这台机器的启动参数预设。
> 详见 releases/v2.15.3.md。
- 后端：HerdsmanModelStart/Stop/Download/Uninstall（skill models 子命令，--wait 长超时）+ HerdsmanLaunchPresets（读 launch_records 取最近成功启动参数）+ HerdsmanOpResult 统一结果
- 前端：模型库卡片操作（运行中→停止；已安装→启动+卸载二次确认；未安装→下载），操作中 loading + 完成后自动刷新；有 launch_records 的模型显示「启动预设」徽标（悬停见参数明细）
- 测试：Go 新增操作结果/预设解析与生命周期 handler 用例；前端新增生命周期 1 例，Vitest 238→239 全绿

## v2.15.2「Herdsman 深挖 P1 · 模型库」（2026-08-13）
> 启动 Herdsman 深挖路线图（docs/superpowers/plans/2026-08-13-herdsman-deep-dive.md），
> 本轮把 Herdsman 完整本地模型目录（90 个已知模型）接进模型中心：从「只能连已存在
> 的模型」升级为「可浏览全部可安装模型与能力」。详见 releases/v2.15.2.md。
- 后端 `App.HerdsmanModelCatalog()`：调 `herdsman.exe skill models list --json`（RPC），解析 90 模型目录（能力/安装/运行/量化/变体/大小/MoE），汇总计数；HERDSMAN_EXE 环境变量优先，回退默认安装路径；CLI 缺失或 Herdsman 未运行返回可读错误，不阻塞模型中心
- 前端模型中心新增「模型库」分类：KPI（已知/已安装/运行中）+ 搜索（名称/能力）+ 状态过滤 + 类型下拉 + 模型卡片（中文名/能力/量化/大小/参数/MoE/状态），复用统一卡片与视觉 token
- 测试：Go 新增解析/排序/汇总/错误/CLI 定位用例 + 真实 90 模型回归校验；前端新增模型库 4 例，Vitest 234→238 全绿，tsc 通过

## v2.15.1「通用办公 · 产物与资料体验收口」（2026-08-13）
> 承接 v2.15.0 模型中心，回头收口通用办公的产物展示一致性与入口兜底：
> 右侧「会话产物」面板补齐图片缩略图与一键复制全部路径，Ctrl+K 命令面板补齐
> 资料/产物/变更跳转，欢迎页任务模板在命令库为空或加载失败时回退内置模板。
- 会话产物面板：图片类交付物渲染缩略图（与对话内交付卡共用 FileThumb，加载失败回退图标）；头部新增「复制全部文件路径」，一次拿到本次会话全部交付物清单
- 命令面板：新增「资料面板」「产物面板」「变更面板」跳转（与已有 文件/统计 面板项对齐）
- 欢迎页：任务模板新增内置兜底（周报/会议纪要/成本测算/方案大纲/数据分析/文档转换/报告拼装/演示文稿），首启或离线不再空白
- 测试：前端新增 5 例（产物缩略图/复制全部/模板兜底），Vitest 229→234 全绿，tsc -b 通过

## v2.15.0「模型中心 P0/P1/P2 + UI 重设计」（2026-08-13）
> 按市场调研（Open WebUI / Cherry Studio / Dify / Jan / Ollama 生态）系统优化模型中心，
> 并做浅色/深色双主题重设计。详见 releases/v2.15.0.md。
- 模型中心 P0：引擎状态→模型可见性联动（未连接模型置灰/禁用动作）；测试连接诊断（延迟+失败原因）；功能绑定回退态 + 一键重置
- 模型中心 P1：模型网格搜索 + 收藏置顶（localStorage 持久化）；本地资源占用可视化（CPU/内存/GPU/显存 + 本地引擎状态）；模型选择器统一按后端过滤 + 兜底
- 模型中心 P2：引擎批量启停 + 隐藏已停用引擎；调用统计收进抽屉
- UI 重设计（redesign-existing-projects / ui-ux-pro-max 审计）：模型卡片/空状态/面板背景与阴影改用 gaea 主题 token，适配浅色/深色
- 测试：前端 Vitest 229/229；tsc -b、vite build 通过；go build/vet、go test ./... 全绿
- 构建：wails build 成功，gaea-v2.15.0.exe 同步桌面与 releases/

## v2.14.12「绘梦 UI 重构落地 + herdsman 生图能力修复」（2026-08-13）
> 完成绘梦板块 UI 全量重构（设计文档 Phase 1-2 + 视觉统一），选项改下拉并修复
> WebView2 弹层卡首帧；herdsman 生图链路修复与图生图接入。详见 releases/v2.14.12.md。
- 绘梦：左栏可折叠分区（基础设置/模型与引擎/画幅与输出/高级参数）、底部常驻生成栏、右侧任务中心三 Tab、玻璃 HUD 视觉统一
- 绘梦：引擎/模型（可搜索）/画幅/时长帧率改下拉；WebView2 下拉弹层兜底（popupClassName 禁用弹层动画 + getPopupContainer=body）
- herdsman：size 契约对齐（文档支持 size）、URL 响应转 data URL、图生图接入 `/v1/images/img2img`（JSON + image 字段）
- 测试：前端 Vitest 204/204；tsc -b、vite build 通过；Playwright 无头断言 32 项通过；go build/vet、go test ./... 全绿
- 构建：wails build 成功，gaea-v2.14.12.exe 同步桌面与 releases/

## v2.14.11「小说板块后端 + 绘梦生成链路闭环」（2026-08-13）
> 小说创作补齐章节流式/状态/书架/世界观/统计/导出后端；绘梦补齐生成队列/取消、历史元数据持久化、
> ComfyUI 任务进度实时反馈、模板与绘照分配、LoRA 动态加载/重试；绘梦 UI 重构已立项，下个会话执行。
> 详见 releases/v2.14.11.md。
- 小说：章节流式创作/状态流转、书架、世界观（含回归测试）、统计、导出收敛
- 绘梦：生成队列/取消、历史元数据持久化、ComfyUI 实时进度、模板/绘照、LoRA 动态加载与重试
- 测试：前端 Vitest 200/200；tsc -b、vite build 通过；go build/vet、go test ./... 全绿
- 构建：wails build 成功，gaea.exe 同步桌面

## v2.14.10「修复办公模型改绑不生效（仍沿用旧模型）」（2026-08-13）
> 定位：办公主 agent 的模型由 bridge provider 在 GaeaInit 时注入并缓存，运行时
> 在模型中心改绑「办公」后，配置已更新、前端也显示新模型，但没触发重新注入 +
> 重建 controller，因此继续用旧的 deepseek。详见 releases/v2.14.10.md。
- `App.SetFeatureModel` / `App.SetFeatureModelEnabled` 覆盖：feature=="gaea" 时重新注入 bridge 模型
- 办公引擎已初始化时同步重建 controller；未初始化则仅更新注入，下次 GaeaInit 生效
- 测试：新增 `TestAppSetFeatureModel_GaeaBindingApplies`
- 验证：go build/vet、go test（chat/app）全绿；wails build 通过（前端未改动）

## v2.14.9「聊天板块后端：原子落库 + AppendMessage 收敛」（2026-08-13）
> 继续收敛聊天后端写入路径：用户/助手消息改为单事务原子落库，AppendMessage 用
> RETURNING 一次拿回 id+seq，失败不再静默忽略。详见 releases/v2.14.9.md。
- `AppendMessage`：用 `INSERT ... RETURNING id, seq` 替代「先查 MAX(seq)+1 再插入」，消除竞态窗口
- 新增 `chat.Store.AppendExchange`：单事务原子写入用户 + 助手消息并刷新 updated_at
- `appendChatExchange` 改走事务，落库失败记录错误日志（不再静默丢弃）
- 测试：新增 `TestStore_AppendExchange`（含不存在话题回滚校验）
- 验证：go build/vet、go test（chat/app）全绿；前端未改动

## v2.14.8「聊天板块后端：会话列表查询收敛 + GetTopic」（2026-08-13）
> 收敛聊天板块后端存储的查询路径：会话列表预览从 N+1 改为单条相关子查询，
> 新增 `GetTopic` 供创建/导入/导出直接按 ID 读取，不再全表列举。详见 releases/v2.14.8.md。
- `chat.Store.ListTopics` 用相关子查询一次取回所有话题的预览，消除 N+1
- 新增 `chat.Store.GetTopic(id)`；`ChatTopicCreate` / `ChatImportTopic` / `ChatTopicExportMarkdown` 改走按 ID 读取
- 测试：新增 `TestStore_GetTopic`
- 验证：go build/vet、go test（chat/app）全绿；前端未改动

## v2.14.7「聊天板块交互收尾：清空确认 + 切换聚焦 + 标题生成收敛」（2026-08-13）
> 收尾几处日常交互细节：清空对话加二次确认防误触，切换话题后自动聚焦输入框，
> 会话标题生成抽成纯函数并补测试。详见 releases/v2.14.7.md。
- 清空当前对话加 Popconfirm 二次确认（避免误清空不可恢复）
- 选中话题后自动聚焦输入框，切过去即可输入
- 会话标题生成抽纯函数 `autoTopicTitle`，前端测试 194→196
- 验证：wails build（含 tsc + vite）通过；vitest 196；go build/vet、go test 全绿

## v2.14.6「聊天板块会话导出为 Markdown」（2026-08-13）
> 补上聊天板块的内容出口：把当前会话导出为 Markdown 文件，落盘到用户数据目录，
> 前端一键导出并复制文件路径。详见 releases/v2.14.6.md。
- 后端 `ChatTopicExportMarkdown`：按话题标题 + 用户/AI 分段导出全部消息为 .md
- 文件名安全规整（`sanitizeChatFilename`），写到用户数据目录 exports/chat 下
- 前端模式栏新增「导出」按钮，成功后提示路径并复制到剪贴板
- 测试：新增 `TestChatTopicExportMarkdown`
- 验证：wails build（含 tsc + vite）；vitest 194；go build/vet、go test（app）全绿

## v2.14.5「聊天板块真实流式输出（普通对话）」（2026-08-13）
> 把普通对话从「整段返回 + 前端模拟打字流」升级为后端逐块流式下发，首字更快、
> 停顿更自然；思考链也随流式下发。角色模式保持整段返回。详见 releases/v2.14.5.md。
- 普通对话真实流式：`ChatStreamPlain` 立即返回 runID，经 `chat-stream:<runID>` 下发 delta/reasoning/done/error
- 角色模式保持整段返回（沿用原有模拟打字流），两种模式统一走发送收尾（自动命名/置顶/预览同步）
- AI 客户端新增 `ChatStreamChunks`：复用同一套请求准备，但暴露底层 SSE 分块供逐块消费
- 测试：新增 `TestChatStreamChunks_EmitsDeltas`、`TestChatStreamPlain_StreamsAndPersists`
- 验证：wails build（含 tsc + vite）通过；vitest 194 例全过；go build/vet、go test（ai/app）全绿

## v2.14.4「聊天板块收口：联网搜索污染修复 + 回到底部 + 侧栏预览同步」（2026-08-13）
> 收口聊天板块最后一处数据正确性问题与一个滚动体验缺口：联网搜索注入只进模型
> 上下文、不再污染用户历史；上翻阅读时提供「回到底部」悬浮入口；侧栏预览随发送/
> 清空即时同步。详见 releases/v2.14.4.md。
- 修复：联网搜索注入不再写入用户消息历史（原实现会把搜索结果一起落库，污染话题预览与历史）
- 回到底部：上翻阅读时出现悬浮按钮，一键回底并恢复自动跟随
- 侧栏预览同步：发送首条消息 / 清空对话后即时更新会话预览，不再等重新进入
- 测试：新增 Go 用例 `TestChatSend_Plain_SearchKeepsOriginalUserMessage`
- 验证：tsc + vite build 通过；vitest 194 例全过；go build/vet、go test（含新用例）全绿

## v2.14.3「聊天板块补强：输入法防误发 + 快速切话题竞态修复 + 模式栏收敛」（2026-08-13）
> 延续聊天板块优化：修掉中文输入法候选确认时 Enter 误发消息、快速切换话题时
> 旧话题响应覆盖当前视图两个高频隐患，并收敛模式栏重复 JSX。前端用例 190→194。
> 详见 releases/v2.14.3.md。
- 输入法防误发：中文/日文 IME 组合态下的 Enter 不再触发发送（纯函数 `shouldSubmitOnEnter`）
- 快速切话题竞态修复：话题消息载入用序号令牌，过期响应直接丢弃，避免旧消息覆盖当前视图
- 模式栏收敛：普通/角色两分支合并为单一操作区，去除重复 JSX
- 测试补强：新增 `shouldSubmitOnEnter` 用例
- 验证：tsc + vite build 通过；vitest 190→194；go build/vet、go test 保持全绿

## v2.14.2「聊天板块优化：会话搜索 + 最近活跃排序 + 智能滚动 + 重开加载修复」（2026-08-13）
> 在通用办公会话化升级之后，回头把「聊天板块」的日常交互体验补到同一档：会话按
> 最近活跃排序并显示相对时间、侧栏新增会话搜索、流式/生成时不再强制吸底、重进聊天
> 板块正确载入历史消息。前端用例 182→190。
> 详见 releases/v2.14.2.md。
- 会话列表：最近活跃优先（新会话/回复会话自动置顶），会话行显示相对时间
- 会话搜索：侧栏新增搜索框，按标题/预览/模式标签过滤（纯函数 `filterChatTopics`）
- 智能滚动：生成/流式输出时用户上翻阅读不再被强制吸底，贴近底部才恢复跟随（`isNearBottom`）
- 修复：重新进入聊天板块时，已选中话题的历史消息未载入（此前显示欢迎屏而非对话）
- 缺陷收口：会话重命名失败提示（不再静默失败）
- 测试补强：新增 `filterChatTopics` / `sortByUpdatedAtDesc` / `isNearBottom` 纯函数用例
- 验证：tsc + vite build 通过；vitest 182→190；go build/vet、go test（chat/app）全绿

## v2.14.1「办公板块缺陷收口 + 测试补强 + 结构收敛」（2026-08-13）
> 承接 v2.14.0：收口会话恢复/重命名失败提示、归档删除二次确认、欢迎页跨项目
> 最近会话、变更面板汇总排序、归档会话搜索；前端用例 138→179，办公前端做
> 首轮结构收敛，并收敛 bridge 动态/静态 import 混用告警。详见 releases/v2.14.1.md。
- 缺陷收口：会话恢复/重命名失败提示；归档会话永久删除二次确认；欢迎页最近会话跨项目；
  变更面板「N 个文件 · M 次」汇总与最近排序；侧栏搜索覆盖归档；会话删除三处注册表全量清理
- 测试补强：前端 138→179 例（store/ChangesPanel/useSessionManager/Sidebar/命令与 @ 解析/相对时间/变更汇总/项目分组搜索/能力面板摘要）
- 结构收敛：App 状态组件、buildSessionChanges、Composer 斜杠与 @ 解析、Sidebar 相对时间与搜索过滤、CapabilitiesPanel 摘要、bridge 静态 import
- 文档：herdsman API 文档更新（08-08 → 08-13）

## v2.14.0「办公板块会话化升级：项目分组 + 会话生命周期 + 任务目标 + 变更面板」（2026-08-13）
> 把通用办公从扁平会话列表升级为按项目聚合的会话工作台，补齐会话生命周期、
> 任务目标（需求→验收）与文件变更可观察性，并修复记忆图谱三元组被成本节点
> 挤掉、会话删除注册表清理不完整、App 层 toast 静默失效等问题。
> 详见 releases/v2.14.0.md。
- 侧边栏「项目」分组：当前工作区置顶，其余为最近打开且有会话的工作区；会话按“当前→置顶→最近”排序
- 会话生命周期：置顶（.pinned.json）、归档（移入 <sessions>/archive/，可恢复）、侧边栏「已归档」分组恢复
- 任务目标：会话首条用户消息自动锚定，随会话持久化，待办栏展示进行中/已验收并支持一键切换
- 文件变更面板：汇总写/改工具实际改动的文件及次数，覆盖 path/file_path/paths/edits/source+destination 等形态
- 恢复会话保留工具过程：GaeaHistory 还原 tool/tool_result，恢复后过程卡与变更面板仍可见
- 专注模式：Aim 按钮 / Ctrl+Shift+F 收起侧栏与右侧面板（状态持久化）
- 修复：会话删除三处注册表全量清理；记忆图谱轻语三元组提前于成本条目入图；App 层 toast 静默失效；恢复历史待办收尾；重复快捷键与 WRITE_TOOL_NAMES 去重

## v2.13.22「修复整轮结束后大过程卡折叠 / 小过程卡合并误展开」（2026-08-12）
> 根因：整轮结束后的合并「大过程卡」复用了运行中首段「小过程卡」的同一组件
> 实例（key 相同），小卡初始为折叠，导致大卡跟着折叠；上一版改为把该实例
> 撑开，又等于「小卡展开成大卡」，违背「小卡默认折叠」。
> 修复：合并大卡改用独立 key 全新挂载（天然默认展开），小卡实例始终折叠，
> 两者互不干扰。
> 详见 releases/v2.13.22.md。
- Transcript：非运行态（整轮结束）的分段改用 `done-` 前缀 key，合并大卡全新挂载
- ProcessCard：移除小卡→大卡实例复用的展开分支（小卡始终默认折叠）
- 回归测试：小卡挂载折叠、大卡挂载展开
- 验证：tsc + 前端 129 例全过；wails build 通过；产物 gaea-v2.13.22.exe

## v2.13.21「办公板块安全审计：封堵子代理绕过持久化写入审批」（2026-08-12）
> 审计发现最严重漏洞：默认 task 子代理继承全部工具（cost_save/remember/
> knowledge_add/promote_session_facts/install_skill）但运行在 headless 审批
> 通道上，可绕过主代理的逐条确认静默写入成本库/记忆/知识库/技能。
> 修复：子代理注册表剔除全部持久化写入工具；主代理的 forget/install_skill
> 一并纳入硬性逐条审批。其余复查（弹窗、上下文注入）无新问题。
> 详见 releases/v2.13.21.md。
- agent/task.go：FilterRegistry 剔除 cost_save/remember/forget/knowledge_add/promote_session_facts/install_skill
- control：hardAskTools 补 forget / install_skill
- 回归测试：子代理注册表不含持久化写入工具；默认子代理工具集更新
- 验证：go build/agent/control/permission 测试通过；wails build 通过；产物 gaea-v2.13.21.exe

## v2.13.20「记忆/知识库写入强制确认 + 记忆索引注入预算」（2026-08-12）
> 与成本库同源问题：remember / knowledge_add / promote_session_facts 由 AI 直接
> 落盘、无确认，写入一堆杂乱记忆并整体注入系统提示词占用上下文。现在这三个
> 工具与 cost_save 一样进入硬性逐条审批（任何权限级别含 yolo 都弹卡、批准仅
> 本次生效）；「Saved memories」索引注入增加预算（3000 runes 截断，其余用
> memory_search 按需查询），控制上下文占用。
> 详见 releases/v2.13.20.md。
- control：hardAskTools 扩展 remember / knowledge_add / promote_session_facts，审批摘要含条目名/描述/分类/来源等
- memory：capMemoryIndex 限制注入系统提示词的记忆索引长度（3000 runes）
- ApprovalModal：硬性审批工具统一隐藏「本会话允许」
- 回归测试：yolo 下记忆/知识写入仍审批、会话放行不记忆、审批摘要格式
- 验证：go build/control/memory 测试通过；前端 tsc + 128 例全过；wails build 通过

## v2.13.19「成本库写入强制逐条确认，AI 不再直接入库」（2026-08-12）
> 修复成本库被 AI 生成的虚高价格污染：此前权限级别为 auto/yolo 时所有工具
> 自动放行，cost_save 直接把数据写进成本库。现在 cost_save 成为硬性审批项——
> 任何权限级别（含 yolo）都必须弹审批卡，逐条确认条目名称/单价/单位/规格/
> 来源后才写入；不提供「本会话允许」，批准仅本次生效。
> 详见 releases/v2.13.19.md。
- permission.Gate：新增 AlwaysAsk 硬门，cost_save 无视权限级别/放行规则/会话记忆
- control：gateApprover 对 cost_save 强制 requestApproval（alwaysPrompt），审批摘要含条目名称/单价/单位/规格/来源
- SetPermLevel：auto/yolo 均保留 cost_save 硬门（yolo 改用空策略 gate 替代 nil gate）
- ApprovalModal：cost_save 隐藏「本会话允许」，显示逐条确认提示
- 回归测试：yolo 下仍触发审批、会话放行不被记忆、审批摘要格式
- 验证：go build/control/permission 测试通过；前端 tsc + 128 例全过；wails build 通过

## v2.13.18「通用办公左侧面板重设计（参考 Codex）」（2026-08-12）
> 左侧面板按 Codex 会话栏风格重排：紧凑头部（logo + 名称 + 新建/折叠图标按钮）、
> 搜索框前置放大镜、会话行改为「标题 + 时间 + 预览」结构（悬停操作、左色条选中态）、
> 区块头统一安静小字，功能与折叠/拖拽行为全部保留。
> 详见 releases/v2.13.18.md。
- Sidebar.tsx：头部精简、会话行重构、搜索框带图标、分组头/区块头统一，删除大号胶囊新建按钮
- 保留：会话搜索/重命名/删除/历史、后台任务、事实底座（导出/沉淀/复制/清空）、记忆/知识库/技能导航、折叠与拖拽调整宽度
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.18.exe

## v2.13.17「修复运行中已完成的小过程卡不折叠」（2026-08-12）
> 修复 v2.13.9 引入的回归：当时为让「大过程卡（整轮合并、包含文本卡的卡）」
> 保持展开，把 ProcessCard 初始状态改成了默认展开，导致运行中不含文本的
> 分段「小过程卡」在段完成后也不再收起。现在小过程卡默认折叠、段完成后
> 收起；含文本的大过程卡保持展开；正在流式的最后一段保持展开。
> 详见 releases/v2.13.17.md。
- Transcript.ProcessCard：新增 small 标记（运行中的分段小卡）；初始折叠、完成/历史段自动收起
- 大过程卡（consolidated，含中间文本）默认展开逻辑不变
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.17.exe

## v2.13.16「办公板块铺满窗口 + 顶部标签删除 + 底栏只显示本地模型」（2026-08-12）
> 通用办公三个显示细节优化：① 删除办公板块顶部「办公」二级标签，模块直接由
> 通用办公工作台承载；② 底部状态栏的已启动模型列表/超载报警只统计本地模型，
> 云端引擎（xai/deepseek/opencode）不再展示与计入；③ 全屏/最大化时通用办公
> 铺满整个窗口（此前内容区对办公页保留了 8/16px 内边距）。
> 详见 releases/v2.13.16.md。
- MainLayout：办公页路由直接挂 GaeaPage，删除 OfficeHubPage 与 office-hub/office-page 样式；内容区对办公页去内边距、铺满
- StatusBar：已启用模型列表按 isLocal 过滤，只显示本地模型；超载报警维持本地统计
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.16.exe

## v2.13.15「删除方案编写，办公板块收敛为单一入口」（2026-08-12）
> 办公板块下线「方案编写」二级分支：删除整个方案编写模块（前端页面、
> proposal/db 后端包、Proposal* 绑定、模块注册、方案库），去除办公二级导航，
> 办公板块直接承载通用办公工作台。左脑记忆源从方案库改为办公记忆 facts，
> 三脑检索与记忆中枢不受影响。
> 详见 releases/v2.13.15.md。
- 前端：删除 OfficePage，OfficeHubPage 去除二级导航（单一通用办公入口）；模块启动器与设置面板清理「方案编写」文案
- 后端：删除 internal/office/proposal 与 internal/office/db，移除全部 Proposal* 绑定与 office 模块注册
- 左脑：办公记忆源改为 Hephaestus facts（三脑检索/记忆图谱继续可用）
- 验证：go build/定向测试通过；前端 tsc + 128 例全过；wails build 通过；产物 gaea-v2.13.15.exe

## v2.13.14「通用办公工具/技能面板显示 Word·Excel·PDF 技能」（2026-08-12）
> 通用办公的「技能面板」看不到 docx/xlsx/pdf/pptx：技能只装在仓库 .gaea/skills，
> 而技能索引只扫「当前工作区 .gaea/skills + 用户级 ~/.gaea/skills」，其它工作区
> 均为空。已把四个 ModelScope 文档技能安装到用户级全局目录（任意工作区可见），
> 并在「工具面板」新增「文档技能」分组展示这四个技能。
> 详见 releases/v2.13.14.md。
- 安装：docx / xlsx / pdf / pptx 复制到 ~/.gaea/skills（全局技能，所有工作区可见）
- CapabilitiesPanel：工具面板新增「文档技能」分组（docx/xlsx/pdf/pptx 卡片）
- 验证：全局技能在任意工作区可见（金具厂实测通过）；tsc + 前端 128 例全过；wails build 通过

## v2.13.13「聊天面板左侧栏可折叠 + 悬浮绑定模型卡随面板隐藏」（2026-08-12）
> 聊天页面左侧会话栏原本固定 264px 宽、无法折叠；左下角「聊天」绑定模型卡
> 悬浮在其上方也不随面板变化。新增折叠窄栏（52px，保留展开/新建按钮），
> 折叠态持久化；折叠时悬浮绑定模型卡一并隐藏。
> 详见 releases/v2.13.13.md。
- ChatTopicSidebar：新增 collapsed/onToggle，折叠为窄栏，展开态头部增加折叠按钮
- ChatPage：折叠状态存 localStorage；折叠时隐藏左下角绑定模型卡
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.13.exe

## v2.13.12「通用办公布局优化：精简右侧边栏 + 绑定模型卡随面板折叠」（2026-08-12）
> 通用办公右侧工作区面板删除「成本库」「搜索」两个标签页；左下角「绑定模型卡」
> 原为绝对定位悬浮，会盖住左侧栏底部的「知识库」「MCP 与技能」按钮，且不随
> 面板折叠。改为放入左侧栏底部（与后台任务/事实底座同规则），折叠时随面板隐藏。
> 详见 releases/v2.13.12.md。
- App.tsx：右侧边栏删除成本库/搜索标签与对应面板，同步移除命令面板的搜索入口
- Sidebar.tsx：绑定模型卡移入左侧栏底部，折叠时隐藏，不再遮挡底部导航
- GaeaPage.tsx：移除原来的绝对定位悬浮层
- 验证：tsc + 前端 128 例全过；wails build 通过；产物 gaea-v2.13.12.exe

## v2.13.11「修复记忆中枢·用户画像打不开」（2026-08-12）
> 记忆中枢「用户画像」页打开即白屏：后端无冲突时返回 nil 切片，序列化成
> JSON null，前端直接读 `conflicts.length` 抛 TypeError，被 ErrorBoundary
> 拦截。后端统一保证空数组，前端对 conflicts/tags 做 null 兜底，并加固
> 图谱与聊天记忆页同型风险。
> 详见 releases/v2.13.11.md。
- GaeaProfileConflicts：无冲突时返回 [] 而非 null（DetectConflicts 改 []string{}）
- ProfileLibrary：conflicts/tags 加 ?? [] 兜底（崩溃根因）
- 顺带加固：GaeaWhisperEpisodes 错误路径、GaeaMemoryGraph 空图、GraphView/WhisperMemoryLibrary null 防护
- 回归测试：Go DetectConflicts 非 nil 空切片 + 前端 ProfileLibrary null 用例
- 验证：go build/测试通过；前端 128 例全过；wails build 通过；产物 gaea-v2.13.11.exe

## v2.13.10「修复办公文档处理反复弹 cmd 黑窗」（2026-08-12）
> 通用办公后台做 docx/xlsx 转换、公式重算、报告导出、绘图、桌面自动化时，
> 子进程没加隐藏窗口参数，每执行一次就闪一个 cmd/conhost 黑窗；批量转换时
> 会"不停弹窗"。统一补上 CREATE_NO_WINDOW，全部静默执行。
> 详见 releases/v2.13.10.md。
- docmd.markitdown：`python -m markitdown` 补 HideWindow（弹窗主因，已实测抓到）
- xlsxedit.Recalc、gaea_export.runPython、diagram_gen、whisper 桌面操作（powershell）一并补上
- 验证：go build 通过；前端 127 例全过；wails build 通过；产物 gaea-v2.13.10.exe

## v2.13.9「每一轮的大过程卡默认展开 · 内部卡默认折叠」（2026-08-12）
> 调整通用办公输出显示：输出完成后每一轮的外层大过程卡都默认保持展开，
> 不再只保留最新回合；大过程卡内部的思考卡改为默认折叠，工具卡/工具组
> 保持默认折叠，只留过程文本直接可见。
> 详见 releases/v2.13.9.md。
- ProcessCard：移除 isLatest 限制，每轮大过程卡完成后均保持展开（用户手动折叠过仍尊重）
- InlineReasoning：思考卡默认折叠，点开才看推理内容；工具卡/工具组保持默认折叠
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.9.exe

## v2.13.8「仅最新回合的大过程卡默认展开」（2026-08-12）
> 修正过程卡展开策略：只有最新回合的外层大过程卡默认展开，旧回合自动折叠，
> 避免“所有过程卡全部摊开”；手动展开过的旧卡仍保留。
> 详见 releases/v2.13.8.md。
- ProcessCard 增加 isLatest：完成时仅最新回合保持展开，新回合开始后旧卡折叠
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.8.exe

## v2.13.7「办公上下文窗口修正 + 大上下文处理提示」（2026-08-12）
> 修复“长任务只有执行中、迟迟不出流式”的根因：办公 provider 上下文窗口此前
> 写死 1M，自动压缩阈值高达 80 万 token，会话膨胀到十几万也从不压缩，模型
> 首字要等几分钟，看起来像没输出。窗口调至 256k 让压缩生效，并加处理提示。
> （v2.13.6 的最终回答兜底补渲染一并包含在本版）
> 详见 releases/v2.13.7.md。
- gaea 办公 provider ContextWindow 1_000_000 → 256_000（80% 阈值≈204k，超限自动压缩）
- 前端 RunStatus：运行 ≥20s 且上下文 ≥4 万 token 时显示「处理大上下文中 · N」，不再误判卡死
- 最终回答兜底：turn_done / 看门狗恢复时校验 History 补渲染缺失的最终回答
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.7.exe

## v2.13.5「运行中强制跟随底部，修复“卡住没输出”假象」（2026-08-12）
> 长任务其实一直在输出，但智能滚动一旦被布局变化/折叠动画置为非跟随状态，
> 视图就冻在旧位置，看起来像卡死。运行中改为强制跟随底部，结束恢复智能滚动。
> 详见 releases/v2.13.5.md。
- Transcript：running 时内容变化强制滚到底部；运行结束后保持原智能滚动
  （用户上翻浏览时不再强拉）
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.5.exe

## v2.13.4「外层过程卡完成后默认展开」（2026-08-12）
> 修正 v2.13.3 的方向：要展开的是包裹「过程文本 + 过程卡」的最外层大过程卡
> （ProcessCard），输出完成后默认保持展开；内部工具卡仍默认折叠。
> 详见 releases/v2.13.4.md。
- ProcessCard：running → 完成后不再 setOpen(false)，外层大过程卡默认保持展开
  （用户手动折叠过仍尊重）；v2.13.3 误加的内部卡自动展开已撤回
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.4.exe

## v2.13.3「过程卡完成后默认展开」（2026-08-12）
> 输出完成后的大过程卡（bash/python 等长输出卡片）默认展开，用户手动折叠后
> 不再干预；连续同名工具组含大卡时组也默认展开。
> 详见 releases/v2.13.3.md。
- ToolCard：输出 ≥10 行或输出/参数 ≥2000 字符的大卡，完成（done/error）后自动展开
- ToolGroup：组内任一成员为大卡且完成时，组默认展开
- 验证：tsc + 前端 127 例全过；wails build 通过；产物 gaea-v2.13.3.exe

## v2.13.2「办公过程文件落盘规范：统一 .gaea/work/」（2026-08-12）
> 办公 agent 执行纪律新增「文件落盘规范」：过程/中间文件（提取文本、OCR 页图、
> 脚本、临时图表等）统一写入 .gaea/work/<任务名>/，交付物进 .gaea/exports/，
> 不再与源文件混在工作空间根目录；启动时自动创建这两个目录。
> 详见 releases/v2.13.2.md。
- 办公 agent 提示词新增「文件落盘规范」章节（single_prompt.go）
- GaeaInit 自动创建 .gaea/work/ 与 .gaea/exports/；.gitignore 增加 .gaea/work/
- 验证：go build/vet 干净；wails build 通过；产物 gaea-v2.13.2.exe

## v2.13.1「修复 @PDF 引用注入二进制导致办公输出不可见」（2026-08-12）
> 补丁版：修复通用办公里 @引用 PDF 时，文本提取失败会把 %PDF-1.4 原始二进制
> 塞进模型上下文，导致任务只剩读秒、看不到任何输出。
> 详见 releases/v2.13.1.md。
- @引用 office 文档（PDF/docx）转换失败时注入占位提示（format_convert / summarize_file），
  绝不回退成原始字节；二进制检测扩展到整个读取窗口（此前只查前 8KB，PDF 头无 NUL 会漏检）
- 验证：go build/vet 干净；wails build 通过；产物 gaea-v2.13.1.exe

## v2.13.0「通用办公打磨 + 模型中心优化 + 本地模型实测」（2026-08-12）
> 承接 v2.12.0，本轮聚焦通用办公稳定性与方案写作质量：修复方案分节 100 字
> 截断（流式与批量两处）、docx 读取乱码、前端“运行中”假卡死；配套模型中心
> 优化与 Herdsman/Ollama 本地模型双模式实测。
> 详见 releases/v2.13.0.md。
- 通用办公：方案「生成该节」与批量生成按章节字数目标续写（WordTarget，缺省 800），
  修复原硬编码 100 字截断；docx 读取统一走 ConvertToMarkdown（MarkItDown / 内置
  转换 / 扫描件提示），修复直接读 zip 字节乱码；前端 turn 结束事件丢失兜底
  （停止按钮无条件复位 + 30s GaeaRunning 看门狗），任务完成不再卡「执行中」；
  待办从未推进时 turn_done 自动收尾，不再残留 0/N
- 工具：format_convert 输出父目录自动创建（与 write_file 对齐）+ 回归测试；
  docmd 新增 MarkItDown 通道（docx/pptx），PDF 页数上限与扫描件 OCR 回退保持
- 模型中心：功能绑定/板块细节优化；本地引擎思考参数（enable_thinking /
  chat_template_kwargs）与路由增强
- 语音/角色库：whisper web_search、角色画像文件化等累积修复
- 实测：Herdsman 三模型 20 任务 × 思考/非思考双模式 + Ollama GLM 实测，
  脚本 scripts/test_herdsman_models.go 可复现；详见
  docs/2026-08-12-herdsman-models-evaluation-report.md
- 验证：go build/vet 干净，方案包等测试全绿；前端 127 例全过，tsc + vite build 通过

## v2.12.0「稳定工程 + 成本库多级分类重设计」（2026-08-11）
> 系统根治 WebView2 rAF 冻结引发的「界面卡死 / 点不了 / 角色卡看不见」，
> 成本库升级为按分类分级保存 + 列表/表格双视图，剧照文件化防 IPC 撑爆。
> 详见 releases/v2.12.0.md。
- 稳定性：全应用 30+ Modal 关闭即卸载 + 禁用过渡动画（弹窗关闭后不再残留遮罩卡死）；
  rAF 持续探测 + 8s 冻结看门狗（运行中晚发节流也能降级）；角色卡/启动器/聊天气泡
  入场动画纳入降级名单（修复角色库卡片全看不见）；前端错误/长任务/心跳诊断进 gaea.log
- 成本库：cost_categories 多级分类树 + 条目 category_path 多级保存；默认按信息价体系
  分类（材料细分三级）；分类增删改、改名自动重写子树路径、按路径含子树过滤；
  面板重设计为分类树 + 列表/表格双视图（排序/批量/价格历史）；cost_search/cost_save
  支持分类路径
- 剧照文件化：base64 剧照落盘存路径，>300KB 内联头像截断，根治巨型 base64 撑爆 IPC
- 记忆中枢/办公：成本导入（预览确认 + AI 归一化）、价格源仓库、价格历史与跳幅识别、
  知识导入/版本历史/查重合并、画像库、办公记忆查重合并、记忆图谱增强；后端导入链路
  加固与批量测试覆盖
- 验证：go build/vet 干净，go test 相关包全绿；前端 127 例全过，tsc + vite build 通过

## v2.11.0「通用办公大优化：四大库能力闭环 + 本地语义检索栈」（2026-08-10）
> 承接 v2.10.1 开工前 P0，落地调研蒸馏出的 P1 三项：工作区全文搜索、
> 常用资料固定装配、任务模板库——把「开工前」从「能搜到文件」升级为
> 「能搜到内容、能钉住常用资料、能一键发起常见任务」。
- 工作区全文搜索（轻量 RAG）：右侧新增「搜索」标签页 + 命令面板入口，在
  docx/xlsx/pdf/md/txt/csv 等正文里定位关键词（中文 bigram 分词 + TF-IDF 打分），
  返回命中片段，可预览或一键 @ 引用；正文按 (路径, mtime, size) 缓存，重复搜索
  不重复解析；.gaea/sessions|archive|cache 等噪音跳过，.gaea/exports 交付产物可索引
- 常用资料固定装配（P1-②）：资料面板每行图钉「固定为常用资料」，清单持久化到
  <工作区>/.gaea/pinned.json（去重、限长、防路径逃逸）；新会话启动时自动把固定
  清单装进系统提示词——文本类附正文摘要、办公文档列名按需读取（装配而非灌输）
- 任务模板库（P1-③）：欢迎页新增「任务模板」区（周报/会议纪要/成本测算/方案大纲/
  数据分析/文档转换/报告拼装/演示文稿），一键填入结构化指令；同源模板同时落盘为
  .gaea/commands/*.md（幂等、不覆盖用户文件），/ 菜单与 Submit 直接可用
- 修复：办公引擎命令发现改为基于工作区根（CommandDirsAt），切换工作空间后
  模板与自定义命令跟随新工作区，与技能/记忆的发现口径一致
- 记忆中枢关联（本轮打通）：记忆中枢首页「三脑检索」并入工作区全文搜索——
  文件命中以「文件」徽标同框展示，可预览 / 一键 @ 引用（回到办公板块自动插入
  输入框）；首页新增「项目资料」卡片（固定常用文件计数）与对应库面板（已固定/
  可固定资料管理、预览、固定/取消，与办公面板共用 .gaea/pinned.json）；固定
  资料作为 material 节点进入记忆 3D 图谱（天蓝色，图谱过滤器可单独开关）
- 记忆生命周期与可控（P1-④/⑤）：facts 新增 last_used_at / source_session /
  source_message（SchemaV3 迁移，修订不重置使用时间）；memory_get 读取即 Touch，
  逐轮记忆注入改为「关键词 + 时间 + 高频」排序 + 预算压缩（RecallBlock 默认
  800 rune，画像 600 rune），procedural 常驻、episodic 按标签触发、相关事实按
  命中带入；记忆面板顶部新增「记忆开关」（持久化配置 + 引擎重建立即生效），
  事实卡片展示来源会话与最近使用时间
- 方法论自动候选（P1-⑥）：记忆面板「建议」标签接入真实后端——从 procedural
  记忆按主题词聚类（≥2 条同主题 → 提议 workflow-<主题> 技能，附证据与正文），
  接受后经既有技能沉淀通道固化并热加载；记忆候选保持为空（自动做梦已直接入库，
  宁缺毋滥）
- 验证：后端新增 wssearch/pins 包测试 + boot 集成测试（工作区命令发现），
  以及记忆中枢关联测试（固定数统计 + 图谱 material 节点）、记忆生命周期/压缩
  注入/技能聚类测试；前端新增 WorkspaceSearchPanel 2 例 + 资料固定 1 例 +
  项目资料库 1 例，vitest 93 例全过，tsc 与 vite build 通过

### 本轮追加：成本库/知识库/办公记忆/工作区文件大优化
- **成本库打通与功能深化**：cost_search/cost_save 内置工具（读/写 Hephaestus.db
  cost_entries，同名 UPSERT）；cost-estimate 模板「先查后沉淀」；办公右侧成本库 Tab
  （浏览/搜索/一键结构化引用）；产物面板「沉淀到成本库」；导入 xlsx/csv + AI 归一化
  （预览确认，无确认不落库）；编辑/删除/批量操作；价格源订阅 + 手动/定时抓取
  （四川造价信息网实测 24 行解析）+ 价格历史 + 单期跳幅 ±20% 异常识别
- **知识库大优化**：md/txt/docx/pdf/xlsx/csv 导入 + AI 多主题拆分（预览确认）；
  批量管理 + 审核流（草稿→现行、审核人留档）+ 版本历史（内容变化自动留档）+ 批量
  导出 Markdown；查重（CJK 二元组 Dice）与相似条目一键合并
- **办公记忆**：两两查重 + 一键合并（textsim 共享相似度），自动做梦产生的同名变体
  可一键归并
- **工作区文件语义索引**：fileindex 扫描 md/txt/csv/docx/xlsx/pdf/pptx 提取正文
  （pptx 零依赖解 zip 读 slide XML），bge-m3 向量化入库（semantic_vectors kind=file，
  内容感知增量重建 + 10 分钟自动维护），搜索面板「语义」模式 + 记忆中枢统一搜索
  并入「语义·文件」命中
- **本地语义检索栈（四库统一）**：子串召回 →（不足时）bge-m3 语义召回 → BM25 排序 →
  （>8 条）bge-reranker-v2-m3 精排，任一层失败自动降级；跨库统一语义检索
  （GaeaSemanticSearch，成本/知识/办公记忆）；Herdsman bge-m3 / bge-reranker-v2-m3
  实测端到端通过，全部本地零 token
- 验证：go test 全量 + go vet 干净；前端 111 例全过，tsc 与 vite build 通过；
  真实 Herdsman 语义召回/精排与四川价格源抓取实测通过；产物 gaea-v2.11.0.exe

## v2.10.1「开工前 · 交付收尾 · 记忆自进化」（2026-08-10）
> 把办公体验从「下完指令等结果」补成完整闭环：开工前先出计划、资料一键引用；
> 对话里所有文件引用可点击预览，交付物收尾成卡片；会话结束后自动「做梦」整理记忆。
- 输出文件全量可点击预览：mdast（remark 插件）识别绝对/相对/裸文件名（参考
  openclaw/llama.cpp 的 AST 插件架构），聊天正文、流式尾部、工具输出、ask 提问均可点击，
  代码块/已有链接/公式天然跳过；后端支持裸文件名自动解析到 exports 等常见输出目录
- 交付收尾：助手消息尾部「交付文件」卡片（缩略图/图标 + 文件名 + 扩展名，预览/复制路径/
  外部打开/定位）；右侧「会话产物」面板（本次会话交付文件清单 + 溯源跳转到生成消息）；
  预览内编辑后自动标「已更新」并刷新文件树；粘贴表格即数据（CSV/TSV 识别 → Markdown 表格）
- 记忆自进化（自动做梦）：轮次成功后后台归纳对话 → 提炼稳定事实/偏好/笔记写入长期记忆
  （按 name 去重、Kimi 二问纪律、单飞、失败静默）；整理结果在聊天内通知
- 开工前：@ 文件引用增强（工作区跨目录搜索 + 最近使用 + 扩展名徽标）；右侧「资料」面板
  （docx/xlsx/pptx/pdf/md/csv 按类型分组、最新在前、一键 @ 引用进输入框）；开工前计划卡片
  （非简单任务先生成计划，用户「确认执行 / 先调整」，auto_plan 可配置，默认开启）
- 三轮竞品优点蒸馏文档：交付收尾 UX、记忆与自进化、开工前上下文；调用链核对文档
  （docs/gaea2/office-model-tool-call-chains.md）
- 发布说明：产物 gaea-v2.10.1.exe（Windows x64），桌面端同步；go test（agent/control/boot/
  app 全量）+ vitest 89 例全过，tsc 与 wails build 通过

## v2.10.0「正式发布 · 通用办公三阶段闭环」（2026-08-09）
> 自 v2.6.9 之后最大一次发布：把「办公前期的文件解析、中期的 Word/Excel
> 编辑、后期的文件输出与预览」串成一条完整闭环，并完成桌面端体验重构与
> 真实模型端到端验收。本版本包含 v2.7.0–v2.9.4 的全部累积特性。
- 前期解析：docx/xlsx/pdf → Markdown 提速（docx 约 12x / PDF 约 4x）；
  扫描件 OCR 四级管线（OvisOCR2 常驻服务 → RapidOCR → WinRT → 本地视觉模型），
  粘贴图片「提取文字/识图」双入口；事实底座（fact_add/list/clear + 侧栏面板 +
  一键沉淀长期记忆，后续对话自动加载）
- 中期编辑：Word 框选即改 + 修订模式逐条接受/拒绝 + diff 回看；Excel 单元格级
  预览/编辑（双击改值/写公式、fx 栏、AI 指令编辑、插入/删除行与列、LibreOffice
  公式重算、数字格式/冻结表头/合并单元格）；跨应用联动（Excel 数据 → 图表 →
  嵌入 Word/PPT 并随数据同步刷新）
- 后期输出：统一交付出口（对话成果一键导出 docx/pptx/xlsx/md）、模板与样式体系、
  成本测算模板（市政道路改造工程，公式全联动 + 原生图表，真实 DeepSeek 绑定
  自主生成端到端验收通过）
- 桌面端体验：Codex 式文件预览布局（右侧文件树、主区域可拖宽预览、Esc/按钮
  快捷收起）、文件就地编辑不跳出对话
- 发布说明：产物 gaea-v2.10.0.exe（Windows x64，约 40MB），桌面端同步；
  go test / vitest 59 例全过，tsc 类型检查与 wails build 通过

## v2.9.4「表格列操作：插入/删除列」（2026-08-09）
> 按用户要求：排序/筛选/条件格式不在 gaea 内做预览层实现（不写回文件），
> 后期交给 Excel/WPS 专业软件处理；本次仅落地会真实写文件的列操作。
- 插入/删除列：点击列字母选中整列，工具栏出现「← 插列 / → 插列 / 删除列」
  （删除二次确认），后端 excelize InsertCols/RemoveCol + 重算刷新，改动落盘
- 新增后端 GaeaXlsxColOps；列选择与单元格选择互斥
- 清理：移除本轮试验性的排序宏（LibreOffice）、AutoFilter 写入与条件格式渲染
- 验证：Go 单测（插列/删列错位、非法操作/非法引用）+ 前端列操作用例；
  go test / vitest 59 例全过；桌面 gaea.exe 已重建同步

## v2.9.3「表格行操作：插入/删除行」（2026-08-09）
> 按用户要求补齐高频表格操作——行级插入与删除。
- 预览头部新增「↑ 插行 / ↓ 插行 / 删除行」：基于选中单元格所在行，
  插入空行或删除整行；删除需二次确认（点击后变为「确认删除」）
- 后端新增 GaeaXlsxRowOps（excelize InsertRows/RemoveRow 平移同表公式与合并区，
  随后 LibreOffice 重算刷新结果并重渲染预览）
- 验证：Go 单测（插入错位/删除上移/非法操作/非法引用）+ 前端插行用例；
  go test / vitest 58 例全过；桌面 gaea.exe 已重建同步

## v2.9.2「公式结果显示 + 表格功能补强」（2026-08-09）
> 修复公式格只显示 fx 无结果的问题，并补上两类高频表格能力。
- 公式结果：根因是 openpyxl 生成的文件公式没有缓存值（未重算），预览渲染时
  只剩 fx 标记。现在 GaeaPreview 自动检测「无缓存值公式」→ LibreOffice 重算后
  再渲染；预览头部新增「重算公式」按钮可手动刷新；新增 GaeaXlsxRecalc 接口
- 数字格式：NumFmt 已提取但前端未应用，现按格式显示千分位/百分比/货币/小数位
  （内置编号 1-11、44-47 及常见自定义格式），公式结果也按单元格格式呈现
- 冻结表头：提取 sheet 冻结窗格（GetPanes），预览中冻结行 sticky 固定，
  滚动数据时表头不跑
- 验证：新增 NeedsRecalc 单测 + 冒烟回归（真实成本表 51 个公式全部有结果，
  预览自动重算 1.5s）+ 前端数字格式用例；go test / vitest 57 例全过；
  桌面 gaea.exe 已重建同步

## v2.9.1「预览拖拽修复 + 表格就地编辑」（2026-08-09）
> 修复 Codex 式预览的两个实测问题：拖拽分割条失效、表格只能看不能改。
- 修复拖拽：预览宽度上下限计算写反导致宽度卡死（恒为最大），改回
  320–1100px 自由拖拽；分割条命中区 6px→10px，悬停/拖拽高亮
- 表格 Excel 式就地编辑：双击单元格直接输入（Enter 保存 / Esc 取消），
  上方 fx 公式栏也可输入值或 `=公式` 回车写回；纯数字按数值写入保持可计算，
  等号开头写公式，写回后 LibreOffice 重算并即时刷新预览
- 新增后端 GaeaXlsxSetCell 直写接口（excelize 写值/公式 + 重算 + 预览）
- 验证：新增 Go 单测（写值/写公式/非法引用）+ 前端双击编辑用例；
  go test / tsc / vitest 56 例全过；桌面 gaea.exe 已重建同步

## v2.9.0「Codex 式文件预览布局」（2026-08-09）
> 参照 Codex 桌面端交互改造办公文件查看：右侧面板收敛为「增强文件树」，
> 点击文件后原右侧面板隐藏，预览在聊天区右侧主区域展开（全高、宽度可拖），
> 聊天与预览互不遮挡，可边对话边看文件。
- 布局：点文件 → 树收起 → 主区域出现预览面板，聊天区保留在左侧；
  预览与聊天之间加 6px 拖拽分割条，宽度 320–1100px 自由调整并持久化（localStorage）
- 预览头部：新增「文件」返回按钮（回到文件树）、文件名/大小、定位/外部打开/关闭；
  Esc 或工具栏面板按钮均可收起预览回到树
- 文件树增强：docx 蓝 / xlsx 绿 / pptx 橙 / pdf 红 / 图片紫 按扩展名着色，
  文件夹优先 + 自然排序；新增手动刷新按钮（强制重载目录）
- 清理：移除已被树内嵌预览取代的「最大化面板」逻辑与重复的“查看文件变更”按钮
- 验证：tsc --noEmit 通过；vitest 55 例全过；wails build 成功，桌面 gaea.exe 已同步

## v2.8.9「自主生成验收 · gaea 自己生成成本测算表」（2026-08-09）
> 让 gaea 智能体用真实模型（办公绑定 deepseek/deepseek-v4-flash）自主生成
> 成本测算表：无头驱动 Controller.Run + bridge provider + 完整工具链，验证
> 产物含公式与原生图表——桌面端已构建，办公功能绑定已生效。
- 新增 TestGaeaSelfGenerateCost（GAEA_SELFGEN 门控）：读取 ~/.gaea_config.json
  办公功能绑定路由引擎/模型；照桌面端接线 engineMgr + ai.Client + bridge；
  提示词要求 openpyxl 建表、公式联动、原生饼图/柱状图、LibreOffice 重算
- 实测结果（900s）：智能体自主完成环境检查→写脚本→生成→重算→自检；
  产物 3 工作表（成本测算/费用汇总/编制说明）、35 个公式、2 个原生图表，
  openpyxl 断言全过；路由日志「来源 feature」确认走了用户绑定
- 顺手修复既有测试全局污染：TestGaeaBootBuild 注入 gaeaConfig.SetLoader
  未恢复，会影响同包后续 boot.Build（真实模型跑测试时被 agent 诊断发现）；
  已 defer 恢复；自生成测试产物自动复制到 .gaea/exports/
- 验证：go vet/test ./internal/app 全绿；桌面端 gaea.exe 已重新构建并
  同步桌面（含全部办公能力与绑定）

## v2.8.8「成本测算模板 · 真实样例交付」（2026-08-09）
> 新增可复用成本测算生成脚本 cost_estimate.py（openpyxl）：市政道路改造工程
> 成本测算工作簿——按费用构成（人工/材料/机械/企管/规费/利润/税金）建表，
> 公式全联动（数量×单价=合价、小计、占比、含税总价、综合单价），原生饼图+
> 柱状图，LibreOffice 重算后带缓存值。
- 交付物：.gaea/exports/市政道路改造工程成本测算.xlsx（含税总价
  9,992,806.15 元，综合单价 462.63 元/m²，53 个公式 0 错误，2 个原生图表）
- 验证：openpyxl 数值/公式/图表断言、PDF 渲染文本核验（表/图表标题/编制说明
  完整）、gaea GaeaPreview 单元格预览管线读取正常（GAEA_SMOKE_COST 门控）

## v2.8.7「P2 跨应用联动 · Excel 数据 → 图表 → 嵌入 Word/PPT」（2026-08-09）
> 联动闭环：xlsx 数据一键生成 matplotlib 图表并嵌入 docx（报告模板：标题+
> 图表+数据表）或 pptx（图表页+数据明细页）；数据源更新后重新导出即刷新图表，
> 实现「图表与数据保持同步」（避免脆弱的 OLE 活链接）。
- 新增 internal/office/crosslink：xlsx 数据提取（自动/显式区域 A1:B6，表头+
  数值列解析、千分位清洗）、matplotlib 图表生成（bar/line/pie/scatter，中文
  字体、无窗口执行）、docx/pptx 嵌入 spec 构建（docx 含数据表，pptx 图表页+
  数据明细页）
- create_pptx.py 新增每页 image 支持（图表页居中嵌入）；脚本镜像同步
- 新增 GaeaCrossEmbed 绑定：格式/图表类型校验、产物与图表落到 .gaea/exports/
- 前端 XlsxPreview 新增「图表→Word」「图表→PPT」按钮：取当前工作表数据一键
  联动，成功后自动定位产物
- 验证：crosslink 单测（数据提取/区域裁剪/spec 构建）+ 真实 matplotlib 图表
  冒烟（23KB PNG，0.5s）+ docx/pptx 嵌入冒烟（产物合法 zip 且含 media 图片）；
  go test ./... 全绿、vet 干净；tsc + vite build 通过；vitest 55 例全过

## v2.8.6「打磨联调 · 边界加固 + 端到端走查」（2026-08-09）
> 把前几轮的能力串成一条全流程验证：解析 → 修订式编辑 → 接受修订 →
> 提取成果 → 模板化多形态交付，并补齐三处常见边界。
- 边界加固：
  - docxedit：表格单元格（w:tbl>w:tr>w:tc>w:p）内修订与接受全链路（合同场景）；
  - xlsxedit：transform 公式保留 $ 绝对引用、跳过函数名（LOG10/ROUND）误判，
    行引用调整专项用例；
  - 交付出口：标题含 Windows 非法字符（< > : " / ? 等）时清洗且保留中文
- 端到端：
  - 新增 TestOfficeFullPipeline（纯 Go）：编辑→接受→提取→导出 xlsx/md；
  - 新增真实文档全链路走查（GAEA_SMOKE_PIPELINE 门控）：26MB 方案文档
    编辑→接受→提取→报告模板导出 docx，全程 4s 完成、产物内容正确
- 验证：go test ./... 全绿、vet 干净；tsc + vite build 通过；vitest 54 例全过

## v2.8.5「后期输出 · 模板与样式体系」（2026-08-09）
> P1 模板库落地：统一交付出口支持 通用/公文/报告/合同 四套版式预设——
> 字体、标题色/标题字体、表头底色按模板切换，事实底座一键出对应样式的 Word。
- create_docx.py 新增 --template 预设：通用（宋体+深蓝标题）、公文（仿宋正文+
  黑体标题）、报告（微软雅黑+蓝标题）、合同（宋体+黑色标题）；封面标题色、
  标题 run、表头底色全部参数化贯穿
- GaeaExportDeliverable 新增 Template 字段（默认通用，非法值明确报错），
  docx 出口透传 --template
- 事实底座侧栏「导出报告」升级为「模板选择 + 导出」：报告（封面+目录）/
  公文 / 合同 / 通用，导出后自动定位文件
- 验证：四套模板实测生成（颜色/字体抽查：通用 1F3864/宋体、报告 2E74B5/
  微软雅黑、公文 000000/仿宋）；docx 导出冒烟带模板通过；go test ./... 全绿；
  tsc + vite build 通过；vitest 54 例全过；技能镜像 ~/.codex/skills 已同步

## v2.8.4「中期编辑 · 修订接受/拒绝 + diff 回看」（2026-08-09）
> P1 信任机制落地：框选即改的修订不再只能去 Word 里处理——预览里直接
> 「接受修订 / 拒绝修订」一键扁平化（按作者 gaea AI，不动他人修订），
> 修订样式本身即 diff 回看（原文划除 + 新文插入）。
- docxedit 新增 AcceptChanges / RejectChanges：字节级扁平化 w:del/w:ins
  （接受=删 w:del 留 w:ins；拒绝=删 w:ins 还原 delText→t 并保留 run 格式），
  只处理指定作者、跳过嵌套/重叠、无修订时明确报错
- 新增 GaeaDocxAcceptChanges 绑定（accept=true/false），返回更新预览
- 前端 DocxPreview：渲染后自动检测 ins/del 修订，标题栏出现「接受修订 /
  拒绝修订」按钮（加载中禁用），操作后重渲染并提示
- 验证：docxedit 接受/拒绝/无修订/他人修订不动四类单测 + 真实 26MB 文档
  「修订→接受」冒烟（1.7s 完成、XML 合法、标记清空）；go test ./... 全绿；
  tsc + vite build 通过；vitest 54 例全过

## v2.8.3「后期输出 · 事实底座 → 多形态交付统一出口」（2026-08-09）
> P0-④ 落地：统一交付管线「受控 Markdown → docx / pptx / xlsx / md」——
> 事实底座与对话成果一稿多用，多形态基于同一底座生成、彼此一致。
- 新增 GaeaExportDeliverable 绑定：格式校验、标题自动推导、时间戳防重名、
  非法字符清洗（保留中文）、交付到 .gaea/exports/ 并返回相对路径
- docx：复用 create_docx.py（python-docx，封面/目录/页眉页脚/页码参数化）；
  pptx：Markdown → slides spec（# 标题开新页、要点/表格行转要点）→ create_pptx.py；
  xlsx：Go 侧 excelize 直接提取 Markdown 表格为工作表（无表格时正文入 Sheet1）；
  md：直写；python 子进程带超时与 stderr 捕获
- 前端两个入口：会话工具栏新增「导出 Word」（对话成果一键交付，成功后自动定位）；
  事实底座侧栏新增「导出报告」（封面+目录，基于同一底座）
- 验证：md/xlsx/校验单测（纯 Go）+ docx/pptx 真实脚本冒烟（GAEA_SMOKE_EXPORT，
  实测通过、产物为合法 zip）；go test ./... 全绿；tsc + vite build 通过；
  vitest 54 例全过

## v2.8.2「中期编辑 · Excel 单元格级操作」（2026-08-09）
> P0-③ 闭环：在单元格预览上「选中单元格 → 指令（求和/平均/拆分列/清洗/替换/
> 自定义）→ AI 规划操作 → excelize 执行 → LibreOffice 重算公式 → 重渲染」，
> 公式可校验、结果可检查。
- 新增 internal/office/xlsxedit：AI 操作集（set_formula/set_value/fill_range/
  transform/replace/split_column/clean）校验与执行；transform 逐行公式自动调整
  相对行引用（跳过 $ 绝对引用与函数名）；操作数量/单元格数上限防滥用
- 新增 ai.XlsxEditOps：表格上下文（工作表/表头/抽样数据）+ 指令 → 严格 JSON 操作集；
  GaeaXlsxEdit 绑定走 office 功能绑定路由，闭环后返回更新预览与逐条摘要
- 公式重算：复用技能环境 recalc.py（LibreOffice 宏，自动探测 soffice），
  best-effort——重算不可用时编辑仍生效并提示；真实重算冒烟 1.75s 通过
- 前端 XlsxPreview 新增选中编辑工具栏：四预设（求和/平均值/拆分列/清洗）+ 自定义
  指令，执行后重渲染并展示应用摘要
- 验证：xlsxedit 单测（公式/逐行变换/填充/替换/拆分/清洗/错误分支）+ app 校验
  单测 + 真实 LibreOffice 重算冒烟；go test ./... 全绿；tsc + vite build 通过；
  vitest 53 例全过

## v2.8.1「中期编辑 · Excel 单元格级预览」（2026-08-09）
> P0-③ 第一步：xlsx 预览从「转 Markdown 弱表格」升级为「单元格级保真视图」——
> sheet 切换、公式标识与公式栏、样式近似还原、合并单元格与列宽，为后续
> 「选中区域 → 指令（公式/清洗/透视）→ 写回」的单元格编辑打底。
- 新增 internal/office/xlsxpreview：excelize v2.11.0（已修复 2026 年三个 CVE）
  提取结构化单元格 JSON（值/公式/类型/样式/合并/列宽）；超大表格截断到
  2000 行 × 100 列并标记；图表/宏表自动跳过；.xls 保持原 markdown 兜底
- GaeaPreview 对 .xlsx 返回 kind=xlsx（body 为 JSON），wire 契约复用原字段
- 前端新增 XlsxPreview 组件：sheet 标签切换、公式栏（点击 fx 单元格显示公式）、
  表头行列冻结、样式（加粗/填充/对齐/边框/颜色）、合并单元格 colspan/rowspan、
  列宽近似；侧栏与弹层两个预览入口接入
- 验证：xlsxpreview 单测（样式/公式/合并/列宽/多 sheet/截断）+ app 层
  GaeaPreview 端到端；go test ./... 全绿；tsc + vite build 通过；vitest 52 例全过

## v2.8.0「中期编辑 · Word 框选即改 / 修订模式」（2026-08-09）
> P0「中期编辑 · Word 框选即改」落地：在保真预览底座上，选中文字 → 指令
> （AI 四预设 + 自定义）→ AI 生成替换 → 以 Word 修订模式（w:del + w:ins）
> 就地写入并重渲染，其余内容与版式零扰动。
- 新增 internal/office/docxedit：字节级 XML 手术（不重建 OOXML）定位选中文本并执行
  修订式替换；保留 run rPr 格式、应用实体转义、空白折叠匹配兜底、优先修订非 hyperlink
  段落（TOC 目录项不被 docx-preview 映射 ins/del）；选区包含图片/制表符/换行等特殊
  格式时明确拒绝
- 新增 GaeaOfficeEditText 绑定（办公向改写提示词：关键信息不变、纯文本输出，走 office
  功能绑定路由）+ GaeaDocxApplyEdit（修订写入并返回更新预览，标记作者「gaea AI」）
- 前端 DocxPreview 新增「框选即改」工具栏：选中文字后出现，四类快捷指令
  （润色/精简/翻译中文/扩写）+ 自定义指令输入，diff 预览（原文 → AI 替换）后
  「应用修订 / 放弃」；应用后修订样式重渲染
- 验证：docxedit 单测（单 run 拆分/跨 run 切分/XML 转义/空白折叠兜底/超链接优先/
  特殊格式拒绝）+ 真实 26MB 方案文档冒烟（0.9s 修订、合法回写、docx-preview 映射
  <ins>/<del>）；go test ./... 全绿；tsc + vite build 通过；vitest 49 例全过

## v2.7.9「通用办公 · 粘贴图片一键提取文字」（2026-08-09）

> 便捷入口闭环：截图/扫描件贴进对话后，点图片附件的「提取文字」即可用本地 OvisOCR2
> 常驻服务抽出图中文字，作为用户消息发给助手——不用再靠模型读图，离线、秒级。
- 新增 GaeaOCRText 绑定（docmd.OCRImageText）：复用常驻 llama-server（自动拉起/共享端口），
  图片不存在或服务不可用时给出明确错误
- Composer 图片附件新增「提取文字」按钮（FileText 图标，位于「识图」旁），识别中显示 spinner，
  完成后以【图片文字提取：文件名】发送给助手；与既有「识图」（本地视觉模型）互补——
  识图给描述、提取文字给原文
- 验证：docmd 新增 OCRImageText 端到端测试（GAEA_TEST_OCR_IMAGE 门控，实测 0.76s 返回
  「项目周报…营收 120 万元…」）；go test ./... 全绿、tsc + vite build 通过、vitest 47 例全过

## v2.7.8「扫描件 OCR · 常驻服务 + format_convert 闭环」（2026-08-09）

> 多页扫描提速 + 内置转换器闭环：llama-server 常驻（按需拉起、共享 8137 端口），
> 多页 PDF 不再每页重载模型（单页约 5s → 整页 2.4s）；format_convert/文件预览的扫描件
> 兜底从「提示装 tesseract」改为直连本地 OvisOCR2 服务，tesseract 降为兜底。
- 常驻服务：ocr_local.py 与 docmd 共用同一端口（GAEA_OCR_URL/GAEA_OCR_PORT 可覆盖），
  健康检查发现未运行即按需拉起 llama-server（Vulkan，隐藏窗口），多页逐页复用；
  路径用 GAEA_OCR_DIR / GAEA_OCR_LLAMA / GAEA_OCR_MODEL / GAEA_OCR_MMPROJ 覆盖
- format_convert 闭环：docmd.ocrPDF 先试 OvisOCR2 服务（按页 POST），不可用时退回
  tesseract；findPdftoppm 探测真实 poppler exe（本机 PATH 里的 .cmd 包装器指向
  不存在路径，直接执行会失败，回退 codex 运行时自带的 pdftoppm.exe）
- 修复扫描件被误当文本：extractRawText 曾把 PDF 对象字典/trailer/ICC/图像流的可打印
  垃圾当正文返回，OCR 永远不触发——新增 stripNonTextStreams（关键字感知流扫描，只保留
  含 BT/ET/Tj 的文本流）+ 内容质量门槛（按空白分词、拒绝高熵符号串），扫描件正确落入 OCR
- 文本提取增强：extractPDFText 支持 TJ 数组的十六进制字符串（UTF-16BE，Word/LibreOffice
  中文 PDF 常见），带 BOM/无 BOM/ASCII 自动判别
- 验证：docmd 新增 httptest 客户端、hex 解码、流剥离、端到端扫描件（env 门控）测试；
  go test ./... 全绿；pdf 技能已同步 ~/.codex/skills 镜像

## v2.7.7「扫描件 OCR · OvisOCR2 本地文档解析」（2026-08-09）

> 按用户建议安装专用本地 OCR 模型：OvisOCR2（0.8B 端到端文档解析，Qwen3.5-0.8B 后训练，
> OmniDocBench 96.58，GGUF + llama.cpp Vulkan，Apache-2.0）成为扫描件 OCR 首选通道，
> 直接输出 Markdown（含 LaTeX 公式与表格），中文印刷体/表格识别显著强于 WinRT。
- 安装：llama.cpp b10333（Vulkan，适配 Radeon 8060S 核显）+ OvisOCR2-Q5_K_M（578MB）+
  mmproj-F16（205MB），落位 C:\AI\gaea-ocr（约 850MB）
- ocr_local.py 新增 --mode ovis 与自动链路：auto = OvisOCR2 → RapidOCR → WinRT → 本地视觉模型，
  文本过短/为空时逐级降级；路径可用 GAEA_OCR_DIR / GAEA_OCR_LLAMA / GAEA_OCR_MODEL /
  GAEA_OCR_MMPROJ 环境变量覆盖
- 顺带补齐 RapidOCR（PP-OCR 转 ONNX，隔离 venv C:\AI\gaea-ocr-env）作为 Ovis 缺失时的离线兜底
- 实测：真实扫描 PDF（微软雅黑渲染后嵌图）→ Ovis 识别「项目周报：本周完成 8 项需求 /
  营收 120 万元，同比增长 18% / 修复目标：砷 ≤ 60 mg/kg（GB 36600-2018）」，
  公式转 LaTeX、单页约 5s（含模型加载）；WinRT / RapidOCR 通道同步验证
- 验证：pdf 技能与 ~/.codex/skills 镜像同步；go build/test 不受影响

## v2.7.6「性能专项 · 大文件转换提速」（2026-08-09）

> P1「性能专项」落地：先用基准量化 format_convert 热点再针对性优化——
> docx→Markdown 百页级文件提速约 12 倍（69ms→5.8ms），PDF 提速约 4 倍（5.0ms→1.2ms）。
- 根因：docxToMarkdown 主循环每次迭代都对剩余全文搜索 `<w:p>` 与 `<w:p ` 两种形式，
  带属性形式在多数文档中不存在，导致 O(n²) 全文扫描（CPU profile 显示 findOpenTag 占 83%）
- 修复：新增 nextOpenTag 单趟扫描（按 `<` 推进 + 标签名边界判断），一次定位最早的段落/表格标签；
  PDF 页数统计从逐 rune 转 string 的热循环改为 strings.Count("/Type /Page")；
  xlsx 工作表按名建索引替代逐 sheet 全文件扫描，sharedStrings 索引解析改用 strconv.Atoi
- 基准（新增 docmd bench_test.go：gooxml 构造百页 docx / 千行 xlsx / 1500 页 PDF）：
  docx 300 段 + 20 表 69ms→5.8ms（约 12x）；PDF 1500 页 5.0ms→1.2ms（约 4x）；xlsx 1000x10 微调
- 验证：docmd 既有用例全部通过（表格 round-trip / 属性段落），go test ./... 全绿

## v2.7.5「方案校验 · 原文定位 + 整改建议」（2026-08-09）

> P1「方案校验规则引擎」增强：全面检查从「报问题」升级为「指出在哪、怎么改」——
> 每条发现带整改建议与原文定位（章节 + 上下文摘录 + 一键跳转）。
- CheckItem 新增 Suggestion（整改建议）与 Locations（原文定位：sectionId/excerpt/offset）：
  废标条款响应、数据一致性（缺失事实/工期冲突/暗标单位）、跨章节重复、暗标格式（加粗/删除线/emoji）、
  规范引用（未引用/编号存疑）逐条补齐可照做的整改建议
- 原文定位：locateNeedle 忽略空白差异在章节内定位命中，excerptAround 按 rune 截取上下文摘录
  （不切坏 UTF-8，前后省略号）；重复检测给出主章节与重复章节两处定位；AI 评分覆盖检查的
  suggestion 同步透传到整改建议
- 前端：校验报告新增「整改建议」行与原文定位 chips（摘录 + 定位按钮，点击跳转对应章节）
- 验证：新增 check_advice 测试（建议/定位/双章节/覆盖建议透传/摘录 UTF-8 安全）；
  go test 全绿、tsc + vite build 通过、vitest 47 例全过

## v2.7.4「事实底座 · 长期记忆沉淀」（2026-08-09）

> P0「记忆与自进化」落地：事实底座不再止步于会话内——一键把交付前沉淀的事实写入长期记忆，
> 后续对话自动加载，越用越懂你、不用反复交代。
- 新增 GaeaFactBasePromote 绑定 + 侧栏「沉淀为长期记忆」按钮：把当前会话事实底座逐条写入
  memory 存储（kind=semantic），按稳定 ASCII slug 去重（中文 key 走 hash 兜底），
  重复沉淀同 key 原位更新不产生重复条目；preference 分类映射为用户画像（type=user），其余为项目事实
- 面板按钮点击后 toast 反馈沉淀条数；事实底座本身保留，可继续编辑/清空
- 验证：新增 promote 单测（写入/去重更新/中文 slug 稳定性/分类映射/单行摘要）；go test 全绿、
  tsc + vite build 通过、vitest 47 例全过

## v2.7.3「事实底座 · 一稿多用」（2026-08-09）

> P0「多形态成果交付」核心闭环：通用办公引入 任务 → 事实底座 → 多形态交付 的默认工作流，
> 交付物（docx/pptx/xlsx/图表）基于同一事实底座生成，跨交付物保持一致、可回看、可复制。
- 新增 fact_add / fact_list / fact_clear 三个会话级事实底座工具（ExtraTool）：
  fact_add 按 key 沉淀/修正事实（含来源与分类，空值即删除），fact_list 输出 Markdown 表格，
  fact_clear 开启新任务时清空；事实按会话持久化到 .gaea/sessions/<session>-facts.json
- 侧栏新增「事实底座」面板：事实列表（key/值/来源）+ 复制 Markdown + 清空，随 fact_* 工具结果
  与回合结束实时刷新；删除会话时事实底座同步清理
- 办公提示词新增「事实底座（一稿多用）」纪律：交付任务先沉淀后交付，交付物基于同一底座生成，
  事实有更新先 fact_add 修正再重新生成，新任务先 fact_clear 隔离上一任务事实
- 验证：新增 factbase 包单测（upsert/清空/Markdown 转义/round-trip/路径）+ app 侧工具与绑定测试；
  go build/test 全绿、tsc + vite build 通过、vitest 47 例全过

## v2.7.1「办公三件套核心闭环修复」（2026-08-09）

> 触及核心：修复 Word/Excel/PDF 三个工具的真实断点，端到端实测打通。
> 此前通用办公的 docx 技能依赖未安装的 node docx-js 与 pandoc，Excel 公式重算在 Windows
> 上因 soffice 不在 PATH 而失败——这三个断点全部修复并补回归测试。
- format_convert（docmd）：修复 docx→Markdown 表格解析把 `<w:tcPr>` 等 XML 当文本的 bug；
  `findWt` 偏移错位导致段落/单元格文本错乱；兼容带属性段落（w14:paraId）与 `<w:tabs>` 内嵌标签
  （新增 TestConvertDocxTableRoundTrip / TestConvertDocxAttrParagraphs 回归测试）
- docx 技能：新增 scripts/create_docx.py（python-docx，环境已装）——Markdown/JSON spec → 排版 Word
  （A4/Letter、封面、目录域、标题层级、表格、列表、页眉页脚、页码、图片），替代未装的 docx-js 成为主路径；
  读取兜底改为 gaea 内置 format_convert；SKILL.md 依赖清单同步更新
- xlsx 技能：scripts/office/soffice.py 增加 Windows 安装路径自动探测（Program Files / (x86) / LOCALAPPDATA），
  recalc.py 不再因 soffice 不在 PATH 失败；SKILL.md 补批量合并/统一格式流程
- docx 技能 soffice.py 同步路径探测（docx→PDF 转换同样受益）；pdf 技能补 PDF→Word 与表格→xlsx 闭环流程
- 实测：Markdown→docx→format_convert 表格 round-trip 正确；openpyxl 写公式→LibreOffice 重算 4 公式 0 错误
  （SUM/差额/利润率读回 220/130/90/40.9%）；reportlab PDF→pdfplumber 表格提取→xlsx 结构完整
- 验证：go test ./... 全绿（docmd 新增 2 例）；技能修改已同步镜像 ~/.codex/skills

## v2.7.2「本地模型优势落地 · 扫描件本地 OCR」（2026-08-09）

> 善用 gaea 本地模型资产：扫描件 PDF 的 OCR 不再依赖未安装的 pytesseract，
> 改走本地双通道——Windows 原生 OCR（WinRT，离线零成本）+ 本地视觉模型（herdsman Qwen3.6）兜底。
- pdf 技能新增 scripts/ocr_local.py：PyMuPDF 渲染 PDF 页（300 DPI）→ windows-ocr.ps1（WinRT zh-Hans）
  → 文本过短/为空时降级本地视觉模型（GAEA_VISION_BASE_URL/GAEA_VISION_MODEL 可覆盖）；
  输出 UTF-8 文本或 JSON（每页 + 工具来源），支持 --mode auto|winrt|local
- pdf 技能自带 scripts/windows-ocr.ps1（WinRT OcrEngine 离线调用，与 ds-vision-skill 同源）
- SKILL.md：扫描件 OCR 节改为本地 OCR 主路径，pytesseract 降为可选外部方案
- 办公提示词新增「本地能力优先」纪律：扫描件/图片文字提取优先本地 OCR 或 vision，敏感文档本地处理不出机
- 实测：中文 docx→PDF→本地 OCR 识别出「项目周报 / 本周完成 8 项需求 / 营收 120 万元，同比增长 18%」；
  表格 PDF→OCR 数字与表头全部识别；英文/中文双链路均离线跑通
- 验证：技能修改已同步镜像 ~/.codex/skills

## v2.7.0「通用办公强化 · PPT 交付 + 技能沉淀 + 便捷入口 + 后台任务」（2026-08-09）

> 按《市场调研：通用办公 AI 智能体》P0 结论落地：新增 pptx 演示文稿技能（python-pptx 一键成稿）、
> 欢迎页新增「演示文稿」入口、一次成功对话一键沉淀为可复用技能、粘贴剪贴板图片即转图片附件上下文、
> 侧栏后台任务面板（运行中任务 + 完成 toast）、办公提示词增强（PPT 交付纪律 + 偏好主动记忆沉淀）。
- pptx 技能：新增 `.gaea/skills/pptx` 并镜像 `~/.codex/skills/pptx`（SKILL.md + scripts/create_pptx.py，
  16:9 封面/章节要点/演讲备注，缺 python-pptx 时自动 pip 安装，生成后回读校验页数）
- 前端入口：欢迎页核心能力新增「演示文稿」卡 + 内置技能新增 pptx chip；icons 兼容层新增 FilePpt
- 提示词：SingleModelPrompt 执行纪律加入 pptx 技能与演示大纲流程；新增「记忆沉淀」检查点
  （用户偏好/项目事实/踩坑经验主动 remember，避免重复交代）
- 便捷入口：Composer 粘贴剪贴板图片（截图/网页图）不再静默丢弃，转为图片附件上下文（复用 SavePastedImage）
- 后台任务：侧栏新增「后台作业」面板（运行中 bash/task 列表 + 状态点 + 相对时间，仅展开态显示）；
  JobDoneNotifier 在任务从运行列表消失时自动 toast「后台任务已完成」
- 技能沉淀：助手消息新增「沉淀为技能」操作，弹窗预填本次任务/回答，确认后写入 .gaea/skills/<name>/SKILL.md
  并镜像 ~/.codex/skills，热加载后立即以 /技能名 调用（GaeaCaptureSkill + skill.RenderSkillFile，含校验/覆盖/测试）
- 验证：go build/test 全绿（含 skill capture 新用例）、tsc + vite build 通过、vitest 47 例全过、
  pptx --demo 端到端生成 3 页演示文稿

## v2.6.9「搜索修复 · 人格重设计 · 移除 VoxCPM2」（2026-08-09）

> 通用办公搜索修复（新增 Bing/DDG 兜底 + 代理接入，不再报「所有搜索引擎失败」）；
> 聊天板块联网搜索接通（ChatSend searchEnabled + 开关生效）；gaea 人格与提示词整体重设计
> （统一 gaea 身份，清除 Ackem/Hermes/大地女神旧文案，清空 config.toml 土壤修复旧覆盖）；
> 首页语音固定 gaea；按用户要求移除实测不达标的 VoxCPM2，本地 TTS 保留 CosyVoice2。
- web_search：Bing/DuckDuckGo Lite 无 key 兜底、代理复用 web_fetch 链路、修复 cancel bug 与 429 重试、UA 换浏览器标识
- 聊天搜索：ChatSend(searchEnabled) 普通+角色对话自动联网注入结果；前端开关生效（普通模式也显示）；whisper WebSearch Bing 优先约 0.3s 带标题/链接
- 人格：personality/canon/product_identity/main_chat 提示词重写，gaea 预设去 goddess 标签；20+ 处 Hermes/Ackem → gaea；AboutPanel/首页语音标签同步
- 首页语音：固定 gaea，删除聊天人格自动同步语音的副作用
- 移除 VoxCPM2：引擎/前端 6 文件删除，本地 C:\AI\voxcpm、C:\AI\llama-omni 清理，释放约 14GB
- 验证：go build/test 全绿、tsc+vite build 通过、wails build 成功（releases/gaea-v2.6.9.exe + SHA256SUMS）

## v2.6.8「模型中心一键启动本地 TTS 服务」（2026-08-09）

> 本地 TTS 不再需要手动运行启动脚本：gaea 启动时自动保活 CosyVoice2 / VoxCPM2；
> 模型中心点击模型的「启动」按钮也会即时拉起对应服务；「测试连接」与语音合成前兜底同样自动拉起。
- 新增 internal/app/tts_service.go：core.ensureLocalTTSService（幂等）+ mediaState.StartLocalTTSService（Wails 绑定）
- 探测：CosyVoice2 http://127.0.0.1:8010/v1/models；VoxCPM2 http://127.0.0.1:8020/v1/status
- 拉起（隐藏窗口 CREATE_NO_WINDOW，不阻塞 UI）；异步轮询就绪（cosyvoice ≤10s / voxcpm ≤180s），就绪/失败通过 tts-service-status 事件通知前端
- 验证：go build/test 全绿、tsc + vite build 通过、wails build 成功（releases/gaea-v2.6.8.exe + SHA256SUMS）

## v2.6.7「VoxCPM2 Vulkan GGUF 加速 · 4 音色替换」（2026-08-09）

> 实测 VoxCPM2 三层架构落地：8030 llama-tts-server（llama.cpp-omni + Vulkan，Q8_0+F16 GGUF）为主后端；
> 8021 ROCm PyTorch 备胎；8020 adapter.py 统一入口（gaea 零改动）。
- 关键修复：SSLServer 空证书导致 bind 失败（改普通 httplib::Server）；AudioVAE 参考特征 frame-major 布局对齐 Python（克隆从近静音恢复）
- 性能：短句克隆 RTF 0.65–0.84（6 步/CFG 1.5），语音设计 0.57–0.60；对比 ROCm 5 步 RTF ≈0.06–0.12，整体快 1.5–8 倍
- 音色：CosyVoice / VoxCPM2 统一替换为火山引擎 4 音色（中文女/男、英文女/男）；参考音频 ≤3s/16kHz，适配器自动音量归一
- 验证：go build/test 全绿、tsc + vite build 通过、四音色端到端实测通过；wails build 成功（releases/gaea-v2.6.7.exe + SHA256SUMS）
## v2.6.6「VoxCPM2 本地语音引擎接入」（2026-08-09）

> 模型中心新增「VoxCPM2 (本地)」引擎：本地 OpenAI 兼容 TTS 服务（127.0.0.1:8020），
> 2B 扩散式多语种 TTS、48kHz 高保真输出，内置 7 音色 + 参考音频零样本克隆 + 声音设计；
> ROCm 驱动 Radeon 8060S 核显 + TunableOp 调优，实测 RTF ≈ 2.0。
- 引擎接入：modelengine 新增 `voxcpm` 类型与内置引擎，启动自动补齐 `VoxCPM2` 模型；
  「语音模型」页与「功能绑定 → 聊天语音」均可选择
- 服务端（`C:\AI\voxcpm\server.py`）：OpenAI 兼容 `/v1/audio/speech`、
  `/v1/audio/info`、`/v1/models`、`/v1/voices`（参考音频注册克隆音色，持久化）；
  Python 3.12 venv + torch 2.9.1+rocm7.2.1（AMD ROCm Windows）+ voxcpm 2.0.3
- 性能：ROCm 核显推理 + TunableOp GEMM 调优缓存（4~5 倍）；CFG 1.5 避免长文本跑飞重试；
  实测 7.7s 中文约 16s（RTF ≈ 2.0），短句 RTF ≈ 1.35；启动预热后首个请求即达稳态
- 音质：48kHz 输出；内置 7 个与 CosyVoice2 同源参考音色（中文女/男、英文女/男、日语男、粤语女、韩语女）；
  支持声音设计（自然语言描述音色）与参考音频克隆
- 验证：go build/test 全绿、tsc 通过；服务端直连合成 / 克隆 / 音色列表实测通过
  （详细记录：`docs/2026-08-09-voxcpm2-integration.md`）

## v2.6.5「CosyVoice2 LLM 核显加速」(2026-08-09)

> CosyVoice2 LLM 环节从 PyTorch CPU 切换到 llama.cpp GGUF + Vulkan（Radeon 8060S 核显），
> 整条合成管线提速约 8–10 倍：短句 6.5s→~1.5s，长句 24.5s→~2.8s。
> 默认使用 f16 GGUF（音质最接近原始权重），q8 为更快备选。
- 调研：Tinysoft/Cosyvoice2-0.5B-GGUF + llama.cpp PR #14711（Qwen2 bias 支持）
- 引擎：`cosyvoice/llm/gguf_engine.py`（token 直喂、KV cache、复刻 ras_sampling），
  `Qwen2LM.load_gguf()` 接入，实测词表布局 logits 相关性 0.9999
- 服务：server.py 默认加载 `gguf\cosyvoice_f16.gguf`（环境变量 `COSYVOICE_LLM_GGUF` 可切），
  启动预热 shader；失败自动回退 torch
- 修复：`mask_to_bias` fp16 下 -1e10 溢出 -inf 导致 ONNX Softmax NaN（fp16 改用 -3e4）
- 否决：fp16 flow estimator（误差累积，波形相关性 0.016）、LLM 单步 ONNX（链式发散）、
  bf16（EOS 失效）、flow 4 步（无稳定收益）
- 验证：直连 `/v1/audio/speech` 短句 ~1.4–1.8s、长句 ~2.8s；7 音色/中英日正常
  （详细记录：`docs/2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md`）
- 发布：`go build/vet/test` + `tsc/vite build` + E 系列回归全绿；`wails build` 成功，
  `releases/gaea-v2.6.5.exe` + `SHA256SUMS-v2.6.5.txt` + `v2.6.5.md`，桌面 `gaea.exe` 已同步

# gaea · 多功能 AI 助手

## v2.6.4「CosyVoice2 提速 + 模型中心音色选择」(2026-08-09)

> CosyVoice2 合成提速约 35%（短句 6.5s → 4.1s）：flow 解码器切 ONNX + DirectML（AMD 核显），
> 采样步数 10→5，音质与 torch 路径相关性 1.0000；
> 「功能绑定 → 聊天语音」卡片新增音色选择（xAI / CosyVoice2 / Herdsman）。
- CosyVoice2 提速：`flow.decoder.estimator.fp32.onnx` + `DmlExecutionProvider` 替换 torch estimator，
  flow 步数 5；服务端 server.py 内置，重启服务即生效
- LLM 优化探索：Qwen2 单步 ONNX 导出成功（fp32 CPU 28.8ms/步、DML 10ms/步），
  但链式推理数值发散导致音频失真，暂不启用；bf16 破坏 EOS 采样，弃用
- 模型中心：聊天语音绑定卡新增「音色」下拉（xAI 26 音色 / CosyVoice2 7 音色 / Herdsman 服务端列表），
  选择即写 `tts_voice` 持久化
- 验证：直连 4.1s/0.92s wav；go test 全绿；tsc + vite build 通过
  （releases/gaea-v2.6.4.exe，SHA256SUMS-v2.6.4.txt）

## v2.6.3「CosyVoice2-0.5B 本地音色克隆接入」(2026-08-08)

> 模型中心新增「CosyVoice2 (本地)」引擎：本地 OpenAI 兼容 TTS 服务（127.0.0.1:8010），
> 聊天语音可绑定 CosyVoice2-0.5B，内置 7 个音色并支持参考音频零样本克隆。
- 引擎接入：modelengine 新增 `cosyvoice` 类型与内置引擎，启动/刷新自动补齐 `CosyVoice2-0.5B` 模型；
  「语音模型」页与「功能绑定 → 聊天语音」均可选择
- 音色支持：`/v1/audio/info` 拉取音色列表（中文女/男、英文女/男、日语男、粤语女、韩语女），
  设置面板/聊天设置新增 CosyVoice 选择器；无效音色回退「中文女」，
  `defaultVoiceForModel`/`ttsVoiceForModel` 补齐 cosyvoice 默认音色
- 服务端：`POST /v1/audio/speech`（OpenAI 兼容）+ `POST /v1/voices`（参考音频注册新音色，持久化到 spk2info.pt）
- 实测：直连合成 6.5s/0.92s 音频（RTF 6.8x，PyTorch CPU）；gaea 客户端端到端 38KB wav；
  go test 全绿、tsc + vite build 通过
  （releases/gaea-v2.6.3.exe，SHA256SUMS-v2.6.3.txt）

## v2.6.2「xAI Grok TTS 接入」(2026-08-08)

> xAI 引擎内置 `grok-tts` 云端语音模型：模型中心「功能绑定 → 聊天语音」可绑定 xAI，
> 语音对话/朗读优先走 Grok 音色（Eve/Ara/Rex/Sal/Leo + 旗舰 21 个），
> 实测合成约 1.4~2.4 秒，快于本地 qwen3-tts（声码器 CPU）的 2.7~3.3 秒。
- xAI TTS 路由：语音管道新增 `tryEngineTTS`，xAI 走 `POST /v1/tts`（复用 OAuth token），
  聊天语音绑定 → 全局 TTS → 自动路由均生效；流式朗读选中 xAI 时同样优先云端
- 模型中心：xAI 引擎保活内置 `grok-tts`（启动/刷新均自动补齐），
  功能绑定与「语音模型」页可直接选择；设置面板/聊天设置新增 xAI 音色选择器，
  无效音色自动回退 `eve`，`tts_voice` 持久化
- xAI TTS 客户端完善：`SynthesizeWithMime` 返回真实 MIME，音色列表静态校验 + 单测
- 验证：`go build/test ./internal/{tts,modelengine,app}/...` 全绿；`tsc` + `vite build` 通过；
  直连 `api.x.ai/v1/tts` 实测 HTTP 200 / audio/mpeg；`wails build` 成功
  （releases/gaea-v2.6.2.exe，SHA256SUMS-v2.6.2.txt）

## v2.6.1「语音交互打通 · 音色可配置 · 首页语音角色与聊天一致」(2026-08-08)

> 按 Herdsman 新版接口重新打通语音对话：AI 回复后 TTS 朗读恢复正常，不再卡在“正在聆听”；
> 语音音色可在设置面板直接选择并持久化；首页语音角色与聊天板块保持一致。
- 语音交互打通：TTS `audio_url` 支持 data URI / 相对路径 / 绝对 URL 三种形态并返回真实 MIME；
  合成前动态查询 `/v1/audio/info` 音色列表，`Cherry` 等已移除音色自动回退可用音色；
  修复 ASR `/v1/v1/audio/transcriptions` 双重前缀 404；TTS 播放期间保持“AI 回复中”状态
- 音色设置：聊天面板「语音设置」新增 Herdsman 音色选择器（实时拉取服务端支持列表），
  写入 `~/.gaea_config.json` 的 `tts_voice` 持久化；设置页聊天设置同步
- 首页语音角色：不再写死 gaea，读取聊天板块保存的同一角色键并动态显示标签；
  后端新增 `voice_personality` 持久化
- 普通对话联动：聊天板块切换「普通对话」时语音回复使用中性助手口吻，
  首页语音同步为「普通对话」；切换人格时首页语音跟随该人格
- 模型中心「功能绑定」新增聊天语音模型：单独绑定聊天语音的引擎+模型（持久化），
  未绑定回退全局 TTS；语音模型列表随引擎自动刷新，便于后续扩展
- 验证：go build/vet 全绿；tsc + vite build OK；前端 E 系列回归守卫 OK；
  直连 Herdsman 端到端验证合成 122KB 真实语音；wails build 成功
  （releases/gaea-v2.6.1.exe，SHA256SUMS-v2.6.1.txt）

## v2.6.0「小说创作工作台重设计 · 角色卡补齐/剧照/合并」(2026-08-08)

> 小说创作页改为三栏写作工作台（章节树 / 编辑器 / 创作设置），写作技能与生成参数
> 常驻可调；章节捕获的角色卡补齐 AI 补档、剧照生成与同名合并；修复设定未注入、
> 办公目录被误判为代码工程等问题。
- 小说创作界面：三栏写作工作台——左章节树（未写/生成中/已写入状态点）、中纸面化
  编辑器（标题/状态/字数/重写/保存/空状态引导）、右创作设置面板（设定摘要/技能/
  生成参数/剧情方向/统计）；技能选择常驻并显示说明与适用场景
- 生成设置真实生效：目标字数（不足自动续写）与生成温度由界面传入后端
  （CreateChapter 新增参数，默认 5000 字/服务端默认温度）
- 正文标题约束：提示词禁止 AI 自拟章节标题或编号行，章节编号由系统管理，
  杜绝「第七章…这是第一章」错乱
- 小说设定注入修复：创作页在挂载/切书/开向导/生成前均重读最新设定，提示词必注入
- 章节角色捕获：新角色只写本书 characters.json，需手动「一次性迁移」进全局库；
  角色页未入库标记 + 刷新联动
- 角色卡能力：未入库角色支持 AI 补齐空缺字段、生成剧照（自动补全关键描述）、
  合并同一人的不同称呼（引用与关系重定向去重）
- 通用办公：Profile.Scan 语言改为按工程清单真实检测，办公目录不再被标成 Go 工程；
  项目画像/技能/记忆基于工作区扫描；开发 mock 的 coding agent 提示词改为办公助手
- 技能加载：修复 SKILL.md 多行 applies_to 解析为空的问题
- 验证：go build/vet/test 全绿、tsc + vite build OK、vitest 47 例全过、
  wails build 成功（releases/gaea-v2.6.0.exe，SHA256SUMS-v2.6.0.txt）

## v2.5.0「全界面科幻视觉重设计 · 绘梦图生图/文生视频 · 角色库模型绑定」(2026-08-08)

> 设置、首页、聊天欢迎屏、小说、绘梦五大板块统一为「玻璃 HUD + 单一强调色」设计语言；
> 绘梦新增图生图与文生视频（ComfyUI 后端）；模型中心功能绑定实时同步并新增角色库绑定；
> 角色库剧照支持独立后端/模型；本地视觉识别管线修复 UTF-8 编码并切换到 Qwen 视觉模型。
- 设置面板：左侧导航改为顶部平铺分类磁贴（吸顶可切换），统一 HUD 角标/聚焦/按压反馈；
  修复 flex + margin auto 导致容器收缩为内容宽度的问题（宽度 641px → 1080px）
- 首页：语音中枢升级为视觉主角——HUD 角标框、雷达脉冲环、声谱均衡条、等宽遥测读数，
  聆听/回复时扫描光带掠过面板；新增 380px 语音球与实时音量展示
- 聊天欢迎屏：普通模式用 VoiceChatOrb、角色模式用 CompanionAvatar（随语音状态变色），
  配 HUD 角标、脉冲环、遥测行；建议卡升级为键盘可达（role/tabIndex/focus-visible）；
  角色模式空状态隐藏重复人格条，修复 orb 被 flex 压缩的问题
- 小说板块：默认 Tabs 改为顶部二级导航平铺（书架/设定/角色/创作/阅读/导出），子页保持挂载；
  新增统一设计层 novel-workspace.css——玻璃面板、HUD 面板头、章节节点树、纸面化衬线编辑面、
  antd Tabs 玻璃化；大纲面板硬编码紫色改为 var(--gaea-glow)；设定页改为左右双栏
  （左设定编辑器 + 右设定 Agent 常驻，修复 flex 宽度收缩）
- 绘梦：顶部三模式导航（文生图 / 图生图 / 文生视频）；图生图支持上传/拖拽参考图 + 重绘幅度；
  文生视频支持分辨率/时长帧率预设；后端新增 ComfyUI 图生图工作流（LoadImage+VAEEncode+低 denoise）
  与 LTX-Video 文生视频工作流，输出解析扩展到 images/gifs/videos，结果舞台/历史胶片/灯箱支持视频；
  新增 GenerateMedia 绑定；修复引擎列表为空时页面崩溃
- 模型中心：功能绑定面板监听 feature-model-changed 事件实时同步（其他页面改绑定即时刷新）；
  聊天合并轻语、办公合并方案（office+gaea 双写），删除重复行；新增角色库 LLM 绑定
  （func_characterlib_*，角色生成/补全走独立绑定）
- 角色库剧照：新增独立图片后端/模型绑定（portrait_backend/portrait_model，空=跟随绘梦），
  模型中心新增「角色库剧照」卡；删除小说卡上的剧照模型标签
- 视觉识别管线：vision 技能默认模型切换为 Qwen3.6-35B-A3B-Uncensored（布局识别准确率明显提升）；
  修复 PowerShell 5.1 下 UTF-8 编码问题（脚本转 BOM + 改用 .NET HttpClient），中文输出不再乱码
- 验证：go build/vet/test 全绿、tsc + vite build OK、vitest 37 例全过、
  Edge headless 各界面实测 + 视觉模型复核

## v2.4.5「通用办公欢迎页重设计」(2026-08-08)

> 通用办公对话窗口的欢迎界面从土壤修复时代的旧版，重设计为贴合当前定位的通用办公入口。

- 内容更新：移除场地调查/投标标书/修复方案/污染风险/成本测算等土壤修复专项卡片，
  改为 6 个通用办公核心能力（文档撰写/表格处理/格式转换/图表生成/报告拼装/知识沉淀）
- 新增内置技能区：format-convert / chart-builder / doc-assemble / docx / xlsx / pdf 六个技能 chips，
  点击即填入对应任务提示词
- 视觉升级：主视觉改为「GAEA OFFICE + 今天想做什么？」大标题层级，logo 带光晕角标，
  能力卡 hover 顶部高光线 + 箭头提示，新增渐次入场动画（尊重 prefers-reduced-motion）
- 保留工作区/模型 pill、最近会话区并同步优化样式
- 文案清理：welcome.tagline 与加载页 skeleton 描述从「土壤修复引擎」改为通用办公，
  删除 locale 中无人引用的土壤修复快捷卡片条目
- 验证：tsc + vite build OK；vitest 37 例全过；Edge headless 实拍确认布局正常

## v2.4.4「文件预览与右侧面板精简」(2026-08-07)

> 右侧面板去掉「消息/报告」标签并默认折叠；对话内可直接点击文件路径打开全新的预览阅览面板。

- 右侧面板：删除「消息」与「报告」标签页（MessageNavigator / ReportPreviewPanel），保留「文件」「统计」；
  面板改为默认折叠，点工具栏按钮展开
- 对话内文件预览：Markdown 渲染中的本地文件路径（.md/.docx/.xlsx/.pdf/.png 等）渲染为可点击的文件 chip，
  用户消息里的 @ 附件同样可点击；点击打开居中大尺寸预览弹层（Esc/遮罩关闭，支持定位与外部打开）
- 预览面板重设计：后端新增 GaeaPreview 统一预览接口——图片返回 dataURL、Markdown/文本原文渲染、
  docx/xlsx/pdf 经 docmd 转 Markdown 内联预览（含 OCR 回退），不支持格式给出外部打开入口；
  工作区「文件」面板的预览同步升级为 Markdown 渲染
- 转换引擎抽取：format_convert 的 docx/xlsx/pdf→Markdown 逻辑迁至 internal/office/docmd 包，
  工具与预览面板共用一份实现；修复 openpyxl 内联字符串单元格（inlineStr）丢表头的问题
- 测试：新增 FilePreviewModal 3 例 + Markdown 本地文件链接 2 例，前端 37 例全过；go test 全绿

## v2.4.3「精简内置工具集」(2026-08-07)

> 按市场调研结论（同类产品 10~20 个核心工具、文档/表格用技能扩展）把内置工具从 38 个精简到 17 个核心工具。

- 删除 21 个冗余内置工具：计算器（calc_math/calc_stats/calc_unit）、压缩（archive）、
  电脑操作（computer_use）、甘特图（gantt_gen）、项目初始化（project_init）、工具链模板（run/save_template）、
  方案 agent 工具（proposal_list/write/export），以及被 ModelScope docx/pdf/xlsx 技能覆盖的文档专项工具
  （docx_read/docx_write/pdf_create/pdf_extract/pptx_create/xlsx_read/xlsx_write/doc_merge/csv_parse）
- 删除与 run_skill 重叠的 parallel_skills（保留 RunDAG 管道基础设施）
- 保留 17 个核心工具：文件与命令（read_file/write_file/ls/bash/bash_output/kill_shell/wait）、
  网络（web_search/web_fetch）、任务（todo_write/complete_step）、记忆与知识（memory_search/knowledge_add/knowledge_search）、
  技能（read_skill）、办公引擎（format_convert/chart_gen）
- 内置技能去工具依赖：chart-builder 改为 bash + python 读表、doc-assemble 改为 format_convert + bash 拼装，
  不再依赖已删除的 xlsx_read/csv_parse/doc_merge/docx_write
- 系统提示词与单模型执行纪律改写为「文档创建/编辑交给 docx/xlsx/pdf 技能，格式转换用 format_convert」
- 修复 chart_gen 在 Windows 被 python3 商店别名劫持的问题，并抑制 matplotlib 字体告警污染 JSON 输出
- 前端能力抽屉工具列表重建为实际 7 组 29 个工具（compact 模式对模型隐藏 kill_shell/wait），
  清理 git/notebook/edit 等死条目与报告来源旧映射
- 验证：go build + go test ./... 全绿；tsc + vite build OK；vitest 32 例全过；format_convert 与 chart_gen 端到端验证通过

## v2.4.2「通用办公改造 · ModelScope 文档技能」(2026-08-07)

> 「智能办公」精简为「通用办公」：删除土壤修复专项工具与技能，安装 ModelScope 的 docx/pdf/xlsx 文档技能。

- 办公定位：DefaultSystemPrompt 与单模型执行纪律改写为通用办公助手（文档/表格/图表/演示/检索/方案报告/任务跟踪）
- 删除 6 个土壤修复专项工具：survey_report/bid_proposal/imple_plan/spec_query/spec_judge/material_query，
  并从 compact 描述/模式表中清理
- 内置子代理技能收敛为 3 个通用技能：format-convert / chart-builder / doc-assemble（含 wrapper 工具与测试）
- 删除项目内 5 个土壤办公技能（site-survey/risk-assessment/remed-design/bid-package/data-report），保留 skill-creator
- 安装 ModelScope 技能：docx / pdf / xlsx（SKILL.md + 完整 scripts/schemas/templates），
  同时写入 ~/.codex/skills 与 .gaea/skills，供 Codex 与 gaea 通用办公使用
- 前端文案统一：智能办公→通用办公，报告类型映射改为通用办公类型，移除 Hephaestus 残留
- 验证：go vet + go test ./... 全绿；tsc + vite build OK；vitest 32 例全过

## v2.4.1「设置面板重设计」(2026-08-07)

> 按当前功能板块重构设置中心：左侧模块导航 + 全局搜索，清除死代码与重复面板。

- 布局重设计：顶部 Tab 改为左侧功能分组导航（通用 / 聊天 / 小说 / 绘梦 / 办公 / 模型 / 关于），
  窄屏自动转横向滚动 chips；保留全局搜索并可过滤分组
- 聊天：合并「AI 伴侣 + 默认人格 + 语音核心项」（语音对话/朗读回复/合成音色），
  移除无人读取的「主动搭话」开关与重复的「清除全部会话」；完整语音面板仍在聊天板块
- 办公：并入「方案编写模型」绑定（原方案 Tab 唯一有效内容），删除纯文档的「方案生成」说明
- 模型（新增）：找回此前丢失入口的全局「推理强度」，展示当前模型并提供「前往模型中心」跳转
- 关于：版本卡片 + 系统信息（配置路径）+ 可折叠更新日志，替换原系统 Tab
- 清理：删除死代码 EnginePanel、重复面板 VoicePanel/ProposalPanel、遗留 WhisperPanel/SystemPanel
- 验证：新增 SettingsPage 4 例回归测试（分组渲染/默认分组/搜索过滤/切换），
  `npm run test` 32 例全过，`tsc` + `vite build` OK，`scripts/ci.ps1` CI OK

## v2.4.0「网页调试桥接 · 办公引擎热加载」(2026-08-07)

> 新增 `GAEA_HTTP_PORT` 网页调试桥接：浏览器/手机直接驱动同一个 Go 内核（RPC + SSE），
> 办公引擎支持从磁盘热加载，技能/工具/插件变更无需重启桌面端即可生效。

- HTTP 调试桥接（`internal/httpbridge`）：`POST /api/rpc` 反射分发全部 Wails 绑定方法、
  `GET /api/stream?id=` SSE 事件推送（15s keep-alive）、`/api/health` 存活探针；
  `core.emit` 统一发布到桥接订阅者（无 Wails 上下文也推送）
- 前端 HTTP 模式：`runtimePolyfill` 补齐 EventsOn/EventsOff/EventsOnMultiple/EventsEmit，
  所有事件经 SSE 对齐桌面端——修复并发订阅只连首个事件的竞态，探测失败后桥接就绪可自动重连；
  `bridge.ts` 将 `window.go.app.App.*` 代理到 `/api/rpc`；Vite 将 `/api` 代理到桥接
- 办公引擎热加载：`GaeaReload` 重新读取磁盘持久化配置并重建 controller
  （Agent 参数/权限/沙箱/技能路径/插件），成功后广播 gaea-ready 令前端 store 重新拉取；
  失败时保持旧引擎继续运行，不替换任何状态
- 前端入口：能力抽屉（MCP/工具/技能）新增「热加载」按钮并展示重建后的工具/技能数量；
  设置→办公新增「从磁盘热加载」；三语 i18n 同步
- 验证：新增 httpbridge RPC/SSE 2 例 + GaeaReload 热加载 1 例；`go vet` + `go test ./...` 全绿、
  `tsc` + `vite build` OK、`scripts/ci.ps1` CI OK

## v2.3.0「界面焕新 · 办公整合」(2026-08-07)

> 在 v2.2.0 基础上迭代：角色库/首页/办公板块全面重设计，办公双模块合并为独立二级导航，随机生成能力补全（含人格）。
- 角色库界面重设计：档案墙（立绘主导 + 身份覆盖 + 档案眉），加载骨架/空状态/深浅色适配，去掉多色硬编码统一主题强调色
- 角色命名：28 个内置角色配中文人名（白霜/温言/苏晚晴/林小满/顾清霜/陆寒川/姜棠/谢临川/叶绵绵/许一诺/秦挽月/阮慈/周栗/季如烟/陈恪/沈墨白/程暖阳/席晚棠/霍承渊/虞栀/江野/晏观澜/裴聿修/夏知微/乐桃/顾衍/卫昭），gaea 保留原名
- 角色相卡面板重设计 + 随机生成完善：新增 `CharacterGenerateRandom`（all 全量随机/按字段随机），字段旁 ↻ 骰子单独随机、五维本地随机、性别/定位/状态/年龄即时随机
- 首页 AI 中枢启动器重设计：单一主题强调色、玻璃档案化卡片、加载/焦点/降级动效处理
- 移除窗口顶部流光扫描条（scanline-top / scanSweep）
- 办公板块整合：合并「办公 + 方案编写」为独立二级导航（智能办公/方案编写），`OFFICE_MODULES` 注册表可扩展，移除顶部重复「方案编写」入口
- 智能办公整层重设计：对话区铺满面板（修复全屏缩成一团）、正文行宽约束、清爽玻璃面层（侧栏/顶栏/消息/输入框/滚动条）
- 方案编写整层重设计：胶囊式标签导航、左侧项目导航唯一入口（移除重复「方案列表」标签页）、隐藏重复步骤条，消除层层标题
- 验证：28 项回归测试通过；`go build` + `tsc` + `vite build` OK；`wails build` 成功（build/bin/gaea.exe ~38MB）

## v2.2.0「统一角色库」(2026-08-07)

> 在 v2.1.0 基础上迭代六轮：全局统一角色库、小说单向引用+抽卡、聊天内选角色、角色记忆隔离、状态/记忆/追踪归集角色库、取消「轻语」称谓。

- 角色库架构重构：新增 `characterlib.db` 全局统一角色存储（小说字段 + 聊天字段同一行），内置人格种子化进库；项目只引用角色并携带项目内弧线状态；导入只增不改，杜绝小说反向污染角色
- 小说角色面板改为「抽卡」：从角色库随机抽取（性别/标签/可聊天过滤），不再自行生成角色；面板只编辑项目内定位/弧线/状态，全局设定去角色库
- 聊天只选角色：聊天内 PersonaPicker 选择器（搜索/头像/类型标签），角色管理/编辑/状态/记忆/追踪全部归集角色库
- 角色记忆隔离：事实/情节/图谱按会话恢复与合并写回，A 角色记忆不串 B 角色；追踪新增会话归属列并按角色持久化
- 取消「轻语」称谓，统称聊天；首页启动器、设置、模型中心、记忆中枢文案全面统一
- 验证：新增 30+ 回归测试；`scripts/ci.ps1` CI OK；`wails build` 成功（build/bin/gaea.exe 38MB）

## v2.1.0「二代完善」(2026-08-07)

> 在 v2.0.1 基础上迭代六轮：模型中心完善、功能级模型启停、Cmd+K 引擎路由、OAuth 回归验证、前端 E 系列核对、小说链路审计。

- 模型中心：引擎状态持久化（`whisper_data/engines.json`）+ 稳定顺序 + 连接状态缓存；修复 `active_engine_id` 只存不读；DeepSeek 脱敏与真实活跃模型展示
- 模型路由：FeatureModelBar 启停改为功能级开关（`func_*_enabled`），不再误关全局引擎；Cmd+K 编辑走 novel 绑定；5 个 agent 未绑定回退按引擎解析默认模型
- OAuth：discovery 配置化（`OIDCDiscoveryURL` 生效）+ 换 token 超时客户端；E04/E13 回归
- 前端：E16/E22/E23/E24 核对 + 静态回归守卫 `scripts/frontend-e-check.mjs` 接入 CI
- 小说：角色注入剥离剧照 base64（`types.PromptView`）；E02/E03 回归
- 验证：go build/vet/test + tsc + vite build + 前端守卫全绿（`scripts/ci.ps1` CI OK）

## v2.0.1「三脑底座」(2026-08-07)

> 在 v1.21.0 基础上迭代：模型路由、三脑记忆、主脑可选编排、基线加固。

- 基线：frontend/package.json 纳入版本控制；scripts/ci.ps1 闸门；E01-E24 评估集 + 首批回归
- 模型：routeModel 降级链（功能绑定→全局→首个可用）；novel/whisper 调用收敛；模型中心"当前生效"
- 记忆：BrainStore 三脑统一访问 + brain_links 跨脑关联；记忆中枢三脑检索
- 编排：ModuleRegistry + RunModule + MainBrainChat（可选入口，不经由模块直达路径）
- 互联：方案生成自动注入跨脑记忆；3.0 模块协议文档预留
- 验证：go build/vet/test 全绿 + tsc + vite build + wails build（见 releases/v2.0.1.md）

## v1.21.0「UI 工作台」(2026-08-05)

> 方案编写板块重构为三栏工作台：左侧项目/方案树（检查分数徽章）＋ 中间文档工作区 ＋ 右侧 AI 上下文面板（招标要点/检查摘要/单人复核清单）；去除 Office 页 emoji 标签与 whisper-theme 依赖。

- 后端：Proposal 增加 CheckSummary（failed/warn/total）与 ReviewChecklist（复核项 Done），SchemaV7 落库；CheckAll 自动写入检查摘要并补默认复核清单（废标逐条/工期一致/评分覆盖/暗标合规/规范引用/签字盖章）；toMap/fromMap 透传
- 前端三栏：左栏项目与方案树（点击过滤/选中，方案卡显示模板、检查分数红/橙徽章）；中栏保留流程步骤条 + 6 个 Tab；右栏 AI 上下文 Drawer（招标要点、检查摘要、复核清单勾选保存、来源定位提示）
- 视觉：移除 Office 页全部 emoji 标签与 `whisper-theme.css` 引用，统一走主题令牌
- 验证：新增 1 测试（摘要与清单持久化）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.20.0「工作流编排」(2026-08-05)

> 方案编写板块引入四阶段流水线（招标解读→投标生成→投标检查→排版导出）与阶段闸门；一键流水线；办公 agent（Hephaestus）新增方案读写/导出工具。

- 阶段模型：Proposal.Stage（parse/generate/check/format，SchemaV6 proposals.stage 列）；解析成功→parse、大纲→generate、全面检查→check、导出→format 自动推进
- 阶段闸门：有招标文件但未解析时，生成大纲/章节/批量生成返回明确提示（空白方案不拦截）
- 一键流水线：RunPipeline 按序执行 解析→大纲→批量生成→全面检查，进度事件 proposal-pipeline-progress
- 办公 agent 打通：proposal 全局服务单例（测试可注入）+ 内置工具 proposal_list / proposal_write / proposal_export，Hephaestus 可直接读写方案需求/章节并导出
- 规范索引抽离共享包 specdata（builtin 工具与 proposal 模块共用，解除导入环）
- 前端：顶部 5 步步骤条（解析/大纲/编制/检查/导出）+ 一键流水线按钮 + 下一步引导（按阶段跳 Tab）
- 验证：新增 6 测试（闸门/阶段推进/流水线/办公工具）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.19.0「排版导出」(2026-08-05)

> 方案编写板块 docx 导出升级为可配置排版引擎：封面/目录/标题层级/页眉页脚页码/Markdown 表格与代码块；按章节导出；暗标规则库（内置土壤修复通用规则，导出自动清理）。

- docx 排版引擎：A4 页边距、页眉（方案标题）、页脚页码（PAGE 字段）、封面页、静态目录、章节三级标题（22/16/14 磅）、正文 12 磅宋体；Markdown 块解析（段落/标题/列表/表格→docx 表格/代码块等宽字体）
- 按章节导出：ExportSectionDocx 单章（含子章节）渲染为独立 docx；按章节 MD 已有
- 暗标规则库：office.db SchemaV5 dark_rules 表；内置“土壤修复通用暗标规则”（无加粗/斜体/下划线/彩色/emoji/特殊符号/压缩空行）；规则可增删改；导出选项选择规则后自动清理内容且标题不加粗
- 绑定：ProposalExportDocxWithOptions（封面/目录/暗标规则）、ProposalExportSectionDocx、ProposalDarkRulesList/Save/Delete
- 前端：导出设置 Modal（封面/目录开关 + 暗标规则选择）、单章导出（章节下拉）、暗标规则库管理 Modal（列表/编辑/选项 Checkbox/删除）
- 验证：新增 8 测试（渲染封面目录表格、单章导出隔离、暗标 seed/清理/CRUD、绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.18.0「校验引擎」(2026-08-05)

> 方案编写板块检查能力升级：可插拔 CheckRule 引擎 + 结构化规则（废标响应/数据一致性/跨章重复/暗标格式/规范引用）+ AI 语义覆盖规则，统一检查报告前端逐项处理。

- 校验引擎框架：CheckRule 接口 + ruleFunc 适配器 + RunChecks 运行器（规则错误转 error item）
- 结构化规则（确定性，无需 LLM）：废标条款响应（未明确回应 warn）、项目事实一致性（未体现 warn/工期冲突 fail/暗标单位名 fail）、跨章重复（20 字 n-gram 交并比，>50% fail / >30% warn）、暗标格式（加粗/删除线/emoji）、规范引用（无引用 warn/编号不在知识库 warn）
- AI 语义覆盖规则：对照招标评分标准 full/partial/none 检查（LLM）
- CheckAll 聚合全部规则输出统一检查报告；既有 CheckCoverage 内部复用并保持绑定兼容
- 绑定 ProposalCheckAll；前端导出 Tab「全面检查」报告面板（规则/状态着色/证据/章节定位跳转）
- 验证：新增 15+ 测试（框架/规则/聚合/绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

### v1.18.0 补充「转换诊断与扫描件 OCR」(2026-08-05)

> 修复桌面实测问题：招标解析失败根因多为第一步转换未成功且不可见。新增扫描件 OCR、转换结果阅览、AI 工作台。

- 根因修复：convertPdfToMD 无页面文字时不再返回“仅页眉”的伪成功（此前扫描件被误判已转换）→ 返回明确错误并触发 OCR；ParseBidFile 全文件转换失败时错误信息包含每个文件名与原因
- 扫描件 OCR：OCRProvider 可插拔接口 + Python 管线（PyMuPDF 渲染 300dpi 页面 → rapidocr_onnxruntime 识别，中文支持）；自动检测，无 OCR 引擎时给出安装指引；FileDoc 增加 error/ocrStatus 字段，OCR 结果标记“OCR 转换”
- 转换结果阅览：文件列表状态细化（已转换/OCR 转换/转换失败+原因 tooltip/待转换）+「查看」抽屉预览转换后 Markdown 或失败原因
- AI 工作台：转换与 AI 分析全过程事件（proposal-ai-progress：start/parse-file/parse-request/parse-reply/done/error），前端阶段动画（Spin+阶段文案）与右侧日志面板（自动滚动）；AI 分析按钮在“任一文件转换成功”时可用（不再被失败文件卡死）
- 验证：新增 6 测试（OCR 调用/OCR 不可用报错/全文件失败明确错误/进度事件）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.17.0「记忆中枢知识资产」(2026-08-05)

> 方案编写板块知识资产统一集成 gaea 记忆中枢：规范条文/素材/历史方案全部入库 Hephaestus.db knowledge 表，spec_query 改读知识库，方案可归档回写并提供同类型参考；模板库落库。

- 规范知识入库：内置土壤修复规范索引（GB 36600/15618、HJ 25.x、HJ 682、HJ 1185 等 15+ 条文）与「土壤修复常用技术」幂等写入记忆中枢（Category=规范标准/经验总结），SearchSpecs 按查询令牌重排（标题 3 分/正文 1 分）
- 素材库：业绩/人员/设备/常用段落以 Category=素材库 入库（AddAsset/ListAssets/SearchAssets/RemoveAsset），名称纳秒级唯一
- 历史方案：ArchiveProposal 将装配全文归档（Category=设计方案 + tag legacy-proposal），SearchLegacyProposals 检索；SectionContext 自动注入同类型历史方案参考摘要（≤600 字，注明不得抄袭）
- spec_query 改读记忆中枢：优先检索知识库规范条文（Top 5），知识库不可用时回退内置索引
- 模板库落库：DefaultTemplates 幂等 seed 到 office.db templates 表，ListTemplates 优先读库（失败回退默认）
- 前端：导出 Tab「归档到记忆中枢」按钮；右上角「素材库」弹窗（搜索/新增/删除）
- 验证：新增 15+ 测试（规范入库/检索重排、素材 CRUD、归档与参考注入、模板 seed、spec_query 知识库优先、绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.16.0「大纲与撰写引擎」(2026-08-05)

> 方案编写板块长篇生成落地：目录策略 + 字数预算分解（以招标文件要求为准）+ 大纲重排/目录导入 + 项目事实基线 + 统一章节上下文 + 批量生成队列（断点续写/合并装配）+ 工艺流程图/组织架构图预设。

- 大纲策略与字数预算：三种目录策略（严格按评标办法/严格按格式要求/参考两者）；解析层提取招标 totalWords，预算按招标要求分配（未要求兜底 10 万、用户可改），递归分配到章→节→小节（叶子合计严格等于总数）
- 大纲重排与目录导入：同级上移/下移自动重编号；Markdown 标题（#/##/###）解析导入替换大纲
- 项目事实基线：office.db SchemaV3 project_facts（工期/业主/修复目标/人员等跨方案共享），SchemaV4 sections 增加 word_target/words 列
- 章节上下文 v2：统一 SectionContext（大纲/评分/废标/格式/暗标/项目事实/字数目标/前章摘要/后节锚点），单章生成与流式生成共用
- 批量生成与合并装配：RunBatch 按叶子单元顺序生成、跳过已完成（断点续写）、失败继续、进度回调（proposal-batch-progress）；Assemble 自动编号（第N章/N.M/N.M.K）并用于导出
- 前端：大纲 Tab 策略/总字数/导入目录/上移下移/字数目标；文本编制 Tab 批量生成/停止/进度条；方案列表右上角项目事实编辑；图表 Tab 工艺流程图/组织架构图/横道图
- 验证：新增 20+ 测试（预算分配/策略提示词/重排/导入/事实往返/上下文注入/批量/装配）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.15.0「招标解析管线」(2026-08-05)

> 方案编写板块招标解析升级：结构化字段提取（概况/工期/资质/评分/废标/格式/暗标）+ 原文来源定位 + parse_results 落库 + 前端结构化卡片与原文抽屉。OCR 仅定义可插拔接口，文字型 PDF 优先。

- 文档分页文本提取（doctext.go）：PDF 按页返回真实页码，DOCX/TXT 单页（Page=0），供来源定位使用
- office.db SchemaV2：新增 parse_results 表（字段/页码/Markdown 偏移/摘录/置信度），Store 提供 SaveParseResults/ListParseResults；AddFile 返回文件 ID
- 来源定位器（locate.go）：AI 摘录 quote 先精确匹配，失败后做忽略空白匹配，映射回原文偏移与页码
- 招标解析管线 v2（parse.go）：逐文件/分块 LLM 提取字段+原文摘录 → 后端确定性定位 → 结果写入 parse_results 与 BidSummary（qualification/format/darkRules/redLineItems/parseStatus，旧字段全兼容）
- 绑定序列化：btm/bsf 改为 JSON 往返，新字段自动透传前端
- 前端：招标解析 Tab 由 JSON 文本框改为结构化字段卡（可编辑保存）+ 来源 chip + 原文预览抽屉（滚动高亮摘录）
- 验证：新增 20+ 测试（分页提取/定位器/解析结果 CRUD/解析管线/序列化往返）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.14.0「方案数据底座」(2026-08-05)

> 方案编写板块数据底座重建：JSON 文件存储迁移 SQLite（office.db），引入项目（标段）层级、版本快照与旧数据无损迁移；方案列表按项目分组。

- office.db 网关（internal/office/db）：SchemaV1 六表（projects/proposals/sections/files/versions/templates）+ schema_meta 迁移链，纯 Go SQLite 驱动，与主脑库同模式
- proposal.Store 迁移 SQLite：项目 CRUD、方案/章节树持久化（含子章节递归归一化）、版本快照（每次更新 +1）、级联删除、附件登记
- Service 接线：启动自动建库 + 确保「未归档项目」+ 旧 JSON 无损迁移（幂等、只读不删原文件）；导出/上传目录迁至 office/exports、office/files；应用退出关闭 office.db
- 后端绑定：新增 ProposalProjectList/Create/Delete，ProposalCreate 支持 projectId；方案/章节数据带 projectId/version/sources
- 前端项目化：方案列表按项目分组（含未分组）、新建方案可选已有项目或新建项目
- 验证：新增 30+ 测试（网关/Store CRUD/树形持久化/版本/迁移/绑定）；go vet clean + go test ./... 全绿 + tsc 0 错误 + vite build + wails build 成功

## v1.13.0「记忆检索升级」(2026-08-04)

> 轻语记忆检索体系升级：检索双轨收敛为单轨（buildTierBBlock）+ FTS 全文检索接线（触发词之外的摘要词召回）+ 语音链路测试补齐（voice/tts）。
> tag v1.13.0

- 记忆检索双轨收敛为单轨：删除零调用孤岛 PrepareTurnContext → MemoryRetriever.Retrieve（重复实现整套检索但从未被消费），统一到 orchestrator.buildTierBBlock 单一主流程；精简 memory_retrieve/types，删除 RelevanceHint/RetrievalResult 类型与 7 个 TestRetrieve_* 死路径测试（净删 353 行）
- FTS5 全文索引修复（此前建而不用）：V2 外部内容表列名与主表不匹配（fact_id vs id）导致 rebuild 必失败 → V11 迁移独立表；rebuild 改为显式全量同步（修复 MaxOpenConns(1) 下 rows 未关时 Exec 死锁）；SearchFactIDsFTS 修复 MATCH 成功但空结果不降级 bug
- 中文全文检索：LIKE 降级升级为 2-gram 多模式（整句 + 相邻两字）——用户说「咖啡」能命中摘要「她喜欢喝美式咖啡」，解决触发词之外摘要词无法召回的问题
- FTS 全文检索接入 TierB 记忆上下文：Orchestrator 新增 FTSSearch 回调（app 层注入 repos 实现，避免循环依赖）；buildTierBBlock 把 FTS 命中事实补入候选（×1.3 加权）；persist 写回后自动重建索引（RebuildFactsFTS/RebuildEpisodesFTS 从零调用变为活跃）
- 语音链路测试补齐：voice 包 0% → 54.2%（31 测试，情绪→TTS 映射/配置校验/VAD 状态机/打断检测，核心状态机 100% 覆盖）；tts 包 0% → 20.9%（26 测试，分句/SSML 转义/RFC6455 握手向量/引擎回退链）
- 验证：新增 60+ 测试（FTS 重建/中文降级/2-gram 模式/引擎回退/语音状态机）；go vet clean + go build ./... 全过 + go test ./... 60 包全绿 + frontend tsc 0 错误

## v1.12.0「轻语记忆贯通」(2026-08-03)

> 轻语记忆系统与 hermes.db 全链路贯通（事实/情节/知识图谱三表持久化）+ TierB 记忆上下文补全（情节/触发词/关联扩散/记忆回声）+ 记忆中枢展示（情节 Tab + 三元组入图）。
> tag v1.12.0，构建 37,743,616 字节。

- 轻语记忆贯通 hermes.db（右脑落地的关键缺口）：Orchestrator 新增 EpisodicStore 运行时实例（原 handler 硬编码 nil，情节从未生成）；restoreWhisperState 从 hermes.db 恢复事实库/情节库/图谱（重启不丢记忆）；persistWhisperState 写回——事实合并写回（本会话以内存为准含退役态，保留其他会话事实），情节/图谱全量写回
- 修复 restoreWhisperState 早期返回缺陷：companion_state/chat_history 无行时提前 return，阻断记忆恢复（首次使用或清空历史后记忆永不加载）
- 修复数据竞争：KnowledgeGraph/EpisodicStore 无锁——extractTriples/情节生成在异步 goroutine 写 + PreLLMTurn 主流程读，Go map 并发写读会 fatal / slice append 实测丢数据 → 加 sync.RWMutex
- 修复 KnowledgeGraph.Query 评分 bug：原遍历全局 entityIdx 给所有三元组加分（图谱越大误命中越多）→ 改为匹配三元组自身 subject/object；单字实体可命中
- TierB 记忆上下文补全：情节检索（EpisodicStore.Search → 【相关记忆片段】）+ 触发词命中事实 boost 1.5x + 关联扩散（Top5 事实经 AssocIndex → 【关联记忆】）
- 记忆回声接线：buildTierBBlock 用检索事实 EmotionalContext 聚合记忆回声（Aff/Sec/Aro/Dom），PreLLMTurn 叠加到状态情绪（ApplyMemoryEcho/ComputeMemoryEchoFacts 从零调用孤岛变为生效，clamp ±100）
- 记忆中枢展示：轻语库新增「事实/情节」Tab（情节时间线流：情绪 emoji 角标 + 强度渐变条 + 关键词 chips + 轮次范围，按 AI 伴侣记忆库调研）；记忆图谱新增轻语三元组（实体节点 t: 复用 whisper 色 + predicate 关系边，共享实体去重）
- FactStore 新增 Restore（保留原 ID/退役态）/ListAll；KnowledgeGraph 新增 Restore/ListAll；影响面：记忆中枢轻语库/总览/归档/图谱首次读到真实数据
- 验证：新增 12+ 测试（FactStore Restore/ListAll、KG 并发/Restore/Query、EpisodicStore 并发、TierB 情节注入/记忆回声/关联扩散、集成往返、图谱三元组）；go test ./... 全绿 + go vet clean + go build 全过 + tsc 0 错误 + vite build 成功

## v1.11.0「界面体验深化 · 全站重设计」(2026-08-02)

> 设置中心外观细化升华（实时预览/三态显示/字体/密度/动效/强调色）+ 聊天 Markdown 消息体验 + 轻语面板 UI 重设计（角色状态头/气泡/情绪回复）+ 虚拟助手面板与角色卡详情重设计 + 轻语测试深化（21.8%）+ P3 archiveExporter 记忆归档导出。
> tag v1.11.0，构建 37,676,544 字节。

- 轻语测试深化：新增 33 个测试覆盖 memory_ingest 管线（LLM 抽取/自动退役/三元组/情节生成/隐私透传）、association_cold_start（文本重叠批量建边/孤儿链接/边去重）、paced_stream（气泡分隔/流排空）、memory 检索路径（触发词 boost/隐私过滤/budget 截断/情节检索/关联扩散/记忆回声）；覆盖率 16.8% → 21.8%
- 修复 paced_stream 真实 bug：流结束发完末段不 finishBubble → MarkDone 等待循环死锁（中文无断点文本必现）；FirstDisplayUnitLen 返回 rune 索引但调用方按字节切片 → 中文句号处切乱码（现转字节偏移）
- P3 archiveExporter：记忆归档导出（对齐 ackem archiveExporter）——README 索引（事实/核心/情节统计 + 领域分布表）+ 每个领域/子类一个 Markdown 文件；退役事实不入档；whisper.WriteArchive 写盘；绑定 GaeaWhisperExportArchive（hermes.db 数据源）+ GaeaPickDirectory 目录选择；记忆中枢轻语库「导出归档」按钮
- 模型中心：LLM 主卡片补三态状态徽章（● 运行中 / ○ 就绪 / ○ 已停止），对齐行业「模型状态三态」基准
- 设置中心：全局搜索（tab 关键词索引 + 实时过滤 + 自动切换到匹配分组 + 匹配计数）+ 即时生效统一徽章（SettingsSection instant prop，5 面板统一）
- 外观设置细化升华：外观实时预览区（主题+模式组合微缩界面，hover 主题卡即时联动「👆 预览中」）+ 主题色系大预览卡（氛围渐变 + 霓虹光晕 + 选中发光对勾）+ 显示模式三卡（暗色/亮色/跟随系统，matchMedia 实时派生，darkMode 保持 boolean 兼容现有消费者零改动）
- 外观设置扩展非颜色维度：字体设置（5 预置字体 + 字号 12-20 带预览）、界面密度（标准/紧凑，ConfigProvider token + .ui-compact CSS）、动效强度（完整/减弱，.ui-reduced-motion 全局禁用动画对齐 macOS 减弱动态）、强调色自定义（取色器覆盖 --gaea-glow/primary 令牌链，留空跟随主题）
- 聊天板块消息体验升级：Markdown 渲染（ChatMarkdown：代码块深色底+复制按钮/表格/列表，完成态渲染）+ 消息分组（同角色连续紧凑 + 组首头像）+ hover 操作栏（复制/朗读/重新生成显隐）+ 重新生成 + 建议卡点击即发送 + 头部元信息（你/gaea AI）+ 错误态红色气泡 + 玻璃内高光
- 轻语面板 UI 重设计：角色状态头（头像 + 关系阶段徽章 + L2 情绪 emoji 徽章 + 信任霓虹进度条 + 对话轮数）+ 消息气泡化（AI 琥珀玻璃气泡 / 用户粉紫渐变）+ 等待期 typing dots 修复（原空白光标）+ 快捷情绪回复 chips + 虚拟助手管理中心按钮卡片化（暖色玻璃入口卡）
- 虚拟助手面板 + 角色卡详情重设计：列表卡视觉区 TisorRadar → CompanionAvatar 粒子光球 + 性格标签 chips + 迷你五维条；详情卡大视觉区粒子球 + 五维区「左 TisorRadar 大雷达 + 右条形列表」并排
- 验证：go vet clean + go build ./... 全过 + go test whisper/app 全绿 + frontend tsc 0 错误（全流程每步验证）

## v1.10.0「科幻记忆中枢 · 架构归拢」(2026-08-02)

> 记忆中枢科幻感首页（中央 3D 图谱 + 霓虹玻璃卡片）+ 3D 图谱白屏修复（three-forcegraph → 3d-force-graph）+ AI 控制台归小说专用 + 小说专属代码归拢 components/novel/。
> tag v1.10.0，构建 37,617,152 字节。

- 记忆中枢科幻首页：中央 3D 图谱 + 四周霓虹玻璃模块卡片（hub.css 玻璃拟态 + 扫描线 + stagger），点击切库面板
- 3D 图谱白屏修复：误装底层库 three-forcegraph（class 需 new）→ 换 3d-force-graph（Kapsule 可调用）；图谱渲染时序修复（数据到达即构图）
- AI 控制台：默认关闭 + 记忆中枢隐藏（面板/按钮双层排除）+ 从 MainLayout 抽出为 components/novel/AIConsole.tsx 仅小说页挂载
- 小说代码归拢：16 组件 + hooks/api 全部移入 components/novel/（git 识别 18 rename），MainLayout 删 ~380 行小说控制台代码
- 界面配色跟随系统主题：hub.css 硬编码深色 → gaea 令牌（--bg/--fg/--accent），图谱背景/label 走令牌
- 验证：go test 57 包 0 失败（后端零改动）+ tsc -b + vite + wails build 全绿

「记忆中枢」(2026-08-02)

> 记忆体系三脑架构落地（命名 Hephaestus/Hermes + 主脑 Hephaestus.db + 左脑办公 SQLite + 右脑 hermes.db + 调度路由 + 知识库 RAG）+ 记忆中枢板块（七库统一管理）+ 3D 记忆图谱 + 成本库。
> tag v1.9.0，构建 36,552,704 字节。

- 命名体系：办公 agent → Hephaestus（火神），轻语 agent → Hermes（信使），gaea 之子女；AIgaea 产品类型统一
- 三脑架构：新建 Hephaestus.db（facts/profile/knowledge 三表 + 迁移链）；memory.Store 后端抽象（文件/SQLite 双后端，调用方零改动）
- 左脑接通：办公记忆 Markdown → Hephaestus.db 幂等迁移 + memory_get 工具 + boot/controller 默认 SQLite
- 右脑更名：whisper.db → hermes.db（首次打开自动迁移，保留备份）
- 调度路由：remember type=user → 主脑画像（profile 跨 agent 共享）+ DetectConflicts 冲突检测
- 知识库 RAG：迁入 Hephaestus.db + 共享向量层 internal/gaea/search（TF-IDF + 中文 bigram 余弦）混合排序
- 记忆中枢板块：知识库入口升级为多库面板（知识/成本/画像/办公三 tab/轻语只读/图谱），12+ 新绑定
- 3D 记忆图谱：three-forcegraph（type 着色/按库过滤/hover/点击详情），后端预计算 nodes+links
- 成本库：cost_entries 表（schema V2）+ cost 包 + CostLibrary（基础版）
- 画像冲突一键裁决（以画像为准 / 以 facts 为准）+ 轻语详情弹窗
- 验证：go test 全量 57 包 0 失败 + tsc + vite + wails build 全绿

「单模型架构 · 知识库板块」(2026-08-02)

> 办公板块删除双模型架构（Hermes/Hephaestus → 单模型）+ 知识库整合为独立板块（统一服务层 + 全文检索）+ AI 聊天双会话面板修复。
> tag v1.8.0，构建 36,084,224 字节。

- 办公板块：删除 Hermes/Hephaestus 双模型 agent（hermes.go 645 行 + 测试），runner 直接用 executor 单 Agent；
  config 删 planner_model/temperature/effort；前端删 Planner 配置/统计列/RunStatus 简化
- 单模型工作流梳理：系统提示词分层（DefaultSystemPrompt=领域知识 / SingleModelPrompt=执行纪律），
  boot 拼接保执行纪律不丢；删除 PlanCard 计划确认死链路（AskQuestion.Plan 字段全链清理）
- 知识库独立板块：knowledge.Service 进程级单例（工具/UI 统一走 ~/.gaea/knowledge），
  新增 GaeaKnowledgeSearch 全文检索（含正文）；KnowledgePage 板块页 + 导航五处注册；
  与记忆系统明确区分（显式知识 vs 隐式事实）
- AI 聊天：删除 ChatTopicSidebar 重复渲染（双会话面板修复）
- 验证：go test 全量 0 失败 + tsc + vite + wails build 全绿

## v1.7.0「设置中心重构 · 模型统一」(2026-08-02)

> 设置中心全面重构（小说/方案/办公/轻语/更新信息）+ 模型绑定统一左下角卡片 + 代码审查修复 9 项并发与绑定 bug。
> tag v1.7.0，构建 36,140,032 字节。

- 审查修复：轻语引擎 per-call 覆盖（删全局切换竞态）、小说 9 处绑定模型生效、ASR 校验、config.Save 加锁、
  manager 副本防 race、轮询进程合并、删 VoiceSetChatTarget 死绑定、finalTranscript 修复
- ComfyUI：findPython 加 standalone-env 兜底（ROCm）+ windows-standalone-build 标志 + 工作流结构测试
- 绑定模型卡：5 板块统一左下角浮动卡片（三态 + 启停），聊天页补渲染；预警只统计本地模型
- 模型中心：功能模型绑定独立标签页（2 列卡片网格）
- 设置中心 6 标签：小说（目录 C:\AI\xiaoshuo）/ 方案（新）/ 办公（完整可编辑）/ 轻语（设置合并）/
  系统（更新信息 + 删迁移）/ 语音
- 轻语界面移除设置弹窗，左栏 240 防卡片遮挡；删空态多余标题
- 验证：go test 全量 + tsc + vite + wails build 全绿

## v1.6.4「设置瘦身」(2026-08-02)

> 设置页删除与模型中心重复的「模型引擎」配置 tab，引擎/模型管理统一收敛到模型中心。
> tag v1.6.4，构建 36,106,240 字节。

- 移除设置页 EnginePanel tab（import/图标/副标题文案同步清理）
- 设置页保留：外观 / 工作空间 / 语音 / 绘梦 / 办公 / 系统
- 验证：go build + tsc -b + vite + wails build 全绿

## v1.6.3「剧照模型可选」(2026-08-02)

> 补丁：小说/轻语角色剧照生成可选手模型（含 ComfyUI 本地 krea2/z-image-turbo/flux）。
> tag v1.6.3，构建 36,110,848 字节。

- 后端 GetImageBackendConfig：availableModels 恒含 ComfyUI 本地模型（无论全局后端），
  小说角色剧照弹窗即可选本地模型
- 轻语角色详情：生成剧照按钮旁加「出图模型」Select（自动加载可用模型，默认当前全局模型），
  handleGeneratePortrait 用所选模型
- 验证：go build + tsc -b + vite + wails build 全绿

## v1.6.2「方案模型条 + 底栏常驻」(2026-08-02)

> 补丁：方案编写模型条不可见修复 + 底栏常驻（无项目时也显示模型监控与资源）。
> tag v1.6.2，构建 36,109,824 字节。

- OfficePage 顶部插入 FeatureModelBar（feature="office"）
- MainLayout 底栏 Footer 去掉 projectOpen 条件 → 常驻显示已启用模型 + CPU/内存/GPU
- 验证：go build + tsc -b + vite + wails build 全绿

## v1.6.1「小说统一模型」(2026-08-02)

> 补丁：小说板块章节/角色/世界观 agent 统一接入 func_novel，整部小说用一个 LLM 模型（v1.6.0 已知限制 1 修复）。
> tag v1.6.1，构建 36,108,288 字节。

- worldview / character / chapter / analysis / outline 5 个 agent 全部加 featureModel + chat（带 func_novel 引擎覆盖），
  替换全部 `ChatSimpleStream(a.cfg.Model)` 调用 → `chat(ctx, ...)` + `ChatSimpleStreamWithOptions(EngineID: FuncNovelEngine)`
- 运行中切换小说模型即时生效（各 agent 动态读 cfg.FuncNovelEngine/Model）
- 验证：go test ./... 53 包全绿 + tsc -b + vite + wails build

## v1.6.0「语音交互 · 角色中心 · 功能模型」(2026-08-02)

> 语言交互全面落地：首页语音对话 + 核心 AI 助手 gaea（大地女神）+ 角色中心（助手即角色）+ 各功能板块独立模型绑定与资源监控。
> 23 提交（v1.5.0 后），tag v1.6.0，构建 36,108,288 字节。详见 releases/v1.6.0.md

### 首页语言交互（全新）
- 深空虚空首页：400px 大语言粒子球（连线网络+双环绕轨道+音量驱动）悬浮，8 透明玻璃卡片浮游，虚空微尘背景
- 「进入语音对话」本页直启麦克风：轻语 voiceManager 管道 → gaea 对话 → 识别/回复气泡 + TTS 朗读
- 布局响应式（粒子球随视口缩放）+ 语音卡片边框透明

### 核心 AI 助手 gaea = 大地女神盖亚
- 新增人格预设 gaea（首位全局默认）：大地之母，温厚沉稳包容滋养（五维 T85/I55/S20/O80/R50，goddess 标签）
- 启动确保 gaea 始终存在（修复旧数据缺失）；唯一「AI 助手」，其余标「角色」

### 角色中心（助手即角色）
- 管理中心改铺满主界面角色中心（角色卡网格 + 详情弹窗）；AI 角色剧照；小说↔轻语互传（参数统一）
- gaea 排第一 > 当前对话 > 启用 > 禁用

### 功能级模型绑定
- ai.Client per-call 引擎覆盖；config 10 键（chat/whisper/novel/office/gaea 引擎+模型）持久化
- 接入：聊天/轻语（含语音）/小说大纲；模型中心「功能模型绑定」UI
- 各窗口 FeatureModelBar（三态状态+一键启停）；底栏资源监控（CPU/内存/GPU+已启用模型）；超载弹窗警告

### 模型中心完善
- 恢复 ComfyUI 引擎（启停/状态/本机路径写死）；ComfyUI 图片模型（krea2/z-image-turbo）入墙
- 语音模型 STT/TTS 三段选择 + 持久化

### 已知限制（范围控制）
- 小说 agent 绑定仅大纲；办公 agent 绑定仅 UI；语音端到端依赖本地 herdsman 未 GUI 自动化实测

## v1.5.0「微信接通」(2026-08-02)

> 微信 ClawBot 通道全链路打通（微信发消息 → AI 自动回复），办公板块 v1.4.0 回归修复。
> 11 提交（v1.4.0 后），tag v1.5.0，构建 36,011,520 字节。
> 详见 releases/v1.5.0.md

### 微信 ClawBot 通道接通（核心）
- 认证修复：`Authorization: Bearer <token>` + `iLink-App-Id` + `iLink-App-ClientVersion`（数字编码 132099）
  + getUpdates 端点全小写 → 消除"会话过期"（errcode -14）
- 消息字段对齐**腾讯官方 openclaw-weixin**：`item_list[].type`（type=1 文本）+ client_id `{prefix}:{ts}-{hex}`
  （此前对齐社区 Rust SDK 误用 `item_type`，消息被静默丢弃 → 微信无回复的最终根因）
- 扫码绑定流程补全：need_verifycode 配对码二次查询（新绑定 WhisperWeixinQRStatusWithCode）、
  scaned_but_redirect/verify_code_blocked 处理；confirmed 返回完整 token + botId
- 会话过期状态透出：UI 显示"微信会话过期 · 需重新绑定"替代虚假"在线"
- 防脱敏 Token：前端含 `*` 用原值、后端含 `*` 拒绝
- 助手名字注入：BuildAckemCanonBlock 名字参数化 + Orchestrator.AssistantName，微信 AI 自称助手名
  （如"峨嵋"）而非默认"轻语"

### 办公板块修复（v1.4.0 回归）
- 右侧面板 Drawer 内联渲染（getContainer=false）+ 改为布局内 grid 列（340px）——不再跨页面残留/遮挡导航栏/凸出遮挡对话区
- 恢复对话区样式（修复 eb7d5e6 误删 chat-pane/transcript/markdown 样式族，输出框消失）
- 移除办公设置 + 明暗配色按钮（与系统重复，由主应用设置中心接管）

### 验证
- go test ./... 53 包全绿；tsc + vite build；wails build 产物 36,011,520 字节
- 微信全链路用户实测通过（收到消息 → AI 回复 → 发送成功）

## v1.4.0「架构统一」(2026-08-02)

> 后端四类重复收敛 + 前端两套 UI 体系统一。净删 6874 行（+555/-7429），10 提交，tag v1.4.0。
> 详见 releases/v1.4.0.md

### 后端架构重构（-526 行）
- HTTP client 统一：netclient 提升共享层，22 处裸 `http.Client` 收敛到 `NewSimpleClient`
- 工具函数收敛：office 死代码 + asr/whisper 手写字符串函数替换标准库
- 配置系统整合：删除 gaea config 零消费者死域（ComfyUI/Statusline/Notify/Serve/LSP）
- AI 通道唯一化：删除 provider/stream_client.go，通道=前端→agent→bridge→ai.Client→modelengine

### 前端两套 UI 体系统一（-6300+ 行）
- 死代码清理：老栈 29 废弃组件 + gaea 7 死代码 + 4 死 hook/util + highlight.js（-5800 行）
- 令牌层统一：gaea 颜色引用老栈 M3 令牌，删独立主题系统，暗亮双向联动
- 图标体系统一：66 个 lucide → @ant-design/icons，lucide 依赖移除
- 布局壳 antd 化：Layout/Drawer/Tooltip 用 antd，功能面板按分层原则保留自绘
- 死样式清理：.app/workspace-panel 样式族删除（-690 行）

### 验证
- go test 77 包全绿；tsc + vite build 通过；wails build 产物 gaea.exe

## v1.3.0「未来感 UI」(2026-08-02)

> 未来感 AI 多功能助手平台：深空星云 × 玻璃拟态 × 霓虹光效全链路统一 + 设置中心整合全部参数。
> 五批 UI 改造（+718/-268 前端），tag v1.3.0，构建 36.1MB。

### 设计系统
- 12 套主题新增 glow/glassBg/auroraBg 三令牌（App.tsx 注入 CSS 变量）
- 未来感 CSS 层：赛博网格 + 星点 + 漂浮光球背景、玻璃拟态、霓虹描边卡片、发光状态点、流光扫描线

### 框架与界面
- 顶栏玻璃化 + logo 光晕 + 菜单霓虹选中态；AI 中枢首页（真实模型状态 + 霓虹卡片墙）
- 模态框/弹层全局玻璃化；主题色块发光胶囊；页面切换过渡 + 霓虹加载态
- 聊天界面：玻璃气泡 + 渐变用户气泡 + 霓虹输入栏；侧栏玻璃化
- 绘梦面板玻璃化 + 画廊 hover 发光；底栏霓虹状态条

### 设置中心（整合全部设置参数）
- SettingsPage 重写为 7 分组 Tabs（外观/工作空间/模型引擎/语音/绘梦/办公/系统）
- 新增 settings/ 8 组件：主题发光选择器、图片保存目录、推理强度、引擎编辑、语音健康检测与阈值、绘梦后端、办公摘要、v4 迁移
- api/settings.ts 补 5 封装；设置页隐藏 AI 控制台

### 验证
- go test ./... 全绿（54 包）；go build / go vet 通过
- npm run build + wails build 产物 gaea.exe（36.1MB）
## v1.2.0「架构瘦身」(2026-08-01)

> 后端臃肿治理：绑定层按域拆分（197 方法迁移）+ 死代码清理 + 办公板块设置写路径接通
> + staticcheck U1000 全仓库清零 + 原子写实现去重。5 批提交，净删 ~600 行。

### 绑定层拆分（App 聚合 + 嵌入提升）
- App 结构体从 20+ 平铺字段收敛为 core + 4 域 State 嵌入引用（writing/media/whisper/office）
- 197 个方法按域归位；Go 嵌入提升保证 Wails 绑定不变，前端零改动
- 子服务持 app 反向指针协调跨域调用

### 办公板块设置写路径接通（20 个存根）
- 新增配置持久化（~/.config/gaea/config.toml 原子写）+ 改配置 → 保存 → 重建 controller
- Agent 参数 / 权限 / 沙箱 / 模型设置真实生效；Provider 增删改转发模型中心
- 回退 / 更新类改为明确错误语义，前端隐藏回退菜单

### 死代码清理（staticcheck U1000 全仓库清零）
- gaea 引擎：notify 包（2 文件）+ filteredSchemas + incompleteCanonicalTodos + runTurn
- whisper：candidate_collector.go 整文件（179 行孤岛）+ emotion_fusion 7 函数 + 3 常量
- ai：buildFluxWorkflow（已被 Z-Image/Krea 取代）

### 原子写去重
- 重建 fileutil.AtomicWrite，5 处手写重复实现收敛，净删 41 行

### 验证
- go build / go vet / 53 包测试全绿；staticcheck U1000 清零；前端 tsc 通过
- 新增测试：配置持久化往返 + 嵌入提升编译期断言

## v1.1.0「质量工程」(2026-08-01)
## v1.1.0「质量工程」(2026-08-01)

> 稳定性加固 + 架构瘦身 + 测试防线建立：21 处 goroutine recover、SSE 阻塞修复、
> 死代码清理 902 行、四个核心模块测试覆盖大幅提升（modelengine 97.1% / office 35.6% / ai 24% / whisper 16.8%）。

### 稳定性（3 批提交）
- 21 处 goroutine 无 recover 防护：桌面应用 panic 崩溃问题根治，最严重为
  controller.runGuarded turn 执行 panic 导致前端永久"生成中"，修复后复位状态 + Emit TurnDone
- SSE 流式发送 select+ctx.Done 保护：取消后 goroutine + HTTP 连接阻塞泄漏根治（hang 类问题根因）

### 架构整理（5 批提交，净删 902 行）
- gaea 模块：管理命令组 / 模糊编辑移植 / 宪法文件 / 7 处未使用字段别名
- 其他模块：剧照旧实现 / 世界观上下文 / 章节迁移等
- staticcheck U1000 84→25 处（剩余为 whisper 对齐 ackem + ComfyUI 迭代相关）

### 测试防线（5 批提交，+1900 行测试）
- modelengine 0%→97.1%、office/proposal 0%→35.6%、ai 7%→24%、whisper 13.1%→16.8%
- 修复 3 处测试暴露缺陷：GenerateSection 漏设 completed、UpdateSection/RemoveRawFile 静默失败

### 发布
- 构建产物 `C:\AI\wubigrokuildin\gaea.exe`，完整说明见 `releases/v1.1.0.md`

## v1.0.0「品牌重塑 · 盖亚」(2026-08-01)

> wubigrok 正式更名为 **gaea**（盖亚，大地女神）——从「小说创作 Agent」升级为「多功能 AI 助手」。
> 全新品牌视觉：翡翠球体 + 破土嫩芽 + 灵感星芒，favicon / appicon 全量替换。

### 品牌重塑
- 全产品品牌替换：窗口标题「gaea · 多功能 AI 助手」、应用名/产物（gaea.exe）、UI 显示、文档、导出署名、图片下载前缀、日志文件
- Go module 重命名 `github.com/wubigork/wubigork` → `github.com/gaea/gaea`（233 个文件 import 同步）
- 新 logo 三件套：`frontend/public/favicon.svg` + `build/appicon.svg` + `build/appicon.png`（1024x1024）
- 版本号从 v5.x 重新起算为 **V1.0.0**（versioninfo.rc / wails.json / CHANGELOG）

### 数据兼容（老用户零丢失）
- 配置文件 `~/.gaea_config.json`（回退读取 `~/.wubigork_config.json`）
- 登录 token 回退读取 `.wubigork_token.json`（免重新登录）
- 项目标记目录 `.gaea/`（识别旧 `.wubigork/` 项目，v4 检测双向兼容）
- localStorage 键 `gaea_*`（回退读取旧键：聊天记录/人格/主题/绘梦模板保留）
- 内部 provider 注册名 `wubigrok` 保留（bridge provider 引擎兼容），UI 显示名全部为 gaea

### 发布
- 构建产物 `C:\AI\wubigrokuildin\gaea.exe`

## v5.76.0「工程办公」(2026-07-31)

> gaeaW（土壤修复工程办公 AI 助手）完整移植：47 个工程工具 + Hermes/Hephaestus 双模型 agent + 6 个工程技能 + gaeaW 原生 UI，模型统一走 gaea 模型中心。

### 新板块：办公
- 后端移植 `internal/gaea/` 30 包（agent/tool/control/skill/command/plugin/knowledge/memory/boot/config），模型经 `provider/bridge` 接入模型中心（空模型动态跟随引擎切换）
- ai 包扩展工具调用支持（OpenAI 兼容 + SSE tool_calls 分片拼装，向后兼容）
- gaeaW 原生 UI 完整移植至 `frontend/src/gaea/`（App + 70 组件 + Tailwind v4），GaeaPage 渲染 gaeaW App
- 适配层：bridge.ts 90+ 方法名映射（Submit→GaeaSend 等），wubigrok 补齐 80+ Gaea* 绑定方法，事件格式精确对齐 WireEvent
- `.gaea/skills/` 6 个工程技能（场地调查/风险评估/修复设计/投标/数据报告）

### 发布
- 完整说明见 `releases/v5.76.0.md`

## v5.73.1「方案编写完善」(2026-07-31)

### 关键修复
- 嵌套章节（AI 大纲的 2/3 级）后端查找/更新全部改为递归展平，子章节可正常撰写/润色/改图/重命名
- 流式撰写内容真正落盘（原为副本指针，刷新即丢）
- 前端自动保存与图表插入支持子章节（updTree 递归更新）
- Word/MD 导出、覆盖/规范检查包含全部层级章节
- 浏览器上传招标文件改为 base64 落盘后转换（原路径为空必然失败）

### 新能力
- 大纲手工编辑：新增子章节/重命名/删除（含确认与编号重排），三个章节树均可用
- 流式撰写上下文补齐：需求、评分标准、废标条款、完整大纲、前一章节
- 失败时向前端发送 error 事件，不再卡死「生成中」

### 发布
- 完整说明见 `releases/v5.73.1.md`

## v5.20.0「精炼」(2025-07-26)

> 提示词全面重设计 + 死代码大清理 + 编译修复。净删 ~3000 行，prompts 23→15。

### 提示词重设计（4 个核心 prompt）
- **create-chapter**：硬编码字符串 → 模板化，「正在写这本书的作者」
- **chapter-generate**：「出版级」→「作者」，去 AI 味交由技能注入
- **plot-branch-browser**：「剧情策划人」→「story breaker」，固定 3 分支
- **worldview-agent**：「设定顾问」→「设定编辑」，代码块输出替换原文

### 死代码大清理（16 文件删除 + 后端精简）
- **删除 9 个废弃 prompt JSON**：brainstorm-ideas, chapter-review, outline-generate-detail, story-thread-chat, story-thread-generate, worldview-chat-section, worldview-check-consistency, worldview-generate-all, bootstrap-reference-summarize
- **删除 6 个前端组件**：AIAssistSheet, BrainstormModal, DialogueModal, StoryBibleModal, StoryBibleSteps, BeatToProse
- **删除 brainstorm_handler.go**（59 行）
- **Go handler 死代码清理**：chapter_handler (-178), copilot_handler (-237), create_chapter_handler (-82), outline_handler (-108), plot_branch_handler (-80), project_handler (-107), worldview_handler (-42)
- **核心模块精简**：chapter.go (-301), outline.go (-203), worldview.go (-215)
- **前端页面清理**：MainLayout (-60), CreatePage (-46)，其他页面移除死引用

### 编译修复
- chapter_handler.go 缺失函数闭合 `}` 修复
- copilot_handler.go 6 个未使用 import 清理
- chapter_handler.go 未使用 "strings" import 清理

## v5.17.0「重塑」(2025-07-26)

> 从 v5.7.1 分支重建。移除移动端代码，新增「创作」面板，章节节点树 + 分支系统。

### 移除
- 删除 `mobile/` 独立移动端应用
- 删除 `internal/mobile/` Go HTTP 服务
- 删除 MobileTabBar、MobileDrawer、useIsMobile 等全部移动端组件

### 创作面板（全新）
- **三栏布局**：左侧节点树 | 中间编辑器 | 右侧 AI 控制台（全局复用）
- **CreateChapter API**：一步到位，跳过对话和大纲阶段，直接生成正文
- **三分支构思**：AI 读取设定 + 前文摘要 → 生成 3 个剧情方向 → 用户选择 → 生成正文
- **节点树系统**：每章永久节点（ID、摘要、parent_id），支持分支/覆盖/删除/重生成
- **摘要注入**：每章生成后自动提取摘要存入节点树，前文注入改为节点摘要拼接

### AI 控制台增强
- 展开 REQ 查看完整 SYSTEM/USER prompt
- 展开 OK 查看 AI 完整响应（修复 response 事件缺失 content 字段）
- 固定宽度 380px + alignSelf: stretch

### 后端
- `internal/app/create_chapter_handler.go` — CreateChapter 一步生成+保存
- 正文末尾 `---CHAPTER_SUMMARY---` 标记自动分离摘要
- `GenerateOutlineWithDialogue` prompt 限制章节数，避免浪费 tokens

## v5.7.0「凝练」(2026-07-06)

> 7 轮组件化拆分迭代：42 文件变更，+3,244 / -1,946，(window as any) 和 @ts-ignore 全面清零

### ⚛️ 组件化重构（7 轮）

#### R3 — 去重 & 组件化
- **BranchSelectorPanel**: 提取 PlotBranchModal + NextChapterModal 共享分支选择 UI（187 行）
- **StoryBibleSteps**: StoryBibleModal 5 步子组件拆分，主组件 489→288 行
- **净效果**: +701/-448，两 Modal 各减少 ~90 行重复代码

#### R4 — 书架模块
- **提取 4 子组件**: WelcomePage / BrainstormModal / CreateNovelModal / ProjectCardItem
- **HomePage**: 640→280 行 (-56%)，Ctrl+N 快捷键 + 骨架屏
- **Utility**: 提取 formatRelativeTime + delay 到 utils/time.ts
- **类型化**: 新增 BrainstormIdea 接口，消除 any
- **移动端**: 统一 btnBase 样式，移除无效 CSS animation

#### R5 — 世界观模块
- **API 抽象层**: api/worldview.ts，消除 6 处 @ts-ignore
- **SectionNav**: 共享维度导航（桌面固定侧栏 / 移动端下拉面板），消除 ~90 行重复
- **ConsistencyReport / MapFullscreen**: 提取行内子组件
- **WorldviewPage**: 490→328 行 (-33%)

#### R6 — 角色模块
- **API 抽象层**: api/character.ts，消除 11 处 @ts-ignore
- **RelationshipModal / OrganizationEditModal / PortraitLightbox**: 提取行内 Modal
- **OrgField**: 移入 CharacterFormHelpers.tsx
- **renderCharEditor()**: 消除桌面/移动端 ~40 行重复
- **CharacterPage**: 489→396 行 (-19%)

#### R7 — 写作模块
- **Wails 类型导入**: 消除 7 处 (window as any)
- **ReviewResult**: 审稿组件（评分圆环 + 优势/不足/改进建议）
- **OutlinePanel**: 大纲面板独立组件（折叠/展开 + 卷/章树形渲染）
- **超长行拆分**: 最长行 620→314 字符，提取 createTabData / handleFinalize / handleNextChapterGenerate

#### R8 — 绘梦模块
- **API 抽象层**: api/image.ts，消除 12 处 (window as any)
- **ResultGallery 增强**: 添加 onDelete 支持
- **PromptBar / CustomTemplateModal**: 提取底部输入栏和模板编辑弹窗
- **ImageGenPage**: 665→521 行 (-22%)

#### R9 — 设置面板
- **API 抽象层**: api/settings.ts，消除 ~15 处 (window as any) + @ts-ignore
- **SettingsCard**: 统一卡片组件，消除 7 处重复 cardStyle
- **SettingField**: 统一「标签 + Input/Select」模式
- **handleToggleMobile / handleMigrate**: 提取命名函数
- **SettingsPage**: 475→356 行 (-25%)

### 🧹 全局清理
- **(window as any) 清零**: 前端所有页面全部替换为 wailsjs/go/app/App 类型导入
- **@ts-ignore 清零**: 通过 API 抽象层消除所有类型跳过
- **catch(_) 清零**: 全部替换为 catch(err) + console.error
- **mountedRef 移除**: React 18 自动批处理已解决

### 📦 构建
```
wails build -o build/bin/wubigork-v5.7.0.exe
```
- 新增 28 文件，修改 14 文件

## v5.6.0「移动端远程操控」(2026-07-03)

> 移动端完整功能实现：RPC 桥接、SSE 流式、静态文件服务、CORS、QR 码、IP 检测

### 📱 移动端功能
- **HTTP RPC 调度器** (`internal/mobile/rpc.go`) — 泛型反射调用 App 方法，黑名单过滤生命周期方法
- **SSE 流式推送** (`internal/mobile/sse.go`) — StreamHub 事件广播 + EventSource 频道
- **前端桥接层** (`frontend/src/api/bridge.ts`) — 非 Wails 环境自动创建 `window.go.app.App` HTTP 代理
- **Runtime 多填充** (`frontend/src/api/runtimePolyfill.ts`) — `EventsOn/Off/Emit` + SSE EventSource 多填充
- **静态文件服务** — embed.FS 生产模式 + `dist/` 前缀兼容 + SPA fallback
- **CORS 中间件** — 手机浏览器跨域访问支持
- **IP 检测** — 虚拟网卡过滤（Docker/WSL/Hyper-V 等），优先 WiFi 真实 IP
- **端口预检** — `net.Listen` 预检端口可用性，占用时立即报错

### 🐛 修复
- **移动端页面白屏** — `serveStaticOrSPA` 路径前导斜杠导致 embed.FS 拒绝读取
- **事件名不一致** — 前端 `prose-stream` → `beat-prose-stream`
- **render 崩溃** — HomePage `card.title` null 守卫
- **IP 错误** — UDP 拨号法回退到 Docker IP，接口名过滤法替代

### 📦 构建
```
wails build -o build/bin/wubigork-v5.6.0.exe
```
- 新增 4 文件，修改 12 文件

## v5.5.0「精炼」— 全量代码优化 + 工程质量加固 (2026-07-03)

> 两轮迭代：后端去重 + 前端巨型组件拆分集成 + 死代码清理 + 类型安全加固

### 🏗 后端重构（12项）
- **EstimateTokens 去重**: context+memory→util，修复中文检测用 unicode.Is
- **ExtractJSON 去重**: style→util
- **401 重试合并**: ai/client.go 三合一 → refreshAndRetry()
- **config.Save 重构**: 18-case switch→map 注册模式
- **marker 解析抽象**: ParseMarkedSections()，worldview/character 共用
- **for i:=1;;i++→ForEachChapter**: 8处死循环统一，缺失章节 continue 而非 break
- **chapter.Generate 拆分**: 200行→3函数(buildGenerateContext/streamAndRetry/postProcess)
- **ChatStream 拆分**: 请求构建+SSE 解析分离
- **mobile 死代码清理**: 删除 SPAHandler，修复 _ = path
- **app.go 类型安全**: comfyUICmd 删除，distFS interface{}→fs.FS
- **skill YAML 加固**: Windows \r\n 兼容，修正 scan() 注释
- **slog.Warn 统一**: 35+处→Agent.warnRead()

### ⚛️ 前端重构（11项）
- **ChapterPage**: 1170→227 行 (-80%)，集成 ChapterEditor/AIAssistSheet/useChapterStream
- **OutlinePage**: 1188→350 行 (-70%)，集成 OutlinePanel/ThreadPanel/DialogueModal/useOutlines
- **StoryBibleModal**: 488行→5步子组件+路由壳
- **PlotBranchModal+NextChapterModal**: 共享 usePlotBranch hook
- **4 新 hooks**: useChapterStream/useOutlines/useWailsEvent/usePlotBranch
- **errorHandler**: handleError+wrapAsync 统一错误处理
- **OutlineNode 类型冲突修复**: api/outlines.ts 删除重复定义
- **StepCreate.tsx**: onChange 死代码修复
- **HomePage Creating**: mount guard 防 unmount 后 setState
- **LorebookModal/SkillModal**: AbortController cleanup
- **TTSPlayer**: onStatusChange ref 冻结

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.5.0.exe -ldflags="-s -w" .
wails build -o build/bin/wubigork-v5.5.0.exe
```
- 新增 16 文件，修改 ~25 文件，删除 ~900 行
- 零新依赖

## v5.4.0「锻造」— 稳定性修复 + 本地生图增强 + 功能打磨 (2026-07-03)

> 修复 13 项 Bug，新增 ComfyUI 一键启停、系统状态监控、角色 Agent、图片管理增强

### 🐛 Bug 修复（13项）
- **xAI 默认模型名**: `"flux"` → `"grok-imagine-image-quality"`，修复云端生图失败
- **xAI API size 参数**: API 不再接受 `size`，请求中清除该字段
- **ComfyUI 编码崩溃**: Python 3.13 + GBK 控制台无法编码 emoji → 补丁 logger.py + prestartup_script.py
- **Login() 后 ComfyUI 丢失**: 重新登录后自动恢复后端配置
- **小说切换数据不同步**: 5 页面 (世界观/角色/大纲/章节/画布) 监听 projectPath 重新加载
- **Token 刷新无超时**: `RefreshAccessToken` 用 `http.DefaultClient` → 15s 超时，防止永久卡死
- **生成失败消息**: 返回具体原因而非笼统的「所有生成尝试均失败」
- **ComfyUI 执行错误**: 错误解析索引 msgArr[2]→msgArr[1]，正确提取异常信息
- **右侧面板隐藏**: 绘梦右侧栏改为始终显示（文件夹按钮在无历史时也可见）
- **CMD 弹窗**: `getCPUUsage` 调用 wmic 时隐藏窗口
- **防重复提交**: 绘梦生成按钮加 useRef 锁

### ✨ 新功能
- **ComfyUI 一键启停**: 绘梦顶部 🟢/⚫ 状态 + 启动/停止按钮，设置页配置安装路径
- **系统状态监控**: 绘梦右侧栏显示 CPU% + GPU 显存占用，3s 刷新
- **角色 Agent**: 角色页底部 ChatPanel，AI 对话自动创建/编辑角色
- **图片单张删除**: 画廊和历史缩略图支持逐张删除
- **图片文件夹快捷打开**: 右侧栏 📁小说图片 / 🖼生成图片 目录
- **世界观地图双击全屏**: AI 地图和 3D 势力图支持双击放大
- **图片自动保存**: 未配置专用目录时自动存到 `<小说>/images/`

### 🎨 UI/UX
- 导航 `项目` → `书架`，其余文案 `项目` → `小说`
- 大纲页卷结构面板拉长填满窗口
- 清理冗余文件回收 ~79MB

### 📦 构建
- `build/bin/wubigork-v5.4.0.exe` — 16MB

---
## v5.3.0「暗夜」— 全局配色重设计 (2026-07-03)

> 暗夜系列 6 套主题 — 手工调色，表面色与强调色分离，完全替换 M3 Tonal Palette

### 🎨 主题系统重做
- **6 套暗夜主题**: 暗夜青（默认）/ 暗夜紫 / 暗夜玫 / 暗夜金 / 暗夜苔 / 暗夜墨
- **表面色与强调色分离**: 不再从单一 seed 派生所有颜色，每套有独立的表面色温
- **手工调色**: 替换 M3 `generateTonalPalette` 自动生成，消除同色系单调问题
- **亮色模式**: 6 套暗夜主题各自对应一套亮色变体
- **零破坏**: 旧 localStorage 中的主题名自动回退到默认暗夜青

### 🔧 累积修复 (v5.2.1 → v5.3.0)
- Z-Image sampler 节点引用 14→13
- 大纲 prompt 矛盾约束（恰好5卷 vs 保留不修改）
- Lightbox z-index 1000→1050（剧照全屏被遮挡）
- AI 绘梦模板系统：类别下拉 + 自定义 + 去重风格

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.3.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **3 文件变更**: +70/-188 行
- **零新依赖**

---
## v5.2.0「凝形」— Bug修复+工程质量+性能优化 (2026-07-03)

> 基于 v5.1.0 审计，修复 24 项问题：10 Bug修复 + 8 工程质量 + 6 性能与体验

### 🐛 Bug 修复（10项）
- **Z-Image 尺寸参数**: width/height 正确映射到 Z-Image ratio（1:1/3:2/4:3/16:9/2:1），不再被忽略
- **Context cancel 泄漏**: `copilot.go` `_ = cancel` → `defer cancel()`，防止 goroutine 泄漏
- **io.ReadAll 错误忽略**: ComfyUI `queuePrompt`/`checkHistory` 两处错误现在正确返回
- **os.MkdirAll 错误忽略**: `export.go` 目录创建失败不再静默继续
- **NovelsDir 硬编码**: `D:\AI\xiaoshuo` → `~/wubigork-novels`，跨系统兼容
- **CancelGhost 空实现**: 真正取消进行中的 Ghost 补全 goroutine，前端收到 `cancelled` 事件
- **事件监听器清理**: `EventsOn('xai-output')` 添加 useEffect cleanup 防止内存泄漏
- **静默吞异常**: 全站 `catch (_) {}` → 带标签的 `console.error`
- **loadOutlines 重复**: ChapterPage/OutlinePage 重复实现 → 提取到 `api/outlines.ts`

### 🏗 工程质量（8项）
- **API 抽象层**: 新建 `frontend/src/api/outlines.ts`，封装后端调用
- **Wails 类型声明**: 新建 `frontend/src/types/wails.d.ts`，`AppAPI` 接口声明 100+ 方法，消除 `@ts-ignore`
- **TTS 配置持久化**: `config.go` 新增 `tts_port`/`tts_backend`/`tts_speed` 的 Load/Save
- **Save() 常量化**: 定义 19 个 `Key*` 常量替代硬编码字符串，防止拼写错误
- **SafeGo 工具**: `util/util.go` 新增 `SafeGo(fn)` — goroutine + panic recover + 日志
- **requirePM() 辅助**: `app.go` 新增读锁获取项目的方法，消除 handler 重复的 nil 检查
- **废弃文件清理**: 删除 `internal/ai/context.go`

### ⚡ 性能与体验
- **移动端 API 对接**: `mobile/handlers.go` `ProjectsProvider` 回调，替换硬编码 JSON 占位
- **统一日志**: 新建 `frontend/src/utils/logger.ts`，四级日志（debug/info/warn/error），生产环境可关闭
- **页面保活**: `MainLayout` `visitedPages` Set 机制，切 tab 不再销毁组件丢失状态
- **大纲五阶段固定卷**: `ContinueOutline(5)` 全量替换（起承转合终），简化 AI 生成流程
- **Z-Image NSFW LoRA**: 工作流新增 `LoraLoaderModelOnly` 节点 (strength 0.7)
- **世界地图图片**: `SaveWorldMapImage`/`GetWorldMapImage` Wails API，base64 存取

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.2.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **27 文件变更**: +1095/-346 行
- **新增 3 文件**: api/outlines.ts / types/wails.d.ts / utils/logger.ts
- **删除 1 文件**: internal/ai/context.go
- **零新依赖**

---
## v5.1.0「绘梦师」— Z-Image-Turbo 极速生图 + AI 绘梦工作台 + 角色剧照打通 (2026-07-03)

> 本地 Z-Image-Turbo 8 步生图（48s）、AI 绘梦双栏工作台重做、角色剧照与 AI 绘梦打通、角色详卡三段式布局

### ⚡ Z-Image-Turbo 本地生图集成
- **新模型支持**: ComfyUI-ZImagePowerNodes + GGUF Q5_K_M (~5.2GB) + Qwen3-4B Q4_K_M (~2.5GB) text encoder
- **8 步极速**: ZSamplerTurbo2Simple，48 秒出图（Flux 需 60-90 秒 20 步）
- **双模型切换**: Flux Dev / Z-Image-Turbo，前/后端完整支持，配置持久化
- **VRAM 优化**: UNet partial offload ~4GB 加载 + 1.4GB 卸载，8GB 显存刚好够用
- **v5.0.0 补全**: 下载脚本 + 工作流测试验证 + 节点参数自动探测

### 🎨 AI 绘梦工作台重做
- **双栏布局**: 左栏 320px 控制面板 / 右栏自适应结果区，桌面端专业工作台体验
- **负向 Prompt**: 可折叠输入区，Flux 工作流节点 8 注入
- **种子控制**: InputNumber + 🎲 随机，精准复现同一张图
- **批量生成**: 一次 1-4 张，循环提交 + 每张独立计时
- **20 个预设模板**: 创作/写实/风格/构图 4 大类，点击自动填入正/负向 prompt
- **全屏灯箱**: 键盘导航（← → Esc），显示种子/模型/尺寸/耗时，一键下载/重用参数
- **会话历史**: 底部横向缩略图画廊，点击回溯，支持清空
- **进度预估**: 按钮下方显示「预计 ~90s」/「上次 48s」，根据模型和数量动态计算
- **6 个新组件**: PromptPanel / GenControls / GenButton / ResultGallery / Lightbox / HistoryStrip
- **移动端适配**: 单栏控制面板 + 底部 Drawer 结果抽屉

### 👤 角色剧照与 AI 绘梦打通
- **剧照后端统一**: GeneratePortrait 改用 `cfg.ImageModel`（支持本地 Z-Image-Turbo/Flux），中文 prompt
- **AI 绘梦→角色剧照**: Lightbox 新增「设为剧照」下拉选择器，选角色自动保存到 `portraits/<id>.png` 并写回 `characters.json`
- **角色卡片缩略图**: 卡片列表圆形头像显示 `portrait_url`（有图时），无图时回退图标
- **SetCharacterPortrait API**: Go `character.Agent.SetPortrait()` + Wails 绑定

### 📋 角色详卡三段式重做
- **概览区**: 剧照 160×160 左侧主视觉 + 姓名/类型/状态/性格标签 + 操作按钮
- **Tab 分组**: 「📋 档案」（基本信息+性格+外貌+身材+动机+弧光+背景）/「🔗 关系」（组织+人物关系+出场章节）
- **剧照三态**: 有图悬浮重生成 / 生成中 Spin / 无图虚线框 CTA
- **删除按钮**: 移至右上角，减小误触

### 🔧 修复
- **config.go**: `comfyui_url` Save case 空白无赋值（严重bug）→ 正确赋值 `ComfyUIURL`
- **config.go**: `image_save_dir` Save case 错误写入了 `ComfyUIURL` → 修正为写入 `ImageSaveDir`
- **config.go**: `ImageModel` 配置字段新增 + Load/Save 支持

### 📦 构建
```bash
go build -o build/bin/wubigork-v5.1.0.exe -ldflags="-s -w" .  # 9.9MB
```
- **新增 7 文件**: 6 个前端组件 + 模板数据
- **修改 9 文件**: Go 后端 5 + 前端 4
- **零新 npm 依赖**

---

## v5.0.0「织梦者·移动」— 移动端远程操控 + M3 设计语言 (2026-07-02)

> 手机远程操控桌面端、Material Design 3 设计迁移、ComfyUI LoRA 生图链、20 commits

### 🎨 Material Design 3 全站迁移
- **M3 Tonal Palette**: 轻量内联实现，5 套主题从种子色自动生成 13 级色调调色板
- **Ant Design Token 驱动**: ConfigProvider 模拟 M3 视觉（surface/surfaceContainer/elevation/outline）
- **CSS 变量标准化**: `--md-sys-color-*` / `--md-sys-elevation-*` M3 命名体系
- **Layered Glass → M3 Surface**: `.glass-panel` → `.md-surface-container`，移除全站 backdrop-filter
- **M3 交互**: CSS-only ripple 涟漪效果、`.md-card` elevation hover lift、触控 44px 最小区域
- **25 变量兼容填充**: 旧 CSS 变量名 → M3 token 映射，零破坏迁移

### 📱 移动端远程操控
- **HTTP 服务**: Go `internal/mobile/` 包 — LAN IP 检测 + 二维码生成 + SPA fallback
- **响应式双导航**: 桌面端 Sidebar / 移动端 Bottom TabBar + Drawer + AppBar
- **Container Query 布局**: `useMediaQuery` hook + 3 级断点（compact/medium/expanded）
- **通用移动组件**: MobileSheet（底部滑出面板）、LongPressable（长按 500ms + 震动反馈）
- **全页面适配**: 9 个页面 + 5 个组件 — 3D 关系图→2D SVG 降级、Canvas→Grid、编辑器→全屏+Sheet
- **设置页**: Switch 开关 + 二维码面板，手机扫码即连

### 🎨 ComfyUI 集成增强
- **内嵌 Web UI**: ImageGenPage 一键切换 iframe 控制台，无需另开浏览器
- **LoRA 链**: Flux.1 Dev → Realism (0.8) → NSFW (0.6)，LoraLoaderModelOnly 级联
- **URL 持久化**: `GetConfig('comfyui_url')` 自动加载

### 🔧 工程质量
- **零新依赖**: 前端无新 npm 包，Go 纯 `net/http` 标准库
- **Wails 绑定**: 3 个新方法 `StartMobileServer` / `StopMobileServer` / `GetMobileServerStatus`
- **20 commits**: 每个 Task 独立审查（Spec ✅ + Quality Approved），2 个 fix round
- **前后端双编译**: `tsc -b && vite build` + `go build` 零错误

### 📦 构建
```bash
wails build                              # 生产包 (build/bin/wubigork.exe, 16MB)
go build -o build/bin/wubigork-v5.0.0.exe .  # 开发包 (不含嵌入 dist)
```

---

## v3.1.0 — Layered Glass 视觉重设计 (2026-06-21)

### 🎨 UI 全面升级 — "Layered Glass"
- **设计系统**: 17 个新 CSS 设计令牌 (accent-rgb, shadow-sm/md/lg/glow, border-subtle, bg-deep/base/elevated/glass, radius-sm/md/lg/xl, transition-fast/normal/slow)，4 套主题统一渐变+辉光
- **玻璃态全站**: 所有面板/卡片 `backdrop-filter: blur(8px)` + 半透明背景 + 柔和边框替代 thin 1px 实线
- **导航重设计**: 顶栏 sticky glass + 菜单去下划线/pill 选中态 + 底栏玻璃态
- **XAI 控制台**: 从等宽字体调试面板 → 圆角玻璃浮层 + 系统字体 + 彩色左边线
- **首页**: 项目卡片 glass-card 悬停上浮 + 空状态品牌水印 + 按钮辉光
- **写作页**: 玻璃侧边栏 + pill 标签页 + 编辑器内凹 inset 阴影 + 工具栏统一玻璃按钮
- **对话框**: Ant Design Modal 全局玻璃覆盖 + 弹簧入场动画 + 遮罩 blur
- **骨架屏**: 三处 loading 从裸 Spin 升级为 Ant Design Skeleton 骨架屏
- **无障碍**: `prefers-reduced-motion` 全局尊重

### ✨ 功能增强
- **TTS 语音朗读**: VoxCPM 本地神经网络合成 + Edge TTS 在线自然语音 + Windows SAPI 零延迟
- **写作页标签页优化**: 可横向滚动 + 关闭未保存确认弹窗
- **设置页**: 工作空间目录可配置

### 🔧 代码质量
- **DRY 清理**: 消除 9 处 `C()` 重复定义 → 统一 `import { C } from utils/theme`；消除 RelationGraph 颜色映射重复；提取 `sortNodes` 到 `utils/outline.ts`
- **错误处理**: 消除 15+ 处 `json.Marshal`/`io.ReadAll` 静默吞错误
- **死代码清理**: 删除 `internal/xai/` + `pkg/novel/` 重复函数 ~350 行

### 🛠 修复
- Edge TTS WebSocket 重写 + 引擎链降级 (SAPI → Edge → VoxCPM)
- VoxCPM 编译为静态链接，消除 DLL 依赖
- exec.Command 隐藏控制台窗口，消除朗读时弹 cmd
- NovelsDir 默认值不再依赖配置文件

---

## v3.0.0 — 革命性跃升 (2026-06-20)

### 🚀 核心创新
- **剧情分支选择器**: AI 推理 3-5 个下一章方向，用户选用或手工录入，自动写入大纲并同步角色/世界观
- **全书发展编辑**: AI 审读全书生成 1500 字编辑信 + 5 维评分 + 角色弧光诊断，对标 $3500 人工编辑
- **自我演化引擎**: 每章生成后自动分析 → 建议 Lorebook 词条 → 追加世界观空维度 → 记录伏笔变化
- **跨模块协作**: 大纲角色点击跳转角色页 + 角色出场章节查询 + 世界观一致性一键编辑

### 🎯 v2.0.0 功能 (2026-06-20)
- Story Bible 引导式创建（5 步向导：灵感 → 一键生成 → 角色优化 → 大纲优化 → 开写）
- 情节画布（水平时间线可视化全文章节 + 角色色条 + 品质情绪标注）
- Skill 管理面板（浏览/导入/创建自定义写作风格 Markdown）

### 💡 v1.5.0 功能 (2026-06-20)
- Lorebook 词条系统（定义概念 → AI 写作时自动注入相关上下文）
- 写作统计仪表盘（每章字数柱状图 + 品质趋势进度条）
- AI 审稿（5 维评分环形图 + 改进建议高亮卡片）
- Brainstorm 脑暴面板（输入题材 → AI 生成 6 个核心点子 → 选用创建）

### 🔧 v1.4.0 功能 (2026-06-20)
- 右键 AI 操作（选中正文 → 丰富描写/扩展场景/重写此段）
- 多章节标签页（同时编辑多个章节，独立状态）
- 大纲拖拽排序（Ant Design Tree draggable + 批量保存）
- 专注写作模式（全屏沉浸，隐藏侧栏和 Agent）

### 🛠 v1.3.2 修复 (2026-06-20)
- 大素材归纳压缩（先 AI 归纳关键信息再注入 prompt，防止上下文溢出）

---

**技术栈**: Go + Wails v2 + React 19 + Ant Design 6 + Three.js + d3-force-3d  
**构建**: `go build -o build/bin/wubigork-v4.0.0.exe .`  
**许可证**: MIT
