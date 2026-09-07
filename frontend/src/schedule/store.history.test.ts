/**
 * store.history.test.ts — 多级撤消/重做（刀C 余项）
 *
 * 用例口径：快照式历史（addTask/updateTask/removeTask/importProject）；
 * 文本编辑合并窗口（800ms 内同 kind+id 只压一次栈，Date.now 注入）；
 * 上限 50 截断；外部回读（setState 直改）不入史。
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { resetHistorySession, useScheduleStore } from './store'
import { makeEmptyProject } from './sample'
import type { SchedProject } from './types'

function fresh(): void {
  useScheduleStore.setState({ project: makeEmptyProject(), past: [], future: [], selectedId: null, hydrated: false })
  resetHistorySession() // lastPush 是模块私有：跨用例清合并游标
}

describe('undo/redo 快照历史', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(0)
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('addTask → undo 消失 → redo 回来', () => {
    fresh()
    useScheduleStore.getState().addTask()
    expect(useScheduleStore.getState().project.tasks).toHaveLength(1)
    useScheduleStore.getState().undo()
    expect(useScheduleStore.getState().project.tasks).toHaveLength(0)
    useScheduleStore.getState().redo()
    expect(useScheduleStore.getState().project.tasks).toHaveLength(1)
  })

  it('removeTask 可撤销；redo 后再编辑则 future 清空', () => {
    fresh()
    useScheduleStore.getState().addTask()
    const id = useScheduleStore.getState().selectedId!
    useScheduleStore.getState().removeTask(id)
    expect(useScheduleStore.getState().project.tasks).toHaveLength(0)
    useScheduleStore.getState().undo()
    expect(useScheduleStore.getState().project.tasks).toHaveLength(1)
    useScheduleStore.getState().redo()
    expect(useScheduleStore.getState().project.tasks).toHaveLength(0)
    useScheduleStore.getState().undo()
    expect(useScheduleStore.getState().future).toHaveLength(1)
    useScheduleStore.getState().addTask() // 新编辑清空 future
    expect(useScheduleStore.getState().future).toHaveLength(0)
  })

  it('文本编辑合并：800ms 内同 kind+id 连续 updateTask 只压一次栈', () => {
    fresh()
    let now = 0
    vi.spyOn(Date, 'now').mockImplementation(() => now)
    useScheduleStore.getState().addTask()
    const id = useScheduleStore.getState().selectedId!
    const before = useScheduleStore.getState().past.length
    useScheduleStore.getState().updateTask(id, { duration: 5 })
    now += 100
    useScheduleStore.getState().updateTask(id, { duration: 6 })
    now += 100
    useScheduleStore.getState().updateTask(id, { duration: 7 })
    expect(useScheduleStore.getState().past).toHaveLength(before + 1)
    useScheduleStore.getState().undo()
    expect(useScheduleStore.getState().project.tasks[0].duration).toBe(3) // 回到整段编辑前
    now += 2000 // 超出合并窗口：再次编辑为新会话
    useScheduleStore.getState().updateTask(id, { duration: 9 })
    expect(useScheduleStore.getState().past).toHaveLength(2) // undo 已弹出 1，新会话再压 1
    expect(useScheduleStore.getState().project.tasks[0].duration).toBe(9)
  })

  it('整体替换（导入/清空）强制压栈：清空后可撤销找回原计划', () => {
    fresh()
    useScheduleStore.getState().addTask()
    const withTask = useScheduleStore.getState().project
    useScheduleStore.getState().clearAll()
    expect(useScheduleStore.getState().project.tasks).toHaveLength(0)
    useScheduleStore.getState().undo()
    expect(useScheduleStore.getState().project.tasks.length).toBe(withTask.tasks.length)
  })

  it('历史上限 50：超过截断最旧', () => {
    fresh()
    let now = 0
    vi.spyOn(Date, 'now').mockImplementation(() => now)
    useScheduleStore.getState().addTask()
    const id = useScheduleStore.getState().selectedId!
    for (let i = 0; i < 55; i++) {
      now += 2000 // 每次都超合并窗口
      useScheduleStore.getState().updateTask(id, { duration: i })
    }
    expect(useScheduleStore.getState().past).toHaveLength(50)
  })

  it('外部回读（setState 直改）不入史：undo 不会跳到外部快照', () => {
    fresh()
    useScheduleStore.getState().addTask()
    const before = useScheduleStore.getState().past.length
    const external: SchedProject = { ...useScheduleStore.getState().project, name: '外部改名' }
    useScheduleStore.setState({ project: external }) // agent 回读同语义
    expect(useScheduleStore.getState().past).toHaveLength(before)
  })
})
