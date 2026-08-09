import { describe, expect, it } from "vitest";
import { applyTableConversion, detectTableBlock } from "./tableData";

describe("detectTableBlock 粘贴表格识别", () => {
  it("识别 CSV 表格块", () => {
    const text = "项目,金额,备注\n设备,120.5,已到货\n材料,80,待采购";
    const block = detectTableBlock(text);
    expect(block).not.toBeNull();
    expect(block?.rows).toBe(3);
    expect(block?.cols).toBe(3);
  });

  it("识别 TSV 表格块", () => {
    const block = detectTableBlock("名称\t数量\nA\t1\nB\t2");
    expect(block?.rows).toBe(3);
    expect(block?.cols).toBe(2);
  });

  it("散文/单行不误报", () => {
    expect(detectTableBlock("项目、金额、备注，请逐一核对")).toBeNull();
    expect(detectTableBlock("只有一行,两个字段")).toBeNull();
    expect(detectTableBlock("列数,不一致\n第一行,有,三个")).toBeNull();
  });
});

describe("applyTableConversion", () => {
  it("把表格块转为 Markdown 表格", () => {
    const text = "以下是成本：\n项目,金额\n设备,120\n\n其余说明";
    const out = applyTableConversion(text, true);
    expect(out).toContain("| 项目 | 金额 |");
    expect(out).toContain("| --- | --- |");
    expect(out).toContain("| 设备 | 120 |");
    expect(out).toContain("以下是成本：");
    expect(out).toContain("其余说明");
  });

  it("关闭时不转换", () => {
    const text = "项目,金额\n设备,120";
    expect(applyTableConversion(text, false)).toBe(text);
  });
});
