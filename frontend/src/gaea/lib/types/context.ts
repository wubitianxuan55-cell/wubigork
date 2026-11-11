import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";


export type ContextInfo = WireShape<AppModels.ContextInfo>;

// ─── 上下文视图（dsh-context Go 移植 Phase A）────────────────────

export interface ContextCategory {
  system: number;
  tools: number;
  user: number;
  inject: number;
  skill: number;
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
  skills?: number;
  cacheHitPercent?: number;
  costEstimate?: number;
  // assistant 三拆子计数（v4.542 刀1）：思考/正文/工具调用参数会话合计。
  assistantReasoning?: number;
  assistantAnswerTokens?: number;
  assistantCallTokens?: number;
}

// ContextCatDelta 是「对比上一步」中单个分类的净变化（有变化才出现在 byCat）。
export interface ContextCatDelta {
  cat: ContextSurfaceNode["cat"];
  items?: number;
  tokens: number;
}

// ContextRequestDelta 是请求间模型可见 surface 差分（v4.79）：+=新增/膨胀、
// −=移除/瘦身；first=首个请求（基线=空）；approx=跨压缩（基线结构性改写，近似）。
export interface ContextRequestDelta {
  items: number;
  tokens: number;
  byCat?: ContextCatDelta[];
  approx?: boolean;
  first?: boolean;
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
  // brief 行在浏览器里的跳转锚点（对应 SurfaceNode.seq；2.5d 趋势→浏览器联动）。
  // 0/缺省=无锚点，不渲染跳转。
  briefUserSeq?: number;
  briefRespSeq?: number;
  promptTokens?: number;
  outputTokens?: number;
  cacheHitTokens?: number;
  cacheMissTokens?: number;
  // 请求估算成本（usage 带价格时计得；洞察逐日成本归因数据源）。
  cost?: number;
  estimated?: boolean; // 回合末未见 usage，按估算分类关闭（旧日志/无用量提供方）
  delta?: ContextRequestDelta; // 较上一步净变化（首个请求 first=true）
}

export interface ContextEvent {
  kind: "inject" | "compact" | "prune" | "switch" | "mode";
  seq: number;
  delta?: number;
  source?: string;
  turn: number;
  step: number;
  ts: number;
  from?: string; // kind=switch 专有：切换前供应商实例名
  to?: string; // kind=switch 专有：切换后供应商实例名
}

export interface FileActivity {
  seq: number;
  ts: number;
  turn: number;
  step: number;
  tool: string;
  action: "read" | "write" | "move" | "dir";
  path: string;
  added?: number; // v4.81 行级增量：写类工具从参数确定性提取（+行）
  removed?: number; // −行（edit/multi_edit/edit_lines 被替换区间）
  hits?: number; // grep 命中行数近似（写入端截断时为下界）
}

export interface ContextSurfaceNode {
  seq: number;
  ts?: number; // 事件时间戳（洞察逐日归因/DNA 模式；旧日志缺省）
  cat: "system" | "tools" | "user" | "inject" | "skill" | "assistant" | "tool";
  tokens: number;
  text?: string;
  gone?: number;
  tool?: string; // 产生该结果的工具名（cat=tool/skill 专有，来源 chip）
  err?: boolean; // 工具结果为错误返回（error 语义点）
  name?: string; // 技能名（cat=skill 专有，来自 run_skill 参数）
  // assistant 三拆子计数（cat=assistant 专有；tokens=三者合计）。
  reasoning?: number;
  answerTokens?: number;
  callTokens?: number;
}

// ContextNodeImage 是详情里一张图片引用的缩略卡数据（2.5b 后半；Go App 层
// 解析：绝对路径 + 尺寸 + 官方 patch 口径 token 估算）。
export interface ContextNodeImage {
  ref: string; // 日志中出现的原始引用
  path: string; // 解析后的绝对路径（供 AttachmentDataURL 加载）
  refCwd?: string; // 相对引用所依据的 cwd
  exists?: boolean; // 文件不存在=false（灰态）
  width?: number; // 像素；0/缺省=解码失败（尺寸未知）
  height?: number;
  scaledW?: number; // 标准档缩放后尺寸
  scaledH?: number;
  stdTokens?: number; // 标准档估算 token
  highTokens?: number; // 高分辨率档估算（悬停详情）
}

// ContextNodeDetailView 是浏览器节点「完整调用」详情（v4.80 懒加载：
// GaeaContextNodeDetail 按 seq 回读当前会话日志）。
export interface ContextNodeDetailView {
  seq: number;
  kind: "tool_result" | "user_message" | "assistant_message";
  ts?: number;
  tool?: string;
  args?: string;
  output?: string;
  err?: string;
  truncated?: boolean; // 日志写入端即已截断
  text?: string;
  lines?: number;
  clamped?: boolean; // 详情超返回上限被截断
  imageRefs?: string[]; // 详情文本/参数里提取的图片引用（纯函数层）
  images?: ContextNodeImage[]; // 解析结果（App 层 I/O；与 imageRefs 对应）
}

// ContextCostRate 是成本费率快照（2.5e 成本 hover；每 1M tokens，供应商口径）。
export interface ContextCostRate {
  inputPer1M?: number;
  outputPer1M?: number;
  cacheHitPer1M?: number;
  currency?: string;
}

// ContextCompactTiers 是内核生效的三层压缩水位（窗口占比 0~1）：
// soft=一次性增长提醒 / ratio=触发摘要压缩 / force=强制压缩。
// 窗口未知或内核未初始化时整体缺席——前端退回静态阈值呈现。
export interface ContextCompactTiers {
  soft: number;
  ratio: number;
  force: number;
}

// ContextCommandRecord 是一次 shell 命令执行的生命周期（v4.542 刀1，
// 对齐 dsh-context CommandRecord 诚实子集）。
export interface ContextCommandRecord {
  seq: number;
  resultSeq?: number;
  time: number;
  tool: string;
  cmd?: string; // 截断预览（全文在日志）
  turn: number;
  step: number;
  start: number;
  end?: number;
  status: "completed" | "failed" | "killed";
  err?: boolean;
  exit?: number; // 仅 bash JSON 信封可提取；plain 输出留零不猜
  signal?: string;
  outChars?: number;
  truncated?: boolean;
  background?: boolean; // run_in_background 派发
  derived?: boolean; // 恒 true：从 bash 工具调用推导
}

// ContextCommandStats 是会话级命令执行摘要。
export interface ContextCommandStats {
  runs: number;
  failed: number;
  ms?: number;
  bg?: number;
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
  files: FileActivity[];
  commands: ContextCommandRecord[]; // shell 命令执行时间线（上限 400 条）
  commandStats?: ContextCommandStats; // 命令执行摘要（无命令时缺席）
  nodesDropped?: number; // 浏览器节点因上限丢弃数（诚实边界）
  nodesFloor?: number; // 幸存节点最小 seq（跳转判界）
  timing?: ContextTiming;
  rate?: ContextCostRate; // 最近一次用量上报的单价快照（成本 hover 用）
  compact?: ContextCompactTiers; // 压缩水位块（窗口未知时缺席）
}

// 耗时统计条目：单个工具名的调用次数与执行时长合计（timing.tools 排行项）。
export interface ContextToolTiming {
  name: string;
  calls: number;
  ms: number;
}

// 耗时统计（对齐 dsh-context TimingTotals 的诚实近似版）：
// wallMs=各轮次活跃时长合计；ttftMs=模型等待（步骤起点→首 token）；
// genMs=生成（首 token→assistant 消息收尾）；calls=模型调用次数；
// toolsMs/toolCalls=工具执行时长合计/配对数（并行重复计，与 dsh 同口径）；
// tools=每工具名 {calls, ms} 排行，按 ms 降序截断 20（数组承载排行序）。
// 日志时间戳为秒级：所有 ms 为秒粒度近似；日志无法支撑的指标整体省略。
export interface ContextTiming {
  wallMs?: number;
  ttftMs?: number;
  genMs?: number;
  calls?: number;
  toolsMs?: number;
  toolCalls?: number;
  tools?: ContextToolTiming[];
}

// ─── 跨会话洞察（v4.542 刀2/刀3，对标 dsh-context Context Insights）──

export type InsightsRangeKey = "today" | "7d" | "30d" | "all";

// InsightsKpi：Tokens/Cost/Requests/缓存命中率按范围内请求归因；Turns 按
// 范围内轮次去重；ToolCalls/Skills 为会话全生命周期合计；ToolsMs/WallMs
// 为会话生命周期合计（日志不支持按日切分）。estimatedTokens 诚实分列。
export interface InsightsKpi {
  sessions: number;
  tokens: number;
  requests: number;
  turns: number;
  toolCalls: number;
  skills: number;
  cost?: number;
  cacheHitPercent?: number;
  toolsMs?: number;
  wallMs?: number;
  estimatedTokens?: number;
}

export interface InsightsDay {
  date: string; // 本地时区 YYYY-MM-DD
  tokens: number;
  requests: number;
  turns?: number;
  toolCalls?: number;
  skills?: number;
  cost?: number;
}

export interface InsightsTally {
  name: string;
  count: number;
  last?: number;
}

export interface InsightsSession {
  path: string;
  title?: string;
  preview?: string;
  spaceId?: string;
  archived?: boolean;
  modTime: number;
  lastActive: number;
  tokens: number;
  requests: number;
  turns: number;
  cost?: number;
  cacheHitPercent?: number;
  composition: ContextCategory;
  estimated?: boolean; // 范围内全部请求均为估算关闭
}

export interface ContextInsightsView {
  ok: boolean;
  range: InsightsRangeKey;
  generatedAt: number;
  kpi: InsightsKpi;
  days: InsightsDay[]; // 按日升序，仅含范围内有活动的日子
  skills: InsightsTally[];
  tools: InsightsTally[];
  programs: InsightsTally[];
  sessions: InsightsSession[];
  scanned: number;
  truncated?: number; // 因上限未折叠的会话数（诚实边界）
}
