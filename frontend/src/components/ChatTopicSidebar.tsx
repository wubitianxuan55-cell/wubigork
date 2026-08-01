import React, { useState, useRef, useEffect } from 'react'
import { Button, Input, Popconfirm, Typography, Tooltip } from 'antd'
import {
  PlusOutlined,
  DeleteOutlined,
  MessageOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'

export interface Topic {
  id: string
  title: string
  createdAt: number
}

interface ChatTopicSidebarProps {
  topics: Topic[]
  activeId: string
  onSelect: (id: string) => void
  onCreate: () => void
  onDelete: (id: string) => void
  onRename: (id: string, title: string) => void
}

const ChatTopicSidebar: React.FC<ChatTopicSidebarProps> = ({
  topics, activeId, onSelect, onCreate, onDelete, onRename,
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
        width: 260,
        flexShrink: 0,
        display: 'flex',
        flexDirection: 'column',
        borderRight: `2px solid ${C('color-primary')}20`,
        background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
        WebkitBackdropFilter: 'blur(18px) saturate(140%)',
        backdropFilter: 'blur(18px) saturate(140%)',
        height: '100%',
        userSelect: 'none',
        boxShadow: `2px 0 12px rgba(0,0,0,0.04)`,
      }}
    >
      {/* 顶栏：标题 + 新建按钮 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '14px 14px 10px',
        }}
      >
        <Typography.Text
          strong
          style={{ color: C('color-text'), fontSize: 14 }}
        >
          <MessageOutlined style={{ marginRight: 8, color: C('color-primary') }} />
          话题
        </Typography.Text>
        <Tooltip title="新建话题" placement="bottom">
          <Button
            type="text"
            size="small"
            icon={<PlusOutlined />}
            onClick={onCreate}
            style={{
              color: C('color-primary'),
              borderRadius: 8,
              transition: 'background 0.15s',
            }}
          />
        </Tooltip>
      </div>

      {/* 话题列表 */}
      <div style={{ flex: 1, overflow: 'auto', padding: '0 8px 8px' }}>
        {topics.length === 0 ? (
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              padding: '48px 16px',
              textAlign: 'center',
            }}
          >
            <div
              style={{
                width: 48,
                height: 48,
                borderRadius: 14,
                background: `${C('color-primary')}10`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                marginBottom: 14,
              }}
            >
              <MessageOutlined style={{ fontSize: 24, color: C('color-primary') }} />
            </div>
            <Typography.Text
              style={{
                color: C('color-text-secondary'),
                fontSize: 13,
                marginBottom: 4,
              }}
            >
              暂无话题
            </Typography.Text>
            <Typography.Text
              style={{
                color: C('color-text-secondary'),
                fontSize: 12,
                opacity: 0.7,
              }}
            >
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
                  onClick={() => onSelect(topic.id)}
                  onMouseEnter={() => setHoveredId(topic.id)}
                  onMouseLeave={() => setHoveredId(null)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    padding: '9px 12px',
                    marginBottom: 2,
                    borderRadius: 10,
                    cursor: 'pointer',
                    background: active
                      ? `${C('color-primary')}15`
                      : hovered
                        ? C('color-bg-container')
                        : 'transparent',
                    border: active
                      ? `1px solid ${C('color-primary')}30`
                      : '1px solid transparent',
                    transition: 'background 0.15s, border 0.15s, box-shadow 0.15s',
                    boxShadow: active
                      ? `0 1px 4px ${C('color-primary')}12`
                      : 'none',
                  }}
                >
                  {/* 标题（可双击编辑） */}
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
                        style={{
                          background: C('color-bg-container'),
                          borderColor: C('color-primary'),
                          color: C('color-text'),
                          borderRadius: 6,
                        }}
                      />
                    ) : (
                      <Typography.Text
                        ellipsis
                        onDoubleClick={() => startEdit(topic)}
                        style={{
                          color: active ? C('color-primary') : C('color-text'),
                          fontSize: 13,
                          fontWeight: active ? 500 : 400,
                          display: 'block',
                          lineHeight: '20px',
                        }}
                      >
                        {topic.title}
                      </Typography.Text>
                    )}
                  </div>

                  {/* 删除按钮 — hover / active 时显示 */}
                  <div
                    style={{
                      width: 26,
                      height: 26,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      flexShrink: 0,
                      marginLeft: 4,
                      opacity: hovered || active ? 1 : 0,
                      transition: 'opacity 0.12s',
                    }}
                  >
                    <Popconfirm
                      title="确定删除此话题？"
                      onConfirm={(e) => {
                        e?.stopPropagation()
                        onDelete(topic.id)
                      }}
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
                          width: 26,
                          height: 26,
                          padding: 0,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          borderRadius: 6,
                          transition: 'color 0.15s, background 0.15s',
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
    </div>
  )
}

export default ChatTopicSidebar
