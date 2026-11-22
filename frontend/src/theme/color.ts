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

export function oklchToHex(spec: string): string {
  const match = /oklch\(\s*([\d.]+)%\s+([\d.]+)\s+([\d.]+)\s*\)/i.exec(spec)
  if (!match) return '#000000' // hex-exempt: unreachable fallback for invalid source input

  const lightness = Number(match[1]) / 100
  const chroma = Number(match[2])
  const hue = (Number(match[3]) * Math.PI) / 180
  const a = chroma * Math.cos(hue)
  const b = chroma * Math.sin(hue)

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
