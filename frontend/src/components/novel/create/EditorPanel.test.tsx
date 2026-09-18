import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { createRef, useState } from 'react'
import EditorPanel, { type EditorPanelHandle } from './EditorPanel'
import type { OutlineNode } from '../../../types'
import type { ChapterAnnotation } from '../../../gaea/lib/bridge/novel'

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

  it('annotations 供给 → 镜像常驻多 mark（rune→code-unit 换算）；正文编辑后整体退场，locate 光标仍可用', async () => {
    const ref = createRef<EditorPanelHandle>()
    const anns: ChapterAnnotation[] = [
      { type: 'conflict', content: '冲突', pos: 1, length: 2 },
      { type: 'suggestion', content: '尾段', pos: 5, length: 1 },
    ]
    function Harness() {
      const [content, setContent] = useState('𝌆一二三四五')
      return <EditorPanel ref={ref} {...baseProps} content={content} onContentChange={setContent} annotations={anns} />
    }
    render(<Harness />)
    // 镜像常驻（非定位触发）
    await screen.findByTestId('editor-annotation-mirror')
    const marks = screen.getAllByTestId('editor-annotation-mark')
    expect(marks).toHaveLength(2)
    expect(marks[0].textContent).toBe('一二') // rune [1,3) → code-unit [2,4)
    expect(marks[1].textContent).toBe('五') // rune [5,6) → code-unit [6,7)
    // 正文编辑 → dirty → 镜像整体退场
    fireEvent.change(document.querySelector('textarea') as HTMLTextAreaElement, { target: { value: '改动' } })
    await waitFor(() => expect(screen.queryByTestId('editor-annotation-mirror')).toBeNull())
    // locate 光标定位契约不受 dirty 影响
    expect(ref.current?.locate(1, 2)).toBe(true)
    const ta = document.querySelector('textarea') as HTMLTextAreaElement
    expect(ta.selectionStart).toBe(1)
    expect(ta.selectionEnd).toBe(2)
  })

  it('越界标注不产生 mark：无有效段则无镜像', async () => {
    const anns: ChapterAnnotation[] = [{ type: 'conflict', content: '越界', pos: 99, length: 2 }]
    function Harness() {
      const [content] = useState('abc')
      return <EditorPanel {...baseProps} content={content} onContentChange={() => {}} annotations={anns} />
    }
    render(<Harness />)
    await waitFor(() => expect(screen.queryByTestId('editor-annotation-mirror')).toBeNull())
  })
})
