import React, { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { Input, Button, Avatar, Typography, Tooltip, Card, Modal, Tag, message } from 'antd'
import {
  SendOutlined, UserOutlined, CopyOutlined, CheckOutlined,
  HeartOutlined, SwapOutlined, SoundOutlined, DeleteOutlined,
  PlusOutlined, MessageOutlined, ApiOutlined, ClearOutlined,
} from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'
import { C } from '../utils/theme'
import { WhisperEmotionPanel } from '../components/WhisperEmotionPanel'

interface Personality {

  id: string; label: string; gender: string; tags?: string[]
  dims: { T: number; I: number; S: number; O: number; R: number }
  voiceGuide?: string; requiresAdult18?: boolean
}
interface Message { id: string; role: 'user' | 'assistant'; content: string; streaming?: boolean; timestamp?: number }
interface Topic { id: string; title: string; messages: Message[]; createdAt: number }

const STORAGE_KEY = 'wubigrok_whisper_topics'
const PERSONALITY_KEY = 'wubigrok_whisper_personality'
const ACTIVE_TOPIC_KEY = 'wubigrok_whisper_active_topic'
let msgId = 0
function nextMsgId() { msgId++; return `wm_${msgId}_${Date.now()}` }
function genTopicId() { return `wt_${Date.now()}_${Math.random().toString(36).slice(2, 8)}` }
function loadTopics(): Topic[] {
  try { const r = localStorage.getItem(STORAGE_KEY); if (r) { const p = JSON.parse(r); if (Array.isArray(p) && p.length > 0) return p } } catch (_) {}
  return [createTopic('新对话')]
}
function saveTopics(t: Topic[]) { try { localStorage.setItem(STORAGE_KEY, JSON.stringify(t)) } catch (_) {} }
function createTopic(title: string): Topic { return { id: genTopicId(), title, messages: [], createdAt: Date.now() } }

const WhisperPage: React.FC = () => {
  const [topics, setTopics] = useState<Topic[]>(() => loadTopics())
  const [activeId, setActiveId] = useState<string>(() => {
    try { return localStorage.getItem(ACTIVE_TOPIC_KEY) || loadTopics()[0]?.id || '' } catch { return loadTopics()[0]?.id || '' }
  })
  const [editingTitle, setEditingTitle] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [personalities, setPersonalities] = useState<Personality[]>([])
  const [activePersonality, setActivePersonality] = useState<string>(() => {
    try { return localStorage.getItem(PERSONALITY_KEY) || 'deredere' } catch { return 'deredere' }
  })
  const [personalityOpen, setPersonalityOpen] = useState(false)
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [streamText, setStreamText] = useState('')
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [speakingId, setSpeakingId] = useState<string | null>(null)
  const [emotion, setEmotion] = useState('')
  const [stage, setStage] = useState(''); const [trust, setTrust] = useState(50)
  const [aff, setAff] = useState(0); const [sec, setSec] = useState(0); const [aro, setAro] = useState(0); const [dom, setDom] = useState(0)
  const [rifts, setRifts] = useState(0); const [totalTurns, setTotalTurns] = useState(0)
  const [engineName, setEngineName] = useState(''); const [modelName, setModelName] = useState('')
  const listRef = useRef<HTMLDivElement>(null); const inputRef = useRef<any>(null)
  const hasInitRef = useRef(false)

  const activeTopic = topics.find(t => t.id === activeId)
  const messages = activeTopic?.messages ?? []
  const currentPersonality = personalities.find(p => p.id === activePersonality)
  const streamingMsg = messages.find(m => m.streaming)
  const hasMessages = messages.length > 0
  const emoColors: Record<string,string> = {SWEET_ATTACHMENT:"#f472b6",SHY_HEARTBEAT:"#fb7185",TSUNDERE:"#f59e0b",HURT_GRIEVANCE:"#a78bfa",ANGRY_ATTACK:"#ef4444",COLD_DETACHED:"#94a3b8",FEARFUL_OBEDIENT:"#c084fc",QUIET_FOND:"#fbbf24",CALM_RATIONAL:"#60a5fa"}
  const emoColor = emoColors[emotion] || "#e85388"
  const topicList = useMemo(() => topics.map(({ id, title, createdAt }) => ({ id, title, createdAt })), [topics])
  useEffect(() => { localStorage.setItem(ACTIVE_TOPIC_KEY, activeId) }, [activeId])
  useEffect(() => { if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight }, [messages, streamText])
  useEffect(() => {
    if (hasInitRef.current || messages.length > 0 || personalities.length === 0) return
    hasInitRef.current = true
    const p = personalities.find(p => p.id === activePersonality)
    if (p) {
      setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: [{ id: nextMsgId(), role: 'assistant' as const, timestamp: Date.now(), content: `你好呀~ 我是「${p.label}」💫\n想和我聊什么呢？` }] } : t))
    }
  }, [personalities, activePersonality, activeId, messages.length])

  const handleSend = useCallback(async () => {
    const text = input.trim(); if (!text || loading) return
    setInput(''); setLoading(true)
    const um: Message = { id: nextMsgId(), role: 'user', content: text, timestamp: Date.now() }
    const am: Message = { id: nextMsgId(), role: 'assistant', content: '', streaming: true, timestamp: Date.now() }
    setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: [...t.messages, um, am] } : t))
    try {
      const res = await App.WhisperChat(text, activePersonality) as any
      const reply = res?.reply
      if (res?.emotion) setEmotion(res.emotion); if (res?.stage) setStage(res.stage)
      if (typeof res?.trust === 'number') setTrust(Math.round(res.trust))
      if (typeof res?.aff === 'number') setAff(Math.round(res.aff))
      if (typeof res?.sec === 'number') setSec(Math.round(res.sec))
      if (typeof res?.aro === 'number') setAro(Math.round(res.aro))
      if (typeof res?.dom === 'number') setDom(Math.round(res.dom))
      if (typeof res?.rifts === 'number') setRifts(res.rifts)
      if (typeof res?.totalTurns === 'number') setTotalTurns(res.totalTurns)
      if (typeof reply === 'string') {
        setStreamText('')
        for (let i = 0; i < reply.length; i++) { setStreamText(reply.slice(0, i + 1)); await new Promise(r => setTimeout(r, 12)) }
        setStreamText(''); am.content = reply; am.streaming = false
        setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: t.messages.map(m => m.id === am.id ? { ...am } : m) } : t))
        setTopics(prev => prev.map(t => { if (t.id !== activeId || t.title !== '新对话') return t; return { ...t, title: text.slice(0, 20) + (text.length > 20 ? '…' : '') } }))
      }
    } catch (err: any) { am.content = `❌ 错误: ${err.message || err}`; am.streaming = false; setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: t.messages.map(m => m.id === am.id ? { ...am } : m) } : t)) }
    finally { setLoading(false) }
  }, [input, loading, activeId, activePersonality])

  const handleKeyDown = (e: React.KeyboardEvent) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }
  const handleCopy = async (content: string, id: string) => {
    try { await navigator.clipboard.writeText(content) } catch { const ta = document.createElement('textarea'); ta.value = content; document.body.appendChild(ta); ta.select(); document.execCommand('copy'); document.body.removeChild(ta) }
    setCopiedId(id); setTimeout(() => setCopiedId(null), 2000)
  }
  const handleSpeak = async (content: string, id: string) => {
    if (speakingId) return; setSpeakingId(id)
    try {
      const result = await (App as any).TTSSpeakBase64(content)
      if (result?.base64) { const b = atob(result.base64); const bytes = new Uint8Array(b.length); for (let i = 0; i < b.length; i++) bytes[i] = b.charCodeAt(i); const a = new Audio(URL.createObjectURL(new Blob([bytes], { type: result.mimeType || 'audio/mp3' }))); a.onended = () => setSpeakingId(null); a.onerror = () => { setSpeakingId(null); message.error('播放失败') }; await a.play(); return }
      message.warning('TTS 未返回音频数据')
    } catch (err: any) { message.error(`朗读失败: ${typeof err === 'string' ? err : err?.message || '未知错误'}`) }
    setSpeakingId(null)
  }

  const handleCreateTopic = useCallback(() => { const t = createTopic('新对话'); setTopics(prev => [...prev, t]); setActiveId(t.id); setEmotion(''); setStage(''); setTrust(50); setAff(0); setSec(0); setAro(0); setDom(0); setRifts(0); setTotalTurns(0) }, [])
  const handleDeleteTopic = useCallback((id: string) => { setTopics(prev => { const next = prev.filter(t => t.id !== id); if (next.length === 0) { const fb = createTopic('新对话'); setActiveId(fb.id); setEmotion(''); return [fb] } if (id === activeId) { const idx = prev.findIndex(t => t.id === id); setActiveId(next[Math.min(idx, next.length - 1)].id); setEmotion('') } return next }) }, [activeId])
  const handleRenameTopic = useCallback((id: string) => { setEditingTitle(id); setEditValue(topics.find(t => t.id === id)?.title || '') }, [topics])
  const handleRenameConfirm = useCallback((id: string) => { if (editValue.trim()) { setTopics(prev => prev.map(t => t.id === id ? { ...t, title: editValue.trim() } : t)) }; setEditingTitle(null) }, [editValue])
  const handleClearMessages = useCallback(async () => {
    setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: [] } : t))
    setEmotion(''); setStage(''); setTrust(50); setAff(0); setSec(0); setAro(0); setDom(0); setRifts(0); setTotalTurns(0)
    try { await (App as any).WhisperClearSession(activePersonality) } catch (_) {}
  }, [activeId, activePersonality])

  const handleSwitchPersonality = useCallback(async (id: string) => {
    // 清除旧人格的后端会话
    try { await (App as any).WhisperClearSession(activePersonality) } catch (_) {}
    setActivePersonality(id); localStorage.setItem(PERSONALITY_KEY, id); setPersonalityOpen(false)
    // 重置前端状态
    setEmotion(''); setStage(''); setTrust(50); setAff(0); setSec(0); setAro(0); setDom(0); setRifts(0); setTotalTurns(0)
    // 添加切换提示消息
    const p = personalities.find(p => p.id === id)
    if (p) {
      setTopics(prev => prev.map(t => t.id === activeId ? {
        ...t,
        messages: [...t.messages, { id: nextMsgId(), role: 'assistant' as const, timestamp: Date.now(), content: `（已切换为「${p.label}」人格）\n想和我聊点什么呢？` }]
      } : t))
    }
  }, [activePersonality, personalities, activeId])

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'row', minHeight: 0 }}>
      {/* 左侧话题侧边栏 */}
      <div style={{ width: 200, minWidth: 200, borderRight: `1px solid ${C('color-border')}`, background: C('color-bg-elevated'), display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <div style={{ padding: '10px 12px', borderBottom: `1px solid ${C('color-border')}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Typography.Text strong style={{ fontSize: 12, color: C('color-text') }}>对话</Typography.Text>
          <Button type="text" size="small" icon={<PlusOutlined />} onClick={handleCreateTopic} style={{ color: C('color-text-secondary'), width: 24, height: 24, padding: 0, borderRadius: 6 }} />
        </div>
        <div style={{ flex: 1, overflow: 'auto', padding: '4px 6px' }}>
          {topicList.map(t => (
            <div key={t.id} onClick={() => setActiveId(t.id)} onDoubleClick={() => handleRenameTopic(t.id)}
              style={{ padding: '7px 10px', borderRadius: 8, cursor: 'pointer', marginBottom: 2, background: t.id === activeId ? `${C('color-primary')}12` : 'transparent', display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, color: t.id === activeId ? C('color-primary') : C('color-text') }}>
              <MessageOutlined style={{ fontSize: 11, opacity: 0.5 }} />
              {editingTitle === t.id ? (
                <input value={editValue} onChange={e => setEditValue(e.target.value)} onBlur={() => handleRenameConfirm(t.id)} onKeyDown={e => { if (e.key === 'Enter') handleRenameConfirm(t.id) }} autoFocus
                  style={{ flex: 1, border: 'none', outline: 'none', background: 'transparent', fontSize: 12, color: C('color-text'), padding: 0 }} onClick={e => e.stopPropagation()} />
              ) : (
                <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{t.title}</span>
              )}
              {t.id === activeId && topics.length > 1 && (
                <Button type="text" size="small" icon={<DeleteOutlined />} onClick={e => { e.stopPropagation(); handleDeleteTopic(t.id) }}
                  style={{ color: C('color-text-secondary'), opacity: 0.3, width: 18, height: 18, padding: 0, minWidth: 18, fontSize: 10, borderRadius: 4 }} />
              )}
            </div>
          ))}
        </div>
        <div style={{ padding: '8px 12px', borderTop: `1px solid ${C('color-border')}`, fontSize: 10, color: C('color-text-secondary'), opacity: 0.6 }}>
          <ApiOutlined style={{ marginRight: 4 }} />{engineName || '默认引擎'}{modelName && <span> / {modelName}</span>}
        </div>
      </div>

      {/* 右侧主聊天区 */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, overflow: 'hidden', background: C('color-bg-container'), position: 'relative' }}>
        {/* 人格信息头 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 16px', borderBottom: `1px solid ${C('color-border')}`, flexShrink: 0, background: C('color-bg-elevated') }}>
          <div style={{ width: 32, height: 32, borderRadius: '50%', background: 'linear-gradient(135deg, #e85388, #a855f7)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 16 }}>
            {currentPersonality?.gender === 'male' ? '🤵' : '👩'}
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <Typography.Text strong style={{ fontSize: 13, color: C('color-text') }}>{currentPersonality?.label || '温柔'}</Typography.Text>
              <Button type="text" size="small" icon={<SwapOutlined />} onClick={() => setPersonalityOpen(true)} style={{ color: C('color-text-secondary'), width: 22, height: 22, padding: 0 }} />
            </div>
          </div>
          {hasMessages && (
            <Tooltip title="清空当前对话"><Button type="text" size="small" icon={<ClearOutlined />} onClick={handleClearMessages} style={{ color: C('color-text-secondary'), opacity: 0.4, width: 26, height: 26, padding: 0 }} /></Tooltip>
          )}
        </div>

        {/* 引擎状态条 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '3px 16px', borderBottom: `1px solid ${C('color-border')}`, flexShrink: 0, fontSize: 10, color: C('color-text-secondary'), background: C('color-bg-elevated'), flexWrap: 'wrap' }}>
            <span>💭{emotion}</span><span style={{ opacity: 0.2 }}>|</span>
            <span>🤝{stage === 'STRANGER' ? '初识' : stage === 'FAMILIAR' ? '熟悉' : stage === 'INTIMATE' ? '亲密' : stage}</span><span style={{ opacity: 0.2 }}>|</span>
            <span>💚{trust}</span><span style={{ opacity: 0.2 }}>|</span>
            <span title="亲密度/安全感/唤醒度/支配感">📐{aff}/{sec}/{aro}/{dom}</span>
            {rifts > 0 && <><span style={{ opacity: 0.2 }}>|</span><span>💔{rifts}</span></>}
            <span style={{ marginLeft: 'auto', opacity: 0.4 }}>#{totalTurns}</span>
          </div>

        {/* 消息区 */}
        <div ref={listRef} style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: hasMessages ? '20px 0 150px' : '0' }}>
          {!hasMessages ? (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', padding: 32 }}>
              <div style={{ width: 72, height: 72, borderRadius: '50%', background: 'linear-gradient(135deg, #e85388, #a855f7)', display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: 20, boxShadow: '0 8px 32px rgba(232,83,136,0.3)' }}>
                <HeartOutlined style={{ fontSize: 32, color: '#fff' }} />
              </div>
              <Typography.Text style={{ color: C('color-text'), fontSize: 20, fontWeight: 700, marginBottom: 4 }}>轻语</Typography.Text>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 13, marginBottom: 20 }}>选择一位AI伴侣，开始对话 💫</Typography.Text>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 8, maxWidth: 460, width: '100%', marginBottom: 20 }}>
                {[{ icon: '💬', label: '聊聊今天', desc: '分享你的日常' }, { icon: '💭', label: '倾诉心情', desc: '说说心里话' }, { icon: '🎵', label: '分享兴趣', desc: '聊聊你喜欢的东西' }, { icon: '🌙', label: '晚安问候', desc: '睡前聊一会儿' }].map(s => (
                  <div key={s.label} onClick={() => { setInput(s.label); inputRef.current?.focus() }}
                    style={{ padding: '10px 12px', borderRadius: 12, background: C('color-bg-elevated'), border: `1px solid ${C('color-border')}`, cursor: 'pointer', transition: 'all 0.15s', userSelect: 'none' }}
                    onMouseEnter={e => { e.currentTarget.style.borderColor = '#e85388'; e.currentTarget.style.transform = 'translateY(-2px)' }}
                    onMouseLeave={e => { e.currentTarget.style.borderColor = C('color-border'); e.currentTarget.style.transform = 'translateY(0)' }}>
                    <div style={{ fontSize: 18, marginBottom: 4 }}>{s.icon}</div><div style={{ color: C('color-text'), fontSize: 12, fontWeight: 500 }}>{s.label}</div><div style={{ color: C('color-text-secondary'), fontSize: 10 }}>{s.desc}</div>
                  </div>
                ))}
              </div>
              <Button type="primary" icon={<SwapOutlined />} onClick={() => setPersonalityOpen(true)}
                style={{ borderRadius: 20, padding: '4px 22px', height: 38, fontSize: 13, background: 'linear-gradient(135deg, #e85388, #a855f7)', border: 'none' }}>选择伴侣人格</Button>
            </div>
          ) : (
            <div style={{ maxWidth: 660, margin: '0 auto', padding: '0 16px' }}>
              {messages.map(msg => {
                const isUser = msg.role === 'user'; const isStreaming = msg.streaming && msg === streamingMsg; const displayContent = isStreaming ? streamText : msg.content
                return (
                  <div key={msg.id} style={{ display: 'flex', gap: 10, marginBottom: 20, flexDirection: isUser ? 'row-reverse' : 'row' }}>
                    <Avatar size={28} icon={isUser ? <UserOutlined /> : <HeartOutlined />} style={{ background: isUser ? C('color-primary') : 'linear-gradient(135deg, #e85388, #a855f7)', color: '#fff', flexShrink: 0 }} />
                    <div style={{ flex: isUser ? undefined : 1, maxWidth: isUser ? '70%' : '100%' }}>
                      <div style={{ color: C('color-text'), whiteSpace: 'pre-wrap', lineHeight: 1.7, fontSize: 13, wordBreak: 'break-word', padding: isUser ? '7px 14px' : '0', borderRadius: isUser ? 16 : 0, background: isUser ? `${C('color-primary')}10` : 'transparent' }}>{displayContent}{isStreaming && <span className="cursor-blink" />}</div>
                      {msg.content && !msg.streaming && (
                        <div style={{ display: 'flex', gap: 2, marginTop: 3 }}>
                          <Tooltip title={copiedId === msg.id ? '已复制' : '复制'}><Button type="text" size="small" icon={copiedId === msg.id ? <CheckOutlined style={{ color: '#52c41a' }} /> : <CopyOutlined />} onClick={() => handleCopy(msg.content, msg.id)} style={{ color: C('color-text-secondary'), opacity: 0.4, fontSize: 11, padding: '0 4px', height: 20 }} /></Tooltip>
                          {!isUser && <Tooltip title={speakingId === msg.id ? '朗读中…' : '朗读'}><Button type="text" size="small" icon={<SoundOutlined />} loading={speakingId === msg.id} onClick={() => handleSpeak(msg.content, msg.id)} style={{ color: C('color-text-secondary'), opacity: 0.4, fontSize: 11, padding: '0 4px', height: 20 }} /></Tooltip>}
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
              {loading && !streamText && !streamingMsg && (
                <div style={{ display: 'flex', gap: 10, marginBottom: 20 }}><Avatar size={28} icon={<HeartOutlined />} style={{ background: 'linear-gradient(135deg, #e85388, #a855f7)', color: '#fff' }} /><div style={{ padding: '6px 0' }}><span className="typing-dots"><span className="typing-dot" /><span className="typing-dot" /><span className="typing-dot" /></span></div></div>
              )}
            </div>
          )}
        </div>

        {/* 输入框 */}
        <div style={{ position: 'absolute', bottom: 0, left: 0, right: 0, display: 'flex', justifyContent: 'center', padding: '0 16px 16px', pointerEvents: 'none' }}>
          <div style={{ width: '100%', maxWidth: 660, display: 'flex', alignItems: 'flex-end', gap: 6, padding: '6px 10px', background: C('color-bg-container'), border: `1px solid ${C('color-border')}`, borderRadius: 16, boxShadow: '0 6px 24px rgba(0,0,0,0.06)', pointerEvents: 'auto' }}>
            <Input.TextArea ref={inputRef} value={input} onChange={e => setInput(e.target.value)} onKeyDown={handleKeyDown} placeholder="输入消息，Enter 发送 / Shift+Enter 换行" disabled={loading} autoSize={{ minRows: 1, maxRows: 5 }} className="chat-input-textarea" style={{ flex: 1, background: 'transparent', border: 'none', color: C('color-text'), borderRadius: 0, resize: 'none', fontSize: 13, lineHeight: 1.5, padding: '4px 2px', boxShadow: 'none' }} />
            <Tooltip title="发送 (Enter)"><Button type="primary" icon={<SendOutlined />} onClick={handleSend} loading={loading} disabled={!input.trim()} style={{ background: input.trim() ? 'linear-gradient(135deg, #e85388, #a855f7)' : C('color-border'), borderColor: 'transparent', borderRadius: 10, width: 34, height: 34, minWidth: 34, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: input.trim() ? '0 2px 8px rgba(232,83,136,0.4)' : 'none', flexShrink: 0, transition: 'transform 0.15s', transform: 'scale(1)' }} onMouseEnter={e=>e.currentTarget.style.transform="scale(1.05)"} onMouseLeave={e=>e.currentTarget.style.transform="scale(1)"} /></Tooltip>
          </div>
        </div>
      </div>

      {/* 右侧情感面板 — 始终可见 */}
      <aside style={{
          width: 260, minWidth: 260, borderLeft: `1px solid ${C('color-border')}`,
          background: C('color-bg-elevated'), overflow: 'auto', flexShrink: 0,
        }}>
          <WhisperEmotionPanel
            emotion={emotion} stage={stage} trust={trust} rifts={rifts}
            aff={aff} sec={sec} aro={aro} dom={dom}
            T={currentPersonality?.dims?.T ?? 50} I={currentPersonality?.dims?.I ?? 50}
            S={currentPersonality?.dims?.S ?? 50} O={currentPersonality?.dims?.O ?? 50}
            R={currentPersonality?.dims?.R ?? 50}
            totalTurns={totalTurns}
            personalityLabel={currentPersonality?.label || '温柔'}
          />
        </aside>

      {/* 人格选择弹窗 */}
      <Modal title="选择伴侣人格" open={personalityOpen} onCancel={() => setPersonalityOpen(false)} footer={null} width={680} centered>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: 10, maxHeight: 400, overflow: 'auto' }}>
          {personalities.map(p => (
            <Card key={p.id} hoverable size="small" onClick={() => handleSwitchPersonality(p.id)}
              style={{ border: activePersonality === p.id ? '2px solid #e85388' : `1px solid ${C('color-border')}`, background: activePersonality === p.id ? '#e8538808' : C('color-bg-elevated'), borderRadius: 12, cursor: 'pointer' }}
              bodyStyle={{ padding: '10px 12px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
                <span>{p.gender === 'male' ? '🤵' : '👩'}</span>
                <Typography.Text strong style={{ fontSize: 13 }}>{p.label}</Typography.Text>
                {activePersonality === p.id && <Tag color="magenta" style={{ marginLeft: 'auto', fontSize: 9, lineHeight: '14px' }}>当前</Tag>}
              </div>
              <Typography.Paragraph type="secondary" style={{ fontSize: 10, marginBottom: 4 }} ellipsis={{ rows: 1 }}>{p.voiceGuide || `${p.label}型伴侣`}</Typography.Paragraph>
            </Card>
          ))}
        </div>
      </Modal>
    </div>
  )
}

export default WhisperPage
