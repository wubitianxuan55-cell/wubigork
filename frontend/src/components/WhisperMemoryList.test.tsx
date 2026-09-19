// WhisperMemoryList — 过滤/分组 memo 化（v4.349 运行开销刀）
//
// 背景：原实现每次渲染都跑 facts.filter + 6 个域各一次 filtered.filter
// （≈7N 次谓词），且每个 fact 里重算 search.toLowerCase() 两次；面板每敲一个
// 字符、每次折叠切换都整表重算。改为 useMemo([facts, search]) 后：
//   - 折叠（无关状态变化）不再重跑过滤；
//   - 过滤结果按输入变化重算（语义与改动前逐字相同，含空串=不过滤）。
import { describe, expect, it, vi, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import WhisperMemoryList from './WhisperMemoryList'
import type { MemoryFact } from './WhisperMemoryModal'

function fact(id: string, domain: string, subject: string, summary = ''): MemoryFact {
  return { id, domain, subject, summary, weight: 5, confidence: 0.8, createdAt: '2026-09-19 10:00' }
}

const FACTS: MemoryFact[] = [
  fact('1', 'IDENTITY', '名字叫小林', '住在城南'),
  fact('2', 'INNER_WORLD', '喜欢雨天', '下雨时喜欢坐窗边'),
  fact('3', 'SOCIAL', '有个妹妹', '妹妹在读大学'),
]

/** 只统计「以 facts 本体为 this」的 filter 调用（避开 React/antd 内部噪声） */
function spyFactsFilter(facts: MemoryFact[]) {
  const calls: number[] = []
  const orig = Array.prototype.filter
  const spy = vi.spyOn(Array.prototype, 'filter').mockImplementation(function (
    this: unknown[],
    ...args: unknown[]
  ) {
    if (this === facts) calls.push(1)
    return (orig as (...a: unknown[]) => unknown).apply(this, args) as unknown[]
  })
  return { calls, spy }
}

afterEach(() => vi.restoreAllMocks())

describe('WhisperMemoryList 过滤/分组（v4.349 memo 化）', () => {
  it('按域分组渲染，且无搜索时不过滤（空串语义与改动前一致）', () => {
    render(<WhisperMemoryList facts={FACTS} onOpenManage={() => {}} />)
    expect(screen.getByText('名字叫小林')).toBeTruthy()
    expect(screen.getByText('喜欢雨天')).toBeTruthy()
    expect(screen.getByText('有个妹妹')).toBeTruthy()
  })

  it('搜索命中 subject/summary；无匹配给「无匹配」空态', () => {
    render(<WhisperMemoryList facts={FACTS} onOpenManage={() => {}} />)
    fireEvent.change(screen.getByPlaceholderText('搜索记忆'), { target: { value: '窗边' } })
    expect(screen.getByText('喜欢雨天')).toBeTruthy()
    expect(screen.queryByText('名字叫小林')).toBeNull()

    fireEvent.change(screen.getByPlaceholderText('搜索记忆'), { target: { value: '不存在的词' } })
    expect(screen.getByText('无匹配')).toBeTruthy()
  })

  it('折叠/展开（无关状态变化）不重跑长表过滤——memo 生效的证据', () => {
    const { calls } = spyFactsFilter(FACTS)
    render(<WhisperMemoryList facts={FACTS} onOpenManage={() => {}} />)
    const afterMount = calls.length
    expect(afterMount).toBeGreaterThan(0)

    fireEvent.click(screen.getByText('身份')) // 折叠「身份」域
    expect(calls.length, '折叠重渲染不该再过滤 facts').toBe(afterMount)

    fireEvent.click(screen.getByText('身份')) // 再展开
    expect(calls.length).toBe(afterMount)
  })

  it('搜索词变化会重新过滤（memo 依赖正确）', () => {
    const { calls } = spyFactsFilter(FACTS)
    render(<WhisperMemoryList facts={FACTS} onOpenManage={() => {}} />)
    const afterMount = calls.length
    fireEvent.change(screen.getByPlaceholderText('搜索记忆'), { target: { value: '雨' } })
    expect(calls.length).toBeGreaterThan(afterMount)
  })

  it('条目行可键盘激活（Enter/Space → onOpenManage），与本刀可访问性口径一致', () => {
    const onOpenManage = vi.fn()
    render(<WhisperMemoryList facts={FACTS} onOpenManage={onOpenManage} />)
    const row = screen.getByText('喜欢雨天').closest('[role="button"]')!
    fireEvent.keyDown(row, { key: 'Enter' })
    expect(onOpenManage).toHaveBeenCalledTimes(1)
    fireEvent.keyDown(row, { key: ' ' })
    expect(onOpenManage).toHaveBeenCalledTimes(2)
  })
})
