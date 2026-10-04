// CreateNovelModal.test.tsx — 新建小说弹窗（标题/题材/文风是父层 HomePage 受控 state）。
// v4.421.0 收口两条：空标题不再「点 OK 静默无反应」（OK 禁用 + 提交前兜底提示）、
// onCreate 在途不可双击重复提交（本地门闩 + confirmLoading）。
// 坑复训：antd 两字按钮自动插空格（「创 建」）→ 按钮名一律正则。
import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'

import CreateNovelModal from './CreateNovelModal'

type Props = Parameters<typeof CreateNovelModal>[0]

function renderModal(overrides: Partial<Props> = {}) {
  const props: Props = {
    open: true,
    onClose: vi.fn(),
    onCreate: vi.fn().mockResolvedValue(undefined),
    title: '风雪夜归',
    onTitleChange: vi.fn(),
    genre: [],
    onGenreChange: vi.fn(),
    style: [],
    onStyleChange: vi.fn(),
    mature: '',
    onMatureChange: vi.fn(),
    ...overrides,
  }
  const view = render(<CreateNovelModal {...props} />)
  return { props, view }
}

/** OK 按钮（文案在在途态变为「创建中…」，用宽正则抓同一个按钮）。 */
const okButton = () => screen.getByRole('button', { name: /创\s*建/ }) as HTMLButtonElement

describe('CreateNovelModal 新建小说（v4.421.0）', () => {
  it('空标题 / 纯空格标题：OK 按钮禁用，点击不触发 onCreate 也不关窗', () => {
    const { props, view } = renderModal({ title: '' })
    expect(okButton().disabled).toBe(true)

    // 纯空格同样算空标题
    view.rerender(<CreateNovelModal {...props} title="   " />)
    expect(okButton().disabled).toBe(true)

    fireEvent.click(okButton())
    expect(props.onCreate).not.toHaveBeenCalled()
    expect(props.onClose).not.toHaveBeenCalled()

    // 填上标题后恢复可点（props 受控：父层 setState 后回传新 title）
    view.rerender(<CreateNovelModal {...props} title="风雪夜归" />)
    expect(okButton().disabled).toBe(false)
  })

  it('onCreate 在途：OK 转 loading 且禁用，重复点击不重复创建；完成后恢复', async () => {
    let resolveCreate: () => void = () => {}
    const onCreate = vi.fn(() => new Promise<void>((r) => { resolveCreate = r }))
    renderModal({ title: '风雪夜归', onCreate })

    fireEvent.click(okButton())
    await waitFor(() => expect(onCreate).toHaveBeenCalledTimes(1))

    const busy = okButton()
    expect(busy.className).toContain('ant-btn-loading')
    expect(busy.disabled).toBe(true)
    fireEvent.click(busy)
    fireEvent.click(busy)
    await new Promise((r) => setTimeout(r, 0))
    expect(onCreate).toHaveBeenCalledTimes(1)

    await act(async () => { resolveCreate() })
    await waitFor(() => expect(okButton().className).not.toContain('ant-btn-loading'))
    expect(okButton().disabled).toBe(false)
  })

  it('有标题提交：onCreate 恰一次且不代关窗（关闭权在父层 HomePage）', async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined)
    const { props } = renderModal({ onCreate })

    fireEvent.click(okButton())
    await waitFor(() => expect(onCreate).toHaveBeenCalledTimes(1))
    expect(props.onClose).not.toHaveBeenCalled()
  })
})

describe('CreateNovelModal 本书定位（v4.441 建档同址可选）', () => {
  it('选直白档：onMatureChange 收到 explicit', () => {
    const { props } = renderModal()
    fireEvent.click(screen.getByText('成人向 · 直白（亲密戏正面直书）'))
    expect(props.onMatureChange).toHaveBeenCalledWith('explicit')
  })
})
