import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { PortraitImg } from './PortraitImg'

const { attachmentDataURLMock } = vi.hoisted(() => ({
  attachmentDataURLMock: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', () => ({
  app: { AttachmentDataURL: attachmentDataURLMock },
}))

describe('PortraitImg 剧照渲染', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('远程 URL 加载失败时显示占位首字而不是裂图', () => {
    render(<PortraitImg src="https://imgen.x.ai/expired.jpeg" alt="秦挽月" />)
    const img = screen.getByRole('img', { name: '秦挽月' }) as HTMLImageElement
    fireEvent.error(img)
    expect(screen.getByText('秦')).toBeTruthy()
    expect(screen.queryByRole('img')).toBeNull()
  })

  it('本地文件路径经 AttachmentDataURL 读取后渲染', async () => {
    attachmentDataURLMock.mockResolvedValue('data:image/png;base64,AAA')
    render(<PortraitImg src="C:/x/portraits/c1.png" alt="陈恪" />)
    await waitFor(() => expect(attachmentDataURLMock).toHaveBeenCalledWith('C:/x/portraits/c1.png'))
    const img = (await screen.findByRole('img', { name: '陈恪' })) as HTMLImageElement
    expect(img.src).toContain('data:image/png')
  })
})
