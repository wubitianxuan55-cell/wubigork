import { describe, expect, it } from 'vitest'
import { chairAssignmentReceipts, receiptStatusLabel } from './chairReceipts'
import type { PantheonContribution, PantheonSession } from './types'

function makeSession(): PantheonSession {
  return {
    id: 'pan_test',
    title: '回执',
    status: 'running',
    phase: 'discussion',
    currentRound: 2,
    createdAt: 100,
    updatedAt: 200,
    request: {
      title: '回执', topic: '回执', context: '', successCriteria: '',
      agentIds: ['alpha', 'beta', 'gamma', 'delta'], chairmanAgentId: 'alpha', rounds: 2, readOnly: true,
    },
    participants: [
      { agentId: 'alpha', name: 'ALPHA', role: 'participant', status: 'finished' },
      { agentId: 'beta', name: 'BETA', role: 'participant', status: 'skipped' },
      { agentId: 'gamma', name: 'GAMMA', role: 'participant', status: 'failed' },
      { agentId: 'delta', name: 'DELTA', role: 'participant', status: 'unavailable' },
    ],
    contributions: [
      {
        id: 'mod-1', phase: 'moderation', round: 1, agentId: 'alpha', agentName: 'ALPHA（主持）',
        content: '小结', createdAt: 100, callId: 'call-1', attemptId: 'attempt-1',
        chairAssignments: [
          { agentId: 'alpha', instruction: '复核' },
          { agentId: 'beta', instruction: '补反例' },
          { agentId: 'gamma', instruction: '核验' },
          { agentId: 'delta', instruction: '补来源' },
        ],
      },
      { id: 'alpha-r2', phase: 'discussion', round: 2, agentId: 'alpha', agentName: 'ALPHA', content: '已复核', createdAt: 200 },
    ],
    events: [
      { seq: 10, sessionId: 'pan_test', phase: 'moderation', kind: 'agent_output', agentId: 'alpha', callId: 'call-1', attemptId: 'attempt-1', at: 100 },
      { seq: 11, sessionId: 'pan_test', phase: 'discussion', kind: 'agent_skipped', agentId: 'beta', at: 110, text: '跳过' },
      { seq: 12, sessionId: 'pan_test', phase: 'discussion', kind: 'agent_error', agentId: 'gamma', at: 120, text: '版本核验超时' },
    ],
  } as PantheonSession
}

describe('chairAssignmentReceipts', () => {
  it('spoke / skipped / failed / unavailable 各归其位，且 spoke 不写成完成', () => {
    const session = makeSession()
    const moderation = session.contributions[0] as PantheonContribution
    const receipts = chairAssignmentReceipts(session, moderation)
    const byAgent = Object.fromEntries(receipts.map((item) => [item.agentId, item]))
    expect(byAgent.alpha.status).toBe('spoke')
    expect(byAgent.alpha.contributionId).toBe('alpha-r2')
    expect(byAgent.beta.status).toBe('skipped')
    expect(byAgent.gamma.status).toBe('failed')
    expect(byAgent.delta.status).toBe('unavailable')
    expect(receiptStatusLabel(byAgent.alpha.status)).toBe('已发言（未证明完成）')
  })

  it('没有下一轮时是 pending，不是失败', () => {
    const session = makeSession()
    session.currentRound = 1
    session.status = 'completed'
    const moderation = session.contributions[0] as PantheonContribution
    session.contributions = [moderation]
    session.events = session.events.filter((event) => event.agentId !== 'beta')
    const receipts = chairAssignmentReceipts(session, moderation)
    const beta = receipts.find((item) => item.agentId === 'beta')
    expect(beta?.status).toBe('pending')
    expect(beta?.detail).toContain('下一轮前结束')
  })
})
