import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import './v3/foundation.css' // 3.0「星枢 Constellation OS」v3 基础样式层（令牌派生/壳层/分区原语）
import App from './App.tsx'
import { ErrorBoundary } from './components/ErrorBoundary'
import { ToastProvider } from './gaea/components/Toast'
import { app } from './gaea/lib/bridge'
import { lazy } from 'react'
import { registerPage } from './boards/pageRegistry'

// ═══ PageRegistry 集中注册（3.0 §5.2 附 B #3/#5）══════════════════════
// manifest.page 键 ↔ 页面组件统一 lazy 包装；MainLayout 渲染时按 manifest 查表。
// home 启动器为壳层（ModuleLauncher），不注册组件。过渡期后删除 MainLayout 的
// legacy lazy import，全部以本注册表为唯一来源。
registerPage('ChatPage', lazy(() => import('./pages/ChatPage')))
registerPage('NovelPage', lazy(() => import('./pages/NovelPage')))
registerPage('ImageGenPage', lazy(() => import('./pages/ImageGenPage')))
registerPage('GaeaPage', lazy(() => import('./pages/GaeaPage')))
registerPage('ProgrammingPage', lazy(() => import('./pages/ProgrammingPage')))
registerPage('MemoryHubPage', lazy(() => import('./pages/MemoryHubPage')))
registerPage('CostLibraryPage', lazy(() => import('./pages/CostLibraryPage')))
registerPage('ModelCenterPage', lazy(() => import('./pages/ModelCenterPage')))
registerPage('CharacterLibraryPage', lazy(() => import('./pages/CharacterLibraryPage')))
registerPage('SettingsPage', lazy(() => import('./pages/SettingsPage')))
// v4.4 触点：微信助手（书房板块，扫码绑定 + 离线代办提醒）
registerPage('WeixinPage', lazy(() => import('./pages/WeixinPage')))
// D7 knowledge 独立板块（页面文件已存在；GetBoardManifests 接线后菜单可点击）——3.0 Wave 3 集成补注册
registerPage('KnowledgePage', lazy(() => import('./pages/KnowledgePage')))


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
  let degraded = false

  const degrade = (fps: number) => {
    if (degraded) return
    degraded = true
    document.documentElement.classList.add('gaea-raf-degraded')
    console.warn(`[gaea-rAF] requestAnimationFrame 帧率过低(${fps}fps)，降级为 setTimeout(16ms) 并关闭入场动画`)
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

  // 持续探测（而非只在启动时测一次）：WebView2 可能在运行中才把 rAF
  // 节流到 ~1fps（失焦/后台误判等），晚发节流同样会让卡片/弹层动画卡首帧。
  let frames = 0
  let last = performance.now()
  let lastBeat = performance.now()
  let sawFrame = false

  const probe = (now: number) => {
    if (degraded) return
    sawFrame = true
    lastBeat = now
    frames++
    if (now - last >= 1000) {
      const fps = Math.round((frames * 1000) / (now - last))
      if (fps < 30) {
        degrade(fps)
        return
      }
      frames = 0
      last = now
    }
    window.requestAnimationFrame(probe)
  }
  window.requestAnimationFrame(probe)

  // 安全网 1：3s 内 rAF 一次都没跑（启动即被冻结）也降级。
  window.setTimeout(() => {
    if (!degraded && !sawFrame) degrade(0)
  }, 3000)
  // 安全网 2：看门狗——运行中 rAF 被彻底冻结（probe 停止推进）也降级。
  window.setInterval(() => {
    if (!degraded && performance.now() - lastBeat > 8000) degrade(0)
  }, 5000)
})()

// ═══ 前端错误与主线程卡死诊断（写入 gaea.log）══════════════════
// 反复出现「界面卡死、什么都点不了」：把 window error / 未处理 rejection /
// 主线程长任务 / 心跳中断全部上报到 gaea.log，下次卡死可直接定位根因。
;(function installFrontendDiagnostics() {
  const log = (tag: string, msg: string) => {
    app.LogFrontendError(`[${tag}] ${msg}`).catch(() => {})
  }

  window.addEventListener('error', (e) => {
    log('window.onerror', `${e.message} @ ${e.filename}:${e.lineno}\n${e.error?.stack || ''}`)
  })
  window.addEventListener('unhandledrejection', (e) => {
    const r = e.reason
    log('unhandledrejection', r instanceof Error ? `${r.message}\n${r.stack}` : String(r))
  })

  // WebView2 的原生 confirm/alert/prompt 会同步阻塞主线程且可能不显示，
  // 一旦被触发整个界面“卡死”（零 CPU、无错误、定时器全停）。全局替换为
  // 非阻塞实现：记录日志并按“取消/空值”返回，杜绝卡死；需要确认的交互
  // 必须改用 antd 的异步 Modal.confirm。
  const freezeGuard = (kind: string) => (msg?: string) => {
    log('blocked-sync-dialog', `${kind}: ${String(msg ?? '').slice(0, 200)}`)
    return false
  }
  try {
    window.confirm = freezeGuard('confirm') as typeof window.confirm
    window.alert = freezeGuard('alert') as typeof window.alert
    window.prompt = (() => null) as typeof window.prompt
  } catch { /* 某些环境只读，忽略 */ }

  // 主线程心跳：若被长时间占用（死循环/巨量渲染），恢复后上报卡死时长。
  let lastBeat = Date.now()
  setInterval(() => { lastBeat = Date.now() }, 500)
  setInterval(() => {
    const stall = Date.now() - lastBeat
    if (stall > 2000) {
      log('main-thread-stall', `主线程被阻塞约 ${Math.round(stall / 1000)}s`)
      lastBeat = Date.now()
    }
  }, 8000)

  // longtask：单次 >50ms 的主线程长任务（死循环/巨量渲染的元凶）。
  let longLogAt = 0
  try {
    const obs = new PerformanceObserver((list) => {
      const now = Date.now()
      if (now - longLogAt < 10000) return
      for (const entry of list.getEntries()) {
        const e = entry as unknown as { duration: number; attribution?: { name?: string; containerType?: string }[] }
        const culprit = e.attribution?.length
          ? e.attribution.map((a) => a.name || a.containerType || '').filter(Boolean).join(';')
          : ''
        log('longtask', `duration=${Math.round(e.duration)}ms culprit=${culprit || 'unknown'}`)
        longLogAt = now
        break
      }
    })
    obs.observe({ entryTypes: ['longtask'] })
  } catch { /* 不支持时忽略 */ }
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
    {/* 根级兜底：antd 弹层等 portal 内渲染异常也会被捕获并记录，
        避免整个窗口变成“什么都点不了”的死图。 */}
    <ErrorBoundary>
      <ToastProvider>
        <App />
      </ToastProvider>
    </ErrorBoundary>
  </StrictMode>,
)
