/**
 * 板块级「当前是否可见」信号（keepAlive 隐藏态治理）。
 *
 * ## 为什么需要
 * 壳层 `MainLayout` 对访问过的板块做 **keepAlive**：`visitedPages` 里所有页面
 * **常驻挂载**，只靠 `display:none` 隐藏（`layouts/MainLayout.tsx:763-783`）。
 * 也就是说「切到别的板块」≠「卸载」——React 组件**无法从自身生命周期**区分
 * 「我是当前可见板块」与「我正在后台保活」。
 *
 * 于是任何**窗口级副作用**在后台板块里照样生效。小说板块的实测症状：进过
 * 「小说 · 阅读」子页后再切到办公/原罪等板块，隐藏的阅读页仍然 `preventDefault`
 * 吞掉 **F11**（浏览器全屏失效）、仍抢 **Ctrl+S**（在别的板块按保存，实际保存的是
 * 那本小说的当前章），`readMode` 残留时 **←/→** 还在翻隐藏阅读页的章。
 * v4.421.0 只把小说**子页之间**的门控做齐（`NovelPage` 下发 `active`），
 * 板块级这一层当时列入观察池（「板块级 keepAlive」）。
 *
 * ## 语义
 * - 壳层在 `page` 变化时调用 `notifyBoardActive(page)`，声明**唯一**可见的板块。
 * - 页面用 `useBoardActive('novel')`（渲染路径）或 `isBoardActive('novel')`
 *   （window 事件处理器等非渲染路径）拿到「本板块当前是否可见」。
 * - **未收到任何通知时恒为 `true`**：兼容「页面被单独渲染」的既有测试写法，
 *   也让真实壳层挂载前后的行为连续（壳层挂载后会立刻通知一次）。
 *
 * ## 与 `lib/pollingGate` 的区别（两个正交维度，勿混用）
 * - 本模块 = 「**板块被切走**」（keepAlive `display:none`，`document` 仍可见）。
 * - `isPageVisible()` = 「**窗口不可见**」（最小化/遮挡，`visibilityState==='hidden'`）。
 * 需要两个都挡的场合（后台轮询）应各自判一次。
 */
import { useSyncExternalStore } from 'react'
import type { BoardId } from '../boards/manifests'

/** '' 表示「尚未收到壳层通知」——按文件头约定视为可见。 */
let activePage: BoardId | '' = ''
const listeners = new Set<() => void>()

function emit(): void {
  for (const l of listeners) l()
}

/** 壳层（MainLayout）在 `page` 变化时调用：声明当前唯一可见的板块。 */
export function notifyBoardActive(page: BoardId): void {
  if (page === activePage) return
  activePage = page
  emit()
}

/** 测试用：回到「未通知」默认态（等价于可见）。 */
export function resetBoardActive(): void {
  if (activePage === '') return
  activePage = ''
  emit()
}

/** 测试用：读当前已声明的板块 id（'' = 未通知）。 */
export function currentBoardActive(): BoardId | '' {
  return activePage
}

function subscribe(onChange: () => void): () => void {
  listeners.add(onChange)
  return () => { listeners.delete(onChange) }
}

/** 同步判读：供 window 事件处理器等非渲染路径使用。 */
export function isBoardActive(page: BoardId): boolean {
  return activePage === '' || activePage === page
}

/** 本板块当前是否为壳层可见板块（未收到通知时恒 true，见文件头「语义」段）。 */
export function useBoardActive(page: BoardId): boolean {
  return useSyncExternalStore(subscribe, () => activePage === '' || activePage === page)
}
