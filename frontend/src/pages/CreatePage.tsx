import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Alert, Button, Dropdown, message, Modal } from 'antd'
import type { MenuProps } from 'antd'
import { DownOutlined } from '@ant-design/icons'
import { app } from '../gaea/lib/bridge'
import { GetNovelState, BuildNovelStatePatch, SettleNovelState, DeSlopChapterAiTaste, RewriteChapterAiTaste, GetEntityRelations, CancelCreateChapter, NovelFingerprintStatus, NovelFingerprintBuild, NovelFingerprintScore, NovelOutlineReconstruct, NovelOutlineReconstructApply, NovelOutlineReconstructStart, NovelOutlineReconstructTaskGet, NovelReviewPlatforms, NovelChapterReview } from '../../wailsjs/go/app/NovelB'
import { useOutlineStore } from '../stores/outlineStore'
import { useAppStore } from '../stores/appStore'
import type { OutlineNode } from '../types'
import { buildTree, flattenTree, buildPrevSummary, flattenChapters } from '../components/novel/create/outlineTree'
import { useChapterStream } from '../components/novel/create/useChapterStream'
import { useChapterGateNotice } from '../components/novel/create/useChapterGateNotice'
import type { AiTasteResult } from '../components/novel/create/chapterStreamTypes'
import type { ChapterReviewPayload, FingerprintScorePayload, FingerprintStatusPayload, ReviewPlatform, ChapterAnnotation, PlanGateReportView } from '../gaea/lib/bridge/novel'
import StyleFingerprintPanel from '../components/novel/StyleFingerprintPanel'
import ChapterReviewPanel from '../components/novel/ChapterReviewPanel'
import ChapterTreePanel from '../components/novel/create/ChapterTreePanel'
import RewriteModal from '../components/novel/RewriteModal'
import PartialRewriteModal from '../components/novel/PartialRewriteModal'
import RewriteHistoryPanel from '../components/novel/RewriteHistoryPanel'
import PromptWorkshopPanel from '../components/novel/PromptWorkshopPanel'
import ChapterAnalysisPanel from '../components/novel/ChapterAnalysisPanel'
import BookHealthPanel from '../components/novel/BookHealthPanel'
import StorySpinePanel from '../components/novel/StorySpinePanel'
import ConvergeModal from '../components/novel/ConvergeModal'
import ContextInventoryPanel from '../components/novel/create/ContextInventoryPanel'
import EditorPanel from '../components/novel/create/EditorPanel'
import type { EditorPanelHandle } from '../components/novel/create/EditorPanel'
import CreateInspector from '../components/novel/create/CreateInspector'
import NewCharactersModal from '../components/novel/create/NewCharactersModal'
import BranchWizardModal, { type Branch, type BranchCastEntry } from '../components/novel/create/BranchWizardModal'
import ChapterPlanCard, { type ChapterPlan } from '../components/novel/ChapterPlanCard'
import { chooseAction, chooseUnsavedAction } from '../components/novel/unsavedGuard'
import { registerNovelDirtyProvider, takeDiscardConfirmed } from '../components/novel/novelSwitchGuard'

interface WizardRequest { prevChapter: number; overwriteChapter: number; branchFromID: string }
const BRAINSTORM_MSG_KEY = 'novel-brainstorm-loading'

// ── 创作参数持久化（v4.421.0）──
// 目标字数/温度/写作技能此前每次进页都回默认值，作者每章都要重设一遍。
const GEN_PREFS_KEY = 'gaea.novel.genPrefs'
interface NovelGenPrefs { minWords: number; temperature: number; skill: string }
const GEN_PREFS_DEFAULT: NovelGenPrefs = { minWords: 5000, temperature: 0, skill: 'story-deslop' }

/** 读创作参数（畸形/越界一律回落默认值——本地存储不可信）。 */
function loadGenPrefs(): NovelGenPrefs {
  try {
    const raw = localStorage.getItem(GEN_PREFS_KEY)
    if (!raw) return GEN_PREFS_DEFAULT
    const v = JSON.parse(raw) as Partial<NovelGenPrefs>
    const minWords = typeof v.minWords === 'number' && Number.isFinite(v.minWords) && v.minWords >= 500 && v.minWords <= 20000
      ? Math.round(v.minWords) : GEN_PREFS_DEFAULT.minWords
    const temperature = typeof v.temperature === 'number' && Number.isFinite(v.temperature) && v.temperature >= 0 && v.temperature <= 2
      ? v.temperature : GEN_PREFS_DEFAULT.temperature
    const skill = typeof v.skill === 'string' ? v.skill : GEN_PREFS_DEFAULT.skill
    return { minWords, temperature, skill }
  } catch {
    return GEN_PREFS_DEFAULT
  }
}

// ── 章节计划硬闸（刀1 线D，规格 docs/gaea-longform-novel-system-2026-09.md §7.2/§7.6）──
// 生成前预检 NovelChapterGatePrecheck 是硬闸唯一判据来源。绑定面已收口
// （bridge/novel.ts：NovelChapterGatePrecheck → PlanGateReportView、
// CreateChapterWithOverride 9 参 allowOverride 均已入 AppBindings 类型面），
// 直接用已导出类型消费（可选字段 ?? 兜底），不再本地重定义/手写窄化（FE5-03）；
// 运行时 typeof 守卫保留——绑定未就绪的降级提示口径不变。

// ── 全文脑图：实体关系图类型 + 纯 SVG 渲染（零依赖、零硬编码 hex）──
interface EntityGraphNode { id?: string; name?: string; type?: string; group?: string | number }
interface EntityGraphEdge { from?: string; to?: string; type?: string }
interface EntityGraph { nodes: EntityGraphNode[]; edges: EntityGraphEdge[] }

// group → 语义令牌色（0 角色 / 1 组织 / 2 地点 / 3 物品 / 4 事件 / 5 其它）
const GRAPH_GROUP_TOKENS = [
  'var(--color-primary)',
  'var(--color-info)',
  'var(--color-success)',
  'var(--color-warning)',
  'var(--color-destructive)',
  'var(--color-text-secondary)',
]

function graphGroupColor(group: EntityGraphNode['group']): string {
  const g = typeof group === 'number' ? group : Number(group)
  const idx = Number.isFinite(g) && g >= 0 && g < GRAPH_GROUP_TOKENS.length ? g : GRAPH_GROUP_TOKENS.length - 1
  return GRAPH_GROUP_TOKENS[idx]
}

/** 把后端未知负载窄化为 EntityGraph（畸形/非对象 → 空图）。 */
function toEntityGraph(value: unknown): EntityGraph {
  if (typeof value === 'object' && value !== null) {
    const rec = value as Record<string, unknown>
    const nodes = Array.isArray(rec.nodes)
      ? rec.nodes.map((n) => (typeof n === 'object' && n !== null ? n as EntityGraphNode : {}))
      : []
    const edges = Array.isArray(rec.edges)
      ? rec.edges.map((e) => (typeof e === 'object' && e !== null ? e as EntityGraphEdge : {}))
      : []
    return { nodes, edges }
  }
  return { nodes: [], edges: [] }
}

/** 内联 SVG：节点=圆形带名字（按环均匀排布），边=直线；viewBox 0 0 W H、宽度自适应。 */
const EntityGraphSvg: React.FC<{ graph: EntityGraph }> = ({ graph }) => {
  const W = 640
  const H = 520
  const cx = W / 2
  const cy = H / 2
  const radius = Math.min(W, H) / 2 - 70
  const byId: Record<string, { x: number; y: number; label: string; color: string }> = {}
  graph.nodes.forEach((n, i) => {
    if (!n.id) return
    const angle = graph.nodes.length === 1 ? -Math.PI / 2 : (2 * Math.PI * i) / graph.nodes.length - Math.PI / 2
    const x = cx + radius * Math.cos(angle)
    const y = cy + radius * Math.sin(angle)
    const raw = n.name || n.id || '未命名'
    const label = raw.length > 5 ? `${raw.slice(0, 5)}…` : raw
    byId[n.id] = { x, y, label, color: graphGroupColor(n.group) }
  })
  const lines = graph.edges
    .map((e) => {
      if (!e.from || !e.to) return null
      const a = byId[e.from]
      const b = byId[e.to]
      return a && b ? { x1: a.x, y1: a.y, x2: b.x, y2: b.y } : null
    })
    .filter((x): x is { x1: number; y1: number; x2: number; y2: number } => x !== null)
  return (
    <svg viewBox={`0 0 ${W} ${H}`} width="100%" style={{ maxHeight: 480 }}>
      {lines.map((l, i) => (
        <line key={i} x1={l.x1} y1={l.y1} x2={l.x2} y2={l.y2} stroke="var(--color-border)" strokeWidth={1} />
      ))}
      {graph.nodes.map((n, i) => {
        const pos = n.id ? byId[n.id] : undefined
        if (!pos) return null
        return (
          <g key={n.id ?? i}>
            <circle cx={pos.x} cy={pos.y} r={26} style={{ fill: pos.color, stroke: pos.color }} fillOpacity={0.16} strokeWidth={1.5} />
            <text x={pos.x} y={pos.y} textAnchor="middle" dominantBaseline="central" fontSize={12} style={{ fill: pos.color, fontFamily: 'inherit' }}>
              {pos.label}
            </text>
          </g>
        )
      })}
    </svg>
  )
}

/**
 * 创作工具轨（简化批）：17 钮平铺 → 高频直钮（章节计划 / 章节分析 / 一键去味）
 * + 三个分组菜单（质检 / 文本 / 结构），功能零删除——低频入口收进菜单，仍一次点击可达。
 * v4.421.0 的「刻意不做折叠菜单」随钮数涨到 17 退役：菜单不增加触达成本，
 * 却把默认视觉噪音从 17 钮压到 6 控件。每组成员由 CreatePage.test「工具轨分组」锁定。
 */

// T6-7.5 拆分后的编排层（≤300 行）：持有页面状态与生成编排；视图拆到 5 个子组件；
// 流式事件经 useChapterStream + chapterStreamTypes 判别联合分发（T6-7.2 停止按钮 + cancelled）。
// v4.421.0：新增 `active`（本 pane 是否当前可见页——隐藏≠卸载，窗口级副作用按它门控）
// 与未保存正文保护（loadedSnapshot/dirty + unsavedGuard 三选确认）。
const CreatePage: React.FC<{ active?: boolean }> = ({ active = true }) => {
  const [setting, setSetting] = useState('')
  const outlines = useOutlineStore(s => s.outlines)
  const loadOutlines = useOutlineStore(s => s.loadOutlines)
  const projectPath = useAppStore(s => s.projectPath)
  const projectOpen = useAppStore(s => s.projectOpen)

  const [activeId, setActiveId] = useState('')
  const [content, setContent] = useState('')
  // 未保存保护（v4.421.0）：loadedSnapshot = 最近一次「载入成功 / 保存成功 / 生成落盘」
  // 之后的正文快照；content 与之不同即 dirty。ref 版本供异步回调（确认弹窗、流式事件）读最新值。
  const [loadedSnapshot, setLoadedSnapshot] = useState('')
  const loadedSnapshotRef = useRef('')
  const contentRef = useRef('')
  // 本页是否当前可见（供 window 级监听门控；事件到达时 prop 可能还没更新，故走 ref）
  const activeRef = useRef(active)
  activeRef.current = active
  loadedSnapshotRef.current = loadedSnapshot
  contentRef.current = content

  // 生成后自动门轻通知（v4.331：chapter-gate 事件消费——契约/质量/AI 味/分析同步）
  // v4.336：通知可点击→跳章节分析面板；跨章时拉该章正文供标注锚定（失败回退空）
  const [gateChapter, setGateChapter] = useState<number | null>(null)
  const [gateContent, setGateContent] = useState('')
  useChapterGateNotice((n) => {
    setGateChapter(n)
    if (n === activeChapterNum) {
      setGateContent('')
    } else {
      void app.GetChapter(n)
        .then((ch) => setGateContent(String((ch?.content as string) ?? '')))
        .catch(() => setGateContent(''))
    }
    setAnalysisOpen(true)
  })
  const [chapterLoading, setChapterLoading] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [saving, setSaving] = useState(false)
  const [genPhase, setGenPhase] = useState('')
  const [genPercent, setGenPercent] = useState(0)
  const [stopping, setStopping] = useState(false)

  const [minWords, setMinWords] = useState(() => loadGenPrefs().minWords)
  const [temperature, setTemperature] = useState(() => loadGenPrefs().temperature)
  const [directPlot, setDirectPlot] = useState('')
  const [stats, setStats] = useState<{ totalWords: number; chapterCount: number } | null>(null)
  // 整章重写弹窗（t4-C3 前端消费）
  const [rwOpen, setRwOpen] = useState(false)
  // 重写版本历史面板（t4-C3 余项：版本库 List/Get 前端消费）
  const [rwHistOpen, setRwHistOpen] = useState(false)
  // 提示词工坊面板（t6 首刀：模板可编辑覆盖层；打开才拉一次）
  const [promptWsOpen, setPromptWsOpen] = useState(false)
  // 章节计划卡（刀1 线D）：工具轨「章节计划」与硬闸弹窗共用同一展开位
  const [planOpen, setPlanOpen] = useState(false)
  /**
   * 计划种子（v4.450.0 三点打通）：硬闸拦下生成时把本次 plotReq（分支意向/剧情
   * 要求）存为创作方向，随计划卡下发 Propose——分支→计划才真正连通。工具轨手动
   * 展开时清空（不带旧方向）。
   */
  const [planSeed, setPlanSeed] = useState('')
  /** 硬闸解析出的目标章号：弹窗里「立即生成计划草案」按它开计划卡（对齐预检章号） */
  const [planChapterOverride, setPlanChapterOverride] = useState(0)
  /** 硬闸弹窗（生成前预检 blocking 时）：缺失清单 + 「立即生成计划草案」/「仍然生成」 */
  const [planGate, setPlanGate] = useState<{ chapterNum: number; missing: string[]; problems: string[]; proceed: () => void } | null>(null)
  /** 本次生成是否经作者显式覆盖硬闸（随 CreateChapterWithOverride 的 allowOverride 下发） */
  const planOverrideRef = useRef(false)
  const [analysisOpen, setAnalysisOpen] = useState(false)
  // 标注定位编辑器（t7 观察池）：面板「编辑器定位」→ 关面板 → 编辑器光标
  // 选区定位到标注区间（rune 偏移，EditorPanel 内部换算 code-unit）。
  // gate 跳转他章时面板章≠编辑章（gateContent 是他章正文），不提供该入口。
  const editorRef = useRef<EditorPanelHandle>(null)
  const handleLocateInEditor = (ann: ChapterAnnotation) => {
    setAnalysisOpen(false)
    setGateChapter(null)
    setGateContent('')
    const ok = editorRef.current?.locate(ann.pos, ann.pos + (ann.length ?? 0)) ?? false
    if (!ok) message.info('请先在左侧选择该章，再定位标注')
  }
  const [healthOpen, setHealthOpen] = useState(false)
  const [spineOpen, setSpineOpen] = useState(false)
  const [convergeOpen, setConvergeOpen] = useState(false)
  const [ctxInvOpen, setCtxInvOpen] = useState(false)
  // 局部重写选区（t4-C3 余项：EditorPanel 选段回调 → PartialRewriteModal；rune 偏移）
  const [pSel, setPSel] = useState<{ start: number; end: number; text: string } | null>(null)
  // 默认启用 story-deslop 去 AI 味润色技能；可在右侧创作设置中切换或清空
  const [selectedSkill, setSelectedSkill] = useState<string | undefined>(() => loadGenPrefs().skill || undefined)
  // 创作参数持久化（改一次即记住；读取失败静默回落默认值）
  useEffect(() => {
    try {
      localStorage.setItem(GEN_PREFS_KEY, JSON.stringify({ minWords, temperature, skill: selectedSkill ?? '' }))
    } catch { /* 持久化失败不影响创作 */ }
  }, [minWords, temperature, selectedSkill])
  // 生成完成后 novelstyle 的 AI 味检测结果（分数 + 命中问题），展示给作者
  const [aiTaste, setAiTaste] = useState<AiTasteResult | null>(null)
  // 叙事状态账本（narrative 审批制结算）UI
  const [stateOpen, setStateOpen] = useState(false)
  const [novelState, setNovelState] = useState<{ version?: number; entities?: Record<string, { name?: string; type?: string; status?: string }> } | null>(null)
  const [statePatch, setStatePatch] = useState<unknown>(null)
  const [stateBusy, setStateBusy] = useState(false)
  const [stateMsg, setStateMsg] = useState('')

  // 全文脑图（实体关系）弹窗状态
  const [graphOpen, setGraphOpen] = useState(false)
  const [graphBusy, setGraphBusy] = useState(false)
  const [graphData, setGraphData] = useState<EntityGraph | null>(null)
  const [graphMsg, setGraphMsg] = useState('')

  // 文风指纹（参考档 + 章节体检）弹窗状态；Status 在打开时自动拉取（无独立刷新按钮）
  const [fpOpen, setFpOpen] = useState(false)
  const [fpStatus, setFpStatus] = useState<FingerprintStatusPayload | null>(null)
  const [fpScore, setFpScore] = useState<FingerprintScorePayload | null>(null)
  const [fpBusy, setFpBusy] = useState(false)
  // 平台评审（v4.282，oh-story 蒸馏 T1）：档位清单 + 单章报告 + 动作态。
  const [reviewOpen, setReviewOpen] = useState(false)
  const [reviewPlatforms, setReviewPlatforms] = useState<ReviewPlatform[]>([])
  const [reviewPlatformId, setReviewPlatformId] = useState('general')
  const [reviewReport, setReviewReport] = useState<ChapterReviewPayload | null>(null)
  const [reviewBusy, setReviewBusy] = useState(false)
  const [reviewMsg, setReviewMsg] = useState('')
  // AI 反推大纲（v4.281）：反推 → 确认 → 幂等合并进大纲
  const [reconstructBusy, setReconstructBusy] = useState(false)
  // v4.340 取消：轮询期「取消反推」可见（cancellable=已入队拿到 taskId），
  // 点击置标志 → 轮询下个检查点退出 + 请求后端协作取消任务。
  const [reconstructCancellable, setReconstructCancellable] = useState(false)
  const reconstructCancelRef = useRef(false)
  const reconstructTaskIdRef = useRef('')
  /** 已等待秒数（v4.421.0：长任务可见化——12 分钟轮询期间此前只有一个静态提示） */
  const [reconstructElapsed, setReconstructElapsed] = useState(0)
  /** 组件是否还在（v4.421.0：卸载/切走后停止轮询，避免对已卸载组件 setState 与空跑请求） */
  const reconstructAliveRef = useRef(true)
  useEffect(() => () => { reconstructAliveRef.current = false }, [])
  const [fpMsg, setFpMsg] = useState('')

  const openGraph = async () => {
    setGraphOpen(true)
    setGraphBusy(true)
    setGraphMsg('')
    try {
      const g = toEntityGraph(await GetEntityRelations())
      setGraphData(g)
      setGraphMsg(g.nodes.length > 0 ? `共 ${g.nodes.length} 个实体 · ${g.edges.length} 条关系` : '')
    } catch (err: unknown) {
      setGraphData(null)
      setGraphMsg(err instanceof Error ? err.message : '加载实体关系失败')
    } finally {
      setGraphBusy(false)
    }
  }
  const [editorFontSize, setEditorFontSize] = useState<number>(() => {
    try {
      const v = Number(localStorage.getItem('gaea.novel.editorFontSize'))
      if (Number.isFinite(v) && v >= 12 && v <= 24) return v
    } catch { /* 读取失败按默认值 */ }
    return 15
  })
  const [wizard, setWizard] = useState<WizardRequest | null>(null)
  const [wizardBranches, setWizardBranches] = useState<Branch[]>([])
  // 选角会议：结果由页面层持有（后台构思期间弹窗收起不丢）
  const [wizardCastSelection, setWizardCastSelection] = useState<BranchCastEntry[]>([])
  // 选角会议数据源：项目名册 / 角色库候选 / 上一章出场（向导打开时拉取；失败静默降级）
  const [wizardCast, setWizardCast] = useState<string[]>([])
  const [wizardLibCast, setWizardLibCast] = useState<{ name: string; note?: string }[]>([])
  const [wizardPrevCast, setWizardPrevCast] = useState<string[]>([])

  const settingLoadToken = useRef(0)
  const chapterLoadToken = useRef(0)
  const generatingRef = useRef(false)
  const brainstormingRef = useRef(false)
  /** 后台构思发起时的向导请求（完成后按原请求重开弹窗，保留覆盖/分支语义） */
  const brainstormReqRef = useRef<WizardRequest | null>(null)
  /** 最新「保存当前章」实现（供确认弹窗的异步回调调用，避开闭包过期） */
  const saveActiveRef = useRef<() => Promise<boolean>>(async () => false)
  /** 当前激活章节 id 的 ref 版本（同理由：异步回调读最新值） */
  const activeIdRef = useRef('')
  activeIdRef.current = activeId
  /**
   * 章节载入在途（A4）：`loadChapter` 在 await **之前**就把 activeId 指向新章，
   * 而 content 仍是上一章正文——此刻保存会拿「新章号 + 旧章正文」组装载荷，把第 5 章
   * 正文写进第 6 章。走 ref 是因为 state 更新是异步的：只在 render 里镜像
   * `chapterLoading` 会留下「已开始载入但仍可保存」的窗口。
   */
  const chapterLoadingRef = useRef(false)
  /** 最新 finishStream（切书 effect 的声明早于该回调，走 ref 规避 TDZ）。 */
  const finishStreamRef = useRef<() => void>(() => {})
  /**
   * 本次生成是否已就「切换小说导致结果失效」提示过（A3）：projectPath effect 与
   * 晚到的流事件都会发现「旧书任务作废」，两处都提示就成了二次惊吓。
   */
  const staleStreamNotifiedRef = useRef(false)

  const handleEditorFontSizeChange = useCallback((v: number) => {
    setEditorFontSize(v)
    try { localStorage.setItem('gaea.novel.editorFontSize', String(v)) } catch { /* 持久化失败不影响使用 */ }
  }, [])

  // 当前生成任务的章节标识（取 CreateChapter 返回值，供停止按钮 CancelCreateChapter）
  const streamTargetRef = useRef({ chapterNum: 0, branch: '' })

  const { attach, detach } = useChapterStream()

  // 拉取最新小说设定；projectPath 变化（切换小说）时重新拉取。
  // v4.421.0：返回 ok 旗——「读取失败」与「设定确实为空」是两件事，旧实现混为一谈，
  // 生成前会把一次磁盘/绑定失败误报成「设定为空，请先在设定页填写」，把作者引到错地方。
  const refreshSetting = useCallback(async (): Promise<{ text: string; ok: boolean }> => {
    const token = ++settingLoadToken.current
    const requestedPath = useAppStore.getState().projectPath
    let fresh = ''
    let ok = true
    try { fresh = await app.GetWorldview() || '' } catch { ok = false }
    if (token !== settingLoadToken.current || requestedPath !== useAppStore.getState().projectPath) {
      return { text: '', ok: false }
    }
    if (ok) setSetting(fresh)
    return { text: fresh, ok }
  }, [])

  // 创作统计（章节数 / 总字数）
  const refreshStats = useCallback(async () => {
    try {
      const s = await app.GetStats()
      if (s) setStats(s as { totalWords: number; chapterCount: number })
    } catch { /* 统计失败不阻塞创作 */ }
  }, [])

  useEffect(() => {
    ;(async () => { loadOutlines(); await refreshSetting(); refreshStats() })()
    // projectOpen 必须在依赖里：应用重启后 projectPath 持久恢复、本 effect 在后端
    // 项目尚未打开时就跑（GetOutlines 空返回）——等价路径打开项目不触发 projectPath
    // 变化，大纲库永远空着，下一章恒算 1 反复覆盖第1章（实弹事故）。
  }, [projectPath, projectOpen, loadOutlines, refreshSetting, refreshStats])

  // 切换小说时清空编辑区与选中节点，避免展示上一个项目的内容。跨页切书已由书架经
  // `novelSwitchGuard` 先行确认（三选：先保存 / 放弃修改 / 取消），故此处只在
  // **未经闸门**的切书路径上兜底告知——不能静默吞掉，也不能对已确认的重复惊吓。
  useEffect(() => {
    const confirmed = takeDiscardConfirmed('create-body')
    if (!confirmed && contentRef.current !== loadedSnapshotRef.current) {
      message.warning('已切换小说：上一本未保存的正文修改未保留')
    }
    // A3：在途生成属于**上一本书**。切书后它的 done 会把旧书正文挂到新书同号章节点、
    // 甚至 `setLoadedSnapshot` 标「已保存」，作者一点保存就写进新书文件。故这里主动停掉
    // 后端任务并本地收尾（不这样做则 generating=true 悬挂：停止按钮留在界面上但任务已换书）。
    // 事件侧另有 requestedPath 守卫，兜住「effect 尚未跑完」的窗口（见 runGeneration）。
    if (generatingRef.current) {
      const { chapterNum, branch } = streamTargetRef.current
      if (chapterNum) void CancelCreateChapter(chapterNum, branch).catch(() => { /* 换书后取消失败可忽略，本地已收尾 */ })
      finishStreamRef.current()
      if (!staleStreamNotifiedRef.current) {
        staleStreamNotifiedRef.current = true
        message.warning('已切换小说：正在进行的生成已停止，该次结果未写入当前书')
      }
    }
    setActiveId(''); setContent(''); setLoadedSnapshot('')
    // B5a：评审报告 / 文风指纹状态与提示语都取自上一本书——不清则切书后弹窗仍整屏显示
    // 上一本的内容（审计确证）。两个弹窗一并关闭：留在屏上比「需要重开一次」更糟。
    setReviewReport(null); setReviewMsg(''); setReviewOpen(false)
    setFpStatus(null); setFpScore(null); setFpMsg(''); setFpOpen(false)
    // 仅在项目变化时执行（content/snapshot/finishStream 均走 ref，不进依赖）
  }, [projectPath])

  // 跨页切书未保存保护（`novelSwitchGuard` 的登记端）：把「创作页正文脏不脏、
  // 能否先保存」暴露给书架页的切书闸门。dirty/save 均经 ref 读最新值，故只登记一次。
  useEffect(() => registerNovelDirtyProvider({
    id: 'create-body',
    label: () => '创作页正文',
    dirty: () => contentRef.current !== loadedSnapshotRef.current,
    // 没有选中节点时无从保存（saveActive 会自己给出可见提示并返回 false）
    canSave: () => !!activeIdRef.current,
    save: () => saveActiveRef.current(),
  }), [])

  /** 真正载入章节正文（不做脏保护——重写应用后的刷新也走它）。 */
  const loadChapter = useCallback(async (node: OutlineNode) => {
    const token = ++chapterLoadToken.current
    const requestedPath = useAppStore.getState().projectPath
    // A4：同步置位（不能只靠 render 镜像 chapterLoading——state 更新是异步的），
    // 让「载入在途」对 saveActive 立刻可见。
    chapterLoadingRef.current = true
    setActiveId(node.id); setChapterLoading(true)
    try {
      const branch = node.branch || ''
      const result = await app.GetChapterBranch(node.order_index || 1, branch)
      if (token === chapterLoadToken.current && requestedPath === useAppStore.getState().projectPath) {
        const text = (result?.content as string) || ''
        setContent(text)
        setLoadedSnapshot(text)
      }
    } catch (err: unknown) {
      // v4.421.0 载入失败可见化：旧实现静默清空编辑区，作者会以为该章是空的，
      // 接着保存/生成就覆盖掉真实正文（阅读页 v4.350 已修同类缺陷，创作页此前是漏网点）。
      if (token === chapterLoadToken.current && requestedPath === useAppStore.getState().projectPath) {
        setContent(''); setLoadedSnapshot('')
        message.error(`第 ${node.order_index || '?'} 章载入失败：${err instanceof Error ? err.message : String(err)}（编辑区已留空，请勿直接保存）`)
      }
    } finally {
      // token 不匹配＝已有更新的载入在跑，在途标志归它管，不得提前解除
      if (token === chapterLoadToken.current) { chapterLoadingRef.current = false; setChapterLoading(false) }
    }
  }, [])

  /**
   * 用户主动切章：生成中拒绝；有未保存修改先确认（先保存 / 放弃修改 / 取消）。
   * v4.421.0：此前直接 setContent 覆盖，未保存的正文静默丢失。
   */
  const selectChapter = useCallback((node: OutlineNode) => {
    if (generatingRef.current) { message.warning('正在生成，请先停止生成再切换章节'); return }
    if (contentRef.current !== loadedSnapshotRef.current) {
      chooseUnsavedAction({
        title: '正文有未保存的修改',
        message: `切到「${node.title || `第${node.order_index || '?'}章`}」会丢弃当前修改。`,
        onSave: () => { void saveActiveRef.current().then((ok) => { if (ok) void loadChapter(node) }) },
        onDiscard: () => { void loadChapter(node) },
      })
      return
    }
    void loadChapter(node)
  }, [loadChapter])

  // 向导拉取 AI 构思分支（注入最新设定与前文摘要；cast=选角会议结果）。
  // v4.452.0：targetChapter=分支目标章（下一章）——驱动后端每10章一阶段的
  // 起承转合位置注入；0=位置未知走旧兜底。
  const fetchWizardBranches = useCallback(async (prevChapter: number, cast: BranchCastEntry[] = [], targetChapter = 0): Promise<Branch[]> => {
    const freshSetting = (await refreshSetting()).text
    const prevSummary = prevChapter > 0 ? buildPrevSummary(outlines, prevChapter) : ''
    const payload = (cast ?? []).map(e => ({ name: e.name, relation: e.relation || '', note: e.note || '' }))
    const res = (await app.QuickBrainstormBranches(freshSetting, prevSummary || '', JSON.stringify(payload), targetChapter)) as { branches?: Array<{ title?: string; summary?: string }> }
    const list = res?.branches || []
    return list.map((b: { title?: string; summary?: string }) => ({ title: b.title ?? '', pitch: b.summary ?? '' }))
  }, [refreshSetting, outlines])

  // 打开分支向导：先选角后构思（选角会议在弹窗内完成，点「构思分支」转后台）
  const openWizard = useCallback(async (prevChapter: number, overwriteChapter = 0, branchFromID = '') => {
    // A7：生成中开向导＝在同一章上叠第二写者，且向导末尾的「生成」会被 startGeneration
    // 的在途早退静默吞掉（作者以为点了没反应）。此处给可见提示，文案口径对齐 selectChapter。
    if (generatingRef.current) { message.warning('正在生成，请先停止生成再生成下一章'); return }
    if (brainstormingRef.current) { message.info('上一轮构思还在后台进行，完成后会自动弹出'); return }
    setWizardBranches([]) // 新章的选角会议从零开始（要延续走「延续上一章」）
    setWizardCastSelection([])
    // 新章章号不在前端算（store 可能空/过期）：CreateChapter 传 0，后端按磁盘
    // 大纲解析为「全树最大章号+1」
    setWizard({ prevChapter, overwriteChapter, branchFromID })
    // 选角会议数据源：项目名册 + 角色库候选 + 上一章出场（失败静默降级，不阻断选角）
    void (async () => {
      try {
        const res = (await app.GetCharacters()) as { characters?: Array<{ name?: string }> } | null
        setWizardCast((res?.characters || []).map(c => (c?.name || '').trim()).filter(Boolean))
      } catch { setWizardCast([]) }
      try {
        const lib = (await app.CharacterList('', '', false, 1, 50)) as { items?: Array<{ name?: string; personality?: string }> } | null
        setWizardLibCast((lib?.items || []).map(c => ({ name: (c?.name || '').trim(), note: (c?.personality || '').trim().slice(0, 40) })).filter(c => c.name))
      } catch { setWizardLibCast([]) }
      if (prevChapter > 0) {
        try {
          const ch = (await app.GetChapter(prevChapter)) as { summary?: { characters_appeared?: string[] } } | null
          setWizardPrevCast((ch?.summary?.characters_appeared || []).map(s => (s || '').trim()).filter(Boolean))
        } catch { setWizardPrevCast([]) }
      } else {
        setWizardPrevCast([])
      }
    })()
  }, [])

  // 后台构思分支：弹窗随即收起（可继续操作其他内容），完成后按原请求自动弹回分支步
  const startBrainstorm = useCallback((cast: BranchCastEntry[]) => {
    if (brainstormingRef.current) { message.info('上一轮构思还在后台进行，完成后会自动弹出'); return }
    const req = wizard ?? { prevChapter: 0, overwriteChapter: 0, branchFromID: '' }
    brainstormReqRef.current = req
    brainstormingRef.current = true
    setWizardCastSelection(cast)
    setWizard(null) // 收起弹窗，不阻塞界面
    message.open({ key: BRAINSTORM_MSG_KEY, content: 'AI 正在构思剧情分支，你可以继续操作…', duration: 0 })
    void (async () => {
      try {
        // 分支目标章：覆盖=被覆盖章；其余=下一章（前端不读 store 顺延口径，
        // prevChapter+1 与后端「最大章号+1」在日常流一致；位置注入只影响教义段，
        // 不影响章号事实源）
        const targetChapter = req.overwriteChapter > 0 ? req.overwriteChapter : req.prevChapter + 1
        const list = await fetchWizardBranches(req.prevChapter, cast, targetChapter)
        if (list.length === 0) {
          message.warning('AI 未构思出剧情分支，可重新打开向导调整选角再试')
          return // 不重开弹窗：保持收起，选角已留在页面层
        }
        setWizardBranches(list)
        setWizard(brainstormReqRef.current) // 按原请求重开 → 弹窗直接进分支步
      } catch (err: unknown) {
        message.error(err instanceof Error ? err.message : '剧情构思失败，可重新打开向导重试')
      } finally {
        message.destroy(BRAINSTORM_MSG_KEY)
        brainstormingRef.current = false
      }
    })()
  }, [wizard, fetchWizardBranches])

  // 流式生成收尾：三路终态（done/error/cancelled）与停止兜底共用
  const finishStream = useCallback(() => {
    setGenPhase(''); setGenPercent(0); setGenerating(false); setStopping(false)
    generatingRef.current = false
    detach()
  }, [detach])
  finishStreamRef.current = finishStream

  // 直接开始生成：注册流式监听并调用后端（带目标字数/温度/技能设置）
  const runGeneration = async (plotReq: string, overwriteChapter = 0, branchFromID = '') => {
    if (generatingRef.current) return
    // A3：记下本任务属于哪本书。旧书生成完成时事件若照单全收，正文会被挂到**新书**同号章
    // 节点（`loadOutlines` 按 order_index 找节点），并 `setLoadedSnapshot` 标「已保存」——
    // 作者此后一点保存就写进新书文件。故事件回调全程按发起时的 projectPath 校验。
    const requestedPath = useAppStore.getState().projectPath
    staleStreamNotifiedRef.current = false
    generatingRef.current = true
    setGenerating(true); setGenPhase('正在生成…'); setGenPercent(0); setContent(''); setLoadedSnapshot(''); setStopping(false); setAiTaste(null)
    contentRef.current = ''

    attach({
      onEvent: (ev) => {
        // 本任务已不属于当前书：整体忽略（不 setState 到新书上下文）。三路终态额外如实提示
        // 一次「结果未写入当前书」——收尾（Cancel + finishStream）由 projectPath effect 负责，
        // 那里一定会跑；此处若擅自 finishStream 反而会摘掉监听、让 effect 误判为无在途任务。
        if (useAppStore.getState().projectPath !== requestedPath) {
          if (ev.type !== 'chunk' && !staleStreamNotifiedRef.current) {
            staleStreamNotifiedRef.current = true
            message.warning('已切换小说：该次生成因切书已失效，结果未写入当前书')
          }
          return
        }
        switch (ev.type) {
          case 'phase':
            if (ev.phase === 'continuing') {
              setGenPhase(`字数不足，正在续写… 第${ev.attempt || 1}次 · ${(ev.current || 0).toLocaleString()}/${(ev.target || minWords).toLocaleString()} 字`)
            } else {
              setGenPhase(`正在生成… 目标 ${(ev.target || minWords).toLocaleString()} 字`)
            }
            break
          case 'chunk':
            setContent((prev) => {
              const next = prev + ev.content
              // 同步镜像到 ref：done/cancelled 可能紧随其后（同 tick）读最新正文
              contentRef.current = next
              return next
            })
            setGenPercent(Math.min(100, Math.round(((ev.total || 0) / Math.max(minWords, 1)) * 100)))
            setGenPhase(`正在生成… ${(ev.total || 0).toLocaleString()}/${minWords.toLocaleString()} 字`)
            break
          case 'done': {
            finishStream()
            // 生成完成＝后端已落盘：把当前正文视为已保存（否则切章会误弹「未保存」确认）
            setLoadedSnapshot(contentRef.current)
            setAiTaste(ev.aiTaste ?? null)
            const chNum = ev.chapterNum || 0
            const branch = ev.branch || ''
            message.success(`${branch ? `第${chNum}${branch}章` : `第${chNum}章`} 生成完成（${(ev.total || 0).toLocaleString()} 字）`)
            refreshStats()
            loadOutlines().then(() => {
              const ch = useOutlineStore.getState().outlines.find(n => n.order_index === chNum && n.branch === branch)
              if (ch) setActiveId(ch.id)
            })
            break
          }
          case 'error':
            finishStream()
            message.error(ev.error || '生成失败')
            break
          case 'cancelled': {
            finishStream()
            // 取消：后端已落盘部分正文（事件携带 content）；保留编辑器已累积正文可继续编辑/保存。
            // 事件**不带** content＝后端未落盘（空稿/落盘失败）→ 保持 dirty（不假装已保存）。
            // v4.421.0：正稿已存在时后端把残稿另存侧车（partialSaved）——此时正稿未变，
            // 前端不得把残稿标成「已保存」，须如实提示落点。
            if (typeof ev.content === 'string' && ev.content.length > 0) {
              setContent(ev.content)
              if (!ev.partialSaved) setLoadedSnapshot(ev.content)
            }
            const chNum = ev.chapterNum || 0
            const branch = ev.branch || ''
            const chapLabel = branch ? `第${chNum}${branch}章` : `第${chNum}章`
            if (ev.partialSaved) {
              message.warning(`${chapLabel} 已有正文，本次取消的 ${(ev.total || 0).toLocaleString()} 字未覆盖正稿，已另存为残稿：${ev.partialPath || '见小说目录'}`)
            } else {
              message.info(`${chapLabel} 已停止生成（已保留 ${(ev.total || 0).toLocaleString()} 字）`)
            }
            refreshStats()
            break
          }
        }
      },
    })

    try {
      // 生成前再读一次最新设定，确保正文提示词注入当前小说设定
      const { text: freshSetting, ok: settingReadOk } = await refreshSetting()
      // v4.421.0：读取失败与「设定确实为空」分开报——旧实现把一次读取失败说成
      // 「设定为空，请先去设定页填写」，把作者引到错地方。
      if (!settingReadOk) throw new Error('小说设定读取失败，请稍后重试（本次未开始生成）')
      if (!freshSetting.trim()) throw new Error('小说设定为空，请先在「设定」页填写世界观')
      // 硬闸覆盖意图（刀1 线D）：作者在硬闸弹窗点「仍然生成（跳过硬闸）」后落此标志。
      // CreateChapterWithOverride（9 参 allowOverride）已在 bridge/novel.ts 类型面
      // （AppBindings），直接类型化调用；运行时 typeof 守卫保留——绑定未就绪的降级
      // 提示口径不变（缺绑定时如实警告并走普通入口）。
      const allowOverride = planOverrideRef.current
      planOverrideRef.current = false
      let result: { nodeId?: string; chapterNum?: number; branch?: string }
      if (allowOverride && typeof app.CreateChapterWithOverride === 'function') {
        message.info('已按你的选择跳过章节计划硬闸')
        result = (await app.CreateChapterWithOverride(
          freshSetting, '', plotReq, overwriteChapter, branchFromID, selectedSkill || '', minWords, temperature, true,
        )) as { nodeId?: string; chapterNum?: number; branch?: string }
      } else {
        if (allowOverride) {
          message.warning('跳过计划硬闸的绑定未就绪（CreateChapterWithOverride），本次仍走后端硬闸判定')
        }
        result = (await app.CreateChapter(freshSetting, '', plotReq, overwriteChapter, branchFromID, selectedSkill || '', minWords, temperature)) as { nodeId?: string; chapterNum?: number; branch?: string }
      }
      // 预创建节点已由后端同步完成，立即激活；记录章节号供停止按钮
      const nodeId = result?.nodeId
      const chapNum = result?.chapterNum
      streamTargetRef.current = { chapterNum: chapNum || 0, branch: result?.branch || '' }
      if (nodeId) {
        const store = useOutlineStore.getState()
        if (!store.outlines.find(n => n.id === nodeId)) {
          store.setOutlines([...store.outlines, { id: nodeId, order_index: chapNum, title: `第${chapNum}章`, status: 'writing', parent_id: '', summary: '' } as OutlineNode])
        }
        setActiveId(nodeId)
        setContent('')
      }
    } catch (err: unknown) {
      finishStream()
      message.error(err instanceof Error ? err.message : '生成失败')
    }
  }

  /**
   * 章节计划硬闸（刀1 线D）：生成前调 NovelChapterGatePrecheck；blocking 且作者未显式
   * 覆盖时不发起生成，改为弹「立即生成计划草案 / 仍然生成（跳过硬闸）」。目标章号与
   * 后端 resolveTargetChapterNum 同源（internal/app/create_chapter_handler.go:654）：
   * 显式章号 > 分支父节点章号 > 顺延新章。
   *
   * v4.450.0 三点打通：blocking 时把本次 plotReq 存为计划创作方向种子、记录预检
   * 目标章号——「立即生成计划草案」打开的计划卡因此带着分支意向、且章号与预检一致。
   *
   * 预检自身不可用（绑定未就绪 / 读取失败）**不拦生成**——硬闸只拦「没有抓手」，
   * 不因预检故障把作者锁死；但如实告知本次未做检查（诚实降级，不假装已预检）。
   */
  const guardPlanGate = async (plotReq: string, overwriteChapter: number, branchFromID: string, proceed: () => void) => {
    // 新章传 0：后端预检按磁盘大纲解析真实目标章（前端 store 可能空/过期）
    const target = overwriteChapter > 0
      ? overwriteChapter
      : branchFromID
        ? (useOutlineStore.getState().outlines.find(n => n.id === branchFromID)?.order_index || 0)
        : 0
    let report: PlanGateReportView | null = null
    try {
      if (typeof app.NovelChapterGatePrecheck !== 'function') {
        throw new Error('章节计划预检接口未就绪')
      }
      report = await app.NovelChapterGatePrecheck(target)
    } catch (err: unknown) {
      message.warning(`章节计划预检未执行（${err instanceof Error ? err.message : String(err)}），本次生成未做硬闸检查`)
      proceed()
      return
    }
    if (report === null || !report.blocking) { proceed(); return }
    setPlanSeed(plotReq.trim())
    setPlanChapterOverride(report.chapterNum || target)
    setPlanGate({
      chapterNum: report.chapterNum || target,
      missing: report.missing ?? [],
      problems: [...(report.planProblems ?? []), ...(report.outlineIssues ?? [])].map(p => `[${p.severity}] ${p.message}`),
      proceed,
    })
  }

  /**
   * 生成入口（含未保存保护，v4.421.0）：生成会清空编辑区，dirty 时先让作者选择
   * 「先保存 / 放弃修改 / 取消」——旧实现直接 `setContent('')`，手写正文静默丢失。
   * 刀1 线D：脏保护通过后再过章节计划硬闸（预检 → 允许 / 弹窗二选）。
   */
  const startGeneration = (plotReq: string, overwriteChapter = 0, branchFromID = '') => {
    // A7：在途早退此前是静默 return——作者点了「生成」却毫无反馈，只会反复点。
    if (generatingRef.current) { message.warning('正在生成，请先停止生成再开始新的生成'); return }
    if (!plotReq.trim()) { message.warning('请选择分支或输入剧情要求'); return }
    const run = () => { void runGeneration(plotReq, overwriteChapter, branchFromID) }
    const gated = () => { void guardPlanGate(plotReq, overwriteChapter, branchFromID, run) }
    if (contentRef.current !== loadedSnapshotRef.current) {
      chooseUnsavedAction({
        title: '正文有未保存的修改',
        message: '开始生成会清空编辑区（生成完成后新正文自动落盘）。',
        onSave: () => { void saveActiveRef.current().then((ok) => { if (ok) gated() }) },
        onDiscard: gated,
      })
      return
    }
    gated()
  }

  // CancelCreateChapter 契约（后端批 1）：wails 再生成的 NovelB 绑定已提供类型化签名，直接调用。

  // 停止生成：取 CreateChapter 返回值中的章节号 + 主线分支 ''（T6-7.2）
  const handleStop = async () => {
    const { chapterNum, branch } = streamTargetRef.current
    if (!chapterNum) { message.warning('暂无进行中的生成任务'); return }
    setStopping(true)
    try {
      const cancelled = await CancelCreateChapter(chapterNum, branch)
      if (cancelled) { setGenPhase('正在停止…') } // 随后收到 cancelled 事件收尾（保留部分正文）
      else { finishStream(); message.info('生成已结束') } // 幂等 false：本地兜底，避免 UI 悬挂
    } catch (err: unknown) {
      setStopping(false)
      message.error(err instanceof Error ? err.message : '取消失败')
    }
  }

  // 直接生成下一章（跳过分支向导）
  const handleDirectGenerate = () => {
    if (!directPlot.trim()) { message.warning('请输入剧情要求'); return }
    const chapNum = activeNode?.order_index || 0
    // 当前章从未生成过（status==='planned'）→ 直接生成它本人，而不是「下一章」；
    // chapter_file 不能当判据（生成开始时就写入，中断/已完成都有，误判会覆盖旧章
    // ——用户实测：第1章被覆盖）。生成过的节点（writing/done）走下一章语义。
    if (chapNum > 0) {
      const active: OutlineNode | undefined = flatChapters.find(nx => nx.order_index === chapNum)
      if (active && (active.status || 'planned') === 'planned') { startGeneration(directPlot, chapNum, ''); return }
    }
    const next: OutlineNode | undefined = flatChapters.find(nx => nx.order_index === chapNum + 1)
    if (next) {
      // v4.421.0：并列分支不再挂在 Modal.confirm 的 onCancel 上——✕/Esc（用户想退出）
      // 此前会触发「作为分支追加」，等于误开一次 AI 生成。
      chooseAction({
        title: `第${chapNum + 1}章已存在「${next.title || `第${chapNum + 1}章`}」`,
        message: '选择生成方式：',
        options: [
          { label: '覆盖下一章', tone: 'danger', run: () => startGeneration(directPlot, chapNum + 1, '') },
          { label: '作为分支追加', tone: 'primary', run: () => startGeneration(directPlot, 0, next.id) },
        ],
      })
    } else { startGeneration(directPlot, 0, '') } // 0=后端按磁盘大纲顺延下一章
  }

  /**
   * 计划卡「生成本章」（v4.450.0 三点打通的最后一公里）：计划落盘后一键回到生成。
   * 剧情要求取计划自身的情节摘要（生成时后端还会按章号注入本计划），不再让作者
   * 自己走回生成入口；此时硬闸复检应放行（计划已在盘上）。
   */
  const handleGenerateFromPlan = (chapterNum: number, plan: ChapterPlan) => {
    const req = plan.plot_summary.trim() || plan.narrative_goal.trim() || planSeed.trim()
    if (!req) { message.warning('计划缺少情节摘要：先补全计划再生成'); return }
    startGeneration(req, chapterNum, '')
  }

  const handleDelete = (node: OutlineNode) => {
    const deleteNow = async () => {
      try {
        await app.DeleteOutlineNode(node.id)
        if (activeIdRef.current === node.id) { setActiveId(''); setContent(''); setLoadedSnapshot('') }
        await loadOutlines(); refreshStats()
        message.success('已删除')
      } catch (err: unknown) { message.error(err instanceof Error ? err.message : '删除失败') }
    }
    // 删除正在编辑的章节会连未保存正文一起清掉——先确认（v4.421.0）
    if (activeIdRef.current === node.id && contentRef.current !== loadedSnapshotRef.current) {
      chooseUnsavedAction({
        title: '正文有未保存的修改',
        message: '删除本章会连同未保存的正文一起清除。',
        onSave: () => { void saveActiveRef.current().then((ok) => { if (ok) void deleteNow() }) },
        onDiscard: () => { void deleteNow() },
      })
      return
    }
    void deleteNow()
  }

  const handleRegenerate = (node: OutlineNode) => openWizard((node.order_index || 1) - 1, node.order_index || 1)

  /**
   * 保存当前章正文；返回是否成功。失败/无内容都有可见反馈（旧实现静默 return，
   * 作者点了保存却不知道发生了什么）。走 ref 读最新值，供确认弹窗的异步回调复用。
   */
  const saveActive = useCallback(async (): Promise<boolean> => {
    // A4：载入在途时 activeId 已是新章、content 还是上一章正文——保存会把上一章正文写进
    // 新章文件（旧实现静默执行）。返回 false 是既有契约：chooseUnsavedAction 的 onSave
    // 依赖它中止后续动作（如切章/生成），不会「保存失败却继续切走」。
    if (chapterLoadingRef.current) { message.warning('正在载入章节，请稍候再保存'); return false }
    const text = contentRef.current
    const node = useOutlineStore.getState().outlines.find(n => n.id === activeIdRef.current)
    if (!node) { message.warning('请先在左侧选择要保存的章节'); return false }
    if (!text.trim()) { message.warning('正文为空，无需保存'); return false }
    setSaving(true)
    try {
      const branch = node.branch || ''
      await app.SaveChapterBranchContent(node.order_index || 1, branch, text)
      setLoadedSnapshot(text)
      message.success('已保存')
      return true
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '保存失败')
      return false
    } finally {
      setSaving(false)
    }
  }, [])
  saveActiveRef.current = saveActive

  const handleSave = () => { void saveActive() }

  /** 打开整章重写前先保护未保存正文（重写结果应用后会整章覆盖）。 */
  const openRewriteModal = useCallback(() => {
    // A6：生成中重写＝同一章上两个写者（半章 / 全文谁最后到谁生效，取决于到达顺序）。
    if (generatingRef.current) { message.warning('正在生成，请先停止生成再重写本章'); return }
    if (contentRef.current !== loadedSnapshotRef.current) {
      chooseUnsavedAction({
        title: '正文有未保存的修改',
        message: '重写结果应用后会覆盖本章正文，建议先保存当前修改。',
        onSave: () => { void saveActiveRef.current().then((ok) => { if (ok) setRwOpen(true) }) },
        onDiscard: () => setRwOpen(true),
      })
      return
    }
    setRwOpen(true)
  }, [])

  /**
   * A2a：「重写历史」入口同 A1——应用版本 / 恢复原文都会写回本章正文，此前**入口无脏检查**
   * （对照整章重写的 openRewriteModal 早已有），审计确证为漏网点。取消（含 ✕/Esc）＝不打开。
   */
  const openRewriteHistory = useCallback(() => {
    if (generatingRef.current) { message.warning('正在生成，请先停止生成再查看重写历史'); return }
    if (contentRef.current !== loadedSnapshotRef.current) {
      chooseUnsavedAction({
        title: '正文有未保存的修改',
        message: '在重写历史里应用版本 / 恢复原文都会写回本章正文，建议先保存当前修改。',
        onSave: () => { void saveActiveRef.current().then((ok) => { if (ok) setRwHistOpen(true) }) },
        onDiscard: () => setRwHistOpen(true),
      })
      return
    }
    setRwHistOpen(true)
  }, [])

  /**
   * A1：打开局部重写前过同一道脏闸。局部重写的选区坐标取自编辑器缓冲，后端却按**磁盘**
   * 正文定位——`saveActive` 之后两者才对齐，故「先保存」是这里的主推动作。
   * 「放弃修改」＝不先保存直接发起：本地未保存正文仍留在编辑区（重写结果不覆盖它，
   * 见 refreshEditorAfterRewriteApplied），所以这不是一次真实的丢弃。
   */
  const openPartialRewrite = useCallback((selection: { start: number; end: number; text: string }) => {
    if (generatingRef.current) { message.warning('正在生成，请先停止生成再局部重写'); return }
    if (contentRef.current !== loadedSnapshotRef.current) {
      chooseUnsavedAction({
        title: '正文有未保存的修改',
        message: '局部重写按磁盘正文定位选区：先保存可让选区与磁盘正文对齐；不保存则按当前编辑区选区发起，编辑区里的本地修改仍会保留。',
        onSave: () => { void saveActiveRef.current().then((ok) => { if (ok) setPSel(selection) }) },
        onDiscard: () => setPSel(selection),
      })
      return
    }
    setPSel(selection)
  }, [])

  // 节点后生成下一章/分支（含覆盖确认）
  const handleAddNext = (node: OutlineNode) => {
    const chapNum = node.order_index || 1
    const next = outlines.find(nx => nx.order_index === chapNum + 1 && !nx.parent_id)
    if (next) {
      // 同 handleDirectGenerate：并列分支显式成按钮，取消就是取消（v4.421.0）
      chooseAction({
        title: `第${chapNum}章后已有「${next.title || `第${chapNum + 1}章`}」`,
        message: '选择生成方式：',
        options: [
          { label: '覆盖下一章', tone: 'danger', run: () => openWizard(chapNum, chapNum + 1) },
          { label: '末尾追加', tone: 'primary', run: () => openWizard(chapNum, 0, node.id) },
        ],
      })
    } else { openWizard(chapNum, 0, '') }
  }

  const flatNodes = flattenTree(buildTree(outlines))
  const activeNode = outlines.find(n => n.id === activeId)
  // 章节点全树展平 + 下一章计算：纯函数在 outlineTree.ts（卷结构/未写判据的
  // 两次实弹教训都沉淀在那里并有单测）
  const flatChapters = useMemo(() => flattenChapters(outlines), [outlines])
  const maxChapterOrder = flatChapters.length ? flatChapters[flatChapters.length - 1].order_index || 0 : 0
  const nextMainChapterNum = maxChapterOrder + 1
  const lastMainChapter = maxChapterOrder

  // ── 叙事状态账本（作者审批制）──
  const activeChapterNum = activeNode?.order_index || lastMainChapter
  // 计划卡章号（刀1 线D）：未选章时取「下一章」——与硬闸预检同源
  // （guardPlanGate 的目标章号），否则卡里显示「先选章节」而生成却被拦，作者无处补计划。
  // v4.450.0：硬闸弹窗「立即生成计划草案」以预检解析出的目标章号覆盖（active 停在
  // 上一章时，card 不能停在已写章上而生成目标是下一章）。
  const planChapterNum = planChapterOverride || (activeChapterNum > 0 ? activeChapterNum : nextMainChapterNum)
  // 切章即让位：覆盖章号只在硬闸弹窗打开计划卡的那一次有效
  useEffect(() => { setPlanChapterOverride(0) }, [activeChapterNum])
  // 持久标注高亮（t7 overlay）：本章标注清单随激活章加载传给编辑器镜像；
  // 正文编辑的失效由 EditorPanel 内部 dirty 纪律处理（编辑即整体退场）。
  const [editorAnns, setEditorAnns] = useState<ChapterAnnotation[]>([])
  useEffect(() => {
    if (!activeChapterNum) { setEditorAnns([]); return undefined }
    let alive = true
    app.NovelChapterAnnotations(activeChapterNum)
      .then(a => { if (alive) setEditorAnns(a ?? []) })
      .catch(() => { if (alive) setEditorAnns([]) })
    return () => { alive = false }
  }, [activeChapterNum])
  const loadState = useCallback(async () => {
    setStateBusy(true); setStateMsg('')
    try {
      const s = (await GetNovelState()) as { version?: number; entities?: Record<string, { name?: string; type?: string; status?: string }> }
      setNovelState(s); setStateMsg('已加载叙事状态账本')
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '加载失败') }
    finally { setStateBusy(false) }
  }, [])
  const proposeState = useCallback(async () => {
    setStateBusy(true); setStateMsg('')
    try {
      const p = await BuildNovelStatePatch(activeChapterNum)
      setStatePatch(p); setStateMsg(`AI 已生成第 ${activeChapterNum} 章状态建议，等待你审批`)
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '生成建议失败') }
    finally { setStateBusy(false) }
  }, [activeChapterNum])
  const approveState = useCallback(async () => {
    if (statePatch === null || statePatch === undefined) { setStateMsg('请先生成 AI 状态建议'); return }
    setStateBusy(true); setStateMsg('')
    try {
      const r = (await SettleNovelState(JSON.stringify(statePatch), true)) as { version?: number }
      setStateMsg(`已审批结算，状态账本版本 → ${r?.version ?? '?'}`)
      await loadState()
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '结算失败') }
    finally { setStateBusy(false) }
  }, [statePatch, loadState])
  // 手动「一键去味」：对当前章节跑确定性 DeSlopRewrite（任意章节可用，不只生成时）。
  // v4.421.0：成功后刷新编辑区（旧实现只更新提示，作者看到的仍是旧文，以为去味没生效）；
  // 若编辑区有未保存的手写修改则不刷新，如实提示，避免把作者的字覆盖掉。
  const refreshEditorAfterServerRewrite = useCallback(() => {
    const node = useOutlineStore.getState().outlines.find(n => n.id === activeIdRef.current)
    if (!node) return
    if (contentRef.current !== loadedSnapshotRef.current) {
      message.info('章节已在磁盘更新；编辑区有未保存修改，未刷新显示（保存后重新打开本章可看到新版）')
      return
    }
    void loadChapter(node)
  }, [loadChapter])

  /**
   * A1/A2a：重写类动作「服务端已生效」后的编辑区刷新闸。与 refreshEditorAfterServerRewrite
   * 同纪律（脏则只提示不刷新），区别在文案必须点明**此时保存会覆盖服务端的重写结果**——
   * 旧实现（局部重写 / 重写历史）直接 loadChapter 灌回磁盘正文，作者的未保存手写正文被
   * 覆盖**且 loadedSnapshot 被对齐、脏标志一并抹掉**，此后切章/关标签都不再提示。
   */
  const refreshEditorAfterRewriteApplied = useCallback(() => {
    const node = useOutlineStore.getState().outlines.find(n => n.id === activeIdRef.current)
    if (!node) return
    if (contentRef.current !== loadedSnapshotRef.current) {
      message.warning('重写已在服务端生效；编辑区有未保存的修改，已保留你的本地版本，未刷新（此时保存会覆盖服务端的重写结果）')
      return
    }
    void loadChapter(node)
  }, [loadChapter])

  const deslopChapter = useCallback(async () => {
    // A6：生成中途去味会把**半章**写盘，随后流式 done 再写全文——两写者的最终结果取决于
    // 到达顺序（旧实现不禁用，rail 按钮在生成中照常可点）。rail 上另外加了 disabled 门控，
    // 这里保留早退以覆盖非按钮调用点。
    if (generatingRef.current) { message.warning('正在生成，请先停止生成再去味'); return }
    setStateBusy(true); setStateMsg('')
    try {
      const r = (await DeSlopChapterAiTaste(activeChapterNum)) as { changes?: number; beforeScore?: number; afterScore?: number; done?: boolean }
      setStateMsg(r?.done ? `已去味 ${r.changes} 处，分数 ${r.beforeScore}→${r.afterScore}` : '未命中 AI 套路，无需去味')
      if (r?.done) refreshEditorAfterServerRewrite()
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '去味失败') }
    finally { setStateBusy(false) }
  }, [activeChapterNum, refreshEditorAfterServerRewrite])
  // 高级去味：LLM 受限重写命中句（质量更高，需模型调用）。
  const llmDeslop = useCallback(async () => {
    // A6：同 deslopChapter——生成中改写同一章，两个写者会互相覆盖。
    if (generatingRef.current) { message.warning('正在生成，请先停止生成再去味'); return }
    setStateBusy(true); setStateMsg('')
    try {
      const r = (await RewriteChapterAiTaste(activeChapterNum)) as { done?: boolean; rewritten?: number; beforeScore?: number; afterScore?: number; reason?: string }
      setStateMsg(r?.done ? `已高级去味 ${r.rewritten} 句，分数 ${r.beforeScore}→${r.afterScore}` : (r?.reason ?? '无命中句'))
      if (r?.done) refreshEditorAfterServerRewrite()
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '高级去味失败') }
    finally { setStateBusy(false) }
  }, [activeChapterNum, refreshEditorAfterServerRewrite])

  // ── 文风指纹（参考档构建 + 章节体检；wailsjs 再生前 NovelB 三方法直接 import）──
  // 打开即拉参考档状态（刷新并入 open 时自动加载）；失败落 rail 消息（stateMsg）。
  const openFingerprint = useCallback(async () => {
    setFpOpen(true); setFpBusy(true); setFpMsg(''); setFpScore(null)
    try {
      setFpStatus(await NovelFingerprintStatus())
    } catch (err: unknown) { setFpMsg(err instanceof Error ? err.message : '加载文风指纹失败') }
    finally { setFpBusy(false) }
  }, [])
  // 构建/重建参考档：用全部已写章节生成风格基线（样本不足时后端 reject 中文提示）。
  const buildFingerprint = useCallback(async () => {
    setFpBusy(true); setFpMsg('')
    try {
      const s = await NovelFingerprintBuild()
      setFpStatus(s)
      setFpMsg(`参考档已构建（${s?.chapters ?? 0} 章 · ${(s?.chars ?? 0).toLocaleString()} 字）`)
    } catch (err: unknown) { setFpMsg(err instanceof Error ? err.message : '构建参考档失败') }
    finally { setFpBusy(false) }
  }, [])
  // 体检当前章：AI 味评分 + 与参考档距离（无参考档时后端按通用阈值打分）。
  const scoreFingerprint = useCallback(async () => {
    setFpBusy(true); setFpMsg('')
    try {
      const r = await NovelFingerprintScore(activeChapterNum)
      setFpScore(r)
      setFpMsg(`第 ${activeChapterNum} 章体检完成`)
    } catch (err: unknown) { setFpMsg(err instanceof Error ? err.message : '章节体检失败') }
    finally { setFpBusy(false) }
  }, [activeChapterNum])

  // ── AI 反推大纲（v4.281，拆书导入 P1）：反推当前工程立项 + 分批章节大纲（只读预览），
  // ── 平台评审（v4.282，oh-story 蒸馏 T1）：档位清单（一次拉取）+ 当前章确定性评审。
  // 打开面板即拉档位（失败落 rail 消息，选择器回退为「通用」一档）；评审零模型调用。
  const openReview = useCallback(async () => {
    setReviewOpen(true)
    setReviewMsg('')
    if (reviewPlatforms.length === 0) {
      try {
        const list = await NovelReviewPlatforms()
        setReviewPlatforms(Array.isArray(list) ? list : [])
      } catch (err: unknown) {
        setReviewMsg(err instanceof Error ? err.message : '加载平台档位失败')
      }
    }
  }, [reviewPlatforms.length])
  const runReview = useCallback(async () => {
    setReviewBusy(true)
    setReviewMsg('')
    try {
      const rep = await NovelChapterReview(activeChapterNum, reviewPlatformId)
      setReviewReport(rep)
      const hit = (rep?.counts?.S1 ?? 0) + (rep?.counts?.S2 ?? 0) + (rep?.counts?.S3 ?? 0) + (rep?.counts?.S4 ?? 0)
      setReviewMsg(`第 ${activeChapterNum} 章评审完成：${rep?.platformLabel ?? ''} · ${rep?.verdict ?? ''} · 命中 ${hit} 项`)
    } catch (err: unknown) {
      setReviewMsg(err instanceof Error ? err.message : '评审失败')
    } finally {
      setReviewBusy(false)
    }
  }, [activeChapterNum, reviewPlatformId])

  // ── AI 反推大纲（v4.281，拆书导入 P1）：反推当前工程立项 + 分批章节大纲（只读预览），
  // 确认后按章号合并进大纲（幂等、不覆盖章节正文）；模型不可用时后端自动规则兜底并如实回报。
  const reconstructOutlines = useCallback(async () => {
    setReconstructBusy(true)
    setStateMsg('AI 反推大纲中…')
    try {
      // v4.291 任务化：反推入任务队列（任务中心可见、页面关闭不丢），
      // 轮询取结果；队列不可用回落同步绑定。v4.340：轮询期可取消——
      // 停止等待并请求后端取消任务（任务完成竞态下取消失败被吞，任务中心可查）。
      let preview: Awaited<ReturnType<typeof NovelOutlineReconstruct>> | null = null
      let cancelled = false
      try {
        const started = await NovelOutlineReconstructStart()
        reconstructTaskIdRef.current = started.taskId || ''
        reconstructCancelRef.current = false
        setReconstructCancellable(true)
        const startedAt = Date.now()
        const deadline = startedAt + 12 * 60_000
        for (;;) {
          if (!reconstructAliveRef.current) return
          if (reconstructCancelRef.current) { cancelled = true; break }
          if (Date.now() > deadline) throw new Error('反推任务超时（12 分钟）')
          await new Promise((r) => setTimeout(r, 3000))
          if (!reconstructAliveRef.current) return
          const waited = Math.floor((Date.now() - startedAt) / 1000)
          setReconstructElapsed(waited)
          setStateMsg(`AI 反推大纲中…已等待 ${Math.floor(waited / 60)} 分 ${waited % 60} 秒`)
          if (reconstructCancelRef.current) { cancelled = true; break }
          const st = await NovelOutlineReconstructTaskGet()
          if (!reconstructAliveRef.current) return
          if (st.status === 'failed') throw new Error(st.error || '反推任务失败')
          if (st.status === 'succeeded' && st.preview) {
            preview = st.preview as Awaited<ReturnType<typeof NovelOutlineReconstruct>>
            break
          }
        }
      } catch (taskErr: unknown) {
        const msg = taskErr instanceof Error ? taskErr.message : ''
        const fallbackable = msg.includes('任务队列不可用') || msg.includes('尚无反推任务')
        if (!fallbackable) throw taskErr
        if (!reconstructAliveRef.current) return
        // 回退路径是同步绑定，没有取消面——如实告知，不让「取消反推」按钮假消失
        setStateMsg('任务队列不可用：已回退同步反推（该路径不可取消，完成后会弹确认）')
        preview = await NovelOutlineReconstruct()
        if (!reconstructAliveRef.current) return
      }
      if (cancelled) {
        // 用户取消：停止等待并请求后端协作取消（任务恰已完成时后端报错，
        // 吞掉即可——等待已停，结果可在任务中心看到）。
        const id = reconstructTaskIdRef.current
        if (id) void app.TaskCancel(id).catch(() => {})
        setStateMsg('已取消反推等待；任务已请求取消，可在任务中心查看')
        return
      }
      const items = preview?.items ?? []
      if (items.length === 0) {
        setStateMsg('反推结果为空，未做修改')
        return
      }
      const head = preview?.aiUsed ? 'AI 反推完成' : '模型不可用，已用规则兜底'
      const warn = preview?.warnings?.length ? `\n提示：${preview.warnings.slice(0, 2).join('；')}` : ''
      const meta = [preview?.genre, preview?.narrativePerspective, preview?.targetWords ? `目标 ${preview.targetWords.toLocaleString()} 字` : '']
        .filter(Boolean).join(' · ')
      // 篇幅路由（oh-story T4）：中/长篇预览按每 segmentSize 章聚合为卷级参考节点
      const seg = preview?.segmentSize ?? 0
      const tierLabel = preview?.tier === 'long' ? '长篇' : preview?.tier === 'mid' ? '中篇' : '短篇'
      const action = seg > 0
        ? `篇幅路由（${tierLabel}）：已按每 ${seg} 章聚合为 ${items.length} 个卷级参考节点，应用将新建这些节点（不影响章节正文，可重复执行）。`
        : '将把每章的概要 / 场景 / 要点 / 情感基调合并进大纲（不覆盖章节正文，可重复执行）。'
      Modal.confirm({
        title: '应用 AI 反推大纲？',
        content: `${head}：共 ${items.length} ${seg > 0 ? '个卷级节点' : '章'}。${action}${meta ? `\n推断：${meta}` : ''}${warn}`,
        okText: '应用到大纲',
        cancelText: '取消',
        onOk: async () => {
          const n = await NovelOutlineReconstructApply(JSON.stringify(items))
          setStateMsg(seg > 0 ? `已应用 ${n} 个卷级参考节点` : `已应用 ${n} 章大纲`)
          await loadOutlines()
        },
      })
    } catch (err: unknown) {
      setStateMsg(err instanceof Error ? err.message : 'AI 反推失败')
    } finally {
      setReconstructBusy(false)
      setReconstructCancellable(false)
      setReconstructElapsed(0)
    }
  }, [loadOutlines])

  // tail×反推串联（v4.292）：导入向导 tail 模式完成后由书架派发本事件，
  // 本页自动开跑反推（任务化后台执行）。
  // v4.421.0 门控：本页不可见时**挂起**而不是直接开跑——书架流程里
  // `novel:goto-tab{create}` 与 `novel:auto-reconstruct` 同 tick 连发，
  // 事件到达时 active prop 还没翻转，直接按 active 拦会把正常流程拦掉。
  const pendingReconstructRef = useRef(false)
  React.useEffect(() => {
    if (!active || !pendingReconstructRef.current) return
    pendingReconstructRef.current = false
    void reconstructOutlines()
  }, [active, reconstructOutlines])

  React.useEffect(() => {
    const handler = () => {
      if (activeRef.current) { void reconstructOutlines(); return }
      pendingReconstructRef.current = true
    }
    window.addEventListener('novel:auto-reconstruct', handler)
    return () => window.removeEventListener('novel:auto-reconstruct', handler)
  }, [reconstructOutlines])

  // ── 工具轨分组菜单（简化批）：成员与直钮分工见 RailGroup 位置的设计注释 ──
  const railQualityItems: MenuProps['items'] = [
    { key: 'health', label: '全书体检' },
    { key: 'spine', label: '故事骨架' },
    { key: 'converge', label: '收敛修补', disabled: !activeChapterNum },
    { key: 'ctxinv', label: '上下文清单' },
    { key: 'review', label: '平台评审' },
    { key: 'fingerprint', label: '文风指纹' },
  ]
  const onRailQuality: MenuProps['onClick'] = ({ key }) => {
    if (key === 'health') setHealthOpen(true)
    else if (key === 'spine') setSpineOpen(true)
    else if (key === 'converge') setConvergeOpen(true)
    else if (key === 'ctxinv') setCtxInvOpen(true)
    else if (key === 'review') void openReview()
    else if (key === 'fingerprint') void openFingerprint()
  }

  const railTextItems: MenuProps['items'] = [
    { key: 'llmdeslop', label: '高级去味（AI）', disabled: stateBusy || generating },
    { type: 'divider' },
    { key: 'rewrite', label: '整章重写', disabled: generating },
    { key: 'history', label: '重写历史', disabled: generating },
  ]
  const onRailText: MenuProps['onClick'] = ({ key }) => {
    if (key === 'llmdeslop') void llmDeslop()
    else if (key === 'rewrite') openRewriteModal()
    else if (key === 'history') openRewriteHistory()
  }

  const railStructItems: MenuProps['items'] = [
    {
      key: 'reconstruct',
      label: reconstructBusy && reconstructElapsed > 0 ? `AI 反推大纲（已等待 ${reconstructElapsed}s）` : 'AI 反推大纲',
      disabled: reconstructBusy,
    },
    { key: 'state', label: '叙事状态' },
    { key: 'graph', label: graphBusy ? '全文脑图（构建中…）' : '全文脑图', disabled: graphBusy },
    { key: 'promptws', label: '提示词工坊' },
  ]
  const onRailStruct: MenuProps['onClick'] = ({ key }) => {
    if (key === 'reconstruct') void reconstructOutlines()
    else if (key === 'state') { setStateOpen(true); void loadState() }
    else if (key === 'graph') void openGraph()
    else if (key === 'promptws') setPromptWsOpen(true)
  }

  return (
    <div className="novel-create-root">
      {aiTaste && (
        <div className="novel-create-alert">
          <Alert
            type={aiTaste.score >= 60 ? 'warning' : aiTaste.score >= 35 ? 'info' : 'success'}
            showIcon
            message={`AI 味检测 ${aiTaste.score} 分${aiTaste.score >= 60 ? '（建议去味）' : ''}`}
            description={aiTaste.deSlop?.beforeScore != null && aiTaste.deSlop.afterScore != null
              ? `已去味 ${(aiTaste.deSlop.changes ?? []).length} 处，分数 ${aiTaste.deSlop.beforeScore}→${aiTaste.deSlop.afterScore}`
              : aiTaste.issues.length > 0
                ? [...new Set(aiTaste.issues.map(i => i.reason))].slice(0, 3).join('；')
                : '未命中明显 AI 套路'}
          />
        </div>
      )}
      <div className="novel-create-rail">
        {/* 高频直钮：每章写作循环（计划→分析→去味）零菜单直达 */}
        <Button size="small" onClick={() => { setPlanSeed(''); setPlanChapterOverride(0); setPlanOpen(v => !v) }}>章节计划</Button>
        <Button size="small" onClick={() => setAnalysisOpen(true)}>章节分析</Button>
        <Button size="small" loading={stateBusy} disabled={generating} onClick={() => void deslopChapter()}>一键去味</Button>
        {/* 低频分组菜单：成员清单见 railQualityItems 等定义处 */}
        <Dropdown trigger={['click']} menu={{ items: railQualityItems, onClick: onRailQuality }}>
          <Button size="small" icon={<DownOutlined />} iconPosition="end" aria-label="质检">质检</Button>
        </Dropdown>
        <Dropdown trigger={['click']} menu={{ items: railTextItems, onClick: onRailText }}>
          <Button size="small" icon={<DownOutlined />} iconPosition="end" aria-label="文本">文本</Button>
        </Dropdown>
        <Dropdown trigger={['click']} menu={{ items: railStructItems, onClick: onRailStruct }}>
          <Button size="small" icon={<DownOutlined />} iconPosition="end" aria-label="结构">结构</Button>
        </Dropdown>
        {reconstructCancellable && (
          <Button size="small" danger data-testid="reconstruct-cancel"
            onClick={() => { reconstructCancelRef.current = true }}>取消反推</Button>
        )}
        {stateMsg ? <span className="novel-create-rail-msg">{stateMsg}</span> : null}
      </div>
      {/* 章节计划卡（刀1 线D）：工具轨展开位；章号 = 硬闸覆盖章号 > 当前激活章 > 下一章。
          v4.450.0：direction=硬闸存下的剧情要求种子；onGenerate=「生成本章」出口。 */}
      {planOpen && (
        <div style={{ flexShrink: 0, padding: '0 12px 8px', maxHeight: 340, overflow: 'auto' }}>
          <ChapterPlanCard
            chapterNum={planChapterNum || null}
            disabled={generating}
            onNeedPlan={() => setPlanOpen(true)}
            onPlanSaved={() => { void loadOutlines() }}
            direction={planSeed}
            onGenerate={handleGenerateFromPlan}
          />
        </div>
      )}
      <RewriteModal
        open={rwOpen}
        chapterNum={activeChapterNum || null}
        onClose={() => setRwOpen(false)}
        onApplied={() => { if (activeNode) void loadChapter(activeNode) }}
      />
      <RewriteHistoryPanel
        open={rwHistOpen}
        chapterNum={activeChapterNum || null}
        onClose={() => setRwHistOpen(false)}
        // A2a：应用版本 / 恢复原文都在服务端改了盘，此处再过一道脏闸（脏则保留本地版本）
        onApplied={refreshEditorAfterRewriteApplied}
      />
      <BookHealthPanel open={healthOpen} onClose={() => setHealthOpen(false)} />
      <StorySpinePanel open={spineOpen} onClose={() => setSpineOpen(false)} />
      <ConvergeModal open={convergeOpen} chapterNum={activeChapterNum ? activeChapterNum : null} onClose={() => setConvergeOpen(false)} onFinished={() => { if (activeNode) void loadChapter(activeNode as typeof activeNode) }} />
      <ContextInventoryPanel open={ctxInvOpen} chapterNum={activeChapterNum || 0} onClose={() => setCtxInvOpen(false)} />
      <ChapterAnalysisPanel
        open={analysisOpen}
        onClose={() => { setAnalysisOpen(false); setGateChapter(null); setGateContent('') }}
        chapterNum={gateChapter ?? (activeChapterNum || null)}
        content={gateContent || content}
        onLocate={gateChapter == null || gateChapter === activeChapterNum ? handleLocateInEditor : undefined}
        chapterOptions={flatNodes.map(tn => tn.node.order_index)}
      />
      <PromptWorkshopPanel
        open={promptWsOpen}
        onClose={() => setPromptWsOpen(false)}
      />
      <PartialRewriteModal
        open={!!pSel}
        chapterNum={activeChapterNum || null}
        selection={pSel}
        onClose={() => setPSel(null)}
        // A1：重写已在服务端生效——脏时不重载（旧实现在这里无条件 loadChapter，把作者
        // 未保存的手写正文连同脏标志一起抹掉）
        onApplied={refreshEditorAfterRewriteApplied}
      />

      <div className="novel-workspace">
      <ChapterTreePanel flatNodes={flatNodes} activeId={activeId} nextChapterNum={nextMainChapterNum}
        onSelect={selectChapter} onRegenerate={handleRegenerate} onDelete={handleDelete}
        onAddNext={handleAddNext} onGenerateNext={() => openWizard(lastMainChapter)} />
      <div className="v3-grip" aria-hidden="true" />
      <EditorPanel ref={editorRef} activeNode={activeNode ?? null} content={content} onContentChange={setContent}
        annotations={editorAnns}
        chapterLoading={chapterLoading}
        generating={generating} genPhase={genPhase} genPercent={genPercent} stopping={stopping} saving={saving}
        onRegenerate={() => activeNode && handleRegenerate(activeNode)} onSave={handleSave} onStop={handleStop}
        hasChapters={flatNodes.length > 0} nextChapterNum={nextMainChapterNum} onOpenWizard={openWizard}
        editorFontSize={editorFontSize} onEditorFontSizeChange={handleEditorFontSizeChange}
        onPartialRewrite={openPartialRewrite} />
      <div className="v3-grip" aria-hidden="true" />
      <CreateInspector setting={setting} onRefreshSetting={() => void refreshSetting()}
        selectedSkill={selectedSkill} onSelectSkill={(v) => setSelectedSkill(v)}
        minWords={minWords} onMinWordsChange={setMinWords}
        temperature={temperature} onTemperatureChange={setTemperature}
        directPlot={directPlot} onDirectPlotChange={setDirectPlot}
        onDirectGenerate={handleDirectGenerate} onOpenWizard={openWizard}
        prevChapterHint={activeNode?.order_index || lastMainChapter}
        stats={stats} chapterCount={flatNodes.length} />
      </div>
      {/* A8：本弹窗常驻挂载，事件通道一直在听——不按 active 门控就会在别的子页上盖出遮罩 */}
      <NewCharactersModal active={active} />
      <BranchWizardModal open={!!wizard}
        overwriteChapter={wizard?.overwriteChapter ?? 0}
        branchFromID={wizard?.branchFromID ?? ''} characters={wizardCast}
        libraryCharacters={wizardLibCast} prevChapterCast={wizardPrevCast}
        cast={wizardCastSelection} onCastChange={setWizardCastSelection}
        preloadedBranches={wizardBranches}
        onStartBrainstorm={startBrainstorm}
        onClose={() => setWizard(null)}
        onStart={startGeneration} />
      <Modal
        title="本章尚未制定章节计划"
        rootClassName="novel-plan-gate-modal"
        open={planGate !== null}
        onCancel={() => setPlanGate(null)}
        footer={[
          <Button key="cancel" size="small" onClick={() => setPlanGate(null)}>取消</Button>,
          <Button key="override" size="small" danger
            onClick={() => {
              // 覆盖意图：本次生成跳过硬闸（走 NovelB.CreateChapterWithOverride 的
              // allowOverride=true 下发；缺绑定时如实警告并回落普通入口）
              const go = planGate?.proceed
              planOverrideRef.current = true
              setPlanGate(null)
              go?.()
            }}>仍然生成（跳过硬闸）</Button>,
          <Button key="plan" size="small" type="primary"
            onClick={() => {
              setPlanGate(null)
              setPlanOpen(true)
              message.info('已展开章节计划卡，请点「生成计划草案」')
            }}>立即生成计划草案</Button>,
        ]}
      >
        <div style={{ fontSize: 13 }}>
          {planGate && planGate.missing.length > 0
            ? `本章尚未制定计划（缺失：${planGate.missing.join('、')}），先去补计划。`
            : '本章尚未制定计划，先去补计划。'}
          {planGate && planGate.problems.length > 0 && (
            <ul style={{ margin: '6px 0 0 16px', padding: 0, fontSize: 12 }}>
              {planGate.problems.map((t, i) => <li key={i}>{t}</li>)}
            </ul>
          )}
          <div style={{ marginTop: 8, fontSize: 12, color: 'var(--color-text-secondary)' }}>
            硬闸只拦「没有抓手」，不评判计划质量；「仍然生成」会跳过本次预检。
          </div>
        </div>
      </Modal>
      <Modal
        title="叙事状态账本（作者审批制）"
        open={stateOpen}
        onCancel={() => setStateOpen(false)}
        footer={[
          <Button key="load" size="small" loading={stateBusy} onClick={() => void loadState()}>刷新</Button>,
          <Button key="propose" size="small" loading={stateBusy} onClick={() => void proposeState()}>AI 生成状态建议</Button>,
          <Button key="approve" size="small" type="primary" loading={stateBusy} onClick={() => void approveState()}>批准结算</Button>,
        ]}
      >
        <div style={{ fontSize: 12, marginBottom: 8 }}>
          版本 {novelState?.version ?? '—'} · 实体 {Object.keys(novelState?.entities ?? {}).length} 个。AI 只会生成「状态建议」，
          必须你点「批准结算」才写入账本（对应「作者是上帝」）。
        </div>
        <div style={{ maxHeight: 240, overflow: 'auto', fontFamily: 'monospace', fontSize: 12 }}>
          {Object.entries(novelState?.entities ?? {}).map(([id, e]) => (
            <div key={id} style={{ borderBottom: '1px solid var(--border-subtle)', padding: '2px 0' }}>
              {id} · {e?.name ?? ''} · {e?.type ?? ''} · <b>{e?.status ?? ''}</b>
            </div>
          )).slice(0, 40)}
        </div>
      </Modal>
      <Modal
        title="全文脑图（实体关系）"
        open={graphOpen}
        onCancel={() => setGraphOpen(false)}
        footer={[
          <Button key="refresh" size="small" loading={graphBusy} onClick={() => void openGraph()}>刷新</Button>,
          <Button key="close" size="small" onClick={() => setGraphOpen(false)}>关闭</Button>,
        ]}
        width={720}
      >
        {graphMsg && (
          <div style={{ fontSize: 12, marginBottom: 8, color: 'var(--color-text-secondary)' }}>
            {graphMsg}
          </div>
        )}
        {graphBusy ? (
          <div style={{ padding: '48px 0', textAlign: 'center', color: 'var(--color-text-secondary)' }}>加载实体关系…</div>
        ) : graphData?.nodes?.length ? (
          <EntityGraphSvg graph={graphData} />
        ) : (
          <Alert type="info" showIcon message="暂无实体" description="尚未提取到可展示的实体关系。" />
        )}
      </Modal>
      <StyleFingerprintPanel
        open={fpOpen}
        onClose={() => setFpOpen(false)}
        busy={fpBusy}
        msg={fpMsg}
        status={fpStatus}
        score={fpScore}
        onBuild={() => void buildFingerprint()}
        onScore={() => void scoreFingerprint()}
        hasChapter={activeChapterNum > 0}
      />
      <ChapterReviewPanel
        open={reviewOpen}
        onClose={() => setReviewOpen(false)}
        busy={reviewBusy}
        msg={reviewMsg}
        platforms={reviewPlatforms}
        platformId={reviewPlatformId}
        onPlatformChange={(id) => { setReviewPlatformId(id); setReviewReport(null) }}
        report={reviewReport}
        onReview={() => void runReview()}
        hasChapter={activeChapterNum > 0}
      />
    </div>
  )
}

export default CreatePage
