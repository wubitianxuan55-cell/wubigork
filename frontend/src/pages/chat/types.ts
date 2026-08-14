// ChatPage 拆分产物：共享类型（行为零变化，T6-10.1）
export interface Personality {
  id: string; label: string; gender: string; tags?: string[]
  dims: { T: number; I: number; S: number; O: number; R: number }
  voiceGuide?: string; requiresAdult18?: boolean
}

export interface ChatMsg {
  key: string
  role: 'user' | 'assistant'
  content: string
  reasoning?: string
  createdAt: string
  streaming?: boolean
  error?: boolean
  extra?: Record<string, unknown>
}

export interface LegacyMsg { id?: string; role: string; content: string; timestamp?: number }
export interface LegacyTopic { id?: string; title?: string; messages?: LegacyMsg[]; createdAt?: number }
