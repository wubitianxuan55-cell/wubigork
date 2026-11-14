/**
 * colFilterPrefs.test.ts — 列筛本机偏好纯函数用例（W5 列筛持久化）
 *
 * 口径：`COL_FILTER_KIND` 是权威类型；归一=只收已知列键 + 逐类校验 + 空壳剔除；
 * 签名=稳定键序（去抖比较用）。全部纯函数，脏输入只退化不抛。
 */
import { describe, expect, it } from 'vitest'
import {
  COL_FILTER_ENUM_MAX,
  COL_FILTER_TEXT_MAX,
  colFilterSignature,
  normalizeColFilters,
  serializeColFilters,
} from './colFilterPrefs'
import type { ColFilters } from './ganttFilter'

describe('normalizeColFilters — text 列', () => {
  it('保留 trim 后的文本；空白文本=无有效条件（不留空壳）', () => {
    expect(normalizeColFilters({ name: { kind: 'text', text: '  土方  ' } })).toEqual({
      name: { kind: 'text', text: '土方' },
    })
    expect(normalizeColFilters({ name: { kind: 'text', text: '   ' } })).toEqual({})
    expect(normalizeColFilters({ name: { kind: 'text' } })).toEqual({})
  })

  it('超长文本截到上限，非字符串文本视为无', () => {
    const long = 'x'.repeat(COL_FILTER_TEXT_MAX + 20)
    const got = normalizeColFilters({ text1: { kind: 'text', text: long } })
    expect(got.text1?.text).toHaveLength(COL_FILTER_TEXT_MAX)
    expect(normalizeColFilters({ text1: { kind: 'text', text: 42 } })).toEqual({})
  })
})

describe('normalizeColFilters — number 列', () => {
  it('op ∈ {ge,le,eq} 原样保留，有限 num 保留', () => {
    const got = normalizeColFilters({
      dur: { kind: 'number', op: 'le', num: 5 },
      progress: { kind: 'number', op: 'eq', num: 0 },
      tf: { kind: 'number', op: 'ge', num: -2 },
    })
    expect(got).toEqual({
      dur: { kind: 'number', op: 'le', num: 5 },
      progress: { kind: 'number', op: 'eq', num: 0 },
      tf: { kind: 'number', op: 'ge', num: -2 },
    })
  })

  it('非法 op 回落 ge；非有限 num（NaN/Infinity/字符串）整条丢弃', () => {
    expect(normalizeColFilters({ dur: { kind: 'number', op: 'gt', num: 3 } })).toEqual({
      dur: { kind: 'number', op: 'ge', num: 3 },
    })
    expect(normalizeColFilters({ dur: { kind: 'number', num: NaN } })).toEqual({})
    expect(normalizeColFilters({ dur: { kind: 'number', num: Infinity } })).toEqual({})
    expect(normalizeColFilters({ dur: { kind: 'number', num: '3' } })).toEqual({})
    expect(normalizeColFilters({ dur: { kind: 'number', op: 'le' } })).toEqual({})
  })
})

describe('normalizeColFilters — date 列', () => {
  it('YYYY-MM-DD 双边保留（from/to 可单边）', () => {
    expect(normalizeColFilters({ start: { kind: 'date', from: '2026-01-01', to: '2026-03-31' } })).toEqual({
      start: { kind: 'date', from: '2026-01-01', to: '2026-03-31' },
    })
    expect(normalizeColFilters({ finish: { kind: 'date', to: '2026-03-31' } })).toEqual({
      finish: { kind: 'date', to: '2026-03-31' },
    })
  })

  it('非法日期单侧丢弃，双侧皆非法=整条丢弃', () => {
    const got = normalizeColFilters({ start: { kind: 'date', from: '2026/01/01', to: '2026-02-28' } })
    expect(got).toEqual({ start: { kind: 'date', to: '2026-02-28' } })
    expect(normalizeColFilters({ start: { kind: 'date', from: '2026-13-01' } })).toEqual({})
    expect(normalizeColFilters({ start: { kind: 'date', from: '2026-02-32' } })).toEqual({})
    expect(normalizeColFilters({ start: { kind: 'date', from: '', to: '' } })).toEqual({})
  })
})

describe('normalizeColFilters — enum 列', () => {
  it('去空、去重、保持首次出现顺序；非字符串项丢弃', () => {
    expect(
      normalizeColFilters({ mode: { kind: 'enum', set: ['manual', '  ', 'auto', 'manual', 7] } }),
    ).toEqual({ mode: { kind: 'enum', set: ['manual', 'auto'] } })
  })

  it('上限 20 项；空集/非数组=整条丢弃', () => {
    const many = Array.from({ length: COL_FILTER_ENUM_MAX + 5 }, (_, i) => `v${i}`)
    const got = normalizeColFilters({ marker: { kind: 'enum', set: many } })
    expect(got.marker?.set).toHaveLength(COL_FILTER_ENUM_MAX)
    expect(got.marker?.set?.[0]).toBe('v0')
    expect(normalizeColFilters({ marker: { kind: 'enum', set: [] } })).toEqual({})
    expect(normalizeColFilters({ marker: { kind: 'enum', set: 'milestone' } })).toEqual({})
  })
})

describe('normalizeColFilters — 空值口径与键校验', () => {
  it('empty ∈ {any,empty,nonEmpty}：any 等同缺省不写，其余保留', () => {
    expect(normalizeColFilters({ text2: { kind: 'text', empty: 'any' } })).toEqual({})
    expect(normalizeColFilters({ text2: { kind: 'text', empty: 'empty' } })).toEqual({
      text2: { kind: 'text', empty: 'empty' },
    })
    // 空值口径与类型条件是 AND：两者并存
    expect(normalizeColFilters({ num1: { kind: 'number', op: 'ge', num: 0, empty: 'nonEmpty' } })).toEqual({
      num1: { kind: 'number', op: 'ge', num: 0, empty: 'nonEmpty' },
    })
    expect(normalizeColFilters({ text2: { kind: 'text', empty: 'hollow' } })).toEqual({})
  })

  it('只认 COL_FILTER_KIND 已知列键：未知键/自定义字段/原型链键丢弃', () => {
    const got = normalizeColFilters({
      name: { kind: 'text', text: 'a' },
      ghost: { kind: 'text', text: 'b' },
      cost: { kind: 'number', op: 'ge', num: 1 }, // 自定义字段（表达式未建模）
    })
    expect(got).toEqual({ name: { kind: 'text', text: 'a' } })
    // localStorage 里的手改 JSON 可带自有 __proto__ / constructor 键：不得命中原型链
    const polluted = JSON.parse(
      '{"name":{"kind":"text","text":"a"},"__proto__":{"kind":"text","text":"p"},' +
        '"constructor":{"kind":"text","text":"c"}}',
    ) as unknown
    const got2 = normalizeColFilters(polluted)
    expect(Object.keys(got2)).toEqual(['name'])
    expect(got2).toEqual({ name: { kind: 'text', text: 'a' } })
  })

  it('kind 与登记类型冲突＝脏数据，整条丢弃（不猜不纠正）', () => {
    expect(normalizeColFilters({ name: { kind: 'number', num: 3 } })).toEqual({})
    expect(normalizeColFilters({ start: { kind: 'text', text: '2026' } })).toEqual({})
    // kind 缺失时按登记类型解析（存量手改数据可容忍）
    expect(normalizeColFilters({ name: { text: 'a' } })).toEqual({ name: { kind: 'text', text: 'a' } })
  })

  it('脏输入一律不抛：null/undefined/数组/标量/嵌套异常', () => {
    const junk: unknown[] = [null, undefined, 0, 42, '', 'nope', [], [1, 2], true, { name: null }]
    for (const raw of junk) expect(() => normalizeColFilters(raw)).not.toThrow()
    expect(normalizeColFilters(null)).toEqual({})
    expect(normalizeColFilters([{ name: { kind: 'text', text: 'a' } }])).toEqual({})
    expect(normalizeColFilters({ name: { kind: 'text', text: { deep: 1 } } })).toEqual({})
    expect(normalizeColFilters({ name: ['x'] })).toEqual({})
  })
})

describe('serializeColFilters', () => {
  it('剔除空条件，且对已归一数据幂等', () => {
    const dirty: ColFilters = {
      name: { kind: 'text', text: '   ' },
      dur: { kind: 'number', op: 'ge', num: 2 },
      start: { kind: 'date', from: '', to: '' },
      mode: { kind: 'enum', set: [] },
    }
    const once = serializeColFilters(dirty)
    expect(once).toEqual({ dur: { kind: 'number', op: 'ge', num: 2 } })
    expect(serializeColFilters(once)).toEqual(once)
  })

  it('空对象落盘仍是空对象（不产生垃圾键）', () => {
    expect(serializeColFilters({})).toEqual({})
    expect(JSON.stringify(serializeColFilters({}))).toBe('{}')
  })
})

describe('colFilterSignature', () => {
  it('键序不同、enum 集合序不同 → 签名相同；空对象=空串', () => {
    const a: ColFilters = {
      name: { kind: 'text', text: 'a' },
      dur: { kind: 'number', op: 'ge', num: 3 },
      mode: { kind: 'enum', set: ['auto', 'manual'] },
    }
    const b: ColFilters = {
      mode: { kind: 'enum', set: ['manual', 'auto'] },
      dur: { kind: 'number', op: 'ge', num: 3 },
      name: { kind: 'text', text: 'a' },
    }
    expect(colFilterSignature(a)).toBe(colFilterSignature(b))
    expect(colFilterSignature({})).toBe('')
    // 脏数据与归一后等价数据签名一致（去抖不误判）
    expect(colFilterSignature({ name: { kind: 'text', text: ' a ' } })).toBe(
      colFilterSignature({ name: { kind: 'text', text: 'a' } }),
    )
  })

  it('值不同 → 签名不同（op/num/from/to/empty 均参与）', () => {
    const base = colFilterSignature({ dur: { kind: 'number', op: 'ge', num: 3 } })
    expect(colFilterSignature({ dur: { kind: 'number', op: 'le', num: 3 } })).not.toBe(base)
    expect(colFilterSignature({ dur: { kind: 'number', op: 'ge', num: 4 } })).not.toBe(base)
    expect(colFilterSignature({ start: { kind: 'date', from: '2026-01-01' } })).not.toBe(
      colFilterSignature({ start: { kind: 'date', to: '2026-01-01' } }),
    )
    expect(colFilterSignature({ name: { kind: 'text', empty: 'empty' } })).not.toBe(
      colFilterSignature({ name: { kind: 'text', empty: 'nonEmpty' } }),
    )
  })
})
