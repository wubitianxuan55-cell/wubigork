import type { PantheonContribution, PantheonSession } from './types'

/**
 * 主席派工的投递回执（第二次蒸馏 E2）。
 *
 * 与后端 `chair_receipt.go` 同口径：只从 contributions/events 派生，不落盘。
 * spoke 只证明「下一轮发过言」，不证明「完成了指令」；UI 文案必须照此写。
 */
export type PantheonChairReceiptStatus =
  | 'spoke'
  | 'silent'
  | 'failed'
  | 'skipped'
  | 'unavailable'
  | 'pending'

export interface PantheonChairReceipt {
  round: number
  agentId: string
  agentName: string
  instruction: string
  status: PantheonChairReceiptStatus
  contributionId?: string
  detail?: string
}

export function chairAssignmentReceipts(
  session: PantheonSession,
  moderation: PantheonContribution,
): PantheonChairReceipt[] {
  const contributions = session.contributions ?? []
  const moderationIndex = contributions.findIndex((item) => item.id === moderation.id)
  if (moderationIndex < 0) return []
  const moderationSeq = moderationEventSeq(session, moderation)
  const names = new Map<string, string>()
  const statuses = new Map<string, string>()
  for (const participant of session.participants ?? []) {
    names.set(participant.agentId, participant.name)
    statuses.set(participant.agentId, participant.status)
  }
  const currentRound = session.currentRound ?? 0
  return (moderation.chairAssignments ?? []).map((assignment) => {
    const receipt: PantheonChairReceipt = {
      round: moderation.round ?? 0,
      agentId: assignment.agentId,
      agentName: names.get(assignment.agentId) || assignment.agentId,
      instruction: assignment.instruction,
      status: 'pending',
    }
    const spoke = firstContributionAfter(contributions, moderationIndex, assignment.agentId)
    if (spoke) {
      receipt.status = 'spoke'
      receipt.contributionId = spoke.id
      receipt.detail = '下一轮已发言（未证明完成指令）'
      return receipt
    }
    const outcome = seatOutcomeAfter(session, moderation, moderationSeq, assignment.agentId)
    if (outcome) {
      receipt.status = outcome.status
      receipt.detail = outcome.detail
      return receipt
    }
    if (statuses.get(assignment.agentId) === 'unavailable') {
      receipt.status = 'unavailable'
      receipt.detail = '席位不可用'
      return receipt
    }
    if ((moderation.round ?? 0) < currentRound) {
      receipt.status = 'silent'
      receipt.detail = '该轮未发言，也未记录失败/跳过'
    } else if (session.status === 'completed' || session.status === 'failed' || session.status === 'cancelled') {
      receipt.status = 'pending'
      receipt.detail = '议事在下一轮前结束'
    } else {
      receipt.status = 'pending'
      receipt.detail = '下一轮尚未开始'
    }
    return receipt
  })
}

export function receiptStatusLabel(status: PantheonChairReceiptStatus | string): string {
  switch (status) {
    case 'spoke': return '已发言（未证明完成）'
    case 'silent': return '未发言'
    case 'failed': return '失败'
    case 'skipped': return '已跳过'
    case 'unavailable': return '缺席'
    case 'pending': return '未到轮'
    default: return status
  }
}

function firstContributionAfter(
  contributions: PantheonContribution[],
  moderationIndex: number,
  agentId: string,
): PantheonContribution | undefined {
  for (let index = moderationIndex + 1; index < contributions.length; index += 1) {
    if (contributions[index].agentId === agentId) return contributions[index]
  }
  return undefined
}

function seatOutcomeAfter(
  session: PantheonSession,
  moderation: PantheonContribution,
  moderationSeq: number,
  agentId: string,
): { status: 'failed' | 'skipped'; detail: string } | undefined {
  let failed: string | undefined
  let skipped = false
  for (const event of session.events ?? []) {
    if (event.agentId !== agentId) continue
    if (moderationSeq > 0 && event.seq <= moderationSeq) continue
    if (moderationSeq <= 0 && event.at < moderation.createdAt) continue
    if (event.kind === 'agent_error') failed = event.text || event.error || '失败'
    else if (event.kind === 'agent_skipped') skipped = true
  }
  if (failed !== undefined) return { status: 'failed', detail: clip(failed) }
  if (skipped) return { status: 'skipped', detail: '该席在派工后的轮次被跳过' }
  return undefined
}

function moderationEventSeq(session: PantheonSession, moderation: PantheonContribution): number {
  let seq = 0
  for (const event of session.events ?? []) {
    if (event.kind !== 'agent_output') continue
    if (moderation.callId && event.callId !== moderation.callId) continue
    if (moderation.attemptId && event.attemptId !== moderation.attemptId) continue
    if (event.seq > seq) seq = event.seq
  }
  return seq
}

function clip(text: string): string {
  const flat = text.replace(/\s+/g, ' ').trim()
  return flat.length > 120 ? `${flat.slice(0, 119)}…` : flat
}
