import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";

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
export type GaeaReloadResult = WireShape<AppModels.GaeaReloadResult>;

export interface WireApproval {
  id: string;
  tool: string;
  subject: string;
  // request_permission 权限升级申请（后端仅在该形态下发这两个字段）：
  // request=true 时本卡是模型主动发起的规则申请而非常规工具审批；
  // reason 是模型给出的申请理由原文——审批卡必须原样展示后才可采信。
  request?: boolean;
  reason?: string;
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
  // v4.26：message 事件可选携带子代理来源引用（后端把子代理最终答复回投主
  // 回合时标注）；reducer 落到 assistant item 上，渲染层据此画「子代理」徽标。
  subagentRef?: string;
}

// Bound-method payloads (desktop/app.go).
// v4.34 线B：assistant 历史条目可携带子代理答复引用（Go HistoryMessage.SubagentRef，
// json:"subagentRef,omitempty"，与实时 message 事件 / GaeaResyncItem.subagentRef
// 同键位同名），恢复会话后据此复现「子代理」徽标。wailsjs/go/models.ts 为
// `wails generate module` 生成物且 gitignored，Go 侧字段合入前生成类尚无此键，
// 先以交叉扩展显式补齐；生成物刷新（wails generate module）后两侧一致，此扩展可回收。
export type HistoryMessage = WireShape<AppModels.HistoryMessage> & { subagentRef?: string };

// SessionStatsView 是会话级 token/成本派生统计（后端从事件日志重放 usage 事件）。
// available=false 表示该会话无事件日志（legacy 会话或路径非法），前端不展示
// 历史统计块。
export type SessionStatsView = WireShape<AppModels.SessionStatsView>;

