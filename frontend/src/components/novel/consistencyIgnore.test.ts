// consistencyIgnore.test.ts — 深检单条忽略记忆模块单测（指纹稳定性 / 项目隔离 /
// 忽略-恢复回环 / 上限丢弃 / 损坏数据容错）
import { beforeEach, describe, expect, it } from 'vitest'
import {
  clearIgnoredIssues,
  deepIssueFingerprint,
  ignoreIssue,
  isIssueIgnored,
  loadIgnoredFingerprints,
} from './consistencyIgnore'
import type { IgnoreFingerprintSource } from './consistencyIgnore'

const projA = 'C:/books/小说A'
const projB = 'C:/books/小说B'

const issue: IgnoreFingerprintSource = {
  category: 'status',
  entity_name: '林晚',
  location: '第3章',
  reason: 'alias',
}

beforeEach(() => {
  localStorage.clear()
})

describe('deepIssueFingerprint', () => {
  it('指纹由类别+实体+位置+原因组成，忽略首尾空白', () => {
    expect(deepIssueFingerprint(issue)).toBe('status|林晚|第3章|alias')
    expect(deepIssueFingerprint({ ...issue, entity_name: ' 林晚 ' })).toBe('status|林晚|第3章|alias')
  })

  it('不含会随修改漂移的描述/证据：同实体同类告警指纹稳定', () => {
    expect(deepIssueFingerprint({ ...issue, reason: undefined })).toBe('status|林晚|第3章|')
    // reason 变化（如后端标注调整）→ 新指纹，不再被视为同一条
    expect(deepIssueFingerprint({ ...issue, reason: 'wording' })).not.toBe(deepIssueFingerprint(issue))
  })
})

describe('忽略记忆回环', () => {
  it('ignoreIssue → loadIgnoredFingerprints/isIssueIgnored → clearIgnoredIssues 恢复', () => {
    expect(loadIgnoredFingerprints(projA)).toEqual([])
    expect(isIssueIgnored(projA, issue)).toBe(false)

    ignoreIssue(projA, issue)
    expect(loadIgnoredFingerprints(projA)).toEqual(['status|林晚|第3章|alias'])
    expect(isIssueIgnored(projA, issue)).toBe(true)

    clearIgnoredIssues(projA)
    expect(loadIgnoredFingerprints(projA)).toEqual([])
    expect(isIssueIgnored(projA, issue)).toBe(false)
  })

  it('同指纹幂等去重，不同指纹各自记录', () => {
    ignoreIssue(projA, issue)
    ignoreIssue(projA, issue)
    expect(loadIgnoredFingerprints(projA)).toHaveLength(1)
    ignoreIssue(projA, { ...issue, location: '第4章' })
    expect(loadIgnoredFingerprints(projA)).toHaveLength(2)
  })

  it('指纹按项目隔离：A 的忽略不影响 B', () => {
    ignoreIssue(projA, issue)
    expect(isIssueIgnored(projB, issue)).toBe(false)
    ignoreIssue(projB, { ...issue, location: '第7章' })
    expect(loadIgnoredFingerprints(projA)).toEqual(['status|林晚|第3章|alias'])
    expect(loadIgnoredFingerprints(projB)).toEqual(['status|林晚|第7章|alias'])
    clearIgnoredIssues(projA)
    expect(loadIgnoredFingerprints(projB)).toHaveLength(1)
  })

  it('空 projectPath 直接忽略操作（不写入也不报错）', () => {
    ignoreIssue('', issue)
    clearIgnoredIssues('')
    expect(loadIgnoredFingerprints('')).toEqual([])
    expect(loadIgnoredFingerprints(projA)).toEqual([])
  })

  it('超出 500 条上限丢最旧', () => {
    for (let i = 0; i < 505; i++) {
      ignoreIssue(projA, { ...issue, location: `第${i}章` })
    }
    const list = loadIgnoredFingerprints(projA)
    expect(list).toHaveLength(500)
    expect(list[0]).toBe('status|林晚|第5章|alias')
    expect(list[list.length - 1]).toBe('status|林晚|第504章|alias')
  })
})

describe('损坏数据容错', () => {
  it('localStorage JSON 损坏/类型不对时返回空且不抛错，写回即修复', () => {
    localStorage.setItem('gaea.novel.consistencyIgnore.v1', '{{{not json')
    expect(loadIgnoredFingerprints(projA)).toEqual([])
    ignoreIssue(projA, issue)
    expect(isIssueIgnored(projA, issue)).toBe(true)
    expect(JSON.parse(localStorage.getItem('gaea.novel.consistencyIgnore.v1') ?? '{}')).toEqual({
      [projA]: ['status|林晚|第3章|alias'],
    })
  })

  it('存储值类型异常（非字符串数组）被过滤', () => {
    localStorage.setItem('gaea.novel.consistencyIgnore.v1', JSON.stringify({ [projA]: ['ok', 42, null, ['x']] }))
    expect(loadIgnoredFingerprints(projA)).toEqual(['ok'])
  })

  it('数组形态的顶层存储视为无效', () => {
    localStorage.setItem('gaea.novel.consistencyIgnore.v1', JSON.stringify(['a', 'b']))
    expect(loadIgnoredFingerprints(projA)).toEqual([])
  })
})
