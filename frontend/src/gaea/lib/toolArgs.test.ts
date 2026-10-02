/**
 * toolArgs.test.ts — FE4-02「工具参数 → before/after」等价用例矩阵（四工具 ×
 * 字段缺失 / 多 hunk / 空内容 / 降级文案）。
 *
 * 背景：同一份 args 被两处各自解析——「变更」tab 的 planDiff.buildChangeDiff
 * 与工具卡内联 diff 的 tools.diffsFor / diffStatFor。两套独立分支 + 各自降级
 * 文案，后端 schema 增删字段要改三处，漏改即「变更 tab 有 diff、工具卡空白」。
 * 本文件的用例只打**公共消费面**（buildChangeDiff / diffsFor / diffStatFor），
 * 因此它在重构前后必须逐条等价：先在此固化现状，再把两套分支收进
 * toolArgs.parseWriteArgs，重构后本文件不改一字仍须全绿。
 *
 * 已锁死的三处分叉（历史行为，不可在重构中「顺手统一」）：
 *   A. CRLF：变更 tab 归一 \r\n→\n 后再做行 diff；工具卡按原文渲染、按原文计数。
 *   B. multi_edit 空 old 片段：变更 tab 丢弃（无原文可还原），工具卡保留（当新增）。
 *   C. multi_edit 片段标注：变更 tab 按**有效片段**连号「编辑 N」，工具卡按
 *      **原始数组下标**标「edit N」；且 edits 字段缺失时变更 tab 与 edits 空数组
 *      同文案，而工具卡 diffstat 是 null（无芯片）而不是 0/0。
 */
import { describe, expect, it } from "vitest";
import { buildChangeDiff, type ChangeDiff } from "./planDiff";
import { parseWriteArgs } from "./toolArgs";
import { diffStatFor, diffsFor, type DiffStat } from "./tools";

interface CardDiff {
  original: string;
  modified: string;
  label?: string;
}

interface MatrixCase {
  id: string;
  tool: string;
  args: string;
  /** 「变更」tab 口径（planDiff） */
  diff: {
    kind: ChangeDiff["kind"];
    /** 降级说明原文（逐字锁死：文案即行为）。 */
    note?: string;
    content?: string;
    hunks?: { label?: string; rows: string[] }[];
  };
  /** 工具卡内联 diff 口径（tools.diffsFor） */
  card: CardDiff[];
  /** 工具卡 diffstat 芯片（tools.diffStatFor）：null = 不出芯片 */
  stat: DiffStat | null;
  /** 期望的 lang（diffsFor 从 path / file_path 推；缺省不校验） */
  lang?: string;
}

const CASES: MatrixCase[] = [
  // ── edit_file ───────────────────────────────────────────────────────
  {
    id: "edit_file 单片段：变更 tab 行级红绿，工具卡同串原文",
    tool: "edit_file",
    args: JSON.stringify({ path: "docs/a.md", old_string: "旧标题\n共有的行", new_string: "新标题\n共有的行" }),
    diff: { kind: "diff", hunks: [{ rows: ["del:旧标题", "add:新标题", "ctx:共有的行"] }] },
    card: [{ original: "旧标题\n共有的行", modified: "新标题\n共有的行" }],
    stat: { add: 1, del: 1 },
    lang: "markdown",
  },
  {
    id: "edit_file CRLF：变更 tab 归一后无差异，工具卡按原文计数（分叉 A）",
    tool: "edit_file",
    args: JSON.stringify({ path: "a.md", old_string: "a\r\nb", new_string: "a\nb" }),
    diff: { kind: "diff", hunks: [{ rows: ["ctx:a", "ctx:b"] }] },
    card: [{ original: "a\r\nb", modified: "a\nb" }],
    stat: { add: 1, del: 1 },
  },
  {
    id: "edit_file old 空串：变更 tab 降级内容预览，工具卡按整段新增",
    tool: "edit_file",
    args: JSON.stringify({ path: "a.md", old_string: "", new_string: "新内容" }),
    diff: { kind: "content", content: "新内容", note: "写入内容预览（原文未记录）" },
    card: [{ original: "", modified: "新内容" }],
    // 现状口径（非笔误）：plusMinus("", x) 走 diffLines，"" 会 split 出一个空行，
    // 于是纯新增也计 del=1。两侧一致，重构不得顺手改成 0（余量已登记）。
    stat: { add: 1, del: 1 },
  },
  {
    id: "edit_file 缺 new_string：两侧无 diff，变更 tab 显式说明缺失",
    tool: "edit_file",
    args: JSON.stringify({ path: "a.md", old_string: "旧" }),
    diff: { kind: "none", note: "参数缺少 old_string/new_string，无法构造 diff" },
    card: [],
    stat: null,
  },
  {
    id: "edit_file 缺 old_string：同文案（缺失即无原文）",
    tool: "edit_file",
    args: JSON.stringify({ path: "a.md", new_string: "新" }),
    diff: { kind: "none", note: "参数缺少 old_string/new_string，无法构造 diff" },
    card: [],
    stat: null,
  },
  {
    id: "edit_file 参数不是 JSON：变更 tab 说明未记录，工具卡不出 diff",
    tool: "edit_file",
    args: "不是JSON",
    diff: { kind: "none", note: "调用参数未记录，无法还原内容变化" },
    card: [],
    stat: null,
  },
  {
    id: "edit_file lang 来源：path 缺省回落 file_path",
    tool: "edit_file",
    args: JSON.stringify({ file_path: "x.ts", old_string: "a", new_string: "b" }),
    diff: { kind: "diff", hunks: [{ rows: ["del:a", "add:b"] }] },
    card: [{ original: "a", modified: "b" }],
    stat: { add: 1, del: 1 },
    lang: "typescript",
  },

  // ── multi_edit ──────────────────────────────────────────────────────
  {
    id: "multi_edit 两片段：逐片段 diff，「编辑 N」与「edit N」双口径",
    tool: "multi_edit",
    args: JSON.stringify({
      path: "a.md",
      edits: [
        { old_string: "one", new_string: "ONE" },
        { old_string: "two\nthree", new_string: "two\nTHREE" },
      ],
    }),
    diff: {
      kind: "diff",
      hunks: [
        { label: "编辑 1", rows: ["del:one", "add:ONE"] },
        { label: "编辑 2", rows: ["ctx:two", "del:three", "add:THREE"] },
      ],
    },
    card: [
      { original: "one", modified: "ONE", label: "edit 1" },
      { original: "two\nthree", modified: "two\nTHREE", label: "edit 2" },
    ],
    stat: { add: 2, del: 2 },
  },
  {
    id: "multi_edit 单片段：变更 tab 不带「编辑 1」（仅多片段才标），工具卡恒带",
    tool: "multi_edit",
    args: JSON.stringify({ path: "a.md", edits: [{ old_string: "a", new_string: "b" }] }),
    diff: { kind: "diff", hunks: [{ rows: ["del:a", "add:b"] }] },
    card: [{ original: "a", modified: "b", label: "edit 1" }],
    stat: { add: 1, del: 1 },
  },
  {
    id: "multi_edit 空 old 片段：变更 tab 丢弃，工具卡保留为新增（分叉 B）",
    tool: "multi_edit",
    args: JSON.stringify({ path: "a.md", edits: [{ old_string: "", new_string: "x" }] }),
    diff: { kind: "none", note: "edits 中没有可还原的 old_string/new_string 片段" },
    card: [{ original: "", modified: "x", label: "edit 1" }],
    // 同上：空 old 片段的 diffstat 计 del=1（diffLines("", "x") 的空行效应）
    stat: { add: 1, del: 1 },
  },
  {
    id: "multi_edit 含非法片段：变更 tab 连号 vs 工具卡原始下标（分叉 C）",
    tool: "multi_edit",
    args: JSON.stringify({
      path: "a.md",
      edits: [
        { old_string: "one", new_string: "ONE" },
        { old_string: 123, new_string: "bad" },
        { old_string: "two", new_string: "TWO" },
      ],
    }),
    diff: {
      kind: "diff",
      hunks: [
        { label: "编辑 1", rows: ["del:one", "add:ONE"] },
        { label: "编辑 2", rows: ["del:two", "add:TWO"] },
      ],
    },
    card: [
      { original: "one", modified: "ONE", label: "edit 1" },
      { original: "two", modified: "TWO", label: "edit 3" },
    ],
    stat: { add: 2, del: 2 },
  },
  {
    id: "multi_edit edits 空数组：变更 tab 无片段文案，工具卡空 diff + 0/0 芯片",
    tool: "multi_edit",
    args: JSON.stringify({ path: "a.md", edits: [] }),
    diff: { kind: "none", note: "edits 中没有可还原的 old_string/new_string 片段" },
    card: [],
    stat: { add: 0, del: 0 },
  },
  {
    id: "multi_edit 无 edits 字段：变更 tab 同文案，工具卡 diffstat 为 null（分叉 C）",
    tool: "multi_edit",
    args: JSON.stringify({ path: "a.md" }),
    diff: { kind: "none", note: "edits 中没有可还原的 old_string/new_string 片段" },
    card: [],
    stat: null,
  },
  {
    id: "multi_edit 参数不是 JSON：变更 tab 说明未记录，工具卡空",
    tool: "multi_edit",
    args: "{坏",
    diff: { kind: "none", note: "调用参数未记录，无法还原内容变化" },
    card: [],
    stat: null,
  },

  // ── edit_lines ──────────────────────────────────────────────────────
  {
    id: "edit_lines 带行号：降级内容预览（含行范围），工具卡给行号占位原文",
    tool: "edit_lines",
    args: JSON.stringify({ path: "a.md", start_line: 3, end_line: 5, new_content: "新行A\n新行B" }),
    diff: {
      kind: "content",
      content: "新行A\n新行B",
      note: "按行号替换（第 3–5 行）：原行内容未随事件记录，以下为新写入内容",
    },
    card: [{ original: "[lines 3-5]", modified: "新行A\n新行B" }],
    stat: null,
  },
  {
    id: "edit_lines 缺行号：范围回落「指定行范围」，工具卡占位显示 undefined",
    tool: "edit_lines",
    args: JSON.stringify({ path: "a.md", new_content: "新行" }),
    diff: {
      kind: "content",
      content: "新行",
      note: "按行号替换（指定行范围）：原行内容未随事件记录，以下为新写入内容",
    },
    card: [{ original: "[lines undefined-undefined]", modified: "新行" }],
    stat: null,
  },
  {
    id: "edit_lines 空内容：两侧都认「有调用但内容为空」，不当作缺参",
    tool: "edit_lines",
    args: JSON.stringify({ path: "a.md", start_line: 1, end_line: 1, new_content: "" }),
    diff: {
      kind: "content",
      content: "",
      note: "按行号替换（第 1–1 行）：原行内容未随事件记录，以下为新写入内容",
    },
    card: [{ original: "[lines 1-1]", modified: "" }],
    stat: null,
  },
  {
    id: "edit_lines 缺 new_content：变更 tab 说明缺失，工具卡不出 diff",
    tool: "edit_lines",
    args: JSON.stringify({ path: "a.md", start_line: 1, end_line: 2 }),
    diff: { kind: "none", note: "参数缺少 new_content，无法展示内容" },
    card: [],
    stat: null,
  },

  // ── write_file ──────────────────────────────────────────────────────
  {
    id: "write_file 正常：降级覆盖写入预览，工具卡全增 diff，无 diffstat 芯片",
    tool: "write_file",
    args: JSON.stringify({ path: "new.md", content: "全文内容" }),
    diff: { kind: "content", content: "全文内容", note: "覆盖写入：写入前内容未记录，以下为写入内容" },
    card: [{ original: "", modified: "全文内容" }],
    stat: null,
  },
  {
    id: "write_file 空内容：仍算「有内容」（空串不是缺参）",
    tool: "write_file",
    args: JSON.stringify({ path: "new.md", content: "" }),
    diff: { kind: "content", content: "", note: "覆盖写入：写入前内容未记录，以下为写入内容" },
    card: [{ original: "", modified: "" }],
    stat: null,
  },
  {
    id: "write_file 缺 content：变更 tab 说明缺失，工具卡不出 diff",
    tool: "write_file",
    args: JSON.stringify({ path: "new.md" }),
    diff: { kind: "none", note: "参数缺少 content，无法展示内容" },
    card: [],
    stat: null,
  },

  // ── 非内容级写类工具 ────────────────────────────────────────────────
  {
    id: "move_file：无内容级 diff（变更 tab 说明，工具卡不出）",
    tool: "move_file",
    args: JSON.stringify({ source: "a.md", destination: "docs/a.md" }),
    diff: { kind: "none", note: "移动/重命名操作，无内容变化记录" },
    card: [],
    stat: null,
  },
  {
    id: "未收录写类工具（delete_range）：诚实说明无法构造行级 diff",
    tool: "delete_range",
    args: JSON.stringify({ path: "a.md", start_line: 1 }),
    diff: { kind: "none", note: "该工具不携带 old/new 片段，无法构造行级 diff" },
    card: [],
    stat: null,
  },
  {
    // 该降级文案原在 planDiff.ts，FE4-02 随解析器搬到 toolArgs（单源）——逐字锁死
    id: "schedule_apply：整计划/ops 无行级 diff 的说明随解析器单源",
    tool: "schedule_apply",
    args: JSON.stringify({ ops: [{ op: "add_task" }] }),
    diff: { kind: "none", note: "整计划替换/ops 调整：无行级 diff，结果以工具回执与进度计划板块为准" },
    card: [],
    stat: null,
  },
];

describe("写类工具参数解析等价矩阵（FE4-02：变更 tab ↔ 工具卡）", () => {
  for (const c of CASES) {
    it(c.id, () => {
      const d = buildChangeDiff(c.tool, c.args);
      expect(d.kind).toBe(c.diff.kind);
      if (c.diff.note !== undefined) expect(d.note).toBe(c.diff.note);
      if (c.diff.content !== undefined) expect(d.content).toBe(c.diff.content);
      if (c.diff.hunks !== undefined) {
        expect(
          d.hunks.map((h) => ({ label: h.label, rows: h.rows.map((r) => `${r.type}:${r.text}`) })),
        ).toEqual(c.diff.hunks);
      }

      const cards = diffsFor(c.tool, c.args);
      expect(cards.map((x) => ({ original: x.original, modified: x.modified, label: x.label }))).toEqual(c.card);
      if (c.lang !== undefined) expect(cards[0]?.lang).toBe(c.lang);

      expect(diffStatFor(c.tool, c.args)).toEqual(c.stat);
    });
  }
});

describe("加固：args 为合法 JSON 但非对象（FE4-02 唯一非等价变更）", () => {
  // 旧两条实现都在 `parsed.old_string` / `str(a,"path")` 处 TypeError 炸穿
  // 变更面板；现按空对象走各工具的字段缺失降级。
  it("null 不再抛：变更 tab 给出字段缺失降级", () => {
    expect(() => buildChangeDiff("edit_file", "null")).not.toThrow();
    expect(buildChangeDiff("edit_file", "null")).toEqual({
      kind: "none",
      hunks: [],
      note: "参数缺少 old_string/new_string，无法构造 diff",
    });
    expect(buildChangeDiff("multi_edit", "null").note).toBe("edits 中没有可还原的 old_string/new_string 片段");
  });

  it("null 在工具卡侧等价于空参数（无 diff / 无芯片）", () => {
    expect(diffsFor("edit_file", "null")).toEqual([]);
    expect(diffsFor("edit_lines", "null")).toEqual([]);
    expect(diffStatFor("edit_file", "null")).toBeNull();
    expect(diffStatFor("multi_edit", "null")).toBeNull();
  });

  it("数字/字符串/数组与旧实现等价（本就是空参数语义）", () => {
    expect(diffsFor("edit_file", "3")).toEqual([]);
    expect(diffsFor("edit_file", "\"x\"")).toEqual([]);
    expect(diffsFor("edit_file", "[]")).toEqual([]);
    expect(buildChangeDiff("write_file", "[]").note).toBe("参数缺少 content，无法展示内容");
  });
});

describe("parseWriteArgs 单一解析器契约（两个消费端的共同上游）", () => {
  it("multi_edit：hunks 保留原序与原始下标，含空 old 片段；editsCount 计全部条目", () => {
    const p = parseWriteArgs(
      "multi_edit",
      JSON.stringify({
        path: "a.md",
        edits: [
          { old_string: "one", new_string: "ONE" },
          "not-an-object",
          { old_string: "", new_string: "x" },
        ],
      }),
    );
    expect(p.path).toBe("a.md");
    expect(p.hunks).toEqual([
      { index: 0, old: "one", new: "ONE" },
      { index: 2, old: "", new: "x" },
    ]);
    expect(p.editsCount).toBe(3);
    expect(p.degrade).toBeUndefined();
  });

  it("editsCount 区分「无 edits 字段」(null) 与「空数组」(0)", () => {
    expect(parseWriteArgs("multi_edit", "{}").editsCount).toBeNull();
    const empty = parseWriteArgs("multi_edit", JSON.stringify({ edits: [] }));
    expect(empty.editsCount).toBe(0);
    expect(empty.degrade).toBe("edits 中没有可还原的 old_string/new_string 片段");
  });

  it("edit_file 空 old：hunks 保留片段 + content 给预览 + degrade 给原因", () => {
    const p = parseWriteArgs("edit_file", JSON.stringify({ path: "a.md", old_string: "", new_string: "新" }));
    expect(p.hunks).toEqual([{ index: 0, old: "", new: "新" }]);
    expect(p.content).toBe("新");
    expect(p.degrade).toBe("写入内容预览（原文未记录）");
  });

  it("path 取唯一主目标：path 优先、回落 file_path，不吞聚合键（source 等）", () => {
    const viaFilePath = JSON.stringify({ file_path: "x.ts", old_string: "a", new_string: "b" });
    expect(parseWriteArgs("edit_file", viaFilePath).path).toBe("x.ts");
    const both = JSON.stringify({ path: "p.md", file_path: "x.ts", old_string: "a", new_string: "b" });
    expect(parseWriteArgs("edit_file", both).path).toBe("p.md");
    expect(parseWriteArgs("move_file", JSON.stringify({ source: "s.md" })).path).toBe("");
  });

  it("edit_lines 行号范围原样透传（格式化留给消费端）", () => {
    const p = parseWriteArgs("edit_lines", JSON.stringify({ path: "a.md", start_line: 3, end_line: 5, new_content: "x" }));
    expect(p.lineRange).toEqual({ start: 3, end: 5 });
    expect(p.hunks).toEqual([]);
    expect(p.content).toBe("x");
    const q = parseWriteArgs("edit_lines", JSON.stringify({ path: "a.md", new_content: "x" }));
    expect(q.lineRange).toEqual({ start: undefined, end: undefined });
  });

  it("语法错误的 args：统一「调用参数未记录」且无片段/无计数", () => {
    expect(parseWriteArgs("edit_file", "{坏")).toEqual({
      path: "",
      hunks: [],
      degrade: "调用参数未记录，无法还原内容变化",
      editsCount: null,
    });
  });
});

