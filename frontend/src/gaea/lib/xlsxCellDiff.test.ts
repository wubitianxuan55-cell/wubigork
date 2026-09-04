// xlsxCellDiff.test.ts — xlsx 结构化对比纯函数测试（A2 收 unsupported 欠账）。
import { describe, expect, it } from "vitest";
import { diffXlsxSheets, MAX_XLSX_SHEET_CELLS, type XlsxSheetDiff } from "./xlsxCellDiff";
import type { XlsxSheet } from "./types";

// 便捷构造：cells = [[ref, value, formula?], ...]，每格包成单格行（与
// XlsxSheet.rows 的 XlsxCell[][] 结构对齐）。
function sheet(name: string, cells: Array<[string, string, string?]>): XlsxSheet {
  return {
    name,
    rows: cells.map(([ref, value, formula]) => [{ ref, value, formula }]),
  } as unknown as XlsxSheet;
}

describe("diffXlsxSheets sheet 级对齐", () => {
  it("完全一致：空结果（无差异 sheet 不进结果）", () => {
    const s = [sheet("Sheet1", [["A1", "100"]])];
    expect(diffXlsxSheets(s, s)).toEqual([]);
  });

  it("新增 sheet（仅当前侧）→ state=add，不逐格罗列", () => {
    const out = diffXlsxSheets([], [sheet("新表", [["A1", "1"]])]);
    expect(out).toHaveLength(1);
    expect(out[0].state).toBe("add");
    expect(out[0].cells).toEqual([]);
    expect(out[0].name).toBe("新表");
  });

  it("删除 sheet（仅基线侧）→ state=del，保持在输出中", () => {
    const out = diffXlsxSheets([sheet("旧表", [["A1", "1"]])], []);
    expect(out).toHaveLength(1);
    expect(out[0].state).toBe("del");
  });

  it("只含有差异的 sheet：同名无差异的不进结果", () => {
    const base = [sheet("不变", [["A1", "1"]]), sheet("变", [["A1", "1"]])];
    const cur = [sheet("不变", [["A1", "1"]]), sheet("变", [["A1", "2"]])];
    const out = diffXlsxSheets(base, cur);
    expect(out).toHaveLength(1);
    expect(out[0].name).toBe("变");
  });
});

describe("diffXlsxSheets 单元格级对齐", () => {
  it("新增/删除/修改单元格：change 含旧值新值，计数全量（截断前）", () => {
    const base = sheet("S", [
      ["A1", "100"],
      ["B1", "旧文"],
      ["C1", "将删"],
    ]);
    const cur = sheet("S", [
      ["A1", "200"], // change
      ["B1", "新文"], // change
      ["D1", "新增"], // add
    ]);
    const out = diffXlsxSheets([base], [cur]);
    expect(out).toHaveLength(1);
    const s = out[0];
    expect(s.state).toBe("changed");
    expect(s.add).toBe(1);
    expect(s.del).toBe(1);
    expect(s.change).toBe(2);
    expect(s.total).toBe(4);
    expect(s.cells.map((c) => [c.kind, c.ref, c.old, c.new])).toEqual([
      ["change", "A1", "100", "200"],
      ["change", "B1", "旧文", "新文"],
      ["del", "C1", "将删", ""],
      ["add", "D1", "", "新增"],
    ]);
  });

  it("公式串当文本如实比较（不做求值语义）：公式变更即 change，fx 透出", () => {
    const base = sheet("S", [["A1", "3", "SUM(B1:B2)"]]);
    const cur = sheet("S", [["A1", "3", "SUM(B1:B3)"]]);
    const out = diffXlsxSheets([base], [cur]);
    expect(out[0].change).toBe(1);
    expect(out[0].cells[0].formula).toBe("SUM(B1:B3)");
  });

  it("输出按行号→列号自然阅读序（B2 先于 A10）", () => {
    const base = sheet("S", []);
    const cur = sheet("S", [
      ["A10", "x"],
      ["B2", "y"],
      ["A1", "z"],
    ]);
    const out = diffXlsxSheets([base], [cur]);
    expect(out[0].cells.map((c) => c.ref)).toEqual(["A1", "B2", "A10"]);
  });

  it("每 sheet 变更列表超 maxPerSheet 截断：cells 取前 N 条，计数不失真", () => {
    const cells: Array<[string, string]> = [];
    for (let i = 1; i <= MAX_XLSX_SHEET_CELLS + 5; i++) cells.push([`A${i}`, `v${i}`]);
    const out = diffXlsxSheets([sheet("S", [])], [sheet("S", cells)], 10);
    const s: XlsxSheetDiff = out[0];
    expect(s.truncated).toBe(true);
    expect(s.cells).toHaveLength(10);
    expect(s.total).toBe(MAX_XLSX_SHEET_CELLS + 5);
    expect(s.add).toBe(MAX_XLSX_SHEET_CELLS + 5);
  });

  it("基线/当前两侧空数组照常产出差异（内容缺失由上层 contentMissing 标注）", () => {
    expect(diffXlsxSheets([], [])).toEqual([]);
    const out = diffXlsxSheets([sheet("S", [["A1", "1"]])], []);
    expect(out[0].state).toBe("del");
  });
});
