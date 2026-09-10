import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import NovelSidebar from './NovelSidebar'

vi.mock('./OutlinePanel', () => ({
  default: () => <div data-testid="outline-stub" />,
}))

describe('NovelSidebar 目录', () => {
  it('未开书时引导去书架', () => {
    const onGo = vi.fn()
    render(
      <NovelSidebar
        outlines={[]}
        activeKey=""
        collapsed={false}
        onToggleCollapse={() => {}}
        onOpenChapter={() => {}}
        onGoBookshelf={onGo}
        projectPath=""
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /去书架/ }))
    expect(onGo).toHaveBeenCalled()
  })

  it('已开书不再重复身份卡，只留目录', () => {
    render(
      <NovelSidebar
        outlines={[]}
        activeKey=""
        collapsed={false}
        onToggleCollapse={() => {}}
        onOpenChapter={() => {}}
        onGoBookshelf={() => {}}
        projectPath="C:/novels/fengxue"
      />,
    )
    expect(screen.queryByText('当前小说')).toBeNull()
    expect(screen.getByText('目录')).toBeTruthy()
    expect(screen.getByTestId('outline-stub')).toBeTruthy()
  })
})
