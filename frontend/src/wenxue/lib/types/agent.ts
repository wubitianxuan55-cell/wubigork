// ─── Agent 网络（dsh-context Agent network 卡）────────────────

export interface AgentNetwork {
  ok: boolean;
  window: number;
  root: AgentNode;
}

export interface AgentNode {
  id: string;
  name: string;
  // fold = 折叠聚合卡（D3 fleetModel 合成，wire 不出现；done/idle 两组信息
  // 在 FleetFoldInfo 上）。
  kind: "root" | "subagent" | "fold";
  // queued = 并行闸排队中；aborted = 用户中止（D4 结算词汇，run 状态直达；
  // 工具调用骨架只有 running/completed/error 三态）。
  status: "running" | "queued" | "completed" | "error" | "failed" | "aborted";
  model?: string;
  task?: string;
  toolCalls: number;
  errors: number;
  tokens: number;
  firstTs?: number;
  lastTs?: number;
  // C2 活动行（D1 会话骨架后由 run 直接落节点）：最后一段 assistant 文本与
  // 最后一次工具调用摘要；工具调用骨架无此数据。
  lastText?: string;
  lastTool?: string;
  children?: AgentNode[];
}
