// ResourcePanel.test.tsx — 资源成本刀3 UI 用例（v4.124）
//
// 覆盖：资源弹层 CRUD/类型切换字段显隐/删除级联；任务行资源 Popover（勾选建分配、
// 取消删分配、units/quantity/amount 编辑、固定成本写入）；store action 容错
// （负数丢弃、分组行 no-op、整体替换清洗、removeTask 级联）；甘特成本列
// （叶=明细合计、分组=子孙求和、无数据留空、循环 fail-closed、分组行入口隐藏）；
// 状态栏总成本段条件显隐。成本期望值全部与 cost.test.ts 同公式口径。

import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { GanttView } from './GanttView'
import { ResourcePanel, TaskResourceEditor } from './ResourcePanel'
import { hasCostData } from './costUi'
import { computeCpm } from './cpm'
import { useScheduleStore } from './store'
import type { SchedAssignment, SchedCalendar, SchedProject, SchedResource, SchedTask } from './types'
import SchedulePage from '../pages/SchedulePage'

// 状态栏用例渲染 SchedulePage：左栏 ChatPane 依赖会话 store 全家桶，mock 掉保持用例聚焦
vi.mock('./ChatPane', () => ({ ScheduleChatPane: () => <div data-testid="mock-chat-pane" /> }))

function leaf(id: string, duration: number, extra: Partial<SchedTask> = {}): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0, ...extra }
}
function group(id: string): SchedTask {
  return { id, name: id, duration: 0, level: 0, progress: 0 }
}
function res(id: string, type: SchedResource['type'], extra: Partial<SchedResource> = {}): SchedResource {
  return { id, name: id, type, ...extra }
}
function asg(taskId: string, resourceId: string, extra: Partial<SchedAssignment> = {}): SchedAssignment {
  return { taskId, resourceId, ...extra }
}

// G1(A 3 + B 2)：A=r1 工时(3×2×100+200=800)+固定 50+c1 100=950；
// B=m1 材料 550 + c1 300=850。G1=1800，总=1800；byResource r1=800/m1=550/c1=400
function costedProject(): SchedProject {
  return {
    name: '成本 UI 样板',
    startDate: '2026-09-07',
    tasks: [group('G1'), leaf('A', 3, { fixedCost: 50 }), leaf('B', 2)],
    links: [],
    resources: [
      res('r1', 'work', { name: '人力', standardRate: 100, costPerUse: 200, maxUnits: 2 }),
      res('m1', 'material', { name: '钢材', standardRate: 55, unit: 't' }),
      res('c1', 'cost', { name: '差旅' }),
    ],
    assignments: [
      asg('A', 'r1', { units: 2 }),
      asg('A', 'c1', { amount: 100 }),
      asg('B', 'm1', { quantity: 10 }),
      asg('B', 'c1', { amount: 300 }),
    ],
  }
}

function plainProject(): SchedProject {
  return {
    name: '无成本样板',
    startDate: '2026-09-07',
    tasks: [group('G1'), leaf('A', 3), leaf('B', 2)],
    links: [],
  }
}

function renderPanel(p: SchedProject) {
  return render(<ResourcePanel cpm={computeCpm(p.tasks, p.links)} />)
}

// 打开行内类型下拉并选中选项文字（下拉落点在触发节点内；文字在 .ant-select-item-option-content 直下）
async function pickOption(row: HTMLElement, label: string) {
  fireEvent.mouseDown(row.querySelector('.ant-select-selector')!)
  const opt = await screen.findByText(label, { selector: '.ant-select-item-option-content' })
  fireEvent.click(opt.closest('.ant-select-item-option')!)
}

describe('ResourcePanel 资源弹层（刀3）', () => {
  beforeEach(() => {
    localStorage.removeItem('gaea.schedule.v1')
    useScheduleStore.setState({
      project: costedProject(),
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      savedAt: null,
      syncError: null,
    })
  })

  it('新增资源：默认 work 类型，id 自动生成，提交即写 store', () => {
    renderPanel(useScheduleStore.getState().project)
    const before = useScheduleStore.getState().project.resources!.length
    fireEvent.click(screen.getByTestId('sched-res-add'))
    const list = useScheduleStore.getState().project.resources!
    expect(list.length).toBe(before + 1)
    expect(list[list.length - 1].type).toBe('work')
    expect(list[list.length - 1].id).not.toBe('')
    expect(screen.getByTestId('sched-resource-panel').querySelectorAll('.sched-res-row:not(.sched-res-head)').length).toBe(before + 1)
  })

  it('编辑名称/费率/每次使用/maxUnits：写入 store，费率带 元/工日 标注', () => {
    renderPanel(useScheduleStore.getState().project)
    const row = screen.getByTestId('sched-res-row-r1')
    expect(row.textContent).toContain('元/工日')
    fireEvent.change(row.querySelector('input')!, { target: { value: '普工' } })
    const nums = row.querySelectorAll('.ant-input-number-input')
    fireEvent.change(nums[0], { target: { value: '350' } })
    fireEvent.change(nums[1], { target: { value: '80' } })
    fireEvent.change(nums[2], { target: { value: '3' } })
    const r = useScheduleStore.getState().project.resources!.find((x) => x.id === 'r1')!
    expect(r.name).toBe('普工')
    expect(r.standardRate).toBe(350)
    expect(r.costPerUse).toBe(80)
    expect(r.maxUnits).toBe(3)
  })

  it('类型切换→材料：出现计量单位输入，费率标注随 unit 变 元/单位→元/t', async () => {
    renderPanel(useScheduleStore.getState().project)
    await pickOption(screen.getByTestId('sched-res-row-r1'), '材料')
    let r = useScheduleStore.getState().project.resources!.find((x) => x.id === 'r1')!
    expect(r.type).toBe('material')
    expect(screen.getByTestId('sched-res-row-r1').textContent).toContain('元/单位')
    const unitInput = screen.getByTestId('sched-res-row-r1').querySelector('input[placeholder="t/m³"]')!
    fireEvent.change(unitInput, { target: { value: 't' } })
    r = useScheduleStore.getState().project.resources!.find((x) => x.id === 'r1')!
    expect(r.unit).toBe('t')
    expect(screen.getByTestId('sched-res-row-r1').textContent).toContain('元/t')
  })

  it('类型切换→成本：费率/每次使用隐藏（— 占位），无关字段在 store 清除', async () => {
    renderPanel(useScheduleStore.getState().project)
    await pickOption(screen.getByTestId('sched-res-row-r1'), '成本')
    const r = useScheduleStore.getState().project.resources!.find((x) => x.id === 'r1')!
    expect(r.type).toBe('cost')
    expect(r.standardRate).toBeUndefined()
    expect(r.costPerUse).toBeUndefined()
    expect(r.maxUnits).toBeUndefined()
    expect(screen.getByTestId('sched-res-row-r1').textContent).toContain('—')
  })

  it('已分配任务数去重计数；资源成本合计来自 computeCosts().byResource', () => {
    renderPanel(useScheduleStore.getState().project)
    expect(screen.getByTestId('sched-res-count-r1').textContent).toBe('1')
    expect(screen.getByTestId('sched-res-count-c1').textContent).toBe('2')
    expect(screen.getByTestId('sched-res-cost-r1').textContent).toBe('¥800')
    expect(screen.getByTestId('sched-res-cost-m1').textContent).toBe('¥550')
    expect(screen.getByTestId('sched-res-cost-c1').textContent).toBe('¥400')
  })

  it('删除资源（Popconfirm）：级联删除其全部分配，其余资源与分配保留', async () => {
    renderPanel(useScheduleStore.getState().project)
    fireEvent.click(screen.getByRole('button', { name: '删除资源人力' }))
    fireEvent.click(await screen.findByRole('button', { name: /OK|确定/ }))
    const p = useScheduleStore.getState().project
    expect(p.resources!.map((x) => x.id)).toEqual(['m1', 'c1'])
    expect(p.assignments!.map((x) => `${x.taskId}>${x.resourceId}`)).toEqual(['A>c1', 'B>m1', 'B>c1'])
  })
})

describe('TaskResourceEditor 任务行资源 Popover（刀3）', () => {
  beforeEach(() => {
    localStorage.removeItem('gaea.schedule.v1')
    useScheduleStore.setState({
      project: costedProject(),
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      savedAt: null,
      syncError: null,
    })
  })

  it('勾选资源即建分配、取消勾选即删（整体替换该任务分配集）', () => {
    render(<TaskResourceEditor task={leaf('A', 3)} />)
    expect(screen.queryByTestId('sched-link-row-A-m1')).not.toBeNull()
    const cb = () => screen.getByTestId('sched-link-row-A-m1').querySelector('input[type=checkbox]')!
    fireEvent.click(cb())
    const mine = () => (useScheduleStore.getState().project.assignments ?? []).filter((a) => a.taskId === 'A')
    expect(mine().some((a) => a.resourceId === 'm1')).toBe(true)
    fireEvent.click(cb())
    expect(mine().some((a) => a.resourceId === 'm1')).toBe(false)
  })

  it('units/quantity/amount 按资源类型分别显示并可编辑（写入 store）', () => {
    // work：A 已挂 r1 → units
    render(<TaskResourceEditor task={leaf('A', 3)} />)
    const rowW = screen.getByTestId('sched-link-row-A-r1')
    fireEvent.change(rowW.querySelector('.ant-input-number-input')!, { target: { value: '3' } })
    expect((useScheduleStore.getState().project.assignments ?? []).find((a) => a.taskId === 'A' && a.resourceId === 'r1')!.units).toBe(3)

    // material：B 已挂 m1 → quantity
    render(<TaskResourceEditor task={leaf('B', 2)} />)
    const rowM = screen.getByTestId('sched-link-row-B-m1')
    fireEvent.change(rowM.querySelector('.ant-input-number-input')!, { target: { value: '12' } })
    expect((useScheduleStore.getState().project.assignments ?? []).find((a) => a.taskId === 'B' && a.resourceId === 'm1')!.quantity).toBe(12)

    // cost：B 已挂 c1 → amount
    const rowC = screen.getByTestId('sched-link-row-B-c1')
    fireEvent.change(rowC.querySelector('.ant-input-number-input')!, { target: { value: '400' } })
    expect((useScheduleStore.getState().project.assignments ?? []).find((a) => a.taskId === 'B' && a.resourceId === 'c1')!.amount).toBe(400)
  })

  it('固定成本输入写入任务 fixedCost', () => {
    render(<TaskResourceEditor task={leaf('A', 3)} />)
    const fc = screen.getByTestId('sched-task-res-panel-A').querySelector('.ant-input-number-input')!
    fireEvent.change(fc, { target: { value: '120' } })
    expect(useScheduleStore.getState().project.tasks.find((t) => t.id === 'A')!.fixedCost).toBe(120)
  })
})

describe('store 资源 action 容错（刀3）', () => {
  beforeEach(() => {
    localStorage.removeItem('gaea.schedule.v1')
    useScheduleStore.setState({
      project: costedProject(),
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      savedAt: null,
      syncError: null,
    })
  })

  it('upsertResource：负数/NaN 数值字段丢弃回落缺省（undefined）', () => {
    useScheduleStore.getState().upsertResource({ id: 'bad', name: '坏数值', type: 'work', standardRate: -5, costPerUse: Number.NaN, maxUnits: Number.POSITIVE_INFINITY })
    const r = useScheduleStore.getState().project.resources!.find((x) => x.id === 'bad')!
    expect(r.standardRate).toBeUndefined()
    expect(r.costPerUse).toBeUndefined()
    expect(r.maxUnits).toBeUndefined()
  })

  it('setTaskAssignments：分组行 no-op（fail-closed）；非法条目清洗；整体替换', () => {
    const before = useScheduleStore.getState().project
    useScheduleStore.getState().setTaskAssignments('G1', [asg('G1', 'r1')])
    expect(useScheduleStore.getState().project).toBe(before) // 分组行拒绝：模型禁止

    useScheduleStore.getState().setTaskAssignments('A', [
      asg('A', 'r1', { units: -1 }), // 负值 → 字段丢弃
      asg('A', 'ghost'), // 悬空资源 → 丢弃
      asg('A', 'm1'),
      asg('A', 'm1', { quantity: 5 }), // 重复 (taskId,resourceId) → 后者丢弃
    ])
    const mine = (useScheduleStore.getState().project.assignments ?? []).filter((a) => a.taskId === 'A')
    expect(mine.map((x) => x.resourceId).sort()).toEqual(['m1', 'r1']) // 整体替换：原 c1 分配不在
    expect(mine.find((x) => x.resourceId === 'r1')!.units).toBeUndefined()
    // 其他任务分配不受影响
    expect((useScheduleStore.getState().project.assignments ?? []).filter((a) => a.taskId === 'B').length).toBe(2)
  })

  it('removeTask 级联删除该任务（含子孙）的全部分配', () => {
    useScheduleStore.getState().removeTask('B')
    const list = useScheduleStore.getState().project.assignments!
    expect(list.some((a) => a.taskId === 'B')).toBe(false)
    expect(list.some((a) => a.taskId === 'A')).toBe(true)
  })

  it('hasCostData：resources/assignments 任一非空或非零 fixedCost 即有数据', () => {
    expect(hasCostData(costedProject())).toBe(true)
    expect(hasCostData(plainProject())).toBe(false)
    expect(hasCostData({ ...plainProject(), tasks: [group('G1'), leaf('A', 3, { fixedCost: 0 })] })).toBe(false)
  })
})

describe('GanttView 成本列（刀3）', () => {
  beforeEach(() => {
    localStorage.removeItem('gaea.schedule.v1')
    useScheduleStore.setState({
      project: costedProject(),
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      savedAt: null,
      syncError: null,
    })
  })

  it('叶任务=明细合计（含固定成本）；分组行=子孙叶求和', () => {
    const p = costedProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(screen.getByTestId('sched-cost-A').textContent).toBe('950') // 800+固定 50
    expect(screen.getByTestId('sched-cost-B').textContent).toBe('850') // 550+300
    expect(screen.getByTestId('sched-cost-G1').textContent).toBe('1800') // 950+850
  })

  it('无任何资源/成本数据：列值留空（诚实呈现，不留 ¥0 假象）', () => {
    const p = plainProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(screen.getByTestId('sched-cost-A').textContent).toBe('')
    expect(screen.getByTestId('sched-cost-G1').textContent).toBe('')
  })

  it('CPM 循环依赖：成本不出（fail-closed），列值留空', () => {
    const p = costedProject()
    p.links.push({ from: 'A', to: 'B', type: 'FS', lag: 0 }, { from: 'B', to: 'A', type: 'FS', lag: 0 })
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(screen.getByTestId('sched-cost-A').textContent).toBe('')
    expect(screen.getByTestId('sched-cost-G1').textContent).toBe('')
  })

  it('行内资源入口：叶任务有、分组行无（fail-closed 口径）', () => {
    const p = costedProject()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    expect(screen.queryByTestId('sched-task-res-G1')).toBeNull()
    expect(screen.getByTestId('sched-task-res-A')).not.toBeNull()
  })
})

describe('状态栏总成本段（刀3）', () => {
  beforeEach(() => {
    localStorage.removeItem('gaea.schedule.v1')
    useScheduleStore.setState({
      project: costedProject(),
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      savedAt: null,
      syncError: null,
    })
  })

  it('有资源/成本数据：常驻显示 总成本 ¥N（computeCosts total）', () => {
    render(<SchedulePage />)
    expect(screen.getByTestId('sched-statusbar-cost').textContent).toContain('¥1800')
  })

  it('无资源/成本数据：不显示该段（不留 ¥0 假象）', () => {
    useScheduleStore.setState({ project: plainProject() })
    render(<SchedulePage />)
    expect(screen.queryByTestId('sched-statusbar-cost')).toBeNull()
  })
})

// ── v4.137 #13 资源级日历：work 行「日历」入口 + 个人周工作制/休假编辑器 ──
// 口径：只影响使用视图可用性/超载判定，不改任务排程；直接 upsertResource 落库无草稿态。

describe('ResourcePanel 资源个人日历（v4.137 #13）', () => {
  beforeEach(() => {
    localStorage.removeItem('gaea.schedule.v1')
    useScheduleStore.setState({
      project: costedProject(),
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      savedAt: null,
      syncError: null,
    })
  })

  /** store 中的 r1（每次断言现取，避免旧引用） */
  const stored = () => useScheduleStore.getState().project.resources!.find((x) => x.id === 'r1')!
  /** 打开 r1 行的日历编辑器 Popover */
  const openEditor = async () => {
    fireEvent.click(screen.getByTestId('sched-res-cal-r1'))
    await screen.findByRole('checkbox', { name: '周六' })
  }
  /** 预置 r1 的个人日历 */
  const withCalendar = (cal: SchedCalendar) => ({
    ...costedProject(),
    resources: costedProject().resources!.map((r) => (r.id === 'r1' ? { ...r, calendar: cal } : r)),
  })

  it('work 资源有「日历」入口（未自定义=default 态），material/cost 无', () => {
    renderPanel(useScheduleStore.getState().project)
    const entry = screen.getByTestId('sched-res-cal-r1')
    expect(entry.className).not.toContain('ant-btn-primary')
    expect(entry.getAttribute('title')).toContain('未自定义')
    expect(screen.queryByTestId('sched-res-cal-m1')).toBeNull()
    expect(screen.queryByTestId('sched-res-cal-c1')).toBeNull()
  })

  it('勾掉周六：upsertResource 落库 calendar.workweek 不含周六，入口转已自定义态', async () => {
    // 项目日历全勾：未自定义资源跟随之（缺省全勾视觉），勾掉周六即固化为个人日历
    useScheduleStore.setState({ project: { ...costedProject(), calendar: { workweek: [0, 1, 2, 3, 4, 5, 6], holidays: [] } } })
    renderPanel(useScheduleStore.getState().project)
    await openEditor()
    expect(screen.getByTestId('sched-res-cal-follow-r1').textContent).toContain('未自定义（跟随项目日历）')
    fireEvent.click(screen.getByRole('checkbox', { name: '周六' }))
    expect(stored().calendar!.workweek).toEqual([0, 1, 2, 3, 4, 5]) // 不含 6=周六
    expect(screen.getByTestId('sched-res-cal-r1').className).toContain('ant-btn-primary') // 已自定义态
    expect(screen.queryByTestId('sched-res-cal-follow-r1')).toBeNull()
  })

  it('节假日：添加入库（格式校验+去重），删 Tag 即移除', async () => {
    renderPanel(useScheduleStore.getState().project)
    await openEditor()
    const input = screen.getByPlaceholderText('2026-10-01')
    const addBtn = () => screen.getByRole('button', { name: /添\s*加/ }) // antd 两汉字按钮文案自动加空格
    fireEvent.change(input, { target: { value: '2026-10-01' } })
    fireEvent.click(addBtn())
    expect(stored().calendar!.holidays).toEqual(['2026-10-01'])
    fireEvent.click(addBtn()) // 同日期去重
    expect(stored().calendar!.holidays).toEqual(['2026-10-01'])
    fireEvent.change(input, { target: { value: 'abc' } }) // 非法格式拒绝
    fireEvent.click(addBtn())
    expect(stored().calendar!.holidays).toEqual(['2026-10-01'])
    fireEvent.click(screen.getByText('2026-10-01').closest('.ant-tag')!.querySelector('.anticon-close')!)
    expect(stored().calendar!.holidays).toEqual([])
  })

  it('「跟随项目日历」清除：resource.calendar 变 undefined，回未自定义态', async () => {
    useScheduleStore.setState({ project: withCalendar({ workweek: [1, 2, 3, 4], holidays: ['2026-10-01'] }) })
    renderPanel(useScheduleStore.getState().project)
    expect(screen.getByTestId('sched-res-cal-r1').className).toContain('ant-btn-primary')
    await openEditor()
    fireEvent.click(screen.getByRole('button', { name: /跟随项目日历/ }))
    expect(stored().calendar).toBeUndefined()
    expect(screen.getByTestId('sched-res-cal-r1').className).not.toContain('ant-btn-primary')
    expect(screen.getByTestId('sched-res-cal-follow-r1').textContent).toContain('未自定义（跟随项目日历）')
  })

  it('全取消周工作制被拒绝：仅剩的一项勾不掉（至少保留一项）', async () => {
    useScheduleStore.setState({ project: withCalendar({ workweek: [6], holidays: [] }) })
    renderPanel(useScheduleStore.getState().project)
    await openEditor()
    fireEvent.click(screen.getByRole('checkbox', { name: '周六' })) // 勾掉唯一工作日 → 该次点击无效
    expect(stored().calendar!.workweek).toEqual([6])
  })
})
