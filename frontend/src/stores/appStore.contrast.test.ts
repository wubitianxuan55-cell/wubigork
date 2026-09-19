// appStore.contrast.test.ts — 亮/暗两态的**令牌对比度契约**（v4.349 前端优化刀）
//
// Why：v4.349 前的亮态令牌直接沿用 Tailwind-600 亮阶，实测（WCAG 2.1 相对亮度）
// colorWarning #d97706 在本预设 surface 上仅 2.90–3.13:1、colorSuccess #059669
// 3.44–3.77:1、Jade/Rose/Amber 的 colorPrimary 2.99–3.98:1——亮态下 accent /
// 成功 / 警示文字（不只是装饰：character-library.css:35 计数标签、
// WhisperTracePanel 状态文字）低于 AA 4.5:1。此前无任何机检，靠人工目视。
//
// How：本用例把「正文语义令牌 × 实际承载背景」做全 12 主题矩阵断言，
// 阈值 4.5:1（AA 正文）。色板改动若退回亮阶会立刻红——比注释可靠。
//
// 判据边界（刻意不扩大）：
//   - 断言背景 = 本预设 surface + colorBgContainer(#fff)，即正文实际落点；
//     不覆盖 surfaceContainerHighest 等**高亮容器层**（它们是块背景，正文不直接
//     压在上面；success/warning 在该层会低于 4.5，属已知且非正文场景）。
//   - 装饰性色（outline/glassBg/auroraBg 渐变）不参与断言。
//   - 「印泥朱印章」.w-seal #d06055 为品牌记号豁免：20px/700 粗体属 WCAG
//     large text（阈值 3:1），白底 3.83:1 达标，故不改色（避免品牌色漂移）。
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { getThemeTokens, type ThemePreset } from './appStore'

const PRESETS: ThemePreset[] = ['nightJade', 'nightViolet', 'nightRose', 'nightAmber', 'nightMoss', 'nightSlate']

// ── WCAG 2.1 相对亮度 / 对比度 ──────────────────────────────────────────
function srgbChannel(v: number): number {
  const c = v / 255
  return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
}
function luminance(hex: string): number {
  const h = hex.trim().replace('#', '')
  const full = h.length === 3 ? h.split('').map((c) => c + c).join('') : h
  const [r, g, b] = [0, 2, 4].map((i) => parseInt(full.slice(i, i + 2), 16))
  return 0.2126 * srgbChannel(r) + 0.7152 * srgbChannel(g) + 0.0722 * srgbChannel(b)
}
/** 对比度（1~21）。参数为 #rgb/#rrggbb。 */
function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

/** 正文语义令牌（必须过 AA 4.5:1 的那些）。 */
const BODY_TOKENS = ['colorText', 'colorTextSecondary', 'colorPrimary', 'colorSuccess', 'colorWarning'] as const

describe('令牌对比度契约（12 主题 × 正文语义令牌 × 承载背景 ≥ 4.5:1）', () => {
  for (const preset of PRESETS) {
    for (const dark of [true, false]) {
      it(`${preset}${dark ? 'D' : 'L'}：正文令牌在 surface / colorBgContainer 上达 AA`, () => {
        const t = getThemeTokens(preset, dark)
        const backgrounds: [string, string][] = [
          ['surface', t.surface],
          ['colorBgContainer', t.colorBgContainer],
        ]
        for (const tok of BODY_TOKENS) {
          for (const [bgName, bg] of backgrounds) {
            expect(bg, `${preset}${dark ? 'D' : 'L'}.${bgName} 必须是 hex`).toMatch(/^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/)
            const r = contrast(t[tok], bg)
            expect(
              r,
              `${preset}${dark ? 'D' : 'L'} ${tok}=${t[tok]} on ${bgName}=${bg} 实测 ${r.toFixed(2)}:1`,
            ).toBeGreaterThanOrEqual(4.5)
          }
        }
      })
    }
  }
})

describe('亮态 accent/语义色不回流 Tailwind-600 亮阶（v4.349 回归锁）', () => {
  it('Jade / Rose / Amber 主色为 Tailwind-700 深阶（原 #0d9488 / #e11d48 / #d97706 均为 600 阶，实测 <4.5）', () => {
    expect(getThemeTokens('nightJade', false).colorPrimary).toBe('#0f766e') // hex-exempt 测试锁值（令牌契约）
    expect(getThemeTokens('nightRose', false).colorPrimary).toBe('#be123c') // hex-exempt 测试锁值
    expect(getThemeTokens('nightAmber', false).colorPrimary).toBe('#b45309') // hex-exempt 测试锁值
  })

  it('亮态 success/warning 全预设统一深阶（#047857 / #92400e），Moss success 保留自有 lime 阶', () => {
    for (const preset of PRESETS) {
      const t = getThemeTokens(preset, false)
      expect(t.colorWarning, `${preset} 亮态 warning`).toBe('#92400e') // hex-exempt 测试锁值
      expect(t.colorSuccess, `${preset} 亮态 success`).toBe(preset === 'nightMoss' ? '#4d7c0f' : '#047857') // hex-exempt 测试锁值
    }
  })

  it('亮态 accentRgb 与 glow 跟随主色（CSS rgb() 消费方不得与 colorPrimary 漂移）', () => {
    for (const preset of PRESETS) {
      const t = getThemeTokens(preset, false)
      const rgb = t.accentRgb.split(',').map((n) => parseInt(n, 10))
      const primary = t.colorPrimary.replace('#', '')
      expect(rgb, `${preset} 亮态 accentRgb`).toEqual([0, 2, 4].map((i) => parseInt(primary.slice(i, i + 2), 16)))
      expect(t.glow, `${preset} 亮态 glow`).toBe(t.colorPrimary)
    }
  })
})

describe('代码语法色板：diff（Lezer tok-*）与 hljs 共用一套色，亮态达 AA', () => {
  const root = resolve(__dirname, '..')

  it('changesdiff-tok.css 零字面色值，全部走 var(--hl-*)（防回流 One Dark 写死）', () => {
    const css = readFileSync(resolve(root, 'gaea/components/changesdiff-tok.css'), 'utf8')
    const body = css.replace(/\/\*[\s\S]*?\*\//g, '') // 去注释
    expect(body).not.toMatch(/#[0-9a-fA-F]{3,8}\b/)
    expect(body).not.toMatch(/rgba?\(/)
    const colorDecls = body.match(/color:\s*[^;]+;/g) ?? []
    expect(colorDecls.length).toBeGreaterThanOrEqual(7)
    for (const decl of colorDecls) expect(decl, decl).toContain('var(--hl-')
  })

  it('ChangesDiff 显式 import 语法色真源（否则懒加载链上 var(--hl-*) 未定义）', () => {
    const src = readFileSync(resolve(root, 'gaea/components/ChangesDiff.tsx'), 'utf8')
    expect(src).toContain('import "../hljs-theme.css"')
  })

  it('hljs 浅色组每个令牌在白底上 ≥ 4.5:1（暗色淡色阶在浅底不可用的根因锁）', () => {
    const css = readFileSync(resolve(root, 'gaea/hljs-theme.css'), 'utf8')
    // 注意用 lastIndexOf：`.hl-scope-dark` 在文件头注释里先出现过一次
    const lightBlock = css.slice(css.indexOf(':root[data-hl="light"]'), css.lastIndexOf('.hl-scope-dark'))
    const decls = [...lightBlock.matchAll(/--hl-light-([a-z]+):\s*(#[0-9a-fA-F]{6})/g)]
    expect(decls.length).toBe(8) // keyword/string/number/comment/func/type/builtin/meta
    for (const [, name, hex] of decls) {
      const r = contrast(hex, '#ffffff') // hex-exempt 测试用基准白底
      expect(r, `--hl-light-${name}=${hex} on #fff 实测 ${r.toFixed(2)}:1`).toBeGreaterThanOrEqual(4.5)
    }
  })
})
