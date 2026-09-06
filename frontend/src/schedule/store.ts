/**
 * schedule/store.ts — 进度计划板块状态（zustand + 文件持久化 v4.113.0 刀4）
 *
 * 计划文件（进度计划/当前计划.gsched.json）是板块与 agent 的共享资产：
 * 挂载时经 GaeaScheduleLoad 水合（文件不存在则把 localStorage 旧数据迁移上
 * 文件），编辑后防抖自动保存（GaeaScheduleSave，Go 侧校验+CPM fail-closed）。
 * localStorage persist 保留为离线缓存与迁移源。agent 写文件后板块靠
 * initScheduleSync 的 focus/可见轮询回读（15s 轻扫，不脏写时才覆盖）。
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { LinkType, SchedCalendar, SchedLink, SchedProject, SchedTask } from './types'
import { makeEmptyProject, makeSampleProject } from './sample'
import { normalizeCalendar } from './calendar'
import { loadScheduleFile, saveScheduleFile } from './api'

export type ScheduleView = 'gantt' | 'pdm' | 'aoa'

/** 文件同步状态（工具栏指示器） */
export type ScheduleSyncState = 'idle' | 'dirty' | 'saving' | 'saved' | 'error'

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
  /** 文件同步：hydrated=已完成水合；sync=指示器；savedAt/syncError 供展示 */
  hydrated: boolean
  sync: ScheduleSyncState
  savedAt: string | null
  syncError: string | null
  setView: (v: ScheduleView) => void
  select: (id: string | null) => void
  renameProject: (name: string) => void
  setStartDate: (d: string) => void
  /** 整体替换工程（XML 导入/文件水合），缺省字段归一 */
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
      hydrated: false,
      sync: 'idle',
      savedAt: null,
      syncError: null,

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

// ── 文件同步（v4.113.0 刀4）────────────────────────────────
// 水合 → 防抖自动保存 → agent 写入回读（focus/可见轻扫）。终态语义：
// 板块不弹窗打断编辑，失败落 syncError 由指示器诚实展示。

let syncStarted = false
let hydrating = false
let applyingExternal = false
let saveTimer: ReturnType<typeof setTimeout> | null = null
/** 最近一次与文件达成一致的 JSON（编辑防抖期间不被外部回读覆盖的基准） */
let lastSyncedRaw = ''
/** agent 写计划的即时回读触发（App 工具事件喂入：schedule_apply 成功回执即调） */
let pollNow: (() => void) | null = null
export function notifyScheduleFileChanged(): void {
  pollNow?.()
}

function projectRaw(p: SchedProject): string {
  return JSON.stringify(normalizeProject(p))
}

async function doSave(): Promise<void> {
  const { project } = useScheduleStore.getState()
  useScheduleStore.setState({ sync: 'saving' })
  try {
    const r = await saveScheduleFile(project)
    lastSyncedRaw = projectRaw(project)
    useScheduleStore.setState({ sync: 'saved', savedAt: r.savedAt, syncError: null })
  } catch (e) {
    useScheduleStore.setState({ sync: 'error', syncError: e instanceof Error ? e.message : String(e) })
  }
}

/** 板块挂载时调用一次：文件水合 + 编辑自动保存 + agent 写入回读 */
export async function initScheduleSync(): Promise<void> {
  if (syncStarted) return
  syncStarted = true
  hydrating = true
  try {
    const r = await loadScheduleFile()
    if (r.exists && r.project) {
      lastSyncedRaw = projectRaw(r.project)
      useScheduleStore.setState({ project: r.project, hydrated: true, sync: 'saved' })
    } else {
      // 迁移：localStorage 旧数据（persist 中间件维护）上文件；无则落当前内存态
      const ls = typeof localStorage !== 'undefined' ? localStorage.getItem('gaea.schedule.v1') : null
      let seed = useScheduleStore.getState().project
      try {
        const legacy = ls ? (JSON.parse(ls)?.state?.project as SchedProject | undefined) : undefined
        if (legacy) seed = normalizeProject(legacy)
      } catch { /* 坏缓存忽略，落内存态 */ }
      useScheduleStore.setState({ project: seed, hydrated: true })
      lastSyncedRaw = projectRaw(seed)
      await doSave()
    }
  } catch (e) {
    useScheduleStore.setState({ hydrated: true, sync: 'error', syncError: e instanceof Error ? e.message : String(e) })
  } finally {
    hydrating = false
  }

  // 编辑 → 防抖 800ms 自动保存（外部回读不触发）
  useScheduleStore.subscribe((s, prev) => {
    if (applyingExternal || hydrating || !s.hydrated) return
    if (s.project === prev.project) return
    const st = useScheduleStore.getState()
    if (st.sync !== 'error' && st.sync !== 'dirty') useScheduleStore.setState({ sync: 'dirty' })
    if (saveTimer !== null) clearTimeout(saveTimer)
    saveTimer = setTimeout(() => { saveTimer = null; void doSave() }, 800)
  })

  // agent 写文件回读：focus + 15s 可见轻扫（无未保存变更/失败态时才覆盖）
  const poll = async (): Promise<void> => {
    const st = useScheduleStore.getState()
    if (!st.hydrated || hydrating || saveTimer !== null) return
    if (st.sync === 'saving' || st.sync === 'dirty' || st.sync === 'error') return
    try {
      const r = await loadScheduleFile()
      if (!r.exists || !r.project) return
      const raw = projectRaw(r.project)
      if (raw !== lastSyncedRaw) {
        lastSyncedRaw = raw
        applyingExternal = true
        try {
          useScheduleStore.setState({ project: r.project, sync: 'saved', syncError: null })
        } finally {
          applyingExternal = false
        }
      }
    } catch { /* 轻扫失败静默，下轮再试 */ }
  }
  pollNow = () => { void poll() }
  if (typeof window !== 'undefined' && typeof document !== 'undefined') {
    window.addEventListener('focus', () => { void poll() })
    setInterval(() => { if (document.visibilityState === 'visible') void poll() }, 15000)
  }
}
