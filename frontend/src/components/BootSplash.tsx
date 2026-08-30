/**
 * BootSplash — gaea 启动动画（星枢唤醒仪式）
 *
 * 职责：
 *  1. 接管 index.html 里的静态启动屏（#boot-splash），避免两段动画叠加；
 *  2. 播放「星枢唤醒」过程：光环旋转 + 徽记辉光 + 分步状态文案 + 进度条
 *     （核心 → 引擎 → 记忆 → 就绪），随后淡出并卸载，露出首页；
 *  3. 健壮性：进度/卸载全部由 timer 驱动（不依赖 CSS 动画完成），
 *     WebView2 rAF 节流（html.gaea-raf-degraded）下依然按时收场；
 *  4. prefers-reduced-motion / ui-reduced-motion：跳过旋转/脉冲等装饰动画，
 *     快速淡入淡出，不制造不适。
 *
 * 视觉：与 gaea 3.0「星枢 Constellation OS」同源——深空 aurora、--gaea-glow
 * 辉光、1px 高光线；零硬编码色值（fallback 兜底 WebView2 冷启动）。
 */
import React, { useEffect, useRef, useState } from 'react'
import { useT } from '../gaea/lib/i18n'
import './boot-splash.css'

/** 启动步骤（i18n 文案键）；true 表示已就绪 */
const BOOT_STEPS = ['boot.stepCore', 'boot.stepEngine', 'boot.stepMemory', 'boot.stepReady'] as const

/** 每步停留时长（ms）；reduced-motion 下整体压缩 */
const STEP_MS = 440
const LEAVE_MS = 380
const MIN_TOTAL_MS = 2100

const BootSplash: React.FC = () => {
  const t = useT()
  const [step, setStep] = useState(0)
  const [leaving, setLeaving] = useState(false)
  const [gone, setGone] = useState(false)
  const reduced = useRef(false)
  const timers = useRef<number[]>([])

  // 接管静态启动屏 + 记录 reduced-motion
  useEffect(() => {
    try {
      document.getElementById('boot-splash')?.remove()
    } catch { /* 忽略 */ }
    const mq = window.matchMedia?.('(prefers-reduced-motion: reduce)')
    reduced.current = mq?.matches ?? false
    const ids = timers.current
    return () => { ids.forEach((id) => window.clearTimeout(id)) }
  }, [])

  // 分步推进（timer 驱动，不依赖 rAF/CSS 动画）
  useEffect(() => {
    const push = (fn: () => void, ms: number) => {
      timers.current.push(window.setTimeout(fn, ms))
    }
    const finish = () => {
      setLeaving(true)
      push(() => setGone(true), LEAVE_MS)
    }
    if (reduced.current) {
      // 静默就绪：状态直接到「一切就绪」，短停留后淡出
      setStep(BOOT_STEPS.length - 1)
      push(finish, 700)
      return
    }
    const total = BOOT_STEPS.length * STEP_MS
    BOOT_STEPS.forEach((_, i) => {
      push(() => setStep(i), i * STEP_MS)
    })
    push(finish, Math.max(total, MIN_TOTAL_MS) + 260)
  }, [])

  if (gone) return null

  const progress = Math.min(100, Math.round(((step + 1) / BOOT_STEPS.length) * 100))
  const label = t(BOOT_STEPS[Math.min(step, BOOT_STEPS.length - 1)])

  return (
    <div
      className={`bs-splash${leaving ? ' is-leaving' : ''}${reduced.current ? ' is-reduced' : ''}`}
      role="status"
      aria-live="polite"
      aria-label="gaea 启动中"
    >
      <div className="bs-aurora" aria-hidden="true" />
      <div className="bs-center">
        <div className="bs-ring" aria-hidden="true">
          <span className="bs-ring-arc" />
          <span className="bs-ring-halo" />
          <span className="bs-orb">
            <img src="/favicon.svg" alt="" width="40" height="40" />
          </span>
        </div>
        <div className="bs-word">gaea</div>
        <div className="bs-status">{label}</div>
        <div className="bs-track" aria-hidden="true">
          <div className="bs-fill" style={{ width: `${progress}%` }} />
        </div>
        <div className="bs-tag">CONSTELLATION OS</div>
      </div>
    </div>
  )
}

export default BootSplash
