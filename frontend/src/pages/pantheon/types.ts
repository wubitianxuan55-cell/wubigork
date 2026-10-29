import type {
  PantheonAgentStatus,
  PantheonEvidence,
  PantheonSession,
  PantheonSessionSummary,
  PantheonStartRequest,
  PantheonWorkspace,
} from '../../gaea/lib/types/pantheon'

export type * from '../../gaea/lib/types/pantheon'

export interface PantheonComposerSeed {
  key: string
  title?: string
  topic?: string
  context?: string
  successCriteria?: string
  workspace?: string
  evidence?: PantheonEvidence[]
  agentIds?: string[]
  chairmanAgentId?: string
  rounds?: number
  untilConsensus?: boolean
  maxRounds?: number
}

export interface PantheonApi {
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
