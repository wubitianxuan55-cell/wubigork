import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  CheckCircleFilled,
  CheckCircleOutlined,
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
  NodeIndexOutlined,
  PictureOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  StopOutlined,
  StepForwardOutlined,
  TeamOutlined,
  ToolOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { Alert, Button, Dropdown, Empty, Input, Spin, Tag, Tooltip } from 'antd'
import { MarkdownContent, mdStyles } from '../../components/MarkdownContent'
import { AGENT_ACCENTS, agentCapabilityLabel, presetLabel, protocolLabel } from './visuals'
import { PantheonAgentAvatar } from './PantheonAgentAvatar'
import { chairAssignmentReceipts, receiptStatusLabel, type PantheonChairReceipt } from './chairReceipts'
import type {
  PantheonChairAssignment,
  PantheonChairDecision,
  PantheonContribution,
  PantheonEvent,
  PantheonIssue,
  PantheonIssueReview,
  PantheonLiveStream,
  PantheonParticipant,
  PantheonSeatUsage,
  PantheonSession,
  PantheonToolEvidence,
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
  onAddUserNote: (content: string) => void
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
  onAnswer: (questionId: string, body: string) => void
  onResolveGate: (action: 'resolve' | 'dismiss', reason: string) => void
}

const PHASE_LABEL: Record<string, string> = {
  preparing: '准备',
  opening: '开场陈述',
  discussion: '补充论证',
  synthesis: '协作汇总',
  attestation: '独立确认',
  independent: '开场陈述',
  review: '互相澄清',
  challenge: '补充论证',
  verdict: '协作汇总',
  completed: '已完成',
  failed: '失败',
  cancelled: '已终止',
}

const DISCUSSION_ACT_META: Record<string, { label: string; hint: string }> = {
  claim: { label: '主张', hint: '提出或重述一条可讨论的主张' },
  challenge: { label: '质疑', hint: '针对具体论证提出证据或适用条件问题' },
  respond: { label: '回应', hint: '对既有质疑给出接受、部分接受或拒绝' },
  revise: { label: '修正', hint: '依据前文证据修订自己的主张' },
  reinforce: { label: '补强', hint: '补充证据或推理支撑既有主张' },
  broaden: { label: '拓宽', hint: '补充新的视角、边界或备选方案' },
}

const RESPONSE_VERDICT_LABEL: Record<string, string> = {
  accept: '接受',
  partial: '部分接受',
  reject: '拒绝',
}

const ISSUE_ORIGIN_LABEL: Record<string, string> = {
  challenge: '定向质疑',
  residual: '剩余分歧',
  user: '用户插话',
}

const TOOL_TRACE_LIMIT = 60
const PUBLIC_EVENT_LIMIT = 10
const AGENT_SLOW_AFTER_MS = 60_000
const STALL_AFTER_MS = 120_000
// 共享流折叠：超过此长度的已完成发言默认收起只留预览；流式中的发言永不折叠。
const FOLDABLE_CHARS = 160
const FOLD_DEFAULT_CHARS = 420
const FOLD_PREVIEW_CHARS = 72
const WINDOW_PREVIEW_CHARS = 150

type RecordView = 'topology' | 'shared' | 'agents' | 'ledger'

function charCount(text: string): number {
  return Array.from(text).length
}

function previewText(text: string): string {
  const flat = text.replace(/\s+/g, ' ').trim()
  const chars = Array.from(flat)
  if (chars.length <= FOLD_PREVIEW_CHARS) return flat
  return `${chars.slice(0, FOLD_PREVIEW_CHARS).join('')}…`
}

function windowPreviewText(text: string): string {
  const flat = text.replace(/\s+/g, ' ').trim()
  const chars = Array.from(flat)
  if (chars.length <= WINDOW_PREVIEW_CHARS) return flat
  return `${chars.slice(0, WINDOW_PREVIEW_CHARS).join('')}…`
}

function seatReturnsResult(status: string): boolean {
  return status === 'finished' || status === 'finished_unverified'
}

function shortProtocolID(id: string): string {
  if (id.length <= 28) return id
  return `${id.slice(0, 14)}…${id.slice(-8)}`
}

const DiscussionActBadge: React.FC<{
  act?: string
  testId?: string
}> = ({ act, testId }) => {
  const meta = act ? DISCUSSION_ACT_META[act] : undefined
  if (!act || !meta) return null
  return (
    <span
      className={`pantheon-discussion-act is-${act}`}
      title={meta.hint}
      data-testid={testId}
    >
      {meta.label}
    </span>
  )
}

const DiscussionActDetail: React.FC<{
  targetContributionId?: string
  targetIssueId?: string
  challengedClaim?: string
  responseVerdict?: string
}> = ({
  targetContributionId,
  targetIssueId,
  challengedClaim,
  responseVerdict,
}) => {
  const responseLabel = responseVerdict ? RESPONSE_VERDICT_LABEL[responseVerdict] : ''
  if (!targetContributionId && !targetIssueId && !challengedClaim && !responseLabel) return null
  return (
    <div className="pantheon-discussion-relation">
      {challengedClaim && (
        <span>
          被挑战命题：<strong>{challengedClaim}</strong>
        </span>
      )}
      {targetContributionId && (
        <span>
          目标贡献：
          <code title={targetContributionId}>{shortProtocolID(targetContributionId)}</code>
        </span>
      )}
      {targetIssueId && (
        <span>
          目标分歧：
          <code title={targetIssueId}>{shortProtocolID(targetIssueId)}</code>
        </span>
      )}
      {responseLabel && <span>回应结论：<strong>{responseLabel}</strong></span>}
    </div>
  )
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
  discussionAct?: string
  targetContributionId?: string
  targetIssueId?: string
  challengedClaim?: string
  responseVerdict?: string
  usage?: PantheonSeatUsage
  chairAssignments?: PantheonChairAssignment[]
  chairDecision?: PantheonChairDecision
  toolEvidence?: PantheonToolEvidence
  receipts?: PantheonChairReceipt[]
  /** M1 输入侧对账锚：席位实际看到的任务书（截断后）指纹与字数。 */
  promptSha256?: string
  promptRunes?: number
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
  kind: 'topic' | 'orchestrator' | 'agent' | 'event' | 'live' | 'warning' | 'user' | 'working'
  label: string
  text: string
  meta?: string
  at: number
  round?: number
  phaseLabel?: string
  durationMs?: number
  agentId?: string
  streaming?: boolean
  discussionAct?: string
  targetContributionId?: string
  targetIssueId?: string
  challengedClaim?: string
  responseVerdict?: string
  /** 共享账对账警示（agent 卡体下挂；meta 对 agent 卡不渲染）。 */
  mismatchNote?: string
}

interface SharedTimelineSection {
  key: string
  heading?: {
    round?: number
    phase: string
    count: number
  }
  entries: SharedEntry[]
}

type AvatarStatus = 'online' | 'offline' | 'running' | 'waiting' | 'finished' | 'skipped' | 'failed' | 'unverified'

function participantStatusLabel(status: string): string {
  switch (status) {
    case 'running': return '正在发言'
    case 'finished': return '已返回'
    case 'finished_unverified': return '未验证返回'
    case 'needs_input': return '等待回答'
    case 'skipped': return '已跳过'
    case 'retrying': return '正在重试'
    case 'failed': return '失败'
    case 'unavailable': return '不可用'
    case 'cancelled': return '已终止'
    case 'waiting': return '候场'
    default: return status
  }
}

/** 席位引擎账目（2026-11-06）：缓存命中率只在引擎真的上报 usage 时才有值。
 *  未上报 ≠ 0%：前者显示空，后者才显示 0%。engineSessionId 进 title，供
 *  用户去引擎侧核对这一席确实调用了 DSH，而不是只看品牌名。 */
function seatUsageLabel(usage?: PantheonSeatUsage): string {
  if (!usage) return ''
  const parts: string[] = []
  const cached = usage.cacheReadTokens ?? 0
  const input = usage.inputTokens ?? 0
  if (cached + input > 0) {
    parts.push(`缓存命中 ${Math.round((cached * 100) / (cached + input))}%`)
  }
  if (usage.steps) parts.push(`${usage.steps} 步`)
  return parts.join(' · ')
}

function seatUsageTitle(usage?: PantheonSeatUsage): string {
  if (!usage) return ''
  const lines: string[] = []
  if (usage.engineSessionId) lines.push(`引擎会话：${usage.engineSessionId}`)
  if (usage.cacheReadTokens || usage.inputTokens) {
    lines.push(`缓存命中 ${usage.cacheReadTokens ?? 0} / 新增 ${usage.inputTokens ?? 0} prompt token`)
  }
  return lines.join('\n')
}

/** E1 工具证据摘要：宿主实际看见的调用，不是模型自述。 */
function seatToolEvidenceLabel(evidence?: PantheonToolEvidence): string {
  if (!evidence) return ''
  const calls = evidence.calls ?? 0
  const ok = evidence.ok ?? 0
  const failed = evidence.failed ?? 0
  const nonEmpty = evidence.nonEmpty ?? 0
  const unresolved = Math.max(0, calls - ok - failed)
  const detail = [
    ok ? `${ok} 成功` : '',
    failed ? `${failed} 失败` : '',
    nonEmpty ? `${nonEmpty} 次有非空返回` : '',
    unresolved ? `${unresolved} 未回报` : '',
  ].filter(Boolean)
  return detail.length > 0 ? `${calls} 次工具（${detail.join(' / ')}）` : `${calls} 次工具`
}

function seatToolEvidenceTitle(evidence?: PantheonToolEvidence): string {
  if (!evidence) return ''
  const lines: string[] = []
  if (evidence.tools?.length) {
    lines.push(`工具：${evidence.tools.map((tool) => `${tool.name}×${tool.count}`).join('、')}`)
  }
  if ((evidence.repeatFailures ?? 0) >= 2) {
    lines.push(`同一工具连续失败 ${evidence.repeatFailures} 次：${evidence.repeatTool ?? '未知工具'}`)
  }
  lines.push('宿主观察，不是质量分')
  return lines.join('\n')
}

/** M1 任务书快照：席位实际看到的任务书（截断后）指纹与字数——tooltip 行。 */
function seatPromptTitle(promptSha256?: string, promptRunes?: number): string {
  if (!promptSha256) return ''
  return `任务书快照：${promptRunes ?? '?'} 字 · sha256:${promptSha256}`
}

/** 共享账对账线（与导出 evidenceMismatchNote 同口径）：陈词标注【工具核验】
 * 但本次调用没有任何非空工具返回 → 不可对证警示。 */
function seatMismatchLabel(evidence?: PantheonToolEvidence, text?: string): string {
  if (!evidence) return ''
  if ((evidence.nonEmpty ?? 0) > 0) return ''
  if (!text || !text.includes('【工具核验】')) return ''
  return '⚠ 工具核验不可对证（本次调用无非空工具返回）'
}

function chairDecisionLabel(status?: string): string {
  switch (status) {
    case 'completed': return '建议收口'
    case 'needs-input': return '需人回答'
    case 'blocked': return '被阻塞'
    case 'continue': return '继续'
    default: return status || '继续'
  }
}

/** 席位活性：观测超过 90 秒没有更新，"正在发言"就该让位给"活性未知"。
 *  这不是失败——外部进程可能只是安静；但界面不能替它保证在干活（M2/M3）。 */
const SEAT_STALE_AFTER_MS = 90_000

function seatLivenessNote(participant: PantheonParticipant, clockNow: number): string {
  if (participant.status !== 'running' && participant.status !== 'retrying') return ''
  const observedAt = participant.observed?.at
  if (!observedAt) return ''
  const age = clockNow - observedAt
  if (age < SEAT_STALE_AFTER_MS) return ''
  return `活性未知（最近观测 ${Math.round(age / 1000)} 秒前）`
}

function seatStatusText(
  participant: PantheonParticipant,
  clockNow: number,
): string {
  const label = participantStatusLabel(participant.status)
  const note = seatLivenessNote(participant, clockNow)
  return note ? `${label} · ${note}` : label
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
    case 'finished_unverified':
      return 'unverified'
    case 'needs_input':
      return 'waiting'
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
    case 'waiting_input':
      return { label: '等待回答', color: 'warning', icon: <ClockCircleOutlined /> }
    case 'interrupted':
      return { label: '已中断', color: 'warning', icon: <StopOutlined /> }
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

// 等待面板（O-E）：整场停在 waiting_input 时，唯一的出路是把这里处理干净。
// 问题与门都从事件推导，不另存字段——重启后重新拉一次快照就能原样重建。
function pendingQuestionItems(events: PantheonEvent[]): Array<{
  question: NonNullable<PantheonEvent['question']>
  text: string
}> {
  const answered = new Set(
    events
      .filter((event) => event.kind === 'answer' && event.answer?.questionId)
      .map((event) => event.answer!.questionId),
  )
  return events
    .filter(
      (event) =>
        event.kind === 'question' &&
        event.question?.questionId &&
        !answered.has(event.question.questionId),
    )
    .map((event) => ({ question: event.question!, text: event.text || '' }))
}

function hasOpenGate(events: PantheonEvent[]): boolean {
  let open = false
  for (const event of events) {
    if (event.kind !== 'gate' || !event.gate) continue
    open = event.gate.action === 'open'
  }
  return open
}

function WaitingPanel({
  session,
  busy,
  onAnswer,
  onResolveGate,
  onResume,
}: {
  session: PantheonSession
  busy: boolean
  onAnswer: (questionId: string, body: string) => void
  onResolveGate: (action: 'resolve' | 'dismiss', reason: string) => void
  onResume: () => void
}): React.ReactElement | null {
  const events = session.events ?? []
  const questions = pendingQuestionItems(events)
  const gateOpen = hasOpenGate(events)
  const [drafts, setDrafts] = useState<Record<string, string>>({})
  const [reason, setReason] = useState('')
  if (!gateOpen && questions.length === 0) {
    // 待办清空后的收尾面板（2026-11-08 实弹 pan_1794098356822）：席位提问
    // 答完、门也放行之后，整场仍停在 waiting_input——此前的 UI 是三无死路
    // （等待面板消失、恢复钮只在失败告示里、续议按钮依赖尚未生成的汇总），
    // 用户没有任何出路。后端 Resume 本就接受 waiting_input：末轮答完后续跑
    // 会跳过轮循环直接进汇总。
    if (session.status !== 'waiting_input') return null
    return (
      <section
        className="pantheon-waiting"
        aria-label="待办已清，可以继续"
        data-testid="pantheon-waiting-clear"
      >
        <header>
          <CheckCircleOutlined aria-hidden />
          <strong>待办已清，讨论可以继续</strong>
          <small>
            提问已全部回答、没有待处理的门。点「继续讨论」从下一轮续跑；若轮次已用完，续跑会直接进入协作汇总。
          </small>
        </header>
        <div className="pantheon-waiting-item">
          <Button
            type="primary"
            icon={<PlayCircleOutlined />}
            onClick={onResume}
            disabled={busy}
            data-testid="pantheon-resume-clear"
          >
            继续讨论
          </Button>
        </div>
      </section>
    )
  }
  const nameOf = (agentId?: string) =>
    session.participants.find((participant) => participant.agentId === agentId)?.name || '某一席'
  return (
    <section
      className="pantheon-waiting"
      aria-label="等待你处理"
      data-testid="pantheon-waiting"
    >
      <header>
        <ClockCircleOutlined aria-hidden />
        <strong>{gateOpen ? '已停在决策门' : '有席位在等你回答'}</strong>
        <small>
          {gateOpen
            ? '放行后从下一轮继续；放弃则用已有发言收尾，理由必须写。'
            : '回答只解除阻塞；续跑要点“继续讨论”，不会自动发生。'}
        </small>
      </header>
      {questions.map(({ question, text }) => {
        const draft = drafts[question.questionId] || ''
        const asker = question.origin === 'chair' ? '主席' : nameOf(question.askedBy)
        const options = question.options ?? []
        const showDefault = Boolean(question.default) && !options.includes(question.default!)
        return (
          <div className="pantheon-waiting-item" key={question.questionId}>
            <span className="pantheon-waiting-asker">{asker} 问</span>
            <p>{text}</p>
            {(options.length > 0 || showDefault) && (
              <div className="pantheon-waiting-options">
                {options.map((option) => (
                  <Button
                    key={option}
                    size="small"
                    disabled={busy}
                    onClick={() => onAnswer(question.questionId, option)}
                  >
                    {option}
                  </Button>
                ))}
                {showDefault && (
                  <Button
                    size="small"
                    disabled={busy}
                    onClick={() => onAnswer(question.questionId, question.default!)}
                  >
                    用默认：{question.default}
                  </Button>
                )}
              </div>
            )}
            <div className="pantheon-waiting-reply">
              <Input
                size="small"
                value={draft}
                placeholder="写一句回答，只投递给提问的那一席"
                aria-label={`回答 ${asker} 的提问`}
                onChange={(event) =>
                  setDrafts((current) => ({ ...current, [question.questionId]: event.target.value }))
                }
              />
              <Button
                size="small"
                type="primary"
                disabled={busy || !draft.trim()}
                onClick={() => onAnswer(question.questionId, draft.trim())}
              >
                回答
              </Button>
            </div>
          </div>
        )
      })}
      {gateOpen && (
        <div className="pantheon-waiting-gate">
          <Input
            size="small"
            value={reason}
            placeholder="放弃时要写理由（放行可留空）"
            aria-label="决策门理由"
            onChange={(event) => setReason(event.target.value)}
          />
          <Button size="small" type="primary" disabled={busy} onClick={() => onResolveGate('resolve', reason.trim())}>
            放行
          </Button>
          <Button size="small" danger disabled={busy || !reason.trim()} onClick={() => onResolveGate('dismiss', reason.trim())}>
            放弃后续轮次
          </Button>
        </div>
      )}
    </section>
  )
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
  if (event.kind === 'did_used') return '发现新的核验动作'
  if (event.kind === 'did_stop') return '停止空转'
  if (event.kind === 'attestation') return '独立确认'
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

const ISSUE_STATUS_META: Record<string, {
  label: string
  note: string
  icon: React.ReactNode
}> = {
  open: {
    label: '待收口',
    note: '等待提出者在后续轮次回应',
    icon: <ClockCircleOutlined aria-hidden />,
  },
  resolved: {
    label: '已撤回',
    note: '提出者已撤回该分歧',
    icon: <CheckCircleFilled aria-hidden />,
  },
  dissent: {
    label: '保留异议',
    note: '结案时仍未撤回，已随纪要留档',
    icon: <WarningOutlined aria-hidden />,
  },
}

const ISSUE_REVIEW_META: Record<string, { label: string; icon: React.ReactNode }> = {
  resolved: { label: '建议撤回', icon: <CheckCircleFilled aria-hidden /> },
  retain: { label: '建议保留', icon: <WarningOutlined aria-hidden /> },
  user_decision: { label: '交用户裁决', icon: <InfoCircleOutlined aria-hidden /> },
  needs_verification: { label: '先补核验', icon: <ClockCircleOutlined aria-hidden /> },
}

const IssueLedger: React.FC<{
  issues?: PantheonIssue[]
  reviews?: PantheonIssueReview[]
}> = ({ issues, reviews }) => {
  const [expanded, setExpanded] = useState(false)
  const issueCount = issues?.length || 0
  useEffect(() => {
    // 讨论运行中首次出现台账时自动展开；用户随后主动收起时，不因普通重渲染反复弹开。
    if (issueCount > 0) setExpanded(true)
  }, [issueCount])
  if (!issues?.length) return null
  const reviewByIssue = new Map((reviews || []).map((review) => [review.issueId, review]))
  const openCount = issues.filter((issue) => issue.status === 'open').length
  const resolvedCount = issues.filter((issue) => issue.status === 'resolved').length
  const dissentCount = issues.filter((issue) => issue.status === 'dissent').length
  const incompleteCount = issues.filter((issue) => (
    !issue.evidence?.trim()
    || !issue.flipCondition?.trim()
    || !issue.verification?.trim()
    || (issue.status === 'resolved' && !issue.resolutionEvidence?.trim())
  )).length
  const headline = openCount > 0
    ? `${openCount} 项分歧仍待收口`
    : dissentCount > 0
      ? `${dissentCount} 项异议已保留`
      : `${resolvedCount} 项分歧均已撤回`

  return (
    <details
      className="pantheon-issue-ledger"
      data-testid="pantheon-issue-ledger"
      open={expanded}
      onToggle={(event) => setExpanded(event.currentTarget.open)}
    >
      <summary>
        <span className="pantheon-issue-ledger-mark" aria-hidden>
          <TeamOutlined />
        </span>
        <span className="pantheon-issue-ledger-title">
          <small>分歧台账 · Issue ledger</small>
          <strong>{headline}</strong>
        </span>
        <span className="pantheon-issue-ledger-counts" aria-label="分歧状态统计">
          {openCount > 0 && <span className="is-open">{openCount} 待收口</span>}
          {dissentCount > 0 && <span className="is-dissent">{dissentCount} 保留</span>}
          {resolvedCount > 0 && <span className="is-resolved">{resolvedCount} 已撤回</span>}
          {incompleteCount > 0 && <span className="is-incomplete">{incompleteCount} 待补证据</span>}
        </span>
        <DownOutlined className="pantheon-issue-ledger-chevron" aria-hidden />
      </summary>
      <div className="pantheon-issue-ledger-body">
        <p>按提出者记录，不因汇总省略而消失；证据、翻转条件与核验路径逐条保留。</p>
        <ol>
          {issues.map((issue, index) => {
            const status = ISSUE_STATUS_META[issue.status] || {
              label: issue.status,
              note: '状态未归类',
              icon: <ClockCircleOutlined aria-hidden />,
            }
            const fields = [
              ...(issue.origin === 'challenge'
                || (!issue.origin && (issue.targetContributionId || issue.challengedClaim))
                ? [
                  { label: '目标贡献', value: issue.targetContributionId },
                  { label: '被挑战命题', value: issue.challengedClaim },
                ]
                : []),
              { label: '分歧依据', value: issue.evidence },
              ...(issue.responseVerdict
                ? [{ label: '回应结论', value: RESPONSE_VERDICT_LABEL[issue.responseVerdict] || issue.responseVerdict }]
                : []),
              ...(issue.responseEvidence
                ? [{ label: '回应证据', value: issue.responseEvidence }]
                : []),
              { label: '翻转条件', value: issue.flipCondition },
              { label: '下一步核验', value: issue.verification },
              ...(issue.status === 'resolved'
                ? [{ label: '撤回依据', value: issue.resolutionEvidence }]
                : []),
            ]
            const review = reviewByIssue.get(issue.id)
            const reviewMeta = review
              ? ISSUE_REVIEW_META[review.outcome] || {
                label: review.outcome,
                icon: <InfoCircleOutlined aria-hidden />,
              }
              : undefined
            return (
              <li
                key={issue.id}
                className={`is-${issue.status}`}
                data-testid={`pantheon-issue-${issue.id}`}
                title={`台账编号：${issue.id}`}
              >
                <span className="pantheon-issue-index" aria-hidden>
                  {String(index + 1).padStart(2, '0')}
                </span>
                <div className="pantheon-issue-copy">
                  <div className="pantheon-issue-head">
                    <strong>{issue.summary}</strong>
                    <span className={`pantheon-issue-status is-${issue.status}`}>
                      {status.icon}
                      {status.label}
                    </span>
                  </div>
                  <p>
                    <span>
                      第 {issue.round} 轮 · 异议方：{issue.agentName}
                      {(issue.lastSeenIn || 0) > issue.round
                        ? ` · 第 ${issue.lastSeenIn} 轮再次保留`
                        : ''}
                    </span>
                    <span>
                      主张方：{issue.claimAgentNames?.length
                        ? issue.claimAgentNames.join('、')
                        : '未定位'}
                    </span>
                    <span>来源：{ISSUE_ORIGIN_LABEL[issue.origin || ''] || '未标注'}</span>
                    <span>去向：{issue.disposition || status.note}</span>
                  </p>
                  <dl className="pantheon-issue-fields" aria-label={`${issue.agentName} 提出的分歧审计字段`}>
                    {fields.map((field) => {
                      const value = field.value?.trim()
                      return (
                        <div
                          key={field.label}
                          className={value ? undefined : 'is-missing'}
                        >
                          <dt>{field.label}</dt>
                          <dd>{value || '未提供，下一轮需补齐'}</dd>
                        </div>
                      )
                    })}
                  </dl>
                  {review && reviewMeta && (
                    <div className={`pantheon-issue-review is-${review.outcome}`}>
                      <div>
                        <strong>
                          {reviewMeta.icon}
                          主席审阅 · {reviewMeta.label}
                        </strong>
                        <small>仅作整理建议，不替代提出者撤回</small>
                      </div>
                      <p>{review.reason}</p>
                      {review.evidence && <small>依据：{review.evidence}</small>}
                    </div>
                  )}
                </div>
              </li>
            )
          })}
        </ol>
      </div>
    </details>
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

const SharedTimelineEntry: React.FC<{
  entry: SharedEntry
  speakingAgentIds: Set<string>
  foldedMap: Record<string, boolean>
  onFold: (id: string, folded: boolean) => void
}> = ({ entry, speakingAgentIds, foldedMap, onFold }) => (
  <article
    className={`pantheon-shared-entry is-${entry.kind}${entry.streaming ? ' is-streaming' : ''}${entry.kind === 'agent' && entry.agentId && speakingAgentIds.has(entry.agentId) ? ' is-seat-speaking' : ''}`}
    data-testid={`pantheon-shared-entry-${entry.id}`}
    data-agent-id={entry.agentId}
    style={entry.agentId ? {
      '--agent-accent': AGENT_ACCENTS[entry.agentId] || 'var(--pantheon-accent)',
    } as React.CSSProperties : undefined}
  >
    <span className="pantheon-shared-avatar" aria-hidden>
      {entry.kind === 'user' ? (
        '你'
      ) : entry.kind === 'working' && entry.agentId ? (
        <PantheonAgentAvatar
          agentId={entry.agentId}
          className="pantheon-shared-agent-avatar"
          status="running"
        />
      ) : entry.agentId ? (
        <PantheonAgentAvatar
          agentId={entry.agentId}
          className="pantheon-shared-agent-avatar"
          status={entry.streaming || (entry.kind === 'agent' && entry.agentId && speakingAgentIds.has(entry.agentId)) ? 'running' : 'finished'}
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
        <DiscussionActBadge
          act={entry.discussionAct}
          testId={`pantheon-discussion-act-${entry.id}`}
        />
        {(entry.kind === 'live' || entry.streaming || entry.kind === 'working') && (
          <span className="pantheon-streaming-state">
            {entry.streaming ? '正在发言' : entry.kind === 'working' ? '正在工作' : '正在发生'}
          </span>
        )}
        {entry.kind === 'agent' && entry.durationMs ? (
          <span className="pantheon-shared-entry-duration">
            用时 {(entry.durationMs / 1000).toFixed(1)}s
          </span>
        ) : null}
        <time>{entry.at === Number.MAX_SAFE_INTEGER ? '现在' : formatClock(entry.at)}</time>
      </header>
      {entry.kind === 'agent' ? (
        <SharedAgentBody
          entry={entry}
          foldedMap={foldedMap}
          onFold={onFold}
        />
      ) : (
        <p>{entry.text}</p>
      )}
      {entry.kind === 'agent' && entry.mismatchNote && (
        <small className="pantheon-shared-mismatch" role="note" data-testid={`pantheon-mismatch-${entry.id}`}>
          {entry.mismatchNote}
        </small>
      )}
      {entry.kind === 'agent' && (
        <DiscussionActDetail
          targetContributionId={entry.targetContributionId}
          targetIssueId={entry.targetIssueId}
          challengedClaim={entry.challengedClaim}
          responseVerdict={entry.responseVerdict}
        />
      )}
      {entry.meta && entry.kind !== 'agent' && <small>{entry.meta}</small>}
    </div>
  </article>
)

const ATTESTATION_META: Record<string, { label: string; icon: React.ReactNode }> = {
  support: { label: '支持', icon: <CheckCircleFilled aria-hidden /> },
  conditional: { label: '有条件支持', icon: <ClockCircleOutlined aria-hidden /> },
  oppose: { label: '反对', icon: <CloseCircleFilled aria-hidden /> },
  unknown: { label: '未识别', icon: <WarningOutlined aria-hidden /> },
}

const AttestationPanel: React.FC<{ session: PantheonSession }> = ({ session }) => {
  if (!session.attestations?.length) return null
  const support = session.attestations.filter((item) => item.position === 'support').length
  const conditional = session.attestations.filter((item) => item.position === 'conditional').length
  const oppose = session.attestations.filter((item) => item.position === 'oppose').length
  return (
    <section className="pantheon-attestations" data-testid="pantheon-attestations">
      <header>
        <div>
          <span className="pantheon-eyebrow">独立确认 · Attestation</span>
          <strong>
            {support} 支持 · {conditional} 有条件 · {oppose} 反对
          </strong>
        </div>
        <Tag>{session.verdict?.revisedFromSha256 ? '已修订一次' : '原稿待签注'}</Tag>
      </header>
      {session.verdict?.revisionReason && (
        <p className="pantheon-attestation-revision">
          {session.verdict.revisionReason}
        </p>
      )}
      <ol>
        {session.attestations.map((attestation) => {
          const meta = ATTESTATION_META[attestation.position] || ATTESTATION_META.unknown
          return (
            <li key={attestation.agentId} className={`is-${attestation.position}`}>
              <span>{meta.icon}{meta.label}</span>
              <div>
                <strong>{attestation.agentName}</strong>
                <p>{attestation.reason || attestation.error || '未提供理由'}</p>
                {attestation.conditions?.length ? (
                  <small>条件：{attestation.conditions.join('；')}</small>
                ) : null}
                {attestation.issueIds?.length ? (
                  <small>相关分歧：{attestation.issueIds.join('、')}</small>
                ) : null}
              </div>
            </li>
          )
        })}
      </ol>
    </section>
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

      {session.verdict.proposedActions?.length ? (
        <section className="pantheon-action-plan" data-testid="pantheon-proposed-actions">
          <header>
            <div>
              <span className="pantheon-eyebrow">行动提案 · 只读</span>
              <strong>{session.verdict.proposedActions.length} 项，签注后另行授权执行</strong>
            </div>
            <Tag>不会自动运行</Tag>
          </header>
          <ol>
            {session.verdict.proposedActions.map((action, index) => (
              <li key={action.id || `${action.action}-${index}`}>
                <span>{action.id || `action-${index + 1}`}</span>
                <div>
                  <strong>{action.action}</strong>
                  {action.rationale && <p>{action.rationale}</p>}
                  <dl>
                    {action.dependsOn?.length ? (
                      <div><dt>依赖</dt><dd>{action.dependsOn.join('、')}</dd></div>
                    ) : null}
                    {action.acceptanceEvidence && (
                      <div><dt>验收证据</dt><dd>{action.acceptanceEvidence}</dd></div>
                    )}
                    {action.suggestedOwner && (
                      <div><dt>建议负责人</dt><dd>{action.suggestedOwner}</dd></div>
                    )}
                    {action.risk && <div><dt>风险</dt><dd>{action.risk}</dd></div>}
                  </dl>
                </div>
              </li>
            ))}
          </ol>
        </section>
      ) : null}

      <AttestationPanel session={session} />

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
  // 每轮输出折叠（2026-11-03 用户反馈「不要全部摊开」）：默认只展开最新一轮，
  // 流式中的恒展开；用户手动开合记录在 map，重渲染不弹回。
  const [openOutputs, setOpenOutputs] = useState<Record<string, boolean>>({})

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
              {outputs.map((output, index) => {
                const manuallyToggled = openOutputs[output.id]
                const expanded = output.streaming
                  ? true
                  : manuallyToggled !== undefined
                    ? manuallyToggled
                    : index === outputs.length - 1
                return (
                  <article
                    key={output.id}
                    className={`pantheon-agent-output${output.streaming ? ' is-streaming' : ''}`}
                    data-testid={`pantheon-agent-output-${output.id}`}
                  >
                    <details
                      className="pantheon-agent-output-fold"
                      open={expanded}
                      onToggle={(event) => {
                        if (output.streaming) return
                        const next = (event.currentTarget as HTMLDetailsElement).open
                        setOpenOutputs((current) => ({ ...current, [output.id]: next }))
                      }}
                    >
                      <summary className="pantheon-agent-output-summary">
                        <header>
                          <span>
                            {output.round ? `第 ${output.round} 轮 · ` : ''}
                            {PHASE_LABEL[output.phase] || output.phase}
                            {output.discussionAct && (
                              <>
                                {' · '}
                                <DiscussionActBadge
                                  act={output.discussionAct}
                                  testId={`pantheon-agent-discussion-act-${output.id}`}
                                />
                              </>
                            )}
                          </span>
                          <small title={[seatUsageTitle(output.usage), seatToolEvidenceTitle(output.toolEvidence), seatPromptTitle(output.promptSha256, output.promptRunes)].filter(Boolean).join('\n')}>
                            {output.streaming
                              ? '正在输出'
                              : [
                                  output.durationMs ? `${(output.durationMs / 1000).toFixed(1)}s` : '独立观点',
                                  seatUsageLabel(output.usage),
                                  seatToolEvidenceLabel(output.toolEvidence),
                                  seatMismatchLabel(output.toolEvidence, output.text),
                                ].filter(Boolean).join(' · ')}
                          </small>
                        </header>
                        <span className="pantheon-agent-output-chevron" aria-hidden />
                      </summary>
                      <div className={`pantheon-agent-output-markdown${output.streaming ? ' is-live' : ''}`}>
                        <MarkdownContent source={output.text} className="md-content" />
                        <DiscussionActDetail
                          targetContributionId={output.targetContributionId}
                          targetIssueId={output.targetIssueId}
                          challengedClaim={output.challengedClaim}
                          responseVerdict={output.responseVerdict}
                        />
                        {output.chairDecision && output.chairDecision.status !== 'continue' && (
                          <p className="pantheon-chair-decision">
                            主席终局：{chairDecisionLabel(output.chairDecision.status)}
                            {output.chairDecision.detail ? `——${output.chairDecision.detail}` : ''}
                          </p>
                        )}
                        {output.chairAssignments && output.chairAssignments.length > 0 && (
                          <div className="pantheon-chair-block">
                            <strong>主席派工</strong>
                            <ul>
                              {output.chairAssignments.map((assignment, index) => {
                                const receipt = output.receipts?.find((item) => item.agentId === assignment.agentId)
                                return (
                                  <li key={`${assignment.agentId}-${index}`}>
                                    <code>{receipt?.agentName || assignment.agentId}</code>
                                    <span>{assignment.instruction}</span>
                                    {receipt && <em title={receipt.detail}>{receiptStatusLabel(receipt.status)}</em>}
                                  </li>
                                )
                              })}
                            </ul>
                          </div>
                        )}
                        {output.streaming && (
                          <span className="pantheon-agent-output-caret pantheon-stream-caret" aria-label="正在输出" />
                        )}
                      </div>
                    </details>
                  </article>
                )
              })}
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

const AgentWindowWall: React.FC<{
  participants: PantheonParticipant[]
  outputsByAgent: Map<string, DiscussionOutput[]>
  tracesByAgent: Map<string, ToolTraceView[]>
  speakingAgentIds: Set<string>
  selectedAgentId: string
  clockNow: number
  onSelect: (agentId: string, trigger: HTMLElement) => void
}> = ({
  participants,
  outputsByAgent,
  tracesByAgent,
  speakingAgentIds,
  selectedAgentId,
  clockNow,
  onSelect,
}) => (
  <section className="pantheon-agent-windows" aria-labelledby="pantheon-agent-windows-title">
    <header className="pantheon-agent-windows-head">
      <div>
        <span className="pantheon-eyebrow">席位视图</span>
        <h3 id="pantheon-agent-windows-title">各席独立窗口</h3>
        <p>公开观点按席位归拢；推理与工具调用在展开详情后查看。</p>
      </div>
      <span>{participants.length} 席</span>
    </header>
    <div className="pantheon-agent-window-grid">
      {participants.map((participant, index) => {
        const outputs = outputsByAgent.get(participant.agentId) ?? []
        const traces = tracesByAgent.get(participant.agentId) ?? []
        const latest = outputs[outputs.length - 1]
        const active = speakingAgentIds.has(participant.agentId)
        const selected = selectedAgentId === participant.agentId
        return (
          <button
            key={participant.agentId}
            type="button"
            className={`pantheon-agent-window${active ? ' is-running' : ''}${selected ? ' is-selected' : ''}`}
            data-testid={`pantheon-agent-window-${participant.agentId}`}
            aria-expanded={selected}
            aria-controls="pantheon-agent-detail"
            aria-label={`${participant.name}，${seatStatusText(participant, clockNow)}，${outputs.length} 条观点，${traces.length} 次工具调用，打开独立窗口`}
            style={{
              '--agent-accent': AGENT_ACCENTS[participant.agentId] || 'var(--pantheon-accent)',
              '--window-order': index,
            } as React.CSSProperties}
            onClick={(event) => onSelect(participant.agentId, event.currentTarget)}
          >
            <span className="pantheon-agent-window-head">
              <PantheonAgentAvatar
                agentId={participant.agentId}
                className="pantheon-agent-window-avatar"
                status={active ? 'running' : avatarStatus(participant.status)}
              />
              <span>
                <strong>{participant.name}</strong>
                <small>
                  {active && <span className="pantheon-live-dot" aria-hidden />}
                  {seatStatusText(participant, clockNow)}
                </small>
              </span>
              <span className="pantheon-agent-window-open">
                {selected ? '已展开' : '打开'}
              </span>
            </span>
            <span className="pantheon-agent-window-body">
              {latest ? (
                <>
                  <small>
                    第 {Math.max(1, latest.round)} 轮
                    {latest.streaming ? ' · 正在输出' : ''}
                  </small>
                  <span>{windowPreviewText(latest.text)}</span>
                </>
              ) : (
                <>
                  <small>尚无公开观点</small>
                  <span>{active ? '正在处理本轮任务。' : '该席位还没有返回可展示的观点。'}</span>
                </>
              )}
            </span>
            <span className="pantheon-agent-window-foot">
              <span>观点 {outputs.length}</span>
              <span>工具 {traces.length}</span>
              <span>{agentCapabilityLabel(participant.agentId)}</span>
            </span>
          </button>
        )
      })}
    </div>
  </section>
)

export const PantheonRecord: React.FC<Props> = ({
  session,
  loading,
  busy,
  error,
  liveStreams,
  onCancel,
  onSkipAgent,
  onAddUserNote,
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
  onAnswer,
  onResolveGate,
}) => {
  const meta = statusMeta(session.status)
  // 中断与等待回答都不是"结束"：秒表要继续走，否则界面会显示成停滞。
  const running = session.status === 'running'
  const liveish = running || session.status === 'interrupted' || session.status === 'waiting_input'
  const clockNow = useSecondClock(liveish)
  // 席位级活性单看拓扑节点就够，但整场"有几席其实没证据"得有个总数：读者不该
  // 靠逐个点开才发现五席里有三席在硬撑（M2）。
  const staleSeats = useMemo(
    () => (session.participants ?? []).filter((participant) => seatLivenessNote(participant, clockNow) !== ''),
    [session.participants, clockNow],
  )
  const streamRef = useRef<HTMLDivElement>(null)
  const detailRef = useRef<HTMLElement>(null)
  const detailTriggerRef = useRef<HTMLElement | null>(null)
  const viewTabsRef = useRef<HTMLElement>(null)
  const conclusionRef = useRef<HTMLDivElement>(null)
  // 用户中途评论（2026-11-03）：running/waiting_input 皆可发言，不打断当前席位。
  const [noteDraft, setNoteDraft] = useState('')
  const [noteSending, setNoteSending] = useState(false)
  const [selectedAgentId, setSelectedAgentId] = useState('')
  const [detailTab, setDetailTab] = useState<'output' | 'trace'>('output')
  const [metaOpen, setMetaOpen] = useState(false)
  const [conclusionPreviewOpen, setConclusionPreviewOpen] = useState(false)
  const [recordView, setRecordView] = useState<RecordView>(() =>
    session.status === 'running' || session.status === 'waiting_input' ? 'topology' : 'shared',
  )
  const [foldedEntries, setFoldedEntries] = useState<Record<string, boolean>>({})
  const [roundOpenState, setRoundOpenState] = useState<Record<string, boolean>>({})
  const toggleFolded = useCallback((id: string, folded: boolean) => {
    setFoldedEntries((current) => ({ ...current, [id]: folded }))
  }, [])
  const setRoundOpen = useCallback((key: string, open: boolean) => {
    setRoundOpenState((current) => (
      current[key] === open ? current : { ...current, [key]: open }
    ))
  }, [])
  const switchRecordView = useCallback((view: RecordView) => {
    setRecordView(view)
    setSelectedAgentId('')
    window.requestAnimationFrame(() => {
      const reduce = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
      viewTabsRef.current?.scrollIntoView({
        block: 'start',
        behavior: reduce ? 'auto' : 'smooth',
      })
    })
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

  const chairReceipts = useMemo<Record<string, PantheonChairReceipt[]>>(() => {
    const map: Record<string, PantheonChairReceipt[]> = {}
    for (const item of session.contributions ?? []) {
      if (item.phase === 'moderation' && (item.chairAssignments?.length ?? 0) > 0) {
        map[item.id] = chairAssignmentReceipts(session, item)
      }
    }
    return map
  }, [session])

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
        discussionAct: item.discussionAct,
        targetContributionId: item.targetContributionId,
        targetIssueId: item.targetIssueId,
        challengedClaim: item.challengedClaim,
        responseVerdict: item.responseVerdict,
        usage: item.usage,
        chairAssignments: item.chairAssignments,
        chairDecision: item.chairDecision,
        toolEvidence: item.toolEvidence,
        receipts: item.phase === 'moderation' ? chairReceipts[item.id] : undefined,
        promptSha256: item.promptSha256,
        promptRunes: item.promptRunes,
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
        promptSha256: undefined,
        promptRunes: undefined,
      }))
  }, [participants, session.contributions, session.events, chairReceipts])

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
    // 一张卡=一次逻辑工具调用。CLI 的结果事件可能带不同 ID（zcode 实弹
    // 2026-10-31：started 用 callId，结果却发两份——同 ID 无名回执+带
    // tool_subagent_* 新 ID 的命名回执），按 ID 严格合并会把每条工具卡
    // 渲染两遍。同席同名 FIFO 配对兜底；无名回执不得抹掉真实工具名。
    type traceCard = ToolTraceView & { namedDone: boolean }
    const cards: traceCard[] = []
    const byID = new Map<string, traceCard>()
    const anonymous = (name: string) => !name || name === '工具'
    for (const event of session.events) {
      if (event.kind !== 'tool_started' && event.kind !== 'tool_result') continue
      const agentId = event.agentId || ''
      const name = event.toolName || ''
      const isResult = event.kind === 'tool_result'
      let card = (event.toolId ? byID.get(event.toolId) : undefined) ?? undefined
      if (!card && isResult && event.toolId) {
        if (!anonymous(name)) {
          card = cards.find((item) => item.agentId === agentId
            && !item.namedDone
            && (item.toolName === name || anonymous(item.toolName)))
        }
        if (!card && anonymous(name)) {
          // 无名且配不上任何 started 的回执是纯噪音，丢弃
          continue
        }
      }
      if (!card) {
        card = {
          id: event.toolId || `tool-event-${event.seq}`,
          agentId,
          agentName:
            participants.find((participant) => participant.agentId === agentId)?.name
            || agentId
            || 'Agent',
          toolName: name || '工具',
          status: event.toolStatus || (isResult ? 'completed' : 'started'),
          input: event.toolInput || '',
          output: event.toolOutput || '',
          error: event.error || '',
          at: event.at,
          namedDone: false,
        }
        cards.push(card)
        if (event.toolId) byID.set(event.toolId, card)
        continue
      }
      if (!anonymous(name)) {
        card.toolName = name
        card.namedDone = true
      }
      card.status = event.toolStatus || card.status
      card.input = card.input || event.toolInput || ''
      card.output = card.output || event.toolOutput || ''
      card.error = card.error || event.error || ''
      card.at = event.at
    }
    return cards.slice(-80)
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
  // 发言中席位集合：共享流里该席卡片持续挂专属动画，直到收声（2026-11-03
  // 用户反馈「动画只在流式瞬间闪一下看不见」——定稿期也要亮）。
  const speakingAgentIds = useMemo(
    () => new Set(activeSpeakers.map((speaker) => speaker.agentId)),
    [activeSpeakers],
  )
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
    (session.status === 'failed' || session.status === 'cancelled' || session.status === 'interrupted')
  const showFailureRecovery = !stalled && (runningFailure || terminalRecoverable)

  const completedCount = participants.filter((participant) => seatReturnsResult(participant.status)).length
  const runningSeatCount = participants.filter(
    (participant) => speakingAgentIds.has(participant.agentId)
      || participant.status === 'running'
      || participant.status === 'retrying',
  ).length
  const waitingSeatCount = participants.filter(
    (participant) => !speakingAgentIds.has(participant.agentId)
      && (participant.status === 'waiting' || participant.status === 'needs_input'),
  ).length
  const failedSeatCount = participants.filter((participant) => participant.status === 'failed').length
  const openIssueCount = (session.issues || []).filter((issue) => issue.status === 'open').length
  const publicOutputCount = displayOutputs.filter(
    (output) => output.phase !== 'user_note' && output.agentId !== 'user',
  ).length
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
  const activeStartTimes = activeSpeakers
    .map((participant) => agentStartedAt(participant))
    .filter((startedAt) => startedAt > 0)
  const activeStartedAt = activeStartTimes.length ? Math.min(...activeStartTimes) : 0
  const activeNames = activeSpeakers.map((participant) => participant.name)
  const phaseLabel = PHASE_LABEL[session.phase] || session.phase
  const activityHeadline = session.status === 'waiting_input'
    ? '等待你回应后继续'
    : activeNames.length
      ? `${joinNames(activeNames)} 正在${phaseLabel}`
      : running
        ? `正在${phaseLabel}，等待下一席位`
        : session.status === 'completed'
          ? '讨论已归档'
          : meta.label
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

  const submitUserNote = useCallback(() => {
    const content = noteDraft.trim()
    if (!content || noteSending) return
    setNoteSending(true)
    try {
      onAddUserNote(content)
      setNoteDraft('')
    } finally {
      setNoteSending(false)
    }
  }, [noteDraft, noteSending, onAddUserNote])

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
      if (![
        'phase', 'consensus', 'did_used', 'did_stop',
        'attestation',
        'warning', 'error', 'agent_skipped', 'agent_retry',
      ].includes(event.kind)) continue
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
        round: event.round,
        phaseLabel: PHASE_LABEL[event.phase] || event.phase,
      })
    }

    for (const output of displayOutputs) {
      // 用户中途评论（2026-11-03）：独立卡片形态，标注「你」，不被折叠逻辑当 agent 发言处理。
      if (output.phase === 'user_note' || output.agentId === 'user') {
        entries.push({
          id: `shared-output-${output.id}`,
          kind: 'user',
          label: '你（插话）',
          text: output.text,
          meta: output.round ? `第 ${output.round} 轮 · 全员可见，后续发言优先回应` : '全员可见，后续发言优先回应',
          at: output.at || session.updatedAt,
          round: output.round,
          phaseLabel: '用户插话',
          agentId: output.agentId,
        })
        continue
      }
      entries.push({
        id: `shared-output-${output.id}`,
        kind: 'agent',
        label: output.agentName,
        text: output.text,
        meta: [
          output.round ? `第 ${output.round} 轮` : '',
          PHASE_LABEL[output.phase] || output.phase,
          output.durationMs ? `用时 ${(output.durationMs / 1000).toFixed(1)}s` : '',
          seatUsageLabel(output.usage),
          seatToolEvidenceLabel(output.toolEvidence),
        ].filter(Boolean).join(' · '),
        mismatchNote: seatMismatchLabel(output.toolEvidence, output.text) || undefined,
        at: output.streaming ? Number.MAX_SAFE_INTEGER : output.at || session.updatedAt,
        round: output.round,
        phaseLabel: PHASE_LABEL[output.phase] || output.phase,
        durationMs: output.durationMs,
        agentId: output.agentId,
        streaming: output.streaming,
        discussionAct: output.discussionAct,
        targetContributionId: output.targetContributionId,
        targetIssueId: output.targetIssueId,
        challengedClaim: output.challengedClaim,
        responseVerdict: output.responseVerdict,
      })
    }

    entries.sort((a, b) => a.at - b.at)

    // 工具期工作条（2026-11-03 用户反馈「是否工作不是只看正文,工具调用也是在工作」）：
    // running 且当前没有正文直播条目的席，补一张工作卡显示最新工具——纯工具期不再隐形。
    const streamingAgents = new Set(
      displayOutputs.filter((output) => output.streaming).map((output) => output.agentId),
    )
    for (const speaker of activeSpeakers) {
      if (streamingAgents.has(speaker.agentId)) continue
      const traces = tracesByAgent.get(speaker.agentId) ?? []
      const latest = traces[traces.length - 1]
      entries.push({
        id: `shared-working-${speaker.agentId}`,
        kind: 'working',
        label: speaker.name,
        text: latest
          ? `正在调用工具：${latest.toolName}（本轮已 ${traces.length} 次调用）`
          : `正在${PHASE_LABEL[session.phase] || '处理'}，暂无工具调用记录`,
        meta: `第 ${session.currentRound || 1} 轮 · 工具调用也是在工作`,
        at: Number.MAX_SAFE_INTEGER - 1,
        round: session.currentRound || 1,
        phaseLabel: PHASE_LABEL[session.phase] || session.phase,
        agentId: speaker.agentId,
      })
    }

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
    session.currentRound,
    tracesByAgent,
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

  const latestSharedUpdate = useMemo(() => {
    const ordered = [...sharedEntries].reverse()
    return ordered.find((entry) => entry.kind === 'agent' || entry.kind === 'user')
      || ordered.find((entry) => entry.kind === 'event' || entry.kind === 'warning')
  }, [sharedEntries])

  const sharedTimelineSections = useMemo<SharedTimelineSection[]>(() => {
    const counts = new Map<string, number>()
    for (const entry of sharedEntries) {
      if (entry.kind !== 'agent' || !entry.phaseLabel) continue
      const key = `${entry.round || 'phase'}:${entry.phaseLabel}`
      counts.set(key, (counts.get(key) || 0) + 1)
    }
    const sections: SharedTimelineSection[] = [{
      key: 'shared-prelude',
      entries: [],
    }]
    let currentSection = sections[0]
    let currentGroup = ''
    for (const entry of sharedEntries) {
      const groupKey = entry.kind === 'agent' && entry.phaseLabel
        ? `${entry.round || 'phase'}:${entry.phaseLabel}`
        : ''
      if (groupKey && groupKey !== currentGroup) {
        currentGroup = groupKey
        currentSection = {
          key: `${groupKey}:${sections.length}`,
          heading: {
            round: entry.round,
            phase: entry.phaseLabel || '讨论',
            count: counts.get(groupKey) || 1,
          },
          entries: [],
        }
        sections.push(currentSection)
      }
      currentSection.entries.push(entry)
    }
    return sections.filter((section) => section.entries.length > 0)
  }, [sharedEntries])
  const latestTimelineSectionKey = [...sharedTimelineSections]
    .reverse()
    .find((section) => section.heading)?.key || ''

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
              <span className="pantheon-eyebrow">议事纪要</span>
              <span>{PHASE_LABEL[session.phase] || session.phase}</span>
              {session.request.workspace && (
                <span className="pantheon-record-workspace" title={session.request.workspace}>
                  <FolderOpenOutlined />
                  {workspaceName(session.request.workspace)}
                </span>
              )}
              {staleSeats.length > 0 && (
                <Tooltip
                  title={`${staleSeats.map((participant) => participant.name).join('、')} 最近 ${Math.round(SEAT_STALE_AFTER_MS / 1000)} 秒没有新证据，界面不替它们保证在干活`}
                >
                  <span className="pantheon-record-stale" data-testid="pantheon-stale-seats">
                    {staleSeats.length} 席活性未知
                  </span>
                </Tooltip>
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
                aria-expanded={metaOpen && recordView === 'ledger'}
                aria-controls="pantheon-meta"
                data-testid="pantheon-meta-toggle"
                onClick={() => {
                  if (!metaOpen || recordView !== 'ledger') {
                    setMetaOpen(true)
                    switchRecordView('ledger')
                    return
                  }
                  setMetaOpen(false)
                }}
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

        <WaitingPanel
          session={session}
          busy={busy}
          onAnswer={onAnswer}
          onResolveGate={onResolveGate}
          onResume={onResume}
        />

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

        <nav ref={viewTabsRef} className="pantheon-record-tabs" aria-label="纪要视图">
          <button
            type="button"
            className={recordView === 'topology' ? 'is-active' : ''}
            aria-pressed={recordView === 'topology'}
            aria-controls="pantheon-panel-topology"
            data-testid="pantheon-view-topology"
            onClick={() => switchRecordView('topology')}
          >
            <NodeIndexOutlined aria-hidden />
            <span>拓扑图</span>
            <small>{participants.length} 席</small>
          </button>
          <button
            type="button"
            className={recordView === 'shared' ? 'is-active' : ''}
            aria-pressed={recordView === 'shared'}
            aria-controls="pantheon-panel-shared"
            data-testid="pantheon-view-shared"
            onClick={() => switchRecordView('shared')}
          >
            <CommentOutlined aria-hidden />
            <span>共享窗口</span>
            <small>{sharedEntries.length} 条</small>
          </button>
          <button
            type="button"
            className={recordView === 'agents' ? 'is-active' : ''}
            aria-pressed={recordView === 'agents'}
            aria-controls="pantheon-panel-agents"
            data-testid="pantheon-view-agents"
            onClick={() => switchRecordView('agents')}
          >
            <TeamOutlined aria-hidden />
            <span>Agent 窗口</span>
            <small>{participants.length} 席</small>
          </button>
          <button
            type="button"
            className={recordView === 'ledger' ? 'is-active' : ''}
            aria-pressed={recordView === 'ledger'}
            aria-controls="pantheon-panel-ledger"
            data-testid="pantheon-view-ledger"
            onClick={() => switchRecordView('ledger')}
          >
            <FileTextOutlined aria-hidden />
            <span>分歧台账</span>
            <small>{session.issues?.length || 0} 条</small>
          </button>
        </nav>

        <div
          id="pantheon-panel-ledger"
          className="pantheon-ledger-view"
          hidden={recordView !== 'ledger'}
        >
          <IssueLedger issues={session.issues} reviews={session.verdict?.issueReviews} />

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
                  <div>
                    <dt>议事形态</dt>
                    <dd title={session.request.preset === 'targeted_challenge'
                      ? '围绕分歧台账逐条质疑、回应与核验'
                      : session.request.preset === 'red_team'
                        ? '围绕最强反例与失败模式讨论，结论生成后全席独立确认'
                        : '独立开场后共享讨论并协作综合'}
                    >
                      {presetLabel(session.request.preset)}
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
        </div>

        <section
          id="pantheon-panel-topology"
          className="pantheon-team-map"
          aria-labelledby="pantheon-team-map-title"
          hidden={recordView !== 'topology'}
        >
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

            <button
              type="button"
              className="pantheon-topology-hub"
              aria-label={[
                '打开共享讨论窗口',
                activityHeadline,
                `${completedCount} / ${participants.length} 位已返回`,
                `${runningSeatCount} 位处理中`,
                waitingSeatCount > 0 ? `${waitingSeatCount} 位候场` : '',
                failedSeatCount > 0 ? `${failedSeatCount} 位失败` : '',
                `${publicOutputCount} 条公开观点`,
                `${toolTraces.length} 次工具调用`,
                openIssueCount > 0 ? `${openIssueCount} 项分歧待收口` : '',
              ].filter(Boolean).join('，')}
              aria-controls="pantheon-panel-shared"
              data-testid="pantheon-open-shared"
              onClick={() => switchRecordView('shared')}
            >
              <span className="pantheon-topology-hub-icon" aria-hidden>
                <TeamOutlined />
              </span>
              <span className="pantheon-topology-hub-content">
                <span className="pantheon-topology-hub-title">
                  <strong>共享讨论</strong>
                  <span className={`pantheon-topology-hub-state is-${session.status}`}>
                    {running && <span className="pantheon-live-dot" aria-hidden />}
                    {meta.label}
                  </span>
                </span>
                <span className={`pantheon-topology-hub-current${activeNames.length ? ' is-live' : ''}`}>
                  {activeNames.length ? `当前：${activityHeadline}` : activityHeadline}
                  {activeSpeakers.length === 1 && activeStartedAt > 0 && (
                    <span className="pantheon-topology-hub-elapsed">
                      工作 {formatElapsed(clockNow - activeStartedAt)}
                    </span>
                  )}
                </span>
                <span
                  className="pantheon-topology-hub-latest"
                  data-testid="pantheon-shared-card-latest"
                >
                  <small>最新</small>
                  <span>
                    {latestSharedUpdate
                      ? `${latestSharedUpdate.label} · ${previewText(latestSharedUpdate.text)}`
                      : '尚无共享发言；观点会按返回顺序进入主线程。'}
                  </span>
                </span>
                <span className="pantheon-topology-hub-counts">
                  <span><b>{completedCount}/{participants.length}</b> 已返回</span>
                  {runningSeatCount > 0 && <span><b>{runningSeatCount}</b> 处理中</span>}
                  {waitingSeatCount > 0 && <span><b>{waitingSeatCount}</b> 候场</span>}
                  {failedSeatCount > 0 && <span className="is-warning"><b>{failedSeatCount}</b> 失败</span>}
                  <span><b>{publicOutputCount}</b> 观点</span>
                  <span><b>{toolTraces.length}</b> 工具</span>
                  {openIssueCount > 0 && (
                    <span className="is-warning"><b>{openIssueCount}</b> 待收口分歧</span>
                  )}
                </span>
                <span className="pantheon-topology-hub-foot">
                  <span>第 {activeRound} / {totalRounds} 轮 · {phaseLabel}</span>
                  <span>打开共享窗口 →</span>
                </span>
              </span>
            </button>

            <div className="pantheon-topology-nodes">
              {participants.map((participant, index) => {
                const active = activeSpeakers.some((speaker) => speaker.agentId === participant.agentId)
                const selected = selectedAgentId === participant.agentId
                const facilitator = session.request.chairmanAgentId === participant.agentId
                const outputs = outputsByAgent.get(participant.agentId) ?? []
                const traces = tracesByAgent.get(participant.agentId) ?? []
                const latestOutput = outputs[outputs.length - 1]
                const latestTrace = traces[traces.length - 1]
                const status = active ? 'running' : participant.status
                const startedAt = agentStartedAt(participant)
                const latestTarget = latestOutput?.targetContributionId
                  ? displayOutputs.find((output) => output.id === latestOutput.targetContributionId)
                  : undefined
                const latestIssue = latestOutput?.targetIssueId
                  ? session.issues?.find((issue) => issue.id === latestOutput.targetIssueId)
                  : undefined
                const latestTargetName = latestTarget?.agentName || latestIssue?.agentName
                const latestAt = Math.max(
                  latestOutput?.at || 0,
                  latestTrace?.at || 0,
                  participant.finishedAt || 0,
                  startedAt,
                )
                const preview = latestOutput
                  ? windowPreviewText(latestOutput.text)
                  : participant.status === 'failed'
                    ? participant.error || '本席调用失败，等待重试或跳过。'
                    : participant.status === 'skipped'
                      ? '本轮已跳过，保留此前观点。'
                      : active
                        ? `正在处理第 ${activeRound} 轮任务…`
                        : '尚无公开观点。'
                return (
                  <div
                    key={participant.agentId}
                    className="pantheon-topology-node-wrap"
                    data-agent-id={participant.agentId}
                    style={{
                      '--agent-accent': AGENT_ACCENTS[participant.agentId] || 'var(--pantheon-accent)',
                    } as React.CSSProperties}
                  >
                  <button
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
                      seatStatusText(participant, clockNow),
                      active && startedAt > 0
                        ? `已持续工作 ${formatElapsed(clockNow - startedAt)}`
                        : '',
                      `${outputs.length} 条观点`,
                      `${traces.length} 次工具调用`,
                      participant.version ? `版本 ${participant.version}` : '',
                      '点击查看独立输出',
                    ].filter(Boolean).join('，')}
                    title={`查看 ${participant.name} 的独立输出`}
                    onClick={(event) => selectAgent(participant.agentId, event.currentTarget)}
                    style={{
                      '--agent-accent': AGENT_ACCENTS[participant.agentId] || 'var(--pantheon-accent)',
                      '--node-order': index,
                    } as React.CSSProperties}
                  >
                    {active && (
                      <span className="pantheon-topology-current">
                        <span className="pantheon-live-dot" aria-hidden />
                        当前
                      </span>
                    )}
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
                        {seatStatusText(participant, clockNow)}
                        {facilitator
                          ? ' · 汇总'
                          : ` · ${agentCapabilityLabel(participant.agentId)}`}
                      </small>
                    </span>
                    <span className="pantheon-topology-preview">
                      <span className="pantheon-topology-preview-meta">
                        <small>
                          {latestOutput
                            ? `第 ${Math.max(1, latestOutput.round)} 轮 · ${formatClock(latestOutput.at)}`
                            : latestAt > 0
                              ? `最后活动 ${formatClock(latestAt)}`
                              : '等待活动'}
                        </small>
                        <DiscussionActBadge
                          act={latestOutput?.discussionAct}
                          testId={`pantheon-topology-act-${participant.agentId}`}
                        />
                        {latestTargetName && latestTargetName !== participant.name && (
                          <span className="pantheon-topology-target">指向 {latestTargetName}</span>
                        )}
                      </span>
                      <span>{preview}</span>
                    </span>
                    <span className="pantheon-topology-metrics">
                      <span>观点 {outputs.length}</span>
                      <span>工具 {traces.length}</span>
                      {participant.version && (
                        <span className="pantheon-topology-version" title="Harness / 模型版本">
                          {participant.version}
                        </span>
                      )}
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
                  {participant.status === 'running' && (
                    <button
                      type="button"
                      className="pantheon-topology-node-skip"
                      data-testid={`pantheon-skip-seat-${participant.agentId}`}
                      aria-label={`跳过 ${participant.name} 当前调用`}
                      title={`跳过 ${participant.name} 当前调用`}
                      onClick={(event) => {
                        event.stopPropagation()
                        onSkipAgent(participant.agentId)
                      }}
                    >
                      <StepForwardOutlined />
                    </button>
                  )}
                  {participant.status === 'failed' && (
                    <button
                      type="button"
                      className="pantheon-topology-node-skip is-retry"
                      data-testid={`pantheon-retry-seat-${participant.agentId}`}
                      aria-label={`重试 ${participant.name}`}
                      title={`重试 ${participant.name}`}
                      onClick={(event) => {
                        event.stopPropagation()
                        onRetryAgent(participant.agentId)
                      }}
                    >
                      <ReloadOutlined />
                    </button>
                  )}
                  </div>
                )
              })}
            </div>
          </div>
        </section>

        <div
          id="pantheon-panel-agents"
          className="pantheon-agent-windows-panel"
          hidden={recordView !== 'agents'}
        >
          {recordView === 'agents' && (
            <AgentWindowWall
              participants={participants}
              outputsByAgent={outputsByAgent}
              tracesByAgent={tracesByAgent}
              speakingAgentIds={speakingAgentIds}
              selectedAgentId={selectedAgentId}
              clockNow={clockNow}
              onSelect={selectAgent}
            />
          )}
        </div>

        <section
          id="pantheon-panel-shared"
          className={`pantheon-shared-dialogue${running ? ' is-live' : ''}`}
          aria-labelledby="pantheon-shared-title"
          data-testid="pantheon-shared-dialogue"
          hidden={recordView !== 'shared'}
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

          {session.verdict && (
            <section
              className={`pantheon-shared-verdict-preview${conclusionPreviewOpen ? ' is-open' : ''}`}
              data-testid="pantheon-shared-conclusion-preview"
            >
              <button
                type="button"
                className="pantheon-shared-verdict-summary"
                aria-expanded={conclusionPreviewOpen}
                aria-controls="pantheon-shared-conclusion-preview-body"
                data-testid="pantheon-shared-conclusion-toggle"
                onClick={() => setConclusionPreviewOpen((current) => !current)}
              >
                <span className="pantheon-shared-verdict-seal" aria-hidden>
                  {session.signoff ? '阅' : '记'}
                </span>
                <span className="pantheon-shared-verdict-copy">
                  <small>
                    共享结论 · {session.signoff ? '已记录' : '待你确认'}
                    {session.verdict.partial ? ' · 信息不完整' : ''}
                  </small>
                  <strong title={session.verdict.recommendation}>
                    {previewText(session.verdict.recommendation || '尚未形成明确的综合建议')}
                  </strong>
                </span>
                <span className="pantheon-shared-verdict-action">
                  {conclusionPreviewOpen ? '收起' : '预览'}
                  <DownOutlined aria-hidden />
                </span>
              </button>
              {conclusionPreviewOpen && (
                <div
                  id="pantheon-shared-conclusion-preview-body"
                  className="pantheon-shared-verdict-body"
                  data-testid="pantheon-shared-conclusion-body"
                >
                  {session.verdict.summary && <p>{session.verdict.summary}</p>}
                  <div className="pantheon-shared-verdict-facts">
                    {Boolean(session.verdict.proposedActions?.length) && (
                      <span>{session.verdict.proposedActions?.length} 项行动提案</span>
                    )}
                    {Boolean(session.verdict.conditions?.length) && (
                      <span>{session.verdict.conditions?.length} 项适用条件</span>
                    )}
                    {Boolean(session.verdict.disagreements?.length) && (
                      <span>{session.verdict.disagreements?.length} 项保留分歧</span>
                    )}
                    {Boolean(session.verdict.risks?.length) && (
                      <span>{session.verdict.risks?.length} 项风险</span>
                    )}
                    {Boolean(session.verdict.evidenceNotes?.length) && (
                      <span>{session.verdict.evidenceNotes?.length} 条证据说明</span>
                    )}
                  </div>
                  <button
                    type="button"
                    className="pantheon-shared-verdict-open"
                    data-testid="pantheon-shared-conclusion-open"
                    onClick={() => {
                      const reduce = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
                      conclusionRef.current?.scrollIntoView({
                        block: 'start',
                        behavior: reduce ? 'auto' : 'smooth',
                      })
                    }}
                  >
                    查看完整结论
                    <span aria-hidden>↓</span>
                  </button>
                </div>
              )}
            </section>
          )}

          <section
            className={`pantheon-shared-context${activeNames.length ? ' is-live' : ''}`}
            data-testid="pantheon-shared-context"
            aria-label="共享窗口当前状态"
          >
            <div className="pantheon-shared-context-main">
              <div className="pantheon-shared-context-lead" aria-live={running ? 'polite' : undefined}>
                <span className="pantheon-shared-context-avatars" aria-hidden>
                  {activeSpeakers.length > 0 ? (
                    <>
                      {activeSpeakers.slice(0, 2).map((participant) => (
                        <PantheonAgentAvatar
                          key={participant.agentId}
                          agentId={participant.agentId}
                          className="pantheon-shared-context-avatar"
                          status="running"
                        />
                      ))}
                      {activeSpeakers.length > 2 && (
                        <span className="pantheon-shared-context-more">+{activeSpeakers.length - 2}</span>
                      )}
                    </>
                  ) : (
                    <span className="pantheon-shared-context-mark">
                      {session.status === 'completed' ? <CheckCircleFilled /> : <TeamOutlined />}
                    </span>
                  )}
                </span>
                <span className="pantheon-shared-context-copy">
                  <small>{activeNames.length ? '当前 Agent' : running ? '协作状态' : '纪要状态'}</small>
                  <strong>{activityHeadline}</strong>
                </span>
                {activeSpeakers.length === 1 && activeStartedAt > 0 && (
                  <AgentElapsed
                    startedAt={activeStartedAt}
                    now={clockNow}
                    className="pantheon-shared-context-elapsed"
                    testId="pantheon-shared-current-elapsed"
                  />
                )}
              </div>
              <div className="pantheon-shared-context-meta">
                <span>{phaseLabel}</span>
                <span>{completedCount} / {participants.length} 已返回</span>
                {openIssueCount > 0 && <span className="is-warning">{openIssueCount} 项分歧待收口</span>}
              </div>
            </div>
            <div className="pantheon-shared-seats" aria-label="各 Agent 当前状态">
              {participants.map((participant) => {
                const active = speakingAgentIds.has(participant.agentId)
                const status = active ? 'running' : participant.status
                return (
                  <button
                    key={participant.agentId}
                    type="button"
                    className={`pantheon-shared-seat is-${status}${active ? ' is-current' : ''}`}
                    data-testid={`pantheon-shared-seat-${participant.agentId}`}
                    aria-current={active ? 'true' : undefined}
                    aria-label={`${participant.name}，${seatStatusText(participant, clockNow)}，打开独立窗口`}
                    style={{
                      '--agent-accent': AGENT_ACCENTS[participant.agentId] || 'var(--pantheon-accent)',
                    } as React.CSSProperties}
                    onClick={(event) => selectAgent(participant.agentId, event.currentTarget)}
                  >
                    <PantheonAgentAvatar
                      agentId={participant.agentId}
                      className="pantheon-shared-seat-avatar"
                      status={avatarStatus(status)}
                    />
                    <span>
                      <strong>{participant.name}</strong>
                      <small>
                        {active && <span className="pantheon-live-dot" aria-hidden />}
                        {seatStatusText(participant, clockNow)}
                      </small>
                    </span>
                  </button>
                )
              })}
            </div>
          </section>

          <div
            ref={streamRef}
            className="pantheon-stream pantheon-shared-stream"
            data-testid="pantheon-stream"
            role="log"
            aria-live={running ? 'polite' : undefined}
            aria-label="共享讨论消息"
          >
            {sharedTimelineSections.map((section) => {
              if (!section.heading) {
                return (
                  <section key={section.key} className="pantheon-shared-stage is-prelude">
                    {section.entries.map((entry) => (
                      <SharedTimelineEntry
                        key={entry.id}
                        entry={entry}
                        speakingAgentIds={speakingAgentIds}
                        foldedMap={foldedEntries}
                        onFold={toggleFolded}
                      />
                    ))}
                  </section>
                )
              }
              const open = roundOpenState[section.key]
                ?? section.key === latestTimelineSectionKey
              return (
                <details
                  key={section.key}
                  className={`pantheon-shared-stage is-round${open ? ' is-open' : ''}`}
                  data-testid={`pantheon-shared-stage-${section.heading.round || 'phase'}-${section.heading.phase}`}
                  open={open}
                >
                  <summary
                    className="pantheon-shared-round"
                    aria-label={`${section.heading.round ? `第 ${section.heading.round} 轮，` : ''}${section.heading.phase}，共 ${section.heading.count} 条发言，${open ? '收起' : '展开'}`}
                    onClick={(event) => {
                      event.preventDefault()
                      setRoundOpen(section.key, !open)
                    }}
                  >
                    <span className="pantheon-shared-round-marker" aria-hidden>
                      {section.heading.round ? String(section.heading.round).padStart(2, '0') : '汇'}
                    </span>
                    <span className="pantheon-shared-round-copy">
                      <strong>
                        {section.heading.round ? `第 ${section.heading.round} 轮` : section.heading.phase}
                      </strong>
                      <span>
                        {section.heading.round ? section.heading.phase : '阶段小结'} · {section.heading.count} 条发言
                      </span>
                    </span>
                    <span className="pantheon-shared-round-toggle">
                      {open ? '收起' : '展开'}
                      <DownOutlined aria-hidden />
                    </span>
                  </summary>
                  <div className="pantheon-shared-stage-body">
                    {section.entries.map((entry) => (
                      <SharedTimelineEntry
                        key={entry.id}
                        entry={entry}
                        speakingAgentIds={speakingAgentIds}
                        foldedMap={foldedEntries}
                        onFold={toggleFolded}
                      />
                    ))}
                  </div>
                </details>
              )
            })}
          </div>

          {(running || session.status === 'waiting_input') && (
            <div className="pantheon-note-composer" data-testid="pantheon-note-composer">
              <textarea
                className="pantheon-note-input"
                data-testid="pantheon-note-input"
                value={noteDraft}
                onChange={(event) => setNoteDraft(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' && !event.shiftKey) {
                    event.preventDefault()
                    submitUserNote()
                  }
                }}
                placeholder="插话评论：你的说明会进入讨论流，后续发言优先回应（Enter 发送，Shift+Enter 换行）"
                rows={2}
                aria-label="用户插话"
              />
              <button
                type="button"
                className="pantheon-note-send"
                data-testid="pantheon-note-send"
                onClick={submitUserNote}
                disabled={!noteDraft.trim() || noteSending}
                aria-label="发送插话"
              >
                发送
              </button>
            </div>
          )}

          {!running && (
            <div ref={conclusionRef} className="pantheon-record-body" aria-label="讨论小结">
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
