import React, { useState, useEffect } from 'react'
import { Typography, Space, Tag, Modal, Spin, Progress, Tabs } from 'antd'
import type { TabsProps } from 'antd'
import {
  ReadOutlined, AimOutlined, EditOutlined, BarChartOutlined, TeamOutlined,
} from '@ant-design/icons'

import { C } from '../utils/theme'

interface ArcCompletion {
  character: string
  progress: number
  note: string
}

interface Scores {
  structure: number
  pacing: number
  characters: number
  prose: number
  creativity: number
}

interface ReviewData {
  letter: string
  totalScore: number
  scores: Scores
  peaks: number[]
  valleys: number[]
  arcCompletions: ArcCompletion[]
}

interface BookReviewModalProps {
  open: boolean
  onClose: () => void
}

const BookReviewModal: React.FC<BookReviewModalProps> = ({ open, onClose }) => {
  const [loading, setLoading] = useState(false)
  const [review, setReview] = useState<ReviewData | null>(null)
  const [tab, setTab] = useState('edit')

  useEffect(() => {
    if (open) {
      setLoading(true)
      setReview(null)
      handleReview()
    }
  }, [open])

  const handleReview = async () => {
    try {
      // @ts-ignore
      const result = await window.go.app.App.ReviewBook()
      setReview(result)
    } catch (err: any) {
      console.error('全书审稿失败:', err)
    } finally { setLoading(false) }
  }

  const tabItems: TabsProps['items'] = [
    { key: 'edit', label: <span><EditOutlined style={{ marginRight: 4 }} />编辑信</span> },
    { key: 'scores', label: <span><BarChartOutlined style={{ marginRight: 4 }} />评分</span> },
    { key: 'arcs', label: <span><TeamOutlined style={{ marginRight: 4 }} />角色弧光</span> },
  ]

  const scoreLabels: Record<string, string> = {
    structure: '结构', pacing: '节奏', characters: '角色',
    prose: '文笔', creativity: '创意',
  }
  const scoreColors: Record<string, string> = {
    structure: '#60a5fa', pacing: '#4ade80', characters: '#c084fc',
    prose: '#f59e0b', creativity: '#f87171',
  }

  const arcColor = (progress: number) =>
    progress >= 80 ? '#4ade80' : progress >= 50 ? '#f59e0b' : '#f87171'

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}><ReadOutlined style={{ color: C('color-primary'), marginRight: 8 }} />全书发展编辑</span>}
      open={open}
      onCancel={onClose}
      footer={null}
      width={720}
      styles={{
        body: { maxHeight: '70vh', overflow: 'auto', padding: 0 },
      }}
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: 60 }}>
          <Spin size="large" />
          <div style={{ color: C('color-text-secondary'), marginTop: 16 }}>
            AI 正在审读全书...
          </div>
        </div>
      ) : review ? (
        <div>
          {/* 总评分头 */}
          <div style={{
            padding: '16px 24px', borderBottom: '1px solid ' + C('color-border'),
            display: 'flex', alignItems: 'center', gap: 16,
          }}>
            <Progress
              type="circle"
              percent={review.totalScore * 10}
              format={() => `${review.totalScore}/10`}
              size={64}
              strokeColor={review.totalScore >= 7 ? '#4ade80' : review.totalScore >= 4 ? '#f59e0b' : '#f87171'}
            />
            <div>
              <Typography.Title level={5} style={{ color: C('color-text'), margin: 0 }}>
                全书总评分
              </Typography.Title>
              <Space size={4}>
                {review.peaks.length > 0 && <Tag color="green" style={{ fontSize: 10 }}>🏔 峰值在{review.peaks.join(',')}章</Tag>}
                {review.valleys.length > 0 && <Tag color="orange" style={{ fontSize: 10 }}>📉 低谷在{review.valleys.join(',')}章</Tag>}
              </Space>
            </div>
          </div>

          <Tabs activeKey={tab} onChange={setTab} items={tabItems} size="small"
            style={{ padding: '0 16px' }} tabBarStyle={{ marginBottom: 0 }} />

          <div style={{ padding: '12px 24px 24px', maxHeight: '55vh', overflow: 'auto' }}>
            {tab === 'edit' && (
              <div style={{
                color: C('color-text'),
                whiteSpace: 'pre-wrap',
                lineHeight: 2,
                fontSize: 14,
                fontFamily: 'Georgia, "Noto Serif SC", serif',
              }}>
                {review.letter}
              </div>
            )}

            {tab === 'scores' && (
              <Space direction="vertical" size={16} style={{ width: '100%' }}>
                {Object.entries(review.scores || {}).map(([k, v]) => (
                  <div key={k}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                      <Typography.Text style={{ color: C('color-text'), fontSize: 13 }}>
                        {scoreLabels[k] || k}
                      </Typography.Text>
                      <Typography.Text style={{ color: scoreColors[k], fontSize: 13, fontWeight: 600 }}>
                        {v}/10
                      </Typography.Text>
                    </div>
                    <Progress
                      percent={v * 10}
                      size="small"
                      strokeColor={scoreColors[k]}
                      showInfo={false}
                    />
                  </div>
                ))}
              </Space>
            )}

            {tab === 'arcs' && (
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                {(review.arcCompletions || []).map((arc) => (
                  <div key={arc.character}
                    style={{
                      background: 'var(--bg-elevated)', borderRadius: 8,
                      padding: '10px 14px', border: '1px solid var(--border-subtle)',
                    }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                      <Space size={4}>
                        <AimOutlined style={{ color: arcColor(arc.progress) }} />
                        <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>
                          {arc.character}
                        </Typography.Text>
                      </Space>
                      <Tag color={arcColor(arc.progress)}>{arc.progress}%</Tag>
                    </div>
                    <Progress
                      percent={arc.progress}
                      size="small"
                      strokeColor={arcColor(arc.progress)}
                      showInfo={false}
                    />
                    {arc.note && (
                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginTop: 6 }}>
                        {arc.note}
                      </Typography.Text>
                    )}
                  </div>
                ))}
              </Space>
            )}
          </div>
        </div>
      ) : null}
    </Modal>
  )
}

export default BookReviewModal
