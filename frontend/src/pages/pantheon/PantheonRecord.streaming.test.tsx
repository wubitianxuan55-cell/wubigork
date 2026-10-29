import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PantheonRecord } from './PantheonRecord'
import type { PantheonLiveStream, PantheonSession } from './types'

const session: PantheonSession = {
  id: 'pan_stream',
  title: '验证共享流式输出',
  status: 'running',
  phase: 'opening',
  currentRound: 1,
  createdAt: Date.now(),
  updatedAt: Date.now(),
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
    at: Date.now(),
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

describe('PantheonRecord streaming', () => {
  it('共享观点池以对话流显示，并同时保留四个独立窗口与右侧设置', () => {
    render(
      <PantheonRecord
        session={session}
        loading={false}
        busy={false}
        error=""
        liveStreams={liveStreams}
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

    const shared = screen.getByTestId('pantheon-shared-message-pan_stream:grok:opening:1')
    expect(shared.textContent).toContain('我正在补充一个关键判断')
    expect(shared.textContent).toContain('Grok')
    expect(shared.querySelector('.pantheon-stream-caret')).toBeTruthy()
    const grokWindow = screen.getByTestId('pantheon-live-seat-grok')
    expect(grokWindow.textContent).toContain('我正在补充一个关键判断')
    expect(grokWindow.querySelector('.pantheon-agent-message.is-streaming')).toBeTruthy()
    expect(grokWindow.querySelector('.pantheon-stream-caret')).toBeTruthy()
    expect(screen.getByTestId('pantheon-live-seat-codex')).toBeTruthy()
    expect(screen.getByTestId('pantheon-live-seat-zcode').textContent).toContain('未参与本次讨论')
    expect(screen.getByTestId('pantheon-live-seat-dsh').textContent).toContain('未参与本次讨论')

    const settings = screen.getByRole('complementary', { name: '讨论纪要摘要' })
    expect(settings.textContent).toContain('讨论范围')
    expect(settings.textContent).toContain('参与者')
    expect(settings.textContent).toContain('发言顺序')
  })
})
