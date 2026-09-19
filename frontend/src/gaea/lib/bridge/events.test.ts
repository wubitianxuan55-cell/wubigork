// events.ts 桌面路径退订语义（v4.350 回归防线）：onTaskEvent/onUpdaterProgress/
// onReady 此前清理走 EventsOff(channel) 全清——任一消费者卸载注销该通道全部
// 监听者（v4.61 gaea-event 同类事故：5 个 gaea-task 订阅点，关一次任务面板
// 就把运行角标/自动打开/价格源任务的监听一起炸掉）。修复后一律走
// subscribeWailsEvent（按监听者精确注销，绝不摸 EventsOff）。
import { afterEach, describe, expect, it, vi } from "vitest";
import { onTaskEvent, onUpdaterProgress, onReady } from "./events";

type Handler = (payload: unknown) => void;

interface MockRuntime {
  EventsOn: (channel: string, handler: Handler) => () => void;
  EventsOff: (channel: string) => void;
}

function installDesktopRuntime() {
  const handlers = new Map<string, Set<Handler>>();
  const offs: ReturnType<typeof vi.fn>[] = [];
  const eventsOff = vi.fn();
  const runtime: MockRuntime = {
    EventsOn: vi.fn((channel: string, handler: Handler) => {
      if (!handlers.has(channel)) handlers.set(channel, new Set());
      handlers.get(channel)!.add(handler);
      const off = vi.fn(() => {
        handlers.get(channel)?.delete(handler);
      });
      offs.push(off);
      return off;
    }),
    EventsOff: eventsOff,
  };
  (window as unknown as { runtime: unknown }).runtime = runtime;
  (window as unknown as { go?: unknown }).go = { app: { main: {} } };
  return {
    runtime,
    eventsOff,
    emit(channel: string, payload: unknown) {
      for (const h of handlers.get(channel) ?? []) h(payload);
    },
    listenerCount(channel: string) {
      return handlers.get(channel)?.size ?? 0;
    },
  };
}

afterEach(() => {
  delete (window as unknown as { runtime?: unknown }).runtime;
  delete (window as unknown as { go?: unknown }).go;
  vi.restoreAllMocks();
});

describe("onTaskEvent 退订语义（v4.350 全清事故根修）", () => {
  it("双订阅：其一退订只摘自己，另一监听者仍收事件", () => {
    const rt = installDesktopRuntime();
    const a = vi.fn();
    const b = vi.fn();
    const offA = onTaskEvent(a);
    onTaskEvent(b);
    expect(rt.listenerCount("gaea-task")).toBe(2);

    offA();
    expect(rt.listenerCount("gaea-task")).toBe(1);
    expect(rt.eventsOff).not.toHaveBeenCalled();

    rt.emit("gaea-task", { id: "t1", spaceId: "work" });
    expect(a).not.toHaveBeenCalled();
    expect(b).toHaveBeenCalledWith({ id: "t1", spaceId: "work" });
  });

  it("space 过滤语义不变：不匹配空间的任务不回调", () => {
    const rt = installDesktopRuntime();
    const cb = vi.fn();
    onTaskEvent(cb, "work");
    rt.emit("gaea-task", { id: "t1", spaceId: "play" });
    expect(cb).not.toHaveBeenCalled();
    rt.emit("gaea-task", { id: "t2", spaceId: "work" });
    expect(cb).toHaveBeenCalledTimes(1);
  });
});

describe("onUpdaterProgress / onReady 同纪律", () => {
  it("updater:progress 退订不触发 EventsOff 全清", () => {
    const rt = installDesktopRuntime();
    const cb = vi.fn();
    const off = onUpdaterProgress(cb);
    off();
    expect(rt.eventsOff).not.toHaveBeenCalled();
    expect(rt.listenerCount("updater:progress")).toBe(0);
  });

  it("gaea-ready 退订不触发 EventsOff 全清", () => {
    const rt = installDesktopRuntime();
    const cb = vi.fn();
    const off = onReady(cb);
    off();
    expect(rt.eventsOff).not.toHaveBeenCalled();
    expect(rt.listenerCount("gaea-ready")).toBe(0);
  });
});
