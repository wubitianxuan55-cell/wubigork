# gaea 瘦身 P2 刀2 补验审计（2026-09-09）

> **状态=审计证据（2026-09-09）**——**G-2 已关闭（v4.169.0）**：`gschedSummary.ts` 增 `baselineCount`（project.baselines 长度），
> ScheduleFileCard 增「基线 N」StatChip（summary.baselineCount>0 时）——「基线数=N」计数口径落地，测试 +5（gschedSummary 3 + ScheduleFileCard 2）。
> G-3 真机走查仍挂真机池（.gsched kind:text 前置：v4.169 摘要卡「基线 N」chip 一并真机核）。
> 审计时点仓库状态：HEAD=v4.167.0（P2 刀0 已随版落档）；工作树含刀1a/刀1b 未提交改动（见 §6 与缺口清单 G-1/G-4）。

## 0. 审计口径

- 权威目标（masterplan 轨道一·1 刀2）：`.gsched` 注册进办公文件面（最近文档可见；FilePreview 摘要卡：任务数/总工期/关键路径/基线数+「打开工作台」）；`.mpp` 走既有「导入为新工程」不做内嵌预览。
- 初步发现（任务简报）：刀2 主体可能已提前在产——ScheduleFileCard.tsx（头注「v4.121.0 刀12 办公联动」）即「FilePreview 摘要卡+打开工作台」，v4.139 加多工程切换（GaeaScheduleProjectOpen 切指针），FilePreview.tsx:619-623 对 .gsched.json 分发到该卡。
- 逐项实证结论：**刀2 三要素 2.5/3 已提前完成，剩余缺口以「验收口径」为主（基线数计数、后端 inMenu 同步），不构成功能缺失**。详见下。

## 1. 刀2 三要素实证表

| # | 要素 | 现状（实证） | 完成度 | 证据 file:line |
|---|---|---|---|---|
| a1 | FilePreview 摘要卡·分发 | 办公预览 .gsched.json 拦截进 ScheduleFileCard（`preview.kind==="text"` 分支内做 `isScheduleFilePath` 判定，解析失败回落原始文本视图） | ✅ | frontend/src/gaea/components/FilePreview.tsx:618-627（:621 isScheduleFilePath + parseSchedSummary、:623 <ScheduleFileCard>、:625 回落 pre、:619-620 注释 v4.121 刀12） |
| a2 | 摘要卡·任务数/总工期/关键路径 | StatChip 四枚常驻：总工期 duration（schedCard.duration）、任务数 taskCount、关键路径 criticalCount（crit 红）、搭接 linkCount；里程碑条件显式；开竣工区间行 | ✅ | frontend/src/gaea/components/ScheduleFileCard.tsx:111-122（:112 duration、:113 taskCount、:114 criticalCount、:115 linkCount、:116-118 milestone） |
| a3 | 摘要卡·基线数 | 显示**活跃基线快照**（名称/保存时间/基线工期→当前/漂移 crit 或 good）；多基线列表（v4.137 #11，max 3 槽）的**条数不显示** → 近似达成，计数口径缺失列缺口 G-2 | ⚠️ | ScheduleFileCard.tsx:126-140；frontend/src/schedule/gschedSummary.ts:42,72-74（baseline 单槽）；frontend/src/schedule/baseline.ts:130-145（upsertBaseline 列表）；frontend/src/schedule/store.ts:322-341 |
| a4 | 摘要卡·「打开工作台」 | 按钮「在进度计划中打开」：当前计划直接 NAVIGATE schedule（jumpToSchedule）；非当前计划先 GaeaScheduleProjectOpen 切指针 + notifyScheduleFileChanged 再跳（v4.139 #15 多工程） | ✅ | ScheduleFileCard.tsx:161-168（按钮）、:74-86（openThisPlan）、:66-68（jumpToSchedule=NAVIGATE {page:'schedule'}）、:46-48（isCurrentPlan）、:79（ScheduleProjectOpen）；测试 frontend/src/gaea/components/ScheduleFileCard.test.tsx:76-80（navigate 事件） |
| a5 | 解析引擎（摘要数据） | parseSchedSummary：normalizeProject + computeCpm → ok/duration/taskCount(level>0)/criticalCount/linkCount/milestoneCount/finishDate/baseline(漂移)/deadline(倒排)；坏 JSON/非 Project 形状返回 null | ✅ | frontend/src/schedule/gschedSummary.ts:48-77（:21-26 路径识别 isScheduleFilePath 大小写不敏感双分隔符）；测试 gschedSummary.test.ts:22-31 |
| b1 | 最近文档可见·目录树 | 办公文件面（files tab=ExplorerView）目录树**无扩展名过滤**：只滤点文件，.gsched.json 在树中可见 | ✅ | frontend/src/gaea/components/FileTree.tsx:388（仅 `.startsWith(".")` 过滤）、:234-237（ListDir 拉取）；sidebarRegistry.ts:92-99（files 视图渲染 ExplorerView） |
| b2 | 最近文档可见·最近栏 | 最近文件单源 lib/recentFiles：办公面打开/引用文件即 recordRecentFile 置顶，RecentFilesBar chip 展示（ExplorerView 顶部挂载）；.gsched 无特殊豁免 | ✅ | frontend/src/gaea/components/RecentFilesBar.tsx:34-40、ExplorerView.tsx:63；frontend/src/gaea/App.tsx:416-417（openFilePreview 即 recordRecentFile）；ExplorerView.tsx:34、useComposerMenus.ts:105（@ 引用同源）；recentFiles.ts:13-42 |
| b3 | 最近文档可见·注意 | schedule 板块**自身不写最近文件**（schedule/ 下无 recordRecentFile 调用）；「最近」语义=办公侧打开/引用过的文件，计划板块内操作不自动入列 | ✅（语义如此） | grep 全仓：recordRecentFile 调用点仅在 gaea/ 侧（App.tsx:416、ExplorerView.tsx:34、RecentFilesBar.tsx:38、useComposerMenus.ts:105） |
| c1 | .mpp 走导入不做内嵌预览 | SchedulePage「导入为新工程」（v4.144 主入口）按扩展名分发：xml/xlsx/xlsm/**mpp** → importScheduleMpp（Go MPP9/12/14 二进制解析）→ importAsProject 落独立 .gsched.json 入索引；不支持格式显式拒绝 | ✅ | frontend/src/pages/SchedulePage.tsx:573-612（:592-595 mpp 分支、:597 拒绝、:601 importAsProject）；frontend/src/schedule/api.ts:162-163；frontend/src/schedule/store.ts:697-702（importAsProject v4.144，slug 去重+切指针，非破坏）；测试 SchedulePage.test.tsx:266-323 + store.sync.test.ts:225-305 |
| c2 | MPP 导入菜单/入口 | 菜单「文件→导入为新工程… / 导入 MPP 替换当前工程…」+ pickImport（壳内走 GaeaPickFiles 系统对话框，v4.162）+ 文件 input accept=".mpp" | ✅ | SchedulePage.tsx:641-646（importMenuItems）、:620-636（pickImport）、:657、:863、:874 |
| c3 | MPP 无内嵌预览 | 办公预览管线无 MPP 分支：FilePreview 仅对 .gsched.json 有卡；Go 侧 Preview/office 无 mpp 处理（mpp 只经 GaeaScheduleImportMpp 导入绑定） | ✅ | internal/schedule/mpp.go:1-3（v4.140.0 刀1 MPP9/12/14）；internal/app/gaea_schedule_mpp.go:18-27；internal/app/bindings_office.go:135；FilePreview.tsx:599-627（markdown/text 分支无 mpp 特判） |

**三要素完成度小结**：摘要卡+打开工作台=✅（v4.121.0 刀12 提前在产，v4.139 增强多工程）；最近文档可见=✅（办公文件面既有设计零改动即满足，.gsched 打开/引用即入最近 + 树中常驻可见）；.mpp 走导入=✅（v4.140 二进制导入 + v4.144 导入为新工程，无内嵌预览）。唯一 ⚠️=「基线数」按活跃基线快照近似达成，多基线条数未上卡（G-2）。

## 2. 命令面板与板块入口现状（任务 1d/1e）

| 项 | 现状（实证） | 结论 | 证据 |
|---|---|---|---|
| d 命令面板·进度计划项 | **HEAD v4.167.0 无**：办公命令面板组=新建会话/记忆/历史/知识库/面板命令，无 schedule；壳层 SearchModal（Ctrl+K）是搜索+意图预览（S4.6），非命令清单、无 schedule 项。**工作树中刀1b 正在添加**（未提交 diff 含 cmd-schedule「进度计划」→ NAVIGATE schedule + 新测试 App.palette.test.ts） | ❌（HEAD）/ 进行中（刀1b）→ 由刀1b 提供，刀2 不背缺口 | frontend/src/gaea/App.tsx:1324-1374（paletteItems）；frontend/src/components/SearchModal.tsx:120-130；工作树 `git diff HEAD -- frontend/src/gaea/App.tsx`（cmd-schedule，注释 v4.168 刀1b）+ `?? frontend/src/gaea/App.palette.test.ts` |
| e1 顶栏菜单 | **前端静态**：HEAD v4.167.0 schedule inMenu:true；工作树（刀1a 未提交）manifests.ts:142 已 **inMenu:false**（注释「瘦身刀1（v4.168.0）」）。**生效侧=后端**：internal/app/board/builtins.go:160 schedule 仍 `InMenu: Bool(true)`，normalizeManifests 后端字段优先（manifests.ts:266）→ **真机顶栏菜单仍显示 schedule**；settings 先例 builtins.go:128 为 false | ⚠️ 前后端分裂（G-1） | manifests.ts:138-144（工作树 diff inMenu:true→false）；builtins.go:157-163（:160 InMenu:true）；manifests.ts:266（inMenu 后端优先）；getActiveMenuBoards manifests.ts:302-306 |
| e2 首页启动器 | deriveLauncherModules 过滤 `!isHome && (inMenu || settings)`：schedule 是否入启动器**跟随生效 inMenu**——真机（后端 true）=含；浏览器/mock（静态 fallback）=刀1a 落地后不含；LAUNCHER_DESC 仍有 schedule 文案（无害残留） | ✅ 现状=真机含；刀1a 后=启动器卡片消失（预期后果，办公入口承接） | launcher.ts:47-62（:53 filter）；launcher.ts:35（LAUNCHER_DESC.schedule）；ModuleLauncher.tsx:294（deriveLauncherModules(activeBoards,…)） |

## 3. 缺口清单（若有）

| # | 缺口 | 证据 | 处置建议（供主代理裁决） |
|---|---|---|---|
| G-1 | **后端 InMenu 未翻 → 前后端分裂**：前端 manifests.ts 工作树已 inMenu:false（刀1a 未提交），后端 builtins.go:160 仍 Bool(true)；生效清单以后端为准 → 真机顶栏菜单/启动器仍显示 schedule，浏览器 dev（静态 fallback）不显示 | builtins.go:157-163 vs manifests.ts:138-144；manifests.ts:266 | 刀1a 收口时同步翻后端 builtins.go（与 settings 先例对齐 builtins.go:128），否则真机/浏览器行为不一致（P2 红线：壳内真机走查） |
| G-2 | **「基线数」计数口径缺失**：摘要卡只显示活跃基线快照+漂移，baselines 列表条数（v4.137 #11 多基线 max3）不上卡 | ScheduleFileCard.tsx:126-140；gschedSummary.ts:42,72-74（baseline 单槽）；baseline.ts:130-145 | **已关闭（v4.169.0）**：gschedSummary 增 baselineCount（project.baselines?.length）+ ScheduleFileCard 「基线 N」StatChip；测试 +5 |
| G-3 | .gsched 分发位于 kind==="text" 分支内（FilePreview.tsx:618），要求 Go Preview 对 .gsched.json 返回 text kind；未见 Go 侧 .json→text 映射证据（低风险，JSON 惯例 text） | FilePreview.tsx:618-627 | 壳内真机走查时点开 .gsched.json 验证一次即可（P2 走查项） |
| G-4 | 命令面板进度计划项：HEAD 无（刀1b 进行中，见 §2-d） | App.tsx paletteItems | 刀1b 收口后复验；非刀2 缺口 |
| G-5 | 补验测试缺口（只读审计不写）：FilePreview.test.tsx 无 .gsched 分发断言（覆盖仅 ScheduleFileCard.test.tsx + gschedSummary.test.ts） | 见 §4 证据注 | 刀2 收口时可补一个分发用例（纳入等价验收表） |

## 4. 主要实证证据注

- 版本史（git）：v4.121.0 `4bc90d7b`（刀12 办公联动三件套=摘要卡+互跳+AI 周报）；v4.125.0 `9765d900`（摘要卡设为当前计划）；v4.139.0 `a4777bf0`（多工程 #15：每文件一工程+索引+切换器+agent 跟随）；v4.150.0 `528478ca`（双工期）；v4.167.0 `49855a66`（瘦身 P2 刀0 白名单解耦，刀1/刀2 依赖语义就绪）。
- 办公文件面装配：GaeaPage=gaea/App.tsx 壳（pages/GaeaPage.tsx:13-23）；右侧工作台 files tab=ExplorerView（sidebarRegistry.ts:89-99，RENDERERS.files=createElement(ExplorerView,…)）；ExplorerView=RecentFilesBar+FileTree（ExplorerView.tsx:48-78）；文件 tab 复用 FilePreview embedded（WorkspacePane.tsx:105-114）。
- 摘要卡测试：ScheduleFileCard.test.tsx:65-80（chips 与 navigate 事件）；gschedSummary.test.ts:22-31（isScheduleFilePath 双分隔符/大小写）。
- MPP 引擎零改动：MPP 解析属 schedule 引擎既有能力（internal/schedule/mpp.go + gaea_schedule_mpp.go v4.140.0），刀2 只承接入口，未触引擎。

## 5. 观察池核验：刀3 工作台全幅内嵌办公（评估）

**结论：现状不可直接嵌入，需中等改造；数据层桥梁已完成，维持观察池（masterplan §6/§1 刀3）。**

| 维度 | 评估 | 证据 |
|---|---|---|
| 页面形态 | SchedulePage 是完整自持页：顶层 sched-shell（左 ScheduleChatPane + 拖宽分隔条 + 主区），自带菜单栏行1（文件/编辑/视图/任务 4 下拉 + 工程切换 Select + 工程管理 + 办公互跳 + AI 栏开关）、工具栏行2（撤销/重做/增删/日历/基线/资源 + 视图 Segmented）、视图主体（横道/单代号/双代号/资源使用）、底部状态栏；manifest layout:'padded' 全幅板块 | SchedulePage.tsx:732-1046（:733 sched-shell、:734-746 chat pane+divider、:751-804 菜单栏、:808-975 工具栏、:965-974 Segmented、:991-994 视图、:1011-1044 状态栏）；manifests.ts:139；builtins.go:159 |
| 已共享面（利好） | 数据层已与办公同源：useScheduleStore 被 ScheduleFileCard 直接消费（ScheduleFileCard.tsx:46）；notifyScheduleFileChanged 事件办公/计划共用（ScheduleFileCard.tsx:81，gaea/App.tsx import 同源）；反向互跳已通：计划页「在办公板块中预览计划文件」→ gaea 页（SchedulePage.tsx:451-454 openInOffice） | 见左列 |
| 未通面 | 面板注册表 RENDERERS（sidebarRegistry.ts:89-99）无 schedule 渲染器；SchedulePage 无 embedded/布局变体（无 prop 可切换壳形态）；入口固定为整页 NAVIGATE（ScheduleFileCard.tsx:66-68），非面板内嵌；keepAlive/高度语义未定义 | sidebarRegistry.ts:89-99；ScheduleFileCard.tsx:66-68；SchedulePage.tsx:732-1046 |
| 改造量 | 中等：把 SchedulePage 抽壳为可嵌入组件（sched-shell/菜单栏按 embedded 变体收敛）+ sidebarRegistry 增 renderer + 面板宿主（WorkspacePane 文件 tab 式）协调 store/keepAlive；或 iframe 内嵌（但 schedule 引擎零改动铁律下 iframe 会引入新宿主复杂度） | — |
| 维持条款 | 刀1/刀2 用一周后再拍板（masterplan 原文）；本审计不改变观察池状态，仅记录「不可嵌/需大改」为基线评估，等 P2 主干临近时复审 | masterplan §6、§1 刀3 |

## 6. 与 v4.121/v4.139 的历史关系注记（刀2 主体提前在产的来龙去脉）

1. **v4.121.0（刀12 办公联动）——刀2「摘要卡+打开工作台」的主体来源**：当时以功能刀 12 形态落地办公联动三件套（计划摘要卡/板块互跳/AI 周报），即 ScheduleFileCard.tsx 头注「v4.121.0 刀12 办公联动」；摘要卡要素（任务数/总工期/关键路径/基线+漂移）与该版本同源成型。masterplan 2026-09-08 立项时把该能力按「刀2」口径重新挂账——即刀2 非新写，是既有能力入瘦身验收口径。
2. **v4.139.0（#15 多工程）——摘要卡升级为任意计划可开**：GaeaScheduleProjectOpen 切指针语义替代原「复制内容为当前计划」（复制会造成双份同源漂移），摘要卡 isCurrentPlan 按 rel 与当前指针全等判定，非当前计划点击先切指针+通知重水合再跳转。刀2「打开工作台」在多工程下语义完备。
3. **v4.140.0 / v4.144（MPP）——「.mpp 走导入不做内嵌预览」已具备**：v4.140 提供 MPP9/12/14 二进制解析绑定（GaeaScheduleImportMpp）；v4.144 确立「导入为新工程」安全默认主入口（扩展名分发 + importAsProject 独立文件落盘 + 切指针，非破坏），.mpp 全程在导入通道内。
4. **v4.167.0（P2 刀0）——刀1/刀2 的导航依赖语义就绪**：deriveNavigateWhitelist 从 `inMenu&&!isHome` 解耦为 `!isHome`（注册即可导航、inMenu 只控菜单），刀1 schedule inMenu:false 不会静默丢导航；v4.164 前科根治。
5. **审计时点（2026-09-09）工作树未提交**：刀1a（manifests.ts schedule inMenu:false + manifests.test.ts）与刀1b（App.tsx cmd-schedule 命令面板项 + App.palette.test.ts）并行进行中，与刀2 重叠面（命令面板入口）由刀1b 承接；后端 builtins.go:160 尚未翻（G-1）。
6. **总评**：刀2 三要素中 a（摘要卡）与 c（.mpp 导入）为提前完成（v4.121/v4.139/v4.140/v4.144），b（最近文档可见）为办公文件面既有设计零改动即满足；刀2 的正式落地动作≈验收口径挂账 + 补验（G-2 基线数计数、G-3 真机预览、G-5 分发测试）+ 与刀1 收口对齐（G-1 后端 inMenu）。无需新增产品代码即可声明刀2 达成（除 G-2 若按计数口径裁定）。

## 7. 审计边界声明

- 只读审计：未修改任何产品代码；本文件为唯一新增产物；未触碰 manifests.ts/launcher.ts/MainLayout/App.tsx（并行子代理刀1 足迹互斥，仅读取取证）。
- 未执行壳内真机走查（本审计线为代码证据层）；G-3 留待 P2 走查。
- 证据行号基于审计时点工作树；刀1a/刀1b 提交后行号可能漂移（责任文件已标注）。