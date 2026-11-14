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
