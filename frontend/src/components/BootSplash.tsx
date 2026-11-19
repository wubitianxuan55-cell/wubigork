/**
 * BootSplash - GAEA "Earthseed" startup sequence.
 *
 * The approved full-frame scenes are used directly. The sequence opens above
 * Earth, then dissolves into the grounded Gaia landscape as the voice enters
 * its second phrase. Only functional controls are drawn above the artwork.
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
const STEP_PROGRESS = [68, 72] as const
const FINISH_MS = 4000
const FINISH_GUARD_MS = FINISH_MS + 300
const EXIT_MS = 520
const SKIP_EXIT_MS = 420
const REDUCED_EXIT_MS = 180

type AudioState = 'starting' | 'live' | 'blocked' | 'muted' | 'unavailable'
const BootSplash: React.FC = () => {
  const t = useT()
  const [step, setStep] = useState(0)
  const [visualProgress, setVisualProgress] = useState(0)
  const [leaving, setLeaving] = useState(false)
  const [gone, setGone] = useState(false)
  const [muted, setMuted] = useState(false)
  const [audioState, setAudioState] = useState<AudioState>('starting')
  const reduced = useRef(false)
  const timers = useRef<number[]>([])
  const timelineStarted = useRef(false)
  const ending = useRef(false)
  const mutedRef = useRef(false)
  const visualProgressRef = useRef(0)
  const progressFrame = useRef<number | null>(null)
  const voiceRef = useRef<HTMLAudioElement>(null)
  const soundRef = useRef<HTMLButtonElement>(null)
  const skipRef = useRef<HTMLButtonElement>(null)

  const clearTimers = useCallback(() => {
    timers.current.forEach((id) => window.clearTimeout(id))
    timers.current = []
  }, [])

  const animateProgressTo = useCallback((target: number, duration = 760) => {
    if (progressFrame.current !== null) {
      window.cancelAnimationFrame(progressFrame.current)
      progressFrame.current = null
    }
    const from = visualProgressRef.current
    if (reduced.current || duration <= 0 || typeof window.requestAnimationFrame !== 'function') {
      visualProgressRef.current = target
      setVisualProgress(target)
      return
    }

    const startedAt = performance.now()
    const tick = (now: number) => {
      const t = Math.min(1, (now - startedAt) / duration)
      const eased = 1 - Math.pow(1 - t, 3)
      const next = Math.round(from + (target - from) * eased)
      visualProgressRef.current = next
      setVisualProgress(next)
      if (t < 1) {
        progressFrame.current = window.requestAnimationFrame(tick)
      } else {
        progressFrame.current = null
      }
    }
    progressFrame.current = window.requestAnimationFrame(tick)
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
    animateProgressTo(100, exitMs)
    timers.current.push(window.setTimeout(() => setGone(true), exitMs))
  }, [animateProgressTo, clearTimers])

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
      if (at === 0) {
        setStep(index)
        animateProgressTo(STEP_PROGRESS[index], 1100)
      } else {
        push(() => {
          setStep(index)
          animateProgressTo(STEP_PROGRESS[index])
        }, at)
      }
    })
    // The media ended event is the normal close. This guard only covers a
    // platform where playback starts but the ended event never arrives.
    push(() => finish(false), FINISH_GUARD_MS)
  }, [animateProgressTo, finish])

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
    const voiceEl = voiceRef.current
    const playing = Boolean(voiceEl && !voiceEl.paused && !voiceEl.ended)

    if (mutedRef.current || !playing) {
      mutedRef.current = false
      setMuted(false)
      if (voiceEl) voiceEl.muted = false
      if (playing && voiceEl && voiceEl.readyState >= 2) {
        setAudioState('live')
        return
      }
      setAudioState('starting')
      tryPlayVoice()
      return
    }

    mutedRef.current = true
    setMuted(true)
    if (voiceEl) voiceEl.muted = true
    if (voiceEl && !voiceEl.paused) {
      setAudioState('muted')
    } else {
      setAudioState('muted')
    }
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
      if (progressFrame.current !== null) window.cancelAnimationFrame(progressFrame.current)
      clearTimers()
    }
  }, [clearTimers, skip])

  useEffect(() => {
    // Start from the actual first audio frame so visual stages do not run ahead
    // while the local WAV is still loading. If autoplay is unavailable, the
    // fallback starts the same timeline after a short load grace period.
    const fallback = window.setTimeout(startTimeline, 450)
    return () => window.clearTimeout(fallback)
  }, [startTimeline])

  if (gone) return null

  const progress = leaving ? 100 : STEP_PROGRESS[Math.min(step, STEP_PROGRESS.length - 1)]
  const scene = step === 0 ? 'earth' : 'gaia'
  const label = t(BOOT_STEPS[Math.min(step, BOOT_STEPS.length - 1)].key)
  const progressStyle = { '--bs-progress': visualProgress / 100 } as CSSProperties
  const audioText = audioState === 'live'
    ? 'AUDIO'
    : audioState === 'muted'
      ? 'MUTED'
      : audioState === 'blocked' || audioState === 'starting'
        ? 'PLAY'
        : audioState === 'unavailable'
          ? 'SILENT'
          : 'AUDIO'
  const soundLabel = muted || audioState !== 'live'
    ? t('boot.soundOff')
    : t('boot.soundOn')
  const tagline = scene === 'earth'
    ? 'YOUR INTELLIGENT WORK ASSISTANT'
    : 'EMPOWER YOUR INTELLIGENCE'

  return (
    <div
      className={`bs-splash${leaving ? ' is-leaving' : ''}${reduced.current ? ' is-reduced' : ''}`}
      data-stage={step}
      data-scene={scene}
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
        <div className="bs-frame">
          <div className="bs-original-scenes" aria-hidden="true">
            <div className="bs-original-scene bs-original-scene--earth">
              <img src="/boot-earthrise-original.png" alt="" draggable={false} />
            </div>
            <div className="bs-original-scene bs-original-scene--gaia">
              <img src="/boot-gaia-original.png" alt="" draggable={false} />
            </div>
          </div>
          <div
            className="bs-progress-live"
            role="progressbar"
            aria-label={t('boot.loading')}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={progress}
            style={progressStyle}
          >
            <div className="bs-progress-live-track">
              <i />
              <span style={{ left: `${visualProgress}%` }} />
            </div>
            <strong>{visualProgress}%</strong>
          </div>
        </div>

        <h1 id="bs-brand" className="bs-sr-only">GAEA</h1>
        <div className="bs-sr-only">{tagline}</div>
        <div id="bs-status" className="bs-sr-only" role="status" aria-live="polite" aria-atomic="true">
          {label}
        </div>
        <div className="bs-utility">
          <button
            ref={soundRef}
            type="button"
            className={`bs-audio is-${audioState}`}
            onClick={toggleSound}
            aria-pressed={!muted && audioState === 'live'}
            aria-label={soundLabel}
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
      </div>
    </div>
  )
}

export default BootSplash
