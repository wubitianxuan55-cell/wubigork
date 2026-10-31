import type {
  PantheonAgentStatus,
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
}
