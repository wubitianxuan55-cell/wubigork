// events.ts — 事件订阅（onEvent/onSubagentText/onUpdaterProgress/onTaskEvent/
// onReady）：真实运行时走 window.runtime 通道，浏览器 dev 回退 mock 订阅流。
import type { TaskView, UpdateProgress, WireEvent } from "../types";
import { subscribeWailsEvent } from "../wailsEvents";
import { realApp } from "./proxy";
import { mockSubscribe, mockTaskSubscribe, updaterListeners } from "../mock";

// Window 类型由 gaea 的 src/types/wails.d.ts 统一声明（go.app.App + runtime）。
// gaeaW 在此不重复声明，避免覆盖 gaea 的 runtime（EventsOff）类型。

// onEvent subscribes to the agent's typed event stream; returns an unsubscribe.
// 清理函数只摘除本监听者（v4.62.2）：此前用 EventsOff(channel) 全清——
// SubagentThread 卸载时把主对话 store 的订阅连带炸掉，对话标签页实时输出
// 全灭（轮询类面板正常），见 lib/wailsEvents.ts 的事故注记。
export function onEvent(cb: (e: WireEvent) => void): () => void {
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    return subscribeWailsEvent(window.runtime, EVENT_CHANNEL, (payload) => cb(payload as WireEvent));
  }
  return mockSubscribe(cb);
}

// Must match desktop/app.go's eventChannel constant.
const EVENT_CHANNEL = "gaea-event";

/** 子代理流式增量事件（专用通道 payload，v4.62.1 与 gaea-event 分道）。 */
export interface SubagentTextEvent {
  kind: "subagent_text";
  text: string;
  subagentRef?: string;
  parentId?: string;
}

// 子代理流式增量走专用 wails 通道（无 seq）：gaea-event 的契约是「seq 与磁盘
// 账本 1:1，丢件可 resync 补拉」（v4.26 防线）；本通道是有损无妨的装饰性实时
// 流，绝不可上 gaea-event 消费 seq（v4.62.0 曾因此把对话窗过程可见性打断）。
// mock 场景复用 mockSubscribe：mock 可用 kind=subagent_text 的事件演示流式。
const SUBAGENT_TEXT_CHANNEL = "gaea-subagent-text";

// onSubagentText subscribes to the subagent streaming-delta channel.
export function onSubagentText(cb: (e: SubagentTextEvent) => void): () => void {
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    return subscribeWailsEvent(
      window.runtime,
      SUBAGENT_TEXT_CHANNEL,
      (payload) => cb(payload as SubagentTextEvent),
    );
  }
  return mockSubscribe(cb as unknown as (e: WireEvent) => void);
}


// channel from the agent stream); returns an unsubscribe.
export function onUpdaterProgress(cb: (p: UpdateProgress) => void): () => void {
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    window.runtime.EventsOn("updater:progress", (p) => cb(p as UpdateProgress));
    return () => window.runtime?.EventsOff?.("updater:progress");
  }
  updaterListeners.add(cb);
  return () => {
    updaterListeners.delete(cb);
  };
}

// onTaskEvent subscribes to the task scheduler's event stream (gaea-task):
// every task status/progress change pushes the latest TaskView. space 非空时在
// 订阅层按 payload.spaceId 过滤（v4.5.1a 红线补课：S2.1 事件空间过滤推广到
// 任务事件——work 消费点传 "work"，play 任务事件不打扰工位 UI）；缺省
// spaceId（旧任务/旧后端）按 work 兼容放行，与 isWorkSpaceTask 同语义。
// Returns an unsubscribe. Falls back to the mock stream outside the Wails shell.
export function onTaskEvent(cb: (t: TaskView) => void, space?: string): () => void {
  const handler = (t: TaskView) => {
    if (space && t.spaceId && t.spaceId !== space) return;
    cb(t);
  };
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    window.runtime.EventsOn("gaea-task", (payload) => handler(payload as TaskView));
    return () => window.runtime?.EventsOff?.("gaea-task");
  }
  return mockTaskSubscribe(handler);
}

// onReady subscribes to the agent:ready event fired when boot.Build completes.
// The frontend re-fetches Meta/Context/History when this lands.
export function onReady(cb: () => void): () => void {
  if (realApp() && typeof window !== "undefined" && window.runtime) {
    window.runtime.EventsOn("gaea-ready", () => cb());
    return () => window.runtime?.EventsOff?.("gaea-ready");
  }
  // In dev mock, fire immediately since there's no real boot sequence.
  cb();
  return () => {};
}

