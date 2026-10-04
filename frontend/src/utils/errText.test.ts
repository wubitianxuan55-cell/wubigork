import { describe, expect, it } from 'vitest'
import { errText } from './errText'

describe('errText', () => {
  it('Error 实例取 message', () => {
    expect(errText(new Error(' boom '), '兜底')).toBe(' boom ')
  })

  it('Error 空 message 回退兜底（|| 语义，非 String(err)）', () => {
    expect(errText(new Error(''), '兜底')).toBe('兜底')
  })

  it('非 Error 值回退兜底', () => {
    expect(errText('纯字符串', '兜底')).toBe('兜底')
    expect(errText(42, '兜底')).toBe('兜底')
    expect(errText(null, '兜底')).toBe('兜底')
    expect(errText(undefined, '兜底')).toBe('兜底')
  })
})
