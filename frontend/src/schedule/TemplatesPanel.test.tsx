/**
 * TemplatesPanel.test.tsx — 项目模板面板 UI 用例（v4.138 #10）
 *
 * 覆盖：空态起步引导文案、另存为模板（输入缺省=工程名；保存后列表出现该行
 * 且整份工程快照落盘）、载入模板（Popconfirm 确认文案→project 变成模板内容，
 * importProject 走 pushHistory 可撤销）、删除模板（Popconfirm 确认→该行消失
 * 回落空态、存储同步删除）。
 * 面板数据走 useScheduleStore（与挂进 SchedulePage 工具栏 Popover 的运行形态
 * 一致）；模板存储走 jsdom localStorage，beforeEach 清两个键隔离用例。
 */

import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { TemplatesPanel } from './TemplatesPanel'
import { computeCpm } from './cpm'
import { useScheduleStore } from './store'
import { TEMPLATES_KEY, getTemplate, listTemplates } from './templates'
import type { SchedProject } from './types'

/** A(3)→B(2)→C(4) 串联：总工期 9（与 BaselinesPanel.test 同款最小工程） */
function chainProject(): SchedProject {
  return {
    name: '链式样板',
    startDate: '2026-09-07',
    tasks: [
      { id: 'A', name: '挖土', duration: 3, level: 1, progress: 0 },
      { id: 'B', name: '垫层', duration: 2, level: 1, progress: 0 },
      { id: 'C', name: '浇筑', duration: 4, level: 1, progress: 0 },
    ],
    links: [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'B', to: 'C', type: 'FS', lag: 0 },
    ],
  }
}

/** 另一份内容可辨的工程（载入模板后的期望内容） */
function otherProject(): SchedProject {
  return {
    name: '厂房工程',
    startDate: '2026-10-01',
    tasks: [
      { id: 'X', name: '备料', duration: 5, level: 1, progress: 0 },
      { id: 'Y', name: '吊装', duration: 7, level: 1, progress: 0 },
    ],
    links: [{ from: 'X', to: 'Y', type: 'FS', lag: 0 }],
  }
}

function setProject(p: SchedProject) {
  useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', syncError: null })
}

function renderPanel(p: SchedProject) {
  setProject(p)
  return render(<TemplatesPanel cpm={computeCpm(p.tasks, p.links)} />)
}

// antd Input 的 data-testid 落在包装层，统一取内部 input
function inputOf(testid: string): HTMLInputElement {
  const el = screen.getByTestId(testid)
  if (el instanceof HTMLInputElement) return el
  const inner = el.querySelector('input')
  if (inner) return inner
  throw new Error(`${testid} 内无 input`)
}

/** 确认 Popconfirm（antd 缺省 locale 为 OK） */
async function confirmPop() {
  fireEvent.click(await screen.findByRole('button', { name: /OK|确定/ }))
}

/** 另存为模板：可改名（省略=用缺省工程名）→ 点存为模板 */
function saveAs(name?: string) {
  if (name !== undefined) fireEvent.change(inputOf('sched-tpl-name-input'), { target: { value: name } })
  fireEvent.click(screen.getByTestId('sched-tpl-save'))
}

describe('TemplatesPanel', () => {
  beforeEach(() => {
    localStorage.removeItem('gaea.schedule.v1')
    localStorage.removeItem(TEMPLATES_KEY)
    useScheduleStore.setState({
      project: chainProject(), selectedId: null, hydrated: true, sync: 'saved',
      savedAt: null, syncError: null, past: [], future: [],
    })
  })

  it('空态：无模板时显示起步引导文案，无列表行', () => {
    renderPanel(chainProject())
    expect(screen.getByTestId('sched-templates-panel').textContent)
      .toContain('暂无模板——把当前计划另存为模板，新工程从这里起步')
    expect(screen.queryByTestId('sched-template-item')).toBeNull()
  })

  it('另存为模板：输入框缺省=工程名；改名保存后列表出现该行，整份工程快照落盘', () => {
    renderPanel(chainProject())
    expect(inputOf('sched-tpl-name-input').value).toBe('链式样板')
    saveAs('自住楼模板')
    const rows = screen.getAllByTestId('sched-template-item')
    expect(rows).toHaveLength(1)
    expect(rows[0].textContent).toContain('自住楼模板')
    // 落盘：真实 localStorage 往返，快照是保存当时的整份工程
    expect(listTemplates().map((t) => t.name)).toEqual(['自住楼模板'])
    const saved = getTemplate('自住楼模板')!
    expect(saved.project.name).toBe('链式样板')
    expect(saved.project.tasks.map((t) => t.id)).toEqual(['A', 'B', 'C'])
  })

  it('载入模板：Popconfirm 确认后 project 变成模板内容，且 pushHistory 可撤销', async () => {
    const view = renderPanel(chainProject())
    saveAs('链条模板')
    // 切到另一份工程（模拟打开别的计划），再从模板载回
    const other = otherProject()
    setProject(other)
    view.rerender(<TemplatesPanel cpm={computeCpm(other.tasks, other.links)} />)
    expect(useScheduleStore.getState().project.name).toBe('厂房工程')

    fireEvent.click(screen.getByTestId('sched-tpl-load'))
    expect(await screen.findByText('载入将覆盖当前计划（可撤销），确定？')).toBeTruthy()
    await confirmPop()

    const s = useScheduleStore.getState().project
    expect(s.name).toBe('链式样板')
    expect(s.tasks.map((t) => t.id)).toEqual(['A', 'B', 'C'])
    expect(s.tasks.map((t) => t.duration)).toEqual([3, 2, 4])
    expect(s.links).toHaveLength(2)
    // importProject 内 pushHistory：留下一次可撤销快照
    expect(useScheduleStore.getState().past).toHaveLength(1)
  })

  it('删除模板：Popconfirm 确认后该行消失、存储同步删除，回落空态文案', async () => {
    renderPanel(chainProject())
    saveAs('要删的模板')
    expect(screen.getAllByTestId('sched-template-item')).toHaveLength(1)

    fireEvent.click(screen.getByTestId('sched-tpl-del'))
    expect(await screen.findByText('删除模板「要删的模板」？')).toBeTruthy()
    await confirmPop()

    expect(screen.queryByTestId('sched-template-item')).toBeNull()
    expect(screen.getByTestId('sched-templates-panel').textContent).toContain('暂无模板')
    expect(listTemplates()).toEqual([])
  })
})
