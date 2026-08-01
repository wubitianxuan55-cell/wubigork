import React, { useState, useCallback, useEffect, useRef } from 'react'
import { Input, Button, Avatar, Typography, Tooltip, message } from 'antd'
import {
  SendOutlined, RobotOutlined, UserOutlined, CopyOutlined, CheckOutlined,
  MessageOutlined, SoundOutlined,
} from '@ant-design/icons'
import ChatTopicSidebar, { type Topic } from '../components/ChatTopicSidebar'
import * as App from '../../wailsjs/go/app/App'
import { C } from '../utils/theme'
import { useFeatureModel } from '../hooks/useFeatureModel'

export interface Message {
  id: string; role: 'user' | 'assistant'; content: string; streaming?: boolean
}
interface StoredTopic {
  id: string; title: string; messages: Message[]; createdAt: number
}

const STORAGE_KEY = 'gaea_chat_topics'
const LEGACY_STORAGE_KEY = 'wubigrok_chat_topics'
function generateId(): string { return `topic_${Date.now()}_${Math.random().toString(36).slice(2, 8)}` }
let msgId = 0
function nextMsgId() { msgId++; return `msg_${msgId}_${Date.now()}` }
function loadTopics(): StoredTopic[] {
  try { const raw = localStorage.getItem(STORAGE_KEY) ?? localStorage.getItem(LEGACY_STORAGE_KEY); if (raw) { const p = JSON.parse(raw); if (Array.isArray(p) && p.length > 0) return p } } catch (_) {}
  return [createTopic('新对话')]
}
function saveTopics(topics: StoredTopic[]): void { try { localStorage.setItem(STORAGE_KEY, JSON.stringify(topics)) } catch (_) {} }
function createTopic(title: string): StoredTopic { return { id: generateId(), title, messages: [], createdAt: Date.now() } }

const ChatPage: React.FC = () => {
  const [topics, setTopics] = useState<StoredTopic[]>(() => loadTopics())
  const [activeId, setActiveId] = useState<string>(() => loadTopics()[0]?.id || '')
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [streamText, setStreamText] = useState('')
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [speakingId, setSpeakingId] = useState<string | null>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<any>(null)

  useEffect(() => { saveTopics(topics) }, [topics])
  const activeTopic = topics.find(t => t.id === activeId)
  const messages = activeTopic?.messages ?? []

  useEffect(() => { if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight }, [messages, streamText])

  // 聊天功能级模型（持久化，模型中心绑定）
  const chatModel = useFeatureModel('chat')
  const chatModelLabel = chatModel.model ? `${chatModel.engine || ''}/${chatModel.model}` : ''

  const handleSend = useCallback(async () => {
    const text = input.trim(); if (!text || loading) return
    setInput(''); setLoading(true)
    const userMsg: Message = { id: nextMsgId(), role: 'user', content: text }
    const aiMsg: Message = { id: nextMsgId(), role: 'assistant', content: '', streaming: true }
    setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: [...t.messages, userMsg, aiMsg] } : t))
    try {
      const result = await App.ChatGeneral(text)
      const reply = (result as any)?.reply
      if (typeof reply === 'string') {
        setStreamText('')
        for (let i = 0; i < reply.length; i++) { setStreamText(reply.slice(0, i + 1)); await new Promise(r => setTimeout(r, 12)) }
        setStreamText(''); aiMsg.content = reply; aiMsg.streaming = false
        setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: t.messages.map(m => m.id === aiMsg.id ? { ...aiMsg } : m) } : t))
      }
    } catch (err: any) {
      aiMsg.content = `❌ 错误: ${err.message || err}`; aiMsg.streaming = false
      setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: t.messages.map(m => m.id === aiMsg.id ? { ...aiMsg } : m) } : t))
    } finally { setLoading(false) }
  }, [input, loading, activeId])

  const handleKeyDown = (e: React.KeyboardEvent) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }

  const handleCopy = async (content: string, id: string) => {
    try { await navigator.clipboard.writeText(content) } catch { const ta = document.createElement('textarea'); ta.value = content; document.body.appendChild(ta); ta.select(); document.execCommand('copy'); document.body.removeChild(ta) }
    setCopiedId(id); setTimeout(() => setCopiedId(null), 2000)
  }

  const handleSpeak = async (content: string, id: string) => {
    if (speakingId) return
    setSpeakingId(id)
    try {
      // @ts-ignore
      const result = await App.TTSSpeakBase64(content)
      console.log('[TTS] result:', result)
      if (result?.base64) {
        const binary = atob(result.base64)
        const bytes = new Uint8Array(binary.length)
        for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
        const audio = new Audio(URL.createObjectURL(new Blob([bytes], { type: result.mimeType || 'audio/mp3' })))
        audio.onended = () => setSpeakingId(null)
        audio.onerror = (e) => { console.error('[TTS] audio error:', e); setSpeakingId(null) }
        await audio.play()
        return
      }
      console.warn('[TTS] 无 base64 数据')
      message.warning('TTS 未返回音频数据')
    } catch (err: any) {
      const msg = typeof err === 'string' ? err : err?.message || err?.error || 'TTS 失败'
      console.error('[TTS]', msg, err)
      message.error(`朗读失败: ${msg}`)
    }
    setSpeakingId(null)
  }

  const handleCreate = useCallback(() => { const t = createTopic('新对话'); setTopics(prev => [...prev, t]); setActiveId(t.id) }, [])
  const handleDelete = useCallback((id: string) => { setTopics(prev => { const next = prev.filter(t => t.id !== id); if (next.length === 0) { const fb = createTopic('新对话'); setActiveId(fb.id); return [fb] } if (id === activeId) { const idx = prev.findIndex(t => t.id === id); setActiveId(next[Math.min(idx, next.length - 1)].id) } return next }) }, [activeId])
  const handleRename = useCallback((id: string, title: string) => { setTopics(prev => prev.map(t => t.id === id ? { ...t, title } : t)) }, [])

  const topicList: Topic[] = topics.map(({ id, title, createdAt }) => ({ id, title, createdAt }))

  const hasMessages = messages.length > 0
  const streamingMsg = messages.find(m => m.streaming)

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'row', minHeight: 0 }}>
      <ChatTopicSidebar topics={topicList} activeId={activeId} onSelect={setActiveId} onCreate={handleCreate} onDelete={handleDelete} onRename={handleRename} />

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, minHeight: 0, overflow: 'hidden', background: 'transparent', position: 'relative' }}>
        <div ref={listRef} style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: hasMessages ? '24px 0 160px' : '0' }}>
          {!hasMessages ? (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', padding: '48px 32px', overflow: 'auto' }}>
              <div style={{ width: 88, height: 88, borderRadius: 26, background: `linear-gradient(135deg, ${C('color-primary')}, ${C('color-primary')}cc)`, display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: 28, boxShadow: `0 8px 32px ${C('color-primary')}33, 0 0 42px color-mix(in srgb, var(--gaea-glow) 30%, transparent)`, border: '1px solid var(--gaea-glow)' }}><RobotOutlined style={{ fontSize: 44, color: '#fff' }} /></div>
              <Typography.Text style={{ color: C('color-text'), fontSize: 24, fontWeight: 700, marginBottom: 6 }}>gaea AI</Typography.Text>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14, marginBottom: 32, textAlign: 'center', lineHeight: 1.6, maxWidth: 400 }}>你的智能 AI 助手——聊天、写作、翻译、学习，随时随地</Typography.Text>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: 10, maxWidth: 600, width: '100%' }}>
                {[{ icon: '💬', label: '随便聊聊', desc: '和 AI 畅聊任何话题' }, { icon: '🔍', label: '帮我查资料', desc: '快速搜索和整理信息' }, { icon: '📝', label: '写篇文章', desc: '博客、报告、文案随时生成' }, { icon: '💡', label: '头脑风暴', desc: '一起碰撞灵感火花' }, { icon: '🌐', label: '翻译内容', desc: '多语言互译，保持原意' }, { icon: '🧠', label: '解释概念', desc: '深入浅出地讲解知识点' }].map(s => (
                  <div key={s.label} className="chat-suggestion-card" onClick={() => { setInput(s.label); inputRef.current?.focus() }} style={{ padding: '14px 16px', borderRadius: 14, background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))', WebkitBackdropFilter: 'blur(12px) saturate(130%)', backdropFilter: 'blur(12px) saturate(130%)', border: '1px solid var(--md-sys-color-outline-variant)', cursor: 'pointer', transition: 'all 0.18s', userSelect: 'none' }}
                    onMouseEnter={e => { e.currentTarget.style.borderColor = 'var(--gaea-glow)'; e.currentTarget.style.boxShadow = '0 4px 20px color-mix(in srgb, var(--gaea-glow) 22%, transparent)'; e.currentTarget.style.transform = 'translateY(-2px)' }}
                    onMouseLeave={e => { e.currentTarget.style.borderColor = 'var(--md-sys-color-outline-variant)'; e.currentTarget.style.boxShadow = 'none'; e.currentTarget.style.transform = 'translateY(0)' }}>
                    <div style={{ fontSize: 20, marginBottom: 6 }}>{s.icon}</div><div style={{ color: C('color-text'), fontSize: 13, fontWeight: 500, marginBottom: 2 }}>{s.label}</div><div style={{ color: C('color-text-secondary'), fontSize: 11, lineHeight: 1.4 }}>{s.desc}</div>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div style={{ maxWidth: 768, margin: '0 auto', padding: '0 24px' }}>
              {messages.map(msg => {
                const isUser = msg.role === 'user'
                const isStreaming = msg.streaming && msg === streamingMsg
                const displayContent = isStreaming ? streamText : msg.content
                return (
                  <div key={msg.id} className="chat-message-item" style={{ display: 'flex', gap: 14, marginBottom: 28, flexDirection: isUser ? 'row-reverse' : 'row', alignItems: 'flex-start' }}>
                    <Avatar size={32} icon={isUser ? <UserOutlined /> : <RobotOutlined />} style={{ background: isUser ? C('color-primary') : C('color-bg-elevated'), color: isUser ? '#fff' : C('color-text-secondary'), flexShrink: 0, marginTop: 2 }} />
                    <div style={{ flex: isUser ? undefined : 1, maxWidth: isUser ? '70%' : '100%', display: 'flex', flexDirection: 'column', alignItems: isUser ? 'flex-end' : 'flex-start' }}>
                      <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.75, fontSize: 14, wordBreak: 'break-word',
                        padding: '10px 16px',
                        borderRadius: isUser ? 18 : 16,
                        background: isUser
                          ? `linear-gradient(135deg, ${C('color-primary')}, color-mix(in srgb, ${C('color-primary')} 78%, #000))`
                          : 'var(--gaea-glass-bg, transparent)',
                        color: isUser ? '#fff' : C('color-text'),
                        WebkitBackdropFilter: isUser ? undefined : 'blur(14px) saturate(130%)',
                        backdropFilter: isUser ? undefined : 'blur(14px) saturate(130%)',
                        border: isUser ? 'none' : '1px solid var(--md-sys-color-outline-variant)',
                        borderLeft: isUser ? 'none' : '3px solid var(--gaea-glow)',
                        boxShadow: isUser ? `0 4px 18px ${C('color-primary')}44` : '0 4px 18px rgba(0,0,0,0.10)' }}>
                        {displayContent}{isStreaming && <span className="cursor-blink" />}
                      </div>
                      {msg.content && !msg.streaming && (
                        <div style={{ display: 'flex', gap: 2, marginTop: 4 }}>
                          <Tooltip title={copiedId === msg.id ? '已复制' : '复制'}>
                            <Button type="text" size="small" icon={copiedId === msg.id ? <CheckOutlined style={{ color: '#52c41a' }} /> : <CopyOutlined />}
                              onClick={() => handleCopy(msg.content, msg.id)}
                              style={{ color: C('color-text-secondary'), opacity: 0.4, fontSize: 12, padding: '0 4px', height: 22 }} />
                          </Tooltip>
                          {!isUser && (
                            <Tooltip title={speakingId === msg.id ? '朗读中...' : '朗读'}>
                              <Button type="text" size="small" icon={<SoundOutlined />}
                                loading={speakingId === msg.id}
                                onClick={() => handleSpeak(msg.content, msg.id)}
                                style={{ color: C('color-text-secondary'), opacity: 0.4, fontSize: 12, padding: '0 4px', height: 22 }} />
                            </Tooltip>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
              {loading && !streamText && !streamingMsg && (
                <div style={{ display: 'flex', gap: 14, marginBottom: 28 }}>
                  <Avatar size={32} icon={<RobotOutlined />} style={{ background: C('color-bg-elevated'), color: C('color-text-secondary') }} />
                  <div style={{ padding: '10px 0' }}><span className="typing-dots"><span className="typing-dot" /><span className="typing-dot" /><span className="typing-dot" /></span></div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* 输入框 */}
        <div style={{ position: 'absolute', bottom: 0, left: 0, right: 0, display: 'flex', justifyContent: 'center', padding: '0 24px 24px', pointerEvents: 'none' }}>
          <div className="chat-input-wrap" style={{ width: '100%', maxWidth: 768, display: 'flex', alignItems: 'flex-end', gap: 6, padding: '8px 12px', background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))', WebkitBackdropFilter: 'blur(20px) saturate(150%)', backdropFilter: 'blur(20px) saturate(150%)', border: '1px solid var(--md-sys-color-outline-variant)', borderRadius: 20, boxShadow: '0 8px 32px rgba(0,0,0,0.12)', pointerEvents: 'auto', transition: 'border-color 0.2s, box-shadow 0.2s' }}>
            <Input.TextArea ref={inputRef} value={input} onChange={e => setInput(e.target.value)} onKeyDown={handleKeyDown}
              placeholder="输入消息，Enter 发送 / Shift+Enter 换行" disabled={loading}
              autoSize={{ minRows: 1, maxRows: 6 }} className="chat-input-textarea"
              style={{ flex: 1, background: 'transparent', border: 'none', color: C('color-text'), borderRadius: 0, resize: 'none', fontSize: 14, lineHeight: 1.6, padding: '6px 2px', boxShadow: 'none' }} />
            <Tooltip title="发送">
              <Button type="primary" icon={<SendOutlined />} onClick={handleSend} loading={loading} disabled={!input.trim() && !loading}
                style={{ background: input.trim() ? C('color-primary') : C('color-border'), borderColor: 'transparent', borderRadius: 14, width: 40, height: 40, minWidth: 40, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: input.trim() ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 45%, transparent)' : 'none', flexShrink: 0 }} />
            </Tooltip>
          </div>
        </div>
      </div>
    </div>
  )
}

export default ChatPage
