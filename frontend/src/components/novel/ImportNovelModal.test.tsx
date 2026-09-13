// ImportNovelModal.test.tsx — 导入成品小说 Modal（v4.287 tail 提取范围出口）。
// 受控 presentational：断言提取范围选项渲染与回调透传，落库语义在 Go 侧锁定。
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import React from 'react'

import ImportNovelModal, { EXTRACT_OPTIONS } from './ImportNovelModal'

function renderModal(overrides: Partial<Parameters<typeof ImportNovelModal>[0]> = {}) {
  const props = {
    open: true,
    fileName: '长书.txt',
    title: '长书',
    genre: [] as string[],
    style: [] as string[],
    importing: false,
    extractRange: 'full',
    onExtractRangeChange: vi.fn(),
    onTitleChange: vi.fn(),
    onGenreChange: vi.fn(),
    onStyleChange: vi.fn(),
    onImport: vi.fn(),
    onClose: vi.fn(),
    ...overrides,
  }
  render(<ImportNovelModal {...props} />)
  return props
}

describe('ImportNovelModal 提取范围（tail 出口）', () => {
  it('选项覆盖全本与 5 的倍数末 N 章（5~50）', () => {
    expect(EXTRACT_OPTIONS[0]).toEqual({ value: 'full', label: '全本' })
    const tails = EXTRACT_OPTIONS.slice(1).map((o) => o.value)
    expect(tails).toEqual(['tail:5', 'tail:10', 'tail:15', 'tail:20', 'tail:25', 'tail:30', 'tail:35', 'tail:40', 'tail:45', 'tail:50'])
  })

  it('选择器渲染当前值（全本），选项值由 EXTRACT_OPTIONS 锁定', () => {
    renderModal()
    const sel = screen.getAllByLabelText('提取范围')[0]
    expect((sel.querySelector('.ant-select-selection-item') as HTMLElement)?.textContent).toBe('全本')
  })

  it('提示语如实说明 tail 语义与 EPUB 限制', () => {
    renderModal()
    expect(screen.getByText(/EPUB 暂仅支持全本/)).toBeTruthy()
  })
})
