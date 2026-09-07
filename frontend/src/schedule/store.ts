/**
 * schedule/store.ts — 进度计划板块状态（zustand + 文件持久化 v4.113.0 刀4）
 *
 * 计划文件（进度计划/*.gsched.json，每文件一工程）是板块与 agent 的共享资产：
 * 挂载时经 GaeaScheduleLoad 水合当前指针指向的工程（文件不存在则把 localStorage
 * 旧数据迁移上文件），编辑后防抖自动保存（GaeaScheduleSave 显式带当前工程
 * rel，Go 侧校验+CPM fail-closed）。localStorage persist 保留为离线缓存与迁移
 * 源（多工程下降级为「当前工程缓存」，切换后旧缓存被覆盖是既定口径）。agent
 * 写文件后板块靠 initScheduleSync 的 focus/可见轮询回读（15s 轻扫，不脏写时
 * 才覆盖）。多工程切换/新建/归档/删除见文件尾（v4.139 #15 刀1，设计文档
 * docs/gaea-schedule-multi-project-design-2026-09.md §3.3/3.4）。
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { AoaPin, LinkType, SchedAssignment, SchedCalendar, SchedLink, SchedProject, SchedResource, SchedTask } from './types'
import { makeEmptyProject, makeSampleProject } from './sample'
import { normalizeCalendar } from './calendar'
import { computeCpm } from './cpm'
import { planFinishOf } from './deadline'
import {
  archiveScheduleProject,
  copyScheduleProject,
  createScheduleProject,
  deleteScheduleProject,
  listScheduleProjects,
  loadScheduleFile,
  openScheduleProject,
  saveScheduleFile,
} from './api'
import type { ScheduleProjectSummary } from './api'
import { snapshotBaseline, upsertBaseline } from './baseline'
import { normalizeAoaLayout } from './aoaLayout'

/**
 * 缺省工程文件 rel（v4.139 #15 刀1 起语义降级为「缺省值」）：单工程老用户/
 * 指针未切换时 currentPath 即此值；水合后一律以 LoadResult.path 为准，展示
 * 口径（状态栏 title/办公卡 isCurrentPlan 等）应读 store 的 currentPath 而
 * 不再直读常量。此处镜像 gschedSummary.ts 的 SCHEDULE_FILE_PATH（常量本体
 * 不动）而不直接转引，是因为 gschedSummary 已 import 本模块的 normalizeProject
 * ——再加 store→gschedSummary 一条边即成模块环，gschedSummary 先于本模块
 * 求值时该 const 尚未初始化（TDZ），会在其先加载的入口（如办公卡）崩溃。
 */
export const DEFAULT_SCHEDULE_PATH = '进度计划/当前计划.gsched.json'

export type ScheduleView = 'gantt' | 'pdm' | 'aoa' | 'usage'

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
  /**
   * 导入为新工程（v4.144）：以导入工程名落独立文件（Go slug 去重+登记索引+
   * 切指针）→ 内容替换 → 立即冲刷 → 刷列表。非破坏口径：绝不触碰导入前的
   * 当前工程（同 v4.139 复制=切指针）。返回 false=Create 失败（syncError
   * 已置位、当前工程原状），UI 据此出诚实提示。
   */
  importAsProject: (p: SchedProject) => Promise<boolean>
  setCalendar: (cal: SchedCalendar) => void
  /** 目标竣工日期（null=清除），倒排校核用 */
  setDeadline: (d: string | null) => void
  /**
   * 保存/更新基线（快照当前排程）。返回 null=成功；否则为失败原因
   * （循环依赖/无叶任务），UI 据此提示——不静默。
   * v4.137 #11：同名=原位更新，否则入槽（FIFO 上限 3）并设为活跃。
   */
  setBaseline: (name?: string) => string | null
  /** 清除基线（无基线时为无害空操作；只清活跃指针，槽位保留可再激活） */
  clearBaseline: () => void
  /** 切换活跃基线槽（v4.137 #11 多基线；名字必须在槽位中） */
  activateBaseline: (name: string) => void
  /** 删除基线槽（v4.137 #11；删活跃槽则活跃指针一并清空） */
  removeBaseline: (name: string) => void
  /** 改自定义字段列名（v4.138 #14：项目级覆盖注册表 label；空串=恢复缺省） */
  setCustomLabel: (key: string, label: string) => void
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
  /** 多级撤消/重做（刀C 余项）：编制动作快照式历史，上限 50，不入持久化 */
  past: SchedProject[]
  future: SchedProject[]
  undo: () => void
  redo: () => void
  // ── 多工程（v4.139 #15 刀1，docs/gaea-schedule-multi-project-design-2026-09.md §3.3/3.4）──
  /**
   * 当前工程文件 rel：初值=缺省路径（DEFAULT_SCHEDULE_PATH），水合后=
   * LoadResult.path（外部回读/轻扫同样校正）。localStorage persist 结构不变，
   * 仍只缓存当前工程——切换后旧缓存被覆盖是既定降级（设计 §6 风险 5）。
   */
  currentPath: string
  /** 工程列表缓存（GaeaScheduleProjects 索引视图；refreshProjects 维护，失败静默=空列表由 UI 出空态） */
  projects: ScheduleProjectSummary[]
  /**
   * 切换当前工程（测试重点）：sync=dirty 先用旧 currentPath 冲刷保存（await）→
   * Open 切指针 → 重水合（新工程新历史：past/future 清空、selectedId 置空、
   * 合并游标复位）。失败 fail-closed：保持原工程/currentPath 不变，只落
   * syncError 提示，绝不静默半切换。
   */
  openProject: (rel: string) => Promise<void>
  /** 新建工程（Go slug 落盘+登记索引+自动切为当前）→ openProject 同款重水合收尾 */
  createProject: (name: string) => Promise<void>
  /** 归档/反归档（文件保留原位，仅索引标记）→ 刷新列表缓存；指针与返回 current 不一致时重水合 */
  archiveProject: (rel: string, archived: boolean) => Promise<void>
  /** 删除工程（物理删文件，不可恢复）→ 刷新列表缓存；删的是当前工程时重水合到返回的 current */
  deleteProject: (rel: string) => Promise<void>
  /**
   * 复制工程（v4.145 另存为）：Go 深拷贝源文件落新 slug+登记索引，指针不动
   * （文件级动作，同归档）。rel 是当前工程时先冲刷在途编辑（拷的是已存盘内容）。
   * 返回 false=失败（syncError 已置位）。
   */
  copyProject: (rel: string, name: string) => Promise<boolean>
  /** 刷新工程列表缓存（失败静默：保留旧缓存/初值空列表） */
  refreshProjects: () => Promise<void>
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

/** 撤销历史上限（编制动作快照数） */
const HISTORY_CAP = 50
/** 文本编辑合并窗口：同 (kind,id) 的连续编辑 800ms 内只压一次栈（打字不逐键入史） */
const HISTORY_COALESCE_MS = 800
let lastPush: { kind: string; id: string; at: number } | null = null

/**
 * 压栈当前计划（动作落库前调用）。kind 提供时做合并窗口判断——连续同类
 * 编辑只保留会话开始前的快照（undo 一次回到整段打字之前）；整体替换
 * （导入/示例/清空）kind 缺省强制压栈。外部回读/水合走 setState 直改，
 * 不经本函数=天然不入史。
 */
function pushHistory(kind?: string, id?: string): void {
  const st = useScheduleStore.getState()
  if (kind) {
    const now = Date.now()
    if (lastPush && lastPush.kind === kind && lastPush.id === id && now - lastPush.at < HISTORY_COALESCE_MS) {
      lastPush.at = now
      return
    }
    lastPush = { kind, id: id ?? '', at: now }
  } else {
    lastPush = null
  }
  const past = [...st.past, st.project]
  useScheduleStore.setState({ past: past.slice(-HISTORY_CAP), future: [] })
}

/** 测试/特殊场景用：清掉合并会话游标（跨用例隔离） */
export function resetHistorySession(): void {
  lastPush = null
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
      past: [],
      future: [],
      // 多工程（v4.139 #15 刀1）：currentPath 初值=缺省常量；projects=空缓存
      currentPath: DEFAULT_SCHEDULE_PATH,
      projects: [],

      undo: () => {
        const st = useScheduleStore.getState()
        if (st.past.length === 0) return
        lastPush = null
        useScheduleStore.setState({
          past: st.past.slice(0, -1),
          future: [...st.future, st.project],
          project: st.past[st.past.length - 1],
        })
      },
      redo: () => {
        const st = useScheduleStore.getState()
        if (st.future.length === 0) return
        lastPush = null
        useScheduleStore.setState({
          past: [...st.past, st.project],
          future: st.future.slice(0, -1),
          project: st.future[st.future.length - 1],
        })
      },

      setView: (view) => set({ view }),
      select: (selectedId) => set({ selectedId }),
      renameProject: (name) => set((s) => { pushHistory('rename'); return { project: { ...s.project, name } } }),
      setStartDate: (startDate) => set((s) => { pushHistory('startDate'); return { project: { ...s.project, startDate } } }),
      importProject: (p) => { pushHistory(); set({ project: normalizeProject(p), selectedId: null }) },
      importAsProject: (p) => importThenSwitch(p),
      setCalendar: (calendar) => set((s) => { pushHistory('calendar'); return { project: { ...s.project, calendar: normalizeCalendar(calendar) } } }),
      setDeadline: (d) => set((s) => { pushHistory('deadline'); return { project: { ...s.project, deadline: d } } }),

      setBaseline: (name) => {
        pushHistory('baseline')
        const { project } = useScheduleStore.getState()
        const cpm = computeCpm(project.tasks, project.links, { planFinish: planFinishOf(project) })
        const r = snapshotBaseline(project, cpm, formatNow(), name)
        if (!r.ok) return r.error
        // v4.137 #11 多基线：存快照进槽位列表（同名原位替换，FIFO 上限 3），并设为活跃
        const baselines = upsertBaseline(project.baselines, r.baseline)
        useScheduleStore.setState({ project: { ...project, baseline: r.baseline, baselines } })
        return null
      },
      clearBaseline: () => set((s) => { pushHistory('baseline'); return { project: { ...s.project, baseline: null } } }),
      /** 切换活跃基线（v4.137 #11）：必须是槽位里已有的名字，否则无害 no-op */
      activateBaseline: (name) => set((s) => {
        const b = s.project.baselines?.find((x) => x.name === name)
        if (!b || s.project.baseline?.name === name) return {}
        pushHistory('baseline')
        return { project: { ...s.project, baseline: b } }
      }),
      /** 删除基线槽（v4.137 #11）：删的是活跃槽时活跃指针一并置空（横道基线条消失，诚实） */
      removeBaseline: (name) => set((s) => {
        if (!s.project.baselines?.some((b) => b.name === name)) return {}
        pushHistory('baseline')
        return {
          project: {
            ...s.project,
            baselines: s.project.baselines!.filter((b) => b.name !== name),
            baseline: s.project.baseline?.name === name ? null : s.project.baseline,
          },
        }
      }),
      /** 改自定义字段列名（v4.138 #14）：空串=恢复缺省（删覆盖键） */
      setCustomLabel: (key, label) => set((s) => {
        pushHistory('customLabel')
        const next = { ...(s.project.customLabels ?? {}) }
        const trimmed = label.trim()
        if (trimmed === '') delete next[key]
        else next[key] = trimmed
        return { project: { ...s.project, customLabels: next } }
      }),

      addTask: (afterId) => set((s) => {
        pushHistory('addTask')
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
        pushHistory('addGroup')
        const tasks = [...s.project.tasks]
        const group: SchedTask = { id: newId('g'), name: '新分组', duration: 0, level: 0, progress: 0 }
        const child: SchedTask = { id: newId('t'), name: '新任务', duration: 3, level: 1, progress: 0 }
        tasks.push(group, child)
        return { project: { ...s.project, tasks }, selectedId: group.id }
      }),

      updateTask: (id, patch) => set((s) => {
        pushHistory('updateTask', id)
        return {
          project: {
            ...s.project,
            tasks: s.project.tasks.map((t) => (t.id === id ? { ...t, ...patch } : t)),
          },
        }
      }),

      removeTask: (id) => set((s) => {
        pushHistory('removeTask')
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
        pushHistory('preds', taskId)
        const kept = s.project.links.filter((l) => l.to !== taskId)
        const added = preds
          .filter((p) => p.from && p.from !== taskId)
          .map((p): SchedLink => ({ from: p.from, to: taskId, type: p.type, lag: Math.round(p.lag) || 0 }))
        return { project: { ...s.project, links: [...kept, ...added] } }
      }),

      // ── 资源成本（v4.124 刀3）：提交即写 project.resources/assignments，
      // 走既有防抖自动保存链路（编辑 → dirty → 800ms → GaeaScheduleSave）。
      upsertResource: (r) => set((s) => {
        pushHistory('resource')
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
          // 个人日历（v4.137 #13）：仅工时资源有按天可用性语义；周历至少一项+字段逐项容错
          calendar: type === 'work' && r.calendar && Array.isArray(r.calendar.workweek) && r.calendar.workweek.length > 0
            ? {
                workweek: [...new Set(r.calendar.workweek.filter((n) => Number.isInteger(n) && n >= 0 && n <= 6))].sort((a, b) => a - b),
                holidays: Array.isArray(r.calendar.holidays)
                  ? [...new Set(r.calendar.holidays.filter((h) => typeof h === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(h)))].sort()
                  : [],
              }
            : undefined,
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
        pushHistory('assign', taskId)
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

      setAoaPins: (pins) => set((s) => { pushHistory('aoaPins'); return { project: { ...s.project, aoaLayout: { pins } } } }),

      loadSample: () => { pushHistory(); set({ project: makeSampleProject(), selectedId: null }) },
      clearAll: () => { pushHistory(); set({ project: makeEmptyProject(), selectedId: null }) },

      // ── 多工程（v4.139 #15 刀1）：实现放在下方模块函数——切换流程要触达
      // doSave/防抖计时器/水合锁等文件同步内部状态，不宜塞进 creator 闭包。
      openProject: (rel) => switchProject(rel),
      createProject: (name) => createThenSwitch(name),
      archiveProject: (rel, archived) => mutateProjectList(() => archiveScheduleProject(rel, archived)),
      deleteProject: (rel) => mutateProjectList(() => deleteScheduleProject(rel)),
      copyProject: (rel, name) => copyProjectFile(rel, name),
      refreshProjects: () => refreshProjectsCache(),
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
  // project 与 currentPath 同刻快照（v4.139 #15 刀1）：保存目标永远=装载时的
  // 工程文件。防抖在途时切工程由 switchProject 先冲刷后切指针兜底——即使保存
  // 迟到也落在旧 rel 上，结构性防「旧内容写进新文件」竞态（设计 §3.2 方案 A）。
  const { project, currentPath } = useScheduleStore.getState()
  useScheduleStore.setState({ sync: 'saving' })
  try {
    const r = await saveScheduleFile(project, currentPath)
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
      // currentPath 以文件回执为准（多工程指针解析在 Go 侧），展示口径随之
      useScheduleStore.setState({ project: r.project, currentPath: r.path, hydrated: true, sync: 'saved' })
    } else {
      // 迁移：localStorage 旧数据（persist 中间件维护）上文件；无则落当前内存态
      const ls = typeof localStorage !== 'undefined' ? localStorage.getItem('gaea.schedule.v1') : null
      let seed = useScheduleStore.getState().project
      try {
        const legacy = ls ? (JSON.parse(ls)?.state?.project as SchedProject | undefined) : undefined
        if (legacy) seed = normalizeProject(legacy)
      } catch { /* 坏缓存忽略，落内存态 */ }
      useScheduleStore.setState({ project: seed, currentPath: r.path, hydrated: true })
      lastSyncedRaw = projectRaw(seed)
      await doSave()
    }
  } catch (e) {
    useScheduleStore.setState({ hydrated: true, sync: 'error', syncError: e instanceof Error ? e.message : String(e) })
  } finally {
    hydrating = false
  }

  // 工程列表缓存顺带刷新（v4.139 #15 刀1；失败静默=空列表，切换器自出空态）
  void refreshProjectsCache()

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
          // 外部回读同样校正 currentPath（agent/办公卡切指针后，轻扫即跟上）
          useScheduleStore.setState({ project: r.project, currentPath: r.path, sync: 'saved', syncError: null })
        } finally {
          applyingExternal = false
        }
      } else if (r.path !== st.currentPath) {
        // 内容未变但指针变了：只校正路径，不动工程内容
        useScheduleStore.setState({ currentPath: r.path })
      }
    } catch { /* 轻扫失败静默，下轮再试 */ }
  }
  pollNow = () => { void poll() }
  if (typeof window !== 'undefined' && typeof document !== 'undefined') {
    window.addEventListener('focus', () => { void poll() })
    setInterval(() => { if (document.visibilityState === 'visible') void poll() }, 15000)
  }
}

// ── 多工程切换/管理（v4.139 #15 刀1，设计 §3.3/3.4）────────────────────────
// 切换语义（§3.3）：GaeaScheduleProjectOpen 只校验 rel + 原子写索引 current，
// 不搬运数据不改文件。板块在切换前后自己守住两条纪律：
// ① dirty 先用「旧 currentPath」冲刷（Save 显式带目标文件，结构性防「防抖在途
//    时切工程，旧内容写进新文件」）；
// ② 切换=新工程新历史（重水合整体落位：past/future 清空、selectedId 置空、
//    文本合并游标复位），失败 fail-closed 保持原工程，不静默半切换。

/**
 * 重水合当前指针指向的工程（切换/新建/删当前后的统一收尾）。
 * Load → normalizeProject（api 层）→ 整体落位；currentPath 以 LoadResult.path
 * 为准。hydrating 锁与 initScheduleSync 水合分支同款：落位不经编辑订阅器，
 * 不触发 dirty/防抖。失败向上抛错，由调用方统一 fail-closed。
 */
async function rehydrateCurrent(): Promise<void> {
  const r = await loadScheduleFile()
  if (!r.exists || !r.project) throw new Error(`工程文件缺失或不可读：${r.path}`)
  hydrating = true
  try {
    lastSyncedRaw = projectRaw(r.project)
    useScheduleStore.setState({
      project: r.project,
      currentPath: r.path,
      selectedId: null,
      past: [],
      future: [],
      sync: 'saved',
      syncError: null,
    })
  } finally {
    hydrating = false
  }
  resetHistorySession()
}

/**
 * dirty 冲刷：作废防抖计时器并立即按当前 state（project+currentPath 同刻快照）
 * 落盘。多工程动作（切换/新建/归档/删除）前置调用——绑定调用改指针/删文件
 * 之前，旧工程的未保存编辑先落回旧 rel，既不丢编辑也杜绝迟到保存写错目标。
 */
async function flushDirty(): Promise<void> {
  if (useScheduleStore.getState().sync !== 'dirty') return
  if (saveTimer !== null) {
    clearTimeout(saveTimer)
    saveTimer = null
  }
  await doSave()
}

/** 切换当前工程（测试重点：dirty 冲刷带旧 rel → Open → 重水合；失败保持原工程） */
async function switchProject(rel: string): Promise<void> {
  // a. 冲刷：立即落盘，此刻 currentPath 仍是旧工程 rel
  await flushDirty()
  try {
    await openScheduleProject(rel)
    await rehydrateCurrent()
    void refreshProjectsCache() // 列表缓存活刷新（失败静默）
  } catch (e) {
    // d. fail-closed：指针/工程保持原状，错误交指示器诚实展示
    useScheduleStore.setState({ syncError: e instanceof Error ? e.message : String(e) })
  }
}

/** 新建工程后走 openProject 同款收尾：先冲刷旧工程未存编辑，再重水合到新 rel */
async function createThenSwitch(name: string): Promise<void> {
  await flushDirty()
  try {
    await createScheduleProject(name)
    await rehydrateCurrent()
    // Create 回执不含列表：显式刷新把新条目带进缓存
    await refreshProjectsCache()
  } catch (e) {
    useScheduleStore.setState({ syncError: e instanceof Error ? e.message : String(e) })
  }
}

/** 导入为新工程（v4.144）：先 Create 落空工程+切指针（slug 去重），成功才替换
 *  内容并立即冲刷。currentPath 未变=Create 失败（createThenSwitch 已置
 *  syncError）→ 返回 false，当前工程保持原状（fail-closed）。名字空兜底
 *  「导入工程」，slug、文件内容、列表摘要三处同源此名。 */
async function importThenSwitch(p: SchedProject): Promise<boolean> {
  const named: SchedProject = { ...p, name: p.name?.trim() || '导入工程' }
  const before = useScheduleStore.getState().currentPath
  await createThenSwitch(named.name)
  if (useScheduleStore.getState().currentPath === before) return false
  useScheduleStore.getState().importProject(named)
  await flushDirty()
  await refreshProjectsCache()
  return true
}

/** 复制工程（v4.145 另存为）：rel 是当前工程时先冲刷在途编辑——拷的是已存盘
 *  内容，不含未落盘编辑；Go 深拷贝落新 slug+登记索引，指针不动。失败只落
 *  syncError 返回 false；成功以回执列表入缓存（不另刷）。 */
async function copyProjectFile(rel: string, name: string): Promise<boolean> {
  if (rel === useScheduleStore.getState().currentPath) {
    await flushDirty()
  }
  try {
    const r = await copyScheduleProject(rel, name)
    useScheduleStore.setState({ projects: r.projects })
    return true
  } catch (e) {
    useScheduleStore.setState({ syncError: e instanceof Error ? e.message : String(e) })
    return false
  }
}

/** Archive/Delete 后的统一收尾：列表缓存取回执；指针与返回 current 不一致（删了当前工程等）则重水合 */
async function mutateProjectList(
  call: () => Promise<{ current: string; projects: ScheduleProjectSummary[] }>,
): Promise<void> {
  await flushDirty()
  try {
    const r = await call()
    if (r.current !== useScheduleStore.getState().currentPath) {
      await rehydrateCurrent() // 当前工程被删/指针被切：Go 已切到返回的 current
    }
    useScheduleStore.setState({ projects: r.projects })
  } catch (e) {
    useScheduleStore.setState({ syncError: e instanceof Error ? e.message : String(e) })
  }
}

/** 工程列表缓存刷新（失败静默=保留旧缓存/初值空列表，UI 自己出空态） */
async function refreshProjectsCache(): Promise<void> {
  try {
    const r = await listScheduleProjects()
    useScheduleStore.setState({ projects: r.projects })
  } catch { /* 静默 */ }
}
