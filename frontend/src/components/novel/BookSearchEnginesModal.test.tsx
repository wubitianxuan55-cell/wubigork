// BookSearchEnginesModal.test.tsx — 泛搜索引擎规则编辑器（t3 余项）。
// bridge mock：Get 返回夹具清单断言渲染；逐字段编辑/启停/新增/删除后保存，
// 断言保存载荷整体替换且高级字段（linkParam 等）原样保留；保存失败如实透出。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import React from 'react'

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      NovelBookSourceEnginesGet: vi.fn(),
      NovelBookSourceEnginesSave: vi.fn(),
    },
  }
})

import BookSearchEnginesModal from './BookSearchEnginesModal'
import { app } from '../../gaea/lib/bridge'

const enginesGet = vi.mocked(app.NovelBookSourceEnginesGet)
const enginesSave = vi.mocked(app.NovelBookSourceEnginesSave)

const fixture = [
  { name: 'bing', url: 'https://www.bing.com/search?q=%s', queryFormat: '%s 小说 免费阅读', result: '.b_algo', title: 'h2 a' },
  { name: 'ddg-html', url: 'https://html.duckduckgo.com/html/?q=%s', result: '.result', title: '.result__a', linkParam: 'uddg', disabled: true },
]

function renderModal() {
  const onClose = vi.fn()
  render(<BookSearchEnginesModal open onClose={onClose} />)
  return { onClose }
}

beforeEach(() => {
  vi.clearAllMocks()
  enginesGet.mockResolvedValue({ path: 'C:/cfg/gaea/booksource/rules/websearch-engines.json', rules: fixture })
})

describe('搜索引擎规则编辑器', () => {
  it('渲染清单：名称/地址/启停与路径', async () => {
    renderModal()
    expect(await screen.findByDisplayValue('bing')).toBeTruthy()
    expect(screen.getByDisplayValue('https://html.duckduckgo.com/html/?q=%s')).toBeTruthy()
    expect(screen.getByText(/规则文件：C:/)).toBeTruthy()
    // ddg 停用态：开关未选中
    const sw = screen.getByLabelText('引擎启用开关 ddg-html')
    expect(sw.getAttribute('aria-checked')).toBe('false')
  })

  it('编辑 + 启停 + 保存：载荷整体替换并保留高级字段', async () => {
    const { onClose } = renderModal()
    enginesSave.mockResolvedValue(2)
    await screen.findByDisplayValue('bing')

    fireEvent.change(screen.getByLabelText('结果条目选择器 1'), { target: { value: '.b_algo.new' } })
    fireEvent.click(screen.getByLabelText('引擎启用开关 ddg-html')) // 停用→启用
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }))

    await waitFor(() => expect(enginesSave).toHaveBeenCalledTimes(1))
    const sent = JSON.parse(enginesSave.mock.calls[0][0]) as Array<Record<string, unknown>>
    expect(sent[0].result).toBe('.b_algo.new')
    expect(sent[1].disabled).toBe(false)
    expect(sent[1].linkParam).toBe('uddg') // 高级字段不丢
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })

  it('保存失败（fail-closed 校验）如实透出不关窗', async () => {
    const { onClose } = renderModal()
    enginesSave.mockRejectedValue(new Error('第 1 条: 搜索引擎规则缺少 name'))
    await screen.findByDisplayValue('bing')

    fireEvent.change(screen.getByLabelText('引擎名称 1'), { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }))

    await waitFor(() => expect(enginesSave).toHaveBeenCalledTimes(1))
    expect(onClose).not.toHaveBeenCalled()
  })

  it('新增与删除行', async () => {
    renderModal()
    await screen.findByDisplayValue('bing')

    fireEvent.click(screen.getByRole('button', { name: /新增引擎/ }))
    expect(screen.getByDisplayValue('新引擎 3')).toBeTruthy()

    fireEvent.click(screen.getByLabelText('删除引擎 1'))
    expect(screen.queryByDisplayValue('bing')).toBeNull()
  })
})
