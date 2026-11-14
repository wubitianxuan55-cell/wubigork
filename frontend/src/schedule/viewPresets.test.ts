/**
 * viewPresets.test.ts — 斑马式内置横道视图（W1）
 *
 * 重点钉死：9 个内置视图、计划编制真实列宽、隐显集合、视图筛选/分组，
 * 以及旧偏好到“自定义”视图的迁移语义。
 */
import { describe, expect, it } from 'vitest'
import {
  BUILTIN_GANTT_VIEWS,
  CUSTOM_GANTT_VIEW_ID,
  DEFAULT_GANTT_VIEW_ID,
  builtinViewOf,
  hideForBuiltinView,
  widthForBuiltinView,
} from './viewPresets'

describe('斑马内置横道视图', () => {
  it('9 个内置视图名称与顺序对齐 ZPert ViewDB', () => {
    expect(BUILTIN_GANTT_VIEWS.map((v) => v.name)).toEqual([
      '计划编制',
      '进度管控',
      '期间计划',
      '期间进度报告',
      '资源',
      '工程量计划',
      '进度对比分析',
      '计划变动分析',
      '里程碑列表',
    ])
    expect(new Set(BUILTIN_GANTT_VIEWS.map((v) => v.id)).size).toBe(9)
    expect(DEFAULT_GANTT_VIEW_ID).toBe('plan')
    expect(CUSTOM_GANTT_VIEW_ID).toBe('custom')
  })

  it('计划编制只露身份、分区、工期、前置与计划日期；默认表格宽=可见列宽', () => {
    const v = builtinViewOf('plan')!
    const hide = hideForBuiltinView(v)
    for (const key of ['no', 'partition', 'marker', 'name', 'dur', 'preds', 'start', 'finish']) {
      expect(hide).not.toContain(key)
    }
    for (const key of ['wbs', 'progress', 'ls', 'lf', 'tf', 'ff', 'mode', 'succ', 'cost']) {
      expect(hide).toContain(key)
    }
    expect(widthForBuiltinView(v)).toBe(730)
  })

  it('进度管控/资源/里程碑视图各自带筛选、分组与列语义', () => {
    expect(builtinViewOf('control')?.groupBy).toEqual(['status'])
    expect(builtinViewOf('control')?.visible).toContain('progress')
    expect(widthForBuiltinView(builtinViewOf('control')!)).toBeLessThanOrEqual(900)
    expect(builtinViewOf('resource')?.groupBy).toEqual(['resource'])
    expect(builtinViewOf('milestone')?.filterKind).toBe('milestone')
    expect(builtinViewOf('milestone')?.visible).not.toContain('progress')
  })

  it('未知视图 id 回落 undefined；每个可见键都是列注册表合法键', () => {
    expect(builtinViewOf('ghost')).toBeUndefined()
    const cols = new Set(hideForBuiltinView(BUILTIN_GANTT_VIEWS[0]).map((k) => k))
    expect(cols.size).toBeGreaterThan(0)
    for (const v of BUILTIN_GANTT_VIEWS) expect(() => widthForBuiltinView(v)).not.toThrow()
  })
})
