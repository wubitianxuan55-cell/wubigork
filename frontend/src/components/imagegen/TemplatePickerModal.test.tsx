import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import TemplatePickerModal from './TemplatePickerModal'

const baseProps = {
  open: true,
  onClose: vi.fn(),
  customTemplates: [],
  onSelect: vi.fn(),
  onAddCustom: vi.fn(),
  onEditCustom: vi.fn(),
  onDeleteCustom: vi.fn(),
}

describe('TemplatePickerModal 模板库重设计', () => {
  it('默认进入「全部模板」，herdsman 模板与分类 pill 都可见', async () => {
    render(<TemplatePickerModal {...baseProps} />)
    expect(screen.getByText('全部模板')).toBeTruthy()
    expect(screen.getByRole('tab', { name: /人像摄影/ })).toBeTruthy()
    expect(screen.getByRole('tab', { name: /UI 界面/ })).toBeTruthy()
    expect(await screen.findByText('沙发棚拍人像')).toBeTruthy()
  })

  it('切换 herdsman 分类后仍能浏览该分类模板', async () => {
    render(<TemplatePickerModal {...baseProps} />)
    fireEvent.click(screen.getByRole('tab', { name: /UI 界面/ }))
    expect(await screen.findByText('健康数据仪表盘')).toBeTruthy()
  })
})
