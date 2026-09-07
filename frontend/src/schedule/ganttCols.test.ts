/**
 * ganttCols.test.ts — 横道列显隐/表格窗格宽度纯函数用例（v4.131.0 刀C；v4.138 #14 增自定义字段列）
 */
import { describe, expect, it } from 'vitest'
import { GANTT_COLS, GANTT_CUSTOM_KEYS, GANTT_FIXED_KEYS, GANTT_HIDEABLE_KEYS, GANTT_LEFT_W_FULL, GANTT_REPORT_KEYS, GANTT_TABLE_W_MIN, clampTableW, colsByKeys, sanitizeHide, visibleCols, visibleLeftW } from './ganttCols'

describe('ganttCols 列定义', () => {
  it('20 列（15 标准+5 自定义）、行号+名称固定、全列宽=各列宽求和', () => {
    expect(GANTT_COLS).toHaveLength(20)
    expect(GANTT_FIXED_KEYS).toEqual(['no', 'name'])
    expect(GANTT_HIDEABLE_KEYS).toHaveLength(18)
    expect(GANTT_LEFT_W_FULL).toBe(GANTT_COLS.reduce((s, c) => s + c.w, 0))
    expect(GANTT_TABLE_W_MIN).toBe(34 + 160)
  })
})

describe('自定义字段列（v4.138 #14）', () => {
  it('GANTT_CUSTOM_KEYS=注册表序 5 槽、排成本列之后、宽 text 90 / number 64', () => {
    expect(GANTT_CUSTOM_KEYS).toEqual(['text1', 'text2', 'text3', 'num1', 'num2'])
    const keys = GANTT_COLS.map((c) => c.key)
    expect(keys[14]).toBe('cost')
    expect(keys.slice(15)).toEqual(GANTT_CUSTOM_KEYS)
    const w = Object.fromEntries(GANTT_COLS.map((c) => [c.key, c.w]))
    expect(w.text1).toBe(90)
    expect(w.text3).toBe(90)
    expect(w.num1).toBe(64)
    expect(w.num2).toBe(64)
  })
  it('visibleCols：hide 含自定义键即隐藏（缺省收起口径），既有 15 列键序零变化', () => {
    const hidden = visibleCols(['progress', ...GANTT_CUSTOM_KEYS])
    const keys = hidden.map((c) => c.key)
    for (const k of GANTT_CUSTOM_KEYS) expect(keys).not.toContain(k)
    expect(keys).toEqual(
      ['no', 'name', 'wbs', 'dur', 'start', 'finish', 'ls', 'lf', 'tf', 'ff', 'mode', 'preds', 'succ', 'cost'],
    )
    // 不藏自定义列时键序=原 15 列 + 自定义 5 列
    expect(visibleCols(['progress']).map((c) => c.key)).toEqual(
      ['no', 'name', 'wbs', 'dur', 'start', 'finish', 'ls', 'lf', 'tf', 'ff', 'mode', 'preds', 'succ', 'cost', ...GANTT_CUSTOM_KEYS],
    )
  })
  it('自定义列名：不传 customLabels=注册表缺省名；项目覆盖优先；空串/空白覆盖回落；标准列名不受影响', () => {
    const dft = visibleCols([], undefined)
    expect(dft.find((c) => c.key === 'text1')!.label).toBe('文本1')
    expect(dft.find((c) => c.key === 'num2')!.label).toBe('数值2')
    const renamed = visibleCols([], { text1: '施工部位', num1: '方量', text2: '  ' })
    expect(renamed.find((c) => c.key === 'text1')!.label).toBe('施工部位')
    expect(renamed.find((c) => c.key === 'num1')!.label).toBe('方量')
    expect(renamed.find((c) => c.key === 'text2')!.label).toBe('文本2')
    expect(renamed.find((c) => c.key === 'name')!.label).toBe('任务名称')
    expect(renamed.find((c) => c.key === 'cost')!.label).toBe('成本')
  })
})

describe('visibleCols / visibleLeftW', () => {
  it('隐藏列从序列与总宽中剔除，固定列永不可隐藏', () => {
    const cols = visibleCols(['ls', 'lf', 'tf', 'ff'])
    expect(cols.map((c) => c.key)).not.toContain('ls')
    expect(cols.map((c) => c.key)).toContain('name')
    const full = visibleLeftW([])
    const hidden = visibleLeftW(['ls', 'lf', 'tf', 'ff'])
    expect(full - hidden).toBe(66 + 66 + 44 + 58)
    // 固定列即使混入 hide 也不生效（防御）
    expect(visibleCols(['no', 'name']).map((c) => c.key)).toEqual(['no', 'name', ...GANTT_HIDEABLE_KEYS])
  })
  it('全藏只剩行号+名称（收纳表格最小集）', () => {
    const cols = visibleCols([...GANTT_HIDEABLE_KEYS])
    expect(cols.map((c) => c.key)).toEqual(['no', 'name'])
    expect(visibleLeftW([...GANTT_HIDEABLE_KEYS])).toBe(GANTT_TABLE_W_MIN)
  })
})

describe('clampTableW', () => {
  it('钳位 [GANTT_TABLE_W_MIN, leftW]；非有限值回落 leftW', () => {
    const full = GANTT_LEFT_W_FULL
    expect(clampTableW(500, full)).toBe(500)
    expect(clampTableW(10, full)).toBe(GANTT_TABLE_W_MIN)
    expect(clampTableW(99999, full)).toBe(full)
    expect(clampTableW(NaN, full)).toBe(full)
    // 已隐藏多列时上限=可见列总宽
    const narrow = visibleLeftW(['ls', 'lf'])
    expect(clampTableW(99999, narrow)).toBe(narrow)
  })
})

describe('sanitizeHide', () => {
  it('非数组/非法键丢弃、去重、上限 32', () => {
    expect(sanitizeHide(null)).toEqual([])
    expect(sanitizeHide('ls')).toEqual([])
    expect(sanitizeHide(['ls', 'ghost', 3, 'ls', 'no'])).toEqual(['ls'])
    expect(sanitizeHide(Array.from({ length: 40 }, (_, i) => GANTT_HIDEABLE_KEYS[i % 12])).length).toBeLessThanOrEqual(12)
  })
})

describe('上报 6 列预设（刀D1）', () => {
  it('GANTT_REPORT_KEYS 保序取列，恰为上报件口径', () => {
    expect(GANTT_REPORT_KEYS).toEqual(['no', 'name', 'dur', 'start', 'finish', 'preds'])
    const cols = colsByKeys(GANTT_REPORT_KEYS)
    expect(cols.map((c) => c.key)).toEqual(GANTT_REPORT_KEYS)
    // 未知键不进结果；重复键幂等
    expect(colsByKeys(['no', 'ghost', 'name', 'name']).map((c) => c.key)).toEqual(['no', 'name'])
  })
})
