import React, { useState, useCallback, useEffect, useRef } from 'react'
import { Modal, Input, Button, Space, Typography, Spin, Tag } from 'antd'
import { ThunderboltOutlined, CheckOutlined, CloseOutlined } from '@ant-design/icons'
import DiffReview from './DiffReview'
import { C } from '../../../utils/theme'

/**
 * CommandBar — Cmd+K 命令编辑
 *
 * 选中文本 → Cmd+K / Ctrl+K → 输入指令 → AI 编辑 → diff 预览 → 接受/拒绝
 *
 * Props:
 *   selectedText — 当前选中的文本
 *   onAccept — 接受编辑后的回调 (editedText: string) => void
 *   onClose — 关闭面板回调
 *   styleProfile — 可选的风格指导
 */
interface CommandBarProps {
  selectedText: string
  onAccept: (editedText: string) => void
  onClose: () => void
  styleProfile?: string
}

const PRESET_COMMANDS = [
  { label: '更紧张', icon: '⚡', instruction: '用更紧张的节奏和更短的句子重写，增加紧迫感' },
  { label: '丰富描写', icon: '🎨', instruction: '添加生动的感官细节描写（视觉、听觉、触觉），让场景更立体' },
  { label: '精简', icon: '✂️', instruction: '精简冗余表达，保留核心信息，控制字数' },
  { label: '改对话', icon: '💬', instruction: '将这段改为生动的对话形式，增强角色个性' },
  { label: '更文艺', icon: '🖋️', instruction: '提升文笔，用更有文学性的语言重写' },
  { label: '展示代替讲述', icon: '👁️', instruction: '用具体场景和动作展示情绪，不要直接告诉读者角色的感受' },
]

const CommandBar: React.FC<CommandBarProps> = ({ selectedText, onAccept, onClose, styleProfile }) => {
  const [instruction, setInstruction] = useState('')
  const [loading, setLoading] = useState(false)
  const [editedText, setEditedText] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const inputRef = useRef<any>(null)

  // 自动聚焦
  useEffect(() => {
    setTimeout(() => inputRef.current?.focus?.(), 100)
  }, [])

  // Cmd+K 快捷键
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        onClose()
      }
      if (e.key === 'Escape') {
        onClose()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  const handleExecute = useCallback(async () => {
    if (!instruction.trim()) return

    setLoading(true)
    setError(null)
    setEditedText(null)

    try {
      // @ts-ignore
      const result = await window.go.app.App.CmdKEdit(selectedText, instruction.trim(), styleProfile || '')
      const edited = result?.edited || ''
      if (!edited) {
        setError('AI 未返回有效编辑结果')
      } else {
        setEditedText(edited)
      }
    } catch (err: any) {
      setError(err?.message || '编辑失败')
    } finally {
      setLoading(false)
    }
  }, [instruction, selectedText, styleProfile])

  const handleAccept = useCallback(() => {
    if (editedText) {
      onAccept(editedText)
    }
  }, [editedText, onAccept])

  const handlePresetClick = useCallback((cmd: typeof PRESET_COMMANDS[number]) => {
    setInstruction(cmd.instruction)
    setTimeout(() => inputRef.current?.focus?.(), 50)
  }, [])

  return (
    <Modal
      title={
        <Space>
          <ThunderboltOutlined style={{ color: C('color-accent') }} />
          <span>AI 编辑</span>
          <Typography.Text type="secondary" style={{ fontSize: 12, fontWeight: 'normal' }}>
            ⌘K
          </Typography.Text>
        </Space>
      }
      open
      onCancel={onClose}
      width={700}
      footer={
        editedText
          ? [
              <Button key="cancel" onClick={onClose} icon={<CloseOutlined />}>
                取消
              </Button>,
              <Button
                key="retry"
                onClick={() => { setEditedText(null); setInstruction('') }}
                disabled={loading}
              >
                重新编辑
              </Button>,
              <Button
                key="accept"
                type="primary"
                onClick={handleAccept}
                icon={<CheckOutlined />}
              >
                接受 (⌘Y)
              </Button>,
            ]
          : [
              <Button key="cancel" onClick={onClose}>
                取消
              </Button>,
              <Button
                key="execute"
                type="primary"
                onClick={handleExecute}
                loading={loading}
                disabled={!instruction.trim()}
                icon={<ThunderboltOutlined />}
              >
                执行 (⌘⏎)
              </Button>,
            ]
      }
    >
      {/* 预设命令 */}
      {!editedText && (
        <div style={{ marginBottom: 12 }}>
          <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 6 }}>
            快速指令:
          </Typography.Text>
          <Space wrap size={[4, 4]}>
            {PRESET_COMMANDS.map(cmd => (
              <Tag
                key={cmd.label}
                style={{ cursor: 'pointer' }}
                onClick={() => handlePresetClick(cmd)}
              >
                {cmd.icon} {cmd.label}
              </Tag>
            ))}
          </Space>
        </div>
      )}

      {/* 指令输入 */}
      {!editedText && (
        <Input.TextArea
          ref={inputRef}
          value={instruction}
          onChange={e => setInstruction(e.target.value)}
          placeholder="输入编辑指令，如「用更紧张的节奏重写」..."
          rows={2}
          onPressEnter={e => {
            if (e.metaKey || e.ctrlKey) {
              e.preventDefault()
              handleExecute()
            }
          }}
          style={{ marginBottom: 12 }}
        />
      )}

      {/* 选中文本预览（编辑前） */}
      {!editedText && selectedText && (
        <div
          style={{
            background: 'var(--bg-elevated)',
            borderRadius: 'var(--radius-md)',
            padding: '8px 12px',
            maxHeight: 120,
            overflow: 'auto',
            fontSize: 13,
            color: 'var(--color-text-secondary)',
            border: '1px solid var(--border-subtle)',
          }}
        >
          <Typography.Text type="secondary" style={{ fontSize: 10, display: 'block', marginBottom: 4 }}>
            选中文本 ({selectedText.length} 字):
          </Typography.Text>
          {selectedText.slice(0, 300)}
          {selectedText.length > 300 && '...'}
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div style={{ textAlign: 'center', padding: '24px 0' }}>
          <Spin />
          <Typography.Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
            AI 正在编辑...
          </Typography.Text>
        </div>
      )}

      {/* Error */}
      {error && (
        <Typography.Text type="danger" style={{ display: 'block', marginTop: 8 }}>
          {error}
        </Typography.Text>
      )}

      {/* Diff 预览 */}
      {editedText && !loading && (
        <DiffReview
          original={selectedText}
          revised={editedText}
          onAccept={handleAccept}
          onReject={onClose}
        />
      )}
    </Modal>
  )
}

export default CommandBar
export type { CommandBarProps }
