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

  it('缺省全列展开：表格窗格宽=全列宽（与旧版一致）', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL}px`)
    expect(tablePane().textContent).toContain('最迟开始')
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
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL}px`)
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
