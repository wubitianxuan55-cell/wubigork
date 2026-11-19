/**
 * ModelCenterPage.tabs.test.tsx — 导航编排契约
 *
 * 13 分类温和合并为 9（2026-10-27）：合并 tab 内纵向多段渲染。批 7（2026-11-19）
 * 曾把四个设置类 tab 退役收编设置页；2026-11-20 用户拍板回摆：「功能绑定/引擎
 * 管理」恢复「接入」节直达（单源 Section 组件双宿主，设置页 model 组消费同
 * 组件），rail 现 7 tab/4 节。本文件钉死编排映射——Section 一律轻打桩
 * （data-testid 探针，内部渲染归各自测试管），聚焦：rail 项/节标结构、合并 tab
 * 的 Section 组合（models 三段 / eval 两段 / governance 两段 / engine 两段先例）、
 * 非合并 tab 单段。Hook 桩照 ModelCenterPage.failover.test.tsx 惯例。
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

describe('ModelCenterPage · 瘦身重组导航编排（批 7 回摆后：5 → 7，接入节直达恢复）', () => {
  it('rail 渲染 7 个分类按钮与 4 个分节标题（功能绑定/引擎管理直达恢复，用户拍板 2026-11-20）', () => {
    render(<ModelCenterPage />)
    const labels = ['总览', '模型一览', '模型库', '测评中心', '策略与成本', '功能绑定', '引擎管理']
    for (const label of labels) expect(screen.getByText(label, { selector: '.mc-rail-label' })).toBeTruthy()
    for (const group of ['概览', '模型', '质量与成本', '接入']) {
      expect(screen.getByText(group, { selector: '.mc-rail-group' })).toBeTruthy()
    }
    // 仍退役的分类名不再出现在导航（图片生成/语音模型收编设置页 imagegen/chat 组）
    for (const gone of ['图片生成', '语音模型', '语言模型', '专业模型', '推荐位', '受控测评', '检索质量', '空间策略', '成本归因']) {
      expect(screen.queryByText(gone, { selector: '.mc-rail-label' })).toBeNull()
    }
  })

  it('「功能绑定」= BindSection 单段；rail-foot 摘要卡点击直达本页绑定 tab', () => {
    render(<ModelCenterPage />)
    fireEvent.click(screen.getByText('功能绑定', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-bind'])
    fireEvent.click(screen.getByLabelText(/管理功能绑定/))
    expect(visibleSections()).toEqual(['sec-bind'])
  })

  it('「引擎管理」= 本地调度 + 引擎管理两段同屏（引擎域操作含保活/预载/故障转移先例）', () => {
    render(<ModelCenterPage />)
    fireEvent.click(screen.getByText('引擎管理', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-scheduling', 'sec-engine'])
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

  it('「策略与成本」= 仅成本归因（空间策略已收编设置页模型组）；非合并 tab 单段渲染', () => {
    render(<ModelCenterPage />)
    fireEvent.click(screen.getByText('策略与成本', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-attribution'])
    fireEvent.click(screen.getByText('模型库', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-catalog'])
    fireEvent.click(screen.getByText('总览', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-overview'])
  })
})
