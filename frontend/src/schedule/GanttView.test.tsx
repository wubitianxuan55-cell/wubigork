/**
 * GanttView.test.tsx — 横道拖拽交互用例（v4.118.0 刀9）
 *
 * 用例口径：fireEvent 鼠标序列（down→move→up）驱动 window 监听器，
 * 断言 store 落库结果（拖移 auto→转 manual 锁定、缩放→改工期、
 * 未移动=无操作、Esc=取消）。
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { GanttView } from './GanttView'
import { computeCpm } from './cpm'
import { useScheduleStore } from './store'
import type { SchedProject } from './types'

/** A(3)→B(2)→C(4) 串联，周一开工：es A=0/B=3/C=5，总工期 9 */
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

/** 取第 idx 个数据行的 .sched-bar（行序=任务表序） */
function barAt(idx: number): HTMLElement {
  const rows = screen.getByTestId('sched-gantt').querySelectorAll('.sched-gantt-row:not(.sched-gantt-head)')
  const bar = rows[idx].querySelector<HTMLElement>('.sched-bar')
  if (!bar) throw new Error(`第 ${idx} 行无 .sched-bar`)
  return bar
}

describe('GanttView 拖拽（刀9）', () => {
  beforeEach(() => {
    useScheduleStore.setState({
      project: chainProject(),
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      syncError: null,
    })
  })

  it('拖移 auto 任务：转 manual 并锁定到落点工作日（60px=3 自然日）', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.mouseDown(barAt(0), { clientX: 0 }) // A：es=0
    fireEvent.mouseMove(window, { clientX: 60 })
    fireEvent.mouseUp(window)
    const t = useScheduleStore.getState().project.tasks.find((x) => x.id === 'A')!
    expect(t.mode).toBe('manual')
    expect(t.manualStart).toBe(3) // 偏移 0+3=周四=wd3（正中命中）
  })

  it('缩放右缘：改工期不动模式（40px，落周六吸附回周五 → 工期 3→4）', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const handle = barAt(0).querySelector<HTMLElement>('.sched-resize-handle')!
    fireEvent.mouseDown(handle, { clientX: 0 }) // A：右缘偏移 3
    fireEvent.mouseMove(window, { clientX: 40 }) // 右缘 3+2=5（周六）→ 吸附 wd4
    fireEvent.mouseUp(window)
    const t = useScheduleStore.getState().project.tasks.find((x) => x.id === 'A')!
    expect(t.mode).toBeUndefined() // 未转手动
    expect(t.duration).toBe(4) // 4 - es 0
  })

  it('未移动（纯点击条形）不产生任何修改', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const before = useScheduleStore.getState().project
    fireEvent.mouseDown(barAt(0), { clientX: 0 })
    fireEvent.mouseUp(window)
    expect(useScheduleStore.getState().project).toBe(before)
  })

  it('Esc 取消拖拽：不落库', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.mouseDown(barAt(0), { clientX: 0 })
    fireEvent.mouseMove(window, { clientX: 80 })
    fireEvent.keyDown(window, { key: 'Escape' })
    const t = useScheduleStore.getState().project.tasks.find((x) => x.id === 'A')!
    expect(t.mode).toBeUndefined()
  })

  it('循环依赖（cpm 不通过）禁拖', () => {
    const p = chainProject()
    p.links.push({ from: 'C', to: 'A', type: 'FS', lag: 0 })
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const before = useScheduleStore.getState().project
    fireEvent.mouseDown(barAt(0), { clientX: 0 })
    fireEvent.mouseMove(window, { clientX: 60 })
    fireEvent.mouseUp(window)
    expect(useScheduleStore.getState().project).toBe(before)
  })
})
