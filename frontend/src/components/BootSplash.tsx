/**
 * BootSplash - GAEA "Earthseed" startup sequence.
 *
 * The brand mark is the seed. Around it, the startup draws an orbit, a
 * topographic field, and two live stages before revealing the workspace.
 * The composition intentionally uses the same deep canvas and teal accent as
 * the app shell so the handoff feels continuous rather than like two products.
 */
import React, { useCallback, useEffect, useRef, useState } from 'react'
import type { CSSProperties } from 'react'
import { useT } from '../gaea/lib/i18n'
import './boot-splash.css'

const BOOT_STEPS = [
  { key: 'boot.stepCore', code: 'CORE' },
  { key: 'boot.stepReady', code: 'READY' },
] as const

// Selected voice: clear, intimate and dry. The two phrases split at 1.596s.
// Keep these boundaries in sync with frontend/public/boot-gaea-voice-selected.wav.
const STEP_AT_MS = [0, 1596] as const
const STEP_PROGRESS = [8, 62] as const
const FINISH_MS = 4000
const FINISH_GUARD_MS = FINISH_MS + 300
const EXIT_MS = 520
const SKIP_EXIT_MS = 420
const REDUCED_EXIT_MS = 180

const CONTOURS = [
  'M-120 735C150 653 323 700 524 640C727 580 828 482 1038 504C1261 528 1421 636 1720 566',
  'M-120 781C155 701 337 749 539 690C733 634 852 536 1057 559C1273 584 1450 681 1720 616',
  'M-120 827C171 755 349 800 555 741C755 684 881 589 1082 610C1292 633 1481 726 1720 670',
  'M-120 875C177 811 375 850 575 794C773 738 907 644 1102 665C1311 687 1505 772 1720 722',
  'M-120 925C193 865 399 901 599 846C792 793 939 701 1123 720C1329 741 1538 820 1720 780',
  'M220 930C398 853 566 857 737 787C881 728 971 655 1095 659',
] as const

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
  const timelineStarted = useRef(false)
  const ending = useRef(false)
  const mutedRef = useRef(false)
  const voiceRef = useRef<HTMLAudioElement>(null)
  const soundRef = useRef<HTMLButtonElement>(null)
  const skipRef = useRef<HTMLButtonElement>(null)

  const clearTimers = useCallback(() => {
    timers.current.forEach((id) => window.clearTimeout(id))
    timers.current = []
  }, [])

  const finish = useCallback((skipped = false) => {
    if (ending.current) return
    ending.current = true
    clearTimers()
    setStep(BOOT_STEPS.length - 1)
    setLeaving(true)
    if (voiceRef.current) {
      voiceRef.current.pause()
      voiceRef.current.currentTime = 0
    }
    const exitMs = reduced.current ? REDUCED_EXIT_MS : skipped ? SKIP_EXIT_MS : EXIT_MS
    timers.current.push(window.setTimeout(() => setGone(true), exitMs))
  }, [clearTimers])

  const skip = useCallback(() => {
    finish(true)
  }, [finish])

  const startTimeline = useCallback(() => {
    if (timelineStarted.current || ending.current) return
    timelineStarted.current = true

    const push = (fn: () => void, ms: number) => {
      timers.current.push(window.setTimeout(fn, ms))
    }

    STEP_AT_MS.forEach((at, index) => {
      if (at === 0) setStep(index)
      else push(() => setStep(index), at)
    })
    // The media ended event is the normal close. This guard only covers a
    // platform where playback starts but the ended event never arrives.
    push(() => finish(false), FINISH_GUARD_MS)
  }, [finish])

  const tryPlayVoice = useCallback(() => {
    const voiceEl = voiceRef.current
    if (!voiceEl) return
    voiceEl.muted = mutedRef.current
    voiceEl.volume = 0.92
    if (ending.current || !voiceEl.paused || voiceEl.readyState < 2) return
    try {
      const playing = voiceEl.play()
      if (playing && typeof playing.then === 'function') {
        void playing
          .then(() => setAudioState(mutedRef.current ? 'muted' : 'live'))
          .catch(() => setAudioState('blocked'))
      } else {
        setAudioState(mutedRef.current ? 'muted' : 'live')
      }
    } catch {
      setAudioState('blocked')
    }
  }, [])

  const toggleSound = useCallback(() => {
    const next = !mutedRef.current
    mutedRef.current = next
    setMuted(next)
    if (voiceRef.current) voiceRef.current.muted = next
    if (next) {
      setAudioState('muted')
      return
    }
    if (voiceRef.current && !voiceRef.current.paused) {
      setAudioState('live')
      return
    }
    setAudioState('starting')
    tryPlayVoice()
  }, [tryPlayVoice])

  const handlePlaying = useCallback(() => {
    setAudioState(mutedRef.current ? 'muted' : 'live')
    startTimeline()
  }, [startTimeline])

  useEffect(() => {
    try {
      document.getElementById('boot-splash')?.remove()
    } catch { /* The static first-paint layer may already be gone. */ }

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
    const resume = () => {
      if (!ending.current) tryPlayVoice()
    }
    const voiceTimer = window.setTimeout(resume, 40)

    window.addEventListener('pointerdown', resume, true)
    window.addEventListener('keydown', resume, true)
    return () => {
      window.clearTimeout(voiceTimer)
      window.removeEventListener('pointerdown', resume, true)
      window.removeEventListener('keydown', resume, true)
    }
  }, [tryPlayVoice])

  useEffect(() => {
    // Start from the actual first audio frame so visual stages do not run ahead
    // while the local WAV is still loading. If autoplay is unavailable, the
    // fallback starts the same timeline after a short load grace period.
    const fallback = window.setTimeout(startTimeline, 450)
    return () => window.clearTimeout(fallback)
  }, [startTimeline])

  if (gone) return null

  const progress = leaving ? 100 : STEP_PROGRESS[Math.min(step, STEP_PROGRESS.length - 1)]
  const label = t(BOOT_STEPS[Math.min(step, BOOT_STEPS.length - 1)].key)
  const progressStyle = { '--bs-progress': progress / 100 } as CSSProperties
  const activeNodes = leaving ? BOOT_STEPS.length : step + 1
  const audioText = audioState === 'live'
    ? 'AUDIO'
    : audioState === 'muted'
      ? 'MUTED'
      : audioState === 'blocked'
        ? 'READY'
        : audioState === 'unavailable'
          ? 'SILENT'
          : 'AUDIO'

  return (
    <div
      className={`bs-splash${leaving ? ' is-leaving' : ''}${reduced.current ? ' is-reduced' : ''}`}
      data-stage={step}
      data-audio-state={audioState}
      style={progressStyle}
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
        src="/boot-gaea-voice-selected.wav"
        preload="auto"
        autoPlay
        aria-hidden="true"
        tabIndex={-1}
        onCanPlay={tryPlayVoice}
        onPlaying={handlePlaying}
        onEnded={() => finish(false)}
        onError={() => setAudioState('blocked')}
      />

      <div className="bs-stage">
        <div className="bs-ambient" aria-hidden="true">
          <div className="bs-grid" />
          <div className="bs-aura" />
          <svg className="bs-contours" viewBox="0 0 1600 960" preserveAspectRatio="xMidYMid slice">
            {CONTOURS.map((d, index) => (
              <path
                key={d}
                className={`bs-contour${index === 5 ? ' bs-contour--accent' : ''}`}
                d={d}
                pathLength={1}
                style={{ animationDelay: `${760 + index * 120}ms` }}
              />
            ))}
          </svg>
          <div className="bs-vignette" />
        </div>

        <header className="bs-topbar">
          <div className="bs-topbar-brand" aria-hidden="true">
            <span className="bs-topbar-pip" />
            <span>GAEA</span>
            <span className="bs-topbar-mode">LOCAL INTELLIGENCE</span>
          </div>
          <div className="bs-topbar-actions">
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
            <button
              ref={skipRef}
              type="button"
              className="bs-skip"
              onClick={skip}
              aria-label={`${t('boot.skip')} (ESC)`}
            >
              <kbd>ESC</kbd>
              <span>{t('boot.skip')}</span>
            </button>
          </div>
        </header>

        <main className="bs-main">
          <div className="bs-core-scene" aria-hidden="true">
            <div className="bs-core-halo" />
            <div className="bs-orbit bs-orbit--wide" />
            <div className="bs-orbit bs-orbit--tight" />
            <div className="bs-core-pulse" key={`pulse-${step}`} />

            <svg className="bs-progress-orbit" viewBox="0 0 400 400">
              <circle className="bs-orbit-track" cx="200" cy="200" r="188" pathLength={100} />
              <circle
                className="bs-progress-stroke"
                cx="200"
                cy="200"
                r="188"
                pathLength={100}
                style={{ strokeDashoffset: 100 - progress }}
              />
            </svg>

            <div className="bs-stage-nodes">
              {BOOT_STEPS.map((item, index) => (
                <span
                  key={item.code}
                  className={`bs-stage-node${index < activeNodes ? ' is-active' : ''}${index === step && !leaving ? ' is-current' : ''}`}
                  style={{ transform: `rotate(${(index * 360) / BOOT_STEPS.length}deg)` }}
                >
                  <i />
                </span>
              ))}
            </div>

            <div className="bs-core-shell">
              <span className="bs-core-sheen" />
              <img src="/favicon.svg" alt="" draggable={false} />
            </div>
            <span className="bs-core-label">E-0001</span>
          </div>

          <h1 id="bs-brand" className="bs-wordmark">GAEA</h1>
          <div className="bs-tagline"><span>EARTHBORN INTELLIGENCE</span></div>

          <div id="bs-status" className="bs-status" role="status" aria-live="polite" aria-atomic="true">
            <span className="bs-status-index">
              {String(step + 1).padStart(2, '0')} / {String(BOOT_STEPS.length).padStart(2, '0')}
            </span>
            <span className="bs-status-rule" />
            <span key={step} className="bs-status-label">{label}</span>
          </div>
        </main>

        <footer className="bs-footer">
          <div
            className="bs-progress"
            role="progressbar"
            aria-label={t('boot.loading')}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={progress}
          >
            <div className="bs-progress-copy">
              <span>{t('boot.loading')}</span>
              <strong>{String(progress).padStart(3, '0')}%</strong>
            </div>
            <div className="bs-progress-track">
              <i />
              <span className="bs-progress-glint" style={{ left: `${progress}%` }} />
            </div>
          </div>
        </footer>
      </div>
    </div>
  )
}

export default BootSplash
