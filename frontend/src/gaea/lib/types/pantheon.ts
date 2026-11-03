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
  /** 议事形态：roundtable=圆桌会诊（默认）；targeted_challenge=定向质询 */
  preset?: 'roundtable' | 'targeted_challenge' | string
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
  /** 开场探针时的 Harness 版本；Grok 席位记录模型名。 */
  version?: string
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
  /** DID 续跑事件消费的核验动作指纹 */
  didKeys?: string[]
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
  residual?: string
  residualEvidence?: string
  residualAgainst?: string
  residualFlipCondition?: string
  residualVerification?: string
  /** 讨论动作：claim / challenge / respond / revise / reinforce / broaden */
  discussionAct?: string
  /** 质疑、回应或修正所指向的前文贡献 ID */
  targetContributionId?: string
  /** 同一目标贡献存在多条分歧时，回应所指向的台账 ID */
  targetIssueId?: string
  challengedClaim?: string
  evidenceHandle?: string
  /** 回应既有质疑：accept / partial / reject */
  responseVerdict?: string
  runMode?: string
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
  issueReviews?: PantheonIssueReview[]
  proposedActions?: PantheonProposedAction[]
  revisedFromSha256?: string
  revisionReason?: string
  chairman: string
  partial: boolean
  createdAt: number
}

export interface PantheonAttestation {
  agentId: string
  agentName: string
  position: 'support' | 'conditional' | 'oppose' | 'unknown' | string
  reason?: string
  conditions?: string[]
  issueIds?: string[]
  verdictSha256?: string
  error?: string
  createdAt: number
}

export interface PantheonVerdictRevision {
  verdict: PantheonVerdict
  sha256: string
  reason: string
  at: number
}

export interface PantheonProposedAction {
  id?: string
  action: string
  rationale?: string
  dependsOn?: string[]
  acceptanceEvidence?: string
  suggestedOwner?: string
  risk?: string
}

export interface PantheonIssueReview {
  issueId: string
  outcome: 'resolved' | 'retain' | 'user_decision' | 'needs_verification' | string
  reason: string
  evidence?: string
}

export interface PantheonSignOff {
  decision: 'accepted' | 'needs_revision' | 'rejected' | string
  note?: string
  at: number
  /** 签注时确认的缺口：同样的小结，缺口不同就不是同一个版本（O-H） */
  gaps?: string[]
}

export interface PantheonIssue {
  id: string
  round: number
  agentId: string
  agentName: string
  summary: string
  /** challenge=定向质疑开条；residual=剩余分歧兜底开条；user=用户插话 */
  origin?: 'challenge' | 'residual' | 'user' | string
  targetContributionId?: string
  challengedClaim?: string
  claimAgentIds?: string[]
  claimAgentNames?: string[]
  dissentAgentIds?: string[]
  dissentAgentNames?: string[]
  counterpointIds?: string[]
  /** 支持保留异议的证据句柄、文件、命令或显式“未核验” */
  evidence?: string
  /** 出现什么新证据时，提出者承诺重新判断 */
  flipCondition?: string
  /** 下一项可执行、能减少不确定性的核验动作 */
  verification?: string
  responseVerdict?: 'accept' | 'partial' | 'reject' | string
  responseEvidence?: string
  responseAgentIds?: string[]
  responseAgentNames?: string[]
  responseContributionIds?: string[]
  /** 该分歧是否实际阻塞过收口 */
  blocking?: boolean
  /** 台账去向：等待回应、撤回轮次或结案保留 */
  disposition?: string
  /** 撤回时引用的证据；空值表示翻转未举证 */
  resolutionEvidence?: string
  /** 同一分歧最近一次仍被保留的轮次 */
  lastSeenIn?: number
  /** open=仍待回应 / resolved=提出者已撤回 / dissent=结案仍保留 */
  status: 'open' | 'resolved' | 'dissent' | string
  resolvedAt?: number
  resolvedIn?: number
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
  /** 分歧台账：从「剩余分歧」协议行自动开条，逐轮保留去向 */
  issues?: PantheonIssue[]
  attestations?: PantheonAttestation[]
  verdictRevisions?: PantheonVerdictRevision[]
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
  issueCount?: number
  openIssueCount?: number
  dissentCount?: number
  error?: string
}

export interface PantheonEvaluationRun {
  sessionId: string
  caseKey: string
  title: string
  variant: string
  status: string
  protocol: string
  preset: string
  agentIds?: string[]
  runtimeVersions?: Record<string, string>
  participantCount: number
  versionedParticipants: number
  expectedRounds: number
  actualRounds: number
  callCount: number
  retryCount: number
  errorCount: number
  toolCallCount: number
  wallDurationMs: number
  agentDurationMs: number
  gapCount: number
  discussionCount: number
  evidenceHandledCount: number
  unverifiedMentions: number
  userNoteCount: number
  issueCount: number
  resolvedIssueCount: number
  dissentIssueCount: number
  attestationCount: number
  unknownAttestations: number
  verdictPartial?: boolean
  verdictRevised?: boolean
  signOffDecision?: string
  runModes?: string[]
  createdAt: number
}

export interface PantheonEvaluationVariantSummary {
  variant: string
  sessions: number
  completedSessions: number
  signedOffSessions: number
  acceptedSessions: number
  avgRounds?: number
  avgWallDurationMs?: number
  avgCalls?: number
  gapSessions?: number
  dissentIssues?: number
  revisedVerdicts?: number
}

export interface PantheonEvaluationCaseCoverage {
  key: string
  title: string
  variants: string[]
  completedVariants?: string[]
  missingVariants?: string[]
}

export interface PantheonEvaluationReport {
  generatedAt: number
  requiredCases: number
  caseCount: number
  qualifiedCases: number
  requiredVariants: string[]
  unsupportedVariants?: string[]
  gateReady: boolean
  variants: PantheonEvaluationVariantSummary[]
  cases: PantheonEvaluationCaseCoverage[]
  runs: PantheonEvaluationRun[]
  warnings?: string[]
}
