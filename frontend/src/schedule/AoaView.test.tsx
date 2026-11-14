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
import { deriveRowPitch } from './networkLayout'
import { AoaView } from './AoaView'
import { useScheduleStore } from './store'
import type { SchedNetworkStyle, SchedProject } from './types'

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

// 逻辑网络图（斑马 5 模式，一次点亮批3）：无时标标尺、无波形线，布图锁定
describe('AoaView 逻辑模式（renderMode=logic）', () => {
  it('不渲染工程标尺与波形线；拖拽锁定（标题提示）', () => {
    const p = chainProject()
    const g = buildAoa(p.tasks, p.links)
    render(<AoaView graph={g} tasks={p.tasks} renderMode="logic" />)
    expect(document.querySelector('[data-testid="sched-aoa-ruler-top"]')).toBeNull()
    expect(document.querySelector('[data-testid="sched-aoa-wave"]')).toBeNull()
    const wrap = document.querySelector('.sched-aoa-node')?.closest('div[title]') as HTMLElement | null
    expect(wrap?.getAttribute('title')).toContain('逻辑')
  })

  it('缺省（timescaled）保留标尺与波形线', () => {
    const p = chainProject()
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    expect(document.querySelector('[data-testid="sched-aoa-ruler-top"]')).not.toBeNull()
  })
})

// ── v4.366 W4「图面专业密度」：图上四开关 + 节点几何参数 ──────────────
//
// 零回归纪律：四个开关全缺省时（=斑马首装口径 节点时间开/工期开/六参数关/
// 起止时间关），渲染输出必须与「工程里根本没有 networkStyle」逐字一致，
// 这条比对是后续所有用例的地基。

describe('AoaView 图面四开关与几何参数（W4）', () => {
  /** 落一份带 networkStyle 的链式工程（无样式=完全不写该字段，模拟旧文件） */
  const styleProject = (style?: SchedNetworkStyle): SchedProject => {
    const p = chainProject()
    if (style) p.networkStyle = style
    useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', syncError: null })
    return p
  }

  it('四开关全缺省 ⇒ 渲染输出与「未落 networkStyle」逐字一致（零回归锚点）', () => {
    const snap = (style?: SchedNetworkStyle) => {
      const p = styleProject(style)
      const { unmount } = render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
      const html = screen.getByTestId('sched-aoa').innerHTML
      unmount()
      return html
    }
    const plain = snap(undefined)
    const explicit = snap({
      showDotTime: true,
      showDuration: true,
      showSixPara: false,
      showStartStopTime: false,
      dotStyle: 'circle',
      arrowStyle: 'solid',
      nodeRadius: 16,
      headHeight: 9,
      lineWidth: 1.5,
    })
    expect(explicit).toBe(plain)
  })

  it('箭头标记缺省口径不变（9×9 / refX 8 / refY 4.5），几何参数只改显式值', () => {
    const p = styleProject(undefined)
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    const marker = screen.getByTestId('sched-aoa').querySelector('marker#aoa-arrow')!
    expect(marker.getAttribute('markerWidth')).toBe('9')
    expect(marker.getAttribute('markerHeight')).toBe('9')
    expect(marker.getAttribute('refX')).toBe('8')
    expect(marker.getAttribute('refY')).toBe('4.5')
    expect(marker.querySelector('path')!.getAttribute('d')).toBe('M0,0 L9,4.5 L0,9 z')
  })

  it('showDotTime=false：去掉节点上下最早/最迟时间，节点与工期标签照旧', () => {
    const p = styleProject({ showDotTime: false })
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    const root = screen.getByTestId('sched-aoa')
    expect(root.querySelectorAll('.sched-aoa-time')).toHaveLength(0)
    expect(root.querySelector('.sched-aoa-node')).not.toBeNull()
    expect(root.querySelector('text.sched-aoa-dur')?.textContent).toBe('3d')
  })

  it('showDuration=false 且 showStartStopTime=true：工期标签整条不画，实工作箭线出第二行起止时间', () => {
    const p = styleProject({ showDuration: false, showStartStopTime: true })
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    const root = screen.getByTestId('sched-aoa')
    expect(root.querySelectorAll('text.sched-aoa-dur')).toHaveLength(0)
    const ss = root.querySelectorAll('text.sched-net-startstop')
    expect(ss.length).toBeGreaterThan(0)
    // 口径=横道计划开始/完成列的 M/D（按工程日历换算），中间用箭头连接
    expect(ss[0].textContent).toContain('→')
    expect(ss[0].textContent).toMatch(/\d+\/\d+ → \d+\/\d+/)
  })

  it('showSixPara=true：节点旁出六行（ES/EF/LS/LF/TF/FF 顺序，含 cpm 数值）', () => {
    const p = styleProject({ showSixPara: true })
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    const block = screen.getByTestId('sched-aoa').querySelector('[data-testid^="sched-net-sixpara-"]')
    expect(block).not.toBeNull()
    const lines = Array.from(block!.querySelectorAll('.sched-net-sixpara-line')).map((el) => el.textContent ?? '')
    expect(lines).toHaveLength(6)
    expect(lines.map((l) => l.split(' ')[0])).toEqual(['ES', 'EF', 'LS', 'LF', 'TF', 'FF'])
    expect(lines[0]).toBe('ES 0') // 链式首工作最早开始=0
  })

  it('dotStyle=rect 与 nodeRadius：节点改方角、尺寸随半径（circle 缺省不出方角类）', () => {
    const rectP = styleProject({ dotStyle: 'rect', nodeRadius: 20 })
    const { unmount } = render(<AoaView graph={buildAoa(rectP.tasks, rectP.links)} tasks={rectP.tasks} />)
    const node = screen.getByTestId('sched-aoa').querySelector<HTMLElement>('.sched-aoa-node')!
    expect(node.className).toContain('sched-net-node-rect')
    expect(node.style.width).toBe('40px')
    expect(node.style.height).toBe('40px')
    unmount()

    const circleP = styleProject(undefined)
    render(<AoaView graph={buildAoa(circleP.tasks, circleP.links)} tasks={circleP.tasks} />)
    expect(screen.getByTestId('sched-aoa').querySelector('.sched-net-node-rect')).toBeNull()
  })

  it('时标逻辑图（networkStyle.mode）：保留标尺与关键色，不画工期标签', () => {
    const p = styleProject({ mode: 'timescaledLogic' })
    render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    const root = screen.getByTestId('sched-aoa')
    expect(screen.getByTestId('sched-aoa-ruler-top')).toBeTruthy() // 时间轴保留
    expect(root.querySelectorAll('text.sched-aoa-dur')).toHaveLength(0) // 工期标签不画
    expect(root.querySelectorAll('.sched-aoa-node-crit').length).toBeGreaterThan(0) // 关键色保留
    expect(root.querySelector('.sched-aoa-taskname')).not.toBeNull() // 工作名称仍在
  })
})

// ── W4 收口：布图参数（density/layerHeight/stagger/namePos/hideUnimportantLink）──
//
// 这一组钉的是「面板能写、画面必须动」以及「缺省一动不动」两件事：
// 每个键都有一条消费用例 + 缺省零回归锚点（节点坐标数组逐字相等）。

describe('AoaView 布图参数消费（W4 收口）', () => {
  const styleProject = (style?: SchedNetworkStyle): SchedProject => {
    const p = chainProject()
    if (style) p.networkStyle = style
    useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', syncError: null })
    return p
  }
  /** 全部节点的 [anchor,left,top] 序列（零回归锚点的比对基准） */
  const coordSnap = (style?: SchedNetworkStyle): string[][] => {
    const p = styleProject(style)
    const { unmount } = render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    const els = Array.from(screen.getByTestId('sched-aoa').querySelectorAll<HTMLElement>('[data-anchor]'))
    const out = els.map((el) => [el.dataset.anchor ?? '', el.style.left, el.style.top])
    unmount()
    return out
  }
  /** 渲染一次并交出 root + 卸载器（screen 只认唯一的 sched-aoa，必须先卸再渲） */
  const renderAoa = (p: SchedProject, style?: SchedNetworkStyle) => {
    useScheduleStore.setState({
      project: style ? { ...p, networkStyle: style } : p,
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      syncError: null,
    })
    const r = render(<AoaView graph={buildAoa(p.tasks, p.links)} tasks={p.tasks} />)
    return { root: screen.getByTestId('sched-aoa'), unmount: r.unmount }
  }
  const wrapIn = (root: HTMLElement, anchor: string): HTMLElement => {
    const el = root.querySelector<HTMLElement>(`[data-anchor="${anchor}"]`)
    if (!el) throw new Error(`无锚点 ${anchor} 的节点`)
    return el
  }
  /** A(2)→B(5)/A→C(1)：C 边非关键（自由时差），是同列竖直箭线 */
  const floatProject = (): SchedProject => ({
    name: '时差样板',
    startDate: '2026-09-07',
    tasks: [
      { id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 },
      { id: 'B', name: '主体', duration: 5, level: 1, progress: 0 },
      { id: 'C', name: '零星', duration: 1, level: 1, progress: 0 },
    ],
    links: [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'A', to: 'C', type: 'FS', lag: 0 },
    ],
  })

  it('缺省零回归锚点（新）：不落 networkStyle vs 落全缺省布图参数 —— 节点坐标数组逐字相等', () => {
    const plain = coordSnap(undefined)
    const explicit = coordSnap({
      density: 'normal',
      stagger: false,
      namePos: 'auto',
      hideUnimportantLink: false,
      showDotTime: true,
      showDuration: true,
    })
    expect(explicit).toEqual(plain)
    expect(plain.length).toBeGreaterThan(0)
  })

  it('density=compact：列距/行距整体收紧（节点坐标变小），节点与事件序不变', () => {
    const normal = coordSnap(undefined)
    const compact = coordSnap({ density: 'compact' })
    const xs = (rows: string[][]) => rows.map((r) => parseFloat(r[1]))
    const ys = (rows: string[][]) => rows.map((r) => parseFloat(r[2]))
    expect(Math.max(...xs(compact))).toBeLessThan(Math.max(...xs(normal))) // 列距收紧=末列左移
    expect(Math.max(...ys(compact))).toBeLessThan(Math.max(...ys(normal))) // 行距收紧=末行上移
    expect(compact.map((r) => r[0])).toEqual(normal.map((r) => r[0])) // 拓扑/事件序逐字不变
  })

  it('density=loose：列距/行距整体放宽（与 compact 反向）', () => {
    const normal = coordSnap(undefined)
    const loose = coordSnap({ density: 'loose' })
    expect(Math.max(...loose.map((r) => parseFloat(r[1])))).toBeGreaterThan(Math.max(...normal.map((r) => parseFloat(r[1]))))
  })

  it('stagger=true：偶数行不动、奇数行再下移半个行距（手动 pin 在场则停用）', () => {
    const p = floatProject()
    const g = buildAoa(p.tasks, p.links)
    const pitch = deriveRowPitch(g.nodes)
    const rows = [...new Set(g.nodes.map((n) => n.y))].sort((a, b) => a - b)
    expect(rows.length).toBeGreaterThan(1) // 本图确有 ≥2 行可比
    const topsOf = (proj: SchedProject, style?: SchedNetworkStyle) => {
      const { root, unmount } = renderAoa(proj, style)
      const m = new Map(g.nodes.map((n) => [n.anchor, parseFloat(wrapIn(root, n.anchor).style.top)]))
      unmount()
      return m
    }
    const plain = topsOf(p, undefined)
    const staggered = topsOf(p, { stagger: true })
    for (const n of g.nodes) {
      const odd = rows.indexOf(n.y) % 2 === 1
      expect(staggered.get(n.anchor)! - plain.get(n.anchor)!).toBeCloseTo(odd ? pitch / 2 : 0)
    }
    // 手动布局（有 pin）时错层停用：坐标与不开启逐字相同
    const manualP: SchedProject = { ...p, aoaLayout: { pins: { S: { x: 550, y: 222 } } } }
    expect(topsOf(manualP, { stagger: true }).get('S')).toBe(topsOf(manualP, { stagger: false }).get('S'))
  })

  it('namePos=vertical：工作名称旋转 −90°（工期标注不受影响）；auto 保持原落位', () => {
    const p = floatProject()
    const auto = renderAoa(p)
    const names = Array.from(auto.root.querySelectorAll('text.sched-aoa-taskname'))
    expect(names.length).toBeGreaterThan(0)
    for (const n of names) expect(n.getAttribute('transform')).toBeNull() // auto=无旋转
    auto.unmount()

    const vert = renderAoa(p, { namePos: 'vertical' })
    const rotated = Array.from(vert.root.querySelectorAll('text.sched-aoa-taskname'))
    expect(rotated.length).toBeGreaterThan(0)
    for (const n of rotated) {
      expect(n.getAttribute('transform')).toMatch(/^rotate\(-90 /)
      expect(n.getAttribute('class')).toContain('sched-net-name-vertical')
    }
    // 工期标注照旧（名称排布只排布名称，不碰工期/时距）
    expect(vert.root.querySelector('text.sched-aoa-dur')?.getAttribute('transform')).toBeNull()
    vert.unmount()
  })

  /**
   * 里程碑 M（工期 0）：其起止事件同列 ⇒ M 的箭线是**竖直箭线**，
   * 正是「名称横向居中 vs 沿竖段挂字」的判别点（普通水平边两种模式落位相同）。
   */
  const milestoneProject = (): SchedProject => ({
    name: '里程碑样板',
    startDate: '2026-09-07',
    tasks: [
      { id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 },
      { id: 'M', name: '验收', duration: 0, level: 1, progress: 0, isMilestone: true },
      { id: 'B', name: '主体', duration: 3, level: 1, progress: 0 },
    ],
    links: [
      { from: 'A', to: 'M', type: 'FS', lag: 0 },
      { from: 'M', to: 'B', type: 'FS', lag: 0 },
    ],
  })

  it('namePos=horizontal：竖直箭线上的名称改水平居中（anchor 由 start 变 middle）', () => {
    const p = milestoneProject()
    const nameOf = (root: HTMLElement, text: string) =>
      Array.from(root.querySelectorAll('text.sched-aoa-taskname')).find((el) => el.textContent === text)!
    const auto = renderAoa(p)
    // auto：同列竖直边的旧口径=右侧 start 挂字
    expect(nameOf(auto.root, '验收').getAttribute('text-anchor')).toBe('start')
    auto.unmount()

    const horiz = renderAoa(p, { namePos: 'horizontal' })
    const el = nameOf(horiz.root, '验收')
    expect(el.getAttribute('text-anchor')).toBe('middle')
    expect(el.getAttribute('transform')).toBeNull()
    horiz.unmount()
  })

  it('hideUnimportantLink=true：非关键实工作箭线整条不画，关键/虚工作保留并给图例提示', () => {
    const p = floatProject()
    const graph = buildAoa(p.tasks, p.links)
    const count = (root: HTMLElement) => ({
      links: root.querySelectorAll('path.sched-aoa-link').length,
      crit: root.querySelectorAll('path.sched-aoa-link-critical').length,
      dummy: root.querySelectorAll('path.sched-aoa-dummy').length,
    })
    const auto = renderAoa(p)
    const before = count(auto.root)
    expect(screen.queryByTestId('sched-aoa-hide-hint')).toBeNull() // 缺省不出提示
    auto.unmount()

    const hiddenP = renderAoa(p, { hideUnimportantLink: true })
    const after = count(hiddenP.root)
    const hidden = graph.edges.filter((e) => e.kind === 'task' && !e.critical).length
    expect(hidden).toBeGreaterThan(0) // 本图确有非关键实工作
    expect(after.links).toBe(before.links - hidden) // 只少掉非关键实工作
    expect(after.crit).toBe(before.crit) // 关键线路全保
    expect(after.dummy).toBe(before.dummy) // 虚工作拓扑全保
    expect(screen.getByTestId('sched-aoa-hide-hint').textContent).toContain(`已隐藏 ${hidden} 条`)
    hiddenP.unmount()
  })
})
