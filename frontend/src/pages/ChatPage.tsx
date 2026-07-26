import React, { useState, useCallback, useEffect, useRef } from 'react'
import { Input, Button, Avatar, Typography, Tooltip } from 'antd'
import {
  SendOutlined,
  RobotOutlined,
  UserOutlined,
  CopyOutlined,
  CheckOutlined,
  MessageOutlined,
} from '@ant-design/icons'
import ChatTopicSidebar, { type Topic } from '../components/ChatTopicSidebar'
import * as App from '../../wailsjs/go/app/App'
import { C } from '../utils/theme'

// ─── 类型 ─────────────────────────────────────────────
export interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  streaming?: boolean
}

interface StoredTopic {
  id: string
  title: string
  messages: Message[]
  createdAt: number
}

// ─── 持久化工具 ───────────────────────────────────────
const STORAGE_KEY = 'wubigrok_chat_topics'

function generateId(): string {
  return `topic_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}
let msgId = 0
function nextMsgId() {
  msgId++
  return `msg_${msgId}_${Date.now()}`
}

function loadTopics(): StoredTopic[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed) && parsed.length > 0) return parsed
    }
  } catch (_) { /* ignore */ }
  return [createTopic('新对话')]
}

function saveTopics(topics: StoredTopic[]): void {
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(topics)) } catch (_) { /* ignore */ }
}

function createTopic(title: string): StoredTopic {
  return { id: generateId(), title, messages: [], createdAt: Date.now() }
}

// ─── 组件 ─────────────────────────────────────────────
const ChatPage: React.FC = () => {
  const [topics, setTopics] = useState<StoredTopic[]>(() => loadTopics())
  const [activeId, setActiveId] = useState<string>(() => loadTopics()[0]?.id || '')
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [streamText, setStreamText] = useState('')
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<any>(null)

  useEffect(() => { saveTopics(topics) }, [topics])

  const activeTopic = topics.find((t) => t.id === activeId)
  const messages = activeTopic?.messages ?? []

  // 自动滚动到底部
  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight
    }
  }, [messages, streamText])

  // ─── 发送消息 ─────────────────────────────────────
  const handleSend = useCallback(async () => {
    const text = input.trim()
    if (!text || loading) return
    setInput('')
    setLoading(true)

    const userMsg: Message = { id: nextMsgId(), role: 'user', content: text }
    const aiMsg: Message = { id: nextMsgId(), role: 'assistant', content: '', streaming: true }

    setTopics((prev) => prev.map((t) =>
      t.id === activeId ? { ...t, messages: [...t.messages, userMsg, aiMsg] } : t
    ))

    try {
      const result = await App.ChatGeneral(text)
      const reply = (result as any)?.reply
      if (typeof reply === 'string') {
        setStreamText('')
        for (let i = 0; i < reply.length; i++) {
          setStreamText(reply.slice(0, i + 1))
          await new Promise((r) => setTimeout(r, 12))
        }
        setStreamText('')
        aiMsg.content = reply
        aiMsg.streaming = false
        setTopics((prev) => prev.map((t) =>
          t.id === activeId ? { ...t, messages: t.messages.map((m) => m.id === aiMsg.id ? { ...aiMsg } : m) } : t
        ))
      }
    } catch (err: any) {
      aiMsg.content = `❌ 错误: ${err.message || err}`
      aiMsg.streaming = false
      setTopics((prev) => prev.map((t) =>
        t.id === activeId ? { ...t, messages: t.messages.map((m) => m.id === aiMsg.id ? { ...aiMsg } : m) } : t
      ))
    } finally {
      setLoading(false)
    }
  }, [input, loading, activeId])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const handleCopy = async (content: string, id: string) => {
    try {
      await navigator.clipboard.writeText(content)
    } catch {
      const ta = document.createElement('textarea')
      ta.value = content
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
  }

  // ─── 话题操作 ─────────────────────────────────────
  const handleCreate = useCallback(() => {
    const topic = createTopic('新对话')
    setTopics((prev) => [...prev, topic])
    setActiveId(topic.id)
  }, [])

  const handleDelete = useCallback((id: string) => {
    setTopics((prev) => {
      const next = prev.filter((t) => t.id !== id)
      if (next.length === 0) {
        const fallback = createTopic('新对话')
        setActiveId(fallback.id)
        return [fallback]
      }
      if (id === activeId) {
        const idx = prev.findIndex((t) => t.id === id)
        const newActive = next[Math.min(idx, next.length - 1)]
        setActiveId(newActive.id)
      }
      return next
    })
  }, [activeId])

  const handleRename = useCallback((id: string, title: string) => {
    setTopics((prev) => prev.map((t) =>
      t.id === id ? { ...t, title } : t
    ))
  }, [])

  const topicList: Topic[] = topics.map(({ id, title, createdAt }) => ({ id, title, createdAt }))

  // ─── 渲染 ─────────────────────────────────────────
  const hasMessages = messages.length > 0
  const streamingMsg = messages.find((m) => m.streaming)

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'row', minHeight: 0 }}>
      {/* 左侧边栏 */}
      <ChatTopicSidebar
        topics={topicList}
        activeId={activeId}
        onSelect={setActiveId}
        onCreate={handleCreate}
        onDelete={handleDelete}
        onRename={handleRename}
      />

      {/* 右侧聊天区 */}
      <div style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        minWidth: 0,
        minHeight: 0,
        overflow: 'hidden',
        background: C('color-bg-container'),
        position: 'relative',
      }}>
        {/* 消息列表 */}
        <div
          ref={listRef}
          style={{
            flex: 1,
            minHeight: 0,
            overflow: 'auto',
            padding: hasMessages ? '24px 0 160px' : '0',
          }}
        >
          {!hasMessages ? (
            /* ── 欢迎界面 ── */
            <div style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              padding: '48px 32px',
              overflow: 'auto',
            }}>
              {/* 品牌标识 */}
              <div style={{
                width: 88,
                height: 88,
                borderRadius: 26,
                background: `linear-gradient(135deg, ${C('color-primary')}, ${C('color-primary')}cc)`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                marginBottom: 28,
                boxShadow: `0 8px 32px ${C('color-primary')}33`,
              }}>
                <RobotOutlined style={{ fontSize: 44, color: '#fff' }} />
              </div>

              <Typography.Text
                style={{ color: C('color-text'), fontSize: 24, fontWeight: 700, marginBottom: 6 }}
              >
                wubigork AI
              </Typography.Text>
              <Typography.Text
                style={{ color: C('color-text-secondary'), fontSize: 14, marginBottom: 32, textAlign: 'center', lineHeight: 1.6, maxWidth: 400 }}
              >
                你的智能 AI 助手——聊天、写作、翻译、学习，随时随地
              </Typography.Text>

              {/* 快捷建议卡片 */}
              <div style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
                gap: 10,
                maxWidth: 600,
                width: '100%',
              }}>
                {[
                  { icon: '💬', label: '随便聊聊', desc: '和 AI 畅聊任何话题' },
                  { icon: '🔍', label: '帮我查资料', desc: '快速搜索和整理信息' },
                  { icon: '📝', label: '写篇文章', desc: '博客、报告、文案随时生成' },
                  { icon: '💡', label: '头脑风暴', desc: '一起碰撞灵感火花' },
                  { icon: '🌐', label: '翻译内容', desc: '多语言互译，保持原意' },
                  { icon: '🧠', label: '解释概念', desc: '深入浅出地讲解知识点' },
                ].map((s) => (
                  <div
                    key={s.label}
                    onClick={() => { setInput(s.label); inputRef.current?.focus() }}
                    style={{
                      padding: '14px 16px',
                      borderRadius: 14,
                      background: C('color-bg-elevated'),
                      border: `1px solid ${C('color-border')}`,
                      cursor: 'pointer',
                      transition: 'all 0.15s',
                      userSelect: 'none',
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.borderColor = C('color-primary')
                      e.currentTarget.style.boxShadow = `0 4px 16px ${C('color-primary')}12`
                      e.currentTarget.style.transform = 'translateY(-2px)'
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.borderColor = C('color-border')
                      e.currentTarget.style.boxShadow = 'none'
                      e.currentTarget.style.transform = 'translateY(0)'
                    }}
                  >
                    <div style={{ fontSize: 20, marginBottom: 6 }}>{s.icon}</div>
                    <div style={{ color: C('color-text'), fontSize: 13, fontWeight: 500, marginBottom: 2 }}>
                      {s.label}
                    </div>
                    <div style={{ color: C('color-text-secondary'), fontSize: 11, lineHeight: 1.4 }}>
                      {s.desc}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            /* ── 消息列表 ── */
            <div style={{ maxWidth: 768, margin: '0 auto', padding: '0 24px' }}>
              {messages.map((msg) => {
                const isUser = msg.role === 'user'
                const isStreaming = msg.streaming && msg === streamingMsg
                const displayContent = isStreaming ? streamText : msg.content

                return (
                  <div
                    key={msg.id}
                    className="chat-message-item"
                    style={{
                      display: 'flex',
                      gap: 14,
                      marginBottom: 28,
                      flexDirection: isUser ? 'row-reverse' : 'row',
                      alignItems: 'flex-start',
                    }}
                  >
                    <Avatar
                      size={32}
                      icon={isUser ? <UserOutlined /> : <RobotOutlined />}
                      style={{
                        background: isUser
                          ? C('color-primary')
                          : C('color-bg-elevated'),
                        color: isUser ? '#fff' : C('color-text-secondary'),
                        flexShrink: 0,
                        marginTop: 2,
                      }}
                    />

                    <div style={{
                      flex: isUser ? undefined : 1,
                      maxWidth: isUser ? '70%' : '100%',
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: isUser ? 'flex-end' : 'flex-start',
                    }}>
                      <div style={{
                        color: C('color-text'),
                        whiteSpace: 'pre-wrap',
                        lineHeight: 1.75,
                        fontSize: 14,
                        wordBreak: 'break-word',
                        padding: isUser ? '8px 16px' : '0',
                        borderRadius: isUser ? 18 : 0,
                        background: isUser ? `${C('color-primary')}12` : 'transparent',
                      }}>
                        {displayContent}
                        {isStreaming && <span className="cursor-blink" />}
                      </div>

                      {/* 复制按钮 */}
                      {msg.content && !msg.streaming && (
                        <Tooltip title={copiedId === msg.id ? '已复制' : '复制'}>
                          <Button
                            type="text"
                            size="small"
                            icon={copiedId === msg.id ? <CheckOutlined style={{ color: '#52c41a' }} /> : <CopyOutlined />}
                            onClick={() => handleCopy(msg.content, msg.id)}
                            style={{
                              color: C('color-text-secondary'),
                              marginTop: 4,
                              opacity: 0.4,
                              fontSize: 12,
                              padding: '0 4px',
                              height: 22,
                            }}
                          />
                        </Tooltip>
                      )}

                      {/* 流式光标占位 */}
                      {isStreaming && !displayContent && (
                        <span className="cursor-blink" style={{ marginTop: 6, display: 'inline-block' }} />
                      )}
                    </div>
                  </div>
                )
              })}

              {/* 加载中（等后端返回） */}
              {loading && !streamText && !streamingMsg && (
                <div style={{ display: 'flex', gap: 14, marginBottom: 28 }}>
                  <Avatar
                    size={32}
                    icon={<RobotOutlined />}
                    style={{ background: C('color-bg-elevated'), color: C('color-text-secondary') }}
                  />
                  <div style={{ padding: '10px 0' }}>
                    <span className="typing-dots">
                      <span className="typing-dot" />
                      <span className="typing-dot" />
                      <span className="typing-dot" />
                    </span>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* ── 悬浮输入框 ── */}
        <div style={{
          position: 'absolute',
          bottom: 0,
          left: 0,
          right: 0,
          display: 'flex',
          justifyContent: 'center',
          padding: '0 24px 24px',
          pointerEvents: 'none',
        }}>
          <div style={{
            width: '100%',
            maxWidth: 768,
            display: 'flex',
            alignItems: 'flex-end',
            gap: 8,
            padding: '10px 14px',
            background: C('color-bg-container'),
            border: `1px solid ${C('color-border')}`,
            borderRadius: 20,
            boxShadow: `0 8px 32px rgba(0,0,0,0.08), 0 2px 8px rgba(0,0,0,0.04)`,
            pointerEvents: 'auto',
          }}>
            <Input.TextArea
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="输入消息，Enter 发送 / Shift+Enter 换行"
              disabled={loading}
              autoSize={{ minRows: 1, maxRows: 6 }}
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
                padding: '6px 2px',
                boxShadow: 'none',
              }}
            />
            <Tooltip title="发送">
              <Button
                type="primary"
                icon={<SendOutlined />}
                onClick={handleSend}
                loading={loading}
                disabled={!input.trim() && !loading}
                style={{
                  background: input.trim() ? C('color-primary') : C('color-border'),
                  borderColor: 'transparent',
                  borderRadius: 14,
                  width: 40,
                  height: 40,
                  minWidth: 40,
                  padding: 0,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  boxShadow: input.trim() ? `0 2px 10px ${C('color-primary')}44` : 'none',
                  transition: 'all 0.2s',
                  flexShrink: 0,
                }}
              />
            </Tooltip>
          </div>
        </div>
      </div>
    </div>
  )
}

export default ChatPage
