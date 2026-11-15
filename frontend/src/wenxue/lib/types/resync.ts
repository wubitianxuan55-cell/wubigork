// ── 事件序号防线（v4.26 对话流式重造） ─────────────────────
// GaeaResyncEvents 的返回：Wails 事件流吞件（gaea-event 密集到达丢件）时，
// 前端检测 seq 跳号 → 补拉当前会话磁盘日志折叠出的对话项全量快照整体替换。
export type GaeaResyncItem = {
  kind: "user" | "assistant" | "tool" | "notice";
  id: string;
  text?: string;
  reasoning?: string;
  level?: string;
  name?: string;
  toolId?: string;
  output?: string;
  err?: string;
  status?: "done" | "error" | "running";
  readOnly?: boolean;
  truncated?: boolean;
  parentId?: string;
  // v4.27.2：assistant 条目可携带子代理来源引用（subagent_message 折叠），
  // 渲染层据此画「子代理」徽标（与实时 message 事件同键位）。
  subagentRef?: string;
};

export interface GaeaResyncResult {
  seq: number; // 转发层当前最新 wire seq：整体替换后以此为 lastSeq 续接
  items: GaeaResyncItem[]; // 会话全量（流式 delta 已合并为单 assistant 条目），恒非空
}
