// 文学工作台（图文）不承接办公域能力。后端引擎侧的收敛在刀1 落地
// （internal/app/wenxue_tool_policy.go deny 名单，与本文目名单同源对齐）；
// 前端在命令、工具卡、统计、能力面板四个可见面做一致过滤。
const DISABLED_COMMAND_PREFIXES = ["knowledge", "schedule", "cad"] as const;
const DISABLED_TOOL_PREFIXES = ["knowledge_", "schedule_", "cad_"] as const;

// 办公域工具（刀1 引擎侧物理卸载的同一批）：本地面板兜底不展示。
const DISABLED_TOOL_NAMES = [
  "office_generate", "office_batch",
  "docx_apply", "docx_accept", "pptx_apply",
  "xlsx_apply", "xlsx_profile", "xlsx_recalc", "xlsx_audit",
  "office_open", "office_status", "office_read", "office_edit", "office_render",
  "office_save", "office_close", "office_help", "office_macro", "office_selection",
  "cad_ask", "cost_search", "cost_save", "cost_compose", "cost_indicators",
  "chart_gen", "diagram_gen", "diagram", "doc_render", "format_convert",
  "screen_capture", "dag_plan", "semantic_search", "browser_",
] as const;

// 办公域技能（刀2 引擎侧技能子集的补集）：能力面板兜底不展示。
// 注意 docx/pptx 技能只藏索引，导出链脚本查找走路径不受影响。
const DISABLED_SKILL_NAMES = [
  "gongwen-9704", "office-gongwen", "cad-takeoff", "office-edit", "schedule-edit",
  "schedule-dcma", "cost-compose", "format-convert", "chart-builder", "doc-assemble",
  "docx", "pptx", "pdf", "xlsx", "pptx-template",
] as const;

function hasPrefix(value: string, prefixes: readonly string[]): boolean {
  const lower = value.trim().toLowerCase();
  return prefixes.some((prefix) => lower === prefix || lower.startsWith(prefix));
}

export function isDisabledWenxueCommand(name: string): boolean {
  return hasPrefix(name, DISABLED_COMMAND_PREFIXES);
}

export function isDisabledWenxueTool(name: string): boolean {
  if (hasPrefix(name, DISABLED_TOOL_PREFIXES)) return true;
  const lower = name.trim().toLowerCase();
  return DISABLED_TOOL_NAMES.some((n) => lower === n || lower.startsWith(n));
}

export function isDisabledWenxueSkill(name: string): boolean {
  const lower = name.trim().toLowerCase();
  return DISABLED_SKILL_NAMES.some((n) => lower === n);
}
