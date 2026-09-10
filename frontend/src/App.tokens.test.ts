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
    const src = readFileSync(resolve(__dirname, 'gaea/components/CostLibraryView.tsx'), 'utf8')
    expect(src).not.toContain('bg-accent text-white')
    expect(src).toContain('bg-accent text-accent-fg')
  })
})
