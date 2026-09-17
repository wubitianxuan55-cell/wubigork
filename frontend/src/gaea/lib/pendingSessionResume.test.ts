// 会话级回源通道（7.3-1 收口）：pending 一次性消费 + 事件派发两态通道。
import { describe, expect, it, beforeEach, vi } from 'vitest'
import { requestSessionResume, consumePendingSessionResume, clearPendingSessionResume, RESUME_SESSION_EVENT } from './pendingSessionResume'

describe('pendingSessionResume 会话级回源通道（7.3-1 收口）', () => {
  beforeEach(() => {
    clearPendingSessionResume()
  })

  it('request 记 pending 且派发事件（keepAlive 已挂载态直达）', () => {
    const heard: string[] = []
    const on = (e: Event) => heard.push((e as CustomEvent<{ path?: string }>).detail?.path ?? '')
    window.addEventListener(RESUME_SESSION_EVENT, on)
    try {
      requestSessionResume('C:/ws/a/sessions/xyz.json')
      expect(heard).toEqual(['C:/ws/a/sessions/xyz.json'])
      expect(consumePendingSessionResume()).toBe('C:/ws/a/sessions/xyz.json')
    } finally {
      window.removeEventListener(RESUME_SESSION_EVENT, on)
    }
  })

  it('consume 一次性（二次为 null）；clear 清空', () => {
    requestSessionResume('p1')
    expect(consumePendingSessionResume()).toBe('p1')
    expect(consumePendingSessionResume()).toBeNull()
    requestSessionResume('p2')
    clearPendingSessionResume()
    expect(consumePendingSessionResume()).toBeNull()
  })

  it('空路径忽略（防御）', () => {
    const spy = vi.fn()
    window.addEventListener(RESUME_SESSION_EVENT, spy)
    try {
      requestSessionResume('')
      expect(spy).not.toHaveBeenCalled()
      expect(consumePendingSessionResume()).toBeNull()
    } finally {
      window.removeEventListener(RESUME_SESSION_EVENT, spy)
    }
  })
})
