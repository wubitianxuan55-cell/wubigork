// sin/sinPanelState.ts — 右栏创作面板的开合/页签本地记忆（v4.263）。
//
// localStorage 键用原罪自有命名空间 gaea.sin.*（与数据面硬隔离同口径，
// 不借用办公 workbench 空间键）。读写失败静默（隐私模式/配额）。

const PANEL_OPEN = 'gaea.sin.panelOpen'
const PANEL_TAB = 'gaea.sin.panelTab'
const PANEL_WIDTH = 'gaea.sin.panelWidth'

/** 面板宽度档位：默认=老版面 268；上限同时受视口收敛（保住故事流可读）。 */
export const SIN_PANEL_DEFAULT_WIDTH = 268
export const SIN_PANEL_MIN_WIDTH = 240
export const SIN_PANEL_MAX_WIDTH = 640

export function clampSinPanelWidth(value: number): number {
  if (!Number.isFinite(value)) return SIN_PANEL_DEFAULT_WIDTH
  const viewportMax = typeof window === 'undefined'
    ? SIN_PANEL_MAX_WIDTH
    : window.innerWidth - 520 // 左故事架 232 + 故事流最低可读余量
  const max = Math.max(SIN_PANEL_MIN_WIDTH + 40, Math.min(SIN_PANEL_MAX_WIDTH, viewportMax))
  return Math.min(max, Math.max(SIN_PANEL_MIN_WIDTH, Math.round(value)))
}

export type SinSideTabId = 'cast' | 'outline' | 'notes' | 'gallery'

function readSideValue(key: string): string {
  try {
    return localStorage.getItem(key) ?? ''
  } catch {
    return ''
  }
}

function writeSideValue(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* 隐私模式/配额：记忆失效可接受 */
  }
}

/** 右栏默认开合：宽窗默认开（沿用旧版面口径），窄窗默认收。 */
export function defaultSinPanelOpen(): boolean {
  if (typeof window === 'undefined') return true
  return window.innerWidth > 1180
}

export function readSinPanelOpen(): boolean {
  const v = readSideValue(PANEL_OPEN)
  if (v === '0') return false
  if (v === '1') return true
  return defaultSinPanelOpen()
}

export function writeSinPanelOpen(open: boolean): void {
  writeSideValue(PANEL_OPEN, open ? '1' : '0')
}

export function readSinPanelTab(): SinSideTabId {
  const v = readSideValue(PANEL_TAB)
  return v === 'outline' || v === 'notes' || v === 'gallery' ? v : 'cast'
}

export function writeSinPanelTab(tab: SinSideTabId): void {
  writeSideValue(PANEL_TAB, tab)
}

export function readSinPanelWidth(): number {
  const raw = Number(readSideValue(PANEL_WIDTH))
  return clampSinPanelWidth(Number.isFinite(raw) && raw > 0 ? raw : SIN_PANEL_DEFAULT_WIDTH)
}

export function writeSinPanelWidth(width: number): void {
  writeSideValue(PANEL_WIDTH, String(clampSinPanelWidth(width)))
}
