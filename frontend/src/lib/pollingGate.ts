/**
 * 系统级后台轮询门控（P1 治理）。
 *
 * 背景：gaea 壳层 keepAlive 页面（display:none）永不卸载，访问过的页面其
 * setInterval 会一直运行（5–30s × ~10 组件）。桌面应用常开一整天、窗口
 * 最小化/切走时这些轮询仍在打后端，是最大的运行时浪费。
 *
 * 约定：document.visibilityState === 'hidden'（窗口最小化/遮挡/切走）时，
 * 轮询 interval 保留但执行体直接返回（空转零成本）；恢复可见时由
 * usePollingGate 驱动 effect 重跑并立即补一次拉取。
 */
export function isPageVisible(): boolean {
  return typeof document === 'undefined' || document.visibilityState !== 'hidden'
}
