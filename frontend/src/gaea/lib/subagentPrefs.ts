// subagentPrefs — 分工面板（v4.24 A1「子代理工作台」）的轻量用户偏好。
//
// 「新子代理自动展开」（对标 dsh-better-sidebar 任务页：新子代理出现自动展开
// 侧栏到本 tab，可关）：默认开——派发子代理时面板/侧栏自动亮出拓扑；用户可关，
// 关闭后出现新子代理只更新数据不抢焦点（不触发 onSubagentStarted 联动）。
//
// 模式对齐 lib/layoutPreferences：localStorage 直读 + try/catch 静默降级
// （隐私模式/存储禁用时不影响面板主功能），读不到或值损坏时回落默认值。

const STORAGE_KEY = "gaea.subagentAutoOpen";

/** 读取「新子代理自动展开」偏好；未设置/损坏/无 window 时默认开。 */
export function loadSubagentAutoOpen(): boolean {
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

/** 持久化「新子代理自动展开」偏好（"1"/"0"，可读可手改）。 */
export function saveSubagentAutoOpen(enabled: boolean): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, enabled ? "1" : "0");
  } catch {
    /* 存储失败静默降级：开关是增强项，不阻塞面板 */
  }
}
