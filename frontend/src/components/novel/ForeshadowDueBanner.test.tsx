// ForeshadowDueBanner.test.tsx — 伏笔到期预警条（阶段八 8.6）
// 覆盖：超期红/本章回收黄/临近 info/零预警零渲染/失败静默；chapterNum 透传。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      ForeshadowDue: vi.fn(),
    },
  }
})

import ForeshadowDueBanner from './ForeshadowDueBanner'
import { app } from '../../gaea/lib/bridge'

describe('ForeshadowDueBanner 伏笔到期预警（阶段八 8.6）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('超期红横幅：overdueBy 计数+明细文案（出口判据「超期伏笔有可见预警」）', async () => {
    vi.mocked(app.ForeshadowDue).mockResolvedValue({
      chapterNum: 5,
      urgent: [
        { id: 'over1', title: '井底徽记', status: 'planted', plantedIn: '001.md', targetIn: '002.md', urgency: 3, overdueBy: 3 },
      ],
      soon: [],
    })
    render(<ForeshadowDueBanner chapterNum={5} />)
    const banner = await screen.findByTestId('foreshadow-due-banner')
    expect(banner.textContent).toContain('已超期')
    expect(banner.textContent).toContain('井底徽记')
    expect(banner.textContent).toContain('超期 3 章')
    expect(app.ForeshadowDue).toHaveBeenCalledWith(5)
  })

  it('本章应回收黄横幅（urgency=2 无超期）', async () => {
    vi.mocked(app.ForeshadowDue).mockResolvedValue({
      chapterNum: 5,
      urgent: [{ id: 'due1', title: '断指旧誓', status: 'hinted', targetIn: '005.md', urgency: 2 }],
      soon: [],
    })
    render(<ForeshadowDueBanner chapterNum={5} />)
    const banner = await screen.findByTestId('foreshadow-due-banner')
    expect(banner.textContent).toContain('本章应回收')
    expect(banner.textContent).toContain('断指旧誓')
  })

  it('仅临近：info 档（soon 桶）', async () => {
    vi.mocked(app.ForeshadowDue).mockResolvedValue({
      chapterNum: 5,
      urgent: [],
      soon: [{ id: 'soon1', title: '远期线', status: 'planted', targetIn: '008.md', urgency: 1 }],
    })
    render(<ForeshadowDueBanner chapterNum={5} />)
    const banner = await screen.findByTestId('foreshadow-due-banner')
    expect(banner.textContent).toContain('远期线')
    expect(banner.textContent).not.toContain('超期')
  })

  it('零预警与失败：均零渲染（不占版面不打扰）', async () => {
    vi.mocked(app.ForeshadowDue).mockResolvedValue({ chapterNum: 5, urgent: [], soon: [] })
    const { container } = render(<ForeshadowDueBanner chapterNum={5} />)
    await screen.findByTestId('foreshadow-due-banner').catch(() => null)
    // 等一拍让 promise 落定
    await new Promise((r) => setTimeout(r, 0))
    expect(container.querySelector('[data-testid="foreshadow-due-banner"]')).toBeNull()

    vi.mocked(app.ForeshadowDue).mockRejectedValue(new Error('网络错误'))
    const { container: c2 } = render(<ForeshadowDueBanner chapterNum={5} />)
    await new Promise((r) => setTimeout(r, 0))
    expect(c2.querySelector('[data-testid="foreshadow-due-banner"]')).toBeNull()
  })

  it('切章重拉：chapterNum 变化触发新请求', async () => {
    vi.mocked(app.ForeshadowDue).mockResolvedValue({ chapterNum: 5, urgent: [], soon: [] })
    const { rerender } = render(<ForeshadowDueBanner chapterNum={5} />)
    await new Promise((r) => setTimeout(r, 0))
    rerender(<ForeshadowDueBanner chapterNum={6} />)
    await new Promise((r) => setTimeout(r, 0))
    expect(app.ForeshadowDue).toHaveBeenCalledTimes(2)
    expect(app.ForeshadowDue).toHaveBeenLastCalledWith(6)
    fireEvent.click(document.body) // 触碰以保活（无操作）
  })
})
