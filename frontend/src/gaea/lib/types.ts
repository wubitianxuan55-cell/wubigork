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
  | "compaction_done";

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
  err?: string;
}

// Bound-method payloads (desktop/app.go).
export interface HistoryMessage {
  role: string;
  content: string;
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
  archives?: MemoryArchive[];
}

// KnowledgeSummary is the lightweight view of a knowledge entry (without body).
export interface KnowledgeSummary {
  name: string;
  title: string;
  category: string;
  tags: string[];
  status: string;
  updatedAt: string;
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
  createdAt: string;
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
  whisperCount: number;
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
  unit: string;
  price: number;
  spec: string;
  source: string;
  tags: string[];
  status: string;
  updatedAt: string;
}
export interface CostEntry extends CostSummary {
  body: string;
  createdAt: string;
}
