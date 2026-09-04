// wailsEvents.test.ts — 事件通道订阅语义回归（v4.62.2 对话标签页失聪事故钉子）。
//
// 事故：EventsOff(channel) 注销该通道全部监听者——SubagentThread 卸载时把
// 主对话 store 的 gaea-event 订阅连带炸掉，对话窗实时输出全灭。本测试锁死：
// subscribeWailsEvent 的清理只摘除自己，同通道其他监听者不受影响；且绝不
// 调用 EventsOff。

import { describe, expect, it, vi } from "vitest";
import { subscribeWailsEvent, type WailsEventRuntime } from "./wailsEvents";

// 仿 wails v2.13 desktop runtime 语义的假实现：EventsOn 返回按监听者注销
// 函数；EventsOff（若存在）按通道全清——并记录调用以便断言「从未发生」。
function fakeRuntime() {
  const listeners = new Map<string, Set<(payload: unknown) => void>>();
  const rt: WailsEventRuntime & { EventsOff: (channel: string) => void; dispatch: (channel: string, payload: unknown) => void } = {
    EventsOn(channel, handler) {
      listeners.set(channel, listeners.get(channel) ?? new Set());
      listeners.get(channel)!.add(handler);
      return () => {
        listeners.get(channel)?.delete(handler);
      };
    },
    EventsOff(channel) {
      listeners.delete(channel);
    },
    dispatch(channel, payload) {
      listeners.get(channel)?.forEach((h) => h(payload));
    },
  };
  return rt;
}

describe("subscribeWailsEvent 按监听者精确注销", () => {
  it("同通道两个订阅者，一个注销后另一个仍收到事件（主对话不被连带炸掉）", () => {
    const rt = fakeRuntime();
    const seen: string[] = [];
    const offA = subscribeWailsEvent(rt, "gaea-event", (p) => seen.push(`A:${String((p as { n: number }).n)}`));
    const offB = subscribeWailsEvent(rt, "gaea-event", (p) => seen.push(`B:${String((p as { n: number }).n)}`));

    rt.dispatch("gaea-event", { n: 1 });
    expect(seen).toEqual(["A:1", "B:1"]);

    // SubagentThread 卸载 → 只摘 B；A（主对话）必须存活
    offB();
    rt.dispatch("gaea-event", { n: 2 });
    expect(seen).toEqual(["A:1", "B:1", "A:2"]);

    offA();
    rt.dispatch("gaea-event", { n: 3 });
    expect(seen).toEqual(["A:1", "B:1", "A:2"]);
  });

  it("绝不调用 EventsOff（全清是事故根源）", () => {
    const rt = fakeRuntime();
    const spy = vi.spyOn(rt, "EventsOff");
    const off1 = subscribeWailsEvent(rt, "gaea-subagent-text", () => {});
    const off2 = subscribeWailsEvent(rt, "gaea-subagent-text", () => {});
    off1();
    off2();
    expect(spy).not.toHaveBeenCalled();
  });

  it("旧运行时 EventsOn 返回 void 时：清理为安全 no-op（宁残留不全清）", () => {
    const legacy: WailsEventRuntime = { EventsOn: () => undefined };
    const off = subscribeWailsEvent(legacy, "gaea-event", () => {});
    expect(() => off()).not.toThrow();
  });
});
