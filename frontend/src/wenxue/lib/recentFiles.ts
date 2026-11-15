// 最近使用文件（localStorage 单源）。
//
// Why: @ 引用菜单（useComposerMenus RECENT_AT_KEY）与「最近文件」快捷区
// （调研 2026-08-16 P0-3）都要读写同一份最近文件列表；此前这份状态
// 内联在 useComposerMenus 里，无法被预览面板/工作区面板复用。这里收敛为
// 独立模块，两处消费同一 key 与同一去重置顶逻辑。
//
// How to apply: `import { loadRecentFiles, recordRecentFile } from "../lib/recentFiles"`。
// 新增写入点只需调用 recordRecentFile(path)，快捷区自动可见。
//
// v4.349 实时化：写入即广播 `RECENT_FILES_EVENT` + 提供 subscribeRecentFiles，
// 消费方用 useSyncExternalStore 订阅；loadRecentFiles 带解析缓存（快照引用稳定，
// 满足 useSyncExternalStore 的 getSnapshot 缓存要求）。此前消费方只在挂载时读一次，
// 而 v4.346 起首页等页常驻保活（keepAlive），导致「在办公打开文档 → 回首页，
// 最近文档仍是首次挂载快照」——同窗口 storage 事件不触发，故自带事件通道。

import type { AtEntry } from "./types";

const RECENT_FILES_KEY = "gaea.atRecentFiles";
const MAX_RECENT_FILES = 20;
/** 变化广播事件（同窗口写入即通知；storage 事件只跨窗口，覆盖不到本场景） */
const RECENT_FILES_EVENT = "gaea:recent-files";

/** 解析缓存：同一份快照引用稳定（useSyncExternalStore 要求）；写入/清空即失效。 */
let cache: AtEntry[] | null = null;

function readFromStorage(): AtEntry[] {
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

/** 读取最近使用文件（按时间倒序；损坏/空返回 []）。 */
export function loadRecentFiles(): AtEntry[] {
  if (!cache) cache = readFromStorage();
  return cache;
}

/** 订阅最近文件变化（返回取消订阅函数）。 */
export function subscribeRecentFiles(cb: () => void): () => void {
  if (typeof window === "undefined") return () => {};
  window.addEventListener(RECENT_FILES_EVENT, cb);
  return () => window.removeEventListener(RECENT_FILES_EVENT, cb);
}

function emit(): void {
  try { window.dispatchEvent(new Event(RECENT_FILES_EVENT)); } catch { /* 非浏览器环境忽略 */ }
}

/** 记录一次文件引用/预览：去重置顶、限长、落盘（幂等）+ 广播。 */
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
    // 配额/隐私模式等：静默失败，不影响主流程（存储与展示均不变）
    return;
  }
  cache = next;
  emit();
}

/** 测试辅助：清空最近文件（vitest 隔离用例间状态）。 */
export function clearRecentFilesForTest(): void {
  try { localStorage.removeItem(RECENT_FILES_KEY); } catch { /* ignore */ }
  cache = null;
}
