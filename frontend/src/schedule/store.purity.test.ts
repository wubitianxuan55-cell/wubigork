/**
 * store.purity.test.ts — FE7-11：set 更新器纯度（模块级可变游标/副作用不得在更新器内改写）
 *
 * 手法：包一层 zustand `create`，把「动作传给 set 的更新器函数」捕获下来（生产代码零改动），
 * 再按 React StrictMode / 任何重放路径的语义**双跑同一个更新器**，断言：
 *  ① 双跑不重复消费模块级 idSeq（addTask/addGroup 两次执行得到同一个 id）；
 *  ② 双跑不刷新 pushHistory 的 800ms 合并游标（不吞掉后续本该入史的撤销步）；
 *  ③ fail-closed 拒绝的更新（分组行挂分配）不入史、不清 redo 栈（副作用不得在拒绝路径发生）。
 *
 * 反向证据：把任一 `pushHistory(...)` / `newId(...)` 搬回 `set((s) => ...)` 更新器体内，
 * 对应用例必红（配方见 docs/code-audit-2026-10-02 批次十四报告）。
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeEmptyProject } from './sample'
import { resetHistorySession, useScheduleStore } from './store'
import type { SchedProject } from './types'

/** 捕获到的 set 更新器（模块私有；zustand 只在动作里以函数形态调用 set） */
const cap = vi.hoisted(() => ({ updaters: [] as unknown[] }))

type SetFn = (...args: unknown[]) => unknown
type Creator = (set: SetFn, get: () => unknown, api: unknown) => unknown

vi.mock('zustand', async (importOriginal) => {
  const actual = await importOriginal<typeof import('zustand')>()
  const actualCreate = actual.create as unknown as (creator: Creator) => unknown
  const wrap = (creator: Creator): Creator => (set, get, api) =>
    creator((...args) => {
      if (typeof args[0] === 'function') cap.updaters.push(args[0])
      return set(...args)
    }, get, api)
  const mockCreate = (creator?: Creator) =>
    creator ? actualCreate(wrap(creator)) : (c: Creator) => actualCreate(wrap(c))
  return { ...actual, create: mockCreate } as unknown as typeof actual
})

type State = ReturnType<typeof useScheduleStore.getState>
type Updater = (s: State) => Partial<Pick<State, 'project' | 'selectedId'>>

/** 取最近一次动作里的更新器（每个用例都先清空 cap，故即本次动作的更新器） */
function lastUpdater(): Updater {
  expect(cap.updaters.length).toBeGreaterThan(0)
  return cap.updaters[cap.updaters.length - 1] as Updater
}

function fresh(): void {
  useScheduleStore.setState({ project: makeEmptyProject(), past: [], future: [], selectedId: null, hydrated: false })
  resetHistorySession() // lastPush 是模块私有：跨用例清合并游标
}

describe('FE7-11 set 更新器纯度', () => {
  beforeEach(() => {
    cap.updaters.length = 0
    fresh()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('addTask：同一更新器双跑不重复消费 idSeq（id 不变）', () => {
    const s0 = useScheduleStore.getState()
    cap.updaters.length = 0
    useScheduleStore.getState().addTask()
    expect(cap.updaters).toHaveLength(1)
    const updater = lastUpdater()

    const run1 = updater(s0)
    const run2 = updater(s0)

    expect(run1.project!.tasks).toHaveLength(1)
    expect(run2.project!.tasks[0].id).toBe(run1.project!.tasks[0].id)
    expect(run2).toEqual(run1)
  })

  it('addGroup：同一更新器双跑不重复消费 idSeq（两个 id 均不变）', () => {
    const s0 = useScheduleStore.getState()
    cap.updaters.length = 0
    useScheduleStore.getState().addGroup()
    expect(cap.updaters).toHaveLength(1)
    const updater = lastUpdater()

    const run1 = updater(s0)
    const run2 = updater(s0)

    expect(run2.project!.tasks.map((t) => t.id)).toEqual(run1.project!.tasks.map((t) => t.id))
    expect(run2).toEqual(run1)
  })

  it('updateTask：重放更新器不刷新 800ms 合并游标（不吞撤销步）', () => {
    let now = 0
    vi.spyOn(Date, 'now').mockImplementation(() => now)

    useScheduleStore.getState().addTask()
    const id = useScheduleStore.getState().selectedId!
    cap.updaters.length = 0
    useScheduleStore.getState().updateTask(id, { duration: 5 }) // 真实编辑 1：压栈 + 游标 at=0
    const pastAfterEdit = useScheduleStore.getState().past.length
    expect(pastAfterEdit).toBe(2) // addTask 1 + updateTask 1
    const updater = lastUpdater()

    now = 700
    updater(useScheduleStore.getState()) // 模拟重放/双跑：纯更新器不得续期合并窗口

    now = 1000
    useScheduleStore.getState().updateTask(id, { duration: 6 }) // 距首次编辑 1000ms > 800ms → 必开新会话
    expect(useScheduleStore.getState().past).toHaveLength(pastAfterEdit + 1)
    expect(useScheduleStore.getState().project.tasks[0].duration).toBe(6)
  })
})

describe('FE7-11 fail-closed 拒绝路径不产生副作用', () => {
  beforeEach(() => {
    cap.updaters.length = 0
    fresh()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('分组行 setTaskAssignments：不压栈、不清 future、project 原样', () => {
    const project: SchedProject = {
      ...makeEmptyProject(),
      tasks: [
        { id: 'G1', name: '分组', duration: 0, level: 0, progress: 0 },
        { id: 'A', name: '任务', duration: 3, level: 1, progress: 0 },
      ],
      resources: [{ id: 'r1', name: '资源', type: 'work' }],
    }
    useScheduleStore.setState({ project })
    // 造一个 redo 栈：编辑 → 撤销（future 非空、past 已清）
    useScheduleStore.getState().updateTask('A', { duration: 5 })
    useScheduleStore.getState().undo()
    expect(useScheduleStore.getState().future).toHaveLength(1)
    const pastBefore = useScheduleStore.getState().past.length

    useScheduleStore.getState().setTaskAssignments('G1', [{ taskId: 'G1', resourceId: 'r1' }])

    expect(useScheduleStore.getState().project).toBe(project) // 拒绝：模型原样（既有用例口径）
    expect(useScheduleStore.getState().past).toHaveLength(pastBefore) // 旧写法：+1（空撤销步）
    expect(useScheduleStore.getState().future).toHaveLength(1) // 旧写法：被 pushHistory 清空
  })
})
