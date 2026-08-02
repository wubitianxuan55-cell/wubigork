import React from 'react'
import { Typography } from 'antd'

/** SettingsSection — 设置中心玻璃区块（未来感：霓虹标题条 + 玻璃卡片）
 * instant 标记设置项是否即时生效（统一视觉徽章，避免各面板文案风格不一）。 */
const SettingsSection: React.FC<{ title: React.ReactNode; desc?: string; instant?: boolean; children: React.ReactNode; noMargin?: boolean }> = ({ title, desc, instant, children, noMargin }) => (
  <div
    className="md-glass"
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
      <Typography.Text strong style={{ fontSize: 14, color: 'var(--md-sys-color-text)' }}>
        {title}
      </Typography.Text>
      {instant && (
        <span style={{
          marginLeft: 'auto', fontSize: 11, lineHeight: 1, padding: '3px 8px',
          borderRadius: 999, color: '#34d399', border: '1px solid #34d39944',
          background: '#34d39914', fontWeight: 500, whiteSpace: 'nowrap',
        }}>
          ⚡ 即时生效
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

export default SettingsSection
