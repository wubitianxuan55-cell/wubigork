/**
 * ModelCenterPage.tabs.test.tsx — 瘦身重组（2026-10-27）导航编排契约
 *
 * 13 分类温和合并为 9：rail 分四节（概览/模型/质量与成本/接入），合并 tab
 * 内纵向多段渲染。本文件钉死 ModelCenterPage 的编排映射——Section 一律轻打桩
 * （data-testid 探针，内部渲染归各自测试管），聚焦三件事：
 * 1. rail 9 项 + 4 节标结构；2. 合并 tab 的 Section 组合（models 三段 /
 *    eval 两段 / governance 两段 / engine 两段既有先例）；3. 非合并 tab 单段。
 * Hook 桩照 ModelCenterPage.failover.test.tsx 惯例。
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('./OverviewSection', () => ({ OverviewSection: () => <div data-testid="sec-overview" /> }))
vi.mock('./LLMSection', () => ({ LLMSection: () => <div data-testid="sec-llm" /> }))
vi.mock('./SpecialtySection', () => ({ SpecialtySection: () => <div data-testid="sec-specialty" /> }))
vi.mock('./RecommendSection', () => ({ RecommendSection: () => <div data-testid="sec-recommend" /> }))
vi.mock('./ImageSection', () => ({ ImageSection: () => <div data-testid="sec-image" /> }))
vi.mock('./VoiceSection', () => ({ VoiceSection: () => <div data-testid="sec-tts" /> }))
vi.mock('./HerdsmanCatalogSection', () => ({ HerdsmanCatalogSection: () => <div data-testid="sec-catalog" /> }))
vi.mock('./BenchmarkSection', () => ({ BenchmarkSection: () => <div data-testid="sec-benchmark" /> }))
vi.mock('./RetrievalEvalSection', () => ({ RetrievalEvalSection: () => <div data-testid="sec-retrieval" /> }))
vi.mock('./SchedulingSection', () => ({ SchedulingSection: () => <div data-testid="sec-scheduling" /> }))
vi.mock('./EngineSection', () => ({ EngineSection: () => <div data-testid="sec-engine" /> }))
vi.mock('./BindSection', () => ({ BindSection: () => <div data-testid="sec-bind" /> }))
vi.mock('./StrategySection', () => ({ StrategySection: () => <div data-testid="sec-strategy" /> }))
vi.mock('./AttributionSection', () => ({ AttributionSection: () => <div data-testid="sec-attribution" /> }))
vi.mock('./StatsSection', () => ({ StatsSection: () => null }))

const mocks = vi.hoisted(() => ({
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

const SECTIONS = ['sec-overview', 'sec-llm', 'sec-specialty', 'sec-recommend', 'sec-image',
  'sec-tts', 'sec-catalog', 'sec-benchmark', 'sec-retrieval', 'sec-scheduling',
  'sec-engine', 'sec-bind', 'sec-strategy', 'sec-attribution'] as const

function visibleSections(): string[] {
  return SECTIONS.filter(id => screen.queryByTestId(id) != null)
}

beforeEach(() => {
  window.runtime = undefined
  mocks.useEngineState.mockImplementation(() => ({
    loading: false, engines: [], loadAll: () => Promise.resolve(),
  }))
  mocks.useStatsState.mockImplementation(() => ({}))
  mocks.useImageState.mockImplementation(() => ({}))
  mocks.useVoiceState.mockImplementation(() => ({ loadVoiceCfg: () => {} }))
  mocks.useBindState.mockImplementation(() => ({ featureCfg: {}, loadFeatureCfg: () => {}, refreshRoutes: () => {} }))
})

describe('ModelCenterPage · 瘦身重组导航编排（13 tab → 9）', () => {
  it('rail 渲染 9 个分类按钮与 4 个分节标题', () => {
    render(<ModelCenterPage />)
    // 「功能绑定」与 rail-foot 摘要标题撞名，label 查询收窄到 rail 按钮 label span
    const labels = ['总览', '模型一览', '模型库', '图片生成', '语音模型', '测评中心', '策略与成本', '功能绑定', '引擎管理']
    for (const label of labels) expect(screen.getByText(label, { selector: '.mc-rail-label' })).toBeTruthy()
    for (const group of ['概览', '模型', '质量与成本', '接入']) {
      expect(screen.getByText(group, { selector: '.mc-rail-group' })).toBeTruthy()
    }
    // 退役的旧分类名不再出现在导航
    for (const gone of ['语言模型', '专业模型', '推荐位', '受控测评', '检索质量', '空间策略', '成本归因']) {
      expect(screen.queryByText(gone, { selector: '.mc-rail-label' })).toBeNull()
    }
  })

  it('「模型一览」= 语言 + 专业 + 选型参考三段同屏', () => {
    render(<ModelCenterPage />)
    fireEvent.click(screen.getByText('模型一览', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-llm', 'sec-specialty', 'sec-recommend'])
  })

  it('「测评中心」= 受控测评 + 检索质量两段同屏', () => {
    render(<ModelCenterPage />)
    fireEvent.click(screen.getByText('测评中心', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-benchmark', 'sec-retrieval'])
  })

  it('「策略与成本」= 空间策略 + 成本归因两段同屏', () => {
    render(<ModelCenterPage />)
    fireEvent.click(screen.getByText('策略与成本', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-strategy', 'sec-attribution'])
  })

  it('「引擎管理」两段既有先例随批钉死；非合并 tab 单段渲染', () => {
    render(<ModelCenterPage />)
    fireEvent.click(screen.getByText('引擎管理', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-scheduling', 'sec-engine'])
    fireEvent.click(screen.getByText('模型库', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-catalog'])
    fireEvent.click(screen.getByText('总览', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-overview'])
  })
})
