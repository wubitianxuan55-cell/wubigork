import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// 屏蔽 Wails 绑定（vi.hoisted 避免 mock 提升导致的初始化顺序问题）
const mocks = vi.hoisted(() => ({
  GaeaDataBackupInfo: vi.fn(),
  GaeaDataBackupRestoreResult: vi.fn(),
  GaeaDataBackupCreate: vi.fn(),
  GaeaDataBackupRestore: vi.fn(),
  GaeaDataBackupCancel: vi.fn(),
  GaeaPickDirectory: vi.fn(),
  GaeaPickFiles: vi.fn(),
}))
vi.mock('../../../src/wailsjsCompat', () => mocks)

import DataPanel from './DataPanel'

const baseInfo = {
  data_root: 'C:\\Users\\u\\AppData\\Roaming\\gaea',
  entries: [
    { path: 'Hephaestus.db', abs: 'C:\\x\\Hephaestus.db', size: 1024, exists: true, sqlite: true },
    { path: 'whisper_data', abs: 'C:\\x\\whisper_data', size: 2048, exists: true },
  ],
  total_bytes: 3072,
  pending: false,
  app_version: '2.20.0',
}

describe('DataPanel 数据备份/恢复', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.GaeaDataBackupInfo.mockResolvedValue(baseInfo)
    mocks.GaeaDataBackupRestoreResult.mockResolvedValue({ has_result: false })
  })

  it('渲染数据根与条目清单', async () => {
    render(<DataPanel />)
    expect(await screen.findByText(/数据根：/)).toBeTruthy()
    expect(screen.getAllByText(/Hephaestus.db/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/whisper_data/).length).toBeGreaterThan(0)
    expect(screen.getByText(/3.0 KB/)).toBeTruthy()
    expect(screen.getByRole('button', { name: /一键备份/ })).toBeTruthy()
    expect(screen.getByRole('button', { name: /从备份恢复/ })).toBeTruthy()
  })

  it('点击一键备份：选目录 → 创建 → 成功提示', async () => {
    mocks.GaeaPickDirectory.mockResolvedValue('D:\\backups')
    mocks.GaeaDataBackupCreate.mockResolvedValue({ zip_path: 'D:\\backups\\gaea-backup-2.20.0-20260814.zip', total_bytes: 3072 })
    render(<DataPanel />)
    fireEvent.click(await screen.findByRole('button', { name: /一键备份/ }))
    await waitFor(() => {
      expect(mocks.GaeaDataBackupCreate).toHaveBeenCalledWith('D:\\backups')
    })
    expect(await screen.findByText(/备份完成：D:\\backups\\gaea-backup/)).toBeTruthy()
  })

  it('从备份恢复：选 zip → 校验 → 提示重启', async () => {
    mocks.GaeaPickFiles.mockResolvedValue([{ path: 'D:\\bk\\a.zip', name: 'a.zip', size: 1000 }])
    mocks.GaeaDataBackupRestore.mockResolvedValue({ restart_required: true, zip_name: 'a.zip', backup_version: '2.20.0' })
    render(<DataPanel />)
    fireEvent.click(await screen.findByRole('button', { name: /从备份恢复/ }))
    await waitFor(() => {
      expect(mocks.GaeaDataBackupRestore).toHaveBeenCalledWith('D:\\bk\\a.zip')
    })
    expect(await screen.findByText(/请重启 gaea 完成恢复/)).toBeTruthy()
  })

  it('有待应用恢复时显示告警与取消按钮', async () => {
    mocks.GaeaDataBackupInfo.mockResolvedValue({ ...baseInfo, pending: true, pending_zip: 'a.zip', pending_at: '2026-08-14 08:00:00' })
    render(<DataPanel />)
    expect(await screen.findByText(/有待应用的恢复/)).toBeTruthy()
    expect(screen.getByRole('button', { name: /取消恢复/ })).toBeTruthy()
  })
})
