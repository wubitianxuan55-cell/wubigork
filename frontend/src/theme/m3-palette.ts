// frontend/src/theme/m3-palette.ts
// Material Design 3 Tonal Palette 轻量实现
// 参考: https://m3.material.io/styles/color/the-color-system/key-colors-tones

// ── Constants ────────────────────────────────────────────────

/** Binary search iterations for luminance blending */
const BLEND_ITERATIONS = 20

/** Convergence tolerance for luminance matching */
const LUM_TOLERANCE = 0.001

/** M3 标准色调级别 */
const TONAL_STOPS = [0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 99, 100]

// ── Internal helpers ─────────────────────────────────────────

/** sRGB 线性化 */
function linearize(c: number): number {
  const s = c / 255
  return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}

/** RGB → 相对亮度 (Y from XYZ) */
function luminance(r: number, g: number, b: number): number {
  return 0.2126 * linearize(r) + 0.7152 * linearize(g) + 0.0722 * linearize(b)
}

/** Validate and normalise a hex colour; returns lowercase `#rrggbb`. */
function sanitizeHex(hex: string): string {
  const cleaned = hex.replace(/^#/, '')
  if (!/^[0-9a-fA-F]{6}$/.test(cleaned)) {
    throw new Error(`Invalid hex color: ${hex}`)
  }
  return `#${cleaned.toLowerCase()}`
}

/**
 * Blend a hex colour toward pure black or pure white until its
 * relative luminance matches `targetLum` (tone / 100).
 *
 * The binary search chooses the blend direction automatically:
 *  - targetLum > srcLum  → blend toward white (255)
 *  - targetLum < srcLum  → blend toward black (0)
 */
function blendTone(hexColor: string, tone: number): string {
  const r = parseInt(hexColor.slice(1, 3), 16)
  const g = parseInt(hexColor.slice(3, 5), 16)
  const b = parseInt(hexColor.slice(5, 7), 16)

  const targetLum = tone / 100
  const srcLum = luminance(r, g, b)

  if (Math.abs(srcLum - targetLum) < LUM_TOLERANCE) {
    return hexColor
  }

  const goTowardWhite = targetLum > srcLum
  const blendTarget = goTowardWhite ? 255 : 0

  let lo = 0
  let hi = 1
  let mix = 0.5

  for (let i = 0; i < BLEND_ITERATIONS; i++) {
    mix = (lo + hi) / 2
    const mixedR = Math.round(r * (1 - mix) + blendTarget * mix)
    const mixedG = Math.round(g * (1 - mix) + blendTarget * mix)
    const mixedB = Math.round(b * (1 - mix) + blendTarget * mix)
    const mixedLum = luminance(mixedR, mixedG, mixedB)

    // When blending toward white, luminance rises with mix;
    // when blending toward black, luminance falls with mix.
    // The correct update direction depends on goTowardWhite.
    if ((mixedLum < targetLum) === goTowardWhite) {
      lo = mix
    } else {
      hi = mix
    }
  }

  const finalR = Math.round(r * (1 - mix) + blendTarget * mix)
  const finalG = Math.round(g * (1 - mix) + blendTarget * mix)
  const finalB = Math.round(b * (1 - mix) + blendTarget * mix)

  const clamp = (v: number) => Math.min(255, Math.max(0, v))
  return (
    '#' +
    clamp(finalR).toString(16).padStart(2, '0') +
    clamp(finalG).toString(16).padStart(2, '0') +
    clamp(finalB).toString(16).padStart(2, '0')
  )
}

/** Convert a hue angle (0‑360) to a vivid RGB triplet at full chroma. */
function hueToRgb(hue: number): [number, number, number] {
  const h = ((hue % 360) + 360) % 360
  const sector = Math.floor(h / 60) % 6
  const f = h / 60 - sector
  const p = 0
  const q = Math.round(255 * (1 - f))
  const t = Math.round(255 * f)

  switch (sector) {
    case 0:
      return [255, t, p]
    case 1:
      return [q, 255, p]
    case 2:
      return [p, 255, t]
    case 3:
      return [p, q, 255]
    case 4:
      return [t, p, 255]
    case 5:
      return [255, p, q]
    default:
      return [255, 0, 0]
  }
}

// ── Public API ───────────────────────────────────────────────

/**
 * Extract an approximate hue angle (0‑360) from a hex colour.
 * Uses the standard RGB → HSL hue formula.
 */
export function sourceColorFromHex(hex: string): number {
  const cleaned = sanitizeHex(hex)
  const r = parseInt(cleaned.slice(1, 3), 16)
  const g = parseInt(cleaned.slice(3, 5), 16)
  const b = parseInt(cleaned.slice(5, 7), 16)

  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  const delta = max - min

  if (delta === 0) return 0

  let hue = 0
  if (max === r) {
    hue = 60 * (((g - b) / delta) % 6)
  } else if (max === g) {
    hue = 60 * ((b - r) / delta + 2)
  } else {
    hue = 60 * ((r - g) / delta + 4)
  }

  return ((hue % 360) + 360) % 360
}

/**
 * Convert hue, chroma, and tone back to a hex colour using the
 * same simplified luminance-blending approach.
 *
 * @param hue    Hue angle 0‑360
 * @param chroma Colourfulness (0 ≈ gray, ~150 = maximum vividness)
 * @param tone   Tonal stop 0‑100 (luminance = tone / 100)
 */
export function hexFromHct(hue: number, chroma: number, tone: number): string {
  const [vr, vg, vb] = hueToRgb(hue)

  // Scale chroma: 0 → gray, maxChroma → full vividness
  const MAX_CHROMA = 150
  const chromaFactor = Math.min(1, chroma / MAX_CHROMA)

  // Luminance of the vivid hue (used as gray reference)
  const gray = Math.round(0.2126 * vr + 0.7152 * vg + 0.0722 * vb)
  const cr = Math.round(gray + (vr - gray) * chromaFactor)
  const cg = Math.round(gray + (vg - gray) * chromaFactor)
  const cb = Math.round(gray + (vb - gray) * chromaFactor)

  const clamp = (v: number) => Math.min(255, Math.max(0, v))
  const hexColor =
    '#' +
    clamp(cr).toString(16).padStart(2, '0') +
    clamp(cg).toString(16).padStart(2, '0') +
    clamp(cb).toString(16).padStart(2, '0')

  return blendTone(hexColor, tone)
}

/**
 * 从种子 hex 色生成完整 Tonal Palette
 * 返回 13 个 hex 色，对应 TONAL_STOPS 的每个级别
 */
export function generateTonalPalette(seedHex: string): string[] {
  const hex = sanitizeHex(seedHex)
  return TONAL_STOPS.map((tone) => blendTone(hex, tone))
}

/**
 * 获取特定 tone 级别的颜色
 */
export function tonalColor(seedHex: string, tone: number): string {
  const hex = sanitizeHex(seedHex)
  return blendTone(hex, tone)
}

/**
 * M3 默认色调映射（暗色主题基准 — Material Design 3 角色 → tone 值）
 */
export const M3_TONE_MAP = {
  primary: 40,
  onPrimary: 100,
  primaryContainer: 90,
  onPrimaryContainer: 10,
  secondary: 40,
  onSecondary: 100,
  secondaryContainer: 90,
  onSecondaryContainer: 10,
  tertiary: 40,
  onTertiary: 100,
  tertiaryContainer: 90,
  onTertiaryContainer: 10,
  error: 40,
  onError: 100,
  errorContainer: 90,
  onErrorContainer: 10,
  surface: 6, // dark: tone 6, light: tone 98
  onSurface: 90, // dark: tone 90, light: tone 10
  surfaceVariant: 30,
  onSurfaceVariant: 80,
  outline: 60,
  outlineVariant: 30,
  surfaceContainer: 8,
  surfaceContainerHigh: 12,
  surfaceContainerHighest: 16,
  inverseSurface: 90,
  inverseOnSurface: 20,
  inversePrimary: 80,
  surfaceDim: 6,
  surfaceBright: 24,
} as const

/** 亮色主题 tone 映射覆盖 */
export const LIGHT_TONES: Record<string, number> = {
  surface: 98,
  onSurface: 10,
  surfaceContainer: 94,
  surfaceContainerHigh: 92,
  surfaceContainerHighest: 90,
  surfaceDim: 86,
  surfaceBright: 98,
}
