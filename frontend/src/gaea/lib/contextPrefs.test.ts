// contextPrefs 单测：逐字段校验与损坏回落（2.5d 设置中心默认）。
import { beforeEach, describe, expect, it } from "vitest";
import {
  CONTEXT_PREFS_DEFAULTS,
  loadContextPrefs,
  saveContextPref,
} from "./contextPrefs";

const KEY = "gaea.context.prefs";

describe("contextPrefs 上下文页偏好", () => {
  beforeEach(() => {
    localStorage.removeItem(KEY);
  });

  it("未设置时返回全默认", () => {
    expect(loadContextPrefs()).toEqual(CONTEXT_PREFS_DEFAULTS);
  });

  it("saveContextPref 合并写入单字段，其余保持默认", () => {
    saveContextPref("trendGranularity", "turn");
    expect(loadContextPrefs()).toEqual({ ...CONTEXT_PREFS_DEFAULTS, trendGranularity: "turn" });
    saveContextPref("fileSort", "path");
    expect(loadContextPrefs()).toEqual({ ...CONTEXT_PREFS_DEFAULTS, trendGranularity: "turn", fileSort: "path" });
  });

  it("损坏 JSON 整包回落默认（不抛错）", () => {
    localStorage.setItem(KEY, "{not-json");
    expect(loadContextPrefs()).toEqual(CONTEXT_PREFS_DEFAULTS);
  });

  it("非法值逐字段回落，合法字段保留（不整包拒绝）", () => {
    localStorage.setItem(KEY, JSON.stringify({ trendGranularity: "bogus", browserSort: "size", extra: 1 }));
    expect(loadContextPrefs()).toEqual({ ...CONTEXT_PREFS_DEFAULTS, browserSort: "size" });
  });

  it("非对象 JSON 回落默认", () => {
    localStorage.setItem(KEY, JSON.stringify([1, 2]));
    expect(loadContextPrefs()).toEqual(CONTEXT_PREFS_DEFAULTS);
  });
});
