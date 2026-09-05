// consistencyDeep.ts — 一致性 AI 深检的前端纯函数助手（与组件解耦，便于 vitest 单测）
// 对齐后端 internal/app/consistency_deep_handler.go CheckConsistencyDeep 载荷。
import type { ConsistencyDeepIssue, ConsistencyDeepResult } from '../../types'

/** AI 深检窗口边界与默认值（后端 >50 或 ≤0 夹到 50，前端先行夹取避免无谓往返） */
export const DEEP_CHAPTER_MIN = 1
export const DEEP_CHAPTER_MAX = 50
export const DEEP_CHAPTER_DEFAULT = 20

/** 夹取章数输入：非有限数回退默认 20，超界夹到 [1, 50] */
export function clampDeepChapters(n: number): number {
  if (!Number.isFinite(n)) return DEEP_CHAPTER_DEFAULT
  return Math.min(DEEP_CHAPTER_MAX, Math.max(DEEP_CHAPTER_MIN, Math.round(n)))
}

/** 最小后端载荷形状：issues 必须是数组才认为有效 */
interface DeepResultLike {
  issues?: unknown
  total_issues?: unknown
  summary?: unknown
  chapters_scanned?: unknown
  ai_available?: unknown
  ai_note?: unknown
  chapters_failed?: unknown
}

/**
 * 收窄后端载荷为 ConsistencyDeepResult；缺 issues 数组等关键结构返回 null
 * （诚实降级：坏载荷宁可显示错误也不伪造结果）。
 */
export function normalizeDeepResult(raw: unknown): ConsistencyDeepResult | null {
  const r = raw as DeepResultLike | null
  if (!r || !Array.isArray(r.issues)) return null
  return {
    issues: r.issues as ConsistencyDeepIssue[],
    total_issues: typeof r.total_issues === 'number' ? r.total_issues : (r.issues as unknown[]).length,
    summary: typeof r.summary === 'string' ? r.summary : '',
    chapters_scanned: typeof r.chapters_scanned === 'number' ? r.chapters_scanned : 0,
    chapters_failed: typeof r.chapters_failed === 'number' ? r.chapters_failed : undefined,
    ai_available: r.ai_available === true,
    ai_note: typeof r.ai_note === 'string' ? r.ai_note : '',
  }
}

/** 来源徽标样式：ai → 紫色「AI」；rule → 默认色「规则」；未知来源返回 null（不渲染徽标） */
export function sourceBadge(source?: string): { label: string; color: string } | null {
  if (source === 'ai') return { label: 'AI', color: 'purple' }
  if (source === 'rule') return { label: '规则', color: 'default' }
  return null
}

// ── 误报缓解：置信度/原因标注 → UI 三档分级 ─────────────────
// 对齐 internal/app/consistency_deep_handler.go deepIssueToMap 输出的
// confidence（high/medium/low）与 reason（缓解原因分类）两个可选字段。

/** 缓解原因分类（后端 reason 字段的合法值） */
export type DeepIssueReason = 'wording' | 'granularity' | 'alias' | 'unexplained'

/** 深检告警 UI 分级：conflict=冲突 / suspected=疑似 / hint=提示 */
export type DeepIssueLevel = 'conflict' | 'suspected' | 'hint'

/** 携带缓解标注的告警（后端 v4.101 起新增的可选字段） */
export interface DeepIssueAnnotation {
  confidence?: string
  reason?: string
}

/** 原因 → 分级映射：措辞差异一律降为「提示」，粒度/别名/缺交代降为「疑似」 */
const REASON_LEVEL: Record<DeepIssueReason, DeepIssueLevel> = {
  wording: 'hint',
  granularity: 'suspected',
  alias: 'suspected',
  unexplained: 'suspected',
}

/**
 * 告警分级：带合法缓解原因的按原因映射（降级），否则按 severity 直映射
 * （error→冲突 / warning→疑似 / 其余→提示）。分级只降不升，绝不吞掉真问题。
 */
export function deepIssueLevel(iss: ConsistencyDeepIssue & DeepIssueAnnotation): DeepIssueLevel {
  const byReason = iss.reason && iss.reason in REASON_LEVEL ? REASON_LEVEL[iss.reason as DeepIssueReason] : undefined
  if (byReason) return byReason
  if (iss.severity === 'error') return 'conflict'
  if (iss.severity === 'warning') return 'suspected'
  return 'hint'
}

/** 缓解原因收窄：未知/缺失 reason 返回 ''（不渲染原因徽标） */
export function deepIssueReason(iss: ConsistencyDeepIssue & DeepIssueAnnotation): DeepIssueReason | '' {
  return iss.reason && iss.reason in REASON_LEVEL ? (iss.reason as DeepIssueReason) : ''
}

/** 深检元信息一行文案：「AI 深检：已扫描 20 章」+ 可选失败后缀；无扫描记录返回 null */
export function deepScanMeta(result: ConsistencyDeepResult): string | null {
  if (result.chapters_scanned <= 0) return null
  const failed = result.chapters_failed && result.chapters_failed > 0 ? `，${result.chapters_failed} 章提取失败已跳过` : ''
  return `AI 深检：已扫描 ${result.chapters_scanned} 章${failed}`
}
