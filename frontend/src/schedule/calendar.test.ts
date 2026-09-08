import { describe, expect, it } from 'vitest'
import { DEFAULT_CALENDAR, cdEarliestStart, cdLatestStart, cdToEf, dateToWd, deadlineWorkdays, isWorkingDate, normalizeCalendar, wdToDate } from './calendar'
import type { SchedCalendar } from './types'

// 2026-09-07 是周一；2026-09-12 周六、09-13 周日
const MON = '2026-09-07'
// 2026-01-05 也是周一（镜像锚点表专用锚：1 月无节假日，跨周末节奏干净）
const MON1 = '2026-01-05'
const custom: SchedCalendar = { workweek: [1, 2, 3, 4, 6], holidays: ['2026-10-01', '2026-10-02'] } // 周日休，周六上班

describe('isWorkingDate / normalizeCalendar', () => {
  it('默认周一~五上班，周末休', () => {
    expect(isWorkingDate(new Date('2026-09-07'), DEFAULT_CALENDAR)).toBe(true) // 周一
    expect(isWorkingDate(new Date('2026-09-12'), DEFAULT_CALENDAR)).toBe(false) // 周六
    expect(isWorkingDate(new Date('2026-09-13'), DEFAULT_CALENDAR)).toBe(false) // 周日
  })
  it('节假日例外命中即非工作日', () => {
    expect(isWorkingDate(new Date('2026-10-01'), custom)).toBe(false) // 周四但国庆
    expect(isWorkingDate(new Date('2026-09-12'), custom)).toBe(true) // 周六上班
  })
  it('normalize 兜底：非法/空日历回默认，去重去非法值', () => {
    expect(normalizeCalendar(undefined)).toEqual(DEFAULT_CALENDAR)
    expect(normalizeCalendar({ workweek: [9, 1, 1], holidays: ['bad', '2026-10-01'] })).toEqual({
      workweek: [1], holidays: ['2026-10-01'],
    })
  })
})

describe('wdToDate（工作日序号 → 日期）', () => {
  it('默认工作制：第 0 个=开工日，第 5 个跨过周末=下周一', () => {
    expect(wdToDate(MON, 0).toISOString().slice(0, 10)).toBe('2026-09-07')
    expect(wdToDate(MON, 4).toISOString().slice(0, 10)).toBe('2026-09-11') // 周五
    expect(wdToDate(MON, 5).toISOString().slice(0, 10)).toBe('2026-09-14') // 跳过周末
  })
  it('开工日为非工作日时顺延', () => {
    expect(wdToDate('2026-09-12', 0).toISOString().slice(0, 10)).toBe('2026-09-14') // 周六开工顺延周一
    expect(wdToDate('2026-09-12', 1).toISOString().slice(0, 10)).toBe('2026-09-15')
  })
  it('节假日跳过；自定义工作制生效', () => {
    // 自定义：周六上班，10-01/02 休。2026-09-28 周一 起：9/28,29,30,10/3(跳过10/1,2)
    expect(wdToDate('2026-09-28', 3, custom).toISOString().slice(0, 10)).toBe('2026-10-03')
  })
})

describe('dateToWd（日期 → 工作日序号）', () => {
  it('工作日返回序号，非工作日返回 null', () => {
    expect(dateToWd(MON, '2026-09-07')).toBe(0)
    expect(dateToWd(MON, '2026-09-11')).toBe(4)
    expect(dateToWd(MON, '2026-09-12')).toBeNull()
    expect(dateToWd(MON, '2026-09-14')).toBe(5)
  })
  it('与 wdToDate 互逆（工作日内）', () => {
    for (let i = 0; i < 30; i++) {
      const d = wdToDate(MON, i)
      expect(dateToWd(MON, d.toISOString().slice(0, 10))).toBe(i)
    }
  })
})

describe('deadlineWorkdays（目标竣工 → 目标总工期，v4.117 刀8）', () => {
  it('工作日竣工含当日：周一开工、周五竣工=5 个工作日', () => {
    expect(deadlineWorkdays(MON, '2026-09-11')).toBe(5)
    expect(deadlineWorkdays(MON, '2026-09-14')).toBe(6) // 跨周末到下周一
  })
  it('竣工日为非工作日回落到此前最近工作日', () => {
    expect(deadlineWorkdays(MON, '2026-09-12')).toBe(5) // 周六 → 记周五 5 个
  })
  it('节假日命中不计；竣工早于开工 → 0', () => {
    expect(deadlineWorkdays(MON, '2026-09-18')).toBe(10) // 两个完整工作周
    expect(deadlineWorkdays(MON, '2026-09-18', { workweek: [1, 2, 3, 4, 5], holidays: ['2026-09-11'] })).toBe(9)
    expect(deadlineWorkdays('2026-09-10', '2026-09-09')).toBe(0)
  })
})

describe('cdToEf / cdLatestStart（双工期刀1：日历天换算，镜像 Go CdToEf/CdLatestStart）', () => {
  it('28cd 周一起：边界=4 周后周一，等效工作日跨度 20（28wd 误录=失真 +8）', () => {
    expect(cdToEf(MON, 0, 28)).toBe(20)
    expect(cdToEf(MON, 0, 1)).toBe(1)
    expect(cdToEf(MON, 0, 0)).toBe(0) // cd=0 → es
  })
  it('ceil 吸附：26/27/28cd 边界落周末 → 同一 ef=20（吸附折叠）', () => {
    expect(cdToEf(MON, 0, 26)).toBe(20) // 边界周六
    expect(cdToEf(MON, 0, 27)).toBe(20) // 边界周日
    expect(cdToEf(MON, 0, 28)).toBe(20) // 边界恰为周一工作日
  })
  it('es>0 锚点随行：周四起 7cd，跨度 5 个工作日', () => {
    expect(cdToEf(MON, 3, 7)).toBe(8) // 09-10(周四)+7 自然日=09-17(周四)
  })
  it('节假日窗口：边界命中节假日顺延其后首个工作日', () => {
    const cal: SchedCalendar = { workweek: [1, 2, 3, 4, 5], holidays: ['2026-09-16'] }
    expect(cdToEf(MON, 0, 10, cal)).toBe(7) // 边界 09-17，其前一节假日 09-16 不计入索引
    expect(cdToEf(MON, 0, 10)).toBe(8) // 无节假日对照：边界 09-17 即索引 8
  })
  it('开工日为非工作日：锚点顺延后起算', () => {
    expect(cdToEf('2026-09-12', 0, 3)).toBe(3) // 周六开工顺延周一 09-14，+3=周四
  })
  it('cdLatestStart：fwd(s)≤lf<fwd(s+1) 性质钉死', () => {
    expect(cdLatestStart(MON, 20, 28)).toBe(0) // fwd(0)=20≤20<fwd(1)=21
    expect(cdLatestStart(MON, 25, 28)).toBe(5) // fwd(5)=25≤25<fwd(6)=26
  })
  it('平段吸附：26cd 时 fwd(0)=fwd(1)=fwd(2)=20，逆推取最大 s=2', () => {
    expect(cdLatestStart(MON, 20, 26)).toBe(2)
  })
  it('lf 过小（负时差极端）：下限截 0，两侧镜像一致', () => {
    expect(cdLatestStart(MON, 0, 28)).toBe(0)
    expect(cdLatestStart(MON, 19, 28)).toBe(0) // 锚点回退越过开工日
  })
})

describe('镜像锚点表（v4.155 全搭接放开：2026-01-05 周一锚、周一~五无节假日，与 Go 侧同表逐字钉死）', () => {
  it('cdToEf(s,7)：s=0..10 平移不吸附', () => {
    expect([0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map((s) => cdToEf(MON1, s, 7))).toEqual([5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15])
  })
  it('cdToEf(s,5)：平台 0-2→5、5-7→10（ceil 吸附折叠）', () => {
    expect([0, 1, 2, 3, 4, 5, 6, 7, 8].map((s) => cdToEf(MON1, s, 5))).toEqual([5, 5, 5, 6, 7, 10, 10, 10, 11])
  })
  it('cdEarliestStart(target,7)：最小 s 使 fwd(s)≥target', () => {
    const table: Array<[number, number]> = [[5, 0], [6, 1], [9, 4], [10, 5], [11, 6], [12, 7]]
    for (const [target, want] of table) expect(cdEarliestStart(MON1, target, 7)).toBe(want)
  })
  it('cdEarliestStart(target,5)：吸附平段跳到平台右端', () => {
    const table: Array<[number, number]> = [[5, 0], [6, 3], [10, 5], [11, 8]]
    for (const [target, want] of table) expect(cdEarliestStart(MON1, target, 5)).toBe(want)
  })
  it('cdLatestStart(lf,7)（回归对照）：fwd(s)≤lf<fwd(s+1)', () => {
    const table: Array<[number, number]> = [[5, 0], [8, 3], [9, 4], [10, 5], [12, 7], [15, 10]]
    for (const [lf, want] of table) expect(cdLatestStart(MON1, lf, 7)).toBe(want)
  })
  it('cdLatestStart(lf,5)（回归对照）：平段取最大 s', () => {
    const table: Array<[number, number]> = [[5, 2], [6, 3], [10, 7]]
    for (const [lf, want] of table) expect(cdLatestStart(MON1, lf, 5)).toBe(want)
  })
  it('cdEarliestStart 最小性性质：fwd(res)≥target 且 fwd(res−1)<target', () => {
    for (const cd of [5, 7, 26]) {
      for (let target = 1; target <= 30; target++) {
        const res = cdEarliestStart(MON1, target, cd)
        expect(cdToEf(MON1, res, cd)).toBeGreaterThanOrEqual(target)
        if (res > 0) expect(cdToEf(MON1, res - 1, cd)).toBeLessThan(target)
      }
    }
  })
  it('target≤0 → 0（防呆与 cdLatestStart 同款）', () => {
    expect(cdEarliestStart(MON1, 0, 7)).toBe(0)
    expect(cdEarliestStart(MON1, -3, 7)).toBe(0)
  })
})
