import { beforeEach, describe, expect, it } from 'vitest'
import { normalizeProject, useScheduleStore } from './store'
import type { SchedView } from './types'

// store.views.test.ts — 视图库三动作+请求信号（斑马复刻 S6.2「保存视图/视图管理」）。
// 快照形状校验在 views.test.ts；此处只测 store 侧：入库/同名原文拒绝/改名/删除/撤销/信号。

const view = (id: string, name: string): SchedView => ({ id, name, kind: 'gantt' })

function resetProject(): void {
  useScheduleStore.setState({
    project: normalizeProject({ name: 'P', startDate: '2026-09-07', tasks: [], links: [] }),
    past: [],
    future: [],
    customViewRequest: null,
  })
}

describe('store 视图库（斑马复刻 S6.2）', () => {
  beforeEach(resetProject)

  it('保存→在库；重名/冲突拒绝原文；删除', () => {
    const s = useScheduleStore.getState()
    expect(s.saveCustomView(view('v1', '管控')).ok).toBe(true)
    expect(useScheduleStore.getState().project.views).toHaveLength(1)
    expect(s.saveCustomView(view('v2', '管控')).reason).toBe('存在同名视图，请重新保存')
    expect(s.saveCustomView(view('v1', '另一个')).reason).toBe('视图 id 冲突，请重试')
    s.removeCustomView('v1')
    expect(useScheduleStore.getState().project.views).toHaveLength(0)
    s.removeCustomView('v1') // 不存在静默（同 removeReview 口径）
  })

  it('改名：成功/撞名原文/空名/不存在；pushHistory 可撤销', () => {
    const s = useScheduleStore.getState()
    s.saveCustomView(view('v1', 'A'))
    expect(s.renameCustomView('v1', 'B').ok).toBe(true)
    expect(useScheduleStore.getState().project.views?.[0]?.name).toBe('B')
    s.saveCustomView(view('v2', 'C'))
    expect(s.renameCustomView('v2', 'B').reason).toBe('存在同名视图，请重新保存')
    expect(s.renameCustomView('v2', '  ').reason).toBe('视图名不能为空')
    expect(s.renameCustomView('v9', 'X').reason).toBe('视图不存在')
    // 历史栈：undo 一次退「保存 C」、再退「改名 A→B」→ 回到 A
    useScheduleStore.getState().undo()
    useScheduleStore.getState().undo()
    expect(useScheduleStore.getState().project.views?.[0]?.name).toBe('A')
  })

  it('保存走 pushHistory：undo 后视图消失', () => {
    useScheduleStore.getState().saveCustomView(view('v1', '管控'))
    expect(useScheduleStore.getState().project.views).toHaveLength(1)
    useScheduleStore.getState().undo()
    expect(useScheduleStore.getState().project.views ?? []).toHaveLength(0)
  })

  it('请求信号 seq 递增（GanttView 按信号应用快照）', () => {
    useScheduleStore.getState().requestCustomView('v1')
    useScheduleStore.getState().requestCustomView('v1')
    expect(useScheduleStore.getState().customViewRequest).toEqual({ id: 'v1', seq: 2 })
  })
})
