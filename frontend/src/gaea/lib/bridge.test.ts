// bridge 错误归一化层测试（T6-1.2 前端错误可见性）：
// 所有绑定调用失败统一归一为结构化 BridgeError（code/message），
// 并调用 LogFrontendError 记录到 gaea.log。
import { afterEach, describe, expect, it, vi } from "vitest";
import { app } from "./bridge";
import { onTaskEvent } from "./bridge";
import { mockTaskListeners } from "./mock/shared";

// 安装假 Wails 门面：让 realApp() 命中 CoreB 板块，绕过 dev mock。
// 返回门面对象供 vi.spyOn / 断言使用。
function installFakeGo(facade: Record<string, unknown>): Record<string, unknown> {
  (window as unknown as { go?: { app?: Record<string, unknown> } }).go = { app: { CoreB: facade } };
  return facade;
}

afterEach(() => {
  delete (window as unknown as { go?: { app?: Record<string, unknown> } }).go;
});

describe("bridge invoke 错误归一化", () => {
  it("后端调用失败归一为 { code, message } 结构化错误（Error → code/message）", async () => {
    installFakeGo({ GaeaMeta: async () => { throw new Error("meta boom"); } });
    const err = await app.Meta().catch((e: unknown) => e);
    expect(err).toMatchObject({ code: "MetaError", message: "meta boom" });
  });

  it("后端已带结构化错误（code/message）时原样透传", async () => {
    installFakeGo({ GaeaBalance: async () => { throw { code: "BalOffline", message: "余额服务不可用" }; } });
    await expect(app.Balance()).rejects.toMatchObject({ code: "BalOffline", message: "余额服务不可用" });
  });

  it("字符串/未知拒绝值也归一为结构化错误", async () => {
    installFakeGo({ GaeaHistory: async () => { throw "raw string error"; } });
    await expect(app.History()).rejects.toMatchObject({ code: "HistoryError", message: "raw string error" });
  });

  it("归一化错误仍是 Error：既有 e instanceof Error / e.message 判定不破坏", async () => {
    installFakeGo({ GaeaMeta: async () => { throw new Error("meta boom"); } });
    const err = await app.Meta().catch((e: unknown) => e);
    expect(err instanceof Error).toBe(true);
    expect((err as Error).message).toBe("meta boom");
  });

  it("失败时调用 LogFrontendError 记录（vi.spyOn）", async () => {
    const facade = {
      GaeaMeta: async (): Promise<never> => { throw new Error("meta boom"); },
      GaeaLogFrontendError: async (): Promise<void> => {},
    };
    installFakeGo(facade);
    const spy = vi.spyOn(facade, "GaeaLogFrontendError");
    await expect(app.Meta()).rejects.toMatchObject({ code: "MetaError" });
    expect(spy).toHaveBeenCalledTimes(1);
    expect(spy).toHaveBeenCalledWith(expect.stringContaining("[MetaError] Meta 失败: meta boom"));
  });

  it("日志通道缺失（未注入 GaeaLogFrontendError）时不影响原始错误透传", async () => {
    installFakeGo({ GaeaJobs: async () => { throw new Error("jobs down"); } });
    await expect(app.Jobs()).rejects.toMatchObject({ code: "JobsError", message: "jobs down" });
  });
});

// v4.5.1a 红线补课：onTaskEvent 订阅层空间过滤——传 "work" 时 play 任务事件
// （payload.spaceId="play"）被丢弃，work/缺省 spaceId 事件照常放行。
describe("onTaskEvent 空间过滤", () => {
  it("work 订阅丢弃 play 任务事件、放行 work/缺省 spaceId 事件", () => {
    const got: string[] = [];
    const off = onTaskEvent((t) => got.push(t.id), "work");
    try {
      mockTaskListeners.forEach((l) => l({
        id: "play-task", kind: "price_fetch", label: "play", status: "running",
        progress: 0, message: "", error: "", retryCount: 0, maxRetries: 2,
        payload: "{}", result: "", createdAt: 0, startedAt: 0, finishedAt: 0,
        spaceId: "play",
      }));
      mockTaskListeners.forEach((l) => l({
        id: "work-task", kind: "price_fetch", label: "work", status: "running",
        progress: 0, message: "", error: "", retryCount: 0, maxRetries: 2,
        payload: "{}", result: "", createdAt: 0, startedAt: 0, finishedAt: 0,
        spaceId: "work",
      }));
      mockTaskListeners.forEach((l) => l({
        id: "legacy-task", kind: "file_index", label: "legacy", status: "succeeded",
        progress: 100, message: "", error: "", retryCount: 0, maxRetries: 2,
        payload: "{}", result: "{}", createdAt: 0, startedAt: 0, finishedAt: 0,
      }));
    } finally {
      off();
    }
    expect(got).toEqual(["work-task", "legacy-task"]);
  });

  it("不传 space 时不过滤（旧行为）", () => {
    const got: string[] = [];
    const off = onTaskEvent((t) => got.push(t.id));
    try {
      mockTaskListeners.forEach((l) => l({
        id: "play-task", kind: "price_fetch", label: "play", status: "running",
        progress: 0, message: "", error: "", retryCount: 0, maxRetries: 2,
        payload: "{}", result: "", createdAt: 0, startedAt: 0, finishedAt: 0,
        spaceId: "play",
      }));
    } finally {
      off();
    }
    expect(got).toEqual(["play-task"]);
  });
});
