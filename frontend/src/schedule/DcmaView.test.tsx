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

// v4.523 纠错建议区：advice.ts（v4.491 纯函数）首次接 UI——体检单下游投影
// 「作者下一步动手清单」，同一 project+cpm 输入下与体检结论必然一致。
describe('DcmaView 纠错建议区（v4.523：scheduleAdvice 呈现）', () => {
  // 坏计划四类全中：孤立 D + 负搭接 A→B(-2) + 悬空 B→GHOST + 冗余（A→B 直达且经 C 间接）
  const badProject: SchedProject = {
    name: 'bad', startDate: '2026-09-01',
    tasks: [task('A', 5), task('B', 10), task('C', 5), task('D', 3)],
    links: [
      { from: 'A', to: 'B', type: 'FS', lag: -2 },
      { from: 'A', to: 'C', type: 'FS', lag: 0 },
      { from: 'C', to: 'B', type: 'FS', lag: 0 },
      { from: 'B', to: 'GHOST', type: 'FS', lag: 0 },
    ],
  }
  const badCpm = computeCpm(badProject.tasks, badProject.links)

  function renderAdvice(p: SchedProject = badProject, c = badCpm) {
    mocks.readFile.mockRejectedValue(new Error('no such file'))
    return render(<DcmaView project={p} cpm={c} dataDate={null} />)
  }

  it('坏计划：四类建议各占一条，含问题文案与修复动作（动手清单）', async () => {
    renderAdvice()
    const section = await screen.findByTestId('dcma-advice')
    const text = section.textContent ?? ''
    // 孤立任务（缺逻辑）
    expect(text).toContain('缺逻辑')
    expect(text).toContain('完全孤立')
    expect(text).toContain('D')
    // 负搭接
    expect(text).toContain('负搭接')
    expect(text).toContain('A → B (-2d)')
    // 悬空引用
    expect(text).toContain('悬空引用')
    expect(text).toContain('GHOST')
    // 冗余搭接
    expect(text).toContain('冗余搭接')
    expect(text).toContain('间接可达')
    // 每条建议都有「建议：」修复动作
    expect(text.match(/建议：/g)?.length).toBeGreaterThanOrEqual(4)
  })

  it('好计划（干净 A→B→C）：建议区零态文案，无建议条目', async () => {
    mocks.readFile.mockRejectedValue(new Error('no such file'))
    render(<DcmaView project={project} cpm={cpm} dataDate={null} />)
    await waitFor(() => expect(screen.getByTestId('dcma-score')).toBeTruthy())
    const section = screen.getByTestId('dcma-advice')
    expect(section.textContent).toContain('未发现需修复的逻辑问题')
    expect(screen.queryAllByTestId('dcma-advice-item')).toHaveLength(0)
  })

  it('建议条目带 kind 标签与涉及任务行（人可读结构）', async () => {
    renderAdvice()
    const items = await screen.findAllByTestId('dcma-advice-item')
    expect(items.length).toBeGreaterThanOrEqual(4)
    // 每条含 kind 标签（data-kind）与涉及任务/关系文本
    for (const it of items) {
      expect(it.querySelector('[data-kind]')).toBeTruthy()
    }
  })
})
