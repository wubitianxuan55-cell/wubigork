import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { SearchPanel } from './SearchPanel'
import type { ChatSearchReference } from './searchCardModel'

// D 批来源面板契约：全量列表渲染（站点名/标题/摘要/编号）、Esc 关闭、空态。
const refs: ChatSearchReference[] = [
  { title: 'DeepSeek 发布 V3.2', url: 'https://api-docs.deepseek.com/zh-cn/news/v3.2', source: 'api-docs.deepseek.com', snippet: '正式发布 V3.2 版本' },
  { title: 'javascript:alert(1)', url: 'javascript:alert(1)' },
  { title: '坏协议', url: 'ftp://example.com/x' },
]

describe('SearchPanel', () => {
  it('渲染有效来源（跳过非 http/https），编号从 1 起', () => {
    render(<SearchPanel data={{ refs, queries: ['deepseek v3.2'] }} onClose={vi.fn()} />)
    expect(screen.getByText('搜索来源（1）')).toBeTruthy()
    expect(screen.getByText('DeepSeek 发布 V3.2')).toBeTruthy()
    expect(screen.getByText('正式发布 V3.2 版本')).toBeTruthy()
    expect(screen.getByText('deepseek v3.2')).toBeTruthy()
    expect(screen.queryByText('坏协议')).toBeNull()
    expect(screen.getByText('1')).toBeTruthy()
  })

  it('Esc 关闭、点关闭钮回调', () => {
    const onClose = vi.fn()
    render(<SearchPanel data={{ refs: refs.slice(0, 1), queries: [] }} onClose={onClose} />)
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByLabelText('关闭来源面板'))
    expect(onClose).toHaveBeenCalledTimes(2)
  })

  it('无来源空态；data=null 整面板不渲染', () => {
    const { container, rerender } = render(<SearchPanel data={{ refs: [], queries: [] }} onClose={vi.fn()} />)
    expect(screen.getByText('本轮没有可展示的来源')).toBeTruthy()
    rerender(<SearchPanel data={null} onClose={vi.fn()} />)
    expect(container.firstChild).toBeNull()
  })
})
