import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { BranchSwitcher } from './BranchSwitcher'

// A 批版本切换器 UI 契约：‹ i/n ›，边界禁用，回调携带兄弟消息 id。
describe('BranchSwitcher', () => {
  it('单版本不渲染', () => {
    const { container } = render(<BranchSwitcher ids={[7]} index={0} disabled={false} onSelect={vi.fn()} />)
    expect(container.firstChild).toBeNull()
  })

  it('多版本：i/n 展示、边界禁用、回调带兄弟 id', () => {
    const onSelect = vi.fn()
    const { rerender } = render(<BranchSwitcher ids={[10, 11, 12]} index={1} disabled={false} onSelect={onSelect} />)
    expect(screen.getByText('2 / 3')).toBeTruthy()
    fireEvent.click(screen.getByLabelText('上一版本'))
    expect(onSelect).toHaveBeenCalledWith(10)
    fireEvent.click(screen.getByLabelText('下一版本'))
    expect(onSelect).toHaveBeenCalledWith(12)
    // 边界禁用（同一实例 rerender，避免多实例同名 aria-label 串扰）
    rerender(<BranchSwitcher ids={[10, 11]} index={0} disabled={false} onSelect={onSelect} />)
    expect((screen.getByLabelText('上一版本') as HTMLButtonElement).disabled).toBe(true)
    rerender(<BranchSwitcher ids={[10, 11]} index={0} disabled onSelect={onSelect} />)
    expect((screen.getByLabelText('下一版本') as HTMLButtonElement).disabled).toBe(true)
  })
})
