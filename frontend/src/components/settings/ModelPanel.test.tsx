/**
 * ModelPanel.test.tsx — T6-6.2 汇率输入收尾测试
 *
 * 保存路径调用 setUsdCnyRate 且传正值；非正数仅提示校验错误、不调用保存。
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../../api/settings', () => ({
  getActiveModel: vi.fn(),
  getConfig: vi.fn(),
  saveConfig: vi.fn(),
}))

vi.mock('../../api/engines', () => ({
  getUsdCnyRate: vi.fn(),
  setUsdCnyRate: vi.fn(),
}))

import ModelPanel from './ModelPanel'
import { getActiveModel, getConfig } from '../../api/settings'
import { getUsdCnyRate, setUsdCnyRate } from '../../api/engines'

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getActiveModel).mockResolvedValue('')
  vi.mocked(getConfig).mockResolvedValue({})
  vi.mocked(getUsdCnyRate).mockResolvedValue(7.2)
  vi.mocked(setUsdCnyRate).mockResolvedValue(undefined)
})

describe('ModelPanel 汇率输入（T6-6.2）', () => {
  it('加载时回填汇率，保存调用 setUsdCnyRate 且传正值', async () => {
    render(<ModelPanel />)
    // 等加载回填完成（precision=2 显示 7.20），避免异步回填覆盖后续输入
    const input = await screen.findByDisplayValue('7.20')
    fireEvent.change(input, { target: { value: '8' } })
    fireEvent.click(screen.getByText('保存汇率'))
    await waitFor(() => {
      expect(setUsdCnyRate).toHaveBeenCalledWith(8)
    })
  })

  it('空值/非正数不调用 setUsdCnyRate（校验拦截并提示，不静默）', async () => {
    render(<ModelPanel />)
    const input = await screen.findByDisplayValue('7.20')
    // antd InputNumber 对输入中的越界值会钳制，这里用清空（→ null）触发校验拦截
    fireEvent.change(input, { target: { value: '' } })
    fireEvent.click(screen.getByText('保存汇率'))
    await waitFor(() => {
      expect(setUsdCnyRate).not.toHaveBeenCalled()
    })
    // 错误提示不静默：向用户展示校验原因
    expect(await screen.findByText('汇率必须是大于 0 的数字')).toBeTruthy()
  })
})
