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
  PantheonDelete(id: string): Promise<void>
  PantheonExport(id: string): Promise<string>
  PantheonSignOff(id: string, decision: string, note: string): Promise<PantheonSession>
  PantheonSetExecutable(id: string, path: string): Promise<PantheonAgentStatus[]>
}
