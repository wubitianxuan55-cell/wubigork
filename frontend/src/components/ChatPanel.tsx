import React, { useState, useRef, useEffect } from 'react'
import { Input, Button, Space, Typography, Avatar, Tooltip } from 'antd'
import {
  SendOutlined,
  RobotOutlined,
  UserOutlined,
  RightOutlined,
  UpOutlined,
  CopyOutlined,
  CheckOutlined,
  ImportOutlined,
} from '@ant-design/icons'
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
  /** 提供后将 AI 消息内容应用到外部（如设定编辑器），AI 气泡上会显示「应用到设定」按钮 */
  onApply?: (msg: Message) => void | Promise<void>
  /** 应用按钮文案，默认「应用到设定」 */
  applyLabel?: string
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
  placeholder = '输入消息，Enter 发送 / Shift+Enter 换行',
  extra,
  defaultCollapsed = false,
  autoSend,
  onAutoSendDone,
  fillHeight = false,
  onApply,
  applyLabel = '应用到设定',
}) => {
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [streamText, setStreamText] = useState('')
  const [collapsed, setCollapsed] = useState(defaultCollapsed)
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const sendRef = useRef<(() => void) | undefined>(undefined)
  const inputRef = useRef<React.ComponentRef<typeof Input.TextArea>>(null)

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
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onAutoSendDone 为可选 props 回调，父组件（NovelSettingPage）未以 useCallback 包裹传入，补依赖会在回调身份变化时重置发送定时器
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
          await new Promise((r) => setTimeout(r, 12))
        }
        aiMsg.content = text
        aiMsg.streaming = false
        setStreamText('')
        onMessagesChange?.([...withAi])
      }
    } catch (err: unknown) {
      const last = withAi[withAi.length - 1]
      if (last && last.role === 'assistant') {
        last.content = `❌ 错误: ${err instanceof Error ? err.message : String(err)}`
        last.streaming = false
      }
      onMessagesChange?.([...withAi])
    } finally {
      setLoading(false)
    }
  }

  sendRef.current = handleSendImpl

  const handleSend = () => handleSendImpl()

  const handleCopy = async (content: string, id: string) => {
    try {
      await navigator.clipboard.writeText(content)
      setCopiedId(id)
      setTimeout(() => setCopiedId(null), 2000)
    } catch {
      // 降级方案
      const ta = document.createElement('textarea')
      ta.value = content
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
      setCopiedId(id)
      setTimeout(() => setCopiedId(null), 2000)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const panelHeight = collapsed ? 'auto' : fillHeight ? '100%' : 280

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: panelHeight,
        minHeight: 0,
        flex: fillHeight ? 1 : undefined,
        borderRadius: collapsed ? 8 : 12,
        background: C('color-bg-container'),
        border: `1px solid ${C('color-border')}`,
        overflow: 'hidden',
        transition: 'border-radius 0.2s',
      }}
    >
      {/* 标题栏 */}
      <div
        onClick={() => { if (collapsed) setCollapsed(false) }}
        style={{
          padding: collapsed ? '8px 14px' : '10px 16px',
          borderBottom: collapsed ? 'none' : `1px solid ${C('color-border')}`,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          cursor: collapsed ? 'pointer' : 'default',
          background: collapsed ? C('color-bg-container') : C('color-bg-elevated'),
          transition: 'background 0.2s, padding 0.2s',
          userSelect: 'none',
        }}
      >
        <Space size={8}>
          {collapsed ? (
            <RightOutlined style={{ color: C('color-text-secondary'), fontSize: 10 }} />
          ) : null}
          <RobotOutlined style={{ color: C('color-primary'), fontSize: collapsed ? 16 : 18 }} />
          <Typography.Text
            strong
            style={{
              color: C('color-text'),
              fontSize: collapsed ? 13 : 14,
            }}
          >
            {title}
          </Typography.Text>
          {messages.length > 0 && !collapsed && (
            <Typography.Text
              style={{
                color: C('color-text-secondary'),
                fontSize: 11,
                marginLeft: 4,
              }}
            >
              {messages.length} 条消息
            </Typography.Text>
          )}
        </Space>
        <Space size={4}>
          {!collapsed && extra}
          {!collapsed && (
            <Tooltip title="收起面板" placement="left">
              <Button
                type="text"
                size="small"
                icon={<UpOutlined />}
                onClick={(e) => { e.stopPropagation(); setCollapsed(true) }}
                style={{ color: C('color-text-secondary'), fontSize: 10 }}
              />
            </Tooltip>
          )}
        </Space>
      </div>

      {!collapsed && (
        <>
          {/* 消息列表 */}
          <div
            ref={listRef}
            style={{
              flex: 1,
              overflow: 'auto',
              padding: '16px',
              background: C('color-bg-container'),
            }}
          >
            {messages.length === 0 && !streaming && (
              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  height: '100%',
                  minHeight: 180,
                  textAlign: 'center',
                  padding: '24px 20px',
                }}
              >
                <div
                  style={{
                    width: 64,
                    height: 64,
                    borderRadius: 20,
                    background: `${C('color-primary')}12`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    marginBottom: 20,
                  }}
                >
                  <RobotOutlined style={{ fontSize: 32, color: C('color-primary') }} />
                </div>
                <Typography.Text
                  strong
                  style={{ color: C('color-text'), fontSize: 15, marginBottom: 8 }}
                >
                  开始对话
                </Typography.Text>
                <Typography.Text
                  style={{ color: C('color-text-secondary'), fontSize: 13, maxWidth: 280, lineHeight: 1.6 }}
                >
                  在下方输入消息，AI 将为你提供帮助。
                  <br />
                  支持写作辅助、角色设计、创意讨论等任务。
                </Typography.Text>
              </div>
            )}

            {messages.map((msg) => {
              const isUser = msg.role === 'user'
              return (
                <div
                  key={msg.id}
                  className="chat-message-item"
                  style={{
                    marginBottom: 20,
                    display: 'flex',
                    gap: 10,
                    flexDirection: isUser ? 'row-reverse' : 'row',
                    alignItems: 'flex-start',
                  }}
                >
                  <Avatar
                    size={34}
                    icon={isUser ? <UserOutlined /> : <RobotOutlined />}
                    style={{
                      background: isUser
                        ? `linear-gradient(135deg, ${C('color-primary')}, ${C('color-primary')}dd)`
                        : `linear-gradient(135deg, ${C('color-border')}, ${C('color-bg-elevated')})`,
                      flexShrink: 0,
                      boxShadow: isUser
                        ? `0 2px 8px ${C('color-primary')}33`
                        : '0 1px 4px rgba(0,0,0,0.06)',
                    }}
                  />
                  <div
                    style={{
                      flex: isUser ? undefined : 1,
                      maxWidth: '82%',
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: isUser ? 'flex-end' : 'flex-start',
                    }}
                  >
                    {/* 发送者标签 */}
                    <Typography.Text
                      style={{
                        color: isUser ? C('color-primary') : C('color-text-secondary'),
                        fontSize: 11,
                        fontWeight: 500,
                        marginBottom: 4,
                        marginLeft: isUser ? 0 : 2,
                        marginRight: isUser ? 2 : 0,
                      }}
                    >
                      {isUser ? '你' : 'AI'}
                    </Typography.Text>

                    {/* 消息气泡 */}
                    <div
                      className="chat-bubble"
                      style={{
                        position: 'relative',
                        padding: '10px 14px',
                        borderRadius: isUser ? '16px 4px 16px 16px' : '4px 16px 16px 16px',
                        background: isUser
                          ? 'var(--md-sys-color-primary-container)'
                          : C('color-bg-elevated'),
                        color: isUser ? 'var(--md-sys-color-on-primary-container)' : C('color-text'),
                        whiteSpace: 'pre-wrap',
                        lineHeight: 1.65,
                        fontSize: 13.5,
                        wordBreak: 'break-word',
                        boxShadow: isUser
                          ? `0 2px 12px ${C('color-primary')}22`
                          : '0 1px 3px rgba(0,0,0,0.04)',
                        border: isUser ? 'none' : `1px solid ${C('color-border')}`,
                      }}
                    >
                      {msg.content}
                      {msg.streaming && <span className="cursor-blink" />}

                      {/* 复制按钮 — hover 显示 */}
                      {msg.content && !msg.streaming && (
                        <Tooltip title={copiedId === msg.id ? '已复制' : '复制'} placement="top">
                          <Button
                            type="text"
                            size="small"
                            className="chat-copy-btn"
                            icon={
                              copiedId === msg.id ? (
                                <CheckOutlined style={{ color: 'var(--md-sys-color-success)' }} />
                              ) : (
                                <CopyOutlined />
                              )
                            }
                            onClick={(e) => {
                              e.stopPropagation()
                              handleCopy(msg.content, msg.id)
                            }}
                            style={{
                              position: 'absolute',
                              top: 4,
                              right: onApply && !isUser ? 34 : 4,
                              opacity: 0,
                              color: isUser ? 'var(--md-sys-color-on-primary-container)' : C('color-text-secondary'),
                              transition: `opacity var(--md-sys-transition-fast)`,
                              background: isUser ? 'color-mix(in srgb, var(--md-sys-color-on-primary-container) 12%, transparent)' : 'rgba(0,0,0,0.04)',
                              borderRadius: 6,
                              width: 28,
                              height: 28,
                              padding: 0,
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                            }}
                          />
                        </Tooltip>
                      )}

                      {/* 应用到设定按钮 — hover 显示（仅 AI 消息） */}
                      {!isUser && msg.content && !msg.streaming && onApply && (
                        <Tooltip title={applyLabel} placement="top">
                          <Button
                            type="text"
                            size="small"
                            className="chat-apply-btn"
                            icon={<ImportOutlined />}
                            aria-label={applyLabel}
                            onClick={(e) => {
                              e.stopPropagation()
                              void onApply(msg)
                            }}
                            style={{
                              position: 'absolute',
                              top: 4,
                              right: 4,
                              opacity: 0,
                              color: C('color-text-secondary'),
                              transition: `opacity var(--md-sys-transition-fast)`,
                              background: 'rgba(0,0,0,0.04)',
                              borderRadius: 6,
                              width: 28,
                              height: 28,
                              padding: 0,
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                            }}
                          />
                        </Tooltip>
                      )}
                    </div>
                  </div>
                </div>
              )
            })}

            {/* 流式输出中的临时消息 */}
            {streaming && streamText && (
              <div
                style={{
                  marginBottom: 20,
                  display: 'flex',
                  gap: 10,
                  alignItems: 'flex-start',
                }}
              >
                <Avatar
                  size={34}
                  icon={<RobotOutlined />}
                  style={{
                    background: `linear-gradient(135deg, ${C('color-border')}, ${C('color-bg-elevated')})`,
                    flexShrink: 0,
                    boxShadow: '0 1px 4px rgba(0,0,0,0.06)',
                  }}
                />
                <div style={{ flex: 1, maxWidth: '82%' }}>
                  <Typography.Text
                    style={{ color: C('color-text-secondary'), fontSize: 11, fontWeight: 500, marginBottom: 4, display: 'block', marginLeft: 2 }}
                  >
                    AI
                  </Typography.Text>
                  <div
                    style={{
                      padding: '10px 14px',
                      borderRadius: '4px 16px 16px 16px',
                      background: C('color-bg-elevated'),
                      border: `1px solid ${C('color-border')}`,
                      color: C('color-text'),
                      whiteSpace: 'pre-wrap',
                      lineHeight: 1.65,
                      fontSize: 13.5,
                      wordBreak: 'break-word',
                      boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
                    }}
                  >
                    {streamText}
                    <span className="cursor-blink" />
                  </div>
                </div>
              </div>
            )}

            {/* 加载中状态（发送中但还没开始流式输出） */}
            {loading && !streamText && messages.length > 0 && messages[messages.length - 1].role === 'user' && (
              <div style={{ display: 'flex', gap: 10, alignItems: 'flex-start', marginBottom: 20 }}>
                <Avatar
                  size={34}
                  icon={<RobotOutlined />}
                  style={{
                    background: `linear-gradient(135deg, ${C('color-border')}, ${C('color-bg-elevated')})`,
                    flexShrink: 0,
                  }}
                />
                <div
                  style={{
                    padding: '12px 18px',
                    borderRadius: '4px 16px 16px 16px',
                    background: C('color-bg-elevated'),
                    border: `1px solid ${C('color-border')}`,
                  }}
                >
                  <span className="typing-dots">
                    <span className="typing-dot" />
                    <span className="typing-dot" />
                    <span className="typing-dot" />
                  </span>
                </div>
              </div>
            )}
          </div>

          {/* 输入框 — 悬浮居中 */}
          <div
            style={{
              padding: '16px 0 20px',
              display: 'flex',
              justifyContent: 'center',
            }}
          >
            <div
              style={{
                width: '100%',
                maxWidth: 720,
                display: 'flex',
                alignItems: 'flex-end',
                gap: 8,
                padding: '8px 10px',
                background: C('color-bg-container'),
                border: `1px solid ${C('color-border')}`,
                borderRadius: 16,
                boxShadow: `0 4px 24px rgba(0,0,0,0.06), 0 1px 4px rgba(0,0,0,0.04)`,
                transition: 'box-shadow 0.2s',
              }}
            >
              <Input.TextArea
                ref={inputRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder={placeholder}
                disabled={loading}
                autoSize={{ minRows: 1, maxRows: 5 }}
                className="chat-input-textarea"
                style={{
                  flex: 1,
                  background: 'transparent',
                  border: 'none',
                  color: C('color-text'),
                  borderRadius: 0,
                  resize: 'none',
                  fontSize: 14,
                  lineHeight: 1.6,
                  padding: '6px 4px',
                  boxShadow: 'none',
                }}
              />
              <Button
                type="primary"
                icon={<SendOutlined />}
                onClick={handleSend}
                loading={loading}
                disabled={!input.trim() && !loading}
                style={{
                  background: input.trim() ? C('color-primary') : C('color-border'),
                  borderColor: 'transparent',
                  borderRadius: 12,
                  width: 38,
                  height: 38,
                  minWidth: 38,
                  padding: 0,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  boxShadow: input.trim()
                    ? `0 2px 8px ${C('color-primary')}44`
                    : 'none',
                  transition: 'all 0.2s',
                  flexShrink: 0,
                }}
              />
            </div>
          </div>
        </>
      )}
    </div>
  )
}

export default ChatPanel
