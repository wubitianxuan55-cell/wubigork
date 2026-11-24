/**
 * BootSplash timeline — pure functions of time.
 *
 * Everything the WebGL layer and the HUD draw is derived here, so any moment of the
 * sequence can be computed (and unit-tested) without a browser:
 *   t     seconds since the voice timeline started (0 → BOOT_FINISH_S)
 *   exit  0..1 progress of the leave animation (0 while the sequence runs)
 */

// Selected voice: clear, intimate and dry. The two phrases split at 1.596s.
// Keep these boundaries in sync with frontend/public/boot-gaea-voice-selected.wav.
export const BOOT_SCENE_AT_S = [0, 1.596] as const

/** The camera dives through the sunrise a few frames after the second phrase begins. */
export const BOOT_DIVE_AT_S = 1.6
export const BOOT_DIVE_S = 0.85
export const BOOT_FINISH_S = 4.0
export const EXIT_MS = 520
export const SKIP_EXIT_MS = 420
export const REDUCED_EXIT_MS = 180

export interface BootMotion {
  /** 0..1 — camera push, parallax and the dive through the sunrise */
  cam: number
  /** 0..1 — ambient life: twinkles, ripples, city lights, mist, comet wobble */
  amb: number
}

export const BOOT_MOTION_FULL: BootMotion = { cam: 1, amb: 1 }
export const BOOT_MOTION_REDUCED: BootMotion = { cam: 0, amb: 0 }

/** [zoom, panX, panY, parallax] — the zoom is anchored on the sun of each plate. */
export type BootCamera = [number, number, number, number]

export interface BootPose {
  t: number
  /** 0 = orbit plate, 1 = valley plate */
  scene: number
  /** warm whiteout at the peak of the dive */
  flash: number
  cam1: BootCamera
  cam2: BootCamera
  amb: number
  logoA: number
  logoBoost: number
  sweep: number
  comet: number
  ringA: number
}

export interface BootHud {
  /** corner lockup, motto, progress and status */
  chrome: number
  wordmark: number
  /** 1 = wide tracking, 0 = settled */
  track: number
  tagline1: number
  tagline2: number
  /** 0..1 */
  progress: number
}

const clamp01 = (x: number): number => (x < 0 ? 0 : x > 1 ? 1 : x)
const smoothstep = (a: number, b: number, x: number): number => {
  const k = clamp01((x - a) / (b - a))
  return k * k * (3 - 2 * k)
}
const easeOutCubic = (x: number): number => 1 - Math.pow(1 - clamp01(x), 3)
const easeInOutCubic = (x: number): number => {
  const k = clamp01(x)
  return k < 0.5 ? 4 * k * k * k : 1 - Math.pow(-2 * k + 2, 3) / 2
}
const easeInQuad = (x: number): number => clamp01(x) * clamp01(x)

/**
 * Progress keeps the approved 68% → 72% → 100% beats. It is paced by the timeline;
 * wiring it to real boot events is a separate change.
 */
export function bootProgressAt(t: number, exit: number): number {
  const e = clamp01(exit)
  if (e > 0) return 94 + 6 * easeOutCubic(e)
  if (t < 1.1) return 68 * easeOutCubic(t / 1.1)
  if (t < 1.6) return 68 + (2 * (t - 1.1)) / 0.5
  if (t < 2.3) return 70 + 2 * easeInOutCubic((t - 1.6) / 0.7)
  return Math.min(94, 72 + (22 * (t - 2.3)) / 1.7)
}

export function bootPose(t: number, exit: number, motion: BootMotion): BootPose {
  const still = motion.cam <= 0
  const dive = still
    ? clamp01((t - BOOT_SCENE_AT_S[1]) / 0.2)
    : easeInOutCubic((t - BOOT_DIVE_AT_S) / BOOT_DIVE_S)
  const e = clamp01(exit)
  const settle = easeOutCubic((t - 1.95) / 0.95)
  const push = easeOutCubic(t / 1.6)
  const drift = clamp01((t - 2.9) / 1.1)
  return {
    t,
    scene: still ? dive : smoothstep(0.4, 0.74, dive),
    flash: still ? 0 : Math.exp(-(((dive - 0.55) / 0.15) ** 2)) * 0.94,
    cam1: [
      (1 + 0.07 * motion.cam * push) * (1 + 2.4 * motion.cam * dive * dive),
      0,
      0,
      0.3 * motion.cam,
    ],
    cam2: [1 + motion.cam * (0.09 * (1 - settle) + 0.025 * drift), 0, 0, 0.3 * motion.cam],
    amb: motion.amb,
    logoA: smoothstep(0.28, 0.95, t) * (1 - smoothstep(0, 0.36, e)),
    logoBoost:
      1 +
      0.55 * Math.exp(-(((t - 0.72) / 0.2) ** 2)) +
      0.35 * Math.exp(-(((t - 2.55) / 0.22) ** 2)),
    sweep: t < 2.2 ? -0.7 + 2.6 * clamp01((t - 0.38) / 0.75) : -0.7 + 2.6 * clamp01((t - 2.35) / 0.75),
    comet: 0.62 - 0.42 * t,
    ringA: smoothstep(0.45, 1.05, t) * (1 - easeInQuad(e)),
  }
}

export function bootHudAt(t: number, exit: number): BootHud {
  const e = clamp01(exit)
  return {
    chrome: smoothstep(0.08, 0.55, t) * (1 - e),
    wordmark: smoothstep(0.55, 1.05, t) * (1 - e),
    track: 1 - easeOutCubic((t - 0.55) / 0.75),
    tagline1: smoothstep(0.95, 1.4, t) * (1 - smoothstep(1.62, 1.84, t)),
    tagline2: smoothstep(2.3, 2.75, t) * (1 - e),
    progress: bootProgressAt(t, e) / 100,
  }
}
