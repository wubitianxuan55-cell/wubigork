import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import ProjectCardItem from './ProjectCardItem'
import type { ProjectCard } from '../stores/appStore'

const card: ProjectCard = {
  title: '风雪夜归',
  genre: '玄幻',
  style: '史诗',
  path: 'C:/novels/fengxue',
  word_count: 12000,
  chapter_count: 8,
  created_at: '2026-09-01T00:00:00Z',
  last_opened_at: '2026-09-10T00:00:00Z',
}

describe('ProjectCardItem 书房书封', () => {
  it('封面带书脊；正在编辑时打角标', () => {
    const { container } = render(
      <ProjectCardItem
        card={card}
        isActive
        onOpen={() => {}}
        onDelete={() => {}}
      />,
    )
    expect(container.querySelector('.novel-cover-spine')).toBeTruthy()
    expect(screen.getByText('正在编辑')).toBeTruthy()
    expect(container.querySelector('.novel-cover-tone-fantasy')).toBeTruthy()
  })

  it('继续阅读走独立回调，不冒泡成打开', () => {
    const onOpen = vi.fn()
    const onContinue = vi.fn()
    const onDelete = vi.fn()
    render(
      <ProjectCardItem
        card={card}
        isActive={false}
        readingChapter="第3章 · 夜色"
        onOpen={onOpen}
        onContinueReading={onContinue}
        onDelete={onDelete}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /继续阅读/ }))
    expect(onContinue).toHaveBeenCalledWith(card)
    expect(onOpen).not.toHaveBeenCalled()
  })
})
