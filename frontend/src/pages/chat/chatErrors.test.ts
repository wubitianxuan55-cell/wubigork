import { describe, expect, it } from 'vitest'
import { describeChatError } from './chatErrors'

// B1 错误分类映射表契约：码 → 文案 + 动作；未命中回落原文。
describe('describeChatError', () => {
  it('引擎不可达/鉴权失败带 modelcenter 动作', () => {
    expect(describeChatError('engine_unreachable', 'dial tcp: refused')).toEqual({
      text: '引擎未连接：请检查本机引擎状态后重试',
      action: 'modelcenter',
    })
    expect(describeChatError('auth', 'HTTP 401')).toMatchObject({ action: 'modelcenter' })
  })

  it('其余已知码给文案不带动作', () => {
    for (const code of ['engine_busy', 'timeout', 'context_overflow', 'persistence', 'empty_reply']) {
      const d = describeChatError(code, 'raw')
      expect(d.text).not.toContain('raw')
      expect(d.action).toBeUndefined()
    }
  })

  it('未命中码回落原文（零行为变化）', () => {
    expect(describeChatError(undefined, '怪错原文')).toEqual({ text: '请求失败：怪错原文' })
    expect(describeErrorText('', undefined)).toBe('请求失败：未知错误')
  })
})

// 小助手避免上面直接内联第二处断言写错
function describeErrorText(code: string, raw: string | undefined): string {
  return describeChatError(code || undefined, raw).text
}
