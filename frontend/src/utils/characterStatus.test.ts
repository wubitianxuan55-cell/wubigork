import { describe, expect, it } from 'vitest'
import {
  CHARACTER_STATUSES,
  CHARACTER_STATUS_LABELS,
  CHARACTER_STATUS_OPTIONS,
  DEFAULT_CHARACTER_STATUS,
  characterStatusLabel,
  isCharacterStatus,
  normalizeCharacterStatus,
} from './characterStatus'

describe('characterStatus 角色状态枚举校验（T6-7.5 状态收敛）', () => {
  it('合法枚举值原样通过', () => {
    for (const s of CHARACTER_STATUSES) {
      expect(isCharacterStatus(s)).toBe(true)
      expect(normalizeCharacterStatus(s)).toBe(s)
    }
  })

  it('非法值（历史数据/AI 异常值/空值）回退默认 Alive', () => {
    expect(normalizeCharacterStatus('Undead')).toBe(DEFAULT_CHARACTER_STATUS)
    expect(normalizeCharacterStatus('alive')).toBe(DEFAULT_CHARACTER_STATUS) // 大小写敏感，视为非法
    expect(normalizeCharacterStatus('')).toBe(DEFAULT_CHARACTER_STATUS)
    expect(normalizeCharacterStatus(undefined)).toBe(DEFAULT_CHARACTER_STATUS)
    expect(normalizeCharacterStatus(null)).toBe(DEFAULT_CHARACTER_STATUS)
    expect(normalizeCharacterStatus(42)).toBe(DEFAULT_CHARACTER_STATUS)
    expect(isCharacterStatus('Undead')).toBe(false)
    expect(isCharacterStatus(undefined)).toBe(false)
  })

  it('中文标签不泄露原始英文串（非法值按默认展示）', () => {
    expect(characterStatusLabel('Alive')).toBe('存活')
    expect(characterStatusLabel('bogus')).toBe('存活') // 回退默认
    expect(CHARACTER_STATUS_LABELS.Alive).toBe('存活')
    expect(CHARACTER_STATUS_OPTIONS).toHaveLength(CHARACTER_STATUSES.length)
    expect(CHARACTER_STATUS_OPTIONS[0]).toEqual({ value: 'Alive', label: '存活' })
  })
})
