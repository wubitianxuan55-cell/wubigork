import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'

// 屏蔽 bridge 绑定（vi.hoisted 避免 mock 提升导致的初始化顺序问题）；
// DataPanel 经 ../../gaea/lib/bridge 的 app 调用，importOriginal 保全集 + 覆写 app。
const mocks = vi.hoisted(() => ({
  DataBackupInfo: vi.fn(),
  DataBackupRestoreResult: vi.fn(),
  DataBackupCreate: vi.fn(),
  DataBackupRestore: vi.fn(),
  DataBackupCancel: vi.fn(),
  DataBackupRollback: vi.fn(),
  PickDirectory: vi.fn(),
  PickFiles: vi.fn(),
}))
vi.mock('../../gaea/lib/bridge', async (importOriginal) => ({
  ...(await importOriginal()),
  app: mocks,
}))

import DataPanel from './DataPanel'
import { LocaleProvider } from '../../gaea/lib/i18n'

// S2.2b i18n：SettingsSection 徽章经 useT 读字典，测试需包 LocaleProvider（zh 为默认语言）
const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>

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
    mocks.DataBackupInfo.mockResolvedValue(baseInfo)
    mocks.DataBackupRestoreResult.mockResolvedValue({ has_result: false })
    mocks.DataBackupRollback.mockResolvedValue(true)
    // 断言为中文文案：固定 zh 语言（jsdom 默认 en-US）
    Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true })
  })

  it('渲染数据根与条目清单', async () => {
    render(wrap(<DataPanel />))
    expect(await screen.findByText(/数据根：/)).toBeTruthy()
    expect(screen.getAllByText(/Hephaestus.db/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/whisper_data/).length).toBeGreaterThan(0)
    expect(screen.getByText(/3.0 KB/)).toBeTruthy()
    expect(screen.getByRole('button', { name: /一键备份/ })).toBeTruthy()
    expect(screen.getByRole('button', { name: /从备份恢复/ })).toBeTruthy()
  })

  it('点击一键备份：选目录 → 创建 → 成功提示', async () => {
    mocks.PickDirectory.mockResolvedValue('D:\\backups')
    mocks.DataBackupCreate.mockResolvedValue({ zip_path: 'D:\\backups\\gaea-backup-2.20.0-20260814.zip', total_bytes: 3072 })
    render(wrap(<DataPanel />))
    fireEvent.click(await screen.findByRole('button', { name: /一键备份/ }))
    await waitFor(() => {
      expect(mocks.DataBackupCreate).toHaveBeenCalledWith('D:\\backups')
    })
    expect(await screen.findByText(/备份完成：D:\\backups\\gaea-backup/)).toBeTruthy()
  })

  it('从备份恢复：选 zip → 校验 → 提示重启', async () => {
    mocks.PickFiles.mockResolvedValue([{ path: 'D:\\bk\\a.zip', name: 'a.zip', size: 1000 }])
    mocks.DataBackupRestore.mockResolvedValue({ restart_required: true, zip_name: 'a.zip', backup_version: '2.20.0' })
    render(wrap(<DataPanel />))
    // Popconfirm 二次确认：先点「从备份恢复」打开确认，再点「选择备份文件」确认
    fireEvent.click(await screen.findByRole('button', { name: /从备份恢复/ }))
    fireEvent.click(await screen.findByRole('button', { name: /选择备份文件/ }))
    await waitFor(() => {
      expect(mocks.DataBackupRestore).toHaveBeenCalledWith('D:\\bk\\a.zip')
    })
    expect(await screen.findByText(/请重启 gaea 完成恢复/)).toBeTruthy()
  })

  it('有待应用恢复时显示告警与取消按钮', async () => {
    mocks.DataBackupInfo.mockResolvedValue({ ...baseInfo, pending: true, pending_zip: 'a.zip', pending_at: '2026-08-14 08:00:00' })
    render(wrap(<DataPanel />))
    expect(await screen.findByText(/有待应用的恢复/)).toBeTruthy()
    expect(screen.getByRole('button', { name: /取消恢复/ })).toBeTruthy()
  })

  // ── GA4-11 备份/回滚可见性（批次十三 round15 线3）────────────────────
  // 触发面：恢复结果 applied=false 的「数据恢复失败」告警里挂的回滚入口
  // （Popconfirm → handleRollback）。后端 RollbackBefore 部分失败时返回非 nil
  // error，前端必须按失败/部分失败显式呈现。
  const failedRestore = { has_result: true, applied: false, error: '写入失败', zip_name: 'a.zip' }

  async function clickRollback() {
    render(wrap(<DataPanel />))
    fireEvent.click(await screen.findByRole('button', { name: /回滚到恢复前/ }))
    // Popconfirm 的 okText 是两字中文，antd Button 会在两字之间插空格（"回 滚"），
    // 故正则容忍空白，同时用 ^$ 锚定避免误命中触发按钮「回滚到恢复前」。
    fireEvent.click(await screen.findByRole('button', { name: /^回\s*滚$/ }))
    await waitFor(() => expect(mocks.DataBackupRollback).toHaveBeenCalled())
  }

  it('回滚完整成功：成功提示，不出失败提示条', async () => {
    mocks.DataBackupRestoreResult.mockResolvedValue(failedRestore)
    mocks.DataBackupRollback.mockResolvedValue(true)
    await clickRollback()
    expect(await screen.findByText(/已回滚到恢复前数据/)).toBeTruthy()
    expect(screen.queryByText(/回滚失败/)).toBeNull()
    expect(screen.queryByTestId('settings-rollback-failure')).toBeNull()
  })

  it('回滚部分失败（返回 error）：按部分失败点名项数 + 原文，只出提示条不弹 modal', async () => {
    mocks.DataBackupRestoreResult.mockResolvedValue(failedRestore)
    mocks.DataBackupRollback.mockRejectedValue(
      new Error('回滚未完成：1/3 项失败（已回滚 2 项）: whisper_data: 移回失败: Access is denied.'),
    )
    await clickRollback()

    // 即时 toast（antd .ant-message 容器）+ 持久提示条两条通道都带后端原文。
    // 分容器断言（批 17）：findAllByText 拿到任一匹配即提前返回，全量满载下
    // toast 晚渲染会被漏数成 1 条（CI 实测红一次、单跑恒绿）；分容器各自
    // waitFor 无早退竞态，断言意图不变（两条通道都带后端原文）。
    const rawRe = /回滚未完成：1\/3 项失败（已回滚 2 项）/
    await waitFor(() => {
      expect(document.querySelector('.ant-message')?.textContent ?? '').toMatch(rawRe)
    })
    expect(within(screen.getByTestId('settings-rollback-failure')).getByText(rawRe)).toBeTruthy()
    // 持久提示条：点名「部分回滚失败 N/M 项」，并给出可重试的余量说明
    expect(await screen.findByText('部分回滚失败：1/3 项失败（已回滚 2 项）')).toBeTruthy()
    expect(screen.getByText(/恢复前数据仍保留在 \.restore-before/)).toBeTruthy()
    expect(screen.getAllByText(/whisper_data: 移回失败: Access is denied\./).length).toBeGreaterThan(0)
    // 不打断编辑：不得出现 modal
    expect(document.querySelector('.ant-modal')).toBeNull()
  })

  it('回滚整体失败（非计数形态 error）：通用「回滚失败」+ 原文，不出计数文案', async () => {
    mocks.DataBackupRestoreResult.mockResolvedValue(failedRestore)
    mocks.DataBackupRollback.mockRejectedValue(new Error('.restore-before 不是目录'))
    await clickRollback()
    expect(await screen.findByText('回滚失败')).toBeTruthy()
    expect(screen.getAllByText(/\.restore-before 不是目录/).length).toBeGreaterThan(0)
    expect(screen.queryByText(/部分回滚失败/)).toBeNull()
    expect(document.querySelector('.ant-modal')).toBeNull()
  })
})
