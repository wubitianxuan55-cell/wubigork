// 文学工作台不承接办公知识库、CAD 与进度计划能力。后端仍共享办公引擎，
// 因此前端在命令、工具卡和统计三个可见面做一致过滤。
const DISABLED_COMMAND_PREFIXES = ["knowledge", "schedule", "cad"] as const;
const DISABLED_TOOL_PREFIXES = ["knowledge_", "schedule_", "cad_"] as const;

function hasPrefix(value: string, prefixes: readonly string[]): boolean {
  const lower = value.trim().toLowerCase();
  return prefixes.some((prefix) => lower === prefix || lower.startsWith(prefix));
}

export function isDisabledWenxueCommand(name: string): boolean {
  return hasPrefix(name, DISABLED_COMMAND_PREFIXES);
}

export function isDisabledWenxueTool(name: string): boolean {
  return hasPrefix(name, DISABLED_TOOL_PREFIXES);
}
