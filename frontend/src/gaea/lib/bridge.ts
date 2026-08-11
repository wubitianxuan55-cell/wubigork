// bridge is the single seam between the React app and the Go kernel. In the Wails
// shell it calls the bound App methods (window.go.main.App.*) and subscribes to
// the runtime event stream (window.runtime.EventsOn). In a plain browser (`pnpm
// dev` outside the shell) those globals are absent, so it falls back to a mock
// that streams a canned turn through the same contract — letting the whole UI be
// developed and laid out without rebuilding the Go side.

import type {
  BalanceInfo,
  WorkspaceChangeView,
  CapabilitiesView,
  CheckpointMeta,
  CommandInfo,
  ContextInfo,
  CrossEmbedInput,
  CrossEmbedResult,
  DirEntry,
  ExportDeliverableInput,
  ExportDeliverableResult,
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
  MemorySuggestion,
  MemorySuggestionsView,
  MemoryView,
  OfficeEditResult,
  XlsxEditResult,
  SkillSuggestion,
  TaskTemplate,
  TabMeta,
  Meta,
  ModelInfo,
  ProviderView,
  QuestionAnswer,
  SessionMeta,
  SettingsView,
  SkillCaptureInput,
  SkillCaptureResult,
  SlashArgsResult,
  UpdateInfo,
  UpdateProgress,
  WireEvent,
  WorkspaceView,
  MemoryHubOverview,
  ProfileFactView,
  WhisperEpisodeView,
  WhisperMemoryView,
  MemoryGraphView,
  WorkspaceSearchHit,
  CostSummary,
  CostEntry,
  CostCategory,
  CostImportPreview,
  PriceSource,
  PriceFetchRecord,
  PriceHistory,
  SemanticHitView,
  KnowledgeImportPreview,
  KnowledgeHistoryView,
  SimilarView,
  MemoryDuplicateView,
  FileIndexStatus,
  FileSemanticHit,
} from "./types";

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
  Approve(id: string, allow: boolean, session: boolean): Promise<void>;
  AnswerQuestion(id: string, answers: QuestionAnswer[]): Promise<void>;
  SetAgentMode(mode: string): Promise<void>;
  AgentMode(): Promise<string>;
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
  ResumeSession(path: string): Promise<HistoryMessage[]>;
  DeleteSession(path: string): Promise<void>;
  RenameSession(path: string, title: string): Promise<void>;
  // Workspace: open a folder chooser and switch to that project (fresh session);
  // returns the chosen path, or "" if cancelled.
  ListWorkspaces(): Promise<WorkspaceView[]>;
  PickWorkspace(): Promise<string>;
  SwitchWorkspace(path: string): Promise<string>;
  ContextUsage(): Promise<ContextInfo>;
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
  // OfficeEditText 框选即改：按指令生成选中文本的替换；DocxApplyEdit 以修订模式
  // （w:del+w:ins）写入 docx 并返回更新后的预览。
  OfficeEditText(selectedText: string, instruction: string): Promise<OfficeEditResult>;
  DocxApplyEdit(rel: string, selectedText: string, replacement: string): Promise<PreviewResult>;
  // DocxAcceptChanges 接受/拒绝 gaea 的待处理修订，返回更新预览。
  DocxAcceptChanges(rel: string, accept: boolean): Promise<PreviewResult>;
  // XlsxEdit 单元格级操作：上下文 → AI 规划 → excelize 执行 + LibreOffice 重算 →
  // 返回更新预览。
  XlsxEdit(rel: string, sheet: string, instruction: string, selection: string): Promise<XlsxEditResult>;
  // XlsxSetCell 直接写单元格（Excel 式双击编辑）：写值/公式 + LibreOffice 重算。
  XlsxSetCell(rel: string, sheet: string, ref: string, value: string): Promise<XlsxEditResult>;
  // XlsxRecalc 手动重算全部公式（LibreOffice）并返回更新预览。
  XlsxRecalc(rel: string): Promise<XlsxEditResult>;
  // XlsxRowOps 行级操作：insert_before / insert_after / delete（基于选中单元格所在行）。
  XlsxRowOps(rel: string, sheet: string, action: string, ref: string): Promise<XlsxEditResult>;
  // XlsxColOps 列级操作：insert_before / insert_after / delete（基于选中单元格所在列）。
  XlsxColOps(rel: string, sheet: string, action: string, ref: string): Promise<XlsxEditResult>;
  // ExportDeliverable 统一交付出口：受控 Markdown → docx/pptx/xlsx/md。
  ExportDeliverable(input: ExportDeliverableInput): Promise<ExportDeliverableResult>;
  // CrossEmbed 跨应用联动：xlsx 数据 → 图表 → 嵌入 docx/pptx。
  CrossEmbed(input: CrossEmbedInput): Promise<CrossEmbedResult>;
  OpenWorkspacePath(rel: string): Promise<void>;
  // WorkspaceChanges returns files modified during this session by the agent.
  WorkspaceChanges(): Promise<WorkspaceChangeView[]>;
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
  MemorySuggestions(): Promise<MemorySuggestionsView>;
  // LogFrontendError 记录前端错误/主线程卡死诊断到 gaea.log。
  LogFrontendError(message: string): Promise<void>;
  AcceptMemorySuggestion(candidate: MemorySuggestion): Promise<string>;
  AcceptSkillSuggestion(candidate: SkillSuggestion): Promise<string>;
  SelectTab(tabID: string): Promise<void>;
  TabMeta(): Promise<TabMeta[]>;
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
  // WhisperEpisodes 聊天情节记忆（hermes.db，时间倒序）。
  WhisperEpisodes(): Promise<WhisperEpisodeView[]>;
  // WhisperExportArchive 导出聊天记忆归档（hermes.db → Markdown 分目录），返回文件数。
  WhisperExportArchive(dir: string): Promise<number>;
  // PickDirectory 系统目录选择对话框，返回所选目录（取消返回空串）。
  PickDirectory(): Promise<string>;
  MemoryGraph(): Promise<MemoryGraphView>;
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
  CostImportApply(rows: CostEntry[]): Promise<number>;
  // 多级分类：分类树（含计数）、新建/改名、删除（有子节点或条目时拒绝）。
  CostCategories(): Promise<CostCategory[]>;
  CostCategorySave(parentId: number, name: string, sort: number, id: number): Promise<number>;
  CostCategoryDelete(id: number): Promise<void>;
  // ── 价格源（定时抓取价格更新）──
  PriceSources(): Promise<PriceSource[]>;
  PriceSourceSave(src: PriceSource): Promise<void>;
  PriceSourceDelete(id: string): Promise<void>;
  // PriceFetch 立即抓取价格源，返回待确认候选（存 pending 记录）。
  PriceFetch(id: string): Promise<PriceFetchRecord>;
  // PriceFetchAll 一键抓取全部启用的价格源（并发），返回成功数 + 失败汇总。
  PriceFetchAll(): Promise<[number, string]>;
  PriceFetches(): Promise<PriceFetchRecord[]>;
  // PriceFetchApply 确认发布抓取结果（按标题选择），返回写入条数。
  PriceFetchApply(fetchId: string, titles: string[]): Promise<number>;
  PriceFetchIgnore(fetchId: string): Promise<void>;
  // PriceHistory 返回某成本条目的价格历史（新→旧）。
  PriceHistory(name: string): Promise<PriceHistory[]>;
  // SemanticSearch 跨库统一语义检索（成本/知识/办公记忆，本地 bge-m3）。
  SemanticSearch(query: string): Promise<SemanticHitView[]>;
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
  // ── 工作区文件语义索引 ──
  FileIndexRebuild(): Promise<FileIndexStatus>;
  FileSemanticSearch(query: string, topN: number): Promise<FileSemanticHit[]>;
  // 画像冲突裁决
  ProfileResolveConflict(name: string, prefer: string): Promise<void>;
  KnowledgeGet(name: string): Promise<KnowledgeEntry | null>;
  KnowledgeSave(entry: KnowledgeSaveRequest): Promise<void>;
  KnowledgeDelete(name: string): Promise<void>;
  // Cost database panel.
  // PickFiles opens a native file dialog and imports selected files.
  PickFiles(): Promise<FilePickResult[]>;
}

// Window 类型由 gaea 的 src/types/wails.d.ts 统一声明（go.app.App + runtime）。
// gaeaW 在此不重复声明，避免覆盖 gaea 的 runtime（EventsOff）类型。

// onEvent subscribes to the agent's typed event stream; returns an unsubscribe.
export function onEvent(cb: (e: WireEvent) => void): () => void {
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    window.runtime.EventsOn(EVENT_CHANNEL, (payload) => cb(payload as WireEvent));
    return () => window.runtime?.EventsOff?.(EVENT_CHANNEL);
  }
  return mockSubscribe(cb);
}

// Must match desktop/app.go's eventChannel constant.
const EVENT_CHANNEL = "gaea-event";

// Resolve the Wails binding at CALL time, not module-load time: in dev the Wails
// runtime can inject window.go AFTER this module first evaluates, so snapshotting
// once would pin the browser mock for the whole session (and show fake data — the
// dev mock's model list leaking into the real app was exactly this bug).
function realApp(): AppBindings | undefined {
  return typeof window !== "undefined" ? (window.go?.app?.App as unknown as AppBindings) : undefined;
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
const gaeaToGaea: Record<string, string> = {
  Submit: "GaeaSend",
  SubmitDisplay: "GaeaSend",
  Cancel: "GaeaCancel",
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
  ListSessions: "GaeaListSessions",
  ResumeSession: "GaeaResumeSession",
  DeleteSession: "GaeaDeleteSession",
  RenameSession: "GaeaRenameSession",
  ListWorkspaces: "GaeaListWorkspaces",
  PickWorkspace: "GaeaPickWorkspace",
  SwitchWorkspace: "GaeaSwitchWorkspace",
  ContextUsage: "GaeaContext",
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
  OfficeEditText: "GaeaOfficeEditText",
  DocxApplyEdit: "GaeaDocxApplyEdit",
  DocxAcceptChanges: "GaeaDocxAcceptChanges",
  XlsxEdit: "GaeaXlsxEdit",
  XlsxSetCell: "GaeaXlsxSetCell",
  XlsxRecalc: "GaeaXlsxRecalc",
  XlsxRowOps: "GaeaXlsxRowOps",
  XlsxColOps: "GaeaXlsxColOps",
  ExportDeliverable: "GaeaExportDeliverable",
  CrossEmbed: "GaeaCrossEmbed",
  OpenWorkspacePath: "GaeaOpenWorkspacePath",
  WorkspaceChanges: "GaeaWorkspaceChanges",
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
  MemorySuggestions: "GaeaMemorySuggestions",
  LogFrontendError: "GaeaLogFrontendError",
  AcceptMemorySuggestion: "GaeaAcceptMemorySuggestion",
  AcceptSkillSuggestion: "GaeaAcceptSkillSuggestion",
  SelectTab: "GaeaSelectTab",
  TabMeta: "GaeaTabMeta",
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
  SetSubagentTemperature: "GaeaSetSubagentTemperature",
  SetEffort: "GaeaSetEffort",
  SetSubagentEffort: "GaeaSetSubagentEffort",
  SetSubagentModel: "GaeaSetSubagentModel",
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
  WhisperEpisodes: "GaeaWhisperEpisodes",
  WhisperExportArchive: "GaeaWhisperExportArchive",
  PickDirectory: "GaeaPickDirectory",
  MemoryGraph: "GaeaMemoryGraph",
  CostList: "GaeaCostList",
  CostSearch: "GaeaCostSearch",
  CostGet: "GaeaCostGet",
  CostSave: "GaeaCostSave",
  CostDelete: "GaeaCostDelete",
  CostImportPreview: "GaeaCostImportPreview",
  CostImportAIParse: "GaeaCostImportAIParse",
  CostImportApply: "GaeaCostImportApply",
  CostCategories: "GaeaCostCategories",
  CostCategorySave: "GaeaCostCategorySave",
  CostCategoryDelete: "GaeaCostCategoryDelete",
  PriceSources: "GaeaPriceSources",
  PriceSourceSave: "GaeaPriceSourceSave",
  PriceSourceDelete: "GaeaPriceSourceDelete",
  PriceFetch: "GaeaPriceFetch",
  PriceFetchAll: "GaeaPriceFetchAll",
  PriceFetches: "GaeaPriceFetches",
  PriceFetchApply: "GaeaPriceFetchApply",
  PriceFetchIgnore: "GaeaPriceFetchIgnore",
  PriceHistory: "GaeaPriceHistory",
  SemanticSearch: "GaeaSemanticSearch",
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
  FileIndexRebuild: "GaeaFileIndexRebuild",
  FileSemanticSearch: "GaeaFileSemanticSearch",
  ProfileResolveConflict: "GaeaProfileResolveConflict",
  KnowledgeGet: "GaeaKnowledgeGet",
  KnowledgeSave: "GaeaKnowledgeSave",
  KnowledgeDelete: "GaeaKnowledgeDelete",
  PickFiles: "GaeaPickFiles",
};

// app proxies each call to the live binding (or the dev mock only when truly
// outside the shell), so a late-injected window.go is picked up transparently.
export const app: AppBindings = new Proxy({} as AppBindings, {
  get(_t, prop) {
    const target = realApp() ?? getMock();
    const key = gaeaToGaea[String(prop)] ?? String(prop);
    const rec = target as unknown as Record<string, unknown>;
    // 真实绑定按 Gaea 前缀查找；浏览器 mock 直接暴露同名字段，需回退。
    const v = rec[key] ?? rec[String(prop)];
    return typeof v === "function" ? (v as (...a: unknown[]) => unknown).bind(target) : v;
  },
});

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

import {
  makeMockApp,
  mockSubscribe,
  updaterListeners,
} from "./mock";

// ── compile-time drift check ──────────────────────────────────────────────
// _CheckGenToApp errors when a generated Go method has no TS counterpart in
// AppBindings. Fix: add the missing method to AppBindings, then `pnpm typecheck`.
// Methods intentionally excluded from the frontend can be listed in the Exclude
// union to silence the check.
// Dev-time type checks (regenerate wails bindings after Go API changes):
// type _CheckGenToApp = AssertNever<Exclude<keyof typeof GeneratedApp, keyof AppBindings | "QuitApp" | "ShowWindow" | "SetBypass" | "SetAgentMode" | "PermLevel" | "SearchSpecs">>;
// export type _CheckGenToApp = AssertNever<Exclude<keyof typeof GeneratedApp, keyof AppBindings | "QuitApp" | "ShowWindow" | "SetBypass" | "SetAgentMode" | "PermLevel" | "SearchSpecs">>;
