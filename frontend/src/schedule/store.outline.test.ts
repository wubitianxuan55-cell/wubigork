import { beforeEach, describe, expect, it } from 'vitest'
import { normalizeProject, useScheduleStore } from './store'
import type { SchedProject } from './types'

// store.outline.test.ts — 大纲三动作（斑马 C13 预演闸）+ 表格粘贴 + 命令信号 + 网络模式。
// 拦截文案沿斑马原文；先预演后落盘。

function resetProject(tasks: SchedProject['tasks'], extra: Partial<SchedProject> = {}): void {
  useScheduleStore.setState({
    project: normalizeProject({ name: 'P', startDate: '2026-09-07', tasks, links: [], ...extra }),
    past: [],
    future: [],
    ganttCommand: null,
    customViewRequest: null,
  })
}

const baseTasks: SchedProject['tasks'] = [
  { id: 'G', name: '分区一', duration: 0, level: 0, progress: 0 },
  { id: 'A', name: '挖土', duration: 3, level: 1, progress: 0 },
  { id: 'B', name: '吊装', duration: 2, level: 1, progress: 0 },
]

describe('store 大纲三动作（斑马复刻 C13 预演）', () => {
  beforeEach(() => resetProject(baseTasks))

  it('升级：叶→分区；有搭接/有分配/有固定成本 拒绝（斑马原文文案）', () => {
    const s = useScheduleStore.getState()
    expect(s.promoteTask('A').ok).toBe(true)
    expect(useScheduleStore.getState().project.tasks.find((t) => t.id === 'A')?.level).toBe(0)
    resetProject([{ ...baseTasks[1], fixedCost: 100 }])
    expect(useScheduleStore.getState().promoteTask('A').reason).toContain('固定成本')
    resetProject([{ ...baseTasks[1] }])
    useScheduleStore.setState((st) => ({ project: { ...st.project, links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }] } }))
    expect(useScheduleStore.getState().promoteTask('A').reason).toContain('该工作不能修改为关系')
    useScheduleStore.setState((st) => ({
      project: { ...st.project, links: [], assignments: [{ taskId: 'A', resourceId: 'r1', units: 1 }] },
    }))
    expect(useScheduleStore.getState().promoteTask('A').reason).toContain('分组行禁挂分配')
  })

  it('降级：分区→叶（前有分组才可）；首行分区拒绝（斑马原文）', () => {
    const s = useScheduleStore.getState()
    // G 在表首（前面没有分组行）→ 斑马：分区不能降级为非分区的子
    expect(s.demoteTask('G').reason).toContain('分区不能降级为非分区的子')
    resetProject([
      { id: 'G1', name: '分区一', duration: 0, level: 0, progress: 0 },
      { id: 'A', name: '挖土', duration: 3, level: 1, progress: 0 },
      { id: 'G2', name: '分区二', duration: 0, level: 0, progress: 0 },
      { id: 'B', name: '吊装', duration: 2, level: 1, progress: 0 },
    ])
    expect(useScheduleStore.getState().demoteTask('G2').ok).toBe(true)
    expect(useScheduleStore.getState().project.tasks.find((t) => t.id === 'G2')?.level).toBe(1)
    expect(useScheduleStore.getState().demoteTask('A').reason).toContain('无法再降级')
  })

  it('上移/下移：相邻换位；分区不与子工作换位（斑马 C13）；边界拒绝', () => {
    const s = useScheduleStore.getState()
    expect(s.moveTask('G', -1).reason).toBe('已到表首')
    expect(s.moveTask('B', 1).reason).toBe('已到表尾')
    // G(分区) 下移：下一行是自己的子工作 → 斑马 C13 拦截
    expect(s.moveTask('G', 1).reason).toContain('分区不能与子工作换位')
    // 叶任务上移跨过分组边界=合法（换到前一分区之前）
    expect(s.moveTask('A', -1).ok).toBe(true)
    expect(useScheduleStore.getState().project.tasks.map((t) => t.id)).toEqual(['A', 'G', 'B'])
  })
})

describe('store 表格粘贴 + 命令信号 + 网络模式（一次点亮批4）', () => {
  beforeEach(() => resetProject(baseTasks))

  it('pasteRows：TSV 行追加为叶任务（工期解析容错、空行跳过）', () => {
    const s = useScheduleStore.getState()
    const res = s.pasteRows('回填土方\t4\n养护\t\n\n水泥运输\t99999')
    expect(res.ok).toBe(true)
    expect(res.count).toBe(3)
    const tasks = useScheduleStore.getState().project.tasks
    expect(tasks.filter((t) => t.name === '回填土方')[0]?.duration).toBe(4)
    expect(tasks.filter((t) => t.name === '养护')[0]?.duration).toBe(1)
    expect(tasks.filter((t) => t.name === '水泥运输')[0]?.duration).toBe(3650)
    expect(s.pasteRows('   ').ok).toBe(false)
    expect(s.pasteRows('\t\t').ok).toBe(false)
  })

  it('sendGanttCommand：seq 递增、payload 透传', () => {
    const s = useScheduleStore.getState()
    s.sendGanttCommand('find', '挖土')
    s.sendGanttCommand('collapse-all')
    expect(useScheduleStore.getState().ganttCommand).toEqual({ kind: 'collapse-all', payload: undefined, seq: 2 })
    s.sendGanttCommand('set-group', JSON.stringify(['status']))
    expect(useScheduleStore.getState().ganttCommand?.payload).toBe('["status"]')
  })

  it('setNetworkMode：写入 networkStyle.mode 承载位（斑马 display_mode 随文档）', () => {
    const s = useScheduleStore.getState()
    s.setNetworkMode('logic')
    expect(useScheduleStore.getState().project.networkStyle?.mode).toBe('logic')
    s.setNetworkMode('timescaled')
    expect(useScheduleStore.getState().project.networkStyle?.mode).toBe('timescaled')
    useScheduleStore.getState().undo()
    expect(useScheduleStore.getState().project.networkStyle?.mode).toBe('logic')
  })
})
