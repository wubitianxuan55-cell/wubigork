/**
 * ScheduleDock.test.tsx — ZPert 式右舷预警信息（无额外分类/算法控件）
 */
import { fireEvent, render, screen } from '@testing-library/react'
import type React from 'react'
import { describe, expect, it, vi } from 'vitest'
import { ScheduleDock } from './ScheduleDock'
import type { TaskWarn, WarnSummary } from './warn'

const warnItem = (extra: Partial<TaskWarn>): TaskWarn => ({
  taskId: 'A',
  name: '土方开挖',
  isMilestone: false,
  level: 'red',
  overdue: 8,
  band: '0-15天',
  compareFinish: 18,
  askFinish: 10,
  algorithm: 'plan-ask',
  ...extra,
})

const warn: WarnSummary = {
  red: 1,
  green: 1,
  unassigned: 0,
  items: [
    warnItem({}),
    warnItem({
      taskId: 'B',
      name: '主体封顶',
      isMilestone: true,
      level: 'green',
      overdue: -2,
      band: null,
      compareFinish: 20,
      askFinish: 22,
    }),
  ],
}

function renderDock(overrides: Partial<React.ComponentProps<typeof ScheduleDock>> = {}) {
  const props: React.ComponentProps<typeof ScheduleDock> = {
    open: true,
    warn,
    onNavigate: vi.fn(),
    onClose: vi.fn(),
    ...overrides,
  }
  return { ...render(<ScheduleDock {...props} />), props }
}

describe('ScheduleDock', () => {
  it('只展示预警信息与条目，不带分类页签或算法开关', () => {
    renderDock()
    expect(screen.getByTestId('sched-dock').textContent).toContain('预警信息')
    expect(screen.getByTestId('sched-dock').textContent).toContain('滞后 8 天 · 0-15天')
    expect(screen.getByTestId('sched-dock').textContent).toContain('提前 2 天')
    expect(screen.queryByRole('tab')).toBeNull()
    expect(screen.queryByText('预测-要求')).toBeNull()
  })

  it('点击预警定位任务；关闭按钮走单一状态', () => {
    const { props } = renderDock()
    fireEvent.click(screen.getByTestId('sched-dock-warn-A'))
    expect(props.onNavigate).toHaveBeenCalledWith('A')
    fireEvent.click(screen.getByRole('button', { name: '关闭预警信息' }))
    expect(props.onClose).toHaveBeenCalled()
  })

  it('空预警展示未下达数量；open=false 不挂载', () => {
    const { unmount } = renderDock({ warn: { red: 0, green: 0, unassigned: 15, items: [] } })
    expect(screen.getByTestId('sched-dock').textContent).toContain('15 项未下达要求完成')
    unmount()
    renderDock({ open: false })
    expect(screen.queryByTestId('sched-dock')).toBeNull()
  })
})
