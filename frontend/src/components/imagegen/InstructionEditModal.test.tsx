// 指令编辑弹窗（阶段一刀 C）：指令生成（mode=edit 透传 initImage/prompt）/
// 原图新图对照 / 用到画布回调 / 错误态 / 空指令守卫。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import type { Mock } from 'vitest'

const mocks = vi.hoisted(() => ({
  generateMedia: vi.fn(),
}))

vi.mock('../../api/image', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../api/image')>()
  return {
    ...actual,
    generateMedia: mocks.generateMedia,
  }
})

import InstructionEditModal from './InstructionEditModal'
import type { GenResult } from './types'

const SOURCE: GenResult = {
  image: 'data:image/png;base64,SOURCE',
  seed: 1, time: 1.2, prompt: '原图描述', model: 'qwen-image', size: '1024x1024',
}

const EDITED: GenResult = {
  image: 'data:image/png;base64,EDITED',
  seed: 0, time: 2.5, prompt: '【编辑】把外套改成红色', model: 'qwen-image-edit', size: '1024x1024',
}

const open = (onApply = vi.fn()) =>
  render(<InstructionEditModal open source={SOURCE} onClose={vi.fn()} onApply={onApply} />)

describe('InstructionEditModal 指令编辑（阶段一刀 C）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('生成编辑：mode=edit 透传 initImage 与指令；对照区出结果与 meta', async () => {
    mocks.generateMedia.mockResolvedValue({ results: [EDITED], mode: 'edit' })
    open()
    expect(screen.getByAltText('原图')).toBeTruthy()
    fireEvent.change(screen.getByTestId('instruct-edit-input'), { target: { value: '把外套改成红色' } })
    fireEvent.click(screen.getByTestId('instruct-edit-run'))
    await waitFor(() => expect(mocks.generateMedia).toHaveBeenCalledTimes(1))
    const params = (mocks.generateMedia as Mock).mock.calls[0][0]
    expect(params.mode).toBe('edit')
    expect(params.initImage).toBe(SOURCE.image)
    expect(params.prompt).toBe('把外套改成红色')
    await waitFor(() => expect(screen.getByTestId('instruct-edit-result')).toBeTruthy())
    expect(screen.getByTestId('instruct-edit-meta').textContent).toContain('qwen-image-edit')
  })

  it('用到画布：onApply 收编辑结果（prompt 带【编辑】前缀）并关闭', async () => {
    mocks.generateMedia.mockResolvedValue({ results: [EDITED], mode: 'edit' })
    const onApply = vi.fn()
    open(onApply)
    fireEvent.change(screen.getByTestId('instruct-edit-input'), { target: { value: '改成夜景' } })
    fireEvent.click(screen.getByTestId('instruct-edit-run'))
    await waitFor(() => expect(screen.getByTestId('instruct-edit-apply')).toBeTruthy())
    fireEvent.click(screen.getByTestId('instruct-edit-apply'))
    expect(onApply).toHaveBeenCalledTimes(1)
    expect((onApply as Mock).mock.calls[0][0].prompt).toContain('【编辑】')
    // 弹窗 open 由父级控制（onClose 已回调、父级置源为 null 才真关）——
    // 此处不断言内容卸载，交由 ImageGenPage 接线语义。
  })

  it('错误态：generateMedia 报 error 展示（不静默）', async () => {
    mocks.generateMedia.mockResolvedValue({ error: 'GLM 图片端点暂不支持指令编辑' })
    open()
    fireEvent.change(screen.getByTestId('instruct-edit-input'), { target: { value: '改' } })
    fireEvent.click(screen.getByTestId('instruct-edit-run'))
    await waitFor(() => expect(screen.getByTestId('instruct-edit-error').textContent).toContain('暂不支持指令编辑'))
  })

  it('空指令：不触发生成', () => {
    open()
    fireEvent.click(screen.getByTestId('instruct-edit-run'))
    expect(mocks.generateMedia).not.toHaveBeenCalled()
  })

  it('说明文案：本地 ComfyUI 档已支持（阶段二刀 A），缺权重行为如实描述', () => {
    open()
    const hint = screen.getByTestId('instruct-edit-hint').textContent ?? ''
    expect(hint).toContain('Qwen-Image-Edit 2511')
    expect(hint).toContain('缺失时错误会列出所需文件')
    expect(hint).toContain('GLM')
  })
})
