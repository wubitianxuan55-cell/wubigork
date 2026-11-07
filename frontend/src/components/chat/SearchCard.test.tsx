import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { SearchCard } from './SearchCard'
import { parseChatSearchSummary } from './searchCardModel'

const searched = {
  state: 'ok',
  queries: ['成都 明天 天气'],
  searches: 1,
  rounds: 2,
  sources: 2,
  elapsed_ms: 1530,
  degraded_reason: '',
  references: [
    {
      title: '成都天气预报',
      url: 'https://weather.example.com/chengdu',
      source: 'tavily',
      snippet: '成都明天多云，14 到 22 度。',
    },
    {
      title: '四川天气',
      url: 'https://news.example.com/sichuan',
      source: 'bing',
    },
  ],
}

describe('SearchCard', () => {
  it('renders a compact card and expands to safe external sources', () => {
    const { container } = render(<SearchCard value={searched} />)
    expect(screen.getByText('联网搜索')).toBeTruthy()
    expect(screen.getByText('1 次检索 · 2 个来源 · 1.5 秒')).toBeTruthy()
    expect(screen.getByText('已完成')).toBeTruthy()

    const details = container.querySelector('details')
    expect(details).toBeTruthy()
    fireEvent.click(container.querySelector('summary')!)
    expect(details!.open).toBe(true)

    const first = screen.getByRole('link', { name: /成都天气预报/ })
    expect(first.getAttribute('href')).toBe('https://weather.example.com/chengdu')
    expect(first.getAttribute('target')).toBe('_blank')
    expect(first.getAttribute('rel')).toBe('noreferrer')
    expect(screen.getByText('成都 明天 天气')).toBeTruthy()
    expect(screen.getByText('成都明天多云，14 到 22 度。')).toBeTruthy()
  })

  it('does not render a card when search was not requested', () => {
    const { container } = render(<SearchCard value={{ ...searched, state: 'not_requested' }} />)
    expect(container.innerHTML).toBe('')
  })

  it('shows degraded state and a readable reason', () => {
    render(<SearchCard value={{
      ...searched,
      state: 'degraded',
      degraded_reason: 'primary-timeout',
      references: [],
      sources: 0,
    }} />)
    expect(screen.getByText('降级结果')).toBeTruthy()
    fireEvent.click(screen.getByText('联网搜索').closest('summary')!)
    expect(screen.getByText('主搜索源超时，当前为备用来源。')).toBeTruthy()
  })

  it('filters non-http reference URLs', () => {
    render(<SearchCard value={{
      ...searched,
      sources: 1,
      references: [{
        title: '危险链接',
        url: 'javascript:alert(1)',
      }],
    }} />)
    fireEvent.click(screen.getByText('联网搜索').closest('summary')!)
    expect(screen.queryByText('危险链接')).toBeNull()
    expect(screen.getByText('本次检索没有可展示的来源。')).toBeTruthy()
  })

  it('parses persisted snake_case fields and rejects malformed values', () => {
    const parsed = parseChatSearchSummary(searched)
    expect(parsed).toMatchObject({
      state: 'ok',
      searches: 1,
      rounds: 2,
      sources: 2,
      elapsedMs: 1530,
    })
    expect(parsed?.references).toHaveLength(2)
    expect(parseChatSearchSummary('bad')).toBeNull()
    expect(parseChatSearchSummary({})).toBeNull()
  })
})
