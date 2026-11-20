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
  getLocalServiceModels: vi.fn(),
  getActiveEngine: vi.fn(),
  getDeepseekKeyStatus: vi.fn(),
  getOpencodeGoKeyStatus: vi.fn(),
  refreshEngineModels: vi.fn(),
  saveEngine: vi.fn(),
  testEngineConnection: vi.fn(),
  setEngineDefaultModel: vi.fn(),
  setActiveEngine: vi.fn(),
  setDeepseekKey: vi.fn(),
  setOpencodeGoKey: vi.fn(),
  addCustomEngine: vi.fn(),
  updateCustomEngine: vi.fn(),
  removeCustomEngine: vi.fn(),
}))

vi.mock('../../../gaea/lib/bridge', async (importOriginal) => ({
  ...(await importOriginal()),
  app: {
    GetActiveModel: vi.fn(async () => ''),
    StartLocalTTSService: vi.fn(async () => ({})),
  },
}))

import {
  getEngines, getLocalServiceModels, getActiveEngine, getDeepseekKeyStatus,
  getOpencodeGoKeyStatus, refreshEngineModels,
  setActiveEngine, setEngineDefaultModel,
  addCustomEngine, updateCustomEngine, removeCustomEngine,
  type EngineConfig,
} from '../../../api/engines'
import { app } from '../../../gaea/lib/bridge'
import { useEngineState } from './useEngineState'

const mGetEngines = vi.mocked(getEngines)
const mRefresh = vi.mocked(refreshEngineModels)
const mAddCustom = vi.mocked(addCustomEngine)
const mUpdateCustom = vi.mocked(updateCustomEngine)
const mRemoveCustom = vi.mocked(removeCustomEngine)

beforeEach(() => {
  vi.clearAllMocks()
  mGetEngines.mockResolvedValue([])
  vi.mocked(getLocalServiceModels).mockResolvedValue([])
  vi.mocked(getActiveEngine).mockResolvedValue('xai')
  vi.mocked(getDeepseekKeyStatus).mockResolvedValue({ configured: false, masked: '' })
  vi.mocked(getOpencodeGoKeyStatus).mockResolvedValue({ configured: false, masked: '' })
  mRefresh.mockResolvedValue([])
  vi.mocked(app.GetActiveModel).mockResolvedValue('')
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
    rerender({ cat: 'models' })
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

      // 新分类下定时器到达 5s：触发一次刷新（一次刷新 = 4 个本地引擎各刷一次：
      // herdsman / breeze / strata / llama / whirl）
      await act(async () => { vi.advanceTimersByTime(200) }) // t=9100
      expect(mRefresh).toHaveBeenCalledTimes(5)
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('useEngineState · 自定义引擎 handler 契约（A 刀）', () => {
  it('handleAddCustomEngine：透传三元组调用 addCustomEngine，成功后 loadAll 刷新并返回 true', async () => {
    const { result } = renderHook(() => useEngineState('engine'))
    await act(async () => {}) // 冲刷挂载 loadAll
    expect(mGetEngines).toHaveBeenCalledTimes(1)

    mAddCustom.mockResolvedValue('custom-1')
    let ok = false
    await act(async () => { ok = await result.current.handleAddCustomEngine('A', 'https://a.com/v1', 'k') })
    expect(mAddCustom).toHaveBeenCalledWith('A', 'https://a.com/v1', 'k')
    expect(ok).toBe(true)
    expect(mGetEngines).toHaveBeenCalledTimes(2) // 成功 → loadAll 刷新
  })

  it('handleAddCustomEngine：失败返回 false（errText 收窄为 message，不抛出）', async () => {
    const { result } = renderHook(() => useEngineState('engine'))
    await act(async () => {})
    mAddCustom.mockRejectedValue(new Error('地址无效'))
    let ok: boolean | null = null
    await act(async () => { ok = await result.current.handleAddCustomEngine('A', 'https://a.com/v1', '') })
    expect(ok).toBe(false)
    expect(mGetEngines).toHaveBeenCalledTimes(1) // 失败不刷新
  })

  it('handleUpdateCustomEngine：apiKey 空串透传（后端语义=不改 Key），成功刷新', async () => {
    const { result } = renderHook(() => useEngineState('engine'))
    await act(async () => {})
    mUpdateCustom.mockResolvedValue(undefined)
    let ok = false
    await act(async () => { ok = await result.current.handleUpdateCustomEngine('custom-1', 'B', 'https://b.com/v1', '') })
    expect(mUpdateCustom).toHaveBeenCalledWith('custom-1', 'B', 'https://b.com/v1', '')
    expect(ok).toBe(true)
    expect(mGetEngines).toHaveBeenCalledTimes(2)
  })

  it('handleRemoveCustomEngine：调用 removeCustomEngine 并刷新（无返回值）', async () => {
    const { result } = renderHook(() => useEngineState('engine'))
    await act(async () => {})
    mRemoveCustom.mockResolvedValue(undefined)
    await act(async () => { await result.current.handleRemoveCustomEngine('custom-1') })
    expect(mRemoveCustom).toHaveBeenCalledWith('custom-1')
    expect(mGetEngines).toHaveBeenCalledTimes(2)
  })
})

describe('useEngineState · 模型元数据透传（B 刀）', () => {
  const glmEngine = (models: EngineConfig['models']): EngineConfig => ({
    id: 'glm', name: 'GLM', type: 'glm', base_url: '', enabled: true, default_model: '', models,
  })

  it('makeModels 把 ModelInfo 目录字段聚合为 meta，llmModels 透传', async () => {
    mGetEngines.mockResolvedValue([
      glmEngine([
        {
          id: 'glm-5.3', owned_by: 'glm', status: 'running', kind: 'llm',
          context_length: 200_000, free: true, caps: ['vision'],
          points_in: 2.5, points_cached: 0.5, points_out: 5,
        },
        { id: 'glm-4', owned_by: 'glm', status: 'running', kind: 'llm' },
      ]),
    ])
    const { result } = renderHook(() => useEngineState('models'))
    await act(async () => {})
    const withMeta = result.current.llmModels.find(m => m.modelId === 'glm-5.3')
    expect(withMeta?.meta).toMatchObject({ context_length: 200_000, free: true, points_in: 2.5 })
    // 目录未下发任何元数据的模型：不构造空 meta（卡片徽标不占位）
    const noMeta = result.current.llmModels.find(m => m.modelId === 'glm-4')
    expect(noMeta?.meta).toBeUndefined()
    expect(result.current.makeModels(glmEngine([
      { id: 'glm-4-air', owned_by: 'glm', status: 'running', kind: 'llm' },
    ]))[0].meta).toBeUndefined()
  })
})

describe('useEngineState · 受管模型绑定（按需启动）', () => {
  it('Strata 模型卡只写绑定，不在这里启动模型', async () => {
    const card = {
      modelId: 'qwen3.8-flash-next-uncensored',
      modelName: 'qwen3.8-flash-next-uncensored',
      engineId: 'strata',
      engineName: 'Strata',
      engineType: 'strata',
      engineEnabled: true,
      status: 'stopped',
      kind: 'llm',
    }
    vi.mocked(setActiveEngine).mockResolvedValue(undefined)
    vi.mocked(setEngineDefaultModel).mockResolvedValue(undefined)

    const { result } = renderHook(() => useEngineState('models'))
    await act(async () => {})
    await act(async () => { await result.current.handleStartModel(card) })

    expect(setActiveEngine).toHaveBeenCalledWith('strata')
    expect(setEngineDefaultModel).toHaveBeenCalledWith('strata', card.modelId)
    // 刻意不接手动逐模型启动（原 startStrataModel 包装已删）：模型卡只写绑定，
    // 实际启停走引擎页的 StrataEngineStart/Stop 档位路径。
  })
})

describe('useEngineState · 本地服务目录 merge', () => {
  it('local:<kind> 卡并入专业/STT 列表，标签走本机服务名', async () => {
    mGetEngines.mockResolvedValue([
      { id: 'herdsman', name: 'Herdsman', type: 'herdsman', enabled: true, models: [
        { id: 'bge-m3', name: 'bge-m3', kind: 'embedding', status: 'running' },
      ] } as unknown as EngineConfig,
    ])
    vi.mocked(getLocalServiceModels).mockResolvedValue([
      { engineId: 'local:ovis', modelId: 'ovis-ocr2', name: 'OvisOCR2（本地·8137）', kind: 'ocr', detail: 'd', ready: true },
      { engineId: 'local:vision', modelId: 'qwen35-mmproj', name: '本地识图（qwen35 mmproj）', kind: 'vision', detail: 'd', ready: false },
      { engineId: 'local:stt', modelId: 'sherpa-zipformer-int8', name: '本地语音识别（sherpa int8）', kind: 'stt', detail: 'd', ready: true },
    ])
    const { result } = renderHook(() => useEngineState('overview'))
    await act(async () => { await Promise.resolve() })
    const spec = result.current.specialtyModels
    const ids = spec.map(m => m.engineId + ':' + m.modelId)
    expect(ids).toContain('herdsman:bge-m3')
    expect(ids).toContain('local:ovis:ovis-ocr2')
    expect(ids).toContain('local:vision:qwen35-mmproj')
    const ovis = spec.find(m => m.engineId === 'local:ovis')
    expect(ovis?.status).toBe('running')
    const stt = result.current.sttModels.map(m => m.engineId + ':' + m.modelId)
    expect(stt).toContain('local:stt:sherpa-zipformer-int8')
    expect(result.current.llmModels).toHaveLength(0)
  })
})
