// sin/storyText.ts — 故事正文解析（插图标记 ↔ 分段/映射），纯函数可单测。
//
// 契约：标记 `@@插图|画面描述@@` 独占一行；一条消息内按出现次序编号（0 起），
// 该编号即插图映射键（Go 侧 sinCueKey / SinIllustrate 的 cue 参数）。

import type { SinToolTrace, SinToolTraceView, StorySegment } from './types'

/** 插图标记定界符（与 Go 侧 sin_prompt.go 常量逐字一致）。 */
export const SIN_CUE_OPEN = '@@插图|'
export const SIN_CUE_CLOSE = '@@'

/** 插图映射键：出现次序（0 起）——跨端唯一约定，改动需同时改 Go sinCueKey。 */
export function sinCueKey(index: number): string {
  return String(index)
}

/**
 * 把故事正文切成 文本段 / 插图段。未闭合的标记按普通文本处理（流式中间态
 * 常见：标记只写了一半，此时不该弹出插图位）。
 */
export function parseStorySegments(content: string): StorySegment[] {
  const out: StorySegment[] = []
  let rest = content ?? ''
  let cueIndex = 0
  for (;;) {
    const start = rest.indexOf(SIN_CUE_OPEN)
    if (start < 0) {
      if (rest) out.push({ kind: 'text', text: rest })
      break
    }
    const end = rest.indexOf(SIN_CUE_CLOSE, start + SIN_CUE_OPEN.length)
    if (end < 0) {
      out.push({ kind: 'text', text: rest })
      break
    }
    const before = rest.slice(0, start)
    if (before.trim()) out.push({ kind: 'text', text: before })
    const prompt = rest.slice(start + SIN_CUE_OPEN.length, end).trim()
    out.push({ kind: 'illustration', cueKey: sinCueKey(cueIndex), prompt })
    cueIndex += 1
    rest = rest.slice(end + SIN_CUE_CLOSE.length)
  }
  return out
}

/** 解析消息 extra 里的插图映射（extra 可能是 JSON 字符串或已解析对象）。 */
export function parseIllustrations(extra: unknown): Record<string, string> {
  const out: Record<string, string> = {}
  let obj: unknown = extra
  if (typeof extra === 'string') {
    const raw = extra.trim()
    if (!raw) return out
    try {
      obj = JSON.parse(raw)
    } catch {
      return out // 坏 JSON 按无插图继续（辅助映射容错）
    }
  }
  if (!obj || typeof obj !== 'object') return out
  const arts = (obj as Record<string, unknown>).illustrations
  if (!arts || typeof arts !== 'object') return out
  for (const [k, v] of Object.entries(arts as Record<string, unknown>)) {
    if (typeof v === 'string' && v) out[k] = v
  }
  return out
}

/** 从 extra 里取 reasoning 文本（后端落库的 reasoning 字段）。 */
export function parseReasoning(extra: unknown): string {
  let obj: unknown = extra
  if (typeof extra === 'string') {
    const raw = extra.trim()
    if (!raw) return ''
    try {
      obj = JSON.parse(raw)
    } catch {
      return ''
    }
  }
  if (!obj || typeof obj !== 'object') return ''
  const r = (obj as Record<string, unknown>).reasoning
  return typeof r === 'string' ? r : ''
}

/**
 * 从 extra 里取工具轨迹（后端落库的 tools 字段）。
 * 容错口径与 parseIllustrations 同源：坏 JSON / 缺字段 / 单条畸形一律跳过，
 * 绝不抛——过程卡是装饰面，不能让它拖垮故事渲染。
 */
export function parseTools(extra: unknown): SinToolTrace[] {
  let obj: unknown = extra
  if (typeof extra === 'string') {
    const raw = extra.trim()
    if (!raw) return []
    try {
      obj = JSON.parse(raw)
    } catch {
      return []
    }
  }
  if (!obj || typeof obj !== 'object') return []
  const list = (obj as Record<string, unknown>).tools
  if (!Array.isArray(list)) return []
  const out: SinToolTrace[] = []
  for (const raw of list) {
    if (!raw || typeof raw !== 'object') continue
    const t = raw as Record<string, unknown>
    const name = typeof t.name === 'string' ? t.name : ''
    if (!name) continue // 无名字的行渲染不出信息，丢弃
    out.push({
      id: typeof t.id === 'string' ? t.id : '',
      name,
      args: typeof t.args === 'string' ? t.args : '',
      output: typeof t.output === 'string' ? t.output : '',
      error: typeof t.error === 'string' ? t.error : '',
      elapsed_ms: typeof t.elapsed_ms === 'number' ? t.elapsed_ms : 0,
      read_only: t.read_only === true,
    })
  }
  return out
}

/** 落库轨迹 → 过程卡视图：有 error 即失败，否则完成（历史消息没有 running 态）。 */
export function toToolViews(traces: SinToolTrace[]): SinToolTraceView[] {
  return traces.map((t) => ({ ...t, status: t.error ? 'failed' : 'done' }))
}

/** 该条消息里还没有插图产物的插图段（供自动生成排队）。 */
export function pendingIllustrations(
  segments: StorySegment[],
  illustrations: Record<string, string>,
): Array<{ cueKey: string; prompt: string }> {
  return segments
    .filter((s): s is Extract<StorySegment, { kind: 'illustration' }> => s.kind === 'illustration')
    .filter((s) => !illustrations[s.cueKey] && s.prompt !== '')
    .map((s) => ({ cueKey: s.cueKey, prompt: s.prompt }))
}

/** 无标记的历史正文（用户消息展示用）：把标记折成简短的「（插图）」提示。 */
export function stripCuesForDisplay(content: string): string {
  return content.split(SIN_CUE_OPEN).join('（插图：').split(SIN_CUE_CLOSE).join('）')
}

/** 故事标题建议：取首条用户消息前 16 字，空则「新故事」。 */
export function suggestStoryTitle(firstUserText: string): string {
  const t = (firstUserText ?? '').trim().replace(/\s+/g, ' ')
  if (!t) return '新故事'
  const r = Array.from(t)
  return r.length > 16 ? r.slice(0, 16).join('') : t
}
