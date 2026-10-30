import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
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
    { agentId: 'grok', name: 'Grok', role: 'participant', status: 'running' },
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

function renderRecord(props: { liveStreams?: PantheonLiveStream[]; contributions?: PantheonContribution[] } = {}) {
  return render(
    <PantheonRecord
      session={{ ...session, contributions: props.contributions ?? [] }}
      loading={false}
      busy={false}
      error=""
      liveStreams={props.liveStreams ?? []}
      onCancel={() => {}}
      onDelete={() => {}}
      onRefresh={() => {}}
      onExport={() => {}}
      onExportJson={() => {}}
      onCopy={() => {}}
      onSignOff={() => {}}
      onContinue={() => {}}
    />,
  )
}

describe('PantheonRecord streaming', () => {
  it('流式发言以统一讨论流消息呈现：is-streaming 高亮 + 流式光标', () => {
    renderRecord({ liveStreams })

    const stream = screen.getByTestId('pantheon-stream')
    expect(stream.getAttribute('aria-live')).toBe('polite')
    const shared = screen.getByTestId('pantheon-shared-message-pan_stream:grok:opening:1')
    expect(stream.contains(shared)).toBe(true)
    expect(shared.className).toContain('is-streaming')
    expect(shared.textContent).toContain('我正在补充一个关键判断')
    expect(shared.textContent).toContain('Grok')
    expect(shared.querySelector('.pantheon-stream-caret')).toBeTruthy()
    // 逐席 accent 落到消息块上
    expect(shared.querySelector('.pantheon-stream-avatar')).toBeTruthy()

    const settings = screen.getByRole('complementary', { name: '讨论纪要摘要' })
    expect(settings.textContent).toContain('讨论范围')
    expect(settings.textContent).toContain('参与者')
    expect(settings.textContent).toContain('发言顺序')
  })

  it('终结发言到账后按同键接管流式项：流内只保留一条该席消息且不再流式', () => {
    renderRecord({ liveStreams, contributions: finalizedContributions })

    const stream = screen.getByTestId('pantheon-stream')
    // 同键（grok:opening:1）的流式项被终结发言替换，不重复出现
    expect(screen.queryByTestId('pantheon-shared-message-pan_stream:grok:opening:1')).toBeNull()
    const finalized = screen.getByTestId('pantheon-shared-message-contribution_grok_1')
    expect(stream.contains(finalized)).toBe(true)
    expect(finalized.className).not.toContain('is-streaming')
    expect(finalized.textContent).toContain('终结发言：关键判断已经成型。')
    expect(finalized.querySelector('.pantheon-stream-caret')).toBeNull()
    // 终结发言按发生时刻排在流里，没有残留空态
    expect(screen.queryByText('等待第一条公开观点')).toBeNull()
  })
})
