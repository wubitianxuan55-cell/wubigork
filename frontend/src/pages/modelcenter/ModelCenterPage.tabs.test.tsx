/**
 * ModelCenterPage.tabs.test.tsx — 导航编排契约
 *
 * 2026-11-20 用户拍板：模型中心一级导航只保留高频操作——模型一览、功能绑定、
 * 引擎管理；模型一览覆盖语言/图像/语音/专业，选型参考等低频内容迁入设置。
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('./ModelGroupNav', () => ({ ModelGroupNav: () => <div data-testid="sec-model-nav" /> }))
vi.mock('./LLMSection', () => ({ LLMSection: () => <div data-testid="sec-llm" /> }))
vi.mock('./ImageSection', () => ({ ImageSection: () => <div data-testid="sec-image" /> }))
vi.mock('./VoiceSection', () => ({ VoiceSection: () => <div data-testid="sec-voice" /> }))
vi.mock('./SpecialtySection', () => ({ SpecialtySection: () => <div data-testid="sec-specialty" /> }))
vi.mock('./EngineSection', () => ({ EngineSection: () => <div data-testid="sec-engine" /> }))
vi.mock('./BindSection', () => ({ BindSection: () => <div data-testid="sec-bind" /> }))
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

const SECTIONS = ['sec-model-nav', 'sec-llm', 'sec-image', 'sec-voice', 'sec-specialty', 'sec-engine', 'sec-bind'] as const

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

describe('ModelCenterPage · 三项高频操作导航编排', () => {
  it('rail 只渲染模型一览/功能绑定/引擎管理三个一级入口', () => {
    render(<ModelCenterPage />)
    expect(screen.getAllByText(/模型一览|功能绑定|引擎管理/, { selector: '.mc-rail-label' }).map(node => node.textContent))
      .toEqual(['模型一览', '功能绑定', '引擎管理'])
    for (const gone of ['概览', '模型库', '选型参考', '测评中心', '策略与成本', '运行策略']) {
      expect(screen.queryByText(gone, { selector: '.mc-rail-label' })).toBeNull()
    }
    expect(document.querySelector('.mc-rail-group')).toBeNull()
    expect(document.querySelector('.mc-rail-foot')).toBeNull()
  })

  it('默认模型一览渲染目录导航与语言/图像/语音/专业四组；功能绑定只渲染 BindSection', () => {
    render(<ModelCenterPage />)
    expect(visibleSections()).toEqual(['sec-model-nav', 'sec-llm', 'sec-image', 'sec-voice', 'sec-specialty'])
    fireEvent.click(screen.getByText('功能绑定', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-bind'])
  })

  it('引擎管理只渲染 EngineSection，运行策略不再留在模型中心', () => {
    render(<ModelCenterPage />)
    fireEvent.click(screen.getByText('引擎管理', { selector: '.mc-rail-label' }))
    expect(visibleSections()).toEqual(['sec-engine'])
  })
})
