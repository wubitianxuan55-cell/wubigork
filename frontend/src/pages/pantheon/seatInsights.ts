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
  round?: number
  /** 回执/错误（轨迹合并刀：展开检查器需要，ToolTraceView 本就有）。 */
  output?: string
  error?: string
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
      round?: number
      phase: string
      text: string
      status: string
      /** 工具入参/回执/错误全文（2026-11-18 轨迹 tab 合并刀：工具调用并入
       * 轨迹时间线后，展开检查器直接吃这三栏，不再依赖独立工具列表）。 */
      input?: string
      output?: string
      error?: string
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
    ...(trace.round ? { round: trace.round } : {}),
    phase: 'tool',
    text: `${trace.toolName}${trace.input ? ` · ${trace.input}` : ''}`,
    status: trace.status,
    ...(trace.input ? { input: trace.input } : {}),
    ...(trace.output ? { output: trace.output } : {}),
    ...(trace.error ? { error: trace.error } : {}),
  }))
  const sorted = [...statements, ...tools].sort((a, b) => a.at - b.at)
  // 旧后端/测试夹具的工具事件可能不报 round。按时间归入此前最近一次发言的
  // 轮次；仍无法归类时留在「轮外记录」，不猜造轮次。
  let currentRound: number | undefined
  return sorted.map((item) => {
    if (item.kind === 'statement' && item.round) currentRound = item.round
    if (item.kind === 'tool' && !item.round && currentRound) {
      return { ...item, round: currentRound }
    }
    return item
  })
}
