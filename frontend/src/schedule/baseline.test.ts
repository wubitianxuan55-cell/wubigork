/**
 * baseline.test.ts — 基线快照与漂移对比纯函数用例（v4.116.0 刀7）
 *
 * 与 internal/schedule/baseline_test.go 互为镜像：同一批场景同一批期望值，
 * 两侧引擎口径必须逐字段一致。
 */
import { describe, expect, it } from 'vitest'
import { computeBaselineDrift, snapshotBaseline } from './baseline'
import { computeCpm } from './cpm'
import type { SchedProject } from './types'

/** 三任务串联样板：A(3) → B(2) → C(4)，总工期 9，全关键 */
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

describe('snapshotBaseline', () => {
  it('固化叶任务 ES/EF/工期/关键标记，里程碑工期记 0', () => {
    const p = chainProject()
    p.tasks.push({ id: 'M', name: '验收', duration: 1, level: 1, progress: 0, isMilestone: true })
    p.links.push({ from: 'C', to: 'M', type: 'FS', lag: 0 })
    const cpm = computeCpm(p.tasks, p.links)
    const r = snapshotBaseline(p, cpm, '2026-09-06 10:00')
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(r.baseline.name).toBe('基线')
    expect(r.baseline.savedAt).toBe('2026-09-06 10:00')
    expect(r.baseline.duration).toBe(9)
    expect(r.baseline.rows.A).toEqual({ name: '挖土', es: 0, ef: 3, dur: 3, critical: true })
    expect(r.baseline.rows.C).toEqual({ name: '浇筑', es: 5, ef: 9, dur: 4, critical: true })
    expect(r.baseline.rows.M).toEqual({ name: '验收', es: 9, ef: 9, dur: 0, critical: true })
  })

  it('自定义基线名去除首尾空白，空白回落「基线」', () => {
    const p = chainProject()
    const cpm = computeCpm(p.tasks, p.links)
    const r1 = snapshotBaseline(p, cpm, '2026-09-06 10:00', '  开工版  ')
    const r2 = snapshotBaseline(p, cpm, '2026-09-06 10:00', '   ')
    expect(r1.ok && r1.baseline.name).toBe('开工版')
    expect(r2.ok && r2.baseline.name).toBe('基线')
  })

  it('循环依赖 fail-closed 拒绝保存', () => {
    const p = chainProject()
    p.links.push({ from: 'C', to: 'A', type: 'FS', lag: 0 })
    const cpm = computeCpm(p.tasks, p.links)
    const r = snapshotBaseline(p, cpm, '2026-09-06 10:00')
    expect(r.ok).toBe(false)
    if (r.ok) return
    expect(r.error).toContain('循环依赖')
  })

  it('空计划（无叶任务）拒绝保存', () => {
    const p: SchedProject = { name: '空', startDate: '2026-09-07', tasks: [], links: [] }
    const cpm = computeCpm(p.tasks, p.links)
    const r = snapshotBaseline(p, cpm, '2026-09-06 10:00')
    expect(r.ok).toBe(false)
    if (r.ok) return
    expect(r.error).toContain('没有叶任务')
  })
})

describe('computeBaselineDrift', () => {
  it('无基线返回 null', () => {
    const p = chainProject()
    expect(computeBaselineDrift(p, computeCpm(p.tasks, p.links))).toBeNull()
  })

  it('计划与基线完全一致：漂移清零、rows 为空', () => {
    const p = chainProject()
    const cpm = computeCpm(p.tasks, p.links)
    const r = snapshotBaseline(p, cpm, '2026-09-06 10:00')
    if (!r.ok) throw new Error('snapshot failed')
    p.baseline = r.baseline
    const d = computeBaselineDrift(p, computeCpm(p.tasks, p.links))
    expect(d).not.toBeNull()
    expect(d!.durationDrift).toBe(0)
    expect(d!.sameCount).toBe(3)
    expect(d!.shiftedCount).toBe(0)
    expect(d!.rows).toEqual([])
    expect(d!.criticalGained).toEqual([])
    expect(d!.criticalLost).toEqual([])
  })

  it('延长关键任务：全链推移、总工期拖后', () => {
    const p = chainProject()
    const cpm = computeCpm(p.tasks, p.links)
    const r = snapshotBaseline(p, cpm, '2026-09-06 10:00')
    if (!r.ok) throw new Error('snapshot failed')
    p.baseline = r.baseline
    p.tasks = p.tasks.map((t) => (t.id === 'A' ? { ...t, duration: 5 } : t))
    const d = computeBaselineDrift(p, computeCpm(p.tasks, p.links))!
    expect(d.baselineDuration).toBe(9)
    expect(d.currentDuration).toBe(11)
    expect(d.durationDrift).toBe(2)
    expect(d.shiftedCount).toBe(3)
    expect(d.addedCount).toBe(0)
    expect(d.removedCount).toBe(0)
    const a = d.rows.find((x) => x.id === 'A')!
    expect(a.kind).toBe('shifted')
    expect(a.esDrift).toBe(0)
    expect(a.efDrift).toBe(2)
    expect(a.durDrift).toBe(2)
    expect(d.rows.find((x) => x.id === 'C')!.esDrift).toBe(2)
  })

  it('压缩关键链使非关键变关键：criticalGained 记录新关键', () => {
    // A(4)→C(4) 关键链总工期 8，B(4) 与 C 并行（SS0）：基线里 B 有 4 天
    // 总时差、非关键。压 A、C 到 2 天后总工期 4，B 追平总工期变关键
    // （es/ef/工期全没变，仅关键标记翻转——单标记变化也计为推移）。
    const p: SchedProject = {
      name: '并行样板',
      startDate: '2026-09-07',
      tasks: [
        { id: 'A', name: '挖土', duration: 4, level: 1, progress: 0 },
        { id: 'B', name: '预埋', duration: 4, level: 1, progress: 0 },
        { id: 'C', name: '浇筑', duration: 4, level: 1, progress: 0 },
      ],
      links: [
        { from: 'A', to: 'C', type: 'FS', lag: 0 },
        { from: 'A', to: 'B', type: 'SS', lag: 0 },
      ],
    }
    const cpm = computeCpm(p.tasks, p.links)
    expect(cpm.duration).toBe(8)
    expect(cpm.rows.B.critical).toBe(false)
    const r = snapshotBaseline(p, cpm, '2026-09-06 10:00')
    if (!r.ok) throw new Error('snapshot failed')
    p.baseline = r.baseline
    p.tasks = p.tasks.map((t) => (t.id === 'A' || t.id === 'C' ? { ...t, duration: 2 } : t))
    const d = computeBaselineDrift(p, computeCpm(p.tasks, p.links))!
    expect(d.durationDrift).toBe(-4)
    expect(d.criticalGained).toEqual(['预埋'])
    const b = d.rows.find((x) => x.id === 'B')!
    expect(b.kind).toBe('shifted')
    expect(b.esDrift).toBe(0)
    expect(b.efDrift).toBe(0)
    expect(b.durDrift).toBe(0)
    expect(b.criticalNow).toBe(true)
    expect(b.criticalBase).toBe(false)
  })

  it('新增任务计入 added，行内无基线值', () => {
    const p = chainProject()
    const cpm = computeCpm(p.tasks, p.links)
    const r = snapshotBaseline(p, cpm, '2026-09-06 10:00')
    if (!r.ok) throw new Error('snapshot failed')
    p.baseline = r.baseline
    p.tasks.push({ id: 'D', name: '养护', duration: 2, level: 1, progress: 0 })
    p.links.push({ from: 'C', to: 'D', type: 'FS', lag: 0 })
    const d = computeBaselineDrift(p, computeCpm(p.tasks, p.links))!
    expect(d.addedCount).toBe(1)
    expect(d.durationDrift).toBe(2)
    const row = d.rows.find((x) => x.id === 'D')!
    expect(row.kind).toBe('added')
    expect(row.base).toBeNull()
    expect(row.esDrift).toBe(9)
    expect(row.now!.es).toBe(9)
    expect(row.now!.ef).toBe(11)
  })

  it('移除任务计入 removed（按 id 排序），名字取自基线行', () => {
    const p = chainProject()
    const cpm = computeCpm(p.tasks, p.links)
    const r = snapshotBaseline(p, cpm, '2026-09-06 10:00')
    if (!r.ok) throw new Error('snapshot failed')
    p.baseline = r.baseline
    p.tasks = p.tasks.filter((t) => t.id !== 'B')
    p.links = p.links.map((l) => (l.from === 'B' ? { ...l, from: 'A' } : l)).filter((l) => l.to !== 'B')
    const d = computeBaselineDrift(p, computeCpm(p.tasks, p.links))!
    expect(d.removedCount).toBe(1)
    expect(d.durationDrift).toBe(-2)
    const row = d.rows.find((x) => x.kind === 'removed')!
    expect(row.id).toBe('B')
    expect(row.name).toBe('垫层')
    expect(row.base!.ef).toBe(5)
    expect(row.now).toBeNull()
  })

  it('rows 只含有偏差行：一致任务不进列表', () => {
    const p = chainProject()
    const cpm = computeCpm(p.tasks, p.links)
    const r = snapshotBaseline(p, cpm, '2026-09-06 10:00')
    if (!r.ok) throw new Error('snapshot failed')
    p.baseline = r.baseline
    // 只改 C 工期：A、B 的排程不变（C 变化不回传），只有 C 进 rows
    p.tasks = p.tasks.map((t) => (t.id === 'C' ? { ...t, duration: 6 } : t))
    const d = computeBaselineDrift(p, computeCpm(p.tasks, p.links))!
    expect(d.rows.map((x) => x.id)).toEqual(['C'])
    expect(d.sameCount).toBe(2)
  })
})
