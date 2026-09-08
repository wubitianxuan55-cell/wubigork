// TaskInspector.test.tsx — 任务检查器（对标 MS Project Task Inspector）UI 用例
//
// 纯受控展示件：任务/CPM 行/依赖行全部由用例造数传入，不依赖 store 与排程引擎。
// 覆盖：概要与六时参换算（MM-DD (+N)）、单驱动/多驱动结论、手动锁定、零前驱、
// 后继表渲染、关闭回调、里程碑工期标签。antd Drawer 渲染在 portal，
// 统一用 document.body 作用域查询。

import { fireEvent, render, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { TaskInspector } from './TaskInspector'
import type { InspectorDepRow, TaskInspectorProps } from './TaskInspector'
import type { SchedTask, TaskCpm } from './types'

// 固定开工日 2026-09-07（周一）：工作日序号可手算（wd0=09-07、wd3=09-10、wd4=09-11、wd8=09-17、wd9=09-18）
const START = '2026-09-07'

function mkTask(extra: Partial<SchedTask> = {}): SchedTask {
  return { id: 'B', name: '主体施工', duration: 5, level: 1, progress: 40, ...extra }
}
function mkRow(extra: Partial<TaskCpm> = {}): TaskCpm {
  return { es: 3, ef: 8, ls: 4, lf: 9, tf: 1, ff: 0, critical: false, ...extra }
}
function mkDep(otherId: string, otherName: string, extra: Partial<InspectorDepRow> = {}): InspectorDepRow {
  return {
    link: { from: otherId, to: 'B', type: 'FS', lag: 0 },
    otherId,
    otherName,
    freeSlack: 0,
    driving: false,
    ...extra,
  }
}

function baseProps(extra: Partial<TaskInspectorProps> = {}): TaskInspectorProps {
  return {
    open: true,
    task: mkTask(),
    row: mkRow(),
    wbs: '1.2',
    predecessors: [mkDep('A', '基础开挖', { link: { from: 'A', to: 'B', type: 'FS', lag: 2 }, driving: true })],
    successors: [],
    startDate: START,
    onClose: vi.fn(),
    onNavigate: vi.fn(),
    ...extra,
  }
}

// Drawer 挂在 body portal，用 document.body 作用域做查询
const ui = () => within(document.body)

describe('TaskInspector 任务检查器', () => {
  it('打开时渲染概要与六时参：日期按工作日历换算 MM-DD (+N)，TF/FF 显工作日数', () => {
    render(<TaskInspector {...baseProps()} />)
    expect(document.body.querySelector('.ant-drawer-title')?.textContent).toBe('主体施工')
    const summary = ui().getByTestId('sched-inspector-summary').textContent ?? ''
    expect(summary).toContain('1.2') // WBS
    expect(summary).toContain('5 天') // 工期
    expect(summary).toContain('自动') // 模式
    expect(summary).toContain('40%') // 进度
    expect(summary).toContain('非关键') // 关键 Tag
    // wd3=周四 09-10、wd8=周四 09-17、wd4=周五 09-11、wd9=周五 09-18
    expect(ui().getByTestId('sched-inspector-es').textContent).toBe('09-10 (+3)')
    expect(ui().getByTestId('sched-inspector-ef').textContent).toBe('09-17 (+8)')
    expect(ui().getByTestId('sched-inspector-ls').textContent).toBe('09-11 (+4)')
    expect(ui().getByTestId('sched-inspector-lf').textContent).toBe('09-18 (+9)')
    expect(ui().getByTestId('sched-inspector-tf').textContent).toBe('1')
    expect(ui().getByTestId('sched-inspector-ff').textContent).toBe('0')
  })

  it('单驱动前驱：结论含前置名与「驱动」；前置表行有驱动 Tag；点名跳转 otherId', () => {
    const onNavigate = vi.fn()
    render(<TaskInspector {...baseProps({ onNavigate })} />)
    const verdict = ui().getByTestId('sched-inspector-verdict')
    expect(verdict.textContent).toContain('基础开挖')
    expect(verdict.textContent).toContain('驱动')
    expect(ui().getByTestId('sched-inspector-preds').textContent).toContain('驱动')
    fireEvent.click(verdict.querySelector('button')!)
    expect(onNavigate).toHaveBeenCalledTimes(1)
    expect(onNavigate).toHaveBeenCalledWith('A')
  })

  it('多条驱动前置：结论为「与 N 条前置同时驱动」', () => {
    render(
      <TaskInspector
        {...baseProps({
          predecessors: [
            mkDep('A', '基础开挖', { driving: true }),
            mkDep('A2', '桩基', { link: { from: 'A2', to: 'B', type: 'SS', lag: 0 }, driving: true }),
          ],
        })}
      />,
    )
    expect(ui().getByTestId('sched-inspector-verdict').textContent).toContain('2 条前置同时驱动')
  })

  it('手动任务：结论=手动锁定（即使存在驱动前驱）', () => {
    render(<TaskInspector {...baseProps({ task: mkTask({ mode: 'manual', manualStart: 5 }) })} />)
    expect(ui().getByTestId('sched-inspector-verdict').textContent).toContain('手动锁定')
  })

  it('零前驱：结论=开工日开始且前置表有占位；后继表正常（类型时距 SS-1）', () => {
    render(
      <TaskInspector
        {...baseProps({
          predecessors: [],
          successors: [mkDep('C', '装饰装修', { link: { from: 'B', to: 'C', type: 'SS', lag: -1 }, freeSlack: 2 })],
        })}
      />,
    )
    expect(ui().getByTestId('sched-inspector-verdict').textContent).toContain('无前置')
    expect(ui().getByTestId('sched-inspector-preds').textContent).toContain('无前置任务')
    const succ = ui().getByTestId('sched-inspector-succs')
    expect(succ.textContent).toContain('装饰装修')
    expect(succ.textContent).toContain('SS-1')
    expect(succ.textContent).toContain('2')
  })

  it('点关闭（closable）触发 onClose', () => {
    const onClose = vi.fn()
    render(<TaskInspector {...baseProps({ onClose })} />)
    fireEvent.click(document.body.querySelector('.ant-drawer-close')!)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('里程碑任务：工期处显示「里程碑」标签（不显天数）', () => {
    render(<TaskInspector {...baseProps({ task: mkTask({ isMilestone: true, duration: 0 }) })} />)
    const summary = ui().getByTestId('sched-inspector-summary').textContent ?? ''
    expect(summary).toContain('里程碑')
    expect(summary).not.toContain('天')
  })
})

describe('双工期（v4.152 刀3）：检查器工期行口径', () => {
  it('cd 任务显「日历」口径与等效工作日跨度；wd 任务维持原样', () => {
    const cd = render(<TaskInspector {...baseProps({ task: mkTask({ duration: 28, durationUnit: 'cd' }), row: mkRow({ es: 0, ef: 20 }) })} />)
    expect(ui().getByTestId('sched-inspector-summary').textContent).toContain('28 天（日历，等效 20 工作日）')
    cd.unmount()
    const wd = render(<TaskInspector {...baseProps({ task: mkTask({ duration: 5 }) })} />)
    expect(ui().getByTestId('sched-inspector-summary').textContent).toContain('5 天')
    expect(ui().getByTestId('sched-inspector-summary').textContent).not.toContain('日历')
    wd.unmount()
  })
})
