/**
 * ganttGroup.test.ts — 横道排序/分组纯函数用例
 *
 * 口径：缺省恒等（原序+原 WBS 组头合成统计）；critical/mode 单级与两级分组
 * 的合成组头；折叠只裁命中组头的子树；duration/progress/name/start/tf 排序
 * 的稳定性（同值保原相对顺序）与 zh 拼音序；visibleIdx 子集统计、空分支与
 * 空组裁剪；cpm.ok=false 不抛错、统计按 0。
 */
import { describe, expect, it } from 'vitest'
import { buildGanttRows } from './ganttGroup'
import type { CpmResult, SchedTask, TaskCpm } from './types'

function t(id: string, name: string, duration: number, level = 1, extra: Partial<SchedTask> = {}): SchedTask {
  return { id, name, duration, level, progress: 0, ...extra }
}

function row(es: number, ef: number, tf: number, critical = false): TaskCpm {
  return { es, ef, ls: es, lf: ef, tf, ff: 0, critical }
}

/** 两级大纲：G1[A(防水,关键), B(地基,手动)] G2[C(材料,关键), D(竣工节点,里程碑)] G3[E(安装,手动)] */
const tasks: SchedTask[] = [
  t('G1', '前期准备', 0, 0),
  t('A', '防水施工', 10, 1, { progress: 50 }),
  t('B', '地基处理', 10, 1, { mode: 'manual', progress: 100 }),
  t('G2', '主体结构', 0, 0),
  t('C', '材料检验', 25, 1, { progress: 20 }),
  t('D', '竣工节点', 0, 1, { isMilestone: true }),
  t('G3', '其他工作', 0, 0),
  t('E', '安装工程', 2, 1, { mode: 'manual', progress: 80 }),
]

const cpm: CpmResult = {
  ok: true,
  duration: 35,
  rows: {
    A: row(0, 10, 0, true),
    B: row(2, 12, 5),
    C: row(10, 35, 0, true),
    D: row(35, 35, 2),
    E: row(5, 7, 28),
  },
}

const seqOf = (rows: ReturnType<typeof buildGanttRows>): (string | number)[] =>
  rows.map((r) => (r.kind === 'task' ? r.idx : r.key))

describe('buildGanttRows', () => {
  it('缺省恒等：行序与原数组一致，原分组行合成为 group 且 count/minEs/maxEf 正确', () => {
    const rows = buildGanttRows(tasks, cpm)
    expect(seqOf(rows)).toEqual(['wbs:0', 1, 2, 'wbs:3', 4, 5, 'wbs:6', 7])
    expect(rows[0]).toEqual({ kind: 'group', key: 'wbs:0', label: '前期准备', level: 0, count: 2, minEs: 0, maxEf: 12 })
    expect(rows[3]).toEqual({ kind: 'group', key: 'wbs:3', label: '主体结构', level: 0, count: 2, minEs: 10, maxEf: 35 })
    expect(rows[6]).toEqual({ kind: 'group', key: 'wbs:6', label: '其他工作', level: 0, count: 1, minEs: 5, maxEf: 7 })
    expect(rows[1]).toEqual({ kind: 'task', idx: 1, level: 1 })
  })

  it("field 'none' 等于不排序：与缺省输出一致", () => {
    expect(buildGanttRows(tasks, cpm, { sort: { field: 'none', dir: 'desc' } })).toEqual(buildGanttRows(tasks, cpm))
  })

  it('单级分组 critical：组头 label/key/level/count 正确、子叶 level=1、分支按首次出现序', () => {
    expect(buildGanttRows(tasks, cpm, { groupBy: ['critical'] })).toEqual([
      { kind: 'group', key: '/critical:是', label: '关键: 是', level: 0, count: 2, minEs: 0, maxEf: 35 },
      { kind: 'task', idx: 1, level: 1 },
      { kind: 'task', idx: 4, level: 1 },
      { kind: 'group', key: '/critical:否', label: '关键: 否', level: 0, count: 3, minEs: 2, maxEf: 35 },
      { kind: 'task', idx: 2, level: 1 },
      { kind: 'task', idx: 5, level: 1 },
      { kind: 'task', idx: 7, level: 1 },
    ])
  })

  it('空分支不产出：visibleIdx 只留关键叶时只产「是」组', () => {
    expect(buildGanttRows(tasks, cpm, { groupBy: ['critical'], visibleIdx: [1, 4] })).toEqual([
      { kind: 'group', key: '/critical:是', label: '关键: 是', level: 0, count: 2, minEs: 0, maxEf: 35 },
      { kind: 'task', idx: 1, level: 1 },
      { kind: 'task', idx: 4, level: 1 },
    ])
  })

  it("两级分组 ['critical','mode']：两级组头嵌套、叶 level=2、组头统计只计本支叶", () => {
    expect(buildGanttRows(tasks, cpm, { groupBy: ['critical', 'mode'] })).toEqual([
      { kind: 'group', key: '/critical:是', label: '关键: 是', level: 0, count: 2, minEs: 0, maxEf: 35 },
      { kind: 'group', key: '/critical:是/mode:自动', label: '模式: 自动', level: 1, count: 2, minEs: 0, maxEf: 35 },
      { kind: 'task', idx: 1, level: 2 },
      { kind: 'task', idx: 4, level: 2 },
      { kind: 'group', key: '/critical:否', label: '关键: 否', level: 0, count: 3, minEs: 2, maxEf: 35 },
      { kind: 'group', key: '/critical:否/mode:手动', label: '模式: 手动', level: 1, count: 2, minEs: 2, maxEf: 12 },
      { kind: 'task', idx: 2, level: 2 },
      { kind: 'task', idx: 7, level: 2 },
      { kind: 'group', key: '/critical:否/mode:自动', label: '模式: 自动', level: 1, count: 1, minEs: 35, maxEf: 35 },
      { kind: 'task', idx: 5, level: 2 },
    ])
  })

  it('折叠第二级组头 key：只藏该组子树（B、E 消失），组头自身保留', () => {
    const rows = buildGanttRows(tasks, cpm, {
      groupBy: ['critical', 'mode'],
      collapsed: new Set(['/critical:否/mode:手动']),
    })
    expect(seqOf(rows)).toEqual([
      '/critical:是',
      '/critical:是/mode:自动',
      1,
      4,
      '/critical:否',
      '/critical:否/mode:手动',
      '/critical:否/mode:自动',
      5,
    ])
  })

  it('折叠一级组：子树全消失、组头在', () => {
    expect(buildGanttRows(tasks, cpm, { groupBy: ['critical'], collapsed: new Set(['/critical:是']) })).toEqual([
      { kind: 'group', key: '/critical:是', label: '关键: 是', level: 0, count: 2, minEs: 0, maxEf: 35 },
      { kind: 'group', key: '/critical:否', label: '关键: 否', level: 0, count: 3, minEs: 2, maxEf: 35 },
      { kind: 'task', idx: 2, level: 1 },
      { kind: 'task', idx: 5, level: 1 },
      { kind: 'task', idx: 7, level: 1 },
    ])
  })

  it('折叠原 WBS 组头（不分组模式）：组头在、直接子叶隐藏', () => {
    expect(seqOf(buildGanttRows(tasks, cpm, { collapsed: new Set(['wbs:0']) }))).toEqual([
      'wbs:0',
      'wbs:3',
      4,
      5,
      'wbs:6',
      7,
    ])
  })

  it('排序 duration desc：组内生效且同值稳定（A/B 同为 10 天保原相对顺序）', () => {
    const rows = buildGanttRows(tasks, cpm, { groupBy: ['critical'], sort: { field: 'duration', dir: 'desc' } })
    // 全局降序 25=C, 10=A, 10=B（同值原序）, 2=E, 0=D → 是[C,A] 否[B,E,D]
    expect(seqOf(rows)).toEqual(['/critical:是', 4, 1, '/critical:否', 2, 7, 5])
  })

  it('排序不分组：各 WBS 组内子叶重排、组头位置不动（progress desc：B 100 > A 50）', () => {
    expect(seqOf(buildGanttRows(tasks, cpm, { sort: { field: 'progress', dir: 'desc' } }))).toEqual([
      'wbs:0',
      2,
      1,
      'wbs:3',
      4,
      5,
      'wbs:6',
      7,
    ])
  })

  it('排序 name：zh 拼音序（地<防 升序重排；竣>材 降序重排）', () => {
    const asc = buildGanttRows(tasks, cpm, { sort: { field: 'name', dir: 'asc' } })
    expect(seqOf(asc)).toEqual(['wbs:0', 2, 1, 'wbs:3', 4, 5, 'wbs:6', 7])
    const desc = buildGanttRows(tasks, cpm, { sort: { field: 'name', dir: 'desc' } })
    expect(seqOf(desc)).toEqual(['wbs:0', 1, 2, 'wbs:3', 5, 4, 'wbs:6', 7])
  })

  it('排序 start：取 CPM es（B es2 > A es0；D es35 > C es10）', () => {
    const rows = buildGanttRows(tasks, cpm, { sort: { field: 'start', dir: 'desc' } })
    expect(seqOf(rows)).toEqual(['wbs:0', 2, 1, 'wbs:3', 5, 4, 'wbs:6', 7])
  })

  it('排序 tf asc：同值 0 跨组稳定（A 原序先于 C）', () => {
    const rows = buildGanttRows(tasks, cpm, { groupBy: ['critical'], sort: { field: 'tf', dir: 'asc' } })
    // 全局升序 A(0,idx1), C(0,idx4), D(2), B(5), E(28) → 是[A,C] 否[D,B,E]
    expect(seqOf(rows)).toEqual(['/critical:是', 1, 4, '/critical:否', 5, 2, 7])
  })

  it('visibleIdx 子集（不分组）：组头 count/minEs/maxEf 只计可见叶', () => {
    const rows = buildGanttRows(tasks, cpm, { visibleIdx: [2, 4, 7] })
    expect(rows).toEqual([
      { kind: 'group', key: 'wbs:0', label: '前期准备', level: 0, count: 1, minEs: 2, maxEf: 12 },
      { kind: 'task', idx: 2, level: 1 },
      { kind: 'group', key: 'wbs:3', label: '主体结构', level: 0, count: 1, minEs: 10, maxEf: 35 },
      { kind: 'task', idx: 4, level: 1 },
      { kind: 'group', key: 'wbs:6', label: '其他工作', level: 0, count: 1, minEs: 5, maxEf: 7 },
      { kind: 'task', idx: 7, level: 1 },
    ])
  })

  it('visibleIdx 空组：某组可见叶为 0 则整组不产出', () => {
    expect(buildGanttRows(tasks, cpm, { visibleIdx: [4] })).toEqual([
      { kind: 'group', key: 'wbs:3', label: '主体结构', level: 0, count: 1, minEs: 10, maxEf: 35 },
      { kind: 'task', idx: 4, level: 1 },
    ])
  })

  it('visibleIdx（分组）：分支只按可见叶产出，原「否」叶先现故先产', () => {
    const rows = buildGanttRows(tasks, cpm, { groupBy: ['critical'], visibleIdx: [2, 4, 7] })
    expect(rows).toEqual([
      { kind: 'group', key: '/critical:否', label: '关键: 否', level: 0, count: 2, minEs: 2, maxEf: 12 },
      { kind: 'task', idx: 2, level: 1 },
      { kind: 'task', idx: 7, level: 1 },
      { kind: 'group', key: '/critical:是', label: '关键: 是', level: 0, count: 1, minEs: 10, maxEf: 35 },
      { kind: 'task', idx: 4, level: 1 },
    ])
  })

  it('cpm.ok=false：不抛错、统计按 0、critical 全落「否」、同值排序保原序', () => {
    const bad: CpmResult = { ok: false, error: '循环依赖', cycle: ['A', 'C'], rows: {}, duration: 0 }
    expect(() =>
      buildGanttRows(tasks, bad, { groupBy: ['critical'], sort: { field: 'start', dir: 'desc' } }),
    ).not.toThrow()
    expect(buildGanttRows(tasks, bad, { groupBy: ['critical'], sort: { field: 'start', dir: 'desc' } })).toEqual([
      { kind: 'group', key: '/critical:否', label: '关键: 否', level: 0, count: 5, minEs: 0, maxEf: 0 },
      { kind: 'task', idx: 1, level: 1 },
      { kind: 'task', idx: 2, level: 1 },
      { kind: 'task', idx: 4, level: 1 },
      { kind: 'task', idx: 5, level: 1 },
      { kind: 'task', idx: 7, level: 1 },
    ])
    const flat = buildGanttRows(tasks, bad)
    expect(
      flat.every((r) => r.kind !== 'group' || (r.minEs === 0 && r.maxEf === 0 && r.count > 0)),
    ).toBe(true)
  })
})
