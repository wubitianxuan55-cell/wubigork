/**
 * schedule/gantt/useDragGesture.ts — 窗口级拖拽三件套单源（FE7-12）
 *
 * AoaView.beginDrag（节点布点）与 GanttView.beginDrag（条形拖移/缩放）此前各自
 * 内联同一套 window 监听三件套（mousemove/mouseup/keydown-Esc）：up=移除三监听
 * → moved 才提交 → 清渲染态；Esc=移除三监听 → 清渲染态（不提交）；监听器回调
 * 内自清理（Esc 与 up 双跑安全）。本模块把外壳收敛为 beginWindowDrag，视图侧
 * 差异（位移折算/吸附/提交落库）全部经 WindowDragSpec 注入——各视图语义逐字段
 * 保留（拖拽手感不变，参数值冻结）。
 *
 * 注意：外壳的 up 顺序统一为「移除监听 → commit（仅 moved）→ end」，与
 * GanttView 原顺序一致；AoaView 原为「end → commit」，end 只清本地渲染态
 * （ref + setDrag(null)），commit 只读 payload 与组件闭包，同一事件批次内
 * React 合并渲染，两种顺序的最终 DOM/落库完全一致。
 */

/** 一次拖拽会话的视图侧契约（payload=视图自有的拖拽载荷，闭包可变对象） */
export interface WindowDragSpec<P> {
  /** mousemove：屏幕位移 → 推进 payload（含渲染态预览写回，视图自管） */
  move(payload: P, ev: MouseEvent): void
  /** mouseup：是否视作已移动（true 才 commit；「未移动不提交」两视图同规） */
  moved(payload: P): boolean
  /** mouseup 且 moved=true：提交落库（渲染期外，updater 不得带副作用） */
  commit(payload: P): void
  /** up / Esc 收尾：清渲染态（预览复位；AoaView 兼清载荷 ref） */
  end(): void
}

/** window 监听三件套（mousemove/mouseup/keydown-Esc）单源外壳 */
export function beginWindowDrag<P>(spec: WindowDragSpec<P>, payload: P): void {
  const onMove = (ev: MouseEvent): void => {
    spec.move(payload, ev)
  }
  const detach = (): void => {
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
    window.removeEventListener('keydown', onKey)
  }
  const onUp = (): void => {
    detach()
    if (spec.moved(payload)) spec.commit(payload)
    spec.end()
  }
  const onKey = (ev: KeyboardEvent): void => {
    if (ev.key !== 'Escape') return
    detach()
    spec.end() // 未提交=取消
  }
  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
  window.addEventListener('keydown', onKey)
}
