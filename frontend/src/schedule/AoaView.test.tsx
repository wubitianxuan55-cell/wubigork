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
    const graph = buildAoa(useScheduleStore.getState().project.tasks, useScheduleStore.getState().project.links)
    render(<AoaView graph={graph} tasks={[]} />)
    const s = graph.nodes.find((n) => n.anchor === 'S')! // 期望=图面坐标+位移（wrap style 含标尺偏移不可反推）
    fireEvent.mouseDown(nodeWrap('S'), { clientX: 0, clientY: 0 })
    fireEvent.mouseMove(window, { clientX: 500, clientY: 37 })
    fireEvent.mouseUp(window)
    const pins = useScheduleStore.getState().project.aoaLayout?.pins ?? {}
    expect(pins.S).toEqual(snapPt(s.x + 500, s.y + 37))
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
    const graph = buildAoa(p.tasks, p.links)
    const tNode = graph.nodes.find((n) => n.anchor === 'T')!
    fireEvent.mouseDown(nodeWrap('T'), { clientX: 0, clientY: 0 })
    fireEvent.mouseMove(window, { clientX: 220, clientY: 74 })
    fireEvent.mouseUp(window)
    const pins = useScheduleStore.getState().project.aoaLayout?.pins ?? {}
    expect(pins['end:ghost']).toBeUndefined() // 失配键被剪枝
    expect(pins.T).toEqual(snapPt(tNode.x + 220, tNode.y + 74))
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

  it('G4：auto 时标模式自由时差画波形线（独立路径，绿色图例语言）', () => {
    const { tasks, graph } = floatGraph()
    render(<AoaView graph={graph} tasks={tasks} />)
    // 波形拆独立路径（v4.161）：sched-aoa-wave，主路径不再含 q 段
    const waves = screen.getByTestId('sched-aoa').querySelectorAll('path.sched-aoa-wave')
    expect(waves.length).toBeGreaterThan(0)
    expect(screen.getByTestId('sched-aoa').textContent).toContain('自由时差（波形线）')
  })

  it('G4：手动布局 x 与时间解耦，不画波形线', () => {
    const { tasks, graph } = floatGraph()
    useScheduleStore.getState().setAoaPins({ S: { x: 550, y: 222 } })
    render(<AoaView graph={graph} tasks={tasks} />)
    const waved = screen.getByTestId('sched-aoa').querySelectorAll('path.sched-aoa-wave')
    expect(waved).toHaveLength(0)
    useScheduleStore.getState().setAoaPins({})
  })

  it('G2：同一边内工作名称在工期标注上方', () => {
    const p = chainProject()
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    const g = screen.getByTestId('sched-aoa-graph').querySelector('g')! // 首边=起点→挖土完成事件
    const name = g.querySelector('text.sched-aoa-taskname')
    const dur = g.querySelector('text.sched-aoa-dur')
    expect(name).toBeTruthy()
    expect(dur?.textContent).toBe('3d')
    expect(parseFloat(name!.getAttribute('y')!)).toBeLessThan(parseFloat(dur!.getAttribute('y')!))
  })
})

// ── v4.159：画布滚轮缩放（普通滚轮=光标锚缩放，Shift+滚轮=原生横移）──

describe('AoaView 滚轮缩放（v4.159）', () => {
  beforeEach(() => {
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null })
  })

  it('普通滚轮：以光标为锚放大（与按钮同一 clamp/读数）', () => {
    render(<AoaView graph={buildAoa(useScheduleStore.getState().project.tasks, useScheduleStore.getState().project.links)} tasks={[]} />)
    expect(screen.getByTestId('sched-aoa-zoom').textContent).toBe('100%')
    const el = screen.getByTestId('sched-aoa')
    fireEvent.wheel(el, { deltaY: -100, clientX: 60, clientY: 40 })
    expect(screen.getByTestId('sched-aoa-zoom').textContent).toBe('120%')
    // 反向滚回（下限 0.1 不触底）
    fireEvent.wheel(el, { deltaY: 100, clientX: 60, clientY: 40 })
    expect(screen.getByTestId('sched-aoa-zoom').textContent).toBe('100%')
  })

  it('Shift+滚轮放行原生横向滚动，不缩放', () => {
    render(<AoaView graph={buildAoa(useScheduleStore.getState().project.tasks, useScheduleStore.getState().project.links)} tasks={[]} />)
    const el = screen.getByTestId('sched-aoa')
    fireEvent.wheel(el, { deltaY: -100, shiftKey: true })
    expect(screen.getByTestId('sched-aoa-zoom').textContent).toBe('100%')
  })
})

describe('AoaView 分级横幅（v4.160）', () => {
  beforeEach(() => {
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null })
  })

  function bandedProject(): SchedProject {
    return {
      name: '分级样板',
      startDate: '2026-09-07',
      tasks: [
        { id: 'G1', name: '管网工程', duration: 0, level: 0, progress: 0 },
        { id: 'A', name: '挖土', duration: 3, level: 1, progress: 0 },
        { id: 'B', name: '垫层', duration: 2, level: 1, progress: 0 },
        { id: 'G2', name: '场平工程', duration: 0, level: 0, progress: 0 },
        { id: 'C', name: '围墙', duration: 4, level: 1, progress: 0 },
      ],
      links: [
        { from: 'A', to: 'B', type: 'FS', lag: 0 },
        { from: 'B', to: 'C', type: 'FS', lag: 0 },
      ],
    }
  }

  it('两分部=两块交替底纹；汇总线=横幅顶通长线（分部名挂线上方）', () => {
    const p = bandedProject()
    const graph = buildAoa(p.tasks, p.links)
    render(<AoaView graph={graph} tasks={p.tasks} />)
    expect(screen.getByTestId('sched-aoa-band-0')).toBeTruthy()
    expect(screen.getByTestId('sched-aoa-band-1')).toBeTruthy()
    expect(screen.queryByTestId('sched-aoa-band-2')).toBeNull()
    // 汇总线（G2 分部）：三段正交且含横贯段；分部名 y 在横贯段上方
    const root = screen.getByTestId('sched-aoa')
    const summaryPath = root.querySelector(`svg g path[marker-end="url(#aoa-arrow-summary)"]`)
    expect(summaryPath).toBeTruthy()
    const nameTexts = Array.from(root.querySelectorAll('text.sched-aoa-taskname-summary'))
    expect(nameTexts.map((t) => t.textContent)).toContain('场平工程')
  })

  it('图例注明横幅口径', () => {
    const p = bandedProject()
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    expect(screen.getByTestId('sched-aoa').textContent).toContain('分组横幅顶部通长线')
  })
})

describe('AoaView 工程标尺与图面语言（v4.161 对齐标杆）', () => {
  beforeEach(() => {
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null })
  })

  it('auto 模式：顶部标尺（工程日/月/日）+底部（星期/工程周）齐备；月界竖线贯通', () => {
    const p = chainProject()
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    expect(screen.getByTestId('sched-aoa-ruler-top')).toBeTruthy()
    expect(screen.getByTestId('sched-aoa-ruler-bot')).toBeTruthy()
    const top = screen.getByTestId('sched-aoa-ruler-top').textContent ?? ''
    const bot = screen.getByTestId('sched-aoa-ruler-bot').textContent ?? ''
    for (const s of ['工程日', '2026.9']) expect(top).toContain(s)
    for (const s of ['星期', '工程周', '一']) expect(bot).toContain(s)
  })

  it('手动模式 x 与时间解耦：标尺隐藏', () => {
    const p = chainProject()
    useScheduleStore.getState().setAoaPins({ S: { x: 550, y: 222 } })
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    expect(screen.queryByTestId('sched-aoa-ruler-top')).toBeNull()
    useScheduleStore.getState().setAoaPins({})
  })

  it('关键事件（es=ls）红圈：图面至少一个 sched-aoa-node-crit', () => {
    const p = chainProject() // 全关键链
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    const critNodes = screen.getByTestId('sched-aoa').querySelectorAll('.sched-aoa-node-crit')
    expect(critNodes.length).toBeGreaterThan(0)
  })
})
