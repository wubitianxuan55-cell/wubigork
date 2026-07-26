import React from 'react'
import { Button, Typography, Space, Card, Tag, Spin } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'

interface StepCharacterProps {
  characters: any[]
  loading: boolean
  onRegenerate: () => void
}

const roleLabels: Record<string, string> = {
  protagonist: '主角', antagonist: '反派', supporting: '配角', minor: '龙套',
}
const roleColors: Record<string, string> = {
  protagonist: '#4ade80', antagonist: '#f87171', supporting: '#60a5fa', minor: '#94a3b8',
}

const StepCharacter: React.FC<StepCharacterProps> = ({ characters, loading, onRegenerate }) => (
  <div>
    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 8 }}>
      已生成的角色
    </Typography.Text>
    {loading ? (
      <div style={{ textAlign: 'center', padding: 20 }}><Spin /></div>
    ) : (
      <Space direction="vertical" size={8} style={{ width: '100%', marginBottom: 12 }}>
        {characters.map((ch, i) => (
          <Card key={i} size="small" style={{
            background: 'rgba(255,255,255,0.03)',
            border: '1px solid var(--border-subtle)',
            borderRadius: 'var(--radius-md)',
            borderLeft: `3px solid ${roleColors[ch.role_type] || '#94a3b8'}`,
          }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Typography.Text strong style={{ color: C('color-text') }}>{ch.name}</Typography.Text>
              <Tag color={roleColors[ch.role_type]} style={{ fontSize: 9 }}>
                {roleLabels[ch.role_type] || ch.role_type}
              </Tag>
            </div>
            <Typography.Text style={{ color: C('color-text-secondary'), display: 'block', fontSize: 11, marginTop: 4 }}>
              {ch.personality} · {ch.background?.slice(0, 60)}
            </Typography.Text>
          </Card>
        ))}
      </Space>
    )}
    <Button icon={<ReloadOutlined />} onClick={onRegenerate} loading={loading}
      style={{ width: '100%' }}>
      重新生成角色
    </Button>
  </div>
)

export default StepCharacter
