// UsageView.test.tsx — 资源使用视图（纯展示件）UI 用例
//
// 组件零业务：数据（UsageResult）全部由用例手写 mock 传入（不 import 计算函数），
// 只共享 usageTypes.ts 类型契约。覆盖：汇总文案与超载红色、左列资源名/超载天数/
// 峰值累计小字、超载格 class 与 title（任务构成 + 负载/可用）、days>62 表头每 10
// 防挤、空态（rows 空）与 ok=false 错误文案、totalOverloadDays 汇总透传。

import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { UsageView } from './UsageView'
import type { UsageDayCell, UsageResult, UsageResourceRow } from './usageTypes'

// 钢筋工：T1 基础开挖 wd0-2、T2 主体施工 wd2-4 → wd2 双任务叠加=负载 2 > 可用 1 → 超载 1 天
const REBAR: UsageResourceRow = {
  resourceId: 'r1',
  name: '钢筋工',
  type: 'work',
  tasks: [
    { taskId: 'T1', name: '基础开挖', es: 0, ef: 3, units: 1 },
    { taskId: 'T2', name: '主体施工', es: 2, ef: 5, units: 1 },
  ],
  cells: [
    { wd: 0, workload: 1, availability: 1, overloaded: false },
    { wd: 1, workload: 1, availability: 1, overloaded: false },
    { wd: 2, workload: 2, availability: 1, overloaded: true },
    { wd: 3, workload: 1, availability: 1, overloaded: false },
    { wd: 4, workload: 1, availability: 1, overloaded: false },
  ],
  peakWorkload: 2,
  overloadDays: 1,
  totalWorkload: 6,
}

// 普工：仅前 2 天有负载，无超载（左列应显灰色「超载 0 天」）
const LABOR: UsageResourceRow = {
  resourceId: 'r2',
  name: '普工',
  type: 'work',
  tasks: [{ taskId: 'T3', name: '场地清理', es: 0, ef: 2, units: 1 }],
  cells: [
    { wd: 0, workload: 1, availability: 1, overloaded: false },
    { wd: 1, workload: 1, availability: 1, overloaded: false },
    { wd: 2, workload: 0, availability: 1, overloaded: false },
    { wd: 3, workload: 0, availability: 1, overloaded: false },
    { wd: 4, workload: 0, availability: 1, overloaded: false },
  ],
  peakWorkload: 1,
  overloadDays: 0,
  totalWorkload: 2,
}

function mkUsage(extra: Partial<UsageResult> = {}): UsageResult {
  return { ok: true, days: 5, rows: [REBAR, LABOR], totalOverloadDays: 1, ...extra }
}

describe('UsageView 资源使用视图', () => {
  it('正常渲染：汇总文案、左列资源名/超载天数/峰值累计、超载格带 sched-usage-over', () => {
    render(<UsageView usage={mkUsage()} />)
    const summary = screen.getByTestId('sched-usage-summary')
    expect(summary.textContent).toContain('共 2 名工时资源')
    expect(summary.textContent).toContain('峰值负载 2')
    expect(summary.textContent).toContain('超载 1 资源-日')
    // 左固定列：资源名 + 超载天数（红/灰）+ 峰值/累计小字
    expect(screen.getByText('钢筋工')).toBeTruthy()
    expect(screen.getByText('普工')).toBeTruthy()
    expect(screen.getByText('超载 1 天')).toBeTruthy()
    expect(screen.getByText('超载 0 天')).toBeTruthy()
    expect(screen.getByText('峰值 2 · 累计 6 工日')).toBeTruthy()
    expect(screen.getByText('峰值 1 · 累计 2 工日')).toBeTruthy()
    // 超载格：按 title 定位（wd2），class 含 sched-usage-over
    const over = document.body.querySelector<HTMLElement>('[title*="负载 2 / 可用 1"]')
    expect(over).not.toBeNull()
    expect(over!.classList.contains('sched-usage-over')).toBe(true)
  })

  it('超载格 title：含「负载 2 / 可用 1」与当日全部在工任务名×units', () => {
    render(<UsageView usage={mkUsage()} />)
    const over = document.body.querySelector<HTMLElement>('.sched-usage-over')
    expect(over).not.toBeNull()
    expect(over!.title).toContain('第 2 工作日')
    expect(over!.title).toContain('负载 2 / 可用 1')
    expect(over!.title).toContain('基础开挖×1')
    expect(over!.title).toContain('主体施工×1')
  })

  it('days>62：表头数字改每 10 的倍数显示（wd=60 有数、wd=55 空）', () => {
    render(
      <UsageView
        usage={mkUsage({
          days: 70,
          rows: [{
            resourceId: 'r9',
            name: '杂工',
            type: 'work',
            tasks: [{ taskId: 'T9', name: '长任务', es: 0, ef: 70, units: 1 }],
            cells: Array.from({ length: 70 }, (_, wd): UsageDayCell => ({ wd, workload: 1, availability: 1, overloaded: false })),
            peakWorkload: 1,
            overloadDays: 0,
            totalWorkload: 70,
          }],
        })}
      />,
    )
    expect(screen.getByTestId('sched-usage-head-0').textContent).toBe('0')
    expect(screen.getByTestId('sched-usage-head-50').textContent).toBe('50')
    expect(screen.getByTestId('sched-usage-head-60').textContent).toBe('60')
    expect(screen.getByTestId('sched-usage-head-55').textContent).toBe('')
    expect(screen.getByTestId('sched-usage-head-62').textContent).toBe('')
  })

  it('空态：rows=[] 显示空态引导（不渲染热力表）', () => {
    render(<UsageView usage={mkUsage({ rows: [] })} />)
    const empty = screen.getByTestId('sched-usage-empty')
    expect(empty.textContent).toContain('暂无工时资源或计划为空')
    expect(empty.textContent).toContain('「资源」中添加工时资源')
    expect(document.body.querySelector('.sched-usage-cell')).toBeNull()
  })

  it('ok=false：显示错误文案（含 error 内容），不显示空态与热力表', () => {
    render(
      <UsageView
        usage={{ ok: false, error: '存在循环依赖：A → B → A', days: 0, rows: [], totalOverloadDays: 0 }}
      />,
    )
    const summary = screen.getByTestId('sched-usage-summary')
    expect(summary.textContent).toContain('使用视图计算失败')
    expect(summary.textContent).toContain('循环依赖')
    expect(screen.queryByTestId('sched-usage-empty')).toBeNull()
    expect(document.body.querySelector('.sched-usage-cell')).toBeNull()
  })

  it('totalOverloadDays 汇总透传：M>0 红色、M=0 灰色', () => {
    const { unmount } = render(<UsageView usage={mkUsage({ totalOverloadDays: 3 })} />)
    expect(screen.getByTestId('sched-usage-summary').textContent).toContain('超载 3 资源-日')
    expect(screen.getByTestId('sched-usage-over-total').getAttribute('style')).toContain('sched-critical')
    unmount()
    render(<UsageView usage={mkUsage({ totalOverloadDays: 0 })} />)
    expect(screen.getByTestId('sched-usage-summary').textContent).toContain('超载 0 资源-日')
    expect(screen.getByTestId('sched-usage-over-total').getAttribute('style')).toBeFalsy()
  })
})
