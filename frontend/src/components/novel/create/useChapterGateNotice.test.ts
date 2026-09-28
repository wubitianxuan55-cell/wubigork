// 生成后自动门通知钩子（v4.331）：文案拼装（契约/质量/AI 味/分析同步·空=完成）/
// 订阅与退订 / 非 report 忽略 / 浏览器无通道静默。
// v4.421.0：退订改走 subscribeWailsEvent（EventsOn 返回的「只摘除自己」清理函数）——
// 用例同步锁定「绝不 EventsOff 全清、同通道别人的监听不受影响」。
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { gateNoticeText, useChapterGateNotice, CHAPTER_GATE_CHANNEL } from './useChapterGateNotice'
import { message } from 'antd'

type Listener = (data: unknown) => void

/** window.runtime 桩：EventsOn 返回「只摘除自己」的退订函数（对齐 wails v2.13 桌面语义）。 */
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

function withRuntime(rt: { runtime: unknown }) {
  const w = window as unknown as { runtime?: unknown }
  const original = w.runtime
  w.runtime = rt.runtime
  return () => { w.runtime = original }
}

describe('useChapterGateNotice 自动门通知（v4.331）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('文案拼装：各维拼入；全空显示完成；分支章带标记', () => {
    expect(gateNoticeText({ type: 'report', chapterNum: 7, outlineIssues: 2, qualityIssues: 1, aiTasteScore: 35, analysisDone: true }))
      .toBe('第 7 章 生成后自动体检：契约 2 项 · 质量 1 项 · AI 味 35 · 分析已同步')
    expect(gateNoticeText({ type: 'report', chapterNum: 7, outlineIssues: 0, qualityIssues: 0, aiTasteScore: null, analysisDone: false }))
      .toBe('第 7 章 生成后自动体检完成')
    expect(gateNoticeText({ type: 'report', chapterNum: 7, branch: 'B1', analysisDone: true }))
      .toBe('第 7 章B1 生成后自动体检：分析已同步')
  })

  it('订阅 chapter-gate：report 载荷触发 message.info（key 防叠）；卸载只摘自己那个监听', () => {
    const rt = stubRuntime()
    const restore = withRuntime(rt)
    const infoSpy = vi.spyOn(message, 'info').mockImplementation(() => ({ then: () => ({}) }) as never)

    try {
      const { unmount } = renderHook(() => useChapterGateNotice())
      expect(rt.eventsOn).toHaveBeenCalledWith(CHAPTER_GATE_CHANNEL, expect.any(Function))
      expect(rt.count(CHAPTER_GATE_CHANNEL)).toBe(1)

      rt.emit(CHAPTER_GATE_CHANNEL, { type: 'report', chapterNum: 3, qualityIssues: 2, analysisDone: true })
      expect(infoSpy).toHaveBeenCalledWith(expect.objectContaining({
        key: 'chapter-gate',
        content: '第 3 章 生成后自动体检：质量 2 项 · 分析已同步',
      }))

      // 非 report 忽略（CustomEvent 包装兼容）
      rt.emit(CHAPTER_GATE_CHANNEL, { detail: { type: 'noise' } })
      expect(infoSpy).toHaveBeenCalledTimes(1)
      rt.emit(CHAPTER_GATE_CHANNEL, { detail: { type: 'report', chapterNum: 4 } })
      expect(infoSpy).toHaveBeenCalledTimes(2)

      unmount()
      // 退订 = EventsOn 返回的清理函数：本监听者摘除，且绝不触发 EventsOff 全清
      expect(rt.count(CHAPTER_GATE_CHANNEL)).toBe(0)
      expect(rt.eventsOff).not.toHaveBeenCalled()
    } finally {
      restore()
    }
  })

  it('退订只解自己那一个监听：同通道别人的监听不受影响（v4.62.2 事故纪律）', () => {
    const rt = stubRuntime()
    const restore = withRuntime(rt)
    const foreign = vi.fn()
    try {
      // 同通道上先挂一个「别人的」监听者（如另一消费者）
      rt.eventsOn(CHAPTER_GATE_CHANNEL, foreign)
      const { unmount } = renderHook(() => useChapterGateNotice())
      expect(rt.count(CHAPTER_GATE_CHANNEL)).toBe(2)

      unmount()
      expect(rt.count(CHAPTER_GATE_CHANNEL)).toBe(1)
      rt.emit(CHAPTER_GATE_CHANNEL, { type: 'report', chapterNum: 8 })
      expect(foreign).toHaveBeenCalledTimes(1)
      expect(rt.eventsOff).not.toHaveBeenCalled()
    } finally {
      restore()
    }
  })

  it('点击通知回调 onOpen(章号)；无效章号不回调；无 onOpen 纯通知（v4.336 跳转分析面板）', () => {
    const rt = stubRuntime()
    const restore = withRuntime(rt)
    const infoSpy = vi.spyOn(message, 'info').mockImplementation(() => ({ then: () => ({}) }) as never)
    const onOpen = vi.fn()

    try {
      const { unmount } = renderHook(() => useChapterGateNotice(onOpen))
      rt.emit(CHAPTER_GATE_CHANNEL, { type: 'report', chapterNum: 5, qualityIssues: 1 })
      const cfg = infoSpy.mock.calls[0][0] as unknown as { onClick: () => void }
      cfg.onClick()
      expect(onOpen).toHaveBeenCalledWith(5)

      // 无效章号：onClick 不触发回调
      rt.emit(CHAPTER_GATE_CHANNEL, { type: 'report' })
      ;(infoSpy.mock.calls[1][0] as unknown as { onClick: () => void }).onClick()
      expect(onOpen).toHaveBeenCalledTimes(1)

      // 无 onOpen：不炸（纯通知）
      unmount()
      renderHook(() => useChapterGateNotice())
      rt.emit(CHAPTER_GATE_CHANNEL, { type: 'report', chapterNum: 2 })
      expect(() => (infoSpy.mock.calls[2][0] as unknown as { onClick: () => void }).onClick()).not.toThrow()
    } finally {
      restore()
    }
  })

  it('浏览器无 runtime：订阅静默跳过不炸', () => {
    const w = window as unknown as { runtime?: unknown }
    const originalRuntime = w.runtime
    delete w.runtime
    try {
      expect(() => renderHook(() => useChapterGateNotice())).not.toThrow()
    } finally {
      w.runtime = originalRuntime
    }
  })
})
