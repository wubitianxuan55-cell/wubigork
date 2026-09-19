// readingOverlays — 阅读浮层的危险动作二次确认（v4.349 可访问性/稳健性刀）
//
// 背景：想法弹窗里「删除高亮」此前是裸 onClick，误点即删（划线笔记不可恢复），
// 而同仓删除类动作一律走 Popconfirm/Modal.confirm。此处改为 Popconfirm 并锁定
// 「未确认不回调」这一契约。
import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import ReadingOverlays, { type ReadingOverlaysProps } from './readingOverlays'
import type { ReadingAnnotation } from '../../utils/readingAnnotations'

const NOTE: ReadingAnnotation = {
  id: 'a1',
  nodeId: 'ch-3',
  title: '第 3 章',
  text: '夜风从窗缝里钻进来',
  color: 'yellow',
  note: '这句好',
  createdAt: 1758000000000,
}

function props(over: Partial<ReadingOverlaysProps> = {}): ReadingOverlaysProps {
  return {
    readMode: false,
    selToolbar: null,
    selText: '',
    onClearSel: vi.fn(),
    onHighlight: vi.fn(),
    onAskSelection: vi.fn(),
    noteTarget: NOTE,
    onNoteClose: vi.fn(),
    noteDraft: '这句好',
    onNoteDraftChange: vi.fn(),
    onSaveNote: vi.fn(),
    onDeleteAnnotation: vi.fn(),
    askTarget: null,
    onAskClose: vi.fn(),
    askMessages: [],
    askLoading: false,
    askError: null,
    askQuestion: '',
    onAskQuestionChange: vi.fn(),
    onRunAsk: vi.fn(),
    onClearAsk: vi.fn(),
    askThreadRef: null,
    ...over,
  }
}

/** 确认 Popconfirm：取弹出层内的确认钮（antd 会给两个汉字按钮插入空格） */
async function confirmPop() {
  await screen.findByText('删除这条高亮？')
  const buttons = document.querySelectorAll<HTMLButtonElement>('.ant-popconfirm-buttons button')
  expect(buttons.length).toBe(2)
  fireEvent.click(buttons[1])
}

describe('readingOverlays 想法弹窗（v4.349）', () => {
  it('「删除高亮」不再裸删：点击只弹确认，未确认不回调', async () => {
    const onDeleteAnnotation = vi.fn()
    render(<ReadingOverlays {...props({ onDeleteAnnotation })} />)

    fireEvent.click(screen.getByRole('button', { name: '删除高亮' }))
    expect(await screen.findByText('删除这条高亮？')).toBeTruthy()
    expect(onDeleteAnnotation).not.toHaveBeenCalled()
  })

  it('确认后才回调 onDeleteAnnotation（且只调一次，带注解 id）', async () => {
    const onDeleteAnnotation = vi.fn()
    render(<ReadingOverlays {...props({ onDeleteAnnotation })} />)

    fireEvent.click(screen.getByRole('button', { name: '删除高亮' }))
    await confirmPop()
    expect(onDeleteAnnotation).toHaveBeenCalledTimes(1)
    expect(onDeleteAnnotation).toHaveBeenCalledWith('a1')
  })

  it('无想法目标时不渲染弹窗内容', () => {
    render(<ReadingOverlays {...props({ noteTarget: null })} />)
    expect(screen.queryByRole('button', { name: '删除高亮' })).toBeNull()
  })
})
