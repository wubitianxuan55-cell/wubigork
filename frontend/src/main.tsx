import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'

// ═══ WebView2 rAF 节流降级 ═══════════════════════════════════════════
// 背景（2026-08-06 根因定位）：Wails WebView2 在特定状态下把 requestAnimationFrame
// 节流到 ~1fps（页面 visibilityState 仍为 visible）。antd 弹层（rc-motion）的打开
// 动画靠 rAF 推进状态机：onEnterPrepare → Promise → rAF → 推进到 active。
// 且 WebView2 会把 CSS 动画 tick 一起挂起（animation 卡在 keyframe 首帧 opacity:0）
// → 表现为「模型中心-功能绑定」下拉点击无反应/闪烁/延迟 3s 才出现。
// 修复：
//   1) 检测到 rAF 帧率 <30fps 时，用 setTimeout(16ms) 模拟 rAF（不受 WebView2 节流影响）；
//   2) 给 <html> 加 gaea-raf-degraded 类，index.css 据此禁用 antd 弹层动画
//      （打开立即显示、关闭立即隐藏），彻底绕开被挂起的 CSS 动画 tick。
// 正常浏览器（60fps）不触发降级，不影响性能敏感组件（3D 图谱等）。
;(function ensureRAF() {
  if (typeof window === 'undefined' || !window.requestAnimationFrame) return
  let frames = 0
  let last = performance.now()
  let degraded = false
  let healthy = false

  const degrade = (fps: number) => {
    if (degraded) return
    degraded = true
    document.documentElement.classList.add('gaea-raf-degraded')
    console.warn(`[gaea-rAF] requestAnimationFrame 帧率过低(${fps}fps)，降级为 setTimeout(16ms) 并关闭 antd 弹层动画`)
    const nativeCancel = window.cancelAnimationFrame.bind(window)
    window.requestAnimationFrame = (cb) => {
      const start = performance.now()
      return window.setTimeout(() => cb(start + 16), 16)
    }
    window.cancelAnimationFrame = (id) => {
      try { nativeCancel(id as number) } catch { /* 忽略 */ }
      window.clearTimeout(id as number)
    }
  }

  const probe = (now: number) => {
    frames++
    if (now - last >= 1000) {
      const fps = Math.round((frames * 1000) / (now - last))
      if (fps < 30) degrade(fps)
      else healthy = true
      frames = 0
      last = now
    }
    if (!degraded) window.requestAnimationFrame(probe)
  }
  window.requestAnimationFrame(probe)

  // 安全网：3s 内未确认健康 rAF（可能被完全冻结）也降级，避免探测永远不触发
  window.setTimeout(() => {
    if (!degraded && !healthy) degrade(0)
  }, 3000)
})()

// ═══ 固定窗口缩放 ═══════════════════════════════════════════════════
// WebView2 默认允许 Ctrl+滚轮 / 触控板捏合（映射为 Ctrl+滚轮）/ Ctrl+±
// 缩放整个页面。这里在 JS 层拦截，让滚轮只滚动、窗口缩放保持固定 100%。
;(function lockWindowZoom() {
  const prevent = (e: Event) => e.preventDefault()
  window.addEventListener(
    'wheel',
    (e: WheelEvent) => {
      if (e.ctrlKey || e.metaKey) e.preventDefault()
    },
    { passive: false },
  )
  document.addEventListener('gesturestart', prevent, { passive: false })
  document.addEventListener('gesturechange', prevent, { passive: false })
  window.addEventListener('keydown', (e: KeyboardEvent) => {
    if (!e.ctrlKey && !e.metaKey) return
    if (['+', '-', '=', '_', '0'].includes(e.key)) e.preventDefault()
  })
})()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
