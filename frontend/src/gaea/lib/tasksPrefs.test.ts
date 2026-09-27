import { beforeEach, describe, expect, it } from "vitest";
import { loadTasksAutoOpenSubagent, saveTasksAutoOpenSubagent } from "./tasksPrefs";

// 设置中心「办公工作台偏好」卡入口：为 gaea.tasks.autoOpenSubagent 补 load/save。
// 2026-09-27 用户拍板「右侧面板启动默认隐藏，不要打开办公板块就打开」：默认由
// 开改关——读取端（本模块与 App 触发端 useTasksAutoOpen）同口径只认显式开启值
// "1"/"true"，未设置/损坏值一律默认关。
describe("tasksPrefs 新任务自动切任务视图偏好", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("未设置时默认关（右侧面板启动默认隐藏）", () => {
    expect(loadTasksAutoOpenSubagent()).toBe(false);
  });

  it("save 后 load 回读一致，且写值与 App 触发端读取约定兼容", () => {
    saveTasksAutoOpenSubagent(true);
    expect(loadTasksAutoOpenSubagent()).toBe(true);
    // 触发端只认显式开启值 "1"
    expect(window.localStorage.getItem("gaea.tasks.autoOpenSubagent")).toBe("1");

    saveTasksAutoOpenSubagent(false);
    expect(loadTasksAutoOpenSubagent()).toBe(false);
    expect(window.localStorage.getItem("gaea.tasks.autoOpenSubagent")).toBe("0");
  });

  it("显式开启值（\"1\"/\"true\"）为开", () => {
    window.localStorage.setItem("gaea.tasks.autoOpenSubagent", "1");
    expect(loadTasksAutoOpenSubagent()).toBe(true);
    window.localStorage.setItem("gaea.tasks.autoOpenSubagent", "true");
    expect(loadTasksAutoOpenSubagent()).toBe(true);
  });

  it("关闭值与损坏值回落默认关（try/catch 降级语义）", () => {
    window.localStorage.setItem("gaea.tasks.autoOpenSubagent", "0");
    expect(loadTasksAutoOpenSubagent()).toBe(false);
    window.localStorage.setItem("gaea.tasks.autoOpenSubagent", "garbage!!");
    expect(loadTasksAutoOpenSubagent()).toBe(false);
  });
});
