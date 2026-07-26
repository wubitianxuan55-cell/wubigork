import React from 'react'
import { Card, Typography } from 'antd'
import { C } from '../utils/theme'

interface SettingsCardProps {
  title: React.ReactNode
  children: React.ReactNode
  noMargin?: boolean
}

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-glass)',
  backdropFilter: 'blur(8px)',
  WebkitBackdropFilter: 'blur(8px)',
  border: '1px solid var(--border-subtle)',
  borderRadius: 'var(--radius-lg)',
  boxShadow: 'var(--shadow-sm)',
  marginBottom: 24,
}

/** SettingsCard — 设置面板的统一卡片包装 */
const SettingsCard: React.FC<SettingsCardProps> = ({ title, children, noMargin }) => (
  <Card style={noMargin ? { ...cardStyle, marginBottom: 0 } : cardStyle}>
    <Typography.Title level={5} style={{ color: C('color-text'), marginTop: 0 }}>
      {title}
    </Typography.Title>
    {children}
  </Card>
)

export default SettingsCard
