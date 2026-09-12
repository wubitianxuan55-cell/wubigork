// sin/useSinNotes.test.tsx — 右栏「设定/大纲」数据源：读取、错误、重读、编辑保存。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'

const bridgeMock = vi.hoisted(() => ({
  SinNotesGet: vi.fn(),
  SinNotesSave: vi.fn(),
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
  bridgeMock.SinNotesSave.mockReset().mockResolvedValue({ notes: [], outline: '' })
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

  it('save：整包写回（基线快照 JSON + 新值），成功后就地更新 doc', async () => {
    bridgeMock.SinNotesSave.mockResolvedValue({ notes: ['改后'], outline: '新大纲' })
    const { result } = renderHook(() => useSinNotes('sin_4'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    const baseline = { notes: ['旧'], outline: '旧大纲' }
    let res!: { ok: boolean; conflict: boolean; message: string }
    await act(async () => {
      res = await result.current.save(baseline, '新大纲', ['改后'], false)
    })
    expect(res).toEqual({ ok: true, conflict: false, message: '' })
    expect(bridgeMock.SinNotesSave).toHaveBeenCalledWith(
      'sin_4', JSON.stringify(baseline), '新大纲', JSON.stringify(['改后']), false,
    )
    expect(result.current.doc).toEqual({ notes: ['改后'], outline: '新大纲' })
  })

  it('save：冲突错误映射 conflict=true（按「底稿冲突：」前缀），其他错误不算冲突', async () => {
    const { result } = renderHook(() => useSinNotes('sin_5'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    bridgeMock.SinNotesSave.mockRejectedValueOnce(new Error('底稿冲突：便签/大纲已被其他端更新，请确认后再保存'))
    let res!: { ok: boolean; conflict: boolean; message: string }
    await act(async () => {
      res = await result.current.save({ notes: [], outline: '' }, '', [], false)
    })
    expect(res.ok).toBe(false)
    expect(res.conflict).toBe(true)
    bridgeMock.SinNotesSave.mockRejectedValueOnce(new Error('故事 ID 非法'))
    await act(async () => {
      res = await result.current.save({ notes: [], outline: '' }, '', [], false)
    })
    expect(res.ok).toBe(false)
    expect(res.conflict).toBe(false)
    expect(res.message).toContain('故事 ID 非法')
  })
})
