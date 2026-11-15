import type {
  PantheonAgentStatus,
  PantheonEvaluationReport,
  PantheonSession,
  PantheonSessionSummary,
  PantheonStartRequest,
  PantheonWorkspace,
} from '../types/pantheon'

export interface PantheonBindings {
  PantheonAgents(): Promise<PantheonAgentStatus[]>
  PantheonWorkspaces(): Promise<PantheonWorkspace[]>
  PantheonPickWorkspace(): Promise<string>
  PantheonList(): Promise<PantheonSessionSummary[]>
  PantheonEvaluation(minCases: number): Promise<PantheonEvaluationReport>
  PantheonGet(id: string): Promise<PantheonSession>
  PantheonStart(req: PantheonStartRequest): Promise<PantheonSession>
  PantheonCancel(id: string): Promise<void>
  PantheonSkipAgent(id: string, agentId: string): Promise<void>
  PantheonRetryAgent(id: string, agentId: string): Promise<void>
  PantheonRestart(id: string): Promise<PantheonSession>
  PantheonResume(id: string): Promise<PantheonSession>
  PantheonDelete(id: string): Promise<void>
  PantheonExport(id: string): Promise<string>
  PantheonOpenResult(id: string): Promise<string>
  PantheonSignOff(id: string, decision: string, note: string): Promise<PantheonSession>
  PantheonSetExecutable(id: string, path: string): Promise<PantheonAgentStatus[]>
  PantheonAskSeat(
    id: string,
    agentId: string,
    question: string,
    options: string[],
  ): Promise<PantheonSession>
  PantheonAnswerSeat(id: string, questionId: string, body: string): Promise<PantheonSession>
  PantheonAddUserNote(id: string, content: string): Promise<PantheonSession>
  PantheonOpenGate(id: string, reason: string): Promise<PantheonSession>
  PantheonResolveGate(
    id: string,
    action: string,
    reason: string,
  ): Promise<PantheonSession>
}
