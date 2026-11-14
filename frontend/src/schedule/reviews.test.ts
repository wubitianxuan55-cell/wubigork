/**
 * schedule/reviews.test.ts — 检视模型用例（斑马复刻 S2）
 *
 * 三条口径必须钉死：① 完成量按**有效工期加权**（不是算术平均）；
 * ② 拉直**只升不降**且冻结后不可重复；③ 期际对比只列真正变化的行。
 */

import { describe, expect, it } from 'vitest'
import {
  canAddReview,
  compareReviews,
  defaultReviewDate,
  latestReview,
  makeReview,
  projectPercent,
  reviewAtOrBefore,
  reviewPercent,
  reviewProgress,
  reviewWorkday,
  sortedReviews,
  straighten,
  taskStatus,
} from './reviews'
import type { CpmResult, SchedProject, SchedReview, SchedTask } from './types'

const t = (id: string, extra: Partial<SchedTask> = {}): SchedTask => ({
  id, name: id, duration: 3, level: 1, progress: 0, ...extra,
})

const mk = (tasks: SchedTask[], reviews?: SchedReview[]): SchedProject => ({
  name: 'P', startDate: '2026-09-07', tasks, links: [], reviews,
})

describe('完成量与执行状态', () => {
  it('reviewPercent：该期回填优先，缺省回落任务 progress', () => {
    const task = t('A', { progress: 40 })
    expect(reviewPercent({ at: 1, date: '2026-09-08', actuals: { A: { percent: 70 } } }, task)).toBe(70)
    expect(reviewPercent({ at: 1, date: '2026-09-08', actuals: {} }, task)).toBe(40)
    expect(reviewPercent(null, task)).toBe(40)
  })

  it('taskStatus 三态：finish/100→已完成，start或>0→进行中，否则未开始', () => {
    const task = t('A', { progress: 0 })
    expect(taskStatus(task, { at: 1, date: '2026-09-08', actuals: { A: { finish: 3, percent: 100 } } })).toBe('已完成')
    expect(taskStatus(task, { at: 1, date: '2026-09-08', actuals: { A: { percent: 100 } } })).toBe('已完成')
    expect(taskStatus(task, { at: 1, date: '2026-09-08', actuals: { A: { start: 0, percent: 30 } } })).toBe('进行中')
    expect(taskStatus(task, { at: 1, date: '2026-09-08', actuals: {} })).toBe('未开始')
  })

  it('projectPercent 按工期加权（不是算术平均）', () => {
    // A 工期 1 完成 100%，B 工期 9 完成 0 → 加权 10%，算术平均会是 50%
    const p = mk([t('A', { duration: 1 }), t('B', { duration: 9 })])
    const r: SchedReview = { at: 2, date: '2026-09-09', actuals: { A: { percent: 100 } } }
    expect(projectPercent(p, r)).toBe(10)
  })

  it('里程碑不进分母；全是里程碑时退化为达成率', () => {
    const p = mk([
      t('M1', { isMilestone: true, duration: 0 }),
      t('M2', { isMilestone: true, duration: 0 }),
    ])
    const r: SchedReview = { at: 1, date: '2026-09-08', actuals: { M1: { percent: 100 } } }
    expect(projectPercent(p, r)).toBe(50)
  })
})

describe('期序列', () => {
  it('sortedReviews 升序、latestReview 取最新、reviewAtOrBefore 取 ≤at 最近一期', () => {
    const p = mk([t('A')], [
      { at: 5, date: '2026-09-14', actuals: {} },
      { at: 1, date: '2026-09-08', actuals: {} },
      { at: 3, date: '2026-09-10', actuals: {} },
    ])
    expect(sortedReviews(p).map((r) => r.at)).toEqual([1, 3, 5])
    expect(latestReview(p)?.at).toBe(5)
    expect(reviewAtOrBefore(p, 4)?.at).toBe(3)
    expect(reviewAtOrBefore(p, 0)).toBeNull()
  })

  it('reviewWorkday：早于开工日夹到 0；defaultReviewDate 不早于开工日', () => {
    const p = mk([t('A')])
    expect(reviewWorkday(p, '2026-08-01')).toBe(0)
    expect(reviewWorkday(p, '2026-09-07')).toBe(0) // 周一=第 0 工作日
    expect(reviewWorkday(p, '2026-09-08')).toBe(1)
    const early = defaultReviewDate(p, new Date('2026-08-01T00:00:00'))
    expect(early).toBe('2026-09-07')
  })

  it('makeReview：只记 progress>0 的任务；100% 顺带填 finish=at', () => {
    const p = mk([t('A', { progress: 0 }), t('B', { progress: 100 }), t('C', { progress: 40 })])
    const r = makeReview(p, 4, '2026-09-11')
    expect(Object.keys(r.actuals).sort()).toEqual(['B', 'C'])
    expect(r.actuals.B).toEqual({ percent: 100, finish: 4 })
    expect(r.actuals.C).toEqual({ percent: 40 })
  })

  it('canAddReview：重复 / 超上限 / 早于已冻结期 三种拒绝', () => {
    const many = Array.from({ length: 30 }, (_, i) => ({ at: i, date: '2026-09-07', actuals: {} }))
    expect(canAddReview(mk([t('A')], [{ at: 2, date: '2026-09-09', actuals: {} }]), 2).reason)
      .toContain('已有一期检视')
    expect(canAddReview(mk([t('A')], many), 99).reason).toContain('上限')
    expect(canAddReview(mk([t('A')], [{ at: 5, date: '2026-09-14', actuals: {}, frozen: true }]), 3).reason)
      .toContain('已冻结')
    expect(canAddReview(mk([t('A')], []), 1).ok).toBe(true)
  })
})

describe('拉直（写回 + 冻结）', () => {
  it('把回填写回 progress，并冻结该期；只升不降', () => {
    const p = mk(
      [t('A', { progress: 10 }), t('B', { progress: 90 })],
      [{ at: 3, date: '2026-09-10', actuals: { A: { percent: 60 }, B: { percent: 20 } } }],
    )
    const r = straighten(p, 3)
    expect(r.frozen).toBe(false)
    expect(r.changed).toBe(1) // 只有 A 提升（B 的 90 不被 20 拉低）
    const tasks = new Map(r.project.tasks.map((x) => [x.id, x]))
    expect(tasks.get('A')!.progress).toBe(60)
    expect(tasks.get('B')!.progress).toBe(90)
    expect(r.project.reviews![0].frozen).toBe(true)
    // 不改变入参（纯函数）
    expect(p.tasks[0].progress).toBe(10)
  })

  it('已冻结的期再拉直 → frozen=true、零改动', () => {
    const p = mk([t('A', { progress: 0 })], [{ at: 1, date: '2026-09-08', actuals: { A: { percent: 50 } }, frozen: true }])
    const r = straighten(p, 1)
    expect(r).toMatchObject({ changed: 0, frozen: true })
    expect(r.project).toBe(p)
  })

  it('不存在的期 → 零改动不冻结', () => {
    const p = mk([t('A')], [])
    expect(straighten(p, 9)).toMatchObject({ changed: 0, frozen: false })
  })
})

describe('期际对比与当期进度', () => {
  it('compareReviews 只列变化的行，并给整体完成量差', () => {
    const p = mk([t('A', { duration: 2 }), t('B', { duration: 2 })], [
      { at: 1, date: '2026-09-08', actuals: { A: { percent: 50 }, B: { percent: 0 } } },
      { at: 3, date: '2026-09-10', actuals: { A: { percent: 100 }, B: { percent: 0 } } },
    ])
    const [from, to] = sortedReviews(p)
    const d = compareReviews(p, from, to)
    expect(d.rows.map((x) => x.id)).toEqual(['A'])
    expect(d.rows[0]).toMatchObject({ fromPct: 50, toPct: 100, delta: 50, status: '已完成' })
    expect(d.fromAt).toBe(1)
    expect(d.toAt).toBe(3)
    expect(d.deltaPercent).toBe(25) // A 从 50%→100%（各占一半工期）→ 整体 +25
  })

  it('from 为空时以"任务当前进度"为起点', () => {
    const p = mk([t('A', { progress: 20 })], [{ at: 2, date: '2026-09-09', actuals: { A: { percent: 80 } } }])
    const d = compareReviews(p, null, sortedReviews(p)[0])
    expect(d.fromAt).toBeNull()
    expect(d.rows[0]).toMatchObject({ fromPct: 20, toPct: 80, delta: 60 })
  })

  it('reviewProgress：计划应完成 vs 实际完成（按 CPM 行算到当期）', () => {
    const p = mk([t('A', { duration: 4 }), t('B', { duration: 4 })])
    const cpm: CpmResult = {
      ok: true,
      duration: 8,
      rows: {
        A: { es: 0, ef: 3, ls: 0, lf: 3, tf: 0, ff: 0, critical: true },
        B: { es: 4, ef: 7, ls: 4, lf: 7, tf: 0, ff: 0, critical: true },
      },
    }
    // 第 4 工作日：A 已完成（4/4），B 刚开工（1/4）
    const r: SchedReview = { at: 4, date: '2026-09-11', actuals: { A: { percent: 100 }, B: { percent: 50 } } }
    const prog = reviewProgress(p, r, cpm)
    expect(prog.totalWorkdays).toBe(8)
    expect(prog.plannedWorkdays).toBe(5) // A 4 + B 1
    expect(prog.actualWorkdays).toBe(6) // A 4 + B 2
    expect(prog.plannedPercent).toBe(63)
    expect(prog.actualPercent).toBe(75)
    expect(prog.gapPercent).toBe(12)
  })
})
