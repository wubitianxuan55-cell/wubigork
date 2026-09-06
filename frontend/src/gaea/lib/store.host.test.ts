/**
 * store.host.test.ts — 多宿主单订阅回归（v4.119.0 刀10）
 *
 * 背景：进度计划左栏对话 pane（ScheduleChatPane）与办公 GaeaApp 在 keepAlive
 * 树下可能同时挂载，二者都调 useController。事件绑定提升为模块级恰好一次
 * （ensureEventsBound）后，多宿主不得产生双订阅——否则 text 增量重复
 * dispatch，气泡翻倍。本测试钉死该语义。
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { emitMock } from './mock'
import { useController, useStore } from './store'

function facadeOk(): Record<string, unknown> {
  return {
    GaeaLogFrontendError: vi.fn(async () => {}),
    GaeaMeta: async () => ({ version: 'test', app: 'gaea' }),
    GaeaContext: async () => ({ used: 0, window: 0 }),
    GaeaHistory: async () => [],
    GaeaBalance: async () => ({ available: true, display: 'CNY 0.00' }),
    GaeaJobs: async () => [],
    GaeaFactBase: async () => ({ facts: [], markdown: '', count: 0, path: '' }),
    GaeaTCCAReport: async () => '{"ok":true}',
  }
}

describe('ensureEventsBound 多宿主恰好一次（刀10）', () => {
  afterEach(() => {
    delete (window as unknown as { go?: { app?: Record<string, unknown> } }).go;
  })

  it('两个宿主同时挂载：text 增量只入库一份', async () => {
    (window as unknown as { go?: { app?: Record<string, unknown> } }).go = { app: { CoreB: facadeOk() } };
    renderHook(() => useController())
    const second = renderHook(() => useController())
    await act(async () => { await new Promise((r) => setTimeout(r, 0)) })
    // 流式增量 ×3：若双订阅，最后一次断言会翻倍
    for (const chunk of ['你', '好', '呀']) {
      await act(async () => {
        emitMock({ kind: 'text', text: chunk })
        await new Promise((r) => setTimeout(r, 0))
      })
    }
    const items = useStore.getState().items
    const last = items[items.length - 1] as { kind: string; text: string; streaming: boolean }
    expect(last.kind).toBe('assistant')
    expect(last.text).toBe('你好呀')
    expect(last.streaming).toBe(true)
    second.unmount()
  })
})
