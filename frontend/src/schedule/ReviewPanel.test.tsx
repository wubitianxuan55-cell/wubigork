/**
 * schedule/ReviewPanel.test.tsx — 检视面板 UI 用例（斑马复刻 S2）
 *
 * 数据全走 useScheduleStore（与被塞进 Popover 的运行形态一致）；
 * 期望值与 reviews.ts 同口径：完成量按有效工期加权、拉直只升不降且冻结。
 */

import { fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { ReviewPanel } from './ReviewPanel'
import { computeCpm } from './cpm'
import { useScheduleStore } from './store'
import type { SchedProject } from './types'

/** A(4)→B(4) 串联，周一开工：A es0..3，B es4..7 */
function chain(): SchedProject {
  return {
    name: '链式样板',
    startDate: '2026-09-07',
    tasks: [
      { id: 'A', name: '挖土', duration: 4, level: 1, progress: 100 },
      { id: 'B', name: '浇筑', duration: 4, level: 1, progress: 50 },
    ],
    links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }],
  }
}

function setProject(p: SchedProject) {
  useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', syncError: null, past: [], future: [] })
}

const renderPanel = (p: SchedProject) => {
  setProject(p)
  return render(<ReviewPanel cpm={computeCpm(p.tasks, p.links)} />)
}

beforeEach(() => {
  localStorage.removeItem('gaea.schedule.v1')
  setProject(chain())
})

describe('ReviewPanel 检视面板', () => {
  it('无检视：给引导文案 + 按钮可用（日期缺省不早于开工日）', () => {
    renderPanel(chain())
    expect(within(screen.getByTestId('sched-review-panel')).getByText(/还没有检视/)).toBeTruthy()
    const add = screen.getByTestId('sched-review-add') as HTMLButtonElement
    expect(add.disabled).toBe(false)
    const date = screen.getByTestId('sched-review-date') as HTMLInputElement
    expect(date.value >= '2026-09-07').toBe(true)
  })

  it('点「打一期检视」：store 多一期，列出期行与当期计划/实际两列', () => {
    renderPanel(chain())
    fireEvent.change(screen.getByTestId('sched-review-date'), { target: { value: '2026-09-11' } })
    fireEvent.click(screen.getByTestId('sched-review-add'))
    const p = useScheduleStore.getState().project
    expect(p.reviews).toHaveLength(1)
    // 2026-09-11 = 第 4 工作日；A/B 进度 100/50 被快照
    expect(p.reviews![0].at).toBe(4)
    expect(p.reviews![0].actuals.A).toEqual({ percent: 100, finish: 4 })
    expect(p.reviews![0].actuals.B).toEqual({ percent: 50 })
    // 面板出现当期区（计划应完成 5/8 工日 = 63%，实际 6/8 = 75%）
    const prog = screen.getByTestId('sched-review-progress').textContent ?? ''
    expect(prog).toContain('63%')
    expect(prog).toContain('75%')
    expect(prog).toContain('+12%')
  })

  it('同期重复：按钮禁用并给出原因（不并存两期）', () => {
    const p = chain()
    p.reviews = [{ at: 4, date: '2026-09-11', actuals: {} }]
    renderPanel(p)
    fireEvent.change(screen.getByTestId('sched-review-date'), { target: { value: '2026-09-11' } })
    expect((screen.getByTestId('sched-review-add') as HTMLButtonElement).disabled).toBe(true)
    expect(within(screen.getByTestId('sched-review-panel')).getByText(/已有一期检视/)).toBeTruthy()
  })

  it('拉直：写回进度并冻结该期（按钮随后禁用）', () => {
    const p = chain()
    p.tasks = p.tasks.map((t) => (t.id === 'B' ? { ...t, progress: 10 } : t))
    p.reviews = [{ at: 4, date: '2026-09-11', actuals: { B: { percent: 60 } } }]
    renderPanel(p)
    fireEvent.click(screen.getByText('拉直'))
    const after = useScheduleStore.getState().project
    expect(after.tasks.find((t) => t.id === 'B')!.progress).toBe(60)
    expect(after.reviews![0].frozen).toBe(true)
    expect((screen.getByRole('button', { name: '拉直' }) as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByTestId('sched-review-msg').textContent).toContain('已拉直')
  })

  it('期际对比：只列有变化的行，整体完成量差可见', () => {
    const p = chain()
    p.reviews = [
      { at: 2, date: '2026-09-09', actuals: { A: { percent: 50 }, B: { percent: 0 } } },
      { at: 4, date: '2026-09-11', actuals: { A: { percent: 100 }, B: { percent: 50 } } },
    ]
    renderPanel(p)
    const diff = screen.getByTestId('sched-review-diff').textContent ?? ''
    expect(diff).toContain('挖土')
    expect(diff).toContain('浇筑')
    expect(within(screen.getByTestId('sched-review-panel')).getByText(/完成量 25% → 75%/)).toBeTruthy()
  })

  it('删除一期：store 中消失，面板回到引导态', () => {
    const p = chain()
    p.reviews = [{ at: 4, date: '2026-09-11', actuals: {} }]
    renderPanel(p)
    fireEvent.click(screen.getByText('删除'))
    fireEvent.click(screen.getByRole('button', { name: 'OK' }))
    expect(useScheduleStore.getState().project.reviews).toHaveLength(0)
  })
})


