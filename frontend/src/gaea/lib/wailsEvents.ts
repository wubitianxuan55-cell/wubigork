// wailsEvents.ts — wails 事件通道订阅的唯一入口（按监听者精确注销）。
//
// 事故教训（v4.61 引入、v4.62.2 修复）：wails 的 EventsOff(channel) 会注销
// 该通道上的**全部**监听者。v4.61 给 SubagentThread 加了 gaea-event 订阅后，
// 它卸载时的 EventsOff 把主对话 store 的订阅连带炸掉——对话标签页从此收不
// 到任何实时事件（轮询类面板正常，最终答案靠 reconcile RPC 兜底仍会出现），
// 直到重开应用。修复：EventsOn 本身返回「只摘除自己」的注销函数（wails
// v2.13 desktop runtime 的 EventsOnMultiple 语义，见其 events.js），一律用
// 它；前端代码中禁止调用 EventsOff（会把别人的监听一起炸掉）。
//
// 任何新的事件订阅都必须经由本模块，不得直接摸 window.runtime。

export interface WailsEventRuntime {
  // 返回值对齐 wails v2.13 desktop runtime：EventsOn 返回「注销本监听者」
  // 的函数；旧运行时可能返回 void（此时只能容忍监听器残留，绝不能
  // EventsOff 全清——那正是本次事故的根源）。
  EventsOn(eventName: string, handler: (payload: unknown) => void): unknown;
}

/**
 * 在 channel 上注册一个监听者，返回「只注销自己」的清理函数。
 * 同一通道上其他监听者（如主对话 store 与 SubagentThread 共用 gaea-event）
 * 不受任何一次清理影响。
 */
export function subscribeWailsEvent(
  rt: WailsEventRuntime,
  channel: string,
  handler: (payload: unknown) => void,
): () => void {
  const off = rt.EventsOn(channel, handler) as unknown;
  return typeof off === "function" ? (off as () => void) : () => {};
}
