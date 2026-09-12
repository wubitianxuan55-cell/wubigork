// sin/sinPanelState.ts — 右栏创作面板的开合/页签本地记忆（v4.263）。
//
// localStorage 键用原罪自有命名空间 gaea.sin.*（与数据面硬隔离同口径，
// 不借用办公 workbench 空间键）。读写失败静默（隐私模式/配额）。

const PANEL_OPEN = 'gaea.sin.panelOpen'
const PANEL_TAB = 'gaea.sin.panelTab'

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
