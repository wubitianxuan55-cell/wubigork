/**
 * gaea 共享类型定义 — 桌面端 (frontend) + 移动端 (mobile) 共用
 */

// ── 项目 ────────────────────────────────────────────────
export interface ProjectMeta {
  id: string
  title: string
  author?: string
  summary?: string
  cover_url?: string
  created_at: number
  updated_at: number
}

// ── 大纲节点 ────────────────────────────────────────────
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
  severity: 'error' | 'warning' | 'info'
  section: string
  description: string
  suggestion: string
}

export interface ConsistencyReportData {
  issues: ConsistencyIssueData[]
  overall_note: string
}

// ── 章节 ────────────────────────────────────────────────
export interface ChapterData {
  num: number
  title: string
  content: string
  summary: string
  scenes: string[]
  keyEvents: string[]
  emotionTone: string
  wordCount: number
}

// ── XAI API ─────────────────────────────────────────────
export interface XAIChatMessage {
  role: 'system' | 'user' | 'assistant'
  content: string
}

export interface XAIChatRequest {
  model?: string
  messages: XAIChatMessage[]
  stream?: boolean
  temperature?: number
  max_tokens?: number
}

export interface XAIChatResponse {
  id: string
  choices: {
    index: number
    message: { role: string; content: string }
    finish_reason: string
  }[]
  usage?: { prompt_tokens: number; completion_tokens: number }
}

export interface XAIStreamChunk {
  type: 'chunk' | 'done' | 'error'
  content?: string
  finishReason?: string
  error?: string
}

export interface XAIImageRequest {
  prompt: string
  model?: string
  n?: number
  size?: string
}

export interface XAIImageResponse {
  data: { url: string; revised_prompt?: string }[]
}

// ── 同步 ────────────────────────────────────────────────
export interface SyncPayload {
  timestamp: number
  projects?: any[]
  chapters?: any[]
}

export interface TaskInfo {
  id: string
  type: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  prompt?: string
  resultUrl?: string
  error?: string
  createdAt: number
}
