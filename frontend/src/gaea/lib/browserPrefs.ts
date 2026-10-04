// browserPrefs — 浏览器观察窗（v4.28 A2）的轻量用户偏好。
//
// 「自动弹出」（gaea 差异化，对标 Playwright Trace Viewer 是事后回放、这里
// 是执行中跟随）：会话轨迹里出现新 browser_* 工具记录时，App 自动把右侧面板
// 切到「浏览器」tab——用户不必手动找。默认开；可关（关后只记数据不抢焦点）。
// 开关 UI 在 BrowserPanel 头部（自动弹出胶囊），持久化键 gaea.browserAutoOpen。
//
// 模式对齐 lib/subagentPrefs（分工面板「新子代理自动展开」同款交互）：
// 2026-10-04 起读写样板收敛到 lib/booleanPref 工厂（四域单一方言），本文件
// 只保留域语义（键、默认开）与 App 接线入口。

import { createBooleanPref } from "./booleanPref";

const pref = createBooleanPref("gaea.browserAutoOpen", true);

/** 读取「自动弹出」偏好；未设置/损坏/无 window 时默认开。 */
export function loadBrowserAutoOpen(): boolean {
  return pref.load();
}

/** 持久化「自动弹出」偏好（"1"/"0"，可读可手改）。 */
export function saveBrowserAutoOpen(enabled: boolean): void {
  pref.save(enabled);
}

/**
 * shouldAutoOpenBrowser — App 接线专用入口（v4.28 A2）：当会话轨迹里出现
 * 新 browser_* 工具记录时由 App 调用，返回是否把右侧面板切到「浏览器」tab。
 * 仅读偏好，不做其它判定（轨迹检测与切换时机在 App，由主代理接线）。
 */
export function shouldAutoOpenBrowser(): boolean {
  return loadBrowserAutoOpen();
}
