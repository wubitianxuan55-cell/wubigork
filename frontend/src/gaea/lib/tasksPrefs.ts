// tasksPrefs — v4.63「新任务自动切任务视图」的轻量用户偏好。
//
// 当前会话出现新子代理/本地模型工具运行时，App 自动把右栏切到「任务」视图
// （对标 dsh better-sidebar 的 0→N 触发 + 500ms 去抖重臂）。默认开；可关
// （关后只记角标语义，不抢右栏焦点）。
//
// 存储键 gaea.tasks.autoOpenSubagent 由 v4.63 落地时即存在于 App.tsx
// （触发时机处 inline localStorage 直读，只认 "0" 为关）。此前无独立 prefs
// 模块、也无设置中心入口；本模块为设置中心（办公分组「办公工作台偏好」卡）
// 补齐 load/save 入口，写值与 App 侧读取约定兼容（"1"/"0"，App 只认 "0" 为关）。
//
// 模式对齐 lib/subagentPrefs：localStorage 直读 + try/catch 静默降级
// （隐私模式/存储禁用不影响主功能），读不到或值损坏时回落默认开；
// 只有显式关闭值（"0"/"false"）视为关，其余（"1"/"true"/垃圾值）一律开。

const STORAGE_KEY = "gaea.tasks.autoOpenSubagent";

/** 读取「新任务自动切任务视图」偏好；未设置/损坏/无 window 时默认开。 */
export function loadTasksAutoOpenSubagent(): boolean {
  if (typeof window === "undefined") return true;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (raw === null) return true;
    // 与 App.tsx 触发端约定一致：只认显式关闭值，其余回落默认开。
    return !(raw === "0" || raw === "false");
  } catch {
    return true;
  }
}

/** 持久化「新任务自动切任务视图」偏好（"1"/"0"，可读可手改）。 */
export function saveTasksAutoOpenSubagent(enabled: boolean): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, enabled ? "1" : "0");
  } catch {
    /* 存储失败静默降级：开关是增强项，不阻塞主功能 */
  }
}
