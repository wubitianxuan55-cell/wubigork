import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { createRef } from 'react'
import EditorPanel, { type EditorPanelHandle } from './EditorPanel'
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

describe('EditorPanel 局部重写入口（t4-C3 余项）', () => {
  const getEditorTextarea = (): HTMLTextAreaElement =>
    screen.getByPlaceholderText(/AI 将在此流式呈现正文/) as HTMLTextAreaElement

  it('无选区点击「局部重写」：message.warning 提示且不回调', async () => {
    const onPartialRewrite = vi.fn()
    render(<EditorPanel {...baseProps} onPartialRewrite={onPartialRewrite} />)
    fireEvent.click(screen.getByRole('button', { name: /局部重写/ }))
    expect(await screen.findByText('请先选中要重写的文字段落')).toBeTruthy()
    expect(onPartialRewrite).not.toHaveBeenCalled()
  })

  it('有选区：回调 rune 偏移（代理对 𝒜 占 2 code-unit/1 rune，code-unit 选区换算）', () => {
    const onPartialRewrite = vi.fn()
    // '𝒜' 为星面字符（surrogate pair）：'a𝒜b文' = 5 code-unit / 4 rune
    const content = 'a𝒜b文'
    render(<EditorPanel {...baseProps} content={content} onPartialRewrite={onPartialRewrite} />)
    const ta = getEditorTextarea()
    // code-unit 选区 [3,5) 选中「b文」；rune 偏移应为 [2,4)
    ta.setSelectionRange(3, 5)
    fireEvent.click(screen.getByRole('button', { name: /局部重写/ }))
    expect(onPartialRewrite).toHaveBeenCalledTimes(1)
    expect(onPartialRewrite).toHaveBeenCalledWith({ start: 2, end: 4, text: 'b文' })
  })
})

describe('EditorPanel 标注定位（t7 观察池：locate 命令句柄）', () => {
  it('locate 按 rune 偏移换算 code-unit 选区并聚焦（𝌆 代理对=2 code-unit/1 rune）', () => {
    const ref = createRef<EditorPanelHandle>()
    // '𝌆一二三四五'：rune0=𝌆(2 code-unit)、rune1..5=一二三四五 → rune [1,3) = code-unit [2,4)
    render(<EditorPanel ref={ref} {...baseProps} content="𝌆一二三四五" />)
    const ta = document.querySelector('textarea') as HTMLTextAreaElement
    expect(ta).toBeTruthy()
    expect(ref.current?.locate(1, 3)).toBe(true)
    expect(ta.selectionStart).toBe(2)
    expect(ta.selectionEnd).toBe(4)
    expect(document.activeElement).toBe(ta)
  })

  it('runeStart 越界钳到文末：选区为空不越界', () => {
    const ref = createRef<EditorPanelHandle>()
    render(<EditorPanel ref={ref} {...baseProps} content="abc" />)
    const ta = document.querySelector('textarea') as HTMLTextAreaElement
    expect(ref.current?.locate(99, 100)).toBe(true)
    expect(ta.selectionStart).toBe(3)
    expect(ta.selectionEnd).toBe(3)
  })

  it('无激活章返回 false（编辑器空态未渲染 textarea）', () => {
    const ref = createRef<EditorPanelHandle>()
    render(<EditorPanel ref={ref} {...baseProps} activeNode={null} content="" />)
    expect(ref.current?.locate(0, 1)).toBe(false)
  })
})
