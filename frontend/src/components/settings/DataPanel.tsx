import React, { useCallback, useEffect, useState } from 'react'
import { Button, Space, Tag, Typography, message, Alert, Popconfirm } from 'antd'
import { DatabaseOutlined, CloudUploadOutlined, CloudDownloadOutlined, UndoOutlined, InboxOutlined, CheckCircleOutlined } from '@ant-design/icons'
import SettingsSection from './SettingsSection'
import * as App from '../../../src/wailsjsCompat'

/**
 * DataPanel — 数据可迁移（P4-3，个人使用收口 v2.20.0）
 *
 * 一键备份：打包 Hephaestus.db（记忆/知识/成本/语义向量）+ whisper_data（轻语/办公/角色库/聊天）
 * + 配置 + sessions 为 zip（SQLite VACUUM INTO 一致性快照，运行中安全）。
 * 从备份恢复：两阶段——校验解压 staging + 写 pending，重启 gaea 后自动应用（恢复前先备份当前数据）。
 */

interface BackupEntry {
  path: string
  abs: string
  size: number
  exists: boolean
  sqlite?: boolean
}

interface BackupInfo {
  data_root: string
  entries: BackupEntry[]
  total_bytes: number
  pending: boolean
  app_version?: string
  pending_zip?: string
  pending_at?: string
}

function fmtSize(n: number): string {
  if (!n || n <= 0) return '0 B'
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  if (n < 1024 * 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
}

export const DataPanel: React.FC = () => {
  const [info, setInfo] = useState<BackupInfo | null>(null)
  const [creating, setCreating] = useState(false)
  const [restoring, setRestoring] = useState(false)
  const [restoreResult, setRestoreResult] = useState<any>(null)

  const load = useCallback(async () => {
    try {
      const res: any = await App.GaeaDataBackupInfo()
      if (res && typeof res.data_root === 'string') setInfo(res as BackupInfo)
    } catch { /* 后端未就绪 */ }
    try {
      const rr: any = await App.GaeaDataBackupRestoreResult()
      if (rr?.has_result) setRestoreResult(rr)
    } catch { /* 忽略 */ }
  }, [])

  useEffect(() => { void load() }, [load])

  const handleBackup = async () => {
    setCreating(true)
    try {
      const destDir: string = await App.GaeaPickDirectory()
      if (!destDir) return // 用户取消
      const res: any = await App.GaeaDataBackupCreate(destDir)
      message.success(`备份完成：${res.zip_path}（${fmtSize(res.total_bytes)}）`)
      await load()
    } catch (err: any) {
      message.error(err?.message || '备份失败')
    } finally {
      setCreating(false)
    }
  }

  const handleRestore = async () => {
    setRestoring(true)
    try {
      const files: { path: string; name: string; size: number }[] = await App.GaeaPickFiles()
      const zip = files.find((f) => f.name.toLowerCase().endsWith('.zip')) || files[0]
      if (!zip) return
      const res: any = await App.GaeaDataBackupRestore(zip.path)
      message.success(`恢复包已就绪（${res.zip_name}）。请重启 gaea 完成恢复——重启时会先自动备份当前数据。`)
      await load()
    } catch (err: any) {
      message.error(err?.message || '恢复失败')
    } finally {
      setRestoring(false)
    }
  }

  const handleCancelPending = async () => {
    try {
      await App.GaeaDataBackupCancel()
      message.success('已取消待应用恢复')
      await load()
    } catch (err: any) {
      message.error(err?.message || '取消失败')
    }
  }

  return (
    <>
      {/* 恢复结果提示（重启后） */}
      {restoreResult?.has_result && (
        restoreResult.applied ? (
          <Alert
            type="success"
            showIcon
            icon={<CheckCircleOutlined />}
            message={`数据恢复已应用（${restoreResult.zip_name || ''}）`}
            description={`恢复前数据已备份至 ${restoreResult.before_dir || ''}。${restoreResult.applied_at ? '生效时间 ' + restoreResult.applied_at : ''}`}
            style={{ marginBottom: 16, borderRadius: 'var(--md-sys-radius-md)' }}
            closable
          />
        ) : (
          <Alert
            type="warning"
            showIcon
            message="数据恢复失败"
            description={`${restoreResult.error || '未知错误'}。恢复任务已保留，可重试或取消。`}
            style={{ marginBottom: 16, borderRadius: 'var(--md-sys-radius-md)' }}
          />
        )
      )}

      <SettingsSection
        icon={<DatabaseOutlined />}
        title="数据备份"
        desc="一键备份全部本地数据：记忆/知识/成本库（Hephaestus.db）、轻语/办公/角色库/聊天（whisper_data）、配置与会话。SQLite 使用一致性快照，运行中备份安全。建议换机、重装或大版本升级前执行。"
      >
        <Space direction="vertical" size={10} style={{ width: '100%' }}>
          {info && (
            <div style={{
              display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap',
              padding: '10px 12px', borderRadius: 'var(--md-sys-radius-md)',
              background: 'var(--md-sys-color-surface-container)',
            }}>
              <InboxOutlined style={{ color: 'var(--md-sys-color-text-secondary)' }} />
              <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>
                数据根：
              </Typography.Text>
              <Typography.Text code style={{ fontSize: 11 }}>{info.data_root}</Typography.Text>
              <Tag color="blue" style={{ marginLeft: 4 }}>{fmtSize(info.total_bytes)}</Tag>
            </div>
          )}

          {/* 数据清单 */}
          {info?.entries?.some((e) => e.exists) && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              {info.entries.filter((e) => e.exists).map((e) => (
                <div key={e.path} style={{
                  display: 'flex', alignItems: 'center', gap: 8,
                  padding: '6px 10px', borderRadius: 8,
                  background: 'var(--md-sys-color-surface-container-high)',
                }}>
                  {e.sqlite ? <DatabaseOutlined style={{ fontSize: 12, color: 'var(--md-sys-color-primary)' }} /> : <InboxOutlined style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }} />}
                  <Typography.Text style={{ fontSize: 12, flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {e.path}
                  </Typography.Text>
                  <Typography.Text style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>{fmtSize(e.size)}</Typography.Text>
                  {e.sqlite && <Tag style={{ margin: 0, fontSize: 10 }}>SQLite</Tag>}
                </div>
              ))}
            </div>
          )}

          <Space wrap>
            <Button type="primary" icon={<CloudUploadOutlined />} loading={creating} onClick={handleBackup}>
              一键备份…
            </Button>
            <Button icon={<CloudDownloadOutlined />} loading={restoring} onClick={handleRestore}>
              从备份恢复…
            </Button>
          </Space>

          {info?.pending && (
            <Alert
              type="warning"
              showIcon
              message={`有待应用的恢复（${info.pending_zip || ''}，${info.pending_at || ''}）`}
              description="重启 gaea 后自动应用。如需放弃本次恢复，可取消。"
              action={(
                <Popconfirm title="放弃本次恢复？" okText="放弃" cancelText="继续恢复" onConfirm={handleCancelPending}>
                  <Button size="small" danger icon={<UndoOutlined />}>取消恢复</Button>
                </Popconfirm>
              )}
            />
          )}
        </Space>
      </SettingsSection>

      <SettingsSection
        icon={<CloudDownloadOutlined />}
        title="恢复说明（个人使用）"
        desc="gaea 个人使用，数据全部在本机。恢复会先自动备份当前数据到 .restore-before-<时间>，再替换——即使恢复失败也可找回原数据。换机迁移：把备份 zip 复制到新机 → 恢复 → 重启。注意：API 凭证（DPAPI 加密）跨机器不可解密，需在新机重新填写。"
      >
        <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', lineHeight: 1.7, display: 'block' }}>
          备份内容：Hephaestus.db（记忆/知识/成本/语义向量）、whisper_data（hermes.db、办公资料、角色库、聊天）、config.toml、sessions、home 配置。
        </Typography.Text>
      </SettingsSection>
    </>
  )
}

export default DataPanel
