import { describe, expect, it } from "vitest";
import { graphemeWidth, lineWidth, truncateMiddleToWidth, truncateToWidth } from "./textMetrics";

describe("textMetrics display width", () => {
  it("CJK and emoji count as two columns; combining marks do not add columns", () => {
    expect(lineWidth("abc")).toBe(3);
    expect(lineWidth("中文")).toBe(4);
    expect(lineWidth("👩‍💻")).toBe(2);
    expect(lineWidth("e\u0301")).toBe(1);
    expect(graphemeWidth("🎉")).toBe(2);
  });

  it("truncateToWidth is grapheme-safe and maxWidth=0 returns empty", () => {
    expect(truncateToWidth("abcdef", 0)).toBe("");
    expect(truncateToWidth("中文测试", 5)).toBe("中文…");
    expect(truncateToWidth("a👩‍💻b", 3)).toBe("a…");
    expect(lineWidth(truncateToWidth("中a🎉中", 5))).toBeLessThanOrEqual(5);
  });

  it("truncateMiddleToWidth keeps both ends without splitting emoji", () => {
    const out = truncateMiddleToWidth("HEAD-" + "中".repeat(20) + "-TAIL🎉", 15);
    expect(out.startsWith("HEAD")).toBe(true);
    expect(out.endsWith("TAIL🎉")).toBe(true);
    expect(lineWidth(out)).toBeLessThanOrEqual(15);
    expect(out).toContain("🎉");
  });
});
