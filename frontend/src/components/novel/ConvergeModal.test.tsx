import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import React from 'react'

const mocks = vi.hoisted(() => ({
  Preview: vi.fn(),
  Converge: vi.fn(),
}))
vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: new Proxy(
      { NovelChapterConvergePreview: mocks.Preview, NovelChapterConverge: mocks.Converge },
      {
        get(target, prop) {
          if (prop in target) return Reflect.get(target, prop)
          return (actual.app as unknown as Record<string, unknown>)[String(prop)]
        },
      },
    ),
  }
})

import ConvergeModal from './ConvergeModal'

describe('ConvergeModal 收敛修补（长篇刀4）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.Preview.mockResolvedValue({
      tasteScore: 72, targetTaste: 40, converged: false, planRounds: 2,
      s1s2: [{ code: 'telegraph_style', severity: 'S2', message: '疑似电报体：短句占比 68%' }],
      s3: [],
    })
    mocks.Converge.mockResolvedValue({ started: true })
  })

  it('打开即预检：分数/信号/清单渲染；未达标可开始', async () => {
    render(<ConvergeModal open chapterNum={3} onClose={vi.fn()} />)
    expect(await screen.findByTestId('converge-body')).toBeTruthy()
    expect(mocks.Preview).toHaveBeenCalledWith(3, 0)
    expect(screen.getByText(/AI 味 72 \/ 目标 ≤40/)).toBeTruthy()
    expect(screen.getByText(/S1\/S2 信号 1 项/)).toBeTruthy()
    expect(screen.getByText(/疑似电报体/)).toBeTruthy()
    expect(screen.getByTestId('converge-run')).toBeTruthy()
  })

  it('已达标：按钮「已达标」禁用 + 成功提示', async () => {
    mocks.Preview.mockResolvedValue({
      tasteScore: 30, targetTaste: 40, converged: true, planRounds: 2, s1s2: [], s3: [],
    })
    render(<ConvergeModal open chapterNum={3} onClose={vi.fn()} />)
    expect(await screen.findByText('已达标')).toBeTruthy()
    expect(screen.getByText(/无需修补/)).toBeTruthy()
    expect((screen.getByTestId('converge-run') as HTMLButtonElement).disabled).toBe(true)
  })

  it('启动即调闭环绑定；alreadyConverged 零轮直答提示', async () => {
    mocks.Converge.mockResolvedValue({ started: true, alreadyConverged: true, rounds: 0 })
    render(<ConvergeModal open chapterNum={3} onClose={vi.fn()} />)
    await screen.findByTestId('converge-body')
    fireEvent.click(screen.getByTestId('converge-run'))
    await waitFor(() => expect(mocks.Converge).toHaveBeenCalledWith(3, 0, 40))
    expect(await screen.findByText(/零轮直答/)).toBeTruthy()
  })

  it('预检失败如实报错', async () => {
    mocks.Preview.mockRejectedValue(new Error('读取章节失败——收敛修补需要已有正文'))
    render(<ConvergeModal open chapterNum={9} onClose={vi.fn()} />)
    expect(await screen.findByText(/收敛修补需要已有正文/)).toBeTruthy()
  })
})
