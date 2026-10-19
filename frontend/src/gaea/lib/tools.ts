// Per-tool presentation helpers. The kernel forwards every tool call the same way
// (name + raw-JSON args + output); these turn that generic payload into the
// recognizable one-liner, inline diff, and collapsed outcome each tool deserves —
// the recognizable "card" vocabulary the desktop uses. Kept pure (no React, no
// highlight.js) so ToolCard stays a renderer and the main bundle stays light.

import { diffLines } from "./diff";
import { t } from "./i18n";
import { extToLang } from "./lang";
import { parseWriteArgs } from "./toolArgs";
import type { DictKey } from "../locales/en";
import type { DiffRow } from "./diff";

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

// truncateSubject 行内主体词界截断（v4.507）：硬 slice 会把词切成残词
// （「# 编码」→「# b.」），优先回退到截断点前最后一个空格；空格太靠前
// （<60% 处，中文/路径串常态）则硬切——避免为凑词界丢掉大半内容。
export function truncateSubject(s: string, cap: number): string {
  if (s.length <= cap) return s;
  const cut = s.slice(0, cap);
  const lastSpace = cut.lastIndexOf(" ");
  const head = lastSpace > cap * 0.6 ? cut.slice(0, lastSpace) : cut;
  return `${head}…`;
}

// 行内主体的最终截断上限（单卡折叠行与组行同源）。
export const SUBJECT_CAP = 48;

// commandHeadline bash 命令的折叠行标题（v4.507）：多行命令（heredoc /
// python -c / 续行）此前整条压平进单行成「一大串」——取首个非空行、行内
// 空白压单空格，再词界截断。完整命令在展开态 $ 命令块里，不丢信息。
export const BASH_SUBJECT_CAP = 64;

export function commandHeadline(command: string, cap: number = BASH_SUBJECT_CAP): string {
  if (!command) return "";
  const firstLine = command.split("\n").map((l) => l.trim()).find((l) => l !== "") ?? "";
  if (firstLine === "") return "";
  return truncateSubject(firstLine.replace(/\s+/g, " "), cap);
}

// subjectOf pulls the most informative one-liner out of a call's args — the
// command for bash, the pattern for search, the path for file tools, the
// description for a sub-task — so the collapsed row reads at a glance.
export function subjectOf(name: string, args: string): string {
  const a = parse(args);
  switch (name) {
    case "bash":
      // v4.507：多行命令取首行 + 词界截断——整条压平上屏是「一大串」主源
      return commandHeadline(str(a, "command"));
    case "grep":
    case "glob": {
      // v4.507：长正则截 32 + 路径只留 basename——`^施\s*工|… , D:/新工作/…`
      // 整串上屏同样是「一大串」；pattern 缺失回退 path（历史口径）。
      const pattern = truncateSubject(str(a, "pattern"), 32);
      const rawPath = str(a, "path");
      if (!pattern) return rawPath;
      const base = rawPath.replace(/[\\/]+$/, "").split(/[\\/]/).pop() ?? "";
      return base ? `${pattern} @ ${base}` : pattern;
    }
    case "web_fetch":
      return str(a, "url");
    case "web_search":
      return str(a, "query");
    case "task":
      return str(a, "description") || str(a, "prompt");
    case "interrupt_agent":
      return str(a, "ref");
    case "remember":
      return str(a, "name") || str(a, "description");
      return ""; // dedicated card, not a subject line
    // ── 办公工具：行首主体（路径/标题/查询词）────────────────────────
    case "chart_gen":
    case "diagram_gen":
      return str(a, "title");
    case "schedule_get":
    case "schedule_apply":
    case "schedule_analyze":
      // 缺省路径在 Go 侧解析（当前计划），args 里可能没有——给可读回退
      return str(a, "path") || "当前计划";
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

// commandOf 提取 shell 类工具的命令文本（bash 的 args.command），供展开态
// 渲染终端命令块——替代整包 JSON args（多行命令在 JSON 里是 \n 转义长串，
// 可读性极差）。非 shell 工具返回 ""，回退现有 args JSON 展示。
export function commandOf(name: string, args: string): string {
  if (name !== "bash") return "";
  return str(parse(args), "command");
}

// diffsFor returns the before/after pairs a writer tool's card renders inline:
// edit_file is one pair, write_file is an all-add (empty original), multi_edit is
// one pair per step. Returns [] for non-writers, so the card folds args/output
// away instead.
// FE4-02：字段识别与降级口径取自 toolArgs.parseWriteArgs（与「变更」tab 同源）；
// 本函数只决定**怎么摆**（原文不归一、write_file 视为全增、multi_edit 按原始
// 下标标 "edit N"、edit_lines 用 "[lines s-e]" 占位原文行）。
export function diffsFor(name: string, args: string): ToolDiff[] {
  const p = parseWriteArgs(name, args);
  const lang = extToLang(p.path);
  if (name === "edit_file") {
    // hunks[0] 存在 ⟺ old_string/new_string 均为字符串（含 old 为空串的整段新增）
    const h = p.hunks[0];
    return h ? [{ original: h.old, modified: h.new, lang }] : [];
  }
  if (name === "write_file") {
    // 覆盖写入：旧内容未记录，按全增展示；无 content 则不出 diff
    return p.content === undefined ? [] : [{ original: "", modified: p.content, lang }];
  }
  if (name === "edit_lines") {
    if (p.content === undefined) return [];
    // 原行未记录：用「[lines s-e]」占位原文行（缺行号时逐字显示 undefined，历史口径）
    const placeholder = `[lines ${String(p.lineRange?.start)}-${String(p.lineRange?.end)}]`;
    return [{ original: placeholder, modified: p.content, lang }];
  }
  if (name === "multi_edit") {
    // 标注用**原始下标**（与变更 tab 的有效片段连号不同，见 toolArgs 分叉 C）
    return p.hunks.map((h) => ({ original: h.old, modified: h.new, lang, label: `edit ${h.index + 1}` }));
  }
  return [];
}

// diffStatFor 汇总写工具的参数差异为 +N/−N 计数；非编辑类工具返回 null。
// 与 diffsFor 同一参数口径（toolArgs）：edit_file 一组、multi_edit 逐组累加。
export function diffStatFor(name: string, args: string): DiffStat | null {  if (name !== "edit_file" && name !== "multi_edit") return null;
  const p = parseWriteArgs(name, args);
  // edits 字段缺失/非数组 → 不出芯片；edits 为空数组 → 仍出 0/0 芯片（历史口径）
  if (name === "multi_edit" && p.editsCount === null) return null;
  if (p.hunks.length === 0) return name === "multi_edit" ? { add: 0, del: 0 } : null;
  let add = 0;
  let del = 0;
    for (const h of p.hunks) {
      const pm = plusMinus(h.old, h.new);
      add += pm.add;
      del += pm.del;
    }
  return { add, del };
}

// unifiedRows 把一对 before/after 展开成行级 ± 视图（v4.504 diff 完成刀）：
// 此前 ToolCard 只渲染 d.original——write_file 全增 diff 的 original 是空串，
// 展开态是一只空盒（写入内容用户看不到）；edit_file 只显旧文不显新文。
// 现在 del 行来自 original、add 行来自 modified（diffLines LCS 复用）；
// original 空串（write_file 全增 / edit_file 整段新增）特判全 add——diffLines
// 会给空串出一行伪 del。两侧各剥一个尾换行防末尾空行噪声。
export const TOOL_CARD_MAX_DIFF_ROWS = 400;

export function unifiedRows(
  original: string,
  modified: string,
): { rows: DiffRow[]; folded: number } {
  const stripTail = (s: string) => (s.endsWith("\n") ? s.slice(0, -1) : s);
  const base =
    original === ""
      ? stripTail(modified)
          .split("\n")
          .map((l) => ({ type: "add" as const, text: l }))
      : diffLines(stripTail(original), stripTail(modified));
  if (base.length <= TOOL_CARD_MAX_DIFF_ROWS) return { rows: base, folded: 0 };
  return { rows: base.slice(0, TOOL_CARD_MAX_DIFF_ROWS), folded: base.length - TOOL_CARD_MAX_DIFF_ROWS };
}

// 行内主体已展示关键参数的只读工具：展开态不再渲染 args JSON 盒——read_file
// 展开是 {"path":…} 纯噪音，卡片可展开性由输出块承担。grep 的 path 字段随之
// 不在展开态显形（命中列表本就逐行含路径，信息不丢）。
export const READER_JSON_TOOLS: ReadonlySet<string> = new Set([
  "read_file",
  "ls",
  "glob",
  "grep",
  "web_fetch",
  "web_search",
]);

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
    // ── 进度计划工具（v4.114 刀5）：返回裸 JSON（经信封剥离兜底），取回执字段 ──
    case "schedule_apply": {
      try {
        const r = JSON.parse(extractOutputFromEnvelope(output)) as { duration?: number; critical?: string[]; changes?: string[] };
        const parts: string[] = [];
        if (typeof r.duration === "number") parts.push(`总工期 ${r.duration} 天`);
        if (Array.isArray(r.critical) && r.critical.length > 0) parts.push(`关键 ${r.critical.length} 项`);
        if (Array.isArray(r.changes) && r.changes.length > 0) parts.push(`${r.changes.length} 处修改`);
        return parts.join(" · ");
      } catch {}
      return "";
    }
    case "schedule_get": {
      try {
        const r = JSON.parse(extractOutputFromEnvelope(output)) as { duration?: number; leafCount?: number };
        if (typeof r.duration === "number") return `总工期 ${r.duration} 天 · ${r.leafCount ?? "?"} 项工作`;
      } catch {}
      return "";
    }
    case "schedule_analyze": {
      try {
        const r = JSON.parse(extractOutputFromEnvelope(output)) as { duration?: number; critical?: string[]; checks?: string[] };
        const parts: string[] = [];
        if (typeof r.duration === "number") parts.push(`总工期 ${r.duration} 天`);
        if (Array.isArray(r.critical)) parts.push(`关键链 ${r.critical.length} 项`);
        if (Array.isArray(r.checks) && r.checks.length > 0) parts.push(`质检 ${r.checks.length} 条`);
        return parts.join(" · ");
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
  /** 超宽被截断的行数（preview 侧；full 不动）。 */
  lineFolded: number;
}

export const TOOL_OUTPUT_MAX_PREVIEW_LINES = 60;

// stripOutputNoise 工具输出显示前清噪（v4.506）：①ANSI 转义序列（CSI 颜色/
// 光标控制 + OSC 标题串）——引擎不剥，FORCE_COLOR/--color=always 的脚本会把
// 「ESC[31m」原样送进输出块；②CRLF→LF——Windows 侧输出在 whitespace-pre 块
// 里以 \r 显示怪相。只影响显示层，store 里的原始 output 不动。
export function stripOutputNoise(s: string): string {
  return s
    // eslint-disable-next-line no-control-regex -- 剥 ANSI OSC 串（\x1b]…\x07）即控制字符，此为本函数天职
    .replace(/\x1b\][^\x07]*\x07/g, "")
    // eslint-disable-next-line no-control-regex -- 剥 ANSI CSI 序列（颜色/光标）同理
    .replace(/\x1b\[[0-9;?]*[ -/]*[@-~]/g, "")
    .replace(/\r\n?/g, "\n");
}

// 办公对话流（ToolCard）输出预览上限：Codex 式紧凑——默认只露 12 行，
// 折叠计数 + 「展开全部」按钮兜底（通用 60 行对 chat 流仍是大墙）。
// SinProcessCard 维持 60 默认不随动。
export const TOOL_CARD_OUTPUT_PREVIEW_LINES = 12;

// TOOL_OUTPUT_LINE_CAP 预览行宽上限（v4.507）：单条超长行（base64、绝对
// 路径串、[Image: source: …] 占位）此前原样上屏，几行就能占满 12 行预览。
// 超宽行在 preview 侧截断加 …，full 保持原文（展开全部可见）。
export const TOOL_OUTPUT_LINE_CAP = 160;

export function boundedOutput(
  output: string | undefined,
  maxPreviewLines: number = TOOL_OUTPUT_MAX_PREVIEW_LINES,
): BoundedOutput {
  const full = stripOutputNoise(output ?? "");
  if (full === "") {
    return { preview: "", full, collapsed: false, hiddenLines: 0, totalLines: 0, lineFolded: 0 };
  }
  const lines = full.split("\n");
  const totalLines = lines.length;
  const clip = (l: string) => (l.length > TOOL_OUTPUT_LINE_CAP ? `${l.slice(0, TOOL_OUTPUT_LINE_CAP)}…` : l);
  const clipped = lines.map(clip);
  const lineFolded = clipped.filter((l, i) => l !== lines[i]).length;
  if (totalLines <= maxPreviewLines) {
    return {
      preview: lineFolded > 0 ? clipped.join("\n") : full,
      full,
      collapsed: false,
      hiddenLines: 0,
      totalLines,
      lineFolded,
    };
  }
  const head = clipped.slice(0, maxPreviewLines).join("\n");
  const hiddenLines = totalLines - maxPreviewLines;
  return {
    preview: `${head}\n… 已折叠 ${hiddenLines} 行`,
    full,
    collapsed: true,
    hiddenLines,
    totalLines,
    lineFolded,
  };
}
