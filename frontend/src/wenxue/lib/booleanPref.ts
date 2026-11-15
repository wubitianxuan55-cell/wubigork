// booleanPref — 轻量布尔 localStorage 偏好的单一方言（2026-10-04 第三轮审计
// §3.2 收敛：browserPrefs/subagentPrefs/tasksPrefs/deliverablePrefs 四份逐字
// 同形的读写样板收敛于此，拍板史与语义注释仍留在各域文件）。
//
// 方言（四域既有口径，语义冻结）：
//   - 未设置（null）/ 无 window / 读写抛错（隐私模式、存储禁用）→ 回落默认值；
//   - 只认「与默认相反」的显式值翻转向：默认开认 "0"/"false"，默认关认
//     "1"/"true"；其余（含损坏垃圾值）一律回落默认；
//   - 持久化恒写 "1"/"0"（可读可手改）。
export interface BooleanPref {
  load: () => boolean
  save: (value: boolean) => void
}

export function createBooleanPref(storageKey: string, defaultValue: boolean): BooleanPref {
  return {
    load(): boolean {
      if (typeof window === "undefined") return defaultValue;
      try {
        const raw = window.localStorage.getItem(storageKey);
        if (raw === null) return defaultValue;
        if (defaultValue) {
          // 默认开：只认显式关闭值；其余（"1"/"true"/损坏垃圾值）一律回落默认开。
          return !(raw === "0" || raw === "false");
        }
        // 默认关：只认显式开启值；其余（"0"/"false"/损坏垃圾值）一律回落默认关。
        return raw === "1" || raw === "true";
      } catch {
        return defaultValue;
      }
    },
    save(value: boolean): void {
      if (typeof window === "undefined") return;
      try {
        window.localStorage.setItem(storageKey, value ? "1" : "0");
      } catch {
        /* 存储失败静默降级：开关是增强项，不阻塞面板主功能 */
      }
    },
  };
}
