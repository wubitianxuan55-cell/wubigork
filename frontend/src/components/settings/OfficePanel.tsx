import React, { useEffect, useState } from 'react'
import { Button, Tag, Typography, message, Space } from 'antd'
import { ApiOutlined, CheckCircleOutlined, RightOutlined, SettingOutlined, RobotOutlined } from '@ant-design/icons'
import { gaeaSettings } from '../../api/settings'
import SettingsSection from './SettingsSection'

/** OfficePanel — 办公引擎设置摘要（完整配置在办公模块内） */
const OfficePanel: React.FC = () => {
  const [view, setView] = useState<Record<string, any>>({})
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    gaeaSettings().then((v) => setView(v || {})).catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const providers: any[] = view.providers || []
  const agent: Record<string, any> = view.agent || {}

  const goOffice = () => {
    try {
      window.dispatchEvent(new CustomEvent('navigate', { detail: { page: 'office' } }))
    } catch {
      message.info('请通过顶栏进入「办公」模块')
    }
  }

  return (
    <>
      <SettingsSection
        title={<>默认办公模型</>}
        desc="办公套件（文档撰写 / 方案编写 / 智能体）使用的默认模型。"
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <RobotOutlined style={{ fontSize: 16, color: 'var(--gaea-glow)', filter: 'drop-shadow(0 0 6px var(--gaea-glow))' }} />
          <Typography.Text strong style={{ fontSize: 14, color: 'var(--gaea-glow)' }}>
            {view.defaultModel || '未配置'}
          </Typography.Text>
        </div>
      </SettingsSection>

      <SettingsSection
        title={<>引擎 Provider</>}
        desc="办公智能体可用的模型提供方（来自模型中心启用的引擎）。"
      >
        {loading ? (
          <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 12 }}>加载中…</Typography.Text>
        ) : providers.length === 0 ? (
          <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 12 }}>暂无可用 Provider</Typography.Text>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {providers.map((p) => (
              <div key={p.name} style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12 }}>
                <ApiOutlined style={{ color: 'var(--gaea-glow)' }} />
                <Typography.Text style={{ color: 'var(--md-sys-color-text)' }}>{p.name}</Typography.Text>
                <Tag style={{ fontSize: 10, margin: 0 }}>{p.kind}</Tag>
                {p.keySet && <CheckCircleOutlined style={{ color: '#4ade80', fontSize: 11 }} />}
                <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 10, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {p.baseURL}
                </Typography.Text>
              </div>
            ))}
          </div>
        )}
      </SettingsSection>

      <SettingsSection
        title={<>智能体参数</>}
        desc="办公智能体的运行参数（由 ~/.config/gaea/config.toml 管理）。"
      >
        <div style={{ display: 'flex', gap: 18, flexWrap: 'wrap' }}>
          <Space size={6}>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>温度</Typography.Text>
            <Typography.Text strong style={{ fontSize: 13, color: 'var(--gaea-glow)' }}>{agent.temperature ?? '-'}</Typography.Text>
          </Space>
          <Space size={6}>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>最大步骤</Typography.Text>
            <Typography.Text strong style={{ fontSize: 13, color: 'var(--gaea-glow)' }}>{agent.maxSteps ?? '-'}</Typography.Text>
          </Space>
          <Space size={6}>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>推理强度</Typography.Text>
            <Typography.Text strong style={{ fontSize: 13, color: 'var(--gaea-glow)' }}>{agent.effort || 'medium'}</Typography.Text>
          </Space>
          <Space size={6}>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>配置路径</Typography.Text>
            <Typography.Text style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>{view.configPath || '-'}</Typography.Text>
          </Space>
        </div>
      </SettingsSection>

      <SettingsSection
        title={<>办公模块</>}
        desc="权限、沙箱、知识库与完整 Provider 配置在办公模块内管理。"
      >
        <Button icon={<SettingOutlined />} onClick={goOffice} style={{ borderRadius: 'var(--md-sys-radius-md)' }}>
          前往办公设置 <RightOutlined style={{ fontSize: 10 }} />
        </Button>
      </SettingsSection>
    </>
  )
}

export default OfficePanel
