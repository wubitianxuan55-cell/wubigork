// FE4-12 回归防线：附件落盘失败不再被空 catch 吞掉。
// 此前三处 catch {} 只归零 pendingPaste：用户拖入大文件后界面毫无变化，无法
// 诊断是后端拒绝、体积超限还是读失败。现在失败一律走 toast（warn）可见上报；
// handlePickFiles 只在「绑定不存在」（typeof app.PickFiles !== 'function'）时静默。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, renderHook, screen, waitFor } from '@testing-library/react'
import type { ReactElement, ReactNode } from 'react'
import { useComposerAttachments } from './useComposerAttachments'
import { ToastProvider } from '../components/Toast'

const bridgeMocks = vi.hoisted(() => ({
  SavePastedImage: vi.fn(),
  SaveAttachmentFile: vi.fn(),
  PickFiles: vi.fn(),
}))

vi.mock('../lib/bridge', () => ({
  app: {
    SavePastedImage: bridgeMocks.SavePastedImage,
    SaveAttachmentFile: bridgeMocks.SaveAttachmentFile,
    PickFiles: bridgeMocks.PickFiles,
  },
}))

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <ToastProvider>{children}</ToastProvider>
}

function renderAttachments() {
  return renderHook(() => useComposerAttachments({
    text: '',
    setText: vi.fn(),
    taRef: { current: null } as React.RefObject<HTMLTextAreaElement>,
    running: false,
    onSend: vi.fn(),
  }), { wrapper })
}

const pngFile = (name: string) => new File([new Uint8Array([1, 2, 3])], name, { type: 'image/png' })
const binFile = (name: string) => new File([new Uint8Array([4, 5, 6])], name, { type: 'application/octet-stream' })

describe('useComposerAttachments 附件失败可见化（FE4-12）', () => {
  afterEach(() => { cleanup(); vi.clearAllMocks() })

  it('图片附件落盘失败 → toast 出现失败原因与文件名', async () => {
    bridgeMocks.SavePastedImage.mockRejectedValue(new Error('后端拒绝：体积超限'))
    const { result } = renderAttachments()

    await act(async () => { await result.current.attachDroppedFiles([pngFile('大截图.png')]) })

    await waitFor(() => expect(screen.getByText(/图片附件失败/)).toBeTruthy())
    const shown = screen.getByText(/图片附件失败/).textContent ?? ''
    expect(shown).toContain('大截图.png')
    expect(shown).toContain('体积超限')
    expect(result.current.attachments).toHaveLength(0)
    expect(result.current.pendingPaste).toBe(0)
  })

  it('普通文件落盘失败 → toast 可见', async () => {
    bridgeMocks.SaveAttachmentFile.mockRejectedValue(new Error('disk full'))
    const { result } = renderAttachments()

    await act(async () => { await result.current.attachDroppedFiles([binFile('数据.bin')]) })

    await waitFor(() => expect(screen.getByText(/文件附件失败/)).toBeTruthy())
    expect(screen.getByText(/文件附件失败/).textContent).toContain('数据.bin')
  })

  it('落盘成功时不出现任何失败提示（反向验证：不是恒弹）', async () => {
    bridgeMocks.SavePastedImage.mockResolvedValue('/ws/pasted.png')
    const { result } = renderAttachments()

    await act(async () => { await result.current.attachDroppedFiles([pngFile('正常.png')]) })

    expect(result.current.attachments).toHaveLength(1)
    expect(screen.queryByText(/附件失败/)).toBeNull()
  })

  it('handlePickFiles：绑定不存在（旧后端）静默，不弹提示', async () => {
    const app = (await import('../lib/bridge')).app as unknown as Record<string, unknown>
    const saved = app.PickFiles
    delete app.PickFiles
    try {
      const { result } = renderAttachments()
      await act(async () => { await result.current.handlePickFiles() })
      expect(screen.queryByText(/选择附件失败/)).toBeNull()
    } finally {
      app.PickFiles = saved
    }
  })

  it('handlePickFiles：绑定存在但调用失败 → 提示可见（不再借「旧后端不支持」静默）', async () => {
    bridgeMocks.PickFiles.mockRejectedValue(new Error('对话框不可用'))
    const { result } = renderAttachments()

    await act(async () => { await result.current.handlePickFiles() })

    await waitFor(() => expect(screen.getByText(/选择附件失败/)).toBeTruthy())
    expect(screen.getByText(/选择附件失败/).textContent).toContain('对话框不可用')
  })
})
