import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MarkdownContent } from './MarkdownContent'

describe('MarkdownContent GFM 渲染', () => {
  it('渲染 GFM 表格', () => {
    render(
      <MarkdownContent
        source={'| 角色 | 定位 |\n| --- | --- |\n| 林晚 | 主角 |\n| 顾辞 | 反派 |'}
      />,
    )
    const table = document.querySelector('table')
    expect(table).toBeTruthy()
    expect(screen.getByText('角色')).toBeTruthy()
    expect(screen.getByText('林晚')).toBeTruthy()
    expect(screen.getByText('顾辞')).toBeTruthy()
  })

  it('渲染标题与列表等基础语法', () => {
    render(<MarkdownContent source={'# 世界观\n\n- A\n- B'} />)
    expect(screen.getByRole('heading', { level: 1, name: '世界观' })).toBeTruthy()
    expect(screen.getByText('A')).toBeTruthy()
    expect(screen.getByText('B')).toBeTruthy()
  })
})
