/**
 * SchedulePage — 「进度计划」一级板块（v4.110 刀1 / v4.111 刀2 / v4.113 刀4）
 *
 * 工程进度计划编制工作台：一套任务表数据驱动三种视图自由切换——
 * 横道图（甘特）/ 单代号网络图（PDM 六格）/ 双代号网络图（AOA 虚工作自动生成）。
 * CPM 引擎纯前端计算（FS/SS/FF/SF + 时距、正逆推、总/自由时差、关键线路、
 * 手动/自动任务模式），工作日历推算日期，MS Project XML 导入导出互通。
 * 刀4 起：计划文件化（进度计划/当前计划.gsched.json）——板块与 agent 共享
 * 同一资产（GaeaScheduleLoad/Save 水合+自动保存，agent 经 schedule_* 工具读写）。
 * v4.139 #15 多工程：每文件一工程+Go 侧索引指针，页头工程切换器（Select）与
 * 管理入口（新建/改名/归档/删除/载入，GaeaScheduleProjects/Project* 五绑定）；
 * 仅一项工程时切换器收窄为纯显示，老用户零感知。
 * v4.140 命令区三行化（对标 MS Project / ProjectLibre）：菜单栏（文件/编辑/
 * 视图/任务，命令全量归属）+ 工具栏（高频命令分组快捷区，导入/导出收拢下拉，
 * 增 MPP 导入）+ 工程信息条（工程名/开工/目标竣工/日历/基线/资源 | 视图档位）。
 */
import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Alert, Button, Dropdown, Input, Menu, Modal, Popover, Select } from 'antd'
import type { MenuProps } from 'antd'
import {
  AimOutlined, AppstoreOutlined, BgColorsOutlined, CalendarOutlined, CheckOutlined, ClearOutlined, ClusterOutlined, CopyOutlined, DeleteOutlined, DownOutlined, ExportOutlined, FileExcelOutlined, FileImageOutlined, FileOutlined, FileTextOutlined, FontColorsOutlined, FontSizeOutlined, FormatPainterOutlined, FundOutlined, ImportOutlined,
  FieldTimeOutlined, FilterOutlined, LineHeightOutlined, MessageOutlined, NodeCollapseOutlined, NodeExpandOutlined, NodeIndexOutlined, PlusOutlined, RedoOutlined, RiseOutlined, SafetyOutlined, SearchOutlined, SettingOutlined, SnippetsOutlined, TableOutlined, TagOutlined, TeamOutlined, ThunderboltOutlined, PartitionOutlined, ToolOutlined, UndoOutlined, UpOutlined, WarningOutlined,
} from '@ant-design/icons'
import { computeCpm } from '../schedule/cpm'
import { computeCosts } from '../schedule/cost'
import { buildAoa } from '../schedule/aoa'
import { buildProjectXml, parseProjectXml } from '../schedule/mspdi'
import { checkProject } from '../schedule/checker'
import { makeCustomView } from '../schedule/views'
import { PredEditor } from '../schedule/gantt/PredEditor'
import { GANTT_GROUP_OPTIONS, MAX_GANTT_GROUP_DEPTH } from '../schedule/ganttGroup'
import { planFinishOf } from '../schedule/deadline'
import { latestReview } from '../schedule/reviews'
import {
  SCHEDULE_COMMANDS,
  commandForShortcut,
  scheduleCommandDisabledReason,
  type RuntimeScheduleCommand,
} from '../schedule/commandRegistry'
import type { GanttGroupField, GanttSort } from '../schedule/ganttGroup'
import type { SchedRenderMode, SchedTaskStyle, SchedView } from '../schedule/types'
import { taskWarn, warnSummary, type WarnAlgorithm } from '../schedule/warn'
import type { SchedLink, SchedProject } from '../schedule/types'
import { useScheduleStore, isGroupRow, initScheduleSync } from '../schedule/store'
/** W6 收尾：状态栏命中区。命中判定与菜单条目都走既有纯函数层（hitRegion/hitMenu），
 *  页面只负责挂 `data-hit` 与把条目渲染成浮层——不另写第二份右键菜单。 */
import { hitRegionOfElement, type HitRegion } from '../schedule/hitRegion'
import { menuForHit, type HitMenuContext, type HitMenuItem } from '../schedule/hitMenu'
import { depFreeSlacks, drivingLinkKeys, linkKey } from '../schedule/pathDriver'
import { TaskInspector, type InspectorDepRow } from '../schedule/TaskInspector'
import { WhatIfPanel } from '../schedule/WhatIfPanel'
import { wbsOf } from '../schedule/ganttGroup'
import { saveExportBlob, inShell } from '../schedule/exportArtifact'
import { b64ToBytes } from '../gaea/lib/bytes'
import { exportScheduleXlsx, importScheduleXlsx, importScheduleMpp } from '../schedule/api'
import { app } from '../gaea/lib/bridge'
import { ResourcePanel } from '../schedule/ResourcePanel'
import { UsageView } from '../schedule/UsageView'
import { computeUsage } from '../schedule/usage'
import { BaselinesPanel } from '../schedule/BaselinesPanel'
import { ReviewPanel } from '../schedule/ReviewPanel'
import { ScheduleDock } from '../schedule/ScheduleDock'
import { TemplatesPanel } from '../schedule/TemplatesPanel'
import { fmtCost, hasCostData } from '../schedule/costUi'
import { SCHEDULE_FILE_PATH } from '../schedule/gschedSummary'
import { usePreviewStore } from '../gaea/lib/store'
import { emitFrontendEvent, FRONTEND_EVENTS } from '../events'
import { ScheduleChatPane } from '../schedule/ChatPane'
import { clampChatWidth, loadChatPrefs, saveChatPrefs, CHAT_WIDTH_DEFAULT } from '../schedule/chatPrefs'
import { DISPLAY_PRESETS, normalizePreset, presetClass, type DisplayPreset } from '../schedule/displayPreset'
import '../schedule/display-preset.css'
import { GanttView } from '../schedule/GanttView'
import { DcmaView } from '../schedule/DcmaView'
import { MonteView } from '../schedule/MonteView'
import { EvmView } from '../schedule/EvmView'
// 瘦身 P3：页面内子区块分片至此目录（正文零改动，仅迁移）
import { ExportDialog } from './schedule/ExportDialog'
import { ProjectsManagePanel } from './schedule/ProjectsManagePanel'
import { ProjectSettingsPanel } from './schedule/ProjectSettingsPanel'
import { ScheduleNetworkWorkspace } from './schedule/ScheduleNetworkWorkspace'
import { ScheduleCommandPalette } from './schedule/ScheduleCommandPalette'
import { ScheduleNetworkStyleControls } from './schedule/ScheduleNetworkStyleControls'
import {
  RibbonColumn,
  RibbonCommand,
  RibbonGroup,
  type ScheduleRibbonTabKey,
} from './schedule/ScheduleRibbon'
import '../schedule/schedule.css'
// W6 收尾：状态栏右键菜单浮层复用 hit.css（与横道命中菜单同一套类名/主题令牌）
import '../schedule/hit.css'
import '../gaea/styles.css'
import '../gaea/redesign.css'
import '../gaea/tailwind.css'
// 斑马外观密度层（纯照 ZPert8：实底/细线/30px 行高/4px 控件圆角/零光效）：
// 必须排在 schedule.css、display-preset.css、hit.css 之后，同特异性规则才压得住
import '../schedule/sched-zebra.css'

const SCHEDULE_RIBBON_TABS: ReadonlyArray<{ key: ScheduleRibbonTabKey; label: string }> = [
  { key: 'file', label: '文件' },
  { key: 'plan', label: '计划' },
  { key: 'resource', label: '资源' },
  { key: 'control', label: '管控' },
  { key: 'collab', label: '协作' },
  { key: 'view', label: '视图' },
  { key: 'help', label: '帮助' },
]

/**
 * 浮层 Esc 收起（统一纪律，W6 键盘可达）。
 *
 * 页面里有 9 个受控浮层（保存视图/视图管理/分组/查找/列筛选/关系/模板/工程管理/图面样式），
 * 键盘契约只该有一条实现：任一浮层打开时按 `Esc` → 收起**并把焦点还给打开它之前的元素**。
 * 为什么焦点要回位：不回位时焦点留在被卸载/隐藏的浮层里，键盘用户会掉回文档根，
 * 之后的 Tab 从页面开头重来——这就是「能从键盘打开、不能从键盘离开」。
 * 就地编辑器（`.sched-inplace`）的 Esc 有自己的取消语义，这里不抢。
 */
function useLayerEsc(layers: ReadonlyArray<{ open: boolean; close: () => void }>) {
  const layersRef = useRef(layers)
  layersRef.current = layers
  const anyOpen = layers.some((layer) => layer.open)
  /** 打开前的焦点（开合切换时记一次；同一个浮层开着期间不变） */
  const returnTo = useRef<HTMLElement | null>(null)
  useEffect(() => {
    if (!anyOpen) return
    const atOpen = document.activeElement
    returnTo.current = atOpen instanceof HTMLElement ? atOpen : null
    const onKey = (ev: KeyboardEvent) => {
      if (ev.key !== 'Escape') return
      const target = ev.target as HTMLElement | null
      if (target && typeof target.closest === 'function' && target.closest('.sched-inplace')) return
      const layer = layersRef.current.find((item) => item.open)
      if (!layer) return
      ev.preventDefault()
      layer.close()
      const back = returnTo.current
      if (back && typeof back.focus === 'function') back.focus()
    }
    // 捕获阶段先收浮层：antd 自身的 Esc 处理在冒泡阶段
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [anyOpen])
}

const SchedulePage: React.FC = () => {
  const project = useScheduleStore((s) => s.project)
  const view = useScheduleStore((s) => s.view)
  const setView = useScheduleStore((s) => s.setView)
  const selectedId = useScheduleStore((s) => s.selectedId)
  const importProject = useScheduleStore((s) => s.importProject)
  const importAsProject = useScheduleStore((s) => s.importAsProject)
  const addTask = useScheduleStore((s) => s.addTask)
  const addGroup = useScheduleStore((s) => s.addGroup)
  const updateTask = useScheduleStore((s) => s.updateTask)
  const removeTask = useScheduleStore((s) => s.removeTask)
  const loadSample = useScheduleStore((s) => s.loadSample)
  // 视图库（斑马复刻 S6.2「保存视图/视图管理」）：保存/应用/删除直连 store
  const saveCustomView = useScheduleStore((s) => s.saveCustomView)
  const removeCustomView = useScheduleStore((s) => s.removeCustomView)
  const renameCustomView = useScheduleStore((s) => s.renameCustomView)
  const requestCustomView = useScheduleStore((s) => s.requestCustomView)
  const setNetworkMode = useScheduleStore((s) => s.setNetworkMode)
  const setNetworkStyle = useScheduleStore((s) => s.setNetworkStyle)
  // 大纲/表格命令（一次点亮批1+2+4）：store 动作 + 信号通道
  const promoteTask = useScheduleStore((s) => s.promoteTask)
  const demoteTask = useScheduleStore((s) => s.demoteTask)
  const moveTask = useScheduleStore((s) => s.moveTask)
  const pasteRows = useScheduleStore((s) => s.pasteRows)
  const sendGanttCommand = useScheduleStore((s) => s.sendGanttCommand)
  const clearAll = useScheduleStore((s) => s.clearAll)
  const past = useScheduleStore((s) => s.past)
  const future = useScheduleStore((s) => s.future)
  const undo = useScheduleStore((s) => s.undo)
  const redo = useScheduleStore((s) => s.redo)
  const fileRef = useRef<HTMLInputElement>(null)
  const [importMsg, setImportMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  // ── 视图库（斑马复刻 S6.2）：保存当前横道视图态 / 管理已存视图 ──
  const customViews = project.views ?? []
  const [saveViewOpen, setSaveViewOpen] = useState(false)
  const [saveViewName, setSaveViewName] = useState('')
  const [saveViewErr, setSaveViewErr] = useState<string | null>(null)
  const [manageViewOpen, setManageViewOpen] = useState(false)
  const [renameViewId, setRenameViewId] = useState<string | null>(null)
  const [renameViewName, setRenameViewName] = useState('')
  const [renameViewErr, setRenameViewErr] = useState<string | null>(null)
  const [outlineMsg, setOutlineMsg] = useState<string | null>(null)
  const [findOpen, setFindOpen] = useState(false)
  const [findText, setFindText] = useState('')
  /** 列筛选入口（W5）：表头 ▼ 是主入口，功能区这份弹层负责说明与一键清除 */
  const [colFilterOpen, setColFilterOpen] = useState(false)
  /** 显示预设（W6）：护眼/黑白只覆盖 --sched-* 领域色，主题令牌不动；本机偏好 */
  const [displayPreset, setDisplayPreset] = useState<DisplayPreset>(() => loadChatPrefs().ganttDisplayPreset)
  const changeDisplayPreset = (next: DisplayPreset) => {
    setDisplayPreset(normalizePreset(next))
    saveChatPrefs({ ganttDisplayPreset: next })
  }
  const [groupOpen, setGroupOpen] = useState(false)
  const [groupPick, setGroupPick] = useState<GanttGroupField[]>([])
  /** 图面样式浮层（W6 键盘可达：改成受控，Esc 才能收起并回焦） */
  const [styleOpen, setStyleOpen] = useState(false)
  const [relationOpen, setRelationOpen] = useState(false)
  const [templatesOpen, setTemplatesOpen] = useState(false)
  const [commandOpen, setCommandOpen] = useState(false)
  const [exportOpen, setExportOpen] = useState(false)
  const [dockOpen, setDockOpen] = useState(true)
  const [ribbonTab, setRibbonTab] = useState<ScheduleRibbonTabKey>('plan')
  /** ── 功能区页签键盘（W6）：roving tabindex + ←/→/Home/End + Enter/Space ────
   *  `.sched-menu-item` 在本仓库只有「功能区页签」一种用法，语义是 tablist 而不是菜单：
   *  页签没有「可关闭对象」，所以 Esc 不在页签上假装关东西；Esc 收起+回焦落在真正
   *  承载浮层的组件（受控 Popover / 右键菜单 / 命令面板）上，见 useLayerEsc 与
   *  GanttView/ScheduleCommandPalette。 */
  const ribbonTabRefs = useRef(new Map<ScheduleRibbonTabKey, HTMLButtonElement>())
  const focusRibbonTab = (key: ScheduleRibbonTabKey) => ribbonTabRefs.current.get(key)?.focus()
  const onRibbonTabKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>, key: ScheduleRibbonTabKey) => {
    const at = SCHEDULE_RIBBON_TABS.findIndex((tab) => tab.key === key)
    if (event.key === 'ArrowRight' || event.key === 'ArrowLeft') {
      event.preventDefault()
      const size = SCHEDULE_RIBBON_TABS.length
      const next = SCHEDULE_RIBBON_TABS[(at + (event.key === 'ArrowRight' ? 1 : -1) + size) % size]
      setRibbonTab(next.key) // 自动激活（ARIA tablist 口径），焦点同步跟到新页签
      focusRibbonTab(next.key)
      return
    }
    if (event.key === 'Home' || event.key === 'End') {
      event.preventDefault()
      const next = event.key === 'Home' ? SCHEDULE_RIBBON_TABS[0] : SCHEDULE_RIBBON_TABS[SCHEDULE_RIBBON_TABS.length - 1]
      setRibbonTab(next.key)
      focusRibbonTab(next.key)
      return
    }
    if (event.key === 'Enter' || event.key === ' ') {
      // 页签是 <button>：真浏览器会自己派发 click，这里显式接一次并 preventDefault 防双触发
      event.preventDefault()
      setRibbonTab(key)
    }
  }
  /** 任务检查器（v4.136 小刀）：inspectId=正在检查的任务；右键菜单「任务检查器」/检查器内点前置名跳转 */
  const [inspectId, setInspectId] = useState<string | null>(null)
  const hydrated = useScheduleStore((s) => s.hydrated)
  const sync = useScheduleStore((s) => s.sync)
  const savedAt = useScheduleStore((s) => s.savedAt)
  const syncError = useScheduleStore((s) => s.syncError)
  /** 左栏 AI 对话偏好（刀11：折叠态+宽度持久化 localStorage） */
  const [chat, setChat] = useState(loadChatPrefs)
  const chatDragRef = useRef<{ startX: number; startW: number } | null>(null)
  /** 多工程（v4.139 #15）：projects/currentPath 由 store 提供（Go 索引指针视图）；
   *  旧 store 形态/未加载时为空 → 切换器整体隐藏，老用户零感知。 */
  const projects = useScheduleStore((s) => s.projects)
  const currentPath = useScheduleStore((s) => s.currentPath)
  const openProject = useScheduleStore((s) => s.openProject)
  const refreshProjects = useScheduleStore((s) => s.refreshProjects)
  /** 切换/管理动作在途（Select loading 态） */
  const [projBusy, setProjBusy] = useState(false)
  /** 工程管理弹层受控开合（菜单「文件 → 工程管理…」与 ⚙ 按钮同一入口） */
  const [manageOpen, setManageOpen] = useState(false)
  const projectList = projects ?? []

  /* ── 底部状态栏命中区（W6 收尾）────────────────────────────────────
   * 缺口：命中区枚举早就含 `status-bar`（hitRegion）且 `hitMenu.statusBarMenu()`
   * 已有条目，但页面状态栏没挂 `data-hit`，右键在它上面毫无反应。
   * 这里只补两件事：给容器挂属性、把右键接到**同一个**统一入口
   * （`hitRegionOfElement` 上溯 + `menuForHit` 出条目）—— 条目内容仍由纯函数层给。
   */
  const [statusMenu, setStatusMenu] = useState<{
    x: number
    y: number
    region: HitRegion
    items: HitMenuItem[]
  } | null>(null)
  /** 状态栏容器（W6：Esc 收起右键菜单后把焦点还给这里，见下方 effect） */
  const statusBarRef = useRef<HTMLDivElement | null>(null)
  /** 状态栏菜单上下文：选中行/分组/序号口径与横道右键一致（statusBarMenu 不读 ctx，仍如实给） */
  const statusMenuCtx = (): HitMenuContext => {
    const idx = selectedId ? project.tasks.findIndex((t) => t.id === selectedId) : -1
    const sel = idx >= 0 ? project.tasks[idx] : null
    return {
      hasSelection: !!sel,
      selectedIsGroup: idx >= 0 && isGroupRow(project.tasks, idx),
      selectedIndex: idx,
      taskCount: project.tasks.length,
      selectedIsMilestone: sel?.isMilestone,
      canUndo: past.length > 0,
      canRedo: future.length > 0,
      column: null,
    }
  }
  const onStatusBarContextMenu = (e: React.MouseEvent<HTMLDivElement>) => {
    // 统一命中：`data-hit` 自限深度上溯 + 优先级裁决（与 GanttView.onHitContextMenu 同口径）
    const region = hitRegionOfElement(e.target as Element | null)
    if (!region) return
    const items = menuForHit(region, statusMenuCtx())
    if (items.length === 0) return
    e.preventDefault()
    e.stopPropagation()
    setStatusMenu({ x: e.clientX, y: e.clientY, region, items })
  }
  /** 关菜单：点外面 / Esc / 滚动都收起（命中浮层「别钉在屏幕上」的同款纪律）
   *  W6：Esc 收起后焦点回状态栏本身（键盘用户不掉回文档根）。 */
  useEffect(() => {
    if (!statusMenu) return
    const close = () => setStatusMenu(null)
    const onKey = (ev: KeyboardEvent) => {
      if (ev.key !== 'Escape') return
      setStatusMenu(null)
      statusBarRef.current?.focus?.()
    }
    window.addEventListener('mousedown', close)
    window.addEventListener('keydown', onKey)
    window.addEventListener('scroll', close, true)
    return () => {
      window.removeEventListener('mousedown', close)
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('scroll', close, true)
    }
  }, [statusMenu])
  /** 功能区受控浮层的统一 Esc 收起 + 焦点回位（九个浮层共用一条实现） */
  useLayerEsc([
    { open: saveViewOpen, close: () => setSaveViewOpen(false) },
    { open: manageViewOpen, close: () => setManageViewOpen(false) },
    { open: groupOpen, close: () => setGroupOpen(false) },
    { open: findOpen, close: () => setFindOpen(false) },
    { open: colFilterOpen, close: () => setColFilterOpen(false) },
    { open: relationOpen, close: () => setRelationOpen(false) },
    { open: templatesOpen, close: () => setTemplatesOpen(false) },
    { open: manageOpen, close: () => setManageOpen(false) },
    { open: styleOpen, close: () => setStyleOpen(false) },
  ])
  /** 命中的条目 → antd Menu 项（禁用原因挂 title，用户知道为什么点不动） */
  const statusMenuItems: MenuProps['items'] = (statusMenu?.items ?? []).map((it) => ({
    key: it.key,
    label: it.label,
    disabled: !it.enabled,
    title: it.disabledReason,
    danger: it.danger,
  }))

  const toggleChat = () => setChat(saveChatPrefs({ collapsed: !chat.collapsed }))

  /** 拖宽分隔条：左移加宽；拖完持久化；双击复位 420 */
  const onDividerMouseDown = (e: React.MouseEvent) => {
    e.preventDefault()
    chatDragRef.current = { startX: e.clientX, startW: chat.width }
    const onMove = (ev: MouseEvent) => {
      const w = clampChatWidth(chatDragRef.current!.startW - (ev.clientX - chatDragRef.current!.startX))
      setChat((p) => ({ ...p, width: w }))
    }
    const onUp = () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      setChat((p) => saveChatPrefs({ width: p.width }))
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }
  const resetChatWidth = () => setChat(saveChatPrefs({ width: CHAT_WIDTH_DEFAULT }))

  /** 反向入口（刀12 办公联动）：在办公板块中预览当前工程计划文件（store 预置，未挂载也能落地）
   *  v4.139 #15：预览目标跟随当前工程 rel（缺省回落常量=老形态）。 */
  const previewRel = currentPath ?? SCHEDULE_FILE_PATH
  const openInOffice = () => {
    usePreviewStore.getState().openFilePreview(previewRel)
    emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: 'gaea' })
  }

  // 文件同步：水合（文件为准，localStorage 迁移）+ 自动保存 + agent 写入回读
  useEffect(() => { void initScheduleSync() }, [])

  const cpm = useMemo(() => computeCpm(project.tasks, project.links, { planFinish: planFinishOf(project), calendar: project.calendar, startDate: project.startDate }), [project])
  // 网络显示模式（斑马 display_mode 随文档持久化；netMode 从 networkStyle 派生为单一事实源）
  const persistedMode = project.networkStyle?.mode
  const netMode: SchedRenderMode = persistedMode === 'logic' || persistedMode === 'timescaledLogic' || persistedMode === 'ganttNetwork'
    ? persistedMode
    : 'timescaled'
  const aoa = useMemo(() => buildAoa(project.tasks, project.links, { planFinish: planFinishOf(project), persisted: project.aoa, mode: netMode }), [project, netMode])
  // 计划检查（斑马复刻 S1）：项目级只把「循环依赖」带进任务抽屉（其余项目级提示属于整份计划）
  const diags = useMemo(() => checkProject(project, cpm), [project, cpm])
  // 资源使用/超载（v4.137 #12/#13）：逐工作日负载 vs 可用性（工时资源；个人日历收紧可用性）
  const usage = useMemo(() => computeUsage(project, cpm), [project, cpm])
  // 资源成本（刀3）：总成本随每次 render 重算（与 CPM 同范式）；无资源/成本数据不显示（诚实呈现）
  const costs = useMemo(() => computeCosts(project, cpm), [project, cpm])
  const showCost = hasCostData(project) && costs.ok

  const selected = project.tasks.find((t) => t.id === selectedId) ?? null
  const selectedIsGroup = selected ? isGroupRow(project.tasks, project.tasks.findIndex((t) => t.id === selected!.id)) : false

  // 任务检查器数据（v4.136）：逐依赖自由时差 + 驱动标记（ProjectLibre E8 口径）。
  // 只查叶任务（分组行无排程语义）；驱动边=ff===0 且后继 auto——即「绑定了本任务日期」的那条搭接。
  const inspectTask = inspectId ? project.tasks.find((t) => t.id === inspectId) ?? null : null
  /** 选中任务的检查发现（含项目级的循环依赖——它会让整张图排不出日期） */
  const inspectDiags = useMemo(
    () => (inspectId ? diags.filter((d) => d.taskId === inspectId || (!d.taskId && d.code === 'cycle')) : []),
    [diags, inspectId],
  )
  // 预警（斑马复刻 S5）：两算法视图态切换（斑马「预警超期算法计划-要求/预测-要求」），
  // 不入文件（同横道前锋线 checkDate 的视图态纪律）；摘要计数走最新一期检视的预测口径。
  const [warnAlgo, setWarnAlgo] = useState<WarnAlgorithm>('plan-ask')
  const latest = useMemo(() => latestReview(project), [project])
  const inspectWarn = useMemo(
    () => (inspectTask && inspectTask.level > 0 ? taskWarn(inspectTask, cpm, latest, warnAlgo) : null),
    [inspectTask, cpm, latest, warnAlgo],
  )
  const warn = useMemo(
    () => warnSummary(project, cpm, latest, warnAlgo),
    [project, cpm, latest, warnAlgo],
  )
  const warnRed = warn.red
  const inspectorData = useMemo(() => {
    if (!inspectTask || inspectTask.level === 0) return null
    const slacks = depFreeSlacks(project.tasks, project.links, cpm)
    const driving = drivingLinkKeys(project.tasks, project.links, cpm)
    const mkRow = (l: SchedLink, otherId: string): InspectorDepRow => ({
      link: l,
      otherId,
      otherName: project.tasks.find((t) => t.id === otherId)?.name ?? otherId,
      freeSlack: slacks[linkKey(l)] ?? 0,
      driving: driving.has(linkKey(l)),
    })
    return {
      predecessors: project.links.filter((l) => l.to === inspectTask.id).map((l) => mkRow(l, l.from)),
      successors: project.links.filter((l) => l.from === inspectTask.id).map((l) => mkRow(l, l.to)),
    }
  }, [inspectTask, project.tasks, project.links, cpm])
  const select = useScheduleStore((s) => s.select)

  const exportXml = async () => {
    // v4.350：补成功/失败反馈——此前无任何反馈，另存为抛错只落 unhandled
    // rejection，用户「点了没反应」（对齐 exportXlsx 的消息口径；触发走菜单
    // 无 busy UI 位，另存为对话框本身模态阻塞，无重入问题）。
    try {
      const xml = buildProjectXml(project, cpm.rows)
      const blob = new Blob([xml], { type: 'application/xml;charset=utf-8' })
      const saved = await saveExportBlob(blob, `${project.name || '进度计划'}.xml`)
      if (saved) setImportMsg({ type: 'success', text: '已导出 XML' })
    } catch (e) {
      setImportMsg({ type: 'error', text: e instanceof Error ? e.message : String(e) })
    }
  }

  const [xmlBusy, setXmlBusy] = useState(false)
  const onImportFile = async (file: File) => {
    // xmlBusy 与 xlsx/mpp 同口径：XML 导入此前无在途反馈，导入按钮 loading 盖不到（审计 #4）
    setXmlBusy(true)
    try {
      const text = await file.text()
      const r = parseProjectXml(text)
      if (r.ok && r.project) {
        importProject(r.project)
        setImportMsg({ type: 'success', text: `已导入「${r.project.name}」：${r.project.tasks.length} 行 / ${r.project.links.length} 条搭接` })
      } else {
        setImportMsg({ type: 'error', text: r.error ?? '导入失败' })
      }
    } finally {
      setXmlBusy(false)
    }
  }

  const [xlsxBusy, setXlsxBusy] = useState(false)
  const xlsxRef = useRef<HTMLInputElement>(null)

  /** 导出上报 Excel（Go excelize 渲染，CPM fail-closed） */
  const exportXlsx = async () => {
    setXlsxBusy(true)
    try {
      await exportScheduleXlsx(project)
      setImportMsg({ type: 'success', text: '已导出 Excel（进度计划.xlsx）' })
    } catch (e) {
      setImportMsg({ type: 'error', text: e instanceof Error ? e.message : String(e) })
    } finally {
      setXlsxBusy(false)
    }
  }

  /** 导入上报 Excel：表头别名识别 + 前置引用回链，排程交回 CPM 重算 */
  const onImportXlsx = async (file: File) => {
    setXlsxBusy(true)
    try {
      const p = await importScheduleXlsx(file)
      importProject(p)
      setImportMsg({ type: 'success', text: `已导入「${p.name}」：${p.tasks.length} 行 / ${p.links.length} 条搭接（已按搭接重排）` })
    } catch (e) {
      setImportMsg({ type: 'error', text: e instanceof Error ? e.message : String(e) })
    } finally {
      setXlsxBusy(false)
    }
  }

  /** 导入 MPP（v4.140）：二进制 MS Project 工程（MPP9/12/14 子集）走 Go 解析器
   *  GaeaScheduleImportMpp，任务/搭接/资源/分配入模型，排程交回 CPM 重算 */
  const [mppBusy, setMppBusy] = useState(false)
  const mppRef = useRef<HTMLInputElement>(null)
  const newRef = useRef<HTMLInputElement>(null)
  const onImportMpp = async (file: File) => {
    setMppBusy(true)
    try {
      const p = await importScheduleMpp(file)
      if (!p.name) p.name = file.name.replace(/\.mpp$/i, '') // 文件内无工程名,以文件名兜底
      importProject(p)
      setImportMsg({ type: 'success', text: `已导入「${p.name}」（MPP）：${p.tasks.length} 行 / ${p.links.length} 条搭接（已按搭接重排；约束日期/日历例外未映射）` })
    } catch (e) {
      setImportMsg({ type: 'error', text: e instanceof Error ? e.message : String(e) })
    } finally {
      setMppBusy(false)
    }
  }

  /** 导入为新工程（v4.144 主入口）：XML/Excel/MPP 按扩展名分发，解析成功即
   *  落独立工程文件入索引——非破坏口径（不覆盖当前工程，同 v4.139 复制=切
   *  指针）；Create 失败返回 false 时当前工程未被改动，提示诚实上屏 */
  const onImportNewFile = async (file: File) => {
    const ext = (file.name.split('.').pop() ?? '').toLowerCase()
    try {
      let p: SchedProject | null = null
      let note = ''
      if (ext === 'xml') {
        const r = parseProjectXml(await file.text())
        if (!r.ok || !r.project) {
          setImportMsg({ type: 'error', text: r.error ?? '导入失败' })
          return
        }
        p = r.project
      } else if (ext === 'xlsx' || ext === 'xlsm') {
        setXlsxBusy(true)
        p = await importScheduleXlsx(file)
        note = '（已按搭接重排）'
      } else if (ext === 'mpp') {
        setMppBusy(true)
        p = await importScheduleMpp(file)
        note = '（已按搭接重排；约束日期/日历例外未映射）'
      } else {
        setImportMsg({ type: 'error', text: `不支持的格式 .${ext || '?'}：导入为新工程支持 XML / Excel / MPP` })
        return
      }
      if (!p.name) p.name = file.name.replace(/\.(xml|xlsx|xlsm|mpp)$/i, '')
      if (!(await importAsProject(p))) {
        setImportMsg({ type: 'error', text: `新工程「${p.name}」创建失败，当前工程未被修改；详见状态栏错误` })
        return
      }
      setImportMsg({ type: 'success', text: `已导入为新工程「${p.name}」：${p.tasks.length} 行 / ${p.links.length} 条搭接${note}` })
    } catch (e) {
      setImportMsg({ type: 'error', text: e instanceof Error ? e.message : String(e) })
    } finally {
      setXlsxBusy(false)
      setMppBusy(false)
    }
  }

  /**
   * 导入选文件（v4.162）：Wails 壳内 <input type=file>.click() 不弹文件对话框
   * （用户实测「导入点击没有反应」），壳内改走 GaeaPickFiles 系统对话框 +
   * GaeaReadFileB64 读内容还原 File 喂原解析器；浏览器回退原 input.click()
   * （浏览器原生弹框，jsdom/浏览器用例口径不变）。picked=空（取消）静默返回。
   */
  const pickImport = async (kind: 'new' | 'xml' | 'xlsx' | 'mpp', fallback: () => void) => {
    if (!inShell()) { fallback(); return }
    try {
      const picked = await app.PickFiles()
      if (!picked?.length) return
      const f = picked[0]
      const b64 = await app.ReadFileB64(f.path)
      const bytes = b64ToBytes(b64)
      const file = new File([bytes], f.name)
      if (kind === 'xml') void onImportFile(file)
      else if (kind === 'xlsx') void onImportXlsx(file)
      else if (kind === 'mpp') void onImportMpp(file)
      else void onImportNewFile(file)
    } catch (e) {
      setImportMsg({ type: 'error', text: e instanceof Error ? e.message : String(e) })
    }
  }

  // ── 菜单栏（v4.140）：命令按 文件/编辑/视图/任务 归属；导入/导出项与工具栏下拉共用 ──
  // v4.144 导入重组：「导入为新工程」=安全默认主入口（独立文件入索引）；
  // 三路「替换当前工程」降为显式标注项（模板/签证调整覆盖场景，可撤销）。

  // ── 视图库回调：快照源=chatPrefs（GanttView 每次参数变更实时回写，实时镜像）──
  const genViewId = () => `cv-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`
  const commitSaveView = () => {
    const prefs = loadChatPrefs()
    const made = makeCustomView(saveViewName, genViewId(), {
      hide: prefs.ganttHide,
      groupBy: prefs.ganttGroup as GanttGroupField[],
      sort: prefs.ganttSort as GanttSort,
      filterKind: prefs.ganttFilterKind,
    }, customViews)
    if (!made.view) { setSaveViewErr(made.reason ?? '保存失败'); return }
    const res = saveCustomView(made.view)
    if (!res.ok) { setSaveViewErr(res.reason ?? '保存失败'); return }
    setSaveViewOpen(false)
    setSaveViewName('')
    setSaveViewErr(null)
  }


  // ── 大纲/表格命令（一次点亮批1+2+4）：统一失败出口=outlineMsg 行内提示 ──
  const runOutline = (fn: () => { ok: boolean; reason?: string }) => {
    const res = fn()
    setOutlineMsg(res.ok ? null : (res.reason ?? '操作失败'))
  }
  const copySelectedRow = () => {
    if (!selected) return
    const row = cpm.rows[selected.id]
    const preds = project.links.filter((l) => l.to === selected.id).map((l) => `${l.from}${l.type === 'FS' ? '' : l.type}${l.lag ? (l.lag > 0 ? '+' : '') + l.lag : ''}`).join(',')
    const tsv = [
      ['名称', '工期(天)', '开始', '完成', '前置'].join('\t'),
      [selected.name, String(selected.isMilestone ? 0 : selected.duration), row ? String(row.es) : '', row ? String(row.ef) : '', preds].join('\t'),
    ].join('\n')
    void navigator.clipboard?.writeText(tsv).catch(() => setOutlineMsg('剪贴板不可用（浏览器权限）'))
  }
  const pasteFromClipboard = () => {
    void navigator.clipboard?.readText().then((text) => {
      const res = pasteRows(text)
      setOutlineMsg(res.ok ? null : (res.reason ?? '粘贴失败'))
    }).catch(() => setOutlineMsg('剪贴板不可用（浏览器权限）'))
  }

  // ── 命令注册表 → 运行期 handler（ZPert W2）──────────────────────────
  // 元数据在 commandRegistry（稳定 ID/分组/快捷键），handler 只在本页注册一次。
  // 功能区、菜单、命令面板与键盘都经 runCommand 进入同一禁用原因和执行路径。
  const selectedIndex = selected ? project.tasks.findIndex((task) => task.id === selected.id) : -1
  const commandAvailability = {
    hasSelection: !!selected,
    selectedIsGroup,
    selectedIndex,
    taskCount: project.tasks.length,
    canUndo: past.length > 0,
    canRedo: future.length > 0,
  }
  const confirmDeleteSelected = () => {
    if (!selected) return
    Modal.confirm({
      title: '删除选中项',
      content: selectedIsGroup ? '分组及其子任务一并删除' : `删除「${selected.name}」及其搭接关系`,
      onOk: () => removeTask(selected.id),
    })
  }
  const commandActions: Record<string, () => void | Promise<void>> = {
    'file.manage': () => { setRibbonTab('file'); setManageOpen(true) },
    'file.import-new': () => pickImport('new', () => newRef.current?.click()),
    'file.import-xml': () => pickImport('xml', () => fileRef.current?.click()),
    'file.import-xlsx': () => pickImport('xlsx', () => xlsxRef.current?.click()),
    'file.import-mpp': () => pickImport('mpp', () => mppRef.current?.click()),
    'file.export-xml': () => exportXml(),
    'file.export-xlsx': () => exportXlsx(),
    'file.export-art': () => setExportOpen(true),
    'edit.undo': undo,
    'edit.redo': redo,
    'edit.copy': copySelectedRow,
    'edit.paste': pasteFromClipboard,
    'edit.find': () => { setRibbonTab('plan'); setView('gantt'); setFindOpen(true) },
    'edit.column-filter': () => { setView('gantt'); setRibbonTab('plan'); setColFilterOpen(true) },
    'edit.delete': confirmDeleteSelected,
    'edit.clear': () => {
      Modal.confirm({ title: '清空全部任务与搭接？', onOk: () => { clearAll(); setImportMsg(null) } })
    },
    'task.add': () => addTask(selectedId ?? undefined),
    'task.group': addGroup,
    'task.relation': () => { setRibbonTab('plan'); setRelationOpen(true) },
    'task.milestone': () => {
      if (selected) updateTask(selected.id, { isMilestone: !selected.isMilestone })
    },
    'task.inspector': () => {
      if (selected) setInspectId(selected.id)
    },
    'outline.collapse': () => { setView('gantt'); sendGanttCommand('collapse-selected') },
    'outline.expand': () => { setView('gantt'); sendGanttCommand('expand-all') },
    'outline.promote': () => {
      if (selected) runOutline(() => promoteTask(selected.id))
    },
    'outline.demote': () => {
      if (selected) runOutline(() => demoteTask(selected.id))
    },
    'outline.up': () => {
      if (selected) runOutline(() => moveTask(selected.id, -1))
    },
    'outline.down': () => {
      if (selected) runOutline(() => moveTask(selected.id, 1))
    },
    'view.gantt': () => setView('gantt'),
    'view.pdm': () => setView('pdm'),
    'view.aoa': () => setView('aoa'),
    'view.timescaled': () => { setNetworkMode('timescaled'); setView('aoa') },
    'view.timescaledLogic': () => { setNetworkStyle({ mode: 'timescaledLogic' }); setView('aoa') },
    'view.logic': () => { setNetworkMode('logic'); setView('aoa') },
    'view.ganttNetwork': () => { setNetworkStyle({ mode: 'ganttNetwork' }); setView('gantt') },
    'view.usage': () => setView('usage'),
    'view.dcma': () => setView('dcma'),
    'view.monte': () => setView('monte'),
    'view.evm': () => setView('evm'),
    'display.dock': () => setDockOpen((open) => !open),
    'display.chat': toggleChat,
    'display.smart-layout': () => { setView('aoa'); sendGanttCommand('smart-layout') },
    'help.commands': () => setCommandOpen(true),
    'help.templates': () => { setRibbonTab('help'); setTemplatesOpen(true) },
    'help.sample': () => { loadSample(); setImportMsg(null) },
  }
  const commandActive: Record<string, boolean | undefined> = {
    'view.gantt': view === 'gantt',
    'view.pdm': view === 'pdm',
    'view.aoa': view === 'aoa',
    'view.timescaled': view === 'aoa' && netMode === 'timescaled',
    'view.timescaledLogic': view === 'aoa' && netMode === 'timescaledLogic',
    'view.logic': view === 'aoa' && netMode === 'logic',
    'view.ganttNetwork': view === 'gantt' && netMode === 'ganttNetwork',
    'view.usage': view === 'usage',
    'view.dcma': view === 'dcma',
    'view.monte': view === 'monte',
    'view.evm': view === 'evm',
    'display.dock': dockOpen,
    'display.chat': !chat.collapsed,
  }
  const runtimeCommands: RuntimeScheduleCommand[] = SCHEDULE_COMMANDS.flatMap((spec): RuntimeScheduleCommand[] => {
    const run = commandActions[spec.id]
    if (!run) return []
    return [{
      spec,
      run,
      active: commandActive[spec.id],
      disabledReason: scheduleCommandDisabledReason(spec.id, commandAvailability),
    }]
  })
  const commandById = new Map(runtimeCommands.map((command) => [command.spec.id, command]))
  const commandReason = (id: string) => commandById.get(id)?.disabledReason
  const commandTitle = (id: string, fallback?: string) => commandReason(id) ?? commandById.get(id)?.spec.detail ?? fallback
  const runCommand = (id: string) => {
    const command = commandById.get(id)
    if (!command) return
    if (command.disabledReason) {
      setOutlineMsg(command.disabledReason)
      return
    }
    setOutlineMsg(null)
    void Promise.resolve(command.run()).catch((error: unknown) => {
      setOutlineMsg(error instanceof Error ? error.message : String(error))
    })
  }
  const runCommandRef = useRef(runCommand)
  runCommandRef.current = runCommand
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.defaultPrevented) return
      const element = document.activeElement
      if (element instanceof HTMLElement && (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA' || element.isContentEditable)) return
      const spec = commandForShortcut(event)
      if (!spec) return
      event.preventDefault()
      runCommandRef.current(spec.id)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])


  // ── 格式组（斑马复刻批5）：外观四项作用于选中任务；色板=工程常用色 ──
  // 工程色板=用户可选内容数据（斑马取色器同源），非 UI 主题色 // hex-exempt
  const STYLE_COLORS = ['#237804', '#d4380d', '#0958d9', '#722ed1', '#c41d7f', '#d46b08', '#614700', '#002c8c'] // hex-exempt
  const STYLE_FONTS = ['默认', '微软雅黑', '宋体', '黑体', '楷体', '等线']
  const STYLE_SIZES = [12, 13, 14, 16, 18, 20]
  const patchStyle = (patch: Partial<SchedTaskStyle>) => {
    if (!selected) return
    const next = { ...selected.style, ...patch }
    if (!next.color) delete next.color
    if (!next.fill) delete next.fill
    if (!next.font) delete next.font
    if (!next.size) delete next.size
    updateTask(selected.id, { style: Object.keys(next).length > 0 ? next : undefined })
  }
  const swatchRow = (key: 'color' | 'fill') => (
    <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
      {[undefined, ...STYLE_COLORS].map((c) => (
        <button
          key={c ?? 'none'}
          type="button"
          aria-label={c ?? '清除'}
          onClick={() => patchStyle({ [key]: c } as Partial<SchedTaskStyle>)}
          style={{
            width: 18,
            height: 18,
            borderRadius: 4,
            border: '1px solid rgba(5,5,5,0.2)',
            background: c ?? 'transparent',
            backgroundImage: c ? undefined : 'linear-gradient(45deg, transparent 45%, #f5222d 45%, #f5222d 55%, transparent 55%)', // hex-exempt（清除态删除线）
            cursor: 'pointer',
          }}
          data-testid={`sched-style-${key}-${c ?? 'none'}`}
        />
      ))}
    </div>
  )

  const importMenuItems: MenuProps['items'] = [
    { key: 'importNew', icon: <ImportOutlined />, label: '导入为新工程…' },
    { key: 'importXml', icon: <ImportOutlined />, label: '导入 XML 替换当前工程…' },
    { key: 'importXlsx', icon: <FileExcelOutlined />, label: '导入 Excel 替换当前工程…' },
    { key: 'importMpp', icon: <FileOutlined />, label: '导入 MPP 替换当前工程…' },
  ]
  const exportMenuItems: MenuProps['items'] = [
    { key: 'exportXml', icon: <ExportOutlined />, label: '导出 MS Project XML' },
    { key: 'exportXlsx', icon: <FileExcelOutlined />, label: '导出 Excel' },
    { key: 'exportArt', icon: <FileImageOutlined />, label: '导出图面（PNG / PDF / 打印）…' },
  ]
  const MENU_COMMAND_ID: Record<string, string> = {
    sample: 'help.sample',
    importNew: 'file.import-new',
    importXml: 'file.import-xml',
    importXlsx: 'file.import-xlsx',
    importMpp: 'file.import-mpp',
    exportXml: 'file.export-xml',
    exportXlsx: 'file.export-xlsx',
    exportArt: 'file.export-art',
    manage: 'file.manage',
    undo: 'edit.undo',
    redo: 'edit.redo',
    add: 'task.add',
    group: 'task.group',
    milestone: 'task.milestone',
    inspector: 'task.inspector',
    del: 'edit.delete',
    clear: 'edit.clear',
    gantt: 'view.gantt',
    pdm: 'view.pdm',
    aoa: 'view.aoa',
    usage: 'view.usage',
    dcma: 'view.dcma',
    monte: 'view.monte',
    evm: 'view.evm',
  }
  const runMenu = (key: string) => {
    const commandId = MENU_COMMAND_ID[key]
    if (commandId) runCommand(commandId)
  }
  const fileMenuCfg: MenuProps = {
    items: [
      { key: 'sample', icon: <ThunderboltOutlined />, label: '载入示例工程' },
      { type: 'divider' },
      ...importMenuItems,
      { type: 'divider' },
      ...exportMenuItems,
      { type: 'divider' },
      { key: 'manage', icon: <SettingOutlined />, label: '工程管理…' },
    ],
    onClick: ({ key }) => runMenu(key),
  }
  const editMenuCfg: MenuProps = {
    items: [
      { key: 'undo', icon: <UndoOutlined />, label: '撤销（Ctrl+Z）', disabled: !!commandReason('edit.undo') },
      { key: 'redo', icon: <RedoOutlined />, label: '重做（Ctrl+Y）', disabled: !!commandReason('edit.redo') },
      { type: 'divider' },
      { key: 'del', icon: <DeleteOutlined />, label: '删除选中', danger: true, disabled: !!commandReason('edit.delete') },
      { key: 'clear', icon: <ClearOutlined />, label: '清空全部任务与搭接', danger: true },
    ],
    onClick: ({ key }) => runMenu(key),
  }
  const VIEW_MENU: { key: typeof view; label: string; icon: React.ReactNode }[] = [
    { key: 'gantt', label: '横道图', icon: <TableOutlined /> },
    { key: 'pdm', label: '单代号网络图', icon: <NodeIndexOutlined /> },
    { key: 'aoa', label: '双代号网络图', icon: <PartitionOutlined /> },
    { key: 'usage', label: '资源使用', icon: <TeamOutlined /> },
    { key: 'dcma', label: '质量体检', icon: <SafetyOutlined /> },
    { key: 'monte', label: '工期模拟', icon: <FundOutlined /> },
    { key: 'evm', label: '挣值分析', icon: <RiseOutlined /> },
  ]
  const viewMenuCfg: MenuProps = {
    items: VIEW_MENU.map((v) => ({
      key: v.key,
      icon: view === v.key ? <CheckOutlined /> : <span className="sched-menu-hold" />,
      label: v.label,
    })),
    onClick: ({ key }) => runMenu(key),
  }
  const importMenuCfg: MenuProps = { items: importMenuItems, onClick: ({ key }) => runMenu(key) }
  const exportMenuCfg: MenuProps = { items: exportMenuItems, onClick: ({ key }) => runMenu(key) }

  return (
    <div className="sched-shell">
      {!chat.collapsed && (
        <div className="sched-chat-pane" style={{ width: chat.width }}>
          <ScheduleChatPane onCollapse={toggleChat} />
        </div>
      )}
      {!chat.collapsed && (
        <div
          className="sched-chat-divider"
          title="拖动调宽 · 双击复位"
          onMouseDown={onDividerMouseDown}
          onDoubleClick={resetChatWidth}
        />
      )}
      <div className={["sched-page", presetClass(displayPreset)].filter(Boolean).join(' ')} data-testid="sched-page" style={{ flex: 1, minWidth: 0 }}>
      {/* ── ZPert 式命令台：菜单行 + 命令面板行 + 文档属性行 ── */}
      <div className="sched-command-deck" data-testid="sched-command-deck">
      <div className="sched-menubar sched-ribbon-tabbar" data-testid="sched-menubar">
        <span className="sched-app-mark" aria-hidden="true">G</span>
        <div className="sched-ribbon-tablist" role="tablist" aria-label="进度计划功能区">
          {SCHEDULE_RIBBON_TABS.map((tab) => (
            <button
              key={tab.key}
              type="button"
              role="tab"
              aria-selected={ribbonTab === tab.key}
              tabIndex={ribbonTab === tab.key ? 0 : -1}
              ref={(el) => {
                if (el) ribbonTabRefs.current.set(tab.key, el)
                else ribbonTabRefs.current.delete(tab.key)
              }}
              onKeyDown={(e) => onRibbonTabKeyDown(e, tab.key)}
              className={`sched-menu-item${ribbonTab === tab.key ? ' is-active' : ''}`}
              onClick={() => setRibbonTab(tab.key)}
              data-testid={`sched-ribbon-tab-${tab.key}`}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <button
          type="button"
          className="sched-command-search-trigger"
          title="搜索命令与快捷键（Ctrl+K）"
          aria-label="搜索命令与快捷键"
          aria-keyshortcuts="Control+K"
          data-testid="sched-command-trigger"
          onClick={() => setCommandOpen(true)}
        >
          <SearchOutlined aria-hidden="true" />
          <span>命令</span>
          <kbd>Ctrl K</kbd>
        </button>
      </div>

      <input
        ref={fileRef}
        type="file"
        accept=".xml,application/xml,text/xml"
        style={{ display: 'none' }}
        onChange={(e) => {
          const f = e.target.files?.[0]
          e.target.value = ''
          if (f) void onImportFile(f)
        }}
      />
      <input
        ref={xlsxRef}
        type="file"
        accept=".xlsx"
        style={{ display: 'none' }}
        onChange={(e) => {
          const f = e.target.files?.[0]
          e.target.value = ''
          if (f) void onImportXlsx(f)
        }}
      />
      <input
        ref={mppRef}
        type="file"
        accept=".mpp"
        style={{ display: 'none' }}
        onChange={(e) => {
          const f = e.target.files?.[0]
          e.target.value = ''
          if (f) void onImportMpp(f)
        }}
      />
      <input
        ref={newRef}
        type="file"
        accept=".xml,.xlsx,.xlsm,.mpp"
        style={{ display: 'none' }}
        onChange={(e) => {
          const f = e.target.files?.[0]
          e.target.value = ''
          if (f) void onImportNewFile(f)
        }}
      />

      {/* ── 工具栏（行2）：高频命令快捷区，分组竖线分隔（ribbon 感）；
          导入/导出收拢为下拉，MPP（v4.140）随 XML/Excel 同列 ── */}
      {ribbonTab === 'plan' && (
      <div className="sched-toolbar">
        <RibbonGroup label="视图">
          <RibbonCommand big icon={<PartitionOutlined />} label="时标网络图" active={commandActive['view.timescaled']}
            title={commandTitle('view.timescaled')} onClick={() => runCommand('view.timescaled')} />
          <RibbonCommand big icon={<TableOutlined />} label="横道图" active={commandActive['view.gantt']}
            title={commandTitle('view.gantt')} onClick={() => runCommand('view.gantt')} />
          <RibbonCommand big icon={<NodeIndexOutlined />} label="逻辑网络图" active={commandActive['view.logic']}
            title={commandTitle('view.logic')} onClick={() => runCommand('view.logic')} />
          <RibbonCommand big icon={<NodeIndexOutlined />} label="单代号网络图" active={commandActive['view.pdm']}
            title={commandTitle('view.pdm')} onClick={() => runCommand('view.pdm')} />
          <RibbonColumn>
            <RibbonCommand icon={<NodeIndexOutlined />} label="时标逻辑图" active={commandActive['view.timescaledLogic']}
              title={commandTitle('view.timescaledLogic')} onClick={() => runCommand('view.timescaledLogic')} />
            <RibbonCommand icon={<PartitionOutlined />} label="横道网络图" active={commandActive['view.ganttNetwork']}
              title={commandTitle('view.ganttNetwork')} onClick={() => runCommand('view.ganttNetwork')} />
          </RibbonColumn>
          <RibbonColumn>
            <Popover
              trigger="click"
              placement="bottomLeft"
              open={saveViewOpen}
              onOpenChange={(o) => { setSaveViewOpen(o); if (!o) { setSaveViewErr(null); setSaveViewName('') } }}
              content={
                <div style={{ width: 240, display: 'flex', flexDirection: 'column', gap: 8 }}>
                  <Input
                    size="small"
                    placeholder="视图名称"
                    value={saveViewName}
                    maxLength={30}
                    onChange={(e) => { setSaveViewName(e.target.value); setSaveViewErr(null) }}
                    onPressEnter={commitSaveView}
                    data-testid="sched-view-save-input"
                  />
                  {saveViewErr && <span style={{ color: 'var(--md-sys-color-error)', fontSize: 12 }}>{saveViewErr}</span>}
                  <Button size="small" type="primary" data-testid="sched-view-save-confirm" onClick={commitSaveView}>保存当前视图</Button>
                  <span className="sched-dim" style={{ fontSize: 12 }}>快照=当前列显隐、分组、排序与筛选档（斑马 ViewDB：视图=列可见性预设）</span>
                </div>
              }
              title="保存视图"
            >
              <RibbonCommand icon={<CopyOutlined />} label="保存视图" title="把当前横道列集/分组/排序/筛选存为命名视图" ariaExpanded={saveViewOpen} />
            </Popover>
            <Popover
              trigger="click"
              placement="bottomLeft"
              open={manageViewOpen}
              onOpenChange={setManageViewOpen}
              content={
                <div style={{ width: 260, display: 'flex', flexDirection: 'column', gap: 4 }} data-testid="sched-view-manage">
                  {customViews.length === 0 && <span className="sched-dim" style={{ fontSize: 12 }}>还没有自定义视图——先用「保存视图」把当前视图态存一条。</span>}
                  {customViews.map((v: SchedView) => (
                    <div key={v.id} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      {renameViewId === v.id ? (
                        <>
                          <Input
                            size="small"
                            value={renameViewName}
                            maxLength={30}
                            autoFocus
                            onChange={(e) => { setRenameViewName(e.target.value); setRenameViewErr(null) }}
                            onPressEnter={() => {
                              const res = renameCustomView(v.id, renameViewName)
                              if (!res.ok) { setRenameViewErr(res.reason ?? '改名失败'); return }
                              setRenameViewId(null)
                              setRenameViewErr(null)
                            }}
                          />
                          <Button size="small" type="link" onClick={() => {
                            const res = renameCustomView(v.id, renameViewName)
                            if (!res.ok) { setRenameViewErr(res.reason ?? '改名失败'); return }
                            setRenameViewId(null)
                            setRenameViewErr(null)
                          }}>存</Button>
                        </>
                      ) : (
                        <>
                          <Button size="small" type="link" style={{ padding: 0 }} onClick={() => { requestCustomView(v.id); setManageViewOpen(false) }}>
                            {v.name}
                          </Button>
                          <span style={{ flex: 1 }} />
                          <Button size="small" type="link" onClick={() => { setRenameViewId(v.id); setRenameViewName(v.name); setRenameViewErr(null) }}>改名</Button>
                          <Button size="small" type="link" danger onClick={() => removeCustomView(v.id)}>删除</Button>
                        </>
                      )}
                    </div>
                  ))}
                  {renameViewErr && <span style={{ color: 'var(--md-sys-color-error)', fontSize: 12 }}>{renameViewErr}</span>}
                </div>
              }
              title="视图管理"
            >
              <RibbonCommand icon={<AppstoreOutlined />} label="视图管理" title="应用/改名/删除已保存的视图" ariaExpanded={manageViewOpen} />
            </Popover>
            <Popover trigger="click" placement="bottomLeft" open={groupOpen} onOpenChange={setGroupOpen}
              content={
                <div style={{ width: 220, display: 'flex', flexDirection: 'column', gap: 4 }} data-testid="sched-group-pick">
                  {GANTT_GROUP_OPTIONS.map((o) => (
                    <label key={o.value} style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13 }}>
                      <input
                        type="checkbox"
                        checked={groupPick.includes(o.value)}
                        onChange={(e) => {
                          const next = e.target.checked ? [...groupPick, o.value] : groupPick.filter((g) => g !== o.value)
                          setGroupPick(next.slice(0, MAX_GANTT_GROUP_DEPTH))
                          sendGanttCommand('set-group', JSON.stringify(next.slice(0, MAX_GANTT_GROUP_DEPTH)))
                        }}
                      />
                      {o.label}
                    </label>
                  ))}
                  <span className="sched-dim" style={{ fontSize: 12 }}>多级分组至多三级（斑马 groupby_setting 口径）</span>
                </div>
              } title="分组">
              <RibbonCommand icon={<ClusterOutlined />} label="分组…" title="按维度分组横道表格（工期/总时差/负责人/里程碑/关键/状态/资源…14 维）" ariaExpanded={groupOpen} />
            </Popover>
          </RibbonColumn>
        </RibbonGroup>
        <RibbonGroup label="编辑">
          <RibbonColumn>
            <RibbonCommand icon={<CopyOutlined />} label="复制" disabled={!!commandReason('edit.copy')}
              title={commandTitle('edit.copy')} onClick={() => runCommand('edit.copy')} />
            <RibbonCommand icon={<SnippetsOutlined />} label="粘贴" disabled={!!commandReason('edit.paste')}
              title={commandTitle('edit.paste')} onClick={() => runCommand('edit.paste')} />
            <RibbonCommand icon={<DeleteOutlined />} label="删除" disabled={!!commandReason('edit.delete')}
              title={commandTitle('edit.delete')} onClick={() => runCommand('edit.delete')} />
          </RibbonColumn>
          <RibbonColumn>
            <Popover trigger="click" placement="bottomLeft" open={findOpen} onOpenChange={setFindOpen}
              content={
                <div style={{ width: 220, display: 'flex', flexDirection: 'column', gap: 8 }}>
                  <Input size="small" placeholder="按名称过滤（空=全部）" value={findText} allowClear
                    onChange={(e) => { setFindText(e.target.value); sendGanttCommand('find', e.target.value) }}
                    data-testid="sched-find-input" />
                  <Button size="small" onClick={() => { setFindText(''); sendGanttCommand('find', ''); setFindOpen(false) }}>清除过滤</Button>
                </div>
              } title="查找">
              <RibbonCommand icon={<SearchOutlined />} label="查找" title={commandTitle('edit.find')} ariaExpanded={findOpen} onClick={() => runCommand('edit.find')} />
            </Popover>
            <RibbonCommand icon={<FormatPainterOutlined />} label="格式刷" disabled={!selected}
              title={selected ? '取选中工作外观，再点击目标行应用（应用后自动收起）' : '请先选中一条工作'}
              onClick={() => { setView('gantt'); sendGanttCommand('format-brush') }} />
            <Popover trigger="click" placement="bottomLeft" open={colFilterOpen} onOpenChange={setColFilterOpen}
              title="列筛选"
              content={
                <div style={{ width: 240, display: 'flex', flexDirection: 'column', gap: 8 }}>
                  <span className="sched-dim" style={{ fontSize: 12 }}>
                    在表格列头点「▼」按列类型筛选：文本包含 / 数值比较 / 日期区间 / 枚举；条件为会话态，不进工程文件。
                  </span>
                  <Button size="small" data-testid="sched-colfilter-clear"
                    onClick={() => { sendGanttCommand('column-filter-clear'); setColFilterOpen(false) }}>清除全部列筛选</Button>
                </div>
              }>
              <RibbonCommand icon={<FilterOutlined />} label="列筛选" title={commandTitle('edit.column-filter')} ariaExpanded={colFilterOpen} onClick={() => runCommand('edit.column-filter')} />
            </Popover>
            <Popover trigger="click" placement="bottomLeft" title="工作分类"
              content={
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8, width: 200 }}>
                  <Input size="small" placeholder="工作分类（如：主体/机电/装饰）" defaultValue={selected?.category ?? ''}
                    key={selected?.id ?? 'none'}
                    onPressEnter={(e) => {
                      const v = (e.target as HTMLInputElement).value.trim()
                      if (selected) updateTask(selected.id, { category: v || undefined })
                    }}
                    data-testid="sched-category-input"
                  />
                  <span className="sched-dim" style={{ fontSize: 12 }}>回车提交；「按工作分类分组」优先取此字段，未设置回落分区名</span>
                </div>
              }>
              <RibbonCommand icon={<TagOutlined />} label="分类" disabled={!selected} title="设置选中工作的工作分类（分组维度可用）" />
            </Popover>
          </RibbonColumn>
        </RibbonGroup>
        <RibbonGroup label="格式">
          <RibbonColumn>
            <Popover trigger="click" placement="bottomLeft" title="字体"
              content={
                <div style={{ display: 'flex', flexDirection: 'column', gap: 4, width: 140 }}>
                  {STYLE_FONTS.map((f) => (
                    <Button key={f} size="small" type="link" style={{ padding: 0, justifyContent: 'flex-start' }}
                      onClick={() => patchStyle({ font: f === '默认' ? undefined : f })}>{f}</Button>
                  ))}
                </div>
              }>
              <RibbonCommand icon={<FontColorsOutlined />} label="字体" disabled={!selected} title="设置选中工作名称字体族" />
            </Popover>
            <Popover trigger="click" placement="bottomLeft" title="字号"
              content={
                <div style={{ display: 'flex', flexDirection: 'column', gap: 4, width: 100 }}>
                  {STYLE_SIZES.map((n) => (
                    <Button key={n} size="small" type="link" style={{ padding: 0, justifyContent: 'flex-start' }}
                      onClick={() => patchStyle({ size: n })}>{n} px</Button>
                  ))}
                </div>
              }>
              <RibbonCommand icon={<FontSizeOutlined />} label="字号" disabled={!selected} title="设置选中工作名称字号（8-72px）" />
            </Popover>
            <Popover trigger="click" placement="bottomLeft" title="字体颜色"
              content={<div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}><span className="sched-dim" style={{ fontSize: 12 }}>作用于选中工作名称</span>{swatchRow('color')}</div>}>
              <RibbonCommand icon={<FontColorsOutlined />} label="字体颜色" disabled={!selected} title="设置选中工作文字颜色" />
            </Popover>
          </RibbonColumn>
          <RibbonColumn>
            <Popover trigger="click" placement="bottomLeft" title="横道填充"
              content={<div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}><span className="sched-dim" style={{ fontSize: 12 }}>作用于选中工作横道条</span>{swatchRow('fill')}</div>}>
              <RibbonCommand icon={<BgColorsOutlined />} label="填充" disabled={!selected} title="设置选中工作横道条填充色" />
            </Popover>
            <Popover
              trigger="click"
              placement="bottomLeft"
              open={styleOpen}
              onOpenChange={setStyleOpen}
              title="图面样式"
              content={
                <ScheduleNetworkStyleControls
                  style={project.networkStyle}
                  networkView={view === 'aoa' || view === 'pdm'}
                  onChange={setNetworkStyle}
                />
              }
            >
              <RibbonCommand
                icon={<LineHeightOutlined />}
                label="图面样式"
                title="节点时间/工期/六参数/起止时间四开关，以及节点与箭线样式"
                ariaExpanded={styleOpen}
                testId="sched-network-style-trigger"
              />
            </Popover>
          </RibbonColumn>
        </RibbonGroup>
        <RibbonGroup label="大纲">
          <RibbonColumn>
            <RibbonCommand icon={<NodeCollapseOutlined />} label="折叠" disabled={!!commandReason('outline.collapse')}
              title={commandTitle('outline.collapse')} onClick={() => runCommand('outline.collapse')} />
            <RibbonCommand icon={<NodeExpandOutlined />} label="升级" disabled={!!commandReason('outline.promote')}
              title={commandTitle('outline.promote')} onClick={() => runCommand('outline.promote')} />
            <RibbonCommand icon={<NodeCollapseOutlined />} label="降级" disabled={!!commandReason('outline.demote')}
              title={commandTitle('outline.demote')} onClick={() => runCommand('outline.demote')} />
          </RibbonColumn>
          <RibbonColumn>
            <RibbonCommand icon={<UpOutlined />} label="上移" disabled={!!commandReason('outline.up')}
              title={commandTitle('outline.up')} onClick={() => runCommand('outline.up')} />
            <RibbonCommand icon={<DownOutlined />} label="下移" disabled={!!commandReason('outline.down')}
              title={commandTitle('outline.down')} onClick={() => runCommand('outline.down')} />
            <RibbonCommand icon={<SearchOutlined />} label="上钻" title="收起全部分组（只看分区行）"
              onClick={() => { setView('gantt'); sendGanttCommand('collapse-all') }} />
          </RibbonColumn>
        </RibbonGroup>
        <RibbonGroup label="插入">
          <RibbonCommand big icon={<PlusOutlined />} label="工作" title={commandTitle('task.add')} onClick={() => runCommand('task.add')} />
          <Popover trigger="click" placement="bottomLeft" open={relationOpen} onOpenChange={setRelationOpen} title={`「${selected?.name ?? ''}」前置任务`}
            content={selected ? <PredEditor task={selected} /> : <span className="sched-dim">先选中一个任务</span>}>
            <RibbonCommand big icon={<ToolOutlined />} label="关系" disabled={!!commandReason('task.relation')}
              title={commandTitle('task.relation')} ariaExpanded={relationOpen} onClick={() => runCommand('task.relation')} />
          </Popover>
          <RibbonCommand
            big
            icon={<AimOutlined />}
            label={selected?.isMilestone ? '取消里程碑' : '里程碑'}
            disabled={!!commandReason('task.milestone')}
            active={!!selected?.isMilestone}
            title={commandTitle('task.milestone')}
            onClick={() => runCommand('task.milestone')}
          />
        </RibbonGroup>
        <RibbonGroup label="高级">
          <RibbonColumn>
            <RibbonCommand icon={<PartitionOutlined />} label="智能布图" title={commandTitle('display.smart-layout')}
              onClick={() => runCommand('display.smart-layout')} />
            <RibbonCommand icon={<FileImageOutlined />} label="按纸张" disabled title="预留：按纸张布图" />
            <Popover trigger="click" placement="bottom" content={<ProjectSettingsPanel cpm={cpm} />} title="工程设置">
              <button type="button" className="sched-ribbon-command is-small" data-testid="sched-project-settings-btn">
                <SettingOutlined /><span>设置</span>
              </button>
            </Popover>
          </RibbonColumn>
          <RibbonColumn>
            <RibbonCommand icon={<WarningOutlined />} label="预警" active={commandActive['display.dock']}
              title={commandTitle('display.dock')} onClick={() => runCommand('display.dock')} />
            <RibbonCommand icon={<SafetyOutlined />} label="检查" active={commandActive['display.dock']}
              title={commandTitle('display.dock')} onClick={() => runCommand('display.dock')} testId="sched-dock-toggle" />
          </RibbonColumn>
        </RibbonGroup>
      </div>
      )}

      {ribbonTab === 'file' && (
        <div className="sched-toolbar">
          <RibbonGroup label="文件">
            <RibbonColumn>
              <Dropdown trigger={['click']} menu={fileMenuCfg}>
                <button type="button" className="sched-ribbon-command is-small" data-testid="sched-menu-file">
                  <FileOutlined /><span>文件菜单</span>
                </button>
              </Dropdown>
              <Dropdown trigger={['click']} menu={importMenuCfg}>
                <button type="button" className="sched-ribbon-command is-small" data-testid="sched-import-btn" disabled={xlsxBusy || mppBusy || xmlBusy}>
                  <ImportOutlined /><span>导入</span>
                </button>
              </Dropdown>
              <Dropdown trigger={['click']} menu={exportMenuCfg}>
                <button type="button" className="sched-ribbon-command is-small" data-testid="sched-export-menu-btn">
                  <ExportOutlined /><span>导出</span>
                </button>
              </Dropdown>
            </RibbonColumn>
          </RibbonGroup>
          <RibbonGroup label="工程">
            {projectList.length > 0 && (
              <Select
                size="small"
                style={{ width: 150 }}
                value={currentPath}
                loading={projBusy}
                disabled={projectList.length <= 1}
                data-testid="sched-project-select"
                options={projectList.filter((p) => !p.archived).map((p) => ({ value: p.rel, label: p.name || p.rel }))}
                onChange={(rel: string) => {
                  setProjBusy(true)
                  void Promise.resolve(openProject(rel)).finally(() => setProjBusy(false))
                }}
              />
            )}
            <Popover
              trigger="click"
              placement="bottomRight"
              open={manageOpen}
              onOpenChange={(open) => { setManageOpen(open); if (open) void Promise.resolve(refreshProjects?.()) }}
              content={<ProjectsManagePanel />}
              title="工程管理"
            >
              <button type="button" className="sched-ribbon-command is-big" data-testid="sched-projects-manage">
                <SettingOutlined /><span>工程管理</span>
              </button>
            </Popover>
            <RibbonCommand big icon={<ToolOutlined />} label="办公预览" onClick={openInOffice} />
            <RibbonCommand big icon={<MessageOutlined />} label="AI 助手" active={commandActive['display.chat']}
              title={commandTitle('display.chat')} onClick={() => runCommand('display.chat')} />
          </RibbonGroup>
          <RibbonGroup label="历史">
            <RibbonColumn>
              <Dropdown trigger={['click']} menu={editMenuCfg}>
                <button type="button" className="sched-ribbon-command is-small" data-testid="sched-menu-edit">
                  <UndoOutlined /><span>编辑菜单</span>
                </button>
              </Dropdown>
              <RibbonCommand icon={<UndoOutlined />} label="撤销" disabled={!!commandReason('edit.undo')}
                title={commandTitle('edit.undo')} onClick={() => runCommand('edit.undo')} testId="sched-undo" />
              <RibbonCommand icon={<RedoOutlined />} label="重做" disabled={!!commandReason('edit.redo')}
                title={commandTitle('edit.redo')} onClick={() => runCommand('edit.redo')} testId="sched-redo" />
            </RibbonColumn>
          </RibbonGroup>
        </div>
      )}

      {outlineMsg && (
        <Alert type="warning" showIcon message={outlineMsg} closable onClose={() => setOutlineMsg(null)}
          style={{ margin: '4px 12px 0' }} data-testid="sched-outline-msg" />
      )}
      {ribbonTab === 'resource' && (
        <div className="sched-toolbar">
          <RibbonGroup label="资源">
            <Popover trigger="click" placement="bottom" content={<ResourcePanel cpm={cpm} />} title="资源与成本">
              <button type="button" className="sched-ribbon-command is-big" data-testid="sched-resource-btn">
                <TeamOutlined /><span>资源工作表</span>
              </button>
            </Popover>
            <RibbonCommand big icon={<TableOutlined />} label="资源使用" active={commandActive['view.usage']}
              title={commandTitle('view.usage')} onClick={() => runCommand('view.usage')} />
          </RibbonGroup>
          <RibbonGroup label="成本">
            <RibbonCommand big icon={<ExportOutlined />} label="导出 Excel" title={commandTitle('file.export-xlsx')}
              onClick={() => runCommand('file.export-xlsx')} />
            <RibbonCommand big icon={<FileImageOutlined />} label="图面导出" title={commandTitle('file.export-art')}
              onClick={() => runCommand('file.export-art')} />
          </RibbonGroup>
          <RibbonGroup label="工程">
            <RibbonCommand icon={<CalendarOutlined />} label="工作日历" onClick={() => undefined} title="工作日历在下方工程信息条打开" />
            <RibbonCommand icon={<FileTextOutlined />} label="当前计划" onClick={() => setRibbonTab('plan')} />
          </RibbonGroup>
        </div>
      )}

      {ribbonTab === 'control' && (
        <div className="sched-toolbar">
          <RibbonGroup label="检视">
            <Popover trigger="click" placement="bottom" content={<ReviewPanel cpm={cpm} />} title="检视（多期进度快照）">
              <button type="button" className="sched-ribbon-command is-big">
                <FieldTimeOutlined /><span>检视时间</span>
              </button>
            </Popover>
            <Popover trigger="click" placement="bottom" content={<BaselinesPanel cpm={cpm} />} title="基线对比">
              <button type="button" className="sched-ribbon-command is-big">
                <FundOutlined /><span>基线</span>
              </button>
            </Popover>
          </RibbonGroup>
          <RibbonGroup label="检查">
            <RibbonCommand big icon={<SafetyOutlined />} label="计划检查" active={commandActive['display.dock']}
              title={commandTitle('display.dock')} onClick={() => runCommand('display.dock')} testId="sched-dock-toggle" />
            <RibbonCommand big icon={<SafetyOutlined />} label="质量体检" active={commandActive['view.dcma']}
              title={commandTitle('view.dcma')} onClick={() => runCommand('view.dcma')} />
          </RibbonGroup>
          <RibbonGroup label="分析">
            <RibbonCommand icon={<FundOutlined />} label="工期模拟" active={commandActive['view.monte']}
              title={commandTitle('view.monte')} onClick={() => runCommand('view.monte')} />
            <RibbonCommand icon={<RiseOutlined />} label="挣值分析" active={commandActive['view.evm']}
              title={commandTitle('view.evm')} onClick={() => runCommand('view.evm')} />
          </RibbonGroup>
        </div>
      )}

      {ribbonTab === 'collab' && (
        <div className="sched-toolbar">
          <RibbonGroup label="协作">
            <RibbonCommand big icon={<MessageOutlined />} label={chat.collapsed ? '展开 AI' : '收起 AI'}
              active={commandActive['display.chat']} title={commandTitle('display.chat')} onClick={() => runCommand('display.chat')} />
            <RibbonCommand big icon={<ToolOutlined />} label="办公预览" onClick={openInOffice} />
          </RibbonGroup>
          <RibbonGroup label="工程">
            <RibbonCommand big icon={<SettingOutlined />} label="工程管理" title={commandTitle('file.manage')}
              onClick={() => runCommand('file.manage')} />
            <RibbonCommand big icon={<FileTextOutlined />} label="切换到计划" onClick={() => setRibbonTab('plan')} />
          </RibbonGroup>
        </div>
      )}

      {ribbonTab === 'view' && (
        <div className="sched-toolbar">
          <RibbonGroup label="视图库">
            <Dropdown trigger={['click']} menu={viewMenuCfg}>
              <button type="button" className="sched-ribbon-command is-big" data-testid="sched-menu-view">
                <TableOutlined /><span>视图列表</span>
              </button>
            </Dropdown>
          </RibbonGroup>
          <RibbonGroup label="图模式">
            <RibbonCommand big icon={<TableOutlined />} label="横道图" active={commandActive['view.gantt']}
              title={commandTitle('view.gantt')} onClick={() => runCommand('view.gantt')} />
            <RibbonCommand big icon={<NodeIndexOutlined />} label="单代号" active={commandActive['view.pdm']}
              title={commandTitle('view.pdm')} onClick={() => runCommand('view.pdm')} />
            <RibbonCommand big icon={<PartitionOutlined />} label="双代号" active={commandActive['view.aoa']}
              title={commandTitle('view.aoa')} onClick={() => runCommand('view.aoa')} />
            <RibbonCommand big icon={<TeamOutlined />} label="资源使用" active={commandActive['view.usage']}
              title={commandTitle('view.usage')} onClick={() => runCommand('view.usage')} />
          </RibbonGroup>
          <RibbonGroup label="分析视图">
            <RibbonColumn>
              <RibbonCommand icon={<SafetyOutlined />} label="质量体检" active={commandActive['view.dcma']}
                title={commandTitle('view.dcma')} onClick={() => runCommand('view.dcma')} />
              <RibbonCommand icon={<FundOutlined />} label="工期模拟" active={commandActive['view.monte']}
                title={commandTitle('view.monte')} onClick={() => runCommand('view.monte')} />
              <RibbonCommand icon={<RiseOutlined />} label="挣值分析" active={commandActive['view.evm']}
                title={commandTitle('view.evm')} onClick={() => runCommand('view.evm')} />
            </RibbonColumn>
          </RibbonGroup>
          <RibbonGroup label="显示">
            <RibbonCommand icon={<SafetyOutlined />} label="检查面板" active={commandActive['display.dock']}
              title={commandTitle('display.dock')} onClick={() => runCommand('display.dock')} />
            <RibbonCommand icon={<MessageOutlined />} label="AI 助手" active={commandActive['display.chat']}
              title={commandTitle('display.chat')} onClick={() => runCommand('display.chat')} />
            <label className="sched-preset-picker" title="显示预设：只换 --sched-* 领域色，不换主题令牌">
              <span className="sched-dim">配色</span>
              <Select
                size="small"
                value={displayPreset}
                onChange={(v) => changeDisplayPreset(v as DisplayPreset)}
                options={DISPLAY_PRESETS.map((p) => ({ value: p.id, label: p.label, title: p.desc }))}
                data-testid="sched-display-preset"
              />
            </label>
          </RibbonGroup>
        </div>
      )}

      {ribbonTab === 'help' && (
        <div className="sched-toolbar">
          <RibbonGroup label="起步">
            <Popover trigger="click" placement="bottom" open={templatesOpen} onOpenChange={setTemplatesOpen}
              content={<TemplatesPanel cpm={cpm} />} title="项目模板">
              <button type="button" className="sched-ribbon-command is-big" data-testid="sched-templates-btn" aria-expanded={templatesOpen}>
                <AppstoreOutlined /><span>项目模板</span>
              </button>
            </Popover>
            <RibbonCommand big icon={<SearchOutlined />} label="命令与快捷键" title="搜索全部命令（Ctrl+K / F1）"
              onClick={() => runCommand('help.commands')} testId="sched-shortcuts-help" />
            <RibbonCommand big icon={<ThunderboltOutlined />} label="示例工程" title={commandTitle('help.sample')}
              onClick={() => runCommand('help.sample')} />
          </RibbonGroup>
          <RibbonGroup label="文件">
            <Dropdown trigger={['click']} menu={importMenuCfg}>
              <button type="button" className="sched-ribbon-command is-big">
                <ImportOutlined /><span>导入指南</span>
              </button>
            </Dropdown>
            <RibbonCommand big icon={<ToolOutlined />} label="办公说明" onClick={openInOffice} />
          </RibbonGroup>
        </div>
      )}

      </div>

      <div className="sched-docbar" data-testid="sched-docbar">
        {(projectList.length > 0 ? projectList.filter((p) => !p.archived) : [{ rel: currentPath ?? SCHEDULE_FILE_PATH, name: project.name }]).map((item) => {
          const active = currentPath ? item.rel === currentPath : item.rel === SCHEDULE_FILE_PATH
          return (
            <button
              key={item.rel}
              type="button"
              className={`sched-doc-tab${active ? ' is-active' : ''}`}
              data-path={item.rel}
              onClick={() => {
                if (active) return
                setProjBusy(true)
                void Promise.resolve(openProject(item.rel)).finally(() => setProjBusy(false))
              }}
              aria-label={item.rel}
            >
              <FileTextOutlined />
              <span>{active ? project.name : item.name}</span>
              {active && (
                <span
                  className="sched-doc-tab-dot"
                  style={{ opacity: sync === 'saved' ? 0.45 : 1 }}
                  aria-label={sync === 'saved' ? '已保存' : '有未保存改动'}
                />
              )}
            </button>
          )
        })}
        <Popover
          trigger="click"
          placement="bottomRight"
          open={manageOpen}
          onOpenChange={(open) => { setManageOpen(open); if (open) void Promise.resolve(refreshProjects?.()) }}
          content={<ProjectsManagePanel />}
          title="工程管理"
        >
          <button type="button" className="sched-doc-add" data-testid="sched-projects-manage" title="工程管理：新建 / 改名 / 归档 / 删除 / 载入" aria-label="工程管理">
            <PlusOutlined />
          </button>
        </Popover>
        <div style={{ flex: 1 }} />
        {projBusy && <span className="sched-dim">切换工程…</span>}
      </div>

      {importMsg && (
        <Alert
          type={importMsg.type}
          showIcon
          closable
          message={importMsg.text}
          onClose={() => setImportMsg(null)}
        />
      )}
      <ScheduleCommandPalette
        open={commandOpen}
        commands={runtimeCommands}
        onClose={() => setCommandOpen(false)}
        onBlocked={setOutlineMsg}
      />
      <ExportDialog open={exportOpen} onClose={() => setExportOpen(false)} project={project} cpm={cpm} aoa={aoa} preset={displayPreset} defaultView={view === 'usage' || view === 'dcma' || view === 'monte' || view === 'evm' ? 'gantt' : view} />
      {!cpm.ok && cpm.error && (
        <Alert type="error" showIcon message={cpm.error} description="请修正搭接关系后重试；网络图视图在循环解除前不可用。" />
      )}

      <div className="sched-workspace" data-testid="sched-workspace">
        <div className="sched-workspace-main">
          {view === 'dcma' && <DcmaView project={project} cpm={cpm} />}
          {view === 'monte' && <MonteView project={project} cpm={cpm} />}
          {view === 'evm' && <EvmView project={project} cpm={cpm} />}
          {view === 'gantt' && <GanttView project={project} cpm={cpm} onInspect={setInspectId} showToolbar={false} />}
          {(view === 'pdm' || view === 'aoa') && <ScheduleNetworkWorkspace view={view} project={project} cpm={cpm} graph={aoa} renderMode={netMode} />}
          {view === 'usage' && <UsageView usage={usage} />}
        </div>
        <ScheduleDock
          open={dockOpen}
          warn={warn}
          onNavigate={(id) => { setInspectId(id); select(id) }}
          onClose={() => setDockOpen(false)}
        />
      </div>

      {/* 任务检查器（v4.136）：该任务日期由哪条搭接决定 + 前驱/后继可点击跳转；
          what-if 推演（v4.523）经 slot 挂进抽屉——计算与 delta 交互态都在
          WhatIfPanel，检查器保持零排程计算契约 */}
      <TaskInspector
        open={!!inspectTask && inspectTask.level > 0 && !!inspectorData}
        task={inspectTask && inspectTask.level > 0 ? inspectTask : null}
        row={inspectTask ? cpm.rows[inspectTask.id] ?? null : null}
        wbs={inspectTask ? wbsOf(project.tasks)[project.tasks.findIndex((t) => t.id === inspectTask.id)] : undefined}
        predecessors={inspectorData?.predecessors ?? []}
        successors={inspectorData?.successors ?? []}
        startDate={project.startDate}
        calendar={project.calendar}
        whatIf={inspectTask && inspectTask.level > 0 ? (
          <WhatIfPanel project={project} taskId={inspectTask.id} />
        ) : undefined}
        diagnostics={inspectDiags}
        warn={inspectWarn}
        warnAlgorithm={warnAlgo}
        onWarnAlgorithmChange={setWarnAlgo}
        onPatch={(patch) => { if (inspectTask) updateTask(inspectTask.id, patch) }}
        onClose={() => setInspectId(null)}
        onNavigate={(id) => { setInspectId(id); select(id) }}
      />

      {/* 底部状态栏（斑马口径：共 N 项工作总工期 N 天 + 关键/工作制常驻）
          W6 收尾：挂 `data-hit="status-bar"` + 右键统一入口（文案/类名/结构不动）
          W6 键盘/语义：`role="status"` + `aria-live="polite"`——**只播变化的那一格**：
          `aria-atomic="false"` 显式覆盖 role=status 的默认整段重播，避免每次计数变动
          都把整条状态栏念一遍（「不逐字播」）。右侧保存指示器是高频文案，单独
          `aria-live="off"`：它每改一个字都不该打断读屏。 */}
      <div
        className="sched-statusbar"
        data-hit="status-bar"
        role="status"
        aria-live="polite"
        aria-atomic="false"
        aria-label="进度计划状态"
        tabIndex={-1}
        ref={statusBarRef}
        onContextMenu={onStatusBarContextMenu}
      >
        <span><span className="sched-sb-strong">{project.tasks.length}</span> 行 · <span className="sched-sb-strong">{project.tasks.filter((t) => t.level === 0).length}</span> 个分组</span>
        <span>工作 <span className="sched-sb-strong">{project.tasks.filter((t) => t.level > 0).length}</span> · 关系 <span className="sched-sb-strong">{project.links.length}</span></span>
        <span>关键工作 <span className="sched-sb-crit">{project.tasks.filter((t) => t.level > 0 && cpm.rows[t.id]?.critical).length}</span> 项</span>
        {warnRed > 0 && (
          <span data-testid="sched-statusbar-warn" title="红灯=突破上级下达的要求完成时间（预警超期算法可在任务检查器切换）">
            预警红灯 <span className="sched-sb-crit">{warnRed}</span> 项
          </span>
        )}
        {showCost && (
          <span data-testid="sched-statusbar-cost" title="总成本 = Σ任务（固定成本 + 分配成本），随工期实时重算（元）">
            总成本 <span className="sched-sb-strong">¥{fmtCost(costs.total)}</span>
          </span>
        )}
        {latest && <span><span className="sched-sb-strong">{latest.date}</span> 检视</span>}
        <span style={{ marginLeft: 'auto' }} aria-live="off" title={syncError ?? currentPath ?? SCHEDULE_FILE_PATH}>
          {!hydrated ? '读取计划…' : sync === 'dirty' ? '改动待保存…'
            : sync === 'saving' ? '保存中…'
            : sync === 'error' ? <span className="sched-sb-crit">保存失败</span>
            : savedAt ? <>已保存 {savedAt} · <span className="sched-dim">agent 可读写</span></> : '已同步 · agent 可读写'}
        </span>
      </div>
      {/* 状态栏右键菜单浮层：条目来自 hitMenu.menuForHit('status-bar', ctx)（同一入口），
          位置跟随命中点（与 GanttView 的 sched-hit-menu-layer 同一套类名/样式） */}
      {statusMenu && (
        <div
          className="sched-hit-menu-layer"
          data-testid="sched-statusbar-hit-menu"
          data-hit-region={statusMenu.region}
          style={{ left: statusMenu.x, top: statusMenu.y }}
          onMouseDown={(e) => e.stopPropagation()}
          onContextMenu={(e) => e.preventDefault()}
        >
          <Menu
            className="sched-hit-menu"
            selectable={false}
            items={statusMenuItems}
            onClick={() => setStatusMenu(null)}
          />
        </div>
      )}
      </div>
    </div>
  )
}

export default SchedulePage


