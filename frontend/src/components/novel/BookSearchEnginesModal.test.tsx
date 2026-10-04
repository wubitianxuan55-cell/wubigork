// BookSearchEnginesModal.test.tsx — 泛搜索引擎规则编辑器（t3 余项）。
// bridge mock：Get 返回夹具清单断言渲染；逐字段编辑/启停/新增/删除后保存，
// 断言保存载荷整体替换且高级字段（linkParam 等）原样保留；保存失败如实透出。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { Modal } from 'antd'

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

// ── v4.425 B8：关闭即丢未保存编辑 → 走脏闸；保存在途禁止关闭 ──
describe('搜索引擎规则编辑器 关闭脏闸（v4.425 B8）', () => {
  afterEach(async () => {
    Modal.destroyAll()
    await new Promise((r) => setTimeout(r, 0))
  })

  const confirms = () => Array.from(document.querySelectorAll<HTMLElement>('.ant-modal-confirm'))

  it('有未保存编辑时点关闭：先弹确认；取消则不关且编辑保留', async () => {
    const { onClose } = renderModal()
    await screen.findByDisplayValue('bing')
    fireEvent.change(screen.getByLabelText('结果条目选择器 1'), { target: { value: '.b_algo.new' } })

    const before = confirms().length
    fireEvent.click(document.querySelector('.ant-modal-close') as HTMLElement)
    await waitFor(() => expect(confirms().length).toBeGreaterThan(before))
    const scoped = within(confirms()[confirms().length - 1])
    expect(scoped.getAllByText(/引擎规则有未保存的修改/).length).toBeGreaterThan(0)

    fireEvent.click(scoped.getByRole('button', { name: /取\s*消/ }))
    await new Promise((r) => setTimeout(r, 0))
    expect(onClose).not.toHaveBeenCalled()
    expect(screen.getByDisplayValue('.b_algo.new')).toBeTruthy()
  })

  it('未编辑时点关闭：不弹确认、直接关闭', async () => {
    const { onClose } = renderModal()
    await screen.findByDisplayValue('bing')

    const before = confirms().length
    fireEvent.click(document.querySelector('.ant-modal-close') as HTMLElement)
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(confirms().length).toBe(before)
  })

  it('保存在途：关闭被拒（不关窗、不弹确认），保存完成后才走正常关闭', async () => {
    let release: () => void = () => {}
    enginesSave.mockImplementationOnce(
      () => new Promise<number>((res) => { release = () => res(2) }) as never,
    )
    const { onClose } = renderModal()
    await screen.findByDisplayValue('bing')

    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }))
    await waitFor(() => expect(enginesSave).toHaveBeenCalledTimes(1))

    const before = confirms().length
    // antd 在 closable=false 时整体不渲染叉号 → 直接断言该开关确实关掉了关闭入口
    expect(document.querySelector('.ant-modal-close')).toBeNull()
    expect(onClose).not.toHaveBeenCalled()
    expect(confirms().length).toBe(before)

    act(() => { release() })
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
  })
})
