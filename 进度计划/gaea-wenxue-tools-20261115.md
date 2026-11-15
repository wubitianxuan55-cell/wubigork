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
