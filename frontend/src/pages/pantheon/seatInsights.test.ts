import { describe, expect, it } from 'vitest'
import {
  buildSeatTrajectory,
  summarizeSeatContext,
  type SeatOutputLike,
} from './seatInsights'

const OUTPUTS: SeatOutputLike[] = [
  {
    id: 'out-1',
    agentId: 'codex',
    agentName: 'Codex',
    phase: 'opening',
    round: 1,
    at: 100,
    text: '复用信息架构，不复制父子树。',
    discussionAct: 'claim',
    usage: {
      engineSessionId: 'session-a',
      steps: 8,
      inputTokens: 80_000,
      cacheReadTokens: 20_000,
      outputTokens: 4_000,
    },
  },
  {
    id: 'out-2',
    agentId: 'dsh',
    agentName: 'DeepSeek Harness',
    phase: 'challenge',
    round: 2,
    at: 200,
    text: '父子关系会破坏平权语义。',
    discussionAct: 'challenge',
    targetContributionId: 'out-1',
    durationMs: 4200,
    usage: {
      engineSessionId: 'session-b',
      steps: 4,
      inputTokens: 40_000,
      cacheReadTokens: 20_000,
      outputTokens: 2_000,
    },
  },
  {
    id: 'out-3',
    agentId: 'grok',
    agentName: 'Grok',
    phase: 'respond',
    round: 2,
    at: 300,
    text: '部分接受，箭头只表示回应。',
    discussionAct: 'respond',
    targetIssueId: 'issue-1',
    responseVerdict: 'partial',
  },
]

describe('seatInsights', () => {
  it('summarizes only reported usage and computes cache share honestly', () => {
    const codex = summarizeSeatContext([OUTPUTS[0]])
    expect(codex).toMatchObject({
      hasReportedUsage: true,
      inputTokens: 80_000,
      cacheReadTokens: 20_000,
      outputTokens: 4_000,
      steps: 8,
      cacheRate: 20,
      engineSessionId: 'session-a',
    })

    expect(summarizeSeatContext([OUTPUTS[2]])).toMatchObject({
      hasReportedUsage: false,
      inputTokens: 0,
      cacheRate: null,
    })
  })

  it('merges statements and tools in time order with human-readable targets', () => {
    const trajectory = buildSeatTrajectory(
      OUTPUTS,
      [
        { id: 'trace-1', agentId: 'dsh', toolName: 'read_file', status: 'completed', input: 'topology.tsx', at: 150 },
      ],
      new Map([
        ['codex', 'Codex'],
        ['dsh', 'DeepSeek Harness'],
        ['grok', 'Grok'],
      ]),
      [{ id: 'issue-1', agentId: 'codex', agentName: 'Codex' }],
    )
    expect(trajectory.map((item) => item.kind)).toEqual(['statement', 'tool', 'statement', 'statement'])
    expect(trajectory[2]).toMatchObject({
      kind: 'statement',
      relation: expect.stringContaining('指向 Codex'),
    })
    expect(trajectory[3]).toMatchObject({
      kind: 'statement',
      relation: expect.stringContaining('回应结论：partial'),
    })
  })
})
