import { describe, expect, it } from 'vitest'
import { computeCpm } from './cpm'
import { planFinishOf } from './deadline'
import {
  sortWarns,
  taskWarn,
  warnBand,
  warnSummary,
  askFinishOfDate,
  askFinishDateOf,
  type TaskWarn,
} from './warn'
import type { SchedProject, SchedTask } from './types'

/** 最小工程：A(4) → B(4)，FS；B 的 CPM ef=8（es=4） */
function proj(tasks: SchedTask[]): SchedProject {
  const base: SchedTask[] = [
    { id: 'A', name: 'A', duration: 4, level: 1, progress: 0 },
    { id: 'B', name: 'B', duration: 4, level: 1, progress: 0 },
  ]
  return {
    name: 'P',
    startDate: '2026-09-07', // 周一
    tasks: [...base, ...tasks],
    links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }],
  }
}

function cpmOf(p: SchedProject) {
  return computeCpm(p.tasks, p.links, { planFinish: planFinishOf(p) })
}

describe('warnBand（斑马滞后五档：0-15/15-30/30-60/60-100/100+）', () => {
  it('档位边界：(0,15]→0-15，(15,30]→15-30，(30,60]→30-60，(60,100]→60-100，>100→100天以上', () => {
    expect(warnBand(1)).toBe('0-15天')
    expect(warnBand(15)).toBe('0-15天')
    expect(warnBand(16)).toBe('15-30天')
    expect(warnBand(30)).toBe('15-30天')
    expect(warnBand(31)).toBe('30-60天')
    expect(warnBand(60)).toBe('30-60天')
    expect(warnBand(61)).toBe('60-100天')
    expect(warnBand(100)).toBe('60-100天')
    expect(warnBand(101)).toBe('100天以上')
    expect(warnBand(400)).toBe('100天以上')
  })
  it('0 与负数无滞后档（0=压线未突破，负=提前）', () => {
    expect(warnBand(0)).toBeNull()
    expect(warnBand(-3)).toBeNull()
  })
})

describe('taskWarn（两算法：plan-ask 计划-要求 / predict-ask 预测-要求）', () => {
  it('plan-ask 红灯：计划完成晚于要求完成 → overdue=ef-ask，档位', () => {
    // B2 无搭接：es=0，d=4 → 完成日=ef-1=3；ask=2 → 红灯滞后 1
    const p2 = proj([{ id: 'B2', name: 'B2', duration: 4, level: 1, progress: 0, askFinish: 2 }])
    const c = cpmOf(p2)
    const w = taskWarn(p2.tasks[2], c, null, 'plan-ask')
    expect(w).not.toBeNull()
    expect(w!.level).toBe('red')
    expect(w!.overdue).toBe(1) // ef=3 - ask=2
    expect(w!.band).toBe('0-15天')
    expect(w!.compareFinish).toBe(3)
  })

  it('plan-ask 绿灯：压线/提前 → band=null；overdue=0 也是绿灯（红灯=突破）', () => {
    const p = proj([{ id: 'B2', name: 'B2', duration: 4, level: 1, progress: 0, askFinish: 3 }])
    const c = cpmOf(p)
    const w = taskWarn(p.tasks[2], c, null, 'plan-ask')!
    expect(w.level).toBe('green')
    expect(w.overdue).toBe(0)
    expect(w.band).toBeNull()
    const pAhead = proj([{ id: 'B2', name: 'B2', duration: 4, level: 1, progress: 0, askFinish: 8 }])
    const w2 = taskWarn(pAhead.tasks[2], cpmOf(pAhead), null, 'plan-ask')!
    expect(w2.overdue).toBe(-5)
    expect(w2.level).toBe('green')
  })

  it('无要求时间 → none（斑马「无级别」），overdue/compareFinish 为 null', () => {
    const p = proj([{ id: 'B2', name: 'B2', duration: 4, level: 1, progress: 0 }])
    const w = taskWarn(p.tasks[2], cpmOf(p), null, 'plan-ask')!
    expect(w.level).toBe('none')
    expect(w.overdue).toBeNull()
    expect(w.compareFinish).toBeNull()
  })

  it('分组行不参与预警（返回 null）', () => {
    const p = proj([{ id: 'G', name: 'G', duration: 0, level: 0, progress: 0, askFinish: 2 }])
    expect(taskWarn(p.tasks[2], cpmOf(p), null, 'plan-ask')).toBeNull()
  })

  it('CPM 缺失/未通过 → none + note（有要求时间但不可比）', () => {
    const p = proj([{ id: 'B2', name: 'B2', duration: 4, level: 1, progress: 0, askFinish: 2 }])
    const w = taskWarn(p.tasks[2], null, null, 'plan-ask')!
    expect(w.level).toBe('none')
    expect(w.note).toBeTruthy()
  })

  it('predict-ask 未开始：无证据偏移，预测=计划完成（与 plan-ask 同判）', () => {
    const p = proj([{ id: 'B2', name: 'B2', duration: 4, level: 1, progress: 0, askFinish: 2 }])
    const c = cpmOf(p)
    const wp = taskWarn(p.tasks[2], c, null, 'predict-ask')!
    const wl = taskWarn(p.tasks[2], c, null, 'plan-ask')!
    expect(wp.level).toBe('red')
    expect(wp.compareFinish).toBe(wl.compareFinish)
  })

  it('predict-ask 按期推进（均匀速率）→ 预测=计划完成：es=0,d=10,at=4,pct=50 → predict=ef=9', () => {
    const p: SchedProject = {
      name: 'P', startDate: '2026-09-07',
      tasks: [{ id: 'T', name: 'T', duration: 10, level: 1, progress: 50, askFinish: 9 }],
      links: [],
    }
    const review = { at: 4, date: '2026-09-11', actuals: { T: { percent: 50 } } }
    const w = taskWarn(p.tasks[0], cpmOf(p), review, 'predict-ask')!
    expect(w.compareFinish).toBe(9)
    expect(w.level).toBe('green')
  })

  it('predict-ask 慢于计划 → 预测后延、红灯档位变化：pct=20 → predict=24', () => {
    const p: SchedProject = {
      name: 'P', startDate: '2026-09-07',
      tasks: [{ id: 'T', name: 'T', duration: 10, level: 1, progress: 20, askFinish: 10 }],
      links: [],
    }
    const review = { at: 4, date: '2026-09-11', actuals: { T: { percent: 20 } } }
    const w = taskWarn(p.tasks[0], cpmOf(p), review, 'predict-ask')!
    expect(w.compareFinish).toBe(24) // rate=2/5，剩余 8 → 4+20
    expect(w.level).toBe('red')
    expect(w.overdue).toBe(14)
    expect(w.band).toBe('0-15天')
  })

  it('predict-ask 快于计划 → 预测提前：pct=80 → predict=6，绿灯', () => {
    const p: SchedProject = {
      name: 'P', startDate: '2026-09-07',
      tasks: [{ id: 'T', name: 'T', duration: 10, level: 1, progress: 80, askFinish: 10 }],
      links: [],
    }
    const review = { at: 4, date: '2026-09-11', actuals: { T: { percent: 80 } } }
    const w = taskWarn(p.tasks[0], cpmOf(p), review, 'predict-ask')!
    expect(w.compareFinish).toBe(6)
    expect(w.overdue).toBe(-4)
    expect(w.level).toBe('green')
  })

  it('predict-ask 已完成：回填 finish 优先，缺省回落检视日', () => {
    const p: SchedProject = {
      name: 'P', startDate: '2026-09-07',
      tasks: [{ id: 'T', name: 'T', duration: 4, level: 1, progress: 100, askFinish: 5 }],
      links: [],
    }
    const c = cpmOf(p)
    const w1 = taskWarn(p.tasks[0], c, { at: 4, date: '2026-09-11', actuals: { T: { percent: 100, finish: 6 } } }, 'predict-ask')!
    expect(w1.compareFinish).toBe(6)
    expect(w1.overdue).toBe(1)
    const w2 = taskWarn(p.tasks[0], c, { at: 4, date: '2026-09-11', actuals: { T: { percent: 100 } } }, 'predict-ask')!
    expect(w2.compareFinish).toBe(4)
    expect(w2.level).toBe('green')
  })

  it('predict-ask 回填 start 参与 elapsed（elapsed 从实际开工起算）', () => {
    const p: SchedProject = {
      name: 'P', startDate: '2026-09-07',
      tasks: [{ id: 'T', name: 'T', duration: 10, level: 1, progress: 40, askFinish: 12 }],
      links: [],
    }
    // 实际第 3 工作日才开工：elapsed=at-start+1=3，rate=4/3，剩余 6 → predict=5+ceil(4.5)=10
    const review = { at: 5, date: '2026-09-12', actuals: { T: { percent: 40, start: 3 } } }
    const w = taskWarn(p.tasks[0], cpmOf(p), review, 'predict-ask')!
    expect(w.compareFinish).toBe(10)
    expect(w.level).toBe('green')
  })
})

describe('warnSummary / sortWarns（计数 + 从滞后到提前排序）', () => {
  it('计数：red/green/unassigned 三类齐且互斥；items 按 overdue 降序', () => {
    const p: SchedProject = {
      name: 'P', startDate: '2026-09-07',
      tasks: [
        { id: 'A', name: 'A', duration: 4, level: 1, progress: 0, askFinish: 0 }, // ef=3 → red 3
        { id: 'B', name: 'B', duration: 4, level: 1, progress: 0, askFinish: 9 }, // ef=8 → green -1
        { id: 'C', name: 'C', duration: 4, level: 1, progress: 0 }, // 未设置
        { id: 'G', name: 'G', duration: 0, level: 0, progress: 0 }, // 分组行不计
      ],
      links: [],
    }
    const s = warnSummary(p, cpmOf(p), null, 'plan-ask')
    expect(s.red).toBe(1)
    expect(s.green).toBe(1)
    expect(s.unassigned).toBe(1)
    expect(s.items).toHaveLength(3)
    expect(s.items[0].level).toBe('red')
    expect(s.items.map((w) => w.taskId)).toEqual(['A', 'B', 'C']) // red 在前，none 殿后
  })

  it('sortWarns 支持反向（从提前到滞后）且不改入参', () => {
    const items: TaskWarn[] = [
      { taskId: 'a', name: 'a', isMilestone: false, level: 'red', overdue: 5, band: '0-15天', compareFinish: 5, askFinish: 0, algorithm: 'plan-ask' },
      { taskId: 'b', name: 'b', isMilestone: false, level: 'red', overdue: 40, band: '30-60天', compareFinish: 40, askFinish: 0, algorithm: 'plan-ask' },
    ]
    const desc = sortWarns(items, 'desc')
    expect(desc.map((w) => w.taskId)).toEqual(['b', 'a'])
    expect(sortWarns(items, 'asc').map((w) => w.taskId)).toEqual(['a', 'b'])
    expect(items[0].taskId).toBe('a') // 入参不变
  })
})

describe('要求完成日期换算（工作日序号 ↔ ISO，走 calendar 唯一口径）', () => {
  it('askFinishOfDate / askFinishDateOf 互逆：2026-09-11 = 第 4 工作日', () => {
    const p = proj([])
    const wd = askFinishOfDate(p, '2026-09-11')
    expect(wd).toBe(4)
    expect(askFinishDateOf(p, wd)).toBe('2026-09-11')
  })
  it('早于开工日的日期回落 0（开工日）', () => {
    const p = proj([])
    expect(askFinishOfDate(p, '2026-09-01')).toBe(0)
    expect(askFinishDateOf(p, 0)).toBe('2026-09-07')
  })
})
