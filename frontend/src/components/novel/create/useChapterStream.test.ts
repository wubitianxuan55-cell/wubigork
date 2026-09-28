// useChapterStream.test.ts — 章节生成流式订阅 hook（T6-7.2 / T6-7.5）。
// v4.421.0：订阅/退订统一走 gaea/lib/wailsEvents.subscribeWailsEvent（EventsOn
// 返回的「只摘除自己」清理函数）——用例锁定两点回归：
//  ① 裸 EventsOff(channel) 会连坐同通道别人的监听（事故纪律，绝不允许）；
//  ② attach 重复调用先退订旧监听，卸载自动退订（无悬挂）。
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useChapterStream, CREATE_CHAPTER_STREAM_CHANNEL } from './useChapterStream'
import type { CreateChapterStreamEvent } from './chapterStreamTypes'

type Listener = (data: unknown) => void

/** window.runtime 桩：EventsOn 返回「只摘除自己」的退订函数（wails v2.13 桌面语义）。 */
function stubRuntime() {
  const listeners = new Map<string, Listener[]>()
  const eventsOn = vi.fn((ch: string, h: Listener) => {
    const arr = listeners.get(ch) ?? []
    arr.push(h)
    listeners.set(ch, arr)
    return () => {
      const cur = listeners.get(ch)
      if (!cur) return
      const i = cur.indexOf(h)
      if (i >= 0) cur.splice(i, 1)
    }
  })
  const eventsOff = vi.fn()
  return {
    eventsOn,
    eventsOff,
    count: (ch: string) => (listeners.get(ch) ?? []).length,
    emit: (ch: string, payload: unknown) => { for (const h of [...(listeners.get(ch) ?? [])]) h(payload) },
    runtime: { EventsOn: eventsOn, EventsOff: eventsOff },
  }
}

function withRuntime(runtime: unknown) {
  const w = window as unknown as { runtime?: unknown }
  const original = w.runtime
  w.runtime = runtime
  return () => { w.runtime = original }
}

describe('useChapterStream 流式订阅（T6-7.2）', () => {
  beforeEach(() => { vi.clearAllMocks() })
  afterEach(() => { vi.restoreAllMocks() })

  it('attach 注册通道并把受校验事件分发给 onEvent；畸形负载忽略', () => {
    const rt = stubRuntime()
    const restore = withRuntime(rt.runtime)
    try {
      const { result } = renderHook(() => useChapterStream())
      const onEvent = vi.fn()
      result.current.attach({ onEvent })

      expect(rt.eventsOn).toHaveBeenCalledWith(CREATE_CHAPTER_STREAM_CHANNEL, expect.any(Function))
      expect(rt.count(CREATE_CHAPTER_STREAM_CHANNEL)).toBe(1)

      rt.emit(CREATE_CHAPTER_STREAM_CHANNEL, { type: 'chunk', content: '第一段', total: 5 })
      expect(onEvent).toHaveBeenCalledWith({ type: 'chunk', content: '第一段', total: 5 })

      // CustomEvent 包装兼容（event.detail）
      rt.emit(CREATE_CHAPTER_STREAM_CHANNEL, { detail: { type: 'done', chapterNum: 1 } })
      expect(onEvent).toHaveBeenCalledTimes(2)
      expect((onEvent.mock.calls[1][0] as CreateChapterStreamEvent).type).toBe('done')

      // 畸形/未知负载被忽略（不抛错、不误分发）
      rt.emit(CREATE_CHAPTER_STREAM_CHANNEL, { type: 'nope' })
      rt.emit(CREATE_CHAPTER_STREAM_CHANNEL, null)
      expect(onEvent).toHaveBeenCalledTimes(2)
    } finally {
      restore()
    }
  })

  it('detach 只摘自己那一个监听：同通道别人的监听仍在，且绝不 EventsOff 全清', () => {
    const rt = stubRuntime()
    const restore = withRuntime(rt.runtime)
    const foreign = vi.fn()
    try {
      // 同通道先挂一个「别人的」监听者（如另一消费者）
      rt.eventsOn(CREATE_CHAPTER_STREAM_CHANNEL, foreign)
      const { result } = renderHook(() => useChapterStream())
      const onEvent = vi.fn()
      result.current.attach({ onEvent })
      expect(rt.count(CREATE_CHAPTER_STREAM_CHANNEL)).toBe(2)

      result.current.detach()
      expect(rt.count(CREATE_CHAPTER_STREAM_CHANNEL)).toBe(1)

      // 本 hook 已退订：事件只到 foreign，不再到 onEvent
      rt.emit(CREATE_CHAPTER_STREAM_CHANNEL, { type: 'chunk', content: 'x' })
      expect(foreign).toHaveBeenCalledTimes(1)
      expect(onEvent).not.toHaveBeenCalled()
      expect(rt.eventsOff).not.toHaveBeenCalled()
    } finally {
      restore()
    }
  })

  it('attach 重复调用先退订旧监听（同通道只留一个自己）；卸载自动退订（无悬挂）', () => {
    const rt = stubRuntime()
    const restore = withRuntime(rt.runtime)
    try {
      const { result, unmount } = renderHook(() => useChapterStream())
      const first = vi.fn()
      const second = vi.fn()
      result.current.attach({ onEvent: first })
      result.current.attach({ onEvent: second })
      expect(rt.count(CREATE_CHAPTER_STREAM_CHANNEL)).toBe(1)

      rt.emit(CREATE_CHAPTER_STREAM_CHANNEL, { type: 'chunk', content: 'y' })
      expect(first).not.toHaveBeenCalled()
      expect(second).toHaveBeenCalledTimes(1)

      unmount()
      expect(rt.count(CREATE_CHAPTER_STREAM_CHANNEL)).toBe(0)
      expect(rt.eventsOff).not.toHaveBeenCalled()
    } finally {
      restore()
    }
  })

  it('浏览器无事件通道：attach 静默跳过不抛错（无监听可退也不炸）', () => {
    const restore = withRuntime(undefined)
    try {
      const { result } = renderHook(() => useChapterStream())
      const onEvent = vi.fn()
      expect(() => result.current.attach({ onEvent })).not.toThrow()
      expect(() => result.current.detach()).not.toThrow()
      expect(onEvent).not.toHaveBeenCalled()
    } finally {
      restore()
    }
  })
})
