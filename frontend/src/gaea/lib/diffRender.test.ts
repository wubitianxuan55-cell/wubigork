// diffRender 单测（2c：改蓝配对/字符高亮/上下文折叠）。
import { describe, expect, it } from "vitest";
import type { DiffRow } from "./diff";
import { charSegments, foldContext, pairModifications, type DiffPresentRow } from "./diffRender";

const row = (type: DiffRow["type"], text: string): DiffRow => ({ type, text });

describe("pairModifications 改蓝配对", () => {
  it("相邻删块+增块按序两两配对", () => {
    const rows = [row("ctx", "a"), row("del", "旧1"), row("del", "旧2"), row("add", "新1"), row("add", "新2"), row("ctx", "b")];
    const present = pairModifications(rows);
    const paired = present.filter((p) => p.kind === "row" && p.pairOld !== undefined);
    expect(paired).toHaveLength(4);
    // 第一对：旧1 ↔ 新1
    expect(paired[0]).toMatchObject({ kind: "row", pairOld: "旧1", pairNew: "新1" });
    expect((paired[0] as { row: DiffRow }).row.text).toBe("旧1");
    expect(paired[1]).toMatchObject({ kind: "row", pairOld: "旧1", pairNew: "新1" });
    expect((paired[1] as { row: DiffRow }).row.text).toBe("新1");
  });

  it("配不上的余量保持原类型（2删1增 → 1对+1余删）", () => {
    const rows = [row("del", "a"), row("del", "b"), row("add", "x")];
    const present = pairModifications(rows);
    const paired = present.filter((p) => p.kind === "row" && p.pairOld !== undefined);
    const plain = present.filter((p) => p.kind === "row" && p.pairOld === undefined);
    expect(paired).toHaveLength(2);
    expect(plain).toHaveLength(1);
    expect((plain[0] as { row: DiffRow }).row).toEqual(row("del", "b"));
  });

  it("纯增块/纯删块不配对", () => {
    const present = pairModifications([row("add", "x"), row("add", "y")]);
    expect(paired_filter(present)).toHaveLength(0);
    const present2 = pairModifications([row("del", "x")]);
    expect(paired_filter(present2)).toHaveLength(0);
  });

  it("交错输入被规范化（del,add,del → 配对两对，余删保留）", () => {
    // run=[del,add,del]：dels=[d1,d2] adds=[a1] → 1 对 + 余删 d2
    const rows = [row("del", "d1"), row("add", "a1"), row("del", "d2")];
    const present = pairModifications(rows);
    const paired = paired_filter(present);
    expect(paired).toHaveLength(2);
    expect((paired[0] as { pairOld?: string }).pairOld).toBe("d1");
    expect((paired[1] as { pairNew?: string }).pairNew).toBe("a1");
    const leftover = present.filter((p) => p.kind === "row" && p.pairOld === undefined);
    expect((leftover[0] as { row: DiffRow }).row.text).toBe("d2");
  });

  function paired_filter(present: ReturnType<typeof pairModifications>) {
    return present.filter((p) => p.kind === "row" && p.pairOld !== undefined);
  }
});

describe("charSegments 行内字符高亮", () => {
  it("标出行内变化的片段（前缀未变/核心变化）", () => {
    const { oldSegs, newSegs } = charSegments("const x = 1;", "const x = 2;");
    // 前缀 "const x = " 未变（同段），old 只变化 "1;"、new 只变化 "2;"
    expect(oldSegs[0]).toEqual({ text: "const x = ", changed: false });
    const changedOld = oldSegs.filter((s) => s.changed).map((s) => s.text).join("");
    const changedNew = newSegs.filter((s) => s.changed).map((s) => s.text).join("");
    expect(changedOld).toBe("1");
    expect(changedNew).toBe("2");
  });

  it("完全相同 → 单段未变；超长行整行视为变化（防 LCS 撑爆）", () => {
    const same = charSegments("abc", "abc");
    expect(same.oldSegs).toEqual([{ text: "abc", changed: false }]);
    const long = charSegments("a".repeat(300), "b".repeat(300));
    expect(long.oldSegs).toEqual([{ text: "a".repeat(300), changed: true }]);
  });
});

describe("foldContext 上下文折叠", () => {
  it("连续 ctx 超过 keep*2 时收起中段（首尾各留 keep）", () => {
    const rows: DiffPresentRow[] = [
      { kind: "row", row: row("del", "-") },
      ...Array.from({ length: 10 }, (_, i) => ({ kind: "row" as const, row: row("ctx", `c${i}`) })),
      { kind: "row", row: row("add", "+") },
    ];
    const out = foldContext(rows);
    const folds = out.filter((p) => p.kind === "fold");
    expect(folds).toHaveLength(1);
    expect((folds[0] as { count: number }).count).toBe(4); // 10 - 3*2
    const ctxRows = out.filter(
      (p) => p.kind === "row" && p.row.type === "ctx",
    );
    expect(ctxRows).toHaveLength(6); // 3 + 3
  });

  it("短 ctx run 与配对行不折叠", () => {
    const rows: DiffPresentRow[] = [
      { kind: "row", row: row("ctx", "c1") },
      { kind: "row", row: row("ctx", "c2"), pairOld: "c2", pairNew: "c2" },
      { kind: "fold", count: 5, rows: [] },
    ];
    expect(foldContext(rows)).toEqual(rows); // 原样透传（非 ctx/已折叠不动）
  });
});
