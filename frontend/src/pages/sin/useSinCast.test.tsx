// sin/useSinCast.test.tsx — 原罪 × 角色库：库列表懒加载（v4.422 起 ensureLibrary
// 才首拉）、本故事选择往返、生效清单回填（悬空 id 由后端过滤后前端只认生效清单）。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'

const bridgeMock = vi.hoisted(() => ({
  CharacterList: vi.fn(),
  SinCastGet: vi.fn(),
  SinCastSet: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../gaea/lib/bridge')>()),
  app: new Proxy({} as Record<string, unknown>, {
    get(_t, prop: string) {
      return (bridgeMock as unknown as Record<string, unknown>)[prop]
    },
  }),
}))

import { useSinCast } from './useSinCast'

const LIB = [
  { id: 'c_lin', name: '林晚', roleType: '女主', personality: '清冷聪慧', background: '前朝遗孤', tags: ['女主角'], portraitUrl: '/tmp/lin.png' },
  { id: 'c_gu', name: '顾城', roleType: '男主', personality: '沉稳寡言', tags: ['将领'] },
]

beforeEach(() => {
  bridgeMock.CharacterList.mockReset().mockResolvedValue({ items: LIB, total: LIB.length })
  bridgeMock.SinCastGet.mockReset().mockResolvedValue([])
  bridgeMock.SinCastSet.mockReset().mockImplementation((_id: string, ids: string[]) => Promise.resolve(ids.slice(0, 8)))
})

describe('useSinCast', () => {
  it('挂载不拉库（懒加载）：ensureLibrary 后才请求角色库列表', async () => {
    const { result } = renderHook(() => useSinCast('sin_1'))
    await waitFor(() => expect(result.current.castIds).toEqual([]))
    expect(bridgeMock.CharacterList).not.toHaveBeenCalled()
    act(() => result.current.ensureLibrary())
    await waitFor(() => expect(result.current.libraryLoading).toBe(false))
    expect(bridgeMock.CharacterList).toHaveBeenCalledTimes(1)
    expect(result.current.library.map((c) => c.name)).toEqual(['林晚', '顾城'])
  })

  it('拉角色库列表 + 读本故事选择：选择映射成角色视图', async () => {
    bridgeMock.SinCastGet.mockResolvedValue(['c_gu', 'c_lin'])
    const { result } = renderHook(() => useSinCast('sin_1'))
    act(() => result.current.ensureLibrary())
    await waitFor(() => expect(result.current.libraryLoading).toBe(false))
    expect(result.current.library.map((c) => c.name)).toEqual(['林晚', '顾城'])
    expect(result.current.castIds).toEqual(['c_gu', 'c_lin'])
    expect(result.current.cast.map((c) => c.name)).toEqual(['顾城', '林晚'])
  })

  it('保存选择：以后端生效清单为准回填（悬空 id 不进视图）', async () => {
    bridgeMock.SinCastSet.mockResolvedValue(['c_lin'])
    const { result } = renderHook(() => useSinCast('sin_1'))
    act(() => result.current.ensureLibrary())
    await waitFor(() => expect(result.current.libraryLoading).toBe(false))
    await act(async () => { await result.current.saveCast(['c_lin', 'ghost']) })
    expect(bridgeMock.SinCastSet).toHaveBeenCalledWith('sin_1', ['c_lin', 'ghost'])
    expect(result.current.castIds).toEqual(['c_lin'])
    expect(result.current.cast.map((c) => c.name)).toEqual(['林晚'])
    expect(result.current.saving).toBe(false)
  })

  it('保存失败如实返回 {ok:false,message}，不再静默假成功（v4.426）', async () => {
    bridgeMock.SinCastSet.mockRejectedValue(new Error('写盘失败'))
    const { result } = renderHook(() => useSinCast('sin_1'))
    act(() => result.current.ensureLibrary())
    await waitFor(() => expect(result.current.libraryLoading).toBe(false))
    let res: { ok: boolean; message: string } | undefined
    await act(async () => { res = await result.current.saveCast(['c_lin']) })
    expect(res).toEqual({ ok: false, message: '写盘失败' })
    expect(result.current.saving).toBe(false)
  })

  it('保存在途切故事：旧故事的生效清单不回填当前面板（跨故事串写根除，v4.426）', async () => {
    let resolveSet!: (v: string[]) => void
    bridgeMock.SinCastSet.mockImplementation(() => new Promise<string[]>((r) => { resolveSet = r }))
    const { result, rerender } = renderHook(
      (props: { id: string }) => useSinCast(props.id),
      { initialProps: { id: 'sin_1' } },
    )
    await waitFor(() => expect(bridgeMock.SinCastGet).toHaveBeenCalledWith('sin_1'))
    let promise!: Promise<{ ok: boolean; message: string }>
    act(() => { promise = result.current.saveCast(['c_lin']) })
    await act(async () => { rerender({ id: 'sin_2' }) })
    await waitFor(() => expect(bridgeMock.SinCastGet).toHaveBeenCalledWith('sin_2'))
    await act(async () => { resolveSet(['c_lin']) })
    const res = await promise
    expect(res.ok).toBe(true) // 保存本身已成功（落的是发起故事 sin_1）
    // 关键断言：sin_2 的面板状态不被 sin_1 的生效清单污染
    expect(result.current.castIds).toEqual([])
    expect(result.current.saving).toBe(false)
  })

  it('库中缺失的选择不进视图（视图侧过滤，后端返回什么就展示什么）', async () => {
    bridgeMock.SinCastGet.mockResolvedValue(['c_lin', 'c_removed'])
    const { result } = renderHook(() => useSinCast('sin_2'))
    act(() => result.current.ensureLibrary())
    await waitFor(() => expect(result.current.cast.length).toBe(1))
    expect(result.current.castIds).toEqual(['c_lin', 'c_removed'])
    expect(result.current.cast.map((c) => c.name)).toEqual(['林晚'])
  })

  it('角色库读取失败：如实给 error，不抛也不编造数据', async () => {
    bridgeMock.CharacterList.mockRejectedValue(new Error('角色库未初始化'))
    const { result } = renderHook(() => useSinCast('sin_3'))
    act(() => result.current.ensureLibrary())
    await waitFor(() => expect(result.current.libraryLoading).toBe(false))
    expect(result.current.library).toEqual([])
    expect(result.current.libraryError).toContain('角色库未初始化')
  })

  it('无故事（activeId 为空）时不读选择，清空视图', async () => {
    const { result } = renderHook(() => useSinCast(''))
    act(() => result.current.ensureLibrary())
    await waitFor(() => expect(result.current.libraryLoading).toBe(false))
    expect(bridgeMock.SinCastGet).not.toHaveBeenCalled()
    expect(result.current.castIds).toEqual([])
  })
})
