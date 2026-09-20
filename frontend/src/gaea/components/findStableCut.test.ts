// findStableCut.test.ts — v4.368 流式分段渲染：切分函数契约
// （段落边界切分 / 未闭合 fence 回退 / fence body 含空行不拦腰切断 / 无空行全不稳定）
import { describe, expect, it } from 'vitest'
import { findStableCut } from '../lib/markdownCut'

describe('findStableCut 稳定切分', () => {
  it('无空行：全部不稳定', () => {
    const [stable, pending] = findStableCut('单段落没有空行边界')
    expect(stable).toBe('')
    expect(pending).toBe('单段落没有空行边界')
  })

  it('常规多段落：在最后一个空行处切', () => {
    const text = '第一段。\n\n第二段。\n\n第三段开头'
    const [stable, pending] = findStableCut(text)
    expect(stable).toBe('第一段。\n\n第二段。\n\n')
    expect(pending).toBe('第三段开头')
  })

  it('切点后打开的未闭合 fence 整体留在尾段', () => {
    const text = '说明。\n\n```js\nconst a = 1'
    const [stable, pending] = findStableCut(text)
    expect(stable).toBe('说明。\n\n')
    expect(pending).toBe('```js\nconst a = 1')
  })

  it('fence body 含空行（未闭合）：不拦腰切断，整体留尾（v4.368 盲区修复）', () => {
    const text = '前言。\n\n```python\nresult = compute(\n\n  arg\n\n)\n'
    const [stable, pending] = findStableCut(text)
    // stable 不得包含悬挂 fence
    const fences = (stable.match(/```/g) ?? []).length
    expect(fences % 2).toBe(0)
    expect(pending.startsWith('```')).toBe(true)
  })

  it('闭合 fence 之后有空行：fence 完整进入稳定段', () => {
    const text = '```js\ncode()\n```\n\n后续段落'
    const [stable, pending] = findStableCut(text)
    expect(stable).toBe('```js\ncode()\n```\n\n')
    expect(pending).toBe('后续段落')
  })

  it('空文本：全不稳定', () => {
    const [stable, pending] = findStableCut('')
    expect(stable).toBe('')
    expect(pending).toBe('')
  })
})
