import type { Item } from "./store";

export interface SessionChange {
  path: string;
  count: number;
  lastTouched?: number;
}

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

// 从会话消息里汇总“写/改过的工作区文件及次数”，按最近改动倒序返回。
export function buildSessionChanges(
  items: Item[],
  writeTools: ReadonlySet<string> = WRITE_TOOL_NAMES,
): SessionChange[] {
  const map = new Map<string, { count: number; lastTouched: number }>();
  items.forEach((it, idx) => {
    if (it.kind !== "tool" || !writeTools.has(it.name)) return;
    for (const p of extractChangedPaths(it.args || "")) {
      const cur = map.get(p) ?? { count: 0, lastTouched: idx };
      map.set(p, { count: cur.count + 1, lastTouched: idx });
    }
  });
  return [...map.entries()]
    .map(([path, v]) => ({ path, count: v.count, lastTouched: v.lastTouched }))
    .sort((a, b) => b.lastTouched - a.lastTouched);
}

// 从写类工具参数提取「交付物」路径（成果面板显式登记用）：与
// extractChangedPaths 的差别是 move_file 只计 destination——源路径
// 不是交付物；去重保持出现顺序。
export function extractDeliverablePaths(args: string): string[] {
  let parsed: Record<string, unknown>;
  try {
    parsed = JSON.parse(args || "{}") as Record<string, unknown>;
  } catch {
    return [];
  }
  const out: string[] = [];
  const seen = new Set<string>();
  const push = (v: unknown) => {
    if (typeof v === "string" && v.trim() !== "") {
      const p = v.trim();
      if (!seen.has(p)) {
        seen.add(p);
        out.push(p);
      }
    }
  };
  for (const key of ["path", "file_path", "notebook_path", "destination"]) {
    push(parsed[key]);
  }
  for (const key of ["paths", "file_paths"]) {
    if (Array.isArray(parsed[key])) (parsed[key] as unknown[]).forEach(push);
  }
  if (Array.isArray(parsed.edits)) {
    for (const edit of parsed.edits as unknown[]) {
      if (edit && typeof edit === "object") {
        const e = edit as Record<string, unknown>;
        push(e.path);
        push(e.file_path);
      }
    }
  }
  return out;
}
