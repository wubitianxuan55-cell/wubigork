/**
 * GanttView.test.tsx — 横道拖拽交互（刀9）+ 工作台双栏（刀C）+ 自定义字段列（v4.138 #14）
 *
 * 用例口径：fireEvent 鼠标序列（down→move→up）驱动 window 监听器，
 * 断言 store 落库结果（拖移 auto→转 manual 锁定、缩放→改工期、
 * 未移动=无操作、Esc=取消）；刀C：分隔条拖动收纳表格/双击复位/列显隐
 * （行号名称固定，其余单列可藏）持久化 chatPrefs；v4.138 #14：自定义列
 * 开启/改名/叶行编辑（清空=删槽键）/分组行留空。
 */
import { act, fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { GanttView } from './GanttView'
import { computeCpm } from './cpm'
import { useScheduleStore } from './store'
import { CHAT_PREFS_KEY } from './chatPrefs'
import { GANTT_LEFT_W_FULL, GANTT_TABLE_W_MIN } from './ganttCols'
import type { CpmResult, SchedProject } from './types'

/** 缺省隐藏列宽合计：进度 56 + 文本×3（90）+ 数值×2（64）= 454（自定义列 v4.138 #14 缺省收起） */
const DEF_HIDE_W = 56 + 90 * 3 + 64 * 2

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

/** 打开列菜单并按名称勾选开启某列（popover 保持打开，返回该菜单行 label 供续操作） */
function enableCol(name: string): HTMLElement {
  fireEvent.click(screen.getByTestId('sched-gantt-cols-btn'))
  const label = Array.from(screen.getByTestId('sched-gantt-colmenu').querySelectorAll('label'))
    .find((l) => l.textContent?.includes(name))
  if (!label) throw new Error(`列菜单无「${name}」`)
  fireEvent.click(label.querySelector('input')!)
  return label
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

  it('缺省：进度列+自定义列默认隐藏，窗格宽=可见列总宽（全宽 1358-454=904），列菜单可开', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    // 20 列全宽 1358，进度列+5 自定义列缺省隐藏 → 可见总宽 904
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - DEF_HIDE_W}px`)
    expect(tablePane().textContent).toContain('最迟开始')
    expect(tablePane().textContent).not.toContain('进度')
    expect(tablePane().textContent).not.toContain('文本1')
  })

  it('分隔条拖动：左移收纳表格并持久化；拖到最窄剩行号+名称', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const split = screen.getByTestId('sched-gantt-split')
    fireEvent.mouseDown(split, { clientX: 900 })
    fireEvent.mouseMove(window, { clientX: 900 - 500 })
    fireEvent.mouseUp(window)
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - 500}px`)
    expect(JSON.parse(localStorage.getItem(CHAT_PREFS_KEY)!).ganttTableW).toBe(GANTT_LEFT_W_FULL - 500)
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
    fireEvent.mouseMove(window, { clientX: 900 - 500 })
    fireEvent.mouseUp(window)
    unmount()
    // 重挂载：读 localStorage 恢复收纳宽度（偏好持久化）
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - 500}px`)
    fireEvent.dblClick(screen.getByTestId('sched-gantt-split'))
    // 复位=钳到可见列总宽（进度列+自定义列仍缺省隐藏）
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - DEF_HIDE_W}px`)
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

/**
 * 自定义字段列（v4.138 #14）：列菜单开启/改名（customLabels 驱动列头与菜单同步）、
 * 叶行 text=Input / num=InputNumber 编辑落 store.custom（清空=删槽键）、分组行留空。
 * 交互约定：菜单行「改名」铅笔→行内受控 Input（初值=当前名），回车/失焦提交
 * setCustomLabel、Esc 取消。
 * StoreGanttView 包装：project 直读 store——对齐真实父组件「订阅 store 再传 props」
 * 的重渲染链路；否则落库后列头/菜单/受控格不重渲染（测试静态 props 的假阴性）。
 */
function StoreGanttView(props: { cpm: CpmResult }) {
  const project = useScheduleStore((s) => s.project)
  return <GanttView project={project} cpm={props.cpm} />
}

describe('GanttView 自定义字段列（v4.138 #14）', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null, past: [], future: [] })
  })

  it('自定义列缺省收起；列菜单开启 text1 → 表头显示缺省名「文本1」，排成本列之后', () => {
    const p = chainProject()
    useScheduleStore.setState({ project: p })
    render(<StoreGanttView cpm={computeCpm(p.tasks, p.links)} />)
    expect(tablePane().textContent).not.toContain('文本1')
    enableCol('文本1')
    expect(tablePane().textContent).toContain('文本1')
    // 列序：末位=文本1、次末位=成本（自定义列排成本列之后）
    const heads = Array.from(tablePane().querySelectorAll('.sched-gantt-head .sched-th')).map((el) => el.textContent)
    expect(heads[heads.length - 1]).toBe('文本1')
    expect(heads[heads.length - 2]).toBe('成本')
  })

  it('叶行 text 槽：Input 输入「垫层底部」落 store.custom；清空=删除槽键', () => {
    const p = chainProject()
    useScheduleStore.setState({ project: p })
    render(<StoreGanttView cpm={computeCpm(p.tasks, p.links)} />)
    enableCol('文本1')
    fireEvent.change(inputOf('sched-custom-text1-B'), { target: { value: '垫层底部' } })
    expect(useScheduleStore.getState().project.tasks[1].custom).toEqual({ text1: '垫层底部' })
    // 无值任务不受牵连（不留空值槽）
    expect(useScheduleStore.getState().project.tasks[0].custom).toBeUndefined()
    fireEvent.change(inputOf('sched-custom-text1-B'), { target: { value: '' } })
    expect(useScheduleStore.getState().project.tasks[1].custom).toEqual({})
  })

  it('叶行 num 槽：InputNumber 输入 3.5 落数字（可小数）；清空=删除槽键', () => {
    const p = chainProject()
    useScheduleStore.setState({ project: p })
    render(<StoreGanttView cpm={computeCpm(p.tasks, p.links)} />)
    enableCol('数值1')
    fireEvent.change(inputOf('sched-custom-num1-A'), { target: { value: '3.5' } })
    expect(useScheduleStore.getState().project.tasks[0].custom).toEqual({ num1: 3.5 })
    fireEvent.change(inputOf('sched-custom-num1-A'), { target: { value: '' } })
    expect(useScheduleStore.getState().project.tasks[0].custom).toEqual({})
  })

  it('列改名：菜单行「改名」铅笔→行内 Input 回车提交 → 表头/菜单同步（customLabels 驱动）', () => {
    const p = chainProject()
    useScheduleStore.setState({ project: p })
    render(<StoreGanttView cpm={computeCpm(p.tasks, p.links)} />)
    const t1 = enableCol('文本1')
    expect(tablePane().textContent).toContain('文本1')
    fireEvent.click(t1.querySelector('button')!) // 改名铅笔（仅自定义列有）
    const input = inputOf('sched-colmenu-rename-text1')
    input.focus()
    expect(input.value).toBe('文本1') // 初值=当前名
    fireEvent.change(input, { target: { value: '施工部位' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(useScheduleStore.getState().project.customLabels).toEqual({ text1: '施工部位' })
    expect(screen.getByTestId('sched-gantt-colmenu').textContent).toContain('施工部位')
    expect(tablePane().textContent).toContain('施工部位')
    expect(tablePane().textContent).not.toContain('文本1')
  })

  it('列改名：失焦提交；Esc 取消不落库', () => {
    const p = chainProject()
    useScheduleStore.setState({ project: p })
    render(<StoreGanttView cpm={computeCpm(p.tasks, p.links)} />)
    // 失焦提交（程序化 blur 非 React 离散事件，需 act 包裹触发同步冲刷）
    const t1 = enableCol('文本1')
    fireEvent.click(t1.querySelector('button')!)
    const input = inputOf('sched-colmenu-rename-text1')
    fireEvent.change(input, { target: { value: '施工部位' } })
    act(() => { input.focus() })
    act(() => { input.blur() })
    expect(useScheduleStore.getState().project.customLabels).toEqual({ text1: '施工部位' })
    // Esc 取消：列名不变、不落库
    const renamed = Array.from(screen.getByTestId('sched-gantt-colmenu').querySelectorAll('label'))
      .find((l) => l.textContent?.includes('施工部位'))!
    fireEvent.click(renamed.querySelector('button')!)
    const input2 = inputOf('sched-colmenu-rename-text1')
    input2.focus()
    fireEvent.change(input2, { target: { value: '乱改' } })
    fireEvent.keyDown(input2, { key: 'Escape' })
    expect(useScheduleStore.getState().project.customLabels).toEqual({ text1: '施工部位' })
    expect(screen.getByTestId('sched-gantt-colmenu').textContent).toContain('施工部位')
  })

  it('分组行自定义列留空（汇总只对叶任务有意义），叶行可编辑', () => {
    const p = groupProject()
    useScheduleStore.setState({ project: p })
    render(<StoreGanttView cpm={computeCpm(p.tasks, p.links)} />)
    enableCol('文本1')
    const rows = tablePane().querySelectorAll('.sched-gantt-row:not(.sched-gantt-head)')
    expect(rows[0].textContent).toContain('土建') // G 组头行
    expect(rows[0].querySelectorAll('input')).toHaveLength(0)
    expect(inputOf('sched-custom-text1-A')).toBeTruthy()
  })
})

describe('GanttView 双层时标（v4.140：自左向右水平）', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null })
  })

  it('月份跨列段=上行、逐日列=下行，两行日宽合计一致（时间轴水平非纵向堆叠）', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const head = document.querySelector('.sched-gantt-datehead')!
    const months = head.querySelectorAll<HTMLElement>('.sched-ts-month')
    const days = head.querySelectorAll<HTMLElement>('.sched-day-col')
    expect(months.length).toBeGreaterThanOrEqual(1)
    expect(days.length).toBeGreaterThanOrEqual(21) // days 下限 21
    // 每个月段宽=该段日数×dayW（跨列段，非逐日竖排）；段宽合计=日列宽合计=chartW
    const monthW = Array.from(months).reduce((s, el) => s + el.offsetWidth, 0)
    const dayW = Array.from(days).reduce((s, el) => s + el.offsetWidth, 0)
    expect(monthW).toBe(dayW)
    // jsdom 无布局：以 style.width 断言跨列口径（月段宽=日列宽和）
    const monthWStyle = Array.from(months).reduce((s, el) => s + Number.parseFloat(el.style.width), 0)
    const dayWStyle = Array.from(days).reduce((s, el) => s + Number.parseFloat(el.style.width), 0)
    expect(monthWStyle).toBe(dayWStyle)
    // 下行日号自左向右递增（1 号起），跨月段含「年月」标注
    const nums = Array.from(head.querySelectorAll('.sched-day-cell')).map((el) => Number.parseInt(el.textContent || '0', 10))
    expect(nums[0]).toBe(7) // 样板开工 2026-09-07
  })
})

describe('GanttView 缩放与周刻度（v4.141：窗口内全览）', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null })
  })

  it('连续缩小跨过 12px 阈值：底层刻度由日切周（月层保留），周段=自然周周一始', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(document.querySelectorAll('.sched-day-col').length).toBeGreaterThan(0)
    fireEvent.click(screen.getByTestId('sched-gantt-zoomout'))
    fireEvent.click(screen.getByTestId('sched-gantt-zoomout')) // 20/1.35²≈11.0 <12 → 周层
    const weeks = document.querySelectorAll('.sched-ts-week')
    expect(weeks.length).toBeGreaterThanOrEqual(3) // 2026-09-07(周一) 起 21 天=3 个自然周
    expect(document.querySelectorAll('.sched-ts-month').length).toBeGreaterThanOrEqual(1)
    expect(weeks[0].textContent).toBe('9/7') // 首周段标注起始日
    expect(document.querySelectorAll('.sched-day-col').length).toBe(0)
  })

  it('连续放大不超过上限；再点全览=整计划适配画布宽并回到日刻度', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    for (let i = 0; i < 8; i++) fireEvent.click(screen.getByTestId('sched-gantt-zoomin'))
    // 连续放大后仍能全览收回
    const pane = screen.getByTestId('sched-gantt-canvas')
    Object.defineProperty(pane, 'clientWidth', { value: 800, configurable: true })
    fireEvent.click(screen.getByTestId('sched-gantt-fit'))
    const w = Array.from(document.querySelectorAll('.sched-ts-month')).reduce(
      (s, el) => s + Number.parseFloat((el as HTMLElement).style.width), 0,
    )
    expect(w).toBeGreaterThan(600) // 21 天铺满 ~800px 窗
    expect(w).toBeLessThanOrEqual(800)
    expect(document.querySelectorAll('.sched-day-col').length).toBeGreaterThan(0) // 日宽回到 ≥12 → 日层
  })
})

describe('GanttView 列头不压缩（v4.142：收纳=裁剪隐藏，绝不挤压错位）', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null })
  })

  it('列头行显式宽度=全部可见列合计（与数据行同宽，收纳时被容器裁剪而非压缩）', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const head = document.querySelector('.sched-gantt-tablepane .sched-gantt-head') as HTMLElement
    const bodyWrap = document.querySelector('.sched-gantt-tbody > div') as HTMLElement
    expect(head.style.width).toBe(bodyWrap.style.width) // 同为 leftW
    const sum = Array.from(head.querySelectorAll('.sched-gantt-cell')).reduce(
      (s, el) => s + Number.parseFloat((el as HTMLElement).style.width), 0,
    )
    expect(Number.parseFloat(head.style.width)).toBe(sum)
  })
})

describe('GanttView 双工期（v4.152 刀3）', () => {
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

  /** 分组 G + wd A + cd H(28) + 里程碑 M：es A=0，H es=2 ef=22（28cd 等效 20） */
  function cdProject(): SchedProject {
    return {
      name: '养护样板',
      startDate: '2026-09-07',
      tasks: [
        { id: 'G', name: '基础', duration: 0, level: 0, progress: 0 },
        { id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 },
        { id: 'H', name: '养护', duration: 28, level: 1, progress: 0, durationUnit: 'cd' },
        { id: 'M', name: '竣工', duration: 0, level: 1, progress: 0, isMilestone: true },
      ],
      links: [{ from: 'A', to: 'H', type: 'FS', lag: 0 }],
    }
  }

  it('工期列单位 chip：cd 行带 cd 态、wd 行有入口、分组行与里程碑无入口', () => {
    const p = cdProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(screen.getByTestId('sched-unit-H').className).toContain('sched-unit-cd')
    expect(screen.getByTestId('sched-unit-A').className).not.toContain('sched-unit-cd')
    expect(screen.queryByTestId('sched-unit-M')).toBeNull()
    expect(screen.queryByTestId('sched-unit-G')).toBeNull()
  })

  it('点击 chip 切单位不换算数值（wd→cd 与 cd→wd 双向）', () => {
    const p = cdProject()
    useScheduleStore.setState({ project: p })
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.click(screen.getByTestId('sched-unit-A'))
    expect(useScheduleStore.getState().project.tasks.find((t) => t.id === 'A')).toMatchObject({ durationUnit: 'cd', duration: 2 })
    fireEvent.click(screen.getByTestId('sched-unit-H'))
    expect(useScheduleStore.getState().project.tasks.find((t) => t.id === 'H')).toMatchObject({ durationUnit: 'wd', duration: 28 })
  })

  it('cd 条形宽=真实日历跨度（28cd 等效 20 工作日跨周末=28 自然日列），title 含等效', () => {
    const p: SchedProject = {
      name: '单养护',
      startDate: '2026-09-07',
      tasks: [{ id: 'H', name: '养护', duration: 28, level: 1, progress: 0, durationUnit: 'cd' }],
      links: [],
    }
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links, { startDate: p.startDate })} />)
    // ef=20：dayNo(20)=28（跨 4 个周末的自然日列），dayW 缺省 20 → 宽 560px
    expect(barAt(0).style.width).toBe('560px')
    expect(barAt(0).getAttribute('title')).toContain('等效 20 工作日')
    expect(barAt(0).getAttribute('title')).toContain('日历天 28')
  })

  it('cd 缩放拖拽回写自然日（+120px 跨周末：wd 版=7 工作日，cd 版=9 自然日）', () => {
    const mk = (unit?: 'cd') => ({
      name: '缩放样板',
      startDate: '2026-09-07',
      tasks: [{ id: 'H', name: '养护', duration: 3, level: 1, progress: 0, ...(unit ? { durationUnit: unit } : {}) }],
      links: [],
    })
    const cd = mk('cd')
    useScheduleStore.setState({ project: cd })
    render(<GanttView project={cd} cpm={computeCpm(cd.tasks, cd.links, { startDate: cd.startDate })} />)
    fireEvent.mouseDown(barAt(0).querySelector<HTMLElement>('.sched-resize-handle')!, { clientX: 0 })
    fireEvent.mouseMove(window, { clientX: 120 }) // 右缘 3→9 自然日（下周三，工作日 wd7）
    fireEvent.mouseUp(window)
    expect(useScheduleStore.getState().project.tasks[0]).toMatchObject({ duration: 9, durationUnit: 'cd' })
  })
})

describe('GanttView Ctrl+滚轮缩放（v4.159）', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null })
  })

  it('Ctrl+滚轮放大日宽（20→27），普通滚轮不缩放（保持滚动行）', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const pane = screen.getByTestId('sched-gantt-canvas')
    // days=21（短计划下限），chartW=21×20=420
    const inner = pane.querySelector<HTMLElement>('.sched-gantt-canvas-inner')!
    expect(inner.style.width).toBe('420px')
    fireEvent.wheel(pane, { deltaY: -100, ctrlKey: true, clientX: 30 })
    expect(inner.style.width).toBe('567px') // 21×27
    // 普通滚轮：原生滚动行，日宽不动
    fireEvent.wheel(pane, { deltaY: -100, clientX: 30 })
    expect(inner.style.width).toBe('567px')
  })
})
