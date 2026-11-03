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
  /** 每轮之间停下，开一道决策门等人确认（O-E） */
  gateEachRound?: boolean
}

/** 席位向用户提的问题（O-E）。回答只能绑 QuestionID，答一次就作废。 */
export interface PantheonQuestion {
  questionId: string
  options?: string[]
  askedBy?: string
  at?: number
}

export interface PantheonAnswer {
  questionId: string
  body?: string
  by?: string
  at?: number
}

/** 决策门：open 开、resolve 放行、dismiss 放弃（放弃必须写理由）。 */
export interface PantheonGate {
  action: 'open' | 'resolve' | 'dismiss' | string
  reason?: string
}

export interface PantheonCallRef {
  callId?: string
  attemptId?: string
  retryOfAttempt?: string
  attempt?: number
}

/** 席位观测：这条状态是谁在什么时候看见的（Orca agent-status 的圆桌版）。 */
export interface PantheonObserved {
  origin?: string
  authorityId?: string
  incarnation?: number
  revision?: number
  at?: number
}

export interface PantheonParticipant extends PantheonCallRef {
  agentId: string
  name: string
  role: string
  status: string
  startedAt?: number
  finishedAt?: number
  durationMs?: number
  error?: string
  /** 一句话证据：这个状态是凭什么来的。 */
  evidence?: string
  observed?: PantheonObserved
}

export interface PantheonEvent extends PantheonCallRef {
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
  question?: PantheonQuestion
  answer?: PantheonAnswer
  gate?: PantheonGate
}

export interface PantheonLiveStream extends PantheonCallRef {
  id: string
  agentId: string
  phase: string
  round: number
  text: string
  streaming: boolean
  startedAt?: number
  updatedAt?: number
}

export interface PantheonContribution extends PantheonCallRef {
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
  /** 签注时确认的缺口：同样的小结，缺口不同就不是同一个版本（O-H） */
  gaps?: string[]
}

export interface PantheonSession {
  id: string
  title: string
  status: 'running' | 'completed' | 'failed' | 'cancelled' | 'waiting_input' | 'interrupted' | string
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
  /** 结案时没交出可用结论的席位（具名 + 原因） */
  gaps?: string[]
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
