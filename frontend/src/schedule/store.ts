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
import type { AoaPin, LinkType, SchedAssignment, SchedCalendar, SchedLink, SchedProject, SchedResource, SchedTask } from './types'
import { makeEmptyProject, makeSampleProject } from './sample'
import { normalizeCalendar } from './calendar'
import { computeCpm } from './cpm'
import { loadScheduleFile, saveScheduleFile } from './api'
import { snapshotBaseline } from './baseline'
import { normalizeAoaLayout } from './aoaLayout'

export type ScheduleView = 'gantt' | 'pdm' | 'aoa'

/** 文件同步状态（工具栏指示器） */
export type ScheduleSyncState = 'idle' | 'dirty' | 'saving' | 'saved' | 'error'

/** 待编辑的前置关系（Popover 行编辑形态） */
export interface PredDraft {
  from: string
  type: LinkType
  lag: number
}

/** 可选数值字段容错：缺省合法；出现时必须为有限非负数（资源成本刀1 口径） */
function validNum(v: unknown): boolean {
  return v === undefined || (typeof v === 'number' && Number.isFinite(v) && v >= 0)
}

/**
 * 数值字段容错写入（资源成本刀3）：合法（有限非负）透传，否则丢弃回落缺省
 * （undefined=引擎按缺省口径处理）。负数/NaN/Inf 的「拒绝」方式=丢弃字段，
 * 与 normalizeProject 坏形字段逐条丢弃同口径，不抛错打断编辑流。
 */
function numOrUndefined(v: unknown): number | undefined {
  return typeof v === 'number' && Number.isFinite(v) && v >= 0 ? v : undefined
}

/**
 * 旧持久化数据兼容：补日历缺省（v4.110 数据无 calendar）；deadline 非法格式丢弃；
 * 资源成本刀1（v4.122）：补 resources/assignments 空数组，坏形条目逐条丢弃
 * （容错进板块；落盘拒绝由 Go Validate fail-closed 承担——两道闸分工同现状）。
 */
export function normalizeProject(p: SchedProject): SchedProject {
  const resources = Array.isArray(p.resources)
    ? p.resources.filter(
        (r): r is SchedResource =>
          !!r && typeof r.id === 'string' && r.id !== '' && typeof r.name === 'string'
          && (r.type === 'work' || r.type === 'material' || r.type === 'cost')
          && validNum(r.standardRate) && validNum(r.costPerUse) && validNum(r.maxUnits),
      )
    : []
  const assignments = Array.isArray(p.assignments)
    ? p.assignments.filter(
        (a): a is SchedAssignment =>
          !!a && typeof a.taskId === 'string' && a.taskId !== ''
          && typeof a.resourceId === 'string' && a.resourceId !== ''
          && validNum(a.units) && validNum(a.quantity) && validNum(a.amount),
      )
    : []
  return {
    ...p,
    calendar: normalizeCalendar(p.calendar),
    tasks: (p.tasks ?? []).map((t) => ({
      ...t,
      progress: t.progress ?? 0,
      fixedCost: validNum(t.fixedCost) ? t.fixedCost : undefined,
    })),
    links: p.links ?? [],
    deadline: typeof p.deadline === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(p.deadline) ? p.deadline : null,
    resources,
    assignments,
    aoaLayout: normalizeAoaLayout(p.aoaLayout), // v4.123 AOA 刀1：结构归一+缺省
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
  /** 目标竣工日期（null=清除），倒排校核用 */
  setDeadline: (d: string | null) => void
  /**
   * 保存/更新基线（快照当前排程）。返回 null=成功；否则为失败原因
   * （循环依赖/无叶任务），UI 据此提示——不静默。
   */
  setBaseline: (name?: string) => string | null
  /** 清除基线（无基线时为无害空操作） */
  clearBaseline: () => void
  addTask: (afterId?: string) => void
  addGroup: () => void
  updateTask: (id: string, patch: Partial<SchedTask>) => void
  removeTask: (id: string) => void
  /** 整体替换某任务的前置关系（Popover 编辑提交） */
  setPreds: (taskId: string, preds: PredDraft[]) => void
  /** 新增/更新资源（v4.124 刀3：id 空=新增生成；数值容错见 numOrUndefined；类型切换联动清理无关字段） */
  upsertResource: (r: SchedResource) => void
  /** 删除资源并级联删除其全部分配 */
  removeResource: (id: string) => void
  /** 整体替换某任务的分配集（set_links「整体替换入边」同语义；分组行拒绝=no-op，fail-closed） */
  setTaskAssignments: (taskId: string, list: SchedAssignment[]) => void
  /** 双代号手动布局：整体替换 pins（拖拽提交/重置；v4.123 AOA 刀1） */
  setAoaPins: (pins: Record<string, AoaPin>) => void
  loadSample: () => void
  clearAll: () => void
}

let idSeq = Date.now() % 100000
const newId = (p: string) => `${p}${(idSeq++).toString(36)}${Math.floor(Math.random() * 1296).toString(36)}`

/** 基线保存时间戳（本地时间 YYYY-MM-DD HH:mm） */
function formatNow(): string {
  const d = new Date()
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

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
      setDeadline: (d) => set((s) => ({ project: { ...s.project, deadline: d } })),

      setBaseline: (name) => {
        const { project } = useScheduleStore.getState()
        const cpm = computeCpm(project.tasks, project.links)
        const r = snapshotBaseline(project, cpm, formatNow(), name)
        if (!r.ok) return r.error
        useScheduleStore.setState({ project: { ...project, baseline: r.baseline } })
        return null
      },
      clearBaseline: () => set((s) => ({ project: { ...s.project, baseline: null } })),

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
            // 级联删分配（v4.124 刀3）：悬空分配会被 Go Validate 拒收，须随任务一并清除
            assignments: s.project.assignments?.filter((a) => !kill.has(a.taskId)),
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

      // ── 资源成本（v4.124 刀3）：提交即写 project.resources/assignments，
      // 走既有防抖自动保存链路（编辑 → dirty → 800ms → GaeaScheduleSave）。
      upsertResource: (r) => set((s) => {
        const type = r.type === 'material' || r.type === 'cost' ? r.type : 'work'
        const id = typeof r.id === 'string' ? r.id : ''
        const clean: SchedResource = {
          id: id || newId('r'),
          name: typeof r.name === 'string' && r.name ? r.name : '未命名资源',
          type,
          // 类型切换联动清理：cost 无费率/每次使用（金额在分配上）；unit 仅材料；maxUnits 仅工时
          standardRate: type === 'cost' ? undefined : numOrUndefined(r.standardRate),
          costPerUse: type === 'cost' ? undefined : numOrUndefined(r.costPerUse),
          unit: type === 'material' && typeof r.unit === 'string' && r.unit ? r.unit : undefined,
          maxUnits: type === 'work' ? numOrUndefined(r.maxUnits) : undefined,
        }
        const resources = s.project.resources ?? []
        const next = resources.some((x) => x.id === clean.id)
          ? resources.map((x) => (x.id === clean.id ? clean : x))
          : [...resources, clean]
        return { project: { ...s.project, resources: next } }
      }),

      removeResource: (id) => set((s) => ({
        project: {
          ...s.project,
          resources: (s.project.resources ?? []).filter((r) => r.id !== id),
          assignments: (s.project.assignments ?? []).filter((a) => a.resourceId !== id), // 级联删其分配
        },
      })),

      setTaskAssignments: (taskId, list) => set((s) => {
        // 分组行禁挂分配（fail-closed：汇总唯一口径=子孙求和，UI 入口已隐藏）
        const task = s.project.tasks.find((t) => t.id === taskId)
        if (!task || task.level === 0) return {}
        const resIds = new Set((s.project.resources ?? []).map((r) => r.id))
        const seen = new Set<string>()
        const next: SchedAssignment[] = []
        for (const a of list) {
          // 整体替换前清洗：悬空资源/重复 (taskId,resourceId)/非法数值条目丢弃（同 normalize 容错口径）
          if (!a || typeof a.resourceId !== 'string' || !resIds.has(a.resourceId)) continue
          if (seen.has(a.resourceId)) continue
          seen.add(a.resourceId)
          next.push({
            taskId,
            resourceId: a.resourceId,
            units: numOrUndefined(a.units),
            quantity: numOrUndefined(a.quantity),
            amount: numOrUndefined(a.amount),
          })
        }
        const kept = (s.project.assignments ?? []).filter((a) => a.taskId !== taskId)
        return { project: { ...s.project, assignments: [...kept, ...next] } }
      }),

      setAoaPins: (pins) => set((s) => ({ project: { ...s.project, aoaLayout: { pins } } })),

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
