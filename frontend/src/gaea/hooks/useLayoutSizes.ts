// 布局常量 + clamp 函数 + localStorage 宽度持久化
// 从 App.tsx 提取，纯函数模块

import { loadLayoutSize, saveLayoutSize } from "../lib/layoutPreferences";

export const SIDEBAR_COLLAPSED_WIDTH = 68;
export const SIDEBAR_DEFAULT_WIDTH = 264;
export const SIDEBAR_MIN_WIDTH = 228;
export const SIDEBAR_MAX_WIDTH = 420;

const SIDEBAR_COLLAPSED_KEY = "gaea.sidebar.collapsed";

export function clampSidebarWidth(width: number): number {
  return Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, Math.round(width)));
}

export function loadSidebarCollapsed(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === "1";
  } catch {
    return false;
  }
}

export function saveSidebarCollapsed(collapsed: boolean): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(SIDEBAR_COLLAPSED_KEY, collapsed ? "1" : "0");
  } catch {
    /* ignore storage failures */
  }
}

export function loadSidebarWidth(): number {
  return loadLayoutSize("sidebarWidth", SIDEBAR_DEFAULT_WIDTH, clampSidebarWidth);
}

export function saveSidebarWidth(width: number): void {
  saveLayoutSize("sidebarWidth", width, clampSidebarWidth);
}
