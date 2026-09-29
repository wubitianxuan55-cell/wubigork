// sin/storyText.ts — 故事正文解析（插图标记 ↔ 分段/映射），纯函数可单测。
//
// 契约：标记 `@@插图|画面描述@@` 独占一行；一条消息内按出现次序编号（0 起），
// 该编号即插图映射键（Go 侧 sinCueKey / SinIllustrate 的 cue 参数）。

import type { SinMessageView, SinToolArtifactView, SinToolTrace, SinToolTraceView, StorySegment } from './types'
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
    if (typeof v === 'string' && v) {
      out[k] = v
      continue
    }
    // v4.428 值形态升级：{path, caption} 对象取 path（历史字符串值照旧）
    if (v && typeof v === 'object') {
      const p = (v as Record<string, unknown>).path
      if (typeof p === 'string' && p) out[k] = p
    }
  }
  return out
}

/** 解析 extra 里的插图 caption（v4.428 起随图落库；历史字符串值没有——画廊回退正文/轨迹反解）。 */
export function parseIllustrationCaptions(extra: unknown): Record<string, string> {
  const out: Record<string, string> = {}
  let obj: unknown = extra
  if (typeof extra === 'string') {
    const raw = extra.trim()
    if (!raw) return out
    try {
      obj = JSON.parse(raw)
    } catch {
      return out
    }
  }
  if (!obj || typeof obj !== 'object') return out
  const arts = (obj as Record<string, unknown>).illustrations
  if (!arts || typeof arts !== 'object') return out
  for (const [k, v] of Object.entries(arts as Record<string, unknown>)) {
    if (v && typeof v === 'object') {
      const c = (v as Record<string, unknown>).caption
      if (typeof c === 'string' && c) out[k] = c
    }
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

/** 该条回复是否被用户手动停止（extra.cancelled，后端取消落库时写入）。 */
export function parseCancelled(extra: unknown): boolean {
  let obj: unknown = extra
  if (typeof extra === 'string') {
    const raw = extra.trim()
    if (!raw) return false
    try {
      obj = JSON.parse(raw)
    } catch {
      return false
    }
  }
  if (!obj || typeof obj !== 'object') return false
  return (obj as Record<string, unknown>).cancelled === true
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
      artifacts: parseArtifacts(t.artifacts),
    })
  }
  return out
}

/** 轨迹单条的产物列表容错解析（畸形条目跳过，不抛）。 */
function parseArtifacts(raw: unknown): SinToolArtifactView[] {
  if (!Array.isArray(raw)) return []
  const out: SinToolArtifactView[] = []
  for (const a of raw) {
    if (!a || typeof a !== 'object') continue
    const o = a as Record<string, unknown>
    if (typeof o.path !== 'string' || !o.path) continue
    out.push({
      kind: typeof o.kind === 'string' ? o.kind : 'image',
      path: o.path,
      caption: typeof o.caption === 'string' ? o.caption : undefined,
    })
  }
  return out
}

/** 落库轨迹 → 过程卡视图：有 error 即失败，否则完成（历史消息没有 running 态）。
 *  artifacts 归一为数组（流式帧/落库轨迹/测试夹具都可能缺省）。 */
export function toToolViews(traces: SinToolTrace[]): SinToolTraceView[] {
  return traces.map((t) => ({ ...t, artifacts: t.artifacts ?? [], status: t.error ? 'failed' : 'done' }))
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

/** 插图画廊条目（右栏面板用）：extra.illustrations 的路径 + 从正文反解的描述；
 *  messageId/cue 是「重新生成」的回写定位（SinIllustrate 按 messageId+cue 覆盖写）。 */
export interface SinGalleryItem {
  key: string
  messageId: number
  cue: string
  path: string
  prompt: string
}

/** cue 排序：数字键（正文标记 0..N）升序在前，toolN 键（工具图）按序号随后；
 *  Go map 序列化是字典序（"10"<"2"），≥11 张时画廊会错序——这里数值化归位。
 *  返回 [类序, 类内序]：toolN 是独立类，不与正文标记的数字 cue 混排。 */
function cueOrder(cue: string): [number, number] {
  const plain = /^(\d+)$/.exec(cue)
  if (plain) return [0, Number(plain[1])]
  const tool = /^tool(\d+)$/.exec(cue)
  if (tool) return [1, Number(tool[1])]
  return [2, Number.MAX_SAFE_INTEGER]
}

// ── 每消息画廊解析缓存 ──
// SinSidePanel 的 gallery useMemo 依赖整个 messages 数组，流式期间每个 delta
// 都产出新数组——不缓存的话长篇每帧都对全部历史消息重跑 parseStorySegments
//（O(全故事字数)/delta，流式期最大热点）。签名 = 影响画廊产物的最小字段集。
const galleryCache = new Map<string, { sig: string; items: SinGalleryItem[] }>()

function gallerySignature(m: Pick<SinMessageView, 'key' | 'messageId' | 'content' | 'illustrations' | 'illustrationCaptions'>): string {
  const ills = Object.entries(m.illustrations ?? {}).map(([k, v]) => `${k}=${v}`).join(',')
  const caps = Object.entries(m.illustrationCaptions ?? {}).map(([k, v]) => `${k}=${v}`).join(',')
  return `${m.messageId}|${m.content.length}|${ills}|${caps}`
}

/** 汇总一个故事的全部插图（消息序即时间序；同一消息内按 cue 序号）。
 *  描述优先级：落库 caption（v4.428 随图写入）> 正文标记反解 > 轨迹产物反解。 */
export function collectIllustrations(
  messages: Array<Pick<SinMessageView, 'key' | 'messageId' | 'content' | 'illustrations' | 'illustrationCaptions' | 'tools'>>,
): SinGalleryItem[] {
  const out: SinGalleryItem[] = []
  const liveKeys = new Set<string>()
  for (const m of messages) {
    liveKeys.add(m.key)
    const sig = gallerySignature(m)
    const hit = galleryCache.get(m.key)
    let items: SinGalleryItem[]
    if (hit && hit.sig === sig) {
      items = hit.items
    } else {
      const entries = Object.entries(m.illustrations ?? {})
        .filter(([, path]) => !!path)
        .sort((a, b) => {
          const [ca, na] = cueOrder(a[0])
          const [cb, nb] = cueOrder(b[0])
          return ca - cb || na - nb || (a[0] < b[0] ? -1 : a[0] > b[0] ? 1 : 0)
        })
      if (entries.length === 0) {
        items = []
      } else {
        const promptByCue = new Map<string, string>()
        for (const s of parseStorySegments(m.content)) {
          if (s.kind === 'illustration') promptByCue.set(s.cueKey, s.prompt)
        }
        // 工具图（toolN）正文里没有标记，描述从轨迹产物反解（按调用序对号）
        let toolIdx = 0
        for (const t of m.tools ?? []) {
          for (const art of t.artifacts ?? []) {
            promptByCue.set(`tool${toolIdx}`, art.caption || '')
            toolIdx++
          }
        }
        const capsByCue = m.illustrationCaptions ?? {}
        items = entries.map(([cue, path]) => ({
          key: `${m.key}:${cue}`,
          messageId: m.messageId,
          cue,
          path,
          prompt: capsByCue[cue] || promptByCue.get(cue) || '',
        }))
      }
      galleryCache.set(m.key, { sig, items })
    }
    out.push(...items)
  }
  // 只保留当前故事的消息缓存（切故事后旧键不驻留）
  for (const k of galleryCache.keys()) {
    if (!liveKeys.has(k)) galleryCache.delete(k)
  }
  return out
}

/** 故事标题建议：取首条用户消息前 16 字，空则「新故事」。 */
export function suggestStoryTitle(firstUserText: string): string {
  const t = (firstUserText ?? '').trim().replace(/\s+/g, ' ')
  if (!t) return '新故事'
  const r = Array.from(t)
  return r.length > 16 ? r.slice(0, 16).join('') : t
}
