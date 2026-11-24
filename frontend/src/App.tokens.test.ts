import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('主色文字令牌（浅色主色上硬编码白字不可见）', () => {
  it('App 向 :root 注入 --color-on-primary / --color-surface', () => {
    const src = readFileSync(resolve(__dirname, 'App.tsx'), 'utf8')
    expect(src).toContain("set('--color-on-primary'")
    expect(src).toContain("set('--color-surface'")
  })

  it('办公 gaea 主色按钮用 accent-fg 而非 text-white', () => {
    // 造价库（2026-11-25 推倒重建）与方案库成对：主色按钮文字色都取 --accent-fg，
    // 样式表里不许出现硬编码白字（浅色主色上白字不可见）。
    for (const [file, selector] of [
      ['pages/cost-database.css', '.cl-btn-primary'],
      ['pages/proposal-library.css', '.pl-btn-primary'],
    ] as const) {
      const css = readFileSync(resolve(__dirname, file), 'utf8')
      const start = css.indexOf(`${selector} {`)
      expect(start, `${file} 缺 ${selector}`).toBeGreaterThanOrEqual(0)
      const rule = css.slice(start, css.indexOf('}', start))
      expect(rule).toContain('background: var(--accent)')
      expect(rule).toContain('color: var(--accent-fg)')
      expect(css).not.toMatch(/(?<![-\w])color:\s*(white|#fff\b)/i)
    }
  })
})
