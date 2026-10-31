import { beforeEach, describe, expect, it } from "vitest";
import {
  FALLBACK_TEMPLATES,
  loadTemplates,
  resetTemplatesCacheForTest,
  resolveTemplates,
} from "./taskTemplates";

beforeEach(() => {
  resetTemplatesCacheForTest();
});

describe("办公任务模板兜底", () => {
  it("空库或失败时返回内置模板", () => {
    expect(resolveTemplates(null)).toHaveLength(FALLBACK_TEMPLATES.length);
    expect(resolveTemplates([])).toHaveLength(FALLBACK_TEMPLATES.length);
    const remote = [{ name: "custom", title: "自定义", description: "d", prompt: "p" }];
    expect(resolveTemplates(remote)).toEqual(remote);
  });

  it("模板加载后命中缓存", async () => {
    const first = await loadTemplates();
    const second = await loadTemplates();
    expect(first.length).toBeGreaterThan(0);
    expect(second).toEqual(first);
  });
});
