// consistencyDeep.test.ts — 一致性 AI 深检前端纯函数单测（载荷收窄 / 章数夹取 / 徽标 / 分级）
import { describe, expect, it } from 'vitest'
import {
  clampDeepChapters,
  deepIssueLevel,
  deepIssueReason,
  deepScanMeta,
  normalizeDeepResult,
  sourceBadge,
} from './consistencyDeep'
import type { ConsistencyDeepIssue } from '../../types'

const aiIssue: ConsistencyDeepIssue = {
  severity: 'error',
  category: 'status',
  entity_name: '林晚',
  description: '林晚 在第2章已死亡，但第3章仍以存活状态出场',
  location: '第3章',
  evidence: '第2章状态卡 status=dead；第3章状态卡 status=alive',
  suggestion: '确认是否为复活/回忆/幻象并在文中明确交代',
  branch: '',
  source: 'ai',
}

describe('clampDeepChapters', () => {
  it('非法输入回退默认 20', () => {
    expect(clampDeepChapters(Number.NaN)).toBe(20)
    expect(clampDeepChapters(Number.POSITIVE_INFINITY)).toBe(20)
  })

  it('越界夹取到 [1, 50]，≤0 归一到 1（后端另有 ≤0→50 兜底）', () => {
    expect(clampDeepChapters(0)).toBe(1)
    expect(clampDeepChapters(-5)).toBe(1)
    expect(clampDeepChapters(51)).toBe(50)
    expect(clampDeepChapters(20)).toBe(20)
    expect(clampDeepChapters(7.6)).toBe(8)
  })
})

describe('normalizeDeepResult', () => {
  it('收窄合法载荷并补齐缺省字段', () => {
    const r = normalizeDeepResult({
      issues: [aiIssue],
      total_issues: 1,
      summary: '发现 1 个问题（1 错误, 0 警告, 0 提示）；AI 深检已扫描 3 章',
      chapters_scanned: 3,
      ai_available: true,
    })
    expect(r).not.toBeNull()
    expect(r!.issues).toHaveLength(1)
    expect(r!.issues[0].source).toBe('ai')
    expect(r!.chapters_scanned).toBe(3)
    expect(r!.ai_available).toBe(true)
    expect(r!.chapters_failed).toBeUndefined()
    expect(r!.ai_note).toBe('')
  })

  it('total_issues 缺失时回退 issues 长度；ai_available 严格 === true', () => {
    const r = normalizeDeepResult({ issues: [aiIssue, aiIssue], ai_available: 'yes' })
    expect(r!.total_issues).toBe(2)
    expect(r!.ai_available).toBe(false)
  })

  it('坏载荷（缺 issues 数组 / null / undefined）返回 null', () => {
    expect(normalizeDeepResult(null)).toBeNull()
    expect(normalizeDeepResult(undefined)).toBeNull()
    expect(normalizeDeepResult({})).toBeNull()
    expect(normalizeDeepResult({ issues: 'not-an-array' })).toBeNull()
  })

  it('降级载荷（ai_available=false + ai_note）原样保留说明', () => {
    const r = normalizeDeepResult({
      issues: [],
      total_issues: 0,
      summary: '✅ 未发现一致性问题',
      chapters_scanned: 0,
      chapters_failed: 3,
      ai_available: false,
      ai_note: 'AI 逐章提取全部失败（3 章，最后错误: LLM 调用失败），仅显示规则检查结果',
    })
    expect(r!.ai_available).toBe(false)
    expect(r!.ai_note).toContain('全部失败')
    expect(deepScanMeta(r!)).toBeNull()
  })
})

describe('sourceBadge', () => {
  it('ai/rule 映射徽标，未知来源返回 null', () => {
    expect(sourceBadge('ai')).toEqual({ label: 'AI', color: 'purple' })
    expect(sourceBadge('rule')).toEqual({ label: '规则', color: 'default' })
    expect(sourceBadge(undefined)).toBeNull()
    expect(sourceBadge('mystery')).toBeNull()
  })
})

describe('deepScanMeta', () => {
  it('有扫描记录时输出章数与可选失败后缀', () => {
    expect(deepScanMeta({ issues: [], total_issues: 0, summary: '', chapters_scanned: 20, ai_available: true, ai_note: '' })).toBe('AI 深检：已扫描 20 章')
    expect(
      deepScanMeta({ issues: [], total_issues: 0, summary: '', chapters_scanned: 2, chapters_failed: 1, ai_available: true, ai_note: '1 章 AI 提取失败已跳过' }),
    ).toBe('AI 深检：已扫描 2 章，1 章提取失败已跳过')
  })
})

// ── 误报缓解：置信度/原因标注 → UI 三档分级 ─────────────────

describe('deepIssueLevel', () => {
  it('无标注时按 severity 直映射：error→冲突 / warning→疑似 / info→提示', () => {
    expect(deepIssueLevel({ ...aiIssue })).toBe('conflict')
    expect(deepIssueLevel({ ...aiIssue, severity: 'warning' })).toBe('suspected')
    expect(deepIssueLevel({ ...aiIssue, severity: 'info' })).toBe('hint')
  })

  it('带缓解原因时按原因降级（只降不升，绝不吞真问题）', () => {
    // 措辞差异 → 提示（即使后端误标 error 也压到提示档）
    expect(deepIssueLevel({ ...aiIssue, severity: 'error', reason: 'wording' })).toBe('hint')
    // 时间粒度差异 / 称谓别名 / 缺少明确交代 → 疑似
    expect(deepIssueLevel({ ...aiIssue, severity: 'warning', reason: 'granularity' })).toBe('suspected')
    expect(deepIssueLevel({ ...aiIssue, severity: 'warning', reason: 'alias' })).toBe('suspected')
    expect(deepIssueLevel({ ...aiIssue, severity: 'warning', reason: 'unexplained' })).toBe('suspected')
  })

  it('未知 reason 视为无标注，回退 severity 映射', () => {
    expect(deepIssueLevel({ ...aiIssue, reason: 'mystery' })).toBe('conflict')
  })
})

describe('deepIssueReason', () => {
  it('合法原因透传，未知/缺失返回空串', () => {
    expect(deepIssueReason({ ...aiIssue, reason: 'wording' })).toBe('wording')
    expect(deepIssueReason({ ...aiIssue, reason: 'granularity' })).toBe('granularity')
    expect(deepIssueReason({ ...aiIssue, reason: 'alias' })).toBe('alias')
    expect(deepIssueReason({ ...aiIssue, reason: 'unexplained' })).toBe('unexplained')
    expect(deepIssueReason({ ...aiIssue, reason: 'mystery' })).toBe('')
    expect(deepIssueReason({ ...aiIssue })).toBe('')
  })
})
