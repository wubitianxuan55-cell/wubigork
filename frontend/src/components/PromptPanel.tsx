import React from 'react'
import { Typography, Input, Tag, Space } from 'antd'
import { C } from '../utils/theme'
import { TEMPLATES, type Template } from '../data/imageTemplates'

const { TextArea } = Input

interface Props {
  prompt: string
  negative: string
  onPromptChange: (v: string) => void
  onNegativeChange: (v: string) => void
  onTemplateSelect: (t: Template) => void
}

const PromptPanel: React.FC<Props> = ({ prompt, negative, onPromptChange, onNegativeChange, onTemplateSelect }) => {
  const [showNegative, setShowNegative] = React.useState(false)

  return (
    <div>
      {/* 正向 Prompt */}
      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>
        描述你想要的画面
      </Typography.Text>
      <TextArea
        placeholder="例如：一座悬浮在云端的东方仙侠城市，琉璃瓦宫殿，瀑布倾泻，落日余晖..."
        value={prompt}
        onChange={(e) => onPromptChange(e.target.value)}
        rows={4}
        autoSize={{ minRows: 3, maxRows: 6 }}
        style={{
          background: 'rgba(255,255,255,0.05)',
          border: '1px solid var(--border-subtle)',
          borderRadius: 'var(--radius-md)',
          color: 'var(--color-text)',
          resize: 'none',
          marginBottom: 8,
        }}
      />

      {/* 负向 Prompt — 可折叠 */}
      <Typography.Link
        onClick={() => setShowNegative(!showNegative)}
        style={{ fontSize: 11, marginBottom: showNegative ? 4 : 12, display: 'block' }}
      >
        {showNegative ? '🚫 收起不想出现的内容' : '🚫 添加不想出现的内容...'}
      </Typography.Link>
      {showNegative && (
        <TextArea
          placeholder="模糊, 低质量, 畸形手指, 多余肢体..."
          value={negative}
          onChange={(e) => onNegativeChange(e.target.value)}
          rows={2}
          autoSize={{ minRows: 1, maxRows: 3 }}
          style={{
            background: 'rgba(255,255,255,0.05)',
            border: '1px solid var(--border-subtle)',
            borderRadius: 'var(--radius-md)',
            color: 'var(--color-text)',
            resize: 'none',
            marginBottom: 12,
          }}
        />
      )}

      {/* 快速模板 */}
      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 6 }}>
        📐 快速模板
      </Typography.Text>
      {Object.entries(TEMPLATES).map(([category, templates]) => (
        <div key={category} style={{ marginBottom: 6 }}>
          <Typography.Text style={{ fontSize: 10, color: C('color-text-secondary'), marginRight: 4 }}>
            {category}
          </Typography.Text>
          <Space wrap size={[4, 4]}>
            {templates.map((t) => (
              <Tag
                key={t.label}
                style={{ cursor: 'pointer', borderRadius: 'var(--radius-sm)', fontSize: 11 }}
                onClick={() => onTemplateSelect(t)}
              >
                {t.label}
              </Tag>
            ))}
          </Space>
        </div>
      ))}
    </div>
  )
}

export default PromptPanel
