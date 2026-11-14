/**
 * ganttFilter.test.ts — 横道行筛选纯函数（刀C 余项）
 *
 * 用例口径：关键（TF=0 且非手动）/手动/里程碑/文本四模；分组行=有可见子孙才保留；
 * 文本不区分大小写；空文本=该模全量。
 * W5 追加：五类列过滤器（文本/数值/日期/枚举/空值）+ 无列筛零回归 + 列筛工具函数。
 */
import { describe, expect, it } from 'vitest'
import {
  activeColFilterCount,
  clearColFilter,
  filterGanttRows,
  GANTT_FILTER_DEFAULT,
  isColFilterActive,
  setColFilter,
  type ColFilterValue,
  type GanttFilter,
} from './ganttFilter'
import { isoOf, wdToDate } from './calendar'
import { computeCpm } from './cpm'
import type { SchedProject, SchedTask } from './types'

function t(id: string, name: string, duration: number, level = 1, extra: Partial<SchedTask> = {}): SchedTask {
  return { id, name, duration, level, progress: 0, ...extra }
}

const tasks: SchedTask[] = [
  t('G1', '前期准备', 0, 0),
  t('A', '场地三通一平', 10),
  t('B', '图纸会审', 5, 1, { mode: 'manual', manualStart: 0 }),
  t('G2', '主体结构', 0, 0),
  t('C', '主体浇筑', 25),
  t('G3', '其他工作', 0, 0),
  t('D', '独立任务', 2),
]
const links = [{ from: 'A', to: 'C', type: 'FS' as const, lag: 0 }]
const project: SchedProject = { name: 'x', startDate: '2026-09-07', tasks, links }
const cpm = computeCpm(project.tasks, project.links)

const idx = (id: string): number => tasks.findIndex((x) => x.id === id)

describe('filterGanttRows', () => {
  it('全部+空文本：全可见（含分组）', () => {
    expect(filterGanttRows(tasks, GANTT_FILTER_DEFAULT, cpm)).toEqual([0, 1, 2, 3, 4, 5, 6])
  })

  it('只看关键：链上叶+其分组可见，独立/手动叶隐藏，无可见子孙的分组不保留', () => {
    const rows = filterGanttRows(tasks, { kind: 'critical', text: '' }, cpm)
    // 关键=A、C（链）；G1 因 A 可见而保留，G2 因 C 可见保留，D 隐藏
    expect(rows).toEqual([idx('G1'), idx('A'), idx('G2'), idx('C')])
  })

  it('只看手动：手动叶可见、其分组保留', () => {
    const rows = filterGanttRows(tasks, { kind: 'manual', text: '' }, cpm)
    expect(rows).toEqual([idx('G1'), idx('B')])
  })

  it('只看里程碑：里程碑列表视图只保留里程碑叶与有可见子孙的分组', () => {
    const withMile = tasks.map((task) =>
      task.id === 'D' ? { ...task, isMilestone: true } : task,
    )
    expect(filterGanttRows(withMile, { kind: 'milestone', text: '' }, cpm)).toEqual([
      idx('G3'),
      idx('D'),
    ])
  })

  it('文本筛选：命中叶+命中分组（分组命中即保留）；大小写不敏感', () => {
    const rows = filterGanttRows(tasks, { kind: 'all', text: '浇筑' }, cpm)
    expect(rows).toEqual([idx('G2'), idx('C')])
    const branch = filterGanttRows(tasks, { kind: 'all', text: '前期' }, cpm)
    expect(branch, '分组名命中展示整支').toEqual([idx('G1'), idx('A'), idx('B')])
  })

  it('组合：关键+文本', () => {
    const rows = filterGanttRows(tasks, { kind: 'critical', text: '主体' }, cpm)
    expect(rows).toEqual([idx('G2'), idx('C')])
  })
})

// ── W5 列级筛选（五类）────────────────────────────────────────────

/** 列筛夹具：无依赖链（全部 es=0，日期列可比）、含 custom 槽与手动/进度差异 */
const cfx: SchedTask[] = [
  t('G1', '结构工程', 0, 0),
  t('A', '钢筋绑扎', 10),
  { ...t('B', '模板支设', 5), mode: 'manual', manualStart: 0, custom: { num1: 3, text1: '甲' } },
  t('G2', '装修工程', 0, 0),
  t('C', '涂料饰面', 25, 1, { progress: 40 }),
]
const cfxProject: SchedProject = {
  name: 'cfx',
  startDate: '2026-09-07',
  tasks: cfx,
  // A→C 一条 FS：让 C 的 es=10（日期列筛才有可分辨的工作日序号）
  links: [{ from: 'A', to: 'C', type: 'FS' as const, lag: 0 }],
}
const cfxCpm = computeCpm(cfxProject.tasks, cfxProject.links)
const ci = (id: string): number => cfx.findIndex((x) => x.id === id)
const f = (colFilters: Record<string, ColFilterValue>): GanttFilter => ({
  ...GANTT_FILTER_DEFAULT,
  colFilters,
})

describe('filterGanttRows · 列级筛选（W5）', () => {
  it('无列筛零回归：三参/四参/空 colFilters 三路结果逐字一致', () => {
    const base = filterGanttRows(cfx, GANTT_FILTER_DEFAULT, cfxCpm)
    expect(base).toEqual([0, 1, 2, 3, 4])
    expect(filterGanttRows(cfx, f({}), cfxCpm)).toEqual(base)
    expect(filterGanttRows(cfx, f({}), cfxCpm, { startDate: cfxProject.startDate })).toEqual(base)
  })

  it('文本列筛：名称包含（不区分大小写），分组行随可见子孙保留', () => {
    expect(filterGanttRows(cfx, f({ name: { kind: 'text', text: '模板' } }), cfxCpm)).toEqual([
      ci('G1'),
      ci('B'),
    ])
    expect(filterGanttRows(cfx, f({ text1: { kind: 'text', text: '甲' } }), cfxCpm)).toEqual([
      ci('G1'),
      ci('B'),
    ])
  })

  it('数值列筛：≥ / ≤ / = 三比较（工期、进度同源）', () => {
    expect(filterGanttRows(cfx, f({ dur: { kind: 'number', op: 'ge', num: 10 } }), cfxCpm)).toEqual([
      ci('G1'),
      ci('A'),
      ci('G2'),
      ci('C'),
    ])
    expect(filterGanttRows(cfx, f({ dur: { kind: 'number', op: 'le', num: 5 } }), cfxCpm)).toEqual([
      ci('G1'),
      ci('B'),
    ])
    expect(
      filterGanttRows(cfx, f({ progress: { kind: 'number', op: 'eq', num: 40 } }), cfxCpm),
    ).toEqual([ci('G2'), ci('C')])
  })

  it('数值列筛缺阈值=不约束（fail-open，不误杀）', () => {
    expect(filterGanttRows(cfx, f({ dur: { kind: 'number', op: 'ge' } }), cfxCpm)).toEqual([
      0, 1, 2, 3, 4,
    ])
  })

  it('日期列筛：区间闭区间（按开工日把工作日序号换算 ISO）', () => {
    const ctx = { startDate: cfxProject.startDate, calendar: cfxProject.calendar }
    const wd10 = isoOf(wdToDate(cfxProject.startDate, 10, cfxProject.calendar))
    expect(filterGanttRows(cfx, f({ start: { kind: 'date', from: wd10 } }), cfxCpm, ctx)).toEqual([
      ci('G2'),
      ci('C'),
    ])
    expect(
      filterGanttRows(cfx, f({ finish: { kind: 'date', to: wd10 } }), cfxCpm, ctx),
    ).toEqual([ci('G1'), ci('A'), ci('B')])
  })

  it('日期列筛缺 startDate 上下文 → 跳过该列筛（不整表清空）', () => {
    expect(filterGanttRows(cfx, f({ start: { kind: 'date', from: '2099-01-01' } }), cfxCpm)).toEqual([
      0, 1, 2, 3, 4,
    ])
  })

  it('枚举列筛：mode 集合（手动/自动）与 marker 集合', () => {
    expect(filterGanttRows(cfx, f({ mode: { kind: 'enum', set: ['manual'] } }), cfxCpm)).toEqual([
      ci('G1'),
      ci('B'),
    ])
    const withMile = cfx.map((x) => (x.id === 'C' ? { ...x, isMilestone: true } : x))
    const mileCpm = computeCpm(withMile, [])
    expect(
      filterGanttRows(withMile, f({ marker: { kind: 'enum', set: ['milestone'] } }), mileCpm),
    ).toEqual([ci('G2'), ci('C')])
  })

  it('空值口径：empty=只看空槽、nonEmpty=只看有值', () => {
    expect(filterGanttRows(cfx, f({ num1: { kind: 'number', empty: 'empty' } }), cfxCpm)).toEqual([
      ci('G1'),
      ci('A'),
      ci('G2'),
      ci('C'),
    ])
    expect(filterGanttRows(cfx, f({ num1: { kind: 'number', empty: 'nonEmpty' } }), cfxCpm)).toEqual([
      ci('G1'),
      ci('B'),
    ])
  })

  it('列筛是硬约束：文本强制集（分组名命中→整支可见）也要过列筛', () => {
    // 文本「结构」强制 G1 整支；列筛工期>=20 把 A(10)/B(5) 全刷掉 → G1 无可见子孙 → 整支消失
    const rows = filterGanttRows(
      cfx,
      { kind: 'all', text: '结构', colFilters: { dur: { kind: 'number', op: 'ge', num: 20 } } },
      cfxCpm,
    )
    expect(rows).toEqual([])
  })
})

describe('列筛工具函数', () => {
  it('isColFilterActive：空条件不算生效（激活态/计数口径）', () => {
    expect(isColFilterActive(undefined)).toBe(false)
    expect(isColFilterActive({ kind: 'text', text: '  ' })).toBe(false)
    expect(isColFilterActive({ kind: 'text', text: 'a' })).toBe(true)
    expect(isColFilterActive({ kind: 'number', op: 'ge' })).toBe(false)
    expect(isColFilterActive({ kind: 'date', from: '', to: '' })).toBe(false)
    expect(isColFilterActive({ kind: 'enum', set: [] })).toBe(false)
    expect(isColFilterActive({ kind: 'number', empty: 'empty' })).toBe(true)
  })

  it('setColFilter / clearColFilter：不可变写入，失效条件删键', () => {
    const a = setColFilter(undefined, 'dur', { kind: 'number', op: 'ge', num: 3 })
    expect(activeColFilterCount(a)).toBe(1)
    const b = setColFilter(a, 'dur', { kind: 'number', op: 'ge' })
    expect(b).toEqual({})
    expect(activeColFilterCount(a), '原对象不被改写').toBe(1)
    const c = setColFilter(a, 'name', { kind: 'text', text: 'x' })
    expect(activeColFilterCount(c)).toBe(2)
    expect(clearColFilter(c, 'name')).toEqual(a)
    expect(clearColFilter(undefined, 'name')).toEqual({})
  })
})
