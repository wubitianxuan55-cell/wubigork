export interface SeatUsageLike {
  engineSessionId?: string
  steps?: number
  inputTokens?: number
  cacheReadTokens?: number
  outputTokens?: number
}

export interface SeatOutputLike {
  id: string
  agentId: string
  agentName: string
  phase: string
  round?: number
  at?: number
  text: string
  durationMs?: number
  discussionAct?: string
  targetContributionId?: string
  targetIssueId?: string
  challengedClaim?: string
  responseVerdict?: string
  usage?: SeatUsageLike
}

export interface SeatToolTraceLike {
  id: string
  agentId: string
  toolName: string
  status: string
  input?: string
  at?: number
}

export interface SeatIssueLike {
  id: string
  agentId: string
  agentName?: string
  summary?: string
}

export interface SeatContextSummary {
  hasReportedUsage: boolean
  inputTokens: number
  cacheReadTokens: number
  outputTokens: number
  steps: number
  durationMs: number
  cacheRate: number | null
  engineSessionId?: string
}

export type SeatTrajectoryItem =
  | {
      id: string
      kind: 'statement'
      at: number
      round?: number
      phase: string
      act?: string
      text: string
      relation?: string
    }
  | {
      id: string
      kind: 'tool'
      at: number
      phase: string
      text: string
      status: string
    }

export function summarizeSeatContext(outputs: readonly SeatOutputLike[]): SeatContextSummary {
  let inputTokens = 0
  let cacheReadTokens = 0
  let outputTokens = 0
  let steps = 0
  let durationMs = 0
  let engineSessionId: string | undefined
  let hasReportedUsage = false

  for (const output of outputs) {
    if (output.durationMs && output.durationMs > 0) durationMs += output.durationMs
    if (output.usage) {
      hasReportedUsage = true
      inputTokens += output.usage.inputTokens ?? 0
      cacheReadTokens += output.usage.cacheReadTokens ?? 0
      outputTokens += output.usage.outputTokens ?? 0
      steps += output.usage.steps ?? 0
      engineSessionId = output.usage.engineSessionId || engineSessionId
    }
  }

  const promptTokens = inputTokens + cacheReadTokens
  return {
    hasReportedUsage,
    inputTokens,
    cacheReadTokens,
    outputTokens,
    steps,
    durationMs,
    cacheRate: hasReportedUsage && promptTokens > 0
      ? Math.round((cacheReadTokens * 100) / promptTokens)
      : null,
    ...(engineSessionId ? { engineSessionId } : {}),
  }
}

export function buildSeatTrajectory(
  outputs: readonly SeatOutputLike[],
  traces: readonly SeatToolTraceLike[],
  agentNameById: ReadonlyMap<string, string>,
  issues: readonly SeatIssueLike[] = [],
  targetOutputs: readonly SeatOutputLike[] = outputs,
): SeatTrajectoryItem[] {
  const outputById = new Map(targetOutputs.map((output) => [output.id, output]))
  const issueById = new Map(issues.map((issue) => [issue.id, issue]))
  const statements = outputs.map((output): SeatTrajectoryItem => {
    const targetOutput = output.targetContributionId
      ? outputById.get(output.targetContributionId)
      : undefined
    const targetIssue = output.targetIssueId
      ? issueById.get(output.targetIssueId)
      : undefined
    const targetName = targetOutput
      ? targetOutput.agentName || agentNameById.get(targetOutput.agentId) || targetOutput.agentId
      : targetIssue
        ? targetIssue.agentName || agentNameById.get(targetIssue.agentId) || targetIssue.agentId
        : ''
    const relation = [
      targetName ? `指向 ${targetName}` : '',
      output.challengedClaim ? `被挑战命题：${output.challengedClaim}` : '',
      output.responseVerdict ? `回应结论：${output.responseVerdict}` : '',
    ].filter(Boolean).join(' · ')
    return {
      id: `statement-${output.id}`,
      kind: 'statement',
      at: output.at ?? 0,
      ...(output.round ? { round: output.round } : {}),
      phase: output.phase,
      ...(output.discussionAct ? { act: output.discussionAct } : {}),
      text: output.text,
      ...(relation ? { relation } : {}),
    }
  })
  const tools = traces.map((trace): SeatTrajectoryItem => ({
    id: `tool-${trace.id}`,
    kind: 'tool',
    at: trace.at ?? 0,
    phase: 'tool',
    text: `${trace.toolName}${trace.input ? ` · ${trace.input}` : ''}`,
    status: trace.status,
  }))
  return [...statements, ...tools].sort((a, b) => a.at - b.at)
}
