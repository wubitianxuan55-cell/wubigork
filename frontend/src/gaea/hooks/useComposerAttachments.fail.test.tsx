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
  RecognizeImage: vi.fn(),
  OCRText: vi.fn(),
}))

vi.mock('../lib/bridge', () => ({
  app: {
    SavePastedImage: bridgeMocks.SavePastedImage,
    SaveAttachmentFile: bridgeMocks.SaveAttachmentFile,
    PickFiles: bridgeMocks.PickFiles,
    RecognizeImage: bridgeMocks.RecognizeImage,
    OCRText: bridgeMocks.OCRText,
  },
}))

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <ToastProvider>{children}</ToastProvider>
}

function renderAttachments(over?: { setText?: ReturnType<typeof vi.fn>; onSend?: ReturnType<typeof vi.fn> }) {
  // vi.fn 过 hook 接口签名须 as 窄化（运行时仍是 mock 原件，调用方可断言）
  const setText = (over?.setText ?? vi.fn()) as unknown as React.Dispatch<React.SetStateAction<string>>
  const onSend = (over?.onSend ?? vi.fn()) as unknown as (d: string, s?: string) => void
  return renderHook(() => useComposerAttachments({
    text: '',
    setText,
    taRef: { current: null } as React.RefObject<HTMLTextAreaElement>,
    running: false,
    onSend,
  }), { wrapper })
}

const pngFile = (name: string) => new File([new Uint8Array([1, 2, 3])], name, { type: 'image/png' })
const binFile = (name: string) => new File([new Uint8Array([4, 5, 6])], name, { type: 'application/octet-stream' })
const sizedFile = (file: File, size: number) => {
  Object.defineProperty(file, 'size', { value: size })
  return file
}

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

  it('普通文件超过 25MB：FileReader 不启动、后端不调用、提示可见', async () => {
    const read = vi.spyOn(FileReader.prototype, 'readAsArrayBuffer')
    const { result } = renderAttachments()
    const tooLarge = sizedFile(binFile('oversize.bin'), 25 * 1024 * 1024 + 1)

    await act(async () => { await result.current.attachDroppedFiles([tooLarge]) })

    expect(read).not.toHaveBeenCalled()
    expect(bridgeMocks.SaveAttachmentFile).not.toHaveBeenCalled()
    expect(screen.getByText(/超过 25MB 上限/)).toBeTruthy()
    expect(result.current.pendingPaste).toBe(0)
    read.mockRestore()
  })

  it('图片超过 10MB：FileReader 不启动、后端不调用', async () => {
    const read = vi.spyOn(FileReader.prototype, 'readAsDataURL')
    const { result } = renderAttachments()
    const tooLarge = sizedFile(pngFile('oversize.png'), 10 * 1024 * 1024 + 1)

    await act(async () => { await result.current.attachDroppedFiles([tooLarge]) })

    expect(read).not.toHaveBeenCalled()
    expect(bridgeMocks.SavePastedImage).not.toHaveBeenCalled()
    expect(screen.getByText(/超过 10MB 上限/)).toBeTruthy()
    read.mockRestore()
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

describe('useComposerAttachments v4.508 文件交互修复', () => {
  afterEach(() => { cleanup(); vi.clearAllMocks() })

  it('PickFiles 选的图片（无 type 字段）按扩展名进 image 通道——bug 回归钉', async () => {
    // 复现契约现场：FilePickResult 只有 path/name/size，无 type 字段
    bridgeMocks.PickFiles.mockResolvedValue([
      { path: 'C:/imgs/现场照片.png', name: '现场照片.png', size: 5 },
      { path: 'C:/docs/报告.docx', name: '报告.docx', size: 7 },
    ])
    const { result } = renderAttachments()
    await act(async () => { await result.current.handlePickFiles() })
    const atts = result.current.attachments
    expect(atts).toHaveLength(2)
    expect(atts[0]).toMatchObject({ path: 'C:/imgs/现场照片.png', type: 'image' })
    expect(atts[1]).toMatchObject({ path: 'C:/docs/报告.docx', type: 'file' })
  })

  it('PickFiles 元数据超限：不加入附件且提示可见', async () => {
    bridgeMocks.PickFiles.mockResolvedValue([
      { path: 'C:/docs/huge.pdf', name: 'huge.pdf', size: 25 * 1024 * 1024 + 1 },
    ])
    const { result } = renderAttachments()
    await act(async () => { await result.current.handlePickFiles() })
    expect(result.current.attachments).toHaveLength(0)
    expect(screen.getByText(/超过 25MB 上限/)).toBeTruthy()
  })

  it('识图发送后输入框草稿保留（此前 setText("") 清用户半句话）', async () => {
    bridgeMocks.RecognizeImage.mockResolvedValue('图中是施工现场')
    const setText = vi.fn()
    const onSend = vi.fn()
    const { result } = renderAttachments({ setText, onSend })
    await act(async () => { await result.current.handleRecognize({ path: 'C:/imgs/a.png', previewUrl: '', type: 'image' }) })
    expect(onSend).toHaveBeenCalledTimes(1)
    expect(onSend.mock.calls[0][0]).toContain('图中是施工现场')
    expect(setText).not.toHaveBeenCalled()
    expect(result.current.attachments).toHaveLength(0)
  })

  it('重复路径附件去重（拖两次/选两次只留一条）', async () => {
    bridgeMocks.SavePastedImage.mockResolvedValue('.gaea/uploads/dup.png')
    const { result } = renderAttachments()
    await act(async () => { await result.current.attachDroppedFiles([pngFile('dup.png')]) })
    await act(async () => { await result.current.attachDroppedFiles([pngFile('dup.png')]) })
    expect(result.current.attachments).toHaveLength(1)
  })

  it('非图片附件 base64 分块编码等价（SaveAttachmentFile 收到正确 b64）', async () => {
    bridgeMocks.SaveAttachmentFile.mockResolvedValue('.gaea/uploads/data.bin')
    const { result } = renderAttachments()
    // binFile 内容 [4,5,6] → base64 BAUG（分块编码与整段 btoa 等价）
    await act(async () => { await result.current.attachDroppedFiles([binFile('data.bin')]) })
    expect(bridgeMocks.SaveAttachmentFile).toHaveBeenCalledWith('data.bin', 'BAUG')
    expect(result.current.attachments[0]).toMatchObject({ type: 'file', size: 3 })
  })
})
