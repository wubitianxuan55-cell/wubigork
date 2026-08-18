// 最近使用文件（localStorage 单源）。
//
// Why: @ 引用菜单（useComposerMenus RECENT_AT_KEY）与「最近文件」快捷区
// （调研 2026-08-16 P0-3）都要读写同一份最近文件列表；此前这份状态
// 内联在 useComposerMenus 里，无法被预览面板/工作区面板复用。这里收敛为
// 独立模块，两处消费同一 key 与同一去重置顶逻辑。
//
// How to apply: `import { loadRecentFiles, recordRecentFile } from "../lib/recentFiles"`。
// 新增写入点只需调用 recordRecentFile(path)，快捷区自动可见。

import type { AtEntry } from "./types";

const RECENT_FILES_KEY = "gaea.atRecentFiles";
const MAX_RECENT_FILES = 20;

/** 读取最近使用文件（按时间倒序；损坏/空返回 []）。 */
export function loadRecentFiles(): AtEntry[] {
  try {
    const raw = JSON.parse(localStorage.getItem(RECENT_FILES_KEY) || "[]");
    if (!Array.isArray(raw)) return [];
    return raw.filter((e): e is AtEntry =>
      !!e && typeof e.path === "string" && typeof e.name === "string" && typeof e.isDir === "boolean",
    ).slice(0, MAX_RECENT_FILES);
  } catch {
    return [];
  }
}

/** 记录一次文件引用/预览：去重置顶、限长、落盘（幂等）。 */
export function recordRecentFile(path: string, name?: string): void {
  if (!path) return;
  const fileName = name ?? path.split(/[/\\]/).pop() ?? path;
  const next = [
    { path, name: fileName, isDir: false },
    ...loadRecentFiles().filter((e) => e.path !== path && !e.isDir),
  ].slice(0, MAX_RECENT_FILES);
  try {
    localStorage.setItem(RECENT_FILES_KEY, JSON.stringify(next));
  } catch {
    // 配额/隐私模式等：静默失败，不影响主流程
  }
}

/** 测试辅助：清空最近文件（vitest 隔离用例间状态）。 */
export function clearRecentFilesForTest(): void {
  try { localStorage.removeItem(RECENT_FILES_KEY); } catch { /* ignore */ }
}
