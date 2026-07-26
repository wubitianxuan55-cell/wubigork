import React, { useState, useRef, useEffect } from 'react'
import { Input, Button, Space, Typography, Avatar } from 'antd'
import { SendOutlined, RobotOutlined, UserOutlined, RightOutlined, UpOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

export interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  streaming?: boolean
}

interface ChatPanelProps {
  title: string
  messages: Message[]
  onSend: (userMsg: string) => Promise<string | void>
  onMessagesChange?: (msgs: Message[]) => void
  streaming?: boolean
  placeholder?: string
  extra?: React.ReactNode
  defaultCollapsed?: boolean
  /** 外部触发的自动发送——设置后自动展开面板并发起对话 */
  autoSend?: string
  /** autoSend 完成后回调 */
  onAutoSendDone?: () => void
  /** 撑满父容器高度（忽略默认 280px） */
  fillHeight?: boolean
}

let msgId = 0
function nextId() {
  msgId++
  return `msg_${msgId}_${Date.now()}`
}

const ChatPanel: React.FC<ChatPanelProps> = ({
  title,
  messages,
  onSend,
  onMessagesChange,
  streaming = false,
  placeholder = '输入消息...',
  extra,
  defaultCollapsed = false,
  autoSend,
  onAutoSendDone,
  fillHeight = false,
}) => {
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [streamText, setStreamText] = useState('')
  const [collapsed, setCollapsed] = useState(defaultCollapsed)
  const listRef = useRef<HTMLDivElement>(null)
  const sendRef = useRef<(() => void) | undefined>(undefined)

  // 当监听到外部 autoSend → 展开面板 → 设输入 → 自动发送
  useEffect(() => {
    if (!autoSend) return
    setCollapsed(false)
    setInput(autoSend)
    const timer = setTimeout(() => {
      sendRef.current?.()
      onAutoSendDone?.()
    }, 100)
    return () => clearTimeout(timer)
  }, [autoSend])

  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight
    }
  }, [messages, streamText])

  async function handleSendImpl() {
    const msg = input.trim()
    if (!msg || loading) return
    setInput('')
    setLoading(true)

    const newMessages = [...messages, { id: nextId(), role: 'user' as const, content: msg }]
    onMessagesChange?.(newMessages)

    const aiMsg: Message = { id: nextId(), role: 'assistant', content: '', streaming: true }
    const withAi = [...newMessages, aiMsg]

    try {
      onMessagesChange?.(withAi)

      const result = await onSend(msg)
      if (typeof result === 'string') {
        const text = result
        setStreamText('')
        for (let i = 0; i < text.length; i++) {
          setStreamText(text.slice(0, i + 1))
          await new Promise((r) => setTimeout(r, 15))
        }
        aiMsg.content = text
        aiMsg.streaming = false
        setStreamText('')
        // 关键修复：通知父组件最终内容
        onMessagesChange?.([...withAi])
      }
    } catch (err: any) {
      const last = withAi[withAi.length - 1]
      if (last && last.role === 'assistant') {
        last.content = `❌ 错误: ${err.message || err}`
        last.streaming = false
      }
      // 关键修复：通知父组件错误消息
      onMessagesChange?.([...withAi])
    } finally {
      setLoading(false)
    }
  }

  // 把 send 方法暴露给 ref——autoSend useEffect 依赖它
  sendRef.current = handleSendImpl

  const handleSend = () => handleSendImpl()

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: collapsed ? 'auto' : fillHeight ? '100%' : 280, minHeight: 0, flex: fillHeight ? 1 : undefined }}>
      {/* 标题栏 */}
      <div
        onClick={() => { if (collapsed) setCollapsed(false) }}
        style={{
          padding: '6px 12px', borderBottom: collapsed ? 'none' : '1px solid ' + C('color-border'),
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          cursor: collapsed ? 'pointer' : 'default',
        }}>
        <Space size={4}>
          {collapsed
            ? <RightOutlined style={{ color: C('color-text-secondary'), fontSize: 10 }} />
            : null}
          <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>
            🤖 {title}
          </Typography.Text>
        </Space>
        <Space size={4}>
          {!collapsed && extra}
          {!collapsed && (
            <Button type="text" size="small" onClick={() => setCollapsed(true)}
              style={{ color: C('color-text-secondary'), fontSize: 10, padding: 0 }}>
              <UpOutlined />
            </Button>
          )}
        </Space>
      </div>

      {!collapsed && (
        <>
          {/* 消息列表 */}
          <div ref={listRef} style={{ flex: 1, overflow: 'auto', padding: '8px 12px' }}>
            {messages.length === 0 && (
              <div style={{ textAlign: 'center', color: '#666', marginTop: 48 }}>
                <RobotOutlined style={{ fontSize: 28, marginBottom: 8 }} />
                <div style={{ fontSize: 12 }}>输入消息，AI 辅助你创作</div>
              </div>
            )}
            {messages.map((msg) => (
              <div key={msg.id} style={{ marginBottom: 16, display: 'flex', gap: 10 }}>
                <Avatar
                  size={32}
                  icon={msg.role === 'user' ? <UserOutlined /> : <RobotOutlined />}
                  style={{ background: msg.role === 'user' ? C('color-primary') : C('color-border'), flexShrink: 0 }}
                />
                <div style={{ flex: 1 }}>
                  <Typography.Text strong style={{ color: msg.role === 'user' ? C('color-primary') : '#60a5fa', fontSize: 12 }}>
                    {msg.role === 'user' ? '你' : 'AI'}
                  </Typography.Text>
                  <div style={{
                    color: C('color-text'),
                    whiteSpace: 'pre-wrap',
                    lineHeight: 1.7,
                    marginTop: 4,
                  }}>
                    {msg.content}
                    {msg.streaming && <span className="cursor-blink" />}
                  </div>
                </div>
              </div>
            ))}
            {streaming && streamText && (
              <div style={{ marginBottom: 16, display: 'flex', gap: 10 }}>
                <Avatar size={32} icon={<RobotOutlined />} style={{ background: C('color-border') }} />
                <div style={{ color: C('color-text'), whiteSpace: 'pre-wrap', lineHeight: 1.7 }}>
                  {streamText}
                  <span className="cursor-blink" />
                </div>
              </div>
            )}
          </div>

          {/* 输入框 */}
          <div style={{ padding: '8px 12px', borderTop: '1px solid ' + C('color-border') }}>
            <Space.Compact style={{ width: '100%' }}>
              <Input
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onPressEnter={handleSend}
                placeholder={placeholder}
                disabled={loading}
                style={{ background: C('color-bg-container'), borderColor: C('color-border'), color: C('color-text') }}
              />
              <Button
                type="primary"
                icon={<SendOutlined />}
                onClick={handleSend}
                loading={loading}
                style={{ background: C('color-primary'), borderColor: C('color-primary') }}
              />
            </Space.Compact>
          </div>
        </>
      )}
    </div>
  )
}

export default ChatPanel
