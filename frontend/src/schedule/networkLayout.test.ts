/**
 * networkLayout.test.ts — 布图参数纯函数单测（W4 收口：density/layerHeight/stagger/namePos）
 *
 * 纪律：缺省必须零回归——`resolveNetworkLayout()` 的 `isDefault=true` 且
 * `applyNetworkLayout` 返回**同一引用**；所有坐标断言都按 AOA_MARGIN/AOA_ROW_H
 * 同一坐标系算死，不写「差不多」。
 */
import { describe, expect, it } from 'vitest'
import { buildAoa } from './aoa'
import { AOA_COL_W, AOA_MARGIN, AOA_ROW_H } from './aoa'
import {
  AOA_BAND_MIN_H,
  AOA_ROW_MAX_H,
  AOA_ROW_MIN_H,
  NETWORK_LAYOUT_DEFAULTS,
  PDM_COL_GAP,
  PDM_NODE_H,
  PDM_NODE_W,
  PDM_RADIUS_BASE,
  PDM_RADIUS_MAX,
  PDM_RADIUS_MIN,
  PDM_ROW_GAP,
  applyNetworkLayout,
  deriveRowPitch,
  edgeMidpoint,
  isUnimportantLink,
  resolveNamePlacement,
  resolveNetworkLayout,
  resolvePdmNodeMetrics,
} from './networkLayout'
import { NETWORK_GEOMETRY_DEFAULTS } from './networkDisplay'
import type { Seg } from './aoaLayout'
import type { SchedLink, SchedTask } from './types'

function t(id: string, duration: number, extra?: Partial<SchedTask>): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0, ...extra }
}
function l(from: string, to: string, lag = 0): SchedLink {
  return { from, to, type: 'FS', lag }
}

/** 菱形（多行 + 单横幅）：A(3) → B(2)/C(4) → D(1) */
function diamond() {
  return buildAoa(
    [t('A', 3), t('B', 2), t('C', 4), t('D', 1)],
    [l('A', 'B'), l('A', 'C'), l('B', 'D'), l('C', 'D')],
  )
}

/** 两横幅（分组行 level 0 = 一级横幅）：G1[A,B] / G2[C] */
function banded() {
  return buildAoa(
    [t('G1', 0, { level: 0 }), t('A', 3), t('B', 2), t('G2', 0, { level: 0 }), t('C', 4)],
    [l('A', 'B'), l('B', 'C')],
  )
}

describe('resolveNetworkLayout：五键解析与缺省零回归', () => {
  it('全缺省：系数 1、isDefault=true；undefined/null/空对象三种入参逐字段相等', () => {
    const a = resolveNetworkLayout(undefined)
    const b = resolveNetworkLayout(null)
    const c = resolveNetworkLayout({})
    expect(a).toEqual(b)
    expect(a).toEqual(c)
    expect(a).toEqual(NETWORK_LAYOUT_DEFAULTS)
    expect(a.isDefault).toBe(true)
    expect(a.colScale).toBe(1)
    expect(a.rowScale).toBe(1)
    expect(a.colW).toBe(AOA_COL_W)
    expect(a.rowMin).toBe(AOA_ROW_MIN_H)
    expect(a.rowMax).toBe(AOA_ROW_MAX_H)
    expect(a.bandMin).toBe(AOA_BAND_MIN_H)
    expect(a.stagger).toBe(false)
    expect(a.namePos).toBe('auto')
    expect(a.hideUnimportantLink).toBe(false)
  })

  it('density 三档系数：compact 0.85 / normal 1 / loose 1.15；非法值回落 normal', () => {
    expect(resolveNetworkLayout({ density: 'compact' }).colScale).toBeCloseTo(0.85)
    expect(resolveNetworkLayout({ density: 'compact' }).rowScale).toBeCloseTo(0.85)
    expect(resolveNetworkLayout({ density: 'compact' }).colW).toBeCloseTo(AOA_COL_W * 0.85)
    expect(resolveNetworkLayout({ density: 'compact' }).isDefault).toBe(false)
    expect(resolveNetworkLayout({ density: 'normal' }).colScale).toBe(1)
    expect(resolveNetworkLayout({ density: 'normal' }).isDefault).toBe(true)
    expect(resolveNetworkLayout({ density: 'loose' }).colScale).toBeCloseTo(1.15)
    // 运行期脏值：非法档位回落 normal（不是 0、不是 NaN）
    const dirty = resolveNetworkLayout({ density: 'huge' as unknown as 'loose' })
    expect(dirty.density).toBe('normal')
    expect(dirty.colScale).toBe(1)
    expect(dirty.isDefault).toBe(true)
  })

  it('layerHeight1/2：二级层高=行距下限、一级层高=行距上限+横幅带高下限', () => {
    const p = resolveNetworkLayout({ layerHeight1: 200, layerHeight2: 64 })
    expect(p.rowMin).toBe(64)
    expect(p.rowMax).toBe(200)
    expect(p.bandMin).toBe(200)
    expect(p.isDefault).toBe(false)
    // 单项：只给 layerHeight2 → 下限变、上限仍缺省
    const only2 = resolveNetworkLayout({ layerHeight2: 64 })
    expect(only2.rowMin).toBe(64)
    expect(only2.rowMax).toBe(AOA_ROW_MAX_H)
    expect(only2.bandMin).toBe(AOA_BAND_MIN_H)
    // 越界（<下限/>上限）回落缺省：不让脏值把行距压成负数或撑爆画布
    const bad = resolveNetworkLayout({ layerHeight1: 5000, layerHeight2: 1 })
    expect(bad.rowMin).toBe(AOA_ROW_MIN_H)
    expect(bad.rowMax).toBe(AOA_ROW_MAX_H)
    expect(bad.bandMin).toBe(AOA_BAND_MIN_H)
    expect(bad.isDefault).toBe(true)
  })

  it('layerHeight1 < layerHeight2（运行期脏值）：上限抬到下限，不产生负区间', () => {
    const p = resolveNetworkLayout({ layerHeight1: 120, layerHeight2: 200 })
    expect(p.rowMin).toBe(200)
    expect(p.rowMax).toBe(200)
    expect(p.rowMax).toBeGreaterThanOrEqual(p.rowMin)
  })

  it('stagger / namePos / hideUnimportantLink 只认显式值，脏值回落缺省', () => {
    expect(resolveNetworkLayout({ stagger: true }).stagger).toBe(true)
    expect(resolveNetworkLayout({ stagger: 'yes' as unknown as boolean }).stagger).toBe(false)
    expect(resolveNetworkLayout({ namePos: 'vertical' }).namePos).toBe('vertical')
    expect(resolveNetworkLayout({ namePos: 'diagonal' as unknown as 'auto' }).namePos).toBe('auto')
    expect(resolveNetworkLayout({ hideUnimportantLink: true }).hideUnimportantLink).toBe(true)
    // 显示开关不是几何量：不改 isDefault（几何仍可走零回归路径）
    expect(resolveNetworkLayout({ hideUnimportantLink: true }).isDefault).toBe(true)
    expect(resolveNetworkLayout({ namePos: 'vertical' }).isDefault).toBe(true)
  })
})

describe('applyNetworkLayout：坐标变换 + 缺省零回归锚点', () => {
  it('缺省零回归锚点：applyNetworkLayout 返回**同一引用**，节点坐标数组逐字相等', () => {
    const g = diamond()
    const params = resolveNetworkLayout(undefined)
    const out = applyNetworkLayout(g, params)
    expect(out).toBe(g)
    expect(out.nodes.map((n) => [n.x, n.y])).toEqual(g.nodes.map((n) => [n.x, n.y]))
    // 显式写上「缺省值」也必须等价（不落 networkStyle vs 落全缺省）
    const explicit = applyNetworkLayout(g, resolveNetworkLayout({ density: 'normal', stagger: false }))
    expect(explicit).toBe(g)
  })

  it('density=compact：列距/行距按 0.85 压缩，拓扑/时间参数/编号逐字不变', () => {
    const g = diamond()
    const out = applyNetworkLayout(g, resolveNetworkLayout({ density: 'compact' }))
    expect(out.nodes).toHaveLength(g.nodes.length)
    const pitch = deriveRowPitch(g.nodes) * 0.85
    for (let i = 0; i < g.nodes.length; i++) {
      const a = g.nodes[i]
      const b = out.nodes[i]
      // 列：以左边距为原点线性压缩（时标线性不破）
      expect(b.x).toBeCloseTo(AOA_MARGIN + (a.x - AOA_MARGIN) * 0.85)
      // 行：按行号 × 压缩后的行距（首行贴 AOA_MARGIN + 半行距）
      const row = Math.round((a.y - AOA_MARGIN - AOA_ROW_H / 2) / AOA_ROW_H)
      expect(b.y).toBeCloseTo(AOA_MARGIN + pitch / 2 + row * pitch)
      // 非坐标字段一律不动：拓扑/时间/编号/横幅
      expect(b.es).toBe(a.es)
      expect(b.ls).toBe(a.ls)
      expect(b.num).toBe(a.num)
      expect(b.band).toBe(a.band)
      expect(b.anchor).toBe(a.anchor)
    }
    expect(out.edges).toBe(g.edges) // 箭线数组引用不变
    expect(out.taskEdge).toBe(g.taskEdge)
  })

  it('layerHeight2 下限：行距被夹到下限为止（不许比二级层高更密）', () => {
    const g = diamond()
    const p = resolveNetworkLayout({ density: 'compact', layerHeight2: 100 })
    const out = applyNetworkLayout(g, p)
    // base 112 × 0.85 = 95.2 < 100 → 取 100
    expect(deriveRowPitch(out.nodes)).toBeCloseTo(100)
    expect(deriveRowPitch(out.nodes)).toBeGreaterThanOrEqual(p.rowMin)
  })

  it('layerHeight1 横幅带高下限：带高不足时**只在它之后的横幅**整体下推（band 0 贴顶）', () => {
    const g = banded()
    const rows = [...new Set(g.nodes.map((n) => n.y))].sort((a, b) => a - b)
    // band0 单行 + 横幅间空档 + band1 单行 ⇒ 两个不同 y，行序号差 2
    expect(rows).toHaveLength(2)
    const plain = applyNetworkLayout(g, resolveNetworkLayout({ density: 'loose' }))
    const tall = applyNetworkLayout(g, resolveNetworkLayout({ density: 'loose', layerHeight1: 400 }))
    const band0Plain = plain.nodes.filter((n) => n.band === 0)
    const band0Tall = tall.nodes.filter((n) => n.band === 0)
    const band1Plain = plain.nodes.filter((n) => n.band === 1)
    const band1Tall = tall.nodes.filter((n) => n.band === 1)
    // band 0 位置不变（贴顶，不产生顶部空档）
    for (let i = 0; i < band0Plain.length; i++) expect(band0Tall[i].y).toBeCloseTo(band0Plain[i].y)
    // band 1 整体下推：Δ = layerHeight1 − band0 带高（单行 = 一个有效行距 = 112×1.15）
    const pitch = AOA_ROW_H * 1.15
    const delta = Math.min(...band1Tall.map((n) => n.y)) - Math.min(...band1Plain.map((n) => n.y))
    expect(delta).toBeCloseTo(400 - pitch)
  })

  it('stagger：奇数行再错开半个行距；手动布局（manual=true）时不生效', () => {
    const g = diamond()
    // 显式写「stagger:false + density normal」= 几何全缺省 ⇒ 同一引用（零回归）
    const base = applyNetworkLayout(g, resolveNetworkLayout({ density: 'normal', stagger: false }))
    expect(base).toBe(g)
    const params = resolveNetworkLayout({ stagger: true })
    expect(params.isDefault).toBe(false)
    const out = applyNetworkLayout(g, params)
    const pitch = deriveRowPitch(g.nodes)
    const rows = [...new Set(g.nodes.map((n) => n.y))].sort((a, b) => a - b)
    for (let i = 0; i < g.nodes.length; i++) {
      const shift = rows.indexOf(g.nodes[i].y) % 2 === 1 ? pitch / 2 : 0
      expect(out.nodes[i].y).toBeCloseTo(g.nodes[i].y + shift)
      expect(out.nodes[i].x).toBeCloseTo(g.nodes[i].x)
    }
    // 手动 pin 在场：错层停用（坐标与不开启逐字相同）
    const manualOut = applyNetworkLayout(g, params, { manual: true })
    expect(manualOut.nodes.map((n) => n.y)).toEqual(g.nodes.map((n) => n.y))
  })
})

describe('deriveRowPitch / edgeMidpoint / isUnimportantLink / resolveNamePlacement', () => {
  it('deriveRowPitch：多行取最小正差；单行/空图回落 AOA_ROW_H', () => {
    expect(deriveRowPitch([{ y: 96 }, { y: 208 }, { y: 320 }])).toBe(112)
    expect(deriveRowPitch([{ y: 96 }, { y: 96 }])).toBe(AOA_ROW_H)
    expect(deriveRowPitch([])).toBe(AOA_ROW_H)
    expect(deriveRowPitch([{ y: 100 }, { y: 340 }], 60)).toBe(240)
  })

  it('edgeMidpoint：水平段取包围盒中心；纯竖段取线 x 与中点 y', () => {
    expect(edgeMidpoint([{ x1: 100, y1: 50, x2: 300, y2: 50 }])).toEqual({ x: 200, y: 50 })
    expect(edgeMidpoint([{ x1: 200, y1: 100, x2: 200, y2: 300 }])).toEqual({ x: 200, y: 200 })
    // H-V-H：包围盒中心（不是第一段水平线中点）
    expect(
      edgeMidpoint([
        { x1: 100, y1: 100, x2: 200, y2: 100 },
        { x1: 200, y1: 100, x2: 200, y2: 300 },
        { x1: 200, y1: 300, x2: 400, y2: 300 },
      ]),
    ).toEqual({ x: 250, y: 200 })
    expect(edgeMidpoint([])).toEqual({ x: 0, y: 0 })
  })

  it('isUnimportantLink：非关键实工作=true；关键工作/虚工作/汇总线=false', () => {
    expect(isUnimportantLink({ kind: 'task', critical: false })).toBe(true)
    expect(isUnimportantLink({ kind: 'task', critical: true })).toBe(false)
    expect(isUnimportantLink({ kind: 'dummy', critical: false })).toBe(false)
    expect(isUnimportantLink({ kind: 'summary', critical: false })).toBe(false)
  })

  it('resolveNamePlacement：auto 逐字等价（无 rotate）；横向=水平居中；纵向=旋转 −90°', () => {
    const h: Seg[] = [{ x1: 100, y1: 50, x2: 300, y2: 50 }]
    const label = { x: 200, y: 45, anchor: 'middle' as const }
    const auto = resolveNamePlacement(label, h, 'auto')
    expect(auto.x).toBe(label.x)
    expect(auto.y).toBe(label.y)
    expect(auto.anchor).toBe(label.anchor)
    expect('rotate' in auto).toBe(false)
    // 横向：包围盒中心水平居中，不旋转
    const hs = [{ x1: 100, y1: 100, x2: 200, y2: 100 } as Seg, { x1: 200, y1: 100, x2: 200, y2: 300 } as Seg, { x1: 200, y1: 300, x2: 400, y2: 300 } as Seg]
    const horiz = resolveNamePlacement({ x: 150, y: 95, anchor: 'middle' }, hs, 'horizontal')
    expect(horiz.x).toBe(250)
    expect(horiz.anchor).toBe('middle')
    expect('rotate' in horiz).toBe(false)
    // 纵向：旋转 −90°；竖边右移 7px 不压线
    const vert = resolveNamePlacement({ x: 207, y: 200, anchor: 'start' }, [{ x1: 200, y1: 100, x2: 200, y2: 300 }], 'vertical')
    expect(vert.rotate).toBe(-90)
    expect(vert.x).toBe(207)
    expect(vert.anchor).toBe('middle')
  })
})

// ── W6 收尾：单代号节点尺度（nodeRadius → k=radius/16）────────────────────
//
// 单代号节点是矩形盒，半径只能当**尺度系数**用：盒宽高、列距、行距同源同系数，
// 缺省 16 ⇒ k=1 ⇒ 四个量逐字等于基准常量（视图输出零回归的算术前提）。
describe('resolvePdmNodeMetrics：单代号半径尺度系数（W6 收尾）', () => {
  it('缺省 / 显式 16 / undefined 三种入参逐字段相等，且等于基准常量', () => {
    const base = resolvePdmNodeMetrics(undefined)
    expect(base).toEqual({
      k: 1,
      nodeW: PDM_NODE_W,
      nodeH: PDM_NODE_H,
      colGap: PDM_COL_GAP,
      rowGap: PDM_ROW_GAP,
      isDefault: true,
    })
    expect(base).toEqual(resolvePdmNodeMetrics(null))
    expect(base).toEqual(resolvePdmNodeMetrics(PDM_RADIUS_BASE))
    expect(base).toEqual(resolvePdmNodeMetrics(NETWORK_GEOMETRY_DEFAULTS.nodeRadius))
    // 基准常量本身没漂移（改这四个数就是改单代号缺省视觉，用例挡一道）
    expect([PDM_NODE_W, PDM_NODE_H, PDM_COL_GAP, PDM_ROW_GAP]).toEqual([150, 92, 74, 40])
  })

  it('20 / 12 等比：盒宽高与列/行距同一系数，isDefault=false', () => {
    const big = resolvePdmNodeMetrics(20)
    expect(big.k).toBeCloseTo(1.25)
    expect(big.nodeW).toBeCloseTo(187.5)
    expect(big.nodeH).toBeCloseTo(115)
    expect(big.colGap).toBeCloseTo(92.5)
    expect(big.rowGap).toBeCloseTo(50)
    expect(big.isDefault).toBe(false)
    const small = resolvePdmNodeMetrics(12)
    expect(small.k).toBeCloseTo(0.75)
    expect(small.nodeW).toBeCloseTo(112.5)
    expect(small.nodeH).toBeCloseTo(69)
    expect(small.colGap).toBeCloseTo(55.5)
    expect(small.rowGap).toBeCloseTo(30)
    expect(small.isDefault).toBe(false)
    // 线性：盒宽比 = 半径比
    expect(big.nodeW / small.nodeW).toBeCloseTo(20 / 12)
  })

  it('脏值/越界夹进 [4,200]：全部有限且为正，任何半径都不会算出非正间距', () => {
    for (const dirty of [Number.NaN, Number.POSITIVE_INFINITY, -20, 0, 999, null, undefined]) {
      const m = resolvePdmNodeMetrics(dirty as number | null | undefined)
      expect(Number.isFinite(m.k)).toBe(true)
      expect(m.nodeW).toBeGreaterThan(0)
      expect(m.nodeH).toBeGreaterThan(0)
      expect(m.colGap).toBeGreaterThan(0)
      expect(m.rowGap).toBeGreaterThan(0)
    }
    // 越界两侧都夹在夹取域端点：999→200、0→4（不是 0 也不是负数）
    expect(resolvePdmNodeMetrics(999).k).toBeCloseTo(PDM_RADIUS_MAX / PDM_RADIUS_BASE)
    expect(resolvePdmNodeMetrics(0).k).toBeCloseTo(PDM_RADIUS_MIN / PDM_RADIUS_BASE)
    // 下限半径下「净列距」仍为正：盒宽 < 盒宽+列距
    const floor = resolvePdmNodeMetrics(PDM_RADIUS_MIN)
    expect(floor.nodeW + floor.colGap).toBeGreaterThan(floor.nodeW)
  })
})
