// sin/types.ts — 原罪板块前端视图模型（闲庭 · 图文故事创作）。
//
// 后端契约：话题/消息复用统一聊天存储（mode=sin），消息 extra 里带
// illustrations 映射（cue 键 → 本地图片路径）；正文里的插图标记形如
// `@@插图|画面描述@@`，解析规则与 Go 侧 sin_prompt.go / sin_handler.go 同源。

/** 故事（话题）视图：字段对齐 Go chat.Topic 的展示面。 */
export interface SinStoryView {
  id: string
  title: string
  createdAt: string
  updatedAt: string
  preview: string
}

/** 故事消息视图：messageId <= 0 = 尚未落库的流式占位消息。 */
export interface SinMessageView {
  key: string
  messageId: number
  role: 'user' | 'assistant'
  content: string
  streaming?: boolean
  error?: boolean
  /** cue 键（出现次序，0 起）→ 插图本地路径。 */
  illustrations: Record<string, string>
  reasoning?: string
  createdAt?: string
}

/** 正文分段：纯文本段（交办公 Markdown 渲染）/ 插图段（就地生成渲染）。 */
export type StorySegment =
  | { kind: 'text'; text: string }
  | { kind: 'illustration'; cueKey: string; prompt: string }

/** sin-stream:<runID> 事件载荷（最小消费面，与 Go emit 逐字段对齐）。 */
export interface SinStreamPayload {
  type?: string
  content?: string
  reply?: string
  reasoning?: string
  error?: string
  message_id?: number
  answered_by?: { engine?: string; model?: string; source?: string; cost_cny?: number }
}
