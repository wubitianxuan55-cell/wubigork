import React, { useState } from 'react'
import { Typography, Button, Space, Input, Modal, Card, Tag, Tooltip, Skeleton, message } from 'antd'
import {
  ThunderboltOutlined, BulbOutlined, HeartOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import type { BrainstormIdea } from '../types'

interface BrainstormModalProps {
  open: boolean
  onClose: () => void
  onAdoptIdea: (idea: BrainstormIdea, genre: string) => void
}

/** BrainstormModal — AI 脑暴弹窗 */
const BrainstormModal: React.FC<BrainstormModalProps> = ({ open, onClose, onAdoptIdea }) => {
  const [genre, setGenre] = useState('')
  const [ideas, setIdeas] = useState<BrainstormIdea[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedId, setSelectedId] = useState<number | null>(null)

  const handleBrainstorm = async () => {
    const g = genre.trim()
    if (!g) { message.warning('请输入题材'); return }
    setLoading(true)
    setIdeas([])
    setSelectedId(null)
    try {
      // @ts-ignore
      const result = await window.go.app.App.BrainstormIdeas(g)
      if (result?.ideas) setIdeas(result.ideas)
    } catch (err: any) {
      message.error(err.message || '脑暴失败')
    } finally { setLoading(false) }
  }

  const handleClose = () => {
    onClose()
    setTimeout(() => { setIdeas([]); setSelectedId(null); setGenre('') }, 200)
  }

  const handleAdopt = (idea: BrainstormIdea) => {
    onAdoptIdea(idea, genre)
    handleClose()
  }

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}><BulbOutlined style={{ color: '#f59e0b', marginRight: 8 }} />AI 脑暴创作点子</span>}
      open={open}
      onCancel={handleClose}
      footer={null}
      width={700}
      styles={{
        body: { background: C('color-bg-container'), maxHeight: '70vh', overflow: 'auto' },
        header: { background: C('color-bg-container') },
      }}
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        {/* 输入区 */}
        <Space.Compact style={{ width: '100%' }}>
          <Input
            placeholder="输入题材（如：修仙 + 科幻、都市悬疑、异世界种田...）"
            value={genre}
            onChange={(e) => setGenre(e.target.value)}
            onPressEnter={handleBrainstorm}
            style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }}
          />
          <Button
            type="primary" icon={<ThunderboltOutlined />}
            onClick={handleBrainstorm} loading={loading}
            style={{ background: '#f59e0b', borderColor: '#f59e0b' }}
          >
            生成点子
          </Button>
        </Space.Compact>

        {/* 加载中 */}
        {loading && (
          <div style={{ padding: 40 }}>
            <Skeleton active paragraph={{ rows: 4 }} />
            <Typography.Text style={{ color: C('color-text-secondary'), display: 'block', textAlign: 'center', marginTop: 12 }}>
              AI 正在头脑风暴...
            </Typography.Text>
          </div>
        )}

        {/* 结果列表 */}
        {ideas.length > 0 && (
          <div style={{
            background: 'var(--bg-glass)', backdropFilter: 'blur(12px)',
            WebkitBackdropFilter: 'blur(12px)',
            border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-xl)',
            padding: 16, marginBottom: 16,
          }}>
            <Space direction="vertical" size={10} style={{ width: '100%' }}>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>
                生成了 {ideas.length} 个点子，点击 <HeartOutlined style={{ color: '#f59e0b' }} /> 选用为小说标题
              </Typography.Text>
              {ideas.map((idea) => {
                const isSelected = selectedId === idea.id
                return (
                  <Card
                    key={idea.id}
                    size="small" hoverable
                    style={{
                      background: isSelected ? C('color-bg-layout') : C('color-bg-container'),
                      borderColor: isSelected ? '#f59e0b' : C('color-border'),
                      borderRadius: 8, transition: 'border-color 0.2s',
                    }}
                    onClick={() => setSelectedId(isSelected ? null : idea.id)}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                      <div style={{ flex: 1 }}>
                        <Typography.Text strong style={{ color: C('color-text'), fontSize: 14 }}>
                          <span style={{ color: '#f59e0b', marginRight: 6 }}>#{idea.id}</span>
                          {idea.title}
                        </Typography.Text>
                        <div style={{ color: C('color-text-secondary'), fontSize: 12, marginTop: 6, lineHeight: 1.6 }}>
                          {idea.pitch}
                        </div>
                        <div style={{ marginTop: 8 }}>
                          <Space size={4} wrap>
                            <Tag color="purple" style={{ fontSize: 10 }}>⚔️ {idea.conflict}</Tag>
                            <Tag color="blue" style={{ fontSize: 10 }}>👥 {idea.audience}</Tag>
                            {(idea.tags || []).map((t: string) => (
                              <Tag key={t} style={{ fontSize: 10 }}>{t}</Tag>
                            ))}
                          </Space>
                        </div>
                      </div>
                      <Tooltip title="选用此点子">
                        <Button
                          type={isSelected ? 'primary' : 'text'} size="small" icon={<HeartOutlined />}
                          onClick={(e) => { e.stopPropagation(); handleAdopt(idea) }}
                          style={{
                            color: isSelected ? '#fff' : C('color-text-secondary'),
                            background: isSelected ? '#f59e0b' : undefined,
                            borderColor: isSelected ? '#f59e0b' : undefined,
                            flexShrink: 0, marginLeft: 12,
                          }}
                        />
                      </Tooltip>
                    </div>
                  </Card>
                )
              })}
            </Space>
          </div>
        )}
      </Space>
    </Modal>
  )
}

export default BrainstormModal
