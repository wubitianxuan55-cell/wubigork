// browserPrefs — 浏览器观察窗（v4.28 A2）的轻量用户偏好。
//
// 「自动弹出」（gaea 差异化，对标 Playwright Trace Viewer 是事后回放、这里
// 是执行中跟随）：会话轨迹里出现新 browser_* 工具记录时，App 自动把右侧面板
// 切到「浏览器」tab——用户不必手动找。默认开；可关（关后只记数据不抢焦点）。
// 开关 UI 在 BrowserPanel 头部（自动弹出胶囊），持久化键 gaea.browserAutoOpen。
//
// 模式对齐 lib/subagentPrefs（分工面板「新子代理自动展开」同款交互）：
// localStorage 直读 + try/catch 静默降级（隐私模式/存储禁用不影响面板主功能），
// 读不到或值损坏时回落默认开；只有显式关闭值（"0"/"false"）视为关，
// 其余（"1"/"true"/垃圾值）一律回落默认开。

const STORAGE_KEY = "gaea.browserAutoOpen";

/** 读取「自动弹出」偏好；未设置/损坏/无 window 时默认开。 */
export function loadBrowserAutoOpen(): boolean {
  if (typeof window === "undefined") return true;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (raw === null) return true;
    // 只认显式关闭值；其余（"1"/"true"/损坏垃圾值）一律回落默认开。
    return !(raw === "0" || raw === "false");
  } catch {
    return true;
  }
}

/** 持久化「自动弹出」偏好（"1"/"0"，可读可手改）。 */
export function saveBrowserAutoOpen(enabled: boolean): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, enabled ? "1" : "0");
  } catch {
    /* 存储失败静默降级：开关是增强项，不阻塞面板 */
  }
}

/**
 * shouldAutoOpenBrowser — App 接线专用入口（v4.28 A2）：当会话轨迹里出现
 * 新 browser_* 工具记录时由 App 调用，返回是否把右侧面板切到「浏览器」tab。
 * 仅读偏好，不做其它判定（轨迹检测与切换时机在 App，由主代理接线）。
 */
export function shouldAutoOpenBrowser(): boolean {
  return loadBrowserAutoOpen();
}
