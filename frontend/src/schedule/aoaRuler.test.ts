/**
 * aoaRuler.test.ts — 工程标尺纯函数（v4.161，时间坐标框架）
 *
 * 口径：列=工作日（2026-09-07 周一起）；步进按总工期 5/10/20；月标签落本月
 * 首个工作日且月界竖线不含 wd=0；工程周每 7 列一档档内居中。
 */
import { describe, expect, it } from 'vitest'
import { AOA_COL_W, AOA_MARGIN } from './aoa'
import { buildAoaRuler } from './aoaRuler'

describe('buildAoaRuler', () => {
  it('短计划（9 天）：步进 5、末刻度=总工期、星期行与工程周齐备', () => {
    const r = buildAoaRuler(9, AOA_COL_W, AOA_MARGIN, '2026-09-07')
    expect(r.step).toBe(5)
    expect(r.dayTicks.map((t) => t.label)).toEqual(['0', '5', '9'])
    expect(r.dayTicks.at(-1)?.total).toBe(true)
    // 2026-09-07=周一：首列星期一，第 7 列（wd=7）也是周一
    expect(r.weekdays[0]).toEqual({ x: AOA_MARGIN, label: '一' })
    expect(r.weeks[0]).toEqual({ x: AOA_MARGIN + 3.5 * AOA_COL_W, label: '1' })
    expect(r.weeks.map((w) => w.label)).toEqual(['1', '2']) // 0-6 / 7-9
    // 日行=自然日号 7,8,…（工作日列）
    expect(r.dayLabels[0].label).toBe('7')
  })

  it('跨月：月标签落本月首个工作日，月界竖线不含首列', () => {
    // 2026-09-28（周一）起 10 天：跨 9 月→10 月（10 月首个工作日=10-1 周四）
    const r = buildAoaRuler(10, AOA_COL_W, AOA_MARGIN, '2026-09-28')
    expect(r.monthLabels.map((m) => m.label)).toEqual(['2026.9', '2026.10'])
    expect(r.monthLines).toHaveLength(1)
    expect(r.monthLines[0].x).toBe(AOA_MARGIN + 3 * AOA_COL_W) // wd=3 → 10-1
  })

  it('长计划步进切换：61 天=10、121 天=20', () => {
    expect(buildAoaRuler(60, 110, 40, '2026-09-07').step).toBe(5)
    expect(buildAoaRuler(61, 110, 40, '2026-09-07').step).toBe(10)
    expect(buildAoaRuler(121, 110, 40, '2026-09-07').step).toBe(20)
  })
})
