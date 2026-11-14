/**
 * inplaceEdit.test.ts — 就地编辑弹层族值语义（W5）
 *
 * 用例口径（契约锁定）：date 严格 YYYY-MM-DD/日历存在、空=清空；dateRange 单边空=
 * 开区间、from<=to、两边空=清空；number NaN 不提交、step 对齐后夹取、空默认清空、
 * 显式 emptyAs:'zero' 才落 0、required 拒空；text trim/截断/空=清空。
 * 另锁：parseInplace 分派、inplaceKindOfCol / inplaceSpecOfCol 列口径。
 */
import { describe, expect, it } from 'vitest'
import {
  INPLACE_COL_KIND,
  inplaceKindOfCol,
  inplaceSpecOfCol,
  isIsoDate,
  parseDateRangeText,
  parseDateText,
  parseInplace,
  parseNumberText,
  parseTextInput,
  type InplaceResult,
} from './inplaceEdit'

/** 取失败结果（缩窄 + 断言失败分支，避免到处 if） */
function failOf(r: InplaceResult) {
  if (r.ok) throw new Error('期望校验失败，实际通过')
  return r
}

describe('date：仅 YYYY-MM-DD，空=清空，非法不提交', () => {
  it('合法日期通过并原样回传', () => {
    const r = parseDateText('2026-09-07')
    expect(r).toMatchObject({ ok: true, kind: 'date', value: '2026-09-07', cleared: false })
  })

  it('前后空白 trim 后仍合法', () => {
    expect(parseDateText('  2026-09-07  ')).toMatchObject({ ok: true, value: '2026-09-07' })
  })

  it('空串=清空（value:null，cleared:true）', () => {
    expect(parseDateText('   ')).toMatchObject({ ok: true, value: null, cleared: true })
  })

  it('格式不符（2026/09/07、2026-9-7）不提交', () => {
    expect(failOf(parseDateText('2026/09/07')).reason).toBe('invalidDate')
    expect(failOf(parseDateText('2026-9-7')).reason).toBe('invalidDate')
  })

  it('日历不存在（2026-02-30）不提交——格式合法不等于日期合法', () => {
    expect(isIsoDate('2026-02-30')).toBe(false)
    expect(failOf(parseDateText('2026-02-30')).reason).toBe('invalidDate')
  })

  it('required 列空值 → emptyForbidden', () => {
    expect(failOf(parseDateText('', { required: true })).reason).toBe('emptyForbidden')
  })
})

describe('dateRange：from<=to，单边空=开区间', () => {
  it('两边都填=闭区间', () => {
    expect(parseDateRangeText('2026-09-01', '2026-09-30')).toMatchObject({
      ok: true,
      value: { from: '2026-09-01', to: '2026-09-30' },
      cleared: false,
    })
  })

  it('单边空=开区间（该侧 null），不算清空', () => {
    expect(parseDateRangeText('2026-09-01', '')).toMatchObject({
      ok: true,
      value: { from: '2026-09-01', to: null },
      cleared: false,
    })
    expect(parseDateRangeText('', '2026-09-30')).toMatchObject({
      ok: true,
      value: { from: null, to: '2026-09-30' },
      cleared: false,
    })
  })

  it('两边都空=整区间清空', () => {
    expect(parseDateRangeText(' ', '')).toMatchObject({
      ok: true,
      value: { from: null, to: null },
      cleared: true,
    })
  })

  it('from > to 不提交（rangeOrder）', () => {
    expect(failOf(parseDateRangeText('2026-10-01', '2026-09-30')).reason).toBe('rangeOrder')
  })

  it('同一天合法（零长区间）', () => {
    expect(parseDateRangeText('2026-09-07', '2026-09-07')).toMatchObject({ ok: true })
  })

  it('单侧格式非法 → invalidRange', () => {
    expect(failOf(parseDateRangeText('2026/09/01', '')).reason).toBe('invalidRange')
  })
})

describe('number：NaN 不提交，step 对齐 + min/max 夹取，空=清空', () => {
  it('普通数值原样通过', () => {
    expect(parseNumberText('12')).toMatchObject({ ok: true, value: 12, adjusted: false, cleared: false })
  })

  it('非数值不提交（abc / 空之外的符号串）', () => {
    expect(failOf(parseNumberText('abc')).reason).toBe('notNumber')
    expect(failOf(parseNumberText('12abc')).reason).toBe('notNumber')
  })

  it('超上限/低于下限夹取（adjusted:true）', () => {
    expect(parseNumberText('150', { min: 0, max: 100 })).toMatchObject({ ok: true, value: 100, adjusted: true })
    expect(parseNumberText('-5', { min: 0, max: 100 })).toMatchObject({ ok: true, value: 0, adjusted: true })
  })

  it('step 对齐：先对齐后夹取（0.5 步长去浮点噪声）', () => {
    expect(parseNumberText('7', { min: 0, step: 5 })).toMatchObject({ ok: true, value: 5, adjusted: true })
    expect(parseNumberText('7.3', { min: 0, max: 100, step: 0.1 })).toMatchObject({ ok: true, value: 7.3 })
  })

  it('空串默认=清空（不隐式变 0）', () => {
    expect(parseNumberText('', { min: 0, max: 3650 })).toMatchObject({ ok: true, value: null, cleared: true })
  })

  it('显式 emptyAs:"zero" 才把空落成 0（工期类声明口径）', () => {
    expect(parseNumberText('', { min: 0, max: 3650, emptyAs: 'zero' })).toMatchObject({
      ok: true,
      value: 0,
      cleared: false,
    })
  })

  it('required 列空值 → emptyForbidden', () => {
    expect(failOf(parseNumberText('', { required: true })).reason).toBe('emptyForbidden')
  })
})

describe('text：trim、maxLen 截断、空=清空', () => {
  it('trim 后提交', () => {
    expect(parseTextInput('  场地三通一平  ')).toMatchObject({ ok: true, value: '场地三通一平' })
  })

  it('超长截断（adjusted:true）', () => {
    expect(parseTextInput('abcdef', { maxLen: 3 })).toMatchObject({
      ok: true,
      value: 'abc',
      adjusted: true,
      cleared: false,
    })
  })

  it('空串（含全空白）=清空', () => {
    expect(parseTextInput('   ')).toMatchObject({ ok: true, value: null, cleared: true })
  })
})

describe('parseInplace 分派 + 列口径', () => {
  it('按草稿 kind 分派到四类解析器', () => {
    expect(parseInplace({ kind: 'date', text: '2026-09-07' })).toMatchObject({ ok: true, kind: 'date' })
    expect(parseInplace({ kind: 'dateRange', from: '2026-09-01', to: '2026-09-02' })).toMatchObject({
      ok: true,
      kind: 'dateRange',
    })
    expect(parseInplace({ kind: 'number', text: '3' }, { min: 0, max: 100 })).toMatchObject({
      ok: true,
      kind: 'number',
      value: 3,
    })
    expect(parseInplace({ kind: 'text', text: ' a ' })).toMatchObject({ ok: true, kind: 'text', value: 'a' })
  })

  it('列键映射：名称 text / 工期 number / 计划开始完成 date / 自定义按类型 / 未知列 null', () => {
    expect(inplaceKindOfCol('name')).toBe('text')
    expect(inplaceKindOfCol('dur')).toBe('number')
    expect(inplaceKindOfCol('start')).toBe('date')
    expect(inplaceKindOfCol('finish')).toBe('date')
    expect(inplaceKindOfCol('text1')).toBe('text')
    expect(inplaceKindOfCol('num1')).toBe('number')
    expect(inplaceKindOfCol('wbs')).toBeNull()
    // 自定义列按注册表类型覆盖同名列口径
    expect(inplaceKindOfCol('name', 'number')).toBe('number')
  })

  it('列规格：工期 0-3650 显式 zero 口径，进度 0-100，未登记列=空规格', () => {
    expect(inplaceSpecOfCol('dur')).toMatchObject({ min: 0, max: 3650, step: 1, emptyAs: 'zero' })
    expect(inplaceSpecOfCol('progress')).toMatchObject({ min: 0, max: 100 })
    expect(inplaceSpecOfCol('wbs')).toEqual({})
    expect(INPLACE_COL_KIND.name).toBe('text')
  })
})
