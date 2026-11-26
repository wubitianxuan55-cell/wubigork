// boardPorts — 被多板块共用的小组件所依赖的「板块作用域」读口缝（2026-11-26 同步审计）。
//
// 背景：壳层对访问过的板块 keepAlive，办公与文学两个板块的 App 可能同时挂载。
// 共享组件（ToolCard / FileLinkText / Composer 附件条）默认读的是办公那侧的
// 模块单例与预览库：
//   - 任务卡注入点（lib/taskActivity）与文件打开入口（lib/paneFileOpen）是模块级
//     单例，两个板块各自注册的打开器会互相覆盖（或另一板块的注册根本读不到）；
//   - 预览库（lib/store 的 usePreviewStore）两板块各一份是有意隔离，共享组件
//     直接写办公那份时，文学界面里点了没反应。
//
// 缝：板块壳在自己的子树里经 Context 提供自己那份读口；未提供（办公）= 原模块
// 单例 / 办公预览库，行为逐字节不变。
import { createContext, useContext } from "react";
import type { TaskCardActivity } from "../lib/taskActivity";

export interface BoardPorts {
  /** 任务卡活动预览（活动 provider 槽位）；查不到返回 undefined。 */
  activity(ref: string, args?: unknown): TaskCardActivity | undefined;
  /** 卡片可点的目标 ref；空串 = 不可点。 */
  openTarget(ref: string, args?: unknown): string;
  /** 执行跳转；返回是否已派发。 */
  openSession(ref: string): boolean;
  /** 该卡是否处于多候选歧义态。 */
  ambiguity(ref: string, args?: unknown): boolean;
  /** 是否已注入歧义点击回调。 */
  hasAmbiguityHandler(): boolean;
  /** 派发歧义点击；返回是否已派发。 */
  fireAmbiguity(args?: unknown): boolean;
  /** 正文/卡片「点文件」的统一打开入口（paneFileOpen 槽位）。 */
  openPaneFile(rel: string): void;
  /** 附件/文件「打开预览」入口（板块自己的预览库）。 */
  openFilePreview(path: string): void;
}

export const BoardPortsContext = createContext<BoardPorts | null>(null);

/** 板块壳提供的读口；null = 未提供（走办公默认）。 */
export function useBoardPorts(): BoardPorts | null {
  return useContext(BoardPortsContext);
}
