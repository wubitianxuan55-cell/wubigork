import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  ArrowRightOutlined,
  CrownOutlined,
  FolderOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  PlusOutlined,
  SafetyCertificateOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import { Alert, Button, Input, Modal, Tag, Tooltip, message } from 'antd'
import { PantheonComposer } from './pantheon/PantheonComposer'
import { PantheonRecord } from './pantheon/PantheonRecord'
import {
  copyMarkdown,
  downloadJSON,
  downloadMarkdown,
  pantheonApi,
  subscribePantheonEvents,
} from './pantheon/api'
import type {
  PantheonAgentStatus,
  PantheonComposerSeed,
  PantheonEvent,
  PantheonLiveStream,
  PantheonSession,
  PantheonSessionSummary,
  PantheonStartRequest,
} from './pantheon/types'
import './pantheon.css'

const STATUS_LABEL: Record<string, string> = {
  running: '进行中',
  completed: '已小结',
  failed: '失败',
  cancelled: '已终止',
}

const PHASE_LABEL: Record<string, string> = {
  preparing: '准备',
  opening: '开场陈述',
  discussion: '补充论证',
  synthesis: '协作汇总',
  independent: '开场陈述',
  review: '互相澄清',
  challenge: '补充论证',
  verdict: '协作汇总',
  completed: '已完成',
  failed: '失败',
  cancelled: '已终止',
}

type SessionFilter = 'all' | 'running' | 'finished'

const SESSION_FILTERS: Array<{ id: SessionFilter; label: string }> = [
  { id: 'all', label: '全部' },
  { id: 'running', label: '进行中' },
  { id: 'finished', label: '已结束' },
]

const RAIL_COLLAPSED_KEY = 'gaea.pantheon.rail-collapsed'

function readRailCollapsed(): boolean {
  try {
    return window.localStorage.getItem(RAIL_COLLAPSED_KEY) === '1'
  } catch {
    return false
  }
}

function persistRailCollapsed(collapsed: boolean): void {
  try {
    window.localStorage.setItem(RAIL_COLLAPSED_KEY, collapsed ? '1' : '0')
  } catch {
    // A blocked localStorage should never prevent the rail from collapsing.
  }
}

function formatListTime(ms: number): string {
  if (!ms) return ''
  const date = new Date(ms)
  const today = new Date()
  const sameDay = date.toDateString() === today.toDateString()
  return sameDay
    ? date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    : date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
}

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error || '未知错误')
}

function workspaceName(path?: string): string {
  if (!path) return ''
  const parts = path.split(/[/\\]/).filter(Boolean)
  return parts[parts.length - 1] || path
}

function clipText(text: string, maxLength: number): string {
  const chars = Array.from(text.trim())
  if (chars.length <= maxLength) return text.trim()
  return `${chars.slice(0, Math.max(0, maxLength - 1)).join('')}…`
}

function exportFilename(session: PantheonSession, extension: 'md' | 'json'): string {
  const stem = Array.from(session.title || session.id)
    .filter((char) => char.charCodeAt(0) >= 32)
    .join('')
    .replace(/[<>:"/\\|?*]/g, '_')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 80)
  return `${stem || session.id}.${extension}`
}

const FALLBACK_SEATS: PantheonAgentStatus[] = [
  { id: 'grok', name: 'Grok', kind: 'Gaea 办公席位', available: false, reason: '等待检测' },
  { id: 'codex', name: 'Codex', kind: 'Codex CLI', available: false, reason: '等待检测' },
  { id: 'zcode', name: 'ZCode', kind: 'ZCode CLI', available: false, reason: '等待检测' },
  { id: 'dsh', name: 'DeepSeek Harness', kind: 'DSH headless', available: false, reason: '等待检测' },
]

const SessionRailItem: React.FC<{
  item: PantheonSessionSummary
  active: boolean
  onClick: () => void
}> = ({ item, active, onClick }) => (
  <button
    type="button"
    className={`pantheon-session-item${active ? ' is-active' : ''}`}
    data-testid={`pantheon-session-${item.id}`}
    onClick={onClick}
  >
    <span className={`pantheon-session-status is-${item.status}`} aria-hidden />
    <span className="pantheon-session-copy">
      <strong>{item.title}</strong>
      {item.workspace && (
        <span className="pantheon-session-workspace" title={item.workspace}>
          <FolderOutlined />
          {workspaceName(item.workspace)}
        </span>
      )}
      <small>
        {item.verdictSummary || item.error || PHASE_LABEL[item.phase] || item.phase}
      </small>
      <span className="pantheon-session-meta">
        {STATUS_LABEL[item.status] || item.status}
        <span>{item.agentCount} 位</span>
      </span>
    </span>
    <time>{formatListTime(item.updatedAt)}</time>
  </button>
)

const PantheonListSkeleton: React.FC = () => (
  <div className="pantheon-skeleton-list" aria-label="正在读取讨论记录">
    <span />
    <span />
    <span />
  </div>
)

const PantheonRecordSkeleton: React.FC = () => (
  <div className="pantheon-record-skeleton" aria-label="正在展开讨论纪要">
    <div className="pantheon-record-skeleton-head">
      <span />
      <span />
    </div>
    <div className="pantheon-record-skeleton-track">
      <span />
      <span />
      <span />
      <span />
    </div>
    <div className="pantheon-record-skeleton-body">
      <span />
      <span />
      <span />
    </div>
  </div>
)

const CouncilProtocol: React.FC = () => (
  <section className="pantheon-protocol" aria-labelledby="pantheon-protocol-title">
    <header className="pantheon-protocol-head">
      <span className="pantheon-eyebrow">讨论原则</span>
      <h3 id="pantheon-protocol-title">在秩序中充分表达，在合作中补强论证。</h3>
    </header>
    <ol className="pantheon-rule-list">
      <li>
        <span className="pantheon-rule-index">01</span>
        <span><b>有序发言</b><small>按指定顺序表达，让前文成为共同讨论基础。</small></span>
      </li>
      <li>
        <span className="pantheon-rule-index">02</span>
        <span><b>多轮补强</b><small>补充证据、澄清假设、修正判断，不重复抬杠。</small></span>
      </li>
      <li>
        <span className="pantheon-rule-index">03</span>
        <span><b>协作综合</b><small>保留合理分歧，整理共识、开放问题与下一步。</small></span>
      </li>
    </ol>
    <div className="pantheon-protocol-boundary">
      <SafetyCertificateOutlined />
      <span>只读运行</span>
      <small>参与者不会修改工作区</small>
    </div>
  </section>
)

const WelcomeCanvas: React.FC<{
  agents: PantheonAgentStatus[]
  onStart: () => void
}> = ({ agents, onStart }) => {
  const seats = agents.length ? agents : FALLBACK_SEATS
  const probing = agents.length === 0
  const available = agents.filter((agent) => agent.available).length
  return (
    <section className="pantheon-welcome" aria-labelledby="pantheon-welcome-title">
      <div className="pantheon-welcome-copy">
        <span className="pantheon-eyebrow">公正、客观、协助、互助</span>
        <h2 id="pantheon-welcome-title">
          让不同判断，<br />在讨论中彼此补强。
        </h2>
        <p>
          多位参与者按序发言、多轮论证，最后形成一份保留共识、分歧与证据缺口的讨论小结。
        </p>
        <div className="pantheon-welcome-actions">
          <Button
            type="primary"
            size="large"
            icon={<PlusOutlined />}
            onClick={onStart}
            data-testid="pantheon-welcome-start"
          >
            发起讨论
          </Button>
          <span className={`pantheon-readiness${probing ? ' is-probing' : ''}`}>
            <i className={!probing && available >= 2 ? 'is-ready' : ''} aria-hidden />
            {probing ? '正在检测本机席位…' : `${available} 位就绪`}
          </span>
        </div>
      </div>

      <div className="pantheon-council-preview" aria-label="本机讨论参与者">
        <div className="pantheon-council-preview-head">
          <div>
            <span className="pantheon-eyebrow">参与者</span>
            <strong>本机 Agent 讨论阵容</strong>
          </div>
          <span>{probing ? '检测中' : `${available} / ${seats.length} 在线`}</span>
        </div>
        <div className="pantheon-seat-map">
          {seats.map((agent, index) => (
            <article
              key={agent.id}
              className={`pantheon-seat-card${agent.available ? ' is-ready' : ''}`}
              style={{ '--seat-order': index } as React.CSSProperties}
            >
              <span className="pantheon-seat-monogram" aria-hidden>
                {agent.name.slice(0, 1).toUpperCase()}
              </span>
              <span className="pantheon-seat-card-copy">
                <strong>{agent.name}</strong>
                <small>{agent.kind}</small>
              </span>
              <span className="pantheon-seat-state">
                {agent.available ? '在线' : probing ? '检测中' : '离线'}
              </span>
            </article>
          ))}
        </div>
        <div className="pantheon-council-foot">
          <span>开场陈述</span>
          <ArrowRightOutlined aria-hidden />
          <span>补充论证</span>
          <ArrowRightOutlined aria-hidden />
          <span>协作综合</span>
        </div>
      </div>
    </section>
  )
}

const PantheonPage: React.FC = () => {
  const [sessions, setSessions] = useState<PantheonSessionSummary[]>([])
  const [agents, setAgents] = useState<PantheonAgentStatus[]>([])
  const [active, setActive] = useState<PantheonSession | null>(null)
  const [showComposer, setShowComposer] = useState(false)
  const [composerSeed, setComposerSeed] = useState<PantheonComposerSeed | null>(null)
  const [loadingList, setLoadingList] = useState(true)
  const [loadingSession, setLoadingSession] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [pathAgent, setPathAgent] = useState<PantheonAgentStatus | null>(null)
  const [pathValue, setPathValue] = useState('')
  const [liveStreams, setLiveStreams] = useState<PantheonLiveStream[]>([])
  const [sessionQuery, setSessionQuery] = useState('')
  const [sessionFilter, setSessionFilter] = useState<SessionFilter>('all')
  const [railCollapsed, setRailCollapsed] = useState(readRailCollapsed)
  const activeIDRef = useRef('')
  const canvasRef = useRef<HTMLElement>(null)
  const runningID = active?.status === 'running' ? active.id : ''
  const activeStatus = active?.status
  const requestSeq = useRef(0)
  const [messageApi, contextHolder] = message.useMessage()
  const filteredSessions = useMemo(() => {
    const query = sessionQuery.trim().toLocaleLowerCase('zh-CN')
    return sessions.filter((item) => {
      if (sessionFilter === 'running' && item.status !== 'running') return false
      if (sessionFilter === 'finished' && item.status === 'running') return false
      if (!query) return true
      return [
        item.title,
        item.workspace,
        item.verdictSummary,
        item.error,
      ].some((value) => value?.toLocaleLowerCase('zh-CN').includes(query))
    })
  }, [sessionFilter, sessionQuery, sessions])
  const sessionFilterCounts = useMemo(() => ({
    all: sessions.length,
    running: sessions.filter((item) => item.status === 'running').length,
    finished: sessions.filter((item) => item.status !== 'running').length,
  }), [sessions])

  useEffect(() => {
    activeIDRef.current = active?.id || ''
  }, [active?.id])

  useEffect(() => {
    const reset = () => {
      const canvas = canvasRef.current
      if (!canvas) return
      canvas.scrollTop = 0
      canvas.scrollLeft = 0
    }
    reset()
    const frame = typeof window.requestAnimationFrame === 'function'
      ? window.requestAnimationFrame(reset)
      : 0
    const timer = window.setTimeout(reset, 100)
    return () => {
      if (frame && typeof window.cancelAnimationFrame === 'function') {
        window.cancelAnimationFrame(frame)
      }
      window.clearTimeout(timer)
    }
  }, [active?.id, composerSeed?.key, showComposer])

  const loadSessions = useCallback(async () => {
    try {
      const list = await pantheonApi.PantheonList()
      setSessions(list)
    } catch (err) {
      setError(`读取讨论记录失败：${errorText(err)}`)
    }
  }, [])

  const loadAgents = useCallback(async () => {
    try {
      const list = await pantheonApi.PantheonAgents()
      setAgents(list)
    } catch (err) {
      setError(`检测本机 Agent 失败：${errorText(err)}`)
    }
  }, [])

  const applyLiveEvent = useCallback((event: PantheonEvent) => {
    if (!event.agentId || !event.sessionId) return
    if (event.kind !== 'agent_delta' && event.kind !== 'agent_output') return
    const id = `${event.sessionId}:${event.agentId}:${event.phase}:${event.round || 0}`
    setLiveStreams((current) => {
      const index = current.findIndex((item) => item.id === id)
      const existing = index >= 0 ? current[index] : undefined
      if (event.kind === 'agent_delta' && existing && !existing.streaming) return current
      const next: PantheonLiveStream = {
        id,
        agentId: event.agentId || '',
        phase: event.phase,
        round: event.round || 0,
        text: event.kind === 'agent_output'
          ? event.text || ''
          : `${existing?.text || ''}${event.text || ''}`,
        streaming: event.kind === 'agent_delta',
      }
      if (index < 0) return [...current.slice(-23), next]
      const updated = [...current]
      updated[index] = next
      return updated
    })
  }, [])

  useEffect(() => {
    let cancelled = false
    void Promise.allSettled([pantheonApi.PantheonList(), pantheonApi.PantheonAgents()])
      .then(([listResult, agentResult]) => {
        if (cancelled) return
        if (listResult.status === 'fulfilled') setSessions(listResult.value)
        else setError(`读取讨论记录失败：${errorText(listResult.reason)}`)
        if (agentResult.status === 'fulfilled') setAgents(agentResult.value)
        else setError((current) => current || `检测本机 Agent 失败：${errorText(agentResult.reason)}`)
      })
      .finally(() => {
        if (!cancelled) setLoadingList(false)
      })
    return () => { cancelled = true }
  }, [])

  const refreshSession = useCallback(async (id: string, quiet = false) => {
    if (!id) return
    const seq = ++requestSeq.current
    if (!quiet) setLoadingSession(true)
    try {
      const session = await pantheonApi.PantheonGet(id)
      if (seq === requestSeq.current && activeIDRef.current === id) {
        setActive(session)
      }
      if (session.status !== 'running') {
        void loadSessions()
      }
    } catch (err) {
      if (seq === requestSeq.current) setError(`读取议事失败：${errorText(err)}`)
    } finally {
      if (!quiet) setLoadingSession(false)
    }
  }, [loadSessions])

  useEffect(() => {
    if (!runningID) return
    const timer = window.setInterval(() => {
      void refreshSession(runningID, true)
    }, 1200)
    return () => window.clearInterval(timer)
  }, [runningID, refreshSession])

  useEffect(() => subscribePantheonEvents((event) => {
    if (event.sessionId && event.sessionId === activeIDRef.current) {
      applyLiveEvent(event)
      if (event.kind === 'agent_delta') return
      void refreshSession(event.sessionId, true)
    } else if (event.kind === 'completed' || event.kind === 'error') {
      void loadSessions()
    }
  }), [applyLiveEvent, refreshSession, loadSessions])

  useEffect(() => {
    if (activeStatus && activeStatus !== 'running') {
      setLiveStreams([])
    }
  }, [activeStatus])

  const openSession = useCallback(async (id: string) => {
    setError('')
    setShowComposer(false)
    setActive(null)
    setLiveStreams([])
    activeIDRef.current = id
    const seq = ++requestSeq.current
    setLoadingSession(true)
    try {
      const session = await pantheonApi.PantheonGet(id)
      if (seq === requestSeq.current) setActive(session)
    } catch (err) {
      if (seq === requestSeq.current) setError(`打开讨论纪要失败：${errorText(err)}`)
    } finally {
      if (seq === requestSeq.current) setLoadingSession(false)
    }
  }, [])

  const openNewCouncil = useCallback(() => {
    setComposerSeed(null)
    setShowComposer(true)
    setActive(null)
    setLiveStreams([])
    setError('')
  }, [])

  const beginCouncil = useCallback(async (request: PantheonStartRequest) => {
    setBusy(true)
    setError('')
    setLiveStreams([])
    try {
      const session = await pantheonApi.PantheonStart(request)
      setActive(session)
      activeIDRef.current = session.id
      setComposerSeed(null)
      setShowComposer(false)
      await loadSessions()
      messageApi.success('圆桌讨论已开始')
    } catch (err) {
      setError(`开始讨论失败：${errorText(err)}`)
    } finally {
      setBusy(false)
    }
  }, [loadSessions, messageApi])

  const cancelCouncil = useCallback(() => {
    if (!active) return
    Modal.confirm({
      title: '终止本次讨论？',
      content: '正在运行的 Agent 会被取消，已经产生的发言会保留。',
      okText: '终止',
      okButtonProps: { danger: true },
      cancelText: '继续讨论',
      onOk: async () => {
        setBusy(true)
        try {
          await pantheonApi.PantheonCancel(active.id)
          await refreshSession(active.id)
          messageApi.warning('已发送终止请求')
        } catch (err) {
          setError(`终止失败：${errorText(err)}`)
        } finally {
          setBusy(false)
        }
      },
    })
  }, [active, messageApi, refreshSession])

  const exportCouncil = useCallback(async () => {
    if (!active) return
    try {
      const markdown = await pantheonApi.PantheonExport(active.id)
      const saved = await downloadMarkdown(exportFilename(active, 'md'), markdown)
      if (saved) messageApi.success('Markdown 议事录已导出')
      else messageApi.info('已取消导出')
    } catch (err) {
      setError(`导出失败：${errorText(err)}`)
    }
  }, [active, messageApi])

  const exportCouncilJson = useCallback(async () => {
    if (!active) return
    try {
      const saved = await downloadJSON(exportFilename(active, 'json'), active)
      if (saved) messageApi.success('JSON 议事录已导出')
      else messageApi.info('已取消导出')
    } catch (err) {
      setError(`导出 JSON 失败：${errorText(err)}`)
    }
  }, [active, messageApi])

  const copyCouncil = useCallback(async () => {
    if (!active) return
    try {
      const markdown = await pantheonApi.PantheonExport(active.id)
      const ok = await copyMarkdown(markdown)
      if (ok) messageApi.success('讨论纪要已复制')
      else messageApi.warning('剪贴板不可用，请使用导出')
    } catch (err) {
      setError(`复制失败：${errorText(err)}`)
    }
  }, [active, messageApi])

  const deleteCouncil = useCallback(() => {
    if (!active) return
    Modal.confirm({
      title: '删除这份讨论纪要？',
      content: '该操作会删除本地保存的全部发言、事件和小结，不能从页面恢复。',
      okText: '删除',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: async () => {
        setBusy(true)
        try {
          await pantheonApi.PantheonDelete(active.id)
          setActive(null)
          setLiveStreams([])
          activeIDRef.current = ''
          setShowComposer(false)
          await loadSessions()
          messageApi.success('讨论纪要已删除')
        } catch (err) {
          setError(`删除失败：${errorText(err)}`)
        } finally {
          setBusy(false)
        }
      },
    })
  }, [active, loadSessions, messageApi])

  const signOffCouncil = useCallback((decision: 'accepted' | 'needs_revision' | 'rejected') => {
    if (!active) return
    const text = decision === 'accepted'
      ? '确认这份讨论小结？'
      : decision === 'needs_revision'
        ? '把本次讨论标记为需要继续？'
        : '暂不采纳这份讨论小结？'
    Modal.confirm({
      title: '确认讨论小结',
      content: text,
      okText: '确认',
      okButtonProps: { danger: decision === 'rejected' },
      cancelText: '取消',
      onOk: async () => {
        setBusy(true)
        try {
          const updated = await pantheonApi.PantheonSignOff(active.id, decision, '')
          setActive(updated)
          await loadSessions()
          messageApi.success('讨论状态已记录')
        } catch (err) {
          setError(`记录讨论状态失败：${errorText(err)}`)
        } finally {
          setBusy(false)
        }
      },
    })
  }, [active, loadSessions, messageApi])

  const continueCouncil = useCallback((session: PantheonSession) => {
    const recommendation = session.verdict?.recommendation
      || '上一轮未形成明确的综合建议。'
    const unresolved = [
      ...(session.verdict?.disagreements || []),
      ...(session.verdict?.risks || []),
    ].slice(0, 6)
    const topic = [
      session.request.topic,
      '',
      '续议说明：',
      `上一轮小结：${recommendation}`,
      unresolved.length
        ? `待继续澄清：${unresolved.join('；')}`
        : '请重点核验上一轮结论的前提、例外与证据强度。',
      '',
      '请承接上一轮结论，补充证据并明确哪些判断应保留、修正或放弃。',
    ].join('\n').trim()

    setComposerSeed({
      key: `follow-up-${session.id}-${Date.now()}`,
      title: `续议：${clipText(session.title, 76)}`,
      topic,
      context: [
        session.request.context,
        `续议依据：上一轮《${session.title}》的讨论小结与保留分歧。`,
      ].filter(Boolean).join('\n\n'),
      successCriteria: session.request.successCriteria,
      workspace: session.request.workspace,
      evidence: session.request.evidence || [],
      agentIds: session.request.agentIds || [],
      chairmanAgentId: session.request.chairmanAgentId,
      rounds: Math.max(2, Math.min(20, session.request.rounds || 2)),
      untilConsensus: Boolean(session.request.untilConsensus),
      maxRounds: session.request.maxRounds,
    })
    setActive(null)
    setShowComposer(true)
    setLiveStreams([])
    setError('')
  }, [])

  const openAgentPath = (agent: PantheonAgentStatus) => {
    setError('')
    setPathAgent(agent)
    setPathValue(agent.executable || '')
  }

  const toggleRail = useCallback(() => {
    setRailCollapsed((current) => {
      const next = !current
      persistRailCollapsed(next)
      return next
    })
  }, [])

  const saveAgentPath = async () => {
    if (!pathAgent) return
    setBusy(true)
    try {
      const updated = await pantheonApi.PantheonSetExecutable(pathAgent.id, pathValue.trim())
      setAgents(updated)
      setPathAgent(null)
      messageApi.success(`${pathAgent.name} 路径已更新`)
    } catch (err) {
      setError(`设置 Agent 路径失败：${errorText(err)}`)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className={`pantheon-shell${railCollapsed ? ' is-rail-collapsed' : ''}`}>
      {contextHolder}
      <aside
        id="pantheon-archive-rail"
        className={`pantheon-rail${railCollapsed ? ' is-collapsed' : ''}`}
        aria-label="讨论档案栏"
      >
        <header className="pantheon-rail-head">
          <div className="pantheon-mark" aria-hidden><CrownOutlined /></div>
          <div className="pantheon-rail-title">
            <span className="pantheon-eyebrow">Gaea Council</span>
            <h1>众神殿</h1>
            <p>多轮讨论与协作纪要</p>
          </div>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            aria-label="新建讨论"
            data-testid="pantheon-new"
            onClick={openNewCouncil}
            className="pantheon-new-button"
          >
            新讨论
          </Button>
        </header>

        <div className="pantheon-rail-meta">
          <span>{agents.filter((agent) => agent.available).length} / {agents.length || 4} 位就绪</span>
          <div className="pantheon-rail-meta-actions">
            <Tag bordered={false} icon={<SafetyCertificateOutlined />}>只读讨论</Tag>
            <span className="pantheon-rail-count" aria-label={`${sessions.length} 场讨论`}>
              {sessions.length}
            </span>
            <Tooltip title={railCollapsed ? '展开讨论档案' : '折叠讨论档案'}>
              <Button
                type="text"
                size="small"
                className="pantheon-rail-toggle"
                icon={railCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                aria-label={railCollapsed ? '展开讨论档案' : '折叠讨论档案'}
                aria-expanded={!railCollapsed}
                aria-controls="pantheon-rail-content"
                data-testid="pantheon-rail-toggle"
                onClick={toggleRail}
              />
            </Tooltip>
          </div>
        </div>

        <div id="pantheon-rail-content" className="pantheon-rail-scroll">
          <div className="pantheon-rail-section-title">
            <span>讨论档案</span>
            <small>{filteredSessions.length}</small>
          </div>
          <div className="pantheon-archive-tools">
            <Input
              size="small"
              allowClear
              value={sessionQuery}
              onChange={(event) => setSessionQuery(event.target.value)}
              prefix={<SearchOutlined />}
              placeholder="搜索议题或工作空间"
              aria-label="搜索讨论档案"
              data-testid="pantheon-session-search"
              className="pantheon-archive-search"
            />
            <div className="pantheon-archive-filters" role="group" aria-label="讨论状态筛选">
              {SESSION_FILTERS.map((filter) => (
                <button
                  key={filter.id}
                  type="button"
                  data-testid={`pantheon-filter-${filter.id}`}
                  aria-pressed={sessionFilter === filter.id}
                  className={sessionFilter === filter.id ? 'is-active' : ''}
                  onClick={() => setSessionFilter(filter.id)}
                >
                  {filter.label}
                  <span>{sessionFilterCounts[filter.id]}</span>
                </button>
              ))}
            </div>
          </div>
          {loadingList && (
            <PantheonListSkeleton />
          )}
          {!loadingList && sessions.length === 0 && (
            <div className="pantheon-rail-empty">
              <span aria-hidden><CrownOutlined /></span>
              <strong>还没有讨论记录</strong>
              <p>第一场圆桌讨论会保存在本机，随时可以回来复核。</p>
              <Button type="link" onClick={openNewCouncil}>
                发起讨论
              </Button>
            </div>
          )}
          <div className="pantheon-session-list">
            {filteredSessions.map((item) => (
              <SessionRailItem
                key={item.id}
                item={item}
                active={active?.id === item.id}
                onClick={() => void openSession(item.id)}
              />
            ))}
          </div>
          {!loadingList && sessions.length > 0 && filteredSessions.length === 0 && (
            <div className="pantheon-archive-empty" role="status">
              <SearchOutlined />
              <strong>没有匹配的讨论</strong>
              <button
                type="button"
                onClick={() => {
                  setSessionQuery('')
                  setSessionFilter('all')
                }}
              >
                清除筛选
              </button>
            </div>
          )}
        </div>

        <footer className="pantheon-rail-foot">
          <div className="pantheon-rail-foot-head">
            <span className="pantheon-rail-foot-label">本机 Agent</span>
            <button type="button" onClick={() => void loadAgents()}>重新检测</button>
          </div>
          <div className="pantheon-agent-mini">
            {agents.map((agent) => {
              const fixed = agent.id === 'grok'
              return (
                <button
                  key={agent.id}
                  type="button"
                  title={
                    fixed
                      ? `${agent.name} 由 Gaea 办公 AgentRunner 驱动，无需配置路径${agent.reason ? `（${agent.reason}）` : ''}`
                      : agent.reason || `${agent.name} ${agent.version || ''}`
                  }
                  className={`${agent.available ? 'is-ready' : 'is-missing'}${fixed ? ' is-fixed' : ''}`}
                  onClick={() => { if (!fixed) openAgentPath(agent) }}
                >
                  <span>{agent.name.slice(0, 1).toUpperCase()}</span>
                  <small>{agent.available ? '就绪' : fixed ? '内置' : '设置'}</small>
                </button>
              )
            })}
          </div>
        </footer>
      </aside>

      <main
        key={active
          ? `record-${active.id}`
          : showComposer
            ? `composer-${composerSeed?.key || 'new'}`
            : 'welcome'}
        ref={canvasRef}
        className="pantheon-canvas"
      >
        {active ? (
          <PantheonRecord
            session={active}
            loading={loadingSession}
            busy={busy}
            error={error}
            liveStreams={liveStreams}
            onCancel={cancelCouncil}
            onDelete={deleteCouncil}
            onRefresh={() => void refreshSession(active.id)}
            onExport={() => void exportCouncil()}
            onExportJson={exportCouncilJson}
            onCopy={() => void copyCouncil()}
            onSignOff={signOffCouncil}
            onContinue={() => continueCouncil(active)}
          />
        ) : showComposer ? (
          <div className="pantheon-flow-stage pantheon-compose-stage">
            <PantheonComposer
              agents={agents}
              busy={busy}
              error={error}
              seed={composerSeed}
              onStart={beginCouncil}
              onRefreshAgents={() => void loadAgents()}
            />
            <CouncilProtocol />
          </div>
        ) : (
          <div className="pantheon-flow-stage pantheon-welcome-stage">
            <WelcomeCanvas
              agents={agents}
              onStart={openNewCouncil}
            />
            <CouncilProtocol />
          </div>
        )}

        {loadingSession && !active && (
          <div className="pantheon-loading-cover"><PantheonRecordSkeleton /></div>
        )}
        {error && !active && !showComposer && (
          <Alert
            className="pantheon-floating-error"
            type="error"
            showIcon
            message={error}
            closable
            onClose={() => setError('')}
          />
        )}
      </main>

      <Modal
        open={Boolean(pathAgent)}
        title={pathAgent ? `设置 ${pathAgent.name}` : '设置 Agent'}
        okText="保存路径"
        cancelText="取消"
        confirmLoading={busy}
        onOk={() => void saveAgentPath()}
        onCancel={() => setPathAgent(null)}
        destroyOnHidden
      >
        <p className="pantheon-path-help">
          填写可执行文件或 CLI 脚本的绝对路径。检测完成后，该参与者可以加入下一次讨论。
        </p>
        <Input
          value={pathValue}
          onChange={(event) => setPathValue(event.target.value)}
          placeholder="例如 C:\Tools\agent-cli.exe"
          onPressEnter={() => void saveAgentPath()}
          aria-label="Agent 可执行文件路径"
        />
        {pathAgent?.reason && <p className="pantheon-path-reason">{pathAgent.reason}</p>}
      </Modal>
    </div>
  )
}

export default PantheonPage
