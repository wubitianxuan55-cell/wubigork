// Per-tool presentation helpers. The kernel forwards every tool call the same way
// (name + raw-JSON args + output); these turn that generic payload into the
// recognizable one-liner, inline diff, and collapsed outcome each tool deserves —
// the recognizable "card" vocabulary the desktop uses. Kept pure (no React, no
// highlight.js) so ToolCard stays a renderer and the main bundle stays light.

import { diffLines } from "./diff";
import { t } from "./i18n";
import { extToLang } from "./lang";
import type { DictKey } from "../locales/en";

export interface ToolDiff {
  original: string;
  modified: string;
  lang: string;
  label?: string; // multi_edit labels each step ("edit 1", …)
}

/** 写工具的行级 diffstat（Codex 式 "Edited file +2−2" 芯片数据源）。 */
export interface DiffStat {
  add: number;
  del: number;
}

function parse(args: string): Record<string, unknown> {
  try {
    return JSON.parse(args) as Record<string, unknown>;
  } catch {
    return {};
  }
}

function str(a: Record<string, unknown>, key: string): string {
  return typeof a[key] === "string" ? (a[key] as string) : "";
}

// subjectOf pulls the most informative one-liner out of a call's args — the
// command for bash, the pattern for search, the path for file tools, the
// description for a sub-task — so the collapsed row reads at a glance.
export function subjectOf(name: string, args: string): string {
  const a = parse(args);
  switch (name) {
    case "bash":
      return str(a, "command");
    case "grep":
    case "glob":
      return str(a, "pattern") || str(a, "path");
    case "web_fetch":
      return str(a, "url");
    case "web_search":
      return str(a, "query");
    case "task":
      return str(a, "description") || str(a, "prompt");
    case "remember":
      return str(a, "name") || str(a, "description");
      return ""; // dedicated card, not a subject line
    // ── 办公工具：行首主体（路径/标题/查询词）────────────────────────
    case "chart_gen":
    case "diagram_gen":
      return str(a, "title");
    case "knowledge_search":
    case "memory_search":
      return str(a, "query");
    case "knowledge_add":
      return str(a, "title");
    case "read_skill":
      return str(a, "name");
    case "move_file": {
      const from = str(a, "source") || str(a, "path");
      const to = str(a, "destination");
      return to ? `${from} → ${to}` : from;
    }
    default:
      return str(a, "path") || str(a, "file_path");
  }
}

// diffsFor returns the before/after pairs a writer tool's card renders inline:
// edit_file is one pair, write_file is an all-add (empty original), multi_edit is
// one pair per step. Returns [] for non-writers, so the card folds args/output
// away instead.
export function diffsFor(name: string, args: string): ToolDiff[] {
  const a = parse(args);
  const lang = extToLang(str(a, "path") || str(a, "file_path"));
  if (name === "edit_file") {
    if (typeof a.old_string === "string" && typeof a.new_string === "string") {
      return [{ original: a.old_string, modified: a.new_string, lang }];
    }
  }
  if (name === "write_file" && typeof a.content === "string") {
    return [{ original: "", modified: a.content, lang }];
  }
  if (name === "edit_lines") {
    if (typeof a.new_content === "string") {
      const oldLines = `[lines ${a.start_line}-${a.end_line}]`;
      return [{ original: oldLines, modified: a.new_content, lang }];
    }
  }
  if (name === "multi_edit" && Array.isArray(a.edits)) {
    const out: ToolDiff[] = [];
    (a.edits as unknown[]).forEach((e, i) => {
      const step = e as Record<string, unknown>;
      if (typeof step?.old_string === "string" && typeof step?.new_string === "string") {
        out.push({ original: step.old_string, modified: step.new_string, lang, label: `edit ${i + 1}` });
      }
    });
    return out;
  }
  return [];
}

// diffStatFor 汇总写工具的参数差异为 +N/−N 计数；非编辑类工具返回 null。
// 与 diffsFor 同一参数口径：edit_file 一组、multi_edit 逐组累加、其余不适用。
export function diffStatFor(name: string, args: string): DiffStat | null {
  const a = parse(args);
  if (name === "edit_file") {
    if (typeof a.old_string === "string" && typeof a.new_string === "string") {
      return plusMinus(a.old_string, a.new_string);
    }
    return null;
  }
  if (name === "multi_edit" && Array.isArray(a.edits)) {
    let add = 0;
    let del = 0;
    for (const e of a.edits as Record<string, unknown>[]) {
      if (typeof e?.old_string === "string" && typeof e?.new_string === "string") {
        const pm = plusMinus(e.old_string, e.new_string);
        add += pm.add;
        del += pm.del;
      }
    }
    return { add, del };
  }
  return null;
}

export type TodoStatus = "pending" | "in_progress" | "completed";

export interface Todo {
  content: string;
  status: TodoStatus | string;
  activeForm?: string;
  level?: number; // 0 = phase, 1 = sub-step of the phase above it
}

// parseTodos pulls the task list out of a todo_write call's args.
export function parseTodos(args: string): Todo[] {
  try {
    const a = JSON.parse(args) as { todos?: Todo[] };
    return Array.isArray(a.todos) ? a.todos : [];
  } catch {
    return [];
  }
}

function plusMinus(original: string, modified: string): { add: number; del: number } {
  let add = 0;
  let del = 0;
  for (const r of diffLines(original, modified)) {
    if (r.type === "add") add++;
    else if (r.type === "del") del++;
  }
  return { add, del };
}

// lineCount counts lines, ignoring a single trailing newline so "a\n" reads as 1.
function lineCount(s: string): number {
  if (!s) return 0;
  const t = s.endsWith("\n") ? s.slice(0, -1) : s;
  return t === "" ? 0 : t.split("\n").length;
}

function nonEmptyLines(s: string): number {
  return s.split("\n").filter((l) => l.trim() !== "").length;
}

// countOf renders a localized "N <noun>" using the singular/plural key pair (zh
// collapses both to one form). Lives here, not the dict, so the counted phrasing
// stays a translation concern.
function countOf(n: number, one: DictKey, other: DictKey): string {
  return t(n === 1 ? one : other, { n });
}

// extractOutputFromEnvelope tries to unwrap the ToolEnvelope JSON that the backend
// wraps around web_fetch/web_search results, returning the real output text.
function extractOutputFromEnvelope(output: string): string {
  try {
    const env = JSON.parse(output) as { ok?: boolean; data?: { result?: string }; message?: string };
    if (env.ok && env.data?.result) return env.data.result;
    if (env.ok && env.message) return env.message;
  } catch {}
  return output;
}

// summarize derives the one-line outcome shown under a finished card (the "⎿"
// secondary line) — counts from the args for writers, from the output for
// readers. "" means there's nothing worth a summary line.
export function summarize(name: string, args: string, output?: string, error?: string): string {
  if (error) return "";
  const a = parse(args);
  switch (name) {
    case "write_file":
      return countOf(lineCount(str(a, "content")), "tool.lineOne", "tool.lineOther");
    case "edit_lines": {
      const start = typeof a.start_line === "number" ? a.start_line : 0;
      const end = typeof a.end_line === "number" ? a.end_line : 0;
      
      const newLineCount = typeof a.new_content === "string" ? lineCount(a.new_content) : 0;
      return `L${start}-${end} → ${countOf(newLineCount, "tool.lineOne", "tool.lineOther")}`;
    }
    case "edit_file": {
      // diffstat 由 ToolCard 的芯片渲染（diffStatFor），摘要不再重复输出。
      return "";
    }
    case "multi_edit": {
      const edits = Array.isArray(a.edits) ? (a.edits as Record<string, unknown>[]) : [];
      // 只保留「N 处修改」；行级增减由 diffStatFor 芯片展示。
      return countOf(edits.length, "tool.editOne", "tool.editOther");
    }
  }

  if (!output) return "";
  switch (name) {
    case "read_file": {
      if (output.startsWith("(empty file)")) return t("tool.emptyFile");
      const arrows = (output.match(/→/g) || []).length;
      return countOf(arrows || lineCount(output), "tool.lineOne", "tool.lineOther");
    }
    case "grep":
      return countOf(nonEmptyLines(output), "tool.matchOne", "tool.matchOther");
    case "glob":
      return countOf(nonEmptyLines(output), "tool.fileOne", "tool.fileOther");
    case "ls":
      return countOf(nonEmptyLines(output), "tool.entryOne", "tool.entryOther");
    case "web_fetch": {
      const text = extractOutputFromEnvelope(output);
      const lines = text.split("\n").filter(l => !l.startsWith("status "));
      const first = lines[0] || "";
      return first.slice(0, 80);
    }
    case "web_search": {
      const text = extractOutputFromEnvelope(output);
      const count = (text.match(/^\d+\. /gm) || []).length;
      return count > 0 ? countOf(count, "tool.resultOne", "tool.resultOther") : "";
    }
    case "bash":
      return output.trim() === "" ? t("tool.noOutput") : countOf(lineCount(output), "tool.lineOne", "tool.lineOther");
    // ── 办公工具：完成态一行结果摘要（此前这些卡除文件名外全空白）──────
    case "format_convert": {
      const text = extractOutputFromEnvelope(output);
      const saved = text.match(/已转换并保存为\s+(\S+?)（(\d+) 字符）/);
      if (saved) {
        return `${saved[1]} · ${countOf(Number(saved[2]), "tool.charOne", "tool.charOther")}`;
      }
      const head = text.match(/^# 文档转换:\s*(.+)$/m);
      return head ? head[1].trim() : "";
    }
    case "chart_gen": {
      const text = extractOutputFromEnvelope(output);
      const type = text.match(/类型:\s*(\w+)/)?.[1];
      if (!type) return "";
      const n = Number(text.match(/数据点:\s*(\d+)/)?.[1] ?? text.match(/类别:\s*(\d+)/)?.[1] ?? 0);
      return n > 0 ? `${type} · ${countOf(n, "tool.pointOne", "tool.pointOther")}` : type;
    }
    case "diagram_gen": {
      // diagram_gen 返回裸 JSON（非 ToolEnvelope）
      try {
        const r = JSON.parse(output) as { ok?: boolean; output?: string };
        if (r.ok && r.output) return r.output;
      } catch {}
      return "";
    }
    case "knowledge_add": {
      const text = extractOutputFromEnvelope(output);
      return text.includes("已保存知识条目") ? t("tool.kbSaved") : "";
    }
    case "knowledge_search": {
      const text = extractOutputFromEnvelope(output);
      const n = (text.match(/^### /gm) || []).length;
      return n > 0 ? countOf(n, "tool.hitOne", "tool.hitOther") : "";
    }
    case "memory_search": {
      // memory_search 直接返回纯文本编号列表（非信封）
      const numbered = (output.match(/^\d+\. /gm) || []).length;
      const more = output.match(/\.\.\. and (\d+) more/);
      const total = numbered + (more ? Number(more[1]) : 0);
      return total > 0 ? countOf(total, "tool.hitOne", "tool.hitOther") : "";
    }
    case "read_skill":
      return countOf(lineCount(output), "tool.lineOne", "tool.lineOther");
    default:
      return "";
  }
}

// ── 大工具输出有界预览（P2-2，调研 2026-08-16）────────────────────────
// 超长工具输出（bash/web_fetch/read_file 等）直接全量渲染会撑爆卡片、
// 拖慢 Transcript。boundedOutput 把输出折叠为「头部 + 折叠计数」，
// 由 ToolCard 提供「展开全部」开关（对标 QwenPaw 超长工具输出折叠）。
export interface BoundedOutput {
  /** 折叠状态下展示的文本（尾部追加折叠提示行）。 */
  preview: string;
  /** 完整输出。 */
  full: string;
  /** 是否发生了折叠（totalLines > maxPreviewLines）。 */
  collapsed: boolean;
  /** 折叠掉的行数。 */
  hiddenLines: number;
  /** 输出总行数。 */
  totalLines: number;
}

export const TOOL_OUTPUT_MAX_PREVIEW_LINES = 60;

export function boundedOutput(
  output: string | undefined,
  maxPreviewLines: number = TOOL_OUTPUT_MAX_PREVIEW_LINES,
): BoundedOutput {
  const full = output ?? "";
  if (full === "") {
    return { preview: "", full, collapsed: false, hiddenLines: 0, totalLines: 0 };
  }
  const lines = full.split("\n");
  const totalLines = lines.length;
  if (totalLines <= maxPreviewLines) {
    return { preview: full, full, collapsed: false, hiddenLines: 0, totalLines };
  }
  const head = lines.slice(0, maxPreviewLines).join("\n");
  const hiddenLines = totalLines - maxPreviewLines;
  return {
    preview: `${head}\n… 已折叠 ${hiddenLines} 行`,
    full,
    collapsed: true,
    hiddenLines,
    totalLines,
  };
}
