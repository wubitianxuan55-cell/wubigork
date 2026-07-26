import React from 'react'
import { Typography, Button, Space, Input, Spin } from 'antd'
import { BookOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

interface ThreadPanelProps {
  storyThread: string
  onThreadChange: (value: string) => void
  onGenerate: () => void
  generating: boolean
}

/**
 * ThreadPanel — 故事主线编辑/生成面板
 * 提取自 OutlinePage
 */
const ThreadPanel: React.FC<ThreadPanelProps> = ({ storyThread, onThreadChange, onGenerate, generating }) => {
  return (
    <div style={{
      background: 'var(--md-sys-color-surface-container)',
      borderRadius: 'var(--radius-lg)',
      border: '1px solid var(--border-subtle)',
      borderLeft: '3px solid #c084fc',
      padding: '12px 16px', marginBottom: 8,
      boxShadow: 'var(--shadow-sm)',
    }}>
      <div style={{
        display: 'flex', justifyContent: 'space-between',
        alignItems: 'center', marginBottom: 8,
      }}>
        <Typography.Text strong style={{ color: '#c084fc', fontSize: 12 }}>
          <BookOutlined style={{ marginRight: 6 }} />故事主线
        </Typography.Text>
        <Space size={4}>
          <Button
            size="small"
            icon={generating ? <Spin size="small" /> : <ThunderboltOutlined />}
            onClick={onGenerate}
            disabled={generating}
            style={{
              fontSize: 11, color: '#c084fc',
              borderColor: 'rgba(192,132,252,0.3)',
            }}
          >
            {generating ? '生成中' : 'AI 生成'}
          </Button>
        </Space>
      </div>
      <Input.TextArea
        value={storyThread}
        onChange={(e) => onThreadChange(e.target.value)}
        placeholder="在此编写故事主线，AI 写作时会参考..."
        rows={4}
        style={{
          fontSize: 12, lineHeight: 1.7,
          background: 'rgba(255,255,255,0.03)',
          border: '1px solid var(--border-subtle)',
          color: 'var(--color-text)',
          borderRadius: 'var(--radius-md)',
        }}
      />
    </div>
  )
}

export default ThreadPanel
