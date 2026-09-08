// bridge is the single seam between the React app and the Go kernel. In the Wails
// shell it calls the bound App methods (window.go.main.App.*) and subscribes to
// the runtime event stream (window.runtime.EventsOn). In a plain browser (`pnpm
// dev` outside the shell) those globals are absent, so it falls back to a mock
// that streams a canned turn through the same contract — letting the whole UI be
// developed and laid out without rebuilding the Go side.
//
// P3 结构版1「巨文件拆分」：绑定面/路由逻辑分置 lib/bridge/ 子模块
// （core/office/memory/cost/model/voice/chat/novel/image/charlib 分域接口 +
// appBindings/mappings/proxy/events/http/drift）。本文件为唯一入口：全部公开名
// 从子模块 re-export，名字逐一同、行为零变化（含浏览器 dev mock 回退）。

export type { AppBindings } from "./bridge/appBindings";
export type { PptxSlideTextEntry } from "./bridge/office";
export type { SubagentTextEvent } from "./bridge/events";
export {
  onEvent,
  onSubagentText,
  onUpdaterProgress,
  onTaskEvent,
  onReady,
} from "./bridge/events";
export {
  BridgeError,
  app,
  workApp,
  playApp,
  sharedApp,
  openExternal,
} from "./bridge/proxy";
export { initBridge } from "./bridge/http";
export type {
  _CheckAppBindingsHasNoStray,
  _CheckAppBindingsCoversAll,
} from "./bridge/drift";
