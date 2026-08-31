import { beforeEach, describe, expect, it } from "vitest";
import { loadSubagentAutoOpen, saveSubagentAutoOpen } from "./subagentPrefs";

describe("subagentPrefs 新子代理自动展开偏好", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("未设置时默认开（对标 better-sidebar 自动展开默认行为）", () => {
    expect(loadSubagentAutoOpen()).toBe(true);
  });

  it("save 后 load 回读一致（开/关往返）", () => {
    saveSubagentAutoOpen(false);
    expect(loadSubagentAutoOpen()).toBe(false);
    expect(window.localStorage.getItem("gaea.subagentAutoOpen")).toBe("0");

    saveSubagentAutoOpen(true);
    expect(loadSubagentAutoOpen()).toBe(true);
    expect(window.localStorage.getItem("gaea.subagentAutoOpen")).toBe("1");
  });

  it("损坏值回落默认开（try/catch 降级语义）", () => {
    window.localStorage.setItem("gaea.subagentAutoOpen", "garbage!!");
    expect(loadSubagentAutoOpen()).toBe(true);
  });
});
