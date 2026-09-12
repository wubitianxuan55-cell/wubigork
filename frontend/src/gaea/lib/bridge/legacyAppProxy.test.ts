// bridge/legacyAppProxy.test.ts — S2-3 兼容代理（window.go.app.App）回归：
// v4.257.1 真机缺陷「AttachmentDataURL is not a function」的根因是兼容代理
// 只认字面名、认不出 gaeaToGaea 的短名映射（Go 侧真名 GaeaAttachmentDataURL）。
// 这里用假门面复现真机形态，锁住「短名 → Go 方法名」回退与 api/image 的读取路径。
import { beforeEach, describe, expect, it, vi } from 'vitest'

type FakeGoApp = {
  App?: unknown
  OfficeB?: Record<string, unknown>
  ImageB?: Record<string, unknown>
}

function installFakeShell(overrides: Partial<FakeGoApp> = {}) {
  const go: { app: FakeGoApp } = { app: {} }
  go.app = {
    // 真机形态：门面上挂的是 Go 方法名（Gaea* 前缀）
    OfficeB: {
      GaeaAttachmentDataURL: vi.fn(async (path: string) => `data:image/png;base64,${path}`),
      GaeaSavePastedImage: vi.fn(async () => 'C:/tmp/pasted.png'),
    },
    ImageB: {
      GetImageBackendInfo: vi.fn(async () => ({ backend: 'comfyui' })),
    },
    ...overrides,
  }
  ;(window as unknown as { go?: unknown }).go = go
  // initBridge 幂等位：每个用例都要重置，否则兼容代理不会被重新安装
  delete (window as unknown as Record<string, unknown>).__bridge_initialized
  return go
}

beforeEach(() => {
  delete (window as unknown as { go?: unknown }).go
})

describe('window.go.app.App 兼容代理（S2-3）', () => {
  it('短名经 gaeaToGaea 映射到 Go 方法名（AttachmentDataURL → GaeaAttachmentDataURL）', async () => {
    const go = installFakeShell()
    const { initBridge } = await import('../bridge')
    initBridge()

    const legacyApp = go.app.App as Record<string, unknown>
    expect(typeof legacyApp.AttachmentDataURL).toBe('function')
    await expect((legacyApp.AttachmentDataURL as (p: string) => Promise<string>)('x'))
      .resolves.toBe('data:image/png;base64,x')
    // 字面名仍然优先（门面上直接有的方法不需要映射）
    await expect((legacyApp.GetImageBackendInfo as () => Promise<{ backend: string }>)())
      .resolves.toEqual({ backend: 'comfyui' })
    // 未映射的 Gaea 前缀方法同样命中
    await expect((legacyApp.SavePastedImage as () => Promise<string>)()).resolves.toBe('C:/tmp/pasted.png')
  })

  it('两端都没有的方法返回 undefined（诚实失败，不编造实现）', async () => {
    const go = installFakeShell()
    const { initBridge } = await import('../bridge')
    initBridge()
    const legacyApp = go.app.App as Record<string, unknown>
    expect(legacyApp.NoSuchBindingAtAll).toBeUndefined()
  })

  it('api/image.readFileAsDataURL 走 bridge 映射路径（真机读图不再抛 not a function）', async () => {
    const go = installFakeShell()
    const { initBridge } = await import('../bridge')
    initBridge()
    const { readFileAsDataURL } = await import('../../../api/image')

    await expect(readFileAsDataURL('C:/tmp/art.png')).resolves.toBe('data:image/png;base64,C:/tmp/art.png')
    expect(go.app.OfficeB?.GaeaAttachmentDataURL).toHaveBeenCalledWith('C:/tmp/art.png')
  })
})
