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

## 打包记录（2026-11-15）

- 包标识 `gaea-wenxue-tools-20261115-0eb7a2fc`，构建自独立干净工作树
  （detached HEAD @ `0eb7a2fc`=打包时 HEAD，主树零未提交——全部工作已入库）。
- 干净树构建：Vite 26.8s + Wails 50.6s（Skip Bindings=false）；构建后 wailsjs
  内容零漂移（29 项 status=CRLF 显示假象）；DLL 预构建件取主树 build/bin。
- 冒烟：payload 正式路径 `/api/health` 200。
- 产物：payload/gaea.exe `BAC42EE4DE7E9849974CF844CFF66AF1FCCC21A37286496F909477E83F9A932D`
  （干净树重建哈希，与主树非 clean-tree 构建的 A0717380…/F5AF2C6B… 不同属预期，
  代码同源）；preview.zip + source.tar.gz 留盘不入库；
  PACKAGE-INFO.md + SHA256SUMS-0eb7a2fc.txt 已入库。
- 工作树 `.worktrees/pkg-0eb7a2fc` 已回收。

## 绑定白名单修复（2026-11-15 晚）

- 用户实弹报「功能绑定失败，未知功能: wenxue」：config 层四向（config_features 四 case /
  config_keys 三键 / config_save 三 setter / config_types 字段）刀1 已齐、App 层重建缝
  applyWenxueFeatureModel 已在位，唯 app 层 `feature_model_handler.go` 两张白名单
  （featureModelKeys / featureModelEnabledKey）漏 "wenxue" case——UI 改绑/启停在
  core.SetFeatureModel 校验位即报「未知功能」，重建缝根本到不了。
- 修复 `a3ba3cd6`：两 case 补齐（KeyFuncWenxueEngine/Model/Enabled 三键）+ 钉
  TestSetFeatureModel_WenxueBinding 三向（绑定 / 启停 / Load 落键读回）。
  整包回归 app 147s + config 绿；ucrt64 裸 wails build 1m41s + 冒烟 200 + INSTALL OK
  （exe SHA256 879DB3D0EDE9…）。
- 教训沉淀：**config 层加功能键时，app 层两张白名单是第三个同批面**（与「新增主区 tab
  白名单漏 id」同形态——刀只对齐了自己的新面，没盘存量枚举点）。新功能键四处同批 =
  config_features 四 case + config_keys/save 键域 + feature_model_handler 两白名单 +
  模型中心 BindSection/utils 档位表。routeFeatures（路由建议面）有意不含 vision，
  wenxue 暂不补（建议面属可选扩展，非缺陷）。

## 首刀收口批（2026-11-15 深夜，众神殿 pan_1794751177012 + pan_1794754879290 两场裁决施工）

用户令：按最新众神殿对文学板块工具的论证优化文学板块，办公零影响。两场四轮+主席裁决，
12 项分歧台账全归宿；本批只施工「采纳」面，**后置面不碰**（完整账本/12→7 合并读入口/
批量 compact/结构化 writeback→刀1.5/lint 收益实测→真实稿到位后重开）。

### 采纳面（本批动刀清单）

1. **`fact_*` 除名**：`fact_add/fact_list/fact_clear` 加进 `wenxueDeniedTools`（探针实锤
   三件当前 KEEP 流进图文会话；语义=办公交付事实库，图文不装）。办公实现零动。
2. **run_skill 接可见 Skill 集合**（issue-1 收口，B 方案）：`runSkillTool` resolver 化
   + 新 `skill.NewRunSkillToolFromSet`——extras 可达（wenxue-workbench 经 run_skill
   返回正文）+ 隐藏名 fail-closed（unknown 名单也只列可见）。**办公引擎（双 nil）保持
   `NewRunSkillTool(store)` 活读原样逐字节**——boot 构造位按 filter/extras 有无分流。
3. **装配收敛单点**：sysprompt 合并改 `skill.VisibleSkills(storeList, extras, filter)`
   ——同名去重 store-first（store/extras 同名不重复且序不变），索引/read_skill/
   run_skill 三面共享同一可见集合。
4. **媒体根分档**：`image_gen`/`ocr` 根注入缝（`mediaRoot func() string` 字段，nil→
   `gaeaCwd()` 办公现状逐字节）；wx 引擎经 `wenxueBaseTools` 注入 `wenxueWorkspace`
   ——图文会话生成图/OCR 相对路径锚小说目录，不再落办公工作区。`vision` 不动
   （builtin-workspace 随 Build cwd 注入，本就正确）。
5. **删假 `dir`**：`wenxue_chapter_list` schema 删 `dir` 输入参数（执行体本就不解析；
   回执 `dir` 字段保留=自动探测结果）。
6. **共享章号 helper + 两条窄告警**（不新增第 13 个工具，纯函数挂回执 `warnings` 字段）：
   - `wenxueChapterWarnings`：未编号文件一条列表级告警 + 根目录回退一条（不逐文件重复）；
   - `wenxueTimelineWarnings`：无章号事件聚合一条（附 文件:行号，封顶 8 处）；
   - `wenxue_timeline_read` 跳过 `#` 开头 Markdown 标题行（裁决=合理默认，标题不进 events）；
   - `wenxue_timeline_append` chapter=0 不强制必填，回执加「本章事件应传 chapter」提示。
7. **workbench 回写句三子串**（`:93` 正文）：「写完一章必须回写」+ 阿拉伯数字锚点样例
   「第12章」+「不要新建 foreshadows.json」——grep 断言三条钉死。
8. **测试包**：技能面五例（①extras 经 run_skill 返回正文 ②被 filter 拒名返回 unknown
   ③办公 nil-filter 行为不变 ④store/extras 同名不重复序不变 ⑤subagent 型技能经
   run_skill 仍进 runner）+ 章节四类 fixture（001-标题/第一章→Number=0/无编号/根回退，
   断言=告警条数+0 号组排序确定）+ 时间线 fixture（合法锚/中文数字锚/标题行/缺章号行）
   + `wenxue_surface_probe_test.go` 转 census 守卫（deny 面不变式+keep 面存在性）。

### 红线与验收

- 办公逐字节零变化：双 nil 缝原样 + nil-filter 行为不变测试钉死；`fact_*` 办公实现/
  RegistryMap 契约行不动（只加图文 deny 名单）；媒体根 nil→gaeaCwd 缺省。
- 三条约束照旧：只用合成 fixture 验规则、真实稿未到不主张收益、不启动账本/跨章检索/
  工具合并。
- 遗留施工期核验（不阻塞）：novel-agents/novel-deslop/novel-review 可达性（allowlist
  三件在 repo `.gaea/skills/`，wx 引擎 store ProjectRoot=小说 cwd——除非用户 gaea.toml
  `skills.paths` 指到仓内 `.gaea/skills`，否则索引不列、亦不报 unknown，B 方案口径自洽）；
  安装分发链是否复制 `.gaea/skills`。

### 施工台账（2026-11-16 凌晨收官，全绿+装机）

- 刀A 技能面：`skill/tools.go` runSkillTool resolver 化（resolve/names 双闭包）+
  `NewRunSkillToolFromSet`（可见集合构造：extras 可达+隐藏名 fail-closed+unknown
  名单只列可见）+ `skill.VisibleSkills` 纯函数（合并+同名去重 store-first+filter 收口）。
  NewRunSkillTool(store) 行为逐字节保留（store 活读，装配后 install 新技能即可用）。
- 刀B boot：`sysprompt.go` 合并改 VisibleSkills 单装配点（索引/read_skill/run_skill
  三面共享）；`boot.go` run_skill 按 filter/extras 有无分流——**办公双 nil 保持 store
  活读原样（红线）**。
- 刀C fact_* 除名：`wenxue_tool_policy.go` deny 加三件（探针实锤 KEEP→DENY）+
  policy 测试翻转 + 前端 `disabledDomains.ts` 同批（两处同批纪律）；
  `wenxue_surface_probe_test.go` 一次性探针转正 census 守卫三例（keep 面 12 件+
  deny 面 fact/office 抽样+底座在场）。
- 刀D 工具面：chapter_list 删假 `dir` 输入参数（schema 无参化）；共享章号 helper
  两件（`wenxueStemNumber` 前导数字解析/`wenxueAnchorNumber` 第N章锚解析——解析
  与告警同源）；`wenxueChapterWarnings`（未编号聚合一条+根回退一条，列表级不逐
  文件）挂 `wenxue_chapter_list` 回执 `warnings`（仅非空携带）；timeline `#` 标题行
  跳过（不进 events）+`wenxueTimelineWarnings`（缺锚聚合一条附 文件:行号 封顶 8）+
  timeline_append 缺锚回执 hint「本章事件应传 chapter」（不强制必填）。
- 刀E 媒体根分档：`imageGenTool`/`ocrTool` 加 `mediaRoot func() string` 缝
  （nil=gaeaCwd 办公现状逐字节）；`gaeaExtraTools`→`extraToolsWithMediaRoot(a, nil)`
  壳；wx 引擎 `wenxueBaseTools`（注入 wenxueWorkspace）——生成图与 OCR 相对路径锚
  当前小说目录。vision 不动（builtin-workspace 随 Build Cwd，本就正确）。
- 刀F 文案：workbench 回写句改写，三子串齐备（写完一章必须回写/第12章/不要新建
  foreshadows.json）。
- 刀G 测试：skill 包五例（extras 可达/隐藏名 fail-closed+名单不泄露/办公活读不变/
  同名去重 store-first/subagent 进 runner）+ app 包五例（章节四类 fixture+0 号组
  排序确定+告警条数/根回退告警/时间线四类 fixture+标题不进 events+行号保序/
  追加 hint/三子串断言）。
- 门禁：Go app 106s+boot+agent+control+skill+tool+config 全量绿（gaea/... 全家族
  ok）；`tsc -b` 0；eslint 0；gofmt 改动面清（bindings_* 生成件不 gofmt 属已知）。
  ucrt64 裸 `wails build -s -ldflags "-s -w" -trimpath` 1m01.8s（bindings 未跳过，
  前置 `npm run build` 32.6s）；冒烟 `/api/health` 200（PID 39564 已回收）；
  INSTALL OK（exe SHA256 322C6E5E4664…，PID 22312 启动 00:11:34>构建 00:11:14）。
- 遗留核验结论：novel-* 三件**当前不可达**（用户 gaea.toml 无 `[skills] paths`，
  wx store ProjectRoot=小说目录不扫仓内 `.gaea/skills`）——索引不列、run_skill 不
  报 unknown，B 方案口径自洽；如需点亮七角色卡，用户侧在 gaea.toml 配
  `[skills] paths = ["<仓>/.gaea/skills"]` 或等安装分发链批。不改默认行为（动配置
  归用户拍板）。

### 验收合同收尾（2026-11-16 续批：动态终验+分发链核验）

- **四段式第四腿闭合**（施工后动态各调一次 read_skill/run_skill）：
  `internal/app/wenxue_skills_dynamic_test.go`——真实 boot.Build 全链装配
  （真实 wenxueSkillFilter/SkillExtras/workbench 正文，mock provider hermetic，
  纪律同 TestGaeaHistoryGolden：chdir 不改 APPDATA）。五断言：图文面
  run_skill/read_skill 动态调 `wenxue-workbench` 双双返回含「写完一章必须回写」
  真实正文；store 真实夹具 `gongwen.md` 被 allowlist 拒名后 run_skill 报 unknown、
  read_skill 报 not found（fail-closed 不依赖用户 home 偶然内容）；办公面（双 nil）
  同一夹具 run_skill 活读成功——nil-filter 行为不变的动态证明。一次通过。
- **安装分发链核验**（遗留第②条）：`gaea_install.ps1` 不复制 `.gaea/skills`（grep
  零命中）；装机 home 根 `~/.gaea/skills` 仅 docx/pdf/pptx/xlsx（无 novel-*，且
  四件全被图文 allowlist 拒）。结论=装机位图文会话的 novel-* 可达性与开发仓一致
  （不可达、口径自洽）；点亮路径仍=用户配 `[skills] paths`（或另立分发批）。
- 工具面口径问答归档：图文会话无 PPT/Excel 工具（xlsx 四件/pptx_apply/office 族
  deny 名单物理不装+技能索引隐藏）；唯一保留=create_docx.py/create_pptx.py 导出
  脚本按路径依赖（藏索引不删目录），交付口径 txt/docx 为主。

## 角色标签面板重设计批（2026-11-16 深夜，v2 master-detail+角色库只读联动，已装机）

用户令：重新设计文学板块角色标签面板，与角色工具联动，新建角色与 gaea 角色卡联动，
**不能污染角色库**。纯前端批（零 Go 改动、零新绑定，绑定面/drift 不变）。

- **v2 形态**（WenxueCharacterHub 重写）：master-detail 双栏——左列表（搜索/刷新/
  新建/卡列表：名字+字数+摘要），右详情：**阅读态=结构化六节**（解析 `# 名`+`## 节`，
  缺节虚线占位「未填写」；自由格式卡回退原文展示）/**编辑态=裸文本 CAS**（保 md
  契约与并发安全；脏态退出编辑有守卫提示）。数据面不变：工作空间角色/ 目录 md，
  与 wenxue_character_* 工具同源。
- **联动（与角色工具）**：①订阅 `gaea-wenxue-filetree` 通道——变更路径落在角色/
  目录即自动重载（agent wenxue_character_write 回写 UI 即见，不再手动刷新）；
  ②未脏态选中卡被外部更新时静默跟随最新（基线同步，脏草稿不动 CAS 兜底）；
  ③「让 AI 完善」保留（Composer 注入六节补全指令+wenxue_character_write）。
- **新建角色双通道**（Modal Tabs）：空白卡（原名输入+六节骨架）/ **从角色库导入**
  （只读联动 gaea 角色库：CharacterList 查询分页列表→选卡→字段映射六节落工作
  空间卡）。映射：设定←定位(RoleType/Gender/Age 中文)+标签+背景+动机；外貌←外貌
  +身材；性格←性格+弧线；口头禅←对白样本(- 行)；关系←留待书中发展（库无关系
  字段）；当前状态←状态(Alive→存活等)+备注。
- **红线（不能污染角色库）**：面板全程只调 `CharacterList`（读）；不触 Save/Delete/
  Import*/Set*/Associate*/Sync 等任何写向/登记绑定——导入=读库卡映射为工作空间
  卡，库卡不受影响，两边此后各自演进。测试钉死：写向绑定 mock 全挂监视，断言
  全流程零调用。
- 测试（WenxueCharacterHub.test.tsx 重写 5 例）：列表/同源说明条；阅读态结构化+
  编辑态 CAS；空白卡六节骨架；**库导入映射+红线零调用**；filetree 通道自动重载
  （桩 window.runtime.EventsOn 捕获订阅回调）。wenxue 全族 15/15 绿；tsc -b 0
  （真退出码）；eslint 0；Vite 31.1s+Wails build+冒烟 200+INSTALL OK（PID 8000
  >构建时刻）。
- 坑：`bodyOf([c.appearance, c.figure])` 可选字段数组→`(string|undefined)[]`
  炸 tsc（noUncheckedIndexedAccess 面）——`?? ""` 归一；管道后 `$?` 是 tail 的
  （三犯），门禁一律 `cmd > log 2>&1; echo EXIT=$?` 落真码。

## 记忆左右脑批（2026-11-16 深夜立项，用户令：图文记忆与办公隔离分属左右脑）

**现状盘点（调研实锤）**：两引擎 `boot.Build→buildSystemPrompt` 都从
`config.MemoryUserDir()`（<AppConfig>/gaea）取记忆——①用户级 doc 记忆/画像文档
共享读写；②ProfileStore（remember 工具画像）同一 DB 全局表；③Hephaestus.db 同
文件（auto-memory 行按 cwd 项目键已天然分，但 DB 本体/其他全局表共享）；④UI 侧
wenxue 主区「记忆」tab 走 app 级全局脑绑定（app.Memory/Remember/Forget/DreamPurge
…）=挂在办公脑上的窗，图文 UI 可读写办公记忆=双向污染。已天然隔离面：会话目录
（SessionDir）、pins（CWD 键）、项目 doc 记忆（AGENTS.md 等随 CWD）、auto-memory
sqlite 行（cwd 项目键）、memory_search 主路径（控制器实例索引）、做梦/晨报/建议
（wx 有意不挂=办公域机制）。

**刀1 引擎脑（Go）**：`boot.Options.MemoryUserDir` 新缝（""=现状
config.MemoryUserDir()，办公逐字节）——effective dir 一处算好，替换四处消费点：
sysprompt 主 Load+Migrate（文档发现/文件后端/DocPath ScopeUser 写全随目录）、
boot.go:405 sessionquery 会话检索 DB、boot.go:435 remember ProfileStore DB、
boot.go:609 引用消毒 store+EventLog。**DB 路径随 userDir 推导**（GetDatabase 池按
目录）→ wx 脑=独立目录+独立 DB 一条缝整脑分离；控制器热重载复用 c.mem.UserDir/DB
零改动；promote_session_facts 走 ctx 控制器实例自动随脑；memory_search 空间分槽
兜底不动（主路径已实例隔离）。wx 引擎传 `<MemoryUserDir()>/wenxue-brain`
（base 空则不 scope）。
**刀2 UI 脑窗退役（前端）**：wenxue 主区「记忆」tab 退役（它是办公脑的窗：做梦
清洗/晨报/建议卡/蒸馏全是办公域机制且 wx 不跑）——ChatTabs 三处单源
（ChatTabId 类型/CHAT_VIEW_TAB_IDS/TABS+Brain 图标）+App 分支/MemoryPanel lazy
import/openMemory+effect/命令 case"memory"/palette cmd-memory/topbar.memory locale
键三语；`const [, setMemView]` 保 hook 传参免大改 useSessionHandlers；图文脑的
读写走会话内 remember/forget/memory_search 工具（工具面不动）。
**刀3 隔离测试（Go）**：boot 级双脑 Build——脑 A remember 落盘后脑 B 不可见，
互不串（SQLite 按目录断言）。
**边界拍板点（用户须知）**：wx 脑**从零起步不迁移**——既有共享记忆属办公脑留存
（办公零影响优先）；图文旧会话 remember 过的事实不可溯源地留在办公脑。

### 施工台账（2026-11-16 深夜收官，全绿+装机）

- 刀1（Go）：`boot.Options.MemoryUserDir` 缝 + Build 内 `memUserDir :=
  orDefault(opts.MemoryUserDir, config.MemoryUserDir())` 一处算好，四处消费点统一：
  buildSystemPrompt 主 Load+Migrate（签名加 memUserDir 参数）、sessionquery 会话检索
  DB、remember ProfileStore DB、引用消毒 store+EventLog。wx 引擎
  `MemoryUserDir: wenxueBrainDir()`（=`<MemoryUserDir>/wenxue-brain`，基目录空则
  不 scope）。办公（不传）=逐字节现状。
- 刀2（前端）：记忆 tab 退役——ChatTabs（ChatTabId 类型/CHAT_VIEW_TAB_IDS 单源/
  TABS 条目+Brain 图标）+App（MemoryPanel lazy import/`const [, setMemView]` 保
  useSessionHandlers 契约/openMemory+激活 effect/命令 case"memory"/palette
  cmd-memory/useSessionHandlers 记忆族解构摘除/Brain 图标 import）+command.ts
  （`/memory` union 成员+解析）+三语 locale 死键 topbar.memory。持久化校验走单源
  自动回落 chat（旧存档 tab=memory 安全降级）。图文脑读写=会话内
  remember/forget/memory_search 工具（工具面不动，随脑）。
- 刀3（测试）：`boot/memory_brain_isolation_test.go` 双脑 Build——脑 A remember
  落盘后：两脑各自独立 Hephaestus.db 落位、脑 A 应回读、脑 B 三向不可见
  （memory_get not found/独立 DB 回读无/索引无）、脑 B 自有事实独立落盘可读、
  脑 A 不可见。hermetic=两脑显式 temp 目录（测试从不触真实默认脑）。
- 坑：`db.GetDatabase` 池按目录持句柄——TempDir 清理前须 `db.CloseDatabase`
  （Windows 锁文件）；boot 的 `testKind` 在 `boot_test` 外部测试包，内部包测试
  文件要自备 kind 序号 helper。
- 门禁：Go 全量绿（gaea/...+app 106s，GO_EXIT=0 双核过）；前端 tsc -b 0、
  eslint 0 错误（2 警告=既有 react-refresh 常量导出同款）、wenxue 族 15/15、
  mock 契约 5/5（绑定面零变化）；Vite 31.2s+Wails 53.0s+冒烟 200+INSTALL OK
  （KILL FAIL 瞬态一次→重试过，PID 35668>构建时刻）。


- 口径漂移顺手修：`wenxue_tools.go` 头注「RegistryMap 九行」→十二行（众神殿指出的
  名单双事实源漂移之一收口）。
- 后置面不动（按裁决）：完整账本/12→7 合并读入口/批量 compact/结构化 writeback
  （刀1.5）/lint 收益实测（真实稿到位后重开）。

## 设定标签页批（2026-11-16 深夜立项，用户令：移编队 tab+增设定 tab——查看/编辑世界观·时间线·规则+AI 生成填充）

**方案**：纯前端批（零 Go——设定目录约定 `设定|worldview|设定集` 与时间线
`时间线|timeline` 已在 wenxueDirCandidates/wenxue_stats/wenxue_timeline_* 工具侧
存在，UI 直读文件即可）。①编队 tab 退役：ChatTabs 三处单源摘 fleet+App 摘
Fleet 分支/useFleetAutoOpen 接线（后台任务自动切编队的钩子随 tab 一起退役；
子代理进度仍在对话流/轨迹可见）。②新增设定 tab（id=settings，ScrollText 图标）
= WenxueSettingsHub：master-detail（左列表「设定文档」组+「时间线」组，右详情）；
设定文档结构化阅读（# 标题/## 小节/列表轻渲染）+CAS 裸文本编辑；时间线结构化
事件视图（第N章锚解析+章序+未锚分组）；AI 生成 Modal（预设 世界观/力量体系/
规则/势力/地理/自定义 → Composer 注入生成指令，agent 用 write_file 落
设定/<名>.md）+新建空白文档+删除；filetree 通道自动重载（设定/时间线目录）。
同源口径：与 agent 的 write_file/时间线工具共享同一数据面。

### 施工台账（2026-11-16 深夜收官，全绿+装机）

- ChatTabs：type/CHAT_VIEW_TAB_IDS/TABS 三处单源——摘 fleet（Blocks 图标随退）、
  加 settings（ScrollText 图标，id 序排在 reading 后）。旧存档 tab=fleet 走单源
  校验自动回落 chat。
- App：Fleet import/渲染分支/useFleetAutoOpen import+接线（openFleetView+hook
  调用，后台任务自动切编队随 tab 退役；subagentRuns 解构同步摘）/
  handleSend deps 里失效的 setChatTab 顺手清（eslint 警告归零）；接
  WenxueSettingsHub lazy import+渲染分支。
- WenxueSettingsHub（新组件 ~470 行）：master-detail 双栏——左列表「设定文档」
  （设定|worldview|设定集）+「时间线」（时间线|timeline）两组+搜索+刷新+新建+
  AI 生成；右详情：设定文档轻渲染（#/## 标题/列表/段落）+CAS 裸文本编辑；时间线
  结构化事件视图（`第N章` ASCII 锚解析、未锚组列最前与后端同口径、中文序号行
  如实展示+改写提示）；编辑态完成钮带脏守卫；删除确认。AI 生成 Modal=预设五类
  （世界观/力量体系/规则/势力/地理→设定/<名>.md 各带 brief）+自定义名；注入
  Composer 指令要求 agent 先 wenxue_chapter_list/character_list/timeline_read
  摸底再 write_file 落盘（已有文件先 read 再修订）；落盘后 filetree 通道自动
  重载出文档。「让 AI 完善」按文档注入一致性补全指令（时间线冲突以时间线为准）。
  未脏态外部更新静默跟随（CAS 基线同步）。
- 测试（WenxueSettingsHub.test.tsx 6 例）：两组列表+同源条；轻渲染+编辑 CAS 保存；
  时间线章序/未锚最前/中文序号提示；AI 生成注入断言（目标文件+摸底工具名）；
  filetree 通道自动重载（桩 window.runtime）；新建空白骨架。wenxue 族 21/21 绿。
- 门禁：tsc -b 0；eslint 0 错误 1 警告（react-refresh 常量导出既有款）；Vite
  32.3s+Wails 55.9s+冒烟 200+INSTALL OK（PID 49276>构建时刻）。零 Go 改动
  （设定/时间线目录约定与工具后端本就存在）；绑定面零变化。

## 功能打包（2026-11-16 深夜收官）

- 包 `releases/gaea-wenxue-v2-20261116-a7cc7c2a/`：独立干净工作树（detached
  @`a7cc7c2a`=打包时 HEAD，主树零未提交；junction node_modules）Vite 29.59s+
  Wails 37.4s（bindings 未跳过）；构建后 wailsjs 内容零漂移（`git diff --quiet`
  通过，warning 全 CRLF 显示假象）；DLL 预构建件取主树 build/bin（同源同哈希）。
- 冒烟：payload 正式路径 `/api/health` 200（PID 49344）。
- 产物：payload/gaea.exe SHA256=`0211079464f765c8…4a2153`（干净树重建哈希，与
  主树装机位构建哈希不同属预期，代码同源）；onnxruntime `68d0d5f5…` /
  sherpa-onnx-c `5dd36dbd…` / sherpa-onnx-cxx `4c563277…`（与上包同源同哈希）。
  PACKAGE-INFO.md + SHA256SUMS-a7cc7c2a.txt 入库；zip/tar 留盘不入库（各约 29 MB）。
- 工作树已回收；顺手清 `.worktrees/` 两个既往包空壳残留（pkg-0eb7a2fc/
  pkg-eb287ecc，零文件空目录）。
- 本包功能面=448743a8..a7cc7c2a 六笔（工具首刀收口→续议修复落库→动态终验→
  角色面板 v2→记忆左右脑→设定标签页），台账见本文件各批段。


