import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import ChatMarkdown from './ChatMarkdown'

// C 批蒸馏测试：codeIndented 禁用（LLM 前导空格列表防御）+ mermaid 围栏双态。
// mermaidRender mock 掉——jsdom 不跑真实 mermaid 渲染，只钉 UI 契约。
vi.mock('../gaea/lib/mermaidRender', () => ({
  ensureMermaid: () => Promise.reject(new Error('jsdom-no-mermaid')),
  mermaidTheme: () => 'default' as const,
}))

describe('ChatMarkdown LLM 输出防御（C2 codeIndented 禁用）', () => {
  it('四空格缩进的列表不再被误判为代码块', () => {
    const { container } = render(<ChatMarkdown text={'    - 甲\n    - 乙'} />)
    expect(container.querySelector('pre')).toBeNull()
    expect(container.querySelector('ul')).not.toBeNull()
    expect(screen.getByText('甲')).toBeTruthy()
    expect(screen.getByText('乙')).toBeTruthy()
  })

  it('围栏代码块仍正常成码块（防御刀不误伤）', () => {
    const { container } = render(<ChatMarkdown text={'```go\nfunc main() {}\n```'} />)
    expect(container.querySelector('pre')).not.toBeNull()
  })
})

describe('ChatMarkdown mermaid 双态（C3）', () => {
  it('mermaid 围栏渲染图表/源码切换，渲染失败可回看源码', async () => {
    render(<ChatMarkdown text={'```mermaid\nflowchart TD\n  A-->B\n```'} />)
    const chartBtn = screen.getByRole('button', { name: '图表' })
    const codeBtn = screen.getByRole('button', { name: '源码' })
    expect(chartBtn).toBeTruthy()
    expect(codeBtn).toBeTruthy()
    // jsdom mock 下渲染必失败 → 状态行提示，可切「源码」看原文
    await screen.findByText('图表渲染失败，可切换「源码」查看')
    fireEvent.click(codeBtn)
    expect(screen.getByText(/flowchart TD/)).toBeTruthy()
    fireEvent.click(chartBtn)
    await screen.findByText('图表渲染失败，可切换「源码」查看')
  })

  it('普通语言围栏仍是纯代码块（无图表切换）', () => {
    render(<ChatMarkdown text={'```python\nprint(1)\n```'} />)
    expect(screen.queryByRole('button', { name: '图表' })).toBeNull()
    expect(screen.queryByRole('button', { name: '源码' })).toBeNull()
  })
})

describe('ChatMarkdown table copy chain', () => {
  it('renders a copy-table action for GFM tables', () => {
    render(<ChatMarkdown text={'| 名称 | 值 |\n| --- | --- |\n| A | 1 |'} />)
    expect(screen.getByRole('button', { name: /复制表格/ })).toBeTruthy()
  })
})
