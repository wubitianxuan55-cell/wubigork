/**
 * schedule/networkDisplay.test.ts — 图面内容四开关与几何参数解析契约（ZPert W4）
 *
 * 钉住三件事（两个网络视图都靠这份解析，任何一条漂移都会让 AOA/PDM 观感不一致）：
 *  1. 缺省值 = 斑马首装口径（节点时间开 / 工期开 / 六参数关 / 起止时间关）；
 *  2. 显式 `false` 必须原样保留（不能用 `?? 缺省` 把用户的关当成没填）；
 *  3. 几何参数越界（0/负数/NaN/超上限）回落缺省，边界值本身合法保留。
 */
import { describe, expect, it } from 'vitest'
import {
  NETWORK_DISPLAY_DEFAULTS,
  NETWORK_GEOMETRY_DEFAULTS,
  resolveNetworkDisplay,
  resolveNetworkGeometry,
  sixParaLines,
} from './networkDisplay'
import type { SchedNetworkStyle } from './types'

/** 脏值构造（模拟手写 JSON / 旧文件里混进来的非法类型） */
const dirty = (patch: Record<string, unknown>): SchedNetworkStyle => patch as unknown as SchedNetworkStyle

describe('resolveNetworkDisplay：四开关解析', () => {
  it('缺省（undefined / null / 空对象）→ 首装口径', () => {
    const want = { showDotTime: true, showDuration: true, showSixPara: false, showStartStopTime: false }
    expect(resolveNetworkDisplay()).toEqual(want)
    expect(resolveNetworkDisplay(null)).toEqual(want)
    expect(resolveNetworkDisplay({})).toEqual(want)
    expect(NETWORK_DISPLAY_DEFAULTS).toEqual(want)
  })

  it('显式 false 全部保留（不回落缺省）', () => {
    const flags = resolveNetworkDisplay({
      showDotTime: false,
      showDuration: false,
      showSixPara: false,
      showStartStopTime: false,
    })
    expect(flags).toEqual({ showDotTime: false, showDuration: false, showSixPara: false, showStartStopTime: false })
  })

  it('显式 true 打开默认关闭项；未给的键仍走缺省（混合口径）', () => {
    expect(resolveNetworkDisplay({ showSixPara: true })).toEqual({
      showDotTime: true,
      showDuration: true,
      showSixPara: true,
      showStartStopTime: false,
    })
    expect(resolveNetworkDisplay({ showStartStopTime: true, showDuration: false })).toEqual({
      showDotTime: true,
      showDuration: false,
      showSixPara: false,
      showStartStopTime: true,
    })
  })

  it('非布尔脏值（字符串/数字/null）按未填处理，回落缺省', () => {
    expect(resolveNetworkDisplay(dirty({ showDotTime: 'false', showDuration: 0, showSixPara: null, showStartStopTime: 1 }))).toEqual(
      NETWORK_DISPLAY_DEFAULTS,
    )
  })
})

describe('resolveNetworkGeometry：几何参数解析', () => {
  it('缺省 → AOA 现行渲染口径（圆形/实心/16px/9px/1.5px）', () => {
    const want = { dotStyle: 'circle', arrowStyle: 'solid', namePos: 'auto', nodeRadius: 16, headHeight: 9, lineWidth: 1.5 }
    expect(resolveNetworkGeometry()).toEqual(want)
    expect(resolveNetworkGeometry(null)).toEqual(want)
    expect(resolveNetworkGeometry({})).toEqual(want)
    expect(NETWORK_GEOMETRY_DEFAULTS).toEqual(want)
  })

  it('越界数值一律回落缺省（0 / 负数 / 超上限 / NaN / Infinity）', () => {
    const geom = resolveNetworkGeometry(
      dirty({ nodeRadius: 0, headHeight: 61, lineWidth: 9, dotStyle: undefined, arrowStyle: undefined, namePos: undefined }),
    )
    expect(geom.nodeRadius).toBe(16)
    expect(geom.headHeight).toBe(9)
    expect(geom.lineWidth).toBe(1.5)

    expect(resolveNetworkGeometry(dirty({ nodeRadius: -8 })).nodeRadius).toBe(16)
    expect(resolveNetworkGeometry(dirty({ nodeRadius: 500 })).nodeRadius).toBe(16)
    expect(resolveNetworkGeometry(dirty({ nodeRadius: Number.NaN })).nodeRadius).toBe(16)
    expect(resolveNetworkGeometry(dirty({ headHeight: Number.POSITIVE_INFINITY })).headHeight).toBe(9)
    expect(resolveNetworkGeometry(dirty({ lineWidth: 0.1 })).lineWidth).toBe(1.5)
  })

  it('合法边界值原样保留（4/200、2/60、0.5/8）', () => {
    expect(resolveNetworkGeometry({ nodeRadius: 4 }).nodeRadius).toBe(4)
    expect(resolveNetworkGeometry({ nodeRadius: 200 }).nodeRadius).toBe(200)
    expect(resolveNetworkGeometry({ headHeight: 2 }).headHeight).toBe(2)
    expect(resolveNetworkGeometry({ headHeight: 60 }).headHeight).toBe(60)
    expect(resolveNetworkGeometry({ lineWidth: 0.5 }).lineWidth).toBe(0.5)
    expect(resolveNetworkGeometry({ lineWidth: 8 }).lineWidth).toBe(8)
  })

  it('枚举参数透传，缺省回落 circle / solid / auto', () => {
    const geom = resolveNetworkGeometry({ dotStyle: 'rect', arrowStyle: 'hollow', namePos: 'vertical' })
    expect([geom.dotStyle, geom.arrowStyle, geom.namePos]).toEqual(['rect', 'hollow', 'vertical'])
    expect(resolveNetworkGeometry({ dotStyle: 'rect' }).arrowStyle).toBe('solid')
  })
})

describe('sixParaLines：六参数文案顺序', () => {
  it('固定 ES → EF → LS → LF → TF → FF，且带前缀标签', () => {
    const lines = sixParaLines({ es: '1', ef: '4', ls: '2', lf: '5', tf: '1', ff: '0' })
    expect(lines).toEqual(['ES 1', 'EF 4', 'LS 2', 'LF 5', 'TF 1', 'FF 0'])
    expect(lines).toHaveLength(6)
    // 顺序即教材/斑马标注口径：最早在前、最迟在后、时差收尾
    expect(lines.map((l) => l.split(' ')[0])).toEqual(['ES', 'EF', 'LS', 'LF', 'TF', 'FF'])
  })
})
