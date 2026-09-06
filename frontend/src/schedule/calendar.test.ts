import { describe, expect, it } from 'vitest'
import { DEFAULT_CALENDAR, dateToWd, deadlineWorkdays, isWorkingDate, normalizeCalendar, wdToDate } from './calendar'
import type { SchedCalendar } from './types'

// 2026-09-07 是周一；2026-09-12 周六、09-13 周日
const MON = '2026-09-07'
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
