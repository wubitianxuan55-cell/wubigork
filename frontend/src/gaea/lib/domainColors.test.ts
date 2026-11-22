import { describe, expect, it } from "vitest";
import { DOMAIN_COLORS, DOMAIN_LABELS, DOMAIN_KEYS } from "./domainColors";

describe("domainColors 领域色单一数据源", () => {
  it("6 个领域分类都有色值与中文标签", () => {
    expect(DOMAIN_KEYS).toHaveLength(9); // 6 库分类 + 语义图谱三节点（entity/event/source，v4.210.0）
    for (const key of DOMAIN_KEYS) {
      expect(DOMAIN_COLORS[key]).toMatch(/^var\(--gaea-chart-(?:\d+|muted)\)$/);
      expect(DOMAIN_LABELS[key].length).toBeGreaterThan(0);
    }
  });

  it("领域色值不重复（图谱节点靠色区分）", () => {
    const values = DOMAIN_KEYS.map((k) => DOMAIN_COLORS[k]);
    expect(new Set(values).size).toBe(values.length);
  });

  it("关键领域映射到稳定语义色槽", () => {
    expect(DOMAIN_COLORS.material).toBe("var(--gaea-chart-4)");
    expect(DOMAIN_COLORS.whisper).toBe("var(--gaea-chart-6)");
  });
});
