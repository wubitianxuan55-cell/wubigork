import { describe, expect, it } from 'vitest'
import { simulateOps } from './opsSim'
import fixtureSource from './ops_golden.fixture.json'
import type { SchedProject } from './types'

// opsSim.golden.test.ts — Go/TS 对拍主阵地（diff 确认闭环刀D）。
//
// ops_golden.fixture.json 由 internal/schedule/ops_golden_test.go 生成并校验
// （Go ApplyOps 为口径权威）；这里读同一份文件，用 TS simulateOps 双跑每一
// 条用例，比对「是否失败 + after 状态」。任何一侧语义漂移（改了 Go 没同步
// TS、或反之）都会在这里显式失败——「批的是 A，落的是 B」风险的硬对冲。
//
// 比对口径：after 两侧各自过 canonicalStringify（键排序+丢 undefined/null/''，
// 消化 Go omitempty 与 TS undefined 的表达差）；错误只比「是否失败」（文案
// 是各自实现细节，不参与对拍）。

interface GoldenCase {
  name: string
  base: SchedProject
  ops: unknown[]
  error?: string
  after?: SchedProject | null
}

const fixture = fixtureSource as { cases: GoldenCase[] }

function canonical(v: unknown): unknown {
  if (v === undefined || v === null || v === '') return undefined
  if (Array.isArray(v)) {
    // 空数组双侧同丢：Go omitempty 省略空切片（resources/assignments），
    // links（无 omitempty）空时两侧都出 []——同规则丢掉仍等价。
    const arr = v.map(canonical).filter((x) => x !== undefined)
    return arr.length > 0 ? arr : undefined
  }
  if (typeof v === 'object') {
    const out: Record<string, unknown> = {}
    for (const k of Object.keys(v as Record<string, unknown>).sort()) {
      const c = canonical((v as Record<string, unknown>)[k])
      if (c !== undefined) out[k] = c
    }
    return out
  }
  return v
}

describe('opsSim Go/TS 对拍（golden fixture，刀D 漂移主阵地）', () => {
  it('fixture 非空且成功/失败两类用例兼备', () => {
    expect(fixture.cases.length).toBeGreaterThanOrEqual(40)
    expect(fixture.cases.some((c) => c.error)).toBe(true)
    expect(fixture.cases.some((c) => c.after)).toBe(true)
  })

  for (const tc of fixture.cases) {
    it(`对拍：${tc.name}`, () => {
      const r = simulateOps(tc.base, tc.ops as never)
      if (tc.error) {
        expect(r.ok).toBe(false) // 失败口径：单条失败即中止
        return
      }
      if (!r.ok) throw new Error(`TS 侧意外失败：${r.error}`)
      expect(canonical(r.project)).toEqual(canonical(tc.after))
    })
  }
})

// ── 批次十三 round15 闸宽边界（手工探针，非 fixture 驱动） ──
//
// patch_task 的负工期闸只在本 op **触及 duration** 时生效（与 Go 生效态块内的
// `pt.Duration != nil` 门一致）：计划里既有 Duration<0 的任务 + 只改 level 的
// 无关补丁**两侧都放行**——那不是本次改动造成的非法态，由 Go 侧 Validate/Save
// 兜底。fixture 里的同名用例只比「双端是否失败」，若两侧一起改坏仍会绿，故这里
// 显式钉住这条有意收窄的边界（改坏任一侧即红）。
describe('opsSim 负工期闸宽边界（手工探针）', () => {
  const brokenBase = () =>
    ({
      name: '既有负工期',
      startDate: '2026-09-07',
      tasks: [
        { id: 'G', name: '分组', duration: 0, level: 0, progress: 0 },
        { id: 'A', name: '挖土', duration: -5, level: 1, progress: 0 },
      ],
      links: [],
    }) as unknown as SchedProject

  it('既有负工期 + 只改 level：放行且改动落地（不被无关字段绊住）', () => {
    const r = simulateOps(brokenBase(), [{ type: 'patch_task', id: 'A', patch: { level: 0 } }])
    if (!r.ok) throw new Error(`不应被既有负工期绊住：${r.error}`)
    expect(r.project.tasks[1].level).toBe(0)
    expect(r.project.tasks[1].duration).toBe(-5)
  })

  it('同一探针触及 duration：仍按统一判据拒绝', () => {
    const r = simulateOps(brokenBase(), [{ type: 'patch_task', id: 'A', patch: { duration: -1 } }])
    expect(r.ok).toBe(false)
    if (!r.ok) expect(r.error).toContain('工期为负')
  })

  it('upsert 缺 duration（零值语义→0）与显式 0 均放行，负值拒绝', () => {
    const base = { name: 'p', startDate: '2026-09-07', tasks: [], links: [] } as unknown as SchedProject
    for (const task of [{ id: 'C', name: '缺省', level: 1 }, { id: 'C', name: '零', duration: 0, level: 1 }, { id: 'C', name: '正', duration: 5, level: 1 }]) {
      const r = simulateOps(base, [{ type: 'upsert_task', task }])
      if (!r.ok) throw new Error(`合法态被拒：${r.error}`)
      expect(r.project.tasks[0].duration).toBe(task.duration ?? 0)
    }
    const bad = simulateOps(base, [{ type: 'upsert_task', task: { id: 'C', name: '负', duration: -5, level: 1 } }])
    expect(bad.ok).toBe(false)
    if (!bad.ok) expect(bad.error).toContain('任务 C 工期为负')
  })
})
