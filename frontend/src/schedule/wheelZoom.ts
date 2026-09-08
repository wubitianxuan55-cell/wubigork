/**
 * schedule/wheelZoom.ts — 网络图画布滚轮缩放（单代号/双代号共用，v4.159）
 *
 * 普通滚轮=以光标为锚缩放（画布类工具口径；此前只有按钮缩放，长图查看
 * 不便——用户实测反馈），Shift+滚轮=放行原生横向滚动。锚定：把「光标下的
 * 图面坐标」记下来，缩放提交后的 layout effect 里回写 scrollLeft/Top——
 * 直接同步回写会被旧内容尺寸钳制（setState 尚未重渲染）。
 */
import React from 'react'

export function useWheelZoom(
  scrollRef: React.RefObject<HTMLElement | null>,
  zoomRef: React.RefObject<number>,
  applyZoom: (z: number) => void,
): void {
  const anchorRef = React.useRef<{ px: number; py: number; gx: number; gy: number } | null>(null)
  const applyRef = React.useRef(applyZoom) // 视图每渲染重建回调：走 ref 免监听器反复拆挂
  applyRef.current = applyZoom
  // 缩放提交（DOM 尺寸更新）后回写锚点滚动量；anchorRef 空时零开销，故不设依赖
  React.useLayoutEffect(() => {
    const el = scrollRef.current
    const a = anchorRef.current
    if (!el || !a) return
    el.scrollLeft = a.gx * (zoomRef.current || 1) - a.px
    el.scrollTop = a.gy * (zoomRef.current || 1) - a.py
    anchorRef.current = null
  })
  React.useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const onWheel = (e: WheelEvent) => {
      if (e.shiftKey) return // 横向滚动交给原生
      e.preventDefault()
      const z0 = zoomRef.current || 1
      const rect = el.getBoundingClientRect()
      const px = e.clientX - rect.left
      const py = e.clientY - rect.top
      anchorRef.current = { px, py, gx: (el.scrollLeft + px) / z0, gy: (el.scrollTop + py) / z0 }
      applyRef.current(Math.min(2, Math.max(0.1, z0 * (e.deltaY < 0 ? 1.2 : 1 / 1.2))))
    }
    el.addEventListener('wheel', onWheel, { passive: false })
    return () => el.removeEventListener('wheel', onWheel)
  }, [scrollRef, zoomRef]) // zoomRef/scrollRef 均为 ref 对象（恒稳定），applyZoom 走 applyRef
}
