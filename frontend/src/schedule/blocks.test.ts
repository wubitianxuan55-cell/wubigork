/**
 * schedule/blocks.test.ts — 斑马复刻四块承载位的前端归一用例（与 Go blocks_test.go 互为镜像）
 *
 * 关注两件事：
 * 1. **能打开**：坏形数据逐条丢弃、不抛异常（normalizeProject 是板块唯一入口）；
 * 2. **不误伤**：合法数据逐字段保真、缺省一律 undefined（旧文件零迁移）。
 */

import { describe, expect, it } from 'vitest'
import {
  MAX_REVIEWS,
  MAX_VIEWS,
  normalizeAoaGraph,
  normalizeNetworkStyle,
  normalizeReviews,
  normalizeViews,
} from './blocks'
import { normalizeProject } from './store'
import type { SchedProject } from './types'

const base = (): SchedProject => ({
  name: '测试',
  startDate: '2026-09-01',
  tasks: [{ id: 'A', name: 'A', duration: 3, level: 1, progress: 0 }],
  links: [],
})

describe('normalizeReviews', () => {
  it('保留合法条目并按 at 升序', () => {
    const out = normalizeReviews([
      { at: 5, date: '2026-09-08', actuals: { A: { percent: 100, start: 0 } } },
      { at: 0, date: '2026-09-01', actuals: {} },
    ])
    expect(out?.map((r) => r.at)).toEqual([0, 5])
    expect(out?.[1].actuals.A).toEqual({ start: 0, percent: 100 })
  })

  it('at 非整数 / date 形状错 → 整条丢弃', () => {
    expect(normalizeReviews([{ at: -1, date: '2026-09-01' }, { at: 1.5, date: '2026-09-01' }])).toBeUndefined()
    expect(normalizeReviews([{ at: 1, date: '2026/09/01' }])).toBeUndefined()
    expect(normalizeReviews([{ at: 1 }])).toBeUndefined()
  })

  it('同一 at 重复 → 保留靠后那条（手改文件的容错口径）', () => {
    const out = normalizeReviews([
      { at: 2, date: '2026-09-03', note: '旧' },
      { at: 2, date: '2026-09-03', note: '新' },
    ])
    expect(out).toHaveLength(1)
    expect(out?.[0].note).toBe('新')
  })

  it('percent 夹到 0-100，非法 start/finish 丢弃该键；条目全空则丢该任务', () => {
    const out = normalizeReviews([
      { at: 1, date: '2026-09-02', actuals: { A: { percent: 168, start: -3 }, B: { percent: 'x' } } },
    ])
    expect(out?.[0].actuals.A).toEqual({ percent: 100 })
    expect(out?.[0].actuals.B).toBeUndefined()
  })

  it('超上限保留最近的若干期（历史头部可丢，近期不可丢）', () => {
    const many = Array.from({ length: MAX_REVIEWS + 5 }, (_, i) => ({ at: i, date: '2026-09-01' }))
    const out = normalizeReviews(many)
    expect(out).toHaveLength(MAX_REVIEWS)
    expect(out?.[out.length - 1].at).toBe(MAX_REVIEWS + 4)
  })

  it('非数组 / 空数组 → undefined（缺省=旧行为）', () => {
    expect(normalizeReviews(undefined)).toBeUndefined()
    expect(normalizeReviews({})).toBeUndefined()
    expect(normalizeReviews([])).toBeUndefined()
  })
})

describe('normalizeAoaGraph', () => {
  const good = {
    dots: [
      { id: 'S', num: 1, isBegin: true },
      { id: 'e1', num: 2, layer: 3 },
      { id: 'T', num: 3, isEnd: true },
    ],
    arrows: [
      { id: 'a1', from: 'S', to: 'e1', kind: 'task', taskId: 'A' },
      { id: 'a2', from: 'e1', to: 'T', kind: 'dummy', lag: -2 },
    ],
    beginDotId: 'S',
  }

  it('合法图逐字段保真（含 lag 负值时距与 beginDotId）', () => {
    const g = normalizeAoaGraph(good)
    expect(g?.dots).toHaveLength(3)
    expect(g?.dots[1]).toEqual({ id: 'e1', num: 2, layer: 3 })
    expect(g?.arrows[1]).toEqual({ id: 'a2', from: 'e1', to: 'T', kind: 'dummy', lag: -2 })
    expect(g?.beginDotId).toBe('S')
  })

  it('点：空 id / num<1 / id 重复 → 丢该点', () => {
    const g = normalizeAoaGraph({
      dots: [{ id: '', num: 1 }, { id: 'a', num: 0 }, { id: 'a', num: 1 }, { id: 'a', num: 2 }, { id: 'b', num: 3 }],
      arrows: [{ id: 'x', from: 'a', to: 'b', kind: 'task' }],
    })
    expect(g?.dots.map((d) => d.id)).toEqual(['a', 'b'])
  })

  it('箭：端点不存在 / 自环 / 类型非法 → 丢该箭', () => {
    const g = normalizeAoaGraph({
      dots: good.dots,
      arrows: [
        { id: 'a1', from: 'S', to: 'nope', kind: 'task' },
        { id: 'a2', from: 'S', to: 'S', kind: 'task' },
        { id: 'a3', from: 'S', to: 'T', kind: 'flow' },
        { id: 'a4', from: 'S', to: 'T', kind: 'summary' },
      ],
    })
    expect(g?.arrows.map((a) => a.id)).toEqual(['a4'])
  })

  it('lag 越界（±9999）丢弃该键；时距非法不整箭丢弃', () => {
    const g = normalizeAoaGraph({
      dots: good.dots,
      arrows: [{ id: 'a1', from: 'S', to: 'T', kind: 'task', lag: 20000 }],
    })
    expect(g?.arrows[0].lag).toBeUndefined()
  })

  it('只有点或只有箭 → undefined（v1 无意义）', () => {
    expect(normalizeAoaGraph({ dots: good.dots, arrows: [] })).toBeUndefined()
    expect(normalizeAoaGraph({ dots: [], arrows: good.arrows })).toBeUndefined()
    expect(normalizeAoaGraph(null)).toBeUndefined()
  })
})

describe('normalizeNetworkStyle', () => {
  it('合法枚举与数值保真', () => {
    const s = normalizeNetworkStyle({
      mode: 'logic', dotStyle: 'rect', arrowStyle: 'hollow', namePos: 'vertical',
      nodeRadius: 20, headHeight: 8, layerHeight1: 112, layerHeight2: 60,
      stagger: true, hideUnimportantLink: false, tickUnit: 'dayHour', tickLines: 'large',
      density: 'compact', aspectMax: 5, maxPages: 120,
    })
    expect(s).toEqual({
      mode: 'logic', dotStyle: 'rect', arrowStyle: 'hollow', namePos: 'vertical',
      nodeRadius: 20, headHeight: 8, layerHeight1: 112, layerHeight2: 60,
      stagger: true, hideUnimportantLink: false, tickUnit: 'dayHour', tickLines: 'large',
      density: 'compact', aspectMax: 5, maxPages: 120,
    })
  })

  it('非法枚举丢该键而非整块；越界数值丢该键', () => {
    const s = normalizeNetworkStyle({ mode: 'fancy', density: 'loose', nodeRadius: 999, aspectMax: 99 })
    expect(s).toEqual({ density: 'loose' })
  })

  it('全无效 → undefined', () => {
    expect(normalizeNetworkStyle({ mode: 'x', nodeRadius: 1 })).toBeUndefined()
    expect(normalizeNetworkStyle(undefined)).toBeUndefined()
    expect(normalizeNetworkStyle([])).toBeUndefined()
  })
})

describe('normalizeViews', () => {
  const v = (over: Record<string, unknown> = {}) => ({ id: 'v1', name: '视图', kind: 'gantt', ...over })

  it('合法视图保真（含列/隐藏/分组/排序/显示模式）', () => {
    const [view] = normalizeViews([
      v({ cols: ['no', 'name'], hide: ['code'], groupBy: ['level'], filter: 'critical',
        sort: { field: 'start', dir: 'desc' }, renderMode: 'timescaledLogic' }),
    ]) ?? []
    expect(view.cols).toEqual(['no', 'name'])
    expect(view.groupBy).toEqual(['level'])
    expect(view.sort).toEqual({ field: 'start', dir: 'desc' })
    expect(view.renderMode).toBe('timescaledLogic')
  })

  it('缺 id/名 或 kind 非法 → 丢该视图；id 重复 → 只留先出现', () => {
    const out = normalizeViews([
      v({ id: '' }), v({ name: '  ' }), v({ kind: 'kanban' }), v({ id: 'dup' }), v({ id: 'dup', name: '第二' }),
    ])
    expect(out?.map((x) => x.id)).toEqual(['dup'])
    expect(out?.[0].name).toBe('视图')
  })

  it('排序方向非法 → 丢 sort 而非整视图', () => {
    const [view] = normalizeViews([v({ sort: { field: 'start', dir: 'up' } })]) ?? []
    expect(view.sort).toBeUndefined()
    expect(view.id).toBe('v1')
  })

  it('超上限截断，空列表 → undefined', () => {
    const many = Array.from({ length: MAX_VIEWS + 3 }, (_, i) => v({ id: `v${i}` }))
    expect(normalizeViews(many)).toHaveLength(MAX_VIEWS)
    expect(normalizeViews([])).toBeUndefined()
  })
})

describe('normalizeProject 透传四块（板块唯一入口）', () => {
  it('四块存活且被归一；缺省仍为 undefined', () => {
    const p = base()
    p.reviews = [{ at: 1, date: '2026-09-02', actuals: { A: { percent: 50 } } }]
    p.aoa = {
      dots: [{ id: 'S', num: 1, isBegin: true }, { id: 'T', num: 2, isEnd: true }],
      arrows: [{ id: 'a1', from: 'S', to: 'T', kind: 'task' }],
    }
    p.networkStyle = { mode: 'timescaled', density: 'normal' }
    p.views = [{ id: 'v1', name: '计划编制', kind: 'gantt' }]

    const out = normalizeProject(p)
    expect(out.reviews?.[0].at).toBe(1)
    expect(out.aoa?.arrows).toHaveLength(1)
    expect(out.networkStyle).toEqual({ mode: 'timescaled', density: 'normal' })
    expect(out.views?.[0].name).toBe('计划编制')

    const legacy = normalizeProject(base())
    expect(legacy.reviews).toBeUndefined()
    expect(legacy.aoa).toBeUndefined()
    expect(legacy.networkStyle).toBeUndefined()
    expect(legacy.views).toBeUndefined()
  })

  it('坏形数据不抛异常（板块必须能打开）', () => {
    const p = base() as unknown as Record<string, unknown>
    p.reviews = 'x'
    p.aoa = 42
    p.networkStyle = []
    p.views = { a: 1 }
    expect(() => normalizeProject(p as unknown as SchedProject)).not.toThrow()
    const out = normalizeProject(p as unknown as SchedProject)
    expect(out.reviews).toBeUndefined()
    expect(out.aoa).toBeUndefined()
    expect(out.networkStyle).toBeUndefined()
    expect(out.views).toBeUndefined()
  })
})
