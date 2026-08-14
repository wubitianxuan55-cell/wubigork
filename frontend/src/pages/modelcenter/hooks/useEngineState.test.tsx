/**
 * useEngineState.test.tsx — T6-6.5 竞态守卫测试
 *
 * 1) 延迟 mock 下旧分类触发的 refreshLocalModels 结果不覆盖新分类；
 * 2) 5s 定时器随 category 重置（切分类后旧定时器不再触发刷新）。
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Category } from '../utils'

vi.mock('../../../api/engines', () => ({
  getEngines: vi.fn(),
  getActiveEngine: vi.fn(),
  getDeepseekKeyStatus: vi.fn(),
  getOpencodeGoKeyStatus: vi.fn(),
  getOpencodeZenKeyStatus: vi.fn(),
  refreshEngineModels: vi.fn(),
  saveEngine: vi.fn(),
  testEngineConnection: vi.fn(),
  setEngineDefaultModel: vi.fn(),
  setActiveEngine: vi.fn(),
  setDeepseekKey: vi.fn(),
  setOpencodeGoKey: vi.fn(),
  setOpencodeZenKey: vi.fn(),
}))

vi.mock('../../../wailsjsCompat', () => ({
  GetActiveModel: vi.fn(),
  StartLocalTTSService: vi.fn(),
}))

import {
  getEngines, getActiveEngine, getDeepseekKeyStatus,
  getOpencodeGoKeyStatus, getOpencodeZenKeyStatus, refreshEngineModels,
} from '../../../api/engines'
import { GetActiveModel } from '../../../wailsjsCompat'
import { useEngineState } from './useEngineState'

const mGetEngines = vi.mocked(getEngines)
const mRefresh = vi.mocked(refreshEngineModels)

beforeEach(() => {
  vi.clearAllMocks()
  mGetEngines.mockResolvedValue([])
  vi.mocked(getActiveEngine).mockResolvedValue('xai')
  vi.mocked(getDeepseekKeyStatus).mockResolvedValue({ configured: false, masked: '' })
  vi.mocked(getOpencodeGoKeyStatus).mockResolvedValue({ configured: false, masked: '' })
  vi.mocked(getOpencodeZenKeyStatus).mockResolvedValue({ configured: false, masked: '' })
  mRefresh.mockResolvedValue([])
  vi.mocked(GetActiveModel).mockResolvedValue('')
})

describe('useEngineState · T6-6.5 竞态守卫', () => {
  it('延迟 mock 下旧分类的刷新结果不覆盖新分类（过期结果被丢弃）', async () => {
    let release!: (v: never[]) => void
    const gate = new Promise<never[]>((r) => { release = r })
    mRefresh.mockReturnValue(gate)

    const { result, rerender } = renderHook(
      ({ cat }: { cat: Category }) => useEngineState(cat),
      { initialProps: { cat: 'overview' } },
    )
    // 挂载：loadAll 读取一次引擎列表
    await act(async () => {})
    expect(mGetEngines).toHaveBeenCalledTimes(1)

    // 发起一次慢速的本地模型刷新（挂起在 gate 上）
    let refreshPromise!: Promise<void>
    act(() => {
      refreshPromise = result.current.refreshLocalModels()
    })

    // 切分类：effect 清理自增 refreshSeq，作废在途刷新
    rerender({ cat: 'llm' })
    await act(async () => {
      release!([])
      await refreshPromise
    })

    // 过期刷新未落地：getEngines 仍只有挂载时一次调用，状态未被旧结果覆盖
    expect(mGetEngines).toHaveBeenCalledTimes(1)
    expect(result.current.engines).toEqual([])
  })

  it('5s 定时器随 category 重置：切分类后旧定时器不再触发刷新', async () => {
    vi.useFakeTimers()
    try {
      const { rerender } = renderHook(
        ({ cat }: { cat: Category }) => useEngineState(cat),
        { initialProps: { cat: 'overview' } },
      )
      await act(async () => {}) // 冲刷挂载 effect 与 loadAll 微任务

      // t=4000：首次刷新定时器（t=5000 触发）尚未到达
      await act(async () => { vi.advanceTimersByTime(4000) })
      expect(mRefresh).not.toHaveBeenCalled()

      // 切分类 → 清理旧定时器并重排（新定时器 t=4000+5000=9000 触发）
      rerender({ cat: 'bind' })
      await act(async () => { vi.advanceTimersByTime(4900) }) // t=8900 旧定时器若未重置早已触发
      expect(mRefresh).not.toHaveBeenCalled()

      // 新分类下定时器到达 5s：触发一次刷新（一次刷新 = 3 个本地引擎各刷一次）
      await act(async () => { vi.advanceTimersByTime(200) }) // t=9100
      expect(mRefresh).toHaveBeenCalledTimes(3)
    } finally {
      vi.useRealTimers()
    }
  })
})
