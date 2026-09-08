import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useImageGenHistory } from './useImageGenHistory'
import { saveExportBlob } from '../gaea/lib/saveFile'
import { loadHistoryMeta, resolveResultImage } from '../components/imagegen/meta'
import type { GenResult } from '../components/imagegen/types'

/**
 * useImageGenHistory.test.ts — 审计刀B a 回归锁（下载双门）：
 * 壳内（inShellEnv）dataURL→Blob 走 saveExportBlob 系统另存为（文件名沿用
 * downloadFileName(r)）；浏览器保留原 <a download> 语义。
 */

vi.mock('../api/image', () => ({
  setCharacterPortrait: vi.fn(async () => undefined),
  readFileAsDataURL: vi.fn(async () => ''),
}))
vi.mock('../components/imagegen/historyMeta', () => ({
  restoreHistoryImages: vi.fn(async () => []),
}))
vi.mock('../components/imagegen/meta', () => ({
  loadHistoryMeta: vi.fn((): GenResult[] => []),
  saveHistoryMeta: vi.fn(),
  resolveResultImage: vi.fn(async (): Promise<string> => ''),
}))
// saveExportBlob 换 spy、dataUrlToBlob 保留真实现（断言 blob 内容/mime）。
vi.mock('../gaea/lib/saveFile', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../gaea/lib/saveFile')>()
  return { ...actual, saveExportBlob: vi.fn(async () => true) }
})

const PNG = 'data:image/png;base64,' + btoa('pngbytes')
const item: GenResult = {
  image: PNG,
  seed: 42,
  time: 1,
  prompt: 'p',
  model: 'm',
  size: '512x512',
}

const stubShell = () => {
  ;(window as unknown as { go?: unknown }).go = { app: { ImageGenTestB: {} } }
}

afterEach(() => {
  delete (window as unknown as { go?: unknown }).go
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

describe('useImageGenHistory 下载双门（审计刀B a）', () => {
  it('壳内：dataURL→Blob 走 saveExportBlob，文件名=downloadFileName(r)', async () => {
    vi.mocked(loadHistoryMeta).mockReturnValue([item])
    vi.mocked(resolveResultImage).mockResolvedValue(PNG)
    stubShell()
    const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1_700_000_000_000)
    const { result } = renderHook(() =>
      useImageGenHistory({
        setPrompt: vi.fn(),
        setNegative: vi.fn(),
        setSeed: vi.fn(),
        setSize: vi.fn(),
      }),
    )
    await act(async () => {
      await result.current.handleDownload(0)
    })
    expect(saveExportBlob).toHaveBeenCalledTimes(1)
    const [blob, name] = vi.mocked(saveExportBlob).mock.calls[0]
    expect(name).toBe('gaea-1700000000000-seed42.png')
    expect(blob.type).toBe('image/png')
    await expect(blob.text()).resolves.toBe('pngbytes')
    nowSpy.mockRestore()
  })

  it('浏览器：保留原 <a download>（href=dataURL、download=downloadFileName），不经 saveExportBlob', async () => {
    vi.mocked(loadHistoryMeta).mockReturnValue([item])
    vi.mocked(resolveResultImage).mockResolvedValue(PNG)
    const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1_700_000_000_000)
    const clicked: HTMLAnchorElement[] = []
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(function clickCapture(this: HTMLAnchorElement) {
        clicked.push(this)
      })
    const { result } = renderHook(() =>
      useImageGenHistory({
        setPrompt: vi.fn(),
        setNegative: vi.fn(),
        setSeed: vi.fn(),
        setSize: vi.fn(),
      }),
    )
    await act(async () => {
      await result.current.handleDownload(0)
    })
    expect(saveExportBlob).not.toHaveBeenCalled()
    expect(click).toHaveBeenCalledTimes(1)
    expect(clicked[0].download).toBe('gaea-1700000000000-seed42.png')
    expect(clicked[0].href).toBe(PNG)
    click.mockRestore()
    nowSpy.mockRestore()
  })
})
