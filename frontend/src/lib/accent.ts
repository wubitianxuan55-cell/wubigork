// accent.ts — 自定义强调色的亮态对比度保障（v4.274）
// 背景：强调色自定义（gaea-accent）不分明暗地覆盖主题 glow/colorPrimary。用户在
// 暗色下调亮的强调色（如 #1dd7bf）切到亮色主题后压浅底对比只剩 ~1.5，全站
// accent 文字/徽标/链接看不清。本模块在亮态渲染时对强调色做自动深化（WCAG AA
// 4.5:1 对白底），暗态原样；用户存储的原始色不变，仅渲染时修正。
// 纯函数：hex → hex，无依赖，可单测。

/** #rrggbb → [r,g,b] (0-255)；非法输入返回 null */
export function hexToRgb256(hex: string): [number, number, number] | null {
  const m = /^#?([0-9a-fA-F]{6})$/.exec(hex.trim())
  if (!m) return null
  const n = parseInt(m[1], 16)
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

function rgbToHex(r: number, g: number, b: number): string {
  const f = (v: number) => Math.max(0, Math.min(255, Math.round(v))).toString(16).padStart(2, '0')
  return `#${f(r)}${f(g)}${f(b)}`
}

/** WCAG 相对亮度 */
function luminance(r: number, g: number, b: number): number {
  const f = (v: number) => {
    const c = v / 255
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
  }
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b)
}

/** 对白底的 WCAG 对比度 */
export function contrastOnWhite(hex: string): number {
  const rgb = hexToRgb256(hex)
  if (!rgb) return 0
  const l = luminance(rgb[0], rgb[1], rgb[2])
  return 1.05 / (l + 0.05)
}

/**
 * 亮态强调色深化：对白底对比不足 4.5:1 时，迭代压 HSL 亮度直到达标
 * （色相/饱和度保持，观感仍是用户选的同色系）。已达标则原样返回。
 * 非法输入原样返回（不猜色）。
 */
export function ensureLightContrast(hex: string, target = 4.5): string {
  const rgb = hexToRgb256(hex)
  if (!rgb) return hex
  if (contrastOnWhite(hex) >= target) return hex
  // RGB → HSL
  const [r0, g0, b0] = rgb.map((v) => v / 255)
  const max = Math.max(r0, g0, b0)
  const min = Math.min(r0, g0, b0)
  const l = (max + min) / 2
  let h = 0
  let s = 0
  if (max !== min) {
    const d = max - min
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min)
    if (max === r0) h = ((g0 - b0) / d + (g0 < b0 ? 6 : 0)) / 6
    else if (max === g0) h = ((b0 - r0) / d + 2) / 6
    else h = ((r0 - g0) / d + 4) / 6
  }
  // 迭代压亮度（每步 ×0.85），下限 0.12 防纯黑
  let li = l
  for (let i = 0; i < 12 && li > 0.12; i++) {
    li *= 0.85
    const q = li < 0.5 ? li * (1 + s) : li + s - li * s
    const pv = 2 * li - q
    const hk = (t: number) => {
      let tt = t
      if (tt < 0) tt += 1
      if (tt > 1) tt -= 1
      if (tt < 1 / 6) return pv + (q - pv) * 6 * tt
      if (tt < 1 / 2) return q
      if (tt < 2 / 3) return pv + (q - pv) * (2 / 3 - tt) * 6
      return pv
    }
    const rr = Math.round(hk(h + 1 / 3) * 255)
    const gg = Math.round(hk(h) * 255)
    const bb = Math.round(hk(h - 1 / 3) * 255)
    const cand = rgbToHex(rr, gg, bb)
    if (contrastOnWhite(cand) >= target) return cand
  }
  // 兜底：极低饱和色（灰）压亮度到不了 4.5 时给达标深灰
  return '#5a5a5a' // hex-exempt 纯函数输出值：非 UI 声明处的颜色计算，无主题 token 可用
}
