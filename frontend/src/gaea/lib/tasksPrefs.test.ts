import { beforeEach, describe, expect, it } from "vitest";
import { loadTasksAutoOpenSubagent, saveTasksAutoOpenSubagent } from "./tasksPrefs";

// 设置中心「办公工作台偏好」卡接入（v4.65 欠账）：为 gaea.tasks.autoOpenSubagent
// 补 load/save 入口。写值必须与 App.tsx 触发端的 inline 读取约定兼容
// （App 只认 "0" 为关，其余一律开）。
describe("tasksPrefs 新任务自动切任务视图偏好", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("未设置时默认开（App 触发端语义：缺省不打断）", () => {
    expect(loadTasksAutoOpenSubagent()).toBe(true);
  });

  it("save 后 load 回读一致，且写值与 App 触发端 inline 读取约定兼容", () => {
    saveTasksAutoOpenSubagent(false);
    expect(loadTasksAutoOpenSubagent()).toBe(false);
    // App.tsx：localStorage.getItem("gaea.tasks.autoOpenSubagent") === "0" → 关
    expect(window.localStorage.getItem("gaea.tasks.autoOpenSubagent")).toBe("0");

    saveTasksAutoOpenSubagent(true);
    expect(loadTasksAutoOpenSubagent()).toBe(true);
    // "1" ≠ "0" → App 触发端按开处理
    expect(window.localStorage.getItem("gaea.tasks.autoOpenSubagent")).toBe("1");
  });

  it("显式关闭值（\"0\"/\"false\"）视为关", () => {
    window.localStorage.setItem("gaea.tasks.autoOpenSubagent", "0");
    expect(loadTasksAutoOpenSubagent()).toBe(false);
    window.localStorage.setItem("gaea.tasks.autoOpenSubagent", "false");
    expect(loadTasksAutoOpenSubagent()).toBe(false);
  });

  it("损坏值回落默认开（try/catch 降级语义）", () => {
    window.localStorage.setItem("gaea.tasks.autoOpenSubagent", "garbage!!");
    expect(loadTasksAutoOpenSubagent()).toBe(true);
    window.localStorage.setItem("gaea.tasks.autoOpenSubagent", "true");
    expect(loadTasksAutoOpenSubagent()).toBe(true);
  });
});
