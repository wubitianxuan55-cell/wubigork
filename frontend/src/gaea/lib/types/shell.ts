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
