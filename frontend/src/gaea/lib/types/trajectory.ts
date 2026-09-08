// ─── 轨迹视图（对齐 DSH ui-trajectory 事件账本）────────────────

export interface Trajectory {
  ok: boolean;
  turns: TrajectoryTurn[];
  betweenTurns?: TrajectoryRecord[];
}

export interface TrajectoryTurn {
  turn: number;
  startedAt?: number;
  // v4.31 轮级耗时（ms，= (turn_done.Ts − turn_started.Ts) × 1000，后端
  // fold.go 在 turn_done 分支计算）：收 v4.26「WorkHeader 历史轮无耗时数据源」
  // 欠账——历史轮读盘折叠自带耗时，轨迹轮次头在轮结束时展示「用时 Ns」。
  // omitempty 语义：悬挂轮/时钟异常/旧日志缺省（老后端不下发该字段）。
  durationMs?: number;
  end?: TrajectoryTurnEnd;
  records: TrajectoryRecord[];
}

export interface TrajectoryTurnEnd {
  seq: number;
  ts: number;
  err?: string;
}

export type TrajectoryRecordKind =
  | "user" | "header" | "assistant" | "tool" | "compact" | "ask" | "approval" | "subagent";

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
  subagent?: TrajectorySubagentRec;
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

// 子代理完成回投记录（v4.26，对齐后端 trajectory.SubagentRec）：task 子代理
// 完成时把最终答复文本回投父回合。ref 为子代理 transcript 引用（临时子代理
// 为空）；parentId 是父 task 调用 ID；text 展示级截断（最长 2000 rune）。
export interface TrajectorySubagentRec {
  ref?: string;
  text?: string;
  parentId?: string;
}

export interface TrajectoryUsage {
  promptTokens?: number;
  completionTokens?: number;
  cacheHitTokens?: number;
  cacheMissTokens?: number;
  reasoningTokens?: number;
}
