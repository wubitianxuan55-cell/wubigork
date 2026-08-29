# 办公工作台前端调研评审报告（gaea 3.0 架构改造前置调研）

> 调研人：办公工作台前端专项（子代理） ｜ 日期：2026-08 ｜ 范围：frontend/src/gaea/（209 个文件 / 约 3 万行）
> 关联文档：docs/2026-08-15-gaea3-architecture-design.md（Step 1 事件日志 / Step 2 板块 Manifest / Step 3 Provider Seam）
> 方法：只读；glob/grep 建文件地图 → 精读 bridge.ts、store.ts、types.ts、App.tsx、useSessionManager、useModeManager、useBridgeWatch、useBridgeWatch、Transcript、ToolCard、ToolGroup、AskCard、WorkspacePanel、FileTree、FilePreview、ChangesPanel、Sidebar、mock 等；Go 侧仅交叉验证事件通道与会话存储（internal/app/gaea_handler.go、gaea_ui.go、agent/session/save.go）。
> 所有路径相对 C:\AI\wubigrok，行号以本次读取为准。

---

## 1. 概览（目录地图 + 文件职责）

### 1.1 目录地图（按文件数与职责分层）

```text
frontend/src/gaea/
├── App.tsx                    920 行  工作台装配根（布局/面板切换/快捷键/派生数据/抽屉）★唯一装配点
├── lib/                       核心纯逻辑层（16 个模块 + mock 子目录）
│   ├── bridge.ts             1221 行  Wails 绑定面代理 + 事件订阅 + 错误归一 + 编译期漂移检查 ★契约核心
│   ├── bindingNames.ts        467 行  由 scripts/gen_bindings -names 生成的 Go 绑定名清单（462 项，勿手改）
│   ├── types.ts              1066 行  WireEvent 等全部跨端契约类型（与 Go wire.go 镜像）
│   ├── store.ts               537 行  Zustand 状态机（useController：事件→items；useItems 细粒度订阅）
│   ├── changes.ts              65 行  写工具→「变更面板」汇总（纯前端派生，无后端依赖）
│   ├── tools.ts               215 行  工具卡展示逻辑（subjectOf/diffsFor/summarize/parseTodos）
│   ├── capabilities.ts         21 行  MCP/Skills 错误摘要纯函数
│   ├── session.ts              15 行  会话标题/时间格式化
│   ├── export.ts               29 行  会话导出 Markdown（exportAsMarkdown/downloadMarkdown）
│   ├── command.ts              16 行  Composer 指令分类（/model /memory 前端拦截）
│   ├── composer.ts             22 行  队列/工作区辅助
│   ├── stats.ts               128 行  统计聚合（aggSteps/colFromUsage/hitRateColor）
│   ├── diff.ts                 36 行  行级 diff（工具卡内联差异）
│   ├── projectGroups.ts        18 行  会话分组过滤
│   ├── layoutPreferences.ts   111 行  面板宽度 localStorage 持久化（多 key 合并 v1）
│   ├── mock.ts + mock/8 域    约1.1k  浏览器开发模式 mock（core/chat/memory/cost/office/retrieval/model/settings）
│   └── i18n.tsx + locales/     中英繁三语（en/zh/zh-TW）
├── hooks/                      12 个
│   ├── useSessionManager.ts   108 行  会话列表/分页/CRUD/项目分组
│   ├── useModeManager.ts       44 行  permLevel(ask/auto/yolo) + thinkLevel（思考深度）
│   ├── useBridgeWatch.ts       55 行  桥接心跳（5s 探测 Meta / 3s 超时）
│   ├── useTodoExtractor.ts     26 行  todo_write 待办提取
│   ├── useToolStats.ts         25 行  工具/技能计数
│   ├── useCapabilitiesData.ts  99 行  能力面板数据 hook（含 Reload 热加载）
│   ├── useSidebar.ts / useLayoutSizes.ts  侧栏宽度/折叠/预览模式持久化
│   └── useComposerWorkspace / useComposerMenus / useComposerAttachments / usePreviewProgress / useCompact / useDebouncedValue 等
└── components/                 面板与卡片体系（31 顶层 + 9 composer + 16 memoryhub + 3 capabilities + 7 测试）
    ├── Transcript.tsx         665 行  会话事件渲染核心（过程卡/工具卡/思考块/压缩卡装配）
    ├── Sidebar.tsx            886 行  会话列表（react-window 虚拟滚动）+ 记忆/能力/知识入口 + 事实底座
    ├── Composer.tsx           输入区（附件/截图/队列/工作区菜单/权限切换）
    ├── WorkspacePanel/FileTree/FilePreview/ChangesPanel   右侧工作区四件套
    ├── TodoPanel/TaskCenter/CommandPalette/JumpBar/StatsPanel/DeliverablesPanel/MaterialsPanel
    ├── MemoryPanel/HistoryPanel/CapabilitiesPanel/KnowledgePanel   四个懒加载抽屉
    ├── ApprovalModal/AskCard  审批弹窗 / 提问卡（含开工计划卡片 WirePlan）
    ├── ToolCard/ToolGroup/Message/ErrorCard/StreamingIndicator/CompactionCard 等卡片
    └── memoryhub/              记忆中枢聚合页（8 库：知识/成本/画像/办公记忆/轻语/资料/图谱/数字生命）
```

### 1.2 职责一句话

- **lib/bridge.ts**：唯一跨端接缝——真实 Wails 绑定（按方法名路由到 10 个门面）+ 浏览器 mock 回退 + 统一错误归一；类型级双向漂移校验（对齐 3.0 G5）。
- **lib/store.ts**：前端状态机，把 gaea-event 事件流约简为渲染用的 items[]（用户/助手/思考/工具/阶段/通知/压缩卡片），并持有 4 个轻量全局 UI store（预览/已更新文件/Composer 插入通道）。
- **lib/types.ts**：与 Go 侧 internal/serve/wire.go 镜像的契约类型，**单事件通道、kind 判别**（types.ts:1-2）——前端不感知 Go 内部事件结构。
- **App.tsx**：唯一装配根；无路由库，全部面板由 App 局部状态控制（rightTab/抽屉布尔/预览路径/宽度持久化）。
- **components/Transcript.tsx**：事件→UI 的最后一段——把 items 切成轮次、过程卡/大过程卡/工具卡/思考块的分段渲染。
- **lib/changes.ts**：**纯前端**从工具事件派生「变更面板」，Go 侧 GaeaWorkspaceChanges 实际返回空（见 5.3）。
- **hooks/useSessionManager + Sidebar**：会话生命周期（列表/恢复/归档/置顶/删除/重命名）的前端状态与虚拟滚动渲染。
- **lib/mock + mock/* 8 域**：浏览器开发模式的完整契约替身（187 方法全覆盖 + 可脚本化场景）。

### 1.3 与 3.0 五层板块清单的关系（前端侧）

3.0 设计文档 §3.1 指出「板块在五层各有各的清单」，本调研聚焦其中**办公板块（canonical id = gaea）**：导航页 GaeaPage、绑定面 OfficeB + MemoryB + CostB（前端经 bridge 门面路由统一消费）、意图域 gaea、服务域 gaea/office/knowledge/memory/cost/skill/tasks。前端侧现状：办公 UI 全部走 lib/bridge.ts 的 app 代理，**不直接触碰 window.go.app.OfficeB.*** 等门面名**——这为 Step 2 绑定面 Manifest 化提供了天然隔离点（前端只需改代理内的映射表，UI 调用零改动）。

---

## 2. bridge 契约与事件流（证据）

### 2.1 绑定面代理（bridge.ts）

- **AppBindings 接口**（bridge.ts:95-412）：187 个方法声明，是办公 UI 对 Go 侧的全部调用面。按域可分：会话/回退（101-141）、工作区（139-141）、成本/记忆中枢（304-388）、任务（386-390）、模型（225-226）、设置（248-282）、更新（279-282）、知识库（286-288）、聊天 chat 域（407-411）等。
- **gaeaToGaea 映射**（bridge.ts:503-682）：173 条「UI 短名 → Gaea 前缀绑定名」（Submit→GaeaSend、Approve→GaeaApprove、History→GaeaHistory…），注释明言历史上有 6 处映射错误（bridge.ts:886-888：KeepWarm*/PreloadPlan*/AgentMode/SummarizeFile/Subagent*）。对话 chat 方法（ChatTopicsList/ChatMessagesList/ChatAppendMessages）在 ChatB 门面上同名无前缀，不经映射（bridge.ts:499-500）。
- **realApp() 门面路由**（bridge.ts:438-454）：window.go.app 下枚举各门面命名空间（CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharLibB），按映射名找到函数并 bind 到该命名空间。**调用时解析而非模块加载时快照**，避免 dev 期 window.go 晚注入导致误锁 mock（bridge.ts:429-432，注释记载了「dev mock 模型列表泄漏进真实应用」的历史 bug）。
- **app 导出代理**（bridge.ts:742-756）：每次属性访问解析 realApp() ?? getMock()；除 LogFrontendError（错误上报通道自身，防递归，bridge.ts:751-753）外，所有方法统一走 invoke（bridge.ts:720-730）→ 失败归一为 BridgeError（bridge.ts:689-699，code/message 结构化 + message 可枚举）→ 写 gaea.log（logFrontendError，bridge.ts:734-738）→ 以 rejected promise 抛回调用方（原有 .catch 契约不变）。
- **initBridge()**（bridge.ts:855-871，幂等，App.tsx 模块作用域最早时机调用）：Wails 原生环境补 window.go.app.App 旧形态兼容代理（ensureLegacyAppProxy，bridge.ts:793-811，按方法名路由到各门面——**这是「门面拆分对旧调用点零影响」的落地先例**，3.0 设计 §3 层 2 引用）；HTTP/移动端模式（非 Wails）创建 /api/rpc 代理（rpcCall，bridge.ts:814-836；Bearer token 鉴权，getHttpToken）——3.0 设计 §2.2「httpbridge 保持调试桥定位」的前端对应物。

### 2.2 事件订阅清单（1 主通道 + 3 辅助通道）

| 通道 | 订阅函数 | 位置 | 触发方 | 前端行为 |
|---|---|---|---|---|
| gaea-event | onEvent | bridge.ts:418-424；常量 427 | Go sink（gaea_handler.go:79-85） | WireEvent → store.applyEvent（主对话流） |
| gaea-ready | onReady | bridge.ts:486-494 | boot.Build 完成 | 重拉 Meta/Context/History/Balance/Jobs/FactBase/TCCA |
| gaea-task | onTaskEvent | bridge.ts:476-482 | 任务调度器（价格抓取/索引重建） | TaskCenter 增量更新（TaskCenter.tsx:56-65） |
| updater:progress | onUpdaterProgress | bridge.ts:462-471 | 更新流程 | UpdateBanner 进度条 |

- 每个订阅在浏览器外回退到 mock（mockSubscribe/mockTaskSubscribe/updaterListeners，bridge.ts:423、481-482、467-470）。
- **Go 侧事件发射链**：internal/gaea/event.Event（类型化 Kind + Sink，单向流）→ controller 的 event.FuncSink（internal/app/gaea_handler.go:79-85）→ a.emit("gaea-event", gaeaEventMap(e))（gaea_handler.go:80）→ runtime.EventsEmit（internal/app/app.go:225）。事件在 Go 侧只经过一个转发点，前端订阅只认一个通道——这是「事件日志作事实源」改造的理想接入点（Step 1 若在 sink 处落日志，前端零改动）。
- **gaeaEventMap**（gaea_handler.go:387-488）把 event.Kind 翻译为前端 kind 字符串（gaeaKindName，gaea_handler.go:491-499）：turn_started/reasoning/text/message/tool_dispatch/tool_result/usage/notice/phase/approval_request/ask_request/turn_done/compaction_started/compaction_done；ToolDispatch/ToolResult 载荷字段（id/name/args/output/recoverable/truncated/partial/parentId/readOnly）与前端 WireTool（types.ts:36-47）逐一对齐；AskRequest 携带可选 plan（开工计划卡片数据，gaea_handler.go:461-477 ↔ types.ts:90-110）。

### 2.3 mock 回退机制

- 判定：realApp() 无 window.go.app 即 mock（bridge.ts:438-454）；mockSingleton 单例（bridge.ts:456-460）。
- 聚合入口 lib/mock.ts：makeMockApp()（mock.ts:40-52）由 buildCore/buildChat/buildMemory/buildCost/buildOffice/buildRetrieval/buildModel/buildSettings 8 个构建器组装（mock.ts:16-24），覆盖全部 187 个方法。
- 场景系统：?mock=fresh（空态）/running（模拟活跃流式）/demo（默认全量），?platform=darwin|windows|linux 覆盖平台检测（mock.ts:7-11）。
- **mock 的完整一轮对话模拟**（mock/chat.ts:24-64）：Submit → turn_started → reasoning 逐字 → tool_dispatch(ls) → tool_result → tool_dispatch(write_file) → tool_result → tool_dispatch(edit_file) → tool_result → text 逐字 → message → usage ×2 → turn_done；Cancel 置 cancelled 并补发 turn_done（mock/chat.ts:68-71）——与真实 Go 事件序列同构，是 UI 全链路（过程卡/工具卡/统计）离线可开发的基础。
- mock 域文件（office.ts）额外覆盖 Preview 分 kind 返回（image/docx/xlsx/markdown/text，mock/office.ts 内 grep 可见）与 WorkspaceChanges 空实现（mock/office.ts:289）。

### 2.4 编译期漂移检查（对齐 3.0 G5「注册表一致性由机器保证」）

- bindingNames.ts：go run ./scripts/gen_bindings -names 生成（bindingNames.ts:1-3）；462 个 Go 导出绑定方法名（字典序）；CI 由 scripts/check-bindings-drift.ps1 校验其与 Go 侧一致。
- 双向类型断言（bridge.ts:899-1220）：方向一 _CheckAppBindingsHasNoStray（AppBindings 映射后的每个方法必须存在于 bindingNames 或显式排除）；方向二 _CheckAppBindingsCoversAll（每个绑定名必须被消费或排除）。排除面 = MockOnlyNames（5 个：SetAgentMode/Compact/SetSubagentTemperature/SetEffort/SetSubagentModel，bridge.ts:913-918，其中后三个是 Go 侧从未实现的假声明）+ LegacySurfaceNames（约 280 个 wailsjsCompat 直调面：小说/聊天/语音/绘图/角色库等，bridge.ts:921-1205）。
- **结论**：办公前端绑定面已是「可机器校验的声明面」——Step 2 Manifest 化时「bindings 清单」的前端侧现成物；且 bridge.ts 的 AppBindings+gaeaToGaea+断言三件套可直接作为「代码板块」绑定面模板。

---

## 3. App 装配与面板体系

### 3.1 App.tsx 装配全景（920 行）

- **状态源**：useController()（App.tsx:63-95，解构 26 个控制函数）；useModeManager（97）；useSessionManager（100）；useBridgeWatch（234，onReconnect→refreshMeta 235-237）；useTodoExtractor（239）；useToolStats（472）。
- **布局**：antd Layout 三栏——Sidebar（603-631）｜ chat-pane（633-740：顶栏 ModelSwitcher/ContextBar/cwd 按钮/工具栏/Transcript/JumpBar/Composer/TodoPanel）｜ 右侧 workspace-pane（762-854）+ 主区域 preview-pane（743-760，宽度可拖 320-1100，layoutPreferences.ts:97-110）。
- **右侧面板 Tab 体系**（rightTab 状态，App.tsx:105）：files(WorkspacePanel) / materials(MaterialsPanel) / deliverables(DeliverablesPanel) / changes(ChangesPanel) / stats(StatsPanel) / tasks(TaskCenter)，Tab 按钮 764-807、渲染分支 809-851。
- **四个懒加载抽屉**：MemoryPanel（876-888）、HistoryPanel（892-900）、CapabilitiesPanel（904）、KnowledgePanel（909），均为 lazy+Suspense（App.tsx:23-26）。
- **浮层**：ApprovalModal（858-865，state.approval 驱动）、AskCard（867-873，state.ask 驱动，可拖拽）、CommandPalette（912-916，Ctrl+K）、预览弹窗通道重定向到嵌入式预览（158-168，usePreviewStore.subscribe）。
- **全局快捷键**（447-470）：Ctrl+N 新建、Ctrl+K 命令面板、Ctrl+Shift+H 历史、Ctrl+Shift+K 知识、Ctrl+B 侧栏、Ctrl+J 文件面板、Ctrl+Shift+F 专注模式、Esc 分层关闭（预览→抽屉→面板）。
- **派生数据**（全部 useMemo 纯前端）：sessionDeliverables（产物面板，App.tsx:475-492，从 assistant 文本用 deliverableMentions 提取交付文件路径）；sessionChanges（变更面板，App.tsx:495，buildSessionChanges）；currentSessionKey（以会话 .jsonl 文件路径为 localStorage key，506-514）；recentSessions（跨项目最近 6 条，336-348）。
- **任务目标（Kun 从需求到验收）**：首条用户消息自动捕获为 Requirement 并持久化（534-541），TodoPanel 展示 + 验收切换（543-547，712-719）。
- **专注/紧凑模式**：focusMode 收起侧栏与右栏（123-146）；compactMode 读写 localStorage（106）；两者都影响卡片密度与布局。

### 3.2 模式管理 useModeManager（44 行）

- permLevel：ask/auto/yolo 三级，UI 在 ComposerToolbar（composer/ComposerToolbar.tsx:20-21 标签、68-77 三档按钮），切换走 app.SetPermLevel（useModeManager.ts:18-21）→ GaeaSetPermLevel（bridge.ts:611）。
- thinkLevel：fast/normal/deep → THINK_TEMPS{fast:0.1,normal:0.3,deep:0.7}（useModeManager.ts:7）→ 读 Settings 后 SetAgentParams(temp, maxSteps, systemPrompt)（useModeManager.ts:23-32）。**注意：thinkLevel/handleThinkLevelChange 无任何组件消费（§6 缺陷 1）**——「思考深度」管理后端能力已具备（GaeaSetAgentParams），前端 UI 未接线。
- switchingModel：换模期间 App 显示 Skeleton（App.tsx:699-701）；换模经 useModeManager.switchModel → store.setModel（store.ts:505，SetModel 后重拉 Meta/Context）。

### 3.3 事件流渲染（useBridgeWatch → store → Transcript）

- **useBridgeWatch**（55 行）：每 5s 调 app.Meta()、3s 超时竞速判活（useBridgeWatch.ts:19-39）；断连→重连时触发 onReconnect 回调（App.tsx:235-237 注册 refreshMeta）。是「桥接存活」的轻量健康探针。
- **useController 事件消费**（store.ts:395-449）：onEvent 每事件 dispatch；text/reasoning 用 queueMicrotask 绕过 React 18 自动批处理保证逐 chunk 渲染（store.ts:400-401，注释解释同步 dispatch 会在同一微任务批量更新导致不渲染中间态）；turn_done 后补拉 Context/Balance/TCCAReport + reconcileFinalAnswer（store.ts:405-413）；turn_done/notice 后拉 Jobs + refreshFactBase（store.ts:414-417）；tool_result 的 fact_* 工具触发 FactBase 刷新（store.ts:418-420）；onReady 全量装载（store.ts:422-431）；**30s 看门狗**用 GaeaRunning 校准（store.ts:434-443，running 但后端已停 → localCancel + reconcile）。
- **applyEvent 约简**（store.ts:113-238）：13 种 kind 的完整映射（详见 §4.1）。
- **Transcript 分段渲染**（Transcript.tsx:157-182）：按 user 消息切轮（turns）；最后一轮运行中 → alternatingSegments（正文与过程卡交替、小过程卡默认折叠，Transcript.tsx:76-115）；已完成轮 → consolidatedSegments（思考/工具/中间正文折叠成大过程卡，最终正文留在外面，119-155）。**这正是 DSH「消息与过程交替、完成即折叠」的桌面实现**。渲染顺序保证 [用户问题]→[过程卡]→[最终输出]（148-153）。

### 3.4 右侧面板与抽屉逐项职责（数据源 + bridge 调用）

| 面板 | 数据源 | bridge 调用（grep 实证） | 关键行为 |
|---|---|---|---|
| WorkspacePanel（文件，App.tsx:810-818） | FileTree 懒加载目录 | ListDir（FileTree.tsx:76） | 点文件→主区域预览（openFilePreview，App.tsx:149-153）；refreshKey 强制重挂载 |
| MaterialsPanel（资料，819-821） | Materials/PinnedMaterials | Materials(120)/PinnedMaterials/UnpinMaterial/PinMaterial/SummarizeFile/OpenWorkspacePath（MaterialsPanel.tsx:48-55 等） | 按 docx/xlsx/pptx/pdf 分组；@ 引用插入 Composer（useComposerInsertStore）；钉住随会话 |
| DeliverablesPanel（产物，834-843） | 前端从 assistant 文本派生（App.tsx:475-492） | OpenWorkspacePath/RevealWorkspacePath（DeliverablesPanel.tsx） | 预览内编辑过的文件显示「已更新」（useUpdatedFilesStore）；一键沉淀成本库（DeliverablesPanel.tsx:53-58） |
| ChangesPanel（变更，844-850） | changes.ts 从 items 派生（App.tsx:495） | 无（纯派生） | 写/改文件汇总，点击预览；Go 侧 GaeaWorkspaceChanges 为空（5.3） |
| StatsPanel（统计，822-833） | store usage 累加 + localStorage | 无（纯消费） | useStatsPersistence 按会话 key 持久化（StatsPanel.tsx:9-27,48-90）；TurnRecord 逐轮 |
| TaskCenter（任务，851） | GaeaTaskList + gaea-task 事件 | TaskList/TaskCancel/TaskRetry（TaskCenter.tsx:47-53,67-75） | 价格抓取/索引重建实时进度；重启续跑 |
| MemoryPanel（抽屉，876-888） | GaeaMemory | 经 store.fetchMemory（store.ts:506）；写入走 Remember/Forget/SaveDoc/UpdateFact/ChangeFactType | 打开时拉快照、写后重拉；记忆建议（MemorySuggestions） |
| HistoryPanel（抽屉，892-900） | GaeaListSessions | 经 useSessionManager.refreshSessions | 日期分组/搜索/重命名/删除；恢复即整会话替换 |
| CapabilitiesPanel（抽屉，904） | useCapabilitiesData | Capabilities/Reload/AddMCPServer/RemoveMCPServer/RetryMCPServer/SetMCPServerEnabled | MCP 服务器 + 技能清单；Reload 热加载引擎（useCapabilitiesData.ts:26-39） |
| KnowledgePanel（抽屉，909） | GaeaKnowledge* | KnowledgeList/KnowledgeSearch/Get/Save/Delete/FindSimilar/History/Export/Review/Merge/PickFiles（KnowledgePanel.tsx） | 知识库 CRUD + 审核流 + 查重合并 |
| ApprovalModal（浮层，858-865） | store.approval（WireApproval） | Approve（store.ts:464） | allow/deny + session 级记忆；审批即点即清 |
| AskCard（浮层，867-873） | store.ask（WireAsk） | AnswerQuestion（store.ts:465） | 多问题/多选/自定义答案 + 开工计划卡片（AskCard.tsx:8-78）；可拖拽 |
| TodoPanel（底部，712-719） | useTodoExtractor + Requirement | 无（纯派生） | todo_write 待办 + 任务目标展示/验收 |
| CommandPalette（浮层，912-916） | App 内构造（551-575） | 无（动作回调） | 命令 + 最近会话快捷跳转 |
| JumpBar（主区，704） | items | 无 | 轮次跳转（scrollToTurnRef 机制，Transcript.tsx:434-444） |

---

## 4. 会话/工作区机制

### 4.1 事件→UI 数据流全链（Go → bridge → store → 卡片）

```text
Go 侧                                   bridge.ts                       store.ts                        UI
event.Event ──sink──> a.emit("gaea-event")        onEvent(cb)                applyEvent(e)             Transcript
(gaea_handler.go:79-85)  (gaea_handler.go:80)      (bridge.ts:418-424)        (store.ts:113-238)        (Transcript.tsx)
        │ gaeaEventMap(e)                              │                          │
        │ (gaea_handler.go:387-488)                     │ dispatch({type:"event"}) │ buildSegments
        ▼                                               ▼                          ▼
  kind:turn_started ──────────────────────────────────────────────► running=true/turnActive=true ──► ProcessCard(running)
  kind:reasoning ────────────────► queueMicrotask ──────────────► assistant.reasoning 追加 ──► InlineReasoning
  kind:text ─────────────────────► queueMicrotask ──────────────► assistant.text 追加 ───────► AssistantMessage
  kind:tool_dispatch ────────────► item {kind:"tool",status:"running"} ─► ToolCard(running) / ToolGroup
  kind:tool_result ──────────────► status done|error + output ────► ToolCard(展开 args/output/error)
  kind:usage ────────────────────► perTurnUsage/turnSteps 累加 ───► StatsPanel/状态栏 RunStatus
  kind:approval_request ─────────► state.approval ────────────────► ApprovalModal
  kind:ask_request ──────────────► state.ask（含 plan） ──────────► AskCard/计划卡片
  kind:notice/phase ─────────────► notice/phase item ─────────────► ErrorCard/notice/phase 行
  kind:compaction_* ─────────────► compaction item ───────────────► CompactionCard
  kind:turn_done ────────────────► 收尾（finalizeStaleTodos/usage 结算）► 大过程卡折叠 + RunStatus 复位
```

- **卡片体系**：ProcessCard（Transcript.tsx:216-334，头部「已工作 Ns · N 个工具 · N 段思考」+ GSAP 折叠）；ToolCard（ToolCard.tsx:36-163，图标表 tool_icons.ts + subjectOf/diffsFor/summarize 纯函数，lib/tools.ts:34-215）；ToolGroup（ToolGroup.tsx:12-62，连续同名工具合并成组，scanGroups ToolGroup.tsx:70-97）；Message/ErrorCard/CompactionCard（Transcript.tsx:645-665）。
- **工具卡数据还原**：写类工具内联 diff（diffsFor，tools.ts:60-88）、输出按行数统计（summarize，tools.ts:151-215）、文件路径可点开预览（FileLinkText）；subagent 调用按 parentId 嵌套（ToolCard.tsx:124-130，subcallsByParent 收集于 Transcript.tsx:464-474）。

### 4.2 useSessionManager（108 行）

- 状态：sidebarSessions（平铺列表，PAGE_SIZE=10 分页，useSessionManager.ts:5,32-47）、projectGroups（按工作区分组）、hasMore/loadMore（42-47）、sidebarQuery 搜索。
- 刷新：refreshSessions（32-40）同时刷平铺列表与项目分组（refreshProjectGroups 走 GaeaListProjectSessions，24-28）；allSessionsRef 缓存全量避免 loadMore 重复请求（22）。
- CRUD 回调（49-99）：startNewSession（newSession → 清搜索 → 刷列表 → 2s 成功提示）；handleResumeSession（失败 toast 到 App）；handleDeleteSession（乐观更新缓存 + 失败重拉）；handleRenameSession。
- 对接 bridge：listSessions/listProjectSessions/resumeSession/archiveSession/unarchiveSession/pinSession/deleteSession/renameSession 全部经 useController 包装（store.ts:468-488）。

### 4.3 会话恢复/归档/回退/任务目标

- **恢复链路**：UI（Sidebar/HistoryPanel/Welcome）→ handleResumeSession（useSessionManager.ts:57-67）→ resumeSession（store.ts:470-483）→ GaeaResumeSession → dispatch reset + dispatch history（rebuildHistoryItems 重建过程卡/变更面板，store.ts:62-83）；失败时注入 warn notice「恢复会话失败」而非静默清空（store.ts:471-478）。
- **跨项目恢复**：resumeSessionInProject（App.tsx:324-332）先 switchFolder(projectPath) 切工作区再恢复；recentSessions 欢迎页入口同链路（350-356）。
- **归档/置顶**：onArchiveSession/onPinSession/onRestoreSession（App.tsx:359-387）；Sidebar 行模型含「显示更多/已归档头/归档项」（Sidebar.tsx:67-74），react-window 虚拟滚动（SESSION_ROW_HEIGHT=44，Sidebar.tsx:55-60）。
- **回退/回滚**：rewind（store.ts:520）按 scope 分发 Fork / SummarizeFrom / SummarizeUpTo / Rewind，随后重新 History() + reset + Context；Checkpoints 列表供 UI 枚举（bridge.ts:113-114）——**全部基于整文件会话快照重放**（见 5.2）。
- **任务目标**：Requirement(path)/SetRequirement/SetRequirementDone（bridge.ts:134-136，store.ts:489-500）；.requirements.json 存于会话目录（gaea_ui.go:129,162）；首条消息自动捕获 + 验收勾选（App.tsx:534-547）。

### 4.4 工作区目录选择（GaeaPickWorkspace）

- bridge：PickWorkspace(): Promise<string>（bridge.ts:140，映射 GaeaPickWorkspace bridge.ts:534）；ListWorkspaces（139）/SwitchWorkspace（141）。
- App：switchFolder（App.tsx:312-320）——pick 成功后清预览、收起面板、刷新会话列表；顶栏 cwd 按钮触发（App.tsx:652）。
- store 层：pickWorkspace/switchWorkspace 成功后 dispatch reset + 重拉 Meta/Context（store.ts:502-503）。
- Composer 内工作区菜单走 useComposerWorkspace → ListWorkspaces（hooks/useComposerWorkspace.ts）。
- 后端视角：gaeaBuildController 的 SessionDir = cwd/.gaea/sessions（gaea_handler.go:90-96），工作区切换即控制器重建，**会话目录随工作区走**——这是「项目分组（ProjectGroup）」数据模型的前端依据（GaeaListProjectSessions 聚合，types.ts:169-177）。

---

## 5. 与 3.0 目标相关的关键发现

### 5.1 已与 DSH Web GUI 同构的机制（可直接对标，无需改造）

1. **单事件通道 + kind 判别的事件流渲染**：gaea-event 一个通道承载全部事件（bridge.ts:418-427），kind 判别 payload（types.ts:4-19）——与 DSH SessionEventMap + client 单通道订阅同构。**本次调研最重要的同构点**：事件是前端 UI 的唯一增量输入，state.items 全部由事件约简而来。
2. **过程卡/工具卡/思考卡**：Transcript 轮次切分 + ProcessCard/ToolCard（Transcript.tsx:157-334；ToolCard.tsx:36-163）等价于 DSH 的 turn 分组渲染；工具调用有 parentId 子调用嵌套（types.ts:46，ToolCard.tsx:124-130）对应 subagent 树。
3. **变更面板 = 纯事件派生**：buildSessionChanges（changes.ts:50-65）只扫 items 里 WRITE_TOOL_NAMES 的 tool 事件并解析 args 路径（extractChangedPaths，changes.ts:22-47，与 Go evidence.go extractPaths 对齐）——**没有任何后端变更追踪 API 参与**；Go 侧 GaeaWorkspaceChanges 干脆返回空（gaea_ui_extra.go:616-617「办公板块不追踪工作区变更，返回空」）。与 DSH「变更面板从日志派生」完全同思路——3.0 设计 §5.1 声称的「日志带来的新增收益：变更面板」其实前端已自实现。
4. **usage 事件累加 → 统计面板**：store.ts:192-222 按 source 拆分 executor/subagent 累加 perTurnUsage/perTurnExecutorUsage/perTurnSubUsage/turnSteps；StatsPanel 按会话路径持久化 localStorage（StatsPanel.tsx:9-27,48-90）。Token/成本派生在前端已存在（与 3.0 日志派生统计同目标、不同数据源）。
5. **事件丢失防御三件套**：看门狗 30s 校准（store.ts:434-443）+ reconcileFinalAnswer 补发最终回答（store.ts:370-389）+ localCancel 本地复位（store.ts:246-253）——前端已把「事件流不可靠」当一等公民处理，这是 DSH 桌面端可借鉴的健壮性模式。
6. **模板/产物/资料/待办面板**：Welcome 拉 TaskTemplates（Welcome.tsx）、DeliverablesPanel 从 assistant 文本提取交付（App.tsx:475-492）、TodoPanel 从 todo_write 解析（useTodoExtractor）——都是事件/文本的纯派生视图，无需后端新 API。

### 5.2 依赖整文件会话 JSONL 快照的前端面（Step 1 事件日志改造的波及面）

Step 1 把 session.Save/Load 改为 append-only 事件日志 + 检查点/投影时，以下前端机制**直接读 GaeaHistory（controller 内存 Messages 的整快照视图）**，是重点回归面：

| # | 前端机制 | 位置 | 依赖方式 |
|---|---|---|---|
| 1 | 启动装载 | store.ts:353-364（loadSessionData）、422-431（onReady） | History() 全量重建 items |
| 2 | 恢复会话 | store.ts:470-483 | GaeaResumeSession 全量返回 → reset+history |
| 3 | 回退/回滚 | store.ts:520 | Rewind/Fork/Summarize* 后重拉 History |
| 4 | 最终回答兜底 | store.ts:370-389 | 轮询 History 比对最后一条 assistant 正文 |
| 5 | 工具/过程卡还原 | gaea_ui.go:93-126 + store.ts:62-83 | GaeaHistory 展开 assistant.ToolCalls 为 tool 条目、tool_result 独立成条；rebuildHistoryItems 按 toolId 合并 |
| 6 | 统计口径 | store.ts:231（sessionTotal 内存累加） | 恢复/换会话即清零；「全会话」只含本次前端窗口 |
| 7 | 会话元数据 | session/save.go:72-78（Info）→ GaeaListSessions | title/preview/turns 来自 JSONL 扫描；归档/置顶/删除按 path 操作（bridge.ts:120-136） |

**关键风险点**：GaeaHistory 输出格式（role:"tool"/"tool_result" + toolName/toolArgs/toolId）是前端 rebuildHistoryItems 的唯一还原依据（store.ts:62-83）——事件日志改造后**必须保持该输出对同一会话逐字节不变**（3.0 设计 §5.1 验收标准「GaeaHistory 输出与改造前逐字节一致」），否则恢复后的过程卡/变更面板/工具卡全部跑偏。前端侧无需预改动，但建议把「恢复后工具卡/变更面板内容一致性」加入 Step 1 前端回归用例。

### 5.3 可复用为未来「编程板块」代码工作台的组件

3.0 设计 §9 已声明「编程 = 办公引擎 + 模式预设 + manifest 新页面，文件树/对话/预览/变更组件复用」，本调研确认复用面是真实的：

- **Transcript + ProcessCard + ToolCard + ToolGroup + Message + ErrorCard**：agent 轮次渲染，无任何办公专属依赖——编程板块的对话区直接复用。
- **ChangesPanel + changes.ts**：写工具聚合逻辑通用（write_file/edit_file/edit_lines/multi_edit 正是编程工具，changes.ts:10-13）。
- **FileTree + WorkspacePanel + FilePreview**：通用工作区浏览（ListDir/Preview 契约与语言无关）；FilePreview 当前只读 + docx/xlsx 编辑（FilePreview.tsx:115-151），编程板块需补「代码高亮 + 保存回写」能力（对标 aionui-panel 的分屏编辑/保存）。
- **TodoPanel + useTodoExtractor**：todo_write 待办卡通用。
- **ApprovalModal / AskCard / CommandPalette / JumpBar / StatsPanel / Composer / Sidebar 会话列表**：权限、提问、快捷、统计、输入、会话管理全部与板块无关。
- **useController/store.ts + useItems**：事件状态机即通用 agent UI 内核；bridge.ts 的「AppBindings + 门面代理 + 双向断言」可直接作为代码板块绑定面模板（或演进为 manifest 数据驱动，见 §7 建议 5）。
- **lib/tools.ts 的 subjectOf/diffsFor/summarize**：工具展示词汇表，编程工具（bash/grep/glob/read_file）已有覆盖（tools.ts:34-54），新增工具只需加 switch 分支。

### 5.4 与 3.0 机制逐条对照小结

| 3.0 目标 | 前端现状 | 结论 |
|---|---|---|
| G2 会话可回放/派生 | 恢复=GaeaHistory 整快照重建；标题/统计/成本=内存+前端派生 | Step 1 投影层接口不变则前端零改动；若新增稳定会话 id，前端统计/需求 key 可顺切 id |
| G4 装配点唯一 | 前端装配点已唯一（App.tsx）；但面板/抽屉/派生数据全部塞在 App 一个组件 | Step 2 前端改动集中在全局导航装配，办公页自身可保留；办公页内 Tab 清单化可作为预演（§7 建议 3） |
| G5 注册表/绑定一致性 | bridge.ts 双向断言 + CI 已就位 | 是 manifest「bindings」校验的现成前端先例 |
| Provider Seam（Step 3） | 前端经 Models/SetModel/SetAgentParams/SetEffort/SetSubagent* 直调 | 前端零感知——seam 化只需后端内部改造（bridge provider 已存在），前端契约不变 |

---

## 6. 缺陷与风险

1. **thinkLevel（思考深度）UI 未接线**：useModeManager 声明并实现 handleThinkLevelChange（useModeManager.ts:15,23-43），但全 frontend/src 无组件消费（grep 仅命中 hook 自身）；App.tsx:97 只解构 permLevel/setPermLevel/switchingModel/switchModel。「思考深度」管理实际只有后端能力（SetAgentParams 温度），前端入口缺失——任务描述与现状不符，属待办缺口。
2. **恢复会话丢失部分状态**：rebuildHistoryItems 把工具状态硬编码为 status:"done"（store.ts:76-78），partial 标志（types.ts:45）不还原；工具调用只有 args 无 output 时卡片空洞。运行中会话被看门狗 localCancel 后保存，恢复后「中断」状态仅靠 SessionMeta.interrupted 提示（types.ts:164-165，Sidebar.tsx:64-65 的过渡期类型断言）。
3. **两套「变更」数据源并存且后端为死代码**：前端 changes.ts 派生 vs GaeaWorkspaceChanges（返回空，gaea_ui_extra.go:616-617；mock/office.ts:289 同步空实现）——后端绑定是死的（bindings_office.go:136），建议 Step 1 用日志派生 API 替换或删除，消除双源（bridge.ts:213-214 同步清理）。
4. **App.tsx 单文件膨胀**：920 行承载所有装配/派生/快捷键/抽屉状态；新增右侧 Tab 需同时改 rightTab 联合类型（App.tsx:105）、Tab 按钮（764-807）、渲染分支（809-851）、CommandPalette 项（551-575）——**与 3.0 批判的「加板块改 6 处」同类问题，只是缩小在办公页内**（页面级装配点唯一，面板级装配仍散）。
5. **事件可靠性依赖重**：看门狗 + reconcileFinalAnswer + queueMicrotask 三件套说明 Wails 事件流有丢事件/批处理问题（store.ts:366-369 注释直说「最终回答只有重启才可见」的 bug 类）；3.0 事件日志改造不会消除 UI 侧这一层，前端防御代码需保留（且 Step 1 回归应含「事件丢失→UI 自愈」场景）。
6. **统计/需求按会话文件路径做 localStorage key**：currentSessionKey 用路径字符替换（App.tsx:510-513），工作区移动/重命名即丢统计；capturedReqPathRef 用路径判断「已捕获」，路径变化会重复捕获（App.tsx:520-541）。Step 1 若提供稳定会话 id（日志文件 id），前端应改用 id。
7. **gaeaToGaea 手维护映射**：173 条映射靠人肉同步（bridge.ts:503-682），虽有两向断言兜底（bridge.ts:1212-1220），但每次新增绑定要动 3 处（AppBindings 声明 + 映射 + mock）；历史已有 6 处映射错误（bridge.ts:886-888）。Step 2 若 manifest 化绑定清单，可消掉映射层（§7 建议 5）。
8. **mock 与真实契约漂移风险**：mock 域文件分散（8 个），mock/office.ts 的流式事件只覆盖 text/preview 类（grep 显示），完整一轮模拟在 mock/chat.ts（含 tool_dispatch/tool_result/usage）；若 mock 未与 WireEvent 全 kind 对齐，浏览器开发会漏测审批/提问/压缩卡路径（approval/ask 场景无 URL 场景开关）。
9. **Drawer 状态分散**：memView/histView/capsOpen/knowledgeOpen 四个布尔在 App 手管（App.tsx:98-104），懒加载 + Suspense 依赖条件渲染；无统一 Drawer 管理器（虽有 ResizableDrawer 组件复用，HistoryPanel.tsx:59 使用）。
10. **wailsjsCompat 双面**：LegacySurfaceNames 约 280 个方法不经 AppBindings 消费（bridge.ts:921-1205），办公板块的代理只认 Gaea*；若 3.0 绑定面拆分/改名，LegacySurfaceNames 清单需同步维护，容易漏（当前靠类型断言兜底）。
11. **会话恢复后的统计/成本即时性**：StatsPanel 的「全会话成本」依赖 usage 事件在本次前端窗口内累计（store.ts:231），恢复旧会话不回溯历史 usage——成本面板对「恢复的长会话」展示不完整（3.0 日志派生统计可根治）。
12. **前端测试面**：lib 层有 bridge.test.ts/store.test.ts/mock-contract-t63.test.ts/mock-contract-e5.test.ts/changes.test.ts 等，组件层有 ProcessCard/ToolCard/AskCard/DeliverableCards/MaterialsPanel/KnowledgePanel 等测试——但 App.tsx 装配层无测试（920 行零覆盖），Step 2 清单化改造前建议先补装配快照测试（3.0 设计 §5.2 验收标准「像素级回归」的前端落点）。

---

## 7. 改造建议

1. **Step 1 前端配合面（建议纳入验收清单）**：以「GaeaHistory 输出逐字节不变」为回归锚点（3.0 设计 §5.1）；前端零代码改动的目标可行——GaeaResumeSession/History/Rewind 的返回结构不变，rebuildHistoryItems（store.ts:62-83）无感。若日志化后 SessionMeta 增加稳定 id，顺手把统计/需求 key 切到 id（修缺陷 6、11）。
2. **变更面板数据源统一（缺陷 3）**：前端派生已自洽，建议 3.0 用日志派生 API 替换后端空实现 GaeaWorkspaceChanges，或直接移除绑定——三处同步删（bridge.ts:213-214、mock/office.ts:26,289、bindings_office.go:136）。变更面板本身不动。
3. **Step 2 前端清单化试点放在办公页内（缺陷 4）**：把右侧面板体系（rightTab + 各面板 props，App.tsx:762-854）声明化为轻量 manifest（id/组件/入口），既是 3.0 全站 MainLayout 清单化的预演，也治 App.tsx 膨胀。注意与全局 BoardManifest 的边界：办公页内 Tab 清单与跨板块导航清单是两层，3.0 设计 §5.2 只覆盖后者。
4. **接线思考深度 UI（缺陷 1）**：ComposerToolbar 已有权限三段按钮（ComposerToolbar.tsx:68-77），同模式加 fast/normal/deep 三档，走现成 handleThinkLevelChange（useModeManager.ts:23-32）——小改动即可兑现「思考深度管理」。
5. **bridge 映射层演进（缺陷 7）**：Step 2 若做绑定面 manifest，gaeaToGaea 可退化为「manifest bindings 列表 + Gaea 前缀规则」——173 条映射中非 Gaea 前缀仅 KeepWarm*/PreloadPlan*/AgentMode/SummarizeFile 等少量特例（bridge.ts:677-680），规则化后砍掉手维护表；两向断言保留为 manifest 一致性自检（对齐 G5）。
6. **mock 契约补全（缺陷 8）**：为审批卡/提问卡/压缩卡补浏览器 mock 场景（?mock=approval|ask|compaction），使事件流 UI 全 kind 可离线开发——复用现有场景系统（mock.ts:7-11）与 mock/chat.ts 的发射模式。
7. **复用面固化（5.3）**：把「可复用为编程板块」的组件清单写进 3.0 设计文档 Step 2 试点计划，避免二期重新评估；FilePreview 增加只读代码高亮 + 保存回写（对标 aionui-panel 预览/编辑分屏），为编程板块预留能力。
8. **抽屉管理器收敛（缺陷 9）**：用一个 useDrawers 合并 memView/histView/capsOpen/knowledgeOpen 四个布尔 + Esc 分层路由（App.tsx:452-458 已有 Esc 分发雏形），减少 App 状态面。
9. **事件可靠性三件套保留并测试（缺陷 5）**：看门狗/reconcile/localCancel 是事件流不丢的 UI 侧保险，Step 1 后保留；建议把「turn_done 丢失→看门狗自愈」场景写进前端 vitest（store.test.ts 已有 applyEvent 覆盖基础）。
10. **装配层补测试（缺陷 12）**：App.tsx 无测试，Step 2 前补「右侧面板渲染 + 快捷键 + 派生数据」快照/交互测试，承接 3.0 设计 §5.2「无 manifest 信息遗漏时导航渲染与现状一致」的验收。
11. **不引入新状态库**：Zustand + 局部 useState 的现状足够；3.0 前端侧不需要为 manifest 引入运行时 DI，保持「App 装配 + 数据驱动清单」即可（与 3.0 设计 §2.2「不移植 Cordis」一致）。
12. **契约演进通道**：bridge.ts 的 types.ts 与 gaeaEventMap（gaea_handler.go:387-488）是前后端契约的两个镜像端点——Step 1 增加事件种类（如 turn_aborted/steer）时，两端同步修改 + 编译断言（前端）+ 单测（Go）兜底，这个既有机制应写进 3.0 的事件演进流程。

---

## 附：关键证据索引（文件:行）

- 绑定代理：frontend/src/gaea/lib/bridge.ts:95-412（AppBindings）、438-454（realApp）、503-682（gaeaToGaea）、689-715（BridgeError/normalizeError）、720-756（invoke/app）、855-871（initBridge）
- 事件订阅：bridge.ts:418-494（onEvent/onUpdaterProgress/onTaskEvent/onReady）、427（EVENT_CHANNEL）
- 漂移检查：bridge.ts:899-1220；bindingNames 生成：lib/bindingNames.ts:1-3
- 状态机：lib/store.ts:16-22（Item）、113-238（applyEvent）、62-83（rebuildHistoryItems）、246-253（localCancel）、348-523（useController）、530-532（useItems）
- 契约类型：lib/types.ts:4-19（EventKind）、36-47（WireTool）、90-110（WireAsk/WirePlan）、117-130（WireEvent）、133-141（HistoryMessage）、152-166（SessionMeta）
- 装配：App.tsx:63-100（hooks）、105（rightTab）、312-320（switchFolder）、447-470（快捷键）、475-495（派生）、506-514（sessionKey）、762-854（右栏）、858-916（浮层/抽屉）
- 事件→UI：components/Transcript.tsx:157-182（buildSegments）、216-334（ProcessCard）；ToolCard.tsx:36-163；ToolGroup.tsx:70-97（scanGroups）；AskCard.tsx:8-78（PlanCard）
- 变更派生：lib/changes.ts:10-65；后端空实现：internal/app/gaea_ui_extra.go:616-617
- Go 事件链：internal/app/gaea_handler.go:79-85（sink）、387-488（gaeaEventMap）、491-499（kind 映射）；internal/app/app.go:225（EventsEmit）
- 会话存储：internal/gaea/agent/session/save.go:19-38（整文件重写 JSONL）、72-78（Info）；GaeaHistory：internal/app/gaea_ui.go:93-126
- mock：lib/mock.ts:40-52；场景参数 mock.ts:7-11；一轮模拟：lib/mock/chat.ts:24-64
- 权限 UI：components/composer/ComposerToolbar.tsx:20-21,68-77；思考深度缺口：hooks/useModeManager.ts:7,15,23-43
