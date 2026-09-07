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
