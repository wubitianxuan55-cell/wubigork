// boardPorts — 文学板块的读口实现（2026-11-26 同步审计）：把本树自己的注入点槽位
// 与预览库交给被共用的办公组件（ToolCard / FileLinkText / Composer 附件条）。
//
// 壳层 keepAlive 下办公与文学两个 App 同挂：办公组件默认读的是办公那份模块单例
// 与预览库——不接这条缝，文学的子代理卡读不到自己的活动/打开器（无实时动态、
// 点击不开会话）、卡内文件链接与输入框附件条落到办公那侧的预览库（界面无反应）。
// 缝的定义见 gaea/components/boardPorts.ts。
import type { BoardPorts } from "../../gaea/components/boardPorts";
import {
  fireTaskCardAmbiguity,
  getTaskCardActivity,
  getTaskCardAmbiguity,
  getTaskCardOpenTarget,
  hasTaskCardAmbiguityHandler,
  openTaskCardSession,
} from "./taskActivity";
import { openPaneFileOrPreview } from "./paneFileOpen";
import { usePreviewStore } from "./store";

export const wenxueBoardPorts: BoardPorts = {
  activity: getTaskCardActivity,
  openTarget: getTaskCardOpenTarget,
  openSession: openTaskCardSession,
  ambiguity: getTaskCardAmbiguity,
  hasAmbiguityHandler: hasTaskCardAmbiguityHandler,
  fireAmbiguity: fireTaskCardAmbiguity,
  openPaneFile: openPaneFileOrPreview,
  openFilePreview: (path) => usePreviewStore.getState().openFilePreview(path),
};
