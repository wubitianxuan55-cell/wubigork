// WhatIfPanel.test.tsx — 工期 what-if 推演面板（v4.523：whatif.ts v4.491 纯函数首次接 UI）
//
// 契约：纯本地重算（computeCpm），不改原计划；delta 步进（−5/−1/重置/+1/+5）；
// 结果=总工期基线→推演 + 关键线路进出名单；里程碑/不存在任务诚实降级（reason）。
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { WhatIfPanel } from './WhatIfPanel'
import { computeCpm } from './cpm'
import type { SchedProject, SchedTask } from './types'

function task(id: string, duration: number, extra: Partial<SchedTask> = {}): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0, ...extra }
}

// 干净链：A(5)→B(10)→C(5) 关键（总 20）；A→D(8)→C 旁路非关键
const project: SchedProject = {
  name: 'p', startDate: '2026-09-01',
  tasks: [task('A', 5), task('B', 10), task('C', 5), task('D', 8)],
  links: [
    { from: 'A', to: 'B', type: 'FS', lag: 0 },
    { from: 'B', to: 'C', type: 'FS', lag: 0 },
    { from: 'A', to: 'D', type: 'FS', lag: 0 },
    { from: 'D', to: 'C', type: 'FS', lag: 0 },
  ],
}
// 基线总工期 20（供用例断言锚定，避免与实现耦合的重复手算漂移）
void computeCpm(project.tasks, project.links)

// 里程碑计划
const msProject: SchedProject = {
  name: 'ms', startDate: '2026-09-01',
  tasks: [task('M', 0, { isMilestone: true }), task('A', 5)],
  links: [{ from: 'A', to: 'M', type: 'FS', lag: 0 }],
}

afterEach(cleanup)

describe('WhatIfPanel 工期推演（what-if）', () => {
  it('初始渲染：标题/只读口径注记/基线总工期/步进按钮组（delta=0 不重算对比）', () => {
    render(<WhatIfPanel project={project} taskId="B" />)
    const root = screen.getByTestId('sched-whatif')
    const text = root.textContent ?? ''
    expect(text).toContain('工期推演')
    expect(text).toContain('本地重算')
    expect(text).toContain('不改原计划')
    expect(text).toContain('20') // 基线总工期
    // 五个步进钮：−5 −1 重置 +1 +5
    expect(within(root).getAllByRole('button').length).toBeGreaterThanOrEqual(5)
  })

  it('关键任务 B +1：总工期 20→21，关键线路不变（新进/脱离为空不渲染名单）', () => {
    render(<WhatIfPanel project={project} taskId="B" />)
    fireEvent.click(screen.getByTestId('sched-whatif-plus1'))
    const text = screen.getByTestId('sched-whatif-result').textContent ?? ''
    expect(text).toContain('21')
    expect(text).toContain('+1')
  })

  it('旁路任务 D +8（8→16 超过 B 的 10）：D 新进关键线路，总工期变 26', () => {
    render(<WhatIfPanel project={project} taskId="D" />)
    fireEvent.click(screen.getByTestId('sched-whatif-plus5'))
    fireEvent.click(screen.getByTestId('sched-whatif-plus1'))
    fireEvent.click(screen.getByTestId('sched-whatif-plus1'))
    fireEvent.click(screen.getByTestId('sched-whatif-plus1'))
    const text = screen.getByTestId('sched-whatif-result').textContent ?? ''
    expect(text).toContain('26')
    expect(text).toContain('新进关键')
    expect(text).toContain('D')
  })

  it('重置：推演后点重置回到基线态（delta=0）', () => {
    render(<WhatIfPanel project={project} taskId="B" />)
    fireEvent.click(screen.getByTestId('sched-whatif-plus1'))
    fireEvent.click(screen.getByTestId('sched-whatif-reset'))
    const text = screen.getByTestId('sched-whatif-result').textContent ?? ''
    expect(text).toContain('20')
    expect(text).not.toContain('21')
  })

  it('里程碑任务：诚实降级（reason 文案，无对比数字）', () => {
    render(<WhatIfPanel project={msProject} taskId="M" />)
    const text = screen.getByTestId('sched-whatif-result').textContent ?? ''
    expect(text).toContain('里程碑')
    expect(text).toContain('无法推演')
  })

  it('任务不存在：降级 reason（不崩溃）', () => {
    render(<WhatIfPanel project={project} taskId="NOPE" />)
    const text = screen.getByTestId('sched-whatif-result').textContent ?? ''
    expect(text).toContain('不存在')
  })
})
