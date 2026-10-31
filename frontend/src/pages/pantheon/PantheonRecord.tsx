import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  CheckCircleFilled,
  ClockCircleOutlined,
  CloseCircleFilled,
  CloseOutlined,
  CommentOutlined,
  CopyOutlined,
  DeleteOutlined,
  DownOutlined,
  DownloadOutlined,
  FolderOpenOutlined,
  FileTextOutlined,
  InfoCircleOutlined,
  PictureOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  StopOutlined,
  StepForwardOutlined,
  TeamOutlined,
  ToolOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { Alert, Button, Dropdown, Empty, Spin, Tag, Tooltip } from 'antd'
import { MarkdownContent, mdStyles } from '../../components/MarkdownContent'
import { AGENT_ACCENTS, agentCapabilityLabel, protocolLabel } from './visuals'
import { PantheonAgentAvatar } from './PantheonAgentAvatar'
import type {
  PantheonContribution,
  PantheonEvent,
  PantheonLiveStream,
  PantheonParticipant,
  PantheonSession,
} from './types'

interface Props {
  session: PantheonSession
  loading: boolean
  busy: boolean
  error: string
  liveStreams: PantheonLiveStream[]
  onCancel: () => void
  onSkipAgent: (agentId: string) => void
  onRetryAgent: (agentId: string) => void
  onRestart: () => void
  onResume: () => void
  onDelete: () => void
  onRefresh: () => void
  onExport: () => void
  onExportJson: () => void
  onCopy: () => void
  onOpenResult: () => void
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

const TOOL_TRACE_LIMIT = 60
const PUBLIC_EVENT_LIMIT = 10
const AGENT_SLOW_AFTER_MS = 60_000
const STALL_AFTER_MS = 120_000
// 共享流折叠：超过此长度的已完成发言默认收起只留预览；流式中的发言永不折叠。
const FOLDABLE_CHARS = 160
const FOLD_DEFAULT_CHARS = 420
const FOLD_PREVIEW_CHARS = 72

function charCount(text: string): number {
  return Array.from(text).length
}

function previewText(text: string): string {
  const flat = text.replace(/\s+/g, ' ').trim()
  const chars = Array.from(flat)
  if (chars.length <= FOLD_PREVIEW_CHARS) return flat
  return `${chars.slice(0, FOLD_PREVIEW_CHARS).join('')}…`
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
  at?: number
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

interface SharedEntry {
  id: string
  kind: 'topic' | 'orchestrator' | 'agent' | 'event' | 'live' | 'warning'
  label: string
  text: string
  meta?: string
  at: number
  agentId?: string
  streaming?: boolean
}

type AvatarStatus = 'online' | 'offline' | 'running' | 'waiting' | 'finished' | 'skipped' | 'failed'

function participantStatusLabel(status: string): string {
  switch (status) {
    case 'running': return '正在发言'
    case 'finished': return '已返回'
    case 'skipped': return '已跳过'
    case 'retrying': return '正在重试'
    case 'failed': return '失败'
    case 'unavailable': return '不可用'
    case 'waiting': return '候场'
    default: return status
  }
}

function avatarStatus(status?: string): AvatarStatus | undefined {
  switch (status) {
    case 'running':
    case 'waiting':
    case 'finished':
    case 'skipped':
    case 'retrying':
    case 'failed':
      return status === 'retrying' ? 'running' : status
    case 'unavailable':
      return 'offline'
    default:
      return undefined
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

function formatClock(ms?: number): string {
  if (!ms) return ''
  return new Date(ms).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatElapsed(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  const pair = (value: number) => String(value).padStart(2, '0')
  return hours > 0
    ? `${pair(hours)}:${pair(minutes)}:${pair(seconds)}`
    : `${pair(minutes)}:${pair(seconds)}`
}

function useSecondClock(enabled: boolean): number {
  const [clock, setClock] = useState(Date.now)
  useEffect(() => {
    setClock(Date.now())
    if (!enabled) return
    const timer = window.setInterval(() => setClock(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [enabled])
  return clock
}

const AgentElapsed: React.FC<{
  startedAt?: number
  now: number
  testId?: string
  className?: string
}> = ({ startedAt, now, testId, className }) => {
  if (!startedAt) return null
  const elapsed = Math.max(0, now - startedAt)
  return (
    <span
      className={[
        'pantheon-agent-elapsed',
        elapsed >= AGENT_SLOW_AFTER_MS ? 'is-slow' : '',
        className || '',
      ].filter(Boolean).join(' ')}
      data-testid={testId}
      title={`当前席位已持续工作 ${formatElapsed(elapsed)}`}
    >
      <ClockCircleOutlined aria-hidden />
      <span>工作 {formatElapsed(elapsed)}</span>
    </span>
  )
}

function workspaceName(path?: string): string {
  if (!path) return ''
  const parts = path.split(/[/\\]/).filter(Boolean)
  return parts[parts.length - 1] || path
}

function participantOrderLabel(agentIds: string[] | undefined, agentId: string): string {
  const order = (agentIds || []).indexOf(agentId)
  return order >= 0 ? `第 ${order + 1} 位 · ` : ''
}

function joinNames(names: string[]): string {
  if (names.length <= 3) return names.join('、')
  return `${names.slice(0, 3).join('、')}等 ${names.length} 位`
}

function eventTitle(event: PantheonEvent): string {
  if (event.kind === 'agent_error') return `${event.agentId || 'Agent'} · 调用失败`
  if (event.kind === 'agent_skipped') return '跳过当前 Agent'
  if (event.kind === 'agent_retry') return '重试当前 Agent'
  if (event.kind === 'consensus') return '全员一致'
  if (event.kind === 'completed') return '讨论完成'
  if (event.kind === 'verdict') return '协作小结'
  if (event.kind === 'warning') return '运行警示'
  if (event.kind === 'phase') return PHASE_LABEL[event.phase] || event.phase
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

// 共享流里 agent 发言的正文：长文默认折叠成一行预览，点按钮展开/收起；
// 流式中的发言始终全文展示。折叠只影响阅读，不影响导出与小结。
const SharedAgentBody: React.FC<{
  entry: SharedEntry
  foldedMap: Record<string, boolean>
  onFold: (id: string, folded: boolean) => void
}> = ({ entry, foldedMap, onFold }) => {
  const markdown = (
    <MarkdownContent source={entry.text} className="md-content pantheon-shared-markdown" />
  )
  if (entry.streaming) return markdown
  if (charCount(entry.text) <= FOLDABLE_CHARS) return markdown
  const folded = foldedMap[entry.id] ?? charCount(entry.text) > FOLD_DEFAULT_CHARS
  if (folded) {
    return (
      <>
        <p className="pantheon-shared-preview">{previewText(entry.text)}</p>
        <button
          type="button"
          className="pantheon-shared-fold"
          aria-expanded={false}
          data-testid={`pantheon-shared-fold-${entry.id}`}
          onClick={() => onFold(entry.id, false)}
        >
          展开全文
        </button>
      </>
    )
  }
  return (
    <>
      {markdown}
      <button
        type="button"
        className="pantheon-shared-fold is-open"
        aria-expanded
        data-testid={`pantheon-shared-fold-${entry.id}`}
        onClick={() => onFold(entry.id, true)}
      >
        收起
      </button>
    </>
  )
}

const ConclusionPanel: React.FC<{
  session: PantheonSession
  busy: boolean
  onOpenResult: () => void
  onSignOff: (decision: 'accepted' | 'needs_revision' | 'rejected') => void
  onContinue: () => void
}> = ({ session, busy, onOpenResult, onSignOff, onContinue }) => {
  if (!session.verdict) {
    return (
      <Alert
        type="warning"
        showIcon
        message="本次讨论未形成自动小结"
        description="讨论没有走到协作汇总这一步。可基于已有发言续议，或在节点详情中定位中断位置。"
      />
    )
  }

  const detailCount = [
    session.verdict.conditions,
    session.verdict.disagreements,
    session.verdict.risks,
    session.verdict.evidenceNotes,
  ].reduce((total, items) => total + (items?.length || 0), 0)

  return (
    <section className={`pantheon-verdict${session.verdict.partial ? ' is-partial' : ''}`}>
      <span className="pantheon-verdict-seal" aria-hidden>
        {session.signoff ? '阅' : '记'}
      </span>
      <div className="pantheon-verdict-head">
        <div>
          <span className="pantheon-eyebrow">
            共享结论{session.signoff ? ' · 已记录' : ' · 待你确认'}
          </span>
          <h3>{session.verdict.recommendation || '尚未形成明确的综合建议'}</h3>
        </div>
        <Tag color={session.verdict.partial ? 'warning' : 'success'}>
          {session.verdict.partial ? '信息不完整' : '讨论已整理'}
        </Tag>
      </div>
      {session.verdict.summary && <p className="pantheon-verdict-summary">{session.verdict.summary}</p>}

      {detailCount > 0 && (
        <details className="pantheon-verdict-details">
          <summary>查看适用条件、分歧与依据 · {detailCount} 项</summary>
          <FieldList title="适用条件" items={session.verdict.conditions} />
          <FieldList title="保留分歧" items={session.verdict.disagreements} tone="disagree" />
          <FieldList title="开放问题与风险" items={session.verdict.risks} tone="risk" />
          <FieldList title="证据与补充" items={session.verdict.evidenceNotes} />
        </details>
      )}

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
          <span>确认只代表是否采用这份小结：</span>
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
        <span>
          已由 {session.verdict.chairman || '主持人'} 整理为最终讨论文档；保留分歧可作为下一场起点。
        </span>
        <Button icon={<FileTextOutlined />} onClick={onOpenResult} disabled={busy}>
          打开最终文档
        </Button>
        <Button icon={<CommentOutlined />} onClick={onContinue} disabled={busy}>
          基于小结续议
        </Button>
      </div>
    </section>
  )
}

const ToolTraceList: React.FC<{ traces: ToolTraceView[] }> = ({ traces }) => {
  if (traces.length === 0) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="该 Agent 尚未调用工具" />
  }
  const visibleTraces = traces.slice(-TOOL_TRACE_LIMIT)
  const foldedCount = traces.length - visibleTraces.length
  return (
    <section className="pantheon-agent-trace-list" aria-label="Agent 工具轨迹">
      {foldedCount > 0 && (
        <p className="pantheon-agent-trace-fold">更早 {foldedCount} 条工具轨迹已折叠</p>
      )}
      {visibleTraces.map((trace) => (
        <details
          key={trace.id}
          className={`pantheon-agent-trace is-${trace.status}`}
          data-testid={`pantheon-tool-${trace.id}`}
          open={trace.status === 'started' ? true : undefined}
        >
          <summary>
            <span className="pantheon-tool-trace-state" aria-hidden>
              {trace.status === 'started' ? <ClockCircleOutlined /> : null}
              {trace.status === 'completed' ? <CheckCircleFilled /> : null}
              {trace.status === 'failed' ? <CloseCircleFilled /> : null}
            </span>
            <span>
              <strong>{trace.toolName || '工具'}</strong>
              <small>
                {trace.status === 'started' ? '执行中' : trace.status === 'failed' ? '失败' : '已完成'}
              </small>
            </span>
            <time>{formatClock(trace.at)}</time>
            <DownOutlined className="pantheon-agent-trace-chevron" />
          </summary>
          <div className="pantheon-agent-trace-detail">
            {trace.input && (
              <>
                <small>输入</small>
                <code>{trace.input}</code>
              </>
            )}
            {trace.error && <p className="pantheon-tool-trace-error">{trace.error}</p>}
            {trace.output && (
              <>
                <small>结果</small>
                <pre>{trace.output}</pre>
              </>
            )}
          </div>
        </details>
      ))}
    </section>
  )
}

const AgentDetailPanel: React.FC<{
  participant?: PantheonParticipant
  agentId: string
  agentName: string
  outputs: DiscussionOutput[]
  traces: ToolTraceView[]
  facilitator: boolean
  phase: string
  startedAt?: number
  now: number
  tab: 'output' | 'trace'
  onTab: (tab: 'output' | 'trace') => void
  onClose: () => void
  panelRef: React.RefObject<HTMLElement>
}> = ({
  participant,
  agentId,
  agentName,
  outputs,
  traces,
  facilitator,
  phase,
  startedAt,
  now,
  tab,
  onTab,
  onClose,
  panelRef,
}) => {
  const liveOutput = outputs.some((output) => output.streaming)
  const running = participant?.status === 'running' || liveOutput

  return (
    <aside
      ref={panelRef}
      id="pantheon-agent-detail"
      className="pantheon-agent-detail"
      aria-label={`${agentName} 独立输出`}
      aria-labelledby="pantheon-agent-detail-title"
      tabIndex={-1}
      data-testid="pantheon-agent-detail"
      data-agent-id={agentId}
      style={{ '--agent-accent': AGENT_ACCENTS[agentId] || 'var(--pantheon-accent)' } as React.CSSProperties}
    >
      <header className="pantheon-agent-detail-head">
        <PantheonAgentAvatar
          agentId={agentId}
          className="pantheon-agent-detail-avatar"
          status={running ? 'running' : avatarStatus(participant?.status)}
        />
        <span className="pantheon-agent-detail-title">
          <span className="pantheon-eyebrow">Agent 独立上下文</span>
          <strong id="pantheon-agent-detail-title">{agentName}</strong>
          <small>
            {running && <span className="pantheon-live-dot" aria-hidden />}
            {participant ? participantStatusLabel(participant.status) : '未加入'}
            {` · ${agentCapabilityLabel(agentId)}`}
            {facilitator ? ' · 汇总' : ''}
          </small>
          {running && (
            <AgentElapsed
              startedAt={startedAt}
              now={now}
              className="pantheon-agent-detail-elapsed"
              testId={`pantheon-agent-elapsed-detail-${agentId}`}
            />
          )}
        </span>
        <Button
          type="text"
          icon={<CloseOutlined />}
          aria-label={`关闭 ${agentName} 详情`}
          data-testid="pantheon-agent-detail-close"
          onClick={onClose}
        />
      </header>

      <div className="pantheon-agent-detail-stats">
        <span><strong>{outputs.length}</strong> 条观点</span>
        <span><strong>{traces.length}</strong> 次工具</span>
        <span><strong>{PHASE_LABEL[phase] || phase}</strong> 当前阶段</span>
      </div>

      <div className="pantheon-agent-detail-tabs" role="group" aria-label={`${agentName} 详情分类`}>
        <button
          type="button"
          aria-pressed={tab === 'output'}
          data-testid="pantheon-agent-tab-output"
          onClick={() => onTab('output')}
        >
          观点
          <span>{outputs.length}</span>
        </button>
        <button
          type="button"
          aria-pressed={tab === 'trace'}
          data-testid="pantheon-agent-tab-trace"
          onClick={() => onTab('trace')}
        >
          工具调用
          <span>{traces.length}</span>
        </button>
      </div>

      <div
        className="pantheon-agent-detail-body"
        role="region"
        aria-label={`${agentName} ${tab === 'output' ? '观点' : '工具调用'}`}
        aria-live={running && tab === 'output' ? 'polite' : undefined}
      >
        {tab === 'output' ? (
          outputs.length > 0 ? (
            <div className="pantheon-agent-output-list">
              {outputs.map((output) => (
                <article
                  key={output.id}
                  className={`pantheon-agent-output${output.streaming ? ' is-streaming' : ''}`}
                  data-testid={`pantheon-agent-output-${output.id}`}
                >
                  <header>
                    <span>
                      {output.round ? `第 ${output.round} 轮 · ` : ''}
                      {PHASE_LABEL[output.phase] || output.phase}
                    </span>
                    <small>
                      {output.streaming
                        ? '正在输出'
                        : output.durationMs
                          ? `${(output.durationMs / 1000).toFixed(1)}s`
                          : '独立观点'}
                    </small>
                  </header>
                  <div className={`pantheon-agent-output-markdown${output.streaming ? ' is-live' : ''}`}>
                    <MarkdownContent source={output.text} className="md-content" />
                    {output.streaming && (
                      <span className="pantheon-agent-output-caret pantheon-stream-caret" aria-label="正在输出" />
                    )}
                  </div>
                </article>
              ))}
            </div>
          ) : (
            <div className="pantheon-agent-detail-empty">
              {running ? <Spin size="small" /> : <CommentOutlined />}
              <strong>{running ? `正在${PHASE_LABEL[phase] || '处理'}…` : '等待该 Agent 形成观点'}</strong>
              <span>推理过程、工具请求与结果只在这里展示。</span>
            </div>
          )
        ) : (
          <ToolTraceList traces={traces} />
        )}
      </div>

      {participant?.status === 'failed' && (
        <p className="pantheon-agent-detail-error">{participant.error || '该 Agent 调用失败'}</p>
      )}
      <footer className="pantheon-agent-detail-foot">
        <ToolOutlined />
        <span>推理过程与工具调用不会写入中央共享讨论。</span>
      </footer>
    </aside>
  )
}

export const PantheonRecord: React.FC<Props> = ({
  session,
  loading,
  busy,
  error,
  liveStreams,
  onCancel,
  onSkipAgent,
  onRetryAgent,
  onRestart,
  onResume,
  onDelete,
  onRefresh,
  onExport,
  onExportJson,
  onCopy,
  onOpenResult,
  onSignOff,
  onContinue,
}) => {
  const meta = statusMeta(session.status)
  const running = session.status === 'running'
  const clockNow = useSecondClock(running)
  const streamRef = useRef<HTMLDivElement>(null)
  const detailRef = useRef<HTMLElement>(null)
  const detailTriggerRef = useRef<HTMLElement | null>(null)
  const [selectedAgentId, setSelectedAgentId] = useState('')
  const [detailTab, setDetailTab] = useState<'output' | 'trace'>('output')
  const [metaOpen, setMetaOpen] = useState(false)
  const [foldedEntries, setFoldedEntries] = useState<Record<string, boolean>>({})
  const toggleFolded = useCallback((id: string, folded: boolean) => {
    setFoldedEntries((current) => ({ ...current, [id]: folded }))
  }, [])

  const participants = useMemo<PantheonParticipant[]>(() => {
    if (session.participants.length > 0) return session.participants
    return (session.request.agentIds || []).map((agentId) => ({
      agentId,
      name: agentId,
      role: 'participant',
      status: 'waiting',
    }))
  }, [session.participants, session.request.agentIds])

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
        at: item.createdAt,
      }))
    }
    return (session.events ?? [])
      .filter((event) => event.kind === 'agent_output' && event.agentId)
      .map((event) => ({
        id: `event-${event.seq}`,
        agentId: event.agentId || '',
        agentName:
          participants.find((participant) => participant.agentId === event.agentId)?.name
          || event.agentId
          || '未知参与者',
        phase: event.phase,
        round: event.round || 0,
        text: event.text || '',
        durationMs: event.durationMs,
        at: event.at,
      }))
  }, [participants, session.contributions, session.events])

  const displayOutputs = useMemo(() => {
    const outputs = [...discussionOutputs]
    // 去重键按 agentId+round（对 phase 字符串漂移免疫）：直播条目与定稿同席
    // 同轮只允许出现一份——2026-10-31 实弹「所有输出重复两次」护栏。
    const keys = new Set(outputs.map((item) => `${item.agentId}:${item.round}`))
    for (const stream of liveStreams) {
      if (!stream.text) continue
      const key = `${stream.agentId}:${stream.round}`
      if (keys.has(key)) continue
      outputs.push({
        id: stream.id,
        agentId: stream.agentId,
        agentName:
          participants.find((participant) => participant.agentId === stream.agentId)?.name
          || stream.agentId,
        phase: stream.phase,
        round: stream.round,
        text: stream.text,
        streaming: stream.streaming,
        at: Number.MAX_SAFE_INTEGER,
      })
    }
    return outputs
  }, [discussionOutputs, liveStreams, participants])

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
          participants.find((participant) => participant.agentId === event.agentId)?.name
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
    return [...byID.values()].slice(-80)
  }, [participants, session.events])

  const outputsByAgent = useMemo(() => {
    const byAgent = new Map<string, DiscussionOutput[]>()
    for (const output of displayOutputs) {
      const list = byAgent.get(output.agentId)
      if (list) list.push(output)
      else byAgent.set(output.agentId, [output])
    }
    return byAgent
  }, [displayOutputs])

  const tracesByAgent = useMemo(() => {
    const byAgent = new Map<string, ToolTraceView[]>()
    for (const trace of toolTraces) {
      if (!trace.agentId) continue
      const list = byAgent.get(trace.agentId)
      if (list) list.push(trace)
      else byAgent.set(trace.agentId, [trace])
    }
    return byAgent
  }, [toolTraces])

  const activeSpeakers = useMemo(() => {
    const activeIds = new Set(
      liveStreams.filter((stream) => stream.streaming).map((stream) => stream.agentId),
    )
    for (const participant of participants) {
      if (participant.status === 'running') activeIds.add(participant.agentId)
    }
    return participants.filter((participant) => activeIds.has(participant.agentId))
  }, [liveStreams, participants])
  const skippableParticipant = participants.find((participant) => participant.status === 'running')
    || activeSpeakers[0]

  const agentStartedAt = useCallback((participant: PantheonParticipant): number => {
    if (participant.startedAt) return participant.startedAt
    for (let index = liveStreams.length - 1; index >= 0; index -= 1) {
      const stream = liveStreams[index]
      if (stream.agentId === participant.agentId && stream.streaming) {
        return stream.startedAt || stream.updatedAt || 0
      }
    }
    for (let index = session.events.length - 1; index >= 0; index -= 1) {
      const event = session.events[index]
      if (event.kind === 'agent_started' && event.agentId === participant.agentId) {
        return event.at
      }
    }
    return 0
  }, [liveStreams, session.events])

  const lastActivityAt = useMemo(() => {
    let latest = Math.max(session.updatedAt || 0, session.createdAt || 0)
    for (const event of session.events) latest = Math.max(latest, event.at || 0)
    for (const contribution of session.contributions ?? []) {
      latest = Math.max(latest, contribution.createdAt || 0)
    }
    for (const stream of liveStreams) {
      latest = Math.max(latest, stream.updatedAt || 0)
    }
    return latest
  }, [liveStreams, session.contributions, session.createdAt, session.events, session.updatedAt])

  const stalledFor = running ? Math.max(0, clockNow - lastActivityAt) : 0
  const stalled = running && lastActivityAt > 0 && stalledFor >= STALL_AFTER_MS
  const stalledAgentNames = activeSpeakers.map((participant) => participant.name)
  const stalledScope = stalledAgentNames.length
    ? joinNames(stalledAgentNames)
    : `“${PHASE_LABEL[session.phase] || session.phase}”阶段`
  const failedParticipants = participants.filter((participant) => participant.status === 'failed')
  const latestAgentError = [...session.events]
    .reverse()
    .find((event) => event.kind === 'agent_error')
  const failedEventName = latestAgentError?.agentId
    ? participants.find((participant) => participant.agentId === latestAgentError.agentId)?.name
      || latestAgentError.agentId
    : ''
  const failedScope = failedParticipants.length
    ? joinNames(failedParticipants.map((participant) => participant.name))
    : failedEventName || '本机 Agent'
  const failedDetail = failedParticipants.find((participant) => participant.error)?.error
    || latestAgentError?.error
    || session.error
    || '调用没有正常返回'
  const runningFailure = running && (failedParticipants.length > 0 || Boolean(latestAgentError))
  const terminalRecoverable = !running &&
    (session.status === 'failed' || session.status === 'cancelled')
  const showFailureRecovery = !stalled && (runningFailure || terminalRecoverable)

  const completedCount = participants.filter((participant) => participant.status === 'finished').length
  const untilConsensus = Boolean(session.request.untilConsensus)
  const totalRounds = Math.max(1, Math.min(20, session.request.rounds || 2))
  const protocol = session.request.protocol
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

  const sharedEntries = useMemo<SharedEntry[]>(() => {
    const entries: SharedEntry[] = [
      {
        id: 'shared-topic',
        kind: 'topic',
        label: '你发起的议题',
        text: session.request.topic || session.title,
        meta: [
          `${participants.length} 位 Agent`,
          untilConsensus ? `至共识（≤${totalRounds} 轮）` : `${totalRounds} 轮`,
          session.request.attachments?.length ? `${session.request.attachments.length} 张图片` : '',
          '只读',
        ].filter(Boolean).join(' · '),
        at: session.createdAt,
      },
      {
        id: 'shared-orchestrator',
        kind: 'orchestrator',
        label: '共享讨论',
        text: `已组织 ${participants.length} 位 Agent 在独立上下文中协作。共享窗口会按发言顺序展示各席位的公开观点；推理过程和工具调用可在上方 Agent 节点中查看。`,
        meta: session.request.chairmanAgentId
          ? `由 ${participants.find((item) => item.agentId === session.request.chairmanAgentId)?.name || session.request.chairmanAgentId} 负责汇总`
          : undefined,
        at: session.createdAt + 1,
      },
    ]

    const seenEvents = new Set<string>()
    for (const event of session.events) {
      if (!['phase', 'consensus', 'warning', 'error', 'agent_skipped', 'agent_retry'].includes(event.kind)) continue
      if (!event.text) continue
      const key = `${event.kind}:${event.phase}:${event.round || 0}:${event.text}`
      if (seenEvents.has(key)) continue
      seenEvents.add(key)
      entries.push({
        id: `shared-event-${event.seq}`,
        kind: event.kind === 'warning' || event.kind === 'error' ? 'warning' : 'event',
        label: eventTitle(event),
        text: event.text,
        meta: [
          event.round ? `第 ${event.round} 轮` : '',
          PHASE_LABEL[event.phase] || event.phase,
        ].filter(Boolean).join(' · '),
        at: event.at,
      })
    }

    for (const output of displayOutputs) {
      entries.push({
        id: `shared-output-${output.id}`,
        kind: 'agent',
        label: output.agentName,
        text: output.text,
        meta: [
          output.round ? `第 ${output.round} 轮` : '',
          PHASE_LABEL[output.phase] || output.phase,
          output.durationMs ? `用时 ${(output.durationMs / 1000).toFixed(1)}s` : '',
        ].filter(Boolean).join(' · '),
        at: output.streaming ? Number.MAX_SAFE_INTEGER - 1 : output.at || session.updatedAt,
        agentId: output.agentId,
        streaming: output.streaming,
      })
    }

    entries.sort((a, b) => a.at - b.at)

    if (running) {
      const speakerNames = activeSpeakers.map((participant) => participant.name)
      entries.push({
        id: 'shared-live',
        kind: 'live',
        label: speakerNames.length ? '正在协作' : '共享讨论等待中',
        text: speakerNames.length
          ? `${joinNames(speakerNames)}正在${PHASE_LABEL[session.phase] || '处理'}，观点会按返回顺序进入共享线程。`
          : `正在${PHASE_LABEL[session.phase] || '讨论'}，等待下一位 Agent 返回。`,
        meta: `${completedCount} / ${participants.length} 位已返回 · 推理与工具过程保留在独立 Agent 详情`,
        at: Number.MAX_SAFE_INTEGER,
      })
    }

    return entries
  }, [
    activeSpeakers,
    completedCount,
    displayOutputs,
    participants,
    running,
    session.createdAt,
    session.events,
    session.phase,
    session.request.attachments,
    session.request.chairmanAgentId,
    session.request.topic,
    session.title,
    session.updatedAt,
    totalRounds,
    untilConsensus,
  ])

  const publicTimeline = useMemo(() => (
    [...session.events]
      .filter((event) => !['agent_output', 'tool_started', 'tool_result'].includes(event.kind))
      .reverse()
      .slice(0, PUBLIC_EVENT_LIMIT)
  ), [session.events])

  const streamStickRef = useRef(true)
  // 共享流自身限高滚动：贴底跟随锚在流容器上（而非整页 canvas），
  // 上滑即停跟，滚回底部自动恢复（2026-10-30 共享窗口优化刀）。
  useEffect(() => {
    const stream = streamRef.current
    if (!stream) return
    const onScroll = () => {
      const distance = stream.scrollHeight - stream.scrollTop - stream.clientHeight
      streamStickRef.current = distance < 80
    }
    stream.addEventListener('scroll', onScroll, { passive: true })
    return () => stream.removeEventListener('scroll', onScroll)
  }, [])

  useEffect(() => {
    if (!running || selectedAgentId) return
    const stream = streamRef.current
    if (stream && streamStickRef.current) {
      if (typeof stream.scrollTo === 'function') {
        stream.scrollTo({ top: stream.scrollHeight, behavior: 'instant' })
      } else {
        stream.scrollTop = stream.scrollHeight
      }
    }
  }, [running, selectedAgentId, sharedEntries, foldedEntries])

  const closeAgentDetail = useCallback(() => {
    setSelectedAgentId('')
    window.requestAnimationFrame(() => detailTriggerRef.current?.focus())
  }, [])

  useEffect(() => {
    if (!selectedAgentId) return
    const frame = window.requestAnimationFrame(() => detailRef.current?.focus())
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') closeAgentDetail()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => {
      window.cancelAnimationFrame(frame)
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [closeAgentDetail, selectedAgentId])

  useEffect(() => {
    if (selectedAgentId && !participants.some((participant) => participant.agentId === selectedAgentId)) {
      setSelectedAgentId('')
    }
  }, [participants, selectedAgentId])

  const selectAgent = (agentId: string, trigger: HTMLElement) => {
    detailTriggerRef.current = trigger
    setDetailTab('output')
    setSelectedAgentId(agentId)
  }

  const selectedParticipant = participants.find(
    (participant) => participant.agentId === selectedAgentId,
  )
  const selectedName = selectedParticipant?.name || selectedAgentId
  const selectedStartedAt = selectedParticipant ? agentStartedAt(selectedParticipant) : 0

  return (
    <div
      className={`pantheon-council-stage${selectedAgentId ? ' is-detail-open' : ''}`}
      data-testid="pantheon-record"
    >
      <style>{mdStyles}</style>
      <article className="pantheon-record-main">
        <header className="pantheon-record-head">
          <div className="pantheon-record-title">
            <div className="pantheon-record-kicker">
              <span className="pantheon-eyebrow">共享讨论纪要</span>
              <span>{PHASE_LABEL[session.phase] || session.phase}</span>
              {session.request.workspace && (
                <span className="pantheon-record-workspace" title={session.request.workspace}>
                  <FolderOpenOutlined />
                  {workspaceName(session.request.workspace)}
                </span>
              )}
            </div>
            <h2 title={session.title}>{session.title}</h2>
            {session.request.topic && session.request.topic !== session.title && (
              <p>{session.request.topic}</p>
            )}
          </div>
          <div className="pantheon-record-actions">
            <Tag icon={meta.icon} color={meta.color}>{meta.label}</Tag>
            <Tooltip title="讨论设置、参与者与证据">
              <Button
                type="text"
                icon={<InfoCircleOutlined />}
                aria-expanded={metaOpen}
                aria-controls="pantheon-meta"
                data-testid="pantheon-meta-toggle"
                onClick={() => setMetaOpen((current) => !current)}
              >
                讨论信息
              </Button>
            </Tooltip>
            <Tooltip title="刷新卷宗">
              <Button
                type="text"
                icon={<ReloadOutlined />}
                aria-label="刷新卷宗"
                onClick={onRefresh}
                loading={loading}
              />
            </Tooltip>
            {running ? (
              <>
                {skippableParticipant && (
                  <>
                    <Button
                      icon={<ReloadOutlined />}
                      onClick={() => onRetryAgent(skippableParticipant.agentId)}
                      disabled={busy}
                      data-testid="pantheon-retry-current"
                    >
                      重试当前
                    </Button>
                    <Button
                      icon={<StepForwardOutlined />}
                      onClick={() => onSkipAgent(skippableParticipant.agentId)}
                      disabled={busy}
                      data-testid="pantheon-skip-current"
                    >
                      跳过当前
                    </Button>
                  </>
                )}
                <Button danger icon={<StopOutlined />} onClick={onCancel} disabled={busy}>
                  终止
                </Button>
              </>
            ) : (
              <>
                <Tooltip title="复制 Markdown">
                  <Button type="text" icon={<CopyOutlined />} aria-label="复制 Markdown" onClick={onCopy} />
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
                  <Button
                    type="text"
                    danger
                    icon={<DeleteOutlined />}
                    aria-label="删除议事录"
                    onClick={onDelete}
                  />
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
        {stalled && (
          <section
            className="pantheon-stall-notice"
            role="status"
            aria-live="polite"
            data-testid="pantheon-stall-notice"
          >
            <WarningOutlined aria-hidden />
            <span>
              <strong>{stalledScope} 已超过 2 分钟没有新输出</strong>
              <small>
                已持续等待 {formatElapsed(stalledFor)}。可只重试当前席位，或跳过它进入下一位；此前发言都会保留。
              </small>
            </span>
            {skippableParticipant && (
              <>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => onRetryAgent(skippableParticipant.agentId)}
                  disabled={busy}
                  data-testid="pantheon-retry-agent"
                >
                  重试 {skippableParticipant.name}
                </Button>
                <Button
                  icon={<StepForwardOutlined />}
                  onClick={() => onSkipAgent(skippableParticipant.agentId)}
                  disabled={busy}
                  data-testid="pantheon-skip-agent"
                >
                  跳过 {skippableParticipant.name}
                </Button>
              </>
            )}
          </section>
        )}
        {showFailureRecovery && (
          <section
            className="pantheon-stall-notice is-failure"
            role="alert"
            aria-live="assertive"
            data-testid="pantheon-failure-notice"
          >
            <CloseCircleFilled aria-hidden />
            <span>
              <strong>
                {runningFailure
                  ? `${failedScope} 调用失败，讨论可能停在等待状态`
                  : session.status === 'cancelled' ? '本次讨论已终止' : '本次讨论未能完成'}
              </strong>
              <small>
                {runningFailure
                  ? `${failedDetail}。当前记录会保留；若后续席位仍未继续，可使用顶部“跳过当前”。`
                  : '当前记录会保留；如确需从头重来，可另建一份全新的整场记录。'}
              </small>
            </span>
            {!runningFailure && (
              <>
                {(session.contributions ?? []).length > 0 && (
                  <Button
                    type="primary"
                    icon={<PlayCircleOutlined />}
                    onClick={onResume}
                    disabled={busy}
                    data-testid="pantheon-resume"
                  >
                    恢复讨论
                  </Button>
                )}
                <Button
                  icon={<ReloadOutlined />}
                  onClick={onRestart}
                  disabled={busy}
                  data-testid="pantheon-restart-all"
                >
                  重开整场
                </Button>
              </>
            )}
          </section>
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

        {metaOpen && (
          <section
            className="pantheon-meta-disclosure"
            id="pantheon-meta"
            aria-label="讨论纪要摘要"
            data-testid="pantheon-meta"
          >
            <div className="pantheon-meta-grid">
              <div>
                <span className="pantheon-eyebrow">讨论范围</span>
                <dl className="pantheon-facts">
                  <div>
                    <dt>讨论轮数</dt>
                    <dd>{untilConsensus ? `不限（≤${totalRounds} 轮）` : totalRounds}</dd>
                  </div>
                  <div>
                    <dt>议事路径</dt>
                    <dd title={protocol === 'independent' ? '第 1 轮不注入同轮前序，各自独立作答' : '同轮发言按顺序互相可见'}>
                      {protocolLabel(session.request.protocol)}
                    </dd>
                  </div>
                  <div><dt>运行边界</dt><dd>只读</dd></div>
                  <div><dt>当前进度</dt><dd>{PHASE_LABEL[session.phase] || session.phase}</dd></div>
                  <div>
                    <dt>主持与汇总</dt>
                    <dd>
                      {participants.find(
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
                  {session.request.parentSessionId && (
                    <div className="pantheon-fact-wide">
                      <dt>父会话</dt>
                      <dd title="续议承接以简报为准，不继承全文">{session.request.parentSessionId}</dd>
                    </div>
                  )}
                </dl>
              </div>
              <div>
                <span className="pantheon-eyebrow">参与者</span>
                <ul className="pantheon-seat-list">
                  {participants.map((participant) => (
                    <li
                      key={participant.agentId}
                      data-agent-id={participant.agentId}
                      aria-current={participant.status === 'running' ? 'true' : undefined}
                      className={participant.status === 'running' ? 'is-speaking' : ''}
                      style={{
                        '--agent-accent': AGENT_ACCENTS[participant.agentId] || 'var(--pantheon-accent)',
                      } as React.CSSProperties}
                    >
                      <PantheonAgentAvatar
                        agentId={participant.agentId}
                        className="pantheon-meta-agent-avatar"
                        status={avatarStatus(participant.status)}
                      />
                      <span>{participant.name}</span>
                      <small>
                        {participantOrderLabel(session.request.agentIds, participant.agentId)}
                        {participantStatusLabel(participant.status)}
                        {participant.status === 'running' && agentStartedAt(participant) > 0
                          ? ` · 工作 ${formatElapsed(clockNow - agentStartedAt(participant))}`
                          : ''}
                      </small>
                    </li>
                  ))}
                </ul>
              </div>
            </div>

            <div className="pantheon-meta-grid is-lower">
              <div>
                <span className="pantheon-eyebrow">证据与警示</span>
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
                {Boolean(session.request.attachments?.length) && (
                  <div className="pantheon-record-attachments">
                    <span>图片附件</span>
                    <ul>
                      {session.request.attachments?.map((attachment) => (
                        <li key={attachment.path} title={attachment.path}>
                          <PictureOutlined aria-hidden />
                          <span>{attachment.name}</span>
                        </li>
                      ))}
                    </ul>
                    <small>已先由本地视觉模型提取共享证据</small>
                  </div>
                )}
                {(() => {
                  const visibleWarnings = (session.warnings ?? []).filter((warning) => warning !== session.error)
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
              </div>
              <div>
                <span className="pantheon-eyebrow">公开进度</span>
                <div className="pantheon-event-stream">
                  {publicTimeline.length === 0 && (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="等待讨论事件" />
                  )}
                  {publicTimeline.map((event) => (
                    <article key={`${event.sessionId}-${event.seq}`} className={`pantheon-event is-${event.kind}`}>
                      <div className="pantheon-event-line">
                        <strong>{eventTitle(event)}</strong>
                        <time>{formatTime(event.at)}</time>
                      </div>
                      {event.error && <p className="pantheon-event-error">{event.error}</p>}
                      {event.text && <p>{event.text}</p>}
                    </article>
                  ))}
                </div>
              </div>
            </div>
          </section>
        )}

        <section className="pantheon-team-map" aria-labelledby="pantheon-team-map-title">
          <header className="pantheon-team-map-head">
            <div>
              <span className="pantheon-eyebrow">团队协作拓扑</span>
              <h3 id="pantheon-team-map-title">一个共享主线程，多个独立 Agent</h3>
            </div>
            <div className="pantheon-team-map-side">
              {session.createdAt > 0 && (
                <span
                  className="pantheon-total-elapsed"
                  data-testid="pantheon-total-elapsed"
                  aria-label={running ? '讨论总时长（实时）' : '讨论总用时'}
                >
                  {running
                    ? `已进行 ${formatElapsed(clockNow - session.createdAt)}`
                    : `总用时 ${formatElapsed((session.updatedAt || session.createdAt) - session.createdAt)}`}
                </span>
              )}
              <span className="pantheon-team-map-hint">点击节点查看独立输出</span>
            </div>
          </header>

          <div
            className="pantheon-topology"
            style={{ '--agent-count': Math.max(participants.length, 1) } as React.CSSProperties}
          >
            <div className="pantheon-topology-svg" aria-hidden>
              <svg viewBox="0 0 100 100" preserveAspectRatio="none">
                {participants.map((participant, index) => {
                  const x = ((index + 0.5) / Math.max(participants.length, 1)) * 100
                  return (
                    <path
                      key={participant.agentId}
                      d={`M 50 30 C 50 48, ${x} 50, ${x} 72`}
                      style={{
                        color: AGENT_ACCENTS[participant.agentId] || 'var(--pantheon-accent)',
                      }}
                      className={activeSpeakers.some((speaker) => speaker.agentId === participant.agentId)
                        ? 'is-active'
                        : ''}
                    />
                  )
                })}
              </svg>
            </div>

            <div className="pantheon-topology-hub">
              <span className="pantheon-topology-hub-icon" aria-hidden>
                <TeamOutlined />
              </span>
              <span>
                <strong>共享讨论</strong>
                <small>
                  {running && <span className="pantheon-live-dot" aria-hidden />}
                  {PHASE_LABEL[session.phase] || session.phase}
                </small>
              </span>
            </div>

            <div className="pantheon-topology-nodes">
              {participants.map((participant, index) => {
                const active = activeSpeakers.some((speaker) => speaker.agentId === participant.agentId)
                const selected = selectedAgentId === participant.agentId
                const facilitator = session.request.chairmanAgentId === participant.agentId
                const outputs = outputsByAgent.get(participant.agentId) ?? []
                const traces = tracesByAgent.get(participant.agentId) ?? []
                const status = active ? 'running' : participant.status
                const startedAt = agentStartedAt(participant)
                return (
                  <button
                    key={participant.agentId}
                    type="button"
                    className={[
                      'pantheon-topology-node',
                      `is-${status}`,
                      selected ? 'is-selected' : '',
                    ].filter(Boolean).join(' ')}
                    data-testid={`pantheon-topology-node-${participant.agentId}`}
                    data-agent-id={participant.agentId}
                    aria-current={active ? 'true' : undefined}
                    aria-expanded={selected}
                    aria-controls="pantheon-agent-detail"
                    aria-label={[
                      participant.name,
                      participantStatusLabel(status),
                      active && startedAt > 0
                        ? `已持续工作 ${formatElapsed(clockNow - startedAt)}`
                        : '',
                      `${outputs.length} 条观点`,
                      `${traces.length} 次工具调用`,
                      '点击查看独立输出',
                    ].filter(Boolean).join('，')}
                    title={`查看 ${participant.name} 的独立输出`}
                    onClick={(event) => selectAgent(participant.agentId, event.currentTarget)}
                    style={{
                      '--agent-accent': AGENT_ACCENTS[participant.agentId] || 'var(--pantheon-accent)',
                      '--node-order': index,
                    } as React.CSSProperties}
                  >
                    <span className="pantheon-topology-order">{String(index + 1).padStart(2, '0')}</span>
                    <PantheonAgentAvatar
                      agentId={participant.agentId}
                      className="pantheon-topology-avatar"
                      status={active ? 'running' : avatarStatus(participant.status)}
                    />
                    <span className="pantheon-topology-copy">
                      <strong>{participant.name}</strong>
                      <small>
                        {active && <span className="pantheon-live-dot" aria-hidden />}
                        {participantStatusLabel(status)}
                        {facilitator
                          ? ' · 汇总'
                          : ` · ${agentCapabilityLabel(participant.agentId)}`}
                      </small>
                    </span>
                    <span className="pantheon-topology-metrics">
                      <span>观点 {outputs.length}</span>
                      <span>工具 {traces.length}</span>
                      {active && startedAt > 0 && (
                        <AgentElapsed
                          startedAt={startedAt}
                          now={clockNow}
                          className="pantheon-topology-elapsed"
                          testId={`pantheon-agent-elapsed-${participant.agentId}`}
                        />
                      )}
                    </span>
                  </button>
                )
              })}
            </div>
          </div>
        </section>

        <section
          className={`pantheon-shared-dialogue${running ? ' is-live' : ''}`}
          aria-labelledby="pantheon-shared-title"
          data-testid="pantheon-shared-dialogue"
        >
          <header className="pantheon-shared-head">
            <div>
              <span className="pantheon-eyebrow">{running ? '主线程实时' : '主线程归档'}</span>
              <h3 id="pantheon-shared-title">
                {running && <span className="pantheon-live-dot" aria-hidden />}
                共享讨论
              </h3>
              <p>按实际发言顺序展示各 Agent 的公开观点；推理过程和工具调用保留在独立详情中。</p>
            </div>
            <div className="pantheon-shared-head-meta">
              <strong>
                {untilConsensus
                  ? `第 ${activeRound} 轮 · 至共识`
                  : `第 ${activeRound} / ${totalRounds} 轮`}
              </strong>
              <span>{completedCount} / {participants.length} 位已返回</span>
            </div>
          </header>

          <div
            ref={streamRef}
            className="pantheon-stream pantheon-shared-stream"
            data-testid="pantheon-stream"
            role="log"
            aria-live={running ? 'polite' : undefined}
            aria-label="共享讨论消息"
          >
            {sharedEntries.map((entry) => (
              <article
                key={entry.id}
                className={`pantheon-shared-entry is-${entry.kind}${entry.streaming ? ' is-streaming' : ''}`}
                data-testid={`pantheon-shared-entry-${entry.id}`}
                data-agent-id={entry.agentId}
                style={entry.agentId ? {
                  '--agent-accent': AGENT_ACCENTS[entry.agentId] || 'var(--pantheon-accent)',
                } as React.CSSProperties : undefined}
              >
                <span className="pantheon-shared-avatar" aria-hidden>
                  {entry.agentId ? (
                    <PantheonAgentAvatar
                      agentId={entry.agentId}
                      className="pantheon-shared-agent-avatar"
                      status={entry.streaming ? 'running' : 'finished'}
                    />
                  ) : entry.kind === 'topic'
                    ? '你'
                    : entry.kind === 'warning'
                      ? <WarningOutlined />
                      : entry.kind === 'live'
                        ? <span className="pantheon-live-dot" />
                        : <TeamOutlined />}
                </span>
                <div className="pantheon-shared-copy">
                  <header>
                    <strong>{entry.label}</strong>
                    {(entry.kind === 'live' || entry.streaming) && (
                      <span className="pantheon-streaming-state">
                        {entry.streaming ? '正在发言' : '正在发生'}
                      </span>
                    )}
                    <time>{entry.at === Number.MAX_SAFE_INTEGER ? '现在' : formatClock(entry.at)}</time>
                  </header>
                  {entry.kind === 'agent' ? (
                    <SharedAgentBody
                      entry={entry}
                      foldedMap={foldedEntries}
                      onFold={toggleFolded}
                    />
                  ) : (
                    <p>{entry.text}</p>
                  )}
                  {entry.meta && <small>{entry.meta}</small>}
                </div>
              </article>
            ))}
          </div>

          {!running && (
            <div className="pantheon-record-body" aria-label="讨论小结">
              <ConclusionPanel
                session={session}
                busy={busy}
                onOpenResult={onOpenResult}
                onSignOff={onSignOff}
                onContinue={onContinue}
              />
            </div>
          )}
        </section>
      </article>

      {selectedAgentId && (
        <AgentDetailPanel
          participant={selectedParticipant}
          agentId={selectedAgentId}
          agentName={selectedName}
          outputs={outputsByAgent.get(selectedAgentId) ?? []}
          traces={tracesByAgent.get(selectedAgentId) ?? []}
          facilitator={session.request.chairmanAgentId === selectedAgentId}
          phase={session.phase}
          startedAt={selectedStartedAt}
          now={clockNow}
          tab={detailTab}
          onTab={setDetailTab}
          onClose={closeAgentDetail}
          panelRef={detailRef}
        />
      )}
    </div>
  )
}
