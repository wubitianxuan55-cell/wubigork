/**
 * hitRegion.test.ts — 命中区枚举/优先级/自限深度上溯（纯逻辑 + 属性读取）
 *
 * 口径来源：`进度计划/gaea-schedule-zpert8-uiux-port-plan-20261114.md` §W4
 * 「命中区顺序固定：列头 → 画布 → 单元格 → 状态栏 → 分隔条 → 行头」，
 * 逆向侧等价物是 `uiux-re` §17.1 的 `hitType` 硬编码 if 链（先判先得）。
 * 本用例只碰 `data-hit` 属性，不读任何 DOM 文本（文本会被就地编辑改掉）。
 */
import { describe, expect, it } from 'vitest'
import {
  HIT_ATTR,
  HIT_LOOKUP_DEPTH,
  HIT_REGION_ORDER,
  hitRegionChain,
  hitRegionOfElement,
  hitRegionRank,
  isHitRegion,
  pickHitRegion,
  type HitRegion,
} from './hitRegion'

/** 建一串嵌套 div，返回 [最内, …, 最外] 与挂到最外层的容器 */
function nest(depth: number): { inner: HTMLElement; outer: HTMLElement } {
  const outer = document.createElement('div')
  let cur = outer
  for (let i = 1; i < depth; i++) {
    const next = document.createElement('div')
    cur.appendChild(next)
    cur = next
  }
  return { inner: cur, outer }
}

describe('hitRegion 优先级表', () => {
  it('HIT_REGION_ORDER 严格等于 W4 固定顺序（列头→画布→单元格→状态栏→分隔条→行头）', () => {
    expect([...HIT_REGION_ORDER]).toEqual([
      'column-header',
      'canvas',
      'cell',
      'status-bar',
      'splitter',
      'row-header',
    ])
    expect(new Set(HIT_REGION_ORDER).size).toBe(6) // 无重复
  })

  it('hitRegionRank 与顺序同构：下标即优先级，未知区域一律 -1', () => {
    HIT_REGION_ORDER.forEach((region, idx) => expect(hitRegionRank(region)).toBe(idx))
    expect(hitRegionRank('column-header')).toBeLessThan(hitRegionRank('row-header'))
    expect(hitRegionRank('nope')).toBe(-1)
    expect(hitRegionRank(null)).toBe(-1)
    expect(hitRegionRank(undefined)).toBe(-1)
  })

  it('isHitRegion 只认六个合法值（空串/大小写差异/脏值都不认）', () => {
    expect(isHitRegion('cell')).toBe(true)
    expect(isHitRegion('')).toBe(false)
    expect(isHitRegion('Cell')).toBe(false) // 属性值大小写敏感，不做猜测式归一
    expect(isHitRegion(null)).toBe(false)
    expect(isHitRegion('canvas ')).toBe(false)
  })
})

describe('pickHitRegion 并列裁决', () => {
  it('空候选 → null（未命中就是未命中，不兜底成某个默认区）', () => {
    expect(pickHitRegion([])).toBeNull()
  })

  it('并列同一区 → 该区（重复声明不改变结果）', () => {
    expect(pickHitRegion(['cell', 'cell', 'cell'])).toBe('cell')
  })

  it('取最高优先级：单元格压过分隔条，列头压过一切', () => {
    expect(pickHitRegion(['row-header', 'splitter'])).toBe('splitter')
    expect(pickHitRegion(['splitter', 'cell'])).toBe('cell')
    expect(pickHitRegion(['canvas', 'cell', 'row-header'])).toBe('canvas') // 画布排在单元格之前
    expect(pickHitRegion(['cell', 'column-header'])).toBe('column-header')
  })

  it('未知/空值被跳过，不影响真实命中；全未知 → null', () => {
    expect(pickHitRegion(['nope', '', 'cell'])).toBe('cell')
    expect(pickHitRegion([null, undefined, 'status-bar'])).toBe('status-bar')
    expect(pickHitRegion(['nope', ''])).toBeNull()
  })
})

describe('hitRegionOfElement 自限深度上溯', () => {
  it('null / 未声明元素 → null', () => {
    expect(hitRegionOfElement(null)).toBeNull()
    const { inner } = nest(4)
    expect(hitRegionOfElement(inner)).toBeNull()
  })

  it('data-hit 向上查找：声明在祖先上，内层元素也能命中', () => {
    const { inner, outer } = nest(4)
    outer.setAttribute(HIT_ATTR, 'canvas')
    expect(hitRegionOfElement(inner)).toBe('canvas')
    expect(hitRegionChain(inner)).toEqual(['canvas'])
  })

  it('沿链声明再按优先级裁决：外层的「单元格」能盖掉内层的「行头」', () => {
    // 这条是设计约束的显式记录：想保住行头，就不能给表格行/表体声明 cell
    const { inner, outer } = nest(3)
    inner.setAttribute(HIT_ATTR, 'row-header')
    outer.setAttribute(HIT_ATTR, 'cell')
    expect(hitRegionChain(inner)).toEqual(['row-header', 'cell'])
    expect(hitRegionOfElement(inner)).toBe('cell')
  })

  it('深度上限：超出 maxDepth 的声明不算命中', () => {
    const { inner, outer } = nest(HIT_LOOKUP_DEPTH + 2)
    outer.setAttribute(HIT_ATTR, 'splitter')
    expect(hitRegionOfElement(inner)).toBeNull() // 缺省深度不够
    expect(hitRegionOfElement(inner, HIT_LOOKUP_DEPTH + 4)).toBe('splitter') // 放宽即命中
    expect(hitRegionChain(inner, HIT_LOOKUP_DEPTH).length).toBe(0)
  })

  it('深度上限含起点：maxDepth=1 只看元素自身', () => {
    const { inner, outer } = nest(2)
    inner.setAttribute(HIT_ATTR, 'cell')
    outer.setAttribute(HIT_ATTR, 'column-header')
    expect(hitRegionOfElement(inner, 1)).toBe('cell')
    expect(hitRegionOfElement(inner, 2)).toBe('column-header') // 第 2 层进入裁决
  })

  it('脏 data-hit 值跳过继续向上（不阻断真实命中）', () => {
    const { inner, outer } = nest(3)
    inner.setAttribute(HIT_ATTR, 'rownum')
    outer.setAttribute(HIT_ATTR, 'row-header')
    expect(hitRegionChain(inner)).toEqual(['row-header'])
    expect(hitRegionOfElement(inner)).toBe('row-header')
  })

  it('可换属性名读取（同一判据复用，不新增第二套约定）', () => {
    const { inner } = nest(2)
    inner.setAttribute('data-region', 'status-bar')
    expect(hitRegionOfElement(inner)).toBeNull()
    expect(hitRegionOfElement(inner, HIT_LOOKUP_DEPTH, 'data-region')).toBe('status-bar')
  })

  it('maxDepth<=0 → 空链（防御脏参数）', () => {
    const { inner } = nest(2)
    inner.setAttribute(HIT_ATTR, 'cell')
    const region: HitRegion = 'cell'
    expect(hitRegionChain(inner, 0)).toEqual([])
    expect(hitRegionOfElement(inner, 0)).toBeNull()
    expect(pickHitRegion([region])).toBe('cell')
  })
})
