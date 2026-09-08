import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";


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
  cat: "system" | "tools" | "user" | "inject" | "assistant" | "tool";
  tokens: number;
  text?: string;
  gone?: number;
  tool?: string; // 产生该结果的工具名（cat=tool 专有，来源 chip）
  err?: boolean; // 工具结果为错误返回（error 语义点）
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
  timing?: ContextTiming;
  rate?: ContextCostRate; // 最近一次用量上报的单价快照（成本 hover 用）
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
