import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Alert, Button, message, Modal } from 'antd'
import { app } from '../gaea/lib/bridge'
import { GetNovelState, BuildNovelStatePatch, SettleNovelState, DeSlopChapterAiTaste, RewriteChapterAiTaste, GetEntityRelations, CancelCreateChapter, NovelFingerprintStatus, NovelFingerprintBuild, NovelFingerprintScore, NovelOutlineReconstruct, NovelOutlineReconstructApply, NovelOutlineReconstructStart, NovelOutlineReconstructTaskGet, NovelReviewPlatforms, NovelChapterReview } from '../../wailsjs/go/app/NovelB'
import { useOutlineStore } from '../stores/outlineStore'
import { useAppStore } from '../stores/appStore'
import type { OutlineNode } from '../types'
import { buildTree, flattenTree, buildPrevSummary } from '../components/novel/create/outlineTree'
import { useChapterStream } from '../components/novel/create/useChapterStream'
import { useChapterGateNotice } from '../components/novel/create/useChapterGateNotice'
import type { AiTasteResult } from '../components/novel/create/chapterStreamTypes'
import type { ChapterReviewPayload, FingerprintScorePayload, FingerprintStatusPayload, ReviewPlatform, ChapterAnnotation } from '../gaea/lib/bridge/novel'
import StyleFingerprintPanel from '../components/novel/StyleFingerprintPanel'
import ChapterReviewPanel from '../components/novel/ChapterReviewPanel'
import ChapterTreePanel from '../components/novel/create/ChapterTreePanel'
import RewriteModal from '../components/novel/RewriteModal'
import PartialRewriteModal from '../components/novel/PartialRewriteModal'
import RewriteHistoryPanel from '../components/novel/RewriteHistoryPanel'
import PromptWorkshopPanel from '../components/novel/PromptWorkshopPanel'
import ChapterAnalysisPanel from '../components/novel/ChapterAnalysisPanel'
import BookHealthPanel from '../components/novel/BookHealthPanel'
import EditorPanel from '../components/novel/create/EditorPanel'
import type { EditorPanelHandle } from '../components/novel/create/EditorPanel'
import CreateInspector from '../components/novel/create/CreateInspector'
import NewCharactersModal from '../components/novel/create/NewCharactersModal'
import BranchWizardModal, { type Branch } from '../components/novel/create/BranchWizardModal'

interface WizardRequest { prevChapter: number; overwriteChapter: number; branchFromID: string }
const BRAINSTORM_MSG_KEY = 'novel-brainstorm-loading'

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

// T6-7.5 拆分后的编排层（≤300 行）：持有页面状态与生成编排；视图拆到 5 个子组件；
// 流式事件经 useChapterStream + chapterStreamTypes 判别联合分发（T6-7.2 停止按钮 + cancelled）。
const CreatePage: React.FC = () => {
  const [setting, setSetting] = useState('')
  const outlines = useOutlineStore(s => s.outlines)
  const loadOutlines = useOutlineStore(s => s.loadOutlines)
  const projectPath = useAppStore(s => s.projectPath)

  const [activeId, setActiveId] = useState('')
  const [content, setContent] = useState('')

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

  const [minWords, setMinWords] = useState(5000)
  const [temperature, setTemperature] = useState(0)
  const [directPlot, setDirectPlot] = useState('')
  const [stats, setStats] = useState<{ totalWords: number; chapterCount: number } | null>(null)
  // 整章重写弹窗（t4-C3 前端消费）
  const [rwOpen, setRwOpen] = useState(false)
  // 重写版本历史面板（t4-C3 余项：版本库 List/Get 前端消费）
  const [rwHistOpen, setRwHistOpen] = useState(false)
  // 提示词工坊面板（t6 首刀：模板可编辑覆盖层；打开才拉一次）
  const [promptWsOpen, setPromptWsOpen] = useState(false)
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
  // 局部重写选区（t4-C3 余项：EditorPanel 选段回调 → PartialRewriteModal；rune 偏移）
  const [pSel, setPSel] = useState<{ start: number; end: number; text: string } | null>(null)
  // 默认启用 story-deslop 去 AI 味润色技能；可在右侧创作设置中切换或清空
  const [selectedSkill, setSelectedSkill] = useState<string | undefined>('story-deslop')
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

  const settingLoadToken = useRef(0)
  const chapterLoadToken = useRef(0)
  const generatingRef = useRef(false)
  const brainstormingRef = useRef(false)

  const handleEditorFontSizeChange = useCallback((v: number) => {
    setEditorFontSize(v)
    try { localStorage.setItem('gaea.novel.editorFontSize', String(v)) } catch { /* 持久化失败不影响使用 */ }
  }, [])

  // 当前生成任务的章节标识（取 CreateChapter 返回值，供停止按钮 CancelCreateChapter）
  const streamTargetRef = useRef({ chapterNum: 0, branch: '' })

  const { attach, detach } = useChapterStream()

  // 拉取最新小说设定；projectPath 变化（切换小说）时重新拉取
  const refreshSetting = useCallback(async () => {
    const token = ++settingLoadToken.current
    const requestedPath = useAppStore.getState().projectPath
    let fresh = ''
    try { fresh = await app.GetWorldview() || '' } catch { /* 设定拉取失败按空处理 */ }
    if (token !== settingLoadToken.current || requestedPath !== useAppStore.getState().projectPath) return ''
    setSetting(fresh)
    return fresh
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
  }, [projectPath, loadOutlines, refreshSetting, refreshStats])

  // 切换小说时清空编辑区与选中节点，避免展示上一个项目的内容
  useEffect(() => { setActiveId(''); setContent('') }, [projectPath])

  const selectChapter = async (node: OutlineNode) => {
    const token = ++chapterLoadToken.current
    const requestedPath = useAppStore.getState().projectPath
    setActiveId(node.id); setChapterLoading(true)
    try {
      const branch = node.branch || ''
      const result = await app.GetChapterBranch(node.order_index || 1, branch)
      if (token === chapterLoadToken.current && requestedPath === useAppStore.getState().projectPath) setContent((result?.content as string) || '')
    } catch {
      if (token === chapterLoadToken.current && requestedPath === useAppStore.getState().projectPath) setContent('')
    } finally {
      if (token === chapterLoadToken.current) setChapterLoading(false)
    }
  }

  // 向导拉取 AI 构思分支（注入最新设定与前文摘要）
  const fetchWizardBranches = useCallback(async (prevChapter: number): Promise<Branch[]> => {
    const freshSetting = await refreshSetting()
    const prevSummary = prevChapter > 0 ? buildPrevSummary(outlines, prevChapter) : ''
    const res = (await app.QuickBrainstormBranches(freshSetting, prevSummary || '')) as { branches?: Array<{ title?: string; summary?: string }> }
    const list = res?.branches || []
    return list.map((b: { title?: string; summary?: string }) => ({ title: b.title ?? '', pitch: b.summary ?? '' }))
  }, [refreshSetting, outlines])

  // 后台构思剧情分支：不弹阻塞弹窗，构思完成后弹窗确认选择
  const openWizard = useCallback(async (prevChapter: number, overwriteChapter = 0, branchFromID = '') => {
    if (brainstormingRef.current) return
    brainstormingRef.current = true
    setWizard({ prevChapter, overwriteChapter, branchFromID })
    setWizardBranches([])
    message.open({ key: BRAINSTORM_MSG_KEY, content: 'AI 正在构思剧情分支，你可以继续操作…', duration: 0 })
    try {
      const list = await fetchWizardBranches(prevChapter)
      if (list.length === 0) {
        setWizard(null)
        message.warning('AI 未构思出剧情分支，可直接输入剧情要求')
        return
      }
      setWizardBranches(list)
    } catch (err: unknown) {
      setWizard(null)
      message.error(err instanceof Error ? err.message : '剧情构思失败，可手动输入剧情要求')
    } finally {
      message.destroy(BRAINSTORM_MSG_KEY)
      brainstormingRef.current = false
    }
  }, [fetchWizardBranches])

  // 流式生成收尾：三路终态（done/error/cancelled）与停止兜底共用
  const finishStream = useCallback(() => {
    setGenPhase(''); setGenPercent(0); setGenerating(false); setStopping(false)
    generatingRef.current = false
    detach()
  }, [detach])

  // 直接开始生成：注册流式监听并调用后端（带目标字数/温度/技能设置）
  const startGeneration = async (plotReq: string, overwriteChapter = 0, branchFromID = '') => {
    if (generatingRef.current) return
    if (!plotReq.trim()) { message.warning('请选择分支或输入剧情要求'); return }
    generatingRef.current = true
    setGenerating(true); setGenPhase('正在生成…'); setGenPercent(0); setContent(''); setStopping(false); setAiTaste(null)

    attach({
      onEvent: (ev) => {
        switch (ev.type) {
          case 'phase':
            if (ev.phase === 'continuing') {
              setGenPhase(`字数不足，正在续写… 第${ev.attempt || 1}次 · ${(ev.current || 0).toLocaleString()}/${(ev.target || minWords).toLocaleString()} 字`)
            } else {
              setGenPhase(`正在生成… 目标 ${(ev.target || minWords).toLocaleString()} 字`)
            }
            break
          case 'chunk':
            setContent((prev) => prev + ev.content)
            setGenPercent(Math.min(100, Math.round(((ev.total || 0) / Math.max(minWords, 1)) * 100)))
            setGenPhase(`正在生成… ${(ev.total || 0).toLocaleString()}/${minWords.toLocaleString()} 字`)
            break
          case 'done': {
            finishStream()
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
            // 取消：后端已落盘部分正文（事件携带 content）；保留编辑器已累积正文可继续编辑/保存
            if (typeof ev.content === 'string' && ev.content.length > 0) setContent(ev.content)
            const chNum = ev.chapterNum || 0
            const branch = ev.branch || ''
            message.info(`${branch ? `第${chNum}${branch}章` : `第${chNum}章`} 已停止生成（已保留 ${(ev.total || 0).toLocaleString()} 字）`)
            refreshStats()
            break
          }
        }
      },
    })

    try {
      // 生成前再读一次最新设定，确保正文提示词注入当前小说设定
      const freshSetting = await refreshSetting()
      if (!freshSetting.trim()) { throw new Error('小说设定为空，请先在「设定」页填写世界观') }
      const result = (await app.CreateChapter(freshSetting, '', plotReq, overwriteChapter, branchFromID, selectedSkill || '', minWords, temperature)) as { nodeId?: string; chapterNum?: number; branch?: string }
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
    const next = outlines.find(nx => nx.order_index === chapNum + 1 && !nx.parent_id)
    if (next) {
      Modal.confirm({
        title: `第${chapNum + 1}章已存在「${next.title || `第${chapNum + 1}章`}」`,
        content: '选择生成方式：', okText: '覆盖下一章', cancelText: '作为分支追加',
        onOk: () => startGeneration(directPlot, chapNum + 1, ''),
        onCancel: () => startGeneration(directPlot, 0, next.id),
      })
    } else { startGeneration(directPlot, 0, '') }
  }

  const handleDelete = async (node: OutlineNode) => {
    try {
      await app.DeleteOutlineNode(node.id)
      if (activeId === node.id) { setActiveId(''); setContent('') }
      await loadOutlines(); refreshStats()
      message.success('已删除')
    } catch (err: unknown) { message.error(err instanceof Error ? err.message : '失败') }
  }

  const handleRegenerate = (node: OutlineNode) => openWizard((node.order_index || 1) - 1, node.order_index || 1)

  const handleSave = async () => {
    const node = outlines.find(n => n.id === activeId)
    if (!node || !content.trim()) return
    setSaving(true)
    try {
      const branch = node.branch || ''
      await app.SaveChapterBranchContent(node.order_index || 1, branch, content)
      message.success('已保存')
    } catch (err: unknown) { message.error(err instanceof Error ? err.message : '失败') }
    finally { setSaving(false) }
  }

  // 节点后生成下一章/分支（含覆盖确认）
  const handleAddNext = (node: OutlineNode) => {
    const chapNum = node.order_index || 1
    const next = outlines.find(nx => nx.order_index === chapNum + 1 && !nx.parent_id)
    if (next) {
      Modal.confirm({
        title: `第${chapNum}章后已有「${next.title || `第${chapNum + 1}章`}」`,
        content: '选择生成方式：', okText: '覆盖下一章', cancelText: '末尾追加',
        onOk: () => openWizard(chapNum, chapNum + 1),
        onCancel: () => openWizard(chapNum, 0, node.id),
      })
    } else { openWizard(chapNum, 0, '') }
  }

  const flatNodes = flattenTree(buildTree(outlines))
  const activeNode = outlines.find(n => n.id === activeId)
  const nextMainChapterNum = Math.max(0, ...outlines.filter(n => !n.parent_id).map(n => n.order_index || 0)) + 1
  const lastMainChapter = nextMainChapterNum - 1

  // ── 叙事状态账本（作者审批制）──
  const activeChapterNum = activeNode?.order_index || lastMainChapter
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
  const deslopChapter = useCallback(async () => {
    setStateBusy(true); setStateMsg('')
    try {
      const r = (await DeSlopChapterAiTaste(activeChapterNum)) as { changes?: number; beforeScore?: number; afterScore?: number; done?: boolean }
      setStateMsg(r?.done ? `已去味 ${r.changes} 处，分数 ${r.beforeScore}→${r.afterScore}` : '未命中 AI 套路，无需去味')
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '去味失败') }
    finally { setStateBusy(false) }
  }, [activeChapterNum])
  // 高级去味：LLM 受限重写命中句（质量更高，需模型调用）。
  const llmDeslop = useCallback(async () => {
    setStateBusy(true); setStateMsg('')
    try {
      const r = (await RewriteChapterAiTaste(activeChapterNum)) as { done?: boolean; rewritten?: number; beforeScore?: number; afterScore?: number; reason?: string }
      setStateMsg(r?.done ? `已高级去味 ${r.rewritten} 句，分数 ${r.beforeScore}→${r.afterScore}` : (r?.reason ?? '无命中句'))
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '高级去味失败') }
    finally { setStateBusy(false) }
  }, [activeChapterNum])

  // ── 文风指纹（参考档构建 + 章节体检；wailsjs 再生前 NovelB 三方法直接 import）──
  // 打开即拉参考档状态（刷新并入 open 时自动加载）；失败落 rail 消息（stateMsg）。
  const openFingerprint = useCallback(async () => {
    setFpOpen(true); setFpBusy(true); setFpMsg(''); setFpScore(null)
    try {
      setFpStatus(await NovelFingerprintStatus())
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '加载文风指纹失败') }
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
        const deadline = Date.now() + 12 * 60_000
        for (;;) {
          if (reconstructCancelRef.current) { cancelled = true; break }
          if (Date.now() > deadline) throw new Error('反推任务超时（12 分钟）')
          await new Promise((r) => setTimeout(r, 3000))
          if (reconstructCancelRef.current) { cancelled = true; break }
          const st = await NovelOutlineReconstructTaskGet()
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
        preview = await NovelOutlineReconstruct()
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
    }
  }, [loadOutlines])

  // tail×反推串联（v4.292）：导入向导 tail 模式完成后由书架派发本事件，
  // 本页自动开跑反推（任务化后台执行）。
  React.useEffect(() => {
    const handler = () => { void reconstructOutlines() }
    window.addEventListener('novel:auto-reconstruct', handler)
    return () => window.removeEventListener('novel:auto-reconstruct', handler)
  }, [reconstructOutlines])

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
        <Button size="small" onClick={() => { setStateOpen(true); void loadState() }}>叙事状态</Button>
        <Button size="small" loading={stateBusy} onClick={() => void deslopChapter()}>一键去味</Button>
        <Button size="small" loading={stateBusy} onClick={() => void llmDeslop()}>高级去味</Button>
        <Button size="small" loading={graphBusy} onClick={() => void openGraph()}>全文脑图</Button>
        <Button size="small" onClick={() => void openFingerprint()}>文风指纹</Button>
        <Button size="small" onClick={() => void openReview()}>平台评审</Button>
        <Button size="small" loading={reconstructBusy} onClick={() => void reconstructOutlines()}>AI 反推大纲</Button>
        {reconstructCancellable && (
          <Button size="small" danger data-testid="reconstruct-cancel"
            onClick={() => { reconstructCancelRef.current = true }}>取消反推</Button>
        )}
        <Button size="small" onClick={() => setRwOpen(true)}>整章重写</Button>
        <Button size="small" onClick={() => setRwHistOpen(true)}>重写历史</Button>
        <Button size="small" onClick={() => setPromptWsOpen(true)}>提示词工坊</Button>
        <Button size="small" onClick={() => setAnalysisOpen(true)}>章节分析</Button>
        <Button size="small" onClick={() => setHealthOpen(true)}>全书体检</Button>
        {stateMsg ? <span className="novel-create-rail-msg">{stateMsg}</span> : null}
      </div>
      <RewriteModal
        open={rwOpen}
        chapterNum={activeChapterNum || null}
        onClose={() => setRwOpen(false)}
        onApplied={() => { if (activeNode) void selectChapter(activeNode) }}
      />
      <RewriteHistoryPanel
        open={rwHistOpen}
        chapterNum={activeChapterNum || null}
        onClose={() => setRwHistOpen(false)}
        onApplied={() => { if (activeNode) void selectChapter(activeNode) }}
      />
      <BookHealthPanel open={healthOpen} onClose={() => setHealthOpen(false)} />
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
        onApplied={() => { if (activeNode) void selectChapter(activeNode) }}
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
        onPartialRewrite={setPSel} />
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
      <NewCharactersModal />
      <BranchWizardModal open={!!wizard && wizardBranches.length > 0}
        prevChapter={wizard?.prevChapter ?? 0} overwriteChapter={wizard?.overwriteChapter ?? 0}
        branchFromID={wizard?.branchFromID ?? ''} onClose={() => { setWizard(null); setWizardBranches([]) }}
        preloadedBranches={wizardBranches}
        onFetchBranches={fetchWizardBranches} onStart={startGeneration} />
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
