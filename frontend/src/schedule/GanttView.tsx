/**
 * schedule/GanttView.tsx — 横道图（甘特图）视图入口（v4.112.0 斑马 UI 对齐）
 *
 * 「瘦身 P3 结构版1：巨文件拆分」：渲染块迁入 schedule/gantt/ 子模块，本文件只留
 * 状态/派生计算/交互 handler 与组合层——导出面与行为零变化（classNames/testids 不变）。
 *
 * W1 起顶部带斑马 ViewDB 的 9 个内置视图；视图切换一次应用列集/表格宽/
 * 分组/排序/筛选，手动改任一显示参数即标为“自定义”。
 *
 * 子模块：
 *  - gantt/ganttUtil.ts      共享常量（ROW_H/W/缩放档位…）+ 纯函数（前置/后续/行号/
 *                            竣工偏移/分组成本/前锋点位/fmtDate），同 aoaRuler.ts 范式；
 *  - gantt/PredEditor.tsx    前置关系编辑 Popover；
 *  - gantt/CustomFieldCell.tsx 自定义字段单元格（text=Input / num=InputNumber）；
 *  - gantt/GanttToolbar.tsx  工具栏（前锋线/基线/筛选/排序/分组/路径/列菜单/缩放全览）；
 *  - gantt/GanttTable.tsx    表格窗格（列头 + 表体逐行，含行菜单/依赖引用/成本/自定义列）；
 *  - gantt/GanttTimeline.tsx 画布双标尺（上行工程标尺=年-月跨列段+日/周刻度线，
 *                             下行进度标尺=周/星期；两行常驻，高 = timescale.GANTT_RULER_H）；
 *  - gantt/GanttBars.tsx     画布行（基线/里程碑/条形/总时差尾/拖拽预览）；
 *  - gantt/GanttOverlay.tsx  画布覆盖层（非工作日底纹/搭接箭线/今日线/竣工线/前锋线）。
 *
 * 视觉/交互口径（v4.131 刀C/4.136/4.138/4.140/4.141，详见各子模块头注释）：
 * 自然日时间轴 + 周末/节假日底纹；表格窗格|画布窗格真双栏（分隔条/列显隐持久化）；
 * 行序列唯一状态源（筛选→排序→多级分组→折叠 合成进 rows）；连续缩放+全览；Ctrl+滚轮。
 */
import React, { useMemo, useRef, useState } from 'react'
import { Menu, type MenuProps } from 'antd'
import type { CpmResult, SchedProject, SchedTask, SchedTaskStyle } from './types'
import { isWorkingDate, normalizeCalendar, wdToDate } from './calendar'
import { dropToWd, resizeToDuration, resizeToNaturalDuration, workdayOffsets } from './drag'
import { computeCosts } from './cost'
import { isCd } from './cpm'
import { latestReview, reviewDate } from './reviews'
import { ganttLinkPath } from './ganttLinks'
import { useScheduleStore } from './store'
import { hasCostData } from './costUi'
import { GANTT_LEFT_W_FULL, GANTT_TABLE_W_MIN, clampTableW, visibleCols, visibleLeftW } from './ganttCols'
import { GANTT_FILTER_DEFAULT, filterGanttRows, type ColFilters, type GanttFilter } from './ganttFilter'
import { buildGanttRows, wbsOf, GANTT_GROUP_OPTIONS, MAX_GANTT_GROUP_DEPTH, type GanttGroupField, type GanttSort } from './ganttGroup'
import { applyCustomViewSnapshot } from './views'

import { drivingChain, linkKey } from './pathDriver'
import { loadChatPrefs, saveChatPrefs } from './chatPrefs'
import { colFilterSignature } from './colFilterPrefs'
import {
  CUSTOM_GANTT_VIEW_ID,
  builtinViewOf,
  hideForBuiltinView,
  widthForBuiltinView,
  widthForHide,
} from './viewPresets'
import { beginWindowDrag } from './gantt/useDragGesture'
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
import { GANTT_RULER_H, buildRulerModel, resolveLineDensity } from './timescale'
import { HIT_ATTR, type HitRegion, hitRegionChain, pickHitRegion } from './hitRegion'
import { HIT_TEXT_COLORS, menuForHit, type HitMenuContext, type HitMenuItem } from './hitMenu'
import './hit.css'

export const GanttView: React.FC<{
  project: SchedProject
  cpm: CpmResult
  onInspect?: (taskId: string) => void
  /** ZPert 复刻工作区隐藏附加工具条；独立测试/工具面仍默认显示。 */
  showToolbar?: boolean
}> = ({ project, cpm, onInspect, showToolbar = true }) => {
  const selectedId = useScheduleStore((s) => s.selectedId)
  const select = useScheduleStore((s) => s.select)
  const updateTask = useScheduleStore((s) => s.updateTask)
  const setCustomLabel = useScheduleStore((s) => s.setCustomLabel)
  const addTask = useScheduleStore((s) => s.addTask)
  const addGroup = useScheduleStore((s) => s.addGroup)
  const removeTask = useScheduleStore((s) => s.removeTask)
  const promoteTask = useScheduleStore((s) => s.promoteTask)
  const demoteTask = useScheduleStore((s) => s.demoteTask)
  const ungroupTask = useScheduleStore((s) => s.ungroupTask)
  const moveRow = useScheduleStore((s) => s.moveRow)
  const moveTask = useScheduleStore((s) => s.moveTask)
  const pasteRows = useScheduleStore((s) => s.pasteRows)
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
  /**
   * 前锋线默认跟随**最新一期检视**（斑马复刻 S2 接线）：有检视就默认打开并停在
   * 那一期的工作日；没有检视才回到"今天"。两者都是**初值**——用户仍可用工具栏
   * 的开关与日期框覆盖（视图态不入文件）。
   */
  const latestRev = useMemo(() => latestReview(project), [project])
  const [showFront, setShowFront] = useState(() => latestRev !== null)
  /** W1：本机视图偏好只读一次，后续视图切换/手动覆盖均走同一状态源。 */
  const [initialPrefs] = useState(() => loadChatPrefs())
  const [viewId, setViewId] = useState(initialPrefs.ganttViewId)
  /** 行筛选（W1）：全部/关键/手动/里程碑 + 名称文本；仅作用于横道。 */
  const [filter, setFilter] = useState<GanttFilter>(() => ({
    ...GANTT_FILTER_DEFAULT,
    kind: initialPrefs.ganttFilterKind,
  }))
  /**
   * 列级筛选（W5）：列键 → 条件（文本/数值/日期/枚举）。W5 第二刀接入本机偏好
   * （chatPrefs.ganttColFilters）：初值只读一次，变更按签名去抖后单点落盘；
   * 视图切换不动列筛（列筛与列集正交，斑马 ViewDB 的 column_filters 表达式未建模）。
   */
  const [colFilters, setColFilters] = useState<ColFilters>(() => initialPrefs.ganttColFilters)
  /** 排序/分组（v4.136 小刀，持久化 chatPrefs）；折叠组头 key 集为会话态 */
  const [sort, setSort] = useState<GanttSort>(initialPrefs.ganttSort as GanttSort)
  const [groupBy, setGroupBy] = useState<GanttGroupField[]>(initialPrefs.ganttGroup as GanttGroupField[])
  const [collapsedKeys, setCollapsedKeys] = useState<Set<string>>(() => new Set())
  /** 路径分析（v4.136 小刀）：沿选中任务的驱动边高亮前驱/后继链 */
  const [pathMode, setPathMode] = useState<'off' | 'pred' | 'succ'>('off')
  /** 基线条显示（有基线才出现开关；默认开，v4.116 刀7） */
  const [showBase, setShowBase] = useState(true)
  const [checkDate, setCheckDate] = useState<string>(() =>
    latestRev ? reviewDate(project, latestRev.at) : new Date().toISOString().slice(0, 10),
  )
  /** 拖拽态（刀9）：move=拖移（auto 转手动锁定）、resize=右缘改工期 */
  const [drag, setDrag] = useState<GanttDragState | null>(null)

  // ── 刀C 双栏：表格窗格宽度/列显隐（持久化 chatPrefs）────────────
  const [tableW, setTableW] = useState<number>(initialPrefs.ganttTableW)
  const [hide, setHide] = useState<string[]>(initialPrefs.ganttHide)
  /** 自定义列改名态（v4.138 #14）：renameKey=改名中的列（null=无）；Esc 取消标记走 ref（先于 blur） */
  const [renameKey, setRenameKey] = useState<string | null>(null)
  const [renameVal, setRenameVal] = useState('')
  const renameCancel = useRef(false)
  const splitRef = useRef<{ startX: number; startW: number; w: number } | null>(null)
  const tbodyRef = useRef<HTMLDivElement | null>(null)
  // ── 命中区 / 统一右键（ZPert8 W4「命中区顺序」+ W5「右键按命中类型追加」）──
  /** 视图根节点：`data-hit` 注解与**单一**右键入口都挂在这里（RE §16.2 统一命中） */
  const rootRef = useRef<HTMLDivElement | null>(null)
  /** 打开的右键菜单：命中区 + 屏幕坐标 + 条目（条目全部来自 hitMenu，视图不写死顺序） */
  /**
   * 格式刷（斑马右键「格式刷」）：先点一行取外观，再点目标行应用；应用后自动收起刷子。
   * 空样式对象是合法刷子（=把目标行外观清回默认），故用 null 表示「未激活」。
   */
  const [brush, setBrush] = useState<{ style?: SchedTaskStyle } | null>(null)
  const applyBrushTo = (id: string) => {
    if (!brush) return
    updateTask(id, { style: brush.style && Object.keys(brush.style).length > 0 ? brush.style : undefined })
    setBrush(null)
  }
  const [hitMenu, setHitMenu] = useState<{
    x: number
    y: number
    region: HitRegion
    items: HitMenuItem[]
  } | null>(null)
  /** 菜单收起后要把焦点还给谁（W6 键盘：右键命中的那一行；键盘用户不掉回 body） */
  const hitMenuFocusBack = useRef<HTMLElement | null>(null)
  /** 本视图剪贴板（复制出的 TSV 行）：粘贴启用条件据此判断，不假装读得到系统剪贴板 */
  const [hitClipboard, setHitClipboard] = useState('')
  /** 最近一次菜单的命中列键（列头「隐藏该列」用；列头上下文与选中行无关） */
  const hitColumnKey = useRef<string | null>(null)
  const cols = useMemo(() => visibleCols(hide, project.customLabels), [hide, project.customLabels])
  const leftW = useMemo(() => visibleLeftW(hide), [hide])
  const effW = Math.min(tableW, leftW)
  const colOn = (key: string) => !hide.includes(key)

  /** W1：视图切换=列集/宽度/分组/排序/筛选一次应用；空格筛选文本随视图清空。
   *  S6.2 视图库：内置预设之外，project.views 的自定义视图同路径应用
   *  （viewbackend-re §6：视图=列可见性预设+开关，同一引擎切列集，不建第二套执行器）。 */
  const applyView = (id: string) => {
    const preset = builtinViewOf(id)
    if (!preset) {
      const custom = (project.views ?? []).find((v) => v.id === id)
      if (!custom) return
      const snap = applyCustomViewSnapshot(custom)
      const nextWidth = widthForHide(snap.hide)
      const nextFilter: GanttFilter = { kind: snap.filterKind, text: '' }
      setViewId(custom.id)
      setHide(snap.hide)
      setTableW(nextWidth)
      setGroupBy([...snap.groupBy])
      setSort({ ...snap.sort })
      setFilter(nextFilter)
      setCollapsedKeys(new Set())
      // 自定义视图 id 不入 chatPrefs（parseViewId 会把未知 id 折回缺省）——
      // 持久化参数本身并标记为自定义，重载后参数还原、标签显示「自定义」。
      saveChatPrefs({
        ganttViewId: CUSTOM_GANTT_VIEW_ID,
        ganttHide: snap.hide,
        ganttTableW: nextWidth,
        ganttGroup: [...snap.groupBy],
        ganttSort: { ...snap.sort },
        ganttFilterKind: snap.filterKind,
      })
      return
    }
    const nextHide = hideForBuiltinView(preset)
    const nextWidth = widthForBuiltinView(preset)
    const nextGroup = [...preset.groupBy]
    const nextSort = { ...preset.sort }
    const nextFilter: GanttFilter = { kind: preset.filterKind, text: '' }
    setViewId(preset.id)
    setHide(nextHide)
    setTableW(nextWidth)
    setGroupBy(nextGroup)
    setSort(nextSort)
    setFilter(nextFilter)
    setCollapsedKeys(new Set())
    saveChatPrefs({
      ganttViewId: preset.id,
      ganttHide: nextHide,
      ganttTableW: nextWidth,
      ganttGroup: nextGroup,
      ganttSort: nextSort,
      ganttFilterKind: preset.filterKind,
    })
  }

  /** 任一显示参数被手动覆盖后，当前状态不再冒称仍等于内置视图。 */
  const markCustomView = () => setViewId(CUSTOM_GANTT_VIEW_ID)

  /** S6.2 视图库：ribbon「视图管理」应用请求（store 信号）→ 应用对应自定义视图。
   *  customViewRequest 初值 null，仅 seq 变化时应用；读 store 内最新 project.views
   *  免闭包陈旧。 */
  const customViewRequest = useScheduleStore((s) => s.customViewRequest)
  /** ribbon 命令信号消费（一次点亮批1）：find/collapse/expand/set-group 直达本组件视图态 */
  const ganttCommand = useScheduleStore((s) => s.ganttCommand)
  const cmdSeqRef = useRef(0)
  React.useEffect(() => {
    if (!ganttCommand || ganttCommand.seq === cmdSeqRef.current) return
    cmdSeqRef.current = ganttCommand.seq
    if (ganttCommand.kind === 'find') {
      setFilter((current) => ({ ...current, text: ganttCommand.payload ?? '' }))
    } else if (ganttCommand.kind === 'collapse-all') {
      setCollapsedKeys(new Set(rows.filter((r) => r.kind === 'group').map((r) => r.key)))
    } else if (ganttCommand.kind === 'expand-all') {
      setCollapsedKeys(new Set())
    } else if (ganttCommand.kind === 'collapse-selected') {
      // 折叠选中任务所在的组（含选中即组头）；无选中=全收
      if (!selectedId) {
        setCollapsedKeys(new Set(rows.filter((r) => r.kind === 'group').map((r) => r.key)))
        return
      }
      let ownerKey: string | null = null
      for (const r of rows) {
        if (r.kind === 'group') ownerKey = r.key
        else if (project.tasks[r.idx]?.id === selectedId) break
      }
      if (ownerKey) setCollapsedKeys(new Set([...collapsedKeys, ownerKey]))
    } else if (ganttCommand.kind === 'format-brush') {
      // 功能区「格式刷」= 取选中行外观；未选中时入空刷子（=清外观）并给提示由状态栏承担
      const cur = useScheduleStore.getState().project.tasks.find((t) => t.id === selectedId)
      setBrush({ style: cur?.style })
    } else if (ganttCommand.kind === 'column-filter-clear') {
      setColFilters({})
    } else if (ganttCommand.kind === 'set-group') {
      try {
        const next = JSON.parse(ganttCommand.payload ?? '[]') as GanttGroupField[]
        changeGroup(next.filter((g) => GANTT_GROUP_OPTIONS.some((o) => o.value === g)).slice(0, MAX_GANTT_GROUP_DEPTH))
      } catch { /* 坏 payload 忽略（信号通道容错） */ }
    }
  }, [ganttCommand]) // eslint-disable-line react-hooks/exhaustive-deps -- 按 seq 信号触发，闭包内状态均为最新渲染值
  const applyRequestSeq = useRef(0)
  React.useEffect(() => {
    if (!customViewRequest || customViewRequest.seq === applyRequestSeq.current) return
    applyRequestSeq.current = customViewRequest.seq
    const custom = useScheduleStore.getState().project.views?.find((v) => v.id === customViewRequest.id)
    if (custom) applyView(custom.id)
  }, [customViewRequest]) // eslint-disable-line react-hooks/exhaustive-deps -- applyView 每渲染重建，按 seq 信号触发

  const changeFilterKind = (kind: GanttFilter['kind']) => {
    setFilter((current) => ({ ...current, kind }))
    markCustomView()
    saveChatPrefs({ ganttViewId: CUSTOM_GANTT_VIEW_ID, ganttFilterKind: kind })
  }

  const changeSort = (next: GanttSort) => {
    setSort(next)
    markCustomView()
    saveChatPrefs({ ganttViewId: CUSTOM_GANTT_VIEW_ID, ganttSort: next })
  }

  const changeGroup = (next: GanttGroupField[]) => {
    setGroupBy(next)
    markCustomView()
    saveChatPrefs({ ganttViewId: CUSTOM_GANTT_VIEW_ID, ganttGroup: next })
  }

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
    markCustomView()
    saveChatPrefs({ ganttViewId: CUSTOM_GANTT_VIEW_ID, ganttHide: next })
  }
  /**
   * 列筛落盘（W5 第二刀）：签名去抖——首帧签名即当前值，不写盘（避免开机把
   * 偏好改回空），只有用户真正改了条件才写一次；写入口唯一，连点不重复落盘。
   */
  const colSig = colFilterSignature(colFilters)
  const colSigRef = useRef(colSig)
  React.useEffect(() => {
    if (colSig === colSigRef.current) return
    colSigRef.current = colSig
    saveChatPrefs({ ganttColFilters: colFilters })
  }, [colSig, colFilters])
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
  const visibleIdx = useMemo(
    () => filterGanttRows(project.tasks, { ...filter, colFilters }, cpm, {
      startDate: project.startDate,
      calendar: project.calendar,
    }),
    [project.tasks, project.startDate, project.calendar, filter, colFilters, cpm],
  )
  /** 可见行序列（v4.136 唯一状态源）：筛选→排序→多级分组→折叠全部合成进 rows；
   *  缺省（不排序不分组不折叠）=恒等原顺序，原分组行合成 kind=group（key=`wbs:原索引`） */
  const rows = useMemo(
    () => buildGanttRows(project.tasks, cpm, {
      sort: sort.field === 'none' ? undefined : sort,
      groupBy,
      collapsed: collapsedKeys.size > 0 ? collapsedKeys : undefined,
      visibleIdx,
      project,
    }),
    [project, cpm, sort, groupBy, collapsedKeys, visibleIdx],
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
  // ── 命中区注解：`data-hit` 是唯一判据（绝不读 DOM 文本）───────────────
  /**
   * 表格列头 / 单元格 / 行号列的 `data-hit` 在这一层统一挂属性。
   * 为什么是效应而不是 JSX 属性：`GanttTable`/`GanttBars`/`GanttRowShell` 是只读子模块
   * （本轮不许改），视图侧挂属性等价于把「命中区表」外置成一份可断言的事实。
   *
   * **为什么行容器声明的是 `row-header` 而不是 `cell`**：`pickHitRegion` 是优先级裁决
   * （列头 → 画布 → 单元格 → 状态栏 → 分隔条 → 行头），单元格比行头优先 —— 行容器若
   * 声明成 `cell`，行号格的 `row-header` 反而会被它裁成 `cell`。反过来让行容器声明
   * `row-header`（本表里优先级最低=兜底区），单元格的 `cell` 自然盖住它：点到格子是
   * 单元格，点到行内边距是行头，两者出的是同一份结构菜单（行为一致，语义不乱）。
   * 依赖只取结构维度（行序列 / 列集）：选中、缩放、拖拽的重渲染复用同一批 DOM 节点，
   * 不会每次 mousemove 都全表重扫。
   */
  React.useLayoutEffect(() => {
    const root = rootRef.current
    if (!root) return
    const mark = (el: Element, region: HitRegion) => {
      if (el.getAttribute(HIT_ATTR) !== region) el.setAttribute(HIT_ATTR, region)
    }
    root.querySelectorAll('.sched-gantt-tablepane .sched-gantt-head').forEach((el) => mark(el, 'column-header'))
    root.querySelectorAll('.sched-gantt-tbody .sched-gantt-row').forEach((el) => mark(el, 'row-header'))
    root.querySelectorAll('.sched-gantt-tbody .sched-gantt-cell').forEach((el) => {
      mark(el, el.classList.contains('sched-row-no') ? 'row-header' : 'cell')
    })
  }, [rows, hide, project.customLabels])

  /** 当前选中任务（右键先选中命中行，菜单拿的就是“刚被命中的那条”上下文） */
  const selectedTask = (): SchedTask | null => {
    const id = useScheduleStore.getState().selectedId
    return id ? project.tasks.find((t) => t.id === id) ?? null : null
  }
  /** 任务序号 → 所在分组行的 key（分组行自身即“该组”；无分组上下文=null） */
  const groupKeyOf = (taskIdx: number): string | null => {
    // 不分组时：分组行 key = `wbs:原索引`（该行本身就是组头）
    if (rows.some((r) => r.kind === 'group' && r.key === `wbs:${taskIdx}`)) return `wbs:${taskIdx}`
    // 分组中：向前找最近的组头
    let key: string | null = null
    for (const r of rows) {
      if (r.kind === 'group') key = r.key
      else if (r.idx === taskIdx) break
    }
    return key
  }
  const setGroupCollapsed = (key: string, collapsed: boolean) => {
    setCollapsedKeys((s) => {
      const n = new Set(s)
      if (collapsed) n.add(key)
      else n.delete(key)
      return n
    })
  }
  /** 命中区 + 当前上下文 → hitMenu 的入参（视图不自己判断“能不能点”，只喂事实） */
  const hitContext = (region: HitRegion, column: { key: string; label: string } | null): HitMenuContext => {
    const t = selectedTask()
    const idx = t ? project.tasks.findIndex((x) => x.id === t.id) : -1
    const groupKey = idx >= 0 ? groupKeyOf(idx) : null
    return {
      hasSelection: !!t,
      selectedIsGroup: !!t && t.level === 0,
      selectedIndex: idx,
      taskCount: project.tasks.length,
      selectedIsMilestone: t?.isMilestone ?? false,
      hasClipboard: hitClipboard.length > 0,
      groupBeforeSelected: idx > 0 && project.tasks.slice(0, idx).some((x) => x.level === 0),
      hasGroupContext: region !== 'column-header' && groupKey !== null,
      selectedGroupCollapsed: groupKey !== null && collapsedKeys.has(groupKey),
      column,
    }
  }
  /** 列头命中列：表头第 i 个格 ↔ 可见列第 i 项（GanttTable 表头是 `cols` 的 1:1 渲染） */
  const columnAt = (target: Element | null): { key: string; label: string } | null => {
    const cell = target?.closest('.sched-gantt-cell')
    const head = cell?.closest('.sched-gantt-head')
    if (!cell || !head) return null
    const idx = Array.from(head.children).indexOf(cell)
    const col = cols[idx]
    return col ? { key: col.key, label: col.label } : null
  }
  /** 复制选中行为 TSV（与功能区「复制」同口径：名称/工期/开始/完成/前置） */
  const copySelectedRow = (): string => {
    const t = selectedTask()
    if (!t) return ''
    const row = cpm.rows[t.id]
    const preds = project.links
      .filter((l) => l.to === t.id)
      .map((l) => `${l.from}${l.type === 'FS' ? '' : l.type}${l.lag ? (l.lag > 0 ? '+' : '') + l.lag : ''}`)
      .join(',')
    return [
      ['名称', '工期(天)', '开始', '完成', '前置'].join('\t'),
      [t.name, String(t.isMilestone ? 0 : t.duration), row ? String(row.es) : '', row ? String(row.ef) : '', preds].join('\t'),
    ].join('\n')
  }
  /** HitMenuItem → antd Menu items：禁用原因走 label 的 title（点不动时也能问出为什么） */
  const toMenuItems = (items: HitMenuItem[]): MenuProps['items'] => {
    const out: NonNullable<MenuProps['items']> = []
    let lastGroup: string | null = null
    for (const item of items) {
      if (lastGroup !== null && item.group !== lastGroup) out.push({ type: 'divider', key: `div-${item.key}` })
      lastGroup = item.group
      out.push({
        key: item.key,
        disabled: !item.enabled,
        danger: item.danger,
        label: (
          <span className="sched-hit-label" data-hit-key={item.key} title={item.disabledReason}>
            {item.label}
          </span>
        ),
        // 子菜单（文字颜色色板）：父项只作分组标题，禁用态与父项一致
        ...(item.children
          ? {
              children: item.children.map((c) => ({
                key: c.key,
                label: <span className="sched-hit-label" data-hit-key={c.key}>{c.label}</span>,
              })),
            }
          : {}),
      })
    }
    return out
  }
  /** 去掉某样式键（文字颜色「恢复默认」用）：空对象返回 undefined，避免写空壳 */
function withoutStyleKey(style: SchedTaskStyle | undefined, key: keyof SchedTaskStyle): SchedTaskStyle | undefined {
  if (!style) return undefined
  const next = { ...style }
  delete next[key]
  return Object.keys(next).length > 0 ? next : undefined
}

/** 菜单动作分发：键空间与 commandRegistry 对齐；禁用项不会进来（antd 不派发 onClick） */
  const runHitAction = (key: string) => {
    const t = selectedTask()
    const idx = t ? project.tasks.findIndex((x) => x.id === t.id) : -1
    switch (key) {
      case 'add-work':
      case 'canvas.add-work':
        addTask(t?.id)
        break
      case 'canvas.insert-pre':
        if (idx > 0) addTask(project.tasks[idx - 1].id)
        break
      case 'canvas.insert-post':
        if (t) addTask(t.id)
        break
      case 'promote':
        if (t) promoteTask(t.id)
        break
      case 'demote':
        if (t) demoteTask(t.id)
        break
      case 'up':
        if (t) moveTask(t.id, -1)
        break
      case 'down':
        if (t) moveTask(t.id, 1)
        break
      case 'group':
        addGroup()
        break
      case 'ungroup':
        if (t) ungroupTask(t.id)
        break
      case 'brush':
        // 格式刷：取当前行外观为刷子，下一次点击行应用（斑马「复制外观→点目标」两步）
        if (t) setBrush({ style: t.style })
        break
      case 'text-color.default':
        if (t) updateTask(t.id, { style: withoutStyleKey(t.style, 'color') })
        break
      case 'text-color.red':
      case 'text-color.orange':
      case 'text-color.green':
      case 'text-color.blue':
      case 'text-color.purple': {
        if (!t) break
        const picked = HIT_TEXT_COLORS.find((c) => c.key === key)
        if (picked) updateTask(t.id, { style: { ...(t.style ?? {}), color: picked.color } })
        break
      }
      case 'column.autofit':
        // 列宽自适应：表格窗格回落到「可见列合计」（与分隔条双击同一落点）
        setTableW(leftW)
        saveChatPrefs({ ganttTableW: leftW })
        break
      case 'copy': {
        const tsv = copySelectedRow()
        if (!tsv) break
        setHitClipboard(tsv)
        void navigator.clipboard?.writeText(tsv).catch(() => undefined)
        break
      }
      case 'paste':
        if (hitClipboard) pasteRows(hitClipboard)
        else void navigator.clipboard?.readText().then((text) => pasteRows(text)).catch(() => undefined)
        break
      case 'delete':
        if (t) removeTask(t.id)
        break
      case 'milestone':
        if (t) updateTask(t.id, { isMilestone: !t.isMilestone })
        break
      case 'inspector':
        if (t) onInspect?.(t.id)
        break
      case 'canvas.collapse-group':
      case 'canvas.expand-group': {
        const groupKey = idx >= 0 ? groupKeyOf(idx) : null
        if (groupKey) setGroupCollapsed(groupKey, key === 'canvas.collapse-group')
        break
      }
      case 'canvas.fit':
        fitToWindow()
        break
      case 'column.hide':
        if (hitColumnKey.current) toggleCol(hitColumnKey.current, false)
        break
      case 'splitter.reset':
        resetSplit()
        break
      case 'splitter.collapse': {
        const w = clampTableW(GANTT_TABLE_W_MIN, leftW)
        setTableW(w)
        saveChatPrefs({ ganttTableW: w })
        break
      }
      default:
        // 未接入的条目（解组/格式刷/文字颜色/插入列/列设置/分区/样式/状态栏）保持无动作
        break
    }
  }
  /**
   * 行菜单（**内部实现细节**）：表格/画布行的 antd Dropdown 契约还挂在子模块上，
   * 条目同样来自 hitMenu（不再各写一份），动作与根入口共用 runHitAction。
   * 真正的右键由根节点的唯一入口在捕获阶段接管（`onHitContextMenu`）。
   */
  const rowMenu = (t: SchedTask, group: boolean) =>
    toMenuItems(
      menuForHit('cell', {
        hasSelection: true,
        selectedIsGroup: group,
        selectedIndex: project.tasks.findIndex((x) => x.id === t.id),
        taskCount: project.tasks.length,
        selectedIsMilestone: t.isMilestone,
        hasClipboard: hitClipboard.length > 0,
        groupBeforeSelected: project.tasks
          .slice(0, Math.max(0, project.tasks.findIndex((x) => x.id === t.id)))
          .some((x) => x.level === 0),
        hasGroupContext: false,
        selectedGroupCollapsed: false,
        column: null,
      }),
    )
  const rowMenuClick = (key: string, _t: SchedTask) => runHitAction(key)
  /**
   * **唯一右键入口**（RE §16.2 `FUN_1403c16d0` 统一命中 → 两条分支）：
   * `data-hit`（自限深度上溯）→ `pickHitRegion` 优先级裁决 → `menuForHit` 出条目。
   * 捕获阶段拦下：子组件的 antd Dropdown 不再各弹一份，菜单来源只有一个。
   * 无命中（真空白处）→ 不弹菜单（空白菜单里没有任何对象动作可给）。
   */
  const onHitContextMenu = (e: React.MouseEvent<HTMLDivElement>) => {
    const target = e.target as Element | null
    const region = pickHitRegion(hitRegionChain(target))
    if (!region) {
      setHitMenu(null)
      return
    }
    e.preventDefault()
    e.stopPropagation()
    // 命中对象 → 先选中该行（ZPert 对象分支 = 选中 + 对象菜单；行内的输入/按钮不代点）
    hitMenuFocusBack.current = null
    if (region === 'cell' || region === 'row-header' || region === 'canvas') {
      const rowEl = target?.closest('.sched-gantt-row')
      if (rowEl && !rowEl.classList.contains('sched-gantt-head') && !target?.closest('input,button,select,textarea')) {
        ;(rowEl as HTMLElement).click()
        // 菜单的「触发者」= 被右键的那一行：收起后焦点回它（表格行已 roving 可聚焦）
        hitMenuFocusBack.current = rowEl as HTMLElement
      }
    } else if (region === 'splitter') {
      hitMenuFocusBack.current = target?.closest('[data-hit="splitter"]') as HTMLElement | null
    }
    const column = region === 'column-header' ? columnAt(target) : null
    hitColumnKey.current = column?.key ?? null
    const items = menuForHit(region, hitContext(region, column))
    if (items.length === 0) {
      setHitMenu(null)
      return
    }
    setHitMenu({ x: e.clientX, y: e.clientY, region, items })
  }
  /** 关菜单：点外面 / Esc / 滚轮滚动都收起来（避免固定在屏幕上的浮层跟着跑掉内容）
   *  W6：Esc 收起后**把焦点还给触发者**——键盘用户按完菜单不会掉回文档根 */
  React.useEffect(() => {
    if (!hitMenu) return
    const close = () => setHitMenu(null)
    const onKey = (ev: KeyboardEvent) => {
      if (ev.key !== 'Escape') return
      setHitMenu(null)
      hitMenuFocusBack.current?.focus?.()
    }
    window.addEventListener('mousedown', close)
    window.addEventListener('keydown', onKey)
    window.addEventListener('scroll', close, true)
    return () => {
      window.removeEventListener('mousedown', close)
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('scroll', close, true)
    }
  }, [hitMenu])
  /** 工作日序号 → 自然日列偏移（条形 x 坐标；跨周末自然变宽，斑马口径） */
  const dayNo = useMemo(() => (wd: number) => {
    return Math.round((wdToDate(project.startDate, wd, cal).getTime() - startMs) / 86400000)
  }, [project.startDate, cal, startMs])

  const days = Math.max(dayNo(Math.max(cpm.duration, 7)) + 3, 21, deadlineOff(project.deadline, startMs) + 2)
  const chartW = days * dayW
  const totalH = rows.length * ROW_H

  /** ZPert 复刻页默认把整份计划压入画布宽度，下方时标自然切到“周”。 */
  React.useEffect(() => {
    if (showToolbar) return
    const raf = requestAnimationFrame(() => {
      const pane = canvasPaneRef.current
      const avail = pane?.clientWidth ?? 0
      if (!pane || avail <= 0) return
      const w = Math.min(DAY_W_MAX, Math.max(DAY_W_MIN, Math.floor(((avail - 2) / days) * 10) / 10))
      dayWRef.current = w
      setDayW(w)
      pane.scrollLeft = 0
    })
    return () => cancelAnimationFrame(raf)
  }, [showToolbar, days])

  /** 自然日列（斑马口径：每天一列，非工作日底纹） */
  const colDates = useMemo(
    () => Array.from({ length: days }, (_, i) => {
      const d = new Date(startMs + i * 86400000)
      return { d, iso: d.toISOString().slice(0, 10), offWork: !isWorkingDate(d, normalizeCalendar(cal)) }
    }),
    [startMs, cal, days],
  )
  /** 月段/周段的合成搬到 timescale.ts（行为不变：周一起始跨月不断、W1 起 4 周步进），
   *  这里只出"判定"——周层阈值仍是 ganttUtil.WEEK_TIER_BELOW 单一来源。 */
  const useWeekTier = dayW < WEEK_TIER_BELOW
  const todayCol = useMemo(() => {
    const off = Math.floor((Date.now() - startMs) / 86400000)
    return off >= 0 && off <= days ? off : null
  }, [startMs, days])

  /** 目标竣工线（自然日列位；超出画布为 null 不画） */
  const deadlineCol = useMemo(() => {
    const off = deadlineOff(project.deadline, startMs)
    return off >= 0 && off <= days ? off : null
  }, [project.deadline, startMs, days])

  /** 刻度线疏密（斑马「时间刻度线」四选一；gaea 落 `SchedNetworkStyle.tickLines`，
   *  缺省 small = 画小刻度线，RE §7.3）。 */
  const lineDensity = resolveLineDensity(project.networkStyle)
  /** 双标尺模型（纯函数产出，GanttTimeline 零计算渲染）：
   *  上行=年-月跨列段+日/周刻度线，下行=周/星期；今日/竣工线横坐标同源于此。 */
  const rulerModel = useMemo(
    () => buildRulerModel({ colDates, dayW, weekTier: useWeekTier, lineDensity, todayCol, deadlineCol }),
    [colDates, dayW, useWeekTier, lineDensity, todayCol, deadlineCol],
  )

  /** 工作日→自然日偏移反查表（拖拽落点吸附用） */
  const wdOffsets = useMemo(
    () => workdayOffsets(project.startDate, Math.max(cpm.duration + 30, 45), project.calendar),
    [project.startDate, project.calendar, cpm.duration],
  )

  // 资源成本（刀3）：computeCosts 每次 render 重算（与 CPM 同范式，工期变→成本自动变）。
  // 无任何资源/成本数据时成本列留空（诚实呈现，不留 ¥0 假象）；CPM 未过成本不出（fail-closed）。
  const costs = useMemo(() => computeCosts(project, cpm), [project, cpm])
  const costShown = hasCostData(project) && costs.ok

  /** 开始拖拽：拖移（转手动锁定）或右缘缩放（改工期）。循环依赖/分组行/里程碑缩放禁用。
   *  window 监听三件套单源 beginWindowDrag（FE7-12，与 AoaView 同壳）；本视图只注入
   *  位移折算（dropToWd/resizeToDuration，cd 行=自然日口径刀3）与提交落库。 */
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
    beginWindowDrag({
      move: (st, ev) => {
        const dx = ev.clientX - startX
        if (dx !== 0) st.moved = true
        st.preview = kind === 'move'
          ? dropToWd(wdOffsets, dayNo(origEs), dx, dayW)
          : cdTask
            ? resizeToNaturalDuration(wdOffsets, origEs, dayNo(origEf), dx, dayW)
            : resizeToDuration(wdOffsets, origEs, dayNo(origEf), dx, dayW)
        setDrag({ ...st })
      },
      moved: (st) => st.moved,
      commit: (st) => {
        if (st.kind === 'move') {
          if (t.mode === 'manual') updateTask(t.id, { manualStart: st.preview })
          else updateTask(t.id, { mode: 'manual', manualStart: st.preview })
        } else {
          updateTask(t.id, { duration: st.preview })
        }
      },
      end: () => setDrag(null),
    }, st)
  }

  /** 依赖线（刀B）：按四种搭接类型取各自锚点画正交折线；两端均可见才画，
   *  y 按可见行序（筛选后行位变化，全表序会错位——刀C 余项口径）。
   *  W4 横道网络图（斑马 ganttNetwork）：同一几何点额外给类型/时距标注与中点。 */
  const relationMode = project.networkStyle?.mode === 'ganttNetwork'
  const linkPaths = project.links.map((l) => {
    const fy = visPos.get(l.from)
    const ty = visPos.get(l.to)
    const f = cpm.rows[l.from]
    const t2 = cpm.rows[l.to]
    if (fy === undefined || ty === undefined || !f || !t2) return null
    const x1s = dayNo(f.es) * dayW
    const x1f = dayNo(f.ef) * dayW
    const x2s = dayNo(t2.es) * dayW
    const x2f = dayNo(t2.ef) * dayW
    return {
      d: ganttLinkPath(l.type, {
        x1s,
        x1f,
        y1: fy * ROW_H + ROW_H / 2,
        x2s,
        x2f,
        y2: ty * ROW_H + ROW_H / 2,
      }),
      key: `${l.from}>${l.to}>${l.type}`,
      lk: linkKey(l),
      ...(relationMode
        ? {
            label: `${l.type}${l.lag ? (l.lag > 0 ? `+${l.lag}` : `${l.lag}`) : ''}`,
            mx: (x1f + x2s) / 2,
            my: ((fy + ty) / 2) * ROW_H + ROW_H / 2,
          }
        : {}),
    }
  }).filter(Boolean) as { d: string; key: string; lk: string; label?: string; mx?: number; my?: number }[]

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
    <div
      className={`sched-gantt${showToolbar ? '' : ' sched-zpert-mode'}${brush ? ' sched-brush-mode' : ''}`}
      data-testid="sched-gantt"
      ref={rootRef}
      onContextMenuCapture={onHitContextMenu}
    >
      {showToolbar && <GanttToolbar
        project={project}
        viewId={viewId}
        onViewChange={applyView}
        showFront={showFront}
        onToggleFront={() => setShowFront((v) => !v)}
        showBase={showBase}
        onToggleBase={() => setShowBase((v) => !v)}
        checkDate={checkDate}
        setCheckDate={setCheckDate}
        filter={filter}
        setFilter={setFilter}
        onFilterKindChange={changeFilterKind}
        sort={sort}
        onSortChange={changeSort}
        groupBy={groupBy}
        onGroupChange={changeGroup}
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
      />}
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
            onSelect={(id) => (brush ? applyBrushTo(id) : select(id))}
            onUpdateTask={updateTask}
            onToggleCollapse={toggleCollapse}
            rowMenu={rowMenu}
            onRowMenuClick={rowMenuClick}
            onInspect={onInspect}
            onMoveRow={moveRow}
            colFilters={colFilters}
            onColFiltersChange={setColFilters}
          />
        </div>
        {/* 分隔条：拖动收纳/展开表格窗格，双击复位全列 */}
        <div
          className="sched-gantt-split"
          data-testid="sched-gantt-split"
          data-hit="splitter"
          onMouseDown={beginSplit}
          onDoubleClick={resetSplit}
          title="拖动收纳/展开表格 · 双击复位"
        />
        {/* 画布窗格（右）：时间轴+条形+覆盖层；横滚独立，纵滚同步表格体 */}
        <div
          className="sched-gantt-canvaspane"
          data-testid="sched-gantt-canvas"
          data-hit="canvas"
          ref={canvasPaneRef}
          onScroll={onCanvasScroll}
        >
          {/* --sched-ruler-h：画布顶部偏移的唯一来源（= 表头高 = GANTT_RULER_H），
              ruler.css / 后续覆盖层偏移都读这一个变量，不再各写字面量。 */}
          <div
            className="sched-gantt-canvas-inner"
            style={{ width: chartW, '--sched-ruler-h': `${GANTT_RULER_H}px` } as React.CSSProperties}
          >
            <div className="sched-gantt-row sched-gantt-head" style={{ height: GANTT_RULER_H, width: chartW }}>
              {/* 双标尺（斑马 ZPert8 W4）：上行工程标尺=年-月跨列段 + 日/周刻度线，
                  下行进度标尺=周/星期；两行常驻，总高 = GANTT_RULER_H（与表格表头同值） */}
              <GanttTimeline chartW={chartW} model={rulerModel} />
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
              relationMode={relationMode}
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
      {/* 右键菜单浮层：条目来自 hitMenu（命中区 + 上下文），位置跟随命中点 */}
      {hitMenu && (
        <div
          className="sched-hit-menu-layer"
          data-testid="sched-hit-menu"
          data-hit-region={hitMenu.region}
          style={{ left: hitMenu.x, top: hitMenu.y }}
          onMouseDown={(e) => e.stopPropagation()}
          onContextMenu={(e) => e.preventDefault()}
        >
          <Menu
            className="sched-hit-menu"
            selectable={false}
            items={toMenuItems(hitMenu.items)}
            onClick={(info) => {
              runHitAction(info.key)
              setHitMenu(null)
            }}
          />
        </div>
      )}
    </div>
  )
}
