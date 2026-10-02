/**
 * gantt/rowDerive.test.ts — 横道行派生单源（FE7-05）
 *
 * deriveRows 是表格窗格（GanttTable）与画布窗格（GanttBars）共用的唯一逐行
 * 派生：合成组头判定/synSeq/idx 提取/字段派生/类名拼接。用例按拆分前两窗格
 * 内联实现的口径逐字段断言（行为零变化的钉子）；反向证据用例见
 * 「组判定改错 → 全红」段（审计硬约束 2①）。
 */
import { describe, expect, it } from 'vitest'
import { deriveRows, isSynGroupRow } from './rowDerive'
import type { CpmResult, SchedProject, SchedTask, TaskCpm } from '../types'
import type { GanttRow } from '../ganttGroup'

/** G(合成组头) A B(wbs 组头) D(里程碑) E(普通叶)，与 identityRows 键口径一致 */
const rows: GanttRow[] = [
  { kind: 'group', key: '/critical:是', label: '关键: 是', level: 0, count: 2, minEs: 0, maxEf: 9 },
  { kind: 'task', idx: 0, level: 1 },
  { kind: 'group', key: 'wbs:2', label: '土建', level: 0, count: 2, minEs: 0, maxEf: 5 },
  { kind: 'task', idx: 3, level: 1 },
  { kind: 'task', idx: 4, level: 1 },
]

const tasks: SchedTask[] = [
  { id: 'A', name: '挖土', duration: 2.4, level: 1, progress: 0 },
  { id: 'B', name: '垫层', duration: 2, level: 0, progress: 0 },
  { id: 'C', name: '埋管', duration: 5, level: 1, progress: 0 },
  { id: 'D', name: '竣工', duration: 0, level: 1, progress: 0, isMilestone: true },
  { id: 'E', name: '回填', duration: 3, level: 1, progress: 0, mode: 'manual' },
]

const cpmRow = (es: number, ef: number): TaskCpm => ({ es, ef, ls: es, lf: ef, tf: 0, ff: 0, critical: true })
const cpm: CpmResult = {
  ok: true,
  duration: 9,
  rows: { A: cpmRow(0, 2), B: cpmRow(2, 5), C: cpmRow(3, 9), D: cpmRow(9, 9), E: cpmRow(0, 3) },
}

const project = { name: '样板', startDate: '2026-09-07', tasks, links: [] } as unknown as SchedProject
const groupColorSeq = [1, 1, 2, 2, 2]
const noDim = (): boolean => false

describe('deriveRows（FE7-05 单源）', () => {
  it('合成组头判定：分组组头=syn、wbs: 组头行=非 syn（走任务行分支）', () => {
    const derived = deriveRows(rows, project, cpm, groupColorSeq, noDim)
    expect(derived.map((d) => d.syn)).toEqual([true, false, false, false, false])
    expect(isSynGroupRow(rows[0])).toBe(true)
    expect(isSynGroupRow(rows[2])).toBe(false)
    expect(isSynGroupRow({ kind: 'task', idx: 0, level: 1 })).toBe(false)
  })

  it('synSeq 只对合成组头递增（与拆分前 `let synSeq = -1` 闭包同值）', () => {
    const derived = deriveRows(rows, project, cpm, groupColorSeq, noDim)
    expect(derived.map((d) => d.synSeq)).toEqual([0, 0, 0, 0, 0])
    const many: GanttRow[] = [
      { kind: 'group', key: '/critical:是', label: '', level: 0, count: 1, minEs: 0, maxEf: 1 },
      { kind: 'task', idx: 0, level: 1 },
      { kind: 'group', key: '/mode:手动', label: '', level: 1, count: 1, minEs: 0, maxEf: 1 },
      { kind: 'group', key: '/mode:自动', label: '', level: 1, count: 1, minEs: 0, maxEf: 1 },
    ]
    expect(deriveRows(many, project, cpm, groupColorSeq, noDim).map((d) => d.synSeq)).toEqual([0, 0, 1, 2])
  })

  it('idx 提取：task 行=idx、wbs: 行=反解 key（slice(4)）', () => {
    const derived = deriveRows(rows, project, cpm, groupColorSeq, noDim)
    expect(derived.map((d) => (d.syn ? -1 : d.i))).toEqual([-1, 0, 2, 3, 4])
  })

  it('字段派生：group/span 只对 wbs 组行；dur=里程碑 0、四舍五入；manual 取任务态', () => {
    const derived = deriveRows(rows, project, cpm, groupColorSeq, noDim)
    const wbs = asTask(derived[2])
    expect(wbs.group).toBe(true)
    expect(wbs.span).toEqual({ es: 0, ef: 5 })
    expect(wbs.manual).toBe(false)
    const leaf = asTask(derived[1])
    expect(leaf.group).toBe(false)
    expect(leaf.span).toBeNull()
    expect(leaf.dur).toBe(2) // 2.4 → round → 2
    expect(leaf.manual).toBe(false)
    const mile = asTask(derived[3])
    expect(mile.dur).toBe(0) // 里程碑
    expect(mile.t.isMilestone).toBe(true)
    const manual = asTask(derived[4])
    expect(manual.manual).toBe(true)
  })

  it('CPM 行随任务 id 取（row=cpm.rows[t.id]）；colorCls 只对组行、含组配色序号', () => {
    const derived = deriveRows(rows, project, cpm, groupColorSeq, noDim)
    const wbs = asTask(derived[2])
    expect(wbs.row).toBe(cpm.rows.C)
    expect(wbs.colorCls).toBe(' sched-group-c2')
    const leaf = asTask(derived[1])
    expect(leaf.row).toBe(cpm.rows.A)
    expect(leaf.colorCls).toBe('')
  })

  it('dimCls：rowDim 命中的非组行加 sched-row-dim（含前导空格）；组行不淡化', () => {
    const dimOnlyA = (id: string): boolean => id === 'A'
    const derived = deriveRows(rows, project, cpm, groupColorSeq, dimOnlyA)
    const a = asTask(derived[1])
    expect(a.dimCls).toBe(' sched-row-dim')
    const wbs = asTask(derived[2])
    expect(wbs.dimCls).toBe('')
    const others = derived.filter((d, k) => k !== 1 && !d.syn).map((d) => asTask(d))
    expect(others.every((d) => d.dimCls === '')).toBe(true)
  })
})

/** TS 收窄辅助：非 syn 行字段访问（syn 行抛错=用例失败） */
function asTask(d: ReturnType<typeof deriveRows>[number]): Extract<ReturnType<typeof deriveRows>[number], { syn: false }> {
  if (d.syn) throw new Error('预期非合成组头行')
  return d
}
