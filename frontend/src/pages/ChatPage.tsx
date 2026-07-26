import React, { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { Input, Button, Avatar, Typography, Tooltip, message } from 'antd'
import {
  SendOutlined, RobotOutlined, UserOutlined, CopyOutlined, CheckOutlined,
  MessageOutlined, AudioOutlined, SoundOutlined,
} from '@ant-design/icons'
import ChatTopicSidebar, { type Topic } from '../components/ChatTopicSidebar'
import VoiceChatOrb from '../components/VoiceChatOrb'
import { useVoiceChat } from '../hooks/useVoiceChat'
import * as App from '../../wailsjs/go/app/App'
import { C } from '../utils/theme'

export interface Message {
  id: string; role: 'user' | 'assistant'; content: string; streaming?: boolean
}
interface StoredTopic {
  id: string; title: string; messages: Message[]; createdAt: number
}

const STORAGE_KEY = 'wubigrok_chat_topics'
function generateId(): string { return `topic_${Date.now()}_${Math.random().toString(36).slice(2, 8)}` }
let msgId = 0
function nextMsgId() { msgId++; return `msg_${msgId}_${Date.now()}` }
function loadTopics(): StoredTopic[] {
  try { const raw = localStorage.getItem(STORAGE_KEY); if (raw) { const p = JSON.parse(raw); if (Array.isArray(p) && p.length > 0) return p } } catch (_) {}
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

  const handleTTS = useCallback(async (text: string) => {
    try {
      // @ts-ignore
      const result = await App.TTSSpeakBase64(text)
      if (result?.base64) {
        const binary = atob(result.base64)
        const bytes = new Uint8Array(binary.length)
        for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
        const blob = new Blob([bytes], { type: result.mimeType || 'audio/mp3' })
        const url = URL.createObjectURL(blob)
        const audio = new Audio(url)
        await audio.play()
        URL.revokeObjectURL(url)
      }
    } catch (_) {}
  }, [])

  const handleSpeechResult = useCallback(async (text: string): Promise<string> => {
    const userMsg: Message = { id: nextMsgId(), role: 'user', content: text }
    const aiMsg: Message = { id: nextMsgId(), role: 'assistant', content: '', streaming: true }
    setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: [...t.messages, userMsg, aiMsg] } : t))
    try {
      const result = await App.ChatGeneral(text)
      const reply = (result as any)?.reply || ''
      aiMsg.content = reply; aiMsg.streaming = false
      setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: t.messages.map(m => m.id === aiMsg.id ? { ...aiMsg } : m) } : t))
      return reply
    } catch (err: any) {
      aiMsg.content = `❌ 错误: ${err.message || err}`; aiMsg.streaming = false
      setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: t.messages.map(m => m.id === aiMsg.id ? { ...aiMsg } : m) } : t))
      throw err
    }
  }, [activeId])

  const { state: voice, start: startVoice, stop: stopVoice } = useVoiceChat({ onSpeechResult: handleSpeechResult, onTTS: handleTTS })

  useEffect(() => { saveTopics(topics) }, [topics])
  const activeTopic = topics.find(t => t.id === activeId)
  const messages = activeTopic?.messages ?? []

  useEffect(() => { if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight }, [messages, streamText])

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
  const orbSize = useMemo(() => Math.min(360, typeof window !== 'undefined' ? window.innerWidth * 0.5 : 360), [])
  const [voiceModelInfo, setVoiceModelInfo] = useState({ llm: '', tts: '', stt: '浏览器语音识别' })
  useEffect(() => { if (!voice.active) return; (async () => { try { /* @ts-ignore */ const [engine, model] = await Promise.all([App.GetActiveEngine(), App.GetActiveModel()]); const en = engine === 'xai' ? 'xAI' : engine === 'herdsman' ? 'Herdsman' : engine === 'ollama' ? 'Ollama' : engine; setVoiceModelInfo(prev => ({ ...prev, llm: `${en} / ${model || '默认'}`, tts: 'Herdsman / qwen3-tts' })) } catch (_) { setVoiceModelInfo({ llm: '默认', tts: 'Herdsman', stt: '浏览器' }) } })() }, [voice.active])

  const hasMessages = messages.length > 0
  const streamingMsg = messages.find(m => m.streaming)

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'row', minHeight: 0 }}>
      <ChatTopicSidebar topics={topicList} activeId={activeId} onSelect={setActiveId} onCreate={handleCreate} onDelete={handleDelete} onRename={handleRename} />

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, minHeight: 0, overflow: 'hidden', background: C('color-bg-container'), position: 'relative' }}>
        <div ref={listRef} style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: hasMessages ? '24px 0 160px' : '0' }}>
          {!hasMessages ? (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', padding: '48px 32px', overflow: 'auto' }}>
              <div style={{ width: 88, height: 88, borderRadius: 26, background: `linear-gradient(135deg, ${C('color-primary')}, ${C('color-primary')}cc)`, display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: 28, boxShadow: `0 8px 32px ${C('color-primary')}33` }}><RobotOutlined style={{ fontSize: 44, color: '#fff' }} /></div>
              <Typography.Text style={{ color: C('color-text'), fontSize: 24, fontWeight: 700, marginBottom: 6 }}>wubigork AI</Typography.Text>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14, marginBottom: 32, textAlign: 'center', lineHeight: 1.6, maxWidth: 400 }}>你的智能 AI 助手——聊天、写作、翻译、学习，随时随地</Typography.Text>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: 10, maxWidth: 600, width: '100%' }}>
                {[{ icon: '💬', label: '随便聊聊', desc: '和 AI 畅聊任何话题' }, { icon: '🔍', label: '帮我查资料', desc: '快速搜索和整理信息' }, { icon: '📝', label: '写篇文章', desc: '博客、报告、文案随时生成' }, { icon: '💡', label: '头脑风暴', desc: '一起碰撞灵感火花' }, { icon: '🌐', label: '翻译内容', desc: '多语言互译，保持原意' }, { icon: '🧠', label: '解释概念', desc: '深入浅出地讲解知识点' }].map(s => (
                  <div key={s.label} onClick={() => { setInput(s.label); inputRef.current?.focus() }} style={{ padding: '14px 16px', borderRadius: 14, background: C('color-bg-elevated'), border: `1px solid ${C('color-border')}`, cursor: 'pointer', transition: 'all 0.15s', userSelect: 'none' }}
                    onMouseEnter={e => { e.currentTarget.style.borderColor = C('color-primary'); e.currentTarget.style.boxShadow = `0 4px 16px ${C('color-primary')}12`; e.currentTarget.style.transform = 'translateY(-2px)' }}
                    onMouseLeave={e => { e.currentTarget.style.borderColor = C('color-border'); e.currentTarget.style.boxShadow = 'none'; e.currentTarget.style.transform = 'translateY(0)' }}>
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
                      <div style={{ color: C('color-text'), whiteSpace: 'pre-wrap', lineHeight: 1.75, fontSize: 14, wordBreak: 'break-word', padding: isUser ? '8px 16px' : '0', borderRadius: isUser ? 18 : 0, background: isUser ? `${C('color-primary')}12` : 'transparent' }}>
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
          <div style={{ width: '100%', maxWidth: 768, display: 'flex', alignItems: 'flex-end', gap: 6, padding: '8px 12px', background: C('color-bg-container'), border: `1px solid ${C('color-border')}`, borderRadius: 20, boxShadow: `0 8px 32px rgba(0,0,0,0.08)`, pointerEvents: 'auto' }}>
            <Tooltip title={voice.active ? '退出语音' : '语音聊天'}>
              <Button type="text" icon={<AudioOutlined />} onClick={() => voice.active ? stopVoice() : startVoice()}
                style={{ color: voice.active ? C('color-primary') : C('color-text-secondary'), borderRadius: 12, width: 36, height: 36, minWidth: 36, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: voice.active ? `${C('color-primary')}12` : 'transparent', flexShrink: 0 }} />
            </Tooltip>
            <Input.TextArea ref={inputRef} value={input} onChange={e => setInput(e.target.value)} onKeyDown={handleKeyDown}
              placeholder="输入消息，Enter 发送 / Shift+Enter 换行" disabled={loading || voice.active}
              autoSize={{ minRows: 1, maxRows: 6 }} className="chat-input-textarea"
              style={{ flex: 1, background: 'transparent', border: 'none', color: C('color-text'), borderRadius: 0, resize: 'none', fontSize: 14, lineHeight: 1.6, padding: '6px 2px', boxShadow: 'none' }} />
            <Tooltip title="发送">
              <Button type="primary" icon={<SendOutlined />} onClick={handleSend} loading={loading} disabled={(!input.trim() && !loading) || voice.active}
                style={{ background: input.trim() ? C('color-primary') : C('color-border'), borderColor: 'transparent', borderRadius: 14, width: 40, height: 40, minWidth: 40, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: input.trim() ? `0 2px 10px ${C('color-primary')}44` : 'none', flexShrink: 0 }} />
            </Tooltip>
          </div>
        </div>

        {/* 语音叠加层 */}
        {voice.active && (
          <div style={{ position: 'absolute', inset: 0, zIndex: 100, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', background: C('color-bg-container') }}>
            <VoiceChatOrb volume={voice.volume} listening={voice.listening} speaking={voice.speaking} aiSpeaking={voice.aiSpeaking} transcript={voice.transcript} size={orbSize} />
            <div style={{ marginTop: 28, display: 'flex', gap: 12, alignItems: 'center', fontSize: 11, color: C('color-text-secondary') }}>
              <span>🎙️ {voiceModelInfo.stt}</span><span style={{ opacity: 0.3 }}>→</span><span>💬 {voiceModelInfo.llm}</span><span style={{ opacity: 0.3 }}>→</span><span>🔊 {voiceModelInfo.tts}</span>
            </div>
            {voice.error && <Typography.Text style={{ color: '#fb7185', fontSize: 13, marginTop: 24 }}>{voice.error}</Typography.Text>}
            <Button onClick={stopVoice} style={{ marginTop: 32, borderRadius: 20, padding: '8px 28px', fontSize: 14, background: C('color-bg-elevated'), border: `1px solid ${C('color-border')}`, color: C('color-text-secondary') }}>退出语音模式</Button>
          </div>
        )}
      </div>
    </div>
  )
}

export default ChatPage
