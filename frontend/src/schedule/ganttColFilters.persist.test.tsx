/**
 * ganttColFilters.persist.test.tsx — W5 列筛持久化接线（chatPrefs.ganttColFilters）
 *
 * 独立成文件的原因：列筛落盘/读回依赖「挂载前 localStorage 就位」，
 * 与 GanttView 主测试文件里几十个共享存储的用例同场跑时顺序敏感；
 * 这里只放两条端到端用例（写盘 / 读回），环境干净、断言稳定。
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { GanttView } from './GanttView'
import { CHAT_PREFS_KEY, saveChatPrefs } from './chatPrefs'
import { computeCpm } from './cpm'
import { useScheduleStore } from './store'
import type { SchedProject } from './types'

function chainProject(): SchedProject {
  return {
    name: '链式样板',
    startDate: '2026-09-07',
    tasks: [
      { id: 'A', name: '挖土', duration: 3, level: 1, progress: 0 },
      { id: 'B', name: '垫层', duration: 2, level: 1, progress: 0 },
      { id: 'C', name: '浇筑', duration: 4, level: 1, progress: 0 },
    ],
    links: [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'B', to: 'C', type: 'FS', lag: 0 },
    ],
  }
}

const seed = (p: SchedProject) => {
  useScheduleStore.setState({
    project: p, selectedId: null, hydrated: true, sync: 'saved',
    savedAt: null, syncError: null, past: [], future: [],
  })
}

const canvasRows = (root: HTMLElement) =>
  root.querySelectorAll('.sched-gantt-canvaspane .sched-gantt-row:not(.sched-gantt-head)').length

describe('W5 列筛持久化：chatPrefs.ganttColFilters 往返', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
  })

  it('改列筛 → 归一后写入偏好（键值形状固定）', async () => {
    const p = chainProject()
    seed(p)
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.click(screen.getByTestId('sched-colfilter-name'))
    const input = screen.getByTestId('sched-colfilter-editor-name').querySelector('input') as HTMLInputElement
    fireEvent.change(input, { target: { value: '垫层' } })
    await waitFor(() => {
      expect(JSON.parse(localStorage.getItem(CHAT_PREFS_KEY)!).ganttColFilters)
        .toEqual({ name: { kind: 'text', text: '垫层' } })
    })
  })

  it('偏好已有列筛 → 挂载即读回并作用于可见行，且首帧不覆盖偏好', () => {
    saveChatPrefs({ ganttColFilters: { name: { kind: 'text', text: '垫层' } } })
    const p = chainProject()
    seed(p)
    const { container } = render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    // 只剩「垫层」一行：筛选在首帧就生效，不需要用户再点一次
    expect(canvasRows(container)).toBe(1)
    // 首帧不写盘：读回的条件原样保留
    expect(JSON.parse(localStorage.getItem(CHAT_PREFS_KEY)!).ganttColFilters.name.text).toBe('垫层')
  })
})
