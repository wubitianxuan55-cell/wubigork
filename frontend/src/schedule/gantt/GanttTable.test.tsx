/**
 * gantt/GanttTable.test.tsx — 表格窗格 W5 交互（就地编辑弹层 + 列级筛选）
 *
 * 用例口径：双击名称格开弹层 / 合法提交走既有 onUpdateTask / Esc 取消不写回 /
 * 非法值（倒序区间）原位提示且不写回 / 列筛变更经 onColFiltersChange 上抛 /
 * 未接线列筛（无 handler）时按钮不出现且旧的单击直编零回归。
 */
import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { SchedProject, SchedTask } from '../types'
import type { GanttRow } from '../ganttGroup'
import { visibleCols, visibleLeftW } from '../ganttCols'
import { computeCpm } from '../cpm'
import { GanttTable } from './GanttTable'

const tasks: SchedTask[] = [
  { id: 'G', name: '结构工程', duration: 0, level: 0, progress: 0 },
  { id: 'A', name: '钢筋绑扎', duration: 10, level: 1, progress: 20, custom: { num1: 3 } },
]

const project: SchedProject = { name: 'p', startDate: '2026-09-07', tasks, links: [] }
const cpm = computeCpm(project.tasks, project.links)
const rows: GanttRow[] = [
  { kind: 'group', key: 'wbs:0', label: '结构工程', level: 0, count: 1, minEs: 0, maxEf: 10 },
  { kind: 'task', idx: 1, level: 1 },
]

/** 表格渲染夹具：默认不接列筛（验证零回归基线） */
function renderTable(over: Record<string, unknown> = {}) {
  const onUpdateTask = vi.fn()
  const onColFiltersChange = vi.fn()
  const onRangeCommit = vi.fn()
  const onSelect = vi.fn()
  render(
    <GanttTable
      leftW={visibleLeftW([])}
      cols={visibleCols([])}
      tbodyRef={null}
      rows={rows}
      wbs={['1', '1.1']}
      project={project}
      cpm={cpm}
      selectedId={null}
      costShown={false}
      costs={{ ok: true, rows: {}, byResource: {}, total: 0 }}
      groupColorSeq={[0, 0]}
      collapsedKeys={new Set<string>()}
      colOn={() => true}
      rowDim={() => false}
      onSelect={onSelect}
      onUpdateTask={onUpdateTask}
      onToggleCollapse={vi.fn()}
      rowMenu={() => []}
      onRowMenuClick={vi.fn()}
      onColFiltersChange={onColFiltersChange}
      onRangeCommit={onRangeCommit}
      {...over}
    />,
  )
  return { onUpdateTask, onColFiltersChange, onRangeCommit, onSelect }
}

/** 取某任务某列的就地编辑锚点（包住既有直编控件） */
function anchorOf(testId: string): HTMLElement {
  return screen.getByTestId(`${testId}-anchor`)
}

/** 双击名称格 → 打开统一弹层，返回弹层输入框 */
function openNameEditor(): HTMLInputElement {
  const cellInput = within(anchorOf('sched-inplace-name-A')).getByDisplayValue('钢筋绑扎')
  fireEvent.doubleClick(cellInput)
  const editor = screen.getByTestId('sched-inplace-name-A')
  return editor.querySelector('input') as HTMLInputElement
}

describe('GanttTable · 双击就地编辑（W5）', () => {
  it('双击名称格打开统一弹层，弹层预填当前名称', () => {
    renderTable()
    expect(screen.queryByTestId('sched-inplace-name-A')).toBeNull()
    const input = openNameEditor()
    expect(input.value).toBe('钢筋绑扎')
  })

  it('合法提交写回：Enter 后走既有 onUpdateTask(id, patch)', () => {
    const { onUpdateTask } = renderTable()
    const input = openNameEditor()
    fireEvent.change(input, { target: { value: '  钢筋绑扎（一区） ' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onUpdateTask).toHaveBeenCalledWith('A', { name: '钢筋绑扎（一区）' })
  })

  it('Esc 取消：不调用 onUpdateTask', () => {
    const { onUpdateTask } = renderTable()
    const input = openNameEditor()
    fireEvent.change(input, { target: { value: '不要这个名' } })
    fireEvent.keyDown(input, { key: 'Escape' })
    expect(onUpdateTask).not.toHaveBeenCalled()
  })

  it('非法值不写回：倒序日期区间提交被拒，原位提示（不写回 onRangeCommit）', () => {
    const { onRangeCommit } = renderTable({
      rangeCols: ['planRange'],
      rangeValueOf: () => ({ from: '2026-10-01', to: '2026-09-30' }),
    })
    const anchor = screen.getByTestId('sched-inplace-planRange-A-anchor')
    // 区间列单元格内容=只读文本（无 input），双击文本即开弹层
    fireEvent.doubleClick(anchor.firstElementChild ?? anchor)
    const editor = screen.getByTestId('sched-inplace-planRange-A')
    const input = editor.querySelector('input') as HTMLInputElement
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onRangeCommit).not.toHaveBeenCalled()
    expect(within(editor).getByTestId('sched-inplace-planRange-A-error').textContent).toBe(
      '开始日期不能晚于结束日期',
    )
  })
})

describe('GanttTable · 列级筛选（W5）', () => {
  it('表头有筛选按钮；数值列筛变更经 onColFiltersChange 上抛', () => {
    const { onColFiltersChange } = renderTable({ colFilters: {} })
    const btn = screen.getByTestId('sched-colfilter-dur')
    fireEvent.click(btn)
    const numInput = within(screen.getByTestId('sched-colfilter-editor-dur')).getByRole('spinbutton')
    fireEvent.change(numInput, { target: { value: '10' } })
    expect(onColFiltersChange).toHaveBeenCalled()
    expect(onColFiltersChange).toHaveBeenLastCalledWith({
      dur: { kind: 'number', op: 'ge', num: 10 },
    })
  })

  it('生效列筛显示激活态；清除按钮回传去除该列的空表', () => {
    const { onColFiltersChange } = renderTable({
      colFilters: { dur: { kind: 'number', op: 'ge', num: 3 } },
    })
    const btn = screen.getByTestId('sched-colfilter-dur')
    expect(btn.className).toContain('sched-inplace-filter-on')
    fireEvent.click(btn)
    fireEvent.click(screen.getByTestId('sched-colfilter-clear-dur'))
    expect(onColFiltersChange).toHaveBeenLastCalledWith({})
  })

  it('未接线列筛：不渲染筛选按钮，旧单击直编照旧写回（零回归）', () => {
    const { onUpdateTask } = renderTable({ onColFiltersChange: undefined })
    expect(screen.queryByTestId('sched-colfilter-dur')).toBeNull()
    const nameInput = within(anchorOf('sched-inplace-name-A')).getByDisplayValue('钢筋绑扎')
    fireEvent.change(nameInput, { target: { value: '改名（单击直编）' } })
    expect(onUpdateTask).toHaveBeenCalledWith('A', { name: '改名（单击直编）' })
  })
})

// ── v3 交互语义（行为线）：键盘可达 ─────────────────────────────────────
//
// 口径：断言**用户可见行为**——roving tabindex 的实际值、↑/↓ 真的换选中并把 DOM
// 焦点带过去、Enter 真的调检查器回调、F2 真的开出名称弹层、表头筛选按钮的
// aria-expanded 真的跟着 Popover 开合、Esc 真的回焦。不测内部 state 名。
describe('GanttTable · 键盘可达（v3：roving / ↑↓ / Enter / F2 / 筛选浮层）', () => {
  const tasks2: SchedTask[] = [
    { id: 'G', name: '结构工程', duration: 0, level: 0, progress: 0 },
    { id: 'A', name: '钢筋绑扎', duration: 4, level: 1, progress: 0 },
    { id: 'B', name: '模板支设', duration: 3, level: 1, progress: 0 },
  ]
  const project2: SchedProject = { name: 'p2', startDate: '2026-09-07', tasks: tasks2, links: [] }
  const cpm2 = computeCpm(project2.tasks, project2.links)
  const rows2: GanttRow[] = [
    { kind: 'group', key: 'wbs:0', label: '结构工程', level: 0, count: 2, minEs: 0, maxEf: 7 },
    { kind: 'task', idx: 1, level: 1 },
    { kind: 'task', idx: 2, level: 1 },
  ]
  /** 两叶任务夹具（↑/↓ 才有落点）；其余 props 沿用 renderTable 的缺省 */
  const renderTwo = (over: Record<string, unknown> = {}) =>
    renderTable({
      rows: rows2,
      project: project2,
      cpm: cpm2,
      wbs: ['1', '1.1', '1.2'],
      groupColorSeq: [0, 0, 0],
      ...over,
    })
  const rowEls = () => Array.from(document.querySelectorAll<HTMLElement>('.sched-gantt-tbody .sched-gantt-row'))

  it('roving tabindex：无选中时首个任务行为 0，分组行与其余行 -1', () => {
    renderTwo()
    expect(rowEls()[0].tabIndex).toBe(-1) // 分组行不参与选中序列
    expect(rowEls()[1].tabIndex).toBe(0) // 无选中 → 首个任务行是 Tab 落点
    expect(rowEls()[2].tabIndex).toBe(-1)
  })

  it('roving tabindex：选中行独占 0，其余 -1', () => {
    renderTwo({ selectedId: 'B' })
    expect(rowEls()[1].tabIndex).toBe(-1)
    expect(rowEls()[2].tabIndex).toBe(0)
  })

  it('↓ / ↑ 移动选中并把 DOM 焦点带到新行（roving 不只是属性）', () => {
    const { onSelect } = renderTwo()
    const first = rowEls()[1]
    first.focus()
    expect(document.activeElement).toBe(first)
    fireEvent.keyDown(first, { key: 'ArrowDown' })
    expect(onSelect).toHaveBeenCalledWith('B')
    expect(document.activeElement).toBe(rowEls()[2])
    fireEvent.keyDown(document.activeElement as HTMLElement, { key: 'ArrowUp' })
    expect(onSelect).toHaveBeenLastCalledWith('A')
    expect(document.activeElement).toBe(rowEls()[1])
  })

  it('Enter 打开任务检查器（与右键菜单同一条 onInspect 回调）', () => {
    const onInspect = vi.fn()
    renderTwo({ onInspect })
    fireEvent.keyDown(rowEls()[1], { key: 'Enter' })
    expect(onInspect).toHaveBeenCalledWith('A')
    fireEvent.keyDown(rowEls()[2], { key: 'Enter' })
    expect(onInspect).toHaveBeenLastCalledWith('B')
  })

  it('F2 进入名称就地编辑：弹层打开、预填当前名称且输入框已获焦', () => {
    renderTwo()
    expect(screen.queryByTestId('sched-inplace-name-A')).toBeNull()
    fireEvent.keyDown(rowEls()[1], { key: 'F2' })
    const editor = screen.getByTestId('sched-inplace-name-A')
    const input = within(editor).getByDisplayValue('钢筋绑扎')
    expect(document.activeElement).toBe(input)
  })

  it('行内控件的 Enter 归控件本身（输入框/按钮），不冒泡成「开检查器」', () => {
    const onInspect = vi.fn()
    renderTwo({ onInspect })
    const nameInput = within(anchorOf('sched-inplace-name-A')).getByDisplayValue('钢筋绑扎')
    fireEvent.keyDown(nameInput, { key: 'Enter' })
    // 行内按钮同理：Enter 是「激活这个按钮」，不该被行抢成检查器
    fireEvent.keyDown(screen.getByTestId('sched-task-res-A'), { key: 'Enter' })
    expect(onInspect).not.toHaveBeenCalled()
  })

  it('表头筛选按钮：aria-expanded 随 Popover 开合，Esc 收起并把焦点还给该按钮', () => {
    renderTwo({ colFilters: {} })
    const btn = screen.getByTestId('sched-colfilter-dur')
    expect(btn.getAttribute('aria-expanded')).toBe('false')
    expect(btn.getAttribute('aria-haspopup')).toBe('dialog')
    btn.focus()
    fireEvent.click(btn)
    expect(btn.getAttribute('aria-expanded')).toBe('true')
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(btn.getAttribute('aria-expanded')).toBe('false')
    expect(document.activeElement).toBe(btn)
  })
})
