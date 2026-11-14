/**
 * GanttView.test.tsx — 横道拖拽交互（刀9）+ 工作台双栏（刀C）+ 自定义字段列（v4.138 #14）
 *
 * 用例口径：fireEvent 鼠标序列（down→move→up）驱动 window 监听器，
 * 断言 store 落库结果（拖移 auto→转 manual 锁定、缩放→改工期、
 * 未移动=无操作、Esc=取消）；刀C：分隔条拖动收纳表格/双击复位/列显隐
 * （行号名称固定，其余单列可藏）持久化 chatPrefs；v4.138 #14：自定义列
 * 开启/改名/叶行编辑（清空=删槽键）/分组行留空；W1：9 视图预设与默认
 * 计划编制 7 列/真实列宽。
 */
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { GanttView } from './GanttView'
import { computeCpm } from './cpm'
import { useScheduleStore } from './store'
import { CHAT_PREFS_KEY } from './chatPrefs'
import { GANTT_LEFT_W_FULL, GANTT_TABLE_W_MIN } from './ganttCols'
import { hitRegionChain, pickHitRegion } from './hitRegion'
import { HIT_STRUCTURE_KEYS } from './hitMenu'
import { BUILTIN_GANTT_VIEWS, DEFAULT_GANTT_VIEW_ID, widthForBuiltinView } from './viewPresets'
import { GANTT_RULER_H } from './timescale'
import type { CpmResult, SchedProject } from './types'

const PLAN_VIEW = BUILTIN_GANTT_VIEWS.find((view) => view.id === DEFAULT_GANTT_VIEW_ID)!
/** W1 缺省=计划编制，表格宽=7 个真实列宽之和；不再把全列宽塞进双栏。 */
const DEF_HIDE_W = GANTT_LEFT_W_FULL - widthForBuiltinView(PLAN_VIEW)

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

  it('缺省计划编制：只露 7 个工作列，窗格宽=可见列总宽，列菜单可开', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - DEF_HIDE_W}px`)
    expect(tablePane().textContent).toContain('分区')
    expect(tablePane().textContent).toContain('标记')
    expect(tablePane().textContent).not.toContain('最迟开始')
    expect(tablePane().textContent).not.toContain('进度')
    expect(tablePane().textContent).not.toContain('文本1')
  })

  it('切换「进度管控」一次应用列集、分组、排序、筛选并落本机偏好', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.mouseDown(screen.getByTestId('sched-gantt-view').querySelector('.ant-select-selector')!)
    fireEvent.click(screen.getByText('进度管控', { exact: true }))
    expect(tablePane().textContent).toContain('进度')
    expect(tablePane().textContent).toContain('执行状态: 未开始')
    const prefs = JSON.parse(localStorage.getItem(CHAT_PREFS_KEY)!)
    expect(prefs.ganttViewId).toBe('control')
    expect(prefs.ganttGroup).toEqual(['status'])
    expect(prefs.ganttFilterKind).toBe('all')
    expect(prefs.ganttSort).toEqual({ field: 'start', dir: 'asc' })
  })

  it('分隔条拖动：左移收纳表格并持久化；拖到最窄剩行号+名称', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const split = screen.getByTestId('sched-gantt-split')
    fireEvent.mouseDown(split, { clientX: 900 })
    fireEvent.mouseMove(window, { clientX: 900 - 300 })
    fireEvent.mouseUp(window)
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - DEF_HIDE_W - 300}px`)
    expect(JSON.parse(localStorage.getItem(CHAT_PREFS_KEY)!).ganttTableW).toBe(GANTT_LEFT_W_FULL - DEF_HIDE_W - 300)
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
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - DEF_HIDE_W - 300}px`)
    fireEvent.dblClick(screen.getByTestId('sched-gantt-split'))
    // 复位=钳到可见列总宽（进度列+自定义列仍缺省隐藏）
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - DEF_HIDE_W}px`)
  })

  it('列显隐：取消勾选「工期」后表头消失并持久化；固定列不提供勾选', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.click(screen.getByTestId('sched-gantt-cols-btn'))
    const menu = screen.getByTestId('sched-gantt-colmenu')
    expect(menu.textContent).not.toContain('工作名称') // 行号/名称固定不进菜单
    const durLabel = Array.from(menu.querySelectorAll('label')).find((l) => l.textContent?.includes('工期'))!
    fireEvent.click(durLabel.querySelector('input')!)
    expect(tablePane().textContent).not.toContain('工期')
    expect(tablePane().textContent).toContain('开始')
    const prefs = JSON.parse(localStorage.getItem(CHAT_PREFS_KEY)!)
    expect(prefs.ganttHide).toContain('dur')
    expect(prefs.ganttViewId).toBe('custom')
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
    expect(tablePane().textContent).toContain('工作名称')
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
    enableCol('成本')
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
    expect(weeks[0].textContent).toBe('W1') // 复刻页首周段标注 W1
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

  it('Ctrl+滚轮放大日宽（20→27），普通滚轮不缩放（保持滚动行）', async () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const pane = screen.getByTestId('sched-gantt-canvas')
    // days=21（短计划下限），chartW=21×20=420
    const inner = pane.querySelector<HTMLElement>('.sched-gantt-canvas-inner')!
    expect(inner.style.width).toBe('420px')
    fireEvent.wheel(pane, { deltaY: -100, ctrlKey: true, clientX: 30 })
    // v4.371：wheel→setDayW 走 rAF 合并（一帧最多一次），jsdom 下等待帧提交
    await act(async () => { await new Promise((r) => setTimeout(r, 20)) })
    expect(inner.style.width).toBe('567px') // 21×27
    // 普通滚轮：原生滚动行，日宽不动
    fireEvent.wheel(pane, { deltaY: -100, clientX: 30 })
    expect(inner.style.width).toBe('567px')
  })
})


// ── 斑马复刻 S2 接线：前锋线默认跟随最新一期检视 ──────────────
describe('前锋线跟随最新检视', () => {
  const withReview = (): SchedProject => {
    const p = chainProject()
    // 第 2 工作日（2026-09-09）：A 完成 100%，B 完成 50%
    return {
      ...p,
      tasks: p.tasks.map((t) => (t.id === 'A' ? { ...t, progress: 100 } : t.id === 'B' ? { ...t, progress: 50 } : t)),
      reviews: [{ at: 2, date: '2026-09-09', actuals: { A: { percent: 100, finish: 2 }, B: { percent: 50 } } }],
    }
  }

  it('有检视：默认打开前锋线并停在那一期（sched-front-check / sched-front-line 出现）', () => {
    const p = withReview()
    useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
    const { container } = render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} onInspect={() => {}} />)
    expect(container.querySelector('.sched-front-check')).not.toBeNull()
    expect(container.querySelector('.sched-front-line')).not.toBeNull()
  })

  it('无检视：默认不画前锋线（保持旧观感，用户手动开）', () => {
    const p = chainProject()
    useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
    const { container } = render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} onInspect={() => {}} />)
    expect(container.querySelector('.sched-front-check')).toBeNull()
  })
})

// ── W4 双标尺接线（本条为新增守卫，未改动任何既有断言）───────────────────
describe('W4 双标尺对齐：表头高 = 画布顶部偏移 = timescale.GANTT_RULER_H（常量只定义一次）', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', syncError: null })
  })

  it('画布表头/表格表头/画布 --sched-ruler-h 三处同值，标尺两行高度合计 = 该值', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const canvasHead = document.querySelector('.sched-gantt-canvaspane .sched-gantt-head') as HTMLElement
    const tableHead = document.querySelector('.sched-gantt-tablepane .sched-gantt-head') as HTMLElement
    const inner = document.querySelector('.sched-gantt-canvas-inner') as HTMLElement
    expect(canvasHead.style.height).toBe(`${GANTT_RULER_H}px`)
    // 表格窗格表头仍是 40px 字面量（GanttTable 不在本线足迹内）——这里守住"两者必须相等"
    expect(tableHead.style.height).toBe(canvasHead.style.height)
    // 画布顶部偏移来源：CSS 变量（ruler.css 与后续覆盖层偏移读同一个值）
    expect(inner.style.getPropertyValue('--sched-ruler-h')).toBe(`${GANTT_RULER_H}px`)
    // 标尺两行常驻，高度合计 = 表头高
    const rows = Array.from(canvasHead.querySelectorAll<HTMLElement>('.sched-ruler-row'))
    expect(rows).toHaveLength(2)
    expect(rows.reduce((s, el) => s + Number.parseFloat(el.style.height), 0)).toBe(GANTT_RULER_H)
  })
})

// ── W4 横道网络图（斑马 ganttNetwork）：关系线加重 + 类型/时距标注 ──────────
describe('W4 横道网络图：关系线标注只在 ganttNetwork 模式出现', () => {
  const netProject = (mode: 'timescaled' | 'ganttNetwork'): SchedProject => ({
    ...chainProject(),
    networkStyle: { mode },
  })

  it('ganttNetwork：每条可见依赖线带类型+时距标签，并挂加重类名', () => {
    const p = netProject('ganttNetwork')
    useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
    const { container } = render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const badges = Array.from(container.querySelectorAll('.sched-link-badge'))
    expect(badges.length).toBe(p.links.length)
    expect(badges.map((b) => b.textContent).join(' ')).toContain('FS')
    expect(container.querySelectorAll('.sched-link-net').length).toBe(p.links.length)
  })

  it('timescaled：同一工程不出现标注与加重类名（旧观感零变化）', () => {
    const p = netProject('timescaled')
    useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
    const { container } = render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(container.querySelector('.sched-link-badge')).toBeNull()
    expect(container.querySelector('.sched-link-net')).toBeNull()
  })
})

// ── W5 列筛接线：表头 ▼ → 可见行收敛 → 清除信号恢复 ───────────────────────
describe('W5 列级筛选：表头条件作用于可见行，清除信号复原', () => {
  const dataRows = () =>
    document.querySelectorAll('[data-testid="sched-gantt-canvas"] .sched-gantt-row:not(.sched-gantt-head)').length

  it('按「工作名称」筛选后横道行收敛，收到 column-filter-clear 后复原', async () => {
    const p = chainProject()
    useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(dataRows()).toBe(3)
    fireEvent.click(screen.getByTestId('sched-colfilter-name'))
    const input = screen.getByTestId('sched-colfilter-editor-name').querySelector('input') as HTMLInputElement
    fireEvent.change(input, { target: { value: '垫层' } })
    await waitFor(() => expect(dataRows()).toBe(1))
    act(() => { useScheduleStore.getState().sendGanttCommand('column-filter-clear') })
    await waitFor(() => expect(dataRows()).toBe(3))
  })
})

// ── ZPert8 W4 剩余项「命中区顺序」+ W5 剩余项「右键按命中类型追加」───────────
/**
 * 口径：`进度计划/gaea-schedule-zpert8-uiux-port-plan-20261114.md` §W4/§W5、
 * `...uiux-re-20261114.md` §15.3（按 hitType 往同一 vector 追加条目）/§16.2
 * （统一命中 → 命中对象走对象菜单 / 未命中走空白侧）。
 * 这里断言的是**接线**：data-hit 挂在哪几层、单一右键入口选中的菜单来源是否正确、
 * 禁用项是否带原因、真动作是否落到 store。
 */
describe('命中区与统一右键（W4/W5 剩余项）', () => {
  const HIT = 'data-hit'
  const tableRows = () =>
    Array.from(
      screen.getByTestId('sched-gantt-table').querySelectorAll<HTMLElement>('.sched-gantt-tbody .sched-gantt-row'),
    )
  const cellsOf = (row: HTMLElement) => Array.from(row.querySelectorAll<HTMLElement>('.sched-gantt-cell'))
  const menuEl = () => screen.queryByTestId('sched-hit-menu')
  const menuKeys = () =>
    Array.from(menuEl()?.querySelectorAll('[data-hit-key]') ?? []).map((el) => el.getAttribute('data-hit-key'))
  const menuNode = (key: string) => {
    const el = menuEl()?.querySelector<HTMLElement>(`[data-hit-key="${key}"]`)
    if (!el) throw new Error(`右键菜单无「${key}」`)
    return el
  }
  const liOf = (key: string) => menuNode(key).closest('li') as HTMLElement

  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({
      project: chainProject(),
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      syncError: null,
      past: [],
      future: [],
    })
  })

  it('data-hit 分层：列头 / 单元格 / 行号格 / 行容器（兜底区）/ 画布 / 分隔条', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const head = screen.getByTestId('sched-gantt-table').querySelector<HTMLElement>('.sched-gantt-head')!
    expect(head.getAttribute(HIT)).toBe('column-header')
    const rows = tableRows()
    expect(rows.length).toBe(3)
    // 行容器=行头兜底（优先级最低），单元格的 cell 会把它盖掉
    expect(rows[0].getAttribute(HIT)).toBe('row-header')
    const cells = cellsOf(rows[0])
    expect(cells[0].classList.contains('sched-row-no')).toBe(true)
    expect(cells[0].getAttribute(HIT)).toBe('row-header')
    const dataCells = cells.filter((cell) => !cell.classList.contains('sched-row-no'))
    expect(dataCells.length).toBeGreaterThan(0)
    expect(dataCells.every((cell) => cell.getAttribute(HIT) === 'cell')).toBe(true)
    // 画布/分隔条由视图 JSX 直接声明
    expect(screen.getByTestId('sched-gantt-canvas').getAttribute(HIT)).toBe('canvas')
    expect(screen.getByTestId('sched-gantt-split').getAttribute(HIT)).toBe('splitter')
    // 优先级裁决在同一份真实 DOM 上成立：单元格压过行容器
    expect(pickHitRegion(hitRegionChain(dataCells[0]))).toBe('cell')
    expect(pickHitRegion(hitRegionChain(rows[0]))).toBe('row-header')
  })

  it('单元格右键：单一入口弹菜单（不叠行内 Dropdown），结构操作 12 项固定序，首项「添加工作」', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const rows = tableRows()
    const dataCell = cellsOf(rows[1]).find((cell) => !cell.classList.contains('sched-row-no'))!
    fireEvent.contextMenu(dataCell)
    expect(menuEl()).not.toBeNull()
    expect(menuEl()!.getAttribute('data-hit-region')).toBe('cell')
    // 子组件的 antd Dropdown 被根入口在捕获阶段拦下：只有一份菜单
    expect(document.querySelectorAll('.ant-dropdown').length).toBe(0)
    expect(menuKeys().slice(0, 12)).toEqual([...HIT_STRUCTURE_KEYS])
    expect(menuNode('add-work').textContent).toBe('添加工作')
    // 命中对象 → 先选中（右键行 B）
    expect(useScheduleStore.getState().selectedId).toBe('B')
  })

  it('分组行右键：「降级」禁用并带原因（前面没有分组=无处归属），「升级」也禁用', () => {
    const p = groupProject() // G(土建) 是首个分组行，前面没有别的分组
    useScheduleStore.setState({ project: p })
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.contextMenu(tableRows()[0])
    expect(useScheduleStore.getState().selectedId).toBe('G')
    expect(liOf('demote').className).toContain('ant-menu-item-disabled')
    expect(menuNode('demote').getAttribute('title')).toBe('降级后无处归属（前面没有分组行）')
    expect(menuNode('promote').getAttribute('title')).toBe('分组行不支持该操作，请选中一条工作')
    // 分组行不给工作对象专有项
    expect(liOf('milestone').className).toContain('ant-menu-item-disabled')
  })

  it('列头右键：「插入列 / 隐藏该列 / 列设置… / 列宽自适应」，点「隐藏该列」真隐藏', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const head = screen.getByTestId('sched-gantt-table').querySelector<HTMLElement>('.sched-gantt-head')!
    const durCell = Array.from(head.children).find((el) => el.textContent?.includes('工期')) as HTMLElement
    fireEvent.contextMenu(durCell)
    expect(menuEl()!.getAttribute('data-hit-region')).toBe('column-header')
    expect(menuKeys()).toEqual(['column.insert', 'column.hide', 'column.settings', 'column.autofit'])
    expect(menuNode('column.hide').textContent).toBe('隐藏该列')
    expect(liOf('column.insert').className).toContain('ant-menu-item-disabled')
    fireEvent.click(menuNode('column.hide'))
    expect(menuEl()).toBeNull()
    expect(tablePane().textContent).not.toContain('工期') // 与列菜单同一 toggleCol 路径
    expect(JSON.parse(localStorage.getItem(CHAT_PREFS_KEY)!).ganttHide).toContain('dur')
  })

  it('空白画布右键：出画布项（不含结构 12 项），「全览整计划」可点', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.contextMenu(screen.getByTestId('sched-gantt-canvas'))
    expect(menuEl()!.getAttribute('data-hit-region')).toBe('canvas')
    expect(menuKeys()).toContain('canvas.fit')
    expect(menuKeys()).not.toContain('add-work')
    expect(menuNode('canvas.fit').textContent).toBe('全览整计划')
    // 没有选中时插入类项禁用并给原因（不假动作）
    expect(menuNode('canvas.insert-pre').getAttribute('title')).toBe('请先选中一行作为插入参照')
    expect(liOf('canvas.insert-pre').className).toContain('ant-menu-item-disabled')
  })

  it('分隔条右键：出「复位表格宽度」，点击=复位全列宽（与双击同一条路径）', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const split = screen.getByTestId('sched-gantt-split')
    fireEvent.mouseDown(split, { clientX: 900 })
    fireEvent.mouseMove(window, { clientX: 600 })
    fireEvent.mouseUp(window)
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - DEF_HIDE_W - 300}px`)
    fireEvent.contextMenu(split)
    expect(menuEl()!.getAttribute('data-hit-region')).toBe('splitter')
    expect(menuKeys()).toEqual(['splitter.reset', 'splitter.collapse'])
    fireEvent.click(menuNode('splitter.reset'))
    expect(tablePane().style.width).toBe(`${GANTT_LEFT_W_FULL - DEF_HIDE_W}px`)
  })

  it('未命中任何命中区（视图根节点）→ 不弹菜单；Esc 收起已开的菜单', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.contextMenu(screen.getByTestId('sched-gantt'))
    expect(menuEl()).toBeNull()
    fireEvent.contextMenu(tableRows()[0])
    expect(menuEl()).not.toBeNull()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(menuEl()).toBeNull()
  })
})

// ── v3 交互语义（行为线）：横道表格与右键菜单的键盘可达 ──────────────────
//
// 口径：断言用户可见行为与 AT 可读语义——roving tabindex 的真实值、↑/↓ 真的换选中
// 并搬焦点、Enter 真的开检查器、F2 真的开出名称弹层、右键菜单容器的 role/aria-disabled
// 与视觉禁用同源、Esc 收起后焦点回到命中的那一行。不测内部 state 名。
describe('GanttView 键盘可达（v3：行 roving / Enter / F2 / 菜单语义）', () => {
  const tableRows = () =>
    Array.from(
      screen.getByTestId('sched-gantt-table').querySelectorAll<HTMLElement>('.sched-gantt-tbody .sched-gantt-row'),
    )
  const dataCellOf = (row: HTMLElement) =>
    Array.from(row.querySelectorAll<HTMLElement>('.sched-gantt-cell')).find((c) => !c.classList.contains('sched-row-no'))!
  const menuEl = () => screen.queryByTestId('sched-hit-menu')

  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({
      project: chainProject(),
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      syncError: null,
      past: [],
      future: [],
    })
  })

  it('roving tabindex：无选中时首任务行 0，其余 -1；选中 B 后 B 行独占 0', () => {
    const p = chainProject()
    const { unmount } = render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(tableRows().map((r) => r.tabIndex)).toEqual([0, -1, -1])
    unmount()
    useScheduleStore.setState({ selectedId: 'B' })
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(tableRows().map((r) => r.tabIndex)).toEqual([-1, 0, -1])
  })

  it('↓ 移动选中到下一行并让焦点跟过去（表格与画布同步换选中）', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const first = tableRows()[0]
    first.focus()
    fireEvent.keyDown(first, { key: 'ArrowDown' })
    expect(useScheduleStore.getState().selectedId).toBe('B')
    expect(document.activeElement).toBe(tableRows()[1])
    expect(tableRows()[1].className).toContain('sched-row-selected')
  })

  it('Enter 打开任务检查器（与右键菜单「任务检查器」同一条 onInspect 回调）', () => {
    const p = chainProject()
    const onInspect = vi.fn()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} onInspect={onInspect} />)
    fireEvent.keyDown(tableRows()[2], { key: 'Enter' })
    expect(onInspect).toHaveBeenCalledWith('C')
  })

  it('F2 开名称就地编辑：弹层出现且输入框获焦（与双击同一受控 open）', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(screen.queryByTestId('sched-inplace-name-A')).toBeNull()
    fireEvent.keyDown(tableRows()[0], { key: 'F2' })
    const editor = screen.getByTestId('sched-inplace-name-A')
    expect(document.activeElement).toBe(editor.querySelector('input'))
  })

  it('右键菜单容器 role=menu/menuitem，禁用项 aria-disabled 与视觉禁用同步', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    fireEvent.contextMenu(dataCellOf(tableRows()[1]))
    const menu = menuEl()!.querySelector('[role="menu"]')
    expect(menu).not.toBeNull()
    const items = Array.from(menuEl()!.querySelectorAll<HTMLElement>('[role="menuitem"]'))
    expect(items.length).toBeGreaterThan(0)
    // 「解组 / 格式刷 / 文字颜色 / 粘贴」在无剪贴板上下文里诚实禁用：视觉类与 ARIA 同源
    const disabledLabels = items
      .filter((li) => li.getAttribute('aria-disabled') === 'true')
      .map((li) => li.textContent?.trim())
    expect(disabledLabels).toContain('解组')
    expect(disabledLabels).toContain('粘贴')
    // 可用项不能同时挂 aria-disabled="true"（否则读屏比视觉更严，按键提示会误报）
    const enabled = items.filter((li) => !li.className.includes('ant-menu-item-disabled'))
    expect(enabled.every((li) => li.getAttribute('aria-disabled') !== 'true')).toBe(true)
  })

  it('Esc 收起右键菜单并把焦点还给命中的那一行（键盘用户不掉回文档根）', () => {
    const p = chainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const row = tableRows()[1]
    row.focus()
    fireEvent.contextMenu(dataCellOf(row))
    expect(menuEl()).not.toBeNull()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(menuEl()).toBeNull()
    expect(document.activeElement).toBe(row)
  })
})

// ── W5 右键真动作：格式刷 / 列宽自适应（斑马 12 项里的两项）────────────────
describe('W5 右键真动作：格式刷与列宽自适应', () => {
  const styledProject = (): SchedProject => {
    const p = chainProject()
    return { ...p, tasks: p.tasks.map((t) => (t.id === 'A' ? { ...t, style: { color: '#d4380d' } } : t)) } // hex-exempt
  }
  const paneW = () => Number.parseFloat((document.querySelector('[data-testid="sched-gantt-table"]') as HTMLElement).style.width)

  it('格式刷：取 A 外观后点 B，B 继承同一 style；刷子自动收起', () => {
    const p = styledProject()
    useScheduleStore.setState({ project: p, selectedId: 'A', hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
    const { container } = render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    act(() => { useScheduleStore.getState().sendGanttCommand('format-brush') })
    expect(container.querySelector('.sched-gantt')?.className).toContain('sched-brush-mode')
    // 表格窗格第 2 行 = B（链式样板 A/B/C 顺序）
    const rows = container.querySelectorAll('.sched-gantt-tbody .sched-gantt-row')
    fireEvent.click(rows[1] as HTMLElement)
    const b = useScheduleStore.getState().project.tasks.find((t) => t.id === 'B')
    expect(b?.style?.color).toBe('#d4380d') // hex-exempt
    expect(container.querySelector('.sched-gantt')?.className).not.toContain('sched-brush-mode')
  })

  it('列宽自适应：表头右键 → 表格窗格铺开到可见列合计', () => {
    // 先把表格窗格收窄（模拟用户拖过），自适应才有可见变化
    localStorage.setItem(CHAT_PREFS_KEY, JSON.stringify({ ganttTableW: 400 }))
    const p = chainProject()
    useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const before = paneW()
    const head = document.querySelector('[data-testid="sched-gantt-table"] .sched-gantt-head') as HTMLElement
    fireEvent.contextMenu(head, { clientX: 30, clientY: 10 })
    const item = document.querySelector('[data-hit-key="column.autofit"]') as HTMLElement
    expect(item).not.toBeNull()
    fireEvent.click(item)
    expect(paneW()).toBeGreaterThan(before)
  })
})

// ── 树形拖动（斑马「树形拖动」）：表格行拖拽改行序/层级 ───────────────────
describe('树形拖动：表格行拖放落到 store.moveRow', () => {
  it('拖 B 到 A 上方 → 顺序变为 垫层/挖土/浇筑（同级 before）', () => {
    const p = chainProject()
    useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
    const { container } = render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const rows = container.querySelectorAll('.sched-gantt-tbody .sched-gantt-row')
    const dt = { effectAllowed: '', dropEffect: '', setData: vi.fn(), getData: () => 'B' }
    fireEvent.dragStart(rows[1], { dataTransfer: dt })
    act(() => { rows[0].dispatchEvent(new MouseEvent('dragover', { bubbles: true, clientY: 2 })) })
    expect(rows[0].className).toContain('sched-drop-before')
    act(() => { rows[0].dispatchEvent(new MouseEvent('drop', { bubbles: true, clientY: 2 })) })
    expect(useScheduleStore.getState().project.tasks.map((t) => t.name)).toEqual(['垫层', '挖土', '浇筑'])
  })

  it('拖工作到分区行中段 → 变成子工作（inside）', () => {
    const p: SchedProject = {
      name: '分组样板', startDate: '2026-09-07',
      tasks: [
        { id: 'G1', name: '分区一', duration: 0, level: 0, progress: 0 },
        { id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 },
        { id: 'G2', name: '分区二', duration: 0, level: 0, progress: 0 },
        { id: 'B', name: '垫层', duration: 2, level: 1, progress: 0 },
      ],
      links: [],
    }
    useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
    const { container } = render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const rows = container.querySelectorAll('.sched-gantt-tbody .sched-gantt-row')
    const dt = { effectAllowed: '', dropEffect: '', setData: vi.fn(), getData: () => 'B' }
    fireEvent.dragStart(rows[3], { dataTransfer: dt })
    act(() => { rows[0].dispatchEvent(new MouseEvent('dragover', { bubbles: true, clientY: 15 })) })
    expect(rows[0].className).toContain('sched-drop-inside')
    act(() => { rows[0].dispatchEvent(new MouseEvent('drop', { bubbles: true, clientY: 15 })) })
    const after = useScheduleStore.getState().project.tasks
    expect(after.map((t) => `${t.id}:${t.level}`)).toEqual(['G1:0', 'A:1', 'B:1', 'G2:0'])
  })
})
