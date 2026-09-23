// SinCastPanel 生成设定卡（v4.403，sin 侧入口）：chip 级一键三视图→存回角色库。
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { SinCastPanel } from './SinCastPanel'
import type { SinCastCharacter } from './useSinCast'

const cast: SinCastCharacter[] = [
  { id: 'c1', name: '林晚', portraitUrl: 'data:image/png;base64,P1' },
  { id: 'c2', name: '沈砚', portraitUrl: '' },
]

const setup = (onGenerateSheet: (id: string) => Promise<void>) =>
  render(
    <SinCastPanel
      cast={cast}
      saving={false}
      onOpenPicker={() => {}}
      onRemove={() => {}}
      onGenerateSheet={onGenerateSheet}
    />,
  )

describe('SinCastPanel 生成设定卡（sin 侧入口）', () => {
  it('chip 设定卡按钮：点击按角色 id 调用，成功后复位', async () => {
    const onGenerateSheet = vi.fn().mockResolvedValue(undefined)
    setup(onGenerateSheet)
    const btn = screen.getByLabelText('生成 林晚 的设定卡')
    fireEvent.click(btn)
    expect(onGenerateSheet).toHaveBeenCalledWith('c1')
    await waitFor(() => expect(btn.hasAttribute('disabled')).toBe(false))
  })

  it('单飞：进行中禁用其他 chip 的设定卡按钮；失败同样复位', async () => {
    let resolve1!: (v: undefined) => void
    const onGenerateSheet = vi
      .fn()
      .mockImplementationOnce(() => new Promise<undefined>((r) => (resolve1 = r)))
      .mockRejectedValueOnce(new Error('仅 ComfyUI 本地档'))
    setup(onGenerateSheet)
    const b1 = screen.getByLabelText('生成 林晚 的设定卡')
    const b2 = screen.getByLabelText('生成 沈砚 的设定卡')
    fireEvent.click(b1)
    await waitFor(() => expect(b2.hasAttribute('disabled')).toBe(true))
    resolve1(undefined)
    await waitFor(() => expect(b2.hasAttribute('disabled')).toBe(false))
    // 失败路径：调用后按钮复位（错误由页面层 message 透出）
    fireEvent.click(b2)
    await waitFor(() => expect(b2.hasAttribute('disabled')).toBe(false))
    expect(onGenerateSheet).toHaveBeenCalledTimes(2)
  })

  it('未传 onGenerateSheet：不渲染设定卡按钮（向后兼容）', () => {
    render(
      <SinCastPanel cast={cast} saving={false} onOpenPicker={() => {}} onRemove={() => {}} />,
    )
    expect(screen.queryByLabelText('生成 林晚 的设定卡')).toBeNull()
  })
})
