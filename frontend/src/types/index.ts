/** gaea 共享前端类型 */

// ── 大纲 ────────────────────────────────────────────────
export interface OutlineNode {
  id: string
  parent_id?: string
  title: string
  summary: string
  status: string
  order_index: number
  key_points?: string[]
  characters?: string[]
  scene_ideas?: string[]
  emotion?: string
  chapter_file?: string
  branch?: string       // 分支字母: "a"/"b"/"c"
  children?: OutlineNode[]
}

// ── 角色 ────────────────────────────────────────────────
export interface CharacterData {
  id: string
  name: string
  role_type: string
  gender: string
  age: string
  personality: string
  background: string
  appearance: string
  figure: string
  motivation: string
  arc: string
  status: string
  portrait_url?: string
}

export interface OrganizationData {
  id: string
  name: string
  type: string
  description: string
  power_level: string
  location?: string
  motto?: string
  members?: string[]
}

export interface RelationshipData {
  from_id: string
  to_id: string
  relation_type: string
  description: string
  intimacy: number
}

// ── 世界观 ──────────────────────────────────────────────
export interface WorldviewSectionData {
  id: string
  title: string
  content: string
  order: number
}

export interface ConsistencyIssueData {
  severity: string // error | warning | info
  section: string
  description: string
  suggestion: string
}

export interface ConsistencyReportData {
  issues: ConsistencyIssueData[]
  overall_note: string
}

// ── 伏笔（v4.3f，对齐 internal/types.Foreshadow）────────────────
export type ForeshadowStatus = 'planted' | 'hinted' | 'revealed'

export interface ForeshadowItemData {
  id: string
  category: string // character / plot / world / relationship
  description: string
  planted_in: string // 章节文件名
  revealed_in?: string // 回收章节
  status: ForeshadowStatus
  is_long_term: boolean
}

// ── 一致性检查（v4.3f，对齐 internal/graph.ConsistencyIssue）────
export type ConsistencySeverity = 'error' | 'warning' | 'info'

export interface ConsistencyCheckIssue {
  severity: ConsistencySeverity
  category: string // attribute / timeline / status / relationship
  entity_name: string
  description: string
  location: string
  evidence: string
  suggestion: string
}

export interface ConsistencyCheckReport {
  issues: ConsistencyCheckIssue[]
  total_issues: number
  summary: string
}

// ── 一致性 AI 深检 v0（对齐 internal/app/consistency_deep_handler.go）────
export type ConsistencyIssueSource = 'rule' | 'ai'

export interface ConsistencyDeepIssue extends ConsistencyCheckIssue {
  /** 告警来源：rule=规则层 / ai=AI 状态卡跨章比对（旧规则接口无此字段） */
  source?: ConsistencyIssueSource
  /** 分支标记：''=主线，'a'/'b'/'c'=分支 */
  branch?: string
}

export interface ConsistencyDeepResult {
  issues: ConsistencyDeepIssue[]
  total_issues: number
  summary: string
  /** AI 成功提取状态卡的章数 */
  chapters_scanned: number
  /** AI 提取失败被跳过的章数 */
  chapters_failed?: number
  /** false=AI 不可用（结果仅含规则层） */
  ai_available: boolean
  /** 降级/跳过说明，正常为空 */
  ai_note?: string
}

// ── 画布 ────────────────────────────────────────────────
export interface CanvasChapterData {
  num: number
  title: string
  summary: string
  keyEvents: string[]
  characters: string[]
  emotionTone: string
  quality: number
}

// ── 写作 ────────────────────────────────────────────────
export interface ChapterTabData {
  node: OutlineNode
  chapterNum: number
  scenes: string[]
  summary: string
  keyEvents: string[]
  emotionTone: string
  saved: boolean
  generating: boolean
  streamSpeed: number
  messages: import('../components/ChatPanel').Message[]
  targetWords: number
  skillName: string
  retryStatus?: { score: number; target: number } | null
}

// ── TTS 语音朗读 ────────────────────────────────────────
export interface TTSConfig {
  modelPath: string
  serverPath: string
  port: number
  backend: string
  speed: number
}

export interface TTSStatus {
  running: boolean
  port: number
}

// ── 书架 ────────────────────────────────────────────────
/** BrainstormIdea — AI 脑暴生成的小说创意 */
export interface BrainstormIdea {
  id: number
  title: string
  pitch: string
  conflict: string
  audience: string
  tags: string[]
}
