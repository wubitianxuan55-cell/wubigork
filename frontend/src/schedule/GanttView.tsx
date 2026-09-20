/**
 * schedule/GanttView.tsx — 横道图（甘特图）视图入口（v4.112.0 斑马 UI 对齐）
 *
 * 「瘦身 P3 结构版1：巨文件拆分」：渲染块迁入 schedule/gantt/ 子模块，本文件只留
 * 状态/派生计算/交互 handler 与组合层——导出面与行为零变化（classNames/testids 不变）。
 *
 * 子模块：
 *  - gantt/ganttUtil.ts      共享常量（ROW_H/W/缩放档位…）+ 纯函数（前置/后续/行号/
 *                            竣工偏移/分组成本/前锋点位/fmtDate），同 aoaRuler.ts 范式；
 *  - gantt/PredEditor.tsx    前置关系编辑 Popover；
 *  - gantt/CustomFieldCell.tsx 自定义字段单元格（text=Input / num=InputNumber）；
 *  - gantt/GanttToolbar.tsx  工具栏（前锋线/基线/筛选/排序/分组/路径/列菜单/缩放全览）；
 *  - gantt/GanttTable.tsx    表格窗格（列头 + 表体逐行，含行菜单/依赖引用/成本/自定义列）；
 *  - gantt/GanttTimeline.tsx 画布双层时标（月层 + 日/周刻度层）；
 *  - gantt/GanttBars.tsx     画布行（基线/里程碑/条形/总时差尾/拖拽预览）；
 *  - gantt/GanttOverlay.tsx  画布覆盖层（非工作日底纹/搭接箭线/今日线/竣工线/前锋线）。
 *
 * 视觉/交互口径（v4.131 刀C/4.136/4.138/4.140/4.141，详见各子模块头注释）：
 * 自然日时间轴 + 周末/节假日底纹；表格窗格|画布窗格真双栏（分隔条/列显隐持久化）；
 * 行序列唯一状态源（筛选→排序→多级分组→折叠 合成进 rows）；连续缩放+全览；Ctrl+滚轮。
 */
import React, { useMemo, useRef, useState } from 'react'
import type { CpmResult, SchedProject, SchedTask } from './types'
import { isWorkingDate, normalizeCalendar, wdToDate } from './calendar'
import { dropToWd, resizeToDuration, resizeToNaturalDuration, workdayOffsets } from './drag'
import { computeCosts } from './cost'
import { isCd } from './cpm'
import { ganttLinkPath } from './ganttLinks'
import { useScheduleStore } from './store'
import { hasCostData } from './costUi'
import { GANTT_LEFT_W_FULL, clampTableW, visibleCols, visibleLeftW } from './ganttCols'
import { GANTT_FILTER_DEFAULT, filterGanttRows, type GanttFilter } from './ganttFilter'
import { buildGanttRows, wbsOf, type GanttGroupField, type GanttSort } from './ganttGroup'
import { drivingChain, linkKey } from './pathDriver'
import { loadChatPrefs, saveChatPrefs } from './chatPrefs'
import {
  ROW_H,
  DAY_W_MIN,
  DAY_W_MAX,
  DAY_W_DEFAULT,
  ZOOM_FACTOR,
  WEEK_TIER_BELOW,
  deadlineOff,
  frontierWd,
  type GanttDragState,
} from './gantt/ganttUtil'
import { GanttToolbar } from './gantt/GanttToolbar'
import { GanttTable } from './gantt/GanttTable'
import { GanttTimeline } from './gantt/GanttTimeline'
import { GanttBars } from './gantt/GanttBars'
import { GanttOverlay } from './gantt/GanttOverlay'

export const GanttView: React.FC<{ project: SchedProject; cpm: CpmResult; onInspect?: (taskId: string) => void }> = ({ project, cpm, onInspect }) => {
  const selectedId = useScheduleStore((s) => s.selectedId)
  const select = useScheduleStore((s) => s.select)
  const updateTask = useScheduleStore((s) => s.updateTask)
  const setCustomLabel = useScheduleStore((s) => s.setCustomLabel)
  const addTask = useScheduleStore((s) => s.addTask)
  const addGroup = useScheduleStore((s) => s.addGroup)
  const removeTask = useScheduleStore((s) => s.removeTask)
  /** 日宽连续缩放（v4.141：步进档改连续值，zoom×1.35；全览=整计划适配窗口宽）。
   *  dayWRef 同步镜像：Ctrl+滚轮（v4.159）在监听器里读现值、不经渲染周期。 */
  const [dayW, setDayW] = useState(DAY_W_DEFAULT)
  const canvasPaneRef = useRef<HTMLDivElement | null>(null)
  const dayWRef = useRef(DAY_W_DEFAULT)
  const setDayWAt = (w: number) => {
    const c = Math.min(DAY_W_MAX, Math.max(DAY_W_MIN, Math.round(w * 10) / 10))
    dayWRef.current = c
    setDayW(c)
  }
  const zoomStep = (dir: 1 | -1) => setDayWAt(dayWRef.current * (dir > 0 ? ZOOM_FACTOR : 1 / ZOOM_FACTOR))
  /** 窗口内全览：整计划时长压进画布可视宽（长计划不再无限横向拉长） */
  const fitToWindow = () => {
    const pane = canvasPaneRef.current
    const avail = pane?.clientWidth ?? 0
    if (avail <= 0) return
    setDayWAt(Math.floor(((avail - 2) / days) * 10) / 10) // floor：全览宁窄勿溢出
    if (pane) pane.scrollLeft = 0
  }
  /** Ctrl+滚轮缩放（v4.159，MS Project 口径；普通滚轮保持滚动行）：以光标下
   *  日期为锚——缩放提交（DOM 宽更新）后在 layout effect 回写 scrollLeft，
   *  同步回写会被旧内容宽钳制。 */
  const ganttAnchorRef = useRef<{ px: number; dayG: number } | null>(null)
  React.useLayoutEffect(() => {
    const pane = canvasPaneRef.current
    const a = ganttAnchorRef.current
    if (!pane || !a) return
    pane.scrollLeft = a.dayG * dayW - a.px
    ganttAnchorRef.current = null
  }, [dayW])
  React.useEffect(() => {
    const pane = canvasPaneRef.current
    if (!pane) return
    // v4.371：rAF 合并——高分辨率触控板每秒 60~120 次 wheel 事件各触发一次
    // setDayW 全三窗格重渲染；合并为每帧最多一次（帧内累积因子，锚点以累积
    // 后的目标日宽换算，终态与逐事件处理一致）。
    let wheelRaf = 0
    let pendingW = 0
    const onWheel = (e: WheelEvent) => {
      if (!e.ctrlKey && !e.metaKey) return // 普通滚轮=滚动行（原生）
      e.preventDefault()
      pendingW = (pendingW || dayWRef.current) * (e.deltaY < 0 ? ZOOM_FACTOR : 1 / ZOOM_FACTOR)
      const rect = pane.getBoundingClientRect()
      const px = e.clientX - rect.left
      ganttAnchorRef.current = { px, dayG: (pane.scrollLeft + px) / pendingW }
      if (wheelRaf) return
      wheelRaf = requestAnimationFrame(() => {
        wheelRaf = 0
        if (pendingW) { setDayWAt(pendingW); pendingW = 0 }
      })
    }
    pane.addEventListener('wheel', onWheel, { passive: false })
    return () => {
      if (wheelRaf) cancelAnimationFrame(wheelRaf)
      pane.removeEventListener('wheel', onWheel)
    }
  }, [])
  const [showFront, setShowFront] = useState(false)
  /** 行筛选（刀C 余项）：全部/关键/手动 + 名称文本；仅作用于横道（网络图=逻辑图不筛选） */
  const [filter, setFilter] = useState<GanttFilter>(GANTT_FILTER_DEFAULT)
  /** 排序/分组（v4.136 小刀，持久化 chatPrefs）；折叠组头 key 集为会话态 */
  const [sort, setSort] = useState<GanttSort>(() => loadChatPrefs().ganttSort as GanttSort)
  const [groupBy, setGroupBy] = useState<GanttGroupField[]>(() => loadChatPrefs().ganttGroup as GanttGroupField[])
  const [collapsedKeys, setCollapsedKeys] = useState<Set<string>>(() => new Set())
  /** 路径分析（v4.136 小刀）：沿选中任务的驱动边高亮前驱/后继链 */
  const [pathMode, setPathMode] = useState<'off' | 'pred' | 'succ'>('off')
  /** 基线条显示（有基线才出现开关；默认开，v4.116 刀7） */
  const [showBase, setShowBase] = useState(true)
  const [checkDate, setCheckDate] = useState<string>(new Date().toISOString().slice(0, 10))
  /** 拖拽态（刀9）：move=拖移（auto 转手动锁定）、resize=右缘改工期 */
  const [drag, setDrag] = useState<GanttDragState | null>(null)

  // ── 刀C 双栏：表格窗格宽度/列显隐（持久化 chatPrefs）────────────
  const [tableW, setTableW] = useState<number>(() => loadChatPrefs().ganttTableW)
  const [hide, setHide] = useState<string[]>(() => loadChatPrefs().ganttHide)
  /** 自定义列改名态（v4.138 #14）：renameKey=改名中的列（null=无）；Esc 取消标记走 ref（先于 blur） */
  const [renameKey, setRenameKey] = useState<string | null>(null)
  const [renameVal, setRenameVal] = useState('')
  const renameCancel = useRef(false)
  const splitRef = useRef<{ startX: number; startW: number; w: number } | null>(null)
  const tbodyRef = useRef<HTMLDivElement | null>(null)
  const cols = useMemo(() => visibleCols(hide, project.customLabels), [hide, project.customLabels])
  const leftW = useMemo(() => visibleLeftW(hide), [hide])
  const effW = Math.min(tableW, leftW)
  const colOn = (key: string) => !hide.includes(key)

  /** 分隔条拖动：右移加宽表格、左移收纳（最小=行号+名称）；拖完持久化 */
  const beginSplit = (e: React.MouseEvent) => {
    e.preventDefault()
    splitRef.current = { startX: e.clientX, startW: tableW, w: tableW }
    const onMove = (ev: MouseEvent) => {
      const s = splitRef.current!
      s.w = clampTableW(s.startW + (ev.clientX - s.startX), leftW)
      setTableW(s.w)
    }
    const onUp = () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      const s = splitRef.current
      splitRef.current = null
      if (s) saveChatPrefs({ ganttTableW: s.w }) // 提交在渲染期外（updater 禁副作用）
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }
  const resetSplit = () => {
    const w = clampTableW(GANTT_LEFT_W_FULL, leftW)
    setTableW(w)
    saveChatPrefs({ ganttTableW: w })
  }
  /** 单列显隐（行号/名称固定不藏） */
  const toggleCol = (key: string, on: boolean) => {
    const next = on ? hide.filter((k) => k !== key) : [...hide, key]
    setHide(next)
    saveChatPrefs({ ganttHide: next })
  }
  /** 提交自定义列改名（v4.138 #14）：回车/失焦提交 setCustomLabel、Esc 置取消标记不落库；
   *  空串由 store 口径恢复缺省名；提交后关闭行内 Input */
  const commitRename = () => {
    if (!renameKey) return
    if (!renameCancel.current) setCustomLabel(renameKey, renameVal)
    renameCancel.current = false
    setRenameKey(null)
  }
  /** 画布纵向滚动 → 表格体同步（直改 DOM，滚动帧不走 setState） */
  const onCanvasScroll = (e: React.UIEvent<HTMLDivElement>) => {
    if (tbodyRef.current) tbodyRef.current.style.transform = `translateY(${-e.currentTarget.scrollTop}px)`
  }

  const cal = project.calendar
  const wbs = useMemo(() => wbsOf(project.tasks), [project.tasks])
  const startMs = useMemo(() => new Date(`${project.startDate}T00:00:00Z`).getTime(), [project.startDate])
  const visibleIdx = useMemo(() => filterGanttRows(project.tasks, filter, cpm), [project.tasks, filter, cpm])
  /** 可见行序列（v4.136 唯一状态源）：筛选→排序→多级分组→折叠全部合成进 rows；
   *  缺省（不排序不分组不折叠）=恒等原顺序，原分组行合成 kind=group（key=`wbs:原索引`） */
  const rows = useMemo(
    () => buildGanttRows(project.tasks, cpm, {
      sort: sort.field === 'none' ? undefined : sort,
      groupBy,
      collapsed: collapsedKeys.size > 0 ? collapsedKeys : undefined,
      visibleIdx,
    }),
    [project.tasks, cpm, sort, groupBy, collapsedKeys, visibleIdx],
  )
  /** 渲染行序 ↔ 任务 id：条形/依赖线 y 坐标按渲染行序算 */
  const visPos = useMemo(() => {
    const m = new Map<string, number>()
    rows.forEach((r, row) => { if (r.kind === 'task') m.set(project.tasks[r.idx].id, row) })
    return m
  }, [rows, project.tasks])
  /** 驱动链（路径分析）：锚点=选中任务；路径关/未选中/循环依赖=null 不参与渲染 */
  const chain = useMemo(() => {
    if (pathMode === 'off' || !selectedId || !cpm.ok) return null
    return drivingChain(project.tasks, project.links, cpm, selectedId, pathMode)
  }, [pathMode, selectedId, cpm, project.tasks, project.links])
  const chainTasks = useMemo(() => (chain ? new Set(chain.taskIds) : null), [chain])
  const chainLinks = useMemo(() => (chain ? new Set(chain.links.map(linkKey)) : null), [chain])
  /** 路径开启时非链行淡化（锚点行与链上行保持） */
  const rowDim = (id: string): boolean => !!chainTasks && id !== selectedId && !chainTasks.has(id)
  /** 组头折叠/展开（▲=展开 ▶=折叠）；折叠集变化不影响 chatPrefs（会话态） */
  const toggleCollapse = (key: string) => {
    setCollapsedKeys((s) => {
      const n = new Set(s)
      if (n.has(key)) n.delete(key)
      else n.add(key)
      return n
    })
  }
  const rowMenu = (t: SchedTask, group: boolean) => [
    { key: 'add', label: '在下方添加任务' },
    { key: 'group', label: '添加分组' },
    ...(group ? [] : [
      { key: 'milestone', label: t.isMilestone ? '取消里程碑' : '设为里程碑' },
      { key: 'inspector', label: '任务检查器' },
    ]),
    { type: 'divider' as const },
    { key: 'del', label: '删除', danger: true },
  ]
  const rowMenuClick = (key: string, t: SchedTask) => {
    if (key === 'add') addTask(t.id)
    else if (key === 'group') addGroup()
    else if (key === 'milestone') updateTask(t.id, { isMilestone: !t.isMilestone })
    else if (key === 'inspector') onInspect?.(t.id)
    else if (key === 'del') removeTask(t.id)
  }
  /** 工作日序号 → 自然日列偏移（条形 x 坐标；跨周末自然变宽，斑马口径） */
  const dayNo = useMemo(() => (wd: number) => {
    return Math.round((wdToDate(project.startDate, wd, cal).getTime() - startMs) / 86400000)
  }, [project.startDate, cal, startMs])

  const days = Math.max(dayNo(Math.max(cpm.duration, 7)) + 3, 21, deadlineOff(project.deadline, startMs) + 2)
  const chartW = days * dayW
  const totalH = rows.length * ROW_H

  /** 自然日列（斑马口径：每天一列，非工作日底纹） */
  const colDates = useMemo(
    () => Array.from({ length: days }, (_, i) => {
      const d = new Date(startMs + i * 86400000)
      return { d, iso: d.toISOString().slice(0, 10), offWork: !isWorkingDate(d, normalizeCalendar(cal)) }
    }),
    [startMs, cal, days],
  )
  /** 月份跨列段（双层时标上行：连续同月合成一段，MS Project 口径） */
  const monthRuns = useMemo(() => {
    const runs: { y: number; m: number; count: number }[] = []
    for (const c of colDates) {
      const y = c.d.getUTCFullYear()
      const m = c.d.getUTCMonth()
      const last = runs[runs.length - 1]
      if (last && last.y === y && last.m === m) last.count++
      else runs.push({ y, m, count: 1 })
    }
    return runs
  }, [colDates])
  /** 周刻度层（v4.141：dayW<12 时底层刻度由日切周）——连续同自然周（周一始）
   *  合成一段，跨月不断开；label=周起始日，title=工程周序号+起止日期 */
  const useWeekTier = dayW < WEEK_TIER_BELOW
  const weekRuns = useMemo(() => {
    const runs: { startIdx: number; count: number; label: string; title: string }[] = []
    colDates.forEach((c, i) => {
      const last = runs[runs.length - 1]
      if (last && c.d.getUTCDay() !== 1) {
        last.count++
        return
      }
      runs.push({ startIdx: i, count: 1, label: '', title: '' })
    })
    for (const r of runs) {
      const start = colDates[r.startIdx].d
      const end = colDates[Math.min(r.startIdx + r.count - 1, colDates.length - 1)].d
      const w = r.count * dayW
      r.label = w >= 44 ? `${start.getUTCMonth() + 1}/${start.getUTCDate()}` : w >= 18 ? `${start.getUTCDate()}` : ''
      r.title = `第 ${Math.floor(r.startIdx / 7) + 1} 周（${start.getUTCMonth() + 1}/${start.getUTCDate()} ~ ${end.getUTCMonth() + 1}/${end.getUTCDate()}）`
    }
    return runs
  }, [colDates, dayW])
  const todayCol = useMemo(() => {
    const off = Math.floor((Date.now() - startMs) / 86400000)
    return off >= 0 && off <= days ? off : null
  }, [startMs, days])

  /** 目标竣工线（自然日列位；超出画布为 null 不画） */
  const deadlineCol = useMemo(() => {
    const off = deadlineOff(project.deadline, startMs)
    return off >= 0 && off <= days ? off : null
  }, [project.deadline, startMs, days])

  /** 工作日→自然日偏移反查表（拖拽落点吸附用） */
  const wdOffsets = useMemo(
    () => workdayOffsets(project.startDate, Math.max(cpm.duration + 30, 45), project.calendar),
    [project.startDate, project.calendar, cpm.duration],
  )

  // 资源成本（刀3）：computeCosts 每次 render 重算（与 CPM 同范式，工期变→成本自动变）。
  // 无任何资源/成本数据时成本列留空（诚实呈现，不留 ¥0 假象）；CPM 未过成本不出（fail-closed）。
  const costs = useMemo(() => computeCosts(project, cpm), [project, cpm])
  const costShown = hasCostData(project) && costs.ok

  /** 开始拖拽：拖移（转手动锁定）或右缘缩放（改工期）。循环依赖/分组行/里程碑缩放禁用。 */
  const beginDrag = (e: React.MouseEvent, t: SchedTask, kind: 'move' | 'resize') => {
    if (!cpm.ok || t.level === 0) return
    if (kind === 'resize' && t.isMilestone) return
    const row = cpm.rows[t.id]
    if (!row) return
    e.preventDefault()
    e.stopPropagation()
    const startX = e.clientX
    const origEs = row.es
    // 缩放右缘起点=条形实际右缘 ef（wd 任务=es+dur 与旧式逐位一致；cd 任务
    // dur 是自然日数不能当工作日序号加）；cd 的 preview/回写=自然日数（刀3）。
    const origEf = row.ef
    const origDur = t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
    const cdTask = isCd(t)
    const st: GanttDragState = {
      id: t.id, kind, origEs, origDur,
      preview: kind === 'move' ? origEs : origDur,
      moved: false,
    }
    const onMove = (ev: MouseEvent) => {
      const dx = ev.clientX - startX
      if (dx !== 0) st.moved = true
      st.preview = kind === 'move'
        ? dropToWd(wdOffsets, dayNo(origEs), dx, dayW)
        : cdTask
          ? resizeToNaturalDuration(wdOffsets, origEs, dayNo(origEf), dx, dayW)
          : resizeToDuration(wdOffsets, origEs, dayNo(origEf), dx, dayW)
      setDrag({ ...st })
    }
    const onUp = () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      window.removeEventListener('keydown', onKey)
      if (st.moved) {
        if (kind === 'move') {
          if (t.mode === 'manual') updateTask(t.id, { manualStart: st.preview })
          else updateTask(t.id, { mode: 'manual', manualStart: st.preview })
        } else {
          updateTask(t.id, { duration: st.preview })
        }
      }
      setDrag(null)
    }
    const onKey = (ev: KeyboardEvent) => {
      if (ev.key !== 'Escape') return
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      window.removeEventListener('keydown', onKey)
      setDrag(null) // 未 moved 提交逻辑不触发=取消
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    window.addEventListener('keydown', onKey)
  }

  /** 依赖线（刀B）：按四种搭接类型取各自锚点画正交折线；两端均可见才画，
   *  y 按可见行序（筛选后行位变化，全表序会错位——刀C 余项口径） */
  const linkPaths = project.links.map((l) => {
    const fy = visPos.get(l.from)
    const ty = visPos.get(l.to)
    const f = cpm.rows[l.from]
    const t2 = cpm.rows[l.to]
    if (fy === undefined || ty === undefined || !f || !t2) return null
    return {
      d: ganttLinkPath(l.type, {
        x1s: dayNo(f.es) * dayW,
        x1f: dayNo(f.ef) * dayW,
        y1: fy * ROW_H + ROW_H / 2,
        x2s: dayNo(t2.es) * dayW,
        x2f: dayNo(t2.ef) * dayW,
        y2: ty * ROW_H + ROW_H / 2,
      }),
      key: `${l.from}>${l.to}>${l.type}`,
      lk: linkKey(l),
    }
  }).filter(Boolean) as { d: string; key: string; lk: string }[]

  /** 前锋线折线点（行中心；x=自然日列；仅可见行参与，y=可见行序） */
  const frontLine = useMemo(() => {
    if (!showFront || !cpm.ok) return null
    const off = Math.round((new Date(`${checkDate}T00:00:00Z`).getTime() - startMs) / 86400000)
    if (!Number.isFinite(off) || off < 0) return null
    const pts: { x: number; y: number }[] = []
    rows.forEach((r, row) => {
      if (r.kind !== 'task') return
      const t = project.tasks[r.idx]
      const w = frontierWd(t, cpm.rows[t.id], checkWdIdx(checkDate))
      if (w !== null) pts.push({ x: dayNo(w) * dayW, y: row * ROW_H + ROW_H / 2 })
    })
    return pts.length >= 2 ? { pts, checkX: Math.min(off, days) * dayW } : null
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [showFront, cpm, project.tasks, rows, checkDate, startMs, dayNo, dayW, days, colDates])

  /** 检查日期 → 工作日序号（前锋点按进度比例需要工作日刻度） */
  function checkWdIdx(dateIso: string): number {
    let idx = 0
    for (const c of colDates) {
      if (c.iso > dateIso) break
      if (!c.offWork) idx++
    }
    return Math.max(0, idx - 1)
  }

  // 分组行配色序号（斑马口径：橙/青/粉/黄/蓝/绿 六色循环，子行随组）
  const groupColorSeq: number[] = []
  let gc = 0
  for (let i = 0; i < project.tasks.length; i++) {
    if (project.tasks[i].level === 0) gc = (gc + 1) % 6
    groupColorSeq[i] = project.tasks[i].level === 0 ? gc : (i > 0 ? groupColorSeq[i - 1] : gc)
  }

  return (
    <div className="sched-gantt" data-testid="sched-gantt">
      <GanttToolbar
        project={project}
        showFront={showFront}
        onToggleFront={() => setShowFront((v) => !v)}
        showBase={showBase}
        onToggleBase={() => setShowBase((v) => !v)}
        checkDate={checkDate}
        setCheckDate={setCheckDate}
        filter={filter}
        setFilter={setFilter}
        sort={sort}
        setSort={setSort}
        groupBy={groupBy}
        setGroupBy={setGroupBy}
        setCollapsedKeys={setCollapsedKeys}
        pathMode={pathMode}
        setPathMode={setPathMode}
        colOn={colOn}
        onToggleCol={toggleCol}
        renameKey={renameKey}
        renameVal={renameVal}
        setRenameKey={setRenameKey}
        setRenameVal={setRenameVal}
        onCommitRename={commitRename}
        renameCancel={renameCancel}
        onZoom={zoomStep}
        onFit={fitToWindow}
      />
      <div className="sched-gantt-frame">
        {/* 表格窗格（左）：宽度=分隔条拖动（chatPrefs 持久化），列显隐收纳 */}
        <div className="sched-gantt-tablepane" data-testid="sched-gantt-table" style={{ width: effW }}>
          <GanttTable
            leftW={leftW}
            cols={cols}
            tbodyRef={tbodyRef}
            rows={rows}
            wbs={wbs}
            project={project}
            cpm={cpm}
            selectedId={selectedId}
            costShown={costShown}
            costs={costs}
            groupColorSeq={groupColorSeq}
            collapsedKeys={collapsedKeys}
            colOn={colOn}
            rowDim={rowDim}
            onSelect={select}
            onUpdateTask={updateTask}
            onToggleCollapse={toggleCollapse}
            rowMenu={rowMenu}
            onRowMenuClick={rowMenuClick}
          />
        </div>
        {/* 分隔条：拖动收纳/展开表格窗格，双击复位全列 */}
        <div
          className="sched-gantt-split"
          data-testid="sched-gantt-split"
          onMouseDown={beginSplit}
          onDoubleClick={resetSplit}
          title="拖动收纳/展开表格 · 双击复位"
        />
        {/* 画布窗格（右）：时间轴+条形+覆盖层；横滚独立，纵滚同步表格体 */}
        <div className="sched-gantt-canvaspane" data-testid="sched-gantt-canvas" ref={canvasPaneRef} onScroll={onCanvasScroll}>
          <div className="sched-gantt-canvas-inner" style={{ width: chartW }}>
            <div className="sched-gantt-row sched-gantt-head" style={{ height: 40, width: chartW }}>
              {/* 双层时标（MS Project 口径，自左向右）：上行=月份跨列段；
                  下行随缩放切换刻度——dayW≥12 逐日号，<12 自然周（周一起始） */}
              <GanttTimeline
                chartW={chartW}
                monthRuns={monthRuns}
                useWeekTier={useWeekTier}
                weekRuns={weekRuns}
                dayW={dayW}
                colDates={colDates}
              />
            </div>
            <GanttBars
              rows={rows}
              project={project}
              cpm={cpm}
              chartW={chartW}
              dayW={dayW}
              dayNo={dayNo}
              selectedId={selectedId}
              showBase={showBase}
              drag={drag}
              groupColorSeq={groupColorSeq}
              rowDim={rowDim}
              onSelect={select}
              onToggleCollapse={toggleCollapse}
              rowMenu={rowMenu}
              onRowMenuClick={rowMenuClick}
              beginDrag={beginDrag}
            />
            {rows.length === 0 && project.tasks.length > 0 && (
              <div className="sched-empty" data-testid="sched-gantt-filter-empty">没有符合筛选条件的任务（切回「全部」或清空搜索）</div>
            )}
            {/* 覆盖层：非工作日底纹 + 今日线 + 前锋线 + 搭接箭线 */}
            <GanttOverlay
              chartW={chartW}
              colDates={colDates}
              dayW={dayW}
              totalH={totalH}
              linkPaths={linkPaths}
              chainLinks={chainLinks}
              todayCol={todayCol}
              deadlineCol={deadlineCol}
              deadline={project.deadline}
              frontLine={frontLine}
            />
          </div>
        </div>
      </div>
      {project.tasks.length === 0 && (
        <div className="sched-empty">暂无任务，点击上方「添加任务」或载入示例工程开始编制</div>
      )}
    </div>
  )
}