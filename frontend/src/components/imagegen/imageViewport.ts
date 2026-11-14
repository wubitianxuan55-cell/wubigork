/**
 * imageViewport.ts — 识图工作台画布视图数学（缩放/平移/刻度/像素探针）
 *
 * 纯函数，零 DOM：组件只负责把返回值落到 transform 与刻度条上，方便单测钉口径。
 * 口径与主流图像查看器一致：
 *  - 缩放钳在 [5%, 1600%]；
 *  - 滚轮缩放**以光标为锚点**（光标下的图像点保持不动）；
 *  - 刻度步长取「≥64px 的最小 1/2/5 档」，避免放大后数字过密、缩小后没有刻度。
 */

/** 视图状态：zoom 为倍数，x/y 为图像左上角在视口内的像素偏移 */
export interface ViewState {
  zoom: number
  x: number
  y: number
}

export const ZOOM_MIN = 0.05
export const ZOOM_MAX = 16
/** 刻度标签最小间距（px）：小于它就换更粗的步长 */
export const TICK_MIN_GAP = 64
const TICK_STEPS = [1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000] as const

export function clampZoom(zoom: number): number {
  if (!Number.isFinite(zoom)) return 1
  return Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, zoom))
}

/** 适合窗口：等比缩到视口内（留 pad 边距），不放大超过 100% */
export function fitZoom(imgW: number, imgH: number, boxW: number, boxH: number, pad = 24): number {
  if (imgW <= 0 || imgH <= 0 || boxW <= 0 || boxH <= 0) return 1
  const w = Math.max(1, boxW - pad * 2)
  const h = Math.max(1, boxH - pad * 2)
  return clampZoom(Math.min(1, w / imgW, h / imgH))
}

/** 居中摆放（fit 后调用）：把图像放到视口正中 */
export function centerView(imgW: number, imgH: number, boxW: number, boxH: number, zoom: number): ViewState {
  return {
    zoom,
    x: Math.round((boxW - imgW * zoom) / 2),
    y: Math.round((boxH - imgH * zoom) / 2),
  }
}

/**
 * 以视口内某点为锚缩放：锚点下的图像坐标在缩放前后保持同一屏幕位置。
 * px/py 是相对视口左上角的像素。
 */
export function zoomAtPoint(view: ViewState, nextZoom: number, px: number, py: number): ViewState {
  const zoom = clampZoom(nextZoom)
  const ix = (px - view.x) / view.zoom
  const iy = (py - view.y) / view.zoom
  return { zoom, x: Math.round(px - ix * zoom), y: Math.round(py - iy * zoom) }
}

/** 视口内像素 → 图像像素（用于光标坐标读数；越界照算，由调用方决定是否显示） */
export function imagePointAt(view: ViewState, px: number, py: number): { x: number; y: number } {
  return { x: (px - view.x) / view.zoom, y: (py - view.y) / view.zoom }
}

/** 刻度步长：保证 step × zoom ≥ TICK_MIN_GAP 的最小档 */
export function tickStep(zoom: number): number {
  const z = clampZoom(zoom)
  for (const step of TICK_STEPS) {
    if (step * z >= TICK_MIN_GAP) return step
  }
  return TICK_STEPS[TICK_STEPS.length - 1]
}

/**
 * 可见刻度：图像坐标 0..imageSize 中落在视口（0..viewportSize）内的刻度。
 * `pos` 是视口内像素位置，`label` 是图像像素坐标（刻度条上显示的数值）。
 */
export function ticksFor(
  view: ViewState,
  imageSize: number,
  viewportSize: number,
): { step: number; ticks: { pos: number; label: number }[] } {
  const step = tickStep(view.zoom)
  const z = clampZoom(view.zoom)
  const startImage = Math.max(0, Math.floor(-view.x / z / step) * step)
  const endImage = Math.min(imageSize, (viewportSize - view.x) / z)
  const ticks: { pos: number; label: number }[] = []
  for (let v = startImage; v <= endImage; v += step) {
    const pos = Math.round(v * z + view.x)
    if (pos < 0 || pos > viewportSize) continue
    ticks.push({ pos, label: v })
  }
  return { step, ticks }
}

/** 缩放读数：整数百分比；>100% 也保持整数（不出现 133.333%） */
export function formatZoom(zoom: number): string {
  return `${Math.round(clampZoom(zoom) * 100)}%`
}

/** 按倍率步进（按钮 +/- 用）：每档 ×1.25，钳到边界 */
export function stepZoom(zoom: number, dir: 1 | -1): number {
  return clampZoom(zoom * (dir > 0 ? 1.25 : 1 / 1.25))
}
