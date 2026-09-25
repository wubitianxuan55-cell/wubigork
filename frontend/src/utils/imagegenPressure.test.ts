// imagegenPressure.test — 内存压力预检提示：节流窗口 + 空 note 忽略 + 重置。
import { beforeEach, describe, expect, it, vi } from 'vitest'

const messageMock = vi.hoisted(() => ({ warning: vi.fn() }))

vi.mock('antd', () => ({
  message: messageMock,
}))

import {
  handleImageGenPressureEvent,
  resetImageGenPressureThrottleForTest,
} from './imagegenPressure'

describe('handleImageGenPressureEvent', () => {
  beforeEach(() => {
    messageMock.warning.mockClear()
    resetImageGenPressureThrottleForTest()
  })

  it('命中提示：第一次调用 message.warning 携带 note', () => {
    handleImageGenPressureEvent({ note: '系统可用内存偏低（2.0 GB / 共 32.0 GB）' })
    expect(messageMock.warning).toHaveBeenCalledTimes(1)
    expect(messageMock.warning).toHaveBeenCalledWith(
      '系统可用内存偏低（2.0 GB / 共 32.0 GB）',
    )
  })

  it('2 分钟节流：窗口内第二次调用不提示，重置后恢复', () => {
    handleImageGenPressureEvent({ note: '第一条' })
    handleImageGenPressureEvent({ note: '第二条' })
    expect(messageMock.warning).toHaveBeenCalledTimes(1)
    resetImageGenPressureThrottleForTest()
    handleImageGenPressureEvent({ note: '重置后' })
    expect(messageMock.warning).toHaveBeenCalledTimes(2)
    expect(messageMock.warning).toHaveBeenLastCalledWith('重置后')
  })

  it('note 缺失/空白忽略', () => {
    handleImageGenPressureEvent(null)
    handleImageGenPressureEvent(undefined)
    handleImageGenPressureEvent({})
    handleImageGenPressureEvent({ note: '   ' })
    expect(messageMock.warning).not.toHaveBeenCalled()
  })
})
