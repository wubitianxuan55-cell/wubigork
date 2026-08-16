import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import EditorPanel from './EditorPanel'
import type { OutlineNode } from '../../../types'

const node: OutlineNode = { id: 'n1', title: '第1章', summary: '', status: 'writing', order_index: 1 }

const baseProps = {
  activeNode: node as OutlineNode | null,
  content: '正文',
  onContentChange: () => {},
  chapterLoading: false,
  generating: false,
  genPhase: '正在生成…',
  genPercent: 10,
  stopping: false,
  saving: false,
  onRegenerate: () => {},
  onSave: () => {},
  onStop: () => {},
  hasChapters: true,
  nextChapterNum: 2,
  onOpenWizard: () => {},
}

describe('EditorPanel 生成状态栏（T6-7.2 停止按钮）', () => {
  it('生成中渲染「停止生成」按钮，点击触发 onStop', () => {
    const onStop = vi.fn()
    render(<EditorPanel {...baseProps} generating onStop={onStop} />)
    const stop = screen.getByRole('button', { name: /停止生成/ })
    fireEvent.click(stop)
    expect(onStop).toHaveBeenCalledTimes(1)
  })

  it('非生成中不渲染停止按钮', () => {
    render(<EditorPanel {...baseProps} generating={false} genPhase="" genPercent={0} />)
    expect(screen.queryByRole('button', { name: /停止生成/ })).toBeNull()
  })

  it('停止请求进行中按钮呈 loading（防止重复点击）', () => {
    render(<EditorPanel {...baseProps} generating stopping />)
    const stop = screen.getByRole('button', { name: /停止生成/ })
    expect(stop.className).toContain('ant-btn-loading')
    expect(stop).toHaveProperty('disabled', false)
  })

  it('字号调节：点击 A+ / A− 回调对应步进', () => {
    const onChange = vi.fn()
    render(<EditorPanel {...baseProps} editorFontSize={15} onEditorFontSizeChange={onChange} />)
    fireEvent.click(screen.getByRole('button', { name: /增大字号/ }))
    expect(onChange).toHaveBeenCalledWith(16)
    fireEvent.click(screen.getByRole('button', { name: /减小字号/ }))
    expect(onChange).toHaveBeenCalledWith(14)
  })

  it('字号到达边界时对应按钮禁用', () => {
    const { rerender } = render(<EditorPanel {...baseProps} editorFontSize={12} />)
    expect(screen.getByRole('button', { name: /减小字号/ })).toHaveProperty('disabled', true)
    expect(screen.getByRole('button', { name: /增大字号/ })).toHaveProperty('disabled', false)
    rerender(<EditorPanel {...baseProps} editorFontSize={24} />)
    expect(screen.getByRole('button', { name: /增大字号/ })).toHaveProperty('disabled', true)
    expect(screen.getByRole('button', { name: /减小字号/ })).toHaveProperty('disabled', false)
  })
})
