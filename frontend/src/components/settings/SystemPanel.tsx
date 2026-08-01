import React, { useEffect, useState } from 'react'
import { Button, Descriptions, Typography, message } from 'antd'
import { DatabaseOutlined, ExperimentOutlined, RocketOutlined } from '@ant-design/icons'
import { getConfig, migrateProjectToV4 } from '../../api/settings'
import SettingsSection from './SettingsSection'

/** SystemPanel — 系统信息 + 数据迁移 */
const SystemPanel: React.FC = () => {
  const [config, setConfig] = useState<Record<string, string>>({})

  useEffect(() => {
    getConfig().then((cfg) => setConfig(cfg || {})).catch(() => {})
  }, [])

  const handleMigrate = async () => {
    try {
      await migrateProjectToV4()
      message.success('项目已升级到 v4.0')
    } catch (err: any) { message.error(err?.message || '升级失败') }
  }

  return (
    <>
      <SettingsSection
        title={<>系统信息</>}
        desc="当前引擎、API 与凭证存储路径。"
      >
        <Descriptions
          column={1}
          size="small"
          labelStyle={{ color: 'var(--md-sys-color-text-secondary)', width: 120 }}
          contentStyle={{ color: 'var(--md-sys-color-text)', wordBreak: 'break-all' }}
        >
          <Descriptions.Item label="引擎">{config.model || '-'}</Descriptions.Item>
          <Descriptions.Item label="API 地址">{config.baseURL || '-'}</Descriptions.Item>
          <Descriptions.Item label="Token 存储">{config.tokenPath || '-'}</Descriptions.Item>
          <Descriptions.Item label="推理强度">{config.reasoning_effort || '-'}</Descriptions.Item>
          <Descriptions.Item label="图片保存目录">{config.image_save_dir || '未配置'}</Descriptions.Item>
        </Descriptions>
      </SettingsSection>

      <SettingsSection
        title={<>数据与升级</>}
        desc="项目结构迁移与数据管理。"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div>
            <Typography.Text style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>
              <RocketOutlined style={{ color: 'var(--gaea-glow)', marginRight: 6 }} />v4.0 项目升级
            </Typography.Text>
            <Typography.Paragraph type="secondary" style={{ fontSize: 12, margin: '6px 0 8px' }}>
              将当前项目升级到 v4.0 结构（非破坏性迁移，原始文件备份到 _v3_backup/）。
              升级后可享受场景编辑、快照版本、AI 协写等新功能。
            </Typography.Paragraph>
            <Button icon={<ExperimentOutlined />} onClick={handleMigrate} style={{ borderRadius: 'var(--md-sys-radius-md)' }}>
              升级到 v4.0
            </Button>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8, opacity: 0.7 }}>
            <DatabaseOutlined style={{ color: 'var(--gaea-glow)' }} />
            <Typography.Text style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
              配置文件：~/.gaea_config.json · 办公配置：~/.config/gaea/config.toml
            </Typography.Text>
          </div>
        </div>
      </SettingsSection>
    </>
  )
}

export default SystemPanel
