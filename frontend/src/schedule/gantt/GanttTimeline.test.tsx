/**
 * schedule/gantt/GanttTimeline.test.tsx — 双标尺渲染（W4）
 *
 * 组件零计算，断言只对着 buildRulerModel 产出的结构：
 *  两行常驻（工程标尺 + 进度标尺）/ 窄窗降级 / 刻度线疏密 / 今日与竣工线穿透两行 /
 *  段宽合计 == chartW（不横向溢出）。
 */
import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { GanttTimeline, type GanttColDate } from './GanttTimeline'
import { GANTT_RULER_H, GANTT_RULER_ROW_H, buildRulerModel, type RulerColumn } from '../timescale'

function cols(startIso: string, n: number): GanttColDate[] {
  const start = new Date(`${startIso}T00:00:00Z`).getTime()
  return Array.from({ length: n }, (_, i) => {
    const d = new Date(start + i * 86400000)
    const wd = d.getUTCDay()
    return { d, iso: d.toISOString().slice(0, 10), offWork: wd === 0 || wd === 6 }
  })
}

function mount(colDates: RulerColumn[], dayW: number, extra: Partial<Parameters<typeof buildRulerModel>[0]> = {}) {
  const chartW = colDates.length * dayW
  const model = buildRulerModel({ colDates, dayW, ...extra })
  const { container } = render(<GanttTimeline chartW={chartW} model={model} />)
  const root = container.querySelector<HTMLElement>('.sched-ruler')!
  return { root, model, chartW }
}

describe('GanttTimeline 双标尺', () => {
  it('两行常驻：上行工程标尺（年-月段 + 日/周刻度线）、下行进度标尺（逐日号）', () => {
    const { root, chartW } = mount(cols('2026-09-07', 21), 20)
    expect(root.className).toContain('sched-gantt-datehead')
    expect(root.style.height).toBe(`${GANTT_RULER_H}px`)
    expect(root.style.width).toBe(`${chartW}px`)
    const rows = root.querySelectorAll('.sched-ruler-row')
    expect(rows).toHaveLength(2)
    expect(rows[0].className).toContain('sched-ruler-engineering')
    expect(rows[1].className).toContain('sched-ruler-progress')
    expect((rows[0] as HTMLElement).style.height).toBe(`${GANTT_RULER_ROW_H}px`)
    // 上行=年月跨列段（含年），下行=日号
    expect(root.querySelectorAll('.sched-ts-month')).toHaveLength(1)
    expect(root.querySelector('.sched-ts-month')!.textContent).toBe('2026年9月')
    expect(root.querySelectorAll('.sched-day-col')).toHaveLength(21)
    expect(root.querySelector('.sched-day-cell')!.textContent).toBe('7')
    // 不横向溢出：月段宽合计 = chartW
    const segW = Array.from(root.querySelectorAll<HTMLElement>('.sched-ts-month'))
      .reduce((s, el) => s + Number.parseFloat(el.style.width), 0)
    expect(segW).toBe(chartW)
  })

  it('刻度线疏密：none 无线、small 含周边界细线、big 只留年/月界，类名随密度切换', () => {
    const none = mount(cols('2026-09-07', 21), 20, { lineDensity: 'none' })
    expect(none.root.querySelectorAll('.sched-ruler-tick')).toHaveLength(0)
    expect(none.root.className).toContain('sched-ruler-density-none')

    const small = mount(cols('2026-09-28', 40), 20, { lineDensity: 'small' })
    const smallTicks = Array.from(small.root.querySelectorAll('.sched-ruler-tick'))
    expect(smallTicks.length).toBeGreaterThan(0)
    expect(smallTicks.some((el) => el.classList.contains('sched-ruler-tick-week'))).toBe(true)

    const big = mount(cols('2026-09-28', 40), 20, { lineDensity: 'big' })
    const bigTicks = Array.from(big.root.querySelectorAll('.sched-ruler-tick'))
    expect(bigTicks.length).toBeGreaterThan(0)
    expect(bigTicks.every((el) => !el.classList.contains('sched-ruler-tick-week'))).toBe(true)
    expect(big.root.className).toContain('sched-ruler-density-big')
  })

  it('窄窗（dayW<8）：上行只画月段文本且无刻度线，下行只画周序号', () => {
    const { root } = mount(cols('2026-09-28', 14), 6)
    expect(root.className).toContain('sched-ruler-narrow')
    expect(root.querySelectorAll('.sched-ruler-tick')).toHaveLength(0)
    // 9 月只剩 3 列（18px）→ 文本降级为空；10 月段仍写「10月」
    const labels = Array.from(root.querySelectorAll('.sched-ts-month')).map((el) => el.textContent)
    expect(labels).toEqual(['', '10月'])
    // 下行切周层，逐段周序号
    expect(root.querySelectorAll('.sched-day-col')).toHaveLength(0)
    const weeks = Array.from(root.querySelectorAll('.sched-ts-week')).map((el) => el.textContent)
    expect(weeks).toEqual(['W1', 'W2'])
  })

  it('今日/竣工竖线穿透两行标尺（左标=列位×日宽；竣工=列右缘）', () => {
    const { root } = mount(cols('2026-09-07', 21), 20, { todayCol: 2, deadlineCol: 5 })
    const today = root.querySelector<HTMLElement>('.sched-ruler-today')!
    const deadline = root.querySelector<HTMLElement>('.sched-ruler-deadline')!
    expect(today.style.left).toBe('40px')
    expect(deadline.style.left).toBe('120px')
    // 两条线都在标尺根节点内（跨两行），不在某一行内
    expect(today.parentElement).toBe(root)
    expect(deadline.parentElement).toBe(root)
  })

  it('行顺序即上下标尺（上=工程标尺、下=进度标尺），日层类名带 tier-day', () => {
    const { root } = mount(cols('2026-09-07', 7), 20)
    const rows = Array.from(root.querySelectorAll<HTMLElement>('.sched-ruler-row'))
    expect(rows.map((r) => r.className)).toEqual([
      'sched-ruler-row sched-ruler-engineering',
      'sched-ts-days sched-ruler-row sched-ruler-progress sched-ruler-tier-day',
    ])
    // 月段 title = 「2026年9月」（悬停即标尺名细目，不占行内文本）
    expect(rows[0].querySelector('.sched-ts-month')!.getAttribute('title')).toBe('2026年9月')
  })
})
