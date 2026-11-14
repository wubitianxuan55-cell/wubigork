import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { scheduleCommand, type RuntimeScheduleCommand } from '../../schedule/commandRegistry'
import { ScheduleCommandPalette } from './ScheduleCommandPalette'

function command(id: string, run: () => void, disabledReason?: string): RuntimeScheduleCommand {
  const spec = scheduleCommand(id)
  if (!spec) throw new Error(`missing command ${id}`)
  return { spec, run, disabledReason }
}

describe('ScheduleCommandPalette', () => {
  it('按命令名过滤，Enter 执行并关闭', () => {
    const runFind = vi.fn()
    const runMilestone = vi.fn()
    const onClose = vi.fn()
    render(
      <ScheduleCommandPalette
        open
        commands={[command('edit.find', runFind), command('task.milestone', runMilestone)]}
        onClose={onClose}
        onBlocked={vi.fn()}
      />,
    )
    const input = screen.getByTestId('sched-command-search')
    fireEvent.change(input, { target: { value: '查找' } })
    expect(screen.queryByText('切换里程碑')).toBeNull()
    expect(screen.getByText('查找工作')).toBeTruthy()
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(runFind).toHaveBeenCalledOnce()
    expect(runMilestone).not.toHaveBeenCalled()
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('禁用命令保留在结果中、显示原因且不执行', () => {
    const run = vi.fn()
    const onBlocked = vi.fn()
    const onClose = vi.fn()
    render(
      <ScheduleCommandPalette
        open
        commands={[command('task.milestone', run, '请先选中一条工作')]}
        onClose={onClose}
        onBlocked={onBlocked}
      />,
    )
    expect(screen.getByText('请先选中一条工作')).toBeTruthy()
    fireEvent.click(screen.getByText('切换里程碑'))
    expect(run).not.toHaveBeenCalled()
    expect(onBlocked).toHaveBeenCalledWith('请先选中一条工作')
    expect(onClose).not.toHaveBeenCalled()
  })

  it('方向键移动活动项，Enter 执行当前项', () => {
    const first = vi.fn()
    const second = vi.fn()
    render(
      <ScheduleCommandPalette
        open
        commands={[command('view.gantt', first), command('view.pdm', second)]}
        onClose={vi.fn()}
        onBlocked={vi.fn()}
      />,
    )
    const input = screen.getByTestId('sched-command-search')
    fireEvent.keyDown(input, { key: 'ArrowDown' })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(first).not.toHaveBeenCalled()
    expect(second).toHaveBeenCalledOnce()
  })

  it('Esc 从搜索框关闭面板', () => {
    const onClose = vi.fn()
    render(
      <ScheduleCommandPalette
        open
        commands={[command('edit.find', vi.fn())]}
        onClose={onClose}
        onBlocked={vi.fn()}
      />,
    )
    fireEvent.keyDown(screen.getByTestId('sched-command-search'), { key: 'Escape' })
    expect(onClose).toHaveBeenCalledOnce()
  })
})

// ── v3 交互语义（行为线）：combobox 语义与列表键盘 ──────────────────────
describe('ScheduleCommandPalette · 键盘与 combobox 语义（v3）', () => {
  const optionIds = () =>
    Array.from(screen.getByTestId('sched-command-palette').querySelectorAll<HTMLElement>('[role="option"]')).map(
      (el) => el.id,
    )

  it('搜索框是 combobox：aria-controls 指到列表、aria-activedescendant 指到活动项', () => {
    render(
      <ScheduleCommandPalette
        open
        commands={[command('view.gantt', vi.fn()), command('view.pdm', vi.fn())]}
        onClose={vi.fn()}
        onBlocked={vi.fn()}
      />,
    )
    const input = screen.getByTestId('sched-command-search')
    expect(input.getAttribute('role')).toBe('combobox')
    expect(input.getAttribute('aria-autocomplete')).toBe('list')
    expect(input.getAttribute('aria-expanded')).toBe('true')
    const listbox = document.getElementById(input.getAttribute('aria-controls')!)
    expect(listbox).not.toBeNull()
    expect(listbox!.getAttribute('role')).toBe('listbox')
    // 活动项 id 必须真实存在（悬空引用会让读屏读不到当前项）
    const activeId = input.getAttribute('aria-activedescendant')!
    expect(document.getElementById(activeId)).not.toBeNull()
    expect(activeId).toBe(optionIds()[0])
  })

  it('选项不占 Tab 位（tabIndex=-1），Home/End 跳到首/末项', () => {
    render(
      <ScheduleCommandPalette
        open
        commands={[command('view.gantt', vi.fn()), command('view.pdm', vi.fn()), command('edit.find', vi.fn())]}
        onClose={vi.fn()}
        onBlocked={vi.fn()}
      />,
    )
    const input = screen.getByTestId('sched-command-search')
    const ids = optionIds()
    expect(ids.length).toBe(3)
    expect(Array.from(document.querySelectorAll<HTMLElement>('[role="option"]')).every((el) => el.tabIndex === -1)).toBe(true)
    fireEvent.keyDown(input, { key: 'End' })
    expect(input.getAttribute('aria-activedescendant')).toBe(ids[2])
    fireEvent.keyDown(input, { key: 'Home' })
    expect(input.getAttribute('aria-activedescendant')).toBe(ids[0])
  })

  it('焦点落在选项上时 Esc 也关闭面板，且只回调一次', () => {
    const onClose = vi.fn()
    render(
      <ScheduleCommandPalette
        open
        commands={[command('edit.find', vi.fn())]}
        onClose={onClose}
        onBlocked={vi.fn()}
      />,
    )
    const option = screen.getByRole('option')
    option.focus()
    fireEvent.keyDown(option, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('活动项滚动入视用 behavior:auto（不引入动画，reduced-motion 下同语义）', () => {
    const proto = Element.prototype as unknown as { scrollIntoView?: (arg?: unknown) => void }
    const original = proto.scrollIntoView
    const scrollSpy = vi.fn()
    proto.scrollIntoView = scrollSpy
    try {
      render(
        <ScheduleCommandPalette
          open
          commands={[command('view.gantt', vi.fn()), command('view.pdm', vi.fn())]}
          onClose={vi.fn()}
          onBlocked={vi.fn()}
        />,
      )
      expect(scrollSpy).toHaveBeenCalled()
      expect(scrollSpy.mock.calls.at(-1)?.[0]).toEqual({ block: 'nearest', behavior: 'auto' })
      // 换活动项再滚一次，参数保持同一条（不给后续改成 smooth 留口子）
      fireEvent.keyDown(screen.getByTestId('sched-command-search'), { key: 'ArrowDown' })
      expect(scrollSpy.mock.calls.at(-1)?.[0]).toEqual({ block: 'nearest', behavior: 'auto' })
    } finally {
      if (original) proto.scrollIntoView = original
      else delete proto.scrollIntoView
    }
  })
})
