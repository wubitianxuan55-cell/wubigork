// IndexStatus.test.tsx — 造价数据库状态栏的「语义索引」状态钮：文案分档与点击补齐。
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { IndexStatus, indexSummary } from './IndexStatus'
import type { CostIndexStatusView } from '../../gaea/lib/types'

const KINDS = ['wc_quota', 'wc_res', 'wc_bill'] as const

function st(kinds: Array<{ label: string; total: number; count: number; lastError?: string; searchError?: string }>, running = true): CostIndexStatusView {
  return {
    model: 'bge-m3',
    running,
    kinds: kinds.map((k, i) => ({ kind: KINDS[i], updatedAt: '', ...k })),
  }
}

describe('indexSummary · 状态钮的文案分档', () => {
  it('三类都建好 = 就绪；悬停明细带模型名与每类的 已建/总数', () => {
    const s = indexSummary(st([{ label: '定额', total: 2, count: 2 }, { label: '工料机', total: 3, count: 3 }, { label: '清单', total: 4, count: 4 }]))
    expect(s.tone).toBe('ok')
    expect(s.text).toBe('语义索引 就绪')
    expect(s.title).toContain('向量模型 bge-m3')
    expect(s.title).toContain('定额 2/2 · 工料机 3/3 · 清单 4/4')
    expect(s.title).toContain('点击立即补齐')
  })

  it('建了一部分 = 建设中，钮上直接给 已建/总数', () => {
    const s = indexSummary(st([{ label: '定额', total: 5, count: 2 }, { label: '工料机', total: 3, count: 1 }]))
    expect(s.tone).toBe('partial')
    expect(s.text).toBe('语义索引 3/8')
  })

  it('有一类出错 = 待补齐（转警示色），明细带原因', () => {
    const s = indexSummary(st([{ label: '定额', total: 2, count: 0, lastError: 'embedding 模型不可用' }, { label: '工料机', total: 3, count: 3 }]))
    expect(s.tone).toBe('warn')
    expect(s.text).toBe('语义索引 待补齐')
    expect(s.title).toContain('定额：embedding 模型不可用')
  })

  it('检索时发现向量模型不可用（索引是齐的）= 语义模型不可用（转警示色），明细带原因', () => {
    const s = indexSummary(st([{ label: '定额', total: 2, count: 2, searchError: '本地向量模型请求失败：embedding HTTP 500' }, { label: '工料机', total: 3, count: 3 }]))
    expect(s.tone).toBe('warn')
    expect(s.text).toBe('语义模型 不可用')
    expect(s.title).toContain('定额：本地向量模型请求失败：embedding HTTP 500')
  })

  it('一条向量都没有 = 未建', () => {
    const s = indexSummary(st([{ label: '定额', total: 2, count: 0 }]))
    expect(s.tone).toBe('empty')
    expect(s.text).toBe('语义索引 未建')
  })

  it('对账数落后于向量数（刚重启、这一轮还没对完）不算建设中', () => {
    const s = indexSummary(st([{ label: '定额', total: 0, count: 2 }]))
    expect(s.tone).toBe('ok')
  })

  it('后台索引器没在跑：明细点出来', () => {
    const s = indexSummary(st([{ label: '定额', total: 2, count: 2 }], false))
    expect(s.title).toContain('后台索引器未运行')
  })
})

describe('IndexStatus · 状态钮', () => {
  it('没有状态（老版本后端 / 读取失败）不渲染；空库（总数 0）也不渲染', () => {
    const { container } = render(<IndexStatus status={null} onBackfill={async () => {}} />)
    expect(container.firstChild).toBeNull()
    const empty = render(<IndexStatus status={st([{ label: '定额', total: 0, count: 0 }])} onBackfill={async () => {}} />)
    expect(empty.container.firstChild).toBeNull()
  })

  it('点击调补齐；忙时钮禁用并显示补齐中，完成后恢复', async () => {
    let finish: () => void = () => {}
    const onBackfill = vi.fn(() => new Promise<void>((resolve) => { finish = resolve }))
    render(<IndexStatus status={st([{ label: '定额', total: 2, count: 1 }])} onBackfill={onBackfill} />)
    const btn = screen.getByTestId('cost-index-status')
    expect(btn.getAttribute('data-tone')).toBe('partial')
    fireEvent.click(btn)
    expect(onBackfill).toHaveBeenCalledTimes(1)
    const busy = await screen.findByText('索引补齐中…')
    expect((busy as HTMLButtonElement).disabled).toBe(true)
    finish()
    await waitFor(() => expect(screen.getByTestId('cost-index-status').textContent).toBe('语义索引 1/2'))
    expect((screen.getByTestId('cost-index-status') as HTMLButtonElement).disabled).toBe(false)
  })
})
