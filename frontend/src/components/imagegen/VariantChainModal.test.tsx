// 变体簇溯源（阶段二刀 D）：沿 parent_id 建链（根→当前倒序）、断链诚实标注、
// 「用到画布」回调构造 GenResult。api 层 mock（imageHubAssets/readFileAsDataURL）。
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react'
import type { Mock } from 'vitest'

const mocks = vi.hoisted(() => ({
  imageHubAssets: vi.fn(),
  readFileAsDataURL: vi.fn(),
}))

vi.mock('../../api/image', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../api/image')>()
  return {
    ...actual,
    imageHubAssets: mocks.imageHubAssets,
    readFileAsDataURL: mocks.readFileAsDataURL,
  }
})

import VariantChainModal from './VariantChainModal'
import type { GenResult } from './types'

// 三代链：A（根）← B ← C（当前）
const ENTRIES = [
  { id: 'ih-c', parent_id: 'ih-b', model: 'qwen-image-edit', created_at: '2026-09-23T10:00:00Z', prompt_truncate: '把外套改成红色', path: 'C:/img/c.png' },
  { id: 'ih-b', parent_id: 'ih-a', model: 'qwen-image-edit', created_at: '2026-09-23T09:00:00Z', prompt_truncate: '扩图四边', path: 'C:/img/b.png' },
  { id: 'ih-a', model: 'krea2', created_at: '2026-09-23T08:00:00Z', prompt_truncate: '初始生成', path: 'C:/img/a.png' },
]

describe('VariantChainModal 变体簇溯源（阶段二刀 D）', () => {
  beforeEach(() => {
    mocks.imageHubAssets.mockResolvedValue(ENTRIES)
    mocks.readFileAsDataURL.mockImplementation(async (p: string) => `data:image/png;base64,${p.endsWith('a.png') ? 'A' : p.endsWith('b.png') ? 'B' : 'C'}`)
  })
  afterEach(() => {
    vi.restoreAllMocks()
    cleanup()
  })

  it('沿 parent_id 建链：根→…→当前 正序三代；当前行高亮且无「用到画布」', async () => {
    render(<VariantChainModal open assetId="ih-c" onClose={vi.fn()} onApply={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('variant-chain-list')).toBeTruthy())
    const first = screen.getByTestId('variant-chain-item-0').textContent ?? ''
    const last = screen.getByTestId('variant-chain-item-2').textContent ?? ''
    expect(first).toContain('第 1 代')
    expect(first).toContain('krea2') // 根在前
    expect(first).toContain('源图')
    expect(last).toContain('当前')
    // 当前行（尾代）没有「用到画布」；第 1/2 代有
    expect(screen.queryByTestId('variant-chain-apply-2')).toBeNull()
    expect(screen.getByTestId('variant-chain-apply-0')).toBeTruthy()
    expect(screen.getByTestId('variant-chain-apply-1')).toBeTruthy()
    expect(screen.queryByTestId('variant-chain-broken')).toBeNull()
  })

  it('用到画布：onApply 收祖先条目（图+file_path+asset_id）并关闭', async () => {
    const onApply = vi.fn()
    const onClose = vi.fn()
    render(<VariantChainModal open assetId="ih-c" onClose={onClose} onApply={onApply} />)
    await waitFor(() => expect(screen.getByTestId('variant-chain-apply-1')).toBeTruthy())
    fireEvent.click(screen.getByTestId('variant-chain-apply-1'))
    expect(onApply).toHaveBeenCalledTimes(1)
    const r = (onApply as Mock).mock.calls[0][0] as GenResult
    expect(r.image).toContain('base64,B')
    expect(r.file_path).toBe('C:/img/b.png')
    expect(r.asset_id).toBe('ih-b')
    expect(onClose).toHaveBeenCalled()
  })

  it('断链诚实标注：祖先缺档显示提示不造假', async () => {
    mocks.imageHubAssets.mockResolvedValue([
      { id: 'ih-c', parent_id: 'ih-gone', model: 'qwen-image-edit', path: 'C:/img/c.png' },
    ])
    render(<VariantChainModal open assetId="ih-c" onClose={vi.fn()} onApply={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('variant-chain-broken')).toBeTruthy())
    expect(screen.getByTestId('variant-chain-item-0').textContent).toContain('第 1 代')
  })

  it('空台账：未找到变体记录（诚实空态）', async () => {
    mocks.imageHubAssets.mockResolvedValue([])
    render(<VariantChainModal open assetId="ih-x" onClose={vi.fn()} onApply={vi.fn()} />)
    await waitFor(() => expect(screen.getByText('未找到变体记录')).toBeTruthy())
  })
})
