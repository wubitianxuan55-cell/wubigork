import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, within } from '@testing-library/react'
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
      onResume={() => {}}
      onDelete={() => {}}
      onRefresh={() => {}}
      onExport={() => {}}
      onExportJson={() => {}}
      onCopy={() => {}}
      onOpenResult={() => {}}
      onSignOff={() => {}}
      onContinue={() => {}}
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
})
