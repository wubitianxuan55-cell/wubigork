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
  MemoryArchivedPage,
  MemorySuggestion,
  MemorySuggestionsView,
  MemoryView,
  OfficeEditResult,
  XlsxEditResult,
  SkillSuggestion,
  TaskTemplate,
  Meta,
  ModelInfo,
  ProviderView,
  QuestionAnswer,
  Requirement,
  SessionMeta,
  SessionStatsView,
  ProjectGroup,
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
  ProgrammingWebLogTail,
  ProgrammingWebPreflight,
  ProgrammingWebStatus,
  WhisperEpisodeView,
  WhisperMemoryView,
  MemoryGraphView,
  WorkspaceSearchHit,
  CostSummary,
  CostEntry,
  CostCategory,
  CostImportPreview,
  CostCompareRow,
  PriceSource,
  PriceFetchRecord,
  PriceHistory,
  SemanticHitView,
  UnifiedSearchView,
  RetrievalEvalReport,
  KnowledgeImportPreview,
  KnowledgeHistoryView,
  SimilarView,
  MemoryDuplicateView,
  FileIndexStatus,
  FileSemanticHit,
  TaskView,
  ModelSwitchEstimate,
} from "./types";
// chat 板块契约类型来自 wails 生成物（wails build 自动生成，勿手改生成物本身）。
// AppBindings 只做类型标注：ChatTopicsList/ChatMessagesList Go 侧为
// ([]chat.Topic, error) / ([]chat.Message, error)，Wails 绑定后失败呈现为
// rejected promise，这里按「[数据, 错误]」元组形态标注。
import type { app as AppModels, chat } from "../../../wailsjs/go/models";

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
  // GaeaRunning 返回办公引擎当前是否真的在跑（看门狗校准用）。
  GaeaRunning(): Promise<boolean>;
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
  // Requirement 是会话「任务目标」：读取 / 设置（空文本清除）/ 标记验收 /
  // 验收清单增删改勾选 / 自动追踪开关（开启后写入 agent goal gate）。
  Requirement(path: string): Promise<Requirement>;
  SetRequirement(path: string, text: string): Promise<void>;
  SetRequirementDone(path: string, done: boolean): Promise<void>;
  AddRequirementItem(path: string, text: string): Promise<void>;
  SetRequirementItem(path: string, index: number, text: string): Promise<void>;
  RemoveRequirementItem(path: string, index: number): Promise<void>;
  SetRequirementItemDone(path: string, index: number, done: boolean): Promise<void>;
  SetRequirementAutoPursue(path: string, on: boolean): Promise<void>;
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
  // UnifiedSearch 跨库统一检索一次调用：工作区关键词命中（topN 条）+ 语义跨库命中。
  UnifiedSearch(query: string, topN?: number): Promise<UnifiedSearchView>;
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
  // ── 办公记忆归档生命周期（T6-8.2）──
  // MemoryArchivedList 分页列出归档（含超过 90 天的硬删除候选）；
  // MemoryCleanupArchived 硬删除归档超过 90 天的事实，返回删除条数（无超期返回 0）。
  MemoryArchivedList(limit: number, offset: number): Promise<MemoryArchivedPage>;
  MemoryCleanupArchived(): Promise<number>;
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
  // TaskRetry 重试失败/已取消的任务。任务实时进度经 onTaskEvent 推送。
  TaskList(): Promise<TaskView[]>;
  TaskCancel(id: string): Promise<void>;
  TaskRetry(id: string): Promise<void>;
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
      const key = gaeaToGaea[String(prop)] ?? String(prop);
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
// every task status/progress change pushes the latest TaskView. Returns an
// unsubscribe. Falls back to the mock stream outside the Wails shell.
export function onTaskEvent(cb: (t: TaskView) => void): () => void {
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    window.runtime.EventsOn("gaea-task", (payload) => cb(payload as TaskView));
    return () => window.runtime?.EventsOff?.("gaea-task");
  }
  return mockTaskSubscribe(cb);
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
  Approve: "GaeaApprove",
  AnswerQuestion: "GaeaAnswer",
  // Go 侧为 GaeaAgentMode（T6-10.3 对齐）。
  AgentMode: "GaeaAgentMode",
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
  Requirement: "GaeaRequirement",
  SetRequirement: "GaeaSetRequirement",
  SetRequirementDone: "GaeaSetRequirementDone",
  AddRequirementItem: "GaeaAddRequirementItem",
  SetRequirementItem: "GaeaSetRequirementItem",
  RemoveRequirementItem: "GaeaRemoveRequirementItem",
  SetRequirementItemDone: "GaeaSetRequirementItemDone",
  SetRequirementAutoPursue: "GaeaSetRequirementAutoPursue",
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
  TaskCancel: "GaeaTaskCancel",
  TaskRetry: "GaeaTaskRetry",
  PriceFetches: "GaeaPriceFetches",
  PriceFetchApply: "GaeaPriceFetchApply",
  PriceFetchIgnore: "GaeaPriceFetchIgnore",
  PriceHistory: "GaeaPriceHistory",
  SemanticSearch: "GaeaSemanticSearch",
  CostCompare: "GaeaCostCompare",
  UnifiedSearch: "GaeaUnifiedSearch",
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

// app proxies each call to the live binding (or the dev mock only when truly
// outside the shell), so a late-injected window.go is picked up transparently.
export const app: AppBindings = new Proxy({} as AppBindings, {
  get(_t, prop) {
    const target = realApp() ?? getMock();
    const key = gaeaToGaea[String(prop)] ?? String(prop);
    const rec = target as unknown as Record<string, unknown>;
    // 真实绑定按 Gaea 前缀查找；浏览器 mock 直接暴露同名字段，需回退。
    const v = rec[key] ?? rec[String(prop)];
    if (typeof v !== "function") return v;
    const bound = (v as (...a: unknown[]) => unknown).bind(target);
    // LogFrontendError 是错误上报通道自身，不套 invoke 归一化层，避免日志
    // 通道故障时无限递归；其余方法统一走 invoke。
    if (String(prop) === "LogFrontendError") return bound;
    return (...args: unknown[]) => invoke(String(prop), bound, args);
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
  | "SetAgentMode" // 无 Go 绑定；mock 返回固定模式（develop），前端未调用
  | "Compact" // 无 Go 绑定；上下文压缩由后端会话事件驱动，无独立绑定
  | "SetSubagentTemperature" // 声明但 Go 侧从未实现（仅 GaeaSetSubagentEffort 存在）
  | "SetEffort" // 同上：Go 侧无 SetEffort，推理强度实际走 GaeaSetSubagentEffort
  | "SetSubagentModel"; // 同上：Go 侧无 SetSubagentModel，实际走 GaeaSetSubagentModelForSkill

/** legacy 绑定面：Go 侧存在但不经 AppBindings 消费（wailsjsCompat 直接调用）。 */
type LegacySurfaceNames =
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
  | "CharacterGenerateRandom"
  | "CharacterGet"
  | "CharacterImportProject"
  | "CharacterList"
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
  | "GetEngineList"
  | "GetEngines"
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
  | "SetDistFS"
  | "SetEngineDefaultModel"
  | "SetFeatureModel"
  | "SetFeatureModelEnabled"
  | "SetImageBackend"
  | "SetOpencodeGoKey"
  | "SetOpencodeZenKey"
  | "SetPortraitConfig"
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
  | "WhisperAssistantDelete"
  | "WhisperAssistantList"
  | "WhisperAssistantSave"
  | "WhisperChat"
  | "WhisperChatWithSearch"
  | "WhisperClearSession"
  | "WhisperDeleteFact"
  | "WhisperGetConfig"
  | "WhisperGetEngine"
  | "WhisperGetEngines"
  | "WhisperGetFacts"
  | "WhisperGetImageModel"
  | "WhisperGetModel"
  | "WhisperGetPersonalities"
  | "WhisperGetState"
  | "WhisperGetTraces"
  | "WhisperSetEngine"
  | "WhisperSetImageModel"
  | "WhisperSetModel"
  | "WhisperTaskPlanResume"
  | "WhisperTaskPlanStatus"
  | "WhisperUpdateFact"
  | "WhisperWebSearch"
  | "WhisperWeixinGetQR"
  | "WhisperWeixinQRStatus"
  | "WhisperWeixinQRStatusWithCode"
  | "WhisperWeixinStatus";

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
