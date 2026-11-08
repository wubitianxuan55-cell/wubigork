import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { PantheonRecord } from './PantheonRecord'
import type { PantheonContribution, PantheonLiveStream, PantheonSession } from './types'

const now = Date.now()

const session: PantheonSession = {
  id: 'pan_stream',
  title: '验证共享流式输出',
  status: 'running',
  phase: 'opening',
  currentRound: 1,
  createdAt: now,
  updatedAt: now,
  request: {
    title: '验证共享流式输出',
    topic: '如何保留独立上下文并共享观点？',
    context: '',
    successCriteria: '',
    workspace: 'C:\\AI\\wubigrok',
    agentIds: ['grok', 'codex'],
    chairmanAgentId: 'grok',
    rounds: 2,
    readOnly: true,
  },
  participants: [
    {
      agentId: 'grok',
      name: 'Grok',
      role: 'participant',
      status: 'running',
      startedAt: now - 12_000,
    },
    { agentId: 'codex', name: 'Codex', role: 'participant', status: 'waiting' },
  ],
  events: [{
    seq: 1,
    sessionId: 'pan_stream',
    phase: 'opening',
    round: 1,
    kind: 'phase',
    text: '开始开场陈述',
    at: now,
  }],
  contributions: [],
}

const liveStreams: PantheonLiveStream[] = [{
  id: 'pan_stream:grok:opening:1',
  agentId: 'grok',
  phase: 'opening',
  round: 1,
  text: '我正在补充一个关键判断',
  streaming: true,
  startedAt: now - 12_000,
  updatedAt: now,
}]

const finalizedContributions: PantheonContribution[] = [{
  id: 'contribution_grok_1',
  phase: 'opening',
  round: 1,
  agentId: 'grok',
  agentName: 'Grok',
  content: '终结发言：关键判断已经成型。',
  createdAt: now - 500,
}]

function renderRecord(props: {
  liveStreams?: PantheonLiveStream[]
  contributions?: PantheonContribution[]
  sessionOverrides?: Partial<PantheonSession>
  onSkipAgent?: (agentId: string) => void
  onRetryAgent?: (agentId: string) => void
  onRestart?: () => void
  onAddUserNote?: (content: string) => void
  onResume?: () => void
} = {}) {
  return render(
    <PantheonRecord
      session={{
        ...session,
        contributions: props.contributions ?? [],
        ...props.sessionOverrides,
      }}
      loading={false}
      busy={false}
      error=""
      liveStreams={props.liveStreams ?? []}
      onCancel={() => {}}
      onSkipAgent={props.onSkipAgent ?? (() => {})}
      onRetryAgent={props.onRetryAgent ?? (() => {})}
      onRestart={props.onRestart ?? (() => {})}
      onResume={props.onResume ?? (() => {})}
      onDelete={() => {}}
      onRefresh={() => {}}
      onExport={() => {}}
      onExportJson={() => {}}
      onCopy={() => {}}
      onOpenResult={() => {}}
      onSignOff={() => {}}
      onContinue={() => {}}
      onAnswer={() => {}}
      onResolveGate={() => {}}
      onAddUserNote={props.onAddUserNote ?? (() => {})}
    />,
  )
}

describe('PantheonRecord streaming', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('当前 Agent 的流式观点同步进入共享线程，推理与工具仍留在详情', () => {
    const onSkipAgent = vi.fn()
    renderRecord({ liveStreams, onSkipAgent })

    const stream = screen.getByTestId('pantheon-stream')
    expect(stream.getAttribute('aria-live')).toBe('polite')
    expect(stream.textContent).toContain('我正在补充一个关键判断')
    expect(stream.textContent).toContain('正在协作')
    expect(stream.textContent).toContain('Grok')
    expect(stream.textContent).toContain('推理与工具过程保留在独立 Agent 详情')
    expect(stream.querySelector('.pantheon-shared-entry.is-agent.is-streaming')?.textContent)
      .toContain('我正在补充一个关键判断')
    expect(document.querySelector('.pantheon-agent-windows')).toBeNull()

    const node = screen.getByTestId('pantheon-topology-node-grok')
    expect(node.className).toContain('is-running')
    expect(node.getAttribute('aria-current')).toBe('true')
    expect(node.textContent).toContain('观点 1')
    expect(screen.getByTestId('pantheon-agent-elapsed-grok').textContent).toMatch(/工作 00:\d{2}/)

    const sharedCard = screen.getByTestId('pantheon-open-shared')
    expect(sharedCard.textContent).toContain('Grok 正在开场陈述')
    expect(sharedCard.textContent).toContain('0/2 已返回')
    expect(sharedCard.textContent).toContain('1 处理中')
    expect(sharedCard.textContent).toContain('1 观点')
    expect(screen.getByTestId('pantheon-shared-card-latest').textContent)
      .toContain('Grok · 我正在补充一个关键判断')

    const sharedContext = screen.getByTestId('pantheon-shared-context')
    expect(sharedContext.textContent).toContain('当前 Agent')
    expect(sharedContext.textContent).toContain('Grok 正在开场陈述')
    expect(screen.getByTestId('pantheon-shared-seat-codex').textContent).toContain('候场')
    const firstRound = document.querySelector('.pantheon-shared-round')
    expect(firstRound?.textContent).toContain('第 1 轮')
    expect(firstRound?.textContent).toContain('开场陈述 · 1 条发言')
    expect(firstRound?.querySelector('.pantheon-shared-round-marker')?.textContent).toBe('01')

    fireEvent.click(screen.getByTestId('pantheon-skip-current'))
    expect(onSkipAgent).toHaveBeenCalledWith('grok')

    fireEvent.click(node)
    const detail = screen.getByTestId('pantheon-agent-detail')
    expect(detail.getAttribute('aria-label')).toBe('Grok 独立输出')
    expect(detail.textContent).toContain('我正在补充一个关键判断')
    expect(detail.textContent).toContain('正在输出')
    expect(detail.querySelector('.pantheon-stream-caret')).toBeTruthy()
    expect(detail.textContent).toContain('推理过程与工具调用不会写入中央共享讨论')
  })

  it('拓扑节点补齐讨论动作、关联席位、版本与工具证据', () => {
    const topicContribution: PantheonContribution = {
      id: 'contribution_grok_claim',
      phase: 'opening',
      round: 1,
      agentId: 'grok',
      agentName: 'Grok',
      content: '先做可中止的只读验证。',
      discussionAct: 'claim',
      createdAt: now - 9000,
    }
    const responseContribution: PantheonContribution = {
      id: 'contribution_codex_reinforce',
      phase: 'discussion',
      round: 2,
      agentId: 'codex',
      agentName: 'Codex',
      content: '补一条单 Agent 基线，避免把讨论形式当成质量增益。',
      discussionAct: 'reinforce',
      targetContributionId: topicContribution.id,
      createdAt: now - 2000,
    }

    renderRecord({
      contributions: [topicContribution, responseContribution],
      sessionOverrides: {
        currentRound: 2,
        participants: [
          {
            ...session.participants[0],
            version: 'grok-4.1',
            status: 'finished',
            finishedAt: now - 8000,
          },
          {
            ...session.participants[1],
            version: 'codex-cli-0.165',
            status: 'finished',
            finishedAt: now - 1000,
          },
        ],
        events: [
          session.events[0],
          {
            seq: 2,
            sessionId: session.id,
            phase: 'discussion',
            round: 2,
            agentId: 'codex',
            kind: 'tool_started',
            toolId: 'tool-codex-1',
            toolName: 'rg',
            toolStatus: 'started',
            at: now - 3000,
          },
        ],
      },
    })

    const codex = screen.getByTestId('pantheon-topology-node-codex')
    expect(codex.textContent).toContain('第 2 轮')
    expect(codex.textContent).toContain('补强')
    expect(codex.textContent).toContain('指向 Grok')
    expect(codex.textContent).toContain('codex-cli-0.165')
    expect(codex.textContent).toContain('工具 1')

    const sharedCard = screen.getByTestId('pantheon-open-shared')
    expect(sharedCard.textContent).toContain('2/2 已返回')
    expect(sharedCard.textContent).toContain('2 观点')
    expect(sharedCard.textContent).toContain('1 工具')
    expect(screen.getByTestId('pantheon-shared-card-latest').textContent)
      .toContain('Codex · 补一条单 Agent 基线')
    expect(document.querySelectorAll('.pantheon-shared-round')).toHaveLength(2)
    expect(document.querySelectorAll('.pantheon-shared-entry.is-agent small')).toHaveLength(0)
  })

  it('无轮次的协作汇总仍有阶段标题，不出现多余的 0 轮', () => {
    renderRecord({
      contributions: [{
        id: 'contribution_grok_synthesis',
        phase: 'synthesis',
        agentId: 'grok',
        agentName: 'Grok',
        content: '各席已形成共同结论。',
        createdAt: now,
      }],
    })

    const headings = document.querySelectorAll('.pantheon-shared-round')
    expect(headings).toHaveLength(1)
    expect(headings[0].querySelector('.pantheon-shared-round-copy')?.textContent)
      .toBe('协作汇总阶段小结 · 1 条发言')
  })

  it('共享时间线默认展开最新轮次，并可按轮次收起展开', async () => {
    renderRecord({
      contributions: [
        {
          id: 'contribution_grok_round_1',
          phase: 'opening',
          round: 1,
          agentId: 'grok',
          agentName: 'Grok',
          content: '第一轮发言。',
          createdAt: now - 2000,
        },
        {
          id: 'contribution_codex_round_2',
          phase: 'discussion',
          round: 2,
          agentId: 'codex',
          agentName: 'Codex',
          content: '第二轮发言。',
          createdAt: now - 1000,
        },
      ],
    })

    const rounds = Array.from(document.querySelectorAll('details.pantheon-shared-stage'))
    expect(rounds).toHaveLength(2)
    const [firstRound, secondRound] = rounds
    expect(firstRound.hasAttribute('open')).toBe(false)
    expect(secondRound.hasAttribute('open')).toBe(true)

    fireEvent.click(firstRound.querySelector('summary')!)
    await waitFor(() => expect(firstRound.hasAttribute('open')).toBe(true))

    fireEvent.click(firstRound.querySelector('summary')!)
    await waitFor(() => expect(firstRound.hasAttribute('open')).toBe(false))
  })

  it('共享结论在时间线上方提供摘要预览和完整结论入口', () => {
    renderRecord({
      sessionOverrides: {
        status: 'completed',
        phase: 'completed',
        verdict: {
          summary: '各席认可先做可逆验证，但周期仍有分歧。',
          recommendation: '先完成小范围验证，再决定是否扩大。',
          conditions: ['只读运行'],
          disagreements: ['验证周期仍待对照'],
          risks: ['样本不足'],
          evidenceNotes: ['已有基线记录'],
          proposedActions: [{
            id: 'action-preview',
            action: '完成单席与多席对照',
          }],
          chairman: 'grok',
          partial: false,
          createdAt: now,
        },
      },
    })

    const preview = screen.getByTestId('pantheon-shared-conclusion-preview')
    const toggle = screen.getByTestId('pantheon-shared-conclusion-toggle')
    expect(preview.textContent).toContain('先完成小范围验证，再决定是否扩大。')
    expect(toggle.getAttribute('aria-expanded')).toBe('false')
    expect(screen.queryByTestId('pantheon-shared-conclusion-body')).toBeNull()

    fireEvent.click(toggle)
    const body = screen.getByTestId('pantheon-shared-conclusion-body')
    expect(body.textContent).toContain('各席认可先做可逆验证，但周期仍有分歧。')
    expect(body.textContent).toContain('1 项行动提案')
    expect(body.textContent).toContain('1 项保留分歧')
    expect(screen.getByTestId('pantheon-shared-conclusion-open').textContent)
      .toContain('查看完整结论')
  })

  it('拓扑、共享、各席窗口与台账都有稳定一级入口', () => {
    renderRecord({ liveStreams })

    const topology = document.getElementById('pantheon-panel-topology') as HTMLElement
    const shared = document.getElementById('pantheon-panel-shared') as HTMLElement
    const agents = document.getElementById('pantheon-panel-agents') as HTMLElement
    const ledger = document.getElementById('pantheon-panel-ledger') as HTMLElement
    expect(topology.hidden).toBe(false)
    expect(shared.hidden).toBe(true)
    expect(agents.hidden).toBe(true)

    fireEvent.click(screen.getByTestId('pantheon-open-shared'))
    expect(topology.hidden).toBe(true)
    expect(shared.hidden).toBe(false)

    fireEvent.click(screen.getByTestId('pantheon-view-topology'))
    expect(topology.hidden).toBe(false)
    expect(shared.hidden).toBe(true)

    fireEvent.click(screen.getByTestId('pantheon-view-agents'))
    expect(shared.hidden).toBe(true)
    expect(agents.hidden).toBe(false)
    expect(screen.getByTestId('pantheon-agent-window-grok').textContent)
      .toContain('我正在补充一个关键判断')
    fireEvent.click(screen.getByTestId('pantheon-agent-window-grok'))
    expect(screen.getByTestId('pantheon-agent-detail')).toBeTruthy()

    fireEvent.click(screen.getByTestId('pantheon-view-ledger'))
    expect(agents.hidden).toBe(true)
    expect(ledger.hidden).toBe(false)
  })

  it('讨论动作、目标论证与回应结论在共享流和台账中都可定位', () => {
    const challenge: PantheonContribution = {
      id: 'contribution_grok_2',
      phase: 'discussion',
      round: 2,
      agentId: 'grok',
      agentName: 'Grok',
      content: '成本边界仍缺少同任务实测。',
      discussionAct: 'challenge',
      targetContributionId: 'contribution_codex_1',
      challengedClaim: '两轮讨论一定优于单席直答',
      responseVerdict: 'partial',
      createdAt: now,
    }
    renderRecord({
      contributions: [finalizedContributions[0], challenge],
      sessionOverrides: {
        status: 'completed',
        phase: 'completed',
        currentRound: 2,
        issues: [{
          id: 'issue-challenge',
          round: 2,
          agentId: 'grok',
          agentName: 'Grok',
          summary: '成本边界缺少同任务实测',
          origin: 'challenge',
          targetContributionId: 'contribution_codex_1',
          challengedClaim: '两轮讨论一定优于单席直答',
          claimAgentIds: ['codex'],
          claimAgentNames: ['Codex'],
          dissentAgentIds: ['grok'],
          dissentAgentNames: ['Grok'],
          evidence: '当前只有单席估算【未核验】',
          responseVerdict: 'partial',
          responseEvidence: '两轮仅在反例发现率上领先',
          flipCondition: '完成 8 组同任务对照',
          verification: '记录单席与四席的反例数和调用成本',
          blocking: true,
          disposition: '部分接受；适用条件仍待闭合',
          status: 'dissent',
        }],
      },
    })

    const entry = screen.getByTestId('pantheon-shared-entry-shared-output-contribution_grok_2')
    expect(within(entry).getByTestId('pantheon-discussion-act-shared-output-contribution_grok_2').textContent)
      .toBe('质疑')
    expect(entry.textContent).toContain('被挑战命题：两轮讨论一定优于单席直答')
    expect(entry.textContent).toContain('目标贡献：contribution_codex_1')
    expect(entry.textContent).toContain('回应结论：部分接受')

    const ledger = screen.getByTestId('pantheon-issue-ledger')
    expect(ledger.textContent).toContain('来源：定向质疑')
    expect(ledger.textContent).toContain('回应结论部分接受')
    expect(ledger.textContent).toContain('回应证据两轮仅在反例发现率上领先')
  })

  it('DID 停止事件进入共享时间线而不是静默缩短轮次', () => {
    renderRecord({
      sessionOverrides: {
        status: 'completed',
        phase: 'completed',
        events: [
          ...session.events,
          {
            seq: 2,
            sessionId: session.id,
            phase: 'discussion',
            round: 2,
            kind: 'did_stop',
            text: '第 2 轮后没有新的可执行核验动作（DID）；停止追加轮次。',
            didKeys: [],
            at: now + 1,
          },
        ],
      },
    })
    expect(screen.getByText('停止空转')).toBeTruthy()
    expect(
      within(screen.getByTestId('pantheon-stream')).getByText(/没有新的可执行核验动作/),
    ).toBeTruthy()
  })

  it('行动提案只读展示依赖、验收与风险', () => {
    renderRecord({
      sessionOverrides: {
        status: 'completed',
        phase: 'completed',
        verdict: {
          summary: '结论',
          recommendation: '先验证',
          proposedActions: [{
            id: 'action-1',
            action: '执行 8 组同任务对照',
            rationale: '验证讨论增益',
            dependsOn: ['冻结模型版本'],
            acceptanceEvidence: '8 组原始记录',
            suggestedOwner: '用户',
            risk: '样本不足',
          }],
          chairman: 'Grok',
          partial: false,
          createdAt: now,
        },
      },
    })
    const plan = screen.getByTestId('pantheon-proposed-actions')
    expect(plan.textContent).toContain('行动提案 · 只读')
    expect(plan.textContent).toContain('执行 8 组同任务对照')
    expect(plan.textContent).toContain('8 组原始记录')
    expect(plan.textContent).toContain('不会自动运行')
  })

  it('红队评审展示独立确认和修订理由', () => {
    renderRecord({
      sessionOverrides: {
        status: 'completed',
        phase: 'completed',
        verdict: {
          summary: '修订后的结论',
          recommendation: '先补成本证据',
          revisedFromSha256: 'original-hash',
          revisionReason: '吸收 Beta 的成本证据缺口',
          chairman: 'Grok',
          partial: false,
          createdAt: now,
        },
        attestations: [
          {
            agentId: 'grok',
            agentName: 'Grok',
            position: 'support',
            reason: '结论与证据一致',
            verdictSha256: 'original-hash',
            createdAt: now,
          },
          {
            agentId: 'codex',
            agentName: 'Codex',
            position: 'oppose',
            reason: '成本证据不足',
            conditions: ['先完成 8 组对照'],
            issueIds: ['issue-cost'],
            verdictSha256: 'original-hash',
            createdAt: now,
          },
        ],
      },
    })
    const panel = screen.getByTestId('pantheon-attestations')
    expect(panel.textContent).toContain('1 支持 · 0 有条件 · 1 反对')
    expect(panel.textContent).toContain('已修订一次')
    expect(panel.textContent).toContain('吸收 Beta 的成本证据缺口')
    expect(panel.textContent).toContain('成本证据不足')
  })

  it('共享流长发言默认折叠成预览并可展开收起，流式与短发言不折叠', () => {
    const longText = '开场判断：先从可中止的垂直切片验证方向。'
      + '补充论据：成本边界可控，失败条件明确，需要更多长期复用数据支撑。'.repeat(14)
    renderRecord({
      contributions: [{
        id: 'contribution_long',
        phase: 'opening',
        round: 1,
        agentId: 'grok',
        agentName: 'Grok',
        content: longText,
        createdAt: now,
      }],
      sessionOverrides: {
        participants: [
          { agentId: 'grok', name: 'Grok', role: 'participant', status: 'finished' },
        ],
      },
    })

    const stream = screen.getByTestId('pantheon-stream')
    const fold = screen.getByTestId('pantheon-shared-fold-shared-output-contribution_long')
    expect(fold.getAttribute('aria-expanded')).toBe('false')
    expect(fold.textContent).toContain('展开全文')
    expect(stream.querySelector('.pantheon-shared-preview')).toBeTruthy()
    expect(stream.textContent).not.toContain(longText.slice(-40))

    fireEvent.click(fold)
    const opened = screen.getByTestId('pantheon-shared-fold-shared-output-contribution_long')
    expect(opened.getAttribute('aria-expanded')).toBe('true')
    expect(opened.textContent).toContain('收起')
    expect(stream.querySelector('.pantheon-shared-preview')).toBeNull()
    expect(stream.textContent).toContain(longText.slice(-40))

    fireEvent.click(opened)
    expect(screen.getByTestId('pantheon-shared-fold-shared-output-contribution_long').textContent)
      .toContain('展开全文')
    expect(screen.getByTestId('pantheon-stream').querySelector('.pantheon-shared-preview'))
      .toBeTruthy()
  })

  it('流式中的长发言保持全文展示不折叠，短发言不出现折叠按钮', () => {
    const longText = '流式判断：先从可中止的垂直切片验证方向。'
      + '补充论据：成本边界可控，失败条件明确，需要更多长期复用数据支撑。'.repeat(14)
    renderRecord({ liveStreams: [{ ...liveStreams[0], text: longText }] })
    const stream = screen.getByTestId('pantheon-stream')
    expect(stream.textContent).toContain(longText.slice(-40))
    expect(screen.queryByTestId('pantheon-shared-fold-shared-output-pan_stream:grok:opening:1'))
      .toBeNull()

    renderRecord({
      contributions: finalizedContributions,
      sessionOverrides: {
        participants: [
          { agentId: 'grok', name: 'Grok', role: 'participant', status: 'finished' },
        ],
      },
    })
    expect(screen.queryByTestId(/pantheon-shared-fold-/)).toBeNull()
  })

  it('运行超过两分钟没有新活动时提示等待时长并可重试当前席位', () => {
    const staleAt = now - 130_000
    const onRetryAgent = vi.fn()
    renderRecord({
      sessionOverrides: {
        createdAt: staleAt,
        updatedAt: staleAt,
        events: [{ ...session.events[0], at: staleAt }],
        participants: [
          {
            agentId: 'grok',
            name: 'Grok',
            role: 'participant',
            status: 'running',
            startedAt: staleAt,
          },
          { agentId: 'codex', name: 'Codex', role: 'participant', status: 'waiting' },
        ],
      },
      liveStreams: [{
        ...liveStreams[0],
        startedAt: staleAt,
        updatedAt: staleAt,
      }],
      onRetryAgent,
    })

    const notice = screen.getByTestId('pantheon-stall-notice')
    expect(notice.textContent).toContain('Grok')
    expect(notice.textContent).toContain('超过 2 分钟没有新输出')
    expect(notice.textContent).toContain('02:1')
    fireEvent.click(screen.getByTestId('pantheon-retry-agent'))
    expect(onRetryAgent).toHaveBeenCalledWith('grok')
  })

  it('Agent 调用报错后立即显示原因和重试入口，不必等待超时', () => {
    const onRetryAgent = vi.fn()
    const failure = 'bridge: chat stream: 等待 API 槽位: context canceled'
    renderRecord({
      sessionOverrides: {
        updatedAt: now,
        participants: [
          {
            agentId: 'grok',
            name: 'Grok',
            role: 'participant',
            status: 'failed',
            error: failure,
          },
          {
            agentId: 'codex',
            name: 'Codex',
            role: 'participant',
            status: 'running',
            startedAt: now,
          },
        ],
        events: [{
          seq: 2,
          sessionId: 'pan_stream',
          phase: 'opening',
          round: 1,
          agentId: 'grok',
          kind: 'agent_error',
          text: 'Grok 调用失败',
          error: failure,
          at: now,
        }],
      },
      onRetryAgent,
    })

    const notice = screen.getByTestId('pantheon-failure-notice')
    expect(notice.textContent).toContain('Grok 调用失败')
    expect(notice.textContent).toContain('等待 API 槽位: context canceled')
    fireEvent.click(screen.getByTestId('pantheon-retry-current'))
    expect(onRetryAgent).toHaveBeenCalledWith('codex')
  })

  it('只有已失败的整场记录才显示重开整场入口', () => {
    const onRestart = vi.fn()
    renderRecord({
      sessionOverrides: {
        status: 'failed',
        phase: 'failed',
        error: '所有参与者均未返回有效发言',
        participants: [
          { agentId: 'grok', name: 'Grok', role: 'participant', status: 'failed' },
        ],
      },
      onRestart,
    })
    expect(screen.queryByTestId('pantheon-retry-agent')).toBeNull()
    fireEvent.click(screen.getByTestId('pantheon-restart-all'))
    expect(onRestart).toHaveBeenCalledTimes(1)
  })

  it('正在工作的 Agent 读秒每秒递增', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-30T14:00:00+08:00'))
    const startedAt = Date.now() - 5_000
    renderRecord({
      sessionOverrides: {
        createdAt: startedAt,
        updatedAt: startedAt,
        participants: [{
          agentId: 'grok',
          name: 'Grok',
          role: 'participant',
          status: 'running',
          startedAt,
        }],
      },
    })

    expect(screen.getByTestId('pantheon-agent-elapsed-grok').textContent).toContain('00:05')
    act(() => {
      vi.advanceTimersByTime(1_000)
    })
    expect(screen.getByTestId('pantheon-agent-elapsed-grok').textContent).toContain('00:06')
  })

  it('Agent 独立窗口与共享窗口都按 Markdown 渲染观点', () => {
    renderRecord({
      contributions: [{
        id: 'contribution_markdown',
        phase: 'opening',
        round: 1,
        agentId: 'grok',
        agentName: 'Grok',
        content: '## 关键判断\n\n- **重点**：先验证\n- 次项：控制成本',
        createdAt: now,
      }],
      sessionOverrides: {
        participants: [
          { agentId: 'grok', name: 'Grok', role: 'participant', status: 'finished' },
        ],
      },
    })

    fireEvent.click(screen.getByTestId('pantheon-topology-node-grok'))
    const detail = screen.getByTestId('pantheon-agent-detail')
    expect(within(detail).getByRole('heading', { level: 2, name: '关键判断' })).toBeTruthy()
    expect(detail.querySelectorAll('li').length).toBe(2)
    expect(detail.textContent).toContain('重点：先验证')
    expect(detail.textContent).not.toContain('**重点**')

    const stream = screen.getByTestId('pantheon-stream')
    expect(stream.querySelector('.pantheon-shared-markdown h2')?.textContent).toBe('关键判断')
  })

  it('终结发言到账后按同键接管流式项，共享线程与详情各保留一条最终观点', () => {
    renderRecord({ liveStreams, contributions: finalizedContributions })

    const stream = screen.getByTestId('pantheon-stream')
    expect(stream.textContent).toContain('终结发言：关键判断已经成型。')
    expect(stream.querySelectorAll('.pantheon-shared-entry.is-agent').length).toBe(1)
    expect(stream.querySelector('.pantheon-shared-entry.is-agent')?.textContent).toContain('Grok')

    fireEvent.click(screen.getByTestId('pantheon-topology-node-grok'))
    const finalized = screen.getByTestId('pantheon-agent-output-contribution_grok_1')
    expect(finalized.className).not.toContain('is-streaming')
    expect(finalized.textContent).toContain('终结发言：关键判断已经成型。')
    expect(finalized.textContent).not.toContain('正在输出')
    expect(finalized.querySelector('.pantheon-stream-caret')).toBeNull()
    expect(screen.queryByTestId('pantheon-agent-output-pan_stream:grok:opening:1')).toBeNull()
  })

  it('用户中途评论以独立卡片进入讨论流，标「你」不被折叠当 agent 发言', () => {
    renderRecord({
      contributions: [{
        id: 'contribution_user_1',
        phase: 'user_note',
        round: 2,
        agentId: 'user',
        agentName: '用户',
        content: '请重点比较两案的运维成本。',
        createdAt: now - 300,
      }],
    })

    const stream = screen.getByTestId('pantheon-stream')
    const note = stream.querySelector('.pantheon-shared-entry.is-user')
    expect(note).not.toBeNull()
    expect(note?.textContent).toContain('你（插话）')
    expect(note?.textContent).toContain('请重点比较两案的运维成本。')
    expect(note?.textContent).toContain('优先回应')
  })

  it('运行中输入框发送插话回调 onAddUserNote，Enter 直发 Shift+Enter 换行', () => {
    const onAddUserNote = vi.fn()
    renderRecord({ onAddUserNote, sessionOverrides: { status: 'running' } })

    const input = screen.getByTestId('pantheon-note-input') as HTMLTextAreaElement
    fireEvent.change(input, { target: { value: '插一句：风险优先。' } })
    fireEvent.keyDown(input, { key: 'Enter', shiftKey: true })
    expect(onAddUserNote).not.toHaveBeenCalled()

    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onAddUserNote).toHaveBeenCalledTimes(1)
    expect(onAddUserNote).toHaveBeenCalledWith('插一句：风险优先。')

    fireEvent.click(screen.getByTestId('pantheon-note-send'))
    expect(onAddUserNote).toHaveBeenCalledTimes(1) // 空草稿不重复发送
  })

  it('详情面板每轮输出默认只展开最新一轮，其余折叠（2026-11-03 反馈）', () => {
    renderRecord({
      sessionOverrides: { status: 'completed' },
      contributions: [
        {
          id: 'contribution_grok_r1',
          phase: 'opening',
          round: 1,
          agentId: 'grok',
          agentName: 'Grok',
          content: '第一轮开场观点。',
          createdAt: now - 6000,
        },
        {
          id: 'contribution_grok_r2',
          phase: 'discussion',
          round: 2,
          agentId: 'grok',
          agentName: 'Grok',
          content: '第二轮补充论证。',
          createdAt: now - 3000,
        },
      ],
    })

    fireEvent.click(screen.getByTestId('pantheon-topology-node-grok'))
    const first = screen.getByTestId('pantheon-agent-output-contribution_grok_r1')
    const second = screen.getByTestId('pantheon-agent-output-contribution_grok_r2')
    expect(first.querySelector('.pantheon-agent-output-fold')?.hasAttribute('open')).toBe(false)
    expect(second.querySelector('.pantheon-agent-output-fold')?.hasAttribute('open')).toBe(true)

    // 手动展开第一轮：状态被记录，重渲染不弹回
    fireEvent.click(first.querySelector('.pantheon-agent-output-summary')!)
    expect(first.querySelector('.pantheon-agent-output-fold')?.hasAttribute('open')).toBe(true)
  })

  it('纯工具期（running 无正文直播流）显示工作卡，标注正在调用工具（2026-11-03 反馈）', () => {
    renderRecord({ sessionOverrides: { status: 'running' } })

    // fakeAdapter 场景无 liveStreams：running 席以工作卡呈现，不再隐形
    const working = document.querySelectorAll('.pantheon-shared-entry.is-working')
    expect(working.length).toBeGreaterThan(0)
    expect(working[0].textContent).toContain('正在')
  })

  it('拓扑节点恒显独立跳过钮，可指定席跳过（并行语义：running 席皆可跳）', () => {
    const onSkipAgent = vi.fn()
    renderRecord({ onSkipAgent, sessionOverrides: { status: 'running' } })

    const button = screen.getByTestId('pantheon-skip-seat-grok')
    fireEvent.click(button)
    expect(onSkipAgent).toHaveBeenCalledWith('grok')
  })
})

// 2026-11-08 实弹 pan_1794098356822 翻车回归：末轮席位提问答完后整场停在
// waiting_input，等待面板消失、恢复钮只在失败告示里、续议按钮依赖尚未生成的
// 汇总——三无死路。修复=待办清空时出「可继续」面板，续跑一键可达。
describe('PantheonRecord waiting clear panel', () => {
  const answeredQuestionEvents = [
    {
      seq: 1,
      sessionId: 'pan_stream',
      phase: 'discussion',
      round: 2,
      kind: 'question',
      text: '接受 fail-closed，还是安排 WSL2/容器？',
      at: now - 1000,
      question: {
        questionId: 'q-281',
        askedBy: 'dsh',
        origin: 'seat',
        default: '先 fail-closed',
        at: now - 1000,
      },
    },
    {
      seq: 2,
      sessionId: 'pan_stream',
      phase: 'discussion',
      kind: 'answer',
      text: '安排 WSL2/容器',
      at: now,
      answer: { questionId: 'q-281', body: '安排 WSL2/容器', by: 'user', at: now },
    },
  ] as PantheonSession['events']

  it('末轮提问答完后，waiting_input 必须给出可继续面板与续跑按钮', () => {
    const onResume = vi.fn()
    renderRecord({
      onResume,
      sessionOverrides: {
        status: 'waiting_input',
        phase: 'discussion',
        currentRound: 2,
        events: answeredQuestionEvents,
      },
    })
    expect(screen.getByTestId('pantheon-waiting-clear')).toBeTruthy()
    const button = screen.getByTestId('pantheon-resume-clear')
    fireEvent.click(button)
    expect(onResume).toHaveBeenCalledTimes(1)
  })

  it('同状态但仍有未答提问时，保持原等待面板（不出清空面板）', () => {
    const unanswered = answeredQuestionEvents!.slice(0, 1)
    renderRecord({
      sessionOverrides: {
        status: 'waiting_input',
        phase: 'discussion',
        currentRound: 2,
        events: unanswered,
      },
    })
    expect(screen.getByTestId('pantheon-waiting')).toBeTruthy()
    expect(screen.queryByTestId('pantheon-waiting-clear')).toBeNull()
  })

  it('非 waiting_input（如 running）且无待办时维持原状不渲染任何面板', () => {
    renderRecord({
      sessionOverrides: {
        status: 'running',
        events: answeredQuestionEvents,
      },
    })
    expect(screen.queryByTestId('pantheon-waiting-clear')).toBeNull()
    expect(screen.queryByTestId('pantheon-waiting')).toBeNull()
  })
})

// 2026-11-08 UI 同步：对账线进记录页——贡献卡在「标注【工具核验】但无非空
// 工具返回」时亮不可对证徽标；正常证据不亮；M1 任务书快照进 tooltip。
describe('PantheonRecord reconciliation badge', () => {
  // completed 态走共享流视图（running=拓扑视图，席位卡默认折叠拿不到 meta）。
  const baseContribution = {
    id: 'contrib_m1',
    phase: 'discussion',
    round: 2,
    agentId: 'grok',
    agentName: 'Grok',
    content: '【工具核验】已读 a.go：package a（出处：a.go 第 1 行）',
    createdAt: now,
    promptSha256: 'a1b2c3d4e5f60718',
    promptRunes: 3188,
    toolEvidence: {
      calls: 1,
      ok: 1,
      nonEmpty: 0,
    },
  }

  it('标注工具核验但零非空返回 → 共享卡亮对账徽标，拓扑卡与 tooltip 同步', () => {
    renderRecord({
      contributions: [{ ...baseContribution } as PantheonContribution],
    })
    // 共享卡：agent 卡体下挂对账警示小字（meta 对 agent 卡不渲染，走独立元素）。
    const stream = screen.getByTestId('pantheon-stream')
    expect(stream.textContent).toContain('工具核验不可对证')
    // 席位拓扑卡：点开节点后工具证据/任务书快照/警示同面可见。
    fireEvent.click(screen.getByTestId('pantheon-topology-node-grok'))
    const card = screen.getByTestId('pantheon-agent-output-contrib_m1')
    expect(card.textContent).toContain('工具核验不可对证')
    expect(card.textContent).toContain('1 次工具')
    const tooltip = card.querySelector('[title]')
    expect(tooltip?.getAttribute('title') || '').toContain('任务书快照：3188 字 · sha256:a1b2c3d4e5f60718')
  })

  it('有非空返回 → 不亮徽标，工具证据在拓扑卡照常显示', () => {
    renderRecord({
      contributions: [{
        ...baseContribution,
        toolEvidence: { calls: 1, ok: 1, nonEmpty: 1 },
      } as PantheonContribution],
    })
    const stream = screen.getByTestId('pantheon-stream')
    expect(stream.textContent).not.toContain('工具核验不可对证')
    // 工具证据标签属拓扑卡面（agent 卡共享体不渲染 meta 标签）。
    fireEvent.click(screen.getByTestId('pantheon-topology-node-grok'))
    const card = screen.getByTestId('pantheon-agent-output-contrib_m1')
    expect(card.textContent).not.toContain('工具核验不可对证')
    expect(card.textContent).toContain('1 次工具')
  })
})

// UI 细节同步（2026-11-08）：互盲驳论会话的记录页协议标签不再错显「圆桌接力」；
// 设了时间预算的会话在讨论范围里露预算行。讨论范围住在台账视图的纪要披露里
// （pantheon-meta-toggle 一键开），测试走真实交互路径。
describe('PantheonRecord meta labels', () => {
  it('互盲驳论会话记录页显示「互盲驳论」标签', () => {
    renderRecord({
      sessionOverrides: { request: { ...session.request, protocol: 'blind_rebuttal' } },
    })
    fireEvent.click(screen.getByTestId('pantheon-meta-toggle'))
    const meta = screen.getByTestId('pantheon-meta')
    expect(meta.textContent).toContain('互盲驳论')
    expect(meta.textContent).not.toContain('圆桌接力')
  })

  it('设了时间预算 → 讨论范围露预算行', () => {
    renderRecord({
      sessionOverrides: { request: { ...session.request, timeBudgetMinutes: 45 } },
    })
    fireEvent.click(screen.getByTestId('pantheon-meta-toggle'))
    const meta = screen.getByTestId('pantheon-meta')
    expect(meta.textContent).toContain('时间预算')
    expect(meta.textContent).toContain('45 分钟')
  })

  it('未设时间预算 → 不露预算行（旧会话零迁移）', () => {
    renderRecord()
    fireEvent.click(screen.getByTestId('pantheon-meta-toggle'))
    expect(screen.getByTestId('pantheon-meta').textContent).not.toContain('时间预算')
  })
})
