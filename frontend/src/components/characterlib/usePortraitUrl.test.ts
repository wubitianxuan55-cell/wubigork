import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getCachedAttachmentDataURL, __clearAttachmentUrlCacheForTest } from './usePortraitUrl'

const { attachmentDataURLMock } = vi.hoisted(() => ({
  attachmentDataURLMock: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', () => ({
  app: { AttachmentDataURL: attachmentDataURLMock },
}))

// jsdom 是否原生提供 createObjectURL 因版本而异——测试显式桩住两分支
function stubObjectUrls() {
  const revoked: string[] = []
  let seq = 0
  const urls = new Map<string, Blob>()
  const createObjectURL = vi.fn((b: Blob) => {
    const u = `blob:mock-${++seq}`
    urls.set(u, b)
    return u
  })
  const revokeObjectURL = vi.fn((u: string) => {
    revoked.push(u)
    urls.delete(u)
  })
  Object.defineProperty(URL, 'createObjectURL', { configurable: true, writable: true, value: createObjectURL })
  Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, writable: true, value: revokeObjectURL })
  return { createObjectURL, revokeObjectURL, revoked, urls }
}

describe('usePortraitUrl 缓存 blob URL 化（v4.391）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    __clearAttachmentUrlCacheForTest()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('桥返回的 data URL 转为 blob: URL 缓存，同路径只打一次桥', async () => {
    const { createObjectURL } = stubObjectUrls()
    attachmentDataURLMock.mockResolvedValue('data:image/png;base64,AAA')
    const u1 = await getCachedAttachmentDataURL('C:/x/a.png')
    const u2 = await getCachedAttachmentDataURL('C:/x/a.png')
    expect(u1).toMatch(/^blob:/)
    expect(u2).toBe(u1)
    expect(createObjectURL).toHaveBeenCalledTimes(1)
    expect(attachmentDataURLMock).toHaveBeenCalledTimes(1)
  })

  it('blob 字节与 MIME 忠实于原 data URL', async () => {
    const { urls } = stubObjectUrls()
    // 'AAECAw==' = bytes [0,1,2,3]
    attachmentDataURLMock.mockResolvedValue('data:image/jpeg;base64,AAECAw==')
    const u = await getCachedAttachmentDataURL('C:/x/b.jpg')
    const blob = urls.get(u)!
    expect(blob.type).toBe('image/jpeg')
    const bytes = new Uint8Array(await blob.arrayBuffer())
    expect(Array.from(bytes)).toEqual([0, 1, 2, 3])
  })

  it('无 createObjectURL 环境（jsdom 旧版/降级）回退 data URL，行为与旧版一致', async () => {
    const origCreate = URL.createObjectURL
    const origRevoke = URL.revokeObjectURL
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: undefined })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: undefined })
    attachmentDataURLMock.mockResolvedValue('data:image/png;base64,AAA')
    try {
      const u = await getCachedAttachmentDataURL('C:/x/c.png')
      expect(u).toBe('data:image/png;base64,AAA')
    } finally {
      Object.defineProperty(URL, 'createObjectURL', { configurable: true, writable: true, value: origCreate })
      Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, writable: true, value: origRevoke })
    }
  })

  it('LRU 淘汰：超 96 条时最旧条目移出缓存并延迟 revoke；命中刷新 recency 不被淘', async () => {
    vi.useFakeTimers()
    const { revoked } = stubObjectUrls()
    attachmentDataURLMock.mockImplementation(async (p: string) => `data:image/png;base64,${btoa(p)}`)
    for (let i = 0; i < 96; i++) {
      await getCachedAttachmentDataURL(`C:/old-${i}.png`)
    }
    // 命中 0 号刷新 recency 后再填 1 条 → 被淘的是 1 号
    await getCachedAttachmentDataURL('C:/old-0.png')
    const first = await getCachedAttachmentDataURL('C:/new-1.png')
    expect(first).toMatch(/^blob:/)
    expect(revoked).toHaveLength(0) // 延迟未到，未 revoke
    // 0 号仍在缓存（命中刷新过），1 号被移出（重打桥）
    const zero = await getCachedAttachmentDataURL('C:/old-0.png')
    expect(zero).toMatch(/^blob:/)
    expect(attachmentDataURLMock).toHaveBeenCalledWith('C:/old-1.png')
    vi.advanceTimersByTime(120_000)
    expect(revoked.length).toBe(1)
  })

  it('淘汰的 blob URL 在延迟期满后 revoke（防浏览器侧无界驻留）', async () => {
    vi.useFakeTimers()
    const { revoked } = stubObjectUrls()
    attachmentDataURLMock.mockImplementation(async (p: string) => `data:image/png;base64,${btoa(p)}`)
    for (let i = 0; i <= 96; i++) {
      await getCachedAttachmentDataURL(`C:/k-${i}.png`)
    }
    expect(revoked).toHaveLength(0)
    vi.advanceTimersByTime(120_000)
    expect(revoked).toHaveLength(1)
    expect(revoked[0]).toMatch(/^blob:/)
  })
})
