import React, { useState, useRef, useEffect } from 'react'
import { Button, Input, Popconfirm, Typography, Tooltip } from 'antd'
import {
  PlusOutlined,
  DeleteOutlined,
  MessageOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'

export interface Topic {
  id: string
  title: string
  createdAt: number
  /** topic mode: 'plain' or personaID */
  mode?: string
  /** display label for the mode badge (empty string for plain) */
  modeLabel?: string
  /** first message preview */
  preview?: string
}

interface ChatTopicSidebarProps {
  topics: Topic[]
  activeId: string
  onSelect: (id: string) => void
  onCreate: () => void
  onDelete: (id: string) => void
  onRename: (id: string, title: string) => void
  /** 折叠态：窄栏只保留切换/新建按钮，随左侧面板一起折叠 */
  collapsed?: boolean
  onToggle?: () => void
}

const ChatTopicSidebar: React.FC<ChatTopicSidebarProps> = ({
  topics, activeId, onSelect, onCreate, onDelete, onRename,
  collapsed = false, onToggle,
}) => {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editText, setEditText] = useState('')
  const [hoveredId, setHoveredId] = useState<string | null>(null)
  const editRef = useRef<any>(null)

  useEffect(() => {
    if (editingId && editRef.current) {
      editRef.current.focus()
    }
  }, [editingId])

  const startEdit = (topic: Topic) => {
    setEditingId(topic.id)
    setEditText(topic.title)
  }

  const commitEdit = () => {
    if (editingId && editText.trim()) {
      onRename(editingId, editText.trim())
    }
    setEditingId(null)
    setEditText('')
  }

  return (
    <div
      style={{
        width: collapsed ? 52 : 264,
        flexShrink: 0,
        display: 'flex',
        flexDirection: 'column',
        borderRight: `1px solid ${C('color-border')}`,
        background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
        WebkitBackdropFilter: 'blur(20px) saturate(140%)',
        backdropFilter: 'blur(20px) saturate(140%)',
        height: '100%',
        userSelect: 'none',
        boxShadow: '2px 0 14px rgba(0,0,0,0.05)',
      }}
    >
      {collapsed ? (
        /* 折叠窄栏：切换按钮 + 新建会话 */
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8, paddingTop: 10 }}>
          {onToggle && (
            <Tooltip title="展开会话列表" placement="right">
              <Button
                type="text"
                size="small"
                icon={<MenuUnfoldOutlined />}
                onClick={onToggle}
                style={{ color: C('color-text-secondary'), borderRadius: 8 }}
              />
            </Tooltip>
          )}
          <Tooltip title="新建会话" placement="right">
            <Button
              type="text"
              size="small"
              icon={<PlusOutlined />}
              onClick={onCreate}
              style={{ color: C('color-primary'), borderRadius: 8 }}
            />
          </Tooltip>
        </div>
      ) : (
        <>
          {/* header: title + new topic button + collapse */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '12px 14px 10px',
            }}
          >
            <Typography.Text strong style={{ color: C('color-text'), fontSize: 13.5 }}>
              <MessageOutlined style={{ marginRight: 8, color: C('color-primary') }} />
              会话
            </Typography.Text>
            <div style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              {onToggle && (
                <Tooltip title="折叠会话列表" placement="bottom">
                  <Button
                    type="text"
                    size="small"
                    icon={<MenuFoldOutlined />}
                    onClick={onToggle}
                    style={{ color: C('color-text-secondary'), borderRadius: 8 }}
                  />
                </Tooltip>
              )}
              <Tooltip title="新建会话" placement="bottom">
                <Button
                  type="text"
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={onCreate}
                  style={{ color: C('color-primary'), borderRadius: 8 }}
                />
              </Tooltip>
            </div>
          </div>

          {/* topic list */}
          <div style={{ flex: 1, overflow: 'auto', padding: '2px 8px 88px' }}>
            {topics.length === 0 ? (
              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '48px 16px', textAlign: 'center' }}>
                <div style={{
                  width: 48, height: 48, borderRadius: 14,
                  background: `${C('color-primary')}10`,
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  marginBottom: 14,
                }}>
                  <MessageOutlined style={{ fontSize: 24, color: C('color-primary') }} />
                </div>
                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 13, marginBottom: 4 }}>
                  暂无会话
                </Typography.Text>
                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, opacity: 0.7 }}>
                  点击 + 创建新对话
                </Typography.Text>
              </div>
            ) : (
              <div>
                {topics.map((topic) => {
              const active = topic.id === activeId
              const hovered = hoveredId === topic.id
              return (
                <div
                  key={topic.id}
                  className={`chat-topic-item${active ? ' active' : ''}`}
                  onClick={() => onSelect(topic.id)}
                  onMouseEnter={() => setHoveredId(topic.id)}
                  onMouseLeave={() => setHoveredId(null)}
                  style={{
                    background: active
                      ? `${C('color-primary')}15`
                      : hovered
                        ? C('color-bg-container')
                        : 'transparent',
                  }}
                >
                  {/* title + preview (double-click to edit) */}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    {editingId === topic.id ? (
                      <Input
                        ref={editRef}
                        size="small"
                        value={editText}
                        onChange={(e) => setEditText(e.target.value)}
                        onBlur={commitEdit}
                        onPressEnter={commitEdit}
                        onClick={(e) => e.stopPropagation()}
                        style={{ background: C('color-bg-container'), borderColor: C('color-primary'), color: C('color-text'), borderRadius: 6 }}
                      />
                    ) : (
                      <>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 0 }}>
                          <Typography.Text
                            ellipsis
                            onDoubleClick={() => startEdit(topic)}
                            style={{
                              color: active ? C('color-primary') : C('color-text'),
                              fontSize: 12.5,
                              fontWeight: active ? 600 : 500,
                              display: 'block',
                              lineHeight: '18px',
                              minWidth: 0,
                            }}
                          >
                            {topic.title}
                          </Typography.Text>
                          {topic.modeLabel && (
                            <span className="chat-topic-badge" title={`${topic.modeLabel} 模式`}>
                              {topic.modeLabel}
                            </span>
                          )}
                        </div>
                        {topic.preview && (
                          <div className="chat-topic-preview" title={topic.preview}>
                            {topic.preview}
                          </div>
                        )}
                      </>
                    )}
                  </div>

                  {/* delete button — visible on hover / active */}
                  <div
                    style={{
                      width: 26, height: 26,
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                      flexShrink: 0, marginLeft: 4,
                      opacity: hovered || active ? 1 : 0,
                      transition: 'opacity 0.12s',
                    }}
                  >
                    <Popconfirm
                      title="确定删除此会话？"
                      onConfirm={(e) => { e?.stopPropagation(); onDelete(topic.id) }}
                      onCancel={(e) => e?.stopPropagation()}
                      okText="删除"
                      cancelText="取消"
                      placement="right"
                    >
                      <Button
                        type="text"
                        size="small"
                        icon={<DeleteOutlined style={{ fontSize: 13 }} />}
                        onClick={(e) => e.stopPropagation()}
                        style={{
                          color: C('color-text-secondary'),
                          width: 26, height: 26, padding: 0,
                          display: 'flex', alignItems: 'center', justifyContent: 'center',
                          borderRadius: 6,
                        }}
                      />
                    </Popconfirm>
                  </div>
                </div>
              )
                })}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}

export default ChatTopicSidebar
