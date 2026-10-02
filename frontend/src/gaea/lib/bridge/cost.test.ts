// cost.test.ts — GA6-09：CostSearchPage.Error → 前端提示条文案（读取失败可见化）。
import { describe, expect, it } from "vitest";
import { costReadErrorText } from "./cost";

describe("costReadErrorText（Go CostSearchPage.Error → 提示条文案）", () => {
  it("Go 文案已含「读取失败」：原样透传，不叠成「读取失败：读取失败（…）」", () => {
    const go = "成本库读取失败（disk io error），结果可能不完整";
    expect(costReadErrorText(go)).toBe(go);
    expect(costReadErrorText(go).match(/读取失败/g)).toHaveLength(1);
  });

  it("其余非空失败文案：统一补「读取失败」前缀（提示条恒含该字样）", () => {
    expect(costReadErrorText("bridge down")).toBe("读取失败：bridge down");
    expect(costReadErrorText("  连接超时  ")).toBe("读取失败：连接超时");
  });

  it("空串/空白/undefined/null：视为无错误（不渲染提示条）", () => {
    for (const v of ["", "   ", undefined, null]) {
      expect(costReadErrorText(v)).toBe("");
    }
  });
});
