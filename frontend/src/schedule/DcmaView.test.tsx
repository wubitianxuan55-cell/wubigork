// FE7-14 回归防线：DCMA 阈值覆盖文件读取/解析失败不再静默。
// 此前空 catch + .catch(()=>{}) 双层静默，用户看到的是「全按内置默认跑」而
// 没有任何提示，体检结论可能与工作区自定义阈值不一致。现失败写进组件态并渲染
// 一行诚实提示（不弹窗、不打断体检）；文件本来不存在=内置默认的预期行为，不提示。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { DcmaView } from './DcmaView'
import { computeCpm } from './cpm'
import type { SchedProject, SchedTask } from './types'

const mocks = vi.hoisted(() => ({ readFile: vi.fn() }))

vi.mock('../gaea/lib/bridge', () => ({
  app: { ReadFile: mocks.readFile },
}))

function task(id: string, duration: number): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0 }
}

const project: SchedProject = {
  name: 'p', startDate: '2026-09-01',
  tasks: [task('A', 5), task('B', 10), task('C', 5)],
  links: [
    { from: 'A', to: 'B', type: 'FS', lag: 0 },
    { from: 'B', to: 'C', type: 'FS', lag: 0 },
  ],
}
const cpm = computeCpm(project.tasks, project.links)

function renderView() {
  return render(<DcmaView project={project} cpm={cpm} dataDate={null} />)
}

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

describe('DcmaView 阈值覆盖失败可见化（FE7-14）', () => {
  it('阈值文件 JSON 坏 → 提示「已回落内置默认」上屏', async () => {
    mocks.readFile.mockResolvedValue({ path: '.gaea/skills/schedule-dcma/thresholds.json', markdown: '{ 坏 JSON' })
    renderView()
    const warn = await screen.findByTestId('dcma-threshold-degraded')
    expect(warn.textContent).toContain('阈值文件读取失败，已回落内置默认')
    expect(warn.textContent).toContain('JSON 解析失败')
  })

  it('阈值文件读取被拒 → 提示可见（不再是空 catch）', async () => {
    mocks.readFile.mockRejectedValue(new Error('EACCES: permission denied'))
    renderView()
    const warn = await screen.findByTestId('dcma-threshold-degraded')
    expect(warn.textContent).toContain('EACCES')
  })

  it('阈值文件不存在 → 静默用内置默认（不刷提示）', async () => {
    mocks.readFile.mockRejectedValue(new Error('open .gaea/skills/schedule-dcma/thresholds.json: no such file or directory'))
    renderView()
    await waitFor(() => expect(screen.getByTestId('dcma-score')).toBeTruthy())
    expect(screen.queryByTestId('dcma-threshold-degraded')).toBeNull()
  })

  it('阈值文件为空内容（无覆盖）→ 静默用内置默认（不刷提示）', async () => {
    mocks.readFile.mockResolvedValue({ path: '.gaea/skills/schedule-dcma/thresholds.json', markdown: '   ', size: 0 })
    renderView()
    await waitFor(() => expect(screen.getByTestId('dcma-score')).toBeTruthy())
    expect(screen.queryByTestId('dcma-threshold-degraded')).toBeNull()
  })

  it('阈值文件正常 → 无降级提示（反向验证）', async () => {
    mocks.readFile.mockResolvedValue({ path: 'x', markdown: JSON.stringify({ highFloatDays: 3 }) })
    renderView()
    await waitFor(() => expect(screen.getByTestId('dcma-score')).toBeTruthy())
    expect(screen.queryByTestId('dcma-threshold-degraded')).toBeNull()
  })
})
