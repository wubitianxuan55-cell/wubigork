import React, { useState, useEffect } from 'react'
import { Typography, Card, Descriptions, Radio, Space, Button, Input, message } from 'antd'
import type { RadioChangeEvent } from 'antd'
import { FileMarkdownOutlined, FolderOpenOutlined } from '@ant-design/icons'
import { useAppStore, type ThemePreset } from '../stores/appStore'
import { C } from '../utils/theme'
import SkillModal from '../components/SkillModal'
import SettingsCard from '../components/SettingsCard'
import {
  getConfig,
  migrateProjectToV4,
} from '../api/settings'

const themeLabels: Record<ThemePreset, { label: string; desc: string; color: string }> = {
  nightJade:   { label: '🌊 暗夜青',  desc: '深海翡翠、冷静专注',   color: '#2dd4bf' },
  nightViolet: { label: '💜 暗夜紫',  desc: '深靛星云、灵感涌动',   color: '#a78bfa' },
  nightRose:   { label: '🌹 暗夜玫',  desc: '深褐暖调、温情创作',   color: '#fb7185' },
  nightAmber:  { label: '🌅 暗夜金',  desc: '深色暖灯、沉浸舒适',   color: '#f59e0b' },
  nightMoss:   { label: '🌿 暗夜苔',  desc: '深色林间、自然舒适',   color: '#84cc16' },
  nightSlate:  { label: '◻ 暗夜墨',  desc: '中性深灰、极简克制',   color: '#94a3b8' },
}

const SettingsPage: React.FC = () => {
  const [config, setConfig] = useState<Record<string, string>>({})
  const { baseTheme, setTheme, novelsDir, setNovelsDir } = useAppStore()
  const [skillOpen, setSkillOpen] = useState(false)
  const [wsDir, setWsDir] = useState(novelsDir)
  const [wsSaving, setWsSaving] = useState(false)

  useEffect(() => { loadConfig() }, [])

  useEffect(() => {
    setWsDir(novelsDir)
  }, [novelsDir])

  const loadConfig = async () => {
    try {
      const cfg = await getConfig()
      setConfig(cfg || {})
    } catch (err) { console.error('[SettingsPage] loadConfig:', err) }
  }

  const handleSaveWorkspace = async () => {
    const dir = wsDir.trim()
    if (!dir) { message.warning('请输入工作空间路径'); return }
    setWsSaving(true)
    try {
      await setNovelsDir(dir)
      message.success('工作空间已更新')
    } catch (err: any) { message.error(err?.message || '保存失败') }
    finally { setWsSaving(false) }
  }

  const handleMigrate = async () => {
    try {
      await migrateProjectToV4()
      message.success('项目已升级到 v4.0')
    } catch (err: any) { message.error(err?.message || '升级失败') }
  }

  return (
    <div>
      <Typography.Title level={4} style={{ color: C('color-text') }}>⚙️ 设置</Typography.Title>

      {/* 主题配色 */}
      <SettingsCard title="主题配色">
        <Radio.Group value={baseTheme} onChange={(e: RadioChangeEvent) => setTheme(e.target.value as ThemePreset)}>
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            {(Object.keys(themeLabels) as ThemePreset[]).map((key) => {
              const t = themeLabels[key]
              return (
                <Radio key={key} value={key} style={{ color: C('color-text') }}>
                  <span style={{ marginRight: 8, fontSize: 14 }}>{t.label}</span>
                  <span style={{ color: C('color-text-secondary'), fontSize: 12 }}>{t.desc}</span>
                  <span style={{
                    display: 'inline-block', width: 12, height: 12, borderRadius: '50%',
                    background: t.color, marginLeft: 8, verticalAlign: 'middle',
                  }} />
                </Radio>
              )
            })}
          </Space>
        </Radio.Group>
      </SettingsCard>

      {/* Skill */}
      <SettingsCard title="写作风格 (Skill)">
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, display: 'block', marginBottom: 12 }}>
          Skill 是写作风格指导文件，AI 写作时会注入到 prompt 中影响文风。
        </Typography.Text>
        <Button icon={<FileMarkdownOutlined />} onClick={() => setSkillOpen(true)}
          style={{ background: 'var(--bg-elevated)', border: '1px solid rgba(96, 165, 250, 0.3)', color: '#60a5fa', borderRadius: 'var(--radius-md)' }}>
          管理 Skill
        </Button>
      </SettingsCard>

      {/* 工作空间 */}
      <SettingsCard title={<><FolderOpenOutlined style={{ marginRight: 8 }} />工作空间</>}>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, display: 'block', marginBottom: 12 }}>
          小说存储目录。修改后书架将刷新到新路径。
        </Typography.Text>
        <Space.Compact style={{ width: '100%', flexDirection: 'row' }}>
          <Input placeholder="例如: D:\AI\xiaoshuo" value={wsDir}
            onChange={(e) => setWsDir(e.target.value)}
            style={{ background: 'rgba(255,255,255,0.05)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)' }} />
          <Button type="primary" onClick={handleSaveWorkspace} loading={wsSaving}
            style={{ boxShadow: 'var(--shadow-glow)', borderRadius: 'var(--radius-md)' }}>保存</Button>
        </Space.Compact>
      </SettingsCard>

      {/* 系统信息 */}
      <SettingsCard title="系统信息" noMargin>
        <Descriptions column={1} labelStyle={{ color: C('color-text-secondary') }} contentStyle={{ color: C('color-text') }}>
          <Descriptions.Item label="引擎">{config.model || '-'}</Descriptions.Item>
          <Descriptions.Item label="API 地址">{config.baseURL || '-'}</Descriptions.Item>
          <Descriptions.Item label="Token 存储">{config.tokenPath || '-'}</Descriptions.Item>
        </Descriptions>
      </SettingsCard>

      <SkillModal open={skillOpen} onClose={() => setSkillOpen(false)} />

      {/* v4.0 项目迁移 */}
      <Card size="small" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-lg)', marginTop: 24 }}
        title={<span>🧬 v4.0 项目升级</span>}>
        <Typography.Paragraph type="secondary" style={{ fontSize: 12, marginBottom: 8 }}>
          将当前项目升级到 v4.0 结构（非破坏性迁移，原始文件备份到 _v3_backup/）。
          升级后可享受场景编辑、快照版本、AI协写等新功能。
        </Typography.Paragraph>
        <Button onClick={handleMigrate}>升级到 v4.0</Button>
      </Card>
    </div>
  )
}

export default SettingsPage
