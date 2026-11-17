import { describe, expect, it } from "vitest";
import { firstFittingLine, lineHintsWidth, wrapHintRows, type HintItem } from "./hintLayout";

const nav: HintItem = { id: "nav", keys: ["↑", "↓"], label: "导航" };
const select: HintItem = { id: "select", keys: ["↵"], label: "选择" };
const close: HintItem = { id: "close", keys: ["esc"], label: "关闭" };

describe("hintLayout", () => {
  it("whole shortcut+label variants are selected by width, never cut", () => {
    const variants = [[nav, select, close], [nav, close], [close]];
    expect(firstFittingLine(variants, 120)?.[0].id).toBe("nav");
    expect(firstFittingLine(variants, lineHintsWidth([nav, close]))?.[1].id).toBe("close");
    expect(firstFittingLine(variants, lineHintsWidth([close]))?.[0].id).toBe("close");
    expect(firstFittingLine(variants, 1)).toBeNull();
  });

  it("wrapHintRows moves whole hints to the next row", () => {
    const rows = wrapHintRows([nav, select, close], lineHintsWidth([nav, select]));
    expect(rows).toEqual([[nav, select], [close]]);
  });
});
