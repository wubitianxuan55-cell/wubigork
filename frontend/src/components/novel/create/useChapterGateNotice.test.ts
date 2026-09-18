// 生成后自动门通知钩子（v4.331）：文案拼装（契约/质量/AI 味/分析同步·空=完成）/
// 订阅与退订 / 非 report 忽略 / 浏览器无通道静默。
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { gateNoticeText, useChapterGateNotice, CHAPTER_GATE_CHANNEL } from './useChapterGateNotice'
import { message } from 'antd'

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

  it('订阅 chapter-gate：report 载荷触发 message.info（key 防叠）；卸载退订', () => {
    const listeners: Record<string, (data: unknown) => void> = {}
    const eventsOn = vi.fn((ch: string, h: (data: unknown) => void) => { listeners[ch] = h })
    const eventsOff = vi.fn()
    // 只换 runtime 属性（整窗 stub 会炸 react-dom 的 document/window 依赖）
    const w = window as unknown as { runtime?: unknown }
    const originalRuntime = w.runtime
    w.runtime = { EventsOn: eventsOn, EventsOff: eventsOff }
    const infoSpy = vi.spyOn(message, 'info').mockImplementation(() => ({ then: () => ({}) }) as never)

    try {
      const { unmount } = renderHook(() => useChapterGateNotice())
      expect(eventsOn).toHaveBeenCalledWith(CHAPTER_GATE_CHANNEL, expect.any(Function))

      listeners[CHAPTER_GATE_CHANNEL]({ type: 'report', chapterNum: 3, qualityIssues: 2, analysisDone: true })
      expect(infoSpy).toHaveBeenCalledWith(expect.objectContaining({
        key: 'chapter-gate',
        content: '第 3 章 生成后自动体检：质量 2 项 · 分析已同步',
      }))

      // 非 report 忽略（CustomEvent 包装兼容）
      listeners[CHAPTER_GATE_CHANNEL]({ detail: { type: 'noise' } })
      expect(infoSpy).toHaveBeenCalledTimes(1)
      listeners[CHAPTER_GATE_CHANNEL]({ detail: { type: 'report', chapterNum: 4 } })
      expect(infoSpy).toHaveBeenCalledTimes(2)

      unmount()
      expect(eventsOff).toHaveBeenCalledWith(CHAPTER_GATE_CHANNEL)
    } finally {
      w.runtime = originalRuntime
    }
  })

  it('点击通知回调 onOpen(章号)；无效章号不回调；无 onOpen 纯通知（v4.336 跳转分析面板）', () => {
    const listeners: Record<string, (data: unknown) => void> = {}
    const eventsOn = vi.fn((ch: string, h: (data: unknown) => void) => { listeners[ch] = h })
    const eventsOff = vi.fn()
    const w = window as unknown as { runtime?: unknown }
    const originalRuntime = w.runtime
    w.runtime = { EventsOn: eventsOn, EventsOff: eventsOff }
    const infoSpy = vi.spyOn(message, 'info').mockImplementation(() => ({ then: () => ({}) }) as never)
    const onOpen = vi.fn()

    try {
      const { unmount } = renderHook(() => useChapterGateNotice(onOpen))
      listeners[CHAPTER_GATE_CHANNEL]({ type: 'report', chapterNum: 5, qualityIssues: 1 })
      const cfg = infoSpy.mock.calls[0][0] as unknown as { onClick: () => void }
      cfg.onClick()
      expect(onOpen).toHaveBeenCalledWith(5)

      // 无效章号：onClick 不触发回调
      listeners[CHAPTER_GATE_CHANNEL]({ type: 'report' })
      ;(infoSpy.mock.calls[1][0] as unknown as { onClick: () => void }).onClick()
      expect(onOpen).toHaveBeenCalledTimes(1)

      // 无 onOpen：不炸（纯通知）
      unmount()
      renderHook(() => useChapterGateNotice())
      listeners[CHAPTER_GATE_CHANNEL]({ type: 'report', chapterNum: 2 })
      expect(() => (infoSpy.mock.calls[2][0] as unknown as { onClick: () => void }).onClick()).not.toThrow()
    } finally {
      w.runtime = originalRuntime
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
