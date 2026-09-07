/**
 * GanttView.test.tsx — 横道拖拽交互（刀9）+ 工作台双栏（刀C）
 *
 * 用例口径：fireEvent 鼠标序列（down→move→up）驱动 window 监听器，
 * 断言 store 落库结果（拖移 auto→转 manual 锁定、缩放→改工期、
 * 未移动=无操作、Esc=取消）；刀C：分隔条拖动收纳表格/双击复位/列显隐
 * （行号名称固定，其余单列可藏）持久化 chatPrefs。
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { GanttView } from './GanttView'
import { computeCpm } from './cpm'
import { useScheduleStore } from './store'
import { CHAT_PREFS_KEY } from './chatPrefs'
import { GANTT_LEFT_W_FULL, GANTT_TABLE_W_MIN } from './ganttCols'
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

/** 取画布窗格第 idx 个数据行的 .sched-bar（行序=任务表序） */
function barAt(idx: number): HTMLElement {
  const rows = screen.getByTestId('sched-gantt-canvas').querySelectorAll('.sched-gantt-row:not(.sched-gantt-head)')
  const bar = rows[idx].querySelector<HTMLElement>('.sched-bar')
  if (!bar) throw new Error(`第 ${idx} 行无 .sched-bar`)
  return bar
}

function tablePane(): HTMLElement {
  return screen.getByTestId('sched-gantt-table')
}

describe('GanttView 拖拽（刀9）', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
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

describe('GanttView 工作台双栏（刀C）', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null })
  })

  it('缺省：进度列默认隐藏，窗格宽=可见列总宽（960-56=904），列菜单可开进度', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    // 15 列全宽 960，进度列缺省隐藏 → 可见总宽 904
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - 56}px`)
    expect(tablePane().textContent).toContain('最迟开始')
    expect(tablePane().textContent).not.toContain('进度')
  })

  it('分隔条拖动：左移收纳表格并持久化；拖到最窄剩行号+名称', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const split = screen.getByTestId('sched-gantt-split')
    fireEvent.mouseDown(split, { clientX: 900 })
    fireEvent.mouseMove(window, { clientX: 900 - 300 })
    fireEvent.mouseUp(window)
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - 300}px`)
    expect(JSON.parse(localStorage.getItem(CHAT_PREFS_KEY)!).ganttTableW).toBe(GANTT_LEFT_W_FULL - 300)
    // 再拖 9999：钳到最小宽（行号+名称），画布让位
    fireEvent.mouseDown(split, { clientX: 0 })
    fireEvent.mouseMove(window, { clientX: -9999 })
    fireEvent.mouseUp(window)
    expect(tablePane().style.width).toBe(`${GANTT_TABLE_W_MIN}px`)
  })

  it('双击分隔条复位全列宽；重挂载读取持久化宽度', () => {
    const p = chainProject()
    const { unmount } = render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const split = screen.getByTestId('sched-gantt-split')
    fireEvent.mouseDown(split, { clientX: 900 })
    fireEvent.mouseMove(window, { clientX: 900 - 300 })
    fireEvent.mouseUp(window)
    unmount()
    // 重挂载：读 localStorage 恢复收纳宽度（偏好持久化）
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - 300}px`)
    fireEvent.dblClick(screen.getByTestId('sched-gantt-split'))
    // 复位=展开到全列宽，钳位到可见列总宽（进度列仍缺省隐藏）
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - 56}px`)
  })

  it('列显隐：取消勾选「最迟开始」后表头消失并持久化；固定列不提供勾选', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.click(screen.getByTestId('sched-gantt-cols-btn'))
    const menu = screen.getByTestId('sched-gantt-colmenu')
    expect(menu.textContent).not.toContain('任务名称') // 行号/名称固定不进菜单
    const lsLabel = Array.from(menu.querySelectorAll('label')).find((l) => l.textContent?.includes('最迟开始'))!
    fireEvent.click(lsLabel.querySelector('input')!)
    expect(tablePane().textContent).not.toContain('最迟开始')
    expect(tablePane().textContent).toContain('最迟完成')
    expect(JSON.parse(localStorage.getItem(CHAT_PREFS_KEY)!).ganttHide).toContain('ls')
  })

  it('隐藏全部可藏列：表格窗格钳到行号+名称最小宽', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.click(screen.getByTestId('sched-gantt-cols-btn'))
    const menu = screen.getByTestId('sched-gantt-colmenu')
    for (const label of Array.from(menu.querySelectorAll("label"))) {
      const input = label.querySelector('input')
      if (input && (input as HTMLInputElement).checked) fireEvent.click(input)
    }
    expect(tablePane().style.width).toBe(`${GANTT_TABLE_W_MIN}px`)
    expect(tablePane().textContent).toContain('任务名称')
  })
})


/** antd 组件 data-testid 落点不一（Input 在 input 本体、InputNumber 在包装层），统一取内部 input */
function inputOf(testid: string): HTMLInputElement {
  const el = screen.getByTestId(testid)
  if (el instanceof HTMLInputElement) return el
  const inner = el.querySelector('input')
  if (inner) return inner
  throw new Error(`${testid} 内无 input`)
}

describe('GanttView 刀C 余项（筛选/右键菜单/进度）', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null, past: [], future: [] })
  })

  it('筛选「手动」：无手动任务时出空态提示；切回「全部」恢复', () => {
    render(<GanttView project={chainProject()} cpm={computeCpm(chainProject().tasks, chainProject().links)} />)
    const rowsBefore = screen.getByTestId('sched-gantt-table').querySelectorAll('.sched-gantt-row:not(.sched-gantt-head)').length
    expect(rowsBefore).toBe(3)
    fireEvent.click(screen.getByTestId('sched-gantt-filter').querySelectorAll('input, [role=radio], button')[2])
    // 手动模：链式样板全为自动 → 空态
    expect(screen.getByTestId('sched-gantt-filter-empty')).toBeTruthy()
    // 切回全部
    fireEvent.click(screen.getByTestId('sched-gantt-filter').querySelectorAll('input, [role=radio], button')[0])
    expect(screen.queryByTestId('sched-gantt-filter-empty')).toBeNull()
  })

  it('文本搜索命中行保留，其余行隐藏（表格与条形同源）', () => {
    render(<GanttView project={chainProject()} cpm={computeCpm(chainProject().tasks, chainProject().links)} />)
    fireEvent.change(inputOf('sched-gantt-search'), { target: { value: '垫层' } })
    const rows = screen.getByTestId('sched-gantt-table').querySelectorAll('.sched-gantt-row:not(.sched-gantt-head)')
    expect(rows).toHaveLength(1)
    // 行名在 Input 的 value 里（非 textContent）
    const nameInput = rows[0].querySelector('.sched-gantt-cell input') as HTMLInputElement
    expect(nameInput.value).toBe('垫层')
  })

  it('行右键菜单：设为里程碑/删除落库；进度格可直接编辑', () => {
    render(<GanttView project={chainProject()} cpm={computeCpm(chainProject().tasks, chainProject().links)} />)
    const rows = screen.getByTestId('sched-gantt-table').querySelectorAll('.sched-gantt-row:not(.sched-gantt-head)')
    fireEvent.contextMenu(rows[0])
    // Dropdown 菜单挂 body：点「设为里程碑」
    const mileItem = screen.getAllByText('设为里程碑').pop()!
    fireEvent.click(mileItem)
    expect(useScheduleStore.getState().project.tasks[0].isMilestone).toBe(true)
    // 进度格（默认隐藏）→ 开列后编辑
    localStorage.removeItem(CHAT_PREFS_KEY)
    fireEvent.click(screen.getByTestId('sched-gantt-cols-btn'))
    const progressLabel = Array.from(screen.getByTestId('sched-gantt-colmenu').querySelectorAll('label'))
      .find((l) => l.textContent?.includes('进度'))!
    fireEvent.click(progressLabel.querySelector('input')!)
    fireEvent.change(inputOf('sched-progress-A'), { target: { value: '60' } })
    expect(useScheduleStore.getState().project.tasks[0].progress).toBe(60)
  })
})

/** 分组样板：G(土建){A(3)→B(2)→C(4)} + 独立 D(1)。A/B/C 关键（tf=0），D tf=8 非关键 */
function groupProject(): SchedProject {
  return {
    name: '分组样板',
    startDate: '2026-09-07',
    tasks: [
      { id: 'G', name: '土建', duration: 0, level: 0, progress: 0 },
      { id: 'A', name: '挖土', duration: 3, level: 1, progress: 0 },
      { id: 'B', name: '垫层', duration: 2, level: 1, progress: 0 },
      { id: 'C', name: '浇筑', duration: 4, level: 1, progress: 0 },
      { id: 'D', name: '独立项', duration: 1, level: 1, progress: 0 },
    ],
    links: [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'B', to: 'C', type: 'FS', lag: 0 },
    ],
  }
}

/** 表格数据行的行名 Input 值（叶行） */
function rowNames(): string[] {
  const rows = screen.getByTestId('sched-gantt-table').querySelectorAll('.sched-gantt-row:not(.sched-gantt-head)')
  return Array.from(rows).map((r) => {
    const inp = r.querySelector('.sched-gantt-cell input') as HTMLInputElement | null
    return inp ? inp.value : (r.querySelector('.sched-group-name')?.textContent ?? '')
  })
}

describe('GanttView v4.136 小刀（排序/分组/折叠/路径/检查器）', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({ project: groupProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null, past: [], future: [] })
  })

  it('排序持久化读：ganttSort=工期降序 → 组内叶序 C(4)、A(3)、B(2)，组头不动', () => {
    localStorage.setItem(CHAT_PREFS_KEY, JSON.stringify({
      collapsed: false, width: 420, ganttTableW: GANTT_LEFT_W_FULL, ganttHide: ['progress'],
      ganttSort: { field: 'duration', dir: 'desc' }, ganttGroup: [],
    }))
    const p = groupProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(rowNames()).toEqual(['▲土建', '浇筑', '挖土', '垫层', '独立项'])
  })

  it('多级分组（受控渲染）：直接以 localStorage 预置 ganttGroup → 合成组头行 + 汇总跨度', () => {
    localStorage.setItem(CHAT_PREFS_KEY, JSON.stringify({
      collapsed: false, width: 420, ganttTableW: GANTT_LEFT_W_FULL, ganttHide: ['progress'],
      ganttSort: { field: 'none', dir: 'asc' }, ganttGroup: ['critical'],
    }))
    const p = groupProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(rowNames()).toEqual(['▲关键: 是', '挖土', '垫层', '浇筑', '▲关键: 否', '独立项'])
    // 折叠「关键: 是」组头 → 该组 3 叶消失，组头与「关键: 否」组保留
    fireEvent.click(screen.getAllByText('关键: 是', { exact: false })[0])
    expect(rowNames()).toEqual(['▶关键: 是', '▲关键: 否', '独立项'])
    fireEvent.click(screen.getAllByText('关键: 是', { exact: false })[0])
    expect(rowNames()).toEqual(['▲关键: 是', '挖土', '垫层', '浇筑', '▲关键: 否', '独立项'])
  })

  it('大纲折叠：点击 WBS 组头名折叠/展开直接子叶（v4.136 顺带销项）', () => {
    const p = groupProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(rowNames()).toHaveLength(5)
    fireEvent.click(screen.getByText('土建', { exact: false }))
    expect(rowNames()).toEqual(['▶土建'])
    fireEvent.click(screen.getByText('土建', { exact: false }))
    expect(rowNames()).toHaveLength(5)
  })

  it('路径分析：选中 B 开前驱链 → 链外 C 行淡化为 sched-row-dim，链上依赖线 sched-link-chain', () => {
    const p = chainProject()
    useScheduleStore.setState({ project: p, selectedId: 'B' })
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.click(screen.getByTestId('sched-gantt-path').querySelectorAll('input, [role=radio], button')[1])
    const rows = screen.getByTestId('sched-gantt-table').querySelectorAll('.sched-gantt-row:not(.sched-gantt-head)')
    expect(rows[0].className).not.toContain('sched-row-dim') // A 链上
    expect(rows[1].className).not.toContain('sched-row-dim') // B 锚点
    expect(rows[2].className).toContain('sched-row-dim') // C 链外
    const canvas = screen.getByTestId('sched-gantt-canvas')
    expect(canvas.querySelectorAll('path.sched-link-chain')).toHaveLength(1) // A→B
    expect(canvas.querySelectorAll('path.sched-link-dim')).toHaveLength(1) // B→C
    // 切回「路径」档 → 全部还原
    fireEvent.click(screen.getByTestId('sched-gantt-path').querySelectorAll('input, [role=radio], button')[0])
    expect(rows[2].className).not.toContain('sched-row-dim')
  })

  it('右键菜单「任务检查器」回调 onInspect（叶行）；路径关闭时无淡化类', () => {
    let inspectResult = ''
    const onInspect = (id: string) => { inspectResult = id }
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} onInspect={onInspect} />)
    const rows = screen.getByTestId('sched-gantt-table').querySelectorAll('.sched-gantt-row:not(.sched-gantt-head)')
    expect(rows[0].className).not.toContain('sched-row-dim')
    fireEvent.contextMenu(rows[0])
    fireEvent.click(screen.getAllByText('任务检查器').pop()!)
    expect(inspectResult).toBe('A')
  })
})
