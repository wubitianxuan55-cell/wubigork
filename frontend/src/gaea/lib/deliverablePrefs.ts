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
// 模式对齐 lib/browserPrefs（同源先例 lib/subagentPrefs）：localStorage 直读
// + try/catch 静默降级（隐私模式/存储禁用不影响面板主功能）。默认值语义与
// browserPrefs 相反：默认关 → 只有显式开启值（"1"/"true"）视为开，
// 其余（未设置/"0"/"false"/损坏垃圾值）一律回落关。

const STORAGE_KEY = "gaea.deliverableAutoOpen";

/** 读取「自动弹出」偏好；未设置/损坏/无 window 时默认关。 */
export function loadDeliverableAutoOpen(): boolean {
  if (typeof window === "undefined") return false;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (raw === null) return false;
    // 默认关：只认显式开启值，其余（"0"/"false"/损坏垃圾值）一律回落关。
    return raw === "1" || raw === "true";
  } catch {
    return false;
  }
}

/** 持久化「自动弹出」偏好（"1"/"0"，可读可手改）。 */
export function saveDeliverableAutoOpen(enabled: boolean): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, enabled ? "1" : "0");
  } catch {
    /* 存储失败静默降级：开关是增强项，不阻塞面板 */
  }
}

/**
 * shouldAutoOpenDeliverables — App 接线专用入口（v4.32 线B）：当本会话出现
 * 新产物（App 侧 diff 得出）时由 App 调用，返回是否把右侧面板切到「产物」tab。
 * 仅读偏好，不做其它判定（新产物检测与切换时机在 App，由主代理接线）。
 */
export function shouldAutoOpenDeliverables(): boolean {
  return loadDeliverableAutoOpen();
}
