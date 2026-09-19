// HistoryRail 渲染窗口（v4.351 有界化）：历史可达数百条，一次性渲染数百
// img/video DOM 卡顿——首屏只渲染 PAGE_SIZE 条，「加载更多」增量展开；
// 窗口外条目不渲染（数据仍完整，见 useImageGenHistory 回填窗口）。
import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { HistoryRail } from './HistoryRail'
import type { GenResult } from './types'

const item = (i: number): GenResult => ({
  image: `data:image/png;base64,IMG${i}`,
  seed: i, time: i, prompt: `p${i}`, model: 'xai', size: '1024x1024',
})

const items = (n: number) => Array.from({ length: n }, (_, i) => item(i))

function renderRail(history: GenResult[]) {
  return render(
    <HistoryRail
      history={history}
      selectedIndex={-1}
      onSelect={vi.fn()}
      onClear={vi.fn()}
    />,
  )
}

describe('HistoryRail 渲染窗口（v4.351）', () => {
  it('超过窗口只渲染前 60 条 + 「加载更多」按钮', () => {
    const { container } = renderRail(items(80))
    expect(container.querySelectorAll('.img-card')).toHaveLength(60)
    const more = screen.getByText(/加载更多/)
    expect(more).toBeTruthy()
  })

  it('点击加载更多增量展开；全部展开后按钮消失', () => {
    const { container } = renderRail(items(80))
    fireEvent.click(screen.getByText(/加载更多/))
    expect(container.querySelectorAll('.img-card')).toHaveLength(80)
    expect(screen.queryByText(/加载更多/)).toBeNull()
  })

  it('不足窗口全量渲染且无按钮（既有行为不回归）', () => {
    const { container } = renderRail(items(5))
    expect(container.querySelectorAll('.img-card')).toHaveLength(5)
    expect(screen.queryByText(/加载更多/)).toBeNull()
  })
})
