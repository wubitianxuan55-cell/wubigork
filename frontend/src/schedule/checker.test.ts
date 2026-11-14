/**
 * schedule/checker.test.ts — 计划检查器规则用例（每条规则至少一正一负）
 *
 * 纪律：断言 code 而不只断言数量——加规则时不能让老用例静默变形。
 */

import { describe, expect, it } from 'vitest'
import { checkProject, diagSummary, sortDiagnostics, HIGH_FLOAT_DAYS } from './checker'
import type { CpmResult, SchedProject, SchedTask } from './types'

const t = (id: string, extra: Partial<SchedTask> = {}): SchedTask => ({
  id, name: id, duration: 3, level: 1, progress: 0, ...extra,
})
const g = (id: string): SchedTask => ({ id, name: id, duration: 0, level: 0, progress: 0 })

const project = (tasks: SchedTask[], links: SchedProject['links'] = [], extra: Partial<SchedProject> = {}): SchedProject => ({
  name: 'P', startDate: '2026-09-01', tasks, links, ...extra,
})

const codes = (p: SchedProject, cpm?: CpmResult | null) => checkProject(p, cpm).map((d) => d.code)

/** 干净计划：一条 A→B 链，无 error 无 warn（no-milestone 是 info） */
const clean = () => project([t('A'), t('B')], [{ from: 'A', to: 'B', type: 'FS', lag: 0 }])

describe('checkProject：结构闸门（error）', () => {
  it('干净计划无 error / warn', () => {
    const s = diagSummary(checkProject(clean()))
    expect(s.error).toBe(0)
    expect(s.warn).toBe(0)
  })

  it('cycle：CPM 失败时把错误原样上抛为 error', () => {
    const cpm: CpmResult = { ok: false, error: '存在循环依赖：A → B → A', rows: {}, duration: 0 }
    const ds = checkProject(clean(), cpm)
    expect(ds[0].code).toBe('cycle')
    expect(ds[0].severity).toBe('error')
    expect(ds[0].message).toContain('循环依赖')
  })

  it('link-dangling / link-self / link-bad-type', () => {
    expect(codes(project([t('A'), t('B')], [{ from: 'A', to: 'ghost', type: 'FS', lag: 0 }])))
      .toContain('link-dangling')
    expect(codes(project([t('A')], [{ from: 'A', to: 'A', type: 'FS', lag: 0 }])))
      .toContain('link-self')
    expect(codes(project([t('A'), t('B')], [{ from: 'A', to: 'B', type: 'XX' as never, lag: 0 }])))
      .toContain('link-bad-type')
  })

  it('group-link：分组行参与搭接（斑马「分区上存在关系」）', () => {
    const p = project([g('G'), t('A')], [{ from: 'G', to: 'A', type: 'FS', lag: 0 }])
    const d = checkProject(p).find((x) => x.code === 'group-link')
    expect(d?.severity).toBe('error')
    expect(d?.taskId).toBe('G')
  })

  it('group-assignment / milestone-duration / duration-negative / duration-cd-over', () => {
    const p = project(
      [g('G'), t('A', { isMilestone: true, duration: 2 }), t('B', { duration: -1 }), t('C', { durationUnit: 'cd', duration: 4000 })],
      [],
      { assignments: [{ taskId: 'G', resourceId: 'r1' }], resources: [{ id: 'r1', name: 'R', type: 'work' }] },
    )
    const cs = codes(p)
    expect(cs).toContain('group-assignment')
    expect(cs).toContain('milestone-duration')
    expect(cs).toContain('duration-negative')
    expect(cs).toContain('duration-cd-over')
  })
})

describe('checkProject：语义发现（warn / info）', () => {
  it('isolated-task：完全无搭接的叶任务', () => {
    const p = project([t('A'), t('B'), t('C')], [{ from: 'A', to: 'B', type: 'FS', lag: 0 }])
    expect(codes(p)).toContain('isolated-task')
  })

  it('no-predecessor：非首个叶任务无入边（首个起点不算）', () => {
    const p = project([t('A'), t('B'), t('C')], [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'C', to: 'B', type: 'FS', lag: 0 },
    ])
    // A 是首个非手动叶任务（计划起点），天然无前置不计 —— 与 Go schedule_analyze 同口径
    const ds = checkProject(p).filter((d) => d.code === 'no-predecessor')
    expect(ds.map((d) => d.taskId)).toEqual(['C'])
  })

  it('open-end：非末位叶任务无后继', () => {
    const p = project([t('A'), t('B'), t('C')], [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'C', to: 'B', type: 'FS', lag: 0 },
    ])
    // C 是末位叶任务（天然收口点）不计；B 被 A、C 汇入后再无后继 → 报
    const ds = checkProject(p).filter((d) => d.code === 'open-end')
    expect(ds.map((d) => d.taskId)).toEqual(['B'])
  })

  it('empty-group：分组后面没有子任务', () => {
    const p = project([g('G1'), t('A'), g('G2')], [{ from: 'A', to: 'A', type: 'FS', lag: 0 }])
    const ds = checkProject(p).filter((d) => d.code === 'empty-group')
    expect(ds.map((d) => d.taskId)).toEqual(['G2'])
  })

  it('no-milestone / no-links 是提示项', () => {
    const cs = codes(project([t('A'), t('B')], []))
    expect(cs).toContain('no-milestone')
    expect(cs).toContain('no-links')
    // 有里程碑则不报 no-milestone
    expect(codes(project([t('A'), t('B', { isMilestone: true, duration: 0 })], [{ from: 'A', to: 'B', type: 'FS', lag: 0 }])))
      .not.toContain('no-milestone')
  })

  it('unpriced-assignment：工时资源未设费率', () => {
    const p = project([t('A'), t('B')], [{ from: 'A', to: 'B', type: 'FS', lag: 0 }], {
      resources: [{ id: 'r1', name: '瓦工', type: 'work' }],
      assignments: [{ taskId: 'A', resourceId: 'r1' }],
    })
    const d = checkProject(p).find((x) => x.code === 'unpriced-assignment')
    expect(d?.severity).toBe('info')
    expect(d?.taskId).toBe('A')
  })

  it('negative-lag：负时距给出 warn（合法但要确认）', () => {
    const p = project([t('A'), t('B')], [{ from: 'A', to: 'B', type: 'FS', lag: -2 }])
    const d = checkProject(p).find((x) => x.code === 'negative-lag')
    expect(d?.severity).toBe('warn')
    expect(d?.linkIndex).toBe(0)
  })

  it('redundant-link：A→B→C 且 A→C（同类同时距）判定为多余关系', () => {
    const p = project([t('A'), t('B'), t('C')], [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'B', to: 'C', type: 'FS', lag: 0 },
      { from: 'A', to: 'C', type: 'FS', lag: 0 },
    ])
    const d = checkProject(p).find((x) => x.code === 'redundant-link')
    expect(d?.linkIndex).toBe(2)
    // 时距不同则不算冗余
    const p2 = project([t('A'), t('B'), t('C')], [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'B', to: 'C', type: 'FS', lag: 0 },
      { from: 'A', to: 'C', type: 'FS', lag: 2 },
    ])
    expect(codes(p2)).not.toContain('redundant-link')
  })
})

describe('checkProject：排程相关提示（需 cpm）', () => {
  const cpmOf = (tf: Record<string, number>): CpmResult => ({
    ok: true,
    duration: 30,
    rows: Object.fromEntries(
      Object.entries(tf).map(([id, v]) => [id, { es: 0, ef: 3, ls: v, lf: 3 + v, tf: v, ff: v, critical: v === 0 }]),
    ),
  })

  it('high-float：总时差超阈值', () => {
    const p = clean()
    expect(codes(p, cpmOf({ A: 0, B: HIGH_FLOAT_DAYS + 1 }))).toContain('high-float')
  })

  it('near-critical：0 < tf ≤ 2', () => {
    expect(codes(clean(), cpmOf({ A: 0, B: 2 }))).toContain('near-critical')
  })

  it('无 cpm 时不跑排程类规则（结构检查可独立使用）', () => {
    const cs = codes(clean(), null)
    expect(cs).not.toContain('high-float')
    expect(cs).not.toContain('near-critical')
  })
})

describe('排序与汇总', () => {
  it('sortDiagnostics：error → warn → info，同级按任务表序', () => {
    const p = project([t('A'), t('B'), t('C')], [
      { from: 'A', to: 'B', type: 'FS', lag: -1 }, // negative-lag(warn, taskId=B)
      { from: 'B', to: 'ghost', type: 'FS', lag: 0 }, // dangling(error)
    ])
    const sorted = sortDiagnostics(checkProject(p), p.tasks)
    const sev = sorted.map((d) => d.severity)
    expect(sev.indexOf('error')).toBeLessThan(sev.lastIndexOf('warn') + 1)
    expect(sorted[0].severity).toBe('error')
  })

  it('diagSummary 分别计数', () => {
    const s = diagSummary([
      { code: 'a', severity: 'error', message: '' },
      { code: 'b', severity: 'warn', message: '' },
      { code: 'c', severity: 'warn', message: '' },
    ])
    expect(s).toEqual({ error: 1, warn: 2, info: 0 })
  })

  it('坏形输入不抛异常（板块必须能打开）', () => {
    const bad = { name: 'x', startDate: '2026-09-01', tasks: null, links: undefined } as unknown as SchedProject
    expect(() => checkProject(bad)).not.toThrow()
    expect(checkProject(bad)).toEqual([])
  })
})

