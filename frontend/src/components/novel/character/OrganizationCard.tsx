import React from 'react'
import { Typography, Card, Tag, Space } from 'antd'
import { ApartmentOutlined } from '@ant-design/icons'
import { C } from '../../../utils/theme'
import type { OrganizationData } from '../../../types'

export interface OrganizationCardProps {
  organization: OrganizationData
  onClick: () => void
}

const OrganizationCard: React.FC<OrganizationCardProps> = ({ organization, onClick }) => {
  const org = organization
  return (
    <Card size="small" hoverable onClick={onClick}
      style={{
        background: 'var(--bg-glass)', backdropFilter: 'blur(8px)', WebkitBackdropFilter: 'blur(8px)',
        border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-lg)', boxShadow: 'var(--shadow-sm)',
        cursor: 'pointer', transition: 'box-shadow var(--transition-slow), transform var(--transition-slow)', height: '100%',
      }}
      bodyStyle={{ padding: '12px 14px' }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10 }}>
        <div style={{
          width: 36, height: 36, borderRadius: '50%', background: 'rgba(192, 132, 252, 0.1)',
          display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
        }}>
          <ApartmentOutlined style={{ fontSize: 18, color: '#c084fc' }} />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>{org.name}</Typography.Text>
            <Space size={2}>
              {org.members && org.members.length > 0 && (
                <Tag color="var(--color-primary)" style={{ fontSize: 9, lineHeight: '14px', padding: '0 4px' }}>{org.members.length}人</Tag>
              )}
              {org.type && <Tag color="#c084fc" style={{ fontSize: 9, lineHeight: '14px', padding: '0 4px' }}>{org.type}</Tag>}
            </Space>
          </div>
          <Typography.Text style={{
            color: C('color-text-secondary'), fontSize: 10, display: 'block', marginTop: 2, lineHeight: 1.4,
            overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
          }}>
            {org.description ? org.description.slice(0, 40) + (org.description.length > 40 ? '...' : '') : (org.power_level || '')}
          </Typography.Text>
        </div>
      </div>
    </Card>
  )
}

export default OrganizationCard
