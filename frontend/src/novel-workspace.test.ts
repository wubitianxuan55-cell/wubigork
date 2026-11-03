import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('novel-workspace 书房工坊契约', () => {
  const css = readFileSync(resolve(__dirname, 'novel-workspace.css'), 'utf8')
  const atelier = readFileSync(resolve(__dirname, 'novel-atelier.css'), 'utf8')

  it('阅读 tab 不把属性检查器 display:none（章节体检入口）', () => {
    const hideChapterInspector = /\.novel-hub\[data-novel-tab='chapter'\][^{]*\{[^}]*novel-inspector-zone[^}]*display:\s*none/s
    expect(css).not.toMatch(hideChapterInspector)
    expect(css).not.toContain(
      ".novel-hub[data-novel-tab='chapter'] > .novel-workspace > .novel-inspector-zone",
    )
  })

  it('书架画廊头 / 正在编辑横条 / 书脊仍在视觉语言里', () => {
    expect(css).toContain('.novel-shelf-head')
    expect(css).toContain('.novel-now-reading')
    expect(css).toContain('.novel-cover-spine')
    expect(css).toContain('.novel-cover-now')
  })

  it('编辑部视觉层保持令牌化，并覆盖六个页签的工作台', () => {
    expect(atelier).toContain('.novel-atelier-bar')
    expect(atelier).toContain('.novel-subnav-index')
    expect(atelier).toContain('.novel-hub[data-novel-tab="sin"]')
    expect(atelier).toContain('.novel-setting-shell')
    expect(atelier).toContain('.novel-reading-empty')
    expect(atelier).not.toMatch(/#[0-9a-f]{3,8}\b/i)
  })
})
