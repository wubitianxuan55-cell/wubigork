export interface ChatSearchReference {
  title: string
  url: string
  source?: string
  snippet?: string
}

export interface ChatSearchSummary {
  state: string
  queries: string[]
  searches: number
  rounds: number
  sources: number
  elapsedMs: number
  degradedReason: string
  references: ChatSearchReference[]
}

function finiteNumber(value: unknown, fallback = 0): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

export function parseChatSearchSummary(value: unknown): ChatSearchSummary | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  const raw = value as Record<string, unknown>
  const state = stringValue(raw.state)
  if (!state) return null
  const queries = Array.isArray(raw.queries)
    ? raw.queries.map(stringValue).filter(Boolean).slice(0, 4)
    : []
  const references = Array.isArray(raw.references)
    ? raw.references.flatMap((item): ChatSearchReference[] => {
      if (!item || typeof item !== 'object' || Array.isArray(item)) return []
      const ref = item as Record<string, unknown>
      const url = stringValue(ref.url)
      if (!url) return []
      return [{
        title: stringValue(ref.title),
        url,
        source: stringValue(ref.source) || undefined,
        snippet: stringValue(ref.snippet) || undefined,
      }]
    }).slice(0, 8)
    : []
  return {
    state,
    queries,
    searches: finiteNumber(raw.searches),
    rounds: finiteNumber(raw.rounds),
    sources: finiteNumber(raw.sources, references.length),
    elapsedMs: finiteNumber(raw.elapsed_ms),
    degradedReason: stringValue(raw.degraded_reason),
    references,
  }
}
