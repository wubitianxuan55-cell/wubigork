// StyleFingerprintPanel.test.tsx — 文风指纹面板（受控 presentational Modal）
// 纯 props 驱动：数据获取全在 CreatePage，这里不需要 mock NovelB 绑定，
// 直接给 status/score fixture 断言三段 body 与 footer 按钮行为。
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import React from 'react'

import StyleFingerprintPanel from './StyleFingerprintPanel'
import type { FingerprintScorePayload, FingerprintStatusPayload } from '../../gaea/lib/bridge/novel'

type PanelProps = React.ComponentProps<typeof StyleFingerprintPanel>

/** 已构建参考档 fixture（全量 summary + 口头禅三源：2 bigram + 0 trigram + 1 签名词）。 */
const builtStatus: FingerprintStatusPayload = {
  exists: true,
  builtAt: '2026-09-13T10:00:00Z',
  chapters: 3,
  chars: 3200,
  summary: {
    sentenceMean: 18.6,
    sentenceSd: 7.4,
    paraMean: 96.3,
    ttr1000: 412.5,
    dialogRatio: 0.34,
    fourCharRatio: 0.052,
    connectiveDensity: 0.018,
    adjAdvDensity: 0.041,
    topBigrams: ['的时候', '看了一眼'],
    topTrigrams: [],
    authorSignWords: ['却说'],
  },
}

/** 体检结果 fixture：有参考档（带 delta）+ 1 条 medium 命中。 */
const scoreWithRef: FingerprintScorePayload = {
  chapterNum: 3,
  refExists: true,
  score: 42,
  delta: 0.31,
  issues: [
    { start: 12, end: 40, reason: '解释腔排比', severity: 'medium', suggestion: '拆成两句直述', excerpt: '这不是巧合，而是伏线。' },
  ],
}

function renderPanel(overrides: Partial<PanelProps> = {}) {
  const props: PanelProps = {
    open: true,
    onClose: vi.fn(),
    busy: false,
    msg: '',
    status: null,
    score: null,
    onBuild: vi.fn(),
    onScore: vi.fn(),
    hasChapter: true,
    ...overrides,
  }
  return { props, ...render(<StyleFingerprintPanel {...props} />) }
}

describe('StyleFingerprintPanel 文风指纹面板', () => {
  it('未构建空态：渲染引导文案，构建按钮可点（onBuild 被调）', () => {
    const { props } = renderPanel({ status: { exists: false } })
    expect(screen.getByText(/用已有章节构建你自己的文风基线；构建后 AI 味体检将对照你的风格打分/)).toBeTruthy()
    const build = screen.getByRole('button', { name: '构建/重建参考档' }) as HTMLButtonElement
    expect(build.disabled).toBe(false)
    fireEvent.click(build)
    expect(props.onBuild).toHaveBeenCalledTimes(1)
    // 体检按钮存在（有当前章时可用），体检区给出操作引导
    expect((screen.getByRole('button', { name: '体检当前章' }) as HTMLButtonElement).disabled).toBe(false)
    expect(screen.getByText(/点击「体检当前章」获取本章 AI 味评分/)).toBeTruthy()
  })

  it('已构建：元信息行、摘要格子数值与口头禅 Tag 渲染', () => {
    renderPanel({ status: builtStatus })
    // 元信息行：N 章 · M 字 · 构建于 <time>
    expect(screen.getByText(/3 章 ·/)).toBeTruthy()
    expect(screen.getByText(/3,?200 字 · 构建于/)).toBeTruthy()
    // 摘要格子：label + 数值（1~2 位小数）
    expect(screen.getByText('句长均值')).toBeTruthy()
    expect(screen.getByText('词汇多样 TTR')).toBeTruthy()
    expect(screen.getByText('形副密度')).toBeTruthy()
    expect(screen.getByText('18.60')).toBeTruthy()
    expect(screen.getByText('412.50')).toBeTruthy()
    expect(screen.getByText('0.34')).toBeTruthy()
    // 口头禅 = topBigrams + topTrigrams + authorSignWords 平铺 Tag（空 trigram 不阻塞）
    expect(screen.getByText('口头禅：')).toBeTruthy()
    expect(screen.getByText('的时候')).toBeTruthy()
    expect(screen.getByText('看了一眼')).toBeTruthy()
    expect(screen.getByText('却说')).toBeTruthy()
  })

  it('体检结果：score 语义分档、delta 基线距离与 issues（severity Tag + excerpt + 建议）渲染', () => {
    renderPanel({ status: builtStatus, score: scoreWithRef })
    // 大数字 42 → 35~59 档「有 AI 痕迹」
    expect(screen.getByText('42')).toBeTruthy()
    expect(screen.getByText(/有 AI 痕迹/)).toBeTruthy()
    expect(screen.getByText(/与你的基线距离 Δ 0.31（越小越像你）/)).toBeTruthy()
    // 命中条目：medium → Tag「中」+ reason + excerpt 引用 + suggestion 小字
    expect(screen.getByText('中')).toBeTruthy()
    expect(screen.getByText('解释腔排比')).toBeTruthy()
    expect(screen.getByText(/这不是巧合，而是伏线。/)).toBeTruthy()
    expect(screen.getByText(/建议：拆成两句直述/)).toBeTruthy()
  })

  it('无 delta：显示「未构建参考档」通用阈值口径；issues 空显示未命中', () => {
    renderPanel({
      score: { chapterNum: 2, refExists: false, score: 70, issues: [] },
    })
    // 70 ≥ 60 → AI 味偏重档
    expect(screen.getByText(/AI 味偏重/)).toBeTruthy()
    expect(screen.getByText('未构建参考档，按通用阈值打分')).toBeTruthy()
    expect(screen.getByText('未命中明显 AI 套路')).toBeTruthy()
  })

  it('hasChapter=false：「体检当前章」禁用（构建按钮仍可用）', () => {
    renderPanel({ hasChapter: false })
    expect((screen.getByRole('button', { name: '体检当前章' }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: '构建/重建参考档' }) as HTMLButtonElement).disabled).toBe(false)
  })
})
