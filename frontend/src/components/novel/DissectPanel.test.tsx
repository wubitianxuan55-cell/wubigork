// DissectPanel.test.tsx — 对标拆解面板（oh-story story-long-analyze 蒸馏批 N6）：
// 打开拉清单/断点续跑跳过已拆章/串行循环进度/报告聚合渲染。
// 坑复训：antd InputNumber 取内层 input 才能 fireEvent.change；Tag 文本含插值
// 节点，断言用归一后的精确全文。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  chapter: vi.fn(),
  list: vi.fn(),
  report: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      NovelDissectChapter: mocks.chapter,
      NovelDissectList: mocks.list,
      NovelDissectReport: mocks.report,
    },
  }
})

import DissectPanel from './DissectPanel'

const entry = (chapter: number) => ({
  chapter,
  payoff: `兑现${chapter}`,
  hook_type: '危机',
  hook_detail: '债主堵门',
  emotion_arc: '压抑→爆发',
  info_push: [],
  craft_formulas: ['条件：谈判破裂→手法：旧物收尾→效果：情绪落地'],
  dissected_at: '2026-10-24T00:00:00Z',
})

const report2ch = {
  chapters_dissected: 2,
  timeline: [
    { chapter: 1, payoff: '兑现1', hook_type: '危机' },
    { chapter: 2, payoff: '兑现2', hook_type: '反转' },
  ],
  hook_dist: { 危机: 1, 反转: 1 },
  emotion_arc: [{ chapter: 1, emotion_arc: '压抑→爆发' }],
  formulas: ['条件：谈判破裂→手法：旧物收尾→效果：情绪落地'],
}

function renderPanel() {
  return render(<DissectPanel open onClose={vi.fn()} />)
}

/** antd InputNumber：data-testid 可能落在 input 本身或外层容器，两种都兼容。 */
const setRange = (testid: string, value: string) => {
  const el = screen.getByTestId(testid)
  const input = (el.tagName === 'INPUT' ? el : el.querySelector('input')) as HTMLInputElement
  fireEvent.change(input, { target: { value } })
}

describe('DissectPanel 对标拆解（oh-story 蒸馏批 N6）', () => {
  beforeEach(() => {
    mocks.chapter.mockReset()
    mocks.list.mockReset()
    mocks.report.mockReset()
    mocks.list.mockResolvedValue({ entries: [] })
  })

  it('打开拉清单：已拆章 Tag 显形；断点续跑跳过已拆章只调缺章', async () => {
    mocks.list.mockResolvedValueOnce({ entries: [entry(1)] })
    mocks.chapter.mockResolvedValueOnce(entry(2))
    renderPanel()

    expect(await screen.findByText('001')).toBeTruthy()
    expect(mocks.list).toHaveBeenCalledTimes(1)

    setRange('dissect-to', '2')
    fireEvent.click(screen.getByTestId('dissect-run'))

    await waitFor(() => expect(mocks.chapter).toHaveBeenCalledTimes(1))
    expect(mocks.chapter).toHaveBeenCalledWith(2, false) // 已拆的第 1 章被跳过；force=false
    expect(await screen.findByText(/本轮拆解完成 1\/1 章/)).toBeTruthy()
    await waitFor(() => expect(mocks.list).toHaveBeenCalledTimes(2)) // 收尾重拉清单
  })

  it('关闭断点续跑：范围内逐章 force 重拆', async () => {
    mocks.chapter.mockResolvedValue(entry(1))
    renderPanel()

    setRange('dissect-to', '2')
    fireEvent.click(screen.getByText('跳过已拆解（断点续跑）'))
    fireEvent.click(screen.getByTestId('dissect-run'))

    await waitFor(() => expect(mocks.chapter).toHaveBeenCalledTimes(2))
    expect(mocks.chapter).toHaveBeenNthCalledWith(1, 1, true)
    expect(mocks.chapter).toHaveBeenNthCalledWith(2, 2, true)
  })

  it('单章失败如实上屏并停止批次，不出假进度', async () => {
    mocks.chapter.mockRejectedValueOnce(new Error('第 2 章拆解失败: 模型未配置'))
    renderPanel()

    setRange('dissect-to', '3')
    fireEvent.click(screen.getByTestId('dissect-run'))

    expect(await screen.findByTestId('dissect-error')).toBeTruthy()
    expect(screen.getByTestId('dissect-error').textContent).toContain('模型未配置')
    // 失败即停：第 3 章不再发起
    await new Promise((r) => setTimeout(r, 20))
    expect(mocks.chapter).toHaveBeenCalledTimes(1)
  })

  it('拆解报告：章数/钩子分布/剧情线/公式渲染', async () => {
    mocks.report.mockResolvedValueOnce(report2ch)
    renderPanel()

    fireEvent.click(screen.getByTestId('dissect-report'))
    const view = await screen.findByTestId('dissect-report-view')
    expect(view.textContent).toContain('拆解报告（2 章')
    expect(view.textContent).toContain('危机×1')
    expect(view.textContent).toContain('反转×1')
    expect(view.textContent).toContain('剧情线（付息链）')
    expect(view.textContent).toContain('第001章 [危机] 兑现1')
    expect(view.textContent).toContain('1. 条件：谈判破裂→手法：旧物收尾→效果：情绪落地')
  })
})
