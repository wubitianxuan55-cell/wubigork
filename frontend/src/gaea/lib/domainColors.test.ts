import { describe, expect, it } from "vitest";
import { DOMAIN_COLORS, DOMAIN_LABELS, DOMAIN_KEYS } from "./domainColors";

describe("domainColors 领域色单一数据源", () => {
  it("6 个领域分类都有色值与中文标签", () => {
    expect(DOMAIN_KEYS).toHaveLength(6);
    for (const key of DOMAIN_KEYS) {
      expect(DOMAIN_COLORS[key]).toMatch(/^#[0-9a-fA-F]{6}$/);
      expect(DOMAIN_LABELS[key].length).toBeGreaterThan(0);
    }
  });

  it("领域色值不重复（图谱节点靠色区分）", () => {
    const values = DOMAIN_KEYS.map((k) => DOMAIN_COLORS[k]);
    expect(new Set(values).size).toBe(values.length);
  });

  it("material=sky 且 whisper=pink 与组件历史值一致（回归锚点）", () => {
    expect(DOMAIN_COLORS.material).toBe("#38bdf8"); // MaterialsLibrary Pin
    expect(DOMAIN_COLORS.whisper).toBe("#f472b6"); // WhisperMemoryLibrary 情节记忆
  });
});
