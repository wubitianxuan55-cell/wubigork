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
  /** 本轮工具调用轨迹（流式累积，落库后由 extra.tools 还原）。 */
  tools?: SinToolTraceView[]
  createdAt?: string
}

/**
 * 工具调用轨迹：`done.tools` 与消息 `extra.tools` 同一形态（后端 sinToolTrace
 * 的 JSON 标签逐字对应，snake_case——不要按办公 wire 的 camelCase 解析）。
 */
export interface SinToolTrace {
  id?: string
  name?: string
  args?: string
  output?: string
  error?: string
  elapsed_ms?: number
  read_only?: boolean
}

/** 过程卡视图：轨迹 + 运行期状态（dispatch 到达 → running；result 到达 → 终态）。 */
export interface SinToolTraceView extends SinToolTrace {
  status: 'running' | 'done' | 'failed'
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
  /** tool_dispatch / tool_result / notice 帧的载荷（见后端 §13.3 线格式）。 */
  id?: string
  name?: string
  args?: string
  read_only?: boolean
  output?: string
  elapsed_ms?: number
  message?: string
  /** done 帧携带的本轮工具轨迹（权威值：覆盖流式累积结果）。 */
  tools?: SinToolTrace[]
  answered_by?: { engine?: string; model?: string; source?: string; cost_cny?: number }
}
