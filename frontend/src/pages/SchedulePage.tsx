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
import { Alert, Button, Dropdown, Input, Modal, Popconfirm, Popover, Select } from 'antd'
import type { MenuProps } from 'antd'
import {
  AimOutlined, AppstoreOutlined, BgColorsOutlined, CalendarOutlined, CheckOutlined, ClearOutlined, ClusterOutlined, CopyOutlined, DeleteOutlined, DownOutlined, ExportOutlined, FileExcelOutlined, FileImageOutlined, FileOutlined, FileTextOutlined, FontColorsOutlined, FontSizeOutlined, FormatPainterOutlined, FundOutlined, ImportOutlined,
  FieldTimeOutlined, LineHeightOutlined, MessageOutlined, NodeCollapseOutlined, NodeExpandOutlined, NodeIndexOutlined, PlusOutlined, RedoOutlined, RiseOutlined, SafetyOutlined, SearchOutlined, SettingOutlined, SnippetsOutlined, TableOutlined, TagOutlined, TeamOutlined, ThunderboltOutlined, PartitionOutlined, ToolOutlined, UndoOutlined, UpOutlined, WarningOutlined,
} from '@ant-design/icons'
import { computeCpm } from '../schedule/cpm'
import { computeCosts } from '../schedule/cost'
import { buildAoa } from '../schedule/aoa'
import { buildProjectXml, parseProjectXml } from '../schedule/mspdi'
import { checkProject } from '../schedule/checker'
import { makeCustomView } from '../schedule/views'
import { planFinishOf } from '../schedule/deadline'
import { latestReview } from '../schedule/reviews'
import type { GanttGroupField, GanttSort } from '../schedule/ganttGroup'
import type { SchedView } from '../schedule/types'
import { taskWarn, warnSummary, type WarnAlgorithm } from '../schedule/warn'
import type { SchedLink, SchedProject } from '../schedule/types'
import { useScheduleStore, isGroupRow, initScheduleSync } from '../schedule/store'
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
import { GanttView } from '../schedule/GanttView'
import { DcmaView } from '../schedule/DcmaView'
import { MonteView } from '../schedule/MonteView'
import { EvmView } from '../schedule/EvmView'
// 瘦身 P3：页面内子区块分片至此目录（正文零改动，仅迁移）
import { ExportDialog } from './schedule/ExportDialog'
import { ProjectsManagePanel } from './schedule/ProjectsManagePanel'
import { ProjectSettingsPanel } from './schedule/ProjectSettingsPanel'
import { ScheduleNetworkWorkspace } from './schedule/ScheduleNetworkWorkspace'
import {
  RibbonColumn,
  RibbonCommand,
  RibbonGroup,
  type ScheduleRibbonTabKey,
} from './schedule/ScheduleRibbon'
import '../schedule/schedule.css'
import '../gaea/styles.css'
import '../gaea/redesign.css'
import '../gaea/tailwind.css'

const SCHEDULE_RIBBON_TABS: ReadonlyArray<{ key: ScheduleRibbonTabKey; label: string }> = [
  { key: 'file', label: '文件' },
  { key: 'plan', label: '计划' },
  { key: 'resource', label: '资源' },
  { key: 'control', label: '管控' },
  { key: 'collab', label: '协作' },
  { key: 'view', label: '视图' },
  { key: 'help', label: '帮助' },
]

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
  const [exportOpen, setExportOpen] = useState(false)
  const [dockOpen, setDockOpen] = useState(true)
  const [ribbonTab, setRibbonTab] = useState<ScheduleRibbonTabKey>('plan')
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

  // 撤销/重做快捷键（刀C 余项）：Ctrl+Z / Ctrl+Y / Ctrl+Shift+Z；输入控件聚焦时不抢
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.ctrlKey || e.metaKey)) return
      const el = document.activeElement
      if (el instanceof HTMLElement && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable)) return
      const k = e.key.toLowerCase()
      if (k === 'z') { e.preventDefault(); if (e.shiftKey) redo(); else undo() } else if (k === 'y') { e.preventDefault(); redo() }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [undo, redo])

  const cpm = useMemo(() => computeCpm(project.tasks, project.links, { planFinish: planFinishOf(project), calendar: project.calendar, startDate: project.startDate }), [project])
  const aoa = useMemo(() => buildAoa(project.tasks, project.links, { planFinish: planFinishOf(project), persisted: project.aoa }), [project])
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
  const runMenu = (key: string) => {
    if (key === 'sample') { loadSample(); setImportMsg(null) }
    else if (key === 'importNew') pickImport('new', () => newRef.current?.click())
    else if (key === 'importXml') pickImport('xml', () => fileRef.current?.click())
    else if (key === 'importXlsx') pickImport('xlsx', () => xlsxRef.current?.click())
    else if (key === 'importMpp') pickImport('mpp', () => mppRef.current?.click())
    else if (key === 'exportXml') exportXml()
    else if (key === 'exportXlsx') void exportXlsx()
    else if (key === 'exportArt') setExportOpen(true)
    else if (key === 'manage') setManageOpen(true)
    else if (key === 'undo') undo()
    else if (key === 'redo') redo()
    else if (key === 'add') addTask(selectedId ?? undefined)
    else if (key === 'group') addGroup()
    else if (key === 'milestone' && selected && !selectedIsGroup) updateTask(selected.id, { isMilestone: !selected.isMilestone })
    else if (key === 'inspector' && selected && !selectedIsGroup) setInspectId(selected.id)
    else if (key === 'del' && selected) {
      Modal.confirm({
        title: '删除选中项',
        content: selectedIsGroup ? '分组及其子任务一并删除' : `删除「${selected.name}」及其搭接关系`,
        onOk: () => removeTask(selected.id),
      })
    } else if (key === 'clear') {
      Modal.confirm({ title: '清空全部任务与搭接？', onOk: () => { clearAll(); setImportMsg(null) } })
    } else if (key === 'gantt' || key === 'pdm' || key === 'aoa' || key === 'usage' || key === 'dcma' || key === 'monte' || key === 'evm') {
      setView(key)
    }
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
      { key: 'undo', icon: <UndoOutlined />, label: '撤销（Ctrl+Z）', disabled: past.length === 0 },
      { key: 'redo', icon: <RedoOutlined />, label: '重做（Ctrl+Y）', disabled: future.length === 0 },
      { type: 'divider' },
      { key: 'del', icon: <DeleteOutlined />, label: '删除选中', danger: true, disabled: !selected },
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
      <div className="sched-page" style={{ flex: 1, minWidth: 0 }}>
      {/* ── ZPert 式命令台：菜单行 + 命令面板行 + 文档属性行 ── */}
      <div className="sched-command-deck" data-testid="sched-command-deck">
      <div className="sched-menubar sched-ribbon-tabbar" data-testid="sched-menubar" role="tablist" aria-label="进度计划功能区">
        <span className="sched-app-mark" aria-hidden="true">G</span>
        {SCHEDULE_RIBBON_TABS.map((tab) => (
          <button
            key={tab.key}
            type="button"
            role="tab"
            aria-selected={ribbonTab === tab.key}
            className={`sched-menu-item${ribbonTab === tab.key ? ' is-active' : ''}`}
            onClick={() => setRibbonTab(tab.key)}
            data-testid={`sched-ribbon-tab-${tab.key}`}
          >
            {tab.label}
          </button>
        ))}
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
          <RibbonCommand big icon={<PartitionOutlined />} label="时标网络图" disabled title="预留：后续补齐时标网络图后端" />
          <RibbonCommand big icon={<TableOutlined />} label="横道图" active={view === 'gantt'} onClick={() => setView('gantt')} />
          <RibbonCommand big icon={<NodeIndexOutlined />} label="逻辑网络图" disabled title="预留：后续补齐逻辑网络图后端" />
          <RibbonCommand big icon={<NodeIndexOutlined />} label="单代号网络图" active={view === 'pdm'} onClick={() => setView('pdm')} />
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
              <RibbonCommand icon={<CopyOutlined />} label="保存视图" title="把当前横道列集/分组/排序/筛选存为命名视图" />
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
              <RibbonCommand icon={<AppstoreOutlined />} label="视图管理" title="应用/改名/删除已保存的视图" />
            </Popover>
            <RibbonCommand icon={<ClusterOutlined />} label="分组…" disabled title="预留：分组入口迁 ribbon（分组能力已在横道工具栏 14 维全接）" />
          </RibbonColumn>
        </RibbonGroup>
        <RibbonGroup label="编辑">
          <RibbonColumn>
            <RibbonCommand icon={<CopyOutlined />} label="复制" disabled title="预留：表格复制" />
            <RibbonCommand icon={<SnippetsOutlined />} label="粘贴" disabled title="预留：表格粘贴" />
            <Popconfirm title="删除选中项" onConfirm={() => selected && removeTask(selected.id)} disabled={!selected}>
              <button type="button" className="sched-ribbon-command is-small" disabled={!selected}>
                <DeleteOutlined /><span>删除</span>
              </button>
            </Popconfirm>
          </RibbonColumn>
          <RibbonColumn>
            <RibbonCommand icon={<SearchOutlined />} label="查找" disabled title="预留：表格查找" />
            <RibbonCommand icon={<FormatPainterOutlined />} label="格式刷" disabled title="预留：格式刷" />
            <RibbonCommand icon={<TagOutlined />} label="分类" disabled title="预留：工作分类" />
          </RibbonColumn>
        </RibbonGroup>
        <RibbonGroup label="格式">
          <RibbonColumn>
            <RibbonCommand icon={<FontColorsOutlined />} label="字体" disabled title="预留：字体设置" />
            <RibbonCommand icon={<FontSizeOutlined />} label="字号" disabled title="预留：字号设置" />
            <RibbonCommand icon={<FontColorsOutlined />} label="字体颜色" disabled title="预留：字体颜色" />
          </RibbonColumn>
          <RibbonColumn>
            <RibbonCommand icon={<BgColorsOutlined />} label="填充" disabled title="预留：单元格填充" />
            <RibbonCommand icon={<LineHeightOutlined />} label="线条宽度" disabled title="预留：线条宽度" />
          </RibbonColumn>
        </RibbonGroup>
        <RibbonGroup label="大纲">
          <RibbonColumn>
            <RibbonCommand icon={<NodeCollapseOutlined />} label="折叠" disabled title="预留：折叠大纲" />
            <RibbonCommand icon={<NodeExpandOutlined />} label="升级" disabled title="预留：升级大纲" />
            <RibbonCommand icon={<NodeCollapseOutlined />} label="降级" disabled title="预留：降级大纲" />
          </RibbonColumn>
          <RibbonColumn>
            <RibbonCommand icon={<UpOutlined />} label="上移" disabled title="预留：上移工作" />
            <RibbonCommand icon={<DownOutlined />} label="下移" disabled title="预留：下移工作" />
            <RibbonCommand icon={<SearchOutlined />} label="上钻" disabled title="预留：上钻大纲" />
          </RibbonColumn>
        </RibbonGroup>
        <RibbonGroup label="插入">
          <RibbonCommand big icon={<PlusOutlined />} label="工作" onClick={() => addTask(selectedId ?? undefined)} />
          <RibbonCommand big icon={<ToolOutlined />} label="关系" disabled title="预留：插入逻辑关系" />
          <RibbonCommand
            big
            icon={<AimOutlined />}
            label={selected?.isMilestone ? '取消里程碑' : '里程碑'}
            disabled={!selected || selectedIsGroup}
            active={!!selected?.isMilestone}
            onClick={() => selected && !selectedIsGroup && updateTask(selected.id, { isMilestone: !selected.isMilestone })}
          />
        </RibbonGroup>
        <RibbonGroup label="高级">
          <RibbonColumn>
            <RibbonCommand icon={<PartitionOutlined />} label="智能布图" disabled title="预留：智能布图" />
            <RibbonCommand icon={<FileImageOutlined />} label="按纸张" disabled title="预留：按纸张布图" />
            <Popover trigger="click" placement="bottom" content={<ProjectSettingsPanel cpm={cpm} />} title="工程设置">
              <button type="button" className="sched-ribbon-command is-small" data-testid="sched-project-settings-btn">
                <SettingOutlined /><span>设置</span>
              </button>
            </Popover>
          </RibbonColumn>
          <RibbonColumn>
            <RibbonCommand icon={<WarningOutlined />} label="预警" active={dockOpen} onClick={() => setDockOpen((open) => !open)} />
            <RibbonCommand icon={<SafetyOutlined />} label="检查" active={dockOpen} onClick={() => setDockOpen((open) => !open)} testId="sched-dock-toggle" />
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
            <RibbonCommand big icon={<MessageOutlined />} label="AI 助手" active={!chat.collapsed} onClick={toggleChat} />
          </RibbonGroup>
          <RibbonGroup label="历史">
            <RibbonColumn>
              <Dropdown trigger={['click']} menu={editMenuCfg}>
                <button type="button" className="sched-ribbon-command is-small" data-testid="sched-menu-edit">
                  <UndoOutlined /><span>编辑菜单</span>
                </button>
              </Dropdown>
              <RibbonCommand icon={<UndoOutlined />} label="撤销" disabled={past.length === 0} onClick={undo} testId="sched-undo" />
              <RibbonCommand icon={<RedoOutlined />} label="重做" disabled={future.length === 0} onClick={redo} testId="sched-redo" />
            </RibbonColumn>
          </RibbonGroup>
        </div>
      )}

      {ribbonTab === 'resource' && (
        <div className="sched-toolbar">
          <RibbonGroup label="资源">
            <Popover trigger="click" placement="bottom" content={<ResourcePanel cpm={cpm} />} title="资源与成本">
              <button type="button" className="sched-ribbon-command is-big" data-testid="sched-resource-btn">
                <TeamOutlined /><span>资源工作表</span>
              </button>
            </Popover>
            <RibbonCommand big icon={<TableOutlined />} label="资源使用" active={view === 'usage'} onClick={() => setView('usage')} />
          </RibbonGroup>
          <RibbonGroup label="成本">
            <RibbonCommand big icon={<ExportOutlined />} label="导出 Excel" onClick={() => void exportXlsx()} />
            <RibbonCommand big icon={<FileImageOutlined />} label="图面导出" onClick={() => setExportOpen(true)} />
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
            <RibbonCommand big icon={<SafetyOutlined />} label="计划检查" active={dockOpen} onClick={() => setDockOpen((open) => !open)} testId="sched-dock-toggle" />
            <RibbonCommand big icon={<SafetyOutlined />} label="质量体检" active={view === 'dcma'} onClick={() => setView('dcma')} />
          </RibbonGroup>
          <RibbonGroup label="分析">
            <RibbonCommand icon={<FundOutlined />} label="工期模拟" active={view === 'monte'} onClick={() => setView('monte')} />
            <RibbonCommand icon={<RiseOutlined />} label="挣值分析" active={view === 'evm'} onClick={() => setView('evm')} />
          </RibbonGroup>
        </div>
      )}

      {ribbonTab === 'collab' && (
        <div className="sched-toolbar">
          <RibbonGroup label="协作">
            <RibbonCommand big icon={<MessageOutlined />} label={chat.collapsed ? '展开 AI' : '收起 AI'} active={!chat.collapsed} onClick={toggleChat} />
            <RibbonCommand big icon={<ToolOutlined />} label="办公预览" onClick={openInOffice} />
          </RibbonGroup>
          <RibbonGroup label="工程">
            <RibbonCommand big icon={<SettingOutlined />} label="工程管理" onClick={() => setManageOpen(true)} />
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
            <RibbonCommand big icon={<TableOutlined />} label="横道图" active={view === 'gantt'} onClick={() => setView('gantt')} />
            <RibbonCommand big icon={<NodeIndexOutlined />} label="单代号" active={view === 'pdm'} onClick={() => setView('pdm')} />
            <RibbonCommand big icon={<PartitionOutlined />} label="双代号" active={view === 'aoa'} onClick={() => setView('aoa')} />
            <RibbonCommand big icon={<TeamOutlined />} label="资源使用" active={view === 'usage'} onClick={() => setView('usage')} />
          </RibbonGroup>
          <RibbonGroup label="分析视图">
            <RibbonColumn>
              <RibbonCommand icon={<SafetyOutlined />} label="质量体检" active={view === 'dcma'} onClick={() => setView('dcma')} />
              <RibbonCommand icon={<FundOutlined />} label="工期模拟" active={view === 'monte'} onClick={() => setView('monte')} />
              <RibbonCommand icon={<RiseOutlined />} label="挣值分析" active={view === 'evm'} onClick={() => setView('evm')} />
            </RibbonColumn>
          </RibbonGroup>
          <RibbonGroup label="显示">
            <RibbonCommand icon={<SafetyOutlined />} label="检查面板" active={dockOpen} onClick={() => setDockOpen((open) => !open)} />
            <RibbonCommand icon={<MessageOutlined />} label="AI 助手" active={!chat.collapsed} onClick={toggleChat} />
          </RibbonGroup>
        </div>
      )}

      {ribbonTab === 'help' && (
        <div className="sched-toolbar">
          <RibbonGroup label="起步">
            <Popover trigger="click" placement="bottom" content={<TemplatesPanel cpm={cpm} />} title="项目模板">
              <button type="button" className="sched-ribbon-command is-big" data-testid="sched-templates-btn">
                <AppstoreOutlined /><span>项目模板</span>
              </button>
            </Popover>
            <RibbonCommand big icon={<ThunderboltOutlined />} label="示例工程" onClick={() => { loadSample(); setImportMsg(null) }} />
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
      <ExportDialog open={exportOpen} onClose={() => setExportOpen(false)} project={project} cpm={cpm} aoa={aoa} defaultView={view === 'usage' || view === 'dcma' || view === 'monte' || view === 'evm' ? 'gantt' : view} />
      {!cpm.ok && cpm.error && (
        <Alert type="error" showIcon message={cpm.error} description="请修正搭接关系后重试；网络图视图在循环解除前不可用。" />
      )}

      <div className="sched-workspace" data-testid="sched-workspace">
        <div className="sched-workspace-main">
          {view === 'dcma' && <DcmaView project={project} cpm={cpm} />}
          {view === 'monte' && <MonteView project={project} cpm={cpm} />}
          {view === 'evm' && <EvmView project={project} cpm={cpm} />}
          {view === 'gantt' && <GanttView project={project} cpm={cpm} onInspect={setInspectId} showToolbar={false} />}
          {(view === 'pdm' || view === 'aoa') && <ScheduleNetworkWorkspace view={view} project={project} cpm={cpm} graph={aoa} />}
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

      {/* 底部状态栏（斑马口径：共 N 项工作总工期 N 天 + 关键/工作制常驻） */}
      <div className="sched-statusbar">
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
        <span style={{ marginLeft: 'auto' }} title={syncError ?? currentPath ?? SCHEDULE_FILE_PATH}>
          {!hydrated ? '读取计划…' : sync === 'dirty' ? '改动待保存…'
            : sync === 'saving' ? '保存中…'
            : sync === 'error' ? <span className="sched-sb-crit">保存失败</span>
            : savedAt ? <>已保存 {savedAt} · <span className="sched-dim">agent 可读写</span></> : '已同步 · agent 可读写'}
        </span>
      </div>
      </div>
    </div>
  )
}

export default SchedulePage


