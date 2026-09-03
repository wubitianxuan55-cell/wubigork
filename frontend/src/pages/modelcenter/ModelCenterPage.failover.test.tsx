/**
 * ModelCenterPage.failover.test.tsx — C 线「model-failover 提示文案 engineLabel 化」
 *
 * 页面级 failover toast 的文案回退链：emit model-failover 后断言 message.info
 * 文案——引擎列表命中 → 显示名（label）；列表未命中（流式竞态）→ 原始 id；
 * to_engine 缺失 → 通用文案。页面子分区与状态 Hook 全部打桩，聚焦 onFailover
 * 回调本体；runtime 事件按 CreatePage.test.tsx 既有 listener-map 惯例打桩。
 */
import { act, render } from '@testing-library/react'
import { message } from 'antd'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// 子分区/检查器打桩：它们只经 Context 消费状态，与 toast 文案无关，
// 打掉后页面仅剩 4 个状态 Hook + 事件回调，无需 mock 后端绑定面。
vi.mock('./OverviewSection', () => ({ OverviewSection: () => null }))
vi.mock('./LLMSection', () => ({ LLMSection: () => null }))
vi.mock('./ImageSection', () => ({ ImageSection: () => null }))
vi.mock('./VoiceSection', () => ({ VoiceSection: () => null }))
vi.mock('./SpecialtySection', () => ({ SpecialtySection: () => null }))
vi.mock('./HerdsmanCatalogSection', () => ({ HerdsmanCatalogSection: () => null }))
vi.mock('./BenchmarkSection', () => ({ BenchmarkSection: () => null }))
vi.mock('./RetrievalEvalSection', () => ({ RetrievalEvalSection: () => null }))
vi.mock('./SchedulingSection', () => ({ SchedulingSection: () => null }))
vi.mock('./EngineSection', () => ({ EngineSection: () => null }))
vi.mock('./BindSection', () => ({ BindSection: () => null }))
vi.mock('./StatsSection', () => ({ StatsSection: () => null }))
vi.mock('./InspectorPanel', () => ({ InspectorPanel: () => null }))

const mocks = vi.hoisted(() => ({
  engines: [] as unknown[], // 当前用例的引擎列表（beforeEach 实现里注入）
  useEngineState: vi.fn(),
  useStatsState: vi.fn(),
  useImageState: vi.fn(),
  useVoiceState: vi.fn(),
  useBindState: vi.fn(),
}))

vi.mock('./hooks/useEngineState', () => ({ useEngineState: mocks.useEngineState }))
vi.mock('./hooks/useStatsState', () => ({ useStatsState: mocks.useStatsState }))
vi.mock('./hooks/useImageState', () => ({ useImageState: mocks.useImageState }))
vi.mock('./hooks/useVoiceState', () => ({ useVoiceState: mocks.useVoiceState }))
vi.mock('./hooks/useBindState', () => ({ useBindState: mocks.useBindState }))

import ModelCenterPage from '../ModelCenterPage'

type Listener = (data: unknown) => void
const runtimeListeners = new Map<string, Listener>()
const EventsOn = vi.fn((name: string, handler: Listener) => {
  runtimeListeners.set(name, handler)
  return () => { runtimeListeners.delete(name) }
})

function emit(name: string, payload: unknown) {
  const handler = runtimeListeners.get(name)
  if (handler) act(() => handler(payload))
}

beforeEach(() => {
  runtimeListeners.clear()
  EventsOn.mockClear()
  Object.defineProperty(window, 'runtime', { configurable: true, writable: true, value: { EventsOn } })
  mocks.engines = []
  // Hook 桩只补页面渲染路径直接解构/遍历的字段，其余经 Context 透传给已打桩子分区。
  mocks.useEngineState.mockImplementation(() => ({
    loading: false,
    engines: mocks.engines,
    loadAll: () => Promise.resolve(),
  }))
  mocks.useStatsState.mockImplementation(() => ({}))
  mocks.useImageState.mockImplementation(() => ({}))
  mocks.useVoiceState.mockImplementation(() => ({ loadVoiceCfg: () => {} }))
  mocks.useBindState.mockImplementation(() => ({ featureCfg: {}, loadFeatureCfg: () => {}, refreshRoutes: () => {} }))
})

describe('ModelCenterPage · model-failover 提示文案（引擎 label 化）', () => {
  it('引擎列表命中时 toast 显示引擎显示名（label）', () => {
    mocks.engines = [{ id: 'deepseek', name: 'DeepSeek', label: '深度求索', type: 'deepseek', base_url: '', enabled: true, default_model: 'deepseek-chat', models: [] }]
    const spy = vi.spyOn(message, 'info')
    render(<ModelCenterPage />)
    emit('model-failover', { from_engine: 'glm', to_engine: 'deepseek', model: 'glm-4.7' })
    expect(spy).toHaveBeenCalledWith('调用失败，已切换到 深度求索 重试')
  })

  it('后端未下发 label 时经内置映射显示（glm → GLM 云端）', () => {
    mocks.engines = [{ id: 'glm', name: 'GLM', type: 'glm', base_url: '', enabled: true, default_model: 'glm-4.7', models: [] }]
    const spy = vi.spyOn(message, 'info')
    render(<ModelCenterPage />)
    emit('model-failover', { to_engine: 'glm' })
    expect(spy).toHaveBeenCalledWith('调用失败，已切换到 GLM 云端 重试')
  })

  it('列表未命中（流式竞态）回退显示原始引擎 id', () => {
    mocks.engines = []
    const spy = vi.spyOn(message, 'info')
    render(<ModelCenterPage />)
    emit('model-failover', { from_engine: 'glm', to_engine: 'deepseek', model: 'x' })
    expect(spy).toHaveBeenCalledWith('调用失败，已切换到 deepseek 重试')
  })

  it('to_engine 缺失时回退通用文案', () => {
    const spy = vi.spyOn(message, 'info')
    render(<ModelCenterPage />)
    emit('model-failover', { from_engine: 'glm' })
    expect(spy).toHaveBeenCalledWith('调用失败，已自动切换引擎重试')
  })
})
