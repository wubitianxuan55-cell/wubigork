import React, { useState } from 'react'
import { Modal, Input, InputNumber, Button, Space, Typography, message } from 'antd'
import { ThunderboltOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

interface DialogueModalProps {
  open: boolean
  onClose: () => void
  onGenerated: (result: any) => void
}

/**
 * DialogueModal — 对话式大纲生成弹窗
 * 使用学生-专家 LLM 对话策略生成大纲
 */
const DialogueModal: React.FC<DialogueModalProps> = ({ open, onClose, onGenerated }) => {
  const [prompt, setPrompt] = useState('')
  const [chapters, setChapters] = useState(4)
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<any>(null)

  const handleGenerate = async () => {
    if (!prompt.trim()) {
      message.warning('请输入故事创意')
      return
    }
    setLoading(true)
    setResult(null)
    try {
      // @ts-ignore
      const res = await window.go.app.App.GenerateOutlineWithDialogue(prompt.trim(), chapters, 5)
      setResult(res)
      onGenerated(res)
      message.success(`故事「${res.storyTitle}」的大纲已生成`)
    } catch (err: any) {
      message.error('生成失败: ' + (err?.message || err))
    } finally {
      setLoading(false)
    }
  }

  const handleClose = () => {
    if (!loading) {
      setPrompt('')
      setResult(null)
      onClose()
    }
  }

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}>对话式大纲生成</span>}
      open={open}
      onCancel={handleClose}
      footer={null}
      width={640}
      styles={{
        body: { background: C('color-bg-container'), maxHeight: '70vh', overflow: 'auto' },
        header: { background: C('color-bg-container') },
      }}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 输入区 */}
        <div>
          <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 8 }}>
            输入故事创意/设定
          </Typography.Text>
          <Input.TextArea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="例如：一个修仙废柴意外获得远古传承，在宗门大比中一鸣惊人..."
            rows={4}
            style={{
              fontSize: 13, lineHeight: 1.7,
              background: 'rgba(255,255,255,0.03)',
              border: '1px solid var(--border-subtle)',
              color: 'var(--color-text)',
              borderRadius: 'var(--radius-md)',
            }}
          />
        </div>

        {/* 章节数 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Typography.Text style={{ color: C('color-text'), fontSize: 12 }}>章节数：</Typography.Text>
          <InputNumber
            min={1} max={20} value={chapters}
            onChange={(v) => setChapters(v || 4)}
            size="small"
            style={{ width: 80 }}
          />
        </div>

        {/* 生成按钮 */}
        <Button
          type="primary"
          icon={<ThunderboltOutlined />}
          onClick={handleGenerate}
          loading={loading}
          size="large"
          style={{ width: '100%' }}
        >
          {loading ? 'AI 正在讨论故事大纲...' : '开始生成'}
        </Button>

        {/* 结果展示 */}
        {result && (
          <div style={{
            padding: 12,
            background: 'rgba(192, 132, 252, 0.06)',
            borderRadius: 'var(--radius-md)',
            border: '1px solid rgba(192, 132, 252, 0.2)',
          }}>
            <Typography.Title level={5} style={{ color: '#c084fc', margin: 0, marginBottom: 8 }}>
              📖 {result.storyTitle}
            </Typography.Title>
            <div style={{ maxHeight: 300, overflow: 'auto' }}>
              {result.chapters?.map((ch: any, i: number) => (
                <div key={i} style={{
                  padding: '8px 10px', marginBottom: 4,
                  background: 'rgba(255,255,255,0.03)',
                  borderRadius: 'var(--radius-sm)',
                }}>
                  <Typography.Text strong style={{ color: C('color-text'), fontSize: 12 }}>
                    第{i + 1}章：{ch.title}
                  </Typography.Text>
                  <Typography.Text style={{
                    color: C('color-text-secondary'), fontSize: 11,
                    display: 'block', marginTop: 2,
                  }}>
                    {ch.summary}
                  </Typography.Text>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* 对话记录 */}
        {result?.dialogue && (
          <div>
            <Typography.Text strong style={{ color: C('color-text'), fontSize: 12, display: 'block', marginBottom: 8 }}>
              AI 讨论过程
            </Typography.Text>
            <div style={{
              maxHeight: 200, overflow: 'auto',
              background: 'rgba(0,0,0,0.1)',
              borderRadius: 'var(--radius-md)', padding: 8,
            }}>
              {result.dialogue.map((d: any, i: number) => (
                <div key={i} style={{
                  marginBottom: 4, padding: '4px 8px',
                  background: d.speaker === 'expert'
                    ? 'rgba(96, 165, 250, 0.1)'
                    : 'rgba(74, 222, 128, 0.1)',
                  borderRadius: 4,
                  fontSize: 11, color: C('color-text'),
                }}>
                  <Tag style={{ fontSize: 8, marginRight: 4 }}>
                    {d.speaker === 'expert' ? '专家' : '学生'}
                  </Tag>
                  {d.content}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </Modal>
  )
}

// 局部 Tag 组件
const Tag: React.FC<{ children: React.ReactNode; style?: React.CSSProperties }> = ({ children, style }) => (
  <span style={{
    display: 'inline-block', padding: '0 4px',
    fontSize: 8, fontWeight: 600,
    borderRadius: 3, background: 'rgba(192,132,252,0.15)',
    color: '#c084fc', ...style,
  }}>
    {children}
  </span>
)

export default DialogueModal
