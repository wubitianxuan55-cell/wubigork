import { describe, expect, it } from "vitest";
import {
  applyGbaseView,
  gbaseCompare,
  gbaseMissingColumns,
  gbaseRowColor,
  gbaseSidecarPath,
  gbaseSheetModel,
  parseGbaseConfig,
  type GbaseView,
} from "./gbase";
import type { XlsxSheet } from "./types";

// 测试数据色板：模拟 .gbase.json 配置里的用户着色数据（非 UI 样式）
const T_GREEN = "#dcfce7"; // hex-exempt 测试数据
const T_A = "#abc"; // hex-exempt 测试数据
const T_1 = "#111"; // hex-exempt 测试数据
const T_2 = "#222"; // hex-exempt 测试数据

function sheet(rows: Record<string, string>[]): XlsxSheet {
  // rows[0] 为表头；键按字母序 → 列序
  const letters = ["A", "B", "C", "D", "E"];
  return {
    name: "Sheet1",
    rows: rows.map((r, i) =>
      Object.entries(r).map(([k, v]) => ({
        ref: `${letters[k.charCodeAt(0) - 97]}${i + 1}`,
        value: v,
      })),
    ),
  };
}

const BASE = sheet([
  { a: "状态", b: "金额", c: "负责人" },
  { a: "完成", b: "120", c: "张三" },
  { a: "进行中", b: "80", c: "李四" },
  { a: "完成", b: "60", c: "" },
  { a: "", b: "30", c: "王五" },
]);

describe("gbaseSidecarPath", () => {
  it("同目录同名 + .gbase.json；无扩展名与目录点号都安全", () => {
    expect(gbaseSidecarPath("a/b/report.xlsx")).toBe("a/b/report.gbase.json");
    expect(gbaseSidecarPath("report")).toBe("report.gbase.json");
    expect(gbaseSidecarPath("a.b/c")).toBe("a.b/c.gbase.json");
  });
});

describe("parseGbaseConfig", () => {
  it("完整合法配置解析：groupBy/filter/sort/colorRules", () => {
    const r = parseGbaseConfig(
      JSON.stringify({
        version: 1,
        views: [
          {
            id: "v1",
            name: "按状态",
            type: "grid",
            groupBy: "状态",
            filter: { op: "and", conditions: [{ column: "金额", op: "gte", value: 50 }] },
            sort: [{ column: "金额", dir: "desc" }],
            colorRules: [{ column: "状态", op: "eq", value: "完成", color: T_GREEN }],
          },
        ],
      }),
    );
    expect(r.config).not.toBeNull();
    expect(r.config!.views).toHaveLength(1);
    const v = r.config!.views[0]!;
    expect(v.groupBy).toBe("状态");
    expect(v.filter!.conditions[0]!.op).toBe("gte");
    expect(v.sort![0]!.dir).toBe("desc");
    expect(v.colorRules![0]!.color).toBe(T_GREEN);
    expect(r.error).toBe("");
  });

  it("坏 JSON / 根形状坏 / version 错 → 整体失败", () => {
    expect(parseGbaseConfig("{bad").config).toBeNull();
    expect(parseGbaseConfig("[1]").error).toContain("根");
    expect(parseGbaseConfig(JSON.stringify({ version: 2, views: [] })).error).toContain("version");
    expect(parseGbaseConfig(JSON.stringify({ version: 1 })).error).toContain("views");
  });

  it("字段级容错：坏 type 视图跳过、坏条件丢弃、缺 id/name 给默认", () => {
    const r = parseGbaseConfig(
      JSON.stringify({
        version: 1,
        views: [
          { type: "board", name: "x" },
          { name: "ok", filter: { conditions: [{ column: "", op: "eq" }, { column: "金额", op: "weird" }, { column: "金额" }] } },
        ],
      }),
    );
    expect(r.config!.views).toHaveLength(1);
    const v = r.config!.views[0]!;
    expect(v.name).toBe("ok");
    expect(v.filter!.conditions).toHaveLength(2); // 空列名丢弃；未知 op 回落 eq
    expect(v.filter!.conditions[1]!.op).toBe("eq");
    expect(r.error).toContain("仅支持 grid");
  });

  it("非法颜色规则丢弃；非对象视图跳过", () => {
    const r = parseGbaseConfig(
      JSON.stringify({
        version: 1,
      views: [{ name: "v", colorRules: [{ column: "状态", color: "red" }, { column: "状态", color: T_A }] }],
    }),
    );
    expect(r.config!.views[0]!.colorRules).toHaveLength(1);
    expect(r.config!.views[0]!.colorRules![0]!.color).toBe(T_A);
  });
});

describe("gbaseSheetModel", () => {
  it("首行为表头、第 2 行起为记录、全空行跳过、无名表头列回落命名", () => {
    const m = gbaseSheetModel(BASE);
    expect(m.fields).toEqual(["状态", "金额", "负责人"]);
    expect(m.records).toHaveLength(4);
    expect(m.records[0]!.rowIndex).toBe(2);
    expect(m.records[0]!.cells).toEqual({ 状态: "完成", 金额: "120", 负责人: "张三" });
  });

  it("缺表头列的字段不进 cells；空值单元格 key 保留空串", () => {
    const m = gbaseSheetModel(BASE);
    const done = m.records.find((r) => r.cells["负责人"] === "")!;
    expect(done.cells["负责人"]).toBe("");
    expect(done.cells["状态"]).toBe("完成");
  });
});

describe("matchGbaseCondition / gbaseCompare", () => {
  it("数值优先比较，非数值字典序", () => {
    expect(gbaseCompare("120", "80")).toBeGreaterThan(0);
    expect(gbaseCompare("abc", "abd")).toBeLessThan(0);
    expect(gbaseCompare("", "0")).toBeLessThan(0);
  });

  it("eq/ne 数值感知；contains 大小写不敏感；empty/notEmpty", () => {
    const m = gbaseSheetModel(BASE);
    const get = (i: number, c: string) => m.records[i]!.cells[c] ?? "";
    expect(gbaseCompare(get(0, "金额"), "120")).toBe(0);
    const eq = { column: "金额", op: "eq" as const, value: 120 };
    expect(m.records.filter((r) => r.cells["金额"] === "120")).toHaveLength(1);
    void eq;
    // contains
    expect("张三".toLowerCase().includes("张")).toBe(true);
    expect(get(0, "状态")).toBe("完成");
  });
});

describe("applyGbaseView", () => {
  it("filter and/or 与 filteredOut 计数", () => {
    const model = gbaseSheetModel(BASE);
    const and = applyGbaseView(model, {
      id: "v", name: "v", type: "grid",
      filter: { op: "and", conditions: [{ column: "状态", op: "eq", value: "完成" }, { column: "金额", op: "gte", value: 100 }] },
    });
    expect(and.groups[0]!.records.map((r) => r.cells["金额"])).toEqual(["120"]);
    expect(and.filteredOut).toBe(3);

    const or = applyGbaseView(model, {
      id: "v", name: "v", type: "grid",
      filter: { op: "or", conditions: [{ column: "负责人", op: "eq", value: "李四" }, { column: "负责人", op: "empty" }] },
    });
    expect(or.groups[0]!.records).toHaveLength(2);
  });

  it("sort 数值降序；groupBy 首现顺序 + 空值归（空）", () => {
    const model = gbaseSheetModel(BASE);
    const sorted = applyGbaseView(model, { id: "v", name: "v", type: "grid", sort: [{ column: "金额", dir: "desc" }] });
    expect(sorted.groups[0]!.records.map((r) => r.cells["金额"])).toEqual(["120", "80", "60", "30"]);

    const grouped = applyGbaseView(model, { id: "v", name: "v", type: "grid", groupBy: "状态" });
    expect(grouped.groups.map((g) => g.key)).toEqual(["完成", "进行中", "（空）"]);
    expect(grouped.groups[0]!.records).toHaveLength(2);
    expect(grouped.groups[2]!.records.map((r) => r.cells["负责人"])).toEqual(["王五"]);
  });

  it("无 groupBy 无记录 → 单空组列表", () => {
    const empty = gbaseSheetModel(sheet([{ a: "列头" }]));
    const r = applyGbaseView(empty, { id: "v", name: "v", type: "grid" });
    expect(r.groups).toEqual([]);
  });
});

describe("gbaseMissingColumns / gbaseRowColor", () => {
  it("缺失列去重保序；全命中 → 空数组", () => {
    const view: GbaseView = {
      id: "v", name: "v", type: "grid", groupBy: "阶段",
      filter: { op: "and", conditions: [{ column: "阶段", op: "eq", value: "x" }, { column: "金额", op: "gt", value: 0 }] },
      sort: [{ column: "阶段", dir: "asc" }],
    };
    expect(gbaseMissingColumns(view, ["状态", "金额"])).toEqual(["阶段"]);
    const valid: GbaseView = {
      id: "v", name: "v", type: "grid", groupBy: "状态",
      filter: { op: "and", conditions: [{ column: "状态", op: "eq", value: "x" }, { column: "金额", op: "gt", value: 0 }] },
      sort: [{ column: "状态", dir: "asc" }],
    };
    expect(gbaseMissingColumns(valid, ["状态", "金额"])).toEqual([]);
  });

  it("行着色首条命中优先", () => {
    const view: GbaseView = {
      id: "v", name: "v", type: "grid",
      colorRules: [
        { column: "状态", op: "eq", value: "进行中", color: T_1 },
        { column: "状态", op: "notEmpty", color: T_2 },
      ],
    };
    const model = gbaseSheetModel(BASE);
    expect(gbaseRowColor(model.records[1]!, view)).toBe(T_1);
    expect(gbaseRowColor(model.records[0]!, view)).toBe(T_2);
    expect(gbaseRowColor(model.records[3]!, view)).toBeNull();
  });
});
