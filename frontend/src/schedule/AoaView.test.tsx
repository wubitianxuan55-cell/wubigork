/**
 * AoaView.test.tsx — 双代号手动布局交互用例（v4.123.0 AOA 刀1）
 *
 * 用例口径：fireEvent 鼠标序列（down→move→up）驱动 window 监听器
 * （GanttView 刀9 同构），断言 store 落库结果（pin 提交/剪枝、未移动
 * 无操作、Esc 取消）与双模式开关（自动↔手动、重置布局）。
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { buildAoa } from './aoa'
import { snapPt } from './aoaLayout'
import { AoaView } from './AoaView'
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

function nodeWrap(anchor: string): HTMLElement {
  const el = screen.getByTestId('sched-aoa').querySelector<HTMLElement>(`[data-anchor="${anchor}"]`)
  if (!el) throw new Error(`无锚点 ${anchor} 的节点`)
  return el
}

describe('AoaView 手动布局（AOA 刀1）', () => {
  beforeEach(() => {
    const p = chainProject()
    useScheduleStore.setState({
      project: p,
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      syncError: null,
    })
  })

  it('自动模式拖拽节点：落点写 pins、视图自动转手动', () => {
    render(<AoaView graph={buildAoa(useScheduleStore.getState().project.tasks, useScheduleStore.getState().project.links)} tasks={[]} />)
    const el = nodeWrap('S')
    const ox = parseFloat(el.style.left) + 16 // R=16：wrap left = pos.x - R
    const oy = parseFloat(el.style.top) + 16
    fireEvent.mouseDown(el, { clientX: 0, clientY: 0 })
    fireEvent.mouseMove(window, { clientX: 500, clientY: 37 })
    fireEvent.mouseUp(window)
    const pins = useScheduleStore.getState().project.aoaLayout?.pins ?? {}
    expect(pins.S).toEqual(snapPt(ox + 500, oy + 37))
    // 开关切到手动（自动模式拖拽=转手动）
    expect(screen.getByTestId('sched-aoa-mode').textContent).toContain('手动')
  })

  it('未移动（纯点击）不提交、不转手动', () => {
    render(<AoaView graph={buildAoa(useScheduleStore.getState().project.tasks, useScheduleStore.getState().project.links)} tasks={[]} />)
    fireEvent.mouseDown(nodeWrap('S'), { clientX: 10, clientY: 10 })
    fireEvent.mouseUp(window)
    expect(useScheduleStore.getState().project.aoaLayout?.pins ?? {}).toEqual({})
  })

  it('Esc 取消：拖拽中不落 pin', () => {
    render(<AoaView graph={buildAoa(useScheduleStore.getState().project.tasks, useScheduleStore.getState().project.links)} tasks={[]} />)
    fireEvent.mouseDown(nodeWrap('S'), { clientX: 0, clientY: 0 })
    fireEvent.mouseMove(window, { clientX: 300, clientY: 0 })
    expect(screen.getByTestId('sched-aoa-drag-tip')).toBeTruthy() // 拖拽中显示吸附坐标 tip
    fireEvent.keyDown(window, { key: 'Escape' })
    fireEvent.mouseUp(window)
    expect(useScheduleStore.getState().project.aoaLayout?.pins ?? {}).toEqual({})
    expect(screen.queryByTestId('sched-aoa-drag-tip')).toBeNull()
  })

  it('手动模式：命中 pin 的事件用手动位、未命中事件保留自动位（混合共存）', () => {
    const p = chainProject()
    p.aoaLayout = { pins: { S: { x: 550, y: 222 } } }
    useScheduleStore.setState({ project: p })
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={[]} />)
    // 默认 pins 非空 → 手动
    expect(screen.getByTestId('sched-aoa-mode').textContent).toContain('手动')
    const start = screen.getByTestId('sched-aoa').querySelector<HTMLElement>('[data-anchor="S"]')!
    expect(start.style.left).toBe(`${550 - 16}px`) // R=16
    const other = screen.getByTestId('sched-aoa').querySelector<HTMLElement>('[data-anchor="end:A"]')!
    expect(other.style.left).not.toBe(`${550 - 16}px`)
  })

  it('重置布局：清 pins 回自动', () => {
    const p = chainProject()
    p.aoaLayout = { pins: { S: { x: 550, y: 222 } } }
    useScheduleStore.setState({ project: p })
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={[]} />)
    fireEvent.click(screen.getByTestId('sched-aoa-reset'))
    expect(useScheduleStore.getState().project.aoaLayout?.pins ?? {}).toEqual({})
  })

  it('提交时剪枝失配键', () => {
    const p = chainProject()
    p.aoaLayout = { pins: { 'end:ghost': { x: 1, y: 1 }, S: { x: 55, y: 37 } } }
    useScheduleStore.setState({ project: p })
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={[]} />)
    const tEl = nodeWrap('T')
    const tox = parseFloat(tEl.style.left) + 16
    const toy = parseFloat(tEl.style.top) + 16
    fireEvent.mouseDown(tEl, { clientX: 0, clientY: 0 })
    fireEvent.mouseMove(window, { clientX: 220, clientY: 74 })
    fireEvent.mouseUp(window)
    const pins = useScheduleStore.getState().project.aoaLayout?.pins ?? {}
    expect(pins['end:ghost']).toBeUndefined() // 失配键被剪枝
    expect(pins.T).toEqual(snapPt(tox + 220, toy + 74))
  })
})

// ── v4.130 刀H：G2 标注归位 + G4 波形线（图例语言）────────────────

describe('AoaView 刀H（G2/G4）', () => {
  beforeEach(() => {
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null })
  })

  /** A(2)→B(5)/A→C(1)：C 完成事件 es=3 早于总工期 7，其汇出虚工作带 4 天自由时差 */
  function floatGraph() {
    const tasks = [
      { id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 },
      { id: 'B', name: '主体', duration: 5, level: 1, progress: 0 },
      { id: 'C', name: '零星', duration: 1, level: 1, progress: 0 },
    ]
    const links = [
      { from: 'A', to: 'B', type: 'FS' as const, lag: 0 },
      { from: 'A', to: 'C', type: 'FS' as const, lag: 0 },
    ]
    return { tasks, links, graph: buildAoa(tasks, links) }
  }

  it('G4：auto 时标模式自由时差画波形线（d 含波浪 q 段）', () => {
    const { tasks, graph } = floatGraph()
    render(<AoaView graph={graph} tasks={tasks} />)
    // 只看网络图连线（图例里的波形小图标 d 同样含 q，不可混入）
    const waved = Array.from(screen.getByTestId('sched-aoa').querySelectorAll('path.sched-aoa-link'))
      .filter((p) => (p.getAttribute('d') ?? '').includes(' q '))
    expect(waved.length).toBeGreaterThan(0)
    expect(screen.getByTestId('sched-aoa').textContent).toContain('自由时差（波形线）')
  })

  it('G4：手动布局 x 与时间解耦，不画波形线', () => {
    const { tasks, graph } = floatGraph()
    useScheduleStore.getState().setAoaPins({ S: { x: 550, y: 222 } })
    render(<AoaView graph={graph} tasks={tasks} />)
    const waved = Array.from(screen.getByTestId('sched-aoa').querySelectorAll('path.sched-aoa-link'))
      .filter((p) => (p.getAttribute('d') ?? '').includes(' q '))
    expect(waved).toHaveLength(0)
    useScheduleStore.getState().setAoaPins({})
  })

  it('G2：同一边内工作名称在工期标注上方', () => {
    const p = chainProject()
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    const g = screen.getByTestId('sched-aoa').querySelector('svg g')! // 首边=起点→挖土完成事件
    const name = g.querySelector('text.sched-aoa-taskname')
    const dur = g.querySelector('text.sched-net-label')
    expect(name).toBeTruthy()
    expect(dur?.textContent).toBe('3d')
    expect(parseFloat(name!.getAttribute('y')!)).toBeLessThan(parseFloat(dur!.getAttribute('y')!))
  })
})
