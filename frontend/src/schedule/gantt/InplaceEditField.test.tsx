/**
 * gantt/InplaceEditField.test.tsx — 统一就地编辑弹层（W5，§15.2 六步手感）
 *
 * 用例口径：双击才开（未打开不渲染内容）/ 打开即聚焦+全选 / Enter 提交（经 inplaceEdit
 * 规范化：trim、夹取）/ Esc 取消不写回 / 失焦提交（≈ editingFinished）/ 非法值原位
 * 提示且不写回 / 日期区间整区间提交。
 */
import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { InplaceEditField } from './InplaceEditField'

/** 打开弹层并取内部输入框（弹层内容渲染在 portal 里，走 screen 查询） */
function openField(triggerText: string, testId: string): { root: HTMLElement; input: HTMLInputElement } {
  fireEvent.doubleClick(screen.getByText(triggerText))
  const root = screen.getByTestId(testId)
  const input = root.querySelector('input') as HTMLInputElement
  return { root, input }
}

describe('InplaceEditField · 打开与聚焦（§15.2 第 4 步）', () => {
  it('未打开不渲染内容；双击打开后输入框获得焦点且内容全选', () => {
    render(
      <InplaceEditField kind="text" value="旧名称" onSubmit={vi.fn()} testId="f1">
        <span>旧名称</span>
      </InplaceEditField>,
    )
    expect(screen.queryByTestId('f1')).toBeNull()
    const { input } = openField('旧名称', 'f1')
    expect(input.value).toBe('旧名称')
    expect(document.activeElement).toBe(input)
    expect(input.selectionStart).toBe(0)
    expect(input.selectionEnd).toBe('旧名称'.length)
  })

  it('autoSelect=false 时只聚焦不全选', () => {
    render(
      <InplaceEditField kind="text" value="abc" autoSelect={false} onSubmit={vi.fn()} testId="f2">
        <span>abc</span>
      </InplaceEditField>,
    )
    const { input } = openField('abc', 'f2')
    expect(document.activeElement).toBe(input)
    // 未全选：选区起点停在文本末尾（全选时起点会是 0）
    expect(input.selectionStart).toBe(input.value.length)
  })
})

describe('InplaceEditField · 提交与取消', () => {
  it('Enter 提交：经 parseInplace trim 后回调，并关闭弹层', () => {
    const onSubmit = vi.fn()
    render(
      <InplaceEditField kind="text" value="旧" onSubmit={onSubmit} testId="f3">
        <span>旧</span>
      </InplaceEditField>,
    )
    const { input } = openField('旧', 'f3')
    fireEvent.change(input, { target: { value: '  场地三通一平  ' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onSubmit).toHaveBeenCalledTimes(1)
    expect(onSubmit).toHaveBeenCalledWith('场地三通一平')
  })

  it('Esc 取消：不写回，只回调 onCancel 并关闭', () => {
    const onSubmit = vi.fn()
    const onCancel = vi.fn()
    const onChange = vi.fn()
    render(
      <InplaceEditField kind="text" value="旧" onSubmit={onSubmit} onCancel={onCancel} onOpenChange={onChange} testId="f4">
        <span>旧</span>
      </InplaceEditField>,
    )
    const { input } = openField('旧', 'f4')
    fireEvent.change(input, { target: { value: '改了不要' } })
    fireEvent.keyDown(input, { key: 'Escape' })
    expect(onSubmit).not.toHaveBeenCalled()
    expect(onCancel).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenLastCalledWith(false)
  })

  it('失焦提交（≈ editingFinished）：focusout 到弹层外即写回', () => {
    const onSubmit = vi.fn()
    render(
      <InplaceEditField kind="text" value="旧" onSubmit={onSubmit} testId="f5">
        <span>旧</span>
      </InplaceEditField>,
    )
    const { input } = openField('旧', 'f5')
    fireEvent.change(input, { target: { value: '新名' } })
    fireEvent.focusOut(input, { relatedTarget: document.body })
    expect(onSubmit).toHaveBeenCalledWith('新名')
  })

  it('数值夹取：超上限按列规格夹到 max 再写回', () => {
    const onSubmit = vi.fn()
    render(
      <InplaceEditField kind="number" value="999" spec={{ min: 0, max: 100, step: 1 }} onSubmit={onSubmit} testId="f6">
        <span>999</span>
      </InplaceEditField>,
    )
    const { input } = openField('999', 'f6')
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onSubmit).toHaveBeenCalledWith(100)
  })

  it('日期区间：整区间原样提交（from/to），不拆成两次写回', () => {
    const onSubmit = vi.fn()
    render(
      <InplaceEditField
        kind="dateRange"
        value={{ from: '2026-09-01', to: '2026-09-30' }}
        onSubmit={onSubmit}
        testId="f7"
      >
        <span>区间</span>
      </InplaceEditField>,
    )
    const { input } = openField('区间', 'f7')
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onSubmit).toHaveBeenCalledWith({ from: '2026-09-01', to: '2026-09-30' })
  })
})

describe('InplaceEditField · 非法值原位提示', () => {
  it('日历不存在的日期：Enter 不写回，弹层内出提示且保持打开', () => {
    const onSubmit = vi.fn()
    render(
      <InplaceEditField kind="date" value="2026-02-30" onSubmit={onSubmit} testId="f8">
        <span>坏日期</span>
      </InplaceEditField>,
    )
    const { root, input } = openField('坏日期', 'f8')
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onSubmit).not.toHaveBeenCalled()
    const err = within(root).getByTestId('f8-error')
    expect(err.textContent).toBe('日期需为 YYYY-MM-DD 格式且日历存在')
    // 仍然打开（用户可改），不静默关掉
    expect(screen.getByTestId('f8')).not.toBeNull()
  })

  it('日期区间 from > to：不写回并提示顺序', () => {
    const onSubmit = vi.fn()
    render(
      <InplaceEditField
        kind="dateRange"
        value={{ from: '2026-10-01', to: '2026-09-30' }}
        onSubmit={onSubmit}
        testId="f9"
      >
        <span>倒序区间</span>
      </InplaceEditField>,
    )
    const { root, input } = openField('倒序区间', 'f9')
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onSubmit).not.toHaveBeenCalled()
    expect(within(root).getByTestId('f9-error').textContent).toBe('开始日期不能晚于结束日期')
  })
})
