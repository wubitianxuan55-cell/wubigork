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

describe('工作日历换算（FE7-04：口径统一走 calendar.ts 的 wdToDate）', () => {
  it('节假日例外参与换算：wd1 落到节假日后首个工作日', () => {
    render(
      <TaskInspector
        {...baseProps({
          row: mkRow({ es: 1, ef: 2, ls: 1, lf: 2 }),
          calendar: { workweek: [1, 2, 3, 4, 5], holidays: ['2026-09-08'] },
        })}
      />,
    )
    // wd0=09-07（周一），09-08 为节假日 → wd1=09-09、wd2=09-10
    expect(ui().getByTestId('sched-inspector-es').textContent).toBe('09-09 (+1)')
    expect(ui().getByTestId('sched-inspector-ef').textContent).toBe('09-10 (+2)')
  })

  it('非默认工作周参与换算：仅周一开工时 wd1 顺延到下周一', () => {
    render(
      <TaskInspector
        {...baseProps({
          row: mkRow({ es: 1, ef: 1 }),
          calendar: { workweek: [1], holidays: [] },
        })}
      />,
    )
    expect(ui().getByTestId('sched-inspector-es').textContent).toBe('09-14 (+1)')
  })
})

// v4.523：what-if 推演经 slot 挂进抽屉（保持本组件「零排程计算」契约——
// 计算与交互态在 WhatIfPanel，本组件只负责按位渲染外部节点）。
describe('TaskInspector what-if slot（v4.523）', () => {
  it('传 whatIf 节点：渲染在后继表之后（抽屉内）', () => {
    render(
      <TaskInspector
        {...baseProps({ whatIf: <div data-testid="slot-probe">推演占位</div> })}
      />,
    )
    expect(ui().getByTestId('slot-probe').textContent).toBe('推演占位')
    // 挂载顺序：后继表在前，slot 在后
    const succs = ui().getByTestId('sched-inspector-succs')
    const probe = ui().getByTestId('slot-probe')
    expect(succs.compareDocumentPosition(probe) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('不传 whatIf：抽屉内容零变化（向后兼容）', () => {
    render(<TaskInspector {...baseProps()} />)
    expect(ui().queryByTestId('slot-probe')).toBeNull()
    expect(ui().getByTestId('sched-inspector-summary')).toBeTruthy()
  })
})

describe('TaskInspector 检查发现（斑马复刻 S1）', () => {
  it('传入 diagnostics 时渲染严重级标签与"下一步"建议', () => {
    render(<TaskInspector {...baseProps({
      diagnostics: [
        { code: 'no-predecessor', severity: 'warn', message: '任务「主体施工」没有前置搭接', hint: '可用 ops auto_chain 补缺省串联' },
        { code: 'milestone-duration', severity: 'error', message: '里程碑工期应为 0' },
      ],
    })} />)
    const box = ui().getByTestId('sched-inspector-diags')
    expect(box.textContent).toContain('没有前置搭接')
    expect(box.textContent).toContain('下一步：可用 ops auto_chain 补缺省串联')
    expect(ui().getByTestId('sched-diag-no-predecessor').textContent).toContain('可疑')
    expect(ui().getByTestId('sched-diag-milestone-duration').textContent).toContain('阻断')
    expect(ui().getByTestId('sched-diag-milestone-duration').textContent).not.toContain('下一步')
  })

  it('无 diagnostics / 空数组时不渲染该段（干净任务保持抽屉简洁）', () => {
    render(<TaskInspector {...baseProps()} />)
    expect(document.body.querySelector('[data-testid="sched-inspector-diags"]')).toBeNull()
  })
})

// 斑马复刻 S5：预警状态机（warn.ts 算好传入，本组件按位渲染 + 算法切换 + 要求完成编辑）
describe('TaskInspector 预警（斑马复刻 S5）', () => {
  const redWarn = {
    taskId: 'B', name: '主体施工', isMilestone: false,
    level: 'red' as const, overdue: 3, band: '0-15天',
    compareFinish: 7, askFinish: 5, algorithm: 'plan-ask' as const,
  }

  it('红灯：渲染滞后天数与档位、比对口径；算法切换回调（计划-要求 → 预测-要求）', () => {
    const onWarnAlgorithmChange = vi.fn()
    render(<TaskInspector {...baseProps({ warn: redWarn, warnAlgorithm: 'plan-ask', onWarnAlgorithmChange })} />)
    const box = ui().getByTestId('sched-inspector-warn')
    expect(ui().getByTestId('sched-warn-red').textContent).toContain('红灯·滞后 3 天')
    expect(box.textContent).toContain('0-15天')
    expect(box.textContent).toContain('按计划完成比对要求完成')
    fireEvent.click(ui().getByText('预测-要求'))
    expect(onWarnAlgorithmChange).toHaveBeenCalledWith('predict-ask')
  })

  it('绿灯：提前天数与预测口径文案', () => {
    render(
      <TaskInspector
        {...baseProps({
          warn: { ...redWarn, level: 'green', overdue: -2, band: null, algorithm: 'predict-ask' },
          warnAlgorithm: 'predict-ask',
        })}
      />,
    )
    const box = ui().getByTestId('sched-inspector-warn')
    expect(ui().getByTestId('sched-warn-green').textContent).toContain('绿灯·未突破（提前 2 天）')
    expect(box.textContent).toContain('按预测完成比对要求完成')
  })

  it('无要求时间（level=none）不渲染预警段；要求完成格存在且无红灯字样', () => {
    render(
      <TaskInspector
        {...baseProps({ warn: { ...redWarn, level: 'none', overdue: null, band: null, compareFinish: null } })}
      />,
    )
    expect(document.body.querySelector('[data-testid="sched-inspector-warn"]')).toBeNull()
    expect(ui().getByTestId('sched-inspector-summary').textContent).toContain('要求完成')
    expect(ui().getByTestId('sched-inspector-summary').textContent).not.toContain('红灯')
  })

  it('要求完成编辑：wd5 渲染 09-14（跨周末顺延）；清除回写 undefined 且输入框清空', () => {
    const onPatch = vi.fn()
    render(<TaskInspector {...baseProps({ task: mkTask({ askFinish: 5 }), onPatch })} />)
    const picker = document.body.querySelector('.ant-picker') as HTMLElement
    const input = picker.querySelector('input') as HTMLInputElement
    expect(input.value).toBe('2026-09-14')
    // 清除（allowEmpty 清除钮常驻 DOM，CSS 控显隐）→ onPatch({ askFinish: undefined })
    const clear = picker.querySelector('.ant-picker-clear')
    expect(clear).toBeTruthy()
    fireEvent.click(clear!)
    expect(onPatch).toHaveBeenCalledWith({ askFinish: undefined })
    expect(input.value).toBe('')
    // 选定日期 → 工作日序号的换算（dateToWd）由 calendar.ts/warn.test.ts 唯一口径覆盖；
    // jsdom 中 rc-picker 打字+Enter 解析不出（吐 null，已知边界），不在此伪造交互。
  })

  it('只读（无 onPatch）：要求完成显日期文本，无输入框', () => {
    render(<TaskInspector {...baseProps({ task: mkTask({ askFinish: 4 }) })} />)
    expect(ui().getByTestId('sched-inspector-summary').textContent).toContain('09-11')
    expect(document.body.querySelector('.ant-picker')).toBeNull()
  })
})
