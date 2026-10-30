import React, { useEffect, useMemo, useRef, useState } from 'react'
import {
  CheckCircleFilled,
  ClockCircleOutlined,
  CloseCircleFilled,
  CommentOutlined,
  CopyOutlined,
  DeleteOutlined,
  DownOutlined,
  DownloadOutlined,
  ExclamationCircleOutlined,
  FolderOpenOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  ReloadOutlined,
  StopOutlined,
  ToolOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { Alert, Button, Dropdown, Empty, Spin, Tag, Tooltip } from 'antd'
import { AGENT_ACCENTS } from './visuals'
import type {
  PantheonContribution,
  PantheonEvent,
  PantheonLiveStream,
  PantheonSession,
} from './types'

interface Props {
  session: PantheonSession
  loading: boolean
  busy: boolean
  error: string
  liveStreams: PantheonLiveStream[]
  onCancel: () => void
  onDelete: () => void
  onRefresh: () => void
  onExport: () => void
  onExportJson: () => void
  onCopy: () => void
  onSignOff: (decision: 'accepted' | 'needs_revision' | 'rejected') => void
  onContinue: () => void
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

const DISCUSSION_SLOTS = [
  { id: 'grok', name: 'Grok' },
  { id: 'codex', name: 'Codex' },
  { id: 'zcode', name: 'ZCode' },
  { id: 'dsh', name: 'DeepSeek Harness' },
]

const INSPECTOR_COLLAPSED_KEY = 'gaea.pantheon.inspector-collapsed'
const SEAT_TRACE_LIMIT = 12

function readInspectorCollapsed(): boolean {
  try {
    return window.localStorage.getItem(INSPECTOR_COLLAPSED_KEY) === '1'
  } catch {
    return false
  }
}

function persistInspectorCollapsed(collapsed: boolean): void {
  try {
    window.localStorage.setItem(INSPECTOR_COLLAPSED_KEY, collapsed ? '1' : '0')
  } catch {
    // A blocked localStorage should never prevent the inspector from collapsing.
  }
}

interface DiscussionOutput {
  id: string
  agentId: string
  agentName: string
  phase: string
  round: number
  text: string
  durationMs?: number
  streaming?: boolean
}

interface ToolTraceView {
  id: string
  agentId: string
  agentName: string
  toolName: string
  status: string
  input: string
  output: string
  error: string
  at: number
}

function participantStatusLabel(status: string): string {
  switch (status) {
    case 'running': return '处理中'
    case 'finished': return '已返回'
    case 'failed': return '失败'
    case 'unavailable': return '不可用'
    case 'waiting': return '候场'
    default: return status
  }
}

function statusMeta(status: string): { label: string; color: string; icon: React.ReactNode } {
  switch (status) {
    case 'running':
      return { label: '进行中', color: 'processing', icon: <Spin size="small" /> }
    case 'completed':
      return { label: '已小结', color: 'success', icon: <CheckCircleFilled /> }
    case 'cancelled':
      return { label: '已终止', color: 'default', icon: <StopOutlined /> }
    case 'failed':
      return { label: '失败', color: 'error', icon: <CloseCircleFilled /> }
    default:
      return { label: status, color: 'default', icon: <ClockCircleOutlined /> }
  }
}

function formatTime(ms?: number): string {
  if (!ms) return '未记录'
  return new Date(ms).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function workspaceName(path?: string): string {
  if (!path) return ''
  const parts = path.split(/[/\\]/).filter(Boolean)
  return parts[parts.length - 1] || path
}

function participantOrderLabel(agentIds: string[], agentId: string): string {
  const order = agentIds.indexOf(agentId)
  return order >= 0 ? `第 ${order + 1} 位 · ` : ''
}

function eventTitle(event: PantheonEvent): string {
  if (event.kind === 'agent_output') return `${event.agentId || 'Agent'} · 返回结果`
  if (event.kind === 'agent_error') return `${event.agentId || 'Agent'} · 调用失败`
  if (event.kind === 'agent_started') return event.text || 'Agent 开始工作'
  if (event.kind === 'tool_started') return `${event.toolName || '工具'} · 执行中`
  if (event.kind === 'tool_result') return `${event.toolName || '工具'} · ${event.toolStatus === 'failed' ? '失败' : '完成'}`
  if (event.kind === 'verdict') return '协作小结'
  if (event.kind === 'consensus') return '全员一致'
  if (event.kind === 'completed') return '讨论完成'
  if (event.kind === 'warning') return '警示'
  return event.text || event.kind
}

const FieldList: React.FC<{ title: string; items?: string[]; tone?: 'risk' | 'disagree' }> = ({
  title,
  items,
  tone,
}) => {
  if (!items?.length) return null
  return (
    <section className={`pantheon-verdict-block${tone ? ` is-${tone}` : ''}`}>
      <h4>{title}</h4>
      <ul>
        {items.map((item, index) => <li key={`${title}-${index}`}>{item}</li>)}
      </ul>
    </section>
  )
}

const ConclusionPanel: React.FC<{
  session: PantheonSession
  busy: boolean
  onSignOff: (decision: 'accepted' | 'needs_revision' | 'rejected') => void
  onContinue: () => void
}> = ({ session, busy, onSignOff, onContinue }) => {
  if (!session.verdict) {
    return (
      <Alert
        type="warning"
        showIcon
        message="本次讨论未形成自动小结"
        description="讨论没有走到协作汇总这一步。可基于已有发言续议，或查看时间线定位中断位置；失败原因见上方提示。"
      />
    )
  }

  return (
    <section className={`pantheon-verdict${session.verdict.partial ? ' is-partial' : ''}`}>
      <span className="pantheon-verdict-seal" aria-hidden>
        {session.signoff ? '阅' : '记'}
      </span>
      <div className="pantheon-verdict-head">
        <div>
          <span className="pantheon-eyebrow">
            协作小结{session.signoff ? ' · 已记录' : ' · 待你确认'}
          </span>
          <h3>{session.verdict.recommendation || '尚未形成明确的综合建议'}</h3>
        </div>
        <Tag color={session.verdict.partial ? 'warning' : 'success'}>
          {session.verdict.partial ? '信息不完整' : '讨论已整理'}
        </Tag>
      </div>
      {session.verdict.summary && <p className="pantheon-verdict-summary">{session.verdict.summary}</p>}
      <FieldList title="适用条件" items={session.verdict.conditions} />
      <FieldList title="保留分歧" items={session.verdict.disagreements} tone="disagree" />
      <FieldList title="开放问题与风险" items={session.verdict.risks} tone="risk" />
      <FieldList title="证据与补充" items={session.verdict.evidenceNotes} />
      <footer>
        <span>主持与汇总：{session.verdict.chairman || '未取得'}</span>
        <span>{formatTime(session.verdict.createdAt)}</span>
      </footer>
      {session.signoff ? (
        <div className={`pantheon-signoff is-${session.signoff.decision}`}>
          <CheckCircleFilled />
          <span>
            {session.signoff.decision === 'accepted' ? '已确认小结'
              : session.signoff.decision === 'needs_revision' ? '已标记需继续讨论' : '暂不采纳'}
          </span>
          <time>{formatTime(session.signoff.at)}</time>
        </div>
      ) : (
        <div className="pantheon-signoff-actions">
          <span>你的确认只代表是否采用这份小结：</span>
          <Button type="primary" onClick={() => onSignOff('accepted')} disabled={busy}>
            确认小结
          </Button>
          <Button onClick={() => onSignOff('needs_revision')} disabled={busy}>
            需继续讨论
          </Button>
          <Button type="text" onClick={() => onSignOff('rejected')} disabled={busy}>
            暂不采纳
          </Button>
        </div>
      )}
      <div className="pantheon-verdict-next">
        <span>保留的分歧可以作为下一场讨论的起点。</span>
        <Button icon={<CommentOutlined />} onClick={onContinue} disabled={busy}>
          基于小结续议
        </Button>
      </div>
    </section>
  )
}

const AgentOutputWindow: React.FC<{
  slot: { id: string; name: string }
  participant?: PantheonSession['participants'][number]
  outputs: DiscussionOutput[]
  traces: ToolTraceView[]
  facilitator: boolean
  phase: string
  collapsed: boolean
  onToggleCollapsed: () => void
}> = ({ slot, participant, outputs, traces, facilitator, phase, collapsed, onToggleCollapsed }) => {
  const feedRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const feed = feedRef.current
    if (feed) feed.scrollTop = feed.scrollHeight
  }, [outputs])
  const status = !participant
    ? '未加入'
    : participant.status === 'running'
      ? '流式输出中'
      : participantStatusLabel(participant.status)
  return (
    <article
      data-testid={`pantheon-live-seat-${slot.id}`}
      className={`pantheon-agent-window${participant ? ` is-${participant.status}` : ' is-absent'}${collapsed ? ' is-collapsed' : ''}`}
      style={{ '--agent-accent': AGENT_ACCENTS[slot.id] || 'var(--gaea-glow)' } as React.CSSProperties}
    >
      <header>
        <span className="pantheon-live-seat-identity">
          <span className="pantheon-seat-monogram" aria-hidden>
            {slot.name.slice(0, 1).toUpperCase()}
          </span>
          <span className="pantheon-seat-pane-title">{participant?.name || slot.name}</span>
          {facilitator && participant && (
            <span className="pantheon-facilitator-mark" title="负责协作汇总">
              <CommentOutlined />
            </span>
          )}
        </span>
        <small>{status}</small>
        <Button
          type="text"
          size="small"
          className={`pantheon-seat-collapse${collapsed ? ' is-collapsed' : ''}`}
          icon={<DownOutlined />}
          aria-expanded={!collapsed}
          aria-label={collapsed ? `展开${participant?.name || slot.name}独立窗口` : `折叠${participant?.name || slot.name}独立窗口`}
          data-testid={`pantheon-seat-collapse-${slot.id}`}
          onClick={onToggleCollapsed}
        />
      </header>
      {!collapsed && (
        <>
          <div ref={feedRef} className="pantheon-agent-feed" aria-live={participant?.status === 'running' ? 'polite' : undefined}>
            {outputs.map((output) => (
              <article
                key={output.id}
                className={`pantheon-agent-message${output.streaming ? ' is-streaming' : ''}`}
              >
                <header>
                  <span>
                    {output.round ? `第 ${output.round} 轮 · ` : ''}
                    {PHASE_LABEL[output.phase] || output.phase}
                  </span>
                  {output.durationMs ? <small>{(output.durationMs / 1000).toFixed(1)}s</small> : <small>独立上下文</small>}
                </header>
                <p>
                  {output.text}
                  {output.streaming && <span className="pantheon-stream-caret" aria-label="正在输出" />}
                </p>
              </article>
            ))}
            {participant?.status === 'running' && (
              <div className="pantheon-agent-generating">
                <Spin size="small" /> 正在{PHASE_LABEL[phase] || '处理'}…
              </div>
            )}
            {participant && outputs.length === 0 && participant.status !== 'running' && (
              <span className="pantheon-muted">等待发言</span>
            )}
            {!participant && (
              <span className="pantheon-muted">未参与本次讨论，独立窗口保留为空。</span>
            )}
            {participant?.status === 'failed' && (
              <span className="pantheon-seat-failed">{participant.error || '调用失败'}</span>
            )}
          </div>
          <SeatTrajectory traces={traces} />
        </>
      )}
    </article>
  )
}

const SeatTrajectory: React.FC<{
  traces: ToolTraceView[]
}> = ({ traces }) => {
  if (traces.length === 0) return null
  const visibleTraces = traces.slice(-SEAT_TRACE_LIMIT)
  const foldedCount = traces.length - visibleTraces.length
  return (
    <section className="pantheon-seat-trace" aria-label="工具轨迹">
      <header className="pantheon-seat-trace-head">
        <ToolOutlined aria-hidden />
        <strong>工具轨迹</strong>
        <em>{traces.length} 次</em>
      </header>
      <ol className="pantheon-seat-trace-list">
        {foldedCount > 0 && (
          <li className="pantheon-seat-trace-fold">更早 {foldedCount} 条已折叠</li>
        )}
        {visibleTraces.map((trace) => (
          <li
            key={trace.id}
            className={`pantheon-tool-trace is-${trace.status}`}
            data-testid={`pantheon-tool-${trace.id}`}
          >
            <span className="pantheon-tool-trace-state" aria-hidden>
              {trace.status === 'started' ? <ClockCircleOutlined /> : null}
              {trace.status === 'completed' ? <CheckCircleFilled /> : null}
              {trace.status === 'failed' ? <CloseCircleFilled /> : null}
            </span>
            <div className="pantheon-tool-trace-main">
              <div className="pantheon-tool-trace-title">
                <strong>{trace.toolName || '工具'}</strong>
                <small>
                  {trace.status === 'started' ? '执行中' : trace.status === 'failed' ? '失败' : '已完成'}
                </small>
                <time>{formatTime(trace.at)}</time>
              </div>
              {trace.input && <code>{trace.input}</code>}
              {trace.error && <p className="pantheon-tool-trace-error">{trace.error}</p>}
              {trace.output && (
                <details>
                  <summary>结果预览</summary>
                  <pre>{trace.output}</pre>
                </details>
              )}
            </div>
          </li>
        ))}
      </ol>
    </section>
  )
}

export const PantheonRecord: React.FC<Props> = ({
  session,
  loading,
  busy,
  error,
  liveStreams,
  onCancel,
  onDelete,
  onRefresh,
  onExport,
  onExportJson,
  onCopy,
  onSignOff,
  onContinue,
}) => {
  const meta = statusMeta(session.status)
  const running = session.status === 'running'
  const sharedFeedRef = useRef<HTMLDivElement>(null)
  const [collapsedSeats, setCollapsedSeats] = useState<Record<string, boolean>>({})
  const [inspectorCollapsed, setInspectorCollapsed] = useState(readInspectorCollapsed)
  const recentEvents = useMemo(() => [...(session.events ?? [])].reverse().slice(0, 14), [session.events])
  const toggleSeatCollapsed = (seatId: string) => {
    setCollapsedSeats((prev) => ({ ...prev, [seatId]: !prev[seatId] }))
  }
  const toggleInspectorCollapsed = () => {
    setInspectorCollapsed((prev) => {
      const next = !prev
      persistInspectorCollapsed(next)
      return next
    })
  }
  const discussionOutputs = useMemo<DiscussionOutput[]>(() => {
    if ((session.contributions ?? []).length > 0) {
      return session.contributions.map((item: PantheonContribution) => ({
        id: item.id,
        agentId: item.agentId,
        agentName: item.agentName,
        phase: item.phase,
        round: item.round || 0,
        text: item.content,
        durationMs: item.durationMs,
      }))
    }
    return (session.events ?? [])
      .filter((event) => event.kind === 'agent_output' && event.agentId)
      .map((event) => ({
        id: `event-${event.seq}`,
        agentId: event.agentId || '',
        agentName:
          (session.participants ?? []).find((participant) => participant.agentId === event.agentId)?.name
          || event.agentId
          || '未知参与者',
        phase: event.phase,
        round: event.round || 0,
        text: event.text || '',
        durationMs: event.durationMs,
      }))
  }, [session.contributions, session.events, session.participants])
  const displayOutputs = useMemo(() => {
    const outputs = [...discussionOutputs]
    const keys = new Set(outputs.map(
      (item) => `${item.agentId}:${item.phase}:${item.round}`,
    ))
    for (const stream of liveStreams) {
      const key = `${stream.agentId}:${stream.phase}:${stream.round}`
      if (keys.has(key)) continue
      outputs.push({
        id: stream.id,
        agentId: stream.agentId,
        agentName:
          session.participants.find((participant) => participant.agentId === stream.agentId)?.name
          || stream.agentId,
        phase: stream.phase,
        round: stream.round,
        text: stream.text,
        streaming: stream.streaming,
      })
    }
    return outputs
  }, [discussionOutputs, liveStreams, session.participants])
  const toolTraces = useMemo<ToolTraceView[]>(() => {
    const byID = new Map<string, ToolTraceView>()
    for (const event of session.events) {
      if (event.kind !== 'tool_started' && event.kind !== 'tool_result') continue
      const id = event.toolId || `tool-event-${event.seq}`
      const existing = byID.get(id)
      byID.set(id, {
        id,
        agentId: event.agentId || existing?.agentId || '',
        agentName:
          session.participants.find((participant) => participant.agentId === event.agentId)?.name
          || existing?.agentName
          || event.agentId
          || 'Agent',
        toolName: event.toolName || existing?.toolName || '工具',
        status: event.toolStatus || existing?.status || 'started',
        input: event.toolInput || existing?.input || '',
        output: event.toolOutput || existing?.output || '',
        error: event.error || existing?.error || '',
        at: event.at,
      })
    }
    return [...byID.values()].slice(-40)
  }, [session.events, session.participants])
  const tracesBySeat = useMemo(() => {
    const bySeat = new Map<string, ToolTraceView[]>()
    for (const trace of toolTraces) {
      if (!trace.agentId) continue
      const list = bySeat.get(trace.agentId)
      if (list) list.push(trace)
      else bySeat.set(trace.agentId, [trace])
    }
    return bySeat
  }, [toolTraces])
  useEffect(() => {
    if (!running) return
    const feed = sharedFeedRef.current
    if (feed) feed.scrollTop = feed.scrollHeight
  }, [displayOutputs, running])
  const untilConsensus = Boolean(session.request.untilConsensus)
  const totalRounds = Math.max(1, Math.min(20, session.request.rounds || 2))
  const discussionSteps = useMemo(() => untilConsensus
    ? [
        { id: 'opening', label: '开场陈述', detail: '按序表达' },
        { id: 'consensus', label: '共识讨论', detail: '全员一致即结束' },
        { id: 'synthesis', label: '协作汇总', detail: '共识与分歧' },
      ]
    : [
        { id: 'opening', label: '开场陈述', detail: '按序表达' },
        ...Array.from({ length: Math.max(0, totalRounds - 1) }, (_, index) => ({
          id: `discussion-${index + 2}`,
          label: `第 ${index + 2} 轮`,
          detail: '补充论证',
        })),
        { id: 'synthesis', label: '协作汇总', detail: '共识与分歧' },
      ], [totalRounds, untilConsensus])
  const activeRound = Math.max(1, Math.min(totalRounds, session.currentRound || 1))
  const currentStepIndex = session.status === 'completed'
    ? discussionSteps.length
    : session.phase === 'preparing'
      ? -1
      : session.phase === 'synthesis' || session.phase === 'verdict'
        ? discussionSteps.length - 1
        : untilConsensus && session.phase === 'opening'
          ? 0
          : untilConsensus && ['discussion', 'challenge', 'review'].includes(session.phase)
            ? 1
        : session.phase === 'discussion' || session.phase === 'challenge' || session.phase === 'review'
          ? Math.max(0, activeRound - 1)
          : 0
  const speakingOrder = useMemo(() => {
    const ids = session.request.agentIds || []
    const offset = (activeRound - 1) % Math.max(ids.length, 1)
    return [...ids.slice(offset), ...ids.slice(0, offset)]
  }, [session.request.agentIds, activeRound])

  return (
    <div
      className={`pantheon-stage-grid pantheon-record${inspectorCollapsed ? ' is-inspector-collapsed' : ''}`}
      data-testid="pantheon-record"
    >
      <article className="pantheon-record-main">
        <header className="pantheon-record-head">
          <div className="pantheon-record-title">
            <div className="pantheon-record-kicker">
              <span className="pantheon-eyebrow">讨论纪要</span>
              <span>{PHASE_LABEL[session.phase] || session.phase}</span>
              {session.request.workspace && (
                <span className="pantheon-record-workspace" title={session.request.workspace}>
                  <FolderOpenOutlined />
                  {workspaceName(session.request.workspace)}
                </span>
              )}
            </div>
            <h2>{session.title}</h2>
            {session.request.topic && session.request.topic !== session.title && (
              <p>{session.request.topic}</p>
            )}
          </div>
          <div className="pantheon-record-actions">
            <Tag icon={meta.icon} color={meta.color}>{meta.label}</Tag>
            <Tooltip title={inspectorCollapsed ? '展开纪要摘要' : '收起纪要摘要'}>
              <Button
                type="text"
                icon={inspectorCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                aria-expanded={!inspectorCollapsed}
                aria-controls="pantheon-inspector"
                data-testid="pantheon-inspector-toggle"
                onClick={toggleInspectorCollapsed}
              />
            </Tooltip>
            <Tooltip title="刷新卷宗">
              <Button type="text" icon={<ReloadOutlined />} onClick={onRefresh} loading={loading} />
            </Tooltip>
            {running ? (
              <Button danger icon={<StopOutlined />} onClick={onCancel} disabled={busy}>
                终止
              </Button>
            ) : (
              <>
                <Tooltip title="复制 Markdown">
                  <Button type="text" icon={<CopyOutlined />} onClick={onCopy} />
                </Tooltip>
                <Dropdown
                  trigger={['click']}
                  menu={{
                    items: [
                      { key: 'markdown', label: 'Markdown 议事录' },
                      { key: 'json', label: 'JSON 完整记录' },
                    ],
                    onClick: ({ key }) => {
                      if (key === 'json') onExportJson()
                      else onExport()
                    },
                  }}
                >
                  <Button icon={<DownloadOutlined />}>
                    导出 <DownOutlined />
                  </Button>
                </Dropdown>
                <Tooltip title="删除议事录">
                  <Button type="text" danger icon={<DeleteOutlined />} onClick={onDelete} />
                </Tooltip>
              </>
            )}
          </div>
        </header>

        {error && <Alert type="error" showIcon message={error} />}
        {session.error && (
          <Alert
            type={session.status === 'cancelled' ? 'warning' : 'error'}
            showIcon
            message={session.error}
          />
        )}

        <ol
          className={`pantheon-phase-track${discussionSteps.length > 6 ? ' is-long' : ''}`}
          aria-label={untilConsensus ? '共识讨论进度' : '多轮讨论进度'}
          style={{ '--round-count': discussionSteps.length } as React.CSSProperties}
        >
          {discussionSteps.map((step, index) => {
            const done = session.status === 'completed' || index < currentStepIndex
            const active = session.status !== 'completed' && index === currentStepIndex
            const failed = active && (session.status === 'failed' || session.status === 'cancelled')
            return (
              <li
                key={step.id}
                className={`${done ? 'is-done' : ''}${active ? ' is-active' : ''}${failed ? ' is-failed' : ''}`}
              >
                <span className="pantheon-phase-index">{String(index + 1).padStart(2, '0')}</span>
                <span>{step.label}</span>
                <small>{step.detail}</small>
              </li>
            )
          })}
        </ol>

        {!running && (
          <div className="pantheon-record-body" aria-label="讨论小结">
            <ConclusionPanel
              session={session}
              busy={busy}
              onSignOff={onSignOff}
              onContinue={onContinue}
            />
          </div>
        )}

        <section
          className={`pantheon-discussion-board${running ? ' is-live' : ''}`}
          aria-live={running ? 'polite' : undefined}
        >
          <div className="pantheon-output-head">
            <div>
              <span className="pantheon-eyebrow">{running ? '讨论进行中' : '共享与独立输出'}</span>
              <h3>
                {running && <span className="pantheon-live-dot" />}
                共享观点池与独立上下文
              </h3>
            </div>
            <div className="pantheon-live-meta">
              <strong>
                {untilConsensus
                  ? `第 ${activeRound} 轮 · 至共识`
                  : `第 ${activeRound} / ${totalRounds} 轮`}
              </strong>
              <span>{PHASE_LABEL[session.phase] || session.phase}</span>
            </div>
          </div>

          {running && (
            <div className="pantheon-speaking-queue" aria-label="本轮发言顺序">
              <span>本轮顺序</span>
              <ol>
                {speakingOrder.map((agentID, index) => {
                  const participant = session.participants.find((item) => item.agentId === agentID)
                  return (
                    <li
                      key={agentID}
                      className={participant?.status === 'running' ? 'is-speaking' : ''}
                    >
                      <small>{String(index + 1).padStart(2, '0')}</small>
                      <span>{participant?.name || agentID}</span>
                    </li>
                  )
                })}
              </ol>
            </div>
          )}

          <article className="pantheon-shared-window">
            <header>
              <span className="pantheon-window-mark" aria-hidden>共</span>
              <span className="pantheon-window-title">
                <strong>共享观点池</strong>
                <small>同步公开输出，不覆盖各 Agent 的独立上下文</small>
              </span>
              <span>{displayOutputs.length} 条发言</span>
            </header>
            <div ref={sharedFeedRef} className="pantheon-shared-feed" aria-live={running ? 'polite' : undefined}>
              {displayOutputs.length === 0 && (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="等待第一条公开观点" />
              )}
              {displayOutputs.map((output) => (
                <article
                  key={output.id}
                  data-testid={`pantheon-shared-message-${output.id}`}
                  className={`pantheon-shared-message${output.streaming ? ' is-streaming' : ''}`}
                >
                  <span className="pantheon-shared-avatar" aria-hidden>
                    {output.agentName.slice(0, 1).toUpperCase()}
                  </span>
                  <div className="pantheon-shared-message-body">
                    <header>
                      <strong>{output.agentName}</strong>
                      <small>
                        {output.round ? `第 ${output.round} 轮 · ` : ''}
                        {PHASE_LABEL[output.phase] || output.phase}
                      </small>
                    </header>
                    <p>
                      {output.text}
                      {output.streaming && <span className="pantheon-stream-caret" aria-label="正在输出" />}
                    </p>
                  </div>
                </article>
              ))}
            </div>
          </article>

          <div className="pantheon-agent-windows">
            {DISCUSSION_SLOTS.map((slot) => {
              const participant = session.participants.find((item) => item.agentId === slot.id)
              const outputs = displayOutputs.filter((item) => item.agentId === slot.id)
              return (
                <AgentOutputWindow
                  key={slot.id}
                  slot={slot}
                  participant={participant}
                  outputs={outputs}
                  traces={tracesBySeat.get(slot.id) ?? []}
                  facilitator={session.request.chairmanAgentId === slot.id}
                  phase={session.phase}
                  collapsed={Boolean(collapsedSeats[slot.id])}
                  onToggleCollapsed={() => toggleSeatCollapsed(slot.id)}
                />
              )
            })}
          </div>
        </section>

      </article>

      {!inspectorCollapsed && (
        <aside className="pantheon-inspector" id="pantheon-inspector" aria-label="讨论纪要摘要">
          <div className="pantheon-inspector-head">
            <span className="pantheon-eyebrow">讨论设置</span>
            <Tag icon={meta.icon} color={meta.color}>{meta.label}</Tag>
          </div>
          <section className="pantheon-inspector-block">
            <div className="pantheon-inspector-title">
              <FolderOpenOutlined />
              <h3>讨论范围</h3>
            </div>
            <dl className="pantheon-facts">
              <div>
                <dt>讨论轮数</dt>
                <dd>{untilConsensus ? `不限（≤${totalRounds} 轮）` : totalRounds}</dd>
              </div>
              <div><dt>运行边界</dt><dd>只读</dd></div>
              <div><dt>当前进度</dt><dd>{PHASE_LABEL[session.phase] || session.phase}</dd></div>
              <div>
                <dt>主持与汇总</dt>
                <dd>
                  {session.participants.find(
                    (participant) => participant.agentId === session.request.chairmanAgentId,
                  )?.name || '未设置'}
                </dd>
              </div>
              <div><dt>创建时间</dt><dd>{formatTime(session.createdAt)}</dd></div>
              {session.request.workspace && (
                <div className="pantheon-fact-wide">
                  <dt>工作空间</dt>
                  <dd title={session.request.workspace}>{workspaceName(session.request.workspace)}</dd>
                </div>
              )}
            </dl>
          </section>

          <section className="pantheon-inspector-block">
            <div className="pantheon-inspector-title">
              <CommentOutlined />
              <h3>参与者</h3>
            </div>
            <ul className="pantheon-seat-list">
              {DISCUSSION_SLOTS.map((slot) => {
                const participant = session.participants.find((item) => item.agentId === slot.id)
                return (
                  <li key={slot.id} className={participant ? '' : 'is-absent'}>
                    <span className={`pantheon-seat-dot${participant ? ` is-${participant.status}` : ''}`} />
                    <span>{participant?.name || slot.name}</span>
                    <small>
                      {participant
                        ? `${participantOrderLabel(session.request.agentIds, participant.agentId)}${participantStatusLabel(participant.status)}`
                        : '未加入'}
                    </small>
                  </li>
                )
              })}
            </ul>
          </section>

          <section className="pantheon-inspector-block">
            <div className="pantheon-inspector-title">
              <CommentOutlined />
              <h3>发言顺序</h3>
            </div>
            <ol className="pantheon-inspector-order">
              {(session.request.agentIds ?? []).map((agentID, index) => (
                <li key={agentID}>
                  <span>{String(index + 1).padStart(2, '0')}</span>
                  <small>
                    {session.participants.find((participant) => participant.agentId === agentID)?.name || agentID}
                  </small>
                </li>
              ))}
            </ol>
            <p className="pantheon-inspector-note">下一轮自动轮换首位，避免固定顺序造成偏置。</p>
          </section>

          <section className="pantheon-inspector-block">
            <div className="pantheon-inspector-title">
              <ExclamationCircleOutlined />
              <h3>证据与警示</h3>
            </div>
            {session.request.evidence?.length ? (
              <details className="pantheon-evidence-details">
                <summary>{session.request.evidence.length} 份材料</summary>
                {session.request.evidence.map((item) => (
                  <div key={item.name}>
                    <strong>{item.name}</strong>
                    <pre className="pantheon-prose">{item.content}</pre>
                  </div>
                ))}
              </details>
            ) : (
              <p className="pantheon-muted">本次未附加材料。</p>
            )}
            {(() => {
              const visibleWarnings = (session.warnings ?? []).filter((w) => w !== session.error)
              return visibleWarnings.length ? (
                <ul className="pantheon-warning-list">
                  {visibleWarnings.map((warning, index) => (
                    <li key={`${warning}-${index}`}><WarningOutlined /> {warning}</li>
                  ))}
                </ul>
              ) : (
                <p className="pantheon-muted">暂无额外警示。</p>
              )
            })()}
          </section>

          <section className="pantheon-inspector-block">
            <div className="pantheon-inspector-title">
              <ClockCircleOutlined />
              <h3>运行时间线</h3>
            </div>
            <details className="pantheon-timeline">
              <summary>最近 {recentEvents.length} 条事件 · 点击展开</summary>
              <div className="pantheon-event-stream">
                {recentEvents.length === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="等待讨论事件" />}
                {recentEvents.map((event) => (
                  <article key={`${event.sessionId}-${event.seq}`} className={`pantheon-event is-${event.kind}`}>
                    <div className="pantheon-event-line">
                      <strong>{eventTitle(event)}</strong>
                      <time>{formatTime(event.at)}</time>
                    </div>
                    {event.error && <p className="pantheon-event-error">{event.error}</p>}
                    {event.text && event.kind !== 'agent_output' && event.kind !== 'agent_started' && <p>{event.text}</p>}
                    {event.kind === 'agent_output' && (
                      <small className="pantheon-event-dim">
                        {event.durationMs ? `${(event.durationMs / 1000).toFixed(1)}s · ` : ''}全文见参与者独立窗口
                      </small>
                    )}
                  </article>
                ))}
              </div>
            </details>
          </section>
        </aside>
      )}
    </div>
  )
}
