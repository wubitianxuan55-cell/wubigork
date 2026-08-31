// verifyDiff.test.ts — 证据链「声明↔实况」前端近似比对的纯函数单测。
import { describe, expect, it } from "vitest";
import {
  buildCellIndex,
  buildVerifyDiff,
  cellKey,
  compareCell,
  describeOp,
  isClaimableOp,
  normFormula,
  opBatchCount,
  opImpact,
  parseOps,
} from "./verifyDiff";

describe("parseOps 声明解析", () => {
  it("解析 JSON 数组 → 操作视图", () => {
    const ops = parseOps(JSON.stringify([
      { type: "set_value", sheet: "预算", target: "B2", value: 120.5 },
      { type: "set_formula", sheet: "预算", target: "B4", formula: "SUM(B2:B3)" },
    ]));
    expect(ops).toHaveLength(2);
    expect(ops[0]).toMatchObject({ type: "set_value", target: "B2", value: 120.5 });
    expect(ops[1]).toMatchObject({ type: "set_formula", formula: "SUM(B2:B3)" });
  });

  it("解析 {ops:[...]} 包裹形式", () => {
    const ops = parseOps(JSON.stringify({ ops: [{ type: "clean", sheet: "预算", range: "A1:B2" }] }));
    expect(ops).toHaveLength(1);
    expect(ops[0].type).toBe("clean");
  });

  it("非法 JSON / 非数组 / 缺省 → 空数组", () => {
    expect(parseOps("not-json{")).toEqual([]);
    expect(parseOps('{"a":1}')).toEqual([]);
    expect(parseOps(undefined)).toEqual([]);
    expect(parseOps("")).toEqual([]);
    expect(parseOps(JSON.stringify([{ nope: 1 }]))).toEqual([]);
  });
});

describe("compareCell 归一化比对", () => {
  it("数值容差 1e-9：120.50 vs 120.5 → match；0.3000000001 vs 0.3 → match", () => {
    expect(compareCell("120.50", "120.5")).toBe("match");
    expect(compareCell("0.3000000001", "0.3")).toBe("match");
    expect(compareCell("120.5", "120.6")).toBe("mismatch");
  });

  it("字符串去空白：' abc ' vs 'abc' → match", () => {
    expect(compareCell(" abc ", "abc")).toBe("match");
    expect(compareCell("abc", "abd")).toBe("mismatch");
  });

  it("公式归一：去前导 = 与空白 → match；函数名大小写不同 → mismatch（只做声明要求的归一）", () => {
    expect(normFormula("=SUM(B2:B3)")).toBe("SUM(B2:B3)");
    expect(normFormula("  = SUM(B2:B3)  ")).toBe("SUM(B2:B3)");
    expect(compareCell("=SUM(B2:B3)", undefined, "SUM(B2:B3)", true)).toBe("match");
    expect(compareCell("SUM(B2:B3)", undefined, "sum(b2:b3)", true)).toBe("mismatch");
  });

  it("实况缺失（无该格/无公式文本）→ skip", () => {
    expect(compareCell("x", undefined, undefined, false)).toBe("skip");
    expect(compareCell("=SUM(A1)", undefined, undefined, true)).toBe("skip");
  });

  it("isClaimableOp 只认 set_value/set_formula/replace", () => {
    expect(isClaimableOp({ type: "set_value" })).toBe(true);
    expect(isClaimableOp({ type: "set_formula" })).toBe(true);
    expect(isClaimableOp({ type: "replace" })).toBe(true);
    expect(isClaimableOp({ type: "fill_range" })).toBe(false);
  });
});

describe("buildCellIndex 实况索引", () => {
  const body = JSON.stringify({
    sheets: [
      {
        name: "预算",
        rows: [
          [{ ref: "B2", value: "120.50", type: "number" }],
          [{ ref: "B4", value: "200.50", formula: "SUM(B2:B3)", type: "string" }],
        ],
      },
    ],
  });

  it("按 sheet!cell 建索引；ref 统一大写", () => {
    const idx = buildCellIndex(body);
    expect(idx[cellKey("预算", "b2")]?.value).toBe("120.50");
    expect(idx["预算!B4"]?.formula).toBe("SUM(B2:B3)");
    expect(cellKey("预算", "b4")).toBe("预算!B4");
  });

  it("非法 body → 空索引", () => {
    expect(buildCellIndex("nope")).toEqual({});
    expect(buildCellIndex(undefined)).toEqual({});
  });
});

describe("buildVerifyDiff 组装", () => {
  const body = JSON.stringify({
    sheets: [
      {
        name: "预算",
        rows: [
          [{ ref: "B2", value: "120.50", type: "number" }],
          [{ ref: "B4", value: "200.50", formula: "SUM(B2:B3)", type: "string" }],
        ],
      },
    ],
  });
  const index = buildCellIndex(body);

  it("set_value/set_formula 单格行 + replace 批量跳过行", () => {
    const ops = parseOps(JSON.stringify([
      { type: "set_value", sheet: "预算", target: "B2", value: 120.5 },
      { type: "set_formula", sheet: "预算", target: "B4", formula: "SUM(B2:B3)" },
      { type: "replace", sheet: "预算", range: "A1:A3", find: "设备", replace: "机械" },
      { type: "fill_range", sheet: "预算", range: "C1:C5", value: 0 },
    ]));
    const rows = buildVerifyDiff(ops, index);
    expect(rows).toHaveLength(3);
    expect(rows[0]).toMatchObject({ sheet: "预算", cell: "B2", claimed: "120.5", actual: "120.50", ok: "match" });
    expect(rows[1]).toMatchObject({ sheet: "预算", cell: "B4", claimed: "fx =SUM(B2:B3)", actual: "fx =SUM(B2:B3)", ok: "match" });
    expect(rows[2]).toMatchObject({ sheet: "预算", cell: "A1:A3", claimed: "设备 → 机械", ok: "skip" });
  });

  it("数值不一致 → mismatch；预览缺格 → skip", () => {
    const ops = parseOps(JSON.stringify([
      { type: "set_value", sheet: "预算", target: "B2", value: 999 },
      { type: "set_value", sheet: "预算", target: "Z9", value: "找不到" },
    ]));
    const rows = buildVerifyDiff(ops, index);
    expect(rows[0].ok).toBe("mismatch");
    expect(rows[1].ok).toBe("skip");
  });

  it("无 sheet 的 op 不产出行", () => {
    const ops = parseOps(JSON.stringify([{ type: "set_value", target: "B2", value: 1 }]));
    expect(buildVerifyDiff(ops, index)).toEqual([]);
  });
});

describe("describeOp / opImpact / opBatchCount 回放描述", () => {
  it("描述对齐 applyOne 风格", () => {
    expect(describeOp({ type: "set_value", sheet: "预算", target: "B4", value: 100 })).toBe("写入值 B4=100");
    expect(describeOp({ type: "set_formula", sheet: "预算", target: "B4", formula: "SUM(B2:B3)" })).toBe("写入公式 B4=SUM(B2:B3)");
    expect(describeOp({ type: "replace", sheet: "预算", range: "A1:B10", find: "旧", replace: "新" })).toBe("替换 A1:B10：旧 → 新");
    expect(describeOp({ type: "fill_range", sheet: "预算", range: "C1:C5", value: 0 })).toBe("填充 C1:C5 = 0");
    expect(describeOp({ type: "merge_cells", sheet: "预算", range: "A1:B1" })).toBe("合并 A1:B1");
    expect(describeOp({ type: "split_column", sheet: "预算", col: "A", sep: ",", newCols: ["B", "C"] })).toBe("拆分 A 列");
    expect(describeOp({ type: "weird", sheet: "预算" })).toBe("未知操作 weird");
  });

  it("影响区域 = sheet!target / sheet!range / sheet!col 列", () => {
    expect(opImpact({ type: "set_value", sheet: "预算", target: "B4" })).toBe("预算!B4");
    expect(opImpact({ type: "replace", sheet: "预算", range: "A1:B10" })).toBe("预算!A1:B10");
    expect(opImpact({ type: "set_col_width", sheet: "预算", col: "A" })).toBe("预算!A 列");
  });

  it("批量 op 折叠计数：fill_range/clean → N 格，transform → N 行，split_column → N 列，replace 不给", () => {
    expect(opBatchCount({ type: "fill_range", range: "A1:B3" })).toBe("6 格");
    expect(opBatchCount({ type: "clean", range: "A1:A5" })).toBe("5 格");
    expect(opBatchCount({ type: "transform", range: "B2:B10" })).toBe("9 行");
    expect(opBatchCount({ type: "split_column", newCols: ["B", "C", "D"] })).toBe("3 列");
    expect(opBatchCount({ type: "replace", range: "A1:B10" })).toBeNull();
    expect(opBatchCount({ type: "set_value", target: "B4" })).toBeNull();
  });
});
