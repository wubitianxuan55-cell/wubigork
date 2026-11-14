/**
 * ScheduleNetworkWorkspace.test.tsx — ZPert 网络图文档双栏
 */
import { fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { buildAoa } from '../../schedule/aoa'
import { computeCpm } from '../../schedule/cpm'
import { useScheduleStore } from '../../schedule/store'
import type { SchedProject } from '../../schedule/types'
import { ScheduleNetworkWorkspace } from './ScheduleNetworkWorkspace'

const project: SchedProject = {
  name: '网络图样板',
  startDate: '2026-09-07',
  tasks: [
    { id: 'G', name: '主体', duration: 0, level: 0, progress: 0 },
    { id: 'A', name: '挖土', duration: 3, level: 1, progress: 30 },
    { id: 'B', name: '浇筑', duration: 2, level: 1, progress: 0 },
  ],
  links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }],
}

describe('ScheduleNetworkWorkspace', () => {
  beforeEach(() => {
    useScheduleStore.setState({
      project,
      selectedId: null,
      hydrated: true,
      sync: 'saved',
      syncError: null,
    })
  })

  it('左表单与单代号画布同屏', () => {
    const cpm = computeCpm(project.tasks, project.links)
    render(<ScheduleNetworkWorkspace view="pdm" project={project} cpm={cpm} graph={buildAoa(project.tasks, project.links)} />)
    expect(screen.getByTestId('sched-network-workspace')).toBeTruthy()
    expect(screen.getByTestId('sched-network-table').textContent).toContain('挖土')
    expect(screen.getByTestId('sched-pdm')).toBeTruthy()
  })

  it('左表单与双代号画布同屏，点表行选中任务', () => {
    const cpm = computeCpm(project.tasks, project.links)
    render(<ScheduleNetworkWorkspace view="aoa" project={project} cpm={cpm} graph={buildAoa(project.tasks, project.links)} />)
    expect(screen.getByTestId('sched-aoa')).toBeTruthy()
    const row = within(screen.getByTestId('sched-network-table')).getByText('挖土').closest('button')
    expect(row).toBeTruthy()
    fireEvent.click(row!)
    expect(useScheduleStore.getState().selectedId).toBe('A')
  })
})
