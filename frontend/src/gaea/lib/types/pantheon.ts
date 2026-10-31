export interface PantheonAgentStatus {
  id: string
  name: string
  kind: string
  available: boolean
  executable?: string
  version?: string
  reason?: string
  capabilities?: string[]
}

export interface PantheonEvidence {
  name: string
  content: string
}

export interface PantheonAttachment {
  name: string
  path: string
  mime: string
  size: number
}

export interface PantheonWorkspace {
  path: string
  name: string
  current: boolean
  recent?: boolean
}

export interface PantheonStartRequest {
  title: string
  topic: string
  context: string
  successCriteria: string
  workspace?: string
  evidence?: PantheonEvidence[]
  attachments?: PantheonAttachment[]
  agentIds: string[]
  chairmanAgentId: string
  rounds: number
  untilConsensus?: boolean
  maxRounds?: number
  readOnly: boolean
  /** 议事路径：council=圆桌接力；independent=独立开场（默认，第 1 轮不注入同轮前序） */
  protocol?: string
  /** 续议来源会话 ID（裁决第三刀：保留父会话谱系；承接以简报为准） */
  parentSessionId?: string
}

export interface PantheonParticipant {
  agentId: string
  name: string
  role: string
  status: string
  startedAt?: number
  finishedAt?: number
  durationMs?: number
  error?: string
}

export interface PantheonEvent {
  seq: number
  sessionId: string
  phase: string
  round?: number
  agentId?: string
  kind: string
  text?: string
  toolId?: string
  toolName?: string
  toolStatus?: 'started' | 'completed' | 'failed' | string
  toolInput?: string
  toolOutput?: string
  at: number
  durationMs?: number
  error?: string
}

export interface PantheonLiveStream {
  id: string
  agentId: string
  phase: string
  round: number
  text: string
  streaming: boolean
  startedAt?: number
  updatedAt?: number
}

export interface PantheonContribution {
  id: string
  phase: string
  round?: number
  agentId: string
  agentName: string
  anonLabel?: string
  content: string
  confidence?: string
  consensus?: 'yes' | 'no' | string
  createdAt: number
  durationMs?: number
}

export interface PantheonVerdict {
  summary: string
  recommendation: string
  conditions?: string[]
  disagreements?: string[]
  risks?: string[]
  evidenceNotes?: string[]
  chairman: string
  partial: boolean
  createdAt: number
}

export interface PantheonSignOff {
  decision: 'accepted' | 'needs_revision' | 'rejected' | string
  note?: string
  at: number
}

export interface PantheonSession {
  id: string
  title: string
  status: 'running' | 'completed' | 'failed' | 'cancelled' | string
  phase: string
  currentRound?: number
  createdAt: number
  updatedAt: number
  request: PantheonStartRequest
  participants: PantheonParticipant[]
  events: PantheonEvent[]
  contributions: PantheonContribution[]
  verdict?: PantheonVerdict
  signoff?: PantheonSignOff
  resultPath?: string
  warnings?: string[]
  error?: string
}

export interface PantheonSessionSummary {
  id: string
  title: string
  workspace?: string
  status: string
  phase: string
  createdAt: number
  updatedAt: number
  agentCount: number
  verdictSummary?: string
  error?: string
}
