// toolArgs.ts — 写类工具参数解析的唯一实现（FE4-02）。
//
// 背景：同一份 args 此前被两处各自解析——「变更」tab 的 planDiff.buildChangeDiff
// 与工具卡内联 diff 的 tools.diffsFor / diffStatFor，两套独立分支 + 两套降级
// 文案；changes.extractChangedPaths 又独立解析 path/edits。后端工具 schema 增删
// 字段要改三处，漏改即「变更 tab 有 diff、工具卡空白」。本模块把
// `edit_file` / `multi_edit` / `edit_lines` / `write_file` 的字段口径收成一份，
// 两个消费端只做各自的**展示策略**（归一、标注、过滤），不再各自认字段。
//
// 为何不并 changes.extractChangedPaths：它的契约是「一次调用 → 全部被改动路径」
// （path/file_path/notebook_path/source/destination + paths/file_paths 数组 +
// edits[].path/file_path），服务 WRITE_TOOL_NAMES 全量工具（含 delete_symbol 等
// 无内容片段者）的聚合计数；本模块的契约是「一次调用的内容级视图」——只要
// **唯一主目标** path（工具卡据此推 lang）+ 可还原的 old/new 片段。两者键集合、
// 基数、工具集合都不同，硬并会把聚合语义塞进内容视图（见余量登记）。
//
// 本模块**零 import**（纯函数 + 类型），任何消费端转引它都不产生模块边。
//
// 三处**故意保留**的两侧分歧（等价矩阵逐条锁死在 toolArgs.test.ts，勿「顺手统一」）：
//   A. CRLF：变更 tab 归一 \r\n→\n 后再做行 diff（展示端行为，留在 planDiff）；
//      工具卡按原文渲染、按原文计数。本模块只给原文，归一由消费端决定。
//   B. multi_edit 里 old 为空串的片段：变更 tab 丢弃（无原文可还原），工具卡保留
//      （当整段新增）。故本模块的 hunks **保留空 old 片段**，过滤是消费端策略。
//   C. multi_edit 片段标注与空表语义：变更 tab 按「原始条目数 > 1」决定是否标
//      「编辑 N」、N 按**有效片段**连号；工具卡按**原始下标**标「edit N」，且
//      edits 字段缺失 → diffstat 为 null（无芯片），edits 为空数组 → 0/0 芯片。
//      故本模块给出 editsCount（原始条目数，非数组为 null）而不是布尔。
//
// 加固（唯一一处非等价变更，另有对应用例）：args 为合法 JSON 但**不是对象**
// （典型 `null`）时，旧两条实现都会在读属性处 TypeError 炸穿变更面板；本模块
// 一律按空对象处理（各工具于是走自己的字段缺失降级）。仅把崩溃变成降级，不改变
// 任何既有非崩溃行为（数字/字符串/数组在旧实现下同样等价于空对象）。

/** 一段可还原的编辑片段（原序保留；old 可能为空串，展示与否由消费端定）。 */
export interface ToolArgHunk {
  /** 片段在原始 args 中的下标（multi_edit 的 edits 下标；单片段工具恒 0）——
   *  工具卡「edit N」用它，故不能压缩成过滤后的连号。 */
  index: number;
  old: string;
  new: string;
}

/** 一次写类工具调用的内容级解析结果（展示策略由消费端各自决定）。 */
export interface ParsedWriteArgs {
  /** 唯一主目标路径（path 优先、回落 file_path；都缺为空串）——工具卡据此推 lang。 */
  path: string;
  /** 全部 old/new 均为字符串的片段（含 old 空串者；原序）。 */
  hunks: ToolArgHunk[];
  /** 「有调用但无原文」时的整段新内容：write_file.content / edit_lines.new_content /
   *  edit_file 的空 old 整段新增。undefined = 本次调用没有内容可展示。 */
  content?: string;
  /** 诚实降级说明：为什么不是真 diff / 为什么无内容。缺省 = 可构造 diff。 */
  degrade?: string;
  /** multi_edit 的 edits 原始条目数（含非法条目）；缺失/非数组为 null。
   *  null 与 0 的区别是两个消费端的既有语义（见文件头分叉 C），不可合并。 */
  editsCount: number | null;
  /** edit_lines 的行号范围（原始值透传：工具卡渲染成 "[lines s-e]" 占位原文行）。 */
  lineRange?: { start?: unknown; end?: unknown };
}

/** 解析 args JSON。语法错误 → null（「参数未记录」）；合法但非对象（null/数字/
 *  字符串/数组）→ 空对象，交由各工具走字段缺失分支。旧实现里 `null` 会在读
 *  属性处 TypeError 炸穿变更面板，这里一并收成显式降级（见文件头「加固」）。 */
function parseObject(args: string): Record<string, unknown> | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(args || "{}");
  } catch {
    return null;
  }
  if (parsed === null || typeof parsed !== "object") return {};
  return parsed as Record<string, unknown>;
}

function str(a: Record<string, unknown>, key: string): string {
  return typeof a[key] === "string" ? (a[key] as string) : "";
}

const NOTE_NO_ARGS = "调用参数未记录，无法还原内容变化";
const NOTE_EDIT_FILE_MISSING = "参数缺少 old_string/new_string，无法构造 diff";
const NOTE_EDIT_FILE_EMPTY_OLD = "写入内容预览（原文未记录）";
const NOTE_MULTI_EDIT_EMPTY = "edits 中没有可还原的 old_string/new_string 片段";
const NOTE_EDIT_LINES_MISSING = "参数缺少 new_content，无法展示内容";
const NOTE_WRITE_FILE = "覆盖写入：写入前内容未记录，以下为写入内容";
const NOTE_WRITE_FILE_MISSING = "参数缺少 content，无法展示内容";
const NOTE_MOVE_FILE = "移动/重命名操作，无内容变化记录";
const NOTE_SCHEDULE_APPLY = "整计划替换/ops 调整：无行级 diff，结果以工具回执与进度计划板块为准";
const NOTE_UNKNOWN = "该工具不携带 old/new 片段，无法构造行级 diff";

function bare(path: string, degrade: string): ParsedWriteArgs {
  return { path, hunks: [], degrade, editsCount: null };
}

/**
 * 解析一次写类工具调用的参数，产出「内容级视图」的原料。
 * 只做字段识别与降级归因，不做任何展示决策（归一/标注/过滤在消费端）。
 */
export function parseWriteArgs(tool: string, args: string): ParsedWriteArgs {
  const a = parseObject(args);
  if (a === null) return bare("", NOTE_NO_ARGS);
  // path 优先、回落 file_path：与工具卡推 lang 的既有口径一致（不多取 notebook_path
  // 等键——那是 extractChangedPaths 的聚合口径，不是这里的「唯一主目标」）。
  const path = str(a, "path") || str(a, "file_path");

  if (tool === "edit_file") {
    const oldS = typeof a.old_string === "string" ? a.old_string : null;
    const newS = typeof a.new_string === "string" ? a.new_string : null;
    if (oldS === null || newS === null) return bare(path, NOTE_EDIT_FILE_MISSING);
    const hunk: ToolArgHunk = { index: 0, old: oldS, new: newS };
    if (oldS === "") {
      // 后端 edit_file 要求 old 非空：空 old 防御性视为纯写入。变更 tab 降级为
      // 内容预览；工具卡仍按 old="" 的整段新增渲染（分叉 B 的单片段形态）。
      return { path, hunks: [hunk], content: newS, degrade: NOTE_EDIT_FILE_EMPTY_OLD, editsCount: null };
    }
    return { path, hunks: [hunk], editsCount: null };
  }

  if (tool === "multi_edit") {
    const edits = Array.isArray(a.edits) ? (a.edits as unknown[]) : null;
    const hunks: ToolArgHunk[] = [];
    if (edits) {
      edits.forEach((e, i) => {
        if (!e || typeof e !== "object") return;
        const rec = e as Record<string, unknown>;
        if (typeof rec.old_string !== "string" || typeof rec.new_string !== "string") return;
        hunks.push({ index: i, old: rec.old_string, new: rec.new_string });
      });
    }
    const editsCount = edits ? edits.length : null;
    // 无可还原片段（含「有片段但 old 全为空」与「edits 缺失/空」）时才给降级说明
    if (hunks.every((h) => h.old === "")) {
      return { path, hunks, degrade: NOTE_MULTI_EDIT_EMPTY, editsCount };
    }
    return { path, hunks, editsCount };
  }

  if (tool === "edit_lines") {
    const start = a.start_line;
    const end = a.end_line;
    const lineRange = { start, end };
    const c = typeof a.new_content === "string" ? a.new_content : null;
    if (c === null) return { path, hunks: [], degrade: NOTE_EDIT_LINES_MISSING, editsCount: null, lineRange };
    const ranged = typeof start === "number" && start > 0 && typeof end === "number" && end >= start;
    const range = ranged ? `第 ${String(start)}–${String(end)} 行` : "指定行范围";
    return {
      path,
      hunks: [],
      content: c,
      degrade: `按行号替换（${range}）：原行内容未随事件记录，以下为新写入内容`,
      editsCount: null,
      lineRange,
    };
  }

  if (tool === "write_file") {
    const c = typeof a.content === "string" ? a.content : null;
    if (c === null) return bare(path, NOTE_WRITE_FILE_MISSING);
    return { path, hunks: [], content: c, degrade: NOTE_WRITE_FILE, editsCount: null };
  }

  if (tool === "move_file") return bare(path, NOTE_MOVE_FILE);
  // schedule_apply（v4.146 刀A）：整计划 JSON / ops 序列，无行级片段可还原——
  // 显式降级说明（变更 tab 可见、可回滚），不伪造红绿 diff。工具卡无对应分支。
  if (tool === "schedule_apply") return bare(path, NOTE_SCHEDULE_APPLY);
  return bare(path, NOTE_UNKNOWN);
}
