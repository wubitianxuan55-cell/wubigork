// CostBackupCard.test.tsx — 概览「数据备份」卡测试。
// 覆盖：①体量速览渲染 ②取消目录选择不触发备份 ③备份成功回执（路径+大小+SHA）
// ④备份失败显形 ⑤体量读取失败不拦备份（按钮可用）。

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  DataBackupInfo: vi.fn(),
  PickDirectory: vi.fn(),
  DataBackupCreate: vi.fn(),
}))
vi.mock('../../lib/bridge', () => ({ app: mocks }))

import { CostBackupCard } from './CostBackupCard'

const LOAD = { timeout: 5000 }

beforeEach(() => {
  vi.clearAllMocks()
})

describe('CostBackupCard 数据备份卡', () => {
  it('渲染体量速览（DataBackupInfo.total_bytes）', async () => {
    mocks.DataBackupInfo.mockResolvedValue({ total_bytes: 262144000, entries: [], pending: false })
    render(<CostBackupCard />)
    expect(await screen.findByText(/库与数据共 250\.0 MB/, {}, LOAD)).toBeTruthy()
    expect(screen.getByText('立即备份')).toBeTruthy()
  })

  it('取消目录选择：不调用备份，按钮复位', async () => {
    mocks.DataBackupInfo.mockResolvedValue({ total_bytes: 100 })
    mocks.PickDirectory.mockResolvedValue('')
    render(<CostBackupCard />)
    const btn = await screen.findByText('立即备份', {}, LOAD)
    fireEvent.click(btn)
    await waitFor(() => expect(screen.getByText('立即备份')).toBeTruthy(), LOAD)
    expect(mocks.DataBackupCreate).not.toHaveBeenCalled()
  })

  it('备份成功：调 DataBackupCreate 并显示 zip 路径回执，体量刷新', async () => {
    mocks.DataBackupInfo.mockResolvedValueOnce({ total_bytes: 100 })
      .mockResolvedValueOnce({ total_bytes: 268435456 })
    mocks.PickDirectory.mockResolvedValue('D:\\backup')
    mocks.DataBackupCreate.mockResolvedValue({
      zip_path: 'D:\\backup\\gaea-backup.zip',
      total_bytes: 94371840,
      sha256: 'a1b2c3d4e5f6a7b8c9d0',
    })
    render(<CostBackupCard />)
    fireEvent.click(await screen.findByText('立即备份', {}, LOAD))
    expect(await screen.findByText(/备份完成：D:\\backup\\gaea-backup\.zip（90\.0 MB，SHA256 a1b2c3d4e5f6…）/u, {}, LOAD)).toBeTruthy()
    // 回执后体量用刷新值。
    await waitFor(() => expect(screen.getByText(/库与数据共 256\.0 MB/)).toBeTruthy(), LOAD)
    expect(mocks.DataBackupCreate).toHaveBeenCalledWith('D:\\backup')
  })

  it('备份失败：错误显形不装成功', async () => {
    mocks.DataBackupInfo.mockResolvedValue({ total_bytes: 100 })
    mocks.PickDirectory.mockResolvedValue('D:\\backup')
    mocks.DataBackupCreate.mockRejectedValue(new Error('磁盘已满'))
    render(<CostBackupCard />)
    fireEvent.click(await screen.findByText('立即备份', {}, LOAD))
    expect(await screen.findByText(/备份失败：磁盘已满/, {}, LOAD)).toBeTruthy()
  })

  it('体量读取失败：显示中性态，备份按钮仍可用', async () => {
    mocks.DataBackupInfo.mockRejectedValue(new Error('binding missing'))
    mocks.PickDirectory.mockResolvedValue('D:\\backup')
    mocks.DataBackupCreate.mockResolvedValue({ zip_path: 'D:\\backup\\x.zip', total_bytes: 10 })
    render(<CostBackupCard />)
    const btn = await screen.findByText('立即备份', {}, LOAD)
    expect(screen.queryByText(/库与数据共/)).toBeNull()
    fireEvent.click(btn)
    await waitFor(() => expect(mocks.DataBackupCreate).toHaveBeenCalled(), LOAD)
  })
})
