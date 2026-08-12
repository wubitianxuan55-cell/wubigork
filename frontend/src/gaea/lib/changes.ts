// 会修改工作区文件的工具（用于「变更」面板汇总）。
export const WRITE_TOOL_NAMES = new Set([
  "write_file", "edit_file", "edit_lines", "multi_edit",
  "move_file", "notebook_edit", "delete_range", "delete_symbol",
]);

function pushPath(out: string[], v: unknown): void {
  if (typeof v === "string" && v.trim() !== "") out.push(v.trim());
}

// 从写类工具的参数里提取被改动的工作区文件路径，与后端
// internal/gaea/evidence/evidence.go 的 extractPaths 及
// internal/gaea/agent/agent_helpers.go 的 extractFilePath 对齐。
export function extractChangedPaths(args: string): string[] {
  let parsed: Record<string, unknown>;
  try {
    parsed = JSON.parse(args || "{}") as Record<string, unknown>;
  } catch {
    return [];
  }
  const out: string[] = [];
  for (const key of ["path", "file_path", "notebook_path", "source", "destination"]) {
    pushPath(out, parsed[key]);
  }
  for (const key of ["paths", "file_paths"]) {
    if (Array.isArray(parsed[key])) (parsed[key] as unknown[]).forEach((v) => pushPath(out, v));
  }
  // multi_edit / edit_file 可能把多个编辑片段放在 edits: [{path, ...}]。
  if (Array.isArray(parsed.edits)) {
    for (const edit of parsed.edits as unknown[]) {
      if (edit && typeof edit === "object") {
        const e = edit as Record<string, unknown>;
        pushPath(out, e.path);
        pushPath(out, e.file_path);
      }
    }
  }
  return out;
}
