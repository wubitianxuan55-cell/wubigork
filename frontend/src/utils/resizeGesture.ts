/**
 * resizeGesture — 指针拖拽改宽手势的共享生命周期（2026-10-04 第三轮审计
 * §3.2：四处逐字相同的 listener 三件套 + body cursor/userSelect 设置与还原
 * 收敛单源）。onDone 先于清理执行（对齐既有顺序：先落宽/落状态，再摘监听）。
 */
export function startColumnResizeGesture(
  onMove: (e: PointerEvent) => void,
  onDone: () => void,
): void {
  const finish = (): void => {
    onDone()
    window.removeEventListener("pointermove", onMove)
    window.removeEventListener("pointerup", finish)
    window.removeEventListener("pointercancel", finish)
    document.body.style.cursor = ""
    document.body.style.userSelect = ""
  }
  document.body.style.cursor = "col-resize"
  document.body.style.userSelect = "none"
  window.addEventListener("pointermove", onMove)
  window.addEventListener("pointerup", finish)
  window.addEventListener("pointercancel", finish)
}
