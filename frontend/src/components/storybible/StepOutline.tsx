import React from 'react'
import { Button, Typography, Space, Tag, Spin } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'

interface StepOutlineProps {
  outlines: any[]
  loading: boolean
  onRegenerate: () => void
}

const StepOutline: React.FC<StepOutlineProps> = ({ outlines, loading, onRegenerate }) => (
  <div>
    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 8 }}>
      已生成的大纲
    </Typography.Text>
    {loading ? (
      <div style={{ textAlign: 'center', padding: 20 }}><Spin /></div>
    ) : (
      <Space direction="vertical" size={4} style={{ width: '100%', marginBottom: 12 }}>
        {outlines.map((node: any, i: number) => (
          <div key={i} style={{
            padding: '8px 10px',
            background: 'rgba(255,255,255,0.03)',
            borderRadius: 'var(--radius-sm)',
            borderLeft: '2px solid ' + (node.status === 'done' ? '#4ade80' : '#c084fc'),
          }}>
            <Typography.Text strong style={{ color: C('color-text'), fontSize: 12 }}>
              第{node.order_index || i + 1}章：{node.title}
            </Typography.Text>
            <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginTop: 2 }}>
              {node.summary?.slice(0, 80)}
            </Typography.Text>
          </div>
        ))}
      </Space>
    )}
    <Button icon={<ReloadOutlined />} onClick={onRegenerate} loading={loading}
      style={{ width: '100%' }}>
      重新生成大纲
    </Button>
  </div>
)

export default StepOutline
