// ForeshadowPanel.test.tsx — 伏笔登记表面板关键路径
// 覆盖：手工登记→全量写回→列表出现；状态流转写回 hinted；保存失败回滚提示；删除（confirm）。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go
vi.mock('../../../src/wailsjsCompat', () => ({
  GetForeshadows: vi.fn().mockResolvedValue({ items: [] }),
  SaveForeshadows: vi.fn().mockResolvedValue(undefined),
}))

import ForeshadowPanel from './ForeshadowPanel'
import { GetForeshadows, SaveForeshadows } from '../../../src/wailsjsCompat'
import type { ForeshadowItemData } from '../../types'

const EXISTING: ForeshadowItemData = {
  id: 'plot_001_abc',
  category: 'character',
  description: '主角左臂旧伤',
  planted_in: '001.md',
  status: 'planted',
  is_long_term: false,
}

describe('ForeshadowPanel 手工登记闭环', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(GetForeshadows).mockResolvedValue({ items: [EXISTING] })
    vi.mocked(SaveForeshadows).mockResolvedValue(undefined)
  })

  it('登记伏笔：提交后 SaveForeshadows 全量写回（保留既有条目 + manual_ 新条目），列表出现', async () => {
    render(<ForeshadowPanel />)
    expect(await screen.findByText('主角左臂旧伤')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: /登记伏笔/ }))
    fireEvent.change(screen.getByPlaceholderText(/伏笔描述/), { target: { value: '神秘铜匣的钥匙' } })
    fireEvent.click(screen.getByRole('button', { name: /登记$/ }))

    await waitFor(() => expect(SaveForeshadows).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(SaveForeshadows).mock.calls[0][0] as string) as ForeshadowItemData[]
    expect(payload).toHaveLength(2)
    // 既有 AI 条目原样保留（全量写回不丢数据）
    expect(payload[0]).toEqual(EXISTING)
    // 新条目：manual_ 前缀 ID、默认 planted、章节号补零
    expect(payload[1].id).toMatch(/^manual_\d+$/)
    expect(payload[1].status).toBe('planted')
    expect(payload[1].description).toBe('神秘铜匣的钥匙')
    expect(payload[1].category).toBe('plot')
    expect(payload[1].planted_in).toBe('001.md')
    expect(payload[1].is_long_term).toBe(false)

    // 列表出现新条目 + 成功提示
    expect(await screen.findByText('神秘铜匣的钥匙')).toBeTruthy()
    expect(await screen.findByText('伏笔已登记')).toBeTruthy()
  })

  it('描述为空时拒登记，不触发写回', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText('主角左臂旧伤')
    fireEvent.click(screen.getByRole('button', { name: /登记伏笔/ }))
    fireEvent.click(screen.getByRole('button', { name: /登记$/ }))
    expect(await screen.findByText('请填写伏笔描述')).toBeTruthy()
    expect(SaveForeshadows).not.toHaveBeenCalled()
  })

  it('状态流转：标记暗示 → 乐观更新徽标并写回 hinted', async () => {
    render(<ForeshadowPanel />)
    expect(await screen.findByText('主角左臂旧伤')).toBeTruthy()
    expect(screen.getByText('已埋设')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: '标记暗示' }))

    await waitFor(() => expect(SaveForeshadows).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(SaveForeshadows).mock.calls[0][0] as string) as ForeshadowItemData[]
    expect(payload[0].status).toBe('hinted')
    expect(await screen.findByText('已暗示')).toBeTruthy()
  })

  it('保存失败：回滚列表并提示', async () => {
    vi.mocked(SaveForeshadows).mockRejectedValueOnce(new Error('磁盘已满'))
    render(<ForeshadowPanel />)
    await screen.findByText('主角左臂旧伤')

    fireEvent.click(screen.getByRole('button', { name: '标记暗示' }))

    expect(await screen.findByText(/伏笔保存失败，已回滚/)).toBeTruthy()
    // 回滚后状态徽标恢复 planted
    expect(await screen.findByText('已埋设')).toBeTruthy()
  })

  it('删除：confirm 后全量写回剩余条目', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText('主角左臂旧伤')

    fireEvent.click(screen.getByRole('button', { name: '删除伏笔：主角左臂旧伤' }))
    // antd Popconfirm 双字按钮会插空格（「删 除」）
    fireEvent.click(await screen.findByRole('button', { name: /^删\s*除$/ }))

    await waitFor(() => expect(SaveForeshadows).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(SaveForeshadows).mock.calls[0][0] as string) as ForeshadowItemData[]
    expect(payload).toEqual([])
    expect(await screen.findByText(/还没有伏笔登记/)).toBeTruthy()
  })
})
