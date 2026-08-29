
// Wire contract — mirrors desktop/wire.go (itself mirroring internal/serve/wire.go).
// One event channel carries every kind; `kind` discriminates the payload.

// 精确匹配生成模型 → 别名（单一事实源 wailsjs/go/models.ts，scripts/check-types-drift.mjs --alias-exact 生成）
import type { app as AppModels } from "../../../wailsjs/go/models";

// WireShape 把 wails 生成类的实例形状剥成纯线格式数据：生成类含实例方法
// convertValues（构造时递归实例化嵌套对象），不属于 JSON 线协议字段；递归映射
// 同时处理嵌套类属性。别名统一用 WireShape<AppModels.X>，消费方拿到的仍是
// 与旧手写 interface 一致的结构类型。
export type WireShape<T> = T extends (...args: never[]) => unknown
  ? never
  : T extends (infer U)[]
    ? WireShape<U>[]
    : T extends object
      ? { [K in keyof T as K extends "convertValues" ? never : K]: WireShape<T[K]> }
      : T;

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
export type GaeaReloadResult = WireShape<AppModels.GaeaReloadResult>;

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
  progress?: WirePreviewProgress;
  path?: string; // preview_progress: 当前预览文件（冗余，便于前端过滤）
  err?: string;
}

// Bound-method payloads (desktop/app.go).
export type HistoryMessage = WireShape<AppModels.HistoryMessage>;

// CheckpointMeta is one rewind point (a user turn) for the rewind UI.
export type CheckpointMeta = WireShape<AppModels.CheckpointMeta>;

// SessionStatsView 是会话级 token/成本派生统计（后端从事件日志重放 usage 事件）。
// available=false 表示该会话无事件日志（legacy 会话或路径非法），前端不展示
// 历史统计块。
export type SessionStatsView = WireShape<AppModels.SessionStatsView>;

// SessionMeta is one saved session for the history panel.
export interface SessionMeta {
  path: string;
  preview: string;
  title?: string; // user-chosen name; falls back to preview when empty
  turns: number;
  modTime: number; // unix milliseconds
  current: boolean;
  pinned?: boolean; // 置顶会话排在同项目会话最前
  archived?: boolean; // 已归档（在 <sessions>/archive/ 下，可恢复）
  // interrupted=true 表示上次运行中断未完成；恢复会话时后端注入「上次会话中断」
  // 摘要提示并清除该标记（可选字段：后端始终返回，前端对缺失按 false 处理）。
  interrupted?: boolean;
  // S1 空间归属（后端 SessionMeta.spaceId，typesGenerationCheck 漂移校验钉死）。
  spaceId?: string;
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

export type WorkspaceView = WireShape<AppModels.WorkspaceView>;

// SpaceOption 是双空间静态枚举项（GaeaSpaceList，work/play 固定两值）。
export type SpaceOption = WireShape<AppModels.SpaceOption>;

// SpaceActiveView 是当前生效空间视图（GaeaSpaceActive / GaeaSpaceActivate）。
// space.mode=off 时分区整体关闭，space 恒报 work（modeOn=false 标记关闭态）。
export type SpaceActiveView = WireShape<AppModels.SpaceActiveView>;

export type ContextInfo = WireShape<AppModels.ContextInfo>;

// ─── 上下文视图（dsh-context Go 移植 Phase A）────────────────────

export interface ContextCategory {
  system: number;
  tools: number;
  user: number;
  inject: number;
  assistant: number;
  tool: number;
}

export interface ContextStats {
  turns: number;
  steps: number;
  injects: number;
  compacts: number;
  prunes: number;
  toolCalls: number;
  images: number;
  cacheHitPercent?: number;
  costEstimate?: number;
}

export interface ContextRequestRecord {
  seq: number;
  ts: number;
  turn: number;
  step: number;
  category: ContextCategory;
  briefUser?: string;
  briefIn?: string[];
  briefResp?: string;
  promptTokens?: number;
  outputTokens?: number;
  cacheHitTokens?: number;
  cacheMissTokens?: number;
}

export interface ContextEvent {
  kind: "inject" | "compact" | "prune" | "switch" | "mode";
  seq: number;
  delta?: number;
  source?: string;
  turn: number;
  step: number;
  ts: number;
}

export interface ContextSurfaceNode {
  seq: number;
  cat: "system" | "tools" | "user" | "inject" | "assistant" | "tool";
  tokens: number;
  text?: string;
  gone?: number;
}

export interface ContextTimeline {
  ok: boolean;
  window: number;
  current: ContextCategory;
  stats: ContextStats;
  requests: ContextRequestRecord[];
  events: ContextEvent[];
  nodes: ContextSurfaceNode[];
  archive: ContextSurfaceNode[];
}

// ─── 轨迹视图（对齐 DSH ui-trajectory 事件账本）────────────────

export interface Trajectory {
  ok: boolean;
  turns: TrajectoryTurn[];
  betweenTurns?: TrajectoryRecord[];
}

export interface TrajectoryTurn {
  turn: number;
  startedAt?: number;
  end?: TrajectoryTurnEnd;
  records: TrajectoryRecord[];
}

export interface TrajectoryTurnEnd {
  seq: number;
  ts: number;
  err?: string;
}

export type TrajectoryRecordKind =
  | "user" | "header" | "assistant" | "tool" | "compact" | "ask" | "approval";

export interface TrajectoryRecord {
  seq: number;
  kind: TrajectoryRecordKind;
  ts: number;
  durationMs?: number;
  step?: number;
  user?: TrajectoryUserRec;
  header?: TrajectoryHeaderRec;
  assistant?: TrajectoryAssistantRec;
  tool?: TrajectoryToolRec;
  compact?: TrajectoryCompactRec;
  ask?: TrajectoryAskRec;
  approval?: TrajectoryApprovalRec;
}

export interface TrajectoryUserRec {
  text: string;
}

export interface TrajectoryHeaderRec {
  system?: string;
  toolCount: number;
  tokens: number;
  change?: "initial" | "system" | "tools" | "system-and-tools";
}

export interface TrajectoryAssistantRec {
  text?: string;
  reasoning?: string;
  usage?: TrajectoryUsage;
}

export interface TrajectoryToolRec {
  id: string;
  name: string;
  args?: string;
  output?: string;
  err?: string;
  truncated?: boolean;
  readOnly?: boolean;
  status: "ok" | "error" | "running";
  parentId?: string;
}

export interface TrajectoryCompactRec {
  trigger?: string;
  summary?: string;
}

export interface TrajectoryAskRec {
  question?: string;
}

export interface TrajectoryApprovalRec {
  tool?: string;
  subject?: string;
}

export interface TrajectoryUsage {
  promptTokens?: number;
  completionTokens?: number;
  cacheHitTokens?: number;
  cacheMissTokens?: number;
  reasoningTokens?: number;
}

// ─── Agent 网络（dsh-context Agent network 卡）────────────────

export interface AgentNetwork {
  ok: boolean;
  window: number;
  root: AgentNode;
}

export interface AgentNode {
  id: string;
  name: string;
  kind: "root" | "subagent";
  status: "running" | "completed" | "error";
  model?: string;
  task?: string;
  toolCalls: number;
  errors: number;
  tokens: number;
  firstTs?: number;
  lastTs?: number;
  children?: AgentNode[];
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

export type DirEntry = WireShape<AppModels.DirEntry>;

export interface FileSearchHit {
  path: string; // 工作区相对路径（/ 分隔）
  name: string;
  isDir: boolean;
  size?: number;
  modTime: number; // S2.3 types 漂移修复：后端返回修改时间（unix ms）
}

/** AtEntry 是 @ 菜单的统一条目（目录浏览 / 工作区搜索 / 最近使用文件）。 */
export interface AtEntry {
  path: string; // 工作区相对路径；目录以 / 结尾
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
export type GaeaSummaryResult = WireShape<AppModels.GaeaSummaryResult>;

// TaskTemplate 是预置办公任务模板（欢迎页「任务模板」区 + slash 命令）。
export type TaskTemplate = WireShape<AppModels.TaskTemplate>;

// 后端 GaeaReadFile 真实契约（gaea_ui_extra.go FilePreview = path/markdown/size）；
// 旧手写体含 body/truncated/binary 为历史残留，已收敛为生成模型别名。
export type FilePreview = WireShape<AppModels.FilePreview>;

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
export type XlsxEditResult = WireShape<AppModels.XlsxEditResult>;

// XlsxCellChange 是规划 diff 中的一处单元格变更（值或公式与原文件不同）。
export interface XlsxCellChange {
  sheet: string;
  cell: string;
  before: string;
  after: string;
  formula?: string; // 变更后为公式时给出（显示 fx）
}

// XlsxPlanResult 是「先规划后应用」的规划结果：ops 原样带回（应用时透传），
// 附变更清单供用户审阅批准（对标 Copilot Plan/Show Changes 范式）。
export type XlsxPlanResult = WireShape<AppModels.XlsxPlanResult>;

// ── 表格「选中区域 → 一键图表」（原生图表嵌入工作簿） ──
export interface XlsxChartInput {
  rel: string; // xlsx 工作区相对路径
  sheet?: string; // 工作表名；空 = 第一个工作表
  refs?: string; // 区域 "A1:B6" 或单单元格 "B2"；空 = 自动取前两列数据行
  chartType?: "bar" | "line" | "pie" | "scatter";
  title?: string;
}

export type XlsxChartResult = WireShape<AppModels.XlsxChartResult>;

// ── 会话产物一键打包（P0-1，对标 Kimi 工作空间 / WorkBuddy） ──
export type ZipDeliverableResult = WireShape<AppModels.ZipDeliverableResult>;

// ConvertPdfResult 是文档转 PDF 的结果（PDF 落 .gaea/exports/）。
export type ConvertPdfResult = WireShape<AppModels.ConvertPdfResult>;

// ── 多智能体分工可见（P2，对标 WorkSwarm 蜂群 / QClaw V2） ──
export interface SubagentRunView {
  ref: string; // sa_YYYYMMDD_HHMMSS_... 稳定引用
  status: "running" | "completed" | "failed";
  model?: string;
  toolScope?: string[];
  task: string; // transcript 首条 user 消息（任务摘要）
  answer?: string; // 最后一条 assistant 回答摘要
  toolCalls: number;
  lastText?: string; // C2 活动行：最后一段 assistant 文本（运行中实时更新）
  lastTool?: string; // C2 活动行：最后一次工具调用摘要（name + 结果头）
  createdAt: string;
  updatedAt: string;
}

export interface SubagentRunsView {
  available: boolean; // false = 会话无 subagents 目录（未派发子代理）
  runs: SubagentRunView[];
  total: number;
  running: number;
}

// ── 统一交付出口（事实底座 → 多形态交付） ──────────────────
export interface ExportDeliverableInput {
  markdown: string;
  format: "docx" | "pptx" | "xlsx" | "md" | "pdf";
  title?: string;
  template?: "通用" | "公文" | "报告" | "合同";
  cover?: boolean;
  toc?: boolean;
  header?: string;
  footer?: string;
}

export type ExportDeliverableResult = WireShape<AppModels.ExportDeliverableResult>;

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

export type CrossEmbedResult = WireShape<AppModels.CrossEmbedResult>;

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
export type SkillView = WireShape<AppModels.SkillView>;
export interface CapabilitiesView {
  servers: ServerView[];
  skills: SkillView[];
}
export type MCPServerInput = WireShape<AppModels.MCPServerInput>;

export type ModelInfo = WireShape<AppModels.ModelInfo>;

// Slash sub-command / argument completion (desktop/app.go SlashArgs). Mirrors the
// CLI's arg hints so the composer can suggest e.g. /skill → list/show/new/paths.
export type SlashArgItem = WireShape<AppModels.SlashArgItem>;
export type SlashArgsResult = WireShape<AppModels.SlashArgsResult>;

// Memory panel payloads (desktop/app.go MemoryView).
export type MemoryDoc = WireShape<AppModels.MemoryDoc>;

export type MemoryFact = WireShape<AppModels.MemoryFact>;

export type MemoryScope = WireShape<AppModels.MemoryScope>;

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
export type ProviderView = WireShape<AppModels.ProviderView>;

// BalanceInfo is the wallet-balance readout (desktop/app.go Balance). available
// is false when the provider declares no balanceUrl or a fetch failed; display is
// the formatted amount (e.g. "¥110.00").
export type BalanceInfo = WireShape<AppModels.BalanceInfo>;

// JobView is one running background job (desktop/app.go Jobs) for the status bar.
export type JobView = WireShape<AppModels.JobView>;

// FactView: one settled fact in the conversation fact base (sidebar panel).
export type FactView = WireShape<AppModels.FactView>;

// FactBaseView: the fact-base panel view: facts + copy-ready Markdown.
export type FactBaseView = WireShape<AppModels.FactBaseView>;

// SkillCaptureInput 是一次成功对话沉淀为技能的输入（桌面端 GaeaCaptureSkill）。
export type SkillCaptureInput = WireShape<AppModels.SkillCaptureInput>;

// SkillCaptureResult 是沉淀结果；reloaded=true 表示技能已热加载进引擎。
export type SkillCaptureResult = WireShape<AppModels.SkillCaptureResult>;

export type PermissionsView = WireShape<AppModels.PermissionsView>;

export type SandboxView = WireShape<AppModels.SandboxView>;

export type AgentView = WireShape<AppModels.AgentView>;

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
  version?: string; // S2.3 types 兼容：后端 stub GaeaCheckUpdate 的 {version, notes} 契约
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
export type MemoryArchivedView = WireShape<AppModels.MemoryArchivedView>;

// MemoryArchivedPage 是归档列表分页结果（GaeaMemoryArchivedList）。
export interface MemoryArchivedPage {
  items: MemoryArchivedView[];
  total: number;
  limit: number;
  offset: number;
  /** 归档保留期（天），后端 gaea_memory_lifecycle.go RetentionDays 下发；mock 未带时 UI 回退 90 */
  retentionDays?: number;
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

export type MemorySuggestionsView = WireShape<AppModels.MemorySuggestionsView>;

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
  /** 文件字节数（P2-4 附件上下文占用展示用；后端 GaeaPickFiles 已返回）。 */
  size?: number;
}

// ── 记忆中枢（Memory Hub）类型 ──────────────────────────────────────

// ProfileFactView 主脑全局画像事实（跨板块共享的用户画像）。
export type ProfileFactView = WireShape<AppModels.ProfileFactView>;

// WhisperMemoryView 聊天（hermes.db）记忆事实只读视图。
export type WhisperMemoryView = WireShape<AppModels.WhisperMemoryView>;

// WhisperEpisodeView 聊天（hermes.db）情节记忆只读视图（时间倒序）。
export type WhisperEpisodeView = WireShape<AppModels.WhisperEpisodeView>;

// MemoryHubOverview 记忆中枢聚合总览。
export type MemoryHubOverview = WireShape<AppModels.MemoryHubOverview>;

// ── 记忆图谱 ────────────────────────────────────────────────────────
export type GraphNode = WireShape<AppModels.GraphNode>;
export type GraphLink = WireShape<AppModels.GraphLink>;
export type MemoryGraphView = WireShape<AppModels.MemoryGraphView>;

// ── 成本库 ──────────────────────────────────────────────────────────
export interface CostSummary {
  name: string;
  title: string;
  category: string;
  // 完整分类路径：一级/二级/…/叶子（多级分类保存与树形过滤依据）。
  categoryPath: string;
    unit: string;
    price: number;
    // 人材机二级汇总（综合单价子目，元）。
    laborFee?: number;
    materialFee?: number;
    machineFee?: number;
    // 人材机组成行数（综合单价子目的二级明细规模）。
    componentCount?: number;
    spec: string;
  source: string;
  // 价格三要素蒸馏（zaojia-database）：地区 + 价格时间/期数；口径与有效期辅助判断可信度。
  region?: string;
  priceDate?: string;
  priceType?: string;
  validUntil?: string;
  sourceRow?: number;
  tags: string[];
  status: string;
  // 新建/导入时前端不发送时间戳（Go 端 time.Time 不接受空串），留空由后端置零。
  updatedAt?: string;
}
  // 综合单价子目的人材机组成明细行（二级）。
  export interface CostComponent {
    kind: string; // 人工/材料/机械（可组合标签，如 人工+机械）
    title: string;
    unit?: string;
    quantity?: number; // 含量/数量（0=未解析出）
    price?: number; // 资源单价（0=未解析出）
    amount?: number; // 金额（含量×单价，含损耗）
    note?: string; // 原始行表达式（含损耗系数，追溯用）
    sort?: number;
  }

  export interface CostEntry extends CostSummary {
    // 费率（仅展示追溯，不参与计算）：管理费/利润/垫资 为金额（元），税率为百分比。
    managementFee?: number;
    profitFee?: number;
    advanceFee?: number;
    taxRate?: number;
    components?: CostComponent[];
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

// ── 测算项目与沉淀闭环（zaojia-database 蒸馏：我的项目/工程量清单/版本留痕）──

// CostProject 测算项目容器（一次报价/测算工作）。
export interface CostProject {
  id: string;
  name: string;
  projectType: string;
  scale: string;
  craft: string;
  status: string; // 编制中 / 已保存版本 / 已沉淀
  note: string;
  createdAt?: string;
  updatedAt?: string;
}

// CostProjectSummary 项目列表视图（含条目数/合计/版本数）。
export interface CostProjectSummary extends CostProject {
  itemCount: number;
  total: number;
  versionCount: number;
}

// CostEstimateItem 测算明细行（工程量清单行；金额=数量×单价自动计算）。
export interface CostEstimateItem {
  id?: number;
  projectId: string;
  name: string; // kebab 稳定名（沉淀为成本条目 name）
  title: string;
  categoryPath: string;
  unit: string;
  quantity: number;
  price: number;
  amount?: number;
  entryName?: string; // 引用的成本条目 name（可空=手动估价）
  source?: string;
  note?: string;
  sort?: number;
  createdAt?: string;
  updatedAt?: string;
}

// CostEstimateVersion 不可变版本快照（保存时对明细行 JSON 快照 + 合计）。
export interface CostEstimateVersion {
  id: number;
  projectId: string;
  version: number;
  total: number;
  snapshot: string;
  note: string;
  createdAt: string;
}

// CostIndicator 造价参考指标（对案例项目明细行的价格聚合，实时计算不落表）。
export interface CostIndicator {
  key: string; // 科目标题 或 一级分类名
  unit: string;
  samples: number;
  min: number;
  max: number;
  mean: number;
  median: number;
  p25: number;
  p75: number;
}

// CostReviewNote 复盘笔记（结论/边界/风险/证据/可信度/有效期/复核状态）。
export interface CostReviewNote {
  id?: number;
  title: string;
  conclusion: string;
  boundary: string;
  risk: string;
  evidence: string;
  confidence: string; // 高/中/低
  validUntil?: string;
  status: string; // 草稿 / 已确认
  category?: string;
  projectType?: string;
  craft?: string;
  refCount?: number;
  createdAt?: string;
  updatedAt?: string;
}

// 导入预览中的一条候选成本条目（前端可编辑后确认导入）。
export interface CostImportRow {
  name: string;
  title: string;
  category: string;
  unit: string;
  price: number;
  // 综合单价架构：人材机二级 = 合计 + 组成明细；费率仅展示追溯。
  laborFee?: number;
  materialFee?: number;
  machineFee?: number;
  managementFee?: number;
  profitFee?: number;
  advanceFee?: number;
  taxRate?: number;
  components?: CostComponent[];
  body?: string;
  spec: string;
  source: string;
  // 原始工作表物理行号（1-based；0=无法确定，如纵向参数表/AI 解析）。
  sourceRow?: number;
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
  kind: "cost" | "knowledge" | "office" | "file";
  name: string;
  score: number;
  text: string;
}

// BrainHit 是三脑统一检索命中（brain.main 主脑 / brain.left 左脑 / brain.right 右脑）。
export interface BrainHit {
  brain: string;
  entity: string;
  text: string;
  score: number;
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

// ── v4.2 造价 AI 化（docs/gaea-v42-cost-ai-design.md）────────────────
// PriceBand 是相似清单的价格带推荐（Go cost.PriceBand 视图：分位数 R-7 口径）。
export interface PriceBandSource {
  name: string;
  title: string;
  category: string;
  unit: string;
  spec: string;
  source: string;
  region: string;
  priceDate: string;
  priceType: string;
  price: number;
  updatedAt: string;
}
export interface PriceBand {
  samples: number;
  min: number;
  max: number;
  mean: number;
  median: number;
  p25: number;
  p75: number;
  spreadPct: number;
  outliers: number;
  confidence: string;
  sources: PriceBandSource[];
}
// CostComposeEvidence 组价证据链一条：溯源字段（来源/地区/期数/口径）即证据格式。
export interface CostComposeEvidence {
  name: string;
  title: string;
  category: string;
  unit: string;
  spec: string;
  price: number;
  source: string;
  region: string;
  priceDate: string;
  priceType: string;
}
// CostComposeView 是 AI 组价建议（无确认不落库；band=null 表示成本库无相似条目）。
export interface CostComposeView {
  description: string;
  unit: string;
  band: PriceBand | null;
  recommendedPrice: number;
  reason: string;
  components?: CostComponent[];
  componentsNote?: string;
  llmUsed: boolean;
  evidence: CostComposeEvidence[];
}
// ── 询价飞轮（四源归一：信息价/OCR报价/供应商比价/手动询价）──
export interface CostInquiryRecord {
  id: number;
  title: string;
  spec: string;
  unit: string;
  price: number;
  source: string;
  supplier: string;
  region: string;
  priceDate: string;
  validUntil: string;
  note: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}
// CostAdjustSuggestion 调差建议：成本库条目 vs 最新询价数据点（|差幅|>2%）。
export interface CostAdjustSuggestion {
  entryName: string;
  entryTitle: string;
  entryPrice: number;
  latestPrice: number;
  latestDate: string;
  latestSource: string;
  diff: number;
  diffPct: number;
  unit: string;
}
// ── 五算对比（估/概/预/结/决，coststage）──
export interface CostStageValue {
  id: number;
  projectId: string;
  stage: string;
  amount: number;
  date: string;
  note: string;
  createdAt: string;
  updatedAt: string;
}
// CostStageCompareRow 五算对比行：固定 5 阶段顺序，缺阶段 hasValue=false。
export interface CostStageCompareRow {
  stage: string;
  amount: number;
  hasValue: boolean;
  prevStage: string;
  hasPrev: boolean;
  chainDiff: number;
  chainDiffPct: number;
  baseDiff: number;
  baseDiffPct: number;
}
// CostStageDeviation 相邻阶段偏差特征（level: 正常/关注/异常，供复盘诊断）。
export interface CostStageDeviation {
  fromStage: string;
  toStage: string;
  fromAmount: number;
  toAmount: number;
  diff: number;
  diffPct: number;
  direction: string;
  level: string;
  suggestion: string;
}

// SearchScope 是统一检索的空间范围（S1.2-C，docs/gaea-memory-isolation-design.md）：
// ""=全部（旧行为，仅用户显式选择「全部」时使用）；"work"/"play"=只搜对应空间。
// 双空间红线：默认只搜当前空间（GaeaSpaceActive 下发的 space）。
export type SearchScope = "" | "work" | "play";

// UnifiedSearchView 是一次「跨库统一检索」调用的完整结果（记忆统一层第一刀：
// hub 搜索由 4 绑定前端拼装收敛为 1 绑定后端聚合）：
// keyword = 工作区全文关键词命中（轻量 RAG），semantic = 跨库语义命中，
// brain = 三脑命中（brain.main/left/right），files = 文件语义命中。
export interface UnifiedSearchView {
  keyword: WorkspaceSearchHit[];
  semantic: SemanticHitView[];
  brain?: BrainHit[];
  files?: FileSemanticHit[];
}

// RetrievalEvalQuery 是检索质量测评中单条查询的明细：期望命中 vs 实际前 10 命中。
// expected/topHits 均为 "kind:name" 形式（如 "cost:hp300"），便于前端直接对比。
export type RetrievalEvalQuery = WireShape<AppModels.RetrievalEvalQuery>;

// RetrievalEvalReport 是检索质量测评结果：内置查询集跑一遍统一检索，
// 统计平均 recall@10，并与达标门槛比较给出通过状态。
export type RetrievalEvalReport = WireShape<AppModels.RetrievalEvalReport>;

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

export type KnowledgeImportPreview = WireShape<AppModels.KnowledgeImportPreview>;

// 知识条目版本历史快照。
export type KnowledgeHistoryView = WireShape<AppModels.KnowledgeHistoryView>;

// 查重命中的相似条目。
export type SimilarView = WireShape<AppModels.SimilarView>;

// 办公记忆疑似重复对（keep 为建议保留项）。
export type MemoryDuplicateView = WireShape<AppModels.MemoryDuplicateView>;

// 工作区文件语义索引状态 / 命中。
export interface FileIndexStatus {
  total: number;
  skipped: number;
  error: string;
}
export type FileSemanticHit = WireShape<AppModels.FileSemanticHit>;

// ── 阶段 5 T5-1：通用任务调度器视图 ──
// 长任务（价格抓取/文件索引重建等）统一走持久化任务队列；gaea-task 事件
// 实时推送任务视图（状态/进度/消息），任务中心据此渲染并支持取消/重试。
// stopping（C1）：用户已请求取消，等待 handler 退出（结束态细分）。
export type TaskStatus = "queued" | "running" | "stopping" | "succeeded" | "failed" | "cancelled";

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
  // S1 空间归属（后端 Task.Space `json:"spaceId,omitempty"`）：任务中心/角标
  // 按当前空间过滤事件（S2.1 docs/gaea-space-shell-design.md §4.7）。
  spaceId?: string;
  // C9 事件视图字段：gaea-task 事件在输出变更/终态时携带输出尾部整尾回放
  // （有界环形缓冲），输出 dock 事件即推（轮询兜底）；列表/查询响应中缺省。
  outputTail?: string;
  outputTruncated?: boolean;
}

// ── v4.1 证据链（docs/gaea-v41-evidence-chain-design.md §3）────────────────
// JournalChangeRecord 是 GaeaJournalList 返回的证据卡视图（对齐 Go
// evidence.ChangeRecord JSON；wailsjs 生成物刷新前手写，字段漂移由
// typesGenerationCheck 同范式在后续 Step 收口）。
export interface JournalChangeRecord {
  id: string;
  sessionId: string;
  space: string;
  turn: number;
  tool: string;
  target: string;
  beforeSummary: string;
  afterSummary: string;
  model?: string;
  at: number; // unix ms
  status: string;
}

// VerdictView 是 Verifier 双通道复核结论（v4.1b，GaeaVerifyRecord）。
export interface VerdictView {
  id: string;
  status: "verified" | "warned" | "failed";
  channelA?: string;
  channelB?: string;
  note?: string;
  at: number;
}

// LintReportView 是中文规范体检结果（v4.1c，GaeaDocumentLint）。
export interface LintIssueView {
  element: string;
  found: boolean;
  note: string;
}
export interface LintReportView {
  path: string;
  issues: LintIssueView[];
  passed: boolean;
  summary: string;
}

// TaskOutputView 是任务实时输出的尾部回放视图（C1：GaeaTaskOutput）。
export type TaskOutputView = WireShape<AppModels.TaskOutputView>;

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

// ── 编程板块：DeepSeek Harness Web 进程管理 ─────────────────────
// GetProgrammingWebStatus 的返回契约：running=3080 端口在服务；
// owned=该实例由 gaea 自启（StopProgrammingWeb 只杀 owned 实例，不误杀外部进程）；
// uptime_s=自启实例已运行秒数（非自启/未运行恒 0）；log=自启日志路径。
export interface ProgrammingWebStatus {
  running: boolean;
  owned: boolean;
  pid: number;
  url: string; // Web UI 地址（默认 http://127.0.0.1:3080）
  root: string; // harness 仓库根目录
  log: string; // gaea 自启 dsh web 日志路径
  uptime_s: number; // 自启实例已运行秒数（非自启/未运行恒 0）
}

// GetProgrammingWebPreflight 的返回契约：启动前置条件逐项检查
// （harness 目录有效 / pnpm 可用 / 依赖已装 / Web 构建产物就绪 / 端口空闲），
// 启动引导视图据此渲染绿/红清单；all_ready 为真才建议一键启动。
export interface ProgrammingWebPreflight {
  harness_valid: boolean;
  pnpm_found: boolean;
  deps_ready: boolean;
  build_ready: boolean;
  port_free: boolean;
  all_ready: boolean;
  root: string; // 实际检查的 harness 目录
}

// ProgrammingWebLogTail 的返回契约：自启日志尾部（lines 最多 n 行，
// n 由调用方钳制 [1,200]）；日志尚未生成时 exists=false + error 提示。
export interface ProgrammingWebLogTail {
  exists: boolean;
  path: string;
  lines: string[];
  error: string;
}
