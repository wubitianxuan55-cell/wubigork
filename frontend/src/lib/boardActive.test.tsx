// boardActive.test.ts — 板块级可见信号（keepAlive 隐藏态治理）
//
// 反向守卫：板块可见信号语义钉子（原以小说板块为探针；2026-11-15 小说退役后探针换文学板块，语义不变）。
// 探针板块切走后其子页不得吞 Ctrl+S / F11 等窗口级快捷键（keepAlive 隐藏态治理）。

import { afterEach, describe, expect, it } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { currentBoardActive, isBoardActive, notifyBoardActive, resetBoardActive, useBoardActive } from './boardActive'

function Probe({ page }: { page: 'wenxue' | 'gaea' }) {
  const active = useBoardActive(page)
  return <div data-testid={`probe-${page}`} data-active={String(active)} />
}

afterEach(() => { act(() => { resetBoardActive() }) })

describe('boardActive（板块级可见信号）', () => {
  it('未收到壳层通知时恒为可见（兼容「单独渲染页面」的既有写法）', () => {
    expect(currentBoardActive()).toBe('')
    expect(isBoardActive('wenxue')).toBe(true)
    expect(isBoardActive('gaea')).toBe(true)
  })

  it('壳层声明当前板块后：本板块可见、其它板块不可见', () => {
    render(<Probe page="wenxue" />)
    render(<Probe page="gaea" />)
    expect(screen.getByTestId('probe-wenxue').getAttribute('data-active')).toBe('true')
    expect(screen.getByTestId('probe-gaea').getAttribute('data-active')).toBe('true')

    act(() => { notifyBoardActive('gaea') })
    expect(screen.getByTestId('probe-wenxue').getAttribute('data-active')).toBe('false')
    expect(screen.getByTestId('probe-gaea').getAttribute('data-active')).toBe('true')
    expect(isBoardActive('wenxue')).toBe(false)
    expect(isBoardActive('gaea')).toBe(true)

    act(() => { notifyBoardActive('wenxue') })
    expect(screen.getByTestId('probe-wenxue').getAttribute('data-active')).toBe('true')
    expect(screen.getByTestId('probe-gaea').getAttribute('data-active')).toBe('false')
  })

  it('同值重复通知不触发额外渲染（幂等）', () => {
    let renders = 0
    function Counting() {
      renders += 1
      useBoardActive('wenxue')
      return null
    }
    render(<Counting />)
    const before = renders
    act(() => { notifyBoardActive('wenxue') })
    act(() => { notifyBoardActive('wenxue') })
    expect(renders).toBe(before)
  })

  it('resetBoardActive 回到「未通知」默认态（等价于可见）', () => {
    act(() => { notifyBoardActive('gaea') })
    expect(isBoardActive('wenxue')).toBe(false)
    act(() => { resetBoardActive() })
    expect(currentBoardActive()).toBe('')
    expect(isBoardActive('wenxue')).toBe(true)
  })

  // 生产者侧 source-guard：壳层不通知 = 本信号恒真 = 板块级门控整体失效
  // （F11/Ctrl+S 在后台板块里重新开始吞键）。壳层 MainLayout 依赖太多、无法轻量单测，
  // 故按仓内 source-guard 风格（perf-guards.test.ts）钉住这一行接线。
  it('壳层 MainLayout 在 page 变化时通知当前板块（否则消费者永远看到「可见」）', () => {
    const src = readFileSync(resolve(__dirname, '../layouts/MainLayout.tsx'), 'utf8')
    expect(src).toMatch(/notifyBoardActive\(page\)/)
    expect(src).toMatch(/\}, \[page\]\)/)
  })
})
