// TaskCenter「素材库」tab 轻扫更新用例（T1 收口）：
// tab 激活即重读 / 30s 低频轮询 / 页面不可见停表 / 收起（隐藏）停表。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { TaskCenter } from './TaskCenter'
import { imageHubAssets, readFileAsDataURL } from '../../api/image'

vi.mock('../../api/image', () => ({
  imageHubAssets: vi.fn(),
  readFileAsDataURL: vi.fn(),
  chapterArtList: vi.fn(),
}))

const imageHubAssetsMock = vi.mocked(imageHubAssets)
const readFileMock = vi.mocked(readFileAsDataURL)

const baseProps = {
  queueItems: [],
  onClearQueue: vi.fn(),
  onCancelQueue: vi.fn(),
  history: [],
  selectedIndex: -1,
  onSelectHistory: vi.fn(),
  onClearHistory: vi.fn(),
  templates: {},
  customTemplates: [],
  onApplyTemplate: vi.fn(),
  onManageTemplates: vi.fn(),
}

const ASSET = {
  id: 'ih-1', kind: 'image', path: 'C:/ws/a.png', space: 'play',
  source_board: 'imagegen', capability: 'media.generate',
  model: 'krea2', cost: '0', created_at: '2026-09-05T10:00:00Z',
}

beforeEach(() => {
  imageHubAssetsMock.mockReset().mockResolvedValue([ASSET])
  readFileMock.mockReset().mockResolvedValue('data:image/png;base64,AAA')
})

afterEach(() => {
  vi.useRealTimers()
  Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
})

async function openAssetsTab() {
  render(<TaskCenter {...baseProps} />)
  fireEvent.click(screen.getByRole('tab', { name: /素材库/ }))
}

describe('TaskCenter 素材库 tab 轻扫更新', () => {
  it('tab 激活即读取（play 空间最近 50 条）并渲染模型/成本列', async () => {
    await openAssetsTab()
    await waitFor(() => expect(imageHubAssetsMock).toHaveBeenCalledWith('play', '', 50))
    await waitFor(() => expect(screen.getByText('krea2')).toBeTruthy())
    expect(screen.getByText('0')).toBeTruthy()
  })

  it('离开再激活 tab 会重读（不再一次性读取后陈旧）', async () => {
    render(<TaskCenter {...baseProps} />)
    fireEvent.click(screen.getByRole('tab', { name: /素材库/ }))
    await waitFor(() => expect(imageHubAssetsMock).toHaveBeenCalledTimes(1))
    fireEvent.click(screen.getByRole('tab', { name: /队列/ }))
    fireEvent.click(screen.getByRole('tab', { name: /素材库/ }))
    await waitFor(() => expect(imageHubAssetsMock).toHaveBeenCalledTimes(2))
  })

  it('30s 低频轮询：可见时每 30s 轻扫一次', async () => {
    vi.useFakeTimers()
    render(<TaskCenter {...baseProps} />)
    fireEvent.click(screen.getByRole('tab', { name: /素材库/ }))
    await vi.advanceTimersByTimeAsync(0)
    expect(imageHubAssetsMock).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(30000)
    expect(imageHubAssetsMock).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(30000)
    expect(imageHubAssetsMock).toHaveBeenCalledTimes(3)
  })

  it('页面不可见（visibilityState=hidden）时停表不轮询', async () => {
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' })
    vi.useFakeTimers()
    render(<TaskCenter {...baseProps} />)
    fireEvent.click(screen.getByRole('tab', { name: /素材库/ }))
    await vi.advanceTimersByTimeAsync(60000)
    expect(imageHubAssetsMock).not.toHaveBeenCalled()
  })
})
