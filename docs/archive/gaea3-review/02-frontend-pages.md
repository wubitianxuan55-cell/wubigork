# 02 前端业务页面层调研报告（gaea 3.0 架构改造评审）

> 调研范围：frontend/src/pages/ 全部页面（ChatPage/NovelPage/ImageGenPage/GaeaPage/MemoryHubPage/ModelCenterPage/CharacterLibraryPage/SettingsPage/HomePage/KnowledgePage/ChapterPage/CharacterPage/CreatePage/ExportPage/NovelSettingPage + pages/chat、pages/modelcenter 子目录）与 frontend/src/components/ 顶层业务组件（chat/、novel/、office/、imagegen/、settings/、characterlib/ 目录与 AppBar/ModuleLauncher/WelcomePage/PersonaPicker 等散件）。只读调研，未修改任何代码。壳层（MainLayout/appStore/bridge）结论见姊妹报告 01-frontend-shell.md，本文只补充页面层证据。

## 1. 概览（每板块文件清单+职责）

### 1.1 页面入口文件（frontend/src/pages/）

| 文件 | 行数 | 职责 |
|---|---|---|
| ChatPage.tsx | 407 | 聊天板块编排层：装配 useChatTopics/useChatStream/useChatVoice + chat/ 展示组件 |
| NovelPage.tsx | 86 | 小说板块壳：6 个子页 tab 切换（书架/设定/角色/创作/阅读/导出）+ 右侧 AIConsole + FeatureModelBar |
| ImageGenPage.tsx | 333 | 绘梦板块编排层：4 个 hook（config/queue/history/templates）装配 7 个展示组件 |
| GaeaPage.tsx | 25 | 办公板块薄壳：LocaleProvider 包 gaea/App（gaea/App.tsx 920 行，Codex 式工作台） |
| MemoryHubPage.tsx | 325 | 记忆中枢：首页 + 8 个库面板（knowledge/cost/profile/office/materials/whisper/graph/digitallife） |
| ModelCenterPage.tsx | 258 | 模型中心：10 个分类 tab + ModelCenterContext + 5 个分类 hook |
| CharacterLibraryPage.tsx | 371 | 全局角色库：档案墙/搜索/分页/编辑/记忆弹窗 |
| SettingsPage.tsx | 178 | 设置中心：9 个分类磁贴 + 关键字搜索过滤 |
| HomePage.tsx | 200 | 小说书架（NovelPage 子页 home tab，未登录时渲染 WelcomePage） |
| KnowledgePage.tsx | 29 | **孤儿页面**：独立挂载 KnowledgePanel(variant="page")，未注册任何导航 |
| ChapterPage.tsx | 288 | 小说「阅读」子页：大纲树 + 章节 Tab 编辑 |
| CharacterPage.tsx | 783 | 小说「角色」子页：角色/组织/关系三 Tab，单向引用角色库 |
| CreatePage.tsx | 288 | 小说「创作」子页：章节生成编排（流式事件 + 分支向导） |
| ExportPage.tsx | 94 | 小说「导出」子页：一键导出 TXT/MD/EPUB |
| NovelSettingPage.tsx | 240 | 小说「设定」子页：Markdown 世界观编辑器 + 设定 Agent 对话 |
| pages/chat/{constants,types,utils}.ts | 25/20/98 | 聊天板块拆分产物：localStorage 键常量/消息类型/纯工具 |
| pages/modelcenter/ 19 个文件 | — | 模型中心拆分产物：12 个 Section + 5 个 hooks + context/utils/charts/resource/ui |

### 1.2 顶层业务组件目录（frontend/src/components/）

| 目录/文件 | 数量 | 职责 |
|---|---|---|
| chat/（6 文件） | 6 | 聊天纯展示组件：MessageList/ChatComposer/WelcomeScreen/SuggestionCard/ChatModeBar/ChatPersonaBar（零 App 直连，全部 props 驱动） |
| novel/（~30 文件） | 30 | 小说组件：AIConsole/ChapterEditor/OutlinePanel/CreateNovelModal/NextChapterModal/PlotBranchModal + create/（ChapterTreePanel/EditorPanel/CreateInspector/BranchWizardModal/NewCharactersModal/useChapterStream）+ editor/（DiffReview/CommandBar/GhostText）+ character/（CharacterCard/OrganizationCard/RelationshipModal/OrganizationEditModal/PortraitLightbox）+ api/（outlines.ts/character.ts）+ hooks/usePlotBranch |
| office/（2 文件） | 2 | ParseSummaryCards/SourcePreviewDrawer（办公解析摘要展示，纯展示） |
| imagegen/（~15 文件） | 15 | 绘梦组件：ControlPanel/ResultStage/GenerationBar/TaskCenter/HistoryRail/CustomTemplateModal/TemplatePickerModal + 纯函数（meta/media/queue/historyMeta/ui） |
| settings/（9 文件） | 9 | 设置面板：AppearancePanel/ChatPanel/WorkspacePanel/ImageGenPanel/OfficePanel/ModelPanel/SecurityPanel/DataPanel/AboutPanel |
| characterlib/（~8 文件） | 8 | 角色库组件：CharacterCard/CharacterLibEditor/CharacterMemoryModal/PortraitImg + css |
| 散件 | ~25 | AppBar/ModuleLauncher/WelcomePage/PersonaPicker/FeatureModelBar/ChatTopicSidebar/ChatPanel/SearchModal/VoiceSettingsPanel/SecurityBanner/ProjectCardItem/TTSPlayer/SkillModal/RelationGraph/Lightbox/CompanionAvatar/VoiceChatOrb/Whisper* 记忆组件等 |

## 2. 各板块组件树与绑定调用面（证据）

### 2.1 聊天板块（ChatPage）

**组件树**：ChatPage → ChatTopicSidebar（左侧会话栏，ChatPage.tsx:281）｜ main 区 → ChatModeBar（:297）/ ChatPersonaBar（:316）/ MessageList（:350）或 WelcomeScreen（:337）/ ChatComposer（:372）；悬浮 FeatureModelBar(feature="chat")（:394）；Modal 内 VoiceSettingsPanel（:398-401）。PersonaPicker 被 ChatModeBar（ChatModeBar.tsx:56）与 ChatPersonaBar（ChatPersonaBar.tsx:33）复用。

**状态管理**：页面本地 useState（ChatPage.tsx:36-59 持 messages/input/personalities 等）+ 3 个 hook：useChatTopics（话题/模式/人格元数据状态机）、useChatStream（流式/模拟打字发送状态机）、useChatVoice（语音集成）。localStorage 键集中定义 pages/chat/constants.ts:3-15。**无 zustand/Context**——聊天状态完全页面私有。

**后端绑定调用清单**（经 wailsjsCompat）：
- useChatTopics.ts：ChatTopicsList(:76/82/89)、ChatImportTopic(:81)、ChatTopicCreate(:86/126)、ChatTopicDelete(:140)、ChatTopicRename(:155/183)、ChatTopicSetMode(:171)、ChatMessagesList(:47)、WhisperGetPersonalities(:109)
- useChatStream.ts：ChatStreamPlain(:120，plain 模式流式)、ChatSend(:164，角色模式整段返回)；订阅 runtime 事件 `chat-stream:<runID>`（:123，动态频道）
- useChatVoice.ts：ChatAppendMessages(:25，语音消息落库)、VoiceApplySettings(:47)
- ChatPage 本体：WhisperClearSession(:171/257)、TTSSpeakBase64(:227，朗读)、ChatTopicClear(:253)、ChatTopicExportMarkdown(:264，导出)
- 事件订阅：model-changed/voice:* 等见姊妹报告 §4.1 的 useVoiceChat（frontend/src/hooks/useVoiceChat.ts）

**关键机制**：plain 模式先订阅后收帧（useChatStream.ts:104-121，30s 无帧超时 STREAM_SILENCE_TIMEOUT_MS=30_000，constants.ts:19）；重试/语音落库（ChatAppendMessages）；旧 localStorage 话题一次性迁移（chat/utils.ts:56-98）。

### 2.2 小说板块（NovelPage 壳 + 6 子页）

**组件树**：NovelPage（tab 壳，NovelPage.tsx:47-83）→ 6 个 lazy 子页（HomePage/NovelSettingPage/CharacterPage/CreatePage/ChapterPage/ExportPage，:10-15，子页常驻挂载 display:none 切换 :69-73）＋ 右侧 AIConsole（:76）＋ FeatureModelBar(feature="novel")（:80）。二级 tab 本地持久化（NOVEL_TAB_KEY，:18）。

- **HomePage**（书架）：window.go.app.App.CreateProject(:65)/CloseProject(:80)/OpenProject(:86)；经 appStore（loadProjects/openProject/deleteProject 等，HomePage.tsx:15-18）
- **NovelSettingPage**：App.GetWorldview(:45)/SaveWorldview(:71)/ChatWorldview(:121，设定 Agent)；导入/导出走本地 Blob（:96-117）
- **CharacterPage**（小说角色）：api/character.ts 的 getCharacters/saveOrganization/deleteOrganization/toggleOrgMember/saveRelationship/deleteRelationship/generateCharacterFill/generateCharacterPortrait/mergeCharacters（character.ts:17-63）+ api/characterlib.ts 的 listProjectCharacters/associateToProject/dissociateFromProject/syncProjectCharacters/importProjectCharacters/drawRandom/setProjectState（CharacterPage.tsx:36-40）；自身 navigateToCharacterLib 函数分发 navigate 事件（:76-77）；订阅 gaea-project-chars-changed 刷新（:188-192）
- **CreatePage**（创作）：App.GetWorldview(:54)/GetStats(:63)/GetChapterBranch(:81)/QuickBrainstormBranches(:98)/CreateChapter(:166)/CancelCreateChapter(:187,195，经类型桥接)/DeleteOutlineNode(:221)/SaveChapterBranchContent(:236)；订阅 create-chapter-stream 流式事件（useChapterStream.ts:46）；共享 outlineStore
- **ChapterPage**（阅读）：GetChapter/GetChapterBranch(:73/110)、SaveChapterContent/SaveChapterBranchContent(:170-172)；TTSPlayer/OutlinePanel/ChapterEditor；读/写阅读进度（readingProgress.ts，:64-95）
- **ExportPage**：App.ExportAll(:16)

**novel/ 组件绑定**：outlines.ts:17 GetOutlines；CommandBar.tsx:70 CmdKEdit（框选即改）；usePlotBranch.ts:36/48 BrainstormBranches/ApplyBranch；AIConsole 订阅 xai-output 事件（AIConsole.tsx:62）。

**状态管理**：页面本地 useState + **outlineStore（zustand 共享）**——CreatePage（:21-22）与 ChapterPage（:36-37）共用同一份大纲；其余子页状态私有。

### 2.3 绘梦板块（ImageGenPage）

**组件树**：ImageGenPage → 左 ControlPanel（:212）/ 中 ResultStage（:244）/ 右 TaskCenter（:257）/ 底 GenerationBar（:274）/ Lightbox（:294）/ CustomTemplateModal（:307）/ TemplatePickerModal（:320）。

**状态管理**：4 个 hook 全本地——useImageGenConfig（配置/引擎/后端切换）、useImageGenQueue（生成队列/任务状态机，queue.ts 纯函数）、useImageGenHistory（历史/灯箱）、useCustomTemplates。**无 store/Context**。

**后端绑定清单**（经 api/image.ts 封装）：GetImageBackendInfo(:49)、GetCharacters(:56)、GetComfyUIStatus(:66)、GetComfyUILoras(:78)、GetSystemStats(:88)、GetPortraitConfig(:98)/SetPortraitConfig(:107)、GenerateFreeImage(:116)、CancelImageGeneration(:132)、GaeaAttachmentDataURL(:140)、GetComfyUITaskProgress(:150)、GenerateDiagram(:158)、GenerateMedia(:182)、StartComfyUI(:204)/StopComfyUI(:209)、OpenImageSaveDir(:214)/OpenNovelImagesDir(:219)、SetCharacterPortrait(:224)。useImageGenConfig 另经 api/settings.ts 的 getImageBackendInfo/setImageBackend 与 api/engines.ts 的 getEngines。

### 2.4 办公板块（GaeaPage → gaea/App）

**组件树**（gaea/App.tsx）：Sidebar（会话列表/记忆/历史/能力/知识入口，:603-631）｜ 聊天区 Transcript(:703) + JumpBar(:704) + Composer(:727) + TodoPanel(:713) + RunStatus(:720) ｜ 右侧面板 6 tab（文件 WorkspacePanel/资料 MaterialsPanel/产物 DeliverablesPanel/变更 ChangesPanel/统计 StatsPanel/任务 TaskCenter，:762-854）｜ 浮层：ApprovalModal(:859)/AskCard(:868)/MemoryPanel(:876)/HistoryPanel(:893)/CapabilitiesPanel(:904)/KnowledgePanel 抽屉(:909)/CommandPalette(:912)/FilePreview(:753)。

**状态管理**：zustand useStore（gaea/lib/store.ts:281）+ 事件驱动 reducer（applyEvent，:113-238 处理 turn_started/text/reasoning/message/tool_dispatch/tool_result/usage/notice/phase/approval_request/ask_request/turn_done）；onEvent 订阅 gaea-event 频道（store.ts:396-421，bridge.ts:420）。辅助 store：usePreviewStore/useComposerInsertStore/useUpdatedFilesStore（store.ts:290-334）。**办公板块是唯一"事件驱动 + 合成状态机"的板块——与 DSH 会话事件日志思想同构，是 Manifest 化的样板。**

**绑定调用面**：全部经 gaea/lib/bridge.ts 的 app 代理（AppBindings 接口约 150 方法，bridge.ts:95-412），短名→Gaea* 前缀映射（gaeaToGaea，bridge.ts:503-682），如 Submit→GaeaSend、History→GaeaHistory、KnowledgeList→GaeaKnowledgeList。办公专属 hooks：useSessionManager/useModeManager/useSidebar/useCompact/useComposerMenus/useComposerWorkspace/useCapabilitiesData/useToolStats/useTodoExtractor/useBridgeWatch/usePreviewProgress/useUpdater（gaea/hooks/）。

### 2.5 记忆中枢板块（MemoryHubPage）

**组件树**：首页（ModuleCard 8 库卡片 + GraphView 3D 图谱 + 三脑检索，:147-280）→ 库面板（:303-316）：knowledge→KnowledgePanel(variant="page")、cost→CostLibrary、profile→ProfileLibrary、office→OfficeMemoryLibrary、materials→MaterialsLibrary、whisper→WhisperMemoryLibrary、graph→GraphView(variant="page")、digitallife→DigitalLifeLibrary；公共 FilePreviewModal(:320)。

**绑定调用面**：经 gaea/lib/bridge.ts 的 app 代理（MemoryHubOverview :132、WorkspaceSearch/SemanticSearch/FileSemanticSearch :90-92、各库 CRUD）＋ window.go.app.App.BrainSearch 直连（:86-89 三脑检索）。**注意：记忆中枢调用的 MemoryB/CostB 门面方法全部是 Gaea* 前缀——它实际复用办公引擎的记忆/成本域，是"跨门面聚合板块"的典型。**

**跨板块联动**：检索命中文件 → usePreviewStore.openFilePreview（:203，办公板块嵌入预览）；一键 @ 引用 → useComposerInsertStore.requestAt（:126，回办公板块自动插入输入框）。

### 2.6 模型中心板块（ModelCenterPage + modelcenter/ 子目录）

**组件树**：ModelCenterPage → ResourceMonitor(:222) + 10 分类 tab（:229-243：overview/llm/image/tts/specialty/catalog/benchmark/retrieval/engine/bind，:138-149）+ ModelCenterContext.Provider(:228) + StatsSection Drawer(:244-252)。分类状态本地 useState（:47）。

**状态管理**：**Context + 5 分类 hook**（ModelCenterPage.tsx:51-55）——useEngineState（引擎/模型/Key）、useStatsState（调用统计）、useImageState（图片后端）、useVoiceState（语音/OCR/聊天语音）、useBindState（功能绑定/模型路由/剧照）。各 Section 经 ModelCenterContext 消费同一份状态（context.ts，107 行）。

**后端绑定清单**：直连 App.*（useEngineState.ts:103 GetActiveModel、:154 StartLocalTTSService；useVoiceState.ts:53 GetVoicePipelineConfig、:78-79 SetActiveASRModel/SetActiveTTSModel、:106/119 SetChatVoiceModel、:136 GetTTSSpeakers；useBindState.ts:72-125 GetFeatureModel/GetFeatureModelEnabled/SetFeatureModel/SetFeatureModelEnabled；ResourceMonitor.tsx:22 GetModelMonitor；BindSection.tsx:183 VoiceApplySettings）+ api/engines.ts 全部引擎/Herdsman/测评/分流方法（getEngines/saveEngine/testEngineConnection/refreshEngineModels/getModelCallStats/getHerdsmanCatalog/startBenchmark/streamProbe/getUsageOverview/getSemanticIndexStatus 等，engines.ts:307-493）+ api/settings.ts。**模型中心是绑定面最宽的板块（约 60+ 方法）。**

### 2.7 角色库板块（CharacterLibraryPage）

**组件树**：页头（统计 + 操作按钮 :228-251）→ 工具栏（搜索/类型/可聊天 :254-275）→ 档案墙 CharacterCard 网格（:316-337）→ CharacterLibEditor（:348）/ CharacterMemoryModal（:357）→ FeatureModelBar(feature="characterlib")（:365）。

**状态管理**：页面本地 useState（查询/分页/编辑/记忆弹窗）。**无 store**。

**后端绑定清单**（api/characterlib.ts）：listCharacters(:29)/getCharacter(:35)/saveCharacter(:41)/deleteCharacter(:46)/importProjectCharacters(:51)/listProjectCharacters(:56)/associateToProject(:61)/setProjectState(:66)/dissociateFromProject(:71)/syncProjectCharacters(:76)/drawRandom(:81)/generateFill(:86)/generateRandom(:95)/fillAll(:109)/generatePortrait(:115)。CharacterMemoryModal 直连 WhisperGetState/WhisperGetFacts/WhisperGetTraces（CharacterMemoryModal.tsx:32/36/40，复用 Whisper* 记忆组件）。订阅 character-fill-progress 事件（:189）。

**跨板块事件**：设为聊天人格 → dispatch `gaea-persona-changed`（:34，ChatPage 消费 ChatPage.tsx:177-186）；移出项目 → dispatch `gaea-project-chars-changed`（:140，CharacterPage 消费）。

### 2.8 页面层复用散件组件（跨板块共享）

| 组件 | 引用方 | 说明 |
|---|---|---|
| FeatureModelBar | ChatPage.tsx:394 / NovelPage.tsx:80 / CharacterLibraryPage.tsx:365 / KnowledgePage.tsx:23 | 功能级模型卡，SetFeatureModelEnabled（FeatureModelBar.tsx:57），订阅 feature-model-changed |
| PersonaPicker | ChatModeBar.tsx:56 / ChatPersonaBar.tsx:33 | 聊天人格选择，App.CharacterList('', '', true, 1, 500)（PersonaPicker.tsx:33） |
| ChatPanel（通用对话组件） | NovelSettingPage.tsx:227（设定 Agent） | 552 行通用聊天面板，纯 props 驱动 |
| SearchModal | MainLayout.tsx:465 | 全局搜索，window.go.app.App.Search（SearchModal.tsx:56） |
| TTSPlayer | ChapterPage.tsx:259 | TTS 播放，订阅 tts-stream（TTSPlayer.tsx:109） |
| WelcomePage | HomePage.tsx:105 | 未登录品牌欢迎页 |
| VoiceChatOrb | ModuleLauncher.tsx:218 / chat/WelcomeScreen.tsx:118 | 语音粒子球（useVoiceChat 状态驱动） |
| Whisper* 记忆组件（4 个） | CharacterMemoryModal.tsx:9-14/77-119 | 角色库里的聊天记忆面板（WhisperGetState/Facts/Traces） |
| SecurityBanner | MainLayout.tsx:403 | 全局安全告警横幅 |
### 2.9 设置板块（SettingsPage）

**组件树**：9 分类磁贴（general/chat/novel/imagegen/office/model/security/data/about，SettingsPage.tsx:30-103）+ 关键字搜索（:111-118）。面板即 CATEGORIES 数组内联 JSX（:37 等）。

**状态管理**：页面本地 useState（query/activeKey）。**无 store**。

**后端绑定清单**：ChatPanel→WhisperGetPersonalities/GetVoicePipelineConfig/GetTTSSpeakers/WhisperClearSession（settings/ChatPanel.tsx:83-126）+ api/settings applyVoiceSettings；OfficePanel→App.GaeaSaveSettings(:77)/GaeaReload(:114)；ModelPanel→api/settings(getActiveModel/getConfig/saveConfig)+api/engines(getUsdCnyRate/setUsdCnyRate)，**且 ModelPanel.tsx:59 分发 navigate 事件跳 modelcenter**；SecurityPanel→window.go.app.App 直连（:26）；DataPanel→GaeaDataBackupInfo/Create/Restore/Rollback/Cancel(:49-113)；AboutPanel→GetAppInfo(:23)；ImageGenPanel→getImageBackendInfo/setImageBackend+getEngines；WorkspacePanel→SkillModal（技能管理）。

## 3. 板块间导航与状态

### 3.1 导航途径全清单（页面层新增证据，壳层清单见姊妹报告 §2.2）

| # | 途径 | 目标页 | 代码位置 |
|---|---|---|---|
| 1 | navigate 事件（页面分发）→ characterlib | 角色库 | pages/chat/utils.ts:13-15（ChatPage 三处引用 :304/323/345）；CharacterPage.tsx:76-77（两处按钮 :464/:735） |
| 2 | navigate 事件（页面分发）→ modelcenter | 模型中心 | components/settings/ModelPanel.tsx:59 |
| 3 | ModuleLauncher onNavigate → setPage | chat/novel/imagegen/gaea/modelcenter/settings | ModuleLauncher.tsx:17-18（LauncherTarget 6 值，缺 memoryhub/characterlib）+ :31-38（modules 数组）+ MainLayout.tsx:446 |
| 4 | gaea-persona-changed（跨板块广播） | characterlib → chat（人格联动） | CharacterLibraryPage.tsx:34 分发 / ChatPage.tsx:184 消费 |
| 5 | gaea-project-chars-changed（跨板块广播） | characterlib/novel → CharacterPage 刷新 | CharacterLibraryPage.tsx:140、NewCharactersModal.tsx:100 分发 / CharacterPage.tsx:190 消费 |
| 6 | useComposerInsertStore（@ 引用通道） | memoryhub → 办公 Composer | MemoryHubPage.tsx:126 / store.ts:319-334 |
| 7 | usePreviewStore（文件预览通道） | memoryhub → 办公嵌入预览 | MemoryHubPage.tsx:203 / store.ts:290-294、gaea/App.tsx:158-168 |
| 8 | VOICE_LAUNCH_FLAG（首页语音 → 聊天） | home → chat | ModuleLauncher.tsx:21 / useChatVoice.ts:53-60 |
| 9 | NovelPage 二级 tab（板块内子页跳转） | 书架/设定/角色/创作/阅读/导出 | NovelPage.tsx:17-37（localStorage 记忆 :18） |
| 10 | ModelCenterPage 分类 tab（板块内） | 10 分类 | ModelCenterPage.tsx:138-162 |
| 11 | MemoryHubPage 库切换（板块内） | 首页 ↔ 8 库 | MemoryHubPage.tsx:71（active state） |
| 12 | SettingsPage 分类磁贴（板块内） | 9 分类 | SettingsPage.tsx:148-160 |

### 3.2 板块内/间状态共享方式汇总

| 板块 | 状态容器 | 共享边界 |
|---|---|---|
| chat | 页面 useState + 3 hook | 完全页面私有；localStorage 持久化（话题/人格/折叠态） |
| novel | 子页 useState + **outlineStore（zustand）** + appStore(projectPath) | 大纲跨 CreatePage/ChapterPage 共享；世界观文本各自拉取 |
| imagegen | 4 hook 全本地 | 无共享；localStorage（历史在内存） |
| gaea(办公) | zustand useStore（事件 reducer）+ 4 个辅助 store | 会话/事实/文件预览/Composer 插入全局共享（板块内） |
| memoryhub | 本地 useState + gaea store | 经 usePreviewStore/useComposerInsertStore 与办公联动 |
| modelcenter | ModelCenterContext + 5 hook | 板块内 Context 共享；与 FeatureModelBar 经 feature-model-changed 事件同步 |
| characterlib | 本地 useState | 与 chat/novel 经自定义事件同步 |
| settings | 本地 useState | 与 modelcenter 经 navigate 事件跳转；无共享状态 |

**结论**：跨板块无"会话级共享状态"——所有板块间通信靠 ① navigate 自定义事件、② gaea-persona-changed/gaea-project-chars-changed 广播、③ zustand 全局辅助 store（仅办公板块内部使用）、④ localStorage。这与 3.0「会话事件日志作事实源」的改造点直接相关：目前聊天/绘梦/小说没有任何事件日志消费面，状态无法跨页/跨端重建。

**补充观察**：MainLayout 的 visitedPages 保活（MainLayout.tsx:223-233）让页面组件实例常驻内存，跨板块跳转不销毁——这是页面状态「看似共享」的假象来源：ChatPage 换到角色库再回来消息还在，但一旦刷新或组件被 ErrorBoundary 重置即全丢；且 8 个页面同时挂载，内存与事件订阅（useChatStream 的 streamCleanupRef 等）随页面常驻累积，是保活策略的隐性成本。

## 4. KnowledgePage 孤儿页面分析

### 4.1 现状证据

- **有实现、无挂载**：KnowledgePage.tsx:13-27 是完整独立页面（LocaleProvider 包 KnowledgePanel(variant="page") + FeatureModelBar(feature="gaea" label="知识库")），但全仓 grep 仅命中其自身定义（frontend/src/pages/KnowledgePage.tsx:13/29），未出现在 MainLayout 的 Page 类型（MainLayout.tsx:27）、menuItems（:33-42）、pageComponents（:44-53）、allPageKeys（:30）、ModuleLauncher 模块表（ModuleLauncher.tsx:31-38）任何一处。
- **能力完整**：KnowledgePanel（gaea/components/KnowledgePanel.tsx，567 行）具备完整知识库 CRUD——列表/全文检索（KnowledgeList/KnowledgeSearch，:53/161）、条目读写（KnowledgeGet/Save/Delete，:208/236/247）、版本历史（KnowledgeHistory，:115）、查重（KnowledgeFindSimilar，:103/141）、合并（KnowledgeMerge，:149）、审核（KnowledgeReview，:132）、批量导出/改状态（KnowledgeExport :122、batchStatus :87）、文件导入（PickFiles + KnowledgeImportPreview/AIParse/Apply，:65 与 memoryhub/KnowledgeImportModal.tsx）。
- **三处复用同组件**：KnowledgePanel 被 gaea/App.tsx:909（办公板块抽屉，variant="modal" 默认）、MemoryHubPage.tsx:306（记忆中枢 knowledge 库，variant="page"）、KnowledgePage.tsx:18（孤儿页，variant="page"）三处挂载——**同一能力三种入口，数据同源（Knowledge* 方法）**。

### 4.2 为什么被 memoryhub 取代 / 是否可独立挂载

- 取代路径：MemoryHubPage 的 knowledge 库（MemoryHubPage.tsx:44-53，标签"知识库·规范/案例/经验条目"）与 KnowledgePage 渲染同一组件、同一数据源；且 memoryhub 首页还有跨库检索（三脑 + 语义 + 文件全文，MemoryHubPage.tsx:88-92），信息密度更高。gaea3 文档 §3.1 也确认"其能力已被 memoryhub 的 knowledge 库取代"。
- **可独立挂载**：技术上零障碍——KnowledgePanel 支持 variant="page"，KnowledgePage 已把挂载参数备好；缺的只是 MainLayout 一行注册（Page 类型 + menuItems + pageComponents + allPageKeys）与 ModuleLauncher 入口。
- **差异点**：孤儿页挂了 FeatureModelBar(feature="gaea" label="知识库")（KnowledgePage.tsx:23）——即知识库作为"办公功能"复用 office 的模型绑定；而 memoryhub 内的 knowledge 库没有模型卡。若并入 memoryhub，此绑定语义需保留或迁移。
- **建议**：gaea3 文档开放问题 D7 倾向"并入 memoryhub"。页面层证据支持该方案：合并后删除 pages/KnowledgePage.tsx，同时把 FeatureModelBar 语义（知识库走 office 功能模型）显式化；若保留独立板块，则 Manifest 化时它正好是"最小新增板块"的试金石（§5.4）。

## 5. 与 3.0 目标相关的关键发现

### 5.1 "板块"概念在页面层的五处独立清单（与 gaea3 文档 §3.1 呼应）

1. **MainLayout Page 类型**（9 值，MainLayout.tsx:27）+ pageComponents/menuItems/allPageKeys——壳层清单（详见姊妹报告）；
2. **ModuleLauncher LauncherTarget**（6 值，ModuleLauncher.tsx:17-18）+ modules 数组（:31-38）——首页启动器清单，缺 memoryhub/characterlib 两块；
3. **NovelPage 二级 tab**（NovelTab 6 值，NovelPage.tsx:17）——"板块内子板块"，自身也是独立清单（localStorage 持久化 :18）；
4. **SettingsPage CATEGORIES**（9 分类，SettingsPage.tsx:30-103）——按功能板块整理的设置分组，与导航清单一一映射但独立维护；
5. **MemoryHubPage LibraryKey**（8 库，MemoryHubPage.tsx:23-53）——记忆中枢聚合清单，横跨 MemoryB/CostB/OfficeB 门面。

**五处清单互不同步的实证**：加一个板块需同时改 Page 类型、menuItems、pageComponents、allPageKeys、ModuleLauncher 的 LauncherTarget+modules、SettingsPage 分类、FeatureModelBar feature 键、后端门面+注册表——这正是 gaea3 文档 §1 缺陷 1（"加板块 = 改壳代码 6 处"）在页面层的完整展开。

### 5.2 FeatureModelBar 的 feature 键是"事实上的板块注册表"

FeatureModelBar 出现在 chat（ChatPage.tsx:394）、novel（NovelPage.tsx:80）、characterlib（CharacterLibraryPage.tsx:365）、gaea（KnowledgePage.tsx:23，label"知识库"）四处；useFeatureModel 注释给出 feature 合法值 `'chat'|'whisper'|'novel'|'office'|'gaea'|'characterlib'`（hooks/useFeatureModel.ts:15）——**功能模型绑定键与板块 id 强耦合**，是 Manifest 化必须纳入的"板块级能力"（bindings/tools 维度）。imagegen/memoryhub/modelcenter/settings 无 FeatureModelBar。

**调用面碎片化的补充证据**：MemoryHubPage 是唯一同时使用两条后端调用路径的页面——大部分走 gaea bridge 的 app 代理（MemoryHubPage.tsx:90-92），三脑检索却直接 window.go.app.App.BrainSearch（:86-89）；CharacterMemoryModal 直接 WhisperGetState/Facts/Traces（CharacterMemoryModal.tsx:32/36/40）。即页面层存在「bridge 代理 / wailsjsCompat / window.go.app.App 直连」三套入口并存，同一板块内也未收敛。

### 5.3 页面与 MainLayout Page 类型耦合点（Manifest 化必须改动的面）

| # | 耦合点 | 位置 | 说明 |
|---|---|---|---|
| 1 | 页面 lazy 注册 | MainLayout.tsx:17-24 | 每页一行 import，新增板块必改 |
| 2 | pageComponents 映射 | MainLayout.tsx:44-53 | 8 页组件实例 |
| 3 | 保活渲染 | MainLayout.tsx:443-449 | visitedPages 遍历渲染，display:none 保活——manifest 的 keepAlive 属性落点 |
| 4 | 面包屑项目名→novel | MainLayout.tsx:415-417 | 项目语义硬编码为 novel 板块 |
| 5 | Content 布局特判 | MainLayout.tsx:424-432 | chat/gaea 专属 padding/背景/overflow——manifest 的 layout 属性落点 |
| 6 | 首页启动器 | MainLayout.tsx:445-447 | home 特判分支 + ModuleLauncher 6 卡 |
| 7 | 设置入口 | MainLayout.tsx:383-387 | settings 只从右上角按钮进，不进菜单 |
| 8 | 快捷键 Ctrl+1~4 | MainLayout.tsx:284-287 | 依赖 allPageKeys 前 4 项顺序 |
| 9 | 板块内二级 tab | NovelPage.tsx:17-37 / SettingsPage.tsx:30-103 / MemoryHubPage.tsx:23-53 / ModelCenterPage.tsx:138-149 | 四个板块各自持有"子视图清单"，Manifest 需支持嵌套导航（nav.children） |
| 10 | feature 模型键 | useFeatureModel.ts:15 + 4 处 FeatureModelBar | 板块能力（模型绑定）维度 |

### 5.4 对改造最有利的既有资产

1. **办公板块已是"板块自治"样板**：GaeaPage 薄壳（GaeaPage.tsx:13-23）+ bridge.ts 事件驱动 + store.ts 事件 reducer + gaeaToGaea 门面映射（bridge.ts:503-682）——「组件入口薄、事件订阅集中、状态合成单一」三要素齐全，Manifest 化可直接复制其形态。
2. **后端绑定面已拆分（10 门面）**：wailsjsCompat.ts:16-25 re-export CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB——前端旧调用形态零改动（姊妹报告 §3.2），说明"门面式改造对旧代码无感"策略已验证。
3. **bridge.ts 的编译期绑定面漂移检查**（bridge.ts:880-939，AssertNever 双向断言 + LegacySurfaceNames 白名单）——Manifest 化后的"板块 id ↔ 门面 ↔ 页面"一致性可复用同一护栏模式。
4. **KnowledgePage 可作为"最小新增板块"试点**：挂载它只需 manifest + 页面注册两处改动，正好验证"加板块只写声明"。

## 6. 缺陷与风险

1. **板块清单五处漂移**（§5.1）：Page 类型/LauncherTarget/NovelTab/SettingsPage 分类/LibraryKey 各自独立维护，任一增删都需人工同步，漏改即出现"有页面无入口/有入口无页面"（KnowledgePage 就是漏改的既成案例）。
2. **跨板块通信全部走隐式通道**：navigate/gaea-persona-changed/gaea-project-chars-changed 事件名字符串硬编码（无常量表），@ 引用/预览走 zustand 辅助 store（useComposerInsertStore/usePreviewStore）——板块间契约无声明、无校验，Manifest 化后这些通道应升级为显式"板块间意图"。
3. **页面状态无事实源**：ChatPage 消息/ImageGenPage 历史/NovelPage 创作状态全部页面本地 useState，切换板块靠 MainLayout visitedPages 保活组件实例（MainLayout.tsx:223-233）；刷新/重启即丢，跨页无法重建——与 gaea3 缺陷 3（会话非事实源）前后端互为因果。
4. **CharacterPage 783 行单文件**（pages/CharacterPage.tsx）：单页承载角色/组织/关系三 Tab + 抽卡/合并/补齐/剧照四类弹窗 + 详情抽屉，是页面层最大的单体组件，改造时应一并拆分（与 T6-10.1 拆 ChatPage 同法）。
5. **FeatureModelBar feature 键语义不一致**：KnowledgePage 用 feature="gaea" 但 label="知识库"（KnowledgePage.tsx:23），与 useFeatureModel.ts:15 注释的合法值集合（'office'/'gaea' 并存）存在二义——知识库到底绑"办公功能"还是独立功能，Manifest 需明确。
6. **模型中心绑定面最宽且直连/封装混用**：同一方法存在 wailsjsCompat 直连（useEngineState.ts:103）与 api/engines.ts 封装（getEngines :308）双路径；模型中心 19 个文件是子目录最多的板块，改造成本集中。
7. **SettingsPage 分类与板块导航耦合但不同步**：CATEGORIES 的 keywords 人工维护（SettingsPage.tsx:36-101），新增板块若忘加设置分类，设置搜索无法命中。
8. **记忆中枢是聚合板块但无自身门面**：8 库横跨 MemoryB/CostB/OfficeB（§2.5），Manifest 的 bindings 字段需要支持多门面引用，不能假定 1:1。
9. **孤儿页误导风险**：KnowledgePage 长期无入口却功能完整，后续开发者可能误以为已挂载而重复实现（KnowledgePanel 已在三处挂载，§4.1）。
10. **ChatTopicSidebar 等 chat 展示组件与 gaea/ 组件重名**：components/chat/ChatTopicSidebar.tsx 与 gaea/components/Sidebar.tsx 并存两套"会话侧栏"实现（聊天板块 vs 办公板块），能力重叠但不共享。

## 7. 改造建议

1. **BoardManifest 覆盖 §5.3 全部 10 个耦合点**：manifest 字段设计建议——`{ id, label, icon, component(lazy), keepAlive, layout:'full'|'padded', shortcut?, menuOrder?, nav:{children?}, featureModel?:'chat'|'novel'|... , bindings:[门面], intents, tools }`；MainLayout/ModuleLauncher/SettingsPage 分类全部改由清单派生（顺带补齐 ModuleLauncher 缺的 memoryhub/characterlib 入口）。
2. **板块内二级导航纳入 manifest**：NovelPage 6 tab、ModelCenterPage 10 tab、SettingsPage 9 分类、MemoryHubPage 8 库全部声明为 nav.children，消除 §5.1 五处清单漂移。
3. **跨板块通道契约化**：navigate 事件名、gaea-persona-changed、gaea-project-chars-changed、VOICE_LAUNCH_FLAG 收敛为常量表；@ 引用/预览通道升级为 manifest 声明的"板块间意图"（inter-board intents）。
4. **会话状态事实源化（Step 1 前端配合）**：ChatPage 消息流改从会话事件日志投影读取（替代 ChatMessagesList 全量重读 + 本地 useState）；办公板块 store.ts 的 applyEvent 事件处理模式可直接复用到聊天/绘梦。
5. **KnowledgePage 决策 + 试点**：按 D7 并入 memoryhub（删页面、保留 FeatureModelBar 语义）或保留为独立板块——若保留，正好作为"最小新增板块"验证 Manifest 流程（写 manifest + 注册页面 = 两处改动上线）。
6. **FeatureModelBar 语义入 manifest**：feature 键/模型绑定声明进 manifest 的 bindings 维度，消除 'gaea'/'office' 二义（§6.5）。
7. **CharacterPage 拆分**：按 T6-10.1 拆 ChatPage 的成功范式（页面编排层 + hooks + 纯展示组件），783 行拆为 3 Tab 各自组件 + 抽卡/合并/补齐 hooks。
8. **绑定面收敛为单 Seam**：模型中心等直连/封装混用的方法统一收口（api/ 模块内），为 Step 3 Provider Seam 预留"调用方不感知 provider"的接口层；复用 bridge.ts:880-939 的 AssertNever 模式给 manifest 与门面做编译期一致性断言。

> 结论：页面层是 3.0「板块 Manifest 化」的前端主体战场——8 个板块页面 + 4 个板块内子导航清单互不同步，跨板块通信靠隐式事件/store 通道，页面状态全部私有（无会话事实源）；办公板块（GaeaPage+bridge+store）已是"板块自治"样板，KnowledgePage 是现成的"最小新增板块"试点，后端 10 门面拆分 + wailsjsCompat 合并层证明"门面式改造对旧代码零影响"策略可行。