// bridge is the single seam between the React app and the Go kernel. In the Wails
// shell it calls the bound App methods (window.go.main.App.*) and subscribes to
// the runtime event stream (window.runtime.EventsOn). In a plain browser (`pnpm
// dev` outside the shell) those globals are absent, so it falls back to a mock
// that streams a canned turn through the same contract — letting the whole UI be
// developed and laid out without rebuilding the Go side.

import type {
  BalanceInfo,
  CapabilitiesView,
  CheckpointMeta,
  CommandInfo,
  ContextInfo,
  ContextTimeline,
  AgentNetwork,
  Trajectory,
  CrossEmbedInput,
  CrossEmbedResult,
  DirEntry,
  ExportDeliverableInput,
  ExportDeliverableResult,
  ConvertPdfResult,
  FactBaseView,
  FileSearchHit,
  FilePickResult,
  FilePreview,
  PreviewResult,
  GaeaSummaryResult,
  GaeaReloadResult,
  HistoryMessage,
  JobView,
  KnowledgeEntry,
  KnowledgeSaveRequest,
  KnowledgeSummary,
  MCPServerInput,
  MemoryArchivedPage,
  MemorySuggestion,
  MemorySuggestionsView,
  MemoryView,
  OfficeEditResult,
  XlsxEditResult,
  XlsxPlanResult,
  XlsxChartInput,
  XlsxChartResult,
  ZipDeliverableResult,
  SubagentRunsView,
  TaskOutputView,
  SkillSuggestion,
  TaskTemplate,
  Meta,
  ModelInfo,
  ProviderView,
  QuestionAnswer,
  SessionMeta,
  SessionStatsView,
  SubagentTranscriptView,
  DeliverableRegistryView,
  GaeaResyncResult,
  ProjectGroup,
  SettingsView,
  SkillCaptureInput,
  SkillCaptureResult,
  SlashArgsResult,
  UpdateInfo,
  UpdateProgress,
  WireEvent,
  WorkspaceView,
  SpaceOption,
  SpaceActiveView,
  MemoryHubOverview,
  ProfileFactView,
  ProgrammingWebLogTail,
  ProgrammingWebPreflight,
  ProgrammingWebStatus,
  WhisperAnchorReplayView,
  WhisperAnchorView,
  WhisperEpisodeView,
  WhisperEpisodeReplayView,
  WhisperMemoryView,
  MemoryGraphView,
  WorkspaceSearchHit,
  CostSummary,
  CostEntry,
  CostCategory,
  CostProject,
  CostProjectSummary,
  CostEstimateItem,
  CostEstimateVersion,
  CostIndicator,
  CostAttribution,
  CostReviewNote,
  CostImportPreview,
  CostCompareRow,
  CostComposeView,
  CostInquiryRecord,
  CostAdjustSuggestion,
  CostStageValue,
  CostStageCompareRow,
  CostStageDeviation,
  PriceSource,
  PriceFetchRecord,
  PriceHistory,
  SemanticHitView,
  SearchScope,
  UnifiedSearchView,
  IntentResultView,
  RetrievalEvalReport,
  KnowledgeImportPreview,
  KnowledgeHistoryView,
  SimilarView,
  MemoryDuplicateView,
  FileSemanticHit,
  TaskView,
  ModelSwitchEstimate,
  JournalChangeRecord,
  VerdictView,
  LintReportView,
  WhisperSubgraph,
  WhisperProactiveResult,
  WhisperProactiveConfigView,
  WeixinAssistantStatusRow,
  WeixinAssistantView,
  WeixinReminderView,
  WeixinReminderConfigView,
  TTSParams,
  PptxOutlineView,
} from "./types";
// BrowserObserveView 定义在 BrowserPanel.tsx（v4.28 A2，导出供 bridge/测试
// 复用——组件文件头有 react-refresh 抑制注明）。
import type { BrowserObserveView } from "../components/BrowserPanel";
import {
  isBindingAllowedInSpace,
  isSharedBinding,
  type GaeaFacetBySpace,
} from "./spaceBindings";
// chat 板块契约类型来自 wails 生成物（wails build 自动生成，勿手改生成物本身）。
// AppBindings 只做类型标注：ChatTopicsList/ChatMessagesList Go 侧为
// ([]chat.Topic, error) / ([]chat.Message, error)，Wails 绑定后失败呈现为
// rejected promise，这里按「[数据, 错误]」元组形态标注。
import type { app as AppModels, chat, whisper } from "../../../wailsjs/go/models";

// AppBindings mirrors desktop/app.go's exported method set. Keep in sync by hand
// (or regenerate with `wails generate module` and import wailsjs instead).
//
// Compile-time drift check: when a Go method is added/renamed but AppBindings is
// not updated, the type assertion below catches it at build time.  Fix: add the
// missing method to AppBindings, then run `pnpm typecheck`.
export interface AppBindings {
  Submit(input: string): Promise<void>;
  SubmitDisplay(display: string, input: string): Promise<void>;
  Cancel(): Promise<void>;
  // Steer 任务运行中插话调整：消息注入当前回合作为补充指引（不打断执行、
  // 不开新回合）；未运行时走 Submit 排队兜底。
  Steer(input: string): Promise<void>;
  // GaeaRunning 返回办公引擎当前是否真的在跑（看门狗校准用）。
  GaeaRunning(): Promise<boolean>;
  Approve(id: string, decision: "allow_once" | "allow_session" | "persist_allow" | "deny" | "abort"): Promise<void>;
  AnswerQuestion(id: string, answers: QuestionAnswer[]): Promise<void>;
  Compact(): Promise<void>;
  NewSession(): Promise<void>;
  // Reload 热加载办公引擎：重新读取磁盘上的持久化配置并重建 controller，
  // 使技能/工具/插件/参数变更无需重启即生效；返回重建后的工具/技能数量。
  Reload(): Promise<GaeaReloadResult>;
  History(): Promise<HistoryMessage[]>;
  // Checkpoints lists the session's rewind points; Rewind restores one (scope
  // "code" | "conversation" | "both"), after which the caller re-reads History.
  Checkpoints(): Promise<CheckpointMeta[]>;
  Rewind(turn: number, scope: string): Promise<void>;
  Fork(turn: number): Promise<void>;
  SummarizeFrom(turn: number): Promise<void>;
  SummarizeUpTo(turn: number): Promise<void>;
  // Session history: list saved sessions, resume one (returns its transcript),
  // delete one, or give one a custom display name ("" clears it).
  ListSessions(): Promise<SessionMeta[]>;
  // ListProjectSessions returns saved sessions grouped by workspace
  // (current first, then recently opened workspaces with sessions).
  ListProjectSessions(): Promise<ProjectGroup[]>;
  ResumeSession(path: string): Promise<HistoryMessage[]>;
  // SessionStats 返回会话级 token/成本派生统计（事件日志重放；legacy 会话
  // available=false）。恢复会话后调用，回填「全会话成本」缺失的历史部分。
  SessionStats(path: string): Promise<SessionStatsView>;
  // ArchiveSession moves a saved session to <sessions>/archive/ (restorable);
  // UnarchiveSession moves it back and returns the active path;
  // PinSession toggles the pinned flag (pinned sessions sort first).
  ArchiveSession(path: string): Promise<void>;
  UnarchiveSession(path: string): Promise<string>;
  PinSession(path: string, pinned: boolean): Promise<void>;
  DeleteSession(path: string): Promise<void>;
  RenameSession(path: string, title: string): Promise<void>;
  // Workspace: open a folder chooser and switch to that project (fresh session);
  // returns the chosen path, or "" if cancelled.
  ListWorkspaces(): Promise<WorkspaceView[]>;
  PickWorkspace(): Promise<string>;
  SwitchWorkspace(path: string): Promise<string>;
  // 双空间（S4，设计 §6）：work/play 静态枚举、当前生效空间、切换持久化。
  // Activate 只写配置（非法 space 拒绝）；生效时机为下次引擎重建/重启。
  GaeaSpaceList(): Promise<SpaceOption[]>;
  GaeaSpaceActive(): Promise<SpaceActiveView>;
  GaeaSpaceActivate(space: string): Promise<SpaceActiveView>;
  ContextUsage(): Promise<ContextInfo>;
  // ContextView 返回当前会话的上下文构成快照（dsh-context Go 移植 Phase A）：
  // 六分类当前组成、逐请求趋势、上下文事件、模型可见节点与归档。
  ContextView(): Promise<ContextTimeline>;
  // Trajectory 返回当前会话的轨迹时间线（轮次→步骤→工具调用）。
  Trajectory(): Promise<Trajectory>;
  // AgentNetwork 返回当前会话的 Agent 网络（主 agent 根 + 子代理树）。
  AgentNetwork(): Promise<AgentNetwork>;
  // TCCA 缓存报告（V3.0）— 返回 CacheReport JSON 字符串
  TCCAReport(): Promise<string>;
  // Balance queries the active provider's wallet balance (a network call);
  // returns an unavailable readout when no balance_url is configured or it fails.
  Balance(): Promise<BalanceInfo>;
  // Jobs lists the running background jobs (bash/task started in the background)
  // for the status-bar indicator.
  Jobs(): Promise<JobView[]>;
  // FactBase reads the current conversation's fact base (settled facts before
  // deliverables); FactBaseClear resets it from the sidebar panel.
  FactBase(): Promise<FactBaseView>;
  FactBaseClear(): Promise<void>;
  // FactBasePromote writes the fact base into permanent memory (dedup by key),
  // returning how many facts were saved/updated.
  FactBasePromote(): Promise<number>;
  // CaptureSkill 把一次成功对话沉淀为可复用技能（写入 .gaea/skills + 全局镜像，
  // 成功后热加载引擎）。同名技能允许覆盖。
  CaptureSkill(input: SkillCaptureInput): Promise<SkillCaptureResult>;
  Meta(): Promise<Meta>;
  Commands(): Promise<CommandInfo[]>;
  // Capabilities feeds the MCP & Skills drawer: connected/failed servers + skills.
  // Add connects + persists a server; Remove disconnects + drops it from config;
  // Retry reconnects a configured server that failed (config untouched).
  Capabilities(): Promise<CapabilitiesView>;
  AddMCPServer(input: MCPServerInput): Promise<number>;
  RemoveMCPServer(name: string): Promise<void>;
  RetryMCPServer(name: string): Promise<void>;
  // SetMCPServerEnabled is the per-session connector toggle (on reconnects, off
  // disconnects; config untouched).
  SetMCPServerEnabled(name: string, enabled: boolean): Promise<void>;
  SlashArgs(input: string): Promise<SlashArgsResult>;
  ListDir(rel: string): Promise<DirEntry[]>;
  // FileSearch 工作区文件名搜索（@ 引用增强：跨目录定位资料）。
  FileSearch(query: string, limit?: number): Promise<FileSearchHit[]>;
  // Materials 工作区资料概览：office/文本文件按修改时间倒序。
  Materials(limit?: number): Promise<FileSearchHit[]>;
  // WorkspaceSearch 工作区全文搜索（轻量 RAG）：正文关键词 + 命中片段。
  WorkspaceSearch(query: string, limit?: number): Promise<WorkspaceSearchHit[]>;
  // 常用资料固定（P1-②）：钉住的文件在新会话启动时自动带入上下文。
  PinnedMaterials(): Promise<FileSearchHit[]>;
  PinMaterial(rel: string): Promise<FileSearchHit[]>;
  UnpinMaterial(rel: string): Promise<FileSearchHit[]>;
  // SummarizeFile 工作区资料分块摘要（map-reduce），供资料面板「摘要后引用」。
  SummarizeFile(rel: string, focus?: string): Promise<GaeaSummaryResult>;
  // TaskTemplates 返回预置办公任务模板库（欢迎页 + slash 共用）。
  TaskTemplates(): Promise<TaskTemplate[]>;
  ReadFile(rel: string): Promise<FilePreview>;
  Preview(rel: string): Promise<PreviewResult>;
  // PptxOutline 读取 pptx 结构化大纲（v4.28 B2）：python-pptx 逐页标题/正文
  // 摘要，配合 pptx 预览的逐页缩略与「针对第 N 页修改」指令。失败结构化。
  PptxOutline(rel: string): Promise<PptxOutlineView>;
  // GaeaBrowserObserve 受控浏览器观察帧（v4.28 A2）：当前页 jpeg 截图+URL/
  // 标题；浏览器未运行 Available=false（被动动作，绝不拉起）。同名直调。
  GaeaBrowserObserve(): Promise<BrowserObserveView>;
  // OfficeEditText 框选即改：按指令生成选中文本的替换；DocxApplyEdit 以修订模式
  // （w:del+w:ins）写入 docx 并返回更新后的预览。
  OfficeEditText(selectedText: string, instruction: string): Promise<OfficeEditResult>;
  DocxApplyEdit(rel: string, selectedText: string, replacement: string): Promise<PreviewResult>;
  // DocxAcceptChanges 接受/拒绝 gaea 的待处理修订，返回更新预览。
  DocxAcceptChanges(rel: string, accept: boolean): Promise<PreviewResult>;
  // XlsxPlanEdit 单元格操作规划（不落盘）：上下文 → AI 规划操作 → 临时副本
  // 试运行 → 返回操作集与单元格级变更清单，供用户审阅批准。
  XlsxPlanEdit(rel: string, sheet: string, instruction: string, selection: string): Promise<XlsxPlanResult>;
  // XlsxApplyEdit 应用已批准的操作集（规划产物原样透传）：excelize 执行 +
  // LibreOffice 重算 → 返回更新预览。
  XlsxApplyEdit(rel: string, ops: string): Promise<XlsxEditResult>;
  // XlsxSetCell 直接写单元格（Excel 式双击编辑）：写值/公式 + LibreOffice 重算。
  XlsxSetCell(rel: string, sheet: string, ref: string, value: string): Promise<XlsxEditResult>;
  // XlsxRecalc 手动重算全部公式（LibreOffice）并返回更新预览。
  XlsxRecalc(rel: string): Promise<XlsxEditResult>;
  // XlsxRowOps 行级操作：insert_before / insert_after / delete（基于选中单元格所在行）。
  XlsxRowOps(rel: string, sheet: string, action: string, ref: string): Promise<XlsxEditResult>;
  // XlsxColOps 列级操作：insert_before / insert_after / delete（基于选中单元格所在列）。
  XlsxColOps(rel: string, sheet: string, action: string, ref: string): Promise<XlsxEditResult>;
  // XlsxChart 表格「选中区域 → 一键图表」：从选中区域提取数据 → excelize 在
  // 工作簿内嵌入原生图表对象（Excel/WPS 可见可编辑）→ 返回锚点与数据供迷你预览。
  XlsxChart(input: XlsxChartInput): Promise<XlsxChartResult>;
  // ZipDeliverables 会话产物一键打包：把本次会话交付文件打成一个 zip。
  ZipDeliverables(paths: string[]): Promise<ZipDeliverableResult>;
  // SubagentRuns 读取当前会话派发的全部子代理分工（状态/任务摘要/回答/工具数）。
  SubagentRuns(sessionPath: string): Promise<SubagentRunsView>;
  // Side Chat 式追问（v4.64）：对已完结的 sa_ 运行追加用户提问，后台运行
  // 即刻返回；文本增量走 gaea-subagent-text 专用通道，完成态经轮询自校正。
  SubagentFollowUp(sessionPath: string, ref: string, prompt: string): Promise<string>;
  // SubagentTranscript 读取某个子代理的完整 transcript（Agent 网络节点查看用）。
  SubagentTranscript(sessionPath: string, ref: string): Promise<SubagentTranscriptView>;
  // DeliverableRegistry 读取会话的权威产物登记表（v4.24 C1）：后端从事件日志
  // 折叠出写类工具落盘登记（路径/工具/轮次/时间/次数），替代正文扩展名白名单。
  DeliverableRegistry(sessionPath: string): Promise<DeliverableRegistryView>;
  // ResyncEvents 事件序号防线补拉（v4.26 对话流式重造）：Wails 事件流吞件时
  // 前端按 seq 跳号调用，后端从当前会话磁盘日志折叠出对话项全量快照整体替换。
  ResyncEvents(afterSeq: number): Promise<GaeaResyncResult>;
  // WriteFile 工作区内联编辑保存（C5）：把文本原子写回工作区相对路径文本文件
  // （路径/扩展名/大小校验在后端；用户显式保存，不走 agent 审批）。
  WriteFile(rel: string, content: string): Promise<void>;
  // ExportDeliverable 统一交付出口：受控 Markdown → docx/pptx/xlsx/md/pdf。
  ExportDeliverable(input: ExportDeliverableInput): Promise<ExportDeliverableResult>;
  // ConvertToPdf 文档转 PDF（LibreOffice 无头转换）：docx/xlsx/pptx/odt/html/
  // txt/csv 直接转换，md 先经 create_docx.py 出 docx 再转；PDF 落 .gaea/exports/。
  ConvertToPdf(rel: string): Promise<ConvertPdfResult>;
  // CrossEmbed 跨应用联动：xlsx 数据 → 图表 → 嵌入 docx/pptx。
  CrossEmbed(input: CrossEmbedInput): Promise<CrossEmbedResult>;
  OpenWorkspacePath(rel: string): Promise<void>;
  RevealWorkspacePath(rel: string): Promise<void>;
  SavePastedImage(dataUrl: string): Promise<string>;
  SaveAttachmentFile(fileName: string, base64Data: string): Promise<string>;
  AttachmentDataURL(path: string): Promise<string>;
  // CaptureScreen 捕获整个屏幕（返回 PNG data URL）；RecognizeImage 用本地
  // 视觉模型识别图片内容，返回文本描述。
  CaptureScreen(): Promise<string>;
  RecognizeImage(imagePath: string, prompt: string): Promise<string>;
  // OCRText 用本地 OvisOCR2 常驻服务提取图片中的文字（办公「提取文字」入口）。
  OCRText(imagePath: string): Promise<string>;
  Models(): Promise<ModelInfo[]>;
  SetModel(name: string): Promise<void>;
  // Memory panel: read the loaded REASONIX.md hierarchy + saved auto-memories,
  // quick-add a note to a scope's REASONIX.md (≡ "#<note>"), and overwrite a doc
  // from the in-place editor.
  Memory(): Promise<MemoryView>;
  Remember(scope: string, note: string): Promise<string>;
  Forget(name: string): Promise<void>;
  SaveDoc(path: string, body: string): Promise<string>;
  UpdateFact(name: string, body: string): Promise<string>;
  ChangeFactType(name: string, type: string): Promise<string>;
  // SetMemoryEnabled 记忆开关（记忆可控性）：关闭后不再注入画像/规则/事实，
  // 持久化并重建办公引擎立即生效。
  SetMemoryEnabled(enabled: boolean): Promise<void>;
  // MorningPreload 读/写晨报预载开关（~/.gaea_config.json，默认开）：work
  // 空间新会话装配时预装配高频工作记忆；写后重建引擎即时生效。
  MorningPreload(): Promise<boolean>;
  SetMorningPreload(enabled: boolean): Promise<void>;
  MemorySuggestions(): Promise<MemorySuggestionsView>;
  // LogFrontendError 记录前端错误/主线程卡死诊断到 gaea.log。
  LogFrontendError(message: string): Promise<void>;
  AcceptMemorySuggestion(candidate: MemorySuggestion): Promise<string>;
  AcceptMergeSuggestion(keep: string, archive: string): Promise<string>;
  AcceptSkillSuggestion(candidate: SkillSuggestion): Promise<string>;
  // Settings panel: read the resolved config and apply edits (each writes config
  // and rebuilds the controller live). Secrets go through SetProviderKey (→ .env).
  Settings(): Promise<SettingsView>;
  SetDefaultModel(ref: string): Promise<void>;
  SaveProvider(p: ProviderView): Promise<void>;
  DeleteProvider(name: string): Promise<void>;
  LoginProvider(name: string): Promise<void>;
  LogoutProvider(name: string): Promise<void>;
  SetProviderKey(apiKeyEnv: string, value: string): Promise<void>;
  SetPermissionMode(mode: string): Promise<void>;
  AddPermissionRule(list: string, rule: string): Promise<void>;
  RemovePermissionRule(list: string, rule: string): Promise<void>;
  SetSandbox(bash: string, network: boolean, workspaceRoot: string, allowWrite: string[]): Promise<void>;
  SetAgentParams(temperature: number, maxSteps: number, systemPrompt: string): Promise<void>;
  // SetSubagentTemperature sets the subagent-specific temperature override.
  // 0 means "use the global temperature".
  SetSubagentTemperature(temp: number): Promise<void>;
  // SetEffort sets the reasoning effort for the executor. "" = provider default.
  SetEffort(effort: string): Promise<void>;
  // SetSubagentEffort sets the reasoning effort for sub-agents. "" = inherit from Effort.
  SetSubagentEffort(effort: string): Promise<void>;
  // SetSubagentModel sets the default model for spawned sub-agents. An empty string
  // clears it so sub-agents inherit the parent's provider.
  SetSubagentModel(ref: string): Promise<void>;
  // SetSubagentModelForSkill sets a per-skill sub-agent model override.
  // skill is one of explore|research|review|security-review. Empty ref = inherit.
  SetSubagentModelForSkill(skill: string, ref: string): Promise<void>;
  // SetPermLevel controls permission strictness: "ask" (default, prompt before writes),
  // "auto" (allow writes without asking), or "yolo" (skip all prompts).
  SetPermLevel(level: string): Promise<void>;
  // Auto-updater (desktop/updater_app.go): the injected build version, a manifest
  // check, applying an update (win/linux self-update; macOS opens the download
  // page), and opening that page directly. Progress streams on "updater:progress".
  Version(): Promise<string>;
  CheckUpdate(): Promise<UpdateInfo | null>;
  ApplyUpdate(): Promise<void>;
  OpenDownloadPage(): Promise<void>;
  // Window state persistence.
  SaveWindowState(state: {width:number;height:number;x:number;y:number;maximised:boolean}): Promise<void>;
  // Knowledge base panel.
  KnowledgeList(): Promise<KnowledgeSummary[]>;
  // KnowledgeSearch 全文检索（标题/分类/标签/正文），空 query 等价于 List。
  KnowledgeSearch(query: string, category: string, phase: string, status: string): Promise<KnowledgeSummary[]>;
  // ── 记忆中枢 ──
  MemoryHubOverview(): Promise<MemoryHubOverview>;
  ProfileList(): Promise<ProfileFactView[]>;
  ProfileSave(f: ProfileFactView): Promise<void>;
  ProfileDelete(name: string): Promise<void>;
  ProfileConflicts(): Promise<string[]>;
  WhisperMemories(): Promise<WhisperMemoryView[]>;
  // ── v4.3 会客厅：关系图谱 + 主动关心 ──
  // WhisperGraphSubgraph 返回指定人格关系图谱中以 entity 为中心、hops 跳内子图。
  WhisperGraphSubgraph(personalityId: string, entity: string, hops: number): Promise<WhisperSubgraph>;
  // WhisperProactiveNow 手动触发主动关心评估（「轻语先开口」按钮/定时器共用）。
  WhisperProactiveNow(personalityId: string): Promise<WhisperProactiveResult>;
  // WhisperProactiveConfig 返回主动关心定时推送配置（频控上限/间隔/时窗/开关）。
  WhisperProactiveConfig(): Promise<WhisperProactiveConfigView>;
  // WhisperProactiveSetConfig 部分更新主动关心定时推送配置（JSON 字符串，校验失败报错）。
  WhisperProactiveSetConfig(cfgJSON: string): Promise<void>;
  // ── v4.4 微信触点（书房·离线代办；WhisperWeixin* 自 LegacySurface 转正）──
  // WhisperWeixinGetQR 获取微信扫码登录二维码（dataURL 图片 + 会话 token）。
  WhisperWeixinGetQR(): Promise<{ qrcode: string; imageUrl: string }>;
  // WhisperWeixinQRStatus 轮询扫码状态（waiting/scanned/confirmed；confirmed 携带 botToken/botId）。
  WhisperWeixinQRStatus(qrcode: string): Promise<Record<string, unknown>>;
  // WhisperWeixinQRStatusWithCode 带手机配对码轮询（need_verifycode 状态时使用）。
  WhisperWeixinQRStatusWithCode(qrcode: string, verifyCode: string): Promise<Record<string, unknown>>;
  // WhisperWeixinStatus 全部助手的微信通道状态（运行/过期/未配置）。
  WhisperWeixinStatus(): Promise<WeixinAssistantStatusRow[]>;
  // v4.48 青鸟人格选择器：WeixinPage 经 bridge 消费（原 legacy wailsjsCompat
  // 直调转正，浏览器 dev mock 可达）；从 LegacySurfaceNames 摘除。
  // WhisperGetPersonalities 轻语预设人格清单。
  WhisperGetPersonalities(): Promise<whisper.PersonalityPreset[]>;
  // CharacterList 角色库分页查询（chatOnly 过滤可聊天角色，返回 {items,total}）。
  CharacterList(query: string, kind: string, chatOnly: boolean, page: number, pageSize: number): Promise<Record<string, unknown>>;
  // WhisperAssistantList 全部虚拟助手（微信绑定/人格/启停）。
  WhisperAssistantList(): Promise<WeixinAssistantView[]>;
  // WhisperAssistantSave 新建/更新助手（含 WxToken/WxBotID 微信绑定，保存后自动重拉通道）。
  WhisperAssistantSave(ast: Partial<WeixinAssistantView>): Promise<void>;
  // WhisperAssistantDelete 删除助手并停其微信通道。
  WhisperAssistantDelete(id: string): Promise<void>;
  // WeixinReminderList 全量微信提醒（待触发/已完成/失败，触发时间升序）。
  WeixinReminderList(): Promise<WeixinReminderView[]>;
  // WeixinReminderAdd 前端手动建提醒（fireAtRFC3339 为 RFC3339 时间串，必须在未来）。
  WeixinReminderAdd(text: string, fireAtRFC3339: string): Promise<{ id: string; fireAt: string; status: string }>;
  // WeixinReminderDelete 删除提醒（任意状态可删）。
  WeixinReminderDelete(id: string): Promise<void>;
  // WeixinReminderConfig 微信任务化配置（当前仅提醒开关）。
  WeixinReminderConfig(): Promise<WeixinReminderConfigView>;
  // WeixinReminderSetConfig 部分更新微信任务化配置（JSON 字符串）。
  WeixinReminderSetConfig(cfgJSON: string): Promise<void>;
  // TTSVoiceParams 返回情绪标签对应的结构化 TTS 参数（预览/调试）。
  TTSVoiceParams(emotion: string): Promise<TTSParams>;
  // GenerateBookCover 生成项目书封（3:4，play exports），返回封面路径。
  GenerateBookCover(projectId: string, promptHint: string): Promise<string>;
  // WhisperEpisodes 聊天情节记忆（hermes.db，时间倒序）。
  WhisperEpisodes(): Promise<WhisperEpisodeView[]>;
  // WhisperEpisodeReplay 情节记忆回放（hermes.db，只读）：按情节 ID 重建原始对话。
  WhisperEpisodeReplay(episodeId: string): Promise<WhisperEpisodeReplayView>;
  // WhisperAnchors 轻语时间锚点列表（hermes.db，play 空间纪念日）。
  WhisperAnchors(): Promise<WhisperAnchorView[]>;
  // WhisperAnchorReplay 按时间锚点回放「重访那一天」：锚点 → 关联情节 → 原始对话。
  WhisperAnchorReplay(anchorId: string): Promise<WhisperAnchorReplayView>;
  // WhisperMemoryRetell 让 gaea 以当前人格口吻把一段记忆重述成故事（LLM 叙事）。
  WhisperMemoryRetell(kind: "episode" | "anchor", id: string, personalityId: string): Promise<string>;
  // WhisperCausalExplain 跨事实因果推断：基于图谱「导致」边 + event_chain 关联
  // 解释「为什么<entity>」，无证据时返回诚实回退文案。
  WhisperCausalExplain(entity: string, personalityId: string): Promise<string>;
  // WhisperExportArchive 导出聊天记忆归档（hermes.db → Markdown 分目录），返回文件数。
  WhisperExportArchive(dir: string): Promise<number>;
  // PickDirectory 系统目录选择对话框，返回所选目录（取消返回空串）。
  PickDirectory(): Promise<string>;
  MemoryGraph(): Promise<MemoryGraphView>;
  // ── 做梦 2.0 晨报（纯本地主动预取）──
  // MemoryMorningBrief 返回「今日晨报」JSON 串（前端 JSON.parse 后渲染）：
  // work 空间记忆 top5 + 常驻规则 + 近 24h dream 沉淀计数。零 LLM、只读。
  MemoryMorningBrief(): Promise<string>;
  // ── 成本库 ──
  CostList(): Promise<CostSummary[]>;
  CostSearch(query: string, category: string, status: string): Promise<CostSummary[]>;
  CostGet(name: string): Promise<CostEntry | null>;
  CostSave(e: CostEntry): Promise<void>;
  CostDelete(name: string): Promise<void>;
  // CostImportPreview 解析 xlsx/csv 报价单/测算表为可确认的成本条目。
  CostImportPreview(path: string): Promise<CostImportPreview>;
  // CostImportAIParse 用办公功能模型把表格行归一化为成本条目（AI 解析）。
  CostImportAIParse(path: string): Promise<CostImportPreview>;
  // CostImportApply 批量写入确认后的成本条目，返回成功条数。
  // v4.6：inquirySource 非空（如 "OCR报价"）时，报价单行自动幂等写入询价库
  // （PDF/图片报价单飞轮反向接线）。
  CostImportApply(rows: CostEntry[], inquirySource?: string): Promise<number>;
  // CostImportVisionPreview 解析 PDF（文字型/扫描件 OCR）/图片报价单为可确认的
  // 成本条目；preview.source 标注识别来源（pdf_text / pdf_scan / image）。
  CostImportVisionPreview(path: string): Promise<CostImportPreview>;
  // 多级分类：分类树（含计数）、新建/改名、删除（有子节点或条目时拒绝）。
  CostCategories(): Promise<CostCategory[]>;
  CostCategorySave(parentId: number, name: string, sort: number, id: number): Promise<number>;
  CostCategoryDelete(id: number): Promise<void>;
  // ── 价格源（定时抓取价格更新）──
  PriceSources(): Promise<PriceSource[]>;
  PriceSourceSave(src: PriceSource): Promise<void>;
  PriceSourceDelete(id: string): Promise<void>;
  // PriceFetch 立即抓取价格源（异步任务，T5-1）：提交任务入队立即返回任务
  // 视图，进度/完成经 gaea-task 事件推送；完成后读 PriceFetches 取 pending 记录。
  PriceFetch(id: string): Promise<TaskView>;
  // PriceFetchAll 一键抓取全部启用的价格源（异步任务），逐源进度事件推送，
  // 失败明细在任务结果/消息里。
  PriceFetchAll(): Promise<TaskView>;
  PriceFetches(): Promise<PriceFetchRecord[]>;
  // PriceFetchApply 确认发布抓取结果（按标题选择），返回写入条数。
  PriceFetchApply(fetchId: string, titles: string[]): Promise<number>;
  PriceFetchIgnore(fetchId: string): Promise<void>;
  // PriceHistory 返回某成本条目的价格历史（新→旧）。
  PriceHistory(name: string): Promise<PriceHistory[]>;
  // SemanticSearch 跨库统一语义检索（成本/知识/办公记忆，本地 bge-m3）。
  SemanticSearch(query: string): Promise<SemanticHitView[]>;
  // CostCompare 返回某成本条目的多来源比价明细（现价/历史/价格源抓取候选）。
  CostCompare(name: string): Promise<CostCompareRow[]>;
  // ── 测算项目与沉淀闭环（我的项目/工程量清单/版本留痕 → 沉淀回成本库）──
  CostProjectSave(p: CostProject): Promise<string>;
  CostProjectList(): Promise<CostProjectSummary[]>;
  CostProjectGet(id: string): Promise<CostProject | null>;
  CostProjectDelete(id: string): Promise<void>;
  CostEstimateItemSave(i: CostEstimateItem): Promise<number>;
  CostEstimateItemDelete(id: number): Promise<void>;
  CostEstimateItems(projectId: string): Promise<CostEstimateItem[]>;
  CostEstimateVersionSave(projectId: string, note: string): Promise<CostEstimateVersion>;
  CostEstimateVersions(projectId: string): Promise<CostEstimateVersion[]>;
  // CostEstimateSediment 沉淀选中明细行回成本库（UPSERT cost_entries），返回条数。
  CostEstimateSediment(projectId: string, itemIds: number[]): Promise<number>;
  // ── 造价参考与复盘笔记（案例指标 + 经验沉淀）──
  // CostIndicators 造价参考指标：group=title（按科目）| category（按一级分类）。
  CostIndicators(group: string): Promise<CostIndicator[]>;
  // CostAttribution 归因对标（v4.6.1）：项目明细 vs 参考指标带宽，产出
  // 差幅等级/贡献金额/主因 TopDrivers（参考池排除本项目）。
  CostAttribution(projectId: string): Promise<CostAttribution>;
  CostNoteSave(n: CostReviewNote): Promise<number>;
  CostNoteList(query: string, status: string): Promise<CostReviewNote[]>;
  CostNoteDelete(id: number): Promise<void>;
  CostNoteBumpRef(id: number): Promise<void>;
  // CostGraph 成本知识图谱（v4.8）：scope=tree 分类聚合总览（默认）| entry 条目
  // 展开（focus=分类路径或项目 ID）；limit=节点上限（<=0 或 >600 归一 600）。
  // 返回 JSON 串（CostGraphView），由前端 JSON.parse（与 CostImportApply 等先例
  // 不同：该视图含大量节点/边，走字符串通道避免绑定层结构映射开销）。
  CostGraph(scope: string, focus: string, limit: number): Promise<string>;
  // ── v4.2 造价 AI 化：AI 组价 + 询价飞轮 + 五算对比 ──
  // CostCompose AI 组价：清单描述 → 相似检索（关键词+语义）→ 价格带推荐 +
  // 证据链 + LLM 人材机拆解；band=null 表示成本库无相似条目。无确认不落库。
  CostCompose(desc: string, unit: string): Promise<CostComposeView>;
  // CostComposeApply 确认组价建议并回写成本库（UPSERT），返回条目 name。
  CostComposeApply(v: CostComposeView): Promise<string>;
  // ── 询价飞轮（四源归一数据点：信息价/OCR报价/供应商比价/手动询价）──
  CostInquirySave(r: CostInquiryRecord): Promise<number>;
  CostInquiryList(query: string, limit: number): Promise<CostInquiryRecord[]>;
  CostInquiryDelete(id: number): Promise<void>;
  // CostInquiryExpiring 到期预警：valid_until <= today+days 的数据点。
  CostInquiryExpiring(days: number): Promise<CostInquiryRecord[]>;
  // CostInquiryAdjust 调差建议：成本库条目 vs 最新询价数据点（|差幅|>2%）。
  CostInquiryAdjust(): Promise<CostAdjustSuggestion[]>;
  // ── 五算对比（估/概/预/结/决）──
  CostStageSave(v: CostStageValue): Promise<void>;
  CostStages(projectId: string): Promise<CostStageValue[]>;
  CostStageCompare(projectId: string): Promise<CostStageCompareRow[]>;
  CostStageDeviations(projectId: string): Promise<CostStageDeviation[]>;
  // UnifiedSearch 跨库统一检索一次调用：工作区关键词命中（topN 条）+ 语义跨库命中。
  // S1.2-B/C（docs/gaea-memory-isolation-design.md）：签名加 scope 参数——
  // ""=全部（旧行为，仅显式选择时使用），"work"/"play"=只搜对应空间；前端默认
  // 传当前生效空间（GaeaSpaceActive，双空间红线：默认不跨空间混搜）。
  // 注意：后端 B 步合入前，旧绑定为 (query, topN)，此签名按约定先行对齐。
  UnifiedSearch(query: string, scope: SearchScope, topN?: number): Promise<UnifiedSearchView>;
  // RouteIntent 统一意图路由（v4.5 指令中枢）命令面板入口（S4.6）：
  // dryRun=true 只解析与校验、零副作用（返回「将发生什么」预览语 + action/target），
  // 面板据此渲染指令预览卡；用户显式确认后以 dryRun=false 真执行并返回回执。
  // 预览-确认制 = 「宁漏勿误」纪律在搜索框面的落地（搜索词不是整句指令入口）。
  RouteIntent(text: string, dryRun: boolean): Promise<IntentResultView>;
  // RetrievalEvalRun 运行检索质量测评：内置查询集统计平均 recall@10，
  // 达标门槛后端固定 0.8，返回指标 + 逐查询命中明细。
  RetrievalEvalRun(): Promise<RetrievalEvalReport>;
  // ── 知识库导入 ──
  KnowledgeImportPreview(path: string): Promise<KnowledgeImportPreview>;
  KnowledgeImportAIParse(path: string): Promise<KnowledgeImportPreview>;
  KnowledgeImportApply(rows: KnowledgeEntry[]): Promise<number>;
  // KnowledgeHistory 返回某条目的版本历史（新→旧）。
  KnowledgeHistory(name: string): Promise<KnowledgeHistoryView[]>;
  // KnowledgeFindSimilar 查重：标题模糊相似的既有条目。
  KnowledgeFindSimilar(title: string): Promise<SimilarView[]>;
  // KnowledgeExport 批量导出知识库为 Markdown，返回条数。
  KnowledgeExport(dir: string): Promise<number>;
  // KnowledgeReview 审核流：approve=true 置为现行并记录审核人，false 驳回。
  KnowledgeReview(name: string, approve: boolean, reviewer: string): Promise<void>;
  // KnowledgeMerge 把 sourceNames 合并进 targetName（标签并集/来源合并），返回目标名。
  KnowledgeMerge(targetName: string, sourceNames: string[]): Promise<string>;
  // ── 办公记忆查重/合并 ──
  MemoryDuplicates(min: number): Promise<MemoryDuplicateView[]>;
  MemoryMerge(targetName: string, sourceNames: string[]): Promise<string>;
  // ── 办公记忆归档生命周期（T6-8.2 / 记忆统一层）──
  // MemoryArchivedList 分页列出归档（含超过 90 天的硬删除候选），
  // 响应含 retentionDays（归档保留期天数，前端展示「归档保留 N 天」）；
  // MemoryCleanupArchived 硬删除归档超过保留期的事实，返回删除条数（无超期返回 0）；
  // MemoryUnarchive 恢复一条已归档记忆回活跃列表（误归档可在保留期内一键恢复）；
  // MemoryUnarchiveBatch 批量恢复（逐条恢复、失败跳过并聚合错误，返回成功数）；
  // MemorySetRetentionDays 设置归档保留期（天，钳制 [1,730]，持久化生效）。
  MemoryArchivedList(limit: number, offset: number): Promise<MemoryArchivedPage>;
  MemoryCleanupArchived(): Promise<number>;
  MemoryUnarchive(name: string): Promise<void>;
  MemoryUnarchiveBatch(names: string[]): Promise<number>;
  MemorySetRetentionDays(days: number): Promise<void>;
  // ── 工作区文件语义索引 ──
  // FileIndexRebuild 重建索引（异步任务，T5-1）：进度经 gaea-task 事件推送，
  // 结果（total/skipped）在任务 result 里。
  FileIndexRebuild(): Promise<TaskView>;
  FileSemanticSearch(query: string, topN: number): Promise<FileSemanticHit[]>;
  // Herdsman 深挖：数字生命记忆总览（只读）与最近异步操作。
  HerdsmanDigitalLife(): Promise<unknown>;
  HerdsmanOperations(): Promise<unknown>;
  // 画像冲突裁决
  ProfileResolveConflict(name: string, prefer: string): Promise<void>;
  KnowledgeGet(name: string): Promise<KnowledgeEntry | null>;
  KnowledgeSave(entry: KnowledgeSaveRequest): Promise<void>;
  KnowledgeDelete(name: string): Promise<void>;
  // Cost database panel.
  // PickFiles opens a native file dialog and imports selected files.
  PickFiles(): Promise<FilePickResult[]>;
  // ── 阶段 5 T5-1 任务中心 ──
  // TaskList 返回最近任务（新→旧）；TaskCancel 取消（running 中断/queued 取消）；
  // TaskRetry 重试失败/已取消的任务；TaskOutput 读取任务实时输出尾部（C1）。
  // 任务实时进度经 onTaskEvent 推送。
  TaskList(space?: string[]): Promise<TaskView[]>;
  TaskCancel(id: string): Promise<void>;
  TaskRetry(id: string): Promise<void>;
  TaskOutput(id: string): Promise<TaskOutputView>;
  // v4.1 证据链：Journal 最近证据卡（跨会话聚合，时间倒序；前端「证据」入口）。
  GaeaJournalList(limit: number): Promise<JournalChangeRecord[]>;
  // v4.1b：双通道复核一张证据卡（A 结构/引用完整性 + B 视觉健全性）。
  VerifyRecord(id: string): Promise<VerdictView>;
  // v4.1b：基线快照回滚（目标被手工修改时拒绝，零覆盖）。
  RollbackRecord(id: string): Promise<void>;
  // v4.1c：中文规范体检（GB/T 9704 红头要素 lint，md/txt/docx）。
  DocumentLint(rel: string): Promise<LintReportView>;
  // ── 阶段 5 T5-3 本地模型调度纵深 ──
  // KeepWarmGet/KeepWarmSet 保活开关：空闲时定期轻量探测，防止本地模型被卸载
  // （~/.gaea_config.json 持久化，重启后仍生效）。
  KeepWarmGet(): Promise<boolean>;
  KeepWarmSet(enabled: boolean): Promise<void>;
  // PreloadPlanGet/PreloadPlanSet 启动自动预载开关：启动时后台预载常用本地模型，
  // 减少首次对话等待。
  PreloadPlanGet(): Promise<boolean>;
  PreloadPlanSet(enabled: boolean): Promise<void>;
  // ModelSwitchEstimate 换模预估：目标本地模型 hot/cold/download/unknown，
  // 前端在非 hot 时提示预计等待秒数并让用户确认是否继续切换。
  ModelSwitchEstimate(engineID: string): Promise<ModelSwitchEstimate>;
  // ── 对话 chat（T6-3 契约同步）────────────────────────────
  // ChatTopicsList/ChatMessagesList Go 侧签名变为 ([]T, error)（T6-3.2 读错
  // 返回 error），Wails 绑定后失败为 rejected promise；这里以 [T[], unknown]
  // 元组形态标注「成功数据 + 失败错误」，调用点必须 try/catch，不能再 || [] 吞错。
  ChatTopicsList(): Promise<[chat.Topic[], unknown]>;
  ChatMessagesList(topicID: string): Promise<[chat.Message[], unknown]>;
  // ChatAppendMessages 语音消息持久化（T6-3.3）：单事务批量追加，
  // role 仅接受 user/assistant（其余后端跳过）。
  ChatAppendMessages(topicID: string, messages: AppModels.ChatMessageInput[]): Promise<void>;
  // ── 编程板块：DeepSeek Harness Web 进程管理 ──────────────
  // GetProgrammingWebStatus 返回运行状态（running/owned/pid/url/root/log/uptime_s）；
  // StartProgrammingWeb 启动 dsh web（已运行幂等返回）；StopProgrammingWeb
  // 仅停止 gaea 自启实例（外部实例返回提示，不误杀）。
  // GetProgrammingWebPreflight 返回启动前置条件逐项检查（harness 有效 / pnpm 可用 /
  // 依赖已装 / 构建就绪 / 端口空闲 + all_ready）；ProgrammingWebLogTail(n) 读自启
  // 日志尾部（n 钳制 [1,200]，默认 50），启动引导视图渲染真实清单与日志。
  GetProgrammingWebStatus(): Promise<ProgrammingWebStatus>;
  StartProgrammingWeb(): Promise<void>;
  StopProgrammingWeb(): Promise<void>;
  GetProgrammingWebPreflight(): Promise<ProgrammingWebPreflight>;
  ProgrammingWebLogTail(n?: number): Promise<ProgrammingWebLogTail>;
}

// Window 类型由 gaea 的 src/types/wails.d.ts 统一声明（go.app.App + runtime）。
// gaeaW 在此不重复声明，避免覆盖 gaea 的 runtime（EventsOff）类型。

// onEvent subscribes to the agent's typed event stream; returns an unsubscribe.
// 清理函数只摘除本监听者（v4.62.2）：此前用 EventsOff(channel) 全清——
// SubagentThread 卸载时把主对话 store 的订阅连带炸掉，对话标签页实时输出
// 全灭（轮询类面板正常），见 lib/wailsEvents.ts 的事故注记。
export function onEvent(cb: (e: WireEvent) => void): () => void {
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    return subscribeWailsEvent(window.runtime, EVENT_CHANNEL, (payload) => cb(payload as WireEvent));
  }
  return mockSubscribe(cb);
}

// Must match desktop/app.go's eventChannel constant.
const EVENT_CHANNEL = "gaea-event";

/** 子代理流式增量事件（专用通道 payload，v4.62.1 与 gaea-event 分道）。 */
export interface SubagentTextEvent {
  kind: "subagent_text";
  text: string;
  subagentRef?: string;
  parentId?: string;
}

// 子代理流式增量走专用 wails 通道（无 seq）：gaea-event 的契约是「seq 与磁盘
// 账本 1:1，丢件可 resync 补拉」（v4.26 防线）；本通道是有损无妨的装饰性实时
// 流，绝不可上 gaea-event 消费 seq（v4.62.0 曾因此把对话窗过程可见性打断）。
// mock 场景复用 mockSubscribe：mock 可用 kind=subagent_text 的事件演示流式。
const SUBAGENT_TEXT_CHANNEL = "gaea-subagent-text";

// onSubagentText subscribes to the subagent streaming-delta channel.
export function onSubagentText(cb: (e: SubagentTextEvent) => void): () => void {
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    return subscribeWailsEvent(
      window.runtime,
      SUBAGENT_TEXT_CHANNEL,
      (payload) => cb(payload as SubagentTextEvent),
    );
  }
  return mockSubscribe(cb as unknown as (e: WireEvent) => void);
}

// Resolve the Wails binding at CALL time, not module-load time: in dev the Wails
// runtime can inject window.go AFTER this module first evaluates, so snapshotting
// once would pin the browser mock for the whole session (and show fake data — the
// dev mock's model list leaking into the real app was exactly this bug).
//
// S2-3「App 绑定面拆分」：后端绑定面已从单一 window.go.app.App 拆为多个板块
// 门面（go.app.CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/
// ImageB/CharLibB）。这里返回一个按方法名路由到对应门面的代理，前端调用点
// （app.Submit 等）零改动。
function realApp(): AppBindings | undefined {
  if (typeof window === "undefined") return undefined;
  const goApp = (window as unknown as { go?: { app?: Record<string, unknown> } }).go?.app;
  if (!goApp || typeof goApp !== "object") return undefined;
  return new Proxy({} as AppBindings, {
    get(_t, prop) {
      const key = (gaeaToGaea as Record<string, string>)[String(prop)] ?? String(prop);
      for (const ns of Object.values(goApp)) {
        if (ns === null || typeof ns !== "object") continue;
        const rec = ns as Record<string, unknown>;
        const v = rec[key];
        if (typeof v === "function") return (v as (...a: unknown[]) => unknown).bind(rec);
      }
      return undefined;
    },
  });
}

let mockSingleton: AppBindings | null = null;
function getMock(): AppBindings {
  if (!mockSingleton) mockSingleton = makeMockApp();
  return mockSingleton;
}
// channel from the agent stream); returns an unsubscribe.
export function onUpdaterProgress(cb: (p: UpdateProgress) => void): () => void {
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    window.runtime.EventsOn("updater:progress", (p) => cb(p as UpdateProgress));
    return () => window.runtime?.EventsOff?.("updater:progress");
  }
  updaterListeners.add(cb);
  return () => {
    updaterListeners.delete(cb);
  };
}

// onTaskEvent subscribes to the task scheduler's event stream (gaea-task):
// every task status/progress change pushes the latest TaskView. space 非空时在
// 订阅层按 payload.spaceId 过滤（v4.5.1a 红线补课：S2.1 事件空间过滤推广到
// 任务事件——work 消费点传 "work"，play 任务事件不打扰工位 UI）；缺省
// spaceId（旧任务/旧后端）按 work 兼容放行，与 isWorkSpaceTask 同语义。
// Returns an unsubscribe. Falls back to the mock stream outside the Wails shell.
export function onTaskEvent(cb: (t: TaskView) => void, space?: string): () => void {
  const handler = (t: TaskView) => {
    if (space && t.spaceId && t.spaceId !== space) return;
    cb(t);
  };
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    window.runtime.EventsOn("gaea-task", (payload) => handler(payload as TaskView));
    return () => window.runtime?.EventsOff?.("gaea-task");
  }
  return mockTaskSubscribe(handler);
}

// onReady subscribes to the agent:ready event fired when boot.Build completes.
// The frontend re-fetches Meta/Context/History when this lands.
export function onReady(cb: () => void): () => void {
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    window.runtime.EventsOn("gaea-ready", () => cb());
    return () => window.runtime?.EventsOff?.("gaea-ready");
  }
  // In dev mock, fire immediately since there's no real boot sequence.
  cb();
  return () => {};
}

// gaeaToGaea maps gaeaW UI method names to the gaea App bindings.
// gaea 的办公板块绑定统一以 Gaea 前缀命名；gaeaW 的 UI 调用短名
// （Submit/Cancel/History/...），这里做名称映射，避免 gaea App 方法名冲突。
// 对话 chat 方法（ChatTopicsList/ChatMessagesList/ChatAppendMessages 等）在
// ChatB 门面上同名（无 Gaea 前缀），无需映射——代理按原名在门面上查找即可。
// as const 保留字面量类型（AppBindingTarget 编译期核验需要）；satisfies 保持
// Record<string, string> 约束。
const gaeaToGaea = {
  Submit: "GaeaSend",
  SubmitDisplay: "GaeaSend",
  Cancel: "GaeaCancel",
  Steer: "GaeaSteer",
  Approve: "GaeaApprove",
  AnswerQuestion: "GaeaAnswer",
  NewSession: "GaeaNewSession",
  Reload: "GaeaReload",
  CaptureSkill: "GaeaCaptureSkill",
  History: "GaeaHistory",
  Checkpoints: "GaeaCheckpoints",
  Rewind: "GaeaRewind",
  Fork: "GaeaFork",
  SummarizeFrom: "GaeaSummarizeFrom",
  SummarizeUpTo: "GaeaSummarizeUpTo",
  // Go 侧为 GaeaSummarizeFile（T6-10.3 对齐）。
  SummarizeFile: "GaeaSummarizeFile",
  ListSessions: "GaeaListSessions",
  ListProjectSessions: "GaeaListProjectSessions",
  ResumeSession: "GaeaResumeSession",
  SessionStats: "GaeaSessionStats",
  ArchiveSession: "GaeaArchiveSession",
  UnarchiveSession: "GaeaUnarchiveSession",
  PinSession: "GaeaPinSession",
  DeleteSession: "GaeaDeleteSession",
  RenameSession: "GaeaRenameSession",
  ListWorkspaces: "GaeaListWorkspaces",
  PickWorkspace: "GaeaPickWorkspace",
  SwitchWorkspace: "GaeaSwitchWorkspace",
  ContextUsage: "GaeaContext",
  ContextView: "GaeaContextView",
  Trajectory: "GaeaTrajectory",
  AgentNetwork: "GaeaAgentNetwork",
  TCCAReport: "GaeaTCCAReport",
  Balance: "GaeaBalance",
  Jobs: "GaeaJobs",
  FactBase: "GaeaFactBase",
  FactBaseClear: "GaeaFactBaseClear",
  FactBasePromote: "GaeaFactBasePromote",
  Meta: "GaeaMeta",
  Commands: "GaeaCommands",
  Capabilities: "GaeaCapabilities",
  AddMCPServer: "GaeaAddMCPServer",
  RemoveMCPServer: "GaeaRemoveMCPServer",
  RetryMCPServer: "GaeaRetryMCPServer",
  SetMCPServerEnabled: "GaeaSetMCPServerEnabled",
  SlashArgs: "GaeaSlashArgs",
  ListDir: "GaeaListDir",
  FileSearch: "GaeaFileSearch",
  Materials: "GaeaMaterials",
  WorkspaceSearch: "GaeaWorkspaceSearch",
  PinnedMaterials: "GaeaPinnedMaterials",
  PinMaterial: "GaeaPinMaterial",
  UnpinMaterial: "GaeaUnpinMaterial",
  TaskTemplates: "GaeaTaskTemplates",
  ReadFile: "GaeaReadFile",
  Preview: "GaeaPreview",
  PptxOutline: "GaeaPptxOutline",
  GaeaBrowserObserve: "GaeaBrowserObserve",
  OfficeEditText: "GaeaOfficeEditText",
  DocxApplyEdit: "GaeaDocxApplyEdit",
  DocxAcceptChanges: "GaeaDocxAcceptChanges",
  XlsxPlanEdit: "GaeaXlsxPlanEdit",
  XlsxApplyEdit: "GaeaXlsxApplyEdit",
  XlsxSetCell: "GaeaXlsxSetCell",
  XlsxRecalc: "GaeaXlsxRecalc",
  XlsxRowOps: "GaeaXlsxRowOps",
  XlsxColOps: "GaeaXlsxColOps",
  XlsxChart: "GaeaXlsxChart",
  ZipDeliverables: "GaeaZipDeliverables",
  SubagentRuns: "GaeaSubagentRuns",
  SubagentTranscript: "GaeaSubagentTranscript",
  DeliverableRegistry: "GaeaDeliverableRegistry",
  ResyncEvents: "GaeaResyncEvents",
  WriteFile: "GaeaWriteFile",
  ExportDeliverable: "GaeaExportDeliverable",
  ConvertToPdf: "GaeaConvertToPdf",
  CrossEmbed: "GaeaCrossEmbed",
  OpenWorkspacePath: "GaeaOpenWorkspacePath",
  RevealWorkspacePath: "GaeaRevealWorkspacePath",
  SavePastedImage: "GaeaSavePastedImage",
  SaveAttachmentFile: "GaeaSaveAttachmentFile",
  AttachmentDataURL: "GaeaAttachmentDataURL",
  CaptureScreen: "GaeaCaptureScreen",
  RecognizeImage: "GaeaRecognizeImage",
  OCRText: "GaeaOCRText",
  Models: "GaeaModels",
  SetModel: "GaeaSetModel",
  Memory: "GaeaMemory",
  Remember: "GaeaRemember",
  Forget: "GaeaForget",
  SaveDoc: "GaeaSaveDoc",
  UpdateFact: "GaeaUpdateFact",
  ChangeFactType: "GaeaChangeFactType",
  SetMemoryEnabled: "GaeaSetMemoryEnabled",
  MorningPreload: "GaeaMorningPreload",
  SetMorningPreload: "GaeaSetMorningPreload",
  MemorySuggestions: "GaeaMemorySuggestions",
  LogFrontendError: "GaeaLogFrontendError",
  AcceptMemorySuggestion: "GaeaAcceptMemorySuggestion",
  AcceptMergeSuggestion: "GaeaAcceptMergeSuggestion",
  AcceptSkillSuggestion: "GaeaAcceptSkillSuggestion",
  Settings: "GaeaSettings",
  SetDefaultModel: "GaeaSetDefaultModel",
  SaveProvider: "GaeaSaveProvider",
  DeleteProvider: "GaeaDeleteProvider",
  LoginProvider: "GaeaLoginProvider",
  LogoutProvider: "GaeaLogoutProvider",
  SetProviderKey: "GaeaSetProviderKey",
  SetPermissionMode: "GaeaSetPermissionMode",
  AddPermissionRule: "GaeaAddPermissionRule",
  RemovePermissionRule: "GaeaRemovePermissionRule",
  SetSandbox: "GaeaSetSandbox",
  SetAgentParams: "GaeaSetAgentParams",
  // SetSubagentTemperature/SetEffort/SetSubagentModel 无 Go 对应绑定（mock-only，
  // 见文件末尾 MockOnlyNames），不映射，按原名查找。
  SetSubagentEffort: "GaeaSetSubagentEffort",
  SetSubagentModelForSkill: "GaeaSetSubagentModelForSkill",
  SetPermLevel: "GaeaSetPermLevel",
  Version: "GaeaVersion",
  CheckUpdate: "GaeaCheckUpdate",
  ApplyUpdate: "GaeaApplyUpdate",
  OpenDownloadPage: "GaeaOpenDownloadPage",
  SaveWindowState: "GaeaSaveWindowState",
  KnowledgeList: "GaeaKnowledgeList",
  KnowledgeSearch: "GaeaKnowledgeSearch",
  MemoryHubOverview: "GaeaMemoryHubOverview",
  ProfileList: "GaeaProfileList",
  ProfileSave: "GaeaProfileSave",
  ProfileDelete: "GaeaProfileDelete",
  ProfileConflicts: "GaeaProfileConflicts",
  WhisperMemories: "GaeaWhisperMemories",
  WhisperGraphSubgraph: "GaeaWhisperGraphSubgraph",
  WhisperProactiveNow: "GaeaWhisperProactiveNow",
  WhisperProactiveConfig: "GaeaWhisperProactiveConfig",
  WhisperProactiveSetConfig: "GaeaWhisperSetProactiveConfig",
  TTSVoiceParams: "GaeaTTSVoiceParams",
  GenerateBookCover: "GaeaGenerateBookCover",
  WhisperEpisodes: "GaeaWhisperEpisodes",
  WhisperEpisodeReplay: "GaeaWhisperEpisodeReplay",
  WhisperAnchors: "GaeaWhisperAnchors",
  WhisperAnchorReplay: "GaeaWhisperAnchorReplay",
  WhisperMemoryRetell: "GaeaWhisperMemoryRetell",
  WhisperCausalExplain: "GaeaWhisperCausalExplain",
  WhisperExportArchive: "GaeaWhisperExportArchive",
  PickDirectory: "GaeaPickDirectory",
  MemoryGraph: "GaeaMemoryGraph",
  MemoryMorningBrief: "GaeaMemoryMorningBrief",
  CostList: "GaeaCostList",
  CostSearch: "GaeaCostSearch",
  CostGet: "GaeaCostGet",
  CostSave: "GaeaCostSave",
  CostDelete: "GaeaCostDelete",
  CostImportPreview: "GaeaCostImportPreview",
  CostImportAIParse: "GaeaCostImportAIParse",
  CostImportApply: "GaeaCostImportApply",
  CostImportVisionPreview: "GaeaCostImportVisionPreview",
  CostCategories: "GaeaCostCategories",
  CostCategorySave: "GaeaCostCategorySave",
  CostCategoryDelete: "GaeaCostCategoryDelete",
  PriceSources: "GaeaPriceSources",
  PriceSourceSave: "GaeaPriceSourceSave",
  PriceSourceDelete: "GaeaPriceSourceDelete",
  PriceFetch: "GaeaPriceFetch",
  PriceFetchAll: "GaeaPriceFetchAll",
  TaskList: "GaeaTaskList",
  SubagentFollowUp: "GaeaSubagentFollowUp",
  TaskCancel: "GaeaTaskCancel",
  TaskRetry: "GaeaTaskRetry",
  TaskOutput: "GaeaTaskOutput",
  GaeaJournalList: "GaeaJournalList",
  VerifyRecord: "GaeaVerifyRecord",
  RollbackRecord: "GaeaRollbackRecord",
  DocumentLint: "GaeaDocumentLint",
  PriceFetches: "GaeaPriceFetches",
  PriceFetchApply: "GaeaPriceFetchApply",
  PriceFetchIgnore: "GaeaPriceFetchIgnore",
  PriceHistory: "GaeaPriceHistory",
  SemanticSearch: "GaeaSemanticSearch",
  CostCompare: "GaeaCostCompare",
  CostCompose: "GaeaCostCompose",
  CostComposeApply: "GaeaCostComposeApply",
  CostInquirySave: "GaeaCostInquirySave",
  CostInquiryList: "GaeaCostInquiryList",
  CostInquiryDelete: "GaeaCostInquiryDelete",
  CostInquiryExpiring: "GaeaCostInquiryExpiring",
  CostInquiryAdjust: "GaeaCostInquiryAdjust",
  CostStageSave: "GaeaCostStageSave",
  CostStages: "GaeaCostStages",
  CostStageCompare: "GaeaCostStageCompare",
  CostStageDeviations: "GaeaCostStageDeviations",
  CostProjectSave: "GaeaCostProjectSave",
  CostProjectList: "GaeaCostProjectList",
  CostProjectGet: "GaeaCostProjectGet",
  CostProjectDelete: "GaeaCostProjectDelete",
  CostEstimateItemSave: "GaeaCostEstimateItemSave",
  CostEstimateItemDelete: "GaeaCostEstimateItemDelete",
  CostEstimateItems: "GaeaCostEstimateItems",
  CostEstimateVersionSave: "GaeaCostEstimateVersionSave",
  CostEstimateVersions: "GaeaCostEstimateVersions",
  CostEstimateSediment: "GaeaCostEstimateSediment",
  CostIndicators: "GaeaCostIndicators",
  CostAttribution: "GaeaCostAttribution",
  CostNoteSave: "GaeaCostNoteSave",
  CostNoteList: "GaeaCostNoteList",
  CostNoteDelete: "GaeaCostNoteDelete",
  CostNoteBumpRef: "GaeaCostNoteBumpRef",
  CostGraph: "GaeaCostGraph",
  UnifiedSearch: "GaeaUnifiedSearch",
  RouteIntent: "GaeaRouteIntent",
  RetrievalEvalRun: "GaeaRetrievalEvalRun",
  KnowledgeImportPreview: "GaeaKnowledgeImportPreview",
  KnowledgeImportAIParse: "GaeaKnowledgeImportAIParse",
  KnowledgeImportApply: "GaeaKnowledgeImportApply",
  KnowledgeHistory: "GaeaKnowledgeHistory",
  KnowledgeFindSimilar: "GaeaKnowledgeFindSimilar",
  KnowledgeExport: "GaeaKnowledgeExport",
  KnowledgeReview: "GaeaKnowledgeReview",
  KnowledgeMerge: "GaeaKnowledgeMerge",
  MemoryDuplicates: "GaeaMemoryDuplicates",
  MemoryMerge: "GaeaMemoryMerge",
  MemoryArchivedList: "GaeaMemoryArchivedList",
  MemoryCleanupArchived: "GaeaMemoryCleanupArchived",
  MemoryUnarchive: "GaeaMemoryUnarchive",
  MemoryUnarchiveBatch: "GaeaMemoryUnarchiveBatch",
  MemorySetRetentionDays: "GaeaMemorySetRetentionDays",
  FileIndexRebuild: "GaeaFileIndexRebuild",
  FileSemanticSearch: "GaeaFileSemanticSearch",
  ProfileResolveConflict: "GaeaProfileResolveConflict",
  KnowledgeGet: "GaeaKnowledgeGet",
  KnowledgeSave: "GaeaKnowledgeSave",
  KnowledgeDelete: "GaeaKnowledgeDelete",
  PickFiles: "GaeaPickFiles",
  // Go 侧方法名为 GetKeepWarm/SetKeepWarm/GetPreloadPlan/SetPreloadPlan（T6-10.3 对齐）。
  KeepWarmGet: "GetKeepWarm",
  KeepWarmSet: "SetKeepWarm",
  PreloadPlanGet: "GetPreloadPlan",
  PreloadPlanSet: "SetPreloadPlan",
  ModelSwitchEstimate: "GaeaModelSwitchEstimate",
} as const satisfies Record<string, string>;

// ── 错误归一化层（T6-1.2 前端错误可见性）────────────────────────────
// BridgeError 是所有绑定调用失败时归一出的结构化错误：code 机器可读
// （后端已带 code 时透传，否则 "<方法名>Error"），message 为人类可读原因
// （后端错误信息原文）。继承 Error 保证既有调用方的 e instanceof Error /
// e.message 判定不受影响（message 保留后端原文）。
export class BridgeError extends Error {
  code: string;
  constructor(code: string, message: string) {
    super(message);
    this.name = "BridgeError";
    this.code = code;
    // Error.message 默认不可枚举（JSON 序列化/结构化断言拿不到），显式
    // 重设为可枚举 own 属性，保证 { code, message } 结构对外稳定。
    Object.defineProperty(this, "message", { value: message, enumerable: true, writable: true, configurable: true });
  }
}

// normalizeError 把任意拒绝值归一为 BridgeError：已结构化（code/message）
// 的错误透传，Error 取 message，其余兜底字符串化。
function normalizeError(method: string, err: unknown): BridgeError {
  if (err instanceof BridgeError) return err;
  if (err && typeof err === "object") {
    const cand = err as { code?: unknown; message?: unknown };
    if (typeof cand.code === "string" && typeof cand.message === "string") {
      return new BridgeError(cand.code, cand.message);
    }
  }
  if (err instanceof Error) {
    return new BridgeError(`${method}Error`, err.message || String(err));
  }
  return new BridgeError(`${method}Error`, String(err ?? "unknown error"));
}

// invoke 是所有绑定调用的统一入口：失败时把错误归一为 BridgeError 并记录
// 到 gaea.log（LogFrontendError），再以同样的拒绝语义抛给调用方——调用方
// 原有的 .catch 行为契约不变（仍拿到 rejected promise + 错误值）。
function invoke(method: string, fn: (...args: unknown[]) => unknown, args: unknown[]): Promise<unknown> {
  return Promise.resolve()
    .then(() => fn(...args))
    .catch((err: unknown) => {
      const normalized = normalizeError(method, err);
      // 记录到 gaea.log；日志通道自身故障不向上抛，避免掩盖原始错误。
      // LogFrontendError 在 app proxy 里不套本层（见下），不会递归。
      logFrontendError(`[${normalized.code}] ${method} 失败: ${normalized.message}`);
      throw normalized;
    });
}

// logFrontendError 上报错误到 gaea.log；日志通道不可用（dev mock 外未注入
// 绑定）或自身失败时静默降级，绝不掩盖原始错误。
function logFrontendError(message: string): void {
  const lfe = app.LogFrontendError;
  if (typeof lfe !== "function") return;
  void Promise.resolve(lfe(message)).catch(() => {});
}

/** 解析单个绑定：按方法名路由到 live binding 或 dev mock（无空间门控的通用解析）。 */
function resolveBinding(prop: string | symbol): unknown {
  const target = realApp() ?? getMock();
  const key = (gaeaToGaea as Record<string, string>)[String(prop)] ?? String(prop);
  const rec = target as unknown as Record<string, unknown>;
  // 真实绑定按 Gaea 前缀查找；浏览器 mock 直接暴露同名字段，需回退。
  const v = rec[key] ?? rec[String(prop)];
  if (typeof v !== "function") return v;
  const bound = (v as (...a: unknown[]) => unknown).bind(target);
  // LogFrontendError 是错误上报通道自身，不套 invoke 归一化层，避免日志
  // 通道故障时无限递归；其余方法统一走 invoke。
  if (String(prop) === "LogFrontendError") return bound;
  return (...args: unknown[]) => invoke(String(prop), bound, args);
}

// app proxies each call to the live binding (or the dev mock only when truly
// outside the shell), so a late-injected window.go is picked up transparently.
export const app: AppBindings = new Proxy({} as AppBindings, {
  get(_t, prop) {
    return resolveBinding(prop);
  },
});

// ── S2.3 bridge 分面（docs/gaea-space-shell-design.md §7）────────────────
// 类型级门面：work/play 各自只暴露「所属空间 + shared + independent」的方法，
// play 页面引用 work 专属方法会 tsc 报错；运行时同样按 spaceBindings 门控
// （越界方法返回 undefined → TypeError，双保险）。sharedApp 只暴露 shared。
function createSpaceFacade<S extends "work" | "play">(space: S): GaeaFacetBySpace[S] {
  return new Proxy({} as GaeaFacetBySpace[S], {
    get(_t, prop) {
      if (prop === "then") return undefined; // 避免被误判为 Promise
      if (!isBindingAllowedInSpace(String(prop), space)) return undefined;
      return resolveBinding(prop);
    },
  });
}

/** 工位门面（work + shared + independent）——办公工作台专用。 */
export const workApp = createSpaceFacade("work");
/** 乐园门面（play + shared + independent）——轻语/小说/绘梦等页面专用。 */
export const playApp = createSpaceFacade("play");
/** 共用门面（仅 shared）——壳层/设置等两空间共用代码专用。 */
export const sharedApp: GaeaFacetBySpace["work"] & GaeaFacetBySpace["play"] = new Proxy(
  {} as GaeaFacetBySpace["work"] & GaeaFacetBySpace["play"],
  {
    get(_t, prop) {
      if (prop === "then") return undefined;
      if (!isSharedBinding(String(prop))) return undefined;
      return resolveBinding(prop);
    },
  },
);

// openExternal opens a URL in the system browser (so links in rendered markdown
// don't navigate the webview away from the app). Falls back to window.open in the
// browser dev mock.
export function openExternal(url: string): void {
  if (typeof window !== "undefined" && window.runtime?.BrowserOpenURL) {
    window.runtime.BrowserOpenURL(url);
  } else if (typeof window !== "undefined") {
    window.open(url, "_blank", "noopener");
  }
}

// ── 桥接归一（T6-10.6）─────────────────────────────────────────────────
// initBridge 原为 frontend/src/api/bridge.ts（S2-2 移动端 HTTP 桥接 + S2-3 旧
// 形态兼容层），T6-10.6 并入本文件，由 App.tsx 在模块作用域最早时机调用：
//   · Wails 原生环境：window.go.app 下是各板块门面（CoreB/OfficeB/...），补一个
//     window.go.app.App 兼容代理，旧调用点（window.go.app.App.Xxx()）按方法名路由；
//   · HTTP 环境（移动端/调试页，非 Wails）：把 window.go.app.App 挂为
//     fetch('/api/rpc') 的 RPC 代理（Bearer token 鉴权，token 来源见 httpToken.ts）。
// 本文件顶层的 app 代理在调用时经 realApp() 感知上述两种注入，无需额外适配。
import { getHttpToken } from "../../api/httpToken";

const API_BASE = "";

interface RPCResponse {
  result?: unknown;
  error?: string;
}

/** 是否运行在 Wails 原生环境：window.go.app 下存在任一板块门面绑定对象。 */
function isWailsNative(): boolean {
  const goApp = (window as unknown as { go?: { app?: Record<string, unknown> } }).go?.app;
  return !!goApp && typeof goApp === "object" && Object.keys(goApp).length > 0;
}

/** S2-3 兼容层：为旧调用点补 window.go.app.App 代理，按方法名路由到对应门面。 */
function ensureLegacyAppProxy(): void {
  const goApp = (window as unknown as { go?: { app?: Record<string, unknown> } }).go?.app;
  if (!goApp || typeof goApp !== "object") return;
  if (goApp.App) return;
  goApp.App = new Proxy(
    {},
    {
      get(_t, prop: string) {
        if (prop === "then") return undefined; // 避免被误判为 Promise
        for (const ns of Object.values(goApp)) {
          if (ns === goApp.App || ns === null || typeof ns !== "object") continue;
          const v = (ns as Record<string, unknown>)[prop];
          if (typeof v === "function") return (v as (...a: unknown[]) => unknown).bind(ns);
        }
        return undefined;
      },
    },
  );
}

/** 移动端 RPC 调用：POST /api/rpc，携带一次性 token（S2-2）。 */
async function rpcCall(method: string, ...args: unknown[]): Promise<unknown> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  const token = getHttpToken();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  const res = await fetch(`${API_BASE}/api/rpc`, {
    method: "POST",
    headers,
    body: JSON.stringify({ method, args }),
  });
  if (res.status === 401) {
    throw new Error("桥接鉴权失败：请携带正确的 token（__GAEA_HTTP_TOKEN / localStorage gaea_http_token）");
  }
  if (!res.ok) {
    throw new Error(`RPC 请求失败: ${res.status} ${res.statusText}`);
  }
  const data: RPCResponse = await res.json();
  if (data.error) {
    throw new Error(data.error);
  }
  return data.result;
}

/** HTTP 模式 App 代理：拦截所有方法调用并转发到 /api/rpc。 */
function createAppProxy(): Record<string, (...args: unknown[]) => Promise<unknown>> {
  return new Proxy(
    {},
    {
      get(_target, prop: string) {
        return (...args: unknown[]) => rpcCall(prop, ...args);
      },
    },
  ) as Record<string, (...args: unknown[]) => Promise<unknown>>;
}

/**
 * 初始化桥接层（App.tsx 模块作用域最早时机调用，幂等）：
 *  - Wails 环境：补齐 window.go.app.App 旧形态兼容代理（板块门面路由）；
 *  - HTTP 环境：创建 window.go.app.App 的 /api/rpc 代理。
 */
export function initBridge(): void {
  // 避免重复初始化
  if ((window as unknown as Record<string, unknown>).__bridge_initialized) return;
  (window as unknown as Record<string, unknown>).__bridge_initialized = true;

  // 显式 ?mock= 场景优先（评审 03-office-frontend.md 缺陷 8）：浏览器开发时
  // 用 URL 参数进入 mock 模式（approval/ask/compaction 等场景），不创建
  // RPC 代理——保持 window.go 为空，realApp() 返回 undefined，app 代理
  // 走 getMock()。此前 initBridge 在非 Wails 环境无条件创建 RPC 代理，
  // ?mock= 从未生效，审批/提问/压缩卡无法离线开发。
  const mockParam = new URLSearchParams(window.location.search).get("mock")?.trim().toLowerCase();
  if (mockParam && !isWailsNative()) {
    console.log(`[bridge] 浏览器 mock 模式（?mock=${mockParam}）`);
    return;
  }

  if (isWailsNative()) {
    ensureLegacyAppProxy();
    console.log("[bridge] Wails 原生环境，已就绪板块门面路由");
    return;
  }

  console.log("[bridge] 移动端 HTTP 模式，创建 RPC 代理");
  const w = window as unknown as { go?: { app?: Record<string, unknown> } };
  if (!w.go) w.go = {};
  if (!w.go.app) w.go.app = {};
  w.go.app.App = createAppProxy();
  // HTTP 模式也补齐板块门面代理（CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB）：
  // wailsjsCompat 从 '../wailsjs/go/app/<门面>' 直调 window.go.app.<门面>.<方法>（如
  // useVoiceChat 的 App.VoiceStop），HTTP 桥接下这些门面不存在会抛
  // "Cannot read properties of undefined (reading 'VoiceStop')"。RPC 端点按方法名路由，
  // 与门面无关，故每门面挂同一 RPC 代理即可（方法名冲突时后注册者生效，方法与门面一一对应无冲突）。
  const FACADES = ["CoreB", "OfficeB", "MemoryB", "CostB", "ModelB", "VoiceB", "ChatB", "NovelB", "ImageB", "CharlibB"] as const;
  for (const ns of FACADES) {
    if (!w.go.app[ns]) w.go.app[ns] = createAppProxy();
  }
}

import {
  makeMockApp,
  mockSubscribe,
  mockTaskSubscribe,
  updaterListeners,
} from "./mock";

// ── 编译期绑定面漂移检查（T6-10.3）──────────────────────────────────────
// bindingNames 由 scripts/gen_bindings -names 生成（Go 侧全部导出绑定方法名），
// CI（scripts/check-bindings-drift.ps1）保证它与 Go 侧同步。这里用类型级双向断言
// 校验前端绑定面（AppBindings + gaeaToGaea）与 bindingNames 一致：
//
//   · 方向一 _CheckAppBindingsHasNoStray：AppBindings 声明并经 gaeaToGaea 映射后
//     实际调用的每个 Go 方法名必须真实存在于 bindingNames。Go 侧改名/删除方法，
//     或 gaeaToGaea 映射目标写错（历史上有 6 处：KeepWarm*/PreloadPlan*/AgentMode/
//     SummarizeFile/Subagent*），都会在此报错。
//   · 方向二 _CheckAppBindingsCoversAll：bindingNames 里每个方法名必须被
//     AppBindings 消费（含 gaeaToGaea 映射）或显式排除。Go 侧新增绑定而前端
//     未认领时在此报错（补 AppBindings/gaeaToGaea，或列入下方两个清单并注明理由）。
//
// S2-3「App 绑定面拆分」后 Go 侧方法分为两半：经 AppBindings 消费的 gaea UI
// 绑定面，以及 wailsjsCompat / window.go.app.App.* 直接调用的 legacy 绑定面
// （小说/聊天/语音/绘图/角色库/旧 store 等），后者全部列入 LegacySurfaceNames。
// 修复提示：Go 方法新增/改名/删除 → 先重新生成 bindingNames.ts，再按报错调整。
import { bindingNames } from "./bindingNames";
import { subscribeWailsEvent } from "./wailsEvents";

/** 泛型工具：T 必须是 never（空联合），否则编译错误。 */
type AssertNever<T extends never> = T;

/** Go 侧全部绑定方法名（与 bindingNames.ts 同步）。 */
type BindingName = (typeof bindingNames)[number];

/** AppBindings 声明的方法经 gaeaToGaea 映射后在 Go 侧实际调用的方法名集合。 */
type AppBindingTarget = {
  [K in keyof AppBindings]: K extends keyof typeof gaeaToGaea
    ? (typeof gaeaToGaea)[K]
    : K;
}[keyof AppBindings];

/** AppBindings mock-only：Go 侧无对应绑定方法（仅 dev mock 提供）。 */
type MockOnlyNames =
  | "Compact" // 无 Go 绑定；上下文压缩由后端会话事件驱动，无独立绑定
  | "SetSubagentTemperature" // 声明但 Go 侧从未实现（仅 GaeaSetSubagentEffort 存在）
  | "SetEffort" // 同上：Go 侧无 SetEffort，推理强度实际走 GaeaSetSubagentEffort
  | "SetSubagentModel"; // 同上：Go 侧无 SetSubagentModel，实际走 GaeaSetSubagentModelForSkill

/** legacy 绑定面：Go 侧存在但不经 AppBindings 消费（wailsjsCompat 直接调用）。 */
type LegacySurfaceNames =
  | "AddCustomEngine"
  | "AddOutlineNode"
  | "AnalyzeChapter"
  | "AnalyzeStyle"
  | "ApplyBranch"
  | "BrainCrossRefs"
  | "BrainSearch"
  | "BrainWrite"
  | "BrainstormBranches"
  | "BuildBacklinkIndex"
  | "BuildContextBudget"
  | "BuildRichContext"
  | "CancelCreateChapter"
  | "CancelImageGeneration"
  | "CharacterAssociate"
  | "CharacterAssociateTo"
  | "CharacterDelete"
  | "CharacterDissociate"
  | "CharacterDrawRandom"
  | "CharacterFillAll"
  | "CharacterGenerateFill"
  | "CharacterGeneratePortrait"
  | "CharacterGeneratePortraitWithRef"
  | "CharacterGenerateRandom"
  | "CharacterGet"
  | "CharacterImportProject"
  | "CharacterListByProject"
  | "CharacterSave"
  | "CharacterSetProjectState"
  | "CharacterSyncProject"
  | "Chat"
  | "ChatCharacter"
  | "ChatCharacterDetail"
  | "ChatGeneral"
  | "ChatImportTopic"
  | "ChatOutline"
  | "ChatOutlineNode"
  | "ChatSend"
  | "ChatStreamPlain"
  | "ChatTopicClear"
  | "ChatTopicCreate"
  | "ChatTopicDelete"
  | "ChatTopicExportMarkdown"
  | "ChatTopicRename"
  | "ChatTopicSetMode"
  | "ChatWorldview"
  | "CheckConsistency"
  | "CheckConsistencyDeep"
  | "CloseProject"
  | "CmdKEdit"
  | "ContinueOutline"
  | "CreateChapter"
  | "CreateProject"
  | "CreateSnapshot"
  | "DeleteCharacter"
  | "DeleteLorebookEntry"
  | "DeleteOrganization"
  | "DeleteOutlineNode"
  | "DeleteProject"
  | "DeleteRelationship"
  | "ExpandOutlineNode"
  | "ExportAll"
  | "ExportHTML"
  | "ExtractCharacterHeatmap"
  | "ExtractEmotionCurve"
  | "ExtractTimeline"
  | "FindLorebookTriggers"
  | "FindUnlinkedMentions"
  | "GaeaBenchmarkDetail"
  | "GaeaBenchmarkExport"
  | "GaeaBenchmarkList"
  | "GaeaBenchmarkStart"
  | "GaeaBenchmarkStreamProbe"
  | "GaeaCallTool"
  | "GaeaDataBackupCancel"
  | "GaeaDataBackupCreate"
  | "GaeaDataBackupInfo"
  | "GaeaDataBackupPending"
  | "GaeaDataBackupRestore"
  | "GaeaDataBackupRestoreResult"
  | "GaeaDataBackupRollback"
  | "GaeaEngines"
  | "GaeaGetUsdCnyRate"
  | "GaeaInit"
  | "GaeaModel"
  | "GaeaPermLevel"
  | "GaeaSaveSettings"
  | "GaeaSemanticIndexStatus"
  | "GaeaSetEngine"
  | "GaeaSetUsdCnyRate"
  | "GaeaSkills"
  | "GaeaTools"
  | "GaeaUsageOverview"
  | "GenerateCharacterPortrait"
  | "GenerateCharacters"
  | "GenerateDefaultCanvas"
  | "GenerateDiagram"
  | "GenerateFreeImage"
  | "GenerateMedia"
  | "GenerateOutlineWithDialogue"
  | "GenerateProjectCharacterFill"
  | "GenerateSceneIllustration"
  | "GenerateSingleCharacter"
  | "GetActiveASRModel"
  | "GetActiveEngine"
  | "GetActiveModel"
  | "GetActiveOCRModel"
  | "GetActiveTTSModel"
  | "GetAllEntityNames"
  | "GetAppInfo"
  | "GetBacklinks"
  | "GetBookData"
  | "GetBoardManifests" // 3.0 Step 2：板块清单经 wailsjsCompat 直接调用（gen_bindings 新增；前端接线见 boards/manifests.ts loadBoardManifests）
  | "CheckModuleIntegrity" // 3.0 Step 2：板块装配启动自检（Startup 内部调用，前端不经 AppBindings 消费）
  | "GetChapter"
  | "GetChapterBranch"
  | "GetChapterScenes"
  | "GetCharacters"
  | "GetChatVoiceModel"
  | "GetComfyUILoras"
  | "GetComfyUIStatus"
  | "GetComfyUITaskProgress"
  | "GetCompileTemplates"
  | "GetConfig"
  | "GetDashboard"
  | "GetDeepseekKeyStatus"
  | "GetGlmKeyStatus"
  | "GetEngineList"
  | "GetEngines"
  | "GetEngineFailover"
  | "GetEntityRelations"
  | "GetFeatureModel"
  | "GetFeatureModelEnabled"
  | "GetForeshadows"
  | "GetImageBackend"
  | "GetImageBackendConfig"
  | "GetImageBackendInfo"
  | "GetLoginStatus"
  | "GetLorebookEntries"
  | "GetModelCallStats"
  | "GetModelMonitor"
  | "GetModelRoute"
  | "GetNovelsDir"
  | "GetOfflineMode"
  | "GetOfficeLocal"
  | "GetOpencodeGoKeyStatus"
  | "GetOpencodeZenKeyStatus"
  | "GetOutlines"
  | "GetPortraitConfig"
  | "GetProjectInfo"
  | "GetSensitiveLocal"
  | "GetStats"
  | "GetStyleProfile"
  | "GetSystemStats"
  | "GetTTSConfig"
  | "GetTTSSpeakers"
  | "GetTTSStatus"
  | "GetVoicePipelineConfig"
  | "GetWorldMapImage"
  | "GetWorldview"
  | "GetWorldviewSections"
  | "HerdsmanHealth"
  | "HerdsmanLaunchPresets"
  | "HerdsmanModelCatalog"
  | "HerdsmanModelDownload"
  | "HerdsmanModelStart"
  | "HerdsmanModelStats"
  | "HerdsmanModelStop"
  | "HerdsmanModelUninstall"
  | "HerdsmanProbe"
  | "HerdsmanSecurityCheck"
  | "ImportNovelBook"
  | "ImportStyleProfile"
  | "InjectMemories"
  | "IsProjectV4"
  | "ListProjects"
  | "ListSkills"
  | "ListSnapshots"
  | "LocalTranslate"
  | "Login"
  | "Logout"
  | "MainBrainChat"
  | "MergeCharacters"
  | "MigrateProjectToV4"
  | "NovelReadingAsk"
  | "NovelSearch"
  | "OfficeCancelJob"
  | "OfficeExecute"
  | "OfficeGetJobState"
  | "OfficeGetMode"
  | "OfficeIsTask"
  | "OfficeListFolder"
  | "OfficeReadFile"
  | "OfficeSetMode"
  | "OpenImageSaveDir"
  | "OpenNovelImagesDir"
  | "OpenProject"
  | "ParseLinks"
  | "QueryEntities"
  | "QuickBrainstormBranches"
  | "RefreshEngineModels"
  | "RemoveCustomEngine"
  | "ReorderScenes"
  | "ResetModelCallStats"
  | "RestoreSnapshot"
  | "ReviewBook"
  | "RunModule"
  | "SaveAllWorldviewSections"
  | "SaveChapterBranchContent"
  | "SaveChapterContent"
  | "SaveCharacter"
  | "SaveCharacters"
  | "SaveCharactersBatch"
  | "SaveConfig"
  | "SaveEngine"
  | "SaveForeshadows"
  | "SaveLorebookEntry"
  | "SaveOrganization"
  | "SaveOutlineNode"
  | "SaveRelationship"
  | "SaveScene"
  | "SaveTTSConfig"
  | "SaveToken"
  | "SaveWorldMapImage"
  | "SaveWorldview"
  | "SaveWorldviewSection"
  | "Search"
  | "SearchMemories"
  | "SetActiveASRModel"
  | "SetActiveEngine"
  | "SetActiveOCRModel"
  | "SetActiveTTSModel"
  | "SetCharacterPortrait"
  | "SetChatVoiceModel"
  | "SetDeepseekKey"
  | "SetGlmEndpoint"
  | "SetGlmKey"
  | "SetDistFS"
  | "SetEngineDefaultModel"
  | "SetEngineFailover"
  | "SetFeatureModel"
  | "SetFeatureModelEnabled"
  | "SetImageBackend"
  | "SetOfflineMode"
  | "SetOfficeLocal"
  | "SetOpencodeGoKey"
  | "SetOpencodeZenKey"
  | "SetPortraitConfig"
  | "SetPromptFS"
  | "SetSensitiveLocal"
  | "Shutdown"
  | "StartComfyUI"
  | "StartLocalTTSService"
  | "StartTTSServer"
  | "Startup"
  | "StopComfyUI"
  | "StopTTSServer"
  | "SyncEntityDB"
  | "TTSSpeak"
  | "TTSSpeakBase64"
  | "TTSSpeakBase64WithParams" // v4.3d：带风格/情绪参数合成（wailsjsCompat 直调）
  | "TTSSpeakStreaming"
  | "TestEngineConnection"
  | "ToggleOrgMember"
  | "VoiceApplySettings"
  | "VoiceCancelTTS"
  | "VoiceChatText"
  | "VoiceGetSettings"
  | "VoiceGetState"
  | "VoiceHealth"
  | "VoicePlaybackDone"
  | "VoicePushAudio"
  | "VoiceRestartService"
  | "VoiceSetInputChannel"
  | "VoiceSetMode"
  | "VoiceSetPTTActive"
  | "VoiceStart"
  | "VoiceStop"
  | "WhisperChat"
  | "WhisperChatWithSearch"
  | "WhisperClearSession"
  | "WhisperDeleteFact"
  | "WhisperGetConfig"
  | "WhisperGetEngine"
  | "WhisperGetEngines"
  | "WhisperGetFacts"
  | "WhisperGetImageModel"
  | "UpdateCustomEngine"
  | "WhisperGetModel"
  | "WhisperGetState"
  | "WhisperGetTraces"
  | "WhisperSetEngine"
  | "WhisperSetImageModel"
  | "WhisperSetModel"
  | "WhisperTaskPlanResume"
  | "WhisperTaskPlanStatus"
  | "WhisperUpdateFact"
  | "WhisperWebSearch";

/** 显式排除 = mock-only + legacy 绑定面。 */
type ExcludeNames = MockOnlyNames | LegacySurfaceNames;

// 方向一：AppBindings 声明的每个绑定（映射后）必须真实存在于 Go 绑定清单。
// 报错 → Go 侧方法被改名/删除，或 gaeaToGaea 映射目标写错。
export type _CheckAppBindingsHasNoStray = AssertNever<
  Exclude<AppBindingTarget, BindingName | ExcludeNames>
>;

// 方向二：Go 绑定清单的每个方法名必须被 AppBindings 消费或显式排除。
// 报错 → Go 侧新增绑定无人认领（补 AppBindings/gaeaToGaea 或 ExcludeNames）。
export type _CheckAppBindingsCoversAll = AssertNever<
  Exclude<BindingName, AppBindingTarget | ExcludeNames>
>;
