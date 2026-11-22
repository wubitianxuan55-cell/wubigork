/** Dependency-free colour helpers for the imported Bloom OKLCH palette. */

function clamp01(value: number): number {
  return Math.min(1, Math.max(0, value))
}

function linearToSrgb(value: number): number {
  const c = clamp01(value)
  return c <= 0.0031308 ? 12.92 * c : 1.055 * Math.pow(c, 1 / 2.4) - 0.055
}

function byteHex(value: number): string {
  return Math.round(clamp01(value) * 255).toString(16).padStart(2, '0')
}

function oklchValueToHex(lightness: number, chroma: number, hue: number): string {
  const radians = (hue * Math.PI) / 180
  const a = chroma * Math.cos(radians)
  const b = chroma * Math.sin(radians)

  const lPrime = lightness + 0.3963377774 * a + 0.2158037573 * b
  const mPrime = lightness - 0.1055613458 * a - 0.0638541728 * b
  const sPrime = lightness - 0.0894841775 * a - 1.291485548 * b
  const l = lPrime ** 3
  const m = mPrime ** 3
  const s = sPrime ** 3

  const r = 4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s
  const g = -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s
  const blue = -0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s
  return `#${byteHex(linearToSrgb(r))}${byteHex(linearToSrgb(g))}${byteHex(linearToSrgb(blue))}`
}

export function oklchToHex(spec: string): string {
  const match = /oklch\(\s*([\d.]+)%\s+([\d.]+)\s+([\d.]+)\s*\)/i.exec(spec)
  if (!match) return '#000000' // hex-exempt: unreachable fallback for invalid source input
  return oklchValueToHex(Number(match[1]) / 100, Number(match[2]), Number(match[3]))
}

export function hexToRgbTuple(hex: string): [number, number, number] {
  const normalized = hex.replace('#', '')
  const full = normalized.length === 3
    ? normalized.split('').map((part) => part + part).join('')
    : normalized
  if (!/^[0-9a-f]{6}$/i.test(full)) return [0, 0, 0]
  return [
    Number.parseInt(full.slice(0, 2), 16),
    Number.parseInt(full.slice(2, 4), 16),
    Number.parseInt(full.slice(4, 6), 16),
  ]
}

function srgbToLinear(value: number): number {
  const c = value / 255
  return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
}

function hexToOklch(hex: string): { l: number; c: number; h: number } {
  const [rByte, gByte, bByte] = hexToRgbTuple(hex)
  const r = srgbToLinear(rByte)
  const g = srgbToLinear(gByte)
  const b = srgbToLinear(bByte)

  const l = 0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b
  const m = 0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b
  const s = 0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b
  const lRoot = Math.cbrt(l)
  const mRoot = Math.cbrt(m)
  const sRoot = Math.cbrt(s)

  const lightness = 0.2104542553 * lRoot + 0.793617785 * mRoot - 0.0040720468 * sRoot
  const a = 1.9779984951 * lRoot - 2.428592205 * mRoot + 0.4505937099 * sRoot
  const bAxis = 0.0259040371 * lRoot + 0.7827717662 * mRoot - 0.808675766 * sRoot
  const chroma = Math.hypot(a, bAxis)
  let hue = (Math.atan2(bAxis, a) * 180) / Math.PI
  if (hue < 0) hue += 360
  return { l: lightness, c: chroma, h: hue }
}

/**
 * Match CSS `color-mix(in oklch, ...)`: interpolate lightness, chroma and the
 * shortest hue arc instead of blending raw sRGB channels.
 */
export function mixOklchHex(a: string, b: string, weightB: number): string {
  const first = hexToOklch(a)
  const second = hexToOklch(b)
  const weight = clamp01(weightB)
  const firstHue = first.c < 0.0001 ? second.h : first.h
  const secondHue = second.c < 0.0001 ? first.h : second.h
  const hueDelta = ((secondHue - firstHue + 540) % 360) - 180
  return oklchValueToHex(
    first.l + (second.l - first.l) * weight,
    first.c + (second.c - first.c) * weight,
    (firstHue + hueDelta * weight + 360) % 360,
  )
}

export function rgba(hex: string, alpha: number): string {
  const [r, g, b] = hexToRgbTuple(hex)
  return `rgba(${r},${g},${b},${clamp01(alpha)})`
}

export function mixHex(a: string, b: string, weightB: number): string {
  const pa = hexToRgbTuple(a)
  const pb = hexToRgbTuple(b)
  const w = clamp01(weightB)
  const channels = pa.map((value, index) => Math.round(value * (1 - w) + pb[index] * w))
  return `#${channels.map((value) => value.toString(16).padStart(2, '0')).join('')}`
}

function srgbChannel(value: number): number {
  const c = value / 255
  return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
}

export function relativeLuminance(hex: string): number {
  const [r, g, b] = hexToRgbTuple(hex)
  return 0.2126 * srgbChannel(r) + 0.7152 * srgbChannel(g) + 0.0722 * srgbChannel(b)
}

export function contrastRatio(a: string, b: string): number {
  const first = relativeLuminance(a)
  const second = relativeLuminance(b)
  return (Math.max(first, second) + 0.05) / (Math.min(first, second) + 0.05)
}
