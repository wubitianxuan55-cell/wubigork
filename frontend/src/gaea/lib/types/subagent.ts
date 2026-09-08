// ── 多智能体分工可见（P2，对标 WorkSwarm 蜂群 / QClaw V2） ──
export interface SubagentRunView {
  ref: string; // sa_YYYYMMDD_HHMMSS_... / mt_YYYYMMDD_HHMMSS_... 稳定引用
  status: "running" | "completed" | "failed";
  // kind 区分两类运行：subagent（task/run_skill 派生的真子代理）与
  // model_tool（vision/summarize_file 等本地模型工具的单轮调用）。旧数据
  // 缺省按 subagent 解析。
  kind?: "subagent" | "model_tool";
  tool?: string; // 仅 model_tool：发起工具名（vision / summarize_file…）
  model?: string;
  toolScope?: string[];
  task: string; // meta.Title 优先，回退 transcript 首条 user 消息（任务摘要）
  answer?: string; // 最后一条 assistant 回答摘要
  toolCalls: number;
  lastText?: string; // C2 活动行：最后一段 assistant 文本（运行中实时更新）
  lastTool?: string; // C2 活动行：最后一次工具调用摘要（name + 结果头）
  // 最近一次追问的后台失败原因摘要（v4.66，meta 透传）：非空 = 该运行最近
  // 一次追问失败；空/缺省 = 无失败。
  followUpError?: string;
  createdAt: string;
  updatedAt: string;
}

export interface SubagentRunsView {
  available: boolean; // false = 会话无 subagents 目录（未派发子代理）
  runs: SubagentRunView[];
  total: number;
  running: number;
}

export interface SubagentTranscriptMessage {
  role: "system" | "user" | "assistant" | "tool";
  name?: string;
  content?: string;
  reasoning?: string;
  toolCalls?: { id: string; name: string; arguments: string }[];
  toolCallId?: string;
}

export interface SubagentTranscriptView {
  ref: string;
  task?: string;
  // 最近一次追问的后台失败原因摘要（v4.66，meta 透传，omitempty）：会话 tab
  // 的追问轮询凭它把乐观气泡转失败态（错误条文案 = 该原因）。
  followUpError?: string;
  messages: SubagentTranscriptMessage[];
}
