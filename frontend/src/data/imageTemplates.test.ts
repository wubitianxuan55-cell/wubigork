import { describe, expect, it } from 'vitest'
import {
  CATEGORIES, TEMPLATES, ALL_CATEGORY_ID, getCategory,
} from './imageTemplates'
import { HERDSMAN_CATEGORIES, HERDSMAN_TEMPLATES } from './herdsmanTemplates'

describe('绘梦模板库合并数据', () => {
  it('内置 7 类 + herdsman 12 类，共 231 个模板', () => {
    expect(CATEGORIES.length).toBe(19)
    expect(HERDSMAN_CATEGORIES.length).toBe(12)
    expect(Object.values(HERDSMAN_TEMPLATES).reduce((n, l) => n + l.length, 0)).toBe(152)
    expect(Object.values(TEMPLATES).reduce((n, l) => n + l.length, 0)).toBe(231)
  })

  it('herdsman 分类有中文名，全部模板都有 label/prompt', () => {
    expect(getCategory('portrait')?.label).toBe('人像摄影')
    expect(getCategory('ui')?.label).toBe('UI 界面')
    expect(getCategory('hm-scene')?.label).toBe('场景氛围') // 与内置 scene 冲突时加 hm- 前缀
    expect(ALL_CATEGORY_ID).toBe('all')
    for (const list of Object.values(TEMPLATES)) {
      for (const t of list) {
        expect(t.label).toBeTruthy()
        expect(t.prompt).toBeTruthy()
      }
    }
  })

  it('herdsman 模板带稳定 id 与图标', () => {
    const portrait = HERDSMAN_TEMPLATES.portrait
    expect(portrait.length).toBe(18)
    expect(portrait[0].id).toContain('portrait:')
    expect(portrait[0].icon).toBeTruthy()
  })
})
