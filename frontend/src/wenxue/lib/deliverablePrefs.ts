// deliverablePrefs — 会话产物面板（v4.32 线B）「自动弹出」的轻量用户偏好。
//
// 收 v4.30 欠账（「产物自动弹 tab（激进版 Auto-open）暂不做可加偏好」）：
// 会话里出现新产物时，App 自动把右侧面板切到「产物」tab。做成可关偏好，
// **默认关**（激进版 Auto-open opt-in）。对标 browserAutoOpen（v4.28 A2）
// 默认开的差异：产物更新比 browser_* 操作更频繁（一轮多文件、反复覆写），
// 每次都抢右栏焦点代价更高，故让用户显式开启；浏览器跟随是低频事件，
// 默认开无打扰感。
//
// 分工边界：本 lib 只管偏好读写，**不做新产物检测**——触发接线在 App
// （新产物 diff effect 里调 shouldAutoOpenDeliverables() 决定是否切 tab，
// 由主代理接线）。开关 UI 在 DeliverablesPanel 头部「自动弹出」胶囊，
// 持久化键 gaea.deliverableAutoOpen。
//
// 2026-10-04 起读写样板收敛到 lib/booleanPref 工厂（四域单一方言），本文件
// 只保留域语义（键、默认关）与 App 接线入口。

import { createBooleanPref } from "./booleanPref";

const pref = createBooleanPref("gaea.deliverableAutoOpen", false);

/** 读取「自动弹出」偏好；未设置/损坏/无 window 时默认关。 */
export function loadDeliverableAutoOpen(): boolean {
  return pref.load();
}

/** 持久化「自动弹出」偏好（"1"/"0"，可读可手改）。 */
export function saveDeliverableAutoOpen(enabled: boolean): void {
  pref.save(enabled);
}

/**
 * shouldAutoOpenDeliverables — App 接线专用入口（v4.32 线B）：当本会话出现
 * 新产物（App 侧 diff 得出）时由 App 调用，返回是否把右侧面板切到「产物」tab。
 * 仅读偏好，不做其它判定（新产物检测与切换时机在 App，由主代理接线）。
 */
export function shouldAutoOpenDeliverables(): boolean {
  return loadDeliverableAutoOpen();
}
