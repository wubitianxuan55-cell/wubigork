// AssetLibrary（T1 画室素材库独立页）行为用例：
// 默认 work 空间读取 / 卡片溯源元数据 / 原语筛选 / 懒加载页递增 / 空间切换 /
// 返回出口 / 空态诚实文案。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { LocaleProvider } from '../../gaea/lib/i18n'
import { AssetLibrary } from './AssetLibrary'
import { imageHubAssets, readFileAsDataURL, type ImageHubAssetView } from '../../api/image'

vi.mock('../../api/image', () => ({
  imageHubAssets: vi.fn(),
  readFileAsDataURL: vi.fn(),
}))

const imageHubAssetsMock = vi.mocked(imageHubAssets)
const readFileMock = vi.mocked(readFileAsDataURL)

function rec(over: Partial<ImageHubAssetView>): ImageHubAssetView {
  return {
    id: 'ih-1', kind: 'image', path: 'C:/ws/.gaea/uploads/a.png', space: 'work',
    source_board: 'imagegen', capability: 'media.generate',
    model: 'krea2', cost: '0', created_at: '2026-09-05T10:00:00Z',
    prompt_truncate: '一只猫', params: { character_id: 'char-1' },
    ...over,
  }
}

function renderLibrary() {
  return render(
    <LocaleProvider>
      <AssetLibrary onClose={vi.fn()} />
    </LocaleProvider>,
  )
}

beforeEach(() => {
  localStorage.setItem('gaea-lang', 'zh')
  imageHubAssetsMock.mockReset()
  readFileMock.mockReset()
  // 对齐 GaeaAttachmentDataURL 口径：位图给 dataURL，.mmd 代码产物不给（占位）。
  readFileMock.mockImplementation((path: string) =>
    path.endsWith('.mmd')
      ? Promise.resolve('')
      : Promise.resolve('data:image/png;base64,AAA'))
})

describe('AssetLibrary 画室素材库独立页', () => {
  it('默认读 work 空间并展示溯源元数据（模型/成本/来源/时间）', async () => {
    imageHubAssetsMock.mockResolvedValue([
      rec({}),
      rec({ id: 'ih-2', path: 'C:/ws/.gaea/uploads/d.mmd', kind: 'diagram', capability: 'media.diagram', model: '', cost: '' }),
    ])
    renderLibrary()
    await waitFor(() => expect(imageHubAssetsMock).toHaveBeenCalledWith('work', '', 120))
    // 缩略卡：模型/成本 tag + 来源/时间脚注；图示 .mmd 无位图 → 诚实占位。
    await waitFor(() => expect(screen.getAllByText('krea2').length).toBeGreaterThan(0))
    expect(screen.getByText('0')).toBeTruthy()
    expect(screen.getAllByText('绘梦 · 2026-09-05T10:00:00Z').length).toBe(2)
    expect(screen.getByText('缩略图不可用')).toBeTruthy()
    expect(screen.getByText('2 项素材')).toBeTruthy()
  })

  it('原语筛选：切到图示只留 media.diagram 素材', async () => {
    imageHubAssetsMock.mockResolvedValue([
      rec({}),
      rec({ id: 'ih-2', path: 'C:/ws/.gaea/uploads/d.mmd', kind: 'diagram', capability: 'media.diagram', model: '', cost: '' }),
    ])
    renderLibrary()
    await waitFor(() => expect(screen.getByText('2 项素材')).toBeTruthy())
    fireEvent.click(screen.getByText('图示'))
    await waitFor(() => expect(screen.getByText('1 项素材')).toBeTruthy())
    expect(screen.getByTitle('C:/ws/.gaea/uploads/d.mmd')).toBeTruthy()
    expect(screen.queryByTitle('C:/ws/.gaea/uploads/a.png')).toBeNull()
  })

  it('懒加载：只转换当前可见页缩略，「加载更多」递增下一页', async () => {
    const many = Array.from({ length: 15 }, (_, i) =>
      rec({ id: `ih-${i}`, path: `C:/ws/.gaea/uploads/p${i}.png` }))
    imageHubAssetsMock.mockResolvedValue(many)
    renderLibrary()
    // 首页 12 张 → 12 次缩略读取。
    await waitFor(() => expect(readFileMock).toHaveBeenCalledTimes(12))
    fireEvent.click(screen.getByText('加载更多'))
    await waitFor(() => expect(readFileMock).toHaveBeenCalledTimes(15))
  })

  it('空间切换：创作空间按 play 空间重读', async () => {
    imageHubAssetsMock.mockResolvedValue([])
    renderLibrary()
    await waitFor(() => expect(imageHubAssetsMock).toHaveBeenCalledWith('work', '', 120))
    const playOpt = screen.getAllByText('创作空间').find((el) => !!el.closest('.ant-segmented-item'))
    fireEvent.click(playOpt as HTMLElement)
    await waitFor(() => expect(imageHubAssetsMock).toHaveBeenCalledWith('play', '', 120))
  })

  it('返回生成台：触发 onClose 出口', async () => {
    imageHubAssetsMock.mockResolvedValue([])
    const onClose = vi.fn()
    render(
      <LocaleProvider>
        <AssetLibrary onClose={onClose} />
      </LocaleProvider>,
    )
    await waitFor(() => expect(imageHubAssetsMock).toHaveBeenCalled())
    fireEvent.click(screen.getByText('返回生成台'))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('空态诚实文案：无登记时提示自动溯源入库', async () => {
    imageHubAssetsMock.mockResolvedValue([])
    renderLibrary()
    await waitFor(() =>
      expect(screen.getByText('该空间还没有登记素材——生成的图片会自动溯源入库。')).toBeTruthy())
  })

  it('详情条：点选卡片展开完整溯源（来源/角色/时间/提示词/路径）', async () => {
    imageHubAssetsMock.mockResolvedValue([rec({ source_board: 'characterlib' })])
    const { container } = renderLibrary()
    await waitFor(() => expect(screen.getByTitle('C:/ws/.gaea/uploads/a.png')).toBeTruthy())
    fireEvent.click(screen.getByTitle('C:/ws/.gaea/uploads/a.png'))
    // 元数据行由 Typography + 值节点拼接，按容器文本断言（逐字段）。
    await waitFor(() => {
      const text = container.textContent || ''
      expect(text).toContain('生图')
      expect(text).toContain('模型: krea2')
      expect(text).toContain('成本: 0')
      expect(text).toContain('来源: 角色')
      expect(text).toContain('角色: char-1')
      expect(text).toContain('时间: 2026-09-05T10:00:00Z')
      expect(text).toContain('提示词: 一只猫')
      expect(text).toContain('路径: C:/ws/.gaea/uploads/a.png')
    })
  })
})
