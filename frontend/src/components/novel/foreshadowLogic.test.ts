// foreshadowLogic.test.ts — 伏笔登记表纯逻辑单测（ID 生成/状态机/章节文件名/载荷收窄）
import { describe, expect, it } from 'vitest'
import {
  advanceForeshadowStatus,
  buildManualForeshadow,
  formatPlantedIn,
  foreshadowFlowLabel,
  makeManualForeshadowID,
  normalizeForeshadowItems,
} from './foreshadowLogic'

describe('foreshadowLogic 状态流转', () => {
  it('planted→hinted→revealed；revealed 可回退到 hinted（闭环可反复）', () => {
    expect(advanceForeshadowStatus('planted')).toBe('hinted')
    expect(advanceForeshadowStatus('hinted')).toBe('revealed')
    expect(advanceForeshadowStatus('revealed')).toBe('hinted') // 回退
    // 回退后可再次推进到回收
    expect(advanceForeshadowStatus(advanceForeshadowStatus('revealed'))).toBe('revealed')
  })

  it('流转按钮文案按状态给出', () => {
    expect(foreshadowFlowLabel('planted')).toBe('标记暗示')
    expect(foreshadowFlowLabel('hinted')).toBe('标记回收')
    expect(foreshadowFlowLabel('revealed')).toBe('回退到暗示')
  })
})

describe('foreshadowLogic manual ID 与章节文件名', () => {
  it('manual_ 前缀 + 毫秒时间戳（与 AI stable ID 不冲突）', () => {
    expect(makeManualForeshadowID(1725000000000)).toBe('manual_1725000000000')
    expect(makeManualForeshadowID()).toMatch(/^manual_\d+$/)
  })

  it('章节号补零为 NNN.md；非法值回落第 1 章', () => {
    expect(formatPlantedIn(1)).toBe('001.md')
    expect(formatPlantedIn(12)).toBe('012.md')
    expect(formatPlantedIn(123)).toBe('123.md')
    expect(formatPlantedIn(0)).toBe('001.md')
    expect(formatPlantedIn(-3)).toBe('001.md')
    expect(formatPlantedIn(Number.NaN)).toBe('001.md')
  })
})

describe('foreshadowLogic buildManualForeshadow', () => {
  it('登记表单 → 新条目：默认 planted、manual_ ID、planted_in 补零', () => {
    const f = buildManualForeshadow(
      { category: 'plot', description: '神秘铜匣的钥匙', chapterNum: 7, isLongTerm: true },
      1725000000000,
    )
    expect(f).toEqual({
      id: 'manual_1725000000000',
      category: 'plot',
      description: '神秘铜匣的钥匙',
      planted_in: '007.md',
      status: 'planted',
      is_long_term: true,
    })
  })
})

describe('foreshadowLogic normalizeForeshadowItems', () => {
  it('收窄 GetForeshadows 载荷；异常输入回落空数组', () => {
    expect(normalizeForeshadowItems({ items: [{ id: 'a' }] })).toHaveLength(1)
    expect(normalizeForeshadowItems(null)).toEqual([])
    expect(normalizeForeshadowItems(undefined)).toEqual([])
    expect(normalizeForeshadowItems({})).toEqual([])
    expect(normalizeForeshadowItems({ items: 'x' })).toEqual([])
  })
})
