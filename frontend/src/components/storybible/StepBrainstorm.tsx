import React from 'react'
import { Button, Input, Typography, Card, Space, Spin, message } from 'antd'
import { ThunderboltOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'

interface StepBrainstormProps {
  genre: string
  onGenreChange: (v: string) => void
  loading: boolean
  brainIdeas: any[]
  onBrainstorm: () => void
  onAdopt: (idea: any) => void
}

const StepBrainstorm: React.FC<StepBrainstormProps> = ({
  genre, onGenreChange, loading, brainIdeas, onBrainstorm, onAdopt,
}) => (
  <div>
    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 8 }}>
      输入故事题材
    </Typography.Text>
    <Input
      value={genre}
      onChange={(e) => onGenreChange(e.target.value)}
      placeholder="例如：玄幻修仙、都市异能、科幻末世..."
      size="large"
      style={{
        marginBottom: 12,
        background: 'rgba(255,255,255,0.03)',
        border: '1px solid var(--border-subtle)',
        color: 'var(--color-text)',
        borderRadius: 'var(--radius-md)',
      }}
    />
    <Button
      type="primary" size="large" icon={<ThunderboltOutlined />}
      onClick={onBrainstorm} loading={loading}
      style={{ width: '100%', marginBottom: 16 }}
    >
      {loading ? 'AI 脑暴中...' : 'AI 脑暴灵感'}
    </Button>
    {brainIdeas.length > 0 && (
      <div>
        <Typography.Text strong style={{ color: C('color-text'), fontSize: 12, display: 'block', marginBottom: 8 }}>
          选择灵感
        </Typography.Text>
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          {brainIdeas.map((idea, i) => (
            <Card
              key={i} hoverable size="small"
              onClick={() => onAdopt(idea)}
              style={{
                background: 'rgba(255,255,255,0.03)',
                border: '1px solid var(--border-subtle)',
                borderRadius: 'var(--radius-md)',
              }}
            >
              <Typography.Text strong style={{ color: C('color-text') }}>{idea.title}</Typography.Text>
              <Typography.Text style={{ color: C('color-text-secondary'), display: 'block', fontSize: 12, marginTop: 4 }}>
                {idea.summary}
              </Typography.Text>
            </Card>
          ))}
        </Space>
      </div>
    )}
  </div>
)

export default StepBrainstorm
