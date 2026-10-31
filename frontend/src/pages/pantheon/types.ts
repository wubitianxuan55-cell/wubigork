import type {
  PantheonAgentStatus,
  PantheonAttachment,
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
  attachments?: PantheonAttachment[]
  agentIds?: string[]
  chairmanAgentId?: string
  rounds?: number
  untilConsensus?: boolean
  maxRounds?: number
  protocol?: string
  parentSessionId?: string
}

export interface PantheonApi {
  PantheonAgents(): Promise<PantheonAgentStatus[]>
  PickFiles(filters?: string): Promise<Array<{
    path: string
    name: string
    size?: number
    type?: 'image' | 'file'
    previewUrl?: string
  }>>
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
