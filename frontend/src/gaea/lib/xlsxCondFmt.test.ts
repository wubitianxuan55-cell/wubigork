// xlsxCondFmt.test.ts — 条件格式渲染判定回归（与后端 condfmt.go 口径同源锚定）
import { describe, expect, it } from "vitest";
import type { XlsxCell, XlsxCondRule } from "./types";
import {
  condRuleCovers,
  condRuleMatches,
  condStyleFor,
  parseCellRef,
  parseRange,
  parseThreshold,
} from "./xlsxCondFmt";

const rule = (over: Partial<XlsxCondRule>): XlsxCondRule => ({
  range: "B2:B100",
  op: "greaterThan",
  formulas: ["100"],
  priority: 1,
  fill: "FFC7CE",
  ...over,
});
const cell = (value: string, type = "number"): XlsxCell => ({ ref: "B5", value, type });

describe("parseCellRef / parseRange / parseThreshold", () => {
  it("引用解析：字母列 + 行号，非法输入拒绝", () => {
    expect(parseCellRef("A1")).toEqual({ col: 1, row: 1 });
    expect(parseCellRef("AB12")).toEqual({ col: 28, row: 12 });
    expect(parseCellRef("1A")).toBeNull();
    expect(parseRange("B2:D6")).toEqual({ c1: 2, r1: 2, c2: 4, r2: 6 });
    expect(parseRange("B2")).toEqual({ c1: 2, r1: 2, c2: 2, r2: 2 });
    expect(parseRange("B2:C:D")).toBeNull();
  });

  it("阈值解析：数字与带引号文本，其余拒绝", () => {
    expect(parseThreshold("100")).toEqual({ num: 100 });
    expect(parseThreshold("0.5")).toEqual({ num: 0.5 });
    expect(parseThreshold(`"设备"`)).toEqual({ text: "设备" });
    expect(parseThreshold("$D$1")).toBeNull();
    expect(parseThreshold("")).toBeNull();
  });
});

describe("condRuleCovers / condRuleMatches", () => {
  it("range 包含判定", () => {
    expect(condRuleCovers(rule({}), "B5")).toBe(true);
    expect(condRuleCovers(rule({}), "B101")).toBe(false);
    expect(condRuleCovers(rule({ range: "B2" }), "B2")).toBe(true);
  });

  it("数值比较：大于/介于/不介于", () => {
    expect(condRuleMatches(rule({}), cell("120"))).toBe(true);
    expect(condRuleMatches(rule({}), cell("80"))).toBe(false);
    expect(condRuleMatches(rule({ op: "between", formulas: ["80", "100"] }), cell("90"))).toBe(true);
    expect(condRuleMatches(rule({ op: "between", formulas: ["100", "80"] }), cell("90"))).toBe(true); // 上下限无序
    expect(condRuleMatches(rule({ op: "notBetween", formulas: ["80", "100"] }), cell("120"))).toBe(true);
  });

  it("文本 equal/notEqual 大小写不敏感；空单元格不参与", () => {
    const r = rule({ op: "equal", formulas: [`"设备"`] });
    expect(condRuleMatches(r, cell("设备", "string"))).toBe(true);
    expect(condRuleMatches(r, cell("设备", "string"))).toBe(true);
    expect(condRuleMatches(r, cell("人工", "string"))).toBe(false);
    expect(condRuleMatches(rule({ op: "notEqual", formulas: [`"X"`] }), cell("", "string"))).toBe(false);
  });
});

describe("condStyleFor", () => {
  it("命中返回规则样式；未命中返回 null", () => {
    const rules = [rule({})];
    expect(condStyleFor(rules, "B5", cell("120"))).toEqual({ fill: "FFC7CE", fontColor: undefined, bold: undefined });
    expect(condStyleFor(rules, "B5", cell("80"))).toBeNull();
    expect(condStyleFor(undefined, "B5", cell("120"))).toBeNull();
  });

  it("多规则 priority 小者（高优先级）整体胜出", () => {
    const rules = [
      rule({ priority: 5, fill: "FFF2CC" }),
      rule({ priority: 2, fill: "FFC7CE", fontColor: "9C0006", bold: true }),
    ];
    const st = condStyleFor(rules, "B5", cell("120"));
    expect(st).toEqual({ fill: "FFC7CE", fontColor: "9C0006", bold: true });
  });
});
