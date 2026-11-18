import { beforeEach, describe, expect, it } from "vitest";
import { loadTasksAutoOpenSubagent, saveTasksAutoOpenSubagent } from "./tasksPrefs";

// 设置中心「办公工作台偏好」卡入口：为 gaea.tasks.autoOpenSubagent 补 load/save。
// 默认值历史：v4.63 开 → 2026-09-27 用户拍板关（反感「进板块就撑开」）→
// 2026-11-18 子代理实弹反馈「前端看不到子代理卡」翻回默认开——触发面已被
// 基线机制收敛为「新任务/子代理首次出现」，显式 "0"/"false" 仍可关。
describe("tasksPrefs 新任务自动切任务视图偏好", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("未设置时默认开（2026-11-18 实弹修复：子代理卡要能被看到）", () => {
    expect(loadTasksAutoOpenSubagent()).toBe(true);
  });

  it("save 后 load 回读一致，且写值与 App 触发端读取约定兼容", () => {
    saveTasksAutoOpenSubagent(true);
    expect(loadTasksAutoOpenSubagent()).toBe(true);
    // 触发端持久化恒写 "1"/"0"
    expect(window.localStorage.getItem("gaea.tasks.autoOpenSubagent")).toBe("1");

    saveTasksAutoOpenSubagent(false);
    expect(loadTasksAutoOpenSubagent()).toBe(false);
    expect(window.localStorage.getItem("gaea.tasks.autoOpenSubagent")).toBe("0");
  });

  it("显式关闭值（\"0\"/\"false\"）为关（默认开方言只认显式关闭翻转）", () => {
    window.localStorage.setItem("gaea.tasks.autoOpenSubagent", "0");
    expect(loadTasksAutoOpenSubagent()).toBe(false);
    window.localStorage.setItem("gaea.tasks.autoOpenSubagent", "false");
    expect(loadTasksAutoOpenSubagent()).toBe(false);
  });

  it("损坏值回落默认开（try/catch 降级语义）", () => {
    window.localStorage.setItem("gaea.tasks.autoOpenSubagent", "garbage!!");
    expect(loadTasksAutoOpenSubagent()).toBe(true);
  });
});
