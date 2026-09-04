/**
 * create-chapter-stream 流式事件类型（T6-7.2 / T6-7.5）。
 *
 * 契约对齐 internal/app/create_chapter_handler.go 的事件负载：
 *  - phase    阶段提示（writing / continuing 续写）
 *  - chunk    正文增量
 *  - done     生成完成（后端已落盘）
 *  - error    生成失败
 *  - cancelled 生成被取消（后端批 1 新增：取消时已把部分正文落盘，
 *              事件携带 content；仅部分为空或落盘失败时不携带）
 *
 * 旧实现用 (data as any) 做字符串 switch，无任何类型保障；这里收敛为
 * 判别联合（discriminated union），负载先经 parseCreateChapterEvent 校验
 * 再分发，未知/畸形负载返回 null 被忽略（与旧 switch 无 default 一致）。
 */

export interface CreateChapterPhaseEvent {
  type: 'phase'
  phase: string
  attempt?: number
  current?: number
  target?: number
}

export interface CreateChapterChunkEvent {
  type: 'chunk'
  content: string
  total?: number
}

/** 生成完成时 novelstyle 输出的 AI 味检测结果（0-100，越高越 AI 味）。 */
export interface AiTasteIssue {
  start: number
  end: number
  reason: string
  severity: string
  suggestion?: string
}
export interface AiTasteResult {
  score: number
  issues: AiTasteIssue[]
  /** story-deslop 确定性定点重写结果（仅当生成时启用且有改善时存在）。 */
  deSlop?: {
    beforeScore?: number
    afterScore?: number
    changes?: unknown[]
    punctFixed?: number
  }
}

export interface CreateChapterDoneEvent {
  type: 'done'
  chapterNum?: number
  branch?: string
  content?: string
  summary?: string
  nodeId?: string
  total?: number
  aiTaste?: AiTasteResult
}

export interface CreateChapterErrorEvent {
  type: 'error'
  error?: string
}

export interface CreateChapterCancelledEvent {
  type: 'cancelled'
  chapterNum?: number
  branch?: string
  nodeId?: string
  total?: number
  content?: string
}

export type CreateChapterStreamEvent =
  | CreateChapterPhaseEvent
  | CreateChapterChunkEvent
  | CreateChapterDoneEvent
  | CreateChapterErrorEvent
  | CreateChapterCancelledEvent

const STREAM_EVENT_TYPES = new Set<string>(['phase', 'chunk', 'done', 'error', 'cancelled'])

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function num(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function str(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined
}

/** 校验 novelstyle 返回的 AI 味结果；畸形/缺 score 时返回 undefined（不阻断 done）。 */
function parseAiTaste(value: unknown): AiTasteResult | undefined {
  if (!isRecord(value)) return undefined
  const score = num(value.score)
  if (score === undefined) return undefined
  const issues = Array.isArray(value.issues) ? value.issues.filter(isRecord) : []
  let deSlop: AiTasteResult['deSlop'] | undefined
  if (isRecord(value.deSlop)) {
    deSlop = {
      beforeScore: num(value.deSlop.beforeScore),
      afterScore: num(value.deSlop.afterScore),
      punctFixed: num(value.deSlop.punctFixed),
      changes: Array.isArray(value.deSlop.changes) ? value.deSlop.changes : [],
    }
  }
  return {
    score,
    issues: issues.map((i) => ({
      start: num(i.start) ?? 0,
      end: num(i.end) ?? 0,
      reason: str(i.reason) ?? '',
      severity: str(i.severity) ?? 'info',
      suggestion: str(i.suggestion),
    })),
    deSlop,
  }
}

/** 把任意事件负载解析为受校验的判别联合事件；非流事件/畸形负载返回 null。 */
export function parseCreateChapterEvent(payload: unknown): CreateChapterStreamEvent | null {
  if (!isRecord(payload)) return null
  const type = payload.type
  if (typeof type !== 'string' || !STREAM_EVENT_TYPES.has(type)) return null
  switch (type) {
    case 'phase':
      // phase 事件必须有 phase 字符串，否则视为畸形负载
      return typeof payload.phase === 'string'
        ? {
            type,
            phase: payload.phase,
            attempt: num(payload.attempt),
            current: num(payload.current),
            target: num(payload.target),
          }
        : null
    case 'chunk':
      // chunk 事件必须携带正文增量
      return typeof payload.content === 'string'
        ? { type, content: payload.content, total: num(payload.total) }
        : null
    case 'done':
      return {
        type,
        chapterNum: num(payload.chapterNum),
        branch: str(payload.branch),
        content: str(payload.content),
        summary: str(payload.summary),
        nodeId: str(payload.nodeId),
        total: num(payload.total),
        aiTaste: parseAiTaste(payload.aiTaste),
      }
    case 'error':
      return { type, error: str(payload.error) }
    case 'cancelled':
      return {
        type,
        chapterNum: num(payload.chapterNum),
        branch: str(payload.branch),
        nodeId: str(payload.nodeId),
        total: num(payload.total),
        content: str(payload.content),
      }
    default:
      return null
  }
}
