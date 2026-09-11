// 变参绑定 wire 契约（v4.236 真机走查抓崩的回归锁）：
// Go 变参绑定（...string）经 Wails 要求每个元素作为独立顶层参数
// （args:["path"]）；JS 侧传数组会得到 reflect []string as string 错误。
// v4.174 退役 wailsjsCompat shim 后，数组约定调用点静默失去展开——本测试
// 钉死桥接面契约：facade 收可选单串，Go 边界收到的是 string 而非数组。
import { afterEach, describe, expect, it, vi } from "vitest";
import { app } from "./bridge";

type G = { go?: { app?: Record<string, Record<string, unknown>> } };
function inject(ns: Record<string, Record<string, unknown>>): void {
  (window as unknown as G).go = { app: ns };
}
function clearInject(): void {
  delete (window as unknown as G).go;
}

describe("变参绑定 wire 契约", () => {
  afterEach(() => clearInject());

  it("AgentNetwork 单路径按字符串传递（顶层 string，非数组包裹）", async () => {
    const spy = vi.fn(async () => ({ ok: true }));
    inject({ OfficeB: { GaeaAgentNetwork: spy, GaeaLogFrontendError: vi.fn(async () => {}) } });
    await app.AgentNetwork("s.json");
    expect(spy).toHaveBeenCalledTimes(1);
    expect(spy.mock.calls[0]).toEqual(["s.json"]);
  });

  it("AgentNetwork 无参：零实参（变参空集）", async () => {
    const spy = vi.fn(async () => ({ ok: true }));
    inject({ OfficeB: { GaeaAgentNetwork: spy, GaeaLogFrontendError: vi.fn(async () => {}) } });
    await app.AgentNetwork();
    expect(spy.mock.calls[0]).toEqual([]);
  });

  it("TaskList 无参：space 变参为空（任务中心全量拉取）", async () => {
    const spy = vi.fn(async () => []);
    inject({ OfficeB: { GaeaTaskList: spy, GaeaLogFrontendError: vi.fn(async () => {}) } });
    await app.TaskList();
    expect(spy.mock.calls[0]).toEqual([]);
  });

  it("Trajectory 单路径同契约", async () => {
    const spy = vi.fn(async () => ({ ok: true }));
    inject({ OfficeB: { GaeaTrajectory: spy, GaeaLogFrontendError: vi.fn(async () => {}) } });
    await app.Trajectory("t.json");
    expect(spy.mock.calls[0]).toEqual(["t.json"]);
  });
});
