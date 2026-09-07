import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import { ScheduleDiffCard } from './ScheduleDiffCard'
import { scheduleApplyArgsOf } from './applyDiff'
import type { SchedProject } from './types'

// 桥接 mock：before 态读取（loadScheduleFile 只服务缺省路径）是本组件唯一
// 下行依赖（diff 确认闭环刀B）。
const mocks = vi.hoisted(() => ({
  load: vi.fn(async (): Promise<{ exists: boolean; path: string; project?: SchedProject }> => ({
    exists: false, path: '进度计划/当前计划.gsched.json',
  })),
}))

vi.mock('./api', () => ({
  loadScheduleFile: () => mocks.load(),
}))

const before: SchedProject = {
  name: '当前计划', startDate: '2026-09-07',
  tasks: [{ id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 }],
  links: [],
}
const after: SchedProject = {
  name: '当前计划', startDate: '2026-09-07',
  tasks: [
    { id: 'A', name: '挖土', duration: 5, level: 1, progress: 0 },
    { id: 'B', name: '垫层', duration: 2, level: 1, progress: 0 },
  ],
  links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }],
}

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

describe('ScheduleDiffCard（schedule_apply 审批卡 diff 卡体，刀B）', () => {
  it('project 通道：加载文件实况为 before → 汇总行+明细行渲染（预览标注）', async () => {
    mocks.load.mockResolvedValue({ exists: true, path: '进度计划/当前计划.gsched.json', project: before })
    render(<ScheduleDiffCard args={JSON.stringify({ project: after })} />)
    const card = await screen.findByTestId('sched-diff-card')
    await screen.findAllByTestId('sched-diff-line')
    expect(card.textContent).toContain('总工期 2→7 天')
    expect(card.textContent).toContain('预览数字，落盘以回执为准')
    const lines = Array.from(document.querySelectorAll('[data-testid="sched-diff-line"]')).map((e) => e.textContent)
    expect(lines.some((t) => t?.includes('新增任务：垫层'))).toBe(true)
    expect(lines.some((t) => t?.includes('工期（工作日）：2 → 5'))).toBe(true)
    expect(lines.some((t) => t?.includes('新增搭接（垫层）'))).toBe(true)
  })

  it('非缺省路径 → 降级为无现状对比说明', async () => {
    render(<ScheduleDiffCard args={JSON.stringify({ path: '进度计划/别的工程.gsched.json', project: after })} />)
    const card = await screen.findByTestId('sched-diff-card')
    await screen.findByTestId('sched-diff-card')
    expect(card.textContent).toContain('非当前计划文件，无现状对比')
    expect(mocks.load).not.toHaveBeenCalled()
  })

  it('ops 通道（刀D 前）→ 诚实降级为意图清单说明', () => {
    render(<ScheduleDiffCard args={JSON.stringify({ ops: [{ op: 'patch_task' }] })} />)
    const card = screen.getByTestId('sched-diff-card')
    expect(card.textContent).toContain('ops 增量调整 1 条')
    expect(card.textContent).toContain('以 apply 回执为准')
  })

  it('args 为 null（参数未就绪）→ 提示且不崩', () => {
    render(<ScheduleDiffCard args={null} />)
    expect(screen.getByText(/参数尚未就绪或解析失败/)).toBeTruthy()
  })

  it('scheduleApplyArgsOf：取最后一条 running 的 schedule_apply args', () => {
    const items = [
      { kind: 'tool', name: 'schedule_apply', status: 'done', args: '{"old":1}' },
      { kind: 'tool', name: 'read_file', status: 'running', args: '{"path":"x"}' },
      { kind: 'tool', name: 'schedule_apply', status: 'running', args: '{"new":2}' },
    ]
    expect(scheduleApplyArgsOf(items)).toBe('{"new":2}')
    expect(scheduleApplyArgsOf([{ kind: 'tool', name: 'schedule_apply', status: 'done', args: '{"a":1}' }])).toBeNull()
  })
})
