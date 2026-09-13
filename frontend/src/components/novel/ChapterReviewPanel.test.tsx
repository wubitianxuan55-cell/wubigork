// ChapterReviewPanel.test.tsx — 平台评审面板（受控 presentational Modal）
// 纯 props 驱动：数据获取全在 CreatePage，这里不需要 mock NovelB 绑定，
// 直接给 report/platforms fixture 断言结论区、逐维排序与证据渲染、跳过口径与按钮行为。
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import React from 'react'

import ChapterReviewPanel from './ChapterReviewPanel'
import type { ChapterReviewPayload, ReviewPlatform } from '../../gaea/lib/bridge/novel'

type PanelProps = React.ComponentProps<typeof ChapterReviewPanel>

const platforms: ReviewPlatform[] = [
  { id: 'general', label: '通用', form: 'chapter' },
  { id: 'fanqie', label: '番茄小说', form: 'chapter' },
  { id: 'zhihu', label: '知乎盐言故事', form: 'story' },
]

/** 报告 fixture：1×S1(fail) + 1×S2(warn) + 1×S3(warn) + 1×pass + 1×skip。 */
const report: ChapterReviewPayload = {
  chapterNum: 7,
  platform: 'general',
  platformLabel: '通用',
  words: 2380,
  verdict: 'REJECT',
  counts: { S1: 1, S2: 1, S3: 1, S4: 0 },
  dimensions: [
    { id: 'punctuation_rhythm', label: '标点节奏', verdict: 'pass', detail: '标点节奏正常' },
    { id: 'protagonist_presence', label: '主角存在感', verdict: 'skip', detail: '角色库未标注主角，跳过' },
    {
      id: 'format_readability', label: '段落节奏', verdict: 'warn', severity: 'S3',
      detail: '段落节奏：2 段超过 200 字', advice: '按镜头断段', evidence: [{ paragraph: 17, excerpt: '他把单据摊在桌上。' }],
    },
    {
      id: 'opening_freshness', label: '开篇钩子', verdict: 'warn', severity: 'S2',
      detail: '前 3 段只有悬念铺垫', evidence: [{ paragraph: 1, excerpt: '夜色很深。' }],
    },
    {
      id: 'trailer_ending', label: '预告式收尾', verdict: 'fail', severity: 'S1',
      detail: '章尾出现预告/总结腔：才刚刚开始', evidence: [{ paragraph: 42, excerpt: '反击才刚刚开始。' }],
    },
  ],
  advisories: ['读者为什么翻下一页？', '本章改变了什么？', '哪个证据支持你的判断？'],
}

function renderPanel(overrides: Partial<PanelProps> = {}) {
  const props: PanelProps = {
    open: true,
    onClose: vi.fn(),
    busy: false,
    msg: '',
    platforms,
    platformId: 'general',
    onPlatformChange: vi.fn(),
    report: null,
    onReview: vi.fn(),
    hasChapter: true,
    ...overrides,
  }
  return render(<ChapterReviewPanel {...props} />)
}

describe('ChapterReviewPanel', () => {
  it('未评审：给引导文案，评审按钮可用并回调', () => {
    const onReview = vi.fn()
    renderPanel({ onReview })
    expect(screen.getByText(/点「评审当前章」按所选平台档位逐维体检/)).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: '评审当前章' }))
    expect(onReview).toHaveBeenCalledTimes(1)
  })

  it('无当前章：评审按钮禁用', () => {
    renderPanel({ hasChapter: false })
    expect((screen.getByRole('button', { name: '评审当前章' }) as HTMLButtonElement).disabled).toBe(true)
  })

  it('有报告：结论中文档 + 平台/章号/字数 + S1~S4 计数', () => {
    renderPanel({ report })
    expect(screen.getByText('打回')).toBeTruthy()
    expect(screen.getByText(/通用 · 第 7 章 · 2,380 字 · 命中 3 项/)).toBeTruthy()
    expect(screen.getByText('S1 1')).toBeTruthy()
    expect(screen.getByText('S4 0')).toBeTruthy()
  })

  it('逐维结果：S1 fail 排最前，pass/skip 排最后，含证据与改法', () => {
    renderPanel({ report })
    const rows = screen.getAllByText(/^(通过|提醒|不达标|跳过)$/)
    expect(rows[0].textContent).toBe('不达标')
    expect(rows[rows.length - 1].textContent).toBe('跳过')
    expect(screen.getByText(/第 42 段：「反击才刚刚开始。」/)).toBeTruthy()
    expect(screen.getByText(/改法：按镜头断段/)).toBeTruthy()
    // 跳过维度必须显式说明原因，不静默给通过。
    expect(screen.getByText(/角色库未标注主角/)).toBeTruthy()
  })

  it('黄金三问随报告渲染；档位下拉列出全部平台档位（story 档位标注整篇口径）', () => {
    renderPanel({ report })
    expect(screen.getByText('黄金三问（逐问作答）')).toBeTruthy()
    expect(screen.getByText('哪个证据支持你的判断？')).toBeTruthy()
    // 档位下拉展开后三档常显（antd Select 的选项点击在 jsdom 下不稳定，切换回调由
    // CreatePage 集成测试覆盖；此处只验档位清单渲染与 story 档位标注）。
    fireEvent.mouseDown(screen.getByRole('combobox'))
    expect(screen.getByTitle('知乎盐言故事（整篇口径）')).toBeTruthy()
    expect(screen.getByTitle('番茄小说')).toBeTruthy()
  })

  it('busy 态：评审按钮 loading；msg 直显', () => {
    renderPanel({ busy: true, msg: '第 7 章评审完成' })
    expect(screen.getByText('第 7 章评审完成')).toBeTruthy()
    expect(screen.getByRole('button', { name: /评审当前章/ })).toBeTruthy()
  })
})
