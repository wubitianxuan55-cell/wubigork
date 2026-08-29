// workbenchStorage.ts — S2.2 工作台 localStorage 空间分键
// （docs/gaea-space-shell-design.md §4.6 补：store/key 空间前缀迁移）
//
// 办公工作台 = 工位空间主体（manifest.space=work），其本地状态键一律加
// `work` 空间段：旧 key `gaea.<x>` → 新 key `gaea.work.<x>`。
// 读路径：新 key 优先，回退旧 key（旧值只读迁移，不主动改写——避免破坏
// 既有数据）；写路径只写新 key。
// 未来乐园工作台接入时用 `play` 段对称扩展（S2.3 bridge 分面统一）。

/** 旧 key → 空间分键（gaea.<x> → gaea.work.<x>） */
export function workbenchKey(legacyKey: string): string {
  return legacyKey.replace(/^gaea\./, "gaea.work.");
}

/** 读取：优先空间分键，回退旧 key（兼容迁移） */
export function readWorkbenchValue(legacyKey: string): string | null {
  try {
    const v = localStorage.getItem(workbenchKey(legacyKey));
    if (v !== null) return v;
    return localStorage.getItem(legacyKey);
  } catch {
    return null;
  }
}

/** 写入：只写空间分键 */
export function writeWorkbenchValue(legacyKey: string, value: string): void {
  try {
    localStorage.setItem(workbenchKey(legacyKey), value);
  } catch {
    /* 隐私模式/配额：静默失败 */
  }
}
