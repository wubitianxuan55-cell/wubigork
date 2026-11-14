/**
 * schedule/timescale.test.ts — 时标刻度内核单测
 *
 * 覆盖 `进度计划/gaea-schedule-zpert8-timescale-re-20261114.md` 的落地口径：
 *  §2 六档间隔集 · §3.1/§3.2 单位编码与索引钳位 · §4.3 自动联动（含 ×0.5 收尾与
 *  0.04..0.4 钳位）· §6/§8.4 像素换算唯一出口 · §7.3 刻度线疏密 · W4 双标尺分段标签。
 * 纯函数、无 DOM（本文件走 jsdom 项目也不碰 DOM）。
 */
import { describe, expect, it } from 'vitest'
import {
  DEFAULT_FV,
  FOV_INTERVALS,
  GANTT_RULER_H,
  GANTT_RULER_ROWS,
  GANTT_RULER_ROW_H,
  SCALE_MAX,
  SCALE_MIN,
  UNIT_INTERVALS,
  WORK_HOURS_PER_DAY,
  applyAutoLink,
  buildMonthRuns,
  buildRulerModel,
  buildWeekRuns,
  buildYearRuns,
  clampInterval,
  clampScale,
  computeAutoLink,
  defaultTimeScale,
  fov6StepFromDensity,
  intervalIndex,
  intervalsForFov,
  intervalsForUnit,
  monthLabelForWidth,
  rawAutoScale,
  refillStep,
  resetStep,
  resolveLineDensity,
  rulerPositions,
  rulerTotalWidth,
  setUnit,
  stepWidthPx,
  tickWidthPx,
  type RulerColumn,
} from './timescale'

/** 造 N 个自然日列（自 startIso 起，周末标 offWork） */
function cols(startIso: string, n: number): RulerColumn[] {
  const start = new Date(`${startIso}T00:00:00Z`).getTime()
  return Array.from({ length: n }, (_, i) => {
    const d = new Date(start + i * 86400000)
    const wd = d.getUTCDay()
    return { d, iso: d.toISOString().slice(0, 10), offWork: wd === 0 || wd === 6 }
  })
}

describe('timescale · 单位与间隔集（RE §2 / §3.1）', () => {
  it('单位→间隔集逐项对齐：年-月{1,2,3,4,6}、月-天{1..7,10,15}、工程周/工程月{1}、6={1,2,3,6}', () => {
    expect(UNIT_INTERVALS[1]).toEqual([1, 2, 3, 4, 6])
    expect(UNIT_INTERVALS[2]).toEqual([1, 2, 3, 4, 5, 6, 7, 10, 15])
    expect(UNIT_INTERVALS[3]).toEqual([1])
    expect(UNIT_INTERVALS[4]).toEqual([1])
    expect(UNIT_INTERVALS[6]).toEqual([1, 2, 3, 6])
    // 8/9/10 = 月-天口径的 10 日 / 15 日 / 月
    expect(UNIT_INTERVALS[8]).toEqual([10])
    expect(UNIT_INTERVALS[9]).toEqual([15])
    expect(UNIT_INTERVALS[10]).toEqual([1])
    expect(intervalsForUnit(3)).toEqual([1])
  })

  it('六档→间隔集：fov1/2/6 按 §2，fov4/5 只有 {1}，fov3 取月-天内 2/4/5', () => {
    expect(FOV_INTERVALS[1]).toEqual([1, 2, 3, 4, 6])
    expect(FOV_INTERVALS[2]).toEqual([1, 2, 3, 4, 5, 6, 7, 10, 15])
    expect(FOV_INTERVALS[6]).toEqual([1, 2, 3, 6])
    expect(FOV_INTERVALS[4]).toEqual([1])
    expect(FOV_INTERVALS[5]).toEqual([1])
    expect(FOV_INTERVALS[3]).toEqual([2, 4, 5])
    expect(intervalsForFov(3)).toEqual([2, 4, 5])
  })

  it('间隔索引钳位（RE §3.2）：集合外一律钳到最近的下界', () => {
    expect(clampInterval(2, 8)).toBe(7) // 月-天无 8
    expect(clampInterval(2, 9)).toBe(7)
    expect(clampInterval(2, 10)).toBe(10)
    expect(clampInterval(2, 16)).toBe(15)
    expect(clampInterval(2, 0)).toBe(1)
    expect(clampInterval(1, 5)).toBe(4) // 年-月无 5
    expect(clampInterval(6, 9)).toBe(6) // fov=6 分支：step<4?…:sel=3
    expect(clampInterval(3, 5)).toBe(1) // 工程周只有 1
    expect(intervalIndex(2, 10)).toBe(7) // stepIdx=7（§4.3 fov=2 分支）
    expect(intervalIndex(2, 15)).toBe(8)
  })

  it('改单位：先归零再按新集合回填首项，fov 随口径重算（RE §1 / §2）', () => {
    expect(resetStep()).toBe(0) // 中间态恒 0
    expect(refillStep(1)).toBe(1)
    expect(refillStep(4)).toBe(1)
    // 原 topStep=15（月-天）→ 切年-月：归零后回填首项 1，不是钳位后的 4
    const cfg = { ...defaultTimeScale(), topUnit: 2 as const, topStep: 15, fov: 3 as const }
    const next = setUnit(cfg, 'top', 1)
    expect(next.topUnit).toBe(1)
    expect(next.topStep).toBe(1)
    expect(next.step).toBe(1)
    expect(next.fov).toBe(3)
    // 切进度标尺到工程周 → fov=4（工作周档）
    const bottom = setUnit(cfg, 'bottom', 3)
    expect(bottom.bottomUnit).toBe(3)
    expect(bottom.bottomStep).toBe(1)
    expect(bottom.fov).toBe(4)
    // 传进来的是新对象（不改原配置）
    expect(cfg.topStep).toBe(15)
  })
})

describe('timescale · 自动联动（RE §4.3 / §8.3）', () => {
  it('fov=1：step 越大 scale 越小，且都在 0.04..0.4 内（含 ×0.5 收尾）', () => {
    const s1 = rawAutoScale(1, 1)!
    const s6 = rawAutoScale(1, 6)!
    // (0.10/2.4/(step+1) + 2.4/15) × 0.5
    expect(s1).toBeCloseTo((0.1 / 2.4 / 2 + 2.4 / 15) * 0.5, 6)
    expect(s6).toBeCloseTo((0.1 / 2.4 / 5 + 2.4 / 15) * 0.5, 6)
    expect(s1).toBeGreaterThan(s6)
    for (const step of [1, 2, 3, 4, 6]) {
      const v = computeAutoLink({ fov: 1, step, minStep: 1, scale: SCALE_MIN }).scale
      expect(v).toBeGreaterThanOrEqual(SCALE_MIN)
      expect(v).toBeLessThanOrEqual(SCALE_MAX)
    }
  })

  it('fov=2：首档原始值为负（1-2.4/15/0.08），钳到下限 0.04；中档钳到上限 0.4', () => {
    expect(rawAutoScale(2, 1)).toBeCloseTo((1 - 2.4 / 15 / 0.08) * 0.5, 6) // = -0.5
    expect(computeAutoLink({ fov: 2, step: 1, minStep: 1, scale: 0.2 }).scale).toBe(SCALE_MIN)
    expect(computeAutoLink({ fov: 2, step: 2, minStep: 1, scale: 0.2 }).scale).toBe(SCALE_MAX)
    // idx=7/8（10 日 / 15 日）又落回下限
    expect(computeAutoLink({ fov: 2, step: 10, minStep: 1, scale: 0.2 }).scale).toBe(SCALE_MIN)
    expect(computeAutoLink({ fov: 2, step: 15, minStep: 1, scale: 0.2 }).scale).toBe(SCALE_MIN)
  })

  it('fov=4/5：单分支原始值同为负，落到 0.04；fov=3/6 不重算 scale（RE 无 scale 赋值分支）', () => {
    expect(computeAutoLink({ fov: 4, step: 1, minStep: 1, scale: 0.3 }).scale).toBe(SCALE_MIN)
    expect(computeAutoLink({ fov: 5, step: 1, minStep: 1, scale: 0.3 }).scale).toBe(SCALE_MIN)
    const f3 = computeAutoLink({ fov: 3, step: 2, minStep: 1, scale: 0.17 })
    expect(f3.scale).toBe(0.17)
    expect(f3.scaleChanged).toBe(false)
    const f6 = computeAutoLink({ fov: 6, step: 3, minStep: 1, scale: 0.22 })
    expect(f6.scale).toBe(0.22)
    expect(f6.scaleChanged).toBe(false)
  })

  it('fov=6：step = max(step, minStep) 后钳进 {1,2,3,6}；密度分档给首档推荐值', () => {
    expect(computeAutoLink({ fov: 6, step: 1, minStep: 2, scale: 0.1 }).step).toBe(2) // min_step 抬到 2
    expect(computeAutoLink({ fov: 6, step: 5, minStep: 1, scale: 0.1 }).step).toBe(3) // 5 不在集合 → 3
    expect(computeAutoLink({ fov: 6, step: 1, minStep: 4, scale: 0.1 }).step).toBe(3) // 4 不在集合 → 3
    expect(fov6StepFromDensity(0.08)).toBe(1)
    expect(fov6StepFromDensity(0.12)).toBe(2)
    expect(fov6StepFromDensity(0.2)).toBe(3)
    expect(fov6StepFromDensity(0.3)).toBe(6)
    expect(fov6StepFromDensity()).toBe(3) // 缺省密度 = 2.4/15 = 0.16
  })

  it('clampScale 边界与非法值；applyAutoLink 把结果写回 fv[fov-1]', () => {
    expect(clampScale(0.01)).toBe(SCALE_MIN)
    expect(clampScale(0.9)).toBe(SCALE_MAX)
    expect(clampScale(0.2)).toBe(0.2)
    expect(clampScale(Number.NaN)).toBe(SCALE_MIN)
    const cfg = { ...defaultTimeScale(), fov: 1 as const, step: 1, scale: 0.2 }
    const linked = applyAutoLink(cfg)
    expect(linked.scale).toBeCloseTo((0.1 / 2.4 / 2 + 2.4 / 15) * 0.5, 6)
    expect(linked.fv[0]).toBeCloseTo(linked.scale, 10)
    expect(linked.fv).toHaveLength(6)
  })

  it('六档缺省比例 fv[6]：长度 6、全在 0.04..0.4、缺省配置 lineDensity=small', () => {
    expect(DEFAULT_FV).toHaveLength(6)
    for (const v of DEFAULT_FV) {
      expect(v).toBeGreaterThanOrEqual(SCALE_MIN)
      expect(v).toBeLessThanOrEqual(SCALE_MAX)
    }
    const cfg = defaultTimeScale()
    expect(cfg.fv).toHaveLength(6)
    expect(cfg.lineDensity).toBe('small')
    expect(cfg.localZoom.length).toBeGreaterThan(0)
  })
})

describe('timescale · 像素换算唯一出口（RE §6 / §8.4）', () => {
  it('1 刻度像素宽 = scale × 2.4；单段 = step × scale × 2.4', () => {
    expect(tickWidthPx(0.2)).toBeCloseTo(0.2 * WORK_HOURS_PER_DAY, 10)
    expect(stepWidthPx(3, 0.2)).toBeCloseTo(3 * 0.2 * 2.4, 10)
    expect(stepWidthPx(0, 0.2)).toBe(0)
  })

  it('pos[i] = Σ step[k] × scale × 2.4，逐元素整数截断（同 FUN_1402c4b40）', () => {
    const pos = rulerPositions([3, 3, 2], 0.2)
    expect(pos[0]).toBe(0)
    expect(pos[1]).toBe(1) // trunc(1.44)
    expect(pos[2]).toBe(2) // trunc(2.88)
    expect(rulerTotalWidth([3, 3, 2], 0.2)).toBeCloseTo(8 * 0.2 * 2.4, 10)
    // 边界：空表 / 零比例 / 上限比例都不返回负数
    expect(rulerPositions([], 0.2)).toEqual([])
    expect(rulerPositions([1, 1], 0)).toEqual([0, 0])
    const max = rulerPositions([1, 2, 3], SCALE_MAX)
    expect(max.every((v) => v >= 0)).toBe(true)
    expect(max[2]).toBeGreaterThan(max[1])
  })
})

describe('timescale · 刻度线疏密（RE §7.3）', () => {
  it('tickLines 三档映射 + 工程日历线开关；缺省 small', () => {
    expect(resolveLineDensity(undefined)).toBe('small')
    expect(resolveLineDensity(null)).toBe('small')
    expect(resolveLineDensity({})).toBe('small')
    expect(resolveLineDensity({ tickLines: 'none' })).toBe('none')
    expect(resolveLineDensity({ tickLines: 'small' })).toBe('small')
    expect(resolveLineDensity({ tickLines: 'large' })).toBe('big')
    expect(resolveLineDensity({ tickLines: 'small' }, true)).toBe('project')
    expect(resolveLineDensity({ tickLines: 'none' }, true)).toBe('none') // 不画优先
    expect(resolveLineDensity({ tickLines: 'large' }, true)).toBe('big')
  })
})

describe('timescale · 分段标签（W4 双标尺）', () => {
  it('年-月跨列段：连续同月合成、跨月/跨年断开', () => {
    const months = buildMonthRuns(cols('2026-09-29', 6)) // 9/29-9/30 + 10/1-10/4
    expect(months).toEqual([{ y: 2026, m: 8, count: 2 }, { y: 2026, m: 9, count: 4 }])
    const years = buildYearRuns(cols('2026-12-30', 5))
    expect(years).toEqual([{ y: 2026, count: 2 }, { y: 2027, count: 3 }])
  })

  it('月段文本降级阶梯：≥76 年月 / ≥44 月 / ≥14 数字 / 窄窗不写数字', () => {
    expect(monthLabelForWidth(2026, 8, 80)).toBe('2026年9月')
    expect(monthLabelForWidth(2026, 8, 50)).toBe('9月')
    expect(monthLabelForWidth(2026, 8, 20)).toBe('9')
    expect(monthLabelForWidth(2026, 8, 10)).toBe('')
    expect(monthLabelForWidth(2026, 8, 20, true)).toBe('') // 窄窗只画月段文本
  })

  it('周段：周一起始跨月不断开，W1 起每 4 周标一次（段宽 ≥12px）', () => {
    const weeks = buildWeekRuns(cols('2026-09-07', 21), 20) // 2026-09-07 是周一
    expect(weeks).toHaveLength(3)
    expect(weeks.map((w) => w.weekNo)).toEqual([1, 2, 3])
    expect(weeks.map((w) => w.label)).toEqual(['W1', '', ''])
    expect(weeks[0].title).toBe('第 1 周（9/7 ~ 9/13）')
    expect(weeks[0].count).toBe(7)
    // 段太窄（7 天 × 1px）→ 不写周序号
    expect(buildWeekRuns(cols('2026-09-07', 21), 1).every((w) => w.label === '')).toBe(true)
  })

  it('首周非周一起始：周序号按段递增，不再出现相邻重复 W1', () => {
    // 2026-09-09 是周三：首段 5 天（三~日），次段周一起
    const weeks = buildWeekRuns(cols('2026-09-09', 14), 20, true)
    expect(weeks.length).toBeGreaterThanOrEqual(2)
    expect(weeks[0].weekNo).toBe(1)
    expect(weeks[1].weekNo).toBe(2)
    expect(weeks[0].label).toBe('W1')
    expect(weeks[1].label).toBe('W2')
  })

  it('窄窗段宽不足 24px：退回每 4 周标一次，避免 W 标签叠字', () => {
    const weeks = buildWeekRuns(cols('2026-09-07', 70), 2.6, true)
    const labels = weeks.map((w) => w.label).filter(Boolean)
    expect(labels.length).toBeLessThan(weeks.length)
    expect(labels[0]).toBe('W1')
    // 只标 W1 / W5 / W9…（4 周步长），不会每周都写
    expect(labels.every((l) => Number(l.slice(1)) % 4 === 1)).toBe(true)
  })

  it('窄窗周序号逐段标注（无 4 周步长）', () => {
    const weeks = buildWeekRuns(cols('2026-09-07', 21), 6, true)
    expect(weeks.map((w) => w.label)).toEqual(['W1', 'W2', 'W3'])
  })

  it('双标尺模型：两行常驻、上行年月段+刻度线、下行周/星期', () => {
    const model = buildRulerModel({ colDates: cols('2026-09-07', 21), dayW: 20, lineDensity: 'small' })
    expect(model.heightPx).toBe(GANTT_RULER_H)
    expect(model.heightPx).toBe(GANTT_RULER_ROW_H * GANTT_RULER_ROWS)
    expect(GANTT_RULER_H).toBe(40) // 与表格表头（GanttTable）/覆盖层 40 同值，改动需三处同步
    expect(model.narrow).toBe(false)
    expect(model.engineering.segments).toHaveLength(1)
    expect(model.engineering.segments[0].widthPx).toBe(21 * 20)
    expect(model.engineering.ticks.length).toBeGreaterThan(0) // small = 月界 + 周界
    expect(model.progress.tier).toBe('day')
    expect(model.progress.days).toHaveLength(21)
    expect(model.progress.days[0].label).toBe('7')
    expect(model.progress.days[0].weekday).toBe('') // dayW=20 < 22 → 星期退到 title
    expect(model.progress.days[0].title).toBe('周一')
  })

  it('日层：日宽 ≥22 才并列星期；非工作日标记来自列数据', () => {
    const model = buildRulerModel({ colDates: cols('2026-09-07', 14), dayW: 22 })
    expect(model.progress.days[0].weekday).toBe('一')
    const sat = model.progress.days[5] // 2026-09-12 周六
    expect(sat.offWork).toBe(true)
    expect(sat.title).toBe('非工作日')
    expect(sat.weekday).toBe('六')
  })

  it('周层：dayW<12 由日切周；窄窗（<8）只留月段文本与周序号、不画刻度线', () => {
    const wide = buildRulerModel({ colDates: cols('2026-09-28', 14), dayW: 11 })
    expect(wide.progress.tier).toBe('week')
    expect(wide.progress.days).toHaveLength(0)
    expect(wide.progress.weeks.length).toBeGreaterThan(0)
    expect(wide.narrow).toBe(false)
    // 窄窗：9 月只剩 3 列（18px）→ 月段文本为空；刻度线整行清空
    const narrow = buildRulerModel({ colDates: cols('2026-09-28', 14), dayW: 6 })
    expect(narrow.narrow).toBe(true)
    expect(narrow.engineering.narrow).toBe(true)
    expect(narrow.engineering.ticks).toEqual([])
    expect(narrow.engineering.segments[0].label).toBe('')
    expect(narrow.engineering.segments[0].widthPx).toBe(3 * 6)
    expect(narrow.engineering.segments[1].label).toBe('10月')
    expect(narrow.progress.weeks.every((w) => w.label.startsWith('W'))).toBe(true)
  })

  it('线密度 none 不产刻度线；big 只画年/月界', () => {
    const c = cols('2026-09-28', 40)
    expect(buildRulerModel({ colDates: c, dayW: 20, lineDensity: 'none' }).engineering.ticks).toEqual([])
    const big = buildRulerModel({ colDates: c, dayW: 20, lineDensity: 'big' }).engineering.ticks
    expect(big.length).toBeGreaterThan(0)
    expect(big.every((t) => t.tier === 'month' || t.tier === 'year')).toBe(true)
    const small = buildRulerModel({ colDates: c, dayW: 20, lineDensity: 'small' }).engineering.ticks
    expect(small.some((t) => t.tier === 'week')).toBe(true)
    expect(small.length).toBeGreaterThan(big.length) // 小刻度线=更密
    // 首列不画边界线（x=0 与标尺左沿重合）
    expect(small.every((t) => t.xPx > 0)).toBe(true)
  })

  it('今日/竣工竖线横坐标：今日=列位×日宽，竣工=列右缘（与覆盖层口径一致）', () => {
    const model = buildRulerModel({ colDates: cols('2026-09-07', 21), dayW: 20, todayCol: 2, deadlineCol: 5 })
    expect(model.todayX).toBe(40)
    expect(model.deadlineX).toBe(5 * 20 + 20)
    const none = buildRulerModel({ colDates: cols('2026-09-07', 21), dayW: 20, todayCol: null, deadlineCol: null })
    expect(none.todayX).toBeNull()
    expect(none.deadlineX).toBeNull()
  })
})
