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
