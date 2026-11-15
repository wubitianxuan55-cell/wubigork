# 图文板块工具梳理·动刀计划（2026-11-15）

> 事实源：调研 `docs/gaea-wenxue-tools-survey-2026-11.md`；本文件=计划与进度台账。
> 拍板（2026-11-15）：案A 图文独立引擎；撤 xlsx/pptx 预览+CrossEmbed、卸 browser_×12、任务模板换小说（导出面保留）；首批专属工具=章节/统计/角色/时间线四族；未批挂池=质检族、文风族。
> 硬红线：**办公板块行为零变化**——办公引擎装配、提示词、技能索引、绑定行为逐字节不变；办公后台任务不受图文任何操作影响。

## 目标

图文板块（wenxue）成为长篇小说创作专属板块：会话引擎独立（Cwd=`C:\AI\tuwen` 当前小说目录）、工具集为小说创作子集+专属四族、技能面为小说域子集、前端办公残余清零、任务模板/欢迎页小说化。

## 刀序（依赖：0/0b 可先行；1 是 2/3 的地基）

### 刀0：前端死面清理（零行为风险，先行）

- 删 WenxueSchedule×8 全链：Go `bindings_wenxue.go` 8 方法 + `wenxue_workspace.go` Schedule fork 段（含仅此段使用的辅助）+ 前端 mappings 8 条 + ScheduleExportXlsx/ImportXlsx/ImportMpp→Gaea* 3 条 + bindingNames/bindingSignatures（真伪两份）+ `mock/office/schedule.ts` + build.ts 组装行 + spaceBindings 条目。
- 删死组件（逐一实证零 importer 后删）：FilePreviewModal（连带 WenxueConvertToPdf 死口绑定链）、DagPanel、TaskInboxPanel、MorningBriefCard、CostLibraryView+cost_library_modals+cost_library_parts、memoryhub/CostInquiryPanel+CostBackupCard、memoryhub 资料库死视图族、知识库残余（types/knowledge.ts、mock 段、domainColors 行、GraphView 文案）。
- 能力面板过滤缺口收口：capabilities/ToolsSection、SkillsSection 按 `disabledDomains.ts` 过滤。
- 验证：`tsc -b`、wenxue 定向 vitest、drift、Go wenxue 测试。

### 刀0b：预览收缩（拍板项）

- FilePreview 撤 PptxOutline/PptxEditPanel/XlsxPreview（含 xlsxpreview 四子件 FormulaBar/MiniChart/SheetGrid/numFmt）与 CrossEmbed 调用、OfficeLiveBadge；保留 md/txt/image/docx/mermaid/html 预览与 DocxOutline。
- 前端 OfficeBindings 接口与 mock xlsx 族同步收缩。
- 验证：同刀0。

### 刀1：案A 图文独立引擎（核心刀）

1. **boot Board 维度**：`boot.Options` 增 `Board string`（仅 "wenxue" 非空；办公路径恒空=逐字节现状）。作用点：addBuiltins/plugins/ExtraTools 装配期 deny 名单、sysprompt（跳过晨报预载与办公项目本体段，图文档前缀）、Skills 子集（刀2 用）。
2. **工具 deny 名单单源**：`internal/app/wenxue_tool_policy.go`——office 全族（generate/apply×4/recalc/audit/batch/COM）、cad_ask、cost×4、knowledge×2、semantic_search、schedule×3、chart_gen、diagram_gen、diagram、doc_render、format_convert、screen_capture、dag_plan、browser×12。与前端 `disabledDomains.ts` 注释互指同源。
3. **独立控制器懒建**：`internal/app/wenxue_engine.go`——`wenxueCtrl()` 仿 `gaeaBuildController`：Cwd=`wenxueWorkspace()`（当前小说目录）、SessionDir=小说目录 sessions、ExtraTools=图文专属（刀3）、Board=wenxue、独立 permission 模式与事件通道。办公 `gaeaCtrl/gaeaRebuildLocked/GaeaReload` 零触碰；图文重建走独立 `WenxueReload`。
4. **聊天流绑定 fork**：前端 `mappings.ts` 中聊天流 Gaea* 全量翻 Wenxue*（Send/Stop/Resume/审批/模式/Commands/Capabilities/上下文/任务面…以 mappings 实际清单为准）；`WenxueMeta/ListSessions/ListProjectSessions/NewSession` 改读 wenxue ctrl；`WenxueNewSession` Running 检查改 wenxue ctrl。
5. **事件分线**：图文控制器 emit 至 `gaea-wenxue-event` 独立通道；前端 wenxue 树事件订阅切换；全仓 grep `gaea-event` 消费面防串台。
- 验证：Go 装配断言（图文档 schema 集不含 deny 名单任一项、办公档与现状快照逐字节一致）+ 提示词快照 + 前端 vitest + 走查。

### 刀2：技能面子集（依赖刀1）

- wenxue 引擎 Store 子集：装配期 allow 名单=novel-agents、novel-deslop、novel-review、story-long-write、story-deslop、wenxue-workbench（新）+ 中性元技能（genui、skill-creator）；包装 Store 过滤 List/Read → 索引/slash/run_skill/read_skill/能力抽屉五面同源收口。
- 内嵌图文技能三件（app 层图文 Build 注入，仿 office builtin 先例）：story-long-write、story-deslop（自 `skills/` 写作域蒸馏改写）、wenxue-workbench（工作台纪律：tuwen 目录约定 `章节/`、`角色/`、`时间线/`、`设定/`，专属工具用法，@会话/@文件 用法）。
- docx/pptx 技能：图文档索引隐藏（导出链脚本查找不受影响）。
- 办公侧零改动（办公 Store/索引/内嵌 builtin 原样）；办公提示词快照逐字节守卫。
- 验证：双引擎技能索引断言 + sysprompt 快照。

### 刀3：图文专属工具四族（依赖刀1/2；extra-app 仅图文引擎装配）

- 命名 `wenxue_*` 前缀，注册单点 `gaeaWenxueExtraTools(a)`（只进图文 Build）：
  - `wenxue_chapter_list` / `wenxue_chapter_read` / `wenxue_chapter_write`：按工作台纪律目录约定感知章节（序号/标题/字数/前后章衔接回执）。
  - `wenxue_stats`：全书/目录字数快照、角色卡/设定/时间线文件计数；可选 `.gaea/stats` 码字账本。
  - `wenxue_character_list` / `wenxue_character_read` / `wenxue_character_write`：角色卡目录 md 卡读写。
  - `wenxue_timeline_read` / `wenxue_timeline_append`：时间线条目读/追加（章号/事件）。
- 目录名宽容匹配（中文/英文变体）单源 helper；schema token 预算审读。
- RegistryMap 加行（extra-app 图文档）+ 守卫测试同步（census/extra/space_assembly）。
- 验证：纯函数单测 + 装配断言（办公引擎不含 wenxue_*）+ mock 前端面 + drift。

### 刀4：模板小说化（拍板项）

- `lib/taskTemplates.ts` 八条办公模板换小说八条（开篇钩子/续写推进/大纲推演/设定扩充/角色卡建立/时间线整理/润色去味/章末钩子）；欢迎页三卡 weekly-report/meeting-minutes/cost-estimate → 开篇/续写/大纲（i18n 新 wenxue.* 键、删 office.* 旧键，跑 locale 死键扫）。
- 验证：vitest + i18n 死键扫。

### 收尾

- 全量门禁：Go 全量 + vitest 全量 + `tsc -b` + eslint + drift。
- 构建装机按 `docs/gaea-build-guide.md` 强制纪律（孤儿进程清理→frontend npm build→ucrt64 PATH+CGO→`wails build -s -ldflags "-s -w" -trimpath`→DLL 同目录→`scripts\smoke.ps1` 冒烟 200→gaea_install.ps1 装机），记录提交号/SHA256/PID。
- 记档：本文件进度段 + `.gaea/todos.md` + progress.md + 记忆。

## 验收标准（办公零影响的实证口径）

1. 办公引擎装配快照：`boot.Build(Board="")` 产出的 schema 集与改动前逐字节一致。
2. 办公系统提示词快照逐字节一致；办公技能索引逐字节一致。
3. 图文引擎断言：deny 名单零出现、wenxue_* 四族在场、技能子集正确、Cwd=tuwen 当前小说。
4. 图文会话读写在 tuwen 内落地（pins/exports 不再锚办公工作区）。
5. 办公后台任务在图文任意操作（新建/切换/删除小说、发消息）下不受影响。
6. 前端 vitest/tsc -b/eslint/drift 全绿；冒烟 200；办公板块走查不变。

## 进度台账

- 2026-11-15：调研落档 + 本计划立档；动刀待令，零代码改动。
- 2026-11-15 晚 全六刀收官+装机（按令动刀）：
  - 刀0 死面清理 83d4cad6（memoryhub 全目录 31 件+7 死组件+Schedule×8/ConvertToPdf/DAG×13/Knowledge×13 绑定链；门面 61→52）
  - 刀0b 预览链收缩 63d33adf（xlsx/pptx 预览编辑+CrossEmbed+COM 胶囊撤除，docx 编辑三件保留）
  - 刀1 独立引擎 a7997cb9（boot.ToolFilter/SkillFilter/SkillExtras 三缝+wx 运行时+43 个会话态 Wenxue* 绑定+事件三分线+42 处 Cwd 换锚；绑定面 817）
  - 刀2 技能子集 79532dbb（allowlist 五面同源+内嵌三件+read_skill 实例化根修——双引擎全局串仓缝）
  - 刀3 专属工具四族 2514d3eb（九件 wenxue_*，RegistryMap 九行+守卫三条+doc 同步；仅 wx 装配，办公零出现有断言钉死）
  - 刀4 模板小说化 bac474cd（八条写作模板+欢迎页三快捷卡+三语 i18n）
  - locale 死键清扫 5a17f731（八组已删面板键 245×3 语）+wailsjs 同步
- 门禁终态：vitest 全量 532 文件/4901 例绿（331.98s）、Go app+gaea 全量绿、tsc -b 0、
  eslint 0 错误、drift OK@817（死绑定检测过，StrataModelStop 在册注记为存量）、locale 死键清零
- 构建装机：ucrt64 裸 wails build -s -ldflags "-s -w" -trimpath（52.1s，bindings 未跳过）；
  冒烟 /api/health 200；INSTALL OK（四件装机，exe SHA256=A0C717ACCF6E27FE6AB7768A4651EEE0
  03F94A5EB0C9A6FF3DCA7314204F6761）；拉起 PID 42724（17:40:59 > 构建时刻 17:39:46 ✓）
- 挂池（未批，后续拍板）：质检工具族、文风工具族、TaskTemplates 后端命令库图文根
  （现共享办公工作区 .gaea/commands）、记忆建议卡图文专属做梦机制、办公模板八条的
  ensureTaskTemplateCommands 图文档落盘
- 补刀（用户实弹反馈「工具面板没更新」）：能力面板工具区双源化——根因=静态清单
  （刀3 只滤办公未加图文九件）；CapabilitiesView 增 tools 实时字段（ctrl.Tools()
  名单回填，加性零影响），ToolsTabContent 按引擎实际装配过滤分组+「其他」组兜底，
  RAW_SECTIONS 补图文专属组；提交 8b??（fix(wenxue) 能力面板工具区）；重构建装机
  exe SHA256 416E4327CF2EFDB52E82C6BC9CC8EE429ED2751BC2A5FDDD8E082D02254AB7AD、
  PID 42680（17:49:52 > 构建时刻 17:49:19）、冒烟 200

## 追加刀：模型中心文学专属功能绑定（2026-11-15 深夜，同批收官）

图文引擎此前与办公共用 `gaea` 功能键（wenxue_engine 构建时读 GetFeatureModel("gaea")）——本刀给它
独立功能键 `wenxue`，模型中心可单独为文学板块绑引擎/模型：

- 配置五件套：config_keys（func_wenxue_engine/model/enabled）+configFile/Config 三字段+loadDefaults
  Enabled:true+applyConfigFile 回读+saveSetters 三 setter+config_features 四 switch
- App 层：SetFeatureModel/SetFeatureModelEnabled 增 wenxue 分支 → applyWenxueFeatureModel（跟随重建）
- **结构性补强（双引擎注入纪律）**：功能绑定注入从「GaeaInit/applyOfficeFeatureModel 预写 bridge 全局槽」
  收敛为「各 Build 位点构造期注入」——gaeaBuildController/wenxueBuildController 开头各自读自己的绑定
  （wenxue 回落链=wenxue 绑定→gaea 绑定→全局活跃；启停停用=清绑定回落全局活跃），boot.Build 同步窗内
  全部 provider（主 agent+子代理）按当前绑定解析；修掉刀1 遗留的「wx 重建可能误取办公绑定」交叉污染缝
- UI：模型中心功能绑定卡 `sin/原罪`（已退役板块的死卡）腾位为 `wenxue/文学`（BookOutlined+回落链说明）；
  原罪插图卡与 sinImage 态保留未动（独立挂池项）
- 门禁：config/app 定向测试绿（新增 config_features_test 三例：四向回环/键域隔离/saveSetters+回读链）、
  modelcenter 15 文件 119 例绿、tsc -b 0、eslint 0 错误、构建冒烟 200、INSTALL OK
  （exe SHA256 E073EE0C9D930A1E…，PID 28084 > 构建时刻 18:01:06）

## 追加刀：角色标签页移植（2026-11-15 深夜，同批收官）

小说板块角色页（SettingHub「人物」视图，characters.json+角色库数据面）形态移植到图文主区第七 tab
「角色」；数据层**不搬**——重接为小说工作空间角色卡 md（角色|characters|人物 目录），与
wenxue_character_* 专属工具同源（UI 改=agent 所见，agent 回写=UI 刷新即见）。组件
WenxueCharacterHub：网格+搜索+编辑（CAS）+新建骨架卡+删除+「让 AI 完善」（Composer 注入
@文件+工具指令）+切书刷新。测试 3 例 hermetic；装机 exe A0717380…PID 20044>构建时刻。
挂池：角色卡立绘（portrait 基建属角色库线）、关系图谱可视化（novel 关系数据模型不同源）。

## 追加刀：定稿目录+阅读标签页（2026-11-15 深夜，同批收官）

- 定稿域：定稿|final|finalized 目录只放发布稿不放草稿；新建小说/切书脚手架定稿目录；
  工具三件 wenxue_final_list/read/write（仅定稿目录内写）+域隔离（chapter_write 拒写定稿、
  final_write 拒越域）；RegistryMap 三行+doc；技能正文补定稿段。
- 阅读标签页：主区第八 tab「阅读」（WenxueReadingHub）——定稿目录导航（章号序、序章在前、
  字数徽标）+Markdown 阅读窗（本地图片按章节路径解析）+翻章+切书刷新；工具面板图文专属组补录。
- 测试：域隔离/脚手架 Go 两例+阅读 Hub 两例 hermetic；门禁全绿。
- **网页版实测翻车与修复（用户实弹反馈）**：角色/阅读 tab 点击不跳转（tab 高亮内容回落对话）——
  useSubagentTabs.handleChatTabSelect 视图白名单漏新 id（2026-11-05 同款事故复发）；修复=视图 id
  收敛单源 CHAT_VIEW_TAB_IDS（ChatTabs 导出）三方同源消费（白名单/持久化校验/渲染）；vite dev
  :5218 ?mock=1 浏览器驱动实证（tab 跳转+内容渲染+往返+截图）。装机 exe F5AF2C6B…PID 40640。
- 教训沉淀：**新增主区视图 tab 的三处同步点=ChatTabs TABS/CHAT_VIEW_TAB_IDS、
  useSubagentTabs 白名单（现已单源化）、App 持久化校验（现已单源化）——单源化后只改 ChatTabs 一处**。
