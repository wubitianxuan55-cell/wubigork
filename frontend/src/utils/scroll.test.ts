import { describe, it, expect } from "vitest";
import { isNearBottom } from "./scroll";

describe("isNearBottom", () => {
  it("阈值内视为贴底", () => {
    expect(isNearBottom(0)).toBe(true);
    expect(isNearBottom(79)).toBe(true);
    expect(isNearBottom(80)).toBe(false);
  });

  it("支持自定义阈值", () => {
    expect(isNearBottom(120, 200)).toBe(true);
  });
});
