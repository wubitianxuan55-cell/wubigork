/**
 * schedule/store.ts — 进度计划板块状态（zustand + localStorage 持久化）
 *
 * 刀1 前端自治：数据落 localStorage（gaea.schedule.v1），与壳层
 * shellPage 持久化同范式；接后端绑定面留给后续刀次。
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { LinkType, SchedCalendar, SchedLink, SchedProject, SchedTask } from './types'
import { makeEmptyProject, makeSampleProject } from './sample'
import { normalizeCalendar } from './calendar'

export type ScheduleView = 'gantt' | 'pdm' | 'aoa'

/** 待编辑的前置关系（Popover 行编辑形态） */
export interface PredDraft {
  from: string
  type: LinkType
  lag: number
}

/** 旧持久化数据兼容：补日历缺省（v4.110 数据无 calendar） */
export function normalizeProject(p: SchedProject): SchedProject {
  return {
    ...p,
    calendar: normalizeCalendar(p.calendar),
    tasks: (p.tasks ?? []).map((t) => ({ ...t, progress: t.progress ?? 0 })),
    links: p.links ?? [],
  }
}

interface ScheduleState {
  project: SchedProject
  view: ScheduleView
  selectedId: string | null
  setView: (v: ScheduleView) => void
  select: (id: string | null) => void
  renameProject: (name: string) => void
  setStartDate: (d: string) => void
  /** 整体替换工程（XML 导入），缺省字段归一 */
  importProject: (p: SchedProject) => void
  setCalendar: (cal: SchedCalendar) => void
  addTask: (afterId?: string) => void
  addGroup: () => void
  updateTask: (id: string, patch: Partial<SchedTask>) => void
  removeTask: (id: string) => void
  /** 整体替换某任务的前置关系（Popover 编辑提交） */
  setPreds: (taskId: string, preds: PredDraft[]) => void
  loadSample: () => void
  clearAll: () => void
}

let idSeq = Date.now() % 100000
const newId = (p: string) => `${p}${(idSeq++).toString(36)}${Math.floor(Math.random() * 1296).toString(36)}`

/** 判断任务是否为分组行（有子级：后面紧跟更深层级的行） */
export function isGroupRow(tasks: SchedTask[], idx: number): boolean {
  const cur = tasks[idx]
  if (!cur || cur.level !== 0) return false
  const next = tasks[idx + 1]
  return !!next && next.level > cur.level
}

/** 指定任务的子孙 id 集合（含自身；扁平数组 + level 语义） */
export function descendantIds(tasks: SchedTask[], id: string): Set<string> {
  const out = new Set<string>()
  const idx = tasks.findIndex((t) => t.id === id)
  if (idx < 0) return out
  const base = tasks[idx].level
  out.add(id)
  for (let i = idx + 1; i < tasks.length && tasks[i].level > base; i++) out.add(tasks[i].id)
  return out
}

export const useScheduleStore = create<ScheduleState>()(
  persist(
    (set) => ({
      project: makeSampleProject(),
      view: 'gantt',
      selectedId: null,

      setView: (view) => set({ view }),
      select: (selectedId) => set({ selectedId }),
      renameProject: (name) => set((s) => ({ project: { ...s.project, name } })),
      setStartDate: (startDate) => set((s) => ({ project: { ...s.project, startDate } })),
      importProject: (p) => set({ project: normalizeProject(p), selectedId: null }),
      setCalendar: (calendar) => set((s) => ({ project: { ...s.project, calendar: normalizeCalendar(calendar) } })),

      addTask: (afterId) => set((s) => {
        const tasks = [...s.project.tasks]
        const task: SchedTask = { id: newId('t'), name: '新任务', duration: 3, level: 1, progress: 0 }
        let at = tasks.length
        if (afterId) {
          const i = tasks.findIndex((x) => x.id === afterId)
          if (i >= 0) {
            at = i + 1
            // 插到选中行同级（若选中是分组行，则插入为其子级行位置，level 取下一行）
            task.level = Math.min(1, tasks[i].level === 0 ? 1 : tasks[i].level)
          }
        }
        tasks.splice(at, 0, task)
        return { project: { ...s.project, tasks }, selectedId: task.id }
      }),

      addGroup: () => set((s) => {
        const tasks = [...s.project.tasks]
        const group: SchedTask = { id: newId('g'), name: '新分组', duration: 0, level: 0, progress: 0 }
        const child: SchedTask = { id: newId('t'), name: '新任务', duration: 3, level: 1, progress: 0 }
        tasks.push(group, child)
        return { project: { ...s.project, tasks }, selectedId: group.id }
      }),

      updateTask: (id, patch) => set((s) => ({
        project: {
          ...s.project,
          tasks: s.project.tasks.map((t) => (t.id === id ? { ...t, ...patch } : t)),
        },
      })),

      removeTask: (id) => set((s) => {
        const kill = descendantIds(s.project.tasks, id)
        return {
          project: {
            ...s.project,
            tasks: s.project.tasks.filter((t) => !kill.has(t.id)),
            links: s.project.links.filter((l) => !kill.has(l.from) && !kill.has(l.to)),
          },
          selectedId: s.selectedId === id ? null : s.selectedId,
        }
      }),

      setPreds: (taskId, preds) => set((s) => {
        const kept = s.project.links.filter((l) => l.to !== taskId)
        const added = preds
          .filter((p) => p.from && p.from !== taskId)
          .map((p): SchedLink => ({ from: p.from, to: taskId, type: p.type, lag: Math.round(p.lag) || 0 }))
        return { project: { ...s.project, links: [...kept, ...added] } }
      }),

      loadSample: () => set({ project: makeSampleProject(), selectedId: null }),
      clearAll: () => set({ project: makeEmptyProject(), selectedId: null }),
    }),
    {
      name: 'gaea.schedule.v1',
      partialize: (s) => ({ project: normalizeProject(s.project) }) as unknown as ScheduleState,
      // v4.110 旧数据无 calendar/mode 字段：水合时归一补缺省
      merge: (persisted, current) => {
        const p = (persisted as Partial<ScheduleState>)?.project
        return { ...current, ...(p ? { project: normalizeProject(p) } : {}) }
      },
    },
  ),
)

/** 供组件读取的任务快捷方法（非 hook 场景用 getState） */
export const scheduleActions = {
  addTask: (afterId?: string) => useScheduleStore.getState().addTask(afterId),
  removeTask: (id: string) => useScheduleStore.getState().removeTask(id),
}
