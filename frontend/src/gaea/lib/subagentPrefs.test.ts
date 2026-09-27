import { beforeEach, describe, expect, it } from "vitest";
import { loadSubagentAutoOpen, saveSubagentAutoOpen } from "./subagentPrefs";

describe("subagentPrefs 新子代理自动展开偏好", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  // 2026-09-27 用户拍板「右侧面板启动默认隐藏，不要打开办公板块就打开」：
  // 默认由开改关——未设置/损坏值一律关，只有显式开启值生效。
  it("未设置时默认关（右侧面板启动默认隐藏）", () => {
    expect(loadSubagentAutoOpen()).toBe(false);
  });

  it("save 后 load 回读一致（开/关往返）", () => {
    saveSubagentAutoOpen(true);
    expect(loadSubagentAutoOpen()).toBe(true);
    expect(window.localStorage.getItem("gaea.subagentAutoOpen")).toBe("1");

    saveSubagentAutoOpen(false);
    expect(loadSubagentAutoOpen()).toBe(false);
    expect(window.localStorage.getItem("gaea.subagentAutoOpen")).toBe("0");
  });

  it("显式开启值（1/true）为开", () => {
    window.localStorage.setItem("gaea.subagentAutoOpen", "1");
    expect(loadSubagentAutoOpen()).toBe(true);
    window.localStorage.setItem("gaea.subagentAutoOpen", "true");
    expect(loadSubagentAutoOpen()).toBe(true);
  });

  it("关闭值与损坏值回落默认关（try/catch 降级语义）", () => {
    window.localStorage.setItem("gaea.subagentAutoOpen", "0");
    expect(loadSubagentAutoOpen()).toBe(false);
    window.localStorage.setItem("gaea.subagentAutoOpen", "garbage!!");
    expect(loadSubagentAutoOpen()).toBe(false);
  });
});
