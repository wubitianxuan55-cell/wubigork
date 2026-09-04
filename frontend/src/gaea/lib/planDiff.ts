// planDiff.ts —「变更」tab 行级 diff 的数据构造（纯函数，零绑定依赖）。
//
// v4.25 变更 tab diff 化：诚实评估事件流（Item.args）里写类工具的参数结构——
//   edit_file  { path, old_string, new_string }            → 有真 before/after，可构造行 diff；
//   multi_edit { path, edits: [{ old_string, new_string }] } → 同上，逐片段构造；
//   edit_lines { path, start_line, end_line, new_content }   → 只有新内容，降级「写入内容预览」；
//   write_file { path, content }                             → 同上（覆盖写入，旧内容未随事件记录）；
//   move_file / delete_range / delete_symbol 等              → 无内容片段，不伪造 diff。
// （对齐 internal/gaea/tool/builtin/{editfile,multiedit,editlines}.go 的 Schema。）
// 无 old/new 的调用一律显式标注降级原因，绝不拿新内容冒充红绿 diff。

import { diffLines, type DiffRow } from "./diff";
import { WRITE_TOOL_NAMES, extractChangedPaths } from "./changes";
import type { Item, ToolStatus } from "./store";

// 一段可 diff 的编辑片段（old → new 的行级 LCS 结果）。
export interface DiffHunk {
  /** multi_edit 一次调用带多个片段时标注「编辑 N」；单片段缺省。 */
  label?: string;
  rows: DiffRow[];
}

// 单次写类工具调用的内容级视图：
//   kind="diff"    能构造行级红绿 diff（edit_file / multi_edit 的 old→new）；
//   kind="content" 无 old，但参数带新内容 → 诚实降级为「写入内容预览」；
//   kind="none"    移动/删除类或参数未记录，无内容可展示。
export interface ChangeDiff {
  kind: "diff" | "content" | "none";
  hunks: DiffHunk[];
  /** kind="content" 时的新内容（write_file 的 content / edit_lines 的 new_content）。 */
  content?: string;
  /** 诚实说明：为什么不是真 diff / 为什么无内容。kind="diff" 时缺省。 */
  note?: string;
}

// 把 CRLF 归一为 LF 后按行展示（edit_file 在 CRLF 文件上要求 old 带 \r\n，
// 行 diff 只关心行内容，展示端统一掉 \r 免得行尾出现不可见字符）。
function normText(s: string): string {
  return s.replace(/\r\n/g, "\n");
}

// 从一次写类工具调用的参数 JSON 构造内容级视图。解析失败/参数缺失时返回
// kind="none" 并说明原因——降级是显式的，不静默吞掉。
export function buildChangeDiff(tool: string, argsJson: string): ChangeDiff {
  let parsed: Record<string, unknown>;
  try {
    parsed = JSON.parse(argsJson || "{}") as Record<string, unknown>;
  } catch {
    return { kind: "none", hunks: [], note: "调用参数未记录，无法还原内容变化" };
  }

  if (tool === "edit_file") {
    const oldS = typeof parsed.old_string === "string" ? parsed.old_string : null;
    const newS = typeof parsed.new_string === "string" ? parsed.new_string : null;
    if (oldS === null || newS === null) {
      return { kind: "none", hunks: [], note: "参数缺少 old_string/new_string，无法构造 diff" };
    }
    if (oldS === "") {
      // edit_file 后端要求 old 非空；防御性兜底：空 old 视为纯写入。
      return { kind: "content", hunks: [], content: newS, note: "写入内容预览（原文未记录）" };
    }
    return { kind: "diff", hunks: [{ rows: diffLines(normText(oldS), normText(newS)) }] };
  }

  if (tool === "multi_edit") {
    const edits = Array.isArray(parsed.edits) ? parsed.edits : [];
    const hunks: DiffHunk[] = [];
    for (const e of edits) {
      if (!e || typeof e !== "object") continue;
      const rec = e as Record<string, unknown>;
      if (typeof rec.old_string !== "string" || typeof rec.new_string !== "string") continue;
      if (rec.old_string === "") continue;
      hunks.push({
        label: edits.length > 1 ? `编辑 ${hunks.length + 1}` : undefined,
        rows: diffLines(normText(rec.old_string), normText(rec.new_string)),
      });
    }
    if (hunks.length === 0) {
      return { kind: "none", hunks: [], note: "edits 中没有可还原的 old_string/new_string 片段" };
    }
    return { kind: "diff", hunks };
  }

  if (tool === "edit_lines") {
    const c = typeof parsed.new_content === "string" ? parsed.new_content : null;
    if (c === null) return { kind: "none", hunks: [], note: "参数缺少 new_content，无法展示内容" };
    const start = typeof parsed.start_line === "number" ? parsed.start_line : 0;
    const end = typeof parsed.end_line === "number" ? parsed.end_line : 0;
    const range = start > 0 && end >= start ? `第 ${start}–${end} 行` : "指定行范围";
    return {
      kind: "content",
      hunks: [],
      content: c,
      note: `按行号替换（${range}）：原行内容未随事件记录，以下为新写入内容`,
    };
  }

  if (tool === "write_file") {
    const c = typeof parsed.content === "string" ? parsed.content : null;
    if (c === null) return { kind: "none", hunks: [], note: "参数缺少 content，无法展示内容" };
    return { kind: "content", hunks: [], content: c, note: "覆盖写入：写入前内容未记录，以下为写入内容" };
  }

  if (tool === "move_file") {
    return { kind: "none", hunks: [], note: "移动/重命名操作，无内容变化记录" };
  }

  return { kind: "none", hunks: [], note: "该工具不携带 old/new 片段，无法构造行级 diff" };
}

// 「变更」面板单文件下的单次调用记录（按会话顺序累积）。
export interface ChangeCall {
  itemId: string;
  tool: string;
  status: ToolStatus;
  /** 该次调用是否成功落盘（失败/中止的调用不构成实际变更，UI 显式标注）。 */
  applied: boolean;
  diff: ChangeDiff;
}

// ── 路径匹配（变更行 ↔ 证据链 Journal 记录 target）────────────────────
// Journal 的 target 与事件流 args 里的 path 可能一边绝对一边相对（后端
// resolveTarget 与工具参数各自保留原样），统一斜杠后先比全等，再退到
// 「带边界的前缀包含」：较长路径以 / + 较短路径结尾，且较短路径至少含
// 两段（避免单文件名 b.md 误配到任意目录下的同名文件）。
export function normalizeWsPath(p: string): string {
  return p.replace(/\\/g, "/").replace(/\/+$/, "").trim();
}

export function pathsMatch(a: string, b: string): boolean {
  const x = normalizeWsPath(a);
  const y = normalizeWsPath(b);
  if (!x || !y) return false;
  if (x === y) return true;
  const [short, long] = x.length <= y.length ? [x, y] : [y, x];
  if (short.split("/").length < 2) return false;
  return long.toLowerCase().endsWith(`/${short.toLowerCase()}`);
}

// 从会话 items 聚合「路径 → 该路径上的写类调用列表」（按出现顺序；展示端再倒序）。
// 与 changes.ts 的 buildSessionChanges 同源同口径：同一个 WRITE_TOOL_NAMES 集合、
// 同一个 extractChangedPaths 提取，保证次数与明细一一对应。
export function buildChangeCalls(
  items: Item[],
  tools: ReadonlySet<string> = WRITE_TOOL_NAMES,
): Map<string, ChangeCall[]> {
  const map = new Map<string, ChangeCall[]>();
  items.forEach((it) => {
    if (it.kind !== "tool" || !tools.has(it.name)) return;
    const paths = extractChangedPaths(it.args || "");
    if (paths.length === 0) return;
    const diff = buildChangeDiff(it.name, it.args || "");
    const call: ChangeCall = {
      itemId: it.id,
      tool: it.name,
      status: it.status,
      applied: it.status === "done",
      diff,
    };
    for (const p of paths) {
      const list = map.get(p);
      if (list) list.push(call);
      else map.set(p, [call]);
    }
  });
  return map;
}

// ── Git unified diff 解析（2b Git 面板）──────────────────────────
// 把 `git diff --no-color -- <path>` 的标准 unified diff 文本解析为
// ChangeDiff（复用 ChangesDiff 渲染：行级红绿 + 上下文行）。解析不了的
// 输入诚实降级 kind="none"，不伪造行。

/**
 * 从 unified diff 文本构造内容级视图。
 * 口径：hunk 头 `@@ -a,b +c,d @@` 起新块；`+`/`-`/` `（空格）分别为增/删/
 * 上下文；`\ No newline at end of file` 行跳过；`diff --git`/`index`/
 * `---`/`+++` 头部行作为 hunk label 来源（取文件对）。
 */
export function buildGitDiff(diffText: string): ChangeDiff {
  const text = (diffText || "").replace(/\r\n/g, "\n");
  if (!text.trim()) return { kind: "none", hunks: [], note: "无差异" };
  const lines = text.split("\n");
  const hunks: DiffHunk[] = [];
  let cur: DiffHunk | null = null;
  let fileLabel = "";
  for (const line of lines) {
    if (line.startsWith("diff --git")) {
      fileLabel = line.replace(/^diff --git a\//, "").replace(/ b\/.*$/, "");
      continue;
    }
    if (line.startsWith("index ") || line.startsWith("old mode") || line.startsWith("new mode")) continue;
    if (line.startsWith("--- ") || line.startsWith("+++ ")) continue;
    if (line.startsWith("@@")) {
      cur = { label: fileLabel ? `${fileLabel} · ${line}` : line, rows: [] };
      hunks.push(cur);
      continue;
    }
    if (line.startsWith("\\ No newline")) continue; // 尾行无换行标记：展示层跳过
    if (!cur) continue; // hunk 之前的杂项（如 mode 行）不入行
    if (line.startsWith("+")) {
      cur.rows.push({ type: "add", text: line.slice(1) });
    } else if (line.startsWith("-")) {
      cur.rows.push({ type: "del", text: line.slice(1) });
    } else if (line.startsWith(" ")) {
      cur.rows.push({ type: "ctx", text: line.slice(1) });
    } else if (line === "") {
      cur.rows.push({ type: "ctx", text: "" });
    }
    // 其他未知前缀行：跳过（诚实丢弃，不猜语义）
  }
  if (hunks.length === 0) return { kind: "none", hunks: [], note: "未能解析出 diff 内容" };
  return { kind: "diff", hunks };
}
