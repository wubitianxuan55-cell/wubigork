// sin/sinToolMeta.ts — 过程卡的纯展示映射（工具名 → 标签/图标/摘要）。
//
// 与组件分文件：组件文件只导出组件（react-refresh 的 only-export-components
// 纪律），纯函数另放一处便于单测与复用。
import { BookOpen, FileText, Globe, ListTree, Users } from '../../gaea/icons'
import { subjectOf } from '../../gaea/lib/tools'
import type { SinToolTraceView } from './types'

/** 工具名 → 中文标签（未知工具如实显示原名，不猜）。 */
export const TOOL_LABELS: Record<string, string> = {
  web_search: '联网搜索',
  web_fetch: '抓取网页',
  sin_cast: '查角色卡',
  sin_notes: '故事便签',
  sin_outline: '故事大纲',
}

export const TOOL_ICONS = {
  web_search: Globe,
  web_fetch: FileText,
  sin_cast: Users,
  sin_notes: BookOpen,
  sin_outline: ListTree,
} as const

const ACTION_LABELS: Record<string, string> = {
  list: '列出', read: '读取', write: '写入', set: '更新', delete: '删除',
}

export function sinToolLabel(name: string): string {
  return TOOL_LABELS[name] ?? name ?? ''
}

export function parseToolArgs(args: string | undefined): Record<string, unknown> {
  if (!args) return {}
  try {
    const v = JSON.parse(args) as unknown
    return v && typeof v === 'object' ? (v as Record<string, unknown>) : {}
  } catch {
    return {}
  }
}

function clip(s: string, max: number): string {
  const r = Array.from(s.trim().replace(/\s+/g, ' '))
  return r.length > max ? `${r.slice(0, max).join('')}…` : r.join('')
}

export function fmtToolElapsed(ms: number | undefined): string {
  if (!ms || ms <= 0) return ''
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`
}

/**
 * 行首摘要：让折叠行一眼看出这次调用了什么。
 * 联网两件套复用办公的 subjectOf（它按 args 取 query/url），其余按原罪工具语义取。
 */
export function sinToolSubject(t: SinToolTraceView): string {
  const args = t.args ?? ''
  const a = parseToolArgs(args)
  const str = (k: string) => (typeof a[k] === 'string' ? (a[k] as string) : '')
  if (t.name === 'sin_cast') return str('name') || '全部角色'
  if (t.name === 'sin_notes' || t.name === 'sin_outline') {
    const action = ACTION_LABELS[str('action')] ?? str('action')
    const content = str('content')
    return content ? `${action} · ${clip(content, 24)}` : action
  }
  const office = subjectOf(t.name ?? '', args)
  if (office) return clip(office, 48)
  return clip(str('query') || str('url') || str('content') || '', 48)
}

/** 头部摘要：按工具分组计数（「联网搜索 ×2 · 故事便签」），最多 3 组。 */
export function sinToolSummary(tools: SinToolTraceView[]): string {
  if (tools.length === 0) return ''
  const counts = new Map<string, number>()
  for (const t of tools) {
    const label = sinToolLabel(t.name ?? '')
    counts.set(label, (counts.get(label) ?? 0) + 1)
  }
  const parts: string[] = []
  for (const [label, n] of counts) {
    if (parts.length >= 3) {
      parts.push('等')
      break
    }
    parts.push(n > 1 ? `${label} ×${n}` : label)
  }
  return parts.join(' · ')
}
