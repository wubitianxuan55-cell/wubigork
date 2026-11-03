/**
 * BootSplash — gaea 大地创生启动序列
 *
 * 参考当下 DSH 莱茵生命主题真正有效的结构：暖白/暖炭底色、单一莱茵橙、
 * 两幕权限叙事、中央样板卡、左侧五段进度和地形式背景；不复制其素材。
 *
 * gaea 自有语言：
 *  - 大地母神而非通用 AI：地核、根系、年轮记忆、生物圈与终端；
 *  - 地质等高线与根系样品代替细胞培养和科幻光环；
 *  - 暖陶土/石白/苔藓绿，避免荧光绿赛博屏。
 */
import React, { useCallback, useEffect, useRef, useState } from 'react'
import type { CSSProperties } from 'react'
import { useT } from '../gaea/lib/i18n'
import { createBootAudio } from './boot-audio'
import type { BootAudio } from './boot-audio'
import './boot-splash.css'

const BOOT_STEPS = [
  { key: 'boot.stepCore', rail: 'ACCESS', code: 'E-00' },
  { key: 'boot.stepRoots', rail: 'ROOT NETWORK', code: 'E-01' },
  { key: 'boot.stepMemory', rail: 'MEMORY', code: 'E-02' },
  { key: 'boot.stepBiosphere', rail: 'BIOSPHERE', code: 'E-03' },
  { key: 'boot.stepReady', rail: 'TERMINAL', code: 'E-04' },
] as const

const MODULE_TILES = [
  ['LANG', 'LANGUAGE CORE'],
  ['MEM', 'MEMORY STRATA'],
  ['VISION', 'IMAGE FIELD'],
  ['CRAFT', 'OFFICE SYSTEM'],
  ['LOCAL', 'LOCAL LINK'],
] as const

const STEP_AT_MS = [0, 900, 1800, 3000, 4300] as const
const STEP_PROGRESS = [6, 18, 42, 68, 92] as const
const FINISH_MS = 7400
const EXIT_MS = 920
const SKIP_EXIT_MS = 440

type AudioState = 'starting' | 'live' | 'blocked' | 'muted' | 'unavailable'

const BootSplash: React.FC = () => {
  const t = useT()
  const [step, setStep] = useState(0)
  const [leaving, setLeaving] = useState(false)
  const [gone, setGone] = useState(false)
  const [muted, setMuted] = useState(false)
  const [audioState, setAudioState] = useState<AudioState>('starting')
  const reduced = useRef(false)
  const timers = useRef<number[]>([])
  const ending = useRef(false)
  const mutedRef = useRef(false)
  const audio = useRef<BootAudio | null>(null)
  const voiceRef = useRef<HTMLAudioElement>(null)
  const soundRef = useRef<HTMLButtonElement>(null)
  const skipRef = useRef<HTMLButtonElement>(null)

  const clearTimers = useCallback(() => {
    timers.current.forEach((id) => window.clearTimeout(id))
    timers.current = []
  }, [])

  const syncAudioState = useCallback(() => {
    const kit = audio.current
    if (!kit) {
      setAudioState('unavailable')
      return
    }
    if (mutedRef.current) {
      setAudioState('muted')
      return
    }
    setAudioState(kit.context.state === 'running' ? 'live' : 'blocked')
  }, [])

  const finish = useCallback((skipped = false) => {
    if (ending.current) return
    ending.current = true
    clearTimers()
    setStep(BOOT_STEPS.length - 1)
    setLeaving(true)
    audio.current?.stop()
    audio.current = null
    if (voiceRef.current) {
      voiceRef.current.pause()
      voiceRef.current.currentTime = 0
    }
    const exitMs = skipped ? SKIP_EXIT_MS : reduced.current ? 180 : EXIT_MS
    timers.current.push(window.setTimeout(() => setGone(true), exitMs))
  }, [clearTimers])

  const skip = useCallback(() => {
    finish(true)
  }, [finish])

  const toggleSound = useCallback(() => {
    const next = !mutedRef.current
    mutedRef.current = next
    setMuted(next)
    const kit = audio.current
    if (!kit) {
      setAudioState('unavailable')
      return
    }
    const now = kit.context.currentTime
    kit.master.gain.cancelScheduledValues(now)
    kit.master.gain.setTargetAtTime(next ? 0.0001 : 0.28, now, 0.035)
    if (voiceRef.current) voiceRef.current.muted = next
    if (next) {
      setAudioState('muted')
      return
    }
    setAudioState('starting')
    void kit.context.resume().then(syncAudioState).catch(() => setAudioState('blocked'))
  }, [syncAudioState])

  const tryPlayVoice = useCallback(() => {
    const voiceEl = voiceRef.current
    if (!voiceEl) return
    voiceEl.muted = mutedRef.current
    voiceEl.volume = 0.92
    if (ending.current || !voiceEl.paused || voiceEl.readyState < 2) return
    try {
      const playing = voiceEl.play()
      if (playing && typeof playing.catch === 'function') {
        void playing.catch(() => setAudioState('blocked'))
      }
    } catch {
      setAudioState('blocked')
    }
  }, [])

  useEffect(() => {
    try {
      document.getElementById('boot-splash')?.remove()
    } catch { /* 忽略 */ }

    const mq = window.matchMedia?.('(prefers-reduced-motion: reduce)')
    reduced.current = mq?.matches ?? false

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        event.stopPropagation()
        skip()
        return
      }
      if (event.key === 'Tab') {
        const controls = [soundRef.current, skipRef.current].filter(
          (node): node is HTMLButtonElement => Boolean(node),
        )
        if (controls.length === 0) return
        event.preventDefault()
        event.stopPropagation()
        const current = controls.indexOf(document.activeElement as HTMLButtonElement)
        const direction = event.shiftKey ? -1 : 1
        const next = current < 0
          ? controls[0]
          : controls[(current + direction + controls.length) % controls.length]
        next.focus({ preventScroll: true })
      }
    }

    window.addEventListener('keydown', onKeyDown, true)
    return () => {
      window.removeEventListener('keydown', onKeyDown, true)
      clearTimers()
    }
  }, [clearTimers, skip])

  useEffect(() => {
    const kit = createBootAudio(FINISH_MS)
    audio.current = kit
    if (!kit) {
      setAudioState('unavailable')
      return
    }

    const resume = () => {
      if (!ending.current && kit.context.state === 'suspended') {
        void kit.context.resume().then(syncAudioState).catch(() => setAudioState('blocked'))
      }
      tryPlayVoice()
    }
    const onStateChange = () => syncAudioState()
    const voiceTimer = window.setTimeout(resume, 160)

    kit.context.addEventListener('statechange', onStateChange)
    void kit.context.resume().then(syncAudioState).catch(() => setAudioState('blocked'))
    syncAudioState()
    window.addEventListener('pointerdown', resume, true)
    window.addEventListener('keydown', resume, true)
    return () => {
      window.clearTimeout(voiceTimer)
      kit.context.removeEventListener('statechange', onStateChange)
      window.removeEventListener('pointerdown', resume, true)
      window.removeEventListener('keydown', resume, true)
      kit.stop()
      if (audio.current === kit) audio.current = null
    }
  }, [syncAudioState, tryPlayVoice])

  useEffect(() => {
    const push = (fn: () => void, ms: number) => {
      timers.current.push(window.setTimeout(fn, ms))
    }

    if (reduced.current) {
      setStep(BOOT_STEPS.length - 1)
      push(() => finish(false), 760)
      return
    }

    STEP_AT_MS.forEach((at, index) => {
      push(() => setStep(index), at)
    })
    push(() => finish(false), FINISH_MS)
  }, [finish])

  if (gone) return null

  const progress = leaving ? 100 : STEP_PROGRESS[Math.min(step, STEP_PROGRESS.length - 1)]
  const label = t(BOOT_STEPS[Math.min(step, BOOT_STEPS.length - 1)].key)
  const progressStyle = { '--bs-progress': progress / 100 } as CSSProperties
  const authorized = step >= 1
  const welcome = step >= 2
  const terminalReady = step >= 4
  const railActive = leaving ? BOOT_STEPS.length : step + 1
  const statusText = terminalReady ? 'TERMINAL READY' : authorized ? 'ACCESS GRANTED' : 'REQUESTING ACCESS'
  const audioText = audioState === 'live'
    ? 'AUDIO LIVE'
    : audioState === 'muted'
      ? 'MUTED'
      : audioState === 'blocked'
        ? 'AUDIO READY'
        : audioState === 'unavailable'
          ? 'SILENT'
          : 'AUDIO'

  return (
    <div
      className={`bs-splash${leaving ? ' is-leaving' : ''}${reduced.current ? ' is-reduced' : ''}`}
      data-stage={step}
      data-audio-state={audioState}
      role="dialog"
      aria-modal="true"
      aria-labelledby="bs-brand"
      aria-describedby="bs-status"
      aria-label={t('boot.loading')}
      data-testid="boot-splash"
    >
      <audio
        ref={voiceRef}
        className="bs-voice-audio"
        src="/boot-gaea-voice.wav"
        preload="auto"
        autoPlay
        aria-hidden="true"
        tabIndex={-1}
        onCanPlay={tryPlayVoice}
        onError={() => setAudioState('blocked')}
      />
      <div className="bs-door bs-door--top" aria-hidden="true" />
      <div className="bs-door bs-door--bottom" aria-hidden="true" />
      <div className="bs-seam" aria-hidden="true" />
      <h1 id="bs-brand" className="bs-sr-status">GAEA</h1>
      <div id="bs-status" className="bs-sr-status" role="status" aria-live="polite">{label}</div>

      <div className="bs-stage">
        <div className="bs-terrain" aria-hidden="true">
          <svg viewBox="0 0 1600 1000" preserveAspectRatio="xMidYMid slice">
            <g className="bs-strata">
              <path d="M686 1000C770 831 819 696 934 569C1066 423 1278 339 1600 295" />
              <path d="M731 1000C814 858 860 748 963 635C1108 477 1322 398 1600 355" />
              <path d="M776 1000C866 877 914 800 1001 701C1142 543 1358 460 1600 414" />
              <path d="M823 1000C906 910 970 844 1048 761C1181 621 1396 520 1600 473" />
              <path d="M874 1000C959 934 1026 888 1098 819C1208 713 1430 580 1600 534" />
              <path d="M930 1000C1006 960 1088 930 1153 882C1253 807 1470 651 1600 594" />
            </g>
            <g className="bs-terraces">
              <path d="M1012 1000C1122 894 1224 831 1365 774C1450 739 1531 717 1600 703V1000Z" />
              <path d="M1093 1000C1178 933 1291 877 1410 841C1481 819 1544 805 1600 797V1000Z" />
              <path d="M1180 1000C1272 957 1368 926 1452 908C1503 897 1553 891 1600 887V1000Z" />
            </g>
          </svg>
        </div>

        <i className="bs-orbit bs-orbit--outer" aria-hidden="true" />
        <i className="bs-orbit bs-orbit--inner" aria-hidden="true" />
        <i className="bs-corner bs-corner--tl" aria-hidden="true" />
        <i className="bs-corner bs-corner--tr" aria-hidden="true" />
        <i className="bs-corner bs-corner--bl" aria-hidden="true" />
        <i className="bs-corner bs-corner--br" aria-hidden="true" />
        <i className="bs-reg bs-reg--top" aria-hidden="true" />
        <i className="bs-reg bs-reg--right" aria-hidden="true" />

        <header className="bs-topbar">
          <div className="bs-topbar-status">
            <span className="bs-square" aria-hidden="true" />
            <strong>{statusText}</strong>
            <span>// 大地创生序列</span>
          </div>
          <div className="bs-topbar-right">
            <button
              ref={soundRef}
              type="button"
              className={`bs-audio is-${audioState}`}
              onClick={toggleSound}
              aria-pressed={!muted && audioState !== 'unavailable'}
              aria-label={muted ? t('boot.soundOff') : t('boot.soundOn')}
            >
              <span className="bs-audio-bars" aria-hidden="true"><i /><i /><i /></span>
              <span>{audioText}</span>
            </button>
            <span className="bs-live"><i aria-hidden="true" />LIVE</span>
            <span>E-GAEA/4.529</span>
            <span>2026.11.03</span>
          </div>
        </header>

        <div className="bs-lockup">
          <div className="bs-kicker">PRIMORDIAL INTELLIGENCE</div>
          <div className="bs-brandline">GAEA</div>
          <div className="bs-brand-sub">EARTH SYSTEM <b>OS</b></div>
        </div>

        <nav className="bs-rail" aria-label="启动阶段">
          <i className="bs-rail-line" aria-hidden="true" />
          <i className="bs-rail-signal" aria-hidden="true" />
          {BOOT_STEPS.map((item, index) => (
            <div
              key={item.code}
              className={`bs-rail-item${index < railActive ? ' is-active' : ''}${index === step ? ' is-current' : ''}`}
            >
              <span className="bs-rail-mark" aria-hidden="true" />
              <span className="bs-rail-no">{String(index + 1).padStart(2, '0')}</span>
              <span className="bs-rail-text">{item.rail}</span>
              <i className="bs-rail-bar" aria-hidden="true" />
            </div>
          ))}
        </nav>

        <main className="bs-center">
          <div className={`bs-emblem${authorized ? ' is-authorized' : ''}`} aria-hidden="true">
            <svg viewBox="0 0 240 240">
              <circle className="bs-emblem-tick" cx="120" cy="120" r="109" />
              <g className="bs-emblem-bezel">
                {Array.from({ length: 32 }, (_, index) => (
                  <path
                    key={index}
                    d={`M120 10V${index % 4 === 0 ? 20 : 16}`}
                    transform={`rotate(${index * 11.25} 120 120)`}
                  />
                ))}
              </g>
              <circle className="bs-emblem-arc" cx="120" cy="120" r="94" />
              <circle className="bs-emblem-hairline" cx="120" cy="120" r="72" />
              <path className="bs-emblem-g" d="M149 91A48 48 0 1 0 155 132M126 120H159" />
              <circle className="bs-emblem-core" cx="120" cy="120" r="14" />
              <g className="bs-emblem-roots">
                <path d="M120 136V164M120 146L105 160M120 146L136 160M112 156L101 165M128 156L140 165" />
              </g>
            </svg>
            <span className="bs-axis bs-axis--x" aria-hidden="true" />
            <span className="bs-axis bs-axis--y" aria-hidden="true" />
          </div>

          <section className={`bs-panel${welcome ? ' is-welcome' : ' is-access'}`}>
            <div className="bs-panel-frame" aria-hidden="true"><i /><i /><i /><i /></div>
            <div className="bs-panel-head">
              <span><i className="bs-square" />BIOSPHERE ACCESS // INTERNAL</span>
              <span>PANEL-G1</span>
            </div>

            {!welcome ? (
              <div className="bs-access-copy">
                <div className="bs-access-kicker">{authorized ? 'PRIMORDIAL SIGNATURE / MATCHED' : 'LOCAL TERMINAL / E-00'}</div>
                <div className="bs-access-title">
                  {authorized ? t('boot.identityReady') : t('boot.identityCheck')}
                  <span className="bs-access-caret" aria-hidden="true" />
                </div>
                <div className="bs-access-en">
                  {authorized ? 'PERMISSION AUTHORIZED' : 'IDENTITY VERIFICATION IN PROGRESS'}
                </div>
                <div className="bs-access-sub">
                  <span>{authorized ? 'CLEARANCE L-03 · TERRA CORE LINKED' : 'VERIFYING PRIMORDIAL CORE...'}</span>
                  <span>{String(progress).padStart(3, '0')}%</span>
                </div>
                <div className="bs-id-grid">
                  <div><span>IDENTITY</span><b>GAEA / E-0001</b></div>
                  <div><span>CLEARANCE</span><b>{authorized ? 'PRIMORDIAL' : 'SCANNING'}</b></div>
                  <div><span>STATUS</span><b>{authorized ? 'CONFIRMED' : 'VERIFYING'}</b></div>
                </div>
                <div className="bs-scan-track" aria-hidden="true"><i /></div>
              </div>
            ) : (
              <div className="bs-welcome-copy">
                <div className="bs-welcome">WELCOME TO</div>
                <h1>GAEA</h1>
                <div className="bs-terminal-title">TERRA INTELLIGENCE</div>
                <div className="bs-link-track" aria-hidden="true">
                  <i />
                  <span className="bs-link-head" />
                  <b>LINK</b>
                </div>
                <div className="bs-panel-sub">PRIMORDIAL NETWORK · LOCAL-FIRST LIFE SYSTEM</div>
                <div className="bs-tiles">
                  {MODULE_TILES.map(([abbr, caption], index) => (
                    <div
                      key={abbr}
                      className={`bs-tile${index < Math.min(step, 4) ? ' is-on' : ''}`}
                      style={{ '--tile-index': index } as CSSProperties}
                    >
                      <span className="bs-tile-no">M-{String(index + 1).padStart(2, '0')}</span>
                      <strong>{abbr}</strong>
                      <small>{caption}</small>
                    </div>
                  ))}
                </div>
                <div className="bs-boot-status">
                  <i aria-hidden="true" />
                  {label}
                </div>
              </div>
            )}
          </section>
        </main>

        <aside className="bs-specimen" aria-hidden="true">
          <div className="bs-specimen-label"><strong>GAIA-01</strong><span>FIELD SAMPLE · SOIL CORE</span></div>
          <div className="bs-sample-disc">
            <div className="bs-soil-grain" />
            <svg viewBox="0 0 160 160">
              <circle cx="80" cy="80" r="57" />
              <path d="M80 32V77M80 61L57 45M80 55L105 37M80 77L51 91M80 77L112 94M64 73L48 108M96 74L116 54M91 88L108 120M70 89L59 123" />
            </svg>
            <span>24.6 g</span>
          </div>
          <div className="bs-specimen-meta">
            <span>ROOT DENSITY</span><b>78.2%</b>
            <span>DEPTH</span><b>120 CM</b>
          </div>
        </aside>

        <div className="bs-watermark" aria-hidden="true">GAEA</div>

        <footer className="bs-footer">
          <div className="bs-file">
            <span>FILE NUMBER · GAEA-SYS / 4.529</span>
            <span>CLASSIFICATION · PRIMORDIAL</span>
            <svg viewBox="0 0 180 18" preserveAspectRatio="none" aria-hidden="true">
              <path d="M0 0v18M4 0v18M7 0v18M12 0v18M15 0v18M20 0v18M24 0v18M30 0v18M33 0v18M38 0v18M45 0v18M48 0v18M54 0v18M58 0v18M64 0v18M67 0v18M72 0v18M77 0v18M81 0v18M88 0v18M91 0v18M95 0v18M102 0v18M107 0v18M111 0v18M118 0v18M122 0v18M128 0v18M132 0v18M139 0v18M144 0v18M148 0v18M155 0v18M159 0v18M166 0v18M170 0v18M176 0v18M180 0v18" />
            </svg>
          </div>

          <button ref={skipRef} type="button" className="bs-skip" onClick={skip} aria-label={`${t('boot.skip')} (ESC)`}>
            <kbd>ESC</kbd><span>{t('boot.skip')}</span>
            <i aria-hidden="true">↗</i>
          </button>

          <div className="bs-caution" aria-hidden="true">
            <span className="bs-stripes" />
            <span>CAUTION · AUTHORIZED PERSONNEL ONLY</span>
          </div>

          <div className="bs-loading" style={progressStyle}>
            <div className="bs-loading-meta"><span>LOADING</span><strong>{String(progress).padStart(3, '0')}%</strong></div>
            <div className="bs-loading-track" aria-hidden="true"><i /></div>
            <span className="bs-loading-phase">PHASE {String(Math.min(step + 1, 5)).padStart(2, '0')} / 05</span>
          </div>
        </footer>
      </div>
    </div>
  )
}

export default BootSplash
