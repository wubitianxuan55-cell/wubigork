import { act, fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { GanttView } from './GanttView'
import { computeCpm } from './cpm'
import { useScheduleStore } from './store'
import { CHAT_PREFS_KEY } from './chatPrefs'
import type { SchedProject } from './types'

// GanttView.customView.test.tsx — 自定义视图应用（斑马复刻 S6.2 视图库）：
// store 请求信号 → GanttView 应用 project.views 快照（分组/列集/筛选一次切），
// chatPrefs 落「自定义」标记（自定义 id 不入偏好，重载参数还原）。

function projWithView(): SchedProject {
  return {
    name: '链式样板',
    startDate: '2026-09-07',
    tasks: [
      { id: 'A', name: '挖土', duration: 3, level: 1, progress: 0 },
      { id: 'B', name: '垫层', duration: 2, level: 1, progress: 0 },
    ],
    links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }],
    views: [
      {
        id: 'cv-x', name: '管控视图', kind: 'gantt',
        hide: ['progress'], groupBy: ['status'],
        sort: { field: 'start', dir: 'asc' }, filter: 'all',
      },
    ],
  }
}

describe('GanttView 自定义视图应用（斑马复刻 S6.2）', () => {
  beforeEach(() => {
    localStorage.removeItem(CHAT_PREFS_KEY)
    useScheduleStore.setState({
      project: projWithView(),
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      syncError: null,
      customViewRequest: null,
    })
  })

  it('请求信号触发快照应用：分组头出现（执行状态）、进度列收起、视图名上屏', () => {
    const p = projWithView()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    // 应用前：无分组头（BUILTIN 缺省=计划编制不分组）
    expect(document.querySelectorAll('.sched-group-row')).toHaveLength(0)
    act(() => { useScheduleStore.getState().requestCustomView('cv-x') })
    // 分组头出现（执行状态: 两叶任务同值 → 1 个组）
    const heads = document.querySelectorAll('.sched-group-name')
    expect(heads.length).toBeGreaterThan(0)
    expect(heads[0].textContent).toContain('执行状态')
    // 视图 Select 上屏自定义名（antd Select：取 selection-item 文本）
    const selText = document.querySelector('[data-testid="sched-gantt-view"] .ant-select-selection-item')?.textContent
    expect(selText).toBe('管控视图')
    // chatPrefs 落参数并标记自定义（自定义 id 不入偏好）
    const prefs = JSON.parse(localStorage.getItem(CHAT_PREFS_KEY) ?? '{}')
    expect(prefs.ganttViewId).toBe('custom')
    expect(prefs.ganttHide).toContain('progress')
    expect(prefs.ganttGroup).toEqual(['status'])
  })

  it('不存在的视图 id：信号无副作用（不炸、状态不变）', () => {
    const p = projWithView()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    act(() => { useScheduleStore.getState().requestCustomView('cv-missing') })
    expect(document.querySelectorAll('.sched-group-row')).toHaveLength(0)
    const selText2 = document.querySelector('[data-testid="sched-gantt-view"] .ant-select-selection-item')?.textContent
    expect(selText2).not.toBe('cv-missing')
  })

  it('Toolbar Select 直选自定义视图（不经信号）同路径生效', async () => {
    const p = projWithView()
    render(<GanttView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    // antd Select：mouseDown 开下拉 → portal 内点选项（ResourcePanel.test 同款配方）
    fireEvent.mouseDown(screen.getByTestId('sched-gantt-view').querySelector('.ant-select-selector')!)
    const opt = await screen.findByText('管控视图', { selector: '.ant-select-item-option-content' })
    fireEvent.click(opt.closest('.ant-select-item-option')!)
    expect(document.querySelectorAll('.sched-group-name').length).toBeGreaterThan(0)
  })
})
