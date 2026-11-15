# 图文（wenxue）板块工具/技能梳理调研（2026-11-15）

> 任务：重新梳理图文板块的工具，删除不需要的工具和技能，增加专属工具；硬红线=不影响办公板块。
> 本文为调研事实源；动刀计划见 `进度计划/gaea-wenxue-tools-20261115.md`。
> 图文板块 = wenxue（文学），2026-11-15 立板：办公板块完整前端拷贝（436 文件），工作空间限定 `C:\AI\tuwen`（每部小说一个子目录），长篇小说创作专属板块。

---

## 1. 现状地图

### 1.1 板块与引擎共享现状

- 前端 manifest：`frontend/src/boards/manifests.ts:153-160`（id=wenxue，space='work'，WenxuePage）。
- 后端 manifest：`internal/app/board/builtins.go:177-188`（SpaceWork，Bindings=[WenxueB, OfficeB, MemoryB, CostB]）；CanonicalIDs 已去 sin 加 wenxue（:223-224）。
- 文件面隔离（已完成）：`internal/app/wenxue_workspace.go`（2141 行，WenxueB 61 方法 fork 自办公，根=`C:\AI\tuwen`，`GAEA_TUWEN_ROOT` 测试可换；头注 :5-9 明言「会话/聊天引擎仍与办公全共享，仅文件面隔离」）。
- 会话落盘隔离（已完成）：`internal/gaea/control/controller_session.go:61-68` `SetSessionDir`——全仓唯一 wenxue 侧引擎改动。
- 前端绑定面：`wenxue/lib/bridge/wenxue.ts`（WenxueExtra 仅 8 个工作空间方法）；`mappings.ts:9-10` Submit→**GaeaSend**（聊天流仍走共享办公 send 链）。
- 门面：`internal/app/bindings_wenxue.go`（生成件，61 委托方法）；装配位 `bindings_manifest.go:19-26`（SinB→WenxueB 位次替换）。

### 1.2 工具装配机制（为什么图文工具面与办公逐字节相同）

- Tool 抽象/注册：`internal/gaea/tool/tool.go:18-32`；builtin 全集 init 自注册（:140-167）；运行时 Registry（:192-358，`FilteredSchemas` :358 = 模型可见 schema 唯一出口；`Hide` :230 隐藏≠摘除）。
- RegistryMap 契约账本：`internal/gaea/tool/registrymap.go:43-152`；`Spaces` 字段（work/shared/play；independent 只是导航概念不入工具空间）；四外部入口=builtin / ExtraTools（`gaeaExtraTools` 单点，`gaea_handler.go:885-933` + specialist `gaea_specialist_tools.go:18-24`）/ 插件 MCP / build-internal。
- 空间过滤唯一谓词：`builtin/spacetags.go:155-164` `AllowsSpace`——**装配期一次性物理过滤**（boot.Build），运行时 executeOne 不过滤；缺省 shared=fail-open；spaceTags 静态表 :27-132。
- 装配链：`gaea_handler.go:108-197` `gaeaBuildController`（space=`cfg.EffectiveSessionSpace()` :134-137）→ `boot.Build`（`boot.go:126`）→ `addBuiltins`（:863-889，AllowsSpace :866/:877）+ ExtraTools（:645-649，同谓词）+ 插件（`plugins.go:49`）+ MCP 热插（`controller_mcp.go:58-76` spaceAllowedTools）。
- 每回合工具清单真实出口：`agent/agent_stream.go:18-45` `currentSchemas`（activeSchemas 覆盖 → resident∪activated → 全量）；`SetActiveSchemas`（`agent.go:578-585`）为**休眠通路**（注释说给 controller 用，全仓无生产调用方）。
- 控制器不做工具过滤：`controller_session.go` NewSession/Resume（:21-104）。
- **board 维度不存在**：`board.Manifest.Tools` 字段（`manifest.go:40`）无消费方（纯声明位）；`wenxue/tuwen` 在 `internal/gaea/{control,tool,boot}` 下仅 `controller_session.go:58` 一条注释。
- 按场景自定义工具集的仓内先例：`internal/app/pantheon_grok_agent.go:30-80`——议席用 `builtin.Workspace.Tools(names...)`（`builtin/workspace.go:32`）白名单自建 Registry。
- dynamic_tools（A2 两刀）：`boot/dynamic_tools.go`（residentToolsFor :36-79）；**可见性≠授权**。
- 权限按空间：`cfg.PermissionsForSpace`（`gaea_spaces.go:143-168`；只有 work/play 两段，无 board 键）。

### 1.3 图文会话现状工具清单（= work 全量）

| 族 | 工具 |
|---|---|
| 文件/编辑 | read_file、write_file、edit_file、edit_lines、multi_edit、move_file、replace、grep、ls |
| shell | bash、bash_output、kill_shell、wait |
| 生成/读图 | format_convert、chart_gen、diagram_gen、diagram、vision、doc_render、screen_capture、ocr |
| 网络/检索 | web_fetch、web_search、semantic_search、knowledge_add、knowledge_search |
| 造价 | cost_search、cost_save、cost_compose、cost_indicators |
| 杂项 | translate_text、routine_llm、fact_add/list/clear、dag_plan、cad_ask、sidebar_open、browser_*×12 |
| office（extra-app） | office_generate、docx_apply、docx_accept、pptx_apply、xlsx_apply、xlsx_profile、xlsx_recalc、xlsx_audit、office_batch + COM 档（office_open/status/read/edit/render/save/close/help/macro/selection，本机有 Office/WPS 才注册） |
| schedule（shared） | schedule_get、schedule_apply、schedule_analyze |
| play（work 不装） | image_gen、author_memory |
| shared 元工具 | ask、todo_write、task、memory_*、remember/forget/promote_session_facts、read_spill、recall_archived、genui_validate、run_skill/read_skill/install_skill、slash_command、summarize_file、session_search、subagent_list、read_subagent_result、exit_plan_mode、interrupt_agent、request_permission、complete_step、discover_tools、MCP 全部 |

### 1.4 Cwd 地基坑（本次最重要发现）

- 共享引擎 `Cwd` 恒=`ga.cfg.Workspace`（`gaea_ui_extra.go:466-479`；persistWorkspaceLocked :1150-1167 仅办公切换路径调用）。图文只切 SessionDir。
- 后果：图文会话内 `read_file/write_file/edit/bash`、**pins 账**（`sysprompt.go:111-113` pins.Block(cwd)；`WenxuePinMaterial` 也走 `pins.Add(gaeaCwd(),…)`，wenxue_workspace.go:1171-1176，与办公共用同一份 pin 账）、`.gaea/exports` 分区——**全锚办公工作区**。
- 图文文件面内部尚有两种锚并存：`wenxueWorkspace()`（:93/:386 ListWorkspaces/FileTreeWatch 用）vs `gaeaCwd()`（:1053 ListDir、:1367 ReadFile 等 fork 件用）——半完成隔离态。
- 结论：任何「图文专属工具集」动刀前必须先把会话工具根切到 `C:\AI\tuwen`，否则名单换什么都操作错目录。

### 1.5 系统提示词组装

- 单点：`boot/sysprompt.go:43-184` `buildSystemPrompt`（boot.go:214 调用）。
- work-only 段：晨报预载（:86-90）、项目本体（:94-98）——图文会话现在也吃这两段。
- pinned materials：`pins.Block(cwd)`（:111-113，锚办公 cwd）。
- 技能索引：:139 `skill.ApplyIndex`（`skill/index.go:40`；`IndexMaxChars=4000` rune 预算单点 index.go:12）。
- 记忆 Compose 按 space 收窄读端（:75）；无 board 维度 → 图文提示词与办公逐字节相同。

### 1.6 技能系统（两套并行 + 一条硬依赖）

**A. 引擎技能 `internal/gaea/skill`（agent 会话用）**
- Store：`skill.go:89`（New :99，List :208，Read :245，parse :311 解析 frontmatter name/description/allowed-tools/runas/context/agent/model）。
- 扫描根：`config.ConventionDirs=[".gaea",".agents",".agent",".claude"]`（`config_paths.go:153`）×（项目根 `<cwd>` → custom paths gaea.toml `[skills] paths`（`config.go:566-573`）→ home）；优先级 project>custom>global>builtin（skill.go:26-35,157-179）。
- 注入：`sysprompt.go:137-141`（Store 每会话装配；ApplyIndex 索引；WireReadSkillResolver）；`run_skill`（`skill/tools.go:34-101`；inline 哨兵 :316-318 / subagent / pipeline :104-125）；`read_skill`（`tool/builtin/readskill.go:20`）；`install_skill` 写 project `.gaea/skills` 或 global `~/.gaea/skills`（tools.go:218-231）；`/name` slash 条目与索引同源切片（boot.go:627-633）；能力抽屉 `GaeaCapabilities`（`gaea_ui_meta.go:632-655`）。
- **作用域机制：无**。space 形参在 sysprompt.go:138 手上但只用于记忆；技能全量注入。最自然过滤点=sysprompt.go:138（同源收口索引/slash/能力抽屉三面）+ run_skill/read_skill 收口（包装 Store 过滤 List/Read）。

**B. 写作域技能 `internal/skill`（小说创作页专用，与引擎无关）**
- Loader：`internal/skill/skill.go:21-50`（frontmatter `applies_to`）；`FilterByAppliesTo` :72；`InjectSkill` :86（整篇拼 system prompt）。
- 实例化：`writing_state.go:51`（`<ResourceDir>/skills`）；注入点 `create_chapter_handler.go:183`；前端 `CreateInspector.tsx:50,172`。
- 现存：story-long-write（黄金三章/节奏/爽点密度）、story-deslop（去 AI 味禁词表）——天然只属于小说创作页，不在引擎索引。

**C. 引擎面现役技能清单（图文会话现在全可见）**

| 技能 | 域 | 来源 | Go 硬依赖 |
|---|---|---|---|
| docx / pptx | 办公 | `.gaea/skills/`（scripts/create_docx.py / create_pptx.py） | **是**：办公+图文导出链 |
| pdf / xlsx / pptx-template | 办公 | `.gaea/skills/` | 否 |
| gongwen-9704 | 办公（公文） | `.gaea/skills/`（scripts/docx_gen.py） | 否 |
| cost-compose | 办公（造价） | `.gaea/skills/` | 软：params.json 覆盖（gaea_cost_compose.go:28） |
| schedule-dcma | 办公（进度体检） | `.gaea/skills/` | 未接线 |
| skill-creator | 元技能 | `.gaea/skills/` | 否 |
| novel-agents | **小说**（七角色子代理卡） | `.gaea/skills/agents/*.md` | 软：spawn 模板（cache/spawn.go:84，boot.go:979） |
| novel-deslop | **小说**（去 AI 味口径） | `.gaea/skills/` | 软：words/patterns.json 覆盖（novel_deslop_handler.go:33,49） |
| novel-review | **小说**（章节评审） | `.gaea/skills/` | 软：rubric.json（novel_review_handler.go:32） |
| office-gongwen / genui / cad-takeoff / office-edit / schedule-edit / format-convert / chart-builder / doc-assemble | 办公/中性 | Go 内嵌 `skill/builtins.go:18-271`（+gongwen_body.go、cad_takeoff_body.go） | 随二进制 |

**D. 硬依赖结论**：`docx/pptx` 技能目录不能删（图文 `WenxueExportDeliverable→wenxueExportDocx/Pptx`（wenxue_workspace.go:1893-1896/1925-1928）经 `wenxueFindSkillScript` :1948-1966 按路径找脚本，不走索引）——索引可藏、脚本可用，互不影响。

### 1.7 小说域存量资产（图文专属工具的原料）

- play 空间 agent 工具仅 2 件：`image_gen`（gaea_tools.go:24）、`author_memory`（gaea_author_memory_tool.go:46 自声明 play）。小说机器**不是 agent 工具**——全部为 NovelB 绑定直连（`bindings_novel.go:12-142` 约 130 方法，接收器 writingState，依赖 getPM() 项目管理器）。
- 机器族：角色（characters.json+角色库 SQLite）、设定（worldview.md）、大纲（outline.json）、章节生成（chapters/，create_chapter_handler.go:39）、章节计划（plans.json）、场景（scenes+场景圣经）、定稿回写（finalize_store）、状态账本（narrative journal）、一致性（consistency_deep_handler.go:66 + internal/graph）、门禁（`internal/novelgate/novelgate.go` 五类确定性校验：unknown_entity/missing_reference/state_regression/cast_mismatch/invalid_action）、伏笔（foreshadows.json+due 预警）、文风（`internal/novelstyle` digest/fingerprint）、评审（novelreview rubric）、拆解/扫榜/技法库、上下文清单（ctx_inventory dry-run）、书源导入（novel_booksource_handler+internal/booksource）、阅读问答。
- 可包壳纯函数库（不依赖 getPM 的优先）：novelgate（五类校验）、novelstyle（指纹/去味）、novelcontext（分维度预算）、internal/graph（规则层）、create_chapter_context.go:14-66 预算常量族。
- prompt 模板 22 个小说相关（prompts/*.json；三级解析 override→磁盘→embed，prompt.go:142；BuildUserPrompt P0/P1/P2 :229）——含 create-chapter-first 开篇钩子、stage-recap 十章合账、chapter-plan 七字段、plot-branch-browser、novel-dissect/scan 等。
- 直接继承的引擎机制（图文会话免开发）：@会话/@文件 引用（`control/refs_session.go:9-45`，refs.go:17/109/171；每书独立 sessionDir 后检索域按书隔离）、compact 双阈值（`agent/compact.go:108-157`，config.go:723-753）。
- sin 核查：功能代码删净（147 文件 −24642 行）；`internal/booksource` 是小说板块自有书源线（novel_booksource_handler.go:7 零 sin 引用声明），**勿误删**；良性残留=历史注释 + chat_service.go:772-785 存量 mode=sin 防御过滤。

### 1.8 前端现状（wenxue 树）

- 唯一过滤器：`wenxue/lib/disabledDomains.ts`（:3-4 前缀 knowledge/schedule/cad ×命令/工具；消费点 `useComposerMenus.ts:24-25`、`App.tsx:290-294` 发送侧拦截、`ToolCard.tsx:227`、`useToolStats.ts:13`）。
- 过滤缺口：`capabilities/ToolsSection.tsx:50-58` SECTIONS 仍展示文档技能（docx/xlsx/pdf/pptx）与通用办公（format_convert/chart_gen/routine_llm）；`SkillsSection.tsx:146-149` 只按搜索词过滤。
- 右栏 Tab：`workspaceTabs.ts:34,53-62`（files/deliverables/browser/git/ui；知识库/CAD/进度计划已撤）。主区六 Tab（ChatTabs.tsx:23-30）。
- 死面（零消费方，可删）：WenxueSchedule×8 绑定全链（`bindingNames.ts:752-759`、两份 bindingSignatures :771-778、mappings :104-114、`mock/office/schedule.ts`、spaceBindings :494-505；UI 消费方零；另 ScheduleExportXlsx/ImportXlsx/ImportMpp→Gaea* 三条映射未 fork 也属进度计划面）、FilePreviewModal（全树无挂载；`WenxueConvertToPdf` 唯一消费方 :143）、DagPanel、TaskInboxPanel、MorningBriefCard、CostLibraryView+cost_library_modals+cost_library_parts、memoryhub/CostInquiryPanel+CostBackupCard、memoryhub 资料库死视图族（MaterialsLibrary/DatabaseLibrary/WhisperMemoryLibrary/OfficeMemoryLibrary/ComposeModal）、知识库残余（types/knowledge.ts、mock/memory.ts:42-44,286-288、domainColors.ts:13,27,39、GraphView.tsx:280 文案）。
- 活面（经用户拍板处置，见 §2）：ExportMenu（App.tsx:233-251,757-763；md/docx/pdf）、ZipDeliverables（DeliverablesPanel.tsx:333-347）、FilePreview 办公预览链（DocxPreview/PptxOutline/PptxEditPanel/XlsxPreview+FormulaBar/MiniChart/SheetGrid/numFmt、OfficeLiveBadge、CrossEmbed XlsxPreview.tsx:246）、Ctrl+K 八条办公任务模板（lib/taskTemplates.ts:6-56，App.tsx:597-607）、欢迎页三办公快捷卡（App.tsx:543-556,931-932，i18n office.quickReport/quickMinutes/quickCost）。
- 跨树依赖：wenxue→`src/gaea` **零**；→src 顶层共享件（genui、characterlib usePortraitUrl、api/httpToken、boards/space、V3Empty、wailsjs models、`src/schedule` CPM 仅 mock 用）。
- mock 组织：`mock.ts` 聚合 16 域；`mock/office/build.ts:11-14`=officeMethods+xlsxMethods+scheduleMethods；drift 锁 MOCK_ONLY_NAMES 空（drift.ts:49-53）。

---

## 2. 用户拍板记录（2026-11-15，AskUserQuestion 三问）

1. **隔离形态=案A 图文独立引擎**：仿 pantheon 先例，图文懒建独立 controller+boot.Build（Cwd=tuwen、专属工具名单、专属提示词/技能过滤），办公引擎零感知。否决案B（单引擎按板块重建——会杀办公后台任务）与案C（轻刀过渡——Cwd 地基不牢）。
2. **前端活面处置**：撤 xlsx/pptx 预览+CrossEmbed；卸载 browser_*×12 工具；任务模板换小说模板。**保留**：导出面（ExportMenu md/docx/pdf + ZipDeliverables 打包）。
3. **首批专属工具四族**：章节工具族、统计族、角色工具族、时间线工具族（后两族用户点名追加）。**未批挂池**：质检工具族、文风工具族。

---

## 3. 刀路方案比选（调研结论存档）

| 方案 | 内容 | 结论 |
|---|---|---|
| E1 SetActiveSchemas 接线 | 会话级 schema 子集（休眠通路现成） | 弃：隐藏≠禁用、提示词仍办公档、切换打断前缀缓存 |
| E2 单引擎按板块重建 | 进图文 gaeaRebuildLocked 换装 | 弃（拍板）：重建杀办公后台任务/子代理 |
| **E3 图文独立引擎（案A）** | 懒建独立 controller+boot.Build，Board 维度 | **采用**；pantheon 先例 pantheon_grok_agent.go:30-80 |
| E4 第三空间 "tuwen" | 扩 space 枚举 | 弃：触碰 work 恒现状路径兼容红线与 S2 恢复 fail-closed 链，回归面最大；board 维度可替代 |
| E5 A2 常驻目录 | residentToolsFor 按板块名单 | 辅助项：仅可见性降载，不能单独用 |

## 4. 风险清单

- 双引擎事件串台：现事件走全局 `gaea-event` 通道，图文独立引擎须分线（图文专用通道），前端订阅面同步切。
- 生命周期交互：办公 `GaeaReload`/`gaeaRebuildLocked` 不得触碰图文引擎；图文引擎重建走独立入口。
- 权限面：图文引擎 permission 模式独立状态（现 WenxueNewSession 检查共享 ctrl Running，须改查 wenxue ctrl）。
- 绑定再生纪律：删/增 Go 绑定后 gen_bindings 再生→AppBindings 认领→legacyBindings 再生（顺序错炸 drift 锁）；发版门用 `tsc -b` 非 `--noEmit`。
- 守卫测试三件套（registrymap_census/space_assembly/registrymap_extra）须随 RegistryMap/spacetags 变更同步扩展。
- gen_bindings 生成件（bindings_wenxue.go）禁手改。
