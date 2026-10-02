// FE3-10 回归防线：mock chunk 未就绪即卸载的订阅者不得永久残留。
// events.ts 冷路径把监听者挂在闭包捕获的局部 off 上：React 同一 tick 内挂载又
// 卸载时 cleanup 先跑（off 还是 null），异步 then 之后才真正 add，闭包里的 off
// 已无人再读 → 监听者永久残留、重复 dispatch（气泡倍增、轮询重复拉取）。
// 本用例先 resetModules 让 mock chunk 处于未就绪态，走冷路径后立刻退订，再等
// chunk 就绪，断言目标集合里零残留（与「事件绑定恰好一次」教训同口径）。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

afterEach(() => {
  delete (window as unknown as { go?: unknown }).go;
  delete (window as unknown as { runtime?: unknown }).runtime;
  vi.restoreAllMocks();
});

describe("events.ts 冷路径退订语义（FE3-10 监听者残留根修）", () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it("onEvent：订阅后立即退订 → mock ready 后不残留监听者", async () => {
    // 无 window.go.app → 走 mock 回退面；resetModules 后 mock chunk 尚未就绪
    const events = await import("./events");
    const mockShared = await import("../mock/shared");
    mockShared.mockListeners.clear();

    const cb = vi.fn();
    const off = events.onEvent(cb);
    // 同一 tick 内卸载：此刻 chunk 未就绪（off 仍为 null）
    off();

    // 等 mock chunk 就绪（旧实现在此之后才 add，且 off 已无人读）
    const proxy = await import("./proxy");
    await proxy.waitMockReady();
    await Promise.resolve();

    expect(mockShared.mockListeners.size).toBe(0);
    // 再确认真的没绑：派发事件不应回调到已退订的监听者
    mockShared.emitMock({ seq: 1 } as never);
    expect(cb).not.toHaveBeenCalled();
  });

  it("onEvent：不退订时正常绑定（反向验证：不是恒不绑）", async () => {
    const events = await import("./events");
    const mockShared = await import("../mock/shared");
    mockShared.mockListeners.clear();

    const cb = vi.fn();
    const off = events.onEvent(cb);
    const proxy = await import("./proxy");
    await proxy.waitMockReady();
    await Promise.resolve();

    expect(mockShared.mockListeners.size).toBe(1);
    mockShared.emitMock({ seq: 2 } as never);
    expect(cb).toHaveBeenCalledTimes(1);
    off();
    expect(mockShared.mockListeners.size).toBe(0);
  });

  it("onTaskEvent：订阅后立即退订 → mock ready 后不残留任务监听者", async () => {
    const events = await import("./events");
    const mockShared = await import("../mock/shared");
    mockShared.mockTaskListeners.clear();

    const cb = vi.fn();
    const off = events.onTaskEvent(cb);
    off();

    const proxy = await import("./proxy");
    await proxy.waitMockReady();
    await Promise.resolve();

    expect(mockShared.mockTaskListeners.size).toBe(0);
  });
  it("onUpdaterProgress：订阅后立即退订 → mock ready 后不残留 updater 监听者", async () => {
    const events = await import("./events");
    const mockShared = await import("../mock/shared");

    const cb = vi.fn();
    const off = events.onUpdaterProgress(cb);
    off();

    const proxy = await import("./proxy");
    await proxy.waitMockReady();
    await Promise.resolve();

    expect(mockShared.updaterListeners.size).toBe(0);
  });
});
