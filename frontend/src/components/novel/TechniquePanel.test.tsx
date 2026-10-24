// TechniquePanel.test.tsx — 技法库面板（webnovel-writer 九表思想蒸馏批 N11）：
// 打开拉分类目录初始态 / 检索命中渲染（摘要+展开+毒点）/ 零命中空态 / 分类 Tag 点击即检索。
// 坑复训：同族短按钮名用 ^…$ 整名锚定；Tag 文本断言用精确全文。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  search: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      NovelTechniqueSearch: mocks.search,
    },
  }
})

import TechniquePanel from './TechniquePanel'

const nineCats = [
  { category: '场景写法', count: 4 },
  { category: '爽点节奏', count: 4 },
  { category: '桥段套路', count: 4 },
  { category: '人设关系', count: 3 },
  { category: '写作技法', count: 4 },
  { category: '命名规则', count: 2 },
  { category: '裁决规则', count: 2 },
  { category: '金手指', count: 2 },
  { category: '题材调性', count: 3 },
]

const hit = {
  id: 'payoff-face-slap',
  category: '爽点节奏',
  keywords: ['打脸', '退婚'],
  genres: [],
  summary: '打脸三拍：先压后扬再踩实',
  detail: '压制期越具体打脸越爽。',
  pitfall: '压制期一笔带过直接打脸，爽感没有落差。',
}

const setQuery = (value: string) => {
  const el = screen.getByTestId('technique-query')
  const input = (el.tagName === 'INPUT' ? el : el.querySelector('input')) as HTMLInputElement
  fireEvent.change(input, { target: { value } })
}

function renderPanel() {
  return render(<TechniquePanel open onClose={vi.fn()} />)
}

describe('TechniquePanel 技法库（webnovel-writer 九表思想蒸馏批 N11）', () => {
  beforeEach(() => {
    mocks.search.mockReset()
    mocks.search.mockResolvedValue({ results: [], categories: nineCats, total: 28 })
  })

  it('打开即拉分类目录初始态（空查询），九类 Tag 显形', async () => {
    renderPanel()
    await waitFor(() => expect(mocks.search).toHaveBeenCalledWith(''))
    expect(await screen.findByTestId('technique-cats')).toBeTruthy()
    expect(screen.getByText('场景写法·4')).toBeTruthy()
    expect(screen.getByText('金手指·2')).toBeTruthy()
  })

  it('检索命中：摘要+展开+毒点三件套渲染；绑定按输入词调用', async () => {
    renderPanel()
    await screen.findByTestId('technique-cats') // 初始态先拉完，避免消费下方一次性 mock
    mocks.search.mockResolvedValueOnce({ results: [hit], categories: nineCats, total: 28 })

    setQuery('打脸')
    fireEvent.click(screen.getByTestId('technique-search'))
    await waitFor(() => expect(mocks.search).toHaveBeenLastCalledWith('打脸'))

    const view = await screen.findByTestId('technique-results')
    expect(view.textContent).toContain('打脸三拍：先压后扬再踩实')
    expect(view.textContent).toContain('毒点：压制期一笔带过直接打脸')
    expect(view.textContent).toContain('爽点节奏')
  })

  it('零命中出诚实空态，不出假结果区', async () => {
    renderPanel()
    await screen.findByTestId('technique-cats')
    setQuery('无关词')
    fireEvent.click(screen.getByTestId('technique-search'))
    expect(await screen.findByTestId('technique-empty')).toBeTruthy()
    expect(screen.queryByTestId('technique-results')).toBeNull()
  })

  it('检索失败如实上屏', async () => {
    renderPanel()
    await screen.findByTestId('technique-cats') // 初始态先拉完，失败 mock 只给检索动作
    mocks.search.mockRejectedValueOnce(new Error('绑定未注册'))
    setQuery('打脸')
    fireEvent.click(screen.getByTestId('technique-search'))
    const err = await screen.findByTestId('technique-error')
    expect(err.textContent).toContain('绑定未注册')
  })

  it('分类 Tag 点击=按分类名检索', async () => {
    renderPanel()
    await screen.findByTestId('technique-cats')
    fireEvent.click(screen.getByText('桥段套路·4'))
    await waitFor(() => expect(mocks.search).toHaveBeenLastCalledWith('桥段套路'))
  })
})
