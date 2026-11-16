import { describe, expect, it, vi } from "vitest";
import { FALLBACK_TEMPLATES, loadTemplates, resetTemplatesCacheForTest } from "./taskTemplates";

// 审查批2 刀1 行为锁：图文模板单源=内置九条。若有人恢复「远端优先」取
// OfficeB 的 GaeaTaskTemplates（办公八条指挥图文已卸的工具），本锁即红。
vi.mock("./bridge", () => ({
  app: {
    TaskTemplates: vi.fn(async () => {
      throw new Error("图文树不应取远端办公模板");
    }),
  },
}));

describe("taskTemplates（图文单源）", () => {
  it("loadTemplates 恒用内置写作模板，不取 OfficeB 远端", async () => {
    resetTemplatesCacheForTest();
    const ts = await loadTemplates();
    expect(ts).toBe(FALLBACK_TEMPLATES);
    expect(ts.length).toBeGreaterThanOrEqual(9);
    const { app } = await import("./bridge");
    expect(vi.mocked(app.TaskTemplates)).not.toHaveBeenCalled();
  });
  it("二次调用走缓存（同一数组引用）", async () => {
    resetTemplatesCacheForTest();
    const first = await loadTemplates();
    const second = await loadTemplates();
    expect(second).toBe(first);
  });
});
