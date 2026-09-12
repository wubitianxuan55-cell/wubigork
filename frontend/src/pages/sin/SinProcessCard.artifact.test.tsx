// SinProcessCard 产物缩略图测试（v4.270）：sin_illustrate 轨迹的 artifacts
// 在展开明细里经附件通道转 data URL 渲染，caption 作图注；读取失败如实显示。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

const apiMock = vi.hoisted(() => ({
  readFileAsDataURL: vi.fn(),
}))

vi.mock('../../api/image', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/image')>()),
  readFileAsDataURL: apiMock.readFileAsDataURL,
}))

import { SinProcessCard } from './SinProcessCard'
import type { SinToolTraceView } from './types'

const TOOL: SinToolTraceView = {
  id: 'call_1',
  name: 'sin_illustrate',
  args: '{"prompt":"雨夜里的短发女人"}',
  output: '已生成插图并保存到：C:/art/x.png',
  elapsed_ms: 12000,
  read_only: false,
  status: 'done',
  artifacts: [{ kind: 'image', path: 'C:/art/x.png', caption: '雨夜' }],
}

function renderCard(tools: SinToolTraceView[]) {
  return render(<SinProcessCard tools={tools} reasoning="" running={false} />)
}

beforeEach(() => {
  apiMock.readFileAsDataURL.mockReset().mockResolvedValue('data:image/png;base64,AAA')
})

describe('过程卡产物缩略图', () => {
  it('sin_illustrate 展开后渲染产物缩略图（附件通道转 data URL）', async () => {
    renderCard([TOOL])
    // 卡头摘要即「生成插图」→ 先展开卡，再展开工具行明细
    fireEvent.click(screen.getByText('生成插图'))
    fireEvent.click(screen.getAllByText('生成插图').pop()!)
    await waitFor(() => expect(apiMock.readFileAsDataURL).toHaveBeenCalledWith('C:/art/x.png'))
    expect(await screen.findByAltText('雨夜')).toBeTruthy()
  })

  it('缩略图读取失败：如实显示失败占位，不留破图', async () => {
    apiMock.readFileAsDataURL.mockRejectedValue(new Error('读取失败'))
    renderCard([TOOL])
    fireEvent.click(screen.getByText('生成插图'))
    fireEvent.click(screen.getAllByText('生成插图').pop()!)
    expect(await screen.findByText('缩略图读取失败')).toBeTruthy()
  })
})
