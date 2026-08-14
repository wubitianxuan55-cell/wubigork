// Wire contract — mirrors desktop/wire.go (itself mirroring internal/serve/wire.go).
// One event channel carries every kind; `kind` discriminates the payload.

export type EventKind =
  | "turn_started"
  | "reasoning"
  | "text"
  | "message"
  | "tool_dispatch"
  | "tool_result"
  | "usage"
  | "notice"
  | "phase"
  | "approval_request"
  | "ask_request"
  | "turn_done"
  | "compaction_started"
  | "compaction_done"
  | "preview_progress";

// WirePreviewProgress 是预览（扫描件 PDF OCR）的逐页进度。
export interface WirePreviewProgress {
  path: string;
  done: number;
  total: number;
}

export interface WireCompaction {
  trigger?: string; // "auto" | "manual"
  messages?: number; // done: how many messages were folded into the summary
  summary?: string; // done: the briefing (empty on an aborted pass)
  archive?: string; // done: archive path, if any
  quality?: string; // V3.2: post-hoc quality assessment (human-readable)
}

export interface WireTool {
  id?: string;
  name: string;
  args?: string;
  output?: string;
  err?: string;
  recoverable?: boolean; // true when agent can fix this on next turn (bad args, wrong file, etc.)
  readOnly: boolean;
  truncated?: boolean;
  partial?: boolean; // an early dispatch (name only) — a full one with args follows
  parentId?: string; // set on a sub-agent's calls — the parent `task` call's id
}

export interface WireUsage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  cacheHitTokens: number;
  cacheMissTokens: number;
  reasoningTokens?: number;
  // Session-cumulative cache tokens — the status bar shows the aggregate
  // hit-rate (Σhit/Σ(hit+miss)), steadier than the single-turn cacheHitTokens.
  sessionCacheHitTokens: number;
  sessionCacheMissTokens: number;
  turn?: number; // 会话 API 调用轮次，由后端 AgentRunner 维护
  costUsd?: number;
  source?: string; // "main" | "subagent"
}

// GaeaReloadResult 是办公引擎热加载的结果摘要（工具/技能数量）。
export interface GaeaReloadResult {
  tools: number;
  skills: number;
}

export interface WireApproval {
  id: string;
  tool: string;
  subject: string;
}

export interface WireAskOption {
  label: string;
  description?: string;
}

export interface WireAskQuestion {
  id: string;
  header?: string;
  prompt: string;
  options: WireAskOption[];
  multi?: boolean;
}

export interface WireAsk {
  id: string;
  questions: WireAskQuestion[];
  // plan 存在时，前端渲染结构化「开工计划卡片」而非纯文本问题。
  plan?: WirePlan;
}

export interface WirePlanStep {
  title: string;
  detail?: string;
  resources?: string[];
  tools?: string[];
  deliverable?: string;
}

export interface WirePlan {
  goal: string;
  steps: WirePlanStep[];
  questions?: string[];
}

// QuestionAnswer is the reply for one question, sent back via AnswerQuestion.
export interface QuestionAnswer {
  questionId: string;
  selected: string[];
}

export interface WireEvent {
  kind: EventKind;
  text?: string;
  reasoning?: string;
  level?: "info" | "warn";
  tool?: WireTool;
  usage?: WireUsage;
  approval?: WireApproval;
  ask?: WireAsk;
  compaction?: WireCompaction;
  progress?: WirePreviewProgress;
  path?: string; // preview_progress: 当前预览文件（冗余，便于前端过滤）
  err?: string;
}

// Bound-method payloads (desktop/app.go).
export interface HistoryMessage {
  role: string;
  content: string;
  // 工具事件还原（恢复会话后过程卡/变更面板仍可见）
  toolName?: string;
  toolArgs?: string;
  toolId?: string;
  toolOutput?: string;
}

// CheckpointMeta is one rewind point (a user turn) for the rewind UI.
export interface CheckpointMeta {
  turn: number;
  prompt: string;
  files: string[];
  time: number; // unix ms
}

// SessionMeta is one saved session for the history panel.
export interface SessionMeta {
  path: string;
  preview: string;
  title?: string; // user-chosen name; falls back to preview when empty
  turns: number;
  modTime: number; // unix milliseconds
  current: boolean;
  pinned?: boolean; // 置顶会话排在同项目会话最前
  hasRequirement?: boolean; // 会话锚定了任务目标（从需求到验收）
  requirementDone?: boolean; // 任务目标已标记验收
  archived?: boolean; // 已归档（在 <sessions>/archive/ 下，可恢复）
  // interrupted=true 表示上次运行中断未完成；恢复会话时后端注入「上次会话中断」
  // 摘要提示并清除该标记（可选字段：后端始终返回，前端对缺失按 false 处理）。
  interrupted?: boolean;
}

// ProjectGroup 是侧边栏「项目」分组：一个工作区 + 它的会话列表。
// 由后端 GaeaListProjectSessions 聚合（当前工作区在前，其余为最近打开过的）。
export interface ProjectGroup {
  path: string;
  name: string;
  current: boolean;
  sessions: SessionMeta[];
  archived: SessionMeta[]; // 已归档会话（项目内「已归档」分组）
  modTime: number; // 分组内最近会话时间（unix milliseconds）
}

// Requirement 是会话的「任务目标」（从需求到验收工作流）：
// 记录目标文本与验收状态，随会话持久化。
export interface Requirement {
  text: string;
  done: boolean;
  updatedAt: number;
}

export interface WorkspaceChangeView {
  path: string;
  added: number;
  removed: number;
}

export interface WorkspaceView {
  path: string;
  name: string;
  current: boolean;
}

export interface ContextInfo {
  used: number;
  window: number;
}

export interface Meta {
  label: string;
  subagentLabel?: string;
  ready: boolean;
  startupErr?: string;
  eventChannel: string;
  cwd: string;
  bypass?: boolean; // YOLO mode on (auto-approve every tool call)
  permLevel?: string; // "ask"|"auto"|"yolo"
  agentMode?: string; // "explore"|"develop"|"orchestrate"
}

// PermLevel controls permission strictness for tool execution.
// ask: prompt before writes (default); auto: allow writes; yolo: skip all approval.
export type PermLevel = "ask" | "auto" | "yolo";

export interface CommandInfo {
  name: string; // without the leading slash
  description: string;
  hint?: string;
  kind: "builtin" | "custom" | "mcp" | "skill";
}

export interface DirEntry {
  name: string;
  isDir: boolean;
}

export interface FileSearchHit {
  path: string; // 工作区相对路径（/ 分隔）
  name: string;
  isDir: boolean;
  size?: number;
}

// WorkspaceSearchHit 是工作区全文搜索的一条命中（轻量 RAG）。
export interface WorkspaceSearchHit {
  path: string;
  name: string;
  size: number;
  modTime: number;
  score: number;
  snippet: string;
  truncated?: boolean;
  skipped?: string;
}

// GaeaSummaryResult 是资料「摘要」操作的结果（map-reduce 分块摘要）。
export interface GaeaSummaryResult {
  path: string;
  totalPages: number;
  chars: number;
  chunks: number;
  summary: string;
}

// TaskTemplate 是预置办公任务模板（欢迎页「任务模板」区 + slash 命令）。
export interface TaskTemplate {
  name: string;
  title: string;
  description: string;
  prompt: string;
}

export interface FilePreview {
  path: string;
  body: string;
  size: number;
  truncated: boolean;
  binary: boolean;
  err?: string;
}

// 文件预览负载：kind 决定渲染方式（image/docx/xlsx/markdown/text/unsupported/error）。
// docx 时 dataUrl 为原始文件（前端 docx-preview 保真渲染）。
// xlsx 时 body 为结构化单元格 JSON（值/公式/样式，前端表格渲染）。
export interface PreviewResult {
  path: string;
  name: string;
  ext: string;
  size: number;
  kind: "image" | "docx" | "xlsx" | "markdown" | "text" | "unsupported" | "error";
  body: string;
  dataUrl: string;
  error: string;
  truncated?: boolean;
  totalPages?: number;
}

// OfficeEditResult 是框选即改的 AI 编辑结果（替换文本）。
export interface OfficeEditResult {
  edited: string;
}

// ── xlsx 单元格级预览 ──────────────────────────────────────
export interface XlsxCellStyle {
  bold?: boolean;
  italic?: boolean;
  underline?: boolean;
  strike?: boolean;
  fontColor?: string;
  fill?: string;
  align?: "left" | "center" | "right";
  wrap?: boolean;
  numFmt?: string;
  border?: boolean;
}

export interface XlsxCell {
  ref: string; // "A1"
  value: string;
  formula?: string;
  type?: string; // number|string|bool|date|formula|error
  style?: XlsxCellStyle;
}

export interface XlsxSheet {
  name: string;
  rows: XlsxCell[][];
  merged?: string[]; // "A1:B2"
  colWidths?: Record<string, number>;
  freeze?: { row?: number; col?: number }; // 冻结窗格（表头）
  truncated?: boolean;
}

export interface XlsxPreview {
  sheets: XlsxSheet[];
}

// XlsxEditResult 是单元格编辑结果：更新后的预览 + 摘要。
export interface XlsxEditResult {
  preview: string;
  summary: string;
  applied: number;
}

// ── 统一交付出口（事实底座 → 多形态交付） ──────────────────
export interface ExportDeliverableInput {
  markdown: string;
  format: "docx" | "pptx" | "xlsx" | "md";
  title?: string;
  template?: "通用" | "公文" | "报告" | "合同";
  cover?: boolean;
  toc?: boolean;
  header?: string;
  footer?: string;
}

export interface ExportDeliverableResult {
  path: string;
  name: string;
  format: string;
  size: number;
}

// ── 跨应用联动（xlsx 数据 → 图表 → 嵌入 docx/pptx） ────────
export interface CrossEmbedInput {
  xlsxRel: string;
  sheet?: string;
  range?: string; // "A1:B6"，空 = 自动
  chartType?: "bar" | "line" | "pie" | "scatter";
  title?: string;
  into: "docx" | "pptx";
  output?: string;
}

export interface CrossEmbedResult {
  path: string;
  name: string;
  size: number;
  chartPath: string;
}

// MCP & Skills drawer (desktop/app.go Capabilities) — the GUI counterpart to
// /mcp + /skill: connected/failed servers and discoverable skills.
export interface ServerView {
  name: string;
  transport: string;
  status: "connected" | "failed" | "disabled";
  tools: number;
  prompts: number;
  resources: number;
  error?: string;
  toolList?: MCPToolView[];
}
export interface MCPToolView {
  name: string;
  description: string;
}
export interface SkillView {
  name: string;
  description: string;
  scope: string;
  runAs: string;
}
export interface CapabilitiesView {
  servers: ServerView[];
  skills: SkillView[];
}
export interface MCPServerInput {
  name: string;
  transport: string; // stdio | http | sse
  command: string;
  args: string[];
  url: string;
  env: Record<string, string>;
}

export interface ModelInfo {
  ref: string; // "provider/model" — pass to SetModel
  provider: string;
  model: string;
  current: boolean;
}

// Slash sub-command / argument completion (desktop/app.go SlashArgs). Mirrors the
// CLI's arg hints so the composer can suggest e.g. /skill → list/show/new/paths.
export interface SlashArgItem {
  label: string;
  insert: string; // token to place at the current position
  hint: string;
  descend: boolean; // re-open the menu one level deeper after accepting
}
export interface SlashArgsResult {
  items: SlashArgItem[];
  from: number; // byte offset where the current token begins
}

// Memory panel payloads (desktop/app.go MemoryView).
export interface MemoryDoc {
  path: string;
  scope: string; // "user" | "ancestor" | "project" | "local"
  body: string;
}

export interface MemoryFact {
  name: string;
  title?: string;
  description: string;
  type: string; // "user" | "feedback" | "project" | "reference"
  body: string;
  lastUsedAt?: string; // RFC3339 最近使用（生命周期/高频展示）
  sourceSession?: string; // 沉淀来源会话
  sourceMessage?: string; // 沉淀来源消息/轮次
}

export interface MemoryScope {
  scope: string; // "user" | "project" | "local"
  path: string;
}

export interface MemoryView {
  docs: MemoryDoc[];
  facts: MemoryFact[];
  scopes: MemoryScope[];
  storeDir: string;
  available: boolean;
  enabled?: boolean; // 记忆开关（当前生效值）
  archives?: MemoryArchive[];
}

// KnowledgeSummary is the lightweight view of a knowledge entry (without body).
export interface KnowledgeSummary {
  name: string;
  title: string;
  category: string;
  tags: string[];
  status: string;
  // 新建/导入时前端不发送时间戳（Go 端 time.Time 不接受空串），留空由后端置零。
  updatedAt?: string;
}

// KnowledgeEntry is the full knowledge entry including body.
export interface KnowledgeEntry extends KnowledgeSummary {
  body: string;
  phase: string;
  discipline: string;
  source: string;
  version: number;
  author: string;
  reviewer: string;
  // 新建/导入时前端不发送时间戳（Go 端 time.Time 不接受空串），留空由后端置零。
  createdAt?: string;
}

/** Alias for clarity when saving. */
export type KnowledgeSaveRequest = KnowledgeEntry;

// Settings panel payloads (desktop/settings_app.go).
export interface ProviderView {
  name: string;
  kind: string;
  baseUrl: string;
  models: string[];
  default: string;
  apiKeyEnv: string;
  keySet: boolean; // the env var currently resolves to a value
  balanceUrl: string; // optional wallet-balance endpoint; "" disables the readout
  contextWindow: number;
  oauthKind: string; // non-empty when OAuth login is available (e.g. "xai")
  oauthReady: boolean; // whether OAuth token is currently valid
}

// BalanceInfo is the wallet-balance readout (desktop/app.go Balance). available
// is false when the provider declares no balanceUrl or a fetch failed; display is
// the formatted amount (e.g. "¥110.00").
export interface BalanceInfo {
  available: boolean;
  display: string;
  err?: string;
}

// JobView is one running background job (desktop/app.go Jobs) for the status bar.
export interface JobView {
  id: string;
  kind: string; // "bash" | "task"
  label: string;
  status: string; // "running"
  startedAt: number; // unix milliseconds
}

// FactView: one settled fact in the conversation fact base (sidebar panel).
export interface FactView {
  key: string;
  value: string;
  source?: string;
  category?: string;
  updatedAt: number;
}

// FactBaseView: the fact-base panel view: facts + copy-ready Markdown.
export interface FactBaseView {
  facts: FactView[];
  markdown: string;
  count: number;
  path: string;
}

// SkillCaptureInput 是一次成功对话沉淀为技能的输入（桌面端 GaeaCaptureSkill）。
export interface SkillCaptureInput {
  name: string;
  description: string;
  task: string;
  solution: string;
}

// SkillCaptureResult 是沉淀结果；reloaded=true 表示技能已热加载进引擎。
export interface SkillCaptureResult {
  name: string;
  description: string;
  path: string;
  reloaded: boolean;
  tools: number;
  skills: number;
}

export interface PermissionsView {
  mode: string; // "ask" | "allow" | "deny"
  allow: string[];
  ask: string[];
  deny: string[];
}

export interface SandboxView {
  bash: string; // "enforce" | "off"
  network: boolean;
  workspaceRoot: string;
  allowWrite: string[];
}

export interface AgentView {
  temperature: number;
  maxSteps: number;
  systemPrompt: string;
  subagentTemperature: number;
  effort: string;
  subagentEffort: string;
}

export interface SettingsView {
  defaultModel: string;
  subagentModel: string;
  subagentModels: Record<string, string>; // per-skill overrides
  subagentSkills: string[]; // builtin subagent skill names
  providers: ProviderView[];
  permissions: PermissionsView;
  sandbox: SandboxView;
  agent: AgentView;
  configPath: string;
  providerKinds: string[]; // provider implementations the kernel registered (for the kind picker)
  bypass: boolean; // DEPRECATED — use permLevel instead
  permLevel?: string; // live permission level this session (\"ask\"|\"auto\"|\"yolo\")
}

// Auto-updater payloads (desktop/updater.go). UpdateInfo drives the update banner;
// UpdateProgress streams on the "updater:progress" event during ApplyUpdate.
export interface UpdateInfo {
  available: boolean;
  current: string;
  latest: string;
  notes: string;
  canSelfUpdate: boolean; // win/linux true; macOS false (no cert → manual download)
  downloadUrl: string; // human-facing releases page (macOS path / fallback link)
  assetSize: number; // running platform's artifact size, for the progress bar
  err?: string; // set when the check itself failed (both endpoints down)
}

export interface UpdateProgress {
  phase: "downloading" | "verifying" | "applying" | "done" | "error";
  received: number;
  total: number;
  err?: string;
}

// TCCA 缓存报告（V3.0 — 匹配 internal/context/metrics.go CacheReport）
export interface TCCAReport {
  l1Size: number;
  l2Size: number;
  l3Version: number;
  l4Messages: number;
  savedByCompact: number;
  savedByFork: number;
  forkCount: number;
  savedUsd: number;
  savedLatencyMs: number;
  compactionCount: number;
  // V5.30: 全会话缓存命中统计 (来自 agent + context metrics)
  cacheHitTokens: number;
  cacheMissTokens: number;
  breakCount: number;
}

export interface MemoryArchive {
  name: string;
  title?: string;
  description: string;
  type: string;
  body: string;
  path?: string;
  archivedAt?: string;
}

// MemoryArchivedView 是办公记忆归档列表中的一条（GaeaMemoryArchivedList）：
// 归档超过 90 天的事实为硬删除候选（GaeaMemoryCleanupArchived 清理）。
export interface MemoryArchivedView {
  name: string;
  title?: string;
  description: string;
  type: string;
  kind: string;
  // RFC3339 字符串，可为空（后端 time.Time 序列化；空表示时间缺失）。
  archivedAt: string;
}

// MemoryArchivedPage 是归档列表分页结果（GaeaMemoryArchivedList）。
export interface MemoryArchivedPage {
  items: MemoryArchivedView[];
  total: number;
  limit: number;
  offset: number;
}

export interface MemorySuggestion {
  id: string;
  name: string;
  title?: string;
  description: string;
  type: string;
  body: string;
  reason: string;
  evidence: string[];
}

export interface SkillSuggestion {
  id: string;
  name: string;
  description: string;
  scope: string;
  body: string;
  reason: string;
  evidence: string[];
}

export interface MemorySuggestionsView {
  memories: MemorySuggestion[];
  skills: SkillSuggestion[];
  generatedAt: string;
  available: boolean;
  source: string;
}

export interface TabMeta {
  id: string;
  scope: string;
  workspaceRoot: string;
  title: string;
  ready: boolean;
  label?: string;
  activityStatus?: string;
}

/** TraceStep 记录一次推理过程中的关键步骤，用于实时推理可视化面板 */
export interface TraceStep {
  id: string;
  turnTimestamp: number; // 所属回合开始时间戳
  seq: number;           // 步骤序号（递增）
  type: "phase" | "tool" | "thinking" | "decision" | "compaction";
  timestamp: number;     // 事件发生时间戳（ms）
  label: string;         // 简短描述（如 "规划阶段"、"read_file"、"压缩上下文"）
  detail?: string;       // 详细内容（展开时显示）
  status?: "pending" | "running" | "done" | "error";  // 当前状态
}

// ── 成本库类型 ───────────────────────────────────────────────────────────


/** FilePickResult 描述从原生对话框选取的一个文件 */
export interface FilePickResult {
  path: string;
  /** 仅图片文件有 previewUrl（data: URL） */
  previewUrl?: string;
  type: "image" | "file";
  name: string;
}

// ── 记忆中枢（Memory Hub）类型 ──────────────────────────────────────

// ProfileFactView 主脑全局画像事实（跨板块共享的用户画像）。
export interface ProfileFactView {
  name: string;
  title: string;
  description: string;
  type: string;
  kind: string;
  tags: string[];
  body: string;
}

// WhisperMemoryView 聊天（hermes.db）记忆事实只读视图。
export interface WhisperMemoryView {
  id: string;
  domain: string;
  subcategory: string;
  subject: string;
  summary: string;
  weight: number;
  confidence: number;
  tier: string;
  status: string;
  updatedAt: string;
}

// WhisperEpisodeView 聊天（hermes.db）情节记忆只读视图（时间倒序）。
export interface WhisperEpisodeView {
  id: string;
  summary: string;
  dominantEmotion: string;
  emotionalIntensity: number;
  keywords: string[];
  startTurn: number;
  endTurn: number;
  createdAt: string;
  sourceSessionId: string;
}

// MemoryHubOverview 记忆中枢聚合总览。
export interface MemoryHubOverview {
  knowledgeCount: number;
  profileCount: number;
  officeCount: number;
  costCount: number;
  whisperCount: number;
  pinnedCount: number; // 项目资料：工作区固定常用文件数
  latestUpdated: string;
}

// ── 记忆图谱 ────────────────────────────────────────────────────────
export interface GraphNode {
  id: string;
  name: string;
  type: string; // knowledge / profile / office / whisper
  desc: string;
  val: number;
}
export interface GraphLink {
  source: string;
  target: string;
  type: string; // same-tag / same-category / reference
}
export interface MemoryGraphView {
  nodes: GraphNode[];
  links: GraphLink[];
}

// ── 成本库 ──────────────────────────────────────────────────────────
export interface CostSummary {
  name: string;
  title: string;
  category: string;
  // 完整分类路径：一级/二级/…/叶子（多级分类保存与树形过滤依据）。
  categoryPath: string;
  unit: string;
  price: number;
  spec: string;
  source: string;
  tags: string[];
  status: string;
  // 新建/导入时前端不发送时间戳（Go 端 time.Time 不接受空串），留空由后端置零。
  updatedAt?: string;
}
export interface CostEntry extends CostSummary {
  body: string;
  createdAt?: string;
}

// 成本分类树节点（多级：parentId 自引用，children 为子树）。
export interface CostCategory {
  id: number;
  parentId: number;
  name: string;
  sort: number;
  count: number;
  children?: CostCategory[];
}

// 导入预览中的一条候选成本条目（前端可编辑后确认导入）。
export interface CostImportRow {
  name: string;
  title: string;
  category: string;
  unit: string;
  price: number;
  spec: string;
  source: string;
  status: string;
  existingName: string;
  existingPrice: number;
  matchNote: string; // 新增 / 将覆盖更新（现价 ¥xxx）
  raw: string;
  skip: boolean;
  skipReason: string;
}

// 文件导入解析结果（无确认不落库，确认走 CostImportApply）。
export interface CostImportPreview {
  path: string;
  fileName: string;
  columns: string[];
  unmapped: string[];
  rows: CostImportRow[];
  message: string;
  aiUsed: boolean;
  // 导入文件来源类型：xlsx/csv 表格解析；pdf_text 文字型 PDF；
  // pdf_scan 扫描件（OCR）；image 图片报价单（OCR）。
  source?: "xlsx" | "csv" | "pdf_text" | "pdf_scan" | "image";
}

// ── 价格源（定时抓取价格更新）──────────────────────────────────
export interface PriceSource {
  id: string;
  name: string;
  url: string;
  parser: string; // sc_table：造价信息网价格表
  frequencyHours: number; // 0=仅手动
  area: string;
  headers?: Record<string, string>;
  enabled: boolean;
  lastFetchAt: string;
  createdAt: string;
}

export interface PriceCandidate {
  title: string;
  spec: string;
  unit: string;
  price: number;
  tax: string;
  existingName: string;
  existingPrice: number;
  status: "更新" | "无变化" | "新增";
  diff: number;
  diffPct: number;
  anomaly: boolean; // 偏离历史价格区间（价格异常）
  anomalyReason: string;
}

export interface PriceFetchRecord {
  id: string;
  sourceId: string;
  sourceName: string;
  url: string;
  period: string;
  fetchedAt: string;
  status: "pending" | "applied" | "ignored";
  candidates: PriceCandidate[];
}

export interface PriceHistory {
  name: string;
  title: string;
  unit: string;
  price: number;
  source: string;
  period: string;
  fetchedAt: string;
  note: string;
}

// 跨库统一语义检索命中（cost / knowledge / office，本地 bge-m3）。
export interface SemanticHitView {
  kind: "cost" | "knowledge" | "office";
  name: string;
  score: number;
  text: string;
}

// CostCompareRow 是一条比价明细：同一成本条目在多个来源/时段的报价对比。
// kind: current=成本库现价 / history=历史快照 / fetch=价格源抓取候选。
export interface CostCompareRow {
  source: string; // 来源（如「成本库」「四川造价信息网」「XX租赁」）
  period: string; // 期号/时段（如 "758"；成本库现价可为空）
  price: number; // 该来源报价
  diffPct: number; // 与基准价（通常为成本库现价）的偏差百分比，基准行为 0
  fetchedAt: string; // 获取时间（ISO 字符串，可为空）
  kind: "current" | "history" | "fetch";
}

// UnifiedSearchView 是一次「跨库统一检索」调用的完整结果：
// keyword = 工作区全文关键词命中（轻量 RAG），semantic = 跨库语义命中。
export interface UnifiedSearchView {
  keyword: WorkspaceSearchHit[];
  semantic: SemanticHitView[];
}

// RetrievalEvalQuery 是检索质量测评中单条查询的明细：期望命中 vs 实际前 10 命中。
// expected/topHits 均为 "kind:name" 形式（如 "cost:hp300"），便于前端直接对比。
export interface RetrievalEvalQuery {
  query: string; // 测评查询文本
  expected: string[]; // 期望命中（kind:name）
  topHits: string[]; // 实际返回的前 10 条命中（kind:name）
  recall: number; // 该查询的 recall@10（0-1）
}

// RetrievalEvalReport 是检索质量测评结果：内置查询集跑一遍统一检索，
// 统计平均 recall@10，并与达标门槛比较给出通过状态。
export interface RetrievalEvalReport {
  total: number; // 测评查询总数
  threshold: number; // 达标门槛（recall@10 需 ≥ 该值，后端固定 0.8）
  recallAt10: number; // 平均 recall@10（0-1）
  passed: boolean; // recallAt10 ≥ threshold
  perQuery: RetrievalEvalQuery[];
  note?: string; // 命中判定规则等说明（后端 omitempty）
}

// ── 知识库导入（无确认不落库）────────────────────────────────
export interface KnowledgeImportRow {
  name: string;
  title: string;
  category: string;
  phase: string;
  discipline: string;
  tags: string[];
  status: string;
  source: string;
  body: string;
  existingName: string;
  matchNote: string; // 新增 / 将覆盖更新
  similarName: string;
  similarNote: string; // 与「xxx」相似 87%，建议合并
  raw: string;
  skip: boolean;
  skipReason: string;
}

export interface KnowledgeImportPreview {
  path: string;
  fileName: string;
  columns: string[];
  unmapped: string[];
  rows: KnowledgeImportRow[];
  message: string;
  aiUsed: boolean;
}

// 知识条目版本历史快照。
export interface KnowledgeHistoryView {
  name: string;
  title: string;
  version: number;
  category: string;
  phase: string;
  discipline: string;
  tags: string[];
  status: string;
  author: string;
  reviewer: string;
  source: string;
  body: string;
  changedAt: string;
  note: string;
}

// 查重命中的相似条目。
export interface SimilarView {
  name: string;
  title: string;
  score: number;
}

// 办公记忆疑似重复对（keep 为建议保留项）。
export interface MemoryDuplicateView {
  keep: string;
  keepTitle: string;
  dup: string;
  dupTitle: string;
  score: number;
}

// 工作区文件语义索引状态 / 命中。
export interface FileIndexStatus {
  total: number;
  skipped: number;
  error: string;
}
export interface FileSemanticHit {
  path: string;
  score: number;
  snippet: string;
}

// ── 阶段 5 T5-1：通用任务调度器视图 ──
// 长任务（价格抓取/文件索引重建等）统一走持久化任务队列；gaea-task 事件
// 实时推送任务视图（状态/进度/消息），任务中心据此渲染并支持取消/重试。
export type TaskStatus = "queued" | "running" | "succeeded" | "failed" | "cancelled";

export interface TaskView {
  id: string;
  kind: string; // price_fetch / price_fetch_all / file_index / …
  label: string;
  status: TaskStatus;
  progress: number; // 0-100
  message: string;
  error: string;
  retryCount: number;
  maxRetries: number;
  payload: string; // 不透明 JSON
  result: string; // 不透明 JSON（完成时按 kind 解析）
  createdAt: number; // unix 毫秒
  startedAt: number;
  finishedAt: number;
}

// ── 阶段 5 T5-3：本地模型调度纵深 ────────────────────────────
// ModelSwitchEstimate 是换模预估结果（GaeaModelSwitchEstimate）：切换本地模型前
// 提示等待时长，让用户决定是否继续（hot=已运行可直接切换；cold=已安装需冷启动；
// download=未安装需先下载；unknown=无法评估）。
export interface ModelSwitchEstimate {
  engine: string; // 引擎 ID（如 herdsman）
  model: string; // 目标模型名（预估基于引擎聚合时可为空）
  status: "hot" | "cold" | "download" | "unknown";
  waitSeconds: number; // 预计等待秒数（hot 为 0）
  note: string; // 人类可读说明（如「已运行，约 0 秒」「需冷启动约 12 秒」）
}
