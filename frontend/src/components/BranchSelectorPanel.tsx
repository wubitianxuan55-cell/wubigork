import React, { useState, useEffect } from 'react'
import { Typography, Button, Space, Tag, Input, Spin, Card, Row, Col } from 'antd'
import {
  ThunderboltOutlined, EditOutlined, HeartOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import { toneColors, type Branch } from '../hooks/usePlotBranch'

interface BranchSelectorPanelProps {
  /** 从 usePlotBranch hook 传入 */
  loading: boolean
  branches: Branch[]
  selected: number
  setSelected: (i: number) => void
  /** AI 脑暴 */
  onBrainstorm: () => void
  /** 应用 loading */
  applying: boolean
  /** 应用回调 (manualMode, inputText) => Promise */
  onApply: (manualMode: boolean, input: string) => Promise<void>
  /** 分支卡片的额外点击副作用（如设置 keyInfo） */
  onBranchClick?: (branch: Branch, index: number) => void
  /** 是否自动触发脑暴（默认 false） */
  autoBrainstorm?: boolean
  header?: React.ReactNode
  /** 脑暴按钮文本（默认 'AI 推理分支'） */
  brainstormText?: string
  /** 脑暴 loading 文本（默认 'AI 正在推理...'） */
  brainstormLoadingText?: string
  /** 分支列表标题（默认 'AI 建议的分支（点击选择）'） */
  branchListTitle?: string
  /** 应用按钮文本（默认 '选用此分支'） */
  applyText?: string
  /** 应用按钮 loading 文本（默认 '正在应用'） */
  applyLoadingText?: string
  /** 手工输入 trigger 文本（默认 '或手工录入分支方向'） */
  manualTriggerText?: string
  /** 手工输入 placeholder（默认 '描述你想要的剧情走向...'） */
  manualPlaceholder?: string
  /** 手工输入应用按钮文本（默认 '应用手工输入'） */
  manualApplyText?: string
}

/**
 * BranchSelectorPanel — 共享的剧情分支选择面板
 *
 * 同时服务于 PlotBranchModal（选用分支 → ApplyBranch）和
 * NextChapterModal（作为续写方向 → onGenerate）。
 */
const BranchSelectorPanel: React.FC<BranchSelectorPanelProps> = ({
  loading, branches, selected, setSelected,
  onBrainstorm, applying, onApply, onBranchClick,
  autoBrainstorm, header,
  brainstormText = 'AI 推理分支',
  brainstormLoadingText = 'AI 正在推理...',
  branchListTitle = 'AI 建议的分支（点击选择）',
  applyText = '选用此分支',
  applyLoadingText = '正在应用',
  manualTriggerText = '或手工录入分支方向',
  manualPlaceholder = '描述你想要的剧情走向...',
  manualApplyText = '应用手工输入',
}) => {
  const [manualMode, setManualMode] = useState(false)
  const [manualInput, setManualInput] = useState('')

  useEffect(() => {
    if (autoBrainstorm) onBrainstorm()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const handleSelectBranch = (i: number, branch: Branch) => {
    setSelected(i)
    setManualMode(false)
    onBranchClick?.(branch, i)
  }

  const handleApplyLocal = async () => {
    if (selected < 0 && !manualMode) return
    await onApply(manualMode, manualInput)
  }

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      {/* AI 推理按钮 */}
      <Button type="primary" icon={<ThunderboltOutlined />} onClick={onBrainstorm}
        loading={loading} style={{ width: '100%' }}>
        {loading ? brainstormLoadingText : brainstormText}
      </Button>

      {loading && (
        <div style={{ textAlign: 'center', padding: 20 }}>
          <Spin tip="正在思考剧情走向..." />
        </div>
      )}

      {/* 头部额外内容（已完成条等） */}
      {header}

      {/* 分支列表 */}
      {branches.length > 0 && !loading && (
        <div>
          <Typography.Text strong style={{ color: C('color-text'), marginBottom: 8, display: 'block' }}>
            {branchListTitle}
          </Typography.Text>
          <Space direction="vertical" size={6} style={{ width: '100%' }}>
            {branches.map((b: Branch, i: number) => (
              <Card
                key={b.id || i}
                hoverable
                size="small"
                onClick={() => handleSelectBranch(i, b)}
                style={{
                  background: selected === i ? 'rgba(192, 132, 252, 0.12)' : 'rgba(255,255,255,0.03)',
                  border: selected === i ? '1px solid #c084fc' : '1px solid var(--border-subtle)',
                  borderRadius: 'var(--radius-md)',
                }}
              >
                <Row justify="space-between" align="middle">
                  <Col span={18}>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 12 }}>
                      {b.title}
                    </Typography.Text>
                    <Typography.Paragraph style={{ color: C('color-text-secondary'), fontSize: 11, margin: '2px 0 0' }}
                      ellipsis={{ rows: 2 }}>
                      {b.summary}
                    </Typography.Paragraph>
                  </Col>
                  <Col span={6} style={{ textAlign: 'right' }}>
                    <Tag color={toneColors[b.tone] || 'default'} style={{ fontSize: 9, margin: 0 }}>
                      {b.tone}
                    </Tag>
                    {selected === i && (
                      <Tag color="#c084fc" style={{ fontSize: 9, marginLeft: 2 }}>已选</Tag>
                    )}
                  </Col>
                </Row>
                {b.characters_involved?.length > 0 && (
                  <div style={{ marginTop: 6 }}>
                    {b.characters_involved.map((c: string) => (
                      <Tag key={c} style={{ fontSize: 8 }}>{c}</Tag>
                    ))}
                  </div>
                )}
              </Card>
            ))}
          </Space>
        </div>
      )}

      {/* 应用分支按钮 */}
      {selected >= 0 && !manualMode && (
        <Button type="primary" icon={<HeartOutlined />} onClick={handleApplyLocal}
          loading={applying} style={{ width: '100%' }}>
          {applying ? applyLoadingText : applyText}
        </Button>
      )}

      {/* 手工模式入口 */}
      {!manualMode && (
        <div style={{ textAlign: 'center', color: C('color-text-secondary'), fontSize: 11 }}>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => setManualMode(true)}>
            {manualTriggerText}
          </Button>
        </div>
      )}

      {/* 手工输入 */}
      {manualMode && (
        <div>
          <Input.TextArea
            value={manualInput}
            onChange={(e) => setManualInput(e.target.value)}
            placeholder={manualPlaceholder}
            rows={3}
            style={{ fontSize: 12, marginBottom: 8 }}
          />
          <Button type="primary" onClick={handleApplyLocal}
            loading={applying} disabled={!manualInput.trim()} style={{ width: '100%' }}>
            {applying ? applyLoadingText : manualApplyText}
          </Button>
        </div>
      )}
    </Space>
  )
}

export default BranchSelectorPanel
