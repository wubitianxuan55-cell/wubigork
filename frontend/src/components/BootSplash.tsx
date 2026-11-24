/**
 * BootSplash — gaea startup sequence.
 *
 * Two generated plates (orbit sunrise → alpine valley) carry the cinematic look. A WebGL2
 * layer gives them life — camera push, a dive through the sunrise, breathing bloom, twinkles,
 * lake ripples, city lights, mist, an orbital comet — and draws the glass mark. The wordmark,
 * taglines, status, progress and the startup controls are DOM on top, so they stay crisp at
 * any resolution. Without WebGL2 the same plates simply cross-fade; the voice timeline, ESC
 * skip and the sound control behave identically either way.
 */
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { useT } from '../gaea/lib/i18n'
import {
  BOOT_FINISH_S,
  BOOT_MOTION_FULL,
  BOOT_MOTION_REDUCED,
  BOOT_SCENE_AT_S,
  EXIT_MS,
  REDUCED_EXIT_MS,
  SKIP_EXIT_MS,
  bootHudAt,
  bootPose,
  bootProgressAt,
} from './boot/bootScene'
import { createBootRenderer } from './boot/bootRenderer'
import type { BootHud, BootPose } from './boot/bootScene'
import type { BootPlateSource, BootRenderer } from './boot/bootRenderer'
import './boot-splash.css'

const BOOT_STEPS = [{ key: 'boot.stepCore' }, { key: 'boot.stepReady' }] as const
const FINISH_GUARD_MS = BOOT_FINISH_S * 1000 + 300
/** Grace after the voice's own length before the guard closes the sequence. */
const VOICE_GUARD_MS = 300
/** If autoplay is unavailable the same timeline starts after this load grace period. */
const TIMELINE_FALLBACK_MS = 450

/** index.html paints a blurred thumbnail before the bundle loads; it leaves once the plate is sharp. */
const STATIC_SPLASH_ID = 'boot-splash'
const STATIC_HANDOFF_MS = 300
const STATIC_HANDOFF_GUARD_MS = 3000

const MARK_SRC = '/boot/mark.webp'
const PLATES = ['/boot/plate-orbit.avif', '/boot/plate-valley.avif'] as const
const VOICE_SRC = '/boot-gaea-voice-selected.wav'
/** Brand copy of the approved key art. */
const MOTTO = 'SMARTER · FASTER · TOGETHER'
const TAGLINES = ['YOUR INTELLIGENT WORK ASSISTANT', 'EMPOWER YOUR INTELLIGENCE'] as const

type AudioState = 'starting' | 'live' | 'blocked' | 'muted' | 'unavailable'

const detectReducedMotion = (): boolean =>
  typeof window.matchMedia === 'function' ? window.matchMedia('(prefers-reduced-motion: reduce)').matches : false

/** Device pixels to draw at: native resolution, capped at 4K on the long edge. */
function targetSize(el: HTMLElement): [number, number] {
  const ratio = Math.min(window.devicePixelRatio || 1, 2)
  const w = el.clientWidth * ratio
  const h = el.clientHeight * ratio
  const cap = Math.min(1, 3840 / Math.max(w, h, 1))
  return [Math.max(2, Math.round(w * cap)), Math.max(2, Math.round(h * cap))]
}

/** Reads an `r, g, b` custom property (0–255) so no colour lives in this file. */
function readRgb(el: HTMLElement, name: string, fallback: string): [number, number, number] {
  const parts = (getComputedStyle(el).getPropertyValue(name).trim() || fallback)
    .split(',')
    .map((part) => Number.parseFloat(part))
  if (parts.length !== 3 || parts.some((n) => !Number.isFinite(n))) return [0.2, 0.9, 0.86]
  return [parts[0] / 255, parts[1] / 255, parts[2] / 255]
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error(`boot asset failed to load: ${src}`))
    img.src = src
  })
}

/** Plates ship at 4K. */
const PLATE_WIDTH = 3840
const PLATE_HEIGHT = 2160
/** Never decode below 720p, whatever a broken measurement says. */
const MIN_PLATE_EDGE = 1280

/**
 * Decodes a plate off the main thread and scales it to what the window can show: a 1080p window
 * never pays for 4K pixels (and samples mip level 0, so it stays sharp), a 4K window gets all of them.
 */
async function loadPlate(src: string, maxEdge: number): Promise<BootPlateSource> {
  if (typeof createImageBitmap === 'function' && typeof fetch === 'function') {
    try {
      const blob = await (await fetch(src)).blob()
      const ratio = Math.min(1, Math.max(MIN_PLATE_EDGE, maxEdge) / PLATE_WIDTH)
      return await createImageBitmap(blob, {
        resizeWidth: Math.round(PLATE_WIDTH * ratio),
        resizeHeight: Math.round(PLATE_HEIGHT * ratio),
        resizeQuality: 'high',
      })
    } catch {
      /* fall through to the plain image path */
    }
  }
  return loadImage(src)
}

/** Frees a decoded plate once its pixels live on the GPU. */
function releaseSource(source: BootPlateSource): void {
  const bitmap = source as ImageBitmap
  if (typeof bitmap.close === 'function') bitmap.close()
}

/** rAF when the platform has it, a 16ms timer otherwise. */
function scheduleFrame(fn: () => void): number {
  return typeof window.requestAnimationFrame === 'function'
    ? window.requestAnimationFrame(fn)
    : window.setTimeout(fn, 16)
}
function cancelFrame(id: number): void {
  if (!id) return
  if (typeof window.cancelAnimationFrame === 'function') window.cancelAnimationFrame(id)
  else window.clearTimeout(id)
}

/** True while the voice element is actually playing (a rejected autoplay or a test DOM never is). */
function isVoiceClock(voice: HTMLAudioElement): boolean {
  return !voice.paused && !voice.ended && voice.readyState >= 2 && voice.duration > 0
}

const BootSplashSequence: React.FC<{ onGone: () => void }> = ({ onGone }) => {
  const t = useT()
  const [scene, setScene] = useState(0)
  const [leaving, setLeaving] = useState(false)
  const [live, setLive] = useState(false)
  const [muted, setMuted] = useState(false)
  const [audioState, setAudioState] = useState<AudioState>('starting')
  const [reduced] = useState(detectReducedMotion)

  const rootRef = useRef<HTMLDivElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const pctRef = useRef<HTMLElement>(null)
  const progressRef = useRef<HTMLDivElement>(null)
  const voiceRef = useRef<HTMLAudioElement>(null)
  const soundRef = useRef<HTMLButtonElement>(null)
  const skipRef = useRef<HTMLButtonElement>(null)
  const rendererRef = useRef<BootRenderer | null>(null)
  const releaseStatic = useRef<(() => void) | null>(null)
  const timers = useRef<number[]>([])
  const timelineStarted = useRef(false)
  const ending = useRef(false)
  const mutedRef = useRef(false)
  /** startAt / finishAt are Date.now() stamps; negative means "not yet". `frozen` is the last timeline time shown. */
  const clock = useRef({ startAt: -1, finishAt: -1, exitMs: EXIT_MS, frozen: 0 })
  const sceneShown = useRef(0)
  const voiceGuard = useRef(false)
  const size = useRef<[number, number]>([2, 2])

  const pushTimer = useCallback((fn: () => void, ms: number) => {
    timers.current.push(window.setTimeout(fn, ms))
  }, [])

  const clearTimers = useCallback(() => {
    timers.current.forEach((id) => window.clearTimeout(id))
    timers.current = []
  }, [])

  const applyHud = useCallback((hud: BootHud, percent: number, pose: BootPose) => {
    const el = rootRef.current
    if (!el) return
    const style = el.style
    // The static plates ride the same camera push, so the WebGL layer takes over without a jump.
    style.setProperty('--bs-push', pose.cam1[0].toFixed(4))
    style.setProperty('--bs-push2', pose.cam2[0].toFixed(4))
    style.setProperty('--bs-chrome', hud.chrome.toFixed(3))
    style.setProperty('--bs-wm', hud.wordmark.toFixed(3))
    style.setProperty('--bs-trk', hud.track.toFixed(3))
    style.setProperty('--bs-tag1', hud.tagline1.toFixed(3))
    style.setProperty('--bs-tag2', hud.tagline2.toFixed(3))
    style.setProperty('--bs-prog', hud.progress.toFixed(4))
    const text = `${percent}%`
    if (pctRef.current && pctRef.current.textContent !== text) pctRef.current.textContent = text
    progressRef.current?.setAttribute('aria-valuenow', String(percent))
  }, [])


  // WebGL layer: attaches whenever the textures are in, even mid-sequence.
  useEffect(() => {
    const canvas = canvasRef.current
    const root = rootRef.current
    if (!canvas || !root) return
    let disposed = false
    const onLost = () => setLive(false)
    canvas.addEventListener('webglcontextlost', onLost)
    // Measure here rather than trusting another effect's order: the size ref still holds its
    // 2×2 placeholder at this point. The window's own resolution is the target — 4K on a 4K display.
    const maxEdge = Math.max(...targetSize(root))
    void Promise.all([loadImage(MARK_SRC), loadPlate(PLATES[0], maxEdge), loadPlate(PLATES[1], maxEdge)])
      .then(([mark, orbit, valley]) => {
        if (disposed) {
          releaseSource(orbit)
          releaseSource(valley)
          return
        }
        rendererRef.current = createBootRenderer(
          canvas,
          { mark, orbit, valley },
          {
            accent: readRgb(root, '--bs-accent-rgb', '50, 228, 219'),
            gold: readRgb(root, '--bs-gold-rgb', '240, 194, 122'),
          },
        )
        releaseSource(orbit)
        releaseSource(valley)
        if (rendererRef.current) setLive(true)
      })
      .catch(() => {
        /* WebGL layer unavailable — the plates keep cross-fading underneath. */
      })
    return () => {
      disposed = true
      canvas.removeEventListener('webglcontextlost', onLost)
      rendererRef.current?.dispose()
      rendererRef.current = null
    }
  }, [])

  // While index.html's blurred thumbnail is still on top, this layer stays see-through so
  // startup never flashes an empty frame. It leaves once the orbit plate is sharp (never later
  // than the guard) and cross-fades into the sharp plate.
  const [overStatic, setOverStatic] = useState(() => Boolean(document.getElementById(STATIC_SPLASH_ID)))
  useEffect(() => {
    const staticSplash = document.getElementById(STATIC_SPLASH_ID)
    if (!staticSplash) return
    let released = false
    const release = () => {
      if (released) return
      released = true
      setOverStatic(false)
      staticSplash.classList.add('is-handing-off')
      window.setTimeout(() => staticSplash.remove(), STATIC_HANDOFF_MS)
    }
    releaseStatic.current = release
    const guard = window.setTimeout(release, STATIC_HANDOFF_GUARD_MS)
    return () => {
      window.clearTimeout(guard)
      releaseStatic.current = null
    }
  }, [])

  const finish = useCallback(
    (skipped: boolean) => {
      if (ending.current) return
      ending.current = true
      clearTimers()
      const state = clock.current
      if (state.startAt < 0) state.startAt = Date.now()
      state.finishAt = Date.now()
      state.exitMs = reduced ? REDUCED_EXIT_MS : skipped ? SKIP_EXIT_MS : EXIT_MS
      setLeaving(true)
      const voice = voiceRef.current
      if (voice) {
        voice.pause()
        voice.currentTime = 0
      }
      pushTimer(() => {
        rendererRef.current?.dispose()
        rendererRef.current = null
        onGone()
      }, state.exitMs)
    },
    [clearTimers, onGone, pushTimer, reduced],
  )

  const skip = useCallback(() => finish(true), [finish])

  const startTimeline = useCallback(() => {
    if (timelineStarted.current || ending.current) return
    timelineStarted.current = true
    clock.current.startAt = Date.now()
    // The voice's ended event is the normal close. This guard only covers a platform where
    // playback starts but the ended event never arrives.
    pushTimer(() => finish(false), FINISH_GUARD_MS)
  }, [finish, pushTimer])

  // One loop drives both layers from the same clock, so the picture and the HUD never drift.
  useEffect(() => {
    let handle = 0
    const frame = () => {
      handle = 0
      const now = Date.now()
      const state = clock.current
      const voice = voiceRef.current
      let time = 0
      let exit = 0
      if (state.finishAt >= 0) {
        // Leaving: the picture holds where it was handed over while it fades out.
        time = state.frozen
        exit = Math.min(1, (now - state.finishAt) / state.exitMs)
      } else if (voice && isVoiceClock(voice)) {
        // While the voice plays it is the clock. The audio thread keeps perfect time even when
        // the main thread is too busy to deliver `playing` promptly, so the scene change stays
        // on the second phrase and the close stays on the last syllable.
        time = state.frozen = Math.min(BOOT_FINISH_S, voice.currentTime)
      } else if (state.startAt >= 0) {
        time = state.frozen = Math.min(BOOT_FINISH_S, (now - state.startAt) / 1000)
      }
      if (time >= BOOT_SCENE_AT_S[1] && sceneShown.current === 0) {
        sceneShown.current = 1
        setScene(1)
      }
      if (voice && state.finishAt < 0 && isVoiceClock(voice) && !voiceGuard.current) {
        // The voice is in charge now: re-arm the close guard on its remaining length, so the
        // fallback's wall-clock guard can never cut the last phrase.
        voiceGuard.current = true
        clearTimers()
        pushTimer(() => finish(false), Math.max(0, (voice.duration - voice.currentTime) * 1000) + VOICE_GUARD_MS)
      }
      const pose = bootPose(time, exit, reduced ? BOOT_MOTION_REDUCED : BOOT_MOTION_FULL)
      rendererRef.current?.render(size.current[0], size.current[1], pose)
      applyHud(bootHudAt(time, exit), Math.round(bootProgressAt(time, exit)), pose)
      handle = scheduleFrame(frame)
    }
    const measure = () => {
      if (rootRef.current) size.current = targetSize(rootRef.current)
    }
    measure()
    const observer = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(measure)
    if (observer && rootRef.current) observer.observe(rootRef.current)
    handle = scheduleFrame(frame)
    return () => {
      cancelFrame(handle)
      observer?.disconnect()
    }
  }, [applyHud, clearTimers, finish, pushTimer, reduced])

  const tryPlayVoice = useCallback(() => {
    const voice = voiceRef.current
    if (!voice) return
    voice.muted = mutedRef.current
    voice.volume = 0.92
    if (ending.current || !voice.paused || voice.readyState < 2) return
    try {
      const playing = voice.play()
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
    const voice = voiceRef.current
    const playing = Boolean(voice && !voice.paused && !voice.ended)

    if (mutedRef.current || !playing) {
      mutedRef.current = false
      setMuted(false)
      if (voice) voice.muted = false
      if (playing && voice && voice.readyState >= 2) {
        setAudioState('live')
        return
      }
      setAudioState('starting')
      tryPlayVoice()
      return
    }

    mutedRef.current = true
    setMuted(true)
    if (voice) voice.muted = true
    setAudioState('muted')
  }, [tryPlayVoice])

  const handlePlaying = useCallback(() => {
    setAudioState(mutedRef.current ? 'muted' : 'live')
    startTimeline()
  }, [startTimeline])

  useEffect(() => {
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
        const next =
          current < 0 ? controls[0] : controls[(current + direction + controls.length) % controls.length]
        next.focus({ preventScroll: true })
      }
    }

    window.addEventListener('keydown', onKeyDown, true)
    // Start from the actual first audio frame so the stages do not run ahead while the local
    // WAV is still loading. If autoplay is unavailable, the fallback starts the same timeline.
    const fallback = window.setTimeout(startTimeline, TIMELINE_FALLBACK_MS)
    return () => {
      window.removeEventListener('keydown', onKeyDown, true)
      window.clearTimeout(fallback)
      clearTimers()
    }
  }, [clearTimers, skip, startTimeline])

  const label = t(BOOT_STEPS[Math.min(scene, BOOT_STEPS.length - 1)].key)
  const audioText =
    audioState === 'live'
      ? 'AUDIO'
      : audioState === 'muted'
        ? 'MUTED'
        : audioState === 'blocked' || audioState === 'starting'
          ? 'PLAY'
          : audioState === 'unavailable'
            ? 'SILENT'
            : 'AUDIO'
  const soundLabel = muted || audioState !== 'live' ? t('boot.soundOff') : t('boot.soundOn')

  return (
    <div
      ref={rootRef}
      className={`bs-splash${leaving ? ' is-leaving' : ''}${live ? ' is-live' : ''}${reduced ? ' is-reduced' : ''}${overStatic ? ' is-over-static' : ''}`}
      data-stage={scene}
      data-scene={scene === 0 ? 'earth' : 'gaia'}
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
        src={VOICE_SRC}
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
        {/* Static plates: the base under the canvas and the whole show without WebGL2. */}
        <img
          className="bs-poster bs-poster--earth"
          src={PLATES[0]}
          alt=""
          draggable={false}
          data-testid="boot-poster-earth"
          onLoad={() => releaseStatic.current?.()}
          onError={() => releaseStatic.current?.()}
        />
        <img
          className="bs-poster bs-poster--gaia"
          src={PLATES[1]}
          alt=""
          draggable={false}
          data-testid="boot-poster-gaia"
        />
        <canvas ref={canvasRef} className="bs-canvas" />

        <div className="bs-deco" aria-hidden="true">
          <div className="bs-lock">
            <img src={MARK_SRC} alt="" draggable={false} />
            <i />
            <span>GAEA</span>
          </div>
          <div className="bs-motto">{MOTTO}</div>
          <div className="bs-brand">
            <div className="bs-word">GAEA</div>
            <div className="bs-tags">
              <div className="bs-tag" data-k="1">
                <i />
                <span>{TAGLINES[0]}</span>
                <i />
              </div>
              <div className="bs-tag" data-k="2">
                <i />
                <span>{TAGLINES[1]}</span>
                <i />
              </div>
            </div>
          </div>
        </div>

        {/* A real control, so it sits outside the decorative layer. */}
        <div className="bs-prog">
          <div
            ref={progressRef}
            className="bs-track"
            role="progressbar"
            aria-label={t('boot.loading')}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={0}
          >
            <div className="bs-fill" />
            <div className="bs-head" />
          </div>
          <b className="bs-pct" ref={pctRef}>
            0%
          </b>
        </div>

        <h1 id="bs-brand" className="bs-sr-only">
          GAEA
        </h1>
        <div id="bs-status" className="bs-status" role="status" aria-live="polite" aria-atomic="true">
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
            <span className="bs-audio-bars" aria-hidden="true">
              <i />
              <i />
              <i />
            </span>
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

const BootSplash: React.FC = () => {
  const [gone, setGone] = useState(false)
  const onGone = useCallback(() => setGone(true), [])
  return gone ? null : <BootSplashSequence onGone={onGone} />
}

export default BootSplash
