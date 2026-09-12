// sin/useSinNotes.test.tsx — 右栏「设定/大纲」数据源：读取、错误、重读。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'

const bridgeMock = vi.hoisted(() => ({
  SinNotesGet: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../gaea/lib/bridge')>()),
  app: new Proxy({} as Record<string, unknown>, {
    get(_t, prop: string) {
      return (bridgeMock as unknown as Record<string, unknown>)[prop]
    },
  }),
}))

import { useSinNotes } from './useSinNotes'

beforeEach(() => {
  bridgeMock.SinNotesGet.mockReset().mockResolvedValue({ notes: [], outline: '' })
})

describe('useSinNotes', () => {
  it('读便签与大纲：坏行过滤，字符串契约外的字段不进视图', async () => {
    bridgeMock.SinNotesGet.mockResolvedValue({
      notes: ['女主叫林晚', 42, null, '男主指节有旧伤'],
      outline: '第一章：雨夜',
    })
    const { result } = renderHook(() => useSinNotes('sin_1'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(bridgeMock.SinNotesGet).toHaveBeenCalledWith('sin_1')
    expect(result.current.doc.notes).toEqual(['女主叫林晚', '男主指节有旧伤'])
    expect(result.current.doc.outline).toBe('第一章：雨夜')
    expect(result.current.error).toBe('')
  })

  it('无故事（activeId 为空）不读且视图为空', async () => {
    const { result } = renderHook(() => useSinNotes(''))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(bridgeMock.SinNotesGet).not.toHaveBeenCalled()
    expect(result.current.doc).toEqual({ notes: [], outline: '' })
  })

  it('读取失败：如实给 error，不编造数据', async () => {
    bridgeMock.SinNotesGet.mockRejectedValue(new Error('故事 ID 非法'))
    const { result } = renderHook(() => useSinNotes('sin_2'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.doc).toEqual({ notes: [], outline: '' })
    expect(result.current.error).toContain('故事 ID 非法')
  })

  it('reload 重读（回合结束后取 AI 新写的便签/大纲）', async () => {
    bridgeMock.SinNotesGet.mockResolvedValueOnce({ notes: [], outline: '' })
    bridgeMock.SinNotesGet.mockResolvedValueOnce({ notes: ['新便签'], outline: '' })
    const { result } = renderHook(() => useSinNotes('sin_3'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.doc.notes).toEqual([])
    await act(async () => { result.current.reload() })
    await waitFor(() => expect(result.current.doc.notes).toEqual(['新便签']))
    expect(bridgeMock.SinNotesGet).toHaveBeenCalledTimes(2)
  })
})
