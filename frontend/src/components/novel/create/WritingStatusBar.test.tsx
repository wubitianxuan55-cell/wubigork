/**
 * WritingStatusBar.test.tsx — 创作页写作状态条契约（2026-10-27 大道至简：
 * 四条横幅收敛为一条）。锁三件事：
 * 1. 全部条目**恒在 DOM**（testid 契约——CreatePage.test 行为锁零改动的前提）
 * 2. 视觉默认只显最高优先级一条（error 恒居首不被折叠吞掉）
 * 3. 多条时「N 条反馈」展开全部
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { WritingStatusBar } from './WritingStatusBar'
import type { AiTasteResult } from './chapterStreamTypes'

const taste = (score: number): AiTasteResult => ({ score, issues: [] }) as AiTasteResult

function visibleItemEls(): HTMLElement[] {
  return Array.from(document.querySelectorAll<HTMLElement>('.novel-create-statusbar-item'))
    .filter(el => el.style.display !== 'none')
}

describe('WritingStatusBar · 写作状态条', () => {
  it('无反馈时不渲染', () => {
    const { container } = render(
      <WritingStatusBar samplerNote="" stageClosing={null} declGate={null} aiTaste={null} onRunHealth={() => {}} />,
    )
    expect(container.querySelector('.novel-create-statusbar')).toBeNull()
  })

  it('单条反馈：无展开钮，直接可见', () => {
    render(<WritingStatusBar samplerNote="已切换采样器" stageClosing={null} declGate={null} aiTaste={null} onRunHealth={() => {}} />)
    expect(screen.getByTestId('sampler-note')).toBeTruthy()
    expect(visibleItemEls()).toHaveLength(1)
    expect(screen.queryByRole('button', { name: /条反馈/ })).toBeNull()
  })

  it('多条反馈：error 恒居首常显，其余恒在 DOM 但默认隐藏；「N 条反馈」展开全部', () => {
    render(
      <WritingStatusBar
        samplerNote="已切换采样器"
        stageClosing={{ stage: 2 }}
        declGate={{ declared: true, accepted: false, issues: ['伏笔 F-3 未交代'] }}
        aiTaste={taste(72)}
        onRunHealth={() => {}}
      />,
    )
    // 四条全部在 DOM（testid 契约）
    for (const id of ['decl-gate-banner', 'stage-closing-banner', 'ai-taste', 'sampler-note']) {
      expect(screen.queryByTestId(id)).toBeTruthy()
    }
    // 默认仅 error 一条可见（优先级最高，不被折叠吞掉）
    expect(visibleItemEls()).toHaveLength(1)
    expect(visibleItemEls()[0].querySelector('[data-testid="decl-gate-banner"]')).toBeTruthy()
    // 展开：四条全可见
    fireEvent.click(screen.getByRole('button', { name: '4 条反馈' }))
    expect(visibleItemEls()).toHaveLength(4)
  })
})
