import React from 'react'
import { Typography, Space, Tag, Empty } from 'antd'
import { CheckCircleOutlined, CloseCircleOutlined, BulbOutlined, WarningOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

interface ReviewResultProps {
  data: {
    qualityScore: number
    strengths?: string[]
    weaknesses?: string[]
    suggestions?: string[]
    overall?: string
  }
}

/** ReviewResult — AI 审稿结果展示组件 */
const ReviewResult: React.FC<ReviewResultProps> = ({ data }) => {
  const score = data.qualityScore || 0
  const scoreColor = score >= 7 ? '#4ade80' : score >= 5 ? '#f59e0b' : '#f87171'
  const scoreLabel = score >= 7 ? '良好' : score >= 5 ? '一般' : '需改进'

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {/* 评分 */}
      <div style={{ textAlign: 'center', padding: '20px 0' }}>
        <div style={{
          width: 80, height: 80, borderRadius: '50%',
          border: `3px solid ${scoreColor}`,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          margin: '0 auto 12px',
        }}>
          <Typography.Title level={2} style={{ color: scoreColor, margin: 0 }}>
            {score}
          </Typography.Title>
        </div>
        <Tag color={scoreColor} style={{ fontSize: 12 }}>{scoreLabel}</Tag>
        {data.overall && (
          <Typography.Paragraph style={{ color: C('color-text-secondary'), marginTop: 12, fontSize: 13 }}>
            {data.overall}
          </Typography.Paragraph>
        )}
      </div>

      {/* 优势 */}
      {data.strengths && data.strengths.length > 0 && (
        <div>
          <Typography.Text strong style={{ color: '#4ade80', display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
            <CheckCircleOutlined /> 优势
          </Typography.Text>
          <ul style={{ margin: 0, paddingLeft: 20 }}>
            {data.strengths.map((s, i) => (
              <li key={i} style={{ color: C('color-text-secondary'), fontSize: 13, marginBottom: 4 }}>{s}</li>
            ))}
          </ul>
        </div>
      )}

      {/* 不足 */}
      {data.weaknesses && data.weaknesses.length > 0 && (
        <div>
          <Typography.Text strong style={{ color: '#f87171', display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
            <CloseCircleOutlined /> 不足
          </Typography.Text>
          <ul style={{ margin: 0, paddingLeft: 20 }}>
            {data.weaknesses.map((w, i) => (
              <li key={i} style={{ color: C('color-text-secondary'), fontSize: 13, marginBottom: 4 }}>{w}</li>
            ))}
          </ul>
        </div>
      )}

      {/* 改进建议 */}
      {data.suggestions && data.suggestions.length > 0 && (
        <div>
          <Typography.Text strong style={{ color: '#60a5fa', display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
            <BulbOutlined /> 改进建议
          </Typography.Text>
          <ul style={{ margin: 0, paddingLeft: 20 }}>
            {data.suggestions.map((s, i) => (
              <li key={i} style={{ color: C('color-text-secondary'), fontSize: 13, marginBottom: 4 }}>{s}</li>
            ))}
          </ul>
        </div>
      )}

      {/* 空状态 */}
      {!data.strengths && !data.weaknesses && !data.suggestions && (
        <Empty description="暂无详细分析数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      )}
    </Space>
  )
}

export default ReviewResult
