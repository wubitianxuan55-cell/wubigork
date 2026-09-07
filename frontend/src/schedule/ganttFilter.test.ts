/**
 * ganttFilter.test.ts — 横道行筛选纯函数（刀C 余项）
 *
 * 用例口径：关键（TF=0 且非手动）/手动/文本三模；分组行=有可见子孙才保留；
 * 文本不区分大小写；空文本=该模全量。
 */
import { describe, expect, it } from 'vitest'
import { filterGanttRows, GANTT_FILTER_DEFAULT } from './ganttFilter'
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
