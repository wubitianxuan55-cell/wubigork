import React from 'react'
import { Typography } from 'antd'
import { ThunderboltOutlined } from '@ant-design/icons'
import { useT } from '../../gaea/lib/i18n'

/** SettingsSection — 设置中心信息层卡片（v3-card 实底：霓虹标题条 + 分组卡片）
 * instant 标记设置项是否即时生效（统一视觉徽章，避免各面板文案风格不一）。
 * icon 为面板级图标（左侧 symbolic 小图标，低调点缀）。 */
const SettingsSection: React.FC<{ title: React.ReactNode; desc?: string; instant?: boolean; icon?: React.ReactNode; children: React.ReactNode; noMargin?: boolean }> = ({ title, desc, instant, icon, children, noMargin }) => {
  const t = useT()
  return (
  <div
    className="v3-card"
    style={{
      position: 'relative',
      borderRadius: 'var(--md-sys-radius-lg)',
      padding: '18px 20px',
      marginBottom: noMargin ? 0 : 16,
      overflow: 'hidden',
    }}
  >
    {/* 霓虹标题条 */}
    <div style={{
      display: 'flex', alignItems: 'center', gap: 8, marginBottom: 14,
    }}>
      <span style={{
        width: 3, height: 16, borderRadius: 2,
        background: 'var(--gaea-glow)',
        boxShadow: '0 0 8px var(--gaea-glow)',
      }} />
      {icon && (
        <span style={{
          display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
          width: 24, height: 24, borderRadius: 7, flexShrink: 0,
          background: 'color-mix(in srgb, var(--gaea-glow) 12%, transparent)',
          color: 'var(--md-sys-color-primary)', fontSize: 13,
        }}>{icon}</span>
      )}
      <Typography.Text strong style={{ fontSize: 14, color: 'var(--md-sys-color-text)' }}>
        {title}
      </Typography.Text>
      {instant && (
        <span style={{
          marginLeft: 'auto', display: 'inline-flex', alignItems: 'center', gap: 4,
          fontSize: 11, lineHeight: 1, padding: '4px 9px',
          borderRadius: 999, color: 'var(--md-sys-color-success)',
          border: '1px solid color-mix(in srgb, var(--md-sys-color-success) 30%, transparent)',
          background: 'color-mix(in srgb, var(--md-sys-color-success) 10%, transparent)',
          fontWeight: 500, whiteSpace: 'nowrap',
        }}>
          <ThunderboltOutlined aria-hidden="true" style={{ fontSize: 10 }} />
          <span>{t('settings.instantBadge')}</span>
        </span>
      )}
    </div>
    {desc && (
      <Typography.Text style={{
        color: 'var(--md-sys-color-text-secondary)', fontSize: 12,
        display: 'block', marginBottom: 12, lineHeight: 1.6,
      }}>
        {desc}
      </Typography.Text>
    )}
    {children}
  </div>
  )
}

export default SettingsSection
